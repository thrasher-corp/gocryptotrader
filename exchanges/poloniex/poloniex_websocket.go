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
	"github.com/thrasher-corp/gocryptotrader/exchanges/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
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
				p.Websocket.DataHandler <- err
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
							log.Debugf(log.ExchangeSys, "poloniex websocket subscribed to channel successfully. %d", chanID)
						}
					} else {
						p.Websocket.DataHandler <- fmt.Errorf("poloniex websocket subscription to channel failed. %d", chanID)
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
									p.Websocket.DataHandler <- errors.New("poloniex.go error - could not find currency pair in map")
									continue
								}

								orderbookData, ok := dataL3map["orderBook"].([]interface{})
								if !ok {
									p.Websocket.DataHandler <- errors.New("poloniex.go error - could not find orderbook data in map")
									continue
								}

								err := p.WsProcessOrderbookSnapshot(orderbookData, currencyPair)
								if err != nil {
									p.Websocket.DataHandler <- err
									continue
								}

								p.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
									Exchange: p.GetName(),
									Asset:    asset.Spot,
									Pair:     currency.NewPairFromString(currencyPair),
								}
							case "o":
								currencyPair := currencyIDMap[chanID]
								err := p.WsProcessOrderbookUpdate(dataL3, currencyPair)
								if err != nil {
									p.Websocket.DataHandler <- err
									continue
								}

								p.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
									Exchange: p.GetName(),
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
								trade.Volume, _ = strconv.ParseFloat(dataL3[3].(string), 64)
								trade.Price, _ = strconv.ParseFloat(dataL3[4].(string), 64)
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
	currencyPair := currencyIDMap[int(tickerData[0].(float64))]
	t.LastPrice, _ = strconv.ParseFloat(tickerData[1].(string), 64)
	t.LowestAsk, _ = strconv.ParseFloat(tickerData[2].(string), 64)
	t.HighestBid, _ = strconv.ParseFloat(tickerData[3].(string), 64)
	t.PercentageChange, _ = strconv.ParseFloat(tickerData[4].(string), 64)
	t.BaseCurrencyVolume24H, _ = strconv.ParseFloat(tickerData[5].(string), 64)
	t.QuoteCurrencyVolume24H, _ = strconv.ParseFloat(tickerData[6].(string), 64)
	isFrozen := false
	if tickerData[7].(float64) == 1 {
		isFrozen = true
	}
	t.IsFrozen = isFrozen
	t.HighestTradeIn24H, _ = strconv.ParseFloat(tickerData[8].(string), 64)
	t.LowestTradePrice24H, _ = strconv.ParseFloat(tickerData[9].(string), 64)

	p.Websocket.DataHandler <- wshandler.TickerData{
		Timestamp:  time.Now(),
		Pair:       currency.NewPairDelimiter(currencyPair, "_"),
		Exchange:   p.GetName(),
		AssetType:  asset.Spot,
		ClosePrice: t.LastPrice,
		LowPrice:   t.LowestAsk,
		HighPrice:  t.HighestBid,
		Quantity:   t.QuoteCurrencyVolume24H,
	}
}

// wsHandleAccountData Parses account data and sends to datahandler
func (p *Poloniex) wsHandleAccountData(accountData [][]interface{}) {
	for i := range accountData {
		switch accountData[i][0].(string) {
		case "b":
			amount, _ := strconv.ParseFloat(accountData[i][3].(string), 64)
			response := WsAccountBalanceUpdateResponse{
				currencyID: accountData[i][1].(float64),
				wallet:     accountData[i][2].(string),
				amount:     amount,
			}
			p.Websocket.DataHandler <- response
		case "n":
			timeParse, _ := time.Parse("2006-01-02 15:04:05", accountData[i][6].(string))
			rate, _ := strconv.ParseFloat(accountData[i][4].(string), 64)
			amount, _ := strconv.ParseFloat(accountData[i][5].(string), 64)

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
			timeParse, _ := time.Parse("2006-01-02 15:04:05", accountData[i][8].(string))
			rate, _ := strconv.ParseFloat(accountData[i][2].(string), 64)
			amount, _ := strconv.ParseFloat(accountData[i][3].(string), 64)
			feeMultiplier, _ := strconv.ParseFloat(accountData[i][4].(string), 64)
			totalFee, _ := strconv.ParseFloat(accountData[i][7].(string), 64)

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
	newOrderBook.LastUpdated = time.Now()
	newOrderBook.Pair = currency.NewPairFromString(symbol)

	return p.Websocket.Orderbook.LoadSnapshot(&newOrderBook, p.GetName(), false)
}

// WsProcessOrderbookUpdate processses new orderbook updates
func (p *Poloniex) WsProcessOrderbookUpdate(target []interface{}, symbol string) error {
	sideCheck := target[1].(float64)

	cP := currency.NewPairFromString(symbol)

	price, err := strconv.ParseFloat(target[2].(string), 64)
	if err != nil {
		return err
	}

	volume, err := strconv.ParseFloat(target[3].(string), 64)
	if err != nil {
		return err
	}

	if sideCheck == 0 {
		return p.Websocket.Orderbook.Update(nil,
			[]orderbook.Item{{Price: price, Amount: volume}},
			cP,
			time.Now(),
			p.GetName(),
			asset.Spot)
	}

	return p.Websocket.Orderbook.Update([]orderbook.Item{{Price: price, Amount: volume}},
		nil,
		cP,
		time.Now(),
		p.GetName(),
		asset.Spot)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (p *Poloniex) GenerateDefaultSubscriptions() {
	var subscriptions []wshandler.WebsocketChannelSubscription
	// Tickerdata is its own channel
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
		enabledCurrencies[j].Delimiter = "_"
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
