package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"uuid"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/cache"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
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
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const (
	// tradesPageLimit is the most public trades a get-trades request returns, and tradesHistoryWindow the longest window
	// it serves
	tradesPageLimit     = 150
	tradesHistoryWindow = 7 * 24 * time.Hour
	// candlesPageLimit is the most candles a get-candlestick request returns
	candlesPageLimit = 300
	// privateHistoryWindow is the window a private history request defaults to before its end, and the most ccxt
	// records Crypto.com serving, so longer windows are requested a day at a time
	privateHistoryWindow = 24 * time.Hour
	// fundingHistoryWindow is how far before its end Crypto.com serves a funding history request, and
	// fundingHistoryPageLimit the most rates a request returns, more than the 336 of a window
	fundingHistoryWindow    = 14 * 24 * time.Hour
	fundingHistoryPageLimit = 1000
	// fundingInterval is how often a perpetual swap's funding settles
	fundingInterval = time.Hour
	// basisPoints is the number of basis points in one
	basisPoints = 10000
	// offlineMakerFeeRate and offlineTakerFeeRate are the highest rates of Crypto.com's spot fee schedule, its first level
	// without a CRO balance
	offlineMakerFeeRate = 0.0025
	offlineTakerFeeRate = 0.005
)

var (
	errMalformedInstrumentName   = errors.New("malformed instrument name")
	errNoDepositAddress          = errors.New("no deposit address")
	errChainRequired             = errors.New("chain required for a currency deposited on several networks")
	errNoDefaultNetwork          = errors.New("no default network")
	errNoIsolatedPosition        = errors.New("no isolated position")
	errLeverageNotWhole          = errors.New("account leverage must be a whole number")
	errOrderAssetMismatch        = errors.New("order asset mismatch")
	errRiskManagementUnsupported = errors.New("take profit and stop loss legs are unsupported")
	errQuoteAmountUnsupported    = errors.New("only a market buy is placed by its quote amount")
	errTriggerOrderQuantity      = errors.New("a trigger order is placed by its quantity")
	errTriggerOrderMargin        = errors.New("a trigger order takes neither leverage nor isolated margin")
	errLeverageUnsupported       = errors.New("leverage unsupported for spot orders")
	errTriggerPriceUnsupported   = errors.New("trigger price unsupported for orders other than trigger orders")
	errInvalidPriceType          = errors.New("invalid trigger price type")
	errTimeInForceNotAmendable   = errors.New("time in force cannot be amended")
	errClientOrderIDNotAmendable = errors.New("an advanced order's client order ID cannot be amended")
	errMarginNotFinite           = errors.New("margin must be a finite number")
	errIsolatedPositionNotOpen   = errors.New("isolated position not open yet")
)

// SetDefaults sets the basic defaults for Crypto.com
func (e *Exchange) SetDefaults() {
	e.Name = "Cryptodotcom"
	e.Enabled = true
	e.Verbose = false
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	// Spot and margin trade the same currency pairs, named with an underscore, and derivatives are named by their
	// underlying and a PERP or expiry date suffix after a dash
	for a, delimiter := range map[asset.Item]string{
		asset.Spot:          currency.UnderscoreDelimiter,
		asset.Margin:        currency.UnderscoreDelimiter,
		asset.PerpetualSwap: currency.DashDelimiter,
		asset.Futures:       currency.DashDelimiter,
	} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: delimiter},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: delimiter},
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
				TickerBatching:        true,
				TickerFetching:        true,
				KlineFetching:         true,
				TradeFetching:         true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				ModifyOrder:           true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				FundingRateFetching:   true,
				PredictedFundingRate:  true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
				// GetDepositAddress needs a chain for a currency deposited on several networks, so the engine's deposit address
				// sync asks for each of its chains rather than for an address without one
				MultiChainDepositRequiresChainSet: true,
			},
			WebsocketCapabilities: protocol.Features{
				AccountBalance:         true,
				GetOrders:              true,
				TickerFetching:         true,
				KlineFetching:          true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				FundingRateFetching:    true,
				PredictedFundingRate:   true,
				SubmitOrder:            true,
				SubmitOrders:           true,
				ModifyOrder:            true,
				CancelOrder:            true,
				DeadMansSwitch:         true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				MessageSequenceNumbers: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission | exchange.AutoWithdrawCryptoWithSetup,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals:  true,
				DateRanges: true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates:                    true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{kline.OneHour: true},
				Positions:                       true,
				CollateralMode:                  true,
				Leverage:                        true,
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
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.SevenDay},
					// Crypto.com's 14D candles open on alternate Mondays, a week from GoCryptoTrader's two-week boundaries,
					// so two-week candles are built from 7D ones
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: candlesPageLimit,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(rateLimits()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:         apiURL,
		exchange.WebsocketSpot:    websocketMarketDataURL,
		exchange.WebsocketPrivate: websocketUserURL,
		exchange.WebsocketTrade:   websocketUserURL,
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.websocketConnectDelay = defaultWebsocketConnectDelay
	e.bookResubscribeDelay = defaultBookResubscribeDelay
	e.marginOrders = cache.New(marginOrdersCapacity)
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

// FetchTradablePairs returns the pairs Crypto.com trades as an asset; the margin pairs are the spot pairs margin trading
// is enabled for, in either direction
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	instruments, err := e.GetInstruments(ctx)
	if err != nil {
		return nil, err
	}
	return tradablePairs(instruments, a)
}

// UpdateTradablePairs updates every asset's available pairs from one instruments request
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	instruments, err := e.GetInstruments(ctx)
	if err != nil {
		return err
	}
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := tradablePairs(instruments, a)
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, a, false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// tradablePairs returns the pairs of an asset's tradable instruments
func tradablePairs(instruments []InstrumentResponse, a asset.Item) (currency.Pairs, error) {
	pairs := make(currency.Pairs, 0, len(instruments))
	for i := range instruments {
		if !instruments[i].Tradable || !instrumentIsAsset(&instruments[i], a) {
			continue
		}
		p, ok, err := instrumentPair(&instruments[i], a)
		if err != nil {
			return nil, err
		}
		if ok {
			pairs = append(pairs, p)
		}
	}
	return pairs, nil
}

// instrumentIsAsset reports whether an instrument trades as an asset
func instrumentIsAsset(inst *InstrumentResponse, a asset.Item) bool {
	switch a {
	case asset.Spot:
		return inst.InstrumentType == InstrumentTypeCurrencyPair
	case asset.Margin:
		return inst.InstrumentType == InstrumentTypeCurrencyPair && (inst.MarginBuyEnabled || inst.MarginSellEnabled)
	case asset.PerpetualSwap:
		return inst.InstrumentType == InstrumentTypePerpetualSwap
	case asset.Futures:
		return inst.InstrumentType == InstrumentTypeFuture
	default:
		return false
	}
}

// instrumentPair returns the pair an asset holds an instrument as, and false for a spot instrument no pair names, such
// as SHIB_BTC@OEX_HK, whose name is not its currencies joined
func instrumentPair(inst *InstrumentResponse, a asset.Item) (p currency.Pair, ok bool, err error) {
	p, err = pairFromInstrument(a, inst.Symbol)
	if err != nil {
		return currency.EMPTYPAIR, false, err
	}
	if (a == asset.Spot || a == asset.Margin) && (!p.Base.Equal(inst.BaseCurrency) || !p.Quote.Equal(inst.QuoteCurrency)) {
		return currency.EMPTYPAIR, false, nil
	}
	return p, true, nil
}

// pairFromInstrument returns the pair of an instrument name as an asset's pairs hold it: a derivative's base is its
// underlying, such as BTCUSD, and its quote PERP or its expiry date, such as 261030, so the pair formats back to the
// name the websocket's pushes carry
func pairFromInstrument(a asset.Item, instrumentName string) (currency.Pair, error) {
	var delimiter string
	switch a {
	case asset.Spot, asset.Margin:
		delimiter = currency.UnderscoreDelimiter
	case asset.PerpetualSwap, asset.Futures:
		delimiter = currency.DashDelimiter
	default:
		return currency.EMPTYPAIR, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	i := strings.LastIndex(instrumentName, delimiter)
	if i <= 0 || i == len(instrumentName)-len(delimiter) {
		return currency.EMPTYPAIR, fmt.Errorf("%w: %q for %s", errMalformedInstrumentName, instrumentName, a)
	}
	return currency.NewPairWithDelimiter(instrumentName[:i], instrumentName[i+len(delimiter):], delimiter), nil
}

// instrumentName returns the instrument name of an asset's pair
func (e *Exchange) instrumentName(a asset.Item, p currency.Pair) (string, error) {
	if p.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	return e.FormatSymbol(p, a)
}

// orderAsset returns the asset an order or position on an instrument trades as; a spot instrument's order trades on
// margin when its execution instructions say so
func orderAsset(instrumentName string, executionInstructions []string) asset.Item {
	switch {
	case strings.HasSuffix(instrumentName, currency.DashDelimiter+"PERP"):
		return asset.PerpetualSwap
	case strings.Contains(instrumentName, currency.DashDelimiter):
		return asset.Futures
	case slices.Contains(executionInstructions, ExecutionInstructionMarginOrder):
		return asset.Margin
	default:
		return asset.Spot
	}
}

// UpdateOrderExecutionLimits loads the price and quantity steps of an asset's instruments; the smallest quantity
// Crypto.com takes is a quantity step
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	instruments, err := e.GetInstruments(ctx)
	if err != nil {
		return err
	}
	l := make([]limits.MinMaxLevel, 0, len(instruments))
	for i := range instruments {
		inst := &instruments[i]
		if !instrumentIsAsset(inst, a) {
			continue
		}
		p, ok, err := instrumentPair(inst, a)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, p),
			PriceStepIncrementSize:  inst.PriceTickSize.Float64(),
			AmountStepIncrementSize: inst.QuantityTickSize.Float64(),
			MinimumBaseAmount:       inst.QuantityTickSize.Float64(),
			Expiry:                  inst.ExpiryTime.Time(),
		})
	}
	return limits.Load(l)
}

// UpdateTickers updates the tickers of an asset's enabled pairs from one request for every instrument's
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	enabled, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	byInstrument := make(map[string]currency.Pair, len(enabled))
	for _, p := range enabled {
		name, err := e.instrumentName(a, p)
		if err != nil {
			return err
		}
		byInstrument[name] = p
	}
	tickers, err := e.GetTickers(ctx, &TickersRequest{})
	if err != nil {
		return err
	}
	var errs error
	for i := range tickers {
		p, ok := byInstrument[tickers[i].InstrumentName]
		if !ok {
			continue
		}
		if err := ticker.ProcessTicker(e.tickerPrice(&tickers[i], a, p)); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// UpdateTicker updates and returns a pair's ticker
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	name, err := e.instrumentName(a, p)
	if err != nil {
		return nil, err
	}
	tickers, err := e.GetTickers(ctx, &TickersRequest{InstrumentName: name})
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(tickers, func(t TickerResponse) bool { return t.InstrumentName == name })
	if i < 0 {
		return nil, fmt.Errorf("%w: %s %s", ticker.ErrTickerNotFound, a, p)
	}
	if err := ticker.ProcessTicker(e.tickerPrice(&tickers[i], a, p)); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// tickerPrice returns a ticker as the websocket's ticker channel does: its price change is a fraction, and its volume
// value is in USD whatever the instrument's quote currency, so it is the quote volume only of an instrument quoted in USD
func (e *Exchange) tickerPrice(t *TickerResponse, a asset.Item, p currency.Pair) *ticker.Price {
	price := &ticker.Price{
		Last:                t.LastPrice.Float64(),
		High:                t.High24Hour.Float64(),
		Low:                 t.Low24Hour.Float64(),
		Bid:                 t.BestBidPrice.Float64(),
		Ask:                 t.BestAskPrice.Float64(),
		BaseVolume:          t.Volume24Hour.Float64(),
		PercentChange24Hour: percentFromFraction(t.PriceChange24Hour),
		OpenInterest:        t.OpenInterest.Float64(),
		Pair:                p,
		ExchangeName:        e.Name,
		AssetType:           a,
		LastUpdated:         t.Timestamp.Time(),
	}
	if quotedInUSD(instrumentAsset{Asset: a, Pair: p}) {
		price.QuoteVolume = t.USDVolume24Hour.Float64()
	}
	return price
}

// percentFromFraction returns a fraction as a percentage, in decimal arithmetic so a fraction such as -0.0078 is -0.78
// rather than the -0.7799999999999999 a float multiplication gives. A quoted "NaN" or "Inf" decodes to a fraction no
// decimal holds, which is multiplied as a float instead
func percentFromFraction(fraction types.Number) float64 {
	if f := fraction.Float64(); math.IsNaN(f) || math.IsInf(f, 0) {
		return f * 100
	}
	return fraction.Decimal().Mul(decimal.NewFromInt(100)).InexactFloat64()
}

// UpdateOrderbook updates and returns a pair's order book, as deep as Crypto.com serves it by default
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	name, err := e.instrumentName(a, p)
	if err != nil {
		return nil, err
	}
	resp, err := e.GetOrderbook(ctx, &OrderbookRequest{InstrumentName: name})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("%w: %s %s", orderbook.ErrOrderbookNotFound, a, p)
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
		LastUpdated:       resp.Data[0].Timestamp.Time(),
		Bids:              orderbookLevels(resp.Data[0].Bids),
		Asks:              orderbookLevels(resp.Data[0].Asks),
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, a)
}

