package bitstamp

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (b *Bitstamp) Start() {
	go b.Run()
}

func (b *Bitstamp) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		go b.PusherClient()
	}

	for b.Enabled {
		for _, x := range b.EnabledPairs {
			currency := x
			go func() {
				ticker, err := b.GetTickerPrice(currency)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("Bitstamp %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				//AddExchangeInfo(b.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *Bitstamp) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := b.GetTicker(currency, false)
	if err != nil {
		return tickerPrice, err

	}
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume
	tickerPrice.High = tick.High
	ticker.ProcessTicker(b.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Bitstamp exchange
func (e *Bitstamp) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetBalance()
	if err != nil {
		return response, err
	}

	var btcExchangeInfo exchange.ExchangeAccountCurrencyInfo
	btcExchangeInfo.CurrencyName = "BTC"
	btcExchangeInfo.TotalValue = accountBalance.BTCBalance
	btcExchangeInfo.Hold = accountBalance.BTCReserved
	response.Currencies = append(response.Currencies, btcExchangeInfo)

	var usdExchangeInfo exchange.ExchangeAccountCurrencyInfo
	usdExchangeInfo.CurrencyName = "USD"
	usdExchangeInfo.TotalValue = accountBalance.USDBalance
	usdExchangeInfo.Hold = accountBalance.USDReserved
	response.Currencies = append(response.Currencies, usdExchangeInfo)

	return response, nil
}
