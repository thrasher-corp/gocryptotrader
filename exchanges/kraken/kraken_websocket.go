package kraken

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// List of all websocket channels to subscribe to
const (
	krakenWSURL              = "wss://ws.kraken.com"
	krakenAuthWSURL          = "wss://ws-auth.kraken.com"
	krakenWSSandboxURL       = "wss://sandbox.kraken.com"
	krakenWSSupportedVersion = "1.4.0"
	// WS endpoints
	krakenWsHeartbeat            = "heartbeat"
	krakenWsSystemStatus         = "systemStatus"
	krakenWsSubscribe            = "subscribe"
	krakenWsSubscriptionStatus   = "subscriptionStatus"
	krakenWsUnsubscribe          = "unsubscribe"
	krakenWsTicker               = "ticker"
	krakenWsOHLC                 = "ohlc"
	krakenWsTrade                = "trade"
	krakenWsSpread               = "spread"
	krakenWsOrderbook            = "book"
	krakenWsOwnTrades            = "ownTrades"
	krakenWsOpenOrders           = "openOrders"
	krakenWsAddOrder             = "addOrder"
	krakenWsCancelOrder          = "cancelOrder"
	krakenWsCancelAll            = "cancelAll"
	krakenWsAddOrderStatus       = "addOrderStatus"
	krakenWsCancelOrderStatus    = "cancelOrderStatus"
	krakenWsCancelAllOrderStatus = "cancelAllStatus"
	krakenWsRateLimit            = 50
	krakenWsPingDelay            = time.Second * 27
	krakenWsOrderbookDepth       = 1000
)

// orderbookMutex Ensures if two entries arrive at once, only one can be
// processed at a time
var (
	subscriptionChannelPair []WebsocketChannelData
	authToken               string
	pingRequest             = WebsocketBaseEventRequest{Event: stream.Ping}
	m                       sync.Mutex
)

// Channels require a topic and a currency
// Format [[ticker,but-t4u],[orderbook,nce-btt]]
var defaultSubscribedChannels = []string{
	krakenWsTicker,
	krakenWsTrade,
	krakenWsOrderbook,
	krakenWsOHLC,
	krakenWsSpread}
var authenticatedChannels = []string{krakenWsOwnTrades, krakenWsOpenOrders}

var cancelOrdersStatusMutex sync.Mutex
var cancelOrdersStatus = make(map[int64]*struct {
	Total        int    // total count of orders in wsCancelOrders request
	Successful   int    // numbers of Successfully canceled orders in wsCancelOrders request
	Unsuccessful int    // numbers of Unsuccessfully canceled orders in wsCancelOrders request
	Error        string // if at least one of requested order return fail, store error here
})

// WsConnect initiates a websocket connection
func (k *Kraken) WsConnect() error {
	if !k.Websocket.IsEnabled() || !k.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	err := k.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	comms := make(chan stream.Response)
	go k.wsReadData(comms)
	go k.wsFunnelConnectionData(k.Websocket.Conn, comms)

	if k.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		authToken, err = k.GetWebsocketToken()
		if err != nil {
			k.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys,
				"%v - authentication failed: %v\n",
				k.Name,
				err)
		} else {
			err = k.Websocket.AuthConn.Dial(&dialer, http.Header{})
			if err != nil {
				k.Websocket.SetCanUseAuthenticatedEndpoints(false)
				log.Errorf(log.ExchangeSys,
					"%v - failed to connect to authenticated endpoint: %v\n",
					k.Name,
					err)
			} else {
				go k.wsFunnelConnectionData(k.Websocket.AuthConn, comms)
				err = k.wsAuthPingHandler()
				if err != nil {
					log.Errorf(log.ExchangeSys,
						"%v - failed setup ping handler for auth connection. Websocket may disconnect unexpectedly. %v\n",
						k.Name,
						err)
				}
			}
		}
	}

	err = k.wsPingHandler()
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%v - failed setup ping handler. Websocket may disconnect unexpectedly. %v\n",
			k.Name,
			err)
	}
	return nil
}

// wsFunnelConnectionData funnels both auth and public ws data into one manageable place
func (k *Kraken) wsFunnelConnectionData(ws stream.Connection, comms chan stream.Response) {
	k.Websocket.Wg.Add(1)
	defer k.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		comms <- resp
	}
}

// wsReadData receives and passes on websocket messages for processing
func (k *Kraken) wsReadData(comms chan stream.Response) {
	k.Websocket.Wg.Add(1)
	defer k.Websocket.Wg.Done()

	for {
		select {
		case <-k.Websocket.ShutdownC:
			return
		case resp := <-comms:
			err := k.wsHandleData(resp.Raw)
			if err != nil {
				k.Websocket.DataHandler <- fmt.Errorf("%s - unhandled websocket data: %v",
					k.Name,
					err)
			}
		}
	}
}

// awaitForCancelOrderResponses used to wait until all responses will received for appropriate CancelOrder request
// success param = was the response from Kraken successful or not
func isAwaitingCancelOrderResponses(requestID int64, success bool) bool {
	cancelOrdersStatusMutex.Lock()
	if stat, ok := cancelOrdersStatus[requestID]; ok {
		if success {
			cancelOrdersStatus[requestID].Successful++
		} else {
			cancelOrdersStatus[requestID].Unsuccessful++
		}

		if stat.Successful+stat.Unsuccessful != stat.Total {
			cancelOrdersStatusMutex.Unlock()
			return true
		}
	}
	cancelOrdersStatusMutex.Unlock()
	return false
}

