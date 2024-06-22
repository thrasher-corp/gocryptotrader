package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
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
	errUnknownError       = errors.New("unknown error")
)

// WsConnect initiates a websocket connection
func (b *Binance) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}

	var dialer websocket.Dialer
	dialer.HandshakeTimeout = b.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	var err error
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err = b.GetWsAuthStreamKey(context.TODO())
		if err != nil {
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys,
				"%v unable to connect to authenticated Websocket. Error: %s",
				b.Name,
				err)
		} else {
			// cleans on failed connection
			clean := strings.Split(b.Websocket.GetWebsocketURL(), "?streams=")
			authPayload := clean[0] + "?streams=" + listenKey
			err = b.Websocket.SetWebsocketURL(authPayload, false, false)
			if err != nil {
				return err
			}
		}
	}

	err = b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s",
			b.Name,
			err)
	}

	if b.Websocket.CanUseAuthenticatedEndpoints() {
		go b.KeepAuthKeyAlive()
	}

	b.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PongMessage,
		Delay:             pingDelay,
	})

	b.Websocket.Wg.Add(1)
	go b.wsReadData()

	b.setupOrderbookManager()

	err = b.WsConnectAPI()
	if err != nil {
		b.SetIsAPIStreamConnected(false)
		log.Errorf(log.ExchangeSys, "could not connect to API stream %v", err)
		return err
	}
	b.SetIsAPIStreamConnected(true)
	return nil
}

func (b *Binance) setupOrderbookManager() {
	if b.obm == nil {
		b.obm = &orderbookManager{
			state: make(map[currency.Code]map[currency.Code]map[asset.Item]*update),
			jobs:  make(chan job, maxWSOrderbookJobs),
		}
	} else {
		// Change state on reconnect for initial sync.
		for _, m1 := range b.obm.state {
			for _, m2 := range m1 {
				for _, update := range m2 {
					update.initialSync = true
					update.needsFetchingBook = true
					update.lastUpdateID = 0
				}
			}
		}
	}

	for i := 0; i < maxWSOrderbookWorkers; i++ {
		// 10 workers for synchronising book
		b.SynchroniseWebsocketOrderbook()
	}
}

// KeepAuthKeyAlive will continuously send messages to
// keep the WS auth key active
func (b *Binance) KeepAuthKeyAlive() {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()
	ticks := time.NewTicker(time.Minute * 30)
	for {
		select {
		case <-b.Websocket.ShutdownC:
			ticks.Stop()
			return
		case <-ticks.C:
			err := b.MaintainWsAuthStreamKey(context.TODO())
			if err != nil {
				b.Websocket.DataHandler <- err
				log.Warnf(log.ExchangeSys,
					b.Name+" - Unable to renew auth websocket token, may experience shutdown")
			}
		}
	}
}

