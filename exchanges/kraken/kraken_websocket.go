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
	krakenWsHeartbeat               = "heartbeat"
	krakenWsSystemStatus            = "systemStatus"
	krakenWsSubscribe               = "subscribe"
	krakenWsSubscriptionStatus      = "subscriptionStatus"
	krakenWsUnsubscribe             = "unsubscribe"
	krakenWsTicker                  = "ticker"
	krakenWsOHLC                    = "ohlc"
	krakenWsTrade                   = "trade"
	krakenWsSpread                  = "spread"
	krakenWsOrderbook               = "book"
	krakenWsOwnTrades               = "ownTrades"
	krakenWsOpenOrders              = "openOrders"
	krakenWsAddOrder                = "addOrder"
	krakenWsCancelOrder             = "cancelOrder"
	krakenWsCancelAll               = "cancelAll"
	krakenWsAddOrderStatus          = "addOrderStatus"
	krakenWsCancelOrderStatus       = "cancelOrderStatus"
	krakenWsCancelAllOrderStatus    = "cancelAllStatus"
	krakenWsRateLimit               = 50
	krakenWsPingDelay               = time.Second * 27
	krakenWsOrderbookDefaultDepth   = 1000
	krakenWsCandlesDefaultTimeframe = 1
)

var (
	authToken string
)

// Channels require a topic and a currency
// Format [[ticker,but-t4u],[orderbook,nce-btt]]
var defaultSubscribedChannels = []string{
	krakenWsTicker,
	krakenWsTrade,
	krakenWsOrderbook,
	krakenWsOHLC,
	krakenWsSpread,
}
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
				k.wsPingHandler(k.Websocket.AuthConn)
			}
		}
	}

	k.wsPingHandler(k.Websocket.Conn)

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
		if err := json.Unmarshal(respRaw, &dataResponse); err != nil {
			return err
		}
		if len(dataResponse) < 3 {
			return fmt.Errorf("websocket data array too short: %s", respRaw)
		}

		// For all types of channel second to last field is the channel Name
		channelName, ok := dataResponse[len(dataResponse)-2].(string)
		if !ok {
			return common.GetTypeAssertError("string", dataResponse[len(dataResponse)-2], "channelName")
		}

		// wsPair is just used for keying the Subs
		wsPair := currency.EMPTYPAIR
		if maybePair, ok2 := dataResponse[len(dataResponse)-1].(string); ok2 {
			var err error
			if wsPair, err = currency.NewPairFromString(maybePair); err != nil {
				return err
			}
		}

		c := k.Websocket.GetSubscription(stream.DefaultChannelKey{Channel: channelName, Currency: wsPair, Asset: asset.Spot})
		if c == nil {
			return fmt.Errorf("%w: %s %s %s", stream.ErrSubscriptionNotFound, asset.Spot, channelName, wsPair)
		}

		return k.wsReadDataResponse(c, dataResponse)
	}

	var eventResponse map[string]interface{}
	err := json.Unmarshal(respRaw, &eventResponse)
	if err != nil {
		return fmt.Errorf("%s - err %s could not parse websocket data: %s",
			k.Name,
			err,
			respRaw)
	}

	event, ok := eventResponse["event"]
	if !ok {
		return nil
	}

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
		if sub.RequestID == 0 {
			return fmt.Errorf("%v %w: %v", k.Name, errNoRequestID, respRaw)
		}
		k.Websocket.Match.IncomingWithData(sub.RequestID, respRaw)

		if sub.Status != "subscribed" && sub.Status != "unsubscribed" {
			return fmt.Errorf("%v %v %v",
				k.Name,
				sub.RequestID,
				sub.ErrorMessage)
		}
	default:
		k.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: k.Name + stream.UnhandledMessage + string(respRaw),
		}
	}

	return nil
}

// wsPingHandler starts a websocket ping handler every 27s
func (k *Kraken) wsPingHandler(conn stream.Connection) {
	conn.SetupPingHandler(stream.PingHandler{
		Message:     []byte(`{"event":"ping"}`),
		Delay:       krakenWsPingDelay,
		MessageType: websocket.TextMessage,
	})
}

// wsReadDataResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) wsReadDataResponse(c *stream.ChannelSubscription, response WebsocketDataResponse) error {
	switch c.Channel {
	case krakenWsTicker:
		t, ok := response[1].(map[string]interface{})
		if !ok {
			return errors.New("received invalid ticker data")
		}
		return k.wsProcessTickers(c, t)
	case krakenWsOHLC:
		o, ok := response[1].([]interface{})
		if !ok {
			return errors.New("received invalid OHLCV data")
		}
		return k.wsProcessCandles(c, o)
	case krakenWsOrderbook:
		if c.State == stream.ChannelUnsubscribing {
			return nil
		}
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
		return k.wsProcessOrderBook(c, ob)
	case krakenWsSpread:
		s, ok := response[1].([]interface{})
		if !ok {
			return errors.New("received invalid spread data")
		}
		k.wsProcessSpread(c, s)
	case krakenWsTrade:
		t, ok := response[1].([]interface{})
		if !ok {
			return errors.New("received invalid trade data")
		}
		return k.wsProcessTrades(c, t)
	case krakenWsOwnTrades:
		return k.wsProcessOwnTrades(response[0])
	case krakenWsOpenOrders:
		return k.wsProcessOpenOrders(response[0])
	default:
		return fmt.Errorf("%s received unidentified data for subscription %s: %+v", k.Name, c.Channel, response)
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

// wsProcessTickers converts ticker data and sends it to the datahandler
func (k *Kraken) wsProcessTickers(c *stream.ChannelSubscription, data map[string]interface{}) error {
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
		AssetType:    c.Asset,
		Pair:         c.Currency,
	}
	return nil
}

