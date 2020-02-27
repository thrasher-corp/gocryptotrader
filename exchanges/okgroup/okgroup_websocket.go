package okgroup

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
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

	allowableIterations = 25
	delimiterColon      = ":"
	delimiterDash       = "-"
	delimiterUnderscore = "_"
)

// orderbookMutex Ensures if two entries arrive at once, only one can be
// processed at a time
var orderbookMutex sync.Mutex

var defaultSpotSubscribedChannels = []string{okGroupWsSpotDepth,
	okGroupWsSpotCandle300s,
	okGroupWsSpotTicker,
	okGroupWsSpotTrade}

var defaultFuturesSubscribedChannels = []string{okGroupWsFuturesDepth,
	okGroupWsFuturesCandle300s,
	okGroupWsFuturesTicker,
	okGroupWsFuturesTrade}

var defaultIndexSubscribedChannels = []string{okGroupWsIndexCandle300s,
	okGroupWsIndexTicker}

var defaultSwapSubscribedChannels = []string{okGroupWsSwapDepth,
	okGroupWsSwapCandle300s,
	okGroupWsSwapTicker,
	okGroupWsSwapTrade,
	okGroupWsSwapFundingRate,
	okGroupWsSwapMarkPrice}

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
	wg.Add(1)
	go o.WsReadData(&wg)
	if o.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err = o.WsLogin()
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v - authentication failed: %v\n",
				o.Name,
				err)
		}
	}

	o.GenerateDefaultSubscriptions()
	// Ensures that we start the routines and we dont race when shutdown occurs
	wg.Wait()
	return nil
}

// WsLogin sends a login request to websocket to enable access to authenticated endpoints
func (o *OKGroup) WsLogin() error {
	o.Websocket.SetCanUseAuthenticatedEndpoints(true)
	unixTime := time.Now().UTC().Unix()
	signPath := "/users/self/verify"
	hmac := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(strconv.FormatInt(unixTime, 10)+http.MethodGet+signPath),
		[]byte(o.API.Credentials.Secret),
	)
	base64 := crypto.Base64Encode(hmac)
	request := WebsocketEventRequest{
		Operation: "login",
		Arguments: []string{
			o.API.Credentials.Key,
			o.API.Credentials.ClientID,
			strconv.FormatInt(unixTime, 10),
			base64,
		},
	}
	err := o.WebsocketConn.SendJSONMessage(request)
	if err != nil {
		o.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// WsReadData receives and passes on websocket messages for processing
func (o *OKGroup) WsReadData(wg *sync.WaitGroup) {
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
				o.Websocket.ReadMessageErrors <- err
				return
			}
			o.Websocket.TrafficAlert <- struct{}{}
			err = o.WsHandleData(resp.Raw)
			if err != nil {
				o.Websocket.DataHandler <- err
			}
		}
	}
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (o *OKGroup) WsHandleData(respRaw []byte) error {
	var dataResponse WebsocketDataResponse
	err := json.Unmarshal(respRaw, &dataResponse)
	if err != nil {
		return err
	}
	if len(dataResponse.Data) > 0 {
		switch o.GetWsChannelWithoutOrderType(dataResponse.Table) {
		case okGroupWsCandle60s, okGroupWsCandle180s, okGroupWsCandle300s,
			okGroupWsCandle900s, okGroupWsCandle1800s, okGroupWsCandle3600s,
			okGroupWsCandle7200s, okGroupWsCandle14400s, okGroupWsCandle21600s,
			okGroupWsCandle43200s, okGroupWsCandle86400s, okGroupWsCandle604900s:
			return o.wsProcessCandles(respRaw)
		case okGroupWsDepth, okGroupWsDepth5:
			return o.WsProcessOrderBook(respRaw)
		case okGroupWsTicker:
			return o.wsProcessTickers(respRaw)
		case okGroupWsTrade:
			return o.wsProcessTrades(respRaw)
		case okGroupWsOrder:
			return o.wsProcessOrder(respRaw)
		}
		o.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: o.Name + wshandler.UnhandledMessage + string(respRaw)}
		return nil
	}

	var errorResponse WebsocketErrorResponse
	err = json.Unmarshal(respRaw, &errorResponse)
	if err == nil && errorResponse.ErrorCode > 0 {
		return fmt.Errorf("%v error - %v message: %s ",
			o.Name,
			errorResponse.ErrorCode,
			errorResponse.Message)
	}
	var eventResponse WebsocketEventResponse
	err = json.Unmarshal(respRaw, &eventResponse)
	if err == nil && eventResponse.Event != "" {
		if eventResponse.Event == "login" {
			o.Websocket.SetCanUseAuthenticatedEndpoints(eventResponse.Success)
		}
		if o.Verbose {
			log.Debug(log.ExchangeSys,
				o.Name+" - "+eventResponse.Event+" on channel: "+eventResponse.Channel)
		}
	}
	return nil
}

