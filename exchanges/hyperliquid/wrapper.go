package hyperliquid

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

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
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
)

const (
	// Builder-deployed perpetual DEXs number their asset IDs from 100000, in blocks of 10000 per DEX index
	builderPerpetualAssetIDBase    = 100000
	builderPerpetualDEXAssetStride = 10000
	// spotAssetIDBase offsets spot universe indices into the asset ID space
	spotAssetIDBase              = 10000
	pairMappingMissCacheDuration = time.Minute
	// Prices allow five significant figures and at most this many decimals less the market's size decimals
	marketPriceSignificantFigures = 5
	perpetualPriceDecimalBase     = 6
	spotPriceDecimalBase          = 8
	// defaultMarketSlippage bounds a market order that sets no slippage tolerance, as in Hyperliquid's Python SDK
	defaultMarketSlippage = 0.05
	// defaultTriggerSlippage bounds a market trigger order or TP/SL child that sets neither a limit price nor a slippage
	// tolerance, as Hyperliquid bounds its TP/SL market orders
	defaultTriggerSlippage = 0.1
	// maximumFrontendOpenOrders is how many of a perpetual DEX's open orders frontendOpenOrders returns for an account
	// with very many, such as one of HLP's child vaults; it returns every open order of most accounts
	maximumFrontendOpenOrders = 100
	// Base tier fee rates for offline fee estimates
	perpetualMakerBaseFeeRate = 0.00015
	perpetualTakerBaseFeeRate = 0.00045
	spotMakerBaseFeeRate      = 0.0004
	spotTakerBaseFeeRate      = 0.0007
	// growthModeFeeMultiplier is the share of its usual fees and rebates that a builder-deployed market in growth mode
	// charges and pays
	growthModeFeeMultiplier = 0.1
	// stablePairFeeMultiplier is the share of its usual fees and rebates that a spot pair between two spot quote assets
	// charges and pays
	stablePairFeeMultiplier = 0.2
)

var (
	errAmbiguousCoinMapping       = errors.New("ambiguous coin mapping")
	errAssetContextNotFound       = errors.New("asset context not found")
	errCrossMarginUnavailable     = errors.New("cross margin is unavailable for an isolated-only market")
	errEndpointEnvironment        = errors.New("endpoint does not match the configured sandbox environment")
	errGroupedOrderChildFailure   = errors.New("one or more grouped TP/SL child orders were rejected")
	errInvalidFilledSize          = errors.New("invalid filled order size")
	errInvalidMarketPrice         = errors.New("invalid market price")
	errInvalidPerpetualDEX        = errors.New("invalid perpetual DEX registry or metadata")
	errMalformedResponse          = errors.New("malformed response")
	errMarketDelisted             = errors.New("market is delisted")
	errMarketMidPriceNotFound     = errors.New("market mid price not found")
	errModifyOrderTypeUnsupported = errors.New("only resting limit orders can be modified")
	errOrderNotModifiable         = errors.New("order is not open for modification")
	errPairMappingNotFound        = errors.New("pair mapping not found")
	errPricePrecision             = errors.New("price exceeds market precision")
	errQuoteAmountUnsupported     = errors.New("quote amounts are unsupported")
	errRiskManagementUnsupported  = errors.New("unsupported risk management configuration")
	errSizePrecision              = errors.New("size exceeds market precision")
	errSlippageTolerance          = errors.New("market order slippage tolerance must be greater than 0 and less than 1")
	errSpotTokenNotFound          = errors.New("spot token metadata not found")
	errTransferCurrencyInvalid    = errors.New("hyperliquid transfers only support USDC")
	errTriggerOrderPerpetualOnly  = errors.New("trigger orders are only supported for perpetual markets")
	errUnsupportedOrderStatus     = errors.New("unsupported order status")
	errWithdrawalAddressTag       = errors.New("hyperliquid transfers do not support an address tag")
	errWithdrawalFeeInput         = errors.New("hyperliquid calculates the transfer fee")
)

// pairMapping links a currency pair to the Hyperliquid coin and asset ID it trades as
type pairMapping struct {
	pair         currency.Pair
	coin         string
	dex          string
	assetID      uint64
	sizeDecimals uint64
	maxLeverage  uint64
	onlyIsolated bool
	// deployerFeeScale and growthMode set the fees of a builder-deployed perpetual market
	deployerFeeScale float64
	growthMode       bool
}

func logDefaultError(err error) {
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// SetDefaults sets the basic defaults for Hyperliquid
func (e *Exchange) SetDefaults() {
	e.Name = "Hyperliquid"
	e.Enabled = true
	e.Verbose = true
	e.BaseCurrencies = currency.Currencies{currency.USDC}
	e.API.CredentialsValidator.RequiresKey = true

	pairFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		logDefaultError(e.SetAssetPairStore(a, currency.PairStore{AssetEnabled: true, RequestFormat: pairFormat, ConfigFormat: pairFormat}))
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:                 true,
				TickerFetching:                 true,
				KlineFetching:                  true,
				TradeFetching:                  true,
				OrderbookFetching:              true,
				AutoPairUpdates:                true,
				AccountBalance:                 true,
				CryptoWithdrawal:               true,
				DepositHistory:                 true,
				WithdrawalHistory:              true,
				GetOrder:                       true,
				GetOrders:                      true,
				CancelOrders:                   true,
				CancelOrder:                    true,
				SubmitOrder:                    true,
				ModifyOrder:                    true,
				TradeFee:                       true,
				FundingRateFetching:            true,
				AuthenticatedEndpoints:         true,
				HasAssetTypeAccountSegregation: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				KlineFetching:          true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				GetOrders:              true,
				AuthenticatedEndpoints: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				FundingRateBatching: map[asset.Item]bool{
					asset.PerpetualContract: true,
				},
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.OneHour: true,
				},
				Leverage: true,
				OpenInterest: exchange.OpenInterestSupport{
					Supported:         true,
					SupportsRestBatch: true,
				},
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.EightHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					// Hyperliquid's 3d, 1w and 1M candles open on its own days, which GCT's candle alignment refuses: 3d
					// on epoch days divisible by three, 1w on Thursdays and 1M, a 30-day candle, on epoch days divisible
					// by 30. Leaving them out builds 3d and 1w from daily candles; GetHistoricCandles refuses 1M
					kline.IntervalCapacity{Interval: kline.OneDay},
				),
				GlobalResultLimit: maximumCandleCount,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(rateLimits))
	logDefaultError(err)
	e.API.Endpoints = e.NewEndpoints()
	logDefaultError(e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      apiURL,
		exchange.WebsocketSpot: websocketURL,
	}))
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.pairMappingsMu.Lock()
	e.pairMappings = make(map[asset.Item][]pairMapping)
	e.pairMappingsMu.Unlock()
	e.pairMappingsFetchMu.Lock()
	e.pairMappingMisses = make(map[string]time.Time)
	e.pairMappingsFetchMu.Unlock()
	e.authorityValidationMu.Lock()
	e.authorityValidations = make(map[authorityValidationKey]struct{})
	e.authorityValidationMu.Unlock()
	e.websocketPendingMu.Lock()
	e.websocketPending = make(map[websocketPendingKey]*websocketPendingOperation)
	e.websocketPendingMu.Unlock()
}

// Setup sets the exchange configuration profile
// With useSandbox set, endpoints still at their production defaults move to testnet; custom endpoints are kept
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
	for _, endpoint := range []struct {
		kind       exchange.URL
		production string
		sandbox    string
	}{
		{kind: exchange.RestSpot, production: apiURL, sandbox: testnetAPIURL},
		{kind: exchange.WebsocketSpot, production: websocketURL, sandbox: testnetWebsocketURL},
	} {
		runningURL, err := e.API.Endpoints.GetURL(endpoint.kind)
		if err != nil {
			return err
		}
		runningURL = strings.TrimRight(runningURL, "/")
		switch {
		case exch.UseSandbox && runningURL == endpoint.production:
			if err := e.API.Endpoints.SetRunningURL(endpoint.kind.String(), endpoint.sandbox); err != nil {
				return err
			}
		case !exch.UseSandbox && runningURL == endpoint.sandbox:
			return fmt.Errorf("%w: %s uses the testnet URL while useSandbox is false", errEndpointEnvironment, endpoint.kind)
		}
	}
	wsURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	// Subscription acknowledgements are awaited outside the connections, so they need the configured limit too
	e.WebsocketResponseMaxLimit = exch.WebsocketResponseMaxLimit
	// Hyperliquid limits subscriptions per IP address rather than per connection, which subscribing enforces across
	// both connections
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exch,
		Features:                     &e.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement: true,
		TradeFeed:                    exch.Features.Enabled.TradeFeed,
		FillsFeed:                    exch.Features.Enabled.FillsFeed,
	}); err != nil {
		return err
	}
	// Authenticated websocket use means the account connection is up, which its connector reports
	e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	// Both connections share one limiter, as Hyperliquid limits the messages an IP address sends
	connectionRateLimit := request.NewWeightedRateLimitByDuration(websocketMessageInterval)
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                         wsURL,
		RateLimit:                   connectionRateLimit,
		ResponseCheckTimeout:        exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:            exch.WebsocketResponseMaxLimit,
		Connector:                   e.connectPublicWebsocket,
		GenerateSubscriptions:       e.generatePublicSubscriptions,
		Subscriber:                  e.websocketSubscribe,
		Unsubscriber:                e.websocketUnsubscribe,
		Handler:                     e.wsHandleData,
		OnDisconnect:                e.publicWebsocketDisconnected,
		MessageFilter:               websocketPublicConnection,
		ConnectWithoutSubscriptions: true,
	}); err != nil {
		return err
	}
	// The account connection is optional, so the public feeds stay up while it cannot connect, and the manager retries
	// it with a backoff that keeps within the 30 new connections a minute Hyperliquid accepts from an IP address
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsURL,
		RateLimit:             connectionRateLimit,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.connectAccountWebsocket,
		GenerateSubscriptions: e.generateAccountSubscriptions,
		Subscriber:            e.websocketSubscribe,
		Unsubscriber:          e.websocketUnsubscribe,
		Handler:               e.wsHandleData,
		OnDisconnect:          e.accountWebsocketDisconnected,
		MessageFilter:         websocketAccountConnection,
		Optional:              true,
	})
}

// FetchTradablePairs returns the active spot markets, or the active perpetual markets of every perpetual DEX
// A display pair listed by more than one market is ambiguous, so it is skipped rather than mapped to either
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	var candidates []pairMapping
	var err error
	switch a {
	case asset.PerpetualContract:
		candidates, err = e.fetchPerpetualPairMappings(ctx)
	case asset.Spot:
		candidates, err = e.fetchSpotPairMappings(ctx)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if err != nil {
		return nil, err
	}
	counts := make(map[currency.Pair]int, len(candidates))
	for i := range candidates {
		counts[candidates[i].pair]++
	}
	pairs := make(currency.Pairs, 0, len(candidates))
	mappings := make([]pairMapping, 0, len(candidates))
	reported := make(map[currency.Pair]struct{})
	for i := range candidates {
		if counts[candidates[i].pair] > 1 {
			if _, ok := reported[candidates[i].pair]; !ok {
				log.Warnf(log.ExchangeSys, "%s skipping ambiguous %s display pair %s", e.Name, a, candidates[i].pair)
				reported[candidates[i].pair] = struct{}{}
			}
			continue
		}
		pairs = append(pairs, candidates[i].pair)
		mappings = append(mappings, candidates[i])
	}
	e.setPairMappings(a, mappings)
	return pairs, nil
}

// fetchPerpetualPairMappings maps every active perpetual market, quoted in its DEX's collateral token, and keeps the
// delisted markets, which only lookups by coin resolve, so the orders and fills of a delisted market still convert
func (e *Exchange) fetchPerpetualPairMappings(ctx context.Context) ([]pairMapping, error) {
	dexes, err := e.getPerpetualDEXNames(ctx)
	if err != nil {
		return nil, err
	}
	collateralNames := map[uint64]currency.Code{0: currency.USDC}
	var candidates, delisted []pairMapping
	for dexIndex, dex := range dexes {
		metadata, err := e.GetPerpetualMetadata(ctx, dex)
		if err != nil {
			return nil, err
		}
		collateralName, ok := collateralNames[metadata.CollateralToken]
		if !ok {
			if collateralNames, err = e.getSpotTokenNames(ctx); err != nil {
				return nil, err
			}
			if collateralName, ok = collateralNames[metadata.CollateralToken]; !ok {
				return nil, fmt.Errorf("%w: collateral token index %d", errSpotTokenNotFound, metadata.CollateralToken)
			}
		}
		if dexIndex != 0 && len(metadata.Universe) > builderPerpetualDEXAssetStride {
			return nil, fmt.Errorf("%w: DEX %q has %d markets, maximum %d", errInvalidPerpetualDEX, dex, len(metadata.Universe), builderPerpetualDEXAssetStride)
		}
		var offset uint64
		if dexIndex != 0 {
			offset = builderPerpetualAssetIDBase + uint64(dexIndex)*builderPerpetualDEXAssetStride
		}
		live := make(map[string]struct{}, len(metadata.Universe))
		var dexDelisted []pairMapping
		for marketIndex := range metadata.Universe {
			market := &metadata.Universe[marketIndex]
			if dex != "" && !strings.HasPrefix(market.Name, dex+":") {
				return nil, fmt.Errorf("%w: market %q is not scoped to DEX %q", errInvalidPerpetualDEX, market.Name, dex)
			}
			pair, err := currency.NewPairFromStrings(market.Name, collateralName.String())
			if err != nil {
				log.Warnf(log.ExchangeSys, "%s skipping perpetual market %q: %s", e.Name, market.Name, err)
				continue
			}
			mapping := pairMapping{
				pair:             pair,
				coin:             market.Name,
				dex:              dex,
				assetID:          offset + uint64(marketIndex),
				sizeDecimals:     market.SizeDecimals,
				maxLeverage:      market.MaxLeverage,
				onlyIsolated:     market.MarginMode != "",
				deployerFeeScale: market.DeployerFeeScale.Float64(),
				growthMode:       market.GrowthMode == "enabled",
			}
			if market.IsDelisted {
				dexDelisted = append(dexDelisted, mapping)
				continue
			}
			live[market.Name] = struct{}{}
			candidates = append(candidates, mapping)
		}
		// A coin a live market reuses names the live market, so the delisted one would only shadow it
		for i := range dexDelisted {
			if _, reused := live[dexDelisted[i].coin]; !reused {
				delisted = append(delisted, dexDelisted[i])
			}
		}
	}
	e.pairMappingsMu.Lock()
	e.delistedPairMappings = delisted
	e.pairMappingsMu.Unlock()
	return candidates, nil
}

