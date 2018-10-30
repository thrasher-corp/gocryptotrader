package exmo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
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
	e.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithSetup |
		exchange.NoFiatWithdrawals

	e.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
			Separator: ",",
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  false,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	e.Requester = request.New(e.Name,
		request.NewRateLimit(time.Minute, exmoAuthRate),
		request.NewRateLimit(time.Minute, exmoUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	e.API.Endpoints.URLDefault = exmoAPIURL
	e.API.Endpoints.URL = e.API.Endpoints.URLDefault
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
		log.Errorf("%s failed to update tradable pairs. Err: %s", e.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *EXMO) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
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
	pairs, err := e.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return e.UpdatePairs(currency.NewPairsFromStrings(pairs), assets.AssetTypeSpot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *EXMO) UpdateTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	pairsCollated, err := e.FormatExchangeCurrencies(e.GetEnabledPairs(assetType), assetType)
	if err != nil {
		return tickerPrice, err
	}

	result, err := e.GetTicker(pairsCollated)
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range e.GetEnabledPairs(assetType) {
		currency := e.FormatExchangeCurrency(x, assetType).String()
		var tickerPrice ticker.Price
		tickerPrice.Pair = x
		tickerPrice.Last = result[currency].Last
		tickerPrice.Ask = result[currency].Sell
		tickerPrice.High = result[currency].High
		tickerPrice.Bid = result[currency].Buy
		tickerPrice.Last = result[currency].Last
		tickerPrice.Low = result[currency].Low
		tickerPrice.Volume = result[currency].Volume

		err = ticker.ProcessTicker(e.Name, &tickerPrice, assetType)
		if err != nil {
			return tickerPrice, err
		}
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (e *EXMO) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tick, err := ticker.GetTicker(e.GetName(), p, assetType)
	if err != nil {
		return e.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (e *EXMO) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(e.GetName(), p, assetType)
	if err != nil {
		return e.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *EXMO) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	pairsCollated, err := e.FormatExchangeCurrencies(e.GetEnabledPairs(assetType), assetType)
	if err != nil {
		return orderBook, err
	}

	result, err := e.GetOrderbook(pairsCollated)
	if err != nil {
		return orderBook, err
	}

	for _, x := range e.GetEnabledPairs(assetType) {
		currency := e.FormatExchangeCurrency(x, assetType)
		data, ok := result[currency.String()]
		if !ok {
			continue
		}

		var obItems []orderbook.Item
		for y := range data.Ask {
			z := data.Ask[y]
			price, _ := strconv.ParseFloat(z[0], 64)
			amount, _ := strconv.ParseFloat(z[1], 64)
			obItems = append(obItems, orderbook.Item{Price: price, Amount: amount})
		}

		orderBook.Asks = obItems
		obItems = []orderbook.Item{}
		for y := range data.Bid {
			z := data.Bid[y]
			price, _ := strconv.ParseFloat(z[0], 64)
			amount, _ := strconv.ParseFloat(z[1], 64)
			obItems = append(obItems, orderbook.Item{Price: price, Amount: amount})
		}

		orderBook.Bids = obItems
		orderBook.Pair = x
		orderBook.ExchangeName = e.GetName()
		orderBook.AssetType = assetType

		err = orderBook.Process()
		if err != nil {
			return orderBook, err
		}
	}
	return orderbook.Get(e.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Exmo exchange
func (e *EXMO) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = e.GetName()
	result, err := e.GetUserInfo()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for x, y := range result.Balances {
		var exchangeCurrency exchange.AccountCurrencyInfo
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

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (e *EXMO) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (e *EXMO) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (e *EXMO) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var oT string

	switch orderType {
	case exchange.LimitOrderType:
		return submitOrderResponse, errors.New("unsupported order type")
	case exchange.MarketOrderType:
		oT = "market_buy"
		if side == exchange.SellOrderSide {
			oT = "market_sell"
		}
	default:
		return submitOrderResponse, errors.New("unsupported order type")
	}

	response, err := e.CreateOrder(p.String(), oT, price, amount)

	if response > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (e *EXMO) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *EXMO) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	return e.CancelExistingOrder(orderIDInt)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *EXMO) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := e.GetOpenOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, order := range openOrders {
		err = e.CancelExistingOrder(order.OrderID)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(order.OrderID, 10)] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (e *EXMO) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
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
func (e *EXMO) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	resp, err := e.WithdrawCryptocurrency(withdrawRequest.Currency.String(), withdrawRequest.Address, withdrawRequest.AddressTag, withdrawRequest.Amount)

	return fmt.Sprintf("%v", resp), err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *EXMO) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *EXMO) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (e *EXMO) GetWebsocket() (*exchange.Websocket, error) {
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
func (e *EXMO) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := e.GetOpenOrders()
	if err != nil {
		return nil, err
	}
	var orders []exchange.OrderDetail
	for _, order := range resp {
		symbol := currency.NewPairDelimiter(order.Pair, "_")
		orderDate := time.Unix(order.Created, 0)
		orderSide := exchange.OrderSide(strings.ToUpper(order.Type))
		orders = append(orders, exchange.OrderDetail{
			ID:           fmt.Sprintf("%v", order.OrderID),
			Amount:       order.Quantity,
			OrderDate:    orderDate,
			Price:        order.Price,
			OrderSide:    orderSide,
			Exchange:     e.Name,
			CurrencyPair: symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *EXMO) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allTrades []UserTrades
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := e.GetUserTrades(e.FormatExchangeCurrency(currency, assets.AssetTypeSpot).String(), "", "10000")
		if err != nil {
			return nil, err
		}
		for _, order := range resp {
			allTrades = append(allTrades, order...)
		}
	}

	var orders []exchange.OrderDetail
	for _, order := range allTrades {
		symbol := currency.NewPairDelimiter(order.Pair, "_")
		orderDate := time.Unix(order.Date, 0)
		orderSide := exchange.OrderSide(strings.ToUpper(order.Type))
		orders = append(orders, exchange.OrderDetail{
			ID:           fmt.Sprintf("%v", order.TradeID),
			Amount:       order.Quantity,
			OrderDate:    orderDate,
			Price:        order.Price,
			OrderSide:    orderSide,
			Exchange:     e.Name,
			CurrencyPair: symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}
