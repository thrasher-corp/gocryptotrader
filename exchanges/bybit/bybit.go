package bybit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Bybit is the overarching type across this package
type Bybit struct {
	exchange.Base
}

const (
	bybitAPIURL       = "https://api.bybit.com"
	defaultRecvWindow = 5 * time.Second

	sideBuy  = "Buy"
	sideSell = "Sell"

	// Public endpoints
	bybitSpotGetSymbols   = "/spot/v1/symbols"
	bybitOrderBook        = "/spot/quote/v1/depth"
	bybitMergedOrderBook  = "/spot/quote/v1/depth/merged"
	bybitRecentTrades     = "/spot/quote/v1/trades"
	bybitCandlestickChart = "/spot/quote/v1/kline"
	bybit24HrsChange      = "/spot/quote/v1/ticker/24hr"
	bybitLastTradedPrice  = "/spot/quote/v1/ticker/price"
	bybitBestBidAskPrice  = "/spot/quote/v1/ticker/book_ticker"

	// Authenticated endpoints
	bybitSpotOrder            = "/spot/v1/order" // create, query, cancel
	bybitBatchCancelSpotOrder = "/spot/order/batch-cancel"
	bybitOpenOrder            = "/spot/v1/open-orders"
	bybitPastOrder            = "/spot/v1/history-orders"
	bybitTradeHistory         = "/spot/v1/myTrades"
	bybitWalletBalance        = "/spot/v1/account"
	bybitServerTime           = "/spot/v1/time"
)

// GetAllPairs gets all pairs on the exchange
func (by *Bybit) GetAllPairs(ctx context.Context) ([]PairData, error) {
	resp := struct {
		Data []PairData `json:"result"`
	}{}
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestSpot, bybitSpotGetSymbols, publicSpotRate, &resp)
}

func processOB(ob [][2]string) ([]orderbook.Item, error) {
	o := make([]orderbook.Item, len(ob))
	for x := range ob {
		var price, amount float64
		amount, err := strconv.ParseFloat(ob[x][1], 64)
		if err != nil {
			return nil, err
		}
		price, err = strconv.ParseFloat(ob[x][0], 64)
		if err != nil {
			return nil, err
		}
		o[x] = orderbook.Item{
			Price:  price,
			Amount: amount,
		}
	}
	return o, nil
}

func constructOrderbook(order orderbookResponse) (s Orderbook, err error) {
	s.Bids, err = processOB(order.Data.Bids)
	if err != nil {
		return s, err
	}
	s.Asks, err = processOB(order.Data.Asks)
	if err != nil {
		return s, err
	}
	s.Time = time.UnixMilli(order.Data.Time)
	return
}

// GetOrderbook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetOrderBook(ctx context.Context, symbol string, depth int64) (Orderbook, error) {
	var order orderbookResponse
	strDepth := "100" // default depth
	if depth > 0 && depth < 100 {
		strDepth = strconv.FormatInt(depth, 10)
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strDepth)
	path := common.EncodeURLValues(bybitOrderBook, params)
	err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &order)
	if err != nil {
		return Orderbook{}, err
	}

	return constructOrderbook(order)
}

// GetMergedOrderBook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetMergedOrderBook(ctx context.Context, symbol string, scale, depth int64) (Orderbook, error) {
	var order orderbookResponse
	params := url.Values{}
	if scale > 0 {
		params.Set("scale", strconv.FormatInt(scale, 10))
	}

	strDepth := "100" // default depth
	if depth > 0 && depth < 100 {
		strDepth = strconv.FormatInt(depth, 10)
	}

	params.Set("symbol", symbol)
	params.Set("limit", strDepth)
	path := common.EncodeURLValues(bybitMergedOrderBook, params)
	err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &order)
	if err != nil {
		return Orderbook{}, err
	}

	return constructOrderbook(order)
}

