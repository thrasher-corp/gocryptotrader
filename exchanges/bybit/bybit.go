package bybit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Bybit is the overarching type across this package
type Bybit struct {
	exchange.Base
}

const (
	bybitAPIURL     = "https://api.bybit.com"
	bybitAPIVersion = "v1"

	sideBuy  = "BUY"
	sideSell = "SELL"

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
func (by *Bybit) GetAllPairs() ([]PairData, error) {
	resp := struct {
		Data []PairData `json:"result"`
	}{}
	return resp.Data, by.SendHTTPRequest(exchange.RestSpot, bybitSpotGetSymbols, &resp)
}

// GetOrderbook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetOrderBook(symbol string, depth int64) (Orderbook, error) {
	resp := struct {
		Data struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
			Time int64      `json:"time"`
		} `json:"result"`
	}{}

	strDepth := "100" // default depth
	if depth > 0 && depth < 100 {
		strDepth = strconv.FormatInt(depth, 10)
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strDepth)
	path := common.EncodeURLValues(bybitOrderBook, params)
	err := by.SendHTTPRequest(exchange.RestSpot, path, &resp)
	if err != nil {
		return Orderbook{}, err
	}

	processOB := func(ob [][]string) ([]OrderbookItem, error) {
		var o []OrderbookItem
		for x := range ob {
			var price, amount float64
			amount, err = strconv.ParseFloat(ob[x][1], 64)
			if err != nil {
				return nil, err
			}
			price, err = strconv.ParseFloat(ob[x][0], 64)
			if err != nil {
				return nil, err
			}
			o = append(o, OrderbookItem{
				Price:  price,
				Amount: amount,
			})
		}
		return o, nil
	}

	var s Orderbook
	s.Bids, err = processOB(resp.Data.Bids)
	if err != nil {
		return s, err
	}
	s.Asks, err = processOB(resp.Data.Asks)
	if err != nil {
		return s, err
	}
	s.Time = time.Unix(0, resp.Data.Time*int64(time.Millisecond))
	return s, nil
}

// GetMergedOrderBook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetMergedOrderBook(symbol string, scale, depth int64) (Orderbook, error) {
	resp := struct {
		Data struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
			Time int64      `json:"time"`
		} `json:"result"`
	}{}

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
	err := by.SendHTTPRequest(exchange.RestSpot, path, &resp)
	if err != nil {
		return Orderbook{}, err
	}

	processOB := func(ob [][]string) ([]OrderbookItem, error) {
		var o []OrderbookItem
		for x := range ob {
			var price, amount float64
			amount, err = strconv.ParseFloat(ob[x][1], 64)
			if err != nil {
				return nil, err
			}
			price, err = strconv.ParseFloat(ob[x][0], 64)
			if err != nil {
				return nil, err
			}
			o = append(o, OrderbookItem{
				Price:  price,
				Amount: amount,
			})
		}
		return o, nil
	}

	var s Orderbook
	s.Bids, err = processOB(resp.Data.Bids)
	if err != nil {
		return s, err
	}
	s.Asks, err = processOB(resp.Data.Asks)
	if err != nil {
		return s, err
	}
	s.Time = time.Unix(0, resp.Data.Time*int64(time.Millisecond))
	return s, nil
}

// GetTrades gets recent trades from the exchange
func (by *Bybit) GetTrades(symbol string, limit int64) ([]TradeItem, error) {
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
	err := by.SendHTTPRequest(exchange.RestSpot, path, &resp)
	if err != nil {
		return nil, err
	}

	var trades []TradeItem
	for x := range resp.Data {
		tradeSide := ""
		if resp.Data[x].IsBuyerMaker {
			tradeSide = sideBuy
		} else {
			tradeSide = sideSell
		}

		trades = append(trades, TradeItem{
			CurrencyPair: symbol,
			Price:        resp.Data[x].Price,
			Side:         tradeSide,
			Volume:       resp.Data[x].Quantity,
			TradeTime:    time.Unix(0, resp.Data[x].Time*int64(time.Millisecond)),
		})
	}
	return trades, nil
}

