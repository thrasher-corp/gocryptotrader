package poloniex

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
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
	poloniexWebsocketAddress        = "wss://ws.poloniex.com/ws/public"
	poloniexPrivateWebsocketAddress = "wss://ws.poloniex.com/ws/private"

	cnlExchange = "exchange"
	cnlAuth     = "auth"

	// Public channels
	cnlSymbols    = "symbols"
	cnlCurrencies = "currencies"
	cnlCandles    = "candles"
	cnlTrades     = "trades"
	cnlTicker     = "ticker"
	cnlBooks      = "book"
	cnlBookLevel2 = "book_lv2"

	// Authenticated channels
	cnlOrders   = "orders"
	cnlBalances = "balances"
)

var defaultSubscriptions = []string{
	cnlCandles,
	cnlTrades,
	cnlTicker,
	cnlBookLevel2,
}

var onceOrderbook map[currency.Pair]struct{}

// WsConnect initiates a websocket connection
func (e *Exchange) WsConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}
	conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"event": "ping"}`),
		Delay:       time.Second * 15,
	})
	onceOrderbook = make(map[currency.Pair]struct{})
	return nil
}

func (e *Exchange) wsAuthConn(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}

	conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"event": "ping"}`),
		Delay:       time.Second * 15,
	})
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
		Channel: []string{cnlAuth},
		Params: AuthRequest{
			Key:             creds.Key,
			SignatureMethod: "hmacSHA256",
			SignTimestamp:   timestamp.UnixMilli(),
			Signature:       base64.StdEncoding.EncodeToString(hmac),
		},
	}
	data, err := conn.SendMessageReturnResponse(ctx, request.UnAuth, cnlAuth, auth)
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
	case cnlAuth:
		if !conn.IncomingWithData("auth", respRaw) {
			return fmt.Errorf("could not match data with %s %s", "auth", respRaw)
		}
		return nil
	case cnlSymbols:
		var response [][]WsSymbol
		return e.processResponse(&result, &response)
	case cnlCurrencies:
		var response [][]WsCurrency
		return e.processResponse(&result, &response)
	case cnlExchange:
		var response WsExchangeStatus
		return e.processResponse(&result, &response)
	case cnlTrades:
		return e.processTrades(&result)
	case cnlTicker:
		return e.processTicker(&result)
	case cnlBooks, cnlBookLevel2:
		return e.processBooks(&result)
	case cnlOrders:
		return e.processOrders(&result)
	case cnlBalances:
		return e.processBalance(&result)
	default:
		if strings.HasPrefix(result.Channel, cnlCandles) {
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
			Action:     orderbook.UpdateAction,
			Asks:       make(orderbook.Levels, len(resp[x].Asks)),
			Bids:       make(orderbook.Levels, len(resp[x].Bids)),
		}
		for i := range resp[x].Asks {
			update.Asks[i].Price = resp[x].Asks[i][0].Float64()
			update.Asks[i].Amount = resp[x].Asks[i][1].Float64()
		}
		for i := range resp[x].Bids {
			update.Bids[i].Price = resp[x].Bids[i][0].Float64()
			update.Bids[i].Amount = resp[x].Bids[i][1].Float64()
		}
		if err := e.Websocket.Orderbook.Update(&update); err != nil {
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

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (e *Exchange) GenerateDefaultSubscriptions(auth bool) (subscription.List, error) {
	enabledCurrencies, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var channels []string
	if auth && e.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels, []string{cnlOrders, cnlBalances}...)
	} else {
		channels = defaultSubscriptions
	}
	subscriptions := make(subscription.List, 0, 6*len(enabledCurrencies))
	for i := range channels {
		switch channels[i] {
		case cnlSymbols, cnlTrades, cnlTicker, cnlBooks, cnlBookLevel2:
			var params map[string]any
			if channels[i] == cnlBooks {
				params = map[string]any{
					"depth": 20,
				}
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Pairs:   enabledCurrencies,
				Channel: channels[i],
				Params:  params,
			})
		case cnlCurrencies:
			currencyMaps := make(map[currency.Code]struct{})
			for x := range enabledCurrencies {
				_, okay := currencyMaps[enabledCurrencies[x].Base]
				if !okay {
					subscriptions = append(subscriptions, &subscription.Subscription{
						Channel: channels[i],
						Pairs:   []currency.Pair{{Base: enabledCurrencies[x].Base}},
					})
					currencyMaps[enabledCurrencies[x].Base] = struct{}{}
				}
				_, okay = currencyMaps[enabledCurrencies[x].Quote]
				if !okay {
					subscriptions = append(subscriptions, &subscription.Subscription{
						Channel: channels[i],
						Pairs:   []currency.Pair{{Base: enabledCurrencies[x].Quote}},
					})
					currencyMaps[enabledCurrencies[x].Quote] = struct{}{}
				}
			}
		case cnlCandles:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[i],
				Pairs:   enabledCurrencies,
				Params: map[string]any{
					"interval": kline.FiveMin,
				},
			})
		case cnlOrders, cnlBalances, cnlExchange:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[i],
			})
		}
	}
	return subscriptions, nil
}

