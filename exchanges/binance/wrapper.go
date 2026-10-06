package binance

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"iter"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var defaultAssetPairStores = map[asset.Item]currency.PairStore{
	asset.Spot: {
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true},
	},
	asset.Margin: {
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true},
	},
	asset.USDTMarginedFutures: {
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	},
	asset.CoinMarginedFutures: {
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	},
	asset.Options: {
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	},
}

var errDefaultNetworkNotFound = errors.New("default network not found")

// SetDefaults sets the basic defaults for Binance
func (e *Exchange) SetDefaults() {
	e.Name = "Binance"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	for a, ps := range defaultAssetPairStores {
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	// Margin trades on the spot order books, but the streams are only subscribed and stored for spot, so margin market
	// data comes over REST
	if err := e.DisableAssetWebsocketSupport(asset.Margin); err != nil {
		log.Errorf(log.ExchangeSys, "%s error disabling %q asset type websocket support: %s", e.Name, asset.Margin, err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:                true,
			Websocket:           true,
			MaximumOrderHistory: kline.OneDay.Duration() * 7,
			RESTCapabilities: protocol.Features{
				TickerBatching:                 true,
				TickerFetching:                 true,
				KlineFetching:                  true,
				OrderbookFetching:              true,
				AutoPairUpdates:                true,
				AccountInfo:                    true,
				CryptoDeposit:                  true,
				CryptoWithdrawal:               true,
				GetOrder:                       true,
				GetOrders:                      true,
				CancelOrders:                   true,
				CancelOrder:                    true,
				SubmitOrder:                    true,
				ModifyOrder:                    true,
				DepositHistory:                 true,
				WithdrawalHistory:              true,
				TradeFetching:                  true,
				UserTradeHistory:               true,
				TradeFee:                       true,
				CryptoWithdrawalFee:            true,
				MultiChainDeposits:             true,
				MultiChainWithdrawals:          true,
				HasAssetTypeAccountSegregation: true,
				FundingRateFetching:            true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:          true,
				TickerFetching:         true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrder:               true,
				GetOrders:              true,
				Subscribe:              true,
				Unsubscribe:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				Positions:      true,
				Leverage:       true,
				CollateralMode: true,
				FundingRates:   true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.OneHour:   true,
					kline.FourHour:  true,
					kline.EightHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.USDTMarginedFutures: true,
					asset.CoinMarginedFutures: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported: true,
				},
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				// One second klines are spot and margin only; the other intervals are common to every asset
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.ThousandMilliseconds},
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.EightHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 1000,
			},
		},
		Subscriptions: subscription.List{
			{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
		},
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimits()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	// Spot and SAPI requests go through RestSpotSupplementary, as they always have, so saved configurations keep
	// working; RestSpot and EdgeCase1 keep their historical defaults for the same reason and are not used
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                 spotAPIURL,
		exchange.RestSpotSupplementary:    apiURL,
		exchange.RestUSDTMargined:         ufuturesAPIURL,
		exchange.RestCoinMargined:         cfuturesAPIURL,
		exchange.RestOptions:              eOptionAPIURL,
		exchange.RestFuturesSupplementary: pMarginAPIURL,
		exchange.EdgeCase1:                "https://www.binance.com",
		exchange.WebsocketSpot:            binanceDefaultWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup takes in the supplied exchange configuration details and sets params
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
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exch,
		Features:                     &e.Features.Supports.WebsocketCapabilities,
		TradeFeed:                    e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement: true,
		// WebSocket API requests draw on the same weight pools as their REST equivalents, while stream control messages
		// fall back to each connection's own message rate limit
		RateLimitDefinitions: e.Requester.GetRateLimiterDefinitions(),
		// Spot, USDⓈ-M and COIN-M connections carry up to 1024 streams but options connections only 200; the manager
		// takes one limit for every connection, so the smallest applies
		MaxWebsocketSubscriptionsPerConnection: 200,
	}); err != nil {
		return err
	}

	// One synchronisation keeps every asset's books, so a book's state does not depend on which connection carries its
	// stream or how often connections are opened
	e.orderbookSync = orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
		FetchDelay:         orderbookmanager.DefaultWSOrderbookUpdateTimeDelay,
		FetchDeadline:      orderbookmanager.DefaultWSOrderbookUpdateDeadline,
		FetchOrderbook:     e.fetchOrderbookSnapshot,
		CheckPendingUpdate: checkPendingUpdate,
		Orderbook:          &e.Websocket.Orderbook,
	})

	if err := e.Websocket.SetupNewConnection(e.spotStreamConnectionSetup(exch)); err != nil {
		return err
	}
	// A connection that fails to connect stops every other connection, so the WebSocket API is only connected when
	// spot or margin, the accounts it serves, is enabled
	if e.CurrencyPairs.IsAssetEnabled(asset.Spot) == nil || e.CurrencyPairs.IsAssetEnabled(asset.Margin) == nil {
		if err := e.Websocket.SetupNewConnection(e.spotWebsocketAPIConnectionSetup(exch)); err != nil {
			return err
		}
	}

	// Derivatives stream connections: USDⓈ-M /public, /market and /private, COIN-M market and user data, and options
	// /public, /market and /private. A generator returns nothing for a disabled asset, or for a private connection
	// when authenticated websocket use is unavailable, so the manager skips that connection instead of failing all
	for _, c := range []struct {
		filter                 string
		url                    string
		generate               func() (subscription.List, error)
		subscribe, unsubscribe func(context.Context, websocket.Connection, subscription.List) error
		handle                 func(context.Context, websocket.Connection, []byte) error
	}{
		{usdtmPublicFilter, fstreamPublicURL, e.generateUFuturesPublicSubscriptions, e.SubscribeDerivatives, e.UnsubscribeDerivatives, e.wsHandleUFuturesData},
		{usdtmMarketFilter, fstreamMarketURL, e.generateUFuturesMarketSubscriptions, e.SubscribeDerivatives, e.UnsubscribeDerivatives, e.wsHandleUFuturesData},
		{usdtmPrivateFilter, fstreamPrivateURL, e.generateUFuturesUserDataSubscriptions, e.SubscribeUserData, e.UnsubscribeUserData, e.wsHandleUFuturesUserData},
		{coinmFilter, binanceCFuturesWebsocketURL, e.generateCFuturesSubscriptions, e.SubscribeDerivatives, e.UnsubscribeDerivatives, e.wsHandleCFuturesData},
		{coinmPrivateFilter, binanceCFuturesWebsocketURL, e.generateCFuturesUserDataSubscriptions, e.SubscribeUserData, e.UnsubscribeUserData, e.wsHandleCFuturesUserData},
		{optionsPublicFilter, fstreamPublicURL, e.generateOptionsPublicSubscriptions, e.SubscribeDerivatives, e.UnsubscribeDerivatives, e.wsHandleOptionsData},
		{optionsMarketFilter, fstreamMarketURL, e.generateOptionsMarketSubscriptions, e.SubscribeDerivatives, e.UnsubscribeDerivatives, e.wsHandleOptionsData},
		{optionsPrivateFilter, fstreamPrivateURL, e.generateOptionsUserDataSubscriptions, e.SubscribeUserData, e.UnsubscribeUserData, e.wsHandleOptionsUserData},
	} {
		if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
			MessageFilter:         c.filter,
			URL:                   c.url,
			Connector:             e.WsDerivativesConnect,
			GenerateSubscriptions: c.generate,
			Subscriber:            c.subscribe,
			Unsubscriber:          c.unsubscribe,
			Handler:               c.handle,
			ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
			RateLimit:             request.NewWeightedRateLimitByDuration(250 * time.Millisecond),
		}); err != nil {
			return err
		}
	}
	return nil
}

