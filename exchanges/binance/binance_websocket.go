package binance

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443/stream"
	pingDelay                  = time.Minute * 9

	wsSubscribeMethod         = "SUBSCRIBE"
	wsUnsubscribeMethod       = "UNSUBSCRIBE"
	wsListSubscriptionsMethod = "LIST_SUBSCRIPTIONS"
)

var listenKey string

var (
	// maxWSUpdateBuffer defines max websocket updates to apply when an
	// orderbook is initially fetched
	maxWSUpdateBuffer = 150
	// maxWSOrderbookJobs defines max websocket orderbook jobs in queue to fetch
	// an orderbook snapshot via REST
	maxWSOrderbookJobs = 2000
	// maxWSOrderbookWorkers defines a max amount of workers allowed to execute
	// jobs from the job channel
	maxWSOrderbookWorkers = 10
)

// WsConnect initiates a websocket connection
func (e *Exchange) WsConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}

	var dialer gws.Dialer
	dialer.HandshakeTimeout = e.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	var err error
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err = e.GetWsAuthStreamKey(ctx)
		if err != nil {
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys,
				"%v unable to connect to authenticated Websocket. Error: %s",
				e.Name,
				err)
		} else {
			// cleans on failed connection
			clean := strings.Split(e.Websocket.GetWebsocketURL(), "?streams=")
			authPayload := clean[0] + "?streams=" + listenKey
			err = e.Websocket.SetWebsocketURL(authPayload, false, false)
			if err != nil {
				return err
			}
		}
	}

	err = e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s",
			e.Name,
			err)
	}

	if e.Websocket.CanUseAuthenticatedEndpoints() {
		go e.KeepAuthKeyAlive(ctx)
	}

	e.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PongMessage,
		Delay:             pingDelay,
	})

	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)

	e.setupOrderbookManager(ctx)
	return nil
}

func (e *Exchange) setupOrderbookManager(ctx context.Context) {
	if e.obm == nil {
		e.obm = &orderbookManager{
			state: make(map[currency.Code]map[currency.Code]map[asset.Item]*update),
			jobs:  make(chan job, maxWSOrderbookJobs),
		}
	} else {
		// Change state on reconnect for initial sync.
		for _, m1 := range e.obm.state {
			for _, m2 := range m1 {
				for _, update := range m2 {
					update.initialSync = true
					update.needsFetchingBook = true
					update.lastUpdateID = 0
				}
			}
		}
	}

	for range maxWSOrderbookWorkers {
		// 10 workers for synchronising book
		e.SynchroniseWebsocketOrderbook(ctx)
	}
}

// KeepAuthKeyAlive will continuously send messages to
// keep the WS auth key active
func (e *Exchange) KeepAuthKeyAlive(ctx context.Context) {
	e.Websocket.Wg.Add(1)
	defer e.Websocket.Wg.Done()
	for {
		select {
		case <-e.Websocket.ShutdownC:
			return
		case <-time.After(time.Minute * 30):
			if err := e.MaintainWsAuthStreamKey(ctx); err != nil {
				if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
					log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
				}
				log.Warnf(log.ExchangeSys, "%s %s: Unable to renew auth websocket token, may experience shutdown", e.Name, e.Websocket.Conn.GetURL())
			}
		}
	}
}

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData(ctx context.Context) {
	defer e.Websocket.Wg.Done()

	for {
		resp := e.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		if err := e.wsHandleData(ctx, resp.Raw); err != nil {
			if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
				log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
			}
		}
	}
}

