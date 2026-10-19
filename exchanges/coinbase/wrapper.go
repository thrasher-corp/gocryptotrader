package coinbase

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const (
	marketDataWebsocketURL         = "wss://advanced-trade-ws.coinbase.com"
	userWebsocketURL               = "wss://advanced-trade-ws-user.coinbase.com"
	derivativesStreamsWebsocketURL = "wss://streams.drb.coinbase.com/ws/api/v2"
	derivativesWebsocketURL        = "wss://drb.coinbase.com/ws/api/v2"

	// tradeBaseURL is Advanced Trade's web app, whose trade pages are named by market and product
	tradeBaseURL = "https://www.coinbase.com/advanced-trade/"

	// accountsPageLimit is the most accounts a page of the account list holds
	accountsPageLimit = 250
	// ordersPageLimit and fillsPageLimit are the most orders and fills a page of their lists holds as the wrapper asks
	// for them
	ordersPageLimit = 1000
	fillsPageLimit  = 1000
	// cancelOrdersBatchSize is the most orders a cancel request names, as Coinbase's earlier documentation capped it
	cancelOrdersBatchSize = 100

	// derivativesBookDepth is how many levels a side of a gateway book UpdateOrderbook fetches
	derivativesBookDepth = 1000
	// derivativesFundingRatePageLimit is the most hourly funding records a gateway funding history page holds
	derivativesFundingRatePageLimit = 744
	// derivativesLatestFundingRateWindow is how far back GetLatestFundingRates looks for a perpetual's latest hourly
	// funding record
	derivativesLatestFundingRateWindow = 24 * time.Hour
	// derivativesFundingInterval is how often a gateway perpetual's funding is recorded
	derivativesFundingInterval = time.Hour
	// derivativesCandleBase is the interval the gateway's candles of a longer interval are built from, as the gateway
	// aligns its own longer candles to its 08:00 UTC daily settlement
	derivativesCandleBase = kline.TwoHour
	// derivativesNoContinuation is the continuation the gateway sends once a history is exhausted
	derivativesNoContinuation = "none"
	// derivativesInstrumentTypeReversed is an inverse instrument's type
	derivativesInstrumentTypeReversed = "reversed"
	// exchangeNetworkStatusOnline is the status of a network Coinbase Exchange moves a currency on
	exchangeNetworkStatusOnline = "online"

	// The fee rates of the lowest fee tier, which the offline fee estimate charges, and of the stable pairs, which
	// Coinbase charges every tier
	offlineTakerFeeRate    = 0.012
	offlineMakerFeeRate    = 0.006
	stablePairTakerFeeRate = 0.000045
	stablePairMakerFeeRate = 0
)

var (
	errProductNotFound           = errors.New("product not found")
	errInstrumentNotFound        = errors.New("instrument not found")
	errPerpetualFuturesNotLoaded = errors.New("US derivatives product data not loaded")
	errAccountNotFound           = errors.New("no account holds the currency")
	errPaymentMethodNotFound     = errors.New("payment method not found")
	errNoTransferChains          = errors.New("no networks transfer the currency")
	errListPagingStalled         = errors.New("list paging did not advance")
	errTradesPageFull            = errors.New("trades from one second fill a whole page")
	errCandleFieldsMismatch      = errors.New("candle fields differ in length")
	errTrailingStopDistance      = errors.New("a trailing stop needs a distance tracking value")
	errOrderLabelNotUnique       = errors.New("label does not identify one open order")
	errReduceOnlyNotSupported    = errors.New("reduce only orders are not supported")
	// errProofTokenRequired is the EU's strong customer authentication asking for a proof token, which only the account
	// holder can give, in place of the list asked for
	errProofTokenRequired = errors.New("a strong customer authentication proof token is required")
)

// stablePairs are the pairs Coinbase charges its stable pair fees for
var stablePairs = []currency.Pair{
	currency.NewPair(currency.USDT, currency.USD),
	currency.NewPair(currency.USDT, currency.EUR),
	currency.NewPair(currency.USDC, currency.EUR),
	currency.NewPair(currency.USDC, currency.GBP),
	currency.NewPair(currency.USDT, currency.GBP),
	currency.NewPair(currency.USDT, currency.USDC),
	currency.NewPair(currency.DAI, currency.USD),
	currency.NewPair(currency.CBETH, currency.ETH),
	currency.NewPair(currency.PYUSD, currency.USD),
	currency.NewPair(currency.EURC, currency.USD),
	currency.NewPair(currency.GUSD, currency.USD),
	currency.NewPair(currency.EURC, currency.EUR),
	currency.NewPair(currency.WBTC, currency.BTC),
	currency.NewPair(currency.LSETH, currency.ETH),
	currency.NewPair(currency.GYEN, currency.USD),
	currency.NewPair(currency.PAX, currency.USD),
}

// SetDefaults sets default values for the exchange
func (e *Exchange) SetDefaults() {
	e.Name = "Coinbase"
	e.Enabled = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	dashFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	underscoreFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}
	for a, format := range map[asset.Item]*currency.PairFormat{
		asset.Spot:                dashFormat,
		asset.Futures:             dashFormat,
		asset.USDCMarginedFutures: underscoreFormat,
		asset.CoinMarginedFutures: dashFormat,
	} {
		if err := e.SetAssetPairStore(a, currency.PairStore{AssetEnabled: true, RequestFormat: format, ConfigFormat: format}); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountBalance:      true,
				AccountInfo:         true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				FiatWithdraw:        true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				ModifyOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				TradeFetching:       true,
				UserTradeHistory:    true,
				TradeFee:            true,
				KlineFetching:       true,
				FundingRateFetching: true,
				// A currency moving over several networks is deposited and withdrawn over the one named
				MultiChainDeposits:             true,
				MultiChainWithdrawals:          true,
				HasAssetTypeAccountSegregation: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				TradeFetching:          true,
				KlineFetching:          true,
				AccountInfo:            true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageSequenceNumbers: true,
				MessageCorrelation:     true,
				// The gateway's orders go over its websocket where the wrapper may use it
				SubmitOrder:  true,
				CancelOrder:  true,
				CancelOrders: true,
				ModifyOrder:  true,
				GetOrder:     true,
				GetOrders:    true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission | exchange.AutoWithdrawFiatWithAPIPermission,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates:                    true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{kline.OneHour: true},
				FundingRateBatching:             map[asset.Item]bool{asset.Futures: true, asset.USDCMarginedFutures: true, asset.CoinMarginedFutures: true},
				Positions:                       true,
				Leverage:                        true,
				OpenInterest:                    exchange.OpenInterestSupport{Supported: true, SupportsRestBatch: true},
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
				),
				GlobalResultLimit: maximumCandles,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	e.rateLimits = GetRateLimits()
	var err error
	if e.Requester, err = request.New(e.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(e.rateLimits)); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                apiURL,
		exchange.RestSandbox:             sandboxAPIURL,
		exchange.RestSpotSupplementary:   exchangeAPIURL,
		exchange.RestFutures:             derivativesAPIURL,
		exchange.WebsocketSpot:           marketDataWebsocketURL,
		exchange.WebsocketPrivate:        userWebsocketURL,
		exchange.WebsocketFutures:        derivativesStreamsWebsocketURL,
		exchange.WebsocketFuturesPrivate: derivativesWebsocketURL,
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup initialises the exchange parameters with the current configuration
func (e *Exchange) Setup(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	if err := e.SetupDefaults(exch); err != nil {
		return err
	}
	return e.setupWebsocket(exch)
}

// isAdvancedTradeAsset reports whether an asset trades through Advanced Trade: spot, and US derivatives as futures
func isAdvancedTradeAsset(a asset.Item) bool {
	return a == asset.Spot || a == asset.Futures
}

// productID returns a pair's Advanced Trade product ID
func (e *Exchange) productID(p currency.Pair, a asset.Item) (string, error) {
	if !isAdvancedTradeAsset(a) {
		return "", fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	return e.formattedSymbol(p, a)
}

// instrumentName returns a pair's gateway instrument name
func (e *Exchange) instrumentName(p currency.Pair, a asset.Item) (string, error) {
	if !isDerivativesAsset(a) {
		return "", fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	return e.formattedSymbol(p, a)
}

// formattedSymbol returns a pair as its asset's requests name it
func (e *Exchange) formattedSymbol(p currency.Pair, a asset.Item) (string, error) {
	if p.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	formatted, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return "", err
	}
	return formatted.String(), nil
}

// productPair returns the pair of an Advanced Trade product ID, as the asset's available pairs hold it
func (e *Exchange) productPair(productID string, a asset.Item) (currency.Pair, error) {
	if p, err := e.MatchSymbolWithAvailablePairs(productID, a, true); err == nil {
		return p, nil
	}
	return currency.NewPairFromString(productID)
}

// FetchTradablePairs returns the pairs an asset trades: the Advanced Trade spot products and US derivatives that are
// open to trading, and the gateway's active perpetuals for the perpetual assets
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch {
	case isAdvancedTradeAsset(a):
		products, err := e.listProducts(ctx, a)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, 0, len(products))
		for i := range products {
			if !tradableProduct(&products[i]) {
				continue
			}
			p, err := currency.NewPairDelimiter(products[i].ProductID, currency.DashDelimiter)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, p)
		}
		return pairs, nil
	case isDerivativesAsset(a):
		instruments, err := e.derivativesPerpetuals(ctx, a)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, 0, len(instruments))
		for i := range instruments {
			if !instruments[i].IsActive {
				continue
			}
			_, p, err := derivativesInstrumentPair(instruments[i].InstrumentName)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, p)
		}
		return pairs, nil
	}
	return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(ctx, a)
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, a, false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// listProducts returns every product of an Advanced Trade asset: the spot products, or the US derivatives that trade
// through Coinbase Financial Markets, view only ones among them. It loads the spot products' aliases and which US
// derivatives pay funding, as both come with product data
func (e *Exchange) listProducts(ctx context.Context, a asset.Item) ([]Product, error) {
	req := &ListProductsRequest{ProductType: ProductTypeSpot}
	if a == asset.Futures {
		req = &ListProductsRequest{ProductType: ProductTypeFuture, ProductVenue: ProductVenueFCM}
	}
	var products []Product
	cursors := pageCursors{}
	for {
		cursors.visit(req.Cursor)
		resp, err := e.ListPublicProducts(ctx, req)
		if err != nil {
			return nil, err
		}
		products = append(products, resp.Products...)
		if !resp.Pagination.HasNext || resp.Pagination.NextCursor == "" {
			break
		}
		if cursors.seen(resp.Pagination.NextCursor) {
			return nil, fmt.Errorf("%w: products cursor %q", errListPagingStalled, resp.Pagination.NextCursor)
		}
		req.Cursor = resp.Pagination.NextCursor
	}
	if a == asset.Futures {
		// The product venue only narrows the list as the docs describe it, so the venue is checked here as well
		products = slices.DeleteFunc(products, func(p Product) bool { return p.ProductVenue != ProductVenueFCM })
		e.loadPerpetualFutures(products)
		return products, nil
	}
	return products, e.loadPairAliases(products)
}

// tradableProduct reports whether a product is open to trading: listed and neither disabled nor view only, as a US
// derivative is until its contract month opens
func tradableProduct(p *Product) bool {
	return !p.TradingDisabled && !p.IsDisabled && !p.ViewOnly
}

// loadPairAliases loads which spot products trade on one book, the USD product and its USDC aliases, each also its
// own alias as the websocket's alias lookups expect
func (e *Exchange) loadPairAliases(products []Product) error {
	aliases := make(map[currency.Pair]currency.Pairs)
	for i := range products {
		p, err := currency.NewPairDelimiter(products[i].ProductID, currency.DashDelimiter)
		if err != nil {
			return err
		}
		aliases[p] = aliases[p].Add(p)
		if products[i].Alias != "" {
			alias, err := currency.NewPairDelimiter(products[i].Alias, currency.DashDelimiter)
			if err != nil {
				return err
			}
			aliases[alias] = aliases[alias].Add(p)
		}
		for _, aliasTo := range products[i].AliasTo {
			alias, err := currency.NewPairDelimiter(aliasTo, currency.DashDelimiter)
			if err != nil {
				return err
			}
			aliases[p] = aliases[p].Add(alias)
		}
	}
	e.pairAliases.Load(aliases)
	return nil
}

// isPerpetualProduct reports whether a US derivative is perpetual style, paying funding rather than expiring soon;
// Coinbase lists these as expiring contracts, years out
func isPerpetualProduct(p *Product) bool {
	return p.FutureProductDetails != nil && p.FutureProductDetails.FundingInterval != 0
}

// loadPerpetualFutures records which US derivatives are perpetual style
func (e *Exchange) loadPerpetualFutures(products []Product) {
	perpetuals := make(map[string]struct{})
	for i := range products {
		if isPerpetualProduct(&products[i]) {
			perpetuals[products[i].ProductID] = struct{}{}
		}
	}
	e.perpetualFuturesMu.Lock()
	e.perpetualFutures = perpetuals
	e.perpetualFuturesMu.Unlock()
}

// derivativesPerpetuals returns the gateway's perpetuals of a perpetual asset, active or not; its dated futures,
// options, spot and combos belong to no supported asset
func (e *Exchange) derivativesPerpetuals(ctx context.Context, a asset.Item) ([]DerivativesInstrument, error) {
	instruments, err := e.GlobalDerivatives().getInstruments(ctx, e.derivativesOverWebsocket(ctx, false), &DerivativesInstrumentsRequest{Currency: currency.NewCode(DerivativesCurrencyAny), Kind: DerivativesKindFuture})
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(instruments, func(i DerivativesInstrument) bool {
		instrumentAsset, _, err := derivativesInstrumentPair(i.InstrumentName)
		return err != nil || instrumentAsset != a
	}), nil
}

// UpdateOrderExecutionLimits stores the price and amount increments and bounds of an asset's products or instruments;
// a gateway instrument's minimum amount is also its amount increment
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	switch {
	case isAdvancedTradeAsset(a):
		products, err := e.listProducts(ctx, a)
		if err != nil {
			return err
		}
		l := make([]limits.MinMaxLevel, len(products))
		for i := range products {
			p := &products[i]
			pair, err := currency.NewPairDelimiter(p.ProductID, currency.DashDelimiter)
			if err != nil {
				return err
			}
			priceStep := p.PriceIncrement.Float64()
			if priceStep == 0 {
				priceStep = p.QuoteIncrement.Float64()
			}
			l[i] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
				PriceStepIncrementSize:  priceStep,
				MinimumBaseAmount:       p.BaseMinimumSize.Float64(),
				MaximumBaseAmount:       p.BaseMaximumSize.Float64(),
				MinimumQuoteAmount:      p.QuoteMinimumSize.Float64(),
				MaximumQuoteAmount:      p.QuoteMaximumSize.Float64(),
				AmountStepIncrementSize: p.BaseIncrement.Float64(),
				QuoteStepIncrementSize:  p.QuoteIncrement.Float64(),
			}
			if d := p.FutureProductDetails; d != nil && !isPerpetualProduct(p) {
				l[i].Expiry = d.ContractExpiry
			}
		}
		return limits.Load(l)
	case isDerivativesAsset(a):
		instruments, err := e.derivativesPerpetuals(ctx, a)
		if err != nil {
			return err
		}
		l := make([]limits.MinMaxLevel, len(instruments))
		for i := range instruments {
			_, pair, err := derivativesInstrumentPair(instruments[i].InstrumentName)
			if err != nil {
				return err
			}
			l[i] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
				PriceStepIncrementSize:  instruments[i].TickSize,
				MinimumBaseAmount:       instruments[i].MinimumTradeAmount,
				AmountStepIncrementSize: instruments[i].MinimumTradeAmount,
			}
		}
		return limits.Load(l)
	}
	return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
}

// UpdateTickers updates the tickers of an asset's enabled pairs: from the product list for Advanced Trade, which
// carries no best bid or ask, and from the book summaries of the currencies the gateway's perpetuals settle in
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	var prices []ticker.Price
	var errs error
	switch {
	case isAdvancedTradeAsset(a):
		products, err := e.listProducts(ctx, a)
		if err != nil {
			return err
		}
		byID := make(map[string]*Product, len(products))
		for i := range products {
			byID[products[i].ProductID] = &products[i]
		}
		for _, p := range pairs {
			id, err := e.productID(p, a)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			product, ok := byID[id]
			if !ok {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", errProductNotFound, id))
				continue
			}
			prices = append(prices, *e.productTickerPrice(product, p, a))
		}
	case isDerivativesAsset(a):
		var currencies currency.Currencies
		for _, p := range pairs {
			if c := derivativesSettlementCurrency(a, p); !currencies.Contains(c) {
				currencies = append(currencies, c)
			}
		}
		summaries, err := e.derivativesBookSummaries(ctx, currencies)
		if err != nil {
			return err
		}
		for _, p := range pairs {
			name, err := e.instrumentName(p, a)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			summary, ok := summaries[name]
			if !ok {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", errInstrumentNotFound, name))
				continue
			}
			prices = append(prices, *summary.tickerPrice(e.Name, p, a))
		}
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if len(prices) == 0 {
		return errs
	}
	// The product list carries no best bid or ask, which the websocket's tickers do
	_, err = e.processMergedTickers(prices, isAdvancedTradeAsset(a), false)
	return common.AppendError(errs, err)
}

