package bybit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
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
	bybitGetTickersV5     = "/v5/market/tickers"

	// Authenticated endpoints
	bybitSpotOrder                = "/spot/v1/order" // create, query, cancel
	bybitFastCancelSpotOrder      = "/spot/v1/order/fast"
	bybitBatchCancelSpotOrder     = "/spot/order/batch-cancel"
	bybitFastBatchCancelSpotOrder = "/spot/order/batch-fast-cancel"
	bybitBatchCancelByIDs         = "/spot/order/batch-cancel-by-ids"
	bybitOpenOrder                = "/spot/v1/open-orders"
	bybitPastOrder                = "/spot/v1/history-orders"
	bybitTradeHistory             = "/spot/v1/myTrades"
	bybitWalletBalance            = "/spot/v1/account"
	bybitServerTime               = "/spot/v1/time"
	bybitAccountFee               = "/v5/account/fee-rate"

	// Account asset endpoint
	bybitGetDepositAddress = "/asset/v1/private/deposit/address"
	bybitWithdrawFund      = "/asset/v1/private/withdraw"
)

var (
	errCategoryNotSet = errors.New("category not set")
	errBaseNotSet     = errors.New("base coin not set when category is option")
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

func constructOrderbook(o *orderbookResponse) (*Orderbook, error) {
	var (
		s   Orderbook
		err error
	)
	s.Bids, err = processOB(o.Data.Bids)
	if err != nil {
		return nil, err
	}
	s.Asks, err = processOB(o.Data.Asks)
	if err != nil {
		return nil, err
	}
	s.Time = o.Data.Time.Time()
	return &s, err
}

// GetOrderBook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetOrderBook(ctx context.Context, symbol string, depth int64) (*Orderbook, error) {
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
		return nil, err
	}

	return constructOrderbook(&o)
}

// GetMergedOrderBook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetMergedOrderBook(ctx context.Context, symbol string, scale, depth int64) (*Orderbook, error) {
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
		return nil, err
	}

	return constructOrderbook(&o)
}

// GetTrades gets recent trades from the exchange
func (by *Bybit) GetTrades(ctx context.Context, symbol string, limit int64) ([]TradeItem, error) {
	resp := struct {
		Data []struct {
			Price        convert.StringToFloat64 `json:"price"`
			Time         bybitTimeMilliSec       `json:"time"`
			Quantity     convert.StringToFloat64 `json:"qty"`
			IsBuyerMaker bool                    `json:"isBuyerMaker"`
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
			Price:        resp.Data[x].Price.Float64(),
			Side:         tradeSide,
			Volume:       resp.Data[x].Quantity.Float64(),
			Time:         resp.Data[x].Time.Time(),
		}
	}
	return trades, nil
}

