package gateio

import (
	"context"
	"encoding/hex"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// SetDefaults sets default values for the exchange
func (e *Exchange) SetDefaults() {
	e.Name = "GateIO"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	if err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.Margin, asset.CrossMargin, asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.DeliveryFutures, asset.Options); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Features = exchange.Features{
		CurrencyTranslations: currency.NewTranslations(map[currency.Code]currency.Code{
			currency.NewCode("MBABYDOGE"): currency.BABYDOGE,
		}),
		TradingRequirements: protocol.TradingRequirements{
			SpotMarketBuyQuotation: true,
			SpotMarketSellBase:     true,
		},
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:        true,
				TickerFetching:        true,
				KlineFetching:         true,
				TradeFetching:         true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
				PredictedFundingRate:  true,
				FundingRateFetching:   true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				TradeFetching:          true,
				KlineFetching:          true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				GetOrder:               true,
				AccountBalance:         true,
				Subscribe:              true,
				Unsubscribe:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.FourHour:  true,
					kline.EightHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.USDTMarginedFutures: true,
					asset.CoinMarginedFutures: true,
				},
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
					kline.IntervalCapacity{Interval: kline.HundredMilliseconds},
					kline.IntervalCapacity{Interval: kline.ThousandMilliseconds},
					kline.IntervalCapacity{Interval: kline.TenSecond},
					kline.IntervalCapacity{Interval: kline.ThirtySecond},
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.EightHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
					kline.IntervalCapacity{Interval: kline.ThreeMonth},
					kline.IntervalCapacity{Interval: kline.SixMonth},
				),
				GlobalResultLimit: 1000,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}
	var err error
	e.Requester, err = request.New(
		e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(packageRateLimits),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	// The margin and cross margin balance channels save under the websocket user ID, which REST saves as "", so their
	// balances would be counted twice
	if err := e.DisableAssetWebsocketSupport(asset.Margin); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	if err := e.DisableAssetWebsocketSupport(asset.CrossMargin); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              gateioTradeURL,
		exchange.RestFutures:           gateioFuturesLiveTradingAlternative,
		exchange.RestSpotSupplementary: gateioFuturesTestnetTrading,
		exchange.WebsocketSpot:         gateioWebsocketEndpoint,
		exchange.WebsocketUSDTMargined: usdtFuturesWebsocketURL,
		exchange.WebsocketCoinMargined: btcFuturesWebsocketURL,
		exchange.WebsocketFutures:      deliveryRealUSDTTradingURL,
		exchange.WebsocketOptions:      optionsWebsocketURL,
		exchange.EdgeCase1:             frontEndURL,
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.wsOBResubMgr = newWSOBResubManager()
	e.wsOBUpdateMgr = orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
		FetchDelay:         orderbookmanager.DefaultWSOrderbookUpdateTimeDelay,
		FetchDeadline:      orderbookmanager.DefaultWSOrderbookUpdateDeadline,
		FetchOrderbook:     e.fetchWSOrderbookSnapshot,
		CheckPendingUpdate: checkPendingUpdate,
		Orderbook:          &e.Websocket.Orderbook,
	})
}

// Bootstrap caches futures account user IDs before websocket subscriptions are generated.
func (e *Exchange) Bootstrap(ctx context.Context) (continueBootstrap bool, err error) {
	e.prepareFuturesUserIDs(ctx)
	return true, nil
}

// Setup sets user configuration
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
		PreConnect:                   e.prepareFuturesUserIDs,
		FillsFeed:                    e.Features.Enabled.FillsFeed,
		TradeFeed:                    e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement: true,
		RateLimitDefinitions:         packageRateLimits,
	}); err != nil {
		return err
	}
	// Spot connection
	wsSpotURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsSpotURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Handler:               e.WebsocketHandleSpotData,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptionsSpot,
		Connector:             e.WebsocketConnectSpot,
		Authenticate:          e.authenticateSpot,
		MessageFilter:         asset.Spot,
	}); err != nil {
		return err
	}
	// Futures connection - USDT margined
	wsUSDTMarginedURL, err := e.API.Endpoints.GetURL(exchange.WebsocketUSDTMargined)
	if err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  wsUSDTMarginedURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, conn websocket.Connection, incoming []byte) error {
			return e.WebsocketHandleFuturesData(ctx, conn, incoming, asset.USDTMarginedFutures)
		},
		Subscriber:   e.FuturesSubscribe,
		Unsubscriber: e.FuturesUnsubscribe,
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.GenerateFuturesDefaultSubscriptions(context.TODO(), asset.USDTMarginedFutures)
		},
		Connector: func(ctx context.Context, conn websocket.Connection) error {
			return e.WebsocketFuturesConnect(ctx, conn, asset.USDTMarginedFutures)
		},
		Authenticate:  e.authenticateFutures,
		MessageFilter: asset.USDTMarginedFutures,
	}); err != nil {
		return err
	}

	// Futures connection - BTC margined
	wsCoinMarginedURL, err := e.API.Endpoints.GetURL(exchange.WebsocketCoinMargined)
	if err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  wsCoinMarginedURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, conn websocket.Connection, incoming []byte) error {
			return e.WebsocketHandleFuturesData(ctx, conn, incoming, asset.CoinMarginedFutures)
		},
		Subscriber:   e.FuturesSubscribe,
		Unsubscriber: e.FuturesUnsubscribe,
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.GenerateFuturesDefaultSubscriptions(context.TODO(), asset.CoinMarginedFutures)
		},
		Connector: func(ctx context.Context, conn websocket.Connection) error {
			return e.WebsocketFuturesConnect(ctx, conn, asset.CoinMarginedFutures)
		},
		MessageFilter: asset.CoinMarginedFutures,
	}); err != nil {
		return err
	}

	// Futures connection - Delivery - USDT margined
	wsDeliveryFuturesURL, err := e.API.Endpoints.GetURL(exchange.WebsocketFutures)
	if err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  wsDeliveryFuturesURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, conn websocket.Connection, incoming []byte) error {
			return e.WebsocketHandleFuturesData(ctx, conn, incoming, asset.DeliveryFutures)
		},
		Subscriber:   e.DeliveryFuturesSubscribe,
		Unsubscriber: e.DeliveryFuturesUnsubscribe,
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.GenerateDeliveryFuturesDefaultSubscriptions(asset.DeliveryFutures)
		},
		Connector: func(ctx context.Context, conn websocket.Connection) error {
			return e.WebsocketDeliveryFuturesConnect(ctx, conn, asset.DeliveryFutures)
		},
		MessageFilter: asset.DeliveryFutures,
	}); err != nil {
		return err
	}

	// Futures connection - Options
	wsOptionsURL, err := e.API.Endpoints.GetURL(exchange.WebsocketOptions)
	if err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsOptionsURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Handler:               e.WebsocketHandleOptionsData,
		Subscriber:            e.OptionsSubscribe,
		Unsubscriber:          e.OptionsUnsubscribe,
		GenerateSubscriptions: e.GenerateOptionsDefaultSubscriptions,
		Connector:             e.WebsocketOptionsConnect,
		MessageFilter:         asset.Options,
	})
}

// futuresBaseVolume returns a futures ticker's 24h volume in the base currency. Gate truncates
// both of the figures it publishes, so each is a lower bound and the larger is the one that lost
// less: volume_24h_base drops the fraction of a base unit, which rounds the thinner contracts away
// entirely and some to zero against a live contract count, while volume_24h drops the fraction of
// a contract, which costs a whole lot on the contracts Gate lists with enable_decimal. Delivery
// futures omit quanto_multiplier and coin-margined reports 0, which zeroes the product and leaves
// the explicit field standing on its own, as it would too if Gate stopped serving the multiplier
// here: the live ticker carries it, but Gate documents it only on the contract
func futuresBaseVolume(t *FuturesTickerResponse) float64 {
	return max(t.Volume24Hour.Float64()*t.QuantoMultiplier.Float64(), t.Volume24HourBase.Float64())
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	fPair, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	fPair = fPair.Upper()
	var tickerData *ticker.Price
	switch a {
	case asset.Margin, asset.Spot, asset.CrossMargin:
		available, err := e.checkInstrumentAvailabilityInSpot(fPair)
		if err != nil {
			return nil, err
		}
		if a != asset.Spot && !available {
			return nil, fmt.Errorf("%v instrument %v does not have ticker data", a, fPair)
		}
		tickerNew, err := e.GetTicker(ctx, fPair, "")
		if err != nil {
			return nil, err
		}
		tickerData = &ticker.Price{
			Pair:         fPair,
			Low:          tickerNew.Low24Hour.Float64(),
			High:         tickerNew.High24Hour.Float64(),
			Bid:          tickerNew.HighestBid.Float64(),
			Ask:          tickerNew.LowestAsk.Float64(),
			Last:         tickerNew.Last.Float64(),
			BaseVolume:   tickerNew.BaseVolume.Float64(),
			QuoteVolume:  tickerNew.QuoteVolume.Float64(),
			ExchangeName: e.Name,
			AssetType:    a,
		}
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(fPair, a)
		if err != nil {
			return nil, err
		}
		var tickers []*FuturesTickerResponse
		if a == asset.DeliveryFutures {
			tickers, err = e.GetDeliveryFutureTickers(ctx, settle, fPair)
		} else {
			tickers, err = e.GetFuturesTickers(ctx, settle, fPair)
		}
		if err != nil {
			return nil, err
		}
		if len(tickers) != 1 {
			return nil, errNoTickerData
		}
		tickerData = &ticker.Price{
			Pair:         fPair,
			Low:          tickers[0].Low24Hour.Float64(),
			High:         tickers[0].High24Hour.Float64(),
			Last:         tickers[0].Last.Float64(),
			BaseVolume:   futuresBaseVolume(tickers[0]),
			QuoteVolume:  tickers[0].Volume24HourQuote.Float64(),
			MarkPrice:    tickers[0].MarkPrice.Float64(),
			IndexPrice:   tickers[0].IndexPrice.Float64(),
			ExchangeName: e.Name,
			AssetType:    a,
		}
	case asset.Options:
		underlying, err := e.GetUnderlyingFromCurrencyPair(fPair)
		if err != nil {
			return nil, err
		}
		tickers, err := e.GetOptionsTickers(ctx, underlying.String())
		if err != nil {
			return nil, err
		}
		for _, tkr := range tickers {
			if !tkr.Name.Equal(fPair) {
				continue
			}
			cleanQuote := strings.ReplaceAll(tkr.Name.Quote.String(), currency.UnderscoreDelimiter, currency.DashDelimiter)
			tkr.Name.Quote = currency.NewCode(cleanQuote)
			tickerData = &ticker.Price{
				Pair:         tkr.Name,
				Last:         tkr.LastPrice.Float64(),
				MarkPrice:    tkr.MarkPrice.Float64(),
				IndexPrice:   tkr.IndexPrice.Float64(),
				Bid:          tkr.Bid1Price.Float64(),
				Ask:          tkr.Ask1Price.Float64(),
				AskSize:      tkr.Ask1Size.Float64(),
				BidSize:      tkr.Bid1Size.Float64(),
				ExchangeName: e.Name,
				AssetType:    a,
			}
			break
		}
		if tickerData == nil {
			return nil, fmt.Errorf("%w for %s %s", ticker.ErrTickerNotFound, a, fPair)
		}
	}
	if err := ticker.ProcessTicker(tickerData); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, fPair, a)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	switch a {
	case asset.Spot:
		tradables, err := e.ListSpotCurrencyPairs(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(tradables))
		for _, cpDetail := range tradables {
			if cpDetail.TradeStatus == "untradable" {
				continue
			}
			pairs = append(pairs, currency.NewPair(cpDetail.Base, cpDetail.Quote))
		}
		return pairs, nil
	case asset.Margin:
		tradables, err := e.GetIsolatedMarginLendingMarkets(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(tradables))
		now := time.Now()
		for x := range tradables {
			if !tradables[x].IsTradable(now) {
				continue
			}
			pairs = append(pairs, tradables[x].Pair)
		}
		return pairs, nil
	case asset.CrossMargin:
		crossMinimums, err := e.getCrossMarginMinimums(ctx)
		if err != nil {
			return nil, err
		}
		tradables, err := e.ListSpotCurrencyPairs(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(tradables))
		for i := range tradables {
			if tradables[i].TradeStatus == "untradable" {
				continue
			}
			_, baseOK := crossMinimums[tradables[i].Base]
			_, quoteOK := crossMinimums[tradables[i].Quote]
			if baseOK && quoteOK {
				pairs = append(pairs, tradables[i].ID)
			}
		}
		return pairs, nil
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
		if err != nil {
			return nil, err
		}
		contracts, err := e.GetAllFutureContracts(ctx, settle, 0, 0)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(contracts))
		for _, fContract := range contracts {
			if fContract.Name.IsEmpty() || (fContract.Type == "inverse" && a != asset.CoinMarginedFutures) || (fContract.Type == "direct" && a != asset.USDTMarginedFutures) {
				continue
			}
			pairs = append(pairs, fContract.Name)
		}
		return slices.Clip(pairs), nil
	case asset.DeliveryFutures:
		contracts, err := e.GetAllDeliveryContracts(ctx, currency.USDT)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(contracts))
		for _, deliveryContract := range contracts {
			if deliveryContract.InDelisting || deliveryContract.Name == "" {
				continue
			}
			cp, err := currency.NewPairFromString(strings.ToUpper(deliveryContract.Name))
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return slices.Clip(pairs), nil
	case asset.Options:
		underlyings, err := e.GetAllOptionsUnderlyings(ctx)
		if err != nil {
			return nil, err
		}
		var pairs []currency.Pair
		for _, optionUly := range underlyings {
			contracts, err := e.GetAllContractOfUnderlyingWithinExpiryDate(ctx, optionUly.Name, time.Time{})
			if err != nil {
				return nil, err
			}
			for _, oContract := range contracts {
				cp, err := currency.NewPairFromString(strings.ReplaceAll(oContract.Name, currency.DashDelimiter, currency.UnderscoreDelimiter))
				if err != nil {
					return nil, err
				}
				cp.Quote = currency.NewCode(strings.ReplaceAll(cp.Quote.String(), currency.UnderscoreDelimiter, currency.DashDelimiter))
				pairs = append(pairs, cp)
			}
		}
		return pairs, nil
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
}

