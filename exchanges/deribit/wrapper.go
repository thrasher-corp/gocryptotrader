package deribit

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
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
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var (
	errUnsupportedOrderType   = errors.New("unsupported order type")
	errUnsupportedOrderStatus = errors.New("unsupported order status")
	errTrailingStopDistance   = errors.New("a trailing stop needs a distance tracking value")
	errBookSummaryNotFound    = errors.New("book summary not found")
	errOrderLabelNotUnique    = errors.New("label does not identify one open order")
	errCandleFieldsMismatch   = errors.New("candle fields differ in length")
	errTradePagingStalled     = errors.New("trades sharing one timestamp fill a whole page")
)

const (
	tradeBaseURL      = "https://www.deribit.com/"
	tradeSpot         = "spot/"
	tradeFutures      = "futures/"
	tradeOptions      = "options/"
	tradeFuturesCombo = "futures-spreads/"
	tradeOptionsCombo = "combos/"

	perpString = "PERPETUAL"

	// orderBookDepth is the number of levels UpdateOrderbook requests on each side
	orderBookDepth = 1000
	// tradesPageSize is the most trades, and maximumFundingRatePage the most hourly funding records, a page returns
	tradesPageSize         = 1000
	maximumFundingRatePage = 744
	// latestFundingRateWindow is how far back GetLatestFundingRates looks for the latest hourly funding record
	latestFundingRateWindow = 16 * time.Hour
)

// anyCurrency selects every currency where a method accepts it; Deribit rejects it upper case
var anyCurrency = currency.ANY.Lower()

// SetDefaults sets the basic defaults for Deribit
func (e *Exchange) SetDefaults() {
	e.Name = "Deribit"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	dashFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	underscoreFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}
	if err := e.SetAssetPairStore(asset.Spot, currency.PairStore{AssetEnabled: true, RequestFormat: underscoreFormat, ConfigFormat: underscoreFormat}); err != nil {
		log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, asset.Spot, err)
	}
	for _, a := range []asset.Item{asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo} {
		if err := e.SetAssetPairStore(a, currency.PairStore{AssetEnabled: true, RequestFormat: dashFormat, ConfigFormat: dashFormat}); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	// Fill out the capabilities/features that the exchange supports
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
				GetOrder:              true,
				GetOrders:             true,
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				ModifyOrder:           true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				Positions:    true,
				Leverage:     true,
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.OneHour:   true,
					kline.EightHour: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported: true,
				},
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.TenMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					// NOTE: The supported time intervals below are returned
					// offset to +8 hours. This may lead to
					// issues with candle quality and conversion as the
					// intervals may be broken up. The below intervals
					// are therefore constructed from the intervals above.
					// kline.IntervalCapacity{Interval: kline.ThreeHour},
					// kline.IntervalCapacity{Interval: kline.SixHour},
					// kline.IntervalCapacity{Interval: kline.TwelveHour},
					// kline.IntervalCapacity{Interval: kline.OneDay},
				),
				GlobalResultLimit: 500,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimits()),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	for _, assetType := range []asset.Item{asset.Options, asset.OptionCombo, asset.FutureCombo} {
		if err = e.DisableAssetWebsocketSupport(assetType); err != nil {
			log.Errorln(log.ExchangeSys, err)
		}
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestFutures:           apiURL,
		exchange.RestSpot:              apiURL,
		exchange.RestSpotSupplementary: testnetAPIURL,
		exchange.WebsocketSpot:         websocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup takes in the supplied exchange configuration details and sets params
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
	wsURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            websocketURL,
		RunningURL:            wsURL,
		Connector:             e.WebsocketConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  e.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// websocketForWrapper reports whether a wrapper method sends its request over the websocket: a public request whenever
// the websocket is connected, and a private request only when the configuration also lets the wrapper use the
// authenticated websocket
func (e *Exchange) websocketForWrapper(private bool) bool {
	if !e.Websocket.IsConnected() {
		return false
	}
	return !private || e.Websocket.CanUseAuthenticatedWebsocketForWrapper()
}

// pairInstrument returns the instrument name of a pair
func (e *Exchange) pairInstrument(p currency.Pair, a asset.Item) (string, error) {
	if p.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	formatted, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return "", err
	}
	return formatPairString(a, formatted), nil
}

// assetKind returns the instrument kind of a supported asset
func (e *Exchange) assetKind(a asset.Item) (string, error) {
	if !e.SupportsAsset(a) {
		return "", fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	return e.GetAssetKind(a), nil
}

// FetchTradablePairs returns the active instruments of an asset as pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	kind, err := e.assetKind(a)
	if err != nil {
		return nil, err
	}
	instruments, err := e.getInstruments(ctx, &InstrumentsRequest{Currency: anyCurrency, Kind: kind}, e.websocketForWrapper(false))
	if err != nil {
		return nil, err
	}
	return e.tradablePairs(a, instruments)
}

// UpdateTradablePairs updates the pairs of every enabled asset from one request for every instrument
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	instruments, err := e.getInstruments(ctx, &InstrumentsRequest{Currency: anyCurrency}, e.websocketForWrapper(false))
	if err != nil {
		return err
	}
	byAsset := make(map[asset.Item][]Instrument)
	for i := range instruments {
		// A kind Deribit adds later is left out until it is supported
		if a, err := e.StringToAssetKind(instruments[i].Kind); err == nil {
			byAsset[a] = append(byAsset[a], instruments[i])
		}
	}
	var errs error
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.tradablePairs(a, byAsset[a])
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		errs = common.AppendError(errs, e.UpdatePairs(pairs, a, false))
	}
	return errs
}

// tradablePairs returns the active instruments as pairs; for spot it also remembers which instruments Deribit routes to
// Coinbase Exchange, whose trades and candles it does not stream
func (e *Exchange) tradablePairs(a asset.Item, instruments []Instrument) (currency.Pairs, error) {
	pairs := make(currency.Pairs, 0, len(instruments))
	routed := make(map[string]struct{})
	for i := range instruments {
		if !instruments[i].IsActive {
			continue
		}
		pair, err := currency.NewPairFromString(instruments[i].InstrumentName)
		if err != nil {
			return nil, err
		}
		pairs = pairs.Add(pair)
		if instruments[i].IsCoinbaseExchangeRouted || instruments[i].IsCoinbaseSpotRouted {
			routed[instruments[i].InstrumentName] = struct{}{}
		}
	}
	if a == asset.Spot {
		e.coinbaseRoutedSpotMtx.Lock()
		e.coinbaseRoutedSpot = routed
		e.coinbaseRoutedSpotMtx.Unlock()
	}
	return pairs, nil
}

// isCoinbaseRoutedSpot reports whether Deribit routes a spot instrument to Coinbase Exchange
func (e *Exchange) isCoinbaseRoutedSpot(instrumentName string) bool {
	e.coinbaseRoutedSpotMtx.RLock()
	defer e.coinbaseRoutedSpotMtx.RUnlock()
	_, ok := e.coinbaseRoutedSpot[instrumentName]
	return ok
}