// symbolStatusTrading is the status of a symbol or contract that is open for trading
const symbolStatusTrading = "TRADING"

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	var pairs currency.Pairs
	switch a {
	case asset.Spot, asset.Margin:
		info, err := e.GetExchangeInfo(ctx, nil)
		if err != nil {
			return nil, err
		}
		pairs = make(currency.Pairs, 0, len(info.Symbols))
		for i := range info.Symbols {
			if s := &info.Symbols[i]; s.Status == symbolStatusTrading && spotSymbolTradable(s, a) {
				pairs = append(pairs, currency.NewPair(s.BaseAsset, s.QuoteAsset))
			}
		}
	case asset.USDTMarginedFutures:
		info, err := e.UExchangeInfo(ctx)
		if err != nil {
			return nil, err
		}
		pairs = make(currency.Pairs, 0, len(info.Symbols))
		for i := range info.Symbols {
			if s := &info.Symbols[i]; s.Status == symbolStatusTrading {
				pairs = append(pairs, uFuturesSymbolPair(s.Symbol, s.BaseAsset, s.QuoteAsset))
			}
		}
	case asset.CoinMarginedFutures:
		info, err := e.FuturesExchangeInfo(ctx)
		if err != nil {
			return nil, err
		}
		pairs = make(currency.Pairs, 0, len(info.Symbols))
		for i := range info.Symbols {
			if info.Symbols[i].ContractStatus != symbolStatusTrading {
				continue
			}
			// COIN-M pairs keep master's representation, the contract pair as base and the contract type or
			// delivery date as quote (BTCUSD_PERP is BTCUSD/PERP), so saved configurations still match
			pair, err := currency.NewPairDelimiter(info.Symbols[i].Symbol, currency.UnderscoreDelimiter)
			if err != nil {
				log.Warnf(log.ExchangeSys, "%s skipping %s symbol: %s", e.Name, a, err)
				continue
			}
			pairs = append(pairs, pair)
		}
	case asset.Options:
		info, err := e.GetOptionsExchangeInformation(ctx)
		if err != nil {
			return nil, err
		}
		pairs = make(currency.Pairs, 0, len(info.OptionSymbols))
		for i := range info.OptionSymbols {
			if info.OptionSymbols[i].Status != symbolStatusTrading {
				continue
			}
			// The underlying asset is the base and the rest of the symbol the quote: BTC-261225-85000-C is
			// BTC/261225-85000-C
			pair, err := currency.NewPairDelimiter(info.OptionSymbols[i].Symbol, currency.DashDelimiter)
			if err != nil {
				log.Warnf(log.ExchangeSys, "%s skipping %s symbol: %s", e.Name, a, err)
				continue
			}
			pairs = append(pairs, pair)
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	format, err := e.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	return pairs.Format(format), nil
}

// spotSymbolTradable reports whether a spot exchange information symbol can be traded as the spot or margin asset.
// Permission sets list the account permissions an order needs, not the assets a symbol trades on
func spotSymbolTradable(s *SymbolInfo, a asset.Item) bool {
	if a == asset.Margin {
		return s.IsMarginTradingAllowed
	}
	return s.IsSpotTradingAllowed
}

// uFuturesSymbolPair returns the pair of a USDⓈ-M contract. A perpetual's base and quote assets cannot be split from
// its symbol alone (TSLAUSDT is TSLA/USDT), while a delivery contract keeps its contract pair as base and its delivery
// date as quote (BTCUSDT_261225 is BTCUSDT/261225)
func uFuturesSymbolPair(symbol string, baseAsset, quoteAsset currency.Code) currency.Pair {
	if contractPair, deliveryDate, ok := strings.Cut(symbol, currency.UnderscoreDelimiter); ok {
		return currency.NewPair(currency.NewCode(contractPair), currency.NewCode(deliveryDate))
	}
	return currency.NewPair(baseAsset, quoteAsset)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	// A failing asset does not stop the others from updating
	var errs error
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(ctx, a)
		if err == nil {
			err = e.UpdatePairs(pairs, a, false)
		}
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s: %w", a, err))
		}
	}
	return common.AppendError(errs, e.EnsureOnePairEnabled())
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	var prices []ticker.Price
	switch a {
	case asset.Spot, asset.Margin:
		// A request without symbols returns every ticker, while one with symbols takes at most 100
		stats, err := e.GetPriceChangeStats(ctx, &PriceChangeStatsRequest{})
		if err != nil {
			return err
		}
		prices = make([]ticker.Price, 0, len(stats))
		for i := range stats {
			pair, ok, err := e.tickerPair(stats[i].Symbol, a)
			if err != nil {
				return err
			}
			if ok {
				prices = append(prices, e.spotTickerPrice(&stats[i], pair, a))
			}
		}
	case asset.USDTMarginedFutures:
		stats, err := e.U24HourTickerPriceChangeStats(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		prices = make([]ticker.Price, 0, len(stats))
		for i := range stats {
			pair, ok, err := e.tickerPair(stats[i].Symbol, a)
			if err != nil {
				return err
			}
			if ok {
				prices = append(prices, e.uFuturesTickerPrice(&stats[i], pair))
			}
		}
	case asset.CoinMarginedFutures:
		stats, err := e.GetFuturesSwapTickerChangeStats(ctx, currency.EMPTYPAIR, currency.EMPTYCODE)
		if err != nil {
			return err
		}
		prices = make([]ticker.Price, 0, len(stats))
		for i := range stats {
			pair, ok, err := e.tickerPair(stats[i].Symbol, a)
			if err != nil {
				return err
			}
			if ok {
				prices = append(prices, e.cFuturesTickerPrice(&stats[i], pair))
			}
		}
	case asset.Options:
		stats, err := e.GetEOptions24hrTickerPriceChangeStatistics(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		prices = make([]ticker.Price, 0, len(stats))
		for i := range stats {
			pair, ok, err := e.tickerPair(stats[i].Symbol, a)
			if err != nil {
				return err
			}
			if ok {
				prices = append(prices, e.optionsTickerPrice(&stats[i], pair))
			}
		}
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	_, err := ticker.ProcessBatch(prices)
	return err
}

// tickerPair returns the available pair of a ticker's symbol, and false for a symbol without one: ticker lists include
// symbols that are not trading
func (e *Exchange) tickerPair(symbol string, a asset.Item) (currency.Pair, bool, error) {
	pair, err := e.MatchSymbolWithAvailablePairs(symbol, a, true)
	switch {
	case err == nil:
		return pair, true, nil
	case errors.Is(err, currency.ErrPairNotFound):
		return currency.EMPTYPAIR, false, nil
	default:
		return currency.EMPTYPAIR, false, err
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var price ticker.Price
	switch a {
	case asset.Spot, asset.Margin:
		stats, err := e.GetPriceChangeStats(ctx, &PriceChangeStatsRequest{Symbols: currency.Pairs{p}})
		if err != nil {
			return nil, err
		}
		if len(stats) == 0 {
			return nil, fmt.Errorf("%w: %s %s", ticker.ErrTickerNotFound, p, a)
		}
		price = e.spotTickerPrice(&stats[0], p, a)
	case asset.USDTMarginedFutures:
		stats, err := e.U24HourTickerPriceChangeStats(ctx, p)
		if err != nil {
			return nil, err
		}
		if len(stats) == 0 {
			return nil, fmt.Errorf("%w: %s %s", ticker.ErrTickerNotFound, p, a)
		}
		price = e.uFuturesTickerPrice(&stats[0], p)
	case asset.CoinMarginedFutures:
		stats, err := e.GetFuturesSwapTickerChangeStats(ctx, p, currency.EMPTYCODE)
		if err != nil {
			return nil, err
		}
		if len(stats) == 0 {
			return nil, fmt.Errorf("%w: %s %s", ticker.ErrTickerNotFound, p, a)
		}
		price = e.cFuturesTickerPrice(&stats[0], p)
	case asset.Options:
		stats, err := e.GetEOptions24hrTickerPriceChangeStatistics(ctx, p)
		if err != nil {
			return nil, err
		}
		if len(stats) == 0 {
			return nil, fmt.Errorf("%w: %s %s", ticker.ErrTickerNotFound, p, a)
		}
		price = e.optionsTickerPrice(&stats[0], p)
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err := ticker.ProcessTicker(&price); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// spotTickerPrice converts a spot or margin 24hr ticker. Close is the previous close, as the ticker stream reports it
func (e *Exchange) spotTickerPrice(t *PriceChangeStats, pair currency.Pair, a asset.Item) ticker.Price {
	return ticker.Price{
		Last:                       t.LastPrice.Float64(),
		LastSize:                   t.LastQuantity.Float64(),
		VolumeWeightedAveragePrice: t.WeightedAveragePrice.Float64(),
		High:                       t.HighPrice.Float64(),
		Low:                        t.LowPrice.Float64(),
		Bid:                        t.BidPrice.Float64(),
		BidSize:                    t.BidQuantity.Float64(),
		Ask:                        t.AskPrice.Float64(),
		AskSize:                    t.AskQuantity.Float64(),
		BaseVolume:                 t.Volume.Float64(),
		QuoteVolume:                t.QuoteVolume.Float64(),
		Open:                       t.OpenPrice.Float64(),
		PercentChange24Hour:        t.PriceChangePercent.Float64(),
		Close:                      t.PreviousClosePrice.Float64(),
		Pair:                       pair,
		ExchangeName:               e.Name,
		AssetType:                  a,
		LastUpdated:                t.CloseTime.Time(),
	}
}

// uFuturesTickerPrice converts a USDⓈ-M 24hr ticker, which has no best bid or ask
func (e *Exchange) uFuturesTickerPrice(t *U24HourPriceChangeStats, pair currency.Pair) ticker.Price {
	return ticker.Price{
		Last:                       t.LastPrice.Float64(),
		LastSize:                   t.LastQuantity.Float64(),
		VolumeWeightedAveragePrice: t.WeightedAveragePrice.Float64(),
		High:                       t.HighPrice.Float64(),
		Low:                        t.LowPrice.Float64(),
		BaseVolume:                 t.Volume.Float64(),
		QuoteVolume:                t.QuoteVolume.Float64(),
		Open:                       t.OpenPrice.Float64(),
		PercentChange24Hour:        t.PriceChangePercent.Float64(),
		Pair:                       pair,
		ExchangeName:               e.Name,
		AssetType:                  asset.USDTMarginedFutures,
		LastUpdated:                t.CloseTime.Time(),
	}
}

// cFuturesTickerPrice converts a COIN-M 24hr ticker, which has no best bid or ask. Its volume counts contracts, so the
// base volume comes from the base asset volume, and the quantities stay in contracts as the ticker stream sends them
func (e *Exchange) cFuturesTickerPrice(t *CFuturesPriceChangeStats, pair currency.Pair) ticker.Price {
	return ticker.Price{
		Last:                       t.LastPrice.Float64(),
		LastSize:                   t.LastQuantity.Float64(),
		VolumeWeightedAveragePrice: t.WeightedAveragePrice.Float64(),
		High:                       t.HighPrice.Float64(),
		Low:                        t.LowPrice.Float64(),
		BaseVolume:                 t.BaseVolume.Float64(),
		Open:                       t.OpenPrice.Float64(),
		PercentChange24Hour:        t.PriceChangePercent.Float64(),
		Pair:                       pair,
		ExchangeName:               e.Name,
		AssetType:                  asset.CoinMarginedFutures,
		LastUpdated:                t.CloseTime.Time(),
	}
}

// optionsTickerPrice converts an options 24hr ticker. Its price change percent is a fraction, and its close time is the
// time of the last trade rather than of the statistics, so the ticker is stamped when it is stored
func (e *Exchange) optionsTickerPrice(t *EOptionTicker, pair currency.Pair) ticker.Price {
	return ticker.Price{
		Last:                t.LastPrice.Float64(),
		LastSize:            t.LastQuantity.Float64(),
		High:                t.High.Float64(),
		Low:                 t.Low.Float64(),
		Bid:                 t.BidPrice.Float64(),
		Ask:                 t.AskPrice.Float64(),
		BaseVolume:          t.Volume.Float64(),
		QuoteVolume:         t.Amount.Float64(),
		Open:                t.Open.Float64(),
		PercentChange24Hour: t.PriceChangePercent.Float64() * 100,
		Pair:                pair,
		ExchangeName:        e.Name,
		AssetType:           asset.Options,
	}
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var book *orderbook.Book
	switch a {
	case asset.Spot, asset.Margin:
		resp, err := e.GetOrderBook(ctx, &OrderBookRequest{Symbol: p, Limit: 1000})
		if err != nil {
			return nil, err
		}
		// The spot order book has no timestamp, so processing stamps it
		book = &orderbook.Book{
			Bids:              resp.Bids.Levels(),
			Asks:              resp.Asks.Levels(),
			Exchange:          e.Name,
			Pair:              p,
			Asset:             a,
			LastUpdateID:      resp.LastUpdateID,
			ValidateOrderbook: e.ValidateOrderbook,
		}
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		var err error
		if book, err = e.fetchDerivativesOrderbook(ctx, p, a); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, a)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	subAccount := accounts.NewSubAccount(assetType, "")
	switch assetType {
	case asset.Spot:
		creds, err := e.GetCredentials(ctx)
		if err != nil {
			return nil, err
		}
		if creds.SubAccount != "" {
			// TODO: implement sub-account endpoints
			return nil, common.ErrNotYetImplemented
		}
		resp, err := e.GetAccount(ctx, false)
		if err != nil {
			return nil, err
		}
		for i := range resp.Balances {
			free := resp.Balances[i].Free.InexactFloat64()
			locked := resp.Balances[i].Locked.InexactFloat64()
			subAccount.Balances.Set(resp.Balances[i].Asset, accounts.Balance{
				Total: free + locked,
				Hold:  locked,
				Free:  free,
			})
		}
	case asset.Margin:
		resp, err := e.GetCrossMarginAccountDetail(ctx)
		if err != nil {
			return nil, err
		}
		for i := range resp.UserAssets {
			free := resp.UserAssets[i].Free.Float64()
			borrowed := resp.UserAssets[i].Borrowed.Float64()
			subAccount.Balances.Set(resp.UserAssets[i].Asset, accounts.Balance{
				Total:                  free + resp.UserAssets[i].Locked.Float64(),
				Hold:                   resp.UserAssets[i].Locked.Float64(),
				Free:                   free,
				AvailableWithoutBorrow: free - borrowed,
				Borrowed:               borrowed,
			})
		}
	case asset.USDTMarginedFutures:
		resp, err := e.UAccountBalanceV3(ctx)
		if err != nil {
			return nil, err
		}
		// The account alias names the account itself, which the user data stream does not send, so the balances are kept
		// under the main account where the stream updates them
		for i := range resp {
			subAccount.Balances.Set(resp[i].Asset, futuresBalance(resp[i].Balance, resp[i].AvailableBalance))
		}
	case asset.CoinMarginedFutures:
		resp, err := e.GetFuturesAccountInfo(ctx)
		if err != nil {
			return nil, err
		}
		for i := range resp.Assets {
			subAccount.Balances.Set(resp.Assets[i].Asset, futuresBalance(resp.Assets[i].WalletBalance, resp.Assets[i].AvailableBalance))
		}
	case asset.Options:
		resp, err := e.GetOptionMarginAccountInformation(ctx)
		if err != nil {
			return nil, err
		}
		for i := range resp.Asset {
			subAccount.Balances.Set(resp.Asset[i].Asset, accounts.Balance{
				Total: resp.Asset[i].MarginBalance.Float64(),
				Hold:  resp.Asset[i].InitialMargin.Float64(),
				Free:  resp.Asset[i].Available.Float64(),
			})
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	subAccts := accounts.SubAccounts{subAccount}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// futuresBalance returns a futures margin asset's balance from its wallet and available balances. Unrealised profit can
// lift the available balance above the wallet balance, which holds nothing
func futuresBalance(walletBalance, availableBalance types.Number) accounts.Balance {
	total, free := walletBalance.Float64(), availableBalance.Float64()
	return accounts.Balance{
		Total: total,
		Hold:  max(total-free, 0),
		Free:  free,
	}
}

// GetAccountFundingHistory returns the deposits and withdrawals of the last 90 days, the window Binance returns by
// default
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	deposits, err := e.allDeposits(ctx)
	if err != nil {
		return nil, err
	}
	withdrawals, err := e.allWithdrawals(ctx, currency.EMPTYCODE)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(deposits)+len(withdrawals))
	for i := range deposits {
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:      e.Name,
			Status:            depositStatusString(deposits[i].Status),
			TransferID:        deposits[i].ID,
			Timestamp:         deposits[i].InsertTime.Time(),
			Currency:          deposits[i].Coin.String(),
			Amount:            deposits[i].Amount.Float64(),
			TransferType:      "deposit",
			CryptoToAddress:   deposits[i].Address,
			CryptoFromAddress: deposits[i].SourceAddress,
			CryptoTxID:        deposits[i].TxID,
			CryptoChain:       deposits[i].Network,
		})
	}
	for i := range withdrawals {
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          withdrawalStatusString(withdrawals[i].Status),
			TransferID:      withdrawals[i].ID,
			Description:     withdrawals[i].Info,
			Timestamp:       withdrawals[i].ApplyTime.Time(),
			Currency:        withdrawals[i].Coin.String(),
			Amount:          withdrawals[i].Amount.Float64(),
			Fee:             withdrawals[i].TransactionFee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals[i].Address,
			CryptoTxID:      withdrawals[i].TxID,
			CryptoChain:     withdrawals[i].Network,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns the withdrawals of a currency, or of every currency when it is empty, from the last 90
// days, the window Binance returns by default
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.allWithdrawals(ctx, c)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals))
	for i := range withdrawals {
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawalStatusString(withdrawals[i].Status),
			TransferID:      withdrawals[i].ID,
			Description:     withdrawals[i].Info,
			Timestamp:       withdrawals[i].ApplyTime.Time(),
			Currency:        withdrawals[i].Coin.String(),
			Amount:          withdrawals[i].Amount.Float64(),
			Fee:             withdrawals[i].TransactionFee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals[i].Address,
			CryptoTxID:      withdrawals[i].TxID,
			CryptoChain:     withdrawals[i].Network,
		}
	}
	return resp, nil
}

// fundingHistoryPageLimit is the most deposit or withdrawal records one request returns
const fundingHistoryPageLimit = 1000

// allDeposits returns every deposit of the default window with its source address, requesting pages until one is not
// full
func (e *Exchange) allDeposits(ctx context.Context) ([]DepositRecord, error) {
	var deposits []DepositRecord
	for {
		page, err := e.DepositHistory(ctx, &DepositHistoryRequest{
			IncludeSource: true,
			Offset:        uint64(len(deposits)),
			Limit:         fundingHistoryPageLimit,
		})
		if err != nil {
			return nil, err
		}
		deposits = append(deposits, page...)
		if len(page) < fundingHistoryPageLimit {
			return deposits, nil
		}
	}
}

// allWithdrawals returns every withdrawal of a coin, or of every coin, in the default window, requesting pages until one
// is not full
func (e *Exchange) allWithdrawals(ctx context.Context, coin currency.Code) ([]WithdrawalRecord, error) {
	var withdrawals []WithdrawalRecord
	for {
		page, err := e.WithdrawHistory(ctx, &WithdrawHistoryRequest{
			Coin:   coin,
			Offset: uint64(len(withdrawals)),
			Limit:  fundingHistoryPageLimit,
		})
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, page...)
		if len(page) < fundingHistoryPageLimit {
			return withdrawals, nil
		}
	}
}

// depositStatusString returns the documented name of a deposit status, or the code of an undocumented one
func depositStatusString(status uint64) string {
	switch status {
	case 0:
		return "Pending"
	case 1:
		return "Success"
	case 2:
		return "Rejected"
	case 6:
		return "Credited but cannot withdraw"
	case 7:
		return "Wrong deposit"
	case 8:
		return "Waiting user confirm"
	default:
		return strconv.FormatUint(status, 10)
	}
}

// withdrawalStatusString returns the documented name of a withdrawal status, or the code of an undocumented one
func withdrawalStatusString(status uint64) string {
	switch status {
	case 0:
		return "Email sent"
	case 2:
		return "Awaiting approval"
	case 3:
		return "Rejected"
	case 4:
		return "Processing"
	case 6:
		return "Completed"
	default:
		return strconv.FormatUint(status, 10)
	}
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	var resp []trade.Data
	switch a {
	case asset.Spot, asset.Margin:
		trades, err := e.GetMostRecentTrades(ctx, &RecentTradeRequest{Symbol: p, Limit: 1000})
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, 0, len(trades))
		for _, t := range trades {
			resp = append(resp, trade.Data{
				TID:          strconv.FormatUint(t.ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         tradeTakerSide(t.IsBuyerMaker),
				Price:        t.Price.Float64(),
				Amount:       t.Quantity.Float64(),
				Timestamp:    t.Time.Time(),
			})
		}
	case asset.USDTMarginedFutures:
		trades, err := e.URecentTrades(ctx, p, 1000)
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, 0, len(trades))
		for i := range trades {
			resp = append(resp, trade.Data{
				TID:          strconv.FormatUint(trades[i].ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         tradeTakerSide(trades[i].IsBuyerMaker),
				Price:        trades[i].Price.Float64(),
				Amount:       trades[i].Quantity.Float64(),
				Timestamp:    trades[i].Time.Time(),
			})
		}
	case asset.CoinMarginedFutures:
		trades, err := e.GetFuturesPublicTrades(ctx, p, 1000)
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, 0, len(trades))
		for i := range trades {
			// The amount counts contracts, as the trade stream, klines and tickers do
			resp = append(resp, trade.Data{
				TID:          strconv.FormatUint(trades[i].ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         tradeTakerSide(trades[i].IsBuyerMaker),
				Price:        trades[i].Price.Float64(),
				Amount:       trades[i].Quantity.Float64(),
				Timestamp:    trades[i].Time.Time(),
			})
		}
	case asset.Options:
		trades, err := e.GetEOptionsRecentTrades(ctx, p, 500)
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, 0, len(trades))
		for i := range trades {
			var side order.Side
			switch trades[i].Side {
			case 1:
				side = order.Buy
			case -1:
				side = order.Sell
			default:
				log.Warnf(log.ExchangeSys, "%s skipping %s trade %d with unknown side %d", e.Name, a, trades[i].TradeID, trades[i].Side)
				continue
			}
			// The trade ID matches the trade stream's; the ID is unique across symbols
			resp = append(resp, trade.Data{
				TID:          strconv.FormatUint(trades[i].TradeID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         side,
				Price:        trades[i].Price.Float64(),
				Amount:       trades[i].Quantity.Float64(),
				Timestamp:    trades[i].Time.Time(),
			})
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return resp, nil
}

// tradeTakerSide returns the side of a trade's taker, which sold when the buyer made the market
func tradeTakerSide(isBuyerMaker bool) order.Side {
	if isBuyerMaker {
		return order.Sell
	}
	return order.Buy
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, from, to time.Time) ([]trade.Data, error) {
	if a != asset.Spot && a != asset.Margin {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	// A limit no window reaches pages through the aggregate trades until the end of the window
	trades, err := e.GetAggregatedTrades(ctx, &AggregatedTradeRequest{
		Symbol:    p,
		StartTime: from,
		EndTime:   to,
		Limit:     math.MaxUint64,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching %s aggregate trades: %w", p, err)
	}
	resp := make([]trade.Data, len(trades))
	for i, t := range trades {
		resp[i] = trade.Data{
			TID:          strconv.FormatUint(t.AggregateTradeID, 10),
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    a,
			Side:         tradeTakerSide(t.IsBuyerMaker),
			Price:        t.Price.Float64(),
			Amount:       t.Quantity.Float64(),
			Timestamp:    t.Timestamp.Time(),
		}
	}
	return resp, nil
}

var (
	errInvalidOrderID                 = errors.New("invalid order ID")
	errTriggerPriceRequired           = errors.New("trigger price is required")
	errTrailingStopPercentageRequired = errors.New("trailing stop orders need a percentage tracking value, their callback rate")
	errBatchCancelResultMissing       = errors.New("cancellation result missing from the reply")
)

// batchCancelLimit is the most orders the derivatives Cancel Multiple Orders endpoints cancel in one request
const batchCancelLimit = 10

// canUseWebsocketAPIForOrders reports whether authenticated spot order requests can go through the WebSocket API
func (e *Exchange) canUseWebsocketAPIForOrders() bool {
	return e.IsAPIStreamConnected() && e.Websocket.CanUseAuthenticatedWebsocketForWrapper()
}

// logOrderIssue logs a value of an order that could not be converted, which leaves that value unset: a listing still
// returns the order, and a placement or modification that Binance accepted is not reported as failed
func (e *Exchange) logOrderIssue(a asset.Item, orderID string, err error) {
	log.Warnf(log.ExchangeSys, "%s %s order %s: %v", e.Name, a, orderID, err)
}

// parseOrderID converts an order ID into Binance's numeric form; an empty ID is 0, which the endpoints leave out
func parseOrderID(orderID string) (uint64, error) {
	if orderID == "" {
		return 0, nil
	}
	id, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w %q: %w", errInvalidOrderID, orderID, err)
	}
	return id, nil
}

// remainingAmount returns an order's unfilled amount, in decimal arithmetic so it is as exact as Binance's amounts
func remainingAmount(original, executed types.Number) float64 {
	return original.Decimal().Sub(executed.Decimal()).InexactFloat64()
}

// averagePrice returns the average price of an order's fills from the quote amount and quantity they total, in decimal
// arithmetic so it is as exact as Binance's amounts
func averagePrice(cumulativeQuote, executed types.Number) float64 {
	if executed <= 0 {
		return 0
	}
	return cumulativeQuote.Decimal().Div(executed.Decimal()).InexactFloat64()
}

// Order type values that Binance's spot and derivatives products share
const (
	limitOrderType        = "LIMIT"
	takeProfitOrderType   = "TAKE_PROFIT"
	trailingStopOrderType = "TRAILING_STOP_MARKET"
)

// orderSideString returns the side Binance takes for an order's direction
func orderSideString(side order.Side) (string, error) {
	switch {
	case side.IsLong():
		return order.Buy.String(), nil
	case side.IsShort():
		return order.Sell.String(), nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrSideIsInvalid, side)
	}
}

// spotOrderTypeString returns the spot and margin order type of t. A post only limit order is a LIMIT_MAKER order,
// Binance's spot post only type
func spotOrderTypeString(t order.Type, tif order.TimeInForce) (string, error) {
	switch t {
	case order.Limit:
		if tif.Is(order.PostOnly) {
			return "LIMIT_MAKER", nil
		}
		return limitOrderType, nil
	case order.LimitMaker:
		return "LIMIT_MAKER", nil
	case order.Market:
		return "MARKET", nil
	case order.StopMarket:
		return "STOP_LOSS", nil
	case order.StopLimit:
		return "STOP_LOSS_LIMIT", nil
	case order.TakeProfitMarket:
		return takeProfitOrderType, nil
	case order.TakeProfitLimit:
		return "TAKE_PROFIT_LIMIT", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
	}
}

// stringToSpotOrderType converts a spot or margin order type, the inverse of spotOrderTypeString; STOP_LOSS and
// TAKE_PROFIT orders become market orders when they trigger
func stringToSpotOrderType(orderType string) (order.Type, error) {
	switch orderType {
	case limitOrderType:
		return order.Limit, nil
	case "MARKET":
		return order.Market, nil
	case "STOP_LOSS":
		return order.StopMarket, nil
	case "STOP_LOSS_LIMIT":
		return order.StopLimit, nil
	case takeProfitOrderType:
		return order.TakeProfitMarket, nil
	case "TAKE_PROFIT_LIMIT":
		return order.TakeProfitLimit, nil
	case "LIMIT_MAKER":
		return order.LimitMaker, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %q", order.ErrUnrecognisedOrderType, orderType)
	}
}

// futuresOrderTypeString returns the USDⓈ-M and COIN-M order type of t, the inverse of derivativesOrderType: STOP and
// TAKE_PROFIT are Binance's stop and take profit limit orders
func futuresOrderTypeString(t order.Type) (string, error) {
	switch t {
	case order.Limit:
		return limitOrderType, nil
	case order.Market:
		return "MARKET", nil
	case order.StopLimit:
		return "STOP", nil
	case order.StopMarket:
		return "STOP_MARKET", nil
	case order.TakeProfitLimit:
		return takeProfitOrderType, nil
	case order.TakeProfitMarket:
		return "TAKE_PROFIT_MARKET", nil
	case order.TrailingStop:
		return trailingStopOrderType, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
	}
}

// resolveTriggerOrderType returns the limit form of a bare stop or take profit order when it has a price and the market
// form otherwise, since Stop and TakeProfit leave open which of Binance's two order types is meant
func resolveTriggerOrderType(t order.Type, price float64) order.Type {
	switch t {
	case order.Stop:
		if price > 0 {
			return order.StopLimit
		}
		return order.StopMarket
	case order.TakeProfit:
		if price > 0 {
			return order.TakeProfitLimit
		}
		return order.TakeProfitMarket
	default:
		return t
	}
}

// isConditionalOrderType reports whether t is a stop, take profit or trailing stop type, which USDⓈ-M places as algo
// orders since Binance moved them off New Order
func isConditionalOrderType(t order.Type) bool {
	switch t {
	case order.Stop, order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitLimit, order.TakeProfitMarket, order.TrailingStop:
		return true
	default:
		return false
	}
}

// timeInForceString returns the time in force a limit priced order on asset a is sent with: GTC unless another is
// requested, post only as USDⓈ-M and COIN-M GTX, and good till day as USDⓈ-M GTD, which GCT's "GTD" string shares
// and which expires at the order's end time
func timeInForceString(a asset.Item, tif order.TimeInForce) (string, error) {
	switch tif {
	case order.UnknownTIF, order.GoodTillCancel:
		return "GTC", nil
	case order.ImmediateOrCancel:
		return "IOC", nil
	case order.FillOrKill:
		return "FOK", nil
	case order.PostOnly, order.GoodTillCrossing, order.PostOnly | order.GoodTillCancel, order.PostOnly | order.GoodTillCrossing:
		if a == asset.USDTMarginedFutures || a == asset.CoinMarginedFutures {
			return "GTX", nil
		}
	case order.GoodTillDay:
		if a == asset.USDTMarginedFutures {
			return "GTD", nil
		}
	}
	return "", fmt.Errorf("%w: %s on %s", order.ErrUnsupportedTimeInForce, tif, a)
}

// workingTypeString returns the price a USDⓈ-M conditional order triggers on: the contract price, Binance's default,
// or the mark price
func workingTypeString(t order.PriceType) (string, error) {
	switch t {
	case order.LastPrice:
		return "", nil
	case order.MarkPrice:
		return "MARK_PRICE", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnknownPriceType, t)
	}
}

// algoOrderStatus converts a USDⓈ-M algo order's status. A TRIGGERING order is being sent to the matching engine and
// a TRIGGERED one rests there as an order of its own, which carries the fills; FINISHED means that order is done
func algoOrderStatus(status string) (order.Status, error) {
	switch status {
	case "NEW":
		return order.New, nil
	case "CANCELED":
		return order.Cancelled, nil
	case "TRIGGERING":
		return order.Pending, nil
	case "TRIGGERED":
		return order.Active, nil
	case "FINISHED":
		return order.Closed, nil
	case "REJECTED":
		return order.Rejected, nil
	case "EXPIRED":
		return order.Expired, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %q", errUnknownOrderStatus, status)
	}
}

// orderListStatus converts an OCO order list's listOrderStatus
func orderListStatus(status string) (order.Status, error) {
	switch status {
	case "EXECUTING":
		return order.Active, nil
	case "ALL_DONE":
		return order.Closed, nil
	case "REJECT":
		return order.Rejected, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %q", errUnknownOrderStatus, status)
	}
}

// spotOrderValues holds the values a spot or margin order is sent with, whichever route places it
type spotOrderValues struct {
	side, orderType, timeInForce                   string
	price, stopPrice, quantity, quoteOrderQuantity float64
}

// newSpotOrderValues works out the values of a spot or margin order. Only limit priced types take a price and a time
// in force, stop and take profit types trigger at TriggerPrice, and a market order without an amount spends or
// receives QuoteAmount
func newSpotOrderValues(s *order.Submit) (*spotOrderValues, error) {
	side, err := orderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	orderType, err := spotOrderTypeString(resolveTriggerOrderType(s.Type, s.Price), s.TimeInForce)
	if err != nil {
		return nil, err
	}
	v := &spotOrderValues{side: side, orderType: orderType}
	switch orderType {
	case limitOrderType, "STOP_LOSS_LIMIT", "TAKE_PROFIT_LIMIT":
		if v.timeInForce, err = timeInForceString(asset.Spot, s.TimeInForce); err != nil {
			return nil, err
		}
		fallthrough
	case "LIMIT_MAKER":
		if s.Price <= 0 {
			return nil, order.ErrPriceMustBeSetIfLimitOrder
		}
		v.price = s.Price
	}
	switch orderType {
	case "STOP_LOSS", "STOP_LOSS_LIMIT", takeProfitOrderType, "TAKE_PROFIT_LIMIT":
		if s.TriggerPrice <= 0 {
			return nil, errTriggerPriceRequired
		}
		v.stopPrice = s.TriggerPrice
	}
	if orderType == "MARKET" && s.Amount == 0 {
		v.quoteOrderQuantity = s.QuoteAmount
	} else {
		v.quantity = s.Amount
	}
	if v.quantity <= 0 && v.quoteOrderQuantity <= 0 {
		return nil, fmt.Errorf("%w: %s orders take a base amount", order.ErrAmountIsInvalid, orderType)
	}
	return v, nil
}

// spotOCOLegs holds the side and the two orders of an OCO order list
type spotOCOLegs struct {
	side                 string
	limitMaker, stopLoss OCOOrderListLeg
}

// newSpotOCOLegs works out an OCO order list's LIMIT_MAKER order at Price and its STOP_LOSS order triggered at
// TriggerPrice, which is a STOP_LOSS_LIMIT order when the stop loss risk management mode has a limit price.
// ClientOrderID identifies the list, so Binance names the legs: deriving their IDs from it could exceed Binance's
// 36 character limit, and their order list ID ties them to the list anyway
func newSpotOCOLegs(s *order.Submit) (*spotOCOLegs, error) {
	side, err := orderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	if s.Price <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	if s.TriggerPrice <= 0 {
		return nil, errTriggerPriceRequired
	}
	legs := &spotOCOLegs{
		side:       side,
		limitMaker: OCOOrderListLeg{Type: "LIMIT_MAKER", Price: s.Price},
		stopLoss:   OCOOrderListLeg{Type: "STOP_LOSS", StopPrice: s.TriggerPrice},
	}
	if limitPrice := s.RiskManagementModes.StopLoss.LimitPrice; limitPrice > 0 {
		legs.stopLoss.Type = "STOP_LOSS_LIMIT"
		legs.stopLoss.Price = limitPrice
		// The limit maker order is post only by type, so the time in force can only be the stop limit order's
		if legs.stopLoss.TimeInForce, err = timeInForceString(asset.Spot, s.TimeInForce&^order.PostOnly); err != nil {
			return nil, err
		}
	}
	return legs, nil
}

// marginSideEffect returns the side effect a margin order is placed with: AutoBorrow borrows what the order needs and
// AutoRepay repays debt with what it receives
func marginSideEffect(s *order.Submit) string {
	switch {
	case s.AutoBorrow && s.AutoRepay:
		return "AUTO_BORROW_REPAY"
	case s.AutoBorrow:
		return "MARGIN_BUY"
	case s.AutoRepay:
		return "AUTO_REPAY"
	default:
		return ""
	}
}

// futuresPositionSide returns the position side and reduce only flag a futures order is sent with. Buy and Sell
// orders trade the one-way position. Long and Short orders follow the account's position mode, fetched for them: in
// Hedge Mode they trade the position of their direction, or with ReduceOnly the other one, which Binance takes as the
// position side since it rejects reduceOnly in Hedge Mode. COIN-M has the USDⓈ-M position mode, which Binance has kept
// both products on since 2026-04-14
func (e *Exchange) futuresPositionSide(ctx context.Context, s *order.Submit) (positionSide string, reduceOnly bool, err error) {
	if s.Side != order.Long && s.Side != order.Short {
		return "", s.ReduceOnly, nil
	}
	mode, err := e.GetCurrentPositionMode(ctx)
	if err != nil {
		return "", false, err
	}
	if !mode.DualSidePosition {
		return "", s.ReduceOnly, nil
	}
	if (s.Side == order.Long) != s.ReduceOnly {
		return "LONG", false, nil
	}
	return "SHORT", false, nil
}

// SubmitOrder submits a new order. Stop and take profit orders trigger at TriggerPrice, and a bare Stop or TakeProfit
// is placed as the limit type when Price is set and the market type otherwise; USDⓈ-M places them, and
// trailing stops, as algo orders identified by their algo ID, a trailing stop with its percentage tracking value as
// the callback rate. Futures Buy and Sell orders trade the one-way position, while Long and Short orders follow the
// account's position mode: in Hedge Mode they trade the position of their direction, or with ReduceOnly the other
// one. An OCO order list is a limit maker order at Price and a stop loss order triggered at TriggerPrice, a stop limit
// order when RiskManagementModes.StopLoss has a limit price, and is identified by its order list ID
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	if s.Leverage != 0 && s.Leverage != 1 {
		return nil, fmt.Errorf("%w received '%v'", order.ErrSubmitLeverageNotSupported, s.Leverage)
	}
	if t := resolveTriggerOrderType(s.Type, s.Price); t != s.Type {
		// The response reports the type placed, which the order store and type filters match exactly
		resolved := *s
		resolved.Type = t
		s = &resolved
	}
	switch s.AssetType {
	case asset.Spot:
		if s.Type == order.OCO {
			return e.submitSpotOCOOrder(ctx, s)
		}
		return e.submitSpotOrder(ctx, s)
	case asset.Margin:
		if s.Type == order.OCO {
			return e.submitMarginOCOOrder(ctx, s)
		}
		return e.submitMarginOrder(ctx, s)
	case asset.USDTMarginedFutures:
		return e.submitUSDTMarginedOrder(ctx, s)
	case asset.CoinMarginedFutures:
		return e.submitCoinMarginedOrder(ctx, s)
	case asset.Options:
		return e.submitOptionsOrder(ctx, s)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, s.AssetType)
	}
}

