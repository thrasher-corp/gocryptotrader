package huobi

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	wsSpotHost    = "api.huobi.pro"
	wsSpotURL     = "wss://" + wsSpotHost
	wsPublicPath  = "/ws"
	wsPrivatePath = "/ws/v2"

	wsCandlesChannel      = "market.%s.kline"
	wsOrderbookChannel    = "market.%s.depth"
	wsTradesChannel       = "market.%s.trade.detail"
	wsMarketDetailChannel = "market.%s.detail"
	wsMyOrdersChannel     = "orders#*"
	wsMyTradesChannel     = "trade.clearing#*#1" // 0=Only trade events, 1=Trade and Cancellation events
	wsMyAccountChannel    = "accounts.update#2"  // 0=Only balance, 1=Balance or Available, 2=Balance and Available when either change
	wsAuthChannel         = "auth"

	wsDateTimeFormatting = "2006-01-02T15:04:05"
	signatureMethod      = "HmacSHA256"
	signatureVersion     = "2.1"
	wsRequestOp          = "req"
	wsSubOp              = "sub"
	wsUnsubOp            = "unsub"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 0}, // Aggregation Levels; 0 is no depth aggregation
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    wsMarketDetailChannel,
	subscription.CandlesChannel:   wsCandlesChannel,
	subscription.OrderbookChannel: wsOrderbookChannel,
	subscription.AllTradesChannel: wsTradesChannel,
	subscription.MyTradesChannel:  wsMyTradesChannel,
	subscription.MyOrdersChannel:  wsMyOrdersChannel,
	subscription.MyAccountChannel: wsMyAccountChannel,
}

// WsConnect initiates a new websocket connection
func (h *HUOBI) WsConnect() error {
	ctx := context.TODO()
	if !h.Websocket.IsEnabled() || !h.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	if err := h.Websocket.Conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}

	h.Websocket.Wg.Add(1)
	go h.wsReadMsgs(ctx, h.Websocket.Conn)

	if h.IsWebsocketAuthenticationSupported() {
		if err := h.wsAuthConnect(ctx); err != nil {
			h.Websocket.SetCanUseAuthenticatedEndpoints(false)
			return fmt.Errorf("error authenticating websocket: %w", err)
		}
		h.Websocket.SetCanUseAuthenticatedEndpoints(true)
		h.Websocket.Wg.Add(1)
		go h.wsReadMsgs(ctx, h.Websocket.AuthConn)
	}

	return nil
}

// wsReadMsgs reads and processes messages from a websocket connection
func (h *HUOBI) wsReadMsgs(ctx context.Context, s websocket.Connection) {
	defer h.Websocket.Wg.Done()
	for {
		msg := s.ReadMessage()
		if msg.Raw == nil {
			return
		}

		if err := h.wsHandleData(ctx, msg.Raw); err != nil {
			h.Websocket.DataHandler <- err
		}
	}
}

func (h *HUOBI) wsHandleData(ctx context.Context, respRaw []byte) error {
	if id, err := jsonparser.GetString(respRaw, "id"); err == nil {
		if h.Websocket.Match.IncomingWithData(id, respRaw) {
			return nil
		}
	}

	if pingValue, err := jsonparser.GetInt(respRaw, "ping"); err == nil {
		return h.wsHandleV1ping(ctx, int(pingValue))
	}

	if action, err := jsonparser.GetString(respRaw, "action"); err == nil {
		switch action {
		case "ping":
			return h.wsHandleV2ping(ctx, respRaw)
		case wsSubOp, wsUnsubOp:
			return h.wsHandleV2subResp(action, respRaw)
		}
	}

	if err := getErrResp(respRaw); err != nil {
		return err
	}

	if ch, err := jsonparser.GetString(respRaw, "ch"); err == nil {
		s := h.Websocket.GetSubscription(ch)
		if s == nil {
			return fmt.Errorf("%w: %q", subscription.ErrNotFound, ch)
		}
		return h.wsHandleChannelMsgs(s, respRaw)
	}

	h.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
		Message: h.Name + websocket.UnhandledMessage + string(respRaw),
	}

	return nil
}

// wsHandleV1ping handles v1 style pings, currently only used with public connections
func (h *HUOBI) wsHandleV1ping(ctx context.Context, pingValue int) error {
	if err := h.Websocket.Conn.SendJSONMessage(ctx, request.Unset, json.RawMessage(`{"pong":`+strconv.Itoa(pingValue)+`}`)); err != nil {
		return fmt.Errorf("error sending pong response: %w", err)
	}
	return nil
}