// StringToOrderStatus converts order status IDs to internal types
func StringToOrderStatus(num int64) (order.Status, error) {
	switch num {
	case -2:
		return order.Rejected, nil
	case -1:
		return order.Cancelled, nil
	case 0:
		return order.Active, nil
	case 1:
		return order.PartiallyFilled, nil
	case 2:
		return order.Filled, nil
	case 3:
		return order.New, nil
	case 4:
		return order.PendingCancel, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%v not recognised as order status", num)
	}
}

func (o *OKGroup) wsProcessOrder(respRaw []byte) error {
	var resp WebsocketSpotOrderResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	for i := range resp.Data {
		var oType order.Type
		var oSide order.Side
		var oStatus order.Status
		oType, err = order.StringToOrderType(resp.Data[i].Type)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}
		oSide, err = order.StringToOrderSide(resp.Data[i].Side)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}
		oStatus, err = StringToOrderStatus(resp.Data[i].State)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}
		o.Websocket.DataHandler <- &order.Detail{
			ImmediateOrCancel: resp.Data[i].OrderType == 3,
			FillOrKill:        resp.Data[i].OrderType == 2,
			PostOnly:          resp.Data[i].OrderType == 1,
			Price:             resp.Data[i].Price,
			Amount:            resp.Data[i].Size,
			ExecutedAmount:    resp.Data[i].LastFillQty,
			RemainingAmount:   resp.Data[i].Size - resp.Data[i].LastFillQty,
			Exchange:          o.Name,
			ID:                resp.Data[i].OrderID,
			Type:              oType,
			Side:              oSide,
			Status:            oStatus,
			AssetType:         o.GetAssetTypeFromTableName(resp.Table),
			Date:              resp.Data[i].CreatedAt,
			Pair:              currency.NewPairFromString(resp.Data[i].InstrumentID),
		}
	}
	return nil
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (o *OKGroup) wsProcessTickers(respRaw []byte) error {
	var response WebsocketTickerData
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	for i := range response.Data {
		a := o.GetAssetTypeFromTableName(response.Table)
		var c currency.Pair
		switch a {
		case asset.Futures, asset.PerpetualSwap:
			f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
			c = currency.NewPairWithDelimiter(f[0]+delimiterDash+f[1], f[2], delimiterUnderscore)
		default:
			f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
			c = currency.NewPairWithDelimiter(f[0], f[1], delimiterDash)
		}
		o.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: o.Name,
			Open:         response.Data[i].Open24h,
			Close:        response.Data[i].Last,
			Volume:       response.Data[i].BaseVolume24h,
			QuoteVolume:  response.Data[i].QuoteVolume24h,
			High:         response.Data[i].High24h,
			Low:          response.Data[i].Low24h,
			Bid:          response.Data[i].BestBid,
			Ask:          response.Data[i].BestAsk,
			Last:         response.Data[i].Last,
			AssetType:    o.GetAssetTypeFromTableName(response.Table),
			Pair:         c,
			LastUpdated:  response.Data[i].Timestamp,
		}
	}
	return nil
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (o *OKGroup) wsProcessTrades(respRaw []byte) error {
	var response WebsocketTradeResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	for i := range response.Data {
		a := o.GetAssetTypeFromTableName(response.Table)
		var c currency.Pair
		switch a {
		case asset.Futures, asset.PerpetualSwap:
			f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
			c = currency.NewPairWithDelimiter(f[0]+delimiterDash+f[1], f[2], delimiterUnderscore)
		default:
			f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
			c = currency.NewPairWithDelimiter(f[0], f[1], delimiterDash)
		}
		tSide, err := order.StringToOrderSide(response.Data[i].Side)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				Err:      err,
			}
		}
		o.Websocket.DataHandler <- wshandler.TradeData{
			Amount:       response.Data[i].Size,
			AssetType:    o.GetAssetTypeFromTableName(response.Table),
			CurrencyPair: c,
			Exchange:     o.Name,
			Price:        response.Data[i].Price,
			Side:         tSide,
			Timestamp:    response.Data[i].Timestamp,
		}
	}
	return nil
}