// submitSpotOrder places a spot order through the WebSocket API when it can be used for orders, and REST otherwise.
// Both routes ask for the FULL reply, which reports the order's status and fills
func (e *Exchange) submitSpotOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	v, err := newSpotOrderValues(s)
	if err != nil {
		return nil, err
	}
	if e.canUseWebsocketAPIForOrders() {
		return e.submitSpotOrderWebsocket(ctx, s, v)
	}
	placed, err := e.NewOrder(ctx, &NewOrderRequest{
		Symbol:               s.Pair,
		Side:                 v.side,
		Type:                 v.orderType,
		TimeInForce:          v.timeInForce,
		Quantity:             v.quantity,
		QuoteOrderQuantity:   v.quoteOrderQuantity,
		Price:                v.price,
		NewClientOrderID:     s.ClientOrderID,
		StopPrice:            v.stopPrice,
		NewOrderResponseType: "FULL",
	})
	if err != nil {
		return nil, err
	}
	return e.spotSubmitResponse(s, placed)
}

// submitSpotOrderWebsocket places a spot order through the WebSocket API, sending what the REST route sends
func (e *Exchange) submitSpotOrderWebsocket(ctx context.Context, s *order.Submit, v *spotOrderValues) (*order.SubmitResponse, error) {
	placed, err := e.WsPlaceNewOrder(ctx, &WsPlaceOrderRequest{
		Symbol:               s.Pair,
		Side:                 v.side,
		Type:                 v.orderType,
		TimeInForce:          v.timeInForce,
		Price:                v.price,
		Quantity:             v.quantity,
		QuoteOrderQuantity:   v.quoteOrderQuantity,
		NewClientOrderID:     s.ClientOrderID,
		NewOrderResponseType: "FULL",
		StopPrice:            v.stopPrice,
	})
	if err != nil {
		return nil, err
	}
	fills := make([]OrderFill, len(placed.Fills))
	for i := range placed.Fills {
		fills[i] = OrderFill(placed.Fills[i])
	}
	return e.spotSubmitResponse(s, &NewOrderResponse{
		OrderID:                  placed.OrderID,
		ClientOrderID:            placed.ClientOrderID,
		TransactTime:             placed.TransactTime,
		OriginalQuantity:         placed.OriginalQuantity,
		ExecutedQuantity:         placed.ExecutedQuantity,
		CummulativeQuoteQuantity: placed.CummulativeQuoteQuantity,
		Status:                   placed.Status,
		Fills:                    fills,
	})
}

// submitMarginOrder places a margin order
func (e *Exchange) submitMarginOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	v, err := newSpotOrderValues(s)
	if err != nil {
		return nil, err
	}
	placed, err := e.PostMarginAccountOrder(ctx, &MarginAccountOrderRequest{
		Symbol:               s.Pair,
		IsIsolated:           s.MarginType == margin.Isolated,
		Side:                 v.side,
		OrderType:            v.orderType,
		Quantity:             v.quantity,
		QuoteOrderQuantity:   v.quoteOrderQuantity,
		Price:                v.price,
		StopPrice:            v.stopPrice,
		NewClientOrderID:     s.ClientOrderID,
		NewOrderResponseType: "FULL",
		SideEffectType:       marginSideEffect(s),
		TimeInForce:          v.timeInForce,
	})
	if err != nil {
		return nil, err
	}
	fills := make([]OrderFill, len(placed.Fills))
	for i := range placed.Fills {
		fills[i] = OrderFill(placed.Fills[i])
	}
	resp, err := e.spotSubmitResponse(s, &NewOrderResponse{
		OrderID:                  placed.OrderID,
		ClientOrderID:            placed.ClientOrderID,
		TransactTime:             placed.TransactTime,
		OriginalQuantity:         placed.OriginalQuantity,
		ExecutedQuantity:         placed.ExecutedQuantity,
		CummulativeQuoteQuantity: placed.CummulativeQuoteQuantity,
		Status:                   placed.Status,
		Fills:                    fills,
	})
	if err != nil {
		return nil, err
	}
	resp.BorrowSize = placed.MarginBuyBorrowAmount.Float64()
	return resp, nil
}

// spotSubmitResponse completes a spot or margin submit response with what Binance reports about the placed order
func (e *Exchange) spotSubmitResponse(s *order.Submit, placed *NewOrderResponse) (*order.SubmitResponse, error) {
	resp, err := s.DeriveSubmitResponse(strconv.FormatUint(placed.OrderID, 10))
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = placed.ClientOrderID
	resp.Date = placed.TransactTime.Time()
	resp.LastUpdated = resp.Date
	if resp.Status, err = stringToOrderStatus(placed.Status); err != nil {
		e.logOrderIssue(s.AssetType, resp.OrderID, err)
	}
	resp.RemainingAmount = remainingAmount(placed.OriginalQuantity, placed.ExecutedQuantity)
	if placed.ExecutedQuantity > 0 {
		resp.Cost = placed.CummulativeQuoteQuantity.Float64()
		resp.AverageExecutedPrice = averagePrice(placed.CummulativeQuoteQuantity, placed.ExecutedQuantity)
	}
	if len(placed.Fills) == 0 {
		return resp, nil
	}
	resp.Trades = make([]order.TradeHistory, len(placed.Fills))
	for i := range placed.Fills {
		resp.Trades[i] = order.TradeHistory{
			Price:     placed.Fills[i].Price.Float64(),
			Amount:    placed.Fills[i].Quantity.Float64(),
			Fee:       placed.Fills[i].Commission.Float64(),
			Exchange:  e.Name,
			TID:       strconv.FormatUint(placed.Fills[i].TradeID, 10),
			Type:      s.Type,
			Side:      s.Side,
			Timestamp: resp.Date,
			FeeAsset:  placed.Fills[i].CommissionAsset.String(),
		}
	}
	return resp, nil
}

// submitSpotOCOOrder places an OCO order list through the WebSocket API when it can be used for orders, and REST
// otherwise. A sell's limit maker order rests above the last price and its stop loss order below; a buy's the other
// way round
func (e *Exchange) submitSpotOCOOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	legs, err := newSpotOCOLegs(s)
	if err != nil {
		return nil, err
	}
	req := &OCOOrderListRequest{
		Symbol:            s.Pair,
		ListClientOrderID: s.ClientOrderID,
		Side:              legs.side,
		Quantity:          s.Amount,
		Above:             legs.limitMaker,
		Below:             legs.stopLoss,
	}
	if legs.side == order.Buy.String() {
		req.Above, req.Below = legs.stopLoss, legs.limitMaker
	}
	if e.canUseWebsocketAPIForOrders() {
		wsList, wsErr := e.WsPlaceOCOOrderList(ctx, req)
		if wsErr != nil {
			return nil, wsErr
		}
		return e.orderListSubmitResponse(s, wsList.OrderListID, wsList.ListClientOrderID, wsList.ListOrderStatus, wsList.TransactionTime.Time())
	}
	list, err := e.NewOCOOrderList(ctx, req)
	if err != nil {
		return nil, err
	}
	return e.orderListSubmitResponse(s, list.OrderListID, list.ListClientOrderID, list.ListOrderStatus, list.TransactionTime.Time())
}

// submitMarginOCOOrder places a margin OCO order list
func (e *Exchange) submitMarginOCOOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	legs, err := newSpotOCOLegs(s)
	if err != nil {
		return nil, err
	}
	list, err := e.NewMarginAccountOCOOrder(ctx, &MarginOCOOrderRequest{
		Symbol:               s.Pair,
		IsIsolated:           s.MarginType == margin.Isolated,
		ListClientOrderID:    s.ClientOrderID,
		Side:                 legs.side,
		Quantity:             s.Amount,
		LimitClientOrderID:   legs.limitMaker.ClientOrderID,
		Price:                legs.limitMaker.Price,
		StopClientOrderID:    legs.stopLoss.ClientOrderID,
		StopPrice:            legs.stopLoss.StopPrice,
		StopLimitPrice:       legs.stopLoss.Price,
		StopLimitTimeInForce: legs.stopLoss.TimeInForce,
		SideEffectType:       marginSideEffect(s),
	})
	if err != nil {
		return nil, err
	}
	resp, err := e.orderListSubmitResponse(s, list.OrderListID, list.ListClientOrderID, list.ListOrderStatus, list.TransactionTime.Time())
	if err != nil {
		return nil, err
	}
	resp.BorrowSize = list.MarginBuyBorrowAmount.Float64()
	return resp, nil
}

// orderListSubmitResponse completes the submit response of an OCO order list, which GCT tracks as one order identified
// by the list's ID
func (e *Exchange) orderListSubmitResponse(s *order.Submit, orderListID uint64, listClientOrderID, listOrderStatus string, transactionTime time.Time) (*order.SubmitResponse, error) {
	resp, err := s.DeriveSubmitResponse(strconv.FormatUint(orderListID, 10))
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = listClientOrderID
	resp.Date = transactionTime
	resp.LastUpdated = transactionTime
	if resp.Status, err = orderListStatus(listOrderStatus); err != nil {
		e.logOrderIssue(s.AssetType, resp.OrderID, err)
	}
	return resp, nil
}

// futuresOrderValues holds the values a USDⓈ-M or COIN-M order is sent with
type futuresOrderValues struct {
	side, positionSide, orderType, timeInForce string
	price                                      float64
	reduceOnly                                 bool
	goodTillDate                               time.Time
}

// newFuturesOrderValues works out the values of a USDⓈ-M or COIN-M order; only limit priced types take a price and a
// time in force
func (e *Exchange) newFuturesOrderValues(ctx context.Context, s *order.Submit) (*futuresOrderValues, error) {
	side, err := orderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	orderType, err := futuresOrderTypeString(resolveTriggerOrderType(s.Type, s.Price))
	if err != nil {
		return nil, err
	}
	if s.Amount <= 0 {
		return nil, fmt.Errorf("%w: futures orders take a base amount", order.ErrAmountIsInvalid)
	}
	v := &futuresOrderValues{side: side, orderType: orderType}
	switch orderType {
	case limitOrderType, "STOP", takeProfitOrderType:
		if s.Price <= 0 {
			return nil, order.ErrPriceMustBeSetIfLimitOrder
		}
		v.price = s.Price
		if v.timeInForce, err = timeInForceString(s.AssetType, s.TimeInForce); err != nil {
			return nil, err
		}
		if v.timeInForce == "GTD" {
			if s.EndTime.IsZero() {
				return nil, errGoodTillDateRequired
			}
			v.goodTillDate = s.EndTime
		}
	}
	if v.positionSide, v.reduceOnly, err = e.futuresPositionSide(ctx, s); err != nil {
		return nil, err
	}
	return v, nil
}

// submitUSDTMarginedOrder places a USDⓈ-M order. Stop, take profit and trailing stop orders go through the algo order
// endpoints, since New Order rejects them
func (e *Exchange) submitUSDTMarginedOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	v, err := e.newFuturesOrderValues(ctx, s)
	if err != nil {
		return nil, err
	}
	if isConditionalOrderType(s.Type) {
		return e.submitUSDTMarginedAlgoOrder(ctx, s, v)
	}
	placed, err := e.UFuturesNewOrder(ctx, &UFuturesNewOrderRequest{
		Symbol:           s.Pair,
		Side:             v.side,
		PositionSide:     v.positionSide,
		OrderType:        v.orderType,
		TimeInForce:      v.timeInForce,
		ReduceOnly:       v.reduceOnly,
		Quantity:         s.Amount,
		Price:            v.price,
		NewClientOrderID: s.ClientOrderID,
		NewOrderRespType: "RESULT",
		GoodTillDate:     v.goodTillDate,
	})
	if err != nil {
		return nil, err
	}
	return e.futuresSubmitResponse(s, placed.OrderID, placed.ClientOrderID, placed.Status, placed.UpdateTime.Time(), placed.OriginalQuantity, placed.ExecutedQuantity)
}

// submitUSDTMarginedAlgoOrder places a USDⓈ-M stop, take profit or trailing stop order as an algo order. A trailing
// stop's callback rate is its percentage tracking value, and its activation price, the last price when unset, is
// TriggerPrice
func (e *Exchange) submitUSDTMarginedAlgoOrder(ctx context.Context, s *order.Submit, v *futuresOrderValues) (*order.SubmitResponse, error) {
	req := &UNewAlgoOrderRequest{
		AlgoType:         "CONDITIONAL",
		Symbol:           s.Pair,
		Side:             v.side,
		PositionSide:     v.positionSide,
		OrderType:        v.orderType,
		TimeInForce:      v.timeInForce,
		Quantity:         s.Amount,
		Price:            v.price,
		ReduceOnly:       v.reduceOnly,
		ClientAlgoID:     s.ClientOrderID,
		NewOrderRespType: "RESULT",
		GoodTillDate:     v.goodTillDate,
	}
	if s.Type == order.TrailingStop {
		if s.TrackingMode != order.Percentage || s.TrackingValue <= 0 {
			return nil, errTrailingStopPercentageRequired
		}
		req.CallbackRate = s.TrackingValue
		req.ActivatePrice = s.TriggerPrice
	} else {
		if s.TriggerPrice <= 0 {
			return nil, errTriggerPriceRequired
		}
		req.TriggerPrice = s.TriggerPrice
	}
	var err error
	if req.WorkingType, err = workingTypeString(s.TriggerPriceType); err != nil {
		return nil, err
	}
	placed, err := e.UNewAlgoOrder(ctx, req)
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(strconv.FormatUint(placed.AlgoID, 10))
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = placed.ClientAlgoID
	resp.Date = placed.CreateTime.Time()
	resp.LastUpdated = placed.UpdateTime.Time()
	if resp.Status, err = algoOrderStatus(placed.AlgoStatus); err != nil {
		e.logOrderIssue(s.AssetType, resp.OrderID, err)
	}
	return resp, nil
}

// submitCoinMarginedOrder places a COIN-M limit or market order. Binance is moving COIN-M stop, take profit and
// trailing stop orders to algo order endpoints it has only announced, after which New Order rejects them
func (e *Exchange) submitCoinMarginedOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if s.Type != order.Limit && s.Type != order.Market {
		return nil, fmt.Errorf("%w: %s on %s", order.ErrUnsupportedOrderType, s.Type, s.AssetType)
	}
	v, err := e.newFuturesOrderValues(ctx, s)
	if err != nil {
		return nil, err
	}
	placed, err := e.FuturesNewOrder(ctx, &FuturesNewOrderRequest{
		Symbol:           s.Pair,
		Side:             v.side,
		PositionSide:     v.positionSide,
		OrderType:        v.orderType,
		TimeInForce:      v.timeInForce,
		Quantity:         s.Amount,
		Price:            v.price,
		ReduceOnly:       v.reduceOnly,
		NewClientOrderID: s.ClientOrderID,
		NewOrderRespType: "RESULT",
	})
	if err != nil {
		return nil, err
	}
	return e.futuresSubmitResponse(s, placed.OrderID, placed.ClientOrderID, placed.Status, placed.UpdateTime.Time(), placed.OriginalQuantity, placed.ExecutedQuantity)
}

// futuresSubmitResponse completes a USDⓈ-M or COIN-M submit response with what Binance reports about the placed
// order, whose update time is the only time the reply carries
func (e *Exchange) futuresSubmitResponse(s *order.Submit, orderID uint64, clientOrderID, status string, updateTime time.Time, original, executed types.Number) (*order.SubmitResponse, error) {
	resp, err := s.DeriveSubmitResponse(strconv.FormatUint(orderID, 10))
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = clientOrderID
	resp.Date = updateTime
	resp.LastUpdated = updateTime
	resp.RemainingAmount = remainingAmount(original, executed)
	if resp.Status, err = derivativesOrderStatus(status); err != nil {
		e.logOrderIssue(s.AssetType, resp.OrderID, err)
	}
	return resp, nil
}

// submitOptionsOrder places an options order, which Binance only takes as a limit order. A post only order is sent
// with the postOnly flag, and its time in force otherwise defaults to GTC
func (e *Exchange) submitOptionsOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if s.Type != order.Limit {
		return nil, fmt.Errorf("%w: %s on %s", order.ErrUnsupportedOrderType, s.Type, s.AssetType)
	}
	side, err := orderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	postOnly := s.TimeInForce.Is(order.PostOnly) || s.TimeInForce.Is(order.GoodTillCrossing)
	tif, err := timeInForceString(s.AssetType, s.TimeInForce&^(order.PostOnly|order.GoodTillCrossing))
	if err != nil {
		return nil, err
	}
	placed, err := e.NewOptionsOrder(ctx, &OptionsOrderRequest{
		Symbol:               s.Pair,
		Side:                 side,
		OrderType:            limitOrderType,
		Quantity:             s.Amount,
		Price:                s.Price,
		TimeInForce:          tif,
		ReduceOnly:           s.ReduceOnly,
		PostOnly:             postOnly,
		NewOrderResponseType: "RESULT",
		ClientOrderID:        s.ClientOrderID,
	})
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(strconv.FormatUint(placed.OrderID, 10))
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = placed.ClientOrderID
	resp.Date = placed.CreateTime.Time()
	resp.LastUpdated = placed.UpdateTime.Time()
	resp.RemainingAmount = remainingAmount(placed.Quantity, placed.ExecutedQuantity)
	resp.AverageExecutedPrice = placed.AveragePrice.Float64()
	resp.Fee = placed.Fee.Float64()
	resp.FeeAsset = placed.QuoteAsset
	if resp.Status, err = derivativesOrderStatus(placed.Status); err != nil {
		e.logOrderIssue(s.AssetType, resp.OrderID, err)
	}
	return resp, nil
}

// ModifyOrder modifies a USDⓈ-M or COIN-M limit order's price and amount, which Binance modifies in place on its
// futures alone; the order moves to the back of the matching queue
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	if action.AssetType != asset.USDTMarginedFutures && action.AssetType != asset.CoinMarginedFutures {
		return nil, fmt.Errorf("%w: order modification on %s", asset.ErrNotSupported, action.AssetType)
	}
	orderID, err := parseOrderID(action.OrderID)
	if err != nil {
		return nil, err
	}
	side, err := orderSideString(action.Side)
	if err != nil {
		return nil, err
	}
	var (
		modifiedID                        uint64
		clientOrderID, orderType, status  string
		price, originalQuantity, executed types.Number
		updated                           types.Time
	)
	if action.AssetType == asset.USDTMarginedFutures {
		modified, err := e.UModifyOrder(ctx, &UModifyOrderRequest{
			OrderID:           orderID,
			OrigClientOrderID: action.ClientOrderID,
			Symbol:            action.Pair,
			Side:              side,
			Quantity:          action.Amount,
			Price:             action.Price,
		})
		if err != nil {
			return nil, err
		}
		modifiedID, clientOrderID, orderType, status = modified.OrderID, modified.ClientOrderID, modified.Type, modified.Status
		price, originalQuantity, executed, updated = modified.Price, modified.OriginalQuantity, modified.ExecutedQuantity, modified.UpdateTime
	} else {
		modified, err := e.FuturesModifyOrder(ctx, &CFuturesModifyOrderRequest{
			OrderID:           orderID,
			OrigClientOrderID: action.ClientOrderID,
			Symbol:            action.Pair,
			Side:              side,
			Quantity:          action.Amount,
			Price:             action.Price,
		})
		if err != nil {
			return nil, err
		}
		modifiedID, clientOrderID, orderType, status = modified.OrderID, modified.ClientOrderID, modified.OrderType, modified.Status
		price, originalQuantity, executed, updated = modified.Price, modified.OriginalQuantity, modified.ExecutedQuantity, modified.UpdateTime
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.OrderID = strconv.FormatUint(modifiedID, 10)
	resp.ClientOrderID = clientOrderID
	resp.Price = price.Float64()
	resp.Amount = originalQuantity.Float64()
	resp.RemainingAmount = remainingAmount(originalQuantity, executed)
	resp.LastUpdated = updated.Time()
	if resp.Type, err = derivativesOrderType(orderType); err != nil {
		e.logOrderIssue(action.AssetType, resp.OrderID, err)
	}
	if resp.Status, err = derivativesOrderStatus(status); err != nil {
		e.logOrderIssue(action.AssetType, resp.OrderID, err)
	}
	return resp, nil
}

// CancelOrder cancels an order by its order ID or client order ID. OCO order lists are cancelled by their order list
// ID or list client order ID, and USDⓈ-M stop, take profit and trailing stop orders by their algo ID or client algo ID
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.PairAssetRequired()); err != nil {
		return err
	}
	if o.OrderID == "" && o.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	orderID, err := parseOrderID(o.OrderID)
	if err != nil {
		return err
	}
	switch o.AssetType {
	case asset.Spot:
		switch {
		case o.Type == order.OCO && e.canUseWebsocketAPIForOrders():
			_, err = e.WsCancelOCOOrder(ctx, &WsCancelOrderListRequest{Symbol: o.Pair, OrderListID: orderID, ListClientOrderID: o.ClientOrderID})
		case o.Type == order.OCO:
			_, err = e.CancelOCOOrder(ctx, &CancelOrderListRequest{Symbol: o.Pair, OrderListID: orderID, ListClientOrderID: o.ClientOrderID})
		case e.canUseWebsocketAPIForOrders():
			_, err = e.WsCancelOrder(ctx, &WsCancelOrderRequest{Symbol: o.Pair, OrderID: orderID, OriginalClientOrderID: o.ClientOrderID})
		default:
			_, err = e.CancelExistingOrder(ctx, &CancelOrderRequest{Symbol: o.Pair, OrderID: orderID, OriginalClientOrderID: o.ClientOrderID})
		}
	case asset.Margin:
		if o.Type == order.OCO {
			_, err = e.CancelMarginAccountOCOOrder(ctx, &MarginOCOCancelRequest{Symbol: o.Pair, IsIsolated: o.MarginType == margin.Isolated, OrderListID: orderID, ListClientOrderID: o.ClientOrderID})
		} else {
			_, err = e.CancelMarginAccountOrder(ctx, &MarginOrderCancelRequest{Symbol: o.Pair, IsIsolated: o.MarginType == margin.Isolated, OrderID: orderID, OriginalClientOrderID: o.ClientOrderID})
		}
	case asset.USDTMarginedFutures:
		if isConditionalOrderType(o.Type) {
			_, err = e.UCancelAlgoOrder(ctx, orderID, o.ClientOrderID)
		} else {
			_, err = e.UCancelOrder(ctx, o.Pair, orderID, o.ClientOrderID)
		}
	case asset.CoinMarginedFutures:
		_, err = e.FuturesCancelOrder(ctx, o.Pair, orderID, o.ClientOrderID)
	case asset.Options:
		_, err = e.CancelOptionsOrder(ctx, o.Pair, o.ClientOrderID, orderID)
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels USDⓈ-M, COIN-M or options orders of one pair. Orders are cancelled ten at a time by order
// ID or by client order ID, and USDⓈ-M algo orders one by one; each order's status is its cancelled status or why its
// cancellation failed, keyed by the ID it was requested with. Errors of whole requests are returned with the response
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	a, pair := o[0].AssetType, o[0].Pair
	switch a {
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options:
	default:
		return nil, fmt.Errorf("%w: batch cancellation on %s", asset.ErrNotSupported, a)
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var orderIDs []uint64
	var clientOrderIDs []string
	var algoOrders []*order.Cancel
	for i := range o {
		if o[i].AssetType != a || !o[i].Pair.Equal(pair) {
			return nil, errBatchCancelRequiresSamePair
		}
		if o[i].OrderID == "" && o[i].ClientOrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
		switch {
		case a == asset.USDTMarginedFutures && isConditionalOrderType(o[i].Type):
			algoOrders = append(algoOrders, &o[i])
		case o[i].OrderID != "":
			id, err := parseOrderID(o[i].OrderID)
			if err != nil {
				return nil, err
			}
			orderIDs = append(orderIDs, id)
		default:
			clientOrderIDs = append(clientOrderIDs, o[i].ClientOrderID)
		}
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string, len(o))}
	var errs error
	for ids := range slices.Chunk(orderIDs, batchCancelLimit) {
		keys := make([]string, len(ids))
		for i := range ids {
			keys[i] = strconv.FormatUint(ids[i], 10)
		}
		statuses, err := e.cancelOrderBatch(ctx, a, pair, ids, nil)
		errs = common.AppendError(errs, recordBatchStatuses(resp, keys, statuses, err))
	}
	for ids := range slices.Chunk(clientOrderIDs, batchCancelLimit) {
		statuses, err := e.cancelOrderBatch(ctx, a, pair, nil, ids)
		errs = common.AppendError(errs, recordBatchStatuses(resp, ids, statuses, err))
	}
	for _, algo := range algoOrders {
		statusKey := algo.OrderID
		if statusKey == "" {
			statusKey = algo.ClientOrderID
		}
		algoID, err := parseOrderID(algo.OrderID)
		if err == nil {
			_, err = e.UCancelAlgoOrder(ctx, algoID, algo.ClientOrderID)
		}
		errs = common.AppendError(errs, recordBatchStatuses(resp, []string{statusKey}, []string{"CANCELED"}, err))
	}
	return resp, errs
}

