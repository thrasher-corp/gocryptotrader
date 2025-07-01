package kraken

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
	krakenWsPong                 = "pong"
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
	ctx := context.TODO()
	if !k.Websocket.IsEnabled() || !k.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}

	var dialer gws.Dialer
	err := k.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}

	comms := make(chan websocket.Response)
	k.Websocket.Wg.Add(2)
	go k.wsReadData(ctx, comms)
	go k.wsFunnelConnectionData(k.Websocket.Conn, comms)

	if k.IsWebsocketAuthenticationSupported() {
		if authToken, err := k.GetWebsocketToken(ctx); err != nil {
			k.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys, "%s - authentication failed: %v\n", k.Name, err)
		} else {
			if err := k.Websocket.AuthConn.Dial(ctx, &dialer, http.Header{}); err != nil {
				k.Websocket.SetCanUseAuthenticatedEndpoints(false)
				log.Errorf(log.ExchangeSys, "%s - failed to connect to authenticated endpoint: %v\n", k.Name, err)
			} else {
				k.setWebsocketAuthToken(authToken)
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
func (k *Kraken) wsFunnelConnectionData(ws websocket.Connection, comms chan websocket.Response) {
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
func (k *Kraken) wsReadData(ctx context.Context, comms chan websocket.Response) {
	defer k.Websocket.Wg.Done()

	for {
		select {
		case <-k.Websocket.ShutdownC:
			select {
			case resp := <-comms:
				err := k.wsHandleData(ctx, resp.Raw)
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
			err := k.wsHandleData(ctx, resp.Raw)
			if err != nil {
				k.Websocket.DataHandler <- err
			}
		}
	}
}

func (k *Kraken) wsHandleData(_ context.Context, respRaw []byte) error {
	if strings.HasPrefix(string(respRaw), "[") {
		var msg []json.RawMessage
		if err := json.Unmarshal(respRaw, &msg); err != nil {
			return err
		}
		if len(msg) < 3 {
			return fmt.Errorf("data array too short: %s", respRaw)
		}

		// For all types of channel second to last field is the channel Name
		var chanName string
		if err := json.Unmarshal(msg[len(msg)-2], &chanName); err != nil {
			return fmt.Errorf("error unmarshalling channel name: %w", err)
		}

		pair := currency.EMPTYPAIR
		var maybePair string
		if err := json.Unmarshal(msg[len(msg)-1], &maybePair); err == nil {
			p, err := currency.NewPairFromString(maybePair)
			if err != nil {
				return err
			}
			pair = p
		}

		return k.wsReadDataResponse(chanName, pair, msg)
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
	case krakenWsPong, krakenWsHeartbeat:
		return nil
	case krakenWsCancelOrderStatus, krakenWsCancelAllOrderStatus, krakenWsAddOrderStatus, krakenWsSubscriptionStatus:
		// All of these should have found a listener already
		return fmt.Errorf("%w: %s %v", websocket.ErrSignatureNotMatched, event, reqID)
	case krakenWsSystemStatus:
		return k.wsProcessSystemStatus(respRaw)
	default:
		k.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
			Message: fmt.Sprintf("%s: %s", websocket.UnhandledMessage, respRaw),
		}
	}

	return nil
}

// startWsPingHandler sets up a websocket ping handler to maintain a connection
func (k *Kraken) startWsPingHandler(conn websocket.Connection) {
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Message:     []byte(`{"event":"ping"}`),
		Delay:       krakenWsPingDelay,
		MessageType: gws.TextMessage,
	})
}

// wsReadDataResponse classifies the WS response and sends to appropriate handler
func (k *Kraken) wsReadDataResponse(c string, pair currency.Pair, response []json.RawMessage) error {
	switch c {
	case krakenWsTicker:
		return k.wsProcessTickers(response[1], pair)
	case krakenWsSpread:
		return k.wsProcessSpread(response[1], pair)
	case krakenWsTrade:
		return k.wsProcessTrades(response[1], pair)
	case krakenWsOwnTrades:
		return k.wsProcessOwnTrades(response[0])
	case krakenWsOpenOrders:
		return k.wsProcessOpenOrders(response[0])
	}

	channelType := strings.TrimRight(c, "-0123456789")
	switch channelType {
	case krakenWsOHLC:
		return k.wsProcessCandle(c, response[1], pair)
	case krakenWsOrderbook:
		return k.wsProcessOrderBook(c, response, pair)
	default:
		return fmt.Errorf("received unidentified data for subscription %s: %+v", c, response)
	}
}

func (k *Kraken) wsProcessSystemStatus(respRaw []byte) error {
	var systemStatus wsSystemStatus
	if err := json.Unmarshal(respRaw, &systemStatus); err != nil {
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

func (k *Kraken) wsProcessOwnTrades(ownOrdersRaw json.RawMessage) error {
	var result []map[string]*WsOwnTrade
	if err := json.Unmarshal(ownOrdersRaw, &result); err != nil {
		return err
	}

	if len(result) == 0 {
		return nil
	}

	for key, val := range result[0] {
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
			Timestamp: val.Time.Time(),
		}
		k.Websocket.DataHandler <- &order.Detail{
			Exchange: k.Name,
			OrderID:  val.OrderTransactionID,
			Trades:   []order.TradeHistory{trade},
		}
	}

	return nil
}

// wsProcessOpenOrders processes open orders from the websocket response
func (k *Kraken) wsProcessOpenOrders(ownOrdersResp json.RawMessage) error {
	var result []map[string]*WsOpenOrder
	if err := json.Unmarshal(ownOrdersResp, &result); err != nil {
		return err
	}

	for r := range result {
		for key, val := range result[r] {
			d := &order.Detail{
				Exchange:             k.Name,
				OrderID:              key,
				AverageExecutedPrice: val.AveragePrice,
				Amount:               val.Volume,
				LimitPriceUpper:      val.LimitPrice,
				ExecutedAmount:       val.ExecutedVolume,
				Fee:                  val.Fee,
				Date:                 val.OpenTime.Time(),
				LastUpdated:          val.LastUpdated.Time(),
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
				// Note: Volume and ExecutedVolume are only populated when status is open
				d.RemainingAmount = val.Volume - val.ExecutedVolume
			}
			k.Websocket.DataHandler <- d
		}
	}
	return nil
}

// wsProcessTickers converts ticker data and sends it to the datahandler
func (k *Kraken) wsProcessTickers(dataRaw json.RawMessage, pair currency.Pair) error {
	var t wsTicker
	if err := json.Unmarshal(dataRaw, &t); err != nil {
		return fmt.Errorf("error unmarshalling ticker data: %w", err)
	}

	k.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: k.Name,
		Ask:          t.Ask[0].Float64(),
		Bid:          t.Bid[0].Float64(),
		Close:        t.Last[0].Float64(),
		Volume:       t.Volume[0].Float64(),
		Low:          t.Low[0].Float64(),
		High:         t.High[0].Float64(),
		Open:         t.Open[0].Float64(),
		AssetType:    asset.Spot,
		Pair:         pair,
	}
	return nil
}

// wsProcessSpread converts spread/orderbook data and sends it to the datahandler
func (k *Kraken) wsProcessSpread(rawData json.RawMessage, pair currency.Pair) error {
	var data wsSpread
	if err := json.Unmarshal(rawData, &data); err != nil {
		return fmt.Errorf("error unmarshalling spread data: %w", err)
	}
	if k.Verbose {
		log.Debugf(log.ExchangeSys, "%s Spread data for %q received. Best bid: '%v' Best ask: '%v' Time: %q, Bid volume: '%v', Ask volume: '%v'",
			k.Name,
			pair,
			data.Bid.Float64(),
			data.Ask.Float64(),
			data.Time.Time(),
			data.BidVolume.Float64(),
			data.AskVolume.Float64())
	}
	return nil
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (k *Kraken) wsProcessTrades(respRaw json.RawMessage, pair currency.Pair) error {
	saveTradeData := k.IsSaveTradeDataEnabled()
	tradeFeed := k.IsTradeFeedEnabled()
	if !saveTradeData && !tradeFeed {
		return nil
	}

	var t []wsTrades
	if err := json.Unmarshal(respRaw, &t); err != nil {
		return fmt.Errorf("error unmarshalling trade data: %w", err)
	}

	trades := make([]trade.Data, len(t))
	for i := range trades {
		side := order.Buy
		if t[i].Side == "s" {
			side = order.Sell
		}
		trades[i] = trade.Data{
			AssetType:    asset.Spot,
			CurrencyPair: pair,
			Exchange:     k.Name,
			Price:        t[i].Price.Float64(),
			Amount:       t[i].Volume.Float64(),
			Timestamp:    t[i].Time.Time().UTC(),
			Side:         side,
		}
	}
	if tradeFeed {
		for i := range trades {
			k.Websocket.DataHandler <- trades[i]
		}
	}
	if saveTradeData {
		return trade.AddTradesToBuffer(trades...)
	}
	return nil
}

func hasKey(raw json.RawMessage, key string) bool {
	_, dataType, _, err := jsonparser.Get(raw, key)
	if err != nil || dataType == jsonparser.NotExist {
		return false
	}
	return true
}

// wsProcessOrderBook handles both partial and full orderbook updates
func (k *Kraken) wsProcessOrderBook(c string, response []json.RawMessage, pair currency.Pair) error {
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

	if isSnapshot := hasKey(response[1], "as") && hasKey(response[1], "bs"); !isSnapshot {
		var update wsUpdate
		if err := json.Unmarshal(response[1], &update); err != nil {
			return fmt.Errorf("error unmarshalling orderbook update: %w", err)
		}
		if len(response) == 5 {
			var update2 wsUpdate
			if err := json.Unmarshal(response[2], &update2); err != nil {
				return fmt.Errorf("error unmarshalling orderbook update: %w", err)
			}
			update.Bids = make([]wsOrderbookItem, len(update2.Bids))
			copy(update.Bids, update2.Bids)
			update.Checksum = update2.Checksum
		}
		err := k.wsProcessOrderBookUpdate(pair, &update)
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

	var snapshot wsSnapshot
	if err := json.Unmarshal(response[1], &snapshot); err != nil {
		return fmt.Errorf("error unmarshalling orderbook snapshot: %w", err)
	}
	return k.wsProcessOrderBookPartial(pair, &snapshot, key.Levels)
}

// wsProcessOrderBookPartial creates a new orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookPartial(pair currency.Pair, obSnapshot *wsSnapshot, levels int) error {
	base := orderbook.Book{
		Pair:                   pair,
		Asset:                  asset.Spot,
		ValidateOrderbook:      k.ValidateOrderbook,
		Bids:                   make(orderbook.Levels, len(obSnapshot.Bids)),
		Asks:                   make(orderbook.Levels, len(obSnapshot.Asks)),
		MaxDepth:               levels,
		ChecksumStringRequired: true,
	}
	// Kraken ob data is timestamped per price, GCT orderbook data is
	// timestamped per entry using the highest last update time, we can attempt
	// to respect both within a reasonable degree
	var highestLastUpdate time.Time
	for i := range obSnapshot.Asks {
		base.Asks[i].Price = obSnapshot.Asks[i].Price
		base.Asks[i].StrPrice = obSnapshot.Asks[i].PriceRaw
		base.Asks[i].Amount = obSnapshot.Asks[i].Amount
		base.Asks[i].StrAmount = obSnapshot.Asks[i].AmountRaw

		askUpdatedTime := obSnapshot.Asks[i].Time.Time()
		if highestLastUpdate.Before(askUpdatedTime) {
			highestLastUpdate = askUpdatedTime
		}
	}

	for i := range obSnapshot.Bids {
		base.Bids[i].Price = obSnapshot.Bids[i].Price
		base.Bids[i].StrPrice = obSnapshot.Bids[i].PriceRaw
		base.Bids[i].Amount = obSnapshot.Bids[i].Amount
		base.Bids[i].StrAmount = obSnapshot.Bids[i].AmountRaw

		bidUpdateTime := obSnapshot.Bids[i].Time.Time()
		if highestLastUpdate.Before(bidUpdateTime) {
			highestLastUpdate = bidUpdateTime
		}
	}
	base.LastUpdated = highestLastUpdate
	base.Exchange = k.Name
	return k.Websocket.Orderbook.LoadSnapshot(&base)
}

// wsProcessOrderBookUpdate updates an orderbook entry for a given currency pair
func (k *Kraken) wsProcessOrderBookUpdate(pair currency.Pair, wsUpdt *wsUpdate) error {
	obUpdate := orderbook.Update{
		Asset: asset.Spot,
		Pair:  pair,
		Bids:  make(orderbook.Levels, len(wsUpdt.Bids)),
		Asks:  make(orderbook.Levels, len(wsUpdt.Asks)),
	}

	// Calculating checksum requires incoming decimal place checks for both
	// price and amount as there is no set standard between currency pairs. This
	// is calculated per update as opposed to snapshot because changes to
	// decimal amounts could occur at any time.
	var highestLastUpdate time.Time
	// Ask data is not always sent
	for i := range wsUpdt.Asks {
		obUpdate.Asks[i].Price = wsUpdt.Asks[i].Price
		obUpdate.Asks[i].StrPrice = wsUpdt.Asks[i].PriceRaw
		obUpdate.Asks[i].Amount = wsUpdt.Asks[i].Amount
		obUpdate.Asks[i].StrAmount = wsUpdt.Asks[i].AmountRaw

		askUpdatedTime := wsUpdt.Asks[i].Time.Time()
		if highestLastUpdate.Before(askUpdatedTime) {
			highestLastUpdate = askUpdatedTime
		}
	}

	// Bid data is not always sent
	for i := range wsUpdt.Bids {
		obUpdate.Bids[i].Price = wsUpdt.Bids[i].Price
		obUpdate.Bids[i].StrPrice = wsUpdt.Bids[i].PriceRaw
		obUpdate.Bids[i].Amount = wsUpdt.Bids[i].Amount
		obUpdate.Bids[i].StrAmount = wsUpdt.Bids[i].AmountRaw

		bidUpdatedTime := wsUpdt.Bids[i].Time.Time()
		if highestLastUpdate.Before(bidUpdatedTime) {
			highestLastUpdate = bidUpdatedTime
		}
	}
	obUpdate.UpdateTime = highestLastUpdate

	err := k.Websocket.Orderbook.Update(&obUpdate)
	if err != nil {
		return err
	}

	book, err := k.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	if err != nil {
		return fmt.Errorf("cannot calculate websocket checksum: book not found for %s %s %w", pair, asset.Spot, err)
	}

	return validateCRC32(book, wsUpdt.Checksum)
}

func validateCRC32(b *orderbook.Book, token uint32) error {
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
func (k *Kraken) wsProcessCandle(c string, resp json.RawMessage, pair currency.Pair) error {
	var data wsCandle
	if err := json.Unmarshal(resp, &data); err != nil {
		return fmt.Errorf("error unmarshalling candle data: %w", err)
	}

	// Faster than getting it through the subscription
	parts := strings.Split(c, "-")
	if len(parts) != 2 {
		return errBadChannelSuffix
	}
	interval := parts[1]

	k.Websocket.DataHandler <- websocket.KlineData{
		AssetType:  asset.Spot,
		Pair:       pair,
		Timestamp:  time.Now(),
		Exchange:   k.Name,
		StartTime:  data.LastUpdateTime.Time(),
		CloseTime:  data.LastUpdateTime.Time(),
		OpenPrice:  data.Open.Float64(),
		HighPrice:  data.High.Float64(),
		LowPrice:   data.Low.Float64(),
		ClosePrice: data.Close.Float64(),
		Volume:     data.Volume.Float64(),
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
	ctx := context.TODO()
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
		k.ParallelChanOp(ctx, groupedSubs, func(ctx context.Context, s subscription.List) error { return k.manageSubs(ctx, krakenWsSubscribe, s) }, 1),
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
	ctx := context.TODO()
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
		k.ParallelChanOp(ctx, subs, func(ctx context.Context, s subscription.List) error { return k.manageSubs(ctx, krakenWsUnsubscribe, s) }, 1),
	)
}

// manageSubs handles both websocket channel subscribe and unsubscribe
func (k *Kraken) manageSubs(ctx context.Context, op string, subs subscription.List) error {
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
		r.Subscription.Token = k.websocketAuthToken()
		conn = k.Websocket.AuthConn
	}

	resps, err := conn.SendMessageReturnResponses(ctx, request.Unset, r.RequestID, r, len(s.Pairs))

	// Ignore an overall timeout, because we'll track individual subscriptions in handleSubResps
	err = common.ExcludeError(err, websocket.ErrSignatureTimeout)
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

// getSubRespErr calls getRespErr and if there's no error from that ensures the status matches the sub operation
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
func (k *Kraken) wsAddOrder(ctx context.Context, req *WsAddOrderRequest) (string, error) {
	if req == nil {
		return "", common.ErrNilPointer
	}
	req.RequestID = k.Websocket.AuthConn.GenerateMessageID(false)
	req.Event = krakenWsAddOrder
	req.Token = k.websocketAuthToken()
	jsonResp, err := k.Websocket.AuthConn.SendMessageReturnResponse(ctx, request.Unset, req.RequestID, req)
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
func (k *Kraken) wsCancelOrders(ctx context.Context, orderIDs []string) error {
	errs := common.CollectErrors(len(orderIDs))
	for _, id := range orderIDs {
		go func() {
			defer errs.Wg.Done()
			errs.C <- k.wsCancelOrder(ctx, id)
		}()
	}

	return errs.Collect()
}

// wsCancelOrder cancels an open order
func (k *Kraken) wsCancelOrder(ctx context.Context, orderID string) error {
	id := k.Websocket.AuthConn.GenerateMessageID(false)
	req := WsCancelOrderRequest{
		Event:          krakenWsCancelOrder,
		Token:          k.websocketAuthToken(),
		TransactionIDs: []string{orderID},
		RequestID:      id,
	}

	resp, err := k.Websocket.AuthConn.SendMessageReturnResponse(ctx, request.Unset, id, req)
	if err != nil {
		return fmt.Errorf("%w %s: %w", errCancellingOrder, orderID, err)
	}

	status, err := jsonparser.GetUnsafeString(resp, "status")
	if err != nil {
		return fmt.Errorf("%w 'status': %w from message: %s", common.ErrParsingWSField, err, resp)
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
func (k *Kraken) wsCancelAllOrders(ctx context.Context) (*WsCancelOrderResponse, error) {
	id := k.Websocket.AuthConn.GenerateMessageID(false)
	req := WsCancelOrderRequest{
		Event:     krakenWsCancelAll,
		Token:     k.websocketAuthToken(),
		RequestID: id,
	}

	jsonResp, err := k.Websocket.AuthConn.SendMessageReturnResponse(ctx, request.Unset, id, req)
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

// websocketAuthToken retrieves the current websocket session's auth token
func (k *Kraken) websocketAuthToken() string {
	k.wsAuthMtx.RLock()
	defer k.wsAuthMtx.RUnlock()
	return k.wsAuthToken
}

func (k *Kraken) setWebsocketAuthToken(token string) {
	k.wsAuthMtx.Lock()
	k.wsAuthToken = token
	k.wsAuthMtx.Unlock()
}
