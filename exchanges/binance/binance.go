package binance

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
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Binance is the overarching type across the Binance package
type Binance struct {
	exchange.Base
	USDTWS              stream.WebsocketConnection
	CoinMarginFuturesWS stream.WebsocketConnection
	// Valid string list that is required by the exchange
	validLimits []int
}

const (
	apiURL         = "https://api.binance.com"
	spotAPIURL     = "https://sapi.binance.com"
	cfuturesAPIURL = "https://dapi.binance.com"
	ufuturesAPIURL = "https://fapi.binance.com"

	// Public endpoints

	// Futures
	cfuturesExchangeInfo       = "/dapi/v1/exchangeInfo?"
	cfuturesOrderbook          = "/dapi/v1/depth?"
	cfuturesRecentTrades       = "/dapi/v1/trades?"
	cfuturesHistoricalTrades   = "/dapi/v1/historicalTrades?"
	cfuturesCompressedTrades   = "/dapi/v1/aggTrades?"
	cfuturesKlineData          = "/dapi/v1/klines?"
	cfuturesContinuousKline    = "/dapi/v1/continuousKlines?"
	cfuturesIndexKline         = "/dapi/v1/indexPriceKlines?"
	cfuturesMarkPriceKline     = "/dapi/v1/markPriceKlines?"
	cfuturesMarkPrice          = "/dapi/v1/premiumIndex?"
	cfuturesFundingRateHistory = "/dapi/v1/fundingRate?"
	cfuturesTickerPriceStats   = "/dapi/v1/ticker/24hr?"
	cfuturesSymbolPriceTicker  = "/dapi/v1/ticker/price?"
	cfuturesSymbolOrderbook    = "/dapi/v1/ticker/bookTicker?"
	cfuturesLiquidationOrders  = "/dapi/v1/allForceOrders?"
	cfuturesOpenInterest       = "/dapi/v1/openInterest?"
	cfuturesOpenInterestStats  = "/futures/data/openInterestHist?"
	cfuturesTopAccountsRatio   = "/futures/data/topLongShortAccountRatio?"
	cfuturesTopPositionsRatio  = "/futures/data/topLongShortPositionRatio?"
	cfuturesLongShortRatio     = "/futures/data/globalLongShortAccountRatio?"
	cfuturesBuySellVolume      = "/futures/data/takerBuySellVol?"
	cfuturesBasis              = "/futures/data/basis?"

	ufuturesExchangeInfo       = "/fapi/v1/exchangeInfo?"
	ufuturesOrderbook          = "/fapi/v1/depth?"
	ufuturesRecentTrades       = "/fapi/v1/trades?"
	ufuturesHistoricalTrades   = "/fapi/v1/historicalTrades?"
	ufuturesCompressedTrades   = "/fapi/v1/aggTrades?"
	ufuturesKlineData          = "/fapi/v1/klines?"
	ufuturesMarkPrice          = "/fapi/v1/premiumIndex?"
	ufuturesFundingRateHistory = "/fapi/v1/fundingRate?"
	ufuturesTickerPriceStats   = "/fapi/v1/ticker/24hr?"
	ufuturesSymbolPriceTicker  = "/fapi/v1/ticker/price?"
	ufuturesSymbolOrderbook    = "/fapi/v1/ticker/bookTicker?"
	ufuturesLiquidationOrders  = "/fapi/v1/allForceOrders?"
	ufuturesOpenInterest       = "/fapi/v1/openInterest?"
	ufuturesOpenInterestStats  = "/futures/data/openInterestHist?"
	ufuturesTopAccountsRatio   = "/futures/data/topLongShortAccountRatio?"
	ufuturesTopPositionsRatio  = "/futures/data/topLongShortPositionRatio?"
	ufuturesLongShortRatio     = "/futures/data/globalLongShortAccountRatio?"
	ufuturesBuySellVolume      = "/futures/data/takerBuySellVol?"

	// Spot
	exchangeInfo      = "/api/v3/exchangeInfo"
	orderBookDepth    = "/api/v3/depth"
	recentTrades      = "/api/v3/trades"
	historicalTrades  = "/api/v3/historicalTrades"
	aggregatedTrades  = "/api/v3/aggTrades"
	candleStick       = "/api/v3/klines"
	averagePrice      = "/api/v3/avgPrice"
	priceChange       = "/api/v3/ticker/24hr"
	symbolPrice       = "/api/v3/ticker/price"
	bestPrice         = "/api/v3/ticker/bookTicker"
	accountInfo       = "/api/v3/account"
	userAccountStream = "/api/v3/userDataStream"
	fundingRate       = "/fapi/v1/fundingRate?"
	perpExchangeInfo  = "/fapi/v1/exchangeInfo"

	// Authenticated endpoints

	// coin margined futures
	cfuturesOrder                 = "/dapi/v1/order"
	cfuturesBatchOrder            = "/dapi/v1/batchOrders"
	cfuturesCancelAllOrders       = "/dapi/v1/allOpenOrders"
	cfuturesCountdownCancel       = "/dapi/v1/countdownCancelAll"
	cfuturesOpenOrder             = "/dapi/v1/openOrder"
	cfuturesAllOpenOrders         = "/dapi/v1/openOrders"
	cfuturesAllOrders             = "/dapi/v1/allOrders"
	cfuturesAccountBalance        = "/dapi/v1/balance"
	cfuturesAccountInfo           = "/dapi/v1/account"
	cfuturesChangeInitialLeverage = "/dapi/v1/leverage"
	cfuturesChangeMarginType      = "/dapi/v1/marginType"
	cfuturesModifyMargin          = "/dapi/v1/positionMargin"
	cfuturesMarginChangeHistory   = "/dapi/v1/positionMargin/history"
	cfuturesPositionInfo          = "/dapi/v1/positionRisk"
	cfuturesAccountTradeList      = "/dapi/v1/userTrades"
	cfuturesIncomeHistory         = "/dapi/v1/income"
	cfuturesNotionalBracket       = "/dapi/v1/leverageBracket"
	cfuturesUsersForceOrders      = "/dapi/v1/forceOrders"
	cfuturesADLQuantile           = "/dapi/v1/adlQuantile"

	// usdt margined futures
	ufuturesOrder                 = "/fapi/v1/order"
	ufuturesBatchOrder            = "/fapi/v1/batchOrders"
	ufuturesCancelAllOrders       = "/fapi/v1/allOpenOrders"
	ufuturesCountdownCancel       = "/fapi/v1/countdownCancelAll"
	ufuturesOpenOrder             = "/fapi/v1/openOrder"
	ufuturesAllOpenOrders         = "/fapi/v1/openOrders"
	ufuturesAllOrders             = "/fapi/v1/allOrders"
	ufuturesAccountBalance        = "/fapi/v2/balance"
	ufuturesAccountInfo           = "/fapi/v2/account"
	ufuturesChangeInitialLeverage = "/fapi/v1/leverage"
	ufuturesChangeMarginType      = "/fapi/v1/marginType"
	ufuturesModifyMargin          = "/fapi/v1/positionMargin"
	ufuturesMarginChangeHistory   = "/fapi/v1/positionMargin/history"
	ufuturesPositionInfo          = "/fapi/v2/positionRisk"
	ufuturesAccountTradeList      = "/fapi/v1/userTrades"
	ufuturesIncomeHistory         = "/fapi/v1/income"
	ufuturesNotionalBracket       = "/fapi/v1/leverageBracket"
	ufuturesUsersForceOrders      = "/fapi/v1/forceOrders"
	ufuturesADLQuantile           = "/fapi/v1/adlQuantile"

	// spot
	newOrderTest  = "/api/v3/order/test"
	orderEndpoint = "/api/v3/order"
	openOrders    = "/api/v3/openOrders"
	allOrders     = "/api/v3/allOrders"

	// Withdraw API endpoints
	withdrawEndpoint            = "/wapi/v3/withdraw.html"
	depositHistory              = "/wapi/v3/depositHistory.html"
	withdrawalHistory           = "/wapi/v3/withdrawHistory.html"
	depositAddress              = "/wapi/v3/depositAddress.html"
	accountStatus               = "/wapi/v3/accountStatus.html"
	systemStatus                = "/wapi/v3/systemStatus.html"
	dustLog                     = "/wapi/v3/userAssetDribbletLog.html"
	tradeFee                    = "/wapi/v3/tradeFee.html"
	assetDetail                 = "/wapi/v3/assetDetail.html"
	undocumentedInterestHistory = "/gateway-api/v1/public/isolated-margin/pair/vip-level"
)