func (e *Exchange) handleSubscriptions(operation string, subscs subscription.List) ([]SubscriptionPayload, error) {
	pairFormat, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	payloads := []SubscriptionPayload{}
	for x := range subscs {
		switch subscs[x].Channel {
		case cnlSymbols, cnlTrades, cnlTicker, cnlBooks, cnlBookLevel2:
			sp := SubscriptionPayload{
				Event:   operation,
				Channel: []string{subscs[x].Channel},
				Symbols: subscs[x].Pairs.Format(pairFormat).Strings(),
			}
			if subscs[x].Channel == cnlBooks {
				depth, okay := subscs[x].Params["depth"]
				if okay {
					sp.Depth, _ = depth.(int64)
				}
			}
			payloads = append(payloads, sp)
		case cnlCurrencies:
			sp := SubscriptionPayload{
				Event:      operation,
				Channel:    []string{subscs[x].Channel},
				Currencies: []string{},
			}
			for _, p := range subscs[x].Pairs {
				if !slices.Contains(sp.Currencies, p.Base.String()) {
					sp.Currencies = append(sp.Currencies, p.Base.String())
				}
			}
			payloads = append(payloads, sp)
		case cnlCandles:
			interval, okay := subscs[x].Params["interval"].(kline.Interval)
			if !okay {
				interval = kline.FiveMin
			}
			intervalString, err := intervalToString(interval)
			if err != nil {
				return nil, err
			}
			channelName := fmt.Sprintf("%s_%s", subscs[x].Channel, strings.ToLower(intervalString))
			payloads = append(payloads, SubscriptionPayload{
				Event:   operation,
				Channel: []string{channelName},
				Symbols: subscs[x].Pairs.Format(pairFormat).Strings(),
			})
		case cnlOrders:
			payloads = append(payloads, SubscriptionPayload{
				Event:   operation,
				Channel: []string{subscs[x].Channel},
				Symbols: []string{"all"},
			})
		case cnlBalances, cnlExchange:
			payloads = append(payloads, SubscriptionPayload{
				Event:   operation,
				Channel: []string{subscs[x].Channel},
			})
		default:
			return nil, subscription.ErrNotSupported
		}
	}
	return payloads, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	payloads, err := e.handleSubscriptions("subscribe", subs)
	if err != nil {
		return err
	}
	for i := range payloads {
		if err := conn.SendJSONMessage(ctx, request.UnAuth, payloads[i]); err != nil {
			return err
		}
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, subs...)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, unsub subscription.List) error {
	payloads, err := e.handleSubscriptions("unsubscribe", unsub)
	if err != nil {
		return err
	}
	for i := range payloads {
		switch payloads[i].Channel[0] {
		case cnlBalances, cnlOrders:
			if e.IsWebsocketAuthenticationSupported() && e.Websocket.CanUseAuthenticatedEndpoints() {
				if err := conn.SendJSONMessage(ctx, request.UnAuth, payloads[i]); err != nil {
					return err
				}
			}
		default:
			if err := conn.SendJSONMessage(ctx, request.UnAuth, payloads[i]); err != nil {
				return err
			}
		}
	}
	return e.Websocket.RemoveSubscriptions(conn, unsub...)
}