func (e *Exchange) wsHandleData(ctx context.Context, respRaw []byte) error {
	if id, err := jsonparser.GetString(respRaw, "id"); err == nil {
		if e.Websocket.Match.IncomingWithData(id, respRaw) {
			return nil
		}
	}

	if resultString, err := jsonparser.GetUnsafeString(respRaw, "result"); err == nil {
		if resultString == "null" {
			return nil
		}
	}
	jsonData, _, _, err := jsonparser.Get(respRaw, "data")
	if err != nil {
		return fmt.Errorf("%s %s %s", e.Name, websocket.UnhandledMessage, string(respRaw))
	}
	var event string
	event, err = jsonparser.GetUnsafeString(jsonData, "e")
	if err == nil {
		switch event {
		case "outboundAccountPosition":
			var data WsAccountPositionData
			err = json.Unmarshal(jsonData, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to outboundAccountPosition structure %s",
					e.Name,
					err)
			}
			return e.Websocket.DataHandler.Send(ctx, data)
		case "balanceUpdate":
			var data WsBalanceUpdateData
			err = json.Unmarshal(jsonData, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to balanceUpdate structure %s",
					e.Name,
					err)
			}
			return e.Websocket.DataHandler.Send(ctx, data)
		case "executionReport":
			var data WsOrderUpdateData
			err = json.Unmarshal(jsonData, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to executionReport structure %s",
					e.Name,
					err)
			}
			avgPrice := 0.0
			if data.CumulativeFilledQuantity != 0 {
				avgPrice = data.CumulativeQuoteTransactedQuantity / data.CumulativeFilledQuantity
			}
			remainingAmount := data.Quantity - data.CumulativeFilledQuantity
			var pair currency.Pair
			var assetType asset.Item
			pair, assetType, err = e.GetRequestFormattedPairAndAssetType(data.Symbol)
			if err != nil {
				return err
			}
			var feeAsset currency.Code
			if data.CommissionAsset != "" {
				feeAsset = currency.NewCode(data.CommissionAsset)
			}
			orderID := strconv.FormatInt(data.OrderID, 10)
			var orderStatus order.Status
			orderStatus, err = stringToOrderStatus(data.OrderStatus)
			if err != nil {
				return err
			}
			clientOrderID := data.ClientOrderID
			if orderStatus == order.Cancelled {
				clientOrderID = data.CancelledClientOrderID
			}
			var orderType order.Type
			orderType, err = order.StringToOrderType(data.OrderType)
			if err != nil {
				return err
			}
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(data.Side)
			if err != nil {
				return err
			}
			return e.Websocket.DataHandler.Send(ctx, &order.Detail{
				Price:                data.Price,
				Amount:               data.Quantity,
				AverageExecutedPrice: avgPrice,
				ExecutedAmount:       data.CumulativeFilledQuantity,
				RemainingAmount:      remainingAmount,
				Cost:                 data.CumulativeQuoteTransactedQuantity,
				CostAsset:            pair.Quote,
				Fee:                  data.Commission,
				FeeAsset:             feeAsset,
				Exchange:             e.Name,
				OrderID:              orderID,
				ClientOrderID:        clientOrderID,
				Type:                 orderType,
				Side:                 orderSide,
				Status:               orderStatus,
				AssetType:            assetType,
				Date:                 data.OrderCreationTime.Time(),
				LastUpdated:          data.TransactionTime.Time(),
				Pair:                 pair,
			})
		case "listStatus":
			var data WsListStatusData
			err = json.Unmarshal(jsonData, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to listStatus structure %s",
					e.Name,
					err)
			}
			return e.Websocket.DataHandler.Send(ctx, data)
		}
	}

	streamStr, err := jsonparser.GetUnsafeString(respRaw, "stream")
	if err != nil {
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return fmt.Errorf("%s %s %s", e.Name, websocket.UnhandledMessage, string(respRaw))
		}
		return err
	}
	streamType := strings.Split(streamStr, "@")
	if len(streamType) <= 1 {
		return fmt.Errorf("%s %s %s", e.Name, websocket.UnhandledMessage, string(respRaw))
	}
	var (
		pair      currency.Pair
		isEnabled bool
		symbol    string
	)
	symbol, err = jsonparser.GetUnsafeString(jsonData, "s")
	if err != nil {
		// there should be a symbol returned for all data types below
		return err
	}
	pair, isEnabled, err = e.MatchSymbolCheckEnabled(symbol, asset.Spot, false)
	if err != nil {
		return err
	}
	if !isEnabled {
		return nil
	}
	switch streamType[1] {
	case "trade":
		saveTradeData := e.IsSaveTradeDataEnabled()
		if !saveTradeData &&
			!e.IsTradeFeedEnabled() {
			return nil
		}

		var t TradeStream
		err := json.Unmarshal(jsonData, &t)
		if err != nil {
			return fmt.Errorf("%v - Could not unmarshal trade data: %s",
				e.Name,
				err)
		}
		td := trade.Data{
			CurrencyPair: pair,
			Timestamp:    t.TimeStamp.Time(),
			Price:        t.Price.Float64(),
			Amount:       t.Quantity.Float64(),
			Exchange:     e.Name,
			AssetType:    asset.Spot,
			TID:          strconv.FormatInt(t.TradeID, 10),
		}

		if t.IsBuyerMaker { // Seller is Taker
			td.Side = order.Sell
		} else { // Buyer is Taker
			td.Side = order.Buy
		}
		return e.Websocket.Trade.Update(saveTradeData, td)
	case "ticker":
		var t TickerStream
		err = json.Unmarshal(jsonData, &t)
		if err != nil {
			return fmt.Errorf("%v - Could not convert to a TickerStream structure %s",
				e.Name,
				err.Error())
		}
		return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
			ExchangeName: e.Name,
			Open:         t.OpenPrice.Float64(),
			Close:        t.ClosePrice.Float64(),
			Volume:       t.TotalTradedVolume.Float64(),
			QuoteVolume:  t.TotalTradedQuoteVolume.Float64(),
			High:         t.HighPrice.Float64(),
			Low:          t.LowPrice.Float64(),
			Bid:          t.BestBidPrice.Float64(),
			Ask:          t.BestAskPrice.Float64(),
			Last:         t.LastPrice.Float64(),
			LastUpdated:  t.EventTime.Time(),
			AssetType:    asset.Spot,
			Pair:         pair,
		})
	case "kline_1m", "kline_3m", "kline_5m", "kline_15m", "kline_30m", "kline_1h", "kline_2h", "kline_4h",
		"kline_6h", "kline_8h", "kline_12h", "kline_1d", "kline_3d", "kline_1w", "kline_1M":
		var kline KlineStream
		err = json.Unmarshal(jsonData, &kline)
		if err != nil {
			return fmt.Errorf("%v - Could not convert to a KlineStream structure %s",
				e.Name,
				err)
		}
		return e.Websocket.DataHandler.Send(ctx, websocket.KlineData{
			Timestamp:  kline.EventTime.Time(),
			Pair:       pair,
			AssetType:  asset.Spot,
			Exchange:   e.Name,
			StartTime:  kline.Kline.StartTime.Time(),
			CloseTime:  kline.Kline.CloseTime.Time(),
			Interval:   kline.Kline.Interval,
			OpenPrice:  kline.Kline.OpenPrice.Float64(),
			ClosePrice: kline.Kline.ClosePrice.Float64(),
			HighPrice:  kline.Kline.HighPrice.Float64(),
			LowPrice:   kline.Kline.LowPrice.Float64(),
			Volume:     kline.Kline.Volume.Float64(),
		})
	case "depth":
		var depth WebsocketDepthStream
		err = json.Unmarshal(jsonData, &depth)
		if err != nil {
			return fmt.Errorf("%v - Could not convert to depthStream structure %s",
				e.Name,
				err)
		}
		var init bool
		init, err = e.UpdateLocalBuffer(&depth)
		if err != nil {
			if init {
				return nil
			}
			return fmt.Errorf("%v - UpdateLocalCache error: %s",
				e.Name,
				err)
		}
		return nil
	default:
		return fmt.Errorf("%s %s %s", e.Name, websocket.UnhandledMessage, string(respRaw))
	}
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "NEW":
		return order.New, nil
	case "PARTIALLY_FILLED":
		return order.PartiallyFilled, nil
	case "FILLED":
		return order.Filled, nil
	case "CANCELED":
		return order.Cancelled, nil
	case "PENDING_CANCEL":
		return order.PendingCancel, nil
	case "REJECTED":
		return order.Rejected, nil
	case "EXPIRED":
		return order.Expired, nil
	default:
		return order.UnknownStatus, errors.New(status + " not recognised as order status")
	}
}