// cancelOrderBatch cancels up to ten orders of a symbol by order ID or by client order ID and returns each order's
// outcome in request order, which is how Binance replies: its status, or why Binance rejected its cancellation
func (e *Exchange) cancelOrderBatch(ctx context.Context, a asset.Item, pair currency.Pair, orderIDs []uint64, clientOrderIDs []string) ([]string, error) {
	switch a {
	case asset.USDTMarginedFutures:
		cancelled, err := e.UCancelBatchOrders(ctx, pair, orderIDs, clientOrderIDs)
		if err != nil {
			return nil, err
		}
		statuses := make([]string, len(cancelled))
		for i := range cancelled {
			statuses[i] = cancelled[i].Status
			if cancelled[i].Code != 0 {
				statuses[i] = cancelled[i].Message
			}
		}
		return statuses, nil
	case asset.CoinMarginedFutures:
		cancelled, err := e.FuturesBatchCancelOrders(ctx, pair, orderIDs, clientOrderIDs)
		if err != nil {
			return nil, err
		}
		statuses := make([]string, len(cancelled))
		for i := range cancelled {
			statuses[i] = cancelled[i].Status
			if cancelled[i].Code != 0 {
				statuses[i] = cancelled[i].Message
			}
		}
		return statuses, nil
	default:
		cancelled, err := e.CancelBatchOptionsOrders(ctx, pair, orderIDs, clientOrderIDs)
		if err != nil {
			return nil, err
		}
		statuses := make([]string, len(cancelled))
		for i := range cancelled {
			statuses[i] = cancelled[i].Status
		}
		return statuses, nil
	}
}

// recordBatchStatuses records the outcome of a cancellation request for each order it covered, and returns the
// request's error, if any, naming the orders it covered
func recordBatchStatuses(resp *order.CancelBatchResponse, keys, statuses []string, err error) error {
	if err != nil {
		for _, key := range keys {
			resp.Status[key] = err.Error()
		}
		return fmt.Errorf("error cancelling orders %v: %w", keys, err)
	}
	for i, key := range keys {
		if i < len(statuses) {
			resp.Status[key] = statuses[i]
		} else {
			resp.Status[key] = errBatchCancelResultMissing.Error()
		}
	}
	return nil
}

// CancelAllOrders cancels every open order of a pair, USDⓈ-M algo orders included, or of every symbol with open
// orders when the pair is empty. The spot and margin replies report each cancelled order's status, while the futures
// and options endpoints only confirm that a symbol's orders were cancelled; errors of individual symbols are returned
// together after every symbol has been tried
func (e *Exchange) CancelAllOrders(ctx context.Context, req *order.Cancel) (order.CancelAllResponse, error) {
	if err := req.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	resp := order.CancelAllResponse{Status: make(map[string]string)}
	var err error
	switch req.AssetType {
	case asset.Spot:
		err = e.cancelAllSpotOrders(ctx, req.Pair, &resp)
	case asset.Margin:
		err = e.cancelAllMarginOrders(ctx, req.Pair, req.MarginType == margin.Isolated, &resp)
	case asset.USDTMarginedFutures:
		err = e.cancelAllUSDTMarginedOrders(ctx, req.Pair)
	case asset.CoinMarginedFutures:
		err = e.cancelAllCoinMarginedOrders(ctx, req.Pair)
	case asset.Options:
		err = e.cancelAllOptionsOrders(ctx, req.Pair)
	default:
		return resp, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	return resp, err
}

// appendSymbol adds symbol to symbols unless it is there already
func appendSymbol(symbols []string, symbol string) []string {
	if slices.Contains(symbols, symbol) {
		return symbols
	}
	return append(symbols, symbol)
}

// symbolPairs returns the pairs of symbols Binance reports, to address them in requests: the matching available pair,
// or for a symbol without one a pair that formats back to the same symbol
func (e *Exchange) symbolPairs(symbols []string, a asset.Item) (currency.Pairs, error) {
	pairs := make(currency.Pairs, len(symbols))
	for i := range symbols {
		pair, err := e.orderPair(currency.EMPTYPAIR, symbols[i], a)
		if err != nil {
			if pair, err = currency.NewPairFromString(symbols[i]); err != nil {
				return nil, err
			}
		}
		pairs[i] = pair
	}
	return pairs, nil
}

// cancelAllSpotOrders cancels the open spot orders and order lists of pair, or of every symbol with open orders
func (e *Exchange) cancelAllSpotOrders(ctx context.Context, pair currency.Pair, resp *order.CancelAllResponse) error {
	pairs := currency.Pairs{pair}
	if pair.IsEmpty() {
		var open []TradeOrderResponse
		var err error
		if e.canUseWebsocketAPIForOrders() {
			open, err = e.WsCurrentOpenOrders(ctx, currency.EMPTYPAIR)
		} else {
			open, err = e.OpenOrders(ctx, currency.EMPTYPAIR)
		}
		if err != nil {
			return err
		}
		var symbols []string
		for i := range open {
			symbols = appendSymbol(symbols, open[i].Symbol)
		}
		if pairs, err = e.symbolPairs(symbols, asset.Spot); err != nil {
			return err
		}
	}
	var errs error
	for _, p := range pairs {
		if e.canUseWebsocketAPIForOrders() {
			cancelled, err := e.WsCancelOpenOrders(ctx, p)
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("error cancelling %s orders: %w", p, err))
				continue
			}
			for i := range cancelled {
				if len(cancelled[i].OrderReports) == 0 {
					resp.Add(strconv.FormatUint(cancelled[i].OrderID, 10), cancelled[i].Status)
				}
				for j := range cancelled[i].OrderReports {
					resp.Add(strconv.FormatUint(cancelled[i].OrderReports[j].OrderID, 10), cancelled[i].OrderReports[j].Status)
				}
			}
			continue
		}
		cancelled, err := e.CancelAllOpenOrderOnSymbol(ctx, p)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("error cancelling %s orders: %w", p, err))
			continue
		}
		for i := range cancelled {
			if len(cancelled[i].OrderReports) == 0 {
				resp.Add(strconv.FormatUint(cancelled[i].OrderID, 10), cancelled[i].Status)
			}
			for j := range cancelled[i].OrderReports {
				resp.Add(strconv.FormatUint(cancelled[i].OrderReports[j].OrderID, 10), cancelled[i].OrderReports[j].Status)
			}
		}
	}
	return errs
}

// cancelAllMarginOrders cancels the open margin orders and order lists of pair, or of every cross margin symbol with
// open orders; isolated margin requires the pair
func (e *Exchange) cancelAllMarginOrders(ctx context.Context, pair currency.Pair, isIsolated bool, resp *order.CancelAllResponse) error {
	pairs := currency.Pairs{pair}
	if pair.IsEmpty() {
		if isIsolated {
			return fmt.Errorf("%w: isolated margin orders are cancelled by pair", currency.ErrCurrencyPairEmpty)
		}
		open, err := e.GetMarginAccountsOpenOrders(ctx, currency.EMPTYPAIR, false)
		if err != nil {
			return err
		}
		var symbols []string
		for i := range open {
			symbols = appendSymbol(symbols, open[i].Symbol)
		}
		if pairs, err = e.symbolPairs(symbols, asset.Margin); err != nil {
			return err
		}
	}
	var errs error
	for _, p := range pairs {
		cancelled, err := e.CancelAllOpenMarginAccountOrdersOnSymbol(ctx, p, isIsolated)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("error cancelling %s orders: %w", p, err))
			continue
		}
		for i := range cancelled {
			if len(cancelled[i].OrderReports) == 0 {
				resp.Add(strconv.FormatUint(cancelled[i].OrderID, 10), cancelled[i].Status)
			}
			for j := range cancelled[i].OrderReports {
				resp.Add(strconv.FormatUint(cancelled[i].OrderReports[j].OrderID, 10), cancelled[i].OrderReports[j].Status)
			}
		}
	}
	return errs
}

// cancelAllUSDTMarginedOrders cancels the open orders and open algo orders of pair, or of every symbol with orders of
// either kind open
func (e *Exchange) cancelAllUSDTMarginedOrders(ctx context.Context, pair currency.Pair) error {
	orderPairs, algoPairs := currency.Pairs{pair}, currency.Pairs{pair}
	if pair.IsEmpty() {
		open, err := e.UAllAccountOpenOrders(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		openAlgo, err := e.UCurrentAllAlgoOpenOrders(ctx, "", currency.EMPTYPAIR, 0)
		if err != nil {
			return err
		}
		var symbols, algoSymbols []string
		for i := range open {
			symbols = appendSymbol(symbols, open[i].Symbol)
		}
		for i := range openAlgo {
			algoSymbols = appendSymbol(algoSymbols, openAlgo[i].Symbol)
		}
		if orderPairs, err = e.symbolPairs(symbols, asset.USDTMarginedFutures); err != nil {
			return err
		}
		if algoPairs, err = e.symbolPairs(algoSymbols, asset.USDTMarginedFutures); err != nil {
			return err
		}
	}
	var errs error
	for _, p := range orderPairs {
		if err := e.UCancelAllOpenOrders(ctx, p); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("error cancelling %s orders: %w", p, err))
		}
	}
	for _, p := range algoPairs {
		if err := e.UCancelAllAlgoOpenOrders(ctx, p); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("error cancelling %s algo orders: %w", p, err))
		}
	}
	return errs
}

// cancelAllCoinMarginedOrders cancels the open orders of pair, or of every symbol with open orders
func (e *Exchange) cancelAllCoinMarginedOrders(ctx context.Context, pair currency.Pair) error {
	pairs := currency.Pairs{pair}
	if pair.IsEmpty() {
		open, err := e.GetFuturesAllOpenOrders(ctx, currency.EMPTYPAIR, currency.EMPTYCODE)
		if err != nil {
			return err
		}
		var symbols []string
		for i := range open {
			symbols = appendSymbol(symbols, open[i].Symbol)
		}
		if pairs, err = e.symbolPairs(symbols, asset.CoinMarginedFutures); err != nil {
			return err
		}
	}
	var errs error
	for _, p := range pairs {
		if _, err := e.FuturesCancelAllOpenOrders(ctx, p); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("error cancelling %s orders: %w", p, err))
		}
	}
	return errs
}

// cancelAllOptionsOrders cancels the open orders of pair, or of every symbol with open orders
func (e *Exchange) cancelAllOptionsOrders(ctx context.Context, pair currency.Pair) error {
	pairs := currency.Pairs{pair}
	if pair.IsEmpty() {
		open, err := e.GetCurrentOpenOptionsOrders(ctx, &OptionsOpenOrdersRequest{})
		if err != nil {
			return err
		}
		var symbols []string
		for i := range open {
			symbols = appendSymbol(symbols, open[i].Symbol)
		}
		if pairs, err = e.symbolPairs(symbols, asset.Options); err != nil {
			return err
		}
	}
	var errs error
	for _, p := range pairs {
		if err := e.CancelAllOptionOrdersOnSpecificSymbol(ctx, p); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("error cancelling %s orders: %w", p, err))
		}
	}
	return errs
}

// GetOrderInfo returns an order's details. An ID no USDⓈ-M order has is looked up as an algo ID, and a margin order
// the cross margin account does not have in the isolated margin account, since the request carries neither
// distinction
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	id, err := parseOrderID(orderID)
	if err != nil {
		return nil, err
	}
	var d order.Detail
	switch a {
	case asset.Spot:
		var o *TradeOrderResponse
		if e.canUseWebsocketAPIForOrders() {
			o, err = e.WsQueryOrder(ctx, pair, id, "")
		} else {
			o, err = e.QueryOrder(ctx, pair, "", id)
		}
		if errors.Is(err, order.ErrOrderNotFound) {
			return e.spotOrderListInfo(ctx, id, pair, err)
		}
		if err != nil {
			return nil, err
		}
		d, err = e.spotOrderDetail(o, pair)
	case asset.Margin:
		o, crossErr := e.GetMarginAccountsOrder(ctx, pair, false, id, "")
		if errors.Is(crossErr, errAPIResponse) {
			var isolatedErr error
			if o, isolatedErr = e.GetMarginAccountsOrder(ctx, pair, true, id, ""); isolatedErr != nil {
				orderErr := common.AppendError(fmt.Errorf("error fetching cross margin order: %w", crossErr), fmt.Errorf("error fetching isolated margin order: %w", isolatedErr))
				if errors.Is(crossErr, order.ErrOrderNotFound) {
					return e.marginOrderListInfo(ctx, id, pair, orderErr)
				}
				return nil, orderErr
			}
		} else if crossErr != nil {
			return nil, crossErr
		}
		d, err = e.marginOrderDetail(o, pair)
	case asset.USDTMarginedFutures:
		o, orderErr := e.UGetOrderData(ctx, pair, id, "")
		switch {
		case orderErr == nil:
			d, err = e.uOrderDetail(o, pair)
		case errors.Is(orderErr, order.ErrOrderNotFound):
			// SubmitOrder identifies a conditional order by its algo ID, which Binance numbers apart from order IDs
			algo, algoErr := e.UQueryAlgoOrder(ctx, id, "")
			if algoErr != nil {
				return nil, common.AppendError(fmt.Errorf("error fetching order: %w", orderErr), fmt.Errorf("error fetching algo order: %w", algoErr))
			}
			var symbol string
			if symbol, err = e.FormatSymbol(pair, a); err != nil {
				return nil, err
			}
			if algo.Symbol != symbol {
				return nil, orderErr
			}
			d, err = e.uAlgoOrderDetail(&algo.UAlgoOrderBase, algo.ActualPrice.Float64(), pair)
			// The query alone reports the quantity the triggered order filled
			d.ExecutedAmount = algo.ActualQuantity.Float64()
		default:
			return nil, orderErr
		}
	case asset.CoinMarginedFutures:
		var o *FuturesOrderDetailResponse
		if o, err = e.FuturesGetOrderData(ctx, pair, id, ""); err != nil {
			return nil, err
		}
		d, err = e.cOrderDetail(o, pair)
	case asset.Options:
		var o *OptionsOrderStatusResponse
		if o, err = e.GetSingleEOptionsOrder(ctx, pair, "", id); err != nil {
			return nil, err
		}
		d, err = e.optionsOrderDetail(&o.OptionsOrder, o.CreateTime.Time(), o.PostOnly, pair)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if err != nil {
		return nil, fmt.Errorf("error converting %s order %s: %w", a, orderID, err)
	}
	return &d, nil
}

// spotOrderListInfo returns pair's spot order list orderListID, as SubmitOrder identifies an OCO order by its order list
// ID. orderErr is the error looking the ID up as an order gave, which stands when pair has no such order list
func (e *Exchange) spotOrderListInfo(ctx context.Context, orderListID uint64, pair currency.Pair, orderErr error) (*order.Detail, error) {
	var list *OCOOrderResponse
	var err error
	if e.canUseWebsocketAPIForOrders() {
		list, err = e.WsQueryOCOOrder(ctx, orderListID, "")
	} else {
		list, err = e.GetOCOOrders(ctx, orderListID, "")
	}
	if err != nil {
		return nil, common.AppendError(orderErr, fmt.Errorf("error fetching order list: %w", err))
	}
	return e.orderListInfo(asset.Spot, list.Symbol, list.OrderListID, list.ListClientOrderID, list.ListOrderStatus, list.TransactionTime.Time(), pair, orderErr)
}

// marginOrderListInfo returns pair's cross or isolated margin order list orderListID, as SubmitOrder identifies an OCO
// order by its order list ID. orderErr is the error looking the ID up as an order gave, which stands when pair has no
// such order list
func (e *Exchange) marginOrderListInfo(ctx context.Context, orderListID uint64, pair currency.Pair, orderErr error) (*order.Detail, error) {
	// Cross margin order list queries take no symbol, isolated ones require it
	list, crossErr := e.GetMarginAccountOCOOrder(ctx, currency.EMPTYPAIR, false, orderListID, "")
	if crossErr != nil {
		var isolatedErr error
		if list, isolatedErr = e.GetMarginAccountOCOOrder(ctx, pair, true, orderListID, ""); isolatedErr != nil {
			listErr := common.AppendError(fmt.Errorf("error fetching cross margin order list: %w", crossErr), fmt.Errorf("error fetching isolated margin order list: %w", isolatedErr))
			return nil, common.AppendError(orderErr, listErr)
		}
	}
	d, err := e.orderListInfo(asset.Margin, list.Symbol, list.OrderListID, list.ListClientOrderID, list.ListOrderStatus, list.TransactionTime.Time(), pair, orderErr)
	if d != nil {
		// Cancellations need the account the list belongs to
		d.MarginType = marginAccountType(list.IsIsolated)
	}
	return d, err
}

// orderListInfo converts the order list GetOrderInfo found when it looked an order ID up as an order list ID, unless the
// list belongs to another symbol, when orderErr, the error looking the ID up as an order gave, stands
func (e *Exchange) orderListInfo(a asset.Item, symbol string, orderListID uint64, listClientOrderID, listOrderStatus string, transactionTime time.Time, pair currency.Pair, orderErr error) (*order.Detail, error) {
	formatted, err := e.FormatSymbol(pair, a)
	if err != nil {
		return nil, err
	}
	if symbol != formatted {
		return nil, orderErr
	}
	d, err := e.orderListDetail(a, orderListID, listClientOrderID, listOrderStatus, transactionTime, pair)
	if err != nil {
		return nil, fmt.Errorf("error converting %s order list %d: %w", a, orderListID, err)
	}
	return &d, nil
}

