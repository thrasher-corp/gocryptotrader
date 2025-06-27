package gateio

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
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
func (g *Gateio) SetDefaults() {
	g.Name = "GateIO"
	g.Enabled = true
	g.Verbose = true
	g.API.CredentialsValidator.RequiresKey = true
	g.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	err := g.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Margin, asset.CrossMargin, asset.DeliveryFutures, asset.Options)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	g.Features = exchange.Features{
		CurrencyTranslations: currency.NewTranslations(map[currency.Code]currency.Code{
			currency.NewCode("MBABYDOGE"): currency.BABYDOGE,
		}),
		TradingRequirements: protocol.TradingRequirements{
			SpotMarketOrderAmountPurchaseQuotationOnly: true,
			SpotMarketOrderAmountSellBaseOnly:          true,
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
	g.Requester, err = request.New(g.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(packageRateLimits),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	// TODO: Majority of margin REST endpoints are labelled as deprecated on the API docs. These will need to be removed.
	err = g.DisableAssetWebsocketSupport(asset.Margin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	// TODO: Add websocket cross margin support.
	err = g.DisableAssetWebsocketSupport(asset.CrossMargin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	g.API.Endpoints = g.NewEndpoints()
	err = g.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              gateioTradeURL,
		exchange.RestFutures:           gateioFuturesLiveTradingAlternative,
		exchange.RestSpotSupplementary: gateioFuturesTestnetTrading,
		exchange.WebsocketSpot:         gateioWebsocketEndpoint,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	g.Websocket = websocket.NewManager()
	g.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	g.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	g.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
	g.wsOBUpdateMgr = newWsOBUpdateManager(defaultWSSnapshotSyncDelay)
}

// Setup sets user configuration
func (g *Gateio) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		g.SetEnabled(false)
		return nil
	}
	err = g.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = g.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exch,
		Features:                     &g.Features.Supports.WebsocketCapabilities,
		FillsFeed:                    g.Features.Enabled.FillsFeed,
		TradeFeed:                    g.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement: true,
		RateLimitDefinitions:         packageRateLimits,
	})
	if err != nil {
		return err
	}
	// Spot connection
	err = g.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                      gateioWebsocketEndpoint,
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         exch.WebsocketResponseMaxLimit,
		Handler:                  g.WsHandleSpotData,
		Subscriber:               g.Subscribe,
		Unsubscriber:             g.Unsubscribe,
		GenerateSubscriptions:    g.generateSubscriptionsSpot,
		Connector:                g.WsConnectSpot,
		Authenticate:             g.authenticateSpot,
		MessageFilter:            asset.Spot,
		BespokeGenerateMessageID: g.GenerateWebsocketMessageID,
	})
	if err != nil {
		return err
	}
	// Futures connection - USDT margined
	err = g.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  usdtFuturesWebsocketURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, incoming []byte) error {
			return g.WsHandleFuturesData(ctx, incoming, asset.USDTMarginedFutures)
		},
		Subscriber:   g.FuturesSubscribe,
		Unsubscriber: g.FuturesUnsubscribe,
		GenerateSubscriptions: func() (subscription.List, error) {
			return g.GenerateFuturesDefaultSubscriptions(asset.USDTMarginedFutures)
		},
		Connector:                g.WsFuturesConnect,
		Authenticate:             g.authenticateFutures,
		MessageFilter:            asset.USDTMarginedFutures,
		BespokeGenerateMessageID: g.GenerateWebsocketMessageID,
	})
	if err != nil {
		return err
	}

	// Futures connection - BTC margined
	err = g.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  btcFuturesWebsocketURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, incoming []byte) error {
			return g.WsHandleFuturesData(ctx, incoming, asset.CoinMarginedFutures)
		},
		Subscriber:   g.FuturesSubscribe,
		Unsubscriber: g.FuturesUnsubscribe,
		GenerateSubscriptions: func() (subscription.List, error) {
			return g.GenerateFuturesDefaultSubscriptions(asset.CoinMarginedFutures)
		},
		Connector:                g.WsFuturesConnect,
		MessageFilter:            asset.CoinMarginedFutures,
		BespokeGenerateMessageID: g.GenerateWebsocketMessageID,
	})
	if err != nil {
		return err
	}

	// TODO: Add BTC margined delivery futures.
	// Futures connection - Delivery - USDT margined
	err = g.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  deliveryRealUSDTTradingURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, incoming []byte) error {
			return g.WsHandleFuturesData(ctx, incoming, asset.DeliveryFutures)
		},
		Subscriber:               g.DeliveryFuturesSubscribe,
		Unsubscriber:             g.DeliveryFuturesUnsubscribe,
		GenerateSubscriptions:    g.GenerateDeliveryFuturesDefaultSubscriptions,
		Connector:                g.WsDeliveryFuturesConnect,
		MessageFilter:            asset.DeliveryFutures,
		BespokeGenerateMessageID: g.GenerateWebsocketMessageID,
	})
	if err != nil {
		return err
	}

	// Futures connection - Options
	return g.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                      optionsWebsocketURL,
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         exch.WebsocketResponseMaxLimit,
		Handler:                  g.WsHandleOptionsData,
		Subscriber:               g.OptionsSubscribe,
		Unsubscriber:             g.OptionsUnsubscribe,
		GenerateSubscriptions:    g.GenerateOptionsDefaultSubscriptions,
		Connector:                g.WsOptionsConnect,
		MessageFilter:            asset.Options,
		BespokeGenerateMessageID: g.GenerateWebsocketMessageID,
	})
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gateio) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if !g.SupportsAsset(a) {
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	fPair, err := g.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	if fPair.IsEmpty() || fPair.Quote.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	fPair = fPair.Upper()
	var tickerData *ticker.Price
	switch a {
	case asset.Margin, asset.Spot, asset.CrossMargin:
		available, err := g.checkInstrumentAvailabilityInSpot(fPair)
		if err != nil {
			return nil, err
		}
		if a != asset.Spot && !available {
			return nil, fmt.Errorf("%v instrument %v does not have ticker data", a, fPair)
		}
		tickerNew, err := g.GetTicker(ctx, fPair.String(), "")
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
			ExchangeName: g.Name,
			AssetType:    a,
		}
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(fPair, a)
		if err != nil {
			return nil, err
		}
		var tickers []FuturesTicker
		if a == asset.DeliveryFutures {
			tickers, err = g.GetDeliveryFutureTickers(ctx, settle, fPair)
		} else {
			tickers, err = g.GetFuturesTickers(ctx, settle, fPair)
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
			ExchangeName: g.Name,
			AssetType:    a,
		}
	case asset.Options:
		var underlying currency.Pair
		var tickers []OptionsTicker
		underlying, err = g.GetUnderlyingFromCurrencyPair(fPair)
		if err != nil {
			return nil, err
		}
		tickers, err = g.GetOptionsTickers(ctx, underlying.String())
		if err != nil {
			return nil, err
		}
		for x := range tickers {
			if !tickers[x].Name.Equal(fPair) {
				continue
			}
			cleanQuote := strings.ReplaceAll(tickers[x].Name.Quote.String(), currency.UnderscoreDelimiter, currency.DashDelimiter)
			tickers[x].Name.Quote = currency.NewCode(cleanQuote)
			tickerData = &ticker.Price{
				Pair:         tickers[x].Name,
				Last:         tickers[x].LastPrice.Float64(),
				Bid:          tickers[x].Bid1Price.Float64(),
				Ask:          tickers[x].Ask1Price.Float64(),
				AskSize:      tickers[x].Ask1Size,
				BidSize:      tickers[x].Bid1Size,
				ExchangeName: g.Name,
				AssetType:    a,
			}
			err = ticker.ProcessTicker(tickerData)
			if err != nil {
				return nil, err
			}
		}
		return ticker.GetTicker(g.Name, fPair, a)
	}
	if err := ticker.ProcessTicker(tickerData); err != nil {
		return nil, err
	}
	return ticker.GetTicker(g.Name, fPair, a)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (g *Gateio) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !g.SupportsAsset(a) {
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	switch a {
	case asset.Spot:
		tradables, err := g.ListSpotCurrencyPairs(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(tradables))
		for x := range tradables {
			if tradables[x].TradeStatus == "untradable" {
				continue
			}
			p := strings.ToUpper(tradables[x].ID)
			if !g.IsValidPairString(p) {
				continue
			}
			cp, err := currency.NewPairFromString(p)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return pairs, nil
	case asset.Margin, asset.CrossMargin:
		tradables, err := g.GetMarginSupportedCurrencyPairs(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(tradables))
		for x := range tradables {
			if tradables[x].Status == 0 {
				continue
			}
			p := strings.ToUpper(tradables[x].Base + currency.UnderscoreDelimiter + tradables[x].Quote)
			if !g.IsValidPairString(p) {
				continue
			}
			cp, err := currency.NewPairFromString(p)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return pairs, nil
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
		if err != nil {
			return nil, err
		}
		contracts, err := g.GetAllFutureContracts(ctx, settle)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(contracts))
		for i := range contracts {
			if contracts[i].InDelisting {
				continue
			}
			p := strings.ToUpper(contracts[i].Name)
			if !g.IsValidPairString(p) {
				continue
			}
			cp, err := currency.NewPairFromString(p)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return slices.Clip(pairs), nil
	case asset.DeliveryFutures:
		contracts, err := g.GetAllDeliveryContracts(ctx, currency.USDT)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(contracts))
		for i := range contracts {
			if contracts[i].InDelisting {
				continue
			}
			p := strings.ToUpper(contracts[i].Name)
			if !g.IsValidPairString(p) {
				continue
			}
			cp, err := currency.NewPairFromString(p)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return slices.Clip(pairs), nil
	case asset.Options:
		underlyings, err := g.GetAllOptionsUnderlyings(ctx)
		if err != nil {
			return nil, err
		}
		var pairs []currency.Pair
		for x := range underlyings {
			contracts, err := g.GetAllContractOfUnderlyingWithinExpiryDate(ctx, underlyings[x].Name, time.Time{})
			if err != nil {
				return nil, err
			}
			for c := range contracts {
				if !g.IsValidPairString(contracts[c].Name) {
					continue
				}
				cp, err := currency.NewPairFromString(strings.ReplaceAll(contracts[c].Name, currency.DashDelimiter, currency.UnderscoreDelimiter))
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
func (g *Gateio) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := g.GetAssetTypes(false)
	for x := range assets {
		pairs, err := g.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return err
		}
		if len(pairs) == 0 {
			return errors.New("no tradable pairs found")
		}
		err = g.UpdatePairs(pairs, assets[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return g.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (g *Gateio) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !g.SupportsAsset(a) {
		return fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		tickers, err := g.GetTickers(ctx, currency.EMPTYPAIR.String(), "")
		if err != nil {
			return err
		}
		for x := range tickers {
			var currencyPair currency.Pair
			currencyPair, err = currency.NewPairFromString(tickers[x].CurrencyPair)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers[x].Last.Float64(),
				High:         tickers[x].High24H.Float64(),
				Low:          tickers[x].Low24H.Float64(),
				Bid:          tickers[x].HighestBid.Float64(),
				Ask:          tickers[x].LowestAsk.Float64(),
				QuoteVolume:  tickers[x].QuoteVolume.Float64(),
				Volume:       tickers[x].BaseVolume.Float64(),
				ExchangeName: g.Name,
				Pair:         currencyPair,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, errs := getSettlementCurrency(currency.EMPTYPAIR, a)
		if errs != nil {
			return errs
		}
		var tickers []FuturesTicker
		if a == asset.DeliveryFutures {
			tickers, errs = g.GetDeliveryFutureTickers(ctx, settle, currency.EMPTYPAIR)
		} else {
			tickers, errs = g.GetFuturesTickers(ctx, settle, currency.EMPTYPAIR)
		}
		for i := range tickers {
			currencyPair, err := currency.NewPairFromString(tickers[i].Contract)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			if err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers[i].Last.Float64(),
				High:         tickers[i].High24H.Float64(),
				Low:          tickers[i].Low24H.Float64(),
				Volume:       tickers[i].Volume24H.Float64(),
				QuoteVolume:  tickers[i].Volume24HQuote.Float64(),
				ExchangeName: g.Name,
				Pair:         currencyPair,
				AssetType:    a,
			}); err != nil {
				errs = common.AppendError(errs, err)
			}
		}
		return errs
	case asset.Options:
		pairs, err := g.GetEnabledPairs(a)
		if err != nil {
			return err
		}
		for i := range pairs {
			underlying, err := g.GetUnderlyingFromCurrencyPair(pairs[i])
			if err != nil {
				return err
			}
			tickers, err := g.GetOptionsTickers(ctx, underlying.String())
			if err != nil {
				return err
			}
			for x := range tickers {
				err = ticker.ProcessTicker(&ticker.Price{
					Last:         tickers[x].LastPrice.Float64(),
					Ask:          tickers[x].Ask1Price.Float64(),
					AskSize:      tickers[x].Ask1Size,
					Bid:          tickers[x].Bid1Price.Float64(),
					BidSize:      tickers[x].Bid1Size,
					Pair:         tickers[x].Name,
					ExchangeName: g.Name,
					AssetType:    a,
				})
				if err != nil {
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
func (g *Gateio) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	return g.UpdateOrderbookWithLimit(ctx, p, a, 0)
}

// UpdateOrderbookWithLimit updates and returns the orderbook for a currency pair with a set orderbook size limit
func (g *Gateio) UpdateOrderbookWithLimit(ctx context.Context, p currency.Pair, a asset.Item, limit uint64) (*orderbook.Book, error) {
	p, err := g.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	var o *Orderbook
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		var available bool
		available, err = g.checkInstrumentAvailabilityInSpot(p)
		if err != nil {
			return nil, err
		}
		if a != asset.Spot && !available {
			return nil, fmt.Errorf("%v instrument %v does not have orderbook data", a, p)
		}
		o, err = g.GetOrderbook(ctx, p.String(), "", limit, true)
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		var settle currency.Code
		settle, err = getSettlementCurrency(p, a)
		if err != nil {
			return nil, err
		}
		o, err = g.GetFuturesOrderbook(ctx, settle, p.String(), "", limit, true)
	case asset.DeliveryFutures:
		o, err = g.GetDeliveryOrderbook(ctx, currency.USDT, "", p, limit, true)
	case asset.Options:
		o, err = g.GetOptionsOrderbook(ctx, p, "", limit, true)
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          g.Name,
		Asset:             a,
		ValidateOrderbook: g.ValidateOrderbook,
		Pair:              p.Upper(),
		LastUpdateID:      o.ID,
		LastUpdated:       o.Update.Time(),
		LastPushed:        o.Current.Time(),
	}
	book.Bids = make(orderbook.Levels, len(o.Bids))
	for x := range o.Bids {
		book.Bids[x] = orderbook.Level{
			Amount: o.Bids[x].Amount.Float64(),
			Price:  o.Bids[x].Price.Float64(),
		}
	}
	book.Asks = make(orderbook.Levels, len(o.Asks))
	for x := range o.Asks {
		book.Asks[x] = orderbook.Level{
			Amount: o.Asks[x].Amount.Float64(),
			Price:  o.Asks[x].Price.Float64(),
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(g.Name, book.Pair, a)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
func (g *Gateio) UpdateAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error) {
	info := account.Holdings{
		Exchange: g.Name,
		Accounts: []account.SubAccount{{
			AssetType: a,
		}},
	}
	switch a {
	case asset.Spot:
		balances, err := g.GetSpotAccounts(ctx, currency.EMPTYCODE)
		if err != nil {
			return info, err
		}
		currencies := make([]account.Balance, len(balances))
		for i := range balances {
			currencies[i] = account.Balance{
				Currency: currency.NewCode(balances[i].Currency),
				Total:    balances[i].Available.Float64() + balances[i].Locked.Float64(),
				Hold:     balances[i].Locked.Float64(),
				Free:     balances[i].Available.Float64(),
			}
		}
		info.Accounts[0].Currencies = currencies
	case asset.Margin, asset.CrossMargin:
		balances, err := g.GetMarginAccountList(ctx, currency.EMPTYPAIR)
		if err != nil {
			return info, err
		}
		currencies := make([]account.Balance, 0, 2*len(balances))
		for i := range balances {
			currencies = append(currencies,
				account.Balance{
					Currency: currency.NewCode(balances[i].Base.Currency),
					Total:    balances[i].Base.Available.Float64() + balances[i].Base.LockedAmount.Float64(),
					Hold:     balances[i].Base.LockedAmount.Float64(),
					Free:     balances[i].Base.Available.Float64(),
				},
				account.Balance{
					Currency: currency.NewCode(balances[i].Quote.Currency),
					Total:    balances[i].Quote.Available.Float64() + balances[i].Quote.LockedAmount.Float64(),
					Hold:     balances[i].Quote.LockedAmount.Float64(),
					Free:     balances[i].Quote.Available.Float64(),
				})
		}
		info.Accounts[0].Currencies = currencies
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
		if err != nil {
			return info, err
		}
		var acc *FuturesAccount
		if a == asset.DeliveryFutures {
			acc, err = g.GetDeliveryFuturesAccounts(ctx, settle)
		} else {
			acc, err = g.QueryFuturesAccount(ctx, settle)
		}
		if err != nil {
			return info, err
		}
		info.Accounts[0].Currencies = []account.Balance{{
			Currency: currency.NewCode(acc.Currency),
			Total:    acc.Total.Float64(),
			Hold:     acc.Total.Float64() - acc.Available.Float64(),
			Free:     acc.Available.Float64(),
		}}
	case asset.Options:
		balance, err := g.GetOptionAccounts(ctx)
		if err != nil {
			return info, err
		}
		info.Accounts[0].Currencies = []account.Balance{{
			Currency: currency.NewCode(balance.Currency),
			Total:    balance.Total.Float64(),
			Hold:     balance.Total.Float64() - balance.Available.Float64(),
			Free:     balance.Available.Float64(),
		}}
	default:
		return info, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	creds, err := g.GetCredentials(ctx)
	if err == nil {
		err = account.Process(&info, creds)
	}
	return info, err
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gateio) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (g *Gateio) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	records, err := g.GetWithdrawalRecords(ctx, c, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return nil, err
	}
	withdrawalHistories := make([]exchange.WithdrawalHistory, len(records))
	for x := range records {
		withdrawalHistories[x] = exchange.WithdrawalHistory{
			Status:          records[x].Status,
			TransferID:      records[x].ID,
			Currency:        records[x].Currency,
			Amount:          records[x].Amount.Float64(),
			CryptoTxID:      records[x].TransactionID,
			CryptoToAddress: records[x].WithdrawalAddress,
			Timestamp:       records[x].Timestamp.Time(),
		}
	}
	return withdrawalHistories, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (g *Gateio) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	p, err := g.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		if p.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		tradeData, err := g.GetMarketTrades(ctx, p, 0, "", false, time.Time{}, time.Time{}, 0)
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(tradeData))
		for i := range tradeData {
			side, err := order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp[i] = trade.Data{
				Exchange:     g.Name,
				TID:          tradeData[i].OrderID,
				CurrencyPair: p,
				AssetType:    a,
				Side:         side,
				Price:        tradeData[i].Price.Float64(),
				Amount:       tradeData[i].Amount.Float64(),
				Timestamp:    tradeData[i].CreateTime.Time(),
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(p, a)
		if err != nil {
			return nil, err
		}
		var futuresTrades []TradingHistoryItem
		if a == asset.DeliveryFutures {
			futuresTrades, err = g.GetDeliveryTradingHistory(ctx, settle, "", p.Upper(), 0, time.Time{}, time.Time{})
		} else {
			futuresTrades, err = g.GetFuturesTradingHistory(ctx, settle, p, 0, 0, "", time.Time{}, time.Time{})
		}
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(futuresTrades))
		for i := range futuresTrades {
			resp[i] = trade.Data{
				TID:          strconv.FormatInt(futuresTrades[i].ID, 10),
				Exchange:     g.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        futuresTrades[i].Price.Float64(),
				Amount:       futuresTrades[i].Size,
				Timestamp:    futuresTrades[i].CreateTime.Time(),
			}
		}
	case asset.Options:
		trades, err := g.GetOptionsTradeHistory(ctx, p.Upper(), "", 0, 0, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(trades))
		for i := range trades {
			resp[i] = trade.Data{
				TID:          strconv.FormatInt(trades[i].ID, 10),
				Exchange:     g.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        trades[i].Price.Float64(),
				Amount:       trades[i].Size,
				Timestamp:    trades[i].CreateTime.Time(),
			}
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	err = g.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (g *Gateio) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
// TODO: support multiple order types (IOC)
func (g *Gateio) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate(g.GetTradingRequirements())
	if err != nil {
		return nil, err
	}

	s.Pair, err = g.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	s.Pair = s.Pair.Upper()

	switch s.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		req, err := g.getSpotOrderRequest(s)
		if err != nil {
			return nil, err
		}
		sOrder, err := g.PlaceSpotOrder(ctx, req)
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
		amountWithDirection, err := getFutureOrderSize(s)
		if err != nil {
			return nil, err
		}
		settle, err := getSettlementCurrency(s.Pair, s.AssetType)
		if err != nil {
			return nil, err
		}
		orderParams := &ContractOrderCreateParams{
			Contract:    s.Pair,
			Size:        amountWithDirection,
			Price:       strconv.FormatFloat(s.Price, 'f', -1, 64), // Cannot be an empty string, requires "0" for market orders.
			Settle:      settle,
			ReduceOnly:  s.ReduceOnly,
			TimeInForce: timeInForceString(s.TimeInForce),
			Text:        s.ClientOrderID,
		}
		var o *Order
		if s.AssetType == asset.DeliveryFutures {
			o, err = g.PlaceDeliveryOrder(ctx, orderParams)
		} else {
			o, err = g.PlaceFuturesOrder(ctx, orderParams)
		}
		if err != nil {
			return nil, err
		}
		resp, err := s.DeriveSubmitResponse(strconv.FormatInt(o.ID, 10))
		if err != nil {
			return nil, err
		}
		if o.Status != statusOpen {
			resp.Status, err = order.StringToOrderStatus(o.FinishAs)
			if err != nil {
				return nil, err
			}
		} else {
			resp.Status = order.Open
		}
		resp.Date = o.CreateTime.Time()
		resp.ClientOrderID = getClientOrderIDFromText(o.Text)
		resp.Amount = math.Abs(o.Size)
		resp.Price = o.OrderPrice.Float64()
		resp.AverageExecutedPrice = o.FillPrice.Float64()
		return resp, nil
	case asset.Options:
		optionOrder, err := g.PlaceOptionOrder(ctx, &OptionOrderParam{
			Contract:   s.Pair.String(),
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

// ModifyOrder will allow of changing orderbook placement and limit to market conversion
func (g *Gateio) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (g *Gateio) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	fPair, err := g.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}
	switch o.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		_, err = g.CancelSingleSpotOrder(ctx, o.OrderID, fPair.String(), o.AssetType == asset.CrossMargin)
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		var settle currency.Code
		if settle, err = getSettlementCurrency(o.Pair, o.AssetType); err == nil {
			if o.AssetType == asset.DeliveryFutures {
				_, err = g.CancelSingleDeliveryOrder(ctx, settle, o.OrderID)
			} else {
				_, err = g.CancelSingleFuturesOrder(ctx, settle, o.OrderID)
			}
		}
	case asset.Options:
		_, err = g.CancelOptionSingleOrder(ctx, o.OrderID)
	default:
		return fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (g *Gateio) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
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
		o[x].Pair, err = g.FormatExchangeCurrency(o[x].Pair, a)
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
		err = o[x].Validate(o[x].StandardCancel())
		if err != nil {
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
			var cancel []CancelOrderByIDResponse
			cancel, err = g.CancelBatchOrdersWithIDList(ctx, input)
			if err != nil {
				return nil, err
			}
			for j := range cancel {
				if cancel[j].Succeeded {
					response.Status[cancel[j].OrderID] = order.Cancelled.String()
				}
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		for i := range o {
			settle, err := getSettlementCurrency(o[i].Pair, a)
			if err != nil {
				return nil, err
			}
			var resp []Order
			if a == asset.DeliveryFutures {
				resp, err = g.CancelMultipleDeliveryOrders(ctx, o[i].Pair, o[i].Side.Lower(), settle)
			} else {
				resp, err = g.CancelMultipleFuturesOpenOrders(ctx, o[i].Pair, o[i].Side.Lower(), settle)
			}
			if err != nil {
				return nil, err
			}
			for j := range resp {
				response.Status[strconv.FormatInt(resp[j].ID, 10)] = resp[j].Status
			}
		}
	case asset.Options:
		for i := range o {
			cancel, err := g.CancelMultipleOptionOpenOrders(ctx, o[i].Pair, o[i].Pair.String(), o[i].Side.Lower())
			if err != nil {
				return nil, err
			}
			for j := range cancel {
				response.Status[strconv.FormatInt(cancel[j].OptionOrderID, 10)] = cancel[j].Status
			}
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	return &response, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (g *Gateio) CancelAllOrders(ctx context.Context, o *order.Cancel) (order.CancelAllResponse, error) {
	err := o.Validate()
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = map[string]string{}
	switch o.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		if o.Pair.IsEmpty() {
			return order.CancelAllResponse{}, currency.ErrCurrencyPairEmpty
		}
		var cancel []SpotPriceTriggeredOrder
		cancel, err = g.CancelMultipleSpotOpenOrders(ctx, o.Pair, o.AssetType)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for x := range cancel {
			cancelAllOrdersResponse.Status[strconv.FormatInt(cancel[x].AutoOrderID, 10)] = cancel[x].Status
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		if o.Pair.IsEmpty() {
			return cancelAllOrdersResponse, currency.ErrCurrencyPairEmpty
		}
		settle, err := getSettlementCurrency(o.Pair, o.AssetType)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		var cancel []Order
		if o.AssetType == asset.DeliveryFutures {
			cancel, err = g.CancelMultipleDeliveryOrders(ctx, o.Pair, o.Side.Lower(), settle)
		} else {
			cancel, err = g.CancelMultipleFuturesOpenOrders(ctx, o.Pair, o.Side.Lower(), settle)
		}
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for f := range cancel {
			cancelAllOrdersResponse.Status[strconv.FormatInt(cancel[f].ID, 10)] = cancel[f].Status
		}
	case asset.Options:
		var underlying currency.Pair
		if !o.Pair.IsEmpty() {
			underlying, err = g.GetUnderlyingFromCurrencyPair(o.Pair)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}
		cancel, err := g.CancelMultipleOptionOpenOrders(ctx, o.Pair, underlying.String(), o.Side.Lower())
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for x := range cancel {
			cancelAllOrdersResponse.Status[strconv.FormatInt(cancel[x].OptionOrderID, 10)] = cancel[x].Status
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, o.AssetType)
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (g *Gateio) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	if err := g.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}

	pair, err := g.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		var spotOrder *SpotOrder
		spotOrder, err = g.GetSpotOrder(ctx, orderID, pair, a)
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
			Exchange:       g.Name,
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
			fOrder, err = g.GetSingleDeliveryOrder(ctx, settle, orderID)
		} else {
			fOrder, err = g.GetSingleFuturesOrder(ctx, settle, orderID)
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
			Exchange:             g.Name,
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
		optionOrder, err := g.GetSingleOptionOrder(ctx, orderID)
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
			Exchange:       g.Name,
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
func (g *Gateio) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	addr, err := g.GenerateCurrencyDepositAddress(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	if chain != "" {
		for x := range addr.MultichainAddresses {
			if addr.MultichainAddresses[x].ObtainFailed == 1 {
				continue
			}
			if addr.MultichainAddresses[x].Chain == chain {
				return &deposit.Address{
					Chain:   addr.MultichainAddresses[x].Chain,
					Address: addr.MultichainAddresses[x].Address,
					Tag:     addr.MultichainAddresses[x].PaymentName,
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
func (g *Gateio) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	response, err := g.WithdrawCurrency(ctx,
		WithdrawalRequestParam{
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
func (g *Gateio) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gateio) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gateio) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !g.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return g.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (g *Gateio) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	format, err := g.GetPairFormat(req.AssetType, false)
	if err != nil {
		return nil, err
	}
	switch req.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		spotOrders, err := g.GetSpotOpenOrders(ctx, 0, 0, req.AssetType == asset.CrossMargin)
		if err != nil {
			return nil, err
		}
		for x := range spotOrders {
			symbol, err := currency.NewPairDelimiter(spotOrders[x].CurrencyPair, format.Delimiter)
			if err != nil {
				return nil, err
			}
			for y := range spotOrders[x].Orders {
				if spotOrders[x].Orders[y].Status != statusOpen {
					continue
				}
				side, err := order.StringToOrderSide(spotOrders[x].Orders[y].Side)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
				}
				oType, err := order.StringToOrderType(spotOrders[x].Orders[y].Type)
				if err != nil {
					return nil, err
				}
				status, err := order.StringToOrderStatus(spotOrders[x].Orders[y].Status)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
				}
				orders = append(orders, order.Detail{
					Side:                 side,
					Type:                 oType,
					Status:               status,
					Pair:                 symbol,
					OrderID:              spotOrders[x].Orders[y].OrderID,
					Amount:               spotOrders[x].Orders[y].Amount.Float64(),
					ExecutedAmount:       spotOrders[x].Orders[y].Amount.Float64() - spotOrders[x].Orders[y].Left.Float64(),
					RemainingAmount:      spotOrders[x].Orders[y].Left.Float64(),
					Price:                spotOrders[x].Orders[y].Price.Float64(),
					AverageExecutedPrice: spotOrders[x].Orders[y].AverageFillPrice.Float64(),
					Date:                 spotOrders[x].Orders[y].CreateTime.Time(),
					LastUpdated:          spotOrders[x].Orders[y].UpdateTime.Time(),
					Exchange:             g.Name,
					AssetType:            req.AssetType,
					ClientOrderID:        spotOrders[x].Orders[y].Text,
					FeeAsset:             currency.NewCode(spotOrders[x].Orders[y].FeeCurrency),
				})
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, req.AssetType)
		if err != nil {
			return nil, err
		}
		var futuresOrders []Order
		if req.AssetType == asset.DeliveryFutures {
			futuresOrders, err = g.GetDeliveryOrders(ctx, currency.EMPTYPAIR, statusOpen, settle, "", 0, 0, 0)
		} else {
			futuresOrders, err = g.GetFuturesOrders(ctx, currency.EMPTYPAIR, statusOpen, "", settle, 0, 0, 0)
		}
		if err != nil {
			return nil, err
		}
		for i := range futuresOrders {
			pair, err := currency.NewPairFromString(futuresOrders[i].Contract)
			if err != nil {
				return nil, err
			}

			if futuresOrders[i].Status != statusOpen || (len(req.Pairs) > 0 && !req.Pairs.Contains(pair, true)) {
				continue
			}
			side, amount, remaining := getSideAndAmountFromSize(futuresOrders[i].Size, futuresOrders[i].RemainingAmount)
			tif, err := timeInForceFromString(futuresOrders[i].TimeInForce)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Status:               order.Open,
				Amount:               amount,
				ContractAmount:       amount,
				Pair:                 pair,
				OrderID:              strconv.FormatInt(futuresOrders[i].ID, 10),
				ClientOrderID:        getClientOrderIDFromText(futuresOrders[i].Text),
				Price:                futuresOrders[i].OrderPrice.Float64(),
				ExecutedAmount:       amount - remaining,
				RemainingAmount:      remaining,
				LastUpdated:          futuresOrders[i].FinishTime.Time(),
				Date:                 futuresOrders[i].CreateTime.Time(),
				Exchange:             g.Name,
				AssetType:            req.AssetType,
				Side:                 side,
				Type:                 order.Limit,
				SettlementCurrency:   settle,
				ReduceOnly:           futuresOrders[i].IsReduceOnly,
				TimeInForce:          tif,
				AverageExecutedPrice: futuresOrders[i].FillPrice.Float64(),
			})
		}
	case asset.Options:
		var optionsOrders []OptionOrderResponse
		optionsOrders, err = g.GetOptionFuturesOrders(ctx, currency.EMPTYPAIR, "", statusOpen, 0, 0, req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
		for i := range optionsOrders {
			var currencyPair currency.Pair
			var status order.Status
			currencyPair, err = currency.NewPairFromString(optionsOrders[i].Contract)
			if err != nil {
				return nil, err
			}
			status, err = order.StringToOrderStatus(optionsOrders[i].Status)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Status:          status,
				Amount:          optionsOrders[i].Size,
				Pair:            currencyPair,
				OrderID:         strconv.FormatInt(optionsOrders[i].OptionOrderID, 10),
				Price:           optionsOrders[i].Price.Float64(),
				ExecutedAmount:  optionsOrders[i].Size - optionsOrders[i].Left,
				RemainingAmount: optionsOrders[i].Left,
				LastUpdated:     optionsOrders[i].FinishTime.Time(),
				Date:            optionsOrders[i].CreateTime.Time(),
				Exchange:        g.Name,
				AssetType:       req.AssetType,
				ClientOrderID:   optionsOrders[i].Text,
			})
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(g.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gateio) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	format, err := g.GetPairFormat(req.AssetType, true)
	if err != nil {
		return nil, err
	}
	switch req.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		for x := range req.Pairs {
			fPair := req.Pairs[x].Format(format)
			spotOrders, err := g.GetMySpotTradingHistory(ctx, fPair, req.FromOrderID, 0, 0, req.AssetType == asset.CrossMargin, req.StartTime, req.EndTime)
			if err != nil {
				return nil, err
			}
			for o := range spotOrders {
				var side order.Side
				side, err = order.StringToOrderSide(spotOrders[o].Side)
				if err != nil {
					return nil, err
				}
				detail := order.Detail{
					OrderID:        spotOrders[o].OrderID,
					Amount:         spotOrders[o].Amount.Float64(),
					ExecutedAmount: spotOrders[o].Amount.Float64(),
					Price:          spotOrders[o].Price.Float64(),
					Date:           spotOrders[o].CreateTime.Time(),
					Side:           side,
					Exchange:       g.Name,
					Pair:           fPair,
					AssetType:      req.AssetType,
					Fee:            spotOrders[o].Fee.Float64(),
					FeeAsset:       currency.NewCode(spotOrders[o].FeeCurrency),
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
			var futuresOrder []TradingHistoryItem
			if req.AssetType == asset.DeliveryFutures {
				futuresOrder, err = g.GetMyDeliveryTradingHistory(ctx, settle, req.FromOrderID, fPair, 0, 0, 0, "")
			} else {
				futuresOrder, err = g.GetMyFuturesTradingHistory(ctx, settle, "", req.FromOrderID, fPair, 0, 0, 0)
			}
			if err != nil {
				return nil, err
			}
			for o := range futuresOrder {
				detail := order.Detail{
					OrderID:   strconv.FormatInt(futuresOrder[o].ID, 10),
					Amount:    futuresOrder[o].Size,
					Price:     futuresOrder[o].Price.Float64(),
					Date:      futuresOrder[o].CreateTime.Time(),
					Exchange:  g.Name,
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
			optionOrders, err := g.GetMyOptionsTradingHistory(ctx, fPair.String(), fPair.Upper(), 0, 0, req.StartTime, req.EndTime)
			if err != nil {
				return nil, err
			}
			for o := range optionOrders {
				detail := order.Detail{
					OrderID:   strconv.FormatInt(optionOrders[o].OrderID, 10),
					Amount:    optionOrders[o].Size,
					Price:     optionOrders[o].Price.Float64(),
					Date:      optionOrders[o].CreateTime.Time(),
					Exchange:  g.Name,
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
	return req.Filter(g.Name, orders), nil
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (g *Gateio) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := g.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	var listCandlesticks []kline.Candle
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		candles, err := g.GetCandlesticks(ctx, req.RequestFormatted, 0, start, end, interval)
		if err != nil {
			return nil, err
		}
		listCandlesticks = make([]kline.Candle, len(candles))
		for i := range candles {
			listCandlesticks[i] = kline.Candle{
				Time:   candles[i].Timestamp.Time(),
				Open:   candles[i].OpenPrice.Float64(),
				High:   candles[i].HighestPrice.Float64(),
				Low:    candles[i].LowestPrice.Float64(),
				Close:  candles[i].ClosePrice.Float64(),
				Volume: candles[i].BaseCcyAmount.Float64(),
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
		settle, err := getSettlementCurrency(pair, a)
		if err != nil {
			return nil, err
		}
		var candles []FuturesCandlestick
		if a == asset.DeliveryFutures {
			candles, err = g.GetDeliveryFuturesCandlesticks(ctx, settle, req.RequestFormatted.Upper(), start, end, 0, interval)
		} else {
			candles, err = g.GetFuturesCandlesticks(ctx, settle, req.RequestFormatted.String(), start, end, 0, interval)
		}
		if err != nil {
			return nil, err
		}
		listCandlesticks = make([]kline.Candle, len(candles))
		for i := range candles {
			listCandlesticks[i] = kline.Candle{
				Time:   candles[i].Timestamp.Time(),
				Open:   candles[i].OpenPrice.Float64(),
				High:   candles[i].HighestPrice.Float64(),
				Low:    candles[i].LowestPrice.Float64(),
				Close:  candles[i].ClosePrice.Float64(),
				Volume: candles[i].Volume,
			}
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	return req.ProcessResponse(listCandlesticks)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (g *Gateio) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := g.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	candlestickItems := make([]kline.Candle, 0, req.Size())
	for _, r := range req.RangeHolder.Ranges {
		switch a {
		case asset.Spot, asset.Margin, asset.CrossMargin:
			candles, err := g.GetCandlesticks(ctx, req.RequestFormatted, 0, r.Start.Time, r.End.Time, interval)
			if err != nil {
				return nil, err
			}
			for j := range candles {
				candlestickItems = append(candlestickItems, kline.Candle{
					Time:   candles[j].Timestamp.Time(),
					Open:   candles[j].OpenPrice.Float64(),
					High:   candles[j].HighestPrice.Float64(),
					Low:    candles[j].LowestPrice.Float64(),
					Close:  candles[j].ClosePrice.Float64(),
					Volume: candles[j].QuoteCcyVolume.Float64(),
				})
			}
		case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures:
			settle, err := getSettlementCurrency(pair, a)
			if err != nil {
				return nil, err
			}
			var candles []FuturesCandlestick
			if a == asset.DeliveryFutures {
				candles, err = g.GetDeliveryFuturesCandlesticks(ctx, settle, req.RequestFormatted.Upper(), r.Start.Time, r.End.Time, 0, interval)
			} else {
				candles, err = g.GetFuturesCandlesticks(ctx, settle, req.RequestFormatted.String(), r.Start.Time, r.End.Time, 0, interval)
			}
			if err != nil {
				return nil, err
			}
			for i := range candles {
				candlestickItems = append(candlestickItems, kline.Candle{
					Time:   candles[i].Timestamp.Time(),
					Open:   candles[i].OpenPrice.Float64(),
					High:   candles[i].HighestPrice.Float64(),
					Low:    candles[i].LowestPrice.Float64(),
					Close:  candles[i].ClosePrice.Float64(),
					Volume: candles[i].Volume,
				})
			}
		default:
			return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
		}
	}
	return req.ProcessResponse(candlestickItems)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (g *Gateio) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	chains, err := g.ListCurrencyChain(ctx, cryptocurrency.Upper())
	if err != nil {
		return nil, err
	}
	availableChains := make([]string, 0, len(chains))
	for x := range chains {
		if chains[x].IsDisabled == 0 {
			availableChains = append(availableChains, chains[x].Chain)
		}
	}
	return availableChains, nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (g *Gateio) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := g.UpdateAccountInfo(ctx, assetType)
	return g.CheckTransientError(err)
}

// checkInstrumentAvailabilityInSpot checks whether the instrument is available in the spot exchange
// if so we can use the instrument to retrieve orderbook and ticker information using the spot endpoints.
func (g *Gateio) checkInstrumentAvailabilityInSpot(instrument currency.Pair) (bool, error) {
	availables, err := g.CurrencyPairs.GetPairs(asset.Spot, false)
	if err != nil {
		return false, err
	}
	return availables.Contains(instrument, true), nil
}

// GetFuturesContractDetails returns details about futures contracts
func (g *Gateio) GetFuturesContractDetails(ctx context.Context, a asset.Item) ([]futures.Contract, error) {
	if !a.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !g.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		contracts, err := g.GetAllFutureContracts(ctx, settle)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, len(contracts))
		for i := range contracts {
			name, err := currency.NewPairFromString(contracts[i].Name)
			if err != nil {
				return nil, err
			}
			contractSettlementType := futures.Linear
			switch {
			case name.Base.Equal(currency.BTC) && settle.Equal(currency.BTC):
				contractSettlementType = futures.Inverse
			case !name.Base.Equal(settle) && !settle.Equal(currency.USDT):
				contractSettlementType = futures.Quanto
			}
			c := futures.Contract{
				Exchange:             g.Name,
				Name:                 name,
				Underlying:           name,
				Asset:                a,
				IsActive:             !contracts[i].InDelisting,
				Type:                 futures.Perpetual,
				SettlementType:       contractSettlementType,
				SettlementCurrencies: currency.Currencies{settle},
				Multiplier:           contracts[i].QuantoMultiplier.Float64(),
				MaxLeverage:          contracts[i].LeverageMax.Float64(),
			}
			c.LatestRate = fundingrate.Rate{
				Time: contracts[i].FundingNextApply.Time().Add(-time.Duration(contracts[i].FundingInterval) * time.Second),
				Rate: contracts[i].FundingRate.Decimal(),
			}
			resp[i] = c
		}
		return resp, nil
	case asset.DeliveryFutures:
		contracts, err := g.GetAllDeliveryContracts(ctx, settle)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, len(contracts))
		for i := range contracts {
			name, err := currency.NewPairFromString(contracts[i].Name)
			if err != nil {
				return nil, err
			}
			underlying, err := currency.NewPairFromString(contracts[i].Underlying)
			if err != nil {
				return nil, err
			}
			// no start information, inferring it based on contract type
			// gateio also reuses contracts for kline data, cannot use a lookup to see the first trade
			var s time.Time
			e := contracts[i].ExpireTime.Time()
			ct := futures.LongDated
			switch contracts[i].Cycle {
			case "WEEKLY":
				ct = futures.Weekly
				s = e.Add(-kline.OneWeek.Duration())
			case "BI-WEEKLY":
				ct = futures.Fortnightly
				s = e.Add(-kline.TwoWeek.Duration())
			case "QUARTERLY":
				ct = futures.Quarterly
				s = e.Add(-kline.ThreeMonth.Duration())
			case "BI-QUARTERLY":
				ct = futures.HalfYearly
				s = e.Add(-kline.SixMonth.Duration())
			}
			resp[i] = futures.Contract{
				Exchange:             g.Name,
				Name:                 name,
				Underlying:           underlying,
				Asset:                a,
				StartDate:            s,
				EndDate:              e,
				SettlementType:       futures.Linear,
				IsActive:             !contracts[i].InDelisting,
				Type:                 ct,
				SettlementCurrencies: currency.Currencies{settle},
				MarginCurrency:       currency.Code{},
				Multiplier:           contracts[i].QuantoMultiplier.Float64(),
				MaxLeverage:          contracts[i].LeverageMax.Float64(),
			}
		}
		return resp, nil
	}
	return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (g *Gateio) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !g.SupportsAsset(a) {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	var limits []order.MinMaxLevel
	switch a {
	case asset.Spot:
		pairsData, err := g.ListSpotCurrencyPairs(ctx)
		if err != nil {
			return err
		}

		limits = make([]order.MinMaxLevel, 0, len(pairsData))
		for i := range pairsData {
			if pairsData[i].TradeStatus == "untradable" {
				continue
			}
			pair, err := g.MatchSymbolWithAvailablePairs(pairsData[i].ID, a, true)
			if err != nil {
				return err
			}

			// Minimum base amounts are not always provided this will default to
			// precision for base deployment. This can't be done for quote.
			minBaseAmount := pairsData[i].MinBaseAmount.Float64()
			if minBaseAmount == 0 {
				minBaseAmount = math.Pow10(-int(pairsData[i].AmountPrecision))
			}

			limits = append(limits, order.MinMaxLevel{
				Asset:                   a,
				Pair:                    pair,
				QuoteStepIncrementSize:  math.Pow10(-int(pairsData[i].Precision)),
				AmountStepIncrementSize: math.Pow10(-int(pairsData[i].AmountPrecision)),
				MinimumBaseAmount:       minBaseAmount,
				MinimumQuoteAmount:      pairsData[i].MinQuoteAmount.Float64(),
			})
		}
	default:
		// TODO: Add in other assets
		return fmt.Errorf("%s %w", a, common.ErrNotYetImplemented)
	}

	return g.LoadLimits(limits)
}

// GetHistoricalFundingRates returns historical funding rates for a futures contract
func (g *Gateio) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
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

	fPair, err := g.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}

	records, err := g.GetFutureFundingRates(ctx, r.PaymentCurrency, fPair, 1000)
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
	for i := range records {
		if (!r.EndDate.IsZero() && r.EndDate.Before(records[i].Timestamp.Time())) ||
			(!r.StartDate.IsZero() && r.StartDate.After(records[i].Timestamp.Time())) {
			continue
		}

		fundingRates = append(fundingRates, fundingrate.Rate{
			Rate: decimal.NewFromFloat(records[i].Rate.Float64()),
			Time: records[i].Timestamp.Time(),
		})
	}

	if len(fundingRates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}

	return &fundingrate.HistoricalRates{
		Exchange:        g.Name,
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
func (g *Gateio) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
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
		fPair, err := g.FormatExchangeCurrency(r.Pair, r.Asset)
		if err != nil {
			return nil, err
		}
		contract, err := g.GetFuturesContract(ctx, settle, fPair.String())
		if err != nil {
			return nil, err
		}
		return []fundingrate.LatestRateResponse{
			contractToFundingRate(g.Name, r.Asset, fPair, contract, r.IncludePredictedRate),
		}, nil
	}

	pairs, err := g.GetEnabledPairs(r.Asset)
	if err != nil {
		return nil, err
	}

	contracts, err := g.GetAllFutureContracts(ctx, settle)
	if err != nil {
		return nil, err
	}
	resp := make([]fundingrate.LatestRateResponse, 0, len(contracts))
	for i := range contracts {
		p := strings.ToUpper(contracts[i].Name)
		if !g.IsValidPairString(p) {
			continue
		}
		cp, err := currency.NewPairFromString(p)
		if err != nil {
			return nil, err
		}
		if !pairs.Contains(cp, false) {
			continue
		}
		resp = append(resp, contractToFundingRate(g.Name, r.Asset, cp, &contracts[i], r.IncludePredictedRate))
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
func (g *Gateio) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.CoinMarginedFutures || a == asset.USDTMarginedFutures, nil
}

// GetOpenInterest returns the open interest rate for a given asset pair
// If no pairs are provided, all enabled assets and pairs will be used
// If keys are provided, those asset pairs only need to be available, not enabled
func (g *Gateio) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
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
			if p, errs = g.MatchSymbolWithAvailablePairs(keys[0].Pair().String(), a, false); errs != nil {
				return nil, errs
			}
		}
		contracts, err := g.getOpenInterestContracts(ctx, a, p)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w fetching %s", err, a))
			continue
		}
		for _, c := range contracts {
			if p.IsEmpty() { // If not exactly one key provided
				p, err = g.MatchSymbolWithAvailablePairs(c.contractName(), a, true)
				if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
					errs = common.AppendError(errs, fmt.Errorf("%w from %s contract %s", err, a, c.contractName()))
					continue
				}
				if len(keys) == 0 { // No keys: All enabled pairs
					if enabled, err := g.IsPairEnabled(p, a); err != nil {
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
				Key: key.ExchangePairAsset{
					Exchange: g.Name,
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
	return c.Name
}

func (c *DeliveryContract) openInterest() float64 {
	return c.QuantoMultiplier.Float64() * float64(c.PositionSize) * c.IndexPrice.Float64()
}

func (c *DeliveryContract) contractName() string {
	return c.Name
}

func (g *Gateio) getOpenInterestContracts(ctx context.Context, a asset.Item, p currency.Pair) ([]openInterestContract, error) {
	settle, err := getSettlementCurrency(p, a)
	if err != nil {
		return nil, err
	}
	if a == asset.DeliveryFutures {
		if p != currency.EMPTYPAIR {
			d, err := g.GetDeliveryContract(ctx, settle, p)
			return []openInterestContract{d}, err
		}
		d, err := g.GetAllDeliveryContracts(ctx, settle)
		contracts := make([]openInterestContract, len(d))
		for i := range d {
			contracts[i] = &d[i]
		}
		return contracts, err
	}
	if p != currency.EMPTYPAIR {
		contract, err := g.GetFuturesContract(ctx, settle, p.String())
		return []openInterestContract{contract}, err
	}
	fc, err := g.GetAllFutureContracts(ctx, settle)
	contracts := make([]openInterestContract, len(fc))
	for i := range fc {
		contracts[i] = &fc[i]
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
func (g *Gateio) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := g.CurrencyPairs.IsPairEnabled(cp, a)
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
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

// WebsocketSubmitOrder submits an order to the exchange
// NOTE: Regarding spot orders, fee is applied to purchased currency.
func (g *Gateio) WebsocketSubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate(g.GetTradingRequirements())
	if err != nil {
		return nil, err
	}

	s.Pair, err = g.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	s.Pair = s.Pair.Upper()

	switch s.AssetType {
	case asset.Spot:
		req, err := g.getSpotOrderRequest(s)
		if err != nil {
			return nil, err
		}

		resp, err := g.WebsocketSpotSubmitOrder(ctx, req)
		if err != nil {
			return nil, err
		}
		return g.deriveSpotWebsocketOrderResponse(resp)
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
		amountWithDirection, err := getFutureOrderSize(s)
		if err != nil {
			return nil, err
		}

		resp, err := g.WebsocketFuturesSubmitOrder(ctx, s.AssetType, &ContractOrderCreateParams{
			Contract:    s.Pair,
			Size:        amountWithDirection,
			Price:       strconv.FormatFloat(s.Price, 'f', -1, 64),
			ReduceOnly:  s.ReduceOnly,
			TimeInForce: timeInForceString(s.TimeInForce),
			Text:        s.ClientOrderID,
		})
		if err != nil {
			return nil, err
		}
		return g.deriveFuturesWebsocketOrderResponse(resp)
	default:
		return nil, common.ErrNotYetImplemented
	}
}

// timeInForceString returns the most relevant time-in-force exchange string for a TimeInForce
// Any TIF value that is combined with POC, IOC or FOK will just return that
// Otherwise the lowercase representation is returned
func timeInForceString(tif order.TimeInForce) string {
	switch {
	case tif.Is(order.PostOnly):
		return "poc"
	case tif.Is(order.ImmediateOrCancel):
		return iocTIF
	case tif.Is(order.FillOrKill):
		return fokTIF
	case tif.Is(order.GoodTillCancel):
		return gtcTIF
	default:
		return tif.Lower()
	}
}

func (g *Gateio) deriveSpotWebsocketOrderResponse(responses *WebsocketOrderResponse) (*order.SubmitResponse, error) {
	resp, err := g.deriveSpotWebsocketOrderResponses([]*WebsocketOrderResponse{responses})
	if err != nil {
		return nil, err
	}
	return resp[0], nil
}

// deriveSpotWebsocketOrderResponses returns the order submission responses for spot
func (g *Gateio) deriveSpotWebsocketOrderResponses(responses []*WebsocketOrderResponse) ([]*order.SubmitResponse, error) {
	if len(responses) == 0 {
		return nil, common.ErrNoResponse
	}

	out := make([]*order.SubmitResponse, 0, len(responses))
	for _, resp := range responses {
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
		out = append(out, &order.SubmitResponse{
			Exchange:             g.Name,
			OrderID:              resp.ID,
			AssetType:            resp.Account,
			Pair:                 resp.CurrencyPair,
			ClientOrderID:        resp.Text,
			Date:                 resp.CreateTimeMs.Time(),
			LastUpdated:          resp.UpdateTimeMs.Time(),
			RemainingAmount:      resp.Left.Float64(),
			Amount:               resp.Amount.Float64(),
			Price:                resp.Price.Float64(),
			AverageExecutedPrice: resp.AverageDealPrice.Float64(),
			Type:                 oType,
			Side:                 side,
			Status:               status,
			TimeInForce:          tif,
			Cost:                 cost,
			Purchased:            purchased,
			Fee:                  resp.Fee.Float64(),
			FeeAsset:             resp.FeeCurrency,
		})
	}
	return out, nil
}

func (g *Gateio) deriveFuturesWebsocketOrderResponse(responses *WebsocketFuturesOrderResponse) (*order.SubmitResponse, error) {
	resp, err := g.deriveFuturesWebsocketOrderResponses([]*WebsocketFuturesOrderResponse{responses})
	if err != nil {
		return nil, err
	}
	return resp[0], nil
}

// deriveFuturesWebsocketOrderResponses returns the order submission responses for futures
func (g *Gateio) deriveFuturesWebsocketOrderResponses(responses []*WebsocketFuturesOrderResponse) ([]*order.SubmitResponse, error) {
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
			Exchange:             g.Name,
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

func (g *Gateio) getSpotOrderRequest(s *order.Submit) (*CreateOrderRequest, error) {
	switch {
	case s.Side.IsLong():
		s.Side = order.Buy
	case s.Side.IsShort():
		s.Side = order.Sell
	default:
		return nil, order.ErrSideIsInvalid
	}

	var timeInForce string
	switch s.TimeInForce {
	case order.ImmediateOrCancel, order.FillOrKill, order.GoodTillCancel:
		timeInForce = s.TimeInForce.Lower()
	case order.PostOnly:
		timeInForce = "poc"
	}

	return &CreateOrderRequest{
		Side:         s.Side.Lower(),
		Type:         s.Type.Lower(),
		Account:      g.assetTypeToString(s.AssetType),
		Amount:       types.Number(s.GetTradeAmount(g.GetTradingRequirements())),
		Price:        types.Number(s.Price),
		CurrencyPair: s.Pair,
		Text:         s.ClientOrderID,
		TimeInForce:  timeInForce,
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
