package mexc

import (
	"context"
	"fmt"
	"math"
	"slices"
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
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// SetDefaults sets the basic defaults for Mexc
func (e *Exchange) SetDefaults() {
	e.Name = "MEXC"
	e.Enabled = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	if err := e.SetAssetPairStore(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: ""},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	// Futures symbols are base and quote joined by an underscore, as BTC_USDT
	if err := e.SetAssetPairStore(asset.Futures, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:    true,
				TickerFetching:    true,
				OrderbookFetching: true,
				KlineFetching:     true,
				TradeFetching:     true,
				AccountInfo:       true,
				SubmitOrder:       true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrder:       true,
				CancelOrders:      true,
				UserTradeHistory:  true,
				TradeFee:          true,
				CryptoDeposit:     true,
				CryptoWithdrawal:  true,
				DepositHistory:    true,
				WithdrawalHistory: true,
				// GetAvailableTransferChains returns the withdraw endpoint's netWork identifiers. The
				// deposit address endpoint names its networks differently, so deposits are not
				// advertised as multi-chain.
				MultiChainWithdrawals: true,
				AutoPairUpdates:       true,
				FundingRateFetching:   true,
				// Spot and futures are separate wallets, funded by transfers, so each is polled for balances
				HasAssetTypeAccountSegregation: true,
			},
			// The spot websocket carries market data and account pushes only; orders are placed over REST.
			// Subscribe and Unsubscribe are left off, so a flush reconnects from scratch: the websocket
			// manager records a connection's subscriptions only when its subscriber returns no error, and
			// the venue refuses a duplicate subscription, so updating a partly refused connection in place
			// would strand the channels it accepted. The incremental flush can be enabled once the manager
			// keeps partial registrations.
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				KlineFetching:          true,
				TradeFetching:          true,
				AccountInfo:            true,
				AuthenticatedEndpoints: true,
			},
			// The wallet API withdraws crypto only.
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				FundingRateBatching: map[asset.Item]bool{
					asset.Futures: true,
				},
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.OneHour:   true,
					kline.FourHour:  true,
					kline.EightHour: true,
					kline.OneDay:    true,
				},
				Positions: true,
				Leverage:  true,
				OpenInterest: exchange.OpenInterestSupport{
					Supported:          true,
					SupportedViaTicker: true,
					SupportsRestBatch:  true,
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
					// Futures only; spot candles have no eight hour interval
					kline.IntervalCapacity{Interval: kline.EightHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				// /api/v3/klines returns at most 500 rows whatever limit is asked for, so each range
				// GetKlineExtendedRequest carves must be sized at 500; a larger limit fills a range half
				// way and zero-pads the remainder silently.
				GlobalResultLimit: 500,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}
	var err error
	e.Requester, err = request.New(
		e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(rateLimits),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:         spotAPIURL,
		exchange.RestFutures:      futuresAPIURL,
		exchange.WebsocketSpot:    spotWebsocketURL,
		exchange.WebsocketFutures: futuresWebsocketURL,
	}); err != nil {
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
	spotWSURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig: exch,
		Features:       &e.Features.Supports.WebsocketCapabilities,
		// MEXC caps a single spot websocket connection at 30 subscriptions.
		MaxWebsocketSubscriptionsPerConnection: 30,
		DefaultURL:                             spotWebsocketURL,
		RunningURL:                             spotWebsocketURL,
		TradeFeed:                              e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement:           true,
	}); err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   spotWSURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      time.Second * 3,
		RateLimit:             request.NewRateLimitWithWeight(time.Second, 2, 1),
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generatePublicSubscriptions,
		Handler:               e.WsHandleData,
		MessageFilter:         asset.Spot,
	}); err != nil {
		return err
	}
	futuresWSURL, err := e.API.Endpoints.GetURL(exchange.WebsocketFutures)
	if err != nil {
		return err
	}
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   futuresWSURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		RateLimit:             request.NewRateLimitWithWeight(time.Second, 10, 1),
		Connector:             e.WebsocketFuturesConnect,
		Subscriber:            e.WebsocketFuturesSubscribe,
		Unsubscriber:          e.WebsocketFuturesUnsubscribe,
		GenerateSubscriptions: e.generateFuturesPublicSubscriptions,
		Handler:               e.WebsocketFuturesHandleData,
		MessageFilter:         futuresPublicConnection,
	}); err != nil {
		return err
	}
	// The private futures pushes follow a login on a connection of their own, so the public ones are not lost when
	// a login is refused
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   futuresWSURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		RateLimit:             request.NewRateLimitWithWeight(time.Second, 10, 1),
		Authenticated:         true,
		Connector:             e.websocketFuturesConnectPrivate,
		Subscriber:            e.websocketFuturesSubscribePrivate,
		Unsubscriber:          e.websocketFuturesUnsubscribePrivate,
		GenerateSubscriptions: e.generateFuturesPrivateSubscriptions,
		Handler:               e.WebsocketFuturesHandleData,
		MessageFilter:         futuresPrivateConnection,
	}); err != nil {
		return err
	}
	// Only the private channels need a listen key, so they have a connection of their own, set up last so
	// that a public connection failing to connect ends the connect before a key is minted.
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   spotWSURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      time.Second * 3,
		RateLimit:             request.NewRateLimitWithWeight(time.Second, 2, 1),
		Authenticated:         true,
		Connector:             e.wsConnectPrivate,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generatePrivateSubscriptions,
		Handler:               e.WsHandleData,
		MessageFilter:         privateConnection,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	pairFormat, err := e.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		result, err := e.GetSymbols(ctx, nil)
		if err != nil {
			return nil, err
		}
		currencyPairs := make(currency.Pairs, 0, len(result.Symbols))
		for i := range result.Symbols {
			// Status is 1 for nearly every symbol in the catalogue; isSpotTradingAllowed is the flag that
			// separates the API-spot-tradable ones, so both must hold.
			if result.Symbols[i].Status.Int64() != 1 || !result.Symbols[i].IsSpotTradingAllowed {
				continue
			}
			pair := currency.NewPair(result.Symbols[i].BaseAsset, result.Symbols[i].QuoteAsset)
			currencyPairs = append(currencyPairs, pair.Format(pairFormat))
		}
		return currencyPairs, nil
	case asset.Futures:
		contracts, err := e.GetFuturesContracts(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		currencyPairs := make(currency.Pairs, 0, len(contracts))
		for _, c := range contracts {
			// A contract closed to API trading is still listed, but orders for it are refused
			if c.State != FuturesContractStateEnabled || !c.APIAllowed {
				continue
			}
			pair, err := currency.NewPairDelimiter(c.Symbol, currency.UnderscoreDelimiter)
			if err != nil {
				return nil, err
			}
			currencyPairs = append(currencyPairs, pair.Format(pairFormat))
		}
		return currencyPairs, nil
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
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
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	pFormat, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch assetType {
	case asset.Spot:
		pairString := pFormat.Format(p)
		tickers, err := e.Get24HourTickerPriceChangeStatistics(ctx, []string{pairString})
		if err != nil {
			return nil, err
		}
		var found bool
		for t := range tickers {
			if tickers[t].Symbol != pairString {
				continue
			}
			found = true
			if err := ticker.ProcessTicker(&ticker.Price{
				Pair:         p,
				ExchangeName: e.Name,
				AssetType:    assetType,
				Last:         tickers[t].LastPrice.Float64(),
				High:         tickers[t].HighPrice.Float64(),
				Low:          tickers[t].LowPrice.Float64(),
				Bid:          tickers[t].BidPrice.Float64(),
				BidSize:      tickers[t].BidQty.Float64(),
				Ask:          tickers[t].AskPrice.Float64(),
				AskSize:      tickers[t].AskQty.Float64(),
				BaseVolume:   tickers[t].Volume.Float64(),
				QuoteVolume:  tickers[t].QuoteVolume.Float64(),
				Open:         tickers[t].OpenPrice.Float64(),
				LastUpdated:  tickers[t].CloseTime.Time(),
			}); err != nil {
				return nil, err
			}
		}
		if !found {
			return nil, fmt.Errorf("%w for currency pair: %s", ticker.ErrTickerNotFound, p)
		}
	case asset.Futures:
		fPair := p.Format(pFormat)
		tickers, err := e.GetFuturesTickers(ctx, fPair)
		if err != nil {
			return nil, err
		}
		if len(tickers) != 1 || tickers[0] == nil || tickers[0].Symbol != fPair.String() {
			return nil, fmt.Errorf("%w for currency pair: %s", ticker.ErrTickerNotFound, p)
		}
		contract, err := e.futuresContract(ctx, fPair.String())
		if err != nil {
			return nil, err
		}
		if err := ticker.ProcessTicker(e.futuresTickerPrice(tickers[0], contract, p)); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// futuresTickerPrice returns a futures ticker as a ticker price. The venue counts volume and open interest in
// contracts, so they are restated in currency using the contract's size
func (e *Exchange) futuresTickerPrice(t *FuturesTickerResponse, c *FuturesContractResponse, pair currency.Pair) *ticker.Price {
	baseVolume, quoteVolume := futuresVolumes(c, t.Volume24Hour, t.Amount24Hour)
	return &ticker.Price{
		Pair:         pair,
		ExchangeName: e.Name,
		AssetType:    asset.Futures,
		Last:         t.LastPrice,
		High:         t.High24Hour,
		Low:          t.Low24Hour,
		Bid:          t.BestBidPrice,
		Ask:          t.BestAskPrice,
		BaseVolume:   baseVolume,
		QuoteVolume:  quoteVolume,
		OpenInterest: t.HoldVolume * c.ContractSize,
		MarkPrice:    t.FairPrice,
		IndexPrice:   t.IndexPrice,
		LastUpdated:  t.Timestamp.Time(),
	}
}

// futuresVolumes returns a volume counted in contracts, with its turnover, as base and quote currency volumes. A linear
// contract's size is in the base currency and its turnover in the quote currency; an inverse contract's size is in
// the quote currency and its turnover in the base currency
func futuresVolumes(c *FuturesContractResponse, contracts, turnover float64) (baseVolume, quoteVolume float64) {
	if c.IsInverse() {
		return turnover, contracts * c.ContractSize
	}
	return contracts * c.ContractSize, turnover
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot:
		tickers, err := e.Get24HourTickerPriceChangeStatistics(ctx, []string{})
		if err != nil {
			return err
		}
		for t := range tickers {
			// Resolve the concatenated symbol against the known pairs. A naive split guesses the
			// base/quote boundary and gets it wrong for any symbol whose split is ambiguous
			// (measured against the live catalogue: 1376 of 2067 spot symbols, e.g. METALUSDT was
			// read as MET/ALUSDT). MatchSymbolWithAvailablePairs looks the symbol up instead. The
			// 24h endpoint returns every listed symbol, so a symbol we do not track (delisted between
			// a catalogue refresh and this poll) is skipped rather than failing the whole update or,
			// as the naive split did, inventing a mis-split pair.
			pair, err := e.MatchSymbolWithAvailablePairs(tickers[t].Symbol, assetType, false)
			if err != nil {
				continue
			}
			if err := ticker.ProcessTicker(&ticker.Price{
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    assetType,
				Last:         tickers[t].LastPrice.Float64(),
				High:         tickers[t].HighPrice.Float64(),
				Low:          tickers[t].LowPrice.Float64(),
				Bid:          tickers[t].BidPrice.Float64(),
				BidSize:      tickers[t].BidQty.Float64(),
				Ask:          tickers[t].AskPrice.Float64(),
				AskSize:      tickers[t].AskQty.Float64(),
				BaseVolume:   tickers[t].Volume.Float64(),
				QuoteVolume:  tickers[t].QuoteVolume.Float64(),
				Open:         tickers[t].OpenPrice.Float64(),
				LastUpdated:  tickers[t].CloseTime.Time(),
			}); err != nil {
				return err
			}
		}
	case asset.Futures:
		tickers, err := e.GetFuturesTickers(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		// The venue lists every contract; one not held as an available pair is skipped, as on spot
		available := make([]*FuturesTickerResponse, 0, len(tickers))
		pairs := make([]currency.Pair, 0, len(tickers))
		for _, t := range tickers {
			if t == nil {
				continue
			}
			pair, err := e.MatchSymbolWithAvailablePairs(t.Symbol, assetType, true)
			if err != nil {
				continue
			}
			available = append(available, t)
			pairs = append(pairs, pair)
		}
		contracts, err := e.futuresContractsFor(ctx, available)
		if err != nil {
			return err
		}
		var errs error
		for i, t := range available {
			c, ok := contracts[t.Symbol]
			if !ok {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", errContractNotFound, t.Symbol))
				continue
			}
			if err := ticker.ProcessTicker(e.futuresTickerPrice(t, c, pairs[i])); err != nil {
				errs = common.AppendError(errs, err)
			}
		}
		return errs
	default:
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	return nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	fPair, err := e.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              fPair,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	switch assetType {
	case asset.Spot:
		result, err := e.GetOrderbook(ctx, fPair, 1000)
		if err != nil {
			return book, err
		}
		book.Bids = result.Bids.Levels()
		book.Asks = result.Asks.Levels()
		// The venue carries its own timestamp and update id; the same correction as wsSendTime on the
		// ws side. Without them Process falls back to time.Now() and a zero update id.
		book.LastUpdated = result.Timestamp.Time()
		book.LastUpdateID = result.LastUpdateID
		if err := book.Process(); err != nil {
			return book, err
		}
	case asset.Futures:
		result, err := e.GetFuturesOrderbook(ctx, fPair, 0)
		if err != nil {
			return book, err
		}
		// Amounts are in contracts, as the venue sends them and as orders are sized
		book.Bids = result.Bids.Levels()
		book.Asks = result.Asks.Levels()
		book.LastUpdated = result.Timestamp.Time()
		book.LastUpdateID = int64(min(result.Version, math.MaxInt64))
		if err := book.Process(); err != nil {
			return book, err
		}
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	return orderbook.Get(e.Name, pair, assetType)
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// accountTypeMatches reports whether the account type the exchange returned is the one asked for.
// The comparison is case-insensitive: the exchange reports "SPOT" while asset.Item renders "spot",
// so a direct comparison never matched and every spot balance request returned ErrNotSupported
// instead of the balances.
func accountTypeMatches(reported string, assetType asset.Item) bool {
	return assetType == asset.Empty || strings.EqualFold(reported, assetType.String())
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	if assetType == asset.Futures {
		assets, err := e.GetFuturesAccountAssets(ctx)
		if err != nil {
			return nil, err
		}
		subAccount := accounts.NewSubAccount(asset.Futures, "")
		for _, a := range assets {
			if a == nil || a.Currency.IsEmpty() {
				continue
			}
			subAccount.Balances.Set(a.Currency, futuresBalance(a.AvailableBalance, a.FrozenBalance, a.PositionMargin))
		}
		subAccounts := accounts.SubAccounts{subAccount}
		return subAccounts, e.Accounts.Save(ctx, subAccounts, true)
	}
	accountInfo, err := e.GetAccountInformation(ctx)
	if err != nil {
		return nil, err
	}
	if !accountTypeMatches(accountInfo.AccountType, assetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}

	subAccount := accounts.SubAccount{
		AssetType: assetType,
		Balances:  make(accounts.CurrencyBalances, len(accountInfo.Balances)),
	}
	for b := range accountInfo.Balances {
		ccy := accountInfo.Balances[b].Asset
		// MEXC reports free (available) and locked (frozen) per asset. The account layer keeps the
		// three views independently: Total = free + locked, Hold = locked, Free = free. Free was left
		// unset, so an asset with free=10/locked=3 reported Total=13/Hold=3/Free=0 - a consumer
		// reading available balance saw nothing.
		subAccount.Balances[ccy] = accounts.Balance{
			Currency: ccy,
			Free:     accountInfo.Balances[b].Free.Float64(),
			Hold:     accountInfo.Balances[b].Locked.Float64(),
			Total:    accountInfo.Balances[b].Free.Float64() + accountInfo.Balances[b].Locked.Float64(),
		}
	}

	subAccounts := accounts.SubAccounts{&subAccount}
	return subAccounts, e.Accounts.Save(ctx, subAccounts, true)
}

func accountStatusToString(status uint8) string {
	switch status {
	case 1:
		return "SMALL"
	case 2:
		return "TIME_DELAY"
	case 3:
		return "LARGE_DELAY"
	case 4:
		return "PENDING"
	case 5:
		return "SUCCESS"
	case 6:
		return "AUDITING"
	case 7:
		return "REJECTED"
	case 8:
		return "REFUND"
	case 9:
		return "PRE_SUCCESS"
	case 10:
		return "INVALID"
	case 11:
		return "RESTRICTED"
	case 12:
		return "COMPLETED"
	}
	return ""
}

func withdrawalStatusToString(withdrawalStatus uint8) string {
	switch withdrawalStatus {
	case 1:
		return "APPLY"
	case 2:
		return "AUDITING"
	case 3:
		return "WAIT"
	case 4:
		return "PROCESSING"
	case 5:
		return "WAIT_PACKAGING"
	case 6:
		return "WAIT_CONFIRM"
	case 7:
		return "SUCCESS"
	case 8:
		return "FAILED"
	case 9:
		return "CANCEL"
	case 10:
		return "MANUAL"
	}
	return ""
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	result, err := e.GetFundDepositHistory(ctx, currency.EMPTYCODE, "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	withdrawals, err := e.GetWithdrawalHistory(ctx, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return nil, err
	}
	depositsLen := len(result)
	resp := make([]exchange.FundingHistory, depositsLen+len(withdrawals))
	for a := range result {
		resp[a] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          accountStatusToString(result[a].Status),
			TransferID:      result[a].TransactionID,
			Timestamp:       result[a].InsertTime.Time(),
			Currency:        result[a].Coin.String(),
			Amount:          result[a].Amount.Float64(),
			CryptoToAddress: result[a].Address,
			TransferType:    "deposit",
		}
	}
	for w := range withdrawals {
		resp[depositsLen+w] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          withdrawalStatusToString(withdrawals[w].Status),
			TransferID:      withdrawals[w].ID,
			Timestamp:       withdrawals[w].UpdateTime.Time(),
			Currency:        withdrawals[w].Coin.String(),
			Amount:          withdrawals[w].Amount.Float64(),
			Fee:             withdrawals[w].TransactionFee.Float64(),
			CryptoToAddress: withdrawals[w].Address,
			CryptoTxID:      withdrawalTxID(withdrawals[w]),
			CryptoChain:     withdrawals[w].Network,
			TransferType:    "withdrawal",
		}
	}
	return resp, nil
}

// withdrawalTxID returns a withdrawal's transaction hash: transHash when the venue sends one, otherwise txId, which
// can be null and can carry an output index after the hash.
func withdrawalTxID(w *WithdrawalInfo) string {
	if w.TransHash != "" {
		return w.TransHash
	}
	return w.TransactionID
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.GetWithdrawalHistory(ctx, c, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals))
	for w := range withdrawals {
		resp[w] = exchange.WithdrawalHistory{
			Status:          withdrawalStatusToString(withdrawals[w].Status),
			TransferID:      withdrawals[w].ID,
			Timestamp:       withdrawals[w].UpdateTime.Time(),
			Currency:        withdrawals[w].Coin.String(),
			Amount:          withdrawals[w].Amount.Float64(),
			Fee:             withdrawals[w].TransactionFee.Float64(),
			CryptoToAddress: withdrawals[w].Address,
			CryptoTxID:      withdrawalTxID(withdrawals[w]),
			CryptoChain:     withdrawals[w].Network,
			TransferType:    "withdrawal",
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		result, err := e.GetRecentTradesList(ctx, p, 0)
		if err != nil {
			return nil, err
		}
		resp := make([]trade.Data, len(result))
		for t := range result {
			side := order.Buy
			if result[t].IsBuyerMaker { // the buyer was the maker, so the taker sold
				side = order.Sell
			}
			resp[t] = trade.Data{
				TID:          result[t].ID,
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        result[t].Price.Float64(),
				Amount:       result[t].Quantity.Float64(),
				Timestamp:    result[t].Time.Time(),
			}
		}
		return resp, nil
	case asset.Futures:
		result, err := e.GetFuturesRecentTrades(ctx, p, futuresRecentTradesLimit)
		if err != nil {
			return nil, err
		}
		resp := make([]trade.Data, 0, len(result))
		for _, t := range result {
			if t == nil {
				continue
			}
			resp = append(resp, trade.Data{
				TID:          t.TradeID.String(),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         futuresTradeSide(t.Side),
				Price:        t.Price,
				Amount:       t.Volume,
				Timestamp:    t.Time.Time(),
			})
		}
		return resp, nil
	default:
		return nil, fmt.Errorf("%w: asset type %v", asset.ErrNotSupported, assetType)
	}
}

// futuresTradeSide returns the taker's side of a public futures trade, 1 for a buy and 2 for a sell
func futuresTradeSide(side uint8) order.Side {
	switch side {
	case 1:
		return order.Buy
	case 2:
		return order.Sell
	}
	return order.UnknownSide
}

// aggregatedTradesPageLimit is the most aggregated trades the venue returns for one request
const aggregatedTradesPageLimit = 1000

// GetHistoricTrades returns historic trade data within the timeframe provided, oldest first. The venue
// keeps about a day of aggregated trades, stamped to the second, and returns at most 1000 of them per
// request, newest first and without trade ids, from a window of at most an hour whose bounds it reads to
// the second. So the window is read in whole seconds, an hour at a time, each hour backwards from its
// end; when a page fills, its oldest second is read again on its own, since the page can stop part-way
// through it.
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, startTime, endTime time.Time) ([]trade.Data, error) {
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	if assetType == asset.Futures {
		// The futures API serves only the latest 100 trades, which GetRecentTrades returns
		return nil, fmt.Errorf("%w for %v", common.ErrFunctionNotSupported, assetType)
	}
	if assetType != asset.Spot {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	// A window reaching past now is read up to now: an hour starting in the future would be refused.
	end := endTime
	if now := time.Now(); end.After(now) {
		end = now
	}
	end = end.Truncate(time.Second)
	// Trades are stamped to the second, so none falls between the window's start and its first whole second.
	first := startTime.Truncate(time.Second)
	if first.Before(startTime) {
		first = first.Add(time.Second)
	}
	var rows []*AggregatedTradeDetail
	for !end.Before(first) {
		if end.Equal(first) {
			in, err := e.aggregatedTradesInSecond(ctx, p, end)
			if err != nil {
				return nil, err
			}
			rows = append(rows, in...)
			break
		}
		start := end.Add(-time.Hour)
		if start.Before(first) {
			start = first
		}
		page, err := e.GetAggregatedTrades(ctx, p, start, end, aggregatedTradesPageLimit)
		if err != nil {
			return nil, err
		}
		if len(page) < aggregatedTradesPageLimit {
			rows = append(rows, page...)
			end = start.Add(-time.Second)
			continue
		}
		oldest := page[0].Timestamp.Time()
		for _, row := range page {
			if row.Timestamp.Time().Before(oldest) {
				oldest = row.Timestamp.Time()
			}
		}
		second := oldest.Truncate(time.Second)
		for _, row := range page {
			if !row.Timestamp.Time().Before(second.Add(time.Second)) {
				rows = append(rows, row)
			}
		}
		in, err := e.aggregatedTradesInSecond(ctx, p, second)
		if err != nil {
			return nil, err
		}
		rows = append(rows, in...)
		end = second.Add(-time.Second)
	}
	// The pages arrive newest first; put the trades oldest first and keep only those inside the window.
	slices.Reverse(rows)
	slices.SortStableFunc(rows, func(a, b *AggregatedTradeDetail) int {
		return a.Timestamp.Time().Compare(b.Timestamp.Time())
	})
	resp := make([]trade.Data, 0, len(rows))
	for _, row := range rows {
		ts := row.Timestamp.Time()
		if ts.Before(startTime) || ts.After(endTime) {
			continue
		}
		side := order.Buy
		if row.MakerBuyer { // the buyer was the maker, so the taker sold
			side = order.Sell
		}
		resp = append(resp, trade.Data{
			TID:          row.LastTradeID,
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        row.Price.Float64(),
			Amount:       row.Quantity.Float64(),
			Timestamp:    ts,
		})
	}
	return resp, nil
}

// aggregatedTradesInSecond returns the aggregated trades stamped within the second starting at second
func (e *Exchange) aggregatedTradesInSecond(ctx context.Context, p currency.Pair, second time.Time) ([]*AggregatedTradeDetail, error) {
	page, err := e.GetAggregatedTrades(ctx, p, second, second.Add(time.Second-time.Millisecond), aggregatedTradesPageLimit)
	if err != nil {
		return nil, err
	}
	if len(page) >= aggregatedTradesPageLimit {
		return nil, fmt.Errorf("%w: %s", errTradesExceedPage, second)
	}
	in := make([]*AggregatedTradeDetail, 0, len(page))
	for _, row := range page {
		if ts := row.Timestamp.Time(); !ts.Before(second) && ts.Before(second.Add(time.Second)) {
			in = append(in, row)
		}
	}
	return in, nil
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	if a == asset.Futures {
		return e.GetFuturesServerTime(ctx)
	}
	serverTime, err := e.GetSystemTime(ctx)
	return serverTime.Time(), err
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	var err error
	s.Pair, err = e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Spot:
		orderTypeString, err := e.OrderTypeStringFromOrderTypeAndTimeInForce(s.Type, s.TimeInForce)
		if err != nil {
			return nil, err
		}
		// MEXC's side enum is only BUY and SELL; order.Side.String() would send Bid/Ask/Long/Short
		// (and Unknown for an unset side), which the venue rejects, so map to the venue's two values.
		var side string
		switch {
		case s.Side.IsLong():
			side = order.Buy.String()
		case s.Side.IsShort():
			side = order.Sell.String()
		default:
			return nil, fmt.Errorf("%w: %v", order.ErrSideIsInvalid, s.Side)
		}
		result, err := e.NewOrder(ctx, s.Pair, s.ClientOrderID, side, orderTypeString, s.Amount, s.QuoteAmount, s.Price)
		if err != nil {
			return nil, err
		}
		if result.ClientOrderID == "" {
			// If the ACK omits the client id, report the id sent with the request.
			result.ClientOrderID = s.ClientOrderID
		}
		orderType, tif, err := e.StringToOrderTypeAndTimeInForce(result.Type)
		if err != nil {
			return nil, err
		}
		orderSide, err := order.StringToOrderSide(result.Side)
		if err != nil {
			return nil, err
		}
		var ordStatus order.Status
		switch {
		case result.Status != "":
			ordStatus, err = orderStatusFromString(result.Status)
			if err != nil {
				return nil, err
			}
		case result.OrderID != "":
			// MEXC's create-order ACK omits status; a populated OrderID from a successful NewOrder
			// means the order was placed, so report New to keep WasOrderPlaced() true instead of
			// UnknownStatus. The sweep resolves the real lifecycle status (FILLED, PARTIALLY_FILLED and so on)
			// from GetOrderInfo afterwards.
			ordStatus = order.New
		}
		resp := &order.SubmitResponse{
			// s.Pair is already in exchange format; the response symbol is concatenated and a naive
			// split mis-reads most MEXC symbols (METALUSDT read as MET/ALUSDT).
			Pair:            s.Pair,
			Exchange:        e.Name,
			Type:            orderType,
			Side:            orderSide,
			AssetType:       asset.Spot,
			Leverage:        s.Leverage,
			ReduceOnly:      s.ReduceOnly,
			Status:          ordStatus,
			QuoteAmount:     s.QuoteAmount,
			OrderID:         result.OrderID,
			ClientOrderID:   result.ClientOrderID,
			Price:           result.Price.Float64(),
			Amount:          result.OrigQty.Float64(),
			Date:            result.TransactTime.Time(),
			LastUpdated:     result.TransactTime.Time(),
			RemainingAmount: result.OrigQty.Float64() - result.ExecutedQty.Float64(),
			TimeInForce:     tif,
		}
		if resp.Amount == 0 && resp.Status == order.New && orderType == order.Market {
			// A market order sized by its quote amount can be acknowledged with origQty 0, and the order manager only
			// follows an order with an amount, so it would stay New: read the order back for its outcome instead.
			if err := e.readBackMarketOrder(ctx, resp); err != nil {
				log.Warnf(log.ExchangeSys, "%s: market order %s (%s) was acknowledged without a quantity and could not be read back: %v", e.Name, resp.OrderID, resp.Pair, err)
			}
		}
		return resp, nil
	case asset.Futures:
		return e.submitFuturesOrder(ctx, s)
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, s.AssetType)
	}
}

// submitFuturesOrder places a futures order sized in contracts. An order without a margin type is isolated, and an
// isolated order opening a position without leverage takes the account's leverage for that side, since the venue
// needs one and order submissions through RPC carry none. A reducing order names the position it closes, as the venue
// recommends, and takes that position's margin mode when it names none
func (e *Exchange) submitFuturesOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if s.QuoteAmount != 0 {
		return nil, fmt.Errorf("%w: futures orders are sized in contracts by Amount", order.ErrAmountIsInvalid)
	}
	side, err := futuresOrderSide(s.Side, s.ReduceOnly)
	if err != nil {
		return nil, err
	}
	orderType, err := futuresOrderTypeFromOrderTypeAndTimeInForce(s.Type, s.TimeInForce)
	if err != nil {
		return nil, err
	}
	openType, err := futuresOpenTypeFromMarginType(s.MarginType)
	if err != nil {
		return nil, err
	}
	if s.Leverage < 0 || s.Leverage != math.Trunc(s.Leverage) {
		return nil, fmt.Errorf("%w: %v, the venue takes whole leverage", errInvalidLeverage, s.Leverage)
	}
	leverage := uint64(s.Leverage)
	var positionID uint64
	if s.ReduceOnly {
		position, err := e.futuresClosingPosition(ctx, s.Pair, side, s.MarginType)
		if err != nil {
			return nil, err
		}
		if position != nil {
			if positionID, err = parseFuturesPositionID(position); err != nil {
				return nil, err
			}
			openType = position.OpenType
		}
	}
	if leverage == 0 && openType == FuturesOpenTypeIsolated && (side == FuturesSideOpenLong || side == FuturesSideOpenShort) {
		if leverage, err = e.futuresCurrentLeverage(ctx, s.Pair, side, openType); err != nil {
			return nil, err
		}
	}
	req := &FuturesOrderRequest{
		Symbol:          s.Pair.String(),
		Volume:          s.Amount,
		Leverage:        leverage,
		Side:            side,
		SideType:        futuresSideType(side),
		Type:            orderType,
		OpenType:        openType,
		ExternalOrderID: s.ClientOrderID,
		PositionID:      positionID,
	}
	if isFuturesLimitOrderType(orderType) {
		req.Price = s.Price
	}
	placed, err := e.PlaceFuturesOrder(ctx, req)
	if err != nil {
		return nil, err
	}
	if placed == nil || placed.OrderID == "" {
		return nil, common.ErrNoResponse
	}
	resp, err := s.DeriveSubmitResponse(placed.OrderID.String())
	if err != nil {
		return nil, err
	}
	// The venue only acknowledges the placement; a market order's fills arrive later, by push or by GetOrderInfo,
	// so no order is reported final here
	resp.Status = order.New
	resp.MarginType = futuresMarginType(openType)
	resp.Leverage = float64(leverage)
	if placedAt := placed.Timestamp.Time(); !placedAt.IsZero() {
		resp.Date = placedAt
		resp.LastUpdated = placedAt
	}
	return resp, nil
}

