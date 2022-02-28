package exmo

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
)

// GetDefaultConfig returns a default exchange config
func (e *EXMO) GetDefaultConfig() (*config.Exchange, error) {
	e.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = e.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = e.BaseCurrencies

	err := e.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if e.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = e.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

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

// Start starts the EXMO go routine
func (e *EXMO) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		e.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the EXMO wrapper
func (e *EXMO) Run() {
	if e.Verbose {
		e.PrintEnabledPairs()
	}

	if !e.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := e.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", e.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *EXMO) FetchTradablePairs(ctx context.Context, asset asset.Item) ([]string, error) {
	pairs, err := e.GetPairSettings(ctx)
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range pairs {
		currencies = append(currencies, x)
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *EXMO) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := e.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}

	return e.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *EXMO) UpdateTickers(ctx context.Context, a asset.Item) error {
	result, err := e.GetTicker(ctx)
	if err != nil {
		return err
	}
	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	for i := range pairs {
		for j := range result {
			if !strings.EqualFold(pairs[i].String(), j) {
				continue
			}

			err = ticker.ProcessTicker(&ticker.Price{
				Pair:         pairs[i],
				Last:         result[j].Last,
				Ask:          result[j].Sell,
				High:         result[j].High,
				Bid:          result[j].Buy,
				Low:          result[j].Low,
				Volume:       result[j].Volume,
				ExchangeName: e.Name,
				AssetType:    a})
			if err != nil {
				return err
			}
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

			book.Asks = append(book.Asks, orderbook.Item{
				Price:  price,
				Amount: amount,
			})
		}

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

			book.Bids = append(book.Bids, orderbook.Item{
				Price:  price,
				Amount: amount,
			})
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

	var currencies []account.Balance
	for x, y := range result.Balances {
		var exchangeCurrency account.Balance
		exchangeCurrency.CurrencyName = currency.NewCode(x)
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
		Currencies: currencies,
	})

	err = account.Process(&response)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (e *EXMO) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(e.Name, assetType)
	if err != nil {
		return e.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (e *EXMO) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *EXMO) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
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
	var resp []trade.Data
	mapData := tradeData[p.String()]
	for i := range mapData {
		var side order.Side
		side, err = order.StringToOrderSide(mapData[i].Type)
		if err != nil {
			return nil, err
		}
		resp = append(resp, trade.Data{
			Exchange:     e.Name,
			TID:          strconv.FormatInt(mapData[i].TradeID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        mapData[i].Price,
			Amount:       mapData[i].Quantity,
			Timestamp:    time.Unix(mapData[i].Date, 0),
		})
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
func (e *EXMO) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	var oT string
	switch s.Type {
	case order.Limit:
		return submitOrderResponse, errors.New("unsupported order type")
	case order.Market:
		if s.Side == order.Sell {
			oT = "market_sell"
		} else {
			oT = "market_buy"
		}
	}

	fPair, err := e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return submitOrderResponse, err
	}

	response, err := e.CreateOrder(ctx, fPair.String(), oT, s.Price, s.Amount)
	if err != nil {
		return submitOrderResponse, err
	}
	if response > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response, 10)
	}

	submitOrderResponse.IsOrderPlaced = true
	if s.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (e *EXMO) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *EXMO) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	return e.CancelExistingOrder(ctx, orderIDInt)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *EXMO) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
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
func (e *EXMO) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
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
func (e *EXMO) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := e.GetOpenOrders(ctx)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(resp[i].Pair, "_")
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(resp[i].Created, 0)
		orderSide := order.Side(strings.ToUpper(resp[i].Type))
		orders = append(orders, order.Detail{
			ID:       strconv.FormatInt(resp[i].OrderID, 10),
			Amount:   resp[i].Quantity,
			Date:     orderDate,
			Price:    resp[i].Price,
			Side:     orderSide,
			Exchange: e.Name,
			Pair:     symbol,
		})
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *EXMO) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if len(req.Pairs) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allTrades []UserTrades
	for i := range req.Pairs {
		fpair, err := e.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
		if err != nil {
			return nil, err
		}

		resp, err := e.GetUserTrades(ctx, fpair.String(), "", "10000")
		if err != nil {
			return nil, err
		}
		for j := range resp {
			allTrades = append(allTrades, resp[j]...)
		}
	}

	var orders []order.Detail
	for i := range allTrades {
		pair, err := currency.NewPairDelimiter(allTrades[i].Pair, "_")
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(allTrades[i].Date, 0)
		orderSide := order.Side(strings.ToUpper(allTrades[i].Type))
		detail := order.Detail{
			ID:             strconv.FormatInt(allTrades[i].TradeID, 10),
			Amount:         allTrades[i].Quantity,
			ExecutedAmount: allTrades[i].Quantity,
			Cost:           allTrades[i].Amount,
			CostAsset:      pair.Quote,
			Date:           orderDate,
			Price:          allTrades[i].Price,
			Side:           orderSide,
			Exchange:       e.Name,
			Pair:           pair,
		}
		detail.InferCostsAndTimes()
		orders = append(orders, detail)
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (e *EXMO) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountInfo(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *EXMO) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *EXMO) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
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
		return nil, errors.New("no available chains")
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
