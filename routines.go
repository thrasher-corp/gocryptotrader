package main

import (
	"fmt"
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func printSummary(result ticker.Price, p pair.CurrencyPair, assetType, exchangeName string, err error) {
	if err != nil {
		log.Printf("failed to get %s %s ticker. Error: %s",
			p.Pair().String(),
			exchangeName,
			err)
		return
	}

	log.Printf("%s %s %s: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f",
		exchangeName,
		exchange.FormatCurrency(p).String(),
		assetType,
		result.Last,
		result.Ask,
		result.Bid,
		result.High,
		result.Low,
		result.Volume)
}

func relayWebsocketEvent(result interface{}, event, assetType, exchangeName string) {
	evt := WebsocketEvent{
		Data:      result,
		Event:     event,
		AssetType: assetType,
		Exchange:  exchangeName,
	}
	err := BroadcastWebsocketMessage(evt)
	if err != nil {
		log.Println(fmt.Errorf("Failed to broadcast websocket event. Error: %s",
			err))
	}
}

func TickerUpdaterRoutine() {
	log.Println("Starting ticker updater routine")
	for {
		for x := range bot.exchanges {
			if bot.exchanges[x].IsEnabled() {
				exchangeName := bot.exchanges[x].GetName()
				enabledCurrencies := bot.exchanges[x].GetEnabledCurrencies()

				var result ticker.Price
				var err error
				var assetTypes []string

				for y := range enabledCurrencies {
					currency := enabledCurrencies[y]
					assetTypes, err = exchange.GetExchangeAssetTypes(exchangeName)
					if err != nil {
						log.Printf("failed to get %s exchange asset types. Error: %s",
							exchangeName, err)
					}
					if len(assetTypes) > 1 {
						for z := range assetTypes {
							result, err = bot.exchanges[x].UpdateTicker(currency,
								assetTypes[z])
							printSummary(result, currency, assetTypes[z], exchangeName, err)
							if err == nil {
								relayWebsocketEvent(result, "ticker_update", assetTypes[z], exchangeName)
							}
						}
					} else {
						result, err = bot.exchanges[x].UpdateTicker(currency,
							assetTypes[0])
						printSummary(result, currency, assetTypes[0], exchangeName, err)
						if err == nil {
							relayWebsocketEvent(result, "ticker_update", assetTypes[0], exchangeName)
						}
					}
				}
			}
		}
		time.Sleep(time.Second * 10)
	}
}

func OrderbookUpdaterRoutine() {
	log.Println("Starting orderbook updater routine")
	for {
		for x := range bot.exchanges {
			if bot.exchanges[x].IsEnabled() {
				exchangeName := bot.exchanges[x].GetName()

				if exchangeName == "ANX" {
					continue
				}

				enabledCurrencies := bot.exchanges[x].GetEnabledCurrencies()

				for y := range enabledCurrencies {
					currency := enabledCurrencies[y]
					result, err := bot.exchanges[x].UpdateOrderbook(currency)
					if err != nil {
						log.Printf("failed to get %s orderbook", currency.Pair().String())
						continue
					}

					log.Printf("%s %s %v",
						exchangeName,
						exchange.FormatCurrency(currency).String(),
						result)

					evt := WebsocketEvent{
						Data:     result,
						Event:    "orderbook_update",
						Exchange: exchangeName,
					}
					BroadcastWebsocketMessage(evt)
				}
			}
		}
		time.Sleep(time.Second * 10)
	}
}