func (k *Kraken) wsHandleData(respRaw []byte) error {
	if strings.HasPrefix(string(respRaw), "[") {
		var dataResponse WebsocketDataResponse
		err := json.Unmarshal(respRaw, &dataResponse)
		if err != nil {
			return err
		}
		if _, ok := dataResponse[0].(float64); ok {
			err = k.wsReadDataResponse(dataResponse)
			if err != nil {
				return err
			}
		}
		if _, ok := dataResponse[1].(string); ok {
			err = k.wsHandleAuthDataResponse(dataResponse)
			if err != nil {
				return err
			}
		}
	} else {
		var eventResponse map[string]interface{}
		err := json.Unmarshal(respRaw, &eventResponse)
		if err != nil {
			return fmt.Errorf("%s - err %s could not parse websocket data: %s",
				k.Name,
				err,
				respRaw)
		}
		if event, ok := eventResponse["event"]; ok {
			switch event {
			case stream.Pong, krakenWsHeartbeat:
				return nil
			case krakenWsCancelOrderStatus:
				var status WsCancelOrderResponse
				err := json.Unmarshal(respRaw, &status)
				if err != nil {
					return fmt.Errorf("%s - err %s unable to parse WsCancelOrderResponse: %s",
						k.Name,
						err,
						respRaw)
				}

				success := true
				if status.Status == "error" {
					success = false
					cancelOrdersStatusMutex.Lock()
					if _, ok := cancelOrdersStatus[status.RequestID]; ok {
						if cancelOrdersStatus[status.RequestID].Error == "" { // save the first error, if any
							cancelOrdersStatus[status.RequestID].Error = status.ErrorMessage
						}
					}
					cancelOrdersStatusMutex.Unlock()
				}

				if isAwaitingCancelOrderResponses(status.RequestID, success) {
					return nil
				}

				// all responses handled, return results stored in cancelOrdersStatus
				if status.RequestID > 0 && !k.Websocket.Match.IncomingWithData(status.RequestID, respRaw) {
					return fmt.Errorf("can't send ws incoming data to Matched channel with RequestID: %d",
						status.RequestID)
				}
			case krakenWsCancelAllOrderStatus:
				var status WsCancelOrderResponse
				err := json.Unmarshal(respRaw, &status)
				if err != nil {
					return fmt.Errorf("%s - err %s unable to parse WsCancelOrderResponse: %s",
						k.Name,
						err,
						respRaw)
				}

				var isChannelExist bool
				if status.RequestID > 0 {
					isChannelExist = k.Websocket.Match.IncomingWithData(status.RequestID, respRaw)
				}

				if status.Status == "error" {
					return fmt.Errorf("%v Websocket status for RequestID %d: '%v'",
						k.Name,
						status.RequestID,
						status.ErrorMessage)
				}

				if !isChannelExist && status.RequestID > 0 {
					return fmt.Errorf("can't send ws incoming data to Matched channel with RequestID: %d",
						status.RequestID)
				}
			case krakenWsSystemStatus:
				var systemStatus wsSystemStatus
				err := json.Unmarshal(respRaw, &systemStatus)
				if err != nil {
					return fmt.Errorf("%s - err %s unable to parse system status response: %s",
						k.Name,
						err,
						respRaw)
				}
				if systemStatus.Status != "online" {
					k.Websocket.DataHandler <- fmt.Errorf("%v Websocket status '%v'",
						k.Name,
						systemStatus.Status)
				}
				if systemStatus.Version > krakenWSSupportedVersion {
					log.Warnf(log.ExchangeSys,
						"%v New version of Websocket API released. Was %v Now %v",
						k.Name,
						krakenWSSupportedVersion,
						systemStatus.Version)
				}
			case krakenWsAddOrderStatus:
				var status WsAddOrderResponse
				err := json.Unmarshal(respRaw, &status)
				if err != nil {
					return fmt.Errorf("%s - err %s unable to parse add order response: %s",
						k.Name,
						err,
						respRaw)
				}

				var isChannelExist bool
				if status.RequestID > 0 {
					isChannelExist = k.Websocket.Match.IncomingWithData(status.RequestID, respRaw)
				}

				if status.Status == "error" {
					return fmt.Errorf("%v Websocket status for RequestID %d: '%v'",
						k.Name,
						status.RequestID,
						status.ErrorMessage)
				}

				k.Websocket.DataHandler <- &order.Detail{
					Exchange: k.Name,
					ID:       status.TransactionID,
					Status:   order.New,
				}

				if !isChannelExist && status.RequestID > 0 {
					return fmt.Errorf("can't send ws incoming data to Matched channel with RequestID: %d",
						status.RequestID)
				}
			case krakenWsSubscriptionStatus:
				var sub wsSubscription
				err := json.Unmarshal(respRaw, &sub)
				if err != nil {
					return fmt.Errorf("%s - err %s unable to parse subscription response: %s",
						k.Name,
						err,
						respRaw)
				}
				if sub.Status != "subscribed" && sub.Status != "unsubscribed" {
					return fmt.Errorf("%v %v %v",
						k.Name,
						sub.RequestID,
						sub.ErrorMessage)
				}
				k.addNewSubscriptionChannelData(&sub)
				if sub.RequestID > 0 {
					k.Websocket.Match.IncomingWithData(sub.RequestID, respRaw)
				}
			default:
				k.Websocket.DataHandler <- stream.UnhandledMessageWarning{
					Message: k.Name + stream.UnhandledMessage + string(respRaw),
				}
			}
			return nil
		}
	}
	return nil
}