// getSpotTokenNames maps spot token indices to names, which builder DEXs use for their collateral token
func (e *Exchange) getSpotTokenNames(ctx context.Context) (map[uint64]currency.Code, error) {
	metadata, err := e.GetSpotMetadata(ctx)
	if err != nil {
		return nil, err
	}
	names := make(map[uint64]currency.Code, len(metadata.Tokens))
	for i := range metadata.Tokens {
		index := metadata.Tokens[i].Index
		if _, exists := names[index]; exists {
			return nil, fmt.Errorf("%w: duplicate spot token index %d", errMalformedResponse, index)
		}
		name := metadata.Tokens[i].Name
		if name.IsEmpty() {
			return nil, fmt.Errorf("%w: token index %d has no name", errSpotTokenNotFound, index)
		}
		if index == 0 && !name.Equal(currency.USDC) {
			return nil, fmt.Errorf("%w: token index 0 is %s", errSpotTokenNotFound, name)
		}
		names[index] = name
	}
	return names, nil
}

// fetchSpotPairMappings maps every spot market to its base and quote token names
func (e *Exchange) fetchSpotPairMappings(ctx context.Context) ([]pairMapping, error) {
	metadata, err := e.GetSpotMetadata(ctx)
	if err != nil {
		return nil, err
	}
	tokens := make(map[uint64]*SpotTokenMetadata, len(metadata.Tokens))
	for i := range metadata.Tokens {
		if _, exists := tokens[metadata.Tokens[i].Index]; exists {
			return nil, fmt.Errorf("%w: duplicate spot token index %d", errMalformedResponse, metadata.Tokens[i].Index)
		}
		tokens[metadata.Tokens[i].Index] = &metadata.Tokens[i]
	}
	candidates := make([]pairMapping, 0, len(metadata.Universe))
	marketIndices := make(map[uint64]struct{}, len(metadata.Universe))
	marketNames := make(map[string]struct{}, len(metadata.Universe))
	for _, market := range metadata.Universe {
		if _, exists := marketIndices[market.Index]; exists {
			return nil, fmt.Errorf("%w: duplicate spot market index %d", errMalformedResponse, market.Index)
		}
		marketIndices[market.Index] = struct{}{}
		if _, exists := marketNames[market.Name]; exists {
			return nil, fmt.Errorf("%w: duplicate spot market name %q", errMalformedResponse, market.Name)
		}
		marketNames[market.Name] = struct{}{}
		if len(market.Tokens) != 2 {
			log.Warnf(log.ExchangeSys, "%s skipping spot market %q: a spot market must reference exactly two tokens, got %d", e.Name, market.Name, len(market.Tokens))
			continue
		}
		baseToken, ok := tokens[market.Tokens[0]]
		if !ok {
			log.Warnf(log.ExchangeSys, "%s skipping spot market %q: %s for base index %d", e.Name, market.Name, errSpotTokenNotFound, market.Tokens[0])
			continue
		}
		quoteToken, ok := tokens[market.Tokens[1]]
		if !ok {
			log.Warnf(log.ExchangeSys, "%s skipping spot market %q: %s for quote index %d", e.Name, market.Name, errSpotTokenNotFound, market.Tokens[1])
			continue
		}
		if baseToken.Name.IsEmpty() || quoteToken.Name.IsEmpty() {
			log.Warnf(log.ExchangeSys, "%s skipping spot market %q with base %q and quote %q", e.Name, market.Name, baseToken.Name, quoteToken.Name)
			continue
		}
		pair := currency.NewPair(baseToken.Name, quoteToken.Name)
		candidates = append(candidates, pairMapping{
			pair:         pair,
			coin:         market.Name,
			assetID:      spotAssetIDBase + market.Index,
			sizeDecimals: baseToken.SizeDecimals,
		})
	}
	return candidates, nil
}

// UpdateTradablePairs updates all available Hyperliquid markets
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

// UpdateTickers updates the tickers of an asset type's enabled pairs from the batch market contexts
// Contexts carry no last trade price, so the ticker's last price is the mid price, falling back to the mark price
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	var prices []ticker.Price
	var errs error
	switch a {
	case asset.PerpetualContract:
		contexts := make(map[string]map[string]*PerpetualAssetContext)
		dexErrors := make(map[string]error)
		for _, pair := range pairs {
			mapping, err := e.getPairMapping(ctx, pair, a)
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%s: %w", pair, err))
				continue
			}
			dexContexts, loaded := contexts[mapping.dex]
			if !loaded && dexErrors[mapping.dex] == nil {
				resp, err := e.GetPerpetualMetadataAndAssetContexts(ctx, mapping.dex)
				if err != nil {
					dexErrors[mapping.dex] = err
				} else {
					dexContexts = make(map[string]*PerpetualAssetContext, len(resp.AssetContexts))
					for i := range resp.Metadata.Universe {
						dexContexts[resp.Metadata.Universe[i].Name] = &resp.AssetContexts[i]
					}
					contexts[mapping.dex] = dexContexts
				}
			}
			if dexErrors[mapping.dex] != nil {
				errs = common.AppendError(errs, fmt.Errorf("%s: %w", pair, dexErrors[mapping.dex]))
				continue
			}
			market, ok := dexContexts[mapping.coin]
			if !ok {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", errAssetContextNotFound, mapping.coin))
				continue
			}
			prices = append(prices, *e.perpetualTickerPrice(pair, market))
		}
	case asset.Spot:
		resp, err := e.GetSpotMetadataAndAssetContexts(ctx)
		if err != nil {
			return err
		}
		contexts := make(map[string]*SpotAssetContext, len(resp.AssetContexts))
		for i := range resp.AssetContexts {
			contexts[resp.AssetContexts[i].Coin] = &resp.AssetContexts[i]
		}
		for _, pair := range pairs {
			coin, err := e.getCoin(ctx, pair, a)
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%s: %w", pair, err))
				continue
			}
			market, ok := contexts[coin]
			if !ok {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", errAssetContextNotFound, coin))
				continue
			}
			prices = append(prices, *e.spotTickerPrice(pair, market))
		}
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if len(prices) == 0 {
		return errs
	}
	_, err = ticker.ProcessBatch(prices)
	return common.AppendError(errs, err)
}

// UpdateTicker updates and returns one pair's ticker from its market context, as UpdateTickers does, so any available
// pair is served and a failing sibling pair does not affect it
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	mapping, err := e.getPairMapping(ctx, p, a)
	if err != nil {
		return nil, err
	}
	var price *ticker.Price
	if a == asset.Spot {
		resp, err := e.GetSpotMetadataAndAssetContexts(ctx)
		if err != nil {
			return nil, err
		}
		index := slices.IndexFunc(resp.AssetContexts, func(market SpotAssetContext) bool { return market.Coin == mapping.coin })
		if index == -1 {
			return nil, fmt.Errorf("%w: %s", errAssetContextNotFound, mapping.coin)
		}
		price = e.spotTickerPrice(p, &resp.AssetContexts[index])
	} else {
		contexts, err := e.getPerpetualAssetContexts(ctx, []pairMapping{mapping})
		if err != nil {
			return nil, err
		}
		price = e.perpetualTickerPrice(p, &contexts[0])
	}
	if err := ticker.ProcessTicker(price); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns one L2 orderbook snapshot
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	coin, err := e.getCoin(ctx, p, a)
	if err != nil {
		return nil, err
	}
	resp, err := e.GetL2Book(ctx, &L2BookRequest{Coin: coin})
	if err != nil {
		return nil, err
	}
	book, err := e.convertL2Book(resp, p, a)
	if err != nil {
		return nil, err
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, a)
}

// GetRecentTrades returns recent public trades for a pair and asset type
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	coin, err := e.getCoin(ctx, p, a)
	if err != nil {
		return nil, err
	}
	resp, err := e.GetRecentTradesForCoin(ctx, coin)
	if err != nil {
		return nil, err
	}
	trades := make([]trade.Data, len(resp))
	for i := range resp {
		if trades[i], err = e.convertTrade(&resp[i], p, a); err != nil {
			return nil, err
		}
	}
	trade.SortByDate(trades)
	return trades, e.AddTradesToBuffer(trades...)
}

// GetHistoricTrades is not supported by Hyperliquid's recent trades endpoint
func (e *Exchange) GetHistoricTrades(context.Context, currency.Pair, asset.Item, time.Time, time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandles returns candles within the requested time range
func (e *Exchange) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	// GCT aligns these intervals to calendar months, which neither Hyperliquid's 1M candle follows nor GCT's conversion
	// from daily candles, as it groups a fixed number of days
	switch interval {
	case kline.OneMonth, kline.ThreeMonth, kline.SixMonth, kline.NineMonth, kline.OneYear:
		return nil, fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval.Word())
	}
	req, err := e.GetKlineRequest(p, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}
	coin, err := e.getCoin(ctx, p, a)
	if err != nil {
		return nil, err
	}
	resp, err := e.GetCandleSnapshot(ctx, &CandleSnapshotRequest{
		Coin:      coin,
		Interval:  req.ExchangeInterval,
		StartTime: req.Start,
		// Hyperliquid's end time is inclusive, but kline request ranges exclude their end
		EndTime: req.End.Add(-time.Millisecond),
	})
	if err != nil {
		return nil, err
	}
	candles := make([]kline.Candle, len(resp))
	for i := range resp {
		candles[i] = convertCandle(&resp[i])
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended is not supported because Hyperliquid retains only the latest 5000 candles
func (e *Exchange) GetHistoricCandlesExtended(context.Context, currency.Pair, asset.Item, kline.Interval, time.Time, time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateAccountBalances retrieves balances for the configured account, vault or subaccount
// Unified and portfolio margin accounts hold one balance across spot and perpetuals, which is stored under spot and
// refreshed with perpetual balances while spot is disabled; their perpetual DEX balances are saved empty so stale
// separate balances are cleared without double counting. While spot is disabled, other accounts' perpetual balances
// are saved with an empty spot balance, which clears a shared balance stored before the account changed mode
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	if a != asset.Spot && a != asset.PerpetualContract {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	address, err := e.getWatchAddress(ctx)
	if err != nil {
		return nil, err
	}
	var subAccounts accounts.SubAccounts
	switch a {
	case asset.Spot:
		subAccount, err := e.spotSubAccount(ctx, address)
		if err != nil {
			return nil, err
		}
		subAccounts = accounts.SubAccounts{subAccount}
	case asset.PerpetualContract:
		mode, err := e.GetUserAbstraction(ctx, address)
		if err != nil {
			return nil, err
		}
		dexes, err := e.getPerpetualDEXNames(ctx)
		if err != nil {
			return nil, err
		}
		subAccounts = make(accounts.SubAccounts, 0, len(dexes))
		if mode == AccountAbstractionUnifiedAccount || mode == AccountAbstractionPortfolioMargin {
			for i := range dexes {
				subAccounts = append(subAccounts, accounts.NewSubAccount(a, dexSubAccountID(address, dexes[i])))
			}
			if e.CurrencyPairs.IsAssetEnabled(asset.Spot) == nil {
				break
			}
			// The engine refreshes only the enabled assets, so with spot disabled the shared balance is refreshed here and
			// saved with the cleared DEX balances, so that no reader sees it beside a stale DEX balance
			spot, err := e.spotSubAccount(ctx, address)
			if err != nil {
				return nil, err
			}
			return subAccounts, e.Accounts.Save(ctx, append(accounts.SubAccounts{spot}, subAccounts...), true)
		}
		tokenNames, err := e.getSpotTokenNames(ctx)
		if err != nil {
			return nil, err
		}
		allMetadata, err := e.GetAllPerpetualMetadata(ctx)
		if err != nil {
			return nil, err
		}
		for i := range dexes {
			metadata := perpetualMetadataForDEX(allMetadata, dexes[i])
			if metadata == nil {
				if metadata, err = e.GetPerpetualMetadata(ctx, dexes[i]); err != nil {
					return nil, err
				}
			}
			collateral, ok := tokenNames[metadata.CollateralToken]
			if !ok {
				return nil, fmt.Errorf("%w: collateral token index %d", errSpotTokenNotFound, metadata.CollateralToken)
			}
			state, err := e.GetClearinghouseState(ctx, address, dexes[i])
			if err != nil {
				return nil, err
			}
			subAccount := accounts.NewSubAccount(a, dexSubAccountID(address, dexes[i]))
			subAccount.Balances.Set(collateral, convertPerpetualBalance(state))
			subAccounts = append(subAccounts, subAccount)
		}
		if e.CurrencyPairs.IsAssetEnabled(asset.Spot) != nil {
			// The engine refreshes only the enabled assets, so with spot disabled a balance stored under spot while the
			// account shared one is cleared with these rather than counted beside them
			return subAccounts, e.Accounts.Save(ctx, append(accounts.SubAccounts{accounts.NewSubAccount(asset.Spot, address)}, subAccounts...), true)
		}
	}
	return subAccounts, e.Accounts.Save(ctx, subAccounts, true)
}

// spotSubAccount returns the account's spot balances, which unified and portfolio margin accounts also trade
// perpetuals with
func (e *Exchange) spotSubAccount(ctx context.Context, address string) (*accounts.SubAccount, error) {
	state, err := e.GetSpotClearinghouseState(ctx, address)
	if err != nil {
		return nil, err
	}
	subAccount := accounts.NewSubAccount(asset.Spot, address)
	for i := range state.Balances {
		subAccount.Balances.Set(state.Balances[i].Coin, convertSpotBalance(&state.Balances[i]))
	}
	return subAccount, nil
}

// perpetualMetadataForDEX identifies a DEX by every market's prefix, never by the bulk response's position. Empty,
// mixed or duplicate entries cannot identify its collateral safely, so callers must fetch that DEX's metadata instead.
func perpetualMetadataForDEX(metadata []PerpetualMetadataResponse, dex string) *PerpetualMetadataResponse {
	var result *PerpetualMetadataResponse
	for i := range metadata {
		var matched, mismatched bool
		for j := range metadata[i].Universe {
			prefix, coin, hasPrefix := strings.Cut(metadata[i].Universe[j].Name, ":")
			if !hasPrefix {
				coin, prefix = prefix, ""
			}
			if prefix == dex {
				matched = true
			} else {
				mismatched = true
			}
			if strings.TrimSpace(coin) == "" || strings.Contains(coin, ":") || (hasPrefix && prefix == "") {
				mismatched = true
			}
		}
		if !matched {
			continue
		}
		if mismatched || result != nil {
			return nil
		}
		result = &metadata[i]
	}
	return result
}

// convertPerpetualBalance converts a perpetual DEX's account summary; an account whose positions have lost more than
// their margin is worth less than the margin it uses, so its hold is capped at its total, as a spot balance's is
func convertPerpetualBalance(state *ClearinghouseStateResponse) accounts.Balance {
	total := state.MarginSummary.AccountValue.Float64()
	return accounts.Balance{
		Total: total,
		Hold:  min(state.MarginSummary.TotalMarginUsed.Float64(), total),
		Free:  state.Withdrawable.Float64(),
	}
}

// convertSpotBalance converts a spot token balance; portfolio margin accounts report a spot hold beside a hold that can
// be negative, so the larger of the two, from zero up to the total, is held and the rest of the total is free
func convertSpotBalance(balance *SpotBalance) accounts.Balance {
	total := balance.Total.Float64()
	hold := min(max(balance.Hold.Float64(), balance.SpotHold.Float64(), 0), total)
	return accounts.Balance{Total: total, Hold: hold, Free: total - hold}
}

// GetAccountFundingHistory returns the configured account's deposits, withdrawals, transfers, rewards and other
// non-funding ledger movements; a fee the account paid in another currency than its movement has an entry of its own
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	user, err := e.getWatchAddress(ctx)
	if err != nil {
		return nil, err
	}
	updates, err := e.getUserNonFundingLedgerUpdatesPaginated(ctx, user, time.UnixMilli(1).UTC())
	if err != nil {
		return nil, err
	}
	result := make([]exchange.FundingHistory, 0, len(updates))
	for i := range updates {
		entries, err := e.convertUserLedgerUpdate(user, &updates[i])
		if err != nil {
			return nil, err
		}
		result = append(result, entries...)
	}
	slices.SortStableFunc(result, func(a, b exchange.FundingHistory) int { return a.Timestamp.Compare(b.Timestamp) })
	return result, nil
}

// GetWithdrawalsHistory returns processed bridge withdrawals
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	if !c.IsEmpty() && !c.Equal(currency.USDC) {
		return []exchange.WithdrawalHistory{}, nil
	}
	user, err := e.getWatchAddress(ctx)
	if err != nil {
		return nil, err
	}
	updates, err := e.getUserNonFundingLedgerUpdatesPaginated(ctx, user, time.UnixMilli(1).UTC())
	if err != nil {
		return nil, err
	}
	result := make([]exchange.WithdrawalHistory, 0)
	for i := range updates {
		if updates[i].Delta.Type != "withdraw" {
			continue
		}
		result = append(result, exchange.WithdrawalHistory{
			Status:       "processed",
			TransferID:   updates[i].Hash,
			Description:  "Hyperliquid bridge withdrawal",
			Timestamp:    updates[i].Time.Time().UTC(),
			Currency:     currency.USDC.String(),
			Amount:       updates[i].Delta.USDC.Float64(),
			Fee:          updates[i].Delta.Fee.Float64(),
			TransferType: "withdrawal",
			CryptoTxID:   updates[i].Hash,
			CryptoChain:  e.getBridgeChain(),
		})
	}
	return result, nil
}

