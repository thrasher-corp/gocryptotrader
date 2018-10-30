package kraken

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// List of all websocket channels to subscribe to
const (
	krakenWSURL              = "wss://ws.kraken.com"
	krakenWSSandboxURL       = "wss://sandbox.kraken.com"
	krakenWSSupportedVersion = "0.1.1"
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
	// Only supported asset type
	krakenWsAssetType    = "SPOT"
	orderbookBufferLimit = 3
)

// orderbookMutex Ensures if two entries arrive at once, only one can be processed at a time
var orderbookMutex sync.Mutex
var subscriptionChannelPair []WebsocketChannelData

// krakenOrderBooks TODO THIS IS A TEMPORARY SOLUTION UNTIL ENGINE BRANCH IS MERGED
// WS orderbook data can only rely on WS orderbook data
// Currently REST and WS runs simultaneously, dirtying the data
var krakenOrderBooks map[int64]orderbook.Base

// orderbookBuffer Stores orderbook updates per channel
var orderbookBuffer map[int64][]orderbook.Base
var subscribeToDefaultChannels = true

// writeToWebsocket sends a message to the websocket endpoint
func (k *Kraken) writeToWebsocket(message []byte) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.Verbose {
		log.Debugf("Sending message to WS: %v",
			string(message))
	}
	// Really basic WS rate limit
	time.Sleep(30 * time.Millisecond)
	return k.WebsocketConn.WriteMessage(websocket.TextMessage, message)
}

// WsConnect initiates a websocket connection
func (k *Kraken) WsConnect() error {
	if !k.Websocket.IsEnabled() || !k.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	if k.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(k.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	if k.Verbose {
		log.Debugf("Attempting to connect to %v",
			k.Websocket.GetWebsocketURL())
	}
	k.WebsocketConn, _, err = dialer.Dial(k.Websocket.GetWebsocketURL(),
		http.Header{})
	if err != nil {
		return fmt.Errorf("%s Unable to connect to Websocket. Error: %s",
			k.Name,
			err)
	}
	if k.Verbose {
		log.Debugf("Successful connection to %v",
			k.Websocket.GetWebsocketURL())
	}
	go k.WsHandleData()
	go k.wsPingHandler()
	if subscribeToDefaultChannels {
		k.WsSubscribeToDefaults()
	}
	return nil
}

// WsSubscribeToDefaults subscribes to the websocket channels
func (k *Kraken) WsSubscribeToDefaults() {
	channelsToSubscribe := []string{krakenWsTicker, krakenWsTrade, krakenWsOrderbook, krakenWsOHLC, krakenWsSpread}
	for _, pair := range k.GetEnabledPairs(assets.AssetTypeSpot) {
		// Kraken WS formats pairs with / but config and REST use -
		formattedPair := strings.ToUpper(strings.Replace(pair.String(), "-", "/", 1))
		for _, channel := range channelsToSubscribe {
			err := k.WsSubscribeToChannel(channel, []string{formattedPair}, 0)
			if err != nil {
				k.Websocket.DataHandler <- err
			}
		}
	}
}

// WsReadData reads data from the websocket connection
func (k *Kraken) WsReadData() (exchange.WebsocketResponse, error) {
	mType, resp, err := k.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}
	k.Websocket.TrafficAlert <- struct{}{}
	var standardMessage []byte
	switch mType {
	case websocket.TextMessage:
		standardMessage = resp

	case websocket.BinaryMessage:
		reader := flate.NewReader(bytes.NewReader(resp))
		standardMessage, err = ioutil.ReadAll(reader)
		reader.Close()
		if err != nil {
			return exchange.WebsocketResponse{}, err
		}
	}
	if k.Verbose {
		log.Debugf("%v Websocket message received: %v",
			k.Name,
			string(standardMessage))
	}

	return exchange.WebsocketResponse{Raw: standardMessage}, nil
}

