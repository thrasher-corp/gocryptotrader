package okcoin

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (o *OKCoin) Start() {
	go o.Run()
}

func (o *OKCoin) Run() {
	if o.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket), o.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", o.GetName(), o.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", o.GetName(), len(o.EnabledPairs), o.EnabledPairs)
	}

	if o.Websocket {
		go o.WebsocketClient()
	}

	for o.Enabled {
		for _, x := range o.EnabledPairs {
			currency := common.StringToLower(x[0:3] + "_" + x[3:])
			if o.APIUrl == OKCOIN_API_URL {
				for _, y := range o.FuturesValues {
					futuresValue := y
					go func() {
						ticker, err := o.GetFuturesTicker(currency, futuresValue)
						if err != nil {
							log.Println(err)
							return
						}
						log.Printf("OKCoin Intl Futures %s (%s): Last %f High %f Low %f Volume %f\n", currency, futuresValue, ticker.Last, ticker.High, ticker.Low, ticker.Vol)
						//AddExchangeInfo(o.GetName(), common.StringToUpper(currency[0:3]), common.StringToUpper(currency[4:]), ticker.Last, ticker.Vol)
					}()
				}
				go func() {
					ticker, err := o.GetTickerPrice(currency)
					if err != nil {
						log.Println(err)
						return
					}
					log.Printf("OKCoin Intl Spot %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
					//AddExchangeInfo(o.GetName(), common.StringToUpper(currency[0:3]), common.StringToUpper(currency[4:]), ticker.Last, ticker.Volume)
				}()
			} else {
				go func() {
					ticker, err := o.GetTickerPrice(currency)
					if err != nil {
						log.Println(err)
						return
					}
					//tickerLastUSD, _ := ConvertCurrency(ticker.Last, "CNY", "USD")
					//tickerHighUSD, _ := ConvertCurrency(ticker.High, "CNY", "USD")
					//tickerLowUSD, _ := ConvertCurrency(ticker.Low, "CNY", "USD")
					//log.Printf("OKCoin China %s: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", currency, tickerLastUSD, ticker.Last, tickerHighUSD, ticker.High, tickerLowUSD, ticker.Low, ticker.Volume)
					log.Printf("OKCoin China %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
					//AddExchangeInfo(o.GetName(), common.StringToUpper(currency[0:3]), common.StringToUpper(currency[4:]), ticker.Last, ticker.Volume)
					//AddExchangeInfo(o.GetName(), common.StringToUpper(currency[0:3]), "USD", tickerLastUSD, ticker.Volume)
				}()
			}
		}
		time.Sleep(time.Second * o.RESTPollingDelay)
	}
}

func (o *OKCoin) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(o.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := o.GetTicker(currency)
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Ask = tick.Sell
	tickerPrice.Bid = tick.Buy
	tickerPrice.FirstCurrency = common.StringToUpper(currency[0:3])
	tickerPrice.SecondCurrency = common.StringToUpper(currency[4:])
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Vol
	tickerPrice.High = tick.High
	ticker.ProcessTicker(o.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

//TODO support for retrieving holdings from OKCOIN
//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the OKCoin exchange
func (e *OKCoin) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}
