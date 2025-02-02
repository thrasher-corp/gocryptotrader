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
	"text/template"
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
	krakenWsUnsubscribe          = "unsubscribe"
	krakenWsSubscribed           = "subscribed"
	krakenWsUnsubscribed         = "unsubscribed"
	krakenWsSubscriptionStatus   = "subscriptionStatus"
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

var channelNames = map[string]string{
	subscription.TickerChannel:    krakenWsTicker,
	subscription.OrderbookChannel: krakenWsOrderbook,
	subscription.CandlesChannel:   krakenWsOHLC,
	subscription.AllTradesChannel: krakenWsTrade,
	subscription.MyTradesChannel:  krakenWsOwnTrades,
	subscription.MyOrdersChannel:  krakenWsOpenOrders,
}
var reverseChannelNames = map[string]string{}

func init() {
	for k, v := range channelNames {
		reverseChannelNames[v] = k
	}
}

var (
	authToken          string
	errParsingWSField  = errors.New("error parsing WS field")
	errCancellingOrder = errors.New("error cancelling order")
	errSubPairMissing  = errors.New("pair missing from subscription response")
	errInvalidChecksum = errors.New("invalid checksum")
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 1000},
	{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true},
}

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
				k.Websocket.DataHandler <- err
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
			return fmt.Errorf("data array too short: %s", respRaw)
		}

		// For all types of channel second to last field is the channel Name
		c, ok := msg[len(msg)-2].(string)
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
		return k.wsReadDataResponse(c, pair, msg)
	}

	event, err := jsonparser.GetString(respRaw, "event")
	if err != nil {
		return fmt.Errorf("%w parsing: %s", err, respRaw)
	}

	if event == krakenWsSubscriptionStatus { // Must happen before IncomingWithData to avoid race
		k.wsProcessSubStatus(respRaw)
	}

	reqID, err := jsonparser.GetInt(respRaw, "reqid")
	if err == nil && reqID != 0 && k.Websocket.Match.IncomingWithData(reqID, respRaw) {
		return nil
	}

	if event == "" {
		return nil
	}

	switch event {
	case stream.Pong, krakenWsHeartbeat:
		return nil
	case krakenWsCancelOrderStatus, krakenWsCancelAllOrderStatus, krakenWsAddOrderStatus, krakenWsSubscriptionStatus:
		// All of these should have found a listener already
		return fmt.Errorf("%w: %s %v", stream.ErrSignatureNotMatched, event, reqID)
	case krakenWsSystemStatus:
		return k.wsProcessSystemStatus(respRaw)
	default:
		k.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: fmt.Sprintf("%s: %s", stream.UnhandledMessage, respRaw),
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
func (k *Kraken) wsReadDataResponse(c string, pair currency.Pair, response []any) error {
	switch c {
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

	channelType := strings.TrimRight(c, "-0123456789")
	switch channelType {
	case krakenWsOHLC:
		return k.wsProcessCandle(c, response, pair)
	case krakenWsOrderbook:
		return k.wsProcessOrderBook(c, response, pair)
	default:
		return fmt.Errorf("received unidentified data for subscription %s: %+v", c, response)
	}
}

func (k *Kraken) wsProcessSystemStatus(respRaw []byte) error {
	var systemStatus wsSystemStatus
	err := json.Unmarshal(respRaw, &systemStatus)
	if err != nil {
		return fmt.Errorf("%s parsing system status: %s", err, respRaw)
	}
	if systemStatus.Status != "online" {
		k.Websocket.DataHandler <- fmt.Errorf("system status not online: %v", systemStatus.Status)
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
	return errors.New("invalid own trades data")
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
		return errors.New("unexpected wsProcessSpread data length")
	}
	bestBid, ok := data[0].(string)
	if !ok {
		return errors.New("wsProcessSpread: unable to type assert bestBid")
	}
	bestAsk, ok := data[1].(string)
	if !ok {
		return errors.New("wsProcessSpread: unable to type assert bestAsk")
	}
	timeData, err := strconv.ParseFloat(data[2].(string), 64)
	if err != nil {
		return fmt.Errorf("wsProcessSpread: unable to parse timeData: %w", err)
	}
	bidVolume, ok := data[3].(string)
	if !ok {
		return errors.New("wsProcessSpread: unable to type assert bidVolume")
	}
	askVolume, ok := data[4].(string)
	if !ok {
		return errors.New("wsProcessSpread: unable to type assert askVolume")
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
func (k *Kraken) wsProcessOrderBook(c string, response []any, pair currency.Pair) error {
	key := &subscription.Subscription{
		Channel: c,
		Asset:   asset.Spot,
		Pairs:   currency.Pairs{pair},
	}
	if err := fqChannelNameSub(key); err != nil {
		return err
	}
	s := k.Websocket.GetSubscription(key)
	if s == nil {
		return fmt.Errorf("%w: %s %s %s", subscription.ErrNotFound, asset.Spot, c, pair)
	}
	if s.State() == subscription.UnsubscribingState {
		// We only care if it's currently unsubscribing
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

		err := k.wsProcessOrderBookUpdate(pair, askData, bidData, checksum)
		if errors.Is(err, errInvalidChecksum) {
			log.Debugf(log.Global, "%s Resubscribing to invalid %s orderbook", k.Name, pair)
			go func() {
				if e2 := k.Websocket.ResubscribeToChannel(k.Websocket.Conn, s); e2 != nil && !errors.Is(e2, subscription.ErrInStateAlready) {
					log.Errorf(log.ExchangeSys, "%s resubscription failure for %v: %v", k.Name, pair, e2)
				}
			}()
		}
		return err
	}

	askSnapshot, askSnapshotExists := ob["as"].([]interface{})
	bidSnapshot, bidSnapshotExists := ob["bs"].([]interface{})
	if !askSnapshotExists && !bidSnapshotExists {
		return fmt.Errorf("%w for %v %v", errNoWebsocketOrderbookData, pair, asset.Spot)
	}

	return k.wsProcessOrderBookPartial(pair, askSnapshot, bidSnapshot, key.Levels)
}

// wsProcessOrderBookPartial creates a new orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookPartial(pair currency.Pair, askData, bidData []any, levels int) error {
	base := orderbook.Base{
		Pair:                   pair,
		Asset:                  asset.Spot,
		VerifyOrderbook:        k.CanVerifyOrderbook,
		Bids:                   make(orderbook.Tranches, len(bidData)),
		Asks:                   make(orderbook.Tranches, len(askData)),
		MaxDepth:               levels,
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
		return fmt.Errorf("%s %s %w %d, expected %d", b.Pair, b.Asset, errInvalidChecksum, check, token)
	}
	return nil
}

// trim removes '.' and prefixed '0' from subsequent string
func trim(s string) string {
	s = strings.Replace(s, ".", "", 1)
	s = strings.TrimLeft(s, "0")
	return s
}

// wsProcessCandle converts candle data and sends it to the data handler
func (k *Kraken) wsProcessCandle(c string, resp []any, pair currency.Pair) error {
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
	parts := strings.Split(c, "-")
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

// GetSubscriptionTemplate returns a subscription channel template
func (k *Kraken) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{"channelName": channelName}).Parse(subTplText)
}

func (k *Kraken) generateSubscriptions() (subscription.List, error) {
	return k.Features.Subscriptions.ExpandTemplates(k)
}

// Subscribe adds a channel subscription to the websocket
func (k *Kraken) Subscribe(in subscription.List) error {
	in, errs := in.ExpandTemplates(k)

	// Collect valid new subs and add to websocket in Subscribing state
	subs := subscription.List{}
	for _, s := range in {
		if s.State() != subscription.ResubscribingState {
			if err := k.Websocket.AddSubscriptions(k.Websocket.Conn, s); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w; Channel: %s Pairs: %s", err, s.Channel, s.Pairs.Join()))
				continue
			}
		}
		subs = append(subs, s)
	}

	// Merge subs by grouping pairs for request; We make a single request to subscribe to N+ pairs, but get N+ responses back
	groupedSubs := subs.GroupPairs()

	errs = common.AppendError(errs,
		k.ParallelChanOp(groupedSubs, func(s subscription.List) error { return k.manageSubs(krakenWsSubscribe, s) }, 1),
	)

	for _, s := range subs {
		if s.State() != subscription.SubscribedState {
			_ = s.SetState(subscription.InactiveState)
			if err := k.Websocket.RemoveSubscriptions(k.Websocket.Conn, s); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("error removing failed subscription: %w; Channel: %s Pairs: %s", err, s.Channel, s.Pairs.Join()))
			}
		}
	}

	return errs
}

