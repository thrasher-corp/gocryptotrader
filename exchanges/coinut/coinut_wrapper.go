package coinut

import (
	"errors"
	"log"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the COINUT go routine
func (c *COINUT) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
}

// Run implements the COINUT wrapper
func (c *COINUT) Run() {
	if c.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", c.GetName(), common.IsEnabled(c.Websocket), coinutWebsocketURL)
		log.Printf("%s polling delay: %ds.\n", c.GetName(), c.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", c.GetName(), len(c.EnabledPairs), c.EnabledPairs)
	}

	if c.Websocket {
		go c.WebsocketClient()
	}

	exchangeProducts, err := c.GetInstruments()
	if err != nil {
		log.Printf("%s Failed to get available products.\n", c.GetName())
		return
	}

	currencies := []string{}
	c.InstrumentMap = make(map[string]int)
	for x, y := range exchangeProducts.Instruments {
		c.InstrumentMap[x] = y[0].InstID
		currencies = append(currencies, x)
	}

	err = c.UpdateCurrencies(currencies, false, false)
	if err != nil {
		log.Printf("%s Failed to update available currencies.\n", c.GetName())
	}
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// COINUT exchange
func (c *COINUT) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	/*
		response.ExchangeName = e.GetName()
		accountBalance, err := e.GetAccounts()
		if err != nil {
			return response, err
		}
		for i := 0; i < len(accountBalance); i++ {
			var exchangeCurrency exchange.AccountCurrencyInfo
			exchangeCurrency.CurrencyName = accountBalance[i].Currency
			exchangeCurrency.TotalValue = accountBalance[i].Available
			exchangeCurrency.Hold = accountBalance[i].Hold

			response.Currencies = append(response.Currencies, exchangeCurrency)
		}
	*/
	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *COINUT) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := c.GetInstrumentTicker(c.InstrumentMap[p.Pair().String()])
	if err != nil {
		return ticker.Price{}, err
	}

	tickerPrice.Pair = p
	tickerPrice.Volume = tick.Volume
	tickerPrice.Last = tick.Last
	tickerPrice.High = tick.HighestBuy
	tickerPrice.Low = tick.LowestSell
	ticker.ProcessTicker(c.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(c.Name, p, assetType)

}

// GetTickerPrice returns the ticker for a currency pair
func (c *COINUT) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (c *COINUT) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(c.GetName(), p, assetType)
	if err != nil {
		return c.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *COINUT) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := c.GetInstrumentOrderbook(c.InstrumentMap[p.Pair().String()], 200)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Buy {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Buy[x].Quantity, Price: orderbookNew.Buy[x].Price})
	}

	for x := range orderbookNew.Sell {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Sell[x].Quantity, Price: orderbookNew.Sell[x].Price})
	}

	orderbook.ProcessOrderbook(c.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(c.Name, p, assetType)
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (c *COINUT) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (c *COINUT) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (c *COINUT) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *COINUT) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (c *COINUT) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (c *COINUT) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (c *COINUT) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (c *COINUT) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *COINUT) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (c *COINUT) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *COINUT) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
