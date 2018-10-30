package okgroup

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/exchanges/assets"

	"github.com/thrasher-/gocryptotrader/currency"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// List of all websocket channels to subscribe to
const (
	// If a checksum fails, then resubscribing to the channel fails, fatal after these attempts
	okGroupWsResubscribeFailureLimit   = 3
	okGroupWsResubscribeDelayInSeconds = 3
	// Orderbook events
	okGroupWsOrderbookUpdate  = "update"
	okGroupWsOrderbookPartial = "partial"
	// API subsections
	okGroupWsSwapSubsection    = "swap/"
	okGroupWsIndexSubsection   = "index/"
	okGroupWsFuturesSubsection = "futures/"
	okGroupWsSpotSubsection    = "spot/"
	// Shared API endpoints
	okGroupWsCandle         = "candle"
	okGroupWsCandle60s      = okGroupWsCandle + "60s"
	okGroupWsCandle180s     = okGroupWsCandle + "180s"
	okGroupWsCandle300s     = okGroupWsCandle + "300s"
	okGroupWsCandle900s     = okGroupWsCandle + "900s"
	okGroupWsCandle1800s    = okGroupWsCandle + "1800s"
	okGroupWsCandle3600s    = okGroupWsCandle + "3600s"
	okGroupWsCandle7200s    = okGroupWsCandle + "7200s"
	okGroupWsCandle14400s   = okGroupWsCandle + "14400s"
	okGroupWsCandle21600s   = okGroupWsCandle + "21600"
	okGroupWsCandle43200s   = okGroupWsCandle + "43200s"
	okGroupWsCandle86400s   = okGroupWsCandle + "86400s"
	okGroupWsCandle604900s  = okGroupWsCandle + "604800s"
	okGroupWsTicker         = "ticker"
	okGroupWsTrade          = "trade"
	okGroupWsDepth          = "depth"
	okGroupWsDepth5         = "depth5"
	okGroupWsAccount        = "account"
	okGroupWsMarginAccount  = "margin_account"
	okGroupWsOrder          = "order"
	okGroupWsFundingRate    = "funding_rate"
	okGroupWsPriceRange     = "price_range"
	okGroupWsMarkPrice      = "mark_price"
	okGroupWsPosition       = "position"
	okGroupWsEstimatedPrice = "estimated_price"
	// Spot endpoints
	okGroupWsSpotTicker        = okGroupWsSpotSubsection + okGroupWsTicker
	okGroupWsSpotCandle60s     = okGroupWsSpotSubsection + okGroupWsCandle60s
	okGroupWsSpotCandle180s    = okGroupWsSpotSubsection + okGroupWsCandle180s
	okGroupWsSpotCandle300s    = okGroupWsSpotSubsection + okGroupWsCandle300s
	okGroupWsSpotCandle900s    = okGroupWsSpotSubsection + okGroupWsCandle900s
	okGroupWsSpotCandle1800s   = okGroupWsSpotSubsection + okGroupWsCandle1800s
	okGroupWsSpotCandle3600s   = okGroupWsSpotSubsection + okGroupWsCandle3600s
	okGroupWsSpotCandle7200s   = okGroupWsSpotSubsection + okGroupWsCandle7200s
	okGroupWsSpotCandle14400s  = okGroupWsSpotSubsection + okGroupWsCandle14400s
	okGroupWsSpotCandle21600s  = okGroupWsSpotSubsection + okGroupWsCandle21600s
	okGroupWsSpotCandle43200s  = okGroupWsSpotSubsection + okGroupWsCandle43200s
	okGroupWsSpotCandle86400s  = okGroupWsSpotSubsection + okGroupWsCandle86400s
	okGroupWsSpotCandle604900s = okGroupWsSpotSubsection + okGroupWsCandle604900s
	okGroupWsSpotTrade         = okGroupWsSpotSubsection + okGroupWsTrade
	okGroupWsSpotDepth         = okGroupWsSpotSubsection + okGroupWsDepth
	okGroupWsSpotDepth5        = okGroupWsSpotSubsection + okGroupWsDepth5
	okGroupWsSpotAccount       = okGroupWsSpotSubsection + okGroupWsAccount
	okGroupWsSpotMarginAccount = okGroupWsSpotSubsection + okGroupWsMarginAccount
	okGroupWsSpotOrder         = okGroupWsSpotSubsection + okGroupWsOrder
	// Swap endpoints
	okGroupWsSwapTicker        = okGroupWsSwapSubsection + okGroupWsTicker
	okGroupWsSwapCandle60s     = okGroupWsSwapSubsection + okGroupWsCandle60s
	okGroupWsSwapCandle180s    = okGroupWsSwapSubsection + okGroupWsCandle180s
	okGroupWsSwapCandle300s    = okGroupWsSwapSubsection + okGroupWsCandle300s
	okGroupWsSwapCandle900s    = okGroupWsSwapSubsection + okGroupWsCandle900s
	okGroupWsSwapCandle1800s   = okGroupWsSwapSubsection + okGroupWsCandle1800s
	okGroupWsSwapCandle3600s   = okGroupWsSwapSubsection + okGroupWsCandle3600s
	okGroupWsSwapCandle7200s   = okGroupWsSwapSubsection + okGroupWsCandle7200s
	okGroupWsSwapCandle14400s  = okGroupWsSwapSubsection + okGroupWsCandle14400s
	okGroupWsSwapCandle21600s  = okGroupWsSwapSubsection + okGroupWsCandle21600s
	okGroupWsSwapCandle43200s  = okGroupWsSwapSubsection + okGroupWsCandle43200s
	okGroupWsSwapCandle86400s  = okGroupWsSwapSubsection + okGroupWsCandle86400s
	okGroupWsSwapCandle604900s = okGroupWsSwapSubsection + okGroupWsCandle604900s
	okGroupWsSwapTrade         = okGroupWsSwapSubsection + okGroupWsTrade
	okGroupWsSwapDepth         = okGroupWsSwapSubsection + okGroupWsDepth
	okGroupWsSwapDepth5        = okGroupWsSwapSubsection + okGroupWsDepth5
	okGroupWsSwapFundingRate   = okGroupWsSwapSubsection + okGroupWsFundingRate
	okGroupWsSwapPriceRange    = okGroupWsSwapSubsection + okGroupWsPriceRange
	okGroupWsSwapMarkPrice     = okGroupWsSwapSubsection + okGroupWsMarkPrice
	okGroupWsSwapPosition      = okGroupWsSwapSubsection + okGroupWsPosition
	okGroupWsSwapAccount       = okGroupWsSwapSubsection + okGroupWsAccount
	okGroupWsSwapOrder         = okGroupWsSwapSubsection + okGroupWsOrder
	// Index endpoints
	okGroupWsIndexTicker        = okGroupWsIndexSubsection + okGroupWsTicker
	okGroupWsIndexCandle60s     = okGroupWsIndexSubsection + okGroupWsCandle60s
	okGroupWsIndexCandle180s    = okGroupWsIndexSubsection + okGroupWsCandle180s
	okGroupWsIndexCandle300s    = okGroupWsIndexSubsection + okGroupWsCandle300s
	okGroupWsIndexCandle900s    = okGroupWsIndexSubsection + okGroupWsCandle900s
	okGroupWsIndexCandle1800s   = okGroupWsIndexSubsection + okGroupWsCandle1800s
	okGroupWsIndexCandle3600s   = okGroupWsIndexSubsection + okGroupWsCandle3600s
	okGroupWsIndexCandle7200s   = okGroupWsIndexSubsection + okGroupWsCandle7200s
	okGroupWsIndexCandle14400s  = okGroupWsIndexSubsection + okGroupWsCandle14400s
	okGroupWsIndexCandle21600s  = okGroupWsIndexSubsection + okGroupWsCandle21600s
	okGroupWsIndexCandle43200s  = okGroupWsIndexSubsection + okGroupWsCandle43200s
	okGroupWsIndexCandle86400s  = okGroupWsIndexSubsection + okGroupWsCandle86400s
	okGroupWsIndexCandle604900s = okGroupWsIndexSubsection + okGroupWsCandle604900s
	// Futures endpoints
	okGroupWsFuturesTicker         = okGroupWsFuturesSubsection + okGroupWsTicker
	okGroupWsFuturesCandle60s      = okGroupWsFuturesSubsection + okGroupWsCandle60s
	okGroupWsFuturesCandle180s     = okGroupWsFuturesSubsection + okGroupWsCandle180s
	okGroupWsFuturesCandle300s     = okGroupWsFuturesSubsection + okGroupWsCandle300s
	okGroupWsFuturesCandle900s     = okGroupWsFuturesSubsection + okGroupWsCandle900s
	okGroupWsFuturesCandle1800s    = okGroupWsFuturesSubsection + okGroupWsCandle1800s
	okGroupWsFuturesCandle3600s    = okGroupWsFuturesSubsection + okGroupWsCandle3600s
	okGroupWsFuturesCandle7200s    = okGroupWsFuturesSubsection + okGroupWsCandle7200s
	okGroupWsFuturesCandle14400s   = okGroupWsFuturesSubsection + okGroupWsCandle14400s
	okGroupWsFuturesCandle21600s   = okGroupWsFuturesSubsection + okGroupWsCandle21600s
	okGroupWsFuturesCandle43200s   = okGroupWsFuturesSubsection + okGroupWsCandle43200s
	okGroupWsFuturesCandle86400s   = okGroupWsFuturesSubsection + okGroupWsCandle86400s
	okGroupWsFuturesCandle604900s  = okGroupWsFuturesSubsection + okGroupWsCandle604900s
	okGroupWsFuturesTrade          = okGroupWsFuturesSubsection + okGroupWsTrade
	okGroupWsFuturesEstimatedPrice = okGroupWsFuturesSubsection + okGroupWsTrade
	okGroupWsFuturesPriceRange     = okGroupWsFuturesSubsection + okGroupWsPriceRange
	okGroupWsFuturesDepth          = okGroupWsFuturesSubsection + okGroupWsDepth
	okGroupWsFuturesDepth5         = okGroupWsFuturesSubsection + okGroupWsDepth5
	okGroupWsFuturesMarkPrice      = okGroupWsFuturesSubsection + okGroupWsMarkPrice
	okGroupWsFuturesAccount        = okGroupWsFuturesSubsection + okGroupWsAccount
	okGroupWsFuturesPosition       = okGroupWsFuturesSubsection + okGroupWsPosition
	okGroupWsFuturesOrder          = okGroupWsFuturesSubsection + okGroupWsOrder
)

