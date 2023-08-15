package bitfinex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var comms = make(chan stream.Response)

type checksum struct {
	Token    int
	Sequence int64
}

// checksumStore quick global for now
var checksumStore = make(map[int]*checksum)
var cMtx sync.Mutex

// WsConnect starts a new websocket connection
func (b *Bitfinex) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	err := b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v unable to connect to Websocket. Error: %s",
			b.Name,
			err)
	}

	b.Websocket.Wg.Add(1)
	go b.wsReadData(b.Websocket.Conn)

	if b.Websocket.CanUseAuthenticatedEndpoints() {
		err = b.Websocket.AuthConn.Dial(&dialer, http.Header{})
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v unable to connect to authenticated Websocket. Error: %s",
				b.Name,
				err)
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
		b.Websocket.Wg.Add(1)
		go b.wsReadData(b.Websocket.AuthConn)
		err = b.WsSendAuth(context.TODO())
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v - authentication failed: %v\n",
				b.Name,
				err)
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	b.Websocket.Wg.Add(1)
	go b.WsDataHandler()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (b *Bitfinex) wsReadData(ws stream.Connection) {
	defer b.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		comms <- resp
	}
}

// WsDataHandler handles data from wsReadData
func (b *Bitfinex) WsDataHandler() {
	defer b.Websocket.Wg.Done()
	for {
		select {
		case <-b.Websocket.ShutdownC:
			select {
			case resp := <-comms:
				err := b.wsHandleData(resp.Raw)
				if err != nil {
					select {
					case b.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr,
							"%s websocket handle data error: %v",
							b.Name,
							err)
					}
				}
			default:
			}
			return
		case resp := <-comms:
			if resp.Type != websocket.TextMessage {
				continue
			}
			err := b.wsHandleData(resp.Raw)
			if err != nil {
				b.Websocket.DataHandler <- err
			}
		}
	}
}