// GetKlines data returns the kline data for a specific symbol. Limitation: It only returns latest 3500 candles irrespective of interval passed
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
			return nil, fmt.Errorf("%v GetKlines: invalid response, array length not as expected, check api docs for updates", by.Name)
		}
		var err error
		startTime, ok := resp.Data[x][0].(float64)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for StartTime", by.Name, errTypeAssert)
		}
		klines[x].StartTime = time.UnixMilli(int64(startTime))

		open, ok := resp.Data[x][1].(string)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for Open", by.Name, errTypeAssert)
		}
		klines[x].Open, err = strconv.ParseFloat(open, 64)
		if err != nil {
			return nil, fmt.Errorf("%v GetKlines: %w for Open", by.Name, errStrParsing)
		}

		high, ok := resp.Data[x][2].(string)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for High", by.Name, errTypeAssert)
		}
		klines[x].High, err = strconv.ParseFloat(high, 64)
		if err != nil {
			return nil, fmt.Errorf("%v GetKlines: %w for High", by.Name, errStrParsing)
		}

		low, ok := resp.Data[x][3].(string)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for Low", by.Name, errTypeAssert)
		}
		klines[x].Low, err = strconv.ParseFloat(low, 64)
		if err != nil {
			return nil, fmt.Errorf("%v GetKlines: %w for Low", by.Name, errStrParsing)
		}

		c, ok := resp.Data[x][4].(string)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for Close", by.Name, errTypeAssert)
		}
		klines[x].Close, err = strconv.ParseFloat(c, 64)
		if err != nil {
			return nil, fmt.Errorf("%v GetKlines: %w for Close", by.Name, errStrParsing)
		}

		volume, ok := resp.Data[x][5].(string)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for Volume", by.Name, errTypeAssert)
		}
		klines[x].Volume, err = strconv.ParseFloat(volume, 64)
		if err != nil {
			return nil, fmt.Errorf("%v GetKlines: %w for Volume", by.Name, errStrParsing)
		}

		endTime, ok := resp.Data[x][6].(float64)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for EndTime", by.Name, errTypeAssert)
		}
		klines[x].EndTime = time.UnixMilli(int64(endTime))
		quoteAssetVolume, ok := resp.Data[x][7].(string)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for QuoteAssetVolume", by.Name, errTypeAssert)
		}
		klines[x].QuoteAssetVolume, err = strconv.ParseFloat(quoteAssetVolume, 64)
		if err != nil {
			return nil, fmt.Errorf("%v GetKlines: %w for QuoteAssetVolume", by.Name, errStrParsing)
		}

		tradesCount, ok := resp.Data[x][8].(float64)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for TradesCount", by.Name, errTypeAssert)
		}
		klines[x].TradesCount = int64(tradesCount)

		klines[x].TakerBaseVolume, ok = resp.Data[x][9].(float64)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for TakerBaseVolume", by.Name, errTypeAssert)
		}

		klines[x].TakerQuoteVolume, ok = resp.Data[x][10].(float64)
		if !ok {
			return nil, fmt.Errorf("%v GetKlines: %w for TakerQuoteVolume", by.Name, errTypeAssert)
		}
	}
	return klines, nil
}

// Get24HrsChange returns price change statistics for the last 24 hours
// If symbol not passed then it will return price change statistics for all pairs
func (by *Bybit) Get24HrsChange(ctx context.Context, symbol string) ([]PriceChangeStats, error) {
	if symbol != "" {
		resp := struct {
			Data PriceChangeStats `json:"result"`
			Error
		}{}

		params := url.Values{}
		params.Set("symbol", symbol)
		path := common.EncodeURLValues(bybit24HrsChange, params)
		err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}
		return []PriceChangeStats{resp.Data}, nil
	}

	resp := struct {
		Data []PriceChangeStats `json:"result"`
		Error
	}{}

	err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybit24HrsChange, publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
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
	if symbol != "" {
		resp := struct {
			Data TickerData `json:"result"`
			Error
		}{}

		params := url.Values{}
		params.Set("symbol", symbol)
		path := common.EncodeURLValues(bybitBestBidAskPrice, params)
		err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}
		return []TickerData{resp.Data}, nil
	}

	resp := struct {
		Data []TickerData `json:"result"`
		Error
	}{}

	err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybitBestBidAskPrice, publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetTickersV5 returns tickers for either "spot", "option" or "inverse".
// Specific symbol is optional.
func (by *Bybit) GetTickersV5(ctx context.Context, category, symbol, baseCoin string) (*ListOfTickers, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}

	if category == "option" && baseCoin == "" {
		return nil, errBaseNotSet
	}

	val := url.Values{}
	val.Set("category", category)

	if symbol != "" {
		val.Set("symbol", symbol)
	}

	if baseCoin != "" {
		val.Set("baseCoin", baseCoin)
	}

	result := struct {
		Data *ListOfTickers `json:"result"`
		Error
	}{}

	err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybitGetTickersV5+"?"+val.Encode(), publicSpotRate, &result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
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
	return &resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, bybitSpotOrder, params, nil, &resp, privateSpotRate)
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
	return &resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitSpotOrder, params, nil, &resp, privateSpotRate)
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
	err := by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitSpotOrder, params, nil, &resp, privateSpotRate)
	if err != nil {
		return nil, err
	}

	// In case open order is cancelled, this endpoint return status as NEW whereas if we try to cancel a already cancelled order then it's status is returned as CANCELED without any error. So this check is added to prevent this obscurity.
	if resp.Data.Status == "CANCELED" {
		return nil, fmt.Errorf("%s order already cancelled", resp.Data.OrderID)
	}
	return &resp.Data, nil
}