// wsHandleV2ping handles v2 style pings, currently only used with private connections
func (h *HUOBI) wsHandleV2ping(ctx context.Context, respRaw []byte) error {
	ts, err := jsonparser.GetInt(respRaw, "data", "ts")
	if err != nil {
		return fmt.Errorf("error getting ts from auth ping: %w", err)
	}
	if err := h.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, json.RawMessage(`{"action":"pong","data":{"ts":`+strconv.Itoa(int(ts))+`}}`)); err != nil {
		return fmt.Errorf("error sending auth pong response: %w", err)
	}
	return nil
}

func (h *HUOBI) wsHandleV2subResp(action string, respRaw []byte) error {
	if ch, err := jsonparser.GetString(respRaw, "ch"); err == nil {
		return h.Websocket.Match.RequireMatchWithData(action+":"+ch, respRaw)
	}
	return nil
}

func (h *HUOBI) wsHandleChannelMsgs(s *subscription.Subscription, respRaw []byte) error {
	switch s.Channel {
	case subscription.TickerChannel:
		return h.wsHandleTickerMsg(s, respRaw)
	case subscription.OrderbookChannel:
		return h.wsHandleOrderbookMsg(s, respRaw)
	case subscription.CandlesChannel:
		return h.wsHandleCandleMsg(s, respRaw)
	case subscription.AllTradesChannel:
		return h.wsHandleAllTradesMsg(s, respRaw)
	case subscription.MyAccountChannel:
		return h.wsHandleMyAccountMsg(respRaw)
	case subscription.MyOrdersChannel:
		return h.wsHandleMyOrdersMsg(s, respRaw)
	case subscription.MyTradesChannel:
		return h.wsHandleMyTradesMsg(s, respRaw)
	}
	return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, s.Channel)
}

func (h *HUOBI) wsHandleCandleMsg(s *subscription.Subscription, respRaw []byte) error {
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var c WsKline
	if err := json.Unmarshal(respRaw, &c); err != nil {
		return err
	}
	h.Websocket.DataHandler <- websocket.KlineData{
		Timestamp:  c.Timestamp.Time(),
		Exchange:   h.Name,
		AssetType:  s.Asset,
		Pair:       s.Pairs[0],
		OpenPrice:  c.Tick.Open,
		ClosePrice: c.Tick.Close,
		HighPrice:  c.Tick.High,
		LowPrice:   c.Tick.Low,
		Volume:     c.Tick.Volume,
		Interval:   s.Interval.String(),
	}
	return nil
}

func (h *HUOBI) wsHandleAllTradesMsg(s *subscription.Subscription, respRaw []byte) error {
	saveTradeData := h.IsSaveTradeDataEnabled()
	tradeFeed := h.IsTradeFeedEnabled()
	if !saveTradeData && !tradeFeed {
		return nil
	}
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var t WsTrade
	if err := json.Unmarshal(respRaw, &t); err != nil {
		return err
	}
	trades := make([]trade.Data, 0, len(t.Tick.Data))
	for i := range t.Tick.Data {
		side := order.Buy
		if t.Tick.Data[i].Direction != "buy" {
			side = order.Sell
		}
		trades = append(trades, trade.Data{
			Exchange:     h.Name,
			AssetType:    s.Asset,
			CurrencyPair: s.Pairs[0],
			Timestamp:    t.Tick.Data[i].Timestamp.Time().UTC(),
			Amount:       t.Tick.Data[i].Amount,
			Price:        t.Tick.Data[i].Price,
			Side:         side,
			TID:          strconv.FormatFloat(t.Tick.Data[i].TradeID, 'f', -1, 64),
		})
	}
	if tradeFeed {
		for i := range trades {
			h.Websocket.DataHandler <- trades[i]
		}
	}
	if saveTradeData {
		return trade.AddTradesToBuffer(trades...)
	}
	return nil
}

func (h *HUOBI) wsHandleTickerMsg(s *subscription.Subscription, respRaw []byte) error {
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var wsTicker WsTick
	if err := json.Unmarshal(respRaw, &wsTicker); err != nil {
		return err
	}
	h.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: h.Name,
		Open:         wsTicker.Tick.Open,
		Close:        wsTicker.Tick.Close,
		Volume:       wsTicker.Tick.Amount,
		QuoteVolume:  wsTicker.Tick.Volume,
		High:         wsTicker.Tick.High,
		Low:          wsTicker.Tick.Low,
		LastUpdated:  wsTicker.Timestamp.Time(),
		AssetType:    s.Asset,
		Pair:         s.Pairs[0],
	}
	return nil
}

