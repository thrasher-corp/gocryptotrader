package bitmex

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
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
	bitmexWSOrderbookL225       = "orderBookL2_25"
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

// WsConnect initiates a new websocket connection
func (b *Bitmex) WsConnect() error {
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
		b.Websocket.ReadMessageErrors <- err
		return err
	}
	b.Websocket.TrafficAlert <- struct{}{}
	var welcomeResp WebsocketWelcome
	err = json.Unmarshal(p.Raw, &welcomeResp)
	if err != nil {
		return err
	}

	if b.Verbose {
		log.Debugf(log.ExchangeSys, "Successfully connected to Bitmex %s at time: %s Limit: %d",
			welcomeResp.Info,
			welcomeResp.Timestamp,
			welcomeResp.Limit.Remaining)
	}

	go b.wsReadData()
	b.GenerateDefaultSubscriptions()
	err = b.websocketSendAuth()
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", b.Name, err)
	}
	b.GenerateAuthenticatedSubscriptions()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (b *Bitmex) wsReadData() {
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
			err = b.wsHandleData(resp.Raw)
			if err != nil {
				b.Websocket.DataHandler <- err
			}
		}
	}
}

func (b *Bitmex) wsHandleData(respRaw []byte) error {
	quickCapture := make(map[string]interface{})
	err := json.Unmarshal(respRaw, &quickCapture)
	if err != nil {
		return err
	}

	var respError WebsocketErrorResponse
	if _, ok := quickCapture["status"]; ok {
		err = json.Unmarshal(respRaw, &respError)
		if err != nil {
			return err
		}
	}

	if _, ok := quickCapture["success"]; ok {
		var decodedResp WebsocketSubscribeResp
		err = json.Unmarshal(respRaw, &decodedResp)
		if err != nil {
			return err
		}

		if decodedResp.Success {
			if len(quickCapture) == 3 {
				if b.Verbose {
					log.Debugf(log.ExchangeSys, "%s websocket: Successfully subscribed to %s",
						b.Name, decodedResp.Subscribe)
				}
			} else {
				b.Websocket.SetCanUseAuthenticatedEndpoints(true)
				if b.Verbose {
					log.Debugf(log.ExchangeSys, "%s websocket: Successfully authenticated websocket connection",
						b.Name)
				}
			}
			return nil
		}

		b.Websocket.DataHandler <- fmt.Errorf("%s websocket error: Unable to subscribe %s",
			b.Name, decodedResp.Subscribe)
	} else if _, ok := quickCapture["table"]; ok {
		var decodedResp WebsocketMainResponse
		err = json.Unmarshal(respRaw, &decodedResp)
		if err != nil {
			return err
		}
		switch decodedResp.Table {
		case bitmexWSOrderbookL2, bitmexWSOrderbookL225, bitmexWSOrderbookL10:
			var orderbooks OrderBookData
			err = json.Unmarshal(respRaw, &orderbooks)
			if err != nil {
				return err
			}
			if len(orderbooks.Data) == 0 {
				return fmt.Errorf("%s - Empty orderbook data received: %s", b.Name, respRaw)
			}
			p := currency.NewPairFromString(orderbooks.Data[0].Symbol)
			var a asset.Item
			a, err = b.GetPairAssetType(p)
			if err != nil {
				return err
			}

			err = b.processOrderbook(orderbooks.Data,
				orderbooks.Action,
				p,
				a)
			if err != nil {
				return err
			}

		case bitmexWSTrade:
			var trades TradeData
			err = json.Unmarshal(respRaw, &trades)
			if err != nil {
				return err
			}

			if trades.Action == bitmexActionInitialData {
				return nil
			}

			for i := range trades.Data {
				var a asset.Item
				p := currency.NewPairFromString(trades.Data[i].Symbol)
				a, err = b.GetPairAssetType(p)
				if err != nil {
					return err
				}
				var oSide order.Side
				oSide, err = order.StringToOrderSide(trades.Data[i].Side)
				if err != nil {
					b.Websocket.DataHandler <- order.ClassificationError{
						Exchange: b.Name,
						Err:      err,
					}
				}

				b.Websocket.DataHandler <- wshandler.TradeData{
					Timestamp:    trades.Data[i].Timestamp,
					Price:        trades.Data[i].Price,
					Amount:       float64(trades.Data[i].Size),
					CurrencyPair: p,
					Exchange:     b.Name,
					AssetType:    a,
					Side:         oSide,
				}
			}

		case bitmexWSAnnouncement:
			var announcement AnnouncementData
			err = json.Unmarshal(respRaw, &announcement)
			if err != nil {
				return err
			}

			if announcement.Action == bitmexActionInitialData {
				return nil
			}

			b.Websocket.DataHandler <- announcement.Data
		case bitmexWSAffiliate:
			var response WsAffiliateResponse
			err = json.Unmarshal(respRaw, &response)
			if err != nil {
				return err
			}
			b.Websocket.DataHandler <- response
		case bitmexWSInstrument:
			// ticker
		case bitmexWSExecution:
			// trades of an order
			var response WsExecutionResponse
			err = json.Unmarshal(respRaw, &response)
			if err != nil {
				return err
			}

			for i := range response.Data {
				p := currency.NewPairFromString(response.Data[i].Symbol)
				var a asset.Item
				a, err = b.GetPairAssetType(p)
				if err != nil {
					return err
				}
				var oStatus order.Status
				oStatus, err = order.StringToOrderStatus(response.Data[i].OrdStatus)
				if err != nil {
					b.Websocket.DataHandler <- order.ClassificationError{
						Exchange: b.Name,
						OrderID:  response.Data[i].OrderID,
						Err:      err,
					}
				}
				var oSide order.Side
				oSide, err = order.StringToOrderSide(response.Data[i].Side)
				if err != nil {
					b.Websocket.DataHandler <- order.ClassificationError{
						Exchange: b.Name,
						OrderID:  response.Data[i].OrderID,
						Err:      err,
					}
				}
				b.Websocket.DataHandler <- &order.Modify{
					Exchange:  b.Name,
					ID:        response.Data[i].OrderID,
					AccountID: strconv.FormatInt(response.Data[i].Account, 10),
					AssetType: a,
					Pair:      p,
					Status:    oStatus,
					Trades: []order.TradeHistory{
						{
							Price:     response.Data[i].Price,
							Amount:    response.Data[i].OrderQuantity,
							Exchange:  b.Name,
							TID:       response.Data[i].ExecID,
							Side:      oSide,
							Timestamp: response.Data[i].Timestamp,
							IsMaker:   false,
						},
					},
				}
			}
		case bitmexWSOrder:
			var response WsOrderResponse
			err = json.Unmarshal(respRaw, &response)
			if err != nil {
				return err
			}
			switch response.Action {
			case "update", "insert":
				for x := range response.Data {
					var p currency.Pair
					var a asset.Item
					p, a, err = b.GetRequestFormattedPairAndAssetType(response.Data[x].Symbol)
					if err != nil {
						return err
					}
					var oSide order.Side
					oSide, err = order.StringToOrderSide(response.Data[x].Side)
					if err != nil {
						b.Websocket.DataHandler <- order.ClassificationError{
							Exchange: b.Name,
							OrderID:  response.Data[x].OrderID,
							Err:      err,
						}
					}
					var oType order.Type
					oType, err = order.StringToOrderType(response.Data[x].OrderType)
					if err != nil {
						b.Websocket.DataHandler <- order.ClassificationError{
							Exchange: b.Name,
							OrderID:  response.Data[x].OrderID,
							Err:      err,
						}
					}
					var oStatus order.Status
					oStatus, err = order.StringToOrderStatus(response.Data[x].OrderStatus)
					if err != nil {
						b.Websocket.DataHandler <- order.ClassificationError{
							Exchange: b.Name,
							OrderID:  response.Data[x].OrderID,
							Err:      err,
						}
					}
					b.Websocket.DataHandler <- &order.Detail{
						Price:     response.Data[x].Price,
						Amount:    response.Data[x].OrderQuantity,
						Exchange:  b.Name,
						ID:        response.Data[x].OrderID,
						AccountID: strconv.FormatInt(response.Data[x].Account, 10),
						Type:      oType,
						Side:      oSide,
						Status:    oStatus,
						AssetType: a,
						Date:      response.Data[x].TransactTime,
						Pair:      p,
					}
				}
			case "delete":
				for x := range response.Data {
					var p currency.Pair
					var a asset.Item
					p, a, err = b.GetRequestFormattedPairAndAssetType(response.Data[x].Symbol)
					if err != nil {
						return err
					}
					var oSide order.Side
					oSide, err = order.StringToOrderSide(response.Data[x].Side)
					if err != nil {
						b.Websocket.DataHandler <- order.ClassificationError{
							Exchange: b.Name,
							OrderID:  response.Data[x].OrderID,
							Err:      err,
						}
					}
					var oType order.Type
					oType, err = order.StringToOrderType(response.Data[x].OrderType)
					if err != nil {
						b.Websocket.DataHandler <- order.ClassificationError{
							Exchange: b.Name,
							OrderID:  response.Data[x].OrderID,
							Err:      err,
						}
					}
					var oStatus order.Status
					oStatus, err = order.StringToOrderStatus(response.Data[x].OrderStatus)
					if err != nil {
						b.Websocket.DataHandler <- order.ClassificationError{
							Exchange: b.Name,
							OrderID:  response.Data[x].OrderID,
							Err:      err,
						}
					}
					b.Websocket.DataHandler <- &order.Cancel{
						Price:     response.Data[x].Price,
						Amount:    response.Data[x].OrderQuantity,
						Exchange:  b.Name,
						ID:        response.Data[x].OrderID,
						AccountID: strconv.FormatInt(response.Data[x].Account, 10),
						Type:      oType,
						Side:      oSide,
						Status:    oStatus,
						AssetType: a,
						Date:      response.Data[x].TransactTime,
						Pair:      p,
					}
				}
			default:
				b.Websocket.DataHandler <- fmt.Errorf("%s - Unsupported order update %+v", b.Name, response)
			}
		case bitmexWSMargin:
			var response WsMarginResponse
			err = json.Unmarshal(respRaw, &response)
			if err != nil {
				return err
			}
			b.Websocket.DataHandler <- response
		case bitmexWSPosition:
			var response WsPositionResponse
			err = json.Unmarshal(respRaw, &response)
			if err != nil {
				return err
			}

		case bitmexWSPrivateNotifications:
			var response WsPrivateNotificationsResponse
			err = json.Unmarshal(respRaw, &response)
			if err != nil {
				return err
			}
			b.Websocket.DataHandler <- response
		case bitmexWSTransact:
			var response WsTransactResponse
			err = json.Unmarshal(respRaw, &response)
			if err != nil {
				return err
			}
			b.Websocket.DataHandler <- response
		case bitmexWSWallet:
			var response WsWalletResponse
			err = json.Unmarshal(respRaw, &response)
			if err != nil {
				return err
			}
			b.Websocket.DataHandler <- response
		default:
			b.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: b.Name + wshandler.UnhandledMessage + string(respRaw)}
			return nil
		}
	}
	return nil
}

