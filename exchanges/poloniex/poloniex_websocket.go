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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	poloniexWebsocketAddress = "wss://api2.poloniex.com"
	wsAccountNotificationID  = 1000
	wsTickerDataID           = 1002
	ws24HourExchangeVolumeID = 1003
	wsHeartbeat              = 1010
)

var (
	// currencyIDMap stores a map of currencies associated with their ID
	currencyIDMap map[float64]string
)

// WsConnect initiates a websocket connection
func (p *Poloniex) WsConnect() error {
	if !p.Websocket.IsEnabled() || !p.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := p.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	err = p.getCurrencyIDMap()
	if err != nil {
		return err
	}

	go p.wsReadData()
	subs, err := p.GenerateDefaultSubscriptions()
	if err != nil {
		return err
	}

	return p.Websocket.SubscribeToChannels(subs)
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
	defer p.Websocket.Wg.Done()

	for {
		resp := p.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := p.wsHandleData(resp.Raw)
		if err != nil {
			p.Websocket.DataHandler <- err
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
										currPair, err = currency.NewPairFromString(currPairFromMap)
										if err != nil {
											return err
										}
									} else {
										// It is better to still log an order which you can recheck later, rather than error out
										p.Websocket.DataHandler <- fmt.Errorf(p.Name+
											" - Unknown currency pair ID. "+
											"Currency will appear as the pair ID: '%v'",
											notification[1].(float64))
										currPair, err = currency.NewPairFromString(strconv.FormatFloat(notification[1].(float64), 'f', -1, 64))
										if err != nil {
											return err
										}
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
					case "o":
						currencyPair := currencyIDMap[channelID]
						dataL3 := dataL2.([]interface{})
						err = p.WsProcessOrderbookUpdate(int64(data[1].(float64)),
							dataL3,
							currencyPair)
						if err != nil {
							return err
						}
					case "t":
						if !p.IsSaveTradeDataEnabled() {
							return nil
						}
						currencyPair := currencyIDMap[channelID]
						var t WsTrade
						t.Symbol = currencyIDMap[channelID]
						dataL3, ok := dataL2.([]interface{})
						if !ok {
							return errors.New("websocket trade update error: type conversion failure")
						}

						if len(dataL3) != 6 {
							return errors.New("websocket trade update error: incorrect data returned")
						}

						// tradeID type intermittently changes
						switch tradeIDData := dataL3[1].(type) {
						case string:
							t.TradeID, err = strconv.ParseInt(tradeIDData, 10, 64)
							if err != nil {
								return err
							}
						case float64:
							t.TradeID = int64(tradeIDData)
						default:
							return fmt.Errorf("unhandled type for websocket trade update: %v", t)
						}

						side := order.Buy
						if dataL3[2].(float64) != 1 {
							side = order.Sell
						}
						t.Volume, err = strconv.ParseFloat(dataL3[3].(string), 64)
						if err != nil {
							return err
						}
						t.Price, err = strconv.ParseFloat(dataL3[4].(string), 64)
						if err != nil {
							return err
						}
						t.Timestamp = int64(dataL3[5].(float64))

						pair, err := currency.NewPairFromString(currencyPair)
						if err != nil {
							return err
						}

						return p.AddTradesToBuffer(trade.Data{
							TID:          strconv.FormatInt(t.TradeID, 10),
							Exchange:     p.Name,
							CurrencyPair: pair,
							AssetType:    asset.Spot,
							Side:         side,
							Price:        t.Price,
							Amount:       t.Volume,
							Timestamp:    time.Unix(t.Timestamp, 0),
						})
					default:
						p.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: p.Name + stream.UnhandledMessage + string(respRaw)}
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
	currencyPair, err := currency.NewPairDelimiter(currencyIDMap[tickerData[0].(float64)],
		currency.UnderscoreDelimiter)
	if err != nil {
		return err
	}

	enabled, err := p.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	if !enabled.Contains(currencyPair, true) {
		var avail currency.Pairs
		avail, err = p.GetAvailablePairs(asset.Spot)
		if err != nil {
			return err
		}

		if !avail.Contains(currencyPair, true) {
			return fmt.Errorf("currency pair %s not found in available pair list",
				currencyPair)
		}
		return nil
	}

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

	var err error
	newOrderBook.Pair, err = currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	newOrderBook.ExchangeName = p.Name

	return p.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// WsProcessOrderbookUpdate processes new orderbook updates
func (p *Poloniex) WsProcessOrderbookUpdate(sequenceNumber int64, target []interface{}, symbol string) error {
	cP, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	price, err := strconv.ParseFloat(target[2].(string), 64)
	if err != nil {
		return err
	}
	volume, err := strconv.ParseFloat(target[3].(string), 64)
	if err != nil {
		return err
	}
	update := &buffer.Update{
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
func (p *Poloniex) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	subscriptions = append(subscriptions, stream.ChannelSubscription{
		Channel: strconv.FormatInt(wsTickerDataID, 10),
	})

	if p.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		subscriptions = append(subscriptions, stream.ChannelSubscription{
			Channel: strconv.FormatInt(wsAccountNotificationID, 10),
		})
	}

	enabledCurrencies, err := p.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for j := range enabledCurrencies {
		enabledCurrencies[j].Delimiter = currency.UnderscoreDelimiter
		subscriptions = append(subscriptions, stream.ChannelSubscription{
			Channel:  "orderbook",
			Currency: enabledCurrencies[j],
			Asset:    asset.Spot,
		})
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (p *Poloniex) Subscribe(sub []stream.ChannelSubscription) error {
	var errs common.Errors
channels:
	for i := range sub {
		subscriptionRequest := WsCommand{
			Command: "subscribe",
		}
		switch {
		case strings.EqualFold(strconv.FormatInt(wsAccountNotificationID, 10),
			sub[i].Channel):
			err := p.wsSendAuthorisedCommand("subscribe")
			if err != nil {
				errs = append(errs, err)
				continue channels
			}
			p.Websocket.AddSuccessfulSubscriptions(sub[i])
			continue channels
		case strings.EqualFold(strconv.FormatInt(wsTickerDataID, 10),
			sub[i].Channel):
			subscriptionRequest.Channel = wsTickerDataID
		default:
			subscriptionRequest.Channel = sub[i].Currency.String()
		}

		err := p.Websocket.Conn.SendJSONMessage(subscriptionRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		p.Websocket.AddSuccessfulSubscriptions(sub[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (p *Poloniex) Unsubscribe(unsub []stream.ChannelSubscription) error {
	var errs common.Errors
channels:
	for i := range unsub {
		unsubscriptionRequest := WsCommand{
			Command: "unsubscribe",
		}
		switch {
		case strings.EqualFold(strconv.FormatInt(wsAccountNotificationID, 10),
			unsub[i].Channel):
			err := p.wsSendAuthorisedCommand("unsubscribe")
			if err != nil {
				errs = append(errs, err)
				continue channels
			}
			p.Websocket.RemoveSuccessfulUnsubscriptions(unsub[i])
			continue channels
		case strings.EqualFold(strconv.FormatInt(wsTickerDataID, 10),
			unsub[i].Channel):
			unsubscriptionRequest.Channel = wsTickerDataID
		default:
			unsubscriptionRequest.Channel = unsub[i].Currency.String()
		}
		err := p.Websocket.Conn.SendJSONMessage(unsubscriptionRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		p.Websocket.RemoveSuccessfulUnsubscriptions(unsub[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (p *Poloniex) wsSendAuthorisedCommand(command string) error {
	nonce := fmt.Sprintf("nonce=%v", time.Now().UnixNano())
	hmac := crypto.GetHMAC(crypto.HashSHA512,
		[]byte(nonce),
		[]byte(p.API.Credentials.Secret))
	request := WsAuthorisationRequest{
		Command: command,
		Channel: 1000,
		Sign:    crypto.HexEncodeToString(hmac),
		Key:     p.API.Credentials.Key,
		Payload: nonce,
	}
	return p.Websocket.Conn.SendJSONMessage(request)
}