var (
	validFuturesIntervals = []string{
		"1m", "3m", "5m", "15m", "30m",
		"1h", "2h", "4h", "6h", "8h",
		"12h", "1d", "3d", "1w", "1M",
	}

	validContractType = []string{
		"ALL", "CURRENT_QUARTER", "NEXT_QUARTER",
	}

	validOrderType = []string{
		"LIMIT", "MARKET", "STOP", "TAKE_PROFIT",
		"STOP_MARKET", "TAKE_PROFIT_MARKET", "TRAILING_STOP_MARKET",
	}

	validNewOrderRespType = []string{"ACK", "RESULT"}

	validWorkingType = []string{"MARK_PRICE", "CONTRACT_TYPE"}

	validPositionSide = []string{"BOTH", "LONG", "SHORT"}

	validMarginType = []string{"ISOLATED", "CROSSED"}

	validIncomeType = []string{"TRANSFER", "WELCOME_BONUS", "REALIZED_PNL", "FUNDING_FEE", "COMMISSION", "INSURANCE_CLEAR"}

	validAutoCloseTypes = []string{"LIQUIDATION", "ADL"}

	validMarginChange = map[string]int64{
		"add":    1,
		"reduce": 2,
	}

	uValidOBLimits = []string{"5", "10", "20", "50", "100", "500", "1000"}

	uValidPeriods = []string{"5m", "15m", "30m", "1h", "2h", "4h", "6h", "12h", "1d"}
)

// UExchangeInfo stores futures data
func (b *Binance) UExchangeInfo() (UFuturesExchangeInfo, error) {
	var resp UFuturesExchangeInfo
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesExchangeInfo, limitDefault, &resp)
}

// UFuturesOrderbook gets orderbook data for uFutures
func (b *Binance) UFuturesOrderbook(symbol string, limit int64) (OrderBook, error) {
	var resp OrderBook
	var data OrderbookData
	params := url.Values{}
	params.Set("symbol", symbol)
	strLimit := strconv.FormatInt(limit, 10)
	if strLimit != "" {
		if !common.StringDataCompare(uValidOBLimits, strLimit) {
			return resp, fmt.Errorf("invalid limit: %v", limit)
		}
		params.Set("limit", strLimit)
	}
	err := b.SendHTTPRequest(exchange.Running+uFutures, ufuturesOrderbook+params.Encode(), limitDefault, &data)
	if err != nil {
		return resp, err
	}
	resp.Symbol = symbol
	resp.LastUpdateID = data.LastUpdateID
	var price, quantity float64
	for x := range data.Asks {
		price, err = strconv.ParseFloat(data.Asks[x][0], 64)
		if err != nil {
			return resp, err
		}
		quantity, err = strconv.ParseFloat(data.Asks[x][1], 64)
		if err != nil {
			return resp, err
		}
		resp.Asks = append(resp.Asks, OrderbookItem{
			Price:    price,
			Quantity: quantity,
		})
	}
	for y := range data.Bids {
		price, err = strconv.ParseFloat(data.Bids[y][0], 64)
		if err != nil {
			return resp, err
		}
		quantity, err = strconv.ParseFloat(data.Bids[y][1], 64)
		if err != nil {
			return resp, err
		}
		resp.Bids = append(resp.Bids, OrderbookItem{
			Price:    price,
			Quantity: quantity,
		})
	}
	return resp, nil
}

// URecentTrades gets recent trades for uFutures
func (b *Binance) URecentTrades(symbol, fromID string, limit int64) ([]UPublicTradesData, error) {
	var resp []UPublicTradesData
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesRecentTrades+params.Encode(), limitDefault, &resp)
}

// UHistoricalTrades gets historical public trades for uFutures
func (b *Binance) UHistoricalTrades(symbol, fromID string, limit int64) ([]UPublicTradesData, error) {
	var resp []UPublicTradesData
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesHistoricalTrades+params.Encode(), limitDefault, &resp)
}

// UCompressedTrades gets compressed public trades for uFutures
func (b *Binance) UCompressedTrades(symbol, fromID string, limit int64, startTime, endTime time.Time) ([]UCompressedTradeData, error) {
	var resp []UCompressedTradeData
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesCompressedTrades+params.Encode(), limitDefault, &resp)
}

// UKlineData gets kline data for uFutures
func (b *Binance) UKlineData(symbol, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	var data [][]interface{}
	var resp []FuturesCandleStick
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(uValidPeriods, interval) {
		return resp, errors.New("invalid interval")
	}
	params.Set("interval", interval)
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	err := b.SendHTTPRequest(exchange.Running+uFutures, ufuturesKlineData+params.Encode(), limitDefault, &data)
	if err != nil {
		return resp, err
	}
	var tempData FuturesCandleStick
	var floatData float64
	var strData string
	var ok bool
	for x := range data {
		floatData, ok = data[x][0].(float64)
		if !ok {
			return resp, errors.New("type casting failed for opentime")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return resp, errors.New("type casting failed for open")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return resp, errors.New("type casting failed for high")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return resp, errors.New("type casting failed for low")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return resp, errors.New("type casting failed for close")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return resp, errors.New("type casting failed for volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return resp, errors.New("type casting failed for ")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return resp, errors.New("type casting failed base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return resp, errors.New("type casting failed for taker buy volume")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return resp, errors.New("type casting failed for taker buy base asset volume")
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

// UGetMarkPrice gets mark price data for USDTMarginedFutures
func (b *Binance) UGetMarkPrice(symbol string) ([]UMarkPrice, error) {
	var data json.RawMessage
	var resp []UMarkPrice
	var singleResp bool
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
		singleResp = true
	}
	err := b.SendHTTPRequest(exchange.Running+uFutures, ufuturesMarkPrice+params.Encode(), limitDefault, &data)
	if err != nil {
		return resp, err
	}
	if singleResp {
		var tempResp UMarkPrice
		err := json.Unmarshal(data, &tempResp)
		if err != nil {
			return resp, err
		}
		resp = append(resp, tempResp)
	} else {
		err := json.Unmarshal(data, &resp)
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// UGetFundingHistory gets funding history for USDTMarginedFutures
func (b *Binance) UGetFundingHistory(symbol string, limit int64, startTime, endTime time.Time) ([]FundingRateHistory, error) {
	var resp []FundingRateHistory
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesFundingRateHistory+params.Encode(), limitDefault, &resp)
}

// U24HTickerPriceChangeStats gets 24hr ticker price change stats for USDTMarginedFutures
func (b *Binance) U24HTickerPriceChangeStats(symbol string) ([]U24HrPriceChangeStats, error) {
	var data json.RawMessage
	var resp []U24HrPriceChangeStats
	var singleResp bool
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
		singleResp = true
	}
	err := b.SendHTTPRequest(exchange.Running+uFutures, ufuturesTickerPriceStats+params.Encode(), limitDefault, &data)
	if err != nil {
		return resp, err
	}
	if singleResp {
		var tempResp U24HrPriceChangeStats
		err := json.Unmarshal(data, &tempResp)
		if err != nil {
			return resp, err
		}
		resp = append(resp, tempResp)
	} else {
		err := json.Unmarshal(data, &resp)
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// USymbolPriceTicker gets symbol price ticker for USDTMarginedFutures
func (b *Binance) USymbolPriceTicker(symbol string) ([]USymbolPriceTicker, error) {
	var resp []USymbolPriceTicker
	var data json.RawMessage
	var singleResp bool
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
		singleResp = true
	}
	err := b.SendHTTPRequest(exchange.Running+uFutures, ufuturesSymbolPriceTicker+params.Encode(), limitDefault, &data)
	if err != nil {
		return resp, err
	}
	if singleResp {
		var tempResp USymbolPriceTicker
		err := json.Unmarshal(data, &tempResp)
		if err != nil {
			return resp, err
		}
		resp = append(resp, tempResp)
	} else {
		err := json.Unmarshal(data, &resp)
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// USymbolOrderbookTicker gets symbol orderbook ticker
func (b *Binance) USymbolOrderbookTicker(symbol string) ([]USymbolOrderbookTicker, error) {
	var resp []USymbolOrderbookTicker
	var singleResp bool
	var data json.RawMessage
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
		singleResp = true
	}
	err := b.SendHTTPRequest(exchange.Running+uFutures, ufuturesSymbolOrderbook+params.Encode(), limitDefault, &data)
	if err != nil {
		return resp, err
	}
	if singleResp {
		var tempResp USymbolOrderbookTicker
		err := json.Unmarshal(data, &tempResp)
		if err != nil {
			return resp, err
		}
		resp = append(resp, tempResp)
	} else {
		err := json.Unmarshal(data, &resp)
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// ULiquidationOrders gets public liquidation orders
func (b *Binance) ULiquidationOrders(symbol string, limit int64, startTime, endTime time.Time) ([]ULiquidationOrdersData, error) {
	var resp []ULiquidationOrdersData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesLiquidationOrders+params.Encode(), limitDefault, &resp)
}

// UOpenInterest gets open interest data for USDTMarginedFutures
func (b *Binance) UOpenInterest(symbol string) (UOpenInterestData, error) {
	var resp UOpenInterestData
	params := url.Values{}
	params.Set("symbol", symbol)
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesOpenInterest+params.Encode(), limitDefault, &resp)
}

// UOpenInterestStats gets open interest stats for USDTMarginedFutures
func (b *Binance) UOpenInterestStats(symbol, period string, limit int64, startTime, endTime time.Time) ([]UOpenInterestStats, error) {
	var resp []UOpenInterestStats
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(uValidPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesOpenInterestStats+params.Encode(), limitDefault, &resp)
}

// UTopAcccountsLongShortRatio gets long/short ratio data for top trader accounts in ufutures
func (b *Binance) UTopAcccountsLongShortRatio(symbol, period string, limit int64, startTime, endTime time.Time) ([]ULongShortRatio, error) {
	var resp []ULongShortRatio
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(uValidPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesTopAccountsRatio+params.Encode(), limitDefault, &resp)
}

// UTopPostionsLongShortRatio gets long/short ratio data for top positions' in ufutures
func (b *Binance) UTopPostionsLongShortRatio(symbol, period string, limit int64, startTime, endTime time.Time) ([]ULongShortRatio, error) {
	var resp []ULongShortRatio
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(uValidPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesTopPositionsRatio+params.Encode(), limitDefault, &resp)
}

// UGlobalLongShortRatio gets the global long/short ratio data for USDTMarginedFutures
func (b *Binance) UGlobalLongShortRatio(symbol, period string, limit int64, startTime, endTime time.Time) ([]ULongShortRatio, error) {
	var resp []ULongShortRatio
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(uValidPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesLongShortRatio+params.Encode(), limitDefault, &resp)
}

// UTakerBuySellVol gets takers' buy/sell ratio for USDTMarginedFutures
func (b *Binance) UTakerBuySellVol(symbol, period string, limit int64, startTime, endTime time.Time) ([]UTakerVolumeData, error) {
	var resp []UTakerVolumeData
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(uValidPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, ufuturesBuySellVolume+params.Encode(), limitDefault, &resp)
}

// UFuturesNewOrder sends a new order for USDTMarginedFutures
func (b *Binance) UFuturesNewOrder(symbol, side, positionSide, orderType, timeInForce,
	newClientOrderID, closePosition, workingType, newOrderRespType string,
	quantity, price, stopPrice, activationPrice, callbackRate float64, reduceOnly bool) (UOrderData, error) {
	var resp UOrderData
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side)
	if positionSide != "" {
		if !common.StringDataCompare(validPositionSide, positionSide) {
			return resp, errors.New("invalid positionSide")
		}
		params.Set("positionSide", positionSide)
	}
	params.Set("type", orderType)
	params.Set("timeInForce", timeInForce)
	if reduceOnly {
		params.Set("reduceOnly", "true")
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderID", newClientOrderID)
	}
	if closePosition != "" {
		params.Set("closePosition", closePosition)
	}
	if workingType != "" {
		if !common.StringDataCompare(validWorkingType, workingType) {
			return resp, errors.New("invalid workingType")
		}
		params.Set("workingType", workingType)
	}
	if newOrderRespType != "" {
		if !common.StringDataCompare(validNewOrderRespType, newOrderRespType) {
			return resp, errors.New("invalid newOrderRespType")
		}
		params.Set("newOrderRespType", newOrderRespType)
	}
	if quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	}
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if stopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(stopPrice, 'f', -1, 64))
	}
	if activationPrice != 0 {
		params.Set("activationPrice", strconv.FormatFloat(activationPrice, 'f', -1, 64))
	}
	if callbackRate != 0 {
		params.Set("callbackRate", strconv.FormatFloat(callbackRate, 'f', -1, 64))
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodPost, ufuturesOrder, params, limitDefault, &resp)
}

// UPlaceBatchOrders places batch orders
func (b *Binance) UPlaceBatchOrders(data []PlaceBatchOrderData) ([]UOrderData, error) {
	var resp []UOrderData
	params := url.Values{}
	for x := range data {
		if data[x].PositionSide != "" {
			if !common.StringDataCompare(validPositionSide, data[x].PositionSide) {
				return resp, errors.New("invalid positionSide")
			}
		}
		if data[x].WorkingType != "" {
			if !common.StringDataCompare(validWorkingType, data[x].WorkingType) {
				return resp, errors.New("invalid workingType")
			}
		}
		if data[x].NewOrderRespType != "" {
			if !common.StringDataCompare(validNewOrderRespType, data[x].NewOrderRespType) {
				return resp, errors.New("invalid newOrderRespType")
			}
		}
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return resp, err
	}
	params.Set("batchOrders", string(jsonData))
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodPost, ufuturesBatchOrder, params, limitDefault, &resp)
}

