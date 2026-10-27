package bitstamp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
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
	perpetualMarketSuffix = "-PERP"
	// maxSubscriptionsPerConnection is the documented subscription limit of a connection; the server silently closes the
	// connection on the next subscription, so the websocket manager opens another connection for any more
	maxSubscriptionsPerConnection = 1024

	// The time filters of user transactions, from which spot order history is aggregated, only reach back 30 days
	maxHistoryAge = 30 * 24 * time.Hour
	// historyAgeBuffer keeps requested start times inside the retention window despite clock differences
	historyAgeBuffer = time.Minute
)

var (
	errCancelAllOrdersFailed   = errors.New("not all orders were cancelled; check order status to verify")
	errTriggerPriceRequired    = errors.New("trigger price is required")
	errTrackingValueRequired   = errors.New("trailing tracking value is required")
	errLeverageSettingNotFound = errors.New("leverage setting not found")
	errInvalidTransferID       = errors.New("transfer ID must be a number")
	errZeroTransferID          = errors.New("transfer ID must not be zero")
)

// SetDefaults sets default for Bitstamp
func (e *Exchange) SetDefaults() {
	e.Name = "Bitstamp"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	requestFmt := &currency.PairFormat{}
	configFmt := &currency.PairFormat{
		Uppercase: true,
		Delimiter: currency.ForwardSlashDelimiter,
	}
	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.PerpetualContract)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
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
				AccountBalance:        true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				FiatDeposit:           true,
				FiatWithdraw:          true,
				TradeFee:              true,
				FiatDepositFee:        true,
				FiatWithdrawalFee:     true,
				CryptoDepositFee:      true,
				CryptoWithdrawalFee:   true,
				FundingRateFetching:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:          true,
				OrderbookFetching:      true,
				FundingRateFetching:    true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				GetOrders:              true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals:  true,
				DateRanges: true,
			},
			MaximumOrderHistory: maxHistoryAge,
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.EightHour: true,
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
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
				),
				GlobalResultLimit: maxResultLimit,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(bitstampRateInterval, bitstampRequestRate, 1)))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      bitstampAPIURL,
		exchange.WebsocketSpot: bitstampWSURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup sets configuration values to bitstamp
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
		ExchangeConfig:                         exch,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		TradeFeed:                              e.Features.Enabled.TradeFeed,
		FillsFeed:                              e.Features.Enabled.FillsFeed,
		UseMultiConnectionManagement:           true,
		MaxWebsocketSubscriptionsPerConnection: maxSubscriptionsPerConnection,
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.WebsocketConnect,
		GenerateSubscriptions: e.generateSubscriptions,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		Handler:               e.wsHandleData,
		MessageFilter:         websocketConnection,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	marketType, err := marketTypeForAsset(a)
	if err != nil {
		return nil, err
	}
	markets, err := e.GetMarkets(ctx, "")
	if err != nil {
		return nil, err
	}
	return tradablePairs(markets, marketType)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	markets, err := e.GetMarkets(ctx, "")
	if err != nil {
		return err
	}
	for _, a := range e.GetAssetTypes(false) {
		marketType, err := marketTypeForAsset(a)
		if err != nil {
			return err
		}
		pairs, err := tradablePairs(markets, marketType)
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, a, false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// tradablePairs returns the pairs of the markets of a market type which have trading enabled
func tradablePairs(markets []MarketResponse, marketType string) (currency.Pairs, error) {
	pairs := make(currency.Pairs, 0, len(markets))
	for i := range markets {
		if markets[i].MarketType != marketType || !markets[i].TradingEnabled {
			continue
		}
		pair, err := currency.NewPairDelimiter(markets[i].Name, currency.ForwardSlashDelimiter)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	marketType, err := marketTypeForAsset(a)
	if err != nil {
		return err
	}
	markets, err := e.GetMarkets(ctx, "")
	if err != nil {
		return err
	}
	l := make([]limits.MinMaxLevel, 0, len(markets))
	for i := range markets {
		m := &markets[i]
		if m.MarketType != marketType || !m.TradingEnabled {
			continue
		}
		pair, err := currency.NewPairDelimiter(m.Name, currency.ForwardSlashDelimiter)
		if err != nil {
			return err
		}
		priceStep := math.Pow10(-int(m.CounterDecimals))
		if m.TickSize > 0 {
			priceStep = m.TickSize.Float64()
		}
		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
			PriceStepIncrementSize:  priceStep,
			AmountStepIncrementSize: math.Pow10(-int(m.BaseDecimals)),
			MinimumBaseAmount:       m.MinimumOrderAmount.Float64(),
			MaximumBaseAmount:       m.MaximumOrderAmount.Float64(),
			MinimumQuoteAmount:      m.MinimumOrderValue.Float64(),
			MaximumQuoteAmount:      m.MaximumOrderValue.Float64(),
		})
	}
	return limits.Load(l)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	marketType, err := marketTypeForAsset(a)
	if err != nil {
		return err
	}
	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	tickers, err := e.GetTickers(ctx)
	if err != nil {
		return err
	}
	var errs error
	for i := range tickers {
		if tickers[i].MarketType != marketType {
			continue
		}
		pair, err := currency.NewPairDelimiter(tickers[i].Market, currency.ForwardSlashDelimiter)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if !pairs.Contains(pair, true) {
			continue
		}
		fPair, err := e.FormatExchangeCurrency(pair, a)
		if err != nil {
			return err
		}
		if err := ticker.ProcessTicker(e.tickerPrice(&tickers[i], fPair, a)); err != nil {
			// A locked market must not prevent later enabled markets from refreshing
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if _, err := marketTypeForAsset(a); err != nil {
		return nil, err
	}
	fPair, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	tick, err := e.GetTicker(ctx, fPair)
	if err != nil {
		return nil, err
	}
	if err := ticker.ProcessTicker(e.tickerPrice(tick, fPair, a)); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, fPair, a)
}

func (e *Exchange) tickerPrice(t *TickerResponse, pair currency.Pair, a asset.Item) *ticker.Price {
	return &ticker.Price{
		Last:                       t.Last.Float64(),
		VolumeWeightedAveragePrice: t.VolumeWeightedAveragePrice.Float64(),
		High:                       t.High.Float64(),
		Low:                        t.Low.Float64(),
		Bid:                        t.BestBid.Float64(),
		Ask:                        t.BestAsk.Float64(),
		BaseVolume:                 t.Volume.Float64(),
		Open:                       t.Open.Float64(),
		Open24Hour:                 t.Open24Hour.Float64(),
		PercentChange24Hour:        t.PercentChange24Hour.Float64(),
		MarkPrice:                  t.MarkPrice.Float64(),
		IndexPrice:                 t.IndexPrice.Float64(),
		OpenInterest:               t.OpenInterest.Float64(),
		OpenInterestValue:          t.OpenInterestValue.Float64(),
		Pair:                       pair,
		ExchangeName:               e.Name,
		AssetType:                  a,
		LastUpdated:                t.Timestamp.Time(),
	}
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if err := common.NilGuard(feeBuilder); err != nil {
		return 0, err
	}
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	fPair, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	ob, err := e.GetOrderbook(ctx, fPair, OrderbookGroupedByPrice)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
		LastUpdated:       ob.Microtimestamp.Time(),
		Bids:              orderbookLevels(ob.Bids),
		Asks:              orderbookLevels(ob.Asks),
	}
	filterOrderbookZeroBidPrice(book)
	if err := book.Process(); err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, p, assetType)
}

func orderbookLevels(levels []OrderbookLevel) orderbook.Levels {
	l := make(orderbook.Levels, len(levels))
	for i := range levels {
		l[i] = orderbook.Level{Price: levels[i].Price, Amount: levels[i].Amount}
	}
	return l
}

// filterOrderbookZeroBidPrice removes the trailing zero priced bid Bitstamp includes in some orderbooks
func filterOrderbookZeroBidPrice(ob *orderbook.Book) {
	if len(ob.Bids) == 0 || ob.Bids[len(ob.Bids)-1].Price != 0 {
		return
	}
	ob.Bids = ob.Bids[0 : len(ob.Bids)-1]
}

