package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func printCurrencyFormat(price float64) string {
	displaySymbol, err := symbol.GetSymbolByCurrencyName(bot.config.Currency.FiatDisplayCurrency)
	if err != nil {
		log.Printf("Failed to get display symbol: %s", err)
	}

	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origCurrency string, origPrice float64) string {
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

	return fmt.Sprintf("%s%.2f %s (%s%.2f %s)",
		displaySymbol,
		conv,
		displayCurrency,
		origSymbol,
		origPrice,
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
		log.Printf("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
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
		if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.Upper().String() == bot.config.Currency.FiatDisplayCurrency {
			log.Printf("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
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
			log.Printf("%s %s %s: TICKER: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f",
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

	if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.String() != bot.config.Currency.FiatDisplayCurrency {
		origCurrency := p.SecondCurrency.Upper().String()
		log.Printf("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s",
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
		if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.Upper().String() == bot.config.Currency.FiatDisplayCurrency {
			log.Printf("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s",
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
			log.Printf("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %f Asks len: %d Amount: %f %s. Total value: %f",
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

// WebsocketRoutine Initial routine management system for websocket
func WebsocketRoutine(verbose bool) {
	log.Println("Connecting exchange websocket services...")

	for i := range bot.exchanges {
		go func(i int) {
			if verbose {
				log.Printf("Establishing websocket connection for %s",
					bot.exchanges[i].GetName())
			}

			ws, err := bot.exchanges[i].GetWebsocket()
			if err != nil {
				return
			}

			// Data handler routine
			go WebsocketDataHandler(ws, verbose)

			err = ws.Connect()
			if err != nil {
				switch err.Error() {
				case exchange.WebsocketNotEnabled:
					// Store in memory if enabled in future
				default:
					log.Println(err)
				}
			}
		}(i)
	}
}

var shutdowner = make(chan struct{}, 1)
var wg sync.WaitGroup

// Websocketshutdown shuts down the exchange routines and then shuts down
// governing routines
func Websocketshutdown(ws *exchange.Websocket) error {
	err := ws.Shutdown() // shutdown routines on the exchange
	if err != nil {
		log.Fatalf("routines.go error - failed to shutodwn %s", err)
	}

	timer := time.NewTimer(5 * time.Second)
	c := make(chan struct{}, 1)

	go func(c chan struct{}) {
		close(shutdowner)
		wg.Wait()
		c <- struct{}{}
	}(c)

	select {
	case <-timer.C:
		return errors.New("routines.go error - failed to shutdown routines")

	case <-c:
		return nil
	}
}

// streamDiversion is a diversion switch from websocket to REST or other
// alternative feed
func streamDiversion(ws *exchange.Websocket, verbose bool) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-shutdowner:
			return

		case <-ws.Connected:
			if verbose {
				log.Printf("exchange %s websocket feed connected", ws.GetName())
			}

		case <-ws.Disconnected:
			if verbose {
				log.Printf("exchange %s websocket feed disconnected, switching to REST functionality",
					ws.GetName())
			}
		}
	}
}

// WebsocketDataHandler handles websocket data coming from a websocket feed
// associated with an exchange
func WebsocketDataHandler(ws *exchange.Websocket, verbose bool) {
	wg.Add(1)
	defer wg.Done()

	go streamDiversion(ws, verbose)

	for {
		select {
		case <-shutdowner:
			return

		case data := <-ws.DataHandler:
			switch data.(type) {
			case string:
				switch data.(string) {
				case exchange.WebsocketNotEnabled:
					if verbose {
						log.Printf("routines.go warning - exchange %s weboscket not enabled",
							ws.GetName())
					}

				default:
					log.Println(data.(string))
				}

			case error:
				switch {
				case common.StringContains(data.(error).Error(), "close 1006"):
					go WebsocketReconnect(ws, verbose)
					continue
				default:
					log.Fatalf("routines.go exchange %s websocket error - %s", ws.GetName(), data)
				}

			case exchange.TradeData:
				// Trade Data
				if verbose {
					log.Println("Websocket trades Updated:   ", data.(exchange.TradeData))
				}

			case exchange.TickerData:
				// Ticker data
				if verbose {
					log.Println("Websocket Ticker Updated:   ", data.(exchange.TickerData))
				}
			case exchange.KlineData:
				// Kline data
				if verbose {
					log.Println("Websocket Kline Updated:    ", data.(exchange.KlineData))
				}
			case exchange.WebsocketOrderbookUpdate:
				// Orderbook data
				if verbose {
					log.Println("Websocket Orderbook Updated:", data.(exchange.WebsocketOrderbookUpdate))
				}
			default:
				if verbose {
					log.Println("Websocket Unknown type:     ", data)
				}
			}
		}
	}
}

// WebsocketReconnect tries to reconnect to a websocket stream
func WebsocketReconnect(ws *exchange.Websocket, verbose bool) {
	if verbose {
		log.Printf("Websocket reconnection requested for %s", ws.GetName())
	}

	err := ws.Shutdown()
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	defer wg.Done()

	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-shutdowner:
			return

		case <-ticker.C:
			err = ws.Connect()
			if err == nil {
				return
			}
		}
	}
}