// UGetOrderData gets order data for USDTMarginedFutures
func (b *Binance) UGetOrderData(symbol, orderID, cliOrderID string) (UOrderData, error) {
	var resp UOrderData
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if cliOrderID != "" {
		params.Set("origClientOrderId", cliOrderID)
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesOrder, params, limitDefault, &resp)
}

// UCancelOrder cancel an order for USDTMarginedFutures
func (b *Binance) UCancelOrder(symbol, orderID, cliOrderID string) (UOrderData, error) {
	var resp UOrderData
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if cliOrderID != "" {
		params.Set("origClientOrderId", cliOrderID)
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodDelete, ufuturesOrder, params, limitDefault, &resp)
}

// UCancelAllOpenOrders cancels all open orders for a symbol ufutures
func (b *Binance) UCancelAllOpenOrders(symbol string) (GenericAuthResponse, error) {
	var resp GenericAuthResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodDelete, ufuturesCancelAllOrders, params, limitDefault, &resp)
}

// UCancelBatchOrders cancel batch order for USDTMarginedFutures
func (b *Binance) UCancelBatchOrders(symbol string, orderIDList, origCliOrdIDList []string) ([]UOrderData, error) {
	var resp []UOrderData
	params := url.Values{}
	params.Set("symbol", symbol)
	if len(orderIDList) != 0 {
		jsonOrders, err := json.Marshal(orderIDList)
		if err != nil {
			return resp, err
		}
		params.Set("orderIdList", string(jsonOrders))
	}
	if len(origCliOrdIDList) != 0 {
		jsonCliOrders, err := json.Marshal(origCliOrdIDList)
		if err != nil {
			return resp, err
		}
		params.Set("origClientOrderIdList", string(jsonCliOrders))
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodDelete, ufuturesBatchOrder, params, limitDefault, &resp)
}

// UAutoCancelAllOpenOrders auto cancels all ufutures open orders for a symbol after the set countdown time
func (b *Binance) UAutoCancelAllOpenOrders(symbol string, countdownTime int64) (AutoCancelAllOrdersData, error) {
	var resp AutoCancelAllOrdersData
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("countdownTime", strconv.FormatInt(countdownTime, 10))
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodPost, ufuturesCountdownCancel, params, limitDefault, &resp)
}

// UFetchOpenOrder sends a request to fetch open order data for USDTMarginedFutures
func (b *Binance) UFetchOpenOrder(symbol, orderID, origClientOrderID string) (UOrderData, error) {
	var resp UOrderData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesOpenOrder, params, limitDefault, &resp)
}

// UAllAccountOpenOrders gets all account's orders for USDTMarginedFutures
func (b *Binance) UAllAccountOpenOrders(symbol string) ([]UOrderData, error) {
	var resp []UOrderData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesAllOpenOrders, params, limitDefault, &resp)
}

// UAllAccountOrders gets all account's orders for USDTMarginedFutures
func (b *Binance) UAllAccountOrders(symbol string, orderID, limit int64, startTime, endTime time.Time) ([]UFuturesOrderData, error) {
	var resp []UFuturesOrderData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesAllOrders, params, limitDefault, &resp)
}

// UAccountBalanceV2 gets V2 account balance data
func (b *Binance) UAccountBalanceV2() ([]UAccountBalanceV2Data, error) {
	var resp []UAccountBalanceV2Data
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesAccountBalance, nil, limitDefault, &resp)
}

// UAccountInformationV2 gets V2 account balance data
func (b *Binance) UAccountInformationV2() (UAccountInformationV2Data, error) {
	var resp UAccountInformationV2Data
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesAccountInfo, nil, limitDefault, &resp)
}

// UChangeInitialLeverageRequest sends a request to change account's initial leverage
func (b *Binance) UChangeInitialLeverageRequest(symbol string, leverage int64) (UChangeInitialLeverage, error) {
	var resp UChangeInitialLeverage
	params := url.Values{}
	params.Set("symbol", symbol)
	if !(leverage > 0 && leverage < 25) {
		return resp, errors.New("invalid leverage")
	}
	params.Set("leverage", strconv.FormatInt(leverage, 10))
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodPost, ufuturesChangeInitialLeverage, params, limitDefault, &resp)
}

// UChangeInitialMarginType sends a request to change account's initial margin type
func (b *Binance) UChangeInitialMarginType(symbol, marginType string) error {
	var resp UAccountInformationV2Data
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(validMarginType, marginType) {
		return errors.New("invalid marginType")
	}
	params.Set("marginType", marginType)
	return b.SendAuthHTTPRequest(uFutures, http.MethodPost, ufuturesChangeMarginType, params, limitDefault, &resp)
}

