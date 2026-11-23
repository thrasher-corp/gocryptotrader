package exchange

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

var (
	errAnswered          = errors.New("answered without a request")
	errBuild             = errors.New("build failed")
	errUnexpectedRequest = errors.New("unexpected request")
)

// venue answers like a live exchange: a call that needs a request fails on a cancelled context unless answer is set,
// as it is for an unsupported function or a validation failure, and any other context would send the request
type venue struct {
	sharedtestvalues.CustomEx
	answer    error
	limits    limits.MinMaxLevel
	limitsErr error
}

func (v *venue) request(ctx context.Context) error {
	if v.answer != nil {
		return v.answer
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	return errUnexpectedRequest
}

func (v *venue) UpdateTradablePairs(ctx context.Context) error {
	return v.request(ctx)
}

func (v *venue) UpdateOrderExecutionLimits(ctx context.Context, _ asset.Item) error {
	return v.request(ctx)
}

func (v *venue) GetOrderExecutionLimits(asset.Item, currency.Pair) (limits.MinMaxLevel, error) {
	return v.limits, v.limitsErr
}

func (v *venue) CheckOrderExecutionLimits(_ asset.Item, _ currency.Pair, price, amount float64, orderType order.Type) error {
	if v.limitsErr != nil {
		return v.limitsErr
	}
	return v.limits.Validate(price, amount, orderType)
}

func (v *venue) GetHistoricCandles(ctx context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, v.request(ctx)
}

func (v *venue) GetHistoricCandlesExtended(ctx context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, v.request(ctx)
}

func (v *venue) GetRecentTrades(ctx context.Context, _ currency.Pair, _ asset.Item) ([]trade.Data, error) {
	return nil, v.request(ctx)
}

func (v *venue) GetHistoricTrades(ctx context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, v.request(ctx)
}

func (v *venue) UpdateOrderbook(ctx context.Context, _ currency.Pair, _ asset.Item) (*orderbook.Book, error) {
	return nil, v.request(ctx)
}

// testProbe checks a call Offline probes returns the venue's answer when it needs no request, and the caller's
// context error instead of made-up data
func testProbe(t *testing.T, call func(context.Context, Offline) error) {
	t.Helper()
	t.Run("answer without a request", func(t *testing.T) {
		t.Parallel()
		err := call(t.Context(), Offline{IBotExchange: &venue{answer: errAnswered}})
		assert.ErrorIs(t, err, errAnswered, "Offline should return the venue's answer")
	})
	t.Run("cancelled caller context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := call(ctx, Offline{IBotExchange: &venue{}})
		assert.ErrorIs(t, err, context.Canceled, "Offline should return the caller's context error")
	})
}

func TestOfflineIfMocked(t *testing.T) {
	t.Parallel()
	v := &venue{}
	if livetest.Mock {
		assert.Equal(t, Offline{IBotExchange: v}, OfflineIfMocked(v), "OfflineIfMocked should wrap the exchange in mock runs")
		return
	}
	assert.Same(t, v, OfflineIfMocked(v), "OfflineIfMocked should return the exchange unchanged in live runs")
}

func TestOfflineBuilderNewExchangeByName(t *testing.T) {
	t.Parallel()

	t.Run("wraps the built exchange", func(t *testing.T) {
		t.Parallel()
		v := &venue{}
		var built string
		e, err := OfflineBuilder{New: func(name string) (exchange.IBotExchange, error) {
			built = name
			return v, nil
		}}.NewExchangeByName("venue")
		require.NoError(t, err, "NewExchangeByName must not error")
		assert.Equal(t, "venue", built, "NewExchangeByName should build the named exchange")
		assert.Equal(t, Offline{IBotExchange: v}, e, "NewExchangeByName should wrap the built exchange")
	})

	t.Run("build error", func(t *testing.T) {
		t.Parallel()
		e, err := OfflineBuilder{New: func(string) (exchange.IBotExchange, error) {
			return nil, errBuild
		}}.NewExchangeByName("venue")
		assert.ErrorIs(t, err, errBuild, "NewExchangeByName should return the build error")
		assert.Nil(t, e, "NewExchangeByName should not return an exchange")
	})
}

func TestAnswered(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "no error", want: true},
		{name: "answer", err: errAnswered, want: true},
		{name: "cancelled", err: context.Canceled},
		{name: "wrapped cancellation", err: fmt.Errorf("sending request: %w", context.Canceled)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, answered(tc.err), "answered should report whether the venue answered")
		})
	}
}

func TestOfflineUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testProbe(t, func(ctx context.Context, o Offline) error {
		return o.UpdateTradablePairs(ctx)
	})
	t.Run("no request", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, Offline{IBotExchange: &venue{}}.UpdateTradablePairs(t.Context()), "UpdateTradablePairs should not error")
	})
}

func TestOfflineUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	testProbe(t, func(ctx context.Context, o Offline) error {
		return o.UpdateOrderExecutionLimits(ctx, asset.Spot)
	})
	t.Run("no request", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, Offline{IBotExchange: &venue{}}.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits should not error")
	})
}

func TestLimitsMissing(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "no error"},
		{name: "exchange limits not loaded", err: limits.ErrExchangeLimitNotLoaded, want: true},
		{name: "pair limit not found", err: fmt.Errorf("%w for pair", limits.ErrOrderLimitNotFound), want: true},
		{name: "other error", err: errAnswered},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, limitsMissing(tc.err), "limitsMissing should report whether the limits are missing")
		})
	}
}

func TestOfflineGetOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	p := currency.NewBTCUSDT()

	t.Run("stored limits", func(t *testing.T) {
		t.Parallel()
		stored := limits.MinMaxLevel{Key: key.NewExchangeAssetPair("customex", asset.Spot, p), MinPrice: 5}
		l, err := Offline{IBotExchange: &venue{limits: stored}}.GetOrderExecutionLimits(asset.Spot, p)
		require.NoError(t, err, "GetOrderExecutionLimits must not error")
		assert.Equal(t, stored, l, "GetOrderExecutionLimits should return the stored limits")
	})

	t.Run("other errors", func(t *testing.T) {
		t.Parallel()
		_, err := Offline{IBotExchange: &venue{limitsErr: errAnswered}}.GetOrderExecutionLimits(asset.Spot, p)
		assert.ErrorIs(t, err, errAnswered, "GetOrderExecutionLimits should return the exchange's error")
	})

	for _, missing := range []error{limits.ErrExchangeLimitNotLoaded, limits.ErrOrderLimitNotFound} {
		t.Run(missing.Error(), func(t *testing.T) {
			t.Parallel()
			l, err := Offline{IBotExchange: &venue{limitsErr: missing}}.GetOrderExecutionLimits(asset.Spot, p)
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			assert.Equal(t, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair("customex", asset.Spot, p),
				MinPrice:                0.01,
				MaxPrice:                10_000_000,
				PriceStepIncrementSize:  0.01,
				MinimumBaseAmount:       0.0001,
				MaximumBaseAmount:       10_000,
				AmountStepIncrementSize: 0.00000001,
			}, l, "GetOrderExecutionLimits should make up limits for the pair")
		})
	}
}

func TestOfflineCheckOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	p := currency.NewBTCUSDT()
	for _, tc := range []struct {
		name          string
		venue         *venue
		price, amount float64
		err           error
	}{
		{name: "within stored limits", venue: &venue{limits: limits.MinMaxLevel{MinPrice: 5}}, price: 10, amount: 1},
		{name: "outside stored limits", venue: &venue{limits: limits.MinMaxLevel{MinPrice: 5}}, price: 1, amount: 1, err: limits.ErrPriceBelowMin},
		{name: "other errors", venue: &venue{limitsErr: errAnswered}, price: 10, amount: 1, err: errAnswered},
		{name: "within made-up limits", venue: &venue{limitsErr: limits.ErrExchangeLimitNotLoaded}, price: 10, amount: 1},
		{name: "outside made-up limits", venue: &venue{limitsErr: limits.ErrOrderLimitNotFound}, price: 0.001, amount: 1, err: limits.ErrPriceBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := Offline{IBotExchange: tc.venue}.CheckOrderExecutionLimits(asset.Spot, p, tc.price, tc.amount, order.Limit)
			assert.ErrorIs(t, err, tc.err, "CheckOrderExecutionLimits should return the expected error")
		})
	}
}

type candlesCall func(Offline, context.Context, currency.Pair, asset.Item, kline.Interval, time.Time, time.Time) (*kline.Item, error)

// testCandles checks a candle call makes up candles on interval boundaries over the range a kline request aligns
func testCandles(t *testing.T, call candlesCall) {
	t.Helper()
	p := currency.NewBTCUSDT()
	start := time.Date(2026, 1, 2, 3, 30, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)
	testProbe(t, func(ctx context.Context, o Offline) error {
		_, err := call(o, ctx, p, asset.Spot, kline.OneHour, start, end)
		return err
	})

	t.Run("candles on interval boundaries", func(t *testing.T) {
		t.Parallel()
		item, err := call(Offline{IBotExchange: &venue{}}, t.Context(), p, asset.Spot, kline.OneHour, start, end)
		require.NoError(t, err, "candles must not error")
		want := &kline.Item{Exchange: "customex", Pair: p, Asset: asset.Spot, Interval: kline.OneHour}
		for hour := 3; hour <= 6; hour++ {
			want.Candles = append(want.Candles, kline.Candle{Time: time.Date(2026, 1, 2, hour, 0, 0, 0, time.UTC), Open: offlinePrice, High: offlinePrice, Low: offlinePrice, Close: offlinePrice, Volume: 1})
		}
		assert.Equal(t, want, item, "candles should cover the aligned range")
	})

	t.Run("invalid interval", func(t *testing.T) {
		t.Parallel()
		_, err := call(Offline{IBotExchange: &venue{}}, t.Context(), p, asset.Spot, kline.Raw, start, end)
		assert.ErrorIs(t, err, kline.ErrInvalidInterval, "candles should reject an interval without a duration")
	})

	t.Run("start after end", func(t *testing.T) {
		t.Parallel()
		_, err := call(Offline{IBotExchange: &venue{}}, t.Context(), p, asset.Spot, kline.OneHour, end, start)
		assert.ErrorIs(t, err, common.ErrStartAfterEnd, "candles should reject a reversed range")
	})
}