// SeedLocalCache seeds depth data
func (e *Exchange) SeedLocalCache(ctx context.Context, p currency.Pair) error {
	ob, err := e.GetOrderBook(ctx, p, 1000)
	if err != nil {
		return err
	}
	return e.SeedLocalCacheWithBook(p, ob)
}

// SeedLocalCacheWithBook seeds the local orderbook cache
func (e *Exchange) SeedLocalCacheWithBook(p currency.Pair, orderbookNew *OrderBookResponse) error {
	t := orderbookNew.Timestamp.Time()
	if t.IsZero() {
		t = time.Now() // Time not provided for this REST book.
	}
	newOrderBook := orderbook.Book{
		Pair:              p,
		Asset:             asset.Spot,
		Exchange:          e.Name,
		LastUpdateID:      orderbookNew.LastUpdateID,
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              orderbookNew.Bids.Levels(),
		Asks:              orderbookNew.Asks.Levels(),
		LastUpdated:       t,
	}
	return e.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// UpdateLocalBuffer updates and returns the most recent iteration of the orderbook
func (e *Exchange) UpdateLocalBuffer(wsdp *WebsocketDepthStream) (bool, error) {
	pair, err := e.MatchSymbolWithAvailablePairs(wsdp.Pair, asset.Spot, false)
	if err != nil {
		return false, err
	}
	err = e.obm.stageWsUpdate(wsdp, pair, asset.Spot)
	if err != nil {
		init, err2 := e.obm.checkIsInitialSync(pair)
		if err2 != nil {
			return false, err2
		}
		return init, err
	}

	err = e.applyBufferUpdate(pair)
	if err != nil {
		e.invalidateAndCleanupOrderbook(pair)
	}

	return false, err
}

func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	for _, s := range e.Features.Subscriptions {
		if s.Asset == asset.Empty {
			// Handle backwards compatibility with config without assets, all binance subs are spot
			s.Asset = asset.Spot
		}
	}
	return e.Features.Subscriptions.ExpandTemplates(e)
}