// GetServerTime returns the time Hyperliquid's exchange status reports; every asset type shares one server, so the asset
// type is not checked
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	status, err := e.GetExchangeStatus(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return status.Time.Time().UTC(), nil
}

// SubmitOrder submits a limit, slippage-bounded market or trigger order, with any grouped TP/SL children
func (e *Exchange) SubmitOrder(ctx context.Context, submit *order.Submit) (*order.SubmitResponse, error) {
	if err := submit.Validate(protocol.TradingRequirements{}); err != nil {
		return nil, err
	}
	if submit.QuoteAmount != 0 {
		return nil, fmt.Errorf("%w: Hyperliquid orders are sized in the base currency", errQuoteAmountUnsupported)
	}
	if _, _, err := e.getSigningCredentials(ctx); err != nil {
		return nil, err
	}
	orders, mapping, grouping, err := e.buildOrderRequests(ctx, submit)
	if err != nil {
		return nil, err
	}
	statuses, err := e.PlaceOrders(ctx, &PlaceOrdersRequest{Orders: orders, Grouping: grouping})
	if err != nil {
		return nil, err
	}
	parent := statuses[0]
	if parent.Error != "" {
		return nil, fmt.Errorf("%w: %s", order.ErrUnableToPlaceOrder, parent.Error)
	}
	if parent.Waiting != "" {
		return nil, fmt.Errorf("%w: parent order cannot be %s", errActionStatusMalformed, parent.Waiting)
	}
	var timeInForce string
	if orders[0].Limit != nil {
		timeInForce = orders[0].Limit.TimeInForce
	}
	var orderID uint64
	status := order.New
	remainingAmount := submit.Amount
	switch {
	case parent.Resting != nil:
		orderID = parent.Resting.OrderID
	case parent.Filled != nil:
		orderID = parent.Filled.OrderID
		if status, remainingAmount, err = deriveFilledOrderState(submit.Amount, parent.Filled.TotalSize.Float64(), mapping.sizeDecimals, timeInForce); err != nil {
			return nil, err
		}
	}
	result, _ := submit.DeriveSubmitResponse(strconv.FormatUint(orderID, 10)) // An accepted order always has a non-zero order ID
	result.Status = status
	result.Price = orders[0].Price
	result.TimeInForce = order.UnknownTIF
	if timeInForce != "" {
		result.TimeInForce, _ = parseTimeInForce(timeInForce) // buildOrderRequest only sets time in force values it can parse
	}
	if parent.Filled != nil {
		result.AverageExecutedPrice = parent.Filled.AveragePrice.Float64()
	}
	result.RemainingAmount = remainingAmount
	var childErrors error
	for i := 1; i < len(statuses); i++ {
		if statuses[i].Error != "" {
			childErrors = common.AppendError(childErrors, fmt.Errorf("child %d: %s", i, statuses[i].Error))
		}
	}
	if childErrors != nil {
		result.SubmissionError = fmt.Errorf("%w: %w", errGroupedOrderChildFailure, childErrors)
	}
	return result, nil
}

// ModifyOrder replaces a resting limit order, keeping any fields the modification omits. Amount is the requested total
// of the current order; its executed size is subtracted before replacement. The response reports the replacement size,
// which becomes the total for subsequent amendments of the replacement order. A take profit or stop loss is refused,
// as the replacement cannot carry one.
// Without always place, Hyperliquid only accepts a replacement that rests: it must not be a trigger order and must be
// ALO or a GTC that would not execute, which then rests as ALO
func (e *Exchange) ModifyOrder(ctx context.Context, modify *order.Modify) (*order.ModifyResponse, error) {
	if err := modify.Validate(); err != nil {
		return nil, err
	}
	if _, _, err := e.getSigningCredentials(ctx); err != nil {
		return nil, err
	}
	// Requested fields are checked before the order lookup; the order's own fields, which unset ones inherit, after it
	if err := validateModifiableOrder(cmp.Or(modify.Type, order.Limit), modify.TimeInForce); err != nil {
		return nil, err
	}
	if modify.TriggerPrice != 0 {
		return nil, fmt.Errorf("%w: trigger price requires a trigger order type", errRiskManagementUnsupported)
	}
	if risk := &modify.RiskManagementModes; risk.TakeProfit.Enabled || risk.StopLoss.Enabled || risk.StopEntry.Enabled {
		return nil, fmt.Errorf("%w: take profit and stop loss cannot be amended", errRiskManagementUnsupported)
	}
	if modify.Side != order.UnknownSide && !order.IsValidOrderSubmissionSide(modify.Side) {
		return nil, fmt.Errorf("%w: %s", order.ErrSideIsInvalid, modify.Side)
	}
	if modify.NewClientOrderID != "" {
		if err := validateClientOrderID(modify.NewClientOrderID); err != nil {
			return nil, err
		}
	}
	modifyRequest := OrderModification{ClientOrderID: modify.ClientOrderID}
	identifier := modify.OrderID
	if modify.OrderID != "" {
		id, err := strconv.ParseUint(modify.OrderID, 10, 64)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("%w: %q", order.ErrOrderIDNotSet, modify.OrderID)
		}
		modifyRequest = OrderModification{OrderID: id}
	} else {
		if err := validateClientOrderID(modify.ClientOrderID); err != nil {
			return nil, err
		}
		identifier = strings.ToLower(modify.ClientOrderID)
	}
	existing, err := e.GetOrderInfo(ctx, identifier, modify.Pair, modify.AssetType)
	if err != nil {
		return nil, err
	}
	if existing.IsInactive() || existing.RemainingAmount <= 0 {
		return nil, fmt.Errorf("%w: %s", errOrderNotModifiable, identifier)
	}
	orderType := cmp.Or(modify.Type, existing.Type)
	timeInForce := cmp.Or(modify.TimeInForce, existing.TimeInForce)
	if err := validateModifiableOrder(orderType, timeInForce); err != nil {
		return nil, err
	}
	side := cmp.Or(modify.Side, existing.Side)
	amount := cmp.Or(modify.Amount, existing.Amount)
	price := cmp.Or(modify.Price, existing.Price)
	clientOrderID := cmp.Or(modify.NewClientOrderID, existing.ClientOrderID)
	replacement, mapping, err := e.buildOrderRequest(ctx, modify.Pair, modify.AssetType, orderType, side, timeInForce, amount, price, 0, 0, existing.ReduceOnly, clientOrderID)
	if err != nil {
		return nil, err
	}
	// buildOrderRequest validates the total before subtraction. Whole size units avoid residue from the venue's
	// original-minus-remaining execution calculation becoming an over-precise replacement.
	scale := math.Pow10(int(mapping.sizeDecimals)) //nolint:gosec // buildOrderRequest limits sizeDecimals to 8
	total := amount
	amount = (math.Round(total*scale) - math.Round(existing.ExecutedAmount*scale)) / scale
	if amount <= 0 {
		return nil, fmt.Errorf("%w: total %v does not exceed executed size %v", order.ErrAmountIsInvalid, total, existing.ExecutedAmount)
	}
	replacement.Size = amount
	modifyRequest.Order = replacement
	statuses, err := e.ModifyOrders(ctx, &ModifyOrdersRequest{Modifies: []OrderModification{modifyRequest}})
	if err != nil {
		return nil, err
	}
	if statuses[0].Error != "" {
		return nil, fmt.Errorf("%w: %s", order.ErrUnableToPlaceOrder, statuses[0].Error)
	}
	status := order.New
	remainingAmount := amount
	orderID, _ := strconv.ParseUint(existing.OrderID, 10, 64) // GetOrderInfo formats a parsed numeric order ID
	switch {
	case statuses[0].Resting != nil:
		orderID = statuses[0].Resting.OrderID
	case statuses[0].Filled != nil:
		orderID = statuses[0].Filled.OrderID
		if status, remainingAmount, err = deriveFilledOrderState(amount, statuses[0].Filled.TotalSize.Float64(), mapping.sizeDecimals, replacement.Limit.TimeInForce); err != nil {
			return nil, err
		}
	}
	return &order.ModifyResponse{
		Exchange:        e.Name,
		OrderID:         strconv.FormatUint(orderID, 10),
		ClientOrderID:   clientOrderID,
		Pair:            modify.Pair,
		Type:            orderType,
		Side:            side,
		Status:          status,
		AssetType:       modify.AssetType,
		TimeInForce:     order.PostOnly,
		Price:           replacement.Price,
		Amount:          amount,
		RemainingAmount: remainingAmount,
		Date:            existing.Date,
		LastUpdated:     time.Now().UTC(),
	}, nil
}

// validateModifiableOrder checks a replacement can rest, which only a limit order that is not IOC can
func validateModifiableOrder(orderType order.Type, timeInForce order.TimeInForce) error {
	if orderType != order.Limit {
		return fmt.Errorf("%w: %w: %s", order.ErrUnsupportedOrderType, errModifyOrderTypeUnsupported, orderType)
	}
	if timeInForce == order.ImmediateOrCancel {
		return fmt.Errorf("%w: %w: %s", order.ErrUnsupportedTimeInForce, errModifyOrderTypeUnsupported, timeInForce)
	}
	_, err := formatOrderTimeInForce(timeInForce)
	return err
}

// CancelOrder cancels one order by order ID or client order ID
func (e *Exchange) CancelOrder(ctx context.Context, cancel *order.Cancel) error {
	if cancel == nil {
		return order.ErrCancelOrderIsNil
	}
	_, err := e.cancelOrderBatch(ctx, []order.Cancel{*cancel})
	return err
}

// CancelBatchOrders cancels a batch of orders by order ID or client order ID
func (e *Exchange) CancelBatchOrders(ctx context.Context, cancels []order.Cancel) (*order.CancelBatchResponse, error) {
	statuses, err := e.cancelOrderBatch(ctx, cancels)
	return &order.CancelBatchResponse{Status: statuses}, err
}

// CancelAllOrders cancels every open order of an asset type, optionally limited to one pair; HIP-4 outcome orders, which
// the first perpetual DEX lists beside spot and perpetual orders, are left open, as no supported asset type covers them
func (e *Exchange) CancelAllOrders(ctx context.Context, cancel *order.Cancel) (order.CancelAllResponse, error) {
	if cancel == nil {
		return order.CancelAllResponse{}, order.ErrCancelOrderIsNil
	}
	if !cancel.AssetType.IsValid() || !e.SupportsAsset(cancel.AssetType) {
		return order.CancelAllResponse{}, fmt.Errorf("%w: %s", asset.ErrNotSupported, cancel.AssetType)
	}
	address, err := e.getWatchAddress(ctx)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	dexes, err := e.openOrderDEXes(ctx, cancel.AssetType)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	var openOrders []BasicOrderResponse
	for _, dex := range dexes {
		orders, err := e.GetOpenOrders(ctx, address, dex)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		openOrders = append(openOrders, orders...)
	}
	cancels := make([]order.Cancel, 0, len(openOrders))
	var errs error
	for i := range openOrders {
		if isOutcomeCoin(openOrders[i].Coin) {
			continue
		}
		mapping, a, err := e.getPairMappingByCoin(ctx, openOrders[i].Coin)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("order %d: %w", openOrders[i].OrderID, err))
			continue
		}
		if a != cancel.AssetType || (!cancel.Pair.IsEmpty() && !mapping.pair.Equal(cancel.Pair)) {
			continue
		}
		cancels = append(cancels, order.Cancel{
			OrderID:   strconv.FormatUint(openOrders[i].OrderID, 10),
			AssetType: a,
			Pair:      mapping.pair,
		})
	}
	statuses := make(map[string]string, len(cancels))
	for chunk := range slices.Chunk(cancels, maximumActionBatchSize) {
		batchStatuses, err := e.cancelOrderBatch(ctx, chunk)
		maps.Copy(statuses, batchStatuses)
		errs = common.AppendError(errs, err)
	}
	return order.CancelAllResponse{Status: statuses}, errs
}

