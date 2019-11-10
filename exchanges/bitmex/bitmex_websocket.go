package bitmex

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/idoall/gocryptotrader/common"
	"github.com/idoall/gocryptotrader/currency"
	exchange "github.com/idoall/gocryptotrader/exchanges"
	"github.com/idoall/gocryptotrader/exchanges/orderbook"
	"github.com/idoall/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/idoall/gocryptotrader/exchanges/websocket/wsorderbook"
	log "github.com/idoall/gocryptotrader/logger"
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

	bitmexActionTradeBucket = "partial"
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
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := b.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	p, err := b.WebsocketConn.ReadMessage()
	if err != nil {
		return err
	}
	b.Websocket.TrafficAlert <- struct{}{}
	var welcomeResp WebsocketWelcome
	err = common.JSONDecode(p.Raw, &welcomeResp)
	if err != nil {
		return err
	}

	if b.Verbose {
		log.Debugf("Successfully connected to Bitmex %s at time: %s Limit: %d",
			welcomeResp.Info,
			welcomeResp.Timestamp,
			welcomeResp.Limit.Remaining)
	}

	go b.wsHandleIncomingData()
	b.GenerateDefaultSubscriptions()

	err = b.websocketSendAuth()
	if err != nil {
		log.Errorf("%v - authentication failed: %v", b.Name, err)
	}
	b.GenerateAuthenticatedSubscriptions()
	return nil
}

// wsHandleIncomingData services incoming data from the websocket connection
func (b *Bitmex) wsHandleIncomingData() {
	b.Websocket.Wg.Add(1)

	defer func() {
		b.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		default:
			resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				b.Websocket.DataHandler <- err
				return
			}
			b.Websocket.TrafficAlert <- struct{}{}
			message := string(resp.Raw)
			if common.StringContains(message, "pong") {
				pongChan <- 1
				continue
			}

			if common.StringContains(message, "ping") {
				err = b.WebsocketConn.SendMessage("pong")
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
			}

			quickCapture := make(map[string]interface{})
			err = common.JSONDecode(resp.Raw, &quickCapture)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

			var respError WebsocketErrorResponse
			if _, ok := quickCapture["status"]; ok {
				err = common.JSONDecode(resp.Raw, &respError)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
				b.Websocket.DataHandler <- errors.New(respError.Error)
				continue
			}

			if _, ok := quickCapture["success"]; ok {
				var decodedResp WebsocketSubscribeResp
				err := common.JSONDecode(resp.Raw, &decodedResp)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				if decodedResp.Success {
					b.Websocket.DataHandler <- decodedResp
					if len(quickCapture) == 3 {
						if b.Verbose {
							log.Debugf("%s websocket: Successfully subscribed to %s",
								b.Name, decodedResp.Subscribe)
						}
					} else {
						b.Websocket.SetCanUseAuthenticatedEndpoints(true)
						if b.Verbose {
							log.Debugf("%s websocket: Successfully authenticated websocket connection",
								b.Name)
						}
					}
					continue
				}

				b.Websocket.DataHandler <- fmt.Errorf("%s websocket error: Unable to subscribe %s",
					b.Name, decodedResp.Subscribe)

			} else if _, ok := quickCapture["table"]; ok {
				var decodedResp WebsocketMainResponse
				err := common.JSONDecode(resp.Raw, &decodedResp)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				switch decodedResp.Table {
				case bitmexWSOrderbookL2:
					var orderbooks OrderBookData
					err = common.JSONDecode(resp.Raw, &orderbooks)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}

					p := currency.NewPairFromString(orderbooks.Data[0].Symbol)
					// TODO: update this to support multiple asset types
					err = b.processOrderbook(orderbooks.Data, orderbooks.Action, p, "CONTRACT")
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}

				case bitmexWSTrade:
					var trades TradeData
					err = common.JSONDecode(resp.Raw, &trades)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}

					if trades.Action == bitmexActionInitialData {
						continue
					}

					for i := range trades.Data {
						var timestamp time.Time
						timestamp, err = time.Parse(time.RFC3339, trades.Data[i].Timestamp)
						if err != nil {
							b.Websocket.DataHandler <- err
							continue
						}
						// TODO: update this to support multiple asset types
						b.Websocket.DataHandler <- wshandler.TradeData{
							Timestamp:    timestamp,
							Price:        trades.Data[i].Price,
							Amount:       float64(trades.Data[i].Size),
							CurrencyPair: currency.NewPairFromString(trades.Data[i].Symbol),
							Exchange:     b.GetName(),
							AssetType:    "CONTRACT",
							Side:         trades.Data[i].Side,
						}
					}

				case bitmexWSAnnouncement:
					var announcement AnnouncementData
					err = common.JSONDecode(resp.Raw, &announcement)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}

					if announcement.Action == bitmexActionInitialData {
						continue
					}

					b.Websocket.DataHandler <- announcement.Data
				case bitmexWSAffiliate:
					var response WsAffiliateResponse
					err = common.JSONDecode(resp.Raw, &response)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					b.Websocket.DataHandler <- response
				case bitmexWSExecution:
					var response WsExecutionResponse
					err = common.JSONDecode(resp.Raw, &response)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					b.Websocket.DataHandler <- response
				case bitmexWSOrder:
					var response WsOrderResponse
					err = common.JSONDecode(resp.Raw, &response)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					b.Websocket.DataHandler <- response
				case bitmexWSMargin:
					var response WsMarginResponse
					err = common.JSONDecode(resp.Raw, &response)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					b.Websocket.DataHandler <- response
				case bitmexWSPosition:
					var response WsPositionResponse
					err = common.JSONDecode(resp.Raw, &response)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					b.Websocket.DataHandler <- response
				case bitmexWSPrivateNotifications:
					var response WsPrivateNotificationsResponse
					err = common.JSONDecode(resp.Raw, &response)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					b.Websocket.DataHandler <- response
				case bitmexWSTransact:
					var response WsTransactResponse
					err = common.JSONDecode(resp.Raw, &response)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					b.Websocket.DataHandler <- response
				case bitmexWSWallet:
					var response WsWalletResponse
					err = common.JSONDecode(resp.Raw, &response)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					b.Websocket.DataHandler <- response
				default:
					b.Websocket.DataHandler <- fmt.Errorf("%s websocket error: Table unknown - %s",
						b.Name, decodedResp.Table)
				}
			}
		}
	}
}

