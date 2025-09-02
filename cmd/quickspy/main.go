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

	btcommon "github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
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
	"github.com/thrasher-corp/gocryptotrader/log"
)

var enc = json.NewEncoder(os.Stdout)

// errless stdout helpers to avoid handling write errors everywhere.
func outPrintf(format string, args ...any) { _, _ = fmt.Fprintf(os.Stdout, format, args...) }
func outPrintln(args ...any)               { _, _ = fmt.Fprintln(os.Stdout, args...) }
func outPrint(args ...any)                 { _, _ = fmt.Fprint(os.Stdout, args...) }

// eventEnvelope is a small wrapper to add metadata around exported data.
// It is printed as NDJSON to stdout for easy streaming/consumption.
type eventEnvelope struct {
	Timestamp time.Time `json:"ts"`
	Focus     string    `json:"focus"`
	Data      any       `json:"data,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// appConfig holds parsed CLI configuration in a structured way.
type appConfig struct {
	Exchange      string
	Asset         asset.Item
	Pair          currency.Pair
	FocusType     quickspy.FocusType
	UseWebsocket  bool
	PollInterval  time.Duration
	WSDataTimeout time.Duration
	Verbose       bool
	BookLevels    int
	Credentials   *account.Credentials
	JSONOnly      bool
}

func main() {
	cfg := parseFlags()
	if cfg.Verbose {
		if err := initVerboseLogger(); err != nil {
			fatalErr(fmt.Errorf("failed to setup logger: %w", err))
		}
		log.Infoln(log.Global, "Verbose logger initialised.")
	}

	// Build QuickSpy with WS fallback if needed
	qs, err := buildQuickSpy(cfg)
	if err != nil {
		fatalErr(err)
	}

	// Safety: ensure non-nil limits for rendering
	if qs.Data != nil && qs.Data.ExecutionLimits == nil {
		qs.Data.ExecutionLimits = &limits.MinMaxLevel{}
	}

	if err := qs.Run(); err != nil {
		fatalErr(err)
	}

	// Context & OS signals for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Wait for initial data
	if err := qs.WaitForInitialDataWithTimer(ctx, cfg.FocusType, cfg.WSDataTimeout); err != nil {
		emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: cfg.FocusType.String(), Error: err.Error()})
		os.Exit(1)
	}

	defer outPrintln("\nGoodbye! 🌞")
	// Dispatch streaming mode
	f, _ := qs.GetFocusByKey(cfg.FocusType)
	if f != nil && f.RequiresWebsocket() {
		streamWS(ctx, qs, cfg.FocusType, f, cfg.BookLevels, cfg.JSONOnly)
		return
	}
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = time.Second
	}
	streamREST(ctx, qs, cfg.FocusType, interval, cfg.JSONOnly)
}

// parseFlags parses CLI flags, validates them, applies sensible defaults, and returns appConfig.
func parseFlags() *appConfig {
	exch := flag.String("exchange", "", "Exchange name, e.g. binance, okx")
	assetStr := flag.String("asset", "spot", "Asset type, e.g. spot, futures, perpetualswap")
	pairStr := flag.String("currencyPair", "BTC-USDT", "Currency pair, e.g. BTC-USDT or ETHUSD")
	focusStr := flag.String("focusType", "ticker", "Focus type: ticker, orderbook, kline, trades, openinterest, fundingrate, accountholdings, activeorders, orderexecution, url, contract")
	pollStr := flag.String("poll", "5s", "Poll interval for REST focus and timeout for websocket initial data wait")
	wsFlag := flag.Bool("websocket", true, "Use websocket when supported (ticker/orderbook). Set false to force REST.")
	bookLevels := flag.Int("book-levels", 15, "Number of levels to render per side for orderbook focus")
	websocketDataTimeout := flag.String("wsDataTimeout", "30s", "Websocket data timeout duration (e.g. 30s, 1m)")
	verbose := flag.Bool("verbose", false, "Verbose logging to stderr")
	jsonOnly := flag.Bool("json", false, "Emit NDJSON only (no ANSI rendering/headers)")

	// credentials until credential manager integrated
	apiKey := flag.String("apiKey", "", "API key (only for auth-required focuses)")
	apiSecret := flag.String("apiSecret", "", "API secret (only for auth-required focuses)")
	subAccount := flag.String("subAccount", "", "Sub-account (optional)")
	clientID := flag.String("clientID", "", "Client ID (optional)")
	otp := flag.String("otp", "", "One-time password (optional)")
	pemKey := flag.String("pemKey", "", "PEM key (optional)")

	flag.Parse()

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

	return &appConfig{
		Exchange:      strings.ToLower(*exch),
		Asset:         ast,
		Pair:          cp,
		FocusType:     fType,
		UseWebsocket:  useWS,
		PollInterval:  pollDur,
		WSDataTimeout: wsDataTimeoutDur,
		Verbose:       *verbose,
		BookLevels:    *bookLevels,
		Credentials:   creds,
		JSONOnly:      *jsonOnly,
	}
}

func initVerboseLogger() error {
	defaultLogSettings := log.GenDefaultSettings()
	defaultLogSettings.AdvancedSettings.ShowLogSystemName = convert.BoolPtr(true)
	defaultLogSettings.AdvancedSettings.Headers.Info = btcommon.CMDColours.Info + "[INFO]" + btcommon.CMDColours.Default
	defaultLogSettings.AdvancedSettings.Headers.Warn = btcommon.CMDColours.Warn + "[WARN]" + btcommon.CMDColours.Default
	defaultLogSettings.AdvancedSettings.Headers.Debug = btcommon.CMDColours.Debug + "[DEBUG]" + btcommon.CMDColours.Default
	defaultLogSettings.AdvancedSettings.Headers.Error = btcommon.CMDColours.Error + "[ERROR]" + btcommon.CMDColours.Default
	return log.SetGlobalLogConfig(defaultLogSettings)
}

func buildQuickSpy(cfg *appConfig) (*quickspy.QuickSpy, error) {
	// Build key
	k := &quickspy.CredentialsKey{
		Credentials:       cfg.Credentials,
		ExchangeAssetPair: key.NewExchangeAssetPair(cfg.Exchange, cfg.Asset, cfg.Pair),
	}

	// Focus config
	focus := quickspy.NewFocusData(cfg.FocusType, false, cfg.UseWebsocket, cfg.PollInterval)
	focus.Init()

	// Create quickspy and start; fallback to REST if websocket unsupported
	qs, err := quickspy.NewQuickSpy(k, []quickspy.FocusData{*focus}, cfg.Verbose)
	if err != nil && cfg.UseWebsocket && strings.Contains(strings.ToLower(err.Error()), "has no websocket") {
		// retry with REST
		focus = quickspy.NewFocusData(cfg.FocusType, false, false, fallbackPoll(cfg.PollInterval))
		focus.Init()
		qs, err = quickspy.NewQuickSpy(k, []quickspy.FocusData{*focus}, cfg.Verbose)
	}
	return qs, err
}

func fallbackPoll(d time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return 5 * time.Second
}

func streamWS(ctx context.Context, qs *quickspy.QuickSpy, ft quickspy.FocusType, f *quickspy.FocusData, bookLevels int, jsonOnly bool) {
	for {
		select {
		case <-ctx.Done():
			qs.Shutdown()
			return
		case d := <-f.Stream:
			if !jsonOnly {
				clearScreen()
				heading := fmt.Sprintf("%s | %s | %s | Websocket", qs.Key.ExchangeAssetPair.Exchange, qs.Key.ExchangeAssetPair.Asset.String(), qs.Key.ExchangeAssetPair.Pair().String())
				outPrintf("%s%s%s\n", ansiBold, heading, ansiReset)
			}

			switch v := d.(type) {
			case error:
				if !jsonOnly {
					outPrintf("Error: %v\n", v)
				}
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Error: v.Error()})
			case *orderbook.Book:
				if !jsonOnly {
					renderOrderbook(v, bookLevels)
				}
				if jsonOnly {
					emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
				}
			case []ticker.Price:
				if len(v) > 0 {
					if !jsonOnly {
						renderTicker(&v[0])
					}
					if jsonOnly {
						emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
					}
				}
			case *ticker.Price:
				if !jsonOnly {
					renderTicker(v)
				}
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			case trade.Data:
				if !jsonOnly {
					renderTrades([]trade.Data{v})
				}
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			case []trade.Data:
				if !jsonOnly {
					renderTrades(v)
				}
				if jsonOnly {
					emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
				}
			case []websocket.KlineData:
				if !jsonOnly {
					renderKlines(v)
				}
				if jsonOnly {
					emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
				}
			case *fundingrate.LatestRateResponse:
				if !jsonOnly {
					renderFundingRate(v)
				}
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			case float64:
				if !jsonOnly {
					renderOpenInterest(v)
				}
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			case []order.Detail:
				if !jsonOnly {
					renderActiveOrders(v)
				}
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			case *account.Holdings:
				if !jsonOnly {
					renderAccountHoldings(v)
				}
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			case *limits.MinMaxLevel:
				if !jsonOnly {
					renderExecutionLimits(v)
				}
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			case string:
				if !jsonOnly {
					renderURL(v)
				}
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			case *futures.Contract:
				if !jsonOnly {
					renderContract(v)
				}
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			default:
				// Unknown payload, just dump JSON
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: v})
			}
		}
	}
}

func streamREST(ctx context.Context, qs *quickspy.QuickSpy, ft quickspy.FocusType, d time.Duration, jsonOnly bool) {
	// we have already waited for initial data before calling this function
	t := time.NewTimer(0)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			qs.Shutdown()
			return
		case <-t.C:
			payload, err := qs.LatestData(ft)
			if err != nil {
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Error: err.Error()})
				t.Reset(d)
				continue
			}
			if !jsonOnly {
				clearScreen()
				heading := fmt.Sprintf("%s | %s | %s | REST", qs.Key.ExchangeAssetPair.Exchange, qs.Key.ExchangeAssetPair.Asset.String(), qs.Key.ExchangeAssetPair.Pair().String())
				outPrintf("%s%s%s\n", ansiBold, heading, ansiReset)
				renderPrettyPayload(payload, 15)
			}
			// Emit NDJSON
			if jsonOnly || (ft != quickspy.TradesFocusType && ft != quickspy.KlineFocusType && ft != quickspy.OrderBookFocusType) {
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: ft.String(), Data: payload})
			}
			t.Reset(d)
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
	outPrint(ansiClear)
}

func renderOrderbook(b *orderbook.Book, levels int) {
	if levels <= 0 {
		levels = 15
	}
	outPrintf("%s%-14s %-14s %-14s%s\n", ansiDim, "Price", "Amount", "Total", ansiReset)
	// Asks (red), display from best (lowest) up to levels
	askCount := intMin(levels, len(b.Asks))
	cumulativeAsks := 0.0
	for i := 0; i < askCount; i++ {
		lvl := b.Asks[i]
		cumulativeAsks += lvl.Amount
		outPrintf("%s% -14.8f % -14.8f % -14.8f%s\n", ansiRed, lvl.Price, lvl.Amount, cumulativeAsks, ansiReset)
	}
	outPrintln()
	// Bids (green), from best (highest) up to levels
	bidCount := intMin(levels, len(b.Bids))
	cumulativeBids := 0.0
	for i := 0; i < bidCount; i++ {
		lvl := b.Bids[i]
		cumulativeBids += lvl.Amount
		outPrintf("%s% -14.8f % -14.8f % -14.8f%s\n", ansiGreen, lvl.Price, lvl.Amount, cumulativeBids, ansiReset)
	}
}

func renderTicker(t *ticker.Price) {
	if t == nil {
		return
	}
	spread := t.Ask - t.Bid
	var spreadPct float64
	if t.Last != 0 {
		spreadPct = (spread / t.Last) * 100
	}
	outPrintf("%sLast:%s %.8f  %sBid:%s %.8f  %sAsk:%s %.8f\n",
		ansiBold, ansiReset, t.Last, ansiGreen, ansiReset, t.Bid, ansiRed, ansiReset, t.Ask)
	outPrintf("%sVol:%s  %.4f  %sSpread:%s %.8f (%.4f%%)  %sMark:%s %.8f  %sIndex:%s %.8f\n",
		ansiDim, ansiReset, t.Volume, ansiDim, ansiReset, spread, spreadPct, ansiDim, ansiReset, t.MarkPrice, ansiDim, ansiReset, t.IndexPrice)
}

func renderTrades(trs []trade.Data) {
	if len(trs) == 0 {
		outPrintln("No trades.")
		return
	}
	// Summaries
	n := len(trs)
	start := trs[0].Timestamp
	end := trs[n-1].Timestamp
	if end.Before(start) {
		start, end = end, start
	}
	var baseVol, quoteVol float64
	var buys, sells int
	for i := range trs {
		baseVol += trs[i].Amount
		quoteVol += trs[i].Amount * trs[i].Price
		if trs[i].Side&order.Buy == order.Buy || trs[i].Side&order.Bid == order.Bid || trs[i].Side&order.Long == order.Long {
			buys++
		} else if trs[i].Side&order.Sell == order.Sell || trs[i].Side&order.Ask == order.Ask || trs[i].Side&order.Short == order.Short {
			sells++
		}
	}
	vwap := 0.0
	if baseVol != 0 {
		vwap = quoteVol / baseVol
	}
	last := trs[n-1]
	span := end.Sub(start)
	outPrintf("%sTrades:%s N=%d Span=%s\n", ansiBold, ansiReset, n, span.Truncate(time.Second))
	outPrintf("Range: %s -> %s\n", start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	outPrintf("VWAP: %.8f  Last: %.8f @ %s\n", vwap, last.Price, last.Timestamp.UTC().Format(time.RFC3339))
	outPrintf("Volume: base=%.8f quote=%.8f  Buys/Sells: %d/%d\n", baseVol, quoteVol, buys, sells)
}

func renderKlines(kl []websocket.KlineData) {
	if len(kl) == 0 {
		outPrintln("No klines.")
		return
	}
	n := len(kl)
	start := kl[0].Timestamp
	end := kl[n-1].Timestamp
	if end.Before(start) {
		start, end = end, start
	}
	firstOpen := kl[0].OpenPrice
	lastClose := kl[n-1].ClosePrice
	change := lastClose - firstOpen
	changePct := 0.0
	if firstOpen != 0 {
		changePct = (change / firstOpen) * 100
	}
	high := kl[0].HighPrice
	low := kl[0].LowPrice
	var totalVol float64
	for i := range kl {
		if kl[i].HighPrice > high {
			high = kl[i].HighPrice
		}
		if kl[i].LowPrice < low {
			low = kl[i].LowPrice
		}
		totalVol += kl[i].Volume
	}
	avgVol := totalVol / float64(n)
	span := end.Sub(start)
	interval := kl[n-1].Interval
	if interval == "" && n > 1 {
		interval = kl[0].Interval
	}
	outPrintf("%sKlines:%s N=%d Interval=%s Span=%s\n", ansiBold, ansiReset, n, interval, span.Truncate(time.Second))
	outPrintf("Range: %s -> %s\n", start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	outPrintf("O/C: %.8f -> %.8f  Change: %+.8f (%.4f%%)\n", firstOpen, lastClose, change, changePct)
	outPrintf("High/Low: %.8f / %.8f  Volume: total=%.4f avg=%.4f\n", high, low, totalVol, avgVol)
}

func renderAccountHoldings(h *account.Holdings) {
	if h == nil || len(h.Accounts) == 0 {
		outPrintln("No holdings.")
		return
	}
	outPrintf("%s%-12s %-8s %-10s %-10s %-10s %-10s%s\n", ansiDim, "Account", "Asset", "Currency", "Total", "Free", "Hold", ansiReset)
	for i := range h.Accounts {
		sa := h.Accounts[i]
		for j := range sa.Currencies {
			c := sa.Currencies[j]
			outPrintf("%-12s %-8s %-10s % -10.8f % -10.8f % -10.8f\n",
				sa.ID, sa.AssetType, c.Currency.String(), c.Total, c.Free, c.Hold)
		}
	}
}

func renderActiveOrders(orders []order.Detail) {
	if len(orders) == 0 {
		outPrintln("No active orders.")
		return
	}
	outPrintf("%s%-19s %-10s %-6s %-10s %-10s %-10s %-10s %-20s%s\n", ansiDim, "OrderID", "Type", "Side", "Price", "Amount", "Filled", "Status", "Updated", ansiReset)
	for i := range orders {
		o := orders[i]
		filled := o.ExecutedAmount
		outPrintf("%-19s %-10s %-6s % -10.8f % -10.8f % -10.8f %-10s %-20s\n",
			o.OrderID, o.Type.String(), o.Side.String(), o.Price, o.Amount, filled, o.Status.String(), o.LastUpdated.UTC().Format(time.RFC3339))
	}
}

func renderExecutionLimits(l *limits.MinMaxLevel) {
	if l == nil {
		return
	}
	outPrintf("%sExecution Limits%s\n", ansiBold, ansiReset)
	outPrintf("MinPrice: %.8f  MaxPrice: %.8f  MinBaseAmt: %.8f  MaxBaseAmt: %.8f  MinQuoteAmt: %.8f  MaxQuoteAmt: %.8f  MinNotional: %.8f\n",
		l.MinPrice, l.MaxPrice, l.MinimumBaseAmount, l.MaximumBaseAmount, l.MinimumQuoteAmount, l.MaximumQuoteAmount, l.MinNotional)
}

func renderURL(u string) {
	outPrintf("%sTrade URL:%s %s\n", ansiBold, ansiReset, u)
}

func renderContract(c *futures.Contract) {
	if c == nil {
		return
	}
	outPrintf("%sContract%s %s  Type:%s  Multiplier:%.4f  Start:%s  End:%s  Settlement:%s\n",
		ansiBold, ansiReset, c.Name.String(), c.Type.String(), c.Multiplier,
		c.StartDate.UTC().Format(time.RFC3339), c.EndDate.UTC().Format(time.RFC3339), c.SettlementCurrencies.Join())
}

func renderPrettyPayload(payload any, bookLevels int) {
	switch v := payload.(type) {
	case *orderbook.Book:
		renderOrderbook(v, bookLevels)
	case []ticker.Price:
		if len(v) > 0 {
			renderTicker(&v[0])
		}
	case *ticker.Price:
		renderTicker(v)
	case []websocket.KlineData:
		renderKlines(v)
	case []trade.Data:
		renderTrades(v)
	case trade.Data:
		renderTrades([]trade.Data{v})
	case *account.Holdings:
		renderAccountHoldings(v)
	case []order.Detail:
		renderActiveOrders(v)
	case float64:
		renderOpenInterest(v)
	case *fundingrate.LatestRateResponse:
		renderFundingRate(v)
	case *futures.Contract:
		renderContract(v)
	case *limits.MinMaxLevel:
		renderExecutionLimits(v)
	case string:
		renderURL(v)
	default:
		outPrintf("%v\n", v)
	}
}

func renderOpenInterest(v float64) {
	outPrintf("%sOpen Interest:%s %.4f\n", ansiBold, ansiReset, v)
}

func renderFundingRate(fr *fundingrate.LatestRateResponse) {
	if fr == nil {
		outPrintln("No funding rate.")
		return
	}
	outPrintf("%sFunding:%s latest=%.6f predicted=%.6f next=%s checked=%s\n",
		ansiBold, ansiReset,
		fr.LatestRate.Rate.InexactFloat64(), fr.PredictedUpcomingRate.Rate.InexactFloat64(),
		fr.TimeOfNextRate.UTC().Format(time.RFC3339), fr.TimeChecked.UTC().Format(time.RFC3339))
}

// intMin for dealing with ints
func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
