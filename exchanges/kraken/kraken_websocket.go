package kraken

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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

	// Websocket Channels
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
	krakenWsPingDelay            = time.Second * 27
)

var standardChannelNames = map[string]string{
	subscription.TickerChannel:    krakenWsTicker,
	subscription.OrderbookChannel: krakenWsOrderbook,
	subscription.CandlesChannel:   krakenWsOHLC,
	subscription.AllTradesChannel: krakenWsTrade,
	subscription.MyTradesChannel:  krakenWsOwnTrades,
	subscription.MyOrdersChannel:  krakenWsOpenOrders,
}
var reverseChannelNames = map[string]string{}

func init() {
	for k, v := range standardChannelNames {
		reverseChannelNames[v] = k
	}
}

var (
	authToken          string
	errParsingWSField  = errors.New("error parsing WS field")
	errCancellingOrder = errors.New("error cancelling order")
)

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
				k.startWsPingHandler(k.Websocket.AuthConn)
			}
		}
	}

	k.startWsPingHandler(k.Websocket.Conn)

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
						log.Errorf(log.WebsocketMgr, "%s websocket handle data error: %v", k.Name, err)
					}
				}
			default:
			}
			return
		case resp := <-comms:
			err := k.wsHandleData(resp.Raw)
			if err != nil {
				k.Websocket.DataHandler <- fmt.Errorf("%s - unhandled websocket data: %v", k.Name, err)
			}
		}
	}
}

func (k *Kraken) wsHandleData(respRaw []byte) error {
	if strings.HasPrefix(string(respRaw), "[") {
		var msg []any
		if err := json.Unmarshal(respRaw, &msg); err != nil {
			return err
		}
		if len(msg) < 3 {
			return fmt.Errorf("websocket data array too short: %s", respRaw)
		}

		// For all types of channel second to last field is the channel Name
		channelName, ok := msg[len(msg)-2].(string)
		if !ok {
			return common.GetTypeAssertError("string", msg[len(msg)-2], "channelName")
		}

		pair := currency.EMPTYPAIR
		if maybePair, ok2 := msg[len(msg)-1].(string); ok2 {
			var err error
			if pair, err = currency.NewPairFromString(maybePair); err != nil {
				return err
			}
		}
		return k.wsReadDataResponse(channelName, pair, msg)
	}

	reqID, err := jsonparser.GetInt(respRaw, "reqid")
	if err == nil && reqID != 0 && k.Websocket.Match.IncomingWithData(reqID, respRaw) {
		return nil
	}

	event, err := jsonparser.GetString(respRaw, "event")
	if err != nil {
		return fmt.Errorf("%s - err %s could not parse websocket data: %s", k.Name, err, respRaw)
	}

	if event == "" {
		return nil
	}

	switch event {
	case stream.Pong, krakenWsHeartbeat:
		return nil
	case krakenWsCancelOrderStatus, krakenWsCancelAllOrderStatus, krakenWsAddOrderStatus, krakenWsSubscriptionStatus:
		// All of these should have found a listener already
		return fmt.Errorf("%w: %s %v", stream.ErrNoMessageListener, event, reqID)
	case krakenWsSystemStatus:
		return k.wsProcessSystemStatus(respRaw)
	default:
		k.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: fmt.Sprintf("%s %s: %s", k.Name, stream.UnhandledMessage, respRaw),
		}
	}

	return nil
}

// startWsPingHandler sets up a websocket ping handler to maintain a connection
func (k *Kraken) startWsPingHandler(conn stream.Connection) {
	conn.SetupPingHandler(request.Unset, stream.PingHandler{
		Message:     []byte(`{"event":"ping"}`),
		Delay:       krakenWsPingDelay,
		MessageType: websocket.TextMessage,
	})
}

// wsReadDataResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) wsReadDataResponse(channelName string, pair currency.Pair, response []any) error {
	switch channelName {
	case krakenWsTicker:
		return k.wsProcessTickers(response, pair)
	case krakenWsSpread:
		return k.wsProcessSpread(response, pair)
	case krakenWsTrade:
		return k.wsProcessTrades(response, pair)
	case krakenWsOwnTrades:
		return k.wsProcessOwnTrades(response[0])
	case krakenWsOpenOrders:
		return k.wsProcessOpenOrders(response[0])
	}

	channelType := strings.TrimRight(channelName, "-0123456789")
	switch channelType {
	case krakenWsOHLC:
		return k.wsProcessCandle(channelName, response, pair)
	case krakenWsOrderbook:
		return k.wsProcessOrderBook(channelName, response, pair)
	default:
		return fmt.Errorf("%s received unidentified data for subscription %s: %+v", k.Name, channelName, response)
	}
}

func (k *Kraken) wsProcessSystemStatus(respRaw []byte) error {
	var systemStatus wsSystemStatus
	err := json.Unmarshal(respRaw, &systemStatus)
	if err != nil {
		return fmt.Errorf("%s - err %s unable to parse system status response: %s", k.Name, err, respRaw)
	}
	if systemStatus.Status != "online" {
		k.Websocket.DataHandler <- fmt.Errorf("%v Websocket status '%v'", k.Name, systemStatus.Status)
	}
	if systemStatus.Version > krakenWSSupportedVersion {
		log.Warnf(log.ExchangeSys, "%v New version of Websocket API released. Was %v Now %v", k.Name, krakenWSSupportedVersion, systemStatus.Version)
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
func (k *Kraken) wsProcessTickers(response []any, pair currency.Pair) error {
	t, ok := response[1].(map[string]any)
	if !ok {
		return errors.New("received invalid ticker data")
	}
	data := map[string]float64{}
	for _, b := range []byte("abcvlho") { // p and t skipped
		key := string(b)
		a, ok := t[key].([]any)
		if !ok {
			return fmt.Errorf("received invalid ticker data: %w", common.GetTypeAssertError("[]any", t[key], "ticker."+key))
		}
		var s string
		if s, ok = a[0].(string); !ok {
			return fmt.Errorf("received invalid ticker data: %w", common.GetTypeAssertError("string", a[0], "ticker."+key+"[0]"))
		}

		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("received invalid ticker data: %w", err)
		}
		data[key] = f
	}

	k.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: k.Name,
		Ask:          data["a"],
		Bid:          data["b"],
		Close:        data["c"],
		Volume:       data["v"],
		Low:          data["l"],
		High:         data["h"],
		Open:         data["o"],
		AssetType:    asset.Spot,
		Pair:         pair,
	}
	return nil
}

// wsProcessSpread converts spread/orderbook data and sends it to the datahandler
func (k *Kraken) wsProcessSpread(response []any, pair currency.Pair) error {
	data, ok := response[1].([]any)
	if !ok {
		return errors.New("received invalid spread data")
	}
	if len(data) < 5 {
		return fmt.Errorf("%s unexpected wsProcessSpread data length", k.Name)
	}
	bestBid, ok := data[0].(string)
	if !ok {
		return fmt.Errorf("%s wsProcessSpread: unable to type assert bestBid", k.Name)
	}
	bestAsk, ok := data[1].(string)
	if !ok {
		return fmt.Errorf("%s wsProcessSpread: unable to type assert bestAsk", k.Name)
	}
	timeData, err := strconv.ParseFloat(data[2].(string), 64)
	if err != nil {
		return fmt.Errorf("%s wsProcessSpread: unable to parse timeData. Error: %s", k.Name, err)
	}
	bidVolume, ok := data[3].(string)
	if !ok {
		return fmt.Errorf("%s wsProcessSpread: unable to type assert bidVolume", k.Name)
	}
	askVolume, ok := data[4].(string)
	if !ok {
		return fmt.Errorf("%s wsProcessSpread: unable to type assert askVolume", k.Name)
	}

	if k.Verbose {
		log.Debugf(log.ExchangeSys,
			"%v Spread data for '%v' received. Best bid: '%v' Best ask: '%v' Time: '%v', Bid volume '%v', Ask volume '%v'",
			k.Name,
			pair,
			bestBid,
			bestAsk,
			convert.TimeFromUnixTimestampDecimal(timeData),
			bidVolume,
			askVolume)
	}
	return nil
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (k *Kraken) wsProcessTrades(response []any, pair currency.Pair) error {
	data, ok := response[1].([]any)
	if !ok {
		return errors.New("received invalid trade data")
	}
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
			CurrencyPair: pair,
			Exchange:     k.Name,
			Price:        price,
			Amount:       amount,
			Timestamp:    convert.TimeFromUnixTimestampDecimal(timeData),
			Side:         tSide,
		}
	}
	return trade.AddTradesToBuffer(k.Name, trades...)
}

