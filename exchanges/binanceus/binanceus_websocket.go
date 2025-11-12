package binanceus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
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
	binanceusDefaultWebsocketURL = "wss://stream.binance.us:9443/stream"
	binanceusAPIURL              = "https://api.binance.us"
	pingDelay                    = time.Minute * 9
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
		e.Websocket.Wg.Add(1)
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

// KeepAuthKeyAlive will continuously send messages to
// keep the WS auth key active
func (e *Exchange) KeepAuthKeyAlive(ctx context.Context) {
	defer e.Websocket.Wg.Done()
	// CloseUserDataStream closes the User data stream and remove the listen key when closing the websocket
	defer func() {
		if err := e.CloseUserDataStream(ctx); err != nil {
			log.Errorf(log.WebsocketMgr, "%s closing user data stream error %v", e.Name, err)
		}
	}()

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

func (e *Exchange) wsHandleData(ctx context.Context, respRaw []byte) error {
	var multiStreamData map[string]any
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
		if strings.EqualFold(method, "subscribe") {
			return nil
		}
		if strings.EqualFold(method, "unsubscribe") {
			return nil
		}
	}
	if newData, ok := multiStreamData["data"].(map[string]any); ok {
		if event, ok := newData["e"].(string); ok {
			switch event {
			case "outboundAccountPosition":
				var data wsAccountPosition
				err = json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to outboundAccountPosition structure %s",
						e.Name,
						err)
				}
				return e.Websocket.DataHandler.Send(ctx, data)
			case "balanceUpdate":
				var data wsBalanceUpdate
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to balanceUpdate structure %s",
						e.Name,
						err)
				}
				return e.Websocket.DataHandler.Send(ctx, data)
			case "executionReport":
				var data wsOrderUpdate
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to executionReport structure %s",
						e.Name,
						err)
				}
				averagePrice := 0.0
				if data.Data.CumulativeFilledQuantity != 0 {
					averagePrice = data.Data.CumulativeQuoteTransactedQuantity / data.Data.CumulativeFilledQuantity
				}
				remainingAmount := data.Data.Quantity - data.Data.CumulativeFilledQuantity
				pair, assetType, err := e.GetRequestFormattedPairAndAssetType(data.Data.Symbol)
				if err != nil {
					return err
				}
				var feeAsset currency.Code
				if data.Data.CommissionAsset != "" {
					feeAsset = currency.NewCode(data.Data.CommissionAsset)
				}
				orderID := strconv.FormatInt(data.Data.OrderID, 10)
				orderStatus, err := stringToOrderStatus(data.Data.OrderStatus)
				if err != nil {
					return err
				}
				clientOrderID := data.Data.ClientOrderID
				if orderStatus == order.Cancelled {
					clientOrderID = data.Data.CancelledClientOrderID
				}
				orderType, err := order.StringToOrderType(data.Data.OrderType)
				if err != nil {
					return err
				}
				orderSide, err := order.StringToOrderSide(data.Data.Side)
				if err != nil {
					return err
				}
				return e.Websocket.DataHandler.Send(ctx, &order.Detail{
					Price:                data.Data.Price,
					Amount:               data.Data.Quantity,
					AverageExecutedPrice: averagePrice,
					ExecutedAmount:       data.Data.CumulativeFilledQuantity,
					RemainingAmount:      remainingAmount,
					Cost:                 data.Data.CumulativeQuoteTransactedQuantity,
					CostAsset:            pair.Quote,
					Fee:                  data.Data.Commission,
					FeeAsset:             feeAsset,
					Exchange:             e.Name,
					OrderID:              orderID,
					ClientOrderID:        clientOrderID,
					Type:                 orderType,
					Side:                 orderSide,
					Status:               orderStatus,
					AssetType:            assetType,
					Date:                 data.Data.OrderCreationTime.Time(),
					LastUpdated:          data.Data.TransactionTime.Time(),
					Pair:                 pair,
				})
			case "listStatus":
				var data WsListStatus
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to listStatus structure %s",
						e.Name,
						err)
				}
				return e.Websocket.DataHandler.Send(ctx, data)
			}
		}
	}
	// Market Data Streams
	if wsStream, ok := multiStreamData["stream"].(string); ok {
		var pairs currency.Pairs
		streamType := strings.Split(wsStream, "@")
		if len(streamType) > 1 {
			if data, ok := multiStreamData["data"]; ok {
				rawData, err := json.Marshal(data)
				if err != nil {
					return err
				}

				pairs, err = e.GetEnabledPairs(asset.Spot)
				if err != nil {
					return err
				}

				format, err := e.GetPairFormat(asset.Spot, true)
				if err != nil {
					return err
				}

				switch streamType[1] {
				case "trade":
					saveTradeData := e.IsSaveTradeDataEnabled()
					if !saveTradeData && !e.IsTradeFeedEnabled() {
						return nil
					}

					var t TradeStream
					err = json.Unmarshal(rawData, &t)
					if err != nil {
						return fmt.Errorf("%v - Could not unmarshal trade data: %s",
							e.Name,
							err)
					}

					pair, err := currency.NewPairFromFormattedPairs(t.Symbol, pairs, format)
					if err != nil {
						return err
					}

					return e.Websocket.Trade.Update(saveTradeData,
						trade.Data{
							CurrencyPair: pair,
							Timestamp:    t.TimeStamp.Time(),
							Price:        t.Price.Float64(),
							Amount:       t.Quantity.Float64(),
							Exchange:     e.Name,
							AssetType:    asset.Spot,
							TID:          strconv.FormatInt(t.TradeID, 10),
						})
				case "ticker":
					var t TickerStream
					err := json.Unmarshal(rawData, &t)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to a TickerStream structure %s",
							e.Name,
							err.Error())
					}

					pair, err := currency.NewPairFromFormattedPairs(t.Symbol, pairs, format)
					if err != nil {
						return err
					}

					return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
						ExchangeName: e.Name,
						Open:         t.OpenPrice,
						Close:        t.ClosePrice,
						Volume:       t.TotalTradedVolume,
						QuoteVolume:  t.TotalTradedQuoteVolume,
						High:         t.HighPrice,
						Low:          t.LowPrice,
						Bid:          t.BestBidPrice,
						Ask:          t.BestAskPrice,
						Last:         t.LastPrice,
						LastUpdated:  t.EventTime.Time(),
						AssetType:    asset.Spot,
						Pair:         pair,
					})
				case "kline_1m", "kline_3m", "kline_5m", "kline_15m", "kline_30m", "kline_1h", "kline_2h", "kline_4h",
					"kline_6h", "kline_8h", "kline_12h", "kline_1d", "kline_3d", "kline_1w", "kline_1M":
					var kline KlineStream
					err := json.Unmarshal(rawData, &kline)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to a KlineStream structure %s",
							e.Name,
							err)
					}

					pair, err := currency.NewPairFromFormattedPairs(kline.Symbol, pairs, format)
					if err != nil {
						return err
					}

					return e.Websocket.DataHandler.Send(ctx, websocket.KlineData{
						Timestamp:  kline.EventTime.Time(),
						Pair:       pair,
						AssetType:  asset.Spot,
						Exchange:   e.Name,
						StartTime:  kline.Kline.StartTime.Time(),
						CloseTime:  kline.Kline.CloseTime.Time(),
						Interval:   kline.Kline.Interval,
						OpenPrice:  kline.Kline.OpenPrice,
						ClosePrice: kline.Kline.ClosePrice,
						HighPrice:  kline.Kline.HighPrice,
						LowPrice:   kline.Kline.LowPrice,
						Volume:     kline.Kline.Volume,
					})
				case "depth":
					var depth WebsocketDepthStream
					err := json.Unmarshal(rawData, &depth)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to depthStream structure %s",
							e.Name,
							err)
					}
					init, err := e.UpdateLocalBuffer(&depth)
					if err != nil {
						if init {
							return nil
						}
						return fmt.Errorf("%v - UpdateLocalCache error: %s",
							e.Name,
							err)
					}
					return nil
				case "depth5", "depth10", "depth20":
					var depth WebsocketDepthDiffStream
					err := json.Unmarshal(rawData, &depth)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to depthStream structure %s",
							e.Name,
							err)
					}
					return e.Websocket.DataHandler.Send(ctx, &depth)
				case "bookTicker":
					var bo OrderBookTickerStream
					err := json.Unmarshal(rawData, &bo)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to bookOrder structure %s ", err, e.Name)
					}
					pair, err := currency.NewPairFromFormattedPairs(bo.S, pairs, format)
					if err != nil {
						return err
					}
					bo.Symbol = pair
					return e.Websocket.DataHandler.Send(ctx, &bo)
				case "aggTrade":
					var agg WebsocketAggregateTradeStream
					err := json.Unmarshal(rawData, &agg)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to aggTrade structure %s ", err, e.Name)
					}
					return e.Websocket.DataHandler.Send(ctx, &agg)
				default:
					return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
						Message: e.Name + websocket.UnhandledMessage + string(respRaw),
					})
				}
			}
		} else if wsStream == "!bookTicker" {
			var bt OrderBookTickerStream
			if data, ok := multiStreamData["data"]; ok {
				rawData, err := json.Marshal(data)
				if err != nil {
					return err
				}
				pairs, err := e.GetEnabledPairs(asset.Spot)
				if err != nil {
					return err
				}

				format, err := e.GetPairFormat(asset.Spot, true)
				if err != nil {
					return err
				}
				err = json.Unmarshal(rawData, &bt)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to bookOrder structure %s ", err, e.Name)
				}
				pair, err := currency.NewPairFromFormattedPairs(bt.S, pairs, format)
				if err != nil {
					return err
				}
				bt.Symbol = pair
				return e.Websocket.DataHandler.Send(ctx, &bt)
			}
		}
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

