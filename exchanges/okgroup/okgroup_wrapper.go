package okgroup

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
func (o *OKGroup) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (o *OKGroup) Run() {
	if o.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket.IsEnabled()), o.WebsocketURL)
		log.Debugf("%s polling delay: %ds.\n", o.GetName(), o.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", o.GetName(), len(o.EnabledPairs), o.EnabledPairs)
	}

	prods, err := o.GetSpotInstruments()
	if err != nil {
		log.Errorf("OKEX failed to obtain available spot instruments. Err: %d", err)
		return
	}

	var pairs []string
	for x := range prods {
		pairs = append(pairs, prods[x].BaseCurrency+"_"+prods[x].QuoteCurrency)
	}

	err = o.UpdateCurrencies(pairs, false, false)
	if err != nil {
		log.Errorf("OKEX failed to update available currencies. Err: %s", err)
		return
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKGroup) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	currency := exchange.FormatExchangeCurrency(o.Name, p).String()
	var tickerPrice ticker.Price

	if assetType != ticker.Spot {
		tick, err := o.GetContractPrice(currency, assetType)
		if err != nil {
			return tickerPrice, err
		}

		tickerPrice.Pair = p
		tickerPrice.Ask = tick.Ticker.Sell
		tickerPrice.Bid = tick.Ticker.Buy
		tickerPrice.Low = tick.Ticker.Low
		tickerPrice.Last = tick.Ticker.Last
		tickerPrice.Volume = tick.Ticker.Vol
		tickerPrice.High = tick.Ticker.High
		ticker.ProcessTicker(o.GetName(), p, tickerPrice, assetType)
	} else {
		tick, err := o.GetSpotTicker(currency)
		if err != nil {
			return tickerPrice, err
		}
		tickerPrice.Pair = p
		tickerPrice.Ask = tick.Ticker.Sell
		tickerPrice.Bid = tick.Ticker.Buy
		tickerPrice.Low = tick.Ticker.Low
		tickerPrice.Last = tick.Ticker.Last
		tickerPrice.Volume = tick.Ticker.Vol
		tickerPrice.High = tick.Ticker.High
		ticker.ProcessTicker(o.GetName(), p, tickerPrice, ticker.Spot)

	}
	return ticker.GetTicker(o.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (o *OKGroup) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(o.GetName(), p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (o *OKGroup) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(o.GetName(), currency, assetType)
	if err != nil {
		return o.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *OKGroup) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	currency := exchange.FormatExchangeCurrency(o.Name, p).String()

	if assetType != ticker.Spot {
		orderbookNew, err := o.GetContractMarketDepth(currency, assetType)
		if err != nil {
			return orderBook, err
		}

		for x := range orderbookNew.Bids {
			data := orderbookNew.Bids[x]
			orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Volume, Price: data.Price})
		}

		for x := range orderbookNew.Asks {
			data := orderbookNew.Asks[x]
			orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Volume, Price: data.Price})
		}

	} else {
		orderbookNew, err := o.GetSpotMarketDepth(ActualSpotDepthRequestParams{
			Symbol: currency,
			Size:   200,
		})
		if err != nil {
			return orderBook, err
		}

		for x := range orderbookNew.Bids {
			data := orderbookNew.Bids[x]
			orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Volume, Price: data.Price})
		}

		for x := range orderbookNew.Asks {
			data := orderbookNew.Asks[x]
			orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Volume, Price: data.Price})
		}
	}

	orderbook.ProcessOrderbook(o.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(o.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// OKEX exchange
func (o *OKGroup) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	bal, err := o.GetBalance()
	if err != nil {
		return info, err
	}

	var balances []exchange.AccountCurrencyInfo
	for _, data := range bal {
		balances = append(balances, exchange.AccountCurrencyInfo{
			CurrencyName: data.Currency,
			TotalValue:   data.Available + data.Hold,
			Hold:         data.Hold,
		})
	}

	info.Exchange = o.GetName()
	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balances,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (o *OKGroup) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (o *OKGroup) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (o *OKGroup) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var oT SpotNewOrderRequestType

	if orderType == exchange.Limit {
		if side == exchange.Buy {
			oT = SpotNewOrderRequestTypeBuy
		} else {
			oT = SpotNewOrderRequestTypeSell
		}
	} else if orderType == exchange.Market {
		if side == exchange.Buy {
			oT = SpotNewOrderRequestTypeBuyMarket
		} else {
			oT = SpotNewOrderRequestTypeSellMarket
		}
	} else {
		return submitOrderResponse, errors.New("Unsupported order type")
	}

	var params = SpotNewOrderRequestParams{
		Amount: amount,
		Price:  price,
		Symbol: p.Pair().String(),
		Type:   oT,
	}

	response, err := o.SpotNewOrder(params)

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
func (o *OKGroup) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (o *OKGroup) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = o.SpotCancelOrder(exchange.FormatExchangeCurrency(o.Name, order.CurrencyPair).String(), orderIDInt)
	return err
}

// CancelAllOrders cancels all orders for all enabled currencies
func (o *OKGroup) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	var allOpenOrders []TokenOrder
	for _, currency := range o.GetEnabledCurrencies() {
		formattedCurrency := exchange.FormatExchangeCurrency(o.Name, currency).String()
		openOrders, err := o.GetTokenOrders(formattedCurrency, -1)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		if !openOrders.Result {
			return cancelAllOrdersResponse, fmt.Errorf("Something went wrong for currency %s", formattedCurrency)
		}

		for _, openOrder := range openOrders.Orders {
			allOpenOrders = append(allOpenOrders, openOrder)
		}
	}

	for _, openOrder := range allOpenOrders {
		_, err := o.SpotCancelOrder(openOrder.Symbol, openOrder.OrderID)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(openOrder.OrderID, 10)] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (o *OKGroup) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (o *OKGroup) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	// NOTE needs API version update to access
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (o *OKGroup) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	resp, err := o.Withdrawal(withdrawRequest.Currency.String(), withdrawRequest.FeeAmount, withdrawRequest.TradePassword, withdrawRequest.Address, withdrawRequest.Amount)
	return fmt.Sprintf("%v", resp), err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKGroup) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKGroup) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (o *OKGroup) GetWebsocket() (*exchange.Websocket, error) {
	return o.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (o *OKGroup) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return o.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (o *OKGroup) GetWithdrawCapabilities() uint32 {
	return o.GetWithdrawPermissions()
}
