package engine

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

func printCurrencyFormat(price float64) string {
	displaySymbol, err := currency.GetSymbolByCurrencyName(Bot.Config.Currency.FiatDisplayCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to get display symbol: %s\n", err)
	}

	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origCurrency currency.Code, origPrice float64) string {
	displayCurrency := Bot.Config.Currency.FiatDisplayCurrency
	conv, err := currency.ConvertCurrency(origPrice,
		origCurrency,
		displayCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to convert currency: %s\n", err)
	}

	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to get display symbol: %s\n", err)
	}

	origSymbol, err := currency.GetSymbolByCurrencyName(origCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to get original currency symbol for %s: %s\n",
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

func printTickerSummary(result *ticker.Price, p currency.Pair, assetType asset.Item, exchangeName, protocol string, err error) {
	if err != nil {
		log.Errorf(log.Ticker, "Failed to get %s %s %s %s ticker. Error: %s\n",
			exchangeName,
			protocol,
			p,
			assetType,
			err)
		return
	}

	stats.Add(exchangeName, p, assetType, result.Last, result.Volume)
	if p.Quote.IsFiatCurrency() &&
		p.Quote != Bot.Config.Currency.FiatDisplayCurrency {
		origCurrency := p.Quote.Upper()
		log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
			exchangeName,
			protocol,
			FormatCurrency(p),
			strings.ToUpper(assetType.String()),
			printConvertCurrencyFormat(origCurrency, result.Last),
			printConvertCurrencyFormat(origCurrency, result.Ask),
			printConvertCurrencyFormat(origCurrency, result.Bid),
			printConvertCurrencyFormat(origCurrency, result.High),
			printConvertCurrencyFormat(origCurrency, result.Low),
			result.Volume)
	} else {
		if p.Quote.IsFiatCurrency() &&
			p.Quote == Bot.Config.Currency.FiatDisplayCurrency {
			log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
				exchangeName,
				protocol,
				FormatCurrency(p),
				strings.ToUpper(assetType.String()),
				printCurrencyFormat(result.Last),
				printCurrencyFormat(result.Ask),
				printCurrencyFormat(result.Bid),
				printCurrencyFormat(result.High),
				printCurrencyFormat(result.Low),
				result.Volume)
		} else {
			log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f\n",
				exchangeName,
				protocol,
				FormatCurrency(p),
				strings.ToUpper(assetType.String()),
				result.Last,
				result.Ask,
				result.Bid,
				result.High,
				result.Low,
				result.Volume)
		}
	}
}