// derivativesBookSummaries returns the gateway's book summaries of the futures settling in the currencies given, by
// instrument name, one request a currency, as the gateway summarises one currency at a time
func (e *Exchange) derivativesBookSummaries(ctx context.Context, currencies currency.Currencies) (map[string]*DerivativesBookSummary, error) {
	overWebsocket := e.derivativesOverWebsocket(ctx, false)
	summaries := make(map[string]*DerivativesBookSummary)
	for _, c := range currencies {
		resp, err := e.GlobalDerivatives().getBookSummaryByCurrency(ctx, overWebsocket, &DerivativesBookSummaryByCurrencyRequest{Currency: c, Kind: DerivativesKindFuture})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c, err)
		}
		for i := range resp {
			summaries[resp[i].InstrumentName] = &resp[i]
		}
	}
	return summaries, nil
}

// derivativesAssetCurrencies returns the currencies a perpetual asset's instruments settle in: USDC for the USDC
// perpetuals, and bitcoin and ether, the gateway's only inverse perpetuals, for the inverse perpetuals
func derivativesAssetCurrencies(a asset.Item) currency.Currencies {
	if a == asset.CoinMarginedFutures {
		return currency.Currencies{currency.BTC, currency.ETH}
	}
	return currency.Currencies{currency.USDC}
}

// derivativesSettlementCurrency returns the currency a gateway perpetual settles and is margined in: USDC for the
// USDC perpetuals, and the base currency for the inverse perpetuals
func derivativesSettlementCurrency(a asset.Item, p currency.Pair) currency.Code {
	if a == asset.USDCMarginedFutures {
		return currency.USDC
	}
	return p.Base.Upper()
}

// UpdateTicker updates and returns a pair's ticker: an Advanced Trade product's day statistics with its best bid and
// ask, or a gateway instrument's ticker
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	var price *ticker.Price
	switch {
	case isAdvancedTradeAsset(a):
		id, err := e.productID(p, a)
		if err != nil {
			return nil, err
		}
		product, err := e.GetPublicProduct(ctx, id)
		if err != nil {
			return nil, err
		}
		trades, err := e.GetPublicMarketTrades(ctx, &MarketTradesRequest{ProductID: id, Limit: 1})
		if err != nil {
			return nil, err
		}
		price = e.productTickerPrice(product, p, a)
		price.Bid, price.Ask = trades.BestBid.Float64(), trades.BestAsk.Float64()
		if len(trades.Trades) != 0 {
			price.LastSize = trades.Trades[0].Size.Float64()
		}
	case isDerivativesAsset(a):
		name, err := e.instrumentName(p, a)
		if err != nil {
			return nil, err
		}
		resp, err := e.GlobalDerivatives().getTicker(ctx, e.derivativesOverWebsocket(ctx, false), name)
		if err != nil {
			return nil, err
		}
		price = resp.tickerPrice(e.Name, p, a)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if err := ticker.ProcessTicker(price); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// processMergedTickers stores tickers, each first given the fields its source leaves out from the pair's stored
// ticker, which another source updated: the best bid and ask, where a ticker has neither and book is set, and the
// quote volume, where quoteVolume is set. The merges and the store are held together, as a merge between another's
// read and write would restore what that one replaced
func (e *Exchange) processMergedTickers(prices []ticker.Price, book, quoteVolume bool) ([]ticker.Price, error) {
	e.tickerMergeMu.Lock()
	defer e.tickerMergeMu.Unlock()
	for i := range prices {
		price := &prices[i]
		stored, err := ticker.GetTicker(price.ExchangeName, price.Pair, price.AssetType)
		if err != nil {
			continue
		}
		if book && price.Bid == 0 && price.Ask == 0 {
			price.Bid, price.BidSize, price.Ask, price.AskSize = stored.Bid, stored.BidSize, stored.Ask, stored.AskSize
		}
		if quoteVolume {
			price.QuoteVolume = stored.QuoteVolume
		}
	}
	return ticker.ProcessBatch(prices)
}

// productTickerPrice returns a product's day statistics as a ticker; a US derivative's volume and open interest are
// in contracts, the unit its base names
func (e *Exchange) productTickerPrice(p *Product, pair currency.Pair, a asset.Item) *ticker.Price {
	price := &ticker.Price{
		ExchangeName:        e.Name,
		Pair:                pair,
		AssetType:           a,
		Last:                p.Price.Float64(),
		High:                p.High24Hour.Float64(),
		Low:                 p.Low24Hour.Float64(),
		Bid:                 p.BestBidPrice.Float64(),
		Ask:                 p.BestAskPrice.Float64(),
		BaseVolume:          p.Volume24Hour.Float64(),
		QuoteVolume:         p.ApproximateQuote24HourVolume.Float64(),
		PercentChange24Hour: p.PricePercentageChange24Hour.Float64(),
	}
	if d := p.FutureProductDetails; d != nil {
		price.IndexPrice = d.IndexPrice.Float64()
		price.OpenInterest = d.OpenInterest.Float64()
	}
	return price
}

// tickerPrice returns a book summary as a ticker; its creation timestamp is when the gateway compiled it
func (b *DerivativesBookSummary) tickerPrice(exchangeName string, p currency.Pair, a asset.Item) *ticker.Price {
	return &ticker.Price{
		ExchangeName:        exchangeName,
		Pair:                p,
		AssetType:           a,
		LastUpdated:         b.CreationTimestamp.Time(),
		Bid:                 b.BidPrice,
		Ask:                 b.AskPrice,
		Last:                b.LastPrice,
		Close:               b.LastPrice,
		High:                b.High,
		Low:                 b.Low,
		BaseVolume:          b.Volume,
		QuoteVolume:         derivativesQuoteVolume(b.VolumeUSD, b.VolumeNotional),
		PercentChange24Hour: b.PriceChange,
		MarkPrice:           b.MarkPrice,
		OpenInterest:        b.OpenInterest,
	}
}

// UpdateOrderbook updates and returns a pair's book, at Advanced Trade's default depth, or derivativesBookDepth levels
// a side on the gateway
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	switch {
	case isAdvancedTradeAsset(a):
		id, err := e.productID(p, a)
		if err != nil {
			return nil, err
		}
		resp, err := e.GetPublicProductBook(ctx, &ProductBookRequest{ProductID: id})
		if err != nil {
			return nil, err
		}
		book.Bids = advancedTradeBookLevels(resp.PriceBook.Bids)
		book.Asks = advancedTradeBookLevels(resp.PriceBook.Asks)
		book.LastUpdated = resp.PriceBook.Time
	case isDerivativesAsset(a):
		name, err := e.instrumentName(p, a)
		if err != nil {
			return nil, err
		}
		resp, err := e.GlobalDerivatives().getOrderBook(ctx, e.derivativesOverWebsocket(ctx, false), &DerivativesOrderBookRequest{InstrumentName: name, Depth: derivativesBookDepth})
		if err != nil {
			return nil, err
		}
		book.Bids = derivativesBookLevels(resp.Bids)
		book.Asks = derivativesBookLevels(resp.Asks)
		book.LastUpdated = resp.Timestamp.Time()
		book.LastUpdateID = int64(resp.ChangeID) //nolint:gosec // The gateway's change IDs fit int64
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, a)
}

// advancedTradeBookLevels returns a side of an Advanced Trade book, best first as Coinbase sends it
func advancedTradeBookLevels(levels []BookLevel) orderbook.Levels {
	converted := make(orderbook.Levels, len(levels))
	for i := range levels {
		converted[i] = orderbook.Level{Price: levels[i].Price.Float64(), Amount: levels[i].Size.Float64()}
	}
	return converted
}

// UpdateAccountBalances stores and returns the balances an asset trades from: the spot accounts for spot and the US
// derivatives accounts for futures, of the portfolio the API key belongs to, leaving out vaults, which cannot trade;
// or the gateway's account summaries, which back both perpetual assets and are split between them by currency, and
// each active isolated margin subaccount's as a sub-account of its own. A gateway balance's free amount is what can be
// withdrawn, which leaves out margin held, session profit and reserved balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	subAccount := accounts.NewSubAccount(a, "")
	subAccounts := accounts.SubAccounts{subAccount}
	switch {
	case isAdvancedTradeAsset(a):
		req := &ListAccountsRequest{Limit: accountsPageLimit}
		cursors := pageCursors{}
		for {
			cursors.visit(req.Cursor)
			resp, err := e.ListAccounts(ctx, req)
			if err != nil {
				return nil, err
			}
			for i := range resp.Accounts {
				acct := &resp.Accounts[i]
				if accountAsset(acct) != a {
					continue
				}
				// Coinbase's available balance leaves out what is held
				subAccount.Balances.Set(acct.Currency, accounts.Balance{
					Total: acct.AvailableBalance.Value.Float64() + acct.Hold.Value.Float64(),
					Hold:  acct.Hold.Value.Float64(),
					Free:  acct.AvailableBalance.Value.Float64(),
				})
			}
			if !resp.HasNext || resp.Cursor == "" {
				break
			}
			if cursors.seen(resp.Cursor) {
				return nil, fmt.Errorf("%w: accounts cursor %q", errListPagingStalled, resp.Cursor)
			}
			req.Cursor = resp.Cursor
		}
	case isDerivativesAsset(a):
		resp, err := e.GlobalDerivatives().getAccountSummaries(ctx, e.derivativesOverWebsocket(ctx, true), &DerivativesGetAccountSummariesRequest{IncludeIsolated: true})
		if err != nil {
			return nil, err
		}
		setDerivativesBalances(subAccount, resp.Summaries)
		// An isolated margin subaccount holds the margin moved to its position apart from the main account, so each is a
		// sub-account of its own, by its ID
		for i := range resp.IsolatedAccountSummaries {
			isolated := accounts.NewSubAccount(a, strconv.FormatUint(resp.IsolatedAccountSummaries[i].ID, 10))
			if setDerivativesBalances(isolated, resp.IsolatedAccountSummaries[i].Summaries) {
				subAccounts = append(subAccounts, isolated)
			}
		}
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	return subAccounts, e.Accounts.Save(ctx, subAccounts, true)
}

// setDerivativesBalances sets a sub-account's balances from the gateway account summaries in the currencies its asset
// reports, a summary's withdrawable funds being what is free, and reports whether it set any
func setDerivativesBalances(subAccount *accounts.SubAccount, summaries []DerivativesAccountSummary) bool {
	var set bool
	for i := range summaries {
		summary := &summaries[i]
		if derivativesBalanceAsset(summary.Currency) != subAccount.AssetType {
			continue
		}
		subAccount.Balances.Set(summary.Currency, accounts.Balance{
			Total: summary.Balance,
			Hold:  max(summary.Balance-summary.AvailableWithdrawalFunds, 0),
			Free:  summary.AvailableWithdrawalFunds,
		})
		set = true
	}
	return set
}

// derivativesBalanceAsset returns the perpetual asset a gateway currency's balance is reported under, so that the one
// gateway account both perpetual assets trade from counts each balance once: bitcoin and ether, which margin the
// inverse perpetuals, under the coin margined futures, and the rest, USDC among them, under the USDC margined futures
func derivativesBalanceAsset(c currency.Code) asset.Item {
	if derivativesAssetCurrencies(asset.CoinMarginedFutures).Contains(c) {
		return asset.CoinMarginedFutures
	}
	return asset.USDCMarginedFutures
}

// accountAsset returns the asset an Advanced Trade account's balance trades: spot for a consumer account, futures for
// a US derivatives account, and none for a vault or an International Exchange account, which no longer trades
func accountAsset(acct *Account) asset.Item {
	if acct.Type == AccountTypeVault {
		return asset.Empty
	}
	switch acct.Platform {
	case AccountPlatformConsumer, "":
		return asset.Spot
	case AccountPlatformCFMConsumer:
		return asset.Futures
	}
	return asset.Empty
}

