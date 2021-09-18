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

// GetFuturesOrderbook gets orderbook data for CoinMarginedFutures
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

// GetFuturesKlineData gets futures kline data for CoinMarginedFutures,
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

// GetFuturesSymbolPriceTicker gets price ticker for symbol
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