func (e *Exchange) getCrossMarginMinimums(ctx context.Context) (map[currency.Code]float64, error) {
	crossCurrencies, err := e.CurrencySupportedByCrossMargin(ctx)
	if err != nil {
		return nil, err
	}
	crossMinimums := make(map[currency.Code]float64, len(crossCurrencies))
	for i := range crossCurrencies {
		minimum := crossCurrencies[i].MinBorrowAmount.Float64()
		if crossCurrencies[i].Status == 0 || !crossCurrencies[i].Loanable || minimum == 0 {
			continue
		}
		crossMinimums[crossCurrencies[i].Name] = minimum
	}
	return crossMinimums, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assets := e.GetAssetTypes(false)
	for x := range assets {
		pairs, err := e.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return fmt.Errorf("%v: %w", assets[x], err)
		}
		if err := e.UpdatePairs(pairs, assets[x], false); err != nil {
			return fmt.Errorf("%v: %w", assets[x], err)
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		tickers, err := e.GetTickers(ctx, currency.EMPTYPAIR, "")
		if err != nil {
			return err
		}
		for _, sTicker := range tickers {
			if err := ticker.ProcessTicker(&ticker.Price{
				Last:         sTicker.Last.Float64(),
				High:         sTicker.High24Hour.Float64(),
				Low:          sTicker.Low24Hour.Float64(),
				Bid:          sTicker.HighestBid.Float64(),
				Ask:          sTicker.LowestAsk.Float64(),
				QuoteVolume:  sTicker.QuoteVolume.Float64(),
				BaseVolume:   sTicker.BaseVolume.Float64(),
				ExchangeName: e.Name,
				Pair:         sTicker.CurrencyPair,
				AssetType:    a,
			}); err != nil {
				return err
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, errs := getSettlementCurrency(currency.EMPTYPAIR, a)
		if errs != nil {
			return errs
		}
		var tickers []*FuturesTickerResponse
		if a == asset.DeliveryFutures {
			tickers, errs = e.GetDeliveryFutureTickers(ctx, settle, currency.EMPTYPAIR)
		} else {
			tickers, errs = e.GetFuturesTickers(ctx, settle, currency.EMPTYPAIR)
		}
		for _, tkr := range tickers {
			// volume_24h counts contracts, so the base volume is derived as it is in UpdateTicker
			if err := ticker.ProcessTicker(&ticker.Price{
				Last:         tkr.Last.Float64(),
				High:         tkr.High24Hour.Float64(),
				Low:          tkr.Low24Hour.Float64(),
				BaseVolume:   futuresBaseVolume(tkr),
				QuoteVolume:  tkr.Volume24HourQuote.Float64(),
				MarkPrice:    tkr.MarkPrice.Float64(),
				IndexPrice:   tkr.IndexPrice.Float64(),
				ExchangeName: e.Name,
				Pair:         tkr.Contract,
				AssetType:    a,
			}); err != nil {
				errs = common.AppendError(errs, err)
			}
		}
		return errs
	case asset.Options:
		pairs, err := e.GetEnabledPairs(a)
		if err != nil {
			return err
		}
		// Each request returns every contract of an underlying, so one per underlying covers all its enabled pairs
		fetched := make(map[string]bool)
		for i := range pairs {
			underlying, err := e.GetUnderlyingFromCurrencyPair(pairs[i])
			if err != nil {
				return err
			}
			if fetched[underlying.String()] {
				continue
			}
			fetched[underlying.String()] = true
			tickers, err := e.GetOptionsTickers(ctx, underlying.String())
			if err != nil {
				return err
			}
			for _, tkr := range tickers {
				if err := ticker.ProcessTicker(&ticker.Price{
					Last:         tkr.LastPrice.Float64(),
					MarkPrice:    tkr.MarkPrice.Float64(),
					IndexPrice:   tkr.IndexPrice.Float64(),
					Ask:          tkr.Ask1Price.Float64(),
					AskSize:      tkr.Ask1Size.Float64(),
					Bid:          tkr.Bid1Price.Float64(),
					BidSize:      tkr.Bid1Size.Float64(),
					Pair:         tkr.Name,
					ExchangeName: e.Name,
					AssetType:    a,
				}); err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	return nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	return e.UpdateOrderbookWithLimit(ctx, p, a, 0)
}

// UpdateOrderbookWithLimit updates and returns the orderbook for a currency pair with a set orderbook size limit
func (e *Exchange) UpdateOrderbookWithLimit(ctx context.Context, p currency.Pair, a asset.Item, limit uint64) (*orderbook.Book, error) {
	book, err := e.fetchOrderbook(ctx, p, a, limit)
	if err != nil {
		return nil, err
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, book.Pair, a)
}

func (e *Exchange) fetchOrderbook(ctx context.Context, p currency.Pair, a asset.Item, limit uint64) (*orderbook.Book, error) {
	p, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	var o *OrderbookResponse
	switch a {
	case asset.Margin, asset.CrossMargin:
		if available, err := e.checkInstrumentAvailabilityInSpot(p); err != nil {
			return nil, err
		} else if !available {
			return nil, fmt.Errorf("%w: %w for %q %q", errFetchingOrderbook, errNoSpotInstrument, a, p)
		}
		fallthrough
	case asset.Spot:
		o, err = e.GetOrderbook(ctx, p, "", limit, true)
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		var settle currency.Code
		settle, err = getSettlementCurrency(p, a)
		if err != nil {
			return nil, err
		}
		o, err = e.GetFuturesOrderbook(ctx, settle, p, "", limit, true)
	case asset.DeliveryFutures:
		o, err = e.GetDeliveryOrderbook(ctx, currency.USDT, "", p, limit, true)
	case asset.Options:
		o, err = e.GetOptionsOrderbook(ctx, p, "", limit, true)
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err != nil {
		return nil, err
	}

	return &orderbook.Book{
		Exchange:          e.Name,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
		Pair:              p,
		LastUpdateID:      o.ID,
		LastUpdated:       o.Update.Time(),
		LastPushed:        o.Current.Time(),
		Bids:              o.Bids.Levels(),
		Asks:              o.Asks.Levels(),
	}, nil
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(a, "")}
	switch a {
	case asset.Spot:
		balances, err := e.GetSpotAccounts(ctx, currency.EMPTYCODE)
		if err != nil {
			return nil, err
		}
		for _, sAccount := range balances {
			subAccts[0].Balances.Set(sAccount.Currency, accounts.Balance{
				Total: sAccount.Available.Float64() + sAccount.Locked.Float64(),
				Hold:  sAccount.Locked.Float64(),
				Free:  sAccount.Available.Float64(),
			})
		}
	case asset.Margin:
		balances, err := e.GetIsolatedMarginAccountList(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		if err := setIsolatedMarginAccountBalances(&subAccts[0].Balances, balances); err != nil {
			return nil, err
		}
	case asset.CrossMargin:
		acc, err := e.GetCrossMarginAccounts(ctx)
		if err != nil {
			return nil, err
		}
		setCrossMarginAccountBalances(&subAccts[0].Balances, acc)
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
		if err != nil {
			return nil, err
		}
		var acc *FuturesAccountResponse
		if a == asset.DeliveryFutures {
			acc, err = e.GetDeliveryFuturesAccounts(ctx, settle)
		} else {
			acc, err = e.QueryFuturesAccount(ctx, settle)
		}
		if err != nil {
			return nil, err
		}
		subAccts[0].Balances.Set(acc.Currency, accounts.Balance{
			Total: acc.Total.Float64(),
			Hold:  acc.Total.Float64() - acc.Available.Float64(),
			Free:  acc.Available.Float64(),
		})
	case asset.Options:
		optionAccount, err := e.GetOptionAccount(ctx)
		if err != nil {
			return nil, err
		}
		subAccts[0].Balances.Set(optionAccount.Currency, accounts.Balance{
			Total: optionAccount.Total.Float64(),
			Hold:  optionAccount.Total.Float64() - optionAccount.Available.Float64(),
			Free:  optionAccount.Available.Float64(),
		})
	default:
		return nil, fmt.Errorf("%w asset type: %q", asset.ErrNotSupported, a)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

func setIsolatedMarginAccountBalances(balances *accounts.CurrencyBalances, response []MarginAccountItemResponse) error {
	for i := range response {
		if err := addIsolatedMarginAccountBalance(balances, response[i].Base); err != nil {
			return err
		}
		if err := addIsolatedMarginAccountBalance(balances, response[i].Quote); err != nil {
			return err
		}
	}
	return nil
}

func addIsolatedMarginAccountBalance(balances *accounts.CurrencyBalances, balance AccountBalanceInformation) error {
	// Interest is already reflected in available, so only principal is removed from available without borrow.
	borrowed := balance.Borrowed.Float64()
	available := balance.Available.Float64()
	locked := balance.LockedAmount.Float64()
	return balances.Add(balance.Currency, accounts.Balance{
		Total:                  available + locked,
		Hold:                   locked,
		Free:                   available,
		AvailableWithoutBorrow: available - borrowed,
		Borrowed:               borrowed,
	})
}

func setCrossMarginAccountBalances(balances *accounts.CurrencyBalances, account *CrossMarginAccountResponse) {
	if account == nil {
		return
	}
	for ccy, balance := range account.Balances {
		borrowed := balance.Borrowed.Float64() + balance.Interest.Float64()
		available := balance.Available.Float64()
		freeze := balance.Freeze.Float64()
		balances.Set(currency.NewCode(ccy), accounts.Balance{
			Total:                  available + freeze,
			Hold:                   freeze,
			Free:                   available,
			AvailableWithoutBorrow: available - borrowed,
			Borrowed:               borrowed,
		})
	}
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	deposits, err := e.GetDepositRecords(ctx, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return nil, err
	}
	withdrawals, err := e.GetWithdrawalRecords(ctx, nil)
	if err != nil {
		return nil, err
	}
	history := make([]exchange.FundingHistory, 0, len(deposits)+len(withdrawals))
	for _, d := range deposits {
		history = append(history, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          d.Status,
			TransferID:      d.ID,
			Timestamp:       d.Timestamp.Time(),
			Currency:        d.Currency.String(),
			Amount:          d.Amount.Float64(),
			Fee:             d.Fee.Float64(),
			TransferType:    "deposit",
			CryptoToAddress: d.Address,
			CryptoTxID:      d.TransactionID,
			CryptoChain:     d.Chain,
		})
	}
	for _, w := range withdrawals {
		history = append(history, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          w.Status,
			TransferID:      w.ID,
			Description:     w.FailReason,
			Timestamp:       w.Timestamp.Time(),
			Currency:        w.Currency.String(),
			Amount:          w.Amount.Float64(),
			Fee:             w.Fee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: w.WithdrawalAddress,
			CryptoTxID:      w.TransactionID,
			CryptoChain:     w.Chain,
		})
	}
	slices.SortStableFunc(history, func(a, b exchange.FundingHistory) int { return a.Timestamp.Compare(b.Timestamp) })
	return history, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	records, err := e.GetWithdrawalRecords(ctx, &WithdrawalRecordsRequest{Currency: c})
	if err != nil {
		return nil, err
	}
	withdrawalHistories := make([]exchange.WithdrawalHistory, len(records))
	for x, withdrawalResp := range records {
		withdrawalHistories[x] = exchange.WithdrawalHistory{
			Status:          withdrawalResp.Status,
			TransferID:      withdrawalResp.ID,
			Description:     withdrawalResp.FailReason,
			Timestamp:       withdrawalResp.Timestamp.Time(),
			Currency:        withdrawalResp.Currency.String(),
			Amount:          withdrawalResp.Amount.Float64(),
			Fee:             withdrawalResp.Fee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawalResp.WithdrawalAddress,
			CryptoTxID:      withdrawalResp.TransactionID,
			CryptoChain:     withdrawalResp.Chain,
		}
	}
	return withdrawalHistories, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	p, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		if p.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		tradeData, err := e.GetMarketTrades(ctx, p, "", false, time.Time{}, time.Time{}, 0, 0)
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(tradeData))
		for i, td := range tradeData {
			side, err := order.StringToOrderSide(td.Side)
			if err != nil {
				return nil, err
			}
			resp[i] = trade.Data{
				Exchange:     e.Name,
				TID:          td.ID,
				CurrencyPair: p,
				AssetType:    a,
				Side:         side,
				Price:        td.Price.Float64(),
				Amount:       td.Amount.Float64(),
				Timestamp:    td.CreateTime.Time(),
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(p, a)
		if err != nil {
			return nil, err
		}
		var futuresTrades []*TradingHistoryItemResponse
		if a == asset.DeliveryFutures {
			futuresTrades, err = e.GetDeliveryTradingHistory(ctx, settle, "", p.Upper(), 0, time.Time{}, time.Time{})
		} else {
			futuresTrades, err = e.GetFuturesTradingHistory(ctx, settle, p, 0, 0, "", time.Time{}, time.Time{})
		}
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(futuresTrades))
		for i, tradeH := range futuresTrades {
			side, amount := tradeSideAndAmount(tradeH.Size.Float64())
			resp[i] = trade.Data{
				TID:          strconv.FormatUint(tradeH.ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         side,
				Price:        tradeH.Price.Float64(),
				Amount:       amount,
				Timestamp:    tradeH.CreateTime.Time(),
			}
		}
	case asset.Options:
		trades, err := e.GetOptionsTradeHistory(ctx, p.Upper(), "", 0, 0, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(trades))
		for i, td := range trades {
			side, amount := tradeSideAndAmount(td.Size.Float64())
			resp[i] = trade.Data{
				TID:          strconv.FormatUint(td.ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         side,
				Price:        td.Price.Float64(),
				Amount:       amount,
				Timestamp:    td.CreateTime.Time(),
			}
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
// TODO: support multiple order types (IOC)
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	if err := e.formatOrderClientIDAndPair(s); err != nil {
		return nil, err
	}

	switch s.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		req, err := e.getSpotOrderRequest(s)
		if err != nil {
			return nil, err
		}
		sOrder, err := e.PlaceSpotOrder(ctx, req)
		if err != nil {
			return nil, err
		}
		response, err := s.DeriveSubmitResponse(sOrder.OrderID)
		if err != nil {
			return nil, err
		}
		side, err := order.StringToOrderSide(sOrder.Side)
		if err != nil {
			return nil, err
		}
		response.Side = side
		status, err := order.StringToOrderStatus(sOrder.Status)
		if err != nil {
			return nil, err
		}
		response.Status = status
		response.Fee = sOrder.FeeDeducted.Float64()
		response.FeeAsset = sOrder.FeeCurrency
		response.Pair = s.Pair
		response.ClientOrderID = getClientOrderIDFromText(sOrder.Text)
		response.Date = sOrder.CreateTime.Time()
		response.LastUpdated = sOrder.UpdateTime.Time()
		return response, nil
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		// TODO: See https://www.gate.io/docs/developers/apiv4/en/#create-a-futures-order
		//	* iceberg orders
		//	* auto_size (close_long, close_short)
		// 	* stp_act (self trade prevention)
		fOrder, err := getFuturesOrderRequest(s)
		if err != nil {
			return nil, err
		}
		fOrder.Settle, err = getSettlementCurrency(s.Pair, s.AssetType)
		if err != nil {
			return nil, err
		}
		o, err := e.PlaceFuturesOrder(ctx, fOrder)
		if err != nil {
			return nil, err
		}
		return deriveFuturesSubmitResponse(s, o)
	case asset.DeliveryFutures:
		fOrder, err := getDeliveryOrderRequest(s)
		if err != nil {
			return nil, err
		}
		fOrder.Settle, err = getSettlementCurrency(s.Pair, s.AssetType)
		if err != nil {
			return nil, err
		}

		o, err := e.PlaceDeliveryOrder(ctx, fOrder)
		if err != nil {
			return nil, err
		}
		return deriveFuturesSubmitResponse(s, o)
	case asset.Options:
		size, err := getFutureOrderSize(s)
		if err != nil {
			return nil, err
		}
		price, tif, err := orderPriceAndTIF(s)
		if err != nil {
			return nil, err
		}
		if tif == fokTIF {
			return nil, fmt.Errorf("%w: %q is not supported for options", order.ErrUnsupportedTimeInForce, s.TimeInForce)
		}
		optionOrder, err := e.PlaceOptionOrder(ctx, &OptionOrderRequest{
			Contract:    s.Pair,
			OrderSize:   types.Number(size),
			Price:       types.Number(price),
			ReduceOnly:  s.ReduceOnly,
			TimeInForce: tif,
			Text:        s.ClientOrderID,
		})
		if err != nil {
			return nil, err
		}
		response, err := s.DeriveSubmitResponse(strconv.FormatUint(optionOrder.OptionOrderID, 10))
		if err != nil {
			return nil, err
		}
		if response.Status, err = futuresOrderStatus(optionOrder.Status, optionOrder.FinishAs); err != nil {
			return nil, err
		}
		_, response.Amount, response.RemainingAmount = getSideAndAmountFromSize(optionOrder.Size.Float64(), optionOrder.RemainingAmount.Float64())
		response.Price = optionOrder.Price.Float64()
		response.AverageExecutedPrice = optionOrder.FillPrice.Float64()
		response.ClientOrderID = getClientOrderIDFromText(optionOrder.Text)
		response.Date = optionOrder.CreateTime.Time()
		response.LastUpdated = orderLastUpdated(types.Time{}, optionOrder.FinishTime, optionOrder.CreateTime)
		return response, nil
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, s.AssetType)
	}
}

// deriveFuturesSubmitResponse returns the submit response for a placed perpetual or delivery futures order
func deriveFuturesSubmitResponse(s *order.Submit, o *FuturesOrderResponse) (*order.SubmitResponse, error) {
	resp, err := s.DeriveSubmitResponse(strconv.FormatUint(o.ID, 10))
	if err != nil {
		return nil, err
	}
	if resp.Status, err = futuresOrderStatus(o.Status, o.FinishAs); err != nil {
		return nil, err
	}
	_, resp.Amount, resp.RemainingAmount = getSideAndAmountFromSize(o.Size.Float64(), o.RemainingAmount.Float64())
	resp.Price = o.Price.Float64()
	resp.AverageExecutedPrice = o.FillPrice.Float64()
	resp.ClientOrderID = getClientOrderIDFromText(o.Text)
	resp.Date = o.CreateTime.Time()
	resp.LastUpdated = orderLastUpdated(o.UpdateTime, o.FinishTime, o.CreateTime)
	return resp, nil
}

// futuresOrderStatus returns the status of a perpetual futures, delivery or options order, which Gate reports only as
// open or finished, finished orders saying how they finished
func futuresOrderStatus(status, finishAs string) (order.Status, error) {
	if status == statusOpen {
		return order.Open, nil
	}
	return orderStatusFromFinishAs(finishAs)
}

// orderStatusFromFinishAs returns the status of an order that finished as Gate reports. Gate also finishes an order
// when it cancels the unfilled remainder for a reason the order package has no status for, such as an unmet time in
// force or a reduce-only conflict, so those report as cancelled. Gate's default futures websocket page spells two
// finish values auto_deleveraging and position_close, where its /en/ page and websocket SDK give auto_deleveraged and
// position_closed, so both spellings are accepted
func orderStatusFromFinishAs(finishAs string) (order.Status, error) {
	switch finishAs {
	case "", statusOpen:
		return order.Open, nil
	case "ioc", "poc", "fok", "small", "depth_not_enough", "trader_not_enough", "liquidate_cancelled", "price_protect_cancelled", "reduce_only", "reduce_out", "mmp_cancelled", "unified_check_failed":
		return order.Cancelled, nil
	case "auto_deleveraging":
		return order.AutoDeleverage, nil
	case "position_close":
		return order.Closed, nil
	case "unknown", "-":
		return order.UnknownStatus, nil
	}
	return order.StringToOrderStatus(finishAs)
}

// orderLastUpdated returns when Gate last changed an order, falling back to when it finished and then to when it was
// created, as delivery and options orders carry no update time and open orders no finish time
func orderLastUpdated(updated, finished, created types.Time) time.Time {
	for _, t := range []types.Time{updated, finished, created} {
		if !t.Time().IsZero() {
			return t.Time()
		}
	}
	return time.Time{}
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	fPair, err := e.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}
	switch o.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		_, err = e.CancelSingleSpotOrder(ctx, o.OrderID, fPair, o.AssetType == asset.CrossMargin)
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		var settle currency.Code
		if settle, err = getSettlementCurrency(o.Pair, o.AssetType); err == nil {
			if o.AssetType == asset.DeliveryFutures {
				_, err = e.CancelSingleDeliveryOrder(ctx, settle, o.OrderID)
			} else {
				_, err = e.CancelSingleFuturesOrder(ctx, settle, o.OrderID, "")
			}
		}
	case asset.Options:
		_, err = e.CancelOptionSingleOrder(ctx, o.OrderID)
	default:
		return fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels orders of one asset by their IDs. Spot, margin and cross margin orders, which may span
// several pairs, and perpetual futures orders are cancelled in batches, while delivery and options orders, which Gate
// cannot cancel in a batch by ID, are cancelled one at a time. Each order's outcome is reported in the response status
// keyed by its ID, as cancelled or as the reason Gate gave for not cancelling it, and the errors of requests that
// failed are returned together once every order has been tried
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	a := o[0].AssetType
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin, asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures, asset.Options:
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	isSpot := a == asset.Spot || a == asset.Margin || a == asset.CrossMargin
	pairs := make(currency.Pairs, len(o))
	var settle currency.Code
	for i := range o {
		if o[i].AssetType != a {
			return nil, fmt.Errorf("%w: %s and %s", errSingleAssetRequired, a, o[i].AssetType)
		}
		if err := o[i].Validate(o[i].StandardCancel()); err != nil {
			return nil, err
		}
		var err error
		if pairs[i], err = e.FormatExchangeCurrency(o[i].Pair, a); err != nil {
			return nil, err
		}
		if !isSpot {
			if settle, err = getSettlementCurrency(pairs[i], a); err != nil {
				return nil, err
			}
		}
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string, len(o))}
	var errs error
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		for start := 0; start < len(o); start += cancelBatchOrderLimit {
			batch := o[start:min(start+cancelBatchOrderLimit, len(o))]
			params := make([]CancelOrderByIDRequest, len(batch))
			for i := range batch {
				params[i] = CancelOrderByIDRequest{CurrencyPair: pairs[start+i], ID: batch[i].OrderID}
			}
			results, err := e.CancelBatchOrdersWithIDList(ctx, params)
			outcomes := make([]cancelOutcome, len(results))
			for i, r := range results {
				outcomes[i] = cancelOutcome{orderID: r.OrderID, text: r.Text, succeeded: r.Succeeded, reason: joinNonEmpty(r.Label, r.Message)}
			}
			errs = common.AppendError(errs, recordCancelOutcomes(resp, batch, outcomes, err))
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		for start := 0; start < len(o); start += cancelBatchOrderLimit {
			batch := o[start:min(start+cancelBatchOrderLimit, len(o))]
			orderIDs := make([]string, len(batch))
			for i := range batch {
				orderIDs[i] = batch[i].OrderID
			}
			results, err := e.CancelBatchFuturesOrders(ctx, settle, orderIDs)
			outcomes := make([]cancelOutcome, len(results))
			for i, r := range results {
				outcomes[i] = cancelOutcome{orderID: r.ID, succeeded: r.Succeeded, reason: r.Message}
			}
			errs = common.AppendError(errs, recordCancelOutcomes(resp, batch, outcomes, err))
		}
	case asset.DeliveryFutures:
		for i := range o {
			_, err := e.CancelSingleDeliveryOrder(ctx, settle, o[i].OrderID)
			errs = common.AppendError(errs, recordCancelOutcomes(resp, o[i:i+1], []cancelOutcome{{orderID: o[i].OrderID, succeeded: true}}, err))
		}
	case asset.Options:
		for i := range o {
			_, err := e.CancelOptionSingleOrder(ctx, o[i].OrderID)
			errs = common.AppendError(errs, recordCancelOutcomes(resp, o[i:i+1], []cancelOutcome{{orderID: o[i].OrderID, succeeded: true}}, err))
		}
	}
	return resp, errs
}

// cancelBatchOrderLimit is the most spot or perpetual futures orders Gate cancels in one batch request
const cancelBatchOrderLimit = 20

// cancelOutcome is Gate's outcome of cancelling one order, identified by its order ID and, for spot orders, its text
type cancelOutcome struct {
	orderID, text string
	succeeded     bool
	reason        string
}

// recordCancelOutcomes reports each requested order's outcome in the batch response, matching Gate's outcomes to the
// orders by order ID or by the custom text a spot order may be cancelled by. A request that failed is reported against
// every order it carried, and an order Gate returned no outcome for is reported as missing; both are returned as errors
func recordCancelOutcomes(resp *order.CancelBatchResponse, requested []order.Cancel, outcomes []cancelOutcome, err error) error {
	if err != nil {
		ids := make([]string, len(requested))
		for i := range requested {
			ids[i] = requested[i].OrderID
			resp.Status[ids[i]] = err.Error()
		}
		return fmt.Errorf("error cancelling orders %v: %w", ids, err)
	}
	var errs error
	for i := range requested {
		id := requested[i].OrderID
		idx := slices.IndexFunc(outcomes, func(o cancelOutcome) bool { return o.orderID == id || (o.text != "" && o.text == id) })
		switch {
		case idx == -1:
			resp.Status[id] = errCancelOutcomeMissing.Error()
			errs = common.AppendError(errs, fmt.Errorf("%w: order %s", errCancelOutcomeMissing, id))
		case outcomes[idx].succeeded:
			resp.Status[id] = order.Cancelled.String()
		default:
			resp.Status[id] = outcomes[idx].reason
		}
	}
	return errs
}

// joinNonEmpty joins Gate's error label and message, either of which may be empty
func joinNonEmpty(label, message string) string {
	switch {
	case label == "":
		return message
	case message == "":
		return label
	}
	return label + ": " + message
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, o *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	if err := o.Validate(); err != nil {
		return resp, err
	}

	fmtPair, err := e.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return resp, err
	}

	spotSide := order.UnknownSide
	futuresSide := ""
	switch {
	case o.Side.IsLong():
		spotSide = order.Buy
		futuresSide = order.Bid.Lower()
	case o.Side.IsShort():
		spotSide = order.Sell
		futuresSide = order.Ask.Lower()
	case o.Side == order.UnknownSide, o.Side == order.AnySide:
	default:
		return resp, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, o.Side)
	}

	switch o.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		// Gate cancels every pair's orders when the pair is omitted, which is wider than this method promises
		if fmtPair.IsEmpty() {
			return resp, currency.ErrCurrencyPairEmpty
		}
		cancel, err := e.CancelAllOpenOrdersSpecifiedCurrencyPair(ctx, &CancelAllSpotOrdersRequest{CurrencyPair: fmtPair, Side: spotSide, Account: o.AssetType.String()})
		if err != nil {
			return resp, err
		}
		for _, sPriceTriggeredOrder := range cancel {
			resp.Add(sPriceTriggeredOrder.OrderID, sPriceTriggeredOrder.Status)
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(fmtPair, o.AssetType)
		if err != nil {
			return resp, err
		}
		var cancel []*FuturesOrderResponse
		if o.AssetType == asset.DeliveryFutures {
			cancel, err = e.CancelMultipleDeliveryOrders(ctx, fmtPair, futuresSide, settle)
		} else {
			cancel, err = e.CancelMultipleFuturesOpenOrders(ctx, settle, &CancelFuturesOrdersRequest{Contract: fmtPair, Side: futuresSide})
		}
		if err != nil {
			return resp, err
		}
		for _, cancelResp := range cancel {
			resp.Add(strconv.FormatUint(cancelResp.ID, 10), cancelResp.FinishAs)
		}
	case asset.Options:
		var underlying currency.Pair
		if !o.Pair.IsEmpty() {
			underlying, err = e.GetUnderlyingFromCurrencyPair(o.Pair)
			if err != nil {
				return resp, err
			}
		}
		optionOrderResponse, err := e.CancelMultipleOptionOpenOrders(ctx, fmtPair, underlying.String(), futuresSide)
		if err != nil {
			return resp, err
		}
		for _, oResp := range optionOrderResponse {
			resp.Add(strconv.FormatUint(oResp.OptionOrderID, 10), oResp.FinishAs)
		}
	default:
		return resp, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, o.AssetType)
	}

	return resp, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}

	pair, err := e.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		spotOrder, err := e.GetSpotOrder(ctx, orderID, pair, a)
		if err != nil {
			return nil, err
		}
		return e.spotOrderDetail(spotOrder, pair, a)
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(pair, a)
		if err != nil {
			return nil, err
		}
		var fOrder *FuturesOrderResponse
		if a == asset.DeliveryFutures {
			fOrder, err = e.GetSingleDeliveryOrder(ctx, settle, orderID)
		} else {
			fOrder, err = e.GetSingleFuturesOrder(ctx, settle, orderID)
		}
		if err != nil {
			return nil, err
		}
		format, err := e.GetPairFormat(a, false)
		if err != nil {
			return nil, err
		}
		return e.futuresOrderDetail(fOrder, a, settle, format)
	case asset.Options:
		optionOrder, err := e.GetSingleOptionOrder(ctx, orderID)
		if err != nil {
			return nil, err
		}
		return e.optionsOrderDetail(optionOrder)
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
}

// spotOrderType returns the type of a spot order, reading a unified account's borrow and repay variants as the limit or
// market order they place
func spotOrderType(t string) (order.Type, error) {
	switch t {
	case "limit_repay", "limit_borrow", "limit_borrow_repay":
		return order.Limit, nil
	case "market_repay", "market_borrow", "market_borrow_repay":
		return order.Market, nil
	}
	return order.StringToOrderType(t)
}

// spotOrderAsset returns the asset of a spot market order from its account label. Gate labels a unified account's
// orders unified, and an empty label means spot
func spotOrderAsset(account string) (asset.Item, error) {
	if account == "" || account == unifiedAccount {
		return asset.Spot, nil
	}
	return asset.New(account)
}

// spotOrderDetail returns the order detail of a spot, margin or cross margin order
func (e *Exchange) spotOrderDetail(o *SpotOrderResponse, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return nil, err
	}
	oType, err := order.StringToOrderType(o.Type)
	if err != nil {
		return nil, err
	}
	status, err := order.StringToOrderStatus(o.Status)
	if err != nil {
		return nil, err
	}
	// A unified account order may trade in a quote currency other than the pair's, which Gate reports as trade_quote
	costAsset := o.TradeQuote
	if costAsset.IsEmpty() {
		costAsset = pair.Quote
	}
	return &order.Detail{
		Exchange:             e.Name,
		OrderID:              o.OrderID,
		ClientOrderID:        getClientOrderIDFromText(o.Text),
		Side:                 side,
		Type:                 oType,
		Status:               status,
		Pair:                 pair,
		AssetType:            a,
		Price:                o.Price.Float64(),
		Amount:               o.Amount.Float64(),
		ExecutedAmount:       o.Amount.Float64() - o.RemainingAmount.Float64(),
		RemainingAmount:      o.RemainingAmount.Float64(),
		AverageExecutedPrice: o.AverageFillPrice.Float64(),
		Cost:                 o.FilledTotal.Float64(),
		CostAsset:            costAsset,
		Fee:                  o.FeeDeducted.Float64(),
		FeeAsset:             o.FeeCurrency,
		Date:                 o.CreateTime.Time(),
		LastUpdated:          o.UpdateTime.Time(),
	}, nil
}