// GetOrderInfo returns one order by order ID or client order ID
// orderStatus returns only an order's latest status change, and once a trigger order has triggered and filled that
// change has no trigger price and gives the trigger time as the order's timestamp, so such an order is returned without
// its trigger price and dated when it triggered; GetOrderHistory recovers both from the order's earlier changes
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if a != asset.Empty && !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	address, err := e.getWatchAddress(ctx)
	if err != nil {
		return nil, err
	}
	statusRequest := &OrderStatusRequest{User: address}
	if statusRequest.OrderID, err = strconv.ParseUint(orderID, 10, 64); err != nil {
		statusRequest.ClientOrderID = orderID
	}
	response, err := e.GetOrderStatus(ctx, statusRequest)
	if err != nil {
		return nil, err
	}
	if response.Status != "order" || response.Order == nil {
		// orderStatus can report an older order that is still open as unknown, so such an order is looked up among the
		// open orders
		return e.findOpenOrder(ctx, statusRequest, pair, a)
	}
	converted, err := e.convertOrder(ctx, &response.Order.Order, response.Order.Status, response.Order.StatusTimestamp.Time())
	if err != nil {
		return nil, err
	}
	if (!pair.IsEmpty() && !converted.Pair.Equal(pair)) || (a != asset.Empty && converted.AssetType != a) {
		return nil, order.ErrOrderNotFound
	}
	return &converted, nil
}

// findOpenOrder returns the open order a status request identifies from the account's open orders of the asset type, or
// of every supported asset type when none is given
func (e *Exchange) findOpenOrder(ctx context.Context, identifier *OrderStatusRequest, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	assets := []asset.Item{asset.PerpetualContract, asset.Spot}
	if a != asset.Empty {
		assets = []asset.Item{a}
	}
	var pairs currency.Pairs
	if !pair.IsEmpty() {
		pairs = currency.Pairs{pair}
	}
	orderID := strconv.FormatUint(identifier.OrderID, 10)
	// An order of one asset type that fails to convert must not hide the requested order of another
	var errs error
	for _, item := range assets {
		orders, err := e.GetActiveOrders(ctx, &order.MultiOrderRequest{AssetType: item, Pairs: pairs, Side: order.AnySide, Type: order.AnyType})
		for i := range orders {
			// The listing's pair filter also matches the reversed pair, which an order status lookup does not
			if !pair.IsEmpty() && !orders[i].Pair.Equal(pair) {
				continue
			}
			if (identifier.OrderID != 0 && orders[i].OrderID == orderID) || (identifier.ClientOrderID != "" && strings.EqualFold(orders[i].ClientOrderID, identifier.ClientOrderID)) {
				return &orders[i], nil
			}
		}
		errs = common.AppendError(errs, err)
	}
	if errs != nil {
		return nil, errs
	}
	return nil, order.ErrOrderNotFound
}

// GetDepositAddress is not supported: Hyperliquid credits bridge deposits to the address that sent them, so there is no
// deposit address to return
func (e *Exchange) GetDepositAddress(context.Context, currency.Code, string, string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds sends USDC to another Hyperliquid account, and requires InternalTransfer to be set: the
// Arbitrum bridge that withdrew funds from Hyperliquid is deprecated in favour of Circle's CCTP, which is not wrapped
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	if !withdrawRequest.InternalTransfer {
		return nil, fmt.Errorf("%w: only internal transfers to other Hyperliquid accounts", common.ErrFunctionNotSupported)
	}
	if !withdrawRequest.Currency.Equal(currency.USDC) {
		return nil, fmt.Errorf("%w: %s", errTransferCurrencyInvalid, withdrawRequest.Currency)
	}
	if strings.TrimSpace(withdrawRequest.Crypto.AddressTag) != "" {
		return nil, errWithdrawalAddressTag
	}
	if withdrawRequest.Crypto.FeeAmount != 0 {
		return nil, errWithdrawalFeeInput
	}
	nonce, err := e.SendCoreUSDC(ctx, &SendCoreUSDCRequest{Destination: withdrawRequest.Crypto.Address, Amount: withdrawRequest.Amount})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   e.Name,
		ID:     strconv.FormatUint(nonce, 10),
		Status: "submitted",
	}, nil
}

// WithdrawFiatFunds is not supported by Hyperliquid
func (e *Exchange) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank is not supported by Hyperliquid
func (e *Exchange) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders returns the configured account's open orders of an asset type, filtered by the request
// frontendOpenOrders returns every open order of most accounts, but only 100 of a DEX's orders, not necessarily the
// newest, for an account with very many, such as one of HLP's child vaults, so a DEX at that count has the rest added
// from openOrders, which omits their order type, time in force and trigger price. Position TP/SL orders, which
// Hyperliquid lists without a size, take the size of the position they close, and HIP-4 outcome orders are skipped, as
// no supported asset type covers them. A DEX whose orders cannot all be read is reported in the error, beside the orders
// that could be read
func (e *Exchange) GetActiveOrders(ctx context.Context, filter *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}
	if !e.SupportsAsset(filter.AssetType) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, filter.AssetType)
	}
	address, err := e.getWatchAddress(ctx)
	if err != nil {
		return nil, err
	}
	dexes, err := e.activeOrderDEXes(ctx, filter)
	if err != nil {
		return nil, err
	}
	var converted []order.Detail
	var errs error
	for _, dex := range dexes {
		frontendOrders, err := e.GetFrontendOpenOrders(ctx, address, dex)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("DEX %q open orders: %w", dex, err))
			continue
		}
		positionCoins := make(map[int]string)
		listed := make(map[uint64]struct{}, len(frontendOrders))
		for i := range frontendOrders {
			listed[frontendOrders[i].OrderID] = struct{}{}
			if isOutcomeCoin(frontendOrders[i].Coin) {
				continue
			}
			detail, err := e.convertOrder(ctx, &frontendOrders[i], "open", time.Time{})
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("order %d: %w", frontendOrders[i].OrderID, err))
				continue
			}
			if detail.AssetType != filter.AssetType {
				continue
			}
			if frontendOrders[i].IsPositionTPSL && detail.Amount == 0 {
				positionCoins[len(converted)] = frontendOrders[i].Coin
			}
			converted = append(converted, detail)
		}
		if len(frontendOrders) >= maximumFrontendOpenOrders {
			basicOrders, err := e.GetOpenOrders(ctx, address, dex)
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("DEX %q open orders beyond the frontend limit: %w", dex, err))
			}
			for i := range basicOrders {
				if _, ok := listed[basicOrders[i].OrderID]; ok || isOutcomeCoin(basicOrders[i].Coin) {
					continue
				}
				detail, err := e.convertOpenOrder(ctx, &basicOrders[i])
				if err != nil {
					errs = common.AppendError(errs, fmt.Errorf("order %d: %w", basicOrders[i].OrderID, err))
					continue
				}
				if detail.AssetType != filter.AssetType {
					continue
				}
				// openOrders omits isPositionTpsl, but only a position TP/SL order rests without a size
				if detail.Amount == 0 {
					positionCoins[len(converted)] = basicOrders[i].Coin
				}
				converted = append(converted, detail)
			}
		}
		if len(positionCoins) != 0 {
			if err := e.sizePositionOrders(ctx, address, dex, converted, positionCoins); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("DEX %q position TP/SL sizes: %w", dex, err))
			}
		}
	}
	return filter.Filter(e.Name, converted), errs
}

// activeOrderDEXes returns the perpetual DEXs whose open orders GetActiveOrders reads for a request: the first DEX, which
// also lists spot orders, for spot; for perpetuals, only the DEXs of the requested pairs, so polling a few pairs reads
// few DEXs, or every DEX when the request names no pairs or a pair whose market is not cached
func (e *Exchange) activeOrderDEXes(ctx context.Context, filter *order.MultiOrderRequest) ([]string, error) {
	if filter.AssetType != asset.PerpetualContract || len(filter.Pairs) == 0 {
		return e.openOrderDEXes(ctx, filter.AssetType)
	}
	dexes := make([]string, 0, len(filter.Pairs))
	for _, p := range filter.Pairs {
		mapping, ok := e.lookupPairMapping(p, filter.AssetType)
		if !ok {
			return e.openOrderDEXes(ctx, filter.AssetType)
		}
		if !slices.Contains(dexes, mapping.dex) {
			dexes = append(dexes, mapping.dex)
		}
	}
	return dexes, nil
}

// sizePositionOrders sets the size of each position TP/SL order, which Hyperliquid lists without one, to the size of
// the position it closes on the DEX; positionCoins holds each such order's coin by its index in orders, and an order
// whose position has closed is left without a size
func (e *Exchange) sizePositionOrders(ctx context.Context, address, dex string, orders []order.Detail, positionCoins map[int]string) error {
	state, err := e.GetClearinghouseState(ctx, address, dex)
	if err != nil {
		return err
	}
	sizes := make(map[string]float64, len(state.AssetPositions))
	for i := range state.AssetPositions {
		sizes[state.AssetPositions[i].Position.Coin] = math.Abs(state.AssetPositions[i].Position.SignedSize.Float64())
	}
	for index, coin := range positionCoins {
		orders[index].Amount = sizes[coin]
		orders[index].RemainingAmount = sizes[coin]
	}
	return nil
}

// convertOpenOrder converts an open order from openOrders, whose order type and time in force are unknown
func (e *Exchange) convertOpenOrder(ctx context.Context, source *BasicOrderResponse) (order.Detail, error) {
	mapping, a, err := e.getPairMappingByCoin(ctx, source.Coin)
	if err != nil {
		return order.Detail{}, err
	}
	detail, err := e.convertBasicOrder(source, "open", time.Time{}, mapping.pair, a)
	if err != nil {
		return order.Detail{}, err
	}
	detail.ReduceOnly = source.ReduceOnly
	return detail, nil
}

// GetOrderHistory returns the latest status of the configured account's orders of an asset type, filtered by the
// request
// Hyperliquid returns the account's 2000 most recent status changes, which list an order once per change, so each order
// is converted once, from its latest status, with the trigger price and placement time of its earlier changes; HIP-4
// outcome orders are skipped, as no supported asset type covers them
func (e *Exchange) GetOrderHistory(ctx context.Context, filter *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}
	if !e.SupportsAsset(filter.AssetType) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, filter.AssetType)
	}
	address, err := e.getWatchAddress(ctx)
	if err != nil {
		return nil, err
	}
	statusChanges, err := e.GetHistoricalOrders(ctx, address)
	if err != nil {
		return nil, err
	}
	history := latestOrderStatuses(statusChanges)
	converted := make([]order.Detail, 0, len(history))
	var errs error
	for i := range history {
		if isOutcomeCoin(history[i].Order.Coin) {
			continue
		}
		detail, err := e.convertOrder(ctx, &history[i].Order, history[i].Status, history[i].StatusTimestamp.Time())
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("order %d: %w", history[i].Order.OrderID, err))
			continue
		}
		if detail.AssetType == filter.AssetType {
			converted = append(converted, detail)
		}
	}
	return filter.Filter(e.Name, converted), errs
}

// latestOrderStatuses keeps each order's latest status change, in the order the orders are first listed; at the same
// time the later status in an order's lifecycle wins, as Hyperliquid can list a market order's open and filled changes,
// or a trigger order's triggered and filled changes, at one time in either order
// A triggered order's filled change has no trigger price and gives the trigger time as the order's timestamp, so the
// kept change takes the trigger price and the earliest timestamp, the placement time, of the order's other changes
func latestOrderStatuses(statusChanges []HistoricalOrderResponse) []HistoricalOrderResponse {
	latest := make([]HistoricalOrderResponse, 0, len(statusChanges))
	positions := make(map[uint64]int, len(statusChanges))
	for i := range statusChanges {
		change := statusChanges[i]
		position, seen := positions[change.Order.OrderID]
		if !seen {
			positions[change.Order.OrderID] = len(latest)
			latest = append(latest, change)
			continue
		}
		kept := &latest[position]
		changed, keptTime := change.StatusTimestamp.Time(), kept.StatusTimestamp.Time()
		if changed.After(keptTime) || (changed.Equal(keptTime) && orderStatusRank(change.Status) > orderStatusRank(kept.Status)) {
			change, *kept = *kept, change
		}
		kept.Order.TriggerPrice = cmp.Or(kept.Order.TriggerPrice, change.Order.TriggerPrice)
		if change.Order.Timestamp.Time().Before(kept.Order.Timestamp.Time()) {
			kept.Order.Timestamp = change.Order.Timestamp
		}
	}
	return latest
}

// orderStatusRank orders statuses through an order's lifecycle: open, then triggered, then any final status
func orderStatusRank(status string) int {
	switch status {
	case "open":
		return 0
	case "triggered":
		return 1
	default:
		return 2
	}
}

// GetFeeByType returns a maker or taker trade fee estimate from the account's effective rates and referral discount when
// credentials are valid, and from the base tier rates otherwise, scaled as the fees page's formula for developers scales
// them; Hyperliquid documents no aligned quote assets on mainnet, so their discounts are not applied
// A pair enabled for both spot and perpetuals cannot identify its fee schedule, so it fails rather than guessing
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, common.ErrNilPointer
	}
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee, exchange.OfflineTradeFee:
	default:
		return 0, common.ErrFunctionNotSupported
	}
	if feeBuilder.Pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	if feeBuilder.PurchasePrice < 0 || feeBuilder.Amount < 0 ||
		math.IsNaN(feeBuilder.PurchasePrice) || math.IsNaN(feeBuilder.Amount) ||
		math.IsInf(feeBuilder.PurchasePrice, 0) || math.IsInf(feeBuilder.Amount, 0) {
		return 0, fmt.Errorf("%w: invalid fee notional", order.ErrAmountIsInvalid)
	}
	var spot, perpetual bool
	if pairs, err := e.GetEnabledPairs(asset.Spot); err == nil {
		spot = pairs.Contains(feeBuilder.Pair, true)
	}
	if pairs, err := e.GetEnabledPairs(asset.PerpetualContract); err == nil {
		perpetual = pairs.Contains(feeBuilder.Pair, true)
	}
	if !spot && !perpetual {
		_, spot = e.lookupPairMapping(feeBuilder.Pair, asset.Spot)
		_, perpetual = e.lookupPairMapping(feeBuilder.Pair, asset.PerpetualContract)
	}
	if spot == perpetual {
		if spot {
			return 0, fmt.Errorf("%w: fee request does not identify spot or perpetual asset", errAmbiguousCoinMapping)
		}
		return 0, fmt.Errorf("%w: %s", errPairMappingNotFound, feeBuilder.Pair)
	}
	a := asset.Spot
	if perpetual {
		a = asset.PerpetualContract
	}
	mapping, err := e.getPairMapping(ctx, feeBuilder.Pair, a)
	if err != nil {
		return 0, err
	}
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	var rate, referralDiscount float64
	if feeBuilder.FeeType == exchange.OfflineTradeFee {
		switch {
		case spot && feeBuilder.IsMaker:
			rate = spotMakerBaseFeeRate
		case spot:
			rate = spotTakerBaseFeeRate
		case feeBuilder.IsMaker:
			rate = perpetualMakerBaseFeeRate
		default:
			rate = perpetualTakerBaseFeeRate
		}
	} else {
		address, err := e.getWatchAddress(ctx)
		if err != nil {
			return 0, err
		}
		fees, err := e.GetUserFees(ctx, address)
		if err != nil {
			return 0, err
		}
		switch {
		case spot && feeBuilder.IsMaker:
			rate = fees.UserSpotMakerFeeRate.Float64()
		case spot:
			rate = fees.UserSpotTakerFeeRate.Float64()
		case feeBuilder.IsMaker:
			rate = fees.UserMakerFeeRate.Float64()
		default:
			rate = fees.UserTakerFeeRate.Float64()
		}
		referralDiscount = fees.ActiveReferralDiscount.Float64()
	}
	stablePair := spot && e.isSpotQuoteAsset(mapping.pair.Base)
	return feeBuilder.PurchasePrice * feeBuilder.Amount * tradeFeeRate(rate, referralDiscount, feeBuilder.IsMaker, stablePair, &mapping), nil
}

