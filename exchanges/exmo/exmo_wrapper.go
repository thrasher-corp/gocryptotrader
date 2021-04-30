package exmo

import (
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
func (e *EXMO) GetDefaultConfig() (*config.ExchangeConfig, error) {
	e.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = e.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = e.BaseCurrencies

	err := e.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if e.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = e.UpdateTradablePairs(true)
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
				TickerBatching:      true,
				TickerFetching:      true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				SubmitOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	e.Requester = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(exmoRateInterval, exmoRequestRate)))
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot: exmoAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// Setup takes in the supplied exchange configuration details and sets params
func (e *EXMO) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	return e.SetupDefaults(exch)
}

// Start starts the EXMO go routine
func (e *EXMO) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		e.Run()
		wg.Done()
	}()
}

// Run implements the EXMO wrapper
func (e *EXMO) Run() {
	if e.Verbose {
		e.PrintEnabledPairs()
	}

	if !e.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := e.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", e.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *EXMO) FetchTradablePairs(asset asset.Item) ([]string, error) {
	pairs, err := e.GetPairSettings()
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
func (e *EXMO) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := e.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}

	return e.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *EXMO) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	result, err := e.GetTicker()
	if err != nil {
		return nil, err
	}
	if _, ok := result[p.String()]; !ok {
		return nil, err
	}
	pairs, err := e.GetEnabledPairs(assetType)
	if err != nil {
		return nil, err
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
				AssetType:    assetType})
			if err != nil {
				return nil, err
			}
		}
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (e *EXMO) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := ticker.GetTicker(e.Name, p, assetType)
	if err != nil {
		return e.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (e *EXMO) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(e.Name, p, assetType)
	if err != nil {
		return e.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *EXMO) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
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

	result, err := e.GetOrderbook(pairsCollated)
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
func (e *EXMO) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = e.Name
	result, err := e.GetUserInfo()
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for x, y := range result.Balances {
		var exchangeCurrency account.Balance
		exchangeCurrency.CurrencyName = currency.NewCode(x)
		for z, w := range result.Reserved {
			if z == x {
				avail, _ := strconv.ParseFloat(y, 64)
				reserved, _ := strconv.ParseFloat(w, 64)
				exchangeCurrency.TotalValue = avail + reserved
				exchangeCurrency.Hold = reserved
			}
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
func (e *EXMO) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(e.Name, assetType)
	if err != nil {
		return e.UpdateAccountInfo(assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (e *EXMO) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *EXMO) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *EXMO) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData map[string][]Trades
	tradeData, err = e.GetTrades(p.String())
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
func (e *EXMO) GetHistoricTrades(_ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (e *EXMO) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
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

	response, err := e.CreateOrder(fPair.String(), oT, s.Price, s.Amount)
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
func (e *EXMO) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *EXMO) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	return e.CancelExistingOrder(orderIDInt)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *EXMO) CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *EXMO) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	openOrders, err := e.GetOpenOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range openOrders {
		err = e.CancelExistingOrder(openOrders[i].OrderID)
		if err != nil {
			cancelAllOrdersResponse.Status[strconv.FormatInt(openOrders[i].OrderID, 10)] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (e *EXMO) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *EXMO) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	fullAddr, err := e.GetCryptoDepositAddress()
	if err != nil {
		return "", err
	}

	addr, ok := fullAddr[cryptocurrency.String()]
	if !ok {
		return "", fmt.Errorf("currency %s could not be found, please generate via the exmo website", cryptocurrency.String())
	}

	return addr, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *EXMO) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.WithdrawCryptocurrency(withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Amount)

	return &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(resp, 10),
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *EXMO) WithdrawFiatFunds(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *EXMO) WithdrawFiatFundsToInternationalBank(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *EXMO) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !e.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (e *EXMO) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := e.GetOpenOrders()
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
func (e *EXMO) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
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

		resp, err := e.GetUserTrades(fpair.String(), "", "10000")
		if err != nil {
			return nil, err
		}
		for j := range resp {
			allTrades = append(allTrades, resp[j]...)
		}
	}

	var orders []order.Detail
	for i := range allTrades {
		symbol, err := currency.NewPairDelimiter(allTrades[i].Pair, "_")
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(allTrades[i].Date, 0)
		orderSide := order.Side(strings.ToUpper(allTrades[i].Type))
		orders = append(orders, order.Detail{
			ID:       strconv.FormatInt(allTrades[i].TradeID, 10),
			Amount:   allTrades[i].Quantity,
			Date:     orderDate,
			Price:    allTrades[i].Price,
			Side:     orderSide,
			Exchange: e.Name,
			Pair:     symbol,
		})
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (e *EXMO) ValidateCredentials(assetType asset.Item) error {
	_, err := e.UpdateAccountInfo(assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *EXMO) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *EXMO) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
