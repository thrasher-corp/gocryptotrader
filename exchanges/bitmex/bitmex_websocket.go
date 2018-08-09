package bitmex

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

const (
	bitmexWSURL = "wss://www.bitmex.com/realtime"

	// Public Subscription Channels
	bitmexWSAnnouncement        = "announcement"
	bitmexWSChat                = "chat"
	bitmexWSConnected           = "connected"
	bitmexWSFunding             = "funding"
	bitmexWSInstrument          = "instrument"
	bitmexWSInsurance           = "insurance"
	bitmexWSLiquidation         = "liquidation"
	bitmexWSOrderbookL2         = "orderBookL2"
	bitmexWSOrderbookL10        = "orderBook10"
	bitmexWSPublicNotifications = "publicNotifications"
	bitmexWSQuote               = "quote"
	bitmexWSQuote1m             = "quoteBin1m"
	bitmexWSQuote5m             = "quoteBin5m"
	bitmexWSQuote1h             = "quoteBin1h"
	bitmexWSQuote1d             = "quoteBin1d"
	bitmexWSSettlement          = "settlement"
	bitmexWSTrade               = "trade"
	bitmexWSTrade1m             = "tradeBin1m"
	bitmexWSTrade5m             = "tradeBin5m"
	bitmexWSTrade1h             = "tradeBin1h"
	bitmexWSTrade1d             = "tradeBin1d"

	// Authenticated Subscription Channels
	bitmexWSAffiliate            = "affiliate"
	bitmexWSExecution            = "execution"
	bitmexWSOrder                = "order"
	bitmexWSMargin               = "margin"
	bitmexWSPosition             = "position"
	bitmexWSPrivateNotifications = "privateNotifications"
	bitmexWSTransact             = "transact"
	bitmexWSWallet               = "wallet"

	bitmexActionInitialData = "partial"
	bitmexActionInsertData  = "insert"
	bitmexActionDeleteData  = "delete"
	bitmexActionUpdateData  = "update"
)

var (
	pongChan = make(chan int, 1)
	timer    *time.Timer
)

