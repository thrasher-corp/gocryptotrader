package gateio

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ws/monitor"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Start starts the GateIO go routine
func (g *Gateio) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		g.Run()
		wg.Done()
	}()
}

// Run implements the GateIO wrapper
func (g *Gateio) Run() {
	if g.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", g.GetName(), common.IsEnabled(g.Websocket.IsEnabled()), g.WebsocketURL)
		log.Debugf("%s polling delay: %ds.\n", g.GetName(), g.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.EnabledPairs), g.EnabledPairs)
	}

	symbols, err := g.GetSymbols()
	if err != nil {
		log.Errorf("%s Unable to fetch symbols.\n", g.GetName())
	} else {
		var newCurrencies currency.Pairs
		for _, p := range symbols {
			newCurrencies = append(newCurrencies,
				currency.NewPairFromString(p))
		}

		err = g.UpdateCurrencies(newCurrencies, false, false)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", g.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gateio) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	result, err := g.GetTickers()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range g.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(g.Name, x).String()
		var tp ticker.Price
		tp.Pair = x
		tp.High = result[currency].High
		tp.Last = result[currency].Last
		tp.Last = result[currency].Last
		tp.Low = result[currency].Low
		tp.Volume = result[currency].Volume

		err = ticker.ProcessTicker(g.Name, &tp, assetType)
		if err != nil {
			return tickerPrice, err
		}
	}

	return ticker.GetTicker(g.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (g *Gateio) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (g *Gateio) GetOrderbookEx(currency currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(g.GetName(), currency, assetType)
	if err != nil {
		return g.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gateio) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	currency := exchange.FormatExchangeCurrency(g.Name, p).String()

	orderbookNew, err := g.GetOrderbook(currency)
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

	orderBook.Pair = p
	orderBook.ExchangeName = g.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(g.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// ZB exchange
func (g *Gateio) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo

	balance, err := g.GetBalances()
	if err != nil {
		return info, err
	}

	var balances []exchange.AccountCurrencyInfo

	switch l := balance.Locked.(type) {
	case map[string]interface{}:
		for x := range l {
			lockedF, err := strconv.ParseFloat(l[x].(string), 64)
			if err != nil {
				return info, err
			}

			balances = append(balances, exchange.AccountCurrencyInfo{
				CurrencyName: currency.NewCode(x),
				Hold:         lockedF,
			})
		}
	default:
		break
	}

	switch v := balance.Available.(type) {
	case map[string]interface{}:
		for x := range v {
			availAmount, err := strconv.ParseFloat(v[x].(string), 64)
			if err != nil {
				return info, err
			}

			var updated bool
			for i := range balances {
				if balances[i].CurrencyName == currency.NewCode(x) {
					balances[i].TotalValue = balances[i].Hold + availAmount
					updated = true
					break
				}
			}
			if !updated {
				balances = append(balances, exchange.AccountCurrencyInfo{
					CurrencyName: currency.NewCode(x),
					TotalValue:   availAmount,
				})
			}
		}
	default:
		break
	}

	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balances,
	})

	info.Exchange = g.GetName()

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gateio) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (g *Gateio) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
// TODO: support multiple order types (IOC)
func (g *Gateio) SubmitOrder(p currency.Pair, side exchange.OrderSide, _ exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var orderTypeFormat SpotNewOrderRequestParamsType

	if side == exchange.BuyOrderSide {
		orderTypeFormat = SpotNewOrderRequestParamsTypeBuy
	} else {
		orderTypeFormat = SpotNewOrderRequestParamsTypeSell
	}

	var spotNewOrderRequestParams = SpotNewOrderRequestParams{
		Amount: amount,
		Price:  price,
		Symbol: p.String(),
		Type:   orderTypeFormat,
	}

	response, err := g.SpotNewOrder(spotNewOrderRequestParams)

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
func (g *Gateio) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (g *Gateio) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}
	_, err = g.CancelExistingOrder(orderIDInt, exchange.FormatExchangeCurrency(g.Name, order.CurrencyPair).String())

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (g *Gateio) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := g.GetOpenOrders("")
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	uniqueSymbols := make(map[string]int)
	for i := range openOrders.Orders {
		uniqueSymbols[openOrders.Orders[i].CurrencyPair]++
	}

	for unique := range uniqueSymbols {
		err = g.CancelAllExistingOrders(-1, unique)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[unique] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (g *Gateio) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail

	orders, err := g.GetOpenOrders("")
	if err != nil {
		return orderDetail, errors.New("failed to get open orders")
	}
	for x := range orders.Orders {
		if orders.Orders[x].OrderNumber != orderID {
			continue
		}
		orderDetail.Exchange = g.GetName()
		orderDetail.ID = orders.Orders[x].OrderNumber
		orderDetail.RemainingAmount = orders.Orders[x].InitialAmount - orders.Orders[x].FilledAmount
		orderDetail.ExecutedAmount = orders.Orders[x].FilledAmount
		orderDetail.Amount = orders.Orders[x].InitialAmount
		orderDetail.OrderDate = time.Unix(orders.Orders[x].Timestamp, 0)
		orderDetail.Status = orders.Orders[x].Status
		orderDetail.Price = orders.Orders[x].Rate
		orderDetail.CurrencyPair = currency.NewPairDelimiter(orders.Orders[x].CurrencyPair, g.ConfigCurrencyPairFormat.Delimiter)
		if strings.EqualFold(orders.Orders[x].Type, exchange.AskOrderSide.ToString()) {
			orderDetail.OrderSide = exchange.AskOrderSide
		} else if strings.EqualFold(orders.Orders[x].Type, exchange.BidOrderSide.ToString()) {
			orderDetail.OrderSide = exchange.BuyOrderSide
		}
		return orderDetail, nil
	}
	return orderDetail, fmt.Errorf("no order found with id %v", orderID)
}