// UModifyIsolatedPositionMarginReq sends a request to modify isolated margin for USDTMarginedFutures
func (b *Binance) UModifyIsolatedPositionMarginReq(symbol, positionSide, changeType string, amount float64) (UModifyIsolatedPosMargin, error) {
	var resp UModifyIsolatedPosMargin
	params := url.Values{}
	params.Set("symbol", symbol)
	if positionSide != "" {
		if !common.StringDataCompare(validPositionSide, positionSide) {
			return resp, errors.New("invalid margin changeType")
		}
	}
	cType, ok := validMarginChange[changeType]
	if !ok {
		return resp, errors.New("invalid margin changeType")
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodPost, ufuturesModifyMargin, params, limitDefault, &resp)
}

// UPositionMarginChangeHistory gets margin change history for USDTMarginedFutures
func (b *Binance) UPositionMarginChangeHistory(symbol, changeType string, limit int64, startTime, endTime time.Time) ([]UPositionMarginChangeHistoryData, error) {
	var resp []UPositionMarginChangeHistoryData
	params := url.Values{}
	params.Set("symbol", symbol)
	cType, ok := validMarginChange[changeType]
	if !ok {
		return resp, errors.New("invalid margin changeType")
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesMarginChangeHistory, params, limitDefault, &resp)
}

// UPositionsInfoV2 gets positions' info for USDTMarginedFutures
func (b *Binance) UPositionsInfoV2(symbol string) ([]UChangeInitialLeverage, error) {
	var resp []UChangeInitialLeverage
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesPositionInfo, params, limitDefault, &resp)
}

// UAccountTradesHistory gets account's trade history data for USDTMarginedFutures
func (b *Binance) UAccountTradesHistory(symbol, fromID string, limit int64, startTime, endTime time.Time) ([]UAccountTradeHistory, error) {
	var resp []UAccountTradeHistory
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesAccountTradeList, params, limitDefault, &resp)
}

// UAccountIncomeHistory gets account's income history data for USDTMarginedFutures
func (b *Binance) UAccountIncomeHistory(symbol, incomeType string, limit int64, startTime, endTime time.Time) ([]UAccountIncomeHistory, error) {
	var resp []UAccountIncomeHistory
	params := url.Values{}
	params.Set("symbol", symbol)
	if incomeType != "" {
		if !common.StringDataCompare(validIncomeType, incomeType) {
			return resp, errors.New("invalid incomeType")
		}
		params.Set("incomeType", incomeType)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesIncomeHistory, params, limitDefault, &resp)
}

// UGetNotionalAndLeverageBrackets gets account's notional and leverage brackets for USDTMarginedFutures
func (b *Binance) UGetNotionalAndLeverageBrackets(symbol string) ([]UNotionalLeverageAndBrakcetsData, error) {
	var resp []UNotionalLeverageAndBrakcetsData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesNotionalBracket, params, limitDefault, &resp)
}

// UPositionsADLEstimate gets estimated ADL data for USDTMarginedFutures positions
func (b *Binance) UPositionsADLEstimate(symbol string) (UPositionADLEstimationData, error) {
	var resp UPositionADLEstimationData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesADLQuantile, params, limitDefault, &resp)
}

// UAccountForcedOrders gets account's forced (liquidation) orders for USDTMarginedFutures
func (b *Binance) UAccountForcedOrders(symbol, autoCloseType string, limit int64, startTime, endTime time.Time) ([]UForceOrdersData, error) {
	var resp []UForceOrdersData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if autoCloseType != "" {
		if !common.StringDataCompare(validAutoCloseTypes, autoCloseType) {
			return resp, errors.New("invalid incomeType")
		}
		params.Set("autoCloseType", autoCloseType)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendAuthHTTPRequest(uFutures, http.MethodGet, ufuturesUsersForceOrders, params, limitDefault, &resp)
}

// Coin Margined Futures

// FuturesExchangeInfo stores CoinMarginedFutures data
func (b *Binance) FuturesExchangeInfo() (CExchangeInfo, error) {
	var resp CExchangeInfo
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesExchangeInfo, limitDefault, &resp)
}

// GetFuturesOrderbook gets orderbook data for CoinMarginedFutures
func (b *Binance) GetFuturesOrderbook(symbol string, limit int64) (OrderBook, error) {
	var resp OrderBook
	var data OrderbookData
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	err := b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesOrderbook+params.Encode(), limitDefault, &data)
	if err != nil {
		return resp, err
	}
	var price, quantity float64
	for x := range data.Asks {
		price, err = strconv.ParseFloat(data.Asks[x][0], 64)
		if err != nil {
			return resp, err
		}
		quantity, err = strconv.ParseFloat(data.Asks[x][1], 64)
		if err != nil {
			return resp, err
		}
		resp.Asks = append(resp.Asks, OrderbookItem{
			Price:    price,
			Quantity: quantity,
		})
	}
	for y := range data.Bids {
		price, err = strconv.ParseFloat(data.Bids[y][0], 64)
		if err != nil {
			return resp, err
		}
		quantity, err = strconv.ParseFloat(data.Bids[y][1], 64)
		if err != nil {
			return resp, err
		}
		resp.Bids = append(resp.Bids, OrderbookItem{
			Price:    price,
			Quantity: quantity,
		})
	}
	return resp, nil
}

// GetFuturesPublicTrades gets recent public trades for CoinMarginedFutures
func (b *Binance) GetFuturesPublicTrades(symbol string, limit int64) ([]FuturesPublicTradesData, error) {
	var resp []FuturesPublicTradesData
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesRecentTrades+params.Encode(), limitDefault, &resp)
}

// GetFuturesHistoricalTrades gets historical public trades for CoinMarginedFutures
func (b *Binance) GetFuturesHistoricalTrades(symbol, fromID string, limit int64) ([]UPublicTradesData, error) {
	var resp []UPublicTradesData
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cfuturesHistoricalTrades, ufuturesHistoricalTrades+params.Encode(), limitDefault, &resp)
}

// GetPastPublicTrades gets past public trades for CoinMarginedFutures
func (b *Binance) GetPastPublicTrades(symbol string, limit, fromID int64) ([]FuturesPublicTradesData, error) {
	var resp []FuturesPublicTradesData
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatInt(fromID, 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesRecentTrades+params.Encode(), limitDefault, &resp)
}

// GetFuturesAggregatedTradesList gets aggregated trades list for CoinMarginedFutures
func (b *Binance) GetFuturesAggregatedTradesList(symbol string, fromID, limit int64, startTime, endTime time.Time) ([]AggregatedTrade, error) {
	var resp []AggregatedTrade
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatInt(fromID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesCompressedTrades+params.Encode(), limitDefault, &resp)
}

// GetIndexAndMarkPrice gets index and mark prices  for CoinMarginedFutures
func (b *Binance) GetIndexAndMarkPrice(symbol, pair string) ([]IndexMarkPrice, error) {
	var resp []IndexMarkPrice
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesMarkPrice+params.Encode(), limitDefault, &resp)
}