// FastCancelExistingOrder cancels existing order based upon orderID or orderLinkID
func (by *Bybit) FastCancelExistingOrder(ctx context.Context, symbol, orderID, orderLinkID string) (bool, error) {
	resp := struct {
		Data struct {
			IsCancelled bool `json:"isCancelled"`
		} `json:"result"`
		Error
	}{}

	if orderID == "" && orderLinkID == "" {
		return resp.Data.IsCancelled, errOrderOrOrderLinkIDMissing
	}

	params := url.Values{}
	if symbol == "" {
		return resp.Data.IsCancelled, errSymbolMissing
	}
	params.Set("symbolId", symbol)

	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}

	return resp.Data.IsCancelled, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitFastCancelSpotOrder, params, nil, &resp, privateSpotRate)
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
	return resp.Result.Success, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitBatchCancelSpotOrder, params, nil, &resp, privateSpotRate)
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
	return resp.Result.Success, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitFastBatchCancelSpotOrder, params, nil, &resp, privateSpotRate)
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
	return resp.Result.Success, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitBatchCancelByIDs, params, nil, &resp, privateSpotRate)
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
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitOpenOrder, params, nil, &resp, privateSpotRate)
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
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitPastOrder, params, nil, &resp, privateSpotRate)
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
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitTradeHistory, params, nil, &resp, privateSpotRate)
}

// GetWalletBalance returns user wallet balance
func (by *Bybit) GetWalletBalance(ctx context.Context) ([]Balance, error) {
	resp := struct {
		Data struct {
			Balances []Balance `json:"balances"`
		} `json:"result"`
		Error
	}{}
	return resp.Data.Balances, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitWalletBalance, url.Values{}, nil, &resp, privateSpotRate)
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

// GetDepositAddressForCurrency returns deposit wallet address based upon the coin.
func (by *Bybit) GetDepositAddressForCurrency(ctx context.Context, coin string) (DepositWalletInfo, error) {
	resp := struct {
		Result DepositWalletInfo `json:"result"`
		Error
	}{}

	params := url.Values{}
	if coin == "" {
		return resp.Result, errInvalidCoin
	}
	params.Set("coin", strings.ToUpper(coin))
	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitGetDepositAddress, params, nil, &resp, publicSpotRate)
}

// WithdrawFund creates request for fund withdrawal.
func (by *Bybit) WithdrawFund(ctx context.Context, coin, chain, address, tag, amount string) (string, error) {
	resp := struct {
		Data struct {
			ID string `json:"id"`
		} `json:"result"`
		Error
	}{}

	params := make(map[string]interface{})
	params["coin"] = coin
	params["chain"] = chain
	params["address"] = address
	params["amount"] = amount
	if tag != "" {
		params["tag"] = tag
	}
	return resp.Data.ID, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, bybitWithdrawFund, nil, params, &resp, privateSpotRate)
}

// GetFeeRate returns user account fee
// Valid  category: "spot", "linear", "inverse", "option"
func (by *Bybit) GetFeeRate(ctx context.Context, category, symbol, baseCoin string) (*AccountFee, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}

	if !common.StringDataContains(validCategory, category) {
		// NOTE: Opted to fail here because if the user passes in an invalid
		// category the error returned is this
		// `Bybit raw response: {"retCode":10005,"retMsg":"Permission denied, please check your API key permissions.","result":{},"retExtInfo":{},"time":1683694010783}`
		return nil, fmt.Errorf("%w, valid category values are %v", errInvalidCategory, validCategory)
	}

	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}

	result := struct {
		Data *AccountFee `json:"result"`
		Error
	}{}

	err := by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, bybitAccountFee, params, &result, privateFeeRate)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
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
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}
	return result.GetError(false)
}