// UpdateAccountBalances retrieves currency balances
// Perpetual contract balances are the collateral balances of the derivatives trade account
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	switch assetType {
	case asset.Spot:
		balances, err := e.GetAccountBalances(ctx)
		if err != nil {
			return nil, err
		}
		for i := range balances {
			subAccts[0].Balances.Set(balances[i].Currency.Upper(), accounts.Balance{
				Total: balances[i].Total.Float64(),
				Hold:  balances[i].Reserved.Float64(),
				Free:  balances[i].Available.Float64(),
			})
		}
	case asset.PerpetualContract:
		info, err := e.GetMarginInfo(ctx)
		if err != nil {
			return nil, err
		}
		for i := range info.Assets {
			subAccts[0].Balances.Set(info.Assets[i].Asset.Upper(), accounts.Balance{
				Total: info.Assets[i].TotalAmount.Float64(),
				Hold:  info.Assets[i].Reserved.Float64(),
				Free:  info.Assets[i].Available.Float64(),
			})
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, assetType)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns the main account's cryptocurrency deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	txs, err := e.allCryptoTransactions(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(txs.Deposits)+len(txs.Withdrawals))
	for i := range txs.Deposits {
		d := &txs.Deposits[i]
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:      e.Name,
			Status:            d.Status,
			TransferID:        strconv.FormatUint(d.ID, 10),
			Timestamp:         d.DateTime.Time(),
			Currency:          d.Currency.Upper().String(),
			Amount:            d.Amount.Float64(),
			TransferType:      "deposit",
			CryptoToAddress:   d.DestinationAddress,
			CryptoFromAddress: d.OriginatorAddress,
			CryptoTxID:        d.TxID,
			CryptoChain:       d.Network,
		})
	}
	for i := range txs.Withdrawals {
		w := &txs.Withdrawals[i]
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Timestamp:       w.DateTime.Time(),
			Currency:        w.Currency.Upper().String(),
			Amount:          w.Amount.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: w.DestinationAddress,
			CryptoTxID:      w.TxID,
			CryptoChain:     w.Network,
		})
	}
	return resp, nil
}

// allCryptoTransactions returns every page of the main account's cryptocurrency deposits and withdrawals
// The limit applies to each list, so pages are fetched until no list fills a page
func (e *Exchange) allCryptoTransactions(ctx context.Context) (*CryptoTransactionsResponse, error) {
	all := new(CryptoTransactionsResponse)
	for offset := uint64(0); ; offset += maxResultLimit {
		if offset > maxResultOffset {
			return nil, fmt.Errorf("%w: more than %d transactions", errInvalidOffset, maxResultOffset)
		}
		page, err := e.GetCryptoTransactions(ctx, &CryptoTransactionsRequest{Limit: maxResultLimit, Offset: offset})
		if err != nil {
			return nil, err
		}
		all.Deposits = append(all.Deposits, page.Deposits...)
		all.Withdrawals = append(all.Withdrawals, page.Withdrawals...)
		all.RippleIOUTransactions = append(all.RippleIOUTransactions, page.RippleIOUTransactions...)
		if len(page.Deposits) < maxResultLimit && len(page.Withdrawals) < maxResultLimit && len(page.RippleIOUTransactions) < maxResultLimit {
			return all, nil
		}
	}
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	var withdrawals []WithdrawalRequestResponse
	seen := make(map[uint64]bool)
	for offset := uint64(0); ; offset += maxResultLimit {
		if offset > maxResultOffset {
			return nil, fmt.Errorf("%w: more than %d withdrawal requests", errInvalidOffset, maxResultOffset)
		}
		page, err := e.GetWithdrawalRequests(ctx, &WithdrawalRequestsRequest{
			TimeDelta: maxWithdrawalRequestsTimeDelta,
			Limit:     maxResultLimit,
			Offset:    offset,
		})
		if err != nil {
			return nil, err
		}
		var added bool
		for i := range page {
			if !seen[page[i].ID] {
				seen[page[i].ID] = true
				withdrawals = append(withdrawals, page[i])
				added = true
			}
		}
		// A page of only known requests means the offset was not applied, so further pages would repeat it
		if len(page) < maxResultLimit || !added {
			break
		}
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(withdrawals))
	for i := range withdrawals {
		w := &withdrawals[i]
		if !c.IsEmpty() && !c.Equal(w.Currency) {
			continue
		}
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          withdrawalStatus(w.Status),
			TransferID:      strconv.FormatUint(w.ID, 10),
			Timestamp:       w.DateTime.Time(),
			Currency:        w.Currency.Upper().String(),
			Amount:          w.Amount.Float64(),
			TransferType:    withdrawalType(w.Type),
			CryptoToAddress: w.Address,
			CryptoTxID:      w.TransactionID,
			CryptoChain:     w.Network,
		})
	}
	return resp, nil
}

// withdrawalStatus returns the description of a withdrawal request status
func withdrawalStatus(status WithdrawalStatus) string {
	switch status {
	case WithdrawalStatusOpen:
		return "Open"
	case WithdrawalStatusInProcess:
		return "In process"
	case WithdrawalStatusFinished:
		return "Finished"
	case WithdrawalStatusCancelled:
		return "Cancelled"
	case WithdrawalStatusFailed:
		return "Failed"
	case WithdrawalStatusReversed:
		return "Reversed"
	default:
		return strconv.FormatUint(uint64(status), 10)
	}
}

// withdrawalType returns the description of a withdrawal request type; bank transfers have dedicated types and every
// other type identifies the cryptocurrency withdrawn
func withdrawalType(t WithdrawalType) string {
	switch t {
	case WithdrawalTypeSEPA:
		return "SEPA"
	case WithdrawalTypeWire:
		return "Wire transfer"
	default:
		return "Cryptocurrency"
	}
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if _, err := marketTypeForAsset(assetType); err != nil {
		return nil, err
	}
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tradeData, err := e.GetTransactions(ctx, p, "")
	if err != nil {
		return nil, err
	}

	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		resp[i] = trade.Data{
			Exchange:     e.Name,
			TID:          strconv.FormatUint(tradeData[i].TradeID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         tradeData[i].Side.Side(),
			Price:        tradeData[i].Price.Float64(),
			Amount:       tradeData[i].Amount.Float64(),
			Timestamp:    tradeData[i].Date.Time(),
		}
	}

	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}

	trade.SortByDate(resp)
	return resp, nil
}

// GetHistoricTrades is not supported as the API only returns recent trades
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
// Market orders with a QuoteAmount are placed as instant orders, which do not support reduce only orders. Perpetual
// contract orders use cross margin unless isolated margin is requested, and are sent with a leverage, as the API
// requires one on limit, market and instant buy orders; when Leverage is not set, the current leverage setting is
// fetched first, which costs a request before the order is placed.
// Trailing stop orders follow the market by a TrackingValue percentage once the TriggerPrice is reached
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	if _, err := marketTypeForAsset(s.AssetType); err != nil {
		return nil, err
	}
	if err := validateOrderSubmission(s); err != nil {
		return nil, err
	}
	fPair, err := e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	var p derivativesOrderParams
	if s.AssetType == asset.PerpetualContract {
		if p.MarginMode, err = formatMarginMode(s.MarginType); err != nil {
			return nil, err
		}
		p.Leverage = s.Leverage
		if p.Leverage == 0 {
			if p.Leverage, err = e.GetLeverage(ctx, s.AssetType, fPair, s.MarginType, s.Side); err != nil {
				return nil, err
			}
		}
		p.ReduceOnly = s.ReduceOnly
		p.Trigger = formatTrigger(s.TriggerPriceType)
	}
	p.ClientOrderID = s.ClientOrderID

	var resp *OrderResponse
	switch {
	case s.Type == order.Limit:
		req := &LimitOrderRequest{
			Pair:          fPair,
			Side:          s.Side,
			Amount:        s.Amount,
			Price:         s.Price,
			ClientOrderID: p.ClientOrderID,
			MarginMode:    p.MarginMode,
			Leverage:      p.Leverage,
			ReduceOnly:    p.ReduceOnly,
		}
		if err := setLimitOrderTimeInForce(req, s.TimeInForce, s.EndTime); err != nil {
			return nil, err
		}
		resp, err = e.PlaceLimitOrder(ctx, req)
	case s.Type == order.Market && s.QuoteAmount > 0:
		resp, err = e.PlaceInstantOrder(ctx, &InstantOrderRequest{
			Pair:            fPair,
			Side:            s.Side,
			Amount:          s.QuoteAmount,
			AmountInCounter: true,
			ClientOrderID:   p.ClientOrderID,
			MarginMode:      p.MarginMode,
			Leverage:        p.Leverage,
			ReduceOnly:      p.ReduceOnly,
		})
	case s.Type == order.Market:
		resp, err = e.PlaceMarketOrder(ctx, &MarketOrderRequest{
			Pair:          fPair,
			Side:          s.Side,
			Amount:        s.Amount,
			ClientOrderID: p.ClientOrderID,
			MarginMode:    p.MarginMode,
			Leverage:      p.Leverage,
			ReduceOnly:    p.ReduceOnly,
		})
	case s.AssetType == asset.PerpetualContract:
		resp, err = e.submitConditionalOrder(ctx, s, fPair, &p)
	default:
		return nil, fmt.Errorf("%w %s for %s", order.ErrUnsupportedOrderType, s.Type, s.AssetType)
	}
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(strconv.FormatUint(resp.ID, 10))
}

