package okgroup

import (
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"sync"
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

	okGroupWsRateLimit = 30
)

// orderbookMutex Ensures if two entries arrive at once, only one can be processed at a time
var orderbookMutex sync.Mutex
var defaultSubscribedChannels = []string{okGroupWsSpotDepth, okGroupWsSpotCandle300s, okGroupWsSpotTicker, okGroupWsSpotTrade}

// WsConnect initiates a websocket connection
func (o *OKGroup) WsConnect() error {
	if !o.Websocket.IsEnabled() || !o.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := o.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if o.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			o.Websocket.GetWebsocketURL())
	}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go o.WsHandleData(&wg)
	go o.wsPingHandler(&wg)
	if o.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err = o.WsLogin()
		if err != nil {
			log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", o.Name, err)
		}
	}

	o.GenerateDefaultSubscriptions()
	// Ensures that we start the routines and we dont race when shutdown occurs
	wg.Wait()
	return nil
}

// wsPingHandler sends a message "ping" every 27 to maintain the connection to the websocket
func (o *OKGroup) wsPingHandler(wg *sync.WaitGroup) {
	o.Websocket.Wg.Add(1)
	defer o.Websocket.Wg.Done()

	ticker := time.NewTicker(time.Second * 27)
	defer ticker.Stop()

	wg.Done()

	for {
		select {
		case <-o.Websocket.ShutdownC:
			return

		case <-ticker.C:
			err := o.WebsocketConn.SendMessage("ping")
			if o.Verbose {
				log.Debugf(log.ExchangeSys, "%v sending ping", o.GetName())
			}
			if err != nil {
				o.Websocket.DataHandler <- err
			}
		}
	}
}

// WsHandleData handles the read data from the websocket connection
func (o *OKGroup) WsHandleData(wg *sync.WaitGroup) {
	o.Websocket.Wg.Add(1)
	defer func() {
		o.Websocket.Wg.Done()
	}()

	wg.Done()

	for {
		select {
		case <-o.Websocket.ShutdownC:
			return

		default:
			resp, err := o.WebsocketConn.ReadMessage()
			if err != nil {
				o.Websocket.DataHandler <- err
				return
			}
			o.Websocket.TrafficAlert <- struct{}{}
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
					log.Debugf(log.ExchangeSys, "WS Error Event: %v Message: %v", errorResponse.Event, errorResponse.Message)
				}
				o.WsHandleErrorResponse(errorResponse)
				continue
			}
			var eventResponse WebsocketEventResponse
			err = common.JSONDecode(resp.Raw, &eventResponse)
			if err == nil && eventResponse.Event != "" {
				if eventResponse.Event == "login" {
					o.Websocket.SetCanUseAuthenticatedEndpoints(eventResponse.Success)
				}
				if o.Verbose {
					log.Debugf(log.ExchangeSys, "WS Event: %v on Channel: %v", eventResponse.Event, eventResponse.Channel)
				}
				o.Websocket.DataHandler <- eventResponse
				continue
			}
		}
	}
}

