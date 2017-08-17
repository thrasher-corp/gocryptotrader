package bitfinex

import (
	"log"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (b *Bitfinex) Start() {
	go b.Run()
}

func (b *Bitfinex) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		go b.WebsocketClient()
	}

	exchangeProducts, err := b.GetSymbols()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", b.GetName())
	} else {
		err = b.UpdateAvailableCurrencies(exchangeProducts)
		if err != nil {
			log.Printf("%s Failed to get config.\n", b.GetName())
		}
	}

	for b.Enabled {
		for _, x := range b.EnabledPairs {
			currency := pair.NewCurrencyPair(x[0:3], x[3:])
			go func() {
				ticker, err := b.GetTickerPrice(currency)
				if err != nil {
					return
				}
				log.Printf("Bitfinex %s Last %f High %f Low %f Volume %f\n", currency.Pair().String(), ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				stats.AddExchangeInfo(b.GetName(), currency.GetFirstCurrency().String(), currency.GetSecondCurrency().String(), ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *Bitfinex) GetTickerPrice(p pair.CurrencyPair) (ticker.TickerPrice, error) {
	tick, err := ticker.GetTicker(b.GetName(), p)
	if err == nil {
		return tick, nil
	}

	var tickerPrice ticker.TickerPrice
	tickerNew, err := b.GetTicker(p.Pair().String(), nil)
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tickerNew.Ask
	tickerPrice.Bid = tickerNew.Bid
	tickerPrice.Low = tickerNew.Low
	tickerPrice.Last = tickerNew.Last
	tickerPrice.Volume = tickerNew.Volume
	tickerPrice.High = tickerNew.High
	ticker.ProcessTicker(b.GetName(), p, tickerPrice)
	return tickerPrice, nil
}

func (b *Bitfinex) GetOrderbookEx(p pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p)
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := b.GetOrderbook(p.Pair().String(), nil)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Asks {
		price, _ := strconv.ParseFloat(orderbookNew.Asks[x].Price, 64)
		amount, _ := strconv.ParseFloat(orderbookNew.Asks[x].Amount, 64)
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Price: price, Amount: amount})
	}

	for x := range orderbookNew.Bids {
		price, _ := strconv.ParseFloat(orderbookNew.Bids[x].Price, 64)
		amount, _ := strconv.ParseFloat(orderbookNew.Bids[x].Amount, 64)
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Price: price, Amount: amount})
	}

	orderBook.Pair = p
	orderbook.ProcessOrderbook(b.GetName(), p, orderBook)
	return orderBook, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Bitfinex exchange
func (b *Bitfinex) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()
	accountBalance, err := b.GetAccountBalance()
	if err != nil {
		return response, err
	}
	if !b.Enabled {
		return response, nil
	}

	type bfxCoins struct {
		OnHold    float64
		Available float64
	}

	accounts := make(map[string]bfxCoins)

	for i := range accountBalance {
		onHold := accountBalance[i].Amount - accountBalance[i].Available
		coins := bfxCoins{
			OnHold:    onHold,
			Available: accountBalance[i].Available,
		}
		result, ok := accounts[accountBalance[i].Currency]
		if !ok {
			accounts[accountBalance[i].Currency] = coins
		} else {
			result.Available += accountBalance[i].Available
			result.OnHold += onHold
			accounts[accountBalance[i].Currency] = result
		}
	}

	for x, y := range accounts {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = common.StringToUpper(x)
		exchangeCurrency.TotalValue = y.Available + y.OnHold
		exchangeCurrency.Hold = y.OnHold
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}

	return response, nil
}