// Unsubscribe removes a channel subscriptions from the websocket
func (k *Kraken) Unsubscribe(keys subscription.List) error {
	var errs error
	// Make sure we have the concrete subscriptions, since we will change the state
	subs := make(subscription.List, 0, len(keys))
	for _, key := range keys {
		if s := k.Websocket.GetSubscription(key); s == nil {
			errs = common.AppendError(errs, fmt.Errorf("%w; Channel: %s Pairs: %s", subscription.ErrNotFound, key.Channel, key.Pairs.Join()))
		} else {
			if s.State() != subscription.ResubscribingState {
				if err := s.SetState(subscription.UnsubscribingState); err != nil {
					errs = common.AppendError(errs, fmt.Errorf("%w; Channel: %s Pairs: %s", err, s.Channel, s.Pairs.Join()))
					continue
				}
			}
			subs = append(subs, s)
		}
	}

	subs = subs.GroupPairs()

	return common.AppendError(errs,
		k.ParallelChanOp(subs, func(s subscription.List) error { return k.manageSubs(krakenWsUnsubscribe, s) }, 1),
	)
}

// manageSubs handles both websocket channel subscribe and unsubscribe
func (k *Kraken) manageSubs(op string, subs subscription.List) error {
	if len(subs) != 1 {
		return subscription.ErrBatchingNotSupported
	}

	s := subs[0]

	if err := enforceStandardChannelNames(s); err != nil {
		return err
	}

	reqFmt := currency.PairFormat{Uppercase: true, Delimiter: "/"}
	r := &WebsocketSubRequest{
		Event:     op,
		RequestID: k.Websocket.Conn.GenerateMessageID(false),
		Subscription: WebsocketSubscriptionData{
			Name:  s.QualifiedChannel,
			Depth: s.Levels,
		},
		Pairs: s.Pairs.Format(reqFmt).Strings(),
	}

	if s.Interval != 0 {
		// TODO: Can Interval type be a kraken specific type with a MarshalText so we don't have to duplicate this
		r.Subscription.Interval = int(time.Duration(s.Interval).Minutes())
	}

	conn := k.Websocket.Conn
	if s.Authenticated {
		r.Subscription.Token = authToken
		conn = k.Websocket.AuthConn
	}

	resps, err := conn.SendMessageReturnResponses(context.TODO(), request.Unset, r.RequestID, r, len(s.Pairs))

	// Ignore an overall timeout, because we'll track individual subscriptions in handleSubResps
	err = common.ExcludeError(err, stream.ErrSignatureTimeout)

	if err != nil {
		return fmt.Errorf("%w; Channel: %s Pair: %s", err, s.Channel, s.Pairs)
	}

	return k.handleSubResps(s, resps, op)
}

