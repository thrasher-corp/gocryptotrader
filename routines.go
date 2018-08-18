package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	"github.com/thrasher-/gocryptotrader/decimal"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const defaultPrintDecimalPrecision = 8

func printCurrencyFormat(price decimal.Decimal) string {
	displaySymbol, err := symbol.GetSymbolByCurrencyName(bot.config.Currency.FiatDisplayCurrency)
	if err != nil {
		log.Printf("Failed to get display symbol: %s", err)
	}

	return fmt.Sprintf("%s%s", displaySymbol, price.StringFixed(defaultPrintDecimalPrecision))
}

func printConvertCurrencyFormat(origCurrency string, origPrice decimal.Decimal) string {
	displayCurrency := bot.config.Currency.FiatDisplayCurrency
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

	return fmt.Sprintf("%s%s %s (%s%s %s)",
		displaySymbol,
		conv.StringFixed(defaultPrintDecimalPrecision),
		displayCurrency,
		origSymbol,
		origPrice.StringFixed(defaultPrintDecimalPrecision),
		origCurrency,
	)
}

func printTickerSummary(result ticker.Price, p pair.CurrencyPair, assetType, exchangeName string, err error) {
	if err != nil {
		log.Printf("Failed to get %s %s ticker. Error: %s",
			p.Pair().String(),
			exchangeName,
			err)
		return
	}

	stats.Add(exchangeName, p, assetType, result.Last, result.Volume)
	if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.String() != bot.config.Currency.FiatDisplayCurrency {
		origCurrency := p.SecondCurrency.Upper().String()
		log.Printf("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %s",
			exchangeName,
			exchange.FormatCurrency(p).String(),
			assetType,
			printConvertCurrencyFormat(origCurrency, result.Last),
			printConvertCurrencyFormat(origCurrency, result.Ask),
			printConvertCurrencyFormat(origCurrency, result.Bid),
			printConvertCurrencyFormat(origCurrency, result.High),
			printConvertCurrencyFormat(origCurrency, result.Low),
			result.Volume.StringFixed(defaultPrintDecimalPrecision))
	} else {
		if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.Upper().String() == bot.config.Currency.FiatDisplayCurrency {
			log.Printf("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %s",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				printCurrencyFormat(result.Last),
				printCurrencyFormat(result.Ask),
				printCurrencyFormat(result.Bid),
				printCurrencyFormat(result.High),
				printCurrencyFormat(result.Low),
				result.Volume.StringFixed(defaultPrintDecimalPrecision))
		} else {
			log.Printf("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %s",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				result.Last.StringFixed(defaultPrintDecimalPrecision),
				result.Ask.StringFixed(defaultPrintDecimalPrecision),
				result.Bid.StringFixed(defaultPrintDecimalPrecision),
				result.High.StringFixed(defaultPrintDecimalPrecision),
				result.Low.StringFixed(defaultPrintDecimalPrecision),
				result.Volume.StringFixed(defaultPrintDecimalPrecision))
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

	if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.String() != bot.config.Currency.FiatDisplayCurrency {
		origCurrency := p.SecondCurrency.Upper().String()
		log.Printf("%s %s %s: ORDERBOOK: Bids len: %d Amount: %s %s. Total value: %s Asks len: %d Amount: %s %s. Total value: %s",
			exchangeName,
			exchange.FormatCurrency(p).String(),
			assetType,
			len(result.Bids),
			bidsAmount.StringFixed(defaultPrintDecimalPrecision),
			p.FirstCurrency.String(),
			printConvertCurrencyFormat(origCurrency, bidsValue),
			len(result.Asks),
			asksAmount.StringFixed(defaultPrintDecimalPrecision),
			p.FirstCurrency.String(),
			printConvertCurrencyFormat(origCurrency, asksValue),
		)
	} else {
		if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.Upper().String() == bot.config.Currency.FiatDisplayCurrency {
			log.Printf("%s %s %s: ORDERBOOK: Bids len: %d Amount: %s %s. Total value: %s Asks len: %d Amount: %s %s. Total value: %s",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				len(result.Bids),
				bidsAmount.StringFixed(defaultPrintDecimalPrecision),
				p.FirstCurrency.String(),
				printCurrencyFormat(bidsValue),
				len(result.Asks),
				asksAmount.StringFixed(defaultPrintDecimalPrecision),
				p.FirstCurrency.String(),
				printCurrencyFormat(asksValue),
			)
		} else {
			log.Printf("%s %s %s: ORDERBOOK: Bids len: %d Amount: %s %s. Total value: %s Asks len: %d Amount: %s %s. Total value: %s",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				len(result.Bids),
				bidsAmount.StringFixed(defaultPrintDecimalPrecision),
				p.FirstCurrency.String(),
				bidsValue.StringFixed(defaultPrintDecimalPrecision),
				len(result.Asks),
				asksAmount.StringFixed(defaultPrintDecimalPrecision),
				p.FirstCurrency.String(),
				asksValue.StringFixed(defaultPrintDecimalPrecision),
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
	log.Println("Starting ticker updater routine.")
	var wg sync.WaitGroup
	for {
		wg.Add(len(bot.exchanges))
		for x := range bot.exchanges {
			go func(x int, wg *sync.WaitGroup) {
				defer wg.Done()
				if bot.exchanges[x] == nil {
					return
				}
				exchangeName := bot.exchanges[x].GetName()
				enabledCurrencies := bot.exchanges[x].GetEnabledCurrencies()
				supportsBatching := bot.exchanges[x].SupportsRESTTickerBatchUpdates()
				assetTypes, err := exchange.GetExchangeAssetTypes(exchangeName)
				if err != nil {
					log.Printf("failed to get %s exchange asset types. Error: %s",
						exchangeName, err)
					return
				}

				processTicker := func(exch exchange.IBotExchange, update bool, c pair.CurrencyPair, assetType string) {
					var result ticker.Price
					var err error
					if update {
						result, err = exch.UpdateTicker(c, assetType)
					} else {
						result, err = exch.GetTickerPrice(c, assetType)
					}
					printTickerSummary(result, c, assetType, exchangeName, err)
					if err == nil {
						bot.comms.StageTickerData(exchangeName, assetType, result)
						if bot.config.Webserver.Enabled {
							relayWebsocketEvent(result, "ticker_update", assetType, exchangeName)
						}
					}
				}

				for y := range assetTypes {
					for z := range enabledCurrencies {
						if supportsBatching && z > 0 {
							processTicker(bot.exchanges[x], false, enabledCurrencies[z], assetTypes[y])
							continue
						}
						processTicker(bot.exchanges[x], true, enabledCurrencies[z], assetTypes[y])
					}
				}
			}(x, &wg)
		}
		wg.Wait()
		log.Println("All enabled currency tickers fetched.")
		time.Sleep(time.Second * 10)
	}
}

// OrderbookUpdaterRoutine fetches and updates the orderbooks for all enabled
// currency pairs and exchanges
func OrderbookUpdaterRoutine() {
	log.Println("Starting orderbook updater routine.")
	var wg sync.WaitGroup
	for {
		wg.Add(len(bot.exchanges))
		for x := range bot.exchanges {
			go func(x int, wg *sync.WaitGroup) {
				defer wg.Done()

				if bot.exchanges[x] == nil {
					return
				}
				exchangeName := bot.exchanges[x].GetName()
				enabledCurrencies := bot.exchanges[x].GetEnabledCurrencies()
				assetTypes, err := exchange.GetExchangeAssetTypes(exchangeName)
				if err != nil {
					log.Printf("failed to get %s exchange asset types. Error: %s",
						exchangeName, err)
					return
				}

				processOrderbook := func(exch exchange.IBotExchange, c pair.CurrencyPair, assetType string) {
					result, err := exch.UpdateOrderbook(c, assetType)
					printOrderbookSummary(result, c, assetType, exchangeName, err)
					if err == nil {
						bot.comms.StageOrderbookData(exchangeName, assetType, result)
						if bot.config.Webserver.Enabled {
							relayWebsocketEvent(result, "orderbook_update", assetType, exchangeName)
						}
					}
				}

				for y := range assetTypes {
					for z := range enabledCurrencies {
						processOrderbook(bot.exchanges[x], enabledCurrencies[z], assetTypes[y])
					}
				}
			}(x, &wg)
		}
		wg.Wait()
		log.Println("All enabled currency orderbooks fetched.")
		time.Sleep(time.Second * 10)
	}
}