// wsProcessSpread converts spread/orderbook data and sends it to the datahandler
func (k *Kraken) wsProcessSpread(c *stream.ChannelSubscription, data []interface{}) {
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
			c.Currency,
			bestBid,
			bestAsk,
			convert.TimeFromUnixTimestampDecimal(timeData),
			bidVolume,
			askVolume)
	}
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (k *Kraken) wsProcessTrades(c *stream.ChannelSubscription, data []interface{}) error {
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
			AssetType:    c.Asset,
			CurrencyPair: c.Currency,
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
func (k *Kraken) wsProcessOrderBook(c *stream.ChannelSubscription, data map[string]interface{}) error {
	// NOTE: Updates are a priority so check if it's an update first as we don't
	// need multiple map lookups to check for snapshot.
	askData, asksExist := data["a"].([]interface{})
	bidData, bidsExist := data["b"].([]interface{})
	if asksExist || bidsExist {
		checksum, ok := data["c"].(string)
		if !ok {
			return fmt.Errorf("could not process orderbook update checksum not found")
		}

		k.wsRequestMtx.Lock()
		defer k.wsRequestMtx.Unlock()
		err := k.wsProcessOrderBookUpdate(c, askData, bidData, checksum)
		if err != nil {
			outbound := c.Currency // Format required "XBT/USD"
			outbound.Delimiter = "/"
			go func(resub *stream.ChannelSubscription) {
				// This was locking the main websocket reader routine and a
				// backlog occurred. So put this into it's own go routine.
				errResub := k.Websocket.ResubscribeToChannel(resub)
				if errResub != nil && errResub != stream.ErrChannelInStateAlready {
					log.Errorf(log.WebsocketMgr,
						"resubscription failure for %v: %v",
						resub,
						errResub)
				}
			}(c)
			return err
		}
		return nil
	}

	askSnapshot, askSnapshotExists := data["as"].([]interface{})
	bidSnapshot, bidSnapshotExists := data["bs"].([]interface{})
	if !askSnapshotExists && !bidSnapshotExists {
		return fmt.Errorf("%w for %v %v", errNoWebsocketOrderbookData, c.Currency, c.Asset)
	}

	return k.wsProcessOrderBookPartial(c, askSnapshot, bidSnapshot)
}

// wsProcessOrderBookPartial creates a new orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookPartial(c *stream.ChannelSubscription, askData, bidData []interface{}) error {
	depth, err := depthFromChan(c)
	if err != nil {
		return err
	}
	base := orderbook.Base{
		Pair:                   c.Currency,
		Asset:                  c.Asset,
		VerifyOrderbook:        k.CanVerifyOrderbook,
		Bids:                   make(orderbook.Items, len(bidData)),
		Asks:                   make(orderbook.Items, len(askData)),
		MaxDepth:               depth,
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
func (k *Kraken) wsProcessOrderBookUpdate(c *stream.ChannelSubscription, askData, bidData []interface{}, checksum string) error {
	update := orderbook.Update{
		Asset: c.Asset,
		Pair:  c.Currency,
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

	book, err := k.Websocket.Orderbook.GetOrderbook(c.Currency, c.Asset)
	if err != nil {
		return fmt.Errorf("cannot calculate websocket checksum: book not found for %s %s %w", c.Currency, c.Asset, err)
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
func (k *Kraken) wsProcessCandles(c *stream.ChannelSubscription, data []interface{}) error {
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
		AssetType: c.Asset,
		Pair:      c.Currency,
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
			c := stream.ChannelSubscription{
				Channel:  defaultSubscribedChannels[i],
				Currency: enabledCurrencies[j],
				Asset:    asset.Spot,
				Params:   map[string]any{},
			}
			switch defaultSubscribedChannels[i] {
			case krakenWsOrderbook:
				c.Params[ChannelOrderbookDepthKey] = krakenWsOrderbookDefaultDepth
			case krakenWsOHLC:
				c.Params[ChannelCandlesTimeframeKey] = krakenWsCandlesDefaultTimeframe
			}

			subscriptions = append(subscriptions, c)
		}
	}
	if k.Websocket.CanUseAuthenticatedEndpoints() {
		for i := range authenticatedChannels {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: authenticatedChannels[i],
				Asset:   asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (k *Kraken) Subscribe(channels []stream.ChannelSubscription) error {
	return k.parallelChanOp(channels, k.subscribeToChan)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (k *Kraken) Unsubscribe(channels []stream.ChannelSubscription) error {
	return k.parallelChanOp(channels, k.unsubscribeFromChan)
}

func (k *Kraken) parallelChanOp(channels []stream.ChannelSubscription, m func(*stream.ChannelSubscription) error) error {
	wg := sync.WaitGroup{}
	wg.Add(len(channels))
	errC := make(chan error, len(channels))

	for i := range channels {
		go func(c *stream.ChannelSubscription) {
			defer wg.Done()
			if err := m(c); err != nil {
				errC <- err
			}
		}(&channels[i])
	}

	wg.Wait()
	close(errC)

	var errs error
	for err := range errC {
		errs = common.AppendError(errs, err)
	}

	return errs
}

// subscribeToChan sends a websocket message to receive data from the channel
func (k *Kraken) subscribeToChan(c *stream.ChannelSubscription) error {
	r, err := k.reqForSub(krakenWsSubscribe, c)
	if err != nil {
		return fmt.Errorf("%w Channel: %s Pair: %s Error: %w", stream.ErrSubscriptionFailure, c.Channel, c.Currency, err)
	}

	if !c.Asset.IsValid() {
		c.Asset = asset.Spot
	}

	err = ensureChannelKeyed(c, r)
	if err != nil {
		return err
	}

	c.State = stream.ChannelSubscribing
	err = k.Websocket.AddSubscription(c)
	if err != nil {
		return fmt.Errorf("%w Channel: %s Pair: %s Error: %w", stream.ErrSubscriptionFailure, c.Channel, c.Currency, err)
	}

	conn := k.Websocket.Conn
	if common.StringDataContains(authenticatedChannels, r.Subscription.Name) {
		r.Subscription.Token = authToken
		conn = k.Websocket.AuthConn
	}

	respRaw, err := conn.SendMessageReturnResponse(r.RequestID, r)
	if err != nil {
		k.Websocket.RemoveSubscriptions(*c)
		return fmt.Errorf("%w Channel: %s Pair: %s Error: %w", stream.ErrSubscriptionFailure, c.Channel, c.Currency, err)
	}

	if err = k.getErrResp(respRaw); err != nil {
		wErr := fmt.Errorf("%w Channel: %s Pair: %s; %w", stream.ErrSubscriptionFailure, c.Channel, c.Currency, err)
		k.Websocket.DataHandler <- wErr
		k.Websocket.RemoveSubscriptions(*c)
		return wErr
	}

	if err = k.Websocket.SetSubscriptionState(c, stream.ChannelSubscribed); err != nil {
		log.Errorf(log.ExchangeSys, "%s error setting channel to subscribed: %s", k.Name, err)
	}

	if k.Verbose {
		log.Debugf(log.ExchangeSys, "%s Subscribed to Channel: %s Pair: %s\n", k.Name, c.Channel, c.Currency)
	}

	return nil
}

// unsubscribeFromChan sends a websocket message to stop receiving data from a channel
func (k *Kraken) unsubscribeFromChan(c *stream.ChannelSubscription) error {
	r, err := k.reqForSub(krakenWsUnsubscribe, c)
	if err != nil {
		return fmt.Errorf("%w Channel: %s Pair: %s Error: %w", stream.ErrUnsubscribeFailure, c.Channel, c.Currency, err)
	}

	c.EnsureKeyed()

	if err = k.Websocket.SetSubscriptionState(c, stream.ChannelUnsubscribing); err != nil {
		// err is probably ErrChannelInStateAlready, but we want to bubble it up to prevent an attempt to Subscribe again
		// We can catch and ignore it in our call to resub
		return fmt.Errorf("%w Channel: %s Pair: %s Error: %w", stream.ErrUnsubscribeFailure, c.Channel, c.Currency, err)
	}

	conn := k.Websocket.Conn
	if common.StringDataContains(authenticatedChannels, c.Channel) {
		conn = k.Websocket.AuthConn
		r.Subscription.Token = authToken
	}

	respRaw, err := conn.SendMessageReturnResponse(r.RequestID, r)
	if err != nil {
		if e2 := k.Websocket.SetSubscriptionState(c, stream.ChannelSubscribed); e2 != nil {
			log.Errorf(log.ExchangeSys, "%s error setting channel to subscribed: %s", k.Name, e2)
		}
		return err
	}

	if err = k.getErrResp(respRaw); err != nil {
		wErr := fmt.Errorf("%w Channel: %s Pair: %s; %w", stream.ErrUnsubscribeFailure, c.Channel, c.Currency, err)
		k.Websocket.DataHandler <- wErr
		if e2 := k.Websocket.SetSubscriptionState(c, stream.ChannelSubscribed); e2 != nil {
			log.Errorf(log.ExchangeSys, "%s error setting channel to subscribed: %s", k.Name, e2)
		}
		return wErr
	}

	k.Websocket.RemoveSubscriptions(*c)

	return nil
}

func (k *Kraken) reqForSub(e string, c *stream.ChannelSubscription) (*WebsocketSubRequest, error) {
	r := &WebsocketSubRequest{
		Event:     e,
		RequestID: k.Websocket.Conn.GenerateMessageID(false),
		Subscription: WebsocketSubscriptionData{
			Name: c.Channel,
		},
	}

	if !c.Currency.IsEmpty() {
		r.Pairs = []string{c.Currency.String()}
	}

	var err error
	switch c.Channel {
	case krakenWsOrderbook:
		r.Subscription.Depth, err = depthFromChan(c)
	case krakenWsOHLC:
		r.Subscription.Interval, err = timeframeFromChan(c)
	}

	return r, err
}

// ensureChannelKeyed wraps the channel EnsureKeyed to add channel name suffixes for Depth and Interval
func ensureChannelKeyed(c *stream.ChannelSubscription, r *WebsocketSubRequest) error {
	key, ok := c.EnsureKeyed().(stream.DefaultChannelKey)
	if !ok {
		return common.GetTypeAssertError("stream.DefaultChannelKey", c.Key, "subscription.Key") // Should be impossible
	}

	if strings.Contains(key.Channel, "-") {
		return nil // Key already has a suffix
	}

	if r.Subscription.Depth > 0 {
		key.Channel += "-" + strconv.Itoa(r.Subscription.Depth) // All responses will have book-N as the channel name
	}

	if r.Subscription.Interval > 0 {
		key.Channel += "-" + strconv.Itoa(r.Subscription.Interval) // All responses will have ohlc-N as the channel name
	}

	c.Key = key

	return nil
}

func depthFromChan(c *stream.ChannelSubscription) (int, error) {
	depthAny, ok := c.Params[ChannelOrderbookDepthKey]
	if !ok {
		return 0, errMaxDepthMissing
	}
	depthInt, ok2 := depthAny.(int)
	if !ok2 {
		return 0, common.GetTypeAssertError("int", depthAny, "Subscription.Depth")
	}
	return depthInt, nil
}

func timeframeFromChan(c *stream.ChannelSubscription) (int, error) {
	timeframeAny, ok := c.Params[ChannelCandlesTimeframeKey]
	if !ok {
		return 0, errTimeframeMissing
	}
	timeframeInt, ok2 := timeframeAny.(int)
	if !ok2 {
		return 0, common.GetTypeAssertError("int", timeframeAny, "Subscription.Interval")
	}
	return timeframeInt, nil
}

// getErrResp takes a json response string and looks for an error event type
// If found it returns the errorMessage
// It might log parsing errors about the nature of the error
// If the error message is not defined it will return a wrapped errUnknownError
func (k *Kraken) getErrResp(resp []byte) error {
	event, err := jsonparser.GetUnsafeString(resp, "event")
	switch {
	case err != nil:
		return fmt.Errorf("error parsing WS event: %w from message: %s", err, resp)
	case event != "error":
		status, _ := jsonparser.GetUnsafeString(resp, "status") // Error is really irrellevant here
		if status != "error" {
			return nil
		}
	}

	var msg string
	if msg, err = jsonparser.GetString(resp, "errorMessage"); err != nil {
		log.Errorf(log.ExchangeSys, "%s error parsing WS errorMessage: %s from message: %s", k.Name, err, resp)
		return fmt.Errorf("error status did not contain errorMessage: %s", resp)
	}
	return errors.New(msg)
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
		return &WsCancelOrderResponse{}, errors.New(k.Name + " - " + resp.ErrorMessage)
	}
	return &resp, nil
}
