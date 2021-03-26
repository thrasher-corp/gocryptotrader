package syncer

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems/apiserver"
)

func printCurrencyFormat(price float64, displayCurrency currency.Code) string {
	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to get display symbol: %s\n", err)
	}

	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origCurrency currency.Code, origPrice float64, displayCurrency currency.Code) string {
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

func printTickerSummary(result *ticker.Price, protocol string, err error) {
	if err != nil {
		if err == common.ErrNotYetImplemented {
			log.Warnf(log.Ticker, "Failed to get %s ticker. Error: %s\n",
				protocol,
				err)
			return
		}
		log.Errorf(log.Ticker, "Failed to get %s ticker. Error: %s\n",
			protocol,
			err)
		return
	}

	stats.Add(result.ExchangeName, result.Pair, result.AssetType, result.Last, result.Volume)
	if result.Pair.Quote.IsFiatCurrency() &&
		engine.Bot != nil &&
		result.Pair.Quote != engine.Bot.Config.Currency.FiatDisplayCurrency {
		origCurrency := result.Pair.Quote.Upper()
		log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
			result.ExchangeName,
			protocol,
			engine.Bot.FormatCurrency(result.Pair),
			strings.ToUpper(result.AssetType.String()),
			printConvertCurrencyFormat(origCurrency, result.Last, engine.Bot.Config.Currency.FiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Ask, engine.Bot.Config.Currency.FiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Bid, engine.Bot.Config.Currency.FiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.High, engine.Bot.Config.Currency.FiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Low, engine.Bot.Config.Currency.FiatDisplayCurrency),
			result.Volume)
	} else {
		if result.Pair.Quote.IsFiatCurrency() &&
			engine.Bot != nil &&
			result.Pair.Quote == engine.Bot.Config.Currency.FiatDisplayCurrency {
			log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
				result.ExchangeName,
				protocol,
				engine.Bot.FormatCurrency(result.Pair),
				strings.ToUpper(result.AssetType.String()),
				printCurrencyFormat(result.Last, engine.Bot.Config.Currency.FiatDisplayCurrency),
				printCurrencyFormat(result.Ask, engine.Bot.Config.Currency.FiatDisplayCurrency),
				printCurrencyFormat(result.Bid, engine.Bot.Config.Currency.FiatDisplayCurrency),
				printCurrencyFormat(result.High, engine.Bot.Config.Currency.FiatDisplayCurrency),
				printCurrencyFormat(result.Low, engine.Bot.Config.Currency.FiatDisplayCurrency),
				result.Volume)
		} else {
			log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f\n",
				result.ExchangeName,
				protocol,
				engine.Bot.FormatCurrency(result.Pair),
				strings.ToUpper(result.AssetType.String()),
				result.Last,
				result.Ask,
				result.Bid,
				result.High,
				result.Low,
				result.Volume)
		}
	}
}

const (
	book = "%s %s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s\n"
)

func printOrderbookSummary(result *orderbook.Base, protocol string, bot *engine.Engine, err error) {
	if err != nil {
		if result == nil {
			log.Errorf(log.OrderBook, "Failed to get %s orderbook. Error: %s\n",
				protocol,
				err)
			return
		}
		if err == common.ErrNotYetImplemented {
			log.Warnf(log.OrderBook, "Failed to get %s orderbook for %s %s %s. Error: %s\n",
				protocol,
				result.ExchangeName,
				result.Pair,
				result.AssetType,
				err)
			return
		}
		log.Errorf(log.OrderBook, "Failed to get %s orderbook for %s %s %s. Error: %s\n",
			protocol,
			result.ExchangeName,
			result.Pair,
			result.AssetType,
			err)
		return
	}

	bidsAmount, bidsValue := result.TotalBidsAmount()
	asksAmount, asksValue := result.TotalAsksAmount()

	var bidValueResult, askValueResult string
	switch {
	case result.Pair.Quote.IsFiatCurrency() && bot != nil && result.Pair.Quote != bot.Config.Currency.FiatDisplayCurrency:
		origCurrency := result.Pair.Quote.Upper()
		bidValueResult = printConvertCurrencyFormat(origCurrency, bidsValue, bot.Config.Currency.FiatDisplayCurrency)
		askValueResult = printConvertCurrencyFormat(origCurrency, asksValue, bot.Config.Currency.FiatDisplayCurrency)
	case result.Pair.Quote.IsFiatCurrency() && bot != nil && result.Pair.Quote == bot.Config.Currency.FiatDisplayCurrency:
		bidValueResult = printCurrencyFormat(bidsValue, bot.Config.Currency.FiatDisplayCurrency)
		askValueResult = printCurrencyFormat(asksValue, bot.Config.Currency.FiatDisplayCurrency)
	default:
		bidValueResult = strconv.FormatFloat(bidsValue, 'f', -1, 64)
		askValueResult = strconv.FormatFloat(asksValue, 'f', -1, 64)
	}

	log.Infof(log.OrderBook, book,
		result.ExchangeName,
		protocol,
		bot.FormatCurrency(result.Pair),
		strings.ToUpper(result.AssetType.String()),
		len(result.Bids),
		bidsAmount,
		result.Pair.Base,
		bidValueResult,
		len(result.Asks),
		asksAmount,
		result.Pair.Base,
		askValueResult,
	)
}

func relayWebsocketEvent(result interface{}, event, assetType, exchangeName string) {
	evt := apiserver.WebsocketEvent{
		Data:      result,
		Event:     event,
		AssetType: assetType,
		Exchange:  exchangeName,
	}
	err := apiserver.BroadcastWebsocketMessage(evt)
	if err != nil {
		log.Errorf(log.WebsocketMgr, "Failed to broadcast websocket event %v. Error: %s\n",
			event, err)
	}
}

var shutdowner = make(chan struct{}, 1)
var wg sync.WaitGroup