// orderbookMutex Ensures if two entries arrive at once, only one can be processed at a time
var orderbookMutex sync.Mutex

// writeToWebsocket sends a message to the websocket endpoint
func (o *OKGroup) writeToWebsocket(message string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.Verbose {
		log.Debugf("Sending message to WS: %v", message)
	}
	return o.WebsocketConn.WriteMessage(websocket.TextMessage, []byte(message))
}

// WsConnect initiates a websocket connection
func (o *OKGroup) WsConnect() error {
	if !o.Websocket.IsEnabled() || !o.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer

	if o.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(o.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	if o.Verbose {
		log.Debugf("Attempting to connect to %v", o.Websocket.GetWebsocketURL())
	}
	o.WebsocketConn, _, err = dialer.Dial(o.Websocket.GetWebsocketURL(),
		http.Header{})
	if err != nil {
		return fmt.Errorf("%s Unable to connect to Websocket. Error: %s",
			o.Name,
			err)
	}
	if o.Verbose {
		log.Debugf("Successful connection to %v", o.Websocket.GetWebsocketURL())
	}

	go o.WsHandleData()
	go o.wsPingHandler()

	err = o.WsSubscribeToDefaults()
	if err != nil {
		return fmt.Errorf("error: Could not subscribe to the OKEX websocket %s",
			err)
	}
	return nil
}

// WsSubscribeToDefaults subscribes to the websocket channels
func (o *OKGroup) WsSubscribeToDefaults() (err error) {
	channelsToSubscribe := []string{okGroupWsSpotDepth, okGroupWsSpotCandle300s, okGroupWsSpotTicker, okGroupWsSpotTrade}
	for _, pair := range o.GetEnabledPairs(assets.AssetTypeSpot) {
		formattedPair := strings.ToUpper(strings.Replace(pair.String(), "_", "-", 1))
		for _, channel := range channelsToSubscribe {
			err = o.WsSubscribeToChannel(fmt.Sprintf("%v:%s", channel, formattedPair))
			if err != nil {
				return
			}
		}
	}

	return nil
}

// WsReadData reads data from the websocket connection
func (o *OKGroup) WsReadData() (exchange.WebsocketResponse, error) {
	mType, resp, err := o.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	o.Websocket.TrafficAlert <- struct{}{}
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
	if o.Verbose {
		log.Debugf("%v Websocket message received: %v", o.Name, string(standardMessage))
	}

	return exchange.WebsocketResponse{Raw: standardMessage}, nil
}

// wsPingHandler sends a message "ping" every 27 to maintain the connection to the websocket
func (o *OKGroup) wsPingHandler() {
	o.Websocket.Wg.Add(1)
	defer o.Websocket.Wg.Done()

	ticker := time.NewTicker(time.Second * 27)

	for {
		select {
		case <-o.Websocket.ShutdownC:
			return

		case <-ticker.C:
			err := o.writeToWebsocket("ping")
			if o.Verbose {
				log.Debugf("%v sending ping", o.GetName())
			}
			if err != nil {
				o.Websocket.DataHandler <- err
				return
			}
		}
	}
}

// WsHandleData handles the read data from the websocket connection
func (o *OKGroup) WsHandleData() {
	o.Websocket.Wg.Add(1)
	defer func() {
		err := o.WebsocketConn.Close()
		if err != nil {
			o.Websocket.DataHandler <- fmt.Errorf("okex_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		o.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-o.Websocket.ShutdownC:
			return
		default:
			resp, err := o.WsReadData()
			if err != nil {
				o.Websocket.DataHandler <- err
				return
			}
			var dataResponse WebsocketDataResponse
			err = common.JSONDecode(resp.Raw, &dataResponse)
			if err == nil && dataResponse.Table != "" {
				if len(dataResponse.Data) > 0 {
					o.WsHandleDataResponse(&dataResponse)
				}
				continue
			}
			var errorResponse WebsocketErrorResponse
			err = common.JSONDecode(resp.Raw, &errorResponse)
			if err == nil && errorResponse.ErrorCode > 0 {
				if o.Verbose {
					log.Debugf("WS Error Event: %v Message: %v", errorResponse.Event, errorResponse.Message)
				}
				o.WsHandleErrorResponse(errorResponse)
				continue
			}
			var eventResponse WebsocketEventResponse
			err = common.JSONDecode(resp.Raw, &eventResponse)
			if err == nil && len(eventResponse.Channel) > 0 {
				if o.Verbose {
					log.Debugf("WS Event: %v on Channel: %v", eventResponse.Event, eventResponse.Channel)
				}
				continue
			}
			o.Websocket.DataHandler <- fmt.Errorf("unrecognised response: %v", resp.Raw)
			continue
		}
	}
}

// WsSubscribeToChannel sends a request to WS to subscribe to supplied channel
func (o *OKGroup) WsSubscribeToChannel(topic string) error {
	resp := WebsocketEventRequest{
		Operation: "subscribe",
		Arguments: []string{topic},
	}
	json, err := common.JSONEncode(resp)
	if err != nil {
		return err
	}
	err = o.writeToWebsocket(string(json))
	if err != nil {
		return err
	}
	return nil
}

// WsUnsubscribeToChannel sends a request to WS to unsubscribe to supplied channel
func (o *OKGroup) WsUnsubscribeToChannel(topic string) error {
	resp := WebsocketEventRequest{
		Operation: "unsubscribe",
		Arguments: []string{topic},
	}
	json, err := common.JSONEncode(resp)
	if err != nil {
		return err
	}
	err = o.writeToWebsocket(string(json))
	if err != nil {
		return err
	}
	return nil
}

// WsLogin sends a login request to websocket to enable access to authenticated endpoints
func (o *OKGroup) WsLogin() error {
	utcTime := time.Now().UTC()
	unixTime := utcTime.Unix()
	signPath := "/users/self/verify"
	hmac := common.GetHMAC(common.HashSHA256, []byte(fmt.Sprintf("%v", unixTime)+http.MethodGet+signPath), []byte(o.API.Credentials.Secret))
	base64 := common.Base64Encode(hmac)
	resp := WebsocketEventRequest{
		Operation: "login",
		Arguments: []string{o.API.Credentials.Key, o.API.Credentials.ClientID, fmt.Sprintf("%v", unixTime), base64},
	}
	json, err := common.JSONEncode(resp)
	if err != nil {
		return err
	}
	err = o.writeToWebsocket(string(json))
	if err != nil {
		return err
	}
	return nil
}

// WsHandleErrorResponse sends an error message to ws handler
func (o *OKGroup) WsHandleErrorResponse(event WebsocketErrorResponse) {
	errorMessage := fmt.Sprintf("%v error - %v message: %s ",
		o.GetName(), event.ErrorCode, event.Message)
	if o.Verbose {
		log.Error(errorMessage)
	}
	o.Websocket.DataHandler <- fmt.Errorf(errorMessage)
}

// GetWsChannelWithoutOrderType takes WebsocketDataResponse.Table and returns
// The base channel name eg receive "spot/depth5:BTC-USDT" return "depth5"
func (o *OKGroup) GetWsChannelWithoutOrderType(table string) string {
	index := strings.Index(table, "/")
	if index == -1 {
		return table
	}
	channel := table[index+1:]
	index = strings.Index(channel, ":")
	// Some events do not contain a currency
	if index == -1 {
		return channel
	}

	return channel[:index]
}

// GetAssetTypeFromTableName gets the asset type from the table name
// eg "spot/ticker:BTCUSD" results in "SPOT"
func (o *OKGroup) GetAssetTypeFromTableName(table string) assets.AssetType {
	assetIndex := strings.Index(table, "/")
	return assets.AssetType(strings.ToUpper(table[:assetIndex]))
}

// WsHandleDataResponse classifies the WS response and sends to appropriate handler
func (o *OKGroup) WsHandleDataResponse(response *WebsocketDataResponse) {
	switch o.GetWsChannelWithoutOrderType(response.Table) {
	case okGroupWsCandle60s, okGroupWsCandle180s, okGroupWsCandle300s, okGroupWsCandle900s,
		okGroupWsCandle1800s, okGroupWsCandle3600s, okGroupWsCandle7200s, okGroupWsCandle14400s,
		okGroupWsCandle21600s, okGroupWsCandle43200s, okGroupWsCandle86400s, okGroupWsCandle604900s:
		if o.Verbose {
			log.Debugf("%v Websocket candle data received", o.GetName())
		}
		o.wsProcessCandles(response)
	case okGroupWsDepth, okGroupWsDepth5:
		if o.Verbose {
			log.Debugf("%v Websocket orderbook data received", o.GetName())
		}
		// Locking, orderbooks cannot be processed out of order
		orderbookMutex.Lock()
		err := o.WsProcessOrderBook(response)
		if err != nil {
			log.Error(err)
			subscriptionChannel := fmt.Sprintf("%v:%v", response.Table, response.Data[0].InstrumentID)
			o.ResubscribeToChannel(subscriptionChannel)
		}
		orderbookMutex.Unlock()
	case okGroupWsTicker:
		if o.Verbose {
			log.Debugf("%v Websocket ticker data received", o.GetName())
		}
		o.wsProcessTickers(response)
	case okGroupWsTrade:
		if o.Verbose {
			log.Debugf("%v Websocket trade data received", o.GetName())
		}
		o.wsProcessTrades(response)
	default:
		logDataResponse(response)
	}
}

// ResubscribeToChannel will attempt to unsubscribe and resubscribe to a channel
func (o *OKGroup) ResubscribeToChannel(channel string) {
	if okGroupWsResubscribeFailureLimit > 0 {
		var successfulUnsubscribe bool
		for i := 0; i < okGroupWsResubscribeFailureLimit; i++ {
			err := o.WsUnsubscribeToChannel(channel)
			if err != nil {
				log.Error(err)
				time.Sleep(okGroupWsResubscribeDelayInSeconds * time.Second)
				continue
			}
			successfulUnsubscribe = true
			break
		}
		if !successfulUnsubscribe {
			log.Fatalf("%v websocket channel %v failed to unsubscribe after %v attempts", o.GetName(), channel, okGroupWsResubscribeFailureLimit)
		}
		successfulSubscribe := true
		for i := 0; i < okGroupWsResubscribeFailureLimit; i++ {
			err := o.WsSubscribeToChannel(channel)
			if err != nil {
				log.Error(err)
				time.Sleep(okGroupWsResubscribeDelayInSeconds * time.Second)
				continue
			}
			successfulSubscribe = true
			break
		}
		if !successfulSubscribe {
			log.Fatalf("%v websocket channel %v failed to resubscribe after %v attempts", o.GetName(), channel, okGroupWsResubscribeFailureLimit)
		}
	} else {
		log.Fatalf("%v websocket channel %v cannot resubscribe. Limit: %v", o.GetName(), channel, okGroupWsResubscribeFailureLimit)
	}
}

// logDataResponse will log the details of any websocket data event
// where there is no websocket datahandler for it
func logDataResponse(response *WebsocketDataResponse) {
	for i := range response.Data {
		log.Errorf("Unhandled channel: '%v'. Instrument '%v' Timestamp '%v', Data '%v",
			response.Table,
			response.Data[i].InstrumentID,
			response.Data[i].Timestamp,
			response.Data[i])
	}
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (o *OKGroup) wsProcessTickers(response *WebsocketDataResponse) {
	for i := range response.Data {
		instrument := currency.NewPairDelimiter(response.Data[i].InstrumentID, "-")
		o.Websocket.DataHandler <- exchange.TickerData{
			Timestamp:  response.Data[i].Timestamp,
			Exchange:   o.GetName(),
			AssetType:  o.GetAssetTypeFromTableName(response.Table),
			HighPrice:  response.Data[i].High24H,
			LowPrice:   response.Data[i].Low24H,
			ClosePrice: response.Data[i].Last,
			Pair:       instrument,
		}
	}
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (o *OKGroup) wsProcessTrades(response *WebsocketDataResponse) {
	for i := range response.Data {
		instrument := currency.NewPairDelimiter(response.Data[i].InstrumentID, "-")
		o.Websocket.DataHandler <- exchange.TradeData{
			Amount:       response.Data[i].Qty,
			AssetType:    o.GetAssetTypeFromTableName(response.Table),
			CurrencyPair: instrument,
			EventTime:    time.Now().Unix(),
			Exchange:     o.GetName(),
			Price:        response.Data[i].WebsocketTradeResponse.Price,
			Side:         response.Data[i].Side,
			Timestamp:    response.Data[i].Timestamp,
		}
	}
}

// wsProcessCandles converts candle data and sends it to the data handler
func (o *OKGroup) wsProcessCandles(response *WebsocketDataResponse) {
	for i := range response.Data {
		instrument := currency.NewPairDelimiter(response.Data[i].InstrumentID, "-")
		timeData, err := time.Parse(time.RFC3339Nano, response.Data[i].WebsocketCandleResponse.Candle[0])
		if err != nil {
			log.Warnf("%v Time data could not be parsed: %v", o.GetName(), response.Data[i].Candle[0])
		}

		candleIndex := strings.LastIndex(response.Table, okGroupWsCandle)
		secondIndex := strings.LastIndex(response.Table, "0s")
		candleInterval := ""
		if candleIndex > 0 || secondIndex > 0 {
			candleInterval = response.Table[candleIndex+len(okGroupWsCandle) : secondIndex]
		}

		klineData := exchange.KlineData{
			AssetType: o.GetAssetTypeFromTableName(response.Table),
			Pair:      instrument,
			Exchange:  o.GetName(),
			Timestamp: timeData,
			Interval:  candleInterval,
		}
		klineData.OpenPrice, _ = strconv.ParseFloat(response.Data[i].Candle[1], 64)
		klineData.HighPrice, _ = strconv.ParseFloat(response.Data[i].Candle[2], 64)
		klineData.LowPrice, _ = strconv.ParseFloat(response.Data[i].Candle[3], 64)
		klineData.ClosePrice, _ = strconv.ParseFloat(response.Data[i].Candle[4], 64)
		klineData.Volume, _ = strconv.ParseFloat(response.Data[i].Candle[5], 64)

		o.Websocket.DataHandler <- klineData
	}
}

// WsProcessOrderBook Validates the checksum and updates internal orderbook values
func (o *OKGroup) WsProcessOrderBook(response *WebsocketDataResponse) (err error) {
	for i := range response.Data {
		instrument := currency.NewPairDelimiter(response.Data[i].InstrumentID, "-")
		if response.Action == okGroupWsOrderbookPartial {
			err = o.WsProcessPartialOrderBook(&response.Data[i], instrument, response.Table)
		} else if response.Action == okGroupWsOrderbookUpdate {
			err = o.WsProcessUpdateOrderbook(&response.Data[i], instrument, response.Table)
		}
	}
	return
}

// AppendWsOrderbookItems adds websocket orderbook data bid/asks into an orderbook item array
func (o *OKGroup) AppendWsOrderbookItems(entries [][]interface{}) (orderbookItems []orderbook.Item) {
	for j := range entries {
		amount, _ := strconv.ParseFloat(entries[j][1].(string), 64)
		price, _ := strconv.ParseFloat(entries[j][0].(string), 64)
		orderbookItems = append(orderbookItems, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
	}
	return
}

// WsProcessPartialOrderBook takes websocket orderbook data and creates an orderbook
// Calculates checksum to ensure it is valid
func (o *OKGroup) WsProcessPartialOrderBook(wsEventData *WebsocketDataWrapper, instrument currency.Pair, tableName string) error {
	signedChecksum := o.CalculatePartialOrderbookChecksum(wsEventData)
	if signedChecksum != wsEventData.Checksum {
		return fmt.Errorf("channel: %v. Orderbook partial for %v checksum invalid", tableName, instrument)
	}
	if o.Verbose {
		log.Debug("Passed checksum!")
	}
	asks := o.AppendWsOrderbookItems(wsEventData.Asks)
	bids := o.AppendWsOrderbookItems(wsEventData.Bids)
	newOrderBook := orderbook.Base{
		Asks:         asks,
		Bids:         bids,
		AssetType:    o.GetAssetTypeFromTableName(tableName),
		LastUpdated:  wsEventData.Timestamp,
		Pair:         instrument,
		ExchangeName: o.GetName(),
	}

	err := o.Websocket.Orderbook.LoadSnapshot(&newOrderBook, o.GetName(), true)
	if err != nil {
		return err
	}
	o.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Exchange: o.GetName(),
		Asset:    o.GetAssetTypeFromTableName(tableName),
		Pair:     instrument,
	}
	return nil
}

// WsProcessUpdateOrderbook updates an existing orderbook using websocket data
// After merging WS data, it will sort, validate and finally update the existing orderbook
func (o *OKGroup) WsProcessUpdateOrderbook(wsEventData *WebsocketDataWrapper, instrument currency.Pair, tableName string) error {
	internalOrderbook, err := o.FetchOrderbook(instrument, o.GetAssetTypeFromTableName(tableName))
	if err != nil {
		return errors.New("orderbook nil, could not load existing orderbook")
	}
	if internalOrderbook.LastUpdated.After(wsEventData.Timestamp) {
		if o.Verbose {
			log.Errorf("Orderbook update out of order. Existing: %v, Attempted: %v", internalOrderbook.LastUpdated.Unix(), wsEventData.Timestamp.Unix())
		}
		return errors.New("updated orderbook is older than existing")
	}
	internalOrderbook.Asks = o.WsUpdateOrderbookEntry(wsEventData.Asks, internalOrderbook.Asks)
	internalOrderbook.Bids = o.WsUpdateOrderbookEntry(wsEventData.Bids, internalOrderbook.Bids)
	sort.Slice(internalOrderbook.Asks, func(i, j int) bool {
		return internalOrderbook.Asks[i].Price < internalOrderbook.Asks[j].Price
	})
	sort.Slice(internalOrderbook.Bids, func(i, j int) bool {
		return internalOrderbook.Bids[i].Price > internalOrderbook.Bids[j].Price
	})
	checksum := o.CalculateUpdateOrderbookChecksum(&internalOrderbook)
	if checksum == wsEventData.Checksum {
		if o.Verbose {
			log.Debug("Orderbook valid")
		}
		internalOrderbook.LastUpdated = wsEventData.Timestamp
		if o.Verbose {
			log.Debug("Internalising orderbook")
		}

		err := o.Websocket.Orderbook.LoadSnapshot(&internalOrderbook, o.GetName(), true)
		if err != nil {
			log.Error(err)
		}
		o.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
			Exchange: o.GetName(),
			Asset:    o.GetAssetTypeFromTableName(tableName),
			Pair:     instrument,
		}
	} else {
		if o.Verbose {
			log.Debug("Orderbook invalid")
		}
		return fmt.Errorf("channel: %v. Orderbook update for %v checksum invalid. Received %v Calculated %v", tableName, instrument, wsEventData.Checksum, checksum)
	}
	return nil
}

// WsUpdateOrderbookEntry takes WS bid or ask data and merges it with existing orderbook bid or ask data
func (o *OKGroup) WsUpdateOrderbookEntry(wsEntries [][]interface{}, existingOrderbookEntries []orderbook.Item) []orderbook.Item {
	for j := range wsEntries {
		wsEntryPrice, _ := strconv.ParseFloat(wsEntries[j][0].(string), 64)
		wsEntryAmount, _ := strconv.ParseFloat(wsEntries[j][1].(string), 64)
		matchFound := false
		for k := 0; k < len(existingOrderbookEntries); k++ {
			if existingOrderbookEntries[k].Price != wsEntryPrice {
				continue
			}
			matchFound = true
			if wsEntryAmount == 0 {
				existingOrderbookEntries = append(existingOrderbookEntries[:k], existingOrderbookEntries[k+1:]...)
				k--
				continue
			}
			existingOrderbookEntries[k].Amount = wsEntryAmount
			continue
		}
		if !matchFound {
			existingOrderbookEntries = append(existingOrderbookEntries, orderbook.Item{
				Amount: wsEntryAmount,
				Price:  wsEntryPrice,
			})
		}
	}
	return existingOrderbookEntries
}

// CalculatePartialOrderbookChecksum alternates over the first 25 bid and ask entries from websocket data
// The checksum is made up of the price and the quantity with a semicolon (:) deliminating them
// This will also work when there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (o *OKGroup) CalculatePartialOrderbookChecksum(orderbookData *WebsocketDataWrapper) int32 {
	var checksum string
	iterations := 25
	for i := 0; i < iterations; i++ {
		bidsMessage := ""
		askMessage := ""
		if len(orderbookData.Bids)-1 >= i {
			bidsMessage = fmt.Sprintf("%v:%v:", orderbookData.Bids[i][0], orderbookData.Bids[i][1])
		}
		if len(orderbookData.Asks)-1 >= i {
			askMessage = fmt.Sprintf("%v:%v:", orderbookData.Asks[i][0], orderbookData.Asks[i][1])

		}
		if checksum == "" {
			checksum = fmt.Sprintf("%v%v", bidsMessage, askMessage)
		} else {
			checksum = fmt.Sprintf("%v%v%v", checksum, bidsMessage, askMessage)
		}
	}
	checksum = strings.TrimSuffix(checksum, ":")
	return int32(crc32.ChecksumIEEE([]byte(checksum)))
}

// CalculateUpdateOrderbookChecksum alternates over the first 25 bid and ask entries of a merged orderbook
// The checksum is made up of the price and the quantity with a semicolon (:) deliminating them
// This will also work when there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (o *OKGroup) CalculateUpdateOrderbookChecksum(orderbookData *orderbook.Base) int32 {
	var checksum string
	iterations := 25
	for i := 0; i < iterations; i++ {
		bidsMessage := ""
		askMessage := ""
		if len(orderbookData.Bids)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Bids[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Bids[i].Amount, 'f', -1, 64)
			bidsMessage = fmt.Sprintf("%v:%v:", price, amount)
		}
		if len(orderbookData.Asks)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Asks[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Asks[i].Amount, 'f', -1, 64)
			askMessage = fmt.Sprintf("%v:%v:", price, amount)
		}
		if checksum == "" {
			checksum = fmt.Sprintf("%v%v", bidsMessage, askMessage)
		} else {
			checksum = fmt.Sprintf("%v%v%v", checksum, bidsMessage, askMessage)
		}
	}
	checksum = strings.TrimSuffix(checksum, ":")
	return int32(crc32.ChecksumIEEE([]byte(checksum)))
}