func (b *Bitfinex) wsHandleData(respRaw []byte) error {
	var result interface{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	switch d := result.(type) {
	case map[string]interface{}:
		event := d["event"]
		switch event {
		case "subscribed":
			chanID, ok := d["chanId"].(float64)
			if !ok {
				return errors.New("unable to type assert chanId")
			}
			channel, ok := d["channel"].(string)
			if !ok {
				return errors.New("unable to type assert channel")
			}
			if symbol, ok := d["symbol"].(string); ok {
				b.WsAddSubscriptionChannel(int(chanID),
					channel,
					symbol,
				)
			} else if key, ok := d["key"].(string); ok {
				// Capture trading subscriptions
				if contents := strings.Split(key, ":"); len(contents) > 3 {
					// Edge case to parse margin strings.
					// map[chanId:139136 channel:candles event:subscribed key:trade:1m:tXAUTF0:USTF0]
					if contents[2][0] == 't' {
						key = contents[2] + ":" + contents[3]
					}
				}
				b.WsAddSubscriptionChannel(int(chanID),
					channel,
					key,
				)
			}
		case "auth":
			status, ok := d["status"].(string)
			if !ok {
				return errors.New("unable to type assert status")
			}
			if status == "OK" {
				b.Websocket.DataHandler <- d
				b.WsAddSubscriptionChannel(0, "account", "")
			} else if status == "fail" {
				if code, ok := d["code"].(string); ok {
					return fmt.Errorf("websocket unable to AUTH. Error code: %s",
						code)
				}
				return errors.New("websocket unable to auth")
			}
		}
	case []interface{}:
		chanF, ok := d[0].(float64)
		if !ok {
			return errors.New("channel ID type assertion failure")
		}

		chanID := int(chanF)
		var datum string
		if datum, ok = d[1].(string); ok {
			// Capturing heart beat
			if datum == "hb" {
				return nil
			}

			// Capturing checksum and storing value
			if datum == "cs" {
				var tokenF float64
				tokenF, ok = d[2].(float64)
				if !ok {
					return errors.New("checksum token type assertion failure")
				}
				var seqNoF float64
				seqNoF, ok = d[3].(float64)
				if !ok {
					return errors.New("sequence number type assertion failure")
				}

				cMtx.Lock()
				checksumStore[chanID] = &checksum{
					Token:    int(tokenF),
					Sequence: int64(seqNoF),
				}
				cMtx.Unlock()
				return nil
			}
		}

		chanInfo, ok := b.WebsocketSubdChannels[chanID]
		if !ok && chanID != 0 {
			return fmt.Errorf("unable to locate chanID: %d",
				chanID)
		}

		var chanAsset = asset.Spot
		var pair currency.Pair
		pairInfo := strings.Split(chanInfo.Pair, ":")
		switch {
		case len(pairInfo) >= 3:
			newPair := pairInfo[2]
			if newPair[0] == 'f' {
				chanAsset = asset.MarginFunding
			}

			pair, err = currency.NewPairFromString(newPair[1:])
			if err != nil {
				return err
			}
		case len(pairInfo) == 1 && chanInfo.Pair != "":
			newPair := pairInfo[0]
			if newPair[0] == 'f' {
				chanAsset = asset.MarginFunding
			}

			pair, err = currency.NewPairFromString(newPair[1:])
			if err != nil {
				return err
			}
		case chanInfo.Pair != "":
			if strings.Contains(chanInfo.Pair, ":") {
				chanAsset = asset.Margin
			}

			pair, err = currency.NewPairFromString(chanInfo.Pair[1:])
			if err != nil {
				return err
			}
		}

		switch chanInfo.Channel {
		case wsBook:
			var newOrderbook []WebsocketBook
			obSnapBundle, ok := d[1].([]interface{})
			if !ok {
				return errors.New("orderbook interface cast failed")
			}
			if len(obSnapBundle) == 0 {
				return errors.New("no data within orderbook snapshot")
			}

			sequenceNo, ok := d[2].(float64)
			if !ok {
				return errors.New("type assertion failure")
			}

			var fundingRate bool
			switch id := obSnapBundle[0].(type) {
			case []interface{}:
				for i := range obSnapBundle {
					data, ok := obSnapBundle[i].([]interface{})
					if !ok {
						return errors.New("type assertion failed for orderbok item data")
					}
					id, okAssert := data[0].(float64)
					if !okAssert {
						return errors.New("type assertion failed for orderbook id data")
					}
					pricePeriod, okAssert := data[1].(float64)
					if !okAssert {
						return errors.New("type assertion failed for orderbook price data")
					}
					rateAmount, okAssert := data[2].(float64)
					if !okAssert {
						return errors.New("type assertion failed for orderbook rate data")
					}
					if len(data) == 4 {
						fundingRate = true
						amount, okFunding := data[3].(float64)
						if !okFunding {
							return errors.New("type assertion failed for orderbook funding data")
						}
						newOrderbook = append(newOrderbook, WebsocketBook{
							ID:     int64(id),
							Period: int64(pricePeriod),
							Price:  rateAmount,
							Amount: amount})
					} else {
						newOrderbook = append(newOrderbook, WebsocketBook{
							ID:     int64(id),
							Price:  pricePeriod,
							Amount: rateAmount})
					}
				}
				if err = b.WsInsertSnapshot(pair, chanAsset, newOrderbook, fundingRate); err != nil {
					return fmt.Errorf("inserting snapshot error: %s",
						err)
				}
			case float64:
				pricePeriod, okSnap := obSnapBundle[1].(float64)
				if !okSnap {
					return errors.New("type assertion failed for orderbook price snapshot data")
				}
				amountRate, okSnap := obSnapBundle[2].(float64)
				if !okSnap {
					return errors.New("type assertion failed for orderbook amount snapshot data")
				}
				if len(obSnapBundle) == 4 {
					fundingRate = true
					var amount float64
					amount, okSnap = obSnapBundle[3].(float64)
					if !okSnap {
						return errors.New("type assertion failed for orderbook amount snapshot data")
					}
					newOrderbook = append(newOrderbook, WebsocketBook{
						ID:     int64(id),
						Period: int64(pricePeriod),
						Price:  amountRate,
						Amount: amount})
				} else {
					newOrderbook = append(newOrderbook, WebsocketBook{
						ID:     int64(id),
						Price:  pricePeriod,
						Amount: amountRate})
				}

				if err = b.WsUpdateOrderbook(pair, chanAsset, newOrderbook, chanID, int64(sequenceNo), fundingRate); err != nil {
					return fmt.Errorf("updating orderbook error: %s",
						err)
				}
			}

			return nil
		case wsCandles:
			if candleBundle, ok := d[1].([]interface{}); ok {
				if len(candleBundle) == 0 {
					return nil
				}

				switch candleData := candleBundle[0].(type) {
				case []interface{}:
					for i := range candleBundle {
						var element []interface{}
						element, ok = candleBundle[i].([]interface{})
						if !ok {
							return errors.New("candle type assertion for element data")
						}
						if len(element) < 6 {
							return errors.New("invalid candleBundle length")
						}
						var klineData stream.KlineData
						if klineData.Timestamp, err = convert.TimeFromUnixTimestampFloat(element[0]); err != nil {
							return fmt.Errorf("unable to convert candle timestamp: %w", err)
						}
						if klineData.OpenPrice, ok = element[1].(float64); !ok {
							return errors.New("unable to type assert candle open price")
						}
						if klineData.ClosePrice, ok = element[2].(float64); !ok {
							return errors.New("unable to type assert candle close price")
						}
						if klineData.HighPrice, ok = element[3].(float64); !ok {
							return errors.New("unable to type assert candle high price")
						}
						if klineData.LowPrice, ok = element[4].(float64); !ok {
							return errors.New("unable to type assert candle low price")
						}
						if klineData.Volume, ok = element[5].(float64); !ok {
							return errors.New("unable to type assert candle volume")
						}
						klineData.Exchange = b.Name
						klineData.AssetType = chanAsset
						klineData.Pair = pair
						b.Websocket.DataHandler <- klineData
					}
				case float64:
					if len(candleBundle) < 6 {
						return errors.New("invalid candleBundle length")
					}
					var klineData stream.KlineData
					if klineData.Timestamp, err = convert.TimeFromUnixTimestampFloat(candleData); err != nil {
						return fmt.Errorf("unable to convert candle timestamp: %w", err)
					}
					if klineData.OpenPrice, ok = candleBundle[1].(float64); !ok {
						return errors.New("unable to type assert candle open price")
					}
					if klineData.ClosePrice, ok = candleBundle[2].(float64); !ok {
						return errors.New("unable to type assert candle close price")
					}
					if klineData.HighPrice, ok = candleBundle[3].(float64); !ok {
						return errors.New("unable to type assert candle high price")
					}
					if klineData.LowPrice, ok = candleBundle[4].(float64); !ok {
						return errors.New("unable to type assert candle low price")
					}
					if klineData.Volume, ok = candleBundle[5].(float64); !ok {
						return errors.New("unable to type assert candle volume")
					}
					klineData.Exchange = b.Name
					klineData.AssetType = chanAsset
					klineData.Pair = pair
					b.Websocket.DataHandler <- klineData
				}
			}
			return nil
		case wsTicker:
			tickerData, ok := d[1].([]interface{})
			if !ok {
				return errors.New("type assertion for tickerData")
			}

			t := &ticker.Price{
				AssetType:    chanAsset,
				Pair:         pair,
				ExchangeName: b.Name,
			}

			if len(tickerData) == 10 {
				if t.Bid, ok = tickerData[0].(float64); !ok {
					return errors.New("unable to type assert ticker bid")
				}
				if t.Ask, ok = tickerData[2].(float64); !ok {
					return errors.New("unable to type assert ticker ask")
				}
				if t.Last, ok = tickerData[6].(float64); !ok {
					return errors.New("unable to type assert ticker last")
				}
				if t.Volume, ok = tickerData[7].(float64); !ok {
					return errors.New("unable to type assert ticker volume")
				}
				if t.High, ok = tickerData[8].(float64); !ok {
					return errors.New("unable to type assert  ticker high")
				}
				if t.Low, ok = tickerData[9].(float64); !ok {
					return errors.New("unable to type assert ticker low")
				}
			} else {
				if t.FlashReturnRate, ok = tickerData[0].(float64); !ok {
					return errors.New("unable to type assert ticker flash return rate")
				}
				if t.Bid, ok = tickerData[1].(float64); !ok {
					return errors.New("unable to type assert ticker bid")
				}
				if t.BidPeriod, ok = tickerData[2].(float64); !ok {
					return errors.New("unable to type assert ticker bid period")
				}
				if t.BidSize, ok = tickerData[3].(float64); !ok {
					return errors.New("unable to type assert ticker bid size")
				}
				if t.Ask, ok = tickerData[4].(float64); !ok {
					return errors.New("unable to type assert ticker ask")
				}
				if t.AskPeriod, ok = tickerData[5].(float64); !ok {
					return errors.New("unable to type assert ticker ask period")
				}
				if t.AskSize, ok = tickerData[6].(float64); !ok {
					return errors.New("unable to type assert ticker ask size")
				}
				if t.Last, ok = tickerData[9].(float64); !ok {
					return errors.New("unable to type assert ticker last")
				}
				if t.Volume, ok = tickerData[10].(float64); !ok {
					return errors.New("unable to type assert ticker volume")
				}
				if t.High, ok = tickerData[11].(float64); !ok {
					return errors.New("unable to type assert ticker high")
				}
				if t.Low, ok = tickerData[12].(float64); !ok {
					return errors.New("unable to type assert ticker low")
				}
				if t.FlashReturnRateAmount, ok = tickerData[15].(float64); !ok {
					return errors.New("unable to type assert ticker flash return rate")
				}
			}
			b.Websocket.DataHandler <- t
			return nil
		case wsTrades:
			if !b.IsSaveTradeDataEnabled() {
				return nil
			}
			if chanAsset == asset.MarginFunding {
				return nil
			}
			var tradeHolder []WebsocketTrade
			switch len(d) {
			case 2:
				snapshot, ok := d[1].([]interface{})
				if !ok {
					return errors.New("unable to type assert trade snapshot data")
				}
				for i := range snapshot {
					elem, ok := snapshot[i].([]interface{})
					if !ok {
						return errors.New("unable to type assert trade snapshot element data")
					}
					tradeID, ok := elem[0].(float64)
					if !ok {
						return errors.New("unable to type assert trade ID")
					}
					timestamp, ok := elem[1].(float64)
					if !ok {
						return errors.New("unable to type assert trade timestamp")
					}
					amount, ok := elem[2].(float64)
					if !ok {
						return errors.New("unable to type assert trade amount")
					}
					wsTrade := WebsocketTrade{
						ID:        int64(tradeID),
						Timestamp: int64(timestamp),
						Amount:    amount,
					}
					if len(elem) == 5 {
						rate, ok := elem[3].(float64)
						if !ok {
							return errors.New("unable to type assert trade rate")
						}
						wsTrade.Rate = rate
						period, ok := elem[4].(float64)
						if !ok {
							return errors.New("unable to type assert trade period")
						}
						wsTrade.Period = int64(period)
					} else {
						price, ok := elem[3].(float64)
						if !ok {
							return errors.New("unable to type assert trade price")
						}
						wsTrade.Rate = price
					}
					tradeHolder = append(tradeHolder, wsTrade)
				}
			case 3:
				event, ok := d[1].(string)
				if !ok {
					return errors.New("unable to type assert data event")
				}
				if event != wsFundingTradeUpdate &&
					event != wsTradeExecutionUpdate {
					return nil
				}
				data, ok := d[2].([]interface{})
				if !ok {
					return errors.New("trade data type assertion error")
				}

				tradeID, ok := data[0].(float64)
				if !ok {
					return errors.New("unable to type assert trade ID")
				}
				timestamp, ok := data[1].(float64)
				if !ok {
					return errors.New("unable to type assert trade timestamp")
				}
				amount, ok := data[2].(float64)
				if !ok {
					return errors.New("unable to type assert trade amount")
				}
				wsTrade := WebsocketTrade{
					ID:        int64(tradeID),
					Timestamp: int64(timestamp),
					Amount:    amount,
				}
				if len(data) == 5 {
					rate, ok := data[3].(float64)
					if !ok {
						return errors.New("unable to type assert trade rate")
					}
					period, ok := data[4].(float64)
					if !ok {
						return errors.New("unable to type assert trade period")
					}
					wsTrade.Rate = rate
					wsTrade.Period = int64(period)
				} else {
					price, ok := data[3].(float64)
					if !ok {
						return errors.New("unable to type assert trade price")
					}
					wsTrade.Price = price
				}
				tradeHolder = append(tradeHolder, wsTrade)
			}
			trades := make([]trade.Data, len(tradeHolder))
			for i := range tradeHolder {
				side := order.Buy
				newAmount := tradeHolder[i].Amount
				if newAmount < 0 {
					side = order.Sell
					newAmount *= -1
				}
				price := tradeHolder[i].Price
				if price == 0 && tradeHolder[i].Rate > 0 {
					price = tradeHolder[i].Rate
				}
				trades[i] = trade.Data{
					TID:          strconv.FormatInt(tradeHolder[i].ID, 10),
					CurrencyPair: pair,
					Timestamp:    time.UnixMilli(tradeHolder[i].Timestamp),
					Price:        price,
					Amount:       newAmount,
					Exchange:     b.Name,
					AssetType:    chanAsset,
					Side:         side,
				}
			}

			return b.AddTradesToBuffer(trades...)
		}

		if authResp, ok := d[1].(string); ok {
			switch authResp {
			case wsHeartbeat, pong:
				return nil
			case wsNotification:
				notification, ok := d[2].([]interface{})
				if !ok {
					return errors.New("unable to type assert notification data")
				}
				if data, ok := notification[4].([]interface{}); ok {
					channelName, ok := notification[1].(string)
					if !ok {
						return errors.New("unable to type assert channelName")
					}
					switch {
					case strings.Contains(channelName, wsFundingOfferNewRequest),
						strings.Contains(channelName, wsFundingOfferUpdateRequest),
						strings.Contains(channelName, wsFundingOfferCancelRequest):
						if data[0] != nil {
							if id, ok := data[0].(float64); ok && id > 0 {
								if b.Websocket.Match.IncomingWithData(int64(id), respRaw) {
									return nil
								}
								offer, err := wsHandleFundingOffer(data, true /* include rate real */)
								if err != nil {
									return err
								}
								b.Websocket.DataHandler <- offer
							}
						}
					case strings.Contains(channelName, wsOrderNewRequest),
						strings.Contains(channelName, wsOrderUpdateRequest),
						strings.Contains(channelName, wsOrderCancelRequest):
						if data[2] != nil {
							if id, ok := data[2].(float64); ok && id > 0 {
								if b.Websocket.Match.IncomingWithData(int64(id), respRaw) {
									return nil
								}
								b.wsHandleOrder(data)
							}
						}
					default:
						return fmt.Errorf("%s - Unexpected data returned %s",
							b.Name,
							respRaw)
					}
				}
				if notification[5] != nil {
					if wsErr, ok := notification[5].(string); ok {
						if strings.EqualFold(wsErr, wsError) {
							if errMsg, ok := notification[6].(string); ok {
								return fmt.Errorf("%s - Error %s",
									b.Name,
									errMsg)
							}
							return fmt.Errorf("%s - unhandled error message: %v", b.Name,
								notification[6])
						}
					}
				}
			case wsOrderSnapshot:
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						for i := range snapBundle {
							if positionData, ok := snapBundle[i].([]interface{}); ok {
								b.wsHandleOrder(positionData)
							}
						}
					}
				}
			case wsOrderCancel, wsOrderNew, wsOrderUpdate:
				if oData, ok := d[2].([]interface{}); ok && len(oData) > 0 {
					b.wsHandleOrder(oData)
				}
			case wsPositionSnapshot:
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						snapshot := make([]WebsocketPosition, len(snapBundle))
						for i := range snapBundle {
							positionData, ok := snapBundle[i].([]interface{})
							if !ok {
								return errors.New("unable to type assert wsPositionSnapshot positionData")
							}
							var position WebsocketPosition
							if position.Pair, ok = positionData[0].(string); !ok {
								return errors.New("unable to type assert position snapshot pair")
							}
							if position.Status, ok = positionData[1].(string); !ok {
								return errors.New("unable to type assert position snapshot status")
							}
							if position.Amount, ok = positionData[2].(float64); !ok {
								return errors.New("unable to type assert position snapshot amount")
							}
							if position.Price, ok = positionData[3].(float64); !ok {
								return errors.New("unable to type assert position snapshot price")
							}
							if position.MarginFunding, ok = positionData[4].(float64); !ok {
								return errors.New("unable to type assert position snapshot margin funding")
							}
							marginFundingType, ok := positionData[5].(float64)
							if !ok {
								return errors.New("unable to type assert position snapshot margin funding type")
							}
							position.MarginFundingType = int64(marginFundingType)
							if position.ProfitLoss, ok = positionData[6].(float64); !ok {
								return errors.New("unable to type assert position snapshot profit loss")
							}
							if position.ProfitLossPercent, ok = positionData[7].(float64); !ok {
								return errors.New("unable to type assert position snapshot profit loss percent")
							}
							if position.LiquidationPrice, ok = positionData[8].(float64); !ok {
								return errors.New("unable to type assert position snapshot liquidation price")
							}
							if position.Leverage, ok = positionData[9].(float64); !ok {
								return errors.New("unable to type assert position snapshot leverage")
							}
							snapshot[i] = position
						}
						b.Websocket.DataHandler <- snapshot
					}
				}
			case wsPositionNew, wsPositionUpdate, wsPositionClose:
				if positionData, ok := d[2].([]interface{}); ok && len(positionData) > 0 {
					var position WebsocketPosition
					if position.Pair, ok = positionData[0].(string); !ok {
						return errors.New("unable to type assert position pair")
					}
					if position.Status, ok = positionData[1].(string); !ok {
						return errors.New("unable to type assert position status")
					}
					if position.Amount, ok = positionData[2].(float64); !ok {
						return errors.New("unable to type assert position amount")
					}
					if position.Price, ok = positionData[3].(float64); !ok {
						return errors.New("unable to type assert position price")
					}
					if position.MarginFunding, ok = positionData[4].(float64); !ok {
						return errors.New("unable to type assert margin position funding")
					}
					marginFundingType, ok := positionData[5].(float64)
					if !ok {
						return errors.New("unable to type assert position margin funding type")
					}
					position.MarginFundingType = int64(marginFundingType)
					if position.ProfitLoss, ok = positionData[6].(float64); !ok {
						return errors.New("unable to type assert position profit loss")
					}
					if position.ProfitLossPercent, ok = positionData[7].(float64); !ok {
						return errors.New("unable to type assert position profit loss percent")
					}
					if position.LiquidationPrice, ok = positionData[8].(float64); !ok {
						return errors.New("unable to type assert position liquidation price")
					}
					if position.Leverage, ok = positionData[9].(float64); !ok {
						return errors.New("unable to type assert position leverage")
					}
					b.Websocket.DataHandler <- position
				}
			case wsTradeExecuted, wsTradeExecutionUpdate:
				if tradeData, ok := d[2].([]interface{}); ok && len(tradeData) > 4 {
					var tData WebsocketTradeData
					var tradeID float64
					if tradeID, ok = tradeData[0].(float64); !ok {
						return errors.New("unable to type assert trade ID")
					}
					tData.TradeID = int64(tradeID)
					if tData.Pair, ok = tradeData[1].(string); !ok {
						return errors.New("unable to type assert trade pair")
					}
					var timestamp float64
					if timestamp, ok = tradeData[2].(float64); !ok {
						return errors.New("unable to type assert trade timestamp")
					}
					tData.Timestamp = int64(timestamp)
					var orderID float64
					if orderID, ok = tradeData[3].(float64); !ok {
						return errors.New("unable to type assert trade order ID")
					}
					tData.OrderID = int64(orderID)
					if tData.AmountExecuted, ok = tradeData[4].(float64); !ok {
						return errors.New("unable to type assert trade amount executed")
					}
					if tData.PriceExecuted, ok = tradeData[5].(float64); !ok {
						return errors.New("unable to type assert trade price executed")
					}
					if tData.OrderType, ok = tradeData[6].(string); !ok {
						return errors.New("unable to type assert trade order type")
					}
					if tData.OrderPrice, ok = tradeData[7].(float64); !ok {
						return errors.New("unable to type assert trade order type")
					}
					var maker float64
					if maker, ok = tradeData[8].(float64); !ok {
						return errors.New("unable to type assert trade maker")
					}
					tData.Maker = maker == 1
					if tData.Fee, ok = tradeData[9].(float64); !ok {
						return errors.New("unable to type assert trade fee")
					}
					if tData.FeeCurrency, ok = tradeData[10].(string); !ok {
						return errors.New("unable to type assert trade fee currency")
					}
					b.Websocket.DataHandler <- tData
				}
			case wsFundingOfferSnapshot:
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						snapshot := make([]*WsFundingOffer, len(snapBundle))
						for i := range snapBundle {
							data, ok := snapBundle[i].([]interface{})
							if !ok {
								return errors.New("unable to type assert wsFundingOrderSnapshot snapBundle data")
							}
							offer, err := wsHandleFundingOffer(data, false /* include rate real */)
							if err != nil {
								return err
							}
							snapshot[i] = offer
						}
						b.Websocket.DataHandler <- snapshot
					}
				}
			case wsFundingOfferNew, wsFundingOfferUpdate, wsFundingOfferCancel:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					offer, err := wsHandleFundingOffer(data, true /* include rate real */)
					if err != nil {
						return err
					}
					b.Websocket.DataHandler <- offer
				}
			case wsFundingCreditSnapshot:
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						snapshot := make([]*WsCredit, len(snapBundle))
						for i := range snapBundle {
							data, ok := snapBundle[i].([]interface{})
							if !ok {
								return errors.New("unable to type assert wsFundingCreditSnapshot snapBundle data")
							}
							fundingCredit, err := wsHandleFundingCreditLoanData(data, true /* include position pair */)
							if err != nil {
								return err
							}
							snapshot[i] = fundingCredit
						}
						b.Websocket.DataHandler <- snapshot
					}
				}
			case wsFundingCreditNew, wsFundingCreditUpdate, wsFundingCreditCancel:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					fundingCredit, err := wsHandleFundingCreditLoanData(data, true /* include position pair */)
					if err != nil {
						return err
					}
					b.Websocket.DataHandler <- fundingCredit
				}
			case wsFundingLoanSnapshot:
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						snapshot := make([]*WsCredit, len(snapBundle))
						for i := range snapBundle {
							data, ok := snapBundle[i].([]interface{})
							if !ok {
								return errors.New("unable to type assert wsFundingLoanSnapshot snapBundle data")
							}
							fundingLoanSnapshot, err := wsHandleFundingCreditLoanData(data, false /* include position pair */)
							if err != nil {
								return err
							}
							snapshot[i] = fundingLoanSnapshot
						}
						b.Websocket.DataHandler <- snapshot
					}
				}
			case wsFundingLoanNew, wsFundingLoanUpdate, wsFundingLoanCancel:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					fundingData, err := wsHandleFundingCreditLoanData(data, false /* include position pair */)
					if err != nil {
						return err
					}
					b.Websocket.DataHandler <- fundingData
				}
			case wsWalletSnapshot:
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						snapshot := make([]WsWallet, len(snapBundle))
						for i := range snapBundle {
							data, ok := snapBundle[i].([]interface{})
							if !ok {
								return errors.New("unable to type assert wsWalletSnapshot snapBundle data")
							}
							var wallet WsWallet
							if wallet.Type, ok = data[0].(string); !ok {
								return errors.New("unable to type assert wallet snapshot type")
							}
							if wallet.Currency, ok = data[1].(string); !ok {
								return errors.New("unable to type assert wallet snapshot currency")
							}
							if wallet.Balance, ok = data[2].(float64); !ok {
								return errors.New("unable to type assert wallet snapshot balance")
							}
							if wallet.UnsettledInterest, ok = data[3].(float64); !ok {
								return errors.New("unable to type assert wallet snapshot unsettled interest")
							}
							if data[4] != nil {
								if wallet.BalanceAvailable, ok = data[4].(float64); !ok {
									return errors.New("unable to type assert wallet snapshot balance available")
								}
							}
							snapshot[i] = wallet
						}
						b.Websocket.DataHandler <- snapshot
					}
				}
			case wsWalletUpdate:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					var wallet WsWallet
					if wallet.Type, ok = data[0].(string); !ok {
						return errors.New("unable to type assert wallet snapshot type")
					}
					if wallet.Currency, ok = data[1].(string); !ok {
						return errors.New("unable to type assert wallet snapshot currency")
					}
					if wallet.Balance, ok = data[2].(float64); !ok {
						return errors.New("unable to type assert wallet snapshot balance")
					}
					if wallet.UnsettledInterest, ok = data[3].(float64); !ok {
						return errors.New("unable to type assert wallet snapshot unsettled interest")
					}
					if data[4] != nil {
						if wallet.BalanceAvailable, ok = data[4].(float64); !ok {
							return errors.New("unable to type assert wallet snapshot balance available")
						}
					}
					b.Websocket.DataHandler <- wallet
				}
			case wsBalanceUpdate:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					var balance WsBalanceInfo
					if balance.TotalAssetsUnderManagement, ok = data[0].(float64); !ok {
						return errors.New("unable to type assert balance total assets under management")
					}
					if balance.NetAssetsUnderManagement, ok = data[1].(float64); !ok {
						return errors.New("unable to type assert balance net assets under management")
					}
					b.Websocket.DataHandler <- balance
				}
			case wsMarginInfoUpdate:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					if eventType, ok := data[0].(string); ok && eventType == "base" {
						baseData, ok := data[1].([]interface{})
						if !ok {
							return errors.New("unable to type assert wsMarginInfoUpdate baseData")
						}
						var marginInfoBase WsMarginInfoBase
						if marginInfoBase.UserProfitLoss, ok = baseData[0].(float64); !ok {
							return errors.New("unable to type assert margin info user profit loss")
						}
						if marginInfoBase.UserSwaps, ok = baseData[1].(float64); !ok {
							return errors.New("unable to type assert margin info user swaps")
						}
						if marginInfoBase.MarginBalance, ok = baseData[2].(float64); !ok {
							return errors.New("unable to type assert margin info balance")
						}
						if marginInfoBase.MarginNet, ok = baseData[3].(float64); !ok {
							return errors.New("unable to type assert margin info net")
						}
						if marginInfoBase.MarginRequired, ok = baseData[4].(float64); !ok {
							return errors.New("unable to type assert margin info required")
						}
						b.Websocket.DataHandler <- marginInfoBase
					}
				}
			case wsFundingInfoUpdate:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					if fundingType, ok := data[0].(string); ok && fundingType == "sym" {
						symbolData, ok := data[2].([]interface{})
						if !ok {
							return errors.New("unable to type assert wsFundingInfoUpdate symbolData")
						}
						var fundingInfo WsFundingInfo
						if fundingInfo.Symbol, ok = data[1].(string); !ok {
							return errors.New("unable to type assert symbol")
						}
						if fundingInfo.YieldLoan, ok = symbolData[0].(float64); !ok {
							return errors.New("unable to type assert funding info update yield loan")
						}
						if fundingInfo.YieldLend, ok = symbolData[1].(float64); !ok {
							return errors.New("unable to type assert funding info update yield lend")
						}
						if fundingInfo.DurationLoan, ok = symbolData[2].(float64); !ok {
							return errors.New("unable to type assert funding info update duration loan")
						}
						if fundingInfo.DurationLend, ok = symbolData[3].(float64); !ok {
							return errors.New("unable to type assert funding info update duration lend")
						}
						b.Websocket.DataHandler <- fundingInfo
					}
				}
			case wsFundingTradeExecuted, wsFundingTradeUpdate:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					var wsFundingTrade WsFundingTrade
					tradeID, ok := data[0].(float64)
					if !ok {
						return errors.New("unable to type assert funding trade ID")
					}
					wsFundingTrade.ID = int64(tradeID)
					if wsFundingTrade.Symbol, ok = data[1].(string); !ok {
						return errors.New("unable to type assert funding trade symbol")
					}
					created, ok := data[2].(float64)
					if !ok {
						return errors.New("unable to type assert funding trade created")
					}
					wsFundingTrade.MTSCreated = time.UnixMilli(int64(created))
					offerID, ok := data[3].(float64)
					if !ok {
						return errors.New("unable to type assert funding trade offer ID")
					}
					wsFundingTrade.OfferID = int64(offerID)
					if wsFundingTrade.Amount, ok = data[4].(float64); !ok {
						return errors.New("unable to type assert funding trade amount")
					}
					if wsFundingTrade.Rate, ok = data[5].(float64); !ok {
						return errors.New("unable to type assert funding trade rate")
					}
					period, ok := data[6].(float64)
					if !ok {
						return errors.New("unable to type assert funding trade period")
					}
					wsFundingTrade.Period = int64(period)
					wsFundingTrade.Maker = data[7] != nil
					b.Websocket.DataHandler <- wsFundingTrade
				}
			default:
				b.Websocket.DataHandler <- stream.UnhandledMessageWarning{
					Message: b.Name + stream.UnhandledMessage + string(respRaw),
				}
				return nil
			}
		}
	}
	return nil
}