// GetTrades gets recent trades from the exchange
func (by *Bybit) GetTrades(ctx context.Context, symbol string, limit int64) ([]TradeItem, error) {
	resp := struct {
		Data []struct {
			Price        float64 `json:"price,string"`
			Time         int64   `json:"time"`
			Quantity     float64 `json:"qty,string"`
			IsBuyerMaker bool    `json:"isBuyerMaker"`
		} `json:"result"`
	}{}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatInt(limit, 10))
	path := common.EncodeURLValues(bybitRecentTrades, params)
	err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}

	trades := make([]TradeItem, len(resp.Data))
	for x := range resp.Data {
		var tradeSide string
		if resp.Data[x].IsBuyerMaker {
			tradeSide = order.Buy.String()
		} else {
			tradeSide = order.Sell.String()
		}

		trades[x] = TradeItem{
			CurrencyPair: symbol,
			Price:        resp.Data[x].Price,
			Side:         tradeSide,
			Volume:       resp.Data[x].Quantity,
			Time:         time.UnixMilli(resp.Data[x].Time),
		}
	}
	return trades, nil
}

// GetKlines data returns the kline data for a specific symbol
func (by *Bybit) GetKlines(ctx context.Context, symbol, period string, limit int64, start, end time.Time) ([]KlineItem, error) {
	resp := struct {
		Data [][]interface{} `json:"result"`
	}{}

	v := url.Values{}
	v.Add("symbol", symbol)
	v.Add("interval", period)
	if !start.IsZero() {
		v.Add("start", strconv.FormatInt(start.Unix(), 10))
	}
	if !end.IsZero() {
		v.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	v.Add("limit", strconv.FormatInt(limit, 10))

	path := common.EncodeURLValues(bybitCandlestickChart, v)
	if err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp); err != nil {
		return nil, err
	}

	klines := make([]KlineItem, len(resp.Data))
	for x := range resp.Data {
		if len(resp.Data[x]) != 11 {
			return klines, fmt.Errorf("%v GetKlines: invalid response, array length not as expected, check api docs for updates", by.Name)
		}
		var kline KlineItem
		var err error
		startTime, ok := resp.Data[x][0].(float64)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for StartTime", by.Name, errTypeAssert)
		}
		kline.StartTime = time.UnixMilli(int64(startTime))

		open, ok := resp.Data[x][1].(string)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for Open", by.Name, errTypeAssert)
		}
		kline.Open, err = strconv.ParseFloat(open, 64)
		if err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for Open", by.Name, errStrParsing)
		}

		high, ok := resp.Data[x][2].(string)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for High", by.Name, errTypeAssert)
		}
		kline.High, err = strconv.ParseFloat(high, 64)
		if err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for High", by.Name, errStrParsing)
		}

		low, ok := resp.Data[x][3].(string)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for Low", by.Name, errTypeAssert)
		}
		kline.Low, err = strconv.ParseFloat(low, 64)
		if err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for Low", by.Name, errStrParsing)
		}

		c, ok := resp.Data[x][4].(string)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for Close", by.Name, errTypeAssert)
		}
		kline.Close, err = strconv.ParseFloat(c, 64)
		if err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for Close", by.Name, errStrParsing)
		}

		volume, ok := resp.Data[x][5].(string)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for Volume", by.Name, errTypeAssert)
		}
		kline.Volume, err = strconv.ParseFloat(volume, 64)
		if err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for Volume", by.Name, errStrParsing)
		}

		endTime, ok := resp.Data[x][6].(float64)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for EndTime", by.Name, errTypeAssert)
		}
		kline.EndTime = time.UnixMilli(int64(endTime))
		quoteAssetVolume, ok := resp.Data[x][7].(string)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for QuoteAssetVolume", by.Name, errTypeAssert)
		}
		kline.QuoteAssetVolume, err = strconv.ParseFloat(quoteAssetVolume, 64)
		if err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for QuoteAssetVolume", by.Name, errStrParsing)
		}

		tradesCount, ok := resp.Data[x][8].(float64)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for TradesCount", by.Name, errTypeAssert)
		}
		kline.TradesCount = int64(tradesCount)

		takerBaseVolume, ok := resp.Data[x][9].(string)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for TakerBaseVolume", by.Name, errTypeAssert)
		}
		kline.TakerBaseVolume, err = strconv.ParseFloat(takerBaseVolume, 64)
		if err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for TakerBaseVolume", by.Name, errStrParsing)
		}

		takerQuoteVolume, ok := resp.Data[x][10].(string)
		if !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for TakerQuoteVolume", by.Name, errTypeAssert)
		}
		kline.TakerQuoteVolume, err = strconv.ParseFloat(takerQuoteVolume, 64)
		if err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for TakerQuoteVolume", by.Name, errStrParsing)
		}

		klines[x] = kline
	}
	return klines, nil
}

