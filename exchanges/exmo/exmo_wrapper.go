package exmo

import (
	"errors"
	"log"
	"strconv"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the EXMO go routine
func (e *EXMO) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		e.Run()
		wg.Done()
	}()
}

// Run implements the EXMO wrapper
func (e *EXMO) Run() {
	if e.Verbose {
		log.Printf("%s polling delay: %ds.\n", e.GetName(), e.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", e.GetName(), len(e.EnabledPairs), e.EnabledPairs)
	}

	exchangeProducts, err := e.GetPairSettings()
	if err != nil {
		log.Printf("%s Failed to get available products.\n", e.GetName())
	} else {
		var currencies []string
		for x := range exchangeProducts {
			currencies = append(currencies, x)
		}
		err = e.UpdateCurrencies(currencies, false, false)
		if err != nil {
			log.Printf("%s Failed to update available currencies.\n", e.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *EXMO) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(e.Name, e.GetEnabledCurrencies())
	if err != nil {
		return tickerPrice, err
	}

	result, err := e.GetTicker(pairsCollated.String())
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range e.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(e.Name, x).String()
		var tickerPrice ticker.Price
		tickerPrice.Pair = x
		tickerPrice.Last = result[currency].Last
		tickerPrice.Ask = result[currency].Sell
		tickerPrice.High = result[currency].High
		tickerPrice.Bid = result[currency].Buy
		tickerPrice.Last = result[currency].Last
		tickerPrice.Low = result[currency].Low
		tickerPrice.Volume = result[currency].Volume
		ticker.ProcessTicker(e.Name, x, tickerPrice, assetType)
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (e *EXMO) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(e.GetName(), p, assetType)
	if err != nil {
		return e.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (e *EXMO) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(e.GetName(), p, assetType)
	if err != nil {
		return e.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *EXMO) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(e.Name, e.GetEnabledCurrencies())
	if err != nil {
		return orderBook, err
	}

	result, err := e.GetOrderbook(pairsCollated.String())
	if err != nil {
		return orderBook, err
	}

	for _, x := range e.GetEnabledCurrencies() {
		currency := exchange.FormatExchangeCurrency(e.Name, x)
		data, ok := result[currency.String()]
		if !ok {
			continue
		}
		orderBook.Pair = x

		var obItems []orderbook.Item
		for y := range data.Ask {
			z := data.Ask[y]
			price, _ := strconv.ParseFloat(z[0], 64)
			amount, _ := strconv.ParseFloat(z[1], 64)
			obItems = append(obItems, orderbook.Item{Price: price, Amount: amount})
		}

		orderBook.Asks = obItems
		obItems = []orderbook.Item{}
		for y := range data.Bid {
			z := data.Bid[y]
			price, _ := strconv.ParseFloat(z[0], 64)
			amount, _ := strconv.ParseFloat(z[1], 64)
			obItems = append(obItems, orderbook.Item{Price: price, Amount: amount})
		}

		orderBook.Bids = obItems
		orderbook.ProcessOrderbook(e.Name, x, orderBook, assetType)
	}
	return orderbook.GetOrderbook(e.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Exmo exchange
func (e *EXMO) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = e.GetName()
	result, err := e.GetUserInfo()
	if err != nil {
		return response, err
	}

	for x, y := range result.Balances {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = common.StringToUpper(x)
		for z, w := range result.Reserved {
			if z == x {
				avail, _ := strconv.ParseFloat(y, 64)
				reserved, _ := strconv.ParseFloat(w, 64)
				exchangeCurrency.TotalValue = avail + reserved
				exchangeCurrency.Hold = reserved
			}
		}
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (e *EXMO) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (e *EXMO) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (e *EXMO) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (e *EXMO) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (e *EXMO) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (e *EXMO) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (e *EXMO) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (e *EXMO) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *EXMO) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *EXMO) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *EXMO) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
