package bingx

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Request limits the wrapper pages by: the most candles each product's candle endpoint returns a request, the deepest
// REST order book each product documents, the most trades each trade endpoint returns, and the most deposit,
// withdrawal or deposit address records a request returns
const (
	spotCandleLimit           = 1440
	swapCandleLimit           = 1440
	coinMarginedCandleLimit   = 1000
	orderbookDepth            = 1000
	spotRecentTradesLimit     = 500
	swapRecentTradesLimit     = 1000
	spotHistoricalTradesLimit = 1000
	fundingRecordsPageLimit   = 1000
	depositAddressPageLimit   = 1000
)

const (
	// coinMarginedContractStatusOnline is the status of a coin-M contract open for trading
	coinMarginedContractStatusOnline = 1
	// offlineSpotTradeFeeRate is BingX's standard spot commission rate, for makers and takers alike
	offlineSpotTradeFeeRate = 0.001

	depositTransferType    = "deposit"
	withdrawalTransferType = "withdrawal"
	internalTransferType   = "internal transfer"
)

var (
	errCandlesBeforeHistory = errors.New("range starts before the candles BingX holds")
	errTradesBeforeHistory  = errors.New("range starts before the trades BingX holds")
	errTradePagingStalled   = errors.New("trade paging made no progress")
	errDefaultNetworkAbsent = errors.New("default network not found")
)

// SetDefaults sets the basic defaults for BingX
func (e *Exchange) SetDefaults() {
	e.Name = "BingX"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	pairFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		if err := e.SetAssetPairStore(a, currency.PairStore{AssetEnabled: true, RequestFormat: pairFormat, ConfigFormat: pairFormat}); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:                 true,
				TickerFetching:                 true,
				KlineFetching:                  true,
				TradeFetching:                  true,
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
				TradeFee:                       true,
				CryptoWithdrawalFee:            true,
				MultiChainDeposits:             true,
				MultiChainWithdrawals:          true,
				HasAssetTypeAccountSegregation: true,
				FundingRateFetching:            true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				Subscribe:              true,
				Unsubscribe:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto | exchange.NoFiatWithdrawals,
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
				// Three day candles are built from daily ones, as USDT-M three day candles open on another grid than GCT's
				Intervals: kline.DeployExchangeIntervals(
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
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				// The smallest of the products' candle limits, so a single request suits every product
				GlobalResultLimit: coinMarginedCandleLimit,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
		TradingRequirements: protocol.TradingRequirements{
			SpotMarketBuyQuotation: true,
			SpotMarketSellBase:     true,
		},
	}

	var err error
	e.Requester, err = request.New(e.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(GetRateLimits()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              apiURL,
		exchange.RestUSDTMargined:      apiURL,
		exchange.RestCoinMargined:      apiURL,
		exchange.WebsocketSpot:         spotWebsocketURL,
		exchange.WebsocketUSDTMargined: usdtMarginedWebsocketURL,
		exchange.WebsocketCoinMargined: coinMarginedWebsocketURL,
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
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		TradeFeed:                              e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement:           true,
		MaxWebsocketSubscriptionsPerConnection: websocketMaxSubscriptionsPerConnection,
		// Subscription requests and heartbeat replies draw on budgets shared by every connection
		RateLimitDefinitions: e.Requester.GetRateLimiterDefinitions(),
	}); err != nil {
		return err
	}
	setups, err := e.wsConnectionSetups(exch)
	if err != nil {
		return err
	}
	for _, setup := range setups {
		if err := e.Websocket.SetupNewConnection(setup); err != nil {
			return err
		}
	}
	return nil
}

