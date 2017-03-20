package alphapoint

import (
	"log"

	"github.com/thraser-/gocryptotrader/exchanges/ticker"
)

func (a *Alphapoint) GetTickerPrice(currency string) TickerPrice {
	var tickerPrice TickerPrice
	ticker, err := a.GetTicker(currency)
	if err != nil {
		log.Println(err)
		return TickerPrice{}
	}
	tickerPrice.Ask = ticker.Ask
	tickerPrice.Bid = ticker.Bid

	return tickerPrice
}