// UpdateLocalBuffer updates and returns the most recent iteration of the orderbook
func (e *Exchange) UpdateLocalBuffer(wsdp *WebsocketDepthStream) (bool, error) {
	enabledPairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return false, err
	}

	format, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return false, err
	}

	currencyPair, err := currency.NewPairFromFormattedPairs(wsdp.Pair,
		enabledPairs,
		format)
	if err != nil {
		return false, err
	}

	err = e.obm.stageWsUpdate(wsdp, currencyPair, asset.Spot)
	if err != nil {
		init, err2 := e.obm.checkIsInitialSync(currencyPair)
		if err2 != nil {
			return false, err2
		}
		return init, err
	}

	err = e.applyBufferUpdate(currencyPair)
	if err != nil {
		e.invalidateAndCleanupOrderbook(currencyPair)
	}

	return false, err
}

// GenerateSubscriptions generates the default subscription set
func (e *Exchange) GenerateSubscriptions() (subscription.List, error) {
	channels := []string{"@ticker", "@trade", "@kline_1m", "@depth@100ms"}
	var subscriptions subscription.List

	pairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}

subs:
	for y := range pairs {
		for z := range channels {
			lp := pairs[y].Lower()
			lp.Delimiter = ""
			if len(subscriptions) >= 1023 {
				log.Warnf(log.WebsocketMgr, "BinanceUS has 1024 subscription limit, only subscribing within limit. Requested %v", len(pairs)*len(channels))
				break subs
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: lp.String() + channels[z],
				Pairs:   currency.Pairs{pairs[y]},
				Asset:   asset.Spot,
			})
		}
	}

	return subscriptions, nil
}

