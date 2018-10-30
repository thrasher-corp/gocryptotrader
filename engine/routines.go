package engine

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

func printCurrencyFormat(price float64) string {
	displaySymbol, err := currency.GetSymbolByCurrencyName(Bot.Config.Currency.FiatDisplayCurrency)
	if err != nil {
		log.Errorf("Failed to get display symbol: %s", err)
	}

	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origCurrency currency.Code, origPrice float64) string {
	displayCurrency := Bot.Config.Currency.FiatDisplayCurrency
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

func printTickerSummary(result *ticker.Price, p currency.Pair, assetType assets.AssetType, exchangeName string, err error) {
	if err != nil {
		log.Errorf("Failed to get %s %s ticker. Error: %s",
			p.String(),
			exchangeName,
			err)
		return
	}

	stats.Add(exchangeName, p, assetType, result.Last, result.Volume)
	if p.Quote.IsFiatCurrency() &&
		p.Quote != Bot.Config.Currency.FiatDisplayCurrency {
		origCurrency := p.Quote.Upper()
		log.Infof("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
			exchangeName,
			FormatCurrency(p).String(),
			assetType,
			printConvertCurrencyFormat(origCurrency, result.Last),
			printConvertCurrencyFormat(origCurrency, result.Ask),
			printConvertCurrencyFormat(origCurrency, result.Bid),
			printConvertCurrencyFormat(origCurrency, result.High),
			printConvertCurrencyFormat(origCurrency, result.Low),
			result.Volume)
	} else {
		if p.Quote.IsFiatCurrency() &&
			p.Quote == Bot.Config.Currency.FiatDisplayCurrency {
			log.Infof("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
				exchangeName,
				FormatCurrency(p).String(),
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
				FormatCurrency(p).String(),
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

func printOrderbookSummary(result *orderbook.Base, p currency.Pair, assetType assets.AssetType, exchangeName string, err error) {
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
		p.Quote != Bot.Config.Currency.FiatDisplayCurrency {
		origCurrency := p.Quote.Upper()
		log.Infof("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s",
			exchangeName,
			FormatCurrency(p).String(),
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
			p.Quote == Bot.Config.Currency.FiatDisplayCurrency {
			log.Infof("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s",
				exchangeName,
				FormatCurrency(p).String(),
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
				FormatCurrency(p).String(),
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
		log.Errorf("Failed to broadcast websocket event. Error: %s",
			err)
	}
}

// TickerUpdaterRoutine fetches and updates the ticker for all enabled
// currency pairs and exchanges
func TickerUpdaterRoutine() {
	log.Debugf("Starting ticker updater routine.")
	var wg sync.WaitGroup
	for {
		wg.Add(len(Bot.Exchanges))
		for x := range Bot.Exchanges {
			go func(x int, wg *sync.WaitGroup) {
				defer wg.Done()

				if Bot.Exchanges[x] == nil || !Bot.Exchanges[x].SupportsREST() {
					return
				}

				exchangeName := Bot.Exchanges[x].GetName()
				supportsBatching := Bot.Exchanges[x].SupportsRESTTickerBatchUpdates()
				assetTypes := Bot.Exchanges[x].GetAssetTypes()

				processTicker := func(exch exchange.IBotExchange, update bool, c currency.Pair, assetType assets.AssetType) {
					var result ticker.Price
					var err error
					if update {
						result, err = exch.UpdateTicker(c, assetType)
					} else {
						result, err = exch.FetchTicker(c, assetType)
					}
					printTickerSummary(&result, c, assetType, exchangeName, err)
					if err == nil {
						Bot.CommsRelayer.StageTickerData(exchangeName, assetType, &result)
						if Bot.Config.RemoteControl.WebsocketRPC.Enabled {
							relayWebsocketEvent(result, "ticker_update", assetType.String(), exchangeName)
						}
					}
				}

				for y := range assetTypes {
					enabledCurrencies := Bot.Exchanges[x].GetEnabledPairs(assetTypes[y])
					for z := range enabledCurrencies {
						if supportsBatching && z > 0 {
							processTicker(Bot.Exchanges[x], false, enabledCurrencies[z], assetTypes[y])
							continue
						}
						processTicker(Bot.Exchanges[x], true, enabledCurrencies[z], assetTypes[y])
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
		wg.Add(len(Bot.Exchanges))
		for x := range Bot.Exchanges {
			go func(x int, wg *sync.WaitGroup) {
				defer wg.Done()

				if Bot.Exchanges[x] == nil || !Bot.Exchanges[x].SupportsREST() {
					return
				}

				exchangeName := Bot.Exchanges[x].GetName()
				assetTypes := Bot.Exchanges[x].GetAssetTypes()

				processOrderbook := func(exch exchange.IBotExchange, c currency.Pair, assetType assets.AssetType) {
					result, err := exch.UpdateOrderbook(c, assetType)
					printOrderbookSummary(&result, c, assetType, exchangeName, err)
					if err == nil {
						Bot.CommsRelayer.StageOrderbookData(exchangeName, assetType, &result)
						if Bot.Config.RemoteControl.WebsocketRPC.Enabled {
							relayWebsocketEvent(result, "orderbook_update", assetType.String(), exchangeName)
						}
					}
				}

				for y := range assetTypes {
					enabledCurrencies := Bot.Exchanges[x].GetEnabledPairs(assetTypes[y])
					for z := range enabledCurrencies {
						processOrderbook(Bot.Exchanges[x], enabledCurrencies[z], assetTypes[y])
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
func WebsocketRoutine() {
	if Bot.Settings.Verbose {
		log.Debugln("Connecting exchange websocket services...")
	}

	for i := range Bot.Exchanges {
		go func(i int) {
			if Bot.Exchanges[i].SupportsWebsocket() {
				if Bot.Settings.Verbose {
					log.Debugf("Exchange %s websocket support: Yes Enabled: %v", Bot.Exchanges[i].GetName(),
						common.IsEnabled(Bot.Exchanges[i].IsWebsocketEnabled()))
				}

				if Bot.Exchanges[i].IsWebsocketEnabled() {
					ws, err := Bot.Exchanges[i].GetWebsocket()
					if err != nil {
						return
					}
					// Data handler routine
					go WebsocketDataHandler(ws)

					err = ws.Connect()
					if err != nil {
						log.Println(err)
					}
				}
			} else if Bot.Settings.Verbose {
				log.Debugf("Exchange %s websocket support: No", Bot.Exchanges[i].GetName())
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
func streamDiversion(ws *exchange.Websocket) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-shutdowner:
			return

		case <-ws.Connected:
			if Bot.Settings.Verbose {
				log.Debugf("exchange %s websocket feed connected", ws.GetName())
			}

		case <-ws.Disconnected:
			if Bot.Settings.Verbose {
				log.Debugf("exchange %s websocket feed disconnected, switching to REST functionality",
					ws.GetName())
			}
		}
	}
}

// WebsocketDataHandler handles websocket data coming from a websocket feed
// associated with an exchange
func WebsocketDataHandler(ws *exchange.Websocket) {
	wg.Add(1)
	defer wg.Done()

	go streamDiversion(ws)

	for {
		select {
		case <-shutdowner:
			return

		case data := <-ws.DataHandler:
			switch d := data.(type) {
			case string:
				switch d {
				case exchange.WebsocketNotEnabled:
					if Bot.Settings.Verbose {
						log.Warnf("routines.go warning - exchange %s weboscket not enabled",
							ws.GetName())
					}

				default:
					log.Infof(d)
				}

			case error:
				switch {
				case common.StringContains(d.Error(), "close 1006"):
					go WebsocketReconnect(ws, Bot.Settings.Verbose)
					continue
				default:
					log.Errorf("routines.go exchange %s websocket error - %s", ws.GetName(), data)
				}

			case exchange.TradeData:
				// Trade Data
				// if Bot.Settings.Verbose {
				//	log.Println("Websocket trades Updated:   ", data.(exchange.TradeData))
				// }

			case exchange.TickerData:
				// Ticker data
				// if Bot.Settings.Verbose {
				//	log.Println("Websocket Ticker Updated:   ", data.(exchange.TickerData))
				// }

				tickerNew := ticker.Price{
					Pair:        d.Pair,
					LastUpdated: d.Timestamp,
					Last:        d.ClosePrice,
					High:        d.HighPrice,
					Low:         d.LowPrice,
					Volume:      d.Quantity,
				}
				Bot.ExchangeCurrencyPairManager.update(ws.GetName(), d.Pair, d.AssetType, SyncItemTicker, nil)
				ticker.ProcessTicker(ws.GetName(), &tickerNew, d.AssetType)
				printTickerSummary(&tickerNew, tickerNew.Pair, d.AssetType, ws.GetName(), nil)
			case exchange.KlineData:
				// Kline data
				if Bot.Settings.Verbose {
					log.Infoln("Websocket Kline Updated:    ", d)
				}
			case exchange.WebsocketOrderbookUpdate:
				// Orderbook data
				result := data.(exchange.WebsocketOrderbookUpdate)
				Bot.ExchangeCurrencyPairManager.update(ws.GetName(), result.Pair, result.Asset, SyncItemOrderbook, nil)
				//nolint:gocritic log.Infof("Websocket %s %s orderbook updated", ws.GetName(), result.Pair.Pair().String())
			default:
				if Bot.Settings.Verbose {
					log.Warnf("Websocket Unknown type:     %s", d)
				}
			}
		}
	}
}

// WebsocketReconnect tries to reconnect to a websocket stream
func WebsocketReconnect(ws *exchange.Websocket, verbose bool) {
	if verbose {
		log.Debugf("Websocket reconnection requested for %s", ws.GetName())
	}

	err := ws.Shutdown()
	if err != nil {
		log.Error(err)
		return
	}

	wg.Add(1)
	defer wg.Done()

	tick := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-shutdowner:
			return

		case <-tick.C:
			err = ws.Connect()
			if err == nil {
				return
			}
		}
	}
}