// wsPingHandler sends a message "ping" every 27 to maintain the connection to the websocket
func (k *Kraken) wsPingHandler() {
	k.Websocket.Wg.Add(1)
	defer k.Websocket.Wg.Done()
	ticker := time.NewTicker(time.Second * 27)
	for {
		select {
		case <-k.Websocket.ShutdownC:
			return

		case <-ticker.C:
			pingEvent := fmt.Sprintf("{\"event\":\"%v\"}", krakenWsPing)
			if k.Verbose {
				log.Debugf("%v sending ping",
					k.GetName())
			}
			err := k.writeToWebsocket([]byte(pingEvent))
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
		err := k.WebsocketConn.Close()
		if err != nil {
			k.Websocket.DataHandler <- fmt.Errorf("%v unable to to close Websocket connection. Error: %s",
				k.GetName(), err)
		}
		k.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-k.Websocket.ShutdownC:
			return
		default:
			resp, err := k.WsReadData()
			if err != nil {
				k.Websocket.DataHandler <- err
				return
			}
			// event response handling
			var eventResponse WebsocketEventResponse
			err = common.JSONDecode(resp.Raw, &eventResponse)
			if err == nil && eventResponse.Event != "" {
				k.WsHandleEventResponse(&eventResponse)
				continue
			}
			// Data response handling
			var dataResponse WebsocketDataResponse
			err = common.JSONDecode(resp.Raw, &dataResponse)
			if err == nil && dataResponse[0].(float64) >= 0 {
				k.WsHandleDataResponse(dataResponse)
				continue
			}
			// Unknown data handling
			k.Websocket.DataHandler <- fmt.Errorf("unrecognised response: %v", string(resp.Raw))
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
			log.Debugf("%v Websocket ticker data received",
				k.GetName())
		}
		k.wsProcessTickers(&channelData, response[1])
	case krakenWsOHLC:
		if k.Verbose {
			log.Debugf("%v Websocket OHLC data received",
				k.GetName())
		}
		k.wsProcessCandles(&channelData, response[1])
	case krakenWsOrderbook:
		if k.Verbose {
			log.Debugf("%v Websocket Orderbook data received",
				k.GetName())
		}
		k.wsProcessOrderBook(&channelData, response[1])
	case krakenWsSpread:
		if k.Verbose {
			log.Debugf("%v Websocket Spread data received",
				k.GetName())
		}
		k.wsProcessSpread(&channelData, response[1])
	case krakenWsTrade:
		if k.Verbose {
			log.Debugf("%v Websocket Trade data received",
				k.GetName())
		}
		k.wsProcessTrades(&channelData, response[1])
	default:
		log.Errorf("%v Unidentified websocket data received: %v",
			k.GetName(), response)
	}
}

// WsHandleEventResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) WsHandleEventResponse(response *WebsocketEventResponse) {
	switch response.Event {
	case krakenWsHeartbeat:
		if k.Verbose {
			log.Debugf("%v Websocket heartbeat data received",
				k.GetName())
		}
	case krakenWsPong:
		if k.Verbose {
			log.Debugf("%v Websocket pong data received",
				k.GetName())
		}
	case krakenWsSystemStatus:
		if k.Verbose {
			log.Debugf("%v Websocket status data received",
				k.GetName())
		}
		if response.Status != "online" {
			k.Websocket.DataHandler <- fmt.Errorf("%v Websocket status '%v'",
				k.GetName(), response.Status)
		}
		if response.WebsocketStatusResponse.Version != krakenWSSupportedVersion {
			log.Warnf("%v New version of Websocket API released. Was %v Now %v",
				k.GetName(), krakenWSSupportedVersion, response.WebsocketStatusResponse.Version)
		}
	case krakenWsSubscriptionStatus:
		if k.Verbose {
			log.Debugf("%v Websocket subscription status data received",
				k.GetName())
		}
		if response.Status != "subscribed" {
			if response.RequestID > 0 {
				k.Websocket.DataHandler <- fmt.Errorf("requestID: '%v'. Error: %v", response.RequestID, response.WebsocketErrorResponse.ErrorMessage)
			} else {
				k.Websocket.DataHandler <- fmt.Errorf(response.WebsocketErrorResponse.ErrorMessage)
			}
			return
		}
		addNewSubscriptionChannelData(response)
	default:
		log.Errorf("%v Unidentified websocket data received: %v", k.GetName(), response)
	}
}

// WsSubscribeToChannel sends a request to WS to subscribe to supplied channel name and pairs
func (k *Kraken) WsSubscribeToChannel(topic string, currencies []string, requestID int64) error {
	resp := WebsocketSubscriptionEventRequest{
		Event: krakenWsSubscribe,
		Pairs: currencies,
		Subscription: WebsocketSubscriptionData{
			Name: topic,
		},
	}
	if requestID > 0 {
		resp.RequestID = requestID
	}
	json, err := common.JSONEncode(resp)
	if err != nil {
		return err
	}
	return k.writeToWebsocket(json)
}

