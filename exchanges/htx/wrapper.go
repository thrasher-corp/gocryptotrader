package htx

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
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
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
)

// SetDefaults sets default values for the exchange
func (e *Exchange) SetDefaults() {
	e.Name = "HTX"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	for _, a := range []asset.Item{asset.Spot, asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Futures} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		}
		switch a {
		case asset.Spot:
			ps.RequestFormat.Uppercase = false
		case asset.CoinMarginedFutures, asset.USDTMarginedFutures:
			ps.RequestFormat.Delimiter = currency.DashDelimiter
		}
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
		TradingRequirements: protocol.TradingRequirements{
			SpotMarketBuyQuotation: true,
			SpotMarketSellBase:     true,
		},
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:                 true,
				TickerBatching:                 true,
				KlineFetching:                  true,
				TradeFetching:                  true,
				OrderbookFetching:              true,
				AutoPairUpdates:                true,
				AccountInfo:                    true,
				GetOrder:                       true,
				GetOrders:                      true,
				CancelOrders:                   true,
				CancelOrder:                    true,
				SubmitOrder:                    true,
				CryptoDeposit:                  true,
				CryptoWithdrawal:               true,
				TradeFee:                       true,
				MultiChainDeposits:             true,
				MultiChainWithdrawals:          true,
				HasAssetTypeAccountSegregation: true,
				FundingRateFetching:            true,
				PredictedFundingRate:           true,
			},
			WebsocketCapabilities: protocol.Features{
				KlineFetching:          true,
				OrderbookFetching:      true,
				TradeFetching:          true,
				SubmitOrder:            true,
				SubmitOrders:           true,
				CancelOrder:            true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				MessageCorrelation:     true,
				GetOrder:               true,
				GetOrders:              true,
				TickerFetching:         true,
				FundingRateFetching:    true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.EightHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.CoinMarginedFutures: true,
					asset.USDTMarginedFutures: true,
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
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.OneYear},
					// NOTE: The supported time intervals below are returned
					// offset to the Asia/Shanghai time zone. This may lead to
					// issues with candle quality and conversion as the
					// intervals may be broken up. The below intervals
					// are constructed from hourly candles.
					// kline.IntervalCapacity{Interval: kline.OneDay},
					// kline.IntervalCapacity{Interval: kline.OneWeek},
					// kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 2000,
			},
		},
		Subscriptions: append(defaultSubscriptions.Clone(), defaultFuturesSubscriptions.Clone()...),
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                     htxAPIURL,
		exchange.RestFutures:                  htxFuturesURL,
		exchange.RestCoinMargined:             htxFuturesURL,
		exchange.RestUSDTMargined:             htxFuturesURL,
		exchange.WebsocketSpot:                wsSpotURL + wsPublicPath,
		exchange.WebsocketPrivate:             wsSpotURL + wsPrivatePath,
		exchange.WebsocketFutures:             wsFuturesURL,
		exchange.WebsocketFuturesPrivate:      wsFuturesPrivateURL,
		exchange.WebsocketCoinMargined:        wsCoinMarginedURL,
		exchange.WebsocketCoinMarginedPrivate: wsCoinMarginedPrivateURL,
		exchange.WebsocketUSDTMargined:        wsUSDTMarginedURL,
		exchange.WebsocketUSDTMarginedPrivate: wsUSDTMarginedPrivateURL,
		exchange.WebsocketTrade:               wsUSDTMarginedTradeURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Bootstrap ensures that future contract expiry codes are loaded if AutoPairUpdates is not enabled
func (e *Exchange) Bootstrap(ctx context.Context) (continueBootstrap bool, err error) {
	continueBootstrap = true

	if !e.GetEnabledFeatures().AutoPairUpdates && e.SupportsAsset(asset.Futures) {
		_, err = e.FetchTradablePairs(ctx, asset.Futures)
	}

	return continueBootstrap, err
}

