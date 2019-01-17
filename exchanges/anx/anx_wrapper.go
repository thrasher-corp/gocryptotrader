package anx

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the ANX go routine
func (a *ANX) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		a.Run()
		wg.Done()
	}()
}

// Run implements the ANX wrapper
func (a *ANX) Run() {
	if a.Verbose {
		log.Debugf("%s polling delay: %ds.\n", a.GetName(), a.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", a.GetName(), len(a.EnabledPairs), a.EnabledPairs)
	}

	exchangeProducts, err := a.GetTradablePairs()
	if err != nil {
		log.Debugf("%s Failed to get available symbols.\n", a.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(a.EnabledPairs, "_") || !common.StringDataContains(a.AvailablePairs, "_") {
			forceUpgrade = true
		}

		if forceUpgrade {
			enabledPairs := []string{"BTC_USD,BTC_HKD,BTC_EUR,BTC_CAD,BTC_AUD,BTC_SGD,BTC_JPY,BTC_GBP,BTC_NZD,LTC_BTC,DOG_EBTC,STR_BTC,XRP_BTC"}
			log.Warn("Enabled pairs for ANX reset due to config upgrade, please enable the ones you would like again.")

			err = a.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Errorf("%s Failed to get config.\n", a.GetName())
			}
		}
		err = a.UpdateCurrencies(exchangeProducts, false, forceUpgrade)
		if err != nil {
			log.Errorf("%s Failed to get config.\n", a.GetName())
		}
	}
}

// GetTradablePairs returns a list of available
func (a *ANX) GetTradablePairs() ([]string, error) {
	result, err := a.GetCurrencies()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range result.CurrencyPairs {
		currencies = append(currencies, result.CurrencyPairs[x].TradedCcy+"_"+result.CurrencyPairs[x].SettlementCcy)
	}

	return currencies, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (a *ANX) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := a.GetTicker(exchange.FormatExchangeCurrency(a.GetName(), p).String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p

	if tick.Data.Sell.Value != "" {
		tickerPrice.Ask, err = strconv.ParseFloat(tick.Data.Sell.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Ask = 0
	}

	if tick.Data.Buy.Value != "" {
		tickerPrice.Bid, err = strconv.ParseFloat(tick.Data.Buy.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Bid = 0
	}

	if tick.Data.Low.Value != "" {
		tickerPrice.Low, err = strconv.ParseFloat(tick.Data.Low.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Low = 0
	}

	if tick.Data.Last.Value != "" {
		tickerPrice.Last, err = strconv.ParseFloat(tick.Data.Last.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Last = 0
	}

	if tick.Data.Vol.Value != "" {
		tickerPrice.Volume, err = strconv.ParseFloat(tick.Data.Vol.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Volume = 0
	}

	if tick.Data.High.Value != "" {
		tickerPrice.High, err = strconv.ParseFloat(tick.Data.High.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.High = 0
	}
	ticker.ProcessTicker(a.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(a.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (a *ANX) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (a *ANX) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *ANX) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := a.GetDepth(exchange.FormatExchangeCurrency(a.GetName(), p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Data.Asks {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{
				Price:  orderbookNew.Data.Asks[x].Price,
				Amount: orderbookNew.Data.Asks[x].Amount})
	}

	for x := range orderbookNew.Data.Bids {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{
				Price:  orderbookNew.Data.Bids[x].Price,
				Amount: orderbookNew.Data.Bids[x].Amount})
	}

	orderbook.ProcessOrderbook(a.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(a.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies on the
// exchange
func (a *ANX) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo

	raw, err := a.GetAccountInformation()
	if err != nil {
		return info, err
	}

	var balance []exchange.AccountCurrencyInfo
	for currency, info := range raw.Wallets {
		balance = append(balance, exchange.AccountCurrencyInfo{
			CurrencyName: currency,
			TotalValue:   info.AvailableBalance.Value,
			Hold:         info.Balance.Value,
		})
	}

	info.Exchange = a.GetName()
	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balance,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (a *ANX) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (a *ANX) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (a *ANX) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	var isBuying bool
	var limitPriceInSettlementCurrency float64

	if side == exchange.Buy {
		isBuying = true
	}

	if orderType == exchange.Limit {
		limitPriceInSettlementCurrency = price
	}

	response, err := a.NewOrder(orderType.ToString(),
		isBuying,
		p.FirstCurrency.String(),
		amount,
		p.SecondCurrency.String(),
		amount,
		limitPriceInSettlementCurrency,
		false,
		"",
		false)

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
func (a *ANX) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (a *ANX) CancelOrder(order exchange.OrderCancellation) error {
	orderIDs := []string{order.OrderID}
	_, err := a.CancelOrderByIDs(orderIDs)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (a *ANX) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	placedOrders, err := a.GetOrderList(true)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var orderIDs []string
	for _, order := range placedOrders {
		orderIDs = append(orderIDs, order.OrderID)
	}

	resp, err := a.CancelOrderByIDs(orderIDs)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, order := range resp.OrderCancellationResponses {
		if order.Error != CancelRequestSubmitted {
			cancelAllOrdersResponse.OrderStatus[order.UUID] = order.Error
		}
	}

	return cancelAllOrdersResponse, err
}

// GetOrderInfo returns information on a current open order
func (a *ANX) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (a *ANX) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return a.GetDepositAddressByCurrency(cryptocurrency.String(), "", false)
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return a.Send(withdrawRequest.Currency.String(), withdrawRequest.Address, "", fmt.Sprintf("%v", withdrawRequest.Amount))
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	// Fiat withdrawals available via website
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	// Fiat withdrawals available via website
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (a *ANX) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (a *ANX) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return a.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (a *ANX) GetWithdrawCapabilities() uint32 {
	return a.GetWithdrawPermissions()
}

// GetActiveOrders retrieves any orders that are active/open
func (a *ANX) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (a *ANX) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var isActiveOrdersOnly bool
	if getOrdersRequest.OrderStatus == exchange.ActiveOrderStatus {
		isActiveOrdersOnly = true
	}

	resp, err := a.GetOrderList(isActiveOrdersOnly)
	if err != nil {
		return nil, err
	}
 
	var orders []exchange.OrderDetail
	for _, order := range resp {
		orderDetail := exchange.OrderDetail{
			Amount:              order.TradedCurrencyAmount,
			BaseCurrency:        order.TradedCurrency,
			OrderPlacementTicks: order.Timestamp,
			Exchange:            a.Name,
			ID:                  order.OrderID,
			OrderType:           order.OrderType,
			Price:               order.SettlementCurrencyAmount,
			QuoteCurrency:       order.SettlementCurrency,
			Status:              order.OrderStatus,
		}

		orders = append(orders, orderDetail)
	}

	a.FilterOrdersByStatusAndType(&orders, getOrdersRequest.OrderType, getOrdersRequest.OrderStatus)
	a.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	a.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}