// GetDepositAddress returns a deposit address for a specified currency, on its default network when chain is empty
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	addr, err := e.GetDepositAddressForCurrency(ctx, cryptocurrency, chain, 0)
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: addr.Address,
		Tag:     addr.Tag,
		Chain:   chain,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	// The address book name is not sent, as Binance saves the address under it
	resp, err := e.WithdrawCrypto(ctx, &WithdrawRequest{
		Coin:            withdrawRequest.Currency,
		WithdrawOrderID: withdrawRequest.ClientOrderID,
		Network:         withdrawRequest.Crypto.Chain,
		Address:         withdrawRequest.Crypto.Address,
		AddressTag:      withdrawRequest.Crypto.AddressTag,
		Amount:          withdrawRequest.Amount,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: resp.ID}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// orderPair returns an order's pair: the pair its orders were requested for, or the available pair matching its symbol
// when every symbol's orders were requested
func (e *Exchange) orderPair(requested currency.Pair, symbol string, a asset.Item) (currency.Pair, error) {
	if !requested.IsEmpty() {
		return requested, nil
	}
	return e.MatchSymbolWithAvailablePairs(symbol, a, a != asset.Spot && a != asset.Margin)
}

// appendListedOrder adds an order of a listing to orders. An order whose symbol is not an available pair is left out,
// and an order with values that cannot be converted is kept without them, each with a logged warning, so one odd order
// does not fail the listing
func (e *Exchange) appendListedOrder(orders []order.Detail, a asset.Item, orderID uint64, requested currency.Pair, symbol string, convert func(currency.Pair) (order.Detail, error)) []order.Detail {
	id := strconv.FormatUint(orderID, 10)
	pair, err := e.orderPair(requested, symbol, a)
	if err != nil {
		e.logOrderIssue(a, id, err)
		return orders
	}
	d, err := convert(pair)
	if err != nil {
		e.logOrderIssue(a, id, err)
	}
	return append(orders, d)
}

// marginAccountType returns the margin type of a margin account order
func marginAccountType(isIsolated bool) margin.Type {
	if isIsolated {
		return margin.Isolated
	}
	return margin.Multi
}

// setFilledCost sets a spot or margin order's cost and average price from the quote amount its fills total, which
// Binance reports as negative for historical orders whose total is not available
func setFilledCost(d *order.Detail, cumulativeQuote, executed types.Number) {
	if executed <= 0 || cumulativeQuote < 0 {
		return
	}
	d.Cost = cumulativeQuote.Float64()
	d.CostAsset = d.Pair.Quote
	d.AverageExecutedPrice = averagePrice(cumulativeQuote, executed)
}

// spotOrderDetail converts a spot order as the order queries return it. Values that cannot be converted are left
// unset and returned as the error
func (e *Exchange) spotOrderDetail(o *TradeOrderResponse, pair currency.Pair) (order.Detail, error) {
	vars, err := compatibleOrderVars(asset.Spot, o.Side, o.Status, o.Type, "")
	d := order.Detail{
		TimeInForce:    o.TimeInForce,
		Price:          o.Price.Float64(),
		Amount:         o.OriginalQuantity.Float64(),
		TriggerPrice:   o.StopPrice.Float64(),
		QuoteAmount:    o.OriginalQuoteOrderQuantity.Float64(),
		ExecutedAmount: o.ExecutedQuantity.Float64(),
		Exchange:       e.Name,
		OrderID:        strconv.FormatUint(o.OrderID, 10),
		ClientOrderID:  o.ClientOrderID,
		Type:           vars.OrderType,
		Side:           vars.Side,
		Status:         vars.Status,
		AssetType:      asset.Spot,
		Date:           o.Time.Time(),
		LastUpdated:    o.UpdateTime.Time(),
		Pair:           pair,
	}
	d.RemainingAmount = remainingAmount(o.OriginalQuantity, o.ExecutedQuantity)
	setFilledCost(&d, o.CummulativeQuoteQuantity, o.ExecutedQuantity)
	return d, err
}

// marginOrderDetail converts a margin order. Values that cannot be converted are left unset and returned as the error
func (e *Exchange) marginOrderDetail(o *MarginTradeOrderResponse, pair currency.Pair) (order.Detail, error) {
	vars, err := compatibleOrderVars(asset.Margin, o.Side, o.Status, o.Type, o.TimeInForce)
	d := order.Detail{
		TimeInForce:    vars.TimeInForce,
		Price:          o.Price.Float64(),
		Amount:         o.OriginalQuantity.Float64(),
		TriggerPrice:   o.StopPrice.Float64(),
		ExecutedAmount: o.ExecutedQuantity.Float64(),
		Exchange:       e.Name,
		OrderID:        strconv.FormatUint(o.OrderID, 10),
		ClientOrderID:  o.ClientOrderID,
		Type:           vars.OrderType,
		Side:           vars.Side,
		Status:         vars.Status,
		AssetType:      asset.Margin,
		Date:           o.Time.Time(),
		LastUpdated:    o.UpdateTime.Time(),
		Pair:           pair,
		MarginType:     marginAccountType(o.IsIsolated),
	}
	d.RemainingAmount = remainingAmount(o.OriginalQuantity, o.ExecutedQuantity)
	setFilledCost(&d, o.CummulativeQuoteQuantity, o.ExecutedQuantity)
	return d, err
}

// orderListDetail converts an OCO order list, which GCT tracks as one order identified by the list's ID; the list
// queries report neither the side nor the amounts of its orders. Values that cannot be converted are left unset and
// returned as the error
func (e *Exchange) orderListDetail(a asset.Item, orderListID uint64, listClientOrderID, listOrderStatus string, transactionTime time.Time, pair currency.Pair) (order.Detail, error) {
	status, err := orderListStatus(listOrderStatus)
	return order.Detail{
		Exchange:      e.Name,
		OrderID:       strconv.FormatUint(orderListID, 10),
		ClientOrderID: listClientOrderID,
		Type:          order.OCO,
		Status:        status,
		AssetType:     a,
		Date:          transactionTime,
		LastUpdated:   transactionTime,
		Pair:          pair,
	}, err
}

// uOrderDetail converts a USDⓈ-M order, whose cost is in its margin asset. Values that cannot be converted are left
// unset and returned as the error
func (e *Exchange) uOrderDetail(o *UOrderResponse, pair currency.Pair) (order.Detail, error) {
	vars, err := compatibleOrderVars(asset.USDTMarginedFutures, o.Side, o.Status, o.Type, o.TimeInForce)
	d := order.Detail{
		TimeInForce:          vars.TimeInForce,
		ReduceOnly:           o.ReduceOnly,
		Price:                o.Price.Float64(),
		Amount:               o.OriginalQuantity.Float64(),
		TriggerPrice:         o.StopPrice.Float64(),
		AverageExecutedPrice: o.AveragePrice.Float64(),
		ExecutedAmount:       o.ExecutedQuantity.Float64(),
		Cost:                 o.CumulativeQuote.Float64(),
		Exchange:             e.Name,
		OrderID:              strconv.FormatUint(o.OrderID, 10),
		ClientOrderID:        o.ClientOrderID,
		Type:                 vars.OrderType,
		Side:                 vars.Side,
		Status:               vars.Status,
		AssetType:            asset.USDTMarginedFutures,
		Date:                 o.Time.Time(),
		LastUpdated:          o.UpdateTime.Time(),
		Pair:                 pair,
	}
	if vars.OrderType == order.TrailingStop || o.OriginalType == trailingStopOrderType {
		// Binance documents the stop price as meaningless for trailing stops, which trigger at their activation price,
		// and a triggered one reports MARKET as its type
		d.TriggerPrice = o.ActivatePrice.Float64()
	}
	d.RemainingAmount = remainingAmount(o.OriginalQuantity, o.ExecutedQuantity)
	return d, err
}

// uAlgoOrderDetail converts a USDⓈ-M algo order, identified by its algo ID; actualPrice is the average price of the
// order it triggered. Values that cannot be converted are left unset and returned as the error
func (e *Exchange) uAlgoOrderDetail(o *UAlgoOrderBase, actualPrice float64, pair currency.Pair) (order.Detail, error) {
	vars, err := compatibleOrderVars(asset.USDTMarginedFutures, o.Side, "", o.OrderType, o.TimeInForce)
	status, statusErr := algoOrderStatus(o.AlgoStatus)
	d := order.Detail{
		TimeInForce:          vars.TimeInForce,
		ReduceOnly:           o.ReduceOnly,
		Price:                o.Price.Float64(),
		Amount:               o.Quantity.Float64(),
		TriggerPrice:         o.TriggerPrice.Float64(),
		AverageExecutedPrice: actualPrice,
		Exchange:             e.Name,
		OrderID:              strconv.FormatUint(o.AlgoID, 10),
		ClientOrderID:        o.ClientAlgoID,
		Type:                 vars.OrderType,
		Side:                 vars.Side,
		Status:               status,
		AssetType:            asset.USDTMarginedFutures,
		Date:                 o.CreateTime.Time(),
		LastUpdated:          o.UpdateTime.Time(),
		Pair:                 pair,
	}
	return d, common.AppendError(err, statusErr)
}

// cOrderDetail converts a COIN-M order, whose amounts are contracts and whose cost is the base asset its fills total.
// Values that cannot be converted are left unset and returned as the error
func (e *Exchange) cOrderDetail(o *FuturesOrderDetailResponse, pair currency.Pair) (order.Detail, error) {
	vars, err := compatibleOrderVars(asset.CoinMarginedFutures, o.Side, o.Status, o.OrderType, o.TimeInForce)
	d := order.Detail{
		TimeInForce:          vars.TimeInForce,
		ReduceOnly:           o.ReduceOnly,
		Price:                o.Price.Float64(),
		Amount:               o.OriginalQuantity.Float64(),
		TriggerPrice:         o.StopPrice.Float64(),
		AverageExecutedPrice: o.AveragePrice.Float64(),
		ExecutedAmount:       o.ExecutedQuantity.Float64(),
		Cost:                 o.CumulativeBase.Float64(),
		Exchange:             e.Name,
		OrderID:              strconv.FormatUint(o.OrderID, 10),
		ClientOrderID:        o.ClientOrderID,
		Type:                 vars.OrderType,
		Side:                 vars.Side,
		Status:               vars.Status,
		AssetType:            asset.CoinMarginedFutures,
		Date:                 o.Time.Time(),
		LastUpdated:          o.UpdateTime.Time(),
		Pair:                 pair,
	}
	if vars.OrderType == order.TrailingStop || o.OriginalType == trailingStopOrderType {
		// Binance documents the stop price as meaningless for trailing stops, which trigger at their activation price,
		// and a triggered one reports MARKET as its type
		d.TriggerPrice = o.ActivatePrice.Float64()
	}
	d.RemainingAmount = remainingAmount(o.OriginalQuantity, o.ExecutedQuantity)
	return d, err
}

// optionsOrderDetail converts an options order; the queries that report postOnly flag post only orders. Values that
// cannot be converted are left unset and returned as the error
func (e *Exchange) optionsOrderDetail(o *OptionsOrder, createTime time.Time, postOnly bool, pair currency.Pair) (order.Detail, error) {
	vars, err := compatibleOrderVars(asset.Options, o.Side, o.Status, o.Type, o.TimeInForce)
	d := order.Detail{
		TimeInForce:          vars.TimeInForce,
		ReduceOnly:           o.ReduceOnly,
		Price:                o.Price.Float64(),
		Amount:               o.Quantity.Float64(),
		AverageExecutedPrice: o.AveragePrice.Float64(),
		ExecutedAmount:       o.ExecutedQuantity.Float64(),
		Cost:                 o.AveragePrice.Decimal().Mul(o.ExecutedQuantity.Decimal()).InexactFloat64(),
		CostAsset:            o.QuoteAsset,
		Exchange:             e.Name,
		OrderID:              strconv.FormatUint(o.OrderID, 10),
		ClientOrderID:        o.ClientOrderID,
		Type:                 vars.OrderType,
		Side:                 vars.Side,
		Status:               vars.Status,
		AssetType:            asset.Options,
		Date:                 createTime,
		LastUpdated:          o.UpdateTime.Time(),
		Pair:                 pair,
	}
	if postOnly {
		d.TimeInForce |= order.PostOnly
	}
	d.RemainingAmount = remainingAmount(o.Quantity, o.ExecutedQuantity)
	return d, err
}

// openOrdersPairs returns the pairs to request open orders for: each requested pair, or every symbol in one request
// when no pair is requested or when requesting each pair would weigh more. Spot weighs 6 a symbol and 80 for every
// symbol, USDⓈ-M, COIN-M and options 1 and 40, while margin weighs every symbol as one request per trading symbol
func openOrdersPairs(a asset.Item, pairs currency.Pairs) currency.Pairs {
	allSymbolsFrom := 40
	switch a {
	case asset.Spot:
		allSymbolsFrom = 14
	case asset.Margin:
		allSymbolsFrom = math.MaxInt
	}
	if len(pairs) == 0 || len(pairs) >= allSymbolsFrom {
		return currency.Pairs{currency.EMPTYPAIR}
	}
	return pairs
}

// GetFee returns an estimate of fee based on type of transaction. Trade fees are spot fees at the account's commission
// rates, and withdrawal fees those of the coin's default network
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		rate, err := e.tradeFeeRate(ctx, feeBuilder.IsMaker)
		if err != nil {
			return 0, err
		}
		fee = rate * feeBuilder.PurchasePrice * feeBuilder.Amount
	case exchange.CryptocurrencyWithdrawalFee:
		var err error
		if fee, err = e.cryptocurrencyWithdrawalFee(ctx, feeBuilder.Pair.Base); err != nil {
			return 0, err
		}
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	// Binance charges no deposit fees and GCT does not move fiat through Binance, so the other fee types cost nothing
	return max(fee, 0), nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.002 * price * amount
}

// tradeFeeRate returns the account's spot maker or taker commission rate. The account's makerCommission and
// takerCommission are in basis points; its commission rates are fractions
func (e *Exchange) tradeFeeRate(ctx context.Context, isMaker bool) (float64, error) {
	account, err := e.GetAccount(ctx, true)
	if err != nil {
		return 0, err
	}
	if isMaker {
		return account.CommissionRates.Maker.Float64(), nil
	}
	return account.CommissionRates.Taker.Float64(), nil
}

// cryptocurrencyWithdrawalFee returns the fee for withdrawing a coin on its default network
func (e *Exchange) cryptocurrencyWithdrawalFee(ctx context.Context, c currency.Code) (float64, error) {
	if c.IsEmpty() {
		return 0, currency.ErrCurrencyCodeEmpty
	}
	coins, err := e.GetAllCoinsInfo(ctx)
	if err != nil {
		return 0, err
	}
	for i := range coins {
		if !coins[i].Coin.Equal(c) {
			continue
		}
		for j := range coins[i].NetworkList {
			if coins[i].NetworkList[j].IsDefault {
				return coins[i].NetworkList[j].WithdrawFee.Float64(), nil
			}
		}
		return 0, fmt.Errorf("%w for %s", errDefaultNetworkNotFound, c)
	}
	return 0, fmt.Errorf("%w: %s", currency.ErrCurrencyNotFound, c)
}

// GetActiveOrders returns the open orders of the requested pairs, or of every symbol when none are requested. OCO
// order lists are listed as orders identified by their order list ID when the OCO type is requested, while their
// orders are listed among the open orders otherwise. USDⓈ-M algo orders are listed unless only limit or market orders
// are requested, and a margin listing is of the isolated account, which needs pairs, when the isolated margin type is
// requested
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	var err error
	switch req.AssetType {
	case asset.Spot:
		orders, err = e.spotActiveOrders(ctx, req)
	case asset.Margin:
		orders, err = e.marginActiveOrders(ctx, req)
	case asset.USDTMarginedFutures:
		orders, err = e.uActiveOrders(ctx, req)
	case asset.CoinMarginedFutures:
		orders, err = e.cActiveOrders(ctx, req)
	case asset.Options:
		orders, err = e.optionsActiveOrders(ctx, req)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	if err != nil {
		return nil, err
	}
	return e.filterOrders(req, orders), nil
}

// filterOrders applies req's filters to orders. A bare Stop or TakeProfit type also covers the limit and market forms
// Binance's stop and take profit orders read back as, since SubmitOrder places a bare type as one of them
func (e *Exchange) filterOrders(req *order.MultiOrderRequest, orders []order.Detail) order.FilteredOrders {
	if req.Type != order.Stop && req.Type != order.TakeProfit {
		return req.Filter(e.Name, orders)
	}
	orders = slices.DeleteFunc(slices.Clone(orders), func(d order.Detail) bool { return d.Type&req.Type != req.Type })
	anyType := *req
	anyType.Type = order.AnyType
	return anyType.Filter(e.Name, orders)
}

// spotActiveOrders returns open spot orders, or open order lists for the OCO type, through the WebSocket API when it
// can be used for orders and REST otherwise
func (e *Exchange) spotActiveOrders(ctx context.Context, req *order.MultiOrderRequest) ([]order.Detail, error) {
	useWebsocket := e.canUseWebsocketAPIForOrders()
	if req.Type == order.OCO {
		var lists []OCOOrderResponse
		var err error
		if useWebsocket {
			lists, err = e.WsCurrentOpenOCOOrders(ctx)
		} else {
			lists, err = e.GetOpenOCOList(ctx)
		}
		if err != nil {
			return nil, err
		}
		orders := make([]order.Detail, 0, len(lists))
		for i := range lists {
			l := &lists[i]
			orders = e.appendListedOrder(orders, asset.Spot, l.OrderListID, currency.EMPTYPAIR, l.Symbol, func(pair currency.Pair) (order.Detail, error) {
				return e.orderListDetail(asset.Spot, l.OrderListID, l.ListClientOrderID, l.ListOrderStatus, l.TransactionTime.Time(), pair)
			})
		}
		return orders, nil
	}
	var orders []order.Detail
	for _, p := range openOrdersPairs(asset.Spot, req.Pairs) {
		var open []TradeOrderResponse
		var err error
		if useWebsocket {
			open, err = e.WsCurrentOpenOrders(ctx, p)
		} else {
			open, err = e.OpenOrders(ctx, p)
		}
		if err != nil {
			return nil, err
		}
		for i := range open {
			o := &open[i]
			orders = e.appendListedOrder(orders, asset.Spot, o.OrderID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
				return e.spotOrderDetail(o, pair)
			})
		}
	}
	return orders, nil
}

// marginOrderListPairs returns the pairs to request margin order lists for: each requested pair of the isolated
// margin account, which Binance requires, or every cross margin symbol in one request, since the cross margin
// account's order list queries take no symbol
func marginOrderListPairs(pairs currency.Pairs, isIsolated bool) (currency.Pairs, error) {
	if !isIsolated {
		return currency.Pairs{currency.EMPTYPAIR}, nil
	}
	if len(pairs) == 0 {
		return nil, fmt.Errorf("%w: isolated margin order lists are listed by pair", currency.ErrCurrencyPairsEmpty)
	}
	return pairs, nil
}

// appendMarginOrderLists adds margin order lists to orders as orders identified by their order list ID
func (e *Exchange) appendMarginOrderLists(orders []order.Detail, requested currency.Pair, lists []MarginOrderListResponse) []order.Detail {
	for i := range lists {
		l := &lists[i]
		orders = e.appendListedOrder(orders, asset.Margin, l.OrderListID, requested, l.Symbol, func(pair currency.Pair) (order.Detail, error) {
			d, err := e.orderListDetail(asset.Margin, l.OrderListID, l.ListClientOrderID, l.ListOrderStatus, l.TransactionTime.Time(), pair)
			d.MarginType = marginAccountType(l.IsIsolated)
			return d, err
		})
	}
	return orders
}

// marginActiveOrders returns open margin orders, or open order lists for the OCO type
func (e *Exchange) marginActiveOrders(ctx context.Context, req *order.MultiOrderRequest) ([]order.Detail, error) {
	isIsolated := req.MarginType == margin.Isolated
	var orders []order.Detail
	if req.Type == order.OCO {
		pairs, err := marginOrderListPairs(req.Pairs, isIsolated)
		if err != nil {
			return nil, err
		}
		for _, p := range pairs {
			var lists []MarginOrderListResponse
			if lists, err = e.GetMarginAccountsOpenOCOOrder(ctx, p, isIsolated); err != nil {
				return nil, err
			}
			orders = e.appendMarginOrderLists(orders, p, lists)
		}
		return orders, nil
	}
	for _, p := range openOrdersPairs(asset.Margin, req.Pairs) {
		open, err := e.GetMarginAccountsOpenOrders(ctx, p, isIsolated)
		if err != nil {
			return nil, err
		}
		for i := range open {
			o := &open[i]
			orders = e.appendListedOrder(orders, asset.Margin, o.OrderID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
				return e.marginOrderDetail(o, pair)
			})
		}
	}
	return orders, nil
}

// uActiveOrders returns open USDⓈ-M orders and algo orders
func (e *Exchange) uActiveOrders(ctx context.Context, req *order.MultiOrderRequest) ([]order.Detail, error) {
	var orders []order.Detail
	for _, p := range openOrdersPairs(asset.USDTMarginedFutures, req.Pairs) {
		if !isConditionalOrderType(req.Type) {
			open, err := e.UAllAccountOpenOrders(ctx, p)
			if err != nil {
				return nil, err
			}
			for i := range open {
				o := &open[i]
				orders = e.appendListedOrder(orders, asset.USDTMarginedFutures, o.OrderID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
					return e.uOrderDetail(o, pair)
				})
			}
		}
		if req.Type == order.AnyType || isConditionalOrderType(req.Type) {
			openAlgo, err := e.UCurrentAllAlgoOpenOrders(ctx, "", p, 0)
			if err != nil {
				return nil, err
			}
			for i := range openAlgo {
				o := &openAlgo[i]
				orders = e.appendListedOrder(orders, asset.USDTMarginedFutures, o.AlgoID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
					return e.uAlgoOrderDetail(&o.UAlgoOrderBase, o.ActualPrice.Float64(), pair)
				})
			}
		}
	}
	return orders, nil
}

// cActiveOrders returns open COIN-M orders
func (e *Exchange) cActiveOrders(ctx context.Context, req *order.MultiOrderRequest) ([]order.Detail, error) {
	var orders []order.Detail
	for _, p := range openOrdersPairs(asset.CoinMarginedFutures, req.Pairs) {
		open, err := e.GetFuturesAllOpenOrders(ctx, p, currency.EMPTYCODE)
		if err != nil {
			return nil, err
		}
		for i := range open {
			o := &open[i]
			orders = e.appendListedOrder(orders, asset.CoinMarginedFutures, o.OrderID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
				return e.cOrderDetail(o, pair)
			})
		}
	}
	return orders, nil
}

// optionsActiveOrders returns open options orders created within the request's time window, when it has one
func (e *Exchange) optionsActiveOrders(ctx context.Context, req *order.MultiOrderRequest) ([]order.Detail, error) {
	var orders []order.Detail
	for _, p := range openOrdersPairs(asset.Options, req.Pairs) {
		open, err := e.GetCurrentOpenOptionsOrders(ctx, &OptionsOpenOrdersRequest{Symbol: p, StartTime: req.StartTime, EndTime: req.EndTime})
		if err != nil {
			return nil, err
		}
		for i := range open {
			o := &open[i]
			orders = e.appendListedOrder(orders, asset.Options, o.OrderID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
				return e.optionsOrderDetail(&o.OptionsOrder, o.CreateTime.Time(), false, pair)
			})
		}
	}
	return orders, nil
}

// GetOrderHistory returns the requested pairs' orders of every status, one page of the most Binance returns per pair:
// orders from FromOrderID on, or within the request's time window, which Binance limits per asset (24 hours on spot
// and margin, 7 days on USDⓈ-M and COIN-M) and defaults when it is unset. OCO order lists are listed when the OCO
// type is requested, in one page for every symbol on spot and cross margin, whose order list queries take no symbol,
// and USDⓈ-M algo orders unless only limit or market orders are requested
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	fromID, err := parseOrderID(req.FromOrderID)
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		orders, err = e.spotOrderHistory(ctx, req, fromID)
	case asset.Margin:
		orders, err = e.marginOrderHistory(ctx, req, fromID)
	case asset.USDTMarginedFutures:
		orders, err = e.uOrderHistory(ctx, req, fromID)
	case asset.CoinMarginedFutures:
		orders, err = e.cOrderHistory(ctx, req, fromID)
	case asset.Options:
		orders, err = e.optionsOrderHistory(ctx, req, fromID)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	if err != nil {
		return nil, err
	}
	return e.filterOrders(req, orders), nil
}

// Order history spans Binance accepts: spot order and order list histories take at most 24 hours, margin ones less than
// 24 hours, and USDⓈ-M and COIN-M order histories less than 7 days
const (
	spotOrderHistorySpan    = 24 * time.Hour
	marginOrderHistorySpan  = 24*time.Hour - time.Millisecond
	futuresOrderHistorySpan = 7*24*time.Hour - time.Millisecond
)

// pagedOrderHistory fetches the orders between start and end in spans of at most maxSpan, or in one span when maxSpan
// is zero, paging forward by time within a span while full pages of limit orders come back. Each span and page starts
// at the time the previous one ended, inclusive, so orders sharing that millisecond are kept, and orders already
// fetched are dropped by ID; only orders beyond a full page within one millisecond, or within a span's last two, cannot
// be reached. Without a time window one request returns Binance's default, its most recent orders, and so does a
// request from an order ID, which returns the orders from that ID on whatever the window
func pagedOrderHistory[T any](start, end time.Time, fromID uint64, maxSpan time.Duration, limit int, fetch func(start, end time.Time) ([]T, error), timeOf func(*T) time.Time, idOf func(*T) uint64) ([]T, error) {
	if fromID != 0 || start.IsZero() || end.IsZero() {
		return fetch(start, end)
	}
	// Binance takes times in milliseconds, so paging works in whole ones, as order times are; a window within one
	// millisecond cannot be paged and is one request
	startMilli, endMilli := start.Truncate(time.Millisecond), end.Truncate(time.Millisecond)
	if !startMilli.Before(endMilli) {
		return fetch(start, end)
	}
	start, end = startMilli, endMilli
	var orders []T
	seen := make(map[uint64]struct{})
	for spanStart := start; ; {
		spanEnd := end
		if maxSpan > 0 && spanStart.Add(maxSpan).Before(end) {
			spanEnd = spanStart.Add(maxSpan)
		}
		for pageStart := spanStart; ; {
			page, err := fetch(pageStart, spanEnd)
			if err != nil {
				return nil, err
			}
			added := 0
			for i := range page {
				id := idOf(&page[i])
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				orders = append(orders, page[i])
				added++
			}
			if len(page) < limit {
				break
			}
			next := timeOf(&page[len(page)-1])
			if next.Before(pageStart) {
				next = pageStart
			}
			if added == 0 {
				// A full page of orders already fetched cannot reach the rest of its millisecond, so paging moves past it
				next = next.Add(time.Millisecond)
			}
			if !next.Before(spanEnd) {
				// A window must start before it ends, so the span's last millisecond gets one more page from the
				// millisecond before it
				last := spanEnd.Add(-time.Millisecond)
				if !pageStart.Before(last) {
					break
				}
				next = last
			}
			pageStart = next
		}
		if !spanEnd.Before(end) {
			return orders, nil
		}
		spanStart = spanEnd
	}
}

