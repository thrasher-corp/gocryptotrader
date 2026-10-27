package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
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

const (
	// publicWebsocketURL serves the public channels; authenticated connections must use authenticatedWebsocketURL
	publicWebsocketURL        = "wss://api-pub.bitfinex.com/ws/2"
	authenticatedWebsocketURL = "wss://api.bitfinex.com/ws/2"

	// wrapperBookLength is how many orders, or funding offers, a side of the book the wrapper fetches holds
	wrapperBookLength = 100
	// tradesPageLimit is the most trades a page holds
	tradesPageLimit = 10000
	// ordersHistoryPageLimit and movementsPageLimit are the most orders and movements a page of their history holds
	ordersHistoryPageLimit = 2500
	movementsPageLimit     = 2500
	// candlesPageLimit is the most candles a page holds
	candlesPageLimit = 10000
	// fundingCandlePeriod is the funding period, in days, of the funding candles the wrapper fetches, as the default
	// websocket subscription's are
	fundingCandlePeriod = 30
	// testListingMarker marks the pairs and currencies Bitfinex lists for testing, which trade nothing real
	testListingMarker = "TEST"

	// minimumDerivativeLeverage and maximumDerivativeLeverage bound a derivative order's leverage
	minimumDerivativeLeverage = 1
	maximumDerivativeLeverage = 100
	// derivativeFundingInterval is how often Bitfinex sets a derivative's funding rate, at each funding event, from the
	// premium accrued over the interval ending then
	derivativeFundingInterval = 8 * time.Hour
	// statusHistoryPageLimit is the most snapshots a page of a derivative's status history holds
	statusHistoryPageLimit = 5000
	// orderHistoryReach is how far back Bitfinex's order history reaches
	orderHistoryReach = 14 * 24 * time.Hour
	// orderTradesReach is how far back an order's trades reach, short of Bitfinex's 10 days by an hour, as its clock and
	// the request's travel are not GCT's
	orderTradesReach = 10*24*time.Hour - time.Hour
	// amountDecimalPlaces is how many decimal places Bitfinex keeps an amount to
	amountDecimalPlaces = 8
)

var (
	errTransferMethodUnknown    = errors.New("no transfer method for currency")
	errTransferMethodNeeded     = errors.New("transfer chain required, as the currency has several")
	errInvalidClientOrderID     = errors.New("client order ID must be an unsigned integer")
	errUnsupportedOrderType     = errors.New("order type not supported")
	errHistoryPageFull          = errors.New("history has more records from one millisecond than a page holds")
	errInvalidLeverage          = errors.New("invalid leverage")
	errOrderLeverageUnknown     = errors.New("order leverage unknown")
	errPositionCollateralNotSet = errors.New("position collateral not set")
	errTriggerPriceRequired     = errors.New("trigger price required")
	errTriggerPriceUnsupported  = errors.New("trigger price not supported")
	errPriceUnsupported         = errors.New("price not supported")
	errOrderTypeUnchangeable    = errors.New("order type cannot be changed")
	errTrackingModeUnsupported  = errors.New("tracking mode not supported")
)

// SetDefaults sets the basic defaults for bitfinex
func (e *Exchange) SetDefaults() {
	e.Name = "Bitfinex"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.MarginFunding, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		}
		if a == asset.Margin {
			ps.ConfigFormat.Delimiter = ":"
		}
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				FiatWithdraw:        true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				ModifyOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				TradeFetching:       true,
				UserTradeHistory:    true,
				TradeFee:            true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				KlineFetching:       true,
				FundingRateFetching: true,
				// A currency deposited and withdrawn over several methods needs one named
				MultiChainDeposits:                true,
				MultiChainWithdrawals:             true,
				MultiChainDepositRequiresChainSet: true,
			},
			WebsocketCapabilities: protocol.Features{
				AccountBalance:         true,
				CancelOrders:           true,
				CancelOrder:            true,
				SubmitOrder:            true,
				ModifyOrder:            true,
				TickerFetching:         true,
				KlineFetching:          true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				FundingRateFetching:    true,
				AccountInfo:            true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission | exchange.AutoWithdrawFiatWithAPIPermission,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates:                    true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{kline.EightHour: true},
				FundingRateBatching:             map[asset.Item]bool{asset.USDTMarginedFutures: true, asset.CoinMarginedFutures: true},
				Positions:                       true,
				Leverage:                        true,
				OpenInterest:                    exchange.OpenInterestSupport{Supported: true, SupportsRestBatch: true},
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
					kline.IntervalCapacity{Interval: kline.ThreeHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					// Made from weekly candles, as readWeeklyForTwoWeeks says
					kline.IntervalCapacity{Interval: kline.TwoWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: candlesPageLimit,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:         apiURL,
		exchange.WebsocketSpot:    publicWebsocketURL,
		exchange.WebsocketPrivate: authenticatedWebsocketURL,
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
	return e.setupWebsocket(exch)
}

// FetchTradablePairs returns the pairs, or for margin funding the currencies, an asset trades, leaving out the ones
// Bitfinex lists for testing; the futures assets trade the derivative pairs their collateral margins
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	var symbols []string
	switch a {
	case asset.Spot:
		spot, err := e.GetSpotPairs(ctx)
		if err != nil {
			return nil, err
		}
		// Bitfinex lists securities among its exchange pairs, which only its securities accounts trade
		securities, err := e.GetSecuritiesPairs(ctx)
		if err != nil {
			return nil, err
		}
		symbols = slices.DeleteFunc(spot, func(s string) bool { return slices.Contains(securities, s) })
	case asset.Margin:
		marginPairs, err := e.GetMarginPairs(ctx)
		if err != nil {
			return nil, err
		}
		symbols = marginPairs
	case asset.MarginFunding:
		// The currencies funding trades in are those with funding tickers
		tickers, err := e.GetTickers(ctx, nil)
		if err != nil {
			return nil, err
		}
		for i := range tickers.Funding {
			symbols = append(symbols, tickers.Funding[i].Symbol)
		}
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		derivatives, err := e.GetFuturesPairs(ctx)
		if err != nil {
			return nil, err
		}
		symbols = derivatives
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	pairs := make(currency.Pairs, 0, len(symbols))
	for _, symbol := range symbols {
		if strings.Contains(symbol, testListingMarker) {
			continue
		}
		p, err := pairFromSymbol(symbol)
		if err != nil {
			return nil, err
		}
		if isDerivativeAsset(a) && derivativeAsset(p.Quote) != a {
			continue
		}
		pairs = append(pairs, p)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(ctx, a)
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, a, false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateOrderExecutionLimits stores the smallest and largest order amounts of an asset's pairs; spot and margin trade
// the same markets, so share them, and the futures assets trade the derivative pairs their collateral margins
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var info []PairInfoEntryResponse
	var err error
	switch a {
	case asset.Spot, asset.Margin:
		info, err = e.GetPairInfo(ctx)
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		info, err = e.GetFuturesPairInfo(ctx)
	case asset.MarginFunding:
		// Bootstrap skips this error for every enabled asset, where asset.ErrNotSupported fails the load
		return common.ErrNotYetImplemented
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if err != nil {
		return err
	}
	l := make([]limits.MinMaxLevel, 0, len(info))
	for i := range info {
		if strings.Contains(info[i].Pair, testListingMarker) {
			continue
		}
		p, err := pairFromSymbol(info[i].Pair)
		if err != nil {
			return err
		}
		if isDerivativeAsset(a) && derivativeAsset(p.Quote) != a {
			continue
		}
		l = append(l, limits.MinMaxLevel{
			Key:               key.NewExchangeAssetPair(e.Name, a, p),
			MinimumBaseAmount: info[i].Info.MinimumOrderSize.Float64(),
			MaximumBaseAmount: info[i].Info.MaximumOrderSize.Float64(),
		})
	}
	return limits.Load(l)
}

// UpdateTickers updates the tickers of an asset's enabled pairs from one request for every ticker
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot, asset.Margin, asset.MarginFunding, asset.USDTMarginedFutures, asset.CoinMarginedFutures:
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	tickers, err := e.GetTickers(ctx, nil)
	if err != nil {
		return err
	}
	var errs error
	process := func(symbol string, price func(currency.Pair) *ticker.Price) {
		p, err := pairFromSymbol(symbol)
		if err != nil {
			// A symbol GCT cannot pair is not one of the enabled pairs
			return
		}
		if enabled, err := e.IsPairEnabled(p, a); err != nil || !enabled {
			return
		}
		if err := ticker.ProcessTicker(price(p)); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	if a == asset.MarginFunding {
		for i := range tickers.Funding {
			process(tickers.Funding[i].Symbol, func(p currency.Pair) *ticker.Price {
				return e.fundingTickerPrice(&tickers.Funding[i].Ticker, p, a)
			})
		}
		return errs
	}
	for i := range tickers.Trading {
		process(tickers.Trading[i].Symbol, func(p currency.Pair) *ticker.Price {
			return e.tickerPrice(&tickers.Trading[i].Ticker, p, a)
		})
	}
	return errs
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbol, err := symbolFor(a, p)
	if err != nil {
		return nil, err
	}
	var price *ticker.Price
	if a == asset.MarginFunding {
		t, err := e.GetFundingTicker(ctx, symbol)
		if err != nil {
			return nil, err
		}
		price = e.fundingTickerPrice(t, p, a)
	} else {
		t, err := e.GetTicker(ctx, symbol)
		if err != nil {
			return nil, err
		}
		price = e.tickerPrice(t, p, a)
	}
	if err := ticker.ProcessTicker(price); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// tickerPrice returns a trading pair's ticker as GCT records it. Bitfinex's REST tickers carry no time; their first
// trade time is not one
func (e *Exchange) tickerPrice(t *TickerResponse, p currency.Pair, a asset.Item) *ticker.Price {
	return &ticker.Price{
		ExchangeName:        e.Name,
		Pair:                p,
		AssetType:           a,
		Bid:                 t.Bid,
		BidSize:             t.BidSize,
		Ask:                 t.Ask,
		AskSize:             t.AskSize,
		Last:                t.LastPrice,
		BaseVolume:          t.Volume,
		High:                t.High,
		Low:                 t.Low,
		Open24Hour:          t.LastPrice - t.DailyChange,
		PercentChange24Hour: t.DailyChangeRelative * 100,
	}
}

// fundingTickerPrice returns a funding currency's ticker as GCT records it, its rates as prices
func (e *Exchange) fundingTickerPrice(t *FundingTickerResponse, p currency.Pair, a asset.Item) *ticker.Price {
	return &ticker.Price{
		ExchangeName:          e.Name,
		Pair:                  p,
		AssetType:             a,
		FlashReturnRate:       t.FlashReturnRate,
		Bid:                   t.Bid,
		BidPeriod:             float64(t.BidPeriod),
		BidSize:               t.BidSize,
		Ask:                   t.Ask,
		AskPeriod:             float64(t.AskPeriod),
		AskSize:               t.AskSize,
		Last:                  t.LastRate,
		BaseVolume:            t.Volume,
		High:                  t.High,
		Low:                   t.Low,
		FlashReturnRateAmount: t.FlashReturnRateAmountAvailable,
	}
}

// UpdateOrderbook updates and returns a pair's book of orders, or a funding currency's of offers, as the default
// websocket subscription keeps it
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}
	symbol, err := symbolFor(a, p)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		PriceDuplication:  true,
		IsFundingRate:     a == asset.MarginFunding,
		ValidateOrderbook: e.ValidateOrderbook,
		LastUpdated:       time.Now(),
	}
	req := &RawBookRequest{Symbol: symbol, Length: wrapperBookLength}
	if a == asset.MarginFunding {
		levels, err := e.GetRawFundingBook(ctx, req)
		if err != nil {
			return nil, err
		}
		for i := range levels {
			addBookLevel(book, &levels[i])
		}
	} else {
		levels, err := e.GetRawBook(ctx, req)
		if err != nil {
			return nil, err
		}
		for i := range levels {
			addBookLevel(book, &levels[i])
		}
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, a)
}

// addBookLevel adds an order, offer or level of Bitfinex's to the side of a book holding it
func addBookLevel(book *orderbook.Book, level wsBookLevel) {
	l, bid, _ := level.level()
	if bid {
		book.Bids = append(book.Bids, l)
		return
	}
	book.Asks = append(book.Asks, l)
}

// UpdateAccountBalances stores and returns the balances of the wallet an asset trades from: exchange for spot, margin
// for margin and funding for margin funding. The futures assets trade from the derivatives wallet, the margin wallet's
// derivative currencies, each from the collateral it margins
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	switch a {
	case asset.Spot, asset.Margin, asset.MarginFunding, asset.USDTMarginedFutures, asset.CoinMarginedFutures:
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	wallets, err := e.GetWallets(ctx)
	if err != nil {
		return nil, err
	}
	subAcct := accounts.NewSubAccount(a, "")
	for i := range wallets {
		if wa, err := walletAsset(wallets[i].Type, wallets[i].Currency); err != nil || wa != a {
			continue
		}
		subAcct.Balances.Set(wallets[i].Currency, walletBalance(&wallets[i]))
	}
	subAccts := accounts.SubAccounts{subAcct}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns the account's deposits and withdrawals in every currency
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	movements, err := e.getMovementsHistory(ctx, currency.EMPTYCODE)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(movements))
	for i := range movements {
		m := &movements[i]
		resp[i] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          m.Status,
			TransferID:      strconv.FormatUint(m.ID, 10),
			Description:     m.Note,
			Timestamp:       m.StartedAt.Time(),
			Currency:        m.Currency.String(),
			Amount:          math.Abs(m.Amount.Float64()),
			Fee:             math.Abs(m.Fees.Float64()),
			TransferType:    movementType(m),
			CryptoToAddress: m.DestinationAddress,
			CryptoTxID:      m.TransactionID,
			CryptoChain:     movementChain(m),
		}
	}
	return resp, nil
}