// futuresClosingPosition returns the open position a reducing order closes: of the side it closes and of the margin
// type it names, or of either margin type when it names none. It returns nil when no such position is open
func (e *Exchange) futuresClosingPosition(ctx context.Context, pair currency.Pair, side uint8, marginType margin.Type) (*FuturesPositionResponse, error) {
	positionType := uint8(FuturesPositionTypeLong)
	if side == FuturesSideCloseShort {
		positionType = FuturesPositionTypeShort
	}
	positions, err := e.GetFuturesOpenPositions(ctx, pair, "")
	if err != nil {
		return nil, err
	}
	var closing *FuturesPositionResponse
	for _, p := range positions {
		if p == nil || p.Symbol != pair.String() || p.PositionType != positionType || p.State == FuturesPositionStateClosed {
			continue
		}
		if marginType != margin.Unset && futuresMarginType(p.OpenType) != marginType {
			continue
		}
		if closing != nil {
			return nil, fmt.Errorf("%w: isolated and cross %s %s positions are open, name a margin type", errMarginTypeAmbiguous, pair, futuresPositionSide(positionType))
		}
		closing = p
	}
	return closing, nil
}

// futuresCurrentLeverage returns the account's leverage for the side and margin mode of a contract's positions
func (e *Exchange) futuresCurrentLeverage(ctx context.Context, pair currency.Pair, side, openType uint8) (uint64, error) {
	positionType := uint8(FuturesPositionTypeLong)
	if side == FuturesSideOpenShort {
		positionType = FuturesPositionTypeShort
	}
	leverages, err := e.GetFuturesLeverage(ctx, pair)
	if err != nil {
		return 0, err
	}
	for _, l := range leverages {
		if l != nil && l.PositionType == positionType && l.OpenType == openType && l.Leverage > 0 {
			return l.Leverage, nil
		}
	}
	return 0, fmt.Errorf("%w: no %s leverage is set for %s %s positions", errInvalidLeverage, futuresMarginType(openType), pair, futuresPositionSide(positionType))
}