// Get24HrsChange returns price change statistics for the last 24 hours
// If symbol not passed then it will return price change statistics for all pairs
func (by *Bybit) Get24HrsChange(ctx context.Context, symbol string) ([]PriceChangeStats, error) {
	type priceChangeStats struct {
		Time         int64   `json:"time"`
		Symbol       string  `json:"symbol"`
		BestBidPrice float64 `json:"bestBidPrice,string"`
		BestAskPrice float64 `json:"bestAskPrice,string"`
		LastPrice    float64 `json:"lastPrice,string"`
		OpenPrice    float64 `json:"openPrice,string"`
		HighPrice    float64 `json:"highPrice,string"`
		LowPrice     float64 `json:"lowPrice,string"`
		Volume       float64 `json:"volume,string"`
		QuoteVolume  float64 `json:"quoteVolume,string"`
	}

	var stats []PriceChangeStats
	if symbol != "" {
		resp := struct {
			Data priceChangeStats `json:"result"`
		}{}

		params := url.Values{}
		params.Set("symbol", symbol)
		path := common.EncodeURLValues(bybit24HrsChange, params)
		err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}

		stats = append(stats, PriceChangeStats{
			time.UnixMilli(resp.Data.Time),
			resp.Data.Symbol,
			resp.Data.BestAskPrice,
			resp.Data.BestAskPrice,
			resp.Data.LastPrice,
			resp.Data.OpenPrice,
			resp.Data.HighPrice,
			resp.Data.LowPrice,
			resp.Data.Volume,
			resp.Data.QuoteVolume,
		})
	} else {
		resp := struct {
			Data []priceChangeStats `json:"result"`
		}{}

		err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybit24HrsChange, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}

		for x := range resp.Data {
			stats = append(stats, PriceChangeStats{
				time.UnixMilli(resp.Data[x].Time),
				resp.Data[x].Symbol,
				resp.Data[x].BestAskPrice,
				resp.Data[x].BestAskPrice,
				resp.Data[x].LastPrice,
				resp.Data[x].OpenPrice,
				resp.Data[x].HighPrice,
				resp.Data[x].LowPrice,
				resp.Data[x].Volume,
				resp.Data[x].QuoteVolume,
			})
		}
	}
	return stats, nil
}

// GetLastTradedPrice returns last trading price
// If symbol not passed then it will return last trading price for all pairs
func (by *Bybit) GetLastTradedPrice(ctx context.Context, symbol string) ([]LastTradePrice, error) {
	var lastTradePrices []LastTradePrice
	if symbol != "" {
		resp := struct {
			Data LastTradePrice `json:"result"`
		}{}

		params := url.Values{}
		params.Set("symbol", symbol)
		path := common.EncodeURLValues(bybitLastTradedPrice, params)
		err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}
		lastTradePrices = append(lastTradePrices, LastTradePrice{
			resp.Data.Symbol,
			resp.Data.Price,
		})
	} else {
		resp := struct {
			Data []LastTradePrice `json:"result"`
		}{}

		err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybitLastTradedPrice, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}
		for x := range resp.Data {
			lastTradePrices = append(lastTradePrices, LastTradePrice{
				resp.Data[x].Symbol,
				resp.Data[x].Price,
			})
		}
	}
	return lastTradePrices, nil
}

