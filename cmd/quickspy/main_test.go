package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/quickspy"
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

func TestParseFocusType(t *testing.T) {
	cases := []struct {
		in    string
		ft    quickspy.FocusType
		useWS bool
	}{
		{"ticker", quickspy.TickerFocusType, true},
		{"orderbook", quickspy.OrderBookFocusType, true},
		{"ob", quickspy.OrderBookFocusType, true},
		{"kline", quickspy.KlineFocusType, false},
		{"trades", quickspy.TradesFocusType, false},
		{"openinterest", quickspy.OpenInterestFocusType, false},
		{"fundingrate", quickspy.FundingRateFocusType, false},
		{"accountholdings", quickspy.AccountHoldingsFocusType, false},
		{"activeorders", quickspy.ActiveOrdersFocusType, false},
		{"orderexecution", quickspy.OrderExecutionFocusType, false},
		{"url", quickspy.URLFocusType, false},
		{"contract", quickspy.ContractFocusType, false},
		{"unknown", quickspy.UnsetFocusType, false},
	}
	for _, c := range cases {
		got, ws := parseFocusType(c.in)
		if got != c.ft || ws != c.useWS {
			t.Fatalf("parseFocusType(%q) = (%v,%v), want (%v,%v)", c.in, got, ws, c.ft, c.useWS)
		}
	}
}

func TestFallbackPoll(t *testing.T) {
	if got := fallbackPoll(0); got != 5*time.Second {
		t.Fatalf("fallbackPoll(0) = %v, want 5s", got)
	}
	if got := fallbackPoll(-1 * time.Second); got != 5*time.Second {
		t.Fatalf("fallbackPoll(-1s) = %v, want 5s", got)
	}
	if got := fallbackPoll(1500 * time.Millisecond); got != 1500*time.Millisecond {
		t.Fatalf("fallbackPoll(1.5s) = %v, want 1.5s", got)
	}
}

func TestRequiresAuth(t *testing.T) {
	if !requiresAuth(quickspy.AccountHoldingsFocusType) {
		t.Fatalf("requiresAuth(AccountHoldings) = false, want true")
	}
	if !requiresAuth(quickspy.ActiveOrdersFocusType) {
		t.Fatalf("requiresAuth(ActiveOrders) = false, want true")
	}
	if !requiresAuth(quickspy.OrderPlacementFocusType) {
		t.Fatalf("requiresAuth(OrderPlacement) = false, want true")
	}
	if requiresAuth(quickspy.TickerFocusType) {
		t.Fatalf("requiresAuth(Ticker) = true, want false")
	}
}

func TestEmitWritesNDJSON(t *testing.T) {
	var buf bytes.Buffer
	enc = json.NewEncoder(&buf)
	now := time.Now().UTC()
	ev := eventEnvelope{Timestamp: now, Focus: "ticker", Data: map[string]any{"x": 1}}
	emit(ev)
	if buf.Len() == 0 {
		t.Fatalf("emit() wrote nothing")
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["focus"].(string) != "ticker" {
		t.Fatalf("unexpected focus: %v", m["focus"])
	}
}

// captureStdout redirects os.Stdout for the duration of f and returns captured output.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()
	f()
	_ = w.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}

func TestClearScreen(t *testing.T) {
	got := captureStdout(t, func() { clearScreen() })
	if got != ansiClear {
		t.Fatalf("clearScreen wrote %q, want %q", got, ansiClear)
	}
}

func TestIntMin(t *testing.T) {
	if intMin(1, 2) != 1 || intMin(2, 1) != 1 || intMin(-1, -2) != -2 {
		t.Fatalf("intMin unexpected result")
	}
}

func TestRenderOrderbook(t *testing.T) {
	b := &orderbook.Book{Bids: orderbook.Levels{orderbook.Level{Price: 10, Amount: 1}}, Asks: orderbook.Levels{orderbook.Level{Price: 11, Amount: 2}}}
	out := captureStdout(t, func() { renderOrderbook(b, 1) })
	if out == "" {
		t.Fatalf("renderOrderbook wrote nothing")
	}
}

