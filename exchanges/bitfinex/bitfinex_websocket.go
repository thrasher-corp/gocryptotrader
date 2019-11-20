package bitfinex

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

var comms = make(chan wshandler.WebsocketResponse, 1)

// WsConnect starts a new websocket connection
func (b *Bitfinex) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	err := b.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v unable to connect to Websocket. Error: %s", b.Name, err)
	}
	go b.WsReadData(b.WebsocketConn)

	if b.Websocket.CanUseAuthenticatedEndpoints() {
		err = b.AuthenticatedWebsocketConn.Dial(&dialer, http.Header{})
		if err != nil {
			log.Errorf(log.ExchangeSys, "%v unable to connect to authenticated Websocket. Error: %s", b.Name, err)
			b.API.AuthenticatedWebsocketSupport = false
		}
		go b.WsReadData(b.AuthenticatedWebsocketConn)
		err = b.WsSendAuth()
		if err != nil {
			log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", b.Name, err)
			b.API.AuthenticatedWebsocketSupport = false
		}
	}

	b.GenerateDefaultSubscriptions()
	go b.WsDataHandler()
	return nil
}

// WsReadData funnels both auth and public ws data into one manageable place
func (b *Bitfinex) WsReadData(ws *wshandler.WebsocketConnection) {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()
	for {
		select {
		case <-b.Websocket.ShutdownC:
			return
		default:
			resp, err := ws.ReadMessage()
			if err != nil {
				b.Websocket.DataHandler <- err
				return
			}
			b.Websocket.TrafficAlert <- struct{}{}
			comms <- resp
		}
	}
}