// GetFuturesKlineData gets futures kline data for CoinMarginedFutures
func (b *Binance) GetFuturesKlineData(symbol, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	var data [][]interface{}
	var resp []FuturesCandleStick
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if limit > 0 && limit <= 1000 {
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
	err := b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesKlineData+params.Encode(), limitDefault, &data)
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
			return resp, errors.New("type casting failed")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return resp, errors.New("type casting failed")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return resp, errors.New("type casting failed")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return resp, errors.New("type casting failed")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return resp, errors.New("type casting failed")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return resp, errors.New("type casting failed")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return resp, errors.New("type casting failed")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return resp, errors.New("type casting failed")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return resp, errors.New("type casting failed")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return resp, errors.New("type casting failed")
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

// GetContinuousKlineData gets continuous kline data
func (b *Binance) GetContinuousKlineData(pair, contractType, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	var data [][]interface{}
	var resp []FuturesCandleStick
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if limit > 0 && limit <= 1000 {
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
	err := b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesContinuousKline+params.Encode(), limitDefault, &data)
	if err != nil {
		return resp, err
	}
	var floatData float64
	var tempData FuturesCandleStick
	for x := range data {
		tempData.OpenTime = time.Unix(int64(data[x][0].(float64)), 0)
		floatData, err = strconv.ParseFloat(data[x][1].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Open = floatData
		floatData, err = strconv.ParseFloat(data[x][2].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.High = floatData
		floatData, err = strconv.ParseFloat(data[x][3].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Low = floatData
		floatData, err = strconv.ParseFloat(data[x][4].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Close = floatData
		floatData, err = strconv.ParseFloat(data[x][5].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Volume = floatData
		tempData.CloseTime = time.Unix(int64(data[x][6].(float64)), 0)
		floatData, err = strconv.ParseFloat(data[x][7].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.BaseAssetVolume = floatData
		tempData.TakerBuyVolume = data[x][8].(float64)
		floatData, err = strconv.ParseFloat(data[x][9].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp = append(resp, tempData)
	}
	return resp, nil
}

// GetIndexPriceKlines gets continuous kline data
func (b *Binance) GetIndexPriceKlines(pair, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	var data [][]interface{}
	var resp []FuturesCandleStick
	params := url.Values{}
	params.Set("pair", pair)
	if limit > 0 && limit <= 1000 {
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
	err := b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesIndexKline+params.Encode(), limitDefault, &data)
	if err != nil {
		return resp, err
	}
	var floatData float64
	var tempData FuturesCandleStick
	for x := range data {
		tempData.OpenTime = time.Unix(int64(data[x][0].(float64)), 0)
		floatData, err = strconv.ParseFloat(data[x][1].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Open = floatData
		floatData, err = strconv.ParseFloat(data[x][2].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.High = floatData
		floatData, err = strconv.ParseFloat(data[x][3].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Low = floatData
		floatData, err = strconv.ParseFloat(data[x][4].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Close = floatData
		floatData, err = strconv.ParseFloat(data[x][5].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Volume = floatData
		tempData.CloseTime = time.Unix(int64(data[x][6].(float64)), 0)
		floatData, err = strconv.ParseFloat(data[x][7].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.BaseAssetVolume = floatData
		tempData.TakerBuyVolume = data[x][8].(float64)
		floatData, err = strconv.ParseFloat(data[x][9].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp = append(resp, tempData)
	}
	return resp, nil
}

// GetMarkPriceKline gets mark price kline data
func (b *Binance) GetMarkPriceKline(symbol, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	var data [][]interface{}
	var resp []FuturesCandleStick
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 && limit <= 1000 {
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
	err := b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesMarkPriceKline+params.Encode(), limitDefault, &data)
	if err != nil {
		return resp, err
	}
	var floatData float64
	var tempData FuturesCandleStick
	for x := range data {
		tempData.OpenTime = time.Unix(int64(data[x][0].(float64)), 0)
		floatData, err = strconv.ParseFloat(data[x][1].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Open = floatData
		floatData, err = strconv.ParseFloat(data[x][2].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.High = floatData
		floatData, err = strconv.ParseFloat(data[x][3].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Low = floatData
		floatData, err = strconv.ParseFloat(data[x][4].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Close = floatData
		floatData, err = strconv.ParseFloat(data[x][5].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.Volume = floatData
		tempData.CloseTime = time.Unix(int64(data[x][6].(float64)), 0)
		floatData, err = strconv.ParseFloat(data[x][7].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.BaseAssetVolume = floatData
		tempData.TakerBuyVolume = data[x][8].(float64)
		floatData, err = strconv.ParseFloat(data[x][9].(string), 64)
		if err != nil {
			return resp, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp = append(resp, tempData)
	}
	return resp, nil
}

// GetFuturesSwapTickerChangeStats gets 24hr ticker change stats for CoinMarginedFutures
func (b *Binance) GetFuturesSwapTickerChangeStats(symbol, pair string) ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesTickerPriceStats+params.Encode(), limitDefault, &resp)
}

// FuturesGetFundingHistory gets funding history for CoinMarginedFutures
func (b *Binance) FuturesGetFundingHistory(symbol string, limit int64, startTime, endTime time.Time) ([]FundingRateHistory, error) {
	var resp []FundingRateHistory
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesFundingRateHistory+params.Encode(), limitDefault, &resp)
}

// GetFuturesSymbolPriceTicker gets price ticker for symbol
func (b *Binance) GetFuturesSymbolPriceTicker(symbol, pair string) ([]SymbolPriceTicker, error) {
	var resp []SymbolPriceTicker
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesSymbolPriceTicker+params.Encode(), limitDefault, &resp)
}

// GetFuturesOrderbookTicker gets orderbook ticker for symbol
func (b *Binance) GetFuturesOrderbookTicker(symbol, pair string) ([]SymbolOrderBookTicker, error) {
	var resp []SymbolOrderBookTicker
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesSymbolOrderbook+params.Encode(), limitDefault, &resp)
}

// GetFuturesLiquidationOrders gets orderbook ticker for symbol
func (b *Binance) GetFuturesLiquidationOrders(symbol, pair string, limit int64, startTime, endTime time.Time) ([]AllLiquidationOrders, error) {
	var resp []AllLiquidationOrders
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesLiquidationOrders+params.Encode(), limitDefault, &resp)
}

// GetOpenInterest gets open interest data for a symbol
func (b *Binance) GetOpenInterest(symbol string) (OpenInterestData, error) {
	var resp OpenInterestData
	params := url.Values{}
	params.Set("symbol", symbol)
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesOpenInterest+params.Encode(), limitDefault, &resp)
}

// GetOpenInterestStats gets open interest stats for a symbol
func (b *Binance) GetOpenInterestStats(pair, contractType, period string, limit int64, startTime, endTime time.Time) ([]OpenInterestStats, error) {
	var resp []OpenInterestStats
	params := url.Values{}
	if pair != "" {
		params.Set("pair", pair)
	}
	if !common.StringDataCompare(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesOpenInterestStats+params.Encode(), limitDefault, &resp)
}

// GetTraderFuturesAccountRatio gets a traders futures account long/short ratio
func (b *Binance) GetTraderFuturesAccountRatio(pair, period string, limit int64, startTime, endTime time.Time) ([]TopTraderAccountRatio, error) {
	var resp []TopTraderAccountRatio
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesTopAccountsRatio+params.Encode(), limitDefault, &resp)
}

// GetTraderFuturesPositionsRatio gets a traders futures positions' long/short ratio
func (b *Binance) GetTraderFuturesPositionsRatio(pair, period string, limit int64, startTime, endTime time.Time) ([]TopTraderPositionRatio, error) {
	var resp []TopTraderPositionRatio
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesTopPositionsRatio+params.Encode(), limitDefault, &resp)
}

// GetMarketRatio gets global long/short ratio
func (b *Binance) GetMarketRatio(pair, period string, limit int64, startTime, endTime time.Time) ([]TopTraderPositionRatio, error) {
	var resp []TopTraderPositionRatio
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesLongShortRatio+params.Encode(), limitDefault, &resp)
}

// GetFuturesTakerVolume gets futures taker buy/sell volumes
func (b *Binance) GetFuturesTakerVolume(pair, contractType, period string, limit int64, startTime, endTime time.Time) ([]TakerBuySellVolume, error) {
	var resp []TakerBuySellVolume
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period parsed")
	}
	params.Set("period", period)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesBuySellVolume+params.Encode(), limitDefault, &resp)
}

// GetFuturesBasisData gets futures basis data
func (b *Binance) GetFuturesBasisData(pair, contractType, period string, limit int64, startTime, endTime time.Time) ([]FuturesBasisData, error) {
	var resp []FuturesBasisData
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period parsed")
	}
	params.Set("period", period)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+cmFutures, cfuturesBasis+params.Encode(), limitDefault, &resp)
}

// FuturesNewOrder sends a new futures order to the exchange
func (b *Binance) FuturesNewOrder(symbol, side, positionSide, orderType, timeInForce,
	newClientOrderID, closePosition, workingType, newOrderRespType string,
	quantity, price, stopPrice, activationPrice, callbackRate float64, reduceOnly bool) (FuturesOrderPlaceData, error) {
	var resp FuturesOrderPlaceData
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side)
	if positionSide != "" {
		if !common.StringDataCompare(validPositionSide, positionSide) {
			return resp, errors.New("invalid positionSide")
		}
		params.Set("positionSide", positionSide)
	}
	params.Set("type", orderType)
	params.Set("timeInForce", timeInForce)
	if reduceOnly {
		params.Set("reduceOnly", "true")
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderID", newClientOrderID)
	}
	if closePosition != "" {
		params.Set("closePosition", closePosition)
	}
	if workingType != "" {
		if !common.StringDataCompare(validWorkingType, workingType) {
			return resp, errors.New("invalid workingType")
		}
		params.Set("workingType", workingType)
	}
	if newOrderRespType != "" {
		if !common.StringDataCompare(validNewOrderRespType, newOrderRespType) {
			return resp, errors.New("invalid newOrderRespType")
		}
		params.Set("newOrderRespType", newOrderRespType)
	}
	if quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	}
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if stopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(stopPrice, 'f', -1, 64))
	}
	if activationPrice != 0 {
		params.Set("activationPrice", strconv.FormatFloat(activationPrice, 'f', -1, 64))
	}
	if callbackRate != 0 {
		params.Set("callbackRate", strconv.FormatFloat(callbackRate, 'f', -1, 64))
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodPost, cfuturesOrder, params, limitDefault, &resp)
}

// FuturesBatchOrder sends a batch order request
func (b *Binance) FuturesBatchOrder(data []PlaceBatchOrderData) ([]FuturesOrderPlaceData, error) {
	var resp []FuturesOrderPlaceData
	params := url.Values{}
	for x := range data {
		if data[x].PositionSide != "" {
			if !common.StringDataCompare(validPositionSide, data[x].PositionSide) {
				return resp, errors.New("invalid positionSide")
			}
		}
		if data[x].WorkingType != "" {
			if !common.StringDataCompare(validWorkingType, data[x].WorkingType) {
				return resp, errors.New("invalid workingType")
			}
		}
		if data[x].NewOrderRespType != "" {
			if !common.StringDataCompare(validNewOrderRespType, data[x].NewOrderRespType) {
				return resp, errors.New("invalid newOrderRespType")
			}
		}
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return resp, err
	}
	params.Set("batchOrders", string(jsonData))
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodPost, cfuturesBatchOrder, params, limitDefault, &resp)
}

// FuturesBatchCancelOrders sends a batch request to cancel orders
func (b *Binance) FuturesBatchCancelOrders(symbol string, orderList, origClientOrderIDList []string) ([]BatchCancelOrderData, error) {
	var resp []BatchCancelOrderData
	params := url.Values{}
	params.Set("symbol", symbol)
	if len(orderList) != 0 {
		jsonOrderList, err := json.Marshal(orderList)
		if err != nil {
			return resp, err
		}
		params.Set("orderIdList", string(jsonOrderList))
	}
	if len(origClientOrderIDList) != 0 {
		jsonCliOrdIDList, err := json.Marshal(origClientOrderIDList)
		if err != nil {
			return resp, err
		}
		params.Set("origClientOrderIdList", string(jsonCliOrdIDList))
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodDelete, cfuturesBatchOrder, params, limitDefault, &resp)
}

// FuturesGetOrderData gets futures order data
func (b *Binance) FuturesGetOrderData(symbol, orderID, origClientOrderID string) (FuturesOrderGetData, error) {
	var resp FuturesOrderGetData
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesOrder, params, limitDefault, &resp)
}

// FuturesCancelOrder cancels a futures order
func (b *Binance) FuturesCancelOrder(symbol, orderID, origClientOrderID string) (FuturesOrderGetData, error) {
	var resp FuturesOrderGetData
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodDelete, cfuturesOrder, params, limitDefault, &resp)
}

// CancelAllOpenOrders cancels a futures order
func (b *Binance) CancelAllOpenOrders(symbol string) (GenericAuthResponse, error) {
	var resp GenericAuthResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodDelete, cfuturesCancelAllOrders, params, limitDefault, &resp)
}