// Subscribe subscribes to a set of channels
func (e *Exchange) Subscribe(channelsToSubscribe subscription.List) error {
	ctx := context.TODO()
	payload := WebsocketPayload{
		Method: "SUBSCRIBE",
	}
	for i := range channelsToSubscribe {
		payload.Params = append(payload.Params, channelsToSubscribe[i].Channel)
		if i%50 == 0 && i != 0 {
			err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, payload)
			if err != nil {
				return err
			}
			payload.Params = []any{}
		}
	}
	if len(payload.Params) > 0 {
		err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, payload)
		if err != nil {
			return err
		}
	}
	return e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, channelsToSubscribe...)
}

// Unsubscribe unsubscribes from a set of channels
func (e *Exchange) Unsubscribe(channelsToUnsubscribe subscription.List) error {
	ctx := context.TODO()
	payload := WebsocketPayload{
		Method: "UNSUBSCRIBE",
	}
	for i := range channelsToUnsubscribe {
		payload.Params = append(payload.Params, channelsToUnsubscribe[i].Channel)
		if i%50 == 0 && i != 0 {
			err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, payload)
			if err != nil {
				return err
			}
			payload.Params = []any{}
		}
	}
	if len(payload.Params) > 0 {
		err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, payload)
		if err != nil {
			return err
		}
	}
	return e.Websocket.RemoveSubscriptions(e.Websocket.Conn, channelsToUnsubscribe...)
}

