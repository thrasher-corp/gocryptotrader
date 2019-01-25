package alphapoint

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// GetAccountInfo retrieves balances for all enabled currencies on the
// Alphapoint exchange
func (a *Alphapoint) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = a.GetName()
	account, err := a.GetAccountInformation()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(account.Currencies); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = account.Currencies[i].Name
		exchangeCurrency.TotalValue = float64(account.Currencies[i].Balance)
		exchangeCurrency.Hold = float64(account.Currencies[i].Hold)

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (a *Alphapoint) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := a.GetTicker(p.Pair().String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Low = tick.Low
	tickerPrice.High = tick.High
	tickerPrice.Volume = tick.Volume
	tickerPrice.Last = tick.Last
	ticker.ProcessTicker(a.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(a.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (a *Alphapoint) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *Alphapoint) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := a.GetOrderbook(p.Pair().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Quantity, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Quantity, Price: data.Price})
	}

	orderbook.ProcessOrderbook(a.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(a.Name, p, assetType)
}

// GetOrderbookEx returns the orderbook for a currency pair
func (a *Alphapoint) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (a *Alphapoint) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	// https://alphapoint.github.io/slate/#generatetreasuryactivityreport
	return fundHistory, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (a *Alphapoint) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order and returns a true value when
// successfully submitted
func (a *Alphapoint) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	response, err := a.CreateOrder(p.Pair().String(), side.ToString(), orderType.ToString(), amount, price)
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
func (a *Alphapoint) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (a *Alphapoint) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = a.CancelExistingOrder(orderIDInt, order.AccountID)

	return err
}

// CancelAllOrders cancels all orders for a given account
func (a *Alphapoint) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	return exchange.CancelAllOrdersResponse{}, a.CancelAllExistingOrders(orderCancellation.AccountID)
}

// GetOrderInfo returns information on a current open order
func (a *Alphapoint) GetOrderInfo(orderID int64) (float64, error) {
	orders, err := a.GetOrders()
	if err != nil {
		return 0, err
	}

	for x := range orders {
		for y := range orders[x].OpenOrders {
			if int64(orders[x].OpenOrders[y].ServerOrderID) == orderID {
				return float64(orders[x].OpenOrders[y].QtyRemaining), nil
			}
		}
	}
	return 0, errors.New("order not found")
}

// GetDepositAddress returns a deposit address for a specified currency
func (a *Alphapoint) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	addreses, err := a.GetDepositAddresses()
	if err != nil {
		return "", err
	}

	for x := range addreses {
		if addreses[x].Name == cryptocurrency.String() {
			return addreses[x].DepositAddress, nil
		}
	}
	return "", errors.New("associated currency address not found")
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *Alphapoint) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (a *Alphapoint) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (a *Alphapoint) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (a *Alphapoint) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (a *Alphapoint) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrFunctionNotSupported
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (a *Alphapoint) GetWithdrawCapabilities() uint32 {
	return a.GetWithdrawPermissions()
}

// GetActiveOrders retrieves any orders that are active/open
func (a *Alphapoint) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := a.GetOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for x := range resp {
		for _, order := range resp[x].OpenOrders {
			if order.State != 1 {
				continue
			}

			orderDetail := exchange.OrderDetail{
				Amount:          float64(order.QtyTotal),
				OrderDate:       order.ReceiveTime,
				Exchange:        fmt.Sprintf("%v - %v", a.Name, order.AccountID),
				ID:              fmt.Sprintf("%v", order.ServerOrderID),
				Price:           float64(order.Price),
				RemainingAmount: float64(order.QtyRemaining),
			}

			if order.Side == 1 {
				orderDetail.OrderSide = string(exchange.BuyOrderSide)
			} else if order.Side == 2 {
				orderDetail.OrderSide = string(exchange.SellOrderSide)
			}

			switch order.OrderType {
			case 1:
				orderDetail.OrderType = string(exchange.MarketOrderType)
			case 2:
				orderDetail.OrderType = string(exchange.LimitOrderType)
			case 3:
				fallthrough
			case 4:
				orderDetail.OrderType = string(exchange.StopOrderType)
			case 5:
				fallthrough
			case 6:
				orderDetail.OrderType = string(exchange.TrailingStopOrderType)
			default:
				orderDetail.OrderType = string(exchange.UnknownOrderType)
			}

			orders = append(orders, orderDetail)
		}
	}

	a.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	a.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	a.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (a *Alphapoint) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := a.GetOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for x := range resp {
		for _, order := range resp[x].OpenOrders {
			if order.State == 1 {
				continue
			}

			orderDetail := exchange.OrderDetail{
				Amount:          float64(order.QtyTotal),
				OrderDate:       order.ReceiveTime,
				Exchange:        fmt.Sprintf("%v - %v", a.Name, order.AccountID),
				ID:              fmt.Sprintf("%v", order.ServerOrderID),
				Price:           float64(order.Price),
				RemainingAmount: float64(order.QtyRemaining),
			}
			if order.Side == 1 {
				orderDetail.OrderSide = string(exchange.BuyOrderSide)
			} else if order.Side == 2 {
				orderDetail.OrderSide = string(exchange.SellOrderSide)
			}

			switch order.OrderType {
			case 1:
				orderDetail.OrderType = string(exchange.MarketOrderType)
			case 2:
				orderDetail.OrderType = string(exchange.LimitOrderType)
			case 3:
				fallthrough
			case 4:
				orderDetail.OrderType = string(exchange.StopOrderType)
			case 5:
				fallthrough
			case 6:
				orderDetail.OrderType = string(exchange.TrailingStopOrderType)
			default:
				orderDetail.OrderType = string(exchange.UnknownOrderType)
			}

			orders = append(orders, orderDetail)
		}
	}

	a.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	a.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	a.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)

	return orders, nil
}