// UpdateTickers updates the tickers of an asset's enabled pairs from the book summaries of the currencies Deribit
// summarises them under, one request per currency, as Deribit summarises no more than one currency at a time; summaries
// leave out the best bid and ask sizes and the index price, which UpdateTicker returns
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	kind, err := e.assetKind(a)
	if err != nil {
		return err
	}
	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	names := make([]string, len(pairs))
	var currencies currency.Currencies
	var errs error
	for i, pair := range pairs {
		if names[i], err = e.pairInstrument(pair, a); err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if c := summaryCurrency(a, names[i]); !currencies.Contains(c) {
			currencies = append(currencies, c)
		}
	}
	overWebsocket := e.websocketForWrapper(false)
	summaries := make(map[string]*BookSummary)
	for _, c := range currencies {
		resp, err := e.getBookSummaryByCurrency(ctx, &BookSummaryByCurrencyRequest{Currency: c, Kind: kind}, overWebsocket)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s: %w", c, err))
			continue
		}
		for i := range resp {
			summaries[resp[i].InstrumentName] = &resp[i]
		}
	}
	prices := make([]ticker.Price, 0, len(pairs))
	for i, pair := range pairs {
		name := names[i]
		if name == "" {
			continue
		}
		summary, ok := summaries[name]
		if !ok {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", errBookSummaryNotFound, name))
			continue
		}
		prices = append(prices, *summary.tickerPrice(e.Name, pair, a))
	}
	if len(prices) == 0 {
		return errs
	}
	_, err = ticker.ProcessBatch(prices)
	return common.AppendError(errs, err)
}

// UpdateTicker updates and returns a pair's ticker
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	name, err := e.pairInstrument(p, a)
	if err != nil {
		return nil, err
	}
	resp, err := e.getTicker(ctx, name, e.websocketForWrapper(false))
	if err != nil {
		return nil, err
	}
	if err := ticker.ProcessTicker(resp.tickerPrice(e.Name, p, a)); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// summaryCurrency returns the currency Deribit summarises an instrument's book under: a spot pair's base currency, or
// a derivative's settlement currency, which is the quote currency of a linear derivative's underlying pair
func summaryCurrency(a asset.Item, instrumentName string) currency.Code {
	underlying, _, _ := strings.Cut(instrumentName, currency.DashDelimiter)
	base, quote, linear := strings.Cut(underlying, currency.UnderscoreDelimiter)
	if linear && a != asset.Spot {
		return currency.NewCode(quote)
	}
	return currency.NewCode(base)
}

// tickerPrice converts a ticker; min_price and max_price bound the price an order may carry rather than the day's
// range, and implied_bid and implied_ask, served only for combos, are priced off the legs, while best_bid_amount and
// best_ask_amount size the combo's own resting orders
func (t *TickerResponse) tickerPrice(exchangeName string, p currency.Pair, a asset.Item) *ticker.Price {
	return &ticker.Price{
		ExchangeName: exchangeName,
		Pair:         p,
		AssetType:    a,
		LastUpdated:  t.Timestamp.Time(),
		Bid:          t.BestBidPrice,
		BidSize:      t.BestBidAmount,
		Ask:          t.BestAskPrice,
		AskSize:      t.BestAskAmount,
		Last:         t.LastPrice,
		Close:        t.LastPrice,
		High:         t.Stats.High,
		Low:          t.Stats.Low,
		BaseVolume:   t.Stats.Volume,
		QuoteVolume:  quoteVolume(t.Stats.VolumeUSD, t.Stats.VolumeNotional, a),
		MarkPrice:    t.MarkPrice,
		IndexPrice:   t.IndexPrice,
		OpenInterest: t.OpenInterest,
	}
}

// tickerPrice converts a book summary, whose creation timestamp is when Deribit compiled it
func (b *BookSummary) tickerPrice(exchangeName string, p currency.Pair, a asset.Item) *ticker.Price {
	return &ticker.Price{
		ExchangeName: exchangeName,
		Pair:         p,
		AssetType:    a,
		LastUpdated:  b.CreationTimestamp.Time(),
		Bid:          b.BidPrice,
		Ask:          b.AskPrice,
		Last:         b.LastPrice,
		Close:        b.LastPrice,
		High:         b.High,
		Low:          b.Low,
		BaseVolume:   b.Volume,
		QuoteVolume:  quoteVolume(b.VolumeUSD, b.VolumeNotional, a),
		MarkPrice:    b.MarkPrice,
		OpenInterest: b.OpenInterest,
	}
}

// quoteVolume returns the 24 hour volume in the instrument's quote currency. Deribit serves
// volume_notional in that currency wherever it applies, while volume_usd is the USD premium
// turnover, which a BTC or ETH quoted option must not report as its own
func quoteVolume(volumeUSD, volumeNotional float64, a asset.Item) float64 {
	if volumeNotional != 0 || a == asset.Options || a == asset.OptionCombo {
		return volumeNotional
	}
	return volumeUSD
}

// UpdateOrderbook updates and returns a pair's order book
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	name, err := e.pairInstrument(p, a)
	if err != nil {
		return nil, err
	}
	resp, err := e.getOrderBook(ctx, &OrderBookRequest{InstrumentName: name, Depth: orderBookDepth}, e.websocketForWrapper(false))
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		Bids:              orderBookLevels(resp.Bids),
		Asks:              orderBookLevels(resp.Asks),
		LastUpdated:       resp.Timestamp.Time(),
		LastUpdateID:      int64(resp.ChangeID), //nolint:gosec // Deribit's change IDs fit int64
		ValidateOrderbook: e.ValidateOrderbook,
	}
	if err := book.Process(); err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, p, a)
}

// orderBookLevels converts book levels, which Deribit sends best first, leaving out a combo's levels at a zero price,
// which its spread can be quoted at but a book cannot hold
func orderBookLevels(levels []OrderBookLevel) orderbook.Levels {
	converted := make(orderbook.Levels, 0, len(levels))
	for i := range levels {
		if levels[i].Price != 0 {
			converted = append(converted, orderbook.Level{Price: levels[i].Price, Amount: levels[i].Amount})
		}
	}
	return converted
}

// UpdateAccountBalances retrieves the account's balance in every currency it holds. Deribit's balances back every kind
// of instrument, so they are saved under the asset requested. The total is the cash balance and the free amount what
// can be withdrawn, which leaves out the margin held, positive session profit and reserved balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	resp, err := e.getAccountSummaries(ctx, &GetAccountSummariesRequest{}, e.websocketForWrapper(true))
	if err != nil {
		return nil, err
	}
	subAccount := accounts.NewSubAccount(a, "")
	for i := range resp.Summaries {
		summary := &resp.Summaries[i]
		subAccount.Balances.Set(summary.Currency, accounts.Balance{
			Total: summary.Balance,
			Hold:  max(summary.Balance-summary.AvailableWithdrawalFunds, 0),
			Free:  summary.AvailableWithdrawalFunds,
		})
	}
	subAccounts := accounts.SubAccounts{subAccount}
	return subAccounts, e.Accounts.Save(ctx, subAccounts, true)
}

