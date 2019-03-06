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
				if o.Verbose {
					log.Debugf("WS Data Event: %v Message: %v", dataResponse.Table, dataResponse.Data)
				}
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
	iso := utcTime.String()
	isoBytes := []byte(iso)
	iso = string(isoBytes[:10]) + "T" + string(isoBytes[11:23]) + "Z"
	signPath := "/users/self/verify"
	hmac := common.GetHMAC(common.HashSHA256, []byte(iso+http.MethodGet+signPath), []byte(o.APISecret))
	base64 := common.Base64Encode(hmac)

	resp := WebsocketEventRequest{
		Operation: "login",
		Arguments: []string{o.APIKey, o.ClientID, iso, base64},
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
			log.Debugf("%v Websocket candle data received", o.GetName())
		}
		o.wsProcessFundingFees(response.Data)
	case first.WebsocketOrderBooksData.Checksum != 0:
		if o.Verbose {
			log.Debugf("%v Websocket candle data received", o.GetName())
		}
		o.WsProcessOrderBook(response)
	case first.WebsocketTickerData.Last > 0:
		if o.Verbose {
			log.Debugf("%v Websocket candle data received", o.GetName())
		}
		o.wsProcessTickers(response.Data)
	case first.WebsocketTradeResponse.Price > 0:
		if o.Verbose {
			log.Debugf("%v Websocket candle data received", o.GetName())
		}
		o.wsProcessTrades(response.Data)
	default:
		log.Errorf("%v unknown Websocket data receieved. Please check subscriptions. %v", o.GetName(), response)
	}
}

func (o *OKGroup) wsProcessTickers(tickers []WebsocketDataWrapper) {
	for _, tickerData := range tickers {
		instrument := pair.NewCurrencyPairDelimiter(tickerData.InstrumentID, "-")
		o.Websocket.DataHandler <- exchange.TickerData{
			Timestamp:  tickerData.Timestamp,
			Exchange:   o.GetName(),
			AssetType:  "SPOT",
			HighPrice:  tickerData.High24H,
			LowPrice:   tickerData.Low24H,
			ClosePrice: tickerData.Last,
			Pair:       instrument,
		}
	}
}

func (o *OKGroup) wsProcessTrades(trades []WebsocketDataWrapper) {
	for _, trade := range trades {
		instrument := pair.NewCurrencyPairDelimiter(trade.InstrumentID, "-")
		o.Websocket.DataHandler <- exchange.TradeData{
			Amount:       trade.Qty,
			AssetType:    "SPOT",
			CurrencyPair: instrument,
			EventTime:    time.Now().Unix(),
			Exchange:     o.GetName(),
			Price:        trade.Price,
			Side:         trade.Side,
			Timestamp:    trade.Timestamp,
		}
	}
}