// orderbookLevels returns order book levels as GoCryptoTrader holds them
func orderbookLevels(levels []OrderbookLevel) orderbook.Levels {
	out := make(orderbook.Levels, len(levels))
	for i := range levels {
		out[i] = orderbook.Level{
			Price:      levels[i].Price.Float64(),
			Amount:     levels[i].Quantity.Float64(),
			OrderCount: int64(min(levels[i].OrderCount, math.MaxInt64)),
		}
	}
	return out
}

// UpdateAccountBalances returns the account's balances, which every asset trades from, as Crypto.com keeps one wallet
// for spot, margin and derivatives. A negative quantity is borrowed, the reserved quantity is held by open orders, and
// an isolated position's balances are held by that position
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	balances, err := e.GetUserBalance(ctx)
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{balanceSubAccount(a, balances)}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// balanceSubAccount returns an asset's sub account holding the account's position balances. An isolated position's
// balances are the account's too, but are allocated to that position rather than free in the wallet, so they are added
// once every wallet balance is set, as held, or as borrowed when negative. A currency's amounts are summed in decimal
// arithmetic and converted to floats once, so no rounding builds up as they are added
func balanceSubAccount(a asset.Item, balances []UserBalanceResponse) *accounts.SubAccount {
	type balanceSum struct {
		total, hold, free, borrowed decimal.Decimal
		updatedAt                   time.Time
	}
	sums := make(map[currency.Code]*balanceSum)
	for i := range balances {
		for j := range balances[i].PositionBalances {
			pb := &balances[i].PositionBalances[j]
			quantity, reserved := pb.Quantity.Decimal(), pb.ReservedQuantity.Decimal()
			s := &balanceSum{total: quantity, hold: reserved, free: quantity.Sub(reserved), updatedAt: pb.UpdateTime.Time()}
			if s.free.IsNegative() {
				s.free = decimal.Zero
			}
			if quantity.IsNegative() {
				s.borrowed = quantity.Neg()
			}
			sums[currency.NewCode(pb.InstrumentName)] = s
		}
	}
	for i := range balances {
		for j := range balances[i].IsolatedPositions {
			for k := range balances[i].IsolatedPositions[j].PositionBalances {
				pb := &balances[i].IsolatedPositions[j].PositionBalances[k]
				code := currency.NewCode(pb.InstrumentName)
				s, ok := sums[code]
				if !ok {
					s = new(balanceSum)
					sums[code] = s
				}
				quantity := pb.Quantity.Decimal()
				s.total = s.total.Add(quantity)
				if quantity.IsNegative() {
					s.borrowed = s.borrowed.Sub(quantity)
				} else {
					s.hold = s.hold.Add(quantity)
				}
				if updatedAt := pb.UpdateTime.Time(); updatedAt.After(s.updatedAt) {
					s.updatedAt = updatedAt
				}
			}
		}
	}
	subAcct := accounts.NewSubAccount(a, "")
	for code, s := range sums {
		subAcct.Balances.Set(code, accounts.Balance{
			Total:     s.total.InexactFloat64(),
			Hold:      s.hold.InexactFloat64(),
			Free:      s.free.InexactFloat64(),
			Borrowed:  s.borrowed.InexactFloat64(),
			UpdatedAt: s.updatedAt,
		})
	}
	return subAcct
}

// ValidateAPICredentials validates the API credentials by fetching the account's balances
func (e *Exchange) ValidateAPICredentials(ctx context.Context, a asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, a)
	return e.CheckTransientError(err)
}

// GetAccountFundingHistory returns the account's deposits and withdrawals of the last 90 days, the window Crypto.com
// defaults to
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	deposits, err := pageWalletHistory(ctx, func(ctx context.Context, page uint64) ([]DepositResponse, error) {
		return e.GetDepositHistory(ctx, &WalletHistoryRequest{PageSize: maxWalletHistoryPageSize, Page: page})
	}, func(d *DepositResponse) string { return d.ID })
	if err != nil {
		return nil, err
	}
	withdrawals, err := e.withdrawalHistory(ctx, currency.EMPTYCODE)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(deposits)+len(withdrawals))
	for i := range deposits {
		address, _ := splitAddressTag(deposits[i].Address)
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          depositStatus(deposits[i].Status),
			TransferID:      deposits[i].ID,
			Timestamp:       deposits[i].CreateTime.Time(),
			Currency:        deposits[i].Currency.String(),
			Amount:          deposits[i].Amount.Float64(),
			Fee:             deposits[i].Fee.Float64(),
			TransferType:    "deposit",
			CryptoToAddress: address,
			CryptoTxID:      deposits[i].TransactionID,
		})
	}
	for i := range withdrawals {
		address, _ := splitAddressTag(withdrawals[i].Address)
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          withdrawalStatus(withdrawals[i].Status),
			TransferID:      withdrawals[i].ID,
			Timestamp:       withdrawals[i].CreateTime.Time(),
			Currency:        withdrawalCurrency(&withdrawals[i]).String(),
			Amount:          withdrawals[i].Amount.Float64(),
			Fee:             withdrawals[i].Fee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: address,
			CryptoTxID:      withdrawals[i].TransactionID,
			CryptoChain:     withdrawals[i].NetworkID,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns the withdrawals of a currency, or of every currency, of the last 90 days, the window
// Crypto.com defaults to
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.withdrawalHistory(ctx, c)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals))
	for i := range withdrawals {
		address, _ := splitAddressTag(withdrawals[i].Address)
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawalStatus(withdrawals[i].Status),
			TransferID:      withdrawals[i].ID,
			Timestamp:       withdrawals[i].CreateTime.Time(),
			Currency:        withdrawalCurrency(&withdrawals[i]).String(),
			Amount:          withdrawals[i].Amount.Float64(),
			Fee:             withdrawals[i].Fee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: address,
			CryptoTxID:      withdrawals[i].TransactionID,
			CryptoChain:     withdrawals[i].NetworkID,
		}
	}
	return resp, nil
}

// withdrawalHistory returns the withdrawals of a currency, or of every currency, of the last 90 days
func (e *Exchange) withdrawalHistory(ctx context.Context, c currency.Code) ([]WithdrawalResponse, error) {
	return pageWalletHistory(ctx, func(ctx context.Context, page uint64) ([]WithdrawalResponse, error) {
		return e.GetWithdrawalHistory(ctx, &WalletHistoryRequest{Currency: c, PageSize: maxWalletHistoryPageSize, Page: page})
	}, func(w *WithdrawalResponse) string { return w.ID })
}

// pageWalletHistory returns every page of a deposit or withdrawal history, stopping at a short page, or at a page
// holding nothing new, as a venue ignoring the page would otherwise be asked for the same page forever
func pageWalletHistory[T any](ctx context.Context, fetch func(context.Context, uint64) ([]T, error), id func(*T) string) ([]T, error) {
	var resp []T
	seen := make(map[string]struct{})
	for page := uint64(0); ; page++ {
		records, err := fetch(ctx, page)
		if err != nil {
			return nil, err
		}
		var added int
		for i := range records {
			recordID := id(&records[i])
			if _, ok := seen[recordID]; ok {
				continue
			}
			seen[recordID] = struct{}{}
			resp = append(resp, records[i])
			added++
		}
		if len(records) < maxWalletHistoryPageSize || added == 0 {
			return resp, nil
		}
	}
}

// withdrawalCurrency returns a withdrawal's currency, which create-withdrawal's documented example names its symbol
func withdrawalCurrency(w *WithdrawalResponse) currency.Code {
	if w.Currency.IsEmpty() {
		return w.Symbol
	}
	return w.Currency
}

// depositStatus returns a deposit's status as text
func depositStatus(status string) string {
	switch status {
	case DepositStatusNotArrived:
		return "not arrived"
	case DepositStatusArrived:
		return "arrived"
	case DepositStatusFailed:
		return "failed"
	case DepositStatusPending:
		return "pending"
	default:
		return status
	}
}

// withdrawalStatus returns a withdrawal's status as text
func withdrawalStatus(status string) string {
	switch status {
	case WithdrawalStatusPending:
		return "pending"
	case WithdrawalStatusProcessing:
		return "processing"
	case WithdrawalStatusRejected:
		return "rejected"
	case WithdrawalStatusPaymentInProgress:
		return "payment in progress"
	case WithdrawalStatusPaymentFailed:
		return "payment failed"
	case WithdrawalStatusCompleted:
		return "completed"
	case WithdrawalStatusCancelled:
		return "cancelled"
	default:
		return status
	}
}

// splitAddressTag splits an address from the tag Crypto.com appends after a question mark
func splitAddressTag(address string) (addr, tag string) {
	addr, tag, _ = strings.Cut(address, "?")
	return addr, tag
}