// tradeFeeRate scales a maker or taker rate as the fees page's formula for developers does: a spot pair between two spot
// quote assets pays a fifth, and a builder-deployed market in growth mode a tenth, of any rate, rebates included, while a
// builder-deployed market's deployer fee scale raises, and the account's referral discount cuts, any rate but a maker
// rebate
func tradeFeeRate(rate, referralDiscount float64, isMaker, stablePair bool, mapping *pairMapping) float64 {
	if stablePair {
		rate *= stablePairFeeMultiplier
	}
	if mapping.growthMode {
		rate *= growthModeFeeMultiplier
	}
	if isMaker && rate <= 0 {
		return rate
	}
	return rate * (max(1, mapping.deployerFeeScale) + mapping.deployerFeeScale) * (1 - referralDiscount)
}

// isSpotQuoteAsset reports whether a token is a spot quote asset; spot metadata does not flag quote assets, so a token
// counts as one when it quotes a cached spot market
func (e *Exchange) isSpotQuoteAsset(token currency.Code) bool {
	e.pairMappingsMu.RLock()
	defer e.pairMappingsMu.RUnlock()
	for i := range e.pairMappings[asset.Spot] {
		if e.pairMappings[asset.Spot][i].pair.Quote.Equal(token) {
			return true
		}
	}
	return false
}

// ValidateAPICredentials validates the configured account, its vault or subaccount authority and any signing key
func (e *Exchange) ValidateAPICredentials(ctx context.Context, a asset.Item) error {
	if a != asset.Empty && !e.SupportsAsset(a) {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	credentials, err := e.GetCredentials(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	if _, err := e.validateCachedAuthority(ctx, credentials, true); err != nil {
		return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	return nil
}

// SetLeverage sets a perpetual market's margin mode and whole-number leverage, on any perpetual DEX
func (e *Exchange) SetLeverage(ctx context.Context, a asset.Item, p currency.Pair, marginType margin.Type, amount float64, _ order.Side) error {
	if a != asset.PerpetualContract {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	mapping, err := e.getPairMapping(ctx, p, a)
	if err != nil {
		return err
	}
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) || math.Trunc(amount) != amount ||
		mapping.maxLeverage == 0 || amount > float64(mapping.maxLeverage) {
		return fmt.Errorf("%w: requested %v, maximum %d", errInvalidLeverage, amount, mapping.maxLeverage)
	}
	var isCross bool
	switch marginType {
	case margin.Unset, margin.Multi:
		if mapping.onlyIsolated {
			return fmt.Errorf("%w: %w", margin.ErrMarginTypeUnsupported, errCrossMarginUnavailable)
		}
		isCross = true
	case margin.Isolated:
	default:
		return fmt.Errorf("%w: %s", margin.ErrMarginTypeUnsupported, marginType)
	}
	return e.UpdateLeverage(ctx, &UpdateLeverageRequest{Asset: mapping.assetID, IsCross: isCross, Leverage: uint64(amount)})
}

// GetLeverage returns the configured cross or isolated leverage of a perpetual market
func (e *Exchange) GetLeverage(ctx context.Context, a asset.Item, p currency.Pair, marginType margin.Type, _ order.Side) (float64, error) {
	if a != asset.PerpetualContract {
		return 0, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	mapping, err := e.getPairMapping(ctx, p, a)
	if err != nil {
		return 0, err
	}
	address, err := e.getWatchAddress(ctx)
	if err != nil {
		return 0, err
	}
	data, err := e.GetActiveAssetData(ctx, address, mapping.coin)
	if err != nil {
		return 0, err
	}
	switch strings.ToLower(data.Leverage.Type) {
	case "cross":
		if marginType != margin.Unset && marginType != margin.Multi {
			return 0, fmt.Errorf("%w: requested %s, account uses cross", margin.ErrMarginTypeUnsupported, marginType)
		}
	case "isolated":
		if marginType != margin.Unset && marginType != margin.Isolated {
			return 0, fmt.Errorf("%w: requested %s, account uses isolated", margin.ErrMarginTypeUnsupported, marginType)
		}
	default:
		return 0, fmt.Errorf("%w: %q", margin.ErrMarginTypeUnsupported, data.Leverage.Type)
	}
	if data.Leverage.Value == 0 {
		return 0, errInvalidLeverage
	}
	return float64(data.Leverage.Value), nil
}

// GetFuturesContractDetails is not supported by this integration
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the current hourly rates of one or all active perpetual markets
func (e *Exchange) GetLatestFundingRates(ctx context.Context, arg *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Asset != asset.PerpetualContract {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, arg.Asset)
	}
	if arg.IncludePredictedRate {
		return nil, fmt.Errorf("%w: predicted funding rates", common.ErrFunctionNotSupported)
	}
	var mappings []pairMapping
	if arg.Pair.IsEmpty() {
		var err error
		if mappings, err = e.getPerpetualPairMappings(ctx); err != nil {
			return nil, err
		}
	} else {
		mapping, err := e.getPairMapping(ctx, arg.Pair, arg.Asset)
		if err != nil {
			return nil, err
		}
		mappings = []pairMapping{mapping}
	}
	contexts, err := e.getPerpetualAssetContexts(ctx, mappings)
	if err != nil {
		return nil, err
	}
	checked := time.Now().UTC()
	responses := make([]fundingrate.LatestRateResponse, len(mappings))
	for i := range mappings {
		responses[i] = fundingrate.LatestRateResponse{
			Exchange:    e.Name,
			Asset:       asset.PerpetualContract,
			Pair:        mappings[i].pair,
			TimeChecked: checked,
			LatestRate: fundingrate.Rate{
				Time: checked.Truncate(time.Hour),
				Rate: contexts[i].Funding.Decimal(),
			},
			TimeOfNextRate: checked.Truncate(time.Hour).Add(time.Hour),
		}
	}
	if len(responses) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}
	return responses, nil
}

// GetHistoricalFundingRates returns the hourly funding rates of one perpetual market, paging through the range
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, arg *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Asset != asset.PerpetualContract {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, arg.Asset)
	}
	if arg.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.IncludePredictedRate || arg.IncludePayments {
		return nil, fmt.Errorf("%w: predicted rates and account payments", common.ErrFunctionNotSupported)
	}
	if err := common.StartEndTimeCheck(arg.StartDate, arg.EndDate); err != nil {
		return nil, err
	}
	mapping, err := e.getPairMapping(ctx, arg.Pair, arg.Asset)
	if err != nil {
		return nil, err
	}
	if !arg.PaymentCurrency.IsEmpty() && !arg.PaymentCurrency.Equal(mapping.pair.Quote) {
		return nil, fmt.Errorf("%w: funding payment currency %s", asset.ErrNotSupported, arg.PaymentCurrency)
	}
	result := &fundingrate.HistoricalRates{
		Exchange:        e.Name,
		Asset:           arg.Asset,
		Pair:            mapping.pair,
		StartDate:       arg.StartDate,
		EndDate:         arg.EndDate,
		PaymentCurrency: mapping.pair.Quote,
	}
	cursor := arg.StartDate
	for !cursor.After(arg.EndDate) {
		records, err := e.GetFundingHistory(ctx, &FundingHistoryRequest{Coin: mapping.coin, StartTime: cursor, EndTime: arg.EndDate})
		if err != nil {
			return nil, err
		}
		if len(records) > maximumFundingHistoryCount {
			return nil, fmt.Errorf("%w: funding page has %d entries, maximum %d", errUnexpectedResponseLength, len(records), maximumFundingHistoryCount)
		}
		for i := range records {
			recordTime := records[i].Time.Time().UTC()
			if records[i].Coin != mapping.coin || recordTime.Before(cursor) || recordTime.After(arg.EndDate) ||
				(i > 0 && !recordTime.After(records[i-1].Time.Time())) {
				return nil, fmt.Errorf("%w: funding history record %d is out of order, outside the range or of another coin", errMalformedResponse, i)
			}
			result.FundingRates = append(result.FundingRates, fundingrate.Rate{
				Time: recordTime,
				Rate: records[i].FundingRate.Decimal(),
			})
		}
		if len(records) < maximumFundingHistoryCount {
			break
		}
		cursor = records[len(records)-1].Time.Time().UTC().Add(time.Millisecond)
	}
	if len(result.FundingRates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}
	result.LatestRate = result.FundingRates[len(result.FundingRates)-1]
	return result, nil
}

// GetOpenInterest returns the current base unit open interest of selected or all active perpetual markets
func (e *Exchange) GetOpenInterest(ctx context.Context, requested ...key.PairAsset) ([]futures.OpenInterest, error) {
	var mappings []pairMapping
	if len(requested) == 0 {
		var err error
		if mappings, err = e.getPerpetualPairMappings(ctx); err != nil {
			return nil, err
		}
	} else {
		mappings = make([]pairMapping, len(requested))
		for i := range requested {
			if requested[i].Asset != asset.PerpetualContract {
				return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, requested[i].Asset)
			}
			var err error
			if mappings[i], err = e.getPairMapping(ctx, requested[i].Pair(), requested[i].Asset); err != nil {
				return nil, err
			}
		}
	}
	contexts, err := e.getPerpetualAssetContexts(ctx, mappings)
	if err != nil {
		return nil, err
	}
	result := make([]futures.OpenInterest, len(mappings))
	for i := range mappings {
		result[i] = futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, asset.PerpetualContract, mappings[i].pair),
			OpenInterest: contexts[i].OpenInterest.Float64(),
		}
	}
	return result, nil
}

// GetCurrencyTradeURL is not supported by this integration
func (e *Exchange) GetCurrencyTradeURL(context.Context, asset.Item, currency.Pair) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits is not yet implemented by this integration
func (e *Exchange) UpdateOrderExecutionLimits(context.Context, asset.Item) error {
	return common.ErrNotYetImplemented
}

func (e *Exchange) setPairMappings(a asset.Item, mappings []pairMapping) {
	e.pairMappingsMu.Lock()
	if e.pairMappings == nil {
		e.pairMappings = make(map[asset.Item][]pairMapping)
	}
	e.pairMappings[a] = slices.Clone(mappings)
	e.pairMappingsMu.Unlock()
}

func (e *Exchange) getCoin(ctx context.Context, p currency.Pair, a asset.Item) (string, error) {
	mapping, err := e.getPairMapping(ctx, p, a)
	if err != nil {
		return "", err
	}
	return mapping.coin, nil
}

// getPairMapping returns a tradable pair's mapping, fetching the asset's markets when it is not cached; a delisted
// market's pair is refused, so it cannot be traded or subscribed to
func (e *Exchange) getPairMapping(ctx context.Context, p currency.Pair, a asset.Item) (pairMapping, error) {
	if !e.SupportsAsset(a) {
		return pairMapping{}, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if p.IsEmpty() {
		return pairMapping{}, currency.ErrCurrencyPairEmpty
	}
	if mapping, ok := e.lookupPairMapping(p, a); ok {
		return mapping, nil
	}
	mapping, err := e.fetchPairMapping(ctx, p, a)
	if !errors.Is(err, errPairMappingNotFound) || a != asset.PerpetualContract {
		return mapping, err
	}
	e.pairMappingsMu.RLock()
	delisted := slices.ContainsFunc(e.delistedPairMappings, func(m pairMapping) bool { return m.pair.Equal(p) })
	e.pairMappingsMu.RUnlock()
	if delisted {
		return pairMapping{}, fmt.Errorf("%w: %w: %s %s", errPairMappingNotFound, errMarketDelisted, a, p)
	}
	return pairMapping{}, err
}

func (e *Exchange) lookupPairMapping(p currency.Pair, a asset.Item) (pairMapping, bool) {
	e.pairMappingsMu.RLock()
	defer e.pairMappingsMu.RUnlock()
	mappings := e.pairMappings[a]
	for i := range mappings {
		if mappings[i].pair.Equal(p) {
			return mappings[i], true
		}
	}
	return pairMapping{}, false
}

// fetchPairMapping refreshes an asset's mappings for a missing pair; a pair still missing is cached as a miss for a
// minute, so repeated lookups of an unknown pair do not refetch every market
func (e *Exchange) fetchPairMapping(ctx context.Context, p currency.Pair, a asset.Item) (pairMapping, error) {
	e.pairMappingsFetchMu.Lock()
	defer e.pairMappingsFetchMu.Unlock()
	if mapping, ok := e.lookupPairMapping(p, a); ok {
		return mapping, nil
	}
	cacheKey := "pair:" + a.String() + ":" + strings.ToLower(p.String())
	if e.isCachedPairMappingMiss(cacheKey) {
		return pairMapping{}, fmt.Errorf("%w: %s %s", errPairMappingNotFound, a, p)
	}
	if _, err := e.FetchTradablePairs(ctx, a); err != nil {
		return pairMapping{}, err
	}
	if mapping, ok := e.lookupPairMapping(p, a); ok {
		return mapping, nil
	}
	e.cachePairMappingMiss(cacheKey, time.Now().Add(pairMappingMissCacheDuration))
	return pairMapping{}, fmt.Errorf("%w: %s %s", errPairMappingNotFound, a, p)
}

// getPairMappingByCoin returns the mapping and asset type of a coin, fetching all markets when it is not cached
func (e *Exchange) getPairMappingByCoin(ctx context.Context, coin string) (pairMapping, asset.Item, error) {
	if strings.TrimSpace(coin) == "" {
		return pairMapping{}, asset.Empty, errCoinRequired
	}
	mapping, a, err := e.lookupPairMappingByCoin(coin)
	if err == nil || errors.Is(err, errAmbiguousCoinMapping) {
		return mapping, a, err
	}
	return e.fetchPairMappingByCoin(ctx, coin)
}

// lookupPairMappingByCoin returns the cached mapping and asset type of a coin; a coin no tradable market uses may be a
// delisted perpetual market's, whose orders and fills still convert
func (e *Exchange) lookupPairMappingByCoin(coin string) (pairMapping, asset.Item, error) {
	e.pairMappingsMu.RLock()
	defer e.pairMappingsMu.RUnlock()
	var found pairMapping
	var foundAsset asset.Item
	var count int
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		mappings := e.pairMappings[a]
		for i := range mappings {
			if mappings[i].coin == coin {
				found = mappings[i]
				foundAsset = a
				count++
			}
		}
	}
	if count == 0 {
		for i := range e.delistedPairMappings {
			if e.delistedPairMappings[i].coin == coin {
				found = e.delistedPairMappings[i]
				foundAsset = asset.PerpetualContract
				count++
			}
		}
	}
	switch count {
	case 0:
		return pairMapping{}, asset.Empty, fmt.Errorf("%w: %s", errPairMappingNotFound, coin)
	case 1:
		return found, foundAsset, nil
	default:
		return pairMapping{}, asset.Empty, fmt.Errorf("%w: %s", errAmbiguousCoinMapping, coin)
	}
}