// AutoCancelAllOpenOrders cancels all open futures orders
// countdownTime 1000 = 1s, example - to cancel all orders after 30s (countdownTime: 30000)
func (b *Binance) AutoCancelAllOpenOrders(symbol string, countdownTime int64) (AutoCancelAllOrdersData, error) {
	var resp AutoCancelAllOrdersData
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("countdownTime", strconv.FormatInt(countdownTime, 10))
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodPost, cfuturesCountdownCancel, params, limitDefault, &resp)
}

// FuturesOpenOrderData gets open order data for CoinMarginedFutures
func (b *Binance) FuturesOpenOrderData(symbol, orderID, origClientOrderID string) (FuturesOrderGetData, error) {
	var resp FuturesOrderGetData
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesOpenOrder, params, limitDefault, &resp)
}

// GetFuturesAllOpenOrders gets all open orders data for CoinMarginedFutures
func (b *Binance) GetFuturesAllOpenOrders(symbol, pair string) ([]FuturesOrderData, error) {
	var resp []FuturesOrderData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesAllOpenOrders, params, limitDefault, &resp)
}

// GetAllFuturesOrders gets all orders active cancelled or filled
func (b *Binance) GetAllFuturesOrders(symbol, pair string, startTime, endTime time.Time, orderID, limit int64) ([]FuturesOrderData, error) {
	var resp []FuturesOrderData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	if orderID != 0 {
		params.Set("orderID", strconv.FormatInt(orderID, 10))
	}
	if limit > 0 && limit <= 100 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesAllOrders, params, limitDefault, &resp)
}

// GetFuturesAccountBalance gets account balance data for CoinMarginedFutures account
func (b *Binance) GetFuturesAccountBalance() ([]FuturesAccountBalanceData, error) {
	var resp []FuturesAccountBalanceData
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesAPIURL+cfuturesAccountBalance, nil, limitDefault, &resp)
}

// GetFuturesAccountInfo gets account info data for CoinMarginedFutures account
func (b *Binance) GetFuturesAccountInfo() (FuturesAccountInformation, error) {
	var resp FuturesAccountInformation
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesAccountInfo, nil, limitDefault, &resp)
}

// FuturesChangeInitialLeverage changes initial leverage for the account
func (b *Binance) FuturesChangeInitialLeverage(symbol string, leverage int64) (FuturesLeverageData, error) {
	var resp FuturesLeverageData
	params := url.Values{}
	params.Set("symbol", symbol)
	if !(leverage >= 1 && leverage <= 125) {
		return resp, errors.New("invalid leverage")
	}
	params.Set("leverage", strconv.FormatInt(leverage, 10))
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodPost, cfuturesChangeInitialLeverage, params, limitDefault, &resp)
}

// FuturesChangeMarginType changes margin type
func (b *Binance) FuturesChangeMarginType(symbol, marginType string) (GenericAuthResponse, error) {
	var resp GenericAuthResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(validMarginType, marginType) {
		return resp, errors.New("invalid marginType")
	}
	params.Set("marginType", marginType)
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodPost, cfuturesChangeMarginType, params, limitDefault, &resp)
}

// ModifyIsolatedPositionMargin changes margin for an isolated position
func (b *Binance) ModifyIsolatedPositionMargin(symbol, positionSide, changeType string, amount float64) (GenericAuthResponse, error) {
	var resp GenericAuthResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(validPositionSide, positionSide) {
		return resp, errors.New("invalid positionSide")
	}
	params.Set("positionSide", positionSide)
	cType, ok := validMarginChange[changeType]
	if !ok {
		return resp, errors.New("invalid changeType")
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodPost, cfuturesModifyMargin, params, limitDefault, &resp)
}

// FuturesMarginChangeHistory gets past margin changes for positions
func (b *Binance) FuturesMarginChangeHistory(symbol, changeType string, startTime, endTime time.Time, limit int64) ([]GetPositionMarginChangeHistoryData, error) {
	var resp []GetPositionMarginChangeHistoryData
	params := url.Values{}
	params.Set("symbol", symbol)
	cType, ok := validMarginChange[changeType]
	if !ok {
		return resp, errors.New("invalid changeType")
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesMarginChangeHistory, params, limitDefault, &resp)
}

// FuturesPositionsInfo gets futures positions info
func (b *Binance) FuturesPositionsInfo(marginAsset, pair string) ([]FuturesPositionInformation, error) {
	var resp []FuturesPositionInformation
	params := url.Values{}
	if marginAsset != "" {
		params.Set("marginAsset", marginAsset)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesPositionInfo, params, limitDefault, &resp)
}

// FuturesTradeHistory gets trade history for CoinMarginedFutures account
func (b *Binance) FuturesTradeHistory(symbol, pair string, startTime, endTime time.Time, limit, fromID int64) ([]FuturesAccountTradeList, error) {
	var resp []FuturesAccountTradeList
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesAccountTradeList, params, limitDefault, &resp)
}

// FuturesIncomeHistory gets income history for CoinMarginedFutures
func (b *Binance) FuturesIncomeHistory(symbol, incomeType string, startTime, endTime time.Time, limit int64) ([]FuturesIncomeHistoryData, error) {
	var resp []FuturesIncomeHistoryData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if incomeType != "" {
		if !common.StringDataCompare(validIncomeType, incomeType) {
			return resp, fmt.Errorf("invalid incomeType: %v", incomeType)
		}
		params.Set("incomeType", incomeType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesIncomeHistory, params, limitDefault, &resp)
}

// FuturesNotionalBracket gets futures notional bracket
func (b *Binance) FuturesNotionalBracket(pair string) ([]NotionalBracketData, error) {
	var resp []NotionalBracketData
	params := url.Values{}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodPost, cfuturesNotionalBracket, params, limitDefault, &resp)
}

