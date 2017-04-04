package main

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

func TickerUpdaterRoutine() {
	log.Println("Starting ticker updater routine")
	for {
		for x := range bot.exchanges {
			if bot.exchanges[x].IsEnabled() {
				exchangeName := bot.exchanges[x].GetName()
				enabledCurrencies := bot.exchanges[x].GetEnabledCurrencies()

				for _, y := range enabledCurrencies {
					currency := pair.NewCurrencyPair(y[0:3], y[3:])
					result, err := bot.exchanges[x].UpdateTicker(currency)
					if err != nil {
						log.Printf("failed to get %s currency", currency.Pair().String())
						continue
					}

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