func TestRenderTicker(t *testing.T) {
	tk := &ticker.Price{Last: 100, Bid: 99, Ask: 101, Volume: 1234, MarkPrice: 100.5, IndexPrice: 100.1}
	out := captureStdout(t, func() { renderTicker(tk) })
	if !bytes.Contains([]byte(out), []byte("Last:")) {
		t.Fatalf("renderTicker missing content: %q", out)
	}
}

func TestRenderTrades(t *testing.T) {
	tr := []trade.Data{{Price: 10, Amount: 2, Side: order.Buy, Timestamp: time.Now()}, {Price: 12, Amount: 3, Side: order.Sell, Timestamp: time.Now().Add(time.Second)}}
	out := captureStdout(t, func() { renderTrades(tr) })
	if !bytes.Contains([]byte(out), []byte("VWAP")) {
		t.Fatalf("renderTrades missing content: %q", out)
	}
	out = captureStdout(t, func() { renderTrades(nil) })
	if !bytes.Contains([]byte(out), []byte("No trades.")) {
		t.Fatalf("renderTrades empty case not handled: %q", out)
	}
}

func TestRenderKlines(t *testing.T) {
	k1 := websocket.KlineData{Timestamp: time.Now(), OpenPrice: 10, ClosePrice: 11, HighPrice: 12, LowPrice: 9, Volume: 5, Interval: "1m"}
	k2 := websocket.KlineData{Timestamp: time.Now().Add(time.Minute), OpenPrice: 11, ClosePrice: 10.5, HighPrice: 13, LowPrice: 8, Volume: 7, Interval: "1m"}
	out := captureStdout(t, func() { renderKlines([]websocket.KlineData{k1, k2}) })
	if !bytes.Contains([]byte(out), []byte("Klines:")) {
		t.Fatalf("renderKlines missing header: %q", out)
	}
	out = captureStdout(t, func() { renderKlines(nil) })
	if !bytes.Contains([]byte(out), []byte("No klines.")) {
		t.Fatalf("renderKlines empty case not handled: %q", out)
	}
}

func TestRenderAccountHoldings(t *testing.T) {
	h := &account.Holdings{Accounts: []account.SubAccount{{ID: "acct1", AssetType: asset.Spot, Currencies: []account.Balance{{Currency: currency.BTC, Total: 1.23, Free: 1.0, Hold: 0.23}}}}}
	out := captureStdout(t, func() { renderAccountHoldings(h) })
	if !bytes.Contains([]byte(out), []byte("acct1")) {
		t.Fatalf("renderAccountHoldings missing content: %q", out)
	}
	out = captureStdout(t, func() { renderAccountHoldings(&account.Holdings{}) })
	if !bytes.Contains([]byte(out), []byte("No holdings.")) {
		t.Fatalf("renderAccountHoldings empty case not handled: %q", out)
	}
}

func TestRenderActiveOrders(t *testing.T) {
	ord := []order.Detail{{OrderID: "1", Type: order.Limit, Side: order.Buy, Price: 10, Amount: 2, ExecutedAmount: 1, Status: order.Active, LastUpdated: time.Now()}}
	out := captureStdout(t, func() { renderActiveOrders(ord) })
	if !bytes.Contains([]byte(out), []byte("OrderID")) {
		t.Fatalf("renderActiveOrders missing header: %q", out)
	}
	out = captureStdout(t, func() { renderActiveOrders(nil) })
	if !bytes.Contains([]byte(out), []byte("No active orders.")) {
		t.Fatalf("renderActiveOrders empty case not handled: %q", out)
	}
}

