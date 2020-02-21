package poloniex

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
)

const (
	poloniexWebsocketAddress = "wss://api2.poloniex.com"
	wsAccountNotificationID  = 1000
	wsTickerDataID           = 1002
	ws24HourExchangeVolumeID = 1003
	wsHeartbeat              = 1010
	delimiterUnderscore      = "_"
)

var (
	// currencyIDMap stores a map of currencies associated with their ID
	currencyIDMap map[float64]string
)

// WsConnect initiates a websocket connection
func (p *Poloniex) WsConnect() error {
	if !p.Websocket.IsEnabled() || !p.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := p.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	err2 := p.getCurrencyIDMap()
	if err2 != nil {
		return err2
	}

	go p.wsReadData()
	p.GenerateDefaultSubscriptions()

	return nil
}

func (p *Poloniex) getCurrencyIDMap() error {
	if currencyIDMap == nil {
		currencyIDMap = make(map[float64]string)
		resp, err := p.GetTicker()
		if err != nil {
			return err
		}

		for k, v := range resp {
			currencyIDMap[v.ID] = k
		}
	}
	return nil
}

func getWSDataType(data interface{}) string {
	subData := data.([]interface{})
	dataType := subData[0].(string)
	return dataType
}

func checkSubscriptionSuccess(data []interface{}) bool {
	return data[1].(float64) == 1
}

// wsReadData handles data from the websocket connection
func (p *Poloniex) wsReadData() {
	p.Websocket.Wg.Add(1)

	defer func() {
		p.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-p.Websocket.ShutdownC:
			return
		default:
			resp, err := p.WebsocketConn.ReadMessage()
			if err != nil {
				p.Websocket.ReadMessageErrors <- err
				return
			}
			p.Websocket.TrafficAlert <- struct{}{}
			err = p.wsHandleData(resp.Raw)
			if err != nil {
				p.Websocket.DataHandler <- err
			}
		}
	}
}

