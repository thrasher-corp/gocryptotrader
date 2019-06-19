package lbank

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Start starts the Lbank go routine
func (l *Lbank) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
}

// Run implements the Lbank wrapper
func (l *Lbank) Run() {
	if l.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", l.GetName(), common.IsEnabled(l.Websocket.IsEnabled()), l.Websocket.GetWebsocketURL())
		log.Debugf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *Lbank) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tickerInfo, err := l.GetTicker(p.String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Last = tickerInfo.Ticker.Latest
	tickerPrice.High = tickerInfo.Ticker.High
	tickerPrice.Volume = tickerInfo.Ticker.Volume
	tickerPrice.Low = tickerInfo.Ticker.Low

	err = ticker.ProcessTicker(l.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(l.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (l *Lbank) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (l *Lbank) GetOrderbookEx(currency currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(l.GetName(), currency, assetType)
	if err != nil {
		return l.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *Lbank) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	a, err := l.GetMarketDepths(p.String(), "60", "1")
	if err != nil {
		return orderBook, err
	}
	for i := range a.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Price:  a.Asks[i][0],
			Amount: a.Asks[i][1]})
	}
	for i := range a.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Price:  a.Bids[i][0],
			Amount: a.Bids[i][1]})
	}
	orderBook.Pair = p
	orderBook.ExchangeName = l.GetName()
	orderBook.AssetType = assetType
	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(l.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Lbank exchange
func (l *Lbank) GetAccountInfo() (exchange.AccountInfo, error) {
	return exchange.AccountInfo{}, common.ErrNotYetImplemented
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *Lbank) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *Lbank) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (l *Lbank) SubmitOrder(p currency.Pair, side exchange.OrderSide, _ exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var resp exchange.SubmitOrderResponse
	return resp, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *Lbank) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
// func (l *Lbank) CancelOrder(order *exchange.OrderCancellation) error {
// 	return common.ErrNotYetImplemented
// }

// CancelAllOrders cancels all orders associated with a currency pair
func (l *Lbank) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	return exchange.CancelAllOrdersResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (l *Lbank) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	return exchange.OrderDetail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *Lbank) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (l *Lbank) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (l *Lbank) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (l *Lbank) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (l *Lbank) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}
