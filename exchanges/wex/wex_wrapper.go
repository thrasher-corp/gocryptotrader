package wex

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

// Start starts the WEX go routine
func (w *WEX) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		w.Run()
		wg.Done()
	}()
}

// Run implements the WEX wrapper
func (w *WEX) Run() {
	if w.Verbose {
		log.Debugf("%s Websocket: %s.", w.GetName(), common.IsEnabled(w.Websocket.IsEnabled()))
		log.Debugf("%s polling delay: %ds.\n", w.GetName(), w.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", w.GetName(), len(w.EnabledPairs), w.EnabledPairs)
	}

	exchangeProducts, err := w.GetTradablePairs()
	if err != nil {
		log.Errorf("%s Failed to get available symbols.\n", w.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(w.EnabledPairs, "_") || !common.StringDataContains(w.AvailablePairs, "_") {
			forceUpgrade = true
		}

		if forceUpgrade {
			enabledPairs := []string{"BTC_USD", "LTC_USD", "LTC_BTC", "ETH_USD"}
			log.Warn("Enabled pairs for WEX reset due to config upgrade, please enable the ones you would like again.")

			err = w.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Errorf("%s Failed to get config.\n", w.GetName())
			}
		}
		err = w.UpdateCurrencies(exchangeProducts, false, forceUpgrade)
		if err != nil {
			log.Errorf("%s Failed to get config.\n", w.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (w *WEX) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(w.Name, w.GetEnabledCurrencies())
	if err != nil {
		return tickerPrice, err
	}

	result, err := w.GetTicker(pairsCollated.String())
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range w.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(w.Name, x).Lower().String()
		var tp ticker.Price
		tp.Pair = x
		tp.Last = result[currency].Last
		tp.Ask = result[currency].Sell
		tp.Bid = result[currency].Buy
		tp.Last = result[currency].Last
		tp.Low = result[currency].Low
		tp.Volume = result[currency].VolumeCurrent
		ticker.ProcessTicker(w.Name, x, tp, assetType)
	}
	return ticker.GetTicker(w.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (w *WEX) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(w.GetName(), p, assetType)
	if err != nil {
		return w.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (w *WEX) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(w.GetName(), p, assetType)
	if err != nil {
		return w.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (w *WEX) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := w.GetDepth(exchange.FormatExchangeCurrency(w.Name, p).String())
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

	orderbook.ProcessOrderbook(w.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(w.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// WEX exchange
func (w *WEX) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = w.GetName()
	accountBalance, err := w.GetAccountInformation()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for x, y := range accountBalance.Funds {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = common.StringToUpper(x)
		exchangeCurrency.TotalValue = y
		exchangeCurrency.Hold = 0
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (w *WEX) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (w *WEX) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (w *WEX) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	response, err := w.Trade(common.StringToLower(p.Pair().String()), common.StringToLower(side.ToString()), amount, price)

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
func (w *WEX) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (w *WEX) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = w.CancelExistingOrder(orderIDInt)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (w *WEX) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	var allActiveOrders map[string]ActiveOrders

	for _, pair := range w.EnabledPairs {
		activeOrders, err := w.GetOpenOrders(pair)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		for k, v := range activeOrders {
			allActiveOrders[k] = v
		}
	}

	for k := range allActiveOrders {
		orderIDInt, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		_, err = w.CancelExistingOrder(orderIDInt)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[k] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (w *WEX) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (w *WEX) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (w *WEX) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	resp, err := w.WithdrawCoins(withdrawRequest.Currency.String(), withdrawRequest.Amount, withdrawRequest.Address)
	return fmt.Sprintf("%v", resp.TID), err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (w *WEX) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (w *WEX) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (w *WEX) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (w *WEX) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return w.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (w *WEX) GetWithdrawCapabilities() uint32 {
	return w.GetWithdrawPermissions()
}

// GetActiveOrders retrieves any orders that are active/open
func (w *WEX) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (w *WEX) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}
