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
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/supported"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
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

	exch, err := supported.CheckExchange(t.ExchangeName)
	if err != nil {
		log.Errorln(log.SyncMgr, err)
		return
	}

	stats.Add(exch, t.Pair, t.AssetType, t.Last, t.Volume)
	if t.Pair.Quote.IsFiatCurrency() &&
		t.Pair.Quote != Bot.Config.Currency.FiatDisplayCurrency {
		origCurrency := t.Pair.Quote.Upper()
		log.Infof(log.Ticker,
			"%s %s: TICKER: %s %s:  Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
			exch,
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
				"%s %s: TICKER: %s %s:  Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
				exch,
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
				"%s %s: TICKER: %s %s: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f\n",
				exch,
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

	exch, err := supported.CheckExchange(o.ExchangeName)
	if err != nil {
		fmt.Println(err)
		return
	}

	bidsAmount, bidsValue := o.TotalBidsAmount()
	asksAmount, asksValue := o.TotalAsksAmount()

	if o.Pair.Quote.IsFiatCurrency() &&
		o.Pair.Quote != Bot.Config.Currency.FiatDisplayCurrency {
		origCurrency := o.Pair.Quote.Upper()
		log.Infof(log.OrderBook,
			"%s %s: ORDERBOOK: %s %s:  Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s Liquidity Ratio: %f\n",
			exch,
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
			bidsValue/asksValue)
	} else {
		if o.Pair.Quote.IsFiatCurrency() &&
			o.Pair.Quote == Bot.Config.Currency.FiatDisplayCurrency {
			log.Infof(log.OrderBook,
				"%s %s: ORDERBOOK: %s %s:  Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s Liquidity Ratio: %f\n",
				exch,
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
				bidsValue/asksValue)
		} else {
			log.Infof(log.OrderBook,
				"%s %s: ORDERBOOK: %s %s: Bids len: %d Amount: %f %s. Total value: %f Asks len: %d Amount: %f %s. Total value: %f Liquidity Ratio: %f\n",
				exch,
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
				bidsValue/asksValue)
		}
	}
}

func printAccountSummary(ai *account.Holdings, protocol string) {
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
				"%s %s: ACCOUNT UPDATE: AccountID:%s Currency:%s Total Amount:%f Hold:%f Free:%f",
				ai.Exchange,
				protocol,
				account,
				ai.Accounts[x].Currencies[y].CurrencyName,
				ai.Accounts[x].Currencies[y].TotalValue,
				ai.Accounts[x].Currencies[y].Hold,
				ai.Accounts[x].Currencies[y].TotalValue-ai.Accounts[x].Currencies[y].Hold)
		}
	}
}

func printTradeSummary(t []order.TradeHistory, protocol string) {
	if len(t) != 0 { // Temp get most recent
		log.Infof(log.Global,
			"%s %s: TRADE: Pair:%s Asset:%s Price:%f Amount:%f TradeID:%s Executed @ %s",
			t[0].Exchange,
			protocol,
			t[0].Pair,
			t[0].AssetType,
			t[0].Price,
			t[0].Amount,
			t[0].TID,
			t[0].Timestamp.Local().Format(time.RFC822))
	}
}

func printOrderSummary(o []order.Detail, protocol string) {
	for i := range o {
		log.Infof(log.Global,
			"%s %s: ORDER: Pair:%s Asset:%s AccountID:%s Pair:%s Asset:%s Price:%f Amount:%f Status:%s Side:%s Type:%s OrderID:%s",
			o[i].Exchange,
			protocol,
			o[i].Pair,
			o[i].AssetType,
			o[i].AccountID,
			o[i].Pair,
			o[i].AssetType,
			o[i].Price,
			o[i].Amount,
			o[i].Status,
			o[i].Side,
			o[i].Type,
			o[i].ID)
	}
}

func printKlineSummary(k *kline.Item, protocol string) {
	for i := range k.Candles {
		log.Infof(log.Global,
			"%s %s: KLINE: Pair:%s Asset:%s Open:%.2f High:%.2f Low:%.2f Close:%.2f Volume:%.2f StartTime: [%s]",
			k.Exchange,
			protocol,
			k.Pair,
			k.Asset,
			k.Candles[i].Open,
			k.Candles[i].High,
			k.Candles[i].Low,
			k.Candles[i].Close,
			k.Candles[i].Volume,
			k.Candles[i].Time)
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
					go WebsocketDataReceiver(ws)

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

// WebsocketDataReceiver handles websocket data coming from a websocket feed
// associated with an exchange
func WebsocketDataReceiver(ws *wshandler.Websocket) {
	wg.Add(1)
	defer wg.Done()

	exchangeName := ws.GetName()
	for {
		select {
		case <-shutdowner:
			return
		case data := <-ws.DataHandler:
			err := WebsocketDataHandler(exchangeName, data)
			if err != nil {
				log.Error(log.WebsocketMgr, err)
			}
		}
	}
}

// WebsocketDataHandler is a central point for exchange websocket implementations to send
// processed data. WebsocketDataHandler will then pass that to an appropriate handler
func WebsocketDataHandler(exchName string, data interface{}) error {
	if data == nil {
		return fmt.Errorf("routines.go - exchange %s nil data sent to websocket",
			exchName)
	}

	switch d := data.(type) {
	case string:
		log.Info(log.WebsocketMgr, d)
	case error:
		return fmt.Errorf("routines.go exchange %s websocket error - %s", exchName, data)
	case *orderbook.Base, order.TradeHistory:
		Bot.SyncManager.StreamUpdate(d)
	case wshandler.FundingData:
		if Bot.Settings.Verbose {
			log.Infof(log.WebsocketMgr, "%s websocket %s %s funding updated %+v",
				exchName,
				FormatCurrency(d.CurrencyPair),
				d.AssetType,
				d)
		}
	case *ticker.Price:
		err := ticker.ProcessTicker(exchName, d, d.AssetType)
		if err != nil {
			fmt.Println(err)
		}
		Bot.SyncManager.StreamUpdate(d)
	case *kline.Item:
		printKlineSummary(d, Websocket)
	case *order.Detail:
		if !Bot.OrderManager.orderStore.exists(d) {
			err := Bot.OrderManager.orderStore.Add(d)
			if err != nil {
				return err
			}
		} else {
			od, err := Bot.OrderManager.orderStore.GetByExchangeAndID(d.Exchange, d.ID)
			if err != nil {
				return err
			}
			od.UpdateOrderFromDetail(d)
		}
	case *order.Cancel:
		return Bot.OrderManager.Cancel(d)
	case *order.Modify:
		od, err := Bot.OrderManager.orderStore.GetByExchangeAndID(d.Exchange, d.ID)
		if err != nil {
			return err
		}
		od.UpdateOrderFromModify(d)
	case order.ClassificationError:
		return errors.New(d.Error())
	case wshandler.UnhandledMessageWarning:
		log.Warn(log.WebsocketMgr, d.Message)
	default:
		if Bot.Settings.Verbose {
			log.Warnf(log.WebsocketMgr,
				"%s websocket Unknown type: %+v",
				exchName,
				d)
		}
	}
	return nil
}