// validateOrderSubmission checks the order types and parameters the API supports before any request is made
func validateOrderSubmission(s *order.Submit) error {
	switch s.Type {
	case order.Limit, order.Market:
	case order.Stop, order.StopMarket, order.TakeProfit, order.TakeProfitMarket, order.StopLimit, order.TakeProfit | order.Limit, order.TrailingStop, order.TrailingStopLimit:
		if s.AssetType != asset.PerpetualContract {
			return fmt.Errorf("%w %s for %s", order.ErrUnsupportedOrderType, s.Type, s.AssetType)
		}
		if s.TriggerPrice <= 0 {
			return fmt.Errorf("%w for %s orders", errTriggerPriceRequired, s.Type)
		}
		if s.Type == order.TrailingStop || s.Type == order.TrailingStopLimit {
			if s.TrackingMode != order.Percentage {
				return fmt.Errorf("%w: %s", order.ErrUnknownTrackingMode, s.TrackingMode)
			}
			if s.TrackingValue <= 0 {
				return errTrackingValueRequired
			}
		}
	default:
		return fmt.Errorf("%w %s for %s", order.ErrUnsupportedOrderType, s.Type, s.AssetType)
	}
	// Quote amounts are placed as instant orders, which only support reduce only sells with a base amount
	if s.ReduceOnly && s.Type == order.Market && s.QuoteAmount > 0 {
		return errReduceOnlyUnsupported
	}
	// Only market orders can be sized by a quote amount, as instant orders; every other order needs a base amount
	if s.Amount <= 0 && s.QuoteAmount > 0 && s.Type != order.Market {
		return fmt.Errorf("%w: %s orders require a base amount", order.ErrAmountIsInvalid, s.Type)
	}
	// Market, stop, take profit and trailing stop orders execute as market orders, which take no time in force flags
	if !s.Type.Is(order.Limit) && s.TimeInForce != order.UnknownTIF && s.TimeInForce != order.GoodTillCancel && s.TimeInForce != order.ImmediateOrCancel {
		return fmt.Errorf("%w %s for %s orders", order.ErrUnsupportedTimeInForce, s.TimeInForce, s.Type)
	}
	return nil
}

// submitConditionalOrder places a stop loss, take profit or trailing stop order on a derivatives market
func (e *Exchange) submitConditionalOrder(ctx context.Context, s *order.Submit, fPair currency.Pair, p *derivativesOrderParams) (*OrderResponse, error) {
	switch s.Type {
	case order.Stop, order.StopMarket, order.TakeProfit, order.TakeProfitMarket:
		subtype := OrderSubtypeStopLoss
		if s.Type.Is(order.TakeProfit) {
			subtype = OrderSubtypeTakeProfit
		}
		return e.PlaceMarketOrder(ctx, &MarketOrderRequest{
			Pair:          fPair,
			Side:          s.Side,
			Amount:        s.Amount,
			ClientOrderID: p.ClientOrderID,
			Subtype:       subtype,
			MarginMode:    p.MarginMode,
			Leverage:      p.Leverage,
			StopPrice:     s.TriggerPrice,
			Trigger:       p.Trigger,
			ReduceOnly:    p.ReduceOnly,
		})
	case order.StopLimit, order.TakeProfit | order.Limit:
		subtype := OrderSubtypeStopLossLimit
		if s.Type.Is(order.TakeProfit) {
			subtype = OrderSubtypeTakeProfitLimit
		}
		req := &LimitOrderRequest{
			Pair:          fPair,
			Side:          s.Side,
			Amount:        s.Amount,
			Price:         s.Price,
			ClientOrderID: p.ClientOrderID,
			Subtype:       subtype,
			MarginMode:    p.MarginMode,
			Leverage:      p.Leverage,
			StopPrice:     s.TriggerPrice,
			Trigger:       p.Trigger,
			ReduceOnly:    p.ReduceOnly,
		}
		if err := setConditionalOrderTimeInForce(req, s.TimeInForce); err != nil {
			return nil, err
		}
		return e.PlaceLimitOrder(ctx, req)
	case order.TrailingStop:
		return e.PlaceMarketOrder(ctx, &MarketOrderRequest{
			Pair:            fPair,
			Side:            s.Side,
			Amount:          s.Amount,
			ClientOrderID:   p.ClientOrderID,
			Subtype:         OrderSubtypeTrailingStopLoss,
			MarginMode:      p.MarginMode,
			Leverage:        p.Leverage,
			Trigger:         p.Trigger,
			ActivationPrice: s.TriggerPrice,
			TrailingDelta:   trailingDeltaBasisPoints(s.TrackingValue),
			ReduceOnly:      p.ReduceOnly,
		})
	case order.TrailingStopLimit:
		req := &LimitOrderRequest{
			Pair:            fPair,
			Side:            s.Side,
			Amount:          s.Amount,
			Price:           s.Price,
			ClientOrderID:   p.ClientOrderID,
			Subtype:         OrderSubtypeTrailingStopLossLimit,
			MarginMode:      p.MarginMode,
			Leverage:        p.Leverage,
			Trigger:         p.Trigger,
			ActivationPrice: s.TriggerPrice,
			TrailingDelta:   trailingDeltaBasisPoints(s.TrackingValue),
			ReduceOnly:      p.ReduceOnly,
		}
		if err := setConditionalOrderTimeInForce(req, s.TimeInForce); err != nil {
			return nil, err
		}
		return e.PlaceLimitOrder(ctx, req)
	default:
		return nil, fmt.Errorf("%w %s for %s", order.ErrUnsupportedOrderType, s.Type, s.AssetType)
	}
}

// trailingDeltaBasisPoints converts a trailing percentage to the basis points the API expects, e.g. 5% to 500
func trailingDeltaBasisPoints(percentage float64) uint64 {
	return uint64(math.Round(percentage * 100))
}

// setConditionalOrderTimeInForce sets the limit order flags of a stop loss, take profit or trailing order, which only
// support good till cancelled, immediate or cancel and fill or kill
func setConditionalOrderTimeInForce(req *LimitOrderRequest, tif order.TimeInForce) error {
	switch tif {
	case order.UnknownTIF, order.GoodTillCancel:
	case order.ImmediateOrCancel:
		req.ImmediateOrCancel = true
	case order.FillOrKill:
		req.FillOrKill = true
	default:
		return fmt.Errorf("%w %s for conditional orders", order.ErrUnsupportedTimeInForce, tif)
	}
	return nil
}

// setLimitOrderTimeInForce sets the limit order flag which matches a time in force
// The API accepts one flag per order, so a post only order cannot also be immediate or cancel, fill or kill, daily or
// good till date
func setLimitOrderTimeInForce(req *LimitOrderRequest, tif order.TimeInForce, endTime time.Time) error {
	if tif.Is(order.GoodTillCrossing) || tif.Is(order.StopOrReduce) {
		return fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, tif)
	}
	var flags int
	for _, t := range []order.TimeInForce{order.ImmediateOrCancel, order.FillOrKill, order.PostOnly, order.GoodTillDay, order.GoodTillTime} {
		if tif.Is(t) {
			flags++
		}
	}
	if flags > 1 {
		return fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, tif)
	}
	req.ImmediateOrCancel = tif.Is(order.ImmediateOrCancel)
	req.FillOrKill = tif.Is(order.FillOrKill)
	req.MakerOrCancel = tif.Is(order.PostOnly)
	req.DailyOrder = tif.Is(order.GoodTillDay)
	if tif.Is(order.GoodTillTime) {
		if endTime.IsZero() {
			return fmt.Errorf("%w: end time is required for %s orders", common.ErrDateUnset, tif)
		}
		req.GoodTillDate = true
		req.ExpireTime = endTime
	}
	return nil
}

// ModifyOrder is not supported, as Bitstamp replaces a modified order with a new order under a new order ID, which the
// order manager cannot follow, and takes the replacement's own amount, which executions in flight can push past the
// order's new total
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// executedAmount returns the base currency amount executed by an order's transactions, summed as decimals so that it
// carries no float residue. A transaction without a base amount is an error rather than nothing executed, as a
// modification would otherwise place its amount again
func executedAmount(transactions []OrderTransaction, base currency.Code) (decimal.Decimal, error) {
	executed := decimal.Zero
	for i := range transactions {
		amount, ok := transactions[i].Amounts[base.Upper()]
		if !ok {
			return decimal.Zero, fmt.Errorf("%w %d: no %s amount", errMalformedTransaction, transactions[i].TradeID, base)
		}
		value, err := gctmath.DecimalFromFloat(math.Abs(amount))
		if err != nil {
			return decimal.Zero, fmt.Errorf("%w %d: %w", errMalformedTransaction, transactions[i].TradeID, err)
		}
		executed = executed.Add(value)
	}
	return executed, nil
}

// CancelOrder cancels an order by its order ID, or by its client order ID when the order ID is not set
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(); err != nil {
		return err
	}
	req := &CancelOrderRequest{ClientOrderID: o.ClientOrderID}
	if o.OrderID != "" {
		orderID, err := strconv.ParseUint(o.OrderID, 10, 64)
		if err != nil {
			return fmt.Errorf("%w %q: %w", order.ErrOrderIDNotSet, o.OrderID, err)
		}
		// The API does not define which ID takes precedence when both are sent
		req = &CancelOrderRequest{OrderID: orderID}
	}
	_, err := e.CancelExistingOrder(ctx, req)
	return err
}

