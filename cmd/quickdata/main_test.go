package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/quickdata"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// captureOutput redirects os.Stdout for the duration of f and returns what was written.
func captureOutput(f func()) string {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = orig
	return buf.String()
}

func TestIntMin(t *testing.T) {
	require.Equal(t, 1, intMin(1, 2))
	require.Equal(t, 1, intMin(2, 1))
	require.Equal(t, -1, intMin(-1, 5))
}

func TestParseFocusType(t *testing.T) {
	t.Parallel()
	type caseDef struct {
		inputs   []string
		expected quickdata.FocusType
		error    error
	}
	cases := []caseDef{
		{inputs: []string{"ticker", "tick", "TICK"}, expected: quickdata.TickerFocusType},
		{inputs: []string{"orderbook", "order_book", "ob", "book"}, expected: quickdata.OrderBookFocusType},
		{inputs: []string{"kline", "candles", "candle", "ohlc"}, expected: quickdata.KlineFocusType},
		{inputs: []string{"trades", "trade"}, expected: quickdata.TradesFocusType},
		{inputs: []string{"openinterest", "oi"}, expected: quickdata.OpenInterestFocusType},
		{inputs: []string{"fundingrate", "funding"}, expected: quickdata.FundingRateFocusType},
		{inputs: []string{"accountholdings", "account", "holdings", "balances"}, expected: quickdata.AccountHoldingsFocusType},
		{inputs: []string{"activeorders", "orders"}, expected: quickdata.ActiveOrdersFocusType},
		{inputs: []string{"orderexecution", "executionlimits", "limits"}, expected: quickdata.OrderLimitsFocusType},
		{inputs: []string{"url", "tradeurl", "trade_url"}, expected: quickdata.URLFocusType},
		{inputs: []string{"contract"}, expected: quickdata.ContractFocusType},
		{inputs: []string{"butts"}, error: quickdata.ErrUnsupportedFocusType},
	}
	for _, tc := range cases {
		for _, in := range tc.inputs {
			t.Run(in, func(t *testing.T) {
				t.Parallel()
				resp, err := parseFocusType(in)
				if tc.error != nil {
					require.ErrorIs(t, err, tc.error)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.expected, resp)
				}
			})
		}
	}
}

func TestRenderFunctions_NoPanicAndOutput(t *testing.T) {
	// orderbook
	ob := &orderbook.Book{Bids: orderbook.Levels{{Price: 50000, Amount: 1}}, Asks: orderbook.Levels{{Price: 50010, Amount: 2}}}
	out := captureOutput(func() { renderOrderbook(ob, 5) })
	assert.Contains(t, out, "Price")

	// ticker
	tp := &ticker.Price{Last: 100, Bid: 99.5, Ask: 100.5, Volume: 1234.56, MarkPrice: 100.2, IndexPrice: 100.1}
	_ = captureOutput(func() { renderTicker(tp) })
	_ = captureOutput(func() { renderTicker(nil) })

	// trades
	trs := []trade.Data{{Price: 10, Amount: 1, Timestamp: time.Now().Add(-time.Minute), Side: order.Buy}, {Price: 11, Amount: 2, Timestamp: time.Now(), Side: order.Sell}}
	_ = captureOutput(func() { renderTrades(trs) })
	_ = captureOutput(func() { renderTrades(nil) })

	// klines
	kl := []websocket.KlineData{{Timestamp: time.Now().Add(-2 * time.Minute), OpenPrice: 10, ClosePrice: 11, HighPrice: 11.5, LowPrice: 9.5, Volume: 100, Interval: "1m"}, {Timestamp: time.Now().Add(-time.Minute), OpenPrice: 11, ClosePrice: 12, HighPrice: 12.5, LowPrice: 10.5, Volume: 150, Interval: "1m"}}
	_ = captureOutput(func() { renderKlines(kl) })
	_ = captureOutput(func() { renderKlines(nil) })

	// account holdings
	holdings := &account.Holdings{Exchange: "binance", Accounts: []account.SubAccount{{ID: "spot", AssetType: asset.Spot, Currencies: []account.Balance{{Currency: currency.BTC, Total: 1, Free: 0.5, Hold: 0.5}}}}}
	_ = captureOutput(func() { renderAccountHoldings(holdings) })
	_ = captureOutput(func() { renderAccountHoldings(nil) })

	// active orders
	ords := []order.Detail{{OrderID: "1", Type: order.Limit, Side: order.Buy, Price: 10, Amount: 5, ExecutedAmount: 2, Status: order.Active, LastUpdated: time.Now()}}
	_ = captureOutput(func() { renderActiveOrders(ords) })
	_ = captureOutput(func() { renderActiveOrders(nil) })

	// execution limits
	mm := &limits.MinMaxLevel{MinPrice: 1, MaxPrice: 2, MinimumBaseAmount: 0.1, MaximumBaseAmount: 10, MinimumQuoteAmount: 1, MaximumQuoteAmount: 1000, MinNotional: 5}
	_ = captureOutput(func() { renderExecutionLimits(mm) })
	_ = captureOutput(func() { renderExecutionLimits(nil) })

	// URL
	_ = captureOutput(func() { renderURL("https://example.com") })

	// contract
	ctr := &futures.Contract{Name: currency.NewBTCUSDT(), StartDate: time.Now().Add(-time.Hour), EndDate: time.Now().Add(time.Hour), Multiplier: 1.0, SettlementCurrencies: currency.Currencies{currency.USDT}}
	_ = captureOutput(func() { renderContract(ctr) })
	_ = captureOutput(func() { renderContract(nil) })

	// misc
	_ = captureOutput(func() { renderOpenInterest(123.45) })
	fr := &fundingrate.LatestRateResponse{LatestRate: fundingrate.Rate{Rate: decimal.NewFromFloat(0.0001)}, PredictedUpcomingRate: fundingrate.Rate{Rate: decimal.NewFromFloat(0.0002)}, TimeOfNextRate: time.Now().Add(time.Hour), TimeChecked: time.Now()}
	_ = captureOutput(func() { renderFundingRate(fr) })
	_ = captureOutput(func() { renderFundingRate(nil) })

	// pretty payload dispatcher exercise all types
	payloads := []any{
		ob,
		[]ticker.Price{*tp},
		tp,
		kl,
		trs,
		trs[0],
		holdings,
		ords,
		float64(42),
		fr,
		ctr,
		mm,
		"https://example.com",
		struct{ A int }{A: 5},
	}
	for i, p := range payloads {
		_ = captureOutput(func() { renderPrettyPayload(p, 10) })
		_ = captureOutput(func() { renderPrettyPayload(p, 0) })
		if i == 0 {
			out2 := captureOutput(func() { renderPrettyPayload(p, 2) })
			assert.NotEmpty(t, out2)
		}
	}
}

func TestStreamDataCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan any, 1)
	cfg := &appConfig{FocusType: quickdata.TickerFocusType, JSONOnly: true}
	ch <- "test-data"
	done := make(chan error, 1)
	go func() { done <- streamData(ctx, ch, cfg) }()
	// Allow goroutine to process the first item
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "streamData did not return after cancellation")
	}
}

func TestParseFlags_Success(t *testing.T) {
	// backup and restore os.Args and flag.CommandLine
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"quickData", "--exchange", "BiNaNcE", "--asset", "spot", "--pair", "ETH-BTC", "--data", "ticker", "--poll", "10s", "--book-levels", "20", "--json"}
	cfg := parseFlags()
	require.Equal(t, "binance", cfg.Exchange)
	require.Equal(t, asset.Spot, cfg.Asset)
	require.Equal(t, "ETH-BTC", cfg.Pair.String())
	require.Equal(t, quickdata.TickerFocusType, cfg.FocusType)
	require.Equal(t, 10*time.Second, cfg.PollInterval)
	require.Equal(t, 20, cfg.BookLevels)
	require.True(t, cfg.JSONOnly)
	require.Nil(t, cfg.Credentials)
}

func TestParseFlags_AuthRequired(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"quickData", "--exchange", "okx", "--asset", "spot", "--pair", "BTC-USDT", "--data", "account", "--apiKey", "k123", "--apiSecret", "s456"}
	cfg := parseFlags()
	require.Equal(t, quickdata.AccountHoldingsFocusType, cfg.FocusType)
	require.NotNil(t, cfg.Credentials)
	require.Equal(t, "k123", cfg.Credentials.Key)
	require.Equal(t, "s456", cfg.Credentials.Secret)
}

func TestStreamData_DefaultRendering(t *testing.T) {
	pair := currency.NewBTCUSDT()
	// Custom capture because global encoder 'enc' was initialised with original os.Stdout
	origStd := os.Stdout
	origEnc := enc
	r, w, _ := os.Pipe()
	os.Stdout = w
	enc = json.NewEncoder(os.Stdout)
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan any, 1)
	cfg := &appConfig{Exchange: "testexch", Asset: asset.Spot, Pair: pair, FocusType: quickdata.TradesFocusType, JSONOnly: false, BookLevels: 10}
	ch <- []trade.Data{{Price: 101, Amount: 0.5, Timestamp: time.Now(), Side: order.Buy}}
	done := make(chan error, 1)
	go func() { done <- streamData(ctx, ch, cfg) }()
	time.Sleep(80 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "streamData did not terminate after cancel")
	}
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = origStd
	enc = origEnc
	out := buf.String()
	assert.Contains(t, out, "testexch")
	assert.Condition(t, func() bool { return strings.Contains(out, "BTCUSDT") || strings.Contains(out, "BTC-USDT") }, "pair missing")
	assert.Contains(t, out, "Trades:")
	assert.Contains(t, out, "\"focus\":\"TradesFocusType\"")
}
