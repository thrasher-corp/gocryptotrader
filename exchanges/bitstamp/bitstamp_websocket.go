package bitstamp

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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bitstampWSURL = "wss://ws.bitstamp.net" //nolint // gosec false positive
	hbInterval    = 8 * time.Second         // Connection monitor defaults to 10s inactivity
)

var (
	errParsingWSPair      = errors.New("unable to parse currency pair from wsResponse.Channel")
	errChannelHyphens     = errors.New("channel name does not contain exactly 0 or 2 hyphens")
	errChannelUnderscores = errors.New("channel name does not contain exactly 2 underscores")

	hbMsg = []byte(`{"event":"bts:heartbeat"}`)
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyTradesChannel, Authenticated: true},
}

var subscriptionNames = map[string]string{
	subscription.OrderbookChannel: bitstampAPIWSOrderbook,
	subscription.AllTradesChannel: bitstampAPIWSTrades,
	subscription.MyOrdersChannel:  bitstampAPIWSMyOrders,
	subscription.MyTradesChannel:  bitstampAPIWSMyTrades,
}

// WsConnect connects to a websocket feed
func (e *Exchange) WsConnect() error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	ctx := context.TODO()
	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	if e.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", e.Name)
	}
	e.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     hbMsg,
		Delay:       hbInterval,
	})
	err = e.seedOrderBook(ctx)
	if err != nil {
		if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
			log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
		}
	}

	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)

	return nil
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
	event, err := jsonparser.GetUnsafeString(respRaw, "event")
	if err != nil {
		return fmt.Errorf("%w `event`: %w", common.ErrParsingWSField, err)
	}

	event = strings.TrimPrefix(event, "bts:")
	switch event {
	case "heartbeat":
		return nil
	case "subscription_succeeded", "unsubscription_succeeded":
		return e.handleWSSubscription(event, respRaw)
	case "data":
		return e.handleWSOrderbook(respRaw)
	case "trade":
		return e.handleWSTrade(respRaw)
	case "order_created", "order_deleted", "order_changed":
		return e.handleWSOrder(ctx, event, respRaw)
	case "request_reconnect":
		go func() {
			if err := e.Websocket.Shutdown(); err != nil { // Connection monitor will reconnect
				log.Errorf(log.WebsocketMgr, "%s failed to shutdown websocket: %v", e.Name, err)
			}
		}()
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
	}
	return nil
}

func (e *Exchange) handleWSSubscription(event string, respRaw []byte) error {
	channel, err := jsonparser.GetUnsafeString(respRaw, "channel")
	if err != nil {
		return fmt.Errorf("%w `channel`: %w", common.ErrParsingWSField, err)
	}
	event = strings.TrimSuffix(event, "scription_succeeded")
	return e.Websocket.Match.RequireMatchWithData(event+":"+channel, respRaw)
}

func (e *Exchange) handleWSTrade(msg []byte) error {
	if !e.IsSaveTradeDataEnabled() {
		return nil
	}

	_, p, err := e.parseChannelName(msg)
	if err != nil {
		return err
	}

	wsTradeTemp := websocketTradeResponse{}
	if err := json.Unmarshal(msg, &wsTradeTemp); err != nil {
		return err
	}

	side := order.Buy
	if wsTradeTemp.Data.Type == 1 {
		side = order.Sell
	}
	return trade.AddTradesToBuffer(trade.Data{
		Timestamp:    wsTradeTemp.Data.Timestamp.Time(),
		CurrencyPair: p,
		AssetType:    asset.Spot,
		Exchange:     e.Name,
		Price:        wsTradeTemp.Data.Price,
		Amount:       wsTradeTemp.Data.Amount,
		Side:         side,
		TID:          strconv.FormatInt(wsTradeTemp.Data.ID, 10),
	})
}

func (e *Exchange) handleWSOrder(ctx context.Context, event string, msg []byte) error {
	channel, p, err := e.parseChannelName(msg)
	if err != nil {
		return err
	}
	if channel != bitstampAPIWSMyOrders {
		return nil // Only process MyOrders, not orders from the LiveOrder channel
	}

	r := &websocketOrderResponse{}
	if err := json.Unmarshal(msg, &r); err != nil {
		return err
	}

	if r.Order.ID == 0 && r.Order.ClientOrderID == "" {
		return fmt.Errorf("unable to parse an order id from order msg: %s", msg)
	}

	var status order.Status
	switch event {
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
		Exchange:        e.Name,
		OrderID:         r.Order.IDStr,
		ClientOrderID:   r.Order.ClientOrderID,
		Side:            r.Order.Side.Side(),
		Status:          status,
		AssetType:       asset.Spot,
		Date:            r.Order.Microtimestamp.Time(),
		Pair:            p,
	}

	return e.Websocket.DataHandler.Send(ctx, d)
}

func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{"channelName": channelName}).Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from a list of channels
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	return e.manageSubsWithCreds(ctx, subs, "sub")
}

// Unsubscribe sends a websocket message to stop receiving data from a list of channels
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	return e.manageSubsWithCreds(ctx, subs, "unsub")
}

func (e *Exchange) manageSubsWithCreds(ctx context.Context, subs subscription.List, op string) error {
	var errs error
	var creds *WebsocketAuthResponse
	if authed := subs.Private(); len(authed) > 0 {
		creds, errs = e.FetchWSAuth(ctx)
	}
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, s subscription.List) error { return e.manageSubs(ctx, s, op, creds) }, 1))
}

