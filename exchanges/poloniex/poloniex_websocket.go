package poloniex

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
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
	websocketURL        = "wss://ws.poloniex.com/ws/public"
	privateWebsocketURL = "wss://ws.poloniex.com/ws/private"

	channelExchange = "exchange"
	channelAuth     = "auth"

	// Public channels
	channelSymbols    = "symbols"
	channelCurrencies = "currencies"
	channelCandles    = "candles"
	channelTrades     = "trades"
	channelTicker     = "ticker"
	channelBooks      = "book"
	channelBookLevel2 = "book_lv2"

	// Authenticated channels
	channelOrders   = "orders"
	channelBalances = "balances"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.FiveMin},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
}

var defaultSpotPrivateSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
}

var subscriptionNames = map[string]string{
	subscription.CandlesChannel:   channelCandles,
	subscription.AllTradesChannel: channelTrades,
	subscription.TickerChannel:    channelTicker,
	subscription.OrderbookChannel: channelBookLevel2,
	subscription.MyOrdersChannel:  channelOrders,
	subscription.MyAccountChannel: channelBalances,
}

func setupPingHandler(conn websocket.Connection) {
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"event": "ping"}`),
		Delay:       time.Second * 15,
	})
}

// wsConnect checks if websocket is enabled and initiates a websocket connection
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}
	setupPingHandler(conn)
	return nil
}

// authenticateSpotAuthConn authenticates a spot websocket connection
func (e *Exchange) authenticateSpotAuthConn(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	timestamp := time.Now()
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		fmt.Appendf(nil, "GET\n/ws\nsignTimestamp=%d", timestamp.UnixMilli()),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}
	auth := &struct {
		Event   string      `json:"event"`
		Channel []string    `json:"channel"`
		Params  AuthRequest `json:"params"`
	}{
		Event:   "subscribe",
		Channel: []string{channelAuth},
		Params: AuthRequest{
			Key:             creds.Key,
			SignatureMethod: "hmacSHA256",
			SignTimestamp:   timestamp.UnixMilli(),
			Signature:       base64.StdEncoding.EncodeToString(hmac),
		},
	}
	data, err := conn.SendMessageReturnResponse(ctx, fWebsocketPrivateEPL, channelAuth, auth)
	if err != nil {
		return err
	}
	var resp *AuthenticationResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if !resp.Data.Success {
		return fmt.Errorf("authentication failed with status code: %s", resp.Data.Message)
	}
	return nil
}

func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var result SubscriptionResponse
	if err := json.Unmarshal(respRaw, &result); err != nil {
		return err
	}
	if result.ID != "" {
		return conn.RequireMatchWithData(result.ID, respRaw)
	} else if result.Event != "" {
		switch result.Event {
		case "pong":
			return nil
		case "subscribe", "unsubscribe", "error":
			return conn.RequireMatchWithData("subscription", respRaw)
		default:
			e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
			log.Debugf(log.ExchangeSys, "Unexpected event message spot %s", string(respRaw))
		}
		return nil
	}
	switch result.Channel {
	case channelAuth:
		return conn.RequireMatchWithData(channelAuth, respRaw)
	case channelSymbols:
		var response []*WsSymbol
		return e.processResponse(&result, &response)
	case channelCurrencies:
		var response []*WsCurrency
		return e.processResponse(&result, &response)
	case channelExchange:
		var response []*WsExchangeStatus
		return e.processResponse(&result, &response)
	case channelTrades:
		return e.processTrades(&result)
	case channelTicker:
		return e.processTicker(&result)
	case channelBookLevel2:
		return e.processBooksLevel2(&result)
	case channelBooks:
		return e.processBooks(&result)
	case channelOrders:
		return e.processOrders(&result)
	case channelBalances:
		return e.processBalance(ctx, &result)
	default:
		if strings.HasPrefix(result.Channel, channelCandles) {
			return e.processCandlestickData(&result)
		}
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
		return fmt.Errorf("%s unhandled message: %s", e.Name, string(respRaw))
	}
}

func (e *Exchange) processBalance(ctx context.Context, result *SubscriptionResponse) error {
	var resp []*WsTradeBalance
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	subAccts := accounts.SubAccounts{}
	for _, r := range resp {
		subAcct := accounts.NewSubAccount(stringToAccountType(r.AccountType), r.AccountID)
		subAcct.Balances.Set(r.Currency, accounts.Balance{
			Currency:  r.Currency,
			Hold:      r.Hold.Float64(),
			Total:     r.Available.Float64(),
			UpdatedAt: r.Timestamp.Time(),
			Free:      r.Available.Float64() - r.Hold.Float64(),
		})
		subAccts = subAccts.Merge(subAcct)
	}
	if err := e.Accounts.Save(ctx, subAccts, true); err != nil {
		return err
	}
	e.Websocket.DataHandler <- subAccts
	return nil
}

func (e *Exchange) processOrders(result *SubscriptionResponse) error {
	response := []*WebsocketTradeOrder{}
	if err := json.Unmarshal(result.Data, &response); err != nil {
		return err
	}
	orderDetails := make([]order.Detail, len(response))
	for x, r := range response {
		oStatus, err := order.StringToOrderStatus(r.State)
		if err != nil {
			return err
		}
		oType, err := order.StringToOrderType(r.Type)
		if err != nil {
			return err
		}
		orderDetails[x] = order.Detail{
			Price:           r.Price.Float64(),
			Amount:          r.Quantity.Float64(),
			QuoteAmount:     r.OrderAmount.Float64(),
			ExecutedAmount:  r.FilledAmount.Float64(),
			RemainingAmount: r.Quantity.Float64() - r.FilledQuantity.Float64(),
			Fee:             r.TradeFee.Float64(),
			FeeAsset:        r.FeeCurrency,
			Exchange:        e.Name,
			OrderID:         r.OrderID,
			ClientOrderID:   r.ClientOrderID,
			Type:            oType,
			Side:            r.Side,
			Status:          oStatus,
			AssetType:       stringToAccountType(r.AccountType),
			Date:            r.CreateTime.Time(),
			LastUpdated:     r.TradeTime.Time(),
			Pair:            r.Symbol,
			Trades: []order.TradeHistory{
				{
					Price:     r.TradePrice.Float64(),
					Amount:    r.TradeQty.Float64(),
					Fee:       r.TradeFee.Float64(),
					Exchange:  e.Name,
					TID:       r.TradeID,
					Type:      oType,
					Side:      r.Side,
					Timestamp: r.Timestamp.Time(),
					FeeAsset:  r.FeeCurrency.String(),
					Total:     r.Quantity.Float64(),
				},
			},
		}
	}
	e.Websocket.DataHandler <- orderDetails
	return nil
}

func (e *Exchange) processBooks(result *SubscriptionResponse) error {
	var resp []*WsBook
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	for _, r := range resp {
		if err := e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Pair:         r.Symbol,
			Exchange:     e.Name,
			LastUpdateID: r.ID,
			Asset:        asset.Spot,
			LastUpdated:  r.CreateTime.Time(),
			LastPushed:   r.Timestamp.Time(),
			Asks:         r.Asks.Levels(),
			Bids:         r.Bids.Levels(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processBooksLevel2(result *SubscriptionResponse) error {
	var resp []*WsBook
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	for _, r := range resp {
		if result.Action == "snapshot" {
			if err := e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
				Exchange:     e.Name,
				Pair:         r.Symbol,
				Asset:        asset.Spot,
				Asks:         r.Asks.Levels(),
				Bids:         r.Bids.Levels(),
				LastUpdateID: r.LastID,
				LastUpdated:  r.Timestamp.Time(),
			}); err != nil {
				return err
			}
			continue
		}

		if err := e.Websocket.Orderbook.Update(&orderbook.Update{
			Pair:       r.Symbol,
			UpdateTime: r.Timestamp.Time(),
			UpdateID:   r.ID,
			Asset:      asset.Spot,
			Action:     orderbook.UpdateAction,
			Asks:       r.Asks.Levels(),
			Bids:       r.Bids.Levels(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processTicker(result *SubscriptionResponse) error {
	var resp []*WsTicker
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	tickerData := make([]ticker.Price, len(resp))
	for x, r := range resp {
		tickerData[x] = ticker.Price{
			MarkPrice:    r.MarkPrice.Float64(),
			High:         r.High.Float64(),
			Low:          r.Low.Float64(),
			Volume:       r.Quantity.Float64(),
			QuoteVolume:  r.Amount.Float64(),
			Open:         r.Open.Float64(),
			Close:        r.Close.Float64(),
			Pair:         r.Symbol,
			AssetType:    asset.Spot,
			ExchangeName: e.Name,
			LastUpdated:  r.Timestamp.Time(),
		}
	}
	e.Websocket.DataHandler <- tickerData
	return nil
}

func (e *Exchange) processTrades(result *SubscriptionResponse) error {
	var resp []*WsTrade
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp))
	for x, r := range resp {
		trades[x] = trade.Data{
			TID:          r.ID.String(),
			Exchange:     e.Name,
			CurrencyPair: r.Symbol,
			Side:         r.TakerSide,
			Price:        r.Price.Float64(),
			Timestamp:    r.Timestamp.Time(),
			Amount:       r.Quantity.Float64(),
		}
	}
	return trade.AddTradesToBuffer(trades...)
}

func (e *Exchange) processCandlestickData(result *SubscriptionResponse) error {
	var resp []*WsCandles
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	candles := make([]websocket.KlineData, len(resp))
	for x, r := range resp {
		candles[x] = websocket.KlineData{
			Pair:       r.Symbol,
			Exchange:   e.Name,
			Timestamp:  r.Timestamp.Time(),
			StartTime:  r.StartTime.Time(),
			CloseTime:  r.CloseTime.Time(),
			OpenPrice:  r.Open.Float64(),
			ClosePrice: r.Close.Float64(),
			HighPrice:  r.High.Float64(),
			LowPrice:   r.Low.Float64(),
			Volume:     r.Quantity.Float64(),
		}
	}
	e.Websocket.DataHandler <- candles
	return nil
}

func (e *Exchange) processResponse(result *SubscriptionResponse, instance any) error {
	if err := json.Unmarshal(result.Data, instance); err != nil {
		return err
	}
	e.Websocket.DataHandler <- instance
	return nil
}

func (e *Exchange) handleSubscription(operation string, s *subscription.Subscription) (*SubscriptionPayload, error) {
	pairFormat, err := e.GetPairFormat(s.Asset, true)
	if err != nil {
		return nil, err
	}
	input := &SubscriptionPayload{
		Event:   operation,
		Channel: []string{strings.ToLower(s.QualifiedChannel)},
	}

	switch {
	case s.Asset == asset.Futures && s.QualifiedChannel == channelFuturesAccount,
		s.Asset == asset.Spot && (s.QualifiedChannel == channelBalances || s.QualifiedChannel == channelOrders):
	case len(s.Pairs) != 0:
		input.Symbols = s.Pairs.Format(pairFormat).Strings()
	}

	switch s.Channel {
	case subscription.OrderbookChannel, channelBooks:
		input.Depth = int64(s.Levels) // supported orderbook levels are 5, 10 and 20
	case channelCurrencies:
		for _, p := range s.Pairs {
			if !slices.Contains(input.Currencies, p.Base.String()) {
				input.Currencies = append(input.Currencies, p.Base.String())
			}
		}
	case subscription.MyOrdersChannel:
		if s.Asset == asset.Spot && len(input.Symbols) == 0 {
			input.Symbols = []string{"all"}
		}
	}
	return input, nil
}

func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

func (e *Exchange) generatePrivateSubscriptions() (subscription.List, error) {
	return defaultSpotPrivateSubscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		"channelName": channelName,
		"interval":    intervalToString,
	}).Parse(subTplText)
}

func channelName(s *subscription.Subscription) string {
	switch s.Asset {
	case asset.Futures:
		if name, ok := futuresSubscriptionNames[s.Channel]; ok {
			return name
		}
	case asset.Spot:
		if name, ok := subscriptionNames[s.Channel]; ok {
			return name
		}
	}
	return s.Channel
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, err := subs.ExpandTemplates(e)
	if err != nil {
		return err
	}
	return e.manageSubs(ctx, "subscribe", conn, subs, &e.spotSubMtx)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, err := subs.ExpandTemplates(e)
	if err != nil {
		return err
	}
	return e.manageSubs(ctx, "unsubscribe", conn, subs, &e.spotSubMtx)
}

func (e *Exchange) manageSubs(ctx context.Context, operation string, conn websocket.Connection, subscs subscription.List, sendMsgLock *sync.Mutex) error {
	var errs error
	for _, s := range subscs {
		payload, err := e.handleSubscription(operation, s)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		epl := sWebsocketPublicEPL
		switch {
		case s.Asset == asset.Futures && s.Authenticated:
			epl = fWebsocketPrivateEPL
		case s.Asset == asset.Futures:
			epl = fWebsocketPublicEPL
		case s.Authenticated:
			epl = sWebsocketPrivateEPL
		}

		sendMsgLock.Lock()
		result, err := conn.SendMessageReturnResponse(ctx, epl, "subscription", &payload)
		sendMsgLock.Unlock()
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w %s channel %q: %w", websocket.ErrSubscriptionFailure, operation, payload.Channel[0], err))
			continue
		}
		var subscriptionResponse *SubscriptionResponse
		if err := json.Unmarshal(result, &subscriptionResponse); err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if subscriptionResponse.Event == "error" {
			errs = common.AppendError(errs, fmt.Errorf("%w %s channel %q", websocket.ErrSubscriptionFailure, operation, payload.Channel[0]))
			continue
		}
		if operation == "subscribe" {
			err = e.Websocket.AddSuccessfulSubscriptions(conn, s)
		} else {
			err = e.Websocket.RemoveSubscriptions(conn, s)
		}
		errs = common.AppendError(errs, err)
	}
	return errs
}

const subTplText = `
{{- range $asset, $pairs := $.AssetPairs -}}
	{{- channelName $.S -}}
	{{- if eq $.S.Channel "candles" -}}_{{- interval $.S.Interval | lower -}}{{- end -}}
	{{ $.AssetSeparator }}
{{- end -}}
`