// GetAccountFundingHistory returns the latest page of deposits and withdrawals in each currency available to the
// account
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	overWebsocket := e.websocketForWrapper(true)
	currencies, err := e.getAccountCurrencies(ctx, overWebsocket)
	if err != nil {
		return nil, err
	}
	var resp []exchange.FundingHistory
	for i := range currencies {
		deposits, err := e.getDeposits(ctx, &WalletHistoryRequest{Currency: currencies[i].Currency, Count: maximumWalletHistoryCount}, overWebsocket)
		if err != nil {
			return nil, err
		}
		for j := range deposits.Deposits {
			received := &deposits.Deposits[j]
			resp = append(resp, exchange.FundingHistory{
				ExchangeName:      e.Name,
				Status:            received.State,
				TransferID:        received.TransactionID,
				Timestamp:         received.ReceivedTimestamp.Time(),
				Currency:          received.Currency.String(),
				Amount:            received.Amount,
				TransferType:      "deposit",
				CryptoToAddress:   received.Address,
				CryptoFromAddress: received.SourceAddress,
				CryptoTxID:        received.TransactionID,
			})
		}
		withdrawals, err := e.getWithdrawals(ctx, &WalletHistoryRequest{Currency: currencies[i].Currency, Count: maximumWalletHistoryCount}, overWebsocket)
		if err != nil {
			return nil, err
		}
		for j := range withdrawals.Withdrawals {
			withdrawal := &withdrawals.Withdrawals[j]
			resp = append(resp, exchange.FundingHistory{
				ExchangeName:    e.Name,
				Status:          withdrawal.State,
				TransferID:      string(withdrawal.ID),
				Timestamp:       withdrawal.CreatedTimestamp.Time(),
				Currency:        withdrawal.Currency.String(),
				Amount:          withdrawal.Amount,
				Fee:             withdrawal.Fee,
				TransferType:    "withdrawal",
				CryptoToAddress: withdrawal.Address,
				CryptoTxID:      withdrawal.TransactionID,
				CryptoChain:     withdrawal.Network,
			})
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns the latest page of a currency's withdrawals, or of each currency available to the
// account when the currency is empty
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	overWebsocket := e.websocketForWrapper(true)
	currencies := []currency.Code{c}
	if c.IsEmpty() {
		available, err := e.getAccountCurrencies(ctx, overWebsocket)
		if err != nil {
			return nil, err
		}
		currencies = make([]currency.Code, len(available))
		for i := range available {
			currencies[i] = available[i].Currency
		}
	}
	var resp []exchange.WithdrawalHistory
	for _, ccy := range currencies {
		withdrawals, err := e.getWithdrawals(ctx, &WalletHistoryRequest{Currency: ccy, Count: maximumWalletHistoryCount}, overWebsocket)
		if err != nil {
			return nil, err
		}
		for i := range withdrawals.Withdrawals {
			withdrawal := &withdrawals.Withdrawals[i]
			resp = append(resp, exchange.WithdrawalHistory{
				Status:          withdrawal.State,
				TransferID:      string(withdrawal.ID),
				Timestamp:       withdrawal.CreatedTimestamp.Time(),
				Currency:        withdrawal.Currency.String(),
				Amount:          withdrawal.Amount,
				Fee:             withdrawal.Fee,
				TransferType:    "withdrawal",
				CryptoToAddress: withdrawal.Address,
				CryptoTxID:      withdrawal.TransactionID,
				CryptoChain:     withdrawal.Network,
			})
		}
	}
	return resp, nil
}

// GetRecentTrades returns a pair's latest trades; Deribit serves no trade tape for spot routed to Coinbase Exchange
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	name, err := e.pairInstrument(p, a)
	if err != nil {
		return nil, err
	}
	resp, err := e.getLastTradesByInstrument(ctx, &LastTradesByInstrumentRequest{InstrumentName: name, Count: tradesPageSize}, e.websocketForWrapper(false))
	if err != nil {
		return nil, err
	}
	trades := make([]trade.Data, 0, len(resp.Trades))
	for i := range resp.Trades {
		converted, err := e.tradeData(&resp.Trades[i], p, a)
		if err != nil {
			return nil, err
		}
		trades = append(trades, *converted)
	}
	return trades, nil
}

// GetHistoricTrades returns a pair's trades within a time range, paging forward a page of trades at a time
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	name, err := e.pairInstrument(p, a)
	if err != nil {
		return nil, err
	}
	overWebsocket := e.websocketForWrapper(false)
	var trades []trade.Data
	seen := make(map[string]struct{})
	for {
		resp, err := e.getLastTradesByInstrumentAndTime(ctx, &LastTradesByInstrumentAndTimeRequest{
			InstrumentName: name,
			StartTime:      start,
			EndTime:        end,
			Count:          tradesPageSize,
			Sorting:        "asc",
		}, overWebsocket)
		if err != nil {
			return nil, err
		}
		for i := range resp.Trades {
			// Each page starts at the last page's final timestamp, which repeats the trades sharing it
			if _, ok := seen[resp.Trades[i].TradeID]; ok {
				continue
			}
			seen[resp.Trades[i].TradeID] = struct{}{}
			converted, err := e.tradeData(&resp.Trades[i], p, a)
			if err != nil {
				return nil, err
			}
			trades = append(trades, *converted)
		}
		if !resp.HasMore || len(resp.Trades) == 0 {
			return trades, nil
		}
		last := resp.Trades[len(resp.Trades)-1].Timestamp.Time()
		if !last.After(start) {
			return nil, fmt.Errorf("%w: %s at %s", errTradePagingStalled, name, last)
		}
		start = last
	}
}

// tradeData converts a public trade
func (e *Exchange) tradeData(t *PublicTrade, p currency.Pair, a asset.Item) (*trade.Data, error) {
	side, err := order.StringToOrderSide(t.Direction)
	if err != nil {
		return nil, err
	}
	return &trade.Data{
		TID:          t.TradeID,
		Exchange:     e.Name,
		CurrencyPair: p,
		AssetType:    a,
		Side:         side,
		Price:        t.Price,
		Amount:       t.Amount,
		Timestamp:    t.Timestamp.Time(),
	}, nil
}

// GetHistoricCandles returns a pair's candles within a time range
func (e *Exchange) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(p, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	candles, err := e.chartCandles(ctx, req.RequestFormatted, a, req.ExchangeInterval, req.Start, req.End)
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended returns a pair's candles within a time range, a request per range Deribit serves at once
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(p, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	candles := make([]kline.Candle, 0, req.Size())
	for i := range req.RangeHolder.Ranges {
		ranged, err := e.chartCandles(ctx, req.RequestFormatted, a, req.ExchangeInterval, req.RangeHolder.Ranges[i].Start.Time, req.RangeHolder.Ranges[i].End.Time)
		if err != nil {
			return nil, err
		}
		candles = append(candles, ranged...)
	}
	return req.ProcessResponse(candles)
}

// chartCandles returns a pair's chart candles within a time range
func (e *Exchange) chartCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) ([]kline.Candle, error) {
	resp, err := e.getTradingViewChartData(ctx, &TradingViewChartDataRequest{
		InstrumentName: formatPairString(a, p),
		StartTime:      start,
		EndTime:        end,
		Resolution:     interval,
	}, e.websocketForWrapper(false))
	if err != nil {
		return nil, err
	}
	length := len(resp.Ticks)
	if len(resp.Open) != length || len(resp.High) != length || len(resp.Low) != length || len(resp.Close) != length || len(resp.Volume) != length {
		return nil, fmt.Errorf("%w: %d ticks, %d opens, %d highs, %d lows, %d closes and %d volumes", errCandleFieldsMismatch, length, len(resp.Open), len(resp.High), len(resp.Low), len(resp.Close), len(resp.Volume))
	}
	candles := make([]kline.Candle, length)
	for i := range resp.Ticks {
		candles[i] = kline.Candle{
			Time:   resp.Ticks[i].Time().UTC(),
			Open:   resp.Open[i],
			High:   resp.High[i],
			Low:    resp.Low[i],
			Close:  resp.Close[i],
			Volume: resp.Volume[i],
		}
	}
	return candles, nil
}

// GetServerTime returns Deribit's current time
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return e.getTime(ctx, e.websocketForWrapper(false))
}

// SubmitOrder places an order; a post only order is rejected rather than repriced when it would take liquidity, which
// Coinbase-routed spot requires
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	name, err := e.pairInstrument(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	req, err := submitOrderRequest(s)
	if err != nil {
		return nil, err
	}
	req.InstrumentName = name
	method := "private/buy"
	if s.Side.IsShort() {
		method = "private/sell"
	}
	placed, err := e.placeOrder(ctx, method, req, e.websocketForWrapper(true))
	if err != nil {
		return nil, err
	}
	detail, err := e.orderDetail(&placed.Order)
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(placed.Order.OrderID)
	if err != nil {
		return nil, err
	}
	resp.Status = detail.Status
	resp.Price = detail.Price
	resp.AverageExecutedPrice = detail.AverageExecutedPrice
	resp.RemainingAmount = detail.RemainingAmount
	resp.Date = detail.Date
	resp.LastUpdated = detail.LastUpdated
	resp.Trades = make([]order.TradeHistory, len(placed.Trades))
	for i := range placed.Trades {
		if resp.Trades[i], err = tradeHistory(e.Name, &placed.Trades[i]); err != nil {
			return nil, err
		}
		resp.Fee += placed.Trades[i].Fee
		resp.FeeAsset = placed.Trades[i].FeeCurrency
	}
	return resp, nil
}