var subTemplate *template.Template

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	var err error
	if subTemplate == nil {
		subTemplate, err = template.New("subscriptions.tmpl").
			Funcs(template.FuncMap{
				"interval": formatChannelInterval,
				"levels":   formatChannelLevels,
				"fmt":      currency.EMPTYFORMAT.Format,
			}).
			Parse(subTplText)
	}
	return subTemplate, err
}

func formatChannelLevels(s *subscription.Subscription) string {
	if s.Levels != 0 {
		return strconv.Itoa(s.Levels)
	}
	return ""
}

func formatChannelInterval(s *subscription.Subscription) string {
	switch s.Channel {
	case subscription.OrderbookChannel:
		if s.Interval.Duration() == time.Second {
			return "@1000ms"
		}
		return "@" + s.Interval.Short()
	case subscription.CandlesChannel:
		return "_" + s.Interval.Short()
	}
	return ""
}

// Subscribe subscribes to a set of channels
func (e *Exchange) Subscribe(channels subscription.List) error {
	ctx := context.TODO()
	return e.ParallelChanOp(ctx, channels, func(ctx context.Context, l subscription.List) error { return e.manageSubs(ctx, wsSubscribeMethod, l) }, 50)
}

// Unsubscribe unsubscribes from a set of channels
func (e *Exchange) Unsubscribe(channels subscription.List) error {
	ctx := context.TODO()
	return e.ParallelChanOp(ctx, channels, func(ctx context.Context, l subscription.List) error { return e.manageSubs(ctx, wsUnsubscribeMethod, l) }, 50)
}

