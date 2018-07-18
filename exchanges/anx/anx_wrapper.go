package anx

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

// Start starts the ANX go routine
func (a *ANX) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		a.Run()
		wg.Done()
	}()
}

// Run implements the ANX wrapper
func (a *ANX) Run() {
	if a.Verbose {
		log.Printf("%s polling delay: %ds.\n", a.GetName(), a.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", a.GetName(), len(a.EnabledPairs), a.EnabledPairs)
	}

	exchangeProducts, err := a.GetTradablePairs()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", a.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(a.EnabledPairs, "_") || !common.StringDataContains(a.AvailablePairs, "_") {
			forceUpgrade = true
		}

		if forceUpgrade {
			enabledPairs := []string{"BTC_USD,BTC_HKD,BTC_EUR,BTC_CAD,BTC_AUD,BTC_SGD,BTC_JPY,BTC_GBP,BTC_NZD,LTC_BTC,DOG_EBTC,STR_BTC,XRP_BTC"}
			log.Println("WARNING: Enabled pairs for ANX reset due to config upgrade, please enable the ones you would like again.")

			err = a.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Printf("%s Failed to get config.\n", a.GetName())
			}
		}
		err = a.UpdateCurrencies(exchangeProducts, false, forceUpgrade)
		if err != nil {
			log.Printf("%s Failed to get config.\n", a.GetName())
		}
	}
}

// GetTradablePairs returns a list of available
func (a *ANX) GetTradablePairs() ([]string, error) {
	result, err := a.GetCurrencies()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range result.CurrencyPairs {
		currencies = append(currencies, result.CurrencyPairs[x].TradedCcy+"_"+result.CurrencyPairs[x].SettlementCcy)
	}

	return currencies, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (a *ANX) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := a.GetTicker(exchange.FormatExchangeCurrency(a.GetName(), p).String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p

	if tick.Data.Sell.Value != "" {
		tickerPrice.Ask, err = strconv.ParseFloat(tick.Data.Sell.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Ask = 0
	}

	if tick.Data.Buy.Value != "" {
		tickerPrice.Bid, err = strconv.ParseFloat(tick.Data.Buy.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Bid = 0
	}

	if tick.Data.Low.Value != "" {
		tickerPrice.Low, err = strconv.ParseFloat(tick.Data.Low.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Low = 0
	}

	if tick.Data.Last.Value != "" {
		tickerPrice.Last, err = strconv.ParseFloat(tick.Data.Last.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Last = 0
	}

	if tick.Data.Vol.Value != "" {
		tickerPrice.Volume, err = strconv.ParseFloat(tick.Data.Vol.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Volume = 0
	}

	if tick.Data.High.Value != "" {
		tickerPrice.High, err = strconv.ParseFloat(tick.Data.High.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.High = 0
	}
	ticker.ProcessTicker(a.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(a.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (a *ANX) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (a *ANX) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *ANX) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := a.GetDepth(exchange.FormatExchangeCurrency(a.GetName(), p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Data.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: orderbookNew.Data.Asks[x].Price, Amount: orderbookNew.Data.Asks[x].Amount})
	}

	for x := range orderbookNew.Data.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: orderbookNew.Data.Bids[x].Price, Amount: orderbookNew.Data.Bids[x].Amount})
	}

	orderbook.ProcessOrderbook(a.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(a.Name, p, assetType)
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the ANX exchange
func (a *ANX) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = a.GetName()
	return response, nil
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (a *ANX) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (a *ANX) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (a *ANX) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (a *ANX) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (a *ANX) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (a *ANX) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (a *ANX) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (a *ANX) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
