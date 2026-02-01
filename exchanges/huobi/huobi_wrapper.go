package huobi

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets default values for the exchange
func (e *Exchange) SetDefaults() {
	e.Name = "Huobi"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	for _, a := range []asset.Item{asset.Spot, asset.CoinMarginedFutures, asset.Futures} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		}
		switch a {
		case asset.Spot:
			ps.RequestFormat.Uppercase = false
		case asset.CoinMarginedFutures:
			ps.RequestFormat.Delimiter = currency.DashDelimiter
		}
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	for _, a := range []asset.Item{asset.Futures, asset.CoinMarginedFutures} {
		if err := e.DisableAssetWebsocketSupport(a); err != nil {
			log.Errorf(log.ExchangeSys, "%s error disabling %q asset type websocket support: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
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
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				MessageCorrelation:     true,
				GetOrder:               true,
				GetOrders:              true,
				TickerFetching:         true,
				FundingRateFetching:    false, // supported but not implemented // TODO when multi-websocket support added
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
		Subscriptions: defaultSubscriptions.Clone(),
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
		exchange.RestSpot:         huobiAPIURL,
		exchange.RestFutures:      huobiFuturesURL,
		exchange.RestCoinMargined: huobiFuturesURL,
		exchange.WebsocketSpot:    wsSpotURL + wsPublicPath,
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

	return
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

	wsRunningURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            wsSpotURL + wsPublicPath,
		RunningURL:            wsRunningURL,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	err = e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		RateLimit:            request.NewWeightedRateLimitByDuration(20 * time.Millisecond),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		RateLimit:            request.NewWeightedRateLimitByDuration(20 * time.Millisecond),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  wsSpotURL + wsPrivatePath,
		Authenticated:        true,
	})
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
	case asset.Futures:
		ticks := []FuturesBatchTicker{}
		// TODO: Linear swap contracts are coin-m assets
		if coinMTicks, err := e.GetBatchLinearSwapContracts(ctx); err != nil {
			errs = common.AppendError(errs, err)
		} else {
			ticks = append(ticks, coinMTicks...)
		}
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
			return nil, errors.New("invalid data for bid")
		}
		if len(marketData.Tick.Ask) == 0 {
			return nil, errors.New("invalid data for Ask")
		}

		err = ticker.ProcessTicker(&ticker.Price{
			High:         marketData.Tick.High,
			Low:          marketData.Tick.Low,
			Volume:       marketData.Tick.Amount,
			QuoteVolume:  marketData.Tick.Vol,
			Open:         marketData.Tick.Open,
			Close:        marketData.Tick.Close,
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

		err = ticker.ProcessTicker(&ticker.Price{
			High:         marketData.Tick.High,
			Low:          marketData.Tick.Low,
			Volume:       marketData.Tick.Amount,
			QuoteVolume:  marketData.Tick.Vol,
			Open:         marketData.Tick.Open,
			Close:        marketData.Tick.Close,
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
		return nil, errors.New("no account returned")
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
				if balances[j].Type == "frozen" {
					err = a.Balances.Add(balances[j].Currency, accounts.Balance{Hold: balances[j].Balance})
				} else {
					err = a.Balances.Add(balances[j].Currency, accounts.Balance{Total: balances[j].Balance})
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
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
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
			Currency:        withdrawals.Data[i].Currency.String(),
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

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}

	var orderID string
	status := order.New
	switch s.AssetType {
	case asset.Spot:
		accountID, err := strconv.ParseInt(s.ClientID, 10, 64)
		if err != nil {
			return nil, err
		}
		var formattedType SpotNewOrderRequestParamsType
		params := SpotNewOrderRequestParams{
			Amount:    s.Amount,
			Source:    "api",
			Symbol:    s.Pair,
			AccountID: int(accountID),
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

		if s.Type == order.Market {
			status = order.Filled
		}
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
			// At present, Huobi Futures does not support unlimited slippage market price when placing an order.
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
				oType = "post_only"
			}
		default:
			oType = "opponent"
		}
		offset := "open"
		if s.ReduceOnly {
			offset = "close"
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
			// At present, Huobi Futures does not support unlimited slippage market price when placing an order.
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
				oType = "post_only"
			}
		default:
			oType = "opponent"
		}
		offset := "open"
		if s.ReduceOnly {
			offset = "close"
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
	}
	resp, err := s.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Status = status
	return resp, nil
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
	var err error
	switch o.AssetType {
	case asset.Spot:
		var orderIDInt int64
		orderIDInt, err = strconv.ParseInt(o.OrderID, 10, 64)
		if err != nil {
			return err
		}
		_, err = e.CancelExistingOrder(ctx, orderIDInt)
	case asset.CoinMarginedFutures:
		_, err = e.CancelSwapOrder(ctx, o.OrderID, o.ClientID, o.Pair)
	case asset.Futures:
		_, err = e.FCancelOrder(ctx, o.Pair.Base, o.ClientID, o.ClientOrderID)
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
	ids := make([]string, 0, len(o))
	cIDs := make([]string, 0, len(o))
	for i := range o {
		switch {
		case o[i].ClientOrderID != "":
			cIDs = append(cIDs, o[i].ClientID)
		case o[i].OrderID != "":
			ids = append(ids, o[i].OrderID)
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}

	cancelledOrders, err := e.CancelOrderBatch(ctx, ids, cIDs)
	if err != nil {
		return nil, err
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string)}
	for i := range cancelledOrders.Success {
		resp.Status[cancelledOrders.Success[i]] = "true"
	}
	for i := range cancelledOrders.Failed {
		resp.Status[cancelledOrders.Failed[i].OrderID] = cancelledOrders.Failed[i].ErrorMessage
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
		enabledPairs, err := e.GetEnabledPairs(asset.Spot)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range enabledPairs {
			resp, err := e.CancelOpenOrdersBatch(ctx,
				orderCancellation.AccountID,
				enabledPairs[i])
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			if resp.Data.FailedCount > 0 {
				return cancelAllOrdersResponse,
					fmt.Errorf("%v orders failed to cancel",
						resp.Data.FailedCount)
			}
			if resp.Status == "error" {
				return cancelAllOrdersResponse, errors.New(resp.ErrorMessage)
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
				split := strings.Split(a.Successes, ",")
				for x := range split {
					cancelAllOrdersResponse.Status[split[x]] = "success"
				}
				for y := range a.Errors {
					cancelAllOrdersResponse.Status[a.Errors[y].OrderID] = "fail: " + a.Errors[y].ErrMsg
				}
			}
		} else {
			a, err := e.CancelAllSwapOrders(ctx, orderCancellation.Pair)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			split := strings.Split(a.Successes, ",")
			for x := range split {
				cancelAllOrdersResponse.Status[split[x]] = "success"
			}
			for y := range a.Errors {
				cancelAllOrdersResponse.Status[a.Errors[y].OrderID] = "fail: " + a.Errors[y].ErrMsg
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
					cancelAllOrdersResponse.Status[split[x]] = "success"
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
				cancelAllOrdersResponse.Status[split[x]] = "success"
			}
			for y := range a.Data.Errors {
				cancelAllOrdersResponse.Status[strconv.FormatInt(a.Data.Errors[y].OrderID, 10)] = "fail: " + a.Data.Errors[y].ErrMsg
			}
		}
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
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
			return nil, errors.New(e.Name + " - GetOrderInfo orderID mismatch. Expected: " +
				orderID + " Received: " + responseID)
		}
		typeDetails := strings.Split(respData.Type, "-")
		orderSide, err := order.StringToOrderSide(typeDetails[0])
		if err != nil {
			return nil, err
		}
		orderType, err := order.StringToOrderType(typeDetails[1])
		if err != nil {
			return nil, err
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
			Date:           respData.CreatedAt.Time(),
			Status:         orderStatus,
			Price:          respData.Price,
			Amount:         respData.Amount,
			ExecutedAmount: respData.FilledAmount,
			Fee:            respData.FilledFees,
			AssetType:      a,
		}
	case asset.CoinMarginedFutures:
		orderInfo, err := e.GetSwapOrderInfo(ctx, pair, orderID, "")
		if err != nil {
			return nil, err
		}
		var orderVars OrderVars
		for x := range orderInfo.Data {
			orderVars, err = compatibleVars(orderInfo.Data[x].Direction, orderInfo.Data[x].OrderPriceType, orderInfo.Data[x].Status)
			if err != nil {
				return nil, err
			}
			maker := false
			if orderVars.OrderType == order.Limit || orderVars.TimeInForce.Is(order.PostOnly) {
				maker = true
			}
			orderDetail.Trades = append(orderDetail.Trades, order.TradeHistory{
				Price:    orderInfo.Data[x].Price,
				Amount:   orderInfo.Data[x].Volume,
				Fee:      orderInfo.Data[x].Fee,
				Exchange: e.Name,
				TID:      orderInfo.Data[x].OrderIDString,
				Type:     orderVars.OrderType,
				Side:     orderVars.Side,
				IsMaker:  maker,
			})
		}
	case asset.Futures:
		fPair, err := e.FormatSymbol(pair, asset.Futures)
		if err != nil {
			return nil, err
		}
		orderInfo, err := e.FGetOrderInfo(ctx, fPair, orderID, "")
		if err != nil {
			return nil, err
		}
		var orderVars OrderVars
		for x := range orderInfo.Data {
			orderVars, err = compatibleVars(orderInfo.Data[x].Direction, orderInfo.Data[x].OrderPriceType, orderInfo.Data[x].Status)
			if err != nil {
				return nil, err
			}

			orderDetail.Trades = append(orderDetail.Trades, order.TradeHistory{
				Price:    orderInfo.Data[x].Price,
				Amount:   orderInfo.Data[x].Volume,
				Fee:      orderInfo.Data[x].Fee,
				Exchange: e.Name,
				TID:      orderInfo.Data[x].OrderIDString,
				Type:     orderVars.OrderType,
				Side:     orderVars.Side,
				IsMaker:  orderVars.OrderType == order.Limit || orderVars.TimeInForce.Is(order.PostOnly),
			})
		}
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
	return nil, errors.New("unable to match deposit address currency or chain")
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
			return nil, errors.New("currency must be supplied")
		}
		side := ""
		if req.Side == order.Sell {
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
					Price:           resp[x].Price,
					Amount:          resp[x].Amount,
					ExecutedAmount:  resp[x].FilledAmount,
					RemainingAmount: resp[x].Amount - resp[x].FilledAmount,
					Pair:            req.Pairs[i],
					Exchange:        e.Name,
					Date:            resp[x].CreatedAt.Time(),
					AccountID:       strconv.FormatInt(resp[x].AccountID, 10),
					Fee:             resp[x].FilledFees,
				}
				setOrderSideStatusAndType(resp[x].State, resp[x].Type, &orderDetail)
				orders = append(orders, orderDetail)
			}
		}
	case asset.CoinMarginedFutures:
		for x := range req.Pairs {
			var currentPage int64
			for done := false; !done; {
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
				currentPage++
				done = currentPage == openOrders.Data.TotalPage
			}
		}
	case asset.Futures:
		for x := range req.Pairs {
			var currentPage int64
			for done := false; !done; {
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
				currentPage++
				done = currentPage == openOrders.Data.TotalPage
			}
		}
	}
	return req.Filter(e.Name, orders), nil
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
			return nil, errors.New("currency must be supplied")
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
					Price:           resp[x].Price,
					Amount:          resp[x].Amount,
					ExecutedAmount:  resp[x].FilledAmount,
					RemainingAmount: resp[x].Amount - resp[x].FilledAmount,
					Cost:            resp[x].FilledCashAmount,
					CostAsset:       req.Pairs[i].Quote,
					Pair:            req.Pairs[i],
					Exchange:        e.Name,
					Date:            resp[x].CreatedAt.Time(),
					CloseTime:       resp[x].FinishedAt.Time(),
					AccountID:       strconv.FormatInt(resp[x].AccountID, 10),
					Fee:             resp[x].FilledFees,
				}
				setOrderSideStatusAndType(resp[x].State, resp[x].Type, &orderDetail)
				orderDetail.InferCostsAndTimes()
				orders = append(orders, orderDetail)
			}
		}
	case asset.CoinMarginedFutures:
		for x := range req.Pairs {
			var currentPage int64
			for done := false; !done; {
				orderHistory, err := e.GetSwapOrderHistory(ctx,
					req.Pairs[x],
					"all",
					"all",
					[]order.Status{order.AnyStatus},
					int64(req.EndTime.Sub(req.StartTime).Hours()/24),
					currentPage,
					50)
				if err != nil {
					return orders, err
				}
				var orderVars OrderVars
				for x := range orderHistory.Data.Orders {
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
				currentPage++
				done = currentPage == orderHistory.Data.TotalPage
			}
		}
	case asset.Futures:
		for x := range req.Pairs {
			var currentPage int64
			for done := false; !done; {
				openOrders, err := e.FGetOrderHistory(ctx,
					req.Pairs[x],
					"",
					"all",
					"all",
					"limit",
					[]order.Status{order.AnyStatus},
					int64(req.EndTime.Sub(req.StartTime).Hours()/24),
					currentPage,
					50)
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
					if req.Side != orderVars.Side {
						continue
					}
					if req.Type != orderVars.OrderType {
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
				currentPage++
				done = currentPage == openOrders.Data.TotalPage
			}
		}
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
	return e.wsLogin(ctx)
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
		for x := range candles.Data {
			timestamp := candles.Data[x].IDTimestamp.Time()
			if timestamp.Before(req.Start) || timestamp.After(req.End) {
				continue
			}
			timeSeries = append(timeSeries, kline.Candle{
				Time:   timestamp,
				Open:   candles.Data[x].Open,
				High:   candles.Data[x].High,
				Low:    candles.Data[x].Low,
				Close:  candles.Data[x].Close,
				Volume: candles.Data[x].Volume,
			})
		}
	case asset.CoinMarginedFutures:
		// if size, from, to are all populated, only size is considered
		size := int64(-1)
		candles, err := e.GetSwapKlineData(ctx, req.Pair, e.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.Start, req.End)
		if err != nil {
			return nil, err
		}
		for x := range candles.Data {
			timestamp := candles.Data[x].IDTimestamp.Time()
			if timestamp.Before(req.Start) || timestamp.After(req.End) {
				continue
			}
			timeSeries = append(timeSeries, kline.Candle{
				Time:   timestamp,
				Open:   candles.Data[x].Open,
				High:   candles.Data[x].High,
				Low:    candles.Data[x].Low,
				Close:  candles.Data[x].Close,
				Volume: candles.Data[x].Volume,
			})
		}
	}

	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
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
			for x := range candles.Data {
				// align response data
				timestamp := candles.Data[x].IDTimestamp.Time()
				if timestamp.Before(req.Start) || timestamp.After(req.End) {
					continue
				}
				timeSeries = append(timeSeries, kline.Candle{
					Time:   timestamp,
					Open:   candles.Data[x].Open,
					High:   candles.Data[x].High,
					Low:    candles.Data[x].Low,
					Close:  candles.Data[x].Close,
					Volume: candles.Data[x].Volume,
				})
			}
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
			for x := range candles.Data {
				// align response data
				timestamp := candles.Data[x].IDTimestamp.Time()
				if timestamp.Before(req.Start) || timestamp.After(req.End) {
					continue
				}
				timeSeries = append(timeSeries, kline.Candle{
					Time:   timestamp,
					Open:   candles.Data[x].Open,
					High:   candles.Data[x].High,
					Low:    candles.Data[x].Low,
					Close:  candles.Data[x].Close,
					Volume: candles.Data[x].Volume,
				})
			}
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
		return resp, errors.New("invalid orderSide")
	}
	switch orderPriceType {
	case "limit":
		resp.OrderType = order.Limit
	case "opponent":
		resp.OrderType = order.Market
	case "post_only":
		resp.OrderType = order.Limit
		resp.TimeInForce = order.PostOnly
	default:
		return resp, errors.New("invalid orderPriceType")
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
		return resp, errors.New("invalid orderStatus")
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
		return nil, errors.New("no chains returned from currencies API")
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
			contractSplitIndex := strings.Split(result[x].ContractCode, currency.DashDelimiter)
			var cp, underlying currency.Pair
			cp, err = currency.NewPairFromStrings(contractSplitIndex[0], contractSplitIndex[1])
			if err != nil {
				return nil, err
			}
			underlying, err = currency.NewPairFromStrings(result[x].Symbol, "USD")
			if err != nil {
				return nil, err
			}
			var s time.Time
			s, err = time.Parse("20060102", result[x].CreateDate)
			if err != nil {
				return nil, err
			}

			resp = append(resp, futures.Contract{
				Exchange:           e.Name,
				Name:               cp,
				Underlying:         underlying,
				Asset:              item,
				StartDate:          s,
				SettlementType:     futures.Inverse,
				IsActive:           result[x].ContractStatus == 1,
				Type:               futures.Perpetual,
				SettlementCurrency: currency.USD,
				Multiplier:         result[x].ContractSize,
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
			contractSplitIndex := strings.Split(result.Data[x].ContractCode, result.Data[x].Symbol)
			var cp, underlying currency.Pair
			cp, err = currency.NewPairFromStrings(result.Data[x].Symbol, contractSplitIndex[1])
			if err != nil {
				return nil, err
			}
			underlying, err = currency.NewPairFromStrings(result.Data[x].Symbol, "USD")
			if err != nil {
				return nil, err
			}
			var startTime, endTime time.Time
			startTime, err = time.Parse("20060102", result.Data[x].CreateDate)
			if err != nil {
				return nil, err
			}
			if result.Data[x].DeliveryTime.Time().IsZero() {
				endTime = result.Data[x].DeliveryTime.Time()
			} else {
				endTime = result.Data[x].SettlementTime.Time()
			}
			contractLength := endTime.Sub(startTime)
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
				StartDate:          startTime,
				EndDate:            endTime,
				SettlementType:     futures.Linear,
				IsActive:           result.Data[x].ContractStatus == 1,
				Type:               ct,
				SettlementCurrency: currency.USD,
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
	if r.Asset != asset.CoinMarginedFutures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}

	var rates []FundingRatesData
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
	resp := make([]fundingrate.LatestRateResponse, 0, len(rates))
	for i := range rates {
		if rates[i].ContractCode == "" {
			// formatting to match documentation
			rates[i].ContractCode = rates[i].Symbol + "-USD"
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
				Rate: decimal.NewFromFloat(rates[i].FundingRate),
			},
			TimeOfNextRate: nft,
			TimeChecked:    time.Now(),
		}
		if r.IncludePredictedRate {
			rate.PredictedUpcomingRate = fundingrate.Rate{
				Time: rate.TimeOfNextRate,
				Rate: decimal.NewFromFloat(rates[i].EstimatedRate),
			}
		}
		resp = append(resp, rate)
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.CoinMarginedFutures, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.Futures && k[i].Asset != asset.CoinMarginedFutures {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	if len(k) == 1 {
		switch k[0].Asset {
		case asset.Futures:
			if !slices.Contains(validContractExpiryCodes, strings.ToUpper(k[0].Pair().Quote.String())) {
				// Huobi does not like requests being made with contract expiry in them (eg BTC240109)
				return nil, fmt.Errorf("%w %v, must use shorthand such as CW (current week)", currency.ErrCurrencyNotSupported, k[0].Pair())
			}
			data, err := e.FContractOpenInterest(ctx, "", "", k[0].Pair())
			if err != nil {
				data2, err2 := e.ContractOpenInterestUSDT(ctx, k[0].Pair(), currency.EMPTYPAIR, "", "")
				if err2 != nil {
					return nil, fmt.Errorf("%w %w", err, err2)
				}
				data.Data = data2
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
			uData, err := e.ContractOpenInterestUSDT(ctx, currency.EMPTYPAIR, currency.EMPTYPAIR, "", "")
			if err != nil {
				return nil, err
			}
			allData := make([]UContractOpenInterest, 0, len(data.Data)+len(uData))
			allData = append(allData, data.Data...)
			allData = append(allData, uData...)
			for i := range allData {
				var p currency.Pair
				var isEnabled, appendData bool
				p, isEnabled, err = e.MatchSymbolCheckEnabled(allData[i].ContractCode, a, true)
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
					OpenInterest: allData[i].Amount,
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
		if !cp.Quote.Equal(currency.USD) && !cp.Quote.Equal(currency.USDT) {
			// todo: support long dated currencies
			return "", fmt.Errorf("%w %v requires translating currency into static contracts eg 'weekly'", common.ErrNotYetImplemented, a)
		}
		cp.Delimiter = currency.DashDelimiter
		return tradeBaseURL + tradeFutures + cp.Upper().String(), nil
	case asset.CoinMarginedFutures:
		if !cp.Quote.Equal(currency.USD) && !cp.Quote.Equal(currency.USDT) {
			// todo: support long dated currencies
			return "", fmt.Errorf("%w %v requires translating currency into static contracts eg 'weekly'", common.ErrNotYetImplemented, a)
		}
		return tradeBaseURL + tradeCoinMargined + cp.Base.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}
