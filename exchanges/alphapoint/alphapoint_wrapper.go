package alphapoint

import (
	"errors"

	"github.com/idoall/gocryptotrader/currency/pair"
	"github.com/idoall/gocryptotrader/exchanges"
	"github.com/idoall/gocryptotrader/exchanges/orderbook"
	"github.com/idoall/gocryptotrader/exchanges/ticker"
)

// GetExchangeAccountInfo retrieves balances for all enabled currencies on the
// Alphapoint exchange
func (a *Alphapoint) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = a.GetName()
	account, err := a.GetAccountInfo()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(account.Currencies); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = account.Currencies[i].Name
		exchangeCurrency.TotalValue = float64(account.Currencies[i].Balance)
		exchangeCurrency.Hold = float64(account.Currencies[i].Hold)

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	//If it all works out
	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (a *Alphapoint) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := a.GetTicker(p.Pair().String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Low = tick.Low
	tickerPrice.High = tick.High
	tickerPrice.Volume = tick.Volume
	tickerPrice.Last = tick.Last
	ticker.ProcessTicker(a.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(a.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (a *Alphapoint) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *Alphapoint) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := a.GetOrderbook(p.Pair().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Quantity, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Quantity, Price: data.Price})
	}

	orderbook.ProcessOrderbook(a.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(a.Name, p, assetType)
}

// GetOrderbookEx returns the orderbook for a currency pair
func (a *Alphapoint) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (a *Alphapoint) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order and returns a true value when
// successfully submitted
func (a *Alphapoint) SubmitExchangeOrder(p pair.CurrencyPair, side string, orderType int, amount, price float64) (int64, error) {
	return a.CreateOrder(p.Pair().String(), side, orderType, amount, price)
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (a *Alphapoint) ModifyExchangeOrder(p pair.CurrencyPair, orderID, action int64) (int64, error) {
	return a.ModifyOrder(p.Pair().String(), orderID, action)
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (a *Alphapoint) CancelExchangeOrder(p pair.CurrencyPair, orderID int64) (int64, error) {
	return a.CancelOrder(p.Pair().String(), orderID)
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (a *Alphapoint) CancelAllExchangeOrders(p pair.CurrencyPair) error {
	return a.CancelAllOrders(p.Pair().String())
}

// GetExchangeOrderInfo returns information on a current open order
func (a *Alphapoint) GetExchangeOrderInfo(orderID int64) (float64, error) {
	orders, err := a.GetOrders()
	if err != nil {
		return 0, err
	}

	for x := range orders {
		for y := range orders[x].Openorders {
			if int64(orders[x].Openorders[y].Serverorderid) == orderID {
				return float64(orders[x].Openorders[y].QtyRemaining), nil
			}
		}
	}
	return 0, errors.New("order not found")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (a *Alphapoint) GetExchangeDepositAddress(p pair.CurrencyPair) (string, error) {
	addreses, err := a.GetDepositAddresses()
	if err != nil {
		return "", err
	}

	for x := range addreses {
		if addreses[x].Name == p.Pair().String() {
			return addreses[x].DepositAddress, nil
		}
	}
	return "", errors.New("associated currency address not found")
}

// WithdrawExchangeFunds returns a withdrawal ID when a withdrawal is submitted
func (a *Alphapoint) WithdrawExchangeFunds(address string, p pair.CurrencyPair, amount float64) (string, error) {
	return "", a.WithdrawCoins(p.Pair().String(), p.GetFirstCurrency().String(), address, amount)
}
