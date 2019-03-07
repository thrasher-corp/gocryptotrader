package btse

import (
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the BTSE go routine
func (b *BTSE) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTSE wrapper
func (b *BTSE) Run() {
	if b.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()), b.Websocket.GetWebsocketURL())
		log.Debugf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	markets, err := b.GetMarkets()
	if err != nil {
		log.Errorf("%s failed to get trading pairs. Err: %s", b.Name, err)
	} else {
		var currencies []string
		for _, m := range *markets {
			currencies = append(currencies, m.ID)
		}
		err = b.UpdateCurrencies(currencies, false, false)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", b.Name)
		}
	}

}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTSE) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price

	t, err := b.GetTicker(exchange.FormatExchangeCurrency(b.Name, p).String())
	if err != nil {
		return tickerPrice, err
	}

	s, err := b.GetMarketStatistics(exchange.FormatExchangeCurrency(b.Name, p).String())
	if err != nil {
		return tickerPrice, err

	}

	tickerPrice.Pair = p
	tickerPrice.Ask = t.Ask
	tickerPrice.Bid = t.Bid
	tickerPrice.Low = s.Low
	tickerPrice.Last = t.Price
	tickerPrice.Volume = s.Volume
	tickerPrice.High = s.High
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *BTSE) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (b *BTSE) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTSE) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	return orderbook.Base{}, common.ErrFunctionNotSupported
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// BTSE exchange
func (b *BTSE) GetAccountInfo() (exchange.AccountInfo, error) {
	return exchange.AccountInfo{}, common.ErrNotYetImplemented
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTSE) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTSE) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *BTSE) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	return exchange.SubmitOrderResponse{}, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTSE) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTSE) CancelOrder(order exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *BTSE) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	return exchange.CancelAllOrdersResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (b *BTSE) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	return exchange.OrderDetail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTSE) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTSE) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTSE) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTSE) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}