func wsHandleFundingOffer(data []interface{}, includeRateReal bool) (*WsFundingOffer, error) {
	var offer WsFundingOffer
	var ok bool
	if data[0] != nil {
		var offerID float64
		if offerID, ok = data[0].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer ID")
		}
		offer.ID = int64(offerID)
	}
	if data[1] != nil {
		if offer.Symbol, ok = data[1].(string); !ok {
			return nil, errors.New("unable to type assert funding offer symbol")
		}
	}
	if data[2] != nil {
		var created float64
		if created, ok = data[2].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer created")
		}
		offer.Created = time.UnixMilli(int64(created))
	}
	if data[3] != nil {
		var updated float64
		if updated, ok = data[3].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer updated")
		}
		offer.Updated = time.UnixMilli(int64(updated))
	}
	if data[4] != nil {
		if offer.Amount, ok = data[4].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer amount")
		}
	}
	if data[5] != nil {
		if offer.OriginalAmount, ok = data[5].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer original amount")
		}
	}
	if data[6] != nil {
		if offer.Type, ok = data[6].(string); !ok {
			return nil, errors.New("unable to type assert funding offer type")
		}
	}
	if data[9] != nil {
		if offer.Flags, ok = data[9].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer flags")
		}
	}
	if data[10] != nil {
		if offer.Status, ok = data[10].(string); !ok {
			return nil, errors.New("unable to type assert funding offer status")
		}
	}
	if data[14] != nil {
		if offer.Rate, ok = data[14].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer rate")
		}
	}
	if data[15] != nil {
		var period float64
		if period, ok = data[15].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer period")
		}
		offer.Period = int64(period)
	}
	if data[16] != nil {
		var notify float64
		if notify, ok = data[16].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer notify")
		}
		offer.Notify = notify == 1
	}
	if data[17] != nil {
		var hidden float64
		if hidden, ok = data[17].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer hidden")
		}
		offer.Hidden = hidden == 1
	}
	if data[19] != nil {
		var renew float64
		if renew, ok = data[19].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer renew")
		}
		offer.Renew = renew == 1
	}
	if includeRateReal && data[20] != nil {
		if offer.RateReal, ok = data[20].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer rate real")
		}
	}
	return &offer, nil
}