// getMovementsHistory returns the account's deposits and withdrawals of a currency, or of every currency when it is
// empty, page by page
func (e *Exchange) getMovementsHistory(ctx context.Context, c currency.Code) ([]MovementResponse, error) {
	return pageBack(time.Time{}, time.Time{}, movementsPageLimit, func(end time.Time) ([]MovementResponse, error) {
		return e.GetMovements(ctx, &MovementsRequest{Currency: c, End: end, Limit: movementsPageLimit})
	}, func(m *MovementResponse) uint64 { return m.ID }, func(m *MovementResponse) time.Time { return m.UpdatedAt.Time() })
}

// movementChain returns the method a movement went over, its chain, in the form GetAvailableTransferChains returns
func movementChain(m *MovementResponse) string {
	return strings.ToLower(m.Method)
}

// movementType returns whether a movement is a deposit, whose amount is positive, or a withdrawal
func movementType(m *MovementResponse) string {
	if m.Amount.Float64() < 0 {
		return "withdrawal"
	}
	return "deposit"
}

// GetWithdrawalsHistory returns the account's withdrawals of a currency, or of every currency when it is empty
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	movements, err := e.getMovementsHistory(ctx, bitfinexCurrency(c))
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(movements))
	for i := range movements {
		m := &movements[i]
		if m.Amount.Float64() >= 0 {
			continue
		}
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          m.Status,
			TransferID:      strconv.FormatUint(m.ID, 10),
			Description:     m.Note,
			Timestamp:       m.StartedAt.Time(),
			Currency:        m.Currency.String(),
			Amount:          -m.Amount.Float64(),
			Fee:             math.Abs(m.Fees.Float64()),
			TransferType:    movementType(m),
			CryptoToAddress: m.DestinationAddress,
			CryptoTxID:      m.TransactionID,
			CryptoChain:     movementChain(m),
		})
	}
	return resp, nil
}

// GetRecentTrades returns a pair's, or a funding currency's, most recent trades
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	resp, err := e.getTrades(ctx, p, a, &PublicTradesRequest{Limit: tradesPageLimit})
	if err != nil {
		return nil, err
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return resp, nil
}

// GetHistoricTrades returns the trades of a pair, or of a funding currency, between two times, paging back from the
// end
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	resp, err := pageBack(start, historyEnd(end), tradesPageLimit, func(pageEnd time.Time) ([]trade.Data, error) {
		return e.getTrades(ctx, p, a, &PublicTradesRequest{Start: start, End: pageEnd, Limit: tradesPageLimit, Sort: -1})
	}, func(t *trade.Data) string { return t.TID }, func(t *trade.Data) time.Time { return t.Timestamp })
	if err != nil {
		return nil, err
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return trade.FilterTradesByTime(resp, start, end), nil
}

// pageBack returns a history Bitfinex sends newest first, a page of at most limit records at a time, paging back from
// end to start. Each page ends at its predecessor's oldest record, which both hold and which is kept once. A full page
// of records all from the millisecond it ends at errors, as no page can reach any more from that millisecond, and so
// does a server ignoring the end
func pageBack[T any, K comparable](start, end time.Time, limit int, fetch func(end time.Time) ([]T, error), id func(*T) K, at func(*T) time.Time) ([]T, error) {
	var resp []T
	seen := make(map[K]struct{})
	for {
		page, err := fetch(end)
		if err != nil {
			return nil, err
		}
		oldest := end
		for i := range page {
			if t := at(&page[i]); oldest.IsZero() || t.Before(oldest) {
				oldest = t
			}
			k := id(&page[i])
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			resp = append(resp, page[i])
		}
		if len(page) < limit {
			return resp, nil
		}
		if !end.IsZero() && !oldest.Before(end) {
			return nil, fmt.Errorf("%w: %d or more from %s", errHistoryPageFull, limit, end.UTC().Format(time.RFC3339Nano))
		}
		if !start.IsZero() && oldest.Before(start) {
			return resp, nil
		}
		end = oldest
	}
}

// historyEnd returns the end of a history window as Bitfinex takes it, no later than now: Bitfinex refuses a window
// ending after the current UTC day, as one ending at the next candle boundary can
func historyEnd(end time.Time) time.Time {
	if now := time.Now(); end.After(now) {
		return now
	}
	return end
}

