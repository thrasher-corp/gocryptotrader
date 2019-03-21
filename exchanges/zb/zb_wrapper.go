package zb

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the OKEX go routine
func (z *ZB) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		z.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (z *ZB) Run() {
	if z.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", z.GetName(), common.IsEnabled(z.Websocket.IsEnabled()), z.WebsocketURL)
		log.Debugf("%s polling delay: %ds.\n", z.GetName(), z.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", z.GetName(), len(z.EnabledPairs), z.EnabledPairs)
	}

	markets, err := z.GetMarkets()
	if err != nil {
		log.Errorf("%s Unable to fetch symbols.\n", z.GetName())
	} else {
		var currencies []string
		for x := range markets {
			currencies = append(currencies, x)
		}

		var newCurrencies currency.Pairs
		for _, p := range currencies {
			newCurrencies = append(newCurrencies,
				currency.NewPairFromString(p))
		}

		err = z.UpdateCurrencies(newCurrencies, false, false)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", z.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (z *ZB) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price

	result, err := z.GetTickers()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range z.GetEnabledCurrencies() {
		currencySplit := common.SplitStrings(exchange.FormatExchangeCurrency(z.Name, x).String(), "_")
		currency := currencySplit[0] + currencySplit[1]
		var tp ticker.Price
		tp.Pair = x
		tp.High = result[currency].High
		tp.Last = result[currency].Last
		tp.Ask = result[currency].Sell
		tp.Bid = result[currency].Buy
		tp.Last = result[currency].Last
		tp.Low = result[currency].Low
		tp.Volume = result[currency].Vol

		err = ticker.ProcessTicker(z.Name, tp, assetType)
		if err != nil {
			return tickerPrice, err
		}
	}

	return ticker.GetTicker(z.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (z *ZB) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(z.GetName(), p, assetType)
	if err != nil {
		return z.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (z *ZB) GetOrderbookEx(currency currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(z.GetName(), currency, assetType)
	if err != nil {
		return z.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (z *ZB) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	currency := exchange.FormatExchangeCurrency(z.Name, p).String()

	orderbookNew, err := z.GetOrderbook(currency)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	orderBook.Pair = p
	orderBook.AssetType = assetType
	orderBook.ExchangeName = z.GetName()

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(z.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// ZB exchange
func (z *ZB) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	bal, err := z.GetAccountInformation()
	if err != nil {
		return info, err
	}

	var balances []exchange.AccountCurrencyInfo
	for _, data := range bal.Result.Coins {
		hold, err := strconv.ParseFloat(data.Freez, 64)
		if err != nil {
			return info, err
		}

		avail, err := strconv.ParseFloat(data.Available, 64)
		if err != nil {
			return info, err
		}

		balances = append(balances, exchange.AccountCurrencyInfo{
			CurrencyName: currency.NewCode(data.EnName),
			TotalValue:   hold + avail,
			Hold:         hold,
		})
	}

	info.Exchange = z.GetName()
	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balances,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (z *ZB) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (z *ZB) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (z *ZB) SubmitOrder(p currency.Pair, side exchange.OrderSide, _ exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var oT SpotNewOrderRequestParamsType

	if side == exchange.BuyOrderSide {
		oT = SpotNewOrderRequestParamsTypeBuy
	} else {
		oT = SpotNewOrderRequestParamsTypeSell
	}

	var params = SpotNewOrderRequestParams{
		Amount: amount,
		Price:  price,
		Symbol: common.StringToLower(p.String()),
		Type:   oT,
	}
	response, err := z.SpotNewOrder(params)

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
func (z *ZB) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (z *ZB) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	return z.CancelExistingOrder(orderIDInt, exchange.FormatExchangeCurrency(z.Name, order.CurrencyPair).String())
}

// CancelAllOrders cancels all orders associated with a currency pair
func (z *ZB) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	var allOpenOrders []Order
	for _, currency := range z.GetEnabledCurrencies() {
		var pageNumber int64
		// Limiting to 10 pages
		for i := 0; i < 10; i++ {
			openOrders, err := z.GetUnfinishedOrdersIgnoreTradeType(exchange.FormatExchangeCurrency(z.Name, currency).String(), 1, 10)
			if err != nil {
				return cancelAllOrdersResponse, err
			}

			if len(openOrders) == 0 {
				break
			}

			allOpenOrders = append(allOpenOrders, openOrders...)
			pageNumber++
		}
	}

	for _, openOrder := range allOpenOrders {
		err := z.CancelExistingOrder(openOrder.ID, openOrder.Currency)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(openOrder.ID, 10)] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (z *ZB) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (z *ZB) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	address, err := z.GetCryptoAddress(cryptocurrency)
	if err != nil {
		return "", err
	}

	return address.Message.Data.Key, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (z *ZB) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return z.Withdraw(withdrawRequest.Currency.Lower().String(), withdrawRequest.Address, withdrawRequest.TradePassword, withdrawRequest.Amount, withdrawRequest.FeeAmount, false)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (z *ZB) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (z *ZB) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (z *ZB) GetWebsocket() (*exchange.Websocket, error) {
	return z.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (z *ZB) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return z.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (z *ZB) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var allOrders []Order
	for _, currency := range getOrdersRequest.Currencies {
		var pageNumber int64
		// Limiting to 10 pages
		for i := 0; i < 10; i++ {
			resp, err := z.GetUnfinishedOrdersIgnoreTradeType(exchange.FormatExchangeCurrency(z.Name, currency).String(), pageNumber, 10)
			if err != nil {
				return nil, err
			}
			if len(resp) == 0 {
				break
			}

			allOrders = append(allOrders, resp...)
			pageNumber++
		}
	}

	var orders []exchange.OrderDetail
	for _, order := range allOrders {
		symbol := currency.NewPairDelimiter(order.Currency,
			z.ConfigCurrencyPairFormat.Delimiter)
		orderDate := time.Unix(int64(order.TradeDate), 0)
		orderSide := orderSideMap[order.Type]
		orders = append(orders, exchange.OrderDetail{
			ID:           fmt.Sprintf("%v", order.ID),
			Amount:       order.TotalAmount,
			Exchange:     z.Name,
			OrderDate:    orderDate,
			Price:        float64(order.Price),
			OrderSide:    orderSide,
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
// This function is not concurrency safe due to orderSide/orderType maps
func (z *ZB) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if getOrdersRequest.OrderSide == exchange.AnyOrderSide || getOrdersRequest.OrderSide == "" {
		return nil, errors.New("specific order side is required")
	}

	var allOrders []Order

	var side int64
	if getOrdersRequest.OrderSide == exchange.BuyOrderSide {
		side = 1
	}

	for _, currency := range getOrdersRequest.Currencies {
		var pageNumber int64
		// Limiting to 10 pages
		for i := 0; i < 10; i++ {
			resp, err := z.GetOrders(exchange.FormatExchangeCurrency(z.Name, currency).String(), pageNumber, side)
			if err != nil {
				return nil, err
			}

			if len(resp) == 0 {
				break
			}

			allOrders = append(allOrders, resp...)
			pageNumber++
		}
	}

	var orders []exchange.OrderDetail
	for _, order := range allOrders {
		symbol := currency.NewPairDelimiter(order.Currency,
			z.ConfigCurrencyPairFormat.Delimiter)
		orderDate := time.Unix(int64(order.TradeDate), 0)
		orderSide := orderSideMap[order.Type]
		orders = append(orders, exchange.OrderDetail{
			ID:           fmt.Sprintf("%v", order.ID),
			Amount:       order.TotalAmount,
			Exchange:     z.Name,
			OrderDate:    orderDate,
			Price:        float64(order.Price),
			OrderSide:    orderSide,
			CurrencyPair: symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)

	return orders, nil
}
