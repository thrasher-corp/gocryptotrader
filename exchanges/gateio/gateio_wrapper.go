package gateio

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// unfundedFuturesAccount defines an error string when an account associated
// with a settlement currency has not been funded. Use specific pairs to avoid
// this error.
const unfundedFuturesAccount = `please transfer funds first to create futures account`

var errNoResponseReceived = errors.New("no response received")

// SetDefaults sets default values for the exchange
func (g *Gateio) SetDefaults() {
	g.Name = "GateIO"
	g.Enabled = true
	g.Verbose = true
	g.API.CredentialsValidator.RequiresKey = true
	g.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	err := g.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.Futures, asset.Margin, asset.CrossMargin, asset.DeliveryFutures, asset.Options)
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
					asset.Futures: true,
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
	g.Websocket = stream.NewWebsocket()
	g.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	g.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	g.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
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

	err = g.Websocket.Setup(&stream.WebsocketSetup{
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
	err = g.Websocket.SetupNewConnection(&stream.ConnectionSetup{
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
	err = g.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		URL:                  futuresWebsocketUsdtURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, incoming []byte) error {
			return g.WsHandleFuturesData(ctx, incoming, asset.Futures)
		},
		Subscriber:               g.FuturesSubscribe,
		Unsubscriber:             g.FuturesUnsubscribe,
		GenerateSubscriptions:    func() (subscription.List, error) { return g.GenerateFuturesDefaultSubscriptions(currency.USDT) },
		Connector:                g.WsFuturesConnect,
		Authenticate:             g.authenticateFutures,
		MessageFilter:            asset.USDTMarginedFutures,
		BespokeGenerateMessageID: g.GenerateWebsocketMessageID,
	})
	if err != nil {
		return err
	}

	// Futures connection - BTC margined
	err = g.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		URL:                  futuresWebsocketBtcURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, incoming []byte) error {
			return g.WsHandleFuturesData(ctx, incoming, asset.Futures)
		},
		Subscriber:               g.FuturesSubscribe,
		Unsubscriber:             g.FuturesUnsubscribe,
		GenerateSubscriptions:    func() (subscription.List, error) { return g.GenerateFuturesDefaultSubscriptions(currency.BTC) },
		Connector:                g.WsFuturesConnect,
		MessageFilter:            asset.CoinMarginedFutures,
		BespokeGenerateMessageID: g.GenerateWebsocketMessageID,
	})
	if err != nil {
		return err
	}

	// TODO: Add BTC margined delivery futures.
	// Futures connection - Delivery - USDT margined
	err = g.Websocket.SetupNewConnection(&stream.ConnectionSetup{
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
	return g.Websocket.SetupNewConnection(&stream.ConnectionSetup{
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
		var available bool
		available, err = g.checkInstrumentAvailabilityInSpot(fPair)
		if err != nil {
			return nil, err
		}
		if a != asset.Spot && !available {
			return nil, fmt.Errorf("%v instrument %v does not have ticker data", a, fPair)
		}
		var tickerNew *Ticker
		tickerNew, err = g.GetTicker(ctx, fPair.String(), "")
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
	case asset.Futures:
		var settle currency.Code
		settle, err = getSettlementFromCurrency(fPair)
		if err != nil {
			return nil, err
		}
		var tickers []FuturesTicker
		tickers, err = g.GetFuturesTickers(ctx, settle, fPair)
		if err != nil {
			return nil, err
		}
		var tick *FuturesTicker
		for x := range tickers {
			if tickers[x].Contract == fPair.String() {
				tick = &tickers[x]
				break
			}
		}
		if tick == nil {
			return nil, errNoTickerData
		}
		tickerData = &ticker.Price{
			Pair:         fPair,
			Low:          tick.Low24H.Float64(),
			High:         tick.High24H.Float64(),
			Last:         tick.Last.Float64(),
			Volume:       tick.Volume24HBase.Float64(),
			QuoteVolume:  tick.Volume24HQuote.Float64(),
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
			if err != nil {
				return nil, err
			}
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
	case asset.DeliveryFutures:
		var settle currency.Code
		settle, err = getSettlementFromCurrency(fPair)
		if err != nil {
			return nil, err
		}
		var tickers []FuturesTicker
		tickers, err = g.GetDeliveryFutureTickers(ctx, settle, fPair)
		if err != nil {
			return nil, err
		}
		for x := range tickers {
			if tickers[x].Contract == fPair.Upper().String() {
				tickerData = &ticker.Price{
					Pair:         fPair,
					Last:         tickers[x].Last.Float64(),
					High:         tickers[x].High24H.Float64(),
					Low:          tickers[x].Low24H.Float64(),
					Volume:       tickers[x].Volume24H.Float64(),
					QuoteVolume:  tickers[x].Volume24HQuote.Float64(),
					ExchangeName: g.Name,
					AssetType:    a,
				}
				break
			}
		}
	}
	err = ticker.ProcessTicker(tickerData)
	if err != nil {
		return nil, err
	}
	return ticker.GetTicker(g.Name, fPair, a)
}

// FetchTicker retrieves a list of tickers.
func (g *Gateio) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tickerNew, err := ticker.GetTicker(g.Name, fPair, assetType)
	if err != nil {
		return g.UpdateTicker(ctx, fPair, assetType)
	}
	return tickerNew, nil
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
	case asset.Futures:
		btcContracts, err := g.GetAllFutureContracts(ctx, currency.BTC)
		if err != nil {
			return nil, err
		}
		usdtContracts, err := g.GetAllFutureContracts(ctx, currency.USDT)
		if err != nil {
			return nil, err
		}
		btcContracts = append(btcContracts, usdtContracts...)
		pairs := make([]currency.Pair, 0, len(btcContracts))
		for x := range btcContracts {
			if btcContracts[x].InDelisting {
				continue
			}
			p := strings.ToUpper(btcContracts[x].Name)
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
	case asset.DeliveryFutures:
		usdtContracts, err := g.GetAllDeliveryContracts(ctx, currency.USDT)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(usdtContracts))
		for x := range usdtContracts {
			if usdtContracts[x].InDelisting {
				continue
			}
			p := strings.ToUpper(usdtContracts[x].Name)
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
	var err error
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		var tickers []Ticker
		tickers, err = g.GetTickers(ctx, currency.EMPTYPAIR.String(), "")
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
	case asset.Futures, asset.DeliveryFutures:
		var tickers []FuturesTicker
		var ticks []FuturesTicker
		for _, settle := range settlementCurrencies {
			// All delivery futures are settled in USDT only, despite the API accepting a settlement currency parameter for all delivery futures endpoints
			if a == asset.DeliveryFutures && !settle.Equal(currency.USDT) {
				continue
			}

			if a == asset.Futures {
				ticks, err = g.GetFuturesTickers(ctx, settle, currency.EMPTYPAIR)
			} else {
				ticks, err = g.GetDeliveryFutureTickers(ctx, settle, currency.EMPTYPAIR)
			}
			if err != nil {
				return err
			}
			tickers = append(tickers, ticks...)
		}
		for x := range tickers {
			currencyPair, err := currency.NewPairFromString(tickers[x].Contract)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers[x].Last.Float64(),
				High:         tickers[x].High24H.Float64(),
				Low:          tickers[x].Low24H.Float64(),
				Volume:       tickers[x].Volume24H.Float64(),
				QuoteVolume:  tickers[x].Volume24HQuote.Float64(),
				ExchangeName: g.Name,
				Pair:         currencyPair,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
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

// FetchOrderbook returns orderbook base on the currency pair
func (g *Gateio) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(g.Name, p, assetType)
	if err != nil {
		return g.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gateio) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	p, err := g.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	var orderbookNew *Orderbook
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
		orderbookNew, err = g.GetOrderbook(ctx, p.String(), "", 0, true)
	case asset.Futures:
		var settle currency.Code
		settle, err = getSettlementFromCurrency(p)
		if err != nil {
			return nil, err
		}
		orderbookNew, err = g.GetFuturesOrderbook(ctx, settle, p.String(), "", 0, true)
	case asset.DeliveryFutures:
		var settle currency.Code
		settle, err = getSettlementFromCurrency(p.Upper())
		if err != nil {
			return nil, err
		}
		orderbookNew, err = g.GetDeliveryOrderbook(ctx, settle, "", p, 0, true)
	case asset.Options:
		orderbookNew, err = g.GetOptionsOrderbook(ctx, p, "", 0, true)
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	if err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        g.Name,
		Asset:           a,
		VerifyOrderbook: g.CanVerifyOrderbook,
		Pair:            p.Upper(),
		LastUpdateID:    orderbookNew.ID,
		LastUpdated:     orderbookNew.Update.Time(),
	}
	book.Bids = make(orderbook.Tranches, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Tranche{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price.Float64(),
		}
	}
	book.Asks = make(orderbook.Tranches, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Tranche{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price.Float64(),
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
	var info account.Holdings
	info.Exchange = g.Name
	var err error
	switch a {
	case asset.Spot:
		var balances []SpotAccount
		balances, err = g.GetSpotAccounts(ctx, currency.EMPTYCODE)
		currencies := make([]account.Balance, len(balances))
		if err != nil {
			return info, err
		}
		for x := range balances {
			currencies[x] = account.Balance{
				Currency: currency.NewCode(balances[x].Currency),
				Total:    balances[x].Available.Float64() - balances[x].Locked.Float64(),
				Hold:     balances[x].Locked.Float64(),
				Free:     balances[x].Available.Float64(),
			}
		}
		info.Accounts = append(info.Accounts, account.SubAccount{
			AssetType:  a,
			Currencies: currencies,
		})
	case asset.Margin, asset.CrossMargin:
		var balances []MarginAccountItem
		balances, err = g.GetMarginAccountList(ctx, currency.EMPTYPAIR)
		if err != nil {
			return info, err
		}
		var currencies []account.Balance
		for x := range balances {
			currencies = append(currencies, account.Balance{
				Currency: currency.NewCode(balances[x].Base.Currency),
				Total:    balances[x].Base.Available.Float64() + balances[x].Base.LockedAmount.Float64(),
				Hold:     balances[x].Base.LockedAmount.Float64(),
				Free:     balances[x].Base.Available.Float64(),
			}, account.Balance{
				Currency: currency.NewCode(balances[x].Quote.Currency),
				Total:    balances[x].Quote.Available.Float64() + balances[x].Quote.LockedAmount.Float64(),
				Hold:     balances[x].Quote.LockedAmount.Float64(),
				Free:     balances[x].Quote.Available.Float64(),
			})
		}
		info.Accounts = append(info.Accounts, account.SubAccount{
			AssetType:  a,
			Currencies: currencies,
		})
	case asset.Futures, asset.DeliveryFutures:
		currencies := make([]account.Balance, 0, 2)
		for x := range settlementCurrencies {
			// All delivery futures are settled in USDT only, despite the API accepting a settlement currency parameter for all delivery futures endpoints
			if a == asset.DeliveryFutures && !settlementCurrencies[x].Equal(currency.USDT) {
				continue
			}

			var balance *FuturesAccount
			if a == asset.Futures {
				balance, err = g.QueryFuturesAccount(ctx, settlementCurrencies[x])
			} else {
				balance, err = g.GetDeliveryFuturesAccounts(ctx, settlementCurrencies[x])
			}
			if err != nil {
				if strings.Contains(err.Error(), unfundedFuturesAccount) {
					if g.Verbose {
						log.Warnf(log.ExchangeSys, "%s %v for settlement: %v", g.Name, err, settlementCurrencies[x])
					}
					continue
				}
				return info, err
			}
			currencies = append(currencies, account.Balance{
				Currency: currency.NewCode(balance.Currency),
				Total:    balance.Total.Float64(),
				Hold:     balance.Total.Float64() - balance.Available.Float64(),
				Free:     balance.Available.Float64(),
			})
		}
		info.Accounts = append(info.Accounts, account.SubAccount{
			AssetType:  a,
			Currencies: currencies,
		})
	case asset.Options:
		var balance *OptionAccount
		balance, err = g.GetOptionAccounts(ctx)
		if err != nil {
			return info, err
		}
		info.Accounts = append(info.Accounts, account.SubAccount{
			AssetType: a,
			Currencies: []account.Balance{
				{
					Currency: currency.NewCode(balance.Currency),
					Total:    balance.Total.Float64(),
					Hold:     balance.Total.Float64() - balance.Available.Float64(),
					Free:     balance.Available.Float64(),
				},
			},
		})
	default:
		return info, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return info, err
	}
	err = account.Process(&info, creds)
	if err != nil {
		return info, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (g *Gateio) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(g.Name, creds, assetType)
	if err != nil {
		return g.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
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
		var tradeData []Trade
		if p.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		tradeData, err = g.GetMarketTrades(ctx, p, 0, "", false, time.Time{}, time.Time{}, 0)
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(tradeData))
		for i := range tradeData {
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Side)
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
				Timestamp:    tradeData[i].CreateTimeMs.Time(),
			}
		}
	case asset.Futures:
		var settle currency.Code
		settle, err = getSettlementFromCurrency(p)
		if err != nil {
			return nil, err
		}
		var futuresTrades []TradingHistoryItem
		futuresTrades, err = g.GetFuturesTradingHistory(ctx, settle, p, 0, 0, "", time.Time{}, time.Time{})
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
	case asset.DeliveryFutures:
		var settle currency.Code
		settle, err = getSettlementFromCurrency(p)
		if err != nil {
			return nil, err
		}
		var deliveryTrades []DeliveryTradingHistory
		deliveryTrades, err = g.GetDeliveryTradingHistory(ctx, settle, "", p.Upper(), 0, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(deliveryTrades))
		for i := range deliveryTrades {
			resp[i] = trade.Data{
				TID:          strconv.FormatInt(deliveryTrades[i].ID, 10),
				Exchange:     g.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        deliveryTrades[i].Price.Float64(),
				Amount:       deliveryTrades[i].Size,
				Timestamp:    deliveryTrades[i].CreateTime.Time(),
			}
		}
	case asset.Options:
		var trades []TradingHistoryItem
		trades, err = g.GetOptionsTradeHistory(ctx, p.Upper(), "", 0, 0, time.Time{}, time.Time{})
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
		response.Date = sOrder.CreateTimeMs.Time()
		response.LastUpdated = sOrder.UpdateTimeMs.Time()
		return response, nil
	case asset.Futures:
		// TODO: See https://www.gate.io/docs/developers/apiv4/en/#create-a-futures-order
		//	* iceberg orders
		//	* auto_size (close_long, close_short)
		// 	* stp_act (self trade prevention)
		settle, err := getSettlementFromCurrency(s.Pair)
		if err != nil {
			return nil, err
		}
		var amountWithDirection float64
		amountWithDirection, err = getFutureOrderSize(s)
		if err != nil {
			return nil, err
		}
		var timeInForce string
		timeInForce, err = getTimeInForce(s)
		if err != nil {
			return nil, err
		}
		fOrder, err := g.PlaceFuturesOrder(ctx, &ContractOrderCreateParams{
			Contract:    s.Pair,
			Size:        amountWithDirection,
			Price:       strconv.FormatFloat(s.Price, 'f', -1, 64), // Cannot be an empty string, requires "0" for market orders.
			Settle:      settle,
			ReduceOnly:  s.ReduceOnly,
			TimeInForce: timeInForce,
			Text:        s.ClientOrderID})
		if err != nil {
			return nil, err
		}
		response, err := s.DeriveSubmitResponse(strconv.FormatInt(fOrder.ID, 10))
		if err != nil {
			return nil, err
		}
		var status = order.Open
		if fOrder.Status != statusOpen {
			status, err = order.StringToOrderStatus(fOrder.FinishAs)
			if err != nil {
				return nil, err
			}
		}
		response.Status = status
		response.Pair = s.Pair
		response.Date = fOrder.CreateTime.Time()
		response.ClientOrderID = getClientOrderIDFromText(fOrder.Text)
		response.ReduceOnly = fOrder.IsReduceOnly
		response.Amount = math.Abs(fOrder.Size)
		response.Price = fOrder.OrderPrice.Float64()
		response.AverageExecutedPrice = fOrder.FillPrice.Float64()
		return response, nil
	case asset.DeliveryFutures:
		settle, err := getSettlementFromCurrency(s.Pair)
		if err != nil {
			return nil, err
		}
		var amountWithDirection float64
		amountWithDirection, err = getFutureOrderSize(s)
		if err != nil {
			return nil, err
		}
		var timeInForce string
		timeInForce, err = getTimeInForce(s)
		if err != nil {
			return nil, err
		}
		newOrder, err := g.PlaceDeliveryOrder(ctx, &ContractOrderCreateParams{
			Contract:    s.Pair,
			Size:        amountWithDirection,
			Price:       strconv.FormatFloat(s.Price, 'f', -1, 64), // Cannot be an empty string, requires "0" for market orders.
			Settle:      settle,
			ReduceOnly:  s.ReduceOnly,
			TimeInForce: timeInForce,
			Text:        s.ClientOrderID,
		})
		if err != nil {
			return nil, err
		}
		response, err := s.DeriveSubmitResponse(strconv.FormatInt(newOrder.ID, 10))
		if err != nil {
			return nil, err
		}
		var status = order.Open
		if newOrder.Status != statusOpen {
			status, err = order.StringToOrderStatus(newOrder.FinishAs)
			if err != nil {
				return nil, err
			}
		}
		response.Status = status
		response.Pair = s.Pair
		response.Date = newOrder.CreateTime.Time()
		response.ClientOrderID = getClientOrderIDFromText(newOrder.Text)
		response.Amount = math.Abs(newOrder.Size)
		response.Price = newOrder.OrderPrice.Float64()
		response.AverageExecutedPrice = newOrder.FillPrice.Float64()
		return response, nil
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
	case asset.Futures, asset.DeliveryFutures:
		var settle currency.Code
		settle, err = getSettlementFromCurrency(fPair)
		if err != nil {
			return err
		}
		if o.AssetType == asset.Futures {
			_, err = g.CancelSingleFuturesOrder(ctx, settle, o.OrderID)
		} else {
			_, err = g.CancelSingleDeliveryOrder(ctx, settle, o.OrderID)
		}
		if err != nil {
			return err
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
	var response order.CancelBatchResponse
	response.Status = map[string]string{}
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
			for x := range cancel {
				response.Status[cancel[x].OrderID] = func() string {
					if cancel[x].Succeeded {
						return order.Cancelled.String()
					}
					return ""
				}()
			}
		}
	case asset.Futures:
		for a := range o {
			cancel, err := g.CancelMultipleFuturesOpenOrders(ctx, o[a].Pair, o[a].Side.Lower(), o[a].Pair.Quote)
			if err != nil {
				return nil, err
			}
			for x := range cancel {
				response.Status[strconv.FormatInt(cancel[x].ID, 10)] = cancel[x].Status
			}
		}
	case asset.DeliveryFutures:
		for a := range o {
			settle, err := getSettlementFromCurrency(o[a].Pair)
			if err != nil {
				return nil, err
			}
			cancel, err := g.CancelMultipleDeliveryOrders(ctx, o[a].Pair, o[a].Side.Lower(), settle)
			if err != nil {
				return nil, err
			}
			for x := range cancel {
				response.Status[strconv.FormatInt(cancel[x].ID, 10)] = cancel[x].Status
			}
		}
	case asset.Options:
		for a := range o {
			cancel, err := g.CancelMultipleOptionOpenOrders(ctx, o[a].Pair, o[a].Pair.String(), o[a].Side.Lower())
			if err != nil {
				return nil, err
			}
			for x := range cancel {
				response.Status[strconv.FormatInt(cancel[x].OptionOrderID, 10)] = cancel[x].Status
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
	case asset.Futures:
		if o.Pair.IsEmpty() {
			return cancelAllOrdersResponse, currency.ErrCurrencyPairEmpty
		}
		var settle currency.Code
		settle, err = getSettlementFromCurrency(o.Pair)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		var cancel []Order
		cancel, err = g.CancelMultipleFuturesOpenOrders(ctx, o.Pair, o.Side.Lower(), settle)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for f := range cancel {
			cancelAllOrdersResponse.Status[strconv.FormatInt(cancel[f].ID, 10)] = cancel[f].Status
		}
	case asset.DeliveryFutures:
		if o.Pair.IsEmpty() {
			return cancelAllOrdersResponse, currency.ErrCurrencyPairEmpty
		}
		var settle currency.Code
		settle, err = getSettlementFromCurrency(o.Pair)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		var cancel []Order
		cancel, err = g.CancelMultipleDeliveryOrders(ctx, o.Pair, o.Side.Lower(), settle)
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
			Date:           spotOrder.CreateTimeMs.Time(),
			LastUpdated:    spotOrder.UpdateTimeMs.Time(),
		}, nil
	case asset.Futures, asset.DeliveryFutures:
		var settle currency.Code
		settle, err = getSettlementFromCurrency(pair)
		if err != nil {
			return nil, err
		}
		var fOrder *Order
		var err error
		if asset.Futures == a {
			fOrder, err = g.GetSingleFuturesOrder(ctx, settle, orderID)
		} else {
			fOrder, err = g.GetSingleDeliveryOrder(ctx, settle, orderID)
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

		ordertype, postonly := getTypeFromTimeInForce(fOrder.TimeInForce)
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
			Type:                 ordertype,
			PostOnly:             postonly,
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
		var spotOrders []SpotOrdersDetail
		spotOrders, err = g.GetSpotOpenOrders(ctx, 0, 0, req.AssetType == asset.CrossMargin)
		if err != nil {
			return nil, err
		}
		for x := range spotOrders {
			var symbol currency.Pair
			symbol, err = currency.NewPairDelimiter(spotOrders[x].CurrencyPair, format.Delimiter)
			if err != nil {
				return nil, err
			}
			for y := range spotOrders[x].Orders {
				if spotOrders[x].Orders[y].Status != statusOpen {
					continue
				}
				var side order.Side
				side, err = order.StringToOrderSide(spotOrders[x].Orders[y].Side)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
				}
				var oType order.Type
				oType, err = order.StringToOrderType(spotOrders[x].Orders[y].Type)
				if err != nil {
					return nil, err
				}
				var status order.Status
				status, err = order.StringToOrderStatus(spotOrders[x].Orders[y].Status)
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
					Date:                 spotOrders[x].Orders[y].CreateTimeMs.Time(),
					LastUpdated:          spotOrders[x].Orders[y].UpdateTimeMs.Time(),
					Exchange:             g.Name,
					AssetType:            req.AssetType,
					ClientOrderID:        spotOrders[x].Orders[y].Text,
					FeeAsset:             currency.NewCode(spotOrders[x].Orders[y].FeeCurrency),
				})
			}
		}
	case asset.Futures, asset.DeliveryFutures:
		settlements := map[currency.Code]bool{}
		if len(req.Pairs) == 0 {
			for x := range settlementCurrencies {
				settlements[settlementCurrencies[x]] = true
			}
		} else {
			for x := range req.Pairs {
				var settle currency.Code
				settle, err = getSettlementFromCurrency(req.Pairs[x])
				if err != nil {
					return nil, err
				}
				settlements[settle] = true
			}
		}

		for settlement := range settlements {
			// All delivery futures are settled in USDT only, despite the API accepting a settlement currency parameter for all delivery futures endpoints
			if req.AssetType == asset.DeliveryFutures && !settlement.Equal(currency.USDT) {
				continue
			}

			var futuresOrders []Order
			if req.AssetType == asset.Futures {
				futuresOrders, err = g.GetFuturesOrders(ctx, currency.EMPTYPAIR, statusOpen, "", settlement, 0, 0, 0)
			} else {
				futuresOrders, err = g.GetDeliveryOrders(ctx, currency.EMPTYPAIR, statusOpen, settlement, "", 0, 0, 0)
			}
			if err != nil {
				if strings.Contains(err.Error(), unfundedFuturesAccount) {
					log.Warnf(log.ExchangeSys, "%s %v", g.Name, err)
					continue
				}
				return nil, err
			}
			for x := range futuresOrders {
				var pair currency.Pair
				pair, err = currency.NewPairFromString(futuresOrders[x].Contract)
				if err != nil {
					return nil, err
				}

				if futuresOrders[x].Status != statusOpen || (len(req.Pairs) > 0 && !req.Pairs.Contains(pair, true)) {
					continue
				}

				side, amount, remaining := getSideAndAmountFromSize(futuresOrders[x].Size, futuresOrders[x].RemainingAmount)
				orders = append(orders, order.Detail{
					Status:               order.Open,
					Amount:               amount,
					ContractAmount:       amount,
					Pair:                 pair,
					OrderID:              strconv.FormatInt(futuresOrders[x].ID, 10),
					ClientOrderID:        getClientOrderIDFromText(futuresOrders[x].Text),
					Price:                futuresOrders[x].OrderPrice.Float64(),
					ExecutedAmount:       amount - remaining,
					RemainingAmount:      remaining,
					LastUpdated:          futuresOrders[x].FinishTime.Time(),
					Date:                 futuresOrders[x].CreateTime.Time(),
					Exchange:             g.Name,
					AssetType:            req.AssetType,
					Side:                 side,
					Type:                 order.Limit,
					SettlementCurrency:   settlement,
					ReduceOnly:           futuresOrders[x].IsReduceOnly,
					PostOnly:             futuresOrders[x].TimeInForce == "poc",
					AverageExecutedPrice: futuresOrders[x].FillPrice.Float64(),
				})
			}
		}
	case asset.Options:
		var optionsOrders []OptionOrderResponse
		optionsOrders, err = g.GetOptionFuturesOrders(ctx, currency.EMPTYPAIR, "", statusOpen, 0, 0, req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
		for x := range optionsOrders {
			var currencyPair currency.Pair
			var status order.Status
			currencyPair, err = currency.NewPairFromString(optionsOrders[x].Contract)
			if err != nil {
				return nil, err
			}
			status, err = order.StringToOrderStatus(optionsOrders[x].Status)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Status:          status,
				Amount:          optionsOrders[x].Size,
				Pair:            currencyPair,
				OrderID:         strconv.FormatInt(optionsOrders[x].OptionOrderID, 10),
				Price:           optionsOrders[x].Price.Float64(),
				ExecutedAmount:  optionsOrders[x].Size - optionsOrders[x].Left,
				RemainingAmount: optionsOrders[x].Left,
				LastUpdated:     optionsOrders[x].FinishTime.Time(),
				Date:            optionsOrders[x].CreateTime.Time(),
				Exchange:        g.Name,
				AssetType:       req.AssetType,
				ClientOrderID:   optionsOrders[x].Text,
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
	err := req.Validate()
	if err != nil {
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
			var spotOrders []SpotPersonalTradeHistory
			spotOrders, err = g.GateIOGetPersonalTradingHistory(ctx, fPair, req.FromOrderID, 0, 0, req.AssetType == asset.CrossMargin, req.StartTime, req.EndTime)
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
	case asset.Futures, asset.DeliveryFutures:
		for x := range req.Pairs {
			fPair := req.Pairs[x].Format(format)
			var settle currency.Code
			settle, err = getSettlementFromCurrency(fPair)
			if err != nil {
				return nil, err
			}
			var futuresOrder []TradingHistoryItem
			if req.AssetType == asset.Futures {
				futuresOrder, err = g.GetMyPersonalTradingHistory(ctx, settle, "", req.FromOrderID, fPair, 0, 0, 0)
			} else {
				futuresOrder, err = g.GetDeliveryPersonalTradingHistory(ctx, settle, req.FromOrderID, fPair, 0, 0, 0, "")
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
			var optionOrders []OptionTradingHistory
			optionOrders, err = g.GetOptionsPersonalTradingHistory(ctx, fPair.String(), fPair.Upper(), 0, 0, req.StartTime, req.EndTime)
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
		var candles []Candlestick
		candles, err = g.GetCandlesticks(ctx, req.RequestFormatted, 0, start, end, interval)
		if err != nil {
			return nil, err
		}
		listCandlesticks = make([]kline.Candle, len(candles))
		for x := range candles {
			listCandlesticks[x] = kline.Candle{
				Time:   candles[x].Timestamp,
				Open:   candles[x].OpenPrice,
				High:   candles[x].HighestPrice,
				Low:    candles[x].LowestPrice,
				Close:  candles[x].ClosePrice,
				Volume: candles[x].QuoteCcyVolume,
			}
		}
	case asset.Futures, asset.DeliveryFutures:
		var settlement currency.Code
		settlement, err = getSettlementFromCurrency(req.RequestFormatted)
		if err != nil {
			return nil, err
		}
		var candles []FuturesCandlestick
		if a == asset.Futures {
			candles, err = g.GetFuturesCandlesticks(ctx, settlement, req.RequestFormatted.String(), start, end, 0, interval)
		} else {
			candles, err = g.GetDeliveryFuturesCandlesticks(ctx, settlement, req.RequestFormatted.Upper(), start, end, 0, interval)
		}
		if err != nil {
			return nil, err
		}
		listCandlesticks = make([]kline.Candle, len(candles))
		for x := range candles {
			listCandlesticks[x] = kline.Candle{
				Time:   candles[x].Timestamp.Time(),
				Open:   candles[x].OpenPrice.Float64(),
				High:   candles[x].HighestPrice.Float64(),
				Low:    candles[x].LowestPrice.Float64(),
				Close:  candles[x].ClosePrice.Float64(),
				Volume: candles[x].Volume,
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
	for b := range req.RangeHolder.Ranges {
		switch a {
		case asset.Spot, asset.Margin, asset.CrossMargin:
			var candles []Candlestick
			candles, err = g.GetCandlesticks(ctx, req.RequestFormatted, 0, req.RangeHolder.Ranges[b].Start.Time, req.RangeHolder.Ranges[b].End.Time, interval)
			if err != nil {
				return nil, err
			}
			for x := range candles {
				candlestickItems = append(candlestickItems, kline.Candle{
					Time:   candles[x].Timestamp,
					Open:   candles[x].OpenPrice,
					High:   candles[x].HighestPrice,
					Low:    candles[x].LowestPrice,
					Close:  candles[x].ClosePrice,
					Volume: candles[x].QuoteCcyVolume,
				})
			}
		case asset.Futures, asset.DeliveryFutures:
			var settle currency.Code
			settle, err = getSettlementFromCurrency(req.RequestFormatted)
			if err != nil {
				return nil, err
			}
			var candles []FuturesCandlestick
			if a == asset.Futures {
				candles, err = g.GetFuturesCandlesticks(ctx, settle, req.RequestFormatted.String(), req.RangeHolder.Ranges[b].Start.Time, req.RangeHolder.Ranges[b].End.Time, 0, interval)
			} else {
				candles, err = g.GetDeliveryFuturesCandlesticks(ctx, settle, req.RequestFormatted.Upper(), req.RangeHolder.Ranges[b].Start.Time, req.RangeHolder.Ranges[b].End.Time, 0, interval)
			}
			if err != nil {
				return nil, err
			}
			for x := range candles {
				candlestickItems = append(candlestickItems, kline.Candle{
					Time:   candles[x].Timestamp.Time(),
					Open:   candles[x].OpenPrice.Float64(),
					High:   candles[x].HighestPrice.Float64(),
					Low:    candles[x].LowestPrice.Float64(),
					Close:  candles[x].ClosePrice.Float64(),
					Volume: candles[x].Volume,
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
func (g *Gateio) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !g.SupportsAsset(item) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	switch item {
	case asset.Futures:
		var resp []futures.Contract
		for k := range settlementCurrencies {
			contracts, err := g.GetAllFutureContracts(ctx, settlementCurrencies[k])
			if err != nil {
				return nil, err
			}
			contractsToAdd := make([]futures.Contract, len(contracts))
			for j := range contracts {
				var name currency.Pair
				name, err = currency.NewPairFromString(contracts[j].Name)
				if err != nil {
					return nil, err
				}
				contractSettlementType := futures.Linear
				switch {
				case name.Base.Equal(currency.BTC) && settlementCurrencies[k].Equal(currency.BTC):
					contractSettlementType = futures.Inverse
				case !name.Base.Equal(settlementCurrencies[k]) && !settlementCurrencies[k].Equal(currency.USDT):
					contractSettlementType = futures.Quanto
				}
				c := futures.Contract{
					Exchange:             g.Name,
					Name:                 name,
					Underlying:           name,
					Asset:                item,
					IsActive:             !contracts[j].InDelisting,
					Type:                 futures.Perpetual,
					SettlementType:       contractSettlementType,
					SettlementCurrencies: currency.Currencies{settlementCurrencies[k]},
					Multiplier:           contracts[j].QuantoMultiplier.Float64(),
					MaxLeverage:          contracts[j].LeverageMax.Float64(),
				}
				if contracts[j].FundingRate > 0 {
					c.LatestRate = fundingrate.Rate{
						Time: contracts[j].FundingNextApply.Time().Add(-time.Duration(contracts[j].FundingInterval) * time.Second),
						Rate: contracts[j].FundingRate.Decimal(),
					}
				}
				contractsToAdd[j] = c
			}
			resp = append(resp, contractsToAdd...)
		}
		return resp, nil
	case asset.DeliveryFutures:
		var resp []futures.Contract
		contracts, err := g.GetAllDeliveryContracts(ctx, currency.USDT)
		if err != nil {
			return nil, err
		}
		contractsToAdd := make([]futures.Contract, len(contracts))
		for j := range contracts {
			var name, underlying currency.Pair
			name, err = currency.NewPairFromString(contracts[j].Name)
			if err != nil {
				return nil, err
			}
			underlying, err = currency.NewPairFromString(contracts[j].Underlying)
			if err != nil {
				return nil, err
			}
			var ct futures.ContractType
			// no start information, inferring it based on contract type
			// gateio also reuses contracts for kline data, cannot use a lookup to see the first trade
			var s, e time.Time
			e = contracts[j].ExpireTime.Time()
			switch contracts[j].Cycle {
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
			default:
				ct = futures.LongDated
			}
			contractsToAdd[j] = futures.Contract{
				Exchange:             g.Name,
				Name:                 name,
				Underlying:           underlying,
				Asset:                item,
				StartDate:            s,
				EndDate:              e,
				SettlementType:       futures.Linear,
				IsActive:             !contracts[j].InDelisting,
				Type:                 ct,
				SettlementCurrencies: currency.Currencies{currency.USDT},
				MarginCurrency:       currency.Code{},
				Multiplier:           contracts[j].QuantoMultiplier.Float64(),
				MaxLeverage:          contracts[j].LeverageMax.Float64(),
			}
			resp = append(resp, contractsToAdd...)
		}
		return resp, nil
	}
	return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (g *Gateio) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !g.SupportsAsset(a) {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	var limits []order.MinMaxLevel
	switch a {
	case asset.Spot:
		var pairsData []CurrencyPairDetail
		pairsData, err := g.ListSpotCurrencyPairs(ctx)
		if err != nil {
			return err
		}

		limits = make([]order.MinMaxLevel, 0, len(pairsData))
		for x := range pairsData {
			if pairsData[x].TradeStatus == "untradable" {
				continue
			}
			var pair currency.Pair
			pair, err = g.MatchSymbolWithAvailablePairs(pairsData[x].ID, a, true)
			if err != nil {
				return err
			}

			// Minimum base amounts are not always provided this will default to
			// precision for base deployment. This can't be done for quote.
			minBaseAmount := pairsData[x].MinBaseAmount.Float64()
			if minBaseAmount == 0 {
				minBaseAmount = math.Pow10(-int(pairsData[x].AmountPrecision))
			}

			limits = append(limits, order.MinMaxLevel{
				Asset:                   a,
				Pair:                    pair,
				QuoteStepIncrementSize:  math.Pow10(-int(pairsData[x].Precision)),
				AmountStepIncrementSize: math.Pow10(-int(pairsData[x].AmountPrecision)),
				MinimumBaseAmount:       minBaseAmount,
				MinimumQuoteAmount:      pairsData[x].MinQuoteAmount.Float64(),
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
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}

	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	if !r.StartDate.IsZero() && !r.EndDate.IsZero() {
		err := common.StartEndTimeCheck(r.StartDate, r.EndDate)
		if err != nil {
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
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}

	if !r.Pair.IsEmpty() {
		resp := make([]fundingrate.LatestRateResponse, 1)
		fPair, err := g.FormatExchangeCurrency(r.Pair, r.Asset)
		if err != nil {
			return nil, err
		}
		var settle currency.Code
		settle, err = getSettlementFromCurrency(fPair)
		if err != nil {
			return nil, err
		}
		contract, err := g.GetSingleContract(ctx, settle, fPair.String())
		if err != nil {
			return nil, err
		}
		resp[0] = contractToFundingRate(g.Name, r.Asset, fPair, contract, r.IncludePredictedRate)
		return resp, nil
	}

	var resp []fundingrate.LatestRateResponse
	pairs, err := g.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}

	for i := range settlementCurrencies {
		contracts, err := g.GetAllFutureContracts(ctx, settlementCurrencies[i])
		if err != nil {
			return nil, err
		}
		for j := range contracts {
			p := strings.ToUpper(contracts[j].Name)
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
			var isPerp bool
			isPerp, err = g.IsPerpetualFutureCurrency(r.Asset, cp)
			if err != nil {
				return nil, err
			}
			if !isPerp {
				continue
			}
			resp = append(resp, contractToFundingRate(g.Name, r.Asset, cp, &contracts[j], r.IncludePredictedRate))
		}
	}

	return resp, nil
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
	return a == asset.Futures, nil
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (g *Gateio) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.DeliveryFutures && k[i].Asset != asset.Futures {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	if len(k) == 1 {
		p, isEnabled, err := g.MatchSymbolCheckEnabled(k[0].Pair().String(), k[0].Asset, false)
		if err != nil {
			return nil, err
		}
		if !isEnabled {
			return nil, fmt.Errorf("%w %v", asset.ErrNotEnabled, k[0].Pair())
		}
		switch k[0].Asset {
		case asset.DeliveryFutures:
			contractResp, err := g.GetSingleDeliveryContracts(ctx, currency.USDT, p)
			if err != nil {
				return nil, err
			}
			openInterest := contractResp.QuantoMultiplier.Float64() * float64(contractResp.PositionSize) * contractResp.IndexPrice.Float64()
			return []futures.OpenInterest{
				{
					Key: key.ExchangePairAsset{
						Exchange: g.Name,
						Base:     k[0].Base,
						Quote:    k[0].Quote,
						Asset:    k[0].Asset,
					},
					OpenInterest: openInterest,
				},
			}, nil
		case asset.Futures:
			for _, s := range settlementCurrencies {
				contractResp, err := g.GetSingleContract(ctx, s, p.String())
				if err != nil {
					continue
				}
				openInterest := contractResp.QuantoMultiplier.Float64() * float64(contractResp.PositionSize) * contractResp.IndexPrice.Float64()
				return []futures.OpenInterest{
					{
						Key: key.ExchangePairAsset{
							Exchange: g.Name,
							Base:     k[0].Base,
							Quote:    k[0].Quote,
							Asset:    k[0].Asset,
						},
						OpenInterest: openInterest,
					},
				}, nil
			}
		}
	}
	var resp []futures.OpenInterest
	for _, a := range g.GetAssetTypes(true) {
		switch a {
		case asset.DeliveryFutures:
			contractResp, err := g.GetAllDeliveryContracts(ctx, currency.USDT)
			if err != nil {
				return nil, err
			}

			for i := range contractResp {
				p, isEnabled, err := g.MatchSymbolCheckEnabled(contractResp[i].Name, a, true)
				if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
					return nil, err
				}
				if !isEnabled {
					continue
				}
				var appendData bool
				for j := range k {
					if k[j].Pair().Equal(p) {
						appendData = true
						break
					}
				}
				if len(k) > 0 && !appendData {
					continue
				}
				openInterest := contractResp[i].QuantoMultiplier.Float64() * float64(contractResp[i].PositionSize) * contractResp[i].IndexPrice.Float64()
				resp = append(resp, futures.OpenInterest{
					Key: key.ExchangePairAsset{
						Exchange: g.Name,
						Base:     p.Base.Item,
						Quote:    p.Quote.Item,
						Asset:    a,
					},
					OpenInterest: openInterest,
				})
			}
		case asset.Futures:
			for _, s := range settlementCurrencies {
				contractResp, err := g.GetAllFutureContracts(ctx, s)
				if err != nil {
					return nil, err
				}

				for i := range contractResp {
					p, isEnabled, err := g.MatchSymbolCheckEnabled(contractResp[i].Name, a, true)
					if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
						return nil, err
					}
					if !isEnabled {
						continue
					}
					var appendData bool
					for j := range k {
						if k[j].Pair().Equal(p) {
							appendData = true
							break
						}
					}
					if len(k) > 0 && !appendData {
						continue
					}
					openInterest := contractResp[i].QuantoMultiplier.Float64() * float64(contractResp[i].PositionSize) * contractResp[i].IndexPrice.Float64()
					resp = append(resp, futures.OpenInterest{
						Key: key.ExchangePairAsset{
							Exchange: g.Name,
							Base:     p.Base.Item,
							Quote:    p.Quote.Item,
							Asset:    a,
						},
						OpenInterest: openInterest,
					})
				}
			}
		}
	}
	return resp, nil
}

// getClientOrderIDFromText returns the client order ID from the text response
func getClientOrderIDFromText(text string) string {
	if strings.HasPrefix(text, "t-") {
		return text
	}
	return ""
}

// getTypeFromTimeInForce returns the order type and if the order is post only
func getTypeFromTimeInForce(tif string) (orderType order.Type, postOnly bool) {
	switch tif {
	case "ioc":
		return order.Market, false
	case "fok":
		return order.Market, false
	case "poc":
		return order.Limit, true
	default:
		return order.Limit, false
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
		return 0, errInvalidOrderSide
	}
}

var errPostOnlyOrderTypeUnsupported = errors.New("post only is only supported for limit orders")

// getTimeInForce returns the time in force for a given order. If Market order
// IOC
func getTimeInForce(s *order.Submit) (string, error) {
	timeInForce := "gtc" // limit order taker/maker
	if s.Type == order.Market || s.ImmediateOrCancel {
		timeInForce = "ioc" // market taker only
	}
	if s.PostOnly {
		if s.Type != order.Limit {
			return "", fmt.Errorf("%w not for %v", errPostOnlyOrderTypeUnsupported, s.Type)
		}
		timeInForce = "poc" // limit order maker only
	}
	if s.FillOrKill {
		timeInForce = "fok" // market order entire fill or kill
	}
	return timeInForce, nil
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
		return tradeBaseURL + tradeSpot + cp.Upper().String(), nil
	case asset.Futures:
		return tradeBaseURL + tradeFutures + cp.Upper().String(), nil
	case asset.DeliveryFutures:
		return tradeBaseURL + tradeDelivery + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

// WebsocketSubmitOrder submits an order to the exchange through the websocket
// connection.
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
		var req *CreateOrderRequest
		req, err = g.getSpotOrderRequest(s)
		if err != nil {
			return nil, err
		}

		var got []WebsocketOrderResponse
		got, err = g.WebsocketSpotSubmitOrder(ctx, req)
		if err != nil {
			return nil, err
		}

		if len(got) == 0 {
			return nil, errNoResponseReceived
		}

		var resp *order.SubmitResponse
		resp, err = s.DeriveSubmitResponse(got[0].ID)
		if err != nil {
			return nil, err
		}
		resp.Side, err = order.StringToOrderSide(got[0].Side)
		if err != nil {
			return nil, err
		}
		resp.Status, err = order.StringToOrderStatus(got[0].Status)
		if err != nil {
			return nil, err
		}
		resp.Pair = s.Pair
		resp.Date = got[0].CreateTime.Time()
		resp.ClientOrderID = got[0].Text
		resp.Date = got[0].CreateTimeMs.Time()
		resp.LastUpdated = got[0].UpdateTimeMs.Time()
		return resp, nil
	case asset.Futures:
		var amountWithDirection float64
		amountWithDirection, err = getFutureOrderSize(s)
		if err != nil {
			return nil, err
		}

		var timeInForce string
		timeInForce, err = getTimeInForce(s)
		if err != nil {
			return nil, err
		}

		var got []WebsocketFuturesOrderResponse
		got, err = g.WebsocketFuturesSubmitOrder(ctx, &ContractOrderCreateParams{
			Contract:    s.Pair,
			Size:        amountWithDirection,
			Price:       strconv.FormatFloat(s.Price, 'f', -1, 64),
			ReduceOnly:  s.ReduceOnly,
			TimeInForce: timeInForce,
			Text:        s.ClientOrderID,
		})
		if err != nil {
			return nil, err
		}

		if len(got) == 0 {
			return nil, errNoResponseReceived
		}

		resp, err := s.DeriveSubmitResponse(strconv.FormatInt(got[0].ID, 10))
		if err != nil {
			return nil, err
		}
		resp.Status = order.Open
		if got[0].Status != statusOpen {
			resp.Status, err = order.StringToOrderStatus(got[0].FinishAs)
			if err != nil {
				return nil, err
			}
		}
		resp.Pair = s.Pair
		resp.Date = got[0].CreateTime.Time()
		resp.ClientOrderID = getClientOrderIDFromText(got[0].Text)
		resp.ReduceOnly = s.ReduceOnly
		resp.Amount = math.Abs(float64(got[0].Size))
		resp.Price = got[0].FillPrice.Float64()
		resp.AverageExecutedPrice = got[0].FillPrice.Float64()
		return resp, nil
	default:
		return nil, common.ErrNotYetImplemented
	}
}

func (g *Gateio) getSpotOrderRequest(s *order.Submit) (*CreateOrderRequest, error) {
	switch {
	case s.Side.IsLong():
		s.Side = order.Buy
	case s.Side.IsShort():
		s.Side = order.Sell
	default:
		return nil, errInvalidOrderSide
	}

	timeInForce, err := getTimeInForce(s)
	if err != nil {
		return nil, err
	}

	return &CreateOrderRequest{
		Side:    s.Side.Lower(),
		Type:    s.Type.Lower(),
		Account: g.assetTypeToString(s.AssetType),
		// When doing spot market orders when purchasing base currency, the quote currency amount is used. When selling
		// the base currency the base currency amount is used.
		Amount:       types.Number(s.GetTradeAmount(g.GetTradingRequirements())),
		Price:        types.Number(s.Price),
		CurrencyPair: s.Pair,
		Text:         s.ClientOrderID,
		TimeInForce:  timeInForce,
	}, nil
}
