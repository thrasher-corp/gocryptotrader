package bybit

import (
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	bybitFuturesAPIVersion = "v2"

	// public endpoint
	cfuturesOrderbook         = "/public/orderBook/L2"
	cfuturesKline             = "public/kline/list"
	cfuturesSymbolPriceTicker = "/public/tickers"
	cfuturesRecentTrades      = "/public/trading-records"
	cfuturesSymbolInfo        = "/public/symbols"
	cfuturesLiquidationOrders = "/public/liq-records"
	cfuturesMarkPriceKline    = "/public/mark-price-kline"
	cfuturesIndexKline        = "/public/index-price-kline"
	cfuturesIndexPremiumKline = "/public/premium-index-kline"
	cfuturesOpenInterest      = "/public/open-interest"
	cfuturesBigDeal           = "/public/big-deal"
	cfuturesAccountRatio      = "/public/account-ratio"

	// auth endpoint
)

// GetFuturesOrderbook gets orderbook data for CoinMarginedFutures.
func (by *Bybit) GetFuturesOrderbook(symbol currency.Pair) (Orderbook, error) {
	var resp Orderbook
	var data []OrderbookData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(cfuturesOrderbook, params)
	err = by.SendHTTPRequest(exchange.RestCoinMargined, path, &data)
	if err != nil {
		return resp, err
	}

	for _, ob := range data {
		var price, quantity float64
		price, err = strconv.ParseFloat(ob.Price, 64)
		if err != nil {
			return resp, err
		}

		quantity = float64(ob.Size)
		if ob.Side == sideBuy {
			resp.Bids = append(resp.Bids, OrderbookItem{
				Price:  price,
				Amount: quantity,
			})
		} else {
			resp.Asks = append(resp.Asks, OrderbookItem{
				Price:  price,
				Amount: quantity,
			})
		}
	}
	return resp, nil
}

// GetFuturesKlineData gets futures kline data for CoinMarginedFutures.
func (by *Bybit) GetFuturesKlineData(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]FuturesCandleStick, error) {
	var resp []FuturesCandleStick
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	} else {
		return resp, errors.New("symbol missing")
	}

	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)

	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(cfuturesKline, params)
	err := by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// GetFuturesSymbolPriceTicker gets price ticker for symbol.
func (by *Bybit) GetFuturesSymbolPriceTicker(symbol currency.Pair) ([]SymbolPriceTicker, error) {
	var resp []SymbolPriceTicker
	params := url.Values{}

	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(cfuturesSymbolPriceTicker, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetPublicTrades gets past public trades for CoinMarginedFutures.
func (by *Bybit) GetPublicTrades(symbol currency.Pair, limit, fromID int64) ([]FuturesPublicTradesData, error) {
	var resp []FuturesPublicTradesData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatInt(fromID, 10))
	}

	path := common.EncodeURLValues(cfuturesRecentTrades, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetSymbolsInfo gets all symbol pair information for CoinMarginedFutures.
func (by *Bybit) GetSymbolsInfo(symbol currency.Pair, limit, fromID int64) ([]SymbolInfo, error) {
	var resp []SymbolInfo
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, cfuturesSymbolInfo, &resp)
}

// GetFuturesLiquidationOrders gets liquidation orders
func (by *Bybit) GetFuturesLiquidationOrders(symbol currency.Pair, fromID, limit int64, startTime, endTime time.Time) ([]AllLiquidationOrders, error) {
	var resp []AllLiquidationOrders
	params := url.Values{}

	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	} else {
		return resp, errors.New("symbol missing")
	}
	if fromID != 0 {
		params.Set("from", strconv.FormatInt(fromID, 10))
	}
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix()*1000, 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix()*1000, 10))
	}

	path := common.EncodeURLValues(cfuturesLiquidationOrders, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetMarkPriceKline gets mark price kline data
func (by *Bybit) GetMarkPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]MarkPriceKlineData, error) {
	var resp []MarkPriceKlineData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(cfuturesMarkPriceKline, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetIndexPriceKline gets index price kline data
func (by *Bybit) GetIndexPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]IndexPriceKlineData, error) {
	var resp []IndexPriceKlineData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(cfuturesIndexKline, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetPremiumIndexPriceKline gets premium index price kline data
func (by *Bybit) GetPremiumIndexPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]IndexPriceKlineData, error) {
	var resp []IndexPriceKlineData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(cfuturesIndexPremiumKline, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetOpenInterest gets open interest data for a symbol.
func (by *Bybit) GetOpenInterest(symbol currency.Pair, period string, limit int64) (OpenInterestData, error) {
	var resp OpenInterestData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp, errors.New("invalid period parsed")
	}
	params.Set("period", period)

	path := common.EncodeURLValues(cfuturesOpenInterest, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetLatestBigDeal gets filled orders worth more than 500,000 USD within the last 24h for symbol.
func (by *Bybit) GetLatestBigDeal(symbol currency.Pair, limit int64) (BigDealData, error) {
	var resp BigDealData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	path := common.EncodeURLValues(cfuturesBigDeal, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetAccountRatio gets user accounts long-short ratio.
func (by *Bybit) GetAccountRatio(symbol currency.Pair, period string, limit int64) (AccountRatioData, error) {
	var resp AccountRatioData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp, errors.New("invalid period parsed")
	}
	params.Set("period", period)

	path := common.EncodeURLValues(cfuturesAccountRatio, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}
