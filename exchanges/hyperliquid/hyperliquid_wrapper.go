package hyperliquid

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
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

type pairMapping struct {
	pair         currency.Pair
	coin         string
	dex          string
	assetID      uint64
	sizeDecimals uint64
	maxLeverage  uint64
	onlyIsolated bool
}

var errAmbiguousCoinMapping = errors.New("ambiguous coin mapping")

func logDefaultError(err error) {
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// SetDefaults sets the basic defaults for Hyperliquid.
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
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: maximumCandleCount,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(GetRateLimits()))
	logDefaultError(err)
	e.API.Endpoints = e.NewEndpoints()
	logDefaultError(e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      hyperliquidAPIURL,
		exchange.RestFutures:   hyperliquidAPIURL,
		exchange.WebsocketSpot: hyperliquidWebsocketURL,
	}))
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
	e.pairMappingsMu.Lock()
	e.pairMappings = make(map[asset.Item][]pairMapping)
	e.pairMappingsMu.Unlock()
	e.pairMappingsFetchMu.Lock()
	e.pairMappingMisses = make(map[string]time.Time)
	e.pairMappingsFetchMu.Unlock()
	e.authorityValidationMu.Lock()
	e.authorityValidationKey = authorityValidationKey{}
	e.authorityValidated = false
	e.authorityValidationMu.Unlock()
	e.websocketPendingMu.Lock()
	e.websocketPending = make(map[websocketPendingKey]*websocketPendingOperation)
	e.websocketPendingMu.Unlock()
}

// Setup sets the exchange configuration profile.
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
	environmentEndpoints := []struct {
		kind       exchange.URL
		production string
		sandbox    string
	}{
		{kind: exchange.RestSpot, production: hyperliquidAPIURL, sandbox: hyperliquidTestnetAPIURL},
		{kind: exchange.RestFutures, production: hyperliquidAPIURL, sandbox: hyperliquidTestnetAPIURL},
		{kind: exchange.WebsocketSpot, production: hyperliquidWebsocketURL, sandbox: hyperliquidTestnetWebsocketURL},
	}
	if exch.UseSandbox {
		for _, endpoint := range environmentEndpoints {
			runningURL, err := e.API.Endpoints.GetURL(endpoint.kind)
			if err != nil {
				return err
			}
			if strings.TrimRight(runningURL, "/") != endpoint.production {
				continue
			}
			_ = e.API.Endpoints.SetRunningURL(endpoint.kind.String(), endpoint.sandbox) // Compile-time endpoint keys and URLs are valid.
		}
	} else {
		for _, endpoint := range environmentEndpoints {
			runningURL, err := e.API.Endpoints.GetURL(endpoint.kind)
			if err != nil {
				return err
			}
			if strings.TrimRight(runningURL, "/") == endpoint.sandbox {
				return fmt.Errorf("%w: %s uses the official testnet URL while useSandbox is false", errEndpointEnvironment, endpoint.kind)
			}
		}
	}
	websocketURL, _ := e.API.Endpoints.GetURL(exchange.WebsocketSpot) // The environment loop validated every configured endpoint.
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		DefaultURL:                             hyperliquidWebsocketURL,
		RunningURL:                             websocketURL,
		RunningURLAuth:                         websocketURL,
		Connector:                              e.WsConnect,
		Subscriber:                             e.Subscribe,
		Unsubscriber:                           e.Unsubscribe,
		GenerateSubscriptions:                  e.generateSubscriptions,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		TradeFeed:                              exch.Features.Enabled.TradeFeed,
		FillsFeed:                              exch.Features.Enabled.FillsFeed,
		MaxWebsocketSubscriptionsPerConnection: maximumWebsocketSubscriptions,
	}); err != nil {
		return err
	}
	connectionRateLimit := request.NewWeightedRateLimitByDuration(websocketMessageInterval)
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		RateLimit:            connectionRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}); err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		RateLimit:            connectionRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  websocketURL,
		Authenticated:        true,
	})
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
	return e.fetchPairMapping(ctx, p, a)
}

func (e *Exchange) lookupPairMapping(p currency.Pair, a asset.Item) (pairMapping, bool) {
	e.pairMappingsMu.RLock()
	defer e.pairMappingsMu.RUnlock()
	for _, mapping := range e.pairMappings[a] {
		if mapping.pair.Equal(p) {
			return mapping, true
		}
	}
	return pairMapping{}, false
}

func (e *Exchange) fetchPairMapping(ctx context.Context, p currency.Pair, a asset.Item) (pairMapping, error) {
	e.pairMappingsFetchMu.Lock()
	defer e.pairMappingsFetchMu.Unlock()
	if mapping, ok := e.lookupPairMapping(p, a); ok {
		return mapping, nil
	}
	cacheKey := "pair:" + a.String() + ":" + strings.ToLower(p.String())
	if expiry, ok := e.pairMappingMisses[cacheKey]; ok {
		if time.Now().Before(expiry) {
			return pairMapping{}, fmt.Errorf("%w: %s %s", errPairMappingNotFound, a, p)
		}
		delete(e.pairMappingMisses, cacheKey)
	}
	if _, err := e.FetchTradablePairs(ctx, a); err != nil {
		return pairMapping{}, err
	}
	if mapping, ok := e.lookupPairMapping(p, a); ok {
		return mapping, nil
	}
	if e.pairMappingMisses == nil {
		e.pairMappingMisses = make(map[string]time.Time)
	}
	e.pairMappingMisses[cacheKey] = time.Now().Add(pairMappingMissCacheDuration)
	return pairMapping{}, fmt.Errorf("%w: %s %s", errPairMappingNotFound, a, p)
}

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

