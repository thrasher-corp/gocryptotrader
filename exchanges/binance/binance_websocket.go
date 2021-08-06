package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443/stream"
	pingDelay                  = time.Minute * 9
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
func (b *Binance) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	dialer.HandshakeTimeout = b.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	var err error
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err = b.GetWsAuthStreamKey()
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

	go b.wsReadData()
	b.setupOrderbookManager()
	return nil
}

func (b *Binance) setupOrderbookManager() {
	if b.obm == nil {
		b.obm = &orderbookManager{
			state: make(map[currency.Code]map[currency.Code]map[asset.Item]*update),
			jobs:  make(chan job, maxWSOrderbookJobs),
		}

		for i := 0; i < maxWSOrderbookWorkers; i++ {
			// 10 workers for synchronising book
			b.SynchroniseWebsocketOrderbook()
		}
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
			err := b.MaintainWsAuthStreamKey()
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
	b.Websocket.Wg.Add(1)
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
	var multiStreamData map[string]interface{}
	err := json.Unmarshal(respRaw, &multiStreamData)
	if err != nil {
		return err
	}

	if r, ok := multiStreamData["result"]; ok {
		if r == nil {
			return nil
		}
	}

	if method, ok := multiStreamData["method"].(string); ok {
		// TODO handle subscription handling
		if strings.EqualFold(method, "subscribe") {
			return nil
		}
		if strings.EqualFold(method, "unsubscribe") {
			return nil
		}
	}
	if newdata, ok := multiStreamData["data"].(map[string]interface{}); ok {
		if e, ok := newdata["e"].(string); ok {
			switch e {
			case "outboundAccountInfo":
				var data wsAccountInfo
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to outboundAccountInfo structure %s",
						b.Name,
						err)
				}
				b.Websocket.DataHandler <- data
				return nil
			case "outboundAccountPosition":
				var data wsAccountPosition
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to outboundAccountPosition structure %s",
						b.Name,
						err)
				}
				b.Websocket.DataHandler <- data
				return nil
			case "balanceUpdate":
				var data wsBalanceUpdate
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to balanceUpdate structure %s",
						b.Name,
						err)
				}
				b.Websocket.DataHandler <- data
				return nil
			case "executionReport":
				var data wsOrderUpdate
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to executionReport structure %s",
						b.Name,
						err)
				}
				var orderID = strconv.FormatInt(data.Data.OrderID, 10)
				oType, err := order.StringToOrderType(data.Data.OrderType)
				if err != nil {
					b.Websocket.DataHandler <- order.ClassificationError{
						Exchange: b.Name,
						OrderID:  orderID,
						Err:      err,
					}
				}
				var oSide order.Side
				oSide, err = order.StringToOrderSide(data.Data.Side)
				if err != nil {
					b.Websocket.DataHandler <- order.ClassificationError{
						Exchange: b.Name,
						OrderID:  orderID,
						Err:      err,
					}
				}
				var oStatus order.Status
				oStatus, err = stringToOrderStatus(data.Data.CurrentExecutionType)
				if err != nil {
					b.Websocket.DataHandler <- order.ClassificationError{
						Exchange: b.Name,
						OrderID:  orderID,
						Err:      err,
					}
				}
				var p currency.Pair
				var a asset.Item
				p, a, err = b.GetRequestFormattedPairAndAssetType(data.Data.Symbol)
				if err != nil {
					return err
				}
				clientOrderID := data.Data.ClientOrderID
				if oStatus == order.Cancelled {
					clientOrderID = data.Data.CancelledClientOrderID
				}
				b.Websocket.DataHandler <- &order.Detail{
					Price:           data.Data.Price,
					Amount:          data.Data.Quantity,
					ExecutedAmount:  data.Data.CumulativeFilledQuantity,
					RemainingAmount: data.Data.Quantity - data.Data.CumulativeFilledQuantity,
					Exchange:        b.Name,
					ID:              orderID,
					Type:            oType,
					Side:            oSide,
					Status:          oStatus,
					AssetType:       a,
					Date:            data.Data.OrderCreationTime,
					Pair:            p,
					ClientOrderID:   clientOrderID,
				}
				return nil
			case "listStatus":
				var data wsListStatus
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to listStatus structure %s",
						b.Name,
						err)
				}
				b.Websocket.DataHandler <- data
				return nil
			}
		}
	}
	if wsStream, ok := multiStreamData["stream"].(string); ok {
		streamType := strings.Split(wsStream, "@")
		if len(streamType) > 1 {
			if data, ok := multiStreamData["data"]; ok {
				rawData, err := json.Marshal(data)
				if err != nil {
					return err
				}

				pairs, err := b.GetEnabledPairs(asset.Spot)
				if err != nil {
					return err
				}

				format, err := b.GetPairFormat(asset.Spot, true)
				if err != nil {
					return err
				}

				switch streamType[1] {
				case "trade":
					if !b.IsSaveTradeDataEnabled() {
						return nil
					}
					var t TradeStream
					err := json.Unmarshal(rawData, &t)
					if err != nil {
						return fmt.Errorf("%v - Could not unmarshal trade data: %s",
							b.Name,
							err)
					}

					price, err := strconv.ParseFloat(t.Price, 64)
					if err != nil {
						return fmt.Errorf("%v - price conversion error: %s",
							b.Name,
							err)
					}

					amount, err := strconv.ParseFloat(t.Quantity, 64)
					if err != nil {
						return fmt.Errorf("%v - amount conversion error: %s",
							b.Name,
							err)
					}

					pair, err := currency.NewPairFromFormattedPairs(t.Symbol, pairs, format)
					if err != nil {
						return err
					}

					return b.AddTradesToBuffer(trade.Data{
						CurrencyPair: pair,
						Timestamp:    t.TimeStamp,
						Price:        price,
						Amount:       amount,
						Exchange:     b.Name,
						AssetType:    asset.Spot,
						TID:          strconv.FormatInt(t.TradeID, 10),
					})
				case "ticker":
					var t TickerStream
					err := json.Unmarshal(rawData, &t)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to a TickerStream structure %s",
							b.Name,
							err.Error())
					}

					pair, err := currency.NewPairFromFormattedPairs(t.Symbol, pairs, format)
					if err != nil {
						return err
					}

					b.Websocket.DataHandler <- &ticker.Price{
						ExchangeName: b.Name,
						Open:         t.OpenPrice,
						Close:        t.ClosePrice,
						Volume:       t.TotalTradedVolume,
						QuoteVolume:  t.TotalTradedQuoteVolume,
						High:         t.HighPrice,
						Low:          t.LowPrice,
						Bid:          t.BestBidPrice,
						Ask:          t.BestAskPrice,
						Last:         t.LastPrice,
						LastUpdated:  t.EventTime,
						AssetType:    asset.Spot,
						Pair:         pair,
					}
					return nil
				case "kline_1m", "kline_3m", "kline_5m", "kline_15m", "kline_30m", "kline_1h", "kline_2h", "kline_4h",
					"kline_6h", "kline_8h", "kline_12h", "kline_1d", "kline_3d", "kline_1w", "kline_1M":
					var kline KlineStream
					err := json.Unmarshal(rawData, &kline)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to a KlineStream structure %s",
							b.Name,
							err)
					}

					pair, err := currency.NewPairFromFormattedPairs(kline.Symbol, pairs, format)
					if err != nil {
						return err
					}

					b.Websocket.DataHandler <- stream.KlineData{
						Timestamp:  kline.EventTime,
						Pair:       pair,
						AssetType:  asset.Spot,
						Exchange:   b.Name,
						StartTime:  kline.Kline.StartTime,
						CloseTime:  kline.Kline.CloseTime,
						Interval:   kline.Kline.Interval,
						OpenPrice:  kline.Kline.OpenPrice,
						ClosePrice: kline.Kline.ClosePrice,
						HighPrice:  kline.Kline.HighPrice,
						LowPrice:   kline.Kline.LowPrice,
						Volume:     kline.Kline.Volume,
					}
					return nil
				case "depth":
					var depth WebsocketDepthStream
					err := json.Unmarshal(rawData, &depth)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to depthStream structure %s",
							b.Name,
							err)
					}
					init, err := b.UpdateLocalBuffer(&depth)
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
					b.Websocket.DataHandler <- stream.UnhandledMessageWarning{
						Message: b.Name + stream.UnhandledMessage + string(respRaw),
					}
				}
			}
		}
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "NEW":
		return order.New, nil
	case "CANCELED":
		return order.Cancelled, nil
	case "REJECTED":
		return order.Rejected, nil
	case "TRADE":
		return order.PartiallyFilled, nil
	case "EXPIRED":
		return order.Expired, nil
	default:
		return order.UnknownStatus, errors.New(status + " not recognised as order status")
	}
}