// submitOrderRequest returns the order request placing an order, but its instrument
func submitOrderRequest(s *order.Submit) (*OrderRequest, error) {
	orderType, err := orderTypeParam(s.Type)
	if err != nil {
		return nil, err
	}
	timeInForce, err := timeInForceParam(s.TimeInForce)
	if err != nil {
		return nil, err
	}
	postOnly := s.TimeInForce.Is(order.PostOnly)
	req := &OrderRequest{
		Amount:         s.Amount,
		Type:           orderType,
		Label:          s.ClientOrderID,
		TimeInForce:    timeInForce,
		PostOnly:       postOnly,
		RejectPostOnly: postOnly,
		ReduceOnly:     s.ReduceOnly,
	}
	switch s.Type {
	case order.Limit, order.StopLimit, order.TakeProfit:
		req.Price = s.Price
	}
	switch s.Type {
	case order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitMarket, order.TrailingStop:
		if req.Trigger, err = triggerParam(s.TriggerPriceType); err != nil {
			return nil, err
		}
		req.TriggerPrice = s.TriggerPrice
	}
	if s.Type == order.TrailingStop {
		if s.TrackingMode != order.Distance || s.TrackingValue <= 0 {
			return nil, errTrailingStopDistance
		}
		req.TriggerOffset = s.TrackingValue
	}
	return req, nil
}

// orderTypeParam returns the order type an order is placed as
func orderTypeParam(t order.Type) (string, error) {
	switch t {
	case order.Limit:
		return OrderTypeLimit, nil
	case order.Market:
		return OrderTypeMarket, nil
	case order.StopLimit:
		return OrderTypeStopLimit, nil
	case order.StopMarket:
		return OrderTypeStopMarket, nil
	case order.TakeProfit:
		return OrderTypeTakeLimit, nil
	case order.TakeProfitMarket:
		return OrderTypeTakeMarket, nil
	case order.TrailingStop:
		return OrderTypeTrailingStop, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
	}
}

// timeInForceParam returns the time in force an order is placed with, which is good till cancelled when unset; post
// only is a separate param
func timeInForceParam(tif order.TimeInForce) (string, error) {
	switch tif &^ order.PostOnly {
	case order.UnknownTIF:
		return "", nil
	case order.GoodTillCancel:
		return "good_til_cancelled", nil
	case order.GoodTillDay:
		return "good_til_day", nil
	case order.FillOrKill:
		return "fill_or_kill", nil
	case order.ImmediateOrCancel:
		return "immediate_or_cancel", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, tif)
	}
}

// triggerParam returns the price a trigger order triggers on; the zero price type is the last price, the only one
// Deribit allows for every instrument
func triggerParam(priceType order.PriceType) (string, error) {
	switch priceType {
	case order.LastPrice:
		return "last_price", nil
	case order.IndexPrice:
		return "index_price", nil
	case order.MarkPrice:
		return "mark_price", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnknownPriceType, priceType)
	}
}

// tradeHistory converts one of the account's trades to an order's trade
func tradeHistory(exchangeName string, t *UserTrade) (order.TradeHistory, error) {
	side, err := order.StringToOrderSide(t.Direction)
	if err != nil {
		return order.TradeHistory{}, err
	}
	orderType, err := parseOrderType(t.OrderType)
	if err != nil {
		return order.TradeHistory{}, err
	}
	return order.TradeHistory{
		Price:     t.Price,
		Amount:    t.Amount,
		Fee:       t.Fee,
		Exchange:  exchangeName,
		TID:       t.TradeID,
		Type:      orderType,
		Side:      side,
		Timestamp: t.Timestamp.Time(),
		IsMaker:   t.Liquidity == "M",
		FeeAsset:  t.FeeCurrency.String(),
	}, nil
}

// ModifyOrder edits an open order, identified by its ID or by its label in the pair; an amount left unset keeps the
// order's amount, which Deribit requires with every edit
func (e *Exchange) ModifyOrder(ctx context.Context, m *order.Modify) (*order.ModifyResponse, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	overWebsocket := e.websocketForWrapper(true)
	postOnly := m.TimeInForce.Is(order.PostOnly)
	var edited *OrderResponse
	if m.OrderID != "" {
		amount := m.Amount
		if amount == 0 {
			existing, err := e.getOrderState(ctx, m.OrderID, overWebsocket)
			if err != nil {
				return nil, err
			}
			amount = existing.Amount
		}
		var err error
		edited, err = e.editOrder(ctx, &EditOrderRequest{
			OrderID:        m.OrderID,
			Amount:         amount,
			Price:          m.Price,
			PostOnly:       postOnly,
			RejectPostOnly: postOnly,
			TriggerPrice:   m.TriggerPrice,
		}, overWebsocket)
		if err != nil {
			return nil, err
		}
	} else {
		name, err := e.pairInstrument(m.Pair, m.AssetType)
		if err != nil {
			return nil, err
		}
		amount := m.Amount
		if amount == 0 {
			existing, err := e.getOrdersByLabel(ctx, "private/get_open_orders_by_label", &OrdersByLabelRequest{Currency: m.Pair.Base, Label: m.ClientOrderID}, overWebsocket)
			if err != nil {
				return nil, err
			}
			existing = slices.DeleteFunc(existing, func(o Order) bool { return o.InstrumentName != name })
			if len(existing) != 1 {
				return nil, fmt.Errorf("%w: %d orders labelled %q in %s", errOrderLabelNotUnique, len(existing), m.ClientOrderID, name)
			}
			amount = existing[0].Amount
		}
		edited, err = e.editOrderByLabel(ctx, &EditOrderByLabelRequest{
			Label:          m.ClientOrderID,
			InstrumentName: name,
			Amount:         amount,
			Price:          m.Price,
			PostOnly:       postOnly,
			RejectPostOnly: postOnly,
			TriggerPrice:   m.TriggerPrice,
		}, overWebsocket)
		if err != nil {
			return nil, err
		}
	}
	detail, err := e.orderDetail(&edited.Order)
	if err != nil {
		return nil, err
	}
	resp, err := m.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.Exchange = e.Name
	resp.OrderID = detail.OrderID
	resp.ClientOrderID = detail.ClientOrderID
	resp.Type = detail.Type
	resp.Side = detail.Side
	resp.Status = detail.Status
	resp.TimeInForce = detail.TimeInForce
	resp.Price = detail.Price
	resp.Amount = detail.Amount
	resp.TriggerPrice = detail.TriggerPrice
	resp.RemainingAmount = detail.RemainingAmount
	resp.Date = detail.Date
	resp.LastUpdated = detail.LastUpdated
	return resp, nil
}

// CancelOrder cancels an order by its ID
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	_, err := e.cancelOrderByID(ctx, o.OrderID, e.websocketForWrapper(true))
	return err
}

// CancelBatchOrders cancels orders by their IDs, one request each, as Deribit cancels no list of orders; each order's
// status is its state once cancelled, or the error cancelling it
func (e *Exchange) CancelBatchOrders(ctx context.Context, cancels []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(cancels) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	overWebsocket := e.websocketForWrapper(true)
	resp := &order.CancelBatchResponse{Status: make(map[string]string, len(cancels))}
	for i := range cancels {
		if err := cancels[i].Validate(cancels[i].StandardCancel()); err != nil {
			return nil, err
		}
		cancelled, err := e.cancelOrderByID(ctx, cancels[i].OrderID, overWebsocket)
		if err != nil {
			resp.Status[cancels[i].OrderID] = err.Error()
			continue
		}
		resp.Status[cancels[i].OrderID] = cancelled.OrderState
	}
	return resp, nil
}

