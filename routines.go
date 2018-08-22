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

// Routines wraps Updater types for specifically shutting down monitored
// routines
type Routines struct {
	Updaters []Updater
	sync.WaitGroup
}

// Shutdown shuts down all routines
func (r *Routines) Shutdown() {
	r.Add(len(r.Updaters))
	for i := range r.Updaters {
		r.Updaters[i].ShutdownRoutines(r.WaitGroup)
	}
	r.Wait()
}

// Updater is an individual that updates and monitors individual currency assets
// associated with an exchange
type Updater struct {
	Exch                 exchange.IBotExchange
	AssetType            string
	CurrencyPair         pair.CurrencyPair
	KeepUpdatedTicker    bool
	KeepUpdatedOrderbook bool
	SeedDBTradeHistory   bool

	TickerLast    ticker.Price
	OrderbookLast orderbook.Base
	HistoricLast  []exchange.TradeHistory

	RESTPollingDelay time.Duration

	Shutdown  chan struct{}
	Ticker    chan struct{}
	Orderbook chan struct{}
	History   chan struct{}
	Finish    chan struct{}

	sync.WaitGroup
}

// StartUpdater intialises
func StartUpdater(updateTicker, updateOrderbook, UpdateHistory bool) *Routines {
	var routines Routines
	for _, exch := range bot.exchanges {
		enabledAssetTypes, err := exchange.GetExchangeAssetTypes(exch.GetName())
		if err != nil {
			log.Fatal(err)
		}

		for _, enabledAssetType := range enabledAssetTypes {
			for _, enabledCurrencyPair := range exch.GetEnabledCurrencies() {
				updater := Updater{
					Exch:                 exch,
					AssetType:            enabledAssetType,
					CurrencyPair:         enabledCurrencyPair,
					KeepUpdatedTicker:    updateTicker,
					KeepUpdatedOrderbook: updateOrderbook,
					SeedDBTradeHistory:   UpdateHistory,
					Finish:               make(chan struct{}, 1),
					Shutdown:             make(chan struct{}, 1),
					RESTPollingDelay:     1 * time.Second, // Test time
				}
				updater.Run()
				routines.Updaters = append(routines.Updaters, updater)
			}
		}
	}
	return &routines
}

// Run runs the updater routines
func (u *Updater) Run() error {
	if u.KeepUpdatedTicker {
		u.Ticker = make(chan struct{}, 1)
		go u.TickerHandler()
	}

	if u.KeepUpdatedOrderbook {
		u.Orderbook = make(chan struct{}, 1)
		go u.OrderbookHandler()
	}

	if u.SeedDBTradeHistory {
		u.History = make(chan struct{}, 1)
		go u.HistoryHandler()
	}

	if !u.KeepUpdatedTicker && !u.KeepUpdatedOrderbook && !u.SeedDBTradeHistory {
		return errors.New("no updating enabled")
	}

	go u.PollingDelay()
	return nil
}

// PollingDelay syncs updater routines to the exchange PollingDelay
func (u *Updater) PollingDelay() {
	for {
		if !u.KeepUpdatedTicker && !u.KeepUpdatedOrderbook && !u.SeedDBTradeHistory {
			return
		}
		if u.KeepUpdatedTicker {
			u.Ticker <- struct{}{}
			<-u.Finish
			time.Sleep(u.RESTPollingDelay)
		}
		if u.KeepUpdatedOrderbook {
			u.Orderbook <- struct{}{}
			<-u.Finish
			time.Sleep(u.RESTPollingDelay)
		}
		if u.SeedDBTradeHistory {
			u.History <- struct{}{}
			<-u.Finish
			time.Sleep(u.RESTPollingDelay)
		}
	}
}

// TickerHandler handles the rest ticker routine
func (u *Updater) TickerHandler() {
	log.Printf("Ticker REST handler started for %s %s %s",
		u.Exch.GetName(),
		u.CurrencyPair.Pair().String(),
		u.AssetType)

	u.Add(1)

	defer func() {
		u.KeepUpdatedTicker = false
		u.Done()
	}()

	for {
		select {
		case <-u.Shutdown:
			log.Printf("%s Ticker Updater Routine shutting down", u.Exch.GetName())
			return
		case <-u.Ticker:
			err := u.ProcessTickerREST()
			if err != nil {
				switch err.Error() {
				default:
					if common.StringContains(err.Error(), "connection reset by peer") {
						log.Printf("%s for %s and %s asset type - connection reset by peer retrying....",
							u.Exch.GetName(),
							u.CurrencyPair.Pair().String(),
							u.AssetType)
					} else if common.StringContains(err.Error(), "net/http: request canceled") {
						log.Printf("%s for %s and %s asset type - connection request canceled retrying....",
							u.Exch.GetName(),
							u.CurrencyPair.Pair().String(),
							u.AssetType)
					} else if common.StringContains(err.Error(), "ValidateData() error") {
						if common.StringContains(err.Error(), "insufficient returned data") {
							log.Printf("%s for %s and %s asset type %s disabling routine",
								u.Exch.GetName(),
								u.CurrencyPair.Pair().String(),
								u.AssetType,
								err.Error())
							return
						}
						log.Printf("%s for %s and %s asset type %s continuing",
							u.Exch.GetName(),
							u.CurrencyPair.Pair().String(),
							u.AssetType,
							err.Error())
					} else {
						log.Printf("%s Ticker Updater Routine error %s disabling ticker fetcher routine",
							u.Exch.GetName(),
							err.Error())
						return
					}
				}
			}
			// NOTE ADD case <-WebsocketEnabled:
		}
	}
}

