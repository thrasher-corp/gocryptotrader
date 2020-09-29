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

// WsConnect initiates a websocket connection
func (b *Binance) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
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

	b.pipe = make(map[string]chan *WebsocketDepthStream)
	b.fetchingbook = make(map[string]bool)
	b.initialSync = make(map[string]bool)
	enabledPairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	b.syncMe = make(chan currency.Pair, len(enabledPairs))
	for i := 0; i < 10; i++ {
		// 10 workers for synchronising book
		b.BookSynchro()
	}

	go b.wsReadData()

	subs, err := b.GenerateSubscriptions()
	if err != nil {
		return err
	}
	return b.Websocket.SubscribeToChannels(subs)
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
						AssetType:    asset.Spot,
						Pair:         pair,
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

					b.Websocket.DataHandler <- stream.KlineData{
						Timestamp:  time.Unix(0, kline.EventTime*int64(time.Millisecond)),
						Pair:       pair,
						AssetType:  asset.Spot,
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
				case "depth":
					var depth WebsocketDepthStream
					err := json.Unmarshal(rawData, &depth)
					if err != nil {
						return fmt.Errorf("%v - Could not convert to depthStream structure %s",
							b.Name,
							err)
					}

					err = b.UpdateLocalBuffer(&depth)
					if err != nil {
						return fmt.Errorf("%v - UpdateLocalCache error: %s",
							b.Name,
							err)
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
func (b *Binance) SeedLocalCache(p currency.Pair) error {
	fPair, err := b.FormatExchangeCurrency(p, asset.Spot)
	if err != nil {
		return err
	}

	ob, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: fPair.String(),
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
	newOrderBook.AssetType = asset.Spot
	newOrderBook.ExchangeName = b.Name
	newOrderBook.LastUpdateID = orderbookNew.LastUpdateID

	return b.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// UpdateLocalBuffer updates and returns the most recent iteration of the orderbook
func (b *Binance) UpdateLocalBuffer(u *WebsocketDepthStream) error {
	enabledPairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := b.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	currencyPair, err := currency.NewPairFromFormattedPairs(u.Pair,
		enabledPairs,
		format)
	if err != nil {
		return err
	}

	return b.CheckAndLoadBook(currencyPair, asset.Spot, u)
}

// CheckAndLoadBook buffers updates, spawns workers for http book request and
// lets slip routine, validates buffer contents and returns required work when
// needed
func (b *Binance) CheckAndLoadBook(c currency.Pair, ass asset.Item, u *WebsocketDepthStream) error {
	b.mtx.Lock() // protects fetching book
	defer b.mtx.Unlock()

	ch, ok := b.pipe[c.String()]
	if !ok {
		ch = make(chan *WebsocketDepthStream, 1000)
		b.pipe[c.String()] = ch
	}
	select {
	case ch <- u:
	default:
		return fmt.Errorf("channel blockage %s", c)
	}

	if b.fetchingbook[c.String()] {
		return nil
	}

	return b.BleedPipe(c, ass)
}

// BleedPipe applies the buffer to the orderbook
func (b *Binance) BleedPipe(cp currency.Pair, a asset.Item) error {
	currentBook := b.Websocket.Orderbook.GetOrderbook(cp, a)
	if currentBook == nil {
		b.initialSync[cp.String()] = true
		b.fetchingbook[cp.String()] = true
		select {
		case b.syncMe <- cp:
		default:
			return errors.New("book synchronisation channel blocked up")
		}
		return nil
	}

loop:
	for {
		select {
		case d := <-b.pipe[cp.String()]:
			if d.LastUpdateID <= currentBook.LastUpdateID {
				// Drop any event where u is <= lastUpdateId in the snapshot.
				continue
			}
			id := currentBook.LastUpdateID + 1
			if b.initialSync[cp.String()] {
				// The first processed event should have U <= lastUpdateId+1 AND
				// u >= lastUpdateId+1.
				if d.FirstUpdateID > id && d.LastUpdateID < id {
					return errors.New("initial sync failure")
				}
				b.initialSync[cp.String()] = false
			} else {
				// While listening to the stream, each new event's U should be
				// equal to the previous event's u+1.
				if d.FirstUpdateID != id {
					return errors.New("synchronisation failure")
				}
			}
			err := b.ProcessUpdate(cp, d)
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
func (b *Binance) ProcessUpdate(cp currency.Pair, ws *WebsocketDepthStream) error {
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

	return b.Websocket.Orderbook.Update(&buffer.Update{
		Bids:     updateBid,
		Asks:     updateAsk,
		Pair:     cp,
		UpdateID: ws.LastUpdateID,
		Asset:    asset.Spot,
	})
}

// BookSynchro synchronises full orderbook for currency pair
func (b *Binance) BookSynchro() {
	b.Websocket.Wg.Add(1)
	go func() {
		defer b.Websocket.Wg.Done()
		for {
			select {
			case job := <-b.syncMe:
				err := b.SeedLocalCache(job)

				if err != nil {
					log.Errorln(log.Global, "seeding local cache for orderbook error", err)
					continue
				}

				b.mtx.Lock()
				err = b.BleedPipe(job, asset.Spot)
				if err != nil {
					log.Errorln(log.Global, "applying orderbook updates error", err)
					continue
				}

				b.fetchingbook[job.String()] = false
				b.mtx.Unlock()

			case <-b.Websocket.ShutdownC:
				return
			}
		}
	}()
}

// GenerateSubscriptions generates the default subscription set
func (b *Binance) GenerateSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{"@ticker", "@trade", "@kline_1m", "@depth@100ms"}
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
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  lp.String() + channels[z],
					Currency: pairs[y],
					Asset:    assets[x],
				})
			}
		}
	}
	if len(subscriptions) > 1024 {
		return nil,
			fmt.Errorf("subscriptions: %d exceed maximum single connection streams of 1024",
				len(subscriptions))
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
	}
	err := b.Websocket.Conn.SendJSONMessage(payload)
	if err != nil {
		return err
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
	}
	err := b.Websocket.Conn.SendJSONMessage(payload)
	if err != nil {
		return err
	}
	b.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe...)
	return nil
}
