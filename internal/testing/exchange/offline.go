package exchange

import (
	"context"
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

// offlinePrice is the price of every candle, trade and order book level Offline makes up
const offlinePrice = 1337

// cancelled asks the wrapped exchange for what it can answer without a request
var cancelled = func() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}()

// Offline stands in for the network calls of an exchange's live API in mock test runs. Each method first asks the
// wrapped exchange with a cancelled context, so an answer that needs no request, such as an unsupported function or
// a validation error, is returned unchanged; only a call that would reach the network gets made-up data instead,
// under the exchange's own name.
type Offline struct {
	exchange.IBotExchange
}

// OfflineIfMocked wraps an exchange as Offline in mock test runs
func OfflineIfMocked(e exchange.IBotExchange) exchange.IBotExchange {
	if livetest.Mock {
		return Offline{IBotExchange: e}
	}
	return e
}

// OfflineBuilder builds exchanges wrapped as Offline, for an engine exchange manager's Builder
type OfflineBuilder struct {
	// New builds the exchange to wrap, such as engine.NewSupportedExchangeByName
	New func(name string) (exchange.IBotExchange, error)
}

// NewExchangeByName returns the named exchange wrapped as Offline
func (b OfflineBuilder) NewExchangeByName(name string) (exchange.IBotExchange, error) {
	e, err := b.New(name)
	if err != nil {
		return nil, err
	}
	return Offline{IBotExchange: e}, nil
}

// answered reports whether the wrapped exchange answered without a request, which a cancelled context fails
func answered(err error) bool {
	return !errors.Is(err, context.Canceled)
}

// UpdateTradablePairs keeps the exchange's configured pairs instead of downloading them
func (o Offline) UpdateTradablePairs(ctx context.Context) error {
	if err := o.IBotExchange.UpdateTradablePairs(cancelled); answered(err) {
		return err
	}
	return ctx.Err()
}

// UpdateOrderExecutionLimits leaves the limits to GetOrderExecutionLimits instead of downloading them
func (o Offline) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if err := o.IBotExchange.UpdateOrderExecutionLimits(cancelled, a); answered(err) {
		return err
	}
	return ctx.Err()
}

// limitsMissing reports whether the exchange holds no limits for a pair, which Offline makes up instead
func limitsMissing(err error) bool {
	return errors.Is(err, limits.ErrExchangeLimitNotLoaded) || errors.Is(err, limits.ErrOrderLimitNotFound)
}

// GetOrderExecutionLimits returns the exchange's stored limits, or spot-like limits for a pair it holds none for
func (o Offline) GetOrderExecutionLimits(a asset.Item, p currency.Pair) (limits.MinMaxLevel, error) {
	if l, err := o.IBotExchange.GetOrderExecutionLimits(a, p); !limitsMissing(err) {
		return l, err
	}
	return limits.MinMaxLevel{
		Key:                     key.NewExchangeAssetPair(o.GetName(), a, p),
		MinPrice:                0.01,
		MaxPrice:                10_000_000,
		PriceStepIncrementSize:  0.01,
		MinimumBaseAmount:       0.0001,
		MaximumBaseAmount:       10_000,
		AmountStepIncrementSize: 0.00000001,
	}, nil
}

// CheckOrderExecutionLimits checks an order against the exchange's stored limits, or against the limits
// GetOrderExecutionLimits makes up for a pair it holds none for
func (o Offline) CheckOrderExecutionLimits(a asset.Item, cp currency.Pair, price, amount float64, orderType order.Type) error {
	if err := o.IBotExchange.CheckOrderExecutionLimits(a, cp, price, amount, orderType); !limitsMissing(err) {
		return err
	}
	l, err := o.GetOrderExecutionLimits(a, cp)
	if err != nil {
		return err
	}
	return l.Validate(price, amount, orderType)
}

// GetHistoricCandles returns a candle for each interval from start until end
func (o Offline) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if item, err := o.IBotExchange.GetHistoricCandles(cancelled, p, a, interval, start, end); answered(err) {
		return item, err
	}
	return o.candles(ctx, p, a, interval, start, end)
}

// GetHistoricCandlesExtended returns a candle for each interval from start until end
func (o Offline) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if item, err := o.IBotExchange.GetHistoricCandlesExtended(cancelled, p, a, interval, start, end); answered(err) {
		return item, err
	}
	return o.candles(ctx, p, a, interval, start, end)
}

// candles aligns the range and processes the made-up candles as a kline request does, so they start on interval
// boundaries and an invalid range or interval fails as it would against the venue
func (o Offline) candles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	req, err := kline.CreateKlineRequest(o.GetName(), p, p, a, interval, interval, start, end, 1)
	if err != nil {
		return nil, err
	}
	ranges, err := kline.CalculateCandleDateRanges(req.Start, req.End, interval, 0)
	if err != nil {
		return nil, err
	}
	var candles []kline.Candle
	for _, r := range ranges.Ranges {
		for _, i := range r.Intervals {
			candles = append(candles, kline.Candle{Time: i.Start.Time, Open: offlinePrice, High: offlinePrice, Low: offlinePrice, Close: offlinePrice, Volume: 1})
		}
	}
	return req.ProcessResponse(candles)
}

// GetRecentTrades returns one trade for the pair
func (o Offline) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	if trades, err := o.IBotExchange.GetRecentTrades(cancelled, p, a); answered(err) {
		return trades, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []trade.Data{{Exchange: o.GetName(), CurrencyPair: p, AssetType: a, Side: order.Buy, Price: offlinePrice, Amount: 1, Timestamp: time.Now()}}, nil
}

// GetHistoricTrades returns a trade for each minute from start until end
func (o Offline) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time) ([]trade.Data, error) {
	if trades, err := o.IBotExchange.GetHistoricTrades(cancelled, p, a, start, end); answered(err) {
		return trades, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	var trades []trade.Data
	for ts := start; ts.Before(end); ts = ts.Add(time.Minute) {
		trades = append(trades, trade.Data{Exchange: o.GetName(), CurrencyPair: p, AssetType: a, Side: order.Buy, Price: offlinePrice, Amount: 1, Timestamp: ts})
	}
	return trades, nil
}

// UpdateOrderbook stores and returns a book ten levels deep on each side of the made-up price
func (o Offline) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if book, err := o.IBotExchange.UpdateOrderbook(cancelled, p, a); answered(err) {
		return book, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	book := &orderbook.Book{Exchange: o.GetName(), Pair: p, Asset: a, LastUpdated: time.Now()}
	for i := range 10 {
		step := float64(i+1) * 0.01
		book.Bids = append(book.Bids, orderbook.Level{Price: offlinePrice - step, Amount: 1})
		book.Asks = append(book.Asks, orderbook.Level{Price: offlinePrice + step, Amount: 1})
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(o.GetName(), p, a)
}