// fetchPairMappingByCoin refetches every market for a coin that is not cached, unless a coin lookup refetched them
// within the last minute: every refetch maps every coin, so lookups of many unknown coins cost at most one refetch a
// minute, and a coin still missing is cached as a miss until the next refetch is due
func (e *Exchange) fetchPairMappingByCoin(ctx context.Context, coin string) (pairMapping, asset.Item, error) {
	e.pairMappingsFetchMu.Lock()
	defer e.pairMappingsFetchMu.Unlock()
	mapping, a, err := e.lookupPairMappingByCoin(coin)
	if err == nil || errors.Is(err, errAmbiguousCoinMapping) {
		return mapping, a, err
	}
	cacheKey := "coin:" + strings.ToLower(coin)
	if e.isCachedPairMappingMiss(cacheKey) {
		return pairMapping{}, asset.Empty, fmt.Errorf("%w: %s", errPairMappingNotFound, coin)
	}
	if time.Since(e.pairMappingsRefreshed) >= pairMappingMissCacheDuration {
		for _, item := range []asset.Item{asset.Spot, asset.PerpetualContract} {
			if !e.SupportsAsset(item) {
				continue
			}
			if _, err := e.FetchTradablePairs(ctx, item); err != nil {
				return pairMapping{}, asset.Empty, err
			}
		}
		e.pairMappingsRefreshed = time.Now()
		mapping, a, err = e.lookupPairMappingByCoin(coin)
	}
	if errors.Is(err, errPairMappingNotFound) {
		e.cachePairMappingMiss(cacheKey, e.pairMappingsRefreshed.Add(pairMappingMissCacheDuration))
	}
	return mapping, a, err
}

// isCachedPairMappingMiss reports whether a lookup miss is still cached; callers hold pairMappingsFetchMu
func (e *Exchange) isCachedPairMappingMiss(cacheKey string) bool {
	expiry, ok := e.pairMappingMisses[cacheKey]
	if !ok {
		return false
	}
	if time.Now().Before(expiry) {
		return true
	}
	delete(e.pairMappingMisses, cacheKey)
	return false
}

// cachePairMappingMiss records a lookup miss until it expires; callers hold pairMappingsFetchMu
func (e *Exchange) cachePairMappingMiss(cacheKey string, expiry time.Time) {
	if e.pairMappingMisses == nil {
		e.pairMappingMisses = make(map[string]time.Time)
	}
	e.pairMappingMisses[cacheKey] = expiry
}

// getPerpetualPairMappings returns every cached perpetual mapping, fetching the markets when none are cached
func (e *Exchange) getPerpetualPairMappings(ctx context.Context) ([]pairMapping, error) {
	e.pairMappingsMu.RLock()
	mappings := slices.Clone(e.pairMappings[asset.PerpetualContract])
	e.pairMappingsMu.RUnlock()
	if len(mappings) != 0 {
		return mappings, nil
	}
	if _, err := e.FetchTradablePairs(ctx, asset.PerpetualContract); err != nil {
		return nil, err
	}
	e.pairMappingsMu.RLock()
	defer e.pairMappingsMu.RUnlock()
	return slices.Clone(e.pairMappings[asset.PerpetualContract]), nil
}

// getPerpetualAssetContexts returns the current context of each mapped market, fetching each perpetual DEX once
func (e *Exchange) getPerpetualAssetContexts(ctx context.Context, mappings []pairMapping) ([]PerpetualAssetContext, error) {
	dexContexts := make(map[string]*PerpetualMetadataAndAssetContextsResponse)
	contexts := make([]PerpetualAssetContext, len(mappings))
	for i := range mappings {
		resp, ok := dexContexts[mappings[i].dex]
		if !ok {
			var err error
			if resp, err = e.GetPerpetualMetadataAndAssetContexts(ctx, mappings[i].dex); err != nil {
				return nil, err
			}
			dexContexts[mappings[i].dex] = resp
		}
		index := slices.IndexFunc(resp.Metadata.Universe, func(market PerpetualAssetMetadata) bool {
			return market.Name == mappings[i].coin && !market.IsDelisted
		})
		if index == -1 {
			return nil, fmt.Errorf("%w: %s", errAssetContextNotFound, mappings[i].coin)
		}
		contexts[i] = resp.AssetContexts[index]
	}
	return contexts, nil
}

// getPerpetualDEXNames returns the name of each perpetual DEX by index, with an empty name for the default DEX
func (e *Exchange) getPerpetualDEXNames(ctx context.Context) ([]string, error) {
	dexes, err := e.GetPerpetualDEXs(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(dexes))
	seen := make(map[string]struct{}, len(dexes)-1)
	for i := 1; i < len(dexes); i++ {
		if dexes[i] == nil || strings.TrimSpace(dexes[i].Name) == "" {
			return nil, fmt.Errorf("%w: entry %d has no name", errInvalidPerpetualDEX, i)
		}
		names[i] = strings.TrimSpace(dexes[i].Name)
		if _, exists := seen[names[i]]; exists {
			return nil, fmt.Errorf("%w: duplicate DEX %q", errInvalidPerpetualDEX, names[i])
		}
		seen[names[i]] = struct{}{}
	}
	return names, nil
}

// openOrderDEXes returns the perpetual DEXs whose open orders include an asset type's orders: the first DEX, which also
// lists spot orders, for spot, and every DEX for perpetuals
func (e *Exchange) openOrderDEXes(ctx context.Context, a asset.Item) ([]string, error) {
	switch a {
	case asset.Spot:
		return []string{""}, nil
	case asset.PerpetualContract:
		return e.getPerpetualDEXNames(ctx)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
}

// getUserNonFundingLedgerUpdatesPaginated pages through every ledger update from start until now; Hyperliquid returns
// at most 2000 updates a request, oldest first, so each full page's last timestamp starts the next page, and the
// records at that boundary, which repeat on the next page, are dropped as identical records
func (e *Exchange) getUserNonFundingLedgerUpdatesPaginated(ctx context.Context, user string, start time.Time) ([]UserLedgerUpdateResponse, error) {
	cursor := start
	var result []UserLedgerUpdateResponse
	seen := make(map[string]struct{})
	for {
		page, err := e.GetUserNonFundingLedgerUpdates(ctx, &UserNonFundingLedgerUpdatesRequest{User: user, StartTime: cursor})
		if err != nil {
			return nil, err
		}
		if len(page) > maximumUserLedgerHistoryCount {
			return nil, fmt.Errorf("%w: ledger page has %d entries, maximum %d", errUnexpectedResponseLength, len(page), maximumUserLedgerHistoryCount)
		}
		for i := range page {
			recordTime := page[i].Time.Time()
			if recordTime.Before(cursor) || (i > 0 && recordTime.Before(page[i-1].Time.Time())) {
				return nil, fmt.Errorf("%w: user ledger record %d is out of order or before the cursor", errMalformedResponse, i)
			}
			encoded, _ := json.Marshal(&page[i]) // Ledger updates hold only strings, numbers, booleans and timestamps
			if _, exists := seen[string(encoded)]; !exists {
				seen[string(encoded)] = struct{}{}
				result = append(result, page[i])
			}
		}
		if len(page) < maximumUserLedgerHistoryCount {
			return result, nil
		}
		last := page[len(page)-1].Time.Time()
		if !last.After(cursor) {
			return nil, fmt.Errorf("%w: user ledger cursor did not advance", errMalformedResponse)
		}
		cursor = last
	}
}

// convertUserLedgerUpdate converts an account's ledger update into funding history, reading the amount and currency
// from the fields its delta type populates. Amounts are positive, so the description gives the direction of a class
// transfer and the addresses that of a transfer between accounts. A transfer's update is the same in the sender's ledger
// as in the recipient's, so a fee is reported only to the account that paid it, the update's user when it names one, in
// the movement's entry when the fee is in its currency and in an entry of its own otherwise; a fee without a fee token is
// in USDC, and a native token fee is in HYPE
func (e *Exchange) convertUserLedgerUpdate(account string, update *UserLedgerUpdateResponse) ([]exchange.FundingHistory, error) {
	deltaType := strings.TrimSpace(update.Delta.Type)
	if deltaType == "" {
		return nil, fmt.Errorf("%w: ledger update type is empty", errMalformedResponse)
	}
	ccy := currency.USDC
	amount := update.Delta.USDC.Float64()
	switch {
	case deltaType == "vaultWithdraw":
		amount = update.Delta.NetWithdrawnUSD.Float64()
	case !update.Delta.Token.IsEmpty():
		// Token transfers, staking, borrowing and lending, rewards, gas auction bids and DEX abstraction activations are
		// denominated in their own token
		ccy, amount = update.Delta.Token, update.Delta.Amount.Float64()
	}
	description := deltaType
	switch {
	case deltaType == "accountClassTransfer" && update.Delta.ToPerpetual:
		description = "spot to perpetual"
	case deltaType == "accountClassTransfer":
		description = "perpetual to spot"
	case deltaType == "cStakingTransfer" && update.Delta.IsDeposit:
		description = "spot to staking"
	case deltaType == "cStakingTransfer":
		description = "staking to spot"
	case deltaType == "borrowLend" && update.Delta.InterestAmount != 0:
		description = deltaType + ": " + update.Delta.Operation + " with " + update.Delta.InterestAmount.String() + " interest"
	case deltaType == "borrowLend":
		description = deltaType + ": " + update.Delta.Operation
	case deltaType == "activateDexAbstraction":
		description = deltaType + ": " + ledgerDEXName(update.Delta.DEX)
	case update.Delta.SourceDEX != "" || update.Delta.DestinationDEX != "":
		description = deltaType + ": " + ledgerDEXName(update.Delta.SourceDEX) + " to " + ledgerDEXName(update.Delta.DestinationDEX)
	}
	entry := exchange.FundingHistory{
		ExchangeName:      e.Name,
		Status:            "processed",
		TransferID:        update.Hash,
		Description:       description,
		Timestamp:         update.Time.Time().UTC(),
		Currency:          ccy.String(),
		Amount:            amount,
		TransferType:      deltaType,
		CryptoToAddress:   update.Delta.Destination,
		CryptoFromAddress: update.Delta.User,
		CryptoTxID:        update.Hash,
	}
	entries := []exchange.FundingHistory{entry}
	if update.Delta.User != "" && !strings.EqualFold(update.Delta.User, account) {
		return entries, nil
	}
	for _, fee := range []struct {
		amount      float64
		ccy         currency.Code
		description string
	}{
		{amount: update.Delta.Fee.Float64(), ccy: update.Delta.FeeToken, description: description + ": fee"},
		{amount: update.Delta.NativeTokenFee.Float64(), ccy: currency.HYPE, description: description + ": native token fee"},
	} {
		if fee.amount == 0 {
			continue
		}
		if fee.ccy.IsEmpty() {
			fee.ccy = currency.USDC
		}
		if fee.ccy.Equal(ccy) {
			entries[0].Fee += fee.amount
			continue
		}
		feeEntry := entry
		feeEntry.Description, feeEntry.Currency, feeEntry.Amount, feeEntry.Fee = fee.description, fee.ccy.String(), 0, fee.amount
		entries = append(entries, feeEntry)
	}
	return entries, nil
}

// ledgerDEXName names a ledger update's DEX, which Hyperliquid leaves empty for the first perpetual DEX
func ledgerDEXName(dex string) string {
	if dex == "" {
		return "perpetual"
	}
	return dex
}

// buildOrderRequest validates an order against its market's precision and builds the order to sign
// Market orders become IOC limit orders bounded by the slippage tolerance from the mid price
func (e *Exchange) buildOrderRequest(ctx context.Context, p currency.Pair, a asset.Item, orderType order.Type, side order.Side, timeInForce order.TimeInForce, amount, price, triggerPrice, slippage float64, reduceOnly bool, clientOrderID string) (OrderRequest, pairMapping, error) {
	if err := validateOrderParameters(a, orderType, timeInForce, price, triggerPrice, slippage, clientOrderID); err != nil {
		return OrderRequest{}, pairMapping{}, err
	}
	mapping, err := e.getPairMapping(ctx, p, a)
	if err != nil {
		return OrderRequest{}, pairMapping{}, err
	}
	if err := validateOrderSize(amount, mapping.sizeDecimals); err != nil {
		return OrderRequest{}, mapping, err
	}
	orderRequest := OrderRequest{
		Asset:         mapping.assetID,
		IsBuy:         side.IsLong(),
		Size:          amount,
		ReduceOnly:    reduceOnly,
		ClientOrderID: strings.ToLower(clientOrderID),
	}
	switch orderType {
	case order.Limit:
		if err := validateLimitPrice(price, a, mapping.sizeDecimals); err != nil {
			return OrderRequest{}, mapping, err
		}
		tif, _ := formatOrderTimeInForce(timeInForce) // validateOrderParameters only accepts a time in force that formats
		orderRequest.Price = price
		orderRequest.Limit = &LimitOrderType{TimeInForce: tif}
	case order.Market:
		mids, err := e.GetAllMids(ctx, mapping.dex)
		if err != nil {
			return OrderRequest{}, mapping, err
		}
		mid, ok := mids[mapping.coin]
		if !ok || mid.Float64() <= 0 {
			return OrderRequest{}, mapping, fmt.Errorf("%w: %s", errMarketMidPriceNotFound, mapping.coin)
		}
		if orderRequest.Price, err = roundMarketPrice(slippagePrice(mid.Float64(), slippage, side), side, a, mapping.sizeDecimals); err != nil {
			return OrderRequest{}, mapping, err
		}
		orderRequest.Limit = &LimitOrderType{TimeInForce: TimeInForceIOC}
	default: // validateOrderParameters only accepts limit, market and trigger order types
		if err := validateLimitPrice(triggerPrice, a, mapping.sizeDecimals); err != nil {
			return OrderRequest{}, mapping, fmt.Errorf("invalid trigger price: %w", err)
		}
		isMarket := isMarketTriggerOrder(orderType)
		if isMarket && price == 0 {
			if price, err = roundMarketPrice(slippagePrice(triggerPrice, slippage, side), side, a, mapping.sizeDecimals); err != nil {
				return OrderRequest{}, mapping, err
			}
		} else if err := validateLimitPrice(price, a, mapping.sizeDecimals); err != nil {
			return OrderRequest{}, mapping, err
		}
		kind := TriggerStopLoss
		if orderType == order.TakeProfit || orderType == order.TakeProfitMarket {
			kind = TriggerTakeProfit
		}
		orderRequest.Price = price
		orderRequest.Trigger = &TriggerOrderType{IsMarket: isMarket, TriggerPrice: triggerPrice, TakeProfitStopLoss: kind}
	}
	return orderRequest, mapping, nil
}

// validateOrderParameters checks the order parameters that need no market data, so an invalid order fails before any
// request
// Hyperliquid documents stop and take orders with reduce-only as an option, and trailing stops that open positions, so
// trigger orders need not be reduce-only; spot trigger orders are refused, as their support is unverified
func validateOrderParameters(a asset.Item, orderType order.Type, timeInForce order.TimeInForce, price, triggerPrice, slippage float64, clientOrderID string) error {
	if clientOrderID != "" {
		if err := validateClientOrderID(clientOrderID); err != nil {
			return err
		}
	}
	if price < 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return errInvalidMarketPrice
	}
	switch orderType {
	case order.Limit, order.Market:
		if triggerPrice != 0 {
			return fmt.Errorf("%w: trigger price requires a trigger order type", errRiskManagementUnsupported)
		}
		if orderType == order.Limit {
			_, err := formatOrderTimeInForce(timeInForce)
			return err
		}
		// Market orders are sent as IOC limit orders, which keep a partial fill, so FOK or a resting order is not honoured
		if timeInForce != order.UnknownTIF && timeInForce != order.ImmediateOrCancel {
			return fmt.Errorf("%w: %s orders are always IOC, got %s", order.ErrUnsupportedTimeInForce, orderType, timeInForce)
		}
	case order.Stop, order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitMarket:
		if a != asset.PerpetualContract {
			return fmt.Errorf("%w: %s", errTriggerOrderPerpetualOnly, a)
		}
		// Trigger orders carry no time in force, and Hyperliquid places a triggered order as GTC
		if timeInForce != order.UnknownTIF && timeInForce != order.GoodTillCancel {
			return fmt.Errorf("%w: %s orders are always GTC, got %s", order.ErrUnsupportedTimeInForce, orderType, timeInForce)
		}
		if math.IsNaN(triggerPrice) || math.IsInf(triggerPrice, 0) {
			return fmt.Errorf("invalid trigger price: %w", errInvalidMarketPrice)
		}
		if triggerPrice <= 0 {
			return errTriggerPriceRequired
		}
		if !isMarketTriggerOrder(orderType) || price != 0 {
			return nil
		}
	default:
		return fmt.Errorf("%w: %s", order.ErrTypeIsInvalid, orderType)
	}
	// Market orders, and market trigger orders without a limit price, are priced by the slippage tolerance
	if slippage <= 0 || slippage >= 1 || math.IsNaN(slippage) {
		return errSlippageTolerance
	}
	return nil
}