func (p *Poloniex) wsHandleData(respRaw []byte) error {
	var result interface{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	if data, ok := result.([]interface{}); ok {
		if len(data) == 0 {
			return nil
		}
		if len(data) == 1 {
			// heartbeat
			return nil
		}
		if len(data) == 2 {
			// subscription acknowledgement / heartbeat
			return nil
		}
		if channelID, ok := data[0].(float64); ok {
			switch channelID {
			case ws24HourExchangeVolumeID:
				return nil
			case wsAccountNotificationID:
				if notificationsArray, ok := data[2].([]interface{}); ok {
					if _, ok := notificationsArray[0].([]interface{}); ok {
						for i := 0; i < len(notificationsArray); i++ {
							if notification, ok := (notificationsArray[i]).([]interface{}); ok {
								switch notification[0].(string) {
								case "o":
									var amount float64
									amount, err = strconv.ParseFloat(notification[2].(string), 64)
									if err != nil {
										return err
									}
									var oStatus order.Status
									var oType = notification[3].(string)
									switch {
									case amount > 0 && (oType == "f" || oType == "s"):
										oStatus = order.PartiallyFilled
									case amount == 0 && (oType == "f" || oType == "s"):
										oStatus = order.Filled
									case amount > 0 && oType == "c":
										oStatus = order.PartiallyCancelled
									case amount == 0 && oType == "c":
										oStatus = order.Cancelled
									}
									response := &order.Modify{
										RemainingAmount: amount,
										Exchange:        p.Name,
										ID:              strconv.FormatFloat(notification[1].(float64), 'f', -1, 64),
										Type:            order.Limit,
										Status:          oStatus,
										AssetType:       asset.Spot,
									}
									p.Websocket.DataHandler <- response
								case "n":
									var timeParse time.Time
									timeParse, err = time.Parse(common.SimpleTimeFormat, notification[6].(string))
									if err != nil {
										return err
									}
									var rate, amount float64
									rate, err = strconv.ParseFloat(notification[4].(string), 64)
									if err != nil {
										return err
									}
									amount, err = strconv.ParseFloat(notification[5].(string), 64)
									if err != nil {
										return err
									}
									var buySell order.Side
									switch notification[2].(float64) {
									case 0:
										buySell = order.Buy
									case 1:
										buySell = order.Sell
									}
									var currPair currency.Pair
									if currPairFromMap, ok := currencyIDMap[notification[1].(float64)]; ok {
										currPair = currency.NewPairFromString(currPairFromMap)
									} else {
										// It is better to still log an order which you can recheck later, rather than error out
										p.Websocket.DataHandler <- fmt.Errorf(p.Name+
											" - Unknown currency pair ID. "+
											"Currency will appear as the pair ID: '%v'",
											notification[1].(float64))
										currPair = currency.NewPairFromString(strconv.FormatFloat(notification[1].(float64), 'f', -1, 64))
									}
									var a asset.Item
									a, err = p.GetPairAssetType(currPair)
									if err != nil {
										return err
									}
									response := &order.Detail{
										Price:     rate,
										Amount:    amount,
										Exchange:  p.Name,
										ID:        strconv.FormatFloat(notification[2].(float64), 'f', -1, 64),
										Type:      order.Limit,
										Side:      buySell,
										Status:    order.New,
										AssetType: a,
										Date:      timeParse,
										Pair:      currPair,
									}
									p.Websocket.DataHandler <- response
								case "b":
									var amount float64
									amount, err = strconv.ParseFloat(notification[3].(string), 64)
									if err != nil {
										return err
									}

									response := WsAccountBalanceUpdateResponse{
										currencyID: notification[1].(float64),
										wallet:     notification[2].(string),
										amount:     amount,
									}
									p.Websocket.DataHandler <- response
								case "t":
									var timeParse time.Time
									timeParse, err = time.Parse(common.SimpleTimeFormat, notification[8].(string))
									if err != nil {
										return err
									}
									var rate, amount, totalFee float64
									rate, err = strconv.ParseFloat(notification[2].(string), 64)
									if err != nil {
										return err
									}
									amount, err = strconv.ParseFloat(notification[3].(string), 64)
									if err != nil {
										return err
									}
									totalFee, err = strconv.ParseFloat(notification[7].(string), 64)
									if err != nil {
										return err
									}
									var trades []order.TradeHistory
									trades = append(trades, order.TradeHistory{
										Price:     rate,
										Amount:    amount,
										Fee:       totalFee,
										Exchange:  p.Name,
										TID:       strconv.FormatFloat(notification[1].(float64), 'f', -1, 64),
										Timestamp: timeParse,
									})
									response := &order.Modify{
										ID:     strconv.FormatFloat(notification[6].(float64), 'f', -1, 64),
										Fee:    totalFee,
										Trades: trades,
									}
									p.Websocket.DataHandler <- response
								case "k":
									response := &order.Modify{
										Exchange: p.Name,
										ID:       strconv.FormatFloat(notification[1].(float64), 'f', -1, 64),
										Status:   order.Cancelled,
									}
									p.Websocket.DataHandler <- response
								}
							}
						}
					}
				}
			case wsTickerDataID:
				return p.wsHandleTickerData(data)
			default:
				subData := data[2].([]interface{})
				for x := range subData {
					dataL2 := subData[x]

					switch getWSDataType(dataL2) {
					case "i":
						dataL3 := dataL2.([]interface{})
						dataL3map := dataL3[1].(map[string]interface{})
						currencyPair, ok := dataL3map["currencyPair"].(string)
						if !ok {
							return fmt.Errorf("%s websocket could not find currency pair in map",
								p.Name)
						}

						orderbookData, ok := dataL3map["orderBook"].([]interface{})
						if !ok {
							return fmt.Errorf("%s websocket could not find orderbook data in map",
								p.Name)
						}

						err = p.WsProcessOrderbookSnapshot(orderbookData,
							currencyPair)
						if err != nil {
							return err
						}

						p.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
							Exchange: p.Name,
							Asset:    asset.Spot,
							Pair:     currency.NewPairFromString(currencyPair),
						}
					case "o":
						currencyPair := currencyIDMap[channelID]
						dataL3 := dataL2.([]interface{})
						err = p.WsProcessOrderbookUpdate(int64(data[1].(float64)),
							dataL3,
							currencyPair)
						if err != nil {
							return err
						}

						p.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
							Exchange: p.Name,
							Asset:    asset.Spot,
							Pair:     currency.NewPairFromString(currencyPair),
						}
					case "t":
						currencyPair := currencyIDMap[channelID]
						var trade WsTrade
						trade.Symbol = currencyIDMap[channelID]
						dataL3 := dataL2.([]interface{})
						trade.TradeID, err = strconv.ParseInt(dataL3[1].(string), 10, 64)
						if err != nil {
							return err
						}
						side := order.Buy
						if dataL3[2].(float64) != 1 {
							side = order.Sell
						}
						trade.Volume, err = strconv.ParseFloat(dataL3[3].(string), 64)
						if err != nil {
							return err
						}
						trade.Price, err = strconv.ParseFloat(dataL3[4].(string), 64)
						if err != nil {
							return err
						}
						trade.Timestamp = int64(dataL3[5].(float64))

						p.Websocket.DataHandler <- wshandler.TradeData{
							Timestamp:    time.Unix(trade.Timestamp, 0),
							CurrencyPair: currency.NewPairFromString(currencyPair),
							Side:         side,
							Amount:       trade.Volume,
							Price:        trade.Price,
						}
					default:
						p.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: p.Name + wshandler.UnhandledMessage + string(respRaw)}
						return nil
					}
				}
			}
		}
	}
	return nil
}