// manageSubs subscribes or unsubscribes from a list of subscriptions
func (e *Exchange) manageSubs(ctx context.Context, op string, subs subscription.List) error {
	if op == wsSubscribeMethod {
		if err := e.Websocket.AddSubscriptions(e.Websocket.Conn, subs...); err != nil { // Note: AddSubscription will set state to subscribing
			return err
		}
	} else {
		if err := subs.SetStates(subscription.UnsubscribingState); err != nil {
			return err
		}
	}

	req := WsPayload{
		ID:     e.MessageID(),
		Method: op,
		Params: subs.QualifiedChannels(),
	}

	respRaw, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err == nil {
		if v, d, _, rErr := jsonparser.Get(respRaw, "result"); rErr != nil {
			err = rErr
		} else if d != jsonparser.Null { // null is the only expected and acceptable response
			err = fmt.Errorf("%w: %s", common.ErrUnknownError, v)
		}
	}

	if err != nil {
		err = fmt.Errorf("%w; Channels: %s", err, strings.Join(subs.QualifiedChannels(), ", "))
		if op == wsSubscribeMethod {
			if err2 := e.Websocket.RemoveSubscriptions(e.Websocket.Conn, subs...); err2 != nil {
				err = common.AppendError(err, err2)
			}
		}
	} else {
		if op == wsSubscribeMethod {
			err = common.AppendError(err, subs.SetStates(subscription.SubscribedState))
		} else {
			err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, subs...)
		}
	}

	return err
}

// ProcessOrderbookUpdate processes the websocket orderbook update
func (e *Exchange) ProcessOrderbookUpdate(cp currency.Pair, a asset.Item, ws *WebsocketDepthStream) error {
	return e.Websocket.Orderbook.Update(&orderbook.Update{
		Bids:       ws.UpdateBids.Levels(),
		Asks:       ws.UpdateAsks.Levels(),
		Pair:       cp,
		UpdateID:   ws.LastUpdateID,
		UpdateTime: ws.Timestamp.Time(),
		Asset:      a,
	})
}