// GetBestBidAskPrice returns best BID and ASK price
// If symbol not passed then it will return best BID and ASK price for all pairs
func (by *Bybit) GetBestBidAskPrice(ctx context.Context, symbol string) ([]TickerData, error) {
	type bestTicker struct {
		Symbol      string  `json:"symbol"`
		BidPrice    float64 `json:"bidPrice,string"`
		BidQuantity float64 `json:"bidQty,string"`
		AskPrice    float64 `json:"askPrice,string"`
		AskQuantity float64 `json:"askQty,string"`
		Time        int64   `json:"time"`
	}

	var tickers []TickerData
	if symbol != "" {
		resp := struct {
			Data bestTicker `json:"result"`
		}{}

		params := url.Values{}
		params.Set("symbol", symbol)
		path := common.EncodeURLValues(bybitBestBidAskPrice, params)
		err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}
		tickers = append(tickers, TickerData{
			resp.Data.Symbol,
			resp.Data.BidPrice,
			resp.Data.BidQuantity,
			resp.Data.AskPrice,
			resp.Data.AskQuantity,
			time.UnixMilli(resp.Data.Time),
		})
	} else {
		resp := struct {
			Data []bestTicker `json:"result"`
		}{}

		err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybitBestBidAskPrice, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}
		for x := range resp.Data {
			tickers = append(tickers, TickerData{
				resp.Data[x].Symbol,
				resp.Data[x].BidPrice,
				resp.Data[x].BidQuantity,
				resp.Data[x].AskPrice,
				resp.Data[x].AskQuantity,
				time.UnixMilli(resp.Data[x].Time),
			})
		}
	}
	return tickers, nil
}

// CreatePostOrder create and post order
func (by *Bybit) CreatePostOrder(ctx context.Context, o *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	if o == nil {
		return nil, errors.New("orderRequest param can't be nil")
	}

	params := url.Values{}
	params.Set("symbol", o.Symbol)
	params.Set("qty", strconv.FormatFloat(o.Quantity, 'f', -1, 64))
	params.Set("side", o.Side)
	params.Set("type", string(o.TradeType))

	if o.TimeInForce != "" {
		params.Set("timeInForce", string(o.TimeInForce))
	}
	if o.TradeType == BybitRequestParamsOrderLimit || o.TradeType == BybitRequestParamsOrderLimitMaker {
		if o.Price == 0 {
			return nil, errors.New("price should be present for Limit and LimitMaker orders")
		}
		params.Set("price", strconv.FormatFloat(o.Price, 'f', -1, 64))
	}
	if o.OrderLinkID != "" {
		params.Set("orderLinkId", o.OrderLinkID)
	}

	resp := struct {
		Data PlaceOrderResponse `json:"result"`
	}{}
	err := by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, bybitSpotOrder, params, &resp, privateSpotRate)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// QueryOrder returns order data based upon orderID or orderLinkID
func (by *Bybit) QueryOrder(ctx context.Context, orderID, orderLinkID string) (*QueryOrderResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, errors.New("atleast one should be present among orderID and orderLinkID")
	}

	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}

	resp := struct {
		Data QueryOrderResponse `json:"result"`
	}{}
	err := by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitSpotOrder, params, &resp, privateSpotRate)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// CancelExistingOrder cancels existing order based upon orderID or orderLinkID
func (by *Bybit) CancelExistingOrder(ctx context.Context, orderID, orderLinkID string) (*CancelOrderResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, errors.New("atleast one should be present among orderID and orderLinkID")
	}

	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}

	resp := struct {
		Data CancelOrderResponse `json:"result"`
	}{}
	return &resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitSpotOrder, params, &resp, privateSpotRate)
}