// ProcessTickerREST processes tickers using an exchange REST interface
func (u *Updater) ProcessTickerREST() error {
	result, err := u.Exch.UpdateTicker(u.CurrencyPair, u.AssetType)
	u.Finish <- struct{}{}
	printTickerSummary(result, u.CurrencyPair, u.AssetType, u.Exch.GetName(), err)
	if err != nil {
		return err
	}

	err = u.ValidateData(result)
	if err != nil {
		return err
	}

	bot.comms.StageTickerData(u.Exch.GetName(), u.AssetType, result)
	if bot.config.Webserver.Enabled {
		relayWebsocketEvent(result, "ticker_update", u.AssetType, u.Exch.GetName())
	}
	return nil
}

// OrderbookHandler handles the orderbook routine
func (u *Updater) OrderbookHandler() {
	log.Printf("Orderbook REST handler started for %s %s %s",
		u.Exch.GetName(),
		u.CurrencyPair.Pair().String(),
		u.AssetType)

	u.Add(1)

	defer func() {
		u.Done()
		u.KeepUpdatedOrderbook = false
	}()

	for {
		select {
		case <-u.Shutdown:
			log.Printf("%s Orderbook Updater Routine shutting down", u.Exch.GetName())
			return
		case <-u.Orderbook:
			err := u.ProcessOrderbookREST()
			if err != nil {
				switch err.Error() {
				default:
					if common.StringContains(err.Error(), "connection reset by peer") {
						log.Printf("%s for %s and %s asset type - connection reset by peer retrying....",
							u.Exch.GetName(),
							u.CurrencyPair.Pair().String(),
							u.AssetType)
					} else if common.StringContains(err.Error(), "net/http: request canceled") {
						log.Printf("%s for %s and %s asset type - connection request canceled retrying....",
							u.Exch.GetName(),
							u.CurrencyPair.Pair().String(),
							u.AssetType)
					} else if common.StringContains(err.Error(), "ValidateData() error") {
						if common.StringContains(err.Error(), "insufficient returned data") {
							log.Printf("%s for %s and %s asset type %s disabling routine",
								u.Exch.GetName(),
								u.CurrencyPair.Pair().String(),
								u.AssetType,
								err.Error())
							return
						}
						log.Printf("%s for %s and %s asset type - %s continuing",
							u.Exch.GetName(),
							u.CurrencyPair.Pair().String(),
							u.AssetType,
							err.Error())
					} else {
						log.Printf("%s Orderbook Updater Routine error %s disabling orderbook fetcher routine",
							u.Exch.GetName(),
							err.Error())
						return
					}
				}
			}
			// NOTE ADD case <-WebsocketEnabled:
		}
	}
}

// ProcessOrderbookREST processes REST orderbook fetching
func (u *Updater) ProcessOrderbookREST() error {
	result, err := u.Exch.UpdateOrderbook(u.CurrencyPair, u.AssetType)
	u.Finish <- struct{}{}
	printOrderbookSummary(result, u.CurrencyPair, u.AssetType, u.Exch.GetName(), err)
	if err != nil {
		return err
	}

	err = u.ValidateData(result)
	if err != nil {
		return err
	}

	bot.comms.StageOrderbookData(u.Exch.GetName(), u.AssetType, result)

	if bot.config.Webserver.Enabled {
		relayWebsocketEvent(result, "orderbook_update", u.AssetType, u.Exch.GetName())
	}
	return nil
}

// IsOrderbookChanged checks to see if the orderbook has deviated since last
// update
func (u *Updater) IsOrderbookChanged(new orderbook.Base) bool {
	if !u.OrderbookLast.LastUpdated.Equal(new.LastUpdated) {
		return false
	}

	if len(u.OrderbookLast.Asks) != len(new.Asks) || len(u.OrderbookLast.Bids) != len(new.Bids) {
		return false
	}

	for i, data := range u.OrderbookLast.Asks {
		if data != new.Asks[i] {
			return false
		}
	}

	for i, data := range u.OrderbookLast.Bids {
		if data != new.Bids[i] {
			return false
		}
	}
	return true
}