// readBackMarketOrder updates a submitted market order's response with the status and quantities the venue holds for it
func (e *Exchange) readBackMarketOrder(ctx context.Context, resp *order.SubmitResponse) error {
	placed, err := e.GetOrderByID(ctx, resp.Pair, "", resp.OrderID)
	if err != nil {
		return err
	}
	if placed == nil {
		return common.ErrNoResponse
	}
	status, err := orderStatusFromString(placed.Status)
	if err != nil {
		return err
	}
	resp.Status = status
	if quantity := placed.OrigQty.Float64(); quantity > 0 {
		resp.Amount = quantity
		resp.RemainingAmount = quantity - placed.ExecutedQty.Float64()
	}
	resp.AverageExecutedPrice = averageExecutedPrice(placed)
	resp.Cost = placed.CummulativeQuoteQty.Float64()
	if updated := placed.UpdateTime.Time(); !updated.IsZero() {
		resp.LastUpdated = updated
	}
	return nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number. A futures order can also be named by its client order
// ID and pair
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if ord != nil && ord.AssetType == asset.Futures {
		return e.cancelFuturesOrder(ctx, ord)
	}
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	if ord.AssetType != asset.Spot {
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
	}
	// MEXC expects the symbol without a delimiter (BTCUSDT). Passing ord.Pair through unformatted
	// sent BTC-USDT when the pair carried a delimiter, which the exchange rejects. An empty pair is
	// left to CancelTradeOrder to reject, preserving its ErrSymbolStringEmpty contract.
	pair := ord.Pair
	if !pair.IsEmpty() {
		var err error
		pair, err = e.FormatExchangeCurrency(pair, ord.AssetType)
		if err != nil {
			return err
		}
	}
	_, err := e.CancelTradeOrder(ctx, pair, ord.OrderID, ord.ClientOrderID, "")
	return err
}

// cancelFuturesOrder cancels a futures order named by its order ID, or by its client order ID and pair
func (e *Exchange) cancelFuturesOrder(ctx context.Context, ord *order.Cancel) error {
	if ord.OrderID == "" {
		if ord.ClientOrderID == "" {
			return order.ErrOrderIDNotSet
		}
		if ord.Pair.IsEmpty() {
			return currency.ErrCurrencyPairEmpty
		}
		pair, err := e.FormatExchangeCurrency(ord.Pair, ord.AssetType)
		if err != nil {
			return err
		}
		results, err := e.CancelFuturesOrderByExternalID(ctx, pair, ord.ClientOrderID)
		if err != nil {
			return err
		}
		// One order is named, so the outcome returned is that order's
		i := slices.IndexFunc(results, func(r *FuturesCancelOrderResponse) bool { return r != nil })
		switch {
		case i == -1:
			return fmt.Errorf("%w: client order %s", errCancelOutcomeMissing, ord.ClientOrderID)
		case results[i].ErrorCode != 0:
			return fmt.Errorf("%w: client order %s: code %d: %s", errFuturesRequestFailed, ord.ClientOrderID, results[i].ErrorCode, results[i].ErrorMessage)
		}
		return nil
	}
	id, err := parseFuturesOrderID(ord.OrderID)
	if err != nil {
		return err
	}
	results, err := e.CancelFuturesOrders(ctx, []uint64{id})
	if err != nil {
		return err
	}
	for _, r := range results {
		if r == nil || r.OrderID.String() != ord.OrderID {
			continue
		}
		if r.ErrorCode != 0 {
			return fmt.Errorf("%w: order %s: code %d: %s", errFuturesRequestFailed, ord.OrderID, r.ErrorCode, r.ErrorMessage)
		}
		return nil
	}
	return fmt.Errorf("%w: order %s", errCancelOutcomeMissing, ord.OrderID)
}

// CancelBatchOrders cancels futures orders by their IDs, up to 50 in a request. Each order's outcome is reported in
// the response's status; a failed request, or an order the venue gives no outcome for, is returned as an error
func (e *Exchange) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(orders) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	ids := make([]uint64, len(orders))
	for i := range orders {
		if orders[i].AssetType != asset.Futures {
			return nil, fmt.Errorf("%w for %v", common.ErrFunctionNotSupported, orders[i].AssetType)
		}
		if err := orders[i].Validate(orders[i].StandardCancel()); err != nil {
			return nil, err
		}
		var err error
		if ids[i], err = parseFuturesOrderID(orders[i].OrderID); err != nil {
			return nil, err
		}
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string, len(orders))}
	var errs error
	for start := 0; start < len(ids); start += futuresBatchLimit {
		end := min(start+futuresBatchLimit, len(ids))
		results, err := e.CancelFuturesOrders(ctx, ids[start:end])
		errs = common.AppendError(errs, recordFuturesCancelOutcomes(resp, orders[start:end], results, err))
	}
	return resp, errs
}

// recordFuturesCancelOutcomes reports each requested order's outcome in the batch response. A request that failed is
// reported against every order it carried, and an order the venue returned no outcome for is reported as missing;
// both are returned as errors
func recordFuturesCancelOutcomes(resp *order.CancelBatchResponse, requested []order.Cancel, results []*FuturesCancelOrderResponse, err error) error {
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
		idx := slices.IndexFunc(results, func(r *FuturesCancelOrderResponse) bool { return r != nil && r.OrderID.String() == id })
		switch {
		case idx == -1:
			resp.Status[id] = errCancelOutcomeMissing.Error()
			errs = common.AppendError(errs, fmt.Errorf("%w: order %s", errCancelOutcomeMissing, id))
		case results[idx].ErrorCode == 0:
			resp.Status[id] = order.Cancelled.String()
		default:
			resp.Status[id] = results[idx].ErrorMessage
		}
	}
	return errs
}