// SeedLocalCache seeds depth data
func (b *Binance) SeedLocalCache(p currency.Pair) error {
	ob, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: p,
		Limit:  1000,
	})
	if err != nil {
		return err
	}
	return b.SeedLocalCacheWithBook(p, &ob)
}

// SeedLocalCacheWithBook seeds the local orderbook cache
func (b *Binance) SeedLocalCacheWithBook(p currency.Pair, orderbookNew *OrderBook) error {
	var newOrderBook orderbook.Base
	for i := range orderbookNew.Bids {
		newOrderBook.Bids = append(newOrderBook.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[i].Quantity,
			Price:  orderbookNew.Bids[i].Price,
		})
	}
	for i := range orderbookNew.Asks {
		newOrderBook.Asks = append(newOrderBook.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[i].Quantity,
			Price:  orderbookNew.Asks[i].Price,
		})
	}

	newOrderBook.Pair = p
	newOrderBook.Asset = asset.Spot
	newOrderBook.Exchange = b.Name
	newOrderBook.LastUpdateID = orderbookNew.LastUpdateID
	newOrderBook.VerifyOrderbook = b.CanVerifyOrderbook

	return b.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// UpdateLocalBuffer updates and returns the most recent iteration of the orderbook
func (b *Binance) UpdateLocalBuffer(wsdp *WebsocketDepthStream) (bool, error) {
	enabledPairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return false, err
	}

	format, err := b.GetPairFormat(asset.Spot, true)
	if err != nil {
		return false, err
	}

	currencyPair, err := currency.NewPairFromFormattedPairs(wsdp.Pair,
		enabledPairs,
		format)
	if err != nil {
		return false, err
	}

	err = b.obm.stageWsUpdate(wsdp, currencyPair, asset.Spot)
	if err != nil {
		init, err2 := b.obm.checkIsInitialSync(currencyPair)
		if err2 != nil {
			return false, err2
		}
		return init, err
	}

	err = b.applyBufferUpdate(currencyPair)
	if err != nil {
		b.flushAndCleanup(currencyPair)
	}

	return false, err
}