// BatchCancelOrder cancels orders in batch based upon symbol, side or orderType
func (by *Bybit) BatchCancelOrder(ctx context.Context, symbol, side, orderTypes string) (bool, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderTypes != "" {
		params.Set("orderTypes", orderTypes)
	}

	resp := struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitBatchCancelSpotOrder, params, &resp, privateSpotRate)
}

// ListOpenOrders returns all open orders
func (by *Bybit) ListOpenOrders(ctx context.Context, symbol, orderID string, limit int64) ([]QueryOrderResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	resp := struct {
		Data []QueryOrderResponse `json:"result"`
	}{}
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitOpenOrder, params, &resp, privateSpotRate)
}

// ListPastOrders returns all past orders from history
func (by *Bybit) ListPastOrders(ctx context.Context, symbol, orderID string, limit int64) ([]QueryOrderResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	resp := struct {
		Data []QueryOrderResponse `json:"result"`
	}{}
	err := by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitPastOrder, params, &resp, privateSpotRate)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetTradeHistory returns user trades
func (by *Bybit) GetTradeHistory(ctx context.Context, symbol string, limit, formID, told int64) ([]HistoricalTrade, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if formID != 0 {
		params.Set("formId", strconv.FormatInt(formID, 10))
	}
	if told != 0 {
		params.Set("told", strconv.FormatInt(told, 10))
	}

	resp := struct {
		Data []HistoricalTrade `json:"result"`
	}{}
	err := by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitPastOrder, params, &resp, privateSpotRate)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetWalletBalance returns user wallet balance
func (by *Bybit) GetWalletBalance(ctx context.Context) ([]Balance, error) {
	resp := struct {
		Data struct {
			Balances []Balance `json:"balances"`
		} `json:"result"`
	}{}
	err := by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitWalletBalance, url.Values{}, &resp, privateSpotRate)
	if err != nil {
		return nil, err
	}
	return resp.Data.Balances, nil
}

// SendHTTPRequest sends an unauthenticated request
func (by *Bybit) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := by.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Result:        result,
		Verbose:       by.Verbose,
		HTTPDebugging: by.HTTPDebugging,
		HTTPRecording: by.HTTPRecording}

	return by.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	})
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (by *Bybit) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, result interface{}, f request.EndpointLimit) error {
	if !by.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", by.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	endpointPath, err := by.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}

	if params == nil {
		params = url.Values{}
	}

	if params.Get("recvWindow") == "" {
		params.Set("recvWindow", strconv.FormatInt(defaultRecvWindow.Milliseconds(), 10))
	}

	return by.SendPayload(ctx, f, func() (*request.Item, error) {
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		params.Set("api_key", by.API.Credentials.Key)
		signature := params.Encode()
		hmacSigned, err := crypto.GetHMAC(crypto.HashSHA256, []byte(signature), []byte(by.API.Credentials.Secret))
		if err != nil {
			return nil, err
		}
		hmacSignedStr := crypto.HexEncodeToString(hmacSigned)

		headers := make(map[string]string)
		var payload []byte
		switch method {
		case http.MethodPost:
			headers["Content-Type"] = "application/json"
			m := make(map[string]string)
			m["api_key"] = by.API.Credentials.Key

			for k, v := range params {
				m[k] = strings.Join(v, "")
			}
			m["sign"] = hmacSignedStr
			payload, err = json.Marshal(m)
			if err != nil {
				return nil, err
			}
		default:
			headers["Content-Type"] = "application/x-www-form-urlencoded"
			path = common.EncodeURLValues(path, params)
			path += "&sign=" + hmacSignedStr
		}
		return &request.Item{
			Method:        method,
			Path:          endpointPath + path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &result,
			AuthRequest:   true,
			Verbose:       by.Verbose,
			HTTPDebugging: by.HTTPDebugging,
			HTTPRecording: by.HTTPRecording}, nil
	})
}
