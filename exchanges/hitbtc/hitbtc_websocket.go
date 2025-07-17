package hitbtc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
	hitbtcWebsocketAddress = "wss://api.hitbtc.com/api/2/ws"
	rpcVersion             = "2.0"
	errAuthFailed          = 1002
)

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    "Ticker",
	subscription.OrderbookChannel: "Orderbook",
	subscription.CandlesChannel:   "Candles",
	subscription.AllTradesChannel: "Trades",
	subscription.MyAccountChannel: "Reports",
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.ThirtyMin, Levels: 100},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel, Levels: 100},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
}

// WsConnect starts a new connection with the websocket API
func (e *Exchange) WsConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}

	e.Websocket.Wg.Add(1)
	go e.wsReadData()

	if e.Websocket.CanUseAuthenticatedEndpoints() {
		err = e.wsLogin(ctx)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", e.Name, err)
		}
	}

	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData() {
	defer e.Websocket.Wg.Done()

	for {
		resp := e.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}

		err := e.wsHandleData(resp.Raw)
		if err != nil {
			e.Websocket.DataHandler <- err
		}
	}
}

func (e *Exchange) wsGetTableName(respRaw []byte) (string, error) {
	var init capture
	err := json.Unmarshal(respRaw, &init)
	if err != nil {
		return "", err
	}
	if init.Error.Code == errAuthFailed {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	if init.ID > 0 {
		if e.Websocket.Match.IncomingWithData(init.ID, respRaw) {
			return "", nil
		}
	}
	if init.Error.Message != "" || init.Error.Code != 0 {
		return "", fmt.Errorf("code: %d, Message: %s",
			init.Error.Code,
			init.Error.Message)
	}
	if _, ok := init.Result.(bool); ok {
		return "", nil
	}
	if init.Method != "" {
		return init.Method, nil
	}
	switch resultType := init.Result.(type) {
	case map[string]any:
		if reportType, ok := resultType["reportType"].(string); ok {
			return reportType, nil
		}
		// check for ids - means it was a specific request
		// and can't go through normal processing
		if responseID, ok := resultType["id"].(string); ok {
			if responseID != "" {
				return "", nil
			}
		}
	case []any:
		if len(resultType) == 0 {
			e.Websocket.DataHandler <- fmt.Sprintf("No data returned. ID: %v", init.ID)
			return "", nil
		}

		data, ok := resultType[0].(map[string]any)
		if !ok {
			return "", errors.New("unable to type assert data")
		}
		if _, ok := data["clientOrderId"]; ok {
			return "order", nil
		} else if _, ok := data["available"]; ok {
			return "trading", nil
		}
	}
	e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
	return "", nil
}

func (e *Exchange) wsHandleData(respRaw []byte) error {
	name, err := e.wsGetTableName(respRaw)
	if err != nil {
		return err
	}
	switch name {
	case "":
		return nil
	case "ticker":
		var wsTicker WsTicker
		err := json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}

		pairs, err := e.GetEnabledPairs(asset.Spot)
		if err != nil {
			return err
		}

		format, err := e.GetPairFormat(asset.Spot, true)
		if err != nil {
			return err
		}

		p, err := currency.NewPairFromFormattedPairs(wsTicker.Params.Symbol,
			pairs,
			format)
		if err != nil {
			return err
		}

		e.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: e.Name,
			Open:         wsTicker.Params.Open,
			Volume:       wsTicker.Params.Volume,
			QuoteVolume:  wsTicker.Params.VolumeQuote,
			High:         wsTicker.Params.High,
			Low:          wsTicker.Params.Low,
			Bid:          wsTicker.Params.Bid,
			Ask:          wsTicker.Params.Ask,
			Last:         wsTicker.Params.Last,
			LastUpdated:  wsTicker.Params.Timestamp,
			AssetType:    asset.Spot,
			Pair:         p,
		}
	case "snapshotOrderbook":
		var obSnapshot WsOrderbook
		err := json.Unmarshal(respRaw, &obSnapshot)
		if err != nil {
			return err
		}
		err = e.WsProcessOrderbookSnapshot(&obSnapshot)
		if err != nil {
			return err
		}
	case "updateOrderbook":
		var obUpdate WsOrderbook
		err := json.Unmarshal(respRaw, &obUpdate)
		if err != nil {
			return err
		}
		err = e.WsProcessOrderbookUpdate(&obUpdate)
		if err != nil {
			return err
		}
	case "snapshotTrades", "updateTrades":
		if !e.IsSaveTradeDataEnabled() {
			return nil
		}
		var tradeSnapshot WsTrade
		err := json.Unmarshal(respRaw, &tradeSnapshot)
		if err != nil {
			return err
		}
		var trades []trade.Data
		p, err := currency.NewPairFromString(tradeSnapshot.Params.Symbol)
		if err != nil {
			return &order.ClassificationError{
				Exchange: e.Name,
				Err:      err,
			}
		}
		for i := range tradeSnapshot.Params.Data {
			side, err := order.StringToOrderSide(tradeSnapshot.Params.Data[i].Side)
			if err != nil {
				return &order.ClassificationError{
					Exchange: e.Name,
					Err:      err,
				}
			}
			trades = append(trades, trade.Data{
				Timestamp:    tradeSnapshot.Params.Data[i].Timestamp,
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    asset.Spot,
				Price:        tradeSnapshot.Params.Data[i].Price,
				Amount:       tradeSnapshot.Params.Data[i].Quantity,
				Side:         side,
				TID:          strconv.FormatInt(tradeSnapshot.Params.Data[i].ID, 10),
			})
		}
		return trade.AddTradesToBuffer(trades...)
	case "activeOrders":
		var o wsActiveOrdersResponse
		err := json.Unmarshal(respRaw, &o)
		if err != nil {
			return err
		}
		for i := range o.Params {
			err = e.wsHandleOrderData(&o.Params[i])
			if err != nil {
				return err
			}
		}
	case "trading":
		var trades WsGetTradingBalanceResponse
		err := json.Unmarshal(respRaw, &trades)
		if err != nil {
			return err
		}
		e.Websocket.DataHandler <- trades
	case "report":
		var o wsReportResponse
		err := json.Unmarshal(respRaw, &o)
		if err != nil {
			return err
		}
		err = e.wsHandleOrderData(&o.OrderData)
		if err != nil {
			return err
		}
	case "order":
		var o wsActiveOrderRequestResponse
		err := json.Unmarshal(respRaw, &o)
		if err != nil {
			return err
		}
		for i := range o.OrderData {
			err = e.wsHandleOrderData(&o.OrderData[i])
			if err != nil {
				return err
			}
		}
	case "replaced", "canceled", "new":
		var o wsOrderResponse
		err := json.Unmarshal(respRaw, &o)
		if err != nil {
			return err
		}
		err = e.wsHandleOrderData(&o.OrderData)
		if err != nil {
			return err
		}
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

// WsProcessOrderbookSnapshot processes a full orderbook snapshot to a local cache
func (e *Exchange) WsProcessOrderbookSnapshot(ob *WsOrderbook) error {
	if len(ob.Params.Bid) == 0 || len(ob.Params.Ask) == 0 {
		return errors.New("no orderbooks to process")
	}

	newOrderBook := orderbook.Book{
		Bids: make(orderbook.Levels, len(ob.Params.Bid)),
		Asks: make(orderbook.Levels, len(ob.Params.Ask)),
	}
	for i := range ob.Params.Bid {
		newOrderBook.Bids[i] = orderbook.Level{
			Amount: ob.Params.Bid[i].Size,
			Price:  ob.Params.Bid[i].Price,
		}
	}
	for i := range ob.Params.Ask {
		newOrderBook.Asks[i] = orderbook.Level{
			Amount: ob.Params.Ask[i].Size,
			Price:  ob.Params.Ask[i].Price,
		}
	}

	pairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	p, err := currency.NewPairFromFormattedPairs(ob.Params.Symbol,
		pairs,
		format)
	if err != nil {
		e.Websocket.DataHandler <- err
		return err
	}

	newOrderBook.Asset = asset.Spot
	newOrderBook.Pair = p
	newOrderBook.Exchange = e.Name
	newOrderBook.ValidateOrderbook = e.ValidateOrderbook
	newOrderBook.LastUpdated = ob.Params.Timestamp

	return e.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

func (e *Exchange) wsHandleOrderData(o *wsOrderData) error {
	var trades []order.TradeHistory
	if o.TradeID > 0 {
		trades = append(trades, order.TradeHistory{
			Price:     o.TradePrice,
			Amount:    o.TradeQuantity,
			Fee:       o.TradeFee,
			Exchange:  e.Name,
			TID:       strconv.FormatFloat(o.TradeID, 'f', -1, 64),
			Timestamp: o.UpdatedAt,
		})
	}
	oType, err := order.StringToOrderType(o.Type)
	if err != nil {
		e.Websocket.DataHandler <- order.ClassificationError{
			Exchange: e.Name,
			OrderID:  o.ID,
			Err:      err,
		}
	}
	o.Status = strings.Replace(o.Status, "canceled", "cancelled", 1)
	oStatus, err := order.StringToOrderStatus(o.Status)
	if err != nil {
		e.Websocket.DataHandler <- order.ClassificationError{
			Exchange: e.Name,
			OrderID:  o.ID,
			Err:      err,
		}
	}
	oSide, err := order.StringToOrderSide(o.Side)
	if err != nil {
		e.Websocket.DataHandler <- order.ClassificationError{
			Exchange: e.Name,
			OrderID:  o.ID,
			Err:      err,
		}
	}

	p, err := currency.NewPairFromString(o.Symbol)
	if err != nil {
		e.Websocket.DataHandler <- order.ClassificationError{
			Exchange: e.Name,
			OrderID:  o.ID,
			Err:      err,
		}
	}

	var a asset.Item
	a, err = e.GetPairAssetType(p)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- &order.Detail{
		Price:           o.Price,
		Amount:          o.Quantity,
		ExecutedAmount:  o.CumQuantity,
		RemainingAmount: o.Quantity - o.CumQuantity,
		Exchange:        e.Name,
		OrderID:         o.ID,
		Type:            oType,
		Side:            oSide,
		Status:          oStatus,
		AssetType:       a,
		Date:            o.CreatedAt,
		LastUpdated:     o.UpdatedAt,
		Pair:            p,
		Trades:          trades,
	}
	return nil
}

// WsProcessOrderbookUpdate updates a local cache
func (e *Exchange) WsProcessOrderbookUpdate(update *WsOrderbook) error {
	if len(update.Params.Bid) == 0 && len(update.Params.Ask) == 0 {
		// Periodically HitBTC sends empty updates which includes a sequence
		// can return this as nil.
		return nil
	}

	bids := make(orderbook.Levels, len(update.Params.Bid))
	for i := range update.Params.Bid {
		bids[i] = orderbook.Level{
			Price:  update.Params.Bid[i].Price,
			Amount: update.Params.Bid[i].Size,
		}
	}

	asks := make(orderbook.Levels, len(update.Params.Ask))
	for i := range update.Params.Ask {
		asks[i] = orderbook.Level{
			Price:  update.Params.Ask[i].Price,
			Amount: update.Params.Ask[i].Size,
		}
	}

	pairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	p, err := currency.NewPairFromFormattedPairs(update.Params.Symbol,
		pairs,
		format)
	if err != nil {
		return err
	}

	return e.Websocket.Orderbook.Update(&orderbook.Update{
		Asks:       asks,
		Bids:       bids,
		Pair:       p,
		UpdateID:   update.Params.Sequence,
		Asset:      asset.Spot,
		UpdateTime: update.Params.Timestamp,
	})
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		"subToReq":        subToReq,
		"isSymbolChannel": isSymbolChannel,
	}).Parse(subTplText)
}

