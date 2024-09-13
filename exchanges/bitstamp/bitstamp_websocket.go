package bitstamp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bitstampWSURL = "wss://ws.bitstamp.net" //nolint // gosec false positive
	hbInterval    = 8 * time.Second         // Connection monitor defaults to 10s inactivity
)

var (
	hbMsg = []byte(`{"event":"bts:heartbeat"}`)

	defaultSubChannels = []string{
		bitstampAPIWSTrades,
		bitstampAPIWSOrderbook,
	}

	defaultAuthSubChannels = []string{
		bitstampAPIWSMyOrders,
		bitstampAPIWSMyTrades,
	}
)

// WsConnect connects to a websocket feed
func (b *Bitstamp) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	err := b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if b.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", b.Name)
	}
	b.Websocket.Conn.SetupPingHandler(request.Unset, stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     hbMsg,
		Delay:       hbInterval,
	})
	err = b.seedOrderBook(context.TODO())
	if err != nil {
		b.Websocket.DataHandler <- err
	}

	b.Websocket.Wg.Add(1)
	go b.wsReadData()

	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (b *Bitstamp) wsReadData() {
	defer b.Websocket.Wg.Done()

	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		if err := b.wsHandleData(resp.Raw); err != nil {
			b.Websocket.DataHandler <- err
		}
	}
}

func (b *Bitstamp) wsHandleData(respRaw []byte) error {
	wsResponse := &websocketResponse{}
	if err := json.Unmarshal(respRaw, wsResponse); err != nil {
		return err
	}

	if err := b.parseChannelName(wsResponse); err != nil {
		return err
	}

	switch wsResponse.Event {
	case "bts:heartbeat":
		return nil
	case "bts:subscribe", "bts:subscription_succeeded":
		if b.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket subscription acknowledgement", b.Name)
		}
	case "bts:unsubscribe":
		if b.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket unsubscribe acknowledgement", b.Name)
		}
	case "bts:request_reconnect":
		if b.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket reconnection request received", b.Name)
		}
		go func() {
			err := b.Websocket.Shutdown()
			if err != nil {
				log.Errorf(log.WebsocketMgr, "%s failed to shutdown websocket: %v", b.Name, err)
			}
		}() // Connection monitor will reconnect
	case "data":
		if err := b.handleWSOrderbook(wsResponse, respRaw); err != nil {
			return err
		}
	case "trade":
		if err := b.handleWSTrade(wsResponse, respRaw); err != nil {
			return err
		}
	case "order_created", "order_deleted", "order_changed":
		// Only process MyOrders, not orders from the LiveOrder channel
		if wsResponse.channelType == bitstampAPIWSMyOrders {
			if err := b.handleWSOrder(wsResponse, respRaw); err != nil {
				return err
			}
		}
	default:
		b.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: b.Name + stream.UnhandledMessage + string(respRaw)}
	}
	return nil
}

func (b *Bitstamp) handleWSOrderbook(wsResp *websocketResponse, msg []byte) error {
	if wsResp.pair.IsEmpty() {
		return errWSPairParsingError
	}

	wsOrderBookTemp := websocketOrderBookResponse{}
	err := json.Unmarshal(msg, &wsOrderBookTemp)
	if err != nil {
		return err
	}

	return b.wsUpdateOrderbook(&wsOrderBookTemp.Data, wsResp.pair, asset.Spot)
}

func (b *Bitstamp) handleWSTrade(wsResp *websocketResponse, msg []byte) error {
	if !b.IsSaveTradeDataEnabled() {
		return nil
	}

	if wsResp.pair.IsEmpty() {
		return errWSPairParsingError
	}

	wsTradeTemp := websocketTradeResponse{}
	if err := json.Unmarshal(msg, &wsTradeTemp); err != nil {
		return err
	}

	side := order.Buy
	if wsTradeTemp.Data.Type == 1 {
		side = order.Sell
	}
	return trade.AddTradesToBuffer(b.Name, trade.Data{
		Timestamp:    time.Unix(wsTradeTemp.Data.Timestamp, 0),
		CurrencyPair: wsResp.pair,
		AssetType:    asset.Spot,
		Exchange:     b.Name,
		Price:        wsTradeTemp.Data.Price,
		Amount:       wsTradeTemp.Data.Amount,
		Side:         side,
		TID:          strconv.FormatInt(wsTradeTemp.Data.ID, 10),
	})
}