func wsHandleFundingCreditLoanData(data []interface{}, includePositionPair bool) (*WsCredit, error) {
	var credit WsCredit
	var ok bool
	if data[0] != nil {
		var id float64
		if id, ok = data[0].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit ID")
		}
		credit.ID = int64(id)
	}
	if data[1] != nil {
		if credit.Symbol, ok = data[1].(string); !ok {
			return nil, errors.New("unable to type assert funding credit symbol")
		}
	}
	if data[2] != nil {
		var side float64
		if side, ok = data[2].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit side")
		}
		credit.Side = int8(side)
	}
	if data[3] != nil {
		var created float64
		if created, ok = data[3].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit created")
		}
		credit.Created = time.UnixMilli(int64(created))
	}
	if data[4] != nil {
		var updated float64
		if updated, ok = data[4].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit updated")
		}
		credit.Updated = time.UnixMilli(int64(updated))
	}
	if data[5] != nil {
		if credit.Amount, ok = data[5].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit amount")
		}
	}
	if data[6] != nil {
		credit.Flags = data[6]
	}
	if data[7] != nil {
		if credit.Status, ok = data[7].(string); !ok {
			return nil, errors.New("unable to type assert funding credit status")
		}
	}
	if data[11] != nil {
		if credit.Rate, ok = data[11].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit rate")
		}
	}
	if data[12] != nil {
		var period float64
		if period, ok = data[12].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit period")
		}
		credit.Period = int64(period)
	}
	if data[13] != nil {
		var opened float64
		if opened, ok = data[13].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit opened")
		}
		credit.Opened = time.UnixMilli(int64(opened))
	}
	if data[14] != nil {
		var lastPayout float64
		if lastPayout, ok = data[14].(float64); !ok {
			return nil, errors.New("unable to type assert last funding credit payout")
		}
		credit.LastPayout = time.UnixMilli(int64(lastPayout))
	}
	if data[15] != nil {
		var notify float64
		if notify, ok = data[15].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit notify")
		}
		credit.Notify = notify == 1
	}
	if data[16] != nil {
		var hidden float64
		if hidden, ok = data[16].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit hidden")
		}
		credit.Hidden = hidden == 1
	}
	if data[18] != nil {
		var renew float64
		if renew, ok = data[18].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit renew")
		}
		credit.Renew = renew == 1
	}
	if data[19] != nil {
		if credit.RateReal, ok = data[19].(float64); !ok {
			return nil, errors.New("unable to type assert rate funding credit real")
		}
	}
	if data[20] != nil {
		var noClose float64
		if noClose, ok = data[20].(float64); !ok {
			return nil, errors.New("unable to type assert no funding credit close")
		}
		credit.NoClose = noClose == 1
	}
	if includePositionPair {
		if data[21] != nil {
			if credit.PositionPair, ok = data[21].(string); !ok {
				return nil, errors.New("unable to type assert funding credit position pair")
			}
		}
	}
	return &credit, nil
}

