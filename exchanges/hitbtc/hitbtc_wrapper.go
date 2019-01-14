package hitbtc

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

// Start starts the HitBTC go routine
func (h *HitBTC) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		h.Run()
		wg.Done()
	}()
}

// Run implements the HitBTC wrapper
func (h *HitBTC) Run() {
	if h.Verbose {
		log.Debugf("%s Websocket: %s (url: %s).\n", h.GetName(), common.IsEnabled(h.Websocket.IsEnabled()), hitbtcWebsocketAddress)
		log.Debugf("%s polling delay: %ds.\n", h.GetName(), h.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", h.GetName(), len(h.EnabledPairs), h.EnabledPairs)
	}

	exchangeProducts, err := h.GetSymbolsDetailed()
	if err != nil {
		log.Errorf("%s Failed to get available symbols.\n", h.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(h.EnabledPairs, "-") || !common.StringDataContains(h.AvailablePairs, "-") {
			forceUpgrade = true
		}
		var currencies []string
		for x := range exchangeProducts {
			currencies = append(currencies, exchangeProducts[x].BaseCurrency+"-"+exchangeProducts[x].QuoteCurrency)
		}

		if forceUpgrade {
			enabledPairs := []string{"BTC-USD"}
			log.Warn("Available pairs for HitBTC reset due to config upgrade, please enable the ones you would like again.")

			err = h.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Errorf("%s Failed to update enabled currencies.\n", h.GetName())
			}
		}
		err = h.UpdateCurrencies(currencies, false, forceUpgrade)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", h.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HitBTC) UpdateTicker(currencyPair pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := h.GetTicker("")
	if err != nil {
		return ticker.Price{}, err
	}

	for _, x := range h.GetEnabledCurrencies() {
		var tp ticker.Price
		curr := exchange.FormatExchangeCurrency(h.GetName(), x).String()
		tp.Pair = x
		tp.Ask = tick[curr].Ask
		tp.Bid = tick[curr].Bid
		tp.High = tick[curr].High
		tp.Last = tick[curr].Last
		tp.Low = tick[curr].Low
		tp.Volume = tick[curr].Volume
		ticker.ProcessTicker(h.GetName(), x, tp, assetType)
	}
	return ticker.GetTicker(h.Name, currencyPair, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (h *HitBTC) GetTickerPrice(currencyPair pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.GetName(), currencyPair, assetType)
	if err != nil {
		return h.UpdateTicker(currencyPair, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (h *HitBTC) GetOrderbookEx(currencyPair pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(h.GetName(), currencyPair, assetType)
	if err != nil {
		return h.UpdateOrderbook(currencyPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HitBTC) UpdateOrderbook(currencyPair pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := h.GetOrderbook(exchange.FormatExchangeCurrency(h.GetName(), currencyPair).String(), 1000)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	orderbook.ProcessOrderbook(h.GetName(), currencyPair, orderBook, assetType)
	return orderbook.GetOrderbook(h.Name, currencyPair, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// HitBTC exchange
func (h *HitBTC) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = h.GetName()
	accountBalance, err := h.GetBalances()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for _, item := range accountBalance {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = item.Currency
		exchangeCurrency.TotalValue = item.Available
		exchangeCurrency.Hold = item.Reserved
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (h *HitBTC) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (h *HitBTC) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (h *HitBTC) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	response, err := h.PlaceOrder(p.Pair().String(), price, amount, common.StringToLower(orderType.ToString()), common.StringToLower(side.ToString()))

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
func (h *HitBTC) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (h *HitBTC) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	_, err = h.CancelExistingOrder(orderIDInt)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (h *HitBTC) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	resp, err := h.CancelAllExistingOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, order := range resp {
		cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(order.ID, 10)] = fmt.Sprintf("Could not cancel order %v. Status: %v", order.ID, order.Status)
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (h *HitBTC) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (h *HitBTC) GetDepositAddress(currency pair.CurrencyItem, accountID string) (string, error) {
	resp, err := h.GetDepositAddresses(currency.String())
	if err != nil {
		return "", err
	}

	return resp.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (h *HitBTC) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	_, err := h.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.Address, withdrawRequest.Amount)

	return "", err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (h *HitBTC) GetWebsocket() (*exchange.Websocket, error) {
	return h.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HitBTC) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return h.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (h *HitBTC) GetWithdrawCapabilities() uint32 {
	return h.GetWithdrawPermissions()
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (h *HitBTC) GetOrderHistory(open, closed, cancelled bool, startDate, endDate string) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}