// isOutcomeCoin reports whether a coin names a HIP-4 outcome market, which no supported asset type maps; the first
// perpetual DEX lists outcome orders and fills beside spot and perpetual ones
func isOutcomeCoin(coin string) bool {
	return strings.HasPrefix(coin, "#")
}

// isMarketTriggerOrder reports whether a trigger order type executes as a market order once triggered
func isMarketTriggerOrder(orderType order.Type) bool {
	return orderType == order.Stop || orderType == order.StopMarket || orderType == order.TakeProfitMarket
}

// buildOrderRequests builds a submission's order and any TP/SL children, which Hyperliquid triggers on the mark price
func (e *Exchange) buildOrderRequests(ctx context.Context, submit *order.Submit) ([]OrderRequest, pairMapping, string, error) {
	switch submit.Type {
	case order.Stop, order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitMarket:
		if submit.TriggerPriceType != order.MarkPrice {
			return nil, pairMapping{}, "", fmt.Errorf("%w: Hyperliquid triggers use mark price", errRiskManagementUnsupported)
		}
	}
	children, err := riskManagementChildren(submit)
	if err != nil {
		return nil, pairMapping{}, "", err
	}
	slippage := orderSlippage(submit.Type, submit.Price, submit.SlippageTolerance)
	parent, mapping, err := e.buildOrderRequest(ctx, submit.Pair, submit.AssetType, submit.Type, submit.Side, submit.TimeInForce, submit.Amount, submit.Price, submit.TriggerPrice, slippage, submit.ReduceOnly, submit.ClientOrderID)
	if err != nil {
		return nil, mapping, "", err
	}
	if len(children) == 0 {
		return []OrderRequest{parent}, mapping, GroupingNone, nil
	}
	orders := []OrderRequest{parent}
	childSide := order.Buy
	if submit.Side.IsLong() {
		childSide = order.Sell
	}
	for i := range children {
		childOrder, _, err := e.buildOrderRequest(ctx, submit.Pair, submit.AssetType, children[i].orderType, childSide, order.UnknownTIF, submit.Amount, children[i].settings.LimitPrice, children[i].settings.Price, children[i].slippage, true, "")
		if err != nil {
			return nil, mapping, "", err
		}
		orders = append(orders, childOrder)
	}
	return orders, mapping, GroupingNormalTPSL, nil
}

// riskManagementChild is an enabled take-profit or stop-loss child of a submission, with the trigger order type and
// slippage tolerance it is placed with
type riskManagementChild struct {
	settings   order.RiskManagement
	takeProfit bool
	orderType  order.Type
	slippage   float64
}

// riskManagementChildren validates a submission's TP/SL children, which Hyperliquid supports only as mark price
// triggers grouped with a perpetual parent, and returns the enabled ones, take profit first; a market child without a
// limit price is bounded by the submission's slippage tolerance, or the trigger default when none is set
func riskManagementChildren(submit *order.Submit) ([]riskManagementChild, error) {
	risk := &submit.RiskManagementModes
	if !risk.TakeProfit.Enabled && !risk.StopLoss.Enabled && !risk.StopEntry.Enabled {
		return nil, nil
	}
	if submit.AssetType != asset.PerpetualContract || risk.StopEntry.Enabled || (risk.Mode != "" && risk.Mode != GroupingNormalTPSL) {
		return nil, errRiskManagementUnsupported
	}
	var children []riskManagementChild
	for _, child := range []riskManagementChild{{settings: risk.TakeProfit, takeProfit: true}, {settings: risk.StopLoss}} {
		if !child.settings.Enabled {
			continue
		}
		if child.settings.Price <= 0 {
			return nil, errTriggerPriceRequired
		}
		if child.settings.TriggerPriceType != order.MarkPrice {
			return nil, fmt.Errorf("%w: Hyperliquid TP/SL triggers use mark price", errRiskManagementUnsupported)
		}
		var err error
		if child.orderType, err = riskManagementOrderType(child.settings.OrderType, child.takeProfit); err != nil {
			return nil, err
		}
		child.slippage = orderSlippage(child.orderType, child.settings.LimitPrice, submit.SlippageTolerance)
		if err := validateOrderParameters(submit.AssetType, child.orderType, order.UnknownTIF, child.settings.LimitPrice, child.settings.Price, child.slippage, ""); err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	return children, nil
}

// orderSlippage returns the slippage tolerance an order is placed with, defaulting it for an order the tolerance prices:
// a market order, or a market trigger order without a limit price
func orderSlippage(orderType order.Type, price, slippage float64) float64 {
	switch {
	case slippage != 0:
		return slippage
	case orderType == order.Market:
		return defaultMarketSlippage
	case isMarketTriggerOrder(orderType) && price == 0:
		return defaultTriggerSlippage
	}
	return 0
}

// riskManagementOrderType maps a TP/SL child's requested order type to a take-profit or stop-loss trigger type
func riskManagementOrderType(orderType order.Type, takeProfit bool) (order.Type, error) {
	switch {
	case takeProfit && (orderType == order.UnknownType || orderType == order.Market || orderType == order.TakeProfitMarket):
		return order.TakeProfitMarket, nil
	case takeProfit && (orderType == order.Limit || orderType == order.TakeProfit):
		return order.TakeProfit, nil
	case !takeProfit && (orderType == order.UnknownType || orderType == order.Market || orderType == order.Stop || orderType == order.StopMarket):
		return order.StopMarket, nil
	case !takeProfit && (orderType == order.Limit || orderType == order.StopLimit):
		return order.StopLimit, nil
	case takeProfit:
		return order.UnknownType, fmt.Errorf("%w: take-profit child type %s", errRiskManagementUnsupported, orderType)
	default:
		return order.UnknownType, fmt.Errorf("%w: stop-loss child type %s", errRiskManagementUnsupported, orderType)
	}
}

// cancelOrderBatch cancels orders by order ID and by client order ID, and maps each identifier to its outcome
func (e *Exchange) cancelOrderBatch(ctx context.Context, cancels []order.Cancel) (map[string]string, error) {
	if len(cancels) == 0 {
		return map[string]string{}, nil
	}
	if len(cancels) > maximumActionBatchSize {
		return nil, fmt.Errorf("%w: maximum %d, got %d", errActionBatchTooLarge, maximumActionBatchSize, len(cancels))
	}
	// Every identifier is checked before any market lookup, so an invalid batch sends no request
	ids := make([]uint64, len(cancels))
	for i := range cancels {
		if err := cancels[i].Validate(cancels[i].PairAssetRequired()); err != nil {
			return nil, err
		}
		switch {
		case cancels[i].OrderID != "":
			id, err := strconv.ParseUint(cancels[i].OrderID, 10, 64)
			if err != nil || id == 0 {
				return nil, fmt.Errorf("%w: %q", order.ErrOrderIDNotSet, cancels[i].OrderID)
			}
			ids[i] = id
		case cancels[i].ClientOrderID != "":
			if err := validateClientOrderID(cancels[i].ClientOrderID); err != nil {
				return nil, err
			}
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}
	var byOrderID []CancelRequest
	var orderIDs []string
	var byClientOrderID []CancelByClientOrderIDRequest
	var clientOrderIDs []string
	for i := range cancels {
		mapping, err := e.getPairMapping(ctx, cancels[i].Pair, cancels[i].AssetType)
		if err != nil {
			return nil, err
		}
		if ids[i] != 0 {
			byOrderID = append(byOrderID, CancelRequest{Asset: mapping.assetID, OrderID: ids[i]})
			orderIDs = append(orderIDs, cancels[i].OrderID)
			continue
		}
		byClientOrderID = append(byClientOrderID, CancelByClientOrderIDRequest{Asset: mapping.assetID, ClientOrderID: cancels[i].ClientOrderID})
		clientOrderIDs = append(clientOrderIDs, cancels[i].ClientOrderID)
	}
	statuses := make(map[string]string, len(cancels))
	var errs error
	if len(byOrderID) != 0 {
		results, err := e.CancelOrders(ctx, &CancelOrdersRequest{Cancels: byOrderID})
		errs = common.AppendError(errs, common.AppendError(err, recordCancelStatuses(statuses, orderIDs, results)))
	}
	if len(byClientOrderID) != 0 {
		results, err := e.CancelOrdersByClientOrderID(ctx, &CancelOrdersByClientOrderIDRequest{Cancels: byClientOrderID})
		errs = common.AppendError(errs, common.AppendError(err, recordCancelStatuses(statuses, clientOrderIDs, results)))
	}
	return statuses, errs
}

// recordCancelStatuses maps each identifier to success or its error, and returns the failed cancels as errors
func recordCancelStatuses(statuses map[string]string, identifiers []string, results []CancelActionStatusResponse) error {
	var errs error
	for i := range results {
		if results[i].Error == "" {
			statuses[identifiers[i]] = "success"
			continue
		}
		statuses[identifiers[i]] = results[i].Error
		errs = common.AppendError(errs, fmt.Errorf("%s: %w: %s", identifiers[i], errActionResponse, results[i].Error))
	}
	return errs
}

// convertOrder converts an order from its coin, fetching market mappings when the coin is not cached
func (e *Exchange) convertOrder(ctx context.Context, source *FrontendOpenOrderResponse, status string, statusTimestamp time.Time) (order.Detail, error) {
	mapping, a, err := e.getPairMappingByCoin(ctx, source.Coin)
	if err != nil {
		return order.Detail{}, err
	}
	return e.convertOrderFromMapping(source, status, statusTimestamp, &mapping, a)
}

func (e *Exchange) convertOrderFromMapping(source *FrontendOpenOrderResponse, status string, statusTimestamp time.Time, mapping *pairMapping, a asset.Item) (order.Detail, error) {
	orderType, err := parseOrderType(source.OrderType)
	if err != nil {
		return order.Detail{}, err
	}
	timeInForce, err := parseTimeInForce(source.TimeInForce)
	if err != nil {
		return order.Detail{}, err
	}
	detail, err := e.convertBasicOrder(&source.BasicOrderResponse, status, statusTimestamp, mapping.pair, a)
	if err != nil {
		return order.Detail{}, err
	}
	detail.Type = orderType
	detail.TimeInForce = timeInForce
	detail.ReduceOnly = source.ReduceOnly
	detail.TriggerPrice = source.TriggerPrice.Float64()
	return detail, nil
}

// convertWsOrder converts an order update, whose feed omits the order type and time in force
func (e *Exchange) convertWsOrder(update *WebsocketOrder) (order.Detail, error) {
	mapping, a, err := e.lookupPairMappingByCoin(update.Order.Coin)
	if err != nil {
		return order.Detail{}, err
	}
	return e.convertBasicOrder(&update.Order, update.Status, update.StatusTimestamp.Time(), mapping.pair, a)
}

// convertBasicOrder converts the order fields every order source provides; its size is the order's remaining size
func (e *Exchange) convertBasicOrder(source *BasicOrderResponse, status string, statusTimestamp time.Time, p currency.Pair, a asset.Item) (order.Detail, error) {
	side, err := parseSide(source.Side)
	if err != nil {
		return order.Detail{}, err
	}
	orderStatus, err := parseOrderStatus(status)
	if err != nil {
		return order.Detail{}, err
	}
	amount := cmp.Or(source.OriginalSize.Float64(), source.Size.Float64())
	executed := max(0, amount-source.Size.Float64())
	// Hyperliquid keeps an order's status through partial fills, so its executed size tells a partly filled order apart
	if executed > 0 {
		switch orderStatus {
		case order.Open:
			orderStatus = order.PartiallyFilled
		case order.Cancelled:
			orderStatus = order.PartiallyFilledCancelled
		case order.Filled:
			// An order that ends filled with part of its size unexecuted, such as an IOC order or a GTC order whose rest is
			// under the minimum order value, keeps its cancelled rest as its size
			if source.Size.Float64() > 0 {
				orderStatus = order.PartiallyFilledCancelled
			}
		}
	}
	placed := source.Timestamp.Time().UTC()
	lastUpdated := statusTimestamp.UTC()
	if lastUpdated.IsZero() {
		lastUpdated = placed
	}
	return order.Detail{
		Price:           source.LimitPrice.Float64(),
		Amount:          amount,
		ExecutedAmount:  executed,
		RemainingAmount: source.Size.Float64(),
		Exchange:        e.Name,
		OrderID:         strconv.FormatUint(source.OrderID, 10),
		ClientOrderID:   source.ClientOrderID,
		Side:            side,
		Status:          orderStatus,
		AssetType:       a,
		Date:            placed,
		LastUpdated:     lastUpdated,
		Pair:            p,
	}, nil
}

// convertL2Book converts an L2 book snapshot into an orderbook
func (e *Exchange) convertL2Book(book *L2BookResponse, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if len(book.Levels) != 2 {
		return nil, fmt.Errorf("%w: expected 2 sides, got %d", errInvalidBookLevelCount, len(book.Levels))
	}
	result := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		LastUpdated:       book.Time.Time().UTC(),
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              make(orderbook.Levels, len(book.Levels[0])),
		Asks:              make(orderbook.Levels, len(book.Levels[1])),
	}
	for i := range book.Levels[0] {
		result.Bids[i] = convertL2Level(&book.Levels[0][i])
	}
	for i := range book.Levels[1] {
		result.Asks[i] = convertL2Level(&book.Levels[1][i])
	}
	return result, nil
}

func convertL2Level(level *L2Level) orderbook.Level {
	return orderbook.Level{
		Price:      level.Price.Float64(),
		Amount:     level.Size.Float64(),
		OrderCount: int64(level.NumberOfOrders), //nolint:gosec // A level's order count is far below the int64 maximum
	}
}

// convertTrade converts a public trade; its side is the aggressor's
func (e *Exchange) convertTrade(t *RecentTradeResponse, p currency.Pair, a asset.Item) (trade.Data, error) {
	side, err := parseSide(t.Side)
	if err != nil {
		return trade.Data{}, err
	}
	return trade.Data{
		TID:          strconv.FormatUint(t.TradeID, 10),
		Exchange:     e.Name,
		CurrencyPair: p,
		AssetType:    a,
		Side:         side,
		Price:        t.Price.Float64(),
		Amount:       t.Size.Float64(),
		Timestamp:    t.Time.Time().UTC(),
	}, nil
}

func convertCandle(c *CandleResponse) kline.Candle {
	return kline.Candle{
		Time:   c.OpenTime.Time().UTC(),
		Open:   c.Open.Float64(),
		High:   c.High.Float64(),
		Low:    c.Low.Float64(),
		Close:  c.Close.Float64(),
		Volume: c.Volume.Float64(),
	}
}

// perpetualTickerPrice builds a ticker from a perpetual market context, whose oracle price is the index price
func (e *Exchange) perpetualTickerPrice(p currency.Pair, market *PerpetualAssetContext) *ticker.Price {
	return &ticker.Price{
		Last:         cmp.Or(market.MidPrice.Float64(), market.MarkPrice.Float64()),
		Open:         market.PreviousDayPrice.Float64(),
		BaseVolume:   market.DayBaseVolume.Float64(),
		QuoteVolume:  market.DayNotionalVolume.Float64(),
		OpenInterest: market.OpenInterest.Float64(),
		MarkPrice:    market.MarkPrice.Float64(),
		IndexPrice:   market.OraclePrice.Float64(),
		Pair:         p,
		ExchangeName: e.Name,
		AssetType:    asset.PerpetualContract,
		LastUpdated:  time.Now().UTC(),
	}
}

// spotTickerPrice builds a ticker from a spot market context
func (e *Exchange) spotTickerPrice(p currency.Pair, market *SpotAssetContext) *ticker.Price {
	return &ticker.Price{
		Last:         cmp.Or(market.MidPrice.Float64(), market.MarkPrice.Float64()),
		Open:         market.PreviousDayPrice.Float64(),
		BaseVolume:   market.DayBaseVolume.Float64(),
		QuoteVolume:  market.DayNotionalVolume.Float64(),
		MarkPrice:    market.MarkPrice.Float64(),
		Pair:         p,
		ExchangeName: e.Name,
		AssetType:    asset.Spot,
		LastUpdated:  time.Now().UTC(),
	}
}

// dexSubAccountID scopes a balance subaccount to a perpetual DEX, leaving the default DEX unscoped
func dexSubAccountID(address, dex string) string {
	if dex == "" {
		return address
	}
	return address + ":" + dex
}

// parseSide converts a side, which is A for ask and B for bid
func parseSide(side string) (order.Side, error) {
	switch side {
	case "A":
		return order.Sell, nil
	case "B":
		return order.Buy, nil
	default:
		return order.UnknownSide, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, side)
	}
}

