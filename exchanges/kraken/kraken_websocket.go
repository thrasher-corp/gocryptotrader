package kraken

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// List of all websocket channels to subscribe to
const (
	krakenWSURL              = "wss://ws.kraken.com"
	krakenAuthWSURL          = "wss://ws-auth.kraken.com"
	krakenWSSandboxURL       = "wss://sandbox.kraken.com"
	krakenWSSupportedVersion = "0.3.0"
	// WS endpoints
	krakenWsHeartbeat          = "heartbeat"
	krakenWsSystemStatus       = "systemStatus"
	krakenWsSubscribe          = "subscribe"
	krakenWsSubscriptionStatus = "subscriptionStatus"
	krakenWsUnsubscribe        = "unsubscribe"
	krakenWsTicker             = "ticker"
	krakenWsOHLC               = "ohlc"
	krakenWsTrade              = "trade"
	krakenWsSpread             = "spread"
	krakenWsOrderbook          = "book"
	krakenWsOwnTrades          = "ownTrades"
	krakenWsOpenOrders         = "openOrders"
	krakenWsAddOrder           = "addOrder"
	krakenWsCancelOrder        = "cancelOrder"
	krakenWsRateLimit          = 50
	krakenWsPingDelay          = time.Second * 27
)

// orderbookMutex Ensures if two entries arrive at once, only one can be processed at a time
var subscriptionChannelPair []WebsocketChannelData
var comms = make(chan wshandler.WebsocketResponse)
var authToken string
var pingRequest = WebsocketBaseEventRequest{Event: wshandler.Ping}

// Channels require a topic and a currency
// Format [[ticker,but-t4u],[orderbook,nce-btt]]
var defaultSubscribedChannels = []string{krakenWsTicker, krakenWsTrade, krakenWsOrderbook, krakenWsOHLC, krakenWsSpread}
var authenticatedChannels = []string{krakenWsOwnTrades, krakenWsOpenOrders}

// WsConnect initiates a websocket connection
func (k *Kraken) WsConnect() error {
	if !k.Websocket.IsEnabled() || !k.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := k.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if k.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		authToken, err = k.GetWebsocketToken()
		if err != nil {
			k.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", k.Name, err)
		}
		err = k.AuthenticatedWebsocketConn.Dial(&dialer, http.Header{})
		if err != nil {
			k.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys, "%v - failed to connect to authenticated endpoint: %v\n", k.Name, err)
		}
		go k.WsReadData(k.AuthenticatedWebsocketConn)
		k.GenerateAuthenticatedSubscriptions()
	}

	go k.WsReadData(k.WebsocketConn)
	go k.wsReadData()
	err = k.wsPingHandler()
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v - failed setup ping handler. Websocket may disconnect unexpectedly. %v\n", k.Name, err)
	}
	k.GenerateDefaultSubscriptions()

	return nil
}

// WsReadData funnels both auth and public ws data into one manageable place
func (k *Kraken) WsReadData(ws *wshandler.WebsocketConnection) {
	k.Websocket.Wg.Add(1)
	defer k.Websocket.Wg.Done()
	for {
		select {
		case <-k.Websocket.ShutdownC:
			return
		default:
			resp, err := ws.ReadMessage()
			if err != nil {
				k.Websocket.DataHandler <- err
				return
			}
			k.Websocket.TrafficAlert <- struct{}{}
			comms <- resp
		}
	}
}

// wsReadData handles the read data from the websocket connection
func (k *Kraken) wsReadData() {
	k.Websocket.Wg.Add(1)
	defer func() {
		k.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-k.Websocket.ShutdownC:
			return
		default:
			resp := <-comms
			// event response handling
			var eventResponse WebsocketEventResponse
			err := json.Unmarshal(resp.Raw, &eventResponse)
			if err == nil && eventResponse.Event != "" {
				k.WsHandleEventResponse(&eventResponse, resp.Raw)
				continue
			}
			// Data response handling
			var dataResponse WebsocketDataResponse
			err = json.Unmarshal(resp.Raw, &dataResponse)
			if err != nil {
				log.Error(log.WebsocketMgr, fmt.Errorf("%s - unhandled websocket data: %v", k.Name, err))
				continue
			}
			if _, ok := dataResponse[0].(float64); ok {
				k.wsReadDataResponse(dataResponse)
			}
			if _, ok := dataResponse[1].(string); ok {
				k.wsHandleAuthDataResponse(dataResponse)
			}
		}
	}
}