func (e *Exchange) manageSubs(ctx context.Context, subs subscription.List, op string, creds *WebsocketAuthResponse) error {
	subs, errs := subs.ExpandTemplates(e)
	for _, s := range subs {
		req := websocketEventRequest{
			Event: "bts:" + op + "scribe",
			Data: websocketData{
				Channel: s.QualifiedChannel,
			},
		}
		if s.Authenticated {
			if creds == nil {
				return request.ErrAuthRequestFailed
			}
			req.Data.Channel = "private-" + req.Data.Channel + "-" + strconv.FormatInt(creds.UserID, 10)
			req.Data.Auth = creds.Token
		}
		_, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, op+":"+req.Data.Channel, req)
		if err == nil {
			if op == "sub" {
				err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s)
			} else {
				err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s)
			}
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}

	return errs
}

func (e *Exchange) handleWSOrderbook(msg []byte) error {
	_, p, err := e.parseChannelName(msg)
	if err != nil {
		return err
	}

	var wsOrderBookResp websocketOrderBookResponse
	if err := json.Unmarshal(msg, &wsOrderBookResp); err != nil {
		return err
	}

	obUpdate := &orderbook.Book{
		Bids:              wsOrderBookResp.Data.Bids.Levels(),
		Asks:              wsOrderBookResp.Data.Asks.Levels(),
		Pair:              p,
		LastUpdated:       wsOrderBookResp.Data.Microtimestamp.Time(),
		Asset:             asset.Spot,
		Exchange:          e.Name,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	filterOrderbookZeroBidPrice(obUpdate)
	return e.Websocket.Orderbook.LoadSnapshot(obUpdate)
}

func (e *Exchange) seedOrderBook(ctx context.Context) error {
	p, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	for x := range p {
		pairFmt, err := e.FormatExchangeCurrency(p[x], asset.Spot)
		if err != nil {
			return err
		}
		orderbookSeed, err := e.GetOrderbook(ctx, pairFmt.String())
		if err != nil {
			return err
		}

		newOrderBook := &orderbook.Book{
			Pair:              p[x],
			Asset:             asset.Spot,
			Exchange:          e.Name,
			ValidateOrderbook: e.ValidateOrderbook,
			Bids:              make(orderbook.Levels, len(orderbookSeed.Bids)),
			Asks:              make(orderbook.Levels, len(orderbookSeed.Asks)),
			LastUpdated:       orderbookSeed.Timestamp,
		}

		for i := range orderbookSeed.Asks {
			newOrderBook.Asks[i] = orderbook.Level{
				Price:  orderbookSeed.Asks[i].Price,
				Amount: orderbookSeed.Asks[i].Amount,
			}
		}
		for i := range orderbookSeed.Bids {
			newOrderBook.Bids[i] = orderbook.Level{
				Price:  orderbookSeed.Bids[i].Price,
				Amount: orderbookSeed.Bids[i].Amount,
			}
		}

		filterOrderbookZeroBidPrice(newOrderBook)

		err = e.Websocket.Orderbook.LoadSnapshot(newOrderBook)
		if err != nil {
			return err
		}
	}
	return nil
}

// FetchWSAuth Retrieves a userID and auth-token from REST for subscribing to a websocket channel
// The token life-expectancy is only about 60s; use it immediately and do not store it
func (e *Exchange) FetchWSAuth(ctx context.Context) (*WebsocketAuthResponse, error) {
	resp := &WebsocketAuthResponse{}
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitstampAPIWSAuthToken, true, nil, resp)
	if err != nil {
		return nil, fmt.Errorf("error fetching auth token: %w", err)
	}
	return resp, nil
}

// parseChannelName splits the ws message channel and returns the channel name and pair
func (e *Exchange) parseChannelName(respRaw []byte) (string, currency.Pair, error) {
	channel, err := jsonparser.GetUnsafeString(respRaw, "channel")
	if err != nil {
		return "", currency.EMPTYPAIR, fmt.Errorf("%w `channel`: %w", common.ErrParsingWSField, err)
	}

	authParts := strings.Split(channel, "-")
	switch len(authParts) {
	case 1:
		// Not an auth channel
	case 3:
		channel = authParts[1]
	default:
		return "", currency.EMPTYPAIR, fmt.Errorf("%w: %s", errChannelHyphens, channel)
	}

	parts := strings.Split(channel, "_")
	if len(parts) != 3 {
		return "", currency.EMPTYPAIR, fmt.Errorf("%w: %s", errChannelUnderscores, channel)
	}

	enabledPairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return "", currency.EMPTYPAIR, err
	}

	pair, err := enabledPairs.DeriveFrom(parts[2])
	if err != nil {
		return "", currency.EMPTYPAIR, fmt.Errorf("%w: %s", errParsingWSPair, err)
	}

	return parts[0] + "_" + parts[1], pair, nil
}

// channelName converts global channel Names to exchange specific ones
// panics if name is not supported, so should be called within a recover chain
func channelName(s *subscription.Subscription) string {
	if s, ok := subscriptionNames[s.Channel]; ok {
		return s
	}
	panic(fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel))
}

const subTplText = `
{{ range $asset, $pairs := $.AssetPairs }}
	{{- with $name := channelName $.S }}
		{{- range $p := $pairs -}}
			{{- $name -}} _ {{- $p -}}
			{{ $.PairSeparator }}
		{{- end -}}
	{{- end }}
	{{ $.AssetSeparator }}
{{- end }}
`