// CancelBatchOrders is not supported as the API has no batch cancel endpoint
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders for a currency pair, or for every market of the asset type when no pair is set
// Status holds the cancelled orders keyed by order ID. Without a pair, the markets cancelled are the enabled pairs and
// the markets of the open orders, since the open orders are cached for 10 seconds and may not list a new order
func (e *Exchange) CancelAllOrders(ctx context.Context, req *order.Cancel) (order.CancelAllResponse, error) {
	if err := req.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	if _, err := marketTypeForAsset(req.AssetType); err != nil {
		return order.CancelAllResponse{}, err
	}
	var markets currency.Pairs
	if !req.Pair.IsEmpty() {
		fPair, err := e.FormatExchangeCurrency(req.Pair, req.AssetType)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		markets = currency.Pairs{fPair}
	} else {
		// Cancelling all orders without a market would also cancel the orders of every other asset type
		enabled, err := e.GetEnabledPairs(req.AssetType)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		for _, p := range enabled {
			fPair, err := e.FormatExchangeCurrency(p, req.AssetType)
			if err != nil {
				return order.CancelAllResponse{}, err
			}
			markets = append(markets, fPair)
		}
		openOrders, err := e.GetOpenOrders(ctx, currency.EMPTYPAIR)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		for i := range openOrders {
			pair, a, err := marketPairAsset(openOrders[i].Market)
			if err != nil {
				return order.CancelAllResponse{}, err
			}
			if a == req.AssetType && !markets.Contains(pair, true) {
				markets = append(markets, pair)
			}
		}
	}

	resp := order.CancelAllResponse{Status: make(map[string]string)}
	var errs error
	for _, market := range markets {
		cancelled, err := e.CancelAllExistingOrders(ctx, market)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s: %w", market, err))
			continue
		}
		for i := range cancelled.Cancelled {
			resp.Status[strconv.FormatUint(cancelled.Cancelled[i].ID, 10)] = order.Cancelled.String()
		}
		if !cancelled.Success {
			errs = common.AppendError(errs, fmt.Errorf("%s: %w", market, errCancelAllOrdersFailed))
		}
	}
	return resp, errs
}

// GetOrderInfo returns order information based on order ID
// Orders closed more than 30 days ago are not found
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	id, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", order.ErrOrderIDNotSet, orderID, err)
	}
	status, err := e.GetOrderStatus(ctx, &OrderStatusRequest{OrderID: id})
	if err != nil {
		return nil, err
	}
	pair, a, err := marketPairAsset(status.Market)
	if err != nil {
		return nil, err
	}
	executed, err := executedAmount(status.Transactions, pair.Base)
	if err != nil {
		return nil, err
	}
	remaining, err := gctmath.DecimalFromFloat(status.AmountRemaining.Float64())
	if err != nil {
		return nil, fmt.Errorf("%w: amount remaining: %w", order.ErrAmountIsInvalid, err)
	}

	// Only derivatives orders have a subtype, so the type of a spot order is unknown
	orderType := order.UnknownType
	if status.Subtype != "" {
		orderType = orderTypeFromSubtype(status.Subtype)
	}
	feeAsset := settlementCurrency(pair, a)
	d := &order.Detail{
		Exchange:        e.Name,
		OrderID:         strconv.FormatUint(status.ID, 10),
		ClientOrderID:   status.ClientOrderID,
		Type:            orderType,
		Side:            status.Side.Side(),
		AssetType:       a,
		Pair:            pair,
		Date:            status.DateTime.Time(),
		LastUpdated:     status.DateTime.Time(),
		RemainingAmount: status.AmountRemaining.Float64(),
		Leverage:        status.Leverage.Float64(),
		MarginType:      marginTypeFromMode(status.MarginMode),
		TriggerPrice:    triggerPrice(status.StopPrice.Float64(), status.ActivationPrice.Float64()),
		FeeAsset:        feeAsset,
		CostAsset:       feeAsset,
		Trades:          make([]order.TradeHistory, len(status.Transactions)),
	}
	var totals tradeTotals
	for i := range status.Transactions {
		tx := &status.Transactions[i]
		amount := math.Abs(tx.Amounts[pair.Base.Upper()])
		d.Trades[i] = order.TradeHistory{
			TID:       strconv.FormatUint(tx.TradeID, 10),
			Price:     tx.Price,
			Amount:    amount,
			Fee:       tx.Fee,
			Exchange:  e.Name,
			Side:      d.Side,
			Timestamp: tx.DateTime,
			FeeAsset:  feeAsset.String(),
		}
		if err := totals.add(amount, tx.Price, tx.Fee); err != nil {
			return nil, fmt.Errorf("%w %d: %w", errMalformedTransaction, tx.TradeID, err)
		}
		if tx.DateTime.After(d.LastUpdated) {
			d.LastUpdated = tx.DateTime
		}
	}
	d.ExecutedAmount = executed.InexactFloat64()
	d.Amount = executed.Add(remaining).InexactFloat64()
	d.Cost = totals.cost.InexactFloat64()
	d.Fee = totals.fee.InexactFloat64()
	d.AverageExecutedPrice = totals.averagePrice()
	d.Status = orderStatus(status.Status, d.ExecutedAmount > 0)
	return d, nil
}

// triggerPrice returns the price which triggers an order: the stop price of a stop order, or the activation price of a
// trailing order, which SubmitOrder sends from the trigger price
func triggerPrice(stopPrice, activationPrice float64) float64 {
	if stopPrice != 0 {
		return stopPrice
	}
	return activationPrice
}