// CancelAllOrders cancels the orders of a type in a pair, or in every pair of the asset when the pair is empty
func (e *Exchange) CancelAllOrders(ctx context.Context, c *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	if err := c.Validate(); err != nil {
		return resp, err
	}
	orderType, err := cancelOrderTypeParam(c.Type)
	if err != nil {
		return resp, err
	}
	overWebsocket := e.websocketForWrapper(true)
	var cancelled *CancelAllResponse
	if c.Pair.IsEmpty() {
		kind, err := e.assetKind(c.AssetType)
		if err != nil {
			return resp, err
		}
		cancelled, err = e.cancelAllByKindOrType(ctx, &CancelAllByKindOrTypeRequest{Currencies: []currency.Code{anyCurrency}, Kind: kind, Type: orderType, Detailed: true}, overWebsocket)
		if err != nil {
			return resp, err
		}
	} else {
		name, err := e.pairInstrument(c.Pair, c.AssetType)
		if err != nil {
			return resp, err
		}
		cancelled, err = e.cancelAllByInstrument(ctx, &CancelAllByInstrumentRequest{InstrumentName: name, Type: orderType, Detailed: true}, overWebsocket)
		if err != nil {
			return resp, err
		}
	}
	for i := range cancelled.ExecutionReports {
		for j := range cancelled.ExecutionReports[i].Result {
			resp.Add(cancelled.ExecutionReports[i].Result[j].OrderID, cancelled.ExecutionReports[i].Result[j].OrderState)
		}
	}
	return resp, nil
}

// cancelOrderTypeParam returns the order type a cancel all request cancels; market orders never rest
func cancelOrderTypeParam(t order.Type) (string, error) {
	switch t {
	case order.AnyType, order.UnknownType:
		return "all", nil
	case order.Limit:
		return OrderTypeLimit, nil
	case order.Stop, order.StopLimit, order.StopMarket:
		return "stop", nil
	case order.TakeProfit, order.TakeProfitMarket:
		return "take", nil
	case order.TrailingStop:
		return OrderTypeTrailingStop, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
	}
}

// GetOrderInfo returns an order by its ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	resp, err := e.getOrderState(ctx, orderID, e.websocketForWrapper(true))
	if err != nil {
		return nil, err
	}
	return e.orderDetail(resp)
}

// GetDepositAddress returns a currency's current deposit address, on a network for a Coinbase wallet type account
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	resp, err := e.getCurrentDepositAddress(ctx, &GetCurrentDepositAddressRequest{Currency: c, Network: chain}, e.websocketForWrapper(true))
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: resp.Address,
		Tag:     resp.Tag,
		Chain:   resp.Network,
	}, nil
}

// WithdrawCryptocurrencyFunds withdraws to an address in the account's address book; on an account with two-factor
// authentication Deribit returns a SecurityKeyChallengeError, which the withdrawal must be confirmed through with
// Withdraw and WithSecurityKeyAuthorisation
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, r *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.withdraw(ctx, &WithdrawRequest{
		Currency: r.Currency,
		Amount:   r.Amount,
		Address:  r.Crypto.Address,
		Tag:      r.Crypto.AddressTag,
		Network:  r.Crypto.Chain,
	}, e.websocketForWrapper(true))
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   e.Name,
		ID:     string(resp.ID),
		Status: resp.State,
	}, nil
}

// WithdrawFiatFunds is not supported, as Deribit holds no fiat
func (e *Exchange) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank is not supported, as Deribit holds no fiat
func (e *Exchange) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders returns the open orders of the requested pairs, or of every pair of the asset when none are
// requested
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	kind, err := e.assetKind(req.AssetType)
	if err != nil {
		return nil, err
	}
	overWebsocket := e.websocketForWrapper(true)
	var open []Order
	if len(req.Pairs) == 0 {
		if open, err = e.getOpenOrders(ctx, &OpenOrdersRequest{Kind: kind}, overWebsocket); err != nil {
			return nil, err
		}
	}
	for _, pair := range req.Pairs {
		name, err := e.pairInstrument(pair, req.AssetType)
		if err != nil {
			return nil, err
		}
		orders, err := e.getOpenOrdersByInstrument(ctx, &OpenOrdersByInstrumentRequest{InstrumentName: name}, overWebsocket)
		if err != nil {
			return nil, err
		}
		open = append(open, orders...)
	}
	return e.filteredOrders(req, open)
}

// GetOrderHistory returns the latest page of each requested pair's orders, including those older than two days and
// those that never filled
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	overWebsocket := e.websocketForWrapper(true)
	var history []Order
	for _, pair := range req.Pairs {
		name, err := e.pairInstrument(pair, req.AssetType)
		if err != nil {
			return nil, err
		}
		resp, err := e.getOrderHistoryByInstrument(ctx, &OrderHistoryByInstrumentRequest{
			InstrumentName:  name,
			Count:           maximumTradingCount,
			IncludeOld:      true,
			IncludeUnfilled: true,
		}, overWebsocket)
		if err != nil {
			return nil, err
		}
		history = append(history, resp.Orders...)
	}
	return e.filteredOrders(req, history)
}

// filteredOrders converts orders and filters them as requested
func (e *Exchange) filteredOrders(req *order.MultiOrderRequest, orders []Order) (order.FilteredOrders, error) {
	details := make([]order.Detail, len(orders))
	for i := range orders {
		detail, err := e.orderDetail(&orders[i])
		if err != nil {
			return nil, err
		}
		details[i] = *detail
	}
	return req.Filter(e.Name, details), nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	var fee float64
	var err error
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee, err = calculateTradingFee(feeBuilder)
		if err != nil {
			return 0, err
		}
	case exchange.CryptocurrencyDepositFee:
	case exchange.CryptocurrencyWithdrawalFee:
		// Withdrawals are processed instantly if the balance in our hot wallet permits so. We keep only a small percentage of coins in hot storage,
		// therefore there is a chance that your withdrawal cannot be processed immediately. If needed, once a day we will replenish the balance of the hot wallet from the cold storage.
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, a asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, a)
	return e.CheckTransientError(err)
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (e *Exchange) AuthenticateWebsocket(ctx context.Context) error {
	return e.wsLogin(ctx)
}

