package poloniex

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
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

var onceOrderbook map[currency.Pair]struct{}

func setupPingHandler(conn websocket.Connection) {
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"event": "ping"}`),
		Delay:       time.Second * 15,
	})
}

// wsConnect initiates a websocket connection
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}
	setupPingHandler(conn)
	onceOrderbook = make(map[currency.Pair]struct{})
	return nil
}

func (e *Exchange) wsAuthConn(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}
	setupPingHandler(conn)
	return nil
}

// authenticateSpotAuthConn authenticates a futures websocket connection
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
	data, err := conn.SendMessageReturnResponse(ctx, request.UnAuth, channelAuth, auth)
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

func (e *Exchange) wsHandleData(_ context.Context, conn websocket.Connection, respRaw []byte) error {
	var result SubscriptionResponse
	if err := json.Unmarshal(respRaw, &result); err != nil {
		return err
	}
	if result.ID != "" {
		if !conn.IncomingWithData(result.ID, respRaw) {
			return fmt.Errorf("could not match trade response with ID: %s Event: %s ", result.ID, result.Event)
		}
		return nil
	}
	if result.Event != "" {
		switch result.Event {
		case "pong", "subscribe":
		case "error":
			if result.Message == "user must be authenticated!" {
				e.Websocket.SetCanUseAuthenticatedEndpoints(false)
				log.Debugf(log.ExchangeSys, "authenticated websocket disabled: %s", string(respRaw))
			}
			fallthrough
		default:
			log.Debugf(log.ExchangeSys, "Unexpected event message %s", string(respRaw))
		}
		return nil
	}
	switch result.Channel {
	case channelAuth:
		return conn.RequireMatchWithData("auth", respRaw)
	case channelSymbols:
		var response [][]WsSymbol
		return e.processResponse(&result, &response)
	case channelCurrencies:
		var response [][]WsCurrency
		return e.processResponse(&result, &response)
	case channelExchange:
		var response WsExchangeStatus
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
		return e.processBalance(&result)
	default:
		if strings.HasPrefix(result.Channel, channelCandles) {
			return e.processCandlestickData(&result)
		}
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
		return fmt.Errorf("%s unhandled message: %s", e.Name, string(respRaw))
	}
}

func (e *Exchange) processBalance(result *SubscriptionResponse) error {
	var resp []WsTradeBalance
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	accountChanges := make([]account.Change, len(resp))
	for x := range resp {
		accountChanges[x] = account.Change{
			Account:   resp[x].AccountType,
			AssetType: stringToAccountType(resp[x].AccountType),
			Balance: &account.Balance{
				Hold:      resp[x].Hold.Float64(),
				Total:     resp[x].Available.Float64(),
				UpdatedAt: resp[x].Timestamp.Time(),
				Currency:  currency.NewCode(resp[x].Currency),
				Free:      resp[x].Available.Float64() - resp[x].Hold.Float64(),
			},
		}
	}
	e.Websocket.DataHandler <- accountChanges
	return nil
}

func (e *Exchange) processOrders(result *SubscriptionResponse) error {
	response := []WebsocketTradeOrder{}
	if err := json.Unmarshal(result.Data, &response); err != nil {
		return err
	}
	orderDetails := make([]order.Detail, len(response))
	for x := range response {
		oType, err := order.StringToOrderType(response[x].Type)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(response[x].Side)
		if err != nil {
			return err
		}
		oStatus, err := order.StringToOrderStatus(response[x].State)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(response[x].Symbol)
		if err != nil {
			return err
		}
		orderDetails[x] = order.Detail{
			Price:           response[x].Price.Float64(),
			Amount:          response[x].Quantity.Float64(),
			QuoteAmount:     response[x].OrderAmount.Float64(),
			ExecutedAmount:  response[x].FilledAmount.Float64(),
			RemainingAmount: response[x].OrderAmount.Float64() - response[x].FilledAmount.Float64(),
			Fee:             response[x].TradeFee.Float64(),
			FeeAsset:        currency.NewCode(response[x].FeeCurrency),
			Exchange:        e.Name,
			OrderID:         response[x].OrderID,
			ClientOrderID:   response[x].ClientOrderID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       stringToAccountType(response[x].AccountType),
			Date:            response[x].CreateTime.Time(),
			LastUpdated:     response[x].TradeTime.Time(),
			Pair:            cp,
			Trades: []order.TradeHistory{
				{
					Price:     response[x].TradePrice.Float64(),
					Amount:    response[x].TradeQty.Float64(),
					Fee:       response[x].TradeFee.Float64(),
					Exchange:  e.Name,
					TID:       response[x].TradeID,
					Type:      oType,
					Side:      oSide,
					Timestamp: response[x].Timestamp.Time(),
					FeeAsset:  response[x].FeeCurrency,
					Total:     response[x].Quantity.Float64(),
				},
			},
		}
	}
	e.Websocket.DataHandler <- orderDetails
	return nil
}

func (e *Exchange) processBooks(result *SubscriptionResponse) error {
	var resp []WsBook
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	for x := range resp {
		cp, err := currency.NewPairFromString(resp[x].Symbol)
		if err != nil {
			return err
		}
		_, okay := onceOrderbook[cp]
		if !okay {
			if onceOrderbook == nil {
				onceOrderbook = make(map[currency.Pair]struct{})
			}
			var (
				orderbooks *orderbook.Book
				err        error
			)
			orderbooks, err = e.UpdateOrderbook(context.Background(), cp, asset.Spot)
			if err != nil {
				return err
			}
			if err := e.Websocket.Orderbook.LoadSnapshot(orderbooks); err != nil {
				return err
			}
			onceOrderbook[cp] = struct{}{}
		}
		update := orderbook.Update{
			Pair:       cp,
			UpdateTime: resp[x].Timestamp.Time(),
			UpdateID:   resp[x].ID,
			Asset:      asset.Spot,
			Action:     orderbook.UpdateOrInsertAction,
		}
		for i := range resp[x].Asks {
			if resp[x].Asks[i][1].Float64() <= 0 {
				continue
			}
			update.Asks = append(update.Asks, orderbook.Level{
				Price:  resp[x].Asks[i][0].Float64(),
				Amount: resp[x].Asks[i][1].Float64(),
			})
		}
		for i := range resp[x].Bids {
			if resp[x].Bids[i][1].Float64() <= 0 {
				continue
			}
			update.Bids = append(update.Bids, orderbook.Level{
				Price:  resp[x].Bids[i][0].Float64(),
				Amount: resp[x].Bids[i][1].Float64(),
			})
		}
		if err := e.Websocket.Orderbook.Update(&update); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processBooksLevel2(result *SubscriptionResponse) error {
	var resp []WsBook
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	for x := range resp {
		cp, err := currency.NewPairFromString(resp[x].Symbol)
		if err != nil {
			return err
		}
		var asks orderbook.Levels
		var bids orderbook.Levels
		for i := range resp[x].Asks {
			if resp[x].Asks[i][1].Float64() <= 0 {
				continue
			}
			asks = append(asks, orderbook.Level{
				Price:  resp[x].Asks[i][0].Float64(),
				Amount: resp[x].Asks[i][1].Float64(),
			})
		}
		for i := range resp[x].Bids {
			if resp[x].Bids[i][1].Float64() <= 0 {
				continue
			}
			bids = append(bids, orderbook.Level{
				Price:  resp[x].Bids[i][0].Float64(),
				Amount: resp[x].Bids[i][1].Float64(),
			})
		}

		if result.Action == "snapshot" {
			if err := e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
				Exchange:     e.Name,
				Pair:         cp,
				Asset:        asset.Spot,
				Asks:         asks,
				Bids:         bids,
				LastUpdateID: resp[x].LastID,
				LastUpdated:  resp[x].Timestamp.Time(),
			}); err != nil {
				return err
			}
			continue
		}

		if err := e.Websocket.Orderbook.Update(&orderbook.Update{
			Pair:       cp,
			UpdateTime: resp[x].Timestamp.Time(),
			UpdateID:   resp[x].ID,
			Asset:      asset.Spot,
			Action:     orderbook.UpdateOrInsertAction,
			Asks:       asks,
			Bids:       bids,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processTicker(result *SubscriptionResponse) error {
	var resp []WsTicker
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	tickerData := make([]ticker.Price, len(resp))
	for x := range resp {
		cp, err := currency.NewPairFromString(resp[x].Symbol)
		if err != nil {
			return err
		}
		tickerData[x] = ticker.Price{
			Last:         resp[x].MarkPrice.Float64(),
			High:         resp[x].High.Float64(),
			Low:          resp[x].Low.Float64(),
			Volume:       resp[x].Quantity.Float64(),
			QuoteVolume:  resp[x].Amount.Float64(),
			Open:         resp[x].Open.Float64(),
			Close:        resp[x].Close.Float64(),
			Pair:         cp,
			AssetType:    asset.Spot,
			ExchangeName: e.Name,
			LastUpdated:  resp[x].Timestamp.Time(),
		}
	}
	e.Websocket.DataHandler <- tickerData
	return nil
}

func (e *Exchange) processTrades(result *SubscriptionResponse) error {
	var resp []WsTrade
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp))
	for x := range resp {
		cp, err := currency.NewPairFromString(resp[x].Symbol)
		if err != nil {
			return err
		}
		trades[x] = trade.Data{
			TID:          resp[x].ID,
			Exchange:     e.Name,
			CurrencyPair: cp,
			Price:        resp[x].Price.Float64(),
			Amount:       resp[x].Amount.Float64(),
			Timestamp:    resp[x].Timestamp.Time(),
		}
	}
	return trade.AddTradesToBuffer(trades...)
}

func (e *Exchange) processCandlestickData(result *SubscriptionResponse) error {
	var resp []WsCandles
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		return err
	}
	candles := make([]websocket.KlineData, len(resp))
	for x := range resp {
		cp, err := currency.NewPairFromString(resp[x].Symbol)
		if err != nil {
			return err
		}
		candles[x] = websocket.KlineData{
			Pair:       cp,
			Exchange:   e.Name,
			Timestamp:  resp[x].Timestamp.Time(),
			StartTime:  resp[x].StartTime.Time(),
			CloseTime:  resp[x].CloseTime.Time(),
			OpenPrice:  resp[x].Open.Float64(),
			ClosePrice: resp[x].Close.Float64(),
			HighPrice:  resp[x].High.Float64(),
			LowPrice:   resp[x].Low.Float64(),
			Volume:     resp[x].Quantity.Float64(),
		}
	}
	e.Websocket.DataHandler <- candles
	return nil
}

func (e *Exchange) processResponse(result *SubscriptionResponse, instance any) error {
	if err := json.Unmarshal(result.Data, instance); err != nil {
		return err
	}
	fullResp := result.GetWsResponse()
	fullResp.Data = instance
	e.Websocket.DataHandler <- fullResp
	return nil
}

func (e *Exchange) handleSubscription(operation string, s *subscription.Subscription) (SubscriptionPayload, error) {
	pairFormat, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return SubscriptionPayload{}, err
	}
	sp := SubscriptionPayload{
		Event:   operation,
		Channel: []string{strings.ToLower(s.QualifiedChannel)},
	}
	if len(s.Pairs) != 0 {
		sp.Symbols = s.Pairs.Format(pairFormat).Strings()
	}

	switch s.Channel {
	case channelBooks:
		sp.Depth = int64(s.Levels)
	case channelCurrencies:
		for _, p := range s.Pairs {
			if !slices.Contains(sp.Currencies, p.Base.String()) {
				sp.Currencies = append(sp.Currencies, p.Base.String())
			}
		}
	case subscription.MyOrdersChannel:
		sp.Symbols = []string{"all"}
	}
	return sp, nil
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

func channelToIntervalSplit(intervalString string) (string, kline.Interval, error) {
	splits := strings.Split(intervalString, "_")
	length := len(splits)
	intervalValue, err := stringToInterval(strings.Join(splits[length-2:], "_"))
	return strings.Join(splits[:length-2], "_"), intervalValue, err
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, err := subs.ExpandTemplates(e)
	if err != nil {
		return err
	}
	return e.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error {
		return e.manageSubs(ctx, conn, "subscribe", l)
	}, 1)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, err := subs.ExpandTemplates(e)
	if err != nil {
		return err
	}
	return e.ParallelChanOp(ctx, subs, func(ctx context.Context, l subscription.List) error {
		return e.manageSubs(ctx, conn, "unsubscribe", l)
	}, 1)
}

func (e *Exchange) manageSubs(ctx context.Context, conn websocket.Connection, operation string, subs subscription.List) error {
	var errs error
	for _, s := range subs {
		if strings.HasSuffix(conn.GetURL(), "private") != s.Authenticated {
			continue
		}
		payload, err := e.handleSubscription(operation, s)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if err := conn.SendJSONMessage(ctx, request.UnAuth, payload); err != nil {
			errs = common.AppendError(errs, err)
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