func (e *Exchange) lookupPairMappingByCoin(coin string) (pairMapping, asset.Item, error) {
	e.pairMappingsMu.RLock()
	defer e.pairMappingsMu.RUnlock()
	var found pairMapping
	var foundAsset asset.Item
	count := 0
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		for _, mapping := range e.pairMappings[a] {
			if mapping.coin == coin {
				found = mapping
				foundAsset = a
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

func (e *Exchange) fetchPairMappingByCoin(ctx context.Context, coin string) (pairMapping, asset.Item, error) {
	e.pairMappingsFetchMu.Lock()
	defer e.pairMappingsFetchMu.Unlock()
	mapping, a, err := e.lookupPairMappingByCoin(coin)
	if err == nil || errors.Is(err, errAmbiguousCoinMapping) {
		return mapping, a, err
	}
	cacheKey := "coin:" + strings.ToLower(coin)
	if expiry, ok := e.pairMappingMisses[cacheKey]; ok {
		if time.Now().Before(expiry) {
			return pairMapping{}, asset.Empty, fmt.Errorf("%w: %s", errPairMappingNotFound, coin)
		}
		delete(e.pairMappingMisses, cacheKey)
	}
	for _, item := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		if !e.SupportsAsset(item) {
			continue
		}
		if _, err := e.FetchTradablePairs(ctx, item); err != nil {
			return pairMapping{}, asset.Empty, err
		}
	}
	mapping, a, err = e.lookupPairMappingByCoin(coin)
	if errors.Is(err, errPairMappingNotFound) {
		if e.pairMappingMisses == nil {
			e.pairMappingMisses = make(map[string]time.Time)
		}
		e.pairMappingMisses[cacheKey] = time.Now().Add(pairMappingMissCacheDuration)
	}
	return mapping, a, err
}

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

// FetchTradablePairs returns active spot markets or active perpetual markets
// across the default and every registered builder-deployed DEX.
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch a {
	case asset.Spot, asset.PerpetualContract:
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	var candidates []pairMapping
	switch a {
	case asset.PerpetualContract:
		dexes, err := e.getPerpetualDEXNames(ctx)
		if err != nil {
			return nil, err
		}
		collateralNames := map[uint64]string{0: perpetualQuoteCurrency}
		spotTokensLoaded := false
		for dexIndex, dex := range dexes {
			metadata, err := e.GetPerpetualMetadataForDEX(ctx, dex)
			if err != nil {
				return nil, err
			}
			collateralName, ok := collateralNames[metadata.CollateralToken]
			if !ok {
				if !spotTokensLoaded {
					spotMetadata, err := e.GetSpotMetadata(ctx)
					if err != nil {
						return nil, err
					}
					seenSpotTokenIndices := make(map[uint64]struct{}, len(spotMetadata.Tokens))
					for i := range spotMetadata.Tokens {
						index := spotMetadata.Tokens[i].Index
						if _, exists := seenSpotTokenIndices[index]; exists {
							return nil, fmt.Errorf("%w: duplicate spot token index %d", errUnexpectedResponseLength, index)
						}
						seenSpotTokenIndices[index] = struct{}{}
						name := strings.TrimSpace(spotMetadata.Tokens[i].Name)
						if name == "" {
							return nil, fmt.Errorf("%w: collateral token index %d has no name", errSpotTokenNotFound, index)
						}
						if index == 0 && !strings.EqualFold(name, perpetualQuoteCurrency) {
							return nil, fmt.Errorf("%w: collateral token index 0 is %q", errSpotTokenNotFound, name)
						}
						collateralNames[index] = name
					}
					spotTokensLoaded = true
					collateralName, ok = collateralNames[metadata.CollateralToken]
				}
				if !ok {
					return nil, fmt.Errorf("%w: collateral token index %d", errSpotTokenNotFound, metadata.CollateralToken)
				}
			}
			if dexIndex != 0 && len(metadata.Universe) > builderPerpetualDEXAssetStride {
				return nil, fmt.Errorf("%w: DEX %q has %d markets, maximum %d", errInvalidPerpetualDEX, dex, len(metadata.Universe), builderPerpetualDEXAssetStride)
			}
			offset := uint64(0)
			if dexIndex != 0 {
				offset = builderPerpetualAssetIDBase + uint64(dexIndex)*builderPerpetualDEXAssetStride
			}
			for marketIndex, market := range metadata.Universe {
				if market.IsDelisted {
					continue
				}
				if dex != "" && !strings.HasPrefix(market.Name, dex+":") {
					return nil, fmt.Errorf("%w: market %q is not scoped to DEX %q", errInvalidPerpetualDEX, market.Name, dex)
				}
				pair, err := currency.NewPairFromStrings(market.Name, collateralName)
				if err != nil {
					log.Warnf(log.ExchangeSys, "%s skipping perpetual market %q: %s", e.Name, market.Name, err)
					continue
				}
				candidates = append(candidates, pairMapping{
					pair:         pair,
					coin:         market.Name,
					dex:          dex,
					assetID:      offset + uint64(marketIndex), //nolint:gosec // A validated universe slice index fits Hyperliquid's asset identifier.
					sizeDecimals: market.SizeDecimals,
					maxLeverage:  market.MaxLeverage,
					onlyIsolated: market.OnlyIsolated,
				})
			}
		}
	default: // Spot is the only remaining validated asset type.
		metadata, err := e.GetSpotMetadata(ctx)
		if err != nil {
			return nil, err
		}
		tokens := make(map[uint64]SpotTokenMetadata, len(metadata.Tokens))
		for _, token := range metadata.Tokens {
			if _, exists := tokens[token.Index]; exists {
				return nil, fmt.Errorf("%w: duplicate spot token index %d", errUnexpectedResponseLength, token.Index)
			}
			tokens[token.Index] = token
		}
		candidates = make([]pairMapping, 0, len(metadata.Universe))
		marketIndices := make(map[uint64]struct{}, len(metadata.Universe))
		marketNames := make(map[string]struct{}, len(metadata.Universe))
		for _, market := range metadata.Universe {
			if _, exists := marketIndices[market.Index]; exists {
				return nil, fmt.Errorf("%w: duplicate spot market index %d", errUnexpectedResponseLength, market.Index)
			}
			marketIndices[market.Index] = struct{}{}
			if _, exists := marketNames[market.Name]; exists {
				return nil, fmt.Errorf("%w: duplicate spot market name %q", errUnexpectedResponseLength, market.Name)
			}
			marketNames[market.Name] = struct{}{}
			if len(market.Tokens) != 2 {
				log.Warnf(log.ExchangeSys, "%s skipping spot market %q: %s; got %d tokens", e.Name, market.Name, errInvalidSpotTokenCount, len(market.Tokens))
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
			if strings.TrimSpace(baseToken.Name) == "" || strings.TrimSpace(quoteToken.Name) == "" {
				log.Warnf(log.ExchangeSys, "%s skipping spot market %q with an empty token name", e.Name, market.Name)
				continue
			}
			pair, err := currency.NewPairFromStrings(baseToken.Name, quoteToken.Name)
			if err != nil {
				log.Warnf(log.ExchangeSys, "%s skipping spot market %q: %s", e.Name, market.Name, err)
				continue
			}
			candidates = append(candidates, pairMapping{
				pair:         pair,
				coin:         market.Name,
				assetID:      10000 + market.Index,
				sizeDecimals: baseToken.SizeDecimals,
			})
		}
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

// UpdateTradablePairs updates all available Hyperliquid markets.
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

// UpdateTickers updates all enabled tickers for an asset type.
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot, asset.PerpetualContract:
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	var errs error
	switch a {
	case asset.PerpetualContract:
		contextsByDEX := make(map[string]map[string]PerpetualAssetContext)
		errorsByDEX := make(map[string]error)
		for _, pair := range pairs {
			mapping, err := e.getPairMapping(ctx, pair, a)
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%s: %w", pair, err))
				continue
			}
			contexts, loaded := contextsByDEX[mapping.dex]
			if !loaded && errorsByDEX[mapping.dex] == nil {
				resp, err := e.GetPerpetualMetadataAndAssetContextsForDEX(ctx, mapping.dex)
				switch {
				case err != nil:
					errorsByDEX[mapping.dex] = err
				case len(resp.Metadata.Universe) != len(resp.AssetContexts):
					errorsByDEX[mapping.dex] = fmt.Errorf("%w: DEX %q returned %d perpetual markets and %d contexts", errUnexpectedResponseLength, mapping.dex, len(resp.Metadata.Universe), len(resp.AssetContexts))
				default:
					contexts = make(map[string]PerpetualAssetContext, len(resp.AssetContexts))
					for i := range resp.Metadata.Universe {
						contexts[resp.Metadata.Universe[i].Name] = resp.AssetContexts[i]
					}
					contextsByDEX[mapping.dex] = contexts
				}
			}
			if errorsByDEX[mapping.dex] != nil {
				errs = common.AppendError(errs, fmt.Errorf("%s: %w", pair, errorsByDEX[mapping.dex]))
				continue
			}
			market, ok := contexts[mapping.coin]
			if !ok {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", errAssetContextNotFound, mapping.coin))
				continue
			}
			// The batch context has no last execution price, so use its current midpoint and fall back to the mark price.
			last := market.MidPrice.Float64()
			if last == 0 {
				last = market.MarkPrice.Float64()
			}
			price := &ticker.Price{
				Last:         last,
				Volume:       market.DayBaseVolume.Float64(),
				QuoteVolume:  market.DayNotionalVolume.Float64(),
				Open:         market.PreviousDayPrice.Float64(),
				OpenInterest: market.OpenInterest.Float64(),
				MarkPrice:    market.MarkPrice.Float64(),
				IndexPrice:   market.OraclePrice.Float64(),
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    a,
				LastUpdated:  time.Now().UTC(),
			}
			if err := ticker.ProcessTicker(price); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%s: %w", pair, err))
			}
		}
	default: // Spot is the only remaining validated asset type.
		resp, err := e.GetSpotMetadataAndAssetContexts(ctx)
		if err != nil {
			return err
		}
		contexts := make(map[string]SpotAssetContext, len(resp.AssetContexts))
		positionallyAligned := len(resp.Metadata.Universe) == len(resp.AssetContexts) &&
			!slices.ContainsFunc(resp.AssetContexts, func(market SpotAssetContext) bool { return market.Coin != "" })
		for i, market := range resp.AssetContexts {
			coin := market.Coin
			if coin == "" {
				if !positionallyAligned {
					return fmt.Errorf("%w: spot context %d has no market identifier", errAssetContextNotFound, i)
				}
				coin = resp.Metadata.Universe[i].Name
			}
			contexts[coin] = market
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
			// The batch context has no last execution price, so use its current midpoint and fall back to the mark price.
			last := market.MidPrice.Float64()
			if last == 0 {
				last = market.MarkPrice.Float64()
			}
			price := &ticker.Price{
				Last:         last,
				Volume:       market.DayBaseVolume.Float64(),
				QuoteVolume:  market.DayNotionalVolume.Float64(),
				Open:         market.PreviousDayPrice.Float64(),
				MarkPrice:    market.MarkPrice.Float64(),
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    a,
				LastUpdated:  time.Now().UTC(),
			}
			if err := ticker.ProcessTicker(price); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%s: %w", pair, err))
			}
		}
	}
	return errs
}

// UpdateTicker updates and returns one ticker.
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := e.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns one L2 orderbook snapshot.
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	coin, err := e.getCoin(ctx, p, a)
	if err != nil {
		return nil, err
	}
	resp, err := e.GetL2Book(ctx, a, &L2BookRequest{Coin: coin})
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		LastUpdated:       resp.Time.Time().UTC(),
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              make(orderbook.Levels, len(resp.Levels[0])),
		Asks:              make(orderbook.Levels, len(resp.Levels[1])),
	}
	for i := range resp.Levels[0] {
		book.Bids[i] = orderbook.Level{Price: resp.Levels[0][i].Price.Float64(), Amount: resp.Levels[0][i].Size.Float64()}
	}
	for i := range resp.Levels[1] {
		book.Asks[i] = orderbook.Level{Price: resp.Levels[1][i].Price.Float64(), Amount: resp.Levels[1][i].Size.Float64()}
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, a)
}

