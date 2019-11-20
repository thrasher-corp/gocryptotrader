package poloniex

import (
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	log "github.com/thrasher-corp/gocryptotrader/logger"
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
	currencyIDMap map[int]string
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

	if currencyIDMap == nil {
		currencyIDMap = make(map[int]string)
		resp, err := p.GetTicker()
		if err != nil {
			return err
		}

		for k, v := range resp {
			currencyIDMap[v.ID] = k
		}
	}

	go p.WsHandleData()
	p.GenerateDefaultSubscriptions()

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

// WsHandleData handles data from the websocket connection
func (p *Poloniex) WsHandleData() {
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
			var result interface{}
			err = common.JSONDecode(resp.Raw, &result)
			if err != nil {
				p.Websocket.DataHandler <- err
				continue
			}
			switch data := result.(type) {
			case map[string]interface{}:
				// subscription error
				p.Websocket.DataHandler <- errors.New(data["error"].(string))
			case []interface{}:
				chanID := int(data[0].(float64))
				if len(data) == 2 && chanID != wsHeartbeat {
					if checkSubscriptionSuccess(data) {
						if p.Verbose {
							log.Debugf(log.ExchangeSys,
								"%s websocket subscribed to channel successfully. %d",
								p.Name,
								chanID)
						}
					} else {
						p.Websocket.DataHandler <- fmt.Errorf("%s websocket subscription to channel failed. %d",
							p.Name,
							chanID)
					}
					continue
				}

				switch chanID {
				case wsAccountNotificationID:
					p.wsHandleAccountData(data[2].([][]interface{}))
				case wsTickerDataID:
					p.wsHandleTickerData(data)
				case ws24HourExchangeVolumeID:
				case wsHeartbeat:
				default:
					if len(data) > 2 {
						subData := data[2].([]interface{})

						for x := range subData {
							dataL2 := subData[x]
							dataL3 := dataL2.([]interface{})

							switch getWSDataType(dataL2) {
							case "i":
								dataL3map := dataL3[1].(map[string]interface{})
								currencyPair, ok := dataL3map["currencyPair"].(string)
								if !ok {
									p.Websocket.DataHandler <- fmt.Errorf("%s websocket could not find currency pair in map",
										p.Name)
									continue
								}

								orderbookData, ok := dataL3map["orderBook"].([]interface{})
								if !ok {
									p.Websocket.DataHandler <- fmt.Errorf("%s websocket could not find orderbook data in map",
										p.Name)
									continue
								}

								err = p.WsProcessOrderbookSnapshot(orderbookData,
									currencyPair)
								if err != nil {
									p.Websocket.DataHandler <- err
									continue
								}

								p.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
									Exchange: p.Name,
									Asset:    asset.Spot,
									Pair:     currency.NewPairFromString(currencyPair),
								}
							case "o":
								currencyPair := currencyIDMap[chanID]
								err = p.WsProcessOrderbookUpdate(int64(data[1].(float64)),
									dataL3,
									currencyPair)
								if err != nil {
									p.Websocket.DataHandler <- err
									continue
								}

								p.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
									Exchange: p.Name,
									Asset:    asset.Spot,
									Pair:     currency.NewPairFromString(currencyPair),
								}
							case "t":
								currencyPair := currencyIDMap[chanID]
								var trade WsTrade
								trade.Symbol = currencyIDMap[chanID]
								trade.TradeID, _ = strconv.ParseInt(dataL3[1].(string), 10, 64)
								// 1 for buy 0 for sell
								side := "buy"
								if dataL3[2].(float64) != 1 {
									side = "sell"
								}
								trade.Side = side
								trade.Volume, err = strconv.ParseFloat(dataL3[3].(string), 64)
								if err != nil {
									p.Websocket.DataHandler <- err
									continue
								}
								trade.Price, err = strconv.ParseFloat(dataL3[4].(string), 64)
								if err != nil {
									p.Websocket.DataHandler <- err
									continue
								}
								trade.Timestamp = int64(dataL3[5].(float64))

								p.Websocket.DataHandler <- wshandler.TradeData{
									Timestamp:    time.Unix(trade.Timestamp, 0),
									CurrencyPair: currency.NewPairFromString(currencyPair),
									Side:         trade.Side,
									Amount:       trade.Volume,
									Price:        trade.Price,
								}
							}
						}
					}
				}
			}
		}
	}
}

