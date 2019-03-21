package yobit

import (
	"errors"
	"fmt"
	"math"
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

// Start starts the Yobit go routine
func (y *Yobit) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		y.Run()
		wg.Done()
	}()
}

// Run implements the Yobit wrapper
func (y *Yobit) Run() {
	if y.Verbose {
		log.Debugf("%s Websocket: %s.", y.GetName(), common.IsEnabled(y.Websocket.IsEnabled()))
		log.Debugf("%s polling delay: %ds.\n", y.GetName(), y.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", y.GetName(), len(y.EnabledPairs), y.EnabledPairs)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (y *Yobit) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(y.Name, y.GetEnabledCurrencies())
	if err != nil {
		return tickerPrice, err
	}

	result, err := y.GetTicker(pairsCollated)
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range y.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(y.Name, x).Lower().String()
		var tickerPrice ticker.Price
		tickerPrice.Pair = x
		tickerPrice.Last = result[currency].Last
		tickerPrice.Ask = result[currency].Sell
		tickerPrice.Bid = result[currency].Buy
		tickerPrice.Last = result[currency].Last
		tickerPrice.Low = result[currency].Low
		tickerPrice.Volume = result[currency].VolumeCurrent

		err = ticker.ProcessTicker(y.Name, tickerPrice, assetType)
		if err != nil {
			return tickerPrice, err
		}
	}
	return ticker.GetTicker(y.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (y *Yobit) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(y.GetName(), p, assetType)
	if err != nil {
		return y.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (y *Yobit) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(y.GetName(), p, assetType)
	if err != nil {
		return y.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (y *Yobit) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := y.GetDepth(exchange.FormatExchangeCurrency(y.Name, p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: data[0], Amount: data[1]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: data[0], Amount: data[1]})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = y.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(y.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Yobit exchange
func (y *Yobit) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = y.GetName()
	accountBalance, err := y.GetAccountInformation()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for x, y := range accountBalance.FundsInclOrders {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(x)
		exchangeCurrency.TotalValue = y
		exchangeCurrency.Hold = 0
		for z, w := range accountBalance.Funds {
			if z == x {
				exchangeCurrency.Hold = y - w
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
func (y *Yobit) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (y *Yobit) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
// Yobit only supports limit orders
func (y *Yobit) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	if orderType != exchange.LimitOrderType {
		return submitOrderResponse, errors.New("only limit orders are allowed")
	}

	response, err := y.Trade(p.String(), side.ToString(), amount, price)
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
func (y *Yobit) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (y *Yobit) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = y.CancelExistingOrder(orderIDInt)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (y *Yobit) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	var allActiveOrders []map[string]ActiveOrders

	for _, pair := range y.EnabledPairs {
		activeOrdersForPair, err := y.GetOpenOrders(pair.String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		allActiveOrders = append(allActiveOrders, activeOrdersForPair)
	}

	for _, activeOrders := range allActiveOrders {
		for key := range activeOrders {
			orderIDInt, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				return cancelAllOrdersResponse, err
			}

			_, err = y.CancelExistingOrder(orderIDInt)
			if err != nil {
				cancelAllOrdersResponse.OrderStatus[key] = err.Error()
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (y *Yobit) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (y *Yobit) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	a, err := y.GetCryptoDepositAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	return a.Return.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (y *Yobit) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	resp, err := y.WithdrawCoinsToAddress(withdrawRequest.Currency.String(), withdrawRequest.Amount, withdrawRequest.Address)
	if err != nil {
		return "", err
	}
	if len(resp.Error) > 0 {
		return "", errors.New(resp.Error)
	}
	return "success", nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (y *Yobit) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (y *Yobit) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (y *Yobit) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (y *Yobit) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return y.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (y *Yobit) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	for _, c := range getOrdersRequest.Currencies {
		resp, err := y.GetOpenOrders(exchange.FormatExchangeCurrency(y.Name,
			c).String())
		if err != nil {
			return nil, err
		}

		for ID, order := range resp {
			symbol := currency.NewPairDelimiter(order.Pair,
				y.ConfigCurrencyPairFormat.Delimiter)
			orderDate := time.Unix(int64(order.TimestampCreated), 0)
			side := exchange.OrderSide(strings.ToUpper(order.Type))
			orders = append(orders, exchange.OrderDetail{
				ID:           ID,
				Amount:       order.Amount,
				Price:        order.Rate,
				OrderSide:    side,
				OrderDate:    orderDate,
				CurrencyPair: symbol,
				Exchange:     y.Name,
			})
		}
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (y *Yobit) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var allOrders []TradeHistory
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := y.GetTradeHistory(0,
			10000,
			math.MaxInt64,
			getOrdersRequest.StartTicks.Unix(),
			getOrdersRequest.EndTicks.Unix(),
			"DESC",
			exchange.FormatExchangeCurrency(y.Name, currency).String())
		if err != nil {
			return nil, err
		}

		for _, order := range resp {
			allOrders = append(allOrders, order)
		}
	}

	var orders []exchange.OrderDetail
	for _, order := range allOrders {
		symbol := currency.NewPairDelimiter(order.Pair,
			y.ConfigCurrencyPairFormat.Delimiter)
		orderDate := time.Unix(int64(order.Timestamp), 0)
		side := exchange.OrderSide(strings.ToUpper(order.Type))
		orders = append(orders, exchange.OrderDetail{
			ID:           fmt.Sprintf("%v", order.OrderID),
			Amount:       order.Amount,
			Price:        order.Rate,
			OrderSide:    side,
			OrderDate:    orderDate,
			CurrencyPair: symbol,
			Exchange:     y.Name,
		})
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}