// getTrades returns a page of a pair's, or a funding currency's, trades
func (e *Exchange) getTrades(ctx context.Context, p currency.Pair, a asset.Item, req *PublicTradesRequest) ([]trade.Data, error) {
	symbol, err := symbolFor(a, p)
	if err != nil {
		return nil, err
	}
	req.Symbol = symbol
	if a == asset.MarginFunding {
		trades, err := e.GetPublicFundingTrades(ctx, req)
		if err != nil {
			return nil, err
		}
		resp := make([]trade.Data, len(trades))
		for i := range trades {
			resp[i] = e.fundingTradeData(p, a, &trades[i])
		}
		return resp, nil
	}
	trades, err := e.GetPublicTrades(ctx, req)
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(trades))
	for i := range trades {
		resp[i] = e.tradeData(p, a, &trades[i])
	}
	return resp, nil
}

// SubmitOrder places an order through the authenticated websocket, or over REST without it
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	req, err := placeOrderRequest(s)
	if err != nil {
		return nil, err
	}
	var placed *OrderResponse
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		if placed, err = e.WebsocketPlaceOrder(ctx, req); err != nil {
			return nil, err
		}
	} else {
		n, err := e.PlaceOrder(ctx, req)
		if err != nil {
			return nil, err
		}
		if len(n.Data) == 0 {
			return nil, common.ErrNoResponse
		}
		placed = &n.Data[0]
	}
	resp, err := s.DeriveSubmitResponse(strconv.FormatUint(placed.ID, 10))
	if err != nil {
		return nil, err
	}
	if status := orderStatusFromBitfinex(placed.Status); status != order.UnknownStatus {
		resp.Status = status
	}
	// Neither stop has a price of its own, which the order manager would copy into their later modifications; a
	// trailing stop's trigger is the stop price Bitfinex works out from its distance
	switch s.Type {
	case order.Stop:
		resp.Price = 0
	case order.TrailingStop:
		resp.Price, resp.TriggerPrice = 0, placed.Price
	}
	return resp, nil
}

// placeOrderRequest returns the request placing an order: an EXCHANGE order for spot, from the exchange wallet, or a
// margin or derivative order from the margin wallet, its amount signed by its side. A derivative order takes its
// leverage, or Bitfinex's default of 10 without one
func placeOrderRequest(s *order.Submit) (*PlaceOrderRequest, error) {
	switch s.AssetType {
	case asset.Spot, asset.Margin, asset.USDTMarginedFutures, asset.CoinMarginedFutures:
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, s.AssetType)
	}
	symbol, err := orderSymbol(s.AssetType, s.Pair)
	if err != nil {
		return nil, err
	}
	req := &PlaceOrderRequest{Symbol: symbol, Amount: s.Amount, Price: s.Price}
	if isDerivativeAsset(s.AssetType) && s.Leverage != 0 {
		if req.Leverage, err = derivativeLeverage(s.Leverage); err != nil {
			return nil, err
		}
	}
	if s.Side.IsShort() {
		req.Amount = -s.Amount
	}
	// Bitfinex's fill or kill and immediate or cancel orders take a limit price, though a market order never rests
	// anyway; a good till crossing order is a post only limit order; and its orders expire only at a time, so a good
	// till day order needs one as good till time. It has no stop or reduce orders
	switch {
	case s.TimeInForce.Is(order.StopOrReduce),
		s.TimeInForce.Is(order.FillOrKill) && s.Type != order.Limit,
		s.TimeInForce.Is(order.ImmediateOrCancel) && s.Type != order.Limit && s.Type != order.Market,
		s.TimeInForce.Is(order.GoodTillCrossing) && s.Type != order.Limit,
		s.TimeInForce.Is(order.GoodTillDay) && !s.TimeInForce.Is(order.GoodTillTime):
		return nil, fmt.Errorf("%w: %s for a %s order", order.ErrUnsupportedTimeInForce, s.TimeInForce, s.Type)
	case s.TimeInForce.Is(order.GoodTillTime) && s.EndTime.IsZero():
		return nil, fmt.Errorf("%w: %s orders need an end time", order.ErrUnsupportedTimeInForce, order.GoodTillTime)
	}
	switch s.Type {
	case order.Limit:
		req.Type = OrderTypeLimit
		switch {
		case s.TimeInForce.Is(order.FillOrKill):
			req.Type = OrderTypeFillOrKill
		case s.TimeInForce.Is(order.ImmediateOrCancel):
			req.Type = OrderTypeImmediateOrCancel
		}
	case order.Market:
		req.Type, req.Price = OrderTypeMarket, 0
	case order.Stop, order.StopLimit:
		// A stop triggers at its trigger price, which Bitfinex takes as its price; a stop limit then places a limit
		// order at its price
		if s.TriggerPrice <= 0 {
			return nil, fmt.Errorf("%w: %s orders trigger at it", errTriggerPriceRequired, s.Type)
		}
		req.Type, req.Price = OrderTypeStop, s.TriggerPrice
		if s.Type == order.StopLimit {
			if s.Price <= 0 {
				return nil, fmt.Errorf("%w: a stop limit order places a limit order at it once triggered", order.ErrPriceMustBeSetIfLimitOrder)
			}
			req.Type, req.AuxiliaryLimitPrice = OrderTypeStopLimit, s.Price
		}
	case order.TrailingStop:
		if s.TrackingMode == order.Percentage {
			return nil, fmt.Errorf("%w: %s, as Bitfinex trails by a price distance", errTrackingModeUnsupported, s.TrackingMode)
		}
		req.Type, req.Price, req.TrailingPrice = OrderTypeTrailingStop, 0, s.TrackingValue
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedOrderType, s.Type)
	}
	if s.AssetType == asset.Spot {
		req.Type = ExchangeOrderTypePrefix + req.Type
	}
	if s.TimeInForce.Is(order.PostOnly) || s.TimeInForce.Is(order.GoodTillCrossing) {
		req.Flags |= OrderFlagPostOnly
	}
	if s.ReduceOnly {
		req.Flags |= OrderFlagReduceOnly
	}
	if s.TimeInForce.Is(order.GoodTillTime) {
		req.TimeInForce = s.EndTime
	}
	if s.ClientOrderID != "" {
		if req.ClientOrderID, err = strconv.ParseUint(s.ClientOrderID, 10, 64); err != nil {
			return nil, fmt.Errorf("%w: %q", errInvalidClientOrderID, s.ClientOrderID)
		}
	}
	return req, nil
}

// derivativeLeverage returns a derivative order's leverage as Bitfinex takes it, a whole number from 1 to 100
func derivativeLeverage(leverage float64) (uint64, error) {
	if leverage < minimumDerivativeLeverage || leverage > maximumDerivativeLeverage || leverage != math.Trunc(leverage) {
		return 0, fmt.Errorf("%w: %v, as Bitfinex takes a whole number from %d to %d", errInvalidLeverage, leverage, minimumDerivativeLeverage, maximumDerivativeLeverage)
	}
	return uint64(leverage), nil
}

// ModifyOrder updates an order's amount and prices through the authenticated websocket, or over REST without it. A
// modification carries none of what Bitfinex reads an update by: the order's type, which its prices follow, what the
// order has executed, as Bitfinex takes the amount an update gives as what remains of the order, and a derivative
// order's leverage, which Bitfinex needs to update it, so the order is looked up for each. A stop moves by its trigger
// price, and a stop limit by its trigger price and its limit price, each when given; a stop has no price of its own, so
// its price is neither sent nor reported, and a price given without a trigger price is refused. A trailing stop's stop
// trails the market, so a price for it is refused. Bitfinex does not change an order's type, so another is refused. An
// order's time in force is left as it is and reported as the order has it, as a modification carries no expiry time and
// the order manager sets each one's time in force to what it holds for the order; UpdateOrder and WebsocketUpdateOrder
// change an order's expiry and flags, such as post only
func (e *Exchange) ModifyOrder(ctx context.Context, m *order.Modify) (*order.ModifyResponse, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	id, err := parseOrderID(m.OrderID)
	if err != nil {
		return nil, err
	}
	// Bitfinex signs an order's amount by its side, buying positive and selling negative
	if m.Amount != 0 && !m.Side.IsLong() && !m.Side.IsShort() {
		return nil, fmt.Errorf("%w: %s, and an order's amount is signed by its side", order.ErrSideIsInvalid, m.Side)
	}
	o, err := e.openOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	d, err := e.orderDetail(o)
	if err != nil {
		return nil, err
	}
	if m.Type != order.UnknownType && m.Type != order.AnyType && m.Type != d.Type {
		return nil, fmt.Errorf("%w: order %d is a %s order, not %s", errOrderTypeUnchangeable, id, d.Type, m.Type)
	}
	req := &UpdateOrderRequest{ID: id}
	var remaining float64
	// What remains as looked up is left unsent, as Bitfinex would set what remains to it, undoing a fill made since; the
	// order manager sends the order's own amount with a price change
	switch {
	case m.Amount == 0:
	case sameAmount(m.Amount, d.Amount):
		remaining = d.RemainingAmount
	default:
		executed := orderExecutedAmount(o)
		if isOrderStatusCut(o.Status) {
			trades, err := e.GetOrderTrades(ctx, &OrderTradesRequest{Symbol: o.Symbol, OrderID: id})
			if err != nil {
				return nil, err
			}
			executed = cutOrderExecutedAmount(o, trades)
		}
		if remaining, err = remainingOrderAmount(m.Amount, id, executed); err != nil {
			return nil, err
		}
		if !sameAmount(remaining, d.RemainingAmount) {
			req.Amount = remaining
			if m.Side.IsShort() {
				req.Amount = -remaining
			}
		}
	}
	if err := setUpdatePrices(req, d.Type, m); err != nil {
		return nil, err
	}
	if isDerivativePair(d.Pair) {
		if req.Leverage, err = orderLeverage(o); err != nil {
			return nil, err
		}
	}
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		if _, err := e.WebsocketUpdateOrder(ctx, req); err != nil {
			return nil, err
		}
	} else if _, err := e.UpdateOrder(ctx, req); err != nil {
		return nil, err
	}
	resp, err := m.DeriveModifyResponse()
	if err == nil {
		resp.Type, resp.TimeInForce, resp.RemainingAmount = d.Type, d.TimeInForce, remaining
		if d.Type == order.Stop {
			resp.Price = 0
		}
	}
	return resp, err
}