// GetAccountFundingHistory returns the deposits and withdrawals of every Coinbase account: crypto sent and received,
// and fiat deposited and withdrawn. It lists each account's transactions in full, a page at a time, which takes a
// request a page of every account, trades included
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	appAccounts, err := e.appAccounts(ctx)
	if err != nil {
		return nil, err
	}
	var resp []exchange.FundingHistory
	for i := range appAccounts {
		transactions, err := e.appTransactions(ctx, appAccounts[i].ID)
		if err != nil {
			return nil, err
		}
		for j := range transactions {
			if transferType, ok := fundingTransferType(transactions[j].Type); ok {
				resp = append(resp, e.fundingHistory(&transactions[j], transferType))
			}
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns the withdrawals of a currency's accounts, or of every account when it is empty: crypto
// sent and fiat withdrawn
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	appAccounts, err := e.appAccounts(ctx)
	if err != nil {
		return nil, err
	}
	var resp []exchange.WithdrawalHistory
	for i := range appAccounts {
		if !c.IsEmpty() && !appAccounts[i].Currency.Code.Equal(c) {
			continue
		}
		transactions, err := e.appTransactions(ctx, appAccounts[i].ID)
		if err != nil {
			return nil, err
		}
		for j := range transactions {
			transferType, ok := fundingTransferType(transactions[j].Type)
			if !ok || transferType != withdrawalTransferType {
				continue
			}
			f := e.fundingHistory(&transactions[j], transferType)
			resp = append(resp, exchange.WithdrawalHistory{
				Status:          f.Status,
				TransferID:      f.TransferID,
				Description:     f.Description,
				Timestamp:       f.Timestamp,
				Currency:        f.Currency,
				Amount:          f.Amount,
				Fee:             f.Fee,
				TransferType:    f.TransferType,
				CryptoToAddress: f.CryptoToAddress,
				CryptoTxID:      f.CryptoTxID,
				CryptoChain:     f.CryptoChain,
			})
		}
	}
	return resp, nil
}

// Funding history transfer types
const (
	depositTransferType    = "deposit"
	withdrawalTransferType = "withdrawal"
)

// fundingTransferType returns whether a Coinbase App transaction deposits or withdraws funds, and false for any other
// transaction, such as a trade or a transfer between the user's own accounts
func fundingTransferType(transactionType string) (string, bool) {
	switch transactionType {
	case AppTransactionTypeReceive, AppTransactionTypeFiatDeposit:
		return depositTransferType, true
	case AppTransactionTypeSend, AppTransactionTypeFiatWithdrawal:
		return withdrawalTransferType, true
	}
	return "", false
}

// fundingHistory returns a deposit or withdrawal as GCT records it; Coinbase signs a transaction's amount by its
// direction, and reports the network fee of a send, in the currency the network charges
func (e *Exchange) fundingHistory(t *AppTransaction, transferType string) exchange.FundingHistory {
	f := exchange.FundingHistory{
		ExchangeName: e.Name,
		Status:       t.Status,
		TransferID:   t.ID,
		Description:  t.Description,
		Timestamp:    t.CreatedAt,
		Currency:     t.Amount.Currency.String(),
		Amount:       math.Abs(t.Amount.Amount.Float64()),
		TransferType: transferType,
	}
	if n := t.Network; n != nil {
		f.CryptoTxID = n.Hash
		f.CryptoChain = n.NetworkName
		if f.CryptoChain == "" {
			f.CryptoChain = n.Name
		}
		if n.TransactionFee != nil {
			f.Fee = n.TransactionFee.Amount.Float64()
		}
	}
	if t.To != nil {
		f.CryptoToAddress = t.To.Address
	}
	if t.From != nil {
		f.CryptoFromAddress = t.From.Address
	}
	return f
}

// appPages calls visit with each record of a Coinbase App list, a page at a time, until the list is exhausted or visit
// reports it has what it needs
func appPages[T any](list func(*AppPageRequest) (*AppListResponse[T], error), id func(*T) string, visit func(*T) bool) error {
	page := &AppPageRequest{Limit: appMaximumPageLimit}
	cursors := pageCursors{}
	for {
		cursors.visit(page.StartingAfter)
		resp, err := list(page)
		if err != nil {
			return err
		}
		for i := range resp.Data {
			if visit(&resp.Data[i]) {
				return nil
			}
		}
		if resp.Pagination.NextURI == "" || len(resp.Data) == 0 {
			return nil
		}
		next := resp.Pagination.NextStartingAfter
		if next == "" {
			next = id(&resp.Data[len(resp.Data)-1])
		}
		if cursors.seen(next) {
			return fmt.Errorf("%w: starting after %q", errListPagingStalled, next)
		}
		page = &AppPageRequest{Limit: appMaximumPageLimit, StartingAfter: next}
	}
}

// pageCursors holds the cursors a list has been paged by, so that a cursor the venue sends again, which would page the
// list in a loop, stops it
type pageCursors map[string]struct{}

// visit records a cursor a page is requested by
func (c pageCursors) visit(cursor string) { c[cursor] = struct{}{} }

// seen reports whether a page was already requested by a cursor
func (c pageCursors) seen(cursor string) bool {
	_, ok := c[cursor]
	return ok
}

// appAccounts returns every Coinbase App account the API key can see
func (e *Exchange) appAccounts(ctx context.Context) ([]AppAccount, error) {
	var resp []AppAccount
	err := appPages(func(page *AppPageRequest) (*AppListResponse[AppAccount], error) {
		return e.App().ListAccounts(ctx, &AppListAccountsRequest{Page: page})
	}, func(a *AppAccount) string { return a.ID }, func(a *AppAccount) bool {
		resp = append(resp, *a)
		return false
	})
	return resp, err
}

// appTransactions returns every transaction of a Coinbase App account
func (e *Exchange) appTransactions(ctx context.Context, accountID string) ([]AppTransaction, error) {
	var resp []AppTransaction
	err := appPages(func(page *AppPageRequest) (*AppListResponse[AppTransaction], error) {
		return e.App().ListTransactions(ctx, &AppListTransactionsRequest{AccountID: accountID, Page: page})
	}, func(t *AppTransaction) string { return t.ID }, func(t *AppTransaction) bool {
		resp = append(resp, *t)
		return false
	})
	return resp, err
}

// appAccountID returns the Coinbase App account holding a currency: the account ID given, or else the currency's
// primary account, or another of its accounts without one
func (e *Exchange) appAccountID(ctx context.Context, c currency.Code, accountID string) (string, error) {
	if accountID != "" {
		return accountID, nil
	}
	if c.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	var found *AppAccount
	err := appPages(func(page *AppPageRequest) (*AppListResponse[AppAccount], error) {
		return e.App().ListAccounts(ctx, &AppListAccountsRequest{Page: page})
	}, func(a *AppAccount) string { return a.ID }, func(a *AppAccount) bool {
		if !a.Currency.Code.Equal(c) {
			return false
		}
		if found == nil || a.Primary {
			account := *a
			found = &account
		}
		return a.Primary
	})
	if err != nil {
		return "", err
	}
	if found == nil {
		return "", fmt.Errorf("%w: %s", errAccountNotFound, c)
	}
	return found.ID, nil
}

// GetRecentTrades returns a pair's latest trades: at most 100 of an Advanced Trade product's, or a page of a gateway
// instrument's
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	var resp []trade.Data
	switch {
	case isAdvancedTradeAsset(a):
		id, err := e.productID(p, a)
		if err != nil {
			return nil, err
		}
		trades, err := e.GetPublicMarketTrades(ctx, &MarketTradesRequest{ProductID: id, Limit: maximumMarketTradesLimit})
		if err != nil {
			return nil, err
		}
		if resp, err = e.marketTrades(trades.Trades, p, a); err != nil {
			return nil, err
		}
	case isDerivativesAsset(a):
		name, err := e.instrumentName(p, a)
		if err != nil {
			return nil, err
		}
		trades, err := e.GlobalDerivatives().getLastTradesByInstrument(ctx, e.derivativesOverWebsocket(ctx, false), &DerivativesLastTradesByInstrumentRequest{InstrumentName: name, Count: derivativesMarketDataMaximumCount})
		if err != nil {
			return nil, err
		}
		if resp, err = e.derivativesTrades(trades.Trades, p, a); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return resp, nil
}

// GetHistoricTrades returns a pair's trades between two times: an Advanced Trade product's paging back from the end,
// and a gateway instrument's paging forward from the start, which the gateway serves for the last 24 hours alone
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	var resp []trade.Data
	var err error
	switch {
	case isAdvancedTradeAsset(a):
		resp, err = e.historicMarketTrades(ctx, p, a, start, end)
	case isDerivativesAsset(a):
		resp, err = e.historicDerivativesTrades(ctx, p, a, start, end)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if err != nil {
		return nil, err
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return trade.FilterTradesByTime(resp, start, end), nil
}

// historicMarketTrades returns a product's trades between two times
func (e *Exchange) historicMarketTrades(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time) ([]trade.Data, error) {
	id, err := e.productID(p, a)
	if err != nil {
		return nil, err
	}
	trades, err := pageMarketTrades(start, end, func(pageEnd time.Time) ([]MarketTrade, error) {
		resp, err := e.GetPublicMarketTrades(ctx, &MarketTradesRequest{ProductID: id, Limit: maximumMarketTradesLimit, Start: start, End: pageEnd})
		if err != nil {
			return nil, err
		}
		return resp.Trades, nil
	})
	if err != nil {
		return nil, err
	}
	return e.marketTrades(trades, p, a)
}

// pageMarketTrades returns a product's trades between two times, which Coinbase sends newest first, a page at a time
// paging back from the end. Coinbase bounds a window by whole seconds, so each page ends a second after the oldest
// trade of the last, and the trades both hold are kept once; a page of trades all from its last second cannot reach
// the trades before them
func pageMarketTrades(start, end time.Time, fetch func(pageEnd time.Time) ([]MarketTrade, error)) ([]MarketTrade, error) {
	var resp []MarketTrade
	seen := make(map[string]struct{})
	for pageEnd := end; ; {
		page, err := fetch(pageEnd)
		if err != nil {
			return nil, err
		}
		oldest := pageEnd
		for i := range page {
			if page[i].Time.Before(oldest) {
				oldest = page[i].Time
			}
			if _, ok := seen[page[i].TradeID]; ok {
				continue
			}
			seen[page[i].TradeID] = struct{}{}
			resp = append(resp, page[i])
		}
		if len(page) < maximumMarketTradesLimit {
			return resp, nil
		}
		next := oldest.Truncate(time.Second).Add(time.Second)
		if !next.After(start) {
			return resp, nil
		}
		if !next.Before(pageEnd) {
			return nil, fmt.Errorf("%w: %d or more at %s", errTradesPageFull, maximumMarketTradesLimit, oldest.Truncate(time.Second).UTC().Format(time.RFC3339))
		}
		pageEnd = next
	}
}

// marketTrades returns an Advanced Trade product's trades as GCT records them, each its taker's side, as Coinbase
// reports the maker's
func (e *Exchange) marketTrades(trades []MarketTrade, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	resp := make([]trade.Data, len(trades))
	for i := range trades {
		side, err := takerSide(trades[i].Side)
		if err != nil {
			return nil, fmt.Errorf("trade %s: %w", trades[i].TradeID, err)
		}
		resp[i] = trade.Data{
			TID:          trades[i].TradeID,
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    a,
			Side:         side,
			Price:        trades[i].Price.Float64(),
			Amount:       trades[i].Size.Float64(),
			Timestamp:    trades[i].Time,
		}
	}
	return resp, nil
}

// historicDerivativesTrades returns a gateway instrument's trades between two times
func (e *Exchange) historicDerivativesTrades(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time) ([]trade.Data, error) {
	name, err := e.instrumentName(p, a)
	if err != nil {
		return nil, err
	}
	overWebsocket := e.derivativesOverWebsocket(ctx, false)
	trades, err := pageDerivativesTrades(start, func(pageStart time.Time) (*DerivativesLastTradesResponse, error) {
		return e.GlobalDerivatives().getLastTradesByInstrumentAndTime(ctx, overWebsocket, &DerivativesLastTradesByInstrumentAndTimeRequest{
			InstrumentName: name,
			StartTime:      pageStart,
			EndTime:        end,
			Count:          derivativesMarketDataMaximumCount,
			Sorting:        DerivativesSortingAscending,
		})
	})
	if err != nil {
		return nil, err
	}
	return e.derivativesTrades(trades, p, a)
}

// pageDerivativesTrades returns a gateway instrument's trades from a time, oldest first a page at a time; each page
// starts at the last trade of the last, repeating the trades they share that time with, which are kept once, and a page
// of trades all from its first time cannot reach the trades after them
func pageDerivativesTrades(start time.Time, fetch func(pageStart time.Time) (*DerivativesLastTradesResponse, error)) ([]DerivativesPublicTrade, error) {
	var resp []DerivativesPublicTrade
	seen := make(map[string]struct{})
	for {
		page, err := fetch(start)
		if err != nil {
			return nil, err
		}
		for i := range page.Trades {
			if _, ok := seen[page.Trades[i].TradeID]; ok {
				continue
			}
			seen[page.Trades[i].TradeID] = struct{}{}
			resp = append(resp, page.Trades[i])
		}
		if !page.HasMore || len(page.Trades) == 0 {
			return resp, nil
		}
		last := page.Trades[len(page.Trades)-1].Timestamp.Time()
		if !last.After(start) {
			return nil, fmt.Errorf("%w: %d or more at %s", errTradesPageFull, len(page.Trades), last.UTC().Format(time.RFC3339Nano))
		}
		start = last
	}
}

// derivativesTrades returns a gateway instrument's trades as GCT records them; a trade's direction is its taker's
func (e *Exchange) derivativesTrades(trades []DerivativesPublicTrade, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	resp := make([]trade.Data, len(trades))
	for i := range trades {
		side, err := order.StringToOrderSide(trades[i].Direction)
		if err != nil {
			return nil, fmt.Errorf("trade %s: %w", trades[i].TradeID, err)
		}
		resp[i] = trade.Data{
			TID:          trades[i].TradeID,
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    a,
			Side:         side,
			Price:        trades[i].Price,
			Amount:       trades[i].Amount,
			Timestamp:    trades[i].Timestamp.Time(),
		}
	}
	return resp, nil
}

// SubmitOrder places an order: an Advanced Trade order, with a client order ID of its own when the order has none, as
// Coinbase requires one, or a gateway order, through the gateway's websocket where the wrapper may use it
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	switch {
	case isAdvancedTradeAsset(s.AssetType):
		id, err := e.productID(s.Pair, s.AssetType)
		if err != nil {
			return nil, err
		}
		req, err := createOrderRequest(s, id)
		if err != nil {
			return nil, err
		}
		created, err := e.CreateOrder(ctx, req)
		if err != nil {
			return nil, err
		}
		if created.SuccessResponse == nil {
			return nil, common.ErrNoResponse
		}
		resp, err := s.DeriveSubmitResponse(created.SuccessResponse.OrderID)
		if err != nil {
			return nil, err
		}
		resp.ClientOrderID = req.ClientOrderID
		// The user channel reports no order's size
		e.orderSizes.set(resp.OrderID, s.Amount, s.QuoteAmount)
		// Coinbase's response says only that it accepted the order, so even a market order is new until its state is
		// read
		resp.Status = order.New
		if s.RetrieveFees {
			// The order is placed whether or not its state can be read, so its response goes back with the error
			placed, err := e.placedOrder(ctx, resp.OrderID, s.RetrieveFeeDelay)
			if err != nil {
				return resp, fmt.Errorf("order %s placed, but error retrieving its fees: %w", resp.OrderID, err)
			}
			if resp.Status, err = standardOrderStatus(placed.Status, placed.FilledSize.Float64()); err != nil {
				return resp, fmt.Errorf("order %s placed, but error reading its status: %w", resp.OrderID, err)
			}
			resp.Fee = placed.TotalFees.Float64()
		}
		return resp, nil
	case isDerivativesAsset(s.AssetType):
		return e.submitDerivativesOrder(ctx, s)
	}
	return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, s.AssetType)
}

// placedOrder returns an Advanced Trade order after a delay, as Coinbase's response to a new order holds neither its
// state nor its fees, and records a new order's fills a moment after placing it
func (e *Exchange) placedOrder(ctx context.Context, orderID string, delay time.Duration) (*Order, error) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}
	return e.GetOrder(ctx, orderID)
}

// createOrderRequest returns the Advanced Trade request placing an order. RFQDisabled turns smart order routing off,
// which keeps an order on the order book; a reduce only order is refused, as Advanced Trade orders cannot carry it
func createOrderRequest(s *order.Submit, productID string) (*CreateOrderRequest, error) {
	if s.ReduceOnly {
		return nil, fmt.Errorf("%w by Advanced Trade", errReduceOnlyNotSupported)
	}
	side, err := advancedTradeSide(s.Side)
	if err != nil {
		return nil, err
	}
	configuration, err := orderConfigurationRequest(s)
	if err != nil {
		return nil, err
	}
	marginType, err := orderMarginType(s.MarginType)
	if err != nil {
		return nil, err
	}
	clientOrderID := s.ClientOrderID
	if clientOrderID == "" {
		clientOrderID = uuid.NewV4().String()
	}
	req := &CreateOrderRequest{
		ClientOrderID:      clientOrderID,
		ProductID:          productID,
		Side:               side,
		OrderConfiguration: configuration,
		Leverage:           s.Leverage,
		MarginType:         marginType,
	}
	if s.RFQDisabled {
		req.SORPreference = SORPreferenceDisabled
	}
	return req, nil
}

// advancedTradeSide returns an order side as Advanced Trade names it
func advancedTradeSide(side order.Side) (string, error) {
	switch {
	case side.IsLong():
		return OrderSideBuy, nil
	case side.IsShort():
		return OrderSideSell, nil
	}
	return "", fmt.Errorf("%w: %s", order.ErrSideIsInvalid, side)
}

// orderConfigurationRequest returns the Advanced Trade configuration of an order: its type and time in force choose
// the configuration, a good till time or good till day order, or one with an end time, taking a good til date
// configuration
func orderConfigurationRequest(s *order.Submit) (OrderConfigurationRequest, error) {
	if err := validateAdvancedTradeTimeInForce(s); err != nil {
		return nil, err
	}
	tif := s.TimeInForce
	postOnly := tif.Is(order.PostOnly)
	untilDate := tif.Is(order.GoodTillTime) || tif.Is(order.GoodTillDay) || !s.EndTime.IsZero()
	switch s.Type {
	case order.Market:
		if tif.Is(order.FillOrKill) {
			return &MarketMarketFOKRequest{BaseSize: s.Amount, QuoteSize: s.QuoteAmount}, nil
		}
		return &MarketMarketIOCRequest{BaseSize: s.Amount, QuoteSize: s.QuoteAmount}, nil
	case order.Limit:
		switch {
		case tif.Is(order.FillOrKill):
			return &LimitLimitFOKRequest{BaseSize: s.Amount, QuoteSize: s.QuoteAmount, LimitPrice: s.Price}, nil
		case tif.Is(order.ImmediateOrCancel):
			return &SORLimitIOCRequest{BaseSize: s.Amount, QuoteSize: s.QuoteAmount, LimitPrice: s.Price}, nil
		case untilDate:
			return &LimitLimitGTDRequest{BaseSize: s.Amount, QuoteSize: s.QuoteAmount, LimitPrice: s.Price, EndTime: s.EndTime, PostOnly: postOnly}, nil
		}
		return &LimitLimitGTCRequest{BaseSize: s.Amount, QuoteSize: s.QuoteAmount, LimitPrice: s.Price, PostOnly: postOnly}, nil
	case order.StopLimit:
		direction := StopDirectionStopDown
		if s.StopDirection == order.StopUp {
			direction = StopDirectionStopUp
		}
		if untilDate {
			return &StopLimitStopLimitGTDRequest{BaseSize: s.Amount, LimitPrice: s.Price, StopPrice: s.TriggerPrice, EndTime: s.EndTime, StopDirection: direction}, nil
		}
		return &StopLimitStopLimitGTCRequest{BaseSize: s.Amount, LimitPrice: s.Price, StopPrice: s.TriggerPrice, StopDirection: direction}, nil
	case order.Bracket:
		if untilDate {
			return &TriggerBracketGTDRequest{BaseSize: s.Amount, LimitPrice: s.Price, StopTriggerPrice: s.TriggerPrice, EndTime: s.EndTime}, nil
		}
		return &TriggerBracketGTCRequest{BaseSize: s.Amount, LimitPrice: s.Price, StopTriggerPrice: s.TriggerPrice}, nil
	}
	return nil, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, s.Type)
}

// validateAdvancedTradeTimeInForce refuses a time in force Advanced Trade has no configuration for, rather than let the
// order take another: a good till date order, which a good till time or good till day order is, runs to its end time;
// a market order fills at once or is killed; a stop limit or bracket order rests; and only a resting limit order can be
// post only
func validateAdvancedTradeTimeInForce(s *order.Submit) error {
	tif := s.TimeInForce
	untilDate := tif.Is(order.GoodTillTime) || tif.Is(order.GoodTillDay)
	if untilDate && s.EndTime.IsZero() {
		return fmt.Errorf("%w: a %s order runs to its end time", common.ErrDateUnset, tif)
	}
	untilDate = untilDate || !s.EndTime.IsZero()
	immediate := tif.Is(order.ImmediateOrCancel) || tif.Is(order.FillOrKill)
	resting := tif.Is(order.GoodTillCancel) || untilDate
	var supported bool
	switch {
	case tif.Is(order.GoodTillCrossing), tif.Is(order.StopOrReduce), tif.Is(order.ImmediateOrCancel | order.FillOrKill),
		tif.Is(order.GoodTillCancel) && untilDate, immediate && (resting || tif.Is(order.PostOnly)):
	case s.Type == order.Market:
		supported = !resting && !tif.Is(order.PostOnly)
	case s.Type == order.StopLimit, s.Type == order.Bracket:
		supported = !immediate && !tif.Is(order.PostOnly)
	default:
		supported = true
	}
	if !supported {
		return fmt.Errorf("%w: %s for a %s order", order.ErrUnsupportedTimeInForce, tif, s.Type)
	}
	return nil
}

// orderMarginType returns a margin type as Advanced Trade names it, or nothing for Coinbase's default of cross margin
func orderMarginType(t margin.Type) (string, error) {
	switch t {
	case margin.Unset:
		return "", nil
	case margin.Multi:
		return OrderMarginTypeCross, nil
	case margin.Isolated:
		return OrderMarginTypeIsolated, nil
	}
	return "", fmt.Errorf("%w: %s", margin.ErrMarginTypeUnsupported, t)
}