// wsProcessCandles converts candle data and sends it to the data handler
func (o *OKGroup) wsProcessCandles(respRaw []byte) error {
	var response WebsocketCandleResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	for i := range response.Data {
		a := o.GetAssetTypeFromTableName(response.Table)
		var c currency.Pair
		switch a {
		case asset.Futures, asset.PerpetualSwap:
			f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
			c = currency.NewPairWithDelimiter(f[0]+delimiterDash+f[1], f[2], delimiterUnderscore)
		default:
			f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
			c = currency.NewPairWithDelimiter(f[0], f[1], delimiterDash)
		}

		timeData, err := time.Parse(time.RFC3339Nano,
			response.Data[i].Candle[0])
		if err != nil {
			return fmt.Errorf("%v Time data could not be parsed: %v",
				o.Name,
				response.Data[i].Candle[0])
		}

		candleIndex := strings.LastIndex(response.Table, okGroupWsCandle)
		secondIndex := strings.LastIndex(response.Table, "0s")
		candleInterval := ""
		if candleIndex > 0 || secondIndex > 0 {
			candleInterval = response.Table[candleIndex+len(okGroupWsCandle) : secondIndex]
		}

		klineData := wshandler.KlineData{
			AssetType: o.GetAssetTypeFromTableName(response.Table),
			Pair:      c,
			Exchange:  o.Name,
			Timestamp: timeData,
			Interval:  candleInterval,
		}
		klineData.OpenPrice, err = strconv.ParseFloat(response.Data[i].Candle[1], 64)
		if err != nil {
			return err
		}
		klineData.HighPrice, err = strconv.ParseFloat(response.Data[i].Candle[2], 64)
		if err != nil {
			return err
		}
		klineData.LowPrice, err = strconv.ParseFloat(response.Data[i].Candle[3], 64)
		if err != nil {
			return err
		}
		klineData.ClosePrice, err = strconv.ParseFloat(response.Data[i].Candle[4], 64)
		if err != nil {
			return err
		}
		klineData.Volume, err = strconv.ParseFloat(response.Data[i].Candle[5], 64)
		if err != nil {
			return err
		}

		o.Websocket.DataHandler <- klineData
	}
	return nil
}

// WsProcessOrderBook Validates the checksum and updates internal orderbook values
func (o *OKGroup) WsProcessOrderBook(respRaw []byte) error {
	var response WebsocketOrderBooksData
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	orderbookMutex.Lock()
	defer orderbookMutex.Unlock()
	for i := range response.Data {
		a := o.GetAssetTypeFromTableName(response.Table)
		var c currency.Pair
		switch a {
		case asset.Futures, asset.PerpetualSwap:
			f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
			c = currency.NewPairWithDelimiter(f[0]+delimiterDash+f[1], f[2], delimiterUnderscore)
		default:
			f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
			c = currency.NewPairWithDelimiter(f[0], f[1], delimiterDash)
		}

		if response.Action == okGroupWsOrderbookPartial {
			err := o.WsProcessPartialOrderBook(&response.Data[i], c, a)
			if err != nil {
				o.wsResubscribeToOrderbook(&response)
				return err
			}
		} else if response.Action == okGroupWsOrderbookUpdate {
			if len(response.Data[i].Asks) == 0 && len(response.Data[i].Bids) == 0 {
				return nil
			}
			err := o.WsProcessUpdateOrderbook(&response.Data[i], c, a)
			if err != nil {
				o.wsResubscribeToOrderbook(&response)
				return err
			}
		}
	}
	return nil
}

func (o *OKGroup) wsResubscribeToOrderbook(response *WebsocketOrderBooksData) {
	for i := range response.Data {
		a := o.GetAssetTypeFromTableName(response.Table)
		var c currency.Pair
		switch a {
		case asset.Futures, asset.PerpetualSwap:
			f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
			c = currency.NewPairWithDelimiter(f[0]+delimiterDash+f[1], f[2], delimiterDash)
		default:
			f := strings.Split(response.Data[i].InstrumentID, delimiterDash)
			c = currency.NewPairWithDelimiter(f[0], f[1], delimiterDash)
		}

		channelToResubscribe := wshandler.WebsocketChannelSubscription{
			Channel:  response.Table,
			Currency: c,
		}
		o.Websocket.ResubscribeToChannel(channelToResubscribe)
	}
}

// AppendWsOrderbookItems adds websocket orderbook data bid/asks into an orderbook item array
func (o *OKGroup) AppendWsOrderbookItems(entries [][]interface{}) ([]orderbook.Item, error) {
	var items []orderbook.Item
	for j := range entries {
		amount, err := strconv.ParseFloat(entries[j][1].(string), 64)
		if err != nil {
			return nil, err
		}
		price, err := strconv.ParseFloat(entries[j][0].(string), 64)
		if err != nil {
			return nil, err
		}
		items = append(items, orderbook.Item{Amount: amount, Price: price})
	}
	return items, nil
}

