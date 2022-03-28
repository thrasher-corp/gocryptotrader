package bitmex

import (
	"context"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
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
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	resp := b.Websocket.Conn.ReadMessage()
	if resp.Raw == nil {
		return errors.New("connection closed")
	}
	var welcomeResp WebsocketWelcome
	err = json.Unmarshal(resp.Raw, &welcomeResp)
	if err != nil {
		return err
	}

	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"Successfully connected to Bitmex %s at time: %s Limit: %d",
			welcomeResp.Info,
			welcomeResp.Timestamp,
			welcomeResp.Limit.Remaining)
	}

	b.Websocket.Wg.Add(1)
	go b.wsReadData()

	err = b.websocketSendAuth(context.TODO())
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%v - authentication failed: %v\n",
			b.Name,
			err)
	} else {
		authsubs, err := b.GenerateAuthenticatedSubscriptions()
		if err != nil {
			return err
		}
		return b.Websocket.SubscribeToChannels(authsubs)
	}
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (b *Bitmex) wsReadData() {
	defer b.Websocket.Wg.Done()

	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsHandleData(resp.Raw)
		if err != nil {
			b.Websocket.DataHandler <- err
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
			var p currency.Pair
			p, err = currency.NewPairFromString(orderbooks.Data[0].Symbol)
			if err != nil {
				return err
			}

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
			if !b.IsSaveTradeDataEnabled() {
				return nil
			}
			var tradeHolder TradeData
			err = json.Unmarshal(respRaw, &tradeHolder)
			if err != nil {
				return err
			}
			var trades []trade.Data
			for i := range tradeHolder.Data {
				if tradeHolder.Data[i].Price == 0 {
					// Please note that indices (symbols starting with .) post trades at intervals to the trade feed.
					// These have a size of 0 and are used only to indicate a changing price.
					continue
				}
				var p currency.Pair
				p, err = currency.NewPairFromString(tradeHolder.Data[i].Symbol)
				if err != nil {
					return err
				}

				var a asset.Item
				a, err = b.GetPairAssetType(p)
				if err != nil {
					return err
				}
				var oSide order.Side
				oSide, err = order.StringToOrderSide(tradeHolder.Data[i].Side)
				if err != nil {
					b.Websocket.DataHandler <- order.ClassificationError{
						Exchange: b.Name,
						Err:      err,
					}
				}

				trades = append(trades, trade.Data{
					TID:          tradeHolder.Data[i].TrdMatchID,
					Exchange:     b.Name,
					CurrencyPair: p,
					AssetType:    a,
					Side:         oSide,
					Price:        tradeHolder.Data[i].Price,
					Amount:       float64(tradeHolder.Data[i].Size),
					Timestamp:    tradeHolder.Data[i].Timestamp,
				})
			}
			return b.AddTradesToBuffer(trades...)
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
				var p currency.Pair
				p, err = currency.NewPairFromString(response.Data[i].Symbol)
				if err != nil {
					return err
				}

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
					b.Websocket.DataHandler <- &order.Modify{
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
			b.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: b.Name + stream.UnhandledMessage + string(respRaw)}
			return nil
		}
	}
	return nil
}