// WsUnsubscribeToChannel sends a request to WS to unsubscribe to supplied channel name and pairs
func (k *Kraken) WsUnsubscribeToChannel(topic string, currencies []string, requestID int64) error {
	resp := WebsocketSubscriptionEventRequest{
		Event: krakenWsUnsubscribe,
		Pairs: currencies,
		Subscription: WebsocketSubscriptionData{
			Name: topic,
		},
	}
	if requestID > 0 {
		resp.RequestID = requestID
	}
	json, err := common.JSONEncode(resp)
	if err != nil {
		return err
	}
	return k.writeToWebsocket(json)
}

// WsUnsubscribeToChannelByChannelID sends a request to WS to unsubscribe to supplied channel ID
func (k *Kraken) WsUnsubscribeToChannelByChannelID(channelID int64) error {
	resp := WebsocketUnsubscribeByChannelIDEventRequest{
		Event:     krakenWsUnsubscribe,
		ChannelID: channelID,
	}
	json, err := common.JSONEncode(resp)
	if err != nil {
		return err
	}
	return k.writeToWebsocket(json)
}

// addNewSubscriptionChannelData stores channel ids, pairs and subscription types to an array
// allowing correlation between subscriptions and returned data
func addNewSubscriptionChannelData(response *WebsocketEventResponse) {
	for i := range subscriptionChannelPair {
		if response.ChannelID == subscriptionChannelPair[i].ChannelID {
			return
		}
	}

	// We change the / to - to maintain compatibility with REST/config
	pair := currency.NewPairWithDelimiter(response.Pair.Base.String(), response.Pair.Quote.String(), "-")
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

// ResubscribeToChannel will attempt to unsubscribe and resubscribe to a channel
func (k *Kraken) ResubscribeToChannel(channel string, pair currency.Pair) {
	// Kraken WS formats pairs with / but config and REST use -
	formattedPair := strings.ToUpper(strings.Replace(pair.String(), "-", "/", 1))
	if krakenWsResubscribeFailureLimit > 0 {
		var successfulUnsubscribe bool
		for i := 0; i < krakenWsResubscribeFailureLimit; i++ {
			err := k.WsUnsubscribeToChannel(channel, []string{formattedPair}, 0)
			if err != nil {
				log.Error(err)
				time.Sleep(krakenWsResubscribeDelayInSeconds * time.Second)
				continue
			}
			successfulUnsubscribe = true
			break
		}
		if !successfulUnsubscribe {
			log.Fatalf("%v websocket channel %v failed to unsubscribe after %v attempts",
				k.GetName(), channel, krakenWsResubscribeFailureLimit)
		}
		successfulSubscribe := true
		for i := 0; i < krakenWsResubscribeFailureLimit; i++ {
			err := k.WsSubscribeToChannel(channel, []string{formattedPair}, 0)
			if err != nil {
				log.Error(err)
				time.Sleep(krakenWsResubscribeDelayInSeconds * time.Second)
				continue
			}
			successfulSubscribe = true
			break
		}
		if !successfulSubscribe {
			log.Fatalf("%v websocket channel %v failed to resubscribe after %v attempts",
				k.GetName(), channel, krakenWsResubscribeFailureLimit)
		}
	} else {
		log.Fatalf("%v websocket channel %v cannot resubscribe. Limit: %v",
			k.GetName(), channel, krakenWsResubscribeFailureLimit)
	}
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (k *Kraken) wsProcessTickers(channelData *WebsocketChannelData, data interface{}) {
	tickerData := data.(map[string]interface{})
	closeData := tickerData["c"].([]interface{})
	openData := tickerData["o"].([]interface{})
	lowData := tickerData["l"].([]interface{})
	highData := tickerData["h"].([]interface{})
	volumeData := tickerData["v"].([]interface{})
	closePrice, _ := strconv.ParseFloat(closeData[0].(string), 64)
	openPrice, _ := strconv.ParseFloat(openData[0].(string), 64)
	highPrice, _ := strconv.ParseFloat(highData[0].(string), 64)
	lowPrice, _ := strconv.ParseFloat(lowData[0].(string), 64)
	quantity, _ := strconv.ParseFloat(volumeData[0].(string), 64)

	k.Websocket.DataHandler <- exchange.TickerData{
		Timestamp:  time.Now(),
		Exchange:   k.GetName(),
		AssetType:  krakenWsAssetType,
		Pair:       channelData.Pair,
		ClosePrice: closePrice,
		OpenPrice:  openPrice,
		HighPrice:  highPrice,
		LowPrice:   lowPrice,
		Quantity:   quantity,
	}
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (k *Kraken) wsProcessSpread(channelData *WebsocketChannelData, data interface{}) {
	spreadData := data.([]interface{})
	bestBid := spreadData[0].(string)
	bestAsk := spreadData[1].(string)
	timeData, _ := strconv.ParseFloat(spreadData[2].(string), 64)
	sec, dec := math.Modf(timeData)
	spreadTimestamp := time.Unix(int64(sec), int64(dec*(1e9)))
	if k.Verbose {
		log.Debugf("Spread data for '%v' received. Best bid: '%v' Best ask: '%v' Time: '%v'",
			channelData.Pair,
			bestBid,
			bestAsk,
			spreadTimestamp)
	}
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (k *Kraken) wsProcessTrades(channelData *WebsocketChannelData, data interface{}) {
	tradeData := data.([]interface{})
	for i := range tradeData {
		trade := tradeData[i].([]interface{})
		timeData, _ := strconv.ParseInt(trade[2].(string), 10, 64)
		timeUnix := time.Unix(timeData, 0)
		price, _ := strconv.ParseFloat(trade[0].(string), 64)
		amount, _ := strconv.ParseFloat(trade[1].(string), 64)

		k.Websocket.DataHandler <- exchange.TradeData{
			AssetType:    krakenWsAssetType,
			CurrencyPair: channelData.Pair,
			EventTime:    time.Now().Unix(),
			Exchange:     k.GetName(),
			Price:        price,
			Amount:       amount,
			Timestamp:    timeUnix,
			Side:         trade[3].(string),
		}
	}
}

// wsProcessOrderBook determines if the orderbook data is partial or update
// Then sends to appropriate fun
func (k *Kraken) wsProcessOrderBook(channelData *WebsocketChannelData, data interface{}) {
	obData := data.(map[string]interface{})
	if _, ok := obData["as"]; ok {
		k.wsProcessOrderBookPartial(channelData, obData)
	} else {
		_, asksExist := obData["a"]
		_, bidsExist := obData["b"]
		if asksExist || bidsExist {
			k.mu.Lock()
			defer k.mu.Unlock()
			k.wsProcessOrderBookBuffer(channelData, obData)
			if len(orderbookBuffer[channelData.ChannelID]) >= orderbookBufferLimit {
				err := k.wsProcessOrderBookUpdate(channelData)
				if err != nil {
					k.ResubscribeToChannel(channelData.Subscription, channelData.Pair)
				}
			}
		}
	}
}

// wsProcessOrderBookPartial creates a new orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookPartial(channelData *WebsocketChannelData, obData map[string]interface{}) {
	ob := orderbook.Base{
		Pair:      channelData.Pair,
		AssetType: krakenWsAssetType,
	}
	// Kraken ob data is timestamped per price, GCT orderbook data is timestamped per entry
	// Using the highest last update time, we can attempt to respect both within a reasonable degree
	var highestLastUpdate time.Time
	askData := obData["as"].([]interface{})
	for i := range askData {
		asks := askData[i].([]interface{})
		price, _ := strconv.ParseFloat(asks[0].(string), 64)
		amount, _ := strconv.ParseFloat(asks[1].(string), 64)
		ob.Asks = append(ob.Asks, orderbook.Item{
			Amount: amount,
			Price:  price,
		})

		timeData, _ := strconv.ParseFloat(asks[2].(string), 64)
		sec, dec := math.Modf(timeData)
		askUpdatedTime := time.Unix(int64(sec), int64(dec*(1e9)))
		if highestLastUpdate.Before(askUpdatedTime) {
			highestLastUpdate = askUpdatedTime
		}
	}

	bidData := obData["bs"].([]interface{})
	for i := range bidData {
		bids := bidData[i].([]interface{})
		price, _ := strconv.ParseFloat(bids[0].(string), 64)
		amount, _ := strconv.ParseFloat(bids[1].(string), 64)
		ob.Bids = append(ob.Bids, orderbook.Item{
			Amount: amount,
			Price:  price,
		})

		timeData, _ := strconv.ParseFloat(bids[2].(string), 64)
		sec, dec := math.Modf(timeData)
		bidUpdateTime := time.Unix(int64(sec), int64(dec*(1e9)))
		if highestLastUpdate.Before(bidUpdateTime) {
			highestLastUpdate = bidUpdateTime
		}
	}

	ob.LastUpdated = highestLastUpdate
	err := k.Websocket.Orderbook.LoadSnapshot(&ob, k.GetName(), true)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	k.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Exchange: k.GetName(),
		Asset:    krakenWsAssetType,
		Pair:     channelData.Pair,
	}

	if krakenOrderBooks == nil {
		krakenOrderBooks = make(map[int64]orderbook.Base)
	}
	krakenOrderBooks[channelData.ChannelID] = ob
}

func (k *Kraken) wsProcessOrderBookBuffer(channelData *WebsocketChannelData, obData map[string]interface{}) {
	ob := orderbook.Base{
		AssetType:    krakenWsAssetType,
		ExchangeName: k.GetName(),
		Pair:         channelData.Pair,
	}

	var highestLastUpdate time.Time
	// Ask data is not always sent
	if _, ok := obData["a"]; ok {
		askData := obData["a"].([]interface{})
		for i := range askData {
			asks := askData[i].([]interface{})
			price, _ := strconv.ParseFloat(asks[0].(string), 64)
			amount, _ := strconv.ParseFloat(asks[1].(string), 64)
			ob.Asks = append(ob.Asks, orderbook.Item{
				Amount: amount,
				Price:  price,
			})

			timeData, _ := strconv.ParseFloat(asks[2].(string), 64)
			sec, dec := math.Modf(timeData)
			askUpdatedTime := time.Unix(int64(sec), int64(dec*(1e9)))
			if highestLastUpdate.Before(askUpdatedTime) {
				highestLastUpdate = askUpdatedTime
			}
		}
	}
	// Bid data is not always sent
	if _, ok := obData["b"]; ok {
		bidData := obData["b"].([]interface{})
		for i := range bidData {
			bids := bidData[i].([]interface{})
			price, _ := strconv.ParseFloat(bids[0].(string), 64)
			amount, _ := strconv.ParseFloat(bids[1].(string), 64)
			ob.Bids = append(ob.Bids, orderbook.Item{
				Amount: amount,
				Price:  price,
			})

			timeData, _ := strconv.ParseFloat(bids[2].(string), 64)
			sec, dec := math.Modf(timeData)
			bidUpdatedTime := time.Unix(int64(sec), int64(dec*(1e9)))
			if highestLastUpdate.Before(bidUpdatedTime) {
				highestLastUpdate = bidUpdatedTime
			}
		}
	}
	ob.LastUpdated = highestLastUpdate
	if orderbookBuffer == nil {
		orderbookBuffer = make(map[int64][]orderbook.Base)
	}
	orderbookBuffer[channelData.ChannelID] = append(orderbookBuffer[channelData.ChannelID], ob)
	if k.Verbose {
		log.Debugf("Adding orderbook to buffer for channel %v. Lastupdated: %v. %v / %v",
			channelData.ChannelID,
			ob.LastUpdated,
			len(orderbookBuffer[channelData.ChannelID]),
			orderbookBufferLimit)
	}
}

// wsProcessOrderBookUpdate updates an orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookUpdate(channelData *WebsocketChannelData) error {
	if k.Verbose {
		log.Debugf("Current orderbook 'LastUpdated': %v",
			krakenOrderBooks[channelData.ChannelID].LastUpdated)
	}
	lowestLastUpdated := orderbookBuffer[channelData.ChannelID][0].LastUpdated
	if k.Verbose {
		log.Debugf("Sorting orderbook. Earliest 'LastUpdated' entry: %v",
			lowestLastUpdated)
	}
	sort.Slice(orderbookBuffer[channelData.ChannelID], func(i, j int) bool {
		return orderbookBuffer[channelData.ChannelID][i].LastUpdated.Before(orderbookBuffer[channelData.ChannelID][j].LastUpdated)
	})

	lowestLastUpdated = orderbookBuffer[channelData.ChannelID][0].LastUpdated
	if k.Verbose {
		log.Debugf("Sorted orderbook. Earliest 'LastUpdated' entry: %v",
			lowestLastUpdated)
	}
	// The earliest update has to be after the previously stored orderbook
	if krakenOrderBooks[channelData.ChannelID].LastUpdated.After(lowestLastUpdated) {
		err := fmt.Errorf("orderbook update out of order. Existing: %v, Attempted: %v",
			krakenOrderBooks[channelData.ChannelID].LastUpdated,
			lowestLastUpdated)
		k.Websocket.DataHandler <- err
		return err
	}

	k.updateChannelOrderbookEntries(channelData)
	highestLastUpdate := orderbookBuffer[channelData.ChannelID][len(orderbookBuffer[channelData.ChannelID])-1].LastUpdated
	if k.Verbose {
		log.Debugf("Saving orderbook. Lastupdated: %v",
			highestLastUpdate)
	}

	ob := krakenOrderBooks[channelData.ChannelID]
	ob.LastUpdated = highestLastUpdate
	err := k.Websocket.Orderbook.LoadSnapshot(&ob, k.GetName(), true)
	if err != nil {
		k.Websocket.DataHandler <- err
		return err
	}

	k.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Exchange: k.GetName(),
		Asset:    krakenWsAssetType,
		Pair:     channelData.Pair,
	}
	// Reset the buffer
	orderbookBuffer[channelData.ChannelID] = []orderbook.Base{}
	return nil
}