func TestRenderExecutionLimits(t *testing.T) {
	l := &limits.MinMaxLevel{MinPrice: 1, MaxPrice: 2, MinimumBaseAmount: 0.1, MaximumBaseAmount: 1, MinimumQuoteAmount: 10, MaximumQuoteAmount: 100, MinNotional: 5}
	out := captureStdout(t, func() { renderExecutionLimits(l) })
	if !bytes.Contains([]byte(out), []byte("Execution Limits")) {
		t.Fatalf("renderExecutionLimits missing content: %q", out)
	}
	// nil should print nothing
	out = captureStdout(t, func() { renderExecutionLimits(nil) })
	if out != "" {
		t.Fatalf("renderExecutionLimits(nil) should be empty, got: %q", out)
	}
}

func TestRenderURL(t *testing.T) {
	out := captureStdout(t, func() { renderURL("https://x.y") })
	if !bytes.Contains([]byte(out), []byte("Trade URL:")) {
		t.Fatalf("renderURL missing label: %q", out)
	}
}

func TestRenderContract(t *testing.T) {
	c := &futures.Contract{Underlying: currency.NewPair(currency.BTC, currency.USD), Name: currency.NewPair(currency.BTC, currency.USD), Type: futures.Perpetual, Multiplier: 0.001, StartDate: time.Now().Add(-time.Hour), EndDate: time.Now().Add(time.Hour), SettlementCurrencies: []currency.Code{currency.USD}}
	out := captureStdout(t, func() { renderContract(c) })
	if !bytes.Contains([]byte(out), []byte("Contract")) {
		t.Fatalf("renderContract missing header: %q", out)
	}
	out = captureStdout(t, func() { renderContract(nil) })
	if out != "" {
		t.Fatalf("renderContract(nil) should be empty, got: %q", out)
	}
}

func TestRenderPrettyPayloadAndBasics(t *testing.T) {
	b := &orderbook.Book{Bids: orderbook.Levels{orderbook.Level{Price: 1, Amount: 1}}, Asks: orderbook.Levels{orderbook.Level{Price: 2, Amount: 1}}}
	pr := &ticker.Price{Last: 1}
	kl := []websocket.KlineData{{Timestamp: time.Now(), OpenPrice: 1, ClosePrice: 2, HighPrice: 2, LowPrice: 1, Volume: 1, Interval: "1m"}}
	trs := []trade.Data{{Price: 1, Amount: 1, Timestamp: time.Now()}}
	h := &account.Holdings{Accounts: []account.SubAccount{{ID: "a", AssetType: asset.Spot}}}
	od := []order.Detail{{OrderID: "x"}}
	fr := &fundingrate.LatestRateResponse{}
	ct := &futures.Contract{Name: currency.NewPair(currency.BTC, currency.USD)}
	ex := &limits.MinMaxLevel{MinPrice: 1}

	cases := []any{b, []ticker.Price{{}}, pr, kl, trs, trade.Data{Price: 1, Amount: 1}, h, od, 1.0, fr, ct, ex, "u", 12345}
	out := captureStdout(t, func() {
		for _, c := range cases {
			renderPrettyPayload(c, 3)
		}
	})
	if out == "" {
		t.Fatalf("renderPrettyPayload wrote nothing")
	}
}

func TestRenderOpenInterestAndFundingRate(t *testing.T) {
	out := captureStdout(t, func() { renderOpenInterest(12.34) })
	if !bytes.Contains([]byte(out), []byte("Open Interest:")) {
		t.Fatalf("renderOpenInterest missing label: %q", out)
	}
	fr := &fundingrate.LatestRateResponse{}
	out = captureStdout(t, func() { renderFundingRate(fr) })
	if !bytes.Contains([]byte(out), []byte("Funding:")) {
		t.Fatalf("renderFundingRate missing label: %q", out)
	}
	out = captureStdout(t, func() { renderFundingRate(nil) })
	if !bytes.Contains([]byte(out), []byte("No funding rate.")) {
		t.Fatalf("renderFundingRate nil case not handled: %q", out)
	}
}

func TestInitVerboseLogger(t *testing.T) {
	if err := initVerboseLogger(); err != nil {
		t.Fatalf("initVerboseLogger error: %v", err)
	}
}

