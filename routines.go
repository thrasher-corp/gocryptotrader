package main

import (
	"fmt"
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
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

	stats.Add(exchangeName, p, assetType, result.Last, result.Volume)
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

func printOrderbookSummary(result orderbook.Base, p pair.CurrencyPair, assetType, exchangeName string, err error) {
	if err != nil {
		log.Printf("failed to get %s %s orderbook. Error: %s",
			p.Pair().String(),
			exchangeName,
			err)
		return
	}

	bidsAmount, bidsValue := result.CalculateTotalBids()
	asksAmount, asksValue := result.CalculateTotalAsks()

	log.Printf("%s %s %s: Orderbook Bids len: %d amount: %f total value: %f Asks len: %d amount: %f total value: %f",
		exchangeName,
		exchange.FormatCurrency(p).String(),
		assetType,
		len(result.Bids),
		bidsAmount,
		bidsValue,
		len(result.Asks),
		asksAmount,
		asksValue,
	)
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

				assetTypes, err = exchange.GetExchangeAssetTypes(exchangeName)
				if err != nil {
					log.Printf("failed to get %s exchange asset types. Error: %s",
						exchangeName, err)
				}

				for y := range enabledCurrencies {
					currency := enabledCurrencies[y]

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
				if bot.exchanges[x].GetName() == "ANX" {
					continue
				}

				exchangeName := bot.exchanges[x].GetName()
				enabledCurrencies := bot.exchanges[x].GetEnabledCurrencies()
				var result orderbook.Base
				var err error
				var assetTypes []string

				assetTypes, err = exchange.GetExchangeAssetTypes(exchangeName)
				if err != nil {
					log.Printf("failed to get %s exchange asset types. Error: %s",
						exchangeName, err)
				}

				for y := range enabledCurrencies {
					currency := enabledCurrencies[y]

					if len(assetTypes) > 1 {
						for z := range assetTypes {
							result, err = bot.exchanges[x].UpdateOrderbook(currency,
								assetTypes[z])
							printOrderbookSummary(result, currency, assetTypes[z], exchangeName, err)
							if err == nil {
								relayWebsocketEvent(result, "orderbook_update", assetTypes[z], exchangeName)
							}
						}
					} else {
						result, err = bot.exchanges[x].UpdateOrderbook(currency,
							assetTypes[0])
						printOrderbookSummary(result, currency, assetTypes[0], exchangeName, err)
						if err == nil {
							relayWebsocketEvent(result, "orderbook_update", assetTypes[0], exchangeName)
						}
					}
				}
			}
		}
		time.Sleep(time.Second * 10)
	}
}