// remainingOrderAmount returns what remains of an order once modified to amount, which is what Bitfinex takes an
// update's amount as, less what the order has executed. It is worked out in decimal, so the amount sent carries no
// binary rounding error
func remainingOrderAmount(amount float64, id uint64, executed decimal.Decimal) (float64, error) {
	total, err := decimal.NewFromString(strconv.FormatFloat(amount, 'f', -1, 64))
	if err != nil {
		return 0, fmt.Errorf("%w: %v: %w", errInvalidAmount, amount, err)
	}
	remaining := total.Sub(executed)
	if !remaining.IsPositive() {
		return 0, fmt.Errorf("%w: %v, as order %d has executed %s of it", errInvalidAmount, amount, id, executed)
	}
	return remaining.InexactFloat64(), nil
}

// sameAmount reports whether two amounts are the same to the 8 decimal places Bitfinex keeps amounts to, as an amount
// carried as a float, such as the order store's, can differ past them from what Bitfinex reports
func sameAmount(a, b float64) bool {
	x, errX := decimal.NewFromString(strconv.FormatFloat(a, 'f', -1, 64))
	y, errY := decimal.NewFromString(strconv.FormatFloat(b, 'f', -1, 64))
	return errX == nil && errY == nil && x.Round(amountDecimalPlaces).Equal(y.Round(amountDecimalPlaces))
}

// setUpdatePrices sets the prices an update sends by the type of the order it updates, as placeOrderRequest places
// them: Bitfinex takes a stop's trigger price as its price, and a stop limit's limit price as its auxiliary limit price.
// A stop moves by its trigger price alone, as it is placed by, so a price without one is refused rather than dropped
func setUpdatePrices(req *UpdateOrderRequest, orderType order.Type, m *order.Modify) error {
	switch orderType {
	case order.Stop:
		if m.Price != 0 && m.TriggerPrice == 0 {
			return fmt.Errorf("%w: a stop order moves by its trigger price, not by price %v", errTriggerPriceRequired, m.Price)
		}
		req.Price = m.TriggerPrice
	case order.StopLimit:
		req.Price, req.AuxiliaryLimitPrice = m.TriggerPrice, m.Price
	case order.TrailingStop:
		if m.Price != 0 || m.TriggerPrice != 0 {
			return fmt.Errorf("%w: a trailing stop's stop trails the market by its distance", errPriceUnsupported)
		}
	default:
		// The modification's response would record a trigger price Bitfinex was never sent
		if m.TriggerPrice != 0 {
			return fmt.Errorf("%w: %s orders have none", errTriggerPriceUnsupported, orderType)
		}
		req.Price = m.Price
	}
	return nil
}

// openOrder returns an open order, by its ID
func (e *Exchange) openOrder(ctx context.Context, id uint64) (*OrderResponse, error) {
	orders, err := e.GetOpenOrders(ctx, &OpenOrdersRequest{IDs: []uint64{id}})
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(orders, func(o OrderResponse) bool { return o.ID == id })
	if i == -1 {
		return nil, fmt.Errorf("%w: %d is not open", order.ErrOrderNotFound, id)
	}
	return &orders[i], nil
}

// orderLeverage returns a derivative order's leverage, which Bitfinex reports in its meta data
func orderLeverage(o *OrderResponse) (uint64, error) {
	leverage, err := jsonparser.GetFloat(o.Meta, orderMetaLeverageKey)
	if err != nil {
		return 0, fmt.Errorf("%w: order %d: %w", errOrderLeverageUnknown, o.ID, err)
	}
	return derivativeLeverage(leverage)
}

// CancelOrder submits an order for cancellation through the authenticated websocket, or over REST without it
func (e *Exchange) CancelOrder(ctx context.Context, c *order.Cancel) error {
	if err := c.Validate(c.StandardCancel()); err != nil {
		return err
	}
	id, err := parseOrderID(c.OrderID)
	if err != nil {
		return err
	}
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		_, err = e.WebsocketCancelExistingOrder(ctx, id)
		return err
	}
	_, err = e.CancelExistingOrder(ctx, &CancelExistingOrderRequest{ID: id})
	return err
}

// parseOrderID returns an order ID as Bitfinex numbers it
func parseOrderID(orderID string) (uint64, error) {
	id, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: %q", errOrderIDRequired, orderID)
	}
	return id, nil
}

// CancelBatchOrders submits orders for cancellation in one request, reporting the orders Bitfinex accepted pending
// cancellation, and any its reply leaves out unknown
func (e *Exchange) CancelBatchOrders(ctx context.Context, cancels []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(cancels) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	ids := make([]uint64, len(cancels))
	for i := range cancels {
		id, err := parseOrderID(cancels[i].OrderID)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	statuses, err := e.submitOrderCancellations(ctx, &CancelOrdersRequest{IDs: ids}, ids)
	if err != nil {
		return nil, err
	}
	return &order.CancelBatchResponse{Status: statuses}, nil
}

// CancelAllOrders submits the account's open orders of an asset, on a pair when one is given, for cancellation. A
// request naming neither, such as the nil one the cancel all orders RPC sends, cancels every open order through
// Bitfinex's own cancellation of them all, which also reaches an order placed after they are listed
func (e *Exchange) CancelAllOrders(ctx context.Context, c *order.Cancel) (order.CancelAllResponse, error) {
	if c == nil {
		c = new(order.Cancel)
	}
	everyAsset := c.AssetType == asset.Empty || c.AssetType == asset.All
	if !everyAsset && !c.AssetType.IsValid() {
		return order.CancelAllResponse{}, fmt.Errorf("%w: %s", asset.ErrNotSupported, c.AssetType)
	}
	open, err := e.openOrderDetails(ctx, c.Pair, c.AssetType)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	ids := make([]uint64, len(open))
	for i := range open {
		if ids[i], err = parseOrderID(open[i].OrderID); err != nil {
			return order.CancelAllResponse{}, err
		}
	}
	req := &CancelOrdersRequest{IDs: ids}
	if c.Pair.IsEmpty() && everyAsset {
		req = &CancelOrdersRequest{All: true}
	} else if len(ids) == 0 {
		return order.CancelAllResponse{Status: map[string]string{}}, nil
	}
	statuses, err := e.submitOrderCancellations(ctx, req, ids)
	return order.CancelAllResponse{Status: statuses}, err
}

// submitOrderCancellations submits orders for cancellation, through the authenticated websocket or over REST,
// reporting the orders listed for it, and any other Bitfinex's reply names, pending cancellation, as Bitfinex confirms
// each cancellation with the order's close event. The websocket's notification names none of the orders; over REST, a
// listed order its reply leaves out is reported unknown
func (e *Exchange) submitOrderCancellations(ctx context.Context, req *CancelOrdersRequest, ids []uint64) (map[string]string, error) {
	statuses := make(map[string]string, len(ids))
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		if err := e.WebsocketCancelOrders(ctx, req); err != nil {
			return nil, err
		}
		for _, id := range ids {
			statuses[strconv.FormatUint(id, 10)] = order.PendingCancel.String()
		}
		return statuses, nil
	}
	n, err := e.CancelOrders(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		statuses[strconv.FormatUint(id, 10)] = order.UnknownStatus.String()
	}
	for i := range n.Data {
		statuses[strconv.FormatUint(n.Data[i].ID, 10)] = order.PendingCancel.String()
	}
	return statuses, nil
}

// openOrderDetails returns the account's open orders of an asset, on a pair when one is given
func (e *Exchange) openOrderDetails(ctx context.Context, p currency.Pair, a asset.Item) ([]order.Detail, error) {
	var orders []OrderResponse
	var err error
	if p.IsEmpty() {
		orders, err = e.GetOpenOrders(ctx, &OpenOrdersRequest{})
	} else {
		var symbol string
		if symbol, err = orderSymbol(a, p); err != nil {
			return nil, err
		}
		orders, err = e.GetOpenOrdersBySymbol(ctx, &OpenOrdersBySymbolRequest{Symbol: symbol})
	}
	if err != nil {
		return nil, err
	}
	return e.orderDetails(orders, a)
}