// WsLogin sends a login request to websocket to enable access to authenticated endpoints
func (o *OKGroup) WsLogin() error {
	o.Websocket.SetCanUseAuthenticatedEndpoints(true)
	utcTime := time.Now().UTC()
	unixTime := utcTime.Unix()
	signPath := "/users/self/verify"
	hmac := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(fmt.Sprintf("%v", unixTime)+http.MethodGet+signPath),
		[]byte(o.API.Credentials.Secret))
	base64 := crypto.Base64Encode(hmac)
	request := WebsocketEventRequest{
		Operation: "login",
		Arguments: []string{o.API.Credentials.Key, o.API.Credentials.ClientID, fmt.Sprintf("%v", unixTime), base64},
	}
	err := o.WebsocketConn.SendMessage(request)
	if err != nil {
		o.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// WsHandleErrorResponse sends an error message to ws handler
func (o *OKGroup) WsHandleErrorResponse(event WebsocketErrorResponse) {
	errorMessage := fmt.Sprintf("%v error - %v message: %s ",
		o.GetName(), event.ErrorCode, event.Message)
	if o.Verbose {
		log.Error(log.ExchangeSys, errorMessage)
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
func (o *OKGroup) GetAssetTypeFromTableName(table string) asset.Item {
	assetIndex := strings.Index(table, "/")
	return asset.Item(table[:assetIndex])
}

// WsHandleDataResponse classifies the WS response and sends to appropriate handler
func (o *OKGroup) WsHandleDataResponse(response *WebsocketDataResponse) {
	switch o.GetWsChannelWithoutOrderType(response.Table) {

	case okGroupWsCandle60s, okGroupWsCandle180s, okGroupWsCandle300s, okGroupWsCandle900s,
		okGroupWsCandle1800s, okGroupWsCandle3600s, okGroupWsCandle7200s, okGroupWsCandle14400s,
		okGroupWsCandle21600s, okGroupWsCandle43200s, okGroupWsCandle86400s, okGroupWsCandle604900s:
		o.wsProcessCandles(response)
	case okGroupWsDepth, okGroupWsDepth5:
		// Locking, orderbooks cannot be processed out of order
		orderbookMutex.Lock()
		err := o.WsProcessOrderBook(response)
		if err != nil {
			pair := currency.NewPairDelimiter(response.Data[0].InstrumentID, "-")
			channelToResubscribe := wshandler.WebsocketChannelSubscription{
				Channel:  response.Table,
				Currency: pair,
			}
			o.Websocket.ResubscribeToChannel(channelToResubscribe)
		}
		orderbookMutex.Unlock()
	case okGroupWsTicker:
		o.wsProcessTickers(response)
	case okGroupWsTrade:
		o.wsProcessTrades(response)
	default:
		logDataResponse(response)
	}
}

// logDataResponse will log the details of any websocket data event
// where there is no websocket datahandler for it
func logDataResponse(response *WebsocketDataResponse) {
	for i := range response.Data {
		log.Errorf(log.ExchangeSys, "Unhandled channel: '%v'. Instrument '%v' Timestamp '%v', Data '%v",
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
		o.Websocket.DataHandler <- wshandler.TickerData{
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
		o.Websocket.DataHandler <- wshandler.TradeData{
			Amount:       response.Data[i].Size,
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
			log.Warnf(log.ExchangeSys, "%v Time data could not be parsed: %v", o.GetName(), response.Data[i].Candle[0])
		}

		candleIndex := strings.LastIndex(response.Table, okGroupWsCandle)
		secondIndex := strings.LastIndex(response.Table, "0s")
		candleInterval := ""
		if candleIndex > 0 || secondIndex > 0 {
			candleInterval = response.Table[candleIndex+len(okGroupWsCandle) : secondIndex]
		}

		klineData := wshandler.KlineData{
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
		log.Debug(log.ExchangeSys, "Passed checksum!")
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

	err := o.Websocket.Orderbook.LoadSnapshot(&newOrderBook, true)
	if err != nil {
		return err
	}
	o.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: o.GetName(),
		Asset:    o.GetAssetTypeFromTableName(tableName),
		Pair:     instrument,
	}
	return nil
}

// WsProcessUpdateOrderbook updates an existing orderbook using websocket data
// After merging WS data, it will sort, validate and finally update the existing orderbook
func (o *OKGroup) WsProcessUpdateOrderbook(wsEventData *WebsocketDataWrapper, instrument currency.Pair, tableName string) error {
	update := wsorderbook.WebsocketOrderbookUpdate{
		AssetType:    asset.Spot,
		CurrencyPair: instrument,
		UpdateTime:   wsEventData.Timestamp,
	}
	update.Asks = o.AppendWsOrderbookItems(wsEventData.Asks)
	update.Bids = o.AppendWsOrderbookItems(wsEventData.Bids)
	err := o.Websocket.Orderbook.Update(&update)
	if err != nil {
		log.Error(log.ExchangeSys, err)
	}
	updatedOb := o.Websocket.Orderbook.GetOrderbook(instrument, asset.Spot)
	checksum := o.CalculateUpdateOrderbookChecksum(updatedOb)
	if checksum == wsEventData.Checksum {
		if o.Verbose {
			log.Debug(log.ExchangeSys, "Orderbook valid")
		}
		o.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
			Exchange: o.GetName(),
			Asset:    o.GetAssetTypeFromTableName(tableName),
			Pair:     instrument,
		}

	} else {
		if o.Verbose {
			log.Warnln(log.ExchangeSys, "Orderbook invalid")
		}
		return fmt.Errorf("channel: %v. Orderbook update for %v checksum invalid. Received %v Calculated %v", tableName, instrument, wsEventData.Checksum, checksum)
	}
	return nil
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

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (o *OKGroup) GenerateDefaultSubscriptions() {
	enabledCurrencies := o.GetEnabledPairs(asset.Spot)
	var subscriptions []wshandler.WebsocketChannelSubscription
	if o.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		defaultSubscribedChannels = append(defaultSubscribedChannels, okGroupWsSpotMarginAccount, okGroupWsSpotAccount, okGroupWsSpotOrder)
	}
	for i := range defaultSubscribedChannels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = "-"
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  defaultSubscribedChannels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	o.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (o *OKGroup) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	request := WebsocketEventRequest{
		Operation: "subscribe",
		Arguments: []string{fmt.Sprintf("%v:%v", channelToSubscribe.Channel, channelToSubscribe.Currency.String())},
	}
	if strings.EqualFold(channelToSubscribe.Channel, okGroupWsSpotAccount) {
		request.Arguments = []string{fmt.Sprintf("%v:%v", channelToSubscribe.Channel, channelToSubscribe.Currency.Base.String())}
	}

	return o.WebsocketConn.SendMessage(request)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (o *OKGroup) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	request := WebsocketEventRequest{
		Operation: "unsubscribe",
		Arguments: []string{fmt.Sprintf("%v:%v", channelToSubscribe.Channel, channelToSubscribe.Currency.String())},
	}
	return o.WebsocketConn.SendMessage(request)
}
