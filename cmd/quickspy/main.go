package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/quickspy"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var enc = json.NewEncoder(os.Stdout)

// eventEnvelope is a small wrapper to add metadata around exported data.
// It is printed as NDJSON to stdout for easy streaming/consumption.
type eventEnvelope struct {
	Timestamp time.Time `json:"ts"`
	Focus     string    `json:"focus"`
	Data      any       `json:"data,omitempty"`
	Error     string    `json:"error,omitempty"`
}

func main() {
	// Flags
	exch := flag.String("exchange", "", "Exchange name, e.g. binance, okx")
	assetStr := flag.String("asset", "spot", "Asset type, e.g. spot, futures, perpetualswap")
	pairStr := flag.String("currencyPair", "BTC-USDT", "Currency pair, e.g. BTC-USDT or ETHUSD")
	focusStr := flag.String("focusType", "ticker", "Focus type: ticker, orderbook, kline, trades, openinterest, fundingrate, accountholdings, activeorders, orderexecution, url, contract")
	pollStr := flag.String("poll", "5s", "Poll interval for REST focus and timeout for websocket initial data wait")
	websocketInterval := flag.String("websocketInterval", "100ms", "Interval for websocket subscriptions")
	wsFlag := flag.Bool("websocket", true, "Use websocket when supported (ticker/orderbook). Set false to force REST.")
	bookLevels := flag.Int("book-levels", 15, "Number of levels to render per side for orderbook focus")
	websocketDataTimeout := flag.String("wsDataTimeout", "30s", "Websocket data timeout duration (e.g. 30s, 1m)")
	verbose := flag.Bool("verbose", false, "Verbose logging to stderr")
	// credentials until credential manager integrated
	apiKey := flag.String("apiKey", "", "API key (only for auth-required focuses)")
	apiSecret := flag.String("apiSecret", "", "API secret (only for auth-required focuses)")
	subAccount := flag.String("subAccount", "", "Sub-account (optional)")
	clientID := flag.String("clientID", "", "Client ID (optional)")
	otp := flag.String("otp", "", "One-time password (optional)")
	pemKey := flag.String("pemKey", "", "PEM key (optional)")

	flag.Parse()
	if *verbose {
		defaultLogSettings := log.GenDefaultSettings()
		defaultLogSettings.AdvancedSettings.ShowLogSystemName = convert.BoolPtr(true)
		defaultLogSettings.AdvancedSettings.Headers.Info = common.CMDColours.Info + "[INFO]" + common.CMDColours.Default
		defaultLogSettings.AdvancedSettings.Headers.Warn = common.CMDColours.Warn + "[WARN]" + common.CMDColours.Default
		defaultLogSettings.AdvancedSettings.Headers.Debug = common.CMDColours.Debug + "[DEBUG]" + common.CMDColours.Default
		defaultLogSettings.AdvancedSettings.Headers.Error = common.CMDColours.Error + "[ERROR]" + common.CMDColours.Default
		err := log.SetGlobalLogConfig(defaultLogSettings)
		if err != nil {
			fmt.Printf("failed to setup logger: %v\n", err)
			os.Exit(1)
		}
		log.Infoln(log.Global, "Verbose logger initialised.")
	}

	if strings.TrimSpace(*exch) == "" || strings.TrimSpace(*assetStr) == "" || strings.TrimSpace(*pairStr) == "" || strings.TrimSpace(*focusStr) == "" {
		_, _ = fmt.Fprintln(os.Stderr, "missing required flags: --exchange, --asset, --currencyPair, --focusType")
		flag.Usage()
		os.Exit(2)
	}

	// Parse asset
	ast, err := asset.New(*assetStr)
	if err != nil {
		fatalErr(fmt.Errorf("invalid asset: %w", err))
	}

	// Parse pair
	cp, err := currency.NewPairFromString(*pairStr)
	if err != nil {
		fatalErr(fmt.Errorf("invalid currencyPair: %w", err))
	}

	// Parse focus type and defaults
	fType, defaultWS := parseFocusType(*focusStr)
	if fType == quickspy.UnsetFocusType {
		fatalErr(fmt.Errorf("unsupported focusType: %s", *focusStr))
	}
	useWS := defaultWS && *wsFlag

	pollDur, err := time.ParseDuration(*pollStr)
	if err != nil {
		fatalErr(fmt.Errorf("invalid poll duration: %w", err))
	}

	wsInterval, err := time.ParseDuration(*websocketInterval)
	if err != nil {
		fatalErr(fmt.Errorf("invalid websocket interval duration: %w", err))
	}

	wsDataTimeoutDur, err := time.ParseDuration(*websocketDataTimeout)
	if err != nil {
		fatalErr(fmt.Errorf("invalid websocket data timeout duration: %w", err))
	}

	// Credentials (only applied when supplied or required by focus)
	var creds *account.Credentials
	if requiresAuth(fType) {
		creds = &account.Credentials{
			Key:             strings.TrimSpace(*apiKey),
			Secret:          strings.TrimSpace(*apiSecret),
			SubAccount:      strings.TrimSpace(*subAccount),
			ClientID:        strings.TrimSpace(*clientID),
			OneTimePassword: strings.TrimSpace(*otp),
			PEMKey:          strings.TrimSpace(*pemKey),
		}
		if creds.IsEmpty() {
			fatalErr(fmt.Errorf("focus %s requires credentials; provide --apiKey and --apiSecret (and others as needed)", fType.String()))
		}
	}

	// Build key
	k := &quickspy.CredentialsKey{
		Credentials:       creds,
		ExchangeAssetPair: key.NewExchangeAssetPair(strings.ToLower(*exch), ast, cp),
	}

	// Focus config
	focus := quickspy.NewFocusData(fType, false, useWS, pollDur, wsInterval)
	focus.Init()
	// Create quickspy and start; fallback to REST if websocket unsupported
	qs, err := quickspy.NewQuickSpy(k, []quickspy.FocusData{*focus}, *verbose)
	if err != nil && useWS && strings.Contains(strings.ToLower(err.Error()), "has no websocket") {
		// retry with REST
		focus = quickspy.NewFocusData(fType, false, false, fallbackPoll(pollDur), wsInterval)
		focus.Init()
		qs, err = quickspy.NewQuickSpy(k, []quickspy.FocusData{*focus}, *verbose)
	}
	if err != nil {
		fatalErr(err)
	}
	// Avoid nil deref in ExportedData when user switches to snapshot mode later
	if qs.Data != nil && qs.Data.ExecutionLimits == nil {
		qs.Data.ExecutionLimits = &limits.MinMaxLevel{}
	}
	if err := qs.Run(); err != nil {
		fatalErr(err)
	}

	// Context & OS signals for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Wait for initial data or timeout based on RESTPollTime
	if err := qs.WaitForInitialDataWithTimer(ctx, fType, wsDataTimeoutDur); err != nil {
		// Emit error and exit
		emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: fType.String(), Error: err.Error()})
		os.Exit(1)
	}

	// Initial emit
	if focus.RequiresWebsocket() {
		// For WS, we'll emit on stream events below
	} else {
		if payload, err := qs.CurrentPayload(fType); err == nil {
			emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: fType.String(), Data: payload})
		} else {
			emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: fType.String(), Error: err.Error()})
		}
	}

	// Streaming
	foc, _ := qs.GetFocusByKey(fType)
	if foc != nil && foc.RequiresWebsocket() {
		streamWS(ctx, qs, fType, foc, *bookLevels)
		return
	}

	// REST streaming via ticker
	interval := pollDur
	if interval <= 0 {
		interval = time.Second
	}
	streamREST(ctx, qs, fType, interval)
}

