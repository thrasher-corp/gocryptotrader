package main

import (
	"context"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
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

var enc = json.NewEncoder(os.Stdout)

// errorless stdout helpers to avoid handling write errors everywhere.
func outPrintf(format string, args ...any) { _, _ = fmt.Fprintf(os.Stdout, format, args...) }
func outPrintln(args ...any)               { _, _ = fmt.Fprintln(os.Stdout, args...) }
func outPrint(args ...any)                 { _, _ = fmt.Fprint(os.Stdout, args...) }

// eventEnvelope is a small wrapper to add metadata around exported data.
// It is printed as NDJSON to stdout for easy streaming/consumption.
type eventEnvelope struct {
	Timestamp time.Time `json:"ts"`
	Focus     string    `json:"focus"`
	Data      any       `json:"data,omitempty"`
	Error     error     `json:"error,omitempty"`
}

// appConfig holds parsed CLI configuration in a structured way.
type appConfig struct {
	Exchange     string
	Asset        asset.Item
	Pair         currency.Pair
	FocusType    quickdata.FocusType
	UseWebsocket bool
	PollInterval time.Duration
	BookLevels   int
	Credentials  *account.Credentials
	JSONOnly     bool
}

func main() {
	outPrintln("Hello! 🌞")
	defer outPrintln("\nGoodbye! 🌚")
	cfg := parseFlags()
	// Context & OS signals for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	ctx = account.DeployCredentialsToContext(ctx, cfg.Credentials)
	k := key.NewExchangeAssetPair(cfg.Exchange, cfg.Asset, cfg.Pair)
	qsChan, err := quickdata.NewQuickestData(ctx, &k, cfg.FocusType)
	if err != nil {
		emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: cfg.FocusType.String(), Error: err})
		return
	}
	outPrintln("QuickData setup, waiting for initial data...")
	if err := streamData(ctx, qsChan, cfg); err != nil {
		emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: cfg.FocusType.String(), Error: err})
		return
	}
}

// parseFlags parses CLI flags, validates them, applies sensible defaults, and returns appConfig.
func parseFlags() *appConfig {
	exch := flag.String("exchange", "", "Exchange name, e.g. binance, okx")
	assetStr := flag.String("asset", "spot", "Asset type, e.g. spot, futures, perpetualswap")
	pairStr := flag.String("pair", "BTC-USDT", "Currency pair, e.g. BTC-USDT or ETHUSD")
	focusStr := flag.String("data", "ticker", "Data type: ticker, orderbook, kline, trades, openinterest, fundingrate, accountholdings, activeorders, orderexecution, url, contract")
	pollStr := flag.String("poll", "5s", "Poll interval for REST focus and timeout for websocket initial data wait")
	bookLevels := flag.Int("book-levels", 15, "Number of levels to render per side for orderbook focus")
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
		_, _ = fmt.Fprintln(os.Stderr, "missing required flags: --exchange, --asset, --pair, --focusType")
		_, _ = fmt.Fprintln(os.Stderr, "please read the readme for more information")
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
	fType, err := parseFocusType(*focusStr)
	if err != nil {
		fatalErr(err)
	}

	pollDur, err := time.ParseDuration(*pollStr)
	if err != nil {
		fatalErr(fmt.Errorf("invalid poll duration: %w", err))
	}

	// Credentials (only applied when supplied or required by focus)
	var creds *account.Credentials
	if quickdata.RequiresAuth(fType) {
		creds = &account.Credentials{
			Key:             *apiKey,
			Secret:          *apiSecret,
			SubAccount:      *subAccount,
			ClientID:        *clientID,
			OneTimePassword: *otp,
			PEMKey:          *pemKey,
		}
		if creds.IsEmpty() {
			fatalErr(fmt.Errorf("focus %s requires credentials; provide --apiKey and --apiSecret (and others as needed)", fType.String()))
		}
		if creds.PEMKey != "" {
			if block, _ := pem.Decode([]byte(creds.PEMKey)); block == nil {
				fatalErr(errors.New("invalid PEM key format"))
			}
		}
	}

	return &appConfig{
		Exchange:     strings.ToLower(*exch),
		Asset:        ast,
		Pair:         cp,
		FocusType:    fType,
		PollInterval: pollDur,
		BookLevels:   *bookLevels,
		Credentials:  creds,
		JSONOnly:     *jsonOnly,
	}
}

func streamData(ctx context.Context, c <-chan any, cfg *appConfig) error {
	heading := fmt.Sprintf("%s | %s | %s", cfg.Exchange, cfg.Asset, cfg.Pair)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d := <-c:
			switch {
			case cfg.JSONOnly:
				emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: cfg.FocusType.String(), Data: d})
			default:
				clearScreen()
				outPrintf("%s%s%s\n", ansiBold, heading, ansiReset)
				renderPrettyPayload(d, cfg.BookLevels)
				if cfg.FocusType != quickdata.TickerFocusType && cfg.FocusType != quickdata.KlineFocusType && cfg.FocusType != quickdata.OrderBookFocusType {
					// executive decision to not render large payloads
					emit(eventEnvelope{Timestamp: time.Now().UTC(), Focus: cfg.FocusType.String(), Data: d})
				}
			}
		}
	}
}

// emit writes NDJSON events to stdout.
func emit(ev eventEnvelope) {
	if err := enc.Encode(ev); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to emit event: %v %v\n", ev, err)
	}
}

func parseFocusType(s string) (quickdata.FocusType, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "ticker", "tick":
		return quickdata.TickerFocusType, nil
	case "orderbook", "order_book", "ob", "book":
		return quickdata.OrderBookFocusType, nil
	case "kline", "candles", "candle", "ohlc":
		return quickdata.KlineFocusType, nil
	case "trades", "trade":
		return quickdata.TradesFocusType, nil
	case "openinterest", "oi":
		return quickdata.OpenInterestFocusType, nil
	case "fundingrate", "funding":
		return quickdata.FundingRateFocusType, nil
	case "accountholdings", "account", "holdings", "balances":
		return quickdata.AccountHoldingsFocusType, nil
	case "activeorders", "orders":
		return quickdata.ActiveOrdersFocusType, nil
	case "orderexecution", "executionlimits", "limits":
		return quickdata.OrderLimitsFocusType, nil
	case "url", "tradeurl", "trade_url":
		return quickdata.URLFocusType, nil
	case "contract":
		return quickdata.ContractFocusType, nil
	default:
		return quickdata.UnsetFocusType, quickdata.ErrUnsupportedFocusType
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
	for i := range askCount {
		lvl := b.Asks[i]
		cumulativeAsks += lvl.Amount
		outPrintf("%s% -14.8f % -14.8f % -14.8f%s\n", ansiRed, lvl.Price, lvl.Amount, cumulativeAsks, ansiReset)
	}
	outPrintln()
	// Bids (green), from best (highest) up to levels
	bidCount := intMin(levels, len(b.Bids))
	cumulativeBids := 0.0
	for i := range bidCount {
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
		for i := range v {
			renderTicker(&v[i])
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