// GetRecentTrades returns a pair's latest trades
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	name, err := e.instrumentName(a, p)
	if err != nil {
		return nil, err
	}
	trades, err := e.GetTrades(ctx, &TradesRequest{InstrumentName: name, Count: tradesPageLimit})
	if err != nil {
		return nil, err
	}
	resp, err := e.tradeData(trades, a, p)
	if err != nil {
		return nil, err
	}
	if e.IsSaveTradeDataEnabled() {
		if err := trade.AddTradesToBuffer(resp...); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// GetHistoricTrades returns a pair's trades from the start time, inclusive, to the end time, exclusive, oldest first,
// requesting seven days at a time, the longest window get-trades serves
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	name, err := e.instrumentName(a, p)
	if err != nil {
		return nil, err
	}
	pager := historyPager[TradeResponse]{
		exchangeName: e.Name,
		limit:        tradesPageLimit,
		fetch: func(ctx context.Context, start, end time.Time) ([]TradeResponse, error) {
			return e.GetTrades(ctx, &TradesRequest{InstrumentName: name, Count: tradesPageLimit, StartTime: start, EndTime: end})
		},
		record: func(t *TradeResponse) (string, time.Time) { return t.TradeID, tradeTime(t) },
	}
	trades, err := pager.page(ctx, start, end, tradesHistoryWindow)
	if err != nil {
		return nil, err
	}
	resp, err := e.tradeData(trades, a, p)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(resp, func(x, y trade.Data) int { return x.Timestamp.Compare(y.Timestamp) })
	if e.IsSaveTradeDataEnabled() {
		if err := trade.AddTradesToBuffer(resp...); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// tradeTime returns a public trade's time, to the nanosecond where Crypto.com sends it
func tradeTime(t *TradeResponse) time.Time {
	if ns := t.TradeTimeNano.Time(); !ns.IsZero() {
		return ns
	}
	return t.TradeTime.Time()
}

// tradeData returns public trades as GoCryptoTrader holds them
func (e *Exchange) tradeData(trades []TradeResponse, a asset.Item, p currency.Pair) ([]trade.Data, error) {
	resp := make([]trade.Data, len(trades))
	for i := range trades {
		side, err := orderSideFromString(trades[i].Side)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			TID:          trades[i].TradeID,
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    a,
			Side:         side,
			Price:        trades[i].Price.Float64(),
			Amount:       trades[i].Quantity.Float64(),
			Timestamp:    tradeTime(&trades[i]),
		}
	}
	return resp, nil
}

// historyPager pages back through a history whose requests return, newest first, the latest limit records of the
// window they name: fetch requests a window, and record returns a record's ID and time
type historyPager[T any] struct {
	exchangeName string
	limit        int
	fetch        func(ctx context.Context, start, end time.Time) ([]T, error)
	record       func(*T) (id string, at time.Time)
}

// page returns the records from the start time, inclusive, to the end time, exclusive, requesting a span of the window
// at a time from its end, as Crypto.com serves no longer span; a window ending after the current time ends at it, as
// Crypto.com refuses a request starting after it, and holds no later records
func (p *historyPager[T]) page(ctx context.Context, start, end time.Time, span time.Duration) ([]T, error) {
	if now := time.Now(); end.After(now) {
		end = now
	}
	var resp []T
	seen := make(map[string]struct{})
	for end.After(start) {
		spanStart := end.Add(-span)
		if spanStart.Before(start) {
			spanStart = start
		}
		records, err := p.backwards(ctx, spanStart, end, seen)
		if err != nil {
			return nil, err
		}
		resp = append(resp, records...)
		end = spanStart
	}
	return resp, nil
}

// backwards returns the records from the start time, inclusive, to the end time, exclusive, leaving out records
// outside the window and those seen holds, and adding those it returns to seen. Records can share a time, which a page
// can split, so each page ends a nanosecond after the oldest record so far. A page whose records all share its oldest
// time, which a page ending a nanosecond later would return again, passes over that time, as no request can page within
// it, and any more records of that time are left out with a warning
func (p *historyPager[T]) backwards(ctx context.Context, start, end time.Time, seen map[string]struct{}) ([]T, error) {
	var resp []T
	for end.After(start) {
		page, err := p.fetch(ctx, start, end)
		if err != nil {
			return nil, err
		}
		var added int
		oldest := end
		for i := range page {
			id, at := p.record(&page[i])
			if at.Before(oldest) {
				oldest = at
			}
			if _, ok := seen[id]; ok || at.Before(start) || !at.Before(end) {
				continue
			}
			seen[id] = struct{}{}
			resp = append(resp, page[i])
			added++
		}
		// A page reaching before the window holds the rest of it, and one holding nothing before its end holds none of it
		if len(page) < p.limit || oldest.Before(start) || !oldest.Before(end) {
			break
		}
		next := oldest.Add(time.Nanosecond)
		if added == 0 || !next.Before(end) {
			log.Warnf(log.ExchangeSys, "%s history holds %d or more records at %s, which no request can page within, so any more are left out", p.exchangeName, p.limit, oldest)
			next = oldest
		}
		end = next
	}
	return resp, nil
}

// GetHistoricCandles returns a pair's candles over a window of up to 300 candles
func (e *Exchange) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(p, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	candles, err := e.candles(ctx, req.RequestFormatted.String(), req.ExchangeInterval, req.Start, req.End)
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended returns a pair's candles over a window, 300 candles a request
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(p, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for i := range req.RangeHolder.Ranges {
		candles, err := e.candles(ctx, req.RequestFormatted.String(), req.ExchangeInterval, req.RangeHolder.Ranges[i].Start.Time, req.RangeHolder.Ranges[i].End.Time)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, candles...)
	}
	return req.ProcessResponse(timeSeries)
}

// candles returns an instrument's candles opening from the start time, inclusive, to the end time, exclusive. Crypto.com
// returns the latest of the candles overlapping its window, which for monthly candles takes in the one opening at its
// end, so the window ends a millisecond early, or that candle would take the place of the first
func (e *Exchange) candles(ctx context.Context, instrumentName string, interval kline.Interval, start, end time.Time) ([]kline.Candle, error) {
	resp, err := e.GetCandlestick(ctx, &CandlestickRequest{
		InstrumentName: instrumentName,
		Interval:       interval,
		Count:          candlesPageLimit,
		StartTime:      start,
		EndTime:        end.Add(-time.Millisecond),
	})
	if err != nil {
		return nil, err
	}
	candles := make([]kline.Candle, len(resp.Data))
	for i := range resp.Data {
		candles[i] = kline.Candle{
			Time:   resp.Data[i].StartTime.Time(),
			Open:   resp.Data[i].Open.Float64(),
			High:   resp.Data[i].High.Float64(),
			Low:    resp.Data[i].Low.Float64(),
			Close:  resp.Data[i].Close.Float64(),
			Volume: resp.Data[i].Volume.Float64(),
		}
	}
	return candles, nil
}

// GetDepositAddress returns a currency's active deposit address on a chain, a Crypto.com network ID, which may be left
// empty for a currency deposited on one network
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	addresses, err := e.GetDepositAddresses(ctx, c)
	if err != nil {
		return nil, err
	}
	var found *DepositAddressResponse
	var networks []string
	for i := range addresses {
		if addresses[i].Status != DepositAddressStatusActive || (chain != "" && !strings.EqualFold(addresses[i].Network, chain)) {
			continue
		}
		if found == nil {
			found = &addresses[i]
		}
		if !slices.Contains(networks, addresses[i].Network) {
			networks = append(networks, addresses[i].Network)
		}
	}
	switch {
	case found == nil && chain != "":
		return nil, fmt.Errorf("%w: %s on %s", errNoDepositAddress, c, chain)
	case found == nil:
		return nil, fmt.Errorf("%w: %s", errNoDepositAddress, c)
	case len(networks) > 1:
		return nil, fmt.Errorf("%w: %s on %s", errChainRequired, c, strings.Join(networks, ", "))
	}
	address, tag := splitAddressTag(found.Address)
	return &deposit.Address{Address: address, Tag: tag, Chain: found.Network}, nil
}

// GetAvailableTransferChains returns the networks a currency is deposited and withdrawn on
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, c currency.Code) ([]string, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	wc, err := e.walletCurrency(ctx, c)
	if err != nil {
		return nil, err
	}
	chains := make([]string, len(wc.Networks))
	for i := range wc.Networks {
		chains[i] = wc.Networks[i].NetworkID
	}
	return chains, nil
}

// walletCurrency returns a currency's networks
func (e *Exchange) walletCurrency(ctx context.Context, c currency.Code) (*WalletCurrency, error) {
	networks, err := e.GetCurrencyNetworks(ctx)
	if err != nil {
		return nil, err
	}
	for code, wc := range networks.Currencies {
		if code.Equal(c) {
			return &wc, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", currency.ErrCurrencyNotFound, c)
}

// WithdrawCryptocurrencyFunds withdraws a currency to a whitelisted address, on the request's chain or else the
// currency's default network
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, w *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := w.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.CreateWithdrawal(ctx, &WithdrawalRequest{
		ClientWithdrawalID: w.ClientOrderID,
		Currency:           w.Currency,
		Amount:             types.Number(w.Amount),
		Address:            w.Crypto.Address,
		AddressTag:         w.Crypto.AddressTag,
		NetworkID:          w.Crypto.Chain,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{Name: e.Name, ID: resp.ID, Status: withdrawalStatus(resp.Status)}, nil
}

// WithdrawFiatFunds is not supported, as Crypto.com withdraws fiat only to a bank account it holds, by its own ID,
// which a withdrawal request cannot name; CreateFiatWithdrawal takes it
func (e *Exchange) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank is not supported, for the reason WithdrawFiatFunds is not
func (e *Exchange) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns a trading fee, at the account's fee rate for the pair, an estimate of a trading fee, at the highest
// rate of Crypto.com's spot fee schedule, or a withdrawal fee, on the currency's default network or its only one
func (e *Exchange) GetFeeByType(ctx context.Context, f *exchange.FeeBuilder) (float64, error) {
	if f == nil {
		return 0, common.ErrNilPointer
	}
	switch f.FeeType {
	case exchange.OfflineTradeFee:
		rate := offlineTakerFeeRate
		if f.IsMaker {
			rate = offlineMakerFeeRate
		}
		return rate * f.PurchasePrice * f.Amount, nil
	case exchange.CryptocurrencyTradeFee:
		a, err := e.GetPairAssetType(f.Pair)
		if err != nil {
			return 0, err
		}
		name, err := e.instrumentName(a, f.Pair)
		if err != nil {
			return 0, err
		}
		rates, err := e.GetInstrumentFeeRate(ctx, name)
		if err != nil {
			return 0, err
		}
		rate := rates.EffectiveTakerRateBasisPoints.Float64()
		if f.IsMaker {
			rate = rates.EffectiveMakerRateBasisPoints.Float64()
		}
		return rate / basisPoints * f.PurchasePrice * f.Amount, nil
	case exchange.CryptocurrencyWithdrawalFee:
		wc, err := e.walletCurrency(ctx, f.Pair.Base)
		if err != nil {
			return 0, err
		}
		for i := range wc.Networks {
			if wc.Networks[i].NetworkID == wc.DefaultNetwork || len(wc.Networks) == 1 {
				return wc.Networks[i].WithdrawalFee.Float64(), nil
			}
		}
		return 0, fmt.Errorf("%w: %s", errNoDefaultNetwork, f.Pair.Base)
	default:
		return 0, common.ErrFunctionNotSupported
	}
}

// GetServerTime is not supported, as Crypto.com has no time endpoint
func (e *Exchange) GetServerTime(context.Context, asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// orderRequest is an order as one of the two create requests: a trigger order is placed by the advanced order methods,
// any other by the trading methods. The client order ID is the request's, given or generated
type orderRequest struct {
	order         *CreateOrderRequest
	trigger       *CreateAdvancedOrderRequest
	clientOrderID string
}

// newOrderRequest returns the request placing an order. A margin order is a spot order on margin, which a trigger order
// states by its MARGIN_ORDER execution instruction, and an isolated margin order a derivative order on an isolated
// position with its own leverage. A margin order without a client order ID is given a unique one, by which its trades,
// which can arrive before its order ID does, are told apart as margin; Crypto.com would give it the request's nonce,
// which other orders can share. The request is checked as creating it checks it, so that orders placed in lists are
// all checked before any list is sent
func (e *Exchange) newOrderRequest(s *order.Submit) (*orderRequest, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	if !e.SupportsAsset(s.AssetType) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, s.AssetType)
	}
	if s.RiskManagementModes.TakeProfit.Price != 0 || s.RiskManagementModes.StopLoss.Price != 0 || s.RiskManagementModes.StopEntry.Price != 0 {
		return nil, errRiskManagementUnsupported
	}
	name, err := e.instrumentName(s.AssetType, s.Pair)
	if err != nil {
		return nil, err
	}
	side, err := orderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	orderType, trigger, err := orderTypeString(s.Type)
	if err != nil {
		return nil, err
	}
	timeInForce, postOnly, err := timeInForceParams(s.TimeInForce)
	if err != nil {
		return nil, err
	}
	var instructions []string
	if postOnly {
		instructions = append(instructions, ExecutionInstructionPostOnly)
	}
	if s.ReduceOnly {
		instructions = append(instructions, ExecutionInstructionReduceOnly)
	}
	switch {
	case s.MarginType == margin.Isolated && !s.AssetType.IsFutures():
		return nil, fmt.Errorf("%w: %s for %s", margin.ErrMarginTypeUnsupported, s.MarginType, s.AssetType)
	case s.MarginType == margin.Isolated:
		instructions = append(instructions, ExecutionInstructionIsolatedMargin)
	}
	// A leverage of one is no leverage, which a spot or trigger order may state
	leverage := s.Leverage
	if leverage <= 1 && (s.AssetType == asset.Spot || trigger) {
		leverage = 0
	}
	if leverage != 0 && s.AssetType == asset.Spot {
		return nil, errLeverageUnsupported
	}
	clientOrderID := s.ClientOrderID
	if clientOrderID == "" && s.AssetType == asset.Margin {
		clientOrderID = uuid.NewV4().String()
	}
	if trigger {
		switch {
		case s.Amount == 0:
			return nil, errTriggerOrderQuantity
		case leverage != 0 || s.MarginType == margin.Isolated:
			return nil, errTriggerOrderMargin
		}
		priceType, err := referencePriceType(s.TriggerPriceType)
		if err != nil {
			return nil, err
		}
		if s.AssetType == asset.Margin {
			instructions = append(instructions, ExecutionInstructionMarginOrder)
		}
		r := &CreateAdvancedOrderRequest{
			InstrumentName:        name,
			Side:                  side,
			OrderType:             orderType,
			Quantity:              types.Number(s.Amount),
			ReferencePrice:        types.Number(s.TriggerPrice),
			ReferencePriceType:    priceType,
			ClientOrderID:         clientOrderID,
			TimeInForce:           timeInForce,
			ExecutionInstructions: instructions,
		}
		if s.Type == order.StopLimit || s.Type == order.TakeProfitLimit {
			r.Price = types.Number(s.Price)
		}
		if err := r.validate(); err != nil {
			return nil, err
		}
		return &orderRequest{trigger: r, clientOrderID: clientOrderID}, nil
	}
	if s.TriggerPrice != 0 {
		return nil, errTriggerPriceUnsupported
	}
	r := &CreateOrderRequest{
		InstrumentName:        name,
		Side:                  side,
		OrderType:             orderType,
		ClientOrderID:         clientOrderID,
		TimeInForce:           timeInForce,
		ExecutionInstructions: instructions,
		Leverage:              types.Number(leverage),
	}
	if s.Type == order.Limit {
		r.Price = types.Number(s.Price)
	}
	switch {
	case s.Amount > 0:
		r.Quantity = types.Number(s.Amount)
	case s.Type == order.Market && s.Side.IsLong():
		r.Notional = types.Number(s.QuoteAmount)
	default:
		return nil, errQuoteAmountUnsupported
	}
	if s.AssetType == asset.Margin {
		r.SpotMargin = SpotMarginMargin
	}
	if _, err := newCreateOrderParams(r); err != nil {
		return nil, err
	}
	return &orderRequest{order: r, clientOrderID: clientOrderID}, nil
}

// setIsolationID names the isolated position an isolated margin order trades, the account's on the order's instrument,
// if any; Crypto.com opens a new isolated position for an order naming none, and refuses that while one is open there
func (e *Exchange) setIsolationID(ctx context.Context, s *order.Submit, r *orderRequest) error {
	if s.MarginType != margin.Isolated || r.order == nil {
		return nil
	}
	id, err := e.isolatedPositionID(ctx, r.order.InstrumentName)
	if err != nil {
		return err
	}
	r.order.IsolationID = id
	return nil
}

// referencePriceType returns the Crypto.com reference price type of a trigger price type; GoCryptoTrader's zero price
// type is the last price, although Crypto.com's default is the mark price
func referencePriceType(t order.PriceType) (string, error) {
	switch t {
	case order.LastPrice:
		return ReferencePriceTypeLastPrice, nil
	case order.MarkPrice:
		return ReferencePriceTypeMarkPrice, nil
	case order.IndexPrice:
		return ReferencePriceTypeIndexPrice, nil
	default:
		return "", fmt.Errorf("%w: %s", errInvalidPriceType, t)
	}
}

// submitResponse returns a placed order's submit response, with the client order ID Crypto.com gave an order placed
// without one. Crypto.com answers before its matching engine takes the order, so the order is new whatever its type,
// rather than the filled DeriveSubmitResponse assumes of a market order
func submitResponse(s *order.Submit, resp *OrderIDResponse) (*order.SubmitResponse, error) {
	sr, err := s.DeriveSubmitResponse(resp.OrderID)
	if err != nil {
		return nil, err
	}
	sr.Status = order.New
	if sr.ClientOrderID == "" {
		sr.ClientOrderID = resp.ClientOrderID
	}
	return sr, nil
}

// SubmitOrder places an order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	return e.submitOrder(ctx, s, e.CreateOrder, e.CreateAdvancedOrder)
}

// WebsocketSubmitOrder places an order over the websocket
func (e *Exchange) WebsocketSubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	return e.submitOrder(ctx, s, e.WebsocketAPI().CreateOrder, e.WebsocketAPI().CreateAdvancedOrder)
}