// WsProcessPartialOrderBook takes websocket orderbook data and creates an orderbook
// Calculates checksum to ensure it is valid
func (o *OKGroup) WsProcessPartialOrderBook(wsEventData *WebsocketOrderBook, instrument currency.Pair, a asset.Item) error {
	signedChecksum := o.CalculatePartialOrderbookChecksum(wsEventData)
	if signedChecksum != wsEventData.Checksum {
		return fmt.Errorf("%s channel: %s. Orderbook partial for %v checksum invalid",
			o.Name,
			a,
			instrument)
	}
	if o.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s passed checksum for instrument %s",
			o.Name,
			instrument)
	}

	asks, err := o.AppendWsOrderbookItems(wsEventData.Asks)
	if err != nil {
		return err
	}

	bids, err := o.AppendWsOrderbookItems(wsEventData.Bids)
	if err != nil {
		return err
	}

	newOrderBook := orderbook.Base{
		Asks:         asks,
		Bids:         bids,
		AssetType:    a,
		LastUpdated:  wsEventData.Timestamp,
		Pair:         instrument,
		ExchangeName: o.Name,
	}

	err = o.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
	if err != nil {
		return err
	}

	o.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: o.Name,
		Asset:    a,
		Pair:     instrument,
	}
	return nil
}

// WsProcessUpdateOrderbook updates an existing orderbook using websocket data
// After merging WS data, it will sort, validate and finally update the existing orderbook
func (o *OKGroup) WsProcessUpdateOrderbook(wsEventData *WebsocketOrderBook, instrument currency.Pair, a asset.Item) error {
	update := wsorderbook.WebsocketOrderbookUpdate{
		Asset:      a,
		Pair:       instrument,
		UpdateTime: wsEventData.Timestamp,
	}

	var err error
	update.Asks, err = o.AppendWsOrderbookItems(wsEventData.Asks)
	if err != nil {
		return err
	}
	update.Bids, err = o.AppendWsOrderbookItems(wsEventData.Bids)
	if err != nil {
		return err
	}

	err = o.Websocket.Orderbook.Update(&update)
	if err != nil {
		return err
	}

	updatedOb := o.Websocket.Orderbook.GetOrderbook(instrument, a)
	checksum := o.CalculateUpdateOrderbookChecksum(updatedOb)

	if checksum != wsEventData.Checksum {
		// re-sub
		log.Warnf(log.ExchangeSys, "%s checksum failure for item %s",
			o.Name,
			wsEventData.InstrumentID)
		return errors.New("checksum failed")
	}

	o.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Exchange: o.Name,
		Asset:    a,
		Pair:     instrument,
	}

	return nil
}

// CalculatePartialOrderbookChecksum alternates over the first 25 bid and ask
// entries from websocket data. The checksum is made up of the price and the
// quantity with a semicolon (:) deliminating them. This will also work when
// there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (o *OKGroup) CalculatePartialOrderbookChecksum(orderbookData *WebsocketOrderBook) int32 {
	var checksum strings.Builder
	for i := 0; i < allowableIterations; i++ {
		if len(orderbookData.Bids)-1 >= i {
			checksum.WriteString(orderbookData.Bids[i][0].(string) +
				delimiterColon +
				orderbookData.Bids[i][1].(string) +
				delimiterColon)
		}
		if len(orderbookData.Asks)-1 >= i {
			checksum.WriteString(orderbookData.Asks[i][0].(string) +
				delimiterColon +
				orderbookData.Asks[i][1].(string) +
				delimiterColon)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), delimiterColon)
	return int32(crc32.ChecksumIEEE([]byte(checksumStr)))
}

