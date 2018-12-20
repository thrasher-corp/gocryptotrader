package coinut

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/thrasher-/gocryptotrader/currency/symbol"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the COINUT go routine
func (c *COINUT) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
}

// Run implements the COINUT wrapper
func (c *COINUT) Run() {
	if c.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", c.GetName(), common.IsEnabled(c.Websocket.IsEnabled()), coinutWebsocketURL)
		log.Printf("%s polling delay: %ds.\n", c.GetName(), c.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", c.GetName(), len(c.EnabledPairs), c.EnabledPairs)
	}

	exchangeProducts, err := c.GetInstruments()
	if err != nil {
		log.Printf("%s Failed to get available products.\n", c.GetName())
		return
	}

	currencies := []string{}
	c.InstrumentMap = make(map[string]int)
	for x, y := range exchangeProducts.Instruments {
		c.InstrumentMap[x] = y[0].InstID
		currencies = append(currencies, x)
	}

	err = c.UpdateCurrencies(currencies, false, false)
	if err != nil {
		log.Printf("%s Failed to update available currencies.\n", c.GetName())
	}
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// COINUT exchange
func (c *COINUT) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	bal, err := c.GetUserBalance()
	if err != nil {
		return info, err
	}

	var balances []exchange.AccountCurrencyInfo
	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.BCH,
		TotalValue:   bal.BCH,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.BTC,
		TotalValue:   bal.BTC,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.BTG,
		TotalValue:   bal.BTG,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.CAD,
		TotalValue:   bal.CAD,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.ETC,
		TotalValue:   bal.ETC,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.ETH,
		TotalValue:   bal.ETH,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.LCH,
		TotalValue:   bal.LCH,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.LTC,
		TotalValue:   bal.LTC,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.MYR,
		TotalValue:   bal.MYR,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.SGD,
		TotalValue:   bal.SGD,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.USD,
		TotalValue:   bal.USD,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.USDT,
		TotalValue:   bal.USDT,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.XMR,
		TotalValue:   bal.XMR,
	})

	balances = append(balances, exchange.AccountCurrencyInfo{
		CurrencyName: symbol.ZEC,
		TotalValue:   bal.ZEC,
	})

	info.ExchangeName = c.GetName()
	info.Currencies = balances
	return info, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *COINUT) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := c.GetInstrumentTicker(c.InstrumentMap[p.Pair().String()])
	if err != nil {
		return ticker.Price{}, err
	}

	tickerPrice.Pair = p
	tickerPrice.Volume = tick.Volume
	tickerPrice.Last = tick.Last
	tickerPrice.High = tick.HighestBuy
	tickerPrice.Low = tick.LowestSell
	ticker.ProcessTicker(c.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(c.Name, p, assetType)

}

// GetTickerPrice returns the ticker for a currency pair
func (c *COINUT) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (c *COINUT) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *COINUT) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := c.GetInstrumentOrderbook(c.InstrumentMap[p.Pair().String()], 200)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Buy {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Buy[x].Quantity, Price: orderbookNew.Buy[x].Price})
	}

	for x := range orderbookNew.Sell {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Sell[x].Quantity, Price: orderbookNew.Sell[x].Price})
	}

	orderbook.ProcessOrderbook(c.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(c.Name, p, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (c *COINUT) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory

	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (c *COINUT) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (c *COINUT) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var err error
	var APIresponse interface{}
	isBuyOrder := side == exchange.Buy
	clientIDInt, err := strconv.ParseUint(clientID, 0, 32)
	clientIDUint := uint32(clientIDInt)

	if err != nil {
		return submitOrderResponse, err
	}
	// Need to get the ID of the currency sent
	instruments, err := c.GetInstruments()
	if err != nil {
		return submitOrderResponse, err
	}

	currencyArray := instruments.Instruments[p.Pair().String()]
	currencyID := currencyArray[0].InstID

	if orderType == exchange.Limit {
		APIresponse, err = c.NewOrder(currencyID, amount, price, isBuyOrder, clientIDUint)
	} else if orderType == exchange.Market {
		APIresponse, err = c.NewOrder(currencyID, amount, 0, isBuyOrder, clientIDUint)
	} else {
		return submitOrderResponse, errors.New("unsupported order type")
	}

	switch APIresponse.(type) {
	case OrdersBase:
		orderResult := APIresponse.(OrdersBase)
		submitOrderResponse.OrderID = fmt.Sprintf("%v", orderResult.OrderID)
	case OrderFilledResponse:
		orderResult := APIresponse.(OrderFilledResponse)
		submitOrderResponse.OrderID = fmt.Sprintf("%v", orderResult.Order.OrderID)
	case OrderRejectResponse:
		orderResult := APIresponse.(OrderRejectResponse)
		submitOrderResponse.OrderID = fmt.Sprintf("%v", orderResult.OrderID)
		err = fmt.Errorf("OrderID: %v was rejected: %v", orderResult.OrderID, orderResult.Reasons)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *COINUT) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *COINUT) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	// Need to get the ID of the currency sent
	instruments, err := c.GetInstruments()

	if err != nil {
		return err
	}

	currencyArray := instruments.Instruments[exchange.FormatExchangeCurrency(c.Name, order.CurrencyPair).String()]
	currencyID := currencyArray[0].InstID
	_, err = c.CancelExistingOrder(currencyID, int(orderIDInt))

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *COINUT) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	// TODO, this is a terrible implementation. Requires DB to improve
	// Coinut provides no way of retrieving orders without a currency
	// So we need to retrieve all currencies, then retrieve orders for each currency
	// Then cancel. Advisable to never use this until DB due to performance
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	instruments, err := c.GetInstruments()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var allTheOrders []OrderResponse
	for _, allInstrumentData := range instruments.Instruments {
		for _, instrumentData := range allInstrumentData {

			openOrders, err := c.GetOpenOrders(instrumentData.InstID)
			if err != nil {
				return cancelAllOrdersResponse, err
			}

			for _, openOrder := range openOrders.Orders {
				allTheOrders = append(allTheOrders, openOrder)
			}
		}
	}

	var allTheOrdersToCancel []CancelOrders
	for _, orderToCancel := range allTheOrders {
		cancelOrder := CancelOrders{
			InstrumentID: orderToCancel.InstrumentID,
			OrderID:      orderToCancel.OrderID,
		}
		allTheOrdersToCancel = append(allTheOrdersToCancel, cancelOrder)
	}

	if len(allTheOrdersToCancel) > 0 {
		resp, err := c.CancelOrders(allTheOrdersToCancel)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		for _, order := range resp.Results {
			if order.Status != "OK" {
				cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(order.OrderID, 10)] = order.Status
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (c *COINUT) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *COINUT) GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *COINUT) WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (c *COINUT) WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *COINUT) WithdrawFiatFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (c *COINUT) GetWebsocket() (*exchange.Websocket, error) {
	return c.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *COINUT) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return c.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (c *COINUT) GetWithdrawCapabilities() uint32 {
	return c.GetWithdrawPermissions()
}
