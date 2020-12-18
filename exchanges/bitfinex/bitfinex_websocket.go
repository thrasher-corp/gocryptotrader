package bitfinex

import (
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
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
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
		go b.wsReadData(b.Websocket.AuthConn)
		err = b.WsSendAuth()
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v - authentication failed: %v\n",
				b.Name,
				err)
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	subs, err := b.GenerateDefaultSubscriptions()
	if err != nil {
		return err
	}
	go b.WsDataHandler()
	return b.Websocket.SubscribeToChannels(subs)
}

// wsReadData receives and passes on websocket messages for processing
func (b *Bitfinex) wsReadData(ws stream.Connection) {
	b.Websocket.Wg.Add(1)
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
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()
	for {
		select {
		case resp := <-comms:
			if resp.Type == websocket.TextMessage {
				err := b.wsHandleData(resp.Raw)
				if err != nil {
					b.Websocket.DataHandler <- err
				}
			}
		case <-b.Websocket.ShutdownC:
			return
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
			if symbol, ok := d["symbol"].(string); ok {
				b.WsAddSubscriptionChannel(int(d["chanId"].(float64)),
					d["channel"].(string),
					symbol,
				)
			} else if key, ok := d["key"].(string); ok {
				// Capture trading subscriptions
				contents := strings.Split(d["key"].(string), ":")
				if len(contents) > 3 {
					// Edge case to parse margin strings.
					// map[chanId:139136 channel:candles event:subscribed key:trade:1m:tXAUTF0:USTF0]
					if contents[2][0] == 't' {
						key = contents[2] + ":" + contents[3]
					}
				}
				b.WsAddSubscriptionChannel(int(d["chanId"].(float64)),
					d["channel"].(string),
					key,
				)
			}
		case "auth":
			status := d["status"].(string)
			if status == "OK" {
				b.Websocket.DataHandler <- d
				b.WsAddSubscriptionChannel(0, "account", "N/A")
			} else if status == "fail" {
				return fmt.Errorf("bitfinex.go error - Websocket unable to AUTH. Error code: %s",
					d["code"].(string))
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
			return fmt.Errorf("bitfinex.go error - Unable to locate chanID: %d",
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
		case len(pairInfo) == 1:
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
					data := obSnapBundle[i].([]interface{})
					id, okAssert := data[0].(float64)
					if !okAssert {
						return errors.New("type assertion failed for orderbook item data")
					}
					pricePeriod, okAssert := data[1].(float64)
					if !okAssert {
						return errors.New("type assertion failed for orderbook item data")
					}
					rateAmount, okAssert := data[2].(float64)
					if !okAssert {
						return errors.New("type assertion failed for orderbook item data")
					}
					if len(data) == 4 {
						fundingRate = true
						amount, okFunding := data[3].(float64)
						if !okFunding {
							return errors.New("type assertion failed for orderbook item data")
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
				err := b.WsInsertSnapshot(pair, chanAsset, newOrderbook, fundingRate)
				if err != nil {
					return fmt.Errorf("bitfinex_websocket.go inserting snapshot error: %s",
						err)
				}
			case float64:
				pricePeriod, okSnap := obSnapBundle[1].(float64)
				if !okSnap {
					return errors.New("type assertion failed for orderbook snapshot data")
				}
				amountRate, okSnap := obSnapBundle[2].(float64)
				if !okSnap {
					return errors.New("type assertion failed for orderbook snapshot data")
				}
				if len(obSnapBundle) == 4 {
					fundingRate = true
					var amount float64
					amount, okSnap = obSnapBundle[3].(float64)
					if !okSnap {
						return errors.New("type assertion failed for orderbook snapshot data")
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

				err := b.WsUpdateOrderbook(pair, chanAsset, newOrderbook, chanID, int64(sequenceNo), fundingRate)
				if err != nil {
					return fmt.Errorf("bitfinex_websocket.go updating orderbook error: %s",
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
						element := candleBundle[i].([]interface{})
						b.Websocket.DataHandler <- stream.KlineData{
							Timestamp:  time.Unix(0, int64(element[0].(float64))*int64(time.Millisecond)),
							Exchange:   b.Name,
							AssetType:  chanAsset,
							Pair:       pair,
							OpenPrice:  element[1].(float64),
							ClosePrice: element[2].(float64),
							HighPrice:  element[3].(float64),
							LowPrice:   element[4].(float64),
							Volume:     element[5].(float64),
						}
					}

				case float64:
					b.Websocket.DataHandler <- stream.KlineData{
						Timestamp:  time.Unix(0, int64(candleData)*int64(time.Millisecond)),
						Exchange:   b.Name,
						AssetType:  chanAsset,
						Pair:       pair,
						OpenPrice:  candleBundle[1].(float64),
						ClosePrice: candleBundle[2].(float64),
						HighPrice:  candleBundle[3].(float64),
						LowPrice:   candleBundle[4].(float64),
						Volume:     candleBundle[5].(float64),
					}
				}
			}
			return nil
		case wsTicker:
			tickerData := d[1].([]interface{})
			b.Websocket.DataHandler <- &ticker.Price{
				ExchangeName: b.Name,
				Bid:          tickerData[0].(float64),
				Ask:          tickerData[2].(float64),
				Last:         tickerData[6].(float64),
				Volume:       tickerData[7].(float64),
				High:         tickerData[8].(float64),
				Low:          tickerData[9].(float64),
				AssetType:    chanAsset,
				Pair:         pair,
			}
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
				snapshot := d[1].([]interface{})
				for i := range snapshot {
					elem := snapshot[i].([]interface{})
					if len(elem) == 5 {
						tradeHolder = append(tradeHolder, WebsocketTrade{
							ID:        int64(elem[0].(float64)),
							Timestamp: int64(elem[1].(float64)),
							Amount:    elem[2].(float64),
							Rate:      elem[3].(float64),
							Period:    int64(elem[4].(float64)),
						})
					} else {
						tradeHolder = append(tradeHolder, WebsocketTrade{
							ID:        int64(elem[0].(float64)),
							Timestamp: int64(elem[1].(float64)),
							Amount:    elem[2].(float64),
							Price:     elem[3].(float64),
						})
					}
				}
			case 3:
				if d[1].(string) != wsFundingTradeUpdate &&
					d[1].(string) != wsTradeExecutionUpdate {
					return nil
				}
				data := d[2].([]interface{})
				if len(data) == 5 {
					tradeHolder = append(tradeHolder, WebsocketTrade{
						ID:        int64(data[0].(float64)),
						Timestamp: int64(data[1].(float64)),
						Amount:    data[2].(float64),
						Rate:      data[3].(float64),
						Period:    int64(data[4].(float64)),
					})
				} else {
					tradeHolder = append(tradeHolder, WebsocketTrade{
						ID:        int64(data[0].(float64)),
						Timestamp: int64(data[1].(float64)),
						Amount:    data[2].(float64),
						Price:     data[3].(float64),
					})
				}
			}
			var trades []trade.Data
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
				trades = append(trades, trade.Data{
					TID:          strconv.FormatInt(tradeHolder[i].ID, 10),
					CurrencyPair: pair,
					Timestamp:    time.Unix(0, tradeHolder[i].Timestamp*int64(time.Millisecond)),
					Price:        price,
					Amount:       newAmount,
					Exchange:     b.Name,
					AssetType:    chanAsset,
					Side:         side,
				})
			}

			return b.AddTradesToBuffer(trades...)
		}

		if authResp, ok := d[1].(string); ok {
			switch authResp {
			case wsHeartbeat, pong:
				return nil
			case wsNotification:
				notification := d[2].([]interface{})
				if data, ok := notification[4].([]interface{}); ok {
					channelName := notification[1].(string)
					switch {
					case strings.Contains(channelName, wsFundingOrderNewRequest),
						strings.Contains(channelName, wsFundingOrderUpdateRequest),
						strings.Contains(channelName, wsFundingOrderCancelRequest):
						if data[0] != nil && data[0].(float64) > 0 {
							id := int64(data[0].(float64))
							if b.Websocket.Match.IncomingWithData(id, respRaw) {
								return nil
							}
							b.wsHandleFundingOffer(data)
						}
					case strings.Contains(channelName, wsOrderNewRequest),
						strings.Contains(channelName, wsOrderUpdateRequest),
						strings.Contains(channelName, wsOrderCancelRequest):
						if data[2] != nil && data[2].(float64) > 0 {
							id := int64(data[2].(float64))
							if b.Websocket.Match.IncomingWithData(id, respRaw) {
								return nil
							}
							b.wsHandleOrder(data)
						}

					default:
						return fmt.Errorf("%s - Unexpected data returned %s",
							b.Name,
							respRaw)
					}
				}
				if notification[5] != nil &&
					strings.EqualFold(notification[5].(string), wsError) {
					return fmt.Errorf("%s - Error %s",
						b.Name,
						notification[6].(string))
				}
			case wsOrderSnapshot:
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						for i := range snapBundle {
							positionData := snapBundle[i].([]interface{})
							b.wsHandleOrder(positionData)
						}
					}
				}
			case wsOrderCancel, wsOrderNew, wsOrderUpdate:
				if oData, ok := d[2].([]interface{}); ok && len(oData) > 0 {
					b.wsHandleOrder(oData)
				}
			case wsPositionSnapshot:
				var snapshot []WebsocketPosition
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						for i := range snapBundle {
							positionData := snapBundle[i].([]interface{})
							position := WebsocketPosition{
								Pair:              positionData[0].(string),
								Status:            positionData[1].(string),
								Amount:            positionData[2].(float64),
								Price:             positionData[3].(float64),
								MarginFunding:     positionData[4].(float64),
								MarginFundingType: int64(positionData[5].(float64)),
								ProfitLoss:        positionData[6].(float64),
								ProfitLossPercent: positionData[7].(float64),
								LiquidationPrice:  positionData[8].(float64),
								Leverage:          positionData[9].(float64),
							}
							snapshot = append(snapshot, position)
						}
						b.Websocket.DataHandler <- snapshot
					}
				}
			case wsPositionNew, wsPositionUpdate, wsPositionClose:
				if positionData, ok := d[2].([]interface{}); ok && len(positionData) > 0 {
					position := WebsocketPosition{
						Pair:              positionData[0].(string),
						Status:            positionData[1].(string),
						Amount:            positionData[2].(float64),
						Price:             positionData[3].(float64),
						MarginFunding:     positionData[4].(float64),
						MarginFundingType: int64(positionData[5].(float64)),
						ProfitLoss:        positionData[6].(float64),
						ProfitLossPercent: positionData[7].(float64),
						LiquidationPrice:  positionData[8].(float64),
						Leverage:          positionData[9].(float64),
					}
					b.Websocket.DataHandler <- position
				}
			case wsTradeExecuted, wsTradeExecutionUpdate:
				if tradeData, ok := d[2].([]interface{}); ok && len(tradeData) > 4 {
					b.Websocket.DataHandler <- WebsocketTradeData{
						TradeID:        int64(tradeData[0].(float64)),
						Pair:           tradeData[1].(string),
						Timestamp:      int64(tradeData[2].(float64)),
						OrderID:        int64(tradeData[3].(float64)),
						AmountExecuted: tradeData[4].(float64),
						PriceExecuted:  tradeData[5].(float64),
						OrderType:      tradeData[6].(string),
						OrderPrice:     tradeData[7].(float64),
						Maker:          tradeData[8].(float64) == 1,
						Fee:            tradeData[9].(float64),
						FeeCurrency:    tradeData[10].(string),
					}
				}
			case wsFundingOrderSnapshot:
				var snapshot []WsFundingOffer
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						for i := range snapBundle {
							data := snapBundle[i].([]interface{})
							offer := WsFundingOffer{
								ID:             int64(data[0].(float64)),
								Symbol:         data[1].(string),
								Created:        int64(data[2].(float64)),
								Updated:        int64(data[3].(float64)),
								Amount:         data[4].(float64),
								OriginalAmount: data[5].(float64),
								Type:           data[6].(string),
								Flags:          data[9].(float64),
								Status:         data[10].(string),
								Rate:           data[14].(float64),
								Period:         int64(data[15].(float64)),
								Notify:         data[16].(float64) == 1,
								Hidden:         data[17].(float64) == 1,
								Insure:         data[18].(float64) == 1,
								Renew:          data[19].(float64) == 1,
								RateReal:       data[20].(float64),
							}
							snapshot = append(snapshot, offer)
						}
						b.Websocket.DataHandler <- snapshot
					}
				}
			case wsFundingOrderNew, wsFundingOrderUpdate, wsFundingOrderCancel:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					b.wsHandleFundingOffer(data)
				}
			case wsFundingCreditSnapshot:
				var snapshot []WsCredit
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						for i := range snapBundle {
							data := snapBundle[i].([]interface{})
							credit := WsCredit{
								ID:           int64(data[0].(float64)),
								Symbol:       data[1].(string),
								Side:         data[2].(string),
								Created:      int64(data[3].(float64)),
								Updated:      int64(data[4].(float64)),
								Amount:       data[5].(float64),
								Flags:        data[6].(string),
								Status:       data[7].(string),
								Rate:         data[11].(float64),
								Period:       int64(data[12].(float64)),
								Opened:       int64(data[13].(float64)),
								LastPayout:   int64(data[14].(float64)),
								Notify:       data[15].(float64) == 1,
								Hidden:       data[16].(float64) == 1,
								Insure:       data[17].(float64) == 1,
								Renew:        data[18].(float64) == 1,
								RateReal:     data[19].(float64),
								NoClose:      data[20].(float64) == 1,
								PositionPair: data[21].(string),
							}
							snapshot = append(snapshot, credit)
						}
						b.Websocket.DataHandler <- snapshot
					}
				}
			case wsFundingCreditNew, wsFundingCreditUpdate, wsFundingCreditCancel:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					b.Websocket.DataHandler <- WsCredit{
						ID:           int64(data[0].(float64)),
						Symbol:       data[1].(string),
						Side:         data[2].(string),
						Created:      int64(data[3].(float64)),
						Updated:      int64(data[4].(float64)),
						Amount:       data[5].(float64),
						Flags:        data[6].(string),
						Status:       data[7].(string),
						Rate:         data[11].(float64),
						Period:       int64(data[12].(float64)),
						Opened:       int64(data[13].(float64)),
						LastPayout:   int64(data[14].(float64)),
						Notify:       data[15].(float64) == 1,
						Hidden:       data[16].(float64) == 1,
						Insure:       data[17].(float64) == 1,
						Renew:        data[18].(float64) == 1,
						RateReal:     data[19].(float64),
						NoClose:      data[20].(float64) == 1,
						PositionPair: data[21].(string),
					}
				}
			case wsFundingLoanSnapshot:
				var snapshot []WsCredit
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						for i := range snapBundle {
							data := snapBundle[i].([]interface{})
							credit := WsCredit{
								ID:         int64(data[0].(float64)),
								Symbol:     data[1].(string),
								Side:       data[2].(string),
								Created:    int64(data[3].(float64)),
								Updated:    int64(data[4].(float64)),
								Amount:     data[5].(float64),
								Flags:      data[6].(string),
								Status:     data[7].(string),
								Rate:       data[11].(float64),
								Period:     int64(data[12].(float64)),
								Opened:     int64(data[13].(float64)),
								LastPayout: int64(data[14].(float64)),
								Notify:     data[15].(float64) == 1,
								Hidden:     data[16].(float64) == 1,
								Insure:     data[17].(float64) == 1,
								Renew:      data[18].(float64) == 1,
								RateReal:   data[19].(float64),
								NoClose:    data[20].(float64) == 1,
							}
							snapshot = append(snapshot, credit)
						}
						b.Websocket.DataHandler <- snapshot
					}
				}
			case wsFundingLoanNew, wsFundingLoanUpdate, wsFundingLoanCancel:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					b.Websocket.DataHandler <- WsCredit{
						ID:         int64(data[0].(float64)),
						Symbol:     data[1].(string),
						Side:       data[2].(string),
						Created:    int64(data[3].(float64)),
						Updated:    int64(data[4].(float64)),
						Amount:     data[5].(float64),
						Flags:      data[6].(string),
						Status:     data[7].(string),
						Rate:       data[11].(float64),
						Period:     int64(data[12].(float64)),
						Opened:     int64(data[13].(float64)),
						LastPayout: int64(data[14].(float64)),
						Notify:     data[15].(float64) == 1,
						Hidden:     data[16].(float64) == 1,
						Insure:     data[17].(float64) == 1,
						Renew:      data[18].(float64) == 1,
						RateReal:   data[19].(float64),
						NoClose:    data[20].(float64) == 1,
					}
				}
			case wsWalletSnapshot:
				var snapshot []WsWallet
				if snapBundle, ok := d[2].([]interface{}); ok && len(snapBundle) > 0 {
					if _, ok := snapBundle[0].([]interface{}); ok {
						for i := range snapBundle {
							data := snapBundle[i].([]interface{})
							var balanceAvailable float64
							if _, ok := data[4].(float64); ok {
								balanceAvailable = data[4].(float64)
							}
							wallet := WsWallet{
								Type:              data[0].(string),
								Currency:          data[1].(string),
								Balance:           data[2].(float64),
								UnsettledInterest: data[3].(float64),
								BalanceAvailable:  balanceAvailable,
							}
							snapshot = append(snapshot, wallet)
						}
						b.Websocket.DataHandler <- snapshot
					}
				}
			case wsWalletUpdate:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					var balanceAvailable float64
					if _, ok := data[4].(float64); ok {
						balanceAvailable = data[4].(float64)
					}
					b.Websocket.DataHandler <- WsWallet{
						Type:              data[0].(string),
						Currency:          data[1].(string),
						Balance:           data[2].(float64),
						UnsettledInterest: data[3].(float64),
						BalanceAvailable:  balanceAvailable,
					}
				}
			case wsBalanceUpdate:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					b.Websocket.DataHandler <- WsBalanceInfo{
						TotalAssetsUnderManagement: data[0].(float64),
						NetAssetsUnderManagement:   data[1].(float64),
					}
				}
			case wsMarginInfoUpdate:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					if data[0].(string) == "base" {
						if infoBase, ok := d[2].([]interface{}); ok && len(infoBase) > 0 {
							baseData := data[1].([]interface{})
							b.Websocket.DataHandler <- WsMarginInfoBase{
								UserProfitLoss: baseData[0].(float64),
								UserSwaps:      baseData[1].(float64),
								MarginBalance:  baseData[2].(float64),
								MarginNet:      baseData[3].(float64),
							}
						}
					}
				}
			case wsFundingInfoUpdate:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					if data[0].(string) == "sym" {
						symbolData := data[1].([]interface{})
						b.Websocket.DataHandler <- WsFundingInfo{
							YieldLoan:    symbolData[0].(float64),
							YieldLend:    symbolData[1].(float64),
							DurationLoan: symbolData[2].(float64),
							DurationLend: symbolData[3].(float64),
						}
					}
				}
			case wsFundingTradeExecuted, wsFundingTradeUpdate:
				if data, ok := d[2].([]interface{}); ok && len(data) > 0 {
					b.Websocket.DataHandler <- WsFundingTrade{
						ID:         int64(data[0].(float64)),
						Symbol:     data[1].(string),
						MTSCreated: int64(data[2].(float64)),
						OfferID:    int64(data[3].(float64)),
						Amount:     data[4].(float64),
						Rate:       data[5].(float64),
						Period:     int64(data[6].(float64)),
						Maker:      data[7].(float64) == 1,
					}
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

func (b *Bitfinex) wsHandleFundingOffer(data []interface{}) {
	var fo WsFundingOffer
	if data[0] != nil {
		fo.ID = int64(data[0].(float64))
	}
	if data[1] != nil {
		fo.Symbol = data[1].(string)[1:]
	}
	if data[2] != nil {
		fo.Created = int64(data[2].(float64))
	}
	if data[3] != nil {
		fo.Updated = int64(data[0].(float64))
	}
	if data[15] != nil {
		fo.Period = int64(data[15].(float64))
	}
	if data[4] != nil {
		fo.Amount = data[4].(float64)
	}
	if data[5] != nil {
		fo.OriginalAmount = data[5].(float64)
	}
	if data[6] != nil {
		fo.Type = data[6].(string)
	}
	if data[9] != nil {
		fo.Flags = data[9].(float64)
	}
	if data[9] != nil {
		fo.Status = data[10].(string)
	}
	if data[9] != nil {
		fo.Rate = data[14].(float64)
	}
	if data[16] != nil {
		fo.Notify = data[16].(float64) == 1
	}
	if data[17] != nil {
		fo.Hidden = data[17].(float64) == 1
	}
	if data[18] != nil {
		fo.Insure = data[18].(float64) == 1
	}
	if data[19] != nil {
		fo.Renew = data[19].(float64) == 1
	}
	if data[20] != nil {
		fo.RateReal = data[20].(float64)
	}

	b.Websocket.DataHandler <- fo
}

func (b *Bitfinex) wsHandleOrder(data []interface{}) {
	var od order.Detail
	var err error
	od.Exchange = b.Name
	if data[0] != nil {
		od.ID = strconv.FormatFloat(data[0].(float64), 'f', -1, 64)
	}
	if data[16] != nil {
		od.Price = data[16].(float64)
	}
	if data[7] != nil {
		od.Amount = data[7].(float64)
	}
	if data[6] != nil {
		od.RemainingAmount = data[6].(float64)
	}
	if data[7] != nil && data[6] != nil {
		od.ExecutedAmount = data[7].(float64) - data[6].(float64)
	}
	if data[4] != nil {
		od.Date = time.Unix(int64(data[4].(float64))*1000, 0)
	}
	if data[5] != nil {
		od.LastUpdated = time.Unix(int64(data[5].(float64))*1000, 0)
	}
	if data[2] != nil {
		od.Pair, od.AssetType, err = b.GetRequestFormattedPairAndAssetType(data[3].(string)[1:])
		if err != nil {
			b.Websocket.DataHandler <- err
			return
		}
	}
	if data[8] != nil {
		oType, err := order.StringToOrderType(data[8].(string))
		if err != nil {
			b.Websocket.DataHandler <- order.ClassificationError{
				Exchange: b.Name,
				OrderID:  od.ID,
				Err:      err,
			}
		}
		od.Type = oType
	}
	if data[13] != nil {
		oStatus, err := order.StringToOrderStatus(data[13].(string))
		if err != nil {
			b.Websocket.DataHandler <- order.ClassificationError{
				Exchange: b.Name,
				OrderID:  od.ID,
				Err:      err,
			}
		}
		od.Status = oStatus
	}
	b.Websocket.DataHandler <- &od
}

// WsInsertSnapshot add the initial orderbook snapshot when subscribed to a
// channel
func (b *Bitfinex) WsInsertSnapshot(p currency.Pair, assetType asset.Item, books []WebsocketBook, fundingRate bool) error {
	if len(books) == 0 {
		return errors.New("bitfinex.go error - no orderbooks submitted")
	}
	var book orderbook.Base
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

	book.AssetType = assetType
	book.Pair = p
	book.ExchangeName = b.Name
	book.NotAggregated = true
	book.IsFundingRate = fundingRate
	return b.Websocket.Orderbook.LoadSnapshot(&book)
}

// WsUpdateOrderbook updates the orderbook list, removing and adding to the
// orderbook sides
func (b *Bitfinex) WsUpdateOrderbook(p currency.Pair, assetType asset.Item, book []WebsocketBook, channelID int, sequenceNo int64, fundingRate bool) error {
	orderbookUpdate := buffer.Update{Asset: assetType, Pair: p}

	for i := range book {
		item := orderbook.Item{
			ID:     book[i].ID,
			Amount: book[i].Amount,
			Price:  book[i].Price,
			Period: book[i].Period,
		}

		if book[i].Price > 0 {
			orderbookUpdate.Action = buffer.UpdateInsert
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
			orderbookUpdate.Action = buffer.Delete
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
		ob := b.Websocket.Orderbook.GetOrderbook(p, assetType)
		if ob == nil {
			return fmt.Errorf("cannot calculate websocket checksum: book not found for %s %s",
				p,
				assetType)
		}

		err := validateCRC32(ob, checkme.Token)
		if err != nil {
			return err
		}
	}

	return b.Websocket.Orderbook.Update(&orderbookUpdate)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *Bitfinex) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{
		wsBook,
		wsTrades,
		wsTicker,
		wsCandles,
	}

	var subscriptions []stream.ChannelSubscription
	assets := b.GetAssetTypes()
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
					params["symbol"] = enabledPairs[k].String()
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
	var errs common.Errors
	checksum := make(map[string]interface{})
	checksum["event"] = "conf"
	checksum["flags"] = bitfinexChecksumFlag + bitfinexWsSequenceFlag
	err := b.Websocket.Conn.SendJSONMessage(checksum)
	if err != nil {
		return err
	}

	for i := range channelsToSubscribe {
		req := make(map[string]interface{})
		req["event"] = "subscribe"
		req["channel"] = channelsToSubscribe[i].Channel

		for k, v := range channelsToSubscribe[i].Params {
			req[k] = v
		}

		err := b.Websocket.Conn.SendJSONMessage(req)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		b.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bitfinex) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var errs common.Errors
	for i := range channelsToUnsubscribe {
		req := make(map[string]interface{})
		req["event"] = "unsubscribe"
		req["channel"] = channelsToUnsubscribe[i].Channel

		for k, v := range channelsToUnsubscribe[i].Params {
			req[k] = v
		}

		err := b.Websocket.Conn.SendJSONMessage(req)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		b.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

// WsSendAuth sends a autheticated event payload
func (b *Bitfinex) WsSendAuth() error {
	if !b.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled",
			b.Name)
	}
	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	payload := "AUTH" + nonce
	request := WsAuthRequest{
		Event:       "auth",
		APIKey:      b.API.Credentials.Key,
		AuthPayload: payload,
		AuthSig: crypto.HexEncodeToString(crypto.GetHMAC(crypto.HashSHA512_384,
			[]byte(payload),
			[]byte(b.API.Credentials.Secret))),
		AuthNonce:     nonce,
		DeadManSwitch: 0,
	}
	err := b.Websocket.AuthConn.SendJSONMessage(request)
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
	responseDataDetail := respData[2].([]interface{})
	responseOrderDetail := responseDataDetail[4].([]interface{})
	var orderID string
	if responseOrderDetail[0] != nil && responseOrderDetail[0].(float64) > 0 {
		orderID = strconv.FormatFloat(responseOrderDetail[0].(float64), 'f', -1, 64)
	}
	errCode := responseDataDetail[6].(string)
	errorMessage := responseDataDetail[7].(string)

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
	responseOrderData := responseData[2].([]interface{})
	errCode := responseOrderData[6].(string)
	errorMessage := responseOrderData[7].(string)
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
	responseOrderData := responseData[2].([]interface{})
	errCode := responseOrderData[6].(string)
	errorMessage := responseOrderData[7].(string)
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
	request := makeRequestInterface(wsFundingOrderNew, data)
	return b.Websocket.AuthConn.SendJSONMessage(request)
}

// WsCancelOffer authenticated cancel offer request
func (b *Bitfinex) WsCancelOffer(orderID int64) error {
	cancel := WsCancelOrderRequest{
		OrderID: orderID,
	}
	request := makeRequestInterface(wsFundingOrderCancel, cancel)
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
	responseOrderData := responseData[2].([]interface{})
	errCode := responseOrderData[6].(string)
	var errorMessage string
	if responseOrderData[7] != nil {
		errorMessage = responseOrderData[7].(string)
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
	return fmt.Errorf("invalid checksum %d, expected %d",
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
			if depth[x].Price == depth[y].Price {
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
