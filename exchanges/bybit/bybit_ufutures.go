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

// TODO: handle rate limiting
const (

	// public endpoint
	ufuturesKline              = "/public/linear/kline"
	ufuturesRecentTrades       = "/public/linear/recent-trading-records"
	ufuturesMarkPriceKline     = "/public/linear/mark-price-kline"
	ufuturesIndexKline         = "/public/linear/index-price-kline"
	ufuturesIndexPremiumKline  = "/public/linear/premium-index-kline"
	ufuturesGetLastFundingRate = "/public/linear/funding/prev-funding-rate"
	ufuturesGetRiskLimit       = "/public/linear/risk-limit"

	// auth endpoint
	ufuturesCreateOrder             = "/private/linear/order/create"
	ufuturesGetActiveOrders         = "/private/linear/order/list"
	ufuturesCancelActiveOrder       = "/private/linear/order/cancel"
	ufuturesCancelAllActiveOrders   = "/private/linear/order/cancel-all"
	ufuturesReplaceActiveOrder      = "/private/linear/order/replace"
	ufuturesGetActiveRealtimeOrders = "/private/linear/order/search"

	ufuturesCreateConditionalOrder       = "/private/linear/stop-order/create"
	ufuturesGetConditionalOrders         = "/private/linear/stop-order/list"
	ufuturesCancelConditionalOrder       = "/private/linear/stop-order/cancel"
	ufuturesCancelAllConditionalOrders   = "/private/linear/stop-order/cancel-all"
	ufuturesReplaceConditionalOrder      = "/private/linear/stop-order/replace"
	ufuturesGetConditionalRealtimeOrders = "/private/linear/stop-order/search"

	ufuturesPosition         = "/private/linear/position/list"
	ufuturesSetAutoAddMargin = "/private/linear/position/set-auto-add-margin"
	ufuturesMarginSwitch     = "/private/linear/position/switch-isolated"
	ufuturesPositionSwitch   = "/private/linear/tpsl/switch-mode"
	ufuturesAddMargin        = "/private/linear/position/add-margin"
	ufuturesSetLeverage      = "/private/linear/position/set-leverage"
	ufuturesSetTradingStop   = "/private/linear/position/trading-stop"
	ufuturesGetTrades        = "/private/linear/trade/execution/list"
	ufuturesGetClosedTrades  = "/private/linear/trade/closed-pnl/list"

	ufuturesSetRiskLimit        = "/private/linear/position/set-risk"
	ufuturesPredictFundingRate  = "/private/linear/funding/predicted-funding"
	ufuturesGetMyLastFundingFee = "/private/linear/funding/prev-funding"
)

// GetUSDTFuturesKlineData gets futures kline data for USDTMarginedFutures.
func (by *Bybit) GetUSDTFuturesKlineData(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]FuturesCandleStick, error) {
	resp := struct {
		Data []FuturesCandleStick `json:"result"`
	}{}

	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return resp.Data, err
		}
		params.Set("symbol", symbolValue)
	} else {
		return resp.Data, errors.New("symbol missing")
	}

	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp.Data, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	params.Set("from", strconv.FormatInt(startTime.Unix(), 10))

	path := common.EncodeURLValues(ufuturesKline, params)
	err := by.SendHTTPRequest(exchange.RestUSDTMargined, path, &resp)
	if err != nil {
		return resp.Data, err
	}

	return resp.Data, nil
}

// GetUSDTPublicTrades gets past public trades for USDTMarginedFutures.
func (by *Bybit) GetUSDTPublicTrades(symbol currency.Pair, limit, fromID int64) ([]FuturesPublicTradesData, error) {
	resp := struct {
		Data []FuturesPublicTradesData `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	path := common.EncodeURLValues(ufuturesRecentTrades, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, &resp)
}
