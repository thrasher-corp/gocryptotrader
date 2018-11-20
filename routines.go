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

const (
	updateTicker    = "update_ticker"
	updateOrderbook = "update_orderbook"
	updateHistory   = "update_history"

	maxWorkers   = 5
	maxFrameSize = 250 * time.Millisecond
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

// Updater is the main update type monitoring multiple assets
var updater Updater

// Updater defines the Enabled and disabled trading assets
type Updater struct {
	// Assets is the enabled list of tradable assets for an exchange
	Assets    []*TradingAsset
	Websocket WebsocketService

	shutdown chan struct{}      // main shutdown
	pool     chan WorkerRequest // to worker pool

	wg sync.WaitGroup
	sync.Mutex
}

// TradingAsset defines a supported trading asset on an exchange
type TradingAsset struct {
	Enabled      bool
	Exchange     exchange.IBotExchange
	AssetType    string
	CurrencyPair pair.CurrencyPair
	Ticker       TradeData
	Orderbook    TradeData
	History      TradeData
}

// TradeData defines actually request trade data
type TradeData struct {
	Enabled     bool
	Updating    bool // Currently being updated
	LastUpdated time.Time
}

// StartUpdater starts a full monitor for all enabled cryptocurrency asset pairs
func StartUpdater(exchanges []exchange.IBotExchange, ticker, orderbook, history, verbose bool) error {
	log.Println("GoCryptoTrader - asset monitor service started")

	updater = Updater{
		shutdown: make(chan struct{}, 1),
		pool:     make(chan WorkerRequest, 10000),
	}

	for i := range exchanges {
		enabledCurrencies := exchanges[i].GetEnabledCurrencies()
		// TODO - Update asset retrieval through interface
		for _, enabledCurrency := range enabledCurrencies {
			updater.Assets = append(updater.Assets, &TradingAsset{
				Exchange:     exchanges[i],
				AssetType:    "SPOT",
				CurrencyPair: enabledCurrency,
				Ticker:       TradeData{Enabled: ticker},
				Orderbook:    TradeData{Enabled: orderbook},
				History:      TradeData{Enabled: history},
			})
		}
	}

	err := updater.StartWebsocketService(verbose)
	if err != nil {
		return err
	}

	// Adds worker to a worker pool
	for i := 0; i < maxWorkers; i++ {
		go updater.Worker(i, verbose)
	}

	go updater.WorkFramer(verbose)

	return nil
}

// WorkerRequest defines a job to be executed
type WorkerRequest struct {
	Exchange     exchange.IBotExchange
	Update       string
	CurrencyPair pair.CurrencyPair
	Asset        string
}

// WorkFramer frames jobs in 250ms frames to minimise CPU usage in main check
// routine & minimise multiple routines in time.Sleep
func (u *Updater) WorkFramer(verbose bool) {
	if verbose {
		log.Println("routines.go WorkFramer() - started")
	}

	u.wg.Add(1)
	defer u.wg.Done()

	ticker := time.NewTicker(maxFrameSize)

	for {
		select {
		case <-u.shutdown:
			return

		case <-ticker.C:
			u.Lock()
			for i := range u.Assets {
				if u.Assets[i].History.Enabled && !u.Assets[i].History.Updating {
					if u.Assets[i].History.LastUpdated.IsZero() {
						u.Assets[i].History.Updating = true
						u.pool <- WorkerRequest{
							Exchange:     u.Assets[i].Exchange,
							Update:       updateHistory,
							CurrencyPair: u.Assets[i].CurrencyPair,
							Asset:        u.Assets[i].AssetType,
						}
					} else {
						if u.Assets[i].History.LastUpdated.Add(10 * time.Second).Before(time.Now()) {
							u.Assets[i].History.Updating = true
							u.pool <- WorkerRequest{
								Exchange:     u.Assets[i].Exchange,
								Update:       updateHistory,
								CurrencyPair: u.Assets[i].CurrencyPair,
								Asset:        u.Assets[i].AssetType,
							}
						}
					}
				}

				if u.Assets[i].Orderbook.Enabled && !u.Assets[i].Orderbook.Updating {
					if u.Assets[i].Orderbook.LastUpdated.IsZero() {
						u.Assets[i].Orderbook.Updating = true
						u.pool <- WorkerRequest{
							Exchange:     u.Assets[i].Exchange,
							Update:       updateOrderbook,
							CurrencyPair: u.Assets[i].CurrencyPair,
							Asset:        u.Assets[i].AssetType,
						}
					} else {
						if u.Assets[i].Orderbook.LastUpdated.Add(10 * time.Second).Before(time.Now()) {
							u.Assets[i].Orderbook.Updating = true
							u.pool <- WorkerRequest{
								Exchange:     u.Assets[i].Exchange,
								Update:       updateOrderbook,
								CurrencyPair: u.Assets[i].CurrencyPair,
								Asset:        u.Assets[i].AssetType,
							}
						}
					}
				}

				if u.Assets[i].Ticker.Enabled && !u.Assets[i].Ticker.Updating {
					if u.Assets[i].Ticker.LastUpdated.IsZero() {
						u.Assets[i].Ticker.Updating = true
						u.pool <- WorkerRequest{
							Exchange:     u.Assets[i].Exchange,
							Update:       updateTicker,
							CurrencyPair: u.Assets[i].CurrencyPair,
							Asset:        u.Assets[i].AssetType,
						}
					} else {
						if u.Assets[i].Ticker.LastUpdated.Add(10 * time.Second).Before(time.Now()) {
							u.Assets[i].Ticker.Updating = true
							u.pool <- WorkerRequest{
								Exchange:     u.Assets[i].Exchange,
								Update:       updateTicker,
								CurrencyPair: u.Assets[i].CurrencyPair,
								Asset:        u.Assets[i].AssetType,
							}
						}
					}
				}
			}
			u.Unlock()
		}
	}
}

// Worker initiates a request from a worker pool
func (u *Updater) Worker(id int, verbose bool) {
	if verbose {
		log.Printf("routines.go - Worker routine started id: %d", id)
	}

	u.wg.Add(1)
	defer u.wg.Done()

	for {
		select {
		case <-u.shutdown:
			return

		case job := <-u.pool:
			if verbose {
				log.Printf("worker routine ID: %d job recieved for Exchange: %s CurrencyPair: %s Asset: %s Update: %s",
					id,
					job.Exchange.GetName(),
					job.CurrencyPair.Pair().String(),
					job.Asset,
					job.Update)
			}

			switch job.Update {
			case updateTicker:
				err := u.ProcessTickerREST(job.Exchange, job.CurrencyPair, job.Asset, job.Exchange.GetName(), verbose)
				if err != nil {
					switch err.Error() {
					default:
						if common.StringContains(err.Error(), "connection reset by peer") {
							log.Printf("%s for %s and %s asset type - connection reset by peer retrying....",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO retry every 30 seconds
						} else if common.StringContains(err.Error(), "net/http: request canceled") {
							log.Printf("%s for %s and %s asset type - connection request canceled retrying....",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO retry every 30 seconds
						} else if common.StringContains(err.Error(), "ValidateData() error") {
							if common.StringContains(err.Error(), "insufficient returned data") {
								log.Printf("%s for %s and %s asset type %s disabling routine",
									job.Exchange.GetName(),
									job.CurrencyPair.Pair().String(),
									job.Asset,
									err.Error())
								// TODO SERIOUS Error! disable job fetching and log
							}
							log.Printf("%s for %s and %s asset type %s continuing",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset,
								err.Error())
							// TODO SERIOUS Error! disable job fetching and log
						} else {
							log.Printf("%s Ticker Updater Routine error %s disabling ticker fetcher routine",
								job.Exchange.GetName(),
								err.Error())
							// TODO disable job fetching and log
						}
					}
				}

			case updateOrderbook:
				err := u.ProcessOrderbookREST(job.Exchange, job.CurrencyPair, job.Asset, verbose)
				if err != nil {
					switch err.Error() {
					default:
						if common.StringContains(err.Error(), "connection reset by peer") {
							log.Printf("%s for %s and %s asset type - connection reset by peer retrying....",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO retry every 30 seconds
						} else if common.StringContains(err.Error(), "net/http: request canceled") {
							log.Printf("%s for %s and %s asset type - connection request canceled retrying....",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO retry every 30 seconds
						} else if common.StringContains(err.Error(), "ValidateData() error") {
							if common.StringContains(err.Error(), "insufficient returned data") {
								log.Printf("%s for %s and %s asset type %s disabling routine",
									job.Exchange.GetName(),
									job.CurrencyPair.Pair().String(),
									job.Asset,
									err.Error())
								// TODO SERIOUS Error! disable job fetching and log
							}
							log.Printf("%s for %s and %s asset type - %s continuing",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset,
								err.Error())
							// TODO SERIOUS Error! disable job fetching and log
						} else {
							log.Printf("%s Orderbook Updater Routine error %s disabling orderbook fetcher routine",
								job.Exchange.GetName(),
								err.Error())
							// TODO disable job fetching and log
						}
					}
				}

			case updateHistory:
				err := u.ProcessHistoryREST(job.Exchange, job.CurrencyPair, job.Asset, verbose)
				if err != nil {
					switch err.Error() {
					case "history up to date":
						if verbose {
							log.Printf("%s history is up to date for %s as %s asset type, sleeping for 5 mins",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO sleep job 5 minutes
						}

					case "no history returned":
						log.Printf("warning %s no history has been returned for for %s as %s asset type, disabling fetcher routine",
							job.Exchange.GetName(),
							job.CurrencyPair.Pair().String(),
							job.Asset)
					// TODO disable job

					case "trade history not yet implemented":
						log.Printf("%s exchange GetExchangeHistory function not enabled, disabling fetcher routine for %s as %s asset type",
							job.Exchange.GetName(),
							job.CurrencyPair.Pair().String(),
							job.Asset)
						// TODO disable job

					default:
						if common.StringContains(err.Error(), "net/http: request canceled") {
							log.Printf("%s exchange error for %s as %s asset type - net/http: request canceled, retrying",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO retry every 30 seconds
						} else {
							log.Printf("%s exchange error for %s as %s asset type - %s, disabling fetcher routine",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset,
								err.Error())
							// General error end job disable request function
						}
					}
				}
			}

			var updated bool
			u.Lock()
			for i := range u.Assets {
				if u.Assets[i].Exchange.GetName() == job.Exchange.GetName() &&
					u.Assets[i].CurrencyPair == job.CurrencyPair &&
					u.Assets[i].AssetType == job.Asset {
					switch job.Update {
					case updateHistory:
						u.Assets[i].History.LastUpdated = time.Now()
						u.Assets[i].History.Updating = false
					case updateOrderbook:
						u.Assets[i].Orderbook.LastUpdated = time.Now()
						u.Assets[i].Orderbook.Updating = false
					case updateTicker:
						u.Assets[i].Ticker.LastUpdated = time.Now()
						u.Assets[i].Ticker.Updating = false
					}
					updated = true
					break
				}
			}
			u.Unlock()

			if !updated {
				log.Fatalf("routines.go error - worker could not update asset %s, %s, %s, for jon %s",
					job.Exchange.GetName(),
					job.CurrencyPair,
					job.Asset,
					job.Update)
			}
		}
	}
}

// ProcessTickerREST processes tickers data, utilising a REST endpoint
func (u *Updater) ProcessTickerREST(exch exchange.IBotExchange, p pair.CurrencyPair, assetType, exchangeName string, verbose bool) error {
	result, err := exch.UpdateTicker(p, assetType)
	printTickerSummary(result, p, assetType, exchangeName, err)
	if err != nil {
		return err
	}

	err = u.ValidateData(result)
	if err != nil {
		return err
	}

	bot.comms.StageTickerData(exchangeName, assetType, result)
	if bot.config.Webserver.Enabled {
		relayWebsocketEvent(result, "ticker_update", assetType, exchangeName)
	}
	return nil
}

// ProcessOrderbookREST processes orderbook data, utilising a REST endpoint
func (u *Updater) ProcessOrderbookREST(exch exchange.IBotExchange, p pair.CurrencyPair, assetType string, verbose bool) error {
	result, err := exch.UpdateOrderbook(p, assetType)
	printOrderbookSummary(result, p, assetType, exch.GetName(), err)
	if err != nil {
		return err
	}

	err = u.ValidateData(result)
	if err != nil {
		return err
	}

	bot.comms.StageOrderbookData(exch.GetName(), assetType, result)

	if bot.config.Webserver.Enabled {
		relayWebsocketEvent(result, "orderbook_update", assetType, exch.GetName())
	}
	return nil
}

// ProcessHistoryREST processes history data, utilising a REST endpoint
func (u *Updater) ProcessHistoryREST(exch exchange.IBotExchange, p pair.CurrencyPair, assetType string, verbose bool) error {
	lastTime, tradeID, err := bot.db.GetExchangeTradeHistoryLast(exch.GetName(),
		p.Pair().String(),
		assetType)
	if err != nil {
		log.Fatal(err)
	}

	if time.Now().Truncate(5*time.Minute).Unix() < lastTime.Unix() {
		return errors.New("history up to date")
	}

	result, err := exch.GetExchangeHistory(p,
		assetType,
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
			p.Pair().String(),
			assetType,
			result[i].Type,
			result[i].Amount,
			result[i].Price,
			result[i].Timestamp)
		if err != nil {
			if common.StringContains(err.Error(), "UNIQUE constraint failed") {
				continue
			}
			log.Fatal(err)
		}
	}
	return nil
}

// ValidateData validates incoming from either a REST endpoint or websocket feed
// by its type
func (u *Updater) ValidateData(i interface{}) error {
	switch i.(type) {
	case ticker.Price:
		if i.(ticker.Price).Ask == 0 && i.(ticker.Price).Bid == 0 &&
			i.(ticker.Price).High == 0 && i.(ticker.Price).Last == 0 &&
			i.(ticker.Price).Low == 0 && i.(ticker.Price).PriceATH == 0 &&
			i.(ticker.Price).Volume == 0 {
			return errors.New("routines.go ValidateData() error - insufficient returned data")
		}
		if i.(ticker.Price).Volume == 0 {
			return errors.New("routines.go ValidateData() error - volume assignment error")
		}
		if i.(ticker.Price).Last == 0 {
			return errors.New("routines.go ValidateData() error - critcal value: Last Price assignment error")
		}

	case orderbook.Base:
		if len(i.(orderbook.Base).Asks) == 0 || len(i.(orderbook.Base).Bids) == 0 {
			return errors.New("routines.go ValidateData() error - insufficient returned data")
		}

	default:
		return errors.New("routines.go ValidateData() error - can't handle data type")
	}
	return nil
}

// WebsocketService defines the websocket connection suite
type WebsocketService struct {
	Conn []*exchange.Websocket
	sync.Mutex
	sync.WaitGroup
}

// StartWebsocketService connects to the websocket endpoint for each enabled
// exchange
func (u *Updater) StartWebsocketService(verbose bool) error {
	if verbose {
		log.Println("Connecting exchange websocket services...")
	}

	u.Websocket = WebsocketService{}

	for i := range bot.exchanges {
		go func(i int) {
			if verbose {
				log.Printf("Establishing websocket connection for %s",
					bot.exchanges[i].GetName())
			}

			ws, err := bot.exchanges[i].GetWebsocket()
			if err != nil {
				if common.StringContains(err.Error(), "not yet implemented") {
					return
				}
				log.Println("Websocket Error:", err)
				return
			}

			// Data handler routine
			go u.WebsocketDataHandler(ws, verbose)

			err = ws.Connect()
			if err != nil {
				if common.StringContains(err.Error(), "websocket disabled") {
					return
				}
				log.Println("Websocket Error:", err)
				return
			}
			u.Websocket.Lock()
			u.Websocket.Conn = append(u.Websocket.Conn, ws)
			u.Websocket.Unlock()
		}(i)
	}

	return nil
}

// WebsocketIsAlive alerts of full websocket disconnection
func (u *Updater) WebsocketIsAlive(ws *exchange.Websocket, verbose bool) {
	u.Websocket.Add(1)
	defer u.Websocket.Done()

	for {
		select {
		case <-u.shutdown:
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
func (u *Updater) WebsocketDataHandler(ws *exchange.Websocket, verbose bool) {
	u.Websocket.Add(1)
	defer u.Websocket.Done()

	go u.WebsocketIsAlive(ws, verbose)

	for {
		select {
		case <-u.shutdown:
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
					go u.WebsocketReconnect(ws, verbose)
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
				wsTicker := data.(exchange.TickerData)
				if verbose {
					log.Println("Websocket Ticker Updated:   ", wsTicker)
				}

				var updated bool
				u.Lock()
				for i := range u.Assets {
					if u.Assets[i].AssetType == wsTicker.AssetType &&
						u.Assets[i].CurrencyPair.Display("", true) == wsTicker.Pair.Display("", true) &&
						u.Assets[i].Exchange.GetName() == wsTicker.Exchange {
						u.Assets[i].Ticker.LastUpdated = time.Now()
						updated = true
						break
					}
				}
				u.Unlock()

				if !updated {
					log.Printf("routines.go error - websocket failed to update ticker asset %s, %s, %s",
						wsTicker.Exchange,
						wsTicker.Pair,
						wsTicker.AssetType)
				}

			case exchange.KlineData:
				// Kline data
				if verbose {
					log.Println("Websocket Kline Updated:    ", data.(exchange.KlineData))
				}

			case exchange.WebsocketOrderbookUpdate:
				wsOrderbook := data.(exchange.WebsocketOrderbookUpdate)
				if verbose {
					log.Println("Websocket Orderbook Updated:", wsOrderbook)
				}

				var updated bool
				u.Lock()
				for i := range u.Assets {
					if u.Assets[i].AssetType == wsOrderbook.Asset &&
						u.Assets[i].CurrencyPair.Display("", true) == wsOrderbook.Pair.Display("", true) &&
						u.Assets[i].Exchange.GetName() == wsOrderbook.Exchange {
						u.Assets[i].Ticker.LastUpdated = time.Now()
						updated = true
						break
					}
				}
				u.Unlock()

				if !updated {
					log.Printf("routines.go error - websocket failed to update orderbook asset %s, %s, %s",
						wsOrderbook.Exchange,
						wsOrderbook.Pair,
						wsOrderbook.Asset)
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
func (u *Updater) WebsocketReconnect(ws *exchange.Websocket, verbose bool) {
	if verbose {
		log.Printf("Websocket reconnection requested for %s", ws.GetName())
	}

	err := ws.Shutdown()
	if err != nil {
		log.Fatal(err)
	}

	u.Websocket.Add(1)
	defer u.Websocket.Done()

	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-u.shutdown:
			return

		case <-ticker.C:
			err = ws.Connect()
			if err == nil {
				return
			}
		}
	}
}
