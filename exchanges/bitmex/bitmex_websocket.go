package bitmex

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
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

	go b.wsHandleIncomingData()
	go b.wsReadData()

	err = b.websocketSubscribe()
	if err != nil {
		closeError := b.WebsocketConn.Close()
		if closeError != nil {
			return fmt.Errorf("bitmex_websocket.go error - Websocket connection could not close %s",
				closeError)
		}
		return err
	}

	if b.AuthenticatedAPISupport {
		err := b.websocketSendAuth()
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Bitmex) wsReadData() {
	b.Websocket.Wg.Add(1)

	defer func() {
		err := b.WebsocketConn.Close()
		if err != nil {
			b.Websocket.DataHandler <- fmt.Errorf("bitmex_websocket.go - Unable to close Websocket connection. Error: %s",
				err)
		}
		b.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		default:
			_, resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				b.Websocket.DataHandler <- fmt.Errorf("bitmex_websocket.go - websocket connection Error: %s",
					err)
				return
			}

			b.Websocket.TrafficAlert <- struct{}{}

			b.Websocket.Intercomm <- exchange.WebsocketResponse{
				Raw: resp,
			}
		}
	}
}

// wsHandleIncomingData services incoming data from the websocket connection
func (b *Bitmex) wsHandleIncomingData() {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		case resp := <-b.Websocket.Intercomm:
			message := string(resp.Raw)
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
			err := common.JSONDecode(resp.Raw, &quickCapture)
			if err != nil {
				log.Fatal(err)
			}

			var respError WebsocketErrorResponse
			if _, ok := quickCapture["status"]; ok {
				err = common.JSONDecode(resp.Raw, &respError)
				if err != nil {
					log.Fatal(err)
				}
				b.Websocket.DataHandler <- errors.New(respError.Error)
				continue
			}

			if _, ok := quickCapture["success"]; ok {
				var decodedResp WebsocketSubscribeResp
				err := common.JSONDecode(resp.Raw, &decodedResp)
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
				err := common.JSONDecode(resp.Raw, &decodedResp)
				if err != nil {
					log.Fatal(err)
				}

				switch decodedResp.Table {
				case bitmexWSOrderbookL2:
					var orderbooks OrderBookData
					err = common.JSONDecode(resp.Raw, &orderbooks)
					if err != nil {
						log.Fatal(err)
					}

					p := pair.NewCurrencyPairFromString(orderbooks.Data[0].Symbol)
					err = b.processOrderbook(orderbooks.Data, orderbooks.Action, p, "CONTRACT")
					if err != nil {
						log.Fatal(err)
					}

				case bitmexWSTrade:
					var trades TradeData
					err = common.JSONDecode(resp.Raw, &trades)
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

					err = common.JSONDecode(resp.Raw, &announcement)
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

var snapshotloaded = make(map[pair.CurrencyPair]map[string]bool)

// ProcessOrderbook processes orderbook updates
func (b *Bitmex) processOrderbook(data []OrderBookL2, action string, currencyPair pair.CurrencyPair, assetType string) error {
	if len(data) < 1 {
		return errors.New("bitmex_websocket.go error - no orderbook data")
	}

	_, ok := snapshotloaded[currencyPair]
	if !ok {
		snapshotloaded[currencyPair] = make(map[string]bool)
	}

	_, ok = snapshotloaded[currencyPair][assetType]
	if !ok {
		snapshotloaded[currencyPair][assetType] = false
	}

	switch action {
	case bitmexActionInitialData:
		if !snapshotloaded[currencyPair][assetType] {
			var newOrderbook orderbook.Base
			var bids, asks []orderbook.Item

			for _, orderbookItem := range data {
				if orderbookItem.Side == "Sell" {
					asks = append(asks, orderbook.Item{
						Price:  orderbookItem.Price,
						Amount: float64(orderbookItem.Size),
					})
					continue
				}
				bids = append(bids, orderbook.Item{
					Price:  orderbookItem.Price,
					Amount: float64(orderbookItem.Size),
				})
			}

			if len(bids) == 0 || len(asks) == 0 {
				return errors.New("bitmex_websocket.go error - snapshot not initialised correctly")
			}

			newOrderbook.Asks = asks
			newOrderbook.Bids = bids
			newOrderbook.AssetType = assetType
			newOrderbook.CurrencyPair = currencyPair.Pair().String()
			newOrderbook.LastUpdated = time.Now()
			newOrderbook.Pair = currencyPair

			err := b.Websocket.Orderbook.LoadSnapshot(newOrderbook, b.GetName())
			if err != nil {
				return fmt.Errorf("bitmex_websocket.go process orderbook error -  %s",
					err)
			}
			snapshotloaded[currencyPair][assetType] = true
			b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
				Pair:     currencyPair,
				Asset:    assetType,
				Exchange: b.GetName(),
			}
		}

	default:
		if snapshotloaded[currencyPair][assetType] {
			var asks, bids []orderbook.Item
			for _, orderbookItem := range data {
				if orderbookItem.Side == "Sell" {
					asks = append(asks, orderbook.Item{
						Price:  orderbookItem.Price,
						Amount: float64(orderbookItem.Size),
					})
					continue
				}
				bids = append(bids, orderbook.Item{
					Price:  orderbookItem.Price,
					Amount: float64(orderbookItem.Size),
				})
			}

			err := b.Websocket.Orderbook.UpdateUsingID(bids,
				asks,
				currencyPair,
				time.Now(),
				b.GetName(),
				assetType,
				action)

			if err != nil {
				return err
			}

			b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
				Pair:     currencyPair,
				Asset:    assetType,
				Exchange: b.GetName(),
			}
		}
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
