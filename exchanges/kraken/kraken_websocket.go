package kraken

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
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
	subscriptionChannelPair     []WebsocketChannelData
	authToken                   string
	pingRequest                 = WebsocketBaseEventRequest{Event: stream.Ping}
	m                           sync.Mutex
	errNoWebsocketOrderbookData = errors.New("no websocket orderbook data")
	errParsingWSField           = errors.New("error parsing WS field")
	errUnknownError             = errors.New("unknown error")
	errCancellingOrder          = errors.New("error cancelling order")
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

// WsConnect initiates a websocket connection
func (k *Kraken) WsConnect() error {
	if !k.Websocket.IsEnabled() || !k.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}

	var dialer websocket.Dialer
	err := k.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	comms := make(chan stream.Response)
	k.Websocket.Wg.Add(2)
	go k.wsReadData(comms)
	go k.wsFunnelConnectionData(k.Websocket.Conn, comms)

	if k.IsWebsocketAuthenticationSupported() {
		authToken, err = k.GetWebsocketToken(context.TODO())
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
				k.Websocket.SetCanUseAuthenticatedEndpoints(true)
				k.Websocket.Wg.Add(1)
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
	defer k.Websocket.Wg.Done()

	for {
		select {
		case <-k.Websocket.ShutdownC:
			select {
			case resp := <-comms:
				err := k.wsHandleData(resp.Raw)
				if err != nil {
					select {
					case k.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr,
							"%s websocket handle data error: %v",
							k.Name,
							err)
					}
				}
			default:
			}
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
			return fmt.Errorf("%s - err %s could not parse websocket data: %s", k.Name, err, respRaw)
		}
		if event, ok := eventResponse["event"]; ok {
			switch event {
			case stream.Pong, krakenWsHeartbeat:
				return nil
			case krakenWsCancelOrderStatus:
				id, err := jsonparser.GetInt(respRaw, "reqid")
				if err != nil {
					return fmt.Errorf("%w 'reqid': %w from message: %s", errParsingWSField, err, respRaw)
				}
				if !k.Websocket.Match.IncomingWithData(id, respRaw) {
					return fmt.Errorf("%v cancel order listener not found", id)
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
					OrderID:  status.TransactionID,
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
					OrderID:  val.OrderTransactionID,
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
				d := &order.Detail{
					Exchange:             k.Name,
					OrderID:              key,
					AverageExecutedPrice: val.AveragePrice,
					Amount:               val.Volume,
					LimitPriceUpper:      val.LimitPrice,
					ExecutedAmount:       val.ExecutedVolume,
					Fee:                  val.Fee,
					Date:                 convert.TimeFromUnixTimestampDecimal(val.OpenTime).Truncate(time.Microsecond),
					LastUpdated:          convert.TimeFromUnixTimestampDecimal(val.LastUpdated).Truncate(time.Microsecond),
				}

				if val.Status != "" {
					if s, err := order.StringToOrderStatus(val.Status); err != nil {
						k.Websocket.DataHandler <- order.ClassificationError{
							Exchange: k.Name,
							OrderID:  key,
							Err:      err,
						}
					} else {
						d.Status = s
					}
				}

				if val.Description.Pair != "" {
					if strings.Contains(val.Description.Order, "sell") {
						d.Side = order.Sell
					} else {
						if oSide, err := order.StringToOrderSide(val.Description.Type); err != nil {
							k.Websocket.DataHandler <- order.ClassificationError{
								Exchange: k.Name,
								OrderID:  key,
								Err:      err,
							}
						} else {
							d.Side = oSide
						}
					}

					if oType, err := order.StringToOrderType(val.Description.OrderType); err != nil {
						k.Websocket.DataHandler <- order.ClassificationError{
							Exchange: k.Name,
							OrderID:  key,
							Err:      err,
						}
					} else {
						d.Type = oType
					}

					if p, err := currency.NewPairFromString(val.Description.Pair); err != nil {
						k.Websocket.DataHandler <- order.ClassificationError{
							Exchange: k.Name,
							OrderID:  key,
							Err:      err,
						}
					} else {
						d.Pair = p
						if d.AssetType, err = k.GetPairAssetType(p); err != nil {
							k.Websocket.DataHandler <- order.ClassificationError{
								Exchange: k.Name,
								OrderID:  key,
								Err:      err,
							}
						}
					}
				}

				if val.Description.Price > 0 {
					d.Leverage = val.Description.Leverage
					d.Price = val.Description.Price
				}

				if val.Volume > 0 {
					// Note: We don't seem to ever get both there values
					d.RemainingAmount = val.Volume - val.ExecutedVolume
				}
				k.Websocket.DataHandler <- d
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
	var err error
	if response.Pair != "" {
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

	maxDepth := 0
	if splits := strings.Split(response.ChannelName, "-"); len(splits) > 1 {
		maxDepth, err = strconv.Atoi(splits[1])
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s exchange error: %s", k.Name, err)
		}
	}
	m.Lock()
	defer m.Unlock()
	subscriptionChannelPair = append(subscriptionChannelPair, WebsocketChannelData{
		Subscription: response.Subscription.Name,
		Pair:         fPair,
		ChannelID:    response.ChannelID,
		MaxDepth:     maxDepth,
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
	if len(data) < 5 {
		k.Websocket.DataHandler <- fmt.Errorf("%s unexpected wsProcessSpread data length", k.Name)
		return
	}
	bestBid, ok := data[0].(string)
	if !ok {
		k.Websocket.DataHandler <- fmt.Errorf("%s wsProcessSpread: unable to type assert bestBid", k.Name)
		return
	}
	bestAsk, ok := data[1].(string)
	if !ok {
		k.Websocket.DataHandler <- fmt.Errorf("%s wsProcessSpread: unable to type assert bestAsk", k.Name)
		return
	}
	timeData, err := strconv.ParseFloat(data[2].(string), 64)
	if err != nil {
		k.Websocket.DataHandler <- fmt.Errorf("%s wsProcessSpread: unable to parse timeData. Error: %s",
			k.Name,
			err)
		return
	}
	bidVolume, ok := data[3].(string)
	if !ok {
		k.Websocket.DataHandler <- fmt.Errorf("%s wsProcessSpread: unable to type assert bidVolume", k.Name)
		return
	}
	askVolume, ok := data[4].(string)
	if !ok {
		k.Websocket.DataHandler <- fmt.Errorf("%s wsProcessSpread: unable to type assert askVolume", k.Name)
		return
	}

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
	trades := make([]trade.Data, len(data))
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
		s, ok := t[3].(string)
		if !ok {
			return common.GetTypeAssertError("string", t[3], "side")
		}
		if s == "s" {
			tSide = order.Sell
		}

		trades[i] = trade.Data{
			AssetType:    asset.Spot,
			CurrencyPair: channelData.Pair,
			Exchange:     k.Name,
			Price:        price,
			Amount:       amount,
			Timestamp:    convert.TimeFromUnixTimestampDecimal(timeData),
			Side:         tSide,
		}
	}
	return trade.AddTradesToBuffer(k.Name, trades...)
}

// wsProcessOrderBook determines if the orderbook data is partial or update
// Then sends to appropriate fun
func (k *Kraken) wsProcessOrderBook(channelData *WebsocketChannelData, data map[string]interface{}) error {
	// NOTE: Updates are a priority so check if it's an update first as we don't
	// need multiple map lookups to check for snapshot.
	askData, asksExist := data["a"].([]interface{})
	bidData, bidsExist := data["b"].([]interface{})
	if asksExist || bidsExist {
		checksum, ok := data["c"].(string)
		if !ok {
			return errors.New("could not process orderbook update checksum not found")
		}

		k.wsRequestMtx.Lock()
		defer k.wsRequestMtx.Unlock()
		err := k.wsProcessOrderBookUpdate(channelData, askData, bidData, checksum)
		if err != nil {
			outbound := channelData.Pair // Format required "XBT/USD"
			outbound.Delimiter = "/"
			go func(resub *subscription.Subscription) {
				// This was locking the main websocket reader routine and a
				// backlog occurred. So put this into it's own go routine.
				errResub := k.Websocket.ResubscribeToChannel(resub)
				if errResub != nil {
					log.Errorf(log.WebsocketMgr,
						"resubscription failure for %v: %v",
						resub,
						errResub)
				}
			}(&subscription.Subscription{
				Channel: krakenWsOrderbook,
				Pair:    outbound,
				Asset:   asset.Spot,
			})
			return err
		}
		return nil
	}

	askSnapshot, askSnapshotExists := data["as"].([]interface{})
	bidSnapshot, bidSnapshotExists := data["bs"].([]interface{})
	if !askSnapshotExists && !bidSnapshotExists {
		return fmt.Errorf("%w for %v %v", errNoWebsocketOrderbookData, channelData.Pair, asset.Spot)
	}

	return k.wsProcessOrderBookPartial(channelData, askSnapshot, bidSnapshot)
}

// wsProcessOrderBookPartial creates a new orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookPartial(channelData *WebsocketChannelData, askData, bidData []interface{}) error {
	base := orderbook.Base{
		Pair:                   channelData.Pair,
		Asset:                  asset.Spot,
		VerifyOrderbook:        k.CanVerifyOrderbook,
		Bids:                   make(orderbook.Items, len(bidData)),
		Asks:                   make(orderbook.Items, len(askData)),
		MaxDepth:               channelData.MaxDepth,
		ChecksumStringRequired: true,
	}
	// Kraken ob data is timestamped per price, GCT orderbook data is
	// timestamped per entry using the highest last update time, we can attempt
	// to respect both within a reasonable degree
	var highestLastUpdate time.Time
	for i := range askData {
		asks, ok := askData[i].([]interface{})
		if !ok {
			return common.GetTypeAssertError("[]interface{}", askData[i], "asks")
		}
		if len(asks) < 3 {
			return errors.New("unexpected asks length")
		}
		priceStr, ok := asks[0].(string)
		if !ok {
			return common.GetTypeAssertError("string", asks[0], "price")
		}
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return err
		}
		amountStr, ok := asks[1].(string)
		if !ok {
			return common.GetTypeAssertError("string", asks[1], "amount")
		}
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return err
		}
		tdStr, ok := asks[2].(string)
		if !ok {
			return common.GetTypeAssertError("string", asks[2], "time")
		}
		timeData, err := strconv.ParseFloat(tdStr, 64)
		if err != nil {
			return err
		}
		base.Asks[i] = orderbook.Item{
			Amount:    amount,
			StrAmount: amountStr,
			Price:     price,
			StrPrice:  priceStr,
		}
		askUpdatedTime := convert.TimeFromUnixTimestampDecimal(timeData)
		if highestLastUpdate.Before(askUpdatedTime) {
			highestLastUpdate = askUpdatedTime
		}
	}

	for i := range bidData {
		bids, ok := bidData[i].([]interface{})
		if !ok {
			return common.GetTypeAssertError("[]interface{}", bidData[i], "bids")
		}
		if len(bids) < 3 {
			return errors.New("unexpected bids length")
		}
		priceStr, ok := bids[0].(string)
		if !ok {
			return common.GetTypeAssertError("string", bids[0], "price")
		}
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return err
		}
		amountStr, ok := bids[1].(string)
		if !ok {
			return common.GetTypeAssertError("string", bids[1], "amount")
		}
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return err
		}
		tdStr, ok := bids[2].(string)
		if !ok {
			return common.GetTypeAssertError("string", bids[2], "time")
		}
		timeData, err := strconv.ParseFloat(tdStr, 64)
		if err != nil {
			return err
		}

		base.Bids[i] = orderbook.Item{
			Amount:    amount,
			StrAmount: amountStr,
			Price:     price,
			StrPrice:  priceStr,
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
	update := orderbook.Update{
		Asset: asset.Spot,
		Pair:  channelData.Pair,
		Bids:  make([]orderbook.Item, len(bidData)),
		Asks:  make([]orderbook.Item, len(askData)),
	}

	// Calculating checksum requires incoming decimal place checks for both
	// price and amount as there is no set standard between currency pairs. This
	// is calculated per update as opposed to snapshot because changes to
	// decimal amounts could occur at any time.
	var highestLastUpdate time.Time
	// Ask data is not always sent
	for i := range askData {
		asks, ok := askData[i].([]interface{})
		if !ok {
			return errors.New("asks type assertion failure")
		}

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

		timeStr, ok := asks[2].(string)
		if !ok {
			return errors.New("time type assertion failure")
		}

		timeData, err := strconv.ParseFloat(timeStr, 64)
		if err != nil {
			return err
		}

		update.Asks[i] = orderbook.Item{
			Amount:    amount,
			StrAmount: amountStr,
			Price:     price,
			StrPrice:  priceStr,
		}

		askUpdatedTime := convert.TimeFromUnixTimestampDecimal(timeData)
		if highestLastUpdate.Before(askUpdatedTime) {
			highestLastUpdate = askUpdatedTime
		}
	}

	// Bid data is not always sent
	for i := range bidData {
		bids, ok := bidData[i].([]interface{})
		if !ok {
			return common.GetTypeAssertError("[]interface{}", bidData[i], "bids")
		}

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

		timeStr, ok := bids[2].(string)
		if !ok {
			return errors.New("time type assertion failure")
		}

		timeData, err := strconv.ParseFloat(timeStr, 64)
		if err != nil {
			return err
		}

		update.Bids[i] = orderbook.Item{
			Amount:    amount,
			StrAmount: amountStr,
			Price:     price,
			StrPrice:  priceStr,
		}

		bidUpdatedTime := convert.TimeFromUnixTimestampDecimal(timeData)
		if highestLastUpdate.Before(bidUpdatedTime) {
			highestLastUpdate = bidUpdatedTime
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

	return validateCRC32(book, uint32(token))
}

func validateCRC32(b *orderbook.Base, token uint32) error {
	var checkStr strings.Builder
	for i := 0; i < 10 && i < len(b.Asks); i++ {
		_, err := checkStr.WriteString(trim(b.Asks[i].StrPrice + trim(b.Asks[i].StrAmount)))
		if err != nil {
			return err
		}
	}

	for i := 0; i < 10 && i < len(b.Bids); i++ {
		_, err := checkStr.WriteString(trim(b.Bids[i].StrPrice) + trim(b.Bids[i].StrAmount))
		if err != nil {
			return err
		}
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
func (k *Kraken) GenerateDefaultSubscriptions() ([]subscription.Subscription, error) {
	enabledPairs, err := k.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions []subscription.Subscription
	for i := range defaultSubscribedChannels {
		for j := range enabledPairs {
			enabledPairs[j].Delimiter = "/"
			subscriptions = append(subscriptions, subscription.Subscription{
				Channel: defaultSubscribedChannels[i],
				Pair:    enabledPairs[j],
				Asset:   asset.Spot,
			})
		}
	}
	if k.Websocket.CanUseAuthenticatedEndpoints() {
		for i := range authenticatedChannels {
			subscriptions = append(subscriptions, subscription.Subscription{
				Channel: authenticatedChannels[i],
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (k *Kraken) Subscribe(channelsToSubscribe []subscription.Subscription) error {
	var subscriptions = make(map[string]*[]WebsocketSubscriptionEventRequest)
channels:
	for i := range channelsToSubscribe {
		s, ok := subscriptions[channelsToSubscribe[i].Channel]
		if !ok {
			s = &[]WebsocketSubscriptionEventRequest{}
			subscriptions[channelsToSubscribe[i].Channel] = s
		}

		for j := range *s {
			(*s)[j].Pairs = append((*s)[j].Pairs, channelsToSubscribe[i].Pair.String())
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
			outbound.Subscription.Depth = krakenWsOrderbookDepth
		}
		if !channelsToSubscribe[i].Pair.IsEmpty() {
			outbound.Pairs = []string{channelsToSubscribe[i].Pair.String()}
		}
		if common.StringDataContains(authenticatedChannels, channelsToSubscribe[i].Channel) {
			outbound.Subscription.Token = authToken
		}

		outbound.Channels = append(outbound.Channels, channelsToSubscribe[i])
		*s = append(*s, outbound)
	}

	var errs error
	for _, subs := range subscriptions {
		for i := range *subs {
			if common.StringDataContains(authenticatedChannels, (*subs)[i].Subscription.Name) {
				_, err := k.Websocket.AuthConn.SendMessageReturnResponse((*subs)[i].RequestID, (*subs)[i])
				if err != nil {
					errs = common.AppendError(errs, err)
					continue
				}
				k.Websocket.AddSuccessfulSubscriptions((*subs)[i].Channels...)
				continue
			}
			_, err := k.Websocket.Conn.SendMessageReturnResponse((*subs)[i].RequestID, (*subs)[i])
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			k.Websocket.AddSuccessfulSubscriptions((*subs)[i].Channels...)
		}
	}
	return errs
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (k *Kraken) Unsubscribe(channelsToUnsubscribe []subscription.Subscription) error {
	var unsubs []WebsocketSubscriptionEventRequest
channels:
	for x := range channelsToUnsubscribe {
		for y := range unsubs {
			if unsubs[y].Subscription.Name == channelsToUnsubscribe[x].Channel {
				unsubs[y].Pairs = append(unsubs[y].Pairs,
					channelsToUnsubscribe[x].Pair.String())
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
			Pairs: []string{channelsToUnsubscribe[x].Pair.String()},
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

	var errs error
	for i := range unsubs {
		if common.StringDataContains(authenticatedChannels, unsubs[i].Subscription.Name) {
			_, err := k.Websocket.AuthConn.SendMessageReturnResponse(unsubs[i].RequestID, unsubs[i])
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			k.Websocket.RemoveSubscriptions(unsubs[i].Channels...)
			continue
		}

		_, err := k.Websocket.Conn.SendMessageReturnResponse(unsubs[i].RequestID, unsubs[i])
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		k.Websocket.RemoveSubscriptions(unsubs[i].Channels...)
	}
	return errs
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
		return "", errors.New(k.Name + " - " + resp.ErrorMessage)
	}
	return resp.TransactionID, nil
}

// wsCancelOrders cancels open orders concurrently
// It does not use the multiple txId facility of the cancelOrder API because the errors are not specific
func (k *Kraken) wsCancelOrders(orderIDs []string) error {
	errs := common.CollectErrors(len(orderIDs))
	for _, id := range orderIDs {
		go func() {
			defer errs.Wg.Done()
			errs.C <- k.wsCancelOrder(id)
		}()
	}

	return errs.Collect()
}

// wsCancelOrder cancels an open order
func (k *Kraken) wsCancelOrder(orderID string) error {
	id := k.Websocket.AuthConn.GenerateMessageID(false)
	request := WsCancelOrderRequest{
		Event:          krakenWsCancelOrder,
		Token:          authToken,
		TransactionIDs: []string{orderID},
		RequestID:      id,
	}

	resp, err := k.Websocket.AuthConn.SendMessageReturnResponse(id, request)
	if err != nil {
		return fmt.Errorf("%w %s: %w", errCancellingOrder, orderID, err)
	}

	status, err := jsonparser.GetUnsafeString(resp, "status")
	if err != nil {
		return fmt.Errorf("%w 'status': %w from message: %s", errParsingWSField, err, resp)
	} else if status == "ok" {
		return nil
	}

	err = errUnknownError
	if msg, pErr := jsonparser.GetUnsafeString(resp, "errorMessage"); pErr == nil && msg != "" {
		err = errors.New(msg)
	}

	return fmt.Errorf("%w %s: %w", errCancellingOrder, orderID, err)
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
		return &WsCancelOrderResponse{}, errors.New(k.Name + " - " + resp.ErrorMessage)
	}
	return &resp, nil
}