// applyBufferUpdate applies the buffer to the orderbook or initiates a new
// orderbook sync by the REST protocol which is off handed to go routine.
func (e *Exchange) applyBufferUpdate(pair currency.Pair) error {
	fetching, needsFetching, err := e.obm.handleFetchingBook(pair)
	if err != nil {
		return err
	}
	if fetching {
		return nil
	}
	if needsFetching {
		if e.Verbose {
			log.Debugf(log.WebsocketMgr, "%s Orderbook: Fetching via REST\n", e.Name)
		}
		return e.obm.fetchBookViaREST(pair)
	}

	recent, err := e.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	if err != nil {
		log.Errorf(
			log.WebsocketMgr,
			"%s error fetching recent orderbook when applying updates: %s\n",
			e.Name,
			err)
	}

	if recent != nil {
		err = e.obm.checkAndProcessOrderbookUpdate(e.ProcessOrderbookUpdate, pair, recent)
		if err != nil {
			log.Errorf(
				log.WebsocketMgr,
				"%s error processing update - initiating new orderbook sync via REST: %s\n",
				e.Name,
				err)
			err = e.obm.setNeedsFetchingBook(pair)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// setNeedsFetchingBook completes the book fetching initiation.
func (o *orderbookManager) setNeedsFetchingBook(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			asset.Spot)
	}
	state.needsFetchingBook = true
	return nil
}

// SynchroniseWebsocketOrderbook synchronises full orderbook for currency pair
// asset
func (e *Exchange) SynchroniseWebsocketOrderbook(ctx context.Context) {
	e.Websocket.Wg.Go(func() {
		for {
			select {
			case <-e.Websocket.ShutdownC:
				for {
					select {
					case <-e.obm.jobs:
					default:
						return
					}
				}
			case j := <-e.obm.jobs:
				if err := e.processJob(ctx, j.Pair); err != nil {
					log.Errorf(log.WebsocketMgr, "%s processing websocket orderbook error: %v", e.Name, err)
				}
			}
		}
	})
}

// processJob fetches and processes orderbook updates
func (e *Exchange) processJob(ctx context.Context, p currency.Pair) error {
	err := e.SeedLocalCache(ctx, p)
	if err != nil {
		return fmt.Errorf("%s %s seeding local cache for orderbook error: %v",
			p, asset.Spot, err)
	}

	err = e.obm.stopFetchingBook(p)
	if err != nil {
		return err
	}

	// Immediately apply the buffer updates so we don't wait for a
	// new update to initiate this.
	err = e.applyBufferUpdate(p)
	if err != nil {
		e.invalidateAndCleanupOrderbook(p)
		return err
	}
	return nil
}

// invalidateAndCleanupOrderbook invalidaates orderbook and cleans local cache
func (e *Exchange) invalidateAndCleanupOrderbook(p currency.Pair) {
	if err := e.Websocket.Orderbook.InvalidateOrderbook(p, asset.Spot); err != nil {
		log.Errorf(log.WebsocketMgr, "%s error invalidating websocket orderbook: %v", e.Name, err)
	}
	if err := e.obm.cleanup(p); err != nil {
		log.Errorf(log.WebsocketMgr, "%s error during websocket orderbook cleanup: %v", e.Name, err)
	}
}

// stageWsUpdate stages websocket update to roll through updates that need to
// be applied to a fetched orderbook via REST.
func (o *orderbookManager) stageWsUpdate(u *WebsocketDepthStream, pair currency.Pair, a asset.Item) error {
	o.Lock()
	defer o.Unlock()
	m1, ok := o.state[pair.Base]
	if !ok {
		m1 = make(map[currency.Code]map[asset.Item]*update)
		o.state[pair.Base] = m1
	}

	m2, ok := m1[pair.Quote]
	if !ok {
		m2 = make(map[asset.Item]*update)
		m1[pair.Quote] = m2
	}

	state, ok := m2[a]
	if !ok {
		state = &update{
			// 100ms update assuming we might have up to a 10 second delay.
			// There could be a potential 100 updates for the currency.
			buffer:            make(chan *WebsocketDepthStream, maxWSUpdateBuffer),
			fetchingBook:      false,
			initialSync:       true,
			needsFetchingBook: true,
		}
		m2[a] = state
	}

	if state.lastUpdateID != 0 && u.FirstUpdateID != state.lastUpdateID+1 {
		// While listening to the stream, each new event's U should be
		// equal to the previous event's u+1.
		return fmt.Errorf("websocket orderbook synchronisation failure for pair %s and asset %s", pair, a)
	}
	state.lastUpdateID = u.LastUpdateID

	select {
	// Put update in the channel buffer to be processed
	case state.buffer <- u:
		return nil
	default:
		<-state.buffer    // pop one element
		state.buffer <- u // to shift buffer on fail
		return fmt.Errorf("channel blockage for %s, asset %s and connection",
			pair, a)
	}
}

// handleFetchingBook checks if a full book is being fetched or needs to be
// fetched
func (o *orderbookManager) handleFetchingBook(pair currency.Pair) (fetching, needsFetching bool, err error) {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return false,
			false,
			fmt.Errorf("check is fetching book cannot match currency pair %s asset type %s",
				pair,
				asset.Spot)
	}

	if state.fetchingBook {
		return true, false, nil
	}

	if state.needsFetchingBook {
		state.needsFetchingBook = false
		state.fetchingBook = true
		return false, true, nil
	}
	return false, false, nil
}

// stopFetchingBook completes the book fetching.
func (o *orderbookManager) stopFetchingBook(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			asset.Spot)
	}
	if !state.fetchingBook {
		return fmt.Errorf("fetching book already set to false for %s %s",
			pair,
			asset.Spot)
	}
	state.fetchingBook = false
	return nil
}

// completeInitialSync sets if an asset type has completed its initial sync
func (o *orderbookManager) completeInitialSync(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("complete initial sync cannot match currency pair %s asset type %s",
			pair,
			asset.Spot)
	}
	if !state.initialSync {
		return fmt.Errorf("initial sync already set to false for %s %s",
			pair,
			asset.Spot)
	}
	state.initialSync = false
	return nil
}

