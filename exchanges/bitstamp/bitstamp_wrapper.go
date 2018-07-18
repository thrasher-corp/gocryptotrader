package bitstamp

import (
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the Bitstamp go routine
func (b *Bitstamp) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitstamp wrapper
func (b *Bitstamp) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		go b.PusherClient()
	}

	pairs, err := b.GetTradingPairs()
	if err != nil {
		log.Printf("%s failed to get trading pairs. Err: %s", b.Name, err)
	} else {
		var currencies []string
		for x := range pairs {
			if pairs[x].Trading != "Enabled" {
				continue
			}
			pair := strings.Split(pairs[x].Name, "/")
			currencies = append(currencies, pair[0]+pair[1])
		}
		err = b.UpdateCurrencies(currencies, false, false)
		if err != nil {
			log.Printf("%s Failed to update available currencies.\n", b.Name)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitstamp) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(p.Pair().String(), false)
	if err != nil {
		return tickerPrice, err

	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume
	tickerPrice.High = tick.High
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Bitstamp) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *Bitstamp) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitstamp) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(p.Pair().String())
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

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Bitstamp exchange
func (b *Bitstamp) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()
	accountBalance, err := b.GetBalance()
	if err != nil {
		return response, err
	}

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "BTC",
		TotalValue:   accountBalance.BTCAvailable,
		Hold:         accountBalance.BTCReserved,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "XRP",
		TotalValue:   accountBalance.XRPAvailable,
		Hold:         accountBalance.XRPReserved,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "USD",
		TotalValue:   accountBalance.USDAvailable,
		Hold:         accountBalance.USDReserved,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "EUR",
		TotalValue:   accountBalance.EURAvailable,
		Hold:         accountBalance.EURReserved,
	})
	return response, nil
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (b *Bitstamp) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitstamp) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (b *Bitstamp) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitstamp) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (b *Bitstamp) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (b *Bitstamp) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (b *Bitstamp) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (b *Bitstamp) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitstamp) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitstamp) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitstamp) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