// GenerateSubscriptions generates the default subscription set
func (b *Binance) GenerateSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{"@ticker", "@trade", "@kline_1m", "@depth@100ms"}
	var subscriptions []stream.ChannelSubscription
	assets := b.GetAssetTypes(true)
	for x := range assets {
		if assets[x] == asset.Spot {
			pairs, err := b.GetEnabledPairs(assets[x])
			if err != nil {
				return nil, err
			}

			for y := range pairs {
				for z := range channels {
					lp := pairs[y].Lower()
					lp.Delimiter = ""
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel:  lp.String() + channels[z],
						Currency: pairs[y],
						Asset:    assets[x],
					})
				}
			}
		}
	}
	return subscriptions, nil
}

// Subscribe subscribes to a set of channels
func (b *Binance) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	payload := WsPayload{
		Method: "SUBSCRIBE",
	}
	for i := range channelsToSubscribe {
		payload.Params = append(payload.Params, channelsToSubscribe[i].Channel)
		if i%50 == 0 && i != 0 {
			err := b.Websocket.Conn.SendJSONMessage(payload)
			if err != nil {
				return err
			}
			payload.Params = []string{}
		}
	}
	if len(payload.Params) > 0 {
		err := b.Websocket.Conn.SendJSONMessage(payload)
		if err != nil {
			return err
		}
	}
	b.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}