func (k *Kraken) updateChannelOrderbookEntries(channelData *WebsocketChannelData) {
	for i := 0; i < len(orderbookBuffer[channelData.ChannelID]); i++ {
		for j := 0; j < len(orderbookBuffer[channelData.ChannelID][i].Asks); j++ {
			k.updateChannelOrderbookAsks(i, j, channelData)
		}
		for j := 0; j < len(orderbookBuffer[channelData.ChannelID][i].Bids); j++ {
			k.updateChannelOrderbookBids(i, j, channelData)
		}
	}
}

func (k *Kraken) updateChannelOrderbookAsks(i, j int, channelData *WebsocketChannelData) {
	askFound := k.updateChannelOrderbookAsk(i, j, channelData)
	if !askFound {
		if k.Verbose {
			log.Debugf("Adding Ask for channel %v. Price %v. Amount %v",
				channelData.ChannelID,
				orderbookBuffer[channelData.ChannelID][i].Asks[j].Price,
				orderbookBuffer[channelData.ChannelID][i].Asks[j].Amount)
		}
		ob := krakenOrderBooks[channelData.ChannelID]
		ob.Asks = append(ob.Asks, orderbookBuffer[channelData.ChannelID][i].Asks[j])
		krakenOrderBooks[channelData.ChannelID] = ob
	}
}

