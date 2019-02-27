package okgroup

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

	go o.WsHandleData()
	go o.wsPingHandler()

	err = o.WsSubscribeToDefaults()
	if err != nil {
		return fmt.Errorf("Error: Could not subscribe to the OKEX websocket %s",
			err)
	}
	return nil
}

// WsSubscribeToDefaults subscribes to the websocket channels
func (o *OKGroup) WsSubscribeToDefaults() (err error) {
	channelsToSubscribe := []string{"spot/ticker", "spot/depth", "spot/trade", "spot/candle60s"}
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
		log.Debugf("%v", string(standardMessage))
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
			if err != nil {
				o.Websocket.DataHandler <- err
				return
			}
		}
	}
}

// WsHandleErrorResponse sends an error message to ws handler
func (o *OKGroup) WsHandleErrorResponse(event WebsocketErrorResponse) {
	o.Websocket.DataHandler <- fmt.Errorf("%v error - %v message: %s ",
		o.GetName(), event.ErrorCode, event.Message)
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
			var eventResponse WebsocketEventResponse
			var dataResponse WebsocketDataResponse
			var errorResponse WebsocketErrorResponse
			// Determine what kind of message was returned
			err = common.JSONDecode(resp.Raw, &dataResponse)
			if err == nil && len(dataResponse.Table) > 0 {
				o.WsHandleDataResponse(dataResponse)
				continue
			}
			err = common.JSONDecode(resp.Raw, &errorResponse)
			if err == nil && errorResponse.ErrorCode > 0 {
				o.WsHandleErrorResponse(errorResponse)
				continue
			}
			err = common.JSONDecode(resp.Raw, &eventResponse)
			if err == nil && len(eventResponse.Channel) > 0 {
				if o.Verbose { // Should we need it to be verbose?
					log.Debugf("WS Event: %v on Channel: %v", eventResponse.Event, eventResponse.Channel)
				}
				continue
			}
			if strings.Contains(string(resp.Raw), "pong") {
				continue
			} else {
				o.Websocket.DataHandler <- fmt.Errorf("Unrecognised response: %v", resp.Raw)
				continue
			}
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

// WsHandleDataResponse sends an error message to ws handler
func (o *OKGroup) WsHandleDataResponse(event WebsocketDataResponse) {
	if len(event.Data) > 0 {
		switch event.Data[0].(type) {
		case WebsocketTickerResponse:
			o.wsProcessTickers(event.Data)
		case WebsocketTradeResponse:
			o.wsProcessTrades(event.Data)
		case WebsocketCandleResponse:
			o.wsProcessCandles(event.Data, event.Table)
		case WebsocketFundingFeeResponse:
			o.wsProcessFundingFees(event.Data)
		case WebsocketOrderBooksResponse:
			o.wsProcessOrderBook(event.Data)
		}
	}
}

func (o *OKGroup) wsProcessTickers(tickers []interface{}) {
	for _, tickerInterface := range tickers {
		tickerData := tickerInterface.(WebsocketTickerResponse)
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

func (o *OKGroup) wsProcessTrades(trades []interface{}) {
	for _, tradeInterface := range trades {
		tradeData := tradeInterface.(WebsocketTradeResponse)
		instrument := pair.NewCurrencyPairDelimiter(tradeData.InstrumentID, "-")
		o.Websocket.DataHandler <- exchange.TradeData{
			Amount:       tradeData.Qty,
			AssetType:    "SPOT",
			CurrencyPair: instrument,
			EventTime:    time.Now().Unix(),
			Exchange:     o.GetName(),
			Price:        tradeData.Price,
			Side:         tradeData.Side,
			Timestamp:    tradeData.Timestamp,
		}
	}
}

func (o *OKGroup) wsProcessCandles(candles []interface{}, interval string) {
	for _, candleInterface := range candles {
		candleData := candleInterface.(WebsocketCandleResponse)
		instrument := pair.NewCurrencyPairDelimiter(candleData.InstrumentID, "-")
		timeData, err := time.Parse(time.RFC3339Nano, candleData.Candle[0])
		parsedInterval := strings.Replace(interval, "swap/candle", "", 1)
		parsedInterval = strings.Replace(parsedInterval, "s", "", 1)
		if err != nil {
			log.Warnf("%v Time data could not be parsed: %v", o.GetName(), candleData.Candle[0])
		}
		klineData := exchange.KlineData{
			AssetType: "SPOT",
			Pair:      instrument,
			Exchange:  o.GetName(),
			Timestamp: timeData,
			Interval:  parsedInterval,
		}
		klineData.OpenPrice, err = strconv.ParseFloat(candleData.Candle[1], 64)
		if err != nil {
			log.Warnf("%v Candle data could not be parsed: %v", o.GetName(), candleData.Candle[1])
		}
		klineData.HighPrice, err = strconv.ParseFloat(candleData.Candle[2], 64)
		if err != nil {
			log.Warnf("%v Candle data could not be parsed: %v", o.GetName(), candleData.Candle[2])
		}
		klineData.LowPrice, err = strconv.ParseFloat(candleData.Candle[3], 64)
		if err != nil {
			log.Warnf("%v Candle data could not be parsed: %v", o.GetName(), candleData.Candle[3])
		}
		klineData.ClosePrice, err = strconv.ParseFloat(candleData.Candle[4], 64)
		if err != nil {
			log.Warnf("%v Candle data could not be parsed: %v", o.GetName(), candleData.Candle[4])
		}
		klineData.Volume, err = strconv.ParseFloat(candleData.Candle[5], 64)
		if err != nil {
			log.Warnf("%v Candle data could not be parsed: %v", o.GetName(), candleData.Candle[5])
		}

		o.Websocket.DataHandler <- klineData
	}
}

func (o *OKGroup) wsProcessFundingFees(fundingFees []interface{}) {
	/* This is not supported yet
	for _, fundingFeeInterface := range fundingFees {
		fundingFeeData := fundingFeeInterface.(WebsocketFundingFeeResponse)
		instrument := pair.NewCurrencyPairDelimiter(fundingFeeData.InstrumentID, "-")
	}
	*/
}

func (o *OKGroup) wsProcessOrderBook(orderbooks []interface{}) {
	for _, orderbooksInterface := range orderbooks {
		orderbookData := orderbooksInterface.(WebsocketOrderBooksResponse)
		instrument := pair.NewCurrencyPairDelimiter(orderbookData.Data[0].InstrumentID, "-")
		var asks, bids []orderbook.Item
		for _, data := range orderbookData.Data {
			for _, ask := range data.Asks {
				amount, err := strconv.ParseFloat(ask[2].(string), 64)
				if err != nil {
					log.Warnf("Could not convert %v to float", ask[2])
				}
				price, err := strconv.ParseFloat(ask[0].(string), 64)
				if err != nil {
					log.Warnf("Could not convert %v to float", ask[0])
				}
				asks = append(asks, orderbook.Item{
					Amount: amount,
					Price:  price,
				})
			}
			for _, bid := range data.Bids {
				amount, err := strconv.ParseFloat(bid[2].(string), 64)
				if err != nil {
					log.Warnf("Could not convert %v to float", bid[2])
				}
				price, err := strconv.ParseFloat(bid[0].(string), 64)
				if err != nil {
					log.Warnf("Could not convert %v to float", bid[0])
				}
				bids = append(bids, orderbook.Item{
					Amount: amount,
					Price:  price,
				})
			}
		}
		err := o.Websocket.Orderbook.Update(bids, asks, instrument, time.Now(), o.GetName(), "SPOT")
		if err != nil {
			log.Error(err)
		}
		o.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
			Exchange: o.GetName(),
			Asset:    "SPOT",
			Pair:     instrument,
		}
	}
}