// SendAuthHTTPRequest sends an authenticated HTTP request
// If payload is non-nil then request is considered to be JSON
func (by *Bybit) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, jsonPayload map[string]interface{}, result UnmarshalTo, f request.EndpointLimit) error {
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

	if params == nil && jsonPayload == nil {
		params = url.Values{}
	}

	if jsonPayload != nil {
		jsonPayload["recvWindow"] = defaultRecvWindow
	} else if params.Get("recvWindow") == "" {
		params.Set("recvWindow", defaultRecvWindow)
	}

	err = by.SendPayload(ctx, f, func() (*request.Item, error) {
		var (
			payload       []byte
			hmacSignedStr string
			headers       = make(map[string]string)
		)

		if jsonPayload != nil {
			headers["Content-Type"] = "application/json"
			jsonPayload["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
			jsonPayload["api_key"] = creds.Key
			hmacSignedStr, err = getJSONRequestSignature(jsonPayload, creds.Secret)
			if err != nil {
				return nil, err
			}
			jsonPayload["sign"] = hmacSignedStr
			payload, err = json.Marshal(jsonPayload)
			if err != nil {
				return nil, err
			}
		} else {
			params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
			params.Set("api_key", creds.Key)
			hmacSignedStr, err = getSign(params.Encode(), creds.Secret)
			if err != nil {
				return nil, err
			}
			headers["Content-Type"] = "application/x-www-form-urlencoded"
			switch method {
			case http.MethodPost:
				params.Set("sign", hmacSignedStr)
				payload = []byte(params.Encode())
			default:
				path = common.EncodeURLValues(path, params)
				path += "&sign=" + hmacSignedStr
			}
		}

		return &request.Item{
			Method:        method,
			Path:          endpointPath + path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &result,
			Verbose:       by.Verbose,
			HTTPDebugging: by.HTTPDebugging,
			HTTPRecording: by.HTTPRecording}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	return result.GetError(true)
}

// SendAuthHTTPRequestV5 sends an authenticated HTTP request
func (by *Bybit) SendAuthHTTPRequestV5(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, result UnmarshalTo, f request.EndpointLimit) error {
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

	err = by.SendPayload(ctx, f, func() (*request.Item, error) {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		headers := make(map[string]string)
		headers["Content-Type"] = "application/x-www-form-urlencoded"
		headers["X-BAPI-TIMESTAMP"] = timestamp
		headers["X-BAPI-API-KEY"] = creds.Key
		headers["X-BAPI-RECV-WINDOW"] = defaultRecvWindow

		var hmacSignedStr string
		hmacSignedStr, err = getSign(timestamp+creds.Key+defaultRecvWindow+params.Encode(), creds.Secret)
		if err != nil {
			return nil, err
		}
		headers["X-BAPI-SIGN"] = hmacSignedStr
		return &request.Item{
			Method:        method,
			Path:          endpointPath + common.EncodeURLValues(path, params),
			Headers:       headers,
			Result:        &result,
			Verbose:       by.Verbose,
			HTTPDebugging: by.HTTPDebugging,
			HTTPRecording: by.HTTPRecording}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}

	return result.GetError(true)
}

// Error defines all error information for each request
type Error struct {
	ReturnCode      int64  `json:"ret_code"`
	ReturnMsg       string `json:"ret_msg"`
	ReturnCodeV5    int64  `json:"retCode"`
	ReturnMessageV5 string `json:"retMsg"`
	ExtCode         string `json:"ext_code"`
	ExtMsg          string `json:"ext_info"`
}

// GetError checks and returns an error if it is supplied.
func (e *Error) GetError(isAuthRequest bool) error {
	if e.ReturnCode != 0 && e.ReturnMsg != "" {
		if isAuthRequest {
			return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, e.ReturnMsg)
		}
		return errors.New(e.ReturnMsg)
	}
	if e.ReturnCodeV5 != 0 && e.ReturnMessageV5 != "" {
		return errors.New(e.ReturnMessageV5)
	}
	if e.ExtCode != "" && e.ExtMsg != "" {
		if isAuthRequest {
			return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, e.ExtMsg)
		}
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

func getJSONRequestSignature(payload map[string]interface{}, secret string) (string, error) {
	payloadArr := make([]string, len(payload))
	var i int
	for p := range payload {
		payloadArr[i] = p
		i++
	}
	sort.Strings(payloadArr)
	var signStr string
	for _, key := range payloadArr {
		if value, found := payload[key]; found {
			if v, ok := value.(string); ok {
				signStr += key + "=" + v + "&"
			}
		} else {
			return "", errors.New("non-string payload parameter not expected")
		}
	}
	return getSign(signStr[:len(signStr)-1], secret)
}

func getSign(sign, secret string) (string, error) {
	hmacSigned, err := crypto.GetHMAC(crypto.HashSHA256, []byte(sign), []byte(secret))
	if err != nil {
		return "", err
	}
	return crypto.HexEncodeToString(hmacSigned), nil
}