func TestOfflineGetHistoricCandles(t *testing.T) {
	t.Parallel()
	testCandles(t, Offline.GetHistoricCandles)
}

func TestOfflineGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	testCandles(t, Offline.GetHistoricCandlesExtended)
}

func TestOfflineGetRecentTrades(t *testing.T) {
	t.Parallel()
	p := currency.NewBTCUSDT()
	testProbe(t, func(ctx context.Context, o Offline) error {
		_, err := o.GetRecentTrades(ctx, p, asset.Spot)
		return err
	})

	t.Run("one trade", func(t *testing.T) {
		t.Parallel()
		trades, err := Offline{IBotExchange: &venue{}}.GetRecentTrades(t.Context(), p, asset.Spot)
		require.NoError(t, err, "GetRecentTrades must not error")
		require.Len(t, trades, 1, "GetRecentTrades must return one trade")
		assert.NotZero(t, trades[0].Timestamp, "trade should have a timestamp")
		trades[0].Timestamp = time.Time{}
		assert.Equal(t, trade.Data{Exchange: "customex", CurrencyPair: p, AssetType: asset.Spot, Side: order.Buy, Price: offlinePrice, Amount: 1}, trades[0], "GetRecentTrades should make up the trade")
	})
}

func TestOfflineGetHistoricTrades(t *testing.T) {
	t.Parallel()
	p := currency.NewBTCUSDT()
	start := time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)
	testProbe(t, func(ctx context.Context, o Offline) error {
		_, err := o.GetHistoricTrades(ctx, p, asset.Spot, start, start.Add(3*time.Minute))
		return err
	})

	t.Run("a trade each minute", func(t *testing.T) {
		t.Parallel()
		trades, err := Offline{IBotExchange: &venue{}}.GetHistoricTrades(t.Context(), p, asset.Spot, start, start.Add(3*time.Minute))
		require.NoError(t, err, "GetHistoricTrades must not error")
		want := make([]trade.Data, 3)
		for i := range want {
			want[i] = trade.Data{Exchange: "customex", CurrencyPair: p, AssetType: asset.Spot, Side: order.Buy, Price: offlinePrice, Amount: 1, Timestamp: start.Add(time.Duration(i) * time.Minute)}
		}
		assert.Equal(t, want, trades, "GetHistoricTrades should make up a trade for each minute")
	})

	t.Run("unset start", func(t *testing.T) {
		t.Parallel()
		_, err := Offline{IBotExchange: &venue{}}.GetHistoricTrades(t.Context(), p, asset.Spot, time.Time{}, start)
		assert.ErrorIs(t, err, common.ErrDateUnset, "GetHistoricTrades should reject an unset start")
	})
}

func TestOfflineUpdateOrderbook(t *testing.T) {
	t.Parallel()
	p := currency.NewBTCUSDT()
	testProbe(t, func(ctx context.Context, o Offline) error {
		_, err := o.UpdateOrderbook(ctx, p, asset.Spot)
		return err
	})

	t.Run("stores a book around the made-up price", func(t *testing.T) {
		t.Parallel()
		book, err := Offline{IBotExchange: &venue{}}.UpdateOrderbook(t.Context(), p, asset.Spot)
		require.NoError(t, err, "UpdateOrderbook must not error")
		require.Len(t, book.Bids, 10, "book must have ten bids")
		require.Len(t, book.Asks, 10, "book must have ten asks")
		for i := range 10 {
			step := float64(i+1) * 0.01
			assert.InDeltaf(t, offlinePrice-step, book.Bids[i].Price, 1e-9, "bid %d should step down from the made-up price", i)
			assert.InDeltaf(t, offlinePrice+step, book.Asks[i].Price, 1e-9, "ask %d should step up from the made-up price", i)
			assert.Equalf(t, 1.0, book.Bids[i].Amount, "bid %d should have an amount of one", i)
			assert.Equalf(t, 1.0, book.Asks[i].Amount, "ask %d should have an amount of one", i)
		}
		stored, err := orderbook.Get("customex", p, asset.Spot)
		require.NoError(t, err, "orderbook.Get must not error")
		assert.Equal(t, book.Bids, stored.Bids, "stored bids should match the returned book")
		assert.Equal(t, book.Asks, stored.Asks, "stored asks should match the returned book")
	})
}