func (b *Bitfinex) wsHandleOrder(data []interface{}) {
	var od order.Detail
	var err error
	od.Exchange = b.Name
	if data[0] != nil {
		if id, ok := data[0].(float64); ok {
			od.OrderID = strconv.FormatFloat(id, 'f', -1, 64)
		}
	}
	if data[16] != nil {
		if price, ok := data[16].(float64); ok {
			od.Price = price
		}
	}
	if data[7] != nil {
		if amount, ok := data[7].(float64); ok {
			od.Amount = amount
		}
	}
	if data[6] != nil {
		if remainingAmount, ok := data[6].(float64); ok {
			od.RemainingAmount = remainingAmount
		}
	}
	if data[7] != nil && data[6] != nil {
		if executedAmount, ok := data[7].(float64); ok {
			od.ExecutedAmount = executedAmount - od.RemainingAmount
		}
	}
	if data[4] != nil {
		if date, ok := data[4].(float64); ok {
			od.Date = time.Unix(int64(date)*1000, 0)
		}
	}
	if data[5] != nil {
		if lastUpdated, ok := data[5].(float64); ok {
			od.LastUpdated = time.Unix(int64(lastUpdated)*1000, 0)
		}
	}
	if data[2] != nil {
		if p, ok := data[3].(string); ok {
			od.Pair, od.AssetType, err = b.GetRequestFormattedPairAndAssetType(p[1:])
			if err != nil {
				b.Websocket.DataHandler <- err
				return
			}
		}
	}
	if data[8] != nil {
		if ordType, ok := data[8].(string); ok {
			oType, err := order.StringToOrderType(ordType)
			if err != nil {
				b.Websocket.DataHandler <- order.ClassificationError{
					Exchange: b.Name,
					OrderID:  od.OrderID,
					Err:      err,
				}
			}
			od.Type = oType
		}
	}
	if data[13] != nil {
		if ordStatus, ok := data[13].(string); ok {
			oStatus, err := order.StringToOrderStatus(ordStatus)
			if err != nil {
				b.Websocket.DataHandler <- order.ClassificationError{
					Exchange: b.Name,
					OrderID:  od.OrderID,
					Err:      err,
				}
			}
			od.Status = oStatus
		}
	}
	b.Websocket.DataHandler <- &od
}