// handleSubResps takes a collection of subscription responses from Kraken
// We submit a subscription for N+ pairs, and we get N+ individual responses
// Returns an error collection of unique errors and its pairs
func (k *Kraken) handleSubResps(s *subscription.Subscription, resps [][]byte, op string) error {
	reqFmt := currency.PairFormat{Uppercase: true, Delimiter: "/"}

	errMap := map[string]error{}
	pairErrs := map[currency.Pair]error{}
	for _, p := range s.Pairs {
		pairErrs[p.Format(reqFmt)] = errSubPairMissing
	}

	subPairs := currency.Pairs{}
	for _, resp := range resps {
		pName, err := jsonparser.GetUnsafeString(resp, "pair")
		if err != nil {
			return fmt.Errorf("%w parsing WS pair from message: %s", err, resp)
		}
		pair, err := currency.NewPairDelimiter(pName, "/")
		if err != nil {
			return fmt.Errorf("%w parsing WS pair; Channel: %s Pair: %s", err, s.Channel, pName)
		}
		if err := k.getSubRespErr(resp, op); err != nil {
			// Remove the pair name from the error so we can group errors
			errStr := strings.TrimSpace(strings.TrimSuffix(err.Error(), pName))
			if _, ok := errMap[errStr]; !ok {
				errMap[errStr] = errors.New(errStr)
			}
			pairErrs[pair] = errMap[errStr]
		} else {
			delete(pairErrs, pair)
			if k.Verbose && op == krakenWsSubscribe {
				subPairs = subPairs.Add(pair)
			}
		}
	}

	// 2) Reverse the collection and report a list of pairs with each unique error, and re-add the missing and error pairs for unsubscribe
	errPairs := map[error]currency.Pairs{}
	for pair, err := range pairErrs {
		errPairs[err] = errPairs[err].Add(pair)
	}

	var errs error
	for err, pairs := range errPairs {
		errs = common.AppendError(errs, fmt.Errorf("%w; Channel: %s Pairs: %s", err, s.Channel, pairs.Join()))
	}

	if k.Verbose && len(subPairs) > 0 {
		log.Debugf(log.ExchangeSys, "%s Subscribed to Channel: %s Pairs: %s", k.Name, s.Channel, subPairs.Join())
	}

	return errs
}

// getSubErrResp calls getRespErr and if there's no error from that ensures the status matches the sub operation
func (k *Kraken) getSubRespErr(resp []byte, op string) error {
	if err := k.getRespErr(resp); err != nil {
		return err
	}
	exp := op + "d" // subscribed or unsubscribed
	if status, err := jsonparser.GetUnsafeString(resp, "status"); err != nil {
		return fmt.Errorf("error parsing WS status: %w from message: %s", err, resp)
	} else if status != exp {
		return fmt.Errorf("wrong WS status: %s; expected: %s from message %s", exp, op, resp)
	}

	return nil
}

