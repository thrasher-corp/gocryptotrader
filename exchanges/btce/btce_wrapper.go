package btce

import (
	"errors"
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (b *BTCE) Start() {
	go b.Run()
}

func (b *BTCE) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	pairs := []string{}
	for _, x := range b.EnabledPairs {
		x = common.StringToLower(x[0:3] + "_" + x[3:6])
		pairs = append(pairs, x)
	}
	pairsString := common.JoinStrings(pairs, "-")

	for b.Enabled {
		go func() {
			ticker, err := b.GetTicker(pairsString)
			if err != nil {
				log.Println(err)
				return
			}
			for x, y := range ticker {
				x = common.StringToUpper(x[0:3] + x[4:])
				log.Printf("BTC-e %s: Last %f High %f Low %f Volume %f\n", x, y.Last, y.High, y.Low, y.Vol_cur)
				b.Ticker[x] = y
				//AddExchangeInfo(b.GetName(), common.StringToUpper(x[0:3]), common.StringToUpper(x[4:]), y.Last, y.Vol_cur)
			}
		}()
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *BTCE) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	var tickerPrice ticker.TickerPrice
	tick, ok := b.Ticker[currency]
	if !ok {
		return tickerPrice, errors.New("Unable to get currency.")
	}
	tickerPrice.Ask = tick.Buy
	tickerPrice.Bid = tick.Sell
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Vol_cur
	tickerPrice.High = tick.High
	ticker.ProcessTicker(b.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the BTCE exchange
func (e *BTCE) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetAccountInfo()
	if err != nil {
		return response, err
	}

	for x, y := range accountBalance.Funds {
		var exchangeCurrency exchange.ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = common.StringToUpper(x)
		exchangeCurrency.TotalValue = y
		exchangeCurrency.Hold = 0
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}

	return response, nil
}