// spotOrderHistory returns spot orders, or every symbol's order lists for the OCO type, up to 1000 a request
func (e *Exchange) spotOrderHistory(ctx context.Context, req *order.MultiOrderRequest, fromID uint64) ([]order.Detail, error) {
	if req.Type == order.OCO {
		lists, err := pagedOrderHistory(req.StartTime, req.EndTime, fromID, spotOrderHistorySpan, 1000, func(start, end time.Time) ([]OCOOrderResponse, error) {
			return e.GetAllOCOOrders(ctx, &AllOrderListsRequest{FromID: fromID, StartTime: start, EndTime: end, Limit: 1000})
		}, func(l *OCOOrderResponse) time.Time { return l.TransactionTime.Time() }, func(l *OCOOrderResponse) uint64 { return l.OrderListID })
		if err != nil {
			return nil, err
		}
		orders := make([]order.Detail, 0, len(lists))
		for i := range lists {
			l := &lists[i]
			orders = e.appendListedOrder(orders, asset.Spot, l.OrderListID, currency.EMPTYPAIR, l.Symbol, func(pair currency.Pair) (order.Detail, error) {
				return e.orderListDetail(asset.Spot, l.OrderListID, l.ListClientOrderID, l.ListOrderStatus, l.TransactionTime.Time(), pair)
			})
		}
		return orders, nil
	}
	var orders []order.Detail
	for _, p := range req.Pairs {
		history, err := pagedOrderHistory(req.StartTime, req.EndTime, fromID, spotOrderHistorySpan, 1000, func(start, end time.Time) ([]TradeOrderResponse, error) {
			return e.AllOrders(ctx, &AllOrdersRequest{Symbol: p, OrderID: fromID, StartTime: start, EndTime: end, Limit: 1000})
		}, func(o *TradeOrderResponse) time.Time { return o.Time.Time() }, func(o *TradeOrderResponse) uint64 { return o.OrderID })
		if err != nil {
			return nil, err
		}
		for i := range history {
			o := &history[i]
			orders = e.appendListedOrder(orders, asset.Spot, o.OrderID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
				return e.spotOrderDetail(o, pair)
			})
		}
	}
	return orders, nil
}

// marginOrderHistory returns margin orders up to 500 a request, or order lists up to 1000 for the OCO type
func (e *Exchange) marginOrderHistory(ctx context.Context, req *order.MultiOrderRequest, fromID uint64) ([]order.Detail, error) {
	isIsolated := req.MarginType == margin.Isolated
	var orders []order.Detail
	if req.Type == order.OCO {
		pairs, err := marginOrderListPairs(req.Pairs, isIsolated)
		if err != nil {
			return nil, err
		}
		for _, p := range pairs {
			var lists []MarginOrderListResponse
			if lists, err = pagedOrderHistory(req.StartTime, req.EndTime, fromID, marginOrderHistorySpan, 1000, func(start, end time.Time) ([]MarginOrderListResponse, error) {
				return e.GetMarginAccountAllOCO(ctx, &MarginAccountAllOCORequest{IsIsolated: isIsolated, Symbol: p, FromID: fromID, StartTime: start, EndTime: end, Limit: 1000})
			}, func(l *MarginOrderListResponse) time.Time { return l.TransactionTime.Time() }, func(l *MarginOrderListResponse) uint64 { return l.OrderListID }); err != nil {
				return nil, err
			}
			orders = e.appendMarginOrderLists(orders, p, lists)
		}
		return orders, nil
	}
	for _, p := range req.Pairs {
		history, err := pagedOrderHistory(req.StartTime, req.EndTime, fromID, marginOrderHistorySpan, 500, func(start, end time.Time) ([]MarginTradeOrderResponse, error) {
			return e.GetMarginAccountAllOrders(ctx, &MarginAccountAllOrdersRequest{Symbol: p, IsIsolated: isIsolated, OrderID: fromID, StartTime: start, EndTime: end, Limit: 500})
		}, func(o *MarginTradeOrderResponse) time.Time { return o.Time.Time() }, func(o *MarginTradeOrderResponse) uint64 { return o.OrderID })
		if err != nil {
			return nil, err
		}
		for i := range history {
			o := &history[i]
			orders = e.appendListedOrder(orders, asset.Margin, o.OrderID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
				return e.marginOrderDetail(o, pair)
			})
		}
	}
	return orders, nil
}

// uOrderHistory returns USDⓈ-M orders and algo orders up to 1000 a request each; FromOrderID applies to orders alone,
// since algo orders are numbered apart
func (e *Exchange) uOrderHistory(ctx context.Context, req *order.MultiOrderRequest, fromID uint64) ([]order.Detail, error) {
	var orders []order.Detail
	for _, p := range req.Pairs {
		if !isConditionalOrderType(req.Type) {
			history, err := pagedOrderHistory(req.StartTime, req.EndTime, fromID, futuresOrderHistorySpan, 1000, func(start, end time.Time) ([]UFuturesOrderData, error) {
				return e.UAllAccountOrders(ctx, &UAllOrdersRequest{Symbol: p, OrderID: fromID, StartTime: start, EndTime: end, Limit: 1000})
			}, func(o *UFuturesOrderData) time.Time { return o.Time.Time() }, func(o *UFuturesOrderData) uint64 { return o.OrderID })
			if err != nil {
				return nil, err
			}
			for i := range history {
				o := &history[i]
				orders = e.appendListedOrder(orders, asset.USDTMarginedFutures, o.OrderID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
					return e.uOrderDetail(&o.UOrderResponse, pair)
				})
			}
		}
		if req.Type == order.AnyType || isConditionalOrderType(req.Type) {
			algoHistory, err := pagedOrderHistory(req.StartTime, req.EndTime, 0, futuresOrderHistorySpan, 1000, func(start, end time.Time) ([]UAlgoOrder, error) {
				return e.UQueryAllAlgoOrders(ctx, &UAllAlgoOrdersRequest{Symbol: p, StartTime: start, EndTime: end, Limit: 1000})
			}, func(o *UAlgoOrder) time.Time { return o.CreateTime.Time() }, func(o *UAlgoOrder) uint64 { return o.AlgoID })
			if err != nil {
				return nil, err
			}
			for i := range algoHistory {
				o := &algoHistory[i]
				orders = e.appendListedOrder(orders, asset.USDTMarginedFutures, o.AlgoID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
					return e.uAlgoOrderDetail(&o.UAlgoOrderBase, o.ActualPrice.Float64(), pair)
				})
			}
		}
	}
	return orders, nil
}

// cOrderHistory returns COIN-M orders up to 100 a request
func (e *Exchange) cOrderHistory(ctx context.Context, req *order.MultiOrderRequest, fromID uint64) ([]order.Detail, error) {
	var orders []order.Detail
	for _, p := range req.Pairs {
		history, err := pagedOrderHistory(req.StartTime, req.EndTime, fromID, futuresOrderHistorySpan, 100, func(start, end time.Time) ([]FuturesOrderData, error) {
			return e.GetAllFuturesOrders(ctx, &CFuturesAllOrdersRequest{Symbol: p, OrderID: fromID, StartTime: start, EndTime: end, Limit: 100})
		}, func(o *FuturesOrderData) time.Time { return o.Time.Time() }, func(o *FuturesOrderData) uint64 { return o.OrderID })
		if err != nil {
			return nil, err
		}
		for i := range history {
			o := &history[i]
			orders = e.appendListedOrder(orders, asset.CoinMarginedFutures, o.OrderID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
				return e.cOrderDetail(&o.FuturesOrderDetailResponse, pair)
			})
		}
	}
	return orders, nil
}

// optionsOrderHistory returns options orders finished within the last five days, up to 1000 a request
func (e *Exchange) optionsOrderHistory(ctx context.Context, req *order.MultiOrderRequest, fromID uint64) ([]order.Detail, error) {
	var orders []order.Detail
	for _, p := range req.Pairs {
		history, err := pagedOrderHistory(req.StartTime, req.EndTime, fromID, 0, 1000, func(start, end time.Time) ([]OptionsOrderHistoryItem, error) {
			return e.GetOptionsOrdersHistory(ctx, &OptionsOrderHistoryRequest{Symbol: p, OrderID: fromID, StartTime: start, EndTime: end, Limit: 1000})
		}, func(o *OptionsOrderHistoryItem) time.Time { return o.CreateTime.Time() }, func(o *OptionsOrderHistoryItem) uint64 { return o.OrderID })
		if err != nil {
			return nil, err
		}
		for i := range history {
			o := &history[i]
			orders = e.appendListedOrder(orders, asset.Options, o.OrderID, p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
				return e.optionsOrderDetail(&o.OptionsOrder, o.CreateTime.Time(), false, pair)
			})
		}
	}
	return orders, nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (e *Exchange) FormatExchangeKlineInterval(interval kline.Interval) string {
	switch interval {
	case kline.OneDay:
		return "1d"
	case kline.ThreeDay:
		return "3d"
	case kline.OneWeek:
		return "1w"
	case kline.OneMonth:
		return "1M"
	default:
		return interval.Short()
	}
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	candles, err := e.klineCandles(ctx, a, req.RequestFormatted, req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended returns candles between a time period for a set
// time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		candles, err := e.klineCandles(ctx, a, req.RequestFormatted, req.ExchangeInterval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time, req.RangeHolder.Limit)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, candles...)
	}
	return req.ProcessResponse(timeSeries)
}

