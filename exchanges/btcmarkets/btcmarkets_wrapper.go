package btcmarkets

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the BTC Markets go routine
func (b *BTCMarkets) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTC Markets wrapper
func (b *BTCMarkets) Run() {
	if b.Verbose {
		log.Debugf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	markets, err := b.GetMarkets()
	if err != nil {
		log.Errorf("%s failed to get active market. Err: %s", b.Name, err)
	} else {
		forceUpgrade := false
		if !common.StringDataContains(b.EnabledPairs, "-") || !common.StringDataContains(b.AvailablePairs, "-") {
			forceUpgrade = true
		}

		var currencies []string
		for x := range markets {
			currencies = append(currencies, markets[x].Instrument+"-"+markets[x].Currency)
		}

		if forceUpgrade {
			enabledPairs := []string{"BTC-AUD"}
			log.Warn("Available pairs for BTC Makrets reset due to config upgrade, please enable the pairs you would like again.")

			err = b.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Errorf("%s failed to update currencies. Err: %s", b.Name, err)
			}
		}
		err = b.UpdateCurrencies(currencies, false, forceUpgrade)
		if err != nil {
			log.Errorf("%s failed to update currencies. Err: %s", b.Name, err)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCMarkets) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(p.FirstCurrency.String(),
		p.SecondCurrency.String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.BestAsk
	tickerPrice.Bid = tick.BestBID
	tickerPrice.Last = tick.LastPrice
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *BTCMarkets) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (b *BTCMarkets) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCMarkets) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(p.FirstCurrency.String(),
		p.SecondCurrency.String())
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

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// BTCMarkets exchange
func (b *BTCMarkets) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = b.GetName()

	accountBalance, err := b.GetAccountBalance()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Balance
		exchangeCurrency.Hold = accountBalance[i].PendingFunds

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTCMarkets) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTCMarkets) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *BTCMarkets) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	response, err := b.NewOrder(p.FirstCurrency.Upper().String(), p.SecondCurrency.Upper().String(), price, amount, side.ToString(), orderType.ToString(), clientID)

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
func (b *BTCMarkets) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTCMarkets) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder([]int64{orderIDInt})
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *BTCMarkets) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := b.GetOpenOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var orderList []int64
	for _, order := range openOrders {
		orderIDInt, err := strconv.ParseInt(order.ID, 10, 64)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[order.ID] = err.Error()
		}
		orderList = append(orderList, orderIDInt)
	}

	if len(orderList) > 0 {
		orders, err := b.CancelExistingOrder(orderList)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		for _, order := range orders {
			if err != nil {
				cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(order.ID, 10)] = err.Error()
			}
		}
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *BTCMarkets) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var OrderDetail exchange.OrderDetail

	orders, err := b.GetOrderDetail([]int64{orderID})
	if err != nil {
		return OrderDetail, err
	}

	if len(orders) > 1 {
		return OrderDetail, errors.New("too many orders returned")
	}

	if len(orders) == 0 {
		return OrderDetail, errors.New("no orders found")
	}

	for _, order := range orders {
		var side exchange.OrderSide
		if order.OrderSide == "ask" {
			side = exchange.SellOrderSide
		} else if order.OrderSide == "bid" {
			side = exchange.BuyOrderSide
		}
		orderDate := time.Unix(int64(order.CreationTime), 0)
		orderType := exchange.OrderType(strings.ToUpper(order.OrderType))

		OrderDetail.Amount = order.Volume
		OrderDetail.OrderDate = orderDate
		OrderDetail.Exchange = b.GetName()
		OrderDetail.ID = order.ID
		OrderDetail.RemainingAmount = order.OpenVolume
		OrderDetail.OrderSide = side
		OrderDetail.OrderType = orderType
		OrderDetail.Price = order.Price
		OrderDetail.Status = order.Status
		OrderDetail.CurrencyPair = pair.NewCurrencyPairWithDelimiter(order.Instrument, order.Currency, b.ConfigCurrencyPairFormat.Delimiter)
	}

	return OrderDetail, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTCMarkets) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (b *BTCMarkets) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return b.WithdrawCrypto(withdrawRequest.Amount, withdrawRequest.Currency.String(), withdrawRequest.Address)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	if withdrawRequest.Currency != symbol.AUD {
		return "", errors.New("Only AUD supported for withdrawals")
	}
	return b.WithdrawAUD(withdrawRequest.BankAccountName, fmt.Sprintf("%v", withdrawRequest.BankAccountNumber), withdrawRequest.BankName, fmt.Sprintf("%v", withdrawRequest.BankCode), withdrawRequest.Amount)
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTCMarkets) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTCMarkets) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (b *BTCMarkets) GetWithdrawCapabilities() uint32 {
	return b.GetWithdrawPermissions()
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTCMarkets) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := b.GetOpenOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, order := range resp {
		var side exchange.OrderSide
		if order.OrderSide == "ask" {
			side = exchange.SellOrderSide
		} else if order.OrderSide == "bid" {
			side = exchange.BuyOrderSide
		}
		orderDate := time.Unix(int64(order.CreationTime), 0)
		orderType := exchange.OrderType(strings.ToUpper(order.OrderType))

		openOrder := exchange.OrderDetail{
			ID:              order.ID,
			Amount:          order.Volume,
			Exchange:        b.Name,
			RemainingAmount: order.OpenVolume,
			OrderDate:       orderDate,
			OrderSide:       side,
			OrderType:       orderType,
			Price:           order.Price,
			Status:          order.Status,
			CurrencyPair:    pair.NewCurrencyPairWithDelimiter(order.Instrument, order.Currency, b.ConfigCurrencyPairFormat.Delimiter),
		}

		for _, trade := range order.Trades {
			tradeDate := time.Unix(int64(trade.CreationTime), 0)
			openOrder.Trades = append(openOrder.Trades, exchange.TradeHistory{
				Amount:      trade.Volume,
				Exchange:    b.Name,
				Price:       trade.Price,
				TID:         trade.ID,
				Timestamp:   tradeDate,
				Fee:         trade.Fee,
				Description: trade.Description,
			})
		}

		orders = append(orders, openOrder)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTCMarkets) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) <= 0 {
		return nil, errors.New("Requires at least one currency pair to retrieve history")
	}

	var respOrders []Order
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := b.GetOrders(currency.FirstCurrency.String(), currency.SecondCurrency.String(), 200, 0, true)
		if err != nil {
			return nil, err
		}
		for _, order := range resp {
			respOrders = append(respOrders, order)
		}
	}

	var orders []exchange.OrderDetail
	for _, order := range respOrders {
		var side exchange.OrderSide
		if order.OrderSide == "ask" {
			side = exchange.SellOrderSide
		} else if order.OrderSide == "bid" {
			side = exchange.BuyOrderSide
		}
		orderDate := time.Unix(int64(order.CreationTime), 0)
		orderType := exchange.OrderType(strings.ToUpper(order.OrderType))

		openOrder := exchange.OrderDetail{
			ID:              order.ID,
			Amount:          order.Volume,
			Exchange:        b.Name,
			RemainingAmount: order.OpenVolume,
			OrderDate:       orderDate,
			OrderSide:       side,
			OrderType:       orderType,
			Price:           order.Price,
			Status:          order.Status,
			CurrencyPair:    pair.NewCurrencyPairWithDelimiter(order.Instrument, order.Currency, b.ConfigCurrencyPairFormat.Delimiter),
		}

		for _, trade := range order.Trades {
			tradeDate := time.Unix(int64(trade.CreationTime), 0)
			openOrder.Trades = append(openOrder.Trades, exchange.TradeHistory{
				Amount:      trade.Volume,
				Exchange:    b.Name,
				Price:       trade.Price,
				TID:         trade.ID,
				Timestamp:   tradeDate,
				Fee:         trade.Fee,
				Description: trade.Description,
			})
		}

		orders = append(orders, openOrder)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}