// WebsocketConnect initiates a new websocket connection
func (b *Bitmex) WebsocketConnect() error {
	var dialer websocket.Dialer
	var err error

	b.WebsocketConn, _, err = dialer.Dial(bitmexWSURL, nil)
	if err != nil {
		return err
	}

	_, p, err := b.WebsocketConn.ReadMessage()
	if err != nil {
		return err
	}

	var welcomeResp WebsocketWelcome
	err = common.JSONDecode(p, &welcomeResp)
	if err != nil {
		return err
	}

	go b.connectionHandler()

	if b.Verbose {
		log.Printf("Successfully connected to Bitmex %s at time: %s Limit: %d",
			welcomeResp.Info,
			welcomeResp.Timestamp,
			welcomeResp.Limit.Remaining)
	}

	go b.handleIncomingData()

	err = b.websocketSubscribe()
	if err != nil {
		b.WebsocketConn.Close()
		return err
	}

	if b.AuthenticatedAPISupport {
		err := b.websocketSendAuth()
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

// Timer handles connection loss or failure
func (b *Bitmex) connectionHandler() {
	defer func() {
		if b.Verbose {
			log.Println("Bitmex websocket: Connection handler routine shutdown")
		}
	}()

	shutdown := b.shutdown.addRoutine()

	timer = time.NewTimer(5 * time.Second)
	for {
		select {
		case <-timer.C:
			timeout := time.After(5 * time.Second)
			err := b.WebsocketConn.WriteJSON("ping")
			if err != nil {
				b.reconnect()
				return
			}
			for {
				select {
				case <-pongChan:
					if b.Verbose {
						log.Println("Bitmex websocket: PONG received")
					}
					break
				case <-timeout:
					log.Println("Bitmex websocket: Connection timed out - Closing connection....")
					b.WebsocketConn.Close()

					log.Println("Bitmex websocket: Connection timed out - Reconnecting...")
					b.reconnect()
					return
				}
			}
		case <-shutdown:
			log.Println("Bitmex websocket: shutdown requested - Closing connection....")
			b.WebsocketConn.Close()
			log.Println("Bitmex websocket: Sending shutdown message")
			b.shutdown.routineShutdown()
			return
		}
	}
}

// Reconnect handles reconnections to websocket API
func (b *Bitmex) reconnect() {
	for {
		err := b.WebsocketConnect()
		if err != nil {
			log.Println("Bitmex websocket: Connection timed out - Failed to connect, sleeping...")
			time.Sleep(time.Second * 2)
			continue
		}
		return
	}
}

// handleIncomingData services incoming data from the websocket connection
func (b *Bitmex) handleIncomingData() {
	defer func() {
		if b.Verbose {
			log.Println("Bitmex websocket: Response data handler routine shutdown")
		}
	}()

	for {
		_, resp, err := b.WebsocketConn.ReadMessage()
		if err != nil {
			if b.Verbose {
				log.Println("Bitmex websocket: Connection error", err)
			}
			return
		}

		message := string(resp)
		if common.StringContains(message, "pong") {
			if b.Verbose {
				log.Println("Bitmex websocket: PONG receieved")
			}
			pongChan <- 1
			continue
		}

		if common.StringContains(message, "ping") {
			err = b.WebsocketConn.WriteJSON("pong")
			if err != nil {
				if b.Verbose {
					log.Println("Bitmex websocket error: ", err)
				}
				return
			}
		}

		if !timer.Reset(5 * time.Second) {
			log.Fatal("Bitmex websocket: Timer failed to set")
		}

		quickCapture := make(map[string]interface{})
		err = common.JSONDecode(resp, &quickCapture)
		if err != nil {
			log.Fatal(err)
		}

		var respError WebsocketErrorResponse
		if _, ok := quickCapture["status"]; ok {
			err = common.JSONDecode(resp, &respError)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Bitmex websocket error: %s", respError.Error)
			continue
		}

		if _, ok := quickCapture["success"]; ok {
			var decodedResp WebsocketSubscribeResp
			err := common.JSONDecode(resp, &decodedResp)
			if err != nil {
				log.Fatal(err)
			}

			if decodedResp.Success {
				if b.Verbose {
					if len(quickCapture) == 3 {
						log.Printf("Bitmex Websocket: Successfully subscribed to %s",
							decodedResp.Subscribe)
					} else {
						log.Println("Bitmex Websocket: Successfully authenticated websocket connection")
					}
				}
				continue
			}
			log.Printf("Bitmex websocket error: Unable to subscribe %s",
				decodedResp.Subscribe)

		} else if _, ok := quickCapture["table"]; ok {
			var decodedResp WebsocketMainResponse
			err := common.JSONDecode(resp, &decodedResp)
			if err != nil {
				log.Fatal(err)
			}

			switch decodedResp.Table {
			case bitmexWSOrderbookL2:
				var orderbooks OrderBookData
				err = common.JSONDecode(resp, &orderbooks)
				if err != nil {
					log.Fatal(err)
				}
				err = b.processOrderbook(orderbooks.Data, orderbooks.Action)
				if err != nil {
					log.Fatal(err)
				}
			case bitmexWSTrade:
				var trades TradeData
				err = common.JSONDecode(resp, &trades)
				if err != nil {
					log.Fatal(err)
				}
				err = b.processTrades(trades.Data, trades.Action)
				if err != nil {
					log.Fatal(err)
				}
			case bitmexWSAnnouncement:
				var announcement AnnouncementData
				err = common.JSONDecode(resp, &announcement)
				if err != nil {
					log.Fatal(err)
				}
				err = b.processAnnouncement(announcement.Data, announcement.Action)
				if err != nil {
					log.Fatal(err)
				}
			default:
				log.Fatal("Bitmex websocket error: Table unknown -", decodedResp.Table)
			}
		}
	}
}

// Temporary local cache of Announcements
var localAnnouncements []Announcement
var partialLoadedAnnouncement bool

// ProcessAnnouncement process announcements
func (b *Bitmex) processAnnouncement(data []Announcement, action string) error {
	switch action {
	case bitmexActionInitialData:
		if !partialLoadedAnnouncement {
			localAnnouncements = data
		}
		partialLoadedAnnouncement = true
	default:
		return fmt.Errorf("Bitmex websocket error: ProcessAnnouncement() unallocated action - %s",
			action)
	}
	return nil
}

// Temporary local cache of orderbooks
var localOb []OrderBookL2
var partialLoaded bool

// ProcessOrderbook processes orderbook updates
func (b *Bitmex) processOrderbook(data []OrderBookL2, action string) error {
	switch action {
	case bitmexActionInitialData:
		if !partialLoaded {
			localOb = data
		}
		partialLoaded = true
	case bitmexActionUpdateData:
		if partialLoaded {
			updated := len(data)
			for _, elem := range data {
				for i := range localOb {
					if localOb[i].ID == elem.ID && localOb[i].Symbol == elem.Symbol {
						localOb[i].Side = elem.Side
						localOb[i].Size = elem.Size
						updated--
						break
					}
				}
			}
			if updated != 0 {
				return errors.New("Bitmex websocket error: Elements not updated correctly")
			}
		}
	case bitmexActionInsertData:
		if partialLoaded {
			updated := len(data)
			for _, elem := range data {
				localOb = append(localOb, OrderBookL2{
					Symbol: elem.Symbol,
					ID:     elem.ID,
					Side:   elem.Side,
					Size:   elem.Size,
					Price:  elem.Price,
				})
				updated--
			}
			if updated != 0 {
				return errors.New("Bitmex websocket error: Elements not updated correctly")
			}
		}
	case bitmexActionDeleteData:
		if partialLoaded {
			updated := len(data)
			for _, elem := range data {
				for i := range localOb {
					if localOb[i].ID == elem.ID && localOb[i].Symbol == elem.Symbol {
						localOb[i] = localOb[len(localOb)-1]
						localOb = localOb[:len(localOb)-1]
						updated--
						break
					}
				}
			}
			if updated != 0 {
				return errors.New("Bitmex websocket error: Elements not updated correctly")
			}
		}
	}
	return nil
}

// Temporary local cache of orderbooks
var localTrades []Trade
var partialLoadedTrades bool

// ProcessTrades processes new trades that have occured
func (b *Bitmex) processTrades(data []Trade, action string) error {
	switch action {
	case bitmexActionInitialData:
		if !partialLoadedTrades {
			localTrades = data
		}
		partialLoadedTrades = true
	case bitmexActionInsertData:
		if partialLoadedTrades {
			localTrades = append(localTrades, data...)
		}
	default:
		return fmt.Errorf("Bitmex websocket error: ProcessTrades() unallocated action - %s",
			action)
	}
	return nil
}

// WebsocketSubscribe subscribes to a websocket channel
func (b *Bitmex) websocketSubscribe() error {
	contracts := b.GetEnabledCurrencies()

	// Subscriber
	var subscriber WebsocketRequest
	subscriber.Command = "subscribe"

	// Announcement subscribe
	subscriber.Arguments = append(subscriber.Arguments, bitmexWSAnnouncement)

	for _, contract := range contracts {
		// Orderbook subscribe
		subscriber.Arguments = append(subscriber.Arguments,
			bitmexWSOrderbookL2+":"+contract.Pair().String())

		// Trade subscribe
		subscriber.Arguments = append(subscriber.Arguments,
			bitmexWSTrade+":"+contract.Pair().String())

		// NOTE more added here in future
	}

	err := b.WebsocketConn.WriteJSON(subscriber)
	if err != nil {
		return err
	}

	return nil
}

// WebsocketSendAuth sends an authenticated subscription
func (b *Bitmex) websocketSendAuth() error {
	timestamp := time.Now().Add(time.Hour * 1).Unix()
	newTimestamp := strconv.FormatInt(timestamp, 10)
	hmac := common.GetHMAC(common.HashSHA256,
		[]byte("GET/realtime"+newTimestamp),
		[]byte(b.APISecret))

	signature := common.HexEncodeToString(hmac)

	var sendAuth WebsocketRequest
	sendAuth.Command = "authKeyExpires"
	sendAuth.Arguments = append(sendAuth.Arguments, b.APIKey)
	sendAuth.Arguments = append(sendAuth.Arguments, timestamp)
	sendAuth.Arguments = append(sendAuth.Arguments, signature)

	return b.WebsocketConn.WriteJSON(sendAuth)
}

// Shutdown to monitor and shut down routines package specific
type Shutdown struct {
	c            chan int
	routineCount int
	finishC      chan int
}

// NewRoutineManagement returns an new initial routine management system
func (b *Bitmex) NewRoutineManagement() *Shutdown {
	return &Shutdown{
		c:       make(chan int, 1),
		finishC: make(chan int, 1),
	}
}

// AddRoutine adds a routine to the monitor and returns a channel
func (r *Shutdown) addRoutine() chan int {
	log.Println("Bitmex Websocket: Routine added to monitor")
	r.routineCount++
	return r.c
}

// RoutineShutdown sends a message to the finisher channel
func (r *Shutdown) routineShutdown() {
	log.Println("Bitmex Websocket: Routine is shutting down")
	r.finishC <- 1
}

// SignalShutdown signals a shutdown across routines
func (r *Shutdown) SignalShutdown() {
	log.Println("Bitmex Websocket: Shutdown signal sending..")
	for i := 0; i < r.routineCount; i++ {
		log.Printf("Bitmex Websocket: Shutdown signal sent to routine %d", i+1)
		r.c <- 1
	}

	for {
		<-r.finishC
		r.routineCount--
		if r.routineCount <= 0 {
			close(r.c)
			close(r.finishC)
			log.Println("Bitmex Websocket: All routines stopped")
			return
		}
	}
}