// submitOrder places an order with the advanced order methods' create for a trigger order, and otherwise with the
// trading methods'. A margin order is tracked by its client order ID before it is placed, as its trades can arrive
// before its order ID does, until Crypto.com refuses it; one whose outcome is unknown, as when the request fails or is
// cancelled, may have been placed, so is still tracked. An isolated margin order is placed on its instrument's isolated
// position, when the account holds one
func (e *Exchange) submitOrder(ctx context.Context, s *order.Submit, create func(context.Context, *CreateOrderRequest) (*OrderIDResponse, error), createAdvanced func(context.Context, *CreateAdvancedOrderRequest) (*OrderIDResponse, error)) (*order.SubmitResponse, error) {
	r, err := e.newOrderRequest(s)
	if err != nil {
		return nil, err
	}
	if err := e.setIsolationID(ctx, s, r); err != nil {
		return nil, err
	}
	e.trackMarginOrder(s.AssetType, "", r.clientOrderID)
	var resp *OrderIDResponse
	if r.trigger != nil {
		resp, err = createAdvanced(ctx, r.trigger)
	} else {
		resp, err = create(ctx, r.order)
	}
	if err != nil {
		if _, refused := errors.AsType[*APIError](err); refused {
			e.untrackClientOrderID(r.clientOrderID)
		}
		return nil, err
	}
	e.trackMarginOrder(s.AssetType, resp.OrderID, resp.ClientOrderID)
	return submitResponse(s, resp)
}

// WebsocketSubmitOrders places orders over the websocket, in lists of up to 10, trigger orders in lists of their own;
// an order Crypto.com rejects has its submission error set. A list whose request fails, or whose answer cannot be read,
// stops the orders after it, and the responses of the lists placed before it are returned with the error, the rest left
// nil. Each list's margin orders are tracked as submitOrder tracks an order, by their client order IDs before the list is
// sent and by their order IDs once it is answered, and an isolated margin order is placed as submitOrder places one, on
// its instrument's isolated position. Orders naming none open a new isolated position each, and Crypto.com refuses a
// second on one instrument, so a second isolated margin order on an instrument without one is refused before any list
// is sent
func (e *Exchange) WebsocketSubmitOrders(ctx context.Context, orders []*order.Submit) ([]*order.SubmitResponse, error) {
	if len(orders) == 0 {
		return nil, errOrderListEmpty
	}
	var regular, triggers []int
	requests := make([]*orderRequest, len(orders))
	opening := make(map[string]int)
	for i, s := range orders {
		r, err := e.newOrderRequest(s)
		if err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
		if err := e.setIsolationID(ctx, s, r); err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
		if s.MarginType == margin.Isolated && r.order != nil && r.order.IsolationID == "" {
			if j, ok := opening[r.order.InstrumentName]; ok {
				return nil, fmt.Errorf("order %d: %w: order %d opens the one on %s, so place it first", i, errIsolatedPositionNotOpen, j, r.order.InstrumentName)
			}
			opening[r.order.InstrumentName] = i
		}
		requests[i] = r
		if r.trigger != nil {
			triggers = append(triggers, i)
		} else {
			regular = append(regular, i)
		}
	}
	responses := make([]*order.SubmitResponse, len(orders))
	for list := range slices.Chunk(regular, maxOrderListLength) {
		creates := make([]CreateOrderRequest, len(list))
		for j, i := range list {
			creates[j] = *requests[i].order
		}
		e.trackListSent(orders, requests, list)
		entries, err := e.WebsocketAPI().CreateOrderList(ctx, creates)
		if err != nil {
			e.trackListRefused(requests, list, err)
			return responses, err
		}
		if err := listSubmitResponses(orders, list, entries, responses); err != nil {
			return responses, err
		}
		e.trackListAnswered(orders, requests, list, responses)
	}
	for list := range slices.Chunk(triggers, maxAdvancedOrderListLength) {
		creates := make([]CreateAdvancedOrderRequest, len(list))
		for j, i := range list {
			creates[j] = *requests[i].trigger
		}
		e.trackListSent(orders, requests, list)
		entries, err := e.WebsocketAPI().CreateAdvancedOrderList(ctx, creates)
		if err != nil {
			e.trackListRefused(requests, list, err)
			return responses, err
		}
		if err := listSubmitResponses(orders, list, entries, responses); err != nil {
			return responses, err
		}
		e.trackListAnswered(orders, requests, list, responses)
	}
	return responses, nil
}

// trackListSent tracks the margin orders of a list about to be sent by their client order IDs
func (e *Exchange) trackListSent(orders []*order.Submit, requests []*orderRequest, list []int) {
	for _, i := range list {
		e.trackMarginOrder(orders[i].AssetType, "", requests[i].clientOrderID)
	}
}

// trackListRefused stops tracking the client order IDs of a list's orders when Crypto.com refused the list; a list
// whose request failed otherwise may have been placed
func (e *Exchange) trackListRefused(requests []*orderRequest, list []int, err error) {
	if _, refused := errors.AsType[*APIError](err); !refused {
		return
	}
	for _, i := range list {
		e.untrackClientOrderID(requests[i].clientOrderID)
	}
}

// trackListAnswered tracks the orders Crypto.com placed from an answered list by their order IDs, and stops tracking
// the client order IDs of those it rejected; an order the answer gives no outcome for may have been placed
func (e *Exchange) trackListAnswered(orders []*order.Submit, requests []*orderRequest, list []int, responses []*order.SubmitResponse) {
	for _, i := range list {
		if responses[i].SubmissionError == nil {
			e.trackMarginOrder(orders[i].AssetType, responses[i].OrderID, responses[i].ClientOrderID)
		} else if _, rejected := errors.AsType[*APIError](responses[i].SubmissionError); rejected {
			e.untrackClientOrderID(requests[i].clientOrderID)
		}
	}
}

// listSubmitResponses sets the submit responses of a list of orders, by their indexes in the submitted orders, from the
// list's entries, which give each order's outcome by its index in the list. An entry it cannot read leaves the list's
// responses unset, rather than reporting rejected the orders whose entries it has not read
func listSubmitResponses(orders []*order.Submit, list []int, entries []OrderListEntryResponse, responses []*order.SubmitResponse) error {
	listResponses := make([]*order.SubmitResponse, len(list))
	for k, i := range list {
		listResponses[k] = &order.SubmitResponse{
			Exchange:        orders[i].Exchange,
			Type:            orders[i].Type,
			Side:            orders[i].Side,
			Pair:            orders[i].Pair,
			AssetType:       orders[i].AssetType,
			ClientOrderID:   orders[i].ClientOrderID,
			Status:          order.Rejected,
			SubmissionError: fmt.Errorf("%w: no result for the order", order.ErrUnableToPlaceOrder),
		}
	}
	for j := range entries {
		index := entries[j].Index.Int64()
		if index < 0 || index >= int64(len(list)) {
			continue
		}
		if code := entries[j].Code.Int64(); code != 0 {
			listResponses[index].SubmissionError = fmt.Errorf("%w: %w", order.ErrUnableToPlaceOrder, &APIError{Code: code, Message: entries[j].Message})
			continue
		}
		sr, err := submitResponse(orders[list[index]], &OrderIDResponse{OrderID: entries[j].OrderID, ClientOrderID: entries[j].ClientOrderID})
		if err != nil {
			return err
		}
		listResponses[index] = sr
	}
	for k, i := range list {
		responses[i] = listResponses[k]
	}
	return nil
}

// orderAmendment is an order modification as one of the two amend requests: a trigger order is amended by the advanced
// order methods, any other by the trading methods. The time in force is the order's, known when it was fetched, as is
// its type when the modification names none
type orderAmendment struct {
	order       *AmendOrderRequest
	trigger     *AmendAdvancedOrderRequest
	price       float64
	amount      float64
	timeInForce order.TimeInForce
	orderType   order.Type
}