// wsProcessOrderBook handles both partial and full orderbook updates
func (k *Kraken) wsProcessOrderBook(channelName string, response []any, pair currency.Pair) error {
	key := &subscription.Subscription{Channel: channelName, Asset: asset.Spot, Pairs: currency.Pairs{pair}}
	if err := fqChannelNameSub(key); err != nil {
		return err
	}
	c := k.Websocket.GetSubscription(key)
	if c == nil {
		return fmt.Errorf("%w: %s %s %s", subscription.ErrNotFound, asset.Spot, channelName, pair)
	}
	if c.State() == subscription.UnsubscribingState {
		return nil
	}

	ob, ok := response[1].(map[string]any)
	if !ok {
		return errors.New("received invalid orderbook data")
	}

	if len(response) == 5 {
		ob2, ok2 := response[2].(map[string]any)
		if !ok2 {
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
	// NOTE: Updates are a priority so check if it's an update first as we don't
	// need multiple map lookups to check for snapshot.
	askData, asksExist := ob["a"].([]interface{})
	bidData, bidsExist := ob["b"].([]interface{})
	if asksExist || bidsExist {
		checksum, ok := ob["c"].(string)
		if !ok {
			return errors.New("could not process orderbook update checksum not found")
		}

		k.wsRequestMtx.Lock()
		defer k.wsRequestMtx.Unlock()
		err := k.wsProcessOrderBookUpdate(pair, askData, bidData, checksum)
		if err != nil {
			outbound := pair
			outbound.Delimiter = "/"
			go func(resub *subscription.Subscription) {
				// This was locking the main websocket reader routine and a
				// backlog occurred. So put this into it's own go routine.
				errResub := k.Websocket.ResubscribeToChannel(resub)
				if errResub != nil && errResub != subscription.ErrInStateAlready {
					log.Errorf(log.WebsocketMgr, "resubscription failure for %v: %v", resub, errResub)
				}
			}(c)
			return err
		}
		return nil
	}

	askSnapshot, askSnapshotExists := ob["as"].([]interface{})
	bidSnapshot, bidSnapshotExists := ob["bs"].([]interface{})
	if !askSnapshotExists && !bidSnapshotExists {
		return fmt.Errorf("%w for %v %v", errNoWebsocketOrderbookData, pair, asset.Spot)
	}

	return k.wsProcessOrderBookPartial(c, pair, askSnapshot, bidSnapshot)
}

// wsProcessOrderBookPartial creates a new orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookPartial(s *subscription.Subscription, pair currency.Pair, askData, bidData []any) error {
	base := orderbook.Base{
		Pair:                   pair,
		Asset:                  asset.Spot,
		VerifyOrderbook:        k.CanVerifyOrderbook,
		Bids:                   make(orderbook.Tranches, len(bidData)),
		Asks:                   make(orderbook.Tranches, len(askData)),
		MaxDepth:               s.Levels,
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
		base.Asks[i] = orderbook.Tranche{
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

		base.Bids[i] = orderbook.Tranche{
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
func (k *Kraken) wsProcessOrderBookUpdate(pair currency.Pair, askData, bidData []any, checksum string) error {
	update := orderbook.Update{
		Asset: asset.Spot,
		Pair:  pair,
		Bids:  make([]orderbook.Tranche, len(bidData)),
		Asks:  make([]orderbook.Tranche, len(askData)),
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

		update.Asks[i] = orderbook.Tranche{
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

		update.Bids[i] = orderbook.Tranche{
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

	book, err := k.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	if err != nil {
		return fmt.Errorf("cannot calculate websocket checksum: book not found for %s %s %w", pair, asset.Spot, err)
	}

	token, err := strconv.ParseInt(checksum, 10, 64)
	if err != nil {
		return err
	}

	return validateCRC32(book, uint32(token))
}

func validateCRC32(b *orderbook.Base, token uint32) error {
	if b == nil {
		return common.ErrNilPointer
	}
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
func (k *Kraken) wsProcessCandle(channelName string, resp []any, pair currency.Pair) error {
	// 8 string quoted floats followed by 1 integer for trade count
	dataRaw, ok := resp[1].([]any)
	if !ok || len(dataRaw) != 9 {
		return errors.New("received invalid candle data")
	}
	data := make([]float64, 8)
	for i := range 8 {
		s, ok := dataRaw[i].(string)
		if !ok {
			return fmt.Errorf("received invalid candle data: %w", common.GetTypeAssertError("string", dataRaw[i], "candle-data"))
		}

		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("received invalid candle data: %w", err)
		}
		data[i] = f
	}

	// Faster than getting it through the subscription
	parts := strings.Split(channelName, "-")
	if len(parts) != 2 {
		return errBadChannelSuffix
	}
	interval := parts[1]

	k.Websocket.DataHandler <- stream.KlineData{
		AssetType:  asset.Spot,
		Pair:       pair,
		Timestamp:  time.Now(),
		Exchange:   k.Name,
		StartTime:  convert.TimeFromUnixTimestampDecimal(data[0]),
		CloseTime:  convert.TimeFromUnixTimestampDecimal(data[1]),
		OpenPrice:  data[2],
		HighPrice:  data[3],
		LowPrice:   data[4],
		ClosePrice: data[5],
		Volume:     data[7],
		Interval:   interval,
	}
	return nil
}

// generateSubscriptions sets up the configured subscriptions for the websocket
func (k *Kraken) generateSubscriptions() (subscription.List, error) {
	subscriptions := subscription.List{}
	pairs, err := k.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	authed := k.Websocket.CanUseAuthenticatedEndpoints()
	for _, baseSub := range k.Features.Subscriptions {
		if !authed && baseSub.Authenticated {
			continue
		}
		s := baseSub.Clone()
		s.Asset = asset.Spot
		s.Pairs = pairs
		subscriptions = append(subscriptions, s)
	}

	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (k *Kraken) Subscribe(subs subscription.List) error {
	return k.ParallelChanOp(subs, k.subscribeToChan, 1)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (k *Kraken) Unsubscribe(subs subscription.List) error {
	return k.ParallelChanOp(subs, k.unsubscribeFromChan, 1)
}

// subscribeToChan sends a websocket message to receive data from the channel
func (k *Kraken) subscribeToChan(subs subscription.List) error {
	if len(subs) != 1 {
		return subscription.ErrBatchingNotSupported
	}

	s := subs[0]

	if err := enforceStandardChannelNames(s); err != nil {
		return fmt.Errorf("%w %w", stream.ErrSubscriptionFailure, err)
	}

	r := &WebsocketSubRequest{
		Event:     krakenWsSubscribe,
		RequestID: k.Websocket.Conn.GenerateMessageID(false),
		Subscription: WebsocketSubscriptionData{
			Name:  apiChannelName(s),
			Depth: s.Levels,
		},
		Pairs: s.Pairs.Format(currency.PairFormat{Uppercase: true, Delimiter: "/"}).Strings(),
	}

	if s.Interval != 0 {
		// TODO: Can Interval type be a kraken specific type with a MarshalText so we don't have to duplicate this
		r.Subscription.Interval = int(time.Duration(s.Interval).Minutes())
	}

	if !s.Asset.IsValid() {
		s.Asset = asset.Spot
	}

	if err := s.SetState(subscription.SubscribingState); err != nil {
		log.Errorf(log.ExchangeSys, "%s error setting channel to subscribed: %s", k.Name, err)
	}

	if err := k.Websocket.AddSubscriptions(s); err != nil {
		return fmt.Errorf("%w Channel: %s Pair: %s Error: %w", stream.ErrSubscriptionFailure, s.Channel, s.Pairs, err)
	}

	conn := k.Websocket.Conn
	if s.Authenticated {
		r.Subscription.Token = authToken
		conn = k.Websocket.AuthConn
	}

	respRaw, err := conn.SendMessageReturnResponse(context.TODO(), request.Unset, r.RequestID, r)
	if err == nil {
		err = k.getSubErrResp(respRaw, krakenWsSubscribe)
	}

	if err != nil {
		err = fmt.Errorf("%w Channel: %s Pair: %s; %w", stream.ErrSubscriptionFailure, s.Channel, s.Pairs, err)
		k.Websocket.DataHandler <- err
		// Currently all or nothing on pairs; Alternatively parse response and remove failing pairs and retry
		_ = k.Websocket.RemoveSubscriptions(s)
		return err
	}

	if err = s.SetState(subscription.SubscribedState); err != nil {
		log.Errorf(log.ExchangeSys, "%s error setting channel to subscribed: %s", k.Name, err)
	}

	if k.Verbose {
		log.Debugf(log.ExchangeSys, "%s Subscribed to Channel: %s Pair: %s\n", k.Name, s.Channel, s.Pairs)
	}

	return nil
}

// unsubscribeFromChan sends a websocket message to stop receiving data from a channel
func (k *Kraken) unsubscribeFromChan(subs subscription.List) error {
	if len(subs) != 1 {
		return subscription.ErrBatchingNotSupported
	}

	s := subs[0]

	if err := enforceStandardChannelNames(s); err != nil {
		return fmt.Errorf("%w %w", stream.ErrUnsubscribeFailure, err)
	}

	r := &WebsocketSubRequest{
		Event:     krakenWsUnsubscribe,
		RequestID: k.Websocket.Conn.GenerateMessageID(false),
		Subscription: WebsocketSubscriptionData{
			Name:  apiChannelName(s),
			Depth: s.Levels,
		},
		Pairs: s.Pairs.Format(currency.PairFormat{Uppercase: true, Delimiter: "/"}).Strings(),
	}

	if s.Interval != 0 {
		// TODO: Can Interval type be a kraken specific type with a MarshalText so we don't have to duplicate this
		r.Subscription.Interval = int(time.Duration(s.Interval).Minutes())
	}

	if err := s.SetState(subscription.UnsubscribingState); err != nil {
		// err is probably ErrChannelInStateAlready, but we want to bubble it up to prevent an attempt to Subscribe again
		// We can catch and ignore it in our call to resub
		return fmt.Errorf("%w Channel: %s Pair: %s Error: %w", stream.ErrUnsubscribeFailure, s.Channel, s.Pairs, err)
	}

	conn := k.Websocket.Conn
	if s.Authenticated {
		conn = k.Websocket.AuthConn
		r.Subscription.Token = authToken
	}

	respRaw, err := conn.SendMessageReturnResponse(context.TODO(), request.Unset, r.RequestID, r)
	if err != nil {
		if e2 := s.SetState(subscription.SubscribedState); e2 != nil {
			log.Errorf(log.ExchangeSys, "%s error setting channel to subscribed: %s", k.Name, e2)
		}
		return err
	}

	if err := k.getSubErrResp(respRaw, krakenWsUnsubscribe); err != nil {
		wErr := fmt.Errorf("%w Channel: %s Pair: %s; %w", stream.ErrUnsubscribeFailure, s.Channel, s.Pairs, err)
		k.Websocket.DataHandler <- wErr
		if e2 := s.SetState(subscription.SubscribedState); e2 != nil {
			log.Errorf(log.ExchangeSys, "%s error setting channel to subscribed: %s", k.Name, e2)
		}
		return wErr
	}

	return k.Websocket.RemoveSubscriptions(s)
}

func (k *Kraken) getSubErrResp(resp []byte, op string) error {
	if err := k.getErrResp(resp); err != nil {
		return err
	}
	exp := op + "d"
	if status, err := jsonparser.GetUnsafeString(resp, "status"); err != nil {
		return fmt.Errorf("error parsing WS status: %w from message: %s", err, resp)
	} else if status != exp {
		return fmt.Errorf("wrong WS status: %s; expected: %s from message %s", exp, op, resp)
	}
	return nil
}

// apiChannelName converts a global channel name to kraken bespoke names
func apiChannelName(s *subscription.Subscription) string {
	if n, ok := standardChannelNames[s.Channel]; ok {
		return n
	}
	return s.Channel
}

func enforceStandardChannelNames(s *subscription.Subscription) error {
	name := strings.Split(s.Channel, "-")
	if n, ok := reverseChannelNames[name[0]]; ok && n != s.Channel {
		return fmt.Errorf("%w: %s => subscription.%s%sChannel", subscription.ErrPrivateChannelName, s.Channel, bytes.ToUpper([]byte{n[0]}), n[1:])
	}
	return nil
}

// fqChannelNameToSub converts an fqChannelName into standard name and subscription params
// e.g. book-5 => subscription.OrderbookChannel with Levels: 5
func fqChannelNameSub(s *subscription.Subscription) error {
	parts := strings.Split(s.Channel, "-")
	name := parts[0]
	if stdName, ok := reverseChannelNames[name]; ok {
		name = stdName
	}

	if name == subscription.OrderbookChannel || name == subscription.CandlesChannel {
		if len(parts) != 2 {
			return errBadChannelSuffix
		}
		i, err := strconv.Atoi(parts[1])
		if err != nil {
			return errBadChannelSuffix
		}
		switch name {
		case subscription.OrderbookChannel:
			s.Levels = i
		case subscription.CandlesChannel:
			s.Interval = kline.Interval(time.Minute * time.Duration(i))
		}
	}

	s.Channel = name

	return nil
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
func (k *Kraken) wsAddOrder(req *WsAddOrderRequest) (string, error) {
	if req == nil {
		return "", common.ErrNilPointer
	}
	req.RequestID = k.Websocket.AuthConn.GenerateMessageID(false)
	req.Event = krakenWsAddOrder
	req.Token = authToken
	jsonResp, err := k.Websocket.AuthConn.SendMessageReturnResponse(context.TODO(), request.Unset, req.RequestID, req)
	if err != nil {
		return "", err
	}
	var resp WsAddOrderResponse
	err = json.Unmarshal(jsonResp, &resp)
	if err != nil {
		return "", err
	}
	if resp.Status == "error" {
		return "", errors.New(k.Name + "AddOrder error: " + resp.ErrorMessage)
	}
	k.Websocket.DataHandler <- &order.Detail{
		Exchange: k.Name,
		OrderID:  resp.TransactionID,
		Status:   order.New,
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
	req := WsCancelOrderRequest{
		Event:          krakenWsCancelOrder,
		Token:          authToken,
		TransactionIDs: []string{orderID},
		RequestID:      id,
	}

	resp, err := k.Websocket.AuthConn.SendMessageReturnResponse(context.TODO(), request.Unset, id, req)
	if err != nil {
		return fmt.Errorf("%w %s: %w", errCancellingOrder, orderID, err)
	}

	status, err := jsonparser.GetUnsafeString(resp, "status")
	if err != nil {
		return fmt.Errorf("%w 'status': %w from message: %s", errParsingWSField, err, resp)
	} else if status == "ok" {
		return nil
	}

	err = common.ErrUnknownError
	if msg, pErr := jsonparser.GetUnsafeString(resp, "errorMessage"); pErr == nil && msg != "" {
		err = errors.New(msg)
	}

	return fmt.Errorf("%w %s: %w", errCancellingOrder, orderID, err)
}

// wsCancelAllOrders cancels all opened orders
// Returns number (count param) of affected orders or 0 if no open orders found
func (k *Kraken) wsCancelAllOrders() (*WsCancelOrderResponse, error) {
	id := k.Websocket.AuthConn.GenerateMessageID(false)
	req := WsCancelOrderRequest{
		Event:     krakenWsCancelAll,
		Token:     authToken,
		RequestID: id,
	}

	jsonResp, err := k.Websocket.AuthConn.SendMessageReturnResponse(context.TODO(), request.Unset, id, req)
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
