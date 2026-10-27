package lbank

import (
	"context"
	"errors"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	// orderbookDepth is the depth an order book is fetched to, the most the spot API returns
	orderbookDepth = 200
	// offlineTradeFeeRate is LBank's standard spot trading fee rate, 0.1% for makers and takers alike
	offlineTradeFeeRate = 0.001
	// errorCodePairNotSupported is the error LBank reports for a pair an endpoint doesn't serve
	errorCodePairNotSupported = 10008
	// maxOrderPages bounds an order listing's pages, so a venue that kept returning full pages couldn't page forever
	maxOrderPages = 1000
)

// SetDefaults sets the basic defaults for LBank
func (e *Exchange) SetDefaults() {
	e.Name = "Lbank"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	for a, ps := range map[asset.Item]currency.PairStore{
		asset.Spot: {
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter},
			ConfigFormat:  &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter},
		},
		asset.USDTMarginedFutures: {
			// Off by default, as GetActiveOrders refuses contracts: with REST authentication on, the order manager would
			// log that on each pass while a contract pair is enabled
			AssetEnabled:  false,
			RequestFormat: &currency.PairFormat{Uppercase: true},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		},
	} {
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}
	// LBank documents no websocket channels for its contracts
	if err := e.DisableAssetWebsocketSupport(asset.USDTMarginedFutures); err != nil {
		log.Errorln(log.ExchangeSys, err)
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
				TickerBatching:        true,
				TickerFetching:        true,
				KlineFetching:         true,
				TradeFetching:         true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrder:           true,
				CancelOrders:          true,
				SubmitOrder:           true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
				FundingRateFetching:   true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				TradeFetching:          true,
				KlineFetching:          true,
				AccountBalance:         true,
				GetOrders:              true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.OneHour:   true,
					kline.FourHour:  true,
					kline.EightHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.USDTMarginedFutures: true,
				},
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				// LBank's day, week and month candles start at midnight UTC+8, so those intervals are built from 4 hour
				// candles instead
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
				),
				GlobalResultLimit: maxKlineSize,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimits()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:         restURL,
		exchange.RestUSDTMargined: contractRESTURL,
		exchange.WebsocketSpot:    websocketURL,
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup sets exchange configuration profile
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
	if e.API.AuthenticatedSupport {
		if creds := e.GetDefaultCredentials(); creds != nil && creds.Secret != "" {
			if _, err := e.getSigner(creds.Secret); err != nil {
				e.API.AuthenticatedSupport = false
				e.API.AuthenticatedWebsocketSupport = false
				log.Errorf(log.ExchangeSys, "%s cannot sign requests with the configured secret, setting authenticated support to false: %v", e.Name, err)
			}
		}
	}
	return e.setupWebsocket(exch)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch a {
	case asset.Spot:
		symbols, err := e.GetCurrencyPairs(ctx)
		if err != nil {
			return nil, err
		}
		return currency.NewPairsFromStrings(symbols)
	case asset.USDTMarginedFutures:
		instruments, err := e.GetContractInstruments(ctx, ProductGroupUSDTSwap)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, len(instruments))
		for i := range instruments {
			pairs[i] = currency.NewPair(instruments[i].BaseCurrency, instruments[i].PriceCurrency)
		}
		return pairs, nil
	}
	return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config. A disabled asset