const (
	subscribeOp   = "subscribe"
	unsubscribeOp = "unsubscribe"
)

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	return e.ParallelChanOp(ctx, subs, func(ctx context.Context, subs subscription.List) error { return e.manageSubs(ctx, subscribeOp, subs) }, 1)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	return e.ParallelChanOp(ctx, subs, func(ctx context.Context, subs subscription.List) error { return e.manageSubs(ctx, unsubscribeOp, subs) }, 1)
}

func (e *Exchange) manageSubs(ctx context.Context, op string, subs subscription.List) error {
	var errs error
	subs, errs = subs.ExpandTemplates(e)
	for _, s := range subs {
		r := WsRequest{
			JSONRPCVersion: rpcVersion,
			ID:             e.Websocket.Conn.GenerateMessageID(false),
		}
		if err := json.Unmarshal([]byte(s.QualifiedChannel), &r); err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		r.Method = op + r.Method
		err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, r) // v2 api does not return an ID with errors, so we don't use ReturnResponse
		if err == nil {
			if op == subscribeOp {
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

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) wsLogin(ctx context.Context) error {
	if !e.IsWebsocketAuthenticationSupported() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", e.Name)
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	n := strconv.FormatInt(time.Now().Unix(), 10)
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(n), []byte(creds.Secret))
	if err != nil {
		return err
	}

	req := WsLoginRequest{
		Method: "login",
		Params: WsLoginData{
			Algo:      "HS256",
			PKey:      creds.Key,
			Nonce:     n,
			Signature: hex.EncodeToString(hmac),
		},
		ID: e.Websocket.Conn.GenerateMessageID(false),
	}

	err = e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, req)
	if err != nil {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}

	return nil
}