// klineCandles returns up to limit candles of an asset's pair from start to end. One second candles are spot and
// margin only
func (e *Exchange) klineCandles(ctx context.Context, a asset.Item, pair currency.Pair, interval kline.Interval, start, end time.Time, limit uint64) ([]kline.Candle, error) {
	if interval < kline.OneMin && a != asset.Spot && a != asset.Margin {
		return nil, fmt.Errorf("%w: %s %s candles", kline.ErrUnsupportedInterval, interval, a)
	}
	var candles []kline.Candle
	switch a {
	case asset.Spot, asset.Margin:
		resp, err := e.GetSpotKline(ctx, &KlinesRequest{
			Symbol:    pair,
			Interval:  e.FormatExchangeKlineInterval(interval),
			StartTime: start,
			EndTime:   end,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
		candles = make([]kline.Candle, len(resp))
		for i, c := range resp {
			candles[i] = kline.Candle{
				Time:   c.OpenTime.Time().UTC(),
				Open:   c.OpenPrice.Float64(),
				High:   c.HighPrice.Float64(),
				Low:    c.LowPrice.Float64(),
				Close:  c.ClosePrice.Float64(),
				Volume: c.Volume.Float64(),
			}
		}
	case asset.USDTMarginedFutures:
		resp, err := e.UKlineData(ctx, &UKlineRequest{
			Symbol:    pair,
			Interval:  e.FormatExchangeKlineInterval(interval),
			StartTime: start,
			EndTime:   end,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
		candles = make([]kline.Candle, len(resp))
		for i := range resp {
			candles[i] = kline.Candle{
				Time:   resp[i].OpenTime.Time().UTC(),
				Open:   resp[i].Open.Float64(),
				High:   resp[i].High.Float64(),
				Low:    resp[i].Low.Float64(),
				Close:  resp[i].Close.Float64(),
				Volume: resp[i].Volume.Float64(),
			}
		}
	case asset.CoinMarginedFutures:
		resp, err := e.GetFuturesKlineData(ctx, &CFuturesKlineRequest{
			Symbol:    pair,
			Interval:  interval,
			StartTime: start,
			EndTime:   end,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
		candles = make([]kline.Candle, len(resp))
		for i := range resp {
			// The volume counts contracts, as the kline stream sends it
			candles[i] = kline.Candle{
				Time:   resp[i].OpenTime.Time().UTC(),
				Open:   resp[i].Open.Float64(),
				High:   resp[i].High.Float64(),
				Low:    resp[i].Low.Float64(),
				Close:  resp[i].Close.Float64(),
				Volume: resp[i].Volume.Float64(),
			}
		}
	case asset.Options:
		resp, err := e.GetEOptionsCandlesticks(ctx, &OptionsKlineRequest{
			Symbol:    pair,
			Interval:  interval,
			StartTime: start,
			EndTime:   end,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
		candles = make([]kline.Candle, len(resp))
		for i := range resp {
			candles[i] = kline.Candle{
				Time:   resp[i].OpenTime.Time().UTC(),
				Open:   resp[i].Open.Float64(),
				High:   resp[i].High.Float64(),
				Low:    resp[i].Low.Float64(),
				Close:  resp[i].Close.Float64(),
				Volume: resp[i].Volume.Float64(),
			}
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	return candles, nil
}

// OrderVars holds an order's side, status, type and time in force converted from Binance's values
type OrderVars struct {
	Side        order.Side
	Status      order.Status
	OrderType   order.Type
	TimeInForce order.TimeInForce
}

// compatibleOrderVars converts an order's side, status, type and time in force as Binance sends them for asset a, the
// same way the websocket order updates are converted. Empty values are left unset, and a value that cannot be
// converted is left unset with its error joined into the returned error, so a listing can keep the order and log the
// error while a single order query returns it
func compatibleOrderVars(a asset.Item, side, status, orderType, timeInForce string) (OrderVars, error) {
	var vars OrderVars
	var errs error
	isSpot := a == asset.Spot || a == asset.Margin
	if side != "" {
		var err error
		vars.Side, err = order.StringToOrderSide(side)
		errs = common.AppendError(errs, err)
	}
	if status != "" {
		var err error
		if isSpot {
			vars.Status, err = stringToOrderStatus(status)
		} else {
			vars.Status, err = derivativesOrderStatus(status)
		}
		errs = common.AppendError(errs, err)
	}
	if orderType != "" {
		var err error
		if isSpot {
			vars.OrderType, err = stringToSpotOrderType(orderType)
		} else {
			vars.OrderType, err = derivativesOrderType(orderType)
		}
		errs = common.AppendError(errs, err)
	}
	if timeInForce != "" {
		var err error
		if isSpot {
			vars.TimeInForce, err = order.StringToTimeInForce(timeInForce)
		} else {
			vars.TimeInForce, err = derivativesTimeInForce(timeInForce)
		}
		errs = common.AppendError(errs, err)
	}
	return vars, errs
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var l []limits.MinMaxLevel
	var err error
	switch a {
	case asset.Spot, asset.Margin:
		l, err = e.FetchExchangeLimits(ctx, a)
	case asset.USDTMarginedFutures:
		l, err = e.FetchUSDTMarginExchangeLimits(ctx)
	case asset.CoinMarginedFutures:
		l, err = e.FetchCoinMarginExchangeLimits(ctx)
	case asset.Options:
		l, err = e.FetchOptionsExchangeLimits(ctx)
	default:
		err = fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err != nil {
		return fmt.Errorf("cannot update exchange execution limits: %w", err)
	}
	return limits.Load(l)
}

// filterType is the type of a symbol's trading rule filter
type filterType string

const (
	priceFilter              filterType = "PRICE_FILTER"
	percentPriceFilter       filterType = "PERCENT_PRICE"
	percentPriceBySideFilter filterType = "PERCENT_PRICE_BY_SIDE"
	lotSizeFilter            filterType = "LOT_SIZE"
	minNotionalFilter        filterType = "MIN_NOTIONAL"
	notionalFilter           filterType = "NOTIONAL"
	icebergPartsFilter       filterType = "ICEBERG_PARTS"
	marketLotSizeFilter      filterType = "MARKET_LOT_SIZE"
	maxNumOrdersFilter       filterType = "MAX_NUM_ORDERS"
	maxNumAlgoOrdersFilter   filterType = "MAX_NUM_ALGO_ORDERS"
)

// FetchExchangeLimits returns the order execution limits of the symbols the spot or margin asset trades
func (e *Exchange) FetchExchangeLimits(ctx context.Context, a asset.Item) ([]limits.MinMaxLevel, error) {
	if a != asset.Spot && a != asset.Margin {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	info, err := e.GetExchangeInfo(ctx, nil)
	if err != nil {
		return nil, err
	}
	l := make([]limits.MinMaxLevel, 0, len(info.Symbols))
	for i := range info.Symbols {
		s := &info.Symbols[i]
		if !spotSymbolTradable(s, a) {
			continue
		}
		mml := limits.MinMaxLevel{
			Key: key.NewExchangeAssetPair(e.Name, a, currency.NewPair(s.BaseAsset, s.QuoteAsset)),
		}
		for j := range s.Filters {
			f := &s.Filters[j]
			switch filterType(f.FilterType) {
			case priceFilter:
				mml.MinPrice = f.MinPrice.Float64()
				mml.MaxPrice = f.MaxPrice.Float64()
				mml.PriceStepIncrementSize = f.TickSize.Float64()
			case percentPriceFilter:
				mml.MultiplierUp = f.MultiplierUp.Float64()
				mml.MultiplierDown = f.MultiplierDown.Float64()
				mml.AveragePriceMinutes = boundedInt64(f.AveragePriceMinutes)
			case percentPriceBySideFilter:
				// The limits hold one price band for both sides, so they take the widest
				mml.MultiplierUp = max(f.BidMultiplierUp.Float64(), f.AskMultiplierUp.Float64())
				mml.MultiplierDown = min(f.BidMultiplierDown.Float64(), f.AskMultiplierDown.Float64())
				mml.AveragePriceMinutes = boundedInt64(f.AveragePriceMinutes)
			case lotSizeFilter:
				mml.MinimumBaseAmount = f.MinQuantity.Float64()
				mml.MaximumBaseAmount = f.MaxQuantity.Float64()
				mml.AmountStepIncrementSize = f.StepSize.Float64()
			case minNotionalFilter, notionalFilter:
				mml.MinNotional = f.MinNotional.Float64()
			case icebergPartsFilter:
				mml.MaxIcebergParts = boundedInt64(f.Limit)
			case marketLotSizeFilter:
				mml.MarketMinQty = f.MinQuantity.Float64()
				mml.MarketMaxQty = f.MaxQuantity.Float64()
				mml.MarketStepIncrementSize = f.StepSize.Float64()
			case maxNumOrdersFilter:
				mml.MaxTotalOrders = boundedInt64(f.MaxNumberOrders)
			case maxNumAlgoOrdersFilter:
				mml.MaxAlgoOrders = boundedInt64(f.MaxNumberAlgoOrders)
			}
		}
		l = append(l, mml)
	}
	return l, nil
}

// FetchUSDTMarginExchangeLimits returns the order execution limits of every USDⓈ-M contract
func (e *Exchange) FetchUSDTMarginExchangeLimits(ctx context.Context) ([]limits.MinMaxLevel, error) {
	info, err := e.UExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}
	l := make([]limits.MinMaxLevel, 0, len(info.Symbols))
	for i := range info.Symbols {
		s := &info.Symbols[i]
		mml := limits.MinMaxLevel{
			Key: key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, uFuturesSymbolPair(s.Symbol, s.BaseAsset, s.QuoteAsset)),
		}
		for j := range s.Filters {
			f := &s.Filters[j]
			switch filterType(f.FilterType) {
			case priceFilter:
				mml.MinPrice = f.MinPrice.Float64()
				mml.MaxPrice = f.MaxPrice.Float64()
				mml.PriceStepIncrementSize = f.TickSize.Float64()
			case percentPriceFilter:
				mml.MultiplierUp = f.MultiplierUp.Float64()
				mml.MultiplierDown = f.MultiplierDown.Float64()
				mml.MultiplierDecimal = f.MultiplierDecimal.Float64()
			case lotSizeFilter:
				mml.MinimumBaseAmount = f.MinQuantity.Float64()
				mml.MaximumBaseAmount = f.MaxQuantity.Float64()
				mml.AmountStepIncrementSize = f.StepSize.Float64()
			case minNotionalFilter:
				mml.MinNotional = f.Notional.Float64()
			case marketLotSizeFilter:
				mml.MarketMinQty = f.MinQuantity.Float64()
				mml.MarketMaxQty = f.MaxQuantity.Float64()
				mml.MarketStepIncrementSize = f.StepSize.Float64()
			case maxNumOrdersFilter:
				mml.MaxTotalOrders = boundedInt64(f.Limit)
			case maxNumAlgoOrdersFilter:
				mml.MaxAlgoOrders = boundedInt64(f.Limit)
			}
		}
		l = append(l, mml)
	}
	return l, nil
}

// FetchCoinMarginExchangeLimits returns the order execution limits of every COIN-M contract, whose quantities count
// contracts
func (e *Exchange) FetchCoinMarginExchangeLimits(ctx context.Context) ([]limits.MinMaxLevel, error) {
	info, err := e.FuturesExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}
	l := make([]limits.MinMaxLevel, 0, len(info.Symbols))
	for i := range info.Symbols {
		s := &info.Symbols[i]
		pair, err := currency.NewPairDelimiter(s.Symbol, currency.UnderscoreDelimiter)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s skipping %s symbol limits: %s", e.Name, asset.CoinMarginedFutures, err)
			continue
		}
		mml := limits.MinMaxLevel{
			Key: key.NewExchangeAssetPair(e.Name, asset.CoinMarginedFutures, pair),
		}
		for j := range s.Filters {
			f := &s.Filters[j]
			switch filterType(f.FilterType) {
			case priceFilter:
				mml.MinPrice = f.MinPrice.Float64()
				mml.MaxPrice = f.MaxPrice.Float64()
				mml.PriceStepIncrementSize = f.TickSize.Float64()
			case percentPriceFilter:
				mml.MultiplierUp = f.MultiplierUp.Float64()
				mml.MultiplierDown = f.MultiplierDown.Float64()
				mml.MultiplierDecimal = f.MultiplierDecimal.Float64()
			case lotSizeFilter:
				mml.MinimumBaseAmount = f.MinQuantity.Float64()
				mml.MaximumBaseAmount = f.MaxQuantity.Float64()
				mml.AmountStepIncrementSize = f.StepSize.Float64()
			case marketLotSizeFilter:
				mml.MarketMinQty = f.MinQuantity.Float64()
				mml.MarketMaxQty = f.MaxQuantity.Float64()
				mml.MarketStepIncrementSize = f.StepSize.Float64()
			case maxNumOrdersFilter:
				mml.MaxTotalOrders = f.Limit
			case maxNumAlgoOrdersFilter:
				mml.MaxAlgoOrders = f.Limit
			}
		}
		l = append(l, mml)
	}
	return l, nil
}

// FetchOptionsExchangeLimits returns the order execution limits of every option. Options take limit orders only and
// have no open order limits
func (e *Exchange) FetchOptionsExchangeLimits(ctx context.Context) ([]limits.MinMaxLevel, error) {
	info, err := e.GetOptionsExchangeInformation(ctx)
	if err != nil {
		return nil, err
	}
	l := make([]limits.MinMaxLevel, 0, len(info.OptionSymbols))
	for i := range info.OptionSymbols {
		s := &info.OptionSymbols[i]
		pair, err := currency.NewPairDelimiter(s.Symbol, currency.DashDelimiter)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s skipping %s symbol limits: %s", e.Name, asset.Options, err)
			continue
		}
		mml := limits.MinMaxLevel{
			Key: key.NewExchangeAssetPair(e.Name, asset.Options, pair),
		}
		for j := range s.Filters {
			f := &s.Filters[j]
			switch filterType(f.FilterType) {
			case priceFilter:
				mml.MinPrice = f.MinPrice.Float64()
				mml.MaxPrice = f.MaxPrice.Float64()
				mml.PriceStepIncrementSize = f.TickSize.Float64()
			case lotSizeFilter:
				mml.MinimumBaseAmount = f.MinQuantity.Float64()
				mml.MaximumBaseAmount = f.MaxQuantity.Float64()
				mml.AmountStepIncrementSize = f.StepSize.Float64()
			}
		}
		l = append(l, mml)
	}
	return l, nil
}

// boundedInt64 converts a count into a signed integer, clamping a count beyond its range
func boundedInt64(count uint64) int64 {
	return int64(min(count, math.MaxInt64))
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	coins, err := e.GetAllCoinsInfo(ctx)
	if err != nil {
		return nil, err
	}
	for i := range coins {
		if !coins[i].Coin.Equal(cryptocurrency) {
			continue
		}
		chains := make([]string, len(coins[i].NetworkList))
		for j := range coins[i].NetworkList {
			chains[j] = coins[i].NetworkList[j].Network
		}
		if len(chains) == 0 {
			break
		}
		return chains, nil
	}
	return nil, fmt.Errorf("%w: no transfer chains for %s", currency.ErrCurrencyNotFound, cryptocurrency)
}

// FormatExchangeCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
// overrides default implementation to use optional delimiter
func (e *Exchange) FormatExchangeCurrency(p currency.Pair, a asset.Item) (currency.Pair, error) {
	pairFmt, err := e.GetPairFormat(a, true)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	if a == asset.USDTMarginedFutures {
		return e.formatUSDTMarginedFuturesPair(p, pairFmt), nil
	}
	return p.Format(pairFmt), nil
}

// FormatSymbol formats the given pair to a string suitable for exchange API requests
// overrides default implementation to use optional delimiter
func (e *Exchange) FormatSymbol(p currency.Pair, a asset.Item) (string, error) {
	pairFmt, err := e.GetPairFormat(a, true)
	if err != nil {
		return p.String(), err
	}
	if a == asset.USDTMarginedFutures {
		p = e.formatUSDTMarginedFuturesPair(p, pairFmt)
		return p.String(), nil
	}
	return pairFmt.Format(p), nil
}

// formatUSDTMarginedFuturesPair Binance USDTMarginedFutures pairs have a delimiter
// only if the contract has an expiry date
func (e *Exchange) formatUSDTMarginedFuturesPair(p currency.Pair, pairFmt currency.PairFormat) currency.Pair {
	quote := p.Quote.String()
	for _, c := range quote {
		if c < '0' || c > '9' {
			// character rune is alphabetic, cannot be expiring contract
			return p.Format(pairFmt)
		}
	}
	pairFmt.Delimiter = currency.UnderscoreDelimiter
	return p.Format(pairFmt)
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	switch a {
	case asset.Spot, asset.Margin:
		resp, err := e.GetExchangeServerTime(ctx)
		if err != nil {
			return time.Time{}, err
		}
		return resp.ServerTime.Time(), nil
	case asset.USDTMarginedFutures:
		return e.UServerTime(ctx)
	case asset.CoinMarginedFutures:
		return e.CFuturesServerTime(ctx)
	case asset.Options:
		resp, err := e.CheckEOptionsServerTime(ctx)
		if err != nil {
			return time.Time{}, err
		}
		return resp.ServerTime.Time(), nil
	default:
		return time.Time{}, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// defaultFundingInterval is the funding interval of the perpetual contracts the funding rate info does not list, which
// only lists those whose interval, cap or floor was adjusted
const defaultFundingInterval = 8 * time.Hour

// fundingRatePageLimit is the most funding rate or income records one request returns
const fundingRatePageLimit = 1000

// GetLatestFundingRates returns the latest funding rate of a perpetual contract, or of every enabled perpetual contract
// of the asset when the request has no pair
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w IncludePredictedRate", common.ErrFunctionNotSupported)
	}
	var resp []fundingrate.LatestRateResponse
	switch r.Asset {
	case asset.USDTMarginedFutures:
		if err := e.requirePerpetual(r.Asset, r.Pair); err != nil {
			return nil, err
		}
		intervals, err := e.uFundingIntervals(ctx)
		if err != nil {
			return nil, err
		}
		marks, err := e.UGetMarkPrice(ctx, r.Pair)
		if err != nil {
			return nil, err
		}
		resp = make([]fundingrate.LatestRateResponse, 0, len(marks))
		for i := range marks {
			rate, ok, err := e.latestFundingRate(r, marks[i].Symbol, marks[i].LastFundingRate, marks[i].NextFundingTime.Time(), marks[i].Time.Time(), intervals)
			if err != nil {
				return nil, err
			}
			if ok {
				resp = append(resp, rate)
			}
		}
	case asset.CoinMarginedFutures:
		if err := e.requirePerpetual(r.Asset, r.Pair); err != nil {
			return nil, err
		}
		intervals, err := e.cFundingIntervals(ctx)
		if err != nil {
			return nil, err
		}
		marks, err := e.GetIndexAndMarkPrice(ctx, r.Pair, currency.EMPTYCODE)
		if err != nil {
			return nil, err
		}
		resp = make([]fundingrate.LatestRateResponse, 0, len(marks))
		for i := range marks {
			rate, ok, err := e.latestFundingRate(r, marks[i].Symbol, marks[i].LastFundingRate, marks[i].NextFundingTime.Time(), marks[i].Time.Time(), intervals)
			if err != nil {
				return nil, err
			}
			if ok {
				resp = append(resp, rate)
			}
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, r.Asset)
	}
	return resp, nil
}

// requirePerpetual returns an error for a pair that is not a perpetual contract of the asset; an empty pair stands for
// every contract
func (e *Exchange) requirePerpetual(a asset.Item, pair currency.Pair) error {
	if pair.IsEmpty() {
		return nil
	}
	isPerpetual, err := e.IsPerpetualFutureCurrency(a, pair)
	if err != nil {
		return err
	}
	if !isPerpetual {
		return fmt.Errorf("%w: %s %s", futures.ErrNotPerpetualFuture, a, pair)
	}
	return nil
}

// latestFundingRate converts a contract's funding rate from its mark price. A request for every pair reports false for
// a contract that is not an enabled perpetual
func (e *Exchange) latestFundingRate(r *fundingrate.LatestRateRequest, symbol string, rate types.Number, nextFundingTime, checked time.Time, intervals map[string]time.Duration) (fundingrate.LatestRateResponse, bool, error) {
	pair := r.Pair
	if pair.IsEmpty() {
		var err error
		if pair, err = e.MatchSymbolWithAvailablePairs(symbol, r.Asset, true); err != nil {
			if errors.Is(err, currency.ErrPairNotFound) {
				return fundingrate.LatestRateResponse{}, false, nil
			}
			return fundingrate.LatestRateResponse{}, false, err
		}
		enabled, err := e.IsPairEnabled(pair, r.Asset)
		if err != nil {
			return fundingrate.LatestRateResponse{}, false, err
		}
		if isPerpetual, err := e.IsPerpetualFutureCurrency(r.Asset, pair); err != nil || !enabled || !isPerpetual {
			return fundingrate.LatestRateResponse{}, false, err
		}
	}
	interval, ok := intervals[symbol]
	if !ok {
		interval = defaultFundingInterval
	}
	return fundingrate.LatestRateResponse{
		Exchange: e.Name,
		Asset:    r.Asset,
		Pair:     pair,
		LatestRate: fundingrate.Rate{
			Time: nextFundingTime.Add(-interval),
			Rate: rate.Decimal(),
		},
		TimeOfNextRate: nextFundingTime,
		TimeChecked:    checked,
	}, true, nil
}

// uFundingIntervals returns the funding interval of each USDⓈ-M contract the funding rate info lists
func (e *Exchange) uFundingIntervals(ctx context.Context) (map[string]time.Duration, error) {
	info, err := e.UGetFundingRateInfo(ctx)
	if err != nil {
		return nil, err
	}
	intervals := make(map[string]time.Duration, len(info))
	for i := range info {
		intervals[info[i].Symbol] = time.Duration(boundedInt64(info[i].FundingIntervalHours)) * time.Hour
	}
	return intervals, nil
}

// cFundingIntervals returns the funding interval of each COIN-M contract the funding rate info lists
func (e *Exchange) cFundingIntervals(ctx context.Context) (map[string]time.Duration, error) {
	info, err := e.GetFundingRateInfo(ctx)
	if err != nil {
		return nil, err
	}
	intervals := make(map[string]time.Duration, len(info))
	for i := range info {
		intervals[info[i].Symbol] = time.Duration(boundedInt64(info[i].FundingIntervalHours)) * time.Hour
	}
	return intervals, nil
}

// GetHistoricalFundingRates returns funding rates for a given asset and currency for a time period
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w IncludePredictedRate", common.ErrFunctionNotSupported)
	}
	if !r.PaymentCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w PaymentCurrency", common.ErrFunctionNotSupported)
	}
	if r.Asset != asset.USDTMarginedFutures && r.Asset != asset.CoinMarginedFutures {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	if err := e.requirePerpetual(r.Asset, r.Pair); err != nil {
		return nil, err
	}
	symbol, err := e.FormatSymbol(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	resp := &fundingrate.HistoricalRates{
		Exchange:  e.Name,
		Asset:     r.Asset,
		Pair:      r.Pair,
		StartDate: r.StartDate,
		EndDate:   r.EndDate,
	}
	if r.Asset == asset.USDTMarginedFutures {
		err = e.uHistoricalFundingRates(ctx, r, symbol, resp)
	} else {
		err = e.cHistoricalFundingRates(ctx, r, symbol, resp)
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// fundingPayment is a funding fee paid or received
type fundingPayment struct {
	time   time.Time
	asset  currency.Code
	amount types.Number
}

// addFundingPayments adds each payment to the payment sum and to the rate nearest it, that of the funding time it was
// paid at; funding times are a few milliseconds past the hour, so payments are matched by proximity. Payments more than
// half a funding interval from every rate fall outside the rates fetched. The rates are in ascending time order
func addFundingPayments(resp *fundingrate.HistoricalRates, payments []fundingPayment, interval time.Duration) {
	for i := range payments {
		x := sort.Search(len(resp.FundingRates), func(j int) bool { return !resp.FundingRates[j].Time.Before(payments[i].time) })
		if x == len(resp.FundingRates) || (x > 0 && payments[i].time.Sub(resp.FundingRates[x-1].Time) < resp.FundingRates[x].Time.Sub(payments[i].time)) {
			x--
		}
		if x < 0 || payments[i].time.Sub(resp.FundingRates[x].Time).Abs() >= interval/2 {
			continue
		}
		if resp.PaymentCurrency.IsEmpty() {
			resp.PaymentCurrency = payments[i].asset
		}
		amount := payments[i].amount.Decimal()
		resp.FundingRates[x].Payment = resp.FundingRates[x].Payment.Add(amount)
		resp.PaymentSum = resp.PaymentSum.Add(amount)
	}
}

// uHistoricalFundingRates adds a USDⓈ-M contract's funding rates, latest rate and, when requested, funding fees to resp
func (e *Exchange) uHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest, symbol string, resp *fundingrate.HistoricalRates) error {
	intervals, err := e.uFundingIntervals(ctx)
	if err != nil {
		return err
	}
	interval := cmp.Or(intervals[symbol], defaultFundingInterval)
	for start := r.StartDate; ; {
		page, err := e.UGetFundingHistory(ctx, r.Pair, start, r.EndDate, fundingRatePageLimit)
		if err != nil {
			return err
		}
		for i := range page {
			resp.FundingRates = append(resp.FundingRates, fundingrate.Rate{Time: page[i].FundingTime.Time(), Rate: page[i].FundingRate.Decimal()})
		}
		if len(page) < fundingRatePageLimit {
			break
		}
		// The start time is inclusive
		start = page[len(page)-1].FundingTime.Time().Add(time.Millisecond)
	}
	marks, err := e.UGetMarkPrice(ctx, r.Pair)
	if err != nil {
		return err
	}
	if len(marks) == 0 {
		return fmt.Errorf("%w: no mark price for %s", fundingrate.ErrNoFundingRatesFound, symbol)
	}
	resp.LatestRate = fundingrate.Rate{Time: marks[0].NextFundingTime.Time().Add(-interval), Rate: marks[0].LastFundingRate.Decimal()}
	resp.TimeOfNextRate = marks[0].NextFundingTime.Time()
	if !r.IncludePayments {
		return nil
	}
	var payments []fundingPayment
	for start, end := range fundingFeeWindows(r.StartDate, r.EndDate) {
		income, err := e.UAccountIncomeHistory(ctx, &UIncomeHistoryRequest{
			Symbol:     r.Pair,
			IncomeType: "FUNDING_FEE",
			StartTime:  start,
			EndTime:    end,
			Limit:      fundingRatePageLimit,
		})
		if err != nil {
			return err
		}
		for i := range income {
			payments = append(payments, fundingPayment{time: income[i].Time.Time(), asset: income[i].Asset, amount: income[i].Income})
		}
	}
	addFundingPayments(resp, payments, interval)
	return nil
}

// cHistoricalFundingRates adds a COIN-M contract's funding rates, latest rate and, when requested, funding fees to resp
func (e *Exchange) cHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest, symbol string, resp *fundingrate.HistoricalRates) error {
	intervals, err := e.cFundingIntervals(ctx)
	if err != nil {
		return err
	}
	interval := cmp.Or(intervals[symbol], defaultFundingInterval)
	for start := r.StartDate; ; {
		page, err := e.FuturesGetFundingHistory(ctx, &CFuturesFundingRateHistoryRequest{
			Symbol:    r.Pair,
			StartTime: start,
			EndTime:   r.EndDate,
			Limit:     fundingRatePageLimit,
		})
		if err != nil {
			return err
		}
		for i := range page {
			resp.FundingRates = append(resp.FundingRates, fundingrate.Rate{Time: page[i].FundingTime.Time(), Rate: page[i].FundingRate.Decimal()})
		}
		if len(page) < fundingRatePageLimit {
			break
		}
		// The start time is inclusive
		start = page[len(page)-1].FundingTime.Time().Add(time.Millisecond)
	}
	marks, err := e.GetIndexAndMarkPrice(ctx, r.Pair, currency.EMPTYCODE)
	if err != nil {
		return err
	}
	if len(marks) == 0 {
		return fmt.Errorf("%w: no mark price for %s", fundingrate.ErrNoFundingRatesFound, symbol)
	}
	resp.LatestRate = fundingrate.Rate{Time: marks[0].NextFundingTime.Time().Add(-interval), Rate: marks[0].LastFundingRate.Decimal()}
	resp.TimeOfNextRate = marks[0].NextFundingTime.Time()
	if !r.IncludePayments {
		return nil
	}
	var payments []fundingPayment
	for start, end := range fundingFeeWindows(r.StartDate, r.EndDate) {
		income, err := e.FuturesIncomeHistory(ctx, &CFuturesIncomeHistoryRequest{
			Symbol:     r.Pair,
			IncomeType: "FUNDING_FEE",
			StartTime:  start,
			EndTime:    end,
			Limit:      fundingRatePageLimit,
		})
		if err != nil {
			return err
		}
		for i := range income {
			payments = append(payments, fundingPayment{time: income[i].Time.Time(), asset: income[i].Asset, amount: income[i].Income})
		}
	}
	addFundingPayments(resp, payments, interval)
	return nil
}

// fundingFeeWindows splits a time range into inclusive windows short enough for their funding fees to fit one page: a
// contract pays at most two fees each funding time, one for each position side in hedge mode, and the windows assume
// Binance's shortest funding interval, an hour, since a contract may have paid more often in the past than it does now
func fundingFeeWindows(start, end time.Time) iter.Seq2[time.Time, time.Time] {
	const span = time.Hour * fundingRatePageLimit / 2
	return func(yield func(time.Time, time.Time) bool) {
		for windowStart := start; !windowStart.After(end); windowStart = windowStart.Add(span) {
			windowEnd := windowStart.Add(span - time.Millisecond)
			if windowEnd.After(end) {
				windowEnd = end
			}
			if !yield(windowStart, windowEnd) {
				return
			}
		}
	}
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	switch a {
	case asset.CoinMarginedFutures:
		if cp.IsEmpty() {
			return false, currency.ErrCurrencyPairEmpty
		}
		return cp.Quote.Equal(currency.PERP), nil
	case asset.USDTMarginedFutures:
		if cp.IsEmpty() {
			return false, currency.ErrCurrencyPairEmpty
		}
		// A delivery contract quotes its delivery date (BTCUSDT/261225), a perpetual its quote asset
		return !isDeliveryDate(cp.Quote), nil
	default:
		return false, nil
	}
}

// isDeliveryDate reports whether a pair's quote is the delivery date of a delivery contract, such as 261225
func isDeliveryDate(quote currency.Code) bool {
	s := quote.String()
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// SetCollateralMode sets the USDⓈ-M account's collateral mode: Multi-Assets Mode or Single-Asset Mode
func (e *Exchange) SetCollateralMode(ctx context.Context, a asset.Item, collateralMode collateral.Mode) error {
	if a != asset.USDTMarginedFutures {
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if collateralMode != collateral.MultiMode && collateralMode != collateral.SingleMode {
		return fmt.Errorf("%w %v", order.ErrCollateralInvalid, collateralMode)
	}
	return e.SetAssetsMode(ctx, collateralMode == collateral.MultiMode)
}

// GetCollateralMode returns the USDⓈ-M account's collateral mode: Multi-Assets Mode or Single-Asset Mode
func (e *Exchange) GetCollateralMode(ctx context.Context, a asset.Item) (collateral.Mode, error) {
	if a != asset.USDTMarginedFutures {
		return collateral.UnknownMode, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	isMulti, err := e.GetAssetsMode(ctx)
	if err != nil {
		return collateral.UnknownMode, err
	}
	if isMulti {
		return collateral.MultiMode, nil
	}
	return collateral.SingleMode, nil
}

// SetMarginType sets a USDⓈ-M or COIN-M symbol's margin type, isolated or cross
func (e *Exchange) SetMarginType(ctx context.Context, item asset.Item, pair currency.Pair, tp margin.Type) error {
	if item != asset.USDTMarginedFutures && item != asset.CoinMarginedFutures {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	mt, err := e.marginTypeToString(tp)
	if err != nil {
		return err
	}
	if item == asset.CoinMarginedFutures {
		_, err = e.FuturesChangeMarginType(ctx, pair, mt)
		return err
	}
	return e.UChangeInitialMarginType(ctx, pair, mt)
}

// ChangePositionMargin adds margin to or removes margin from an isolated USDⓈ-M or COIN-M position: the difference
// between its original and its new allocated margin. MarginSide is the position side (BOTH, LONG or SHORT), which
// Hedge Mode positions require
func (e *Exchange) ChangePositionMargin(ctx context.Context, req *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionChangeRequest", common.ErrNilPointer)
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Asset != asset.USDTMarginedFutures && req.Asset != asset.CoinMarginedFutures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	if req.NewAllocatedMargin == 0 {
		return nil, fmt.Errorf("%w %v %v", margin.ErrNewAllocatedMarginRequired, req.Asset, req.Pair)
	}
	if req.OriginalAllocatedMargin == 0 {
		return nil, fmt.Errorf("%w %v %v", margin.ErrOriginalPositionMarginRequired, req.Asset, req.Pair)
	}
	if req.MarginType == margin.Multi {
		return nil, fmt.Errorf("%w %v %v", margin.ErrMarginTypeUnsupported, req.Asset, req.Pair)
	}
	for _, m := range []float64{req.NewAllocatedMargin, req.OriginalAllocatedMargin} {
		if math.IsNaN(m) || math.IsInf(m, 0) {
			return nil, fmt.Errorf("%w: %v", errInvalidMarginAmount, m)
		}
	}
	// Decimal arithmetic keeps the change as precise as the amounts, which float subtraction would not
	change := decimal.MustFromFloat(req.NewAllocatedMargin).Sub(decimal.MustFromFloat(req.OriginalAllocatedMargin))
	if change.IsZero() {
		return nil, fmt.Errorf("%w: the new allocated margin equals the original", errMarginAmountRequired)
	}
	amount := change.Abs().InexactFloat64()
	positionSide := strings.ToUpper(req.MarginSide)
	var err error
	if req.Asset == asset.CoinMarginedFutures {
		changeType := uint64(1) // 1 adds position margin and 2 reduces it
		if change.IsNegative() {
			changeType = 2
		}
		_, err = e.ModifyIsolatedPositionMargin(ctx, &CFuturesModifyIsolatedPositionMarginRequest{Symbol: req.Pair, PositionSide: positionSide, Amount: amount, Type: changeType})
	} else {
		changeType := "add"
		if change.IsNegative() {
			changeType = "reduce"
		}
		_, err = e.UModifyIsolatedPositionMarginReq(ctx, req.Pair, positionSide, changeType, amount)
	}
	if err != nil {
		return nil, err
	}
	return &margin.PositionChangeResponse{
		Exchange:        e.Name,
		Pair:            req.Pair,
		Asset:           req.Asset,
		MarginType:      req.MarginType,
		AllocatedMargin: req.NewAllocatedMargin,
	}, nil
}

// marginTypeToString converts the GCT margin type to Binance's string
func (e *Exchange) marginTypeToString(mt margin.Type) (string, error) {
	switch mt {
	case margin.Isolated:
		return margin.Isolated.Upper(), nil
	case margin.Multi:
		return "CROSSED", nil
	}
	return "", fmt.Errorf("%w %v", margin.ErrInvalidMarginType, mt)
}

// positionSideFor returns the position side of a direction: LONG or SHORT, Hedge Mode's positions, or BOTH, the
// one-way position, without a direction
func positionSideFor(direction order.Side) string {
	switch {
	case direction.IsLong():
		return "LONG"
	case direction.IsShort():
		return "SHORT"
	default:
		return "BOTH"
	}
}

// GetFuturesPositionSummary returns a USDⓈ-M or COIN-M position's summary. In Hedge Mode the request's Direction
// selects the long or the short position; without one the summary is of the one-way position. A COIN-M position's
// collateral is its margin asset: the underlying pair's base when the request has one, and the symbol's margin asset
// from exchange information otherwise
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if req == nil {
		return nil, fmt.Errorf("%w GetFuturesPositionSummary", common.ErrNilPointer)
	}
	if req.CalculateOffline {
		return nil, common.ErrCannotCalculateOffline
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch req.Asset {
	case asset.USDTMarginedFutures:
		return e.uPositionSummary(ctx, req)
	case asset.CoinMarginedFutures:
		return e.cPositionSummary(ctx, req)
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
}

// uPositionSummary returns a USDⓈ-M position's summary from the position information, with the position's leverage,
// margin type and collateral from the account information. Multi-Assets Mode collateral is the account's, valued in
// USD
func (e *Exchange) uPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	symbol, err := e.FormatSymbol(req.Pair, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	positionSide := positionSideFor(req.Direction)
	account, err := e.UAccountInformationV2(ctx)
	if err != nil {
		return nil, err
	}
	var accountPosition *UPosition
	for i := range account.Positions {
		if account.Positions[i].Symbol == symbol && account.Positions[i].PositionSide == positionSide {
			accountPosition = &account.Positions[i]
			break
		}
	}
	if accountPosition == nil {
		return nil, fmt.Errorf("%w: %s %s %s account position", futures.ErrNoPositionsFound, req.Asset, req.Pair, positionSide)
	}
	positions, err := e.UPositionsInfoV3(ctx, req.Pair)
	if err != nil {
		return nil, err
	}
	var position *UPositionInformationV3
	for i := range positions {
		if positions[i].Symbol == symbol && positions[i].PositionSide == positionSide {
			position = &positions[i]
			break
		}
	}
	if position == nil {
		return nil, fmt.Errorf("%w: %s %s %s position information", futures.ErrNoPositionsFound, req.Asset, req.Pair, positionSide)
	}
	summary := &futures.PositionSummary{
		Pair:                         req.Pair,
		Asset:                        req.Asset,
		MarginType:                   marginAccountType(accountPosition.Isolated),
		CollateralMode:               collateral.SingleMode,
		Currency:                     position.MarginAsset,
		NotionalSize:                 position.Notional.Decimal(),
		Leverage:                     accountPosition.Leverage.Decimal(),
		MaintenanceMarginRequirement: position.MaintenanceMargin.Decimal(),
		InitialMarginRequirement:     position.PositionInitialMargin.Decimal(),
		EstimatedLiquidationPrice:    position.LiquidationPrice.Decimal(),
		CollateralUsed:               position.PositionInitialMargin.Decimal(),
		MarkPrice:                    position.MarkPrice.Decimal(),
		CurrentSize:                  position.PositionAmount.Decimal(),
		ContractSettlementType:       futures.Linear,
		AverageOpenPrice:             position.EntryPrice.Decimal(),
		UnrealisedPNL:                position.UnrealizedProfit.Decimal(),
	}
	if accountPosition.Isolated {
		summary.IsolatedMargin = position.IsolatedWallet.Decimal()
		summary.CollateralUsed = summary.IsolatedMargin
	}
	if account.MultiAssetsMargin {
		summary.CollateralMode = collateral.MultiMode
		summary.TotalCollateral = account.TotalWalletBalance.Decimal()
		summary.FreeCollateral = account.AvailableBalance.Decimal()
	} else {
		for i := range account.Assets {
			if account.Assets[i].Asset.Equal(position.MarginAsset) {
				summary.TotalCollateral = account.Assets[i].WalletBalance.Decimal()
				summary.FreeCollateral = account.Assets[i].AvailableBalance.Decimal()
				break
			}
		}
	}
	setCollateralFractions(summary)
	return summary, nil
}

// cPositionSummary returns a COIN-M position's summary from the position information, with the position's margin
// requirements, margin type and collateral from the account information
func (e *Exchange) cPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	symbol, err := e.FormatSymbol(req.Pair, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	positionSide := positionSideFor(req.Direction)
	marginAsset := req.UnderlyingPair.Base
	if marginAsset.IsEmpty() {
		var info *CFuturesExchangeInfoResponse
		if info, err = e.FuturesExchangeInfo(ctx); err != nil {
			return nil, err
		}
		for i := range info.Symbols {
			if info.Symbols[i].Symbol == symbol {
				marginAsset = info.Symbols[i].MarginAsset
				break
			}
		}
		if marginAsset.IsEmpty() {
			return nil, fmt.Errorf("%w: %s margin asset", currency.ErrCurrencyNotFound, symbol)
		}
	}
	account, err := e.GetFuturesAccountInfo(ctx)
	if err != nil {
		return nil, err
	}
	var accountPosition *FuturesAccountInformationPosition
	for i := range account.Positions {
		if account.Positions[i].Symbol == symbol && account.Positions[i].PositionSide == positionSide {
			accountPosition = &account.Positions[i]
			break
		}
	}
	if accountPosition == nil {
		return nil, fmt.Errorf("%w: %s %s %s account position", futures.ErrNoPositionsFound, req.Asset, req.Pair, positionSide)
	}
	var accountAsset *FuturesAccountAsset
	for i := range account.Assets {
		if account.Assets[i].Asset.Equal(marginAsset) {
			accountAsset = &account.Assets[i]
			break
		}
	}
	if accountAsset == nil {
		return nil, fmt.Errorf("%w: %s collateral", currency.ErrCurrencyNotFound, marginAsset)
	}
	// The contract pair, such as BTCUSD, is what the position information takes, which is the GCT pair's base
	positions, err := e.FuturesPositionsInfo(ctx, currency.EMPTYCODE, req.Pair.Base)
	if err != nil {
		return nil, err
	}
	var position *FuturesPositionInformation
	for i := range positions {
		if positions[i].Symbol == symbol && positions[i].PositionSide == positionSide {
			position = &positions[i]
			break
		}
	}
	if position == nil {
		return nil, fmt.Errorf("%w: %s %s %s position information", futures.ErrNoPositionsFound, req.Asset, req.Pair, positionSide)
	}
	summary := &futures.PositionSummary{
		Pair:                         req.Pair,
		Asset:                        req.Asset,
		MarginType:                   marginAccountType(accountPosition.Isolated),
		CollateralMode:               collateral.SingleMode,
		Currency:                     marginAsset,
		NotionalSize:                 position.NotionalValue.Decimal(),
		Leverage:                     position.Leverage.Decimal(),
		MaintenanceMarginRequirement: accountPosition.MaintenanceMargin.Decimal(),
		InitialMarginRequirement:     accountPosition.PositionInitialMargin.Decimal(),
		EstimatedLiquidationPrice:    position.LiquidationPrice.Decimal(),
		CollateralUsed:               accountPosition.PositionInitialMargin.Decimal(),
		MarkPrice:                    position.MarkPrice.Decimal(),
		CurrentSize:                  position.PositionAmount.Decimal(),
		ContractSettlementType:       futures.Inverse,
		AverageOpenPrice:             position.EntryPrice.Decimal(),
		UnrealisedPNL:                position.UnrealizedProfit.Decimal(),
		FreeCollateral:               accountAsset.AvailableBalance.Decimal(),
		TotalCollateral:              accountAsset.WalletBalance.Decimal(),
	}
	if accountPosition.Isolated {
		summary.IsolatedMargin = position.IsolatedWallet.Decimal()
		summary.CollateralUsed = summary.IsolatedMargin
	}
	setCollateralFractions(summary)
	return summary, nil
}

// setCollateralFractions sets the collateral a position summary's account holds back and the share of it the
// position's maintenance margin takes, in percent
func setCollateralFractions(summary *futures.PositionSummary) {
	summary.FrozenBalance = summary.TotalCollateral.Sub(summary.FreeCollateral)
	if !summary.TotalCollateral.IsZero() {
		summary.MaintenanceMarginFraction = summary.MaintenanceMarginRequirement.Div(summary.TotalCollateral).Mul(decimal.NewFromInt(100))
	}
}

// futuresOrdersWindow is the longest time window the USDⓈ-M and COIN-M All Orders endpoints take, which must be
// shorter than seven days
const futuresOrdersWindow = 7*24*time.Hour - time.Millisecond

// futuresOrdersInWindow collects every order from start to end through fetch, which returns at most limit orders. A
// window longer than the endpoints take is split first, and a full reply, which may leave orders out, has its window
// halved until each part's orders fit. Windows are whole milliseconds, as Binance's times are, so the parts of a
// window cover it without gaps
func futuresOrdersInWindow[T any](start, end time.Time, limit int, fetch func(start, end time.Time) ([]T, error)) ([]T, error) {
	start, end = start.Truncate(time.Millisecond), end.Truncate(time.Millisecond)
	if end.Sub(start) > futuresOrdersWindow {
		first, err := futuresOrdersInWindow(start, start.Add(futuresOrdersWindow), limit, fetch)
		if err != nil {
			return nil, err
		}
		rest, err := futuresOrdersInWindow(start.Add(futuresOrdersWindow+time.Millisecond), end, limit, fetch)
		if err != nil {
			return nil, err
		}
		return append(first, rest...), nil
	}
	orders, err := fetch(start, end)
	if err != nil {
		return nil, err
	}
	if len(orders) < limit || end.Sub(start) < 2*time.Millisecond {
		return orders, nil
	}
	middle := start.Add(end.Sub(start) / 2).Truncate(time.Millisecond)
	first, err := futuresOrdersInWindow(start, middle, limit, fetch)
	if err != nil {
		return nil, err
	}
	rest, err := futuresOrdersInWindow(middle.Add(time.Millisecond), end, limit, fetch)
	if err != nil {
		return nil, err
	}
	return append(first, rest...), nil
}

// GetFuturesPositionOrders returns the orders of each pair's position from the request's start, at most the
// exchange's maximum order history ago, to its end, now when unset. Every order of the window is collected (see
// futuresOrdersInWindow), each with the position's leverage and margin type
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w GetFuturesPositionOrders", common.ErrNilPointer)
	}
	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	var settlementType futures.ContractSettlementType
	switch req.Asset {
	case asset.USDTMarginedFutures:
		settlementType = futures.Linear
	case asset.CoinMarginedFutures:
		settlementType = futures.Inverse
	default:
		return nil, fmt.Errorf("%w: futures position orders for %s", asset.ErrNotSupported, req.Asset)
	}
	now := time.Now()
	start, end := req.StartDate, req.EndDate
	if end.IsZero() {
		end = now
	}
	if time.Since(start) > e.Features.Supports.MaximumOrderHistory+time.Hour {
		earliest := now.Add(-e.Features.Supports.MaximumOrderHistory)
		if !req.RespectOrderHistoryLimits {
			return nil, fmt.Errorf("%w max lookup %v", futures.ErrOrderHistoryTooLarge, earliest)
		}
		start = earliest
	}
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	resp := make([]futures.PositionResponse, len(req.Pairs))
	for i, p := range req.Pairs {
		var orders []order.Detail
		var err error
		if req.Asset == asset.USDTMarginedFutures {
			orders, err = e.uPositionOrders(ctx, p, start, end)
		} else {
			orders, err = e.cPositionOrders(ctx, p, start, end)
		}
		if err != nil {
			return nil, err
		}
		resp[i] = futures.PositionResponse{
			Pair:                   p,
			Asset:                  req.Asset,
			ContractSettlementType: settlementType,
			Orders:                 orders,
		}
	}
	return resp, nil
}

// positionMarginType converts a position's margin type for its orders, leaving one it cannot convert unset with a
// logged warning rather than failing the orders
func (e *Exchange) positionMarginType(a asset.Item, pair currency.Pair, marginType string) margin.Type {
	mt, err := margin.StringToMarginType(marginType)
	if err != nil {
		log.Warnf(log.ExchangeSys, "%s %s %s position: %v", e.Name, a, pair, err)
	}
	return mt
}

// uPositionOrders returns a USDⓈ-M pair's orders from start to end, with its position's leverage and margin type
func (e *Exchange) uPositionOrders(ctx context.Context, pair currency.Pair, start, end time.Time) ([]order.Detail, error) {
	positions, err := e.UPositionsInfoV2(ctx, pair)
	if err != nil {
		return nil, err
	}
	var leverage float64
	var marginType margin.Type
	if len(positions) > 0 {
		// Binance applies one leverage and margin type to every position of a symbol
		leverage = positions[0].Leverage.Float64()
		marginType = e.positionMarginType(asset.USDTMarginedFutures, pair, positions[0].MarginType)
	}
	const limit = 1000 // the most Binance returns a request
	history, err := futuresOrdersInWindow(start, end, limit, func(windowStart, windowEnd time.Time) ([]UFuturesOrderData, error) {
		return e.UAllAccountOrders(ctx, &UAllOrdersRequest{Symbol: pair, StartTime: windowStart, EndTime: windowEnd, Limit: limit})
	})
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, 0, len(history))
	for i := range history {
		o := &history[i]
		orders = e.appendListedOrder(orders, asset.USDTMarginedFutures, o.OrderID, pair, o.Symbol, func(p currency.Pair) (order.Detail, error) {
			d, convertErr := e.uOrderDetail(&o.UOrderResponse, p)
			d.Leverage = leverage
			d.MarginType = marginType
			return d, convertErr
		})
	}
	return orders, nil
}

// cPositionOrders returns a COIN-M pair's orders from start to end, with its position's leverage and margin type
func (e *Exchange) cPositionOrders(ctx context.Context, pair currency.Pair, start, end time.Time) ([]order.Detail, error) {
	symbol, err := e.FormatSymbol(pair, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	// The contract pair, such as BTCUSD, is what the position information takes, which is the GCT pair's base
	positions, err := e.FuturesPositionsInfo(ctx, currency.EMPTYCODE, pair.Base)
	if err != nil {
		return nil, err
	}
	var leverage float64
	var marginType margin.Type
	for i := range positions {
		if positions[i].Symbol != symbol {
			continue
		}
		leverage = positions[i].Leverage.Float64()
		marginType = e.positionMarginType(asset.CoinMarginedFutures, pair, positions[i].MarginType)
		break
	}
	const limit = 100 // the most Binance returns a request
	history, err := futuresOrdersInWindow(start, end, limit, func(windowStart, windowEnd time.Time) ([]FuturesOrderData, error) {
		return e.GetAllFuturesOrders(ctx, &CFuturesAllOrdersRequest{Symbol: pair, StartTime: windowStart, EndTime: windowEnd, Limit: limit})
	})
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, 0, len(history))
	for i := range history {
		o := &history[i]
		orders = e.appendListedOrder(orders, asset.CoinMarginedFutures, o.OrderID, pair, o.Symbol, func(p currency.Pair) (order.Detail, error) {
			d, convertErr := e.cOrderDetail(&o.FuturesOrderDetailResponse, p)
			d.Leverage = leverage
			d.MarginType = marginType
			return d, convertErr
		})
	}
	return orders, nil
}

// SetLeverage sets a USDⓈ-M or COIN-M symbol's initial leverage, or the cross margin account's maximum leverage (3, 5
// or 10), which Binance takes as a whole number; isolated margin leverage follows Binance's fixed tiers
func (e *Exchange) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, amount float64, _ order.Side) error {
	switch item {
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
	case asset.Margin:
		if marginType != margin.Multi && marginType != margin.Unset {
			return fmt.Errorf("%w: %s leverage on %s", asset.ErrNotSupported, marginType, item)
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	if amount < 1 || amount != math.Trunc(amount) {
		return fmt.Errorf("%w: %v", errInvalidLeverage, amount)
	}
	var err error
	switch item {
	case asset.USDTMarginedFutures:
		_, err = e.UChangeInitialLeverageRequest(ctx, pair, uint64(amount))
	case asset.CoinMarginedFutures:
		_, err = e.FuturesChangeInitialLeverage(ctx, pair, uint64(amount))
	default:
		_, err = e.AdjustCrossMarginMaxLeverage(ctx, uint64(amount))
	}
	return err
}

// GetLeverage returns a USDⓈ-M or COIN-M symbol's initial leverage. Binance has no endpoint returning the cross margin
// maximum leverage SetLeverage sets
func (e *Exchange) GetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, _ order.Side) (float64, error) {
	if pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	switch item {
	case asset.USDTMarginedFutures:
		positions, err := e.UPositionsInfoV2(ctx, pair)
		if err != nil {
			return 0, err
		}
		if len(positions) == 0 {
			return 0, fmt.Errorf("%w %v %v", futures.ErrPositionNotFound, item, pair)
		}
		// Binance applies one leverage to every position of a symbol
		return positions[0].Leverage.Float64(), nil
	case asset.CoinMarginedFutures:
		symbol, err := e.FormatSymbol(pair, item)
		if err != nil {
			return 0, err
		}
		// The contract pair, such as BTCUSD, is what the position information takes, which is the GCT pair's base;
		// it returns the positions of each of the pair's contracts
		positions, err := e.FuturesPositionsInfo(ctx, currency.EMPTYCODE, pair.Base)
		if err != nil {
			return 0, err
		}
		for i := range positions {
			if positions[i].Symbol == symbol {
				return positions[i].Leverage.Float64(), nil
			}
		}
		return 0, fmt.Errorf("%w %v %v", futures.ErrPositionNotFound, item, pair)
	default:
		return 0, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if item != asset.USDTMarginedFutures && item != asset.CoinMarginedFutures {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, item)
	}
	format, err := e.GetPairFormat(item, false)
	if err != nil {
		return nil, err
	}
	var resp []futures.Contract
	if item == asset.USDTMarginedFutures {
		info, err := e.UGetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		fundingRateLimits := make(map[string]*FundingRateInfoResponse, len(info))
		for i := range info {
			fundingRateLimits[info[i].Symbol] = &info[i]
		}
		ei, err := e.UExchangeInfo(ctx)
		if err != nil {
			return nil, err
		}
		resp = make([]futures.Contract, 0, len(ei.Symbols))
		for i := range ei.Symbols {
			s := &ei.Symbols[i]
			c := futures.Contract{
				Exchange:           e.Name,
				Name:               uFuturesSymbolPair(s.Symbol, s.BaseAsset, s.QuoteAsset).Format(format),
				Underlying:         currency.NewPair(s.BaseAsset, s.QuoteAsset),
				Asset:              item,
				StartDate:          s.OnboardDate.Time(),
				EndDate:            contractEndDate(s.ContractType, s.DeliveryDate),
				IsActive:           s.Status == symbolStatusTrading,
				Status:             s.Status,
				Type:               e.futuresContractType(s.ContractType, s.Symbol),
				SettlementType:     futures.Linear,
				SettlementCurrency: s.MarginAsset,
				MarginCurrency:     s.MarginAsset,
			}
			if l, ok := fundingRateLimits[s.Symbol]; ok {
				c.FundingRateFloor = l.AdjustedFundingRateFloor.Decimal()
				c.FundingRateCeiling = l.AdjustedFundingRateCap.Decimal()
			}
			resp = append(resp, c)
		}
		return resp, nil
	}
	info, err := e.GetFundingRateInfo(ctx)
	if err != nil {
		return nil, err
	}
	fundingRateLimits := make(map[string]*CFuturesFundingRateInfo, len(info))
	for i := range info {
		fundingRateLimits[info[i].Symbol] = &info[i]
	}
	ei, err := e.FuturesExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}
	resp = make([]futures.Contract, 0, len(ei.Symbols))
	for i := range ei.Symbols {
		s := &ei.Symbols[i]
		name, err := currency.NewPairDelimiter(s.Symbol, currency.UnderscoreDelimiter)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s skipping %s contract: %s", e.Name, item, err)
			continue
		}
		// A contract is worth its contract size in the quote currency, which the multiplier holds
		c := futures.Contract{
			Exchange:           e.Name,
			Name:               name.Format(format),
			Underlying:         currency.NewPair(s.BaseAsset, s.QuoteAsset),
			Asset:              item,
			StartDate:          s.OnboardDate.Time(),
			EndDate:            contractEndDate(s.ContractType, s.DeliveryDate),
			IsActive:           s.ContractStatus == symbolStatusTrading,
			Status:             s.ContractStatus,
			Type:               e.futuresContractType(s.ContractType, s.Symbol),
			SettlementType:     futures.Inverse,
			SettlementCurrency: s.MarginAsset,
			MarginCurrency:     s.MarginAsset,
			Multiplier:         float64(s.ContractSize),
		}
		if l, ok := fundingRateLimits[s.Symbol]; ok {
			c.FundingRateFloor = l.AdjustedFundingRateFloor.Decimal()
			c.FundingRateCeiling = l.AdjustedFundingRateCap.Decimal()
		}
		resp = append(resp, c)
	}
	return resp, nil
}

// futuresContractType converts a USDⓈ-M or COIN-M contract type; an undocumented one is logged and reported as unknown
func (e *Exchange) futuresContractType(contractType, symbol string) futures.ContractType {
	switch contractType {
	case "PERPETUAL", "TRADIFI_PERPETUAL", "PERPETUAL_DELIVERING":
		return futures.Perpetual
	case "CURRENT_MONTH", "NEXT_MONTH":
		return futures.Monthly
	case "CURRENT_QUARTER", "NEXT_QUARTER":
		return futures.Quarterly
	default:
		log.Warnf(log.ExchangeSys, "%s contract %s has unknown contract type %q", e.Name, symbol, contractType)
		return futures.Unknown
	}
}

// contractEndDate returns the delivery date of a contract that has one. Perpetuals that are not being delivered carry
// a placeholder date in 2100
func contractEndDate(contractType string, deliveryDate types.Time) time.Time {
	if contractType == "PERPETUAL" || contractType == "TRADIFI_PERPETUAL" {
		return time.Time{}
	}
	return deliveryDate.Time()
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	if len(k) == 0 {
		return nil, fmt.Errorf("%w: open interest is only available by pair", common.ErrFunctionNotSupported)
	}
	for i := range k {
		if k[i].Asset != asset.USDTMarginedFutures && k[i].Asset != asset.CoinMarginedFutures {
			// Checked before any request, so no request is made for a list that cannot be answered in full
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	resp := make([]futures.OpenInterest, len(k))
	for i := range k {
		pair := k[i].Pair()
		var openInterest float64
		if k[i].Asset == asset.USDTMarginedFutures {
			oi, err := e.UOpenInterest(ctx, pair)
			if err != nil {
				return nil, err
			}
			openInterest = oi.OpenInterest.Float64()
		} else {
			// COIN-M open interest counts contracts
			oi, err := e.OpenInterest(ctx, pair)
			if err != nil {
				return nil, err
			}
			openInterest = oi.OpenInterest.Float64()
		}
		resp[i] = futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, k[i].Asset, pair),
			OpenInterest: openInterest,
		}
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(ctx context.Context, a asset.Item, cp currency.Pair) (string, error) {
	if cp.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	switch a {
	case asset.Spot, asset.Margin, asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options:
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	symbol, err := e.FormatSymbol(cp, a)
	if err != nil {
		return "", err
	}
	switch a {
	case asset.Spot:
		return tradeBaseURL + "trade/" + symbol + "?type=spot", nil
	case asset.Margin:
		return tradeBaseURL + "trade/" + symbol + "?type=cross", nil
	case asset.USDTMarginedFutures:
		if !isDeliveryDate(cp.Quote) {
			return tradeBaseURL + "futures/" + symbol, nil
		}
		// Delivery contracts trade on pages named after the contract pair and quarter
		info, err := e.UExchangeInfo(ctx)
		if err != nil {
			return "", err
		}
		for i := range info.Symbols {
			if info.Symbols[i].Symbol == symbol {
				return tradeBaseURL + "futures/" + info.Symbols[i].Pair + deliveryTradePageSuffix(info.Symbols[i].ContractType), nil
			}
		}
	case asset.CoinMarginedFutures:
		if cp.Quote.Equal(currency.PERP) {
			return tradeBaseURL + "delivery/" + cp.Base.Upper().String(), nil
		}
		info, err := e.FuturesExchangeInfo(ctx)
		if err != nil {
			return "", err
		}
		for i := range info.Symbols {
			if info.Symbols[i].Symbol == symbol {
				return tradeBaseURL + "delivery/" + info.Symbols[i].Pair + deliveryTradePageSuffix(info.Symbols[i].ContractType), nil
			}
		}
	case asset.Options:
		info, err := e.GetOptionsExchangeInformation(ctx)
		if err != nil {
			return "", err
		}
		for i := range info.OptionSymbols {
			if info.OptionSymbols[i].Symbol == symbol {
				return tradeBaseURL + "eoptions/" + info.OptionSymbols[i].Underlying + "/" + symbol, nil
			}
		}
	}
	return "", fmt.Errorf("%w: %s %s", currency.ErrPairNotFound, a, symbol)
}

// deliveryTradePageSuffix returns the suffix of the trade page of a delivery contract's type
func deliveryTradePageSuffix(contractType string) string {
	switch contractType {
	case "CURRENT_QUARTER":
		return "_QUARTER"
	case "NEXT_QUARTER":
		return "_BI-QUARTER"
	default:
		return ""
	}
}