// Setup sets user configuration
func (e *Exchange) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	err = e.SetupDefaults(exch)
	if err != nil {
		return err
	}

	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exch,
		Features:                     &e.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement: true,
	}); err != nil {
		return err
	}

	for _, ws := range []struct {
		endpoint exchange.URL
		asset    asset.Item
		private  bool
		trade    bool
	}{
		{endpoint: exchange.WebsocketSpot, asset: asset.Spot},
		{endpoint: exchange.WebsocketPrivate, asset: asset.Spot, private: true},
		{endpoint: exchange.WebsocketFutures, asset: asset.Futures},
		{endpoint: exchange.WebsocketFuturesPrivate, asset: asset.Futures, private: true},
		{endpoint: exchange.WebsocketCoinMargined, asset: asset.CoinMarginedFutures},
		{endpoint: exchange.WebsocketCoinMarginedPrivate, asset: asset.CoinMarginedFutures, private: true},
		{endpoint: exchange.WebsocketUSDTMargined, asset: asset.USDTMarginedFutures},
		{endpoint: exchange.WebsocketUSDTMarginedPrivate, asset: asset.USDTMarginedFutures, private: true},
		{endpoint: exchange.WebsocketTrade, asset: asset.USDTMarginedFutures, private: true, trade: true},
	} {
		runningURL, err := e.API.Endpoints.GetURL(ws.endpoint)
		if err != nil {
			return err
		}
		rateLimitDuration := 20 * time.Millisecond
		if ws.private && ws.asset != asset.Spot {
			rateLimitDuration = 25 * time.Millisecond
		}
		if ws.trade {
			rateLimitDuration = 3 * time.Second / 24
		}
		setup := &websocket.ConnectionSetup{
			URL:                      runningURL,
			RateLimit:                request.NewWeightedRateLimitByDuration(rateLimitDuration),
			ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:         exch.WebsocketResponseMaxLimit,
			Connector:                e.wsConnect,
			Handler:                  e.wsHandleData,
			Subscriber:               e.subscribeConnection,
			Unsubscriber:             e.unsubscribeConnection,
			MessageFilter:            ws.endpoint,
			SubscriptionsNotRequired: ws.trade,
			GenerateSubscriptions: func() (subscription.List, error) {
				if ws.trade {
					return nil, nil
				}
				return e.generateSubscriptionsForAsset(ws.asset, ws.private)
			},
		}
		if ws.private {
			setup.Authenticate = e.wsAuthenticateConnection
		}
		if ws.trade {
			setup.ConnectionEnabled = e.Websocket.CanUseAuthenticatedEndpoints
		}
		if err := e.Websocket.SetupNewConnection(setup); err != nil {
			return err
		}
	}
	return nil
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}

	var pairs []currency.Pair
	switch a {
	case asset.Spot:
		symbols, err := e.GetSymbols(ctx)
		if err != nil {
			return nil, err
		}

		pairs = make([]currency.Pair, 0, len(symbols))
		for x := range symbols {
			if symbols[x].State != "online" {
				continue
			}

			pair, err := currency.NewPairFromStrings(symbols[x].BaseCurrency,
				symbols[x].QuoteCurrency)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	case asset.CoinMarginedFutures:
		symbols, err := e.GetSwapMarkets(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}

		pairs = make([]currency.Pair, 0, len(symbols))
		for z := range symbols {
			if symbols[z].ContractStatus != 1 {
				continue
			}
			pair, err := currency.NewPairFromString(symbols[z].ContractCode)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	case asset.USDTMarginedFutures:
		symbols, err := e.GetLinearSwapMarkets(ctx, currency.EMPTYPAIR, "", "swap", "swap")
		if err != nil {
			return nil, err
		}

		pairs = make([]currency.Pair, 0, len(symbols))
		for z := range symbols {
			if symbols[z].ContractStatus != 1 {
				continue
			}
			pair, err := currency.NewPairFromString(symbols[z].ContractCode)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	case asset.Futures:
		symbols, err := e.FGetContractInfo(ctx, "", "", currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		pairs = make([]currency.Pair, 0, len(symbols.Data))
		expiryCodeDates := map[string]currency.Code{}
		for i := range symbols.Data {
			c := symbols.Data[i]
			if c.ContractStatus != 1 {
				continue
			}
			pair, err := currency.NewPairFromString(c.ContractCode)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
			if cType, ok := contractExpiryNames[c.ContractType]; ok {
				if v, ok := expiryCodeDates[cType]; !ok {
					expiryCodeDates[cType] = currency.NewCode(pair.Quote.String())
				} else if v.String() != pair.Quote.String() {
					return nil, fmt.Errorf("%w: %s (%s vs %s)", errInconsistentContractExpiry, cType, v.String(), pair.Quote.String())
				}
			}
		}
		// We cache contract expiries on the exchange locally right now because there's no exchange base holder for them
		// It's not as dangerous as it seems, because when contracts change, so would tradeable pairs,
		// so by caching them in FetchTradablePairs we're not adding any extra-layer of out-of-date data
		e.futureContractCodesMutex.Lock()
		e.futureContractCodes = expiryCodeDates
		e.futureContractCodesMutex.Unlock()
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assets := e.GetAssetTypes(false)
	for x := range assets {
		pairs, err := e.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, assets[x], false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateCurrencyStates refreshes spot trading, deposit and withdrawal availability.
func (e *Exchange) UpdateCurrencyStates(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	currencies, err := e.GetCurrenciesIncludingChains(ctx, currency.EMPTYCODE)
	if err != nil {
		return err
	}
	updates := make(map[currency.Code]currencystate.Options, len(currencies))
	for i := range currencies {
		var canDeposit, canWithdraw bool
		for j := range currencies[i].ChainData {
			if currencies[i].ChainData[j] == nil {
				continue
			}
			canDeposit = canDeposit || currencies[i].ChainData[j].DepositStatus == "allowed"
			canWithdraw = canWithdraw || currencies[i].ChainData[j].WithdrawStatus == "allowed"
		}
		updates[currency.NewCode(currencies[i].Currency)] = currencystate.Options{
			Deposit:  convert.BoolPtr(canDeposit),
			Withdraw: convert.BoolPtr(canWithdraw),
			Trade:    convert.BoolPtr(currencies[i].InstStatus == "normal"),
		}
	}
	return e.States.UpdateAll(a, updates)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	var errs error
	switch a {
	case asset.Spot:
		ticks, err := e.GetTickers(ctx)
		if err != nil {
			return err
		}
		for i := range ticks.Data {
			var cp currency.Pair
			cp, _, err = e.MatchSymbolCheckEnabled(ticks.Data[i].Symbol, a, false)
			if err != nil {
				if !errors.Is(err, currency.ErrPairNotFound) {
					errs = common.AppendError(errs, err)
				}
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				High:         ticks.Data[i].High,
				Low:          ticks.Data[i].Low,
				Bid:          ticks.Data[i].Bid,
				Ask:          ticks.Data[i].Ask,
				Volume:       ticks.Data[i].Amount,
				QuoteVolume:  ticks.Data[i].Volume,
				Open:         ticks.Data[i].Open,
				Close:        ticks.Data[i].Close,
				BidSize:      ticks.Data[i].BidSize,
				AskSize:      ticks.Data[i].AskSize,
				Pair:         cp,
				ExchangeName: e.Name,
				AssetType:    a,
				LastUpdated:  time.Now(),
			})
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		}
	case asset.CoinMarginedFutures:
		ticks, err := e.GetBatchCoinMarginSwapContracts(ctx)
		if err != nil {
			return err
		}
		for i := range ticks {
			var cp currency.Pair
			cp, _, err = e.MatchSymbolCheckEnabled(ticks[i].ContractCode, a, true)
			if err != nil {
				if !errors.Is(err, currency.ErrPairNotFound) {
					errs = common.AppendError(errs, err)
				}
				continue
			}
			if len(ticks[i].Bid) < 2 {
				errs = common.AppendError(errs, fmt.Errorf("%w for %s", errInvalidBidData, cp))
				continue
			}
			if len(ticks[i].Ask) < 2 {
				errs = common.AppendError(errs, fmt.Errorf("%w for %s", errInvalidAskData, cp))
				continue
			}
			tt := ticks[i].Timestamp.Time()
			err = ticker.ProcessTicker(&ticker.Price{
				High:         ticks[i].High.Float64(),
				Low:          ticks[i].Low.Float64(),
				Volume:       ticks[i].Amount.Float64(),
				QuoteVolume:  ticks[i].Volume.Float64(),
				Open:         ticks[i].Open.Float64(),
				Close:        ticks[i].Close.Float64(),
				Bid:          ticks[i].Bid[0],
				BidSize:      ticks[i].Bid[1],
				Ask:          ticks[i].Ask[0],
				AskSize:      ticks[i].Ask[1],
				Pair:         cp,
				ExchangeName: e.Name,
				AssetType:    a,
				LastUpdated:  tt,
			})
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		}
	case asset.USDTMarginedFutures:
		ticks, err := e.GetBatchLinearSwapContracts(ctx)
		if err != nil {
			return err
		}
		for i := range ticks {
			cp, _, err := e.MatchSymbolCheckEnabled(ticks[i].ContractCode, a, true)
			if err != nil {
				if !errors.Is(err, currency.ErrPairNotFound) {
					errs = common.AppendError(errs, err)
				}
				continue
			}
			if len(ticks[i].Bid) < 2 {
				errs = common.AppendError(errs, fmt.Errorf("%w for %s", errInvalidBidData, cp))
				continue
			}
			if len(ticks[i].Ask) < 2 {
				errs = common.AppendError(errs, fmt.Errorf("%w for %s", errInvalidAskData, cp))
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				High:         ticks[i].High.Float64(),
				Low:          ticks[i].Low.Float64(),
				Volume:       ticks[i].Amount.Float64(),
				QuoteVolume:  ticks[i].Volume.Float64(),
				Open:         ticks[i].Open.Float64(),
				Close:        ticks[i].Close.Float64(),
				Bid:          ticks[i].Bid[0],
				BidSize:      ticks[i].Bid[1],
				Ask:          ticks[i].Ask[0],
				AskSize:      ticks[i].Ask[1],
				Pair:         cp,
				ExchangeName: e.Name,
				AssetType:    a,
				LastUpdated:  ticks[i].Timestamp.Time(),
			})
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		}
	case asset.Futures:
		ticks := []FuturesBatchTicker{}
		if futureTicks, err := e.GetBatchFuturesContracts(ctx); err != nil {
			errs = common.AppendError(errs, err)
		} else {
			ticks = append(ticks, futureTicks...)
		}
		for i := range ticks {
			var cp currency.Pair
			var err error
			if ticks[i].Symbol != "" {
				cp, err = currency.NewPairFromString(ticks[i].Symbol)
				if err == nil {
					cp, err = e.pairFromContractExpiryCode(cp)
				}
				if err == nil {
					cp, _, err = e.MatchSymbolCheckEnabled(cp.String(), a, true)
				}
			} else {
				cp, _, err = e.MatchSymbolCheckEnabled(ticks[i].ContractCode, a, true)
			}
			if err != nil {
				if !errors.Is(err, currency.ErrPairNotFound) {
					errs = common.AppendError(errs, err)
				}
				continue
			}
			if len(ticks[i].Bid) < 2 {
				errs = common.AppendError(errs, fmt.Errorf("%w for %s", errInvalidBidData, cp))
				continue
			}
			if len(ticks[i].Ask) < 2 {
				errs = common.AppendError(errs, fmt.Errorf("%w for %s", errInvalidAskData, cp))
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				High:         ticks[i].High.Float64(),
				Low:          ticks[i].Low.Float64(),
				Volume:       ticks[i].Amount.Float64(),
				QuoteVolume:  ticks[i].Volume.Float64(),
				Open:         ticks[i].Open.Float64(),
				Close:        ticks[i].Close.Float64(),
				Bid:          ticks[i].Bid[0],
				BidSize:      ticks[i].Bid[1],
				Ask:          ticks[i].Ask[0],
				AskSize:      ticks[i].Ask[1],
				Pair:         cp,
				ExchangeName: e.Name,
				AssetType:    a,
				LastUpdated:  ticks[i].Timestamp.Time(),
			})
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		}
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	return errs
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	switch a {
	case asset.Spot:
		tickerData, err := e.Get24HrMarketSummary(ctx, p)
		if err != nil {
			return nil, err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			High:         tickerData.Tick.High,
			Low:          tickerData.Tick.Low,
			Volume:       tickerData.Tick.Amount,
			QuoteVolume:  tickerData.Tick.Volume,
			Open:         tickerData.Tick.Open,
			Close:        tickerData.Tick.Close,
			Pair:         p,
			ExchangeName: e.Name,
			AssetType:    asset.Spot,
		})
		if err != nil {
			return nil, err
		}
	case asset.CoinMarginedFutures:
		marketData, err := e.GetSwapMarketOverview(ctx, p)
		if err != nil {
			return nil, err
		}

		if len(marketData.Tick.Bid) == 0 {
			return nil, errInvalidBidData
		}
		if len(marketData.Tick.Ask) == 0 {
			return nil, errInvalidAskData
		}

		err = ticker.ProcessTicker(&ticker.Price{
			High:         marketData.Tick.High.Float64(),
			Low:          marketData.Tick.Low.Float64(),
			Volume:       marketData.Tick.Amount.Float64(),
			QuoteVolume:  marketData.Tick.Vol.Float64(),
			Open:         marketData.Tick.Open.Float64(),
			Close:        marketData.Tick.Close.Float64(),
			Pair:         p,
			Bid:          marketData.Tick.Bid[0],
			Ask:          marketData.Tick.Ask[0],
			ExchangeName: e.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	case asset.USDTMarginedFutures:
		marketData, err := e.GetLinearSwapMarketOverview(ctx, p)
		if err != nil {
			return nil, err
		}

		if len(marketData.Tick.Bid) == 0 {
			return nil, errInvalidBidData
		}
		if len(marketData.Tick.Ask) == 0 {
			return nil, errInvalidAskData
		}

		err = ticker.ProcessTicker(&ticker.Price{
			High:         marketData.Tick.High.Float64(),
			Low:          marketData.Tick.Low.Float64(),
			Volume:       marketData.Tick.Amount.Float64(),
			QuoteVolume:  marketData.Tick.Vol.Float64(),
			Open:         marketData.Tick.Open.Float64(),
			Close:        marketData.Tick.Close.Float64(),
			Pair:         p,
			Bid:          marketData.Tick.Bid[0],
			Ask:          marketData.Tick.Ask[0],
			ExchangeName: e.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	case asset.Futures:
		marketData, err := e.FGetMarketOverviewData(ctx, p)
		if err != nil {
			return nil, err
		}
		if len(marketData.Tick.Bid) == 0 {
			return nil, errInvalidBidData
		}
		if len(marketData.Tick.Ask) == 0 {
			return nil, errInvalidAskData
		}

		err = ticker.ProcessTicker(&ticker.Price{
			High:         marketData.Tick.High.Float64(),
			Low:          marketData.Tick.Low.Float64(),
			Volume:       marketData.Tick.Amount.Float64(),
			QuoteVolume:  marketData.Tick.Vol.Float64(),
			Open:         marketData.Tick.Open.Float64(),
			Close:        marketData.Tick.Close.Float64(),
			Pair:         p,
			Bid:          marketData.Tick.Bid[0],
			Ask:          marketData.Tick.Ask[0],
			ExchangeName: e.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	return e.UpdateOrderbookWithLimit(ctx, p, assetType, 0)
}

// UpdateOrderbookWithLimit updates and returns an orderbook capped to the requested depth.
func (e *Exchange) UpdateOrderbookWithLimit(ctx context.Context, p currency.Pair, assetType asset.Item, limit uint64) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !assetType.IsValid() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	var err error
	switch assetType {
	case asset.Spot:
		var orderbookNew *Orderbook
		orderbookNew, err = e.GetDepth(ctx,
			&OrderBookDataRequestParams{
				Symbol: p,
				Type:   OrderBookDataRequestParamsTypeStep0,
			})
		if err != nil {
			return book, err
		}

		book.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
		for x := range orderbookNew.Bids {
			book.Bids[x] = orderbook.Level{
				Amount: orderbookNew.Bids[x][1],
				Price:  orderbookNew.Bids[x][0],
			}
		}
		book.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
		for x := range orderbookNew.Asks {
			book.Asks[x] = orderbook.Level{
				Amount: orderbookNew.Asks[x][1],
				Price:  orderbookNew.Asks[x][0],
			}
		}

	case asset.Futures:
		var orderbookNew *OBData
		orderbookNew, err = e.FGetMarketDepth(ctx, p, "step0")
		if err != nil {
			return book, err
		}

		book.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
		for x := range orderbookNew.Asks {
			book.Asks[x] = orderbook.Level{
				Amount: orderbookNew.Asks[x].Quantity,
				Price:  orderbookNew.Asks[x].Price,
			}
		}
		book.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
		for y := range orderbookNew.Bids {
			book.Bids[y] = orderbook.Level{
				Amount: orderbookNew.Bids[y].Quantity,
				Price:  orderbookNew.Bids[y].Price,
			}
		}

	case asset.CoinMarginedFutures:
		var orderbookNew SwapMarketDepthData
		orderbookNew, err = e.GetSwapMarketDepth(ctx, p, "step0")
		if err != nil {
			return book, err
		}

		book.Asks = make(orderbook.Levels, len(orderbookNew.Tick.Asks))
		for x := range orderbookNew.Tick.Asks {
			book.Asks[x] = orderbook.Level{
				Amount: orderbookNew.Tick.Asks[x][1],
				Price:  orderbookNew.Tick.Asks[x][0],
			}
		}

		book.Bids = make(orderbook.Levels, len(orderbookNew.Tick.Bids))
		for y := range orderbookNew.Tick.Bids {
			book.Bids[y] = orderbook.Level{
				Amount: orderbookNew.Tick.Bids[y][1],
				Price:  orderbookNew.Tick.Bids[y][0],
			}
		}
	case asset.USDTMarginedFutures:
		var orderbookNew SwapMarketDepthData
		orderbookNew, err = e.GetLinearSwapMarketDepth(ctx, p, "step0")
		if err != nil {
			return book, err
		}

		book.Asks = make(orderbook.Levels, len(orderbookNew.Tick.Asks))
		for x := range orderbookNew.Tick.Asks {
			book.Asks[x] = orderbook.Level{
				Amount: orderbookNew.Tick.Asks[x][1],
				Price:  orderbookNew.Tick.Asks[x][0],
			}
		}

		book.Bids = make(orderbook.Levels, len(orderbookNew.Tick.Bids))
		for y := range orderbookNew.Tick.Bids {
			book.Bids[y] = orderbook.Level{
				Amount: orderbookNew.Tick.Bids[y][1],
				Price:  orderbookNew.Tick.Bids[y][0],
			}
		}
	default:
		return book, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if limit != 0 {
		book.Asks = book.Asks[:min(uint64(len(book.Asks)), limit)]
		book.Bids = book.Bids[:min(uint64(len(book.Bids)), limit)]
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, p, assetType)
}

// GetAccountID returns the account ID for trades
func (e *Exchange) GetAccountID(ctx context.Context) ([]Account, error) {
	acc, err := e.GetAccounts(ctx)
	if err != nil {
		return nil, err
	}

	if len(acc) < 1 {
		return nil, errNoAccountReturned
	}

	return acc, nil
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (subAccts accounts.SubAccounts, err error) {
	switch assetType {
	case asset.Spot:
		resp, err := e.GetAccountID(ctx)
		if err != nil {
			return nil, err
		}
		subAccts = make(accounts.SubAccounts, 0, len(resp))
		for i := range resp {
			if resp[i].Type != "spot" {
				continue
			}
			a := accounts.NewSubAccount(assetType, strconv.FormatInt(resp[i].ID, 10))
			balances, err := e.GetAccountBalance(ctx, a.ID)
			if err != nil {
				return nil, err
			}
			for j := range balances {
				amount := balances[j].Balance.Float64()
				code := balances[j].Currency.Upper()
				switch balances[j].Type {
				case "frozen":
					err = a.Balances.Add(code, accounts.Balance{Total: amount, Hold: amount})
				case "trade":
					err = a.Balances.Add(code, accounts.Balance{Total: amount, Free: amount})
				default:
					continue
				}
				if err != nil {
					return nil, err
				}
			}
			subAccts = subAccts.Merge(a)
		}
	case asset.CoinMarginedFutures:
		mainResp, err := e.GetSwapAccountInfo(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		subAccts = accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
		for i := range mainResp.Data {
			subAccts[0].Balances.Set(mainResp.Data[i].Symbol, accounts.Balance{
				Total: mainResp.Data[i].MarginBalance,
				Hold:  mainResp.Data[i].MarginFrozen,
				Free:  mainResp.Data[i].MarginAvailable,
			})
		}
		subResp, err := e.GetSwapAllSubAccAssets(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		for i := range subResp.Data {
			resp, err := e.SwapSingleSubAccAssets(ctx, currency.EMPTYPAIR, subResp.Data[i].SubUID)
			if err != nil {
				return nil, err
			}
			a := accounts.NewSubAccount(assetType, strconv.FormatInt(subResp.Data[i].SubUID, 10))
			for j := range resp.Data {
				a.Balances.Set(resp.Data[j].Symbol, accounts.Balance{
					Total: resp.Data[j].MarginBalance,
					Hold:  resp.Data[j].MarginFrozen,
					Free:  resp.Data[j].MarginAvailable,
				})
			}
			subAccts = subAccts.Merge(a)
		}
	case asset.USDTMarginedFutures:
		resp, err := e.GetV5AccountBalance(ctx)
		if err != nil {
			return nil, err
		}
		subAccts = accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
		for i := range resp.Data.Details {
			curr := currency.NewCode(resp.Data.Details[i].Currency)
			free := resp.Data.Details[i].Available.Float64() + resp.Data.Details[i].IsolatedAvailable.Float64()
			subAccts[0].Balances.Set(curr, accounts.Balance{
				Total: resp.Data.Details[i].Equity.Float64(),
				Hold:  resp.Data.Details[i].Equity.Float64() - free,
				Free:  free,
			})
		}
	case asset.Futures:
		mainResp, err := e.FGetAccountInfo(ctx, currency.EMPTYCODE)
		if err != nil {
			return nil, err
		}
		subAccts = accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
		for i := range mainResp.AccData {
			subAccts[0].Balances.Set(mainResp.AccData[i].Symbol, accounts.Balance{
				Total: mainResp.AccData[i].MarginBalance,
				Hold:  mainResp.AccData[i].MarginFrozen,
				Free:  mainResp.AccData[i].MarginAvailable,
			})
		}
		subResp, err := e.FGetAllSubAccountAssets(ctx, currency.EMPTYCODE)
		if err != nil {
			return nil, err
		}
		for i := range subResp.Data {
			a := accounts.NewSubAccount(assetType, strconv.FormatInt(subResp.Data[i].SubUID, 10))
			resp, err := e.FGetSingleSubAccountInfo(ctx, "", a.ID)
			if err != nil {
				return nil, err
			}
			for j := range resp.AssetsData {
				a.Balances.Set(resp.AssetsData[j].Symbol, accounts.Balance{
					Total: resp.AssetsData[j].MarginBalance,
					Hold:  resp.AssetsData[j].MarginFrozen,
					Free:  resp.AssetsData[j].MarginAvailable,
				})
			}
			subAccts = subAccts.Merge(a)
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals.
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	const resultLimit = 500
	deposits, err := e.SearchForExistedWithdrawsAndDeposits(ctx, currency.EMPTYCODE, "deposit", "next", 0, resultLimit)
	if err != nil {
		return nil, err
	}
	withdrawals, err := e.SearchForExistedWithdrawsAndDeposits(ctx, currency.EMPTYCODE, "withdraw", "next", 0, resultLimit)
	if err != nil {
		return nil, err
	}
	records := make([]exchange.FundingHistory, 0, len(deposits.Data)+len(withdrawals.Data))
	histories := make([]WithdrawalData, len(deposits.Data)+len(withdrawals.Data))
	copy(histories, deposits.Data)
	copy(histories[len(deposits.Data):], withdrawals.Data)
	for i := range histories {
		history := &histories[i]
		records = append(records, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          history.State,
			TransferID:      strconv.FormatInt(history.ID, 10),
			Description:     history.ErrorMessage,
			Timestamp:       history.CreatedAt.Time(),
			Currency:        history.Currency.Upper().String(),
			Amount:          history.Amount,
			Fee:             history.Fee,
			TransferType:    history.Type,
			CryptoToAddress: history.Address,
			CryptoTxID:      history.TransactionHash,
			CryptoChain:     history.Chain,
		})
	}
	return records, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	withdrawals, err := e.SearchForExistedWithdrawsAndDeposits(ctx, c, "withdraw", "", 0, 500)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals.Data))
	for i := range withdrawals.Data {
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawals.Data[i].State,
			TransferID:      withdrawals.Data[i].TransactionHash,
			Timestamp:       withdrawals.Data[i].CreatedAt.Time(),
			Currency:        withdrawals.Data[i].Currency.Upper().String(),
			Amount:          withdrawals.Data[i].Amount,
			Fee:             withdrawals.Data[i].Fee,
			TransferType:    withdrawals.Data[i].Type,
			CryptoToAddress: withdrawals.Data[i].Address,
			CryptoTxID:      withdrawals.Data[i].TransactionHash,
			CryptoChain:     withdrawals.Data[i].Chain,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	var resp []trade.Data
	pFmt, err := e.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}

	p = p.Format(pFmt)
	switch a {
	case asset.Spot:
		var sTrades []TradeHistory
		sTrades, err = e.GetTradeHistory(ctx, p, 2000)
		if err != nil {
			return nil, err
		}
		for i := range sTrades {
			for j := range sTrades[i].Trades {
				var side order.Side
				side, err = order.StringToOrderSide(sTrades[i].Trades[j].Direction)
				if err != nil {
					return nil, err
				}
				resp = append(resp, trade.Data{
					Exchange:     e.Name,
					TID:          strconv.FormatFloat(sTrades[i].Trades[j].TradeID, 'f', -1, 64),
					CurrencyPair: p,
					AssetType:    a,
					Side:         side,
					Price:        sTrades[i].Trades[j].Price,
					Amount:       sTrades[i].Trades[j].Amount,
					Timestamp:    sTrades[i].Timestamp.Time(),
				})
			}
		}
	case asset.Futures:
		var fTrades FBatchTradesForContractData
		fTrades, err = e.FRequestPublicBatchTrades(ctx, p, 2000)
		if err != nil {
			return nil, err
		}
		for i := range fTrades.Data {
			for j := range fTrades.Data[i].Data {
				var side order.Side
				if fTrades.Data[i].Data[j].Direction != "" {
					side, err = order.StringToOrderSide(fTrades.Data[i].Data[j].Direction)
					if err != nil {
						return nil, err
					}
				}
				resp = append(resp, trade.Data{
					Exchange:     e.Name,
					TID:          strconv.FormatInt(fTrades.Data[i].Data[j].ID, 10),
					CurrencyPair: p,
					AssetType:    a,
					Side:         side,
					Price:        fTrades.Data[i].Data[j].Price,
					Amount:       fTrades.Data[i].Data[j].Amount,
					Timestamp:    fTrades.Data[i].Data[j].Timestamp.Time(),
				})
			}
		}
	case asset.CoinMarginedFutures:
		var cTrades BatchTradesData
		cTrades, err = e.GetBatchTrades(ctx, p, 2000)
		if err != nil {
			return nil, err
		}
		for i := range cTrades.Data {
			var side order.Side
			if cTrades.Data[i].Direction != "" {
				side, err = order.StringToOrderSide(cTrades.Data[i].Direction)
				if err != nil {
					return nil, err
				}
			}
			resp = append(resp, trade.Data{
				Exchange:     e.Name,
				TID:          strconv.FormatInt(cTrades.Data[i].ID, 10),
				CurrencyPair: p,
				AssetType:    a,
				Side:         side,
				Price:        cTrades.Data[i].Price,
				Amount:       cTrades.Data[i].Amount,
				Timestamp:    cTrades.Data[i].Timestamp.Time(),
			})
		}
	case asset.USDTMarginedFutures:
		var linearTrades BatchTradesData
		linearTrades, err = e.GetLinearSwapBatchTrades(ctx, p, 2000)
		if err != nil {
			return nil, err
		}
		for i := range linearTrades.Data {
			var side order.Side
			if linearTrades.Data[i].Direction != "" {
				side, err = order.StringToOrderSide(linearTrades.Data[i].Direction)
				if err != nil {
					return nil, err
				}
			}
			resp = append(resp, trade.Data{
				Exchange:     e.Name,
				TID:          strconv.FormatInt(linearTrades.Data[i].ID, 10),
				CurrencyPair: p,
				AssetType:    a,
				Side:         side,
				Price:        linearTrades.Data[i].Price,
				Amount:       linearTrades.Data[i].Amount,
				Timestamp:    linearTrades.Data[i].Timestamp.Time(),
			})
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}

	err = e.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// getV5PositionModeName returns the current position mode required when placing V5 orders.
func (e *Exchange) getV5PositionModeName(ctx context.Context) (string, error) {
	mode, err := e.GetV5PositionMode(ctx)
	if err != nil {
		return "", err
	}
	if mode == nil || mode.Data.PositionMode == "" {
		return "", errEmptyResult
	}
	switch mode.Data.PositionMode {
	case "single_side", "dual_side":
		return mode.Data.PositionMode, nil
	default:
		return "", fmt.Errorf("%w %q", errInvalidPositionMode, mode.Data.PositionMode)
	}
}

// formatV5OrderRequest maps the generic order model to HTX's current position and margin modes.
func (e *Exchange) formatV5OrderRequest(s *order.Submit, positionMode string) (*V5OrderRequest, error) {
	if s == nil {
		return nil, order.ErrSubmissionIsNil
	}
	if s.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	formattedPair, err := e.FormatSymbol(s.Pair, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	req := &V5OrderRequest{
		ContractCode:  formattedPair,
		MarginMode:    "cross",
		PositionSide:  "both",
		ClientOrderID: s.ClientOrderID,
		Volume:        types.Number(s.Amount),
		TimeInForce:   "gtc",
	}
	switch s.MarginType {
	case margin.Unset, margin.Multi:
	case margin.Isolated:
		req.MarginMode = "isolated"
	default:
		return nil, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, s.MarginType)
	}
	switch {
	case s.Side.IsLong():
		req.Side = "buy"
		if positionMode == "dual_side" {
			req.PositionSide = "long"
		}
	case s.Side.IsShort():
		req.Side = "sell"
		if positionMode == "dual_side" {
			req.PositionSide = "short"
		}
	default:
		return nil, order.ErrSideIsInvalid
	}
	switch s.Type {
	case order.Market:
		req.Type = "market"
	case order.Limit:
		req.Type = "limit"
		req.Price = types.Number(s.Price)
	default:
		return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, s.Type)
	}
	switch {
	case s.TimeInForce.Is(order.PostOnly):
		req.Type = orderPriceTypePostOnly
	case s.TimeInForce == order.UnknownTIF, s.TimeInForce.Is(order.GoodTillCancel):
	case s.TimeInForce.Is(order.ImmediateOrCancel):
		req.TimeInForce = "ioc"
	case s.TimeInForce.Is(order.FillOrKill):
		req.TimeInForce = "fok"
	default:
		return nil, fmt.Errorf("%w %v", order.ErrUnsupportedTimeInForce, s.TimeInForce)
	}
	if s.ReduceOnly {
		req.ReduceOnly = 1
	}
	return req, nil
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}

	var orderID string
	switch s.AssetType {
	case asset.Spot:
		accountID, err := strconv.ParseInt(s.ClientID, 10, 64)
		if err != nil {
			return nil, err
		}
		var formattedType SpotNewOrderRequestParamsType
		params := SpotNewOrderRequestParams{
			AccountID:     int(accountID),
			ClientOrderID: s.ClientOrderID,
			Amount:        s.GetTradeAmount(e.GetTradingRequirements()),
			Source:        "api",
			Symbol:        s.Pair,
		}
		switch {
		case s.Side.IsLong() && s.Type == order.Market:
			formattedType = SpotNewOrderRequestTypeBuyMarket
		case s.Side.IsShort() && s.Type == order.Market:
			formattedType = SpotNewOrderRequestTypeSellMarket
		case s.Side.IsLong() && s.Type == order.Limit:
			formattedType = SpotNewOrderRequestTypeBuyLimit
			params.Price = s.Price
		case s.Side.IsShort() && s.Type == order.Limit:
			formattedType = SpotNewOrderRequestTypeSellLimit
			params.Price = s.Price
		}
		params.Type = formattedType
		response, err := e.SpotNewOrder(ctx, &params)
		if err != nil {
			return nil, err
		}
		orderID = strconv.FormatInt(response, 10)

	case asset.CoinMarginedFutures:
		var oDirection string
		switch {
		case s.Side.IsLong():
			oDirection = "BUY"
		case s.Side.IsShort():
			oDirection = "SELL"
		}
		var oType string
		switch s.Type {
		case order.Market:
			// https://huobiapi.github.io/docs/dm/v1/en/#order-and-trade
			// At present, HTX Futures does not support unlimited slippage market price when placing an order.
			// To increase the probability of a transaction, users can choose to place an order based on BBO price (opponent),
			// optimal 5 (optimal_5), optimal 10 (optimal_10), optimal 20 (optimal_20), among which the success probability of
			// optimal 20 is the largest, while the slippage always is the largest as well.
			//
			// It is important to note that the above methods will not guarantee the order to be fully-filled
			// The exchange will obtain the optimal N price when the order is placed
			oType = "optimal_20"
			switch {
			case s.TimeInForce.Is(order.ImmediateOrCancel):
				oType = "optimal_20_ioc"
			case s.TimeInForce.Is(order.FillOrKill):
				oType = "optimal_20_fok"
			}
		case order.Limit:
			oType = "limit"
			if s.TimeInForce.Is(order.PostOnly) {
				oType = orderPriceTypePostOnly
			}
		default:
			oType = "opponent"
		}
		offset := "open"
		if s.ReduceOnly {
			offset = orderOffsetClose
		}
		orderResp, err := e.PlaceSwapOrders(ctx,
			s.Pair,
			s.ClientOrderID,
			oDirection,
			offset,
			oType,
			s.Price,
			s.Amount,
			s.Leverage)
		if err != nil {
			return nil, err
		}
		orderID = orderResp.Data.OrderIDString
	case asset.USDTMarginedFutures:
		positionMode, err := e.getV5PositionModeName(ctx)
		if err != nil {
			return nil, err
		}
		v5Req, err := e.formatV5OrderRequest(s, positionMode)
		if err != nil {
			return nil, err
		}
		orderResp, err := e.PlaceV5Order(ctx, v5Req)
		if err != nil {
			return nil, err
		}
		if orderResp == nil {
			return nil, errEmptyResult
		}
		orderID = orderResp.Data.OrderID
	case asset.Futures:
		var oDirection string
		switch {
		case s.Side.IsLong():
			oDirection = "BUY"
		case s.Side.IsShort():
			oDirection = "SELL"
		}
		var oType string
		switch s.Type {
		case order.Market:
			// https://huobiapi.github.io/docs/dm/v1/en/#order-and-trade
			// At present, HTX Futures does not support unlimited slippage market price when placing an order.
			// To increase the probability of a transaction, users can choose to place an order based on BBO price (opponent),
			// optimal 5 (optimal_5), optimal 10 (optimal_10), optimal 20 (optimal_20), among which the success probability of
			// optimal 20 is the largest, while the slippage always is the largest as well.
			//
			// It is important to note that the above methods will not guarantee the order to be fully-filled
			// The exchange will obtain the optimal N price when the order is placed
			oType = "optimal_20"
			switch {
			case s.TimeInForce.Is(order.ImmediateOrCancel):
				oType = "optimal_20_ioc"
			case s.TimeInForce.Is(order.FillOrKill):
				oType = "optimal_20_fok"
			}
		case order.Limit:
			oType = "limit"
			if s.TimeInForce.Is(order.PostOnly) {
				oType = orderPriceTypePostOnly
			}
		default:
			oType = "opponent"
		}
		offset := "open"
		if s.ReduceOnly {
			offset = orderOffsetClose
		}
		o, err := e.FOrder(ctx,
			s.Pair,
			"",
			"",
			s.ClientOrderID,
			oDirection,
			offset,
			oType,
			s.Price,
			s.Amount,
			s.Leverage)
		if err != nil {
			return nil, err
		}
		orderID = o.Data.OrderIDStr
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, s.AssetType)
	}
	resp, err := s.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Status = order.New
	return resp, nil
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WebsocketSubmitOrder submits a USDT-margined order through HTX's V5 trade connection.
func (e *Exchange) WebsocketSubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	if s.AssetType != asset.USDTMarginedFutures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, s.AssetType)
	}
	positionMode, err := e.getV5PositionModeName(ctx)
	if err != nil {
		return nil, err
	}
	req, err := e.formatV5OrderRequest(s, positionMode)
	if err != nil {
		return nil, err
	}
	orderResp, err := e.WSPlaceV5Order(ctx, req)
	if err != nil {
		return nil, err
	}
	if orderResp == nil {
		return nil, errEmptyResult
	}
	resp, err := s.DeriveSubmitResponse(orderResp.Data.OrderID)
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = orderResp.Data.ClientOrderID
	resp.Status = order.New
	return resp, nil
}

// WebsocketSubmitOrders submits USDT-margined orders through HTX's V5 trade connection.
func (e *Exchange) WebsocketSubmitOrders(ctx context.Context, orders []*order.Submit) ([]*order.SubmitResponse, error) {
	if len(orders) == 0 {
		return nil, common.ErrEmptyParams
	}
	requests := make([]*V5OrderRequest, len(orders))
	for i := range orders {
		if err := orders[i].Validate(e.GetTradingRequirements()); err != nil {
			return nil, err
		}
		if orders[i].AssetType != asset.USDTMarginedFutures {
			return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, orders[i].AssetType)
		}
	}
	positionMode, err := e.getV5PositionModeName(ctx)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		requests[i], err = e.formatV5OrderRequest(orders[i], positionMode)
		if err != nil {
			return nil, err
		}
	}
	orderResp, err := e.WSPlaceV5BatchOrders(ctx, requests)
	if err != nil {
		return nil, err
	}
	if orderResp == nil {
		return nil, errEmptyResult
	}
	if len(orderResp.Data) != len(orders) {
		return nil, fmt.Errorf("%w: got %d, expected %d", errUnexpectedBatchResponseCount, len(orderResp.Data), len(orders))
	}
	responses := make([]*order.SubmitResponse, len(orders))
	for i := range orders {
		responses[i], err = orders[i].DeriveSubmitResponse(orderResp.Data[i].OrderID)
		if err != nil {
			return nil, err
		}
		responses[i].ClientOrderID = orderResp.Data[i].ClientOrderID
		responses[i].Status = order.New
		if orderResp.Data[i].Code != 0 && orderResp.Data[i].Code != 200 {
			responses[i].SubmissionError = fmt.Errorf("%d %w", orderResp.Data[i].Code, htxError(orderResp.Data[i].Message))
		}
	}
	return responses, nil
}

// WebsocketCancelOrder cancels a USDT-margined order through HTX's V5 trade connection.
func (e *Exchange) WebsocketCancelOrder(ctx context.Context, ord *order.Cancel) error {
	if ord == nil {
		return order.ErrCancelOrderIsNil
	}
	if err := ord.Validate(ord.PairAssetRequired()); err != nil {
		return err
	}
	if ord.AssetType != asset.USDTMarginedFutures {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, ord.AssetType)
	}
	if ord.OrderID == "" && ord.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	formattedPair, err := e.FormatSymbol(ord.Pair, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	resp, err := e.WSCancelV5Order(ctx, &V5CancelOrderRequest{
		ContractCode:  formattedPair,
		OrderID:       ord.OrderID,
		ClientOrderID: ord.ClientOrderID,
	})
	if err != nil {
		return err
	}
	if resp == nil {
		return errEmptyResult
	}
	if resp.Data.Code != 0 && resp.Data.Code != 200 {
		return fmt.Errorf("%d %w", resp.Data.Code, htxError(resp.Data.Message))
	}
	return nil
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if o == nil {
		return order.ErrCancelOrderIsNil
	}
	if err := o.Validate(o.PairAssetRequired()); err != nil {
		return err
	}
	if o.OrderID == "" && o.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	var err error
	switch o.AssetType {
	case asset.Spot:
		if o.OrderID == "" {
			var cancelledOrders *CancelOrderBatch
			cancelledOrders, err = e.CancelOrderBatch(ctx, nil, []string{o.ClientOrderID})
			if err != nil {
				return err
			}
			if cancelledOrders == nil {
				return errEmptyResult
			}
			for i := range cancelledOrders.Failed {
				if cancelledOrders.Failed[i].ClientOrderID == o.ClientOrderID {
					return fmt.Errorf("failed to cancel client order %s: %w", o.ClientOrderID, htxError(cancelledOrders.Failed[i].ErrorMessage))
				}
			}
			if !slices.Contains(cancelledOrders.Success, o.ClientOrderID) {
				return errEmptyResult
			}
			return nil
		}
		var orderIDInt int64
		orderIDInt, err = strconv.ParseInt(o.OrderID, 10, 64)
		if err != nil {
			return err
		}
		_, err = e.CancelExistingOrder(ctx, orderIDInt)
	case asset.CoinMarginedFutures:
		_, err = e.CancelSwapOrder(ctx, o.OrderID, o.ClientOrderID, o.Pair)
	case asset.USDTMarginedFutures:
		var cancelledOrder *V5OrderResponse
		cancelledOrder, err = e.CancelV5Order(ctx, o.Pair, o.OrderID, o.ClientOrderID)
		if err == nil && cancelledOrder == nil {
			return errEmptyResult
		}
	case asset.Futures:
		_, err = e.FCancelOrder(ctx, o.Pair.Base, o.OrderID, o.ClientOrderID)
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	item := o[0].AssetType
	pair := o[0].Pair
	ids := make([]string, 0, len(o))
	cIDs := make([]string, 0, len(o))
	for i := range o {
		if o[i].AssetType != item {
			return nil, fmt.Errorf("%w: %v and %v", errBatchAssetMismatch, item, o[i].AssetType)
		}
		if item != asset.Spot && !o[i].Pair.Equal(pair) {
			return nil, fmt.Errorf("%w: %s and %s", errBatchPairMismatch, pair, o[i].Pair)
		}
		switch {
		case o[i].ClientOrderID != "":
			cIDs = append(cIDs, o[i].ClientOrderID)
		case o[i].OrderID != "":
			ids = append(ids, o[i].OrderID)
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string)}
	switch item {
	case asset.Spot:
		cancelledOrders, err := e.CancelOrderBatch(ctx, ids, cIDs)
		if err != nil {
			return nil, err
		}
		if cancelledOrders == nil {
			return nil, errEmptyResult
		}
		for i := range cancelledOrders.Success {
			resp.Status[cancelledOrders.Success[i]] = htxStatusSuccess
		}
		for i := range cancelledOrders.Failed {
			resp.Status[cancelledOrders.Failed[i].OrderID] = cancelledOrders.Failed[i].ErrorMessage
		}
	case asset.CoinMarginedFutures:
		if pair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		cancelledOrders, err := e.CancelSwapOrder(ctx, strings.Join(ids, ","), strings.Join(cIDs, ","), pair)
		if err != nil {
			return nil, err
		}
		for id := range strings.SplitSeq(cancelledOrders.Data.Successes, ",") {
			if id != "" {
				resp.Status[id] = htxStatusSuccess
			}
		}
		for i := range cancelledOrders.Data.Errors {
			resp.Status[cancelledOrders.Data.Errors[i].OrderID] = cancelledOrders.Data.Errors[i].ErrMsg
		}
	case asset.USDTMarginedFutures:
		if len(ids)+len(cIDs) > 10 {
			return nil, errBatchOrderLimitExceeded
		}
		if pair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		formattedPair, err := e.FormatSymbol(pair, item)
		if err != nil {
			return nil, err
		}
		cancelledOrders, err := e.CancelV5BatchOrders(ctx, &V5CancelBatchOrdersRequest{
			ContractCode:   formattedPair,
			OrderIDs:       ids,
			ClientOrderIDs: cIDs,
		})
		if err != nil {
			return nil, err
		}
		if cancelledOrders == nil {
			return nil, errEmptyResult
		}
		for i := range cancelledOrders.Data {
			id := cancelledOrders.Data[i].OrderID
			if id == "" {
				id = cancelledOrders.Data[i].ClientOrderID
			}
			if cancelledOrders.Data[i].Code == 0 || cancelledOrders.Data[i].Code == 200 {
				resp.Status[id] = htxStatusSuccess
			} else {
				resp.Status[id] = cancelledOrders.Data[i].Message
			}
		}
	case asset.Futures:
		if pair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		cancelledOrders, err := e.FCancelOrder(ctx, pair.Base, strings.Join(ids, ","), strings.Join(cIDs, ","))
		if err != nil {
			return nil, err
		}
		for id := range strings.SplitSeq(cancelledOrders.Data.Successes, ",") {
			if id != "" {
				resp.Status[id] = htxStatusSuccess
			}
		}
		for i := range cancelledOrders.Data.Errors {
			resp.Status[strconv.FormatInt(cancelledOrders.Data.Errors[i].OrderID, 10)] = cancelledOrders.Data.Errors[i].ErrMsg
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch orderCancellation.AssetType {
	case asset.Spot:
		pairs := currency.Pairs{orderCancellation.Pair}
		if orderCancellation.Pair.IsEmpty() {
			var err error
			pairs, err = e.GetEnabledPairs(asset.Spot)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}
		for i := range pairs {
			resp, err := e.CancelOpenOrdersBatch(ctx,
				orderCancellation.AccountID,
				pairs[i])
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			if resp.Data.FailedCount > 0 {
				return cancelAllOrdersResponse,
					fmt.Errorf("%v orders failed to cancel",
						resp.Data.FailedCount)
			}
			if resp.Status == "error" {
				return cancelAllOrdersResponse, htxError(resp.ErrorMessage)
			}
		}
	case asset.CoinMarginedFutures:
		if orderCancellation.Pair.IsEmpty() {
			enabledPairs, err := e.GetEnabledPairs(asset.CoinMarginedFutures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				a, err := e.CancelAllSwapOrders(ctx, enabledPairs[i])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				split := strings.Split(a.Data.Successes, ",")
				for x := range split {
					if split[x] != "" {
						cancelAllOrdersResponse.Status[split[x]] = htxStatusSuccess
					}
				}
				for y := range a.Data.Errors {
					cancelAllOrdersResponse.Status[a.Data.Errors[y].OrderID] = "fail: " + a.Data.Errors[y].ErrMsg
				}
			}
		} else {
			a, err := e.CancelAllSwapOrders(ctx, orderCancellation.Pair)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			split := strings.Split(a.Data.Successes, ",")
			for x := range split {
				if split[x] != "" {
					cancelAllOrdersResponse.Status[split[x]] = htxStatusSuccess
				}
			}
			for y := range a.Data.Errors {
				cancelAllOrdersResponse.Status[a.Data.Errors[y].OrderID] = "fail: " + a.Data.Errors[y].ErrMsg
			}
		}
	case asset.USDTMarginedFutures:
		if orderCancellation.Pair.IsEmpty() {
			enabledPairs, err := e.GetEnabledPairs(asset.USDTMarginedFutures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				a, err := e.CancelAllV5Orders(ctx, enabledPairs[i], "", "")
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				if a == nil {
					return cancelAllOrdersResponse, errEmptyResult
				}
				for j := range a.Data {
					id := a.Data[j].OrderID
					if id == "" {
						id = a.Data[j].ClientOrderID
					}
					if id == "" {
						continue
					}
					if a.Data[j].Code == 0 || a.Data[j].Code == 200 {
						cancelAllOrdersResponse.Status[id] = htxStatusSuccess
					} else {
						cancelAllOrdersResponse.Status[id] = a.Data[j].Message
					}
				}
			}
		} else {
			a, err := e.CancelAllV5Orders(ctx, orderCancellation.Pair, "", "")
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			if a == nil {
				return cancelAllOrdersResponse, errEmptyResult
			}
			for j := range a.Data {
				id := a.Data[j].OrderID
				if id == "" {
					id = a.Data[j].ClientOrderID
				}
				if id == "" {
					continue
				}
				if a.Data[j].Code == 0 || a.Data[j].Code == 200 {
					cancelAllOrdersResponse.Status[id] = htxStatusSuccess
				} else {
					cancelAllOrdersResponse.Status[id] = a.Data[j].Message
				}
			}
		}
	case asset.Futures:
		if orderCancellation.Pair.IsEmpty() {
			enabledPairs, err := e.GetEnabledPairs(asset.Futures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				a, err := e.FCancelAllOrders(ctx, enabledPairs[i], "", "")
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				split := strings.Split(a.Data.Successes, ",")
				for x := range split {
					if split[x] != "" {
						cancelAllOrdersResponse.Status[split[x]] = htxStatusSuccess
					}
				}
				for y := range a.Data.Errors {
					cancelAllOrdersResponse.Status[strconv.FormatInt(a.Data.Errors[y].OrderID, 10)] = "fail: " + a.Data.Errors[y].ErrMsg
				}
			}
		} else {
			a, err := e.FCancelAllOrders(ctx, orderCancellation.Pair, "", "")
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			split := strings.Split(a.Data.Successes, ",")
			for x := range split {
				if split[x] != "" {
					cancelAllOrdersResponse.Status[split[x]] = htxStatusSuccess
				}
			}
			for y := range a.Data.Errors {
				cancelAllOrdersResponse.Status[strconv.FormatInt(a.Data.Errors[y].OrderID, 10)] = "fail: " + a.Data.Errors[y].ErrMsg
			}
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("%w %v", asset.ErrNotSupported, orderCancellation.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// formatV5OrderDetail converts V5 REST and websocket responses into the canonical order type.
func (e *Exchange) formatV5OrderDetail(data *V5OrderData, item asset.Item) (order.Detail, error) {
	if data == nil {
		return order.Detail{}, common.ErrNilPointer
	}
	side, err := order.StringToOrderSide(data.Side)
	if err != nil {
		return order.Detail{}, err
	}
	orderType := order.Limit
	timeInForce := order.UnknownTIF
	if data.Type == orderPriceTypePostOnly {
		timeInForce = order.PostOnly
	} else {
		orderType, err = order.StringToOrderType(data.Type)
		if err != nil {
			return order.Detail{}, err
		}
	}
	status, err := order.StringToOrderStatus(string(data.State))
	if err != nil {
		return order.Detail{}, err
	}
	if timeInForce == order.UnknownTIF {
		timeInForce, err = order.StringToTimeInForce(data.TimeInForce)
		if err != nil {
			return order.Detail{}, err
		}
	}
	marginType, err := margin.StringToMarginType(data.MarginMode)
	if err != nil {
		return order.Detail{}, err
	}
	pair, err := currency.NewPairFromString(data.ContractCode)
	if err != nil {
		return order.Detail{}, err
	}
	detail := order.Detail{
		Exchange:             e.Name,
		OrderID:              data.OrderID,
		ClientOrderID:        data.ClientOrderID,
		Pair:                 pair,
		Type:                 orderType,
		Side:                 side,
		TimeInForce:          timeInForce,
		Date:                 data.CreatedTime.Time(),
		LastUpdated:          data.UpdatedTime.Time(),
		Status:               status,
		Price:                data.Price.Float64(),
		Amount:               data.Volume.Float64(),
		ExecutedAmount:       data.TradeVolume.Float64(),
		RemainingAmount:      data.Volume.Float64() - data.TradeVolume.Float64(),
		Cost:                 data.TradeTurnover.Float64(),
		AverageExecutedPrice: data.TradeAveragePrice.Float64(),
		Fee:                  data.Fee.Float64(),
		FeeAsset:             currency.NewCode(data.FeeCurrency),
		Leverage:             data.LeverageRate.Float64(),
		ReduceOnly:           data.ReduceOnly.Bool(),
		AssetType:            item,
		MarginType:           marginType,
	}
	detail.InferCostsAndTimes()
	return detail, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	var orderDetail order.Detail
	switch assetType {
	case asset.Spot:
		oID, err := strconv.ParseInt(orderID, 10, 64)
		if err != nil {
			return nil, err
		}
		resp, err := e.GetOrder(ctx, oID)
		if err != nil {
			return nil, err
		}
		respData := &resp
		if respData.ID == 0 {
			return nil, fmt.Errorf("%s - order not found for orderid %s", e.Name, orderID)
		}
		responseID := strconv.FormatInt(respData.ID, 10)
		if responseID != orderID {
			return nil, fmt.Errorf("%s - GetOrderInfo order ID mismatch; expected %s, received %s", e.Name, orderID, responseID)
		}
		if !strings.Contains(respData.Type, "-") {
			return nil, fmt.Errorf("%w %q", errInvalidOrderPriceType, respData.Type)
		}
		orderSide, err := stringToOrderSide(respData.Type)
		if err != nil {
			return nil, err
		}
		orderType, err := stringToOrderType(respData.Type)
		if err != nil {
			return nil, err
		}
		timeInForce := order.UnknownTIF
		switch {
		case strings.Contains(respData.Type, "maker"):
			timeInForce = order.PostOnly
		case strings.HasSuffix(respData.Type, "-fok"):
			timeInForce = order.FillOrKill
		case strings.HasSuffix(respData.Type, "-ioc"):
			timeInForce = order.ImmediateOrCancel
		}
		orderStatus, err := order.StringToOrderStatus(respData.State)
		if err != nil {
			return nil, err
		}
		var p currency.Pair
		var a asset.Item
		p, a, err = e.GetRequestFormattedPairAndAssetType(respData.Symbol)
		if err != nil {
			return nil, err
		}
		orderDetail = order.Detail{
			Exchange:       e.Name,
			OrderID:        orderID,
			AccountID:      strconv.FormatInt(respData.AccountID, 10),
			Pair:           p,
			Type:           orderType,
			Side:           orderSide,
			TimeInForce:    timeInForce,
			Date:           respData.CreatedAt.Time(),
			Status:         orderStatus,
			Price:          respData.Price.Float64(),
			Amount:         respData.Amount.Float64(),
			ExecutedAmount: respData.FilledAmount.Float64(),
			Fee:            respData.FilledFees.Float64(),
			AssetType:      a,
		}
	case asset.CoinMarginedFutures:
		orderInfo, err := e.GetSwapOrderInfo(ctx, pair, orderID, "")
		if err != nil {
			return nil, err
		}
		if len(orderInfo.Data) == 0 {
			return nil, errEmptyResult
		}
		data := orderInfo.Data[0]
		orderVars, err := compatibleVars(data.Direction, data.OrderPriceType, data.Status)
		if err != nil {
			return nil, err
		}
		orderDetail = order.Detail{
			Exchange:             e.Name,
			OrderID:              data.OrderIDString,
			Pair:                 pair,
			Type:                 orderVars.OrderType,
			Side:                 orderVars.Side,
			TimeInForce:          orderVars.TimeInForce,
			Date:                 time.UnixMilli(data.CreatedAt),
			Status:               orderVars.Status,
			Price:                data.Price,
			Amount:               data.Volume,
			ExecutedAmount:       data.TradeVolume,
			RemainingAmount:      data.Volume - data.TradeVolume,
			Cost:                 data.TradeTurnover,
			AverageExecutedPrice: data.TradeAvgPrice,
			Fee:                  data.Fee,
			FeeAsset:             currency.NewCode(data.FeeAsset),
			Leverage:             float64(data.LeverRate),
			ReduceOnly:           data.Offset == orderOffsetClose,
			AssetType:            assetType,
			MarginType:           margin.Isolated,
		}
		if orderDetail.OrderID == "" {
			orderDetail.OrderID = strconv.FormatInt(data.OrderID, 10)
		}
		if orderDetail.OrderID != orderID {
			return nil, fmt.Errorf("GetOrderInfo order ID mismatch: expected %s, received %s", orderID, orderDetail.OrderID)
		}
		if data.ClientOrderID != 0 {
			orderDetail.ClientOrderID = strconv.FormatInt(data.ClientOrderID, 10)
		}
		if data.CancelledAt != 0 {
			orderDetail.CloseTime = time.UnixMilli(data.CancelledAt)
		}
		orderDetail.InferCostsAndTimes()
	case asset.USDTMarginedFutures:
		var orderInfo *V5OrderQueryResponse
		var lookupErr error
		for _, marginMode := range []string{"cross", "isolated"} {
			var err error
			orderInfo, err = e.GetV5Order(ctx, pair, marginMode, orderID, "")
			if err != nil {
				lookupErr = common.AppendError(lookupErr, fmt.Errorf("%s margin order lookup: %w", marginMode, err))
				continue
			}
			if orderInfo != nil && orderInfo.Data.OrderID != "" {
				break
			}
		}
		if orderInfo == nil || orderInfo.Data.OrderID == "" {
			if lookupErr != nil {
				return nil, lookupErr
			}
			return nil, errEmptyResult
		}
		var err error
		orderDetail, err = e.formatV5OrderDetail(&orderInfo.Data, assetType)
		if err != nil {
			return nil, err
		}
		if orderDetail.OrderID != orderID {
			return nil, fmt.Errorf("GetOrderInfo order ID mismatch: expected %s, received %s", orderID, orderDetail.OrderID)
		}
	case asset.Futures:
		fPair, err := e.FormatSymbol(pair, asset.Futures)
		if err != nil {
			return nil, err
		}
		orderInfo, err := e.FGetOrderInfo(ctx, fPair, "", orderID)
		if err != nil {
			return nil, err
		}
		if len(orderInfo.Data) == 0 {
			return nil, errEmptyResult
		}
		data := orderInfo.Data[0]
		orderVars, err := compatibleVars(data.Direction, data.OrderPriceType, data.Status)
		if err != nil {
			return nil, err
		}
		orderDetail = order.Detail{
			Exchange:             e.Name,
			OrderID:              data.OrderIDString,
			Pair:                 pair,
			Type:                 orderVars.OrderType,
			Side:                 orderVars.Side,
			TimeInForce:          orderVars.TimeInForce,
			Date:                 time.UnixMilli(data.CreatedAt),
			Status:               orderVars.Status,
			Price:                data.Price,
			Amount:               data.Volume,
			ExecutedAmount:       data.TradeVolume,
			RemainingAmount:      data.Volume - data.TradeVolume,
			Cost:                 data.TradeTurnover,
			AverageExecutedPrice: data.TradeAvgPrice,
			Fee:                  data.Fee,
			FeeAsset:             currency.NewCode(data.FeeAsset),
			Leverage:             float64(data.LeverRate),
			ReduceOnly:           data.Offset == orderOffsetClose,
			AssetType:            assetType,
			MarginType:           margin.Isolated,
		}
		if orderDetail.OrderID == "" {
			orderDetail.OrderID = strconv.FormatInt(data.OrderID, 10)
		}
		if orderDetail.OrderID != orderID {
			return nil, fmt.Errorf("GetOrderInfo order ID mismatch: expected %s, received %s", orderID, orderDetail.OrderID)
		}
		if data.ClientOrderID != 0 {
			orderDetail.ClientOrderID = strconv.FormatInt(data.ClientOrderID, 10)
		}
		if data.CanceledAt != 0 {
			orderDetail.CloseTime = time.UnixMilli(data.CanceledAt)
		}
		orderDetail.InferCostsAndTimes()
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return &orderDetail, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	resp, err := e.QueryDepositAddress(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}

	for x := range resp {
		if chain != "" && strings.EqualFold(resp[x].Chain, chain) {
			return &deposit.Address{
				Address: resp[x].Address,
				Tag:     resp[x].AddressTag,
			}, nil
		} else if chain == "" && strings.EqualFold(resp[x].Currency, cryptocurrency.String()) {
			return &deposit.Address{
				Address: resp[x].Address,
				Tag:     resp[x].AddressTag,
			}, nil
		}
	}
	return nil, errDepositAddressNotFound
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.Withdraw(ctx,
		withdrawRequest.Currency,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Crypto.Chain,
		withdrawRequest.Amount,
		withdrawRequest.Crypto.FeeAmount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(resp, 10),
	}, err
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
	if !e.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		if len(req.Pairs) == 0 {
			return nil, errCurrencyNotSupplied
		}
		side := ""
		if req.Side == order.Buy || req.Side == order.Sell {
			side = req.Side.Lower()
		}
		creds, err := e.GetCredentials(ctx)
		if err != nil {
			return nil, err
		}
		for i := range req.Pairs {
			resp, err := e.GetOpenOrders(ctx,
				req.Pairs[i],
				creds.ClientID,
				side,
				500)
			if err != nil {
				return nil, err
			}
			for x := range resp {
				orderDetail := order.Detail{
					OrderID:         strconv.FormatInt(resp[x].ID, 10),
					Price:           resp[x].Price.Float64(),
					Amount:          resp[x].Amount.Float64(),
					ExecutedAmount:  resp[x].FilledAmount.Float64(),
					RemainingAmount: resp[x].Amount.Float64() - resp[x].FilledAmount.Float64(),
					Pair:            req.Pairs[i],
					Exchange:        e.Name,
					Date:            resp[x].CreatedAt.Time(),
					AccountID:       strconv.FormatInt(resp[x].AccountID, 10),
					Fee:             resp[x].FilledFees.Float64(),
				}
				setOrderSideStatusAndType(resp[x].State, resp[x].Type, &orderDetail)
				orders = append(orders, orderDetail)
			}
		}
	case asset.CoinMarginedFutures:
		for x := range req.Pairs {
			for currentPage := int64(1); ; currentPage++ {
				openOrders, err := e.GetSwapOpenOrders(ctx,
					req.Pairs[x], currentPage, 50)
				if err != nil {
					return orders, err
				}

				for x := range openOrders.Data.Orders {
					orderVars, err := compatibleVars(openOrders.Data.Orders[x].Direction,
						openOrders.Data.Orders[x].OrderPriceType,
						openOrders.Data.Orders[x].Status)
					if err != nil {
						return orders, err
					}
					p, err := currency.NewPairFromString(openOrders.Data.Orders[x].ContractCode)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						TimeInForce:     orderVars.TimeInForce,
						Leverage:        openOrders.Data.Orders[x].LeverageRate,
						Price:           openOrders.Data.Orders[x].Price,
						Amount:          openOrders.Data.Orders[x].Volume,
						ExecutedAmount:  openOrders.Data.Orders[x].TradeVolume,
						RemainingAmount: openOrders.Data.Orders[x].Volume - openOrders.Data.Orders[x].TradeVolume,
						Fee:             openOrders.Data.Orders[x].Fee,
						Exchange:        e.Name,
						AssetType:       req.AssetType,
						OrderID:         openOrders.Data.Orders[x].OrderIDString,
						Side:            orderVars.Side,
						Type:            orderVars.OrderType,
						Status:          orderVars.Status,
						Pair:            p,
					})
				}
				if openOrders.Data.TotalPage == 0 || currentPage >= openOrders.Data.TotalPage {
					break
				}
			}
		}
	case asset.USDTMarginedFutures:
		marginModes := []string{"cross", "isolated"}
		switch req.MarginType {
		case margin.Unset:
		case margin.Multi:
			marginModes = marginModes[:1]
		case margin.Isolated:
			marginModes = marginModes[1:]
		default:
			return nil, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, req.MarginType)
		}
		for x := range req.Pairs {
			for _, marginMode := range marginModes {
				var from uint64
				for {
					openOrders, err := e.GetV5OpenOrders(ctx, req.Pairs[x], marginMode, "", "", from, 100, "next")
					if err != nil {
						return orders, err
					}
					if openOrders == nil {
						return orders, errEmptyResult
					}
					for y := range openOrders.Data {
						detail, err := e.formatV5OrderDetail(&openOrders.Data[y], req.AssetType)
						if err != nil {
							return orders, err
						}
						orders = append(orders, detail)
					}
					if len(openOrders.Data) < 100 {
						break
					}
					cursor := openOrders.Data[len(openOrders.Data)-1].ID
					if cursor == "" {
						cursor = openOrders.Data[len(openOrders.Data)-1].OrderID
					}
					next, err := strconv.ParseUint(cursor, 10, 64)
					if err != nil {
						return orders, fmt.Errorf("invalid HTX order cursor %q: %w", cursor, err)
					}
					if next == from {
						break
					}
					from = next
				}
			}
		}
	case asset.Futures:
		for x := range req.Pairs {
			for currentPage := int64(1); ; currentPage++ {
				openOrders, err := e.FGetOpenOrders(ctx,
					req.Pairs[x].Base, currentPage, 50)
				if err != nil {
					return orders, err
				}
				var orderVars OrderVars
				for x := range openOrders.Data.Orders {
					orderVars, err = compatibleVars(openOrders.Data.Orders[x].Direction,
						openOrders.Data.Orders[x].OrderPriceType,
						openOrders.Data.Orders[x].Status)
					if err != nil {
						return orders, err
					}
					p, err := currency.NewPairFromString(openOrders.Data.Orders[x].ContractCode)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						TimeInForce:     orderVars.TimeInForce,
						Leverage:        openOrders.Data.Orders[x].LeverageRate,
						Price:           openOrders.Data.Orders[x].Price,
						Amount:          openOrders.Data.Orders[x].Volume,
						ExecutedAmount:  openOrders.Data.Orders[x].TradeVolume,
						RemainingAmount: openOrders.Data.Orders[x].Volume - openOrders.Data.Orders[x].TradeVolume,
						Fee:             openOrders.Data.Orders[x].Fee,
						Exchange:        e.Name,
						AssetType:       req.AssetType,
						OrderID:         openOrders.Data.Orders[x].OrderIDString,
						Side:            orderVars.Side,
						Type:            orderVars.OrderType,
						Status:          orderVars.Status,
						Pair:            p,
					})
				}
				if openOrders.Data.TotalPage == 0 || currentPage >= openOrders.Data.TotalPage {
					break
				}
			}
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(e.Name, orders), nil
}

type v3HistoryWindow struct {
	start time.Time
	end   time.Time
}

func getV3HistoryWindows(startTime, endTime time.Time) ([]v3HistoryWindow, error) {
	if startTime.IsZero() && endTime.IsZero() {
		return []v3HistoryWindow{{}}, nil
	}
	if startTime.IsZero() || endTime.IsZero() {
		return nil, errInvalidCreateDate
	}
	if startTime.After(endTime) {
		return nil, errStartTimeAfterEndTime
	}
	if endTime.Sub(startTime) > 90*24*time.Hour {
		return nil, errInvalidCreateDate
	}
	windows := make([]v3HistoryWindow, 0, int(endTime.Sub(startTime)/(48*time.Hour))+1)
	for !startTime.After(endTime) {
		windowEnd := startTime.Add(48 * time.Hour)
		if windowEnd.After(endTime) {
			windowEnd = endTime
		}
		windows = append(windows, v3HistoryWindow{start: startTime, end: windowEnd})
		startTime = windowEnd.Add(time.Millisecond)
	}
	return windows, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		if len(req.Pairs) == 0 {
			return nil, errCurrencyNotSupplied
		}
		states := "partial-canceled,filled,canceled"
		for i := range req.Pairs {
			resp, err := e.GetOrders(ctx,
				req.Pairs[i],
				"",
				"",
				"",
				states,
				"",
				"",
				"")
			if err != nil {
				return nil, err
			}
			for x := range resp {
				orderDetail := order.Detail{
					OrderID:         strconv.FormatInt(resp[x].ID, 10),
					Price:           resp[x].Price.Float64(),
					Amount:          resp[x].Amount.Float64(),
					ExecutedAmount:  resp[x].FilledAmount.Float64(),
					RemainingAmount: resp[x].Amount.Float64() - resp[x].FilledAmount.Float64(),
					Cost:            resp[x].FilledCashAmount.Float64(),
					CostAsset:       req.Pairs[i].Quote,
					Pair:            req.Pairs[i],
					Exchange:        e.Name,
					Date:            resp[x].CreatedAt.Time(),
					CloseTime:       resp[x].FinishedAt.Time(),
					AccountID:       strconv.FormatInt(resp[x].AccountID, 10),
					Fee:             resp[x].FilledFees.Float64(),
				}
				setOrderSideStatusAndType(resp[x].State, resp[x].Type, &orderDetail)
				orderDetail.InferCostsAndTimes()
				orders = append(orders, orderDetail)
			}
		}
	case asset.CoinMarginedFutures:
		windows, err := getV3HistoryWindows(req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
		for x := range req.Pairs {
			seen := make(map[int64]struct{})
			for _, window := range windows {
				var cursor int64
				for {
					orderHistory, err := e.GetSwapOrderHistoryByTimeRange(ctx,
						req.Pairs[x],
						"all",
						"all",
						[]order.Status{order.AnyStatus},
						window.start,
						window.end,
						cursor,
						50)
					if err != nil {
						return orders, err
					}
					var orderVars OrderVars
					for x := range orderHistory.Data.Orders {
						if orderHistory.Data.Orders[x].QueryID != 0 {
							if _, ok := seen[orderHistory.Data.Orders[x].QueryID]; ok {
								continue
							}
							seen[orderHistory.Data.Orders[x].QueryID] = struct{}{}
						}
						p, err := currency.NewPairFromString(orderHistory.Data.Orders[x].ContractCode)
						if err != nil {
							return orders, err
						}
						orderVars, err = compatibleVars(orderHistory.Data.Orders[x].Direction,
							orderHistory.Data.Orders[x].OrderPriceType,
							orderHistory.Data.Orders[x].Status)
						if err != nil {
							return orders, err
						}
						orders = append(orders, order.Detail{
							TimeInForce:     orderVars.TimeInForce,
							Leverage:        orderHistory.Data.Orders[x].LeverageRate,
							Price:           orderHistory.Data.Orders[x].Price,
							Amount:          orderHistory.Data.Orders[x].Volume,
							ExecutedAmount:  orderHistory.Data.Orders[x].TradeVolume,
							RemainingAmount: orderHistory.Data.Orders[x].Volume - orderHistory.Data.Orders[x].TradeVolume,
							Fee:             orderHistory.Data.Orders[x].Fee,
							Exchange:        e.Name,
							AssetType:       req.AssetType,
							OrderID:         orderHistory.Data.Orders[x].OrderIDString,
							Side:            orderVars.Side,
							Type:            orderVars.OrderType,
							Status:          orderVars.Status,
							Pair:            p,
						})
					}
					if orderHistory.Data.TotalPage > 0 {
						cursor++
						if cursor >= orderHistory.Data.TotalPage {
							break
						}
						continue
					}
					if len(orderHistory.Data.Orders) < 50 {
						break
					}
					nextCursor := cursor
					for x := range orderHistory.Data.Orders {
						nextCursor = max(nextCursor, orderHistory.Data.Orders[x].QueryID)
					}
					if nextCursor == cursor {
						break
					}
					cursor = nextCursor
				}
			}
		}
	case asset.USDTMarginedFutures:
		windows, err := getV3HistoryWindows(req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
		marginModes := []string{"cross", "isolated"}
		switch req.MarginType {
		case margin.Unset:
		case margin.Multi:
			marginModes = marginModes[:1]
		case margin.Isolated:
			marginModes = marginModes[1:]
		default:
			return nil, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, req.MarginType)
		}
		for x := range req.Pairs {
			contractCode, err := e.FormatSymbol(req.Pairs[x], req.AssetType)
			if err != nil {
				return nil, err
			}
			for _, marginMode := range marginModes {
				seen := make(map[string]struct{})
				for _, window := range windows {
					var from uint64
					for {
						orderHistory, err := e.GetV5OrderHistory(ctx, &V5OrderHistoryRequest{
							ContractCode: contractCode,
							MarginMode:   marginMode,
							StartTime:    window.start,
							EndTime:      window.end,
							From:         from,
							Limit:        100,
							Direction:    "next",
						})
						if err != nil {
							return orders, err
						}
						if orderHistory == nil {
							return orders, errEmptyResult
						}
						for i := range orderHistory.Data {
							orderKey := orderHistory.Data[i].OrderID
							if orderKey == "" {
								orderKey = orderHistory.Data[i].ID
							}
							if orderKey != "" {
								if _, found := seen[orderKey]; found {
									continue
								}
								seen[orderKey] = struct{}{}
							}
							detail, err := e.formatV5OrderDetail(&orderHistory.Data[i], req.AssetType)
							if err != nil {
								return orders, err
							}
							orders = append(orders, detail)
						}
						if len(orderHistory.Data) < 100 {
							break
						}
						cursor := orderHistory.Data[len(orderHistory.Data)-1].ID
						if cursor == "" {
							cursor = orderHistory.Data[len(orderHistory.Data)-1].OrderID
						}
						next, err := strconv.ParseUint(cursor, 10, 64)
						if err != nil {
							return orders, fmt.Errorf("invalid HTX order cursor %q: %w", cursor, err)
						}
						if next == from {
							break
						}
						from = next
					}
				}
			}
		}
	case asset.Futures:
		windows, err := getV3HistoryWindows(req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
		for x := range req.Pairs {
			seen := make(map[int64]struct{})
			for _, window := range windows {
				var cursor int64
				for {
					openOrders, err := e.FGetOrderHistoryByTimeRange(ctx,
						req.Pairs[x],
						"",
						"all",
						"all",
						"limit",
						[]order.Status{order.AnyStatus},
						window.start,
						window.end,
						cursor,
						50)
					if err != nil {
						return orders, err
					}
					var orderVars OrderVars
					for x := range openOrders.Data.Orders {
						if openOrders.Data.Orders[x].QueryID != 0 {
							if _, ok := seen[openOrders.Data.Orders[x].QueryID]; ok {
								continue
							}
							seen[openOrders.Data.Orders[x].QueryID] = struct{}{}
						}
						orderVars, err = compatibleVars(openOrders.Data.Orders[x].Direction,
							openOrders.Data.Orders[x].OrderPriceType,
							openOrders.Data.Orders[x].Status)
						if err != nil {
							return orders, err
						}
						if req.Side != order.AnySide && req.Side != orderVars.Side {
							continue
						}
						if req.Type != order.AnyType && req.Type != orderVars.OrderType {
							continue
						}
						p, err := currency.NewPairFromString(openOrders.Data.Orders[x].ContractCode)
						if err != nil {
							return orders, err
						}
						orders = append(orders, order.Detail{
							TimeInForce:     orderVars.TimeInForce,
							Leverage:        openOrders.Data.Orders[x].LeverageRate,
							Price:           openOrders.Data.Orders[x].Price,
							Amount:          openOrders.Data.Orders[x].Volume,
							ExecutedAmount:  openOrders.Data.Orders[x].TradeVolume,
							RemainingAmount: openOrders.Data.Orders[x].Volume - openOrders.Data.Orders[x].TradeVolume,
							Fee:             openOrders.Data.Orders[x].Fee,
							Exchange:        e.Name,
							AssetType:       req.AssetType,
							OrderID:         openOrders.Data.Orders[x].OrderIDString,
							Side:            orderVars.Side,
							Type:            orderVars.OrderType,
							Status:          orderVars.Status,
							Pair:            p,
							Date:            openOrders.Data.Orders[x].CreateDate.Time(),
						})
					}
					if openOrders.Data.TotalPage > 0 {
						cursor++
						if cursor >= openOrders.Data.TotalPage {
							break
						}
						continue
					}
					if len(openOrders.Data.Orders) < 50 {
						break
					}
					nextCursor := cursor
					for x := range openOrders.Data.Orders {
						nextCursor = max(nextCursor, openOrders.Data.Orders[x].QueryID)
					}
					if nextCursor == cursor {
						break
					}
					cursor = nextCursor
				}
			}
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(e.Name, orders), nil
}

func setOrderSideStatusAndType(orderState, requestType string, orderDetail *order.Detail) {
	var err error
	if orderDetail.Status, err = order.StringToOrderStatus(orderState); err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", orderDetail.Exchange, err)
	}

	switch SpotNewOrderRequestParamsType(requestType) {
	case SpotNewOrderRequestTypeBuyMarket:
		orderDetail.Side = order.Buy
		orderDetail.Type = order.Market
	case SpotNewOrderRequestTypeSellMarket:
		orderDetail.Side = order.Sell
		orderDetail.Type = order.Market
	case SpotNewOrderRequestTypeBuyLimit:
		orderDetail.Side = order.Buy
		orderDetail.Type = order.Limit
	case SpotNewOrderRequestTypeSellLimit:
		orderDetail.Side = order.Sell
		orderDetail.Type = order.Limit
	}
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (e *Exchange) AuthenticateWebsocket(ctx context.Context) error {
	return e.wsLogin(ctx, e.Websocket.AuthConn)
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (e *Exchange) FormatExchangeKlineInterval(in kline.Interval) string {
	switch in {
	case kline.OneMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin:
		return in.Short() + "in"
	case kline.OneHour:
		return "60min"
	case kline.FourHour:
		return "4hour"
	case kline.OneDay:
		return "1day"
	case kline.OneMonth:
		return "1mon"
	case kline.OneWeek:
		return "1week"
	case kline.OneYear:
		return "1year"
	}
	return ""
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	req, err := e.GetKlineRequest(pair, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	switch a {
	case asset.Spot:
		candles, err := e.GetSpotKline(ctx, KlinesRequestParams{
			Period: e.FormatExchangeKlineInterval(req.ExchangeInterval),
			Symbol: req.Pair,
			Size:   req.RequestLimit,
		})
		if err != nil {
			return nil, err
		}

		for x := range candles {
			timestamp := candles[x].IDTimestamp.Time()
			if timestamp.Before(req.Start) || timestamp.After(req.End) {
				continue
			}
			timeSeries = append(timeSeries, kline.Candle{
				Time:   timestamp,
				Open:   candles[x].Open,
				High:   candles[x].High,
				Low:    candles[x].Low,
				Close:  candles[x].Close,
				Volume: candles[x].Volume,
			})
		}
	case asset.Futures:
		// if size, from, to are all populated, only size is considered
		size := int64(-1)
		candles, err := e.FGetKlineData(ctx, req.Pair, e.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.Start, req.End)
		if err != nil {
			return nil, err
		}
		timeSeries = appendFuturesCandles(timeSeries, candles.Data, req.Start, req.End)
	case asset.CoinMarginedFutures:
		// if size, from, to are all populated, only size is considered
		size := int64(-1)
		candles, err := e.GetSwapKlineData(ctx, req.Pair, e.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.Start, req.End)
		if err != nil {
			return nil, err
		}
		timeSeries = appendFuturesCandles(timeSeries, candles.Data, req.Start, req.End)
	case asset.USDTMarginedFutures:
		size := int64(-1)
		candles, err := e.GetLinearSwapKlineData(ctx, req.Pair, e.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.Start, req.End)
		if err != nil {
			return nil, err
		}
		timeSeries = appendFuturesCandles(timeSeries, candles.Data, req.Start, req.End)
	}

	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	switch a {
	case asset.Spot:
		return nil, common.ErrFunctionNotSupported
	case asset.Futures:
		for i := range req.RangeHolder.Ranges {
			// if size, from, to are all populated, only size is considered
			size := int64(-1)
			var candles FKlineData
			candles, err = e.FGetKlineData(ctx, req.Pair, e.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.RangeHolder.Ranges[i].Start.Time, req.RangeHolder.Ranges[i].End.Time)
			if err != nil {
				return nil, err
			}
			timeSeries = appendFuturesCandles(timeSeries, candles.Data, req.Start, req.End)
		}
	case asset.CoinMarginedFutures:
		for i := range req.RangeHolder.Ranges {
			// if size, from, to are all populated, only size is considered
			size := int64(-1)
			var candles SwapKlineData
			candles, err = e.GetSwapKlineData(ctx, req.Pair, e.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.RangeHolder.Ranges[i].Start.Time, req.RangeHolder.Ranges[i].End.Time)
			if err != nil {
				return nil, err
			}
			timeSeries = appendFuturesCandles(timeSeries, candles.Data, req.Start, req.End)
		}
	case asset.USDTMarginedFutures:
		for i := range req.RangeHolder.Ranges {
			size := int64(-1)
			var candles SwapKlineData
			candles, err = e.GetLinearSwapKlineData(ctx, req.Pair, e.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.RangeHolder.Ranges[i].Start.Time, req.RangeHolder.Ranges[i].End.Time)
			if err != nil {
				return nil, err
			}
			timeSeries = appendFuturesCandles(timeSeries, candles.Data, req.Start, req.End)
		}
	}

	return req.ProcessResponse(timeSeries)
}

// compatibleVars gets compatible variables for order vars
func compatibleVars(side, orderPriceType string, status int64) (OrderVars, error) {
	var resp OrderVars
	switch side {
	case "buy":
		resp.Side = order.Buy
	case "sell":
		resp.Side = order.Sell
	default:
		return resp, errUnrecognisedOrderSide
	}
	switch orderPriceType {
	case "limit":
		resp.OrderType = order.Limit
	case "opponent":
		resp.OrderType = order.Market
	case orderPriceTypePostOnly:
		resp.OrderType = order.Limit
		resp.TimeInForce = order.PostOnly
	default:
		return resp, errInvalidOrderPriceType
	}
	switch status {
	case 1, 2, 11:
		resp.Status = order.UnknownStatus
	case 3:
		resp.Status = order.Active
	case 4:
		resp.Status = order.PartiallyFilled
	case 5:
		resp.Status = order.PartiallyCancelled
	case 6:
		resp.Status = order.Filled
	case 7:
		resp.Status = order.Cancelled
	default:
		return resp, errInvalidOrderStatus
	}
	return resp, nil
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	resp, err := e.GetCurrenciesIncludingChains(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, errNoTransferChains
	}

	chains := resp[0].ChainData

	availableChains := make([]string, 0, len(chains))
	for _, c := range chains {
		if c.DepositStatus == "allowed" || c.WithdrawStatus == "allowed" {
			availableChains = append(availableChains, c.Chain)
		}
	}
	return availableChains, nil
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return e.GetCurrentServerTime(ctx)
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !e.SupportsAsset(item) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}

	switch item {
	case asset.CoinMarginedFutures:
		result, err := e.GetSwapMarkets(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, 0, len(result))
		for x := range result {
			cp, err := currency.NewPairFromString(result[x].ContractCode)
			if err != nil {
				return nil, err
			}
			underlying, err := currency.NewPairFromStrings(result[x].Symbol, "USD")
			if err != nil {
				return nil, err
			}

			resp = append(resp, futures.Contract{
				Exchange:           e.Name,
				Name:               cp,
				Underlying:         underlying,
				Asset:              item,
				StartDate:          result[x].CreateDate.Time(),
				SettlementType:     futures.Inverse,
				IsActive:           result[x].ContractStatus == 1,
				Type:               futures.Perpetual,
				SettlementCurrency: cp.Base,
				Multiplier:         result[x].ContractSize,
			})
		}
		return resp, nil
	case asset.USDTMarginedFutures:
		result, err := e.GetLinearSwapMarkets(ctx, currency.EMPTYPAIR, "", "swap", "swap")
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, 0, len(result))
		for x := range result {
			cp, err := currency.NewPairFromString(result[x].ContractCode)
			if err != nil {
				return nil, err
			}
			underlying, err := currency.NewPairFromString(result[x].Pair)
			if err != nil {
				return nil, err
			}
			resp = append(resp, futures.Contract{
				Exchange:           e.Name,
				Name:               cp,
				Underlying:         underlying,
				Asset:              item,
				StartDate:          result[x].CreateDate.Time(),
				SettlementType:     futures.Linear,
				IsActive:           result[x].ContractStatus == 1,
				Type:               futures.Perpetual,
				SettlementCurrency: currency.USDT,
				Multiplier:         result[x].ContractSize.Float64(),
			})
		}
		return resp, nil
	case asset.Futures:
		result, err := e.FGetContractInfo(ctx, "", "", currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, 0, len(result.Data))
		for x := range result.Data {
			expiry, ok := strings.CutPrefix(result.Data[x].ContractCode, result.Data[x].Symbol)
			if !ok || expiry == "" {
				return nil, fmt.Errorf("%w from contract code %q", currency.ErrCreatingPair, result.Data[x].ContractCode)
			}
			cp, err := currency.NewPairFromStrings(result.Data[x].Symbol, expiry)
			if err != nil {
				return nil, err
			}
			underlying, err := currency.NewPairFromStrings(result.Data[x].Symbol, "USD")
			if err != nil {
				return nil, err
			}
			endTime := result.Data[x].DeliveryTime.Time()
			if endTime.IsZero() {
				endTime = result.Data[x].SettlementTime.Time()
			}
			contractLength := endTime.Sub(result.Data[x].CreateDate.Time())
			var ct futures.ContractType
			switch {
			case contractLength <= kline.OneWeek.Duration()+kline.ThreeDay.Duration():
				ct = futures.Weekly
			case contractLength <= kline.TwoWeek.Duration()+kline.ThreeDay.Duration():
				ct = futures.Fortnightly
			case contractLength <= kline.ThreeMonth.Duration()+kline.ThreeWeek.Duration():
				ct = futures.Quarterly
			case contractLength <= kline.SixMonth.Duration()+kline.ThreeWeek.Duration():
				ct = futures.HalfYearly
			default:
				ct = futures.Perpetual
			}

			resp = append(resp, futures.Contract{
				Exchange:           e.Name,
				Name:               cp,
				Underlying:         underlying,
				Asset:              item,
				StartDate:          result.Data[x].CreateDate.Time(),
				EndDate:            endTime,
				SettlementType:     futures.Inverse,
				IsActive:           result.Data[x].ContractStatus == 1,
				Type:               ct,
				SettlementCurrency: cp.Base,
				Multiplier:         result.Data[x].ContractSize,
			})
		}
		return resp, nil
	}
	return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.CoinMarginedFutures && r.Asset != asset.USDTMarginedFutures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	if r.Asset == asset.USDTMarginedFutures && r.IncludePredictedRate {
		return nil, fmt.Errorf("%w IncludePredictedRate", common.ErrFunctionNotSupported)
	}

	var rates []FundingRatesData
	switch r.Asset {
	case asset.CoinMarginedFutures:
		if r.Pair.IsEmpty() {
			batchRates, err := e.GetSwapFundingRates(ctx)
			if err != nil {
				return nil, err
			}
			rates = batchRates.Data
		} else {
			rateResp, err := e.GetSwapFundingRate(ctx, r.Pair)
			if err != nil {
				return nil, err
			}
			rates = append(rates, rateResp)
		}
	case asset.USDTMarginedFutures:
		var pairs currency.Pairs
		if r.Pair.IsEmpty() {
			var err error
			pairs, err = e.GetEnabledPairs(asset.USDTMarginedFutures)
			if err != nil {
				return nil, err
			}
		} else {
			pairs = currency.Pairs{r.Pair}
		}
		for start := 0; start < len(pairs); start += 10 {
			end := min(start+10, len(pairs))
			rateResp, err := e.GetV5FundingRates(ctx, &V5FundingRatesRequest{ContractCodes: pairs[start:end]})
			if err != nil {
				return nil, err
			}
			for i := range rateResp.Data {
				rates = append(rates, FundingRatesData{
					FundingRate:     rateResp.Data[i].FundingRate,
					ContractCode:    rateResp.Data[i].ContractCode,
					FundingTime:     rateResp.Data[i].FundingTime,
					NextFundingTime: rateResp.Data[i].NextFundingTime,
				})
			}
		}
	}
	resp := make([]fundingrate.LatestRateResponse, 0, len(rates))
	for i := range rates {
		if rates[i].ContractCode == "" {
			// formatting to match documentation
			if r.Asset == asset.CoinMarginedFutures {
				rates[i].ContractCode = rates[i].Symbol + "-USD"
			} else {
				rates[i].ContractCode = rates[i].Symbol + "-USDT"
			}
		}
		cp, isEnabled, err := e.MatchSymbolCheckEnabled(rates[i].ContractCode, r.Asset, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !isEnabled {
			continue
		}
		var isPerp bool
		isPerp, err = e.IsPerpetualFutureCurrency(r.Asset, cp)
		if err != nil {
			return nil, err
		}
		if !isPerp {
			continue
		}
		ft, nft := rates[i].FundingTime.Time(), rates[i].NextFundingTime.Time()
		var fri time.Duration
		if len(e.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
			// can infer funding rate interval from the only funding rate frequency defined
			for k := range e.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
				fri = k.Duration()
			}
		}
		if rates[i].FundingTime.Time().IsZero() {
			ft = nft.Add(-fri)
		}
		if ft.After(time.Now()) {
			ft = ft.Add(-fri)
			nft = nft.Add(-fri)
		}
		rate := fundingrate.LatestRateResponse{
			Exchange: e.Name,
			Asset:    r.Asset,
			Pair:     cp,
			LatestRate: fundingrate.Rate{
				Time: ft,
				Rate: decimal.NewFromFloat(rates[i].FundingRate.Float64()),
			},
			TimeOfNextRate: nft,
			TimeChecked:    time.Now(),
		}
		if r.IncludePredictedRate {
			rate.PredictedUpcomingRate = fundingrate.Rate{
				Time: rate.TimeOfNextRate,
				Rate: decimal.NewFromFloat(rates[i].EstimatedRate.Float64()),
			}
		}
		resp = append(resp, rate)
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.CoinMarginedFutures || a == asset.USDTMarginedFutures, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	var l []limits.MinMaxLevel
	switch a {
	case asset.Spot:
		symbols, err := e.GetSymbols(ctx)
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(symbols))
		for i := range symbols {
			if symbols[i].State != "online" {
				continue
			}
			p, err := currency.NewPairFromStrings(symbols[i].BaseCurrency, symbols[i].QuoteCurrency)
			if err != nil {
				return err
			}
			minBaseAmt := symbols[i].LimitOrderMinOrderAmt
			if minBaseAmt == 0 {
				minBaseAmt = symbols[i].MinOrderAmt
			}
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, p),
				MinimumBaseAmount:       minBaseAmt,
				MaximumBaseAmount:       symbols[i].LimitOrderMaxOrderAmt,
				MinimumQuoteAmount:      symbols[i].MinOrderValue,
				AmountStepIncrementSize: math.Pow10(-int(symbols[i].AmountPrecision)),
				PriceStepIncrementSize:  math.Pow10(-int(symbols[i].PricePrecision)),
				QuoteStepIncrementSize:  math.Pow10(-int(symbols[i].ValuePrecision)),
			})
		}
	case asset.Futures:
		contracts, err := e.FGetContractInfo(ctx, "", "", currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(contracts.Data))
		for i := range contracts.Data {
			if contracts.Data[i].ContractStatus != 1 {
				continue
			}
			p, err := e.MatchSymbolWithAvailablePairs(contracts.Data[i].ContractCode, a, true)
			if err != nil {
				return err
			}
			endTime := contracts.Data[i].DeliveryTime.Time()
			if endTime.IsZero() {
				endTime = contracts.Data[i].SettlementTime.Time()
			}
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, p),
				MinimumBaseAmount:       1,
				AmountStepIncrementSize: 1, // orders are in number of contracts
				PriceStepIncrementSize:  contracts.Data[i].PriceTick,
				MultiplierDecimal:       contracts.Data[i].ContractSize,
				Listed:                  contracts.Data[i].CreateDate.Time(),
				Expiry:                  endTime,
			})
		}
	case asset.CoinMarginedFutures:
		contracts, err := e.GetSwapMarkets(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(contracts))
		for i := range contracts {
			if contracts[i].ContractStatus != 1 {
				continue
			}
			p, err := e.MatchSymbolWithAvailablePairs(contracts[i].ContractCode, a, true)
			if err != nil {
				if errors.Is(err, currency.ErrPairNotFound) {
					continue
				}
				return err
			}

			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, p),
				MinimumBaseAmount:       1,
				AmountStepIncrementSize: 1, // orders are in number of contracts
				PriceStepIncrementSize:  contracts[i].PriceTick,
				MultiplierDecimal:       contracts[i].ContractSize,
				Listed:                  contracts[i].CreateDate.Time(),
				Delisted:                contracts[i].DeliveryTime.Time(),
			})
		}
	case asset.USDTMarginedFutures:
		contracts, err := e.GetLinearSwapMarkets(ctx, currency.EMPTYPAIR, "", "swap", "swap")
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(contracts))
		for i := range contracts {
			if contracts[i].ContractStatus != 1 {
				continue
			}
			p, err := e.MatchSymbolWithAvailablePairs(contracts[i].ContractCode, a, true)
			if err != nil {
				if errors.Is(err, currency.ErrPairNotFound) {
					continue
				}
				return err
			}

			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, p),
				MinimumBaseAmount:       1,
				AmountStepIncrementSize: 1,
				PriceStepIncrementSize:  contracts[i].PriceTick.Float64(),
				MultiplierDecimal:       contracts[i].ContractSize.Float64(),
				Listed:                  contracts[i].CreateDate.Time(),
			})
		}
	}
	return limits.Load(l)
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.Futures && k[i].Asset != asset.CoinMarginedFutures && k[i].Asset != asset.USDTMarginedFutures {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	if len(k) == 1 {
		switch k[0].Asset {
		case asset.Futures:
			if !slices.Contains(validContractExpiryCodes, strings.ToUpper(k[0].Pair().Quote.String())) {
				// HTX does not like requests being made with contract expiry in them (eg BTC240109)
				return nil, fmt.Errorf("%w %v, must use shorthand such as CW (current week)", currency.ErrCurrencyNotSupported, k[0].Pair())
			}
			data, err := e.FContractOpenInterest(ctx, "", "", k[0].Pair())
			if err != nil {
				return nil, err
			}

			for i := range data.Data {
				var p currency.Pair
				p, err = e.MatchSymbolWithAvailablePairs(data.Data[i].ContractCode, k[0].Asset, true)
				if err != nil {
					if errors.Is(err, currency.ErrPairNotFound) {
						continue
					}
					return nil, err
				}
				return []futures.OpenInterest{
					{
						Key:          key.NewExchangeAssetPair(e.Name, k[0].Asset, p),
						OpenInterest: data.Data[i].Amount,
					},
				}, nil
			}
		case asset.USDTMarginedFutures:
			data, err := e.GetV5OpenInterest(ctx, k[0].Pair())
			if err != nil {
				return nil, err
			}
			p, err := e.MatchSymbolWithAvailablePairs(data.Data.ContractCode, k[0].Asset, true)
			if err != nil {
				if !errors.Is(err, currency.ErrPairNotFound) {
					return nil, err
				}
				break
			}
			return []futures.OpenInterest{
				{
					Key:          key.NewExchangeAssetPair(e.Name, k[0].Asset, p),
					OpenInterest: data.Data.Amount.Float64(),
				},
			}, nil
		case asset.CoinMarginedFutures:
			data, err := e.SwapOpenInterestInformation(ctx, k[0].Pair())
			if err != nil {
				return nil, err
			}
			for i := range data.Data {
				var p currency.Pair
				p, err = e.MatchSymbolWithAvailablePairs(data.Data[i].ContractCode, k[0].Asset, true)
				if err != nil {
					if errors.Is(err, currency.ErrPairNotFound) {
						continue
					}
					return nil, err
				}
				return []futures.OpenInterest{
					{
						Key:          key.NewExchangeAssetPair(e.Name, k[0].Asset, p),
						OpenInterest: data.Data[i].Amount,
					},
				}, nil
			}
		}
	}
	var resp []futures.OpenInterest
	for _, a := range e.GetAssetTypes(true) {
		switch a {
		case asset.Futures:
			data, err := e.FContractOpenInterest(ctx, "", "", currency.EMPTYPAIR)
			if err != nil {
				return nil, err
			}
			for i := range data.Data {
				var p currency.Pair
				var isEnabled, appendData bool
				p, isEnabled, err = e.MatchSymbolCheckEnabled(data.Data[i].ContractCode, a, true)
				if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
					return nil, err
				}
				if !isEnabled {
					continue
				}
				for j := range k {
					if k[j].Pair().Equal(p) {
						appendData = true
						break
					}
				}
				if len(k) > 0 && !appendData {
					continue
				}
				resp = append(resp, futures.OpenInterest{
					Key:          key.NewExchangeAssetPair(e.Name, a, p),
					OpenInterest: data.Data[i].Amount,
				})
			}
		case asset.USDTMarginedFutures:
			enabledPairs, err := e.GetEnabledPairs(asset.USDTMarginedFutures)
			if err != nil {
				return nil, err
			}
			for i := range enabledPairs {
				data, err := e.GetV5OpenInterest(ctx, enabledPairs[i])
				if err != nil {
					return nil, err
				}
				p, isEnabled, err := e.MatchSymbolCheckEnabled(data.Data.ContractCode, a, true)
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
				resp = append(resp, futures.OpenInterest{
					Key:          key.NewExchangeAssetPair(e.Name, a, p),
					OpenInterest: data.Data.Amount.Float64(),
				})
			}
		case asset.CoinMarginedFutures:
			data, err := e.SwapOpenInterestInformation(ctx, currency.EMPTYPAIR)
			if err != nil {
				return nil, err
			}
			for i := range data.Data {
				p, isEnabled, err := e.MatchSymbolCheckEnabled(data.Data[i].ContractCode, a, true)
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
				resp = append(resp, futures.OpenInterest{
					Key:          key.NewExchangeAssetPair(e.Name, a, p),
					OpenInterest: data.Data[i].Amount,
				})
			}
		}
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the UR˜L to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	switch a {
	case asset.Spot:
		cp.Delimiter = currency.UnderscoreDelimiter
		return tradeBaseURL + tradeSpot + cp.Lower().String(), nil
	case asset.Futures:
		if slices.Contains(validContractExpiryCodes, strings.ToUpper(cp.Quote.String())) {
			cp, err = e.pairFromContractExpiryCode(cp)
			if err != nil {
				return "", err
			}
		}
		cp.Delimiter = currency.DashDelimiter
		return tradeBaseURL + tradeFutures + cp.Upper().String(), nil
	case asset.USDTMarginedFutures:
		cp.Delimiter = currency.DashDelimiter
		return tradeBaseURL + tradeFutures + cp.Upper().String(), nil
	case asset.CoinMarginedFutures:
		if slices.Contains(validContractExpiryCodes, strings.ToUpper(cp.Quote.String())) {
			cp, err = e.pairFromContractExpiryCode(cp)
			if err != nil {
				return "", err
			}
		}
		return tradeBaseURL + tradeCoinMargined + cp.Base.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}
