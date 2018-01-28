package main

import (
	"fmt"
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func printCurrencyFormat(price float64) string {
	displaySymbol, err := symbol.GetSymbolByCurrencyName(bot.config.FiatDisplayCurrency)
	if err != nil {
		log.Printf("Failed to get display symbol: %s", err)
	}

	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origCurrency string, origPrice float64) string {
	displayCurrency := bot.config.FiatDisplayCurrency
	conv, err := currency.ConvertCurrency(origPrice, origCurrency, displayCurrency)
	if err != nil {
		log.Printf("Failed to convert currency: %s", err)
	}

	displaySymbol, err := symbol.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Printf("Failed to get display symbol: %s", err)
	}

	origSymbol, err := symbol.GetSymbolByCurrencyName(origCurrency)
	if err != nil {
		log.Printf("Failed to get original currency symbol: %s", err)
	}

	return fmt.Sprintf("%s%.2f %s (%s%.2f %s)",
		displaySymbol,
		conv,
		displayCurrency,
		origSymbol,
		origPrice,
		origCurrency,
	)
}

func printSummary(result ticker.Price, p pair.CurrencyPair, assetType, exchangeName string, err error) {
	if err != nil {
		log.Printf("Failed to get %s %s ticker. Error: %s",
			p.Pair().String(),
			exchangeName,
			err)
		return
	}

	stats.Add(exchangeName, p, assetType, result.Last, result.Volume)
	if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.String() != bot.config.FiatDisplayCurrency {
		origCurrency := p.SecondCurrency.Upper().String()
		log.Printf("%s %s %s: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
			exchangeName,
			exchange.FormatCurrency(p).String(),
			assetType,
			printConvertCurrencyFormat(origCurrency, result.Last),
			printConvertCurrencyFormat(origCurrency, result.Ask),
			printConvertCurrencyFormat(origCurrency, result.Bid),
			printConvertCurrencyFormat(origCurrency, result.High),
			printConvertCurrencyFormat(origCurrency, result.Low),
			result.Volume)
	} else {
		if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.Upper().String() == bot.config.FiatDisplayCurrency {
			log.Printf("%s %s %s: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				printCurrencyFormat(result.Last),
				printCurrencyFormat(result.Ask),
				printCurrencyFormat(result.Bid),
				printCurrencyFormat(result.High),
				printCurrencyFormat(result.Low),
				result.Volume)
		} else {
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
	}
}

func printOrderbookSummary(result orderbook.Base, p pair.CurrencyPair, assetType, exchangeName string, err error) {
	if err != nil {
		log.Printf("Failed to get %s %s orderbook. Error: %s",
			p.Pair().String(),
			exchangeName,
			err)
		return
	}
	bidsAmount, bidsValue := result.CalculateTotalBids()
	asksAmount, asksValue := result.CalculateTotalAsks()

	if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.String() != bot.config.FiatDisplayCurrency {
		origCurrency := p.SecondCurrency.Upper().String()
		log.Printf("%s %s %s: Orderbook Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s",
			exchangeName,
			exchange.FormatCurrency(p).String(),
			assetType,
			len(result.Bids),
			bidsAmount,
			p.FirstCurrency.String(),
			printConvertCurrencyFormat(origCurrency, bidsValue),
			len(result.Asks),
			asksAmount,
			p.FirstCurrency.String(),
			printConvertCurrencyFormat(origCurrency, asksValue),
		)
	} else {
		if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.Upper().String() == bot.config.FiatDisplayCurrency {
			log.Printf("%s %s %s: Orderbook Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				len(result.Bids),
				bidsAmount,
				p.FirstCurrency.String(),
				printCurrencyFormat(bidsValue),
				len(result.Asks),
				asksAmount,
				p.FirstCurrency.String(),
				printCurrencyFormat(asksValue),
			)
		} else {
			log.Printf("%s %s %s: Orderbook Bids len: %d Amount: %f %s. Total value: %f Asks len: %d Amount: %f %s. Total value: %f",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				len(result.Bids),
				bidsAmount,
				p.FirstCurrency.String(),
				bidsValue,
				len(result.Asks),
				asksAmount,
				p.FirstCurrency.String(),
				asksValue,
			)
		}
	}

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

// TickerUpdaterRoutine fetches and updates the ticker for all enabled
// currency pairs and exchanges
func TickerUpdaterRoutine() {
	log.Println("Starting ticker updater routine")
	for {
		for x := range bot.exchanges {
			if bot.exchanges[x] == nil {
				continue
			}
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
		time.Sleep(time.Second * 10)
	}
}

// OrderbookUpdaterRoutine fetches and updates the orderbooks for all enabled
// currency pairs and exchanges
func OrderbookUpdaterRoutine() {
	log.Println("Starting orderbook updater routine")
	for {
		for x := range bot.exchanges {
			if bot.exchanges[x] == nil {
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
		time.Sleep(time.Second * 10)
	}
}