// CancelAllOrders cancels all orders associated with a currency pair, or every open order of the
// account when no pair is given
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// This is a symbol-wide cancel: it cancels every open order for the pair and takes no order id,
	// so StandardCancel() (which requires an OrderID) must not gate it - it rejected a valid
	// symbol-wide request with order.ErrOrderIDNotSet.
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	resp := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	var err error
	switch orderCancellation.AssetType {
	case asset.Spot:
		if orderCancellation.Pair.IsEmpty() {
			// The account-wide cancel names no orders, so the status map stays empty on success.
			return resp, e.CancelAllOpenOrders(ctx)
		}
		orderCancellation.Pair, err = e.FormatExchangeCurrency(orderCancellation.Pair, orderCancellation.AssetType)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		result, err := e.CancelAllOpenOrdersBySymbol(ctx, orderCancellation.Pair)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		for r := range result {
			resp.Status[result[r].OrderID] = "cancelled"
		}
		return resp, nil
	case asset.Futures:
		pair := orderCancellation.Pair
		if !pair.IsEmpty() {
			if pair, err = e.FormatExchangeCurrency(pair, orderCancellation.AssetType); err != nil {
				return order.CancelAllResponse{}, err
			}
		}
		// The venue confirms the cancellation without naming the orders, so the status map stays empty on success
		return resp, e.CancelAllFuturesOrders(ctx, pair)
	default:
		return order.CancelAllResponse{}, asset.ErrNotSupported
	}
}

// averageExecutedPrice returns the average fill price of a REST order, cummulativeQuoteQty over
// executedQty. The price field is not the average: it is the limit on a limit order, and a filled
// market order reports a price that differs from its fills.
func averageExecutedPrice(o *OrderDetail) float64 {
	executed := o.ExecutedQty.Float64()
	if executed <= 0 {
		return 0
	}
	return o.CummulativeQuoteQty.Float64() / executed
}

// accountTradesPageLimit is the most fills the venue returns for one myTrades request
const accountTradesPageLimit = 1000

// accountTradesMaxPages bounds the myTrades requests made to read the fills of one order
const accountTradesMaxPages = 20

// tradesForOrder fetches the fills of a spot order and maps them to domain trade records plus the
// aggregated commission. MEXC charges commission per fill in an asset the venue chooses (base,
// quote, or the MX discount token), so the fee currency is read from the fill and never assumed;
// when the fills charged a commission disagree on its asset, the aggregate is left unset. It is
// best-effort: a myTrades failure must not sink the order lookup, so callers pass through the base
// order.
//
// myTrades returns at most 1000 fills per request, newest first and stamped to the second, with no
// cursor, and its fill ids do not follow the fills' order. When a page comes back full, the fills
// are read on in windows from the order's creation to the end of the oldest second of the page,
// which a full page can stop part-way through, dropping fills already read by id. A second holding
// a page of fills or more is passed over with a warning, since the rest of its fills cannot be read,
// and the reads go on from the second before it unless the order was created in that second or later.
// The reads stop when a page is not full, or with a warning when a page holds fills after the end of its
// window or after accountTradesMaxPages requests; the fills read so far are then reported. executed is
// the order's executed quantity: when the fills read sum to less, they are reported with a warning. It
// is read before the fills, so an order still filling can have more fills than it counts, and a sum at
// or above it is not proof that none is missing.
func (e *Exchange) tradesForOrder(ctx context.Context, pair currency.Pair, orderID string, created time.Time, executed decimal.Decimal) (trades []order.TradeHistory, totalFee float64, feeAsset currency.Code) {
	// MEXC can report an order as filled a moment before its fills surface in myTrades, so a single
	// immediate lookup sometimes finds nothing for a just-completed order. Retry once with a short
	// gap rather than polling repeatedly. Best-effort throughout: the order lookup still returns
	// without commission on failure, but the reason is named rather than dropped silently.
	var page []*AccountTrade
	for attempt := 0; ; attempt++ {
		var err error
		page, err = e.GetAccountTradeList(ctx, pair, orderID, time.Time{}, time.Time{}, accountTradesPageLimit)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s: myTrades lookup failed for order %s (%s): %v", e.Name, orderID, pair, err)
			return nil, 0, currency.EMPTYCODE
		}
		if len(page) > 0 || attempt >= 1 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, 0, currency.EMPTYCODE
		case <-time.After(time.Second):
		}
	}
	if len(page) == 0 {
		log.Warnf(log.ExchangeSys, "%s: myTrades returned no fills for order %s (%s); commission not materialised", e.Name, orderID, pair)
		return nil, 0, currency.EMPTYCODE
	}
	fills := make([]*AccountTrade, 0, len(page))
	seen := make(map[string]struct{}, len(page))
	// The order's time carries milliseconds but its fills are stamped to the second, so the windows start at
	// the second the order was created in; starting at its time would leave out the fills of that second.
	start := created.Truncate(time.Second)
	// end is the end of the window the page was read from; the first page is read without one
	var end time.Time
	for requests := 1; ; requests++ {
		for _, f := range page {
			if _, ok := seen[f.ID]; !ok {
				seen[f.ID] = struct{}{}
				fills = append(fills, f)
			}
		}
		if !end.IsZero() && slices.ContainsFunc(page, func(f *AccountTrade) bool { return f.Time.Time().After(end) }) {
			log.Warnf(log.ExchangeSys, "%s: myTrades answered for order %s (%s) outside the window read; commission covers %d fills", e.Name, orderID, pair, len(fills))
			break
		}
		if len(page) < accountTradesPageLimit {
			break
		}
		if requests >= accountTradesMaxPages {
			log.Warnf(log.ExchangeSys, "%s: order %s (%s) has more fills than %d myTrades pages read; commission covers %d fills", e.Name, orderID, pair, accountTradesMaxPages, len(fills))
			break
		}
		oldest := page[0].Time.Time()
		for _, f := range page {
			if f.Time.Time().Before(oldest) {
				oldest = f.Time.Time()
			}
		}
		second := oldest.Truncate(time.Second)
		if end.Equal(second.Add(time.Second - time.Millisecond)) {
			// A full page read up to the end of its oldest second never left that second, so the second holds a
			// page of fills or more: the rest of them cannot be read, but the seconds before it can.
			log.Warnf(log.ExchangeSys, "%s: order %s (%s) has a myTrades page or more of fills in the second at %s; commission may not cover all of them", e.Name, orderID, pair, second)
			if !start.Before(second) {
				break
			}
			end = second.Add(-time.Millisecond)
		} else {
			end = second.Add(time.Second - time.Millisecond)
		}
		from := start
		if !from.Before(end) {
			from = time.Time{}
		}
		var err error
		if page, err = e.GetAccountTradeList(ctx, pair, orderID, from, end, accountTradesPageLimit); err != nil {
			log.Warnf(log.ExchangeSys, "%s: myTrades lookup failed for order %s (%s) after %d fills: %v", e.Name, orderID, pair, len(fills), err)
			break
		}
	}
	var filled decimal.Decimal
	for _, f := range fills {
		filled = filled.Add(f.Quantity.Decimal())
	}
	if filled.LessThan(executed) {
		log.Warnf(log.ExchangeSys, "%s: fills incomplete for order %s (%s): myTrades fills sum to %s of %s executed; commission covers the fills read", e.Name, orderID, pair, filled, executed)
	}
	trades = make([]order.TradeHistory, 0, len(fills))
	uniformFee := true
	// charged records whether any fill was charged a commission, and unchargedAsset is the asset the fills
	// name when none was, unless they name different ones
	var charged, unchargedMixed bool
	var unchargedAsset currency.Code
	for _, f := range fills {
		side := order.Buy
		if !f.IsBuyer {
			side = order.Sell
		}
		fillAsset := f.CommissionAsset
		totalFee += f.Commission.Float64()
		switch {
		case f.Commission.Float64() == 0:
			// A fill charged nothing adds nothing in any asset, so it only names the fee asset when no fill was charged
			if unchargedAsset.IsEmpty() {
				unchargedAsset = fillAsset
			} else if !fillAsset.IsEmpty() && !unchargedAsset.Equal(fillAsset) {
				unchargedMixed = true
			}
		case !charged:
			charged = true
			feeAsset = fillAsset
		case !feeAsset.Equal(fillAsset):
			uniformFee = false
		}
		trades = append(trades, order.TradeHistory{
			Price:     f.Price.Float64(),
			Amount:    f.Quantity.Float64(),
			Fee:       f.Commission.Float64(),
			Exchange:  e.Name,
			TID:       f.ID,
			Side:      side,
			Timestamp: f.Time.Time(),
			IsMaker:   f.IsMaker,
			FeeAsset:  f.CommissionAsset.String(),
			Total:     f.QuoteQuantity.Float64(),
		})
	}
	if !charged && !unchargedMixed {
		feeAsset = unchargedAsset
	}
	if !uniformFee {
		// Commissions charged in different assets cannot be summed into a single figure: the total
		// would be a meaningless mix of currencies with a stale label. Report no aggregate fee and no
		// currency; each fill's own commission and asset are still carried on the TradeHistory records.
		totalFee = 0
		feeAsset = currency.EMPTYCODE
	}
	return trades, totalFee, feeAsset
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	pairFormat, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		if pair.IsEmpty() {
			return nil, currency.ErrSymbolStringEmpty
		}
		result, err := e.GetOrderByID(ctx, pair.Format(pairFormat), "", orderID)
		if err != nil {
			return nil, err
		}
		oType, tif, err := e.StringToOrderTypeAndTimeInForce(result.Type)
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(result.Side)
		if err != nil {
			return nil, err
		}
		var oStatus order.Status
		if result.Status != "" {
			oStatus, err = orderStatusFromString(result.Status)
			if err != nil {
				return nil, err
			}
		}
		// The Query Order response carries time and updateTime but no transactTime (that field only
		// exists on the New Order response), so LastUpdated must come from updateTime and fall back to
		// the creation time when the order is still open (updateTime null decodes to the zero time).
		lastUpdated := result.UpdateTime.Time()
		if lastUpdated.IsZero() {
			lastUpdated = result.Time.Time()
		}
		detail := &order.Detail{
			Price:       result.Price.Float64(),
			Amount:      result.OrigQty.Float64(),
			QuoteAmount: result.CummulativeQuoteQty.Float64(),
			// Cost is the quote actually spent (cumulative filled value); the rpc server maps
			// Detail.Cost to the proto cost field, so a market order's real executed cost reaches
			// the caller instead of a zero. Price alone is the protective limit, not the average.
			Cost:                 result.CummulativeQuoteQty.Float64(),
			CostAsset:            pair.Quote,
			AverageExecutedPrice: averageExecutedPrice(result),
			TriggerPrice:         result.StopPrice.Float64(),
			ExecutedAmount:       result.ExecutedQty.Float64(),
			RemainingAmount:      result.OrigQty.Float64() - result.ExecutedQty.Float64(),
			Exchange:             e.Name,
			OrderID:              result.OrderID,
			ClientOrderID:        result.ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            asset.Spot,
			Date:                 result.Time.Time(),
			LastUpdated:          lastUpdated,
			// pair is the pair the caller asked for; the response symbol is concatenated and a naive
			// split mis-reads most MEXC symbols (METALUSDT read as MET/ALUSDT).
			Pair:        pair.Format(pairFormat),
			TimeInForce: tif,
		}
		// Enrich with the venue's commission facts when the order has executed volume. The Query
		// Order response carries no commission, so the fee and its currency come from myTrades keyed
		// by this order.
		if result.ExecutedQty.Float64() > 0 {
			if trades, fee, feeAsset := e.tradesForOrder(ctx, pair.Format(pairFormat), orderID, result.Time.Time(), result.ExecutedQty.Decimal()); len(trades) > 0 {
				detail.Trades = trades
				detail.Fee = fee
				detail.FeeAsset = feeAsset
			}
		}
		return detail, nil
	case asset.Futures:
		result, err := e.GetFuturesOrder(ctx, orderID)
		if err != nil {
			return nil, err
		}
		if result == nil || result.OrderID == "" {
			return nil, common.ErrNoResponse
		}
		// A pair which has left the available pairs falls back to the one the caller named
		fallback := currency.EMPTYPAIR
		if !pair.IsEmpty() {
			fallback = pair.Format(pairFormat)
		}
		detail, err := e.futuresOrderDetail(ctx, result, fallback)
		if err != nil {
			return nil, err
		}
		if result.DealVolume > 0 {
			deals, err := e.GetFuturesOrderDealDetails(ctx, orderID)
			if err != nil {
				// The fills only enrich the order, so a failed lookup does not sink it
				log.Warnf(log.ExchangeSys, "%s: fills lookup failed for futures order %s (%s): %v", e.Name, orderID, detail.Pair, err)
			} else {
				detail.Trades = futuresTradeHistory(e.Name, deals)
			}
		}
		return &detail, nil
	default:
		return nil, fmt.Errorf("%w: asset type: %v", order.ErrAssetNotSet, assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, code currency.Code, _, chain string) (*deposit.Address, error) {
	result, err := e.GetDepositAddressOfCoin(ctx, code, chain)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, deposit.ErrAddressNotFound
	}
	// Without a pinned network the venue returns one address per network; take the first. The
	// destination tag arrives as memo (the field table documents memo, not tag), so read memo and
	// fall back to tag.
	tag := result[0].Memo
	if tag == "" {
		tag = result[0].Tag
	}
	return &deposit.Address{
		Address: result[0].Address,
		Tag:     tag,
		Chain:   result[0].Network,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.WithdrawCapital(ctx,
		withdrawRequest.Amount,
		withdrawRequest.Currency,
		withdrawRequest.ClientOrderID,
		withdrawRequest.Crypto.Chain,
		"", // contract address
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Description)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: resp.ID,
	}, nil
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency.
// Each chain is the network's netWork value, the identifier the withdraw endpoint takes; network is
// only a display name the venue is retiring.
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	coins, err := e.GetCurrencyInformation(ctx)
	if err != nil {
		return nil, err
	}
	var chains []string
	for i := range coins {
		if !coins[i].Coin.Equal(cryptocurrency) {
			continue
		}
		for j := range coins[i].NetworkList {
			if coins[i].NetworkList[j].NetWork != "" {
				chains = append(chains, coins[i].NetworkList[j].NetWork)
			}
		}
	}
	return chains, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// orderDetailFromRESTOrder maps a spot REST order record to a domain order.Detail. It is the single
