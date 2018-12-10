package huobihadax

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
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
		log.Printf("%s Websocket: %s. (url: %s).\n", h.GetName(), common.IsEnabled(h.Websocket.IsEnabled()), h.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", h.GetName(), h.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", h.GetName(), len(h.EnabledPairs), h.EnabledPairs)
	}

	exchangeProducts, err := h.GetSymbols()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", h.GetName())
	} else {
		var currencies []string
		for x := range exchangeProducts {
			newCurrency := exchangeProducts[x].BaseCurrency + "-" + exchangeProducts[x].QuoteCurrency
			currencies = append(currencies, newCurrency)
		}

		err = h.UpdateCurrencies(currencies, false, false)
		if err != nil {
			log.Printf("%s Failed to update available currencies.\n", h.GetName())
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

var mtx sync.Mutex

// GetAccountID returns the account ID for trades NOTE interim implementation
// does not account for multiple account IDs
func (h *HUOBIHADAX) GetAccountID() (string, error) {
	mtx.Lock()
	defer mtx.Unlock()

	if h.AccountID == "" {
		acc, err := h.GetAccounts()
		if err != nil {
			return "", err
		}

		if len(acc) > 0 {
			return strconv.FormatInt(acc[0].ID, 10), nil
		}

		return "", errors.New("no account ID fetched")
	}

	return h.AccountID, nil
}

//GetAccountInfo retrieves balances for all enabled currencies for the
// HUOBIHADAX exchange - to-do
func (h *HUOBIHADAX) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	info.ExchangeName = h.GetName()

	accID, err := h.GetAccountID()
	if err != nil {
		return info, err
	}

	acc, err := h.GetAccountBalance(accID)
	if err != nil {
		return info, err
	}

	type hold struct {
		Avail float64
		Hold  float64
	}

	var currencyData = make(map[string]*hold)
	for _, data := range acc {
		_, ok := currencyData[data.Currency]
		if !ok {
			currencyData[data.Currency] = &hold{}
		}

		if data.Type == "trade" {
			currencyData[data.Currency].Avail = data.Balance
		} else {
			currencyData[data.Currency].Hold = data.Balance
		}
	}

	var balances []exchange.AccountCurrencyInfo

	for key, data := range currencyData {
		balances = append(balances, exchange.AccountCurrencyInfo{
			CurrencyName: key,
			TotalValue:   data.Avail + data.Hold,
			Hold:         data.Hold,
		})
	}

	info.Currencies = balances
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
func (h *HUOBIHADAX) ModifyOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, common.ErrNotYetImplemented
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
func (h *HUOBIHADAX) CancelAllOrders() error {
	return common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (h *HUOBIHADAX) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (h *HUOBIHADAX) GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (h *HUOBIHADAX) WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBIHADAX) WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBIHADAX) WithdrawFiatFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (h *HUOBIHADAX) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HUOBIHADAX) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return h.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (h *HUOBIHADAX) GetWithdrawCapabilities() uint32 {
	return h.GetWithdrawPermissions()
}