// GetRecentTrades returns recent public trades for a pair and asset type.
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	coin, err := e.getCoin(ctx, p, a)
	if err != nil {
		return nil, err
	}
	resp, err := e.GetRecentTradesForCoin(ctx, coin, a)
	if err != nil {
		return nil, err
	}
	trades := make([]trade.Data, len(resp))
	for i := range resp {
		var side order.Side
		switch resp[i].Side {
		case "A":
			side = order.Sell
		case "B":
			side = order.Buy
		default:
			return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, resp[i].Side)
		}
		trades[i] = trade.Data{
			TID:          strconv.FormatUint(resp[i].TradeID, 10),
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    a,
			Side:         side,
			Price:        resp[i].Price.Float64(),
			Amount:       resp[i].Size.Float64(),
			Timestamp:    resp[i].Time.Time().UTC(),
		}
	}
	sort.Sort(trade.ByDate(trades))
	return trades, e.AddTradesToBuffer(trades...)
}

// GetHistoricCandles returns candles within the requested time range.
func (e *Exchange) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(p, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}
	coin, err := e.getCoin(ctx, p, a)
	if err != nil {
		return nil, err
	}
	resp, err := e.GetCandles(ctx, a, &CandleRequest{Coin: coin, Interval: req.ExchangeInterval, StartTime: req.Start, EndTime: req.End})
	if err != nil {
		return nil, err
	}
	candles := make([]kline.Candle, 0, len(resp))
	for i := range resp {
		candles = append(candles, kline.Candle{
			Time:   resp[i].OpenTime.Time().UTC(),
			Open:   resp[i].Open.Float64(),
			High:   resp[i].High.Float64(),
			Low:    resp[i].Low.Float64(),
			Close:  resp[i].Close.Float64(),
			Volume: resp[i].Volume.Float64(),
		})
	}
	return req.ProcessResponse(candles)
}