// FetchTradablePairs returns the tradable pairs of an asset: spot symbols online with buying or selling open to the
// API, USDT- and USDC-settled perpetuals online and open to the API, and coin-M perpetuals online
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	var symbols []string
	switch a {
	case asset.Spot:
		resp, err := e.GetSpotSymbols(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		for i := range resp {
			if resp[i].Status == SpotSymbolStatusOnline && (resp[i].APIStateBuy || resp[i].APIStateSell) {
				symbols = append(symbols, resp[i].Symbol)
			}
		}
	case asset.USDTMarginedFutures:
		resp, err := e.GetSwapContracts(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		for i := range resp {
			if resp[i].Status == SwapContractStatusOnline && resp[i].APIStateOpen.Bool() {
				symbols = append(symbols, resp[i].Symbol)
			}
		}
	case asset.CoinMarginedFutures:
		resp, err := e.GetCoinMarginedContracts(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		for i := range resp {
			if resp[i].Status == coinMarginedContractStatusOnline {
				symbols = append(symbols, resp[i].Symbol)
			}
		}
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	format, err := e.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	pairs := make(currency.Pairs, 0, len(symbols))
	for _, symbol := range symbols {
		pair, err := symbolPair(symbol, format)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s skipping %s symbol: %s", e.Name, a, err)
			continue
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// symbolPair returns the pair of a BingX symbol in format, split at its last dash, as a few spot bases contain one
func symbolPair(symbol string, format currency.PairFormat) (currency.Pair, error) {
	base, quote, err := wsSplitSymbol(symbol)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	return currency.NewPairWithDelimiter(base, quote, format.Delimiter).Format(format), nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
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
	if err := e.EnsureOnePairEnabled(); err != nil && errs == nil {
		return err
	}
	return errs
}

// availablePair returns the available pair of a symbol an endpoint listed, and false for a symbol without one, as
// listings include symbols that are not tradable
func (e *Exchange) availablePair(symbol string, a asset.Item) (currency.Pair, bool, error) {
	base, quote, err := wsSplitSymbol(symbol)
	var pair currency.Pair
	if err == nil {
		pair, err = e.MatchSymbolWithAvailablePairs(base+quote, a, false)
	}
	switch {
	case err == nil:
		return pair, true, nil
	case errors.Is(err, errWebsocketInvalidSymbol), errors.Is(err, currency.ErrPairNotFound):
		// FetchTradablePairs skips a symbol it cannot split, so such a symbol has no available pair either
		return currency.EMPTYPAIR, false, nil
	default:
		return currency.EMPTYPAIR, false, err
	}
}

// UpdateTickers updates the tickers of every available pair of an asset from one request for every symbol's ticker. A
// ticker that cannot be converted or stored is reported, and the others are stored
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	var prices []ticker.Price
	var errs error
	switch a {
	case asset.Spot:
		resp, err := e.GetSpotTicker(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		prices = make([]ticker.Price, 0, len(resp))
		for i := range resp {
			pair, ok, err := e.availablePair(resp[i].Symbol, a)
			if err != nil {
				return err
			}
			if ok {
				prices = append(prices, e.spotTickerPrice(&resp[i], pair))
			}
		}
	case asset.USDTMarginedFutures:
		resp, err := e.GetSwap24HourTicker(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		prices = make([]ticker.Price, 0, len(resp))
		for i := range resp {
			pair, ok, err := e.availablePair(resp[i].Symbol, a)
			if err != nil {
				return err
			}
			if ok {
				prices = append(prices, e.swapTickerPrice(&resp[i], pair))
			}
		}
	case asset.CoinMarginedFutures:
		resp, err := e.GetCoinMargined24HourPriceChange(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		prices = make([]ticker.Price, 0, len(resp))
		for i := range resp {
			pair, ok, err := e.availablePair(resp[i].Symbol, a)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			prices = append(prices, e.coinMarginedTickerPrice(&resp[i], pair))
		}
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	_, err := ticker.ProcessBatch(prices)
	return common.AppendError(errs, err)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var price ticker.Price
	switch a {
	case asset.Spot:
		resp, err := e.GetSpotTicker(ctx, p)
		if err != nil {
			return nil, err
		}
		if len(resp) == 0 {
			return nil, fmt.Errorf("%w: %s %s", ticker.ErrTickerNotFound, a, p)
		}
		price = e.spotTickerPrice(&resp[0], p)
	case asset.USDTMarginedFutures:
		resp, err := e.GetSwap24HourTicker(ctx, p)
		if err != nil {
			return nil, err
		}
		if len(resp) == 0 {
			return nil, fmt.Errorf("%w: %s %s", ticker.ErrTickerNotFound, a, p)
		}
		price = e.swapTickerPrice(&resp[0], p)
	case asset.CoinMarginedFutures:
		resp, err := e.GetCoinMargined24HourPriceChange(ctx, p)
		if err != nil {
			return nil, err
		}
		if len(resp) == 0 {
			return nil, fmt.Errorf("%w: %s %s", ticker.ErrTickerNotFound, a, p)
		}
		price = e.coinMarginedTickerPrice(&resp[0], p)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if err := ticker.ProcessTicker(&price); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// spotTickerPrice converts a spot 24 hour ticker, which reports no last trade size
func (e *Exchange) spotTickerPrice(t *SpotTickerResponse, pair currency.Pair) ticker.Price {
	return ticker.Price{
		Last:                t.LastPrice,
		High:                t.HighPrice,
		Low:                 t.LowPrice,
		Bid:                 t.BidPrice,
		BidSize:             t.BidQuantity,
		Ask:                 t.AskPrice,
		AskSize:             t.AskQuantity,
		BaseVolume:          t.Volume,
		QuoteVolume:         t.QuoteVolume,
		Open:                t.OpenPrice,
		PercentChange24Hour: t.PriceChangePercent.Float64(),
		Pair:                pair,
		ExchangeName:        e.Name,
		AssetType:           asset.Spot,
		LastUpdated:         t.CloseTime.Time(),
	}
}

// swapTickerPrice converts a USDT-M perpetual 24 hour ticker
func (e *Exchange) swapTickerPrice(t *Swap24HourTickerResponse, pair currency.Pair) ticker.Price {
	return ticker.Price{
		Last:                t.LastPrice.Float64(),
		LastSize:            t.LastQuantity.Float64(),
		High:                t.HighPrice.Float64(),
		Low:                 t.LowPrice.Float64(),
		Bid:                 t.BidPrice.Float64(),
		BidSize:             t.BidQuantity.Float64(),
		Ask:                 t.AskPrice.Float64(),
		AskSize:             t.AskQuantity.Float64(),
		BaseVolume:          t.Volume.Float64(),
		QuoteVolume:         t.QuoteVolume.Float64(),
		Open:                t.OpenPrice.Float64(),
		PercentChange24Hour: t.PriceChangePercent.Float64(),
		Pair:                pair,
		ExchangeName:        e.Name,
		AssetType:           asset.USDTMarginedFutures,
		LastUpdated:         t.CloseTime.Time(),
	}
}

// coinMarginedTickerPrice converts a coin-M perpetual 24 hour ticker. Its sizes count contracts, as the ticker stream's
// do, and its base volume is the base coin volume, which BingX names quoteVolume; the USD volume the ticker stream
// reports is absent here
func (e *Exchange) coinMarginedTickerPrice(t *CoinMargined24HourPriceChangeResponse, pair currency.Pair) ticker.Price {
	return ticker.Price{
		Last:                t.LastPrice.Float64(),
		LastSize:            t.LastQuantity.Float64(),
		High:                t.HighPrice.Float64(),
		Low:                 t.LowPrice.Float64(),
		Bid:                 t.BidPrice.Float64(),
		BidSize:             t.BidQuantity.Float64(),
		Ask:                 t.AskPrice.Float64(),
		AskSize:             t.AskQuantity.Float64(),
		BaseVolume:          t.QuoteVolume.Float64(),
		Open:                t.OpenPrice.Float64(),
		PercentChange24Hour: t.PriceChangePercent.Float64(),
		Pair:                pair,
		ExchangeName:        e.Name,
		AssetType:           asset.CoinMarginedFutures,
		LastUpdated:         t.CloseTime.Time(),
	}
}

// UpdateOrderbook updates and returns the orderbook for a currency pair, at the deepest REST depth BingX documents.
// Coin-M quantities count contracts, as the order book stream's do
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	switch a {
	case asset.Spot:
		resp, err := e.GetSpotOrderBook(ctx, p, orderbookDepth)
		if err != nil {
			return nil, err
		}
		book.Bids, book.Asks = resp.Bids.Levels(), resp.Asks.Levels()
		book.LastUpdated = resp.Timestamp.Time()
		book.LastUpdateID = int64(min(resp.LastUpdateID, math.MaxInt64))
	case asset.USDTMarginedFutures:
		resp, err := e.GetSwapOrderBook(ctx, p, orderbookDepth)
		if err != nil {
			return nil, err
		}
		book.Bids, book.Asks = resp.Bids.Levels(), resp.Asks.Levels()
		book.LastUpdated = resp.Time.Time()
	case asset.CoinMarginedFutures:
		resp, err := e.GetCoinMarginedOrderBook(ctx, p, orderbookDepth)
		if err != nil {
			return nil, err
		}
		book.Bids, book.Asks = resp.Bids.Levels(), resp.Asks.Levels()
		book.LastUpdated = resp.Time.Time()
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	// Spot and coin-M asks arrive highest price first
	wsSortLevels(book.Bids, book.Asks)
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, a)
}

// UpdateAccountBalances retrieves currency balances. A spot asset's free amount is free and its locked amount, what
// open orders hold, is held, both together being the total. A perpetual asset's wallet balance is the total, the
// margin its positions use and its open orders freeze is held, and its available margin is free. The accounts store
// stamps a snapshot with its arrival, as it does the account streams' updates, since neither carries a time to order
// them by, so balances a stream event stores while the request is under way may be newer than its reply: the balances
// are then requested again, balanceSnapshotAttempts times in all, and a reply each request of which a stream event
// overtook is returned with errBalanceSnapshotSuperseded, without being stored. The check and the save hold the lock a
// stream event's writes hold, so the writes land wholly before the check or wholly after the save. A reply whose
// request started before that of the snapshot last stored is not stored either, so a slow request cannot replace a
// newer snapshot, and the stored balances, which are newer, are returned instead. Snapshots and stream writes are
// ordered for each account, which context credentials can switch per call, as the accounts store keeps each account's
// balances apart, and a snapshot requests, orders and stores the balances of the credentials ctx names when it starts,
// although the exchange's default credentials change meanwhile
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	ctx, k, err := e.accountBalanceContext(ctx, a)
	if err != nil {
		return nil, err
	}
	for attempt := 1; ; attempt++ {
		ticket, generation := e.startBalanceSnapshot(&k)
		subAccount, err := e.accountBalances(ctx, a)
		if err != nil {
			return nil, err
		}
		subAccounts := accounts.SubAccounts{subAccount}
		if reported, done, err := e.saveBalanceSnapshot(ctx, &k, ticket, generation, subAccounts); done {
			return reported, err
		}
		if attempt == balanceSnapshotAttempts {
			return subAccounts, fmt.Errorf("%w: stream balance updates overtook every %s request", errBalanceSnapshotSuperseded, a)
		}
	}
}

// accountBalances returns an asset's balances as BingX's REST balance endpoints report them
func (e *Exchange) accountBalances(ctx context.Context, a asset.Item) (*accounts.SubAccount, error) {
	subAccount := accounts.NewSubAccount(a, "")
	switch a {
	case asset.Spot:
		resp, err := e.GetSpotAccountAssets(ctx)
		if err != nil {
			return nil, err
		}
		for i := range resp.Balances {
			free, locked := resp.Balances[i].Free.Float64(), resp.Balances[i].Locked.Float64()
			subAccount.Balances.Set(resp.Balances[i].Asset, accounts.Balance{
				Total: free + locked,
				Hold:  locked,
				Free:  free,
			})
		}
	case asset.USDTMarginedFutures:
		resp, err := e.GetSwapAccountData(ctx)
		if err != nil {
			return nil, err
		}
		for i := range resp {
			subAccount.Balances.Set(resp[i].Asset, perpetualBalance(resp[i].Balance, resp[i].UsedMargin, resp[i].FreezedMargin, resp[i].AvailableMargin))
		}
	case asset.CoinMarginedFutures:
		resp, err := e.GetCoinMarginedAccountAssets(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		for i := range resp {
			subAccount.Balances.Set(resp[i].Asset, perpetualBalance(resp[i].Balance, resp[i].UsedMargin, resp[i].FreezedMargin, resp[i].AvailableMargin))
		}
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	return subAccount, nil
}

// perpetualBalance returns a perpetual account asset's balance: the wallet balance is the total, the margin positions
// use and open orders freeze is held, and the available margin is free
func perpetualBalance(balance, usedMargin, frozenMargin, availableMargin types.Number) accounts.Balance {
	return accounts.Balance{
		Total: balance.Float64(),
		Hold:  usedMargin.Float64() + frozenMargin.Float64(),
		Free:  availableMargin.Float64(),
	}
}

// Balance snapshots and refreshes: a snapshot is requested at most balanceSnapshotAttempts times while stream events
// store balances during its requests, and the refreshes perpetual balance events ask for run at most once each
// balanceRefreshInterval
const (
	balanceSnapshotAttempts = 3
	balanceRefreshInterval  = time.Second
)

var errBalanceSnapshotSuperseded = errors.New("newer balances superseded the balance snapshot")

// balanceState orders the stream's balance writes and REST balance snapshots, and runs the REST refreshes perpetual
// balance events and account connections ask for, one at a time for each account's asset
type balanceState struct {
	// writes is held across a stream event's balance writes and across a snapshot's check and save, so neither lands
	// inside the other
	writes sync.Mutex
	// generations counts each account asset's stream balance writes, which a snapshot requested before one must not
	// overwrite
	generations map[balanceKey]uint64
	// tickets numbers each account asset's snapshot requests as they start, and saved holds the ticket of the snapshot
	// last stored, which a reply to an earlier request must not overwrite
	tickets map[balanceKey]uint64
	saved   map[balanceKey]uint64

	mu sync.Mutex
	// running marks an account asset whose refresh runs, and pending holds the refresh the latest event that changed
	// its balances again while it ran asked for
	running map[balanceKey]bool
	pending map[balanceKey]balanceRefresh
	// interval is the least time between an account asset's refreshes, balanceRefreshInterval when zero
	interval time.Duration
}

// balanceKey is an account's asset in the balance state, the account named by the whole of its credentials, as the
// accounts store keeps the balances of credentials differing in any value apart
type balanceKey struct {
	credentials accounts.Credentials
	asset       asset.Item
}

// balanceRefresh is a balance refresh an event asked for: the event's context, carrying the credentials it named when
// the event arrived, which name the account, and the websocket session of the connection the event arrived on
type balanceRefresh struct {
	ctx     context.Context //nolint:containedctx // A pending refresh runs with the credentials of the event that asked for it
	session <-chan struct{}
}

// accountBalanceContext returns ctx carrying the credentials it names now, and the balance state key of their account's
// asset a. A snapshot, refresh or stream write runs with the context returned, so its requests, its key and its save
// all name one account although the exchange's default credentials change meanwhile
func (e *Exchange) accountBalanceContext(ctx context.Context, a asset.Item) (context.Context, balanceKey, error) {
	switch a {
	case asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures:
	default:
		return nil, balanceKey{}, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, balanceKey{}, err
	}
	if creds.IsEmpty() {
		// The accounts store keeps no balances for empty credentials, and looks up every account's for them
		return nil, balanceKey{}, fmt.Errorf("%s %w", e.Name, exchange.ErrCredentialsAreEmpty)
	}
	return accounts.DeployCredentialsToContext(ctx, creds), balanceKey{credentials: *creds, asset: a}, nil
}

// startBalanceSnapshot returns the ticket of an account asset's snapshot request starting now and the count of its
// stream balance writes so far
func (e *Exchange) startBalanceSnapshot(k *balanceKey) (ticket, generation uint64) {
	b := &e.balances
	b.writes.Lock()
	defer b.writes.Unlock()
	if b.tickets == nil {
		b.tickets = make(map[balanceKey]uint64)
	}
	b.tickets[*k]++
	return b.tickets[*k], b.generations[*k]
}

// saveBalanceSnapshot stores the reply to the snapshot request startBalanceSnapshot issued ticket and generation, and
// returns the balances the request reports and whether it is done. A reply passed to the accounts store is returned
// with the store's error, and counts as stored whatever the store returned, as the store can save balances and then
// fail to publish them. A reply whose request started before that of the snapshot last stored is not stored, and the
// stored balances, which are newer, are returned instead. A reply a stream balance write overtook is left for the
// snapshot to be requested again. Balances are stored and looked up for the credentials k names
func (e *Exchange) saveBalanceSnapshot(ctx context.Context, k *balanceKey, ticket, generation uint64, subAccounts accounts.SubAccounts) (accounts.SubAccounts, bool, error) {
	b := &e.balances
	b.writes.Lock()
	defer b.writes.Unlock()
	if ticket < b.saved[*k] {
		stored, err := e.Accounts.SubAccounts(&k.credentials, k.asset)
		return stored, true, err
	}
	if b.generations[*k] != generation {
		return nil, false, nil
	}
	err := e.Accounts.Save(accounts.DeployCredentialsToContext(ctx, &k.credentials), subAccounts, true)
	if b.saved == nil {
		b.saved = make(map[balanceKey]uint64)
	}
	b.saved[*k] = ticket
	return subAccounts, true, err
}

// refreshBalances refreshes the balances of ctx's account's asset over REST after a stream event changed them or a
// perpetual account connection came up, or has the refresh running for them go once more after it, so a burst of
// events costs one refresh more. A refresh runs for session, the websocket session of the connection the event arrived
// on, and ends with it, while a refresh the event leaves pending runs with the event's context for the session of the
// event, even once the running refresh's own session has ended
func (e *Exchange) refreshBalances(ctx context.Context, a asset.Item, session <-chan struct{}) {
	if session == nil {
		return
	}
	select {
	case <-session:
		return
	default:
	}
	ctx, k, err := e.accountBalanceContext(ctx, a)
	if err != nil {
		log.Errorf(log.WebsocketMgr, "%s %s balance refresh failed: %v", e.Name, a, err)
		return
	}
	// The refresh outlives the event, whose context ends with it
	refresh := balanceRefresh{ctx: context.WithoutCancel(ctx), session: session}
	b := &e.balances
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running[k] {
		b.pending[k] = refresh
		return
	}
	if b.running == nil {
		b.running = make(map[balanceKey]bool)
		b.pending = make(map[balanceKey]balanceRefresh)
	}
	b.running[k] = true
	interval := cmp.Or(b.interval, balanceRefreshInterval)
	e.wsState.background.Go(func() { e.runBalanceRefresh(&k, refresh, interval) })
}

// runBalanceRefresh runs refresh, a refresh of an account asset's balances, then once more after each interval in which
// another event changed them, with the context of the latest. Once the refresh's session ends, a refresh an event of a
// newer session left pending runs for that session, and one an ended session left pending is dropped
func (e *Exchange) runBalanceRefresh(k *balanceKey, refresh balanceRefresh, interval time.Duration) {
	b := &e.balances
	for {
		e.updateSessionBalances(refresh.ctx, k.asset, refresh.session)
		select {
		case <-time.After(interval):
		case <-refresh.session:
		}
		b.mu.Lock()
		next, ok := b.pending[*k]
		delete(b.pending, *k)
		if ok {
			// A pending refresh whose session has ended is dropped
			select {
			case <-next.session:
				ok = false
			default:
			}
		}
		if !ok {
			delete(b.running, *k)
		}
		b.mu.Unlock()
		if !ok {
			return
		}
		refresh = next
	}
}

// updateSessionBalances refreshes an asset's balances with a request that ends once session does, logging a failure
// while session lasts; one once it has ended goes unlogged, as the next connection refreshes the balances again
func (e *Exchange) updateSessionBalances(ctx context.Context, a asset.Item, session <-chan struct{}) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	e.wsState.background.Go(func() {
		select {
		case <-session:
			cancel()
		case <-ctx.Done():
		}
	})
	_, err := e.UpdateAccountBalances(ctx, a)
	if err == nil {
		return
	}
	select {
	case <-session:
	default:
		log.Errorf(log.WebsocketMgr, "%s %s balance refresh failed: %v", e.Name, a, err)
	}
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, a asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, a)
	return e.CheckTransientError(err)
}

// GetAccountFundingHistory returns the account's deposits and withdrawals, every page of the records BingX returns
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
		d := &deposits[i]
		// Deposit records carry no record ID, so their transaction ID identifies them
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:      e.Name,
			Status:            depositStatusString(d.Status),
			TransferID:        d.TxID,
			Timestamp:         d.InsertTime.Time(),
			Currency:          d.Coin.String(),
			Amount:            d.Amount.Float64(),
			TransferType:      depositTransferType,
			CryptoToAddress:   d.Address,
			CryptoFromAddress: d.SourceAddress,
			CryptoTxID:        d.TxID,
			CryptoChain:       d.Network,
		})
	}
	for i := range withdrawals {
		w := &withdrawals[i]
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:      e.Name,
			Status:            withdrawStatusString(w.Status),
			TransferID:        w.ID,
			Description:       w.Info,
			Timestamp:         w.ApplyTime,
			Currency:          w.Coin.String(),
			Amount:            w.Amount.Float64(),
			Fee:               w.TransactionFee.Float64(),
			TransferType:      withdrawalTransferTypeString(w.TransferType),
			CryptoToAddress:   w.Address,
			CryptoFromAddress: w.SourceAddress,
			CryptoTxID:        w.TxID,
			CryptoChain:       w.Network,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns the withdrawals of a currency, or of every currency when it is empty; withdrawals
// leave the account whatever asset is asked for
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.allWithdrawals(ctx, c)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals))
	for i := range withdrawals {
		w := &withdrawals[i]
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawStatusString(w.Status),
			TransferID:      w.ID,
			Description:     w.Info,
			Timestamp:       w.ApplyTime,
			Currency:        w.Coin.String(),
			Amount:          w.Amount.Float64(),
			Fee:             w.TransactionFee.Float64(),
			TransferType:    withdrawalTransferTypeString(w.TransferType),
			CryptoToAddress: w.Address,
			CryptoTxID:      w.TxID,
			CryptoChain:     w.Network,
		}
	}
	return resp, nil
}

// allDeposits returns every deposit BingX returns, requesting pages until one is not full
func (e *Exchange) allDeposits(ctx context.Context) ([]DepositRecordResponse, error) {
	var deposits []DepositRecordResponse
	for {
		page, err := e.GetDepositRecords(ctx, &DepositRecordsRequest{Offset: uint64(len(deposits)), Limit: fundingRecordsPageLimit})
		if err != nil {
			return nil, err
		}
		deposits = append(deposits, page...)
		if len(page) < fundingRecordsPageLimit {
			return deposits, nil
		}
	}
}

// allWithdrawals returns every withdrawal of a coin, or of every coin, BingX returns, requesting pages until one is not
// full
func (e *Exchange) allWithdrawals(ctx context.Context, coin currency.Code) ([]WithdrawRecordResponse, error) {
	var withdrawals []WithdrawRecordResponse
	for {
		page, err := e.GetWithdrawRecords(ctx, &WithdrawRecordsRequest{Coin: coin, Offset: uint64(len(withdrawals)), Limit: fundingRecordsPageLimit})
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, page...)
		if len(page) < fundingRecordsPageLimit {
			return withdrawals, nil
		}
	}
}

// depositStatusString returns the documented name of a deposit status, or the code of an undocumented one
func depositStatusString(status DepositStatus) string {
	switch status {
	case DepositStatusInProgress:
		return "In progress"
	case DepositStatusCompleted:
		return "Completed"
	case DepositStatusChainUploaded:
		return "Chain uploaded"
	default:
		return strconv.FormatUint(uint64(status), 10)
	}
}

// withdrawStatusString returns the documented name of a withdrawal status, or the code of an undocumented one
func withdrawStatusString(status WithdrawStatus) string {
	switch status {
	case WithdrawStatusUnderReview:
		return "Under review"
	case WithdrawStatusFailed:
		return "Failed"
	case WithdrawStatusCompleted:
		return "Completed"
	default:
		return strconv.FormatUint(uint64(status), 10)
	}
}

// withdrawalTransferTypeString returns how a withdrawal moved: on chain, or as a transfer within BingX
func withdrawalTransferTypeString(transferType TransferType) string {
	if transferType == TransferTypeInternalTransfer {
		return internalTransferType
	}
	return withdrawalTransferType
}

// GetRecentTrades returns the most recent trades for a currency and asset; BingX serves no coin-M trades over REST
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	var resp []trade.Data
	switch a {
	case asset.Spot:
		trades, err := e.GetSpotRecentTrades(ctx, p, spotRecentTradesLimit)
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(trades))
		for i := range trades {
			resp[i] = trade.Data{
				TID:          strconv.FormatUint(trades[i].ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         spotRecentTradeSide(trades[i].BuyerMaker),
				Price:        trades[i].Price,
				Amount:       trades[i].Quantity,
				Timestamp:    trades[i].Time.Time(),
			}
		}
	case asset.USDTMarginedFutures:
		trades, err := e.GetSwapRecentTrades(ctx, p, swapRecentTradesLimit)
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(trades))
		for i := range trades {
			resp[i] = trade.Data{
				TID:          trades[i].FillID,
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         wsTakerSide(trades[i].IsBuyerMaker),
				Price:        trades[i].Price.Float64(),
				Amount:       trades[i].Quantity.Float64(),
				Timestamp:    trades[i].Time.Time(),
			}
		}
	case asset.CoinMarginedFutures:
		return nil, fmt.Errorf("%w: BingX serves no %s trades", common.ErrFunctionNotSupported, a)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return resp, nil
}