func (h *HUOBI) wsHandleOrderbookMsg(s *subscription.Subscription, respRaw []byte) error {
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var update WsDepth
	if err := json.Unmarshal(respRaw, &update); err != nil {
		return err
	}
	bids := make(orderbook.Levels, len(update.Tick.Bids))
	for i := range update.Tick.Bids {
		price, ok := update.Tick.Bids[i][0].(float64)
		if !ok {
			return errors.New("unable to type assert bid price")
		}
		amount, ok := update.Tick.Bids[i][1].(float64)
		if !ok {
			return errors.New("unable to type assert bid amount")
		}
		bids[i] = orderbook.Level{
			Price:  price,
			Amount: amount,
		}
	}

	asks := make(orderbook.Levels, len(update.Tick.Asks))
	for i := range update.Tick.Asks {
		price, ok := update.Tick.Asks[i][0].(float64)
		if !ok {
			return errors.New("unable to type assert ask price")
		}
		amount, ok := update.Tick.Asks[i][1].(float64)
		if !ok {
			return errors.New("unable to type assert ask amount")
		}
		asks[i] = orderbook.Level{
			Price:  price,
			Amount: amount,
		}
	}

	var newOrderBook orderbook.Book
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.Pair = s.Pairs[0]
	newOrderBook.Asset = asset.Spot
	newOrderBook.Exchange = h.Name
	newOrderBook.ValidateOrderbook = h.ValidateOrderbook
	newOrderBook.LastUpdated = update.Timestamp.Time()

	return h.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

func (h *HUOBI) wsHandleMyOrdersMsg(s *subscription.Subscription, respRaw []byte) error {
	var msg wsOrderUpdateMsg
	if err := json.Unmarshal(respRaw, &msg); err != nil {
		return err
	}
	o := msg.Data
	p, err := h.CurrencyPairs.Match(o.Symbol, s.Asset)
	if err != nil {
		return err
	}
	d := &order.Detail{
		ClientOrderID:   o.ClientOrderID,
		Price:           o.Price,
		Amount:          o.Size,
		ExecutedAmount:  o.ExecutedAmount,
		RemainingAmount: o.RemainingAmount,
		Exchange:        h.Name,
		Side:            o.Side,
		AssetType:       s.Asset,
		Pair:            p,
	}
	if o.OrderID != 0 {
		d.OrderID = strconv.FormatInt(o.OrderID, 10)
	}
	switch o.EventType {
	case "trigger", "deletion", "cancellation":
		d.LastUpdated = o.LastActTime.Time()
	case "creation":
		d.LastUpdated = o.CreateTime.Time()
	case "trade":
		d.LastUpdated = o.TradeTime.Time()
	}
	if d.Status, err = order.StringToOrderStatus(o.OrderStatus); err != nil {
		return &order.ClassificationError{
			Exchange: h.Name,
			OrderID:  d.OrderID,
			Err:      err,
		}
	}
	if o.Side == order.UnknownSide {
		d.Side, err = stringToOrderSide(o.OrderType)
		if err != nil {
			return &order.ClassificationError{
				Exchange: h.Name,
				OrderID:  d.OrderID,
				Err:      err,
			}
		}
	}
	if o.OrderType != "" {
		d.Type, err = stringToOrderType(o.OrderType)
		if err != nil {
			return &order.ClassificationError{
				Exchange: h.Name,
				OrderID:  d.OrderID,
				Err:      err,
			}
		}
	}
	h.Websocket.DataHandler <- d
	if o.ErrCode != 0 {
		return fmt.Errorf("error with order %q: %s (%v)", o.ClientOrderID, o.ErrMessage, o.ErrCode)
	}
	return nil
}

func (h *HUOBI) wsHandleMyTradesMsg(s *subscription.Subscription, respRaw []byte) error {
	var msg wsTradeUpdateMsg
	if err := json.Unmarshal(respRaw, &msg); err != nil {
		return err
	}
	t := msg.Data
	p, err := h.CurrencyPairs.Match(t.Symbol, s.Asset)
	if err != nil {
		return err
	}
	d := &order.Detail{
		ClientOrderID: t.ClientOrderID,
		Price:         t.OrderPrice,
		Amount:        t.OrderSize,
		Exchange:      h.Name,
		Side:          t.Side,
		AssetType:     s.Asset,
		Pair:          p,
		Date:          t.OrderCreateTime.Time(),
		LastUpdated:   t.TradeTime.Time(),
		OrderID:       strconv.FormatInt(t.OrderID, 10),
	}
	if d.Status, err = order.StringToOrderStatus(t.OrderStatus); err != nil {
		return &order.ClassificationError{
			Exchange: h.Name,
			OrderID:  d.OrderID,
			Err:      err,
		}
	}
	if t.Side == order.UnknownSide {
		d.Side, err = stringToOrderSide(t.OrderType)
		if err != nil {
			return &order.ClassificationError{
				Exchange: h.Name,
				OrderID:  d.OrderID,
				Err:      err,
			}
		}
	}
	if t.OrderType != "" {
		d.Type, err = stringToOrderType(t.OrderType)
		if err != nil {
			return &order.ClassificationError{
				Exchange: h.Name,
				OrderID:  d.OrderID,
				Err:      err,
			}
		}
	}
	d.Trades = []order.TradeHistory{
		{
			Price:     t.TradePrice,
			Amount:    t.TradeVolume,
			Fee:       t.TransactFee,
			Exchange:  h.Name,
			TID:       strconv.FormatInt(t.TradeID, 10),
			Type:      d.Type,
			Side:      d.Side,
			IsMaker:   !t.IsTaker,
			Timestamp: t.TradeTime.Time(),
		},
	}
	h.Websocket.DataHandler <- d
	return nil
}

func (h *HUOBI) wsHandleMyAccountMsg(respRaw []byte) error {
	u := &wsAccountUpdateMsg{}
	if err := json.Unmarshal(respRaw, u); err != nil {
		return err
	}
	h.Websocket.DataHandler <- u.Data
	return nil
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (h *HUOBI) generateSubscriptions() (subscription.List, error) {
	return h.Features.Subscriptions.ExpandTemplates(h)
}

// GetSubscriptionTemplate returns a subscription channel template
func (h *HUOBI) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":       channelName,
		"isWildcardChannel": isWildcardChannel,
		"interval":          h.FormatExchangeKlineInterval,
	}).Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from the channel
