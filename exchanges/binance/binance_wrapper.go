package binance

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
func (b *Binance) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (b *Binance) Run() {
	if b.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n%s polling delay: %ds.\n%s %d currencies enabled: %s.\n",
			b.GetName(),
			common.IsEnabled(b.Websocket.IsEnabled()),
			b.Websocket.GetWebsocketURL(),
			b.GetName(),
			b.RESTPollingDelay,
			b.GetName(),
			len(b.EnabledPairs),
			b.EnabledPairs)
	}

	symbols, err := b.GetExchangeValidCurrencyPairs()
	if err != nil {
		log.Errorf("%s Failed to get exchange info.\n", b.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(b.EnabledPairs, "-") ||
			!common.StringDataContains(b.AvailablePairs, "-") {
			forceUpgrade = true
		}

		if forceUpgrade {
			enabledPairs := []string{"BTC-USDT"}
			log.Warn("Available pairs for Binance reset due to config upgrade, please enable the ones you would like again")

			err = b.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Errorf("%s Failed to get config.\n", b.GetName())
			}
		}
		err = b.UpdateCurrencies(symbols, false, forceUpgrade)
		if err != nil {
			log.Errorf("%s Failed to get config.\n", b.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Binance) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTickers()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range b.GetEnabledCurrencies() {
		curr := exchange.FormatExchangeCurrency(b.Name, x)
		for y := range tick {
			if tick[y].Symbol == curr.String() {
				tickerPrice.Pair = x
				tickerPrice.Ask = tick[y].AskPrice
				tickerPrice.Bid = tick[y].BidPrice
				tickerPrice.High = tick[y].HighPrice
				tickerPrice.Last = tick[y].LastPrice
				tickerPrice.Low = tick[y].LowPrice
				tickerPrice.Volume = tick[y].Volume
				ticker.ProcessTicker(b.Name, x, tickerPrice, assetType)
			}
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Binance) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (b *Binance) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), currency, assetType)
	if err != nil {
		return b.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Binance) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderBook(OrderBookDataRequestParams{Symbol: exchange.FormatExchangeCurrency(b.Name, p).String(), Limit: 1000})
	if err != nil {
		return orderBook, err
	}

	for _, bids := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{Amount: bids.Quantity, Price: bids.Price})
	}

	for _, asks := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{Amount: asks.Quantity, Price: asks.Price})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Bithumb exchange
func (b *Binance) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	raw, err := b.GetAccount()
	if err != nil {
		return info, err
	}

	var currencyBalance []exchange.AccountCurrencyInfo
	for _, balance := range raw.Balances {
		freeCurrency, err := strconv.ParseFloat(balance.Free, 64)
		if err != nil {
			return info, err
		}

		lockedCurrency, err := strconv.ParseFloat(balance.Locked, 64)
		if err != nil {
			return info, err
		}

		currencyBalance = append(currencyBalance, exchange.AccountCurrencyInfo{
			CurrencyName: balance.Asset,
			TotalValue:   freeCurrency + lockedCurrency,
			Hold:         freeCurrency,
		})
	}

	info.Exchange = b.GetName()
	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: currencyBalance,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Binance) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Binance) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory
	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Binance) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	var sideType RequestParamsSideType
	if side == exchange.Buy {
		sideType = BinanceRequestParamsSideBuy
	} else {
		sideType = BinanceRequestParamsSideSell
	}

	var requestParamsOrderType RequestParamsOrderType
	if orderType == exchange.Market {
		requestParamsOrderType = BinanceRequestParamsOrderMarket
	} else if orderType == exchange.Limit {
		requestParamsOrderType = BinanceRequestParamsOrderLimit
	} else {
		submitOrderResponse.IsOrderPlaced = false
		return submitOrderResponse, errors.New("Unsupported order type")
	}

	var orderRequest = NewOrderRequest{
		Symbol:    p.FirstCurrency.String() + p.SecondCurrency.String(),
		Side:      sideType,
		Price:     price,
		Quantity:  amount,
		TradeType: requestParamsOrderType,
	}

	response, err := b.NewOrder(orderRequest)

	if response.OrderID > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response.OrderID)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Binance) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Binance) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder(exchange.FormatExchangeCurrency(b.Name, order.CurrencyPair).String(),
		orderIDInt,
		order.AccountID)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Binance) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := b.OpenOrders("")
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, order := range openOrders {
		_, err = b.CancelExistingOrder(order.Symbol, order.OrderID, "")
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(order.OrderID, 10)] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Binance) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Binance) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return b.GetDepositAddressForCurrency(cryptocurrency.String())
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Binance) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	amountStr := strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64)
	id, err := b.WithdrawCrypto(withdrawRequest.Currency.String(), withdrawRequest.Address, withdrawRequest.AddressTag, withdrawRequest.Description, amountStr)

	return strconv.FormatInt(id, 10), err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Binance) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Binance) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (b *Binance) GetWithdrawCapabilities() uint32 {
	return b.GetWithdrawPermissions()
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Binance) GetOrderHistory(orderHistoryRequest exchange.OrderHistoryRequest) ([]exchange.OrderDetail, error) {
	if len(orderHistoryRequest.Currencies) <= 0 {
		return nil, errors.New("At least one currency is required to fetch order history")
	}

	var orders []exchange.OrderDetail

	for _, symbol := range orderHistoryRequest.Currencies {
		resp, err := b.AllOrders(symbol, "", "1000")
		if err != nil {
			return nil, err
		}

		for _, openOrder := range resp {
			orderDetail := exchange.OrderDetail{
				Amount:              openOrder.OrigQty,
				BaseCurrency:        openOrder.Symbol,
				OrderPlacementTicks: int64(openOrder.Time),
				Exchange:            b.Name,
				ID:                  fmt.Sprintf("%v", openOrder.OrderID),
				OrderSide:           openOrder.Side,
				OrderType:           openOrder.Type,
				Price:               openOrder.Price,
				QuoteCurrency:       openOrder.Symbol,
				Status:              openOrder.Status,
			}

			orders = append(orders, orderDetail)
		}
	}

	b.FilterOrdersByStatusAndType(&orders, orderHistoryRequest.OrderType, orderHistoryRequest.OrderStatus)
	b.FilterOrdersByTickRange(&orders, orderHistoryRequest.StartTicks, orderHistoryRequest.EndTicks)

	return orders, nil
}