// ProcessOrderbook processes orderbook updates
func (b *Bitmex) processOrderbook(data []OrderBookL2, action string, currencyPair currency.Pair, assetType string) error { // nolint: unparam
	if len(data) < 1 {
		return errors.New("bitmex_websocket.go error - no orderbook data")
	}

	switch action {
	case bitmexActionInitialData:
		var newOrderBook orderbook.Base
		var bids, asks []orderbook.Item
		for i := range data {
			if strings.EqualFold(data[i].Side, exchange.SellOrderSide.ToString()) {
				asks = append(asks, orderbook.Item{
					Price:  data[i].Price,
					Amount: float64(data[i].Size),
				})
				continue
			}
			bids = append(bids, orderbook.Item{
				Price:  data[i].Price,
				Amount: float64(data[i].Size),
			})
		}

		if len(bids) == 0 || len(asks) == 0 {
			return errors.New("bitmex_websocket.go error - snapshot not initialised correctly")
		}

		newOrderBook.Asks = asks
		newOrderBook.Bids = bids
		newOrderBook.AssetType = assetType
		newOrderBook.Pair = currencyPair
		err := b.Websocket.Orderbook.LoadSnapshot(&newOrderBook, false)
		if err != nil {
			return fmt.Errorf("bitmex_websocket.go process orderbook error -  %s",
				err)
		}
		b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
			Pair:     currencyPair,
			Asset:    assetType,
			Exchange: b.GetName(),
		}
	default:
		var asks, bids []orderbook.Item
		for i := range data {
			if strings.EqualFold(data[i].Side, "Sell") {
				asks = append(asks, orderbook.Item{
					Price:  data[i].Price,
					Amount: float64(data[i].Size),
				})
				continue
			}
			bids = append(bids, orderbook.Item{
				Price:  data[i].Price,
				Amount: float64(data[i].Size),
			})
		}

		err := b.Websocket.Orderbook.Update(&wsorderbook.WebsocketOrderbookUpdate{
			Bids:         bids,
			Asks:         asks,
			CurrencyPair: currencyPair,
			UpdateTime:   time.Now(),
			AssetType:    assetType,
			Action:       action,
		})
		if err != nil {
			return err
		}

		b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
			Pair:     currencyPair,
			Asset:    assetType,
			Exchange: b.GetName(),
		}
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *Bitmex) GenerateDefaultSubscriptions() {
	contracts := b.GetEnabledCurrencies()
	channels := []string{bitmexWSOrderbookL2, bitmexWSTrade}
	subscriptions := []wshandler.WebsocketChannelSubscription{
		{
			Channel: bitmexWSAnnouncement,
		},
	}

	for i := range channels {
		for j := range contracts {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  fmt.Sprintf("%v:%v", channels[i], contracts[j].String()),
				Currency: contracts[j],
			})
		}
	}
	b.Websocket.SubscribeToChannels(subscriptions)
}