// spotRecentTradeSide returns the taker's side of a spot REST trade, whose buyerMaker is true when the taker bought,
// unlike the trade stream's m
func spotRecentTradeSide(buyerMaker bool) order.Side {
	if buyerMaker {
		return order.Buy
	}
	return order.Sell
}

// GetHistoricTrades returns the spot trades from startTime to endTime, paging back from the most recent trades until
// one older than startTime. BingX's USDT-M trade endpoints return their latest 1000 trades whatever is asked, and
// coin-M has none, so neither can serve a time range
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, startTime, endTime time.Time) ([]trade.Data, error) {
	switch a {
	case asset.Spot:
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		return nil, fmt.Errorf("%w: BingX serves no %s trades by time", common.ErrFunctionNotSupported, a)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	var resp []trade.Data
	seen := make(map[string]bool)
	var fromID string
	var oldest time.Time
	for {
		page, err := e.GetSpotHistoricalTrades(ctx, &SpotHistoricalTradesRequest{Symbol: p, Limit: spotHistoricalTradesLimit, FromID: fromID})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			if fromID == "" {
				// The symbol has no trades at all
				break
			}
			// A short page is not the end of the history, an empty one is
			return nil, fmt.Errorf("%w: %s %s trades begin at %s, after %s", errTradesBeforeHistory, a, p, oldest.UTC(), startTime.UTC())
		}
		var progressed, reachedStart bool
		for i := range page {
			t := &page[i]
			if seen[t.TradeID] {
				continue
			}
			seen[t.TradeID], progressed = true, true
			at := t.Time.Time()
			if oldest.IsZero() || at.Before(oldest) {
				oldest = at
			}
			if at.Before(startTime) {
				reachedStart = true
				continue
			}
			if at.After(endTime) {
				continue
			}
			side := order.Buy
			if t.TradeSide == SpotTradeSideSell {
				side = order.Sell
			}
			resp = append(resp, trade.Data{
				TID:          t.TradeID,
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         side,
				Price:        t.Price,
				Amount:       t.Quantity,
				Timestamp:    at,
			})
		}
		if reachedStart {
			break
		}
		if !progressed {
			return nil, fmt.Errorf("%w: %s %s before trade %s", errTradePagingStalled, a, p, fromID)
		}
		// Pages hold the newest trades first, and fromId returns the trades before the one it names
		fromID = page[len(page)-1].TradeID
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return resp, nil
}

// GetServerTime returns the current exchange server time, which every product shares
func (e *Exchange) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	switch a {
	case asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures:
	default:
		return time.Time{}, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	resp, err := e.GetSystemTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// GetHistoricCandles returns candles between a time period for a set time interval, in UTC periods. A range starting
// before the candles BingX holds is reported, rather than padded: for a USDT-M contract with a trading schedule, whose
// candles stop while its market is closed, a range starting before its launch
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	candles, err := e.candles(ctx, a, req.RequestFormatted, req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
	if err != nil {
		return nil, err
	}
	if err := e.checkCandleHistory(ctx, a, req.RequestFormatted, req.ExchangeInterval, candles, req.Start); err != nil {
		return nil, err
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval, in UTC periods, paging by
// the most candles each product's endpoint returns a request. A range starting before the candles BingX holds is
// reported, rather than padded: for a USDT-M contract with a trading schedule, whose candles stop while its market is
// closed, a range starting before its launch
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	limit, err := candleLimit(a)
	if err != nil {
		return nil, err
	}
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	if req.RangeHolder, err = req.GetRanges(limit); err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		candles, err := e.candles(ctx, a, req.RequestFormatted, req.ExchangeInterval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time, limit)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, candles...)
	}
	if err := e.checkCandleHistory(ctx, a, req.RequestFormatted, req.ExchangeInterval, timeSeries, req.Start); err != nil {
		return nil, err
	}
	return req.ProcessResponse(timeSeries)
}

// candleLimit returns the most candles a product's candle endpoint returns a request
func candleLimit(a asset.Item) (uint64, error) {
	switch a {
	case asset.Spot:
		return spotCandleLimit, nil
	case asset.USDTMarginedFutures:
		return swapCandleLimit, nil
	case asset.CoinMarginedFutures:
		return coinMarginedCandleLimit, nil
	default:
		return 0, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
}

// candles returns up to limit of an asset's candles opening from start until end, exclusive, in UTC periods. Spot
// klines include a candle opening at their end time, so a spot request ends a millisecond before end. Coin-M volumes
// count contracts, as the candle stream's do
func (e *Exchange) candles(ctx context.Context, a asset.Item, pair currency.Pair, interval kline.Interval, start, end time.Time, limit uint64) ([]kline.Candle, error) {
	var candles []kline.Candle
	switch a {
	case asset.Spot:
		resp, err := e.GetSpotKlines(ctx, &SpotKlinesRequest{
			Symbol:    pair,
			Interval:  interval,
			StartTime: start,
			EndTime:   end.Add(-time.Millisecond),
			TimeZone:  SpotKlineTimeZoneUTC,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
		candles = make([]kline.Candle, len(resp))
		for i := range resp {
			candles[i] = kline.Candle{
				Time:        resp[i].OpenTime.Time().UTC(),
				Open:        resp[i].Open,
				High:        resp[i].High,
				Low:         resp[i].Low,
				Close:       resp[i].Close,
				Volume:      resp[i].Volume,
				QuoteVolume: resp[i].QuoteVolume,
			}
		}
	case asset.USDTMarginedFutures:
		resp, err := e.GetSwapKlines(ctx, &SwapKlinesRequest{
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
				Time:   resp[i].Time.Time().UTC(),
				Open:   resp[i].Open.Float64(),
				High:   resp[i].High.Float64(),
				Low:    resp[i].Low.Float64(),
				Close:  resp[i].Close.Float64(),
				Volume: resp[i].Volume.Float64(),
			}
		}
	case asset.CoinMarginedFutures:
		resp, err := e.GetCoinMarginedKlineData(ctx, &CoinMarginedKlineDataRequest{
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
				Time:   resp[i].Time.Time().UTC(),
				Open:   resp[i].Open.Float64(),
				High:   resp[i].High.Float64(),
				Low:    resp[i].Low.Float64(),
				Close:  resp[i].Close.Float64(),
				Volume: resp[i].Volume.Float64(),
			}
		}
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	return candles, nil
}

// checkCandleHistory returns errCandlesBeforeHistory when a range's candles start after the range does. BingX sends a
// candle for every period of a crypto symbol, traded or not, so a late first candle means it holds none before it: its
// retention for the interval ends there, or the symbol was listed later. A USDT-M contract with a trading schedule, such
// as a forex contract, has no candles while its market is closed, so its late first candle is left for the request
// processing to pad when the contract launched by the range's start and BingX holds the interval that far back, which
// BTC-USDT's candle at the start shows, as BingX keeps each interval for the same span on every contract. An empty
// series is left for the request processing to report
func (e *Exchange) checkCandleHistory(ctx context.Context, a asset.Item, pair currency.Pair, interval kline.Interval, candles []kline.Candle, start time.Time) error {
	if len(candles) == 0 {
		return nil
	}
	first := slices.MinFunc(candles, func(x, y kline.Candle) int { return x.Time.Compare(y.Time) }).Time
	if !first.After(start) {
		return nil
	}
	if a == asset.USDTMarginedFutures {
		schedules, err := e.GetSwapTradingSchedule(ctx, pair)
		var apiErr *APIError
		switch {
		case err == nil && len(schedules) != 0:
			contracts, err := e.GetSwapContracts(ctx, pair)
			if err != nil {
				return fmt.Errorf("error checking %s launch: %w", pair, err)
			}
			i := slices.IndexFunc(contracts, func(c SwapContractResponse) bool {
				return slices.ContainsFunc(schedules, func(s SwapTradingScheduleResponse) bool { return s.Symbol == c.Symbol })
			})
			if i >= 0 && !contracts[i].LaunchTime.Time().IsZero() && !start.Before(contracts[i].LaunchTime.Time()) {
				held, err := e.candles(ctx, a, currency.NewBTCUSDT(), interval, start, start.Add(interval.Duration()), 1)
				if err != nil {
					return fmt.Errorf("error checking %s candle history: %w", interval, err)
				}
				if len(held) != 0 {
					return nil
				}
			}
		case err != nil && (!errors.As(err, &apiErr) || apiErr.Code != 109419):
			return fmt.Errorf("error checking %s trading schedule: %w", pair, err)
		}
	}
	return fmt.Errorf("%w: the first candle opens at %s, after the requested start %s", errCandlesBeforeHistory, first.UTC(), start.UTC())
}

// GetFeeByType returns an estimate of fee based on the type of transaction. Trade fees are spot fees at the account's
// commission rates for the pair, or at BingX's standard spot rate without credentials, and withdrawal fees those of the
// coin's default network
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if err := common.NilGuard(feeBuilder); err != nil {
		return 0, err
	}
	feeType := feeBuilder.FeeType
	if feeType == exchange.CryptocurrencyTradeFee && !e.AreCredentialsValid(ctx) {
		feeType = exchange.OfflineTradeFee
	}
	switch feeType {
	case exchange.CryptocurrencyTradeFee:
		rates, err := e.GetSpotCommissionRate(ctx, feeBuilder.Pair)
		if err != nil {
			return 0, err
		}
		rate := rates.TakerCommissionRate
		if feeBuilder.IsMaker {
			rate = rates.MakerCommissionRate
		}
		return rate * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	case exchange.CryptocurrencyWithdrawalFee:
		network, err := e.defaultNetwork(ctx, feeBuilder.Pair.Base)
		if err != nil {
			return 0, err
		}
		return network.WithdrawFee.Float64(), nil
	case exchange.OfflineTradeFee:
		return offlineSpotTradeFeeRate * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	default:
		return 0, nil
	}
}

// coinNetworks returns a coin's networks from its deposit and withdrawal configuration
func (e *Exchange) coinNetworks(ctx context.Context, coin currency.Code) ([]CurrencyNetwork, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	configs, err := e.GetCurrencyConfig(ctx, coin, "")
	if err != nil {
		return nil, err
	}
	for i := range configs {
		if configs[i].Coin.Equal(coin) && len(configs[i].NetworkList) != 0 {
			return configs[i].NetworkList, nil
		}
	}
	return nil, fmt.Errorf("%w: no networks for %s", currency.ErrCurrencyNotFound, coin)
}

// defaultNetwork returns a coin's default network
func (e *Exchange) defaultNetwork(ctx context.Context, coin currency.Code) (*CurrencyNetwork, error) {
	networks, err := e.coinNetworks(ctx, coin)
	if err != nil {
		return nil, err
	}
	for i := range networks {
		if networks[i].IsDefault {
			return &networks[i], nil
		}
	}
	return nil, fmt.Errorf("%w for %s", errDefaultNetworkAbsent, coin)
}

// GetAvailableTransferChains returns the networks of a coin's deposit and withdrawal configuration
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	networks, err := e.coinNetworks(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	chains := make([]string, len(networks))
	for i := range networks {
		chains[i] = networks[i].Network
	}
	return chains, nil
}

// GetDepositAddress returns the main account's deposit address of a coin on chain, or on the coin's only or default
// network when chain is empty
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	resp, err := e.GetMainAccountDepositAddress(ctx, &MainAccountDepositAddressRequest{Coin: cryptocurrency, Limit: depositAddressPageLimit})
	if err != nil {
		return nil, err
	}
	network := chain
	if network == "" && len(resp.Data) != 1 {
		defaultNetwork, err := e.defaultNetwork(ctx, cryptocurrency)
		if err != nil {
			return nil, err
		}
		network = defaultNetwork.Network
	}
	for i := range resp.Data {
		if network == "" || strings.EqualFold(resp.Data[i].Network, network) {
			return &deposit.Address{
				Address: resp.Data[i].Address,
				Tag:     resp.Data[i].Tag,
				Chain:   resp.Data[i].Network,
			}, nil
		}
	}
	return nil, fmt.Errorf("%w: %s on %q", deposit.ErrAddressNotFound, cryptocurrency, network)
}

// WithdrawCryptocurrencyFunds withdraws a cryptocurrency from the fund account, where deposits arrive, returning the
// withdrawal's ID
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.Withdraw(ctx, &WithdrawRequest{
		Coin:            withdrawRequest.Currency,
		Network:         withdrawRequest.Crypto.Chain,
		Address:         withdrawRequest.Crypto.Address,
		AddressTag:      withdrawRequest.Crypto.AddressTag,
		Amount:          withdrawRequest.Amount,
		WalletType:      WalletTypeFund,
		WithdrawOrderID: withdrawRequest.ClientOrderID,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: resp.ID}, nil
}

// WithdrawFiatFunds is not supported: BingX withdraws no fiat through its API
func (e *Exchange) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank is not supported: BingX withdraws no fiat through its API
func (e *Exchange) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits loads the order execution limits of an asset's symbols: spot tick, step and notional
// limits, USDT-M price and quantity precisions with the minimum quantity and value, and coin-M price precisions with
// the minimum contract count. Spot minimum and maximum quantities are left out, as BingX derives them from the
// notional limits and the price
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures:
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	format, err := e.GetPairFormat(a, false)
	if err != nil {
		return err
	}
	var l []limits.MinMaxLevel
	switch a {
	case asset.Spot:
		symbols, err := e.GetSpotSymbols(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(symbols))
		for i := range symbols {
			s := &symbols[i]
			pair, err := symbolPair(s.Symbol, format)
			if err != nil {
				log.Warnf(log.ExchangeSys, "%s skipping %s symbol limits: %s", e.Name, a, err)
				continue
			}
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
				PriceStepIncrementSize:  s.TickSize,
				AmountStepIncrementSize: s.StepSize,
				MinNotional:             s.MinNotional,
				// A maximum notional below zero means no limit
				MaximumQuoteAmount: max(s.MaxNotional, 0),
				Listed:             s.TimeOnline.Time(),
				Delisting:          s.OffTime.Time(),
			})
		}
	case asset.USDTMarginedFutures:
		contracts, err := e.GetSwapContracts(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(contracts))
		for i := range contracts {
			c := &contracts[i]
			pair, err := symbolPair(c.Symbol, format)
			if err != nil {
				log.Warnf(log.ExchangeSys, "%s skipping %s symbol limits: %s", e.Name, a, err)
				continue
			}
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
				PriceStepIncrementSize:  precisionStep(c.PricePrecision),
				AmountStepIncrementSize: precisionStep(c.QuantityPrecision),
				MinimumBaseAmount:       c.TradeMinQuantity,
				MinNotional:             c.TradeMinUSDT,
				Listed:                  c.LaunchTime.Time(),
				Delisting:               c.OffTime.Time(),
			})
		}
	case asset.CoinMarginedFutures:
		contracts, err := e.GetCoinMarginedContracts(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(contracts))
		for i := range contracts {
			c := &contracts[i]
			pair, err := symbolPair(c.Symbol, format)
			if err != nil {
				log.Warnf(log.ExchangeSys, "%s skipping %s symbol limits: %s", e.Name, a, err)
				continue
			}
			l = append(l, limits.MinMaxLevel{
				Key:                    key.NewExchangeAssetPair(e.Name, a, pair),
				PriceStepIncrementSize: precisionStep(c.PricePrecision),
				MinimumBaseAmount:      c.MinQuantity.Float64(),
				Listed:                 c.TimeOnline.Time(),
			})
		}
	}
	return limits.Load(l)
}