// orderStatus returns the order status of an order status description
func orderStatus(status string, partiallyExecuted bool) order.Status {
	switch status {
	case OrderStatusOpen:
		if partiallyExecuted {
			return order.PartiallyFilled
		}
		return order.Open
	case OrderStatusFinished:
		return order.Filled
	case OrderStatusCancelled:
		if partiallyExecuted {
			return order.PartiallyCancelled
		}
		return order.Cancelled
	case OrderStatusExpired:
		return order.Expired
	default:
		return order.UnknownStatus
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	addr, err := e.GetCryptoDepositAddress(ctx, cryptocurrency, chain)
	if err != nil {
		return nil, err
	}
	tag := addr.MemoID
	switch {
	case addr.DestinationTag != 0:
		tag = strconv.FormatUint(addr.DestinationTag, 10)
	case addr.TransferID != 0:
		tag = strconv.FormatUint(addr.TransferID, 10)
	}
	return &deposit.Address{
		Address: addr.Address,
		Tag:     tag,
		Chain:   chain,
	}, nil
}

// GetAvailableTransferChains returns the networks which support deposits or withdrawals of a currency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	currencies, err := e.GetCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	for i := range currencies {
		if !currencies[i].Currency.Equal(cryptocurrency) {
			continue
		}
		chains := make([]string, 0, len(currencies[i].Networks))
		for j := range currencies[i].Networks {
			n := &currencies[i].Networks[j]
			if n.DepositEnabled || n.WithdrawalEnabled {
				chains = append(chains, n.Network)
			}
		}
		return chains, nil
	}
	return nil, fmt.Errorf("%w: %s", currency.ErrCurrencyNotFound, cryptocurrency)
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
// A withdrawal to an address in the account holder's name is declared with Travel.IsSelf; withdrawals to third parties
// which need Travel Rule details are made with CryptoWithdrawal
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	req := &CryptoWithdrawalRequest{
		Currency: withdrawRequest.Currency,
		Network:  withdrawRequest.Crypto.Chain,
		Amount:   withdrawRequest.Amount,
		Address:  withdrawRequest.Crypto.Address,
	}
	if withdrawRequest.Travel.IsSelf {
		req.BeneficiaryThirdParty = new(false)
	}
	if tag := withdrawRequest.Crypto.AddressTag; tag != "" {
		// The tag field depends on the network: XRP Ledger addresses take destination tags, Casper addresses take
		// numeric transfer IDs and other tagged networks such as Stellar take memo IDs
		switch network := strings.ToLower(withdrawRequest.Crypto.Chain); {
		case network == "xrpl", network == "" && withdrawRequest.Currency.Equal(currency.XRP):
			req.DestinationTag = tag
		case network == "casper", network == "" && withdrawRequest.Currency.Equal(currency.CSPR):
			transferID, err := strconv.ParseUint(tag, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w %q: %w", errInvalidTransferID, tag, err)
			}
			// CryptoWithdrawal leaves an unset transfer ID out, so a zero would send the withdrawal without the ID given
			if transferID == 0 {
				return nil, fmt.Errorf("%w %q", errZeroTransferID, tag)
			}
			req.TransferID = transferID
		default:
			req.MemoID = tag
		}
	}
	resp, err := e.CryptoWithdrawal(ctx, req)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: strconv.FormatUint(resp.ID, 10),
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a SEPA withdrawal is submitted
// Bitstamp asks for the account holder's address, which banking accounts do not hold, so the bank's address is sent
func (e *Exchange) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.OpenBankWithdrawal(ctx, &BankWithdrawalRequest{
		Amount:          withdrawRequest.Amount,
		AccountCurrency: withdrawRequest.Currency,
		Name:            withdrawRequest.Fiat.Bank.AccountName,
		IBAN:            withdrawRequest.Fiat.Bank.IBAN,
		BIC:             withdrawRequest.Fiat.Bank.SWIFTCode,
		Address:         withdrawRequest.Fiat.Bank.BankAddress,
		PostalCode:      withdrawRequest.Fiat.Bank.BankPostalCode,
		City:            withdrawRequest.Fiat.Bank.BankPostalCity,
		Country:         withdrawRequest.Fiat.Bank.BankCountry,
		Comment:         withdrawRequest.Description,
		Type:            BankWithdrawalTypeSEPA,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: strconv.FormatUint(resp.WithdrawalID, 10),
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when an international wire withdrawal is submitted
// Bitstamp asks for the account holder's address, which banking accounts do not hold, so the bank's address is sent
func (e *Exchange) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.OpenBankWithdrawal(ctx, &BankWithdrawalRequest{
		Amount:          withdrawRequest.Amount,
		AccountCurrency: withdrawRequest.Currency,
		Name:            withdrawRequest.Fiat.Bank.AccountName,
		IBAN:            withdrawRequest.Fiat.Bank.IBAN,
		BIC:             withdrawRequest.Fiat.Bank.SWIFTCode,
		Address:         withdrawRequest.Fiat.Bank.BankAddress,
		PostalCode:      withdrawRequest.Fiat.Bank.BankPostalCode,
		City:            withdrawRequest.Fiat.Bank.BankPostalCity,
		Country:         withdrawRequest.Fiat.Bank.BankCountry,
		Comment:         withdrawRequest.Description,
		Type:            BankWithdrawalTypeInternational,
		BankName:        withdrawRequest.Fiat.Bank.BankName,
		BankAddress:     withdrawRequest.Fiat.Bank.BankAddress,
		BankPostalCode:  withdrawRequest.Fiat.Bank.BankPostalCode,
		BankCity:        withdrawRequest.Fiat.Bank.BankPostalCity,
		BankCountry:     withdrawRequest.Fiat.Bank.BankCountry,
		Currency:        currency.NewCode(withdrawRequest.Fiat.WireCurrency),
		// The intermediary bank is identified by its routing number or BIC
		IntermediaryRoutingNumberOrBIC: withdrawRequest.Fiat.IntermediarySwiftCode,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: strconv.FormatUint(resp.WithdrawalID, 10),
	}, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if _, err := marketTypeForAsset(req.AssetType); err != nil {
		return nil, err
	}
	var pair currency.Pair
	if len(req.Pairs) == 1 {
		var err error
		if pair, err = e.FormatExchangeCurrency(req.Pairs[0], req.AssetType); err != nil {
			return nil, err
		}
	}
	openOrders, err := e.GetOpenOrders(ctx, pair)
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, 0, len(openOrders))
	for i := range openOrders {
		o := &openOrders[i]
		p, a, err := marketPairAsset(o.Market)
		if err != nil {
			return nil, err
		}
		if a != req.AssetType {
			continue
		}
		amount := o.AmountAtCreate.Float64()
		if amount == 0 {
			amount = o.RemainingAmount.Float64()
		}
		executedDecimal, err := gctmath.DecimalDifferenceOfFloats(amount, o.RemainingAmount.Float64())
		if err != nil {
			return nil, fmt.Errorf("%w for order %d: %w", order.ErrAmountIsInvalid, o.ID, err)
		}
		executed := executedDecimal.InexactFloat64()
		status := order.Open
		if executed > 0 {
			status = order.PartiallyFilled
		}
		orders = append(orders, order.Detail{
			Exchange:        e.Name,
			OrderID:         strconv.FormatUint(o.ID, 10),
			ClientOrderID:   o.ClientOrderID,
			Type:            orderTypeFromSubtype(o.Subtype),
			Side:            o.Side.Side(),
			Status:          status,
			AssetType:       a,
			Pair:            p,
			Date:            o.DateTime.Time(),
			Price:           o.Price.Float64(),
			Amount:          amount,
			ExecutedAmount:  executed,
			RemainingAmount: o.RemainingAmount.Float64(),
			Leverage:        o.Leverage.Float64(),
			MarginType:      marginTypeFromMode(o.MarginMode),
			TriggerPrice:    triggerPrice(o.StopPrice.Float64(), o.ActivationPrice.Float64()),
			ReduceOnly:      o.ReduceOnly,
		})
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves executed orders, aggregated from their trades within the request's time range
// A spot start more than 30 days ago is moved to 30 days ago, as far back as the transactions' time filters reach
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	var err error
	switch req.AssetType {
	case asset.Spot:
		orders, err = e.spotOrderHistory(ctx, req)
	case asset.PerpetualContract:
		pairs := req.Pairs
		if len(pairs) == 0 {
			if pairs, err = e.GetEnabledPairs(req.AssetType); err != nil {
				return nil, err
			}
		}
		orders, err = e.derivativesOrderHistory(ctx, pairs, req.StartTime, req.EndTime)
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, req.AssetType)
	}
	if err != nil {
		return nil, err
	}
	return req.Filter(e.Name, withinTimeRange(orders, req.StartTime, req.EndTime)), nil
}

// withinTimeRange returns the orders which were last updated within a time range, either bound of which may be zero
func withinTimeRange(orders []order.Detail, start, end time.Time) []order.Detail {
	filtered := orders[:0]
	for i := range orders {
		if (!start.IsZero() && orders[i].LastUpdated.Before(start)) || (!end.IsZero() && orders[i].Date.After(end)) {
			continue
		}
		filtered = append(filtered, orders[i])
	}
	return filtered
}

// spotOrderHistory returns spot orders aggregated from the account's market trade transactions
// The transactions' time filters only reach back 30 days, so ranges which end before then return no orders
func (e *Exchange) spotOrderHistory(ctx context.Context, req *order.MultiOrderRequest) ([]order.Detail, error) {
	earliest := time.Now().Add(-maxHistoryAge + historyAgeBuffer)
	if !req.EndTime.IsZero() && req.EndTime.Before(earliest) {
		return nil, nil
	}
	params := &UserTransactionsRequest{
		Limit: maxResultLimit,
		Sort:  SortAscending,
		Until: req.EndTime,
	}
	if !req.StartTime.IsZero() {
		params.Since = req.StartTime
		if params.Since.Before(earliest) {
			params.Since = earliest
		}
	}
	if len(req.Pairs) == 1 {
		fPair, err := e.FormatExchangeCurrency(req.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Pair = fPair
	}
	txs, err := e.allUserTransactions(ctx, params)
	if err != nil {
		return nil, err
	}
	aggregated := make(map[uint64]*order.Detail)
	for i := range txs {
		tx := &txs[i]
		if tx.Type != TransactionTypeMarketTrade || tx.OrderID == 0 {
			continue
		}
		// The API's time bounds are whole seconds, so a fractional start can also return trades from earlier in its second
		if (!req.StartTime.IsZero() && tx.DateTime.Before(req.StartTime)) || (!req.EndTime.IsZero() && tx.DateTime.After(req.EndTime)) {
			continue
		}
		pair, price, ok := tradedPair(tx)
		if !ok {
			continue
		}
		baseAmount := tx.Amounts[pair.Base]
		side := order.Buy
		if baseAmount < 0 {
			side = order.Sell
		}
		d, ok := aggregated[tx.OrderID]
		if !ok {
			d = &order.Detail{
				Exchange:  e.Name,
				OrderID:   strconv.FormatUint(tx.OrderID, 10),
				Side:      side,
				AssetType: asset.Spot,
				Pair:      currency.NewPairWithDelimiter(pair.Base.String(), pair.Quote.String(), currency.ForwardSlashDelimiter),
				FeeAsset:  pair.Quote,
				CostAsset: pair.Quote,
				Date:      tx.DateTime,
			}
			aggregated[tx.OrderID] = d
		}
		addTrade(d, &order.TradeHistory{
			TID:       strconv.FormatUint(tx.ID, 10),
			Price:     price,
			Amount:    math.Abs(baseAmount),
			Fee:       tx.Fee,
			Exchange:  e.Name,
			Side:      side,
			Timestamp: tx.DateTime,
			FeeAsset:  pair.Quote.String(),
		})
	}
	return aggregatedOrders(aggregated)
}

// allUserTransactions returns every page of user transactions matching a request sorted oldest first
// The API skips at most 200000 transactions, so the pages after that continue from the last transaction's ID, as the
// API documents for older history, which takes transaction IDs to rise with the dates the pages are sorted by
func (e *Exchange) allUserTransactions(ctx context.Context, params *UserTransactionsRequest) ([]UserTransactionResponse, error) {
	var txs []UserTransactionResponse
	seen := make(map[uint64]bool)
	params.Offset = 0
	var bySinceID bool
	for {
		page, err := e.GetUserTransactions(ctx, params)
		if err != nil {
			return nil, err
		}
		var added bool
		for i := range page {
			if !seen[page[i].ID] {
				seen[page[i].ID] = true
				txs = append(txs, page[i])
				added = true
			}
		}
		if len(page) < maxResultLimit {
			return txs, nil
		}
		if !added {
			// A page of only known transactions means the offset or ID was not applied, so further pages would repeat
			// it. Past the maximum offset that leaves later transactions out, so the history is refused, not cut short
			if bySinceID {
				return nil, fmt.Errorf("%w: more than %d transactions, and none after ID %d were returned", errInvalidOffset, maxResultOffset, params.SinceID)
			}
			return txs, nil
		}
		if !bySinceID && params.Offset+maxResultLimit <= maxResultOffset {
			params.Offset += maxResultLimit
			continue
		}
		bySinceID = true
		params.Offset = 0
		params.SinceID = page[len(page)-1].ID
	}
}

// tradedPair returns the market and execution price of a trade transaction from its exchange rates
func tradedPair(tx *UserTransactionResponse) (currency.Pair, float64, bool) {
	for pair, rate := range tx.ExchangeRates {
		if rate != 0 && tx.Amounts[pair.Base] != 0 {
			return pair, rate, true
		}
	}
	return currency.EMPTYPAIR, 0, false
}

// derivativesOrderHistory returns derivatives orders aggregated from the account's trades on each market
// A self trade matches two of the account's orders, so it is added to both, with the fee charged to the first
func (e *Exchange) derivativesOrderHistory(ctx context.Context, pairs currency.Pairs, start, end time.Time) ([]order.Detail, error) {
	var orders []order.Detail
	for _, p := range pairs {
		fPair, err := e.FormatExchangeCurrency(p, asset.PerpetualContract)
		if err != nil {
			return nil, err
		}
		trades, err := e.allDerivativesTrades(ctx, &DerivativesTradeHistoryRequest{
			Pair:  fPair,
			Limit: maxResultLimit,
			Sort:  SortAscending,
			Since: start,
			Until: end,
		})
		if err != nil {
			return nil, err
		}
		aggregated := make(map[uint64]*order.Detail)
		for i := range trades {
			t := &trades[i]
			// The API's time bounds are whole seconds, so a fractional start can also return trades from earlier in its second
			if executed := t.DateTime.Time(); (!start.IsZero() && executed.Before(start)) || (!end.IsZero() && executed.After(end)) {
				continue
			}
			side := order.UnknownSide
			if t.Side != TradeSideSelf {
				if side, err = order.StringToOrderSide(t.Side); err != nil {
					return nil, err
				}
			}
			pair, _, err := marketPairAsset(t.Market)
			if err != nil {
				return nil, err
			}
			fee, err := gctmath.DecimalSumOfFloats(t.Fee.Float64(), t.LiquidationFee.Float64())
			if err != nil {
				return nil, fmt.Errorf("%w %d: %w", errMalformedTransaction, t.TradeID, err)
			}
			execution := &order.TradeHistory{
				TID:       strconv.FormatUint(t.TradeID, 10),
				Price:     t.Price.Float64(),
				Amount:    t.Amount.Float64(),
				Fee:       fee.InexactFloat64(),
				Exchange:  e.Name,
				Side:      side,
				Timestamp: t.DateTime.Time(),
				FeeAsset:  t.FeeCurrency.Upper().String(),
			}
			addTrade(aggregatedOrder(aggregated, e.Name, t, t.OrderID, pair, side), execution)
			if t.Side != TradeSideSelf || t.SelfTradeOrderID == "" {
				continue
			}
			selfTradeOrderID, err := strconv.ParseUint(t.SelfTradeOrderID, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w %q: %w", order.ErrOrderIDNotSet, t.SelfTradeOrderID, err)
			}
			complementary := *execution
			complementary.Fee = 0
			addTrade(aggregatedOrder(aggregated, e.Name, t, selfTradeOrderID, pair, side), &complementary)
		}
		pairOrders, err := aggregatedOrders(aggregated)
		if err != nil {
			return nil, err
		}
		orders = append(orders, pairOrders...)
	}
	return orders, nil
}

// aggregatedOrder returns the order a derivatives trade belongs to, adding it when it has not been seen
// An order first seen through a self trade has an unknown side until one of its other trades reveals it
func aggregatedOrder(aggregated map[uint64]*order.Detail, exchangeName string, t *DerivativesTradeResponse, orderID uint64, pair currency.Pair, side order.Side) *order.Detail {
	d, ok := aggregated[orderID]
	if !ok {
		d = &order.Detail{
			Exchange:   exchangeName,
			OrderID:    strconv.FormatUint(orderID, 10),
			Side:       side,
			AssetType:  asset.PerpetualContract,
			Pair:       pair,
			FeeAsset:   t.FeeCurrency.Upper(),
			CostAsset:  t.FeeCurrency.Upper(),
			Leverage:   t.Leverage.Float64(),
			MarginType: marginTypeFromMode(t.MarginMode),
			Date:       t.DateTime.Time(),
		}
		aggregated[orderID] = d
	}
	if d.Side == order.UnknownSide {
		d.Side = side
	}
	return d
}

// allDerivativesTrades returns every page of derivatives trades matching a request, oldest first
// Each page starts from the last trade ID of the previous page, which may be returned again
func (e *Exchange) allDerivativesTrades(ctx context.Context, req *DerivativesTradeHistoryRequest) ([]DerivativesTradeResponse, error) {
	var trades []DerivativesTradeResponse
	seen := make(map[uint64]bool)
	for {
		page, err := e.GetDerivativesTradeHistory(ctx, req)
		if err != nil {
			return nil, err
		}
		var added bool
		for i := range page {
			if !seen[page[i].TradeID] {
				seen[page[i].TradeID] = true
				trades = append(trades, page[i])
				added = true
			}
		}
		if len(page) < maxResultLimit || !added {
			return trades, nil
		}
		req.AfterID = page[len(page)-1].TradeID
	}
}

// addTrade adds an executed trade to an order, whose totals aggregatedOrders sums from its trades
func addTrade(d *order.Detail, t *order.TradeHistory) {
	d.Trades = append(d.Trades, *t)
	if t.Timestamp.Before(d.Date) {
		d.Date = t.Timestamp
	}
	if t.Timestamp.After(d.LastUpdated) {
		d.LastUpdated = t.Timestamp
	}
}

// aggregatedOrders returns orders aggregated from their trades sorted by order date, and by order ID when their dates
// are equal, such as both orders of a self trade
func aggregatedOrders(aggregated map[uint64]*order.Detail) ([]order.Detail, error) {
	orders := make([]order.Detail, 0, len(aggregated))
	for _, id := range slices.Sorted(maps.Keys(aggregated)) {
		d := aggregated[id]
		var totals tradeTotals
		for i := range d.Trades {
			if err := totals.add(d.Trades[i].Amount, d.Trades[i].Price, d.Trades[i].Fee); err != nil {
				return nil, fmt.Errorf("%w %s: %w", errMalformedTransaction, d.Trades[i].TID, err)
			}
		}
		d.Amount = totals.amount.InexactFloat64()
		d.ExecutedAmount = d.Amount
		d.Cost = totals.cost.InexactFloat64()
		d.Fee = totals.fee.InexactFloat64()
		if totals.amount.IsPositive() {
			d.AverageExecutedPrice = totals.averagePrice()
			d.Price = d.AverageExecutedPrice
		}
		orders = append(orders, *d)
	}
	slices.SortStableFunc(orders, func(a, b order.Detail) int {
		return a.Date.Compare(b.Date)
	})
	return orders, nil
}

// tradeTotals sums the amounts, costs and fees of trades as decimals and converts each total once, as float sums carry
// a residue, such as 30.299999999999997 for 0.3 at 101, which reaches the order store
type tradeTotals struct {
	amount, cost, fee decimal.Decimal
}

// add adds a trade's amount, cost and fee to the totals
func (t *tradeTotals) add(amount, price, fee float64) error {
	a, err := gctmath.DecimalFromFloat(amount)
	if err != nil {
		return err
	}
	p, err := gctmath.DecimalFromFloat(price)
	if err != nil {
		return err
	}
	f, err := gctmath.DecimalFromFloat(fee)
	if err != nil {
		return err
	}
	t.amount, t.cost, t.fee = t.amount.Add(a), t.cost.Add(a.Mul(p)), t.fee.Add(f)
	return nil
}

// averagePrice returns the trades' average price, or zero without an executed amount
func (t *tradeTotals) averagePrice() float64 {
	return gctmath.FloatQuotientOfDecimals(t.cost, t.amount)
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
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
	candles, err := e.candlesInRange(ctx, req.RequestFormatted, req.ExchangeInterval, req.Start, req.End)
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		candles, err := e.candlesInRange(ctx, req.RequestFormatted, req.ExchangeInterval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, candles...)
	}
	return req.ProcessResponse(timeSeries)
}

// candlesInRange returns the candles starting within a time range
// The API ignores the start time when an end time is also sent and returns the limit of candles before the end time,
// so only the start time and the number of candles in the range are sent
func (e *Exchange) candlesInRange(ctx context.Context, pair currency.Pair, interval kline.Interval, start, end time.Time) ([]kline.Candle, error) {
	resp, err := e.GetOHLC(ctx, &OHLCRequest{
		Pair:  pair,
		Step:  interval,
		Limit: kline.TotalCandlesPerInterval(start, end, interval),
		Start: start,
	})
	if err != nil {
		return nil, err
	}
	candles := make([]kline.Candle, len(resp.Data.OHLC))
	for i := range resp.Data.OHLC {
		c := &resp.Data.OHLC[i]
		candles[i] = kline.Candle{
			Time:   c.Timestamp.Time(),
			Open:   c.Open.Float64(),
			High:   c.High.Float64(),
			Low:    c.Low.Float64(),
			Close:  c.Close.Float64(),
			Volume: c.Volume.Float64(),
		}
	}
	return candles, nil
}

// GetServerTime is not supported as the API has no server time endpoint
func (e *Exchange) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, a asset.Item) ([]futures.Contract, error) {
	if !a.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if a != asset.PerpetualContract {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	markets, err := e.GetMarkets(ctx, "")
	if err != nil {
		return nil, err
	}
	contracts := make([]futures.Contract, 0, len(markets))
	for i := range markets {
		m := &markets[i]
		if m.MarketType != MarketTypePerpetual {
			continue
		}
		pair, err := currency.NewPairDelimiter(m.Name, currency.ForwardSlashDelimiter)
		if err != nil {
			return nil, err
		}
		settlementType := futures.UnsetSettlementType
		if strings.EqualFold(m.PayoffType, "Linear") {
			settlementType = futures.Linear
		}
		contracts = append(contracts, futures.Contract{
			Exchange:           e.Name,
			Name:               pair,
			Underlying:         currency.NewPair(m.BaseCurrency.Upper(), m.CounterCurrency.Upper()),
			Asset:              a,
			IsActive:           bool(m.TradingEnabled),
			Status:             m.TradingEnabled.String(),
			Type:               futures.Perpetual,
			SettlementType:     settlementType,
			SettlementCurrency: m.CounterCurrency.Upper(),
			MarginCurrency:     m.CounterCurrency.Upper(),
			Multiplier:         m.ContractSize.Float64(),
			MaxLeverage:        m.MaxLeverage.Float64(),
		})
	}
	return contracts, nil
}

// GetLatestFundingRates returns the latest settled funding rates of perpetual markets
// When no pair is set the rates of all enabled pairs are returned. The current rate is settled at the next funding
// time, so it is returned as the predicted rate when requested
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if isPerp, err := e.IsPerpetualFutureCurrency(r.Asset, r.Pair); err != nil {
		return nil, err
	} else if !isPerp {
		return nil, fmt.Errorf("%w %s %s", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
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
		fPair, err := e.FormatExchangeCurrency(p, r.Asset)
		if err != nil {
			return nil, err
		}
		rate, err := e.GetFundingRate(ctx, fPair)
		if err != nil {
			return nil, err
		}
		history, err := e.GetFundingRateHistory(ctx, &FundingRateHistoryRequest{Pair: fPair, Limit: 1})
		if err != nil {
			return nil, err
		}
		latest := fundingrate.LatestRateResponse{
			Exchange:       e.Name,
			Asset:          r.Asset,
			Pair:           fPair,
			TimeOfNextRate: rate.NextFundingTime.Time(),
			TimeChecked:    time.Now(),
		}
		if n := len(history.FundingRateHistory); n > 0 {
			settled := &history.FundingRateHistory[n-1]
			latest.LatestRate = fundingrate.Rate{
				Time: settled.Timestamp.Time(),
				Rate: settled.FundingRate.Decimal(),
			}
		}
		if r.IncludePredictedRate {
			latest.PredictedUpcomingRate = fundingrate.Rate{
				Time: rate.NextFundingTime.Time(),
				Rate: rate.FundingRate.Decimal(),
			}
		}
		resp = append(resp, latest)
	}
	return resp, nil
}

// GetHistoricalFundingRates returns the settled funding rates of a perpetual market within a time range, or the latest
// 100 without a start date
// The current rate is settled at the next funding time, so it is returned as the predicted rate when requested
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.PerpetualContract {
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
	rates, err := e.fundingRatesInRange(ctx, fPair, r.StartDate, r.EndDate)
	if err != nil {
		return nil, err
	}
	if len(rates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
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
		rate, err := e.GetFundingRate(ctx, fPair)
		if err != nil {
			return nil, err
		}
		resp.TimeOfNextRate = rate.NextFundingTime.Time()
		resp.PredictedUpcomingRate = fundingrate.Rate{
			Time: rate.NextFundingTime.Time(),
			Rate: rate.FundingRate.Decimal(),
		}
	}
	return resp, nil
}

// fundingRatesInRange returns the funding rates of a perpetual market within a time range, oldest first
// The API returns the latest 100 rates up to a range's end, both bounds inclusive, and refuses a range which holds no
// rate, so after a full page the range ends at the page's oldest rate, which is returned again. A range without a start
// is only the latest page
func (e *Exchange) fundingRatesInRange(ctx context.Context, pair currency.Pair, start, end time.Time) ([]fundingrate.Rate, error) {
	req := &FundingRateHistoryRequest{
		Pair:  pair,
		Limit: maxFundingRateHistoryLimit,
		Since: start,
		Until: end,
	}
	var rates []fundingrate.Rate
	seen := make(map[int64]bool)
	for {
		history, err := e.GetFundingRateHistory(ctx, req)
		if err != nil {
			// The API refuses a range which holds no rate with this message, which it also gives for a limit over 100, so the
			// first page's refusal means there are none. A later page ends at a rate the page before returned
			if len(seen) == 0 && errors.Is(err, errAPIResponse) && strings.Contains(err.Error(), "Cannot fetch funding rate history for market "+formatMarketName(pair)) {
				return nil, fmt.Errorf("%w: %w", fundingrate.ErrNoFundingRatesFound, err)
			}
			return nil, err
		}
		var oldest time.Time
		var added bool
		for i := range history.FundingRateHistory {
			settled := history.FundingRateHistory[i].Timestamp.Time()
			if oldest.IsZero() || settled.Before(oldest) {
				oldest = settled
			}
			if seen[settled.Unix()] {
				continue
			}
			seen[settled.Unix()] = true
			added = true
			// The API's bounds are whole seconds, so a rate in the second before a fractional start is returned too
			if (!start.IsZero() && settled.Before(start)) || (!end.IsZero() && settled.After(end)) {
				continue
			}
			rates = append(rates, fundingrate.Rate{
				Time: settled,
				Rate: history.FundingRateHistory[i].FundingRate.Decimal(),
			})
		}
		// A page of only known rates means the end was not applied, so further pages would repeat it
		if start.IsZero() || len(history.FundingRateHistory) < maxFundingRateHistoryLimit || !added || !oldest.After(start) {
			break
		}
		req.Until = oldest
	}
	slices.SortFunc(rates, func(a, b fundingrate.Rate) int {
		return a.Time.Compare(b.Time)
	})
	return rates, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.PerpetualContract, nil
}

// GetCollateralCurrencyForContract returns the settlement currency of a perpetual contract
func (e *Exchange) GetCollateralCurrencyForContract(a asset.Item, cp currency.Pair) (currency.Code, asset.Item, error) {
	if a != asset.PerpetualContract {
		return currency.EMPTYCODE, asset.Empty, fmt.Errorf("%w %q", futures.ErrNotPerpetualFuture, a)
	}
	if cp.IsEmpty() {
		return currency.EMPTYCODE, asset.Empty, currency.ErrCurrencyPairEmpty
	}
	return currency.NewCode(strings.TrimSuffix(cp.Quote.Upper().String(), perpetualMarketSuffix)).Upper(), a, nil
}

// GetOpenInterest returns the open interest of perpetual markets in their base currency
func (e *Exchange) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range keys {
		if keys[i].Asset != asset.PerpetualContract {
			// Avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, keys[i].Asset, keys[i].Pair())
		}
	}
	tickers, err := e.GetTickers(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.OpenInterest, 0, len(tickers))
	for i := range tickers {
		if tickers[i].MarketType != MarketTypePerpetual {
			continue
		}
		pair, err := currency.NewPairDelimiter(tickers[i].Market, currency.ForwardSlashDelimiter)
		if err != nil {
			return nil, err
		}
		if len(keys) > 0 && !slices.ContainsFunc(keys, func(k key.PairAsset) bool { return k.Pair().Equal(pair) }) {
			continue
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, asset.PerpetualContract, pair),
			OpenInterest: tickers[i].OpenInterest.Float64(),
		})
	}
	return resp, nil
}