func (h *HUOBI) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	subs, errs := subs.ExpandTemplates(h)
	return common.AppendError(errs, h.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error { return h.manageSubs(ctx, wsSubOp, l) }, 1))
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HUOBI) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	subs, errs := subs.ExpandTemplates(h)
	return common.AppendError(errs, h.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error { return h.manageSubs(ctx, wsUnsubOp, l) }, 1))
}

func (h *HUOBI) manageSubs(ctx context.Context, op string, subs subscription.List) error {
	if len(subs) != 1 {
		return subscription.ErrBatchingNotSupported
	}
	s := subs[0]
	var c websocket.Connection
	var req any
	if s.Authenticated {
		c = h.Websocket.AuthConn
		req = wsReq{Action: op, Channel: s.QualifiedChannel}
	} else {
		c = h.Websocket.Conn
		if op == wsSubOp {
			// Set the id to the channel so that V1 errors can make it back to us
			req = wsSubReq{ID: wsSubOp + ":" + s.QualifiedChannel, Sub: s.QualifiedChannel}
		} else {
			req = wsSubReq{Unsub: s.QualifiedChannel}
		}
	}
	if op == wsSubOp {
		s.SetKey(s.QualifiedChannel)
		if err := h.Websocket.AddSubscriptions(c, s); err != nil {
			return fmt.Errorf("%w: %s; error: %w", websocket.ErrSubscriptionFailure, s, err)
		}
	}
	respRaw, err := c.SendMessageReturnResponse(ctx, request.Unset, wsSubOp+":"+s.QualifiedChannel, req)
	if err == nil {
		err = getErrResp(respRaw)
	}
	if err != nil {
		if op == wsSubOp {
			_ = h.Websocket.RemoveSubscriptions(c, s)
		}
		return fmt.Errorf("%s: %w", s, err)
	}
	if op == wsSubOp {
		err = s.SetState(subscription.SubscribedState)
		if h.Verbose {
			log.Debugf(log.ExchangeSys, "%s Subscribed to %s", h.Name, s)
		}
	} else {
		err = h.Websocket.RemoveSubscriptions(c, s)
	}
	return err
}

func (h *HUOBI) wsGenerateSignature(creds *account.Credentials, timestamp string) ([]byte, error) {
	values := url.Values{}
	values.Set("accessKey", creds.Key)
	values.Set("signatureMethod", signatureMethod)
	values.Set("signatureVersion", signatureVersion)
	values.Set("timestamp", timestamp)
	payload := http.MethodGet + "\n" + wsSpotHost + "\n" + wsPrivatePath + "\n" + values.Encode()
	return crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(creds.Secret))
}