// wsPingHandler sends a message "ping" every 27 to maintain the connection to the websocket
func (k *Kraken) wsPingHandler() error {
	message, err := json.Marshal(pingRequest)
	if err != nil {
		return err
	}
	k.WebsocketConn.SetupPingHandler(wshandler.WebsocketPingHandler{
		Message:     message,
		Delay:       krakenWsPingDelay,
		MessageType: websocket.TextMessage,
	})
	return nil
}

// wsReadDataResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) wsReadDataResponse(response WebsocketDataResponse) {
	if cID, ok := response[0].(float64); ok {
		channelID := int64(cID)
		channelData := getSubscriptionChannelData(channelID)
		switch channelData.Subscription {
		case krakenWsTicker:
			if k.Verbose {
				log.Debugf(log.ExchangeSys, "%v Websocket ticker data received",
					k.Name)
			}
			k.wsProcessTickers(&channelData, response[1].(map[string]interface{}))
		case krakenWsOHLC:
			if k.Verbose {
				log.Debugf(log.ExchangeSys, "%v Websocket OHLC data received",
					k.Name)
			}
			k.wsProcessCandles(&channelData, response[1].([]interface{}))
		case krakenWsOrderbook:
			if k.Verbose {
				log.Debugf(log.ExchangeSys, "%v Websocket Orderbook data received",
					k.Name)
			}
			k.wsProcessOrderBook(&channelData, response[1].(map[string]interface{}))
		case krakenWsSpread:
			if k.Verbose {
				log.Debugf(log.ExchangeSys, "%v Websocket Spread data received",
					k.Name)
			}
			k.wsProcessSpread(&channelData, response[1].([]interface{}))
		case krakenWsTrade:
			if k.Verbose {
				log.Debugf(log.ExchangeSys, "%v Websocket Trade data received",
					k.Name)
			}
			k.wsProcessTrades(&channelData, response[1].([]interface{}))
		default:
			log.Errorf(log.ExchangeSys, "%v Unidentified websocket data received: %v",
				k.Name,
				response)
		}
	}
}

// WsHandleEventResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) WsHandleEventResponse(response *WebsocketEventResponse, rawResponse []byte) {
	switch response.Event {
	case wshandler.Pong:
		break
	case krakenWsHeartbeat:
		if k.Verbose {
			log.Debugf(log.ExchangeSys, "%v Websocket heartbeat data received",
				k.Name)
		}
	case krakenWsSystemStatus:
		if k.Verbose {
			log.Debugf(log.ExchangeSys, "%v Websocket status data received",
				k.Name)
		}
		if response.Status != "online" {
			k.Websocket.DataHandler <- fmt.Errorf("%v Websocket status '%v'",
				k.Name, response.Status)
		}
		if response.WebsocketStatusResponse.Version > krakenWSSupportedVersion {
			log.Warnf(log.ExchangeSys, "%v New version of Websocket API released. Was %v Now %v",
				k.Name, krakenWSSupportedVersion, response.WebsocketStatusResponse.Version)
		}
	case krakenWsSubscriptionStatus:
		k.WebsocketConn.AddResponseWithID(response.RequestID, rawResponse)
		if response.Status != "subscribed" {
			k.Websocket.DataHandler <- fmt.Errorf("%v %v %v", k.Name, response.RequestID, response.WebsocketErrorResponse.ErrorMessage)
			return
		}
		addNewSubscriptionChannelData(response)
	default:
		log.Errorf(log.ExchangeSys, "%v Unidentified websocket data received: %v",
			k.Name, response)
	}
}

func (k *Kraken) wsHandleAuthDataResponse(response WebsocketDataResponse) {
	if chName, ok := response[1].(string); ok {
		switch chName {
		case krakenWsOwnTrades:
			if k.Verbose {
				log.Debugf(log.ExchangeSys, "%v Websocket auth own trade data received",
					k.Name)
			}
			k.wsProcessOwnTrades(&response[0])
		case krakenWsOpenOrders:
			if k.Verbose {
				log.Debugf(log.ExchangeSys, "%v Websocket auth open order data received",
					k.Name)
			}
			k.wsProcessOpenOrders(&response[0])
		}
	}
}