// UpdateAccountBalances retrieves balances for the configured account, vault, or subaccount.
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	address, err := e.getWatchAddress(ctx)
	if err != nil {
		return nil, err
	}
	setSpotBalances := func(subAccount *accounts.SubAccount, state *SpotClearinghouseState) {
		for i := range state.Balances {
			code := currency.NewCode(state.Balances[i].Coin)
			subAccount.Balances.Set(code, accounts.Balance{
				Total: state.Balances[i].Total.Float64(),
				Hold:  state.Balances[i].Hold.Float64(),
				Free:  state.Balances[i].Total.Float64() - state.Balances[i].Hold.Float64(),
			})
		}
	}
	var subAccounts accounts.SubAccounts
	switch a {
	case asset.Spot:
		state, err := e.GetSpotClearinghouseState(ctx, address)
		if err != nil {
			return nil, err
		}
		subAccount := accounts.NewSubAccount(a, address)
		setSpotBalances(subAccount, state)
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
		if mode == AccountAbstractionUnified || mode == AccountAbstractionPortfolio {
			// Hyperliquid reports the single shared unified/portfolio balance
			// through spotClearinghouseState. Keep it canonical under Spot and
			// save empty snapshots for every registered perpetual DEX to clear
			// stale separate balances without double counting aggregate totals.
			subAccounts = make(accounts.SubAccounts, 0, len(dexes))
			for i := range dexes {
				subAccountID := address
				if dexes[i] != "" {
					subAccountID += ":" + dexes[i]
				}
				subAccounts = append(subAccounts, accounts.NewSubAccount(a, subAccountID))
			}
			break
		}

		spotMetadata, err := e.GetSpotMetadata(ctx)
		if err != nil {
			return nil, err
		}
		tokens := make(map[uint64]SpotTokenMetadata, len(spotMetadata.Tokens))
		for i := range spotMetadata.Tokens {
			if _, exists := tokens[spotMetadata.Tokens[i].Index]; exists {
				return nil, fmt.Errorf("%w: duplicate spot token index %d", errUnexpectedResponseLength, spotMetadata.Tokens[i].Index)
			}
			tokens[spotMetadata.Tokens[i].Index] = spotMetadata.Tokens[i]
		}
		subAccounts = make(accounts.SubAccounts, 0, len(dexes))
		for i := range dexes {
			metadata, err := e.GetPerpetualMetadataForDEX(ctx, dexes[i])
			if err != nil {
				return nil, err
			}
			token, ok := tokens[metadata.CollateralToken]
			if !ok || strings.TrimSpace(token.Name) == "" {
				return nil, fmt.Errorf("%w: collateral token index %d", errSpotTokenNotFound, metadata.CollateralToken)
			}
			state, err := e.GetClearinghouseStateForDEX(ctx, address, dexes[i])
			if err != nil {
				return nil, err
			}
			subAccountID := address
			if dexes[i] != "" {
				subAccountID += ":" + dexes[i]
			}
			subAccount := accounts.NewSubAccount(a, subAccountID)
			subAccount.Balances.Set(currency.NewCode(token.Name), accounts.Balance{
				Total: state.MarginSummary.AccountValue.Float64(),
				Hold:  state.MarginSummary.TotalMarginUsed.Float64(),
				Free:  state.Withdrawable.Float64(),
			})
			subAccounts = append(subAccounts, subAccount)
		}
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	return subAccounts, e.Accounts.Save(ctx, subAccounts, true)
}