// FuturesForceOrders gets futures forced orders
func (b *Binance) FuturesForceOrders(symbol, autoCloseType string, startTime, endTime time.Time) ([]ForcedOrdersData, error) {
	var resp []ForcedOrdersData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if autoCloseType != "" {
		if !common.StringDataCompare(validAutoCloseTypes, autoCloseType) {
			return resp, errors.New("invalid autoCloseType")
		}
		params.Set("autoCloseType", autoCloseType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesUsersForceOrders, params, limitDefault, &resp)
}

// FuturesPositionsADLEstimate estimates ADL on positions
func (b *Binance) FuturesPositionsADLEstimate(symbol string) ([]ADLEstimateData, error) {
	var resp []ADLEstimateData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return resp, b.SendAuthHTTPRequest(cmFutures, http.MethodGet, cfuturesADLQuantile, params, limitDefault, &resp)
}

// GetInterestHistory gets interest history for currency/currencies provided
func (b *Binance) GetInterestHistory() (MarginInfoData, error) {
	var resp MarginInfoData
	if err := b.SendHTTPRequest(exchange.Running+interestHistoryEdgeCase, undocumentedInterestHistory, limitDefault, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetPerpMarkets returns exchange information. Check binance_types for more
// information
func (b *Binance) GetPerpMarkets() (PerpsExchangeInfo, error) {
	var resp PerpsExchangeInfo
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, perpExchangeInfo, limitDefault, &resp)
}

// GetMarginMarkets returns exchange information. Check binance_types for more
// information
func (b *Binance) GetMarginMarkets() (PerpsExchangeInfo, error) {
	var resp PerpsExchangeInfo
	return resp, b.SendHTTPRequest(exchange.Running+spot, perpExchangeInfo, limitDefault, &resp)
}

// GetFundingRates gets funding rate history for perpetual contracts
func (b *Binance) GetFundingRates(symbol, limit string, startTime, endTime time.Time) ([]FundingRateData, error) {
	var resp []FundingRateData
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit != "" {
		params.Set("limit", limit)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixNano(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixNano(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.Running+uFutures, fundingRate+params.Encode(), limitDefault, &resp)
}

// GetExchangeInfo returns exchange information. Check binance_types for more
// information
func (b *Binance) GetExchangeInfo() (ExchangeInfo, error) {
	var resp ExchangeInfo
	return resp, b.SendHTTPRequest(exchange.Running+exchange.DefaultSpot, exchangeInfo, limitDefault, &resp)
}

// GetOrderBook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (b *Binance) GetOrderBook(obd OrderBookDataRequestParams) (OrderBook, error) {
	var orderbook OrderBook
	if err := b.CheckLimit(obd.Limit); err != nil {
		return orderbook, err
	}

	params := url.Values{}
	params.Set("symbol", strings.ToUpper(obd.Symbol))
	params.Set("limit", fmt.Sprintf("%d", obd.Limit))

	var resp OrderBookData
	if err := b.SendHTTPRequest(exchange.Running+exchange.DefaultSpot, orderBookDepth+"?"+params.Encode(), orderbookLimit(obd.Limit), &resp); err != nil {
		return orderbook, err
	}

	for x := range resp.Bids {
		price, err := strconv.ParseFloat(resp.Bids[x][0], 64)
		if err != nil {
			return orderbook, err
		}

		amount, err := strconv.ParseFloat(resp.Bids[x][1], 64)
		if err != nil {
			return orderbook, err
		}

		orderbook.Bids = append(orderbook.Bids, OrderbookItem{
			Price:    price,
			Quantity: amount,
		})
	}

	for x := range resp.Asks {
		price, err := strconv.ParseFloat(resp.Asks[x][0], 64)
		if err != nil {
			return orderbook, err
		}

		amount, err := strconv.ParseFloat(resp.Asks[x][1], 64)
		if err != nil {
			return orderbook, err
		}

		orderbook.Asks = append(orderbook.Asks, OrderbookItem{
			Price:    price,
			Quantity: amount,
		})
	}

	orderbook.LastUpdateID = resp.LastUpdateID
	return orderbook, nil
}

// GetMostRecentTrades returns recent trade activity
// limit: Up to 500 results returned
func (b *Binance) GetMostRecentTrades(rtr RecentTradeRequestParams) ([]RecentTrade, error) {
	var resp []RecentTrade

	params := url.Values{}
	params.Set("symbol", strings.ToUpper(rtr.Symbol))
	params.Set("limit", fmt.Sprintf("%d", rtr.Limit))

	path := recentTrades + "?" + params.Encode()

	return resp, b.SendHTTPRequest(exchange.Running+exchange.DefaultSpot, path, limitDefault, &resp)
}

// GetHistoricalTrades returns historical trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
// fromID:
func (b *Binance) GetHistoricalTrades(symbol string, limit int, fromID int64) ([]HistoricalTrade, error) {
	// Dropping support due to response for market data is always
	// {"code":-2014,"msg":"API-key format invalid."}
	// TODO: replace with newer API vs REST endpoint
	return nil, common.ErrFunctionNotSupported
}

// GetAggregatedTrades returns aggregated trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
func (b *Binance) GetAggregatedTrades(symbol string, limit int) ([]AggregatedTrade, error) {
	var resp []AggregatedTrade

	if err := b.CheckLimit(limit); err != nil {
		return resp, err
	}

	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	path := aggregatedTrades + "?" + params.Encode()
	return resp, b.SendHTTPRequest(exchange.Running+exchange.DefaultSpot, path, limitDefault, &resp)
}

// GetSpotKline returns kline data
//
// KlinesRequestParams supports 5 parameters
// symbol: the symbol to get the kline data for
// limit: optinal
// interval: the interval time for the data
// startTime: startTime filter for kline data
// endTime: endTime filter for the kline data
func (b *Binance) GetSpotKline(arg KlinesRequestParams) ([]CandleStick, error) {
	var resp interface{}
	var klineData []CandleStick

	params := url.Values{}
	params.Set("symbol", arg.Symbol)
	params.Set("interval", arg.Interval)
	if arg.Limit != 0 {
		params.Set("limit", strconv.Itoa(arg.Limit))
	}
	if arg.StartTime != 0 {
		params.Set("startTime", strconv.FormatInt(arg.StartTime, 10))
	}
	if arg.EndTime != 0 {
		params.Set("endTime", strconv.FormatInt(arg.EndTime, 10))
	}

	path := candleStick + "?" + params.Encode()

	if err := b.SendHTTPRequest(exchange.Running+exchange.DefaultSpot, path, limitDefault, &resp); err != nil {
		return klineData, err
	}

	for _, responseData := range resp.([]interface{}) {
		var candle CandleStick
		for i, individualData := range responseData.([]interface{}) {
			switch i {
			case 0:
				tempTime := individualData.(float64)
				var err error
				candle.OpenTime, err = convert.TimeFromUnixTimestampFloat(tempTime)
				if err != nil {
					return klineData, err
				}
			case 1:
				candle.Open, _ = strconv.ParseFloat(individualData.(string), 64)
			case 2:
				candle.High, _ = strconv.ParseFloat(individualData.(string), 64)
			case 3:
				candle.Low, _ = strconv.ParseFloat(individualData.(string), 64)
			case 4:
				candle.Close, _ = strconv.ParseFloat(individualData.(string), 64)
			case 5:
				candle.Volume, _ = strconv.ParseFloat(individualData.(string), 64)
			case 6:
				tempTime := individualData.(float64)
				var err error
				candle.CloseTime, err = convert.TimeFromUnixTimestampFloat(tempTime)
				if err != nil {
					return klineData, err
				}
			case 7:
				candle.QuoteAssetVolume, _ = strconv.ParseFloat(individualData.(string), 64)
			case 8:
				candle.TradeCount = individualData.(float64)
			case 9:
				candle.TakerBuyAssetVolume, _ = strconv.ParseFloat(individualData.(string), 64)
			case 10:
				candle.TakerBuyQuoteAssetVolume, _ = strconv.ParseFloat(individualData.(string), 64)
			}
		}
		klineData = append(klineData, candle)
	}
	return klineData, nil
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (b *Binance) GetAveragePrice(symbol string) (AveragePrice, error) {
	resp := AveragePrice{}
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	path := averagePrice + "?" + params.Encode()
	return resp, b.SendHTTPRequest(exchange.Running+exchange.DefaultSpot, path, limitDefault, &resp)
}

// GetPriceChangeStats returns price change statistics for the last 24 hours
//
// symbol: string of currency pair
func (b *Binance) GetPriceChangeStats(symbol string) (PriceChangeStats, error) {
	resp := PriceChangeStats{}
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	path := priceChange + "?" + params.Encode()
	return resp, b.SendHTTPRequest(exchange.Running+exchange.DefaultSpot, path, limitDefault, &resp)
}

// GetTickers returns the ticker data for the last 24 hrs
func (b *Binance) GetTickers() ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	return resp, b.SendHTTPRequest(exchange.Running+exchange.DefaultSpot, priceChange, limitPriceChangeAll, &resp)
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (b *Binance) GetLatestSpotPrice(symbol string) (SymbolPrice, error) {
	resp := SymbolPrice{}
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	path := symbolPrice + "?" + params.Encode()

	return resp, b.SendHTTPRequest(exchange.Running+exchange.DefaultSpot, path, symbolPriceLimit(symbol), &resp)
}

// GetBestPrice returns the latest best price for symbol
//
// symbol: string of currency pair
func (b *Binance) GetBestPrice(symbol string) (BestPrice, error) {
	resp := BestPrice{}
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	path := bestPrice + "?" + params.Encode()
	return resp, b.SendHTTPRequest(exchange.Running+exchange.DefaultSpot, path, bestPriceLimit(symbol), &resp)
}

// NewOrder sends a new order to Binance
func (b *Binance) NewOrder(o *NewOrderRequest) (NewOrderResponse, error) {
	var resp NewOrderResponse
	if err := b.newOrder(orderEndpoint, o, &resp); err != nil {
		return resp, err
	}

	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}

	return resp, nil
}

// NewOrderTest sends a new test order to Binance
func (b *Binance) NewOrderTest(o *NewOrderRequest) error {
	var resp NewOrderResponse
	return b.newOrder(newOrderTest, o, &resp)
}

func (b *Binance) newOrder(api string, o *NewOrderRequest, resp *NewOrderResponse) error {

	params := url.Values{}
	params.Set("symbol", o.Symbol)
	params.Set("side", o.Side)
	params.Set("type", string(o.TradeType))
	if o.QuoteOrderQty > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(o.QuoteOrderQty, 'f', -1, 64))
	} else {
		params.Set("quantity", strconv.FormatFloat(o.Quantity, 'f', -1, 64))
	}
	if o.TradeType == BinanceRequestParamsOrderLimit {
		params.Set("price", strconv.FormatFloat(o.Price, 'f', -1, 64))
	}
	if o.TimeInForce != "" {
		params.Set("timeInForce", string(o.TimeInForce))
	}

	if o.NewClientOrderID != "" {
		params.Set("newClientOrderID", o.NewClientOrderID)
	}

	if o.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(o.StopPrice, 'f', -1, 64))
	}

	if o.IcebergQty != 0 {
		params.Set("icebergQty", strconv.FormatFloat(o.IcebergQty, 'f', -1, 64))
	}

	if o.NewOrderRespType != "" {
		params.Set("newOrderRespType", o.NewOrderRespType)
	}
	return b.SendAuthHTTPRequest(exchange.DefaultSpot, http.MethodPost, api, params, limitOrder, resp)
}

