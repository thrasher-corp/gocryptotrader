package btse

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	btseWebsocket      = "wss://ws.btse.com/ws/spot"
	btseWebsocketTimer = time.Second * 57
)

var errWebsocketAuthenticationFailed = errors.New("websocket authentication failed")

var subscriptionNames = map[string]string{
	subscription.MyTradesChannel:  "notificationApi",
	subscription.AllTradesChannel: "tradeHistoryApi",
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true},
}

// WsConnect starts the managed websocket lifecycle.
func (e *Exchange) WsConnect() error {
	if !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	return e.Websocket.Connect(context.TODO())
}

func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.PingMessage,
		Delay:       btseWebsocketTimer,
	})
	return nil
}

// WsAuthenticate Send an authentication message to receive auth data
func (e *Exchange) WsAuthenticate(ctx context.Context) error {
	conn, err := e.Websocket.GetConnection(asset.Spot)
	if err != nil {
		return err
	}
	return e.wsAuthenticate(ctx, conn)
}

func (e *Exchange) wsAuthenticate(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	nonce := strconv.FormatInt(time.Now().UnixMilli(), 10)
	path := "/ws/spot" + nonce

	hmac, err := crypto.GetHMAC(crypto.HashSHA512_384, []byte(path), []byte(creds.Secret))
	if err != nil {
		return err
	}

	req := wsSub{
		Operation: "authKeyExpires",
		Arguments: []string{creds.Key, nonce, hex.EncodeToString(hmac)},
	}
	response, err := conn.SendMessageReturnResponse(ctx, request.Unset, "login", req)
	if err != nil {
		return err
	}
	var login WsLoginAcknowledgement
	if err := json.Unmarshal(response, &login); err != nil {
		return err
	}
	if !login.Success {
		return errWebsocketAuthenticationFailed
	}
	return nil
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "ORDER_INSERTED", "TRIGGER_INSERTED":
		return order.New, nil
	case "ORDER_CANCELLED":
		return order.Cancelled, nil
	case "ORDER_FULL_TRANSACTED":
		return order.Filled, nil
	case "ORDER_PARTIALLY_TRANSACTED":
		return order.PartiallyFilled, nil
	case "TRIGGER_ACTIVATED":
		return order.Active, nil
	case "INSUFFICIENT_BALANCE":
		return order.InsufficientBalance, nil
	case "MARKET_UNAVAILABLE":
		return order.MarketUnavailable, nil
	default:
		return order.UnknownStatus, errors.New(status + " not recognised as order status")
	}
}