func (e *Exchange) getOpenOrdersForAsset(ctx context.Context, user string, a asset.Item) ([]OpenOrder, error) {
	switch a {
	case asset.Spot:
		return e.GetOpenOrdersForUserForDEX(ctx, user, "")
	case asset.PerpetualContract:
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	dexes, err := e.getPerpetualDEXNames(ctx)
	if err != nil {
		return nil, err
	}
	var result []OpenOrder
	for i := range dexes {
		orders, err := e.GetOpenOrdersForUserForDEX(ctx, user, dexes[i])
		if err != nil {
			return nil, err
		}
		result = append(result, orders...)
	}
	return result, nil
}

func (e *Exchange) getUserNonFundingLedgerUpdatesPaginated(
	ctx context.Context,
	user string,
	start,
	end time.Time,
) ([]UserLedgerUpdate, error) {
	cursor := start
	var result []UserLedgerUpdate
	seen := make(map[UserLedgerUpdate]struct{})
	for !cursor.After(end) {
		page, err := e.GetUserNonFundingLedgerUpdates(ctx, user, cursor, end)
		if err != nil {
			return nil, err
		}
		if len(page) > maximumUserLedgerHistoryCount {
			return nil, fmt.Errorf(
				"%w: ledger page has %d entries, maximum %d",
				errUnexpectedResponseLength,
				len(page),
				maximumUserLedgerHistoryCount)
		}
		for i := range page {
			recordTime := page[i].Time.Time().UTC()
			if recordTime.Before(cursor) ||
				recordTime.After(end) ||
				(i > 0 && recordTime.Before(page[i-1].Time.Time())) {
				return nil, fmt.Errorf("%w: malformed user ledger page", errUnexpectedResponseLength)
			}
			if _, exists := seen[page[i]]; !exists {
				seen[page[i]] = struct{}{}
				result = append(result, page[i])
			}
		}
		if len(page) < maximumUserLedgerHistoryCount {
			break
		}
		last := page[len(page)-1].Time.Time().UTC()
		if !last.After(cursor) {
			return nil, fmt.Errorf("%w: user ledger cursor did not advance", errUnexpectedResponseLength)
		}
		cursor = last
	}
	return result, nil
}

func (e *Exchange) convertUserLedgerUpdate(update *UserLedgerUpdate) (exchange.FundingHistory, error) {
	if update == nil {
		return exchange.FundingHistory{}, common.ErrNilPointer
	}
	deltaType := strings.TrimSpace(update.Delta.Type)
	if deltaType == "" {
		return exchange.FundingHistory{}, fmt.Errorf("%w: ledger update type is empty", errUnexpectedResponseLength)
	}
	ccy := perpetualQuoteCurrency
	amount := update.Delta.USDC.Float64()
	switch deltaType {
	case "spotTransfer", "spotGenesis", "send":
		if update.Delta.Token != "" {
			ccy, _, _ = strings.Cut(update.Delta.Token, ":")
		}
		amount = update.Delta.Amount.Float64()
	case "rewardsClaim":
		amount = update.Delta.Amount.Float64()
	case "vaultWithdraw":
		amount = update.Delta.NetWithdrawnUSD.Float64()
	}
	description := deltaType
	if deltaType == "accountClassTransfer" {
		description = "perpetual to spot"
		if update.Delta.ToPerp {
			description = "spot to perpetual"
		}
	} else if update.Delta.SourceDEX != "" || update.Delta.DestinationDEX != "" {
		description = fmt.Sprintf("%s: %s to %s", deltaType, update.Delta.SourceDEX, update.Delta.DestinationDEX)
	}
	return exchange.FundingHistory{
		ExchangeName:      e.Name,
		Status:            "processed",
		TransferID:        update.Hash,
		Description:       description,
		Timestamp:         update.Time.Time().UTC(),
		Currency:          ccy,
		Amount:            amount,
		Fee:               update.Delta.Fee.Float64(),
		TransferType:      deltaType,
		CryptoToAddress:   update.Delta.Destination,
		CryptoFromAddress: update.Delta.User,
		CryptoTxID:        update.Hash,
	}, nil
}

// GetAccountFundingHistory returns the configured account's deposits,
// withdrawals, transfers, rewards, and other non-funding ledger movements.
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	user, err := e.getWatchAddress(ctx)
	if err != nil {
		return nil, err
	}
	end := time.Now().UTC()
	updates, err := e.getUserNonFundingLedgerUpdatesPaginated(ctx, user, time.UnixMilli(1).UTC(), end)
	if err != nil {
		return nil, err
	}
	result := make([]exchange.FundingHistory, len(updates))
	for i := range updates {
		result[i], err = e.convertUserLedgerUpdate(&updates[i])
		if err != nil {
			return nil, err
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	return result, nil
}

// GetWithdrawalsHistory returns processed Hyperliquid bridge withdrawals.
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	if !c.IsEmpty() && !c.Equal(currency.USDC) {
		return []exchange.WithdrawalHistory{}, nil
	}
	user, err := e.getWatchAddress(ctx)
	if err != nil {
		return nil, err
	}
	end := time.Now().UTC()
	updates, err := e.getUserNonFundingLedgerUpdatesPaginated(ctx, user, time.UnixMilli(1).UTC(), end)
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
			Currency:     perpetualQuoteCurrency,
			Amount:       math.Abs(updates[i].Delta.USDC.Float64()),
			Fee:          math.Abs(updates[i].Delta.Fee.Float64()),
			TransferType: "withdrawal",
			CryptoTxID:   updates[i].Hash,
			CryptoChain:  e.getBridgeChain(),
		})
	}
	return result, nil
}