// wsPlaceOrder sends a websocket message to submit an order
func (e *Exchange) wsPlaceOrder(ctx context.Context, pair currency.Pair, side string, price, quantity float64) (*WsSubmitOrderSuccessResponse, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", e.Name)
	}

	id := e.Websocket.Conn.GenerateMessageID(false)
	fPair, err := e.FormatExchangeCurrency(pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	req := WsSubmitOrderRequest{
		Method: "newOrder",
		Params: WsSubmitOrderRequestData{
			ClientOrderID: id,
			Symbol:        fPair.String(),
			Side:          strings.ToLower(side),
			Price:         price,
			Quantity:      quantity,
		},
		ID: id,
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, id, req)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	var response WsSubmitOrderSuccessResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", e.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsCancelOrder sends a websocket message to cancel an order
func (e *Exchange) wsCancelOrder(ctx context.Context, clientOrderID string) (*WsCancelOrderResponse, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", e.Name)
	}
	req := WsCancelOrderRequest{
		Method: "cancelOrder",
		Params: WsCancelOrderRequestData{
			ClientOrderID: clientOrderID,
		},
		ID: e.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	var response WsCancelOrderResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", e.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsReplaceOrder sends a websocket message to replace an order
func (e *Exchange) wsReplaceOrder(ctx context.Context, clientOrderID string, quantity, price float64) (*WsReplaceOrderResponse, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", e.Name)
	}
	req := WsReplaceOrderRequest{
		Method: "cancelReplaceOrder",
		Params: WsReplaceOrderRequestData{
			ClientOrderID:   clientOrderID,
			RequestClientID: strconv.FormatInt(time.Now().Unix(), 10),
			Quantity:        quantity,
			Price:           price,
		},
		ID: e.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	var response WsReplaceOrderResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", e.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetActiveOrders sends a websocket message to get all active orders
func (e *Exchange) wsGetActiveOrders(ctx context.Context) (*wsActiveOrdersResponse, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot get active orders", e.Name)
	}
	req := WsReplaceOrderRequest{
		Method: "getOrders",
		Params: WsReplaceOrderRequestData{},
		ID:     e.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	var response wsActiveOrdersResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", e.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetTradingBalance sends a websocket message to get trading balance
func (e *Exchange) wsGetTradingBalance(ctx context.Context) (*WsGetTradingBalanceResponse, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", e.Name)
	}
	req := WsReplaceOrderRequest{
		Method: "getTradingBalance",
		Params: WsReplaceOrderRequestData{},
		ID:     e.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	var response WsGetTradingBalanceResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", e.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetCurrencies sends a websocket message to get trading balance
func (e *Exchange) wsGetCurrencies(ctx context.Context, currencyItem currency.Code) (*WsGetCurrenciesResponse, error) {
	req := WsGetCurrenciesRequest{
		Method: "getCurrency",
		Params: WsGetCurrenciesRequestParameters{
			Currency: currencyItem,
		},
		ID: e.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	var response WsGetCurrenciesResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", e.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetSymbols sends a websocket message to get trading balance
func (e *Exchange) wsGetSymbols(ctx context.Context, c currency.Pair) (*WsGetSymbolsResponse, error) {
	fPair, err := e.FormatExchangeCurrency(c, asset.Spot)
	if err != nil {
		return nil, err
	}

	req := WsGetSymbolsRequest{
		Method: "getSymbol",
		Params: WsGetSymbolsRequestParameters{
			Symbol: fPair.String(),
		},
		ID: e.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	var response WsGetSymbolsResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", e.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetSymbols sends a websocket message to get trading balance
func (e *Exchange) wsGetTrades(ctx context.Context, c currency.Pair, limit int64, sort, by string) (*WsGetTradesResponse, error) {
	fPair, err := e.FormatExchangeCurrency(c, asset.Spot)
	if err != nil {
		return nil, err
	}

	req := WsGetTradesRequest{
		Method: "getTrades",
		Params: WsGetTradesRequestParameters{
			Symbol: fPair.String(),
			Limit:  limit,
			Sort:   sort,
			By:     by,
		},
		ID: e.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	var response WsGetTradesResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", e.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", e.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// subToReq returns the subscription as a map to populate WsRequest
func subToReq(s *subscription.Subscription, maybePair ...currency.Pair) *WsRequest {
	name, ok := subscriptionNames[s.Channel]
	if !ok {
		panic(fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel))
	}
	r := &WsRequest{
		Method: name,
	}
	if len(maybePair) != 0 {
		r.Params = &WsParams{
			Symbol: maybePair[0].String(),
			Limit:  s.Levels,
		}
		if s.Interval != 0 {
			var err error
			if r.Params.Period, err = formatExchangeKlineInterval(s.Interval); err != nil {
				panic(err)
			}
		}
	} else if s.Levels != 0 {
		r.Params = &WsParams{
			Limit: s.Levels,
		}
	}
	return r
}

// isSymbolChannel returns if the channel expects receive a symbol
func isSymbolChannel(s *subscription.Subscription) bool {
	return s.Channel != subscription.MyAccountChannel
}

const subTplText = `
{{- if isSymbolChannel $.S }} 
	{{ range $asset, $pairs := $.AssetPairs }}
		{{- range $p := $pairs -}}
			{{- subToReq $.S $p | mustToJson }}
			{{ $.PairSeparator }}
		{{- end }}
		{{ $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- subToReq $.S | mustToJson }}
{{- end }}
`
