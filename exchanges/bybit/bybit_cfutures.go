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
func (by *Bybit) GetFuturesOrderbook(symbol currency.Pair, limit int64) (Orderbook, error) {
	var resp Orderbook
	var data []OrderbookData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	path := common.EncodeURLValues(cfuturesOrderbook, params)
	err = by.SendHTTPRequest(exchange.RestCoinMargined, path, &data)
	if err != nil {
		return resp, err
	}
	var price, quantity float64

	for _, ob := range data {
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

// GetFuturesSymbolPriceTicker gets price ticker for symbol
func (by *Bybit) GetFuturesSymbolPriceTicker(symbol currency.Pair, pair string) ([]SymbolPriceTicker, error) {
	var resp []SymbolPriceTicker
	params := url.Values{}

	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	if pair != "" {
		params.Set("pair", pair)
	}

	path := common.EncodeURLValues(cfuturesSymbolPriceTicker, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetFuturesKlineData gets futures kline data for CoinMarginedFutures,
func (by *Bybit) GetFuturesKlineData(symbol currency.Pair, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	var data [][10]interface{}
	var resp []FuturesCandleStick
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if limit > 0 && limit <= 1500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}

	path := common.EncodeURLValues(cfuturesKline, params)
	err := by.SendHTTPRequest(exchange.RestCoinMargined, path, &data)
	if err != nil {
		return resp, err
	}
	var floatData float64
	var strData string
	var ok bool
	var tempData FuturesCandleStick
	for x := range data {
		floatData, ok = data[x][0].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for open time")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return resp, errors.New("type assertion failed for open")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return resp, errors.New("type assertion failed for high")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return resp, errors.New("type assertion failed for low")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return resp, errors.New("type assertion failed for close")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return resp, errors.New("type assertion failed for volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for close time")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return resp, errors.New("type assertion failed for base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for taker buy volume")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return resp, errors.New("type assertion failed for taker buy base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp = append(resp, tempData)
	}
	return resp, nil
}