// CalculateUpdateOrderbookChecksum alternates over the first 25 bid and ask
// entries of a merged orderbook. The checksum is made up of the price and the
// quantity with a semicolon (:) deliminating them. This will also work when
// there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (o *OKGroup) CalculateUpdateOrderbookChecksum(orderbookData *orderbook.Base) int32 {
	var checksum strings.Builder
	for i := 0; i < allowableIterations; i++ {
		if len(orderbookData.Bids)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Bids[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Bids[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + delimiterColon + amount + delimiterColon)
		}
		if len(orderbookData.Asks)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Asks[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Asks[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + delimiterColon + amount + delimiterColon)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), delimiterColon)
	return int32(crc32.ChecksumIEEE([]byte(checksumStr)))
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be
// handled by ManageSubscriptions()
func (o *OKGroup) GenerateDefaultSubscriptions() {
	var subscriptions []wshandler.WebsocketChannelSubscription
	assets := o.GetAssetTypes()
	for x := range assets {
		enabledCurrencies := o.GetEnabledPairs(assets[x])
		if len(enabledCurrencies) == 0 {
			continue
		}

		switch assets[x] {
		case asset.Spot:
			for i := range enabledCurrencies {
				for y := range defaultSpotSubscribedChannels {
					subscriptions = append(subscriptions,
						wshandler.WebsocketChannelSubscription{
							Channel: defaultSpotSubscribedChannels[y],
							Currency: o.FormatExchangeCurrency(enabledCurrencies[i],
								asset.Spot),
						})
				}
			}

			if o.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
				subscriptions = append(subscriptions,
					wshandler.WebsocketChannelSubscription{
						Channel: okGroupWsSpotMarginAccount,
					},
					wshandler.WebsocketChannelSubscription{
						Channel: okGroupWsSpotAccount,
					},
					wshandler.WebsocketChannelSubscription{
						Channel: okGroupWsSpotOrder,
					})
			}
		case asset.Futures:
			for i := range enabledCurrencies {
				for y := range defaultFuturesSubscribedChannels {
					subscriptions = append(subscriptions,
						wshandler.WebsocketChannelSubscription{
							Channel: defaultFuturesSubscribedChannels[y],
							Currency: o.FormatExchangeCurrency(enabledCurrencies[i],
								asset.Futures),
						})
				}
			}

			if o.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
				subscriptions = append(subscriptions,
					wshandler.WebsocketChannelSubscription{
						Channel: okGroupWsFuturesAccount,
					},
					wshandler.WebsocketChannelSubscription{
						Channel: okGroupWsFuturesPosition,
					},
					wshandler.WebsocketChannelSubscription{
						Channel: okGroupWsFuturesOrder,
					})
			}
		case asset.PerpetualSwap:
			for i := range enabledCurrencies {
				for y := range defaultSwapSubscribedChannels {
					subscriptions = append(subscriptions,
						wshandler.WebsocketChannelSubscription{
							Channel: defaultSwapSubscribedChannels[y],
							Currency: o.FormatExchangeCurrency(enabledCurrencies[i],
								asset.PerpetualSwap),
						})
				}
			}

			if o.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
				subscriptions = append(subscriptions,
					wshandler.WebsocketChannelSubscription{
						Channel: okGroupWsSwapAccount,
					},
					wshandler.WebsocketChannelSubscription{
						Channel: okGroupWsSwapPosition,
					},
					wshandler.WebsocketChannelSubscription{
						Channel: okGroupWsSwapOrder,
					})
			}
		case asset.Index:
			for i := range enabledCurrencies {
				for y := range defaultIndexSubscribedChannels {
					subscriptions = append(subscriptions,
						wshandler.WebsocketChannelSubscription{
							Channel:  defaultIndexSubscribedChannels[y],
							Currency: o.FormatExchangeCurrency(enabledCurrencies[i], asset.Index),
						})
				}
			}
		default:
			o.Websocket.DataHandler <- errors.New("unhandled asset type")
		}
	}

	o.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (o *OKGroup) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	c := channelToSubscribe.Currency.String()
	request := WebsocketEventRequest{
		Operation: "subscribe",
		Arguments: []string{channelToSubscribe.Channel + delimiterColon + c},
	}
	if strings.EqualFold(channelToSubscribe.Channel, okGroupWsSpotAccount) {
		request.Arguments = []string{channelToSubscribe.Channel +
			delimiterColon +
			channelToSubscribe.Currency.Base.String()}
	}

	return o.WebsocketConn.SendJSONMessage(request)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (o *OKGroup) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	request := WebsocketEventRequest{
		Operation: "unsubscribe",
		Arguments: []string{channelToSubscribe.Channel +
			delimiterColon +
			channelToSubscribe.Currency.String()},
	}
	return o.WebsocketConn.SendJSONMessage(request)
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
	switch table[:assetIndex] {
	case asset.Futures.String():
		return asset.Futures
	case asset.Spot.String():
		return asset.Spot
	case "swap":
		return asset.PerpetualSwap
	case asset.Index.String():
		return asset.Index
	default:
		log.Warnf(log.ExchangeSys, "%s unhandled asset type %s",
			o.Name,
			table[:assetIndex])
		return asset.Item(table[:assetIndex])
	}
}