// futuresOrderDetail returns the order detail of a perpetual or delivery futures order, whose size Gate signs by side
func (e *Exchange) futuresOrderDetail(o *FuturesOrderResponse, a asset.Item, settle currency.Code, format currency.PairFormat) (*order.Detail, error) {
	status, err := futuresOrderStatus(o.Status, o.FinishAs)
	if err != nil {
		return nil, err
	}
	tif, err := timeInForceFromString(o.TimeInForce)
	if err != nil {
		return nil, err
	}
	side, amount, remaining := getSideAndAmountFromSize(o.Size.Float64(), o.RemainingAmount.Float64())
	return &order.Detail{
		Exchange:             e.Name,
		OrderID:              strconv.FormatUint(o.ID, 10),
		ClientOrderID:        getClientOrderIDFromText(o.Text),
		Side:                 side,
		Type:                 getTypeFromTimeInForce(o.TimeInForce, o.Price.Float64()),
		Status:               status,
		TimeInForce:          tif,
		ReduceOnly:           o.IsReduceOnly,
		Pair:                 o.Contract.Format(format),
		AssetType:            a,
		SettlementCurrency:   settle,
		Price:                o.Price.Float64(),
		Amount:               amount,
		ContractAmount:       amount,
		ExecutedAmount:       amount - remaining,
		RemainingAmount:      remaining,
		AverageExecutedPrice: o.FillPrice.Float64(),
		Date:                 o.CreateTime.Time(),
		LastUpdated:          orderLastUpdated(o.UpdateTime, o.FinishTime, o.CreateTime),
	}, nil
}