func (o *OKGroup) wsProcessCandles(data WebsocketDataResponse) {
	for _, candle := range data.Data {
		instrument := pair.NewCurrencyPairDelimiter(candle.InstrumentID, "-")
		timeData, err := time.Parse(time.RFC3339Nano, candle.Candle[0])
		parsedInterval := strings.Replace(data.Table, "swap/candle", "", 1)
		parsedInterval = strings.Replace(parsedInterval, "s", "", 1)
		if err != nil {
			log.Warnf("%v Time data could not be parsed: %v", o.GetName(), candle.Candle[0])
		}

		klineData := exchange.KlineData{
			AssetType: "SPOT",
			Pair:      instrument,
			Exchange:  o.GetName(),
			Timestamp: timeData,
			Interval:  parsedInterval,
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
func (o *OKGroup) wsProcessFundingFees(data []WebsocketDataWrapper) {
	// This is not supported yet
	for _, fundingFee := range data {
		log.Infof("funding fee for currency %v rate %v, interest rate %v, time %v",
			fundingFee.InstrumentID,
			fundingFee.WebsocketFundingFeeResponse.FundingRate,
			fundingFee.WebsocketFundingFeeResponse.InterestRate,
			fundingFee.WebsocketFundingFeeResponse.FundingTime)
	}
}

// WsCalculateOrderBookChecksum calculates the orderbook checksum and compares to received value
func (o *OKGroup) WsCalculateOrderBookChecksum(orderbookData WebsocketDataWrapper) int32 {
	var checksum string
	if len(orderbookData.Asks) == len(orderbookData.Bids) {
		if o.Verbose {
			log.Debug("alternating checksum")
		}
		iterations := 25
		for i := 0; i < iterations && i < len(orderbookData.Bids); i++ {
			bidsMessage := fmt.Sprintf("%v:%v", orderbookData.Bids[i][0], orderbookData.Bids[i][1])
			askMessage := fmt.Sprintf("%v:%v", orderbookData.Asks[i][0], orderbookData.Asks[i][1])
			log.Debugf("adding ask %v", i)

			if checksum == "" {
				checksum = fmt.Sprintf("%v:%v", bidsMessage, askMessage)
			} else {
				checksum = fmt.Sprintf("%v:%v:%v", checksum, bidsMessage, askMessage)
			}
		}
	} else {
		if o.Verbose {
			log.Debug("bids first, then asks checksum")
		}
		iterations := 25
		for i := 0; i < iterations; i++ {
			bidsMessage := ""
			askMessage := ""
			if len(orderbookData.Bids)-1 >= i {
				log.Debugf("adding bid %v", i)
				bidsMessage = fmt.Sprintf("%v:%v", orderbookData.Bids[i][0], orderbookData.Bids[i][1])
			}
			if len(orderbookData.Asks)-1 >= i {
				log.Debugf("adding ask %v", i)
				askMessage = fmt.Sprintf("%v:%v", orderbookData.Asks[i][0], orderbookData.Asks[i][1])
			}
			if checksum == "" {
				checksum = fmt.Sprintf("%v:%v", bidsMessage, askMessage)
			} else {
				checksum = fmt.Sprintf("%v:%v:%v", checksum, bidsMessage, askMessage)
			}
		}
		/*for i := 0; i < iterations && i < len(orderbookData.Bids); i++ {
			log.Debugf("adding bid %v", i)
			if len(checksum) == 0 {
				checksum = fmt.Sprintf("%v:%v", orderbookData.Bids[i][0], orderbookData.Bids[i][1])
			} else {
				checksum = fmt.Sprintf("%v:%v:%v", checksum, orderbookData.Bids[i][0], orderbookData.Bids[i][1])
			}
		}
		for i := 0; i < iterations && i < len(orderbookData.Asks); i++ {
			log.Debugf("adding ask %v", i)
			if len(checksum) == 0 {
				checksum = fmt.Sprintf("%v:%v", orderbookData.Asks[i][0], orderbookData.Asks[i][1])
			} else {
				checksum = fmt.Sprintf("%v:%v:%v", checksum, orderbookData.Asks[i][0], orderbookData.Asks[i][1])
			}
		}*/

	}
	return int32(crc32.ChecksumIEEE([]byte(checksum)))
}

// WsValidateOrderBookChecksum ensures that checksum matches and handles subscription status
// If invalid, will unsubscribe and resubscribe to the affected channel
func (o *OKGroup) WsValidateOrderBookChecksum(wsEvent WebsocketDataResponse) bool {
	for i := range wsEvent.Data {
		signedChecksum := o.WsCalculateOrderBookChecksum(wsEvent.Data[i])
		if signedChecksum != wsEvent.Data[i].Checksum {
			log.Warnf("orderbook checksum does not match. Resubscribing")
			o.WsUnsubscribeToChannel(wsEvent.Table)
			o.WsSubscribeToChannel(wsEvent.Table)
			return false
		}
	}
	log.Debug("Passed checksum!")
	return true
}

// WsProcessOrderBook Validates the checksum and updates internal orderbook values
func (o *OKGroup) WsProcessOrderBook(wsEvent WebsocketDataResponse) {
	for i := range wsEvent.Data {
		instrument := pair.NewCurrencyPairDelimiter(wsEvent.Data[i].InstrumentID, "-")
		if wsEvent.Action == "partial" {
			if !o.WsValidateOrderBookChecksum(wsEvent) {
				return
			}
			var asks, bids []orderbook.Item
			for j := range wsEvent.Data[i].Asks {
				amount := wsEvent.Data[i].Asks[j][2].(float64)
				price, _ := strconv.ParseFloat(wsEvent.Data[i].Asks[j][0].(string), 64)
				asks = append(asks, orderbook.Item{
					Amount: amount,
					Price:  price,
				})
			}
			for j := range wsEvent.Data[i].Bids {
				amount := wsEvent.Data[i].Bids[j][2].(float64)
				price, _ := strconv.ParseFloat(wsEvent.Data[i].Bids[j][0].(string), 64)
				bids = append(bids, orderbook.Item{
					Amount: amount,
					Price:  price,
				})
			}
			newOrderBook := orderbook.Base{
				Asks:         asks,
				Bids:         bids,
				AssetType:    "SPOT",
				CurrencyPair: wsEvent.Data[0].InstrumentID,
				LastUpdated:  time.Now(),
				Pair:         pair.NewCurrencyPairDelimiter(wsEvent.Data[i].InstrumentID, "-"),
			}
			err := o.Websocket.Orderbook.LoadSnapshot(newOrderBook, o.GetName(), false)
			if err != nil {
				log.Error(err)
			}
		} else if wsEvent.Action == "update" {
			ob, err := o.GetOrderbookEx(instrument, "SPOT")
			if err != nil {
				log.Error(err)
			}
			var asksToRemove []int
			for k := range wsEvent.Data[i].Asks {
				askUpdated := false
				if askUpdated {
					continue
				}
				for j := range ob.Asks {
					if askUpdated {
						continue
					}
					// Check if it exists in both, update if necessary, then quit
					newAskPrice, _ := strconv.ParseFloat(wsEvent.Data[i].Asks[k][0].(string), 64)
					newAskAmount, _ := strconv.ParseFloat(wsEvent.Data[i].Asks[k][1].(string), 64)
					/*log.Debugf("iteration: %v", j)
					log.Debugf("asklength: %v", len(ob.Asks))
					log.Debugf("ask: %v", ob.Asks[j])
					log.Debugf("new ask price: %v", newAskPrice)*/
					if ob.Asks[j].Price == newAskPrice {
						// Found! Now update quantity
						ob.Asks[j].Amount = newAskAmount
						askUpdated = true
						// If there aren't any more orders, we dont want it anywhere near our precious order book
						if ob.Asks[j].Amount == 0 {
							asksToRemove = append(asksToRemove, j)
						}
						continue
					}
				}
				if !askUpdated {
					newAskPrice, _ := strconv.ParseFloat(wsEvent.Data[i].Asks[k][0].(string), 64)
					newAskAmount, _ := strconv.ParseFloat(wsEvent.Data[i].Asks[k][1].(string), 64)
					ob.Asks = append(ob.Asks, orderbook.Item{
						Amount: newAskAmount,
						Price:  newAskPrice,
					})
				}
			}
			for j := range asksToRemove {
				ob.Asks = append(ob.Asks[:j], ob.Asks[j+1:]...)
			}

			var bidsToRemove []int
			for k := range wsEvent.Data[i].Bids {
				bidUpdated := false
				for j := range ob.Bids {
					if bidUpdated {
						continue
					}
					// Check if it exists in both, update if necessary, then quit
					newBidPrice, _ := strconv.ParseFloat(wsEvent.Data[i].Bids[k][0].(string), 64)
					newBidAmount, _ := strconv.ParseFloat(wsEvent.Data[i].Bids[k][1].(string), 64)
					/*log.Debugf("iteration: %v", j)
					log.Debugf("asklength: %v", len(ob.Bids))
					log.Debugf("ask: %v", ob.Bids[j])
					log.Debugf("new ask price: %v", newBidPrice)*/
					if ob.Bids[j].Price == newBidPrice {
						// Found! Now update quantity
						ob.Bids[j].Amount = newBidAmount
						bidUpdated = true
						// If there aren't any more orders, we dont want it anywhere near our precious order book
						if ob.Bids[j].Amount == 0 {
							bidsToRemove = append(bidsToRemove, j)
						}
					}
				}
				if !bidUpdated {
					newBidPrice, _ := strconv.ParseFloat(wsEvent.Data[i].Bids[k][0].(string), 64)
					newBidAmount, _ := strconv.ParseFloat(wsEvent.Data[i].Bids[k][1].(string), 64)
					ob.Bids = append(ob.Bids, orderbook.Item{
						Amount: newBidAmount,
						Price:  newBidPrice,
					})
				}
			}
			for j := range bidsToRemove {
				ob.Bids = append(ob.Bids[:j], ob.Bids[j+1:]...)
			}
			sort.Slice(ob.Asks, func(i, j int) bool {
				return ob.Asks[i].Price < ob.Asks[j].Price
			})
			sort.Slice(ob.Bids, func(i, j int) bool {
				return ob.Bids[i].Price < ob.Bids[j].Price
			})
			if !o.ValidateTheThing(ob, wsEvent.Data[i].Checksum) {
				log.Warnf("orderbook checksum does not match. Resubscribing")
				o.WsUnsubscribeToChannel(wsEvent.Table)
				o.WsSubscribeToChannel(wsEvent.Table)
				return
			}
			err = o.Websocket.Orderbook.Update(ob.Bids, ob.Asks, instrument, time.Now(), o.GetName(), "SPOT")
			if err != nil {
				log.Error(err)
			}
		}

		o.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
			Exchange: o.GetName(),
			Asset:    "SPOT",
			Pair:     instrument,
		}
	}
}

// DoTheThing calculates the orderbook checksum and compares to received value
func (o *OKGroup) DoTheThing(orderbookData orderbook.Base) int32 {
	var newChecksum string
	if len(orderbookData.Asks) == len(orderbookData.Bids) {
		if o.Verbose {
			log.Debug("alternating checksum")
		}
		iterations := 25
		for i := 0; i < iterations && i < len(orderbookData.Bids); i++ {
			bidsMessage := fmt.Sprintf("%v:%v", orderbookData.Bids[i].Price, orderbookData.Bids[i].Amount)
			askMessage := fmt.Sprintf("%v:%v", orderbookData.Asks[i].Price, orderbookData.Asks[i].Amount)
			log.Debugf("adding bid and ask %v", i)

			if newChecksum == "" {
				newChecksum = fmt.Sprintf("%v:%v", bidsMessage, askMessage)
			} else {
				newChecksum = fmt.Sprintf("%v:%v:%v", newChecksum, bidsMessage, askMessage)
			}
		}
	} else {
		if o.Verbose {
			log.Debug("bids first, then asks checksum")
		}
		iterations := 25
		for i := 0; i < iterations; i++ {
			bidsMessage := ""
			askMessage := ""
			if len(orderbookData.Bids)-1 >= i {
				log.Debugf("adding bid %v", i)
				bidsMessage = fmt.Sprintf("%v:%v", orderbookData.Bids[i].Price, orderbookData.Bids[i].Amount)
			}
			if len(orderbookData.Asks)-1 >= i {
				log.Debugf("adding ask %v", i)
				askMessage = fmt.Sprintf("%v:%v", orderbookData.Asks[i].Price, orderbookData.Asks[i].Amount)
			}
			if newChecksum == "" {
				newChecksum = fmt.Sprintf("%v:%v", bidsMessage, askMessage)
			} else {
				newChecksum = fmt.Sprintf("%v:%v:%v", newChecksum, bidsMessage, askMessage)
			}
		}
		/*for i := 0; i < iterations && i < len(orderbookData.Bids); i++ {
			log.Debugf("adding bid %v", i)
			if len(checksum) == 0 {
				checksum = fmt.Sprintf("%v:%v", orderbookData.Bids[i][0], orderbookData.Bids[i][1])
			} else {
				checksum = fmt.Sprintf("%v:%v:%v", checksum, orderbookData.Bids[i][0], orderbookData.Bids[i][1])
			}
		}
		for i := 0; i < iterations && i < len(orderbookData.Asks); i++ {
			log.Debugf("adding ask %v", i)
			if len(checksum) == 0 {
				checksum = fmt.Sprintf("%v:%v", orderbookData.Asks[i][0], orderbookData.Asks[i][1])
			} else {
				checksum = fmt.Sprintf("%v:%v:%v", checksum, orderbookData.Asks[i][0], orderbookData.Asks[i][1])
			}
		}*/

	}
	return int32(crc32.ChecksumIEEE([]byte(newChecksum)))
}

// ValidateTheThing ensures that checksum matches and handles subscription status
// If invalid, will unsubscribe and resubscribe to the affected channel
func (o *OKGroup) ValidateTheThing(wsEvent orderbook.Base, checksum int32) bool {
	signedChecksum := o.DoTheThing(wsEvent)
	log.Debugf("signed checksum %v. Original: %v", signedChecksum, checksum)
	if signedChecksum != checksum {
		return false
	}
	log.Debug("Passed checksum!")
	return true
}