// GetFuturesContractDetails returns the specifications of every future
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, a asset.Item) ([]futures.Contract, error) {
	if !a.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if a != asset.Futures {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	instruments, err := e.getInstruments(ctx, &InstrumentsRequest{Currency: anyCurrency, Kind: KindFuture}, e.websocketForWrapper(false))
	if err != nil {
		return nil, err
	}
	contracts := make([]futures.Contract, len(instruments))
	for i := range instruments {
		contract, err := e.futuresContract(&instruments[i])
		if err != nil {
			return nil, err
		}
		contracts[i] = *contract
	}
	return contracts, nil
}

// futuresContract converts a future's specification; a perpetual expires in the year 3000, which is left out
func (e *Exchange) futuresContract(instrument *Instrument) (*futures.Contract, error) {
	pair, err := currency.NewPairFromString(instrument.InstrumentName)
	if err != nil {
		return nil, err
	}
	contract := &futures.Contract{
		Exchange:           e.Name,
		Name:               pair,
		Underlying:         currency.NewPair(instrument.BaseCurrency, instrument.QuoteCurrency),
		Asset:              asset.Futures,
		StartDate:          instrument.CreationTimestamp.Time(),
		EndDate:            instrument.ExpirationTimestamp.Time(),
		IsActive:           instrument.IsActive,
		Status:             instrument.State,
		Type:               futuresContractType(instrument.SettlementPeriod),
		SettlementType:     futures.Linear,
		SettlementCurrency: instrument.SettlementCurrency,
		Multiplier:         instrument.ContractSize,
		MaxLeverage:        instrument.MaxLeverage,
	}
	if instrument.InstrumentType == "reversed" {
		contract.SettlementType = futures.Inverse
	}
	if contract.Type == futures.Perpetual {
		contract.EndDate = time.Time{}
	}
	return contract, nil
}

// futuresContractType converts a future's settlement period; quarterly futures settle monthly, like monthly ones
func futuresContractType(settlementPeriod string) futures.ContractType {
	switch settlementPeriod {
	case "perpetual":
		return futures.Perpetual
	case "day":
		return futures.Daily
	case "week":
		return futures.Weekly
	case "month":
		return futures.Monthly
	default:
		return futures.Unknown
	}
}

// UpdateOrderExecutionLimits loads the price and amount increments of an asset's instruments, whose minimum amount is
// also their amount increment
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	kind, err := e.assetKind(a)
	if err != nil {
		return err
	}
	instruments, err := e.getInstruments(ctx, &InstrumentsRequest{Currency: anyCurrency, Kind: kind}, e.websocketForWrapper(false))
	if err != nil {
		return err
	}
	levels := make([]limits.MinMaxLevel, len(instruments))
	for i := range instruments {
		pair, err := currency.NewPairFromString(instruments[i].InstrumentName)
		if err != nil {
			return err
		}
		levels[i] = limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
			PriceStepIncrementSize:  instruments[i].TickSize,
			MinimumBaseAmount:       instruments[i].MinTradeAmount,
			AmountStepIncrementSize: instruments[i].MinTradeAmount,
		}
	}
	return limits.Load(levels)
}

// GetFuturesPositionSummary returns the account's position in a future. Sizes are in the base currency, the notional
// size its value at the mark price, and the contract size the number of contracts
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, r *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, r)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w: %s", futures.ErrNotFuturesAsset, r.Asset)
	}
	name, err := e.pairInstrument(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	instrument, err := e.getInstrument(ctx, name, e.websocketForWrapper(false))
	if err != nil {
		return nil, err
	}
	position, err := e.getPosition(ctx, name, e.websocketForWrapper(true))
	if err != nil {
		return nil, err
	}
	settlementType := futures.Linear
	if instrument.InstrumentType == "reversed" {
		settlementType = futures.Inverse
	}
	var contracts float64
	if instrument.ContractSize != 0 {
		contracts = position.Size / instrument.ContractSize
	}
	return &futures.PositionSummary{
		Pair:                         r.Pair,
		Asset:                        r.Asset,
		Currency:                     instrument.SettlementCurrency,
		NotionalSize:                 decimal.MustFromFloat(position.SizeCurrency * position.MarkPrice),
		Leverage:                     decimal.MustFromFloat(position.Leverage),
		MaintenanceMarginRequirement: decimal.MustFromFloat(position.MaintenanceMargin),
		InitialMarginRequirement:     decimal.MustFromFloat(position.InitialMargin),
		EstimatedLiquidationPrice:    decimal.MustFromFloat(position.EstimatedLiquidationPrice),
		MarkPrice:                    decimal.MustFromFloat(position.MarkPrice),
		CurrentSize:                  decimal.MustFromFloat(position.SizeCurrency),
		ContractSize:                 decimal.MustFromFloat(contracts),
		ContractMultiplier:           decimal.MustFromFloat(instrument.ContractSize),
		ContractSettlementType:       settlementType,
		AverageOpenPrice:             decimal.MustFromFloat(position.AveragePrice),
		UnrealisedPNL:                decimal.MustFromFloat(position.TotalProfitLoss - position.RealisedProfitLoss),
		RealisedPNL:                  decimal.MustFromFloat(position.RealisedProfitLoss),
	}, nil
}

// GetOpenInterest returns the open interest of each requested pair
func (e *Exchange) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: a pair is required", common.ErrFunctionNotSupported)
	}
	overWebsocket := e.websocketForWrapper(false)
	resp := make([]futures.OpenInterest, 0, len(keys))
	for i := range keys {
		if keys[i].Asset == asset.Spot {
			return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, keys[i].Asset)
		}
		name, err := e.pairInstrument(keys[i].Pair(), keys[i].Asset)
		if err != nil {
			return nil, err
		}
		summaries, err := e.getBookSummaryByInstrument(ctx, name, overWebsocket)
		if err != nil {
			return nil, err
		}
		if len(summaries) == 0 {
			return nil, fmt.Errorf("%w: %s", errBookSummaryNotFound, name)
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, keys[i].Asset, keys[i].Pair()),
			OpenInterest: summaries[0].OpenInterest,
		})
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	if cp.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	switch a {
	case asset.Futures:
		isPerp, err := e.IsPerpetualFutureCurrency(a, cp)
		if err != nil {
			return "", err
		}
		if isPerp {
			return tradeBaseURL + tradeFutures + cp.Base.Upper().String() + currency.UnderscoreDelimiter + cp.Quote.Upper().String(), nil
		}
		return tradeBaseURL + tradeFutures + cp.Upper().String(), nil
	case asset.Spot:
		cp.Delimiter = currency.UnderscoreDelimiter
		return tradeBaseURL + tradeSpot + cp.Upper().String(), nil
	case asset.Options:
		baseString := cp.Base.Upper().String()
		quoteString := cp.Quote.Upper().String()
		if _, optionType, found := strings.CutLast(quoteString, currency.DashDelimiter); found && (optionType == "C" || optionType == "P") {
			expiry, _, _ := strings.Cut(quoteString, currency.DashDelimiter)
			return tradeBaseURL + tradeOptions + baseString + "/" + baseString + currency.DashDelimiter + expiry, nil
		}
		return tradeBaseURL + tradeOptions + baseString, nil
	case asset.FutureCombo:
		return tradeBaseURL + tradeFuturesCombo + cp.Upper().String(), nil
	case asset.OptionCombo:
		return tradeBaseURL + tradeOptionsCombo + cp.Base.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
// differs by exchange
func (e *Exchange) IsPerpetualFutureCurrency(assetType asset.Item, pair currency.Pair) (bool, error) {
	if pair.IsEmpty() {
		return false, currency.ErrCurrencyPairEmpty
	}
	if assetType != asset.Futures {
		// deribit considers future combo, even if ending in "PERP" to not be a perpetual
		return false, nil
	}
	pqs := strings.Split(pair.Quote.Upper().String(), currency.DashDelimiter)
	return pqs[len(pqs)-1] == perpString, nil
}

// GetLatestFundingRates returns a perpetual's latest hourly funding record, as its 8 hour rate
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, r)
	}
	name, err := e.perpetualInstrument(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	records, err := e.getFundingRateHistory(ctx, &InstrumentTimeRangeRequest{InstrumentName: name, StartTime: now.Add(-latestFundingRateWindow), EndTime: now}, e.websocketForWrapper(false))
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%w: %s", fundingrate.ErrNoFundingRatesFound, name)
	}
	latest := records[len(records)-1]
	return []fundingrate.LatestRateResponse{{
		Exchange:    e.Name,
		Asset:       r.Asset,
		Pair:        r.Pair,
		LatestRate:  fundingrate.Rate{Time: latest.Timestamp.Time(), Rate: decimal.MustFromFloat(latest.Interest8Hour)},
		TimeChecked: now,
	}}, nil
}