func (k *Kraken) updateChannelOrderbookAsk(i, j int, channelData *WebsocketChannelData) bool {
	askFound := false
	for l := 0; l < len(krakenOrderBooks[channelData.ChannelID].Asks); l++ {
		if krakenOrderBooks[channelData.ChannelID].Asks[l].Price == orderbookBuffer[channelData.ChannelID][i].Asks[j].Price {
			askFound = true
			if orderbookBuffer[channelData.ChannelID][i].Asks[j].Amount == 0 {
				// Remove existing entry
				if k.Verbose {
					log.Debugf("Removing Ask for channel %v. Price %v. Old amount %v. Buffer %v",
						channelData.ChannelID,
						orderbookBuffer[channelData.ChannelID][i].Asks[j].Price,
						krakenOrderBooks[channelData.ChannelID].Asks[l].Amount, i)
				}
				ob := krakenOrderBooks[channelData.ChannelID]
				ob.Asks = append(ob.Asks[:l], ob.Asks[l+1:]...)
				krakenOrderBooks[channelData.ChannelID] = ob
				l--
			} else if krakenOrderBooks[channelData.ChannelID].Asks[l].Amount != orderbookBuffer[channelData.ChannelID][i].Asks[j].Amount {
				if k.Verbose {
					log.Debugf("Updating Ask for channel %v. Price %v. Old amount %v, New Amount %v",
						channelData.ChannelID,
						orderbookBuffer[channelData.ChannelID][i].Asks[j].Price,
						krakenOrderBooks[channelData.ChannelID].Asks[l].Amount,
						orderbookBuffer[channelData.ChannelID][i].Asks[j].Amount)
				}
				krakenOrderBooks[channelData.ChannelID].Asks[l].Amount = orderbookBuffer[channelData.ChannelID][i].Asks[j].Amount
			}
			return askFound
		}
	}
	return askFound
}

