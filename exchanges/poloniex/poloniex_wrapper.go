package poloniex

import (
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
func (p *Poloniex) GetDefaultConfig() (*config.ExchangeConfig, error) {
	p.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = p.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = p.BaseCurrencies

	err := p.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if p.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = p.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default settings for poloniex
func (p *Poloniex) SetDefaults() {
	p.Name = "Poloniex"
	p.Enabled = true
	p.Verbose = true
	p.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithAPIPermission |
		exchange.NoFiatWithdrawals
	p.API.CredentialsValidator.RequiresKey = true
	p.API.CredentialsValidator.RequiresSecret = true

	p.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
	}

	p.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	p.Requester = request.New(p.Name,
		request.NewRateLimit(time.Second, poloniexAuthRate),
		request.NewRateLimit(time.Second, poloniexUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	p.API.Endpoints.URLDefault = poloniexAPIURL
	p.API.Endpoints.URL = p.API.Endpoints.URLDefault
	p.WebsocketInit()
	p.Websocket.Functionality = exchange.WebsocketTradeDataSupported |
		exchange.WebsocketOrderbookSupported |
		exchange.WebsocketTickerSupported
}

// Setup sets user exchange configuration settings
func (p *Poloniex) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		p.SetEnabled(false)
		return nil
	}

	err := p.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return p.WebsocketSetup(p.WsConnect,
		exch.Name,
		exch.Features.Enabled.Websocket,
		poloniexWebsocketAddress,
		exch.API.Endpoints.WebsocketURL)
}

// Start starts the Poloniex go routine
func (p *Poloniex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		p.Run()
		wg.Done()
	}()
}

