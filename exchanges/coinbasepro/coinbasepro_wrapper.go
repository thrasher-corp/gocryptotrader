package coinbasepro

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/asset"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (c *CoinbasePro) GetDefaultConfig() (*config.ExchangeConfig, error) {
	c.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = c.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = c.BaseCurrencies

	err := c.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if c.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = c.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default values for the exchange
func (c *CoinbasePro) SetDefaults() {
	c.Name = "CoinbasePro"
	c.Enabled = true
	c.Verbose = true
	c.API.CredentialsValidator.RequiresKey = true
	c.API.CredentialsValidator.RequiresSecret = true
	c.API.CredentialsValidator.RequiresClientID = true
	c.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	c.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Delimiter: "-",
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}

	c.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  false,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.AutoWithdrawFiatWithAPIPermission,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	c.Requester = request.New(c.Name,
		request.NewRateLimit(time.Second, coinbaseproAuthRate),
		request.NewRateLimit(time.Second, coinbaseproUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	c.API.Endpoints.URLDefault = coinbaseproAPIURL
	c.API.Endpoints.URL = c.API.Endpoints.URLDefault
	c.WebsocketInit()
	c.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketOrderbookSupported |
		exchange.WebsocketSubscribeSupported |
		exchange.WebsocketUnsubscribeSupported
}

// Setup initialises the exchange parameters with the current configuration
func (c *CoinbasePro) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		c.SetEnabled(false)
		return nil
	}

	err := c.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return c.WebsocketSetup(c.WsConnect,
		c.Subscribe,
		c.Unsubscribe,
		exch.Name,
		exch.Features.Enabled.Websocket,
		exch.Verbose,
		coinbaseproWebsocketURL,
		exch.API.Endpoints.WebsocketURL)
}

// Start starts the coinbasepro go routine
func (c *CoinbasePro) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
}

// Run implements the coinbasepro wrapper
func (c *CoinbasePro) Run() {
	if c.Verbose {
		log.Debugf(log.SubSystemExchSys, "%s Websocket: %s. (url: %s).\n", c.GetName(), common.IsEnabled(c.Websocket.IsEnabled()), coinbaseproWebsocketURL)
		c.PrintEnabledPairs()
	}

	if !c.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := c.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.SubSystemExchSys, "%s failed to update tradable pairs. Err: %s", c.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (c *CoinbasePro) FetchTradablePairs(asset asset.Item) ([]string, error) {
	pairs, err := c.GetProducts()
	if err != nil {
		return nil, err
	}

	var products []string
	for x := range pairs {
		products = append(products, pairs[x].BaseCurrency+pairs[x].QuoteCurrency)
	}

	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (c *CoinbasePro) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := c.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return c.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// coinbasepro exchange
func (c *CoinbasePro) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = c.GetName()
	accountBalance, err := c.GetAccounts()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance[i].Currency)
		exchangeCurrency.TotalValue = accountBalance[i].Available
		exchangeCurrency.Hold = accountBalance[i].Hold

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *CoinbasePro) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := c.GetTicker(c.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return ticker.Price{}, err
	}

	stats, err := c.GetStats(c.FormatExchangeCurrency(p, assetType).String())

	if err != nil {
		return ticker.Price{}, err
	}

	tickerPrice.Pair = p
	tickerPrice.Volume = stats.Volume
	tickerPrice.Last = tick.Price
	tickerPrice.High = stats.High
	tickerPrice.Low = stats.Low

	err = ticker.ProcessTicker(c.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(c.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (c *CoinbasePro) FetchTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (c *CoinbasePro) FetchOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	ob, err := orderbook.Get(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *CoinbasePro) UpdateOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := c.GetOrderbook(c.FormatExchangeCurrency(p, assetType).String(), 2)
	if err != nil {
		return orderBook, err
	}

	obNew := orderbookNew.(OrderbookL1L2)

	for x := range obNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: obNew.Bids[x].Amount, Price: obNew.Bids[x].Price})
	}

	for x := range obNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: obNew.Asks[x].Amount, Price: obNew.Asks[x].Price})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = c.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(c.Name, p, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (c *CoinbasePro) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (c *CoinbasePro) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (c *CoinbasePro) SubmitOrder(order *exchange.OrderSubmission) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	if order == nil {
		return submitOrderResponse, exchange.ErrOrderSubmissionIsNil
	}

	if err := order.Validate(); err != nil {
		return submitOrderResponse, err
	}

	var response string
	var err error
	switch order.OrderType {
	case exchange.MarketOrderType:
		response, err = c.PlaceMarketOrder("",
			order.Amount,
			order.Amount,
			order.OrderSide.ToString(),
			order.Pair.String(),
			"")
	case exchange.LimitOrderType:
		response, err = c.PlaceLimitOrder("",
			order.Price,
			order.Amount,
			order.OrderSide.ToString(),
			"",
			"",
			order.Pair.String(),
			"",
			false)
	default:
		err = errors.New("order type not supported")
	}

	if response != "" {
		submitOrderResponse.OrderID = response
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *CoinbasePro) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *CoinbasePro) CancelOrder(order *exchange.OrderCancellation) error {
	return c.CancelExistingOrder(order.OrderID)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *CoinbasePro) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	// CancellAllExisting orders returns a list of successful cancellations, we're only interested in failures
	_, err := c.CancelAllExistingOrders("")
	return exchange.CancelAllOrdersResponse{}, err
}