// orderDetails returns orders as GCT records them, leaving out those of other assets when an asset is given
func (e *Exchange) orderDetails(orders []OrderResponse, a asset.Item) ([]order.Detail, error) {
	details := make([]order.Detail, 0, len(orders))
	for i := range orders {
		d, err := e.orderDetail(&orders[i])
		if err != nil {
			return nil, fmt.Errorf("order %d: %w", orders[i].ID, err)
		}
		if a.IsValid() && d.AssetType != a {
			continue
		}
		details = append(details, *d)
	}
	return details, nil
}

// GetOrderInfo returns an order, open or from the last two weeks' history, with the trades it made up to 10 days ago.
// An order whose trades show a fill its read does not is read again, with its trades, so its status and amounts
// include the fill; a fill made after that is left to the order's next read
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, p currency.Pair, a asset.Item) (*order.Detail, error) {
	id, err := parseOrderID(orderID)
	if err != nil {
		return nil, err
	}
	o, closed, err := e.orderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	trades, err := e.GetOrderTrades(ctx, &OrderTradesRequest{Symbol: o.Symbol, OrderID: id})
	if err != nil {
		return nil, err
	}
	if isOrderReadBehindTrades(o, trades) {
		if o, closed, err = e.orderByID(ctx, id); err != nil {
			return nil, err
		}
		if trades, err = e.GetOrderTrades(ctx, &OrderTradesRequest{Symbol: o.Symbol, OrderID: id}); err != nil {
			return nil, err
		}
	}
	d, err := e.orderDetail(o)
	if err != nil {
		return nil, err
	}
	if closed {
		d.Status = closedOrderStatus(d.Status)
	}
	if (!p.IsEmpty() && !d.Pair.Equal(p)) || (a.IsValid() && d.AssetType != a) {
		return nil, fmt.Errorf("%w: %s is a %s %s order", order.ErrOrderNotFound, orderID, d.AssetType, d.Pair)
	}
	for j := range trades {
		t := &trades[j]
		side, err := orderSideFromAmount(t.ExecutedAmount)
		if err != nil {
			return nil, err
		}
		// Bitfinex sends fees as negative, as charged
		d.Trades = append(d.Trades, order.TradeHistory{
			Price:     t.ExecutedPrice,
			Amount:    math.Abs(t.ExecutedAmount),
			Fee:       -t.Fee,
			FeeAsset:  t.FeeCurrency.String(),
			Exchange:  e.Name,
			TID:       strconv.FormatUint(t.ID, 10),
			Type:      d.Type,
			Side:      side,
			Timestamp: t.ExecutedAt.Time(),
			IsMaker:   t.Maker == 1,
		})
		d.Fee -= t.Fee
	}
	if isOrderStatusCut(o.Status) {
		executed := cutOrderExecutedAmount(o, trades)
		d.Amount, d.ExecutedAmount = executed.Add(decimal.MustFromFloat(d.RemainingAmount)).InexactFloat64(), executed.InexactFloat64()
	}
	return d, nil
}

// isOrderReadBehindTrades reports whether an order's trades show a fill its read does not: one made after the read, or
// more than a status listing every fill lists
func isOrderReadBehindTrades(o *OrderResponse, trades []AccountTradeResponse) bool {
	updated := o.UpdatedAt.Time()
	if updated.IsZero() {
		return false
	}
	traded := decimal.Zero
	for i := range trades {
		if trades[i].ExecutedAt.Time().After(updated) {
			return true
		}
		traded = traded.Add(decimal.MustFromFloat(math.Abs(trades[i].ExecutedAmount)))
	}
	return !isOrderStatusCut(o.Status) && traded.GreaterThan(orderStatusExecutedAmount(o.Status))
}

// cutOrderExecutedAmount returns what an order whose status was cut short had executed when read: what its trades up to
// the read executed, when they count every fill, as they do for an order created within their reach that count at least
// the fills its status lists, and otherwise the estimate from its amounts, or its trades when they count more. Without
// the read's update time, its trades cannot be told from fills made since
func cutOrderExecutedAmount(o *OrderResponse, trades []AccountTradeResponse) decimal.Decimal {
	estimate := orderExecutedAmount(o)
	updated := o.UpdatedAt.Time()
	if updated.IsZero() {
		return estimate
	}
	traded := decimal.Zero
	for i := range trades {
		if !trades[i].ExecutedAt.Time().After(updated) {
			traded = traded.Add(decimal.MustFromFloat(math.Abs(trades[i].ExecutedAmount)))
		}
	}
	complete := time.Since(o.CreatedAt.Time()) < orderTradesReach && traded.GreaterThanOrEqual(orderStatusExecutedAmount(o.Status))
	if complete || traded.GreaterThan(estimate) {
		return traded
	}
	return estimate
}

// orderByID returns an order, open or from the last two weeks' history, and whether it was from the history
func (e *Exchange) orderByID(ctx context.Context, id uint64) (o *OrderResponse, closed bool, err error) {
	orders, err := e.GetOpenOrders(ctx, &OpenOrdersRequest{IDs: []uint64{id}})
	if err != nil {
		return nil, false, err
	}
	i := slices.IndexFunc(orders, func(o OrderResponse) bool { return o.ID == id })
	if i == -1 {
		if orders, err = e.GetOrdersHistory(ctx, &OrdersHistoryRequest{IDs: []uint64{id}}); err != nil {
			return nil, false, err
		}
		if i = slices.IndexFunc(orders, func(o OrderResponse) bool { return o.ID == id }); i == -1 {
			return nil, false, fmt.Errorf("%w: %d", order.ErrOrderNotFound, id)
		}
		closed = true
	}
	return &orders[i], closed, nil
}

// GetActiveOrders returns the account's open orders of an asset, on the request's pairs, or every pair when it names
// none
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	orders, err := e.GetOpenOrders(ctx, &OpenOrdersRequest{})
	if err != nil {
		return nil, err
	}
	details, err := e.orderDetails(orders, req.AssetType)
	if err != nil {
		return nil, err
	}
	return req.Filter(e.Name, details), nil
}

// GetOrderHistory returns the account's closed and cancelled orders from up to two weeks ago, of an asset, on the
// request's pairs, or every pair when it names none
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	orderID := func(o *OrderResponse) uint64 { return o.ID }
	createdAt := func(o *OrderResponse) time.Time { return o.CreatedAt.Time() }
	var orders []OrderResponse
	if len(req.Pairs) == 0 {
		var err error
		orders, err = pageBack(req.StartTime, req.EndTime, ordersHistoryPageLimit, func(end time.Time) ([]OrderResponse, error) {
			return e.GetOrdersHistory(ctx, &OrdersHistoryRequest{Start: req.StartTime, End: end, Limit: ordersHistoryPageLimit})
		}, orderID, createdAt)
		if err != nil {
			return nil, err
		}
	}
	for _, p := range req.Pairs {
		symbol, err := orderSymbol(req.AssetType, p)
		if err != nil {
			return nil, err
		}
		pairOrders, err := pageBack(req.StartTime, req.EndTime, ordersHistoryPageLimit, func(end time.Time) ([]OrderResponse, error) {
			return e.GetOrdersHistoryBySymbol(ctx, &OrdersHistoryBySymbolRequest{Symbol: symbol, Start: req.StartTime, End: end, Limit: ordersHistoryPageLimit})
		}, orderID, createdAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, pairOrders...)
	}
	details, err := e.orderDetails(orders, req.AssetType)
	if err != nil {
		return nil, err
	}
	for i := range details {
		details[i].Status = closedOrderStatus(details[i].Status)
	}
	return req.Filter(e.Name, details), nil
}

// GetDepositAddress returns an address depositing a currency, over a chain when it has several, into a wallet: the
// account ID names exchange, margin or funding, and funding takes deposits when it names none
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, accountID, chain string) (*deposit.Address, error) {
	if accountID == "" {
		accountID = WalletTypeFunding
	}
	method, err := e.transferMethod(ctx, c, chain)
	if err != nil {
		return nil, err
	}
	n, err := e.RequestDepositAddress(ctx, &DepositAddressRequest{Wallet: accountID, Method: method})
	if err != nil {
		return nil, err
	}
	// A currency needing a tag, memo or payment ID has its address in the pool address, and the tag in its place
	if n.Data.PoolAddress != "" {
		return &deposit.Address{Address: n.Data.PoolAddress, Tag: n.Data.Address, Chain: method}, nil
	}
	return &deposit.Address{Address: n.Data.Address, Chain: method}, nil
}

