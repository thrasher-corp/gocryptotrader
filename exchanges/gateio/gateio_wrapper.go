package gateio

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
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
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(packageRateLimits),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	if err := e.DisableAssetWebsocketSupport(asset.Margin); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	// TODO: Add websocket cross margin support.
	if err := e.DisableAssetWebsocketSupport(asset.CrossMargin); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              gateioTradeURL,
		exchange.RestFutures:           gateioFuturesLiveTradingAlternative,
		exchange.RestSpotSupplementary: gateioFuturesTestnetTrading,
		exchange.WebsocketSpot:         gateioWebsocketEndpoint,
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
	e.wsOBUpdateMgr = newWsOBUpdateManager(defaultWsOrderbookUpdateTimeDelay, defaultWSOrderbookUpdateDeadline)
	e.wsOBResubMgr = newWSOBResubManager()
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
		FillsFeed:                    e.Features.Enabled.FillsFeed,
		TradeFeed:                    e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement: true,
		RateLimitDefinitions:         packageRateLimits,
	}); err != nil {
		return err
	}
	// Spot connection
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   gateioWebsocketEndpoint,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Handler:               e.WsHandleSpotData,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptionsSpot,
		Connector:             e.WsConnectSpot,
		Authenticate:          e.authenticateSpot,
		MessageFilter:         asset.Spot,
	}); err != nil {
		return err
	}
	// Futures connection - USDT margined
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  usdtFuturesWebsocketURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, conn websocket.Connection, incoming []byte) error {
			return e.WsHandleFuturesData(ctx, conn, incoming, asset.USDTMarginedFutures)
		},
		Subscriber:   e.FuturesSubscribe,
		Unsubscriber: e.FuturesUnsubscribe,
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.GenerateFuturesDefaultSubscriptions(asset.USDTMarginedFutures)
		},
		Connector:     e.WsFuturesConnect,
		Authenticate:  e.authenticateFutures,
		MessageFilter: asset.USDTMarginedFutures,
	}); err != nil {
		return err
	}

	// Futures connection - BTC margined
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  btcFuturesWebsocketURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, conn websocket.Connection, incoming []byte) error {
			return e.WsHandleFuturesData(ctx, conn, incoming, asset.CoinMarginedFutures)
		},
		Subscriber:   e.FuturesSubscribe,
		Unsubscriber: e.FuturesUnsubscribe,
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.GenerateFuturesDefaultSubscriptions(asset.CoinMarginedFutures)
		},
		Connector:     e.WsFuturesConnect,
		MessageFilter: asset.CoinMarginedFutures,
	}); err != nil {
		return err
	}

	// Futures connection - Delivery - BTC margined
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  deliveryRealBTCTradingURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, conn websocket.Connection, incoming []byte) error {
			return e.WsHandleFuturesData(ctx, conn, incoming, asset.BTCMarginedDeliveryFutures)
		},
		Subscriber:   e.DeliveryFuturesSubscribe,
		Unsubscriber: e.DeliveryFuturesUnsubscribe,
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.GenerateDeliveryFuturesDefaultSubscriptions(asset.BTCMarginedDeliveryFutures)
		},
		Connector: func(ctx context.Context, conn websocket.Connection) error {
			return e.WsDeliveryFuturesConnect(ctx, conn, asset.BTCMarginedDeliveryFutures)
		},
		MessageFilter: asset.BTCMarginedDeliveryFutures,
	}); err != nil {
		return err
	}

	// Futures connection - Delivery - USDT margined
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  deliveryRealUSDTTradingURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, conn websocket.Connection, incoming []byte) error {
			return e.WsHandleFuturesData(ctx, conn, incoming, asset.DeliveryFutures)
		},
		Subscriber:   e.DeliveryFuturesSubscribe,
		Unsubscriber: e.DeliveryFuturesUnsubscribe,
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.GenerateDeliveryFuturesDefaultSubscriptions(asset.DeliveryFutures)
		},
		Connector: func(ctx context.Context, conn websocket.Connection) error {
			return e.WsDeliveryFuturesConnect(ctx, conn, asset.DeliveryFutures)
		},
		MessageFilter: asset.DeliveryFutures,
	}); err != nil {
		return err
	}

	// Futures connection - Options
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   optionsWebsocketURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Handler:               e.WsHandleOptionsData,
		Subscriber:            e.OptionsSubscribe,
		Unsubscriber:          e.OptionsUnsubscribe,
		GenerateSubscriptions: e.GenerateOptionsDefaultSubscriptions,
		Connector:             e.WsOptionsConnect,
		MessageFilter:         asset.Options,
	})
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
			Low:          tickerNew.Low24H.Float64(),
			High:         tickerNew.High24H.Float64(),
			Bid:          tickerNew.HighestBid.Float64(),
			Ask:          tickerNew.LowestAsk.Float64(),
			Last:         tickerNew.Last.Float64(),
			ExchangeName: e.Name,
			AssetType:    a,
		}
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(fPair, a)
		if err != nil {
			return nil, err
		}
		var tickers []*FuturesTicker
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
			Low:          tickers[0].Low24H.Float64(),
			High:         tickers[0].High24H.Float64(),
			Last:         tickers[0].Last.Float64(),
			Volume:       tickers[0].Volume24HBase.Float64(),
			QuoteVolume:  tickers[0].Volume24HQuote.Float64(),
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
				Bid:          tkr.Bid1Price.Float64(),
				Ask:          tkr.Ask1Price.Float64(),
				AskSize:      tkr.Ask1Size,
				BidSize:      tkr.Bid1Size,
				ExchangeName: e.Name,
				AssetType:    a,
			}
			if err := ticker.ProcessTicker(tickerData); err != nil {
				return nil, err
			}
		}
		return ticker.GetTicker(e.Name, fPair, a)
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
	case asset.Margin, asset.CrossMargin:
		tradables, err := e.GetMarginSupportedCurrencyPairs(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(tradables))
		for _, mPairDetail := range tradables {
			if mPairDetail.Status == 0 {
				continue
			}
			pairs = append(pairs, mPairDetail.ID)
		}
		return pairs, nil
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
		if err != nil {
			return nil, err
		}
		contracts, err := e.GetAllFutureContracts(ctx, settle)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(contracts))
		for _, fContract := range contracts {
			if !fContract.DelistedTime.Time().IsZero() && fContract.DelistedTime.Time().Before(time.Now()) {
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
			p := strings.ToUpper(deliveryContract.Name)
			cp, err := currency.NewPairFromString(p)
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

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assets := e.GetAssetTypes(false)
	for x := range assets {
		pairs, err := e.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			panic(fmt.Errorf("%w %v", err, assets[x]))
			return err
		}
		if err := e.UpdatePairs(pairs, assets[x], false); err != nil {
			panic(fmt.Errorf("%w %v", err, assets[x]))
			return err
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
			var currencyPair currency.Pair
			currencyPair, err = currency.NewPairFromString(sTicker.CurrencyPair)
			if err != nil {
				return err
			}
			if err := ticker.ProcessTicker(&ticker.Price{
				Last:         sTicker.Last.Float64(),
				High:         sTicker.High24H.Float64(),
				Low:          sTicker.Low24H.Float64(),
				Bid:          sTicker.HighestBid.Float64(),
				Ask:          sTicker.LowestAsk.Float64(),
				QuoteVolume:  sTicker.QuoteVolume.Float64(),
				Volume:       sTicker.BaseVolume.Float64(),
				ExchangeName: e.Name,
				Pair:         currencyPair,
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
		var tickers []*FuturesTicker
		if a == asset.DeliveryFutures {
			tickers, errs = e.GetDeliveryFutureTickers(ctx, settle, currency.EMPTYPAIR)
		} else {
			tickers, errs = e.GetFuturesTickers(ctx, settle, currency.EMPTYPAIR)
		}
		for _, tkr := range tickers {
			currencyPair, err := currency.NewPairFromString(tkr.Contract)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			if err := ticker.ProcessTicker(&ticker.Price{
				Last:         tkr.Last.Float64(),
				High:         tkr.High24H.Float64(),
				Low:          tkr.Low24H.Float64(),
				Volume:       tkr.Volume24H.Float64(),
				QuoteVolume:  tkr.Volume24HQuote.Float64(),
				ExchangeName: e.Name,
				Pair:         currencyPair,
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
		for i := range pairs {
			underlying, err := e.GetUnderlyingFromCurrencyPair(pairs[i])
			if err != nil {
				return err
			}
			tickers, err := e.GetOptionsTickers(ctx, underlying.String())
			if err != nil {
				return err
			}
			for _, tkr := range tickers {
				if err := ticker.ProcessTicker(&ticker.Price{
					Last:         tkr.LastPrice.Float64(),
					Ask:          tkr.Ask1Price.Float64(),
					AskSize:      tkr.Ask1Size,
					Bid:          tkr.Bid1Price.Float64(),
					BidSize:      tkr.Bid1Size,
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
	var o *Orderbook
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
	case asset.Margin, asset.CrossMargin:
		balances, err := e.GetMarginAccountList(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		for _, marginAcc := range balances {
			subAccts[0].Balances.Set(marginAcc.Base.Currency, accounts.Balance{
				Total: marginAcc.Base.Available.Float64() + marginAcc.Base.LockedAmount.Float64(),
				Hold:  marginAcc.Base.LockedAmount.Float64(),
				Free:  marginAcc.Base.Available.Float64(),
			})
			subAccts[0].Balances.Set(marginAcc.Quote.Currency, accounts.Balance{
				Total: marginAcc.Quote.Available.Float64() + marginAcc.Quote.LockedAmount.Float64(),
				Hold:  marginAcc.Quote.LockedAmount.Float64(),
				Free:  marginAcc.Quote.Available.Float64(),
			})
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
		if err != nil {
			return nil, err
		}
		var acc *FuturesAccount
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
		balance, err := e.GetOptionAccounts(ctx)
		if err != nil {
			return nil, err
		}
		subAccts[0].Balances.Set(balance.Currency, accounts.Balance{
			Total: balance.Total.Float64(),
			Hold:  balance.Total.Float64() - balance.Available.Float64(),
			Free:  balance.Available.Float64(),
		})
	default:
		return nil, fmt.Errorf("%w asset type: %q", asset.ErrNotSupported, a)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	records, err := e.GetWithdrawalRecords(ctx, c, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return nil, err
	}
	withdrawalHistories := make([]exchange.WithdrawalHistory, len(records))
	for x, withdrawalResp := range records {
		withdrawalHistories[x] = exchange.WithdrawalHistory{
			Status:          withdrawalResp.Status,
			TransferID:      withdrawalResp.ID,
			Currency:        withdrawalResp.Currency,
			Amount:          withdrawalResp.Amount.Float64(),
			CryptoTxID:      withdrawalResp.TransactionID,
			CryptoToAddress: withdrawalResp.WithdrawalAddress,
			Timestamp:       withdrawalResp.Timestamp.Time(),
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
		tradeData, err := e.GetMarketTrades(ctx, p, 0, "", false, time.Time{}, time.Time{}, 0)
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
				TID:          td.OrderID,
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
		var futuresTrades []*TradingHistoryItem
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
			resp[i] = trade.Data{
				TID:          strconv.FormatInt(tradeH.ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        tradeH.Price.Float64(),
				Amount:       tradeH.Size,
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
			resp[i] = trade.Data{
				TID:          strconv.FormatInt(td.ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        td.Price.Float64(),
				Amount:       td.Size,
				Timestamp:    td.CreateTime.Time(),
			}
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	sort.Sort(trade.ByDate(resp))
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

	s.ClientOrderID = formatClientOrderID(s.ClientOrderID)
	var err error
	s.Pair, err = e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	s.Pair = s.Pair.Upper()

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
		response.FeeAsset = currency.NewCode(sOrder.FeeCurrency)
		response.Pair = s.Pair
		response.Date = sOrder.CreateTime.Time()
		response.ClientOrderID = sOrder.Text
		response.Date = sOrder.CreateTime.Time()
		response.LastUpdated = sOrder.UpdateTime.Time()
		return response, nil
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
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
		op := e.PlaceFuturesOrder
		if s.AssetType == asset.DeliveryFutures {
			op = e.PlaceDeliveryOrder
		}
		o, err := op(ctx, fOrder)
		if err != nil {
			return nil, err
		}
		resp, err := s.DeriveSubmitResponse(strconv.FormatInt(o.ID, 10))
		if err != nil {
			return nil, err
		}
		resp.Status = order.Open
		if o.Status != statusOpen {
			if resp.Status, err = order.StringToOrderStatus(o.FinishAs); err != nil {
				return nil, err
			}
		}
		resp.Date = o.CreateTime.Time()
		resp.ClientOrderID = getClientOrderIDFromText(o.Text)
		resp.Amount = math.Abs(o.Size)
		resp.Price = o.OrderPrice.Float64()
		resp.AverageExecutedPrice = o.FillPrice.Float64()
		return resp, nil
	case asset.Options:
		optionOrder, err := e.PlaceOptionOrder(ctx, &OptionOrderParam{
			Contract:   s.Pair,
			OrderSize:  s.Amount,
			Price:      types.Number(s.Price),
			ReduceOnly: s.ReduceOnly,
			Text:       s.ClientOrderID,
		})
		if err != nil {
			return nil, err
		}
		response, err := s.DeriveSubmitResponse(strconv.FormatInt(optionOrder.OptionOrderID, 10))
		if err != nil {
			return nil, err
		}
		status, err := order.StringToOrderStatus(optionOrder.Status)
		if err != nil {
			return nil, err
		}
		response.Status = status
		response.Pair = s.Pair
		response.Date = optionOrder.CreateTime.Time()
		response.ClientOrderID = optionOrder.Text
		return response, nil
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, s.AssetType)
	}
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
				_, err = e.CancelSingleFuturesOrder(ctx, settle, o.OrderID)
			}
		}
	case asset.Options:
		_, err = e.CancelOptionSingleOrder(ctx, o.OrderID)
	default:
		return fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	response := order.CancelBatchResponse{
		Status: map[string]string{},
	}
	if len(o) == 0 {
		return nil, errors.New("no cancel order passed")
	}
	var err error
	var cancelSpotOrdersParam []CancelOrderByIDParam
	a := o[0].AssetType
	for x := range o {
		o[x].Pair, err = e.FormatExchangeCurrency(o[x].Pair, a)
		if err != nil {
			return nil, err
		}
		o[x].Pair = o[x].Pair.Upper()
		if a != o[x].AssetType {
			return nil, errors.New("cannot cancel orders of different asset types")
		}
		if a == asset.Spot || a == asset.Margin || a == asset.CrossMargin {
			cancelSpotOrdersParam = append(cancelSpotOrdersParam, CancelOrderByIDParam{
				ID:           o[x].OrderID,
				CurrencyPair: o[x].Pair,
			})
			continue
		}
		if err := o[x].Validate(o[x].StandardCancel()); err != nil {
			return nil, err
		}
	}
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		loop := int(math.Ceil(float64(len(cancelSpotOrdersParam)) / 10))
		for count := range loop {
			var input []CancelOrderByIDParam
			if (count + 1) == loop {
				input = cancelSpotOrdersParam[count*10:]
			} else {
				input = cancelSpotOrdersParam[count*10 : (count*10)+10]
			}
			cancelResponses, err := e.CancelBatchOrdersWithIDList(ctx, input)
			if err != nil {
				return nil, err
			}
			for j, cancelResp := range cancelResponses {
				if cancelResponses[j].Succeeded {
					response.Status[cancelResp.OrderID] = order.Cancelled.String()
				}
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		for i := range o {
			settle, err := getSettlementCurrency(o[i].Pair, a)
			if err != nil {
				return nil, err
			}
			var resp []*Order
			if a == asset.DeliveryFutures {
				resp, err = e.CancelMultipleDeliveryOrders(ctx, o[i].Pair, o[i].Side.Lower(), settle)
			} else {
				resp, err = e.CancelMultipleFuturesOpenOrders(ctx, o[i].Pair, o[i].Side.Lower(), settle)
			}
			if err != nil {
				return nil, err
			}
			for _, ord := range resp {
				response.Status[strconv.FormatInt(ord.ID, 10)] = ord.Status
			}
		}
	case asset.Options:
		for i := range o {
			cancel, err := e.CancelMultipleOptionOpenOrders(ctx, o[i].Pair, o[i].Pair.String(), o[i].Side.Lower())
			if err != nil {
				return nil, err
			}
			for _, optionOrd := range cancel {
				response.Status[strconv.FormatInt(optionOrd.OptionOrderID, 10)] = optionOrd.Status
			}
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	return &response, nil
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

	var side string
	switch {
	case o.Side.IsLong():
		side = order.Bid.Lower()
	case o.Side.IsShort():
		side = order.Ask.Lower()
	case o.Side == order.UnknownSide, o.Side == order.AnySide:
	default:
		return resp, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, o.Side)
	}

	switch o.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		cancel, err := e.CancelMultipleSpotOpenOrders(ctx, fmtPair, o.AssetType)
		if err != nil {
			return resp, err
		}
		for _, sPriceTriggeredOrder := range cancel {
			resp.Add(strconv.FormatInt(sPriceTriggeredOrder.AutoOrderID, 10), sPriceTriggeredOrder.Status)
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(fmtPair, o.AssetType)
		if err != nil {
			return resp, err
		}
		var cancelResponses []*Order
		if o.AssetType == asset.DeliveryFutures {
			cancelResponses, err = e.CancelMultipleDeliveryOrders(ctx, fmtPair, side, settle)
		} else {
			cancelResponses, err = e.CancelMultipleFuturesOpenOrders(ctx, fmtPair, side, settle)
		}
		if err != nil {
			return resp, err
		}
		for _, cancelResp := range cancelResponses {
			resp.Add(strconv.FormatInt(cancelResp.ID, 10), cancelResp.FinishAs)
		}
	case asset.Options:
		var underlying currency.Pair
		if !o.Pair.IsEmpty() {
			underlying, err = e.GetUnderlyingFromCurrencyPair(o.Pair)
			if err != nil {
				return resp, err
			}
		}
		optionOrderResponse, err := e.CancelMultipleOptionOpenOrders(ctx, fmtPair, underlying.String(), side)
		if err != nil {
			return resp, err
		}
		for _, oResp := range optionOrderResponse {
			resp.Add(strconv.FormatInt(oResp.OptionOrderID, 10), oResp.FinishAs)
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
		var spotOrder *SpotOrder
		spotOrder, err = e.GetSpotOrder(ctx, orderID, pair, a)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(spotOrder.Side)
		if err != nil {
			return nil, err
		}
		var orderType order.Type
		orderType, err = order.StringToOrderType(spotOrder.Type)
		if err != nil {
			return nil, err
		}
		var orderStatus order.Status
		orderStatus, err = order.StringToOrderStatus(spotOrder.Status)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			Amount:         spotOrder.Amount.Float64(),
			Exchange:       e.Name,
			OrderID:        spotOrder.OrderID,
			Side:           side,
			Type:           orderType,
			Pair:           pair,
			Cost:           spotOrder.FeeDeducted.Float64(),
			AssetType:      a,
			Status:         orderStatus,
			Price:          spotOrder.Price.Float64(),
			ExecutedAmount: spotOrder.Amount.Float64() - spotOrder.Left.Float64(),
			Date:           spotOrder.CreateTime.Time(),
			LastUpdated:    spotOrder.UpdateTime.Time(),
		}, nil
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(pair, a)
		if err != nil {
			return nil, err
		}
		var fOrder *Order
		if a == asset.DeliveryFutures {
			fOrder, err = e.GetSingleDeliveryOrder(ctx, settle, orderID)
		} else {
			fOrder, err = e.GetSingleFuturesOrder(ctx, settle, orderID)
		}
		if err != nil {
			return nil, err
		}
		orderStatus := order.Open
		if fOrder.Status != statusOpen {
			orderStatus, err = order.StringToOrderStatus(fOrder.FinishAs)
			if err != nil {
				return nil, err
			}
		}
		pair, err = currency.NewPairFromString(fOrder.Contract)
		if err != nil {
			return nil, err
		}

		side, amount, remaining := getSideAndAmountFromSize(fOrder.Size, fOrder.RemainingAmount)
		tif, err := timeInForceFromString(fOrder.TimeInForce)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			Amount:               amount,
			ExecutedAmount:       amount - remaining,
			RemainingAmount:      remaining,
			Exchange:             e.Name,
			OrderID:              orderID,
			ClientOrderID:        getClientOrderIDFromText(fOrder.Text),
			Status:               orderStatus,
			Price:                fOrder.OrderPrice.Float64(),
			AverageExecutedPrice: fOrder.FillPrice.Float64(),
			Date:                 fOrder.CreateTime.Time(),
			LastUpdated:          fOrder.FinishTime.Time(),
			Pair:                 pair,
			AssetType:            a,
			Type:                 getTypeFromTimeInForce(fOrder.TimeInForce, fOrder.OrderPrice.Float64()),
			TimeInForce:          tif,
			Side:                 side,
		}, nil
	case asset.Options:
		optionOrder, err := e.GetSingleOptionOrder(ctx, orderID)
		if err != nil {
			return nil, err
		}
		orderStatus, err := order.StringToOrderStatus(optionOrder.Status)
		if err != nil {
			return nil, err
		}
		pair, err = currency.NewPairFromString(optionOrder.Contract)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			Amount:         optionOrder.Size,
			ExecutedAmount: optionOrder.Size - optionOrder.Left,
			Exchange:       e.Name,
			OrderID:        orderID,
			Status:         orderStatus,
			Price:          optionOrder.Price.Float64(),
			Date:           optionOrder.CreateTime.Time(),
			LastUpdated:    optionOrder.FinishTime.Time(),
			Pair:           pair,
			AssetType:      a,
		}, nil
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
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
					Tag:     addrMultiChain.PaymentName,
				}, nil
			}
		}
		return nil, fmt.Errorf("network %s not found", chain)
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
	response, err := e.WithdrawCurrency(ctx, &WithdrawalRequestParam{
		Amount:   types.Number(withdrawRequest.Amount),
		Currency: withdrawRequest.Currency,
		Address:  withdrawRequest.Crypto.Address,
		Chain:    withdrawRequest.Crypto.Chain,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   response.Chain,
		ID:     response.TransactionID,
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
		spotOrders, err := e.GetSpotOpenOrders(ctx, 0, 0, req.AssetType == asset.CrossMargin)
		if err != nil {
			return nil, err
		}
		for _, sOrder := range spotOrders {
			symbol, err := currency.NewPairDelimiter(sOrder.CurrencyPair, format.Delimiter)
			if err != nil {
				return nil, err
			}
			for _, sOrderDetail := range sOrder.Orders {
				if sOrderDetail.Status != statusOpen {
					continue
				}
				side, err := order.StringToOrderSide(sOrderDetail.Side)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
				}
				oType, err := order.StringToOrderType(sOrderDetail.Type)
				if err != nil {
					return nil, err
				}
				status, err := order.StringToOrderStatus(sOrderDetail.Status)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
				}
				orders = append(orders, order.Detail{
					Side:                 side,
					Type:                 oType,
					Status:               status,
					Pair:                 symbol,
					OrderID:              sOrderDetail.OrderID,
					Amount:               sOrderDetail.Amount.Float64(),
					ExecutedAmount:       sOrderDetail.Amount.Float64() - sOrderDetail.Left.Float64(),
					RemainingAmount:      sOrderDetail.Left.Float64(),
					Price:                sOrderDetail.Price.Float64(),
					AverageExecutedPrice: sOrderDetail.AverageFillPrice.Float64(),
					Date:                 sOrderDetail.CreateTime.Time(),
					LastUpdated:          sOrderDetail.UpdateTime.Time(),
					Exchange:             e.Name,
					AssetType:            req.AssetType,
					ClientOrderID:        sOrderDetail.Text,
					FeeAsset:             currency.NewCode(sOrderDetail.FeeCurrency),
				})
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, req.AssetType)
		if err != nil {
			return nil, err
		}
		var futuresOrders []*Order
		if req.AssetType == asset.DeliveryFutures {
			futuresOrders, err = e.GetDeliveryOrders(ctx, currency.EMPTYPAIR, settle, statusOpen, "", 0, 0, false)
		} else {
			futuresOrders, err = e.GetFuturesOrders(ctx, currency.EMPTYPAIR, statusOpen, "", settle, 0, 0, false)
		}
		if err != nil {
			return nil, err
		}
		for _, fOrder := range futuresOrders {
			pair, err := currency.NewPairFromString(fOrder.Contract)
			if err != nil {
				return nil, err
			}

			if fOrder.Status != statusOpen || (len(req.Pairs) > 0 && !req.Pairs.Contains(pair, true)) {
				continue
			}
			side, amount, remaining := getSideAndAmountFromSize(fOrder.Size, fOrder.RemainingAmount)
			tif, err := timeInForceFromString(fOrder.TimeInForce)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Status:               order.Open,
				Amount:               amount,
				ContractAmount:       amount,
				Pair:                 pair,
				OrderID:              strconv.FormatInt(fOrder.ID, 10),
				ClientOrderID:        getClientOrderIDFromText(fOrder.Text),
				Price:                fOrder.OrderPrice.Float64(),
				ExecutedAmount:       amount - remaining,
				RemainingAmount:      remaining,
				LastUpdated:          fOrder.FinishTime.Time(),
				Date:                 fOrder.CreateTime.Time(),
				Exchange:             e.Name,
				AssetType:            req.AssetType,
				Side:                 side,
				Type:                 order.Limit,
				SettlementCurrency:   settle,
				ReduceOnly:           fOrder.IsReduceOnly,
				TimeInForce:          tif,
				AverageExecutedPrice: fOrder.FillPrice.Float64(),
			})
		}
	case asset.Options:
		optionsOrders, err := e.GetOptionFuturesOrders(ctx, currency.EMPTYPAIR, "", statusOpen, 0, 0, req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
		for _, optOrd := range optionsOrders {
			var currencyPair currency.Pair
			var status order.Status
			currencyPair, err = currency.NewPairFromString(optOrd.Contract)
			if err != nil {
				return nil, err
			}
			status, err = order.StringToOrderStatus(optOrd.Status)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Status:          status,
				Amount:          optOrd.Size,
				Pair:            currencyPair,
				OrderID:         strconv.FormatInt(optOrd.OptionOrderID, 10),
				Price:           optOrd.Price.Float64(),
				ExecutedAmount:  optOrd.Size - optOrd.Left,
				RemainingAmount: optOrd.Left,
				LastUpdated:     optOrd.FinishTime.Time(),
				Date:            optOrd.CreateTime.Time(),
				Exchange:        e.Name,
				AssetType:       req.AssetType,
				ClientOrderID:   optOrd.Text,
			})
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
	var orders []order.Detail
	format, err := e.GetPairFormat(req.AssetType, true)
	if err != nil {
		return nil, err
	}
	switch req.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		for x := range req.Pairs {
			fPair := req.Pairs[x].Format(format)
			spotOrders, err := e.GetMySpotTradingHistory(ctx, fPair, req.FromOrderID, 0, 0, req.AssetType == asset.CrossMargin, req.StartTime, req.EndTime)
			if err != nil {
				return nil, err
			}
			for _, sTradeDetail := range spotOrders {
				var side order.Side
				side, err = order.StringToOrderSide(sTradeDetail.Side)
				if err != nil {
					return nil, err
				}
				detail := order.Detail{
					OrderID:        sTradeDetail.OrderID,
					Amount:         sTradeDetail.Amount.Float64(),
					ExecutedAmount: sTradeDetail.Amount.Float64(),
					Price:          sTradeDetail.Price.Float64(),
					Date:           sTradeDetail.CreateTime.Time(),
					Side:           side,
					Exchange:       e.Name,
					Pair:           fPair,
					AssetType:      req.AssetType,
					Fee:            sTradeDetail.Fee.Float64(),
					FeeAsset:       currency.NewCode(sTradeDetail.FeeCurrency),
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		for x := range req.Pairs {
			fPair := req.Pairs[x].Format(format)
			settle, err := getSettlementCurrency(fPair, req.AssetType)
			if err != nil {
				return nil, err
			}
			var futuresOrder []*TradingHistoryItem
			if req.AssetType == asset.DeliveryFutures {
				futuresOrder, err = e.GetMyDeliveryTradingHistory(ctx, settle, req.FromOrderID, fPair, 0, 0, 0, "")
			} else {
				futuresOrder, err = e.GetMyFuturesTradingHistory(ctx, settle, "", req.FromOrderID, fPair, 0, 0, 0)
			}
			if err != nil {
				return nil, err
			}
			for _, fOrder := range futuresOrder {
				detail := order.Detail{
					OrderID:   strconv.FormatInt(fOrder.ID, 10),
					Amount:    fOrder.Size,
					Price:     fOrder.Price.Float64(),
					Date:      fOrder.CreateTime.Time(),
					Exchange:  e.Name,
					Pair:      fPair,
					AssetType: req.AssetType,
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
		}
	case asset.Options:
		for x := range req.Pairs {
			fPair := req.Pairs[x].Format(format)
			optionOrders, err := e.GetMyOptionsTradingHistory(ctx, fPair.String(), fPair.Upper(), 0, 0, req.StartTime, req.EndTime)
			if err != nil {
				return nil, err
			}
			for _, optionTradeDetail := range optionOrders {
				detail := order.Detail{
					OrderID:   strconv.FormatInt(optionTradeDetail.OrderID, 10),
					Amount:    optionTradeDetail.Size,
					Price:     optionTradeDetail.Price.Float64(),
					Date:      optionTradeDetail.CreateTime.Time(),
					Exchange:  e.Name,
					Pair:      fPair,
					AssetType: req.AssetType,
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(e.Name, orders), nil
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
		candles, err := e.GetCandlesticks(ctx, req.RequestFormatted, 0, start, end, interval)
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
				Volume: sCandle.BaseCcyAmount.Float64(),
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(pair, a)
		if err != nil {
			return nil, err
		}
		var candles []*FuturesCandlestick
		if a == asset.DeliveryFutures {
			candles, err = e.GetDeliveryFuturesCandlesticks(ctx, settle, req.RequestFormatted.Upper(), start, end, 0, interval)
		} else {
			candles, err = e.GetFuturesCandlesticks(ctx, settle, req.RequestFormatted, start, end, 0, interval)
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
				Volume: fCandle.Volume,
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
			candles, err := e.GetCandlesticks(ctx, req.RequestFormatted, 0, r.Start.Time, r.End.Time, interval)
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
					Volume: sCandle.QuoteCcyVolume.Float64(),
				})
			}
		case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
			settle, err := getSettlementCurrency(pair, a)
			if err != nil {
				return nil, err
			}
			var candles []*FuturesCandlestick
			if a == asset.DeliveryFutures {
				candles, err = e.GetDeliveryFuturesCandlesticks(ctx, settle, req.RequestFormatted.Upper(), r.Start.Time, r.End.Time, 0, interval)
			} else {
				candles, err = e.GetFuturesCandlesticks(ctx, settle, req.RequestFormatted, r.Start.Time, r.End.Time, 0, interval)
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
					Volume: fCandle.Volume,
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
		contracts, err := e.GetAllFutureContracts(ctx, settle)
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
			}
			c.LatestRate = fundingrate.Rate{
				Time: fContract.FundingNextApply.Time().Add(-time.Duration(fContract.FundingInterval) * time.Second),
				Rate: fContract.FundingRate.Decimal(),
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
				QuoteStepIncrementSize:  math.Pow10(-int(cpDetail.PricePrecision)),
				AmountStepIncrementSize: math.Pow10(-int(cpDetail.AmountPrecision)),
				MinimumBaseAmount:       minBaseAmount,
				MinimumQuoteAmount:      cpDetail.MinQuoteAmount.Float64(),
				Delisted:                cpDetail.DelistingTime.Time(),
			})
		}
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		settlement := currency.USDT
		if a == asset.CoinMarginedFutures {
			settlement = currency.BTC
		}
		contractInfo, err := e.GetAllFutureContracts(ctx, settlement)
		if err != nil {
			return err
		}
		// MBABYDOGE price is 1e6 x spot price
		divCurrency := currency.NewCode("MBABYDOGE")
		l = make([]limits.MinMaxLevel, 0, len(contractInfo))
		for _, fContract := range contractInfo {
			priceDiv := 1.0
			if fContract.Name.Base.Equal(divCurrency) {
				priceDiv = 1e6
			}

			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, fContract.Name),
				MinimumBaseAmount:       fContract.OrderSizeMin.Float64(),
				MaximumBaseAmount:       fContract.OrderSizeMax.Float64(),
				PriceStepIncrementSize:  fContract.OrderPriceRound.Float64(),
				AmountStepIncrementSize: 1, // 1 Contract
				MultiplierDecimal:       fContract.QuantoMultiplier.Float64(),
				PriceDivisor:            priceDiv,
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
				l = append(l, limits.MinMaxLevel{
					Key:                     key.NewExchangeAssetPair(e.Name, a, cp),
					MinimumBaseAmount:       float64(dContract.OrderSizeMin),
					MaximumBaseAmount:       float64(dContract.OrderSizeMax),
					PriceStepIncrementSize:  dContract.OrderPriceRound.Float64(),
					AmountStepIncrementSize: 1,
					Expiry:                  dContract.ExpireTime.Time(),
				})
			}
		}
	case asset.Options:
		underlyings, err := e.GetAllOptionsUnderlyings(ctx)
		if err != nil {
			return err
		}
		for _, optionUly := range underlyings {
			contracts, err := e.GetAllContractOfUnderlyingWithinExpiryDate(ctx, optionUly.Name, time.Time{})
			if err != nil {
				return err
			}
			l = make([]limits.MinMaxLevel, 0, len(contracts))
			for _, optionContract := range contracts {
				cp, err := currency.NewPairFromString(strings.ReplaceAll(optionContract.Name, currency.DashDelimiter, currency.UnderscoreDelimiter))
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
				})
			}
		}
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}

	return limits.Load(l)
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

	records, err := e.GetFutureFundingRates(ctx, r.PaymentCurrency, fPair, 1000)
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
			Rate: decimal.NewFromFloat(fFundingRate.Rate.Float64()),
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

	contracts, err := e.GetAllFutureContracts(ctx, settle)
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

func contractToFundingRate(name string, item asset.Item, fPair currency.Pair, contract *FuturesContract, includeUpcomingRate bool) fundingrate.LatestRateResponse {
	resp := fundingrate.LatestRateResponse{
		Exchange: name,
		Asset:    item,
		Pair:     fPair,
		LatestRate: fundingrate.Rate{
			Time: contract.FundingNextApply.Time().Add(-time.Duration(contract.FundingInterval) * time.Second),
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
		var p currency.Pair
		if len(keys) == 1 && a == keys[0].Asset {
			if p, errs = e.MatchSymbolWithAvailablePairs(keys[0].Pair().String(), a, false); errs != nil {
				return nil, errs
			}
		}
		contracts, err := e.getOpenInterestContracts(ctx, a, p)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w fetching %s", err, a))
			continue
		}
		for _, c := range contracts {
			if p.IsEmpty() { // If not exactly one key provided
				p, err = e.MatchSymbolWithAvailablePairs(c.contractName(), a, true)
				if err != nil {
					if err := common.ExcludeError(err, currency.ErrPairNotFound); err != nil {
						errs = common.AppendError(errs, fmt.Errorf("%w from %s contract %s", err, a, c.contractName()))
					}
					continue
				}
				if len(keys) == 0 { // No keys: All enabled pairs
					if enabled, err := e.IsPairEnabled(p, a); err != nil {
						errs = common.AppendError(errs, fmt.Errorf("%w: %s %s", err, a, p))
						continue
					} else if !enabled {
						continue
					}
				} else { // More than one key; Any available pair
					if !slices.ContainsFunc(keys, func(k key.PairAsset) bool { return a == k.Asset && k.Pair().Equal(p) }) {
						continue
					}
				}
			}
			resp = append(resp, futures.OpenInterest{
				Key: key.ExchangeAssetPair{
					Exchange: e.Name,
					Base:     p.Base.Item,
					Quote:    p.Quote.Item,
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

func (c *FuturesContract) openInterest() float64 {
	i := float64(c.PositionSize) * c.IndexPrice.Float64()
	if q := c.QuantoMultiplier.Float64(); q != 0 {
		i *= q
	}
	return i
}

func (c *FuturesContract) contractName() string {
	return c.Name.String()
}

func (c *DeliveryContract) openInterest() float64 {
	return c.QuantoMultiplier.Float64() * float64(c.PositionSize) * c.IndexPrice.Float64()
}

func (c *DeliveryContract) contractName() string {
	return c.Name
}

func (e *Exchange) getOpenInterestContracts(ctx context.Context, a asset.Item, p currency.Pair) ([]openInterestContract, error) {
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
	futuresContracts, err := e.GetAllFutureContracts(ctx, settle)
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

func formatClientOrderID(clientOrderID string) string {
	if clientOrderID == "" || strings.HasPrefix(clientOrderID, "t-") {
		return clientOrderID
	}
	return "t-" + clientOrderID
}

// getTypeFromTimeInForce returns the order type and if the order is post only
func getTypeFromTimeInForce(tif string, price float64) (orderType order.Type) {
	switch tif {
	case iocTIF, fokTIF:
		return order.Market
	case pocTIF, gtcTIF:
		return order.Limit
	default:
		if price == 0 {
			return order.Market
		}
		return order.Limit
	}
}

// getSideAndAmountFromSize returns the order side, amount and remaining amounts
func getSideAndAmountFromSize(size, left float64) (side order.Side, amount, remaining float64) {
	if size < 0 {
		return order.Short, math.Abs(size), math.Abs(left)
	}
	return order.Long, size, left
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
		return tradeBaseURL + futuresPath + settle.String() + "/" + cp.Upper().String(), nil
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

	s.ClientOrderID = formatClientOrderID(s.ClientOrderID)

	s.Pair, err = e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	s.Pair = s.Pair.Upper()

	switch s.AssetType {
	case asset.Spot:
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
		return e.deriveFuturesWebsocketOrderResponse(resp)
	default:
		return nil, common.ErrNotYetImplemented
	}
}

func getFuturesOrderRequest(s *order.Submit) (*ContractOrderCreateParams, error) {
	amountWithDirection, err := getFutureOrderSize(s)
	if err != nil {
		return nil, err
	}

	tif, err := toExchangeTIF(s.TimeInForce, s.Price)
	if err != nil {
		return nil, err
	}

	return &ContractOrderCreateParams{
		Contract:    s.Pair,
		Size:        amountWithDirection,
		Price:       number(s.Price),
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
		status := order.Open
		if resp.FinishAs != "" && resp.FinishAs != statusOpen {
			status, err = order.StringToOrderStatus(resp.FinishAs)
			if err != nil {
				return nil, err
			}
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
		out[i] = &order.SubmitResponse{
			Exchange:             e.Name,
			OrderID:              resp.ID,
			AssetType:            resp.Account,
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

func (e *Exchange) deriveFuturesWebsocketOrderResponse(responses *WebsocketFuturesOrderResponse) (*order.SubmitResponse, error) {
	resp, err := e.deriveFuturesWebsocketOrderResponses([]*WebsocketFuturesOrderResponse{responses})
	if err != nil {
		return nil, err
	}
	return resp[0], nil
}

// deriveFuturesWebsocketOrderResponses returns the order submission responses for futures
func (e *Exchange) deriveFuturesWebsocketOrderResponses(responses []*WebsocketFuturesOrderResponse) ([]*order.SubmitResponse, error) {
	if len(responses) == 0 {
		return nil, common.ErrNoResponse
	}

	out := make([]*order.SubmitResponse, 0, len(responses))
	for _, resp := range responses {
		status := order.Open
		if resp.FinishAs != "" && resp.FinishAs != statusOpen {
			var err error
			status, err = order.StringToOrderStatus(resp.FinishAs)
			if err != nil {
				return nil, err
			}
		}

		oType := order.Market
		if resp.Price != 0 {
			oType = order.Limit
		}

		side := order.Long
		if resp.Size < 0 {
			side = order.Short
		}

		var clientOrderID string
		if resp.Text != "" && strings.HasPrefix(resp.Text, "t-") {
			clientOrderID = resp.Text
		}
		tif, err := order.StringToTimeInForce(resp.TimeInForce)
		if err != nil {
			return nil, err
		}
		out = append(out, &order.SubmitResponse{
			Exchange:             e.Name,
			OrderID:              strconv.FormatInt(resp.ID, 10),
			AssetType:            asset.Futures,
			Pair:                 resp.Contract,
			ClientOrderID:        clientOrderID,
			Date:                 resp.CreateTime.Time(),
			LastUpdated:          resp.UpdateTime.Time(),
			RemainingAmount:      math.Abs(resp.Left),
			Amount:               math.Abs(resp.Size),
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

	tif, err := toExchangeTIF(s.TimeInForce, s.Price)
	if err != nil {
		return nil, err
	}

	return &CreateOrderRequest{
		Side:         side,
		Type:         s.Type.Lower(),
		Account:      s.AssetType,
		Amount:       types.Number(s.GetTradeAmount(e.GetTradingRequirements())),
		Price:        types.Number(s.Price),
		CurrencyPair: s.Pair,
		Text:         s.ClientOrderID,
		TimeInForce:  tif,
	}, nil
}

func getSettlementCurrency(p currency.Pair, a asset.Item) (currency.Code, error) {
	switch a {
	case asset.DeliveryFutures:
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
		for x, ord := range orders {
			var err error
			if reqs[x], err = e.getSpotOrderRequest(ord); err != nil {
				return nil, err
			}
		}
		resp, err := e.WebsocketSpotSubmitOrders(ctx, reqs...)
		if err != nil {
			return nil, err
		}
		return e.deriveSpotWebsocketOrderResponses(resp)
	default:
		return nil, fmt.Errorf("%w for %s", common.ErrNotYetImplemented, a)
	}
}

// MessageID returns a unique ID conforming to Gate's max length of 32 bytes for request IDs
func (e *Exchange) MessageID() string {
	u := uuid.Must(uuid.NewV7())
	var buf [32]byte
	hex.Encode(buf[:], u[:])
	return string(buf[:])
}