func (h *HUOBI) wsAuthConnect(ctx context.Context) error {
	if err := h.Websocket.AuthConn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return fmt.Errorf("authenticated dial failed: %w", err)
	}
	if err := h.wsLogin(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	return nil
}

func (h *HUOBI) wsLogin(ctx context.Context) error {
	creds, err := h.GetCredentials(ctx)
	if err != nil {
		return err
	}

	ts := time.Now().UTC().Format(wsDateTimeFormatting)
	hmac, err := h.wsGenerateSignature(creds, ts)
	if err != nil {
		return err
	}
	req := wsReq{
		Action:  wsRequestOp,
		Channel: wsAuthChannel,
		Params: wsAuthReq{
			AuthType:         "api",
			AccessKey:        creds.Key,
			SignatureMethod:  signatureMethod,
			SignatureVersion: signatureVersion,
			Signature:        base64.StdEncoding.EncodeToString(hmac),
			Timestamp:        ts,
		},
	}
	c := h.Websocket.AuthConn
	if err := c.SendJSONMessage(ctx, request.Unset, req); err != nil {
		return err
	}
	resp := c.ReadMessage()
	if resp.Raw == nil {
		return &gws.CloseError{Code: gws.CloseAbnormalClosure}
	}

	return getErrResp(resp.Raw)
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "rejected":
		return order.Rejected, nil
	case "submitted":
		return order.New, nil
	case "partial-filled":
		return order.PartiallyFilled, nil
	case "filled":
		return order.Filled, nil
	case "partial-canceled":
		return order.PartiallyCancelled, nil
	case "canceled":
		return order.Cancelled, nil
	default:
		return order.UnknownStatus, errors.New(status + " not recognised as order status")
	}
}

func stringToOrderSide(side string) (order.Side, error) {
	switch {
	case strings.Contains(side, "buy"):
		return order.Buy, nil
	case strings.Contains(side, "sell"):
		return order.Sell, nil
	}

	return order.UnknownSide, errors.New(side + " not recognised as order side")
}

func stringToOrderType(oType string) (order.Type, error) {
	switch {
	case strings.Contains(oType, "limit"):
		return order.Limit, nil
	case strings.Contains(oType, "market"):
		return order.Market, nil
	}

	return order.UnknownType,
		errors.New(oType + " not recognised as order type")
}

/*
getErrResp looks for any of the following to determine an error:
- An err-code (V1)
- A code field that isn't 200 (V2)
Error message is retreieved from the field err-message or message.
Errors are returned in the format of <message> (<code>)
*/
func getErrResp(msg []byte) error {
	var errCode string
	errMsg, _ := jsonparser.GetString(msg, "err-msg")
	errCode, err := jsonparser.GetString(msg, "err-code")
	switch err {
	case nil: // Nothing to do
	case jsonparser.KeyPathNotFoundError: // Look for a V2 error
		errCodeInt, err := jsonparser.GetInt(msg, "code")
		if errCodeInt == 200 || errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%w 'code': %w from message: %s", common.ErrParsingWSField, err, msg)
		}
		errCode = strconv.Itoa(int(errCodeInt))
		errMsg, _ = jsonparser.GetString(msg, "message")
	}
	if errCode != "" {
		return fmt.Errorf("%s (%v)", errMsg, errCode)
	}
	return nil
}

// channelName converts global channel Names used in config of channel input into exchange channel names
// returns the name unchanged if no match is found
func channelName(s *subscription.Subscription, p ...currency.Pair) string {
	if n, ok := subscriptionNames[s.Channel]; ok {
		if strings.Contains(n, "%s") {
			return fmt.Sprintf(n, p[0])
		}
		return n
	}
	panic(subscription.ErrUseConstChannelName)
}

func isWildcardChannel(s *subscription.Subscription) bool {
	return s.Channel == subscription.MyTradesChannel || s.Channel == subscription.MyOrdersChannel
}

const subTplText = `
{{- if $.S.Asset }}
	{{ range $asset, $pairs := $.AssetPairs }}
		{{- if isWildcardChannel $.S }}
			{{- channelName $.S -}}
		{{- else }}
			{{- range $p := $pairs }}
				{{- channelName $.S $p -}}
				{{- if eq $.S.Channel "candles" -}} . {{- interval $.S.Interval }}{{ end }}
				{{- if eq $.S.Channel "orderbook" -}} .step {{- $.S.Levels }}{{ end }}
				{{ $.PairSeparator }}
			{{- end }}
		{{- end }}
		{{ $.AssetSeparator }}
	{{- end }}
{{- else -}}
	{{ channelName $.S }}
{{- end }}
`