// ProcessOrderbook processes orderbook updates
func (b *Bitmex) processOrderbook(data []OrderBookL2, action string, p currency.Pair, a asset.Item) error {
	if len(data) < 1 {
		return errors.New("no orderbook data")
	}

	switch action {
	case bitmexActionInitialData:
		var book orderbook.Base
		for i := range data {
			item := orderbook.Item{
				Price:  data[i].Price,
				Amount: float64(data[i].Size),
				ID:     data[i].ID,
			}
			switch {
			case strings.EqualFold(data[i].Side, order.Sell.String()):
				book.Asks = append(book.Asks, item)
			case strings.EqualFold(data[i].Side, order.Buy.String()):
				book.Bids = append(book.Bids, item)
			default:
				return fmt.Errorf("could not process websocket orderbook update, order side could not be matched for %s",
					data[i].Side)
			}
		}
		book.Asks.Reverse() // Reverse asks for correct alignment
		book.Asset = a
		book.Pair = p
		book.Exchange = b.Name
		book.VerifyOrderbook = b.CanVerifyOrderbook

		err := b.Websocket.Orderbook.LoadSnapshot(&book)
		if err != nil {
			return fmt.Errorf("process orderbook error -  %s",
				err)
		}
	default:
		var asks, bids []orderbook.Item
		for i := range data {
			nItem := orderbook.Item{
				Price:  data[i].Price,
				Amount: float64(data[i].Size),
				ID:     data[i].ID,
			}
			if strings.EqualFold(data[i].Side, "Sell") {
				asks = append(asks, nItem)
				continue
			}
			bids = append(bids, nItem)
		}

		err := b.Websocket.Orderbook.Update(&buffer.Update{
			Bids:   bids,
			Asks:   asks,
			Pair:   p,
			Asset:  a,
			Action: buffer.Action(action),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *Bitmex) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channels := []string{bitmexWSOrderbookL2, bitmexWSTrade}
	subscriptions := []stream.ChannelSubscription{
		{
			Channel: bitmexWSAnnouncement,
		},
	}

	assets := b.GetAssetTypes(true)
	for x := range assets {
		contracts, err := b.GetEnabledPairs(assets[x])
		if err != nil {
			return nil, err
		}
		for y := range contracts {
			for z := range channels {
				if assets[x] == asset.Index && channels[z] == bitmexWSOrderbookL2 {
					// There are no L2 orderbook for index assets
					continue
				}
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[z] + ":" + contracts[y].String(),
					Currency: contracts[y],
					Asset:    assets[x],
				})
			}
		}
	}
	return subscriptions, nil
}

// GenerateAuthenticatedSubscriptions Adds authenticated subscriptions to websocket to be handled by ManageSubscriptions()
func (b *Bitmex) GenerateAuthenticatedSubscriptions() ([]stream.ChannelSubscription, error) {
	if !b.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, nil
	}
	contracts, err := b.GetEnabledPairs(asset.PerpetualContract)
	if err != nil {
		return nil, err
	}
	channels := []string{bitmexWSExecution,
		bitmexWSPosition,
	}
	subscriptions := []stream.ChannelSubscription{
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
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channels[i] + ":" + contracts[j].String(),
				Currency: contracts[j],
				Asset:    asset.PerpetualContract,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe subscribes to a websocket channel
func (b *Bitmex) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var subscriber WebsocketRequest
	subscriber.Command = "subscribe"

	for i := range channelsToSubscribe {
		subscriber.Arguments = append(subscriber.Arguments,
			channelsToSubscribe[i].Channel)
	}
	err := b.Websocket.Conn.SendJSONMessage(subscriber)
	if err != nil {
		return err
	}
	b.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bitmex) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var unsubscriber WebsocketRequest
	unsubscriber.Command = "unsubscribe"

	for i := range channelsToUnsubscribe {
		unsubscriber.Arguments = append(unsubscriber.Arguments,
			channelsToUnsubscribe[i].Channel)
	}
	err := b.Websocket.Conn.SendJSONMessage(unsubscriber)
	if err != nil {
		return err
	}
	b.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe...)
	return nil
}

// WebsocketSendAuth sends an authenticated subscription
func (b *Bitmex) websocketSendAuth(ctx context.Context) error {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}
	b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	timestamp := time.Now().Add(time.Hour * 1).Unix()
	newTimestamp := strconv.FormatInt(timestamp, 10)
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte("GET/realtime"+newTimestamp),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}
	signature := crypto.HexEncodeToString(hmac)

	var sendAuth WebsocketRequest
	sendAuth.Command = "authKeyExpires"
	sendAuth.Arguments = append(sendAuth.Arguments, creds.Key, timestamp,
		signature)
	err = b.Websocket.Conn.SendJSONMessage(sendAuth)
	if err != nil {
		b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}
