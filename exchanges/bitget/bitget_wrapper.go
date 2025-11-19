package bitget

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (e *Exchange) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	e.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = e.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = e.BaseCurrencies
	if err := e.SetupDefaults(exchCfg); err != nil {
		return nil, err
	}
	if e.Features.Supports.RESTCapabilities.AutoPairUpdates {
		if err := e.UpdateTradablePairs(ctx); err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Bitget
func (e *Exchange) SetDefaults() {
	e.Name = "Bitget"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	e.API.CredentialsValidator.RequiresClientID = true
	requestFmt := &currency.PairFormat{Uppercase: true}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.Futures, asset.Margin, asset.CrossMargin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:                    true, // Supported for spot and futures, but not margin
				AutoPairUpdates:                   true,
				AccountBalance:                    true,
				CryptoDeposit:                     true,
				CryptoWithdrawal:                  true,
				FiatWithdraw:                      false,
				GetOrder:                          true,
				GetOrders:                         true,
				CancelOrders:                      true,
				CancelOrder:                       true,
				SubmitOrder:                       true,
				SubmitOrders:                      true,
				ModifyOrder:                       true,
				DepositHistory:                    true,
				WithdrawalHistory:                 true,
				TradeHistory:                      true,
				UserTradeHistory:                  true,
				TradeFee:                          true,
				FiatDepositFee:                    false,
				FiatWithdrawalFee:                 false,
				CryptoDepositFee:                  false,
				CryptoWithdrawalFee:               false,
				TickerFetching:                    true,
				KlineFetching:                     true,
				TradeFetching:                     true,
				OrderbookFetching:                 true,
				AccountInfo:                       true,
				FiatDeposit:                       false,
				DeadMansSwitch:                    false,
				FundingRateFetching:               true,
				AuthenticatedEndpoints:            true,
				CandleHistory:                     true,
				MultiChainDeposits:                true,
				MultiChainWithdrawals:             true,
				MultiChainDepositRequiresChainSet: true,
				HasAssetTypeAccountSegregation:    true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerBatching:                 false,
				AccountBalance:                 true,
				CryptoDeposit:                  false,
				CryptoWithdrawal:               false,
				FiatWithdraw:                   false,
				GetOrder:                       false,
				GetOrders:                      true,
				CancelOrders:                   false,
				CancelOrder:                    false,
				SubmitOrder:                    false,
				SubmitOrders:                   false,
				ModifyOrder:                    false,
				DepositHistory:                 false,
				WithdrawalHistory:              false,
				TradeHistory:                   false,
				UserTradeHistory:               false,
				TradeFee:                       false,
				FiatDepositFee:                 false,
				FiatWithdrawalFee:              false,
				CryptoDepositFee:               false,
				CryptoWithdrawalFee:            false,
				TickerFetching:                 true,
				KlineFetching:                  true,
				TradeFetching:                  true,
				OrderbookFetching:              true,
				AccountInfo:                    true,
				FiatDeposit:                    false,
				DeadMansSwitch:                 false,
				FundingRateFetching:            false,
				PredictedFundingRate:           false,
				Subscribe:                      true,
				Unsubscribe:                    true,
				AuthenticatedEndpoints:         true,
				MessageCorrelation:             false,
				MessageSequenceNumbers:         false,
				CandleHistory:                  false,
				MultiChainDeposits:             false,
				MultiChainWithdrawals:          false,
				HasAssetTypeAccountSegregation: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
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
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 200,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
		TradingRequirements: protocol.TradingRequirements{
			SpotMarketBuyQuotation: false,
			SpotMarketSellBase:     true,
			ClientOrderID:          false,
		},
	}
	if e.Requester, err = request.New(e.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(rateLimits)); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	if err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                bitgetAPIURL,
		exchange.WebsocketSpot:           bitgetPublicWSURL,
		exchange.WebsocketSandboxPublic:  bitgetPublicSandboxWSUrl,
		exchange.WebsocketSandboxPrivate: bitgetPrivateSandboxWSUrl,
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
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
	wsRunningEndpoint, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		DefaultURL:                             bitgetPublicWSURL,
		RunningURL:                             wsRunningEndpoint,
		Connector:                              e.WsConnect,
		Subscriber:                             e.Subscribe,
		Unsubscriber:                           e.Unsubscribe,
		GenerateSubscriptions:                  e.generateDefaultSubscriptions,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		MaxWebsocketSubscriptionsPerConnection: 1000,
		RateLimitDefinitions:                   rateLimits,
	}); err != nil {
		return err
	}
	var wsPub string
	var wsPriv string
	switch e.isDemoTrading {
	case true:
		wsPub = bitgetPublicSandboxWSUrl
		wsPriv = bitgetPrivateSandboxWSUrl
	case false:
		wsPub = bitgetPublicWSURL
		wsPriv = bitgetPrivateWSURL
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  wsPub,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            rateLimits[rateSubscription],
	}); err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  wsPriv,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Authenticated:        true,
		RateLimit:            rateLimits[rateSubscription],
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch a {
	case asset.Spot:
		resp, err := e.GetSymbolInfo(ctx, currency.Pair{})
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, len(resp))
		var filter int
		for x := range resp {
			if (resp[x].PricePrecision == 0 && resp[x].QuantityPrecision == 0 && resp[x].QuotePrecision == 0) || resp[x].OpenTime.Time().After(time.Now().Add(time.Hour*24*365)) {
				continue
			}
			pairs[filter] = currency.NewPair(resp[x].BaseCoin, resp[x].QuoteCoin)
			filter++
		}
		return pairs[:filter:filter], nil
	case asset.Futures:
		var resp []FutureTickerResp
		req := []string{"USDT-FUTURES", "COIN-FUTURES", "USDC-FUTURES"}
		for x := range req {
			resp2, err := e.GetAllFuturesTickers(ctx, req[x])
			if err != nil {
				return nil, err
			}
			resp = append(resp, resp2...)
		}
		pairs := make(currency.Pairs, len(resp))
		for x := range resp {
			pair, err := pairFromStringHelper(resp[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs[x] = pair
		}
		return pairs, nil
	case asset.Margin, asset.CrossMargin:
		resp, err := e.GetSupportedCurrencies(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, len(resp))
		var filter int
		for x := range resp {
			if (a == asset.Margin && !resp[x].IsIsolatedBaseBorrowable && !resp[x].IsIsolatedQuoteBorrowable) || (a == asset.CrossMargin && !resp[x].IsCrossBorrowable) || resp[x].Symbol == "ENAUSDT" {
				continue
			}
			pairs[filter] = currency.NewPair(resp[x].BaseCoin, resp[x].QuoteCoin)
			filter++
		}
		return pairs[:filter:filter], nil
	}
	return nil, asset.ErrNotSupported
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assetTypes := e.GetAssetTypes(false)
	for x := range assetTypes {
		pairs, err := e.FetchTradablePairs(ctx, assetTypes[x])
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, assetTypes[x], false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	var tickerPrice *ticker.Price
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		tick, err := e.GetSpotTickerInformation(ctx, p)
		if err != nil {
			return nil, err
		}
		if len(tick) == 0 {
			return nil, common.ErrNoResults
		}
		tickerPrice = &ticker.Price{
			High:         tick[0].High24H.Float64(),
			Low:          tick[0].Low24H.Float64(),
			Bid:          tick[0].BidPrice.Float64(),
			Ask:          tick[0].AskPrice.Float64(),
			Volume:       tick[0].BaseVolume.Float64(),
			QuoteVolume:  tick[0].QuoteVolume.Float64(),
			Open:         tick[0].Open.Float64(),
			Close:        tick[0].LastPrice.Float64(),
			LastUpdated:  tick[0].Timestamp.Time(),
			ExchangeName: e.Name,
			AssetType:    assetType,
			Pair:         p,
		}
	case asset.Futures:
		tick, err := e.GetFuturesTicker(ctx, p, getProductType(p))
		if err != nil {
			return nil, err
		}
		if len(tick) == 0 {
			return nil, common.ErrNoResults
		}
		tickerPrice = &ticker.Price{
			High:         tick[0].High24H.Float64(),
			Low:          tick[0].Low24H.Float64(),
			Bid:          tick[0].BidPrice.Float64(),
			Ask:          tick[0].AskPrice.Float64(),
			Volume:       tick[0].BaseVolume.Float64(),
			QuoteVolume:  tick[0].QuoteVolume.Float64(),
			Open:         tick[0].Open24H.Float64(),
			Close:        tick[0].LastPrice.Float64(),
			IndexPrice:   tick[0].IndexPrice.Float64(),
			LastUpdated:  tick[0].Timestamp.Time(),
			ExchangeName: e.Name,
			AssetType:    assetType,
			Pair:         p,
		}
	case asset.Margin, asset.CrossMargin:
		tick, err := e.GetSpotCandlestickData(ctx, p, formatExchangeKlineIntervalSpot(kline.OneDay), time.Now().Add(-time.Hour*24), time.Now(), 2, false)
		if err != nil {
			return nil, err
		}
		if len(tick) == 0 {
			return nil, common.ErrNoResults
		}
		tickerPrice = &ticker.Price{
			High:         tick[0].High.Float64(),
			Low:          tick[0].Low.Float64(),
			Volume:       tick[0].BaseVolume.Float64(),
			QuoteVolume:  tick[0].QuoteVolume.Float64(),
			Open:         tick[0].Open.Float64(),
			Close:        tick[0].Close.Float64(),
			LastUpdated:  tick[0].Timestamp.Time(),
			ExchangeName: e.Name,
			AssetType:    assetType,
			Pair:         p,
		}
	default:
		return nil, asset.ErrNotSupported
	}
	tickerPrice.Pair = p
	tickerPrice.ExchangeName = e.Name
	tickerPrice.AssetType = assetType
	if err := ticker.ProcessTicker(tickerPrice); err != nil {
		return tickerPrice, err
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot:
		ticks, err := e.GetSpotTickerInformation(ctx, currency.Pair{})
		if err != nil {
			return err
		}
		for x := range ticks {
			p, err := e.MatchSymbolWithAvailablePairs(ticks[x].Symbol, assetType, false)
			if err != nil {
				if errors.Is(err, currency.ErrPairNotFound) && ticks[x].High24H.Float64() == 0 {
					// Screen inactive pairs with no price movement
					continue
				}
				return err
			}
			if err := ticker.ProcessTicker(&ticker.Price{
				High:         ticks[x].High24H.Float64(),
				Low:          ticks[x].Low24H.Float64(),
				Bid:          ticks[x].BidPrice.Float64(),
				Ask:          ticks[x].AskPrice.Float64(),
				Volume:       ticks[x].BaseVolume.Float64(),
				QuoteVolume:  ticks[x].QuoteVolume.Float64(),
				Open:         ticks[x].Open.Float64(),
				Close:        ticks[x].LastPrice.Float64(),
				LastUpdated:  ticks[x].Timestamp.Time(),
				Pair:         p,
				ExchangeName: e.Name,
				AssetType:    assetType,
			}); err != nil {
				return err
			}
		}
	case asset.Futures:
		for i := range prodTypes {
			tick, err := e.GetAllFuturesTickers(ctx, prodTypes[i])
			if err != nil {
				return err
			}
			for x := range tick {
				p, err := e.MatchSymbolWithAvailablePairs(tick[x].Symbol, assetType, false)
				if err != nil {
					return err
				}
				if err := ticker.ProcessTicker(&ticker.Price{
					High:         tick[x].High24H.Float64(),
					Low:          tick[x].Low24H.Float64(),
					Bid:          tick[x].BidPrice.Float64(),
					Ask:          tick[x].AskPrice.Float64(),
					Volume:       tick[x].BaseVolume.Float64(),
					QuoteVolume:  tick[x].QuoteVolume.Float64(),
					Open:         tick[x].Open24H.Float64(),
					Close:        tick[x].LastPrice.Float64(),
					IndexPrice:   tick[x].IndexPrice.Float64(),
					LastUpdated:  tick[x].Timestamp.Time(),
					Pair:         p,
					ExchangeName: e.Name,
					AssetType:    assetType,
				}); err != nil {
					return err
				}
			}
		}
	case asset.Margin, asset.CrossMargin:
		pairs, err := e.GetAvailablePairs(assetType)
		if err != nil {
			return err
		}
		check, err := e.GetSymbolInfo(ctx, currency.Pair{})
		if err != nil {
			return err
		}
		checkSlice := make([]string, len(check))
		var filter int
		for i := range check {
			if (check[i].PricePrecision == 0 && check[i].QuantityPrecision == 0 && check[i].QuotePrecision == 0) || check[i].OpenTime.Time().After(time.Now().Add(time.Hour)) {
				continue
			}
			checkSlice[filter] = check[i].Symbol
			filter++
		}
		checkSlice = checkSlice[:filter:filter]
		for x := range pairs {
			if !slices.Contains(checkSlice, pairs[x].String()) {
				continue
			}
			p, err := e.MatchSymbolWithAvailablePairs(pairs[x].String(), assetType, false)
			if err != nil {
				return err
			}
			if p, err = e.FormatExchangeCurrency(p, assetType); err != nil {
				return err
			}
			resp, err := e.GetSpotCandlestickData(ctx, p, formatExchangeKlineIntervalSpot(kline.OneDay), time.Now().Add(-time.Hour*24), time.Now(), 2, false)
			if err != nil {
				return err
			}
			if len(resp) == 0 {
				return common.ErrNoResults
			}
			if err := ticker.ProcessTicker(&ticker.Price{
				High:         resp[0].High.Float64(),
				Low:          resp[0].Low.Float64(),
				Volume:       resp[0].BaseVolume.Float64(),
				QuoteVolume:  resp[0].QuoteVolume.Float64(),
				Open:         resp[0].Open.Float64(),
				Close:        resp[0].Close.Float64(),
				LastUpdated:  resp[0].Timestamp.Time(),
				Pair:         p,
				ExchangeName: e.Name,
				AssetType:    assetType,
			}); err != nil {
				return err
			}
		}
	default:
		return asset.ErrNotSupported
	}
	return nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	pair, err := e.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	var bids, asks orderbook.Levels
	var ts time.Time
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		orderbookNew, err := e.GetOrderbookDepth(ctx, pair, "", 150)
		if err != nil {
			return nil, err
		}
		bids = orderbookNew.Bids.Levels()
		asks = orderbookNew.Asks.Levels()
		ts = orderbookNew.Timestamp.Time()
	case asset.Futures:
		orderbookNew, err := e.GetFuturesMergeDepth(ctx, pair, getProductType(pair), "", "max")
		if err != nil {
			return nil, err
		}

		bids = orderbookNew.Bids.Levels()
		asks = orderbookNew.Asks.Levels()
		ts = orderbookNew.Timestamp.Time()
	default:
		return nil, asset.ErrNotSupported
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
		MaxDepth:          150,
		Bids:              bids,
		Asks:              asks,
		LastUpdated:       ts,
	}
	if err := book.Process(); err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, pair, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	info, err := e.GetAccountInfo(ctx)
	if err != nil {
		return nil, err
	}

	var subAccts accounts.SubAccounts
	switch a {
	case asset.Spot:
		resp, err := e.GetAccountAssets(ctx, currency.EMPTYCODE, "")
		if err != nil {
			return nil, err
		}
		subAcc := accounts.NewSubAccount(a, strconv.FormatUint(info.UserID, 10))
		for x := range resp {
			hold := resp[x].Frozen.Float64() + resp[x].Locked.Float64() + resp[x].LimitAvailable.Float64()
			subAcc.Balances.Set(resp[x].Coin, accounts.Balance{
				Total: resp[x].Available.Float64() + hold,
				Free:  resp[x].Available.Float64(),
				Hold:  hold,
			})
		}
		subAccts = append(subAccts, subAcc)
	case asset.Futures:
		for i := range prodTypes {
			resp, err := e.GetAllFuturesAccounts(ctx, prodTypes[i])
			if err != nil {
				return nil, err
			}
			subAcc := accounts.NewSubAccount(a, fmt.Sprintf("%s-%s", strconv.FormatUint(info.UserID, 10), prodTypes[i]))
			for x := range resp {
				subAcc.Balances.Set(resp[x].MarginCoin, accounts.Balance{
					Total: resp[x].Locked.Float64() + resp[x].Available.Float64(),
					Free:  resp[x].Available.Float64(),
					Hold:  resp[x].Locked.Float64(),
				})
			}
			subAccts = append(subAccts, subAcc)
		}
	case asset.Margin:
		resp, err := e.GetIsolatedAccountAssets(ctx, currency.Pair{})
		if err != nil {
			return nil, err
		}
		subAcc := accounts.NewSubAccount(a, strconv.FormatUint(info.UserID, 10))
		for x := range resp {
			subAcc.Balances.Set(resp[x].Coin, accounts.Balance{
				Total:    resp[x].TotalAmount.Float64(),
				Free:     resp[x].Available.Float64(),
				Hold:     resp[x].Frozen.Float64(),
				Borrowed: resp[x].Borrow.Float64(),
			})
		}
		subAccts = append(subAccts, subAcc)
	case asset.CrossMargin:
		resp, err := e.GetCrossAccountAssets(ctx, currency.Code{})
		if err != nil {
			return nil, err
		}
		subAcc := accounts.NewSubAccount(a, strconv.FormatUint(info.UserID, 10))
		for x := range resp {
			subAcc.Balances.Set(resp[x].Coin, accounts.Balance{
				Total:    resp[x].TotalAmount.Float64(),
				Free:     resp[x].Available.Float64(),
				Hold:     resp[x].Frozen.Float64(),
				Borrowed: resp[x].Borrow.Float64(),
			})
		}
		subAccts = append(subAccts, subAcc)
	default:
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	// This exchange only allows requests covering the last 90 days
	resp, err := e.withdrawalHistGrabber(ctx, currency.Code{})
	if err != nil {
		return nil, err
	}
	funHist := make([]exchange.FundingHistory, len(resp))
	for x := range resp {
		funHist[x] = exchange.FundingHistory{
			ExchangeName:      e.Name,
			Status:            resp[x].Status,
			TransferID:        strconv.FormatUint(resp[x].OrderID, 10),
			Timestamp:         resp[x].CreationTime.Time(),
			Currency:          resp[x].Coin.String(),
			Amount:            resp[x].Size.Float64(),
			TransferType:      "Withdrawal",
			CryptoToAddress:   resp[x].ToAddress,
			CryptoFromAddress: resp[x].FromAddress,
			CryptoChain:       resp[x].Chain,
		}
		if resp[x].Destination == "on_chain" {
			funHist[x].CryptoTxID = strconv.FormatUint(resp[x].TradeID, 10)
		}
	}
	var pagination uint64
	pagination = 0
	for {
		resp, err := e.GetDepositRecords(ctx, currency.Code{}, 0, pagination, 100, time.Now().Add(-time.Hour*24*90), time.Now())
		if err != nil {
			return nil, err
		}
		// Not sure that this is the right end to use for pagination
		if len(resp) == 0 || pagination == resp[len(resp)-1].OrderID {
			break
		}
		pagination = resp[len(resp)-1].OrderID
		tempHist := make([]exchange.FundingHistory, len(resp))
		for x := range resp {
			tempHist[x] = exchange.FundingHistory{
				ExchangeName:      e.Name,
				Status:            resp[x].Status,
				TransferID:        strconv.FormatUint(resp[x].OrderID, 10),
				Timestamp:         resp[x].CreationTime.Time(),
				Currency:          resp[x].Coin.String(),
				Amount:            resp[x].Size.Float64(),
				TransferType:      "Deposit",
				CryptoToAddress:   resp[x].ToAddress,
				CryptoFromAddress: resp[x].FromAddress,
				CryptoChain:       resp[x].Chain,
			}
			if resp[x].Destination == "on_chain" {
				tempHist[x].CryptoTxID = strconv.FormatUint(resp[x].TradeID, 10)
			}
		}
		funHist = slices.Concat(funHist, tempHist)
	}
	return funHist, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	// This exchange only allows requests covering the last 90 days
	resp, err := e.withdrawalHistGrabber(ctx, c)
	if err != nil {
		return nil, err
	}
	funHist := make([]exchange.WithdrawalHistory, len(resp))
	for x := range resp {
		funHist[x] = exchange.WithdrawalHistory{
			Status:          resp[x].Status,
			TransferID:      strconv.FormatUint(resp[x].OrderID, 10),
			Timestamp:       resp[x].CreationTime.Time(),
			Currency:        resp[x].Coin.String(),
			Amount:          resp[x].Size.Float64(),
			TransferType:    "Withdrawal",
			CryptoToAddress: resp[x].ToAddress,
			CryptoChain:     resp[x].Chain,
		}
		if resp[x].Destination == "on_chain" {
			funHist[x].CryptoTxID = strconv.FormatUint(resp[x].TradeID, 10)
		}
	}
	return funHist, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		resp, err := e.GetRecentSpotFills(ctx, p, 500)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp))
		for x := range resp {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp[x].TradeID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp[x].Side),
				Price:        resp[x].Price.Float64(),
				Amount:       resp[x].Size.Float64(),
				Timestamp:    resp[x].Timestamp.Time(),
			}
		}
		return trades, nil
	case asset.Futures:
		resp, err := e.GetRecentFuturesFills(ctx, p, getProductType(p), 100)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp))
		for x := range resp {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp[x].TradeID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp[x].Side),
				Price:        resp[x].Price.Float64(),
				Amount:       resp[x].Size.Float64(),
				Timestamp:    resp[x].Timestamp.Time(),
			}
		}
		return trades, nil
	}
	return nil, asset.ErrNotSupported
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	// This exchange only allows requests covering the last 7 days
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		resp, err := e.GetSpotMarketTrades(ctx, p, timestampStart, timestampEnd, 1000, 0)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp))
		for x := range resp {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp[x].TradeID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp[x].Side),
				Price:        resp[x].Price.Float64(),
				Amount:       resp[x].Size.Float64(),
				Timestamp:    resp[x].Timestamp.Time(),
			}
		}
		return trades, nil
	case asset.Futures:
		resp, err := e.GetFuturesMarketTrades(ctx, p, getProductType(p), 1000, 0, timestampStart, timestampEnd)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp))
		for x := range resp {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp[x].TradeID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp[x].Side),
				Price:        resp[x].Price.Float64(),
				Amount:       resp[x].Size.Float64(),
				Timestamp:    resp[x].Timestamp.Time(),
			}
		}
		return trades, nil
	}
	return nil, asset.ErrNotSupported
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	resp, err := e.GetTime(ctx)
	return resp.ServerTime.Time(), err
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	var IDs *OrderIDStruct
	strategy, err := strategyTruthTable(s.TimeInForce)
	if err != nil {
		return nil, err
	}
	cID, err := uuid.DefaultGenerator.NewV4()
	if err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Spot:
		IDs, err = e.PlaceSpotOrder(ctx, &PlaceSingleSpotOrderParams{Pair: s.Pair, Side: s.Side.String(), OrderType: s.Type.Lower(), Strategy: strategy, ClientOrderID: cID.String(), Price: s.Price, Amount: s.Amount, TriggerPrice: s.TriggerPrice}, false)
	case asset.Futures:
		IDs, err = e.PlaceFuturesOrder(ctx, &PlaceSingleFuturesOrderParams{Pair: s.Pair, ProductType: getProductType(s.Pair), MarginMode: marginStringer(s.MarginType), Side: sideEncoder(s.Side, false), OrderType: s.Type.Lower(), Strategy: strategy, ClientOrderID: cID.String(), MarginCoin: s.Pair.Quote, Amount: s.Amount, Price: s.Price, ReduceOnly: YesNoBool(s.ReduceOnly)}, false)
	case asset.Margin, asset.CrossMargin:
		loanType := "normal"
		if s.AutoBorrow {
			loanType = "autoLoan"
		}
		if s.AssetType == asset.Margin {
			IDs, err = e.PlaceIsolatedOrder(ctx, &PlaceMarginOrderParams{
				Pair:          s.Pair,
				OrderType:     s.Type.Lower(),
				LoanType:      loanType,
				Strategy:      strategy,
				ClientOrderID: cID.String(),
				Side:          s.Side.String(),
				Price:         s.Price,
				BaseAmount:    s.Amount,
				QuoteAmount:   s.QuoteAmount,
			})
		} else {
			IDs, err = e.PlaceCrossOrder(ctx, &PlaceMarginOrderParams{
				Pair:          s.Pair,
				OrderType:     s.Type.Lower(),
				LoanType:      loanType,
				Strategy:      strategy,
				ClientOrderID: cID.String(),
				Side:          s.Side.String(),
				Price:         s.Price,
				BaseAmount:    s.Amount,
				QuoteAmount:   s.QuoteAmount,
			})
		}
	default:
		return nil, asset.ErrNotSupported
	}
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(strconv.FormatUint(uint64(IDs.OrderID), 10))
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = IDs.ClientOrderID
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to market conversion
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	var IDs *OrderIDStruct
	originalID, err := strconv.ParseUint(action.OrderID, 10, 64)
	if err != nil {
		return nil, err
	}
	switch action.AssetType {
	case asset.Spot:
		IDs, err = e.ModifyPlanSpotOrder(ctx, originalID, action.ClientOrderID, action.Type.String(), action.TriggerPrice, action.Price, action.Amount)
	case asset.Futures:
		var cID uuid.UUID
		if cID, err = uuid.DefaultGenerator.NewV4(); err != nil {
			return nil, err
		}
		IDs, err = e.ModifyFuturesOrder(ctx, &ModifyFuturesOrderParams{
			OrderID:          originalID,
			ClientOrderID:    action.ClientOrderID,
			ProductType:      getProductType(action.Pair),
			NewClientOrderID: cID.String(),
			Pair:             action.Pair,
			NewAmount:        action.Amount,
			NewPrice:         action.Price,
		})
	default:
		return nil, asset.ErrNotSupported
	}
	if err != nil {
		return nil, err
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.OrderID = strconv.FormatUint(uint64(IDs.OrderID), 10)
	resp.ClientOrderID = IDs.ClientOrderID
	return resp, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	originalID, err := strconv.ParseUint(ord.OrderID, 10, 64)
	if err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Spot:
		_, err = e.CancelSpotOrderByID(ctx, ord.Pair, ord.ClientOrderID, "", originalID)
	case asset.Futures:
		_, err = e.CancelFuturesOrder(ctx, ord.Pair, getProductType(ord.Pair), ord.ClientOrderID, ord.Pair.Quote, originalID)
	case asset.Margin:
		// Consider warning the user if they're trying to input both the client order ID and the order ID
		_, err = e.CancelIsolatedOrder(ctx, ord.Pair, "", originalID)
	case asset.CrossMargin:
		// Consider warning the user if they're trying to input both the client order ID and the order ID
		_, err = e.CancelCrossOrder(ctx, ord.Pair, "", originalID)
	default:
		return asset.ErrNotSupported
	}
	if err != nil {
		return err
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (*order.CancelBatchResponse, error) {
	batchByAsset := make(map[asset.Item][]order.Cancel)
	for i := range orders {
		batchByAsset[orders[i].AssetType] = append(batchByAsset[orders[i].AssetType], orders[i])
	}
	resp := &order.CancelBatchResponse{}
	resp.Status = make(map[string]string)
	for assetType, batch := range batchByAsset {
		var status *BatchOrderResp
		batchByPair, err := pairBatcher(batch)
		if err != nil {
			return resp, err
		}
		for pair, batch := range batchByPair {
			switch assetType {
			case asset.Spot:
				// This no longer needs to be batched by pair, refactor if many others get similar changes
				batchConv := make([]CancelSpotOrderParams, len(batch))
				for i := range batch {
					batchConv[i] = CancelSpotOrderParams{
						OrderID:       uint64(batch[i].OrderID),
						ClientOrderID: batch[i].ClientOrderID,
					}
				}
				status, err = e.BatchCancelOrders(ctx, pair, false, batchConv)
			case asset.Futures:
				status, err = e.BatchCancelFuturesOrders(ctx, batch, pair, getProductType(pair), pair.Quote)
			case asset.Margin:
				status, err = e.BatchCancelIsolatedOrders(ctx, pair, batch)
			case asset.CrossMargin:
				status, err = e.BatchCancelCrossOrders(ctx, pair, batch)
			default:
				return resp, asset.ErrNotSupported
			}
			if err != nil {
				return resp, err
			}
			// The earlier error returns do so with resp, as earlier calls can leave valid status data in that
			addStatuses(status, resp)
		}
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	if err := orderCancellation.Validate(); err != nil {
		return resp, err
	}
	switch orderCancellation.AssetType {
	case asset.Spot:
		if _, err := e.CancelOrdersBySymbol(ctx, orderCancellation.Pair); err != nil {
			return resp, err
		}
	case asset.Futures:
		resp2, err := e.CancelAllFuturesOrders(ctx, orderCancellation.Pair, getProductType(orderCancellation.Pair), orderCancellation.Pair.Quote, time.Second*60)
		if err != nil {
			return resp, err
		}
		resp.Status = make(map[string]string)
		for i := range resp2.SuccessList {
			resp.Status[resp2.SuccessList[i].ClientOrderID] = "success"
			resp.Status[strconv.FormatUint(uint64(resp2.SuccessList[i].OrderID), 10)] = "success"
		}
		for i := range resp2.FailureList {
			resp.Status[resp2.FailureList[i].ClientOrderID] = resp2.FailureList[i].ErrorMessage
			resp.Status[strconv.FormatUint(uint64(resp2.FailureList[i].OrderID), 10)] = resp2.FailureList[i].ErrorMessage
		}
	default:
		return resp, asset.ErrNotSupported
	}
	return resp, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	ordID, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil {
		return nil, err
	}
	resp := &order.Detail{
		Exchange:  e.Name,
		Pair:      pair,
		AssetType: assetType,
		OrderID:   orderID,
	}
	switch assetType {
	case asset.Spot:
		ordInfo, err := e.GetSpotOrderDetails(ctx, ordID, "", time.Minute)
		if err != nil {
			return nil, err
		}
		if len(ordInfo) == 0 {
			return nil, order.ErrOrderNotFound
		}
		resp.AccountID = strconv.FormatUint(ordInfo[0].UserID, 10)
		resp.ClientOrderID = ordInfo[0].ClientOrderID
		resp.Price = ordInfo[0].Price
		resp.Amount = ordInfo[0].Size
		resp.Type = typeDecoder(ordInfo[0].OrderType)
		resp.Side = sideDecoder(ordInfo[0].Side)
		resp.Status = statusDecoder(ordInfo[0].Status)
		resp.AverageExecutedPrice = ordInfo[0].PriceAverage
		resp.QuoteAmount = ordInfo[0].QuoteVolume
		resp.Date = ordInfo[0].CreationTime.Time()
		resp.LastUpdated = ordInfo[0].UpdateTime.Time()
		for s, f := range ordInfo[0].FeeDetail {
			if s != "newFees" {
				resp.FeeAsset = f.FeeCoinCode
				resp.Fee = f.TotalFee
				break
			}
		}
		fillInfo, err := e.GetSpotFills(ctx, pair, time.Time{}, time.Time{}, 0, 0, ordID)
		if err != nil {
			return nil, err
		}
		resp.Trades = make([]order.TradeHistory, len(fillInfo))
		for x := range fillInfo {
			resp.Trades[x] = order.TradeHistory{
				TID:       strconv.FormatUint(fillInfo[x].TradeID, 10),
				Type:      typeDecoder(fillInfo[x].OrderType),
				Side:      sideDecoder(fillInfo[x].Side),
				Price:     fillInfo[x].PriceAverage.Float64(),
				Amount:    fillInfo[x].Size.Float64(),
				Fee:       fillInfo[x].FeeDetail.TotalFee.Float64(),
				FeeAsset:  fillInfo[x].FeeDetail.FeeCoin.String(),
				Timestamp: fillInfo[x].CreationTime.Time(),
			}
		}
	case asset.Futures:
		ordInfo, err := e.GetFuturesOrderDetails(ctx, pair, getProductType(pair), "", ordID)
		if err != nil {
			return nil, err
		}
		resp.Amount = ordInfo.Size.Float64()
		resp.ClientOrderID = ordInfo.ClientOrderID
		resp.AverageExecutedPrice = ordInfo.PriceAverage.Float64()
		resp.Fee = ordInfo.Fee.Float64()
		resp.Price = ordInfo.Price.Float64()
		resp.Status = statusDecoder(ordInfo.State)
		resp.Side = sideDecoder(ordInfo.Side)
		resp.TimeInForce = strategyDecoder(ordInfo.Force)
		resp.SettlementCurrency = ordInfo.MarginCoin
		resp.LimitPriceUpper = ordInfo.PresetStopSurplusPrice.Float64()
		resp.LimitPriceLower = ordInfo.PresetStopLossPrice.Float64()
		resp.QuoteAmount = ordInfo.QuoteVolume.Float64()
		resp.Type = typeDecoder(ordInfo.OrderType)
		resp.Leverage = ordInfo.Leverage.Float64()
		resp.MarginType = marginDecoder(ordInfo.MarginMode)
		resp.ReduceOnly = bool(ordInfo.ReduceOnly)
		resp.Date = ordInfo.CreationTime.Time()
		resp.LastUpdated = ordInfo.UpdateTime.Time()
		fillInfo, err := e.GetFuturesFills(ctx, ordID, 0, 100, pair, getProductType(pair), time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		resp.Trades = make([]order.TradeHistory, len(fillInfo.FillList))
		for x := range fillInfo.FillList {
			resp.Trades[x] = order.TradeHistory{
				TID:       strconv.FormatUint(fillInfo.FillList[x].TradeID, 10),
				Price:     fillInfo.FillList[x].Price.Float64(),
				Amount:    fillInfo.FillList[x].BaseVolume.Float64(),
				Side:      sideDecoder(fillInfo.FillList[x].Side),
				Timestamp: fillInfo.FillList[x].CreationTime.Time(),
			}
			for i := range fillInfo.FillList[x].FeeDetail {
				resp.Trades[x].Fee += fillInfo.FillList[x].FeeDetail[i].TotalFee.Float64()
				resp.Trades[x].FeeAsset = fillInfo.FillList[x].FeeDetail[i].FeeCoin.String()
			}
			if fillInfo.FillList[x].TradeScope == "maker" {
				resp.Trades[x].IsMaker = true
			}
		}
	case asset.Margin, asset.CrossMargin:
		var ordInfo *MarginOrders
		var fillInfo *MarginOrderFills
		if assetType == asset.Margin {
			if ordInfo, err = e.GetIsolatedOpenOrders(ctx, pair, "", ordID, 2, 0, time.Now().Add(-time.Hour*24*90), time.Now()); err != nil {
				return nil, err
			}
			fillInfo, err = e.GetIsolatedOrderFills(ctx, pair, ordID, 0, 500, time.Now().Add(-time.Hour*24*90), time.Now())
		} else {
			if ordInfo, err = e.GetCrossOpenOrders(ctx, pair, "", ordID, 2, 0, time.Now().Add(-time.Hour*24*90), time.Now()); err != nil {
				return nil, err
			}
			fillInfo, err = e.GetCrossOrderFills(ctx, pair, ordID, 0, 500, time.Now().Add(-time.Hour*24*90), time.Now())
		}
		if err != nil {
			return nil, err
		}
		if len(ordInfo.OrderList) == 0 {
			return nil, order.ErrOrderNotFound
		}
		resp.Type = typeDecoder(ordInfo.OrderList[0].OrderType)
		resp.ClientOrderID = ordInfo.OrderList[0].ClientOrderID
		resp.Price = ordInfo.OrderList[0].Price.Float64()
		resp.Side = sideDecoder(ordInfo.OrderList[0].Side)
		resp.Status = statusDecoder(ordInfo.OrderList[0].Status)
		resp.Amount = ordInfo.OrderList[0].Size.Float64()
		resp.QuoteAmount = ordInfo.OrderList[0].QuoteSize.Float64()
		resp.TimeInForce = strategyDecoder(ordInfo.OrderList[0].Force)
		resp.Date = ordInfo.OrderList[0].CreationTime.Time()
		resp.LastUpdated = ordInfo.OrderList[0].UpdateTime.Time()
		resp.Trades = make([]order.TradeHistory, len(fillInfo.Fills))
		for x := range fillInfo.Fills {
			resp.Trades[x] = order.TradeHistory{
				TID:       strconv.FormatUint(fillInfo.Fills[x].TradeID, 10),
				Type:      typeDecoder(fillInfo.Fills[x].OrderType),
				Side:      sideDecoder(fillInfo.Fills[x].Side),
				Price:     fillInfo.Fills[x].PriceAverage.Float64(),
				Amount:    fillInfo.Fills[x].Size.Float64(),
				Timestamp: fillInfo.Fills[x].CreationTime.Time(),
				Fee:       fillInfo.Fills[x].FeeDetail.TotalFee.Float64(),
				FeeAsset:  fillInfo.Fills[x].FeeDetail.FeeCoin.String(),
			}
		}
	default:
		return nil, asset.ErrNotSupported
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	resp, err := e.GetDepositAddressForCurrency(ctx, c, chain, 0)
	if err != nil {
		return nil, err
	}
	add := &deposit.Address{
		Address: resp.Address,
		Chain:   resp.Chain,
		Tag:     resp.Tag,
	}
	return add, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.WithdrawFunds(ctx, &WithdrawFundsParams{Cur: withdrawRequest.Currency, TransferType: "on_chain", Address: withdrawRequest.Crypto.Address, Chain: withdrawRequest.Crypto.Chain, Tag: withdrawRequest.Crypto.AddressTag, Note: withdrawRequest.Description, Amount: withdrawRequest.Amount})
	if err != nil {
		return nil, err
	}
	ret := &withdraw.ExchangeResponse{
		ID: strconv.FormatUint(uint64(resp.OrderID), 10),
	}
	return ret, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	for x := range getOrdersRequest.Pairs {
		if getOrdersRequest.Pairs[x], err = e.FormatExchangeCurrency(getOrdersRequest.Pairs[x], getOrdersRequest.AssetType); err != nil {
			return nil, err
		}
	}
	if len(getOrdersRequest.Pairs) == 0 {
		getOrdersRequest.Pairs = append(getOrdersRequest.Pairs, currency.Pair{})
	}
	var resp order.FilteredOrders
	for x := range getOrdersRequest.Pairs {
		switch getOrdersRequest.AssetType {
		case asset.Spot:
			var pagination uint64
			for {
				genOrds, err := e.GetUnfilledOrders(ctx, getOrdersRequest.Pairs[x], "", time.Time{}, time.Time{}, 100, pagination, 0, time.Minute)
				if err != nil {
					return nil, err
				}
				if len(genOrds) == 0 ||
					pagination == uint64(genOrds[len(genOrds)-1].OrderID) {
					break
				}
				pagination = uint64(genOrds[len(genOrds)-1].OrderID)
				tempOrds := make([]order.Detail, len(genOrds))
				for i := range genOrds {
					tempOrds[i] = order.Detail{
						Exchange:             e.Name,
						AssetType:            asset.Spot,
						AccountID:            strconv.FormatUint(genOrds[i].UserID, 10),
						OrderID:              strconv.FormatUint(uint64(genOrds[i].OrderID), 10),
						ClientOrderID:        genOrds[i].ClientOrderID,
						AverageExecutedPrice: genOrds[i].PriceAverage.Float64(),
						Amount:               genOrds[i].Size.Float64(),
						Type:                 typeDecoder(genOrds[i].OrderType),
						Side:                 sideDecoder(genOrds[i].Side),
						Status:               statusDecoder(genOrds[i].Status),
						Price:                genOrds[i].BasePrice.Float64(),
						QuoteAmount:          genOrds[i].QuoteVolume.Float64(),
						Date:                 genOrds[i].CreationTime.Time(),
						LastUpdated:          genOrds[i].UpdateTime.Time(),
					}
					if !getOrdersRequest.Pairs[x].IsEmpty() {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						if tempOrds[i].Pair, err = pairFromStringHelper(genOrds[i].Symbol); err != nil {
							return nil, err
						}
					}
				}
				resp = append(resp, tempOrds...)
			}
			// TODO: Return spot plan orders once GetOrderInfo has more parameters to handle that
		case asset.Futures:
			if !getOrdersRequest.Pairs[x].IsEmpty() {
				if resp, err = e.activeFuturesOrderHelper(ctx, getProductType(getOrdersRequest.Pairs[x]), getOrdersRequest.Pairs[x], resp); err != nil {
					return nil, err
				}
			} else {
				for y := range prodTypes {
					if resp, err = e.activeFuturesOrderHelper(ctx, prodTypes[y], currency.Pair{}, resp); err != nil {
						return nil, err
					}
				}
			}
		case asset.Margin, asset.CrossMargin:
			var pagination uint64
			var genOrds *MarginOrders
			for {
				if getOrdersRequest.AssetType == asset.Margin {
					genOrds, err = e.GetIsolatedOpenOrders(ctx, getOrdersRequest.Pairs[x], "", 0, 500, pagination, time.Now().Add(-time.Hour*24*90), time.Time{})
				} else {
					genOrds, err = e.GetCrossOpenOrders(ctx, getOrdersRequest.Pairs[x], "", 0, 500, pagination, time.Now().Add(-time.Hour*24*90), time.Time{})
				}
				if err != nil {
					return nil, err
				}
				if genOrds == nil || len(genOrds.OrderList) == 0 || pagination == uint64(genOrds.MaximumID) {
					break
				}
				pagination = uint64(genOrds.MaximumID)
				tempOrds := make([]order.Detail, len(genOrds.OrderList))
				for i := range genOrds.OrderList {
					tempOrds[i] = order.Detail{
						Exchange:      e.Name,
						AssetType:     getOrdersRequest.AssetType,
						OrderID:       strconv.FormatUint(genOrds.OrderList[i].OrderID, 10),
						Type:          typeDecoder(genOrds.OrderList[i].OrderType),
						ClientOrderID: genOrds.OrderList[i].ClientOrderID,
						Price:         genOrds.OrderList[i].Price.Float64(),
						Side:          sideDecoder(genOrds.OrderList[i].Side),
						Status:        statusDecoder(genOrds.OrderList[i].Status),
						QuoteAmount:   genOrds.OrderList[i].QuoteSize.Float64(),
						Amount:        genOrds.OrderList[i].Size.Float64(),
						Date:          genOrds.OrderList[i].CreationTime.Time(),
						LastUpdated:   genOrds.OrderList[i].UpdateTime.Time(),
						TimeInForce:   strategyDecoder(genOrds.OrderList[i].Force),
					}
					if !getOrdersRequest.Pairs[x].IsEmpty() {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						if tempOrds[i].Pair, err = pairFromStringHelper(genOrds.OrderList[i].Symbol); err != nil {
							return nil, err
						}
					}
				}
				resp = append(resp, tempOrds...)
			}
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information. Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	for x := range getOrdersRequest.Pairs {
		if getOrdersRequest.Pairs[x], err = e.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Spot); err != nil {
			return nil, err
		}
	}
	if len(getOrdersRequest.Pairs) == 0 {
		getOrdersRequest.Pairs = append(getOrdersRequest.Pairs, currency.Pair{})
	}
	var resp order.FilteredOrders
	for x := range getOrdersRequest.Pairs {
		switch getOrdersRequest.AssetType {
		case asset.Spot:
			fillMap := make(map[uint64][]order.TradeHistory)
			var pagination uint64
			if !getOrdersRequest.Pairs[x].IsEmpty() {
				if err := e.spotFillsHelper(ctx, getOrdersRequest.Pairs[x], fillMap); err != nil {
					return nil, err
				}
				if resp, err = e.spotHistoricPlanOrdersHelper(ctx, getOrdersRequest.Pairs[x], resp, fillMap); err != nil {
					return nil, err
				}
			} else {
				newPairs, err := e.FetchTradablePairs(ctx, asset.Spot)
				if err != nil {
					return nil, err
				}
				for y := range newPairs {
					callStr, err := e.FormatExchangeCurrency(newPairs[y], asset.Spot)
					if err != nil {
						return nil, err
					}
					if err = e.spotFillsHelper(ctx, callStr, fillMap); err != nil {
						return nil, err
					}
					if resp, err = e.spotHistoricPlanOrdersHelper(ctx, callStr, resp, fillMap); err != nil {
						return nil, err
					}
				}
			}
			for {
				genOrds, err := e.GetHistoricalSpotOrders(ctx, getOrdersRequest.Pairs[x], time.Time{}, time.Time{}, 100, pagination, 0, "", time.Minute)
				if err != nil {
					return nil, err
				}
				if len(genOrds) == 0 || pagination == uint64(genOrds[len(genOrds)-1].OrderID) {
					break
				}
				pagination = uint64(genOrds[len(genOrds)-1].OrderID)
				tempOrds := make([]order.Detail, len(genOrds))
				for i := range genOrds {
					tempOrds[i] = order.Detail{
						Exchange:             e.Name,
						AssetType:            asset.Spot,
						AccountID:            strconv.FormatUint(genOrds[i].UserID, 10),
						OrderID:              strconv.FormatUint(uint64(genOrds[i].OrderID), 10),
						ClientOrderID:        genOrds[i].ClientOrderID,
						Price:                genOrds[i].Price,
						Amount:               genOrds[i].Size,
						Type:                 typeDecoder(genOrds[i].OrderType),
						Side:                 sideDecoder(genOrds[i].Side),
						Status:               statusDecoder(genOrds[i].Status),
						AverageExecutedPrice: genOrds[i].PriceAverage,
						QuoteAmount:          genOrds[i].QuoteVolume,
						Date:                 genOrds[i].CreationTime.Time(),
						LastUpdated:          genOrds[i].UpdateTime.Time(),
					}
					if !getOrdersRequest.Pairs[x].IsEmpty() {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						if tempOrds[i].Pair, err = pairFromStringHelper(genOrds[i].Symbol); err != nil {
							return nil, err
						}
					}
					for y := range genOrds[i].FeeDetail {
						tempOrds[i].Fee += genOrds[i].FeeDetail[y].TotalFee
						tempOrds[i].FeeAsset = genOrds[i].FeeDetail[y].FeeCoinCode
					}
					if len(fillMap[uint64(genOrds[i].OrderID)]) > 0 {
						tempOrds[i].Trades = fillMap[uint64(genOrds[i].OrderID)]
					}
				}
				resp = append(resp, tempOrds...)
			}
		case asset.Futures:
			if !getOrdersRequest.Pairs[x].IsEmpty() {
				if resp, err = e.historicalFuturesOrderHelper(ctx, getProductType(getOrdersRequest.Pairs[x]), getOrdersRequest.Pairs[x], resp); err != nil {
					return nil, err
				}
			} else {
				for y := range prodTypes {
					if resp, err = e.historicalFuturesOrderHelper(ctx, prodTypes[y], currency.Pair{}, resp); err != nil {
						return nil, err
					}
				}
			}
		case asset.Margin, asset.CrossMargin:
			var pagination uint64
			var genFills *MarginOrderFills
			fillMap := make(map[uint64][]order.TradeHistory)
			for {
				if getOrdersRequest.AssetType == asset.Margin {
					genFills, err = e.GetIsolatedOrderFills(ctx, getOrdersRequest.Pairs[x], 0, pagination, 500, time.Now().Add(-time.Hour*24*90), time.Now())
				} else {
					genFills, err = e.GetCrossOrderFills(ctx, getOrdersRequest.Pairs[x], 0, pagination, 500, time.Now().Add(-time.Hour*24*90), time.Now())
				}
				if err != nil {
					return nil, err
				}
				if genFills == nil || len(genFills.Fills) == 0 || pagination == uint64(genFills.MaximumID) {
					break
				}
				pagination = uint64(genFills.MaximumID)
				for i := range genFills.Fills {
					fillMap[genFills.Fills[i].TradeID] = append(fillMap[genFills.Fills[i].TradeID], order.TradeHistory{
						TID:       strconv.FormatUint(genFills.Fills[i].TradeID, 10),
						Type:      typeDecoder(genFills.Fills[i].OrderType),
						Side:      sideDecoder(genFills.Fills[i].Side),
						Price:     genFills.Fills[i].PriceAverage.Float64(),
						Amount:    genFills.Fills[i].Size.Float64(),
						Timestamp: genFills.Fills[i].CreationTime.Time(),
						Fee:       genFills.Fills[i].FeeDetail.TotalFee.Float64(),
						FeeAsset:  genFills.Fills[i].FeeDetail.FeeCoin.String(),
					})
				}
			}
			pagination = 0
			var genOrds *MarginOrders
			for {
				if getOrdersRequest.AssetType == asset.Margin {
					genOrds, err = e.GetIsolatedHistoricalOrders(ctx, getOrdersRequest.Pairs[x], "", "", 0, 500, pagination, time.Now().Add(-time.Hour*24*90), time.Time{})
				} else {
					genOrds, err = e.GetCrossHistoricalOrders(ctx, getOrdersRequest.Pairs[x], "", "", 0, 500, pagination, time.Now().Add(-time.Hour*24*90), time.Time{})
				}
				if err != nil {
					return nil, err
				}
				if genOrds == nil || len(genOrds.OrderList) == 0 || pagination == uint64(genOrds.MaximumID) {
					break
				}
				pagination = uint64(genOrds.MaximumID)
				tempOrds := make([]order.Detail, len(genOrds.OrderList))
				for i := range genOrds.OrderList {
					tempOrds[i] = order.Detail{
						Exchange:             e.Name,
						AssetType:            getOrdersRequest.AssetType,
						OrderID:              strconv.FormatUint(genOrds.OrderList[i].OrderID, 10),
						Type:                 typeDecoder(genOrds.OrderList[i].OrderType),
						ClientOrderID:        genOrds.OrderList[i].ClientOrderID,
						Price:                genOrds.OrderList[i].Price.Float64(),
						Side:                 sideDecoder(genOrds.OrderList[i].Side),
						Status:               statusDecoder(genOrds.OrderList[i].Status),
						Amount:               genOrds.OrderList[i].Size.Float64(),
						QuoteAmount:          genOrds.OrderList[i].QuoteSize.Float64(),
						AverageExecutedPrice: genOrds.OrderList[i].PriceAverage.Float64(),
						Date:                 genOrds.OrderList[i].CreationTime.Time(),
						LastUpdated:          genOrds.OrderList[i].UpdateTime.Time(),
						TimeInForce:          strategyDecoder(genOrds.OrderList[i].Force),
					}
					if !getOrdersRequest.Pairs[x].IsEmpty() {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						if tempOrds[i].Pair, err = pairFromStringHelper(genOrds.OrderList[i].Symbol); err != nil {
							return nil, err
						}
					}
					if len(fillMap[genOrds.OrderList[i].OrderID]) > 0 {
						tempOrds[i].Trades = fillMap[genOrds.OrderList[i].OrderID]
					}
				}
				resp = append(resp, tempOrds...)
			}
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	fee, err := e.GetTradeRate(ctx, feeBuilder.Pair, "spot")
	if err != nil {
		return 0, err
	}
	if feeBuilder.IsMaker {
		return fee.MakerFeeRate.Float64() * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
	}
	return fee.TakerFeeRate.Float64() * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
}

// ValidateAPICredentials validates current credentials used for wrapper
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	var resp []kline.Candle
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		cndl, err := e.GetSpotCandlestickData(ctx, req.RequestFormatted, formatExchangeKlineIntervalSpot(req.ExchangeInterval), req.Start, req.End, 200, true)
		if err != nil {
			return nil, err
		}
		resp = make([]kline.Candle, len(cndl))
		for i := range cndl {
			resp[i] = kline.Candle{
				Time:   cndl[i].Timestamp.Time(),
				Low:    cndl[i].Low.Float64(),
				High:   cndl[i].High.Float64(),
				Open:   cndl[i].Open.Float64(),
				Close:  cndl[i].Close.Float64(),
				Volume: cndl[i].BaseVolume.Float64(),
			}
		}
	case asset.Futures:
		cndl, err := e.GetFuturesCandlestickData(ctx, req.RequestFormatted, getProductType(pair), formatExchangeKlineIntervalFutures(req.ExchangeInterval), "", req.Start, req.End, 200, CallModeHistory)
		if err != nil {
			return nil, err
		}
		resp = make([]kline.Candle, len(cndl))
		for i := range cndl {
			resp[i] = kline.Candle{
				Time:   cndl[i].Timestamp.Time(),
				Low:    cndl[i].Low.Float64(),
				High:   cndl[i].High.Float64(),
				Open:   cndl[i].Entry.Float64(),
				Close:  cndl[i].Exit.Float64(),
				Volume: cndl[i].BaseVolume.Float64(),
			}
		}
	default:
		return nil, asset.ErrNotSupported
	}
	return req.ProcessResponse(resp)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	var resp []kline.Candle
	for x := range req.RangeHolder.Ranges {
		switch a {
		case asset.Spot, asset.Margin, asset.CrossMargin:
			cndl, err := e.GetSpotCandlestickData(ctx, req.RequestFormatted, formatExchangeKlineIntervalSpot(req.ExchangeInterval), req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time, 200, true)
			if err != nil {
				return nil, err
			}
			temp := make([]kline.Candle, len(cndl))
			for i := range cndl {
				temp[i] = kline.Candle{
					Time:   cndl[i].Timestamp.Time(),
					Low:    cndl[i].Low.Float64(),
					High:   cndl[i].High.Float64(),
					Open:   cndl[i].Open.Float64(),
					Close:  cndl[i].Close.Float64(),
					Volume: cndl[i].BaseVolume.Float64(),
				}
			}
			resp = append(resp, temp...)
		case asset.Futures:
			cndl, err := e.GetFuturesCandlestickData(ctx, req.RequestFormatted, getProductType(pair), formatExchangeKlineIntervalFutures(req.ExchangeInterval), "", req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time, 200, CallModeHistory)
			if err != nil {
				return nil, err
			}
			temp := make([]kline.Candle, len(cndl))
			for i := range cndl {
				temp[i] = kline.Candle{
					Time:   cndl[i].Timestamp.Time(),
					Low:    cndl[i].Low.Float64(),
					High:   cndl[i].High.Float64(),
					Open:   cndl[i].Entry.Float64(),
					Close:  cndl[i].Exit.Float64(),
					Volume: cndl[i].BaseVolume.Float64(),
				}
			}
			resp = append(resp, temp...)
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return req.ProcessResponse(resp)
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, _ asset.Item) ([]futures.Contract, error) {
	var contracts []futures.Contract
	for i := range prodTypes {
		resp, err := e.GetContractConfig(ctx, currency.Pair{}, prodTypes[i])
		if err != nil {
			return nil, err
		}
		temp := make([]futures.Contract, len(resp))
		for x := range resp {
			temp[x] = futures.Contract{
				Exchange:    e.Name,
				Name:        currency.NewPair(resp[x].BaseCoin, resp[x].QuoteCoin),
				Multiplier:  resp[x].SizeMultiplier.Float64(),
				Asset:       itemDecoder(resp[x].SymbolType),
				Type:        contractTypeDecoder(resp[x].SymbolType),
				Status:      resp[x].SymbolStatus,
				StartDate:   resp[x].DeliveryStartTime.Time(),
				EndDate:     resp[x].DeliveryTime.Time(),
				MaxLeverage: resp[x].MaximumLeverage.Float64(),
			}
			set := make(currency.Currencies, len(resp[x].SupportMarginCoins))
			for y := range resp[x].SupportMarginCoins {
				set[y] = currency.NewCode(resp[x].SupportMarginCoins[y])
			}
			if len(set) > 0 {
				temp[x].SettlementCurrency = set[0]
				if len(set) > 1 {
					temp[x].AdditionalSettlementCurrencies = set[1:]
				}
			}
			if resp[x].SymbolStatus == "listed" || resp[x].SymbolStatus == "normal" {
				temp[x].IsActive = true
			}
		}
		contracts = append(contracts, temp...)
	}
	return contracts, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, req *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	fPair, err := e.FormatExchangeCurrency(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	curRate, err := e.GetFundingCurrent(ctx, fPair, getProductType(fPair))
	if err != nil {
		return nil, err
	}
	nextTime, err := e.GetNextFundingTime(ctx, fPair, getProductType(fPair))
	if err != nil {
		return nil, err
	}
	resp := []fundingrate.LatestRateResponse{
		{
			Exchange:       e.Name,
			Pair:           fPair,
			TimeOfNextRate: nextTime[0].NextFundingTime.Time(),
			TimeChecked:    time.Now(),
		},
	}
	dec := decimal.NewFromFloat(curRate[0].FundingRate.Float64())
	resp[0].LatestRate.Rate = dec
	return resp, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var lim []limits.MinMaxLevel
	switch a {
	case asset.Spot:
		resp, err := e.GetSymbolInfo(ctx, currency.Pair{})
		if err != nil {
			return err
		}
		lim = make([]limits.MinMaxLevel, len(resp))
		for i := range resp {
			lim[i] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, currency.NewPair(resp[i].BaseCoin, resp[i].QuoteCoin)),
				PriceStepIncrementSize:  math.Pow10(-int(resp[i].PricePrecision)),
				AmountStepIncrementSize: math.Pow10(-int(resp[i].QuantityPrecision)),
				QuoteStepIncrementSize:  math.Pow10(-int(resp[i].QuotePrecision)),
				MinNotional:             resp[i].MinimumTradeUSDT.Float64(),
				MarketMinQty:            resp[i].MinimumTradeAmount.Float64(),
				MarketMaxQty:            resp[i].MaximumTradeAmount.Float64(),
			}
		}
	case asset.Futures:
		for i := range prodTypes {
			resp, err := e.GetContractConfig(ctx, currency.Pair{}, prodTypes[i])
			if err != nil {
				return err
			}
			limitsTemp := make([]limits.MinMaxLevel, len(resp))
			for i := range resp {
				limitsTemp[i] = limits.MinMaxLevel{
					Key:            key.NewExchangeAssetPair(e.Name, a, currency.NewPair(resp[i].BaseCoin, resp[i].QuoteCoin)),
					MinNotional:    resp[i].MinimumTradeUSDT.Float64(),
					MaxTotalOrders: resp[i].MaximumSymbolOrderNumber,
				}
			}
			lim = append(lim, limitsTemp...) //nolint:makezero // False positive; the non-zero make is in a different branch
		}
	case asset.Margin, asset.CrossMargin:
		resp, err := e.GetSupportedCurrencies(ctx)
		if err != nil {
			return err
		}
		lim = make([]limits.MinMaxLevel, len(resp))
		for i := range resp {
			lim[i] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, currency.NewPair(resp[i].BaseCoin, resp[i].QuoteCoin)),
				MinNotional:             resp[i].MinimumTradeUSDT.Float64(),
				MarketMinQty:            resp[i].MinimumTradeAmount.Float64(),
				MarketMaxQty:            resp[i].MaximumTradeAmount.Float64(),
				QuoteStepIncrementSize:  math.Pow10(-int(resp[i].PricePrecision)),
				AmountStepIncrementSize: math.Pow10(-int(resp[i].QuantityPrecision)),
			}
		}
	default:
		return asset.ErrNotSupported
	}
	return limits.Load(lim)
}

// UpdateCurrencyStates updates currency states
func (e *Exchange) UpdateCurrencyStates(ctx context.Context, a asset.Item) error {
	payload := make(map[currency.Code]currencystate.Options)
	resp, err := e.GetCoinInfo(ctx, currency.Code{})
	if err != nil {
		return err
	}
	for i := range resp {
		var isWithdraw bool
		var isDeposit bool
		var isTrade bool
		for j := range resp[i].Chains {
			if resp[i].Chains[j].Withdrawable {
				isWithdraw = true
			}
			if resp[i].Chains[j].Rechargeable {
				isDeposit = true
			}
		}
		if isWithdraw && isDeposit {
			isTrade = true
		}
		payload[resp[i].Coin] = currencystate.Options{
			Withdraw: &isWithdraw,
			Deposit:  &isDeposit,
			Trade:    &isTrade,
		}
	}
	return e.States.UpdateAll(a, payload)
}

// GetAvailableTransferChains returns a list of supported transfer chains based on the supplied cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cur currency.Code) ([]string, error) {
	if cur.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	resp, err := e.GetCoinInfo(ctx, cur)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, common.ErrNoResults
	}
	chains := make([]string, len(resp[0].Chains))
	for i := range resp[0].Chains {
		chains[i] = resp[0].Chains[i].Chain
	}
	return chains, nil
}

// GetMarginRatesHistory returns the margin rate history for the supplied currency
func (e *Exchange) GetMarginRatesHistory(ctx context.Context, req *margin.RateHistoryRequest) (*margin.RateHistoryResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	var pagination uint64
	rates := new(margin.RateHistoryResponse)
loop:
	for {
		switch req.Asset {
		case asset.Margin:
			resp, err := e.GetIsolatedInterestHistory(ctx, req.Pair, req.Currency, req.StartDate, req.EndDate, 500, pagination)
			if err != nil {
				return nil, err
			}
			if resp == nil || len(resp.ResultList) == 0 || pagination == uint64(resp.MaximumID) {
				break loop
			}
			pagination = uint64(resp.MaximumID)
			for i := range resp.ResultList {
				rates.Rates = append(rates.Rates, margin.Rate{
					DailyBorrowRate: decimal.NewFromFloat(resp.ResultList[i].DailyInterestRate.Float64()),
					Time:            resp.ResultList[i].CreationTime.Time(),
				})
			}
		case asset.CrossMargin:
			resp, err := e.GetCrossInterestHistory(ctx, req.Currency, req.StartDate, req.EndDate, 500, pagination)
			if err != nil {
				return nil, err
			}
			if resp == nil || len(resp.ResultList) == 0 || pagination == uint64(resp.MaximumID) {
				break loop
			}
			pagination = uint64(resp.MaximumID)
			for i := range resp.ResultList {
				rates.Rates = append(rates.Rates, margin.Rate{
					DailyBorrowRate: decimal.NewFromFloat(resp.ResultList[i].DailyInterestRate.Float64()),
					Time:            resp.ResultList[i].CreationTime.Time(),
				})
			}
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return rates, nil
}

// GetFuturesPositionSummary returns stats for a future position
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	resp, err := e.GetSinglePosition(ctx, getProductType(req.Pair), req.Pair, req.Pair.Quote)
	if err != nil {
		return nil, err
	}
	if len(resp) != 1 {
		// I'm not sure that it should actually return one data point in this case, replace this with a properly formatted error message once certain (i.e. once you can test GetSinglePosition properly)
		return nil, fmt.Errorf("expected 1 position, received %v", len(resp))
	}
	summary := &futures.PositionSummary{
		Pair:                         req.Pair,
		Asset:                        req.Asset,
		CurrentSize:                  decimal.NewFromFloat(resp[0].OpenDelegateSize.Float64()),
		InitialMarginRequirement:     decimal.NewFromFloat(resp[0].MarginSize.Float64()),
		AvailableEquity:              decimal.NewFromFloat(resp[0].Available.Float64()),
		FrozenBalance:                decimal.NewFromFloat(resp[0].Locked.Float64()),
		Leverage:                     decimal.NewFromFloat(resp[0].Leverage.Float64()),
		RealisedPNL:                  decimal.NewFromFloat(resp[0].AchievedProfits.Float64()),
		AverageOpenPrice:             decimal.NewFromFloat(resp[0].OpenPriceAverage.Float64()),
		UnrealisedPNL:                decimal.NewFromFloat(resp[0].UnrealizedProfitLoss.Float64()),
		MaintenanceMarginRequirement: decimal.NewFromFloat(resp[0].KeepMarginRate.Float64()),
		MarkPrice:                    decimal.NewFromFloat(resp[0].MarkPrice.Float64()),
		StartDate:                    resp[0].CreationTime.Time(),
	}
	return summary, nil
}

// GetFuturesPositions returns futures positions for all currencies
func (e *Exchange) GetFuturesPositions(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionDetails, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	var resp []futures.PositionDetails
	// This exchange needs pairs to be passed through, since a MarginCoin has to be provided
	for i := range req.Pairs {
		temp, err := e.GetAllPositions(ctx, getProductType(req.Pairs[i]), req.Pairs[i].Quote)
		if err != nil {
			return nil, err
		}
		for x := range temp {
			pair, err := pairFromStringHelper(temp[x].Symbol)
			if err != nil {
				return nil, err
			}
			ord := []order.Detail{
				{
					Exchange:             e.Name,
					AssetType:            req.Asset,
					Pair:                 pair,
					Side:                 sideDecoder(temp[x].HoldSide),
					RemainingAmount:      temp[x].OpenDelegateSize.Float64(),
					Amount:               temp[x].Total.Float64(),
					Leverage:             temp[x].Leverage.Float64(),
					AverageExecutedPrice: temp[x].OpenPriceAverage.Float64(),
					MarginType:           marginDecoder(temp[x].MarginMode),
					Price:                temp[x].MarkPrice.Float64(),
					Date:                 temp[x].CreationTime.Time(),
				},
			}
			resp = append(resp, futures.PositionDetails{
				Exchange: e.Name,
				Pair:     pair,
				Asset:    req.Asset,
				Orders:   ord,
			})
		}
	}
	return resp, nil
}

// GetFuturesPositionOrders returns futures positions orders
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	pairs := make([]string, len(req.Pairs))
	for x := range req.Pairs {
		pairs[x] = req.Pairs[x].String()
	}
	var resp []futures.PositionResponse
	var err error
	if len(pairs) == 0 {
		for y := range prodTypes {
			if resp, err = e.allFuturesOrderHelper(ctx, prodTypes[y], currency.Pair{}, resp); err != nil {
				return nil, err
			}
		}
	}
	for x := range pairs {
		if resp, err = e.allFuturesOrderHelper(ctx, getProductType(req.Pairs[x]), req.Pairs[x], resp); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// GetHistoricalFundingRates returns historical funding rates for a future
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, req *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	resp, err := e.GetFundingHistorical(ctx, req.Pair, getProductType(req.Pair), 100, 0)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, common.ErrNoResults
	}
	rates := make([]fundingrate.Rate, len(resp))
	for i := range resp {
		rates[i] = fundingrate.Rate{
			Time: resp[i].FundingTime.Time(),
			Rate: decimal.NewFromFloat(resp[i].FundingRate.Float64()),
		}
	}
	rateStruct := &fundingrate.HistoricalRates{
		Exchange:     e.Name,
		Asset:        req.Asset,
		Pair:         req.Pair,
		StartDate:    rates[0].Time,
		EndDate:      rates[len(rates)-1].Time,
		LatestRate:   rates[0],
		FundingRates: rates,
	}
	if len(rates) > 1 {
		rateStruct.TimeOfNextRate = rates[0].Time.Add(rates[0].Time.Sub(rates[1].Time))
	}
	return rateStruct, nil
}

// SetCollateralMode sets the account's collateral mode for the asset type
func (e *Exchange) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return common.ErrFunctionNotSupported
}

// GetCollateralMode returns the account's collateral mode for the asset type
func (e *Exchange) GetCollateralMode(_ context.Context, _ asset.Item) (collateral.Mode, error) {
	return 0, common.ErrFunctionNotSupported
}

// SetMarginType sets the account's margin type for the asset type
func (e *Exchange) SetMarginType(ctx context.Context, a asset.Item, p currency.Pair, t margin.Type) error {
	switch a {
	case asset.Futures:
		var str string
		switch t {
		case margin.Isolated:
			str = "isolated"
		case margin.Multi:
			str = "crossed"
		}
		if _, err := e.ChangeMarginMode(ctx, p, getProductType(p), str, p.Quote); err != nil {
			return err
		}
	default:
		return asset.ErrNotSupported
	}
	return nil
}

// ChangePositionMargin changes the margin type for a position
func (e *Exchange) ChangePositionMargin(_ context.Context, _ *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (e *Exchange) SetLeverage(ctx context.Context, a asset.Item, p currency.Pair, _ margin.Type, f float64, s order.Side) error {
	switch a {
	case asset.Futures:
		if _, err := e.ChangeLeverage(ctx, p, getProductType(p), sideEncoder(s, true), p.Quote, f); err != nil {
			return err
		}
	default:
		return asset.ErrNotSupported
	}
	return nil
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (e *Exchange) GetLeverage(ctx context.Context, a asset.Item, p currency.Pair, t margin.Type, s order.Side) (float64, error) {
	lev := -1.1
	switch a {
	case asset.Futures:
		resp, err := e.GetOneFuturesAccount(ctx, p, getProductType(p), p.Quote)
		if err != nil {
			return lev, err
		}
		switch t {
		case margin.Isolated:
			switch s {
			case order.Buy, order.Long:
				lev = resp.IsolatedLongLeverage
			case order.Sell, order.Short:
				lev = resp.IsolatedShortLeverage
			default:
				return lev, order.ErrSideIsInvalid
			}
		case margin.Multi:
			lev = resp.CrossedMarginleverage
		default:
			return lev, margin.ErrMarginTypeUnsupported
		}
	case asset.Margin:
		resp, err := e.GetIsolatedInterestRateAndMaxBorrowable(ctx, p)
		if err != nil {
			return lev, err
		}
		if len(resp) == 0 {
			return lev, common.ErrNoResults
		}
		lev = resp[0].Leverage.Float64()
	case asset.CrossMargin:
		resp, err := e.GetCrossInterestRateAndMaxBorrowable(ctx, p.Quote)
		if err != nil {
			return lev, err
		}
		if len(resp) == 0 {
			return lev, common.ErrNoResults
		}
		lev = resp[0].Leverage.Float64()
	default:
		return lev, asset.ErrNotSupported
	}
	return lev, nil
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, pairs ...key.PairAsset) ([]futures.OpenInterest, error) {
	openInterest := make([]futures.OpenInterest, len(pairs))
	for i := range pairs {
		resp, err := e.GetOpenPositions(ctx, pairs[i].Pair(), getProductType(pairs[i].Pair()))
		if err != nil {
			return nil, err
		}
		if len(resp.OpenInterestList) == 0 {
			return nil, common.ErrNoResults
		}
		openInterest[i] = futures.OpenInterest{
			OpenInterest: resp.OpenInterestList[0].Size.Float64(),
			Key: key.ExchangeAssetPair{
				Exchange: e.Name,
				Base:     pairs[i].Base,
				Quote:    pairs[i].Quote,
				Asset:    pairs[i].Asset,
			},
		}
	}
	return openInterest, nil
}

// GetProductType is a helper function that returns the appropriate product type for a given currency pair
func getProductType(p currency.Pair) string {
	var prodType string
	switch p.Quote {
	case currency.USDT:
		prodType = "USDT-FUTURES"
	case currency.PERP, currency.USDC:
		prodType = "USDC-FUTURES"
	default:
		prodType = "COIN-FUTURES"
	}
	return prodType
}

// SideDecoder is a helper function that returns the appropriate order side for a given string
func sideDecoder(d string) order.Side {
	switch strings.ToLower(d) {
	case "buy", "long":
		return order.Buy
	case "sell", "short":
		return order.Sell
	}
	return order.UnknownSide
}

// StrategyTruthTable is a helper function that returns the appropriate strategy for a given set of booleans
func strategyTruthTable(tif order.TimeInForce) (string, error) {
	if tif.Is(order.ImmediateOrCancel) && tif.Is(order.FillOrKill) || tif.Is(order.FillOrKill) && tif.Is(order.PostOnly) || tif.Is(order.ImmediateOrCancel) && tif.Is(order.PostOnly) {
		return "", errStrategyMutex
	}
	if tif.Is(order.ImmediateOrCancel) {
		return "ioc", nil
	}
	if tif.Is(order.FillOrKill) {
		return "fok", nil
	}
	if tif.Is(order.PostOnly) {
		return "post_only", nil
	}
	return "gtc", nil
}

// MarginStringer is a helper function that returns the appropriate string for a given margin type
func marginStringer(m margin.Type) string {
	switch m {
	case margin.Isolated:
		return "isolated"
	case margin.Multi:
		return "crossed"
	}
	return ""
}

// SideEncoder is a helper function that returns the appropriate string for a given order side
func sideEncoder(s order.Side, longshort bool) string {
	switch s {
	case order.Buy, order.Long:
		if longshort {
			return "long"
		}
		return "buy"
	case order.Sell, order.Short:
		if longshort {
			return "short"
		}
		return "sell"
	}
	return "unknown side"
}

// PairBatcher is a helper function that batches orders by currency pair
func pairBatcher(orders []order.Cancel) (map[currency.Pair][]OrderIDStruct, error) {
	batchByPair := make(map[currency.Pair][]OrderIDStruct)
	for i := range orders {
		originalID, err := strconv.ParseUint(orders[i].OrderID, 10, 64)
		if err != nil {
			return nil, err
		}
		batchByPair[orders[i].Pair] = append(batchByPair[orders[i].Pair], OrderIDStruct{
			ClientOrderID: orders[i].ClientOrderID,
			OrderID:       EmptyInt(originalID),
		})
	}
	return batchByPair, nil
}

// AddStatuses is a helper function that adds statuses to a response
func addStatuses(status *BatchOrderResp, resp *order.CancelBatchResponse) {
	for i := range status.SuccessList {
		resp.Status[status.SuccessList[i].ClientOrderID] = "success"
		resp.Status[strconv.FormatUint(uint64(status.SuccessList[i].OrderID), 10)] = "success"
	}
	for i := range status.FailureList {
		resp.Status[status.FailureList[i].ClientOrderID] = status.FailureList[i].ErrorMessage
		resp.Status[strconv.FormatUint(uint64(status.FailureList[i].OrderID), 10)] = status.FailureList[i].ErrorMessage
	}
}

// StatusDecoder is a helper function that returns the appropriate status for a given string
func statusDecoder(status string) order.Status {
	switch status {
	case "live":
		return order.Pending
	case "new":
		return order.New
	case "partially_filled", "partially_fill":
		return order.PartiallyFilled
	case "filled", "full_fill":
		return order.Filled
	case "cancelled", "canceled":
		return order.Cancelled
	case "not_trigger":
		return order.PendingTrigger
	}
	return order.UnknownStatus
}

// StrategyDecoder is a helper function that returns the appropriate TimeInForce for a given string
func strategyDecoder(s string) order.TimeInForce {
	switch strings.ToLower(s) {
	case "ioc":
		return order.ImmediateOrCancel
	case "fok":
		return order.FillOrKill
	case "post_only":
		return order.PostOnly
	}
	return order.UnknownTIF
}

// TypeDecoder is a helper function that returns the appropriate order type for a given string
func typeDecoder(s string) order.Type {
	switch s {
	case "limit":
		return order.Limit
	case "market":
		return order.Market
	}
	return order.UnknownType
}

// WithdrawalHistGrabber is a helper function that repeatedly calls GetWithdrawalRecords and returns all data
func (e *Exchange) withdrawalHistGrabber(ctx context.Context, cur currency.Code) ([]WithdrawRecordsResp, error) {
	var allData []WithdrawRecordsResp
	var pagination uint64
	for {
		resp, err := e.GetWithdrawalRecords(ctx, cur, "", time.Now().Add(-time.Hour*24*90), time.Now(), pagination, 0, 100)
		if err != nil {
			return nil, err
		}
		if len(resp) == 0 || pagination == resp[len(resp)-1].OrderID {
			break
		}
		pagination = resp[len(resp)-1].OrderID
		allData = append(allData, resp...)
	}
	return allData, nil
}

// PairFromStringHelper is a helper function that does some checks to help with common ambiguous cases in this exchange
func pairFromStringHelper(s string) (currency.Pair, error) {
	i := strings.LastIndex(s, "USD")
	if i == -1 {
		if i = strings.Index(s, "PERP"); i == -1 {
			return currency.EMPTYPAIR, fmt.Errorf("%w: %q", errUnknownPairQuote, s)
		}
	}
	pair, err := currency.NewPairFromStrings(s[:i], s[i:])
	if err != nil {
		return pair, err
	}
	return pair.Upper(), nil
}

// MarginDecoder is a helper function that returns the appropriate margin type for a given string
func marginDecoder(s string) margin.Type {
	switch s {
	case "isolated":
		return margin.Isolated
	case "cross", "crossed":
		return margin.Multi
	}
	return margin.Unknown
}

// ActiveFuturesOrderHelper is a helper function that repeatedly calls GetPendingFuturesOrders and GetPendingFuturesTriggerOrders, returning the data formatted appropriately
func (e *Exchange) activeFuturesOrderHelper(ctx context.Context, productType string, pairCan currency.Pair, resp []order.Detail) ([]order.Detail, error) {
	var pagination uint64
	for {
		genOrds, err := e.GetPendingFuturesOrders(ctx, 0, pagination, 100, "", productType, "", pairCan, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination == uint64(genOrds.EndID) {
			break
		}
		pagination = uint64(genOrds.EndID)
		tempOrds := make([]order.Detail, len(genOrds.EntrustedList))
		for i := range genOrds.EntrustedList {
			tempOrds[i] = order.Detail{
				Exchange:             e.Name,
				AssetType:            asset.Futures,
				Amount:               genOrds.EntrustedList[i].Size.Float64(),
				OrderID:              strconv.FormatUint(genOrds.EntrustedList[i].OrderID, 10),
				ClientOrderID:        genOrds.EntrustedList[i].ClientOrderID,
				Fee:                  float64(genOrds.EntrustedList[i].Fee),
				Price:                float64(genOrds.EntrustedList[i].Price),
				AverageExecutedPrice: float64(genOrds.EntrustedList[i].PriceAverage),
				Status:               statusDecoder(genOrds.EntrustedList[i].Status),
				Side:                 sideDecoder(genOrds.EntrustedList[i].Side),
				SettlementCurrency:   genOrds.EntrustedList[i].MarginCoin,
				QuoteAmount:          genOrds.EntrustedList[i].QuoteVolume.Float64(),
				Leverage:             genOrds.EntrustedList[i].Leverage.Float64(),
				MarginType:           marginDecoder(genOrds.EntrustedList[i].MarginMode),
				Type:                 typeDecoder(genOrds.EntrustedList[i].OrderType),
				Date:                 genOrds.EntrustedList[i].CreationTime.Time(),
				LastUpdated:          genOrds.EntrustedList[i].UpdateTime.Time(),
				LimitPriceUpper:      float64(genOrds.EntrustedList[i].PresetStopSurplusPrice),
				LimitPriceLower:      float64(genOrds.EntrustedList[i].PresetStopLossPrice),
				TimeInForce:          strategyDecoder(genOrds.EntrustedList[i].Force),
			}
			if !pairCan.IsEmpty() {
				tempOrds[i].Pair = pairCan
			} else {
				tempOrds[i].Pair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
				if err != nil {
					return nil, err
				}
			}
		}
		resp = append(resp, tempOrds...)
	}
	for y := range planTypes {
		pagination = 0
		for {
			genOrds, err := e.GetPendingTriggerFuturesOrders(ctx, 0, pagination, 100, "", planTypes[y], productType, pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination == uint64(genOrds.EndID) {
				break
			}
			pagination = uint64(genOrds.EndID)
			tempOrds := make([]order.Detail, len(genOrds.EntrustedList))
			for i := range genOrds.EntrustedList {
				tempOrds[i] = order.Detail{
					Exchange:           e.Name,
					AssetType:          asset.Futures,
					Amount:             genOrds.EntrustedList[i].Size.Float64(),
					OrderID:            strconv.FormatUint(genOrds.EntrustedList[i].OrderID, 10),
					ClientOrderID:      genOrds.EntrustedList[i].ClientOrderID,
					Price:              float64(genOrds.EntrustedList[i].Price),
					TriggerPrice:       float64(genOrds.EntrustedList[i].TriggerPrice),
					Status:             statusDecoder(genOrds.EntrustedList[i].PlanStatus),
					Side:               sideDecoder(genOrds.EntrustedList[i].Side),
					SettlementCurrency: genOrds.EntrustedList[i].MarginCoin,
					MarginType:         marginDecoder(genOrds.EntrustedList[i].MarginMode),
					Type:               typeDecoder(genOrds.EntrustedList[i].OrderType),
					Date:               genOrds.EntrustedList[i].CreationTime.Time(),
					LastUpdated:        genOrds.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:    float64(genOrds.EntrustedList[i].TakeProfitExecutePrice),
					LimitPriceLower:    float64(genOrds.EntrustedList[i].StopLossExecutePrice),
				}
				if !pairCan.IsEmpty() {
					tempOrds[i].Pair = pairCan
				} else {
					tempOrds[i].Pair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
			}
			resp = append(resp, tempOrds...)
		}
	}
	return resp, nil
}

// SpotHistoricPlanOrdersHelper is a helper function that repeatedly calls GetHistoricalSpotOrders and returns all data formatted appropriately
func (e *Exchange) spotHistoricPlanOrdersHelper(ctx context.Context, pairCan currency.Pair, resp []order.Detail, fillMap map[uint64][]order.TradeHistory) ([]order.Detail, error) {
	var pagination uint64
	for {
		genOrds, err := e.GetSpotPlanOrderHistory(ctx, pairCan, time.Now().Add(-time.Hour*24*90), time.Now().Add(-time.Second), 100, pagination) // Even with time synced, the exchange's clock can lag behind and reject requests; bumping time back by a second to account for this
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.OrderList) == 0 || pagination == uint64(genOrds.IDLessThan) {
			break
		}
		pagination = uint64(genOrds.IDLessThan)
		tempOrds := make([]order.Detail, len(genOrds.OrderList))
		for i := range genOrds.OrderList {
			tempOrds[i] = order.Detail{
				Exchange:      e.Name,
				AssetType:     asset.Spot,
				OrderID:       strconv.FormatUint(genOrds.OrderList[i].OrderID, 10),
				ClientOrderID: genOrds.OrderList[i].ClientOrderID,
				TriggerPrice:  genOrds.OrderList[i].TriggerPrice.Float64(),
				Type:          typeDecoder(genOrds.OrderList[i].OrderType),
				Price:         float64(genOrds.OrderList[i].ExecutePrice),
				Amount:        genOrds.OrderList[i].Size.Float64(),
				Status:        statusDecoder(genOrds.OrderList[i].Status),
				Side:          sideDecoder(genOrds.OrderList[i].Side),
				Date:          genOrds.OrderList[i].CreationTime.Time(),
				LastUpdated:   genOrds.OrderList[i].UpdateTime.Time(),
			}
			tempOrds[i].Pair = pairCan
			if len(fillMap[genOrds.OrderList[i].OrderID]) > 0 {
				tempOrds[i].Trades = fillMap[genOrds.OrderList[i].OrderID]
			}
		}
		resp = append(resp, tempOrds...)
		if !genOrds.NextFlag {
			break
		}
	}
	return resp, nil
}

// HistoricalFuturesOrderHelper is a helper function that repeatedly calls GetFuturesFills, GetHistoricalFuturesOrders, and GetHistoricalTriggerFuturesOrders, returning the data formatted appropriately
func (e *Exchange) historicalFuturesOrderHelper(ctx context.Context, productType string, pairCan currency.Pair, resp []order.Detail) ([]order.Detail, error) {
	var pagination uint64
	fillMap := make(map[uint64][]order.TradeHistory)
	for {
		fillOrds, err := e.GetFuturesFills(ctx, 0, pagination, 100, pairCan, productType, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		if fillOrds == nil || len(fillOrds.FillList) == 0 || pagination == uint64(fillOrds.EndID) {
			break
		}
		pagination = uint64(fillOrds.EndID)
		for i := range fillOrds.FillList {
			tempFill := order.TradeHistory{
				TID:       strconv.FormatUint(fillOrds.FillList[i].TradeID, 10),
				Price:     fillOrds.FillList[i].Price.Float64(),
				Amount:    fillOrds.FillList[i].BaseVolume.Float64(),
				Side:      sideDecoder(fillOrds.FillList[i].Side),
				Timestamp: fillOrds.FillList[i].CreationTime.Time(),
			}
			for y := range fillOrds.FillList[i].FeeDetail {
				tempFill.Fee += fillOrds.FillList[i].FeeDetail[y].TotalFee.Float64()
				tempFill.FeeAsset = fillOrds.FillList[i].FeeDetail[y].FeeCoin.String()
			}
			fillMap[fillOrds.FillList[i].OrderID] = append(fillMap[fillOrds.FillList[i].OrderID], tempFill)
		}
	}
	pagination = 0
	for {
		genOrds, err := e.GetHistoricalFuturesOrders(ctx, 0, pagination, 100, "", productType, "", pairCan, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination == uint64(genOrds.EndID) {
			break
		}
		pagination = uint64(genOrds.EndID)
		tempOrds := make([]order.Detail, len(genOrds.EntrustedList))
		for i := range genOrds.EntrustedList {
			tempOrds[i] = order.Detail{
				Exchange:             e.Name,
				AssetType:            asset.Futures,
				Amount:               genOrds.EntrustedList[i].Size.Float64(),
				OrderID:              strconv.FormatUint(genOrds.EntrustedList[i].OrderID, 10),
				ClientOrderID:        genOrds.EntrustedList[i].ClientOrderID,
				Fee:                  float64(genOrds.EntrustedList[i].Fee),
				Price:                float64(genOrds.EntrustedList[i].Price),
				AverageExecutedPrice: float64(genOrds.EntrustedList[i].PriceAverage),
				Status:               statusDecoder(genOrds.EntrustedList[i].Status),
				Side:                 sideDecoder(genOrds.EntrustedList[i].Side),
				SettlementCurrency:   genOrds.EntrustedList[i].MarginCoin,
				QuoteAmount:          genOrds.EntrustedList[i].QuoteVolume.Float64(),
				Leverage:             genOrds.EntrustedList[i].Leverage.Float64(),
				MarginType:           marginDecoder(genOrds.EntrustedList[i].MarginMode),
				Type:                 typeDecoder(genOrds.EntrustedList[i].OrderType),
				Date:                 genOrds.EntrustedList[i].CreationTime.Time(),
				LastUpdated:          genOrds.EntrustedList[i].UpdateTime.Time(),
				LimitPriceUpper:      float64(genOrds.EntrustedList[i].PresetStopSurplusPrice),
				LimitPriceLower:      float64(genOrds.EntrustedList[i].PresetStopLossPrice),
				TimeInForce:          strategyDecoder(genOrds.EntrustedList[i].Force),
			}
			if !pairCan.IsEmpty() {
				tempOrds[i].Pair = pairCan
			} else {
				tempOrds[i].Pair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
				if err != nil {
					return nil, err
				}
			}
			if len(fillMap[genOrds.EntrustedList[i].OrderID]) > 0 {
				tempOrds[i].Trades = fillMap[genOrds.EntrustedList[i].OrderID]
			}
		}
		resp = append(resp, tempOrds...)
	}
	for y := range planTypes {
		pagination = 0
		for {
			genOrds, err := e.GetHistoricalTriggerFuturesOrders(ctx, 0, pagination, 100, "", planTypes[y], "", productType, pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination == uint64(genOrds.EndID) {
				break
			}
			pagination = uint64(genOrds.EndID)
			tempOrds := make([]order.Detail, len(genOrds.EntrustedList))
			for i := range genOrds.EntrustedList {
				tempOrds[i] = order.Detail{
					Exchange:             e.Name,
					AssetType:            asset.Futures,
					Amount:               genOrds.EntrustedList[i].Size.Float64(),
					OrderID:              strconv.FormatUint(genOrds.EntrustedList[i].OrderID, 10),
					ClientOrderID:        genOrds.EntrustedList[i].ClientOrderID,
					Status:               statusDecoder(genOrds.EntrustedList[i].PlanStatus),
					Price:                float64(genOrds.EntrustedList[i].Price),
					AverageExecutedPrice: float64(genOrds.EntrustedList[i].PriceAverage),
					TriggerPrice:         float64(genOrds.EntrustedList[i].TriggerPrice),
					Side:                 sideDecoder(genOrds.EntrustedList[i].Side),
					SettlementCurrency:   genOrds.EntrustedList[i].MarginCoin,
					MarginType:           marginDecoder(genOrds.EntrustedList[i].MarginMode),
					Type:                 typeDecoder(genOrds.EntrustedList[i].OrderType),
					Date:                 genOrds.EntrustedList[i].CreationTime.Time(),
					LastUpdated:          genOrds.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:      float64(genOrds.EntrustedList[i].PresetTakeProfitPrice),
					LimitPriceLower:      float64(genOrds.EntrustedList[i].PresetStopLossPrice),
				}
				if !pairCan.IsEmpty() {
					tempOrds[i].Pair = pairCan
				} else {
					tempOrds[i].Pair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
				if len(fillMap[genOrds.EntrustedList[i].OrderID]) > 0 {
					tempOrds[i].Trades = fillMap[genOrds.EntrustedList[i].OrderID]
				}
			}
			resp = append(resp, tempOrds...)
		}
	}
	return resp, nil
}

// SpotFillsHelper is a helper function that repeatedly calls GetSpotFills, directly altering the supplied map with that data
func (e *Exchange) spotFillsHelper(ctx context.Context, pair currency.Pair, fillMap map[uint64][]order.TradeHistory) error {
	var pagination uint64
	for {
		genFills, err := e.GetSpotFills(ctx, pair, time.Time{}, time.Time{}, 100, pagination, 0)
		if err != nil {
			return err
		}
		if len(genFills) == 0 || pagination == genFills[len(genFills)-1].TradeID {
			break
		}
		pagination = genFills[len(genFills)-1].TradeID
		for i := range genFills {
			fillMap[genFills[i].TradeID] = append(fillMap[genFills[i].TradeID],
				order.TradeHistory{
					TID:       strconv.FormatUint(genFills[i].TradeID, 10),
					Type:      typeDecoder(genFills[i].OrderType),
					Side:      sideDecoder(genFills[i].Side),
					Price:     genFills[i].PriceAverage.Float64(),
					Amount:    genFills[i].Size.Float64(),
					Fee:       genFills[i].FeeDetail.TotalFee.Float64(),
					FeeAsset:  genFills[i].FeeDetail.FeeCoin.String(),
					Timestamp: genFills[i].CreationTime.Time(),
				})
		}
	}
	return nil
}

// FormatExchangeKlineIntervalSpot is a helper function used to convert kline.Interval to the string format required by the spot API
func formatExchangeKlineIntervalSpot(interval kline.Interval) string {
	switch interval {
	case kline.OneMin:
		return "1min"
	case kline.FiveMin:
		return "5min"
	case kline.FifteenMin:
		return "15min"
	case kline.ThirtyMin:
		return "30min"
	case kline.OneHour:
		return "1h"
	case kline.FourHour:
		return "4h"
	case kline.SixHour:
		return "6Hutc"
	case kline.TwelveHour:
		return "12Hutc"
	case kline.OneDay:
		return "1Dutc"
	case kline.ThreeDay:
		return "3Dutc"
	case kline.OneWeek:
		return "1Wutc"
	case kline.OneMonth:
		return "1Mutc"
	}
	return fmt.Sprintf("%v: %v", errIntervalNotSupported, interval)
}

// FormatExchangeKlineIntervalFutures is a helper function used to convert kline.Interval to the string format required by the futures API
func formatExchangeKlineIntervalFutures(interval kline.Interval) string {
	switch interval {
	case kline.OneMin:
		return "1m"
	case kline.ThreeMin:
		return "3m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1H"
	case kline.FourHour:
		return "4H"
	case kline.SixHour:
		return "6Hutc"
	case kline.TwelveHour:
		return "12Hutc"
	case kline.OneDay:
		return "1Dutc"
	case kline.ThreeDay:
		return "3Dutc"
	case kline.OneWeek:
		return "1Wutc"
	case kline.OneMonth:
		return "1Mutc"
	}
	return fmt.Sprintf("%v: %v", errIntervalNotSupported, interval)
}

// ItemDecoder is a helper function that returns the appropriate asset.Item for a given string
func itemDecoder(s string) asset.Item {
	switch s {
	case "spot", "SPOT":
		return asset.Spot
	case "margin", "MARGIN":
		return asset.Margin
	case "futures", "USDT-FUTURES", "COIN-FUTURES", "USDC-FUTURES", "SUSD-FUTURES", "SCOIN-FUTURES", "SUSDC-FUTURES":
		return asset.Futures
	case "perpetual":
		return asset.PerpetualContract
	case "delivery":
		return asset.DeliveryFutures
	}
	return asset.Empty
}

// contractTypeDecoder is a helper function that returns the appropriate contract type for a given string
func contractTypeDecoder(s string) futures.ContractType {
	switch s {
	case "delivery":
		return futures.LongDated
	case "perpetual":
		return futures.Perpetual
	}
	return futures.Unknown
}

// AllFuturesOrderHelper is a helper function that repeatedly calls GetPendingFuturesOrders and GetPendingFuturesTriggerOrders, returning the data formatted appropriately
func (e *Exchange) allFuturesOrderHelper(ctx context.Context, productType string, pairCan currency.Pair, resp []futures.PositionResponse) ([]futures.PositionResponse, error) {
	var pagination1 uint64
	var pagination2 uint64
	var breakbool1 bool
	var breakbool2 bool
	tempOrds := make(map[currency.Pair][]order.Detail)
	for {
		var genOrds *FuturesOrdResp
		var err error
		if !breakbool1 {
			genOrds, err = e.GetPendingFuturesOrders(ctx, 0, pagination1, 100, "", productType, "", pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination1 == uint64(genOrds.EndID) {
				breakbool1 = true
				genOrds = nil
			} else {
				pagination1 = uint64(genOrds.EndID)
			}
		}
		if !breakbool2 {
			genOrds2, err := e.GetHistoricalFuturesOrders(ctx, 0, pagination2, 100, "", productType, "", pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds2 == nil || len(genOrds2.EntrustedList) == 0 || pagination2 == uint64(genOrds2.EndID) {
				breakbool2 = true
			} else {
				if genOrds == nil {
					genOrds = new(FuturesOrdResp)
				}
				genOrds.EntrustedList = append(genOrds.EntrustedList, genOrds2.EntrustedList...)
				pagination2 = uint64(genOrds2.EndID)
			}
		}
		if breakbool1 && breakbool2 {
			break
		}
		for i := range genOrds.EntrustedList {
			var thisPair currency.Pair
			if !pairCan.IsEmpty() {
				thisPair = pairCan
			} else {
				thisPair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
				if err != nil {
					return nil, err
				}
			}
			tempOrds[thisPair] = append(tempOrds[thisPair], order.Detail{
				Exchange:             e.Name,
				Pair:                 thisPair,
				AssetType:            asset.Futures,
				Amount:               genOrds.EntrustedList[i].Size.Float64(),
				OrderID:              strconv.FormatUint(genOrds.EntrustedList[i].OrderID, 10),
				ClientOrderID:        genOrds.EntrustedList[i].ClientOrderID,
				Fee:                  float64(genOrds.EntrustedList[i].Fee),
				Price:                float64(genOrds.EntrustedList[i].Price),
				AverageExecutedPrice: float64(genOrds.EntrustedList[i].PriceAverage),
				Status:               statusDecoder(genOrds.EntrustedList[i].Status),
				Side:                 sideDecoder(genOrds.EntrustedList[i].Side),
				SettlementCurrency:   genOrds.EntrustedList[i].MarginCoin,
				QuoteAmount:          genOrds.EntrustedList[i].QuoteVolume.Float64(),
				Leverage:             genOrds.EntrustedList[i].Leverage.Float64(),
				MarginType:           marginDecoder(genOrds.EntrustedList[i].MarginMode),
				Type:                 typeDecoder(genOrds.EntrustedList[i].OrderType),
				Date:                 genOrds.EntrustedList[i].CreationTime.Time(),
				LastUpdated:          genOrds.EntrustedList[i].UpdateTime.Time(),
				LimitPriceUpper:      float64(genOrds.EntrustedList[i].PresetStopSurplusPrice),
				LimitPriceLower:      float64(genOrds.EntrustedList[i].PresetStopLossPrice),
				TimeInForce:          strategyDecoder(genOrds.EntrustedList[i].Force),
			})
		}
	}
	for y := range planTypes {
		pagination1 = 0
		for {
			genOrds, err := e.GetPendingTriggerFuturesOrders(ctx, 0, pagination1, 100, "", planTypes[y], productType, pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination1 == uint64(genOrds.EndID) {
				break
			}
			pagination1 = uint64(genOrds.EndID)
			for i := range genOrds.EntrustedList {
				var thisPair currency.Pair
				if !pairCan.IsEmpty() {
					thisPair = pairCan
				} else {
					thisPair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
				tempOrds[thisPair] = append(tempOrds[thisPair], order.Detail{
					Exchange:           e.Name,
					Pair:               thisPair,
					AssetType:          asset.Futures,
					Amount:             genOrds.EntrustedList[i].Size.Float64(),
					OrderID:            strconv.FormatUint(genOrds.EntrustedList[i].OrderID, 10),
					ClientOrderID:      genOrds.EntrustedList[i].ClientOrderID,
					Price:              float64(genOrds.EntrustedList[i].Price),
					TriggerPrice:       float64(genOrds.EntrustedList[i].TriggerPrice),
					Status:             statusDecoder(genOrds.EntrustedList[i].PlanStatus),
					Side:               sideDecoder(genOrds.EntrustedList[i].Side),
					SettlementCurrency: genOrds.EntrustedList[i].MarginCoin,
					MarginType:         marginDecoder(genOrds.EntrustedList[i].MarginMode),
					Type:               typeDecoder(genOrds.EntrustedList[i].OrderType),
					Date:               genOrds.EntrustedList[i].CreationTime.Time(),
					LastUpdated:        genOrds.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:    float64(genOrds.EntrustedList[i].TakeProfitExecutePrice),
					LimitPriceLower:    float64(genOrds.EntrustedList[i].StopLossExecutePrice),
				})
			}
		}
		pagination1 = 0
		for {
			genOrds, err := e.GetHistoricalTriggerFuturesOrders(ctx, 0, pagination1, 100, "", planTypes[y], "", productType, pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination1 == uint64(genOrds.EndID) {
				break
			}
			pagination1 = uint64(genOrds.EndID)
			for i := range genOrds.EntrustedList {
				var thisPair currency.Pair
				if !pairCan.IsEmpty() {
					thisPair = pairCan
				} else {
					thisPair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
				tempOrds[thisPair] = append(tempOrds[thisPair], order.Detail{
					Exchange:             e.Name,
					Pair:                 thisPair,
					AssetType:            asset.Futures,
					Amount:               genOrds.EntrustedList[i].Size.Float64(),
					OrderID:              strconv.FormatUint(genOrds.EntrustedList[i].OrderID, 10),
					ClientOrderID:        genOrds.EntrustedList[i].ClientOrderID,
					Status:               statusDecoder(genOrds.EntrustedList[i].PlanStatus),
					Price:                float64(genOrds.EntrustedList[i].Price),
					AverageExecutedPrice: float64(genOrds.EntrustedList[i].PriceAverage),
					TriggerPrice:         float64(genOrds.EntrustedList[i].TriggerPrice),
					Side:                 sideDecoder(genOrds.EntrustedList[i].Side),
					SettlementCurrency:   genOrds.EntrustedList[i].MarginCoin,
					MarginType:           marginDecoder(genOrds.EntrustedList[i].MarginMode),
					Type:                 typeDecoder(genOrds.EntrustedList[i].OrderType),
					Date:                 genOrds.EntrustedList[i].CreationTime.Time(),
					LastUpdated:          genOrds.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:      float64(genOrds.EntrustedList[i].PresetTakeProfitPrice),
					LimitPriceLower:      float64(genOrds.EntrustedList[i].PresetStopLossPrice),
				})
			}
		}
	}
	for x, y := range tempOrds {
		resp = append(resp, futures.PositionResponse{
			Pair:   x,
			Orders: y,
			Asset:  asset.Futures,
		})
	}
	return resp, nil
}

// ItemEncoder encodes an asset.Item into a string
func itemEncoder(a asset.Item, pair currency.Pair) string {
	switch a {
	case asset.Spot:
		return "SPOT"
	case asset.Futures:
		return getProductType(pair)
	case asset.Margin, asset.CrossMargin:
		return "MARGIN"
	}
	return ""
}

// PositionModeDecoder is a helper function that returns the appropriate position mode for a given string
func positionModeDecoder(s string) futures.PositionMode {
	switch s {
	case "one_way_mode":
		return futures.OneWayMode
	case "hedge_mode":
		return futures.HedgeMode
	}
	return futures.UnknownMode
}
