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