// wsPingHandler sends a message "ping" every 27 to maintain the connection to the websocket
func (k *Kraken) wsPingHandler() error {
	message, err := json.Marshal(pingRequest)
	if err != nil {
		return err
	}
	k.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Message:     message,
		Delay:       krakenWsPingDelay,
		MessageType: websocket.TextMessage,
	})
	return nil
}

// wsAuthPingHandler sends a message "ping" every 27 to maintain the connection to the websocket
func (k *Kraken) wsAuthPingHandler() error {
	message, err := json.Marshal(pingRequest)
	if err != nil {
		return err
	}
	k.Websocket.AuthConn.SetupPingHandler(stream.PingHandler{
		Message:     message,
		Delay:       krakenWsPingDelay,
		MessageType: websocket.TextMessage,
	})
	return nil
}

// wsReadDataResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) wsReadDataResponse(response WebsocketDataResponse) error {
	if cID, ok := response[0].(float64); ok {
		channelID := int64(cID)
		channelData, err := getSubscriptionChannelData(channelID)
		if err != nil {
			return err
		}
		switch channelData.Subscription {
		case krakenWsTicker:
			t, ok := response[1].(map[string]interface{})
			if !ok {
				return errors.New("received invalid ticker data")
			}
			return k.wsProcessTickers(&channelData, t)
		case krakenWsOHLC:
			o, ok := response[1].([]interface{})
			if !ok {
				return errors.New("received invalid OHLCV data")
			}
			return k.wsProcessCandles(&channelData, o)
		case krakenWsOrderbook:
			ob, ok := response[1].(map[string]interface{})
			if !ok {
				return errors.New("received invalid orderbook data")
			}

			if len(response) == 5 {
				ob2, okob2 := response[2].(map[string]interface{})
				if !okob2 {
					return errors.New("received invalid orderbook data")
				}

				// Squish both maps together to process
				for k, v := range ob2 {
					if _, ok := ob[k]; ok {
						return errors.New("cannot merge maps, conflict is present")
					}
					ob[k] = v
				}
			}
			return k.wsProcessOrderBook(&channelData, ob)
		case krakenWsSpread:
			s, ok := response[1].([]interface{})
			if !ok {
				return errors.New("received invalid spread data")
			}
			k.wsProcessSpread(&channelData, s)
		case krakenWsTrade:
			t, ok := response[1].([]interface{})
			if !ok {
				return errors.New("received invalid trade data")
			}
			return k.wsProcessTrades(&channelData, t)
		default:
			return fmt.Errorf("%s received unidentified data for subscription %s: %+v",
				k.Name,
				channelData.Subscription,
				response)
		}
	}

	return nil
}

func (k *Kraken) wsHandleAuthDataResponse(response WebsocketDataResponse) error {
	if chName, ok := response[1].(string); ok {
		switch chName {
		case krakenWsOwnTrades:
			return k.wsProcessOwnTrades(response[0])
		case krakenWsOpenOrders:
			return k.wsProcessOpenOrders(response[0])
		default:
			return fmt.Errorf("%v Unidentified websocket data received: %+v",
				k.Name, response)
		}
	}
	return nil
}

func (k *Kraken) wsProcessOwnTrades(ownOrders interface{}) error {
	if data, ok := ownOrders.([]interface{}); ok {
		for i := range data {
			trades, err := json.Marshal(data[i])
			if err != nil {
				return err
			}
			var result map[string]*WsOwnTrade
			err = json.Unmarshal(trades, &result)
			if err != nil {
				return err
			}
			for key, val := range result {
				oSide, err := order.StringToOrderSide(val.Type)
				if err != nil {
					k.Websocket.DataHandler <- order.ClassificationError{
						Exchange: k.Name,
						OrderID:  key,
						Err:      err,
					}
				}
				oType, err := order.StringToOrderType(val.OrderType)
				if err != nil {
					k.Websocket.DataHandler <- order.ClassificationError{
						Exchange: k.Name,
						OrderID:  key,
						Err:      err,
					}
				}
				trade := order.TradeHistory{
					Price:     val.Price,
					Amount:    val.Vol,
					Fee:       val.Fee,
					Exchange:  k.Name,
					TID:       key,
					Type:      oType,
					Side:      oSide,
					Timestamp: convert.TimeFromUnixTimestampDecimal(val.Time),
				}
				k.Websocket.DataHandler <- &order.Detail{
					Exchange: k.Name,
					ID:       val.OrderTransactionID,
					Trades:   []order.TradeHistory{trade},
				}
			}
		}
		return nil
	}
	return errors.New(k.Name + " - Invalid own trades data")
}