// newOrderAmendment returns the request amending an order. Crypto.com takes an amended order's quantity and limit price
// together, so those the modification leaves unset are restated from the order, which is fetched for them, for its type
// when the modification does not give it, and for its time in force when the modification gives one, which an
// amendment cannot change; an order the advanced order methods hold, such as a leg of an OTO list, is amended by them
// whatever its type
func (e *Exchange) newOrderAmendment(ctx context.Context, m *order.Modify) (*orderAmendment, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if !e.SupportsAsset(m.AssetType) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, m.AssetType)
	}
	a := &orderAmendment{price: m.Price, amount: m.Amount}
	advanced := isTriggerOrder(m.Type)
	marketTrigger := m.Type == order.Stop || m.Type == order.StopMarket || m.Type == order.TakeProfit || m.Type == order.TakeProfitMarket
	if a.amount == 0 || (a.price == 0 && !marketTrigger) || m.Type == order.UnknownType || m.Type == order.AnyType || m.TimeInForce != order.UnknownTIF {
		o, fromAdvanced, err := e.orderByID(ctx, m.OrderID, m.ClientOrderID)
		if err != nil {
			return nil, err
		}
		orderType, err := orderTypeFromString(o.OrderType)
		if err != nil {
			return nil, err
		}
		advanced = fromAdvanced || isTriggerOrder(orderType)
		if m.Type == order.UnknownType || m.Type == order.AnyType {
			// The order's own type decides, as a price sent for a trigger market order would make it a trigger limit order
			marketTrigger = orderType == order.Stop || orderType == order.TakeProfit
			a.orderType = orderType
		}
		if a.timeInForce, err = timeInForceFromResponse(o.TimeInForce, o.ExecutionInstructions); err != nil {
			return nil, err
		}
		// Crypto.com amends only an order's quantity and prices, so a time in force other than the order's is refused
		// rather than reported as amended
		if m.TimeInForce != order.UnknownTIF && effectiveTimeInForce(m.TimeInForce) != effectiveTimeInForce(a.timeInForce) {
			return nil, fmt.Errorf("%w: %s, not %s", errTimeInForceNotAmendable, a.timeInForce, m.TimeInForce)
		}
		if a.amount == 0 {
			a.amount = o.Quantity.Float64()
		}
		if a.price == 0 {
			a.price = o.LimitPrice.Float64()
		}
	}
	if marketTrigger {
		// A limit price would make the trigger market order a trigger limit order
		a.price = 0
	}
	if advanced {
		// The advanced order methods' amend takes a client order ID only to name the order, so a new one would be dropped
		if m.NewClientOrderID != "" {
			return nil, fmt.Errorf("%w: %q", errClientOrderIDNotAmendable, m.NewClientOrderID)
		}
		price := types.Number(a.price)
		a.trigger = &AmendAdvancedOrderRequest{
			OrderID:           m.OrderID,
			ClientOrderID:     m.ClientOrderID,
			NewQuantity:       types.Number(a.amount),
			NewPrice:          &price,
			NewReferencePrice: types.Number(m.TriggerPrice),
		}
		return a, nil
	}
	if m.TriggerPrice != 0 {
		return nil, errTriggerPriceUnsupported
	}
	a.order = &AmendOrderRequest{
		OrderID:               m.OrderID,
		OriginalClientOrderID: m.ClientOrderID,
		ClientOrderID:         m.NewClientOrderID,
		NewPrice:              types.Number(a.price),
		NewQuantity:           types.Number(a.amount),
	}
	return a, nil
}

// modifyResponse returns an amended order's modify response, with the quantity and price it was amended to, and the
// order's time in force where it is known, which the amendment does not change, as is its type where the modification
// names none
func modifyResponse(m *order.Modify, a *orderAmendment, resp *OrderIDResponse) (*order.ModifyResponse, error) {
	mr, err := m.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	mr.Price = a.price
	mr.Amount = a.amount
	mr.TimeInForce = a.timeInForce
	if a.orderType != order.UnknownType {
		mr.Type = a.orderType
	}
	if resp.OrderID != "" {
		mr.OrderID = resp.OrderID
	}
	if resp.ClientOrderID != "" {
		mr.ClientOrderID = resp.ClientOrderID
	}
	return mr, nil
}

// ModifyOrder amends an order's quantity, price or trigger price
func (e *Exchange) ModifyOrder(ctx context.Context, m *order.Modify) (*order.ModifyResponse, error) {
	a, err := e.newOrderAmendment(ctx, m)
	if err != nil {
		return nil, err
	}
	resp, err := amendOrder(ctx, a, e.AmendOrder, e.AmendAdvancedOrder)
	if err != nil {
		return nil, err
	}
	return modifyResponse(m, a, resp)
}

// WebsocketModifyOrder amends an order's quantity, price or trigger price over the websocket; an order fetched to
// restate values the modification leaves unset is fetched over REST
func (e *Exchange) WebsocketModifyOrder(ctx context.Context, m *order.Modify) (*order.ModifyResponse, error) {
	a, err := e.newOrderAmendment(ctx, m)
	if err != nil {
		return nil, err
	}
	resp, err := amendOrder(ctx, a, e.WebsocketAPI().AmendOrder, e.WebsocketAPI().AmendAdvancedOrder)
	if err != nil {
		return nil, err
	}
	return modifyResponse(m, a, resp)
}

// amendOrder amends an order with the advanced order methods' amend for an order they hold, and otherwise with the
// trading methods' amend, falling back to the advanced order methods' should Crypto.com refuse it, as only they amend a
// leg of an order list; an amendment giving the order a new client order ID, which they cannot, does not fall back
func amendOrder(ctx context.Context, a *orderAmendment, amend func(context.Context, *AmendOrderRequest) (*OrderIDResponse, error), amendAdvanced func(context.Context, *AmendAdvancedOrderRequest) (*OrderIDResponse, error)) (*OrderIDResponse, error) {
	if a.trigger != nil {
		return amendAdvanced(ctx, a.trigger)
	}
	resp, err := amend(ctx, a.order)
	if !refusedForOrder(err) {
		return resp, err
	}
	if a.order.ClientOrderID != "" {
		return nil, common.AppendError(err, errClientOrderIDNotAmendable)
	}
	price := a.order.NewPrice
	resp, advancedErr := amendAdvanced(ctx, &AmendAdvancedOrderRequest{
		OrderID:       a.order.OrderID,
		ClientOrderID: a.order.OriginalClientOrderID,
		NewQuantity:   a.order.NewQuantity,
		NewPrice:      &price,
	})
	if advancedErr != nil {
		return nil, common.AppendError(err, advancedErr)
	}
	return resp, nil
}

// isTriggerOrder reports whether an order type is a trigger order's, which the advanced order methods manage
func isTriggerOrder(t order.Type) bool {
	_, trigger, err := orderTypeString(t)
	return err == nil && trigger
}

// CancelOrder cancels an order by its order ID or client order ID. A trigger order is cancelled by the advanced order
// methods; an order of another type the trading methods cannot cancel, such as a leg of an OTO list, is cancelled by
// them too
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	return cancelOrder(ctx, o, e.CancelExistingOrder, e.CancelAdvancedOrder)
}

// WebsocketCancelOrder cancels an order by its order ID or client order ID over the websocket, as CancelOrder does
func (e *Exchange) WebsocketCancelOrder(ctx context.Context, o *order.Cancel) error {
	return cancelOrder(ctx, o, e.WebsocketAPI().CancelExistingOrder, e.WebsocketAPI().CancelAdvancedOrder)
}

// cancelOrder cancels an order with the advanced order methods' cancel for a trigger order, and otherwise with the
// trading methods' cancel, falling back to the advanced order methods' should Crypto.com refuse it
func cancelOrder(ctx context.Context, o *order.Cancel, cancel func(context.Context, *CancelOrderRequest) (*OrderIDResponse, error), cancelAdvanced func(context.Context, *AdvancedOrderIDRequest) (*OrderIDResponse, error)) error {
	if err := o.Validate(); err != nil {
		return err
	}
	if o.OrderID == "" && o.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	advanced := &AdvancedOrderIDRequest{OrderID: o.OrderID, ClientOrderID: o.ClientOrderID}
	if isTriggerOrder(o.Type) {
		_, err := cancelAdvanced(ctx, advanced)
		return err
	}
	_, err := cancel(ctx, &CancelOrderRequest{OrderID: o.OrderID, ClientOrderID: o.ClientOrderID})
	if !refusedForOrder(err) {
		return err
	}
	if _, advancedErr := cancelAdvanced(ctx, advanced); advancedErr != nil {
		return common.AppendError(err, advancedErr)
	}
	return nil
}

// orderCancel is an order to cancel by its order ID or client order ID, and whether the advanced order methods hold it
type orderCancel struct {
	OrderID       string
	ClientOrderID string
	Advanced      bool
}

// CancelBatchOrders cancels orders by their order IDs or client order IDs, in lists of up to 10, trigger orders, which
// must have their type given, in lists the advanced order methods cancel. A list whose request fails stops the lists
// after it, and the outcomes of those sent before it are returned with the error
func (e *Exchange) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(orders) == 0 {
		return nil, errOrderListEmpty
	}
	cancels := make([]orderCancel, len(orders))
	for i := range orders {
		if orders[i].OrderID == "" && orders[i].ClientOrderID == "" {
			return nil, fmt.Errorf("order %d: %w", i, order.ErrOrderIDNotSet)
		}
		cancels[i] = orderCancel{OrderID: orders[i].OrderID, ClientOrderID: orders[i].ClientOrderID, Advanced: isTriggerOrder(orders[i].Type)}
	}
	status, err := e.cancelOrders(ctx, cancels)
	return &order.CancelBatchResponse{Status: status}, err
}

// CancelAllOrders cancels the open orders of an asset, those of the pair given or of every pair, those the advanced
// order methods hold included. Spot and margin orders share their instruments, which a cancel-all request would cancel
// together, so the open orders are listed and those of the asset cancelled by their order IDs
func (e *Exchange) CancelAllOrders(ctx context.Context, o *order.Cancel) (order.CancelAllResponse, error) {
	if err := o.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	if !e.SupportsAsset(o.AssetType) {
		return order.CancelAllResponse{}, fmt.Errorf("%w: %s", asset.ErrNotSupported, o.AssetType)
	}
	var pairs currency.Pairs
	if !o.Pair.IsEmpty() {
		pairs = currency.Pairs{o.Pair}
	}
	open, advanced, err := e.openOrders(ctx, o.AssetType, pairs)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	cancels := make([]orderCancel, 0, len(open)+len(advanced))
	for i := range open {
		if orderAsset(open[i].InstrumentName, open[i].ExecutionInstructions) == o.AssetType {
			cancels = append(cancels, orderCancel{OrderID: open[i].OrderID})
		}
	}
	for i := range advanced {
		if orderAsset(advanced[i].InstrumentName, advanced[i].ExecutionInstructions) == o.AssetType {
			cancels = append(cancels, orderCancel{OrderID: advanced[i].OrderID, Advanced: true})
		}
	}
	if len(cancels) == 0 {
		return order.CancelAllResponse{Status: map[string]string{}}, nil
	}
	status, err := e.cancelOrders(ctx, cancels)
	return order.CancelAllResponse{Status: status}, err
}

// cancelOrders cancels orders in lists of up to 10, those the advanced order methods hold in lists they cancel, and
// returns each order's outcome by its order ID, or its client order ID without one. Only the advanced order methods
// cancel a leg of an order list, so an order the trading methods refuse to cancel is tried by them too. A list whose
// request fails stops the lists after it, and the outcomes of those sent before it are returned with the error
func (e *Exchange) cancelOrders(ctx context.Context, cancels []orderCancel) (map[string]string, error) {
	var regular, advanced []orderCancel
	for i := range cancels {
		if cancels[i].Advanced {
			advanced = append(advanced, cancels[i])
		} else {
			regular = append(regular, cancels[i])
		}
	}
	status := make(map[string]string, len(cancels))
	refused := make(map[string]string)
	for list := range slices.Chunk(regular, maxOrderListLength) {
		entries := make([]CancelOrderListEntryRequest, len(list))
		for i := range list {
			entries[i] = CancelOrderListEntryRequest{OrderID: list[i].OrderID, ClientOrderID: list[i].ClientOrderID}
		}
		results, err := e.CancelOrderList(ctx, entries)
		if err != nil {
			return status, err
		}
		for _, i := range listCancelStatus(list, results, status) {
			refused[list[i].cancelID()] = status[list[i].cancelID()]
			advanced = append(advanced, list[i])
		}
	}
	for list := range slices.Chunk(advanced, maxAdvancedOrderListLength) {
		entries := make([]AdvancedOrderIDRequest, len(list))
		for i := range list {
			entries[i] = AdvancedOrderIDRequest{OrderID: list[i].OrderID, ClientOrderID: list[i].ClientOrderID}
		}
		results, err := e.CancelAdvancedOrderList(ctx, entries)
		if err != nil {
			return status, err
		}
		listCancelStatus(list, results, status)
		// An order both refused keeps the trading methods' refusal beside the advanced order methods' outcome
		for i := range list {
			id := list[i].cancelID()
			if previous, ok := refused[id]; ok && status[id] != order.Cancelled.String() {
				status[id] = previous + "; " + status[id]
			}
		}
	}
	return status, nil
}

// cancelID returns the ID an order's cancel outcome is keyed by, its order ID, or its client order ID without one
func (c *orderCancel) cancelID() string {
	if c.OrderID != "" {
		return c.OrderID
	}
	return c.ClientOrderID
}