// mapping shared by GetActiveOrders and GetOrderHistory: both previously omitted the pair and the
// order's real timestamps, and GetOrderHistory parsed the type with the generic order.StringToOrderType,
// which does not know MEXC's IMMEDIATE_OR_CANCEL/FILL_OR_KILL/LIMIT_MAKER types and failed the whole
// query, returning nothing.
func (e *Exchange) orderDetailFromRESTOrder(o *OrderDetail, fallbackPair currency.Pair) (order.Detail, error) {
	// A symbol that has left the available pairs (delisted with a working order, or a catalogue not
	// yet refreshed) must not sink the whole listing; fall back to the pair the caller asked for
	// instead of failing, the same way UpdateTickers skips an untracked symbol rather than erroring.
	pair, err := e.MatchSymbolWithAvailablePairs(o.Symbol, asset.Spot, false)
	if err != nil {
		pair = fallbackPair
	}
	oType, tif, err := e.StringToOrderTypeAndTimeInForce(o.Type)
	if err != nil {
		return order.Detail{}, err
	}
	oSide, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return order.Detail{}, err
	}
	var oStatus order.Status
	if o.Status != "" {
		oStatus, err = orderStatusFromString(o.Status)
		if err != nil {
			return order.Detail{}, err
		}
	}
	// MEXC returns updateTime:null on an open (still-working) order, which decodes to the zero time.
	// Fall back to the creation time so LastUpdated is never the zero time.
	lastUpdated := o.UpdateTime.Time()
	if lastUpdated.IsZero() {
		lastUpdated = o.Time.Time()
	}
	return order.Detail{
		Price:                o.Price.Float64(),
		Amount:               o.OrigQty.Float64(),
		AverageExecutedPrice: averageExecutedPrice(o),
		TriggerPrice:         o.StopPrice.Float64(),
		QuoteAmount:          o.CummulativeQuoteQty.Float64(),
		// Cost is the quote actually spent (cumulative filled value), mapped to the proto cost
		// field by the rpc server; without it a market order reports a zero cost to the caller.
		Cost:            o.CummulativeQuoteQty.Float64(),
		CostAsset:       pair.Quote,
		ExecutedAmount:  o.ExecutedQty.Float64(),
		RemainingAmount: o.OrigQty.Float64() - o.ExecutedQty.Float64(),
		Exchange:        e.Name,
		OrderID:         o.OrderID,
		ClientOrderID:   o.ClientOrderID,
		Type:            oType,
		Side:            oSide,
		Status:          oStatus,
		AssetType:       asset.Spot,
		Date:            o.Time.Time(),
		LastUpdated:     lastUpdated,
		Pair:            pair,
		TimeInForce:     tif,
	}, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	pairFormat, err := e.GetPairFormat(getOrdersRequest.AssetType, true)
	if err != nil {
		return nil, err
	}
	switch getOrdersRequest.AssetType {
	case asset.Spot:
		if len(getOrdersRequest.Pairs) == 0 {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		var details []order.Detail
		for p := range getOrdersRequest.Pairs {
			result, err := e.GetOpenOrders(ctx, getOrdersRequest.Pairs[p].Format(pairFormat))
			if err != nil {
				return nil, err
			}
			for r := range result {
				detail, err := e.orderDetailFromRESTOrder(result[r], getOrdersRequest.Pairs[p].Format(pairFormat))
				if err != nil {
					return nil, err
				}
				details = append(details, detail)
			}
		}
		// The request's side, type and time filters were ignored; apply them here so a caller asking
		// for only buys or only limit orders is not handed the full open-order set.
		return getOrdersRequest.Filter(e.Name, details), nil
	case asset.Futures:
		// The venue lists open orders across every contract, so the pairs asked for are picked out by the filter
		var details []order.Detail
		seen := make(map[FuturesID]struct{})
		for page := uint64(1); ; page++ {
			result, err := e.GetFuturesOpenOrders(ctx, &FuturesPageRequest{PageNumber: page, PageSize: futuresDefaultPageSizeLimit})
			if err != nil {
				return nil, err
			}
			added, err := e.appendFuturesOrders(ctx, &details, seen, result)
			if err != nil {
				return nil, err
			}
			// A page of only known orders means the page number was not applied, so further pages would repeat it
			if len(result) < futuresDefaultPageSizeLimit || !added {
				break
			}
		}
		return getOrdersRequest.Filter(e.Name, details), nil
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
}

// appendFuturesOrders appends the orders not yet seen as order details and reports whether any was added
func (e *Exchange) appendFuturesOrders(ctx context.Context, details *[]order.Detail, seen map[FuturesID]struct{}, orders []*FuturesOrderResponse) (bool, error) {
	var added bool
	for _, o := range orders {
		if o == nil {
			continue
		}
		if _, ok := seen[o.OrderID]; ok {
			continue
		}
		seen[o.OrderID] = struct{}{}
		added = true
		detail, err := e.futuresOrderDetail(ctx, o, currency.EMPTYPAIR)
		if err != nil {
			return added, err
		}
		*details = append(*details, detail)
	}
	return added, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status. Without a start and end time the venue returns the
// last 24 hours of spot orders, and it serves at most the last 7 days of them. Futures orders are read
// in 90 day parts up to now, and without a start and end time the venue returns the last 7 days.
func (e *Exchange) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	pairFormat, err := e.GetPairFormat(getOrdersRequest.AssetType, true)
	if err != nil {
		return nil, err
	}
	switch getOrdersRequest.AssetType {
	case asset.Spot:
		if len(getOrdersRequest.Pairs) == 0 {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		var details []order.Detail
		for p := range getOrdersRequest.Pairs {
			pair := getOrdersRequest.Pairs[p].Format(pairFormat)
			result, err := e.GetAllOrders(ctx, pair, getOrdersRequest.StartTime, getOrdersRequest.EndTime, 1000)
			if err != nil {
				return nil, err
			}
			for r := range result {
				detail, err := e.orderDetailFromRESTOrder(result[r], pair)
				if err != nil {
					return nil, err
				}
				details = append(details, detail)
			}
		}
		// The request's side and type filters were ignored; apply them here so a caller asking for
		// only sells or only limit orders is not handed the full history.
		return getOrdersRequest.Filter(e.Name, details), nil
	case asset.Futures:
		if len(getOrdersRequest.Pairs) == 0 {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		var details []order.Detail
		seen := make(map[FuturesID]struct{})
		for _, p := range getOrdersRequest.Pairs {
			for _, window := range futuresHistoryWindows(getOrdersRequest.StartTime, getOrdersRequest.EndTime) {
				for page := uint64(1); ; page++ {
					result, err := e.GetFuturesOrderHistory(ctx, &FuturesOrderHistoryRequest{
						Symbol:     p.Format(pairFormat),
						States:     []uint8{FuturesOrderStateFilled, FuturesOrderStateCancelled, FuturesOrderStateInvalid},
						StartTime:  window.start,
						EndTime:    window.end,
						PageNumber: page,
						PageSize:   futuresDefaultPageSizeLimit,
					})
					if err != nil {
						return nil, err
					}
					added, err := e.appendFuturesOrders(ctx, &details, seen, result)
					if err != nil {
						return nil, err
					}
					if len(result) < futuresDefaultPageSizeLimit || !added {
						break
					}
				}
			}
		}
		return getOrdersRequest.Filter(e.Name, details), nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
}

// historyWindow is a window of a history request
type historyWindow struct {
	start, end time.Time
}

// futuresHistoryWindows splits a window into the 90 day windows the futures history endpoints serve. A window missing
// a bound is sent as it is, and the venue then returns the last 7 days
func futuresHistoryWindows(start, end time.Time) []historyWindow {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return []historyWindow{{start: start, end: end}}
	}
	now := time.Now()
	var windows []historyWindow
	for from := start; from.Before(end); from = from.Add(futuresHistoryWindow) {
		if len(windows) > 0 && from.After(now) {
			// A later window starting after now could hold no orders, and its request would be refused
			break
		}
		to := from.Add(futuresHistoryWindow)
		if to.After(end) {
			to = end
		}
		windows = append(windows, historyWindow{start: from, end: to})
	}
	return windows
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	// GetFeeByType returns the absolute fee amount, not the rate. The amount is rate * price * quantity;
	// returning the bare rate reported e.g. 0.0005 as if it were the fee. The offline branch is the
	// same calculation against a fixed worst-case rate, used when no credentials are available to ask
	// the exchange for the account's own schedule.
	if err := common.NilGuard(feeBuilder); err != nil {
		return 0, err
	}
	// The account's own rates need credentials; without them the offline estimate is the best answer.
	if feeBuilder.FeeType == exchange.CryptocurrencyTradeFee && !e.AreCredentialsValid(ctx) {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		if feeBuilder.IsMaker {
			return 0., nil
		}
		return 0.0005 * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	case exchange.CryptocurrencyTradeFee:
		pair, err := e.FormatExchangeCurrency(feeBuilder.Pair, asset.Spot)
		if err != nil {
			return 0, err
		}
		result, err := e.GetSymbolTradingFee(ctx, pair)
		if err != nil {
			return 0, err
		}
		rate := result.Data.TakerCommission
		if feeBuilder.IsMaker {
			rate = result.Data.MakerCommission
		}
		return rate * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	case exchange.CryptocurrencyWithdrawalFee:
	case exchange.CryptocurrencyDepositFee:
	case exchange.InternationalBankDepositFee:
	}
	return 0, nil
}

// candlesFromCandlestick maps the exchange's candlestick rows to kline candles. The candle is stamped
// with its open time (the interval start), which is this repository's kline convention: the exchange
// also reports a close time, and stamping the candle with that shifted every candle forward by one
// interval. It is the single mapping shared by GetHistoricCandles and GetHistoricCandlesExtended.
func candlesFromCandlestick(result []*CandlestickData) []kline.Candle {
	candles := make([]kline.Candle, len(result))
	for i := range result {
		candles[i] = kline.Candle{
			Open:   result[i].OpenPrice.Float64(),
			High:   result[i].HighPrice.Float64(),
			Low:    result[i].LowPrice.Float64(),
			Close:  result[i].ClosePrice.Float64(),
			Volume: result[i].Volume.Float64(),
			Time:   result[i].OpenTime.Time(),
		}
	}
	return candles
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	pair, err := e.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		intervalString, err := intervalToString(req.ExchangeInterval)
		if err != nil {
			return nil, err
		}
		// Pass the request's aligned window: MEXC's klines endpoint drops a partially elapsed candle,
		// so a start that is not on an interval boundary loses the first candle and ProcessResponse
		// pads the gap with an all-zero candle indistinguishable from real data.
		result, err := e.GetCandlestick(ctx, pair, intervalString, req.Start, req.End, 0)
		if err != nil {
			return nil, err
		}
		return req.ProcessResponse(candlesFromCandlestick(result))
	case asset.Futures:
		candles, err := e.futuresCandles(ctx, pair, req.ExchangeInterval, req.Start, req.End)
		if err != nil {
			return nil, err
		}
		return req.ProcessResponse(candles)
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	pFormat, err := e.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		intervalString, err := intervalToString(req.ExchangeInterval)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, 0, req.Size())
		for x := range req.RangeHolder.Ranges {
			result, err := e.GetCandlestick(
				ctx,
				pair.Format(pFormat),
				intervalString,
				req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time,
				req.RequestLimit,
			)
			if err != nil {
				return nil, err
			}
			timeSeries = append(timeSeries, candlesFromCandlestick(result)...)
		}
		return req.ProcessResponse(timeSeries)
	case asset.Futures:
		timeSeries := make([]kline.Candle, 0, req.Size())
		for x := range req.RangeHolder.Ranges {
			candles, err := e.futuresCandles(ctx, pair.Format(pFormat), req.ExchangeInterval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
			if err != nil {
				return nil, err
			}
			timeSeries = append(timeSeries, candles...)
		}
		return req.ProcessResponse(timeSeries)
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
}

// futuresCandles returns a contract's candles within a window. Each candle is stamped with its open time, and its
// volume is restated from contracts in the base currency
func (e *Exchange) futuresCandles(ctx context.Context, pair currency.Pair, interval kline.Interval, start, end time.Time) ([]kline.Candle, error) {
	result, err := e.GetFuturesCandles(ctx, &FuturesCandlesRequest{
		Symbol:    pair,
		Interval:  interval,
		StartTime: start,
		// The venue reads the end as the last candle's open, and the window's end is exclusive
		EndTime: end.Add(-time.Second),
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, common.ErrNoResponse
	}
	n := len(result.Time)
	if len(result.Open) != n || len(result.High) != n || len(result.Low) != n || len(result.Close) != n || len(result.Volume) != n || len(result.Amount) != n {
		return nil, fmt.Errorf("%w: candle fields of unequal lengths", errMalformedCandles)
	}
	contract, err := e.futuresContract(ctx, pair.String())
	if err != nil {
		return nil, err
	}
	candles := make([]kline.Candle, n)
	for i := range n {
		baseVolume, _ := futuresVolumes(contract, result.Volume[i], result.Amount[i])
		candles[i] = kline.Candle{
			Time:   result.Time[i].Time(),
			Open:   result.Open[i],
			High:   result.High[i],
			Low:    result.Low[i],
			Close:  result.Close[i],
			Volume: baseVolume,
		}
	}
	return candles, nil
}

// GetFuturesContractDetails returns the details of every futures contract
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, a asset.Item) ([]futures.Contract, error) {
	if !a.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if a != asset.Futures {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	contracts, err := e.GetFuturesContracts(ctx, currency.EMPTYPAIR)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, 0, len(contracts))
	for _, c := range contracts {
		pair, err := currency.NewPairDelimiter(c.Symbol, currency.UnderscoreDelimiter)
		if err != nil {
			return nil, err
		}
		contractType := futures.Perpetual
		if c.FutureType == FuturesTypeDelivery {
			// The venue lists no delivery contracts and documents no expiry cycle for them
			contractType = futures.Unknown
		}
		settlementType := futures.Linear
		if c.IsInverse() {
			settlementType = futures.Inverse
		}
		startDate := c.OpeningTime.Time()
		if startDate.IsZero() {
			startDate = c.CreateTime.Time()
		}
		resp = append(resp, futures.Contract{
			Exchange:           e.Name,
			Name:               pair,
			Underlying:         currency.NewPair(c.BaseCoin, c.QuoteCoin),
			Asset:              a,
			StartDate:          startDate,
			EndDate:            c.DeliveryTime.Time(),
			IsActive:           c.State == FuturesContractStateEnabled && c.APIAllowed,
			Status:             futuresContractStatus(c.State),
			Type:               contractType,
			SettlementType:     settlementType,
			SettlementCurrency: c.SettleCoin,
			MarginCurrency:     c.SettleCoin,
			Multiplier:         c.ContractSize,
			MaxLeverage:        float64(c.MaxLeverage),
		})
	}
	return resp, nil
}

// futuresContractStatus names a contract state
func futuresContractStatus(state uint8) string {
	switch state {
	case FuturesContractStateEnabled:
		return "enabled"
	case FuturesContractStateDelivery:
		return "delivery"
	case FuturesContractStateDelivered:
		return "delivered"
	case FuturesContractStateOffline:
		return "offline"
	case FuturesContractStatePaused:
		return "paused"
	}
	return "unknown"
}

// GetLatestFundingRates returns the latest settled funding rates of perpetual contracts, those of the enabled pairs
// when no pair is set. The current rate is settled at the next funding time, so it is returned as the predicted rate
// when requested
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %s %s", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	pairs := currency.Pairs{r.Pair}
	if r.Pair.IsEmpty() {
		var err error
		if pairs, err = e.GetEnabledPairs(r.Asset); err != nil {
			return nil, err
		}
	}
	for i := range pairs {
		var err error
		if pairs[i], err = e.FormatExchangeCurrency(pairs[i], r.Asset); err != nil {
			return nil, err
		}
	}
	// One pair is read on its own, and several from the list of every contract's rate
	query := currency.EMPTYPAIR
	if !r.Pair.IsEmpty() {
		query = pairs[0]
	}
	current, err := e.GetFuturesFundingRates(ctx, query)
	if err != nil {
		return nil, err
	}
	currentBySymbol := make(map[string]*FuturesFundingRateResponse, len(current))
	for _, c := range current {
		if c != nil {
			currentBySymbol[c.Symbol] = c
		}
	}
	resp := make([]fundingrate.LatestRateResponse, 0, len(pairs))
	for _, p := range pairs {
		rate, ok := currentBySymbol[p.String()]
		if !ok {
			return nil, fmt.Errorf("%w: %s", fundingrate.ErrNoFundingRatesFound, p)
		}
		history, err := e.GetFuturesFundingRateHistory(ctx, p, &FuturesPageRequest{PageNumber: 1, PageSize: 1})
		if err != nil {
			return nil, err
		}
		latest := fundingrate.LatestRateResponse{
			Exchange:       e.Name,
			Asset:          r.Asset,
			Pair:           p,
			TimeOfNextRate: rate.NextSettleTime.Time(),
			TimeChecked:    time.Now(),
		}
		if history != nil && len(history.ResultList) > 0 {
			latest.LatestRate = fundingrate.Rate{
				Time: history.ResultList[0].SettleTime.Time(),
				Rate: decimal.MustFromFloat(history.ResultList[0].FundingRate),
			}
		}
		if r.IncludePredictedRate {
			latest.PredictedUpcomingRate = fundingrate.Rate{
				Time: rate.NextSettleTime.Time(),
				Rate: decimal.MustFromFloat(rate.FundingRate),
			}
		}
		resp = append(resp, latest)
	}
	return resp, nil
}

// futuresFundingRateHistoryMaxPages bounds the funding rate history pages read for one request
const futuresFundingRateHistoryMaxPages = 50

// GetHistoricalFundingRates returns a perpetual contract's settled funding rates within a window, oldest first;
// without a start date, the latest 1000 settled by the end date, or by now without one. The current rate is settled at
// the next funding time, so it is returned as the predicted rate when requested
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %s %s", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if r.IncludePayments {
		return nil, fmt.Errorf("%w: funding payments", common.ErrFunctionNotSupported)
	}
	if !r.StartDate.IsZero() && !r.EndDate.IsZero() {
		if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
			return nil, err
		}
	}
	fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	// The history is paged newest first, so pages are read until one reaches back to the start
	var rates []fundingrate.Rate
	seen := make(map[int64]bool)
	for page := uint64(1); page <= futuresFundingRateHistoryMaxPages; page++ {
		history, err := e.GetFuturesFundingRateHistory(ctx, fPair, &FuturesPageRequest{PageNumber: page, PageSize: futuresFundingRateHistoryPageSizeLimit})
		if err != nil {
			return nil, err
		}
		if history == nil {
			break
		}
		var added, reachedStart bool
		for i := range history.ResultList {
			settled := history.ResultList[i].SettleTime.Time()
			if seen[settled.UnixMilli()] {
				continue
			}
			seen[settled.UnixMilli()] = true
			added = true
			if !r.StartDate.IsZero() && settled.Before(r.StartDate) {
				reachedStart = true
				continue
			}
			if !r.EndDate.IsZero() && settled.After(r.EndDate) {
				continue
			}
			rates = append(rates, fundingrate.Rate{
				Time: settled,
				Rate: decimal.MustFromFloat(history.ResultList[i].FundingRate),
			})
		}
		// A page of only known rates means the page number was not applied, so further pages would repeat it
		if reachedStart || !added || len(history.ResultList) < futuresFundingRateHistoryPageSizeLimit ||
			(r.StartDate.IsZero() && len(rates) >= futuresFundingRateHistoryPageSizeLimit) {
			break
		}
	}
	if len(rates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}
	slices.SortFunc(rates, func(a, b fundingrate.Rate) int {
		return a.Time.Compare(b.Time)
	})
	if r.StartDate.IsZero() && len(rates) > futuresFundingRateHistoryPageSizeLimit {
		rates = rates[len(rates)-futuresFundingRateHistoryPageSizeLimit:]
	}
	resp := &fundingrate.HistoricalRates{
		Exchange:     e.Name,
		Asset:        r.Asset,
		Pair:         fPair,
		StartDate:    rates[0].Time,
		EndDate:      rates[len(rates)-1].Time,
		LatestRate:   rates[len(rates)-1],
		FundingRates: rates,
	}
	if r.IncludePredictedRate {
		current, err := e.GetFuturesFundingRates(ctx, fPair)
		if err != nil {
			return nil, err
		}
		if len(current) != 1 || current[0] == nil {
			return nil, fmt.Errorf("%w: %s", fundingrate.ErrNoFundingRatesFound, fPair)
		}
		resp.TimeOfNextRate = current[0].NextSettleTime.Time()
		resp.PredictedUpcomingRate = fundingrate.Rate{
			Time: current[0].NextSettleTime.Time(),
			Rate: decimal.MustFromFloat(current[0].FundingRate),
		}
	}
	return resp, nil
}