// GetHistoricalFundingRates returns a perpetual's hourly funding rates within a time range. Deribit returns the latest
// 744 hours of a range, oldest first, so earlier pages end before the first record of the last
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, r)
	}
	if r.IncludePayments {
		return nil, fmt.Errorf("include payments %w", common.ErrNotYetImplemented)
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	name, err := e.perpetualInstrument(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	overWebsocket := e.websocketForWrapper(false)
	var rates []fundingrate.Rate
	end := r.EndDate
	for {
		records, err := e.getFundingRateHistory(ctx, &InstrumentTimeRangeRequest{InstrumentName: name, StartTime: r.StartDate, EndTime: end}, overWebsocket)
		if err != nil {
			return nil, err
		}
		page := make([]fundingrate.Rate, 0, len(records)+len(rates))
		for i := range records {
			page = append(page, fundingrate.Rate{Time: records[i].Timestamp.Time(), Rate: decimal.MustFromFloat(records[i].Interest1Hour)})
		}
		rates = append(page, rates...)
		if len(records) < maximumFundingRatePage {
			break
		}
		next := records[0].Timestamp.Time().Add(-time.Millisecond)
		if !next.After(r.StartDate) || !next.Before(end) {
			break
		}
		end = next
	}
	if len(rates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}
	return &fundingrate.HistoricalRates{
		Exchange:        e.Name,
		Asset:           r.Asset,
		Pair:            r.Pair,
		FundingRates:    rates,
		StartDate:       rates[0].Time,
		EndDate:         r.EndDate,
		LatestRate:      rates[len(rates)-1],
		PaymentCurrency: r.PaymentCurrency,
	}, nil
}

// perpetualInstrument returns a perpetual's instrument name
func (e *Exchange) perpetualInstrument(a asset.Item, p currency.Pair) (string, error) {
	isPerpetual, err := e.IsPerpetualFutureCurrency(a, p)
	if err != nil {
		return "", err
	}
	if !isPerpetual {
		return "", fmt.Errorf("%w: %s %s", futures.ErrNotPerpetualFuture, a, p)
	}
	return e.pairInstrument(p, a)
}

func formatPairString(assetType asset.Item, pair currency.Pair) string {
	if pair.Delimiter == "" {
		if assetType == asset.Spot {
			pair.Delimiter = currency.UnderscoreDelimiter
		} else {
			pair.Delimiter = currency.DashDelimiter
		}
	}

	switch assetType {
	case asset.Futures:
		return formatFuturesTradablePair(pair)
	case asset.Options:
		return optionPairToString(pair)
	case asset.OptionCombo:
		return optionComboPairToString(pair)
	case asset.FutureCombo:
		return futureComboPairToString(pair)
	}
	return pair.String()
}

func timeInForceFromString(timeInForceString string, postOnly bool) (order.TimeInForce, error) {
	tif, err := order.StringToTimeInForce(timeInForceString)
	if err != nil {
		return order.UnknownTIF, err
	}
	if postOnly {
		tif |= order.PostOnly
	}
	return tif, nil
}

// GetResolutionFromInterval returns the string representation of intervals given kline.Interval instance.
func (e *Exchange) GetResolutionFromInterval(interval kline.Interval) (string, error) {
	switch interval {
	case kline.HundredMilliseconds:
		return "100ms", nil
	case kline.OneMin:
		return "1", nil
	case kline.ThreeMin:
		return "3", nil
	case kline.FiveMin:
		return "5", nil
	case kline.TenMin:
		return "10", nil
	case kline.FifteenMin:
		return "15", nil
	case kline.ThirtyMin:
		return "30", nil
	case kline.OneHour:
		return "60", nil
	case kline.TwoHour:
		return "120", nil
	case kline.ThreeHour:
		return "180", nil
	case kline.SixHour:
		return "360", nil
	case kline.TwelveHour:
		return "720", nil
	case kline.OneDay:
		return "1D", nil
	case kline.Raw:
		return interval.Word(), nil
	default:
		return "", kline.ErrUnsupportedInterval
	}
}

// GetAssetKind returns the asset type (kind) string representation.
func (e *Exchange) GetAssetKind(assetType asset.Item) string {
	switch assetType {
	case asset.Options:
		return "option"
	case asset.Futures:
		return "future"
	case asset.FutureCombo, asset.OptionCombo, asset.Spot:
		return assetType.String()
	default:
		return "any"
	}
}

// StringToAssetKind returns the asset of an instrument kind
func (e *Exchange) StringToAssetKind(kind string) (asset.Item, error) {
	switch strings.ToLower(kind) {
	case KindOption:
		return asset.Options, nil
	case KindFuture:
		return asset.Futures, nil
	case KindSpot:
		return asset.Spot, nil
	case KindFutureCombo:
		return asset.FutureCombo, nil
	case KindOptionCombo:
		return asset.OptionCombo, nil
	default:
		return asset.Empty, fmt.Errorf("%w: kind %q", asset.ErrNotSupported, kind)
	}
}

// getAssetPairByInstrument is able to determine the asset type and currency pair
// based on the received instrument ID
func getAssetPairByInstrument(instrument string) (asset.Item, currency.Pair, error) {
	if instrument == "" {
		return asset.Empty, currency.EMPTYPAIR, currency.ErrSymbolStringEmpty
	}

	item, err := getAssetFromInstrument(instrument)
	if err != nil {
		return asset.Empty, currency.EMPTYPAIR, err
	}
	cp, err := currency.NewPairFromString(instrument)
	if err != nil {
		return asset.Empty, currency.EMPTYPAIR, err
	}
	return item, cp, nil
}

// getAssetFromInstrument extrapolates the asset type from the instrument formatting as each type is unique
func getAssetFromInstrument(instrument string) (asset.Item, error) {
	currencyParts := strings.Split(instrument, currency.DashDelimiter)
	partsLen := len(currencyParts)
	currencySuffix := currencyParts[partsLen-1]
	hasUnderscore := strings.Contains(instrument, currency.UnderscoreDelimiter)
	switch {
	case partsLen == 1 && !hasUnderscore: // no pair delimiter found
		return asset.Empty, fmt.Errorf("%w %s", errUnsupportedInstrumentFormat, instrument)
	case partsLen == 1: // spot pairs use underscore eg BTC_USDC
		return asset.Spot, nil
	case partsLen == 2: // futures pairs use single dash eg ETH_USDC-PERPETUAL, BTC-12SEP25
		return asset.Futures, nil
	case currencySuffix == "C", currencySuffix == "P": // options end in P or C to denote puts or calls eg BTC-26SEP25-30000-C
		return asset.Options, nil
	case partsLen >= 3 && currencyParts[partsLen-2] == "FS" && strings.Contains(currencySuffix, currency.UnderscoreDelimiter):
		// futures combos have underlying-FS-shortLeg_longLeg
		// eg BTC-FS-28NOV25_PERP or BTC-USDC-FS-28NOV25_PERP
		return asset.FutureCombo, nil
	case partsLen == 4: // option combos with more than 3 parts eg BTC_USDC-PS-19SEP25-113000_111000
		return asset.OptionCombo, nil
	default: // deribit has changed their format and needs a review
		return asset.Empty, fmt.Errorf("%w %s", errUnsupportedInstrumentFormat, instrument)
	}
}

func calculateTradingFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	assetType, err := getAssetFromInstrument(feeBuilder.Pair.String())
	if err != nil {
		return 0, err
	}
	switch assetType {
	case asset.Futures, asset.FutureCombo:
		switch {
		case strings.HasSuffix(feeBuilder.Pair.String(), "USDC-PERPETUAL"):
			if feeBuilder.IsMaker {
				return 0, nil
			}
			return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0005, nil
		case strings.HasPrefix(feeBuilder.Pair.String(), currency.BTC.String()),
			strings.HasPrefix(feeBuilder.Pair.String(), currency.ETH.String()):
			if strings.HasSuffix(feeBuilder.Pair.String(), perpString) {
				if feeBuilder.IsMaker {
					return 0, nil
				}
				return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0005, nil
			}
			// weekly futures contracts
			if feeBuilder.IsMaker {
				return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0001, nil
			}
			return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0005, nil
		case strings.HasPrefix(feeBuilder.Pair.String(), "SOL"): // perpetual and weekly SOL contracts
			if feeBuilder.IsMaker {
				return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.00002, nil
			}
			return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0005, nil
		}
	case asset.Options, asset.OptionCombo:
		switch {
		case strings.HasPrefix(feeBuilder.Pair.String(), currency.BTC.String()),
			strings.HasPrefix(feeBuilder.Pair.String(), currency.ETH.String()):
			return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0003, nil
		case strings.HasPrefix(feeBuilder.Pair.String(), "SOL"):
			if feeBuilder.IsMaker {
				return 0, nil
			}
			return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.0003, nil
		}
	}
	return 0, fmt.Errorf("%w asset: %s", asset.ErrNotSupported, assetType.String())
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.0003 * price * amount
}