// optionsOrderDetail returns the order detail of an options order, whose size Gate signs by side
func (e *Exchange) optionsOrderDetail(o *OptionOrderResponse) (*order.Detail, error) {
	pair, err := currency.NewPairFromString(o.Contract)
	if err != nil {
		return nil, err
	}
	status, err := futuresOrderStatus(o.Status, o.FinishAs)
	if err != nil {
		return nil, err
	}
	tif, err := timeInForceFromString(o.TimeInForce)
	if err != nil {
		return nil, err
	}
	side, amount, remaining := getSideAndAmountFromSize(o.Size.Float64(), o.RemainingAmount.Float64())
	return &order.Detail{
		Exchange:             e.Name,
		OrderID:              strconv.FormatUint(o.OptionOrderID, 10),
		ClientOrderID:        getClientOrderIDFromText(o.Text),
		Side:                 side,
		Type:                 getTypeFromTimeInForce(o.TimeInForce, o.Price.Float64()),
		Status:               status,
		TimeInForce:          tif,
		ReduceOnly:           o.IsReduceOnly,
		Pair:                 pair,
		AssetType:            asset.Options,
		Price:                o.Price.Float64(),
		Amount:               amount,
		ExecutedAmount:       amount - remaining,
		RemainingAmount:      remaining,
		AverageExecutedPrice: o.FillPrice.Float64(),
		Date:                 o.CreateTime.Time(),
		LastUpdated:          orderLastUpdated(types.Time{}, o.FinishTime, o.CreateTime),
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	addr, err := e.GenerateCurrencyDepositAddress(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	if chain != "" {
		for _, addrMultiChain := range addr.MultichainAddresses {
			if addrMultiChain.ObtainFailed == 1 {
				continue
			}
			if addrMultiChain.Chain == chain {
				return &deposit.Address{
					Chain:   addrMultiChain.Chain,
					Address: addrMultiChain.Address,
					Tag:     addrMultiChain.PaymentID,
				}, nil
			}
		}
		return nil, fmt.Errorf("%w for currency %s chain %s", deposit.ErrAddressNotFound, cryptocurrency, chain)
	}
	return &deposit.Address{
		Address: addr.Address,
		Chain:   chain,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	response, err := e.WithdrawCurrency(ctx, &WithdrawalRequest{
		Amount:          types.Number(withdrawRequest.Amount),
		Currency:        withdrawRequest.Currency,
		Address:         withdrawRequest.Crypto.Address,
		Memo:            withdrawRequest.Crypto.AddressTag,
		Chain:           withdrawRequest.Crypto.Chain,
		WithdrawOrderID: withdrawRequest.ClientOrderID,
	})
	if err != nil {
		return nil, err
	}
	// The record ID identifies the withdrawal from submission, whereas Gate only fills the transaction hash once the
	// withdrawal reaches the chain
	return &withdraw.ExchangeResponse{
		Name:   e.Name,
		ID:     response.ID,
		Status: response.Status,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
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
	if !e.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	format, err := e.GetPairFormat(req.AssetType, false)
	if err != nil {
		return nil, err
	}
	switch req.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		// Gate rejects account=cross_margin on spot/open_orders for some
		// account types, so the listing is fetched without an account filter
		// and matched against the requested asset via each order's account
		// field below.
		spotOrders, err := e.GetSpotOpenOrders(ctx, 0, 0, "")
		if err != nil {
			return nil, err
		}
		for _, sOrder := range spotOrders {
			for _, sOrderDetail := range sOrder.Orders {
				if sOrderDetail.Status != statusOpen {
					continue
				}
				if a, err := spotOrderAsset(sOrderDetail.Account); err != nil || a != req.AssetType {
					continue
				}
				detail, err := e.spotOrderDetail(sOrderDetail, sOrder.CurrencyPair.Format(format), req.AssetType)
				if err != nil {
					return nil, err
				}
				orders = append(orders, *detail)
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, req.AssetType)
		if err != nil {
			return nil, err
		}
		var futuresOrders []*FuturesOrderResponse
		if req.AssetType == asset.DeliveryFutures {
			futuresOrders, err = e.GetDeliveryOrders(ctx, currency.EMPTYPAIR, statusOpen, settle, "", 0, 0, false)
		} else {
			futuresOrders, err = e.GetFuturesOrders(ctx, currency.EMPTYPAIR, statusOpen, "", settle, 0, 0)
		}
		if err != nil {
			return nil, err
		}
		for _, fOrder := range futuresOrders {
			if fOrder.Status != statusOpen || (len(req.Pairs) > 0 && !req.Pairs.Contains(fOrder.Contract, true)) {
				continue
			}
			detail, err := e.futuresOrderDetail(fOrder, req.AssetType, settle, format)
			if err != nil {
				return nil, err
			}
			orders = append(orders, *detail)
		}
	case asset.Options:
		optionsOrders, err := e.GetOptionFuturesOrders(ctx, currency.EMPTYPAIR, "", statusOpen, 0, 0, req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
		for _, optOrd := range optionsOrders {
			detail, err := e.optionsOrderDetail(optOrd)
			if err != nil {
				return nil, err
			}
			orders = append(orders, *detail)
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var resp []order.Detail

	switch req.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		format, err := e.GetPairFormat(req.AssetType, true)
		if err != nil {
			return nil, err
		}
		pairs := req.Pairs
		if len(pairs) == 0 {
			pairs = currency.Pairs{currency.EMPTYPAIR}
		}
		for i := range pairs {
			fp := pairs[i]
			if fp.IsPopulated() {
				fp = pairs[i].Format(format)
			}
			o, err := e.GetMySpotTradingHistory(ctx, &GetMySpotTradingHistoryRequest{
				CurrencyPair: fp,
				OrderID:      req.FromOrderID,
				Limit:        1000,
				From:         req.StartTime,
				To:           req.EndTime,
			})
			if err != nil {
				return nil, err
			}
			for _, sTradeDetail := range o {
				fp, err = e.MatchSymbolWithAvailablePairs(sTradeDetail.CurrencyPair.String(), req.AssetType, true)
				if err != nil {
					return nil, err
				}
				side, err := order.StringToOrderSide(sTradeDetail.Side)
				if err != nil {
					return nil, err
				}
				od := order.Detail{
					OrderID:        sTradeDetail.OrderID,
					Amount:         sTradeDetail.Amount.Float64(),
					ExecutedAmount: sTradeDetail.Amount.Float64(),
					Price:          sTradeDetail.Price.Float64(),
					Date:           sTradeDetail.CreateTime.Time(),
					Side:           side,
					Exchange:       e.Name,
					Pair:           fp,
					AssetType:      req.AssetType,
					Fee:            sTradeDetail.Fee.Float64(),
					FeeAsset:       sTradeDetail.FeeCurrency,
					Trades: []order.TradeHistory{{
						TID:       sTradeDetail.TradeID,
						Price:     sTradeDetail.Price.Float64(),
						Amount:    sTradeDetail.Amount.Float64(),
						Fee:       sTradeDetail.Fee.Float64(),
						FeeAsset:  sTradeDetail.FeeCurrency.String(),
						Side:      side,
						IsMaker:   sTradeDetail.Role == "maker",
						Timestamp: sTradeDetail.CreateTime.Time(),
						Exchange:  e.Name,
					}},
				}
				od.InferCostsAndTimes()
				resp = append(resp, od)
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		format, err := e.GetPairFormat(req.AssetType, true)
		if err != nil {
			return nil, err
		}
		pairs := req.Pairs
		if len(pairs) == 0 {
			pairs = currency.Pairs{currency.EMPTYPAIR}
		}
		for i := range pairs {
			fp := pairs[i]
			if fp.IsPopulated() {
				fp = pairs[i].Format(format)
			}
			settle, err := getSettlementCurrency(fp, req.AssetType)
			if err != nil {
				return nil, err
			}
			var o []*TradingHistoryItemResponse
			if req.AssetType == asset.DeliveryFutures {
				o, err = e.GetMyDeliveryTradingHistory(ctx, settle, req.FromOrderID, fp, 0, 0, 0, "")
			} else {
				o, err = e.GetMyFuturesTradingHistory(ctx, settle, "", req.FromOrderID, fp, 0, 0)
			}
			if err != nil {
				return nil, err
			}
			for _, fill := range o {
				fp, err = e.MatchSymbolWithAvailablePairs(fill.Contract, req.AssetType, true)
				if err != nil {
					return nil, err
				}
				side, amount, _ := getSideAndAmountFromSize(fill.Size.Float64(), 0)
				od := order.Detail{
					OrderID:   fill.OrderID,
					Side:      side,
					Amount:    amount,
					Price:     fill.Price.Float64(),
					Date:      fill.CreateTime.Time(),
					Exchange:  e.Name,
					Pair:      fp,
					AssetType: req.AssetType,
					// Costs are kept in the settlement currency, as a delivery contract's quote names the contract
					CostAsset: settle,
					Trades: []order.TradeHistory{{
						TID:       strconv.FormatUint(fill.ID, 10),
						Price:     fill.Price.Float64(),
						Amount:    amount,
						Fee:       fill.Fee.Float64(),
						Side:      side,
						IsMaker:   fill.Role == "maker",
						Timestamp: fill.CreateTime.Time(),
						Exchange:  e.Name,
					}},
				}
				od.InferCostsAndTimes()
				resp = append(resp, od)
			}
		}
	case asset.Options:
		if len(req.Pairs) == 0 {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		format, err := e.GetPairFormat(req.AssetType, true)
		if err != nil {
			return nil, err
		}
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, req.AssetType)
		if err != nil {
			return nil, err
		}
		for i := range req.Pairs {
			fp := req.Pairs[i].Format(format)
			underlying, err := e.GetUnderlyingFromCurrencyPair(fp)
			if err != nil {
				return nil, err
			}
			o, err := e.GetMyOptionsTradingHistory(ctx, underlying.String(), fp.Upper(), 0, 0, req.StartTime, req.EndTime)
			if err != nil {
				return nil, err
			}
			for _, fill := range o {
				side, amount, _ := getSideAndAmountFromSize(fill.Size.Float64(), 0)
				od := order.Detail{
					OrderID:   strconv.FormatUint(fill.OrderID, 10),
					Side:      side,
					Amount:    amount,
					Price:     fill.Price.Float64(),
					Date:      fill.CreateTime.Time(),
					Exchange:  e.Name,
					Pair:      fp,
					AssetType: req.AssetType,
					// Costs are kept in the settlement currency, as an options contract's quote names the contract
					CostAsset: settle,
					Trades: []order.TradeHistory{{
						TID:       strconv.FormatUint(fill.ID, 10),
						Price:     fill.Price.Float64(),
						Amount:    amount,
						Side:      side,
						IsMaker:   fill.TradeRole == "maker",
						Timestamp: fill.CreateTime.Time(),
						Exchange:  e.Name,
					}},
				}
				od.InferCostsAndTimes()
				resp = append(resp, od)
			}
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(e.Name, resp), nil
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	var listCandlesticks []kline.Candle
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		candles, err := e.GetCandlesticks(ctx, req.RequestFormatted, 0, req.Start, req.End, req.ExchangeInterval)
		if err != nil {
			return nil, err
		}
		listCandlesticks = make([]kline.Candle, len(candles))
		for i, sCandle := range candles {
			listCandlesticks[i] = kline.Candle{
				Time:   sCandle.Timestamp.Time(),
				Open:   sCandle.OpenPrice.Float64(),
				High:   sCandle.HighestPrice.Float64(),
				Low:    sCandle.LowestPrice.Float64(),
				Close:  sCandle.ClosePrice.Float64(),
				Volume: sCandle.BaseCurrencyAmount.Float64(),
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(pair, a)
		if err != nil {
			return nil, err
		}
		var candles []*FuturesCandlestickResponse
		if a == asset.DeliveryFutures {
			candles, err = e.GetDeliveryFuturesCandlesticks(ctx, settle, req.RequestFormatted.Upper(), req.Start, req.End, 0, req.ExchangeInterval)
		} else {
			candles, err = e.GetFuturesCandlesticks(ctx, settle, req.RequestFormatted, req.Start, req.End, 0, req.ExchangeInterval, "")
		}
		if err != nil {
			return nil, err
		}
		listCandlesticks = make([]kline.Candle, len(candles))
		for i, fCandle := range candles {
			listCandlesticks[i] = kline.Candle{
				Time:   fCandle.Timestamp.Time(),
				Open:   fCandle.OpenPrice.Float64(),
				High:   fCandle.HighestPrice.Float64(),
				Low:    fCandle.LowestPrice.Float64(),
				Close:  fCandle.ClosePrice.Float64(),
				Volume: fCandle.Volume.Float64(),
			}
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	return req.ProcessResponse(listCandlesticks)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	candlestickItems := make([]kline.Candle, 0, req.Size())
	for _, r := range req.RangeHolder.Ranges {
		switch a {
		case asset.Spot, asset.Margin, asset.CrossMargin:
			candles, err := e.GetCandlesticks(ctx, req.RequestFormatted, 0, r.Start.Time, r.End.Time, req.ExchangeInterval)
			if err != nil {
				return nil, err
			}
			for _, sCandle := range candles {
				candlestickItems = append(candlestickItems, kline.Candle{
					Time:   sCandle.Timestamp.Time(),
					Open:   sCandle.OpenPrice.Float64(),
					High:   sCandle.HighestPrice.Float64(),
					Low:    sCandle.LowestPrice.Float64(),
					Close:  sCandle.ClosePrice.Float64(),
					Volume: sCandle.BaseCurrencyAmount.Float64(),
				})
			}
		case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
			settle, err := getSettlementCurrency(pair, a)
			if err != nil {
				return nil, err
			}
			var candles []*FuturesCandlestickResponse
			if a == asset.DeliveryFutures {
				candles, err = e.GetDeliveryFuturesCandlesticks(ctx, settle, req.RequestFormatted.Upper(), r.Start.Time, r.End.Time, 0, req.ExchangeInterval)
			} else {
				candles, err = e.GetFuturesCandlesticks(ctx, settle, req.RequestFormatted, r.Start.Time, r.End.Time, 0, req.ExchangeInterval, "")
			}
			if err != nil {
				return nil, err
			}
			for _, fCandle := range candles {
				candlestickItems = append(candlestickItems, kline.Candle{
					Time:   fCandle.Timestamp.Time(),
					Open:   fCandle.OpenPrice.Float64(),
					High:   fCandle.HighestPrice.Float64(),
					Low:    fCandle.LowestPrice.Float64(),
					Close:  fCandle.ClosePrice.Float64(),
					Volume: fCandle.Volume.Float64(),
				})
			}
		default:
			return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
		}
	}
	return req.ProcessResponse(candlestickItems)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	chains, err := e.ListCurrencyChain(ctx, cryptocurrency.Upper())
	if err != nil {
		return nil, err
	}
	availableChains := make([]string, 0, len(chains))
	for _, cChain := range chains {
		if cChain.IsDisabled == 0 {
			availableChains = append(availableChains, cChain.Chain)
		}
	}
	return availableChains, nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// checkInstrumentAvailabilityInSpot checks whether the instrument is available in the spot exchange
// if so we can use the instrument to retrieve orderbook and ticker information using the spot endpoints.
func (e *Exchange) checkInstrumentAvailabilityInSpot(instrument currency.Pair) (bool, error) {
	availables, err := e.CurrencyPairs.GetPairs(asset.Spot, false)
	if err != nil {
		return false, err
	}
	return availables.Contains(instrument, true), nil
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, a asset.Item) ([]futures.Contract, error) {
	if !a.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		contracts, err := e.GetAllFutureContracts(ctx, settle, 0, 0)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, len(contracts))
		for i, fContract := range contracts {
			contractSettlementType := futures.Linear
			switch {
			case fContract.Name.Base.Equal(currency.BTC) && settle.Equal(currency.BTC):
				contractSettlementType = futures.Inverse
			case !fContract.Name.Base.Equal(settle) && !settle.Equal(currency.USDT):
				contractSettlementType = futures.Quanto
			}
			c := futures.Contract{
				Exchange:           e.Name,
				Name:               fContract.Name,
				Underlying:         fContract.Name,
				Asset:              a,
				IsActive:           fContract.DelistedTime.Time().IsZero() || fContract.DelistedTime.Time().After(time.Now()),
				Type:               futures.Perpetual,
				SettlementType:     contractSettlementType,
				SettlementCurrency: settle,
				Multiplier:         fContract.QuantoMultiplier.Float64(),
				MaxLeverage:        fContract.LeverageMax.Float64(),
				LatestRate: fundingrate.Rate{
					Time: fContract.FundingNextApply.Time().Add(-fContract.fundingInterval()),
					Rate: fContract.FundingRate.Decimal(),
				},
			}
			resp[i] = c
		}
		return resp, nil
	case asset.DeliveryFutures:
		contracts, err := e.GetAllDeliveryContracts(ctx, settle)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, len(contracts))
		for i, dContract := range contracts {
			name, err := currency.NewPairFromString(dContract.Name)
			if err != nil {
				return nil, err
			}
			underlying, err := currency.NewPairFromString(dContract.Underlying)
			if err != nil {
				return nil, err
			}
			// no start information, inferring it based on contract type
			// gateio also reuses contracts for kline data, cannot use a lookup to see the first trade
			var startTime time.Time
			endTime := dContract.ExpireTime.Time()
			ct := futures.LongDated
			switch dContract.Cycle {
			case "WEEKLY":
				ct = futures.Weekly
				startTime = endTime.Add(-kline.OneWeek.Duration())
			case "BI-WEEKLY":
				ct = futures.Fortnightly
				startTime = endTime.Add(-kline.TwoWeek.Duration())
			case "QUARTERLY":
				ct = futures.Quarterly
				startTime = endTime.Add(-kline.ThreeMonth.Duration())
			case "BI-QUARTERLY":
				ct = futures.HalfYearly
				startTime = endTime.Add(-kline.SixMonth.Duration())
			}
			resp[i] = futures.Contract{
				Exchange:           e.Name,
				Name:               name,
				Underlying:         underlying,
				Asset:              a,
				StartDate:          startTime,
				EndDate:            endTime,
				SettlementType:     futures.Linear,
				IsActive:           !dContract.InDelisting,
				Type:               ct,
				SettlementCurrency: settle,
				Multiplier:         dContract.QuantoMultiplier.Float64(),
				MaxLeverage:        dContract.LeverageMax.Float64(),
			}
		}
		return resp, nil
	}
	return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	var l []limits.MinMaxLevel
	switch a {
	case asset.Spot:
		pairsData, err := e.ListSpotCurrencyPairs(ctx)
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(pairsData))
		for _, cpDetail := range pairsData {
			if cpDetail.TradeStatus == "untradable" {
				continue
			}
			// Minimum base amounts are not always provided this will default to
			// precision for base deployment. This can't be done for quote.
			minBaseAmount := cpDetail.MinBaseAmount.Float64()
			if minBaseAmount == 0 {
				minBaseAmount = math.Pow10(-int(cpDetail.AmountPrecision))
			}
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, currency.NewPair(cpDetail.Base, cpDetail.Quote)),
				PriceStepIncrementSize:  math.Pow10(-int(cpDetail.PricePrecision)),
				AmountStepIncrementSize: math.Pow10(-int(cpDetail.AmountPrecision)),
				MinimumBaseAmount:       minBaseAmount,
				MinimumQuoteAmount:      cpDetail.MinQuoteAmount.Float64(),
				MaximumBaseAmount:       cpDetail.MaxBaseAmount.Float64(),
				MaximumQuoteAmount:      cpDetail.MaxQuoteAmount.Float64(),
				Delisted:                cpDetail.DelistingTime.Time(),
				Listed:                  earliestTime(time.Now(), cpDetail.SellStart.Time(), cpDetail.BuyStart.Time()),
				MultiplierUp:            cpDetail.MaximumQuoteRisePercentage.Float64(),
				MultiplierDown:          cpDetail.MaximumQuoteDeclinePercentage.Float64(),
				MarketMaxQty:            cpDetail.MarketOrderMaxStock.Float64(),
			})
		}
	case asset.Margin:
		marginPairs, err := e.GetIsolatedMarginLendingMarkets(ctx)
		if err != nil {
			return err
		}

		supported := make(map[currency.Pair]*IsolatedMarginLendingMarketResponse, len(marginPairs))
		now := time.Now()
		for i := range marginPairs {
			if !marginPairs[i].IsTradable(now) {
				continue
			}
			supported[marginPairs[i].Pair] = &marginPairs[i]
		}

		// Required for spot trading limits
		pairsData, err := e.ListSpotCurrencyPairs(ctx)
		if err != nil {
			return err
		}

		l = make([]limits.MinMaxLevel, 0, len(supported))
		unsupported := maps.Clone(supported)
		for i := range pairsData {
			mInfo, ok := supported[pairsData[i].ID]
			if !ok {
				continue
			}
			delete(unsupported, pairsData[i].ID) // Remove ids returned by the API. Any remaining will trigger a warning
			minBaseAmount := pairsData[i].MinBaseAmount.Float64()
			if minBaseAmount == 0 {
				minBaseAmount = math.Pow10(-int(pairsData[i].AmountPrecision))
			}
			delisted := mInfo.DelistedTime.Time()
			if delisted.IsZero() {
				delisted = pairsData[i].DelistingTime.Time()
			}
			l = append(l, limits.MinMaxLevel{
				Key:                      key.NewExchangeAssetPair(e.Name, a, pairsData[i].ID),
				PriceStepIncrementSize:   math.Pow10(-int(pairsData[i].PricePrecision)),
				AmountStepIncrementSize:  math.Pow10(-int(pairsData[i].AmountPrecision)),
				MinimumBaseAmount:        minBaseAmount,
				MinimumQuoteAmount:       pairsData[i].MinQuoteAmount.Float64(),
				Delisted:                 delisted,
				MinimumBorrowAmountBase:  mInfo.BaseMinimumBorrowAmount.Float64(),
				MinimumBorrowAmountQuote: mInfo.QuoteMinimumBorrowAmount.Float64(),
			})
		}
		if len(unsupported) > 0 {
			unsupportedPairs := currency.Pairs(slices.Collect(maps.Keys(unsupported))).Strings()
			slices.Sort(unsupportedPairs)
			log.Warnf(log.ExchangeSys, "%s %d unsupported margin pairs found, no execution limits loaded for: %v", e.Name, len(unsupportedPairs), unsupportedPairs)
		}
	case asset.CrossMargin:
		crossMinimums, err := e.getCrossMarginMinimums(ctx)
		if err != nil {
			return err
		}

		pairsData, err := e.ListSpotCurrencyPairs(ctx)
		if err != nil {
			return err
		}

		l = make([]limits.MinMaxLevel, 0, len(pairsData))
		for i := range pairsData {
			if pairsData[i].TradeStatus == "untradable" {
				continue
			}
			baseMinimum, baseOK := crossMinimums[pairsData[i].Base]
			quoteMinimum, quoteOK := crossMinimums[pairsData[i].Quote]
			if !baseOK || !quoteOK {
				continue
			}
			minBaseAmount := pairsData[i].MinBaseAmount.Float64()
			if minBaseAmount == 0 {
				minBaseAmount = math.Pow10(-int(pairsData[i].AmountPrecision))
			}
			l = append(l, limits.MinMaxLevel{
				Key:                      key.NewExchangeAssetPair(e.Name, a, pairsData[i].ID),
				PriceStepIncrementSize:   math.Pow10(-int(pairsData[i].PricePrecision)),
				AmountStepIncrementSize:  math.Pow10(-int(pairsData[i].AmountPrecision)),
				MinimumBaseAmount:        minBaseAmount,
				MinimumQuoteAmount:       pairsData[i].MinQuoteAmount.Float64(),
				Delisted:                 pairsData[i].DelistingTime.Time(),
				MinimumBorrowAmountBase:  baseMinimum,
				MinimumBorrowAmountQuote: quoteMinimum,
			})
		}
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		settlement := currency.USDT
		if a == asset.CoinMarginedFutures {
			settlement = currency.BTC
		}
		contractInfo, err := e.GetAllFutureContracts(ctx, settlement, 0, 0)
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(contractInfo))
		for _, fContract := range contractInfo {
			pd, err := priceDivisor(a, fContract.Name)
			if err != nil {
				return err
			}

			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, fContract.Name),
				MinimumBaseAmount:       fContract.OrderSizeMin.Float64(),
				MaximumBaseAmount:       fContract.OrderSizeMax.Float64(),
				PriceStepIncrementSize:  fContract.OrderPriceRound.Float64(),
				AmountStepIncrementSize: 1, // 1 Contract
				MultiplierDecimal:       fContract.QuantoMultiplier.Float64(),
				PriceDivisor:            pd,
				Delisting:               fContract.DelistingTime.Time(),
				Delisted:                fContract.DelistedTime.Time(),
				Listed:                  fContract.LaunchTime.Time(),
			})
		}
	case asset.DeliveryFutures:
		for _, settlement := range []currency.Code{currency.BTC, currency.USDT} {
			contractInfo, err := e.GetAllDeliveryContracts(ctx, settlement)
			if err != nil {
				return err
			}
			l = slices.Grow(l, len(contractInfo))
			for _, dContract := range contractInfo {
				p := strings.ToUpper(dContract.Name)
				cp, err := currency.NewPairFromString(p)
				if err != nil {
					return err
				}
				pd, err := priceDivisor(a, cp)
				if err != nil {
					return err
				}
				l = append(l, limits.MinMaxLevel{
					Key:                     key.NewExchangeAssetPair(e.Name, a, cp),
					MinimumBaseAmount:       float64(dContract.OrderSizeMin),
					MaximumBaseAmount:       float64(dContract.OrderSizeMax),
					PriceStepIncrementSize:  dContract.OrderPriceRound.Float64(),
					AmountStepIncrementSize: 1,
					Expiry:                  dContract.ExpireTime.Time(),
					MultiplierDecimal:       dContract.QuantoMultiplier.Float64(),
					PriceDivisor:            pd,
				})
			}
		}
	case asset.Options:
		underlyings, err := e.GetAllOptionsUnderlyings(ctx)
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0)
		for x := range underlyings {
			contracts, err := e.GetAllContractOfUnderlyingWithinExpiryDate(ctx, underlyings[x].Name, time.Time{})
			if err != nil {
				return err
			}
			l = slices.Grow(l, len(contracts))
			for _, optionContract := range contracts {
				cp, err := currency.NewPairFromString(strings.ReplaceAll(optionContract.Name, currency.DashDelimiter, currency.UnderscoreDelimiter))
				if err != nil {
					return err
				}

				pd, err := priceDivisor(a, cp)
				if err != nil {
					return err
				}

				cp.Quote = currency.NewCode(strings.ReplaceAll(cp.Quote.String(), currency.UnderscoreDelimiter, currency.DashDelimiter))
				l = append(l, limits.MinMaxLevel{
					Key:                     key.NewExchangeAssetPair(e.Name, a, cp),
					MinimumBaseAmount:       float64(optionContract.OrderSizeMin),
					MaximumBaseAmount:       float64(optionContract.OrderSizeMax),
					PriceStepIncrementSize:  optionContract.OrderPriceRound.Float64(),
					AmountStepIncrementSize: 1,
					MultiplierDecimal:       optionContract.Multiplier.Float64(),
					PriceDivisor:            pd,
				})
			}
		}
		return limits.Load(l)
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}

	return limits.Load(l)
}

// earliestTime returns the earliest non-zero time before now from a list of times. If no such time exists, it returns a zero time.
func earliestTime(now time.Time, times ...time.Time) time.Time {
	var earliest time.Time
	for _, ts := range times {
		if ts.IsZero() || !ts.Before(now) {
			continue
		}
		if earliest.IsZero() || ts.Before(earliest) {
			earliest = ts
		}
	}
	return earliest
}

// MBABYDOGE price is 1e6 x spot price for futures contracts. This is the only currency that has this characteristic.
var divisorCurrency = currency.NewCode("MBABYDOGE")

// priceDivisor returns the price divisor for a given asset and currency pair
func priceDivisor(a asset.Item, p currency.Pair) (float64, error) {
	if !p.Base.Equal(divisorCurrency) {
		return 1, nil
	}
	if a.IsFutures() {
		return 1e6, nil
	}
	return 0, fmt.Errorf("price divisor %w: %q %q", currency.ErrCurrencyNotSupported, p, a)
}

// GetHistoricalFundingRates returns historical funding rates for a futures contract
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.CoinMarginedFutures && r.Asset != asset.USDTMarginedFutures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}

	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	if !r.StartDate.IsZero() && !r.EndDate.IsZero() {
		if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
			return nil, err
		}
	}

	// NOTE: Opted to fail here as a misconfigured request will result in
	// {"label":"CONTRACT_NOT_FOUND"} and rather not mutate request using
	// quote currency as the settlement currency.
	if r.PaymentCurrency.IsEmpty() {
		return nil, fundingrate.ErrPaymentCurrencyCannotBeEmpty
	}

	if r.IncludePayments {
		return nil, fmt.Errorf("include payments %w", common.ErrNotYetImplemented)
	}

	if r.IncludePredictedRate {
		return nil, fmt.Errorf("include predicted rate %w", common.ErrNotYetImplemented)
	}

	fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}

	records, err := e.GetFutureFundingRates(ctx, r.PaymentCurrency, fPair, 1000, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}

	if !r.StartDate.IsZero() && !r.RespectHistoryLimits && r.StartDate.Before(records[len(records)-1].Timestamp.Time()) {
		return nil, fmt.Errorf("%w start date requested: %v last returned record: %v", fundingrate.ErrFundingRateOutsideLimits, r.StartDate, records[len(records)-1].Timestamp.Time())
	}

	fundingRates := make([]fundingrate.Rate, 0, len(records))
	for _, fFundingRate := range records {
		if (!r.EndDate.IsZero() && r.EndDate.Before(fFundingRate.Timestamp.Time())) ||
			(!r.StartDate.IsZero() && r.StartDate.After(fFundingRate.Timestamp.Time())) {
			continue
		}

		fundingRates = append(fundingRates, fundingrate.Rate{
			Rate: decimal.MustFromFloat(fFundingRate.Rate.Float64()),
			Time: fFundingRate.Timestamp.Time(),
		})
	}

	if len(fundingRates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}

	return &fundingrate.HistoricalRates{
		Exchange:        e.Name,
		Asset:           r.Asset,
		Pair:            r.Pair,
		FundingRates:    fundingRates,
		StartDate:       fundingRates[len(fundingRates)-1].Time,
		EndDate:         fundingRates[0].Time,
		LatestRate:      fundingRates[0],
		PaymentCurrency: r.PaymentCurrency,
	}, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.CoinMarginedFutures && r.Asset != asset.USDTMarginedFutures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}

	settle, err := getSettlementCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}

	if !r.Pair.IsEmpty() {
		fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
		if err != nil {
			return nil, err
		}
		contract, err := e.GetFuturesContract(ctx, settle, fPair.String())
		if err != nil {
			return nil, err
		}
		return []fundingrate.LatestRateResponse{
			contractToFundingRate(e.Name, r.Asset, fPair, contract, r.IncludePredictedRate),
		}, nil
	}

	pairs, err := e.GetEnabledPairs(r.Asset)
	if err != nil {
		return nil, err
	}

	contracts, err := e.GetAllFutureContracts(ctx, settle, 0, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]fundingrate.LatestRateResponse, 0, len(contracts))
	for _, fContract := range contracts {
		if !pairs.Contains(fContract.Name, true) {
			continue
		}
		resp = append(resp, contractToFundingRate(e.Name, r.Asset, fContract.Name, fContract, r.IncludePredictedRate))
	}

	return slices.Clip(resp), nil
}

