package poloniex

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var errIntervalRequired = errors.New("interval is required")

// Reference data endpoints.

// GetSymbolInformation all symbols and their tradeLimit info. priceScale is referring to the max number of decimals allowed for a given symbol.
func (p *Poloniex) GetSymbolInformation(ctx context.Context, symbol currency.Pair) ([]SymbolDetail, error) {
	var resp []SymbolDetail
	path := "/markets"
	if !symbol.IsEmpty() {
		path = fmt.Sprintf("%s/%s", path, symbol)
	}
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetCurrencyInformations retrieves list of currencies and theiir detailed information.
func (p *Poloniex) GetCurrencyInformations(ctx context.Context) ([]CurrencyDetail, error) {
	var resp []CurrencyDetail
	path := "/currencies"
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetCurrencyInformation retrieves currency and their detailed information.
func (p *Poloniex) GetCurrencyInformation(ctx context.Context, ccy currency.Code) (CurrencyDetail, error) {
	var resp CurrencyDetail
	path := "/currencies"
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	path = fmt.Sprintf("%s/%s", path, ccy.String())
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetV2CurrencyInformations retrieves list of currency details for V2 API.
func (p *Poloniex) GetV2CurrencyInformations(ctx context.Context) ([]CurrencyV2Information, error) {
	var resp []CurrencyV2Information
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/currencies", &resp)
}

// GetV2CurrencyInformations retrieves currency details for V2 API.
func (p *Poloniex) GetV2CurrencyInformation(ctx context.Context, ccy currency.Code) (*CurrencyV2Information, error) {
	var resp CurrencyV2Information
	path := "/v2/currencies"
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	path = fmt.Sprintf("%s/%s", path, ccy.String())
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetSystemTimestamp retrieves current server time.
func (p *Poloniex) GetSystemTimestamp(ctx context.Context) (time.Time, error) {
	resp := &struct {
		ServerTime convert.ExchangeTime `json:"serverTime"`
	}{}
	return resp.ServerTime.Time(), p.SendHTTPRequest(ctx, exchange.RestSpot, "/timestamp", &resp)
}

// Marker Data endpoints.

// GetMarketPrices retrieves latest trade price for all symbols.
func (p *Poloniex) GetMarketPrices(ctx context.Context) ([]MarketPrice, error) {
	var resp []MarketPrice
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/markets/price", &resp)
}

// GetMarketPrice retrieves latest trade price for all symbols.
func (p *Poloniex) GetMarketPrice(ctx context.Context, symbol currency.Pair) (*MarketPrice, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp MarketPrice
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/price", symbol.String()), &resp)
}

// GetMarkPrices retrieves latest mark price for a single cross margin
func (p *Poloniex) GetMarkPrices(ctx context.Context) ([]MarkPrice, error) {
	var resp []MarkPrice
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/markets/markPrice", &resp)
}

// GetMarkPrice retrieves latest mark price for all cross margin symbol.
func (p *Poloniex) GetMarkPrice(ctx context.Context, symbol currency.Pair) (*MarkPrice, error) {
	var resp MarkPrice
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/markPrice", symbol.String()), &resp)
}

// MarkPriceComponents retrieves components of the mark price for a given symbol.
func (p *Poloniex) MarkPriceComponents(ctx context.Context, symbol currency.Pair) (*MarkPriceComponent, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp MarkPriceComponent
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/markPriceComponents", symbol.String()), &resp)
}

// GetOrderbook retrieves the order book for a given symbol. Scale and limit values are optional.
// For valid scale values, please refer to the scale values defined for each symbol .
// If scale is not supplied, then no grouping/aggregation will be applied.
func (p *Poloniex) GetOrderbook(ctx context.Context, symbol currency.Pair) (*OrderbookData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp OrderbookData
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/orderBook", symbol.String()), &resp)
}

// GetCandlesticks retrieves OHLC for a symbol at given timeframe (interval).
func (p *Poloniex) GetCandlesticks(ctx context.Context, symbol currency.Pair, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]CandlestickData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	} else if intervalString == "" {
		return nil, errIntervalRequired
	}
	params := url.Values{}
	params.Set("interval", intervalString)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []CandlestickArrayData
	err = p.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf("/markets/%s/candles", symbol.String()), params), &resp)
	if err != nil {
		return nil, err
	}
	return processCandlestickData(resp)
}

