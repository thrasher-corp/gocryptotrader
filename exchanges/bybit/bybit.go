package bybit

import (
	"bytes"
	"context"
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

// TODO: Visit change logs for all asset which were made after development started
const (
	bybitAPIURL       = "https://api.bybit.com"
	defaultRecvWindow = "5000" // 5000 milli second

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
	bybitSpotOrder                = "/spot/v1/order" // create, query, cancel
	bybitFastCancelSpotOrder      = "/spot/v1/order/fast"
	bybitBatchCancelSpotOrder     = "/spot/order/batch-cancel"
	bybitFastBatchCancelSpotOrder = "/spot/order/batch-fast-cancel"
	bybitOpenOrder                = "/spot/v1/open-orders"
	bybitPastOrder                = "/spot/v1/history-orders"
	bybitTradeHistory             = "/spot/v1/myTrades"
	bybitWalletBalance            = "/spot/v1/account"
	bybitServerTime               = "/spot/v1/time"
)

// GetAllSpotPairs gets all pairs on the exchange
func (by *Bybit) GetAllSpotPairs(ctx context.Context) ([]PairData, error) {
	resp := struct {
		Data []PairData `json:"result"`
		Error
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

func constructOrderbook(o *orderbookResponse) (s Orderbook, err error) {
	s.Bids, err = processOB(o.Data.Bids)
	if err != nil {
		return s, err
	}
	s.Asks, err = processOB(o.Data.Asks)
	if err != nil {
		return s, err
	}
	s.Time = o.Data.Time.Time()
	return
}

// GetOrderBook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetOrderBook(ctx context.Context, symbol string, depth int64) (Orderbook, error) {
	var o orderbookResponse
	strDepth := "100" // default depth
	if depth > 0 && depth < 100 {
		strDepth = strconv.FormatInt(depth, 10)
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strDepth)
	path := common.EncodeURLValues(bybitOrderBook, params)
	err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &o)
	if err != nil {
		return Orderbook{}, err
	}

	return constructOrderbook(&o)
}

// GetMergedOrderBook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetMergedOrderBook(ctx context.Context, symbol string, scale, depth int64) (Orderbook, error) {
	var o orderbookResponse
	params := url.Values{}
	if scale > 0 {
		params.Set("scale", strconv.FormatInt(scale, 10))
	}

	strDepth := "100" // default depth
	if depth > 0 && depth <= 200 {
		strDepth = strconv.FormatInt(depth, 10)
	}

	params.Set("symbol", symbol)
	params.Set("limit", strDepth)
	path := common.EncodeURLValues(bybitMergedOrderBook, params)
	err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &o)
	if err != nil {
		return Orderbook{}, err
	}

	return constructOrderbook(&o)
}

// GetTrades gets recent trades from the exchange
func (by *Bybit) GetTrades(ctx context.Context, symbol string, limit int64) ([]TradeItem, error) {
	resp := struct {
		Data []struct {
			Price        float64           `json:"price,string"`
			Time         bybitTimeMilliSec `json:"time"`
			Quantity     float64           `json:"qty,string"`
			IsBuyerMaker bool              `json:"isBuyerMaker"`
		} `json:"result"`
		Error
	}{}

	params := url.Values{}
	params.Set("symbol", symbol)

	strLimit := "60" // default limit
	if limit > 0 && limit < 60 {
		strLimit = strconv.FormatInt(limit, 10)
	}
	params.Set("limit", strLimit)
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
			Time:         resp.Data[x].Time.Time(),
		}
	}
	return trades, nil
}