// listCancelStatus sets the outcome of each order of a cancelled list from the list's entries, which give each order's
// outcome by its index in the list, and returns the indexes of the orders Crypto.com refused to cancel for the order
// named, which the advanced order methods may hold
func listCancelStatus(list []orderCancel, entries []OrderListEntryResponse, status map[string]string) (refused []int) {
	for i := range list {
		status[list[i].cancelID()] = "no result for the order"
	}
	for i := range entries {
		index := entries[i].Index.Int64()
		if index < 0 || index >= int64(len(list)) {
			continue
		}
		if code := entries[i].Code.Int64(); code != 0 {
			apiErr := &APIError{Code: code, Message: entries[i].Message}
			status[list[index].cancelID()] = apiErr.Error()
			if refusedForOrder(apiErr) {
				refused = append(refused, int(index))
			}
			continue
		}
		status[list[index].cancelID()] = order.Cancelled.String()
	}
	return refused
}

// refusedForOrder reports whether Crypto.com refused a request for the order it names, which the advanced order methods
// may hold, rather than refusing the request itself, for its authentication, nonce, rate or timing or for an internal
// error, which a request to the advanced order methods would meet too
func refusedForOrder(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.Code {
	case codeUnauthorised, codeInvalidNonce, codeIPIllegal, codeUserTierInvalid, codeRequestTimeout, codeTooManyRequests, codeInternalError:
		return false
	}
	return true
}

// orderByID returns an order by its order ID or client order ID, and whether the advanced order methods hold it;
// Crypto.com keeps trigger orders and order lists apart, so an order the trading methods do not know is looked up among
// them
func (e *Exchange) orderByID(ctx context.Context, orderID, clientOrderID string) (o *OrderResponse, advanced bool, err error) {
	o, err = e.GetOrderDetail(ctx, &OrderDetailRequest{OrderID: orderID, ClientOrderID: clientOrderID})
	if err == nil {
		return o, false, nil
	}
	if !refusedForOrder(err) {
		return nil, false, err
	}
	o, advancedErr := e.GetAdvancedOrderDetail(ctx, &AdvancedOrderIDRequest{OrderID: orderID, ClientOrderID: clientOrderID})
	if advancedErr != nil {
		return nil, false, common.AppendError(err, advancedErr)
	}
	return o, true, nil
}

// GetOrderInfo returns an order of an asset by its order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, a asset.Item) (*order.Detail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	o, _, err := e.orderByID(ctx, orderID, "")
	if err != nil {
		return nil, err
	}
	if orderAsset(o.InstrumentName, o.ExecutionInstructions) != a {
		return nil, fmt.Errorf("%w: order %s is not a %s order", errOrderAssetMismatch, orderID, a)
	}
	p, err := pairFromInstrument(a, o.InstrumentName)
	if err != nil {
		return nil, err
	}
	return e.orderDetail(o, a, p)
}

// requestInstrument returns the instrument an order request for an asset's pairs names: the one pair's, or none, every
// instrument, for more pairs, whose orders are then filtered
func (e *Exchange) requestInstrument(a asset.Item, pairs currency.Pairs) (string, error) {
	if len(pairs) != 1 {
		return "", nil
	}
	return e.instrumentName(a, pairs[0])
}

// openOrders returns the open orders of an asset's pairs, or of every instrument: those the trading methods hold, and
// those the advanced order methods hold, trigger orders and order list legs
func (e *Exchange) openOrders(ctx context.Context, a asset.Item, pairs currency.Pairs) (open, advanced []OrderResponse, err error) {
	name, err := e.requestInstrument(a, pairs)
	if err != nil {
		return nil, nil, err
	}
	if open, err = e.GetOpenOrders(ctx, &OpenOrdersRequest{InstrumentName: name}); err != nil {
		return nil, nil, err
	}
	if advanced, err = e.GetAdvancedOpenOrders(ctx, &GetAdvancedOpenOrdersRequest{InstrumentName: name}); err != nil {
		return nil, nil, err
	}
	return open, advanced, nil
}

// orderDetails returns the orders of an asset as order details, leaving out other assets' orders, as a request for
// every instrument returns every asset's, and spot and margin orders share their instruments
func (e *Exchange) orderDetails(orders []OrderResponse, a asset.Item) ([]order.Detail, error) {
	details := make([]order.Detail, 0, len(orders))
	for i := range orders {
		if orderAsset(orders[i].InstrumentName, orders[i].ExecutionInstructions) != a {
			continue
		}
		p, err := pairFromInstrument(a, orders[i].InstrumentName)
		if err != nil {
			return nil, err
		}
		d, err := e.orderDetail(&orders[i], a, p)
		if err != nil {
			return nil, err
		}
		details = append(details, *d)
	}
	return details, nil
}

// GetActiveOrders returns the open orders of an asset, those the advanced order methods hold included
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if !e.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	open, advanced, err := e.openOrders(ctx, req.AssetType, req.Pairs)
	if err != nil {
		return nil, err
	}
	details, err := e.orderDetails(append(open, advanced...), req.AssetType)
	if err != nil {
		return nil, err
	}
	return req.Filter(e.Name, details), nil
}

// GetOrderHistory returns the past orders of an asset, trigger orders included, created from the start time to the end
// time; without a start time the day before the end time, the window Crypto.com defaults to
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if !e.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	name, err := e.requestInstrument(req.AssetType, req.Pairs)
	if err != nil {
		return nil, err
	}
	orders, err := e.orderHistory(ctx, name, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	details, err := e.orderDetails(orders, req.AssetType)
	if err != nil {
		return nil, err
	}
	return req.Filter(e.Name, details), nil
}

// orderHistory returns an instrument's past orders, or every instrument's, trigger orders included, created from the
// start time, by default the day before the end time, to the end time, by default the current time
func (e *Exchange) orderHistory(ctx context.Context, instrumentName string, start, end time.Time) ([]OrderResponse, error) {
	if end.IsZero() {
		end = time.Now()
	}
	if start.IsZero() {
		start = end.Add(-privateHistoryWindow)
	}
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	regular := historyPager[OrderResponse]{
		exchangeName: e.Name,
		limit:        maxTradingHistoryLimit,
		fetch: func(ctx context.Context, start, end time.Time) ([]OrderResponse, error) {
			return e.GetUserOrderHistory(ctx, &UserOrderHistoryRequest{InstrumentName: instrumentName, StartTime: start, EndTime: end, Limit: maxTradingHistoryLimit})
		},
		record: orderRecord,
	}
	orders, err := regular.page(ctx, start, end, privateHistoryWindow)
	if err != nil {
		return nil, err
	}
	advanced := historyPager[OrderResponse]{
		exchangeName: e.Name,
		limit:        maxAdvancedOrderHistoryLimit,
		fetch: func(ctx context.Context, start, end time.Time) ([]OrderResponse, error) {
			return e.GetAdvancedOrderHistory(ctx, &GetAdvancedOrderHistoryRequest{InstrumentName: instrumentName, StartTime: start, EndTime: end, Limit: maxAdvancedOrderHistoryLimit})
		},
		record: orderRecord,
	}
	triggers, err := advanced.page(ctx, start, end, privateHistoryWindow)
	if err != nil {
		return nil, err
	}
	return append(orders, triggers...), nil
}

// orderRecord returns an order's ID and its creation time, which order history pages by
func orderRecord(o *OrderResponse) (id string, at time.Time) {
	if t := o.CreateTimeNano.Time(); !t.IsZero() {
		return o.OrderID, t
	}
	return o.OrderID, o.CreateTime.Time()
}

// orderSideString returns the Crypto.com side of an order side; a long position opens with a buy and a short with a
// sell
func orderSideString(s order.Side) (string, error) {
	switch s {
	case order.Buy, order.Bid, order.Long:
		return SideBuy, nil
	case order.Sell, order.Ask, order.Short:
		return SideSell, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrSideIsInvalid, s)
	}
}

// orderSideFromString returns the order side of a Crypto.com side, in either case, as REST trades send it in lower case
func orderSideFromString(s string) (order.Side, error) {
	switch strings.ToUpper(s) {
	case SideBuy:
		return order.Buy, nil
	case SideSell:
		return order.Sell, nil
	default:
		return order.UnknownSide, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, s)
	}
}

// orderTypeString returns the Crypto.com order type of an order type, and whether it is a trigger order, which the
// advanced order methods place. A stop or a take profit without a limit triggers a market order
func orderTypeString(t order.Type) (orderType string, trigger bool, err error) {
	switch t {
	case order.Limit:
		return OrderTypeLimit, false, nil
	case order.Market:
		return OrderTypeMarket, false, nil
	case order.Stop, order.StopMarket:
		return OrderTypeStopLoss, true, nil
	case order.StopLimit:
		return OrderTypeStopLimit, true, nil
	case order.TakeProfit, order.TakeProfitMarket:
		return OrderTypeTakeProfit, true, nil
	case order.TakeProfitLimit:
		return OrderTypeTakeProfitLimit, true, nil
	default:
		return "", false, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
	}
}

// orderTypeFromString returns the order type of a Crypto.com order type
func orderTypeFromString(s string) (order.Type, error) {
	switch s {
	case OrderTypeLimit:
		return order.Limit, nil
	case OrderTypeMarket:
		return order.Market, nil
	case OrderTypeStopLoss:
		return order.Stop, nil
	case OrderTypeStopLimit:
		return order.StopLimit, nil
	case OrderTypeTakeProfit:
		return order.TakeProfit, nil
	case OrderTypeTakeProfitLimit:
		return order.TakeProfitLimit, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %q", order.ErrUnrecognisedOrderType, s)
	}
}

// timeInForceParams returns the Crypto.com time in force of a time in force, and whether the order is post only, which
// Crypto.com takes as an execution instruction and only with good till cancel; an unset time in force is left to
// Crypto.com's default, good till cancel
func timeInForceParams(tif order.TimeInForce) (timeInForce string, postOnly bool, err error) {
	if !tif.IsValid() {
		return "", false, fmt.Errorf("%w: %s", order.ErrInvalidTimeInForce, tif)
	}
	postOnly = tif.Is(order.PostOnly)
	switch tif &^ order.PostOnly {
	case order.UnknownTIF:
		if postOnly {
			return TimeInForceGoodTillCancel, true, nil
		}
		return "", false, nil
	case order.GoodTillCancel:
		return TimeInForceGoodTillCancel, postOnly, nil
	case order.ImmediateOrCancel:
		// IsValid refuses either with another flag, so neither is post only
		return TimeInForceImmediateOrCancel, false, nil
	case order.FillOrKill:
		return TimeInForceFillOrKill, false, nil
	}
	return "", false, fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, tif)
}

// effectiveTimeInForce returns the time in force Crypto.com gives an order placed with a time in force: good till
// cancelled, its default, when it names none besides post only
func effectiveTimeInForce(tif order.TimeInForce) order.TimeInForce {
	if tif&^order.PostOnly == order.UnknownTIF {
		return tif | order.GoodTillCancel
	}
	return tif
}

// timeInForceFromResponse returns the time in force of a Crypto.com order, post only when its execution instructions
// hold either post only instruction
func timeInForceFromResponse(timeInForce string, executionInstructions []string) (order.TimeInForce, error) {
	var tif order.TimeInForce
	switch timeInForce {
	case TimeInForceGoodTillCancel:
		tif = order.GoodTillCancel
	case TimeInForceImmediateOrCancel:
		tif = order.ImmediateOrCancel
	case TimeInForceFillOrKill:
		tif = order.FillOrKill
	case "":
	default:
		return order.UnknownTIF, fmt.Errorf("%w: %q", order.ErrInvalidTimeInForce, timeInForce)
	}
	if slices.Contains(executionInstructions, ExecutionInstructionPostOnly) || slices.Contains(executionInstructions, ExecutionInstructionSmartPostOnly) {
		tif |= order.PostOnly
	}
	return tif, nil
}