func contractToFundingRate(name string, item asset.Item, fPair currency.Pair, contract *FuturesContractResponse, includeUpcomingRate bool) fundingrate.LatestRateResponse {
	resp := fundingrate.LatestRateResponse{
		Exchange: name,
		Asset:    item,
		Pair:     fPair,
		LatestRate: fundingrate.Rate{
			Time: contract.FundingNextApply.Time().Add(-contract.fundingInterval()),
			Rate: contract.FundingRate.Decimal(),
		},
		TimeOfNextRate: contract.FundingNextApply.Time(),
		TimeChecked:    time.Now(),
	}
	if includeUpcomingRate {
		resp.PredictedUpcomingRate = fundingrate.Rate{
			Time: contract.FundingNextApply.Time(),
			Rate: contract.FundingRateIndicative.Decimal(),
		}
	}
	return resp
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (e *Exchange) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, amount float64, _ order.Side) error {
	switch item {
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		if pair.IsEmpty() {
			return currency.ErrCurrencyPairEmpty
		}
		if amount <= 0 {
			return fmt.Errorf("%w: %f", errInvalidLeverage, amount)
		}
		fPair, err := e.FormatExchangeCurrency(pair, item)
		if err != nil {
			return err
		}
		settle, err := getSettlementCurrency(fPair, item)
		if err != nil {
			return err
		}
		if item == asset.DeliveryFutures {
			switch marginType {
			case margin.Isolated, margin.Unset:
				_, err = e.UpdateDeliveryPositionLeverage(ctx, settle, fPair, amount)
				return err
			default:
				return fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
			}
		}

		leverage := amount
		crossLeverageLimit := 0.0
		switch marginType {
		case margin.Multi:
			leverage = 0
			crossLeverageLimit = amount
		case margin.Isolated, margin.Unset:
			// defaults already set
		default:
			return fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
		}

		_, err = e.UpdateFuturesPositionLeverage(ctx, settle, fPair, leverage, crossLeverageLimit, 0)
		return err
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.CoinMarginedFutures || a == asset.USDTMarginedFutures, nil
}

// GetOpenInterest returns the open interest rate for a given asset pair
// If no pairs are provided, all enabled assets and pairs will be used
// If keys are provided, those asset pairs only need to be available, not enabled
func (e *Exchange) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
	var errs error
	resp := make([]futures.OpenInterest, 0, len(keys))
	assets := asset.Items{}
	if len(keys) == 0 {
		assets = asset.Items{asset.DeliveryFutures, asset.CoinMarginedFutures, asset.USDTMarginedFutures}
	} else {
		for _, k := range keys {
			if !slices.Contains(assets, k.Asset) {
				assets = append(assets, k.Asset)
			}
		}
	}
	for _, a := range assets {
		requestedPair, err := getRequestedOpenInterestPair(e, keys, a)
		if err != nil {
			return nil, err
		}
		contracts, err := e.getOpenInterestContracts(ctx, a, requestedPair)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w fetching %s", err, a))
			continue
		}
		for _, c := range contracts {
			pair := requestedPair
			if pair.IsEmpty() { // If not exactly one key provided
				pair, err = e.MatchSymbolWithAvailablePairs(c.contractName(), a, true)
				if err != nil {
					if err := common.ExcludeError(err, currency.ErrPairNotFound); err != nil {
						errs = common.AppendError(errs, fmt.Errorf("%w from %s contract %s", err, a, c.contractName()))
					}
					continue
				}
				if len(keys) == 0 { // No keys: All enabled pairs
					if enabled, err := e.IsPairEnabled(pair, a); err != nil {
						errs = common.AppendError(errs, fmt.Errorf("%w: %s %s", err, a, pair))
						continue
					} else if !enabled {
						continue
					}
				} else { // More than one key; Any available pair
					if !slices.ContainsFunc(keys, func(k key.PairAsset) bool { return a == k.Asset && k.Pair().Equal(pair) }) {
						continue
					}
				}
			}

			resp = append(resp, futures.OpenInterest{
				Key: key.ExchangeAssetPair{
					Exchange: e.Name,
					Base:     pair.Base.Item,
					Quote:    pair.Quote.Item,
					Asset:    a,
				},
				OpenInterest: c.openInterest(),
			})
		}
	}
	return slices.Clip(resp), errs
}

