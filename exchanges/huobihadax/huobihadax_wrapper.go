package huobihadax

import (
	"errors"
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

// Start starts the OKEX go routine
func (h *HUOBIHADAX) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		h.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (h *HUOBIHADAX) Run() {
	if h.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", h.GetName(), common.IsEnabled(h.Websocket.IsEnabled()), h.WebsocketURL)
		log.Debugf("%s polling delay: %ds.\n", h.GetName(), h.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", h.GetName(), len(h.EnabledPairs), h.EnabledPairs)
	}

	exchangeProducts, err := h.GetSymbols()
	if err != nil {
		log.Debugf("%s Failed to get available symbols.\n", h.GetName())
	} else {
		var currencies []string
		for x := range exchangeProducts {
			newCurrency := exchangeProducts[x].BaseCurrency + "-" + exchangeProducts[x].QuoteCurrency
			currencies = append(currencies, newCurrency)
		}

		err = h.UpdateCurrencies(currencies, false, false)
		if err != nil {
			log.Debugf("%s Failed to update available currencies.\n", h.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HUOBIHADAX) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := h.GetMarketDetailMerged(exchange.FormatExchangeCurrency(h.Name, p).String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Close
	tickerPrice.Volume = tick.Volume
	tickerPrice.High = tick.High

	if len(tick.Ask) > 0 {
		tickerPrice.Ask = tick.Ask[0]
	}

	if len(tick.Bid) > 0 {
		tickerPrice.Bid = tick.Bid[0]
	}

	ticker.ProcessTicker(h.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(h.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (h *HUOBIHADAX) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (h *HUOBIHADAX) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HUOBIHADAX) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := h.GetDepth(exchange.FormatExchangeCurrency(h.Name, p).String(), "step1")
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

	orderbook.ProcessOrderbook(h.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(h.Name, p, assetType)
}

// GetAccountID returns the account info
func (h *HUOBIHADAX) GetAccountID() ([]Account, error) {
	acc, err := h.GetAccounts()
	if err != nil {
		return nil, err
	}

	if len(acc) < 1 {
		return nil, errors.New("no account returned")
	}

	return acc, nil
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// HUOBIHADAX exchange
func (h *HUOBIHADAX) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	info.Exchange = h.GetName()

	accounts, err := h.GetAccountID()
	if err != nil {
		return info, err
	}

	for _, account := range accounts {
		var acc exchange.Account

		acc.ID = strconv.FormatInt(account.ID, 10)

		balances, err := h.GetAccountBalance(acc.ID)
		if err != nil {
			return info, err
		}

		var currencyDetails []exchange.AccountCurrencyInfo
		for _, balance := range balances {
			var frozen bool
			if balance.Type == "frozen" {
				frozen = true
			}

			var updated bool
			for i := range currencyDetails {
				if currencyDetails[i].CurrencyName == balance.Currency {
					if frozen {
						currencyDetails[i].Hold = balance.Balance
					} else {
						currencyDetails[i].TotalValue = balance.Balance
					}
					updated = true
				}
			}

			if updated {
				continue
			}

			if frozen {
				currencyDetails = append(currencyDetails,
					exchange.AccountCurrencyInfo{
						CurrencyName: balance.Currency,
						Hold:         balance.Balance,
					})
			} else {
				currencyDetails = append(currencyDetails,
					exchange.AccountCurrencyInfo{
						CurrencyName: balance.Currency,
						TotalValue:   balance.Balance,
					})
			}
		}

		acc.Currencies = currencyDetails
		info.Accounts = append(info.Accounts, acc)
	}

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (h *HUOBIHADAX) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (h *HUOBIHADAX) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (h *HUOBIHADAX) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	accountID, err := strconv.ParseInt(clientID, 0, 64)
	if err != nil {
		return submitOrderResponse, err
	}

	var formattedType SpotNewOrderRequestParamsType
	var params = SpotNewOrderRequestParams{
		Amount:    amount,
		Source:    "api",
		Symbol:    common.StringToLower(p.Pair().String()),
		AccountID: int(accountID),
	}

	if side == exchange.Buy && orderType == exchange.Market {
		formattedType = SpotNewOrderRequestTypeBuyMarket
	} else if side == exchange.Sell && orderType == exchange.Market {
		formattedType = SpotNewOrderRequestTypeSellMarket
	} else if side == exchange.Buy && orderType == exchange.Limit {
		formattedType = SpotNewOrderRequestTypeBuyLimit
		params.Price = price
	} else if side == exchange.Sell && orderType == exchange.Limit {
		formattedType = SpotNewOrderRequestTypeSellLimit
		params.Price = price
	} else {
		return submitOrderResponse, errors.New("Unsupported order type")
	}

	params.Type = formattedType

	response, err := h.SpotNewOrder(params)

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
func (h *HUOBIHADAX) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (h *HUOBIHADAX) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	_, err = h.CancelExistingOrder(orderIDInt)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (h *HUOBIHADAX) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	for _, currency := range h.GetEnabledCurrencies() {
		resp, err := h.CancelOpenOrdersBatch(orderCancellation.AccountID, exchange.FormatExchangeCurrency(h.Name, currency).String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		if resp.Data.FailedCount > 0 {
			return cancelAllOrdersResponse, fmt.Errorf("%v orders failed to cancel", resp.Data.FailedCount)
		}

		if resp.Status == "error" {
			return cancelAllOrdersResponse, errors.New(resp.ErrorMessage)
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (h *HUOBIHADAX) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (h *HUOBIHADAX) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (h *HUOBIHADAX) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	resp, err := h.Withdraw(withdrawRequest.Address, withdrawRequest.Currency.Lower().String(), withdrawRequest.AddressTag, withdrawRequest.Amount, withdrawRequest.FeeAmount)
	return fmt.Sprintf("%v", resp), err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBIHADAX) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBIHADAX) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (h *HUOBIHADAX) GetWebsocket() (*exchange.Websocket, error) {
	return h.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HUOBIHADAX) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return h.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (h *HUOBIHADAX) GetWithdrawCapabilities() uint32 {
	return h.GetWithdrawPermissions()
}

// GetActiveOrders retrieves any orders that are active/open
func (h *HUOBIHADAX) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (h *HUOBIHADAX) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}