// marginTypeFromOrder returns the margin type an Advanced Trade order reports
func marginTypeFromOrder(marginType string) margin.Type {
	switch marginType {
	case OrderMarginTypeCross:
		return margin.Multi
	case OrderMarginTypeIsolated:
		return margin.Isolated
	}
	return margin.Unset
}

// submitDerivativesOrder places a gateway order, which reports its state and the trades it made at once
func (e *Exchange) submitDerivativesOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	name, err := e.instrumentName(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	req, err := derivativesOrderRequest(s, name)
	if err != nil {
		return nil, err
	}
	method := derivativesBuyMethod
	if s.Side.IsShort() {
		method = derivativesSellMethod
	}
	placed, err := e.GlobalDerivatives().placeOrder(ctx, e.derivativesOverWebsocket(ctx, true), method, req)
	if err != nil {
		return nil, err
	}
	detail, err := e.derivativesOrderDetail(&placed.Order)
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(placed.Order.OrderID)
	if err != nil {
		return nil, err
	}
	resp.Status = detail.Status
	resp.Price = detail.Price
	// The gateway can reduce a reduce only order to the position it closes
	resp.Amount = detail.Amount
	resp.AverageExecutedPrice = detail.AverageExecutedPrice
	resp.RemainingAmount = detail.RemainingAmount
	resp.Date = detail.Date
	resp.LastUpdated = detail.LastUpdated
	resp.Trades = make([]order.TradeHistory, len(placed.Trades))
	for i := range placed.Trades {
		if resp.Trades[i], err = e.derivativesTradeHistory(&placed.Trades[i]); err != nil {
			return nil, err
		}
		resp.Fee += placed.Trades[i].Fee
		resp.FeeAsset = placed.Trades[i].FeeCurrency
	}
	return resp, nil
}

// derivativesOrderRequest returns the gateway request placing an order. A post only order is rejected rather than
// repriced when it would take liquidity, and isolated margin places it in the isolated margin subaccount the gateway
// binds to the instrument; leverage is the instrument's, which SetLeverage sets, so an order asking for leverage other
// than 1 is refused
func derivativesOrderRequest(s *order.Submit, instrumentName string) (*DerivativesOrderRequest, error) {
	if s.Leverage != 0 && s.Leverage != 1 {
		return nil, fmt.Errorf("%w: %v, as the gateway takes an instrument's leverage from SetLeverage", order.ErrSubmitLeverageNotSupported, s.Leverage)
	}
	if !s.Side.IsLong() && !s.Side.IsShort() {
		return nil, fmt.Errorf("%w: %s", order.ErrSideIsInvalid, s.Side)
	}
	orderType, err := derivativesOrderTypeParam(s.Type)
	if err != nil {
		return nil, err
	}
	timeInForce, err := derivativesTimeInForceParam(s.TimeInForce)
	if err != nil {
		return nil, err
	}
	if !s.EndTime.IsZero() {
		return nil, fmt.Errorf("%w: the gateway's orders run to no end time, a good till day order to its daily settlement", order.ErrUnsupportedTimeInForce)
	}
	isolated, err := derivativesIsolated(s.MarginType)
	if err != nil {
		return nil, err
	}
	postOnly := s.TimeInForce.Is(order.PostOnly)
	req := &DerivativesOrderRequest{
		InstrumentName: instrumentName,
		Amount:         s.Amount,
		Type:           orderType,
		Label:          s.ClientOrderID,
		TimeInForce:    timeInForce,
		PostOnly:       postOnly,
		RejectPostOnly: postOnly,
		ReduceOnly:     s.ReduceOnly,
		Isolated:       isolated,
	}
	switch s.Type {
	case order.Limit, order.StopLimit, order.TakeProfit:
		req.Price = s.Price
	}
	switch s.Type {
	case order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitMarket, order.TrailingStop:
		if req.Trigger, err = derivativesTriggerParam(s.TriggerPriceType); err != nil {
			return nil, err
		}
		req.TriggerPrice = s.TriggerPrice
	}
	if s.Type == order.TrailingStop {
		if s.TrackingMode != order.Distance || s.TrackingValue <= 0 {
			return nil, errTrailingStopDistance
		}
		req.TriggerOffset = s.TrackingValue
	}
	return req, nil
}

// derivativesOrderTypeParam returns an order type as the gateway names it
func derivativesOrderTypeParam(t order.Type) (string, error) {
	switch t {
	case order.Limit:
		return DerivativesOrderTypeLimit, nil
	case order.Market:
		return DerivativesOrderTypeMarket, nil
	case order.StopLimit:
		return DerivativesOrderTypeStopLimit, nil
	case order.StopMarket:
		return DerivativesOrderTypeStopMarket, nil
	case order.TakeProfit:
		return DerivativesOrderTypeTakeLimit, nil
	case order.TakeProfitMarket:
		return DerivativesOrderTypeTakeMarket, nil
	case order.TrailingStop:
		return DerivativesOrderTypeTrailingStop, nil
	}
	return "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
}

// derivativesTimeInForceParam returns a time in force as the gateway names it, good till cancelled when it is unset;
// post only is a separate param
func derivativesTimeInForceParam(tif order.TimeInForce) (string, error) {
	switch tif &^ order.PostOnly {
	case order.UnknownTIF:
		return "", nil
	case order.GoodTillCancel:
		return DerivativesTimeInForceGoodTilCancelled, nil
	case order.GoodTillDay:
		return DerivativesTimeInForceGoodTilDay, nil
	case order.FillOrKill:
		return DerivativesTimeInForceFillOrKill, nil
	case order.ImmediateOrCancel:
		return DerivativesTimeInForceImmediateOrCancel, nil
	}
	return "", fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, tif)
}

// derivativesTriggerParam returns the price a trigger order triggers on; the zero price type is the last price
func derivativesTriggerParam(priceType order.PriceType) (string, error) {
	switch priceType {
	case order.LastPrice:
		return DerivativesTriggerLastPrice, nil
	case order.IndexPrice:
		return DerivativesTriggerIndexPrice, nil
	case order.MarkPrice:
		return DerivativesTriggerMarkPrice, nil
	}
	return "", fmt.Errorf("%w: %s", order.ErrUnknownPriceType, priceType)
}

// derivativesIsolated reports whether a margin type places a gateway order under isolated margin; the gateway applies
// cross margin otherwise
func derivativesIsolated(t margin.Type) (bool, error) {
	switch t {
	case margin.Unset, margin.Multi:
		return false, nil
	case margin.Isolated:
		return true, nil
	}
	return false, fmt.Errorf("%w: %s", margin.ErrMarginTypeUnsupported, t)
}

// derivativesTradeHistory returns one of the account's gateway trades as an order's trade
func (e *Exchange) derivativesTradeHistory(t *DerivativesUserTrade) (order.TradeHistory, error) {
	side, err := order.StringToOrderSide(t.Direction)
	if err != nil {
		return order.TradeHistory{}, fmt.Errorf("trade %s: %w", t.TradeID, err)
	}
	orderType, err := derivativesOrderType(t.OrderType)
	if err != nil {
		return order.TradeHistory{}, fmt.Errorf("trade %s: %w", t.TradeID, err)
	}
	return order.TradeHistory{
		Price:     t.Price,
		Amount:    t.Amount,
		Fee:       t.Fee,
		FeeAsset:  t.FeeCurrency.String(),
		Exchange:  e.Name,
		TID:       t.TradeID,
		Type:      orderType,
		Side:      side,
		Timestamp: t.Timestamp.Time(),
		IsMaker:   t.Liquidity == derivativesLiquidityMaker,
	}, nil
}

// ModifyOrder changes an open order's price and amount: an Advanced Trade limit order, good till cancelled, whose
// price and size Coinbase requires together, so the order's own fill in for those left unset; or a gateway order,
// identified by its ID or its label in the pair, whose amount the gateway requires with every edit. A time in force
// the edit cannot apply is refused: Advanced Trade's edits change price and size alone, and the gateway's can make an
// order post only or not
func (e *Exchange) ModifyOrder(ctx context.Context, m *order.Modify) (*order.ModifyResponse, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	switch {
	case isAdvancedTradeAsset(m.AssetType):
		if m.OrderID == "" {
			return nil, errOrderIDRequired
		}
		price, size, tif := m.Price, m.Amount, m.TimeInForce
		if price == 0 || size == 0 || tif != order.UnknownTIF {
			existing, err := e.GetOrder(ctx, m.OrderID)
			if err != nil {
				return nil, err
			}
			values := orderConfigurationValues(existing.OrderConfiguration)
			if price == 0 {
				price = values.limitPrice
			}
			if size == 0 {
				size = values.baseSize
			}
			// Coinbase edits an order's price and size alone
			existingTIF, err := standardTimeInForce(existing.TimeInForce, values.postOnly)
			if err != nil {
				return nil, err
			}
			if tif != order.UnknownTIF && tif != existingTIF {
				return nil, fmt.Errorf("%w: %s, as Advanced Trade edits an order's price and size alone", order.ErrUnsupportedTimeInForce, tif)
			}
			tif = existingTIF
		}
		if _, err := e.EditOrder(ctx, &EditOrderRequest{OrderID: m.OrderID, Price: price, Size: size, StopPrice: m.TriggerPrice}); err != nil {
			return nil, err
		}
		resp, err := m.DeriveModifyResponse()
		if err != nil {
			return nil, err
		}
		resp.Price, resp.Amount, resp.TimeInForce = price, size, tif
		return resp, nil
	case isDerivativesAsset(m.AssetType):
		return e.modifyDerivativesOrder(ctx, m)
	}
	return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, m.AssetType)
}

// modifyDerivativesOrder edits a gateway order, by its ID, or else by its label in the pair, in the main account or the
// isolated margin subaccount holding it. The gateway takes an edit's flags afresh, post only defaulting to true and
// reduce only to false, so the order is read first: the amount, price and trigger price the modification leaves unset
// are kept, as are its reduce only flag and, unless the modification sets a time in force, its post only flags
func (e *Exchange) modifyDerivativesOrder(ctx context.Context, m *order.Modify) (*order.ModifyResponse, error) {
	overWebsocket := e.derivativesOverWebsocket(ctx, true)
	g := e.GlobalDerivatives()
	var existing *DerivativesOrder
	var isolated bool
	if m.OrderID != "" {
		var err error
		if existing, err = derivativesOrderLookup(false, func(lookupIsolated bool) (*DerivativesOrder, error) {
			isolated = lookupIsolated
			return g.getOrderState(ctx, overWebsocket, &DerivativesOrderStateRequest{OrderID: m.OrderID, Isolated: lookupIsolated})
		}); err != nil {
			return nil, err
		}
	} else {
		name, err := e.instrumentName(m.Pair, m.AssetType)
		if err != nil {
			return nil, err
		}
		// The open orders by label leave out the isolated margin subaccounts', so the order is found among every open
		// order
		open, err := g.getOpenOrders(ctx, overWebsocket, &DerivativesOpenOrdersRequest{Kind: DerivativesKindFuture, IncludeIsolated: true})
		if err != nil {
			return nil, err
		}
		open = slices.DeleteFunc(open, func(o DerivativesOrder) bool { return o.Label != m.ClientOrderID || o.InstrumentName != name })
		switch len(open) {
		case 0:
			return nil, fmt.Errorf("%w: no open order labelled %q in %s", order.ErrOrderNotFound, m.ClientOrderID, name)
		case 1:
			existing, isolated = &open[0], open[0].Isolated
		default:
			return nil, fmt.Errorf("%w: %d orders labelled %q in %s", errOrderLabelNotUnique, len(open), m.ClientOrderID, name)
		}
	}
	amount, price, triggerPrice := m.Amount, m.Price, m.TriggerPrice
	if amount == 0 {
		amount = existing.Amount
	}
	if price == 0 && !existing.Price.Market {
		price = existing.Price.Value
	}
	if triggerPrice == 0 && existing.OrderType != DerivativesOrderTypeTrailingStop {
		triggerPrice = existing.TriggerPrice
	}
	postOnly, rejectPostOnly := existing.PostOnly, existing.RejectPostOnly
	if m.TimeInForce != order.UnknownTIF {
		// An edit can make an order post only or not, but not change how long it rests
		existingTIF, err := order.StringToTimeInForce(existing.TimeInForce)
		if err != nil {
			return nil, err
		}
		if lifetime := m.TimeInForce &^ order.PostOnly; lifetime != order.UnknownTIF && lifetime != existingTIF {
			return nil, fmt.Errorf("%w: %s, as a gateway edit changes an order's post only flag but not its time in force", order.ErrUnsupportedTimeInForce, m.TimeInForce)
		}
		postOnly = m.TimeInForce.Is(order.PostOnly)
		rejectPostOnly = postOnly
	}
	edited, err := derivativesOrderLookup(isolated, func(lookupIsolated bool) (*DerivativesOrderResponse, error) {
		if m.OrderID != "" {
			return g.editOrder(ctx, overWebsocket, &DerivativesEditOrderRequest{
				OrderID:        m.OrderID,
				Amount:         amount,
				Price:          price,
				PostOnly:       postOnly,
				ReduceOnly:     existing.ReduceOnly,
				RejectPostOnly: rejectPostOnly,
				TriggerPrice:   triggerPrice,
				TriggerOffset:  existing.TriggerOffset,
				Isolated:       lookupIsolated,
			})
		}
		return g.editOrderByLabel(ctx, overWebsocket, &DerivativesEditOrderByLabelRequest{
			Label:          m.ClientOrderID,
			InstrumentName: existing.InstrumentName,
			Amount:         amount,
			Price:          price,
			PostOnly:       postOnly,
			ReduceOnly:     existing.ReduceOnly,
			RejectPostOnly: rejectPostOnly,
			TriggerPrice:   triggerPrice,
			Isolated:       lookupIsolated,
		})
	})
	if err != nil {
		return nil, err
	}
	detail, err := e.derivativesOrderDetail(&edited.Order)
	if err != nil {
		return nil, err
	}
	resp, err := m.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.OrderID = detail.OrderID
	resp.ClientOrderID = detail.ClientOrderID
	resp.Type = detail.Type
	resp.Side = detail.Side
	resp.Status = detail.Status
	resp.TimeInForce = detail.TimeInForce
	resp.Price = detail.Price
	resp.Amount = detail.Amount
	resp.TriggerPrice = detail.TriggerPrice
	resp.RemainingAmount = detail.RemainingAmount
	resp.Date = detail.Date
	resp.LastUpdated = detail.LastUpdated
	return resp, nil
}

// derivativesOrderLookup calls a gateway method naming an order by its ID in one scope, the main account or the
// isolated margin subaccounts, and then the other when the gateway does not find the order, as the gateway searches
// only the scope a request names
func derivativesOrderLookup[T any](isolatedFirst bool, call func(isolated bool) (T, error)) (T, error) {
	resp, err := call(isolatedFirst)
	if !isDerivativesOrderNotFound(err) {
		return resp, err
	}
	if resp, err = call(!isolatedFirst); isDerivativesOrderNotFound(err) {
		return resp, fmt.Errorf("%w: %w", order.ErrOrderNotFound, err)
	}
	return resp, err
}

// isDerivativesOrderNotFound reports whether the gateway did not find an order in the scope a request named
func isDerivativesOrderNotFound(err error) bool {
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	return ok && (derivativesErr.Message == derivativesOrderNotFound || derivativesErr.Message == derivativesNotOwnerOfOrder)
}

// CancelOrder submits an order for cancellation: an Advanced Trade order, which cancels asynchronously, or a gateway
// order, in the main account or the isolated margin subaccount holding it
func (e *Exchange) CancelOrder(ctx context.Context, c *order.Cancel) error {
	if err := c.Validate(c.StandardCancel()); err != nil {
		return err
	}
	switch {
	case isAdvancedTradeAsset(c.AssetType):
		_, err := e.CancelOrders(ctx, []string{c.OrderID})
		return err
	case isDerivativesAsset(c.AssetType):
		_, err := e.cancelDerivativesOrder(ctx, c.OrderID, c.MarginType == margin.Isolated, e.derivativesOverWebsocket(ctx, true))
		return err
	}
	return fmt.Errorf("%w: %s", asset.ErrNotSupported, c.AssetType)
}

// cancelDerivativesOrder cancels a gateway order by its ID, looking in the isolated margin subaccounts first when
// asked
func (e *Exchange) cancelDerivativesOrder(ctx context.Context, orderID string, isolatedFirst, overWebsocket bool) (*DerivativesOrder, error) {
	return derivativesOrderLookup(isolatedFirst, func(isolated bool) (*DerivativesOrder, error) {
		return e.GlobalDerivatives().cancelOrderByID(ctx, overWebsocket, &DerivativesCancelOrderByIDRequest{OrderID: orderID, Isolated: isolated})
	})
}

