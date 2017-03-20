package poloniex

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
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
			currency := x
			go func() {
				ticker, err := p.GetTickerPrice(currency)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("Poloniex %s Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				//currencyPair := common.SplitStrings(currency, "_")
				//AddExchangeInfo(p.GetName(), currencyPair[0], currencyPair[1], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * p.RESTPollingDelay)
	}
}

func (p *Poloniex) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(p.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := p.GetTicker()
	if err != nil {
		return tickerPrice, err
	}

	currencyPair := common.SplitStrings(currency, "_")
	tickerPrice.FirstCurrency = currencyPair[0]
	tickerPrice.SecondCurrency = currencyPair[1]
	tickerPrice.Ask = tick[currency].Last
	tickerPrice.Bid = tick[currency].HighestBid
	tickerPrice.High = tick[currency].HighestBid
	tickerPrice.Last = tick[currency].Last
	tickerPrice.Low = tick[currency].LowestAsk
	tickerPrice.Volume = tick[currency].BaseVolume
	ticker.ProcessTicker(p.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Poloniex exchange
func (e *Poloniex) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetBalances()
	if err != nil {
		return response, err
	}
	currencies := e.AvailablePairs
	for i := 0; i < len(currencies); i++ {
		var exchangeCurrency exchange.ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = currencies[i]
		exchangeCurrency.TotalValue = accountBalance.Currency[currencies[i]]
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}