// WsInsertSnapshot add the initial orderbook snapshot when subscribed to a
// channel
func (b *Bitfinex) WsInsertSnapshot(p currency.Pair, assetType asset.Item, books []WebsocketBook, fundingRate bool) error {
	if len(books) == 0 {
		return errors.New("no orderbooks submitted")
	}
	var book orderbook.Base
	book.Bids = make(orderbook.Items, 0, len(books))
	book.Asks = make(orderbook.Items, 0, len(books))
	for i := range books {
		item := orderbook.Item{
			ID:     books[i].ID,
			Amount: books[i].Amount,
			Price:  books[i].Price,
			Period: books[i].Period,
		}
		if fundingRate {
			if item.Amount < 0 {
				item.Amount *= -1
				book.Bids = append(book.Bids, item)
			} else {
				book.Asks = append(book.Asks, item)
			}
		} else {
			if books[i].Amount > 0 {
				book.Bids = append(book.Bids, item)
			} else {
				item.Amount *= -1
				book.Asks = append(book.Asks, item)
			}
		}
	}

	book.Asset = assetType
	book.Pair = p
	book.Exchange = b.Name
	book.PriceDuplication = true
	book.IsFundingRate = fundingRate
	book.VerifyOrderbook = b.CanVerifyOrderbook
	book.LastUpdated = time.Now()
	return b.Websocket.Orderbook.LoadSnapshot(&book)
}

