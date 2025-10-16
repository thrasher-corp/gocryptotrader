package bithumb

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	wsEndpoint       = "wss://pubwss.bithumb.com/pub/ws"
	tickerTimeLayout = "20060102150405"
	tradeTimeLayout  = time.DateTime + ".000000"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Interval: kline.ThirtyMin}, // alternatives "1H", "12H", "24H"
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
}

// WsConnect initiates a websocket connection
func (e *Exchange) WsConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}

	var dialer gws.Dialer
	dialer.HandshakeTimeout = e.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment

	err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %w", e.Name, err)
	}

	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)

	e.setupOrderbookManager(ctx)
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData(ctx context.Context) {
	defer e.Websocket.Wg.Done()

	for {
		select {
		case <-e.Websocket.ShutdownC:
			return
		default:
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
}

func (e *Exchange) wsHandleData(ctx context.Context, respRaw []byte) error {
	var resp WsResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}

	if resp.Status != "" {
		if resp.Status == "0000" {
			return nil
		}
		return fmt.Errorf("%s: %w",
			resp.ResponseMessage,
			websocket.ErrSubscriptionFailure)
	}

	switch resp.Type {
	case "ticker":
		var tick WsTicker
		err = json.Unmarshal(resp.Content, &tick)
		if err != nil {
			return err
		}
		var lu time.Time
		lu, err = time.ParseInLocation(tickerTimeLayout, tick.Date+tick.Time, e.location)
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
			ExchangeName: e.Name,
			AssetType:    asset.Spot,
			Last:         tick.PreviousClosePrice,
			Pair:         tick.Symbol,
			Open:         tick.OpenPrice,
			Close:        tick.ClosePrice,
			Low:          tick.LowPrice,
			High:         tick.HighPrice,
			QuoteVolume:  tick.Value,
			Volume:       tick.Volume,
			LastUpdated:  lu,
		})
	case "transaction":
		if !e.IsSaveTradeDataEnabled() {
			return nil
		}

		var trades WsTransactions
		err = json.Unmarshal(resp.Content, &trades)
		if err != nil {
			return err
		}

		toBuffer := make([]trade.Data, len(trades.List))
		var lu time.Time
		for x := range trades.List {
			lu, err = time.ParseInLocation(tradeTimeLayout, trades.List[x].ContractTime, e.location)
			if err != nil {
				return err
			}

			toBuffer[x] = trade.Data{
				Exchange:     e.Name,
				AssetType:    asset.Spot,
				CurrencyPair: trades.List[x].Symbol,
				Timestamp:    lu,
				Price:        trades.List[x].ContractPrice,
				Amount:       trades.List[x].ContractAmount,
			}
		}

		err = e.AddTradesToBuffer(toBuffer...)
		if err != nil {
			return err
		}
	case "orderbookdepth":
		var orderbooks WsOrderbooks
		err = json.Unmarshal(resp.Content, &orderbooks)
		if err != nil {
			return err
		}
		init, err := e.UpdateLocalBuffer(&orderbooks)
		if err != nil && !init {
			return fmt.Errorf("%v - UpdateLocalCache error: %s", e.Name, err)
		}
		return nil
	default:
		return fmt.Errorf("unhandled response type %s", resp.Type)
	}

	return nil
}

// generateSubscriptions generates the default subscription set
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{"subToReq": subToReq}).Parse(subTplText)
}

// Subscribe subscribes to a set of channels
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	var errs error
	for _, s := range subs {
		err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, json.RawMessage(s.QualifiedChannel))
		if err == nil {
			err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// subToReq returns the subscription as a map to populate WsSubscribe
func subToReq(s *subscription.Subscription, p currency.Pairs) *WsSubscribe {
	req := &WsSubscribe{
		Type:    s.Channel,
		Symbols: common.SortStrings(p),
	}
	switch s.Channel {
	case subscription.TickerChannel:
		// As-is
	case subscription.OrderbookChannel:
		req.Type = "orderbookdepth"
	case subscription.AllTradesChannel:
		req.Type = "transaction"
	default:
		panic(fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel))
	}
	if s.Interval > 0 {
		req.TickTypes = []string{strings.ToUpper(s.Interval.Short())}
	}
	return req
}

const subTplText = `
{{ range $asset, $pairs := $.AssetPairs }}
	{{- subToReq $.S $pairs | mustToJson }}
	{{- $.AssetSeparator }}
{{- end }}
`