// HistoryHandler handles keeping currency pair/asset types up to date
func (u *Updater) HistoryHandler() {
	log.Printf("History REST handler started for %s %s %s",
		u.Exch.GetName(),
		u.CurrencyPair.Pair().String(),
		u.AssetType)

	u.Add(1)

	defer func() {
		u.Done()
		u.SeedDBTradeHistory = false
	}()

	for {
		select {
		case <-u.Shutdown:
			log.Printf("%s History Updater Routine shutting down", u.Exch.GetName())
			return
		case <-u.History:
			err := u.ProcessHistoryREST()
			if err != nil {
				switch err.Error() {
				case "history up to date":
					log.Printf("%s history is up to date for %s as %s asset type, sleeping for 5 mins",
						u.Exch.GetName(),
						u.CurrencyPair.Pair().String(),
						u.AssetType)
					time.Sleep(5 * time.Minute)
				case "no history returned":
					log.Printf("warning %s no history has been returned for for %s as %s asset type, disabling fetcher routine",
						u.Exch.GetName(),
						u.CurrencyPair.Pair().String(),
						u.AssetType)
					return
				case "trade history not yet implemented":
					log.Printf("%s exchange GetExchangeHistory function not enabled, disabling fetcher routine for %s as %s asset type",
						u.Exch.GetName(),
						u.CurrencyPair.Pair().String(),
						u.AssetType)
					return
				default:
					if common.StringContains(err.Error(), "net/http: request canceled") {
						log.Printf("%s exchange error for %s as %s asset type - net/http: request canceled, retrying",
							u.Exch.GetName(),
							u.CurrencyPair.Pair().String(),
							u.AssetType)
					} else {
						log.Printf("%s exchange error for %s as %s asset type - %s, disabling fetcher routine",
							u.Exch.GetName(),
							u.CurrencyPair.Pair().String(),
							u.AssetType,
							err.Error())
						return
					}
				}
			}
			// NOTE ADD case <-WebsocketEnabled:
		}
	}
}

// ProcessHistoryREST fetches historic values and inserts them into the database
// using the exchange REST interface
func (u *Updater) ProcessHistoryREST() error {
	lastTime, tradeID, err := bot.db.GetExchangeTradeHistoryLast(u.Exch.GetName(),
		u.CurrencyPair.Pair().String())
	u.Finish <- struct{}{}
	if err != nil {
		log.Fatal(err)
	}

	if time.Now().Truncate(5*time.Minute).Unix() < lastTime.Unix() {
		return errors.New("history up to date")
	}

	result, err := u.Exch.GetExchangeHistory(u.CurrencyPair,
		u.AssetType,
		lastTime,
		tradeID)

	if err != nil {
		return err
	}

	if len(result) < 1 {
		return errors.New("no history returned")
	}

	for i := range result {
		err := bot.db.InsertExchangeTradeHistoryData(result[i].TID,
			result[i].Exchange,
			u.CurrencyPair.Pair().String(),
			u.AssetType,
			result[i].Type,
			result[i].Amount,
			result[i].Price,
			result[i].Timestamp)
		if err != nil {
			if err.Error() == "row already found" {
				continue
			}
			log.Fatal(err)
		}
	}
	return nil
}

// ValidateData validates incoming data for the REST interfaces
func (u *Updater) ValidateData(i interface{}) error {
	switch i.(type) {
	case ticker.Price:
		if i.(ticker.Price).Ask == 0 && i.(ticker.Price).Bid == 0 &&
			i.(ticker.Price).High == 0 && i.(ticker.Price).Last == 0 &&
			i.(ticker.Price).Low == 0 && i.(ticker.Price).PriceATH == 0 &&
			i.(ticker.Price).Volume == 0 {
			return errors.New("ValidateData() error, insufficient returned data")
		}
		if i.(ticker.Price).Volume == 0 {
			return errors.New("ValidateData() error, Volume assignment error")
		}
		if i.(ticker.Price).Last == 0 {
			return errors.New("ValidateData() error, critcal value: Last Price assignment error")
		}
		if u.CurrencyPair.Pair().String() != i.(ticker.Price).CurrencyPair ||
			i.(ticker.Price).Pair != u.CurrencyPair {
			return errors.New("ValidateData() error, currency pair mismatch")
		}
		if u.TickerLast != (ticker.Price{}) {
			if u.TickerLast == i.(ticker.Price) {
				return errors.New("ValidateData() error, last ticker unchanged")
			}
		}
		u.TickerLast = i.(ticker.Price)
		return nil
	case orderbook.Base:
		if len(i.(orderbook.Base).Asks) == 0 || len(i.(orderbook.Base).Bids) == 0 {
			return errors.New("ValidateData() error, insufficient returned data")
		}
		if len(u.OrderbookLast.Asks) != 0 && len(u.OrderbookLast.Bids) != 0 {
			if !u.IsOrderbookChanged(i.(orderbook.Base)) {
				return errors.New("ValidateData() error, last orderbook unchanged")
			}
		}
		u.OrderbookLast = i.(orderbook.Base)
		return nil
	}
	return errors.New("insufficient data type")
}

