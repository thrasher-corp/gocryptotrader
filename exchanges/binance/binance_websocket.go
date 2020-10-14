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
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443/stream"
	pingDelay                  = time.Minute * 9
)

var listenKey string

// Job defines a syncro job
type Job struct {
	Pair  currency.Pair
	Asset asset.Item
}

// WsConnect initiates a websocket connection
func (b *Binance) WsConnect(conn stream.Connection) error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	var err error
	if conn.IsAuthenticated() {
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

			fmt.Println("Authconn:", conn)

			err = conn.Dial(&dialer, http.Header{})
			if err != nil {
				return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s",
					b.Name,
					err)
			}

			if b.Websocket.CanUseAuthenticatedEndpoints() {
				b.Websocket.Wg.Add(1)
				go b.KeepAuthKeyAlive()
			}

			conn.SetupPingHandler(stream.PingHandler{
				UseGorillaHandler: true,
				MessageType:       websocket.PongMessage,
				Delay:             pingDelay,
			})

			b.Websocket.Wg.Add(1)
			go b.wsReadData(conn)

			return nil
		}
	}

	fmt.Println("Unauth Conn:", conn)

	err = conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s",
			b.Name,
			err)
	}

	conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PongMessage,
		Delay:             pingDelay,
	})

	b.preConnectionSetup(conn)

	b.Websocket.Wg.Add(1)
	go b.wsReadData(conn)

	// subs, err := b.GenerateSubscriptions(stream.SubscriptionOptions{})
	// if err != nil {
	// 	return err
	// }
	// return b.Websocket.SubscribeToChannels(subs)
	return nil
}

func (b *Binance) spawnConnection(setup stream.ConnectionSetup) (stream.Connection, error) {
	return &stream.WebsocketConnection{
		Verbose:       b.Verbose,
		ExchangeName:  b.Name,
		URL:           setup.URL,
		ProxyURL:      b.Websocket.GetProxyAddress(),
		Authenticated: setup.DedicatedAuthenticatedConn,
		Match:         b.Websocket.Match,
		Wg:            b.Websocket.Wg,
		Traffic:       b.Websocket.TrafficAlert,
	}, nil
}

func (b *Binance) preConnectionSetup(conn stream.Connection) {
	b.buffer = make(map[string]map[asset.Item]chan *WebsocketDepthStream)
	b.fetchingbook = make(map[string]map[asset.Item]bool)
	b.initialSync = make(map[string]map[asset.Item]bool)
	b.jobs = make(chan Job, 100)
	for i := 0; i < 10; i++ {
		// 10 workers for synchronising book
		b.SynchroniseWebsocketOrderbook(conn)
	}
}