// WithdrawCryptocurrencyFunds withdraws a cryptocurrency, over a chain when it has several, from the exchange wallet,
// or the wallet the request's wallet ID names
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, w *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := w.Validate(); err != nil {
		return nil, err
	}
	method, err := e.transferMethod(ctx, w.Currency, w.Crypto.Chain)
	if err != nil {
		return nil, err
	}
	wallet := w.WalletID
	if wallet == "" {
		wallet = WalletTypeExchange
	}
	travel, err := withdrawalTravelRule(&w.Travel)
	if err != nil {
		return nil, err
	}
	req := &WithdrawalRequest{Wallet: wallet, Method: method, Amount: w.Amount, Address: w.Crypto.Address, PaymentID: w.Crypto.AddressTag, Note: w.Description, TravelRule: travel}
	n, err := e.RequestWithdrawal(ctx, req)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: strconv.FormatUint(n.Data.ID, 10), Status: n.Status}, nil
}

// withdrawalTravelRule returns the travel rule details of a withdrawal, or none when it names no beneficiary and no
// provider: the account itself as the beneficiary, or the person it names, whose name Bitfinex takes as first names
// and a last name, split at its last space, and the provider the withdrawal goes to
func withdrawalTravelRule(t *withdraw.TravelRule) (*TravelRuleDetails, error) {
	name := strings.Fields(t.BeneficiaryName)
	if !t.IsSelf && len(name) == 0 && t.BeneficiaryFinancialInstitution == "" {
		return nil, nil
	}
	details := &TravelRuleDetails{BeneficiarySelf: t.IsSelf, VASPName: t.BeneficiaryFinancialInstitution}
	// The account's verification data names it, so a name given with it is not sent
	if t.IsSelf || len(name) == 0 {
		return details, nil
	}
	if len(name) == 1 {
		return nil, fmt.Errorf("%w: %q", errTravelRuleNameIncomplete, t.BeneficiaryName)
	}
	details.FirstName, details.LastName = strings.Join(name[:len(name)-1], " "), name[len(name)-1]
	return details, nil
}

// WithdrawFiatFunds withdraws fiat by wire from the exchange wallet, or the wallet the request's wallet ID names,
// through Bitfinex's v1 API, as v2 has no equivalent
func (e *Exchange) WithdrawFiatFunds(ctx context.Context, w *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := w.Validate(); err != nil {
		return nil, err
	}
	wallet := w.WalletID
	if wallet == "" {
		wallet = WalletTypeExchange
	}
	resp, err := e.RequestFiatWithdrawal(ctx, &FiatWithdrawalRequest{Wallet: wallet, Withdrawal: w})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: strconv.FormatUint(resp.WithdrawalID, 10), Status: resp.Status}, nil
}

// WithdrawFiatFundsToInternationalBank withdraws fiat by wire, as WithdrawFiatFunds does
func (e *Exchange) WithdrawFiatFundsToInternationalBank(ctx context.Context, w *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return e.WithdrawFiatFunds(ctx, w)
}

// GetFeeByType returns an estimate of a fee: trading fees from the account's summary, and withdrawal fees from
// Bitfinex's configuration
func (e *Exchange) GetFeeByType(ctx context.Context, f *exchange.FeeBuilder) (float64, error) {
	if err := common.NilGuard(f); err != nil {
		return 0, err
	}
	if !e.AreCredentialsValid(ctx) && f.FeeType == exchange.CryptocurrencyTradeFee {
		f.FeeType = exchange.OfflineTradeFee
	}
	var fee float64
	switch f.FeeType {
	case exchange.CryptocurrencyTradeFee:
		summary, err := e.GetAccountSummary(ctx)
		if err != nil {
			return 0, err
		}
		fee = tradingFeeRate(&summary.FeeRates, f.Pair, f.IsMaker) * f.PurchasePrice * f.Amount
	case exchange.CryptocurrencyWithdrawalFee:
		var err error
		if fee, err = e.cryptocurrencyWithdrawalFee(ctx, f.Pair.Base); err != nil {
			return 0, err
		}
	case exchange.InternationalBankDepositFee:
		// Bitfinex charges 0.1% of a wire deposit, and at least 60 dollars or euros
		fee = max(0.001*f.Amount, wireFeeMinimum(f.FiatCurrency, 60))
	case exchange.InternationalBankWithdrawalFee:
		// Bitfinex charges 0.1% of a wire withdrawal, and at least 100 dollars or euros
		fee = max(0.001*f.Amount, wireFeeMinimum(f.FiatCurrency, 100))
	case exchange.OfflineTradeFee:
		// Bitfinex charges no trading fees
	}
	// A maker rebate is no fee
	return max(fee, 0), nil
}

// cryptocurrencyWithdrawalFee returns what Bitfinex charges to withdraw a currency. Bitfinex prices Tether's tokens by
// chain instead, each under its method less the TETHER prefix, such as USE for tetheruse, so a token withdrawn over one
// chain is charged that chain's fee, and one withdrawn over several needs its chain, which a fee request cannot give
func (e *Exchange) cryptocurrencyWithdrawalFee(ctx context.Context, c currency.Code) (float64, error) {
	fees, err := e.GetCurrencyWithdrawalFees(ctx)
	if err != nil {
		return 0, err
	}
	feeOf := func(key string) (float64, bool) {
		i := slices.IndexFunc(fees, func(w CurrencyWithdrawalFeeResponse) bool { return strings.EqualFold(w.Currency.String(), key) })
		if i == -1 {
			return 0, false
		}
		return fees[i].Fee.Amount, true
	}
	c = bitfinexCurrency(c)
	if fee, ok := feeOf(c.String()); ok {
		return fee, nil
	}
	method, err := e.transferMethod(ctx, c, "")
	if err != nil && !errors.Is(err, errTransferMethodUnknown) {
		return 0, err
	}
	if chain, ok := strings.CutPrefix(method, "tether"); ok {
		if fee, ok := feeOf(chain); ok {
			return fee, nil
		}
	}
	return 0, fmt.Errorf("%w: no withdrawal fee for %s", currency.ErrCurrencyNotSupported, c)
}

// wireFeeMinimum returns the least Bitfinex charges for a wire in a currency, which it sets for dollars and euros
func wireFeeMinimum(c currency.Code, minimum float64) float64 {
	if c.Equal(currency.USD) || c.Equal(currency.EUR) {
		return minimum
	}
	return 0
}

// tradingFeeRate returns the account's fee rate trading a pair, which Bitfinex sets for derivatives apart, and for other
// pairs by what they are quoted in: fiat, a stablecoin or another cryptocurrency
func tradingFeeRate(rates *SummaryFeeRates, p currency.Pair, maker bool) float64 {
	if isDerivativePair(p) {
		if maker {
			// Bitfinex reports what a derivative maker is paid, the fee negated
			return -rates.Maker.DerivativeRebate
		}
		return rates.Taker.Derivative
	}
	quote := bitfinexCurrency(p.Quote)
	switch {
	case maker && quote.IsFiatCurrency():
		return rates.Maker.ToFiat
	case maker && (quote.IsStableCurrency() || quote.Equal(currency.UST)):
		return rates.Maker.ToStable
	case maker:
		return rates.Maker.ToCrypto
	case quote.IsFiatCurrency():
		return rates.Taker.ToFiat
	case quote.IsStableCurrency() || quote.Equal(currency.UST):
		return rates.Taker.ToStable
	}
	return rates.Taker.ToCrypto
}

// bitfinexCurrency returns the code Bitfinex names a currency by: UST for Tether's USDT
func bitfinexCurrency(c currency.Code) currency.Code {
	if c.Equal(currency.USDT) {
		return currency.UST
	}
	return c
}

// transferMethod returns the method a currency is deposited and withdrawn over: the chain given, which may be
// required when the currency has several
func (e *Exchange) transferMethod(ctx context.Context, c currency.Code, chain string) (string, error) {
	methods, err := e.GetAvailableTransferChains(ctx, c)
	if err != nil {
		return "", err
	}
	switch {
	case chain != "":
		if i := slices.IndexFunc(methods, func(m string) bool { return strings.EqualFold(m, chain) }); i != -1 {
			return methods[i], nil
		}
		return "", fmt.Errorf("%w %s over %q, which has %s", errTransferMethodUnknown, c, chain, methods)
	case len(methods) > 1:
		return "", fmt.Errorf("%w: %s has %s", errTransferMethodNeeded, c, methods)
	}
	return methods[0], nil
}

// GetAvailableTransferChains returns the methods a currency is deposited and withdrawn over, in effect its chains, such
// as bitcoin, or tetheruse and tetherusx for USDT on Ethereum and Tron
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, c currency.Code) ([]string, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	methods, err := e.GetTransferMethods(ctx)
	if err != nil {
		return nil, err
	}
	c = bitfinexCurrency(c)
	var chains []string
	for i := range methods {
		if slices.ContainsFunc(methods[i].Currencies, c.Equal) {
			chains = append(chains, strings.ToLower(methods[i].Method))
		}
	}
	if len(chains) == 0 {
		return nil, fmt.Errorf("%w %s", errTransferMethodUnknown, c)
	}
	return chains, nil
}