// ProcessOrderbook processes orderbook updates
func (b *Bitmex) processOrderbook(data []OrderBookL2, action string, currencyPair currency.Pair, assetType asset.Item) error {
	if len(data) < 1 {
		return errors.New("bitmex_websocket.go error - no orderbook data")
	}

	switch action {
	case bitmexActionInitialData:
		var newOrderBook orderbook.Base
		for i := range data {
			if strings.EqualFold(data[i].Side, order.Sell.String()) {
				newOrderBook.Asks = append(newOrderBook.Asks, orderbook.Item{
					Price:  data[i].Price,
					Amount: float64(data[i].Size),
					ID:     data[i].ID,
				})
				continue
			}
			newOrderBook.Bids = append(newOrderBook.Bids, orderbook.Item{
				Price:  data[i].Price,
				Amount: float64(data[i].Size),
				ID:     data[i].ID,
			})
		}
		newOrderBook.AssetType = assetType
		newOrderBook.Pair = currencyPair
		newOrderBook.ExchangeName = b.Name

		err := b.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
		if err != nil {
			return fmt.Errorf("bitmex_websocket.go process orderbook error -  %s",
				err)
		}
		b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
			Pair:     currencyPair,
			Asset:    assetType,
			Exchange: b.Name,
		}
	default:
		var asks, bids []orderbook.Item
		for i := range data {
			if strings.EqualFold(data[i].Side, "Sell") {
				asks = append(asks, orderbook.Item{
					Amount: float64(data[i].Size),
					ID:     data[i].ID,
				})
				continue
			}
			bids = append(bids, orderbook.Item{
				Amount: float64(data[i].Size),
				ID:     data[i].ID,
			})
		}

		err := b.Websocket.Orderbook.Update(&wsorderbook.WebsocketOrderbookUpdate{
			Bids:   bids,
			Asks:   asks,
			Pair:   currencyPair,
			Asset:  assetType,
			Action: action,
		})
		if err != nil {
			return err
		}

		b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
			Pair:     currencyPair,
			Asset:    assetType,
			Exchange: b.Name,
		}
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *Bitmex) GenerateDefaultSubscriptions() {
	assets := b.GetAssetTypes()
	var allPairs currency.Pairs

	for x := range assets {
		contracts := b.GetEnabledPairs(assets[x])
		for y := range contracts {
			allPairs = allPairs.Add(contracts[y])
		}
	}

	channels := []string{bitmexWSOrderbookL2, bitmexWSTrade}
	subscriptions := []wshandler.WebsocketChannelSubscription{
		{
			Channel: bitmexWSAnnouncement,
		},
	}

	for i := range channels {
		for j := range allPairs {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  channels[i] + ":" + allPairs[j].String(),
				Currency: allPairs[j],
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
	contracts := b.GetEnabledPairs(asset.PerpetualContract)
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
				Channel:  channels[i] + ":" + contracts[j].String(),
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
	return b.WebsocketConn.SendJSONMessage(subscriber)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bitmex) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var subscriber WebsocketRequest
	subscriber.Command = "unsubscribe"
	subscriber.Arguments = append(subscriber.Arguments,
		channelToSubscribe.Params["args"],
		channelToSubscribe.Channel+":"+channelToSubscribe.Currency.String())
	return b.WebsocketConn.SendJSONMessage(subscriber)
}

// WebsocketSendAuth sends an authenticated subscription
func (b *Bitmex) websocketSendAuth() error {
	if !b.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", b.Name)
	}
	b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	timestamp := time.Now().Add(time.Hour * 1).Unix()
	newTimestamp := strconv.FormatInt(timestamp, 10)
	hmac := crypto.GetHMAC(crypto.HashSHA256,
		[]byte("GET/realtime"+newTimestamp),
		[]byte(b.API.Credentials.Secret))
	signature := crypto.HexEncodeToString(hmac)

	var sendAuth WebsocketRequest
	sendAuth.Command = "authKeyExpires"
	sendAuth.Arguments = append(sendAuth.Arguments, b.API.Credentials.Key, timestamp,
		signature)
	err := b.WebsocketConn.SendJSONMessage(sendAuth)
	if err != nil {
		b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}