func fallbackPoll(d time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return 5 * time.Second
}

func streamWS(ctx context.Context, qs *quickspy.QuickSpy, ft quickspy.FocusType, f *quickspy.FocusData, bookLevels int) {
	for {
		select {
		case <-ctx.Done():
			qs.Shutdown()
			return
		case d := <-f.Stream:
			clearScreen()
			heading := fmt.Sprintf("%s | %s | %s", qs.Key.ExchangeAssetPair.Exchange, qs.Key.ExchangeAssetPair.Asset.String(), qs.Key.ExchangeAssetPair.Pair().String())
			fmt.Fprintf(os.Stdout, "%s%s%s\n", ansiBold, heading, ansiReset)

			switch v := d.(type) {
			case error:
				fmt.Fprintf(os.Stdout, "Error: %v\n", v)
			case *orderbook.Book:
				renderOrderbook(v, bookLevels)
			case []ticker.Price:
				if len(v) > 0 {
					emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
				}
			case *ticker.Price:
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			default:
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			}
		}
	}
}

func streamREST(ctx context.Context, qs *quickspy.QuickSpy, ft quickspy.FocusType, d time.Duration) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			qs.Shutdown()
			return
		case <-t.C:
			payload, err := qs.CurrentPayload(ft)
			if err != nil {
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Error: err.Error()})
				continue
			}
			emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: payload})
		}
	}
}