// GenerateAuthenticatedSubscriptions Adds authenticated subscriptions to websocket to be handled by ManageSubscriptions()
func (b *Bitmex) GenerateAuthenticatedSubscriptions() {
	if !b.Websocket.CanUseAuthenticatedEndpoints() {
		return
	}
	contracts := b.GetEnabledCurrencies()
	channels := []string{bitmexWSExecution,
		bitmexWSPosition,
	}
	subscriptions := []wshandler.WebsocketChannelSubscription{
		{
			Channel: bitmexWSAffiliate,
		},
		{
			Channel: bitmexWSOrder,
		},
		{
			Channel: bitmexWSMargin,
		},
		{
			Channel: bitmexWSPrivateNotifications,
		},
		{
			Channel: bitmexWSTransact,
		},
		{
			Channel: bitmexWSWallet,
		},
	}
	for i := range channels {
		for j := range contracts {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  fmt.Sprintf("%v:%v", channels[i], contracts[j].String()),
				Currency: contracts[j],
			})
		}
	}
	b.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe subscribes to a websocket channel
func (b *Bitmex) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var subscriber WebsocketRequest
	subscriber.Command = "subscribe"
	subscriber.Arguments = append(subscriber.Arguments, channelToSubscribe.Channel)
	return b.WebsocketConn.SendMessage(subscriber)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bitmex) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var subscriber WebsocketRequest
	subscriber.Command = "unsubscribe"
	subscriber.Arguments = append(subscriber.Arguments,
		channelToSubscribe.Params["args"],
		channelToSubscribe.Channel+":"+channelToSubscribe.Currency.String())
	return b.WebsocketConn.SendMessage(subscriber)
}

