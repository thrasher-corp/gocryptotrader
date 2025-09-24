package exmo

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
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

// SetDefaults sets the basic defaults for exmo
func (e *Exchange) SetDefaults() {
	e.Name = "EXMO"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
		Separator: ",",
	}
	configFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}
	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
			RESTCapabilities: protocol.Features{
				TickerBatching:                    true,
				TickerFetching:                    true,
				TradeFetching:                     true,
				OrderbookFetching:                 true,
				AutoPairUpdates:                   true,
				AccountInfo:                       true,
				GetOrder:                          true,
				GetOrders:                         true,
				CancelOrder:                       true,
				SubmitOrder:                       true,
				DepositHistory:                    true,
				WithdrawalHistory:                 true,
				UserTradeHistory:                  true,
				CryptoDeposit:                     true,
				CryptoWithdrawal:                  true,
				TradeFee:                          true,
				FiatDepositFee:                    true,
				FiatWithdrawalFee:                 true,
				CryptoDepositFee:                  true,
				CryptoWithdrawalFee:               true,
				MultiChainDeposits:                true,
				MultiChainWithdrawals:             true,
				MultiChainDepositRequiresChainSet: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(exmoRateInterval, exmoRequestRate, 1)))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot: exmoAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
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
	return e.SetupDefaults(exch)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, asset.ErrNotSupported
	}

	symbols, err := e.GetPairSettings(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, 0, len(symbols))
	for key := range symbols {
		var pair currency.Pair
		pair, err = currency.NewPairFromString(key)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	pairs, err := e.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	if err := e.UpdatePairs(pairs, asset.Spot, false); err != nil {
		return err
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}
	result, err := e.GetTicker(ctx)
	if err != nil {
		return err
	}

	var enabled bool
	for symbol, tick := range result {
		var pair currency.Pair
		pair, enabled, err = e.MatchSymbolCheckEnabled(symbol, asset.Spot, true)
		if err != nil {
			if !errors.Is(err, currency.ErrPairNotFound) {
				return err
			}
		}
		if !enabled {
			continue
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Pair:         pair,
			Last:         tick.Last,
			Ask:          tick.Sell,
			High:         tick.High,
			Bid:          tick.Buy,
			Low:          tick.Low,
			Volume:       tick.Volume,
			LastUpdated:  tick.Updated.Time(),
			ExchangeName: e.Name,
			AssetType:    a,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := e.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	callingBook := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	enabledPairs, err := e.GetEnabledPairs(assetType)
	if err != nil {
		return callingBook, err
	}

	pairsCollated, err := e.FormatExchangeCurrencies(enabledPairs, assetType)
	if err != nil {
		return callingBook, err
	}

	result, err := e.GetOrderbook(ctx, pairsCollated)
	if err != nil {
		return callingBook, err
	}

	for i := range enabledPairs {
		book := &orderbook.Book{
			Exchange:          e.Name,
			Pair:              enabledPairs[i],
			Asset:             assetType,
			ValidateOrderbook: e.ValidateOrderbook,
		}

		curr, err := e.FormatExchangeCurrency(enabledPairs[i], assetType)
		if err != nil {
			return callingBook, err
		}

		data, ok := result[curr.String()]
		if !ok {
			continue
		}

		book.Asks = make(orderbook.Levels, len(data.Asks))
		for y := range data.Asks {
			book.Asks[y].Price = data.Asks[y][0].Float64()
			book.Asks[y].Amount = data.Asks[y][1].Float64()
		}

		book.Bids = make(orderbook.Levels, len(data.Bids))
		for y := range data.Bids {
			book.Bids[y].Price = data.Bids[y][0].Float64()
			book.Bids[y].Amount = data.Bids[y][1].Float64()
		}

		err = book.Process()
		if err != nil {
			return book, err
		}
	}
	return orderbook.Get(e.Name, p, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	resp, err := e.GetUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for k, bal := range resp.Balances {
		avail := bal.Float64()
		reserved := 0.0
		if r, ok := resp.Reserved[k]; ok {
			reserved = r.Float64()
		}
		subAccts[0].Balances.Set(currency.NewCode(k), accounts.Balance{
			Total: avail + reserved,
			Hold:  reserved,
			Free:  avail,
		})
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	hist, err := e.GetWalletHistory(ctx, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(hist.History))
	for i := range hist.History {
		if hist.History[i].Type != "deposit" {
			continue
		}
		resp = append(resp, exchange.FundingHistory{
			Status:     hist.History[i].Status,
			TransferID: hist.History[i].TXID,
			Timestamp:  hist.History[i].Timestamp.Time(),
			Currency:   hist.History[i].Currency,
			Amount:     hist.History[i].Amount,
			BankFrom:   hist.History[i].Provider,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	hist, err := e.GetWalletHistory(ctx, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(hist.History))
	for i := range hist.History {
		if hist.History[i].Type != "withdrawal" {
			continue
		}
		resp = append(resp, exchange.WithdrawalHistory{
			Status:     hist.History[i].Status,
			TransferID: hist.History[i].TXID,
			Timestamp:  hist.History[i].Timestamp.Time(),
			Currency:   hist.History[i].Currency,
			Amount:     hist.History[i].Amount,
			CryptoTxID: hist.History[i].TXID,
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData map[string][]Trades
	tradeData, err = e.GetTrades(ctx, p.String())
	if err != nil {
		return nil, err
	}

	mapData := tradeData[p.String()]
	resp := make([]trade.Data, len(mapData))
	for i := range mapData {
		var side order.Side
		side, err = order.StringToOrderSide(mapData[i].Type)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			Exchange:     e.Name,
			TID:          strconv.FormatInt(mapData[i].TradeID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        mapData[i].Price,
			Amount:       mapData[i].Quantity,
			Timestamp:    mapData[i].Date.Time(),
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

	var orderType string
	switch s.Type {
	case order.Limit:
		return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, s.Type)
	case order.Market:
		if s.Side.IsShort() {
			orderType = "market_sell"
		} else {
			orderType = "market_buy"
		}
	}

	fPair, err := e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	response, err := e.CreateOrder(ctx, fPair.String(), orderType, s.Price, s.Amount)
	if err != nil {
		return nil, err
	}

	return s.DeriveSubmitResponse(strconv.FormatInt(response, 10))
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

	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}

	return e.CancelExistingOrder(ctx, orderIDInt)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	openOrders, err := e.GetOpenOrders(ctx)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range openOrders {
		err = e.CancelExistingOrder(ctx, openOrders[i].OrderID)
		if err != nil {
			cancelAllOrdersResponse.Status[strconv.FormatInt(openOrders[i].OrderID, 10)] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(_ context.Context, _ string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	fullAddr, err := e.GetCryptoDepositAddress(ctx)
	if err != nil {
		return nil, err
	}

	curr := cryptocurrency.Upper().String()
	if chain != "" && !strings.EqualFold(chain, curr) {
		curr += strings.ToUpper(chain)
	}

	addr, ok := fullAddr[curr]
	if !ok {
		chains, err := e.GetAvailableTransferChains(ctx, cryptocurrency)
		if err != nil {
			return nil, err
		}

		if len(chains) > 1 {
			// rather than assume, return an error
			return nil, fmt.Errorf("currency %s has %v chains available, one must be specified", cryptocurrency, chains)
		}
		return nil, fmt.Errorf("deposit address for %s could not be found, please generate via the exmo website", cryptocurrency.String())
	}

	var tag string
	if strings.Contains(addr, ",") {
		split := strings.Split(addr, ",")
		addr, tag = split[0], split[1]
	}

	return &deposit.Address{
		Address: addr,
		Tag:     tag,
		Chain:   chain,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.WithdrawCryptocurrency(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Crypto.Chain,
		withdrawRequest.Amount)

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

	resp, err := e.GetOpenOrders(ctx)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, 0, len(resp))
	for i := range resp {
		symbol, err := currency.NewPairDelimiter(resp[i].Pair, "_")
		if err != nil {
			return nil, err
		}
		side, err := order.StringToOrderSide(resp[i].Type)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order.Detail{
			OrderID:  strconv.FormatInt(resp[i].OrderID, 10),
			Amount:   resp[i].Quantity,
			Date:     resp[i].Created.Time(),
			Price:    resp[i].Price,
			Side:     side,
			Exchange: e.Name,
			Pair:     symbol,
		})
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

	if len(req.Pairs) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allTrades []UserTrades
	for i := range req.Pairs {
		fPair, err := e.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
		if err != nil {
			return nil, err
		}

		resp, err := e.GetUserTrades(ctx, fPair.String(), "", "10000")
		if err != nil {
			return nil, err
		}
		for j := range resp {
			allTrades = append(allTrades, resp[j]...)
		}
	}

	orders := make([]order.Detail, len(allTrades))
	for i := range allTrades {
		pair, err := currency.NewPairDelimiter(allTrades[i].Pair, "_")
		if err != nil {
			return nil, err
		}
		side, err := order.StringToOrderSide(allTrades[i].Type)
		if err != nil {
			return nil, err
		}
		detail := order.Detail{
			OrderID:        strconv.FormatInt(allTrades[i].TradeID, 10),
			Amount:         allTrades[i].Quantity,
			ExecutedAmount: allTrades[i].Quantity,
			Cost:           allTrades[i].Amount,
			CostAsset:      pair.Quote,
			Date:           allTrades[i].Date.Time(),
			Price:          allTrades[i].Price,
			Side:           side,
			Exchange:       e.Name,
			Pair:           pair,
		}
		detail.InferCostsAndTimes()
		orders[i] = detail
	}
	return req.Filter(e.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	chains, err := e.GetCryptoPaymentProvidersList(ctx)
	if err != nil {
		return nil, err
	}

	methods, ok := chains[cryptocurrency.Upper().String()]
	if !ok {
		return nil, fmt.Errorf("%w no available chains for %v", currency.ErrCurrencyNotFound, cryptocurrency)
	}

	var availChains []string
	for x := range methods {
		if methods[x].Type == "deposit" && methods[x].Enabled {
			chain := methods[x].Name
			if strings.Contains(chain, "(") && strings.Contains(chain, ")") {
				chain = chain[strings.Index(chain, "(")+1 : strings.Index(chain, ")")]
			}
			availChains = append(availChains, chain)
		}
	}

	return availChains, nil
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.UnderscoreDelimiter
	return tradeBaseURL + cp.Upper().String() + "/", nil
}