// CancelBatchOrders submits orders for cancellation: Advanced Trade orders in batches, each reported pending
// cancellation once Coinbase submits its cancel request, which completes asynchronously, and gateway orders one request
// each, each reported in its state once cancelled. An order that could not be cancelled reports the error
func (e *Exchange) CancelBatchOrders(ctx context.Context, cancels []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(cancels) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	var advancedTradeIDs []string
	for i := range cancels {
		if err := cancels[i].Validate(cancels[i].StandardCancel()); err != nil {
			return nil, err
		}
		switch {
		case isAdvancedTradeAsset(cancels[i].AssetType):
			advancedTradeIDs = append(advancedTradeIDs, cancels[i].OrderID)
		case isDerivativesAsset(cancels[i].AssetType):
		default:
			return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, cancels[i].AssetType)
		}
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string, len(cancels))}
	if err := e.cancelAdvancedTradeOrders(ctx, advancedTradeIDs, resp.Status); err != nil {
		return nil, err
	}
	overWebsocket := e.derivativesOverWebsocket(ctx, true)
	for i := range cancels {
		if !isDerivativesAsset(cancels[i].AssetType) {
			continue
		}
		cancelled, err := e.cancelDerivativesOrder(ctx, cancels[i].OrderID, cancels[i].MarginType == margin.Isolated, overWebsocket)
		if err != nil {
			resp.Status[cancels[i].OrderID] = err.Error()
			continue
		}
		resp.Status[cancels[i].OrderID] = derivativesOrderStatusString(cancelled)
	}
	return resp, nil
}

// cancelAdvancedTradeOrders submits Advanced Trade orders for cancellation in batches, recording each order's status:
// pending cancellation once Coinbase submits its cancel request, which completes asynchronously, or why it did not
func (e *Exchange) cancelAdvancedTradeOrders(ctx context.Context, orderIDs []string, statuses map[string]string) error {
	for batch := range slices.Chunk(orderIDs, cancelOrdersBatchSize) {
		resp, err := e.CancelOrders(ctx, batch)
		if resp == nil {
			return err
		}
		for i := range resp.Results {
			if err := resp.Results[i].Err(); err != nil {
				statuses[resp.Results[i].OrderID] = err.Error()
				continue
			}
			statuses[resp.Results[i].OrderID] = order.PendingCancel.String()
		}
	}
	return nil
}

// derivativesOrderStatusString returns a gateway order's status as GCT names it, or its state when GCT has no name
// for it
func derivativesOrderStatusString(o *DerivativesOrder) string {
	status, err := derivativesOrderStatus(o.OrderState, o.FilledAmount)
	if err != nil {
		return o.OrderState
	}
	return status.String()
}

// CancelAllOrders submits the account's open orders of an asset for cancellation, of a pair when one is given and of
// the request's type, side, margin type and time in force when they are set. Advanced Trade orders are listed and
// cancelled in batches, each reported pending cancellation. The gateway cancels an instrument's orders together, in the
// main account and the isolated margin subaccounts, by categories of order type, so a request it cannot express, by
// side, margin type, time in force or a type narrower than its categories, has its orders listed and cancelled one at a
// time
func (e *Exchange) CancelAllOrders(ctx context.Context, c *order.Cancel) (order.CancelAllResponse, error) {
	resp := order.CancelAllResponse{Status: make(map[string]string)}
	if err := common.NilGuard(c); err != nil {
		return resp, err
	}
	req := &order.MultiOrderRequest{
		AssetType:   c.AssetType,
		Pairs:       pairsFor(c.Pair),
		Type:        anyTypeFor(c.Type),
		Side:        anySideFor(c.Side),
		MarginType:  c.MarginType,
		TimeInForce: c.TimeInForce,
	}
	switch {
	case isAdvancedTradeAsset(c.AssetType):
		open, err := e.GetActiveOrders(ctx, req)
		if err != nil {
			return resp, err
		}
		ids := make([]string, len(open))
		for i := range open {
			ids[i] = open[i].OrderID
		}
		return resp, e.cancelAdvancedTradeOrders(ctx, ids, resp.Status)
	case isDerivativesAsset(c.AssetType):
		cancelType, bulk, err := derivativesCancelType(c.Type)
		if err != nil {
			return resp, err
		}
		overWebsocket := e.derivativesOverWebsocket(ctx, true)
		if !bulk || req.Side != order.AnySide || c.MarginType != margin.Unset || c.TimeInForce != order.UnknownTIF {
			// The type is matched here, as the gateway's stop orders are stop limit or stop market orders
			req.Type = order.AnyType
			open, err := e.GetActiveOrders(ctx, req)
			if err != nil {
				return resp, err
			}
			for i := range open {
				if !derivativesCancelTypeMatches(c.Type, open[i].Type) {
					continue
				}
				cancelled, err := e.cancelDerivativesOrder(ctx, open[i].OrderID, open[i].MarginType == margin.Isolated, overWebsocket)
				if err != nil {
					resp.Status[open[i].OrderID] = err.Error()
					continue
				}
				resp.Status[open[i].OrderID] = derivativesOrderStatusString(cancelled)
			}
			return resp, nil
		}
		var instruments []string
		if c.Pair.IsEmpty() {
			req.Type = order.AnyType
			open, err := e.GetActiveOrders(ctx, req)
			if err != nil {
				return resp, err
			}
			for i := range open {
				name, err := e.instrumentName(open[i].Pair, c.AssetType)
				if err != nil {
					return resp, err
				}
				if !slices.Contains(instruments, name) {
					instruments = append(instruments, name)
				}
			}
		} else {
			name, err := e.instrumentName(c.Pair, c.AssetType)
			if err != nil {
				return resp, err
			}
			instruments = []string{name}
		}
		for _, name := range instruments {
			cancelled, err := e.GlobalDerivatives().cancelAllByInstrument(ctx, overWebsocket, &DerivativesCancelAllByInstrumentRequest{InstrumentName: name, Type: cancelType, Detailed: true, IncludeIsolated: true})
			if err != nil {
				return resp, err
			}
			for i := range cancelled.ExecutionReports {
				for j := range cancelled.ExecutionReports[i].Result {
					o := &cancelled.ExecutionReports[i].Result[j]
					resp.Status[o.OrderID] = derivativesOrderStatusString(o)
				}
			}
		}
		return resp, nil
	}
	return resp, fmt.Errorf("%w: %s", asset.ErrNotSupported, c.AssetType)
}

// pairsFor returns a pair as a list of the pairs a request selects, none when it is empty
func pairsFor(p currency.Pair) currency.Pairs {
	if p.IsEmpty() {
		return nil
	}
	return currency.Pairs{p}
}

// anyTypeFor returns an order type a request filters by, any type when it is unset
func anyTypeFor(t order.Type) order.Type {
	if t == order.UnknownType {
		return order.AnyType
	}
	return t
}

// anySideFor returns an order side a request filters by, any side when it is unset
func anySideFor(s order.Side) order.Side {
	if s == order.UnknownSide {
		return order.AnySide
	}
	return s
}

// derivativesCancelType returns the category of order types a gateway cancel all request cancels for an order type,
// and whether the category is exactly that type: stop orders are the stop limit and stop market orders, and take
// profit orders the take limit and take market orders. Market orders never rest
func derivativesCancelType(t order.Type) (category string, exact bool, err error) {
	switch t {
	case order.AnyType, order.UnknownType:
		return DerivativesCancelOrderTypeAll, true, nil
	case order.Limit:
		return DerivativesCancelOrderTypeLimit, true, nil
	case order.Stop:
		return DerivativesCancelOrderTypeStop, true, nil
	case order.StopLimit, order.StopMarket:
		return DerivativesCancelOrderTypeStop, false, nil
	case order.TakeProfit, order.TakeProfitMarket:
		return DerivativesCancelOrderTypeTake, false, nil
	case order.TrailingStop:
		return DerivativesCancelOrderTypeTrailingStop, true, nil
	}
	return "", false, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
}

// derivativesCancelTypeMatches reports whether an open gateway order is of the type a cancel all request names, stop
// naming the stop limit and stop market orders
func derivativesCancelTypeMatches(t, orderType order.Type) bool {
	switch t {
	case order.AnyType, order.UnknownType:
		return true
	case order.Stop:
		return orderType == order.StopLimit || orderType == order.StopMarket
	}
	return orderType == t
}