// checkIsInitialSync checks status if the book is Initial Sync being via the REST
// protocol.
func (o *orderbookManager) checkIsInitialSync(pair currency.Pair) (bool, error) {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return false,
			fmt.Errorf("checkIsInitialSync of orderbook cannot match currency pair %s asset type %s",
				pair,
				asset.Spot)
	}
	return state.initialSync, nil
}

// fetchBookViaREST pushes a job of fetching the orderbook via the REST protocol
// to get an initial full book that we can apply our buffered updates too.
func (o *orderbookManager) fetchBookViaREST(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()

	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("fetch book via rest cannot match currency pair %s asset type %s",
			pair,
			asset.Spot)
	}

	state.initialSync = true
	state.fetchingBook = true

	select {
	case o.jobs <- job{pair}:
		return nil
	default:
		return fmt.Errorf("%s %s book synchronisation channel blocked up",
			pair,
			asset.Spot)
	}
}

func (o *orderbookManager) checkAndProcessOrderbookUpdate(processor func(currency.Pair, asset.Item, *WebsocketDepthStream) error, pair currency.Pair, recent *orderbook.Book) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair [%s] asset type [%s] in hash table to process websocket orderbook update",
			pair, asset.Spot)
	}

	// This will continuously remove updates from the buffered channel and
	// apply them to the current orderbook.
buffer:
	for {
		select {
		case d := <-state.buffer:
			process, err := state.validate(d, recent)
			if err != nil {
				return err
			}
			if process {
				err := processor(pair, asset.Spot, d)
				if err != nil {
					return fmt.Errorf("%s %s processing update error: %w",
						pair, asset.Spot, err)
				}
			}
		default:
			break buffer
		}
	}
	return nil
}

// validate checks for correct update alignment
func (u *update) validate(updt *WebsocketDepthStream, recent *orderbook.Book) (bool, error) {
	if updt.LastUpdateID <= recent.LastUpdateID {
		// Drop any event where u is <= lastUpdateId in the snapshot.
		return false, nil
	}

	id := recent.LastUpdateID + 1
	if u.initialSync {
		// The first processed event should have U <= lastUpdateId+1 AND
		// u >= lastUpdateId+1.
		if updt.FirstUpdateID > id || updt.LastUpdateID < id {
			return false, fmt.Errorf("initial websocket orderbook sync failure for pair %s and asset %s",
				recent.Pair,
				asset.Spot)
		}
		u.initialSync = false
	}
	return true, nil
}

// cleanup cleans up buffer and reset fetch and init
func (o *orderbookManager) cleanup(pair currency.Pair) error {
	o.Lock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		o.Unlock()
		return fmt.Errorf("cleanup cannot match %s %s to hash table",
			pair,
			asset.Spot)
	}

bufferEmpty:
	for {
		select {
		case <-state.buffer:
			// bleed and discard buffer
		default:
			break bufferEmpty
		}
	}
	o.Unlock()
	// disable rest orderbook synchronisation
	_ = o.stopFetchingBook(pair)
	_ = o.completeInitialSync(pair)
	_ = o.stopNeedsFetchingBook(pair)
	return nil
}

// stopNeedsFetchingBook completes the book fetching initiation.
func (o *orderbookManager) stopNeedsFetchingBook(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			asset.Spot)
	}
	if !state.needsFetchingBook {
		return fmt.Errorf("needs fetching book already set to false for %s %s",
			pair,
			asset.Spot)
	}
	state.needsFetchingBook = false
	return nil
}

const subTplText = `
{{ range $pair := index $.AssetPairs $.S.Asset }}
  {{ fmt $pair -}} @
  {{- with $c := $.S.Channel -}}
  {{ if eq $c "ticker"         -}} ticker
  {{ else if eq $c "allTrades" -}} trade
  {{ else if eq $c "candles"   -}} kline  {{- interval $.S }}
  {{ else if eq $c "orderbook" -}} depth  {{- levels $.S }}{{ interval $.S }}
  {{- end }}{{ end }}
  {{ $.PairSeparator }}
{{end}}
`