// WsUpdateOrderbook updates the orderbook list, removing and adding to the
// orderbook sides
func (b *Bitfinex) WsUpdateOrderbook(p currency.Pair, assetType asset.Item, book []WebsocketBook, channelID int, sequenceNo int64, fundingRate bool) error {
	orderbookUpdate := orderbook.Update{
		Asset:      assetType,
		Pair:       p,
		Bids:       make([]orderbook.Item, 0, len(book)),
		Asks:       make([]orderbook.Item, 0, len(book)),
		UpdateTime: time.Now(),
	}

	for i := range book {
		item := orderbook.Item{
			ID:     book[i].ID,
			Amount: book[i].Amount,
			Price:  book[i].Price,
			Period: book[i].Period,
		}

		if book[i].Price > 0 {
			orderbookUpdate.Action = orderbook.UpdateInsert
			if fundingRate {
				if book[i].Amount < 0 {
					item.Amount *= -1
					orderbookUpdate.Bids = append(orderbookUpdate.Bids, item)
				} else {
					orderbookUpdate.Asks = append(orderbookUpdate.Asks, item)
				}
			} else {
				if book[i].Amount > 0 {
					orderbookUpdate.Bids = append(orderbookUpdate.Bids, item)
				} else {
					item.Amount *= -1
					orderbookUpdate.Asks = append(orderbookUpdate.Asks, item)
				}
			}
		} else {
			orderbookUpdate.Action = orderbook.Delete
			if fundingRate {
				if book[i].Amount == 1 {
					// delete bid
					orderbookUpdate.Asks = append(orderbookUpdate.Asks, item)
				} else {
					// delete ask
					orderbookUpdate.Bids = append(orderbookUpdate.Bids, item)
				}
			} else {
				if book[i].Amount == 1 {
					// delete bid
					orderbookUpdate.Bids = append(orderbookUpdate.Bids, item)
				} else {
					// delete ask
					orderbookUpdate.Asks = append(orderbookUpdate.Asks, item)
				}
			}
		}
	}

	cMtx.Lock()
	checkme := checksumStore[channelID]
	if checkme == nil {
		cMtx.Unlock()
		return b.Websocket.Orderbook.Update(&orderbookUpdate)
	}
	checksumStore[channelID] = nil
	cMtx.Unlock()

	if checkme.Sequence+1 == sequenceNo {
		// Sequence numbers get dropped, if checksum is not in line with
		// sequence, do not check.
		ob, err := b.Websocket.Orderbook.GetOrderbook(p, assetType)
		if err != nil {
			return fmt.Errorf("cannot calculate websocket checksum: book not found for %s %s %w",
				p,
				assetType,
				err)
		}

		err = validateCRC32(ob, checkme.Token)
		if err != nil {
			return err
		}
	}

	return b.Websocket.Orderbook.Update(&orderbookUpdate)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *Bitfinex) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var wsPairFormat = currency.PairFormat{Uppercase: true}
	var channels = []string{wsBook, wsTrades, wsTicker, wsCandles}

	var subscriptions []stream.ChannelSubscription
	assets := b.GetAssetTypes(true)
	for i := range assets {
		enabledPairs, err := b.GetEnabledPairs(assets[i])
		if err != nil {
			return nil, err
		}

		for j := range channels {
			for k := range enabledPairs {
				params := make(map[string]interface{})
				if channels[j] == wsBook {
					params["prec"] = "R0"
					params["len"] = "100"
				}

				if channels[j] == wsCandles {
					// TODO: Add ability to select timescale && funding period
					var fundingPeriod string
					prefix := "t"
					if assets[i] == asset.MarginFunding {
						prefix = "f"
						fundingPeriod = ":p30"
					}
					params["key"] = "trade:1m:" + prefix + enabledPairs[k].String() + fundingPeriod
				} else {
					params["symbol"] = wsPairFormat.Format(enabledPairs[k])
				}

				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[j],
					Currency: enabledPairs[k],
					Params:   params,
					Asset:    assets[i],
				})
			}
		}
	}

	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (b *Bitfinex) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	checksum := make(map[string]interface{})
	checksum["event"] = "conf"
	checksum["flags"] = bitfinexChecksumFlag + bitfinexWsSequenceFlag
	err := b.Websocket.Conn.SendJSONMessage(checksum)
	if err != nil {
		return err
	}

	var errs error
	for i := range channelsToSubscribe {
		req := make(map[string]interface{})
		req["event"] = "subscribe"
		req["channel"] = channelsToSubscribe[i].Channel

		for k, v := range channelsToSubscribe[i].Params {
			req[k] = v
		}

		err := b.Websocket.Conn.SendJSONMessage(req)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		b.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	return errs
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bitfinex) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var errs error
	for i := range channelsToUnsubscribe {
		req := make(map[string]interface{})
		req["event"] = "unsubscribe"
		req["channel"] = channelsToUnsubscribe[i].Channel

		for k, v := range channelsToUnsubscribe[i].Params {
			req[k] = v
		}

		err := b.Websocket.Conn.SendJSONMessage(req)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		b.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	return errs
}

// WsSendAuth sends a authenticated event payload
func (b *Bitfinex) WsSendAuth(ctx context.Context) error {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}

	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	payload := "AUTH" + nonce

	hmac, err := crypto.GetHMAC(crypto.HashSHA512_384,
		[]byte(payload),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}
	request := WsAuthRequest{
		Event:         "auth",
		APIKey:        creds.Key,
		AuthPayload:   payload,
		AuthSig:       crypto.HexEncodeToString(hmac),
		AuthNonce:     nonce,
		DeadManSwitch: 0,
	}
	err = b.Websocket.AuthConn.SendJSONMessage(request)
	if err != nil {
		b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// WsAddSubscriptionChannel adds a new subscription channel to the
// WebsocketSubdChannels map in bitfinex.go (Bitfinex struct)
func (b *Bitfinex) WsAddSubscriptionChannel(chanID int, channel, pair string) {
	chanInfo := WebsocketChanInfo{Pair: pair, Channel: channel}
	b.WebsocketSubdChannels[chanID] = chanInfo

	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Subscribed to Channel: %s Pair: %s ChannelID: %d\n",
			b.Name,
			channel,
			pair,
			chanID)
	}
}

// WsNewOrder authenticated new order request
func (b *Bitfinex) WsNewOrder(data *WsNewOrderRequest) (string, error) {
	data.CustomID = b.Websocket.AuthConn.GenerateMessageID(false)
	request := makeRequestInterface(wsOrderNew, data)
	resp, err := b.Websocket.AuthConn.SendMessageReturnResponse(data.CustomID, request)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New(b.Name + " - Order message not returned")
	}
	var respData []interface{}
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return "", err
	}

	if len(respData) < 3 {
		return "", errors.New("unexpected respData length")
	}
	responseDataDetail, ok := respData[2].([]interface{})
	if !ok {
		return "", errors.New("unable to type assert respData")
	}

	if len(responseDataDetail) < 4 {
		return "", errors.New("invalid responseDataDetail length")
	}

	responseOrderDetail, ok := responseDataDetail[4].([]interface{})
	if !ok {
		return "", errors.New("unable to type assert responseOrderDetail")
	}
	var orderID string
	if responseOrderDetail[0] != nil {
		if ordID, ordOK := responseOrderDetail[0].(float64); ordOK && ordID > 0 {
			orderID = strconv.FormatFloat(ordID, 'f', -1, 64)
		}
	}
	var errorMessage, errCode string
	if len(responseDataDetail) > 6 {
		errCode, ok = responseDataDetail[6].(string)
		if !ok {
			return "", errors.New("unable to type assert errCode")
		}
	}
	if len(responseDataDetail) > 7 {
		errorMessage, ok = responseDataDetail[7].(string)
		if !ok {
			return "", errors.New("unable to type assert errorMessage")
		}
	}
	if strings.EqualFold(errCode, wsError) {
		return orderID, errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}
	return orderID, nil
}