func printOrderbookSummary(result *orderbook.Base, p currency.Pair, assetType asset.Item, exchangeName, protocol string, err error) {
	if err != nil {
		log.Errorf(log.OrderBook, "Failed to get %s %s %s orderbook of type %s. Error: %s\n",
			exchangeName,
			protocol,
			p,
			assetType,
			err)
		return
	}

	bidsAmount, bidsValue := result.TotalBidsAmount()
	asksAmount, asksValue := result.TotalAsksAmount()

	if p.Quote.IsFiatCurrency() &&
		p.Quote != Bot.Config.Currency.FiatDisplayCurrency {
		origCurrency := p.Quote.Upper()
		log.Infof(log.OrderBook, "%s %s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s\n",
			exchangeName,
			protocol,
			FormatCurrency(p),
			strings.ToUpper(assetType.String()),
			len(result.Bids),
			bidsAmount,
			p.Base,
			printConvertCurrencyFormat(origCurrency, bidsValue),
			len(result.Asks),
			asksAmount,
			p.Base,
			printConvertCurrencyFormat(origCurrency, asksValue),
		)
	} else {
		if p.Quote.IsFiatCurrency() &&
			p.Quote == Bot.Config.Currency.FiatDisplayCurrency {
			log.Infof(log.OrderBook, "%s %s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s\n",
				exchangeName,
				protocol,
				FormatCurrency(p),
				strings.ToUpper(assetType.String()),
				len(result.Bids),
				bidsAmount,
				p.Base,
				printCurrencyFormat(bidsValue),
				len(result.Asks),
				asksAmount,
				p.Base,
				printCurrencyFormat(asksValue),
			)
		} else {
			log.Infof(log.OrderBook, "%s %s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %f Asks len: %d Amount: %f %s. Total value: %f\n",
				exchangeName,
				protocol,
				FormatCurrency(p),
				strings.ToUpper(assetType.String()),
				len(result.Bids),
				bidsAmount,
				p.Base,
				bidsValue,
				len(result.Asks),
				asksAmount,
				p.Base,
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
		log.Errorf(log.WebsocketMgr, "Failed to broadcast websocket event %v. Error: %s\n",
			event, err)
	}
}

// WebsocketRoutine Initial routine management system for websocket
func WebsocketRoutine() {
	if Bot.Settings.Verbose {
		log.Debugln(log.WebsocketMgr, "Connecting exchange websocket services...")
	}

	exchanges := GetExchanges()
	for i := range exchanges {
		go func(i int) {
			if exchanges[i].SupportsWebsocket() {
				if Bot.Settings.Verbose {
					log.Debugf(log.WebsocketMgr,
						"Exchange %s websocket support: Yes Enabled: %v\n",
						exchanges[i].GetName(),
						common.IsEnabled(exchanges[i].IsWebsocketEnabled()),
					)
				}

				// TO-DO: expose IsConnected() and IsConnecting so this can be simplified
				if exchanges[i].IsWebsocketEnabled() {
					ws, err := exchanges[i].GetWebsocket()
					if err != nil {
						log.Errorf(
							log.WebsocketMgr,
							"Exchange %s GetWebsocket error: %s\n",
							exchanges[i].GetName(),
							err,
						)
						return
					}

					// Exchange sync manager might have already started ws
					// service or is in the process of connecting, so check
					if ws.IsConnected() || ws.IsConnecting() {
						return
					}

					// Data handler routine
					go WebsocketDataHandler(ws)

					err = ws.Connect()
					if err != nil {
						log.Errorf(log.WebsocketMgr, "%v\n", err)
					}
				}
			} else if Bot.Settings.Verbose {
				log.Debugf(log.WebsocketMgr,
					"Exchange %s websocket support: No\n",
					exchanges[i].GetName(),
				)
			}
		}(i)
	}
}

var shutdowner = make(chan struct{}, 1)
var wg sync.WaitGroup

// Websocketshutdown shuts down the exchange routines and then shuts down
// governing routines
func Websocketshutdown(ws *wshandler.Websocket) error {
	err := ws.Shutdown() // shutdown routines on the exchange
	if err != nil {
		log.Errorf(log.WebsocketMgr, "routines.go error - failed to shutdown %s\n", err)
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

// WebsocketDataHandler handles websocket data coming from a websocket feed
// associated with an exchange
func WebsocketDataHandler(ws *wshandler.Websocket) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-shutdowner:
			return

		case data := <-ws.DataHandler:
			switch d := data.(type) {
			case string:
				switch d {
				case wshandler.WebsocketNotEnabled:
					if Bot.Settings.Verbose {
						log.Warnf(log.WebsocketMgr, "routines.go warning - exchange %s websocket not enabled\n",
							ws.GetName())
					}
				default:
					log.Info(log.WebsocketMgr, d)
				}
			case error:
				log.Errorf(log.WebsocketMgr, "routines.go exchange %s websocket error - %s", ws.GetName(), data)
			case wshandler.TradeData:
				// Websocket Trade Data
				if Bot.Settings.Verbose {
					log.Infof(log.WebsocketMgr, "%s websocket %s %s trade updated %+v\n",
						ws.GetName(),
						FormatCurrency(d.CurrencyPair),
						d.AssetType,
						d)
				}
			case wshandler.FundingData:
				// Websocket Funding Data
				if Bot.Settings.Verbose {
					log.Infof(log.WebsocketMgr, "%s websocket %s %s funding updated %+v\n",
						ws.GetName(),
						FormatCurrency(d.CurrencyPair),
						d.AssetType,
						d)
				}
			case *ticker.Price:
				// Websocket Ticker Data
				if Bot.Settings.EnableExchangeSyncManager && Bot.ExchangeCurrencyPairManager != nil {
					Bot.ExchangeCurrencyPairManager.update(ws.GetName(),
						d.Pair,
						d.AssetType,
						SyncItemTicker,
						nil)
				}
				err := ticker.ProcessTicker(ws.GetName(), d, d.AssetType)
				printTickerSummary(d, d.Pair, d.AssetType, ws.GetName(), "websocket", err)
			case wshandler.KlineData:
				// Websocket Kline Data
				if Bot.Settings.Verbose {
					log.Infof(log.WebsocketMgr, "%s websocket %s %s kline updated %+v\n",
						ws.GetName(),
						FormatCurrency(d.Pair),
						d.AssetType,
						d)
				}
			case wshandler.WebsocketOrderbookUpdate:
				// Websocket Orderbook Data
				result := data.(wshandler.WebsocketOrderbookUpdate)
				if Bot.Settings.EnableExchangeSyncManager && Bot.ExchangeCurrencyPairManager != nil {
					Bot.ExchangeCurrencyPairManager.update(ws.GetName(),
						result.Pair,
						result.Asset,
						SyncItemOrderbook,
						nil)
				}

				if Bot.Settings.Verbose {
					log.Infof(log.WebsocketMgr,
						"%s websocket %s %s orderbook updated\n",
						ws.GetName(),
						FormatCurrency(result.Pair),
						d.Asset)
				}
			default:
				if Bot.Settings.Verbose {
					log.Warnf(log.WebsocketMgr,
						"%s websocket Unknown type: %+v\n",
						ws.GetName(),
						d)
				}
			}
		}
	}
}