func (p *Poloniex) wsHandleTickerData(data []interface{}) {
	tickerData := data[2].([]interface{})
	var t WsTicker
	currencyPair := currency.NewPairDelimiter(currencyIDMap[int(tickerData[0].(float64))], delimiterUnderscore)
	if !p.GetEnabledPairs(asset.Spot).Contains(currencyPair, true) {
		return
	}

	var err error
	t.LastPrice, err = strconv.ParseFloat(tickerData[1].(string), 64)
	if err != nil {
		p.Websocket.DataHandler <- err
		return
	}

	t.LowestAsk, err = strconv.ParseFloat(tickerData[2].(string), 64)
	if err != nil {
		p.Websocket.DataHandler <- err
		return
	}

	t.HighestBid, err = strconv.ParseFloat(tickerData[3].(string), 64)
	if err != nil {
		p.Websocket.DataHandler <- err
		return
	}

	t.PercentageChange, err = strconv.ParseFloat(tickerData[4].(string), 64)
	if err != nil {
		p.Websocket.DataHandler <- err
		return
	}

	t.BaseCurrencyVolume24H, err = strconv.ParseFloat(tickerData[5].(string), 64)
	if err != nil {
		p.Websocket.DataHandler <- err
		return
	}

	t.QuoteCurrencyVolume24H, err = strconv.ParseFloat(tickerData[6].(string), 64)
	if err != nil {
		p.Websocket.DataHandler <- err
		return
	}

	t.IsFrozen = tickerData[7].(float64) == 1
	t.HighestTradeIn24H, err = strconv.ParseFloat(tickerData[8].(string), 64)
	if err != nil {
		p.Websocket.DataHandler <- err
		return
	}

	t.LowestTradePrice24H, err = strconv.ParseFloat(tickerData[9].(string), 64)
	if err != nil {
		p.Websocket.DataHandler <- err
		return
	}

	p.Websocket.DataHandler <- wshandler.TickerData{
		Exchange:    p.Name,
		Volume:      t.BaseCurrencyVolume24H,
		QuoteVolume: t.QuoteCurrencyVolume24H,
		High:        t.HighestBid,
		Low:         t.LowestAsk,
		Bid:         t.HighestBid,
		Ask:         t.LowestAsk,
		Last:        t.LastPrice,
		Timestamp:   time.Now(),
		AssetType:   asset.Spot,
		Pair:        currencyPair,
	}
}

// wsHandleAccountData Parses account data and sends to datahandler
func (p *Poloniex) wsHandleAccountData(accountData [][]interface{}) {
	for i := range accountData {
		switch accountData[i][0].(string) {
		case "b":
			amount, err := strconv.ParseFloat(accountData[i][3].(string), 64)
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			response := WsAccountBalanceUpdateResponse{
				currencyID: accountData[i][1].(float64),
				wallet:     accountData[i][2].(string),
				amount:     amount,
			}
			p.Websocket.DataHandler <- response
		case "n":
			timeParse, err := time.Parse("2006-01-02 15:04:05", accountData[i][6].(string))
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			rate, err := strconv.ParseFloat(accountData[i][4].(string), 64)
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			amount, err := strconv.ParseFloat(accountData[i][5].(string), 64)
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			response := WsNewLimitOrderResponse{
				currencyID:  accountData[i][1].(float64),
				orderNumber: accountData[i][2].(float64),
				orderType:   accountData[i][3].(float64),
				rate:        rate,
				amount:      amount,
				date:        timeParse,
			}
			p.Websocket.DataHandler <- response
		case "o":
			response := WsOrderUpdateResponse{
				OrderNumber: accountData[i][1].(float64),
				NewAmount:   accountData[i][2].(string),
			}
			p.Websocket.DataHandler <- response
		case "t":
			timeParse, err := time.Parse("2006-01-02 15:04:05", accountData[i][8].(string))
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			rate, err := strconv.ParseFloat(accountData[i][2].(string), 64)
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			amount, err := strconv.ParseFloat(accountData[i][3].(string), 64)
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			feeMultiplier, err := strconv.ParseFloat(accountData[i][4].(string), 64)
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			totalFee, err := strconv.ParseFloat(accountData[i][7].(string), 64)
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			response := WsTradeNotificationResponse{
				TradeID:       accountData[i][1].(float64),
				Rate:          rate,
				Amount:        amount,
				FeeMultiplier: feeMultiplier,
				FundingType:   accountData[i][5].(float64),
				OrderNumber:   accountData[i][6].(float64),
				TotalFee:      totalFee,
				Date:          timeParse,
			}
			p.Websocket.DataHandler <- response
		}
	}
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
		Channel: fmt.Sprintf("%v", wsTickerDataID),
	})

	if p.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
			Channel: fmt.Sprintf("%v", wsAccountNotificationID),
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
	case strings.EqualFold(fmt.Sprintf("%v", wsAccountNotificationID), channelToSubscribe.Channel):
		return p.wsSendAuthorisedCommand("subscribe")
	case strings.EqualFold(fmt.Sprintf("%v", wsTickerDataID), channelToSubscribe.Channel):
		subscriptionRequest.Channel = wsTickerDataID
	default:
		subscriptionRequest.Channel = channelToSubscribe.Currency.String()
	}
	return p.WebsocketConn.SendMessage(subscriptionRequest)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (p *Poloniex) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	unsubscriptionRequest := WsCommand{
		Command: "unsubscribe",
	}
	switch {
	case strings.EqualFold(fmt.Sprintf("%v", wsAccountNotificationID), channelToSubscribe.Channel):
		return p.wsSendAuthorisedCommand("unsubscribe")
	case strings.EqualFold(fmt.Sprintf("%v", wsTickerDataID), channelToSubscribe.Channel):
		unsubscriptionRequest.Channel = wsTickerDataID
	default:
		unsubscriptionRequest.Channel = channelToSubscribe.Currency.String()
	}
	return p.WebsocketConn.SendMessage(unsubscriptionRequest)
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
	return p.WebsocketConn.SendMessage(request)
}
