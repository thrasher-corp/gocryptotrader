package lbank

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"text/template"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	lbankWsSubscribe   = "subscribe"
	lbankWsUnsubscribe = "unsubscribe"
	lbankWsTicker      = "tick"
	lbankWsTrades      = "trade"
	lbankWsOrderbook   = "depth"
)

var (
	errMissingTickData  = errors.New("lbank: missing tick data in websocket message")
	errMissingTradeData = errors.New("lbank: missing trade data in websocket message")
	errMissingDepthData = errors.New("lbank: missing depth data in websocket message")
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    lbankWsTicker,
	subscription.AllTradesChannel: lbankWsTrades,
	subscription.OrderbookChannel: lbankWsOrderbook,
}

var defaultSubscriptionTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"channelName": func(s *subscription.Subscription) string {
		return subscriptionNames[s.Channel]
	},
}).Parse(`
{{- range $asset, $pairs := $.AssetPairs -}}
	{{- range $p := $pairs -}}
		{{- channelName $.S }}_{{ $p.Lower.String }}
		{{- $.PairSeparator }}
	{{- end -}}
	{{- $.AssetSeparator }}
{{- end -}}
`))

// WsConnect connects to the LBank websocket
func (e *Exchange) WsConnect() error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	ctx := context.TODO()
	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{}, nil)
	if err != nil {
		return err
	}
	if e.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", e.Name)
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)
	return nil
}

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
	var result map[string]json.RawMessage
	if err := json.Unmarshal(respRaw, &result); err != nil {
		return err
	}

	typeRaw, ok := result["type"]
	if !ok {
		if msgRaw, ok := result["message"]; ok {
			var errMsg string
			if err := json.Unmarshal(msgRaw, &errMsg); err != nil {
				return err
			}
			return fmt.Errorf("lbank websocket error: %s", errMsg)
		}
		return nil
	}

	var msgType string
	if err := json.Unmarshal(typeRaw, &msgType); err != nil {
		return err
	}

	switch msgType {
	case lbankWsTicker, lbankWsTrades, lbankWsOrderbook:
		pairRaw, ok := result["pair"]
		if !ok {
			return fmt.Errorf("lbank: missing pair in websocket message: %s", respRaw)
		}
		var pairStr string
		if err := json.Unmarshal(pairRaw, &pairStr); err != nil {
			return err
		}
		p, err := currency.NewPairFromString(pairStr)
		if err != nil {
			return err
		}

		switch msgType {
		case lbankWsTicker:
			return e.wsHandleTicker(ctx, result, p, respRaw)
		case lbankWsTrades:
			return e.wsHandleTrades(ctx, result, p, respRaw)
		case lbankWsOrderbook:
			return e.wsHandleOrderbook(result, p, respRaw)
		}
	}

	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
		Message: e.Name + websocket.UnhandledMessage + string(respRaw),
	})
}

func (e *Exchange) wsHandleTicker(ctx context.Context, result map[string]json.RawMessage, p currency.Pair, respRaw []byte) error {
	var tick struct {
		High   types.Number `json:"high"`
		Low    types.Number `json:"low"`
		Latest types.Number `json:"latest"`
		Vol    types.Number `json:"vol"`
		Change types.Number `json:"change"`
	}
	tickRaw, ok := result[lbankWsTicker]
	if !ok {
		return fmt.Errorf("%w: %s", errMissingTickData, respRaw)
	}
	if err := json.Unmarshal(tickRaw, &tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, (&ticker.Price{
		ExchangeName: e.Name,
		Pair:         p,
		AssetType:    asset.Spot,
		High:         tick.High.Float64(),
		Low:          tick.Low.Float64(),
		Last:         tick.Latest.Float64(),
		Volume:       tick.Vol.Float64(),
	}))
}