// Run implements the Poloniex wrapper
func (p *Poloniex) Run() {
	if p.Verbose {
		log.Debugf("%s Websocket: %s (url: %s).\n", p.GetName(), common.IsEnabled(p.Websocket.IsEnabled()), poloniexWebsocketAddress)
		p.PrintEnabledPairs()
	}

	forceUpdate := false
	if common.StringDataCompare(p.GetAvailablePairs(assets.AssetTypeSpot).Strings(), "BTC_USDT") {
		log.Warnf("%s contains invalid pair, forcing upgrade of available currencies.\n",
			p.Name)
		forceUpdate = true
	}

	if !p.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := p.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", p.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (p *Poloniex) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	resp, err := p.GetTicker()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range resp {
		currencies = append(currencies, x)
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (p *Poloniex) UpdateTradablePairs(forceUpgrade bool) error {
	pairs, err := p.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return p.UpdatePairs(currency.NewPairsFromStrings(pairs), assets.AssetTypeSpot, false, forceUpgrade)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (p *Poloniex) UpdateTicker(currencyPair currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := p.GetTicker()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range p.GetEnabledPairs(assetType) {
		var tp ticker.Price
		curr := p.FormatExchangeCurrency(x, assetType).String()
		tp.Pair = x
		tp.Ask = tick[curr].LowestAsk
		tp.Bid = tick[curr].HighestBid
		tp.High = tick[curr].High24Hr
		tp.Last = tick[curr].Last
		tp.Low = tick[curr].Low24Hr
		tp.Volume = tick[curr].BaseVolume

		err = ticker.ProcessTicker(p.GetName(), &tp, assetType)
		if err != nil {
			return tickerPrice, err
		}
	}
	return ticker.GetTicker(p.Name, currencyPair, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (p *Poloniex) FetchTicker(currencyPair currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(p.GetName(), currencyPair, assetType)
	if err != nil {
		return p.UpdateTicker(currencyPair, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (p *Poloniex) FetchOrderbook(currencyPair currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(p.GetName(), currencyPair, assetType)
	if err != nil {
		return p.UpdateOrderbook(currencyPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (p *Poloniex) UpdateOrderbook(currencyPair currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := p.GetOrderbook("", 1000)
	if err != nil {
		return orderBook, err
	}

	for _, x := range p.GetEnabledPairs(assetType) {
		currency := p.FormatExchangeCurrency(x, assetType).String()
		data, ok := orderbookNew.Data[currency]
		if !ok {
			continue
		}

		var obItems []orderbook.Item
		for y := range data.Bids {
			obData := data.Bids[y]
			obItems = append(obItems,
				orderbook.Item{Amount: obData.Amount, Price: obData.Price})
		}

		orderBook.Bids = obItems
		obItems = []orderbook.Item{}
		for y := range data.Asks {
			obData := data.Asks[y]
			obItems = append(obItems,
				orderbook.Item{Amount: obData.Amount, Price: obData.Price})
		}

		orderBook.Pair = x
		orderBook.Asks = obItems
		orderBook.ExchangeName = p.GetName()
		orderBook.AssetType = assetType

		err = orderBook.Process()
		if err != nil {
			return orderBook, err
		}
	}
	return orderbook.Get(p.Name, currencyPair, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Poloniex exchange
func (p *Poloniex) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = p.GetName()
	accountBalance, err := p.GetBalances()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for x, y := range accountBalance.Currency {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(x)
		exchangeCurrency.TotalValue = y
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (p *Poloniex) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (p *Poloniex) GetExchangeHistory(currencyPair currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (p *Poloniex) SubmitOrder(currencyPair currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	fillOrKill := orderType == exchange.MarketOrderType
	isBuyOrder := side == exchange.BuyOrderSide

	response, err := p.PlaceOrder(currencyPair.String(),
		price,
		amount,
		false,
		fillOrKill,
		isBuyOrder)

	if response.OrderNumber > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response.OrderNumber)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (p *Poloniex) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	oID, err := strconv.ParseInt(action.OrderID, 10, 64)
	if err != nil {
		return "", err
	}

	resp, err := p.MoveOrder(oID,
		action.Price,
		action.Amount,
		action.PostOnly,
		action.ImmediateOrCancel)
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(resp.OrderNumber, 10), nil
}

// CancelOrder cancels an order by its corresponding ID number
func (p *Poloniex) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	_, err = p.CancelExistingOrder(orderIDInt)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (p *Poloniex) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := p.GetOpenOrdersForAllCurrencies()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, openOrderPerCurrency := range openOrders.Data {
		for _, openOrder := range openOrderPerCurrency {
			_, err = p.CancelExistingOrder(openOrder.OrderNumber)
			if err != nil {
				cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(openOrder.OrderNumber, 10)] = err.Error()
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (p *Poloniex) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (p *Poloniex) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	a, err := p.GetDepositAddresses()
	if err != nil {
		return "", err
	}

	address, ok := a.Addresses[cryptocurrency.Upper().String()]
	if !ok {
		return "", fmt.Errorf("cannot find deposit address for %s",
			cryptocurrency)
	}

	return address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (p *Poloniex) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	_, err := p.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.Address, withdrawRequest.Amount)
	return "", err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (p *Poloniex) GetWebsocket() (*exchange.Websocket, error) {
	return p.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (p *Poloniex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !p.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return p.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (p *Poloniex) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := p.GetOpenOrdersForAllCurrencies()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for currencyPair, openOrders := range resp.Data {
		symbol := currency.NewPairDelimiter(currencyPair,
			p.CurrencyPairs.ConfigFormat.Delimiter)

		for _, order := range openOrders {
			orderSide := exchange.OrderSide(strings.ToUpper(order.Type))
			orderDate, err := time.Parse(poloniexDateLayout, order.Date)
			if err != nil {
				log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
					p.Name, "GetActiveOrders", order.OrderNumber, order.Date)
			}

			orders = append(orders, exchange.OrderDetail{
				ID:           fmt.Sprintf("%v", order.OrderNumber),
				OrderSide:    orderSide,
				Amount:       order.Amount,
				OrderDate:    orderDate,
				Price:        order.Rate,
				CurrencyPair: symbol,
				Exchange:     p.Name,
			})
		}
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (p *Poloniex) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := p.GetAuthenticatedTradeHistory(getOrdersRequest.StartTicks.Unix(),
		getOrdersRequest.EndTicks.Unix(),
		10000)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for currencyPair, historicOrders := range resp.Data {
		symbol := currency.NewPairDelimiter(currencyPair,
			p.CurrencyPairs.ConfigFormat.Delimiter)

		for _, order := range historicOrders {
			orderSide := exchange.OrderSide(strings.ToUpper(order.Type))
			orderDate, err := time.Parse(poloniexDateLayout, order.Date)
			if err != nil {
				log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
					p.Name, "GetActiveOrders", order.OrderNumber, order.Date)
			}

			orders = append(orders, exchange.OrderDetail{
				ID:           fmt.Sprintf("%v", order.GlobalTradeID),
				OrderSide:    orderSide,
				Amount:       order.Amount,
				OrderDate:    orderDate,
				Price:        order.Rate,
				CurrencyPair: symbol,
				Exchange:     p.Name,
			})
		}
	}

	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}