// GetKlines data returns the kline data for a specific symbol. Limitation: It only returns lastest 3500 candles irrespective of interval passed
func (by *Bybit) GetKlines(ctx context.Context, symbol, period string, limit int64, start, end time.Time) ([]KlineItem, error) {
	resp := struct {
		Data [][]interface{} `json:"result"`
		Error
	}{}

	v := url.Values{}
	v.Add("symbol", symbol)
	v.Add("interval", period)
	if !start.IsZero() {
		v.Add("startTime", strconv.FormatInt(start.UnixMilli(), 10))
	}
	if !end.IsZero() {
		v.Add("endTime", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if limit <= 0 || limit > 1000 {
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
		Time         bybitTimeMilliSec `json:"time"`
		Symbol       string            `json:"symbol"`
		BestBidPrice float64           `json:"bestBidPrice,string"`
		BestAskPrice float64           `json:"bestAskPrice,string"`
		LastPrice    float64           `json:"lastPrice,string"`
		OpenPrice    float64           `json:"openPrice,string"`
		HighPrice    float64           `json:"highPrice,string"`
		LowPrice     float64           `json:"lowPrice,string"`
		Volume       float64           `json:"volume,string"`
		QuoteVolume  float64           `json:"quoteVolume,string"`
	}

	var stats []PriceChangeStats
	if symbol != "" {
		resp := struct {
			Data priceChangeStats `json:"result"`
			Error
		}{}

		params := url.Values{}
		params.Set("symbol", symbol)
		path := common.EncodeURLValues(bybit24HrsChange, params)
		err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}

		stats = append(stats, PriceChangeStats{
			resp.Data.Time.Time(),
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
			Error
		}{}

		err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybit24HrsChange, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}

		for x := range resp.Data {
			stats = append(stats, PriceChangeStats{
				resp.Data[x].Time.Time(),
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
			Error
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
			Error
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
		Symbol      string            `json:"symbol"`
		BidPrice    float64           `json:"bidPrice,string"`
		BidQuantity float64           `json:"bidQty,string"`
		AskPrice    float64           `json:"askPrice,string"`
		AskQuantity float64           `json:"askQty,string"`
		Time        bybitTimeMilliSec `json:"time"`
	}

	var tickers []TickerData
	if symbol != "" {
		resp := struct {
			Data bestTicker `json:"result"`
			Error
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
			resp.Data.Time.Time(),
		})
	} else {
		resp := struct {
			Data []bestTicker `json:"result"`
			Error
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
				resp.Data[x].Time.Time(),
			})
		}
	}
	return tickers, nil
}

// CreatePostOrder create and post order
func (by *Bybit) CreatePostOrder(ctx context.Context, o *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	if o == nil {
		return nil, errInvalidOrderRequest
	}

	params := url.Values{}
	params.Set("symbol", o.Symbol)
	params.Set("qty", strconv.FormatFloat(o.Quantity, 'f', -1, 64))
	params.Set("side", o.Side)
	params.Set("type", o.TradeType)

	if o.TimeInForce != "" {
		params.Set("timeInForce", o.TimeInForce)
	}
	if (o.TradeType == BybitRequestParamsOrderLimit || o.TradeType == BybitRequestParamsOrderLimitMaker) && o.Price == 0 {
		return nil, errMissingPrice
	}
	if o.Price != 0 {
		params.Set("price", strconv.FormatFloat(o.Price, 'f', -1, 64))
	}
	if o.OrderLinkID != "" {
		params.Set("orderLinkId", o.OrderLinkID)
	}

	resp := struct {
		Data PlaceOrderResponse `json:"result"`
		Error
	}{}
	return &resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, bybitSpotOrder, params, &resp, privateSpotRate)
}

// QueryOrder returns order data based upon orderID or orderLinkID
func (by *Bybit) QueryOrder(ctx context.Context, orderID, orderLinkID string) (*QueryOrderResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, errOrderOrOrderLinkIDMissing
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
		Error
	}{}
	return &resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitSpotOrder, params, &resp, privateSpotRate)
}

// CancelExistingOrder cancels existing order based upon orderID or orderLinkID
func (by *Bybit) CancelExistingOrder(ctx context.Context, orderID, orderLinkID string) (*CancelOrderResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, errOrderOrOrderLinkIDMissing
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
		Error
	}{}
	return &resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitSpotOrder, params, &resp, privateSpotRate)
}

// FastCancelExistingOrder cancels existing order based upon orderID or orderLinkID
func (by *Bybit) FastCancelExistingOrder(ctx context.Context, symbol, orderID, orderLinkID string) (*CancelOrderResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, errOrderOrOrderLinkIDMissing
	}

	params := url.Values{}
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params.Set("symbolId", symbol)

	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}

	resp := struct {
		Data CancelOrderResponse `json:"result"`
		Error
	}{}
	return &resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitFastCancelSpotOrder, params, &resp, privateSpotRate)
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
		Result struct {
			Success bool `json:"success"`
		} `json:"result"`
		Error
	}{}
	return resp.Result.Success, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitBatchCancelSpotOrder, params, &resp, privateSpotRate)
}

// BatchFastCancelOrder cancels orders in batch based upon symbol, side or orderType
func (by *Bybit) BatchFastCancelOrder(ctx context.Context, symbol, side, orderTypes string) (bool, error) {
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
		Result struct {
			Success bool `json:"success"`
		} `json:"result"`
		Error
	}{}
	return resp.Result.Success, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitFastBatchCancelSpotOrder, params, &resp, privateSpotRate)
}

// BatchCancelOrderByIDs cancels orders in batch based on comma separated order id's
func (by *Bybit) BatchCancelOrderByIDs(ctx context.Context, orderIDs []string) (bool, error) {
	params := url.Values{}
	if len(orderIDs) == 0 {
		return false, errEmptyOrderIDs
	}
	params.Set("orderIds", strings.Join(orderIDs, ","))

	resp := struct {
		Result struct {
			Success bool `json:"success"`
		} `json:"result"`
		Error
	}{}
	return resp.Result.Success, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitFastBatchCancelSpotOrder, params, &resp, privateSpotRate)
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
		Error
	}{}
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitOpenOrder, params, &resp, privateSpotRate)
}