func (e *Exchange) setupOrderbookManager(ctx context.Context) {
	if e.obm == nil {
		e.obm = &orderbookManager{
			state: make(map[currency.Code]map[currency.Code]map[asset.Item]*update),
			jobs:  make(chan job, maxWSOrderbookJobs),
		}
	} else {
		// Change state on reconnect for initial sync.
		for x := range e.obm.state {
			for _, m2 := range e.obm.state[x] {
				for y := range m2 {
					m2[y].initialSync = true
					m2[y].needsFetchingBook = true
					m2[y].lastUpdateID = 0
				}
			}
		}
	}
	for range maxWSOrderbookWorkers {
		// 10 workers for synchronising book
		e.SynchroniseWebsocketOrderbook(ctx)
	}
}

// SynchroniseWebsocketOrderbook synchronises full orderbook for currency pair asset
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

// ProcessOrderbookUpdate processes the websocket orderbook update
func (e *Exchange) ProcessOrderbookUpdate(cp currency.Pair, a asset.Item, wsDSUpdate *WebsocketDepthStream) error {
	return e.Websocket.Orderbook.Update(&orderbook.Update{
		Bids:       wsDSUpdate.UpdateBids.Levels(),
		Asks:       wsDSUpdate.UpdateAsks.Levels(),
		Pair:       cp,
		UpdateID:   wsDSUpdate.LastUpdateID,
		UpdateTime: wsDSUpdate.Timestamp.Time(),
		Asset:      a,
	})
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

// SeedLocalCache seeds depth data
func (e *Exchange) SeedLocalCache(ctx context.Context, p currency.Pair) error {
	ob, err := e.GetOrderBookDepth(ctx, p, 1000)
	if err != nil {
		return err
	}
	return e.SeedLocalCacheWithBook(p, ob)
}

// SeedLocalCacheWithBook seeds the local orderbook cache
func (e *Exchange) SeedLocalCacheWithBook(p currency.Pair, orderbookNew *OrderBook) error {
	return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Pair:              p,
		Asset:             asset.Spot,
		Exchange:          e.Name,
		LastUpdateID:      orderbookNew.LastUpdateID,
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              orderbookNew.Bids,
		Asks:              orderbookNew.Asks,
		LastUpdated:       time.Now(), // Time not provided in REST book.
	})
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

// invalidateAndCleanupOrderbook invalidaates orderbook and cleans local cache
func (e *Exchange) invalidateAndCleanupOrderbook(p currency.Pair) {
	if err := e.Websocket.Orderbook.InvalidateOrderbook(p, asset.Spot); err != nil {
		log.Errorf(log.WebsocketMgr, "%s invalidate orderbook websocket error: %v", e.Name, err)
	}
	if err := e.obm.cleanup(p); err != nil {
		log.Errorf(log.WebsocketMgr, "%s cleanup websocket error: %v", e.Name, err)
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
		if updt.FirstUpdateID > id || updt.LastUpdateID < id {
			return false, fmt.Errorf("initial websocket orderbook sync failure for pair %s and asset %s",
				recent.Pair,
				asset.Spot)
		}
		u.initialSync = false
	}
	return true, nil
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