// ShutdownRoutines shutsdown all routines attached to the Updater type
func (u *Updater) ShutdownRoutines(wg sync.WaitGroup) {
	log.Println("Shutdown Routine called for", u.Exch.GetName())
	var routineCount int
	if u.KeepUpdatedTicker {
		u.KeepUpdatedTicker = false
		routineCount++
	}
	if u.KeepUpdatedOrderbook {
		u.KeepUpdatedOrderbook = false
		routineCount++
	}
	if u.SeedDBTradeHistory {
		u.KeepUpdatedOrderbook = false
		routineCount++
	}
	for i := 0; i < routineCount; i++ {
		u.Shutdown <- struct{}{}
	}
	u.Wait()
	wg.Done()
}

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

// HistoricExchangeDataUpdaterRoutine creates routines for getting historic
// price action from an enabled exchange
func HistoricExchangeDataUpdaterRoutine() {
	log.Println("Exchange history updater routine started")
	for _, exch := range bot.exchanges {
		enabledAssetTypes, err := exchange.GetExchangeAssetTypes(exch.GetName())
		if err != nil {
			log.Fatal(err)
		}
		for _, enabledAssetType := range enabledAssetTypes {
			for _, enabledCurrencyPair := range exch.GetEnabledCurrencies() {
				go Processor(exch, enabledCurrencyPair, enabledAssetType)
			}
		}
	}
}

// Processor is a routine handler for each individual currency pair associated
// asset class which will keep it updated either as new updates get pushed or
// via a polling approach.
func Processor(exch exchange.IBotExchange, currencyPair pair.CurrencyPair, assetType string) {
	// This is the initial fallback REST service
	tick := NewUpdaterTicker(assetType, currencyPair)
	for {
		err := processHistory(exch, currencyPair, assetType)
		if err != nil {
			switch err.Error() {
			case "history up to date":
				log.Printf("%s history is up to date for %s as %s asset type, sleeping for 5 mins",
					exch.GetName(),
					currencyPair.Pair().String(),
					assetType)
				time.Sleep(5 * time.Minute)
			case "no history returned":
				log.Printf("warning %s no history has been returned for for %s as %s asset type, disabling fetcher routine",
					exch.GetName(),
					currencyPair.Pair().String(),
					assetType)
				return
			case "trade history not yet implemented":
				log.Printf("%s exchange GetExchangeHistory function not enabled, disabling fetcher routine for %s as %s asset type",
					exch.GetName(),
					currencyPair.Pair().String(),
					assetType)
				return
			default:
				if common.StringContains(err.Error(), "net/http: request canceled") {
					log.Printf("%s exchange error for %s as %s asset type - net/http: request canceled, retrying",
						exch.GetName(),
						currencyPair.Pair().String(),
						assetType)
				} else {
					log.Printf("%s exchange error for %s as %s asset type - %s, disabling fetcher routine",
						exch.GetName(),
						currencyPair.Pair().String(),
						assetType,
						err.Error())
					return
				}
			}
		}
		<-tick.C
	}
}

// processHistory fetches historic values and inserts them into the database
func processHistory(exch exchange.IBotExchange, c pair.CurrencyPair, assetType string) error {
	lastTime, tradeID, err := bot.db.GetExchangeTradeHistoryLast(exch.GetName(), c.Pair().String())
	if err != nil {
		log.Fatal(err)
	}

	if time.Now().Truncate(5*time.Minute).Unix() < lastTime.Unix() {
		return errors.New("history up to date")
	}

	result, err := exch.GetExchangeHistory(c, assetType, lastTime, tradeID)
	if err != nil {
		return err
	}

	if len(result) < 1 {
		return errors.New("no history returned")
	}

	for i := range result {
		err := bot.db.InsertExchangeTradeHistoryData(result[i].TID,
			result[i].Exchange,
			c.Pair().String(),
			assetType,
			result[i].Type,
			result[i].Amount,
			result[i].Price,
			result[i].Timestamp)
		if err != nil {
			if err.Error() == "row already found" {
				continue
			}
			log.Fatal(err)
		}
	}
	return nil
}

// NewUpdaterTicker returns a time.Ticker to keep individual currency pairs
// updated NOTE will be updated with tailored time for each exchange.
func NewUpdaterTicker(assetType string, currencyPair pair.CurrencyPair) *time.Ticker {
	return time.NewTicker(10 * time.Second)
}