func (k *Kraken) wsProcessOpenOrders(ownOrders interface{}) error {
	if data, ok := ownOrders.([]interface{}); ok {
		for i := range data {
			orders, err := json.Marshal(data[i])
			if err != nil {
				return err
			}
			var result map[string]*WsOpenOrder
			err = json.Unmarshal(orders, &result)
			if err != nil {
				return err
			}
			for key, val := range result {
				var oStatus order.Status
				oStatus, err = order.StringToOrderStatus(val.Status)
				if err != nil {
					k.Websocket.DataHandler <- order.ClassificationError{
						Exchange: k.Name,
						OrderID:  key,
						Err:      err,
					}
				}
				if val.Description.Price > 0 {
					oSide, err := order.StringToOrderSide(val.Description.Type)
					if err != nil {
						k.Websocket.DataHandler <- order.ClassificationError{
							Exchange: k.Name,
							OrderID:  key,
							Err:      err,
						}
					}
					if strings.Contains(val.Description.Order, "sell") {
						oSide = order.Sell
					}
					oType, err := order.StringToOrderType(val.Description.Type)
					if err != nil {
						k.Websocket.DataHandler <- order.ClassificationError{
							Exchange: k.Name,
							OrderID:  key,
							Err:      err,
						}
					}

					p, err := currency.NewPairFromString(val.Description.Pair)
					if err != nil {
						k.Websocket.DataHandler <- order.ClassificationError{
							Exchange: k.Name,
							OrderID:  key,
							Err:      err,
						}
					}

					var a asset.Item
					a, err = k.GetPairAssetType(p)
					if err != nil {
						return err
					}
					k.Websocket.DataHandler <- &order.Modify{
						Leverage:        val.Description.Leverage,
						Price:           val.Price,
						Amount:          val.Volume,
						LimitPriceUpper: val.LimitPrice,
						ExecutedAmount:  val.ExecutedVolume,
						RemainingAmount: val.Volume - val.ExecutedVolume,
						Fee:             val.Fee,
						Exchange:        k.Name,
						ID:              key,
						Type:            oType,
						Side:            oSide,
						Status:          oStatus,
						AssetType:       a,
						Date:            convert.TimeFromUnixTimestampDecimal(val.OpenTime),
						Pair:            p,
					}
				} else {
					k.Websocket.DataHandler <- &order.Modify{
						Exchange: k.Name,
						ID:       key,
						Status:   oStatus,
					}
				}
			}
		}
		return nil
	}
	return errors.New(k.Name + " - Invalid own trades data")
}

// addNewSubscriptionChannelData stores channel ids, pairs and subscription types to an array
// allowing correlation between subscriptions and returned data
func (k *Kraken) addNewSubscriptionChannelData(response *wsSubscription) {
	// We change the / to - to maintain compatibility with REST/config
	var pair, fPair currency.Pair
	if response.Pair != "" {
		var err error
		pair, err = currency.NewPairFromString(response.Pair)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s exchange error: %s", k.Name, err)
			return
		}
		fPair, err = k.FormatExchangeCurrency(pair, asset.Spot)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s exchange error: %s", k.Name, err)
			return
		}
	}
	m.Lock()
	defer m.Unlock()
	subscriptionChannelPair = append(subscriptionChannelPair, WebsocketChannelData{
		Subscription: response.Subscription.Name,
		Pair:         fPair,
		ChannelID:    response.ChannelID,
	})
}

// getSubscriptionChannelData retrieves WebsocketChannelData based on response ID
func getSubscriptionChannelData(id int64) (WebsocketChannelData, error) {
	m.Lock()
	defer m.Unlock()
	for i := range subscriptionChannelPair {
		if subscriptionChannelPair[i].ChannelID == nil {
			continue
		}
		if id == *subscriptionChannelPair[i].ChannelID {
			return subscriptionChannelPair[i], nil
		}
	}
	return WebsocketChannelData{},
		fmt.Errorf("could not get subscription data for id %d", id)
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (k *Kraken) wsProcessTickers(channelData *WebsocketChannelData, data map[string]interface{}) error {
	closePrice, err := strconv.ParseFloat(data["c"].([]interface{})[0].(string), 64)
	if err != nil {
		return err
	}
	openPrice, err := strconv.ParseFloat(data["o"].([]interface{})[0].(string), 64)
	if err != nil {
		return err
	}
	highPrice, err := strconv.ParseFloat(data["h"].([]interface{})[0].(string), 64)
	if err != nil {
		return err
	}
	lowPrice, err := strconv.ParseFloat(data["l"].([]interface{})[0].(string), 64)
	if err != nil {
		return err
	}
	quantity, err := strconv.ParseFloat(data["v"].([]interface{})[0].(string), 64)
	if err != nil {
		return err
	}
	ask, err := strconv.ParseFloat(data["a"].([]interface{})[0].(string), 64)
	if err != nil {
		return err
	}
	bid, err := strconv.ParseFloat(data["b"].([]interface{})[0].(string), 64)
	if err != nil {
		return err
	}

	k.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: k.Name,
		Open:         openPrice,
		Close:        closePrice,
		Volume:       quantity,
		High:         highPrice,
		Low:          lowPrice,
		Bid:          bid,
		Ask:          ask,
		AssetType:    asset.Spot,
		Pair:         channelData.Pair,
	}
	return nil
}

