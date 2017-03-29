package alphapoint

import (
	"log"

	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Alphapoint exchange
func (e *Alphapoint) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	account, err := e.GetAccountInfo()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(account.Currencies); i++ {
		var exchangeCurrency exchange.ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = account.Currencies[i].Name
		exchangeCurrency.TotalValue = float64(account.Currencies[i].Balance)
		exchangeCurrency.Hold = float64(account.Currencies[i].Hold)

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	//If it all works out
	return response, nil
}

func (a *Alphapoint) GetTickerPrice(currency string) ticker.TickerPrice {
	var tickerPrice ticker.TickerPrice
	tick, err := a.GetTicker(currency)
	if err != nil {
		log.Println(err)
		return ticker.TickerPrice{}
	}
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid

	return tickerPrice
}
