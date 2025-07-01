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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets default values for the exchange
func (h *HUOBI) SetDefaults() {
	h.Name = "Huobi"
	h.Enabled = true
	h.Verbose = true
	h.API.CredentialsValidator.RequiresKey = true
	h.API.CredentialsValidator.RequiresSecret = true

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
		if err := h.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", h.Name, a, err)
		}
	}

	for _, a := range []asset.Item{asset.Futures, asset.CoinMarginedFutures} {
		if err := h.DisableAssetWebsocketSupport(a); err != nil {
			log.Errorf(log.ExchangeSys, "%s error disabling %q asset type websocket support: %s", h.Name, a, err)
		}
	}

	h.Features = exchange.Features{
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
	h.Requester, err = request.New(h.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	h.API.Endpoints = h.NewEndpoints()
	err = h.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:         huobiAPIURL,
		exchange.RestFutures:      huobiFuturesURL,
		exchange.RestCoinMargined: huobiFuturesURL,
		exchange.WebsocketSpot:    wsSpotURL + wsPublicPath,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	h.Websocket = websocket.NewManager()
	h.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	h.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	h.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Bootstrap ensures that future contract expiry codes are loaded if AutoPairUpdates is not enabled
func (h *HUOBI) Bootstrap(ctx context.Context) (continueBootstrap bool, err error) {
	continueBootstrap = true

	if !h.GetEnabledFeatures().AutoPairUpdates && h.SupportsAsset(asset.Futures) {
		_, err = h.FetchTradablePairs(ctx, asset.Futures)
	}

	return
}

// Setup sets user configuration
func (h *HUOBI) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		h.SetEnabled(false)
		return nil
	}
	err = h.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := h.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = h.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            wsSpotURL + wsPublicPath,
		RunningURL:            wsRunningURL,
		Connector:             h.WsConnect,
		Subscriber:            h.Subscribe,
		Unsubscriber:          h.Unsubscribe,
		GenerateSubscriptions: h.generateSubscriptions,
		Features:              &h.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	err = h.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		RateLimit:            request.NewWeightedRateLimitByDuration(20 * time.Millisecond),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
	if err != nil {
		return err
	}

	return h.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		RateLimit:            request.NewWeightedRateLimitByDuration(20 * time.Millisecond),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  wsSpotURL + wsPrivatePath,
		Authenticated:        true,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (h *HUOBI) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !h.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}

	var pairs []currency.Pair
	switch a {
	case asset.Spot:
		symbols, err := h.GetSymbols(ctx)
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
		symbols, err := h.GetSwapMarkets(ctx, currency.EMPTYPAIR)
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
		symbols, err := h.FGetContractInfo(ctx, "", "", currency.EMPTYPAIR)
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
		h.futureContractCodesMutex.Lock()
		h.futureContractCodes = expiryCodeDates
		h.futureContractCodesMutex.Unlock()
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (h *HUOBI) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := h.GetAssetTypes(false)
	for x := range assets {
		pairs, err := h.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return err
		}
		err = h.UpdatePairs(pairs, assets[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return h.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (h *HUOBI) UpdateTickers(ctx context.Context, a asset.Item) error {
	var errs error
	switch a {
	case asset.Spot:
		ticks, err := h.GetTickers(ctx)
		if err != nil {
			return err
		}
		for i := range ticks.Data {
			var cp currency.Pair
			cp, _, err = h.MatchSymbolCheckEnabled(ticks.Data[i].Symbol, a, false)
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
				ExchangeName: h.Name,
				AssetType:    a,
				LastUpdated:  time.Now(),
			})
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		}
	case asset.CoinMarginedFutures:
		ticks, err := h.GetBatchCoinMarginSwapContracts(ctx)
		if err != nil {
			return err
		}
		for i := range ticks {
			var cp currency.Pair
			cp, _, err = h.MatchSymbolCheckEnabled(ticks[i].ContractCode, a, true)
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
				ExchangeName: h.Name,
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
		if coinMTicks, err := h.GetBatchLinearSwapContracts(ctx); err != nil {
			errs = common.AppendError(errs, err)
		} else {
			ticks = append(ticks, coinMTicks...)
		}
		if futureTicks, err := h.GetBatchFuturesContracts(ctx); err != nil {
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
					cp, err = h.pairFromContractExpiryCode(cp)
				}
				if err == nil {
					cp, _, err = h.MatchSymbolCheckEnabled(cp.String(), a, true)
				}
			} else {
				cp, _, err = h.MatchSymbolCheckEnabled(ticks[i].ContractCode, a, true)
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
				ExchangeName: h.Name,
				AssetType:    a,
				LastUpdated:  ticks[i].Timestamp.Time(),
			})
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	return errs
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HUOBI) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !h.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	switch a {
	case asset.Spot:
		tickerData, err := h.Get24HrMarketSummary(ctx, p)
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
			ExchangeName: h.Name,
			AssetType:    asset.Spot,
		})
		if err != nil {
			return nil, err
		}
	case asset.CoinMarginedFutures:
		marketData, err := h.GetSwapMarketOverview(ctx, p)
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
			ExchangeName: h.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	case asset.Futures:
		marketData, err := h.FGetMarketOverviewData(ctx, p)
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
			ExchangeName: h.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	}
	return ticker.GetTicker(h.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HUOBI) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !assetType.IsValid() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	book := &orderbook.Book{
		Exchange:          h.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: h.ValidateOrderbook,
	}
	var err error
	switch assetType {
	case asset.Spot:
		var orderbookNew *Orderbook
		orderbookNew, err = h.GetDepth(ctx,
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
		orderbookNew, err = h.FGetMarketDepth(ctx, p, "step0")
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
		orderbookNew, err = h.GetSwapMarketDepth(ctx, p, "step0")
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
	return orderbook.Get(h.Name, p, assetType)
}

// GetAccountID returns the account ID for trades
func (h *HUOBI) GetAccountID(ctx context.Context) ([]Account, error) {
	acc, err := h.GetAccounts(ctx)
	if err != nil {
		return nil, err
	}

	if len(acc) < 1 {
		return nil, errors.New("no account returned")
	}

	return acc, nil
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// HUOBI exchange - to-do
func (h *HUOBI) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = h.Name
	switch assetType {
	case asset.Spot:
		accounts, err := h.GetAccountID(ctx)
		if err != nil {
			return info, err
		}
		for i := range accounts {
			if accounts[i].Type != "spot" {
				continue
			}
			acc.ID = strconv.FormatInt(accounts[i].ID, 10)
			balances, err := h.GetAccountBalance(ctx, acc.ID)
			if err != nil {
				return info, err
			}

			var currencyDetails []account.Balance
		balance:
			for j := range balances {
				frozen := balances[j].Type == "frozen"
				for i := range currencyDetails {
					if currencyDetails[i].Currency.String() == balances[j].Currency {
						if frozen {
							currencyDetails[i].Hold = balances[j].Balance
						} else {
							currencyDetails[i].Total = balances[j].Balance
						}
						continue balance
					}
				}

				if frozen {
					currencyDetails = append(currencyDetails,
						account.Balance{
							Currency: currency.NewCode(balances[j].Currency),
							Hold:     balances[j].Balance,
						})
				} else {
					currencyDetails = append(currencyDetails,
						account.Balance{
							Currency: currency.NewCode(balances[j].Currency),
							Total:    balances[j].Balance,
						})
				}
			}
			acc.Currencies = currencyDetails
		}

	case asset.CoinMarginedFutures:
		// fetch swap account info
		acctInfo, err := h.GetSwapAccountInfo(ctx, currency.EMPTYPAIR)
		if err != nil {
			return info, err
		}

		var mainAcctBalances []account.Balance
		for x := range acctInfo.Data {
			mainAcctBalances = append(mainAcctBalances, account.Balance{
				Currency: currency.NewCode(acctInfo.Data[x].Symbol),
				Total:    acctInfo.Data[x].MarginBalance,
				Hold:     acctInfo.Data[x].MarginFrozen,
				Free:     acctInfo.Data[x].MarginAvailable,
			})
		}

		info.Accounts = append(info.Accounts, account.SubAccount{
			Currencies: mainAcctBalances,
			AssetType:  assetType,
		})

		// fetch subaccounts data
		subAccsData, err := h.GetSwapAllSubAccAssets(ctx, currency.EMPTYPAIR)
		if err != nil {
			return info, err
		}
		var currencyDetails []account.Balance
		for x := range subAccsData.Data {
			a, err := h.SwapSingleSubAccAssets(ctx,
				currency.EMPTYPAIR,
				subAccsData.Data[x].SubUID)
			if err != nil {
				return info, err
			}
			for y := range a.Data {
				currencyDetails = append(currencyDetails, account.Balance{
					Currency: currency.NewCode(a.Data[y].Symbol),
					Total:    a.Data[y].MarginBalance,
					Hold:     a.Data[y].MarginFrozen,
					Free:     a.Data[y].MarginAvailable,
				})
			}
		}
		acc.Currencies = currencyDetails
	case asset.Futures:
		// fetch main account data
		mainAcctData, err := h.FGetAccountInfo(ctx, currency.EMPTYCODE)
		if err != nil {
			return info, err
		}

		var mainAcctBalances []account.Balance
		for x := range mainAcctData.AccData {
			mainAcctBalances = append(mainAcctBalances, account.Balance{
				Currency: currency.NewCode(mainAcctData.AccData[x].Symbol),
				Total:    mainAcctData.AccData[x].MarginBalance,
				Hold:     mainAcctData.AccData[x].MarginFrozen,
				Free:     mainAcctData.AccData[x].MarginAvailable,
			})
		}

		info.Accounts = append(info.Accounts, account.SubAccount{
			Currencies: mainAcctBalances,
			AssetType:  assetType,
		})

		// fetch subaccounts data
		subAccsData, err := h.FGetAllSubAccountAssets(ctx, currency.EMPTYCODE)
		if err != nil {
			return info, err
		}
		var currencyDetails []account.Balance
		for x := range subAccsData.Data {
			a, err := h.FGetSingleSubAccountInfo(ctx,
				"",
				strconv.FormatInt(subAccsData.Data[x].SubUID, 10))
			if err != nil {
				return info, err
			}
			for y := range a.AssetsData {
				currencyDetails = append(currencyDetails, account.Balance{
					Currency: currency.NewCode(a.AssetsData[y].Symbol),
					Total:    a.AssetsData[y].MarginBalance,
					Hold:     a.AssetsData[y].MarginFrozen,
					Free:     a.AssetsData[y].MarginAvailable,
				})
			}
		}
		acc.Currencies = currencyDetails
	}
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)
	creds, err := h.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	if err := account.Process(&info, creds); err != nil {
		return info, err
	}
	return info, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (h *HUOBI) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (h *HUOBI) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	withdrawals, err := h.SearchForExistedWithdrawsAndDeposits(ctx, c, "withdraw", "", 0, 500)
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
func (h *HUOBI) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	var resp []trade.Data
	pFmt, err := h.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}

	p = p.Format(pFmt)
	switch a {
	case asset.Spot:
		var sTrades []TradeHistory
		sTrades, err = h.GetTradeHistory(ctx, p, 2000)
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
					Exchange:     h.Name,
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
		fTrades, err = h.FRequestPublicBatchTrades(ctx, p, 2000)
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
					Exchange:     h.Name,
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
		cTrades, err = h.GetBatchTrades(ctx, p, 2000)
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
				Exchange:     h.Name,
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

	err = h.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (h *HUOBI) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (h *HUOBI) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(h.GetTradingRequirements()); err != nil {
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
		response, err := h.SpotNewOrder(ctx, &params)
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
		orderResp, err := h.PlaceSwapOrders(ctx,
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
		o, err := h.FOrder(ctx,
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

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (h *HUOBI) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (h *HUOBI) CancelOrder(ctx context.Context, o *order.Cancel) error {
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
		_, err = h.CancelExistingOrder(ctx, orderIDInt)
	case asset.CoinMarginedFutures:
		_, err = h.CancelSwapOrder(ctx, o.OrderID, o.ClientID, o.Pair)
	case asset.Futures:
		_, err = h.FCancelOrder(ctx, o.Pair.Base, o.ClientID, o.ClientOrderID)
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (h *HUOBI) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
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

	cancelledOrders, err := h.CancelOrderBatch(ctx, ids, cIDs)
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
func (h *HUOBI) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch orderCancellation.AssetType {
	case asset.Spot:
		enabledPairs, err := h.GetEnabledPairs(asset.Spot)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range enabledPairs {
			resp, err := h.CancelOpenOrdersBatch(ctx,
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
			enabledPairs, err := h.GetEnabledPairs(asset.CoinMarginedFutures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				a, err := h.CancelAllSwapOrders(ctx, enabledPairs[i])
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
			a, err := h.CancelAllSwapOrders(ctx, orderCancellation.Pair)
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
			enabledPairs, err := h.GetEnabledPairs(asset.Futures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				a, err := h.FCancelAllOrders(ctx, enabledPairs[i], "", "")
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
			a, err := h.FCancelAllOrders(ctx, orderCancellation.Pair, "", "")
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
func (h *HUOBI) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := h.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	var orderDetail order.Detail
	switch assetType {
	case asset.Spot:
		oID, err := strconv.ParseInt(orderID, 10, 64)
		if err != nil {
			return nil, err
		}
		resp, err := h.GetOrder(ctx, oID)
		if err != nil {
			return nil, err
		}
		respData := &resp
		if respData.ID == 0 {
			return nil, fmt.Errorf("%s - order not found for orderid %s", h.Name, orderID)
		}
		responseID := strconv.FormatInt(respData.ID, 10)
		if responseID != orderID {
			return nil, errors.New(h.Name + " - GetOrderInfo orderID mismatch. Expected: " +
				orderID + " Received: " + responseID)
		}
		typeDetails := strings.Split(respData.Type, "-")
		orderSide, err := order.StringToOrderSide(typeDetails[0])
		if err != nil {
			if h.Websocket.IsConnected() {
				h.Websocket.DataHandler <- order.ClassificationError{
					Exchange: h.Name,
					OrderID:  orderID,
					Err:      err,
				}
			} else {
				return nil, err
			}
		}
		orderType, err := order.StringToOrderType(typeDetails[1])
		if err != nil {
			if h.Websocket.IsConnected() {
				h.Websocket.DataHandler <- order.ClassificationError{
					Exchange: h.Name,
					OrderID:  orderID,
					Err:      err,
				}
			} else {
				return nil, err
			}
		}
		orderStatus, err := order.StringToOrderStatus(respData.State)
		if err != nil {
			if h.Websocket.IsConnected() {
				h.Websocket.DataHandler <- order.ClassificationError{
					Exchange: h.Name,
					OrderID:  orderID,
					Err:      err,
				}
			} else {
				return nil, err
			}
		}
		var p currency.Pair
		var a asset.Item
		p, a, err = h.GetRequestFormattedPairAndAssetType(respData.Symbol)
		if err != nil {
			return nil, err
		}
		orderDetail = order.Detail{
			Exchange:       h.Name,
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
		orderInfo, err := h.GetSwapOrderInfo(ctx, pair, orderID, "")
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
				Exchange: h.Name,
				TID:      orderInfo.Data[x].OrderIDString,
				Type:     orderVars.OrderType,
				Side:     orderVars.Side,
				IsMaker:  maker,
			})
		}
	case asset.Futures:
		fPair, err := h.FormatSymbol(pair, asset.Futures)
		if err != nil {
			return nil, err
		}
		orderInfo, err := h.FGetOrderInfo(ctx, fPair, orderID, "")
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
				Exchange: h.Name,
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
func (h *HUOBI) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	resp, err := h.QueryDepositAddress(ctx, cryptocurrency)
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
func (h *HUOBI) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := h.Withdraw(ctx,
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
func (h *HUOBI) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBI) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HUOBI) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !h.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return h.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (h *HUOBI) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
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
		creds, err := h.GetCredentials(ctx)
		if err != nil {
			return nil, err
		}
		for i := range req.Pairs {
			resp, err := h.GetOpenOrders(ctx,
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
					Exchange:        h.Name,
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
				openOrders, err := h.GetSwapOpenOrders(ctx,
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
						Exchange:        h.Name,
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
				openOrders, err := h.FGetOpenOrders(ctx,
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
						Exchange:        h.Name,
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
	return req.Filter(h.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (h *HUOBI) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
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
			resp, err := h.GetOrders(ctx,
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
					Exchange:        h.Name,
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
				orderHistory, err := h.GetSwapOrderHistory(ctx,
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
						Exchange:        h.Name,
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
				openOrders, err := h.FGetOrderHistory(ctx,
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
						Exchange:        h.Name,
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
	return req.Filter(h.Name, orders), nil
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
func (h *HUOBI) AuthenticateWebsocket(ctx context.Context) error {
	return h.wsLogin(ctx)
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (h *HUOBI) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := h.UpdateAccountInfo(ctx, assetType)
	return h.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (h *HUOBI) FormatExchangeKlineInterval(in kline.Interval) string {
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
func (h *HUOBI) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := h.GetKlineRequest(pair, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	switch a {
	case asset.Spot:
		candles, err := h.GetSpotKline(ctx, KlinesRequestParams{
			Period: h.FormatExchangeKlineInterval(req.ExchangeInterval),
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
		candles, err := h.FGetKlineData(ctx, req.Pair, h.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.Start, req.End)
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
		candles, err := h.GetSwapKlineData(ctx, req.Pair, h.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.Start, req.End)
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
func (h *HUOBI) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := h.GetKlineExtendedRequest(pair, a, interval, start, end)
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
			candles, err = h.FGetKlineData(ctx, req.Pair, h.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.RangeHolder.Ranges[i].Start.Time, req.RangeHolder.Ranges[i].End.Time)
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
			candles, err = h.GetSwapKlineData(ctx, req.Pair, h.FormatExchangeKlineInterval(req.ExchangeInterval), size, req.RangeHolder.Ranges[i].Start.Time, req.RangeHolder.Ranges[i].End.Time)
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
func (h *HUOBI) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	resp, err := h.GetCurrenciesIncludingChains(ctx, cryptocurrency)
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
func (h *HUOBI) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return h.GetCurrentServerTime(ctx)
}

// GetFuturesContractDetails returns details about futures contracts
func (h *HUOBI) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !h.SupportsAsset(item) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}

	switch item {
	case asset.CoinMarginedFutures:
		result, err := h.GetSwapMarkets(ctx, currency.EMPTYPAIR)
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
				Exchange:             h.Name,
				Name:                 cp,
				Underlying:           underlying,
				Asset:                item,
				StartDate:            s,
				SettlementType:       futures.Inverse,
				IsActive:             result[x].ContractStatus == 1,
				Type:                 futures.Perpetual,
				SettlementCurrencies: currency.Currencies{currency.USD},
				Multiplier:           result[x].ContractSize,
			})
		}
		return resp, nil
	case asset.Futures:
		result, err := h.FGetContractInfo(ctx, "", "", currency.EMPTYPAIR)
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
			var s, e time.Time
			s, err = time.Parse("20060102", result.Data[x].CreateDate)
			if err != nil {
				return nil, err
			}
			if result.Data[x].DeliveryTime.Time().IsZero() {
				e = result.Data[x].DeliveryTime.Time()
			} else {
				e = result.Data[x].SettlementTime.Time()
			}
			contractLength := e.Sub(s)
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
				Exchange:             h.Name,
				Name:                 cp,
				Underlying:           underlying,
				Asset:                item,
				StartDate:            s,
				EndDate:              e,
				SettlementType:       futures.Linear,
				IsActive:             result.Data[x].ContractStatus == 1,
				Type:                 ct,
				SettlementCurrencies: currency.Currencies{currency.USD},
				Multiplier:           result.Data[x].ContractSize,
			})
		}
		return resp, nil
	}
	return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
}

// GetLatestFundingRates returns the latest funding rates data
func (h *HUOBI) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.CoinMarginedFutures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}

	var rates []FundingRatesData
	if r.Pair.IsEmpty() {
		batchRates, err := h.GetSwapFundingRates(ctx)
		if err != nil {
			return nil, err
		}
		rates = batchRates.Data
	} else {
		rateResp, err := h.GetSwapFundingRate(ctx, r.Pair)
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
		cp, isEnabled, err := h.MatchSymbolCheckEnabled(rates[i].ContractCode, r.Asset, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !isEnabled {
			continue
		}
		var isPerp bool
		isPerp, err = h.IsPerpetualFutureCurrency(r.Asset, cp)
		if err != nil {
			return nil, err
		}
		if !isPerp {
			continue
		}
		ft, nft := rates[i].FundingTime.Time(), rates[i].NextFundingTime.Time()
		var fri time.Duration
		if len(h.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
			// can infer funding rate interval from the only funding rate frequency defined
			for k := range h.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
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
			Exchange: h.Name,
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
func (h *HUOBI) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.CoinMarginedFutures, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (h *HUOBI) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (h *HUOBI) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
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
			data, err := h.FContractOpenInterest(ctx, "", "", k[0].Pair())
			if err != nil {
				data2, err2 := h.ContractOpenInterestUSDT(ctx, k[0].Pair(), currency.EMPTYPAIR, "", "")
				if err2 != nil {
					return nil, fmt.Errorf("%w %w", err, err2)
				}
				data.Data = data2
			}

			for i := range data.Data {
				var p currency.Pair
				p, err = h.MatchSymbolWithAvailablePairs(data.Data[i].ContractCode, k[0].Asset, true)
				if err != nil {
					if errors.Is(err, currency.ErrPairNotFound) {
						continue
					}
					return nil, err
				}
				return []futures.OpenInterest{
					{
						Key: key.ExchangePairAsset{
							Exchange: h.Name,
							Base:     p.Base.Item,
							Quote:    p.Quote.Item,
							Asset:    k[0].Asset,
						},
						OpenInterest: data.Data[i].Amount,
					},
				}, nil
			}
		case asset.CoinMarginedFutures:
			data, err := h.SwapOpenInterestInformation(ctx, k[0].Pair())
			if err != nil {
				return nil, err
			}
			for i := range data.Data {
				var p currency.Pair
				p, err = h.MatchSymbolWithAvailablePairs(data.Data[i].ContractCode, k[0].Asset, true)
				if err != nil {
					if errors.Is(err, currency.ErrPairNotFound) {
						continue
					}
					return nil, err
				}
				return []futures.OpenInterest{
					{
						Key: key.ExchangePairAsset{
							Exchange: h.Name,
							Base:     p.Base.Item,
							Quote:    p.Quote.Item,
							Asset:    k[0].Asset,
						},
						OpenInterest: data.Data[i].Amount,
					},
				}, nil
			}
		}
	}
	var resp []futures.OpenInterest
	for _, a := range h.GetAssetTypes(true) {
		switch a {
		case asset.Futures:
			data, err := h.FContractOpenInterest(ctx, "", "", currency.EMPTYPAIR)
			if err != nil {
				return nil, err
			}
			uData, err := h.ContractOpenInterestUSDT(ctx, currency.EMPTYPAIR, currency.EMPTYPAIR, "", "")
			if err != nil {
				return nil, err
			}
			allData := make([]UContractOpenInterest, 0, len(data.Data)+len(uData))
			allData = append(allData, data.Data...)
			allData = append(allData, uData...)
			for i := range allData {
				var p currency.Pair
				var isEnabled, appendData bool
				p, isEnabled, err = h.MatchSymbolCheckEnabled(allData[i].ContractCode, a, true)
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
					Key: key.ExchangePairAsset{
						Exchange: h.Name,
						Base:     p.Base.Item,
						Quote:    p.Quote.Item,
						Asset:    a,
					},
					OpenInterest: allData[i].Amount,
				})
			}
		case asset.CoinMarginedFutures:
			data, err := h.SwapOpenInterestInformation(ctx, currency.EMPTYPAIR)
			if err != nil {
				return nil, err
			}
			for i := range data.Data {
				p, isEnabled, err := h.MatchSymbolCheckEnabled(data.Data[i].ContractCode, a, true)
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
					Key: key.ExchangePairAsset{
						Exchange: h.Name,
						Base:     p.Base.Item,
						Quote:    p.Quote.Item,
						Asset:    a,
					},
					OpenInterest: data.Data[i].Amount,
				})
			}
		}
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the UR˜L to the exchange's trade page for the given asset and currency pair
func (h *HUOBI) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := h.CurrencyPairs.IsPairEnabled(cp, a)
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
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}