// GetOrderInfo returns an order with the trades it made: an Advanced Trade order with its fills, or a gateway order,
// from the main account or the isolated margin subaccount holding it, with its trades. The gateway reports an order's
// state until 30 minutes after it closes, so an older order is found in its pair's order history
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, p currency.Pair, a asset.Item) (*order.Detail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var detail *order.Detail
	switch {
	case isAdvancedTradeAsset(a):
		listed := time.Now()
		o, err := e.GetOrder(ctx, orderID)
		if err != nil {
			return nil, err
		}
		if detail, err = e.orderDetail(o, listed); err != nil {
			return nil, err
		}
		if detail.Trades, err = e.orderTrades(ctx, detail); err != nil {
			return nil, err
		}
	case isDerivativesAsset(a):
		overWebsocket := e.derivativesOverWebsocket(ctx, true)
		g := e.GlobalDerivatives()
		var isolated bool
		o, err := derivativesOrderLookup(false, func(lookupIsolated bool) (*DerivativesOrder, error) {
			isolated = lookupIsolated
			return g.getOrderState(ctx, overWebsocket, &DerivativesOrderStateRequest{OrderID: orderID, Isolated: lookupIsolated})
		})
		if errors.Is(err, order.ErrOrderNotFound) {
			if o, err = e.derivativesClosedOrder(ctx, orderID, p, a); err == nil {
				isolated = o.Isolated
			}
		}
		if err != nil {
			return nil, err
		}
		if detail, err = e.derivativesOrderDetail(o); err != nil {
			return nil, err
		}
		trades, err := e.derivativesOrderTrades(ctx, overWebsocket, orderID, isolated)
		if err != nil {
			return nil, err
		}
		detail.Trades = make([]order.TradeHistory, len(trades))
		for i := range trades {
			if detail.Trades[i], err = e.derivativesTradeHistory(&trades[i]); err != nil {
				return nil, err
			}
			detail.Fee += trades[i].Fee
			detail.FeeAsset = trades[i].FeeCurrency
		}
		if isolated {
			detail.MarginType = margin.Isolated
		}
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if detail.AssetType != a || (!p.IsEmpty() && !detail.Pair.Equal(p)) {
		return nil, fmt.Errorf("%w: %s is a %s %s order", order.ErrOrderNotFound, orderID, detail.AssetType, detail.Pair)
	}
	return detail, nil
}

// derivativesOrderTrades returns a gateway order's trades, oldest first, from the recent store, which keeps a trade for
// 24 hours, and the historical store, which indexes it after a short delay
func (e *Exchange) derivativesOrderTrades(ctx context.Context, overWebsocket bool, orderID string, isolated bool) ([]DerivativesUserTrade, error) {
	byID := make(map[string]DerivativesUserTrade)
	for _, historical := range []bool{false, true} {
		trades, err := e.GlobalDerivatives().getUserTradesByOrder(ctx, overWebsocket, &DerivativesUserTradesByOrderRequest{OrderID: orderID, Sorting: DerivativesSortingAscending, Historical: historical, Isolated: isolated})
		if err != nil {
			return nil, err
		}
		for i := range trades {
			byID[trades[i].TradeID] = trades[i]
		}
	}
	trades := slices.Collect(maps.Values(byID))
	slices.SortFunc(trades, func(a, b DerivativesUserTrade) int {
		return cmp.Or(a.Timestamp.Time().Compare(b.Timestamp.Time()), cmp.Compare(a.TradeSequence, b.TradeSequence), strings.Compare(a.TradeID, b.TradeID))
	})
	return trades, nil
}

// orderConfiguration holds the sizes, prices, end time and post only flag of an Advanced Trade order's configuration
type orderConfiguration struct {
	baseSize, quoteSize, limitPrice, stopPrice float64
	endTime                                    time.Time
	postOnly                                   bool
}

// orderConfigurationValues returns the sizes, prices and post only flag of an Advanced Trade order's configuration, of
// whichever configuration it has
func orderConfigurationValues(c *OrderConfiguration) orderConfiguration {
	switch {
	case c == nil:
	case c.MarketMarketIOC != nil:
		return orderConfiguration{baseSize: c.MarketMarketIOC.BaseSize.Float64(), quoteSize: c.MarketMarketIOC.QuoteSize.Float64()}
	case c.MarketMarketFOK != nil:
		return orderConfiguration{baseSize: c.MarketMarketFOK.BaseSize.Float64(), quoteSize: c.MarketMarketFOK.QuoteSize.Float64()}
	case c.SORLimitIOC != nil:
		return orderConfiguration{baseSize: c.SORLimitIOC.BaseSize.Float64(), quoteSize: c.SORLimitIOC.QuoteSize.Float64(), limitPrice: c.SORLimitIOC.LimitPrice.Float64()}
	case c.LimitLimitGTC != nil:
		return orderConfiguration{baseSize: c.LimitLimitGTC.BaseSize.Float64(), quoteSize: c.LimitLimitGTC.QuoteSize.Float64(), limitPrice: c.LimitLimitGTC.LimitPrice.Float64(), postOnly: c.LimitLimitGTC.PostOnly}
	case c.LimitLimitGTD != nil:
		return orderConfiguration{baseSize: c.LimitLimitGTD.BaseSize.Float64(), quoteSize: c.LimitLimitGTD.QuoteSize.Float64(), limitPrice: c.LimitLimitGTD.LimitPrice.Float64(), endTime: c.LimitLimitGTD.EndTime, postOnly: c.LimitLimitGTD.PostOnly}
	case c.LimitLimitFOK != nil:
		return orderConfiguration{baseSize: c.LimitLimitFOK.BaseSize.Float64(), quoteSize: c.LimitLimitFOK.QuoteSize.Float64(), limitPrice: c.LimitLimitFOK.LimitPrice.Float64()}
	case c.TWAPLimitGTD != nil:
		return orderConfiguration{baseSize: c.TWAPLimitGTD.BaseSize.Float64(), quoteSize: c.TWAPLimitGTD.QuoteSize.Float64(), limitPrice: c.TWAPLimitGTD.LimitPrice.Float64(), endTime: c.TWAPLimitGTD.EndTime}
	case c.StopLimitStopLimitGTC != nil:
		return orderConfiguration{baseSize: c.StopLimitStopLimitGTC.BaseSize.Float64(), limitPrice: c.StopLimitStopLimitGTC.LimitPrice.Float64(), stopPrice: c.StopLimitStopLimitGTC.StopPrice.Float64()}
	case c.StopLimitStopLimitGTD != nil:
		return orderConfiguration{baseSize: c.StopLimitStopLimitGTD.BaseSize.Float64(), limitPrice: c.StopLimitStopLimitGTD.LimitPrice.Float64(), stopPrice: c.StopLimitStopLimitGTD.StopPrice.Float64(), endTime: c.StopLimitStopLimitGTD.EndTime}
	case c.TriggerBracketGTC != nil:
		return orderConfiguration{baseSize: c.TriggerBracketGTC.BaseSize.Float64(), limitPrice: c.TriggerBracketGTC.LimitPrice.Float64(), stopPrice: c.TriggerBracketGTC.StopTriggerPrice.Float64()}
	case c.TriggerBracketGTD != nil:
		return orderConfiguration{baseSize: c.TriggerBracketGTD.BaseSize.Float64(), limitPrice: c.TriggerBracketGTD.LimitPrice.Float64(), stopPrice: c.TriggerBracketGTD.StopTriggerPrice.Float64(), endTime: c.TriggerBracketGTD.EndTime}
	case c.ScaledLimitGTC != nil:
		return orderConfiguration{baseSize: c.ScaledLimitGTC.BaseSize.Float64(), quoteSize: c.ScaledLimitGTC.QuoteSize.Float64()}
	}
	return orderConfiguration{}
}

// orderDetail returns an Advanced Trade order as GCT records it, from a request begun at listed; an order sized in the
// quote currency has no remaining base amount to report
func (e *Exchange) orderDetail(o *Order, listed time.Time) (*order.Detail, error) {
	a, err := productTypeAsset(o.ProductType)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	pair, err := e.productPair(o.ProductID, a)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	orderType, err := standardOrderType(o.OrderType)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	status, err := standardOrderStatus(o.Status, o.FilledSize.Float64())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	values := orderConfigurationValues(o.OrderConfiguration)
	tif, err := standardTimeInForce(o.TimeInForce, values.postOnly)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	d := &order.Detail{
		Exchange:             e.Name,
		OrderID:              o.OrderID,
		ClientOrderID:        o.ClientOrderID,
		AccountID:            o.RetailPortfolioID,
		Pair:                 pair,
		AssetType:            a,
		Side:                 side,
		Type:                 orderType,
		Status:               status,
		TimeInForce:          tif,
		Price:                values.limitPrice,
		TriggerPrice:         values.stopPrice,
		Amount:               values.baseSize,
		QuoteAmount:          values.quoteSize,
		ExecutedAmount:       o.FilledSize.Float64(),
		AverageExecutedPrice: o.AverageFilledPrice.Float64(),
		Cost:                 o.FilledValue.Float64(),
		Fee:                  o.TotalFees.Float64(),
		Leverage:             o.Leverage.Float64(),
		MarginType:           marginTypeFromOrder(o.MarginType),
		Date:                 o.CreatedTime,
		LastUpdated:          o.LastUpdateTime,
	}
	if !o.SizeInQuote && values.baseSize > 0 {
		d.RemainingAmount = max(values.baseSize-d.ExecutedAmount, 0)
	}
	// Coinbase reports no time an order was cancelled; a filled order closed at its last fill, and an expired one at its
	// end time
	switch status {
	case order.Filled:
		d.CloseTime = o.LastFillTime
	case order.Expired:
		d.CloseTime = values.endTime
	}
	// The user channel reports no order's size, so a live order's is recorded for its updates
	switch {
	case isLiveOrderStatus(status):
		e.orderSizes.setListed(o.OrderID, values.baseSize, values.quoteSize, listed)
	case isClosedOrderStatus(status):
		e.orderSizes.remove(o.OrderID)
	}
	return d, nil
}

// orderTrades returns an Advanced Trade order's fills as its trades, a page at a time; the fills Coinbase adjusts a
// fill with are left out, as the fill they adjust is already counted
func (e *Exchange) orderTrades(ctx context.Context, d *order.Detail) ([]order.TradeHistory, error) {
	req := &ListFillsRequest{OrderIDs: []string{d.OrderID}, Limit: fillsPageLimit}
	var trades []order.TradeHistory
	cursors := pageCursors{}
	for {
		cursors.visit(req.Cursor)
		resp, err := e.ListFills(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp.ProofTokenRequired {
			return nil, fmt.Errorf("%w: listing fills", errProofTokenRequired)
		}
		for i := range resp.Fills {
			f := &resp.Fills[i]
			if f.TradeType != "" && f.TradeType != FillTradeTypeFill {
				continue
			}
			price, amount := f.Price.Float64(), f.Size.Float64()
			if f.SizeInQuote && price != 0 {
				amount /= price
			}
			trades = append(trades, order.TradeHistory{
				Price:     price,
				Amount:    amount,
				Fee:       f.Commission.Float64(),
				Exchange:  e.Name,
				TID:       f.TradeID,
				Type:      d.Type,
				Side:      d.Side,
				Timestamp: f.TradeTime,
				IsMaker:   f.LiquidityIndicator == LiquidityIndicatorMaker,
				Total:     price * amount,
			})
		}
		if resp.Cursor == "" || len(resp.Fills) == 0 {
			return trades, nil
		}
		if cursors.seen(resp.Cursor) {
			return nil, fmt.Errorf("%w: fills cursor %q", errListPagingStalled, resp.Cursor)
		}
		req.Cursor = resp.Cursor
	}
}

// listOrders returns every order an Advanced Trade order list request selects, a page at a time
func (e *Exchange) listOrders(ctx context.Context, req *ListOrdersRequest) ([]Order, error) {
	req.Limit = ordersPageLimit
	var orders []Order
	cursors := pageCursors{}
	for {
		cursors.visit(req.Cursor)
		resp, err := e.ListOrders(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp.ProofTokenRequired {
			return nil, fmt.Errorf("%w: listing orders", errProofTokenRequired)
		}
		orders = append(orders, resp.Orders...)
		if !resp.HasNext || resp.Cursor == "" {
			return orders, nil
		}
		if cursors.seen(resp.Cursor) {
			return nil, fmt.Errorf("%w: orders cursor %q", errListPagingStalled, resp.Cursor)
		}
		req.Cursor = resp.Cursor
	}
}

// advancedTradeOrders returns the account's Advanced Trade orders of an asset in the given statuses, of the request's
// pairs, or every pair when it names none, created within its window
func (e *Exchange) advancedTradeOrders(ctx context.Context, req *order.MultiOrderRequest, statuses []string) ([]order.Detail, error) {
	productIDs := make([]string, len(req.Pairs))
	for i, p := range req.Pairs {
		var err error
		if productIDs[i], err = e.productID(p, req.AssetType); err != nil {
			return nil, err
		}
	}
	productType := ProductTypeSpot
	if req.AssetType == asset.Futures {
		productType = ProductTypeFuture
	}
	listed := time.Now()
	orders, err := e.listOrders(ctx, &ListOrdersRequest{
		ProductIDs:    productIDs,
		ProductType:   productType,
		OrderStatuses: statuses,
		StartDate:     req.StartTime,
		EndDate:       req.EndTime,
	})
	if err != nil {
		return nil, err
	}
	details := make([]order.Detail, len(orders))
	for i := range orders {
		d, err := e.orderDetail(&orders[i], listed)
		if err != nil {
			return nil, err
		}
		details[i] = *d
	}
	return details, nil
}

// derivativesOrderDetails returns gateway orders as GCT records them, leaving out the orders of instruments outside
// the asset
func (e *Exchange) derivativesOrderDetails(orders []DerivativesOrder, a asset.Item) ([]order.Detail, error) {
	details := make([]order.Detail, 0, len(orders))
	for i := range orders {
		if instrumentAsset, _, err := derivativesInstrumentPair(orders[i].InstrumentName); err != nil || instrumentAsset != a {
			continue
		}
		d, err := e.derivativesOrderDetail(&orders[i])
		if err != nil {
			return nil, err
		}
		details = append(details, *d)
	}
	return details, nil
}

// GetActiveOrders returns the account's open orders of an asset, of the request's pairs, or every pair when it names
// none: Advanced Trade's open, pending and queued orders, those queued for an edit or a cancellation among them, and
// the gateway's, the isolated margin subaccounts' included
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var details []order.Detail
	switch {
	case isAdvancedTradeAsset(req.AssetType):
		var err error
		// Coinbase lists open orders apart from orders in its other live statuses
		if details, err = e.advancedTradeOrders(ctx, req, []string{OrderStatusOpen}); err != nil {
			return nil, err
		}
		queued, err := e.advancedTradeOrders(ctx, req, []string{OrderStatusPending, OrderStatusQueued, OrderStatusEditQueued, OrderStatusCancelQueued})
		if err != nil {
			return nil, err
		}
		// An order changing status between the lists could be in both, and the later list holds its newer state
		for i := range queued {
			if j := slices.IndexFunc(details, func(d order.Detail) bool { return d.OrderID == queued[i].OrderID }); j != -1 {
				details[j] = queued[i]
				continue
			}
			details = append(details, queued[i])
		}
	case isDerivativesAsset(req.AssetType):
		// The open orders by instrument leave out the isolated margin subaccounts', so the request's pairs are filtered
		// from every open order
		open, err := e.GlobalDerivatives().getOpenOrders(ctx, e.derivativesOverWebsocket(ctx, true), &DerivativesOpenOrdersRequest{Kind: DerivativesKindFuture, IncludeIsolated: true})
		if err != nil {
			return nil, err
		}
		if details, err = e.derivativesOrderDetails(open, req.AssetType); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	return e.filterOrders(req, details), nil
}

// filterOrders applies a request's filters to orders: its margin type and time in force, which the shared filter leaves
// out, and the rest
func (e *Exchange) filterOrders(req *order.MultiOrderRequest, details []order.Detail) order.FilteredOrders {
	details = slices.DeleteFunc(details, func(d order.Detail) bool {
		return (req.MarginType != margin.Unset && d.MarginType != req.MarginType) ||
			(req.TimeInForce != order.UnknownTIF && !d.TimeInForce.Is(req.TimeInForce))
	})
	return req.Filter(e.Name, details)
}

// GetOrderHistory returns the account's closed orders of an asset, of the request's pairs, or every pair when it names
// none: Advanced Trade's filled, cancelled, expired and failed orders created within the request's window, and the
// gateway's filled and cancelled orders, older ones and those that never filled among them, back to the window's start
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var details []order.Detail
	switch {
	case isAdvancedTradeAsset(req.AssetType):
		var err error
		if details, err = e.advancedTradeOrders(ctx, req, []string{OrderStatusFilled, OrderStatusCancelled, OrderStatusExpired, OrderStatusFailed}); err != nil {
			return nil, err
		}
	case isDerivativesAsset(req.AssetType):
		history, err := e.derivativesOrderHistory(ctx, req)
		if err != nil {
			return nil, err
		}
		if details, err = e.derivativesOrderDetails(history, req.AssetType); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	return e.filterOrders(req, details), nil
}

// derivativesOrderHistory returns the gateway's closed orders of the request's instruments, or of the currencies its
// asset settles in when it names none, newest first, back to the request's start, keeping each order's latest state
func (e *Exchange) derivativesOrderHistory(ctx context.Context, req *order.MultiOrderRequest) ([]DerivativesOrder, error) {
	latest := make(map[string]DerivativesOrder)
	if err := e.pageDerivativesOrderHistory(ctx, req, func(orders []DerivativesOrder) bool {
		for i := range orders {
			if o, ok := latest[orders[i].OrderID]; !ok || orders[i].LastUpdateTimestamp.Time().After(o.LastUpdateTimestamp.Time()) {
				latest[orders[i].OrderID] = orders[i]
			}
		}
		return false
	}); err != nil {
		return nil, err
	}
	history := slices.Collect(maps.Values(latest))
	slices.SortFunc(history, func(a, b DerivativesOrder) int {
		return cmp.Or(b.CreationTimestamp.Time().Compare(a.CreationTimestamp.Time()), strings.Compare(a.OrderID, b.OrderID))
	})
	return history, nil
}

// derivativesClosedOrder returns a gateway order from the order history of its pair, or of its asset's currencies
// when no pair is given, as the gateway reports an order's state only until 30 minutes after it closes; the history is
// paged until the order is found
func (e *Exchange) derivativesClosedOrder(ctx context.Context, orderID string, p currency.Pair, a asset.Item) (*DerivativesOrder, error) {
	var found *DerivativesOrder
	if err := e.pageDerivativesOrderHistory(ctx, &order.MultiOrderRequest{AssetType: a, Pairs: pairsFor(p)}, func(orders []DerivativesOrder) bool {
		if i := slices.IndexFunc(orders, func(o DerivativesOrder) bool { return o.OrderID == orderID }); i != -1 {
			found = &orders[i]
		}
		return found != nil
	}); err != nil {
		return nil, err
	}
	if found == nil {
		return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, orderID)
	}
	return found, nil
}

// errDerivativesHistoryPagingStopped stops paging a gateway history once its visitor has what it needs
var errDerivativesHistoryPagingStopped = errors.New("gateway history paging stopped")

// pageDerivativesOrderHistory pages the gateway's order history of a request's instruments, or of the currencies its
// asset settles in when it names none, newest first, back to the request's start, passing each page to visit until it
// returns true. It reads the recent store, which keeps an order for 30 minutes, and the historical store, which indexes
// it after a short delay, for the main account and each isolated margin subaccount, whose history the gateway returns
// only for its subaccount ID, by currency, as a recycled subaccount can hold several instruments' orders
func (e *Exchange) pageDerivativesOrderHistory(ctx context.Context, req *order.MultiOrderRequest, visit func([]DerivativesOrder) bool) error {
	if err := e.walkDerivativesOrderHistory(ctx, req, visit); err != nil && !errors.Is(err, errDerivativesHistoryPagingStopped) {
		return err
	}
	return nil
}

// walkDerivativesOrderHistory is pageDerivativesOrderHistory, ending with errDerivativesHistoryPagingStopped when visit
// stops it
func (e *Exchange) walkDerivativesOrderHistory(ctx context.Context, req *order.MultiOrderRequest, visit func([]DerivativesOrder) bool) error {
	overWebsocket := e.derivativesOverWebsocket(ctx, true)
	g := e.GlobalDerivatives()
	names := make([]string, len(req.Pairs))
	currencies := derivativesAssetCurrencies(req.AssetType)
	if len(req.Pairs) != 0 {
		currencies = nil
		for i, p := range req.Pairs {
			var err error
			if names[i], err = e.instrumentName(p, req.AssetType); err != nil {
				return err
			}
			if c := derivativesSettlementCurrency(req.AssetType, p); !currencies.Contains(c) {
				currencies = append(currencies, c)
			}
		}
	}
	subaccounts, err := g.getSubaccounts(ctx, overWebsocket, &DerivativesGetSubaccountsRequest{IncludeIsolated: true})
	if err != nil {
		return err
	}
	page := func(fetch func(continuation string) (*DerivativesOrderHistoryResponse, error)) error {
		var continuation string
		continuations := pageCursors{}
		for {
			continuations.visit(continuation)
			resp, err := fetch(continuation)
			if err != nil {
				return err
			}
			if visit(resp.Orders) {
				return errDerivativesHistoryPagingStopped
			}
			if len(resp.Orders) == 0 || resp.Continuation == "" || resp.Continuation == derivativesNoContinuation {
				return nil
			}
			// An order updates after it is created, so once a page's last update is before the start, every earlier order
			// was created before it, whichever time the gateway sorts its history by
			if !req.StartTime.IsZero() && resp.Orders[len(resp.Orders)-1].LastUpdateTimestamp.Time().Before(req.StartTime) {
				return nil
			}
			if continuations.seen(resp.Continuation) {
				return fmt.Errorf("%w: order history continuation %q", errListPagingStalled, resp.Continuation)
			}
			continuation = resp.Continuation
		}
	}
	byCurrency := func(c currency.Code, subaccountID uint64, historical bool) error {
		return page(func(continuation string) (*DerivativesOrderHistoryResponse, error) {
			return g.getOrderHistoryByCurrency(ctx, overWebsocket, &DerivativesOrderHistoryByCurrencyRequest{
				Currency:         c,
				Kind:             DerivativesKindFuture,
				Count:            derivativesMaximumTradingCount,
				IncludeOld:       true,
				IncludeUnfilled:  true,
				WithContinuation: true,
				Continuation:     continuation,
				Historical:       historical,
				SubaccountID:     subaccountID,
			})
		})
	}
	for _, historical := range []bool{false, true} {
		for _, name := range names {
			if err := page(func(continuation string) (*DerivativesOrderHistoryResponse, error) {
				return g.getOrderHistoryByInstrument(ctx, overWebsocket, &DerivativesOrderHistoryByInstrumentRequest{
					InstrumentName:   name,
					Count:            derivativesMaximumTradingCount,
					IncludeOld:       true,
					IncludeUnfilled:  true,
					WithContinuation: true,
					Continuation:     continuation,
					Historical:       historical,
				})
			}); err != nil {
				return err
			}
		}
		for _, c := range currencies {
			if len(names) == 0 {
				if err := byCurrency(c, 0, historical); err != nil {
					return err
				}
			}
			for i := range subaccounts {
				if !subaccounts[i].Isolated {
					continue
				}
				if err := byCurrency(c, subaccounts[i].ID, historical); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// GetDepositAddress returns an address depositing a currency into a Coinbase account, the account ID given or else the
// currency's primary account, on the network given or else the currency's default network, which Coinbase Exchange
// names: the account's newest address on that network, or a new one when it has none
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, accountID, chain string) (*deposit.Address, error) {
	accountID, err := e.appAccountID(ctx, c, accountID)
	if err != nil {
		return nil, err
	}
	if chain == "" {
		// An address takes deposits on its own network alone; a currency on one network may name no default
		details, err := e.GetCurrency(ctx, c)
		if err != nil {
			return nil, err
		}
		chain = details.DefaultNetwork
	}
	var found *AppAddress
	if err := appPages(func(page *AppPageRequest) (*AppListResponse[AppAddress], error) {
		return e.App().ListAddresses(ctx, &AppListAddressesRequest{AccountID: accountID, Page: page})
	}, func(a *AppAddress) string { return a.ID }, func(a *AppAddress) bool {
		if chain != "" && !strings.EqualFold(a.Network, chain) {
			return false
		}
		address := *a
		found = &address
		return true
	}); err != nil {
		return nil, err
	}
	if found == nil {
		if found, err = e.App().CreateAddress(ctx, &AppCreateAddressRequest{AccountID: accountID, Network: chain}); err != nil {
			return nil, err
		}
	}
	return &deposit.Address{Address: found.Address, Tag: found.DestinationTag, Chain: found.Network}, nil
}

// WithdrawCryptocurrencyFunds sends a cryptocurrency from a Coinbase account, the request's wallet ID or else the
// currency's primary account, to an address on the request's chain, or else the currency's default network, with the
// travel rule details the request holds
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, w *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := w.Validate(); err != nil {
		return nil, err
	}
	accountID, err := e.appAccountID(ctx, w.Currency, w.WalletID)
	if err != nil {
		return nil, err
	}
	sent, err := e.App().SendMoney(ctx, &AppSendMoneyRequest{
		AccountID:      accountID,
		To:             w.Crypto.Address,
		Amount:         w.Amount,
		Currency:       w.Currency,
		Description:    w.Description,
		IdempotencyKey: w.IdempotencyToken,
		DestinationTag: w.Crypto.AddressTag,
		Network:        w.Crypto.Chain,
		TravelRuleData: withdrawalTravelRule(&w.Travel),
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: sent.ID, Status: sent.Status}, nil
}

// withdrawalTravelRule returns a withdrawal's travel rule details, or none when it names no beneficiary details
func withdrawalTravelRule(t *withdraw.TravelRule) *AppTravelRuleData {
	address := &AppTravelRuleAddress{
		Address1:   t.BeneficiaryAddress.Address1,
		Address2:   t.BeneficiaryAddress.Address2,
		Address3:   t.BeneficiaryAddress.Address3,
		City:       t.BeneficiaryAddress.City,
		State:      t.BeneficiaryAddress.State,
		Country:    t.BeneficiaryAddress.Country,
		PostalCode: t.BeneficiaryAddress.PostalCode,
	}
	if *address == (AppTravelRuleAddress{}) {
		address = nil
	}
	if !t.IsSelf && address == nil && t.BeneficiaryWalletType == "" && t.BeneficiaryName == "" && t.BeneficiaryFinancialInstitution == "" && t.TransferPurpose == "" {
		return nil
	}
	isSelf := AppTravelRuleIsSelfFalse
	if t.IsSelf {
		isSelf = AppTravelRuleIsSelfTrue
	}
	return &AppTravelRuleData{
		BeneficiaryWalletType:           t.BeneficiaryWalletType,
		IsSelf:                          isSelf,
		BeneficiaryName:                 t.BeneficiaryName,
		BeneficiaryAddress:              address,
		BeneficiaryFinancialInstitution: t.BeneficiaryFinancialInstitution,
		TransferPurpose:                 t.TransferPurpose,
	}
}

// WithdrawFiatFunds withdraws fiat from a Coinbase account, the request's wallet ID or else the currency's primary
// account, to the payment method named as the request's bank, committing the withdrawal at once
func (e *Exchange) WithdrawFiatFunds(ctx context.Context, w *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := w.Validate(); err != nil {
		return nil, err
	}
	methods, err := e.ListPaymentMethods(ctx)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(methods, func(m PaymentMethod) bool { return m.Name == w.Fiat.Bank.BankName && m.AllowWithdraw })
	if i == -1 {
		return nil, fmt.Errorf("%w: %q", errPaymentMethodNotFound, w.Fiat.Bank.BankName)
	}
	accountID, err := e.appAccountID(ctx, w.Currency, w.WalletID)
	if err != nil {
		return nil, err
	}
	transfer, err := e.App().WithdrawFunds(ctx, &AppFiatTransferRequest{AccountID: accountID, Amount: w.Amount, Currency: w.Currency, PaymentMethodID: methods[i].ID, Commit: true})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{Name: methods[i].Name, ID: transfer.ID, Status: transfer.Status}, nil
}

// WithdrawFiatFundsToInternationalBank withdraws fiat to a payment method, as WithdrawFiatFunds does
func (e *Exchange) WithdrawFiatFundsToInternationalBank(ctx context.Context, w *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return e.WithdrawFiatFunds(ctx, w)
}

// GetFeeByType returns an estimate of a trading fee: at the account's spot fee tier, or offline at the lowest tier's;
// Coinbase charges stable pairs their own rates, at every tier
func (e *Exchange) GetFeeByType(ctx context.Context, f *exchange.FeeBuilder) (float64, error) {
	if err := common.NilGuard(f); err != nil {
		return 0, err
	}
	// US derivatives are charged by the contract and the gateway's perpetuals at their own tier, which an estimate from
	// a price and amount cannot follow
	if a, ok := e.derivativesPairAsset(f.Pair); ok {
		return 0, fmt.Errorf("%w: fee estimates for %s, which Coinbase charges apart from spot", common.ErrFunctionNotSupported, a)
	}
	// The caller's request is left as it is, so it estimates at the account's tier once credentials are valid
	feeType := f.FeeType
	if !e.AreCredentialsValid(ctx) && feeType == exchange.CryptocurrencyTradeFee {
		feeType = exchange.OfflineTradeFee
	}
	var rate float64
	switch {
	case feeType != exchange.CryptocurrencyTradeFee && feeType != exchange.OfflineTradeFee:
		return 0, fmt.Errorf("%w: fee type %d", common.ErrFunctionNotSupported, feeType)
	case isStablePair(f.Pair) && f.IsMaker:
		rate = stablePairMakerFeeRate
	case isStablePair(f.Pair):
		rate = stablePairTakerFeeRate
	case feeType == exchange.OfflineTradeFee && f.IsMaker:
		rate = offlineMakerFeeRate
	case feeType == exchange.OfflineTradeFee:
		rate = offlineTakerFeeRate
	default:
		summary, err := e.GetTransactionSummary(ctx, &TransactionSummaryRequest{ProductType: ProductTypeSpot})
		if err != nil {
			return 0, err
		}
		rate = summary.FeeTier.TakerFeeRate.Float64()
		if f.IsMaker {
			rate = summary.FeeTier.MakerFeeRate.Float64()
		}
	}
	return rate * f.PurchasePrice * f.Amount, nil
}

// derivativesPairAsset returns the derivatives asset a pair trades as: the asset whose pairs hold it, or else the asset
// its symbol names, a US derivative's ending -CDE and a gateway instrument's naming its perpetual, before pairs load
func (e *Exchange) derivativesPairAsset(p currency.Pair) (asset.Item, bool) {
	if a, err := e.GetPairAssetType(p); err == nil {
		return a, a != asset.Spot
	}
	if strings.HasSuffix(strings.ToUpper(p.Quote.String()), "-CDE") {
		return asset.Futures, true
	}
	for _, delimiter := range []string{currency.UnderscoreDelimiter, currency.DashDelimiter} {
		if a, _, err := derivativesInstrumentPair(strings.ToUpper(p.Base.String() + delimiter + p.Quote.String())); err == nil {
			return a, true
		}
	}
	return asset.Empty, false
}

// isStablePair reports whether Coinbase charges a pair its stable pair fees
func isStablePair(p currency.Pair) bool {
	return slices.ContainsFunc(stablePairs, func(stable currency.Pair) bool { return stable.Equal(p) })
}

// GetAvailableTransferChains returns the networks a currency is deposited and withdrawn over, as Coinbase Exchange
// lists them, such as ethereum and solana for USDC
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, c currency.Code) ([]string, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	resp, err := e.GetCurrency(ctx, c)
	if err != nil {
		return nil, err
	}
	chains := make([]string, 0, len(resp.SupportedNetworks))
	for i := range resp.SupportedNetworks {
		if resp.SupportedNetworks[i].Status == exchangeNetworkStatusOnline {
			chains = append(chains, resp.SupportedNetworks[i].ID)
		}
	}
	if len(chains) == 0 {
		return nil, fmt.Errorf("%w: %s", errNoTransferChains, c)
	}
	return chains, nil
}

// AuthenticateWebsocket authenticates the gateway's private connection; Advanced Trade's user channels are
// authenticated with each subscription
func (e *Exchange) AuthenticateWebsocket(ctx context.Context) error {
	conn, err := e.Websocket.GetConnection(derivativesPrivateConnection)
	if err != nil {
		return err
	}
	return e.wsDerivativesSendAuth(ctx, conn)
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, a asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, a)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(p, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	candles, err := e.getCandles(ctx, req.RequestFormatted, req.Asset, req.ExchangeInterval, req.Start, req.End)
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval, over several requests
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(p, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for i := range req.RangeHolder.Ranges {
		candles, err := e.getCandles(ctx, req.RequestFormatted, req.Asset, req.ExchangeInterval, req.RangeHolder.Ranges[i].Start.Time, req.RangeHolder.Ranges[i].End.Time)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, candles...)
	}
	return req.ProcessResponse(timeSeries)
}

// getCandles returns a product's or instrument's candles between two times, oldest first
func (e *Exchange) getCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) ([]kline.Candle, error) {
	switch {
	case isAdvancedTradeAsset(a):
		granularity, err := candleGranularity(interval)
		if err != nil {
			return nil, err
		}
		// Coinbase includes the candle at a window's end and keeps the latest 350, so a full window would leave out the
		// candle at its start; ending it a second early holds the candles from the start up to the end
		candles, err := e.GetPublicProductCandles(ctx, &ProductCandlesRequest{ProductID: p.String(), Start: start, End: end.Add(-time.Second), Granularity: granularity})
		if err != nil {
			return nil, err
		}
		resp := make([]kline.Candle, len(candles))
		for i := range candles {
			// Coinbase sends candles newest first
			resp[len(candles)-1-i] = kline.Candle{
				Time:   candles[i].Start.Time().UTC(),
				Open:   candles[i].Open.Float64(),
				High:   candles[i].High.Float64(),
				Low:    candles[i].Low.Float64(),
				Close:  candles[i].Close.Float64(),
				Volume: candles[i].Volume.Float64(),
			}
		}
		return resp, nil
	case isDerivativesAsset(a):
		return e.derivativesCandles(ctx, p.String(), interval, start, end)
	}
	return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
}

// candleGranularity returns a candle interval as Advanced Trade names it
func candleGranularity(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return GranularityOneMinute, nil
	case kline.FiveMin:
		return GranularityFiveMinute, nil
	case kline.FifteenMin:
		return GranularityFifteenMinute, nil
	case kline.ThirtyMin:
		return GranularityThirtyMinute, nil
	case kline.OneHour:
		return GranularityOneHour, nil
	case kline.TwoHour:
		return GranularityTwoHour, nil
	case kline.FourHour:
		return GranularityFourHour, nil
	case kline.SixHour:
		return GranularitySixHour, nil
	case kline.OneDay:
		return GranularityOneDay, nil
	}
	return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
}

// derivativesCandles returns a gateway instrument's candles between two times, oldest first. The gateway aligns its 3,
// 6 and 12 hour and daily candles to its 08:00 UTC daily settlement and has no 4 hour candles, so candles of those
// intervals are built from 2 hour candles, which it aligns to UTC
func (e *Exchange) derivativesCandles(ctx context.Context, instrumentName string, interval kline.Interval, start, end time.Time) ([]kline.Candle, error) {
	resolution := interval
	if !derivativesCandleAligned(interval) {
		resolution = derivativesCandleBase
	}
	// The gateway includes the candle at a window's end, which belongs to the next window and would leave a candle built
	// from candles partly in it short of its last
	resp, err := e.GlobalDerivatives().getTradingViewChartData(ctx, e.derivativesOverWebsocket(ctx, false), &DerivativesTradingViewChartDataRequest{
		InstrumentName: instrumentName,
		StartTime:      start,
		EndTime:        end.Add(-time.Millisecond),
		Resolution:     resolution,
	})
	if err != nil {
		return nil, err
	}
	length := len(resp.Ticks)
	if len(resp.Open) != length || len(resp.High) != length || len(resp.Low) != length || len(resp.Close) != length || len(resp.Volume) != length {
		return nil, fmt.Errorf("%w: %d ticks, %d opens, %d highs, %d lows, %d closes and %d volumes", errCandleFieldsMismatch, length, len(resp.Open), len(resp.High), len(resp.Low), len(resp.Close), len(resp.Volume))
	}
	candles := make([]kline.Candle, length)
	for i := range resp.Ticks {
		candles[i] = kline.Candle{
			Time:   resp.Ticks[i].Time().UTC(),
			Open:   resp.Open[i],
			High:   resp.High[i],
			Low:    resp.Low[i],
			Close:  resp.Close[i],
			Volume: resp.Volume[i],
		}
	}
	if resolution != interval {
		return aggregateCandles(candles, interval), nil
	}
	return candles, nil
}

// derivativesCandleAligned reports whether the gateway serves an interval's candles aligned to UTC; it aligns its 3, 6
// and 12 hour and daily candles to its 08:00 UTC daily settlement
func derivativesCandleAligned(interval kline.Interval) bool {
	switch interval {
	case kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.TenMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour:
		return true
	}
	return false
}

// aggregateCandles merges consecutive candles, oldest first, into candles of a longer interval, each starting at a
// multiple of the interval in UTC
func aggregateCandles(candles []kline.Candle, interval kline.Interval) []kline.Candle {
	var merged []kline.Candle
	for i := range candles {
		start := candles[i].Time.Truncate(interval.Duration())
		if n := len(merged); n != 0 && merged[n-1].Time.Equal(start) {
			c := &merged[n-1]
			c.High = max(c.High, candles[i].High)
			c.Low = min(c.Low, candles[i].Low)
			c.Close = candles[i].Close
			c.Volume += candles[i].Volume
			continue
		}
		c := candles[i]
		c.Time = start
		merged = append(merged, c)
	}
	return merged
}

// GetServerTime returns the time of the Advanced Trade API or the gateway, for the asset given
func (e *Exchange) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	if isDerivativesAsset(a) {
		return e.GlobalDerivatives().getTime(ctx, e.derivativesOverWebsocket(ctx, false))
	}
	resp, err := e.GetPublicServerTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return resp.ISO, nil
}

// GetFuturesContractDetails returns a futures asset's contracts: the US derivatives Coinbase Financial Markets lists,
// dated monthly contracts and perpetual style ones paying funding, cash settled in USD; or the gateway's perpetuals,
// linear in USDC or inverse in their base currency
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, a asset.Item) ([]futures.Contract, error) {
	if !a.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	switch {
	case a == asset.Futures:
		products, err := e.listProducts(ctx, a)
		if err != nil {
			return nil, err
		}
		contracts := make([]futures.Contract, 0, len(products))
		for i := range products {
			p := &products[i]
			d := p.FutureProductDetails
			if d == nil {
				continue
			}
			pair, err := currency.NewPairDelimiter(p.ProductID, currency.DashDelimiter)
			if err != nil {
				return nil, err
			}
			c := futures.Contract{
				Exchange:           e.Name,
				Name:               pair,
				Underlying:         currency.NewPair(currency.NewCode(d.ContractRootUnit), currency.NewCode(p.QuoteCurrencyID)),
				Asset:              a,
				StartDate:          p.NewAt,
				EndDate:            d.ContractExpiry,
				IsActive:           tradableProduct(p),
				Type:               futures.Monthly,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.NewCode(p.QuoteCurrencyID),
				MarginCurrency:     currency.NewCode(p.QuoteCurrencyID),
				Multiplier:         d.ContractSize.Float64(),
			}
			if s := p.FCMTradingSessionDetails; s != nil {
				c.Status = s.SessionState
			}
			if isPerpetualProduct(p) {
				c.Type, c.EndDate = futures.Perpetual, time.Time{}
				c.LatestRate = fundingrate.Rate{Time: d.FundingTime, Rate: decimal.MustFromFloat(d.FundingRate.Float64())}
			}
			contracts = append(contracts, c)
		}
		return contracts, nil
	case isDerivativesAsset(a):
		instruments, err := e.derivativesPerpetuals(ctx, a)
		if err != nil {
			return nil, err
		}
		contracts := make([]futures.Contract, len(instruments))
		for i := range instruments {
			instrument := &instruments[i]
			_, pair, err := derivativesInstrumentPair(instrument.InstrumentName)
			if err != nil {
				return nil, err
			}
			contracts[i] = futures.Contract{
				Exchange:           e.Name,
				Name:               pair,
				Underlying:         currency.NewPair(instrument.BaseCurrency, instrument.CounterCurrency),
				Asset:              a,
				StartDate:          instrument.CreationTimestamp.Time(),
				IsActive:           instrument.IsActive,
				Status:             instrument.State,
				Type:               futures.Perpetual,
				SettlementType:     derivativesSettlementType(instrument.InstrumentType),
				SettlementCurrency: instrument.SettlementCurrency,
				MarginCurrency:     instrument.SettlementCurrency,
				Multiplier:         instrument.ContractSize,
				MaxLeverage:        float64(instrument.MaximumLeverage),
			}
		}
		return contracts, nil
	}
	return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
}