// IsPerpetualFutureCurrency reports whether a contract is perpetual; every contract the venue lists is
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.Futures, nil
}

// GetCollateralCurrencyForContract returns the settlement currency of a contract
func (e *Exchange) GetCollateralCurrencyForContract(a asset.Item, cp currency.Pair) (currency.Code, asset.Item, error) {
	if a != asset.Futures {
		return currency.EMPTYCODE, asset.Empty, fmt.Errorf("%w %q", futures.ErrNotFuturesAsset, a)
	}
	if cp.IsEmpty() {
		return currency.EMPTYCODE, asset.Empty, currency.ErrCurrencyPairEmpty
	}
	fPair, err := e.FormatExchangeCurrency(cp, a)
	if err != nil {
		return currency.EMPTYCODE, asset.Empty, err
	}
	e.futuresContractsMu.RLock()
	c, ok := e.futuresContracts[fPair.String()]
	e.futuresContractsMu.RUnlock()
	if !ok {
		return currency.EMPTYCODE, asset.Empty, fmt.Errorf("%w: %s", errContractNotFound, fPair)
	}
	return c.SettleCoin, a, nil
}

// GetOpenInterest returns the open interest of contracts: in the base currency for a linear contract and in the
// quote currency for an inverse one, the currency its size is counted in
func (e *Exchange) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range keys {
		if keys[i].Asset != asset.Futures {
			// Avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, keys[i].Asset, keys[i].Pair())
		}
	}
	symbol := currency.EMPTYPAIR
	if len(keys) == 1 {
		var err error
		if symbol, err = e.FormatExchangeCurrency(keys[0].Pair(), asset.Futures); err != nil {
			return nil, err
		}
	}
	tickers, err := e.GetFuturesTickers(ctx, symbol)
	if err != nil {
		return nil, err
	}
	wanted := make([]*FuturesTickerResponse, 0, len(tickers))
	pairs := make([]currency.Pair, 0, len(tickers))
	for _, t := range tickers {
		if t == nil {
			continue
		}
		pair, enabled, err := e.MatchSymbolCheckEnabled(t.Symbol, asset.Futures, true)
		if err != nil {
			continue
		}
		switch {
		case len(keys) == 0 && !enabled:
			continue
		case len(keys) > 0 && !slices.ContainsFunc(keys, func(k key.PairAsset) bool { return k.Pair().Equal(pair) }):
			continue
		}
		wanted = append(wanted, t)
		pairs = append(pairs, pair)
	}
	contracts, err := e.futuresContractsFor(ctx, wanted)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.OpenInterest, 0, len(wanted))
	for i, t := range wanted {
		c, ok := contracts[t.Symbol]
		if !ok {
			return nil, fmt.Errorf("%w: %s", errContractNotFound, t.Symbol)
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, asset.Futures, pairs[i]),
			OpenInterest: t.HoldVolume * c.ContractSize,
		})
	}
	return resp, nil
}

