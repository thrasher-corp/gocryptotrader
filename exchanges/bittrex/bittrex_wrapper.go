package bittrex

import (
	"log"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start stats the Bittrex go routine
func (b *Bittrex) Start() {
	go b.Run()
}

// Run implements the Bittrex wrapper
func (b *Bittrex) Run() {
	if b.Verbose {
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	exchangeProducts, err := b.GetMarkets()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", b.GetName())
	} else {
		var currencies []string
		for x := range exchangeProducts {
			if !exchangeProducts[x].IsActive {
				continue
			}
			currencies = append(currencies,
				strings.Replace(exchangeProducts[x].MarketName, "-", "", -1))
		}
		err = b.UpdateAvailableCurrencies(currencies)
		if err != nil {
			log.Printf("%s Failed to get config.\n", b.GetName())
		}
	}

	for b.Enabled {
		for _, x := range b.EnabledPairs {
			currency := pair.NewCurrencyPair(x[0:3], x[3:])
			currency.Delimiter = "-"
			go func() {
				ticker, err := b.GetTickerPrice(currency)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("Bittrex %s Last %f Bid %f Ask %f Volume %f\n", currency.Pair().String(), ticker.Last, ticker.Bid, ticker.Ask, ticker.Volume)
				stats.AddExchangeInfo(b.GetName(), currency.GetFirstCurrency().String(), currency.GetSecondCurrency().String(), ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

//GetExchangeAccountInfo Retrieves balances for all enabled currencies for the Bittrexexchange
func (b *Bittrex) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()
	accountBalance, err := b.GetAccountBalances()
	if err != nil {
		return response, err
	}

	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Balance
		exchangeCurrency.Hold = accountBalance[i].Balance - accountBalance[i].Available
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

// GetTickerPrice returns the ticker for a currencyp pair
func (b *Bittrex) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p)
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := b.GetMarketSummary(p.Pair().Lower().String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick[0].Ask
	tickerPrice.Bid = tick[0].Bid
	tickerPrice.Last = tick[0].Last
	tickerPrice.Volume = tick[0].Volume
	ticker.ProcessTicker(b.GetName(), p, tickerPrice)
	return tickerPrice, nil
}

// GetOrderbookEx returns the orderbook for a currencyp pair
func (b *Bittrex) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p)
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := b.GetOrderbook(p.Pair().Lower().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Buy {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.OrderbookItem{
				Amount: orderbookNew.Buy[x].Quantity,
				Price:  orderbookNew.Buy[x].Rate,
			},
		)
	}

	for x := range orderbookNew.Sell {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.OrderbookItem{
				Amount: orderbookNew.Sell[x].Quantity,
				Price:  orderbookNew.Sell[x].Rate,
			},
		)
	}

	orderBook.Pair = p
	orderbook.ProcessOrderbook(b.GetName(), p, orderBook)
	return orderBook, nil
}