// GetKlines data returns the kline data for a specific symbol
func (by *Bybit) GetKlines(symbol, period string, limit int64, start, end time.Time) ([]KlineItem, error) {
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
	if err := by.SendHTTPRequest(exchange.RestSpot, path, &resp); err != nil {
		return nil, err
	}

	var klines []KlineItem
	for x := range resp.Data {
		if len(resp.Data[x]) != 11 {
			return klines, fmt.Errorf("%v GetKlines: invalid response, array length not as expected, check api docs for updates", by.Name)
		}
		var (
			kline                                                                               KlineItem
			ok                                                                                  bool
			err                                                                                 error
			startTime, endTime, tradesCount                                                     float64
			open, high, low, close, volume, quoteAssetVolume, takerBaseVolume, takerQuoteVolume string
		)

		if startTime, ok = resp.Data[x][0].(float64); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for StartTime", by.Name, errTypeAssert)
		}
		kline.StartTime = time.Unix(0, int64(startTime)*int64(time.Millisecond))

		if open, ok = resp.Data[x][1].(string); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for Open", by.Name, errTypeAssert)
		}
		if kline.Open, err = strconv.ParseFloat(open, 64); err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for Open", by.Name, errStrParsing)
		}

		if high, ok = resp.Data[x][2].(string); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for High", by.Name, errTypeAssert)
		}
		if kline.High, err = strconv.ParseFloat(high, 64); err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for High", by.Name, errStrParsing)
		}

		if low, ok = resp.Data[x][3].(string); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for Low", by.Name, errTypeAssert)
		}
		if kline.Low, err = strconv.ParseFloat(low, 64); err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for Low", by.Name, errStrParsing)
		}

		if close, ok = resp.Data[x][4].(string); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for Close", by.Name, errTypeAssert)
		}
		if kline.Close, err = strconv.ParseFloat(close, 64); err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for Close", by.Name, errStrParsing)
		}

		if volume, ok = resp.Data[x][5].(string); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for Volume", by.Name, errTypeAssert)
		}
		if kline.Volume, err = strconv.ParseFloat(volume, 64); err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for Volume", by.Name, errStrParsing)
		}

		if endTime, ok = resp.Data[x][6].(float64); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for EndTime", by.Name, errTypeAssert)
		}
		kline.EndTime = time.Unix(0, int64(endTime)*int64(time.Millisecond))

		if quoteAssetVolume, ok = resp.Data[x][7].(string); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for QuoteAssetVolume", by.Name, errTypeAssert)
		}
		if kline.QuoteAssetVolume, err = strconv.ParseFloat(quoteAssetVolume, 64); err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for QuoteAssetVolume", by.Name, errStrParsing)
		}

		if tradesCount, ok = resp.Data[x][8].(float64); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for TradesCount", by.Name, errTypeAssert)
		}
		kline.TradesCount = int64(tradesCount)

		if takerBaseVolume, ok = resp.Data[x][9].(string); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for TakerBaseVolume", by.Name, errTypeAssert)
		}
		if kline.TakerBaseVolume, err = strconv.ParseFloat(takerBaseVolume, 64); err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for TakerBaseVolume", by.Name, errStrParsing)
		}

		if takerQuoteVolume, ok = resp.Data[x][10].(string); !ok {
			return klines, fmt.Errorf("%v GetKlines: %w for TakerQuoteVolume", by.Name, errTypeAssert)
		}
		if kline.TakerQuoteVolume, err = strconv.ParseFloat(takerQuoteVolume, 64); err != nil {
			return klines, fmt.Errorf("%v GetKlines: %w for TakerQuoteVolume", by.Name, errStrParsing)
		}

		klines = append(klines, kline)
	}
	return klines, nil
}