// orderStatus returns the status of a Crypto.com order: an active order that has executed some of its quantity is
// partially filled, and a cancelled one partially cancelled
func orderStatus(o *OrderResponse) (order.Status, error) {
	switch o.Status {
	case OrderStatusPending:
		return order.Pending, nil
	case OrderStatusNew:
		return order.New, nil
	case OrderStatusActive:
		if o.CumulativeQuantity.Float64() > 0 {
			return order.PartiallyFilled, nil
		}
		return order.Active, nil
	case OrderStatusFilled:
		return order.Filled, nil
	case OrderStatusCanceled:
		if o.CumulativeQuantity.Float64() > 0 {
			return order.PartiallyCancelled, nil
		}
		return order.Cancelled, nil
	case OrderStatusRejected:
		return order.Rejected, nil
	case OrderStatusExpired:
		return order.Expired, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %q", order.ErrUnsupportedStatusType, o.Status)
	}
}

// orderDetail returns a Crypto.com order as an order detail of an asset and pair. The amount is the order's quantity,
// so a market buy placed by its notional, which carries no quantity, has its quantity unset and its notional as the
// quote amount. Margin orders are cross margined, and derivatives margined and settled in USD, cross margined unless
// isolated
func (e *Exchange) orderDetail(o *OrderResponse, a asset.Item, p currency.Pair) (*order.Detail, error) {
	side, err := orderSideFromString(o.Side)
	if err != nil {
		return nil, err
	}
	orderType, err := orderTypeFromString(o.OrderType)
	if err != nil {
		return nil, err
	}
	status, err := orderStatus(o)
	if err != nil {
		return nil, err
	}
	tif, err := timeInForceFromResponse(o.TimeInForce, o.ExecutionInstructions)
	if err != nil {
		return nil, err
	}
	d := &order.Detail{
		Exchange:             e.Name,
		OrderID:              o.OrderID,
		ClientOrderID:        o.ClientOrderID,
		AccountID:            o.AccountID,
		Pair:                 p,
		AssetType:            a,
		Side:                 side,
		Type:                 orderType,
		Status:               status,
		TimeInForce:          tif,
		ReduceOnly:           slices.Contains(o.ExecutionInstructions, ExecutionInstructionReduceOnly),
		Price:                o.LimitPrice.Float64(),
		Amount:               o.Quantity.Float64(),
		TriggerPrice:         o.ReferencePrice.Float64(),
		AverageExecutedPrice: o.AveragePrice.Float64(),
		ExecutedAmount:       o.CumulativeQuantity.Float64(),
		Cost:                 o.CumulativeValue.Float64(),
		CostAsset:            p.Quote,
		Fee:                  o.CumulativeFee.Float64(),
		FeeAsset:             o.FeeInstrumentName,
		Leverage:             o.Leverage.Float64(),
		Date:                 o.CreateTime.Time(),
		LastUpdated:          o.UpdateTime.Time(),
	}
	// The nanosecond creation time is the more precise, where Crypto.com sends it
	if t := o.CreateTimeNano.Time(); !t.IsZero() {
		d.Date = t
	}
	switch {
	case a.IsFutures():
		d.CostAsset = currency.USD
		d.SettlementCurrency = currency.USD
		d.MarginType = margin.Multi
		if isIsolated(o.IsolationID) {
			d.MarginType = margin.Isolated
		}
	case a == asset.Margin:
		d.MarginType = margin.Multi
	}
	if d.Amount > 0 {
		d.RemainingAmount = max(d.Amount-d.ExecutedAmount, 0)
	} else if d.Side == order.Buy && (d.Type == order.Market || d.Type == order.Stop || d.Type == order.TakeProfit) {
		// A stop loss or take profit buy is a market buy once triggered, so it can spend a notional as a market buy does
		d.QuoteAmount = o.OrderValue.Float64()
	}
	if status.IsInactive() {
		d.CloseTime = d.LastUpdated
	}
	return d, nil
}

// GetCurrencyTradeURL returns the URL of a pair's trading page; Crypto.com redirects US visitors from a derivative's
// page to spot, so derivatives' pages are named as CoinGecko's derivatives listings link them
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, p currency.Pair) (string, error) {
	if !e.SupportsAsset(a) {
		return "", fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	name, err := e.instrumentName(a, p)
	if err != nil {
		return "", err
	}
	return tradeBaseURL + name, nil
}

// GetFuturesContractDetails returns the contracts of an asset's perpetual swaps or futures, which Crypto.com margins
// and settles in USD
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, a asset.Item) ([]futures.Contract, error) {
	if !a.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	instruments, err := e.GetInstruments(ctx)
	if err != nil {
		return nil, err
	}
	contracts := make([]futures.Contract, 0, len(instruments))
	for i := range instruments {
		inst := &instruments[i]
		if !instrumentIsAsset(inst, a) {
			continue
		}
		p, err := pairFromInstrument(a, inst.Symbol)
		if err != nil {
			return nil, err
		}
		c := futures.Contract{
			Exchange:           e.Name,
			Name:               p,
			Underlying:         currency.NewPair(inst.BaseCurrency, inst.QuoteCurrency),
			Asset:              a,
			IsActive:           inst.Tradable,
			Type:               futures.Perpetual,
			SettlementType:     futures.Linear,
			SettlementCurrency: currency.USD,
			MarginCurrency:     currency.USD,
			Multiplier:         inst.ContractSize.Float64(),
			MaxLeverage:        inst.MaxLeverage.Float64(),
		}
		if a == asset.Futures {
			c.Type = futures.LongDated
			c.EndDate = inst.ExpiryTime.Time()
		}
		contracts = append(contracts, c)
	}
	return contracts, nil
}

// IsPerpetualFutureCurrency reports whether an asset's pairs are perpetual swaps
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.PerpetualSwap, nil
}

// GetCollateralCurrencyForContract returns the currency derivatives are margined in, USD
func (e *Exchange) GetCollateralCurrencyForContract(a asset.Item, _ currency.Pair) (currency.Code, asset.Item, error) {
	if !a.IsFutures() || !e.SupportsAsset(a) {
		return currency.EMPTYCODE, asset.Empty, fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, a)
	}
	return currency.USD, a, nil
}

// GetCollateralMode returns the collateral mode of margin and derivatives: Crypto.com's smart cross margin values every
// collateral eligible currency, after its haircut
func (e *Exchange) GetCollateralMode(_ context.Context, a asset.Item) (collateral.Mode, error) {
	if (a != asset.Margin && !a.IsFutures()) || !e.SupportsAsset(a) {
		return collateral.UnknownMode, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	return collateral.MultiMode, nil
}

// GetLatestFundingRates returns the funding rate of a perpetual swap, or of every enabled one, which settles hourly at
// the end of the hour, and with IncludePredictedRate the rate estimated for the next funding interval
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if r.Asset != asset.PerpetualSwap {
		return nil, fmt.Errorf("%w: %s", futures.ErrNotPerpetualFuture, r.Asset)
	}
	pairs := currency.Pairs{r.Pair}
	if r.Pair.IsEmpty() {
		var err error
		if pairs, err = e.GetEnabledPairs(r.Asset); err != nil {
			return nil, err
		}
	}
	resp := make([]fundingrate.LatestRateResponse, 0, len(pairs))
	for _, p := range pairs {
		name, err := e.instrumentName(r.Asset, p)
		if err != nil {
			return nil, err
		}
		latest, err := e.latestValuation(ctx, name, ValuationTypeFundingRate)
		if err != nil {
			return nil, err
		}
		rate := fundingrate.LatestRateResponse{
			Exchange:       e.Name,
			Asset:          r.Asset,
			Pair:           p,
			LatestRate:     fundingrate.Rate{Time: latest.Timestamp.Time(), Rate: latest.Value.Decimal()},
			TimeOfNextRate: latest.Timestamp.Time().Truncate(fundingInterval).Add(fundingInterval),
			TimeChecked:    time.Now(),
		}
		if r.IncludePredictedRate {
			predicted, err := e.latestValuation(ctx, name, ValuationTypeEstimatedFundingRate)
			if err != nil {
				return nil, err
			}
			rate.PredictedUpcomingRate = fundingrate.Rate{Time: predicted.Timestamp.Time(), Rate: predicted.Value.Decimal()}
		}
		resp = append(resp, rate)
	}
	return resp, nil
}

// latestValuation returns an instrument's latest value of a valuation type
func (e *Exchange) latestValuation(ctx context.Context, instrumentName, valuationType string) (*Valuation, error) {
	resp, err := e.GetValuations(ctx, &ValuationsRequest{InstrumentName: instrumentName, ValuationType: valuationType, Count: 1})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("%w: no %s for %s", common.ErrNoResponse, valuationType, instrumentName)
	}
	return &resp.Data[0], nil
}

// GetHistoricalFundingRates returns the hourly funding rates a perpetual swap settled from the start date, inclusive, to
// the end date, exclusive. Crypto.com serves the 14 days before a request's end, and can add the rate of the hour before
// its start, so the window is requested 14 days at a time from its end and rates outside each request's window dropped.
// With IncludePayments the account's funding payments over the window are summed from its funding transactions
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if r.Asset != asset.PerpetualSwap {
		return nil, fmt.Errorf("%w: %s", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	name, err := e.instrumentName(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	// No rate settles after the current time, so a window ending later is requested to it
	end := r.EndDate
	if now := time.Now(); end.After(now) {
		end = now
	}
	var rates []fundingrate.Rate
	for ; end.After(r.StartDate); end = end.Add(-fundingHistoryWindow) {
		start := end.Add(-fundingHistoryWindow)
		if start.Before(r.StartDate) {
			start = r.StartDate
		}
		resp, err := e.GetValuations(ctx, &ValuationsRequest{InstrumentName: name, ValuationType: ValuationTypeFundingHistory, Count: fundingHistoryPageLimit, StartTime: start, EndTime: end})
		if err != nil {
			return nil, err
		}
		for i := range resp.Data {
			if at := resp.Data[i].Timestamp.Time(); !at.Before(start) && at.Before(end) {
				rates = append(rates, fundingrate.Rate{Time: at, Rate: resp.Data[i].Value.Decimal()})
			}
		}
	}
	slices.SortFunc(rates, func(x, y fundingrate.Rate) int { return x.Time.Compare(y.Time) })
	resp := &fundingrate.HistoricalRates{
		Exchange:     e.Name,
		Asset:        r.Asset,
		Pair:         r.Pair,
		StartDate:    r.StartDate,
		EndDate:      r.EndDate,
		FundingRates: rates,
	}
	if r.IncludePredictedRate {
		predicted, err := e.latestValuation(ctx, name, ValuationTypeEstimatedFundingRate)
		if err != nil {
			return nil, err
		}
		resp.PredictedUpcomingRate = fundingrate.Rate{Time: predicted.Timestamp.Time(), Rate: predicted.Value.Decimal()}
	}
	if r.IncludePayments {
		if err := e.fundingPayments(ctx, name, resp); err != nil {
			return nil, err
		}
	}
	if len(rates) > 0 {
		resp.LatestRate = rates[len(rates)-1]
		resp.TimeOfNextRate = resp.LatestRate.Time.Add(fundingInterval)
	}
	return resp, nil
}

// fundingPayments sums the account's funding payments over a funding history's window from its funding transactions,
// and adds each to the rate of its hour
func (e *Exchange) fundingPayments(ctx context.Context, instrumentName string, resp *fundingrate.HistoricalRates) error {
	pager := historyPager[TransactionResponse]{
		exchangeName: e.Name,
		limit:        maxTradingHistoryLimit,
		fetch: func(ctx context.Context, start, end time.Time) ([]TransactionResponse, error) {
			return e.GetTransactions(ctx, &TransactionsRequest{InstrumentName: instrumentName, JournalType: JournalTypeFunding, StartTime: start, EndTime: end, Limit: maxTradingHistoryLimit})
		},
		record: func(t *TransactionResponse) (string, time.Time) { return string(t.JournalID), transactionTime(t) },
	}
	payments, err := pager.page(ctx, resp.StartDate, resp.EndDate, privateHistoryWindow)
	if err != nil {
		return err
	}
	resp.PaymentCurrency = currency.USD
	for i := range payments {
		payment := payments[i].TransactionQuantity.Decimal()
		resp.PaymentSum = resp.PaymentSum.Add(payment)
		// A funding payment is made as its hour's rate settles, so it is the payment of the rate of that hour
		settled := transactionTime(&payments[i]).Truncate(fundingInterval)
		if j := slices.IndexFunc(resp.FundingRates, func(rate fundingrate.Rate) bool { return rate.Time.Equal(settled) }); j >= 0 {
			resp.FundingRates[j].Payment = resp.FundingRates[j].Payment.Add(payment)
		}
	}
	return nil
}

// transactionTime returns a transaction's time, to the nanosecond where Crypto.com sends it
func transactionTime(t *TransactionResponse) time.Time {
	if ns := t.EventTimeNano.Time(); !ns.IsZero() {
		return ns
	}
	return t.EventTime.Time()
}

// GetOpenInterest returns the open interest of derivatives pairs, or of every enabled derivatives pair, from one request
// for every instrument's ticker
func (e *Exchange) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range keys {
		if !keys[i].Asset.IsFutures() || !e.SupportsAsset(keys[i].Asset) {
			return nil, fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, keys[i].Asset)
		}
	}
	if len(keys) == 0 {
		for _, a := range e.GetAssetTypes(true) {
			if !a.IsFutures() {
				continue
			}
			pairs, err := e.GetEnabledPairs(a)
			if err != nil {
				return nil, err
			}
			for _, p := range pairs {
				keys = append(keys, key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a})
			}
		}
	}
	byInstrument := make(map[string]key.PairAsset, len(keys))
	for i := range keys {
		name, err := e.instrumentName(keys[i].Asset, keys[i].Pair())
		if err != nil {
			return nil, err
		}
		byInstrument[name] = keys[i]
	}
	tickers, err := e.GetTickers(ctx, &TickersRequest{})
	if err != nil {
		return nil, err
	}
	resp := make([]futures.OpenInterest, 0, len(keys))
	for i := range tickers {
		k, ok := byInstrument[tickers[i].InstrumentName]
		if !ok {
			continue
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.ExchangeAssetPair{Exchange: e.Name, Base: k.Base, Quote: k.Quote, Asset: k.Asset},
			OpenInterest: tickers[i].OpenInterest.Float64(),
		})
	}
	return resp, nil
}