// WsDataHandler handles data from WsReadData
func (b *Bitfinex) WsDataHandler() {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return
		default:
			stream := <-comms
			if stream.Type == websocket.TextMessage {
				var result interface{}
				err = json.Unmarshal(stream.Raw, &result)
				if err != nil {
					b.Websocket.DataHandler <- err
					return
				}
				switch reflect.TypeOf(result).String() {
				case "map[string]interface {}":
					eventData := result.(map[string]interface{})
					event := eventData["event"]
					switch event {
					case "subscribed":
						if symbol, ok := eventData["pair"].(string); ok {
							b.WsAddSubscriptionChannel(int(eventData["chanId"].(float64)),
								eventData["channel"].(string),
								symbol,
							)
						} else if key, ok := eventData["key"].(string); ok {
							b.WsAddSubscriptionChannel(int(eventData["chanId"].(float64)),
								eventData["channel"].(string),
								key,
							)
						}
					case "auth":
						status := eventData["status"].(string)
						if status == "OK" {
							b.Websocket.DataHandler <- eventData
							b.WsAddSubscriptionChannel(0, "account", "N/A")
						} else if status == "fail" {
							b.Websocket.DataHandler <- fmt.Errorf("bitfinex.go error - Websocket unable to AUTH. Error code: %s",
								eventData["code"].(string))
						}
					}
				case "[]interface {}":
					chanData := result.([]interface{})
					chanID := int(chanData[0].(float64))

					chanInfo, ok := b.WebsocketSubdChannels[chanID]
					if !ok {
						b.Websocket.DataHandler <- fmt.Errorf("bitfinex.go error - Unable to locate chanID: %d",
							chanID)
						continue
					}
					if len(chanData) == 2 {
						if reflect.TypeOf(chanData[1]).String() == "string" {
							if chanData[1].(string) == websocketHeartbeat {
								continue
							} else if chanData[1].(string) == "pong" {
								continue
							}
						}
					}
					if _, ok := chanData[1].(string); ok {
						switch chanData[1].(string) {
						// Notifications channel provides updates from requests
						case notification:
							notification := chanData[2].([]interface{})
							if data, ok := notification[4].([]interface{}); ok {
								channelName := notification[1].(string)
								switch {
								case strings.Contains(channelName, orderUpdate),
									strings.Contains(channelName, orderCancel),
									strings.Contains(channelName, fundingOrderCancel):
									if data[0] != nil && data[0].(float64) > 0 {
										b.AuthenticatedWebsocketConn.AddResponseWithID(int64(data[0].(float64)), stream.Raw)
										continue
									}
								case strings.Contains(channelName, orderNew):
									if data[2] != nil && data[2].(float64) > 0 {
										b.AuthenticatedWebsocketConn.AddResponseWithID(int64(data[2].(float64)), stream.Raw)
										continue
									}
								}
								b.Websocket.DataHandler <- fmt.Errorf("%s - Unexpected data returned %s", b.Name, stream.Raw)
								continue
							}
							if notification[5] != nil && strings.EqualFold(notification[5].(string), "ERROR") {
								b.Websocket.DataHandler <- fmt.Errorf("%s - Error %s", b.Name, notification[6].(string))
							}
						case websocketPositionSnapshot:
							var snapshot []WebsocketPosition
							if snapBundle, ok := chanData[2].([]interface{}); ok && len(snapBundle) > 0 {
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
						case websocketPositionNew, websocketPositionUpdate, websocketPositionClose:
							if positionData, ok := chanData[2].([]interface{}); ok && len(positionData) > 0 {
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
						case websocketTradeExecutionUpdate:
							if tradeData, ok := chanData[2].([]interface{}); ok && len(tradeData) > 4 {
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
						case fos:
							var snapshot []WsFundingOffer
							if snapBundle, ok := chanData[2].([]interface{}); ok && len(snapBundle) > 0 {
								if _, ok := snapBundle[0].([]interface{}); ok {
									for i := range snapBundle {
										data := snapBundle[i].([]interface{})
										offer := WsFundingOffer{
											ID:         int64(data[0].(float64)),
											Symbol:     data[1].(string),
											Created:    int64(data[2].(float64)),
											Updated:    int64(data[3].(float64)),
											Amount:     data[4].(float64),
											AmountOrig: data[5].(float64),
											Type:       data[6].(string),
											Flags:      data[9].(float64),
											Status:     data[10].(string),
											Rate:       data[14].(float64),
											Period:     int64(data[15].(float64)),
											Notify:     data[16].(float64) == 1,
											Hidden:     data[17].(float64) == 1,
											Insure:     data[18].(float64) == 1,
											Renew:      data[19].(float64) == 1,
											RateReal:   data[20].(float64),
										}
										snapshot = append(snapshot, offer)
									}
									b.Websocket.DataHandler <- snapshot
								}
							}
						case fundingOrderNew, fundingOrderUpdate, fundingOrderCancel:
							if data, ok := chanData[2].([]interface{}); ok && len(data) > 0 {
								b.Websocket.DataHandler <- WsFundingOffer{
									ID:         int64(data[0].(float64)),
									Symbol:     data[1].(string),
									Created:    int64(data[2].(float64)),
									Updated:    int64(data[3].(float64)),
									Amount:     data[4].(float64),
									AmountOrig: data[5].(float64),
									Type:       data[6].(string),
									Flags:      data[9].(float64),
									Status:     data[10].(string),
									Rate:       data[14].(float64),
									Period:     int64(data[15].(float64)),
									Notify:     data[16].(float64) == 1,
									Hidden:     data[17].(float64) == 1,
									Insure:     data[18].(float64) == 1,
									Renew:      data[19].(float64) == 1,
									RateReal:   data[20].(float64),
								}
							}
						case fundingCreditSnapshot:
							var snapshot []WsCredit
							if snapBundle, ok := chanData[2].([]interface{}); ok && len(snapBundle) > 0 {
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
						case fundingCreditNew, fundingCreditUpdate, fundingCreditCancel:
							if data, ok := chanData[2].([]interface{}); ok && len(data) > 0 {
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
						case fundingLoanSnapshot:
							var snapshot []WsCredit
							if snapBundle, ok := chanData[2].([]interface{}); ok && len(snapBundle) > 0 {
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
						case fundingLoanNew, fundingLoanUpdate, fundingLoanCancel:
							if data, ok := chanData[2].([]interface{}); ok && len(data) > 0 {
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
						case websocketWalletSnapshot:
							var snapshot []WsWallet
							if snapBundle, ok := chanData[2].([]interface{}); ok && len(snapBundle) > 0 {
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
						case websocketWalletUpdate:
							if data, ok := chanData[2].([]interface{}); ok && len(data) > 0 {
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
						case balanceUpdate:
							if data, ok := chanData[2].([]interface{}); ok && len(data) > 0 {
								b.Websocket.DataHandler <- WsBalanceInfo{
									TotalAssetsUnderManagement: data[0].(float64),
									NetAssetsUnderManagement:   data[1].(float64),
								}
							}
						case marginInfoUpdate:
							if data, ok := chanData[2].([]interface{}); ok && len(data) > 0 {
								if data[0].(string) == "base" {
									if infoBase, ok := chanData[2].([]interface{}); ok && len(infoBase) > 0 {
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
						case fundingInfoUpdate:
							if data, ok := chanData[2].([]interface{}); ok && len(data) > 0 {
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
						case fundingTradeExecuted, fundingTradeUpdate:
							if data, ok := chanData[2].([]interface{}); ok && len(data) > 0 {
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
						}
					}
					switch chanInfo.Channel {
					case "book":
						var newOrderbook []WebsocketBook
						curr := currency.NewPairFromString(chanInfo.Pair)
						if obSnapBundle, ok := chanData[1].([]interface{}); ok {
							if _, ok := obSnapBundle[0].([]interface{}); ok {
								for i := range obSnapBundle {
									obSnap := obSnapBundle[i].([]interface{})
									newOrderbook = append(newOrderbook, WebsocketBook{
										ID:     int(obSnap[0].(float64)),
										Price:  obSnap[1].(float64),
										Amount: obSnap[2].(float64)})
								}
								err := b.WsInsertSnapshot(curr,
									asset.Spot,
									newOrderbook)
								if err != nil {
									b.Websocket.DataHandler <- fmt.Errorf("bitfinex_websocket.go inserting snapshot error: %s",
										err)
								}
							} else if _, ok := obSnapBundle[0].(float64); ok {
								newOrderbook = append(newOrderbook, WebsocketBook{
									ID:     int(obSnapBundle[0].(float64)),
									Price:  obSnapBundle[1].(float64),
									Amount: obSnapBundle[2].(float64)})
								err := b.WsUpdateOrderbook(curr,
									asset.Spot,
									newOrderbook)
								if err != nil {
									b.Websocket.DataHandler <- fmt.Errorf("bitfinex_websocket.go inserting snapshot error: %s",
										err)
								}
							}
						}
					case "candles":
						curr := currency.NewPairFromString(chanInfo.Pair)
						if candleBundle, ok := chanData[1].([]interface{}); ok {
							if len(candleBundle) == 0 {
								continue
							}
							if _, ok := candleBundle[0].([]interface{}); ok {
								for i := range candleBundle {
									candle := candleBundle[i].([]interface{})
									b.Websocket.DataHandler <- wshandler.KlineData{
										Timestamp:  time.Unix(0, candle[0].(int64)),
										Exchange:   b.Name,
										AssetType:  asset.Spot,
										Pair:       curr,
										OpenPrice:  candle[1].(float64),
										ClosePrice: candle[2].(float64),
										HighPrice:  candle[3].(float64),
										LowPrice:   candle[4].(float64),
										Volume:     candle[5].(float64),
									}
								}

							} else if _, ok := candleBundle[0].(float64); ok {
								b.Websocket.DataHandler <- wshandler.KlineData{
									Timestamp:  time.Unix(0, candleBundle[0].(int64)),
									Exchange:   b.Name,
									AssetType:  asset.Spot,
									Pair:       curr,
									OpenPrice:  candleBundle[1].(float64),
									ClosePrice: candleBundle[2].(float64),
									HighPrice:  candleBundle[3].(float64),
									LowPrice:   candleBundle[4].(float64),
									Volume:     candleBundle[5].(float64),
								}
							}
						}
					case "ticker":
						tickerData := chanData[1].([]interface{})
						b.Websocket.DataHandler <- wshandler.TickerData{
							Exchange:  b.Name,
							Bid:       tickerData[0].(float64),
							Ask:       tickerData[2].(float64),
							Last:      tickerData[6].(float64),
							Volume:    tickerData[7].(float64),
							High:      tickerData[8].(float64),
							Low:       tickerData[9].(float64),
							AssetType: asset.Spot,
							Pair:      currency.NewPairFromString(chanInfo.Pair),
						}
					case "trades":
						var trades []WebsocketTrade
						switch len(chanData) {
						case 2:
							data := chanData[1].([]interface{})
							for i := range data {
								y := data[i].([]interface{})
								if _, ok := y[0].(string); ok {
									continue
								}
								trades = append(trades,
									WebsocketTrade{
										ID:        int64(y[0].(float64)),
										Timestamp: int64(y[1].(float64)),
										Price:     y[3].(float64),
										Amount:    y[2].(float64)})
							}
						case 3:
							if chanData[1].(string) == websocketTradeExecuted {
								// the te update contains less data then the "tu"
								continue
							}
							data := chanData[2].([]interface{})
							trades = append(trades, WebsocketTrade{
								ID:        int64(data[0].(float64)),
								Timestamp: int64(data[1].(float64)),
								Price:     data[3].(float64),
								Amount:    data[2].(float64)})
						}
						if len(trades) > 0 {
							for i := range trades {
								side := "BUY"
								newAmount := trades[i].Amount
								if newAmount < 0 {
									side = "SELL"
									newAmount *= -1
								}
								b.Websocket.DataHandler <- wshandler.TradeData{
									CurrencyPair: currency.NewPairFromString(chanInfo.Pair),
									Timestamp:    time.Unix(trades[i].Timestamp, 0),
									Price:        trades[i].Price,
									Amount:       newAmount,
									Exchange:     b.Name,
									AssetType:    asset.Spot,
									Side:         side,
								}
							}
						}
					}
				}
			}
		}
	}
}

// WsInsertSnapshot add the initial orderbook snapshot when subscribed to a
// channel
func (b *Bitfinex) WsInsertSnapshot(p currency.Pair, assetType asset.Item, books []WebsocketBook) error {
	if len(books) == 0 {
		return errors.New("bitfinex.go error - no orderbooks submitted")
	}
	var bid, ask []orderbook.Item
	for i := range books {
		if books[i].Amount >= 0 {
			bid = append(bid, orderbook.Item{Amount: books[i].Amount, Price: books[i].Price})
		} else {
			ask = append(ask, orderbook.Item{Amount: books[i].Amount * -1, Price: books[i].Price})
		}
	}
	if len(bid) == 0 && len(ask) == 0 {
		return errors.New("bitfinex.go error - no orderbooks in item lists")
	}
	var newOrderBook orderbook.Base
	newOrderBook.Asks = ask
	newOrderBook.AssetType = assetType
	newOrderBook.Bids = bid
	newOrderBook.Pair = p
	newOrderBook.ExchangeName = b.Name

	err := b.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
	if err != nil {
		return fmt.Errorf("bitfinex.go error - %s", err)
	}
	b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: p,
		Asset:    assetType,
		Exchange: b.Name}
	return nil
}

// WsUpdateOrderbook updates the orderbook list, removing and adding to the
// orderbook sides
func (b *Bitfinex) WsUpdateOrderbook(p currency.Pair, assetType asset.Item, book []WebsocketBook) error {
	orderbookUpdate := wsorderbook.WebsocketOrderbookUpdate{
		Asks:  []orderbook.Item{},
		Bids:  []orderbook.Item{},
		Asset: assetType,
		Pair:  p,
	}

	for i := 0; i < len(book); i++ {
		switch {
		case book[i].Price > 0:
			if book[i].Amount > 0 {
				// update bid
				orderbookUpdate.Bids = append(orderbookUpdate.Bids, orderbook.Item{Amount: book[i].Amount, Price: book[i].Price})
			} else if book[i].Amount < 0 {
				// update ask
				orderbookUpdate.Asks = append(orderbookUpdate.Asks, orderbook.Item{Amount: book[i].Amount * -1, Price: book[i].Price})
			}
		case book[i].Price == 0:
			if book[i].Amount == 1 {
				// delete bid
				orderbookUpdate.Bids = append(orderbookUpdate.Bids, orderbook.Item{Amount: 0, Price: book[i].Price})
			} else if book[i].Amount == -1 {
				// delete ask
				orderbookUpdate.Asks = append(orderbookUpdate.Asks, orderbook.Item{Amount: 0, Price: book[i].Price})
			}
		}
	}
	err := b.Websocket.Orderbook.Update(&orderbookUpdate)
	if err != nil {
		return err
	}

	b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: p,
		Asset:    assetType,
		Exchange: b.Name}

	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *Bitfinex) GenerateDefaultSubscriptions() {
	var channels = []string{
		"book",
		"trades",
		"ticker",
		"candles",
	}
	var subscriptions []wshandler.WebsocketChannelSubscription
	for i := range channels {
		enabledPairs := b.GetEnabledPairs(asset.Spot)
		for j := range enabledPairs {
			if strings.HasPrefix(enabledPairs[j].Base.String(), "f") {
				log.Warnf(log.WebsocketMgr,
					"%v - Websocket does not support funding currency %v, skipping",
					b.Name, enabledPairs[j])
				continue
			}
			b.appendOptionalDelimiter(&enabledPairs[j])
			params := make(map[string]interface{})
			if channels[i] == "book" {
				params["prec"] = "R0"
				params["len"] = "100"
			}

			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledPairs[j],
				Params:   params,
			})
		}
	}
	b.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (b *Bitfinex) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	req := make(map[string]interface{})
	req["event"] = "subscribe"
	req["channel"] = channelToSubscribe.Channel
	if channelToSubscribe.Currency.String() != "" {
		if channelToSubscribe.Channel == "candles" {
			req["key"] = fmt.Sprintf("trade:1D:%v",
				b.FormatExchangeCurrency(channelToSubscribe.Currency, asset.Spot).String())
		} else {
			req["symbol"] = b.FormatExchangeCurrency(channelToSubscribe.Currency,
				asset.Spot).String()
		}
	}
	if len(channelToSubscribe.Params) > 0 {
		for k, v := range channelToSubscribe.Params {
			req[k] = v
		}
	}
	return b.WebsocketConn.SendMessage(req)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bitfinex) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	req := make(map[string]interface{})
	req["event"] = "unsubscribe"
	req["channel"] = channelToSubscribe.Channel

	if len(channelToSubscribe.Params) > 0 {
		for k, v := range channelToSubscribe.Params {
			req[k] = v
		}
	}
	return b.WebsocketConn.SendMessage(req)
}

// WsPingHandler sends a ping request to the websocket server
func (b *Bitfinex) WsPingHandler() error {
	req := make(map[string]string)
	req["event"] = "ping"
	return b.WebsocketConn.SendMessage(req)
}

// WsSendAuth sends a autheticated event payload
func (b *Bitfinex) WsSendAuth() error {
	if !b.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", b.Name)
	}
	nonce := fmt.Sprintf("%v", time.Now().Unix())
	payload := "AUTH" + nonce
	request := WsAuthRequest{
		Event:       "auth",
		APIKey:      b.API.Credentials.Key,
		AuthPayload: payload,
		AuthSig: crypto.HexEncodeToString(
			crypto.GetHMAC(
				crypto.HashSHA512_384,
				[]byte(payload),
				[]byte(b.API.Credentials.Secret))),
		AuthNonce:     nonce,
		DeadManSwitch: 0,
	}
	err := b.AuthenticatedWebsocketConn.SendMessage(request)
	if err != nil {
		b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// WsSendUnauth sends an unauthenticated payload
func (b *Bitfinex) WsSendUnauth() error {
	req := make(map[string]string)
	req["event"] = "unauth"

	return b.WebsocketConn.SendMessage(req)
}

// WsAddSubscriptionChannel adds a new subscription channel to the
// WebsocketSubdChannels map in bitfinex.go (Bitfinex struct)
func (b *Bitfinex) WsAddSubscriptionChannel(chanID int, channel, pair string) {
	chanInfo := WebsocketChanInfo{Pair: pair, Channel: channel}
	b.WebsocketSubdChannels[chanID] = chanInfo

	if b.Verbose {
		log.Debugf(log.ExchangeSys, "%s Subscribed to Channel: %s Pair: %s ChannelID: %d\n",
			b.Name,
			channel,
			pair,
			chanID)
	}
}

// WsNewOrder authenticated new order request
func (b *Bitfinex) WsNewOrder(data *WsNewOrderRequest) (string, error) {
	data.CustomID = b.AuthenticatedWebsocketConn.GenerateMessageID(false)
	request := makeRequestInterface(orderNew, data)
	resp, err := b.AuthenticatedWebsocketConn.SendMessageReturnResponse(data.CustomID, request)
	if resp == nil {
		return "", errors.New(b.Name + " - Order message not returned")
	}
	if err != nil {
		return "", err
	}
	var respData []interface{}
	err = common.JSONDecode(resp, &respData)
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

	if strings.EqualFold(errCode, "ERROR") {
		return orderID, errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}

	return orderID, nil
}

// WsModifyOrder authenticated modify order request
func (b *Bitfinex) WsModifyOrder(data *WsUpdateOrderRequest) error {
	request := makeRequestInterface(orderUpdate, data)
	resp, err := b.AuthenticatedWebsocketConn.SendMessageReturnResponse(data.OrderID, request)
	if resp == nil {
		return errors.New(b.Name + " - Order message not returned")
	}
	if err != nil {
		return err
	}
	var responseData []interface{}
	err = common.JSONDecode(resp, &responseData)
	if err != nil {
		return err
	}
	responseOrderData := responseData[2].([]interface{})
	errCode := responseOrderData[6].(string)
	errorMessage := responseOrderData[7].(string)
	if strings.EqualFold(errCode, "ERROR") {
		return errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}

	return nil
}

// WsCancelMultiOrders authenticated cancel multi order request
func (b *Bitfinex) WsCancelMultiOrders(orderIDs []int64) error {
	cancel := WsCancelGroupOrdersRequest{
		OrderID: orderIDs,
	}
	request := makeRequestInterface(cancelMultipleOrders, cancel)
	return b.AuthenticatedWebsocketConn.SendMessage(request)
}

// WsCancelOrder authenticated cancel order request
func (b *Bitfinex) WsCancelOrder(orderID int64) error {
	cancel := WsCancelOrderRequest{
		OrderID: orderID,
	}
	request := makeRequestInterface(orderCancel, cancel)
	resp, err := b.AuthenticatedWebsocketConn.SendMessageReturnResponse(orderID, request)
	if resp == nil {
		return fmt.Errorf("%v - Order %v failed to cancel", b.Name, orderID)
	}
	if err != nil {
		return err
	}
	var responseData []interface{}
	err = common.JSONDecode(resp, &responseData)
	if err != nil {
		return err
	}
	responseOrderData := responseData[2].([]interface{})
	errCode := responseOrderData[6].(string)
	errorMessage := responseOrderData[7].(string)
	if strings.EqualFold(errCode, "ERROR") {
		return errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}

	return nil
}

// WsCancelAllOrders authenticated cancel all orders request
func (b *Bitfinex) WsCancelAllOrders() error {
	cancelAll := WsCancelAllOrdersRequest{All: 1}
	request := makeRequestInterface(cancelMultipleOrders, cancelAll)
	return b.AuthenticatedWebsocketConn.SendMessage(request)
}

// WsNewOffer authenticated new offer request
func (b *Bitfinex) WsNewOffer(data *WsNewOfferRequest) error {
	request := makeRequestInterface(fundingOrderNew, data)
	return b.AuthenticatedWebsocketConn.SendMessage(request)
}

// WsCancelOffer authenticated cancel offer request
func (b *Bitfinex) WsCancelOffer(orderID int64) error {
	cancel := WsCancelOrderRequest{
		OrderID: orderID,
	}
	request := makeRequestInterface(fundingOrderCancel, cancel)
	resp, err := b.AuthenticatedWebsocketConn.SendMessageReturnResponse(orderID, request)
	if resp == nil {
		return fmt.Errorf("%v - Order %v failed to cancel", b.Name, orderID)
	}
	if err != nil {
		return err
	}
	var responseData []interface{}
	err = common.JSONDecode(resp, &responseData)
	if err != nil {
		return err
	}
	responseOrderData := responseData[2].([]interface{})
	errCode := responseOrderData[6].(string)
	var errorMessage string
	if responseOrderData[7] != nil {
		errorMessage = responseOrderData[7].(string)
	}
	if strings.EqualFold(errCode, "ERROR") {
		return errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}

	return nil
}

func makeRequestInterface(channelName string, data interface{}) []interface{} {
	var namelessType []interface{}
	namelessType = append(namelessType, 0)
	namelessType = append(namelessType, channelName)
	namelessType = append(namelessType, nil)
	namelessType = append(namelessType, data)

	return namelessType
}