func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	type Result map[string]any
	var result Result
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		if strings.Contains(string(respRaw), "connect success") {
			return nil
		}
		return err
	}
	if result == nil {
		return nil
	}

	if result["event"] != nil {
		event, ok := result["event"].(string)
		if !ok {
			return errors.New(e.Name + websocket.UnhandledMessage + string(respRaw))
		}
		switch event {
		case "subscribe":
			var subscribe WsSubscriptionAcknowledgement
			err = json.Unmarshal(respRaw, &subscribe)
			if err != nil {
				return err
			}
			if e.Verbose {
				log.Infof(log.WebsocketMgr, "%v subscribed to %v", e.Name, strings.Join(subscribe.Channel, ", "))
			}
		case "login":
			var login WsLoginAcknowledgement
			err = json.Unmarshal(respRaw, &login)
			if err != nil {
				return err
			}
			e.Websocket.SetCanUseAuthenticatedEndpoints(login.Success)
			conn.IncomingWithData("login", respRaw)
			if e.Verbose {
				log.Infof(log.WebsocketMgr, "%v websocket authenticated: %v", e.Name, login.Success)
			}
		default:
			return errors.New(e.Name + websocket.UnhandledMessage + string(respRaw))
		}
		return nil
	}

	topic, ok := result["topic"].(string)
	if !ok {
		return errors.New(e.Name + websocket.UnhandledMessage + string(respRaw))
	}
	switch {
	case topic == "notificationApi":
		var notification wsNotification
		err = json.Unmarshal(respRaw, &notification)
		if err != nil {
			return err
		}
		for i := range notification.Data {
			var oType order.Type
			var oSide order.Side
			var oStatus order.Status
			oType, err = order.StringToOrderType(notification.Data[i].Type)
			if err != nil {
				return err
			}

			oSide, err = order.StringToOrderSide(strings.ReplaceAll(notification.Data[i].OrderMode, "MODE_", ""))
			if err != nil {
				return err
			}
			oStatus, err = stringToOrderStatus(notification.Data[i].Status)
			if err != nil {
				return err
			}

			var p currency.Pair
			p, err = currency.NewPairFromString(notification.Data[i].Symbol)
			if err != nil {
				return err
			}

			var a asset.Item
			a, err = e.GetPairAssetType(p)
			if err != nil {
				return err
			}

			if err := e.Websocket.DataHandler.Send(ctx, &order.Detail{
				Price:        notification.Data[i].Price,
				Amount:       notification.Data[i].Size,
				TriggerPrice: notification.Data[i].TriggerPrice,
				Exchange:     e.Name,
				OrderID:      notification.Data[i].OrderID,
				Type:         oType,
				Side:         oSide,
				Status:       oStatus,
				AssetType:    a,
				Date:         notification.Data[i].Timestamp.Time(),
				Pair:         p,
			}); err != nil {
				return err
			}
		}
	case strings.Contains(topic, "tradeHistoryApi"):
		saveTradeData := e.IsSaveTradeDataEnabled()
		tradeFeed := e.IsTradeFeedEnabled()
		if !saveTradeData && !tradeFeed {
			return nil
		}

		var tradeHistory wsTradeHistory
		err = json.Unmarshal(respRaw, &tradeHistory)
		if err != nil {
			return err
		}
		var trades []trade.Data
		for x := range tradeHistory.Data {
			var p currency.Pair
			p, err = currency.NewPairFromString(tradeHistory.Data[x].Symbol)
			if err != nil {
				return err
			}
			var a asset.Item
			a, err = e.GetPairAssetType(p)
			if err != nil {
				return err
			}
			trades = append(trades, trade.Data{
				Timestamp:    tradeHistory.Data[x].Timestamp.Time().UTC(),
				CurrencyPair: p,
				AssetType:    a,
				Exchange:     e.Name,
				Price:        tradeHistory.Data[x].Price,
				Amount:       tradeHistory.Data[x].Size,
				Side:         tradeHistory.Data[x].Side,
				TID:          strconv.FormatInt(tradeHistory.Data[x].TID, 10),
			})
		}
		if tradeFeed {
			for i := range trades {
				if err := e.Websocket.DataHandler.Send(ctx, trades[i]); err != nil {
					return err
				}
			}
		}
		if saveTradeData {
			return trade.AddTradesToBuffer(trades...)
		}
	case strings.Contains(topic, "orderBookL2Api"): // TODO: Fix orderbook updates.
		var t wsOrderBook
		err = json.Unmarshal(respRaw, &t)
		if err != nil {
			return err
		}
		newOB := orderbook.Book{
			Bids: make(orderbook.Levels, 0, len(t.Data.BuyQuote)),
			Asks: make(orderbook.Levels, 0, len(t.Data.SellQuote)),
		}
		var price, amount float64
		for i := range t.Data.SellQuote {
			p := strings.ReplaceAll(t.Data.SellQuote[i].Price, ",", "")
			price, err = strconv.ParseFloat(p, 64)
			if err != nil {
				return err
			}
			a := strings.ReplaceAll(t.Data.SellQuote[i].Size, ",", "")
			amount, err = strconv.ParseFloat(a, 64)
			if err != nil {
				return err
			}
			if e.orderbookFilter(price, amount) {
				continue
			}
			newOB.Asks = append(newOB.Asks, orderbook.Level{
				Price:  price,
				Amount: amount,
			})
		}
		for j := range t.Data.BuyQuote {
			p := strings.ReplaceAll(t.Data.BuyQuote[j].Price, ",", "")
			price, err = strconv.ParseFloat(p, 64)
			if err != nil {
				return err
			}
			a := strings.ReplaceAll(t.Data.BuyQuote[j].Size, ",", "")
			amount, err = strconv.ParseFloat(a, 64)
			if err != nil {
				return err
			}
			if e.orderbookFilter(price, amount) {
				continue
			}
			newOB.Bids = append(newOB.Bids, orderbook.Level{
				Price:  price,
				Amount: amount,
			})
		}
		_, symbol, _ := strings.Cut(t.Topic, ":")
		symbol, _, _ = strings.Cut(symbol, currency.UnderscoreDelimiter)
		p, err := currency.NewPairFromString(symbol)
		if err != nil {
			return err
		}
		var a asset.Item
		a, err = e.GetPairAssetType(p)
		if err != nil {
			return err
		}
		newOB.Pair = p
		newOB.Asset = a
		newOB.Exchange = e.Name
		newOB.Asks.Reverse() // Reverse asks for correct alignment
		newOB.ValidateOrderbook = e.ValidateOrderbook
		newOB.LastUpdated = time.Now() // NOTE: Temp to fix test.
		err = e.Websocket.Orderbook.LoadSnapshot(ctx, &newOB)
		if err != nil {
			return err
		}
	default:
		return errors.New(e.Name + websocket.UnhandledMessage + string(respRaw))
	}

	return nil
}

