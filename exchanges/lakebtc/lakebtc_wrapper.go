package lakebtc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the LakeBTC go routine
func (l *LakeBTC) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
}

// Run implements the LakeBTC wrapper
func (l *LakeBTC) Run() {
	if l.Verbose {
		log.Debugf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	exchangeProducts, err := l.GetTradablePairs()
	if err != nil {
		log.Errorf("%s Failed to get available products.\n", l.GetName())
	} else {
		var newExchangeProducts currency.Pairs
		for _, p := range exchangeProducts {
			newExchangeProducts = append(newExchangeProducts,
				currency.NewPairFromString(p))
		}

		err = l.UpdateCurrencies(newExchangeProducts, false, false)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", l.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LakeBTC) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	tick, err := l.GetTicker()
	if err != nil {
		return ticker.Price{}, err
	}

	for _, x := range l.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(l.Name, x).String()
		var tickerPrice ticker.Price
		tickerPrice.Pair = x
		tickerPrice.Ask = tick[currency].Ask
		tickerPrice.Bid = tick[currency].Bid
		tickerPrice.Volume = tick[currency].Volume
		tickerPrice.High = tick[currency].High
		tickerPrice.Low = tick[currency].Low
		tickerPrice.Last = tick[currency].Last

		err = ticker.ProcessTicker(l.GetName(), tickerPrice, assetType)
		if err != nil {
			return tickerPrice, err
		}
	}
	return ticker.GetTicker(l.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (l *LakeBTC) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (l *LakeBTC) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *LakeBTC) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := l.GetOrderBook(p.String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	err = orderbook.ProcessOrderbook(l.GetName(), orderBook, assetType)
	if err != nil {
		return orderBook, err
	}

	return orderbook.GetOrderbook(l.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// LakeBTC exchange
func (l *LakeBTC) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = l.GetName()
	accountInfo, err := l.GetAccountInformation()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for x, y := range accountInfo.Balance {
		for z, w := range accountInfo.Locked {
			if z != x {
				continue
			}
			var exchangeCurrency exchange.AccountCurrencyInfo
			exchangeCurrency.CurrencyName = currency.NewCurrencyCode(x)
			exchangeCurrency.TotalValue, _ = strconv.ParseFloat(y, 64)
			exchangeCurrency.Hold, _ = strconv.ParseFloat(w, 64)
			currencies = append(currencies, exchangeCurrency)
		}
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *LakeBTC) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *LakeBTC) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (l *LakeBTC) SubmitOrder(p currency.Pair, side exchange.OrderSide, _ exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	isBuyOrder := side == exchange.BuyOrderSide
	response, err := l.Trade(isBuyOrder, amount, price, p.Lower().String())

	if response.ID > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response.ID)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *LakeBTC) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *LakeBTC) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	return l.CancelExistingOrder(orderIDInt)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *LakeBTC) CancelAllOrders(_ exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := l.GetOpenOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var ordersToCancel []string
	for _, order := range openOrders {
		orderIDString := strconv.FormatInt(order.ID, 10)
		ordersToCancel = append(ordersToCancel, orderIDString)
	}

	return cancelAllOrdersResponse, l.CancelExistingOrders(ordersToCancel)

}

// GetOrderInfo returns information on a current open order
func (l *LakeBTC) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *LakeBTC) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	if !strings.EqualFold(cryptocurrency.String(), currency.BTC.String()) {
		return "", fmt.Errorf("unsupported currency %s deposit address can only be BTC, manual deposit is required for other currencies",
			cryptocurrency.String())
	}

	info, err := l.GetAccountInformation()
	if err != nil {
		return "", err
	}

	return info.Profile.BTCDepositAddress, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *LakeBTC) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	if withdrawRequest.Currency != currency.BTC {
		return "", errors.New("only BTC supported for withdrawals")
	}

	resp, err := l.CreateWithdraw(withdrawRequest.Amount, withdrawRequest.Description)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", resp.ID), nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (l *LakeBTC) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (l *LakeBTC) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (l *LakeBTC) GetWebsocket() (*exchange.Websocket, error) {
	// Documents are too vague to implement
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (l *LakeBTC) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return l.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (l *LakeBTC) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := l.GetOpenOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, order := range resp {
		symbol := currency.NewPairDelimiter(order.Symbol, l.ConfigCurrencyPairFormat.Delimiter)
		orderDate := time.Unix(order.At, 0)
		side := exchange.OrderSide(strings.ToUpper(order.Type))

		orders = append(orders, exchange.OrderDetail{
			Amount:       order.Amount,
			ID:           fmt.Sprintf("%v", order.ID),
			Price:        order.Price,
			OrderSide:    side,
			OrderDate:    orderDate,
			CurrencyPair: symbol,
			Exchange:     l.Name,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (l *LakeBTC) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := l.GetOrders([]int64{})
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, order := range resp {
		if order.State == "active" {
			continue
		}

		symbol := currency.NewPairDelimiter(order.Symbol,
			l.ConfigCurrencyPairFormat.Delimiter)
		orderDate := time.Unix(order.At, 0)
		side := exchange.OrderSide(strings.ToUpper(order.Type))

		orders = append(orders, exchange.OrderDetail{
			Amount:       order.Amount,
			ID:           fmt.Sprintf("%v", order.ID),
			Price:        order.Price,
			OrderSide:    side,
			OrderDate:    orderDate,
			CurrencyPair: symbol,
			Exchange:     l.Name,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}
