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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// List of all websocket channels to subscribe to
const (
	krakenWSURL              = "wss://ws.kraken.com"
	krakenWSSandboxURL       = "wss://sandbox.kraken.com"
	krakenWSSupportedVersion = "0.2.0"
	// If a checksum fails, then resubscribing to the channel fails, fatal after these attempts
	krakenWsResubscribeFailureLimit   = 3
	krakenWsResubscribeDelayInSeconds = 3
	// WS endpoints
	krakenWsHeartbeat          = "heartbeat"
	krakenWsPing               = "ping"
	krakenWsPong               = "pong"
	krakenWsSystemStatus       = "systemStatus"
	krakenWsSubscribe          = "subscribe"
	krakenWsSubscriptionStatus = "subscriptionStatus"
	krakenWsUnsubscribe        = "unsubscribe"
	krakenWsTicker             = "ticker"
	krakenWsOHLC               = "ohlc"
	krakenWsTrade              = "trade"
	krakenWsSpread             = "spread"
	krakenWsOrderbook          = "book"

	orderbookBufferLimit = 3
	krakenWsRateLimit    = 50
)

// orderbookMutex Ensures if two entries arrive at once, only one can be processed at a time
var subscriptionChannelPair []WebsocketChannelData
var subscribeToDefaultChannels = true

// Channels require a topic and a currency
// Format [[ticker,but-t4u],[orderbook,nce-btt]]
var defaultSubscribedChannels = []string{krakenWsTicker, krakenWsTrade, krakenWsOrderbook, krakenWsOHLC, krakenWsSpread}

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
	go k.WsHandleData()
	go k.wsPingHandler()
	if subscribeToDefaultChannels {
		k.GenerateDefaultSubscriptions()
	}

	return nil
}

// wsPingHandler sends a message "ping" every 27 to maintain the connection to the websocket
func (k *Kraken) wsPingHandler() {
	k.Websocket.Wg.Add(1)
	defer k.Websocket.Wg.Done()
	ticker := time.NewTicker(time.Second * 27)
	defer ticker.Stop()

	for {
		select {
		case <-k.Websocket.ShutdownC:
			return
		case <-ticker.C:
			pingEvent := WebsocketBaseEventRequest{Event: krakenWsPing}
			if k.Verbose {
				log.Debugf(log.ExchangeSys, "%v sending ping",
					k.Name)
			}
			err := k.WebsocketConn.SendMessage(pingEvent)
			if err != nil {
				k.Websocket.DataHandler <- err
			}
		}
	}
}

// WsHandleData handles the read data from the websocket connection
func (k *Kraken) WsHandleData() {
	k.Websocket.Wg.Add(1)
	defer func() {
		k.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-k.Websocket.ShutdownC:
			return
		default:
			resp, err := k.WebsocketConn.ReadMessage()
			if err != nil {
				k.Websocket.ReadMessageErrors <- err
				return
			}
			k.Websocket.TrafficAlert <- struct{}{}
			// event response handling
			var eventResponse WebsocketEventResponse
			err = json.Unmarshal(resp.Raw, &eventResponse)
			if err == nil && eventResponse.Event != "" {
				k.WsHandleEventResponse(&eventResponse, resp.Raw)
				continue
			}
			// Data response handling
			var dataResponse WebsocketDataResponse
			err = json.Unmarshal(resp.Raw, &dataResponse)
			if err == nil && dataResponse[0].(float64) >= 0 {
				k.WsHandleDataResponse(dataResponse)
				continue
			}
			continue
		}
	}
}

// WsHandleDataResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) WsHandleDataResponse(response WebsocketDataResponse) {
	channelID := int64(response[0].(float64))
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

// WsHandleEventResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) WsHandleEventResponse(response *WebsocketEventResponse, rawResponse []byte) {
	switch response.Event {
	case krakenWsHeartbeat:
		if k.Verbose {
			log.Debugf(log.ExchangeSys, "%v Websocket heartbeat data received",
				k.Name)
		}
	case krakenWsPong:
		if k.Verbose {
			log.Debugf(log.ExchangeSys, "%v Websocket pong data received",
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
		if response.WebsocketStatusResponse.Version != krakenWSSupportedVersion {
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

	k.Websocket.DataHandler <- wshandler.TickerData{
		Exchange:  k.Name,
		Open:      openPrice,
		Close:     closePrice,
		Volume:    quantity,
		High:      highPrice,
		Low:       lowPrice,
		Bid:       bid,
		Ask:       ask,
		Timestamp: time.Now(),
		AssetType: asset.Spot,
		Pair:      channelData.Pair,
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
			EventTime:    time.Now().Unix(),
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

// Subscribe sends a websocket message to receive data from the channel
func (k *Kraken) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var depth int64
	if channelToSubscribe.Channel == "book" {
		depth = 1000
	}

	resp := WebsocketSubscriptionEventRequest{
		Event: krakenWsSubscribe,
		Pairs: []string{channelToSubscribe.Currency.String()},
		Subscription: WebsocketSubscriptionData{
			Name:  channelToSubscribe.Channel,
			Depth: depth,
		},
		RequestID: k.WebsocketConn.GenerateMessageID(false),
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