// GetHistoricTrades is not supported by Hyperliquid's recent-trades endpoint.
func (e *Exchange) GetHistoricTrades(context.Context, currency.Pair, asset.Item, time.Time, time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime is not supported by Hyperliquid.
func (e *Exchange) GetServerTime(context.Context, asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// SubmitOrder submits one signed limit or slippage-bounded market order.
func (e *Exchange) SubmitOrder(ctx context.Context, submit *order.Submit) (*order.SubmitResponse, error) {
	return e.submitOrder(ctx, submit)
}

// ModifyOrder replaces one order while preserving fields omitted by the modification request.
func (e *Exchange) ModifyOrder(ctx context.Context, modify *order.Modify) (*order.ModifyResponse, error) {
	if err := modify.Validate(); err != nil {
		return nil, err
	}
	if _, _, err := e.getSigningCredentials(ctx); err != nil {
		return nil, err
	}
	identifier := modify.OrderID
	var wireOrderID any
	var err error
	if modify.OrderID != "" {
		wireOrderID, err = strconv.ParseUint(modify.OrderID, 10, 64)
		if err != nil || wireOrderID == uint64(0) {
			return nil, fmt.Errorf("%w: %q", order.ErrOrderIDNotSet, modify.OrderID)
		}
	} else {
		if err := validateClientOrderID(modify.ClientOrderID); err != nil {
			return nil, err
		}
		identifier = strings.ToLower(modify.ClientOrderID)
		wireOrderID = identifier
	}
	existing, err := e.GetOrderInfo(ctx, identifier, modify.Pair, modify.AssetType)
	if err != nil {
		return nil, err
	}
	if existing.IsInactive() || existing.RemainingAmount <= 0 {
		return nil, fmt.Errorf("%w: %s", errOrderNotModifiable, identifier)
	}
	side := modify.Side
	if side == order.UnknownSide {
		side = existing.Side
	}
	orderType := modify.Type
	if orderType == order.UnknownType {
		orderType = existing.Type
	}
	timeInForce := modify.TimeInForce
	if timeInForce == order.UnknownTIF {
		timeInForce = existing.TimeInForce
	}
	amount := modify.Amount
	if amount == 0 {
		amount = existing.RemainingAmount
	}
	price := modify.Price
	if price == 0 {
		price = existing.Price
	}
	triggerPrice := modify.TriggerPrice
	switch orderType {
	case order.Stop, order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitMarket:
		if modify.TriggerPriceType != order.MarkPrice {
			return nil, fmt.Errorf("%w: Hyperliquid triggers use mark price", errRiskManagementUnsupported)
		}
		if triggerPrice == 0 {
			triggerPrice = existing.TriggerPrice
		}
	default:
		triggerPrice = modify.TriggerPrice
	}
	clientOrderID := existing.ClientOrderID
	if modify.NewClientOrderID != "" {
		clientOrderID = modify.NewClientOrderID
	}
	wire, mapping, err := e.buildOrderWire(ctx,
		modify.Pair,
		modify.AssetType,
		orderType,
		side,
		timeInForce,
		amount,
		price,
		triggerPrice,
		modify.SlippageTolerance,
		existing.ReduceOnly,
		clientOrderID)
	if err != nil {
		return nil, err
	}
	action := batchModifyAction{Type: "batchModify", Modifies: []modifyWire{{OrderID: wireOrderID, Order: wire}}}
	var response exchangeActionResponse
	if err := e.sendSignedAction(ctx, action, 1, &response); err != nil {
		return nil, err
	}
	statuses, err := parseOrderActionStatuses(&response, 1)
	if err != nil {
		return nil, err
	}
	if statuses[0].Error != "" {
		return nil, fmt.Errorf("%w: %s", order.ErrUnableToPlaceOrder, statuses[0].Error)
	}
	wireTimeInForce := ""
	if wire.Type.Limit != nil {
		wireTimeInForce = wire.Type.Limit.TimeInForce
	}
	status := order.New
	remainingAmount := amount
	orderID, _ := strconv.ParseUint(existing.OrderID, 10, 64)
	if statuses[0].Resting != nil {
		orderID = statuses[0].Resting.OrderID
	}
	if statuses[0].Filled != nil {
		status, remainingAmount, err = deriveFilledOrderState(amount, statuses[0].Filled.TotalSize.Float64(), mapping.sizeDecimals, wireTimeInForce)
		if err != nil {
			return nil, err
		}
		orderID = statuses[0].Filled.OrderID
	}
	responsePrice, _ := strconv.ParseFloat(wire.Price, 64) // buildOrderWire guarantees a valid wire-format number.
	responseTimeInForce := order.UnknownTIF
	if wireTimeInForce != "" {
		responseTimeInForce, _ = classifyHyperliquidTimeInForce(wireTimeInForce)
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
		TimeInForce:     responseTimeInForce,
		Price:           responsePrice,
		Amount:          amount,
		RemainingAmount: remainingAmount,
		TriggerPrice:    triggerPrice,
		Date:            existing.Date,
		LastUpdated:     time.Now().UTC(),
	}, nil
}

// CancelOrder cancels one order by numeric order ID or client order ID.
func (e *Exchange) CancelOrder(ctx context.Context, cancel *order.Cancel) error {
	if cancel == nil {
		return order.ErrCancelOrderIsNil
	}
	_, err := e.cancelOrders(ctx, []order.Cancel{*cancel})
	return err
}

// CancelBatchOrders cancels a batch of numeric and client-ID orders.
func (e *Exchange) CancelBatchOrders(ctx context.Context, cancels []order.Cancel) (*order.CancelBatchResponse, error) {
	statuses, err := e.cancelOrders(ctx, cancels)
	return &order.CancelBatchResponse{Status: statuses}, err
}

// CancelAllOrders cancels every matching open order for one asset and optional pair.
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
	openOrders, err := e.getOpenOrdersForAsset(ctx, address, cancel.AssetType)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	cancels := make([]order.Cancel, 0, len(openOrders))
	var errs error
	for i := range openOrders {
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
	for start := 0; start < len(cancels); start += maximumActionBatchSize {
		end := min(start+maximumActionBatchSize, len(cancels))
		batchStatuses, err := e.cancelOrders(ctx, cancels[start:end])
		maps.Copy(statuses, batchStatuses)
		errs = common.AppendError(errs, err)
	}
	return order.CancelAllResponse{Status: statuses}, errs
}

// GetOrderInfo returns one order by numeric order ID or client order ID.
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if a != asset.Empty && !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	var identifier any
	numericID, err := strconv.ParseUint(orderID, 10, 64)
	if err == nil {
		identifier = numericID
	} else {
		if err := validateClientOrderID(orderID); err != nil {
			return nil, err
		}
		identifier = strings.ToLower(orderID)
	}
	address, err := e.getWatchAddress(ctx)
	if err != nil {
		return nil, err
	}
	response, err := e.GetOrderStatusForUser(ctx, address, identifier)
	if err != nil {
		return nil, err
	}
	if response.Status != "order" || response.Order == nil {
		return nil, order.ErrOrderNotFound
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

// GetDepositAddress is not supported by the current integration.
func (e *Exchange) GetDepositAddress(context.Context, currency.Code, string, string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetAvailableTransferChains returns the environment-specific Arbitrum bridge
// network used for USDC withdrawals.
func (e *Exchange) GetAvailableTransferChains(_ context.Context, c currency.Code) ([]string, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if !c.Equal(currency.USDC) {
		return nil, fmt.Errorf("%w: %s", errTransferCurrencyInvalid, c)
	}
	return []string{e.getBridgeChain()}, nil
}

// WithdrawCryptocurrencyFunds submits a USDC bridge withdrawal, or a Core
// USDC send when InternalTransfer is set.
func (e *Exchange) WithdrawCryptocurrencyFunds(
	ctx context.Context,
	withdrawRequest *withdraw.Request,
) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
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
	chain := strings.TrimSpace(withdrawRequest.Crypto.Chain)
	if chain != "" && !strings.EqualFold(chain, e.getBridgeChain()) {
		return nil, fmt.Errorf("%w: expected %s", errBridgeChainInvalid, e.getBridgeChain())
	}
	var (
		nonce uint64
		err   error
	)
	if withdrawRequest.InternalTransfer {
		nonce, err = e.SendCoreUSDC(ctx, withdrawRequest.Crypto.Address, withdrawRequest.Amount)
	} else {
		nonce, err = e.WithdrawFromBridge(ctx, withdrawRequest.Crypto.Address, withdrawRequest.Amount)
	}
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   e.Name,
		ID:     strconv.FormatUint(nonce, 10),
		Status: "submitted",
	}, nil
}

// WithdrawFiatFunds is not supported by Hyperliquid.
func (e *Exchange) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank is not supported by Hyperliquid.
func (e *Exchange) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders returns and filters open orders for the configured account.
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
	openOrders, err := e.getOpenOrdersForAsset(ctx, address, filter.AssetType)
	if err != nil {
		return nil, err
	}
	converted := make([]order.Detail, 0, len(openOrders))
	var errs error
	for i := range openOrders {
		detail, err := e.convertOrder(ctx, &openOrders[i], "open", time.Time{})
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("order %d: %w", openOrders[i].OrderID, err))
			continue
		}
		if detail.AssetType == filter.AssetType {
			converted = append(converted, detail)
		}
	}
	return filter.Filter(e.Name, converted), errs
}

