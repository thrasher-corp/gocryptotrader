package lakebtc

import (
	"errors"
	"log"
	"strconv"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the LakeBTC go routine
func (l *LakeBTC) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
}

// Run implements the LakeBTC wrapper
func (l *LakeBTC) Run() {
	if l.Verbose {
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	exchangeProducts, err := l.GetTradablePairs()
	if err != nil {
		log.Printf("%s Failed to get available products.\n", l.GetName())
	} else {
		err = l.UpdateCurrencies(exchangeProducts, false, false)
		if err != nil {
			log.Printf("%s Failed to update available currencies.\n", l.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LakeBTC) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := l.GetTicker()
	if err != nil {
		return ticker.Price{}, err
	}

	for _, x := range l.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(l.Name, x).String()
		var tickerPrice ticker.Price
		tickerPrice.Pair = x
		tickerPrice.Ask = tick[currency].Ask
		tickerPrice.Bid = tick[currency].Bid
		tickerPrice.Volume = tick[currency].Volume
		tickerPrice.High = tick[currency].High
		tickerPrice.Low = tick[currency].Low
		tickerPrice.Last = tick[currency].Last
		ticker.ProcessTicker(l.GetName(), x, tickerPrice, assetType)
	}
	return ticker.GetTicker(l.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (l *LakeBTC) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (l *LakeBTC) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *LakeBTC) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := l.GetOrderBook(p.Pair().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderbook.ProcessOrderbook(l.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(l.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// LakeBTC exchange
func (l *LakeBTC) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = l.GetName()
	accountInfo, err := l.GetAccountInfo()
	if err != nil {
		return response, err
	}

	for x, y := range accountInfo.Balance {
		for z, w := range accountInfo.Locked {
			if z == x {
				var exchangeCurrency exchange.AccountCurrencyInfo
				exchangeCurrency.CurrencyName = common.StringToUpper(x)
				exchangeCurrency.TotalValue, _ = strconv.ParseFloat(y, 64)
				exchangeCurrency.Hold, _ = strconv.ParseFloat(w, 64)
				response.Currencies = append(response.Currencies, exchangeCurrency)
			}
		}
	}
	return response, nil
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (l *LakeBTC) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *LakeBTC) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (l *LakeBTC) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *LakeBTC) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (l *LakeBTC) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (l *LakeBTC) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (l *LakeBTC) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (l *LakeBTC) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *LakeBTC) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (l *LakeBTC) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (l *LakeBTC) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
