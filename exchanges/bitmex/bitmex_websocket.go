package bitmex

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"

	"github.com/thrasher-/gocryptotrader/exchanges"

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
)

// WsConnector initiates a new websocket connection
func (b *Bitmex) WsConnector() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	var err error

	if b.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(b.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	b.WebsocketConn, _, err = dialer.Dial(b.Websocket.GetWebsocketURL(), nil)
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

	if b.Verbose {
		log.Printf("Successfully connected to Bitmex %s at time: %s Limit: %d",
			welcomeResp.Info,
			welcomeResp.Timestamp,
			welcomeResp.Limit.Remaining)
	}

	var c = make(chan []byte, 1)
	go b.wsHandleIncomingData(c)
	go b.wsReadData(c)

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

func (b *Bitmex) wsReadData(c chan []byte) {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		default:
			_, resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

			b.Websocket.TrafficTimer.Reset(exchange.WebsocketTrafficLimitTime)

			c <- resp
		}
	}
}

// wsHandleIncomingData services incoming data from the websocket connection
func (b *Bitmex) wsHandleIncomingData(c chan []byte) {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		case resp := <-c:
			message := string(resp)
			if common.StringContains(message, "pong") {
				pongChan <- 1
				continue
			}

			if common.StringContains(message, "ping") {
				err := b.WebsocketConn.WriteJSON("pong")
				if err != nil {
					b.Websocket.DataHandler <- err
				}
			}

			quickCapture := make(map[string]interface{})
			err := common.JSONDecode(resp, &quickCapture)
			if err != nil {
				log.Fatal(err)
			}

			var respError WebsocketErrorResponse
			if _, ok := quickCapture["status"]; ok {
				err = common.JSONDecode(resp, &respError)
				if err != nil {
					log.Fatal(err)
				}
				b.Websocket.DataHandler <- errors.New(respError.Error)
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

				b.Websocket.DataHandler <- fmt.Errorf("Bitmex websocket error: Unable to subscribe %s",
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

					if trades.Action == bitmexActionInitialData {
						continue
					}

					for _, trade := range trades.Data {
						timestamp, err := time.Parse(time.RFC3339, trade.Timestamp)
						if err != nil {
							log.Fatal(err)
						}

						b.Websocket.DataHandler <- exchange.TradeData{
							Timestamp:    timestamp,
							Price:        trade.Price,
							Amount:       float64(trade.Size),
							CurrencyPair: pair.NewCurrencyPairFromString(trade.Symbol),
							Exchange:     b.GetName(),
							AssetType:    "CONTRACT",
							Side:         trade.Side,
						}
					}

				case bitmexWSAnnouncement:
					var announcement AnnouncementData

					err = common.JSONDecode(resp, &announcement)
					if err != nil {
						log.Fatal(err)
					}

					if announcement.Action == bitmexActionInitialData {
						continue
					}

					b.Websocket.DataHandler <- announcement.Data

				default:
					log.Fatal("Bitmex websocket error: Table unknown -", decodedResp.Table)
				}
			}
		}
	}
}

// Temporary local cache of orderbooks
var localOb []OrderBookL2
var obMtx sync.Mutex
var partialLoaded bool

// ProcessOrderbook processes orderbook updates
func (b *Bitmex) processOrderbook(data []OrderBookL2, action string) error {
	if len(data) < 1 {
		return errors.New("no data receieved")
	}

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

	if !partialLoaded {
		b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdated
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

// WsShutdown terminates websocket connection and shuts down routines
func (b *Bitmex) WsShutdown() error {
	var (
		c     = make(chan struct{}, 1)
		timer = time.NewTimer(5 * time.Second)
	)

	go func(c chan struct{}) {
		close(b.Websocket.ShutdownC)
		b.Websocket.Wg.Wait()
		c <- struct{}{}
	}(c)

	select {
	case <-timer.C:
		return errors.New("routines did not shutdown")

	case <-c:
		return nil
	}
}