// SetLeverage sets the leverage used for a perpetual market and margin type
func (e *Exchange) SetLeverage(ctx context.Context, a asset.Item, pair currency.Pair, marginType margin.Type, amount float64, _ order.Side) error {
	if a != asset.PerpetualContract {
		return fmt.Errorf("%w %q", futures.ErrNotPerpetualFuture, a)
	}
	// UpdateLeverageSetting omits a zero leverage to only change the margin mode, so it is rejected here
	if err := validateLeverage(amount); err != nil {
		return err
	}
	marginMode, err := formatMarginMode(marginType)
	if err != nil {
		return err
	}
	fPair, err := e.FormatExchangeCurrency(pair, a)
	if err != nil {
		return err
	}
	_, err = e.UpdateLeverageSetting(ctx, &LeverageSettingRequest{
		Pair:       fPair,
		MarginMode: marginMode,
		Leverage:   amount,
	})
	return err
}

// GetLeverage returns the leverage used for a perpetual market and margin type
func (e *Exchange) GetLeverage(ctx context.Context, a asset.Item, pair currency.Pair, marginType margin.Type, _ order.Side) (float64, error) {
	if a != asset.PerpetualContract {
		return 0, fmt.Errorf("%w %q", futures.ErrNotPerpetualFuture, a)
	}
	if pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	marginMode, err := formatMarginMode(marginType)
	if err != nil {
		return 0, err
	}
	fPair, err := e.FormatExchangeCurrency(pair, a)
	if err != nil {
		return 0, err
	}
	settings, err := e.GetLeverageSettings(ctx, &LeverageSettingsRequest{MarginMode: marginMode, Pair: fPair})
	if err != nil {
		return 0, err
	}
	market := formatMarketName(fPair)
	for i := range settings {
		if settings[i].Market == market && settings[i].MarginMode == marginMode {
			return settings[i].LeverageCurrent.Float64(), nil
		}
	}
	return 0, fmt.Errorf("%w: %s %s", errLeverageSettingNotFound, marginMode, market)
}