// GetOrderInfo returns information on a current open order
func (c *CoinbasePro) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *CoinbasePro) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.CryptoWithdrawRequest) (string, error) {
	resp, err := c.WithdrawCrypto(withdrawRequest.Amount, withdrawRequest.Currency.String(), withdrawRequest.Address)
	return resp.ID, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawFiatFunds(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	paymentMethods, err := c.GetPayMethods()
	if err != nil {
		return "", err
	}

	selectedWithdrawalMethod := PaymentMethod{}
	for i := range paymentMethods {
		if withdrawRequest.BankName == paymentMethods[i].Name {
			selectedWithdrawalMethod = paymentMethods[i]
			break
		}
	}
	if selectedWithdrawalMethod.ID == "" {
		return "", fmt.Errorf("could not find payment method '%v'. Check the name via the website and try again", withdrawRequest.BankName)
	}

	resp, err := c.WithdrawViaPaymentMethod(withdrawRequest.Amount, withdrawRequest.Currency.String(), selectedWithdrawalMethod.ID)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *CoinbasePro) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return c.WithdrawFiatFunds(withdrawRequest)
}

// GetWebsocket returns a pointer to the exchange websocket
func (c *CoinbasePro) GetWebsocket() (*exchange.Websocket, error) {
	return c.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *CoinbasePro) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !c.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return c.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (c *CoinbasePro) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var respOrders []GeneralizedOrderResponse
	for i := range getOrdersRequest.Currencies {
		resp, err := c.GetOrders([]string{"open", "pending", "active"},
			c.FormatExchangeCurrency(getOrdersRequest.Currencies[i], asset.Spot).String())
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for i := range respOrders {
		currency := currency.NewPairDelimiter(respOrders[i].ProductID,
			c.CurrencyPairs.Get(asset.Spot).ConfigFormat.Delimiter)
		orderSide := exchange.OrderSide(strings.ToUpper(respOrders[i].Side))
		orderType := exchange.OrderType(strings.ToUpper(respOrders[i].Type))
		orderDate, err := time.Parse(time.RFC3339, respOrders[i].CreatedAt)
		if err != nil {
			log.Warnf(log.SubSystemExchSys, "Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				c.Name, "GetActiveOrders", respOrders[i].ID, respOrders[i].CreatedAt)
		}

		orders = append(orders, exchange.OrderDetail{
			ID:             respOrders[i].ID,
			Amount:         respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			OrderType:      orderType,
			OrderDate:      orderDate,
			OrderSide:      orderSide,
			CurrencyPair:   currency,
			Exchange:       c.Name,
		})
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *CoinbasePro) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var respOrders []GeneralizedOrderResponse
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := c.GetOrders([]string{"done", "settled"},
			c.FormatExchangeCurrency(currency, asset.Spot).String())
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for i := range respOrders {
		currency := currency.NewPairDelimiter(respOrders[i].ProductID,
			c.CurrencyPairs.Get(asset.Spot).ConfigFormat.Delimiter)
		orderSide := exchange.OrderSide(strings.ToUpper(respOrders[i].Side))
		orderType := exchange.OrderType(strings.ToUpper(respOrders[i].Type))
		orderDate, err := time.Parse(time.RFC3339, respOrders[i].CreatedAt)
		if err != nil {
			log.Warnf(log.SubSystemExchSys, "Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				c.Name, "GetActiveOrders", respOrders[i].ID, respOrders[i].CreatedAt)
		}

		orders = append(orders, exchange.OrderDetail{
			ID:             respOrders[i].ID,
			Amount:         respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			OrderType:      orderType,
			OrderDate:      orderDate,
			OrderSide:      orderSide,
			CurrencyPair:   currency,
			Exchange:       c.Name,
		})
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (c *CoinbasePro) SubscribeToWebsocketChannels(channels []exchange.WebsocketChannelSubscription) error {
	c.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (c *CoinbasePro) UnsubscribeToWebsocketChannels(channels []exchange.WebsocketChannelSubscription) error {
	c.Websocket.UnsubscribeToChannels(channels)
	return nil
}