// AuthenticateWebsocket authenticates the account connection
func (e *Exchange) AuthenticateWebsocket(ctx context.Context) error {
	conn, err := e.Websocket.GetConnection(wsAccountConnection)
	if err != nil {
		return err
	}
	return e.wsSendAuth(ctx, conn)
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, a asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, a)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(p, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	weekly, err := e.readWeeklyForTwoWeeks(req)
	if err != nil {
		return nil, err
	}
	if weekly {
		// GetKlineRequest counted the two-week candles against the request's limit, which one request reads weekly
		if count := kline.TotalCandlesPerInterval(req.Start, req.End, req.ExchangeInterval); count > req.RequestLimit {
			return nil, fmt.Errorf("%w: %d weekly candles make the two-week candles requested, more than the %d a request returns", kline.ErrRequestExceedsExchangeLimits, count, req.RequestLimit)
		}
	}
	candles, err := e.getCandles(ctx, req.Pair, req.Asset, req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval, over several requests
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(p, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	weekly, err := e.readWeeklyForTwoWeeks(req.Request)
	if err != nil {
		return nil, err
	}
	if weekly {
		// The weekly candles are paged, and matched with the ranges they fill, a week at a time
		if req.RangeHolder, err = req.GetRanges(req.RequestLimit); err != nil {
			return nil, err
		}
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for i := range req.RangeHolder.Ranges {
		candles, err := e.getCandles(ctx, req.Pair, req.Asset, req.ExchangeInterval, req.RangeHolder.Ranges[i].Start.Time, req.RangeHolder.Ranges[i].End.Time, req.RequestLimit)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, candles...)
	}
	return req.ProcessResponse(timeSeries)
}

// readWeeklyForTwoWeeks has a request for two-week candles read weekly ones, up to their limit, from which its candles
// are made, reporting whether it does: Bitfinex's 14D candles open every 14 days from the Unix epoch, on Thursdays,
// where GCT's two weeks open on Mondays, as its weeks do. The request is then the one GCT builds to make candles from a
// shorter interval, whose candles its callers count against the request's limit, and an extended request pages
func (e *Exchange) readWeeklyForTwoWeeks(r *kline.Request) (bool, error) {
	if r.ExchangeInterval != kline.TwoWeek {
		return false, nil
	}
	limit, err := e.Features.Enabled.Kline.GetIntervalResultLimit(kline.OneWeek)
	if err != nil {
		return false, err
	}
	r.ExchangeInterval, r.RequestLimit = kline.OneWeek, limit
	return true, nil
}

// getCandles returns a pair's, or a funding currency's, candles oldest first
func (e *Exchange) getCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time, limit uint64) ([]kline.Candle, error) {
	symbol, err := symbolFor(a, p)
	if err != nil {
		return nil, err
	}
	req := &CandlesHistoryRequest{Symbol: symbol, Interval: interval, Start: start, End: historyEnd(end), Limit: limit, Sort: 1}
	if a == asset.MarginFunding {
		req.Period = fundingCandlePeriod
	}
	candles, err := e.GetCandlesHistory(ctx, req)
	if err != nil {
		return nil, err
	}
	resp := make([]kline.Candle, len(candles))
	for i := range candles {
		resp[i] = kline.Candle{
			Time:   candles[i].Time.Time(),
			Open:   candles[i].Open,
			High:   candles[i].High,
			Low:    candles[i].Low,
			Close:  candles[i].Close,
			Volume: candles[i].Volume,
		}
	}
	return resp, nil
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(context.Context, asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// GetFuturesContractDetails returns the perpetual contracts of a futures asset, the derivative pairs its collateral
// margins, with each one's latest funding rate. Bitfinex's derivatives are all linear perpetual contracts, settled in
// their collateral
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, a asset.Item) ([]futures.Contract, error) {
	if !a.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !isDerivativeAsset(a) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	info, err := e.GetFuturesPairInfo(ctx)
	if err != nil {
		return nil, err
	}
	statuses, err := e.GetDerivativesStatus(ctx, nil)
	if err != nil {
		return nil, err
	}
	statusByKey := make(map[string]*DerivativeStatusResponse, len(statuses))
	for i := range statuses {
		statusByKey[statuses[i].Key] = &statuses[i].Status
	}
	contracts := make([]futures.Contract, 0, len(info))
	for i := range info {
		if strings.Contains(info[i].Pair, testListingMarker) {
			continue
		}
		p, err := pairFromSymbol(info[i].Pair)
		if err != nil {
			return nil, err
		}
		if derivativeAsset(p.Quote) != a {
			continue
		}
		c := futures.Contract{
			Exchange:           e.Name,
			Name:               p,
			Underlying:         currency.NewPair(underlyingCurrency(p.Base), underlyingCurrency(p.Quote)),
			Asset:              a,
			StartDate:          info[i].Info.FirstTradeTime.Time(),
			IsActive:           true,
			Type:               futures.Perpetual,
			SettlementType:     futures.Linear,
			SettlementCurrency: p.Quote,
			MarginCurrency:     p.Quote,
			Multiplier:         1,
		}
		// The margin opening a position needs bounds its leverage
		if initial := info[i].Info.InitialMargin; initial > 0 {
			c.MaxLeverage = 1 / initial
		}
		if s, ok := statusByKey["t"+info[i].Pair]; ok {
			c.LatestRate = latestFundingRate(s)
			// Bitfinex caps a funding payment, either way
			c.FundingRateFloor = decimal.MustFromFloat(-s.ClampMaximum)
			c.FundingRateCeiling = decimal.MustFromFloat(s.ClampMaximum)
		}
		contracts = append(contracts, c)
	}
	return contracts, nil
}

// latestFundingRate returns the funding rate Bitfinex set at a derivative's latest funding event, which it reports
// until the next, a funding interval on
func latestFundingRate(s *DerivativeStatusResponse) fundingrate.Rate {
	r := fundingrate.Rate{Rate: decimal.MustFromFloat(s.CurrentFunding)}
	if next := s.NextFundingTime.Time(); !next.IsZero() {
		r.Time = next.Add(-derivativeFundingInterval)
	}
	return r
}

// GetLatestFundingRates returns the funding rates Bitfinex set at the latest funding events of a futures asset's pair,
// or of its enabled pairs when none is given, with the time of each pair's next. Bitfinex reports the funding accruing
// towards the next event rather than a predicted rate, so one cannot be asked for
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if !isDerivativeAsset(r.Asset) {
		return nil, fmt.Errorf("%w: %s %s", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w: predicted funding rates", common.ErrFunctionNotSupported)
	}
	pairs := currency.Pairs{r.Pair}
	if r.Pair.IsEmpty() {
		var err error
		if pairs, err = e.GetEnabledPairs(r.Asset); err != nil {
			return nil, err
		}
		if len(pairs) == 0 {
			return nil, fmt.Errorf("%w: %s", currency.ErrCurrencyPairsEmpty, r.Asset)
		}
	}
	keys := make([]string, len(pairs))
	for i, p := range pairs {
		var err error
		if keys[i], err = symbolFor(r.Asset, p); err != nil {
			return nil, err
		}
	}
	statuses, err := e.GetDerivativesStatus(ctx, keys)
	if err != nil {
		return nil, err
	}
	resp := make([]fundingrate.LatestRateResponse, len(pairs))
	for i, p := range pairs {
		j := slices.IndexFunc(statuses, func(s DerivativeStatusEntryResponse) bool { return s.Key == keys[i] })
		if j == -1 {
			return nil, fmt.Errorf("%w: %s %s", fundingrate.ErrNoFundingRatesFound, r.Asset, p)
		}
		resp[i] = fundingrate.LatestRateResponse{
			Exchange:       e.Name,
			Asset:          r.Asset,
			Pair:           p,
			LatestRate:     latestFundingRate(&statuses[j].Status),
			TimeOfNextRate: statuses[j].Status.NextFundingTime.Time(),
			TimeChecked:    time.Now(),
		}
	}
	return resp, nil
}

// GetHistoricalFundingRates returns the funding rates Bitfinex set at a perpetual contract's funding events within a
// window, from its status history: each status snapshot reports the rate set at the event before it. Funding payments
// and predicted rates are not supported, as Bitfinex's ledgers record payments by currency rather than by contract, and
// it gives no predicted rate
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if !isDerivativeAsset(r.Asset) {
		return nil, fmt.Errorf("%w: %s %s", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	switch {
	case r.IncludePayments:
		return nil, fmt.Errorf("%w: funding payments", common.ErrFunctionNotSupported)
	case r.IncludePredictedRate:
		return nil, fmt.Errorf("%w: predicted funding rates", common.ErrFunctionNotSupported)
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	symbol, err := symbolFor(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	// The snapshots up to an interval after the window report the rates of the window's last events
	snapshots, err := pageBack(r.StartDate, historyEnd(r.EndDate.Add(derivativeFundingInterval)), statusHistoryPageLimit, func(pageEnd time.Time) ([]DerivativeStatusResponse, error) {
		return e.GetDerivativesStatusHistory(ctx, &DerivativesStatusHistoryRequest{Key: symbol, Start: r.StartDate, End: pageEnd, Limit: statusHistoryPageLimit, Sort: -1})
	}, func(s *DerivativeStatusResponse) int64 { return s.Time.Time().UnixMilli() }, func(s *DerivativeStatusResponse) time.Time { return s.Time.Time() })
	if err != nil {
		return nil, err
	}
	var rates []fundingrate.Rate
	seen := make(map[int64]struct{})
	for i := range snapshots {
		rate := latestFundingRate(&snapshots[i])
		if rate.Time.IsZero() || rate.Time.Before(r.StartDate) || rate.Time.After(r.EndDate) {
			continue
		}
		if _, ok := seen[rate.Time.UnixMilli()]; ok {
			continue
		}
		seen[rate.Time.UnixMilli()] = struct{}{}
		rates = append(rates, rate)
	}
	if len(rates) == 0 {
		return nil, fmt.Errorf("%w: %s %s from %s to %s", fundingrate.ErrNoFundingRatesFound, r.Asset, r.Pair, r.StartDate, r.EndDate)
	}
	slices.SortFunc(rates, func(a, b fundingrate.Rate) int { return a.Time.Compare(b.Time) })
	return &fundingrate.HistoricalRates{
		Exchange:     e.Name,
		Asset:        r.Asset,
		Pair:         r.Pair,
		StartDate:    rates[0].Time,
		EndDate:      rates[len(rates)-1].Time,
		LatestRate:   rates[len(rates)-1],
		FundingRates: rates,
	}, nil
}

// IsPerpetualFutureCurrency reports whether an asset trades perpetual contracts, as Bitfinex's derivatives all are
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return isDerivativeAsset(a), nil
}

// GetCollateralCurrencyForContract returns the currency a perpetual contract's positions are collateralised in, its
// quote, such as USTF0, which the derivatives wallet holds
func (e *Exchange) GetCollateralCurrencyForContract(a asset.Item, p currency.Pair) (currency.Code, asset.Item, error) {
	if !isDerivativeAsset(a) {
		return currency.EMPTYCODE, asset.Empty, fmt.Errorf("%w: %s", futures.ErrNotPerpetualFuture, a)
	}
	if _, err := symbolFor(a, p); err != nil {
		return currency.EMPTYCODE, asset.Empty, err
	}
	return p.Quote.Upper(), a, nil
}

// GetOpenInterest returns the open interest of perpetual contracts in their base currency: of the pairs given, or of
// every derivative pair Bitfinex lists
func (e *Exchange) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
	symbols := make([]string, len(keys))
	for i := range keys {
		if !isDerivativeAsset(keys[i].Asset) {
			return nil, fmt.Errorf("%w: %s %s", asset.ErrNotSupported, keys[i].Asset, keys[i].Pair())
		}
		var err error
		if symbols[i], err = symbolFor(keys[i].Asset, keys[i].Pair()); err != nil {
			return nil, err
		}
	}
	statuses, err := e.GetDerivativesStatus(ctx, symbols)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.OpenInterest, 0, len(statuses))
	for i := range statuses {
		if len(keys) == 0 && strings.Contains(statuses[i].Key, testListingMarker) {
			continue
		}
		p, err := pairFromSymbol(statuses[i].Key)
		if err != nil {
			return nil, err
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, derivativeAsset(p.Quote), p),
			OpenInterest: statuses[i].Status.OpenInterest,
		})
	}
	return resp, nil
}