// KeepAuthKeyAlive will continuously send messages to
// keep the WS auth key active
func (b *Binance) KeepAuthKeyAlive() {
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
func (b *Binance) wsReadData(conn stream.Connection) {
	defer b.Websocket.Wg.Done()

	for {
		// fmt.Println("WSREADDATA")
		resp := conn.ReadMessage()
		if resp.Raw == nil {
			// fmt.Println("nil")
			return
		}
		err := b.wsHandleData(resp.Raw, conn)
		if err != nil {
			b.Websocket.DataHandler <- err
		}
		// fmt.Println("RETURN PROCESS")
	}
}

func (b *Binance) wsHandleData(respRaw []byte, conn stream.Connection) error {
	// fmt.Println("Data Come through", string(respRaw))
	var multiStreamData map[string]interface{}
	err := json.Unmarshal(respRaw, &multiStreamData)
	if err != nil {
		return err
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
			case "outboundAccountPosition":
				var data wsAccountPosition
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to outboundAccountPosition structure %s",
						b.Name,
						err)
				}
				b.Websocket.DataHandler <- data
			case "balanceUpdate":
				var data wsBalanceUpdate
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to balanceUpdate structure %s",
						b.Name,
						err)
				}
				b.Websocket.DataHandler <- data
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
					Date:            time.Unix(0, data.Data.OrderCreationTime*int64(time.Millisecond)),
					Pair:            p,
				}
			case "listStatus":
				var data wsListStatus
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to listStatus structure %s",
						b.Name,
						err)
				}
				b.Websocket.DataHandler <- data
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
					var trade TradeStream
					err := json.Unmarshal(rawData, &trade)
					if err != nil {
						return fmt.Errorf("%v - Could not unmarshal trade data: %s",
							b.Name,
							err)
					}

					price, err := strconv.ParseFloat(trade.Price, 64)
					if err != nil {
						return fmt.Errorf("%v - price conversion error: %s",
							b.Name,
							err)
					}

					amount, err := strconv.ParseFloat(trade.Quantity, 64)
					if err != nil {
						return fmt.Errorf("%v - amount conversion error: %s",
							b.Name,
							err)
					}

					pair, err := currency.NewPairFromFormattedPairs(trade.Symbol, pairs, format)
					if err != nil {
						return err
					}

					b.Websocket.DataHandler <- stream.TradeData{
						CurrencyPair: pair,
						Timestamp:    time.Unix(0, trade.TimeStamp*int64(time.Millisecond)),
						Price:        price,
						Amount:       amount,
						Exchange:     b.Name,
						AssetType:    asset.Spot,
					}
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

					assets, err := b.Websocket.Connections.GetAssetsBySubscriptionType(conn, stream.Ticker, pair)
					if err != nil {
						return err
					}

					for i := range assets {
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
							LastUpdated:  time.Unix(0, t.EventTime*int64(time.Millisecond)),
							AssetType:    assets[i],
							Pair:         pair,
						}
					}
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

					assets, err := b.Websocket.Connections.GetAssetsBySubscriptionType(conn, stream.Kline, pair)
					if err != nil {
						return err
					}

					for i := range assets {
						b.Websocket.DataHandler <- stream.KlineData{
							Timestamp:  time.Unix(0, kline.EventTime*int64(time.Millisecond)),
							Pair:       pair,
							AssetType:  assets[i],
							Exchange:   b.Name,
							StartTime:  time.Unix(0, kline.Kline.StartTime*int64(time.Millisecond)),
							CloseTime:  time.Unix(0, kline.Kline.CloseTime*int64(time.Millisecond)),
							Interval:   kline.Kline.Interval,
							OpenPrice:  kline.Kline.OpenPrice,
							ClosePrice: kline.Kline.ClosePrice,
							HighPrice:  kline.Kline.HighPrice,
							LowPrice:   kline.Kline.LowPrice,
							Volume:     kline.Kline.Volume,
						}
					}
				case "depth":
					var depth WebsocketDepthStream
					err := json.Unmarshal(rawData, &depth)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to depthStream structure %s",
							b.Name,
							err)
					}

					fmt.Println("PAIR:", depth.Pair)

					pair, err := currency.NewPairFromString(depth.Pair)
					if err != nil {
						return err
					}

					assets, err := b.Websocket.Connections.GetAssetsBySubscriptionType(conn, stream.Orderbook, pair)
					if err != nil {
						return err
					}

					fmt.Println("ASSETS:", assets)

					for i := range assets {
						err = b.UpdateLocalBuffer(assets[i], &depth)
						if err != nil {
							return fmt.Errorf("%v - UpdateLocalCache error: %s",
								b.Name,
								err)
						}
					}
				default:
					b.Websocket.DataHandler <- stream.UnhandledMessageWarning{
						Message: b.Name + stream.UnhandledMessage + string(respRaw),
					}
				}
			}
		}
	}
	return nil
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "NEW":
		return order.New, nil
	case "CANCELLED":
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
func (b *Binance) SeedLocalCache(p currency.Pair, a asset.Items) error {
	// fPair, err := b.FormatExchangeCurrency(p, a)
	// if err != nil {
	// 	return err
	// }

	ob, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: p.String(),
		Limit:  1000,
	})
	if err != nil {
		return err
	}

	for i := range a {
		err = b.SeedLocalCacheWithBook(p, &ob, a[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// SeedLocalCacheWithBook seeds the local orderbook cache
func (b *Binance) SeedLocalCacheWithBook(p currency.Pair, orderbookNew *OrderBook, a asset.Item) error {
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
	newOrderBook.AssetType = asset.Spot
	newOrderBook.ExchangeName = b.Name
	newOrderBook.LastUpdateID = orderbookNew.LastUpdateID

	return b.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// UpdateLocalBuffer updates and returns the most recent iteration of the orderbook
func (b *Binance) UpdateLocalBuffer(a asset.Item, u *WebsocketDepthStream) error {
	enabledPairs, err := b.GetEnabledPairs(a)
	if err != nil {
		return err
	}

	format, err := b.GetPairFormat(a, true)
	if err != nil {
		return err
	}

	currencyPair, err := currency.NewPairFromFormattedPairs(u.Pair,
		enabledPairs,
		format)
	if err != nil {
		return err
	}

	return b.CheckAndLoadBook(currencyPair, a, u)
}

// CheckAndLoadBook buffers updates, spawns workers for http book request and
// lets slip routine, validates buffer contents and returns required work when
// needed
func (b *Binance) CheckAndLoadBook(c currency.Pair, a asset.Item, u *WebsocketDepthStream) error {
	b.mtx.Lock() // protects fetching book
	defer b.mtx.Unlock()

	ch, ok := b.buffer[c.String()][a]
	if !ok {
		ch = make(chan *WebsocketDepthStream, 1000)
		b.buffer[c.String()] = make(map[asset.Item]chan *WebsocketDepthStream)
		b.buffer[c.String()][a] = ch
	}
	select {
	case ch <- u:
	default:
		return fmt.Errorf("channel blockage %s", c)
	}

	if b.fetchingbook[c.String()][a] {
		return nil
	}

	err := b.applyBufferUpdate(c, a)
	if err != nil {
		flushError := b.Websocket.Orderbook.FlushOrderbook(c, a)
		if flushError != nil {
			log.Errorln(log.WebsocketMgr, "flushing websocket error:", flushError)
		}
		return err
	}

	return nil
}

// BleedPipe applies the buffer to the orderbook
func (b *Binance) applyBufferUpdate(cp currency.Pair, a asset.Item) error {
	currentBook := b.Websocket.Orderbook.GetOrderbook(cp, a)
	if currentBook == nil {
		_, ok := b.initialSync[cp.String()]
		if !ok {
			b.initialSync[cp.String()] = make(map[asset.Item]bool)
		}

		b.initialSync[cp.String()][a] = true

		_, ok = b.fetchingbook[cp.String()]
		if !ok {
			b.fetchingbook[cp.String()] = make(map[asset.Item]bool)
		}
		b.fetchingbook[cp.String()][a] = true

		select {
		case b.jobs <- Job{cp, a}:
		default:
			return errors.New("book synchronisation channel blocked up")
		}
		return nil
	}

loop:
	for {
		select {
		case d := <-b.buffer[cp.String()][a]:
			if d.LastUpdateID <= currentBook.LastUpdateID {
				// Drop any event where u is <= lastUpdateId in the snapshot.
				continue
			}
			id := currentBook.LastUpdateID + 1
			if b.initialSync[cp.String()][a] {
				// The first processed event should have U <= lastUpdateId+1 AND
				// u >= lastUpdateId+1.
				if d.FirstUpdateID > id && d.LastUpdateID < id {
					return errors.New("initial sync failure")
				}
				b.initialSync[cp.String()][a] = false
			} else {
				// While listening to the stream, each new event's U should be
				// equal to the previous event's u+1.
				if d.FirstUpdateID != id {
					return errors.New("synchronisation failure")
				}
			}
			err := b.ProcessUpdate(cp, a, d)
			if err != nil {
				return err
			}
		default:
			break loop
		}
	}
	return nil
}

// ProcessUpdate processes the websocket orderbook update
func (b *Binance) ProcessUpdate(cp currency.Pair, a asset.Item, ws *WebsocketDepthStream) error {
	var updateBid []orderbook.Item
	for i := range ws.UpdateBids {
		p, err := strconv.ParseFloat(ws.UpdateBids[i][0].(string), 64)
		if err != nil {
			return err
		}
		a, err := strconv.ParseFloat(ws.UpdateBids[i][1].(string), 64)
		if err != nil {
			return err
		}
		updateBid = append(updateBid, orderbook.Item{Price: p, Amount: a})
	}

	var updateAsk []orderbook.Item
	for i := range ws.UpdateAsks {
		p, err := strconv.ParseFloat(ws.UpdateAsks[i][0].(string), 64)
		if err != nil {
			return err
		}
		a, err := strconv.ParseFloat(ws.UpdateAsks[i][1].(string), 64)
		if err != nil {
			return err
		}
		updateAsk = append(updateAsk, orderbook.Item{Price: p, Amount: a})
	}

	fmt.Println("asset Item update:", a)

	return b.Websocket.Orderbook.Update(&buffer.Update{
		Bids:     updateBid,
		Asks:     updateAsk,
		Pair:     cp,
		UpdateID: ws.LastUpdateID,
		Asset:    a,
	})
}

// SynchroniseWebsocketOrderbook synchronises full orderbook for currency pair
// asset
func (b *Binance) SynchroniseWebsocketOrderbook(conn stream.Connection) {
	b.Websocket.Wg.Add(1)
	go func() {
		defer b.Websocket.Wg.Done()
		for {
			select {
			case job := <-b.jobs:
				assets, err := b.Websocket.Connections.GetAssetsBySubscriptionType(conn, stream.Orderbook, job.Pair)
				if err != nil {
					fmt.Println(b.Websocket.Connections.GetAllSubscriptions())
					log.Errorln(log.WebsocketMgr, "cannot fetch asssociated asset types", err)
					continue
				}

				err = b.SeedLocalCache(job.Pair, assets)
				if err != nil {
					log.Errorln(log.WebsocketMgr, "seeding local cache for orderbook error", err)
					continue
				}

				for i := range assets {
					b.mtx.Lock()
					err = b.applyBufferUpdate(job.Pair, assets[i])
					if err != nil {
						log.Errorln(log.WebsocketMgr, "applying orderbook updates error", err)
						err = b.Websocket.Orderbook.FlushOrderbook(job.Pair, assets[i])
						if err != nil {
							log.Errorln(log.WebsocketMgr, "flushing websocket error:", err)
						}
						continue
					}

					b.fetchingbook[job.Pair.String()][assets[i]] = false
					b.mtx.Unlock()
				}

			case <-b.Websocket.ShutdownC:
				return
			}
		}
	}()
}

// Channel adds an application of a subscription type
type Channel struct {
	Definition string
	Type       stream.Subscription
}

// GenerateSubscriptions generates the default subscription set
func (b *Binance) GenerateSubscriptions(options stream.SubscriptionOptions) ([]stream.ChannelSubscription, error) {
	var channels []Channel
	// if options.Features.TickerFetching {
	// 	channels = append(channels, Channel{
	// 		Definition: "@ticker",
	// 		Type:       stream.Ticker,
	// 	})
	// }

	// if options.Features.TradeFetching {
	// 	channels = append(channels, Channel{
	// 		Definition: "@trade",
	// 		Type:       stream.Trade,
	// 	})
	// }

	// if options.Features.KlineFetching {
	// 	channels = append(channels, Channel{
	// 		Definition: "@kline_1m",
	// 		Type:       stream.Kline,
	// 	})
	// }

	if options.Features.OrderbookFetching {
		channels = append(channels, Channel{
			Definition: "@depth@100ms",
			Type:       stream.Orderbook,
		})
	}

	var subscriptions []stream.ChannelSubscription
	assets := b.GetAssetTypes()
	for x := range assets {
		pairs, err := b.GetEnabledPairs(assets[x])
		if err != nil {
			return nil, err
		}

		for y := range pairs {
			for z := range channels {
				lp := pairs[y].Lower()
				lp.Delimiter = ""
				channel := lp.String() + channels[z].Definition
				subscriptions = append(subscriptions,
					stream.ChannelSubscription{
						Channel:          channel,
						Currency:         pairs[y],
						Asset:            assets[x],
						SubscriptionType: channels[z].Type,
					})
			}
		}
	}
	return subscriptions, nil
}

// Subscribe subscribes to a set of channels
func (b *Binance) Subscribe(sub stream.SubscriptionParamaters) error {
	fmt.Println("Subbing:", sub)
	payload := WsPayload{
		Method: "SUBSCRIBE",
	}
	for i := range sub.Items {
		payload.Params = append(payload.Params, sub.Items[i].Channel)
	}
	err := sub.Conn.SendJSONMessage(payload)
	if err != nil {
		return err
	}
	return sub.Manager.AddSuccessfulSubscriptions(sub.Conn, sub.Items)
}

// Unsubscribe unsubscribes from a set of channels
func (b *Binance) Unsubscribe(unsub stream.SubscriptionParamaters) error {
	payload := WsPayload{
		Method: "UNSUBSCRIBE",
	}
	for i := range unsub.Items {
		payload.Params = append(payload.Params, unsub.Items[i].Channel)
	}
	err := unsub.Conn.SendJSONMessage(payload)
	if err != nil {
		return err
	}
	return unsub.Manager.RemoveSuccessfulUnsubscriptions(unsub.Conn, unsub.Items)
}

// // NewProtocolSync sets up a new protocol syncrhonisation system for buffering,
// // syncing and deploying currency pairs
// func NewProtocolSync(jobCap int) (*ProtocolSync, error) {
// 	return &ProtocolSync{
// 		buffer: make(map[string]Buffer),
// 		job:    make(chan currency.Pair, jobCap),
// 	}, nil
// }

// // ProtocolSync provides a sychronisation method for different currency pair
// // assets
// type ProtocolSync struct {
// 	sync.Mutex
// 	buffer map[string]Buffer
// 	job    chan currency.Pair
// }

// // Buffer defines a buffered channel of potential updates and sync mechanism
// // to determine actionable through put
// type Buffer struct {
// 	b     chan *WebsocketDepthStream
// 	fetch int32
// }

// // GetEnabledPairsCombineAsset returns the full list with the combination of
// // assets as margin and spot are the same currency pair and use the same book
// func (b *Binance) GetEnabledPairsCombineAsset() (currency.Pairs, error) {
// 	var pairs currency.Pairs
// 	spot, err := b.GetEnabledPairs(asset.Spot)
// 	if err != nil {
// 		return nil, err
// 	}
// 	pairs = append(pairs, spot...)

// 	margin, err := b.GetEnabledPairs(asset.Margin)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for i := range margin {
// 		if !pairs.Contains(margin[i], true) {
// 			pairs = append(pairs, margin[i])
// 		}
// 	}
// 	return pairs, nil
// }
