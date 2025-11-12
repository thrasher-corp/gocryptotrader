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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/cmd/quickdata/app"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

func TestIntMin(t *testing.T) {
	require.Equal(t, 1, intMin(1, 2))
	require.Equal(t, 1, intMin(2, 1))
	require.Equal(t, -1, intMin(-1, 5))
}

func TestParseFocusType(t *testing.T) {
	t.Parallel()
	type caseDef struct {
		inputs   []string
		expected app.FocusType
		error    error
	}
	cases := []caseDef{
		{inputs: []string{"ticker", "tick", "TICK"}, expected: app.TickerFocusType},
		{inputs: []string{"orderbook", "order_book", "ob", "book"}, expected: app.OrderBookFocusType},
		{inputs: []string{"kline", "candles", "candle", "ohlc"}, expected: app.KlineFocusType},
		{inputs: []string{"trades", "trade"}, expected: app.TradesFocusType},
		{inputs: []string{"openinterest", "oi"}, expected: app.OpenInterestFocusType},
		{inputs: []string{"fundingrate", "funding"}, expected: app.FundingRateFocusType},
		{inputs: []string{"accountholdings", "account", "holdings", "balances"}, expected: app.AccountHoldingsFocusType},
		{inputs: []string{"activeorders", "orders"}, expected: app.ActiveOrdersFocusType},
		{inputs: []string{"orderexecution", "executionlimits", "limits"}, expected: app.OrderLimitsFocusType},
		{inputs: []string{"url", "tradeurl", "trade_url"}, expected: app.URLFocusType},
		{inputs: []string{"contract"}, expected: app.ContractFocusType},
		{inputs: []string{"butts"}, error: app.ErrUnsupportedFocusType},
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

func TestStreamDataCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan any, 1)
	cfg := &appConfig{FocusType: app.TickerFocusType, JSONOnly: true}
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
	assert.Equal(t, "binance", cfg.Exchange)
	assert.Equal(t, asset.Spot, cfg.Asset)
	assert.Equal(t, "ETH-BTC", cfg.Pair.String())
	assert.Equal(t, app.TickerFocusType, cfg.FocusType)
	assert.Equal(t, 10*time.Second, cfg.PollInterval)
	assert.Equal(t, 20, cfg.BookLevels)
	assert.True(t, cfg.JSONOnly)
	assert.Nil(t, cfg.Credentials)
}

func TestParseFlags_AuthRequired(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"quickData", "--exchange", "okx", "--asset", "spot", "--pair", "BTC-USDT", "--data", "account", "--apiKey", "k123", "--apiSecret", "s456"}
	cfg := parseFlags()
	require.Equal(t, app.AccountHoldingsFocusType, cfg.FocusType)
	assert.NotNil(t, cfg.Credentials)
	assert.Equal(t, "k123", cfg.Credentials.Key)
	assert.Equal(t, "s456", cfg.Credentials.Secret)
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
	cfg := &appConfig{Exchange: "testexch", Asset: asset.Spot, Pair: pair, FocusType: app.TradesFocusType, JSONOnly: false, BookLevels: 10}
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