// GetTrades returns a list of recent trades, request param limit is optional, its default value is 500, and max value is 1000.
func (p *Poloniex) GetTrades(ctx context.Context, symbol currency.Pair, limit int64) ([]Trade, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []Trade
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf("/markets/%s/trades", symbol.String()), params), &resp)
}

// GetTicker retrieve ticker in last 24 hours for all symbols.
func (p *Poloniex) GetTickers(ctx context.Context) ([]TickerData, error) {
	var resp []TickerData
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/markets/ticker24h", &resp)
}

// GetTicker retrieve ticker in last 24 hours for provided symbols.
func (p *Poloniex) GetTicker(ctx context.Context, symbol currency.Pair) (*TickerData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp TickerData
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/ticker24h", symbol.String()), &resp)
}

// Margin endpoints.

// GetCollateralInfos retrieves collateral information for all currencies.
func (p *Poloniex) GetCollateralInfos(ctx context.Context) ([]CollateralInfo, error) {
	var resp []CollateralInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/markets/collateralInfo", &resp)
}

// GetCollateralInfo retrieves collateral information for all currencies.
func (p *Poloniex) GetCollateralInfo(ctx context.Context, ccy currency.Code) (*CollateralInfo, error) {
	var resp CollateralInfo
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/collateralInfo", ccy.String()), &resp)
}

// GetBorrowRateInfo retrieves borrow rates information for all tiers and currencies.
func (p *Poloniex) GetBorrowRateInfo(ctx context.Context) ([]BorrowRateinfo, error) {
	var resp []BorrowRateinfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/markets/borrowRatesInfo", &resp)
}

// -------- end --------------------

func intervalToString(interval kline.Interval) (string, error) {
	intervalMap := map[kline.Interval]string{
		kline.OneMin:     "MINUTE_1",
		kline.FiveMin:    "MINUTE_5",
		kline.TenMin:     "MINUTE_10",
		kline.FifteenMin: "MINUTE_15",
		kline.ThirtyMin:  "MINUTE_30",
		kline.OneHour:    "HOUR_1",
		kline.TwoHour:    "HOUR_2",
		kline.FourHour:   "HOUR_4",
		kline.SixHour:    "HOUR_6",
		kline.TwelveHour: "HOUR_12",
		kline.OneDay:     "DAY_1",
		kline.ThreeDay:   "DAY_3",
		kline.SevenDay:   "WEEK_1",
		kline.OneMonth:   "MONTH_1",
	}
	intervalString, okay := intervalMap[interval]
	if okay {
		return intervalString, nil
	}
	return "", kline.ErrUnsupportedInterval
}

func stringToInterval(interval string) (kline.Interval, error) {
	intervalMap := map[string]kline.Interval{
		"MINUTE_1":  kline.OneMin,
		"MINUTE_5":  kline.FiveMin,
		"MINUTE_10": kline.TenMin,
		"MINUTE_15": kline.FifteenMin,
		"MINUTE_30": kline.ThirtyMin,
		"HOUR_1":    kline.OneHour,
		"HOUR_2":    kline.TwoHour,
		"HOUR_4":    kline.FourHour,
		"HOUR_6":    kline.SixHour,
		"HOUR_12":   kline.TwelveHour,
		"DAY_1":     kline.OneDay,
		"DAY_3":     kline.ThreeDay,
		"WEEK_1":    kline.SevenDay,
		"MONTH_1":   kline.OneMonth,
	}
	intervalInstance, okay := intervalMap[interval]
	if okay {
		return intervalInstance, nil
	}
	return kline.Interval(0), kline.ErrUnsupportedInterval
}