// whose pairs fail to load keeps the pairs it has, so a contract API failure doesn't stop the spot pairs updating
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(ctx, a)
		if err == nil {
			err = e.UpdatePairs(pairs, a, false)
		}
		if err == nil {
			continue
		}
		if e.CurrencyPairs.IsAssetEnabled(a) == nil {
			return err
		}
		log.Warnf(log.ExchangeSys, "%s could not update the pairs of disabled asset %s: %v", e.Name, a, err)
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	enabled, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	var tickers []*ticker.Price
	switch a {
	case asset.Spot:
		standard, err := e.GetTickers(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		// Leveraged tokens are spot pairs whose tickers come from an endpoint of their own
		leveraged, err := e.GetLeveragedTokenTickers(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		tickers, err = e.spotTickers(slices.Concat(standard, leveraged), enabled)
		if err != nil {
			return err
		}
	case asset.USDTMarginedFutures:
		data, err := e.GetContractMarketData(ctx, ProductGroupUSDTSwap)
		if err != nil {
			return err
		}
		if tickers, err = e.contractTickers(data, enabled); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	var errs error
	for _, t := range tickers {
		if err := ticker.ProcessTicker(t); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// spotTickers returns the tickers of the pairs given
func (e *Exchange) spotTickers(data []TickerResponse, pairs currency.Pairs) ([]*ticker.Price, error) {
	wanted, err := e.symbolPairs(pairs, asset.Spot)
	if err != nil {
		return nil, err
	}
	tickers := make([]*ticker.Price, 0, len(pairs))
	for i := range data {
		pair, ok := wanted[data[i].Symbol]
		if !ok {
			continue
		}
		tickers = append(tickers, &ticker.Price{
			ExchangeName:        e.Name,
			Pair:                pair,
			AssetType:           asset.Spot,
			Last:                data[i].Ticker.LastPrice.Float64(),
			High:                data[i].Ticker.High24Hour.Float64(),
			Low:                 data[i].Ticker.Low24Hour.Float64(),
			BaseVolume:          data[i].Ticker.Volume24Hour.Float64(),
			QuoteVolume:         data[i].Ticker.Turnover24Hour.Float64(),
			PercentChange24Hour: data[i].Ticker.ChangePercentage24Hour.Float64(),
			LastUpdated:         data[i].Timestamp.Time(),
		})
	}
	return tickers, nil
}

// contractTickers returns the tickers of the pairs given
func (e *Exchange) contractTickers(data []ContractMarketDataResponse, pairs currency.Pairs) ([]*ticker.Price, error) {
	wanted, err := e.symbolPairs(pairs, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	tickers := make([]*ticker.Price, 0, len(pairs))
	for i := range data {
		pair, ok := wanted[data[i].Symbol]
		if !ok {
			continue
		}
		tickers = append(tickers, &ticker.Price{
			ExchangeName: e.Name,
			Pair:         pair,
			AssetType:    asset.USDTMarginedFutures,
			Last:         data[i].LastPrice.Float64(),
			High:         data[i].HighPrice24Hour.Float64(),
			Low:          data[i].LowPrice24Hour.Float64(),
			Open24Hour:   data[i].OpenPrice24Hour.Float64(),
			BaseVolume:   data[i].Volume24Hour.Float64(),
			QuoteVolume:  data[i].Turnover24Hour.Float64(),
			MarkPrice:    data[i].MarkPrice.Float64(),
			IndexPrice:   data[i].UnderlyingPrice.Float64(),
			LastUpdated:  data[i].LastTime.Time(),
		})
	}
	return tickers, nil
}

// symbolPairs returns the pairs given keyed by their request symbol
func (e *Exchange) symbolPairs(pairs currency.Pairs, a asset.Item) (map[string]currency.Pair, error) {
	m := make(map[string]currency.Pair, len(pairs))
	for _, p := range pairs {
		symbol, err := e.FormatSymbol(p, a)
		if err != nil {
			return nil, err
		}
		m[symbol] = p
	}
	return m, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var tickers []*ticker.Price
	switch a {
	case asset.Spot:
		data, err := e.GetTickers(ctx, p)
		if apiErr := (*requestError)(nil); errors.As(err, &apiErr) && apiErr.code == errorCodePairNotSupported {
			// The standard ticker endpoint refuses leveraged tokens, which have their own
			data, err = e.GetLeveragedTokenTickers(ctx, p)
		}
		if err != nil {
			return nil, err
		}
		if tickers, err = e.spotTickers(data, currency.Pairs{p}); err != nil {
			return nil, err
		}
	case asset.USDTMarginedFutures:
		data, err := e.GetContractMarketData(ctx, ProductGroupUSDTSwap)
		if err != nil {
			return nil, err
		}
		if tickers, err = e.contractTickers(data, currency.Pairs{p}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	if len(tickers) == 0 {
		return nil, fmt.Errorf("%w: %s %s", ticker.ErrTickerNotFound, p, a)
	}
	if err := ticker.ProcessTicker(tickers[0]); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
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
		depth, err := e.GetMarketDepth(ctx, p, orderbookDepth)
		if err != nil {
			return nil, err
		}
		book.Bids = depth.Bids.Levels()
		book.Asks = depth.Asks.Levels()
		book.LastUpdated = depth.Timestamp.Time()
	case asset.USDTMarginedFutures:
		depth, err := e.GetContractOrderbook(ctx, p, orderbookDepth)
		if err != nil {
			return nil, err
		}
		book.Bids = contractLevels(depth.Bids)
		book.Asks = contractLevels(depth.Asks)
		// The contract order book carries no time, so it is dated when it arrives
		book.LastUpdated = time.Now()
	default:
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, a)
}

// contractLevels returns the order book levels of a contract order book side
func contractLevels(side []ContractOrderbookLevel) orderbook.Levels {
	levels := make(orderbook.Levels, len(side))
	for i := range side {
		levels[i] = orderbook.Level{
			Price:      side[i].Price.Float64(),
			Amount:     side[i].Volume.Float64(),
			OrderCount: side[i].OrderCount.Int64(),
		}
	}
	return levels
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	if err := checkSpot(a); err != nil {
		return nil, err
	}
	info, err := e.GetAccountInformation(ctx)
	if err != nil {
		return nil, err
	}
	subAccount := accounts.NewSubAccount(a, "")
	for i := range info.Balances {
		free, locked := info.Balances[i].Free.Float64(), info.Balances[i].Locked.Float64()
		subAccount.Balances.Set(info.Balances[i].Asset, accounts.Balance{
			Total: free + locked,
			Hold:  locked,
			Free:  free,
		})
	}
	subAccounts := accounts.SubAccounts{subAccount}
	return subAccounts, e.Accounts.Save(ctx, subAccounts, true)
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	deposits, err := e.GetDepositHistory(ctx, nil)
	if err != nil {
		return nil, err
	}
	withdrawals, err := e.GetWithdrawalHistory(ctx, nil)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(deposits.Deposits)+len(withdrawals.Withdrawals))
	for i := range deposits.Deposits {
		d := &deposits.Deposits[i]
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          depositStatus(d.Status),
			TransferID:      d.TransactionID,
			Timestamp:       d.InsertTime.Time(),
			Currency:        d.Coin.String(),
			Amount:          d.Amount.Float64(),
			TransferType:    "deposit",
			CryptoToAddress: d.Address,
			CryptoTxID:      d.TransactionID,
			CryptoChain:     d.NetworkName,
		})
	}
	for i := range withdrawals.Withdrawals {
		w := &withdrawals.Withdrawals[i]
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          withdrawalStatus(w.Status),
			TransferID:      strconv.FormatUint(w.ID, 10),
			Timestamp:       w.ApplyTime.Time(),
			Currency:        w.Coin.String(),
			Amount:          w.Amount.Float64(),
			Fee:             w.Fee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: w.Address,
			CryptoTxID:      w.TransactionID,
			CryptoChain:     w.NetworkName,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if err := checkSpot(a); err != nil {
		return nil, err
	}
	withdrawals, err := e.GetWithdrawalHistory(ctx, &WithdrawalHistoryRequest{Coin: c})
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals.Withdrawals))
	for i := range withdrawals.Withdrawals {
		w := &withdrawals.Withdrawals[i]
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawalStatus(w.Status),
			TransferID:      strconv.FormatUint(w.ID, 10),
			Timestamp:       w.ApplyTime.Time(),
			Currency:        w.Coin.String(),
			Amount:          w.Amount.Float64(),
			Fee:             w.Fee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: w.Address,
			CryptoTxID:      w.TransactionID,
			CryptoChain:     w.NetworkName,
		}
	}
	return resp, nil
}

// depositStatus returns the description of a deposit status
func depositStatus(status string) string {
	switch status {
	case "1":
		return "applying"
	case "2":
		return "successful"
	case "3":
		return "failed"
	case "4":
		return "cancelled"
	case "5":
		return "transfer"
	}
	return status
}

// withdrawalStatus returns the description of a withdrawal status
func withdrawalStatus(status string) string {
	switch status {
	case "1":
		return "applying"
	case "2":
		return "cancelled"
	case "3":
		return "failed"
	case "4":
		return "completed"
	}
	return status
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	if err := checkSpot(a); err != nil {
		return nil, err
	}
	trades, err := e.GetTrades(ctx, p, maxTradesSize, time.Time{})
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(trades))
	for i := range trades {
		resp[i] = e.tradeData(&trades[i], p)
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	// LBank lists the latest trades newest first, so they are reversed into the order they traded; the stable sort only
	// moves a trade LBank listed out of order
	slices.Reverse(resp)
	slices.SortStableFunc(resp, func(a, b trade.Data) int { return a.Timestamp.Compare(b.Timestamp) })
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided. LBank returns the trades after a time,
// oldest first, so the trades are paged forward from the start
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time) ([]trade.Data, error) {
	if err := checkSpot(a); err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	var resp []trade.Data
	seen := make(map[string]struct{})
	// The trades after a time are those strictly after it, so the first page asks from a millisecond before the start
	after := start.Add(-time.Millisecond)
	for {
		trades, err := e.GetTrades(ctx, p, maxTradesSize, after)
		if err != nil {
			return nil, err
		}
		var last time.Time
		var pastEnd bool
		for i := range trades {
			t := trades[i].Time.Time()
			if t.After(end) {
				pastEnd = true
				break
			}
			last = t
			if _, ok := seen[trades[i].ID]; ok || t.Before(start) {
				continue
			}
			seen[trades[i].ID] = struct{}{}
			resp = append(resp, e.tradeData(&trades[i], p))
		}
		if pastEnd || len(trades) < maxTradesSize {
			break
		}
		// The next page asks from a millisecond before the last trade, since more trades may share its millisecond; the
		// trades it returns again are skipped
		next := last.Add(-time.Millisecond)
		if !next.After(after) {
			return nil, fmt.Errorf("%w at %s", errHistoryPageFull, last)
		}
		after = next
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	// A stable sort keeps the venue's order of the trades sharing a millisecond
	slices.SortStableFunc(resp, func(a, b trade.Data) int { return a.Timestamp.Compare(b.Timestamp) })
	return resp, nil
}

// tradeData returns the trade data of a public trade
func (e *Exchange) tradeData(t *TradeResponse, p currency.Pair) trade.Data {
	side := order.Sell
	if t.IsBuyerMaker {
		side = order.Buy
	}
	return trade.Data{
		Exchange:     e.Name,
		TID:          t.ID,
		CurrencyPair: p,
		AssetType:    asset.Spot,
		Side:         side,
		Price:        t.Price,
		Amount:       t.Quantity,
		Timestamp:    t.Time.Time(),
	}
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	if err := checkSpot(s.AssetType); err != nil {
		return nil, err
	}
	orderType, err := orderTypeString(s.Side, s.Type, s.TimeInForce)
	if err != nil {
		return nil, err
	}
	req := &PlaceOrderRequest{
		Pair:     s.Pair,
		Type:     orderType,
		CustomID: s.ClientOrderID,
	}
	switch {
	case s.Type == order.Market && s.Side.IsLong():
		// A market buy spends a quote amount, which LBank takes as its price
		req.Price = s.QuoteAmount
	case s.Type == order.Market:
		req.Amount = s.Amount
	default:
		req.Price = s.Price
		req.Amount = s.Amount
	}
	resp, err := e.PlaceOrder(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(resp.OrderID)
}

// orderTypeString returns the LBank order type of a side, type and time in force
func orderTypeString(side order.Side, orderType order.Type, tif order.TimeInForce) (string, error) {
	var prefix string
	switch {
	case side.IsLong():
		prefix = "buy"
	case side.IsShort():
		prefix = "sell"
	default:
		return "", fmt.Errorf("%w: %s", order.ErrSideIsInvalid, side)
	}
	switch orderType {
	case order.Market:
		if tif != order.UnknownTIF && tif != order.ImmediateOrCancel {
			return "", fmt.Errorf("%w: %s with a %s order", order.ErrUnsupportedTimeInForce, tif, orderType)
		}
		return prefix + "_market", nil
	case order.Limit:
		switch tif {
		case order.UnknownTIF, order.GoodTillCancel:
			return prefix, nil
		case order.PostOnly, order.GoodTillCancel | order.PostOnly:
			return prefix + "_maker", nil
		case order.ImmediateOrCancel:
			return prefix + "_ioc", nil
		case order.FillOrKill:
			return prefix + "_fok", nil
		}
		return "", fmt.Errorf("%w: %s with a %s order", order.ErrUnsupportedTimeInForce, tif, orderType)
	}
	return "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, orderType)
}

// parseOrderType returns the side, type and time in force of an LBank order type, which is the side alone for a limit
// order, or the side and a suffix
func parseOrderType(orderType string) (order.Side, order.Type, order.TimeInForce, error) {
	sideText, suffix, _ := strings.Cut(orderType, "_")
	var side order.Side
	switch sideText {
	case "buy":
		side = order.Buy
	case "sell":
		side = order.Sell
	default:
		return order.UnknownSide, order.UnknownType, order.UnknownTIF, fmt.Errorf("%q %w", orderType, order.ErrSideIsInvalid)
	}
	switch suffix {
	case "":
		return side, order.Limit, order.GoodTillCancel, nil
	case "market":
		return side, order.Market, order.UnknownTIF, nil
	case "maker":
		return side, order.Limit, order.PostOnly, nil
	case "ioc":
		return side, order.Limit, order.ImmediateOrCancel, nil
	case "fok":
		return side, order.Limit, order.FillOrKill, nil
	}
	return order.UnknownSide, order.UnknownType, order.UnknownTIF, fmt.Errorf("%q %w", orderType, order.ErrUnsupportedOrderType)
}

// orderStatus returns the order status of an LBank order status
func orderStatus(status int64) (order.Status, error) {
	switch status {
	case -1:
		return order.Cancelled, nil
	case 0:
		return order.Active, nil
	case 1:
		return order.PartiallyFilled, nil
	case 2:
		return order.Filled, nil
	case 3:
		return order.PartiallyFilledCancelled, nil
	case 4:
		return order.Cancelling, nil
	}
	return order.UnknownStatus, fmt.Errorf("%w: %d", order.ErrUnsupportedStatusType, status)
}

// orderDetail returns the order detail of an order on a spot pair
func (e *Exchange) orderDetail(o *OrderResponse, pair currency.Pair) (*order.Detail, error) {
	side, orderType, tif, err := parseOrderType(o.Type)
	if err != nil {
		return nil, err
	}
	status, err := orderStatus(o.Status)
	if err != nil {
		return nil, err
	}
	d := &order.Detail{
		Exchange:       e.Name,
		OrderID:        o.OrderID,
		ClientOrderID:  o.ClientOrderID,
		Pair:           pair,
		AssetType:      asset.Spot,
		Side:           side,
		Type:           orderType,
		TimeInForce:    tif,
		Status:         status,
		Price:          o.Price.Float64(),
		Amount:         o.OriginalQuantity.Float64(),
		QuoteAmount:    o.OriginalQuoteOrderQuantity.Float64(),
		ExecutedAmount: o.ExecutedQuantity.Float64(),
		Cost:           o.CumulativeQuoteQuantity.Float64(),
		Date:           o.CreateTime.Time(),
		LastUpdated:    o.UpdateTime.Time(),
	}
	// A market buy orders a quote amount, leaving its base amount, and so its remaining amount, unknown
	if d.Amount > 0 {
		d.RemainingAmount = max(d.Amount-d.ExecutedAmount, 0)
	}
	if d.ExecutedAmount > 0 {
		d.AverageExecutedPrice = d.Cost / d.ExecutedAmount
	}
	return d, nil
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.PairAssetRequired()); err != nil {
		return err
	}
	if err := checkSpot(o.AssetType); err != nil {
		return err
	}
	if o.OrderID == "" && o.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	_, err := e.CancelExistingOrder(ctx, &OrderReferenceRequest{
		Pair:          o.Pair,
		OrderID:       o.OrderID,
		ClientOrderID: o.ClientOrderID,
	})
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers. LBank's current API has no endpoint cancelling
// several orders by their IDs
func (e *Exchange) CancelBatchOrders(context.Context, []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, o *order.Cancel) (order.CancelAllResponse, error) {
	if err := o.Validate(o.PairAssetRequired()); err != nil {
		return order.CancelAllResponse{}, err
	}
	if err := checkSpot(o.AssetType); err != nil {
		return order.CancelAllResponse{}, err
	}
	cancelled, err := e.CancelPairOrders(ctx, o.Pair)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	resp := order.CancelAllResponse{Status: make(map[string]string, len(cancelled))}
	for i := range cancelled {
		resp.Status[cancelled[i].OrderID] = order.Cancelled.String()
	}
	return resp, nil
}

// GetServerTime returns the current exchange server time
func (e *Exchange) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	switch a {
	case asset.Spot:
		return e.GetTimestamp(ctx)
	case asset.USDTMarginedFutures:
		return e.GetContractServerTime(ctx)
	}
	return time.Time{}, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	if err := checkSpot(a); err != nil {
		return nil, err
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	resp, err := e.QueryOrder(ctx, &OrderReferenceRequest{Pair: pair, OrderID: orderID})
	if err != nil {
		return nil, err
	}
	return e.orderDetail(resp, pair)
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	resp, err := e.GetCoinDepositAddress(ctx, c, chain)
	if err != nil {
		return nil, err
	}
	if resp.Address == "" {
		return nil, deposit.ErrAddressNotFound
	}
	if resp.NetworkName != "" {
		chain = resp.NetworkName
	}
	return &deposit.Address{Address: resp.Address, Tag: resp.Memo, Chain: chain}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted. LBank requires the chain's
// withdrawal fee, which is looked up when the request has none
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, r *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	fee := r.Crypto.FeeAmount
	if fee == 0 && !r.InternalTransfer {
		var err error
		if fee, err = e.networkWithdrawalFee(ctx, r.Currency, r.Crypto.Chain, r.Amount); err != nil {
			return nil, err
		}
	}
	resp, err := e.Withdraw(ctx, &WithdrawRequest{
		Address:          r.Crypto.Address,
		Coin:             r.Currency,
		Amount:           r.Amount,
		Fee:              fee,
		NetworkName:      r.Crypto.Chain,
		Memo:             r.Crypto.AddressTag,
		Mark:             r.Description,
		WithdrawOrderID:  r.ClientOrderID,
		InternalTransfer: r.InternalTransfer,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: strconv.FormatUint(resp.WithdrawID, 10)}, nil
}

// networkWithdrawalFee returns the account's withdrawal fee for an amount of a coin on a chain, or on the coin's default
// chain when the chain is empty
func (e *Exchange) networkWithdrawalFee(ctx context.Context, coin currency.Code, chain string, amount float64) (float64, error) {
	coins, err := e.GetAllCoinInformation(ctx)
	if err != nil {
		return 0, err
	}
	for i := range coins {
		if !coins[i].Coin.Equal(coin) {
			continue
		}
		for j := range coins[i].Networks {
			n := &coins[i].Networks[j]
			if (chain == "" && n.IsDefault) || (chain != "" && strings.EqualFold(n.Name, chain)) {
				return withdrawalFee(n.FeeType, n.WithdrawFee.Float64(), n.WithdrawFeeRate.Float64(), amount), nil
			}
		}
	}
	return 0, fmt.Errorf("%w: no withdrawal fee for %s on chain %q", currency.ErrCurrencyNotFound, coin, chain)
}

// withdrawalFee returns the fee of a withdrawal under LBank's fee types: 1 a fixed fee, 2 a rate and 3 both
func withdrawalFee(feeType uint64, fixed, rate, amount float64) float64 {
	switch feeType {
	case 2:
		return amount * rate
	case 3:
		return fixed + amount*rate
	}
	return fixed
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := checkSpot(req.AssetType); err != nil {
		return nil, err
	}
	orders, err := e.listOrders(ctx, req.Pairs, e.GetOpenOrders)
	if err != nil {
		return nil, err
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information. LBank returns the cancelled and filled orders placed in the last
// 24 hours
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := checkSpot(req.AssetType); err != nil {
		return nil, err
	}
	orders, err := e.listOrders(ctx, req.Pairs, e.QueryOrderHistory)
	if err != nil {
		return nil, err
	}
	return req.Filter(e.Name, orders), nil
}

// listOrders returns every order a paged listing returns for each pair, or for each enabled spot pair when none are
// given
func (e *Exchange) listOrders(ctx context.Context, pairs currency.Pairs, list func(context.Context, *OrdersPageRequest) (*OrdersPageResponse, error)) ([]order.Detail, error) {
	if len(pairs) == 0 {
		var err error
		if pairs, err = e.GetEnabledPairs(asset.Spot); err != nil {
			return nil, err
		}
	}
	var details []order.Detail
	for _, pair := range pairs {
		var previousFirst string
		for page := uint64(1); ; page++ {
			if page > maxOrderPages {
				return nil, fmt.Errorf("%w: %d pages of %s orders", errHistoryPageLimit, maxOrderPages, pair)
			}
			resp, err := list(ctx, &OrdersPageRequest{Pair: pair, CurrentPage: page, PageLength: maxOrdersPageLength})
			if err != nil {
				return nil, err
			}
			if len(resp.Orders) == 0 {
				break
			}
			// A venue answering every page with the first would otherwise be paged until the limit
			if resp.Orders[0].OrderID == previousFirst {
				break
			}
			previousFirst = resp.Orders[0].OrderID
			for i := range resp.Orders {
				d, err := e.orderDetail(&resp.Orders[i], pair)
				if err != nil {
					return nil, err
				}
				details = append(details, *d)
			}
			if len(resp.Orders) < maxOrdersPageLength || (resp.Total > 0 && page >= resp.Total) {
				break
			}
		}
	}
	return details, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if err := common.NilGuard(feeBuilder); err != nil {
		return 0, err
	}
	if feeBuilder.FeeType == exchange.CryptocurrencyTradeFee && !e.AreCredentialsValid(ctx) {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		return offlineTradeFeeRate * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	case exchange.CryptocurrencyTradeFee:
		rates, err := e.GetTradingFeeRates(ctx, feeBuilder.Pair)
		if err != nil {
			return 0, err
		}
		if len(rates) == 0 {
			return 0, fmt.Errorf("%w: no fee rate for %s", currency.ErrCurrencyNotFound, feeBuilder.Pair)
		}
		percent := rates[0].TakerCommissionPercent
		if feeBuilder.IsMaker {
			percent = rates[0].MakerCommissionPercent
		}
		return percent.Float64() / 100 * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	case exchange.CryptocurrencyWithdrawalFee:
		configs, err := e.GetAssetConfigs(ctx, feeBuilder.Pair.Base)
		if err != nil {
			return 0, err
		}
		if len(configs) == 0 {
			return 0, fmt.Errorf("%w: no withdrawal fee for %s", currency.ErrCurrencyNotFound, feeBuilder.Pair.Base)
		}
		fee := &configs[0].AssetFee
		return withdrawalFee(fee.Type, fee.FeeAmount.Float64(), fee.FeeRate.Float64(), feeBuilder.Amount), nil
	}
	return 0, nil
}

// GetAvailableTransferChains returns the chains a currency can be deposited and withdrawn on
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, c currency.Code) ([]string, error) {
	configs, err := e.GetAssetConfigs(ctx, c)
	if err != nil {
		return nil, err
	}
	chains := make([]string, len(configs))
	for i := range configs {
		chains[i] = configs[i].ChainName
	}
	return chains, nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, a asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, a)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if err := checkSpot(a); err != nil {
		return nil, err
	}
	// LBank serves candles from any start time, so the window isn't bound to the latest candles
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	candles, err := e.GetKlines(ctx, &KlineRequest{
		Pair:     req.RequestFormatted,
		Interval: req.ExchangeInterval,
		Start:    req.Start,
		Size:     req.Size(),
	})
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(klineCandles(candles))
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if err := checkSpot(a); err != nil {
		return nil, err
	}
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		r := &req.RangeHolder.Ranges[x]
		candles, err := e.GetKlines(ctx, &KlineRequest{
			Pair:     req.RequestFormatted,
			Interval: req.ExchangeInterval,
			Start:    r.Start.Time,
			Size:     kline.TotalCandlesPerInterval(r.Start.Time, r.End.Time, req.ExchangeInterval),
		})
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, klineCandles(candles)...)
	}
	return req.ProcessResponse(timeSeries)
}

// klineCandles returns the kline candles of LBank candles
func klineCandles(candles []KlineResponse) []kline.Candle {
	timeSeries := make([]kline.Candle, len(candles))
	for i := range candles {
		timeSeries[i] = kline.Candle{
			Time:   candles[i].Time,
			Open:   candles[i].Open,
			High:   candles[i].High,
			Low:    candles[i].Low,
			Close:  candles[i].Close,
			Volume: candles[i].Volume,
		}
	}
	return timeSeries
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, a asset.Item) ([]futures.Contract, error) {
	if !a.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if a != asset.USDTMarginedFutures {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	instruments, err := e.GetContractInstruments(ctx, ProductGroupUSDTSwap)
	if err != nil {
		return nil, err
	}
	contracts := make([]futures.Contract, len(instruments))
	for i := range instruments {
		pair := currency.NewPair(instruments[i].BaseCurrency, instruments[i].PriceCurrency)
		contracts[i] = futures.Contract{
			Exchange:           e.Name,
			Name:               pair,
			Underlying:         pair,
			Asset:              a,
			IsActive:           !instruments[i].NeedSuspend.Bool(),
			Type:               futures.Perpetual,
			SettlementType:     futures.Linear,
			SettlementCurrency: instruments[i].ClearCurrency,
			MarginCurrency:     instruments[i].ClearCurrency,
			Multiplier:         instruments[i].VolumeMultiple,
			MaxLeverage:        instruments[i].MaximumLeverage,
		}
	}
	return contracts, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.USDTMarginedFutures {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, r.Asset)
	}
	pairs := currency.Pairs{r.Pair}
	if r.Pair.IsEmpty() {
		var err error
		if pairs, err = e.GetEnabledPairs(r.Asset); err != nil {
			return nil, err
		}
	}
	wanted, err := e.symbolPairs(pairs, r.Asset)
	if err != nil {
		return nil, err
	}
	data, err := e.GetContractMarketData(ctx, ProductGroupUSDTSwap)
	if err != nil {
		return nil, err
	}
	checked := time.Now()
	rates := make([]fundingrate.LatestRateResponse, 0, len(pairs))
	for i := range data {
		pair, ok := wanted[data[i].Symbol]
		if !ok {
			continue
		}
		next := data[i].NextFundingTime.Time()
		rates = append(rates, fundingrate.LatestRateResponse{
			Exchange: e.Name,
			Asset:    r.Asset,
			Pair:     pair,
			LatestRate: fundingrate.Rate{
				Time: next.Add(-time.Duration(data[i].FundingIntervalSeconds) * time.Second), //nolint:gosec // LBank's funding intervals are hours long, far inside int64's range
				Rate: data[i].FundingRate.Decimal(),
			},
			TimeOfNextRate: next,
			TimeChecked:    checked,
		})
	}
	if !r.Pair.IsEmpty() && len(rates) == 0 {
		return nil, fmt.Errorf("%w: %s", currency.ErrCurrencyNotFound, r.Pair)
	}
	return rates, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future; every LBank contract is
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.USDTMarginedFutures, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var levels []limits.MinMaxLevel
	switch a {
	case asset.Spot:
		info, err := e.GetPairInfo(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		levels = make([]limits.MinMaxLevel, 0, len(info))
		for i := range info {
			pair, err := currency.NewPairFromString(info[i].Symbol)
			if err != nil {
				return err
			}
			levels = append(levels, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
				PriceStepIncrementSize:  math.Pow10(-int(info[i].PriceDecimals.Int64())),
				AmountStepIncrementSize: math.Pow10(-int(info[i].QuantityDecimals.Int64())),
				MinimumBaseAmount:       info[i].MinimumQuantity.Float64(),
				MinNotional:             info[i].MinimumOrderAmount.Float64(),
			})
		}
	case asset.USDTMarginedFutures:
		instruments, err := e.GetContractInstruments(ctx, ProductGroupUSDTSwap)
		if err != nil {
			return err
		}
		levels = make([]limits.MinMaxLevel, len(instruments))
		for i := range instruments {
			levels[i] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, currency.NewPair(instruments[i].BaseCurrency, instruments[i].PriceCurrency)),
				PriceStepIncrementSize:  instruments[i].PriceTick,
				AmountStepIncrementSize: instruments[i].VolumeTick,
				MinimumBaseAmount:       instruments[i].MinimumOrderVolume.Float64(),
				MaximumBaseAmount:       instruments[i].MaximumOrderVolume.Float64(),
				MinNotional:             instruments[i].MinimumOrderCost.Float64(),
			}
		}
	default:
		return fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	return limits.Load(levels)
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	if _, err := e.CurrencyPairs.IsPairEnabled(cp, a); err != nil {
		return "", err
	}
	switch a {
	case asset.Spot:
		return tradeBaseURL + cp.Format(currency.PairFormat{Delimiter: currency.UnderscoreDelimiter}).String(), nil
	case asset.USDTMarginedFutures:
		return futuresTradeBaseURL + cp.Format(currency.EMPTYFORMAT).String(), nil
	}
	return "", fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
}

// checkSpot returns an error for an asset other than spot, the only asset LBank trades through its API
func checkSpot(a asset.Item) error {
	if a != asset.Spot {
		return fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	return nil
}
