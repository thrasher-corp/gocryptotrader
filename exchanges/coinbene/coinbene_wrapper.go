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
		for p := range exchangeCurrencies.Symbol {
			newExchangeCurrencies = append(newExchangeCurrencies,
				currency.NewPairFromString(exchangeCurrencies.Symbol[p].Symbol))
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
	var tempResp TickerResponse
	var err error
	allPairs := c.GetEnabledCurrencies()
	log.Println(allPairs)
	for x := range allPairs {
		tempResp, err = c.FetchTicker(exchange.FormatExchangeCurrency(c.Name, allPairs[x]).String())
		if err != nil {
			return resp, err
		}
		resp.Pair = allPairs[x]
		resp.Last = tempResp.TickerData[0].Last
		resp.High = tempResp.TickerData[0].DailyHigh
		resp.Low = tempResp.TickerData[0].DailyLow
		resp.Bid = tempResp.TickerData[0].Bid
		resp.Ask = tempResp.TickerData[0].Ask
		resp.Volume = tempResp.TickerData[0].DailyVol
		resp.LastUpdated = time.Now()
		ticker.ProcessTicker(c.Name, &resp, assetType)
	}
	resp, err = ticker.GetTicker(c.Name, p, assetType)
	if err != nil {
		return resp, err
	}
	return resp, nil
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
	tempResp, err := c.FetchOrderbooks(strPair)
	if err != nil {
		return resp, err
	}
	resp.ExchangeName = c.Name
	resp.Pair = p
	resp.AssetType = assetType
	for i := range tempResp.Orderbook.Asks {
		var tempAsks orderbook.Item
		tempAsks.Amount = tempResp.Orderbook.Asks[i].Quantity
		tempAsks.Price = tempResp.Orderbook.Asks[i].Price
		resp.Asks = append(resp.Asks, tempAsks)
	}
	for j := range tempResp.Orderbook.Bids {
		var tempBids orderbook.Item
		tempBids.Amount = tempResp.Orderbook.Bids[j].Quantity
		tempBids.Price = tempResp.Orderbook.Bids[j].Price
		resp.Bids = append(resp.Bids, tempBids)
	}
	err = resp.Process()
	if err != nil {
		return resp, err
	}
	return resp, nil
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
	for key := range data.Balance {
		c := currency.NewCode(data.Balance[key].Asset)
		hold := data.Balance[key].Reserved
		available := data.Balance[key].Available
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
	return nil, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (c *Coinbene) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (c *Coinbene) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var resp exchange.SubmitOrderResponse
	if side != exchange.BuyOrderSide && side != exchange.SellOrderSide {
		return resp, fmt.Errorf("%s orderside is not supported by this exchange", side)
	}
	tempResp, err := c.PlaceOrder(price, amount, p.Lower().String(), orderType.ToString())
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
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (c *Coinbene) CancelOrder(order *exchange.OrderCancellation) error {
	_, err := c.RemoveOrder(order.OrderID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *Coinbene) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	var resp exchange.CancelAllOrdersResponse
	orders, err := c.FetchOpenOrders(orderCancellation.CurrencyPair.Lower().String())
	if err != nil {
		return resp, err
	}
	for x := range orders.OpenOrders {
		_, err := c.RemoveOrder(orders.OpenOrders[x].OrderID)
		if err != nil {
			resp.OrderStatus[orders.OpenOrders[x].OrderID] = "Failed"
		} else {
			resp.OrderStatus[orders.OpenOrders[x].OrderID] = "Success"
		}
	}
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
	resp.CurrencyPair = currency.NewPairFromString(tempResp.Order.Symbol)
	timestamp, err := strconv.ParseInt(tempResp.Order.CreateTime, 10, 64)
	if err != nil {
		return resp, err
	}
	resp.OrderDate = time.Unix(timestamp, 9)
	resp.ExecutedAmount = tempResp.Order.FilledAmount
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *Coinbene) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (c *Coinbene) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (c *Coinbene) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *Coinbene) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (c *Coinbene) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
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

// // GetAllOpenOrderID returns all open orders by currency pairs
// func (c *Coinbene) GetAllOpenOrderID() (map[string][]string, error) {
// 	allPairs := c.GetEnabledCurrencies()
// 	resp := make(map[string][]string)
// 	for a := range allPairs {
// 		p := exchange.FormatExchangeCurrency(c.Name, allPairs[a])
// 		b := int64(1)
// 		tempResp, err := c.FetchOpenOrders(p.String(), strconv.Format
// 	}
// }