// WebsocketSendAuth sends an authenticated subscription
func (b *Bitmex) websocketSendAuth() error {
	if !b.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", b.Name)
	}
	b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	timestamp := time.Now().Add(time.Hour * 1).Unix()
	newTimestamp := strconv.FormatInt(timestamp, 10)
	hmac := common.GetHMAC(common.HashSHA256,
		[]byte("GET/realtime"+newTimestamp),
		[]byte(b.APISecret))
	signature := common.HexEncodeToString(hmac)
	var sendAuth WebsocketRequest
	sendAuth.Command = "authKeyExpires"
	sendAuth.Arguments = append(sendAuth.Arguments, b.APIKey, timestamp,
		signature)
	err := b.WebsocketConn.SendMessage(sendAuth)
	if err != nil {
		b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// WsSend sends data to the websocket server
// func (b *Bitmex) wsSend(data interface{}) error {
// 	b.wsRequestMtx.Lock()
// 	defer b.wsRequestMtx.Unlock()
// 	if b.Verbose {
// 		log.Debugf("%v sending message to websocket %v", b.Name, data)
// 	}
// 	return b.WebsocketConn.WriteJSON(data)
// }

// WebsocketKline 读取K线
// func (b *Bitmex) WebsocketKline(ch chan *TradeBucketData, timeIntervals []TimeInterval, symbolList []string, done <-chan struct{}) {

// 	for b.Enabled && b.Websocket {
// 		select {
// 		case <-done:
// 			return
// 		default:
// 			var dialer websocket.Dialer
// 			var err error

// 			b.WebsocketConn, _, err = dialer.Dial(bitmexWSURL, nil)
// 			if err != nil {
// 				log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.Name, err)
// 				continue
// 			} else if b.Verbose {
// 				log.Printf("%s Connected to Websocket.\n", b.Name)
// 			}

// 			_, p, err := b.WebsocketConn.ReadMessage()
// 			if err != nil {
// 				b.WebsocketConn.Close()
// 				log.Fatal(fmt.Sprintf("First ReadMessage:%s", err.Error()))
// 			}

// 			//解析欢迎 信息
// 			var welcomeResp WebsocketWelcome
// 			err = common.JSONDecode(p, &welcomeResp)
// 			if err != nil {
// 				log.Fatal(fmt.Sprintf("WelCome ReadMessage:%s", err.Error()))
// 			}
// 			if b.Verbose {
// 				log.Printf("Successfully connected to Bitmex %s at time: %s Limit: %d",
// 					welcomeResp.Info,
// 					welcomeResp.Timestamp,
// 					welcomeResp.Limit.Remaining)
// 			}

// 			//订阅信息
// 			var subscriber WebsocketRequest
// 			subscriber.Command = "subscribe"

// 			// 重新匹配要订阅的名称
// 			for _, tv := range timeIntervals {
// 				tradeName := ""
// 				switch tv {
// 				case TimeIntervalMinute:
// 					tradeName = bitmexWSTrade1m
// 				case TimeIntervalFiveMinutes:
// 					tradeName = bitmexWSTrade5m
// 				case TimeIntervalHour:
// 					tradeName = bitmexWSTrade1h
// 				case TimeIntervalDay:
// 					tradeName = bitmexWSTrade1d
// 				default:
// 					continue
// 				}
// 				for _, sv := range symbolList {
// 					subscriber.Arguments = append(subscriber.Arguments, fmt.Sprintf("%s:%s", tradeName, common.StringToUpper(sv)))
// 				}
// 			}

// 			if b.Verbose {
// 				if b, err := common.JSONEncode(subscriber); err != nil {
// 					log.Fatal(err)
// 				} else {
// 					log.Printf("subscriber:%s\n", b)
// 				}
// 			}

// 			err = b.WebsocketConn.WriteJSON(subscriber)
// 			if err != nil {
// 				log.Fatal(err)
// 			}

// 			for b.Enabled && b.Websocket {
// 				select {
// 				case <-done:
// 					return
// 				default:
// 					_, resp, err := b.WebsocketConn.ReadMessage()
// 					if err != nil {
// 						if b.Verbose {

// 							log.Println("Bitmex websocket: Connection error", err)
// 						}
// 						return
// 					}

// 					message := string(resp)
// 					if common.StringContains(message, "pong") {
// 						if b.Verbose {
// 							log.Println("Bitmex websocket: PONG receieved")
// 						}
// 						continue
// 					}

// 					if common.StringContains(message, "ping") {
// 						err = b.WebsocketConn.WriteJSON("pong")
// 						if err != nil {
// 							if b.Verbose {
// 								log.Println("Bitmex websocket error: ", err)
// 							}
// 							return
// 						}
// 					}

// 					quickCapture := make(map[string]interface{})
// 					err = common.JSONDecode(resp, &quickCapture)
// 					if err != nil {
// 						fmt.Printf("Err resp:%s \n", resp)
// 						log.Fatal(time.Now().Format("2006-01-02 15:04:05") + " " + err.Error())
// 					}

// 					//解析错误信息
// 					var respError WebsocketErrorResponse
// 					if _, ok := quickCapture["status"]; ok {
// 						err = common.JSONDecode(resp, &respError)
// 						if err != nil {
// 							log.Fatal(err)
// 						}
// 						log.Printf("Bitmex websocket error: %s", respError.Error)
// 						continue
// 					}

// 					if _, ok := quickCapture["success"]; ok {
// 						var decodedResp WebsocketSubscribeResp
// 						err := common.JSONDecode(resp, &decodedResp)
// 						if err != nil {
// 							log.Fatal(err)
// 						}

// 						if decodedResp.Success {
// 							if b.Verbose {
// 								if len(quickCapture) == 3 {
// 									log.Printf("Bitmex Websocket: Successfully subscribed to %s",
// 										decodedResp.Subscribe)
// 								} else {
// 									log.Println("Bitmex Websocket: Successfully authenticated websocket connection")
// 								}
// 							}
// 							continue
// 						}
// 						log.Printf("Bitmex websocket error: Unable to subscribe %s",
// 							decodedResp.Subscribe)

// 					} else if _, ok := quickCapture["table"]; ok {
// 						var decodedResp WebsocketMainResponse
// 						err := common.JSONDecode(resp, &decodedResp)
// 						if err != nil {
// 							log.Fatal(err)
// 						}
// 						// fmt.Println(decodedResp.Table)
// 						// fmt.Printf("[%s]%s\n", decodedResp.Table, resp)
// 						switch decodedResp.Table {
// 						case bitmexWSTrade1m:
// 							var tradeBucketData TradeBucketData
// 							err = common.JSONDecode(resp, &tradeBucketData)
// 							if err != nil {
// 								log.Fatal(err)
// 							}
// 							tradeBucketData.TimeInterval = TimeIntervalMinute
// 							ch <- &tradeBucketData
// 						case bitmexWSTrade5m:
// 							var tradeBucketData TradeBucketData
// 							err = common.JSONDecode(resp, &tradeBucketData)
// 							if err != nil {
// 								log.Fatal(err)
// 							}
// 							tradeBucketData.TimeInterval = TimeIntervalFiveMinutes
// 							ch <- &tradeBucketData
// 						case bitmexWSTrade1h:
// 							var tradeBucketData TradeBucketData
// 							err = common.JSONDecode(resp, &tradeBucketData)
// 							if err != nil {
// 								log.Fatal(err)
// 							}
// 							tradeBucketData.TimeInterval = TimeIntervalHour
// 							ch <- &tradeBucketData
// 						case bitmexWSTrade1d:
// 							var tradeBucketData TradeBucketData
// 							err = common.JSONDecode(resp, &tradeBucketData)
// 							if err != nil {
// 								log.Fatal(err)
// 							}
// 							fmt.Printf("日线:%s \n", resp)
// 							tradeBucketData.TimeInterval = TimeIntervalDay
// 							ch <- &tradeBucketData

// 						default:
// 							log.Fatal("Bitmex websocket error: Table unknown -", decodedResp.Table)
// 						}
// 					}

// 				}
// 			}

// 			b.WebsocketConn.Close()
// 			log.Printf("%s Websocket client disconnected.", b.Name)
// 		}
// 	}
// }

// WebsocketLastPrice 读取 最新报价
// func (b *Bitmex) WebsocketLastPrice(ch chan *WSInstrumentData, symbolList []string, done <-chan struct{}) {

// 	for b.Enabled && b.Websocket {
// 		select {
// 		case <-done:
// 			return
// 		default:
// 			var dialer websocket.Dialer
// 			var err error

// 			b.WebsocketConn, _, err = dialer.Dial(bitmexWSURL, nil)
// 			if err != nil {
// 				log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.Name, err)
// 				continue
// 			} else if b.Verbose {
// 				log.Printf("%s Connected to Websocket.\n", b.Name)
// 			}

// 			_, p, err := b.WebsocketConn.ReadMessage()
// 			if err != nil {
// 				b.WebsocketConn.Close()
// 				log.Fatal(fmt.Sprintf("First ReadMessage:%s", err.Error()))
// 			}

// 			//解析欢迎 信息
// 			var welcomeResp WebsocketWelcome
// 			err = common.JSONDecode(p, &welcomeResp)
// 			if err != nil {
// 				log.Fatal(fmt.Sprintf("WelCome ReadMessage:%s", err.Error()))
// 			}
// 			if b.Verbose {
// 				log.Printf("Successfully connected to Bitmex %s at time: %s Limit: %d",
// 					welcomeResp.Info,
// 					welcomeResp.Timestamp,
// 					welcomeResp.Limit.Remaining)
// 			}

// 			//订阅信息
// 			var subscriber WebsocketRequest
// 			subscriber.Command = "subscribe"

// 			// 重新匹配要订阅的名称
// 			for _, sv := range symbolList {
// 				subscriber.Arguments = append(subscriber.Arguments, fmt.Sprintf("%s:%s", bitmexWSInstrument, common.StringToUpper(sv)))
// 			}

// 			if b.Verbose {
// 				if b, err := common.JSONEncode(subscriber); err != nil {
// 					log.Fatal(err)
// 				} else {
// 					log.Printf("subscriber:%s\n", b)
// 				}
// 			}

// 			err = b.WebsocketConn.WriteJSON(subscriber)
// 			if err != nil {
// 				log.Fatal(err)
// 			}

// 			for b.Enabled && b.Websocket {
// 				select {
// 				case <-done:
// 					return
// 				default:
// 					_, resp, err := b.WebsocketConn.ReadMessage()
// 					if err != nil {
// 						if b.Verbose {
// 							log.Println("Bitmex websocket: Connection error", err)
// 						}
// 						return
// 					}

// 					quickCapture := make(map[string]interface{})
// 					err = common.JSONDecode(resp, &quickCapture)
// 					if err != nil {
// 						fmt.Printf("Err resp:%s \n", resp)
// 						log.Fatal(time.Now().Format("2006-01-02 15:04:05") + " " + err.Error())
// 					}

// 					//解析错误信息
// 					var respError WebsocketErrorResponse
// 					if _, ok := quickCapture["status"]; ok {
// 						err = common.JSONDecode(resp, &respError)
// 						if err != nil {
// 							log.Fatal(err)
// 						}
// 						log.Printf("Bitmex websocket error: %s", respError.Error)
// 						continue
// 					}

// 					if _, ok := quickCapture["success"]; ok {
// 						var decodedResp WebsocketSubscribeResp
// 						err := common.JSONDecode(resp, &decodedResp)
// 						if err != nil {
// 							log.Fatal(err)
// 						}

// 						if decodedResp.Success {
// 							if b.Verbose {
// 								if len(quickCapture) == 3 {
// 									log.Printf("Bitmex Websocket: Successfully subscribed to %s",
// 										decodedResp.Subscribe)
// 								} else {
// 									log.Println("Bitmex Websocket: Successfully authenticated websocket connection")
// 								}
// 							}
// 							continue
// 						}
// 						log.Printf("Bitmex websocket error: Unable to subscribe %s",
// 							decodedResp.Subscribe)

// 					} else if _, ok := quickCapture["table"]; ok {
// 						var decodedResp WebsocketMainResponse
// 						err := common.JSONDecode(resp, &decodedResp)
// 						if err != nil {
// 							log.Fatal(err)
// 						}

// 						switch decodedResp.Table {

// 						case bitmexWSInstrument:
// 							var wsInstrumentData WSInstrumentData
// 							err = common.JSONDecode(resp, &wsInstrumentData)
// 							if err != nil {
// 								log.Fatal(err)
// 							}

// 							if wsInstrumentData.Data[0].LastPrice != 0 {
// 								ch <- &wsInstrumentData
// 							}

// 						default:
// 							log.Fatal("Bitmex websocket error: Table unknown -", decodedResp.Table)
// 						}
// 					}

// 				}
// 			}

// 			b.WebsocketConn.Close()
// 			log.Printf("%s Websocket client disconnected.", b.Name)
// 		}
// 	}
// }