// orderbookFilter is needed on book levels from this exchange as their data
// is incorrect
func (e *Exchange) orderbookFilter(price, amount float64) bool {
	// Amount filtering occurs when the amount exceeds the decimal returned.
	// e.g. {"price":"1.37","size":"0.00"} currency: SFI-ETH
	// Opted to not round up to 0.01 as this might skew calculations
	// more than removing from the books completely.

	// Price filtering occurs when we are deep in the bid book and there are
	// prices that are less than 4 decimal places
	// e.g. {"price":"0.0000","size":"14219"} currency: TRX-PAX
	// We cannot load a zero price and this will ruin calculations
	return price == 0 || amount == 0
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":     channelName,
		"isSymbolChannel": isSymbolChannel,
	}).Parse(subTplText)
}

// Subscribe subscribes to channels on the supplied managed connection.
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	req := wsSub{Operation: "subscribe"}
	for _, s := range subs {
		req.Arguments = append(req.Arguments, s.QualifiedChannel)
	}
	err := conn.SendJSONMessage(ctx, request.Unset, req)
	if err == nil {
		err = e.Websocket.AddSuccessfulSubscriptions(conn, subs...)
	}
	return err
}

// Unsubscribe removes channels from the supplied managed connection.
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	req := wsSub{Operation: "unsubscribe"}
	for _, s := range subs {
		req.Arguments = append(req.Arguments, s.QualifiedChannel)
	}
	err := conn.SendJSONMessage(ctx, request.Unset, req)
	if err == nil {
		err = e.Websocket.RemoveSubscriptions(conn, subs...)
	}
	return err
}

// channelName returns the correct channel name for the asset
func channelName(s *subscription.Subscription) string {
	if name, ok := subscriptionNames[s.Channel]; ok {
		return name
	}
	panic("Channel not supported: " + s.Channel)
}

// isSymbolChannel returns if the channel expects receive a symbol
func isSymbolChannel(s *subscription.Subscription) bool {
	return s.Channel != subscription.MyTradesChannel
}

const subTplText = `
{{- with $name := channelName $.S }}
	{{ range $asset, $pairs := $.AssetPairs }}
		{{- if isSymbolChannel $.S }}
			{{- range $p := $pairs -}}
				{{- $name -}} : {{- $p -}}
				{{- $.PairSeparator }}
			{{- end }}
		{{- else -}}
			{{ $name }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- end }}
`
