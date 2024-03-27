package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Without subscriptions
	binanceEOptionWebsocketURL       = "wss://nbstream.binance.com/eoptions/ws"
	binanceEOptionWebsocketURLStream = "wss://nbstream.binance.com/eoptions/stream"
)

// CheckEOptionsServerTime retrieves the server time.
func (b *Binance) CheckEOptionsServerTime(ctx context.Context) (convert.ExchangeTime, error) {
	resp := &struct {
		ServerTime convert.ExchangeTime `json:"serverTime"`
	}{}
	return resp.ServerTime, b.SendHTTPRequest(ctx, exchange.RestOptions, "/eapi/v1/time", spotDefaultRate, &resp)
}

// GetOptionsExchangeInformation retrieves an exchange information through the options endpoint.
func (b *Binance) GetOptionsExchangeInformation(ctx context.Context) (*EOptionExchangeInfo, error) {
	var resp *EOptionExchangeInfo
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, "/eapi/v1/exchangeInfo", spotDefaultRate, &resp)
}

// GetEOptionsOrderbook retrieves european options orderbook information for specific symbol
func (b *Binance) GetEOptionsOrderbook(ctx context.Context, symbol string, limit int64) (*EOptionsOrderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *EOptionsOrderbook
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/depth", params), spotDefaultRate, &resp)
}

// GetEOptionsRecentTrades retrieves recent market trades
func (b *Binance) GetEOptionsRecentTrades(ctx context.Context, symbol string, limit int64) ([]EOptionsTradeItem, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []EOptionsTradeItem
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/trades", params), spotDefaultRate, &resp)
}

// GetEOptionsTradeHistory retrieves older market historical trades.
func (b *Binance) GetEOptionsTradeHistory(ctx context.Context, symbol string, fromID, limit int64) ([]EOptionsTradeItem, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []EOptionsTradeItem
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/historicalTrades", params), spotDefaultRate, &resp)
}

// GetEOptionsCandlesticks retrieves kline/candlestick bars for an option symbol. Klines are uniquely identified by their open time.
func (b *Binance) GetEOptionsCandlesticks(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]EOptionsCandlestick, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if interval == 0 || interval.String() == "" {
		return nil, kline.ErrInvalidInterval
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", b.FormatExchangeKlineInterval(interval))
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []EOptionsCandlestick
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/klines", params), spotDefaultRate, &resp)
}

// GetOptionMarkPrice option mark price and greek info.
func (b *Binance) GetOptionMarkPrice(ctx context.Context, symbol string) ([]OptionMarkPrice, error) {
	params := url.Values{}
	if symbol == "" {
		params.Set("symbol", symbol)
	}
	var resp []OptionMarkPrice
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/mark", params), spotDefaultRate, &resp)
}

// GetEOptions24hrTickerPriceChangeStatistics 24 hour rolling window price change statistics.
func (b *Binance) GetEOptions24hrTickerPriceChangeStatistics(ctx context.Context, symbol string) ([]EOptionTicker, error) {
	params := url.Values{}
	if symbol == "" {
		params.Set("symbol", symbol)
	}
	var resp []EOptionTicker
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/ticker", params), spotDefaultRate, &resp)
}

// GetEOptionsSymbolPriceTicker represents a symbol ticker instances.
func (b *Binance) GetEOptionsSymbolPriceTicker(ctx context.Context, underlying string) (*EOptionIndexSymbolPriceTicker, error) {
	if underlying == "" {
		return nil, errors.New("underlying is required")
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	var resp *EOptionIndexSymbolPriceTicker
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/index", params), spotDefaultRate, &resp)
}

// GetEOptionsHistoricalExerciseRecords retrieves historical exercise records.
func (b *Binance) GetEOptionsHistoricalExerciseRecords(ctx context.Context, underlying string, startTime, endTime time.Time, limit int64) ([]ExerciseHistoryItem, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []ExerciseHistoryItem
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/exerciseHistory", params), spotDefaultRate, &resp)
}

// GetEOptionsOpenInterests retrieves  open interest for specific underlying asset on specific expiration date.
func (b *Binance) GetEOptionsOpenInterests(ctx context.Context, underlyingAsset string, expiration time.Time) ([]OpenInterest, error) {
	if underlyingAsset == "" {
		return nil, errors.New("underlying asset is required")
	}
	if expiration.IsZero() {
		return nil, errors.New("expiration time is required")
	}
	params := url.Values{}
	params.Set("underlyingAsset", underlyingAsset)
	params.Set("expiration", fmt.Sprintf("%d%s%d", expiration.Day(), expiration.Month(), (expiration.Year()%2000)))
	var resp []OpenInterest
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/openInterest", params), spotDefaultRate, &resp)
}

// ----------------------------------------------------------- Account trade endpoints ---------------------------------------------------------------------

// GetOptionsAccountInformation retrieves the current account information.
func (b *Binance) GetOptionsAccountInformation(ctx context.Context) (*EOptionsAccountInformation, error) {
	var resp *EOptionsAccountInformation
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, "/eapi/v1/account", spotDefaultRate, &resp)
}

// NewOptionsOrder places a new european options order instance.
func (b *Binance) NewOptionsOrder(ctx context.Context, arg *OptionsOrderParams) (*OptionOrder, error) {
	if arg == nil || *arg == (OptionsOrderParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol.String())
	params.Set("side", arg.Side)
	params.Set("type", arg.OrderType)
	params.Set("quantity", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	arg.OrderType = strings.ToUpper(arg.OrderType)
	if arg.OrderType == "LIMIT" && arg.Price <= 0 {
		return nil, fmt.Errorf("%w, price is required for limit orders", order.ErrPriceBelowMin)
	}
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.PostOnly {
		params.Set("postOnly", "true")
	}
	if arg.NewOrderResponseType != "" {
		params.Set("newOrderRespType", arg.NewOrderResponseType)
	}
	if arg.ClientOrderID != "" {
		params.Set("clientOrderId", arg.ClientOrderID)
	}
	if arg.IsMarketMakerProtection {
		params.Set("isMmp", "true")
	}
	var resp *OptionOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/order", params, spotDefaultRate, &resp)
}