// WsModifyOrder authenticated modify order request
func (b *Bitfinex) WsModifyOrder(data *WsUpdateOrderRequest) error {
	request := makeRequestInterface(wsOrderUpdate, data)
	resp, err := b.Websocket.AuthConn.SendMessageReturnResponse(data.OrderID, request)
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New(b.Name + " - Order message not returned")
	}

	var responseData []interface{}
	err = json.Unmarshal(resp, &responseData)
	if err != nil {
		return err
	}
	if len(responseData) < 3 {
		return errors.New("unexpected responseData length")
	}
	responseOrderData, ok := responseData[2].([]interface{})
	if !ok {
		return errors.New("unable to type assert responseOrderData")
	}
	var errorMessage, errCode string
	if len(responseOrderData) > 6 {
		errCode, ok = responseOrderData[6].(string)
		if !ok {
			return errors.New("unable to type assert errCode")
		}
	}
	if len(responseOrderData) > 7 {
		errorMessage, ok = responseOrderData[7].(string)
		if !ok {
			return errors.New("unable to type assert errorMessage")
		}
	}
	if strings.EqualFold(errCode, wsError) {
		return errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}
	return nil
}

// WsCancelMultiOrders authenticated cancel multi order request
func (b *Bitfinex) WsCancelMultiOrders(orderIDs []int64) error {
	cancel := WsCancelGroupOrdersRequest{
		OrderID: orderIDs,
	}
	request := makeRequestInterface(wsCancelMultipleOrders, cancel)
	return b.Websocket.AuthConn.SendJSONMessage(request)
}

// WsCancelOrder authenticated cancel order request
func (b *Bitfinex) WsCancelOrder(orderID int64) error {
	cancel := WsCancelOrderRequest{
		OrderID: orderID,
	}
	request := makeRequestInterface(wsOrderCancel, cancel)
	resp, err := b.Websocket.AuthConn.SendMessageReturnResponse(orderID, request)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("%v - Order %v failed to cancel", b.Name, orderID)
	}
	var responseData []interface{}
	err = json.Unmarshal(resp, &responseData)
	if err != nil {
		return err
	}
	if len(responseData) < 3 {
		return errors.New("unexpected responseData length")
	}
	responseOrderData, ok := responseData[2].([]interface{})
	if !ok {
		return errors.New("unable to type assert responseOrderData")
	}
	var errorMessage, errCode string
	if len(responseOrderData) > 6 {
		errCode, ok = responseOrderData[6].(string)
		if !ok {
			return errors.New("unable to type assert errCode")
		}
	}
	if len(responseOrderData) > 7 {
		errorMessage, ok = responseOrderData[7].(string)
		if !ok {
			return errors.New("unable to type assert errorMessage")
		}
	}
	if strings.EqualFold(errCode, wsError) {
		return errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}
	return nil
}

// WsCancelAllOrders authenticated cancel all orders request
func (b *Bitfinex) WsCancelAllOrders() error {
	cancelAll := WsCancelAllOrdersRequest{All: 1}
	request := makeRequestInterface(wsCancelMultipleOrders, cancelAll)
	return b.Websocket.AuthConn.SendJSONMessage(request)
}

// WsNewOffer authenticated new offer request
func (b *Bitfinex) WsNewOffer(data *WsNewOfferRequest) error {
	request := makeRequestInterface(wsFundingOfferNew, data)
	return b.Websocket.AuthConn.SendJSONMessage(request)
}

// WsCancelOffer authenticated cancel offer request
func (b *Bitfinex) WsCancelOffer(orderID int64) error {
	cancel := WsCancelOrderRequest{
		OrderID: orderID,
	}
	request := makeRequestInterface(wsFundingOfferCancel, cancel)
	resp, err := b.Websocket.AuthConn.SendMessageReturnResponse(orderID, request)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("%v - Order %v failed to cancel", b.Name, orderID)
	}
	var responseData []interface{}
	err = json.Unmarshal(resp, &responseData)
	if err != nil {
		return err
	}
	if len(responseData) < 3 {
		return errors.New("unexpected responseData length")
	}
	responseOrderData, ok := responseData[2].([]interface{})
	if !ok {
		return errors.New("unable to type assert responseOrderData")
	}
	var errorMessage, errCode string
	if len(responseOrderData) > 6 {
		errCode, ok = responseOrderData[6].(string)
		if !ok {
			return errors.New("unable to type assert errCode")
		}
	}
	if len(responseOrderData) > 7 {
		errorMessage, ok = responseOrderData[7].(string)
		if !ok {
			return errors.New("unable to type assert errorMessage")
		}
	}
	if strings.EqualFold(errCode, wsError) {
		return errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}

	return nil
}

func makeRequestInterface(channelName string, data interface{}) []interface{} {
	return []interface{}{0, channelName, nil, data}
}

func validateCRC32(book *orderbook.Base, token int) error {
	// Order ID's need to be sub-sorted in ascending order, this needs to be
	// done on the main book to ensure that we do not cut price levels out below
	reOrderByID(book.Bids)
	reOrderByID(book.Asks)

	// RO precision calculation is based on order ID's and amount values
	var bids, asks []orderbook.Item
	for i := 0; i < 25; i++ {
		if i < len(book.Bids) {
			bids = append(bids, book.Bids[i])
		}
		if i < len(book.Asks) {
			asks = append(asks, book.Asks[i])
		}
	}

	// ensure '-' (negative amount) is passed back to string buffer as
	// this is needed for calcs - These get swapped if funding rate
	bidmod := float64(1)
	if book.IsFundingRate {
		bidmod = -1
	}

	askMod := float64(-1)
	if book.IsFundingRate {
		askMod = 1
	}

	var check strings.Builder
	for i := 0; i < 25; i++ {
		if i < len(bids) {
			check.WriteString(strconv.FormatInt(bids[i].ID, 10))
			check.WriteString(":")
			check.WriteString(strconv.FormatFloat(bidmod*bids[i].Amount, 'f', -1, 64))
			check.WriteString(":")
		}

		if i < len(asks) {
			check.WriteString(strconv.FormatInt(asks[i].ID, 10))
			check.WriteString(":")
			check.WriteString(strconv.FormatFloat(askMod*asks[i].Amount, 'f', -1, 64))
			check.WriteString(":")
		}
	}

	checksumStr := strings.TrimSuffix(check.String(), ":")
	checksum := crc32.ChecksumIEEE([]byte(checksumStr))
	if checksum == uint32(token) {
		return nil
	}
	return fmt.Errorf("invalid checksum for %s %s: calculated [%d] does not match [%d]",
		book.Asset,
		book.Pair,
		checksum,
		uint32(token))
}

// reOrderByID sub sorts orderbook items by its corresponding ID when price
// levels are the same. TODO: Deprecate and shift to buffer level insertion
// based off ascending ID.
func reOrderByID(depth []orderbook.Item) {
subSort:
	for x := 0; x < len(depth); {
		var subset []orderbook.Item
		// Traverse forward elements
		for y := x + 1; y < len(depth); y++ {
			if depth[x].Price == depth[y].Price &&
				// Period matching is for funding rates, this was undocumented
				// but these need to be matched with price for the correct ID
				// alignment
				depth[x].Period == depth[y].Period {
				// Append element to subset when price match occurs
				subset = append(subset, depth[y])
				// Traverse next
				continue
			}
			if len(subset) != 0 {
				// Append root element
				subset = append(subset, depth[x])
				// Sort IDs by ascending
				sort.Slice(subset, func(i, j int) bool {
					return subset[i].ID < subset[j].ID
				})
				// Re-align elements with sorted ID subset
				for z := range subset {
					depth[x+z] = subset[z]
				}
			}
			// When price is not matching change checked element to root
			x = y
			continue subSort
		}
		break
	}
}
