package main

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

func printCurrencyFormat(price float64) string {
	displaySymbol, err := currency.GetSymbolByCurrencyName(bot.config.Currency.FiatDisplayCurrency)
	if err != nil {
		log.Errorf("Failed to get display symbol: %s", err)
	}

	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origCurrency currency.Code, origPrice float64) string {
	displayCurrency := bot.config.Currency.FiatDisplayCurrency
	conv, err := currency.ConvertCurrency(origPrice,
		origCurrency,
		displayCurrency)
	if err != nil {
		log.Errorf("Failed to convert currency: %s", err)
	}

	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf("Failed to get display symbol: %s", err)
	}

	origSymbol, err := currency.GetSymbolByCurrencyName(origCurrency)
	if err != nil {
		log.Errorf("Failed to get original currency symbol for %s: %s",
			origCurrency,
			err)
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

func printTickerSummary(result *ticker.Price, p currency.Pair, assetType, exchangeName string, err error) {
	if err != nil {
		log.Errorf("Failed to get %s %s ticker. Error: %s",
			p.String(),
			exchangeName,
			err)
		return
	}

	stats.Add(exchangeName, p, assetType, result.Last, result.Volume)
	if p.Quote.IsFiatCurrency() &&
		p.Quote != bot.config.Currency.FiatDisplayCurrency {
		origCurrency := p.Quote.Upper()
		log.Infof("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
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
		if p.Quote.IsFiatCurrency() &&
			p.Quote == bot.config.Currency.FiatDisplayCurrency {
			log.Infof("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
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
			log.Infof("%s %s %s: TICKER: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f",
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

func printOrderbookSummary(result *orderbook.Base, p currency.Pair, assetType, exchangeName string, err error) {
	if err != nil {
		log.Errorf("Failed to get %s %s orderbook of type %s. Error: %s",
			p,
			exchangeName,
			assetType,
			err)
		return
	}

	bidsAmount, bidsValue := result.TotalBidsAmount()
	asksAmount, asksValue := result.TotalAsksAmount()

	if p.Quote.IsFiatCurrency() &&
		p.Quote != bot.config.Currency.FiatDisplayCurrency {
		origCurrency := p.Quote.Upper()
		log.Infof("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s",
			exchangeName,
			exchange.FormatCurrency(p).String(),
			assetType,
			len(result.Bids),
			bidsAmount,
			p.Base.String(),
			printConvertCurrencyFormat(origCurrency, bidsValue),
			len(result.Asks),
			asksAmount,
			p.Base.String(),
			printConvertCurrencyFormat(origCurrency, asksValue),
		)
	} else {
		if p.Quote.IsFiatCurrency() &&
			p.Quote == bot.config.Currency.FiatDisplayCurrency {
			log.Infof("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				len(result.Bids),
				bidsAmount,
				p.Base.String(),
				printCurrencyFormat(bidsValue),
				len(result.Asks),
				asksAmount,
				p.Base.String(),
				printCurrencyFormat(asksValue),
			)
		} else {
			log.Infof("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %f Asks len: %d Amount: %f %s. Total value: %f",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				len(result.Bids),
				bidsAmount,
				p.Base.String(),
				bidsValue,
				len(result.Asks),
				asksAmount,
				p.Base.String(),
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
		log.Errorf("Failed to broadcast websocket event %v. Error: %s",
			event, err)
	}
}

// TickerUpdaterRoutine fetches and updates the ticker for all enabled
// currency pairs and exchanges
func TickerUpdaterRoutine() {
	log.Debugf("Starting ticker updater routine.")
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
					log.Debugf("failed to get %s exchange asset types. Error: %s",
						exchangeName, err)
					return
				}

				processTicker := func(exch exchange.IBotExchange, update bool, c currency.Pair, assetType string) {
					var result ticker.Price
					var err error
					if update {
						result, err = exch.UpdateTicker(c, assetType)
					} else {
						result, err = exch.GetTickerPrice(c, assetType)
					}
					printTickerSummary(&result, c, assetType, exchangeName, err)
					if err == nil {
						bot.comms.StageTickerData(exchangeName, assetType, &result)
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
		log.Debugln("All enabled currency tickers fetched.")
		time.Sleep(time.Second * 10)
	}
}

// OrderbookUpdaterRoutine fetches and updates the orderbooks for all enabled
// currency pairs and exchanges
func OrderbookUpdaterRoutine() {
	log.Debugln("Starting orderbook updater routine.")
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
					log.Errorf("failed to get %s exchange asset types. Error: %s",
						exchangeName, err)
					return
				}

				processOrderbook := func(exch exchange.IBotExchange, c currency.Pair, assetType string) {
					result, err := exch.UpdateOrderbook(c, assetType)
					printOrderbookSummary(&result, c, assetType, exchangeName, err)
					if err == nil {
						bot.comms.StageOrderbookData(exchangeName, assetType, &result)
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
		log.Debugln("All enabled currency orderbooks fetched.")
		time.Sleep(time.Second * 10)
	}
}

// WebsocketRoutine Initial routine management system for websocket
func WebsocketRoutine(verbose bool) {
	log.Debugln("Connecting exchange websocket services...")

	for i := range bot.exchanges {
		go func(i int) {
			if verbose {
				log.Debugf("Establishing websocket connection for %s",
					bot.exchanges[i].GetName())
			}

			ws, err := bot.exchanges[i].GetWebsocket()
			if err != nil {
				log.Debugf("Websocket not enabled for %s",
					bot.exchanges[i].GetName())
				return
			}

			// Data handler routine
			go WebsocketDataHandler(ws, verbose)

			err = ws.Connect()
			if err != nil {
				switch err.Error() {
				case exchange.WebsocketNotEnabled:
					log.Warnf("%s - websocket disabled", bot.exchanges[i].GetName())
				default:
					log.Error(err)
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
		log.Errorf("routines.go error - failed to shutodwn %s", err)
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
				log.Debugf("exchange %s websocket feed connected", ws.GetName())
			}

		case <-ws.Disconnected:
			if verbose {
				log.Debugf("exchange %s websocket feed disconnected, switching to REST functionality",
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
			switch d := data.(type) {
			case string:
				switch d {
				case exchange.WebsocketNotEnabled:
					if verbose {
						log.Warnf("routines.go warning - exchange %s weboscket not enabled",
							ws.GetName())
					}

				default:
					log.Infof(d)
				}

			case error:
				switch {
				case common.StringContains(d.Error(), "close 1006"):
					go ws.WebsocketReset()
					continue
				default:
					log.Errorf("routines.go exchange %s websocket error - %s", ws.GetName(), data)
				}

			case exchange.TradeData:
				// Trade Data
				if verbose {
					log.Infoln("Websocket trades Updated:   ", d)
				}

			case exchange.TickerData:
				// Ticker data
				if verbose {
					log.Infoln("Websocket Ticker Updated:   ", d)
				}
			case exchange.KlineData:
				// Kline data
				if verbose {
					log.Infoln("Websocket Kline Updated:    ", d)
				}
			case exchange.WebsocketOrderbookUpdate:
				// Orderbook data
				if verbose {
					log.Infoln("Websocket Orderbook Updated:", d)
				}
			default:
				if verbose {
					log.Warnf("Websocket Unknown type:     %s", d)
				}
			}
		}
	}
}