// Unsubscribe unsubscribes from a set of channels
func (b *Binance) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	payload := WsPayload{
		Method: "UNSUBSCRIBE",
	}
	for i := range channelsToUnsubscribe {
		payload.Params = append(payload.Params, channelsToUnsubscribe[i].Channel)
		if i%50 == 0 && i != 0 {
			err := b.Websocket.Conn.SendJSONMessage(payload)
			if err != nil {
				return err
			}
			payload.Params = []string{}
		}
	}
	if len(payload.Params) > 0 {
		err := b.Websocket.Conn.SendJSONMessage(payload)
		if err != nil {
			return err
		}
	}
	b.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe...)
	return nil
}

// ProcessUpdate processes the websocket orderbook update
func (b *Binance) ProcessUpdate(cp currency.Pair, a asset.Item, ws *WebsocketDepthStream) error {
	var updateBid []orderbook.Item
	for i := range ws.UpdateBids {
		price, ok := ws.UpdateBids[i][0].(string)
		if !ok {
			return errors.New("type assertion failed for bid price")
		}
		p, err := strconv.ParseFloat(price, 64)
		if err != nil {
			return err
		}
		amount, ok := ws.UpdateBids[i][1].(string)
		if !ok {
			return errors.New("type assertion failed for bid amount")
		}
		a, err := strconv.ParseFloat(amount, 64)
		if err != nil {
			return err
		}
		updateBid = append(updateBid, orderbook.Item{Price: p, Amount: a})
	}

	var updateAsk []orderbook.Item
	for i := range ws.UpdateAsks {
		price, ok := ws.UpdateAsks[i][0].(string)
		if !ok {
			return errors.New("type assertion failed for ask price")
		}
		p, err := strconv.ParseFloat(price, 64)
		if err != nil {
			return err
		}
		amount, ok := ws.UpdateAsks[i][1].(string)
		if !ok {
			return errors.New("type assertion failed for ask amount")
		}
		a, err := strconv.ParseFloat(amount, 64)
		if err != nil {
			return err
		}
		updateAsk = append(updateAsk, orderbook.Item{Price: p, Amount: a})
	}

	return b.Websocket.Orderbook.Update(&buffer.Update{
		Bids:       updateBid,
		Asks:       updateAsk,
		Pair:       cp,
		UpdateID:   ws.LastUpdateID,
		UpdateTime: ws.Timestamp,
		Asset:      a,
	})
}

// applyBufferUpdate applies the buffer to the orderbook or initiates a new
// orderbook sync by the REST protocol which is off handed to go routine.
func (b *Binance) applyBufferUpdate(pair currency.Pair) error {
	fetching, err := b.obm.checkIsFetchingBook(pair)
	if err != nil {
		return err
	}
	if fetching {
		return nil
	}

	recent, err := b.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	if err != nil || (recent.Asks == nil && recent.Bids == nil) {
		return b.obm.fetchBookViaREST(pair)
	}

	return b.obm.checkAndProcessUpdate(b.ProcessUpdate, pair, recent)
}

// SynchroniseWebsocketOrderbook synchronises full orderbook for currency pair
// asset
func (b *Binance) SynchroniseWebsocketOrderbook() {
	b.Websocket.Wg.Add(1)
	go func() {
		defer b.Websocket.Wg.Done()
		for {
			select {
			case j := <-b.obm.jobs:
				err := b.processJob(j.Pair)
				if err != nil {
					log.Errorf(log.WebsocketMgr,
						"%s processing websocket orderbook error %v",
						b.Name, err)
				}
			case <-b.Websocket.ShutdownC:
				return
			}
		}
	}()
}

// processJob fetches and processes orderbook updates
func (b *Binance) processJob(p currency.Pair) error {
	err := b.SeedLocalCache(p)
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
			buffer:       make(chan *WebsocketDepthStream, maxWSUpdateBuffer),
			fetchingBook: false,
			initialSync:  true,
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

// checkIsFetchingBook checks status if the book is currently being via the REST
// protocol.
func (o *orderbookManager) checkIsFetchingBook(pair currency.Pair) (bool, error) {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return false,
			fmt.Errorf("check is fetching book cannot match currency pair %s asset type %s",
				pair,
				asset.Spot)
	}
	return state.fetchingBook, nil
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
		return fmt.Errorf("initital sync already set to false for %s %s",
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
	return nil
}