func TestBuildQuickSpyRESTOnly(t *testing.T) {
	cfg := &appConfig{
		Exchange:     "binance",
		Asset:        asset.Spot,
		Pair:         currency.NewPair(currency.BTC, currency.USDT),
		FocusType:    quickspy.TickerFocusType,
		UseWebsocket: false,
		PollInterval: time.Second,
	}
	qs, err := buildQuickSpy(cfg)
	if err != nil {
		t.Fatalf("buildQuickSpy error: %v", err)
	}
	if qs == nil || qs.Key == nil {
		t.Fatalf("buildQuickSpy returned nil QuickSpy or Key")
	}
}

func TestParseFlagsHappyPath(t *testing.T) {
	oldFS := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	defer func() { flag.CommandLine = oldFS }()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{
		"quickspy",
		"--exchange", "binance",
		"--asset", "spot",
		"--currencyPair", "BTC-USDT",
		"--focusType", "ticker",
		"--poll", "1s",
		"--websocket", "false",
		"--book-levels", "10",
		"--wsDataTimeout", "1s",
	}
	cfg := parseFlags()
	if cfg.Exchange != "binance" || cfg.Asset != asset.Spot || cfg.Pair.String() != "BTC-USDT" || cfg.FocusType != quickspy.TickerFocusType {
		t.Fatalf("parseFlags unexpected cfg: %+v", cfg)
	}
}

func makeDummyQuickSpy(tb testing.TB) *quickspy.QuickSpy {
	f := quickspy.NewFocusData(quickspy.TickerFocusType, false, true, time.Second)
	f.Init()
	q, err := quickspy.NewQuickSpy(
		&quickspy.CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair("Binance", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))},
		[]quickspy.FocusData{*f},
		false)
	require.NoError(tb, err)
	q.Data = &quickspy.Data{Key: &quickspy.CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair("Binance", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))}}
	return q
}

func TestStreamWS_JSONOnlyMultiplePayloads(t *testing.T) {
	qs := makeDummyQuickSpy(t)
	f, _ := qs.GetFocusByKey(quickspy.TickerFocusType)

	var buf bytes.Buffer
	enc = json.NewEncoder(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Run streamWS
	go streamWS(ctx, qs, quickspy.TickerFocusType, f, 5, true)

	// Send a variety of payloads
	f.Stream <- errors.New("oops")
	f.Stream <- &orderbook.Book{}
	f.Stream <- []ticker.Price{{Last: 1}}
	f.Stream <- &ticker.Price{Last: 2}
	f.Stream <- trade.Data{Price: 1, Amount: 1, Timestamp: time.Now()}
	f.Stream <- []trade.Data{{Price: 2, Amount: 2}}
	f.Stream <- []websocket.KlineData{{Timestamp: time.Now()}}
	f.Stream <- &fundingrate.LatestRateResponse{}
	f.Stream <- 1.23
	f.Stream <- []order.Detail{{OrderID: "1"}}
	f.Stream <- &account.Holdings{}
	f.Stream <- &limits.MinMaxLevel{}
	f.Stream <- "http://x"
	f.Stream <- &futures.Contract{Name: currency.NewPair(currency.BTC, currency.USD)}
	f.Stream <- struct{ Unknown int }{Unknown: 1}

	// Allow processing then cancel
	time.Sleep(50 * time.Millisecond)
	cancel()
	// Give goroutine a moment to exit
	time.Sleep(20 * time.Millisecond)

	if buf.Len() == 0 {
		t.Fatalf("streamWS didn't emit NDJSON")
	}
}

func TestStreamREST_JSONOnlyURLPayload(t *testing.T) {
	qs := makeDummyQuickSpy(t)
	qs.Data.URL = "http://example"

	var buf bytes.Buffer
	enc = json.NewEncoder(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go streamREST(ctx, qs, quickspy.URLFocusType, 10*time.Millisecond, true)
	// First tick happens immediately
	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)
	if buf.Len() == 0 {
		t.Fatalf("streamREST didn't emit NDJSON")
	}
}
