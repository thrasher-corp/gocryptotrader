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

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// List of all websocket channels to subscribe to
const (
	// Orderbook events
	okGroupWsOrderbookUpdate  = "update"
	okGroupWsOrderbookPartial = "partial"
	// API subsections
	okGroupWsSwapSubsection    = "swap/"
	okGroupWsIndexSubsection   = "index/"
	okGroupWsFuturesSubsection = "futures/"
	okGroupWsSpotSubsection    = "spot/"
	// Shared API endpoints
	okGroupWsCandle        = "candle"
	okGroupWsCandle60s     = okGroupWsCandle + "60s"
	okGroupWsCandle180s    = okGroupWsCandle + "180s"
	okGroupWsCandle300s    = okGroupWsCandle + "300s"
	okGroupWsCandle900s    = okGroupWsCandle + "900s"
	okGroupWsCandle1800s   = okGroupWsCandle + "1800s"
	okGroupWsCandle3600s   = okGroupWsCandle + "3600s"
	okGroupWsCandle7200s   = okGroupWsCandle + "7200s"
	okGroupWsCandle14400s  = okGroupWsCandle + "14400s"
	okGroupWsCandle21600s  = okGroupWsCandle + "21600"
	okGroupWsCandle43200s  = okGroupWsCandle + "43200s"
	okGroupWsCandle86400s  = okGroupWsCandle + "86400s"
	okGroupWsCandle604900s = okGroupWsCandle + "604800s"
	okGroupWsTicker        = "ticker"
	okGroupWsTrade         = "trade"
	okGroupWsDepth         = "depth"
	okGroupWsDepth5        = "depth5"
	okGroupWsAccount       = "account"
	okGroupWsMarginAccount = "margin_account"
	okGroupWsOrder         = "order"
	okGroupWsFundingRate   = "funding_rate"
	okGroupWsPriceRange    = "price_range"
	okGroupWsMarkPrice     = "mark_price"
	okGroupWsPosition      = "position"
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

var orderbookMutex sync.Mutex
var internalOrderbook orderbook.Base
var secretTimeStampStorage []time.Time

func (o *OKGroup) writeToWebsocket(message string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.Verbose {
		log.Printf("Sending message to WS: %v", message)
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
	log.Printf("Attempting to connect to %v", o.Websocket.GetWebsocketURL())
	o.WebsocketConn, _, err = dialer.Dial(o.Websocket.GetWebsocketURL(),
		http.Header{})
	if err != nil {
		return fmt.Errorf("%s Unable to connect to Websocket. Error: %s",
			o.Name,
			err)
	}
	log.Printf("Successful connection to %v", o.Websocket.GetWebsocketURL())

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
	for _, pair := range o.EnabledPairs {
		formattedPair := strings.ToUpper(strings.Replace(pair, "_", "-", 1))
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
					o.WsHandleDataResponse(dataResponse)
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
	hmac := common.GetHMAC(common.HashSHA256, []byte(fmt.Sprintf("%v", unixTime)+http.MethodGet+signPath), []byte(o.APISecret))
	base64 := common.Base64Encode(hmac)

	resp := WebsocketEventRequest{
		Operation: "login",
		Arguments: []string{o.APIKey, o.ClientID, fmt.Sprintf("%v", unixTime), base64},
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

// WsHandleDataResponse classifies the WS response and sends to appropriate handler
func (o *OKGroup) WsHandleDataResponse(response WebsocketDataResponse) {
	first := response.Data[0]
	switch {
	case len(first.WebsocketCandleResponse.Candle) > 0:
		if o.Verbose {
			log.Debugf("%v Websocket candle data received", o.GetName())
		}
		o.wsProcessCandles(response)
	case first.WebsocketFundingFeeResponse.FundingRate > 0:
		if o.Verbose {
			log.Debugf("%v Websocket funding fee data received", o.GetName())
		}
		o.wsProcessFundingFees(response)
	case first.WebsocketOrderBooksData.Checksum != 0:
		if o.Verbose {
			log.Debugf("%v Websocket orderbook data received", o.GetName())
		}
		// Locking, orderbooks cannot be processed out of order
		orderbookMutex.Lock()
		err := o.WsProcessOrderBook(response)
		if err != nil {
			o.WsUnsubscribeToChannel(response.Table)
			o.WsSubscribeToChannel(response.Table)
		}
		orderbookMutex.Unlock()
	case first.WebsocketTickerData.Last > 0:
		if o.Verbose {
			log.Debugf("%v Websocket ticker data received", o.GetName())
		}
		o.wsProcessTickers(response)
	case first.WebsocketTradeResponse.Price > 0:
		if o.Verbose {
			log.Debugf("%v Websocket trade data received", o.GetName())
		}
		o.wsProcessTrades(response)
	default:
		log.Errorf("%v unknown Websocket data receieved. Please check subscriptions. %v", o.GetName(), response)
	}
}

func (o *OKGroup) wsProcessTickers(data WebsocketDataResponse) {
	for _, tickerData := range data.Data {
		instrument := pair.NewCurrencyPairDelimiter(tickerData.InstrumentID, "-")
		o.Websocket.DataHandler <- exchange.TickerData{
			Timestamp:  tickerData.Timestamp,
			Exchange:   o.GetName(),
			AssetType:  o.GetAssetTypeFromTableName(data.Table),
			HighPrice:  tickerData.High24H,
			LowPrice:   tickerData.Low24H,
			ClosePrice: tickerData.Last,
			Pair:       instrument,
		}
	}
}

func (o *OKGroup) wsProcessTrades(data WebsocketDataResponse) {
	for _, trade := range data.Data {
		instrument := pair.NewCurrencyPairDelimiter(trade.InstrumentID, "-")
		o.Websocket.DataHandler <- exchange.TradeData{
			Amount:       trade.Qty,
			AssetType:    o.GetAssetTypeFromTableName(data.Table),
			CurrencyPair: instrument,
			EventTime:    time.Now().Unix(),
			Exchange:     o.GetName(),
			Price:        trade.Price,
			Side:         trade.Side,
			Timestamp:    trade.Timestamp,
		}
	}
}

func (o *OKGroup) GetAssetTypeFromTableName(table string) string {
	assetIndex := strings.IndexAny(table, "/")
	return strings.ToUpper(table[:assetIndex])
}

func (o *OKGroup) wsProcessCandles(data WebsocketDataResponse) {
	for _, candle := range data.Data {
		instrument := pair.NewCurrencyPairDelimiter(candle.InstrumentID, "-")
		timeData, err := time.Parse(time.RFC3339Nano, candle.Candle[0])
		if err != nil {
			log.Warnf("%v Time data could not be parsed: %v", o.GetName(), candle.Candle[0])
		}
		candleIndex := strings.LastIndex(data.Table, okGroupWsCandle)
		secondIndex := strings.LastIndex(data.Table, "0s")
		candleInterval := ""
		if candleIndex > 0 || secondIndex > 0 {
			candleInterval = data.Table[candleIndex+len(okGroupWsCandle) : secondIndex]
		}

		klineData := exchange.KlineData{
			AssetType: o.GetAssetTypeFromTableName(data.Table),
			Pair:      instrument,
			Exchange:  o.GetName(),
			Timestamp: timeData,
			Interval:  candleInterval,
		}
		klineData.OpenPrice, err = strconv.ParseFloat(candle.Candle[1], 64)
		if err != nil {
			log.Warnf("%v Candle data could not be parsed: %v", o.GetName(), candle.Candle[1])
		}
		klineData.HighPrice, err = strconv.ParseFloat(candle.Candle[2], 64)
		if err != nil {
			log.Warnf("%v Candle data could not be parsed: %v", o.GetName(), candle.Candle[2])
		}
		klineData.LowPrice, err = strconv.ParseFloat(candle.Candle[3], 64)
		if err != nil {
			log.Warnf("%v Candle data could not be parsed: %v", o.GetName(), candle.Candle[3])
		}
		klineData.ClosePrice, err = strconv.ParseFloat(candle.Candle[4], 64)
		if err != nil {
			log.Warnf("%v Candle data could not be parsed: %v", o.GetName(), candle.Candle[4])
		}
		klineData.Volume, err = strconv.ParseFloat(candle.Candle[5], 64)
		if err != nil {
			log.Warnf("%v Candle data could not be parsed: %v", o.GetName(), candle.Candle[5])
		}

		o.Websocket.DataHandler <- klineData
	}
}

// wsProcessFundingFees handles websocket funding fees
// There is currently no handler for this type of information
func (o *OKGroup) wsProcessFundingFees(data WebsocketDataResponse) {
	// This is not supported yet
	for _, fundingFee := range data.Data {
		if o.Verbose {
			log.Infof("funding fee for currency %v rate %v, interest rate %v, time %v",
				fundingFee.InstrumentID,
				fundingFee.WebsocketFundingFeeResponse.FundingRate,
				fundingFee.WebsocketFundingFeeResponse.InterestRate,
				fundingFee.WebsocketFundingFeeResponse.FundingTime)
		}
	}
}

// WsProcessOrderBook Validates the checksum and updates internal orderbook values
func (o *OKGroup) WsProcessOrderBook(wsEvent WebsocketDataResponse) (err error) {
	for i := range wsEvent.Data {
		instrument := pair.NewCurrencyPairDelimiter(wsEvent.Data[i].InstrumentID, "-")
		if wsEvent.Action == okGroupWsOrderbookPartial {
			err = o.WsProcessPartialOrderBook(wsEvent.Data[i], instrument, wsEvent.Table)
		} else if wsEvent.Action == okGroupWsOrderbookUpdate {
			err = o.WsProcessUpdateOrderbook(wsEvent.Data[i], instrument, wsEvent.Table)
		}
	}
	return
}

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

// WsValidateOrderBookChecksum ensures that checksum matches and handles subscription status
func (o *OKGroup) WsValidateOrderBookChecksum(wsEvent WebsocketDataResponse) bool {
	for i := range wsEvent.Data {
		signedChecksum := o.CalculatePartialOrderbookChecksum(wsEvent.Data[i])
		if signedChecksum != wsEvent.Data[i].Checksum {
			log.Errorf("orderbook checksum does not match")
			return false
		}
	}
	if o.Verbose {
		log.Debug("Passed checksum!")
	}
	return true
}

// WsProcessPartialOrderBook takes websocket orderbook data and creates an orderbook
// Calculates checksum to ensure it is valid
func (o *OKGroup) WsProcessPartialOrderBook(wsEventData WebsocketDataWrapper, instrument pair.CurrencyPair, tableName string) error {
	signedChecksum := o.CalculatePartialOrderbookChecksum(wsEventData)
	if signedChecksum != wsEventData.Checksum {
		return errors.New("checksum not valid")
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
		CurrencyPair: wsEventData.InstrumentID,
		LastUpdated:  wsEventData.Timestamp,
		Pair:         instrument,
	}

	err := o.Websocket.Orderbook.LoadSnapshot(newOrderBook, o.GetName(), true)
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
func (o *OKGroup) WsProcessUpdateOrderbook(wsEventData WebsocketDataWrapper, instrument pair.CurrencyPair, tableName string) error {
	var err error
	if internalOrderbook.LastUpdated.IsZero() {
		internalOrderbook, err = o.GetOrderbookEx(instrument, o.GetAssetTypeFromTableName(tableName))
		if err != nil {
			return errors.New("orderbook nil, could not load existing orderbook")
		}
	}
	if internalOrderbook.LastUpdated.After(wsEventData.Timestamp) {
		if o.Verbose {
			log.Errorf("existing: %v, Attempted: %v", internalOrderbook.LastUpdated.Unix(), wsEventData.Timestamp.Unix())
		}
		return errors.New("updated orderbook is older than existing")
	}
	//Update orderbook entries with new data
	internalOrderbook.Asks = o.WsUpdateOrderbookEntry(wsEventData.Asks, internalOrderbook.Asks)
	internalOrderbook.Bids = o.WsUpdateOrderbookEntry(wsEventData.Bids, internalOrderbook.Bids)
	//Validate all the checksums:
	sort.Slice(internalOrderbook.Asks, func(i, j int) bool {
		return internalOrderbook.Asks[i].Price < internalOrderbook.Asks[j].Price
	})
	sort.Slice(internalOrderbook.Bids, func(i, j int) bool {
		return internalOrderbook.Bids[i].Price > internalOrderbook.Bids[j].Price
	})
	// Calculating checksum on sorted orderbook data
	checksum := o.CalculateUpdateOrderbookChecksum(internalOrderbook)
	if checksum == wsEventData.Checksum {
		if o.Verbose {
			log.Debug("Orderbook valid")
		}
		internalOrderbook.LastUpdated = wsEventData.Timestamp
		if o.Verbose {
			log.Debug("Internalising orderbook")
		}

		err := o.Websocket.Orderbook.LoadSnapshot(internalOrderbook, o.GetName(), true)
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
		return errors.New("orderbook update checksum invalid")
	}
	return nil
}

// WsUpdateOrderbookEntry takes WS bid or ask data and merges it with existing orderbook bid or ask data
func (o *OKGroup) WsUpdateOrderbookEntry(wsEntries [][]interface{}, existingOrderbookEntries []orderbook.Item) []orderbook.Item {
	var newRange []orderbook.Item
	for k := range existingOrderbookEntries {
		matched := false
		for j := range wsEntries {
			wsEntryPrice, _ := strconv.ParseFloat(wsEntries[j][0].(string), 64)
			wsEntryAmount, _ := strconv.ParseFloat(wsEntries[j][1].(string), 64)
			// Check if it exists in both, update if necessary, then quit
			if existingOrderbookEntries[k].Price == wsEntryPrice {
				matched = true
				if wsEntryAmount == 0 {
					if o.Verbose {
						log.Debugf("Removing price entry %v from orderbook", wsEntryPrice)
					}
					continue
				}
				newRange = append(newRange, orderbook.Item{
					Amount: wsEntryAmount,
					Price:  wsEntryPrice,
				})
				continue
			}
		}
		if !matched {
			newRange = append(newRange, existingOrderbookEntries[k])
		}
	}
	for j := range wsEntries {
		isWsEntryInExistingOrderBook := false
		wsEntryPrice, _ := strconv.ParseFloat(wsEntries[j][0].(string), 64)
		wsEntryAmount, _ := strconv.ParseFloat(wsEntries[j][1].(string), 64)
		for k := range newRange {
			if newRange[k].Price == wsEntryPrice {
				isWsEntryInExistingOrderBook = true
			}
		}
		if !isWsEntryInExistingOrderBook {
			if wsEntryAmount != 0 {
				if o.Verbose {
					log.Debugf("Adding new price entry %v to orderbook", wsEntryPrice)
				}
				newRange = append(newRange, orderbook.Item{
					Amount: wsEntryAmount,
					Price:  wsEntryPrice,
				})
			}
		}
	}
	return newRange
}

// CalculatePartialOrderbookChecksum alternates over the first 25 bid and ask entries from websocket data
// The checksum is made up of the price and the quantity with a semicolon (:) deliminating them
// This will also work when there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (o *OKGroup) CalculatePartialOrderbookChecksum(orderbookData WebsocketDataWrapper) int32 {
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
func (o *OKGroup) CalculateUpdateOrderbookChecksum(orderbookData orderbook.Base) int32 {
	var checksum string
	iterations := 25
	for i := 0; i < iterations; i++ {
		bidsMessage := ""
		askMessage := ""
		if len(orderbookData.Bids)-1 >= i {
			bidsMessage = fmt.Sprintf("%v:%v:", orderbookData.Bids[i].Price, orderbookData.Bids[i].Amount)
		}
		if len(orderbookData.Asks)-1 >= i {
			askMessage = fmt.Sprintf("%v:%v:", orderbookData.Asks[i].Price, orderbookData.Asks[i].Amount)
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