// derivativesSettlementType returns how a gateway instrument settles: inverse for a reversed instrument, which is
// margined in its base currency, and linear otherwise
func derivativesSettlementType(instrumentType string) futures.ContractSettlementType {
	if instrumentType == derivativesInstrumentTypeReversed {
		return futures.Inverse
	}
	return futures.Linear
}

// GetLatestFundingRates returns the latest funding rates of a futures asset's perpetual, or of its enabled perpetuals
// when no pair is given: a perpetual style US derivative's hourly rate, as its product reports it, or a gateway
// perpetual's latest hourly funding record. Neither venue predicts the next rate
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w: predicted funding rates", common.ErrFunctionNotSupported)
	}
	switch {
	case r.Asset == asset.Futures:
		return e.futuresLatestFundingRates(ctx, r.Pair)
	case isDerivativesAsset(r.Asset):
		return e.derivativesLatestFundingRates(ctx, r.Asset, r.Pair)
	}
	return nil, fmt.Errorf("%w: %s %s", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
}

// futuresLatestFundingRates returns the funding rates of a perpetual style US derivative, or of the enabled ones when
// no pair is given
func (e *Exchange) futuresLatestFundingRates(ctx context.Context, p currency.Pair) ([]fundingrate.LatestRateResponse, error) {
	if !p.IsEmpty() {
		id, err := e.productID(p, asset.Futures)
		if err != nil {
			return nil, err
		}
		product, err := e.GetPublicProduct(ctx, id)
		if err != nil {
			return nil, err
		}
		if !isPerpetualProduct(product) {
			return nil, fmt.Errorf("%w: %s %s", futures.ErrNotPerpetualFuture, asset.Futures, p)
		}
		return []fundingrate.LatestRateResponse{e.futuresFundingRate(product, p)}, nil
	}
	pairs, err := e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	products, err := e.listProducts(ctx, asset.Futures)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*Product, len(products))
	for i := range products {
		byID[products[i].ProductID] = &products[i]
	}
	var resp []fundingrate.LatestRateResponse
	for _, pair := range pairs {
		id, err := e.productID(pair, asset.Futures)
		if err != nil {
			return nil, err
		}
		if product, ok := byID[id]; ok && isPerpetualProduct(product) {
			resp = append(resp, e.futuresFundingRate(product, pair))
		}
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("%w: %s", fundingrate.ErrNoFundingRatesFound, asset.Futures)
	}
	return resp, nil
}