// CancelExistingOrder sends a cancel order to Binance
func (b *Binance) CancelExistingOrder(symbol string, orderID int64, origClientOrderID string) (CancelOrderResponse, error) {
	var resp CancelOrderResponse

	params := url.Values{}
	params.Set("symbol", symbol)

	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}

	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return resp, b.SendAuthHTTPRequest(exchange.DefaultSpot, http.MethodDelete, orderEndpoint, params, limitOrder, &resp)
}

// OpenOrders Current open orders. Get all open orders on a symbol.
// Careful when accessing this with no symbol: The number of requests counted against the rate limiter
// is significantly higher
func (b *Binance) OpenOrders(symbol string) ([]QueryOrderData, error) {
	var resp []QueryOrderData

	params := url.Values{}

	if symbol != "" {
		params.Set("symbol", strings.ToUpper(symbol))
	}

	if err := b.SendAuthHTTPRequest(exchange.DefaultSpot, http.MethodGet, openOrders, params, openOrdersLimit(symbol), &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// AllOrders Get all account orders; active, canceled, or filled.
// orderId optional param
// limit optional param, default 500; max 500
func (b *Binance) AllOrders(symbol, orderID, limit string) ([]QueryOrderData, error) {
	var resp []QueryOrderData

	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	if err := b.SendAuthHTTPRequest(exchange.DefaultSpot, http.MethodGet, allOrders, params, limitOrdersAll, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// QueryOrder returns information on a past order
func (b *Binance) QueryOrder(symbol, origClientOrderID string, orderID int64) (QueryOrderData, error) {
	var resp QueryOrderData

	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}

	if err := b.SendAuthHTTPRequest(exchange.DefaultSpot, http.MethodGet, orderEndpoint, params, limitOrder, &resp); err != nil {
		return resp, err
	}

	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}
	return resp, nil
}

// GetAccount returns binance user accounts
func (b *Binance) GetAccount() (*Account, error) {
	type response struct {
		Response
		Account
	}

	var resp response
	params := url.Values{}

	if err := b.SendAuthHTTPRequest(exchange.DefaultSpot, http.MethodGet, accountInfo, params, request.Unset, &resp); err != nil {
		return &resp.Account, err
	}

	if resp.Code != 0 {
		return &resp.Account, errors.New(resp.Msg)
	}

	return &resp.Account, nil
}

// SendHTTPRequest sends an unauthenticated request
func (b *Binance) SendHTTPRequest(ePath, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := b.API.Endpoints.Get(ePath)
	if err != nil {
		return err
	}
	return b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
		Endpoint:      f})
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (b *Binance) SendAuthHTTPRequest(ePath, method, path string, params url.Values, f request.EndpointLimit, result interface{}) error {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}
	endpointPath, err := b.API.Endpoints.Get(ePath)
	if err != nil {
		return err
	}
	path = endpointPath + path
	if params == nil {
		params = url.Values{}
	}
	recvWindow := 5 * time.Second
	params.Set("recvWindow", strconv.FormatInt(convert.RecvWindow(recvWindow), 10))
	params.Set("timestamp", strconv.FormatInt(time.Now().Unix()*1000, 10))
	signature := params.Encode()
	hmacSigned := crypto.GetHMAC(crypto.HashSHA256, []byte(signature), []byte(b.API.Credentials.Secret))
	hmacSignedStr := crypto.HexEncodeToString(hmacSigned)
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.API.Credentials.Key
	if b.Verbose {
		log.Debugf(log.ExchangeSys, "sent path: %s", path)
	}

	path = common.EncodeURLValues(path, params)
	path += "&signature=" + hmacSignedStr
	interim := json.RawMessage{}
	errCap := struct {
		Success bool   `json:"success"`
		Message string `json:"msg"`
		Code    int64  `json:"code"`
	}{}
	ctx, cancel := context.WithTimeout(context.Background(), recvWindow)
	defer cancel()
	err = b.SendPayload(ctx, &request.Item{
		Method:        method,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBuffer(nil),
		Result:        &interim,
		AuthRequest:   true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
		Endpoint:      f})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(interim, &errCap); err == nil {
		if !errCap.Success && errCap.Message != "" && errCap.Code != 200 {
			return errors.New(errCap.Message)
		}
	}
	return json.Unmarshal(interim, result)
}

// CheckLimit checks value against a variable list
func (b *Binance) CheckLimit(limit int) error {
	for x := range b.validLimits {
		if b.validLimits[x] == limit {
			return nil
		}
	}
	return errors.New("incorrect limit values - valid values are 5, 10, 20, 50, 100, 500, 1000")
}

// SetValues sets the default valid values
func (b *Binance) SetValues() {
	b.validLimits = []int{5, 10, 20, 50, 100, 500, 1000, 5000}
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Binance) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		multiplier, err := b.getMultiplier(feeBuilder.IsMaker)
		if err != nil {
			return 0, err
		}
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, multiplier)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.002 * price * amount
}

// getMultiplier retrieves account based taker/maker fees
func (b *Binance) getMultiplier(isMaker bool) (float64, error) {
	var multiplier float64
	account, err := b.GetAccount()
	if err != nil {
		return 0, err
	}
	if isMaker {
		multiplier = float64(account.MakerCommission)
	} else {
		multiplier = float64(account.TakerCommission)
	}
	return multiplier, nil
}

// calculateTradingFee returns the fee for trading any currency on Bittrex
func calculateTradingFee(purchasePrice, amount, multiplier float64) float64 {
	return (multiplier / 100) * purchasePrice * amount
}

// getCryptocurrencyWithdrawalFee returns the fee for withdrawing from the exchange
func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

// WithdrawCrypto sends cryptocurrency to the address of your choosing
func (b *Binance) WithdrawCrypto(asset, address, addressTag, name, amount string) (string, error) {
	var resp WithdrawResponse

	params := url.Values{}
	params.Set("asset", asset)
	params.Set("address", address)
	params.Set("amount", amount)
	if len(name) > 0 {
		params.Set("name", name)
	}
	if len(addressTag) > 0 {
		params.Set("addressTag", addressTag)
	}

	if err := b.SendAuthHTTPRequest(exchange.DefaultSpot, http.MethodPost, withdrawEndpoint, params, request.Unset, &resp); err != nil {
		return "", err
	}

	if !resp.Success {
		return resp.ID, errors.New(resp.Msg)
	}

	return resp.ID, nil
}

// GetDepositAddressForCurrency retrieves the wallet address for a given currency
func (b *Binance) GetDepositAddressForCurrency(currency string) (string, error) {
	resp := struct {
		Address    string `json:"address"`
		Success    bool   `json:"success"`
		AddressTag string `json:"addressTag"`
	}{}

	params := url.Values{}
	params.Set("asset", currency)
	params.Set("status", "true")

	return resp.Address,
		b.SendAuthHTTPRequest(exchange.DefaultSpot, http.MethodGet, depositAddress, params, request.Unset, &resp)
}

// GetWsAuthStreamKey will retrieve a key to use for authorised WS streaming
func (b *Binance) GetWsAuthStreamKey() (string, error) {
	endpointPath, err := b.API.Endpoints.Get(exchange.DefaultSpot)
	if err != nil {
		return "", err
	}
	var resp UserAccountStream
	path := endpointPath + userAccountStream
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.API.Credentials.Key
	err = b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodPost,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBuffer(nil),
		Result:        &resp,
		AuthRequest:   true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
	if err != nil {
		return "", err
	}
	return resp.ListenKey, nil
}

// MaintainWsAuthStreamKey will keep the key alive
func (b *Binance) MaintainWsAuthStreamKey() error {
	endpointPath, err := b.API.Endpoints.Get(exchange.DefaultSpot)
	if err != nil {
		return err
	}
	if listenKey == "" {
		listenKey, err = b.GetWsAuthStreamKey()
		return err
	}
	path := endpointPath + userAccountStream
	params := url.Values{}
	params.Set("listenKey", listenKey)
	path = common.EncodeURLValues(path, params)
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.API.Credentials.Key
	return b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodPut,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBuffer(nil),
		AuthRequest:   true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}