// GetOrderHistory returns and filters the latest order history for the configured account.
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
	history, err := e.GetHistoricalOrdersForUser(ctx, address)
	if err != nil {
		return nil, err
	}
	converted := make([]order.Detail, 0, len(history))
	var errs error
	for i := range history {
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

// GetFeeByType returns an account-specific maker or taker trade fee estimate.
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
	if feeBuilder.PurchasePrice < 0 ||
		feeBuilder.Amount < 0 ||
		math.IsNaN(feeBuilder.PurchasePrice) ||
		math.IsNaN(feeBuilder.Amount) ||
		math.IsInf(feeBuilder.PurchasePrice, 0) ||
		math.IsInf(feeBuilder.Amount, 0) {
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
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) &&
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	var rate float64
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
		rate = fees.UserCrossRate.Float64()
		if feeBuilder.IsMaker {
			rate = fees.UserAddRate.Float64()
		}
		if spot {
			rate = fees.UserSpotCrossRate.Float64()
			if feeBuilder.IsMaker {
				rate = fees.UserSpotAddRate.Float64()
			}
		}
	}
	return feeBuilder.PurchasePrice * feeBuilder.Amount * rate, nil
}

// ValidateAPICredentials validates the configured watch address and optional signer relationship.
func (e *Exchange) ValidateAPICredentials(ctx context.Context, a asset.Item) error {
	if a != asset.Empty && !e.SupportsAsset(a) {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	credentials, err := e.GetCredentials(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	_, err = e.validateCachedAuthority(ctx, credentials, true)
	if err != nil {
		return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	return nil
}

// GetHistoricCandlesExtended is unavailable because Hyperliquid retains only the latest 5000 candles.
func (e *Exchange) GetHistoricCandlesExtended(context.Context, currency.Pair, asset.Item, kline.Interval, time.Time, time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLeverage returns the configured cross or isolated leverage for a perpetual market.
func (e *Exchange) GetLeverage(ctx context.Context, a asset.Item, p currency.Pair, marginType margin.Type, _ order.Side) (float64, error) {
	if a != asset.PerpetualContract {
		return 0, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if p.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
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
	value := data.Leverage.Value.Float64()
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errInvalidLeverage
	}
	return value, nil
}

// GetFuturesContractDetails is not supported by the initial integration.
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns current hourly rates for one or all active perpetual markets.
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
		e.pairMappingsMu.RLock()
		mappings = slices.Clone(e.pairMappings[asset.PerpetualContract])
		e.pairMappingsMu.RUnlock()
		if len(mappings) == 0 {
			if _, err := e.FetchTradablePairs(ctx, asset.PerpetualContract); err != nil {
				return nil, err
			}
			e.pairMappingsMu.RLock()
			mappings = slices.Clone(e.pairMappings[asset.PerpetualContract])
			e.pairMappingsMu.RUnlock()
		}
	} else {
		mapping, err := e.getPairMapping(ctx, arg.Pair, arg.Asset)
		if err != nil {
			return nil, err
		}
		mappings = []pairMapping{mapping}
	}
	checked := time.Now().UTC()
	responses := make([]fundingrate.LatestRateResponse, 0, len(mappings))
	contextsByDEX := make(map[string]*PerpetualMetadataAndAssetContexts)
	for i := range mappings {
		contexts := contextsByDEX[mappings[i].dex]
		if contexts == nil {
			var err error
			contexts, err = e.GetPerpetualMetadataAndAssetContextsForDEX(ctx, mappings[i].dex)
			if err != nil {
				return nil, err
			}
			if len(contexts.Metadata.Universe) != len(contexts.AssetContexts) {
				return nil, fmt.Errorf("%w: DEX %q expected %d contexts, got %d", errUnexpectedResponseLength, mappings[i].dex, len(contexts.Metadata.Universe), len(contexts.AssetContexts))
			}
			contextsByDEX[mappings[i].dex] = contexts
		}
		index := -1
		for j := range contexts.Metadata.Universe {
			if contexts.Metadata.Universe[j].Name == mappings[i].coin && !contexts.Metadata.Universe[j].IsDelisted {
				index = j
				break
			}
		}
		if index == -1 {
			return nil, fmt.Errorf("%w: %s", errAssetContextNotFound, mappings[i].coin)
		}
		responses = append(responses, fundingrate.LatestRateResponse{
			Exchange:    e.Name,
			Asset:       asset.PerpetualContract,
			Pair:        mappings[i].pair,
			TimeChecked: checked,
			LatestRate: fundingrate.Rate{
				Time: checked.Truncate(time.Hour),
				Rate: contexts.AssetContexts[index].Funding.Decimal(),
			},
			TimeOfNextRate: checked.Truncate(time.Hour).Add(time.Hour),
		})
	}
	if len(responses) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}
	return responses, nil
}

// GetHistoricalFundingRates returns paginated hourly funding rates for one market.
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
		records, err := e.GetFundingHistory(ctx, mapping.coin, cursor, arg.EndDate)
		if err != nil {
			return nil, err
		}
		if len(records) > maximumFundingHistoryCount {
			return nil, fmt.Errorf(
				"%w: funding page has %d entries, maximum %d",
				errUnexpectedResponseLength,
				len(records),
				maximumFundingHistoryCount)
		}
		for i := range records {
			recordTime := records[i].Time.Time().UTC()
			if records[i].Coin != mapping.coin ||
				recordTime.Before(cursor) ||
				recordTime.After(arg.EndDate) ||
				(i > 0 && !recordTime.After(records[i-1].Time.Time())) {
				return nil, fmt.Errorf("%w: malformed funding history page", errUnexpectedResponseLength)
			}
			result.FundingRates = append(result.FundingRates, fundingrate.Rate{
				Time: recordTime,
				Rate: records[i].FundingRate.Decimal(),
			})
		}
		if len(records) < maximumFundingHistoryCount {
			break
		}
		last := records[len(records)-1].Time.Time().UTC()
		cursor = last.Add(time.Millisecond)
	}
	if len(result.FundingRates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}
	result.LatestRate = result.FundingRates[len(result.FundingRates)-1]
	return result, nil
}

