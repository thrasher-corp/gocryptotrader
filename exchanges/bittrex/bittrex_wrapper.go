package bittrex

import (
	"errors"
	"fmt"
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

// Start starts the Bittrex go routine
func (b *Bittrex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bittrex wrapper
func (b *Bittrex) Run() {
	if b.Verbose {
		log.Debugf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	exchangeProducts, err := b.GetMarkets()
	if err != nil {
		log.Errorf("%s Failed to get available symbols.\n", b.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(b.EnabledPairs.String(), "-") ||
			!common.StringDataContains(b.AvailablePairs.String(), "-") {
			forceUpgrade = true
		}
		var currencies []string
		for x := range exchangeProducts.Result {
			if !exchangeProducts.Result[x].IsActive ||
				exchangeProducts.Result[x].MarketName == "" {
				continue
			}
			currencies = append(currencies, exchangeProducts.Result[x].MarketName)
		}

		if forceUpgrade {
			enabledPairs := currency.Pairs{currency.Pair{Base: currency.USDT,
				Quote: currency.BTC, Delimiter: "-"}}

			log.Warn("Available pairs for Bittrex reset due to config upgrade, please enable the ones you would like again")

			err = b.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Errorf("%s Failed to get config.", b.GetName())
			}
		}

		var newCurrencies currency.Pairs
		for _, p := range currencies {
			newCurrencies = append(newCurrencies,
				currency.NewCurrencyPairFromString(p))
		}

		err = b.UpdateCurrencies(newCurrencies, false, forceUpgrade)
		if err != nil {
			log.Errorf("%s Failed to get config.", b.GetName())
		}
	}
}

// GetAccountInfo Retrieves balances for all enabled currencies for the
// Bittrex exchange
func (b *Bittrex) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = b.GetName()
	accountBalance, err := b.GetAccountBalances()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(accountBalance.Result); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance.Result[i].Currency
		exchangeCurrency.TotalValue = accountBalance.Result[i].Balance
		exchangeCurrency.Hold = accountBalance.Result[i].Balance - accountBalance.Result[i].Available
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bittrex) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetMarketSummaries()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range b.GetEnabledCurrencies() {
		curr := exchange.FormatExchangeCurrency(b.Name, x)
		for y := range tick.Result {
			if tick.Result[y].MarketName != curr.String() {
				continue
			}
			tickerPrice.Pair = x
			tickerPrice.High = tick.Result[y].High
			tickerPrice.Low = tick.Result[y].Low
			tickerPrice.Ask = tick.Result[y].Ask
			tickerPrice.Bid = tick.Result[y].Bid
			tickerPrice.Last = tick.Result[y].Last
			tickerPrice.Volume = tick.Result[y].Volume
			ticker.ProcessTicker(b.GetName(), tickerPrice, assetType)
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Bittrex) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, ticker.Spot)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *Bittrex) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bittrex) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(exchange.FormatExchangeCurrency(b.GetName(), p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Result.Buy {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{
				Amount: orderbookNew.Result.Buy[x].Quantity,
				Price:  orderbookNew.Result.Buy[x].Rate,
			},
		)
	}

	for x := range orderbookNew.Result.Sell {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{
				Amount: orderbookNew.Result.Sell[x].Quantity,
				Price:  orderbookNew.Result.Sell[x].Rate,
			},
		)
	}

	err = orderbook.ProcessOrderbook(b.GetName(), orderBook, assetType)
	if err != nil {
		return orderBook, err
	}

	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bittrex) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bittrex) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bittrex) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	buy := side == exchange.BuyOrderSide
	var response UUID
	var err error

	if orderType != exchange.LimitOrderType {
		return submitOrderResponse, errors.New("not supported on exchange")
	}

	if buy {
		response, err = b.PlaceBuyLimit(p.String(), amount, price)
	} else {
		response, err = b.PlaceSellLimit(p.String(), amount, price)
	}

	if response.Result.ID != "" {
		submitOrderResponse.OrderID = response.Result.ID
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bittrex) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bittrex) CancelOrder(order exchange.OrderCancellation) error {
	_, err := b.CancelExistingOrder(order.OrderID)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bittrex) CancelAllOrders(_ exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := b.GetOpenOrders("")
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, order := range openOrders.Result {
		_, err := b.CancelExistingOrder(order.OrderUUID)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[order.OrderUUID] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Bittrex) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bittrex) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	depositAddr, err := b.GetCryptoDepositAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	return depositAddr.Result.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bittrex) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	uuid, err := b.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.AddressTag, withdrawRequest.Address, withdrawRequest.Amount)
	return fmt.Sprintf("%v", uuid), err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bittrex) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bittrex) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bittrex) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bittrex) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)

}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bittrex) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var currPair string
	if len(getOrdersRequest.Currencies) == 1 {
		currPair = getOrdersRequest.Currencies[0].String()
	}

	resp, err := b.GetOpenOrders(currPair)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, order := range resp.Result {
		orderDate, err := time.Parse(time.RFC3339, order.Opened)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				b.Name, "GetActiveOrders", order.OrderUUID, order.Opened)
		}

		pair := currency.NewCurrencyPairDelimiter(order.Exchange,
			b.ConfigCurrencyPairFormat.Delimiter)
		orderType := exchange.OrderType(strings.ToUpper(order.Type))

		orders = append(orders, exchange.OrderDetail{
			Amount:          order.Quantity,
			RemainingAmount: order.QuantityRemaining,
			Price:           order.Price,
			OrderDate:       orderDate,
			ID:              order.OrderUUID,
			Exchange:        b.Name,
			OrderType:       orderType,
			CurrencyPair:    pair,
		})
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bittrex) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var currPair string
	if len(getOrdersRequest.Currencies) == 1 {
		currPair = getOrdersRequest.Currencies[0].String()
	}

	resp, err := b.GetOrderHistoryForCurrency(currPair)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, order := range resp.Result {
		orderDate, err := time.Parse(time.RFC3339, order.TimeStamp)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				b.Name, "GetActiveOrders", order.OrderUUID, order.Opened)
		}

		pair := currency.NewCurrencyPairDelimiter(order.Exchange,
			b.ConfigCurrencyPairFormat.Delimiter)
		orderType := exchange.OrderType(strings.ToUpper(order.Type))

		orders = append(orders, exchange.OrderDetail{
			Amount:          order.Quantity,
			RemainingAmount: order.QuantityRemaining,
			Price:           order.Price,
			OrderDate:       orderDate,
			ID:              order.OrderUUID,
			Exchange:        b.Name,
			OrderType:       orderType,
			Fee:             order.Commission,
			CurrencyPair:    pair,
		})
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}
