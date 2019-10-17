package coinbene

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Start starts the Coinbene go routine
func (c *Coinbene) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
}

// Run implements the Coinbene wrapper
func (c *Coinbene) Run() {
	if c.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", c.GetName(), common.IsEnabled(c.Websocket.IsEnabled()), c.Websocket.GetWebsocketURL())
		log.Debugf("%s polling delay: %ds.\n", c.GetName(), c.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", c.GetName(), len(c.EnabledPairs), c.EnabledPairs)
	}
	exchangeCurrencies, err := c.GetAllPairs()
	if err != nil {
		log.Errorf("%s Failed to get available symbols.\n", c.GetName())
	} else {
		var newExchangeCurrencies currency.Pairs
		for p := range exchangeCurrencies.Data {
			newExchangeCurrencies = append(newExchangeCurrencies,
				currency.NewPairFromString(exchangeCurrencies.Data[p].Symbol))
		}
		err = c.UpdateCurrencies(newExchangeCurrencies, false, true)
		if err != nil {
			log.Errorf("%s Failed to update available currencies %s.\n", c.GetName(), err)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *Coinbene) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var resp ticker.Price
	allPairs := c.GetEnabledCurrencies()
	for x := range allPairs {
		tempResp, err := c.FetchTicker(exchange.FormatExchangeCurrency(c.Name, allPairs[x]).String())
		if err != nil {
			return resp, err
		}
		resp.Pair = allPairs[x]
		resp.Last = tempResp.TickerData.LatestPrice
		resp.High = tempResp.TickerData.DailyHigh
		resp.Low = tempResp.TickerData.DailyLow
		resp.Bid = tempResp.TickerData.BestBid
		resp.Ask = tempResp.TickerData.BestAsk
		resp.Volume = tempResp.TickerData.DailyVol
		resp.LastUpdated = time.Now()
		err = ticker.ProcessTicker(c.Name, &resp, assetType)
		if err != nil {
			return resp, err
		}
	}
	return ticker.GetTicker(c.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (c *Coinbene) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (c *Coinbene) GetOrderbookEx(currency currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(c.GetName(), currency, assetType)
	if err != nil {
		return c.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *Coinbene) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var resp orderbook.Base
	strPair := exchange.FormatExchangeCurrency(c.Name, p).String()
	tempResp, err := c.FetchOrderbooks(strPair, 100)
	if err != nil {
		return resp, err
	}
	resp.ExchangeName = c.Name
	resp.Pair = p
	resp.AssetType = assetType
	for i := range tempResp.Orderbook.Asks {
		var tempAsks orderbook.Item
		tempAsks.Amount, err = strconv.ParseFloat(tempResp.Orderbook.Asks[i][1], 64)
		if err != nil {
			return resp, err
		}
		tempAsks.Price, err = strconv.ParseFloat(tempResp.Orderbook.Asks[i][0], 64)
		if err != nil {
			return resp, err
		}
		resp.Asks = append(resp.Asks, tempAsks)
	}
	for j := range tempResp.Orderbook.Bids {
		var tempBids orderbook.Item
		tempBids.Amount, err = strconv.ParseFloat(tempResp.Orderbook.Bids[j][1], 64)
		if err != nil {
			return resp, err
		}
		tempBids.Price, err = strconv.ParseFloat(tempResp.Orderbook.Bids[j][0], 64)
		if err != nil {
			return resp, err
		}
		resp.Bids = append(resp.Bids, tempBids)
	}
	err = resp.Process()
	if err != nil {
		return resp, err
	}
	return orderbook.Get(c.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Coinbene exchange
func (c *Coinbene) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	data, err := c.GetUserBalance()
	if err != nil {
		return info, err
	}
	var account exchange.Account
	for key := range data.Data {
		c := currency.NewCode(data.Data[key].Asset)
		hold := data.Data[key].Reserved
		available := data.Data[key].Available
		account.Currencies = append(account.Currencies,
			exchange.AccountCurrencyInfo{CurrencyName: c,
				TotalValue: hold + available,
				Hold:       hold})
	}
	info.Accounts = append(info.Accounts, account)
	info.Exchange = c.Name
	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (c *Coinbene) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (c *Coinbene) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (c *Coinbene) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var resp exchange.SubmitOrderResponse
	if side != exchange.BuyOrderSide && side != exchange.SellOrderSide {
		return resp, fmt.Errorf("%s orderside is not supported by this exchange", side)
	}
	tempResp, err := c.PlaceOrder(price, amount, exchange.FormatExchangeCurrency(c.Name, p).String(), orderType.ToString(), clientID)
	if err != nil {
		return resp, err
	}
	resp.IsOrderPlaced = true
	resp.OrderID = tempResp.OrderID
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *Coinbene) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *Coinbene) CancelOrder(order *exchange.OrderCancellation) error {
	_, err := c.RemoveOrder(order.OrderID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *Coinbene) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	var resp exchange.CancelAllOrdersResponse
	tempMap := make(map[string]string)
	orders, err := c.FetchOpenOrders(exchange.FormatExchangeCurrency(c.Name, orderCancellation.CurrencyPair).String())
	if err != nil {
		return resp, err
	}
	for x := range orders.OpenOrders {
		_, err := c.RemoveOrder(orders.OpenOrders[x].OrderID)
		if err != nil {
			tempMap[orders.OpenOrders[x].OrderID] = "Failed"
		} else {
			tempMap[orders.OpenOrders[x].OrderID] = "Success"
		}
	}
	resp.OrderStatus = tempMap
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (c *Coinbene) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var resp exchange.OrderDetail
	tempResp, err := c.FetchOrderInfo(orderID)
	if err != nil {
		return resp, err
	}
	resp.Exchange = c.Name
	resp.ID = orderID
	resp.CurrencyPair = currency.NewPairWithDelimiter(tempResp.Order.BaseAsset, "/", tempResp.Order.QuoteAsset)
	orderTime, err := time.Parse(time.RFC3339, tempResp.Order.OrderTime)
	if err != nil {
		return resp, err
	}
	resp.OrderDate = orderTime
	resp.ExecutedAmount = tempResp.Order.FilledAmount
	resp.Fee = tempResp.Order.TotalFee
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *Coinbene) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (c *Coinbene) GetWebsocket() (*wshandler.Websocket, error) {
	return c.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (c *Coinbene) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var resp []exchange.OrderDetail
	var tempResp exchange.OrderDetail
	var tempData OpenOrderResponse
	if len(getOrdersRequest.Currencies) == 0 {
		allPairs, err := c.GetAllPairs()
		if err != nil {
			return resp, err
		}
		for a := range allPairs.Data {
			getOrdersRequest.Currencies = append(getOrdersRequest.Currencies, currency.NewPairFromString(allPairs.Data[a].Symbol))
		}
	}
	var err error
	for x := range getOrdersRequest.Currencies {
		tempData, err = c.FetchOpenOrders(exchange.FormatExchangeCurrency(c.Name, getOrdersRequest.Currencies[x]).String())
		if err != nil {
			return resp, err
		}
		for y := range tempData.OpenOrders {
			tempResp.Exchange = c.Name
			tempResp.CurrencyPair = getOrdersRequest.Currencies[x]
			if tempData.OpenOrders[y].OrderType == buy {
				tempResp.OrderSide = exchange.BuyOrderSide
			}
			if tempData.OpenOrders[y].OrderType == sell {
				tempResp.OrderSide = exchange.SellOrderSide
			}
			orderTime, err := time.Parse(time.RFC3339, tempData.OpenOrders[y].OrderTime)
			if err != nil {
				return resp, err
			}
			tempResp.OrderDate = orderTime
			tempResp.Status = tempData.OpenOrders[y].OrderStatus
			tempResp.Price = tempData.OpenOrders[y].AvgPrice
			tempResp.Amount = tempData.OpenOrders[y].Amount
			tempResp.ExecutedAmount = tempData.OpenOrders[y].FilledAmount
			tempResp.RemainingAmount = tempData.OpenOrders[y].Amount - tempData.OpenOrders[y].FilledAmount
			tempResp.Fee = tempData.OpenOrders[y].TotalFee
			resp = append(resp, tempResp)
		}
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *Coinbene) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var resp []exchange.OrderDetail
	var tempResp exchange.OrderDetail
	var tempData ClosedOrderResponse
	if len(getOrdersRequest.Currencies) == 0 {
		allPairs, err := c.GetAllPairs()
		if err != nil {
			return resp, err
		}
		for a := range allPairs.Data {
			getOrdersRequest.Currencies = append(getOrdersRequest.Currencies, currency.NewPairFromString(allPairs.Data[a].Symbol))
		}
	}
	var err error
	for x := range getOrdersRequest.Currencies {
		tempData, err = c.FetchClosedOrders(exchange.FormatExchangeCurrency(c.Name, getOrdersRequest.Currencies[x]).String(), "")
		if err != nil {
			return resp, err
		}
		for y := range tempData.Data {
			tempResp.Exchange = c.Name
			tempResp.CurrencyPair = getOrdersRequest.Currencies[x]
			if tempData.Data[y].OrderType == buy {
				tempResp.OrderSide = exchange.BuyOrderSide
			}
			if tempData.Data[y].OrderType == sell {
				tempResp.OrderSide = exchange.SellOrderSide
			}
			orderTime, err := time.Parse(time.RFC3339, tempData.Data[y].OrderTime)
			if err != nil {
				return resp, err
			}
			tempResp.OrderDate = orderTime
			tempResp.Status = tempData.Data[y].OrderStatus
			tempResp.Price = tempData.Data[y].AvgPrice
			tempResp.Amount = tempData.Data[y].Amount
			tempResp.ExecutedAmount = tempData.Data[y].FilledAmount
			tempResp.RemainingAmount = tempData.Data[y].Amount - tempData.Data[y].FilledAmount
			tempResp.Fee = tempData.Data[y].TotalFee
			resp = append(resp, tempResp)
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (c *Coinbene) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrFunctionNotSupported
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (c *Coinbene) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrNotYetImplemented
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (c *Coinbene) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrNotYetImplemented
}

// GetSubscriptions returns a copied list of subscriptions
func (c *Coinbene) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrNotYetImplemented
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (c *Coinbene) AuthenticateWebsocket() error {
	return common.ErrNotYetImplemented
}