func (b *Bitstamp) handleWSOrder(wsResp *websocketResponse, msg []byte) error {
	r := &websocketOrderResponse{}
	if err := json.Unmarshal(msg, &r); err != nil {
		return err
	}

	if r.Order.ID == 0 && r.Order.ClientOrderID == "" {
		return fmt.Errorf("unable to parse an order id from order msg: %s", msg)
	}

	var status order.Status
	switch wsResp.Event {
	case "order_created":
		status = order.New
	case "order_changed":
		if r.Order.ExecutedAmount > 0 {
			status = order.PartiallyFilled
		}
	case "order_deleted":
		if r.Order.RemainingAmount == 0 && r.Order.Amount > 0 {
			status = order.Filled
		} else {
			status = order.Cancelled
		}
	}

	// r.Order.ExecutedAmount is an atomic partial fill amount; We want total
	executedAmount := r.Order.Amount - r.Order.RemainingAmount

	d := &order.Detail{
		Price:           r.Order.Price,
		Amount:          r.Order.Amount,
		RemainingAmount: r.Order.RemainingAmount,
		ExecutedAmount:  executedAmount,
		Exchange:        b.Name,
		OrderID:         r.Order.IDStr,
		ClientOrderID:   r.Order.ClientOrderID,
		Side:            r.Order.Side.Side(),
		Status:          status,
		AssetType:       asset.Spot,
		Date:            r.Order.Microtimestamp.Time(),
		Pair:            wsResp.pair,
	}

	b.Websocket.DataHandler <- d

	return nil
}