func (k *Kraken) updateChannelOrderbookBids(i, j int, channelData *WebsocketChannelData) {
	bidFound := k.updateChannelOrderbookBid(i, j, channelData)
	if !bidFound {
		if k.Verbose {
			log.Debugf("Adding Bid for channel %v. Price %v. Amount %v",
				channelData.ChannelID,
				orderbookBuffer[channelData.ChannelID][i].Bids[j].Price,
				orderbookBuffer[channelData.ChannelID][i].Bids[j].Amount)
		}
		ob := krakenOrderBooks[channelData.ChannelID]
		ob.Bids = append(ob.Bids, orderbookBuffer[channelData.ChannelID][i].Bids[j])
		krakenOrderBooks[channelData.ChannelID] = ob
	}
}

func (k *Kraken) updateChannelOrderbookBid(i, j int, channelData *WebsocketChannelData) bool {
	bidFound := false
	for l := 0; l < len(krakenOrderBooks[channelData.ChannelID].Bids); l++ {
		if krakenOrderBooks[channelData.ChannelID].Bids[l].Price == orderbookBuffer[channelData.ChannelID][i].Bids[j].Price {
			bidFound = true
			if orderbookBuffer[channelData.ChannelID][i].Bids[j].Amount == 0 {
				// Remove existing entry
				if k.Verbose {
					log.Debugf("Removing Bid for channel %v. Price %v. Old amount %v. Buffer %v",
						channelData.ChannelID,
						orderbookBuffer[channelData.ChannelID][i].Bids[j].Price,
						krakenOrderBooks[channelData.ChannelID].Bids[l].Amount, i)
				}
				ob := krakenOrderBooks[channelData.ChannelID]
				ob.Bids = append(ob.Bids[:l], ob.Bids[l+1:]...)
				krakenOrderBooks[channelData.ChannelID] = ob
				l--
			} else if krakenOrderBooks[channelData.ChannelID].Bids[l].Amount != orderbookBuffer[channelData.ChannelID][i].Bids[j].Amount {
				if k.Verbose {
					log.Debugf("Updating Bid for channel %v. Price %v. Old amount %v, New Amount %v",
						channelData.ChannelID,
						orderbookBuffer[channelData.ChannelID][i].Bids[j].Price,
						krakenOrderBooks[channelData.ChannelID].Bids[l].Amount,
						orderbookBuffer[channelData.ChannelID][i].Bids[j].Amount)
				}
				krakenOrderBooks[channelData.ChannelID].Bids[l].Amount = orderbookBuffer[channelData.ChannelID][i].Bids[j].Amount
			}
			return bidFound
		}
	}
	return bidFound
}

// wsProcessCandles converts candle data and sends it to the data handler
func (k *Kraken) wsProcessCandles(channelData *WebsocketChannelData, data interface{}) {
	candleData := data.([]interface{})
	startTimeData, _ := strconv.ParseInt(candleData[0].(string), 10, 64)
	startTimeUnix := time.Unix(startTimeData, 0)
	endTimeData, _ := strconv.ParseInt(candleData[1].(string), 10, 64)
	endTimeUnix := time.Unix(endTimeData, 0)
	openPrice, _ := strconv.ParseFloat(candleData[2].(string), 64)
	highPrice, _ := strconv.ParseFloat(candleData[3].(string), 64)
	lowPrice, _ := strconv.ParseFloat(candleData[4].(string), 64)
	closePrice, _ := strconv.ParseFloat(candleData[5].(string), 64)
	volume, _ := strconv.ParseFloat(candleData[7].(string), 64)

	k.Websocket.DataHandler <- exchange.KlineData{
		AssetType: krakenWsAssetType,
		Pair:      channelData.Pair,
		Timestamp: time.Now(),
		Exchange:  k.GetName(),
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
