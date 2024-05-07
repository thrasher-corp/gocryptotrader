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

// SetDefaults sets the basic defaults for exmo
func (e *EXMO) SetDefaults() {
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
		request.WithLimiter(request.NewBasicRateLimit(exmoRateInterval, exmoRequestRate)))
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
func (e *EXMO) Setup(exch *config.Exchange) error {
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
func (e *EXMO) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
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
func (e *EXMO) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := e.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	err = e.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
	if err != nil {
		return err
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *EXMO) UpdateTickers(ctx context.Context, a asset.Item) error {
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
			LastUpdated:  time.Unix(tick.Updated, 0),
			ExchangeName: e.Name,
			AssetType:    a})
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *EXMO) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := e.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (e *EXMO) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := ticker.GetTicker(e.Name, p, assetType)
	if err != nil {
		return e.UpdateTicker(ctx, p, assetType)
	}
	return tick, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (e *EXMO) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(e.Name, p, assetType)
	if err != nil {
		return e.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *EXMO) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	callingBook := &orderbook.Base{
		Exchange:        e.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: e.CanVerifyOrderbook,
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
		book := &orderbook.Base{
			Exchange:        e.Name,
			Pair:            enabledPairs[i],
			Asset:           assetType,
			VerifyOrderbook: e.CanVerifyOrderbook,
		}

		curr, err := e.FormatExchangeCurrency(enabledPairs[i], assetType)
		if err != nil {
			return callingBook, err
		}

		data, ok := result[curr.String()]
		if !ok {
			continue
		}

		book.Asks = make(orderbook.Items, len(data.Ask))
		for y := range data.Ask {
			var price, amount float64
			price, err = strconv.ParseFloat(data.Ask[y][0], 64)
			if err != nil {
				return book, err
			}

			amount, err = strconv.ParseFloat(data.Ask[y][1], 64)
			if err != nil {
				return book, err
			}

			book.Asks[y] = orderbook.Item{
				Price:  price,
				Amount: amount,
			}
		}

		book.Bids = make(orderbook.Items, len(data.Bid))
		for y := range data.Bid {
			var price, amount float64
			price, err = strconv.ParseFloat(data.Bid[y][0], 64)
			if err != nil {
				return book, err
			}

			amount, err = strconv.ParseFloat(data.Bid[y][1], 64)
			if err != nil {
				return book, err
			}

			book.Bids[y] = orderbook.Item{
				Price:  price,
				Amount: amount,
			}
		}

		err = book.Process()
		if err != nil {
			return book, err
		}
	}
	return orderbook.Get(e.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Exmo exchange
func (e *EXMO) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = e.Name
	result, err := e.GetUserInfo(ctx)
	if err != nil {
		return response, err
	}

	currencies := make([]account.Balance, 0, len(result.Balances))
	for x, y := range result.Balances {
		var exchangeCurrency account.Balance
		exchangeCurrency.Currency = currency.NewCode(x)
		for z, w := range result.Reserved {
			if z != x {
				continue
			}
			var avail, reserved float64
			avail, err = strconv.ParseFloat(y, 64)
			if err != nil {
				return response, err
			}
			reserved, err = strconv.ParseFloat(w, 64)
			if err != nil {
				return response, err
			}
			exchangeCurrency.Total = avail + reserved
			exchangeCurrency.Hold = reserved
			exchangeCurrency.Free = avail
		}
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, account.SubAccount{
		AssetType:  assetType,
		Currencies: currencies,
	})

	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&response, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (e *EXMO) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(e.Name, creds, assetType)
	if err != nil {
		return e.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *EXMO) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
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
			Timestamp:  time.Unix(hist.History[i].Timestamp, 0),
			Currency:   hist.History[i].Currency,
			Amount:     hist.History[i].Amount,
			BankFrom:   hist.History[i].Provider,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *EXMO) GetWithdrawalsHistory(ctx context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
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
			Timestamp:  time.Unix(hist.History[i].Timestamp, 0),
			Currency:   hist.History[i].Currency,
			Amount:     hist.History[i].Amount,
			CryptoTxID: hist.History[i].TXID,
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *EXMO) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
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
			Timestamp:    time.Unix(mapData[i].Date, 0),
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
func (e *EXMO) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (e *EXMO) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
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

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (e *EXMO) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *EXMO) CancelOrder(ctx context.Context, o *order.Cancel) error {
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
func (e *EXMO) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (e *EXMO) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *EXMO) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
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
func (e *EXMO) GetOrderInfo(_ context.Context, _ string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *EXMO) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
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
func (e *EXMO) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
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
func (e *EXMO) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *EXMO) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *EXMO) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
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
func (e *EXMO) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
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
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(resp[i].Pair, "_")
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(resp[i].Created, 0)
		var side order.Side
		side, err = order.StringToOrderSide(resp[i].Type)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order.Detail{
			OrderID:  strconv.FormatInt(resp[i].OrderID, 10),
			Amount:   resp[i].Quantity,
			Date:     orderDate,
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
func (e *EXMO) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
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
		orderDate := time.Unix(allTrades[i].Date, 0)
		var side order.Side
		side, err = order.StringToOrderSide(allTrades[i].Type)
		if err != nil {
			return nil, err
		}
		detail := order.Detail{
			OrderID:        strconv.FormatInt(allTrades[i].TradeID, 10),
			Amount:         allTrades[i].Quantity,
			ExecutedAmount: allTrades[i].Quantity,
			Cost:           allTrades[i].Amount,
			CostAsset:      pair.Quote,
			Date:           orderDate,
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
func (e *EXMO) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountInfo(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *EXMO) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *EXMO) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (e *EXMO) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
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
func (e *EXMO) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (e *EXMO) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *EXMO) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *EXMO) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.UnderscoreDelimiter
	return tradeBaseURL + cp.Upper().String() + "/", nil
}