func (p *Poloniex) wsHandleTickerData(data []interface{}) error {
	tickerData := data[2].([]interface{})
	var t WsTicker
	currencyPair := currency.NewPairDelimiter(currencyIDMap[tickerData[0].(float64)], delimiterUnderscore)
	if !p.GetEnabledPairs(asset.Spot).Contains(currencyPair, true) {
		// Ticker subscription receives all currencies, no specific subscription
		// There should be no error associated with receiving data of disabled currency ticker data
		return nil
	}

	var err error
	t.LastPrice, err = strconv.ParseFloat(tickerData[1].(string), 64)
	if err != nil {
		return err
	}

	t.LowestAsk, err = strconv.ParseFloat(tickerData[2].(string), 64)
	if err != nil {
		return err
	}

	t.HighestBid, err = strconv.ParseFloat(tickerData[3].(string), 64)
	if err != nil {
		return err
	}

	t.PercentageChange, err = strconv.ParseFloat(tickerData[4].(string), 64)
	if err != nil {
		return err
	}

	t.BaseCurrencyVolume24H, err = strconv.ParseFloat(tickerData[5].(string), 64)
	if err != nil {
		return err
	}

	t.QuoteCurrencyVolume24H, err = strconv.ParseFloat(tickerData[6].(string), 64)
	if err != nil {
		return err
	}

	t.IsFrozen = tickerData[7].(float64) == 1
	t.HighestTradeIn24H, err = strconv.ParseFloat(tickerData[8].(string), 64)
	if err != nil {
		return err
	}

	t.LowestTradePrice24H, err = strconv.ParseFloat(tickerData[9].(string), 64)
	if err != nil {
		return err
	}

	p.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: p.Name,
		Volume:       t.BaseCurrencyVolume24H,
		QuoteVolume:  t.QuoteCurrencyVolume24H,
		High:         t.HighestBid,
		Low:          t.LowestAsk,
		Bid:          t.HighestBid,
		Ask:          t.LowestAsk,
		Last:         t.LastPrice,
		AssetType:    asset.Spot,
		Pair:         currencyPair,
	}
	return nil
}