// wsHandleTrades handles trade websocket messages
func (e *Exchange) wsHandleTrades(_ context.Context, result map[string]json.RawMessage, p currency.Pair, respRaw []byte) error {
	if !e.IsSaveTradeDataEnabled() {
		return nil
	}
	var trades []struct {
		DateMs types.Time   `json:"date_ms"`
		Amount types.Number `json:"amount"`
		Price  types.Number `json:"price"`
		Type   string       `json:"type"`
		TID    string       `json:"tid"`
	}
	tradeRaw, ok := result[lbankWsTrades]
	if !ok {
		return fmt.Errorf("%w: %s", errMissingTradeData, respRaw)
	}
	if err := json.Unmarshal(tradeRaw, &trades); err != nil {
		return err
	}
	out := make([]trade.Data, len(trades))
	for i, t := range trades {
		side, err := order.StringToOrderSide(t.Type)
		if err != nil {
			return err
		}
		out[i] = trade.Data{
			Exchange:     e.Name,
			AssetType:    asset.Spot,
			CurrencyPair: p,
			Price:        t.Price.Float64(),
			Amount:       t.Amount.Float64(),
			Timestamp:    t.DateMs.Time(),
			Side:         side,
			TID:          t.TID,
		}
	}
	return trade.AddTradesToBuffer(out...)
}

func (e *Exchange) wsHandleOrderbook(result map[string]json.RawMessage, p currency.Pair, respRaw []byte) error {
	var depth struct {
		Asks [][]types.Number `json:"asks"`
		Bids [][]types.Number `json:"bids"`
	}
	depthRaw, ok := result[lbankWsOrderbook]
	if !ok {
		return fmt.Errorf("%w: %s", errMissingDepthData, respRaw)
	}
	if err := json.Unmarshal(depthRaw, &depth); err != nil {
		return err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             asset.Spot,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	book.Asks = make(orderbook.Levels, len(depth.Asks))
	for i := range depth.Asks {
		if len(depth.Asks[i]) < 2 {
			return fmt.Errorf("lbank: invalid ask level length %d", len(depth.Asks[i]))
		}
		book.Asks[i] = orderbook.Level{Price: depth.Asks[i][0].Float64(), Amount: depth.Asks[i][1].Float64()}
	}
	book.Bids = make(orderbook.Levels, len(depth.Bids))
	for i := range depth.Bids {
		if len(depth.Bids[i]) < 2 {
			return fmt.Errorf("lbank: invalid bid level length %d", len(depth.Bids[i]))
		}
		book.Bids[i] = orderbook.Level{Price: depth.Bids[i][0].Float64(), Amount: depth.Bids[i][1].Float64()}
	}
	return book.Process()
}

func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns the subscription template for LBank
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return defaultSubscriptionTemplate, nil
}

// Subscribe subscribes to a list of websocket channels
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	for _, s := range subs {
		chName, ok := subscriptionNames[s.Channel]
		if !ok {
			return fmt.Errorf("lbank: unsupported channel %s", s.Channel)
		}
		for _, p := range s.Pairs {
			req := map[string]string{
				"action":    lbankWsSubscribe,
				"subscribe": chName + "_" + p.Lower().String(),
			}
			if err := e.Websocket.Conn.SendJSONMessage(ctx, 0, req); err != nil {
				return err
			}
		}
		if err := e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s); err != nil {
			return err
		}
	}
	return nil
}

// Unsubscribe unsubscribes from a list of websocket channels
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	for _, s := range subs {
		chName, ok := subscriptionNames[s.Channel]
		if !ok {
			return fmt.Errorf("lbank: unsupported channel %s", s.Channel)
		}
		for _, p := range s.Pairs {
			req := map[string]string{
				"action":    lbankWsUnsubscribe,
				"subscribe": chName + "_" + p.Lower().String(),
			}
			if err := e.Websocket.Conn.SendJSONMessage(ctx, 0, req); err != nil {
				return err
			}
		}
		if err := e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s); err != nil {
			return err
		}
	}
	return nil
}