// getRespErr takes a json response string and looks for an error event type
// If found it returns the errorMessage
// It might log parsing errors about the nature of the error
// If the error message is not defined it will return a wrapped errUnknownError
func (k *Kraken) getRespErr(resp []byte) error {
	event, err := jsonparser.GetUnsafeString(resp, "event")
	switch {
	case err != nil:
		return fmt.Errorf("error parsing WS event: %w from message: %s", err, resp)
	case event != "error":
		status, _ := jsonparser.GetUnsafeString(resp, "status") // Error is really irrelevant here
		if status != "error" {
			return nil
		}
	}

	var msg string
	if msg, err = jsonparser.GetString(resp, "errorMessage"); err != nil {
		log.Errorf(log.ExchangeSys, "%s error parsing WS errorMessage: %s from message: %s", k.Name, err, resp)
		return fmt.Errorf("%w: error message did not contain errorMessage: %s", common.ErrUnknownError, resp)
	}
	return errors.New(msg)
}

// wsProcessSubStatus handles creating or removing Subscriptions as soon as we receive a message
// It's job is to ensure that subscription state is kept correct sequentially between WS messages
// If this responsibility was moved to Subscribe then we would have a race due to the channel connecting IncomingWithData
func (k *Kraken) wsProcessSubStatus(resp []byte) {
	pName, err := jsonparser.GetUnsafeString(resp, "pair")
	if err != nil {
		return
	}
	pair, err := currency.NewPairFromString(pName)
	if err != nil {
		return
	}
	c, err := jsonparser.GetUnsafeString(resp, "channelName")
	if err != nil {
		return
	}
	if err = k.getRespErr(resp); err != nil {
		return
	}
	status, err := jsonparser.GetUnsafeString(resp, "status")
	if err != nil {
		return
	}
	key := &subscription.Subscription{
		// We don't use asset because it's either Empty or Spot, but not both
		Channel: c,
		Pairs:   currency.Pairs{pair},
	}

	if err = fqChannelNameSub(key); err != nil {
		return
	}
	s := k.Websocket.GetSubscription(&subscription.IgnoringAssetKey{Subscription: key})
	if s == nil {
		log.Errorf(log.ExchangeSys, "%s %s Channel: %s Pairs: %s", k.Name, subscription.ErrNotFound, key.Channel, key.Pairs.Join())
		return
	}

	if status == krakenWsSubscribed {
		err = s.SetState(subscription.SubscribedState)
	} else if s.State() != subscription.ResubscribingState { // Do not remove a resubscribing sub which just unsubbed
		err = k.Websocket.RemoveSubscriptions(k.Websocket.Conn, s)
		if e2 := s.SetState(subscription.UnsubscribedState); e2 != nil {
			err = common.AppendError(err, e2)
		}
	}

	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %s Channel: %s Pairs: %s", k.Name, err, s.Channel, s.Pairs.Join())
	}
}

// channelName converts a global channel name to kraken bespoke names
func channelName(s *subscription.Subscription) string {
	if n, ok := channelNames[s.Channel]; ok {
		return n
	}
	return s.Channel
}

func enforceStandardChannelNames(s *subscription.Subscription) error {
	name := strings.Split(s.Channel, "-") // Protect against attempted usage of book-N as a channel name
	if n, ok := reverseChannelNames[name[0]]; ok && n != s.Channel {
		return fmt.Errorf("%w: %s => subscription.%s%sChannel", subscription.ErrUseConstChannelName, s.Channel, bytes.ToUpper([]byte{n[0]}), n[1:])
	}
	return nil
}

// fqChannelNameSub converts an fully qualified channel name into standard name and subscription params
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
		return "", errors.New("AddOrder error: " + resp.ErrorMessage)
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
		return &WsCancelOrderResponse{}, errors.New(resp.ErrorMessage)
	}
	return &resp, nil
}

/*
One sub per-pair. We don't use one sub with many pairs because:
  - Kraken will fan out in responses anyay
  - resubscribe is messy when our subs don't match their respsonses
  - FlushChannels and GetChannelDiff would incorrectly resub existing subs if we don't generate the same as we've stored
*/
const subTplText = `
{{- if $.S.Asset -}}
	{{ range $asset, $pairs := $.AssetPairs }}
		{{- range $p := $pairs  -}}
			{{- channelName $.S }}
			{{- $.PairSeparator }}
		{{- end -}}
		{{ $.AssetSeparator }}
	{{- end -}}
{{- else -}}
	{{- channelName $.S }}
{{- end }}
`