// parseOrderStatus converts an order status, taking triggered as open: a triggered limit order rests on the book as
// triggered until it fills or is cancelled, while a triggered market order fills at once and gets a filled status then
func parseOrderStatus(status string) (order.Status, error) {
	switch status {
	case "open", "triggered":
		return order.Open, nil
	case "filled":
		return order.Filled, nil
	case "canceled", "scheduledCancel", "marginCanceled", "vaultWithdrawalCanceled", "openInterestCapCanceled", "selfTradeCanceled", "reduceOnlyCanceled", "siblingFilledCanceled", "delistedCanceled", "liquidatedCanceled":
		return order.Cancelled, nil
	case "rejected", "tickRejected", "minTradeNtlRejected", "perpMarginRejected", "reduceOnlyRejected", "badAloPxRejected", "iocCancelRejected", "badTriggerPxRejected", "marketOrderNoLiquidityRejected", "positionIncreaseAtOpenInterestCapRejected", "positionFlipAtOpenInterestCapRejected", "tooAggressiveAtOpenInterestCapRejected", "openInterestIncreaseRejected", "insufficientSpotBalanceRejected", "oracleRejected", "perpMaxPositionRejected":
		return order.Rejected, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %s", errUnsupportedOrderStatus, status)
	}
}

// parseOrderType converts the order types the frontend displays, such as Limit, Stop Market or Take Profit Limit
// A trigger order keeps its type once it has triggered, while Hyperliquid clears its trigger flag, so the type alone
// classifies it
func parseOrderType(orderType string) (order.Type, error) {
	lowerOrderType := strings.ToLower(orderType)
	switch {
	case strings.Contains(lowerOrderType, "take") && strings.Contains(lowerOrderType, "market"):
		return order.TakeProfitMarket, nil
	case strings.Contains(lowerOrderType, "take"):
		return order.TakeProfit, nil
	case strings.Contains(lowerOrderType, "stop") && strings.Contains(lowerOrderType, "market"):
		return order.StopMarket, nil
	case strings.Contains(lowerOrderType, "stop") && strings.Contains(lowerOrderType, "limit"):
		return order.StopLimit, nil
	case strings.Contains(lowerOrderType, "stop"):
		return order.Stop, nil
	case strings.Contains(lowerOrderType, "market"):
		return order.Market, nil
	case strings.Contains(lowerOrderType, "limit"):
		return order.Limit, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %s", order.ErrTypeIsInvalid, orderType)
	}
}

// parseTimeInForce converts a time in force; FrontendMarket marks the IOC orders the frontend places as market orders
func parseTimeInForce(timeInForce string) (order.TimeInForce, error) {
	switch strings.ToLower(timeInForce) {
	case "", "gtc":
		return order.GoodTillCancel, nil
	case "alo":
		return order.PostOnly, nil
	case "ioc", "frontendmarket", "liquidationmarket": // Market orders and liquidations execute as IOC orders
		return order.ImmediateOrCancel, nil
	default:
		return order.UnknownTIF, fmt.Errorf("%w: %s", order.ErrInvalidTimeInForce, timeInForce)
	}
}

func formatOrderTimeInForce(timeInForce order.TimeInForce) (string, error) {
	switch timeInForce {
	case order.UnknownTIF, order.GoodTillCancel:
		return TimeInForceGTC, nil
	case order.PostOnly, order.GoodTillCancel | order.PostOnly:
		return TimeInForceALO, nil
	case order.ImmediateOrCancel:
		return TimeInForceIOC, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, timeInForce)
	}
}

// validateOrderSize checks a size is positive, finite and fits the market's size decimals
func validateOrderSize(size float64, sizeDecimals uint64) error {
	if size <= 0 || math.IsNaN(size) || math.IsInf(size, 0) || sizeDecimals > 8 {
		return fmt.Errorf("%w: size %v with %d decimals", errSizePrecision, size, sizeDecimals)
	}
	scale := math.Pow10(int(sizeDecimals))
	if math.Abs(math.RoundToEven(size*scale)/scale-size) >= 1e-12 {
		return fmt.Errorf("%w: %v allows %d decimals", errSizePrecision, size, sizeDecimals)
	}
	return nil
}

// deriveFilledOrderState derives the status and remaining size of an order that filled on placement; only a GTC or IOC
// order can fill partially
func deriveFilledOrderState(requested, filled float64, sizeDecimals uint64, timeInForce string) (order.Status, float64, error) {
	if err := validateOrderSize(requested, sizeDecimals); err != nil {
		return order.UnknownStatus, 0, fmt.Errorf("%w: invalid requested size: %w", errInvalidFilledSize, err)
	}
	if err := validateOrderSize(filled, sizeDecimals); err != nil {
		return order.UnknownStatus, 0, fmt.Errorf("%w: invalid reported size: %w", errInvalidFilledSize, err)
	}
	scale := math.Pow10(int(sizeDecimals)) //nolint:gosec // validateOrderSize limits sizeDecimals to 8
	requestedUnits := math.Round(requested * scale)
	filledUnits := math.Round(filled * scale)
	switch {
	case filledUnits > requestedUnits:
		return order.UnknownStatus, 0, fmt.Errorf("%w: reported %v exceeds requested %v", errInvalidFilledSize, filled, requested)
	case filledUnits == requestedUnits:
		return order.Filled, 0, nil
	}
	remaining := (requestedUnits - filledUnits) / scale
	// An order answered filled is not resting, so the size it left unfilled was cancelled
	switch timeInForce {
	case TimeInForceGTC, TimeInForceIOC:
		return order.PartiallyFilledCancelled, remaining, nil
	default:
		return order.UnknownStatus, 0, fmt.Errorf("%w: partial fill for %q time in force", errActionStatusMalformed, timeInForce)
	}
}

// priceDecimalBase returns the maximum price decimals of an asset type before subtracting the size decimals
func priceDecimalBase(a asset.Item, sizeDecimals uint64) (uint64, error) {
	var decimalBase uint64
	switch a {
	case asset.Spot:
		decimalBase = spotPriceDecimalBase
	case asset.PerpetualContract:
		decimalBase = perpetualPriceDecimalBase
	default:
		return 0, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if sizeDecimals > decimalBase {
		return 0, fmt.Errorf("%w: %s prices allow at most %d size decimals, got %d", errSizePrecision, a, decimalBase, sizeDecimals)
	}
	return decimalBase, nil
}

// validateLimitPrice checks a price has at most five significant figures and fits the market's price decimals
func validateLimitPrice(price float64, a asset.Item, sizeDecimals uint64) error {
	if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return errInvalidMarketPrice
	}
	decimalBase, err := priceDecimalBase(a, sizeDecimals)
	if err != nil {
		return err
	}
	wire, err := floatToWire(price)
	if err != nil {
		return err
	}
	whole, fractional, hasFraction := strings.Cut(wire, ".")
	if !hasFraction {
		return nil // Integer prices are valid regardless of their significant figures
	}
	if uint64(len(fractional)) > decimalBase-sizeDecimals {
		return fmt.Errorf("%w: %s allows at most %d price decimals", errPricePrecision, a, decimalBase-sizeDecimals)
	}
	if significant := strings.TrimLeft(whole+fractional, "0"); len(significant) > marketPriceSignificantFigures {
		return fmt.Errorf("%w: got %d significant figures", errPricePrecision, len(significant))
	}
	return nil
}

// roundMarketPrice rounds a slippage-bounded price to a valid price within the bound, down for a buy and up for a sell. A
// valid price is an integer, or has at most five significant figures and the market's price decimals
func roundMarketPrice(price float64, side order.Side, a asset.Item, sizeDecimals uint64) (float64, error) {
	if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return 0, errInvalidMarketPrice
	}
	decimalBase, err := priceDecimalBase(a, sizeDecimals)
	if err != nil {
		return 0, err
	}
	// The exponent comes from the price's decimal form, which always has one for a finite price, as Log10 can be one out
	// next to a power of ten
	_, exponentText, _ := strings.Cut(strconv.FormatFloat(price, 'e', -1, 64), "e")
	exponent, _ := strconv.Atoi(exponentText)
	decimals := min(max(marketPriceSignificantFigures-1-exponent, 0), int(decimalBase-sizeDecimals)) //nolint:gosec // priceDecimalBase bounds the difference to 8
	scale := math.Pow10(decimals)
	// Rounding the product first keeps an architecture that fuses multiplies and adds to the same ticks
	ticks := float64(price * scale)
	// The bound carries the rounding error of computing it, a few units in the last place, which must not move it off a
	// tick it lands on
	if nearest := math.Round(ticks); math.Abs(ticks-nearest) <= 4*(math.Nextafter(nearest, math.Inf(1))-nearest) {
		ticks = nearest
	}
	if side.IsLong() {
		ticks = math.Floor(ticks)
	} else {
		ticks = math.Ceil(ticks)
	}
	if ticks <= 0 {
		return 0, errInvalidMarketPrice
	}
	return ticks / scale, nil
}

// slippagePrice moves a reference price against the order by the slippage tolerance
func slippagePrice(reference, slippage float64, side order.Side) float64 {
	if side.IsLong() {
		return reference * (1 + slippage)
	}
	return reference * (1 - slippage)
}