// GetFuturesPositionSummary returns a derivative's position, its cross margin position if it has one, with the
// account's margin, which every cross margin position shares, or else its isolated position, with that position's.
// get-positions leaves out a position's margins, prices and leverage, so the mark price is the instrument's, the
// leverage the isolated position's or else the account's, and the margin requirements an isolated position's own
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, r *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if !r.Asset.IsFutures() || !e.SupportsAsset(r.Asset) {
		return nil, fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, r.Asset)
	}
	name, err := e.instrumentName(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	positions, err := e.GetPositions(ctx, &PositionsRequest{InstrumentName: name})
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(positions, func(p PositionResponse) bool { return p.InstrumentName == name && !isIsolated(p.IsolationID) })
	if i < 0 {
		i = slices.IndexFunc(positions, func(p PositionResponse) bool { return p.InstrumentName == name })
	}
	if i < 0 {
		return nil, fmt.Errorf("%w: %s %s", futures.ErrNoPositionsFound, r.Asset, r.Pair)
	}
	pos := &positions[i]
	balances, err := e.GetUserBalance(ctx)
	if err != nil {
		return nil, err
	}
	mark, err := e.latestValuation(ctx, name, ValuationTypeMarkPrice)
	if err != nil {
		return nil, err
	}
	size := pos.Quantity.Decimal()
	summary := &futures.PositionSummary{
		Pair:                   r.Pair,
		Asset:                  r.Asset,
		MarginType:             margin.Multi,
		CollateralMode:         collateral.MultiMode,
		Currency:               currency.USD,
		MarkPrice:              mark.Value.Decimal(),
		NotionalSize:           size.Abs().Mul(mark.Value.Decimal()),
		CurrentSize:            size,
		ContractSettlementType: futures.Linear,
		UnrealisedPNL:          pos.OpenPositionPNL.Decimal(),
	}
	if !size.IsZero() {
		summary.AverageOpenPrice = pos.OpenPositionCost.Decimal().Div(size).Abs()
	}
	var account *BalanceSummary
	if isIsolated(pos.IsolationID) {
		summary.MarginType = margin.Isolated
		iso := findIsolatedPosition(balances, func(p *IsolatedPositionBalance) bool { return p.IsolationID == pos.IsolationID })
		if iso == nil {
			return nil, fmt.Errorf("%w: %s %s", errNoIsolatedPosition, name, pos.IsolationID)
		}
		account = &iso.BalanceSummary
		summary.Leverage = iso.Leverage.Decimal()
		summary.IsolatedMargin = iso.TotalMarginBalance.Decimal()
		summary.InitialMarginRequirement = iso.TotalInitialMargin.Decimal()
		summary.MaintenanceMarginRequirement = iso.TotalMaintenanceMargin.Decimal()
	} else {
		if len(balances) > 0 {
			account = &balances[0].BalanceSummary
		}
		settings, err := e.GetAccountSettings(ctx)
		if err != nil {
			return nil, err
		}
		if len(settings) > 0 {
			summary.Leverage = settings[0].Leverage.Decimal()
		}
	}
	if account != nil {
		summary.AvailableEquity = account.TotalAvailableBalance.Decimal()
		summary.CashBalance = account.TotalCashBalance.Decimal()
		summary.MarginBalance = account.TotalMarginBalance.Decimal()
		summary.NotionalLeverage = account.TotalEffectiveLeverage.Decimal()
		summary.TotalCollateral = account.TotalCollateralValue.Decimal()
		summary.FreeCollateral = account.TotalAvailableBalance.Decimal()
	}
	return summary, nil
}

// GetFuturesPositionOrders returns the orders of derivatives pairs created from the start date to the end date; without
// a start date the day before the end date, the window Crypto.com defaults to
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, r *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if !r.Asset.IsFutures() || !e.SupportsAsset(r.Asset) {
		return nil, fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, r.Asset)
	}
	if len(r.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	resp := make([]futures.PositionResponse, len(r.Pairs))
	for i, p := range r.Pairs {
		name, err := e.instrumentName(r.Asset, p)
		if err != nil {
			return nil, err
		}
		orders, err := e.orderHistory(ctx, name, r.StartDate, r.EndDate)
		if err != nil {
			return nil, err
		}
		details, err := e.orderDetails(orders, r.Asset)
		if err != nil {
			return nil, err
		}
		resp[i] = futures.PositionResponse{Pair: p, Asset: r.Asset, ContractSettlementType: futures.Linear, Orders: details}
	}
	return resp, nil
}

// isIsolated reports whether an isolation ID names an isolated position; Crypto.com sends 0 or nothing for the cross
// margin one
func isIsolated(isolationID string) bool {
	return isolationID != "" && isolationID != "0"
}

// findIsolatedPosition returns the first isolated position of the balances that match reports, or nil
func findIsolatedPosition(balances []UserBalanceResponse, match func(*IsolatedPositionBalance) bool) *IsolatedPositionBalance {
	for i := range balances {
		for j := range balances[i].IsolatedPositions {
			if match(&balances[i].IsolatedPositions[j]) {
				return &balances[i].IsolatedPositions[j]
			}
		}
	}
	return nil
}

// isolatedPositionID returns the isolation ID of the account's isolated position on an instrument, or nothing when it
// holds none
func (e *Exchange) isolatedPositionID(ctx context.Context, instrumentName string) (string, error) {
	positions, err := e.GetPositions(ctx, &PositionsRequest{InstrumentName: instrumentName})
	if err != nil {
		return "", err
	}
	if i := slices.IndexFunc(positions, func(p PositionResponse) bool { return p.InstrumentName == instrumentName && isIsolated(p.IsolationID) }); i >= 0 {
		return positions[i].IsolationID, nil
	}
	return "", nil
}

// isolatedPosition returns the account's isolated position on an instrument; an isolated position's balance names its
// currency, such as USD, rather than its instrument, so it is found by the isolation ID of the instrument's position
func (e *Exchange) isolatedPosition(ctx context.Context, instrumentName string) (*IsolatedPositionBalance, error) {
	id, err := e.isolatedPositionID(ctx, instrumentName)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("%w: %s", errNoIsolatedPosition, instrumentName)
	}
	balances, err := e.GetUserBalance(ctx)
	if err != nil {
		return nil, err
	}
	if iso := findIsolatedPosition(balances, func(p *IsolatedPositionBalance) bool { return p.IsolationID == id }); iso != nil {
		return iso, nil
	}
	return nil, fmt.Errorf("%w: %s %s", errNoIsolatedPosition, instrumentName, id)
}

// GetLeverage returns the leverage of a derivative's isolated position, or the account's leverage, which its cross
// margin positions share
func (e *Exchange) GetLeverage(ctx context.Context, a asset.Item, p currency.Pair, marginType margin.Type, _ order.Side) (float64, error) {
	if !a.IsFutures() || !e.SupportsAsset(a) {
		return 0, fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, a)
	}
	if marginType == margin.Isolated {
		name, err := e.instrumentName(a, p)
		if err != nil {
			return 0, err
		}
		pos, err := e.isolatedPosition(ctx, name)
		if err != nil {
			return 0, err
		}
		return pos.Leverage.Float64(), nil
	}
	settings, err := e.GetAccountSettings(ctx)
	if err != nil {
		return 0, err
	}
	if len(settings) == 0 {
		return 0, common.ErrNoResponse
	}
	return settings[0].Leverage.Float64(), nil
}

// SetLeverage sets the leverage of a derivative's isolated position, or the account's leverage, a whole number up to
// 100, which its cross margin positions share
func (e *Exchange) SetLeverage(ctx context.Context, a asset.Item, p currency.Pair, marginType margin.Type, amount float64, _ order.Side) error {
	if !a.IsFutures() || !e.SupportsAsset(a) {
		return fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, a)
	}
	if marginType == margin.Isolated {
		name, err := e.instrumentName(a, p)
		if err != nil {
			return err
		}
		pos, err := e.isolatedPosition(ctx, name)
		if err != nil {
			return err
		}
		return e.ChangeIsolatedMarginLeverage(ctx, &ChangeIsolatedMarginLeverageRequest{IsolationID: pos.IsolationID, Leverage: types.Number(amount)})
	}
	if amount < 0 || amount != math.Trunc(amount) {
		return fmt.Errorf("%w: %v", errLeverageNotWhole, amount)
	}
	accts, err := e.GetAccounts(ctx, &AccountsRequest{})
	if err != nil {
		return err
	}
	return e.ChangeAccountLeverage(ctx, &ChangeAccountLeverageRequest{AccountID: accts.ThisAccount.UUID, Leverage: uint64(amount)})
}

// ChangePositionMargin moves margin between the account and a derivative's isolated position, to give the position the
// new allocated margin
func (e *Exchange) ChangePositionMargin(ctx context.Context, r *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if !r.Asset.IsFutures() || !e.SupportsAsset(r.Asset) {
		return nil, fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, r.Asset)
	}
	if r.MarginType != margin.Isolated {
		return nil, fmt.Errorf("%w: %s", margin.ErrMarginTypeUnsupported, r.MarginType)
	}
	if r.NewAllocatedMargin == 0 {
		return nil, margin.ErrNewAllocatedMarginRequired
	}
	if r.OriginalAllocatedMargin == 0 {
		return nil, margin.ErrOriginalPositionMarginRequired
	}
	// gctcli's float flags parse NaN and infinities, which have no decimal to take the difference in
	for _, m := range []float64{r.OriginalAllocatedMargin, r.NewAllocatedMargin} {
		if math.IsNaN(m) || math.IsInf(m, 0) {
			return nil, fmt.Errorf("%w: %v", errMarginNotFinite, m)
		}
	}
	name, err := e.instrumentName(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	pos, err := e.isolatedPosition(ctx, name)
	if err != nil {
		return nil, err
	}
	// The difference is taken in decimal arithmetic, as 8 less 5.4 in floats is 2.6000000000000005
	direction, amount := IsolatedMarginTransferDirectionCredit, decimal.MustFromFloat(r.NewAllocatedMargin).Sub(decimal.MustFromFloat(r.OriginalAllocatedMargin))
	if amount.IsNegative() {
		direction, amount = IsolatedMarginTransferDirectionDebit, amount.Neg()
	}
	if err := e.CreateIsolatedMarginTransfer(ctx, &CreateIsolatedMarginTransferRequest{IsolationID: pos.IsolationID, Direction: direction, Amount: types.Number(amount.InexactFloat64())}); err != nil {
		return nil, err
	}
	return &margin.PositionChangeResponse{
		Exchange:        e.Name,
		Pair:            r.Pair,
		Asset:           r.Asset,
		AllocatedMargin: r.NewAllocatedMargin,
		MarginType:      margin.Isolated,
	}, nil
}