// SetLeverage sets the leverage of a contract's long or short positions in a margin mode. A buy side sets the long
// positions' leverage and a sell side the short positions'; an unset side sets both. An open position is named by its
// ID, as the venue asks, and a side without one has the leverage set for the positions it opens
func (e *Exchange) SetLeverage(ctx context.Context, a asset.Item, pair currency.Pair, marginType margin.Type, amount float64, side order.Side) error {
	if a != asset.Futures {
		return fmt.Errorf("%w %q", futures.ErrNotFuturesAsset, a)
	}
	if amount <= 0 || amount != math.Trunc(amount) {
		return fmt.Errorf("%w: %v, the venue takes whole leverage", errInvalidLeverage, amount)
	}
	openType, err := futuresOpenTypeFromMarginType(marginType)
	if err != nil {
		return err
	}
	positionTypes, err := futuresPositionTypes(side)
	if err != nil {
		return err
	}
	fPair, err := e.FormatExchangeCurrency(pair, a)
	if err != nil {
		return err
	}
	positions, err := e.GetFuturesOpenPositions(ctx, fPair, "")
	if err != nil {
		return err
	}
	for _, positionType := range positionTypes {
		req := &FuturesChangeLeverageRequest{Leverage: uint64(amount), OpenType: openType, Symbol: fPair.String(), PositionType: positionType}
		if i := slices.IndexFunc(positions, func(p *FuturesPositionResponse) bool {
			return p != nil && p.Symbol == fPair.String() && p.PositionType == positionType && p.OpenType == openType && p.State != FuturesPositionStateClosed
		}); i != -1 {
			positionID, err := parseFuturesPositionID(positions[i])
			if err != nil {
				return err
			}
			req = &FuturesChangeLeverageRequest{PositionID: positionID, Leverage: uint64(amount)}
		}
		if err := e.ChangeFuturesLeverage(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// parseFuturesPositionID returns a position's ID as the number the venue's request bodies take
func parseFuturesPositionID(p *FuturesPositionResponse) (uint64, error) {
	id, err := strconv.ParseUint(p.PositionID.String(), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: position %q", errInvalidFuturesID, p.PositionID)
	}
	return id, nil
}

// futuresPositionTypes returns the position types a side names: a buy names the long positions and a sell the short
// ones, and an unset side names both
func futuresPositionTypes(side order.Side) ([]uint8, error) {
	switch {
	case side == order.UnknownSide:
		return []uint8{FuturesPositionTypeLong, FuturesPositionTypeShort}, nil
	case side.IsLong():
		return []uint8{FuturesPositionTypeLong}, nil
	case side.IsShort():
		return []uint8{FuturesPositionTypeShort}, nil
	}
	return nil, fmt.Errorf("%w: %v", order.ErrSideIsInvalid, side)
}

// GetLeverage returns the leverage of a contract's long or short positions in a margin mode. A buy side names the long
// positions and a sell side the short ones; an unset side needs both to share one leverage
func (e *Exchange) GetLeverage(ctx context.Context, a asset.Item, pair currency.Pair, marginType margin.Type, side order.Side) (float64, error) {
	if a != asset.Futures {
		return 0, fmt.Errorf("%w %q", futures.ErrNotFuturesAsset, a)
	}
	if pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	openType, err := futuresOpenTypeFromMarginType(marginType)
	if err != nil {
		return 0, err
	}
	positionTypes, err := futuresPositionTypes(side)
	if err != nil {
		return 0, err
	}
	fPair, err := e.FormatExchangeCurrency(pair, a)
	if err != nil {
		return 0, err
	}
	leverages, err := e.GetFuturesLeverage(ctx, fPair)
	if err != nil {
		return 0, err
	}
	var leverage uint64
	for _, positionType := range positionTypes {
		i := slices.IndexFunc(leverages, func(l *FuturesLeverageResponse) bool {
			return l != nil && l.PositionType == positionType && l.OpenType == openType
		})
		if i == -1 {
			return 0, fmt.Errorf("%w: no %s leverage for %s %s positions", errInvalidLeverage, futuresMarginType(openType), fPair, futuresPositionSide(positionType))
		}
		if leverage != 0 && leverages[i].Leverage != leverage {
			return 0, fmt.Errorf("%w: %s long and short positions use different leverage, name a side", errInvalidLeverage, fPair)
		}
		leverage = leverages[i].Leverage
	}
	return float64(leverage), nil
}

// GetFuturesPositionSummary returns a summary of a contract's open position. With both sides open in hedge mode the
// direction of the request names the side; isolated positions are preferred over cross positions
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, r *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %q", futures.ErrNotFuturesAsset, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	positions, err := e.GetFuturesOpenPositions(ctx, fPair, "")
	if err != nil {
		return nil, err
	}
	pos, err := selectFuturesPosition(positions, fPair.String(), r.Direction)
	if err != nil {
		return nil, err
	}
	contract, err := e.futuresContract(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	size := decimal.MustFromFloat(pos.HoldVolume)
	if pos.PositionType == FuturesPositionTypeShort {
		size = size.Neg()
	}
	settlementType := futures.Linear
	if contract.IsInverse() {
		settlementType = futures.Inverse
	}
	summary := &futures.PositionSummary{
		Pair:                      r.Pair,
		Asset:                     r.Asset,
		MarginType:                futuresMarginType(pos.OpenType),
		Currency:                  contract.SettleCoin,
		NotionalSize:              decimal.MustFromFloat(futuresCost(contract, pos.HoldVolume, pos.HoldAveragePrice)),
		Leverage:                  decimal.NewFromInt(int64(min(pos.Leverage, math.MaxInt64))),
		EstimatedLiquidationPrice: decimal.MustFromFloat(pos.LiquidatePrice),
		CollateralUsed:            decimal.MustFromFloat(pos.InitialMargin),
		CurrentSize:               size,
		ContractSize:              decimal.MustFromFloat(contract.ContractSize),
		ContractMultiplier:        decimal.MustFromFloat(contract.ContractSize),
		ContractSettlementType:    settlementType,
		AverageOpenPrice:          decimal.MustFromFloat(pos.OpenAveragePrice),
		UnrealisedPNL:             decimal.MustFromFloat(pos.UnrealisedPNL),
		RealisedPNL:               decimal.MustFromFloat(pos.RealisedPNL),
	}
	if pos.OpenType == FuturesOpenTypeIsolated {
		summary.IsolatedMargin = decimal.MustFromFloat(pos.InitialMargin)
	}
	return summary, nil
}

// selectFuturesPosition returns a contract's open position, of the side the direction names when one is named, and
// otherwise the only one held; isolated positions are preferred over cross positions
func selectFuturesPosition(positions []*FuturesPositionResponse, symbol string, direction order.Side) (*FuturesPositionResponse, error) {
	var selected *FuturesPositionResponse
	sides := make(map[uint8]bool)
	for _, p := range positions {
		if p == nil || p.Symbol != symbol || p.State == FuturesPositionStateClosed {
			continue
		}
		switch {
		case direction.IsLong() && p.PositionType != FuturesPositionTypeLong:
			continue
		case direction.IsShort() && p.PositionType != FuturesPositionTypeShort:
			continue
		}
		sides[p.PositionType] = true
		if selected == nil || p.OpenType == FuturesOpenTypeIsolated {
			selected = p
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("%w for %s", futures.ErrNoPositionsFound, symbol)
	}
	if len(sides) > 1 {
		return nil, fmt.Errorf("%w: %s has long and short positions open, name a direction", order.ErrSideIsInvalid, symbol)
	}
	return selected, nil
}

// GetFuturesPositionOrders returns the orders of contracts' positions, from their order history within the window
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, r *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %q", futures.ErrNotFuturesAsset, r.Asset)
	}
	if len(r.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	resp := make([]futures.PositionResponse, len(r.Pairs))
	for i, p := range r.Pairs {
		orders, err := e.GetOrderHistory(ctx, &order.MultiOrderRequest{
			Pairs:     currency.Pairs{p},
			AssetType: r.Asset,
			Type:      order.AnyType,
			Side:      order.AnySide,
			StartTime: r.StartDate,
			EndTime:   r.EndDate,
		})
		if err != nil {
			return nil, err
		}
		settlementType := futures.Linear
		fPair, err := e.FormatExchangeCurrency(p, r.Asset)
		if err != nil {
			return nil, err
		}
		contract, err := e.futuresContract(ctx, fPair.String())
		if err != nil {
			return nil, err
		}
		if contract.IsInverse() {
			settlementType = futures.Inverse
		}
		resp[i] = futures.PositionResponse{
			Pair:                   p,
			Asset:                  r.Asset,
			ContractSettlementType: settlementType,
			Orders:                 orders,
		}
	}
	return resp, nil
}

// ChangePositionMargin sets the margin of an isolated position. With both sides open in hedge mode the margin side
// names the position
func (e *Exchange) ChangePositionMargin(ctx context.Context, r *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %q", futures.ErrNotFuturesAsset, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if r.MarginType != margin.Isolated {
		return nil, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, r.MarginType)
	}
	if r.NewAllocatedMargin <= 0 {
		return nil, margin.ErrNewAllocatedMarginRequired
	}
	// A margin side which cannot be read would select whichever position is open, so it is refused before any request
	side := order.UnknownSide
	if r.MarginSide != "" {
		var err error
		if side, err = order.StringToOrderSide(r.MarginSide); err != nil {
			return nil, err
		}
	}
	fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	positions, err := e.GetFuturesOpenPositions(ctx, fPair, "")
	if err != nil {
		return nil, err
	}
	isolated := slices.DeleteFunc(slices.Clone(positions), func(p *FuturesPositionResponse) bool {
		return p == nil || p.OpenType != FuturesOpenTypeIsolated
	})
	pos, err := selectFuturesPosition(isolated, fPair.String(), side)
	if err != nil {
		return nil, err
	}
	positionID, err := parseFuturesPositionID(pos)
	if err != nil {
		return nil, err
	}
	// The venue changes margin by an amount, so the change is the difference to the position's current margin
	change := futuresMarginChange(r.NewAllocatedMargin, pos.InitialMargin)
	if change == 0 {
		return &margin.PositionChangeResponse{Exchange: e.Name, Pair: r.Pair, Asset: r.Asset, MarginType: r.MarginType, AllocatedMargin: pos.InitialMargin}, nil
	}
	req := &FuturesChangeMarginRequest{PositionID: positionID, Amount: change, Type: FuturesMarginAdd}
	if change < 0 {
		req.Amount, req.Type = -change, FuturesMarginSubtract
	}
	if err := e.ChangeFuturesPositionMargin(ctx, req); err != nil {
		return nil, err
	}
	return &margin.PositionChangeResponse{
		Exchange:        e.Name,
		Pair:            r.Pair,
		Asset:           r.Asset,
		MarginType:      r.MarginType,
		AllocatedMargin: r.NewAllocatedMargin,
	}, nil
}

// futuresMarginChange returns the amount which moves a position's margin to a new margin, as the shortest decimal
// within the error of their float64 difference, so that no binary remainder such as 2.555624999999999 is sent. A
// difference within that error is no change
func futuresMarginChange(newMargin, currentMargin float64) float64 {
	change := newMargin - currentMargin
	larger := max(math.Abs(newMargin), math.Abs(currentMargin))
	tolerance := 2 * (math.Nextafter(larger, math.Inf(1)) - larger)
	if math.Abs(change) <= tolerance {
		return 0
	}
	for digits := 1; ; digits++ {
		// Seventeen significant digits identify any float64, and FormatFloat's output always parses
		rounded, _ := strconv.ParseFloat(strconv.FormatFloat(change, 'g', digits, 64), 64)
		if digits == 17 || math.Abs(rounded-change) <= tolerance {
			return rounded
		}
	}
}

// UpdateOrderExecutionLimits updates order execution limits
//
// The PERCENT_PRICE_BY_SIDE filter is stored as ratios to the last trade price, the form the Binance
// wrapper loads into MultiplierUp and MultiplierDown: 1+bidMultiplierUp caps a buy and
// 1-askMultiplierDown floors a sell. The limits store does not enforce them itself. MinimumQuoteAmount and MinNotional carry
// quoteAmountPrecision; quoteAmountPrecisionMarket equals it on every listed symbol. The market-order
// quote cap maxQuoteAmountMarket is not loaded: MarketMaxQty is a base amount and MaximumQuoteAmount
// already holds the limit-order cap maxQuoteAmount. Symbols whose orderTypes omit MARKET are not
// flagged either, as the limits carry no order-type set.
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot:
		result, err := e.GetSymbols(ctx, nil)
		if err != nil {
			return err
		}
		pairFormat, err := e.GetPairFormat(assetType, false)
		if err != nil {
			return err
		}
		l := make([]limits.MinMaxLevel, len(result.Symbols))
		for a := range result.Symbols {
			pair := currency.NewPair(result.Symbols[a].BaseAsset, result.Symbols[a].QuoteAsset)
			// quoteAmountPrecision is the minimum quote order amount (measured live for METALUSDT:
			// "1" USDT), not a price step; the price tick is 10^-quotePrecision (quotePrecision=5 =>
			// 0.00001) and the base amount step is 10^-baseAssetPrecision. The previous mapping put
			// the min-notional value ("1") into both the price and quote step, quantizing prices to
			// whole units.
			l[a] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, assetType, pair.Format(pairFormat)),
				PriceStepIncrementSize:  math.Pow(10, -result.Symbols[a].QuotePrecision),
				AmountStepIncrementSize: math.Pow(10, -result.Symbols[a].BaseAssetPrecision),
				QuoteStepIncrementSize:  math.Pow(10, -result.Symbols[a].QuoteAssetPrecision),
				MinimumQuoteAmount:      result.Symbols[a].QuoteAmountPrecision.Float64(),
				MinNotional:             result.Symbols[a].QuoteAmountPrecision.Float64(),
				MaximumQuoteAmount:      result.Symbols[a].MaxQuoteAmount.Float64(),
				MinimumBaseAmount:       result.Symbols[a].BaseSizePrecision.Float64(),
			}
			for _, f := range result.Symbols[a].Filters {
				if f.FilterType == filterPercentPriceBySide {
					l[a].MultiplierUp = 1 + f.BidMultiplierUp.Float64()
					l[a].MultiplierDown = 1 - f.AskMultiplierDown.Float64()
				}
			}
		}
		if err := limits.Load(l); err != nil {
			return err
		}
	case asset.Futures:
		contracts, err := e.GetFuturesContracts(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		pairFormat, err := e.GetPairFormat(assetType, false)
		if err != nil {
			return err
		}
		l := make([]limits.MinMaxLevel, 0, len(contracts))
		for _, c := range contracts {
			pair, err := currency.NewPairDelimiter(c.Symbol, currency.UnderscoreDelimiter)
			if err != nil {
				return err
			}
			// Amounts are in contracts. A limit order may be larger than a market order, so the limit order cap is
			// the maximum and the market order cap MarketMaxQty
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, assetType, pair.Format(pairFormat)),
				PriceStepIncrementSize:  c.PriceUnit,
				AmountStepIncrementSize: c.VolumeUnit,
				MinimumBaseAmount:       c.MinVolume,
				MaximumBaseAmount:       max(c.LimitMaxVolume, c.MaxVolume),
				MarketMinQty:            c.MinVolume,
				MarketMaxQty:            c.MaxVolume,
				MarketStepIncrementSize: c.VolumeUnit,
				MultiplierDecimal:       c.ContractSize,
				Listed:                  c.OpeningTime.Time(),
				Delisting:               c.DeliveryTime.Time(),
			})
		}
		if err := limits.Load(l); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	return nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	switch a {
	case asset.Spot:
		if _, err := e.CurrencyPairs.IsPairEnabled(cp, a); err != nil {
			return "", err
		}
		cp.Delimiter = currency.UnderscoreDelimiter
		return tradeBaseURL + "exchange/" + cp.Upper().String(), nil
	case asset.Futures:
		if _, err := e.CurrencyPairs.IsPairEnabled(cp, a); err != nil {
			return "", err
		}
		cp.Delimiter = currency.UnderscoreDelimiter
		return tradeBaseURL + "futures/" + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}
}