// SetLeverage is not supported, as Bitfinex takes a derivative's leverage with each order, from order.Submit's
// Leverage, rather than as a setting
func (e *Exchange) SetLeverage(context.Context, asset.Item, currency.Pair, margin.Type, float64, order.Side) error {
	return fmt.Errorf("%w: Bitfinex takes leverage with each order", common.ErrFunctionNotSupported)
}

// GetLeverage returns the leverage of the active position on a perpetual contract; Bitfinex takes leverage with each
// order, so a contract has none without a position
func (e *Exchange) GetLeverage(ctx context.Context, a asset.Item, p currency.Pair, marginType margin.Type, _ order.Side) (float64, error) {
	if marginType != margin.Isolated {
		return 0, fmt.Errorf("%w: %s, as each derivative position holds its own collateral", margin.ErrMarginTypeUnsupported, marginType)
	}
	pos, err := e.derivativePosition(ctx, a, p)
	if err != nil {
		return 0, err
	}
	return pos.Leverage, nil
}

// derivativePosition returns the active position on a perpetual contract
func (e *Exchange) derivativePosition(ctx context.Context, a asset.Item, p currency.Pair) (*PositionResponse, error) {
	if !isDerivativeAsset(a) {
		return nil, fmt.Errorf("%w: %s", futures.ErrNotPerpetualFuture, a)
	}
	symbol, err := symbolFor(a, p)
	if err != nil {
		return nil, err
	}
	positions, err := e.GetActivePositions(ctx)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(positions, func(pos PositionResponse) bool { return pos.Symbol == symbol })
	if i == -1 {
		return nil, fmt.Errorf("%w: %s %s", futures.ErrNoPositionsFound, a, p)
	}
	return &positions[i], nil
}

// GetFuturesPositionSummary returns the active position on a perpetual contract, which holds its own collateral, as an
// isolated margin position does
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, r *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	pos, err := e.derivativePosition(ctx, r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	return &futures.PositionSummary{
		Pair:                      r.Pair,
		Asset:                     r.Asset,
		MarginType:                margin.Isolated,
		Currency:                  r.Pair.Quote.Upper(),
		StartDate:                 pos.CreatedAt.Time(),
		IsolatedMargin:            decimal.MustFromFloat(pos.Collateral),
		Leverage:                  decimal.MustFromFloat(pos.Leverage),
		EstimatedLiquidationPrice: decimal.MustFromFloat(pos.LiquidationPrice),
		CurrentSize:               decimal.MustFromFloat(pos.Amount),
		ContractSettlementType:    futures.Linear,
		AverageOpenPrice:          decimal.MustFromFloat(pos.BasePrice),
		UnrealisedPNL:             decimal.MustFromFloat(pos.ProfitLoss),
	}, nil
}

// GetFuturesPositionOrders returns the orders of perpetual contracts placed within a window, open or closed, from up to
// two weeks ago, which is as far back as Bitfinex's order history reaches
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, r *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if !isDerivativeAsset(r.Asset) {
		return nil, fmt.Errorf("%w: %s", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if len(r.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	start := r.StartDate
	if earliest := time.Now().Add(-orderHistoryReach); start.Before(earliest) {
		if !r.RespectOrderHistoryLimits {
			return nil, fmt.Errorf("%w: Bitfinex's order history reaches back to %s", futures.ErrOrderHistoryTooLarge, earliest.UTC().Format(time.RFC3339))
		}
		start = earliest
	}
	resp := make([]futures.PositionResponse, len(r.Pairs))
	for i, p := range r.Pairs {
		req := &order.MultiOrderRequest{AssetType: r.Asset, Pairs: currency.Pairs{p}, StartTime: start, EndTime: r.EndDate, Side: order.AnySide, Type: order.AnyType}
		closed, err := e.GetOrderHistory(ctx, req)
		if err != nil {
			return nil, err
		}
		open, err := e.openOrderDetails(ctx, p, r.Asset)
		if err != nil {
			return nil, err
		}
		resp[i] = futures.PositionResponse{
			Pair:                   p,
			Asset:                  r.Asset,
			ContractSettlementType: futures.Linear,
			Orders:                 append(req.Filter(e.Name, open), closed...),
		}
	}
	return resp, nil
}

// ChangePositionMargin sets the collateral of the active position on a perpetual contract, which holds its own, as an
// isolated margin position does
func (e *Exchange) ChangePositionMargin(ctx context.Context, r *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if !isDerivativeAsset(r.Asset) {
		return nil, fmt.Errorf("%w: %s", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if r.MarginType != margin.Isolated {
		return nil, fmt.Errorf("%w: %s, as each derivative position holds its own collateral", margin.ErrMarginTypeUnsupported, r.MarginType)
	}
	if r.NewAllocatedMargin <= 0 {
		return nil, margin.ErrNewAllocatedMarginRequired
	}
	symbol, err := symbolFor(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	set, err := e.SetDerivativePositionCollateral(ctx, &DerivativePositionCollateralRequest{Symbol: symbol, Collateral: r.NewAllocatedMargin})
	if err != nil {
		return nil, err
	}
	if !set.Success {
		return nil, fmt.Errorf("%w: %s %s", errPositionCollateralNotSet, r.Asset, r.Pair)
	}
	return &margin.PositionChangeResponse{
		Exchange:        e.Name,
		Pair:            r.Pair,
		Asset:           r.Asset,
		MarginType:      r.MarginType,
		AllocatedMargin: r.NewAllocatedMargin,
	}, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, p currency.Pair) (string, error) {
	if _, err := e.CurrencyPairs.IsPairEnabled(p, a); err != nil {
		return "", err
	}
	base, quote := p.Base.Upper().String(), p.Quote.Upper().String()
	switch a {
	case asset.Spot:
		return tradeBaseURL + "/t/" + base + ":" + quote + "?type=exchange", nil
	case asset.Margin:
		return tradeBaseURL + "/t/" + base + ":" + quote + "?type=margin", nil
	case asset.MarginFunding:
		return tradeBaseURL + "/f/" + base, nil
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		// A derivative pair trades as nothing else, so its page needs no type
		return tradeBaseURL + "/t/" + base + ":" + quote, nil
	}
	return "", fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
}