// GetDepositAddress returns a deposit address for a specified currency
func (g *Gateio) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	addr, err := g.GetCryptoDepositAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	// Waits for new generated address if not created yet, its variable per
	// currency
	if addr == gateioGenerateAddress {
		time.Sleep(10 * time.Second)
		addr, err = g.GetCryptoDepositAddress(cryptocurrency.String())
		if err != nil {
			return "", err
		}
		if addr == gateioGenerateAddress {
			return "", errors.New("new deposit address is being generated, please retry again shortly")
		}
		return addr, nil
	}

	return addr, err
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (g *Gateio) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return g.WithdrawCrypto(withdrawRequest.Currency.String(), withdrawRequest.Address, withdrawRequest.Amount)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gateio) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gateio) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (g *Gateio) GetWebsocket() (*monitor.Websocket, error) {
	return g.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gateio) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (g.APIKey == "" || g.APISecret == "") && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return g.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (g *Gateio) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var currPair string
	if len(getOrdersRequest.Currencies) == 1 {
		currPair = getOrdersRequest.Currencies[0].String()
	}

	resp, err := g.GetOpenOrders(currPair)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp.Orders {
		if resp.Orders[i].Status != "open" {
			continue
		}

		symbol := currency.NewPairDelimiter(resp.Orders[i].CurrencyPair,
			g.ConfigCurrencyPairFormat.Delimiter)
		side := exchange.OrderSide(strings.ToUpper(resp.Orders[i].Type))
		orderDate := time.Unix(resp.Orders[i].Timestamp, 0)

		orders = append(orders, exchange.OrderDetail{
			ID:              resp.Orders[i].OrderNumber,
			Amount:          resp.Orders[i].Amount,
			Price:           resp.Orders[i].Rate,
			RemainingAmount: resp.Orders[i].FilledAmount,
			OrderDate:       orderDate,
			OrderSide:       side,
			Exchange:        g.Name,
			CurrencyPair:    symbol,
			Status:          resp.Orders[i].Status,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gateio) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var trades []TradesResponse
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := g.GetTradeHistory(currency.String())
		if err != nil {
			return nil, err
		}
		trades = append(trades, resp.Trades...)
	}

	var orders []exchange.OrderDetail
	for _, trade := range trades {
		symbol := currency.NewPairDelimiter(trade.Pair,
			g.ConfigCurrencyPairFormat.Delimiter)
		side := exchange.OrderSide(strings.ToUpper(trade.Type))
		orderDate := time.Unix(trade.TimeUnix, 0)
		orders = append(orders, exchange.OrderDetail{
			ID:           strconv.FormatInt(trade.OrderID, 10),
			Amount:       trade.Amount,
			Price:        trade.Rate,
			OrderDate:    orderDate,
			OrderSide:    side,
			Exchange:     g.Name,
			CurrencyPair: symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (g *Gateio) SubscribeToWebsocketChannels(channels []monitor.WebsocketChannelSubscription) error {
	g.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (g *Gateio) UnsubscribeToWebsocketChannels(channels []monitor.WebsocketChannelSubscription) error {
	g.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (g *Gateio) GetSubscriptions() ([]monitor.WebsocketChannelSubscription, error) {
	return g.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (g *Gateio) AuthenticateWebsocket() error {
	_, err := g.wsServerSignIn()
	return err
}