// wsProcessSpread converts spread/orderbook data and sends it to the datahandler
func (k *Kraken) wsProcessSpread(channelData *WebsocketChannelData, data []interface{}) {
	bestBid := data[0].(string)
	bestAsk := data[1].(string)
	timeData, err := strconv.ParseFloat(data[2].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- err
		return
	}

	bidVolume := data[3].(string)
	askVolume := data[4].(string)
	if k.Verbose {
		log.Debugf(log.ExchangeSys,
			"%v Spread data for '%v' received. Best bid: '%v' Best ask: '%v' Time: '%v', Bid volume '%v', Ask volume '%v'",
			k.Name,
			channelData.Pair,
			bestBid,
			bestAsk,
			convert.TimeFromUnixTimestampDecimal(timeData),
			bidVolume,
			askVolume)
	}
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (k *Kraken) wsProcessTrades(channelData *WebsocketChannelData, data []interface{}) error {
	if !k.IsSaveTradeDataEnabled() {
		return nil
	}
	var trades []trade.Data
	for i := range data {
		t, ok := data[i].([]interface{})
		if !ok {
			return errors.New("unidentified trade data received")
		}
		timeData, err := strconv.ParseFloat(t[2].(string), 64)
		if err != nil {
			return err
		}

		price, err := strconv.ParseFloat(t[0].(string), 64)
		if err != nil {
			return err
		}

		amount, err := strconv.ParseFloat(t[1].(string), 64)
		if err != nil {
			return err
		}
		var tSide = order.Buy
		if t[3].(string) == "s" {
			tSide = order.Sell
		}

		trades = append(trades, trade.Data{
			AssetType:    asset.Spot,
			CurrencyPair: channelData.Pair,
			Exchange:     k.Name,
			Price:        price,
			Amount:       amount,
			Timestamp:    convert.TimeFromUnixTimestampDecimal(timeData),
			Side:         tSide,
		})
	}
	return trade.AddTradesToBuffer(k.Name, trades...)
}

// wsProcessOrderBook determines if the orderbook data is partial or update
// Then sends to appropriate fun
func (k *Kraken) wsProcessOrderBook(channelData *WebsocketChannelData, data map[string]interface{}) error {
	askSnapshot, askSnapshotExists := data["as"].([]interface{})
	bidSnapshot, bidSnapshotExists := data["bs"].([]interface{})
	if askSnapshotExists || bidSnapshotExists {
		err := k.wsProcessOrderBookPartial(channelData, askSnapshot, bidSnapshot)
		if err != nil {
			return err
		}
	} else {
		askData, asksExist := data["a"].([]interface{})
		bidData, bidsExist := data["b"].([]interface{})
		checksum, ok := data["c"].(string)
		if !ok {
			return fmt.Errorf("could not process orderbook update checksum not found")
		}
		if asksExist || bidsExist {
			k.wsRequestMtx.Lock()
			defer k.wsRequestMtx.Unlock()
			err := k.wsProcessOrderBookUpdate(channelData, askData, bidData, checksum)
			if err != nil {
				go func(resub *stream.ChannelSubscription) {
					// This was locking the main websocket reader routine and a
					// backlog occurred. So put this into it's own go routine.
					errResub := k.Websocket.ResubscribeToChannel(resub)
					if errResub != nil {
						log.Errorf(log.WebsocketMgr,
							"resubscription failure for %v: %v",
							resub,
							errResub)
					}
				}(&stream.ChannelSubscription{
					Channel:  krakenWsOrderbook,
					Currency: channelData.Pair,
					Asset:    asset.Spot,
				})
				return err
			}
		}
	}
	return nil
}

// wsProcessOrderBookPartial creates a new orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookPartial(channelData *WebsocketChannelData, askData, bidData []interface{}) error {
	base := orderbook.Base{
		Pair:            channelData.Pair,
		Asset:           asset.Spot,
		VerifyOrderbook: k.CanVerifyOrderbook,
	}
	// Kraken ob data is timestamped per price, GCT orderbook data is
	// timestamped per entry using the highest last update time, we can attempt
	// to respect both within a reasonable degree
	var highestLastUpdate time.Time
	for i := range askData {
		asks := askData[i].([]interface{})
		price, err := strconv.ParseFloat(asks[0].(string), 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(asks[1].(string), 64)
		if err != nil {
			return err
		}
		base.Asks = append(base.Asks, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
		timeData, err := strconv.ParseFloat(asks[2].(string), 64)
		if err != nil {
			return err
		}
		askUpdatedTime := convert.TimeFromUnixTimestampDecimal(timeData)
		if highestLastUpdate.Before(askUpdatedTime) {
			highestLastUpdate = askUpdatedTime
		}
	}

	for i := range bidData {
		bids := bidData[i].([]interface{})
		price, err := strconv.ParseFloat(bids[0].(string), 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(bids[1].(string), 64)
		if err != nil {
			return err
		}
		base.Bids = append(base.Bids, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
		timeData, err := strconv.ParseFloat(bids[2].(string), 64)
		if err != nil {
			return err
		}
		bidUpdateTime := convert.TimeFromUnixTimestampDecimal(timeData)
		if highestLastUpdate.Before(bidUpdateTime) {
			highestLastUpdate = bidUpdateTime
		}
	}
	base.LastUpdated = highestLastUpdate
	base.Exchange = k.Name
	return k.Websocket.Orderbook.LoadSnapshot(&base)
}

// wsProcessOrderBookUpdate updates an orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookUpdate(channelData *WebsocketChannelData, askData, bidData []interface{}, checksum string) error {
	update := buffer.Update{
		Asset:    asset.Spot,
		Pair:     channelData.Pair,
		MaxDepth: krakenWsOrderbookDepth,
	}

	// Calculating checksum requires incoming decimal place checks for both
	// price and amount as there is no set standard between currency pairs. This
	// is calculated per update as opposed to snapshot because changes to
	// decimal amounts could occur at any time.
	var priceDP, amtDP int
	var highestLastUpdate time.Time
	// Ask data is not always sent
	for i := range askData {
		asks := askData[i].([]interface{})

		priceStr, ok := asks[0].(string)
		if !ok {
			return errors.New("price type assertion failure")
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return err
		}

		amountStr, ok := asks[1].(string)
		if !ok {
			return errors.New("amount type assertion failure")
		}

		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return err
		}

		update.Asks = append(update.Asks, orderbook.Item{
			Amount: amount,
			Price:  price,
		})

		timeStr, ok := asks[2].(string)
		if !ok {
			return errors.New("time type assertion failure")
		}

		timeData, err := strconv.ParseFloat(timeStr, 64)
		if err != nil {
			return err
		}

		askUpdatedTime := convert.TimeFromUnixTimestampDecimal(timeData)
		if highestLastUpdate.Before(askUpdatedTime) {
			highestLastUpdate = askUpdatedTime
		}

		if i == len(askData)-1 {
			pSplit := strings.Split(priceStr, ".")
			if len(pSplit) != 2 {
				return errors.New("incorrect decimal data returned for price")
			}

			priceDP = len(pSplit[1])
			aSplit := strings.Split(amountStr, ".")
			if len(aSplit) != 2 {
				return errors.New("incorrect decimal data returned for amount")
			}

			amtDP = len(aSplit[1])
		}
	}

	// Bid data is not always sent
	for i := range bidData {
		bids := bidData[i].([]interface{})

		priceStr, ok := bids[0].(string)
		if !ok {
			return errors.New("price type assertion failure")
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return err
		}

		amountStr, ok := bids[1].(string)
		if !ok {
			return errors.New("amount type assertion failure")
		}

		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return err
		}

		update.Bids = append(update.Bids, orderbook.Item{
			Amount: amount,
			Price:  price,
		})

		timeStr, ok := bids[2].(string)
		if !ok {
			return errors.New("time type assertion failure")
		}

		timeData, err := strconv.ParseFloat(timeStr, 64)
		if err != nil {
			return err
		}

		bidUpdatedTime := convert.TimeFromUnixTimestampDecimal(timeData)
		if highestLastUpdate.Before(bidUpdatedTime) {
			highestLastUpdate = bidUpdatedTime
		}

		if i == len(bidData)-1 {
			pSplit := strings.Split(priceStr, ".")
			if len(pSplit) != 2 {
				return errors.New("incorrect decimal data returned for price")
			}

			priceDP = len(pSplit[1])
			aSplit := strings.Split(amountStr, ".")
			if len(aSplit) != 2 {
				return errors.New("incorrect decimal data returned for amount")
			}

			amtDP = len(aSplit[1])
		}
	}
	update.UpdateTime = highestLastUpdate
	err := k.Websocket.Orderbook.Update(&update)
	if err != nil {
		return err
	}

	book, err := k.Websocket.Orderbook.GetOrderbook(channelData.Pair, asset.Spot)
	if err != nil {
		return fmt.Errorf("cannot calculate websocket checksum: book not found for %s %s %w",
			channelData.Pair,
			asset.Spot,
			err)
	}

	token, err := strconv.ParseInt(checksum, 10, 64)
	if err != nil {
		return err
	}

	return validateCRC32(book, uint32(token), priceDP, amtDP)
}

func validateCRC32(b *orderbook.Base, token uint32, decPrice, decAmount int) error {
	if len(b.Asks) < 10 || len(b.Bids) < 10 {
		return fmt.Errorf("%s %s insufficient bid and asks to calculate checksum",
			b.Pair,
			b.Asset)
	}

	if decPrice == 0 || decAmount == 0 {
		return fmt.Errorf("%s %s trailing decimal count not calculated",
			b.Pair,
			b.Asset)
	}

	var checkStr strings.Builder
	for i := 0; i < 10; i++ {
		priceStr := trim(strconv.FormatFloat(b.Asks[i].Price, 'f', decPrice, 64))
		checkStr.WriteString(priceStr)
		amountStr := trim(strconv.FormatFloat(b.Asks[i].Amount, 'f', decAmount, 64))
		checkStr.WriteString(amountStr)
	}

	for i := 0; i < 10; i++ {
		priceStr := trim(strconv.FormatFloat(b.Bids[i].Price, 'f', decPrice, 64))
		checkStr.WriteString(priceStr)
		amountStr := trim(strconv.FormatFloat(b.Bids[i].Amount, 'f', decAmount, 64))
		checkStr.WriteString(amountStr)
	}

	if check := crc32.ChecksumIEEE([]byte(checkStr.String())); check != token {
		return fmt.Errorf("%s %s invalid checksum %d, expected %d",
			b.Pair,
			b.Asset,
			check,
			token)
	}
	return nil
}

// trim removes '.' and prefixed '0' from subsequent string
func trim(s string) string {
	s = strings.Replace(s, ".", "", 1)
	s = strings.TrimLeft(s, "0")
	return s
}

// wsProcessCandles converts candle data and sends it to the data handler
func (k *Kraken) wsProcessCandles(channelData *WebsocketChannelData, data []interface{}) error {
	startTime, err := strconv.ParseFloat(data[0].(string), 64)
	if err != nil {
		return err
	}

	endTime, err := strconv.ParseFloat(data[1].(string), 64)
	if err != nil {
		return err
	}

	openPrice, err := strconv.ParseFloat(data[2].(string), 64)
	if err != nil {
		return err
	}

	highPrice, err := strconv.ParseFloat(data[3].(string), 64)
	if err != nil {
		return err
	}

	lowPrice, err := strconv.ParseFloat(data[4].(string), 64)
	if err != nil {
		return err
	}

	closePrice, err := strconv.ParseFloat(data[5].(string), 64)
	if err != nil {
		return err
	}

	volume, err := strconv.ParseFloat(data[7].(string), 64)
	if err != nil {
		return err
	}

	k.Websocket.DataHandler <- stream.KlineData{
		AssetType: asset.Spot,
		Pair:      channelData.Pair,
		Timestamp: time.Now(),
		Exchange:  k.Name,
		StartTime: convert.TimeFromUnixTimestampDecimal(startTime),
		CloseTime: convert.TimeFromUnixTimestampDecimal(endTime),
		// Candles are sent every 60 seconds
		Interval:   "60",
		HighPrice:  highPrice,
		LowPrice:   lowPrice,
		OpenPrice:  openPrice,
		ClosePrice: closePrice,
		Volume:     volume,
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (k *Kraken) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	enabledCurrencies, err := k.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions []stream.ChannelSubscription
	for i := range defaultSubscribedChannels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = "/"
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  defaultSubscribedChannels[i],
				Currency: enabledCurrencies[j],
				Asset:    asset.Spot,
			})
		}
	}
	if k.Websocket.CanUseAuthenticatedEndpoints() {
		for i := range authenticatedChannels {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: authenticatedChannels[i],
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (k *Kraken) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var subscriptions = make(map[string]*[]WebsocketSubscriptionEventRequest)
channels:
	for i := range channelsToSubscribe {
		s, ok := subscriptions[channelsToSubscribe[i].Channel]
		if !ok {
			s = &[]WebsocketSubscriptionEventRequest{}
			subscriptions[channelsToSubscribe[i].Channel] = s
		}

		for j := range *s {
			if len((*s)[j].Channels) >= 20 {
				// Batch outgoing subscriptions as there are limitations on the
				// orderbook snapshots
				continue
			}
			(*s)[j].Pairs = append((*s)[j].Pairs, channelsToSubscribe[i].Currency.String())
			(*s)[j].Channels = append((*s)[j].Channels, channelsToSubscribe[i])
			continue channels
		}

		id := k.Websocket.Conn.GenerateMessageID(false)
		outbound := WebsocketSubscriptionEventRequest{
			Event:     krakenWsSubscribe,
			RequestID: id,
			Subscription: WebsocketSubscriptionData{
				Name: channelsToSubscribe[i].Channel,
			},
		}
		if channelsToSubscribe[i].Channel == "book" {
			outbound.Subscription.Depth = 1000
		}
		if !channelsToSubscribe[i].Currency.IsEmpty() {
			outbound.Pairs = []string{channelsToSubscribe[i].Currency.String()}
		}
		if common.StringDataContains(authenticatedChannels, channelsToSubscribe[i].Channel) {
			outbound.Subscription.Token = authToken
		}

		outbound.Channels = append(outbound.Channels, channelsToSubscribe[i])
		*s = append(*s, outbound)
	}

	var errs common.Errors
	for subType, subs := range subscriptions {
		for i := range *subs {
			if common.StringDataContains(authenticatedChannels, (*subs)[i].Subscription.Name) {
				_, err := k.Websocket.AuthConn.SendMessageReturnResponse((*subs)[i].RequestID, (*subs)[i])
				if err != nil {
					errs = append(errs, err)
					continue
				}
				k.Websocket.AddSuccessfulSubscriptions((*subs)[i].Channels...)
				continue
			}
			if subType == "book" {
				// There is an undocumented subscription limit that is present
				// on websocket orderbooks, to subscribe to the channel while
				// actually receiving the snapshots a rudimentary sleep is
				// imposed and requests are batched in allotments of 20 items.
				time.Sleep(time.Second)
			}
			_, err := k.Websocket.Conn.SendMessageReturnResponse((*subs)[i].RequestID, (*subs)[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			k.Websocket.AddSuccessfulSubscriptions((*subs)[i].Channels...)
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (k *Kraken) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var unsubs []WebsocketSubscriptionEventRequest
channels:
	for x := range channelsToUnsubscribe {
		for y := range unsubs {
			if unsubs[y].Subscription.Name == channelsToUnsubscribe[x].Channel {
				unsubs[y].Pairs = append(unsubs[y].Pairs,
					channelsToUnsubscribe[x].Currency.String())
				unsubs[y].Channels = append(unsubs[y].Channels,
					channelsToUnsubscribe[x])
				continue channels
			}
		}
		var depth int64
		if channelsToUnsubscribe[x].Channel == "book" {
			depth = krakenWsOrderbookDepth
		}

		var id int64
		if common.StringDataContains(authenticatedChannels, channelsToUnsubscribe[x].Channel) {
			id = k.Websocket.AuthConn.GenerateMessageID(false)
		} else {
			id = k.Websocket.Conn.GenerateMessageID(false)
		}

		unsub := WebsocketSubscriptionEventRequest{
			Event: krakenWsUnsubscribe,
			Pairs: []string{channelsToUnsubscribe[x].Currency.String()},
			Subscription: WebsocketSubscriptionData{
				Name:  channelsToUnsubscribe[x].Channel,
				Depth: depth,
			},
			RequestID: id,
		}
		if common.StringDataContains(authenticatedChannels, channelsToUnsubscribe[x].Channel) {
			unsub.Subscription.Token = authToken
		}
		unsub.Channels = append(unsub.Channels, channelsToUnsubscribe[x])
		unsubs = append(unsubs, unsub)
	}

	var errs common.Errors
	for i := range unsubs {
		if common.StringDataContains(authenticatedChannels, unsubs[i].Subscription.Name) {
			_, err := k.Websocket.AuthConn.SendMessageReturnResponse(unsubs[i].RequestID, unsubs[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			k.Websocket.RemoveSuccessfulUnsubscriptions(unsubs[i].Channels...)
			continue
		}

		_, err := k.Websocket.Conn.SendMessageReturnResponse(unsubs[i].RequestID, unsubs[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		k.Websocket.RemoveSuccessfulUnsubscriptions(unsubs[i].Channels...)
	}
	if errs != nil {
		return errs
	}
	return nil
}

// wsAddOrder creates an order, returned order ID if success
func (k *Kraken) wsAddOrder(request *WsAddOrderRequest) (string, error) {
	id := k.Websocket.AuthConn.GenerateMessageID(false)
	request.RequestID = id
	request.Event = krakenWsAddOrder
	request.Token = authToken
	jsonResp, err := k.Websocket.AuthConn.SendMessageReturnResponse(id, request)
	if err != nil {
		return "", err
	}
	var resp WsAddOrderResponse
	err = json.Unmarshal(jsonResp, &resp)
	if err != nil {
		return "", err
	}
	if resp.ErrorMessage != "" {
		return "", fmt.Errorf(k.Name + " - " + resp.ErrorMessage)
	}
	return resp.TransactionID, nil
}

// wsCancelOrders cancels one or more open orders passed in orderIDs param
func (k *Kraken) wsCancelOrders(orderIDs []string) error {
	id := k.Websocket.AuthConn.GenerateMessageID(false)
	request := WsCancelOrderRequest{
		Event:          krakenWsCancelOrder,
		Token:          authToken,
		TransactionIDs: orderIDs,
		RequestID:      id,
	}

	cancelOrdersStatus[id] = &struct {
		Total        int
		Successful   int
		Unsuccessful int
		Error        string
	}{
		Total: len(orderIDs),
	}

	defer delete(cancelOrdersStatus, id)

	_, err := k.Websocket.AuthConn.SendMessageReturnResponse(id, request)
	if err != nil {
		return err
	}

	successful := cancelOrdersStatus[id].Successful

	if cancelOrdersStatus[id].Error != "" || len(orderIDs) != successful { // strange Kraken logic ...
		var reason string
		if cancelOrdersStatus[id].Error != "" {
			reason = fmt.Sprintf(" Reason: %s", cancelOrdersStatus[id].Error)
		}
		return fmt.Errorf("%s cancelled %d out of %d orders.%s",
			k.Name, successful, len(orderIDs), reason)
	}
	return nil
}

// wsCancelAllOrders cancels all opened orders
// Returns number (count param) of affected orders or 0 if no open orders found
func (k *Kraken) wsCancelAllOrders() (*WsCancelOrderResponse, error) {
	id := k.Websocket.AuthConn.GenerateMessageID(false)
	request := WsCancelOrderRequest{
		Event:     krakenWsCancelAll,
		Token:     authToken,
		RequestID: id,
	}

	jsonResp, err := k.Websocket.AuthConn.SendMessageReturnResponse(id, request)
	if err != nil {
		return &WsCancelOrderResponse{}, err
	}
	var resp WsCancelOrderResponse
	err = json.Unmarshal(jsonResp, &resp)
	if err != nil {
		return &WsCancelOrderResponse{}, err
	}
	if resp.ErrorMessage != "" {
		return &WsCancelOrderResponse{}, fmt.Errorf(k.Name + " - " + resp.ErrorMessage)
	}
	return &resp, nil
}