// Get24HrsChange returns price change statistics for the last 24 hours
// If symbol not passed then it will return price change statistics for all pairs
func (by *Bybit) Get24HrsChange(symbol string) ([]PriceChangeStats, error) {
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
		err := by.SendHTTPRequest(exchange.RestSpot, path, &resp)
		if err != nil {
			return nil, err
		}

		stats = append(stats, PriceChangeStats{
			time.Unix(0, resp.Data.Time*int64(time.Millisecond)),
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

		err := by.SendHTTPRequest(exchange.RestSpot, bybit24HrsChange, &resp)
		if err != nil {
			return nil, err
		}

		for x := range resp.Data {
			stats = append(stats, PriceChangeStats{
				time.Unix(0, resp.Data[x].Time*int64(time.Millisecond)),
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
func (by *Bybit) GetLastTradedPrice(symbol string) ([]LastTradePrice, error) {
	var lastTradePrices []LastTradePrice
	if symbol != "" {
		resp := struct {
			Data LastTradePrice `json:"result"`
		}{}

		params := url.Values{}
		params.Set("symbol", symbol)
		path := common.EncodeURLValues(bybitLastTradedPrice, params)
		err := by.SendHTTPRequest(exchange.RestSpot, path, &resp)
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

		err := by.SendHTTPRequest(exchange.RestSpot, bybitLastTradedPrice, &resp)
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
func (by *Bybit) GetBestBidAskPrice(symbol string) ([]TickerData, error) {
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
		err := by.SendHTTPRequest(exchange.RestSpot, path, &resp)
		if err != nil {
			return nil, err
		}
		tickers = append(tickers, TickerData{
			resp.Data.Symbol,
			resp.Data.BidPrice,
			resp.Data.BidQuantity,
			resp.Data.AskPrice,
			resp.Data.AskQuantity,
			time.Unix(0, resp.Data.Time*int64(time.Millisecond)),
		})
	} else {
		resp := struct {
			Data []bestTicker `json:"result"`
		}{}

		err := by.SendHTTPRequest(exchange.RestSpot, bybitBestBidAskPrice, &resp)
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
				time.Unix(0, resp.Data[x].Time*int64(time.Millisecond)),
			})
		}
	}
	return tickers, nil
}

func (by *Bybit) CreatePostOrder(o *PlaceOrderRequest) (*PlaceOrderResponse, error) {
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
		params.Set("orderLinkId", string(o.OrderLinkID))
	}

	resp := struct {
		Data PlaceOrderResponse `json:"result"`
	}{}
	err := by.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, bybitSpotOrder, params, resp, bybitAuthRate)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (by *Bybit) QueryOrder(orderID, orderLinkID string) (*QueryOrderResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, errors.New("atleast one should be present among orderID and orderLinkID")
	}

	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderID != "" {
		params.Set("orderLinkId", orderLinkID)
	}

	resp := struct {
		Data QueryOrderResponse `json:"result"`
	}{}
	err := by.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, bybitSpotOrder, params, resp, bybitAuthRate)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (by *Bybit) CancelExistingOrder(orderID, orderLinkID string) (*CancelOrderResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, errors.New("atleast one should be present among orderID and orderLinkID")
	}

	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderID != "" {
		params.Set("orderLinkId", orderLinkID)
	}

	resp := struct {
		Data CancelOrderResponse `json:"result"`
	}{}
	err := by.SendAuthHTTPRequest(exchange.RestSpot, http.MethodDelete, bybitSpotOrder, params, resp, bybitAuthRate)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (by *Bybit) BatchCancelOrder(symbol, side, orderTypes string) (bool, error) {
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
	err := by.SendAuthHTTPRequest(exchange.RestSpot, http.MethodDelete, bybitBatchCancelSpotOrder, params, resp, bybitAuthRate)
	if err != nil {
		return false, err
	}
	return resp.Success, nil
}

func (by *Bybit) ListOpenOrders(symbol, orderID string, limit int64) ([]QueryOrderResponse, error) {
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
	err := by.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, bybitOpenOrder, params, resp, bybitAuthRate)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (by *Bybit) ListPastOrders(symbol, orderID string, limit int64) ([]QueryOrderResponse, error) {
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
	err := by.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, bybitPastOrder, params, resp, bybitAuthRate)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (by *Bybit) GetTradeHistory(symbol string, limit, formID, told int64) ([]HistoricalTrade, error) {
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
	err := by.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, bybitPastOrder, params, resp, bybitAuthRate)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (by *Bybit) GetWalletBalance() ([]Balance, error) {
	resp := struct {
		Data struct {
			Balances []Balance `json:"balances"`
		} `json:"result"`
	}{}
	err := by.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, bybitWalletBalance, url.Values{}, resp, bybitAuthRate)
	if err != nil {
		return nil, err
	}
	return resp.Data.Balances, nil
}

// SendHTTPRequest sends an unauthenticated request
func (by *Bybit) SendHTTPRequest(ePath exchange.URL, path string, result interface{}) error {
	endpointPath, err := by.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	return by.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Result:        result,
		Verbose:       by.Verbose,
		HTTPDebugging: by.HTTPDebugging,
		HTTPRecording: by.HTTPRecording,
	})
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (by *Bybit) SendAuthHTTPRequest(ePath exchange.URL, method, path string, params url.Values, result interface{}, f request.EndpointLimit) error {
	endpointPath, err := by.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}

	recvWindow := 5 * time.Second
	if params.Get("recvWindow") != "" {
		// convert recvWindow value into time.Duration
		var recvWindowParam int64
		recvWindowParam, err = convert.Int64FromString(params.Get("recvWindow"))
		if err != nil {
			return err
		}
		recvWindow = time.Duration(recvWindowParam) * time.Millisecond
	} else {
		params.Set("recvWindow", strconv.FormatInt(convert.RecvWindow(recvWindow), 10))
	}
	params.Set("recvWindow", strconv.FormatInt(convert.RecvWindow(recvWindow), 10))
	params.Set("timestamp", strconv.FormatInt(time.Now().Unix()*1000, 10))
	params.Set("api_key", by.API.Credentials.Key)
	signature := params.Encode()
	hmacSigned := crypto.GetHMAC(crypto.HashSHA256, []byte(signature), []byte(by.API.Credentials.Secret))
	hmacSignedStr := crypto.HexEncodeToString(hmacSigned)
	if by.Verbose {
		log.Debugf(log.ExchangeSys, "sent path: %s", path)
	}

	path = common.EncodeURLValues(path, params)
	path += "&sign=" + hmacSignedStr

	headers := make(map[string]string)

	switch {
	case method == http.MethodPost:
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}

	return by.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          endpointPath + path,
		Headers:       headers,
		Body:          bytes.NewBuffer(nil),
		Result:        &result,
		AuthRequest:   true,
		Verbose:       by.Verbose,
		HTTPDebugging: by.HTTPDebugging,
		HTTPRecording: by.HTTPRecording,
		Endpoint:      f,
	})
}