// GetFuturesPositionSummary returns a summary of the open position of a perpetual market
// Isolated positions are preferred over cross margin positions when both are open for the market
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, r *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.PerpetualContract {
		return nil, fmt.Errorf("%w %q", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	positions, err := e.GetOpenPositions(ctx, fPair)
	if err != nil {
		return nil, err
	}
	pos, err := selectPosition(positions, formatMarketName(fPair))
	if err != nil {
		return nil, err
	}
	size := pos.Size.Decimal()
	if pos.Side == PositionSideShort {
		size = size.Abs().Neg()
	}
	return &futures.PositionSummary{
		Pair:                         r.Pair,
		Asset:                        r.Asset,
		MarginType:                   marginTypeFromMode(pos.MarginMode),
		Currency:                     pos.SettlementCurrency.Upper(),
		IsolatedMargin:               pos.CollateralReserved.Decimal(),
		NotionalSize:                 pos.CurrentValue.Decimal(),
		Leverage:                     pos.Leverage.Decimal(),
		MaintenanceMarginRequirement: pos.MaintenanceMargin.Decimal(),
		InitialMarginRequirement:     pos.InitialMargin.Decimal(),
		EstimatedLiquidationPrice:    pos.EstimatedLiquidationPrice.Decimal(),
		CollateralUsed:               pos.CurrentMargin.Decimal(),
		MarkPrice:                    pos.MarkPrice.Decimal(),
		CurrentSize:                  size,
		ContractSettlementType:       futures.Linear,
		AverageOpenPrice:             pos.EntryPrice.Decimal(),
		UnrealisedPNL:                pos.PNLUnrealised.Decimal(),
		RealisedPNL:                  pos.PNLRealised.Decimal(),
		MaintenanceMarginFraction:    pos.MaintenanceMarginRatio.Decimal(),
	}, nil
}

// selectPosition returns the open position of a market, preferring isolated margin positions
func selectPosition(positions []PositionResponse, market string) (*PositionResponse, error) {
	var selected *PositionResponse
	for i := range positions {
		if positions[i].Market != market {
			continue
		}
		if selected == nil || positions[i].MarginMode == MarginModeIsolated {
			selected = &positions[i]
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("%w for %s", futures.ErrNoPositionsFound, market)
	}
	return selected, nil
}

// GetFuturesPositionOrders returns the orders of perpetual market positions, aggregated from their trades
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, r *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.PerpetualContract {
		return nil, fmt.Errorf("%w %q", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if len(r.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	resp := make([]futures.PositionResponse, len(r.Pairs))
	for i, p := range r.Pairs {
		orders, err := e.derivativesOrderHistory(ctx, currency.Pairs{p}, r.StartDate, r.EndDate)
		if err != nil {
			return nil, err
		}
		resp[i] = futures.PositionResponse{
			Pair:                   p,
			Asset:                  r.Asset,
			ContractSettlementType: futures.Linear,
			Orders:                 withinTimeRange(orders, r.StartDate, r.EndDate),
		}
	}
	return resp, nil
}

// ChangePositionMargin sets the collateral of an isolated margin position
func (e *Exchange) ChangePositionMargin(ctx context.Context, r *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.PerpetualContract {
		return nil, fmt.Errorf("%w %q", futures.ErrNotPerpetualFuture, r.Asset)
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
	fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	positions, err := e.GetOpenPositions(ctx, fPair)
	if err != nil {
		return nil, err
	}
	market := formatMarketName(fPair)
	var positionID string
	for i := range positions {
		if positions[i].Market == market && positions[i].MarginMode == MarginModeIsolated {
			positionID = positions[i].ID
			break
		}
	}
	if positionID == "" {
		return nil, fmt.Errorf("%w for isolated %s", futures.ErrNoPositionsFound, market)
	}
	if err := e.AdjustPositionCollateral(ctx, &AdjustPositionCollateralRequest{PositionID: positionID, NewAmount: r.NewAllocatedMargin}); err != nil {
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

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	return tradeBaseURL + formatMarketSymbol(cp) + "/", nil
}

// marketTypeForAsset returns the market type of an asset type
func marketTypeForAsset(a asset.Item) (string, error) {
	switch a {
	case asset.Spot:
		return MarketTypeSpot, nil
	case asset.PerpetualContract:
		return MarketTypePerpetual, nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// settlementCurrency returns the currency in which fees and costs of a market are paid: the quote currency of a spot
// market or the settlement currency of a perpetual contract
func settlementCurrency(pair currency.Pair, a asset.Item) currency.Code {
	if a == asset.PerpetualContract {
		return currency.NewCode(strings.TrimSuffix(pair.Quote.Upper().String(), perpetualMarketSuffix)).Upper()
	}
	return pair.Quote.Upper()
}

// marketPairAsset returns the currency pair and asset type of a market name, e.g. BTC/USD or BTC/USD-PERP
func marketPairAsset(market string) (currency.Pair, asset.Item, error) {
	pair, err := currency.NewPairDelimiter(market, currency.ForwardSlashDelimiter)
	if err != nil {
		return currency.EMPTYPAIR, asset.Empty, err
	}
	if strings.HasSuffix(market, perpetualMarketSuffix) {
		return pair, asset.PerpetualContract, nil
	}
	return pair, asset.Spot, nil
}

// formatMarginMode returns the margin mode of a margin type; cross margin is used when the margin type is unset
func formatMarginMode(t margin.Type) (string, error) {
	switch t {
	case margin.Unset, margin.Multi:
		return MarginModeCross, nil
	case margin.Isolated:
		return MarginModeIsolated, nil
	default:
		return "", fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, t)
	}
}

// marginTypeFromMode returns the margin type of a margin mode
func marginTypeFromMode(mode string) margin.Type {
	switch mode {
	case MarginModeCross:
		return margin.Multi
	case MarginModeIsolated:
		return margin.Isolated
	default:
		return margin.Unset
	}
}

// formatTrigger returns the trigger of a stop order price type
func formatTrigger(t order.PriceType) string {
	switch t {
	case order.IndexPrice:
		return TriggerIndexPrice
	case order.MarkPrice:
		return TriggerMarkPrice
	default:
		return TriggerLastTradedPrice
	}
}

// orderTypeFromSubtype returns the order type of a derivatives order subtype; orders without a subtype are limit orders
func orderTypeFromSubtype(subtype string) order.Type {
	switch subtype {
	case "", OrderSubtypeLimit:
		return order.Limit
	case OrderSubtypeMarket, OrderSubtypeInstant, OrderSubtypeCash:
		return order.Market
	case OrderSubtypeStopMarket, OrderSubtypeStopLoss:
		return order.StopMarket
	case OrderSubtypeStopLimit, OrderSubtypeStopLossLimit:
		return order.StopLimit
	case OrderSubtypeTakeProfit:
		return order.TakeProfitMarket
	case OrderSubtypeTakeProfitLimit:
		return order.TakeProfit | order.Limit
	case OrderSubtypeTrailingStopLoss, OrderSubtypeTrailingTakeProfit:
		return order.TrailingStop
	case OrderSubtypeTrailingStopLossLimit, OrderSubtypeTrailingTakeProfitLimit:
		return order.TrailingStopLimit
	default:
		return order.UnknownType
	}
}