func formatFuturesTradablePair(pair currency.Pair) string {
	var instrumentID string
	if result := strings.Split(pair.String(), currency.DashDelimiter); len(result) == 3 {
		instrumentID = strings.Join(result[:2], currency.UnderscoreDelimiter) + currency.DashDelimiter + result[2]
	} else {
		instrumentID = pair.String()
	}
	return instrumentID
}

// optionPairToString formats an options pair as: SYMBOL-EXPIRE-STRIKE-TYPE
// SYMBOL may be a currency (BTC) or a pair (XRP_USDC)
// EXPIRE is DDMMMYY
// STRIKE may include a d for decimal point in linear options
// TYPE is Call or Put
func optionPairToString(pair currency.Pair) string {
	initialDelimiter := currency.DashDelimiter
	q := pair.Quote.String()
	if strings.HasPrefix(q, "USDC") && len(q) > 11 { // Linear option
		initialDelimiter = currency.UnderscoreDelimiter
		// Replace a capital D with d for decimal place in Strike price
		// Char 11 is either the hyphen before Strike price or first digit
		q = q[:11] + strings.Replace(q[11:], "D", "d", 1)
	}
	return pair.Base.String() + initialDelimiter + q
}

// optionComboPairToString formats an option combo pair to deribit request format
// e.g. XRP-USDC-CS-26SEP25-3D3_3D5 -> XRP_USDC-CS-26SEP25-3d3_3d5
func optionComboPairToString(pair currency.Pair) string {
	parts := strings.Split(pair.String(), "-")
	// Deribit uses lowercase 'd' to represent the decimal point
	lastIdx := len(parts) - 1
	parts[lastIdx] = strings.ReplaceAll(parts[lastIdx], "D", "d")
	// Leave unchanged when:
	// * length <= 3 (not enough info to be a combo needing underscore)
	// * length == 4 and second token is not USDC (original logic kept as-is)
	if len(parts) <= 3 || (len(parts) == 4 && parts[1] != "USDC") {
		return strings.Join(parts, "-")
	}
	// Otherwise insert underscore after base (covers:
	// * any length > 4
	// * length == 4 with USDC as second token)
	return parts[0] + "_" + strings.Join(parts[1:], "-")
}

// futureComboPairToString formats a future combo pair to deribit request format
// e.g. ETH-USDC-FS-28NOV25_PERP -> ETH_USDC-FS-28NOV25_PERP (linear)
// e.g. BTC-FS-28NOV25_PERP -> BTC-FS-28NOV25_PERP (inverse, unchanged)
func futureComboPairToString(pair currency.Pair) string {
	s := pair.String()
	before, after, found := strings.Cut(s, "-")
	if !found {
		return s
	}

	// Must have "USDC-" immediately after first dash (linear combo)
	if len(after) < 5 || after[:5] != "USDC-" {
		return s
	}

	// Need at least one more dash after "USDC-" to have 4+ parts
	if !strings.Contains(after[5:], "-") {
		return s
	}

	return before + "_" + after
}

// orderDetail converts an order as Deribit reports it
func (e *Exchange) orderDetail(o *Order) (*order.Detail, error) {
	a, pair, err := getAssetPairByInstrument(o.InstrumentName)
	if err != nil {
		return nil, err
	}
	side, err := order.StringToOrderSide(o.Direction)
	if err != nil {
		return nil, err
	}
	orderType, err := parseOrderType(o.OrderType)
	if err != nil {
		return nil, err
	}
	status, err := parseOrderStatus(o.OrderState, o.FilledAmount)
	if err != nil {
		return nil, err
	}
	tif, err := timeInForceFromString(o.TimeInForce, o.PostOnly)
	if err != nil {
		return nil, err
	}
	return &order.Detail{
		Exchange:             e.Name,
		OrderID:              o.OrderID,
		ClientOrderID:        o.Label,
		Pair:                 pair,
		AssetType:            a,
		Side:                 side,
		Type:                 orderType,
		Status:               status,
		TimeInForce:          tif,
		ReduceOnly:           o.ReduceOnly,
		Price:                o.Price.Value,
		TriggerPrice:         o.TriggerPrice,
		Amount:               o.Amount,
		ExecutedAmount:       o.FilledAmount,
		RemainingAmount:      o.Amount - o.FilledAmount,
		AverageExecutedPrice: o.AveragePrice,
		Date:                 o.CreationTimestamp.Time(),
		LastUpdated:          o.LastUpdateTimestamp.Time(),
	}, nil
}

// userTradeFill converts one of the account's trades to a fill
func (e *Exchange) userTradeFill(t *UserTrade) (*fill.Data, error) {
	a, pair, err := getAssetPairByInstrument(t.InstrumentName)
	if err != nil {
		return nil, err
	}
	side, err := order.StringToOrderSide(t.Direction)
	if err != nil {
		return nil, err
	}
	return &fill.Data{
		ID:            t.TradeID,
		TradeID:       t.TradeID,
		Timestamp:     t.Timestamp.Time(),
		Exchange:      e.Name,
		AssetType:     a,
		CurrencyPair:  pair,
		Side:          side,
		OrderID:       t.OrderID,
		ClientOrderID: t.Label,
		Price:         t.Price,
		Amount:        t.Amount,
	}, nil
}

// parseOrderType converts a Deribit order type; a market_limit order is a market order whose unfilled remainder rests
// as a limit order, and liquidation is only a trade's order type
func parseOrderType(orderType string) (order.Type, error) {
	switch orderType {
	case OrderTypeLimit:
		return order.Limit, nil
	case OrderTypeMarket, OrderTypeMarketLimit:
		return order.Market, nil
	case OrderTypeStopLimit:
		return order.StopLimit, nil
	case OrderTypeStopMarket:
		return order.StopMarket, nil
	case OrderTypeTakeLimit:
		return order.TakeProfit, nil
	case OrderTypeTakeMarket:
		return order.TakeProfitMarket, nil
	case OrderTypeTrailingStop:
		return order.TrailingStop, nil
	case "liquidation":
		return order.Liquidation, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %q", errUnsupportedOrderType, orderType)
	}
}

// parseOrderStatus converts an order state, reporting partly filled open and cancelled orders as such; an untriggered
// trigger order is pending, as Bybit's are, and a triggered one open
func parseOrderStatus(state string, filledAmount float64) (order.Status, error) {
	switch state {
	case "open", "triggered":
		if filledAmount > 0 {
			return order.PartiallyFilled, nil
		}
		return order.Open, nil
	case "untriggered":
		return order.Pending, nil
	case "filled":
		return order.Filled, nil
	case "cancelled":
		if filledAmount > 0 {
			return order.PartiallyFilledCancelled, nil
		}
		return order.Cancelled, nil
	case "rejected":
		return order.Rejected, nil
	case "archive":
		return order.Closed, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %q", errUnsupportedOrderStatus, state)
	}
}