// wsReadData receives and passes on websocket messages for processing
func (b *Binance) wsReadData() {
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

func (b *Binance) wsHandleData(respRaw []byte) error {
	if id, err := jsonparser.GetInt(respRaw, "id"); err == nil {
		if b.Websocket.Match.IncomingWithData(id, respRaw) {
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
		return fmt.Errorf("%s %s %s", b.Name, stream.UnhandledMessage, string(respRaw))
	}
	var event string
	event, err = jsonparser.GetUnsafeString(jsonData, "e")
	if err == nil {
		switch event {
		case "outboundAccountPosition":
			var data wsAccountPosition
			err = json.Unmarshal(respRaw, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to outboundAccountPosition structure %s",
					b.Name,
					err)
			}
			b.Websocket.DataHandler <- data
			return nil
		case "balanceUpdate":
			var data wsBalanceUpdate
			err = json.Unmarshal(respRaw, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to balanceUpdate structure %s",
					b.Name,
					err)
			}
			b.Websocket.DataHandler <- data
			return nil
		case "executionReport":
			var data wsOrderUpdate
			err = json.Unmarshal(respRaw, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to executionReport structure %s",
					b.Name,
					err)
			}
			avgPrice := 0.0
			if data.Data.CumulativeFilledQuantity != 0 {
				avgPrice = data.Data.CumulativeQuoteTransactedQuantity / data.Data.CumulativeFilledQuantity
			}
			remainingAmount := data.Data.Quantity - data.Data.CumulativeFilledQuantity
			var pair currency.Pair
			var assetType asset.Item
			pair, assetType, err = b.GetRequestFormattedPairAndAssetType(data.Data.Symbol)
			if err != nil {
				return err
			}
			var feeAsset currency.Code
			if data.Data.CommissionAsset != "" {
				feeAsset = currency.NewCode(data.Data.CommissionAsset)
			}
			orderID := strconv.FormatInt(data.Data.OrderID, 10)
			var orderStatus order.Status
			orderStatus, err = stringToOrderStatus(data.Data.OrderStatus)
			if err != nil {
				b.Websocket.DataHandler <- order.ClassificationError{
					Exchange: b.Name,
					OrderID:  orderID,
					Err:      err,
				}
			}
			clientOrderID := data.Data.ClientOrderID
			if orderStatus == order.Cancelled {
				clientOrderID = data.Data.CancelledClientOrderID
			}
			var orderType order.Type
			orderType, err = order.StringToOrderType(data.Data.OrderType)
			if err != nil {
				b.Websocket.DataHandler <- order.ClassificationError{
					Exchange: b.Name,
					OrderID:  orderID,
					Err:      err,
				}
			}
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(data.Data.Side)
			if err != nil {
				b.Websocket.DataHandler <- order.ClassificationError{
					Exchange: b.Name,
					OrderID:  orderID,
					Err:      err,
				}
			}
			b.Websocket.DataHandler <- &order.Detail{
				Price:                data.Data.Price,
				Amount:               data.Data.Quantity,
				AverageExecutedPrice: avgPrice,
				ExecutedAmount:       data.Data.CumulativeFilledQuantity,
				RemainingAmount:      remainingAmount,
				Cost:                 data.Data.CumulativeQuoteTransactedQuantity,
				CostAsset:            pair.Quote,
				Fee:                  data.Data.Commission,
				FeeAsset:             feeAsset,
				Exchange:             b.Name,
				OrderID:              orderID,
				ClientOrderID:        clientOrderID,
				Type:                 orderType,
				Side:                 orderSide,
				Status:               orderStatus,
				AssetType:            assetType,
				Date:                 data.Data.OrderCreationTime.Time(),
				LastUpdated:          data.Data.TransactionTime.Time(),
				Pair:                 pair,
			}
			return nil
		case "listStatus":
			var data wsListStatus
			err = json.Unmarshal(respRaw, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to listStatus structure %s",
					b.Name,
					err)
			}
			b.Websocket.DataHandler <- data
			return nil
		case "outboundAccountInfo":
			var data wsAccountInfo
			err = json.Unmarshal(respRaw, &data)
			if err != nil {
				return fmt.Errorf("%v - Could not convert to outboundAccountInfo structure %s",
					b.Name,
					err)
			}
			b.Websocket.DataHandler <- data
			return nil
		}
	}

	streamStr, err := jsonparser.GetUnsafeString(respRaw, "stream")
	if err != nil {
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return fmt.Errorf("%s %s %s", b.Name, stream.UnhandledMessage, string(respRaw))
		}
		return err
	}
	streamType := strings.Split(streamStr, "@")
	if len(streamType) <= 1 {
		return fmt.Errorf("%s %s %s", b.Name, stream.UnhandledMessage, string(respRaw))
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
	pair, isEnabled, err = b.MatchSymbolCheckEnabled(symbol, asset.Spot, false)
	if err != nil {
		return err
	}
	if !isEnabled {
		return nil
	}
	switch streamType[1] {
	case "trade":
		saveTradeData := b.IsSaveTradeDataEnabled()
		if !saveTradeData &&
			!b.IsTradeFeedEnabled() {
			return nil
		}

		var t TradeStream
		err = json.Unmarshal(jsonData, &t)
		if err != nil {
			return fmt.Errorf("%v - Could not unmarshal trade data: %s",
				b.Name,
				err)
		}
		return b.Websocket.Trade.Update(saveTradeData,
			trade.Data{
				CurrencyPair: pair,
				Timestamp:    t.TimeStamp.Time(),
				Price:        t.Price.Float64(),
				Amount:       t.Quantity.Float64(),
				Exchange:     b.Name,
				AssetType:    asset.Spot,
				TID:          strconv.FormatInt(t.TradeID, 10),
			})
	case "ticker":
		var t TickerStream
		err = json.Unmarshal(jsonData, &t)
		if err != nil {
			return fmt.Errorf("%v - Could not convert to a TickerStream structure %s",
				b.Name,
				err.Error())
		}
		b.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: b.Name,
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
		}
		return nil
	case "kline_1m", "kline_3m", "kline_5m", "kline_15m", "kline_30m", "kline_1h", "kline_2h", "kline_4h",
		"kline_6h", "kline_8h", "kline_12h", "kline_1d", "kline_3d", "kline_1w", "kline_1M":
		var kline KlineStream
		err = json.Unmarshal(jsonData, &kline)
		if err != nil {
			return fmt.Errorf("%v - Could not convert to a KlineStream structure %s",
				b.Name,
				err)
		}
		b.Websocket.DataHandler <- stream.KlineData{
			Timestamp:  kline.EventTime.Time(),
			Pair:       pair,
			AssetType:  asset.Spot,
			Exchange:   b.Name,
			StartTime:  kline.Kline.StartTime.Time(),
			CloseTime:  kline.Kline.CloseTime.Time(),
			Interval:   kline.Kline.Interval,
			OpenPrice:  kline.Kline.OpenPrice.Float64(),
			ClosePrice: kline.Kline.ClosePrice.Float64(),
			HighPrice:  kline.Kline.HighPrice.Float64(),
			LowPrice:   kline.Kline.LowPrice.Float64(),
			Volume:     kline.Kline.Volume.Float64(),
		}
		return nil
	case "depth":
		var depth WebsocketDepthStream
		err = json.Unmarshal(jsonData, &depth)
		if err != nil {
			return fmt.Errorf("%v - Could not convert to depthStream structure %s",
				b.Name,
				err)
		}
		var init bool
		init, err = b.UpdateLocalBuffer(&depth)
		if err != nil {
			if init {
				return nil
			}
			return fmt.Errorf("%v - UpdateLocalCache error: %s",
				b.Name,
				err)
		}
		return nil
	default:
		return fmt.Errorf("%s %s %s", b.Name, stream.UnhandledMessage, string(respRaw))
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
func (b *Binance) SeedLocalCache(ctx context.Context, p currency.Pair) error {
	ob, err := b.GetOrderBook(ctx,
		OrderBookDataRequestParams{
			Symbol: p,
			Limit:  1000,
		})
	if err != nil {
		return err
	}
	return b.SeedLocalCacheWithBook(p, ob)
}

// SeedLocalCacheWithBook seeds the local orderbook cache
func (b *Binance) SeedLocalCacheWithBook(p currency.Pair, orderbookNew *OrderBook) error {
	newOrderBook := orderbook.Base{
		Pair:            p,
		Asset:           asset.Spot,
		Exchange:        b.Name,
		LastUpdateID:    orderbookNew.LastUpdateID,
		VerifyOrderbook: b.CanVerifyOrderbook,
		Bids:            make(orderbook.Tranches, len(orderbookNew.Bids)),
		Asks:            make(orderbook.Tranches, len(orderbookNew.Asks)),
		LastUpdated:     time.Now(), // Time not provided in REST book.
	}
	for i := range orderbookNew.Bids {
		newOrderBook.Bids[i] = orderbook.Tranche{
			Amount: orderbookNew.Bids[i].Quantity,
			Price:  orderbookNew.Bids[i].Price,
		}
	}
	for i := range orderbookNew.Asks {
		newOrderBook.Asks[i] = orderbook.Tranche{
			Amount: orderbookNew.Asks[i].Quantity,
			Price:  orderbookNew.Asks[i].Price,
		}
	}
	return b.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// UpdateLocalBuffer updates and returns the most recent iteration of the orderbook
func (b *Binance) UpdateLocalBuffer(wsdp *WebsocketDepthStream) (bool, error) {
	pair, err := b.MatchSymbolWithAvailablePairs(wsdp.Pair, asset.Spot, false)
	if err != nil {
		return false, err
	}
	err = b.obm.stageWsUpdate(wsdp, pair, asset.Spot)
	if err != nil {
		init, err2 := b.obm.checkIsInitialSync(pair)
		if err2 != nil {
			return false, err2
		}
		return init, err
	}

	err = b.applyBufferUpdate(pair)
	if err != nil {
		b.flushAndCleanup(pair)
	}

	return false, err
}

// generateSubscriptions generates the default subscription set
func (b *Binance) generateSubscriptions() (subscription.List, error) {
	var channels = make([]string, 0, len(b.Features.Subscriptions))
	for i := range b.Features.Subscriptions {
		name, err := channelName(b.Features.Subscriptions[i])
		if err != nil {
			return nil, err
		}
		channels = append(channels, name)
	}
	var subscriptions subscription.List
	pairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for y := range pairs {
		for z := range channels {
			lp := pairs[y].Lower()
			lp.Delimiter = ""
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: lp.String() + "@" + channels[z],
				Pairs:   currency.Pairs{pairs[y]},
				Asset:   asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// channelName converts a Subscription Config into binance format channel suffix
func channelName(s *subscription.Subscription) (string, error) {
	name, ok := subscriptionNames[s.Channel]
	if !ok {
		return name, fmt.Errorf("%w: %s", stream.ErrSubscriptionNotSupported, s.Channel)
	}

	switch s.Channel {
	case subscription.OrderbookChannel:
		if s.Levels != 0 {
			name += "@" + strconv.Itoa(s.Levels)
		}
		if s.Interval.Duration() == time.Second {
			name += "@1000ms"
		} else {
			name += "@" + s.Interval.Short()
		}
	case subscription.CandlesChannel:
		name += "_" + s.Interval.Short()
	}
	return name, nil
}

// Subscribe subscribes to a set of channels
func (b *Binance) Subscribe(channels subscription.List) error {
	return b.ParallelChanOp(channels, b.subscribeToChan, 50)
}

// subscribeToChan handles a single subscription and parses the result
// on success it adds the subscription to the websocket
func (b *Binance) subscribeToChan(chans subscription.List) error {
	id := b.Websocket.Conn.GenerateMessageID(false)

	cNames := make([]string, len(chans))
	for i := range chans {
		c := chans[i]
		cNames[i] = c.Channel
		if err := b.Websocket.AddSubscriptions(c); err != nil {
			return fmt.Errorf("%w Channel: %s Pair: %s Error: %w", stream.ErrSubscriptionFailure, c.Channel, c.Pairs, err)
		}
	}

	req := &struct {
		WsPayload
		Params []string `json:"params"`
	}{
		WsPayload: WsPayload{
			Method: wsUnsubscribeMethod,
			ID:     id,
		},
		Params: cNames,
	}

	respRaw, err := b.Websocket.Conn.SendMessageReturnResponse(id, req)
	if err == nil {
		if v, d, _, rErr := jsonparser.Get(respRaw, "result"); rErr != nil {
			err = rErr
		} else if d != jsonparser.Null { // null is the only expected and acceptable response
			err = fmt.Errorf("%w: %s", errUnknownError, v)
		}
	}

	if err != nil {
		if err2 := b.Websocket.RemoveSubscriptions(chans...); err2 != nil {
			err = common.AppendError(err, err2)
		}
		err = fmt.Errorf("%w: %w; Channels: %s", stream.ErrSubscriptionFailure, err, strings.Join(cNames, ", "))
		b.Websocket.DataHandler <- err
	} else {
		for _, s := range chans {
			if sErr := s.SetState(subscription.SubscribedState); sErr != nil {
				err = common.AppendError(err, sErr)
			}
		}
	}

	return err
}

// Unsubscribe unsubscribes from a set of channels
func (b *Binance) Unsubscribe(channels subscription.List) error {
	return b.ParallelChanOp(channels, b.unsubscribeFromChan, 50)
}

// unsubscribeFromChan sends a websocket message to stop receiving data from a channel
func (b *Binance) unsubscribeFromChan(chans subscription.List) error {
	id := b.Websocket.Conn.GenerateMessageID(false)

	cNames := make([]string, len(chans))
	for i := range chans {
		cNames[i] = chans[i].Channel
	}

	req := &struct {
		WsPayload
		Params []string `json:"params"`
	}{
		WsPayload: WsPayload{
			Method: wsUnsubscribeMethod,
			ID:     id,
		},
		Params: cNames,
	}

	respRaw, err := b.Websocket.Conn.SendMessageReturnResponse(id, req)
	if err == nil {
		if v, d, _, rErr := jsonparser.Get(respRaw, "result"); rErr != nil {
			err = rErr
		} else if d != jsonparser.Null { // null is the only expected and acceptable response
			err = fmt.Errorf("%w: %s", errUnknownError, v)
		}
	}

	if err != nil {
		err = fmt.Errorf("%w: %w; Channels: %s", stream.ErrUnsubscribeFailure, err, strings.Join(cNames, ", "))
		b.Websocket.DataHandler <- err
	} else {
		err = b.Websocket.RemoveSubscriptions(chans...)
	}

	return err
}

// ProcessUpdate processes the websocket orderbook update
func (b *Binance) ProcessUpdate(cp currency.Pair, a asset.Item, ws *WebsocketDepthStream) error {
	updateBid := make([]orderbook.Tranche, len(ws.UpdateBids))
	for i := range ws.UpdateBids {
		updateBid[i] = orderbook.Tranche{
			Price:  ws.UpdateBids[i][0].Float64(),
			Amount: ws.UpdateBids[i][1].Float64(),
		}
	}
	updateAsk := make([]orderbook.Tranche, len(ws.UpdateAsks))
	for i := range ws.UpdateAsks {
		updateAsk[i] = orderbook.Tranche{
			Price:  ws.UpdateAsks[i][0].Float64(),
			Amount: ws.UpdateAsks[i][1].Float64(),
		}
	}
	return b.Websocket.Orderbook.Update(&orderbook.Update{
		Bids:       updateBid,
		Asks:       updateAsk,
		Pair:       cp,
		UpdateID:   ws.LastUpdateID,
		UpdateTime: ws.Timestamp.Time(),
		Asset:      a,
	})
}

// applyBufferUpdate applies the buffer to the orderbook or initiates a new
// orderbook sync by the REST protocol which is off handed to go routine.
func (b *Binance) applyBufferUpdate(pair currency.Pair) error {
	fetching, needsFetching, err := b.obm.handleFetchingBook(pair)
	if err != nil {
		return err
	}
	if fetching {
		return nil
	}
	if needsFetching {
		if b.Verbose {
			log.Debugf(log.WebsocketMgr, "%s Orderbook: Fetching via REST\n", b.Name)
		}
		return b.obm.fetchBookViaREST(pair)
	}

	recent, err := b.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	if err != nil {
		log.Errorf(
			log.WebsocketMgr,
			"%s error fetching recent orderbook when applying updates: %s\n",
			b.Name,
			err)
	}

	if recent != nil {
		err = b.obm.checkAndProcessUpdate(b.ProcessUpdate, pair, recent)
		if err != nil {
			log.Errorf(
				log.WebsocketMgr,
				"%s error processing update - initiating new orderbook sync via REST: %s\n",
				b.Name,
				err)
			err = b.obm.setNeedsFetchingBook(pair)
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
func (b *Binance) SynchroniseWebsocketOrderbook() {
	b.Websocket.Wg.Add(1)
	go func() {
		defer b.Websocket.Wg.Done()
		for {
			select {
			case <-b.Websocket.ShutdownC:
				for {
					select {
					case <-b.obm.jobs:
					default:
						return
					}
				}
			case j := <-b.obm.jobs:
				err := b.processJob(j.Pair)
				if err != nil {
					log.Errorf(log.WebsocketMgr,
						"%s processing websocket orderbook error %v",
						b.Name, err)
				}
			}
		}
	}()
}

// processJob fetches and processes orderbook updates
func (b *Binance) processJob(p currency.Pair) error {
	err := b.SeedLocalCache(context.TODO(), p)
	if err != nil {
		return fmt.Errorf("%s %s seeding local cache for orderbook error: %v",
			p, asset.Spot, err)
	}

	err = b.obm.stopFetchingBook(p)
	if err != nil {
		return err
	}

	// Immediately apply the buffer updates so we don't wait for a
	// new update to initiate this.
	err = b.applyBufferUpdate(p)
	if err != nil {
		b.flushAndCleanup(p)
		return err
	}
	return nil
}

// flushAndCleanup flushes orderbook and clean local cache
func (b *Binance) flushAndCleanup(p currency.Pair) {
	errClean := b.Websocket.Orderbook.FlushOrderbook(p, asset.Spot)
	if errClean != nil {
		log.Errorf(log.WebsocketMgr,
			"%s flushing websocket error: %v",
			b.Name,
			errClean)
	}
	errClean = b.obm.cleanup(p)
	if errClean != nil {
		log.Errorf(log.WebsocketMgr, "%s cleanup websocket error: %v",
			b.Name,
			errClean)
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

func (o *orderbookManager) checkAndProcessUpdate(processor func(currency.Pair, asset.Item, *WebsocketDepthStream) error, pair currency.Pair, recent *orderbook.Base) error {
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
func (u *update) validate(updt *WebsocketDepthStream, recent *orderbook.Base) (bool, error) {
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