func requiresAuth(f quickspy.FocusType) bool {
	return f == quickspy.AccountHoldingsFocusType || f == quickspy.ActiveOrdersFocusType || f == quickspy.OrderPlacementFocusType
}

// emit writes NDJSON events to stdout.
func emit(ev eventEnvelope) {
	_ = enc.Encode(ev)
}

func parseFocusType(s string) (quickspy.FocusType, bool) {
	// returns (focusType, useWebsocketDefault)
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "ticker":
		return quickspy.TickerFocusType, true
	case "orderbook", "order_book", "ob", "book":
		return quickspy.OrderBookFocusType, true
	case "kline", "candles", "candle", "ohlc":
		// websocket subscriptions for kline are not wired via quickspy yet
		return quickspy.KlineFocusType, false
	case "trades", "trade":
		// websocket support exists in quickspy handler, but subscription mapping is not wired for trades
		return quickspy.TradesFocusType, false
	case "openinterest", "oi":
		return quickspy.OpenInterestFocusType, false
	case "fundingrate", "funding":
		return quickspy.FundingRateFocusType, false
	case "accountholdings", "account", "holdings", "balances":
		return quickspy.AccountHoldingsFocusType, false
	case "activeorders", "orders":
		return quickspy.ActiveOrdersFocusType, false
	case "orderexecution", "executionlimits", "limits":
		return quickspy.OrderExecutionFocusType, false
	case "url", "tradeurl", "trade_url":
		return quickspy.URLFocusType, false
	case "contract":
		return quickspy.ContractFocusType, false
	default:
		return quickspy.UnsetFocusType, false
	}
}

func fatalErr(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// -------- Rendering helpers --------

const (
	ansiClear = "\033[2J\033[H"
	ansiReset = "\033[0m"
	ansiGreen = "\033[32m"
	ansiRed   = "\033[31m"
	ansiDim   = "\033[2m"
	ansiBold  = "\033[1m"
)

func clearScreen() {
	fmt.Fprint(os.Stdout, ansiClear)
}

func renderOrderbook(b *orderbook.Book, levels int) {
	if levels <= 0 {
		levels = 15
	}
	fmt.Fprintf(os.Stdout, "%s%-14s %-14s %-14s%s\n", ansiDim, "Price", "Amount", "Total", ansiReset)
	// Asks (red), display from best (lowest) up to levels
	askCount := intMin(levels, len(b.Asks))
	cumulativeAsks := 0.0
	for i := 0; i < askCount; i++ {
		lvl := b.Asks[i]
		cumulativeAsks += lvl.Amount
		fmt.Fprintf(os.Stdout, "%s% -14.8f % -14.8f % -14.8f%s\n", ansiRed, lvl.Price, lvl.Amount, cumulativeAsks, ansiReset)
	}
	fmt.Fprintln(os.Stdout)
	// Bids (green), from best (highest) up to levels
	bidCount := intMin(levels, len(b.Bids))
	cumulativeBids := 0.0
	for i := 0; i < bidCount; i++ {
		lvl := b.Bids[i]
		cumulativeBids += lvl.Amount
		fmt.Fprintf(os.Stdout, "%s% -14.8f % -14.8f % -14.8f%s\n", ansiGreen, lvl.Price, lvl.Amount, cumulativeBids, ansiReset)
	}
}

// intMin for dealing with ints
func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