// WsProcessOrderbookSnapshot processes a new orderbook snapshot into a local
// of orderbooks
func (p *Poloniex) WsProcessOrderbookSnapshot(ob []interface{}, symbol string) error {
	askdata := ob[0].(map[string]interface{})
	var asks []orderbook.Item
	for price, volume := range askdata {
		assetPrice, err := strconv.ParseFloat(price, 64)
		if err != nil {
			return err
		}

		assetVolume, err := strconv.ParseFloat(volume.(string), 64)
		if err != nil {
			return err
		}

		asks = append(asks, orderbook.Item{
			Price:  assetPrice,
			Amount: assetVolume,
		})
	}

	bidData := ob[1].(map[string]interface{})
	var bids []orderbook.Item
	for price, volume := range bidData {
		assetPrice, err := strconv.ParseFloat(price, 64)
		if err != nil {
			return err
		}

		assetVolume, err := strconv.ParseFloat(volume.(string), 64)
		if err != nil {
			return err
		}

		bids = append(bids, orderbook.Item{
			Price:  assetPrice,
			Amount: assetVolume,
		})
	}

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.AssetType = asset.Spot
	newOrderBook.Pair = currency.NewPairFromString(symbol)
	newOrderBook.ExchangeName = p.Name

	return p.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// WsProcessOrderbookUpdate processes new orderbook updates
func (p *Poloniex) WsProcessOrderbookUpdate(sequenceNumber int64, target []interface{}, symbol string) error {
	cP := currency.NewPairFromString(symbol)
	price, err := strconv.ParseFloat(target[2].(string), 64)
	if err != nil {
		return err
	}
	volume, err := strconv.ParseFloat(target[3].(string), 64)
	if err != nil {
		return err
	}
	update := &wsorderbook.WebsocketOrderbookUpdate{
		Pair:     cP,
		Asset:    asset.Spot,
		UpdateID: sequenceNumber,
	}
	if target[1].(float64) == 1 {
		update.Bids = []orderbook.Item{{Price: price, Amount: volume}}
	} else {
		update.Asks = []orderbook.Item{{Price: price, Amount: volume}}
	}
	return p.Websocket.Orderbook.Update(update)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (p *Poloniex) GenerateDefaultSubscriptions() {
	var subscriptions []wshandler.WebsocketChannelSubscription
	subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
		Channel: strconv.FormatInt(wsTickerDataID, 10),
	})

	if p.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
			Channel: strconv.FormatInt(wsAccountNotificationID, 10),
		})
	}

	enabledCurrencies := p.GetEnabledPairs(asset.Spot)
	for j := range enabledCurrencies {
		enabledCurrencies[j].Delimiter = delimiterUnderscore
		subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
			Channel:  "orderbook",
			Currency: enabledCurrencies[j],
		})
	}
	p.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (p *Poloniex) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	subscriptionRequest := WsCommand{
		Command: "subscribe",
	}
	switch {
	case strings.EqualFold(strconv.FormatInt(wsAccountNotificationID, 10), channelToSubscribe.Channel):
		return p.wsSendAuthorisedCommand("subscribe")
	case strings.EqualFold(strconv.FormatInt(wsTickerDataID, 10), channelToSubscribe.Channel):
		subscriptionRequest.Channel = wsTickerDataID
	default:
		subscriptionRequest.Channel = channelToSubscribe.Currency.String()
	}
	return p.WebsocketConn.SendJSONMessage(subscriptionRequest)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (p *Poloniex) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	unsubscriptionRequest := WsCommand{
		Command: "unsubscribe",
	}
	switch {
	case strings.EqualFold(strconv.FormatInt(wsAccountNotificationID, 10), channelToSubscribe.Channel):
		return p.wsSendAuthorisedCommand("unsubscribe")
	case strings.EqualFold(strconv.FormatInt(wsTickerDataID, 10), channelToSubscribe.Channel):
		unsubscriptionRequest.Channel = wsTickerDataID
	default:
		unsubscriptionRequest.Channel = channelToSubscribe.Currency.String()
	}
	return p.WebsocketConn.SendJSONMessage(unsubscriptionRequest)
}

func (p *Poloniex) wsSendAuthorisedCommand(command string) error {
	nonce := fmt.Sprintf("nonce=%v", time.Now().UnixNano())
	hmac := crypto.GetHMAC(crypto.HashSHA512, []byte(nonce), []byte(p.API.Credentials.Secret))
	request := WsAuthorisationRequest{
		Command: command,
		Channel: 1000,
		Sign:    crypto.HexEncodeToString(hmac),
		Key:     p.API.Credentials.Key,
		Payload: nonce,
	}
	return p.WebsocketConn.SendJSONMessage(request)
}
