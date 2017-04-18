package poloniex

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (p *Poloniex) Start() {
	go p.Run()
}

func (p *Poloniex) Run() {
	if p.Verbose {
		log.Printf("%s Websocket: %s (url: %s).\n", p.GetName(), common.IsEnabled(p.Websocket), POLONIEX_WEBSOCKET_ADDRESS)
		log.Printf("%s polling delay: %ds.\n", p.GetName(), p.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", p.GetName(), len(p.EnabledPairs), p.EnabledPairs)
	}

	if p.Websocket {
		go p.WebsocketClient()
	}

	for p.Enabled {
		for _, x := range p.EnabledPairs {
			currency := pair.NewCurrencyPairDelimiter(x, "_")
			go func() {
				ticker, err := p.GetTickerPrice(currency)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("Poloniex %s Last %f High %f Low %f Volume %f\n", currency.Pair().String(), ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				stats.AddExchangeInfo(p.GetName(), currency.GetFirstCurrency().String(), currency.GetSecondCurrency().String(), ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * p.RESTPollingDelay)
	}
}

func (p *Poloniex) GetTickerPrice(currencyPair pair.CurrencyPair) (ticker.TickerPrice, error) {
	currency := currencyPair.Pair().String()
	tickerNew, err := ticker.GetTicker(p.GetName(), currencyPair)
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := p.GetTicker()
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = currencyPair
	tickerPrice.Ask = tick[currency].Last
	tickerPrice.Bid = tick[currency].HighestBid
	tickerPrice.High = tick[currency].HighestBid
	tickerPrice.Last = tick[currency].Last
	tickerPrice.Low = tick[currency].LowestAsk
	tickerPrice.Volume = tick[currency].BaseVolume
	ticker.ProcessTicker(p.GetName(), currencyPair, tickerPrice)
	return tickerPrice, nil
}

func (p *Poloniex) GetOrderbookEx(currencyPair pair.CurrencyPair) (orderbook.OrderbookBase, error) {
	currency := currencyPair.Pair().String()
	ob, err := orderbook.GetOrderbook(p.GetName(), currencyPair)
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := p.GetOrderbook(currency, 1000)
	if err != nil {
		return orderBook, err
	}

	for x, _ := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Amount: data.Amount, Price: data.Price})
	}

	for x, _ := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Amount: data.Amount, Price: data.Price})
	}
	orderBook.Pair = currencyPair
	orderbook.ProcessOrderbook(p.GetName(), currencyPair, orderBook)
	return orderBook, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Poloniex exchange
func (e *Poloniex) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetBalances()
	if err != nil {
		return response, err
	}

	for x, y := range accountBalance.Currency {
		var exchangeCurrency exchange.ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = x
		exchangeCurrency.TotalValue = y
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}