// precisionStep returns the step of a number of decimal places, such as 0.01 for 2
func precisionStep(places uint64) float64 {
	return math.Pow10(-int(min(places, 300)))
}

// UpdateCurrencyStates updates whether each coin can be deposited and withdrawn, which it can while any of its
// networks allows it. BingX's coin configuration is the account's, so the states are kept for spot, the asset its
// funds trade on
func (e *Exchange) UpdateCurrencyStates(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot:
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		return fmt.Errorf("%w: BingX keeps no %s currency states", common.ErrFunctionNotSupported, a)
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	configs, err := e.GetCurrencyConfig(ctx, currency.EMPTYCODE, "")
	if err != nil {
		return err
	}
	states := make(map[currency.Code]currencystate.Options, len(configs))
	for i := range configs {
		var depositEnabled, withdrawEnabled bool
		for j := range configs[i].NetworkList {
			depositEnabled = depositEnabled || configs[i].NetworkList[j].DepositEnable
			withdrawEnabled = withdrawEnabled || configs[i].NetworkList[j].WithdrawEnable
		}
		states[configs[i].Coin] = currencystate.Options{
			Deposit:  new(depositEnabled),
			Withdraw: new(withdrawEnabled),
		}
	}
	return e.States.UpdateAll(a, states)
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	if cp.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	switch a {
	case asset.Spot:
		return tradeBaseURL + "spot/" + cp.Base.Upper().String() + cp.Quote.Upper().String(), nil
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		symbol, err := e.FormatSymbol(cp, a)
		if err != nil {
			return "", err
		}
		return tradeBaseURL + "perpetual/" + symbol, nil
	default:
		return "", fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
}

var _ exchange.IBotExchange = (*Exchange)(nil)