// GetPastOrders returns all past orders from history
func (by *Bybit) GetPastOrders(ctx context.Context, symbol, orderID string, limit int64, startTime, endTime time.Time) ([]QueryOrderResponse, error) {
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
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	resp := struct {
		Data []QueryOrderResponse `json:"result"`
		Error
	}{}
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitPastOrder, params, &resp, privateSpotRate)
}

// GetTradeHistory returns user trades
func (by *Bybit) GetTradeHistory(ctx context.Context, limit int64, symbol, fromID, toID, orderID string, startTime, endTime time.Time) ([]HistoricalTrade, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != "" {
		params.Set("fromTicketId", fromID)
	}
	if toID != "" {
		params.Set("toTicketId", toID)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	resp := struct {
		Data []HistoricalTrade `json:"result"`
		Error
	}{}
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitTradeHistory, params, &resp, privateSpotRate)
}

// GetWalletBalance returns user wallet balance
func (by *Bybit) GetWalletBalance(ctx context.Context) ([]Balance, error) {
	resp := struct {
		Data struct {
			Balances []Balance `json:"balances"`
		} `json:"result"`
		Error
	}{}
	return resp.Data.Balances, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitWalletBalance, url.Values{}, &resp, privateSpotRate)
}

// GetSpotServerTime returns server time
func (by *Bybit) GetSpotServerTime(ctx context.Context) (time.Time, error) {
	resp := struct {
		Result struct {
			ServerTime int64 `json:"serverTime"`
		} `json:"result"`
		Error
	}{}
	err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybitServerTime, publicSpotRate, &resp)
	return time.UnixMilli(resp.Result.ServerTime), err
}

// SendHTTPRequest sends an unauthenticated request
func (by *Bybit) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result UnmarshalTo) error {
	endpointPath, err := by.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}

	err = by.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + path,
			Result:        result,
			Verbose:       by.Verbose,
			HTTPDebugging: by.HTTPDebugging,
			HTTPRecording: by.HTTPRecording}, nil
	})
	if err != nil {
		return err
	}
	return result.GetError()
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (by *Bybit) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, result UnmarshalTo, f request.EndpointLimit) error {
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}

	if result == nil {
		result = &Error{}
	}

	endpointPath, err := by.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}

	if params == nil {
		params = url.Values{}
	}

	if params.Get("recvWindow") == "" {
		params.Set("recvWindow", defaultRecvWindow)
	}

	err = by.SendPayload(ctx, f, func() (*request.Item, error) {
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		params.Set("api_key", creds.Key)
		signature := params.Encode()
		var hmacSigned []byte
		hmacSigned, err = crypto.GetHMAC(crypto.HashSHA256, []byte(signature), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		hmacSignedStr := crypto.HexEncodeToString(hmacSigned)

		headers := make(map[string]string)
		var payload []byte
		headers["Content-Type"] = "application/x-www-form-urlencoded"
		switch method {
		case http.MethodPost:
			params.Set("sign", hmacSignedStr)
			payload = []byte(params.Encode())
		default:
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
	if err != nil {
		return err
	}
	return result.GetError()
}

// Error defines all error information for each request
type Error struct {
	ReturnCode int64  `json:"ret_code"`
	ReturnMsg  string `json:"ret_msg"`
	ExtCode    string `json:"ext_code"`
	ExtMsg     string `json:"ext_info"`
}

// GetError checks and returns an error if it is supplied.
func (e Error) GetError() error {
	if e.ReturnCode != 0 && e.ReturnMsg != "" {
		return errors.New(e.ReturnMsg)
	}
	if e.ExtCode != "" && e.ExtMsg != "" {
		return errors.New(e.ExtMsg)
	}
	return nil
}

func getSide(side string) order.Side {
	switch side {
	case sideBuy:
		return order.Buy
	case sideSell:
		return order.Sell
	default:
		return order.UnknownSide
	}
}

func getTradeType(tradeType string) order.Type {
	switch tradeType {
	case BybitRequestParamsOrderLimit:
		return order.Limit
	case BybitRequestParamsOrderMarket:
		return order.Market
	case BybitRequestParamsOrderLimitMaker:
		return order.Limit
	default:
		return order.UnknownType
	}
}

func getOrderStatus(status string) order.Status {
	switch status {
	case "NEW":
		return order.New
	case "PARTIALLY_FILLED":
		return order.PartiallyFilled
	case "FILLED":
		return order.Filled
	case "CANCELED":
		return order.Cancelled
	case "PENDING_CANCEL":
		return order.PendingCancel
	case "PENDING_NEW":
		return order.Pending
	case "REJECTED":
		return order.Rejected
	default:
		return order.UnknownStatus
	}
}