type openInterestContract interface {
	openInterest() float64
	contractName() string
}

// openInterest returns the open interest in the contract's quote currency. An inverse contract is
// worth one unit of that currency, which is why Gate reports quanto_multiplier "0" for it: the
// position size is already the figure, and scaling it by the index price would restate a total
// that is already in USD, inflating it by the whole index price
func (c *FuturesContractResponse) openInterest() float64 {
	if c.Type == contractTypeInverse {
		return c.PositionSize.Float64()
	}
	// ordered as the multiply this replaces was, so a direct contract keeps its exact float64 result
	return c.PositionSize.Float64() * c.IndexPrice.Float64() * c.QuantoMultiplier.Float64()
}

// fundingInterval returns the contract's funding interval, which Gate gives in seconds; the cap keeps an out of range
// value from overflowing the duration
func (c *FuturesContractResponse) fundingInterval() time.Duration {
	return time.Duration(min(c.FundingInterval, uint64(math.MaxInt64/time.Second))) * time.Second
}

func (c *FuturesContractResponse) contractName() string {
	return c.Name.String()
}

// openInterest matches FuturesContractResponse's, so an inverse delivery contract could not be inflated by
// the index price were Gate to list one
func (c *DeliveryContractResponse) openInterest() float64 {
	if c.Type == contractTypeInverse {
		return c.PositionSize.Float64()
	}
	return c.PositionSize.Float64() * c.IndexPrice.Float64() * c.QuantoMultiplier.Float64()
}