// futuresFundingRate returns a perpetual style US derivative's latest funding rate, which it pays each funding
// interval
func (e *Exchange) futuresFundingRate(p *Product, pair currency.Pair) fundingrate.LatestRateResponse {
	d := p.FutureProductDetails
	resp := fundingrate.LatestRateResponse{
		Exchange:    e.Name,
		Asset:       asset.Futures,
		Pair:        pair,
		LatestRate:  fundingrate.Rate{Time: d.FundingTime, Rate: decimal.MustFromFloat(d.FundingRate.Float64())},
		TimeChecked: time.Now(),
	}
	if !d.FundingTime.IsZero() {
		resp.TimeOfNextRate = d.FundingTime.Add(d.FundingInterval.Duration())
	}
	return resp
}

// derivativesLatestFundingRates returns the latest hourly funding records of a gateway perpetual, or of the asset's
// enabled perpetuals when no pair is given, a request each
func (e *Exchange) derivativesLatestFundingRates(ctx context.Context, a asset.Item, p currency.Pair) ([]fundingrate.LatestRateResponse, error) {
	pairs := currency.Pairs{p}
	if p.IsEmpty() {
		var err error
		if pairs, err = e.GetEnabledPairs(a); err != nil {
			return nil, err
		}
		if len(pairs) == 0 {
			return nil, fmt.Errorf("%w: %s", currency.ErrCurrencyPairsEmpty, a)
		}
	}
	overWebsocket := e.derivativesOverWebsocket(ctx, false)
	now := time.Now()
	resp := make([]fundingrate.LatestRateResponse, len(pairs))
	for i, pair := range pairs {
		name, err := e.instrumentName(pair, a)
		if err != nil {
			return nil, err
		}
		records, err := e.GlobalDerivatives().getFundingRateHistory(ctx, overWebsocket, &DerivativesInstrumentTimeRangeRequest{InstrumentName: name, StartTime: now.Add(-derivativesLatestFundingRateWindow), EndTime: now})
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return nil, fmt.Errorf("%w: %s", fundingrate.ErrNoFundingRatesFound, name)
		}
		latest := &records[len(records)-1]
		resp[i] = fundingrate.LatestRateResponse{
			Exchange:       e.Name,
			Asset:          a,
			Pair:           pair,
			LatestRate:     fundingrate.Rate{Time: latest.Timestamp.Time(), Rate: decimal.MustFromFloat(latest.Interest1Hour)},
			TimeOfNextRate: latest.Timestamp.Time().Add(derivativesFundingInterval),
			TimeChecked:    now,
		}
	}
	return resp, nil
}

// GetHistoricalFundingRates returns a gateway perpetual's hourly funding rates within a time range. The gateway returns
// the latest page of a range, oldest first, so earlier pages end before the first record of the last. US derivatives
// have no funding history, and neither venue reports payments or predicts rates
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Asset == asset.Futures:
		return nil, fmt.Errorf("%w: US derivatives funding history", common.ErrFunctionNotSupported)
	case !isDerivativesAsset(r.Asset):
		return nil, fmt.Errorf("%w: %s %s", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	case r.IncludePayments:
		return nil, fmt.Errorf("%w: funding payments", common.ErrFunctionNotSupported)
	case r.IncludePredictedRate:
		return nil, fmt.Errorf("%w: predicted funding rates", common.ErrFunctionNotSupported)
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	name, err := e.instrumentName(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	overWebsocket := e.derivativesOverWebsocket(ctx, false)
	var rates []fundingrate.Rate
	for end := r.EndDate; ; {
		records, err := e.GlobalDerivatives().getFundingRateHistory(ctx, overWebsocket, &DerivativesInstrumentTimeRangeRequest{InstrumentName: name, StartTime: r.StartDate, EndTime: end})
		if err != nil {
			return nil, err
		}
		page := make([]fundingrate.Rate, 0, len(records)+len(rates))
		for i := range records {
			page = append(page, fundingrate.Rate{Time: records[i].Timestamp.Time(), Rate: decimal.MustFromFloat(records[i].Interest1Hour)})
		}
		rates = append(page, rates...)
		if len(records) < derivativesFundingRatePageLimit {
			break
		}
		next := records[0].Timestamp.Time().Add(-time.Millisecond)
		if !next.After(r.StartDate) || !next.Before(end) {
			break
		}
		end = next
	}
	if len(rates) == 0 {
		return nil, fmt.Errorf("%w: %s %s from %s to %s", fundingrate.ErrNoFundingRatesFound, r.Asset, r.Pair, r.StartDate, r.EndDate)
	}
	return &fundingrate.HistoricalRates{
		Exchange:        e.Name,
		Asset:           r.Asset,
		Pair:            r.Pair,
		FundingRates:    rates,
		StartDate:       rates[0].Time,
		EndDate:         rates[len(rates)-1].Time,
		LatestRate:      rates[len(rates)-1],
		PaymentCurrency: r.PaymentCurrency,
	}, nil
}

// IsPerpetualFutureCurrency reports whether a futures pair is perpetual: every gateway pair is, and a US derivative is
// when it pays funding, which product data loads
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, p currency.Pair) (bool, error) {
	switch {
	case isDerivativesAsset(a):
		return true, nil
	case a != asset.Futures:
		return false, nil
	}
	id, err := e.productID(p, a)
	if err != nil {
		return false, err
	}
	e.perpetualFuturesMu.RLock()
	defer e.perpetualFuturesMu.RUnlock()
	if e.perpetualFutures == nil {
		return false, errPerpetualFuturesNotLoaded
	}
	_, ok := e.perpetualFutures[id]
	return ok, nil
}

// GetCollateralCurrencyForContract returns the currency a contract is margined in: USD for US derivatives, USDC for
// the USDC perpetuals and the base currency for the inverse perpetuals
func (e *Exchange) GetCollateralCurrencyForContract(a asset.Item, p currency.Pair) (currency.Code, asset.Item, error) {
	switch {
	case a == asset.Futures:
		return currency.USD, a, nil
	case isDerivativesAsset(a):
		if p.IsEmpty() {
			return currency.EMPTYCODE, asset.Empty, currency.ErrCurrencyPairEmpty
		}
		return derivativesSettlementCurrency(a, p), a, nil
	}
	return currency.EMPTYCODE, asset.Empty, fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, a)
}

// GetOpenInterest returns the open interest of the futures pairs given, or of every futures pair the venues list: a
// US derivative's in contracts, and a gateway perpetual's in its base currency, or USD for an inverse perpetual
func (e *Exchange) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range keys {
		if keys[i].Asset != asset.Futures && !isDerivativesAsset(keys[i].Asset) {
			return nil, fmt.Errorf("%w: %s %s", asset.ErrNotSupported, keys[i].Asset, keys[i].Pair())
		}
	}
	wanted := func(a asset.Item) bool {
		return len(keys) == 0 || slices.ContainsFunc(keys, func(k key.PairAsset) bool { return k.Asset == a })
	}
	var resp []futures.OpenInterest
	if wanted(asset.Futures) {
		products, err := e.listProducts(ctx, asset.Futures)
		if err != nil {
			return nil, err
		}
		for i := range products {
			d := products[i].FutureProductDetails
			if d == nil {
				continue
			}
			pair, err := currency.NewPairDelimiter(products[i].ProductID, currency.DashDelimiter)
			if err != nil {
				return nil, err
			}
			resp = append(resp, futures.OpenInterest{Key: key.NewExchangeAssetPair(e.Name, asset.Futures, pair), OpenInterest: d.OpenInterest.Float64()})
		}
	}
	for _, a := range []asset.Item{asset.USDCMarginedFutures, asset.CoinMarginedFutures} {
		if !wanted(a) {
			continue
		}
		summaries, err := e.derivativesBookSummaries(ctx, derivativesAssetCurrencies(a))
		if err != nil {
			return nil, err
		}
		for name, summary := range summaries {
			instrumentAsset, pair, err := derivativesInstrumentPair(name)
			if err != nil || instrumentAsset != a {
				continue
			}
			resp = append(resp, futures.OpenInterest{Key: key.NewExchangeAssetPair(e.Name, a, pair), OpenInterest: summary.OpenInterest})
		}
	}
	if len(keys) == 0 {
		return resp, nil
	}
	filtered := make([]futures.OpenInterest, 0, len(keys))
	for i := range keys {
		j := slices.IndexFunc(resp, func(o futures.OpenInterest) bool {
			return o.Key.Asset == keys[i].Asset && o.Key.Pair().Equal(keys[i].Pair())
		})
		if j == -1 {
			return nil, fmt.Errorf("%w: %s %s", errInstrumentNotFound, keys[i].Asset, keys[i].Pair())
		}
		filtered = append(filtered, resp[j])
	}
	return filtered, nil
}

// SetLeverage sets a gateway perpetual's leverage, for its cross margin or isolated margin positions; US derivatives
// are margined by their contracts, without leverage to set
func (e *Exchange) SetLeverage(ctx context.Context, a asset.Item, p currency.Pair, marginType margin.Type, amount float64, _ order.Side) error {
	if a == asset.Futures {
		return fmt.Errorf("%w: US derivatives take no leverage", common.ErrFunctionNotSupported)
	}
	name, err := e.instrumentName(p, a)
	if err != nil {
		return err
	}
	isolated, err := derivativesIsolated(marginType)
	if err != nil {
		return err
	}
	_, err = e.GlobalDerivatives().setLeverage(ctx, e.derivativesOverWebsocket(ctx, true), &DerivativesSetLeverageRequest{InstrumentName: name, Leverage: &amount, Isolated: isolated})
	return err
}

// GetLeverage returns a gateway perpetual's leverage, for its cross margin or isolated margin positions: its custom
// leverage, or its maximum leverage without one
func (e *Exchange) GetLeverage(ctx context.Context, a asset.Item, p currency.Pair, marginType margin.Type, _ order.Side) (float64, error) {
	if a == asset.Futures {
		return 0, fmt.Errorf("%w: US derivatives take no leverage", common.ErrFunctionNotSupported)
	}
	name, err := e.instrumentName(p, a)
	if err != nil {
		return 0, err
	}
	isolated, err := derivativesIsolated(marginType)
	if err != nil {
		return 0, err
	}
	resp, err := e.GlobalDerivatives().getLeverage(ctx, e.derivativesOverWebsocket(ctx, true), &DerivativesGetLeverageRequest{InstrumentName: name, Isolated: isolated})
	if err != nil {
		return 0, err
	}
	i := slices.IndexFunc(resp, func(l DerivativesLeverage) bool { return l.InstrumentName == name })
	if i == -1 {
		return 0, fmt.Errorf("%w: %s", errInstrumentNotFound, name)
	}
	if resp[i].Leverage != 0 {
		return resp[i].Leverage, nil
	}
	return resp[i].MaximumLeverage, nil
}

// GetFuturesPositionSummary returns the account's position in a futures contract: a US derivative's, whose sizes are in
// contracts, or a gateway perpetual's cross margin position, whose sizes are in its base currency, as are a linear
// perpetual's amounts, and an inverse one's in USD. A short position's size is negative
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, r *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Asset == asset.Futures:
		id, err := e.productID(r.Pair, r.Asset)
		if err != nil {
			return nil, err
		}
		position, err := e.GetFuturesPosition(ctx, id)
		if err != nil {
			return nil, err
		}
		product, err := e.GetPublicProduct(ctx, id)
		if err != nil {
			return nil, err
		}
		var multiplier float64
		if d := product.FutureProductDetails; d != nil {
			multiplier = d.ContractSize.Float64()
		}
		contracts := position.NumberOfContracts.Float64()
		if position.Side == FuturesPositionSideShort {
			contracts = -contracts
		}
		return &futures.PositionSummary{
			Pair:                   r.Pair,
			Asset:                  r.Asset,
			MarginType:             margin.Multi,
			Currency:               currency.USD,
			NotionalSize:           decimal.MustFromFloat(math.Abs(contracts) * multiplier * position.CurrentPrice.Float64()),
			MarkPrice:              decimal.MustFromFloat(position.CurrentPrice.Float64()),
			CurrentSize:            decimal.MustFromFloat(contracts),
			ContractSize:           decimal.MustFromFloat(contracts),
			ContractMultiplier:     decimal.MustFromFloat(multiplier),
			ContractSettlementType: futures.Linear,
			AverageOpenPrice:       decimal.MustFromFloat(position.AverageEntryPrice.Float64()),
			UnrealisedPNL:          decimal.MustFromFloat(position.UnrealisedPNL.Float64()),
			RealisedPNL:            decimal.MustFromFloat(position.DailyRealisedPNL.Float64()),
		}, nil
	case isDerivativesAsset(r.Asset):
		name, err := e.instrumentName(r.Pair, r.Asset)
		if err != nil {
			return nil, err
		}
		instrument, err := e.GlobalDerivatives().getInstrument(ctx, e.derivativesOverWebsocket(ctx, false), name)
		if err != nil {
			return nil, err
		}
		position, err := e.GlobalDerivatives().getPosition(ctx, e.derivativesOverWebsocket(ctx, true), &DerivativesGetPositionRequest{InstrumentName: name})
		if err != nil {
			return nil, err
		}
		var contracts float64
		if instrument.ContractSize != 0 {
			contracts = position.Size / instrument.ContractSize
		}
		return &futures.PositionSummary{
			Pair:                         r.Pair,
			Asset:                        r.Asset,
			MarginType:                   margin.Multi,
			Currency:                     instrument.SettlementCurrency,
			NotionalSize:                 decimal.MustFromFloat(position.SizeCurrency * position.MarkPrice),
			Leverage:                     decimal.MustFromFloat(position.Leverage),
			MaintenanceMarginRequirement: decimal.MustFromFloat(position.MaintenanceMargin),
			InitialMarginRequirement:     decimal.MustFromFloat(position.InitialMargin),
			EstimatedLiquidationPrice:    decimal.MustFromFloat(position.EstimatedLiquidationPrice),
			MarkPrice:                    decimal.MustFromFloat(position.MarkPrice),
			CurrentSize:                  decimal.MustFromFloat(position.SizeCurrency),
			ContractSize:                 decimal.MustFromFloat(contracts),
			ContractMultiplier:           decimal.MustFromFloat(instrument.ContractSize),
			ContractSettlementType:       derivativesSettlementType(instrument.InstrumentType),
			AverageOpenPrice:             decimal.MustFromFloat(position.AveragePrice),
			UnrealisedPNL:                decimal.MustFromFloat(position.TotalProfitLoss - position.RealisedProfitLoss),
			RealisedPNL:                  decimal.MustFromFloat(position.RealisedProfitLoss),
		}, nil
	}
	return nil, fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, r.Asset)
}

// GetFuturesPositionOrders returns the orders of futures contracts within a window, open or closed
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, r *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.Futures && !isDerivativesAsset(r.Asset) {
		return nil, fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, r.Asset)
	}
	if len(r.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	settlementType := futures.Linear
	if r.Asset == asset.CoinMarginedFutures {
		settlementType = futures.Inverse
	}
	resp := make([]futures.PositionResponse, len(r.Pairs))
	for i, p := range r.Pairs {
		req := &order.MultiOrderRequest{AssetType: r.Asset, Pairs: currency.Pairs{p}, StartTime: r.StartDate, EndTime: r.EndDate, Side: order.AnySide, Type: order.AnyType}
		closed, err := e.GetOrderHistory(ctx, req)
		if err != nil {
			return nil, err
		}
		open, err := e.GetActiveOrders(ctx, req)
		if err != nil {
			return nil, err
		}
		resp[i] = futures.PositionResponse{
			Pair:                   p,
			Asset:                  r.Asset,
			ContractSettlementType: settlementType,
			Orders:                 append(open, closed...),
		}
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair. Coinbase's
// web app trades the gateway's USDC perpetuals alone, so the inverse perpetuals have no trade page
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, p currency.Pair) (string, error) {
	if _, err := e.CurrencyPairs.IsPairEnabled(p, a); err != nil {
		return "", err
	}
	symbol, err := e.formattedSymbol(p, a)
	if err != nil {
		return "", err
	}
	switch a {
	case asset.Spot:
		return tradeBaseURL + "spot/" + symbol, nil
	case asset.Futures:
		return tradeBaseURL + "futures/" + symbol, nil
	case asset.USDCMarginedFutures:
		return tradeBaseURL + "perpetuals/" + symbol, nil
	case asset.CoinMarginedFutures:
		return "", fmt.Errorf("%w: Coinbase's web app has no trade page for the gateway's inverse perpetuals", common.ErrFunctionNotSupported)
	}
	return "", fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
}

// derivativesOverWebsocket reports whether a wrapper method sends a gateway request over the gateway's private
// websocket connection: a public request whenever it is connected, and a private request when the configuration also
// lets the wrapper use the authenticated websocket and the connection is authenticated with the API key of the
// context's credentials, which can name another account. The connection is only dialled with a private gateway
// subscription, so the wrapper's requests otherwise go over HTTP
func (e *Exchange) derivativesOverWebsocket(ctx context.Context, private bool) bool {
	if !e.Websocket.IsConnected() || (private && !e.Websocket.CanUseAuthenticatedWebsocketForWrapper()) {
		return false
	}
	conn, err := e.Websocket.GetConnection(derivativesPrivateConnection)
	if err != nil {
		return false
	}
	if !private {
		return true
	}
	creds, err := e.GetCredentials(ctx)
	return err == nil && creds.Key != "" && creds.Key == e.derivativesSessions.authenticatedKey(conn)
}
