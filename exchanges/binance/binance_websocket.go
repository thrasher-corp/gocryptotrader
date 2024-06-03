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
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	errUnknownError = errors.New("unknown error")
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

	b.OrderbookBuilder, err = exchange.NewOrderbookBuilder(b, b.GetBuildableBook, b.Validate)
	if err != nil {
		return err
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

	return nil
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
	defer b.OrderbookBuilder.Release()

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
				Date:                 data.Data.OrderCreationTime,
				LastUpdated:          data.Data.TransactionTime,
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
				Timestamp:    t.TimeStamp,
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
			LastUpdated:  t.EventTime,
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
			Timestamp:  kline.EventTime,
			Pair:       pair,
			AssetType:  asset.Spot,
			Exchange:   b.Name,
			StartTime:  kline.Kline.StartTime,
			CloseTime:  kline.Kline.CloseTime,
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
			return err
		}
		return b.ProcessUpdate(context.TODO(), &depth)
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

// GenerateSubscriptions generates the default subscription set
func (b *Binance) GenerateSubscriptions() ([]subscription.Subscription, error) {
	var channels = make([]string, 0, len(b.Features.Subscriptions))
	for i := range b.Features.Subscriptions {
		name, err := channelName(b.Features.Subscriptions[i])
		if err != nil {
			return nil, err
		}
		channels = append(channels, name)
	}
	var subscriptions []subscription.Subscription
	pairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for y := range pairs {
		for z := range channels {
			lp := pairs[y].Lower()
			lp.Delimiter = ""
			subscriptions = append(subscriptions, subscription.Subscription{
				Channel: lp.String() + "@" + channels[z],
				Pair:    pairs[y],
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
func (b *Binance) Subscribe(channels []subscription.Subscription) error {
	return b.ParallelChanOp(channels, b.subscribeToChan, 50)
}

// subscribeToChan handles a single subscription and parses the result
// on success it adds the subscription to the websocket
func (b *Binance) subscribeToChan(chans []subscription.Subscription) error {
	id := b.Websocket.Conn.GenerateMessageID(false)

	cNames := make([]string, len(chans))
	for i := range chans {
		c := chans[i]
		cNames[i] = c.Channel
		c.State = subscription.SubscribingState
		if err := b.Websocket.AddSubscription(&c); err != nil {
			return fmt.Errorf("%w Channel: %s Pair: %s Error: %w", stream.ErrSubscriptionFailure, c.Channel, c.Pair, err)
		}
	}

	req := WsPayload{
		Method: wsSubscribeMethod,
		Params: cNames,
		ID:     id,
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
		b.Websocket.RemoveSubscriptions(chans...)
		err = fmt.Errorf("%w: %w; Channels: %s", stream.ErrSubscriptionFailure, err, strings.Join(cNames, ", "))
		b.Websocket.DataHandler <- err
	} else {
		b.Websocket.AddSuccessfulSubscriptions(chans...)
	}

	return err
}

// Unsubscribe unsubscribes from a set of channels
func (b *Binance) Unsubscribe(channels []subscription.Subscription) error {
	return b.ParallelChanOp(channels, b.unsubscribeFromChan, 50)
}

// unsubscribeFromChan sends a websocket message to stop receiving data from a channel
func (b *Binance) unsubscribeFromChan(chans []subscription.Subscription) error {
	id := b.Websocket.Conn.GenerateMessageID(false)

	cNames := make([]string, len(chans))
	for i := range chans {
		cNames[i] = chans[i].Channel
	}

	req := WsPayload{
		Method: wsUnsubscribeMethod,
		Params: cNames,
		ID:     id,
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
		b.Websocket.RemoveSubscriptions(chans...)
	}

	return nil
}