func (b *Bitstamp) generateDefaultSubscriptions() (subscription.List, error) {
	enabledCurrencies, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions subscription.List
	for i := range enabledCurrencies {
		p, err := b.FormatExchangeCurrency(enabledCurrencies[i], asset.Spot)
		if err != nil {
			return nil, err
		}
		for j := range defaultSubChannels {
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: defaultSubChannels[j] + "_" + p.String(),
				Asset:   asset.Spot,
				Pairs:   currency.Pairs{p},
			})
		}
		if b.Websocket.CanUseAuthenticatedEndpoints() {
			for j := range defaultAuthSubChannels {
				subscriptions = append(subscriptions, &subscription.Subscription{
					Channel: defaultAuthSubChannels[j] + "_" + p.String(),
					Asset:   asset.Spot,
					Pairs:   currency.Pairs{p},
					Params: map[string]interface{}{
						"auth": struct{}{},
					},
				})
			}
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (b *Bitstamp) Subscribe(channelsToSubscribe subscription.List) error {
	var errs error
	var auth *WebsocketAuthResponse

	for i := range channelsToSubscribe {
		if _, ok := channelsToSubscribe[i].Params["auth"]; ok {
			var err error
			auth, err = b.FetchWSAuth(context.TODO())
			if err != nil {
				errs = common.AppendError(errs, err)
			}
			break
		}
	}

	for _, s := range channelsToSubscribe {
		req := websocketEventRequest{
			Event: "bts:subscribe",
			Data: websocketData{
				Channel: s.Channel,
			},
		}
		if _, ok := s.Params["auth"]; ok && auth != nil {
			req.Data.Channel = "private-" + req.Data.Channel + "-" + strconv.Itoa(int(auth.UserID))
			req.Data.Auth = auth.Token
		}
		err := b.Websocket.Conn.SendJSONMessage(context.TODO(), request.Unset, req)
		if err == nil {
			err = b.Websocket.AddSuccessfulSubscriptions(s)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}

	return errs
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bitstamp) Unsubscribe(channelsToUnsubscribe subscription.List) error {
	var errs error
	for _, s := range channelsToUnsubscribe {
		req := websocketEventRequest{
			Event: "bts:unsubscribe",
			Data: websocketData{
				Channel: s.Channel,
			},
		}
		err := b.Websocket.Conn.SendJSONMessage(context.TODO(), request.Unset, req)
		if err == nil {
			err = b.Websocket.RemoveSubscriptions(s)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

func (b *Bitstamp) wsUpdateOrderbook(update *websocketOrderBook, p currency.Pair, assetType asset.Item) error {
	if len(update.Asks) == 0 && len(update.Bids) == 0 {
		return errors.New("no orderbook data")
	}

	obUpdate := &orderbook.Base{
		Bids:            make(orderbook.Tranches, len(update.Bids)),
		Asks:            make(orderbook.Tranches, len(update.Asks)),
		Pair:            p,
		LastUpdated:     time.UnixMicro(update.Microtimestamp),
		Asset:           assetType,
		Exchange:        b.Name,
		VerifyOrderbook: b.CanVerifyOrderbook,
	}

	for i := range update.Asks {
		target, err := strconv.ParseFloat(update.Asks[i][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Asks[i][1], 64)
		if err != nil {
			return err
		}
		obUpdate.Asks[i] = orderbook.Tranche{Price: target, Amount: amount}
	}
	for i := range update.Bids {
		target, err := strconv.ParseFloat(update.Bids[i][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Bids[i][1], 64)
		if err != nil {
			return err
		}
		obUpdate.Bids[i] = orderbook.Tranche{Price: target, Amount: amount}
	}
	filterOrderbookZeroBidPrice(obUpdate)
	return b.Websocket.Orderbook.LoadSnapshot(obUpdate)
}

func (b *Bitstamp) seedOrderBook(ctx context.Context) error {
	p, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	for x := range p {
		pairFmt, err := b.FormatExchangeCurrency(p[x], asset.Spot)
		if err != nil {
			return err
		}
		orderbookSeed, err := b.GetOrderbook(ctx, pairFmt.String())
		if err != nil {
			return err
		}

		newOrderBook := &orderbook.Base{
			Pair:            p[x],
			Asset:           asset.Spot,
			Exchange:        b.Name,
			VerifyOrderbook: b.CanVerifyOrderbook,
			Bids:            make(orderbook.Tranches, len(orderbookSeed.Bids)),
			Asks:            make(orderbook.Tranches, len(orderbookSeed.Asks)),
			LastUpdated:     time.Unix(orderbookSeed.Timestamp, 0),
		}

		for i := range orderbookSeed.Asks {
			newOrderBook.Asks[i] = orderbook.Tranche{
				Price:  orderbookSeed.Asks[i].Price,
				Amount: orderbookSeed.Asks[i].Amount,
			}
		}
		for i := range orderbookSeed.Bids {
			newOrderBook.Bids[i] = orderbook.Tranche{
				Price:  orderbookSeed.Bids[i].Price,
				Amount: orderbookSeed.Bids[i].Amount,
			}
		}

		filterOrderbookZeroBidPrice(newOrderBook)

		err = b.Websocket.Orderbook.LoadSnapshot(newOrderBook)
		if err != nil {
			return err
		}
	}
	return nil
}

// FetchWSAuth Retrieves a userID and auth-token from REST for subscribing to a websocket channel
// The token life-expectancy is only about 60s; use it immediately and do not store it
func (b *Bitstamp) FetchWSAuth(ctx context.Context) (*WebsocketAuthResponse, error) {
	resp := &WebsocketAuthResponse{}
	err := b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIWSAuthToken, true, nil, resp)
	if err != nil {
		return nil, fmt.Errorf("error fetching auth token: %w", err)
	}
	return resp, nil
}

// parseChannel splits the ws response channel and sets the channel type and pair
func (b *Bitstamp) parseChannelName(r *websocketResponse) error {
	if r.Channel == "" {
		return nil
	}

	chanName := r.Channel
	authParts := strings.Split(r.Channel, "-")
	switch len(authParts) {
	case 1:
		// Not an auth channel
	case 3:
		chanName = authParts[1]
	default:
		return fmt.Errorf("channel name does not contain exactly 0 or 2 hyphens: %v", r.Channel)
	}

	parts := strings.Split(chanName, "_")
	if len(parts) != 3 {
		return fmt.Errorf("%w: channel name does not contain exactly 2 underscores: %v", errWSPairParsingError, r.Channel)
	}

	r.channelType = parts[0] + "_" + parts[1]
	symbol := parts[2]

	enabledPairs, err := b.GetEnabledPairs(asset.Spot)
	if err == nil {
		r.pair, err = enabledPairs.DeriveFrom(symbol)
	}

	return err
}
