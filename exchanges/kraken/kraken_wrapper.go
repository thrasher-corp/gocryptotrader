package kraken

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

// Start starts the Kraken go routine
func (k *Kraken) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		k.Run()
		wg.Done()
	}()
}

// Run implements the Kraken wrapper
func (k *Kraken) Run() {
	if k.Verbose {
		log.Debugf("%s polling delay: %ds.\n", k.GetName(), k.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", k.GetName(), len(k.EnabledPairs), k.EnabledPairs)
	}

	assetPairs, err := k.GetAssetPairs()
	if err != nil {
		log.Errorf("%s Failed to get available symbols.\n", k.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(k.EnabledPairs.Strings(), "-") ||
			!common.StringDataContains(k.AvailablePairs.Strings(), "-") {
			forceUpgrade = true
		}

		var exchangeProducts []string
		for _, v := range assetPairs {
			if common.StringContains(v.Altname, ".d") {
				continue
			}
			if v.Base[0] == 'X' {
				if len(v.Base) > 3 {
					v.Base = v.Base[1:]
				}
			}
			if v.Quote[0] == 'Z' || v.Quote[0] == 'X' {
				v.Quote = v.Quote[1:]
			}
			exchangeProducts = append(exchangeProducts, v.Base+"-"+v.Quote)
		}

		if forceUpgrade {
			enabledPairs := currency.Pairs{currency.Pair{
				Base: currency.XBT, Quote: currency.USD, Delimiter: "-"}}

			log.Warn("Available pairs for Kraken reset due to config upgrade, please enable the ones you would like again")

			err = k.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Errorf("%s Failed to get config.\n", k.GetName())
			}
		}

		var newExchangeProducts currency.Pairs
		for _, p := range exchangeProducts {
			newExchangeProducts = append(newExchangeProducts,
				currency.NewPairFromString(p))
		}

		err = k.UpdateCurrencies(newExchangeProducts, false, forceUpgrade)
		if err != nil {
			log.Errorf("%s Failed to get config.\n", k.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (k *Kraken) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	pairs := k.GetEnabledCurrencies()
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(k.Name, pairs)
	if err != nil {
		return tickerPrice, err
	}
	tickers, err := k.GetTickers(pairsCollated)
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range pairs {
		for y, z := range tickers {
			if !common.StringContains(y, x.Base.Upper().String()) ||
				!common.StringContains(y, x.Quote.Upper().String()) {
				continue
			}
			var tp ticker.Price
			tp.Pair = x
			tp.Last = z.Last
			tp.Ask = z.Ask
			tp.Bid = z.Bid
			tp.High = z.High
			tp.Low = z.Low
			tp.Volume = z.Volume
			ticker.ProcessTicker(k.GetName(), &tp, assetType)
		}
	}
	return ticker.GetTicker(k.GetName(), p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (k *Kraken) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(k.GetName(), p, assetType)
	if err != nil {
		return k.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (k *Kraken) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(k.GetName(), p, assetType)
	if err != nil {
		return k.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (k *Kraken) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := k.GetDepth(exchange.FormatExchangeCurrency(k.GetName(), p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = k.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(k.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Kraken exchange - to-do
func (k *Kraken) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	info.Exchange = k.GetName()

	bal, err := k.GetBalance()
	if err != nil {
		return info, err
	}

	var balances []exchange.AccountCurrencyInfo
	for key, data := range bal {
		balances = append(balances, exchange.AccountCurrencyInfo{
			CurrencyName: currency.NewCode(key),
			TotalValue:   data,
		})
	}

	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balances,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (k *Kraken) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (k *Kraken) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (k *Kraken) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var args = AddOrderOptions{}

	response, err := k.AddOrder(p.String(),
		side.ToString(),
		orderType.ToString(),
		amount,
		price,
		0,
		0,
		&args)

	if len(response.TransactionIds) > 0 {
		submitOrderResponse.OrderID = strings.Join(response.TransactionIds, ", ")
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (k *Kraken) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (k *Kraken) CancelOrder(order *exchange.OrderCancellation) error {
	_, err := k.CancelExistingOrder(order.OrderID)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (k *Kraken) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	var emptyOrderOptions OrderInfoOptions
	openOrders, err := k.GetOpenOrders(emptyOrderOptions)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	if openOrders.Count > 0 {
		for orderID := range openOrders.Open {
			_, err = k.CancelExistingOrder(orderID)
			if err != nil {
				cancelAllOrdersResponse.OrderStatus[orderID] = err.Error()
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (k *Kraken) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (k *Kraken) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	methods, err := k.GetDepositMethods(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	var method string
	for _, m := range methods {
		method = m.Method
	}

	if method == "" {
		return "", errors.New("method not found")
	}

	return k.GetCryptoDepositAddress(method, cryptocurrency.String())
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal
// Populate exchange.WithdrawRequest.TradePassword with withdrawal key name, as set up on your account
func (k *Kraken) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return k.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.TradePassword, withdrawRequest.Amount)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (k *Kraken) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return k.WithdrawCryptocurrencyFunds(withdrawRequest)
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (k *Kraken) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return k.WithdrawCryptocurrencyFunds(withdrawRequest)
}

// GetWebsocket returns a pointer to the exchange websocket
func (k *Kraken) GetWebsocket() (*exchange.Websocket, error) {
	return k.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (k *Kraken) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (k.APIKey == "" || k.APISecret == "") && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return k.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (k *Kraken) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := k.GetOpenOrders(OrderInfoOptions{})
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp.Open {
		symbol := currency.NewPairDelimiter(resp.Open[i].Descr.Pair,
			k.ConfigCurrencyPairFormat.Delimiter)
		orderDate := time.Unix(int64(resp.Open[i].StartTm), 0)
		side := exchange.OrderSide(strings.ToUpper(resp.Open[i].Descr.Type))

		orders = append(orders, exchange.OrderDetail{
			ID:              i,
			Amount:          resp.Open[i].Vol,
			RemainingAmount: (resp.Open[i].Vol - resp.Open[i].VolExec),
			ExecutedAmount:  resp.Open[i].VolExec,
			Exchange:        k.Name,
			OrderDate:       orderDate,
			Price:           resp.Open[i].Price,
			OrderSide:       side,
			CurrencyPair:    symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (k *Kraken) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	req := GetClosedOrdersOptions{}
	if getOrdersRequest.StartTicks.Unix() > 0 {
		req.Start = fmt.Sprintf("%v", getOrdersRequest.StartTicks.Unix())
	}
	if getOrdersRequest.EndTicks.Unix() > 0 {
		req.End = fmt.Sprintf("%v", getOrdersRequest.EndTicks.Unix())
	}

	resp, err := k.GetClosedOrders(req)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp.Closed {
		symbol := currency.NewPairDelimiter(resp.Closed[i].Descr.Pair,
			k.ConfigCurrencyPairFormat.Delimiter)
		orderDate := time.Unix(int64(resp.Closed[i].StartTm), 0)
		side := exchange.OrderSide(strings.ToUpper(resp.Closed[i].Descr.Type))

		orders = append(orders, exchange.OrderDetail{
			ID:              i,
			Amount:          resp.Closed[i].Vol,
			RemainingAmount: (resp.Closed[i].Vol - resp.Closed[i].VolExec),
			ExecutedAmount:  resp.Closed[i].VolExec,
			Exchange:        k.Name,
			OrderDate:       orderDate,
			Price:           resp.Closed[i].Price,
			OrderSide:       side,
			CurrencyPair:    symbol,
		})
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}
