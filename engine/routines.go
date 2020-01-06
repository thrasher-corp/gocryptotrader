package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
		log.Errorf(log.Global,
			"Failed to get original currency symbol for %s: %s\n",
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

func printTickerSummary(t *ticker.Price, protocol string, err error) {
	if err != nil {
		log.Errorf(log.Ticker,
			"Failed to get %s %s %s %s ticker. Error: %s\n",
			t.ExchangeName,
			protocol,
			t.Pair,
			t.AssetType,
			err)
		return
	}

	stats.Add(t.ExchangeName, t.Pair, t.AssetType, t.Last, t.Volume)
	if t.Pair.Quote.IsFiatCurrency() &&
		t.Pair.Quote != Bot.Config.Currency.FiatDisplayCurrency {
		origCurrency := t.Pair.Quote.Upper()
		log.Infof(log.Ticker,
			"%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
			t.ExchangeName,
			protocol,
			FormatCurrency(t.Pair),
			strings.ToUpper(t.AssetType.String()),
			printConvertCurrencyFormat(origCurrency, t.Last),
			printConvertCurrencyFormat(origCurrency, t.Ask),
			printConvertCurrencyFormat(origCurrency, t.Bid),
			printConvertCurrencyFormat(origCurrency, t.High),
			printConvertCurrencyFormat(origCurrency, t.Low),
			t.Volume)
	} else {
		if t.Pair.Quote.IsFiatCurrency() &&
			t.Pair.Quote == Bot.Config.Currency.FiatDisplayCurrency {
			log.Infof(log.Ticker,
				"%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
				t.ExchangeName,
				protocol,
				FormatCurrency(t.Pair),
				strings.ToUpper(t.AssetType.String()),
				printCurrencyFormat(t.Last),
				printCurrencyFormat(t.Ask),
				printCurrencyFormat(t.Bid),
				printCurrencyFormat(t.High),
				printCurrencyFormat(t.Low),
				t.Volume)
		} else {
			log.Infof(log.Ticker,
				"%s %s %s %s: TICKER: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f\n",
				t.ExchangeName,
				protocol,
				FormatCurrency(t.Pair),
				strings.ToUpper(t.AssetType.String()),
				t.Last,
				t.Ask,
				t.Bid,
				t.High,
				t.Low,
				t.Volume)
		}
	}
}

func printOrderbookSummary(o *orderbook.Base, protocol string, err error) {
	if err != nil {
		log.Errorf(log.OrderBook,
			"Failed to get %s %s %s orderbook of type %s. Error: %s\n",
			o.ExchangeName,
			protocol,
			o.Pair,
			o.AssetType,
			err)
		return
	}

	bidsAmount, bidsValue := o.TotalBidsAmount()
	asksAmount, asksValue := o.TotalAsksAmount()

	if o.Pair.Quote.IsFiatCurrency() &&
		o.Pair.Quote != Bot.Config.Currency.FiatDisplayCurrency {
		origCurrency := o.Pair.Quote.Upper()
		log.Infof(log.OrderBook,
			"%s %s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s Liquidity Ratio: %f\n",
			o.ExchangeName,
			protocol,
			FormatCurrency(o.Pair),
			strings.ToUpper(o.AssetType.String()),
			len(o.Bids),
			bidsAmount,
			o.Pair.Base,
			printConvertCurrencyFormat(origCurrency, bidsValue),
			len(o.Asks),
			asksAmount,
			o.Pair.Base,
			printConvertCurrencyFormat(origCurrency, asksValue),
			bidsValue/asksValue,
		)
	} else {
		if o.Pair.Quote.IsFiatCurrency() &&
			o.Pair.Quote == Bot.Config.Currency.FiatDisplayCurrency {
			log.Infof(log.OrderBook,
				"%s %s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s Liquidity Ratio: %f\n",
				o.ExchangeName,
				protocol,
				FormatCurrency(o.Pair),
				strings.ToUpper(o.AssetType.String()),
				len(o.Bids),
				bidsAmount,
				o.Pair.Base,
				printCurrencyFormat(bidsValue),
				len(o.Asks),
				asksAmount,
				o.Pair.Base,
				printCurrencyFormat(asksValue),
				bidsValue/asksValue,
			)
		} else {
			log.Infof(log.OrderBook,
				"%s %s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %f Asks len: %d Amount: %f %s. Total value: %f Liquidity Ratio: %f\n",
				o.ExchangeName,
				protocol,
				FormatCurrency(o.Pair),
				strings.ToUpper(o.AssetType.String()),
				len(o.Bids),
				bidsAmount,
				o.Pair.Base,
				bidsValue,
				len(o.Asks),
				asksAmount,
				o.Pair.Base,
				asksValue,
				bidsValue/asksValue,
			)
		}
	}
}

func printAccountSummary(ai *exchange.AccountInfo, protocol string) {
	for x := range ai.Accounts {
		var account string
		if ai.Accounts[x].ID != "" {
			account = ai.Accounts[x].ID
		} else {
			account = strconv.FormatInt(int64(x), 10)
		}

		for y := range ai.Accounts[x].Currencies {
			if ai.Accounts[x].Currencies[y].TotalValue == 0 {
				continue
			}
			log.Infof(log.Global,
				"%s %s: ACCOUNT UPDATE: AccountID:%s Currency:%s Amount:%f Hold:%f",
				ai.Exchange,
				protocol,
				account,
				ai.Accounts[x].Currencies[y].CurrencyName,
				ai.Accounts[x].Currencies[y].TotalValue,
				ai.Accounts[x].Currencies[y].Hold)
		}
	}
}

func printTradeSummary(t []order.Trade, protocol string) {
	if len(t) != 0 {
		i := len(t) - 1 // Temp stop spam
		log.Infof(log.Global,
			"%s %s: TRADE: Pair:%s Asset:%s Price:%f Amount:%f TradeID:%s Executed @ %s",
			t[i].Exchange,
			protocol,
			t[i].Pair,
			t[i].AssetType,
			t[i].Price,
			t[i].Amount,
			t[i].TID,
			t[i].Timestamp.Format(time.RFC822),
		)
	}
}

func printOrderSummary(o []order.Detail, protocol string) {
	for i := range o {
		log.Infof(log.Global,
			"%s %s: ORDER UPDATE: AccountID:%s Pair:%s Asset:%s Price:%f Amount:%f Status:%s Side:%s Type:%s OrderID:%s",
			o[i].Exchange,
			protocol,
			o[i].AccountID,
			o[i].CurrencyPair,
			o[i].AssetType,
			o[i].Price,
			o[i].Amount,
			o[i].Status,
			o[i].OrderSide,
			o[i].OrderType,
			o[i].ID)
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

	for i := range Bot.Exchanges {
		go func(i int) {
			if Bot.Exchanges[i].SupportsWebsocket() {
				if Bot.Settings.Verbose {
					log.Debugf(log.WebsocketMgr, "Exchange %s websocket support: Yes Enabled: %v\n", Bot.Exchanges[i].GetName(),
						common.IsEnabled(Bot.Exchanges[i].IsWebsocketEnabled()))
				}

				// TO-DO: expose IsConnected() and IsConnecting so this can be
				// simplified
				if Bot.Exchanges[i].IsWebsocketEnabled() {
					ws, err := Bot.Exchanges[i].GetWebsocket()
					if err != nil {
						log.Errorf(log.WebsocketMgr, "Exchange %s GetWebsocket error: %s\n",
							Bot.Exchanges[i].GetName(), err)
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
				log.Debugf(log.WebsocketMgr, "Exchange %s websocket support: No\n", Bot.Exchanges[i].GetName())
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
						log.Warnf(log.WebsocketMgr,
							"routines.go warning - exchange %s websocket not enabled\n",
							ws.GetName())
					}
				default:
					log.Info(log.WebsocketMgr, d)
				}
			case error:
				log.Errorf(log.WebsocketMgr,
					"routines.go exchange %s websocket error - %s",
					ws.GetName(),
					data)
			case []order.Detail:
				Bot.ExchangeCurrencyPairManager.StreamUpdate(d)
			case wshandler.TradeData:
				Bot.ExchangeCurrencyPairManager.StreamUpdate(d)
			case []order.Trade:
				Bot.ExchangeCurrencyPairManager.StreamUpdate(d)
			case *exchange.Funding:
				Bot.ExchangeCurrencyPairManager.StreamUpdate(d)
			case *ticker.Price:
				Bot.ExchangeCurrencyPairManager.StreamUpdate(d)
			case wshandler.KlineData:
				Bot.ExchangeCurrencyPairManager.StreamUpdate(d)
			case wshandler.WebsocketOrderbookUpdate:
				// TODO: RM this as this adds overhead, pass the pointer
				// to the orderbook around
				storedOB, err := orderbook.Get(d.Exchange, d.Pair, d.Asset)
				if err != nil {
					log.Errorf(log.WebsocketMgr,
						"fetching internal orderbook %s",
						err)
					continue
				}
				Bot.ExchangeCurrencyPairManager.StreamUpdate(storedOB)
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