func (c *DeliveryContractResponse) contractName() string {
	return c.Name
}

func getRequestedOpenInterestPair(e *Exchange, keys []key.PairAsset, a asset.Item) (currency.Pair, error) {
	if len(keys) != 1 || keys[0].Asset != a {
		return currency.EMPTYPAIR, nil
	}
	return e.MatchSymbolWithAvailablePairs(keys[0].Pair().String(), a, false)
}

func (e *Exchange) getOpenInterestContracts(ctx context.Context, a asset.Item, p currency.Pair) ([]openInterestContract, error) {
	if err := validateFuturesAsset(a); err != nil {
		return nil, err
	}
	settle, err := getSettlementCurrency(p, a)
	if err != nil {
		return nil, err
	}
	if a == asset.DeliveryFutures {
		if p != currency.EMPTYPAIR {
			d, err := e.GetDeliveryContract(ctx, settle, p)
			return []openInterestContract{d}, err
		}
		deliveryContracts, err := e.GetAllDeliveryContracts(ctx, settle)
		contracts := make([]openInterestContract, len(deliveryContracts))
		for i, dContract := range deliveryContracts {
			contracts[i] = dContract
		}
		return contracts, err
	}
	if p != currency.EMPTYPAIR {
		contract, err := e.GetFuturesContract(ctx, settle, p.String())
		return []openInterestContract{contract}, err
	}
	futuresContracts, err := e.GetAllFutureContracts(ctx, settle, 0, 0)
	contracts := make([]openInterestContract, len(futuresContracts))
	for i, fContract := range futuresContracts {
		contracts[i] = fContract
	}
	return contracts, err
}

// getClientOrderIDFromText returns the client order ID from the text response
func getClientOrderIDFromText(text string) string {
	if strings.HasPrefix(text, "t-") {
		return text
	}
	return ""
}

// getTypeFromTimeInForce returns the order type and if the order is post only
func getTypeFromTimeInForce(tif string, price float64) (orderType order.Type) {
	if price == 0 {
		switch tif {
		case iocTIF, fokTIF:
			return order.Market
		}
	}
	return order.Limit
}

// getSideAndAmountFromSize returns the order side, amount and remaining amounts
func getSideAndAmountFromSize(size, left float64) (side order.Side, amount, remaining float64) {
	if size < 0 {
		return order.Short, math.Abs(size), math.Abs(left)
	}
	return order.Long, size, left
}

// tradeSideAndAmount splits a public futures or options trade size, whose sign Gate uses for the taker's side, into
// that side and an unsigned amount
func tradeSideAndAmount(size float64) (side order.Side, amount float64) {
	if size < 0 {
		return order.Sell, -size
	}
	return order.Buy, size
}

// getFutureOrderSize sets the amount to a negative value if shorting.
func getFutureOrderSize(s *order.Submit) (float64, error) {
	switch {
	case s.Side.IsLong():
		return s.Amount, nil
	case s.Side.IsShort():
		return -s.Amount, nil
	default:
		return 0, order.ErrSideIsInvalid
	}
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.UnderscoreDelimiter
	switch a {
	case asset.Spot, asset.CrossMargin, asset.Margin:
		return tradeBaseURL + "trade/" + cp.Upper().String(), nil
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(cp, a)
		if err != nil {
			return "", err
		}
		if a == asset.DeliveryFutures {
			return tradeBaseURL + "futures-delivery/" + settle.String() + "/" + cp.Upper().String(), nil
		}
		return tradeBaseURL + "futures/" + settle.String() + "/" + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// WebsocketSubmitOrder submits an order to the exchange
// NOTE: Regarding spot orders, fee is applied to purchased currency.
func (e *Exchange) WebsocketSubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate(e.GetTradingRequirements())
	if err != nil {
		return nil, err
	}

	if err := e.formatOrderClientIDAndPair(s); err != nil {
		return nil, err
	}

	switch s.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		req, err := e.getSpotOrderRequest(s)
		if err != nil {
			return nil, err
		}

		resp, err := e.WebsocketSpotSubmitOrder(ctx, req)
		if err != nil {
			return nil, err
		}
		return e.deriveSpotWebsocketOrderResponse(resp)
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		req, err := getFuturesOrderRequest(s)
		if err != nil {
			return nil, err
		}
		resp, err := e.WebsocketFuturesSubmitOrder(ctx, s.AssetType, req)
		if err != nil {
			return nil, err
		}
		return e.deriveFuturesWebsocketOrderResponse(resp, s.AssetType)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, s.AssetType)
	}
}

func formatClientOrderID(clientOrderID string) string {
	if clientOrderID == "" || strings.HasPrefix(clientOrderID, "t-") {
		return clientOrderID
	}
	return "t-" + clientOrderID
}

func (e *Exchange) formatOrderClientIDAndPair(s *order.Submit) error {
	s.ClientOrderID = formatClientOrderID(s.ClientOrderID)
	var err error
	s.Pair, err = e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return err
	}
	s.Pair = s.Pair.Upper()
	return nil
}

