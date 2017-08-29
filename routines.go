package main

import (
	"log"
	"time"

	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

func TickerUpdaterRoutine() {
	log.Println("Starting ticker updater routine")
	for {
		for x := range bot.exchanges {
			if bot.exchanges[x].IsEnabled() {
				exchangeName := bot.exchanges[x].GetName()
				enabledCurrencies := bot.exchanges[x].GetEnabledCurrencies()

				for y := range enabledCurrencies {
					currency := enabledCurrencies[y]
					result, err := bot.exchanges[x].UpdateTicker(currency)
					if err != nil {
						log.Printf("failed to get %s currency", currency.Pair().String())
						continue
					}

					log.Printf("%s %s: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f",
						exchangeName,
						exchange.FormatCurrency(currency).String(),
						result.Last,
						result.Ask,
						result.Bid,
						result.High,
						result.Low,
						result.Volume)

					evt := WebsocketEvent{
						Data:     result,
						Event:    "ticker_update",
						Exchange: exchangeName,
					}
					BroadcastWebsocketMessage(evt)
				}
			}
		}
		time.Sleep(time.Second * 10)
	}
}
