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
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
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

	// maxBatchedPayloads defines an upper restriction on outbound batched
	// subscriptions. There seems to be a byte limit that is not documented.
	// Max bytes == 4096. 150 is arbitrary.
	maxBatchedPayloads = 150
)

var listenKey string

// WsConnect initiates a websocket connection
func (b *Binance) WsConnect(conn stream.Connection) error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	var err error

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

	b.preConnectionSetup()
	return nil
}

// WsConnectAuth authenticates connection
func (b *Binance) WsConnectAuth(conn stream.Connection) error {
	fmt.Println("AUTH CONNECTION_____________________________________________")
	var dialer websocket.Dialer
	var err error
	listenKey, err = b.GetWsAuthStreamKey()
	if err != nil {
		// TODO: Use this functionality in streaming package.
		b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		// --
		return fmt.Errorf("%v unable to connect to authenticated Websocket. Error: %s",
			b.Name,
			err)
	}

	// TODO: Check -- cleans on failed connection
	clean := strings.Split(b.Websocket.GetWebsocketURL(), "?streams=")
	authPayload := clean[0] + "?streams=" + listenKey
	err = b.Websocket.SetWebsocketURL(authPayload, false, false)
	if err != nil {
		return err
	}

	err = conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s",
			b.Name,
			err)
	}

	b.Websocket.Wg.Add(1)
	go b.KeepAuthKeyAlive()

	conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PongMessage,
		Delay:             pingDelay,
	})

	return nil
}

func (b *Binance) spawnConnection(setup stream.ConnectionSetup) (stream.Connection, error) {
	if setup.URL == "" {
		return nil, errors.New("url not specified when generating connection")
	}
	return &stream.WebsocketConnection{
		Verbose:          b.Verbose,
		ExchangeName:     b.Name,
		URL:              setup.URL,
		ProxyURL:         b.Websocket.GetProxyAddress(),
		Authenticated:    setup.DedicatedAuthenticatedConn,
		Match:            b.Websocket.Match,
		Wg:               b.Websocket.Wg,
		Traffic:          b.Websocket.TrafficAlert,
		RateLimit:        250,
		ResponseMaxLimit: time.Second * 10,
		Conf:             &setup,
	}, nil
}

func (b *Binance) preConnectionSetup() {
	if b.obm == nil {
		b.obm = &orderbookManager{
			buffer:       make(map[currency.Code]map[currency.Code]map[asset.Item]chan *WebsocketDepthStream),
			fetchingBook: make(map[currency.Code]map[currency.Code]map[asset.Item]*bool),
			initialSync:  make(map[currency.Code]map[currency.Code]map[asset.Item]*bool),
			jobs:         make(chan orderbookWsJob, 2000),
		}

		for i := 0; i < 10; i++ {
			// 10 workers for synchronising book
			b.SynchroniseWebsocketOrderbook()
		}
	}
}

// KeepAuthKeyAlive will continuously send messages to keep the WS auth key
// active
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