func (k *Kraken) wsProcessOwnTrades(ownOrders interface{}) {
	if data, ok := ownOrders.([]interface{}); ok {
		for i := range data {
			ownTrade := data[i].(map[string]interface{})
			for _, val := range ownTrade {
				tradeData := val.(map[string]interface{})
				cost, err := strconv.ParseFloat(tradeData["cost"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				fee, err := strconv.ParseFloat(tradeData["fee"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				margin, err := strconv.ParseFloat(tradeData["margin"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				vol, err := strconv.ParseFloat(tradeData["vol"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				price, err := strconv.ParseFloat(tradeData["price"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				timeTogether, err := strconv.ParseFloat(tradeData["time"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				first, second, err := convert.SplitFloatDecimals(timeTogether)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				k.Websocket.DataHandler <- WsOwnTrade{
					Cost:               cost,
					Fee:                fee,
					Margin:             margin,
					OrderTransactionID: tradeData["ordertxid"].(string),
					OrderType:          tradeData["ordertype"].(string),
					Pair:               tradeData["pair"].(string),
					PostTransactionID:  tradeData["postxid"].(string),
					Price:              price,
					Time:               time.Unix(first, second),
					Type:               tradeData["type"].(string),
					Vol:                vol,
				}
			}
		}
	} else {
		k.Websocket.DataHandler <- errors.New(k.Name + " - Invalid own trades data")
	}
}

func (k *Kraken) wsProcessOpenOrders(ownOrders interface{}) {
	if data, ok := ownOrders.([]interface{}); ok {
		for i := range data {
			ownTrade := data[i].(map[string]interface{})
			for key, val := range ownTrade {
				tradeData := val.(map[string]interface{})
				if len(tradeData) == 1 {
					// just a status update
					if status, ok := tradeData["status"].(string); ok {
						k.Websocket.DataHandler <- k.Name + " - Order " + key + " " + status
					}
				}
				startTimeConv, err := strconv.ParseFloat(tradeData["starttm"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				startTime, startTimeNano, err := convert.SplitFloatDecimals(startTimeConv)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				openTimeConv, err := strconv.ParseFloat(tradeData["opentm"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				openTime, openTimeNano, err := convert.SplitFloatDecimals(openTimeConv)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				expireTimeConv, err := strconv.ParseFloat(tradeData["expiretm"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				expireTime, expireTimeNano, err := convert.SplitFloatDecimals(expireTimeConv)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				cost, err := strconv.ParseFloat(tradeData["cost"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				executedVolume, err := strconv.ParseFloat(tradeData["vol_exec"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				volume, err := strconv.ParseFloat(tradeData["vol"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				userReference, err := strconv.ParseFloat(tradeData["userref"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				stopPrice, err := strconv.ParseFloat(tradeData["stopprice"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				price, err := strconv.ParseFloat(tradeData["price"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				limitPrice, err := strconv.ParseFloat(tradeData["limitprice"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				fee, err := strconv.ParseFloat(tradeData["fee"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				descriptionSubData := tradeData["description"].(map[string]interface{})
				descriptionPrice, err := strconv.ParseFloat(descriptionSubData["price"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				descriptionPrice2, err := strconv.ParseFloat(descriptionSubData["price2"].(string), 64)
				if err != nil {
					k.Websocket.DataHandler <- err
				}
				description := WsOpenOrderDescription{
					Close:     descriptionSubData["close"].(string),
					Leverage:  descriptionSubData["leverage"].(string),
					Order:     descriptionSubData["order"].(string),
					OrderType: descriptionSubData["ordertype"].(string),
					Pair:      descriptionSubData["pair"].(string),
					Price:     descriptionPrice,
					Price2:    descriptionPrice2,
					Type:      descriptionSubData["type"].(string),
				}

				k.Websocket.DataHandler <- WsOpenOrders{
					Cost:           cost,
					ExpireTime:     time.Unix(expireTime, expireTimeNano),
					Description:    description,
					Fee:            fee,
					LimitPrice:     limitPrice,
					Misc:           tradeData["misc"].(string),
					OFlags:         tradeData["oflags"].(string),
					OpenTime:       time.Unix(openTime, openTimeNano),
					Price:          price,
					RefID:          tradeData["refid"].(string),
					StartTime:      time.Unix(startTime, startTimeNano),
					Status:         tradeData["status"].(string),
					StopPrice:      stopPrice,
					UserReference:  userReference,
					Volume:         volume,
					ExecutedVolume: executedVolume,
				}
			}
		}
	} else {
		k.Websocket.DataHandler <- errors.New(k.Name + " - Invalid own trades data")
	}
}

// addNewSubscriptionChannelData stores channel ids, pairs and subscription types to an array
// allowing correlation between subscriptions and returned data
func addNewSubscriptionChannelData(response *WebsocketEventResponse) {
	// We change the / to - to maintain compatibility with REST/config
	pair := currency.NewPairWithDelimiter(response.Pair.Base.String(),
		response.Pair.Quote.String(), "-")
	subscriptionChannelPair = append(subscriptionChannelPair, WebsocketChannelData{
		Subscription: response.Subscription.Name,
		Pair:         pair,
		ChannelID:    response.ChannelID,
	})
}

// getSubscriptionChannelData retrieves WebsocketChannelData based on response ID
func getSubscriptionChannelData(id int64) WebsocketChannelData {
	for i := range subscriptionChannelPair {
		if id == subscriptionChannelPair[i].ChannelID {
			return subscriptionChannelPair[i]
		}
	}
	return WebsocketChannelData{}
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (k *Kraken) wsProcessTickers(channelData *WebsocketChannelData, data map[string]interface{}) {
	closePrice, err := strconv.ParseFloat(data["c"].([]interface{})[0].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	openPrice, err := strconv.ParseFloat(data["o"].([]interface{})[0].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	highPrice, err := strconv.ParseFloat(data["h"].([]interface{})[0].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	lowPrice, err := strconv.ParseFloat(data["l"].([]interface{})[0].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	quantity, err := strconv.ParseFloat(data["v"].([]interface{})[0].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	ask, err := strconv.ParseFloat(data["a"].([]interface{})[0].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	bid, err := strconv.ParseFloat(data["b"].([]interface{})[0].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	k.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: k.Name,
		Open:         openPrice,
		Close:        closePrice,
		Volume:       quantity,
		High:         highPrice,
		Low:          lowPrice,
		Bid:          bid,
		Ask:          ask,
		AssetType:    asset.Spot,
		Pair:         channelData.Pair,
	}
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (k *Kraken) wsProcessSpread(channelData *WebsocketChannelData, data []interface{}) {
	bestBid := data[0].(string)
	bestAsk := data[1].(string)
	timeData, err := strconv.ParseFloat(data[2].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	bidVolume := data[3].(string)
	askVolume := data[4].(string)
	sec, dec := math.Modf(timeData)
	spreadTimestamp := time.Unix(int64(sec), int64(dec*(1e9)))
	if k.Verbose {
		log.Debugf(log.ExchangeSys,
			"%v Spread data for '%v' received. Best bid: '%v' Best ask: '%v' Time: '%v', Bid volume '%v', Ask volume '%v'",
			k.Name,
			channelData.Pair,
			bestBid,
			bestAsk,
			spreadTimestamp,
			bidVolume,
			askVolume)
	}
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (k *Kraken) wsProcessTrades(channelData *WebsocketChannelData, data []interface{}) {
	for i := range data {
		trade := data[i].([]interface{})
		timeData, err := strconv.ParseFloat(trade[2].(string), 64)
		if err != nil {
			k.Websocket.DataHandler <- err
			return
		}
		sec, dec := math.Modf(timeData)
		timeUnix := time.Unix(int64(sec), int64(dec*(1e9)))

		price, err := strconv.ParseFloat(trade[0].(string), 64)
		if err != nil {
			k.Websocket.DataHandler <- err
			return
		}

		amount, err := strconv.ParseFloat(trade[1].(string), 64)
		if err != nil {
			k.Websocket.DataHandler <- err
			return
		}

		k.Websocket.DataHandler <- wshandler.TradeData{
			AssetType:    asset.Spot,
			CurrencyPair: channelData.Pair,
			Exchange:     k.Name,
			Price:        price,
			Amount:       amount,
			Timestamp:    timeUnix,
			Side:         trade[3].(string),
		}
	}
}

// wsProcessOrderBook determines if the orderbook data is partial or update
// Then sends to appropriate fun
func (k *Kraken) wsProcessOrderBook(channelData *WebsocketChannelData, data map[string]interface{}) {
	if fullAsk, ok := data["as"].([]interface{}); ok {
		fullBids := data["as"].([]interface{})
		k.wsProcessOrderBookPartial(channelData, fullAsk, fullBids)
	} else {
		askData, asksExist := data["a"].([]interface{})
		bidData, bidsExist := data["b"].([]interface{})
		if asksExist || bidsExist {
			k.wsRequestMtx.Lock()
			defer k.wsRequestMtx.Unlock()
			err := k.wsProcessOrderBookUpdate(channelData, askData, bidData)
			if err != nil {
				subscriptionToRemove := wshandler.WebsocketChannelSubscription{
					Channel:  krakenWsOrderbook,
					Currency: channelData.Pair,
				}
				k.Websocket.ResubscribeToChannel(subscriptionToRemove)
			}
		}
	}
}

// wsProcessOrderBookPartial creates a new orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookPartial(channelData *WebsocketChannelData, askData, bidData []interface{}) {
	base := orderbook.Base{
		Pair:      channelData.Pair,
		AssetType: asset.Spot,
	}
	// Kraken ob data is timestamped per price, GCT orderbook data is
	// timestamped per entry using the highest last update time, we can attempt
	// to respect both within a reasonable degree
	var highestLastUpdate time.Time
	for i := range askData {
		asks := askData[i].([]interface{})
		price, err := strconv.ParseFloat(asks[0].(string), 64)
		if err != nil {
			k.Websocket.DataHandler <- err
			return
		}
		amount, err := strconv.ParseFloat(asks[1].(string), 64)
		if err != nil {
			k.Websocket.DataHandler <- err
			return
		}
		base.Asks = append(base.Asks, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
		timeData, err := strconv.ParseFloat(asks[2].(string), 64)
		if err != nil {
			k.Websocket.DataHandler <- err
			return
		}
		sec, dec := math.Modf(timeData)
		askUpdatedTime := time.Unix(int64(sec), int64(dec*(1e9)))
		if highestLastUpdate.Before(askUpdatedTime) {
			highestLastUpdate = askUpdatedTime
		}
	}

	for i := range bidData {
		bids := bidData[i].([]interface{})
		price, err := strconv.ParseFloat(bids[0].(string), 64)
		if err != nil {
			k.Websocket.DataHandler <- err
			return
		}
		amount, err := strconv.ParseFloat(bids[1].(string), 64)
		if err != nil {
			k.Websocket.DataHandler <- err
			return
		}
		base.Bids = append(base.Bids, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
		timeData, err := strconv.ParseFloat(bids[2].(string), 64)
		if err != nil {
			k.Websocket.DataHandler <- err
			return
		}
		sec, dec := math.Modf(timeData)
		bidUpdateTime := time.Unix(int64(sec), int64(dec*(1e9)))
		if highestLastUpdate.Before(bidUpdateTime) {
			highestLastUpdate = bidUpdateTime
		}
	}
	base.LastUpdated = highestLastUpdate
	base.ExchangeName = k.Name
	err := k.Websocket.Orderbook.LoadSnapshot(&base)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}
	k.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: k.Name,
		Asset:    asset.Spot,
		Pair:     channelData.Pair,
	}
}

// wsProcessOrderBookUpdate updates an orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookUpdate(channelData *WebsocketChannelData, askData, bidData []interface{}) error {
	update := wsorderbook.WebsocketOrderbookUpdate{
		Asset: asset.Spot,
		Pair:  channelData.Pair,
	}

	var highestLastUpdate time.Time
	// Ask data is not always sent
	for i := range askData {
		asks := askData[i].([]interface{})
		price, err := strconv.ParseFloat(asks[0].(string), 64)
		if err != nil {
			return err
		}

		amount, err := strconv.ParseFloat(asks[1].(string), 64)
		if err != nil {
			return err
		}

		update.Asks = append(update.Asks, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
		timeData, err := strconv.ParseFloat(asks[2].(string), 64)
		if err != nil {
			return err
		}

		sec, dec := math.Modf(timeData)
		askUpdatedTime := time.Unix(int64(sec), int64(dec*(1e9)))
		if highestLastUpdate.Before(askUpdatedTime) {
			highestLastUpdate = askUpdatedTime
		}
	}

	// Bid data is not always sent
	for i := range bidData {
		bids := bidData[i].([]interface{})
		price, err := strconv.ParseFloat(bids[0].(string), 64)
		if err != nil {
			return err
		}

		amount, err := strconv.ParseFloat(bids[1].(string), 64)
		if err != nil {
			return err
		}

		update.Bids = append(update.Bids, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
		timeData, err := strconv.ParseFloat(bids[2].(string), 64)
		if err != nil {
			return err
		}

		sec, dec := math.Modf(timeData)
		bidUpdatedTime := time.Unix(int64(sec), int64(dec*(1e9)))
		if highestLastUpdate.Before(bidUpdatedTime) {
			highestLastUpdate = bidUpdatedTime
		}
	}
	update.UpdateTime = highestLastUpdate
	err := k.Websocket.Orderbook.Update(&update)
	if err != nil {
		k.Websocket.DataHandler <- err
		return err
	}
	k.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: k.Name,
		Asset:    asset.Spot,
		Pair:     channelData.Pair,
	}
	return nil
}

// wsProcessCandles converts candle data and sends it to the data handler
func (k *Kraken) wsProcessCandles(channelData *WebsocketChannelData, data []interface{}) {
	startTime, err := strconv.ParseFloat(data[0].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}
	sec, dec := math.Modf(startTime)
	startTimeUnix := time.Unix(int64(sec), int64(dec*(1e9)))

	endTime, err := strconv.ParseFloat(data[1].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}
	sec, dec = math.Modf(endTime)
	endTimeUnix := time.Unix(int64(sec), int64(dec*(1e9)))

	openPrice, err := strconv.ParseFloat(data[2].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	highPrice, err := strconv.ParseFloat(data[3].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	lowPrice, err := strconv.ParseFloat(data[4].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	closePrice, err := strconv.ParseFloat(data[5].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	volume, err := strconv.ParseFloat(data[7].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	k.Websocket.DataHandler <- wshandler.KlineData{
		AssetType: asset.Spot,
		Pair:      channelData.Pair,
		Timestamp: time.Now(),
		Exchange:  k.Name,
		StartTime: startTimeUnix,
		CloseTime: endTimeUnix,
		// Candles are sent every 60 seconds
		Interval:   "60",
		HighPrice:  highPrice,
		LowPrice:   lowPrice,
		OpenPrice:  openPrice,
		ClosePrice: closePrice,
		Volume:     volume,
	}
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (k *Kraken) GenerateDefaultSubscriptions() {
	enabledCurrencies := k.GetEnabledPairs(asset.Spot)
	var subscriptions []wshandler.WebsocketChannelSubscription
	for i := range defaultSubscribedChannels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = "/"
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  defaultSubscribedChannels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	k.Websocket.SubscribeToChannels(subscriptions)
}

// GenerateAuthenticatedSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (k *Kraken) GenerateAuthenticatedSubscriptions() {
	var subscriptions []wshandler.WebsocketChannelSubscription
	for i := range authenticatedChannels {
		params := make(map[string]interface{})
		subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
			Channel: authenticatedChannels[i],
			Params:  params,
		})
	}
	k.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (k *Kraken) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	resp := WebsocketSubscriptionEventRequest{
		Event: krakenWsSubscribe,
		Subscription: WebsocketSubscriptionData{
			Name: channelToSubscribe.Channel,
		},
		RequestID: k.WebsocketConn.GenerateMessageID(false),
	}
	if channelToSubscribe.Channel == "book" {
		// TODO: Add ability to make depth customisable
		resp.Subscription.Depth = 1000
	}
	if !channelToSubscribe.Currency.IsEmpty() {
		resp.Pairs = []string{channelToSubscribe.Currency.String()}
	}
	if channelToSubscribe.Params != nil {
		resp.Subscription.Token = authToken
	}

	_, err := k.WebsocketConn.SendMessageReturnResponse(resp.RequestID, resp)
	return err
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (k *Kraken) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	resp := WebsocketSubscriptionEventRequest{
		Event: krakenWsUnsubscribe,
		Pairs: []string{channelToSubscribe.Currency.String()},
		Subscription: WebsocketSubscriptionData{
			Name: channelToSubscribe.Channel,
		},
		RequestID: k.WebsocketConn.GenerateMessageID(false),
	}
	_, err := k.WebsocketConn.SendMessageReturnResponse(resp.RequestID, resp)
	return err
}

func (k *Kraken) wsAddOrder(request *WsAddOrderRequest) (string, error) {
	id := k.AuthenticatedWebsocketConn.GenerateMessageID(false)
	request.UserReferenceID = strconv.FormatInt(id, 10)
	request.Event = krakenWsAddOrder
	request.Token = authToken
	jsonResp, err := k.AuthenticatedWebsocketConn.SendMessageReturnResponse(id, request)
	if err != nil {
		return "", err
	}
	var resp WsAddOrderResponse
	err = json.Unmarshal(jsonResp, &resp)
	if err != nil {
		return "", err
	}
	if resp.ErrorMessage != "" {
		return "", fmt.Errorf(k.Name + " - " + resp.ErrorMessage)
	}
	return resp.TransactionID, nil
}

func (k *Kraken) wsCancelOrders(orderIDs []string) error {
	request := WsCancelOrderRequest{
		Event:          krakenWsCancelOrder,
		Token:          authToken,
		TransactionIDs: orderIDs,
	}
	return k.AuthenticatedWebsocketConn.SendJSONMessage(request)
}