// GetOpenInterest returns current base-unit open interest for selected or all active perpetual markets.
func (e *Exchange) GetOpenInterest(ctx context.Context, requested ...key.PairAsset) ([]futures.OpenInterest, error) {
	mappings := make([]pairMapping, 0, len(requested))
	if len(requested) == 0 {
		e.pairMappingsMu.RLock()
		mappings = slices.Clone(e.pairMappings[asset.PerpetualContract])
		e.pairMappingsMu.RUnlock()
		if len(mappings) == 0 {
			if _, err := e.FetchTradablePairs(ctx, asset.PerpetualContract); err != nil {
				return nil, err
			}
			e.pairMappingsMu.RLock()
			mappings = slices.Clone(e.pairMappings[asset.PerpetualContract])
			e.pairMappingsMu.RUnlock()
		}
	} else {
		for i := range requested {
			if requested[i].Asset != asset.PerpetualContract {
				return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, requested[i].Asset)
			}
			mapping, err := e.getPairMapping(ctx, requested[i].Pair(), requested[i].Asset)
			if err != nil {
				return nil, err
			}
			mappings = append(mappings, mapping)
		}
	}
	result := make([]futures.OpenInterest, 0, len(mappings))
	contextsByDEX := make(map[string]*PerpetualMetadataAndAssetContexts)
	for i := range mappings {
		contexts := contextsByDEX[mappings[i].dex]
		if contexts == nil {
			var err error
			contexts, err = e.GetPerpetualMetadataAndAssetContextsForDEX(ctx, mappings[i].dex)
			if err != nil {
				return nil, err
			}
			if len(contexts.Metadata.Universe) != len(contexts.AssetContexts) {
				return nil, fmt.Errorf("%w: DEX %q expected %d contexts, got %d", errUnexpectedResponseLength, mappings[i].dex, len(contexts.Metadata.Universe), len(contexts.AssetContexts))
			}
			contextsByDEX[mappings[i].dex] = contexts
		}
		index := -1
		for j := range contexts.Metadata.Universe {
			if contexts.Metadata.Universe[j].Name == mappings[i].coin && !contexts.Metadata.Universe[j].IsDelisted {
				index = j
				break
			}
		}
		if index == -1 {
			return nil, fmt.Errorf("%w: %s", errAssetContextNotFound, mappings[i].coin)
		}
		result = append(result, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, asset.PerpetualContract, mappings[i].pair),
			OpenInterest: contexts.AssetContexts[index].OpenInterest.Float64(),
		})
	}
	return result, nil
}

// GetCurrencyTradeURL is not supported by the initial integration.
func (e *Exchange) GetCurrencyTradeURL(context.Context, asset.Item, currency.Pair) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits is not supported by the current integration.
func (e *Exchange) UpdateOrderExecutionLimits(context.Context, asset.Item) error {
	return common.ErrNotYetImplemented
}