func (b *Binance) wsHandleData(respRaw []byte, conn stream.Connection) error {
	var multiStreamData map[string]interface{}
	err := json.Unmarshal(respRaw, &multiStreamData)
	if err != nil {
		return err
	}

	if err, ok := multiStreamData["error"]; ok {
		hello, ok := err.(map[string]interface{})
		if !ok {
			return errors.New("could not type cast to map[string]interface{}")
		}

		code, ok := hello["code"].(float64)
		if !ok {
			return errors.New("invalid data for error code")
		}

		message, ok := hello["msg"].(string)
		if !ok {
			return errors.New("invalid data for error message")
		}

		return fmt.Errorf("websocket error code [%d], %s",
			int(code),
			message)
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

				// TODO: Need to infer asset by connection
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

					assets, err := conn.GetAssetsBySubscriptionType(stream.Trade, pair)
					if err != nil {
						return err
					}

					for i := range assets {
						b.Websocket.DataHandler <- stream.TradeData{
							CurrencyPair: pair,
							Timestamp:    time.Unix(0, trade.TimeStamp*int64(time.Millisecond)),
							Price:        price,
							Amount:       amount,
							Exchange:     b.Name,
							AssetType:    assets[i],
						}
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

					assets, err := conn.GetAssetsBySubscriptionType(stream.Ticker, pair)
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

					assets, err := conn.GetAssetsBySubscriptionType(stream.Kline, pair)
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

					err = b.UpdateLocalBuffer(&depth, conn)
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
func (b *Binance) SeedLocalCache(p currency.Pair, a asset.Items) error {
	fPair, err := b.FormatExchangeCurrency(p, a[0])
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
	for i := range a {
		err = b.SeedLocalCacheWithBook(fPair, &ob, a[i])
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
	newOrderBook.AssetType = a
	newOrderBook.ExchangeName = b.Name
	newOrderBook.LastUpdateID = orderbookNew.LastUpdateID
	return b.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// UpdateLocalBuffer stages update to a related asset type associated with a
// connection.
func (b *Binance) UpdateLocalBuffer(u *WebsocketDepthStream, conn stream.Connection) error {
	// TODO: Infer the asset type from the connection
	pairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	pair, err := currency.NewPairFromFormattedPairs(u.Pair, pairs, currency.PairFormat{Uppercase: true})
	if err != nil {
		return err
	}

	assets, err := conn.GetAssetsBySubscriptionType(stream.Orderbook, pair)
	if err != nil {
		return err
	}

	var errs common.Errors
	for i := range assets {
		err = b.obm.stageWsUpdate(u, pair, assets[i])
		if err != nil {
			return err
		}

		err := b.applyBufferUpdate(pair, assets[i], conn)
		if err != nil {
			errs = append(errs, err)
			err = b.Websocket.Orderbook.FlushOrderbook(pair, assets[i])
			if err != nil {
				log.Errorln(log.WebsocketMgr, "flushing websocket error:", err)
			}

			err = b.obm.cleanup(pair, assets[i])
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if errs != nil {
		return errs
	}
	return nil
}

// applyBufferUpdate applies the buffer to the orderbook or initiates a new
// orderbook sync by the REST protocol which is off handed to go routine.
func (b *Binance) applyBufferUpdate(pair currency.Pair, a asset.Item, conn stream.Connection) error {
	var fetching bool
	fetching, err := b.obm.checkIsFetchingBook(pair, a)
	if err != nil {
		return err
	}

	if fetching {
		return nil
	}

	recent := b.Websocket.Orderbook.GetOrderbook(pair, a)
	if recent == nil {
		return b.obm.fetchBookViaREST(pair, a, conn)
	}

	return b.obm.checkAndProcessUpdate(b.ProcessUpdate, pair, a, *recent)
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
func (b *Binance) SynchroniseWebsocketOrderbook() {
	b.Websocket.Wg.Add(1)
	go func() {
		defer b.Websocket.Wg.Done()
		for {
			select {
			case job := <-b.obm.jobs:
				assets, err := job.Conn.GetAssetsBySubscriptionType(stream.Orderbook, job.Pair)
				if err != nil {
					log.Errorln(log.WebsocketMgr, "cannot fetch asssociated asset types", err)
					continue
				}

				err = b.SeedLocalCache(job.Pair, assets)
				if err != nil {
					log.Errorln(log.WebsocketMgr, "seeding local cache for orderbook error", err)
					continue
				}

				// Immediatly apply the buffer updates so we don't wait for a
				// new update to initiate this.
				for i := range assets {
					err = b.obm.stopFetchingBook(job.Pair, assets[i])
					if err != nil {
						log.Errorln(log.WebsocketMgr, "applying orderbook updates error", err)
						continue
					}
					err = b.applyBufferUpdate(job.Pair, assets[i], job.Conn)
					if err != nil {
						log.Errorln(log.WebsocketMgr, "applying orderbook updates error", err)
						err = b.Websocket.Orderbook.FlushOrderbook(job.Pair, assets[i])
						if err != nil {
							log.Errorln(log.WebsocketMgr, "flushing websocket error:", err)
						}
						err = b.obm.cleanup(job.Pair, assets[i])
						if err != nil {
							log.Errorln(log.WebsocketMgr, "cleanup websocket error:", err)
						}
						continue
					}
				}

			case <-b.Websocket.ShutdownC:
				return
			}
		}
	}()
}

// GenerateSubscriptions generates the default subscription set
func (b *Binance) GenerateSubscriptions(options stream.SubscriptionOptions) ([]stream.ChannelSubscription, error) {
	var channels []WsChannel
	enabled, err := options.Features.Websocket.Functionality()
	if err != nil {
		return nil, err
	}
	if enabled.TickerFetching {
		channels = append(channels, WsChannel{
			Definition: "@ticker",
			Type:       stream.Ticker,
		})
	}

	if enabled.TradeFetching {
		channels = append(channels, WsChannel{
			Definition: "@trade",
			Type:       stream.Trade,
		})
	}

	if enabled.KlineFetching {
		var intervals []kline.Interval
		intervals, err = options.Features.GetEnabledKlineIntervals()
		if err != nil {
			return nil, err
		}

		for i := range intervals {
			var fmtInterval string
			fmtInterval, err = formatKlineInterval(intervals[i])
			if err != nil {
				return nil, err
			}
			channels = append(channels, WsChannel{
				Definition: "@kline_" + fmtInterval,
				Type:       stream.Kline,
			})
		}
	}

	if enabled.OrderbookFetching {
		channels = append(channels, WsChannel{
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

func formatKlineInterval(k kline.Interval) (string, error) {
	switch k {
	case kline.OneMin:
		return "1m", nil
	case kline.ThreeMin:
		return "3m", nil
	case kline.FiveMin:
		return "5m", nil
	case kline.FifteenMin:
		return "15m", nil
	case kline.ThirtyMin:
		return "30m", nil
	case kline.OneHour:
		return "1h", nil
	case kline.TwoHour:
		return "2h", nil
	case kline.FourHour:
		return "4h", nil
	case kline.SixHour:
		return "6h", nil
	case kline.TwelveHour:
		return "12h", nil
	case kline.OneDay:
		return "1d", nil
	case kline.ThreeDay:
		return "3d", nil
	case kline.OneWeek:
		return "1w", nil
	case kline.OneMonth:
		return "1M", nil
	default:
		return "", fmt.Errorf("kline interval %s unsupported by exchange", k)
	}
}

// Subscribe subscribes to a set of channels
func (b *Binance) Subscribe(sub stream.SubscriptionParameters) error {
	payload := WsPayload{
		Method: "SUBSCRIBE",
	}

	var subbed []stream.ChannelSubscription
	for i := range sub.Items {
		payload.Params = append(payload.Params, sub.Items[i].Channel)
		subbed = append(subbed, sub.Items[i])
		if (i+1)%maxBatchedPayloads != 0 {
			continue
		}

		err := sub.Conn.SendJSONMessage(payload)
		if err != nil {
			return err
		}

		err = sub.Conn.AddSuccessfulSubscriptions(subbed)
		if err != nil {
			return err
		}
		payload.Params = nil
		subbed = nil
	}

	if payload.Params != nil {
		err := sub.Conn.SendJSONMessage(payload)
		if err != nil {
			return err
		}

		err = sub.Conn.AddSuccessfulSubscriptions(subbed)
		if err != nil {
			return err
		}
	}
	return nil
}

// Unsubscribe unsubscribes from a set of channels
func (b *Binance) Unsubscribe(unsub stream.SubscriptionParameters) error {
	payload := WsPayload{
		Method: "UNSUBSCRIBE",
	}

	var unsubbed []stream.ChannelSubscription
	for i := range unsub.Items {
		payload.Params = append(payload.Params, unsub.Items[i].Channel)
		unsubbed = append(unsubbed, unsub.Items[i])
		if (i+1)%maxBatchedPayloads != 0 {
			continue
		}

		err := unsub.Conn.SendJSONMessage(payload)
		if err != nil {
			return err
		}

		err = unsub.Conn.RemoveSuccessfulUnsubscriptions(unsubbed)
		if err != nil {
			return err
		}
		payload.Params = nil
		unsubbed = nil
	}

	if payload.Params != nil {
		err := unsub.Conn.SendJSONMessage(payload)
		if err != nil {
			return err
		}

		err = unsub.Conn.RemoveSuccessfulUnsubscriptions(unsubbed)
		if err != nil {
			return err
		}
	}
	return nil
}

// stageWsUpdate stages websocket update to roll through updates that need to
// be applied to a fetched orderbook via REST.
func (o *orderbookManager) stageWsUpdate(u *WebsocketDepthStream, pair currency.Pair, a asset.Item) error {
	o.bmtx.Lock()
	defer o.bmtx.Unlock()

	_, ok := o.buffer[pair.Base]
	if !ok {
		o.buffer[pair.Base] = make(map[currency.Code]map[asset.Item]chan *WebsocketDepthStream)
	}

	_, ok = o.buffer[pair.Base][pair.Quote]
	if !ok {
		o.buffer[pair.Base][pair.Quote] = make(map[asset.Item]chan *WebsocketDepthStream)
	}

	ch, ok := o.buffer[pair.Base][pair.Quote][a]
	if !ok {
		ch = make(chan *WebsocketDepthStream, 100) // 100ms update assuming we
		// might have up to a 10 second delay. There could be a potential 100
		// updates for the currency.
		o.buffer[pair.Base][pair.Quote][a] = ch

		// Set init and fetching tables.
		_, ok := o.fetchingBook[pair.Base]
		if !ok {
			o.fetchingBook[pair.Base] = make(map[currency.Code]map[asset.Item]*bool)
		}
		_, ok = o.fetchingBook[pair.Base][pair.Quote]
		if !ok {
			o.fetchingBook[pair.Base][pair.Quote] = make(map[asset.Item]*bool)
		}
		o.fetchingBook[pair.Base][pair.Quote][a] = convert.BoolPtr(false)

		_, ok = o.initialSync[pair.Base]
		if !ok {
			o.initialSync[pair.Base] = make(map[currency.Code]map[asset.Item]*bool)
		}
		_, ok = o.initialSync[pair.Base][pair.Quote]
		if !ok {
			o.initialSync[pair.Base][pair.Quote] = make(map[asset.Item]*bool)
		}
		o.initialSync[pair.Base][pair.Quote][a] = convert.BoolPtr(false)
	}

	select {
	// Put update in the channel to be processed
	case ch <- u:
		return nil
	default:
		return fmt.Errorf("channel blockage for %s, asset %s and connection",
			pair, a)
	}
}

// checkIsFetchingBook checks status if the book is currently being via the REST
// protocol.
func (o *orderbookManager) checkIsFetchingBook(pair currency.Pair, a asset.Item) (bool, error) {
	o.fmtx.Lock()
	defer o.fmtx.Unlock()
	fetching, ok := o.fetchingBook[pair.Base][pair.Quote][a]
	if !ok {
		return false,
			fmt.Errorf("check is fetching book cannot match currency pair %s asset type %s",
				pair,
				a)
	}
	return *fetching, nil
}

// checkInitialSync checks if book is ready for initial sync
func (o *orderbookManager) checkInitialSync(pair currency.Pair, a asset.Item) (bool, error) {
	o.fmtx.Lock()
	defer o.fmtx.Unlock()
	isInit, ok := o.initialSync[pair.Base][pair.Quote][a]
	if !ok {
		return false,
			fmt.Errorf("check initial sync cannot match currency pair %s asset type %s",
				pair,
				a)
	}
	return *isInit, nil
}

// stopFetchingBook completes the book fetching.
func (o *orderbookManager) stopFetchingBook(pair currency.Pair, a asset.Item) error {
	o.fmtx.Lock()
	defer o.fmtx.Unlock()
	ptr, ok := o.fetchingBook[pair.Base][pair.Quote][a]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			a)
	}
	if !*ptr {
		return fmt.Errorf("fetching book already set to false for %s %s",
			pair,
			a)
	}
	*ptr = false
	return nil

}

// completeInitialSync sets if an asset type has completed its initial sync
func (o *orderbookManager) completeInitialSync(pair currency.Pair, a asset.Item) error {
	o.fmtx.Lock()
	defer o.fmtx.Unlock()
	ptr, ok := o.initialSync[pair.Base][pair.Quote][a]
	if !ok {
		return fmt.Errorf("complete initial sync cannot match currency pair %s asset type %s",
			pair,
			a)
	}
	if !*ptr {
		return fmt.Errorf("initital sync already set to false for %s %s",
			pair,
			a)
	}
	*ptr = false
	return nil
}

// fetchBookViaREST pushes a job of fetching the orderbook via the REST protocol
// to get an initial full book that we can apply our buffered updates too.
func (o *orderbookManager) fetchBookViaREST(pair currency.Pair, a asset.Item, conn stream.Connection) error {
	o.fmtx.Lock()
	defer o.fmtx.Unlock()

	i, ok := o.initialSync[pair.Base][pair.Quote][a]
	if !ok {
		return fmt.Errorf("fetch book via rest cannot match currency pair %s asset type %s",
			pair,
			a)
	}
	*i = true

	f, ok := o.fetchingBook[pair.Base][pair.Quote][a]
	if !ok {
		return fmt.Errorf("fetch book via rest cannot match currency pair %s asset type %s",
			pair,
			a)
	}
	*f = true

	select {
	case o.jobs <- orderbookWsJob{pair, conn}:
		return nil
	default:
		return errors.New("book synchronisation channel blocked up")
	}
}

func (o *orderbookManager) checkAndProcessUpdate(processor func(currency.Pair, asset.Item, *WebsocketDepthStream) error, pair currency.Pair, a asset.Item, recent orderbook.Base) error {
	o.bmtx.Lock()
	defer o.bmtx.Unlock()

	ch, ok := o.buffer[pair.Base][pair.Quote][a]
	if !ok {
		return fmt.Errorf("could not match pair [%s] asset type [%s] in hash table to process websocket orderbook update",
			pair, a)
	}

	// This will continuously remove updates from the buffered channel and
	// apply them to the current orderbook.
loop:
	for {
		select {
		case d := <-ch:
			if d.LastUpdateID <= recent.LastUpdateID {
				// Drop any event where u is <= lastUpdateId in the snapshot.
				continue
			}

			initSync, err := o.checkInitialSync(pair, a)
			if err != nil {
				return err
			}
			id := recent.LastUpdateID + 1
			if initSync {
				// The first processed event should have U <= lastUpdateId+1 AND
				// u >= lastUpdateId+1.
				if d.FirstUpdateID > id && d.LastUpdateID < id {
					return fmt.Errorf("initial websocket orderbook sync failure for pair %s and asset %s",
						pair,
						a)
				}
				err := o.completeInitialSync(pair, a)
				if err != nil {
					return err
				}
			} else {
				// While listening to the stream, each new event's U should be
				// equal to the previous event's u+1.
				if d.FirstUpdateID != id {
					return fmt.Errorf("websocket orderbook synchronisation failure for pair %s and aset %s",
						pair,
						a)
				}
			}
			err = processor(pair, a, d)
			if err != nil {
				return err
			}
		default:
			break loop
		}
	}
	return nil
}

// cleanup cleans up buffer and reset fetch and init
func (o *orderbookManager) cleanup(pair currency.Pair, a asset.Item) error {
	o.bmtx.Lock()
	defer o.bmtx.Unlock()

	ch, ok := o.buffer[pair.Base][pair.Quote][a]
	if !ok {
		return fmt.Errorf("cleanup cannot match %s %s to hash table", pair, a)
	}

bufferEmpty:
	for {
		select {
		case _ = <-ch:
			// bleed and disgard buffer
		default:
			break bufferEmpty
		}
	}

	// reset underlying bools
	_ = o.stopFetchingBook(pair, a)
	_ = o.completeInitialSync(pair, a)
	return nil
}