// orderStatusFromString converts a MEXC order status into the common order.Status.
//
// MEXC reports "PARTIALLY_CANCELED", a spelling the shared order.StringToOrderStatus parser does
// not know. That spelling belongs to this venue's adapter rather than to the shared parser:
// extending the shared vocabulary would change behaviour for every exchange for the sake of one.
// All other spellings are delegated to the shared parser unchanged.
func orderStatusFromString(status string) (order.Status, error) {
	if status == "PARTIALLY_CANCELED" {
		return order.PartiallyCancelled, nil
	}
	return order.StringToOrderStatus(status)
}

// futuresOrderSide returns the futures side of an order: a buy opens a long position, or closes a short one when it
// only reduces, and a sell opens a short position, or closes a long one
func futuresOrderSide(side order.Side, reduceOnly bool) (uint8, error) {
	switch {
	case side.IsLong() && reduceOnly:
		return FuturesSideCloseShort, nil
	case side.IsLong():
		return FuturesSideOpenLong, nil
	case side.IsShort() && reduceOnly:
		return FuturesSideCloseLong, nil
	case side.IsShort():
		return FuturesSideOpenShort, nil
	}
	return 0, fmt.Errorf("%w: %v", order.ErrSideIsInvalid, side)
}

// futuresSideType returns the side type the venue names a futures side by: buy for opening a long or closing a short,
// and sell for opening a short or closing a long. An unknown side has none
func futuresSideType(side uint8) string {
	switch side {
	case FuturesSideOpenLong, FuturesSideCloseShort:
		return FuturesSideTypeBuy
	case FuturesSideOpenShort, FuturesSideCloseLong:
		return FuturesSideTypeSell
	}
	return ""
}

// orderSideFromFuturesSide returns an order's side and whether it only reduces a position
func orderSideFromFuturesSide(side uint8) (order.Side, bool) {
	switch side {
	case FuturesSideOpenLong:
		return order.Buy, false
	case FuturesSideCloseShort:
		return order.Buy, true
	case FuturesSideOpenShort:
		return order.Sell, false
	case FuturesSideCloseLong:
		return order.Sell, true
	}
	return order.UnknownSide, false
}

// futuresPositionSide returns the side of a position type
func futuresPositionSide(positionType uint8) order.Side {
	switch positionType {
	case FuturesPositionTypeLong:
		return order.Long
	case FuturesPositionTypeShort:
		return order.Short
	}
	return order.UnknownSide
}

// futuresOrderTypeFromOrderTypeAndTimeInForce returns the futures order type of an order. The time in force is
// carried by the order type, so a limit order which must not rest is sent as an IOC or FOK order
func futuresOrderTypeFromOrderTypeAndTimeInForce(oType order.Type, tif order.TimeInForce) (uint8, error) {
	switch oType {
	case order.Limit:
		// A limit order rests until cancelled anyway, so a GTC flag beside the one constraint the venue carries is
		// harmless; any other flag would be dropped on the wire
		switch tif &^ order.GoodTillCancel {
		case order.UnknownTIF:
			return FuturesOrderTypeLimit, nil
		case order.PostOnly:
			return FuturesOrderTypePostOnly, nil
		case order.ImmediateOrCancel:
			return FuturesOrderTypeImmediateOrCancel, nil
		case order.FillOrKill:
			return FuturesOrderTypeFillOrKill, nil
		}
		return 0, fmt.Errorf("%w %q for a limit order", order.ErrUnsupportedTimeInForce, tif)
	case order.Market:
		switch tif {
		case order.UnknownTIF, order.GoodTillCancel, order.ImmediateOrCancel:
			return FuturesOrderTypeMarket, nil
		}
		return 0, fmt.Errorf("%w %q for a market order", order.ErrUnsupportedTimeInForce, tif)
	}
	return 0, fmt.Errorf("%w %q with %w", order.ErrUnsupportedOrderType, oType, order.ErrUnsupportedTimeInForce)
}

// orderTypeFromFuturesOrder returns an order's type and time in force from its futures order type and category.
// Orders the venue places to liquidate a position are reported as liquidations whatever their type
func orderTypeFromFuturesOrder(orderType, category uint8) (order.Type, order.TimeInForce) {
	switch category {
	case FuturesOrderCategoryLiquidationCustody, FuturesOrderCategoryCustodyClose, FuturesOrderCategoryLiquidationReduction:
		return order.Liquidation, order.UnknownTIF
	}
	switch orderType {
	case FuturesOrderTypeLimit:
		return order.Limit, order.GoodTillCancel
	case FuturesOrderTypePostOnly:
		return order.Limit, order.PostOnly
	case FuturesOrderTypeImmediateOrCancel:
		return order.Limit, order.ImmediateOrCancel
	case FuturesOrderTypeFillOrKill:
		return order.Limit, order.FillOrKill
	case FuturesOrderTypeMarket, FuturesOrderTypeMarketToLimit:
		return order.Market, order.UnknownTIF
	}
	return order.UnknownType, order.UnknownTIF
}

// orderStatusFromFuturesState returns an order's status from its futures state and filled volume
func orderStatusFromFuturesState(state uint8, dealVolume float64) order.Status {
	switch state {
	case FuturesOrderStatePending:
		return order.New
	case FuturesOrderStateUnfilled:
		if dealVolume > 0 {
			return order.PartiallyFilled
		}
		return order.New
	case FuturesOrderStateFilled:
		return order.Filled
	case FuturesOrderStateCancelled:
		if dealVolume > 0 {
			return order.PartiallyCancelled
		}
		return order.Cancelled
	case FuturesOrderStateInvalid:
		return order.Rejected
	}
	return order.UnknownStatus
}

// futuresOpenTypeFromMarginType returns the open type of a margin type; an unset margin type is isolated, which puts
// only the position's own margin at risk
func futuresOpenTypeFromMarginType(m margin.Type) (uint8, error) {
	switch m {
	case margin.Isolated, margin.Unset:
		return FuturesOpenTypeIsolated, nil
	case margin.Multi:
		return FuturesOpenTypeCross, nil
	}
	return 0, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, m)
}

// futuresMarginType returns the margin type of an open type
func futuresMarginType(openType uint8) margin.Type {
	switch openType {
	case FuturesOpenTypeIsolated:
		return margin.Isolated
	case FuturesOpenTypeCross:
		return margin.Multi
	}
	return margin.Unset
}

// futuresBalance returns a futures account balance. The venue reports what is available apart from what open orders
// freeze and what positions hold as margin, so both count as held; the total leaves out unrealised PNL, which the
// venue's equity counts, so that it moves with funds rather than with the mark price
func futuresBalance(available, frozen, positionMargin float64) accounts.Balance {
	return accounts.Balance{
		Free:  available,
		Hold:  frozen + positionMargin,
		Total: available + frozen + positionMargin,
	}
}

// futuresSymbolPair returns the available pair of a futures symbol, or the pair the symbol names when it is not
// available, so that orders on a contract which has left the available pairs are still reported
func (e *Exchange) futuresSymbolPair(symbol string) (currency.Pair, error) {
	if pair, err := e.MatchSymbolWithAvailablePairs(symbol, asset.Futures, true); err == nil {
		return pair, nil
	}
	pair, err := currency.NewPairDelimiter(symbol, currency.UnderscoreDelimiter)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	pairFormat, err := e.GetPairFormat(asset.Futures, false)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	return pair.Format(pairFormat), nil
}

// futuresOrderDetail returns a futures order as an order detail. Amounts are in contracts, and the cost is in the
// contract's quote currency
func (e *Exchange) futuresOrderDetail(ctx context.Context, o *FuturesOrderResponse, fallback currency.Pair) (order.Detail, error) {
	pair, err := e.futuresSymbolPair(o.Symbol)
	if err != nil {
		if fallback.IsEmpty() {
			return order.Detail{}, err
		}
		pair = fallback
	}
	oType, tif := orderTypeFromFuturesOrder(o.OrderType, o.Category)
	side, reduceOnly := orderSideFromFuturesSide(o.Side)
	lastUpdated := o.UpdateTime.Time()
	if lastUpdated.IsZero() {
		lastUpdated = o.CreateTime.Time()
	}
	detail := order.Detail{
		Exchange:             e.Name,
		OrderID:              o.OrderID.String(),
		ClientOrderID:        o.ExternalOrderID,
		Pair:                 pair,
		AssetType:            asset.Futures,
		Price:                o.Price,
		Amount:               o.Volume,
		ContractAmount:       o.Volume,
		ExecutedAmount:       o.DealVolume,
		RemainingAmount:      o.Volume - o.DealVolume,
		AverageExecutedPrice: o.DealAveragePrice,
		Fee:                  o.TakerFee + o.MakerFee,
		FeeAsset:             o.FeeCurrency,
		Leverage:             float64(o.Leverage),
		MarginType:           futuresMarginType(o.OpenType),
		ReduceOnly:           reduceOnly || o.ReduceOnly,
		Type:                 oType,
		TimeInForce:          tif,
		Side:                 side,
		Status:               orderStatusFromFuturesState(o.State, o.DealVolume),
		Date:                 o.CreateTime.Time(),
		LastUpdated:          lastUpdated,
	}
	if o.DealVolume > 0 && o.DealAveragePrice > 0 {
		contract, err := e.futuresContract(ctx, o.Symbol)
		if err != nil {
			// The cost only enriches the order, so a contract the venue no longer lists does not sink it
			log.Warnf(log.ExchangeSys, "%s: cost unknown for futures order %s: %v", e.Name, detail.OrderID, err)
			return detail, nil
		}
		detail.Cost, detail.CostAsset = futuresQuoteCost(contract, o.DealVolume, o.DealAveragePrice), contract.QuoteCoin
	}
	return detail, nil
}

// futuresCost returns what contracts filled at a price are worth in the contract's settlement currency
func futuresCost(c *FuturesContractResponse, contracts, price float64) float64 {
	if c.IsInverse() {
		return contracts * c.ContractSize / price
	}
	return contracts * c.ContractSize * price
}

// futuresQuoteCost returns what contracts filled at a price cost in the contract's quote currency, the currency an
// order's cost is reported in; an inverse contract's size is counted in it already
func futuresQuoteCost(c *FuturesContractResponse, contracts, price float64) float64 {
	if c.IsInverse() {
		return contracts * c.ContractSize
	}
	return contracts * c.ContractSize * price
}

// futuresTradeHistory returns futures fills as trade history; amounts are in contracts
func futuresTradeHistory(exchangeName string, deals []*FuturesOrderDealResponse) []order.TradeHistory {
	trades := make([]order.TradeHistory, 0, len(deals))
	for _, d := range deals {
		if d == nil {
			continue
		}
		side, _ := orderSideFromFuturesSide(d.Side)
		trades = append(trades, order.TradeHistory{
			Price:     d.Price,
			Amount:    d.Volume,
			Fee:       d.Fee,
			FeeAsset:  d.FeeCurrency.String(),
			Exchange:  exchangeName,
			TID:       d.ID.String(),
			Side:      side,
			Timestamp: d.Timestamp.Time(),
			IsMaker:   !d.WasTaker(),
		})
	}
	return trades
}

// futuresContractsFor returns the details of the contracts behind tickers, reading every contract's once when one is
// not yet known
func (e *Exchange) futuresContractsFor(ctx context.Context, tickers []*FuturesTickerResponse) (map[string]*FuturesContractResponse, error) {
	snapshot := func() (map[string]*FuturesContractResponse, bool) {
		e.futuresContractsMu.RLock()
		defer e.futuresContractsMu.RUnlock()
		contracts := make(map[string]*FuturesContractResponse, len(e.futuresContracts))
		complete := true
		for _, t := range tickers {
			if t == nil {
				continue
			}
			c, ok := e.futuresContracts[t.Symbol]
			if !ok {
				complete = false
				continue
			}
			contracts[t.Symbol] = c
		}
		return contracts, complete
	}
	if contracts, complete := snapshot(); complete {
		return contracts, nil
	}
	if _, err := e.GetFuturesContracts(ctx, currency.EMPTYPAIR); err != nil {
		return nil, err
	}
	contracts, _ := snapshot()
	return contracts, nil
}