// orderPriceAndTIF returns the price and time in force to send for an order. Gate takes a market order without a price
// and only as IOC or FOK. A contract order has no order type field, so a zero price is what makes it a market order and
// any other type would go out as a plain limit order; spot and margin orders are only limit or market orders.
// Conditional orders go through the price triggered order endpoints instead
func orderPriceAndTIF(s *order.Submit) (price float64, tif string, err error) {
	switch s.Type {
	case order.Limit:
		price = s.Price
	case order.Market:
	default:
		return 0, "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, s.Type)
	}
	if tif, err = toExchangeTIF(s.TimeInForce, price); err != nil {
		return 0, "", err
	}
	if s.Type == order.Market && tif != iocTIF && tif != fokTIF {
		return 0, "", fmt.Errorf("%w: %q; a market order must be IOC or FOK", order.ErrUnsupportedTimeInForce, s.TimeInForce)
	}
	return price, tif, nil
}

func getFuturesOrderRequest(s *order.Submit) (*FuturesOrderCreateRequest, error) {
	amountWithDirection, err := getFutureOrderSize(s)
	if err != nil {
		return nil, err
	}
	price, tif, err := orderPriceAndTIF(s)
	if err != nil {
		return nil, err
	}
	return &FuturesOrderCreateRequest{
		Contract:    s.Pair,
		Size:        types.Number(amountWithDirection),
		Price:       types.Number(price),
		ReduceOnly:  s.ReduceOnly,
		TimeInForce: tif,
		Text:        s.ClientOrderID,
	}, nil
}

func getDeliveryOrderRequest(s *order.Submit) (*DeliveryOrderCreateRequest, error) {
	amountWithDirection, err := getFutureOrderSize(s)
	if err != nil {
		return nil, err
	}
	price, tif, err := orderPriceAndTIF(s)
	if err != nil {
		return nil, err
	}
	return &DeliveryOrderCreateRequest{
		Contract:    s.Pair,
		Size:        types.Number(amountWithDirection),
		Price:       types.Number(price),
		ReduceOnly:  s.ReduceOnly,
		TimeInForce: tif,
		Text:        s.ClientOrderID,
	}, nil
}

// toExchangeTIF converts a TimeInForce to its corresponding exchange-compatible string.
func toExchangeTIF(tif order.TimeInForce, price float64) (string, error) {
	switch {
	case tif == order.UnknownTIF:
		if price == 0 {
			return iocTIF, nil // Market orders default to IOC
		}
		return gtcTIF, nil // Default to GTC for limit orders
	case tif.Is(order.PostOnly):
		return pocTIF, nil
	case tif.Is(order.ImmediateOrCancel):
		return iocTIF, nil
	case tif.Is(order.FillOrKill):
		return fokTIF, nil
	case tif.Is(order.GoodTillCancel):
		return gtcTIF, nil
	default:
		return "", fmt.Errorf("%w: %q", order.ErrUnsupportedTimeInForce, tif)
	}
}

func (e *Exchange) deriveSpotWebsocketOrderResponse(responses *WebsocketOrderResponse) (*order.SubmitResponse, error) {
	resp, err := e.deriveSpotWebsocketOrderResponses([]*WebsocketOrderResponse{responses})
	if err != nil {
		return nil, err
	}
	return resp[0], nil
}

// deriveSpotWebsocketOrderResponses returns the order submission responses for spot
func (e *Exchange) deriveSpotWebsocketOrderResponses(responses []*WebsocketOrderResponse) ([]*order.SubmitResponse, error) {
	if len(responses) == 0 {
		return nil, common.ErrNoResponse
	}

	out := make([]*order.SubmitResponse, len(responses))
	for i, resp := range responses {
		if resp.Label != "" { // batch only, denotes error type in string format
			out[i] = &order.SubmitResponse{
				Exchange:        e.Name,
				ClientOrderID:   resp.Text,
				SubmissionError: fmt.Errorf("%w reason label:%q message:%q", order.ErrUnableToPlaceOrder, resp.Label, resp.Message),
			}
			continue
		}

		side, err := order.StringToOrderSide(resp.Side)
		if err != nil {
			return nil, err
		}
		status, err := orderStatusFromFinishAs(resp.FinishAs)
		if err != nil {
			return nil, err
		}
		oType, err := order.StringToOrderType(resp.Type)
		if err != nil {
			return nil, err
		}

		var cost float64
		var purchased float64
		if resp.AverageDealPrice != 0 {
			if side.IsLong() {
				cost = resp.FilledTotal.Float64()
				purchased = resp.FilledTotal.Decimal().Div(resp.AverageDealPrice.Decimal()).InexactFloat64()
			} else {
				cost = resp.Amount.Float64()
				purchased = resp.FilledTotal.Float64()
			}
		}
		tif, err := order.StringToTimeInForce(resp.TimeInForce)
		if err != nil {
			return nil, err
		}
		a, err := spotOrderAsset(resp.Account)
		if err != nil {
			return nil, err
		}
		if side.IsLong() && resp.FilledAmount != 0 {
			purchased = resp.FilledAmount.Float64() // Exact, where dividing by the rounded average price is not
		}
		out[i] = &order.SubmitResponse{
			Exchange:             e.Name,
			OrderID:              resp.ID,
			AssetType:            a,
			Pair:                 resp.CurrencyPair,
			ClientOrderID:        resp.Text,
			Date:                 resp.CreateTimeMs.Time(),
			LastUpdated:          resp.UpdateTimeMs.Time(),
			RemainingAmount:      resp.Left.Float64(),
			Amount:               resp.Amount.Float64(),
			Price:                resp.Price.Float64(),
			Type:                 oType,
			Side:                 side,
			Fee:                  resp.Fee.Float64(),
			FeeAsset:             resp.FeeCurrency,
			TimeInForce:          tif,
			Cost:                 cost,
			Purchased:            purchased,
			Status:               status,
			AverageExecutedPrice: resp.AverageDealPrice.Float64(),
		}
	}
	return out, nil
}

func (e *Exchange) deriveFuturesWebsocketOrderResponse(responses *WebsocketFuturesOrderResponse, a asset.Item) (*order.SubmitResponse, error) {
	resp, err := e.deriveFuturesWebsocketOrderResponses([]*WebsocketFuturesOrderResponse{responses}, a)
	if err != nil {
		return nil, err
	}
	return resp[0], nil
}

// deriveFuturesWebsocketOrderResponses returns the order submission responses for futures
func (e *Exchange) deriveFuturesWebsocketOrderResponses(responses []*WebsocketFuturesOrderResponse, a asset.Item) ([]*order.SubmitResponse, error) {
	if len(responses) == 0 {
		return nil, common.ErrNoResponse
	}

	out := make([]*order.SubmitResponse, 0, len(responses))
	for _, resp := range responses {
		if resp.Succeeded != nil && !*resp.Succeeded { // batch only, the order was rejected
			out = append(out, &order.SubmitResponse{
				Exchange:        e.Name,
				AssetType:       a,
				ClientOrderID:   getClientOrderIDFromText(resp.Text),
				SubmissionError: fmt.Errorf("%w reason label:%q detail:%q", order.ErrUnableToPlaceOrder, resp.Label, resp.Detail),
			})
			continue
		}
		status, err := orderStatusFromFinishAs(resp.FinishAs)
		if err != nil {
			return nil, err
		}

		oType := order.Market
		if resp.Price != 0 {
			oType = order.Limit
		}

		side := order.Long
		if resp.Size < 0 {
			side = order.Short
		}

		tif, err := order.StringToTimeInForce(resp.TimeInForce)
		if err != nil {
			return nil, err
		}
		out = append(out, &order.SubmitResponse{
			Exchange:             e.Name,
			OrderID:              strconv.FormatUint(resp.ID, 10),
			AssetType:            a,
			Pair:                 resp.Contract,
			ClientOrderID:        getClientOrderIDFromText(resp.Text),
			Date:                 resp.CreateTime.Time(),
			LastUpdated:          orderLastUpdated(resp.UpdateTime, resp.FinishTime, resp.CreateTime),
			RemainingAmount:      math.Abs(resp.Left.Float64()),
			Amount:               math.Abs(resp.Size.Float64()),
			Price:                resp.Price.Float64(),
			AverageExecutedPrice: resp.FillPrice.Float64(),
			Type:                 oType,
			Side:                 side,
			Status:               status,
			TimeInForce:          tif,
			ReduceOnly:           resp.IsReduceOnly,
		})
	}
	return out, nil
}

func (e *Exchange) getSpotOrderRequest(s *order.Submit) (*CreateOrderRequest, error) {
	var side order.Side
	switch {
	case s.Side.IsLong():
		side = order.Buy
	case s.Side.IsShort():
		side = order.Sell
	default:
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, s.Side)
	}

	// A market order that carries a price, as the backtester's live market orders do, would otherwise go out as GTC,
	// which Gate refuses for market orders
	price, tif, err := orderPriceAndTIF(s)
	if err != nil {
		return nil, err
	}

	amount := s.GetTradeAmount(e.GetTradingRequirements())
	// Gate sizes a market buy in the quote currency on margin accounts too, which the trading requirements only cover
	// for spot
	if s.Type == order.Market && side == order.Buy && s.AssetType != asset.Spot {
		if s.QuoteAmount <= 0 {
			return nil, fmt.Errorf("%w: a %s market buy is sized by its quote amount", order.ErrAmountMustBeSet, s.AssetType)
		}
		amount = s.QuoteAmount
	}

	return &CreateOrderRequest{
		Side:         side.Lower(),
		Type:         s.Type.Lower(),
		Account:      s.AssetType,
		Amount:       types.Number(amount),
		Price:        types.Number(price),
		CurrencyPair: s.Pair,
		Text:         s.ClientOrderID,
		TimeInForce:  tif,
		AutoBorrow:   s.AutoBorrow,
		AutoRepay:    s.AutoRepay,
	}, nil
}

func getSettlementCurrency(p currency.Pair, a asset.Item) (currency.Code, error) {
	switch a {
	case asset.DeliveryFutures, asset.Options:
		return currency.USDT, nil
	case asset.USDTMarginedFutures:
		if p.IsEmpty() || p.Quote.Equal(currency.USDT) {
			return currency.USDT, nil
		}
		return currency.EMPTYCODE, fmt.Errorf("%w %s %s", errInvalidSettlementQuote, a, p)
	case asset.CoinMarginedFutures:
		if !p.IsEmpty() {
			if !p.Base.Equal(currency.BTC) { // Only BTC endpoint currently available
				return currency.EMPTYCODE, fmt.Errorf("%w %s %s", errInvalidSettlementBase, a, p)
			}
			if !p.Quote.Equal(currency.USD) { // We expect all Coin-M to be quoted in USD
				return currency.EMPTYCODE, fmt.Errorf("%w %s %s", errInvalidSettlementQuote, a, p)
			}
		}
		return currency.BTC, nil
	}
	return currency.EMPTYCODE, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
}

func validateFuturesAsset(a asset.Item) error {
	switch a {
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		return nil
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
}

// WebsocketSubmitOrders submits orders to the exchange through the websocket
func (e *Exchange) WebsocketSubmitOrders(ctx context.Context, orders []*order.Submit) ([]*order.SubmitResponse, error) {
	var a asset.Item
	for _, ord := range orders {
		if err := ord.Validate(e.GetTradingRequirements()); err != nil {
			return nil, err
		}
		if a == asset.Empty {
			a = ord.AssetType
			continue
		}
		if a != ord.AssetType {
			return nil, fmt.Errorf("%w; Passed %q and %q", errSingleAssetRequired, a, ord.AssetType)
		}
	}

	if !e.CurrencyPairs.IsAssetSupported(a) {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}

	switch a {
	case asset.Spot:
		reqs := make([]*CreateOrderRequest, len(orders))
		for i, ord := range orders {
			if err := e.formatOrderClientIDAndPair(ord); err != nil {
				return nil, err
			}
			var err error
			reqs[i], err = e.getSpotOrderRequest(ord)
			if err != nil {
				return nil, err
			}
		}
		resp, err := e.WebsocketSpotSubmitOrders(ctx, reqs...)
		if err != nil {
			return nil, err
		}
		return e.deriveSpotWebsocketOrderResponses(resp)
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		reqs := make([]*FuturesOrderCreateRequest, len(orders))
		for i, s := range orders {
			if err := e.formatOrderClientIDAndPair(s); err != nil {
				return nil, err
			}
			var err error
			reqs[i], err = getFuturesOrderRequest(s)
			if err != nil {
				return nil, err
			}
		}
		resp, err := e.WebsocketFuturesSubmitOrders(ctx, a, reqs...)
		if err != nil {
			return nil, err
		}
		return e.deriveFuturesWebsocketOrderResponses(resp, a)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
}

// MessageID returns a unique ID conforming to Gate's max length of 32 bytes for request IDs
func (e *Exchange) MessageID() string {
	u := uuid.NewV7()
	var buf [32]byte
	hex.Encode(buf[:], u[:])
	return string(buf[:])
}
