package huobi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	baseWSURL    = "wss://api.huobi.pro"
	futuresWSURL = "wss://api.hbdm.com/"

	wsMarketURL           = baseWSURL + "/ws"
	wsCandlesChannel      = "market.%s.kline"
	wsOrderbookChannel    = "market.%s.depth"
	wsTradesChannel       = "market.%s.trade.detail"
	wsMarketDetailChannel = "market.%s.detail"
	wsMyOrdersChannel     = "orders.%s"
	wsMyTradesChannel     = "orders.%s.update"

	wsAccountsOrdersEndPoint = "/ws/v1"
	wsAccountsList           = "accounts.list"
	wsOrdersList             = "orders.list"
	wsOrdersDetail           = "orders.detail"
	wsAccountsOrdersURL      = baseWSURL + wsAccountsOrdersEndPoint
	wsAccountListEndpoint    = wsAccountsOrdersEndPoint + "/" + wsAccountsList
	wsOrdersListEndpoint     = wsAccountsOrdersEndPoint + "/" + wsOrdersList
	wsOrdersDetailEndpoint   = wsAccountsOrdersEndPoint + "/" + wsOrdersDetail

	wsDateTimeFormatting = "2006-01-02T15:04:05"

	signatureMethod  = "HmacSHA256"
	signatureVersion = "2"
	requestOp        = "req"
	authOp           = "auth"

	loginDelay = 50 * time.Millisecond
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
	/* 	TODO: Pending upcoming V2 support, these are dropped from the translation table so that the sub conf will be correct and not need upgrading, but will error on usage
	subscription.MyTradesChannel:  wsMyOrdersChannel,
	subscription.MyOrdersChannel:  wsMyTradesChannel,
	subscription.MyAccountChannel: wsMyAccountChannel,
	*/
}

// Instantiates a communications channel between websocket connections
var comms = make(chan WsMessage)

// WsConnect initiates a new websocket connection
func (h *HUOBI) WsConnect() error {
	if !h.Websocket.IsEnabled() || !h.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	err := h.wsDial(&dialer)
	if err != nil {
		return err
	}

	if h.Websocket.CanUseAuthenticatedEndpoints() {
		err = h.wsAuthenticatedDial(&dialer)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v - authenticated dial failed: %v\n",
				h.Name,
				err)
		}
		err = h.wsLogin(context.TODO())
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v - authentication failed: %v\n",
				h.Name,
				err)
			h.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	h.Websocket.Wg.Add(1)
	go h.wsReadData()
	return nil
}

func (h *HUOBI) wsDial(dialer *websocket.Dialer) error {
	err := h.Websocket.Conn.Dial(dialer, http.Header{})
	if err != nil {
		return err
	}
	h.Websocket.Wg.Add(1)
	go h.wsFunnelConnectionData(h.Websocket.Conn, wsMarketURL)
	return nil
}

func (h *HUOBI) wsAuthenticatedDial(dialer *websocket.Dialer) error {
	if !h.IsWebsocketAuthenticationSupported() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled",
			h.Name)
	}
	err := h.Websocket.AuthConn.Dial(dialer, http.Header{})
	if err != nil {
		return err
	}

	h.Websocket.Wg.Add(1)
	go h.wsFunnelConnectionData(h.Websocket.AuthConn, wsAccountsOrdersURL)
	return nil
}

// wsFunnelConnectionData manages data from multiple endpoints and passes it to
// a channel
func (h *HUOBI) wsFunnelConnectionData(ws stream.Connection, url string) {
	defer h.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		comms <- WsMessage{Raw: resp.Raw, URL: url}
	}
}

// wsReadData receives and passes on websocket messages for processing
func (h *HUOBI) wsReadData() {
	defer h.Websocket.Wg.Done()
	for {
		select {
		case <-h.Websocket.ShutdownC:
			select {
			case resp := <-comms:
				err := h.wsHandleData(resp.Raw)
				if err != nil {
					select {
					case h.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr,
							"%s websocket handle data error: %v",
							h.Name,
							err)
					}
				}
			default:
			}
			return
		case resp := <-comms:
			err := h.wsHandleData(resp.Raw)
			if err != nil {
				h.Websocket.DataHandler <- err
			}
		}
	}
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "submitted":
		return order.New, nil
	case "canceled":
		return order.Cancelled, nil
	case "partial-filled":
		return order.PartiallyFilled, nil
	case "partial-canceled":
		return order.PartiallyCancelled, nil
	default:
		return order.UnknownStatus,
			errors.New(status + " not recognised as order status")
	}
}

func stringToOrderSide(side string) (order.Side, error) {
	switch {
	case strings.Contains(side, "buy"):
		return order.Buy, nil
	case strings.Contains(side, "sell"):
		return order.Sell, nil
	}

	return order.UnknownSide,
		errors.New(side + " not recognised as order side")
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

func (h *HUOBI) wsHandleData(respRaw []byte) error {
	var init WsResponse
	err := json.Unmarshal(respRaw, &init)
	if err != nil {
		return err
	}
	if init.Subscribed != "" ||
		init.UnSubscribed != "" ||
		init.Op == "sub" ||
		init.Op == "unsub" {
		// TODO handle subs
		return nil
	}
	if init.Ping != 0 {
		h.sendPingResponse(init.Ping)
		return nil
	}

	if init.Op == "ping" {
		authPing := authenticationPing{
			OP: "pong",
			TS: init.TS,
		}
		err := h.Websocket.AuthConn.SendJSONMessage(context.TODO(), request.Unset, authPing)
		if err != nil {
			log.Errorln(log.ExchangeSys, err)
		}
		return nil
	}

	if init.ErrorMessage != "" {
		if init.ErrorMessage == "api-signature-not-valid" {
			h.Websocket.SetCanUseAuthenticatedEndpoints(false)
			return errors.New(h.Name +
				" - invalid credentials. Authenticated requests disabled")
		}

		codes, _ := init.ErrorCode.(string)
		return errors.New(h.Name + " Code:" + codes + " Message:" + init.ErrorMessage)
	}

	if init.ClientID > 0 {
		if h.Websocket.Match.IncomingWithData(init.ClientID, respRaw) {
			return nil
		}
	}

	switch {
	case strings.EqualFold(init.Op, authOp):
		h.Websocket.SetCanUseAuthenticatedEndpoints(true)
		// Auth captured
		return nil
	case strings.EqualFold(init.Topic, "accounts"):
		var response WsAuthenticatedAccountsResponse
		err := json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		h.Websocket.DataHandler <- response

	case strings.Contains(init.Topic, "orders") &&
		strings.Contains(init.Topic, "update"):
		var response WsAuthenticatedOrdersUpdateResponse
		err := json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		data := strings.Split(response.Topic, ".")
		if len(data) < 2 {
			return errors.New(h.Name +
				" - currency could not be extracted from response")
		}
		orderID := strconv.FormatInt(response.Data.OrderID, 10)
		var oSide order.Side
		oSide, err = stringToOrderSide(response.Data.OrderType)
		if err != nil {
			h.Websocket.DataHandler <- order.ClassificationError{
				Exchange: h.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		var oType order.Type
		oType, err = stringToOrderType(response.Data.OrderType)
		if err != nil {
			h.Websocket.DataHandler <- order.ClassificationError{
				Exchange: h.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		var oStatus order.Status
		oStatus, err = stringToOrderStatus(response.Data.OrderState)
		if err != nil {
			h.Websocket.DataHandler <- order.ClassificationError{
				Exchange: h.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		var p currency.Pair
		var a asset.Item
		p, a, err = h.GetRequestFormattedPairAndAssetType(data[1])
		if err != nil {
			return err
		}
		h.Websocket.DataHandler <- &order.Detail{
			Price:           response.Data.Price,
			Amount:          response.Data.UnfilledAmount + response.Data.FilledAmount,
			ExecutedAmount:  response.Data.FilledAmount,
			RemainingAmount: response.Data.UnfilledAmount,
			Exchange:        h.Name,
			OrderID:         orderID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       a,
			LastUpdated:     time.Unix(response.TS*1000, 0),
			Pair:            p,
		}

	case strings.Contains(init.Topic, "orders"):
		var response WsOldOrderUpdate
		err := json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		h.Websocket.DataHandler <- response
	case strings.Contains(init.Channel, "depth"):
		var depth WsDepth
		err := json.Unmarshal(respRaw, &depth)
		if err != nil {
			return err
		}

		data := strings.Split(depth.Channel, ".")
		err = h.WsProcessOrderbook(&depth, data[1])
		if err != nil {
			return err
		}
	case strings.Contains(init.Channel, "kline"):
		var kline WsKline
		err := json.Unmarshal(respRaw, &kline)
		if err != nil {
			return err
		}
		data := strings.Split(kline.Channel, ".")
		var p currency.Pair
		var a asset.Item
		p, a, err = h.GetRequestFormattedPairAndAssetType(data[1])
		if err != nil {
			return err
		}
		h.Websocket.DataHandler <- stream.KlineData{
			Timestamp:  time.UnixMilli(kline.Timestamp),
			Exchange:   h.Name,
			AssetType:  a,
			Pair:       p,
			OpenPrice:  kline.Tick.Open,
			ClosePrice: kline.Tick.Close,
			HighPrice:  kline.Tick.High,
			LowPrice:   kline.Tick.Low,
			Volume:     kline.Tick.Volume,
			Interval:   data[3],
		}
	case strings.Contains(init.Channel, "trade.detail"):
		if !h.IsSaveTradeDataEnabled() {
			return nil
		}
		var t WsTrade
		err := json.Unmarshal(respRaw, &t)
		if err != nil {
			return err
		}
		data := strings.Split(t.Channel, ".")
		var p currency.Pair
		var a asset.Item
		p, a, err = h.GetRequestFormattedPairAndAssetType(data[1])
		if err != nil {
			return err
		}
		var trades []trade.Data
		for i := range t.Tick.Data {
			side := order.Buy
			if t.Tick.Data[i].Direction != "buy" {
				side = order.Sell
			}
			trades = append(trades, trade.Data{
				Exchange:     h.Name,
				AssetType:    a,
				CurrencyPair: p,
				Timestamp:    time.UnixMilli(t.Tick.Data[i].Timestamp),
				Amount:       t.Tick.Data[i].Amount,
				Price:        t.Tick.Data[i].Price,
				Side:         side,
				TID:          strconv.FormatFloat(t.Tick.Data[i].TradeID, 'f', -1, 64),
			})
		}
		return trade.AddTradesToBuffer(h.Name, trades...)
	case strings.Contains(init.Channel, "detail"),
		strings.Contains(init.Rep, "detail"):
		var wsTicker WsTick
		err := json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}
		var data []string
		if wsTicker.Channel != "" {
			data = strings.Split(wsTicker.Channel, ".")
		}
		if wsTicker.Rep != "" {
			data = strings.Split(wsTicker.Rep, ".")
		}

		var p currency.Pair
		var a asset.Item
		p, a, err = h.GetRequestFormattedPairAndAssetType(data[1])
		if err != nil {
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
			LastUpdated:  time.UnixMilli(wsTicker.Timestamp),
			AssetType:    a,
			Pair:         p,
		}
	default:
		h.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: h.Name + stream.UnhandledMessage + string(respRaw),
		}
		return nil
	}
	return nil
}

func (h *HUOBI) sendPingResponse(pong int64) {
	err := h.Websocket.Conn.SendJSONMessage(context.TODO(), request.Unset, WsPong{Pong: pong})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// WsProcessOrderbook processes new orderbook data
func (h *HUOBI) WsProcessOrderbook(update *WsDepth, symbol string) error {
	pairs, err := h.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := h.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	p, err := currency.NewPairFromFormattedPairs(symbol,
		pairs,
		format)
	if err != nil {
		return err
	}

	bids := make(orderbook.Tranches, len(update.Tick.Bids))
	for i := range update.Tick.Bids {
		price, ok := update.Tick.Bids[i][0].(float64)
		if !ok {
			return errors.New("unable to type assert bid price")
		}
		amount, ok := update.Tick.Bids[i][1].(float64)
		if !ok {
			return errors.New("unable to type assert bid amount")
		}
		bids[i] = orderbook.Tranche{
			Price:  price,
			Amount: amount,
		}
	}

	asks := make(orderbook.Tranches, len(update.Tick.Asks))
	for i := range update.Tick.Asks {
		price, ok := update.Tick.Asks[i][0].(float64)
		if !ok {
			return errors.New("unable to type assert ask price")
		}
		amount, ok := update.Tick.Asks[i][1].(float64)
		if !ok {
			return errors.New("unable to type assert ask amount")
		}
		asks[i] = orderbook.Tranche{
			Price:  price,
			Amount: amount,
		}
	}

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.Pair = p
	newOrderBook.Asset = asset.Spot
	newOrderBook.Exchange = h.Name
	newOrderBook.VerifyOrderbook = h.CanVerifyOrderbook
	newOrderBook.LastUpdated = time.UnixMilli(update.Timestamp)

	return h.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (h *HUOBI) generateSubscriptions() (subscription.List, error) {
	return h.Features.Subscriptions.ExpandTemplates(h)
}

// GetSubscriptionTemplate returns a subscription channel template
func (h *HUOBI) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName": channelName,
		"interval":    h.FormatExchangeKlineInterval,
	}).Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from the channel
func (h *HUOBI) Subscribe(subs subscription.List) error {
	ctx := context.Background()
	var errs error
	var creds *account.Credentials
	if len(subs.Private()) > 0 {
		if creds, errs = h.GetCredentials(ctx); errs != nil {
			return errs
		}
	}
	for _, s := range subs {
		var err error
		if s.Authenticated {
			if err = h.wsAuthenticatedSubscribe(creds, "sub", wsAccountsOrdersEndPoint+"/"+s.QualifiedChannel, s.QualifiedChannel); err == nil {
				err = h.Websocket.AddSuccessfulSubscriptions(h.Websocket.Conn, s)
			}
		} else {
			if err = h.Websocket.Conn.SendJSONMessage(ctx, request.Unset, WsRequest{Subscribe: s.QualifiedChannel}); err == nil {
				err = h.Websocket.AddSuccessfulSubscriptions(h.Websocket.AuthConn, s)
			}
		}
		errs = common.AppendError(errs, err)
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HUOBI) Unsubscribe(subs subscription.List) error {
	ctx := context.Background()
	var errs error
	var creds *account.Credentials
	if len(subs.Private()) > 0 {
		if creds, errs = h.GetCredentials(ctx); errs != nil {
			return errs
		}
	}
	for _, s := range subs {
		var err error
		if s.Authenticated {
			err = h.wsAuthenticatedSubscribe(creds, "unsub", wsAccountsOrdersEndPoint+"/"+s.QualifiedChannel, s.QualifiedChannel)
		} else {
			err = h.Websocket.Conn.SendJSONMessage(ctx, request.Unset, WsRequest{Unsubscribe: s.QualifiedChannel})
		}
		if err == nil {
			err = h.Websocket.RemoveSubscriptions(h.Websocket.Conn, s)
		}
		errs = common.AppendError(errs, err)
	}
	return errs
}

func (h *HUOBI) wsGenerateSignature(creds *account.Credentials, timestamp, endpoint string) ([]byte, error) {
	values := url.Values{}
	values.Set("AccessKeyId", creds.Key)
	values.Set("SignatureMethod", signatureMethod)
	values.Set("SignatureVersion", signatureVersion)
	values.Set("Timestamp", timestamp)
	host := "api.huobi.pro"
	payload := fmt.Sprintf("%s\n%s\n%s\n%s",
		http.MethodGet, host, endpoint, values.Encode())
	return crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(creds.Secret))
}

func (h *HUOBI) wsLogin(ctx context.Context) error {
	if !h.IsWebsocketAuthenticationSupported() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", h.Name)
	}
	creds, err := h.GetCredentials(ctx)
	if err != nil {
		return err
	}

	h.Websocket.SetCanUseAuthenticatedEndpoints(true)
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	req := WsAuthenticationRequest{
		Op:               authOp,
		AccessKeyID:      creds.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
	}
	hmac, err := h.wsGenerateSignature(creds, timestamp, wsAccountsOrdersEndPoint)
	if err != nil {
		return err
	}
	req.Signature = crypto.Base64Encode(hmac)
	err = h.Websocket.AuthConn.SendJSONMessage(context.TODO(), request.Unset, req)
	if err != nil {
		h.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}

	time.Sleep(loginDelay)
	return nil
}

func (h *HUOBI) wsAuthenticatedSubscribe(creds *account.Credentials, operation, endpoint, topic string) error {
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	req := WsAuthenticatedSubscriptionRequest{
		Op:               operation,
		AccessKeyID:      creds.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            topic,
	}
	hmac, err := h.wsGenerateSignature(creds, timestamp, endpoint)
	if err != nil {
		return err
	}
	req.Signature = crypto.Base64Encode(hmac)
	return h.Websocket.AuthConn.SendJSONMessage(context.TODO(), request.Unset, req)
}

func (h *HUOBI) wsGetAccountsList(ctx context.Context) (*WsAuthenticatedAccountsListResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated cannot get accounts list", h.Name)
	}
	creds, err := h.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	req := WsAuthenticatedAccountsListRequest{
		Op:               requestOp,
		AccessKeyID:      creds.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            wsAccountsList,
	}
	hmac, err := h.wsGenerateSignature(creds, timestamp, wsAccountListEndpoint)
	if err != nil {
		return nil, err
	}
	req.Signature = crypto.Base64Encode(hmac)
	req.ClientID = h.Websocket.AuthConn.GenerateMessageID(true)
	resp, err := h.Websocket.AuthConn.SendMessageReturnResponse(context.TODO(), request.Unset, req.ClientID, req)
	if err != nil {
		return nil, err
	}
	var response WsAuthenticatedAccountsListResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}

	code, _ := response.ErrorCode.(int)
	if code != 0 {
		return nil, errors.New(response.ErrorMessage)
	}
	return &response, nil
}

func (h *HUOBI) wsGetOrdersList(ctx context.Context, accountID int64, pair currency.Pair) (*WsAuthenticatedOrdersResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated cannot get orders list", h.Name)
	}

	creds, err := h.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}

	fPair, err := h.FormatExchangeCurrency(pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	req := WsAuthenticatedOrdersListRequest{
		Op:               requestOp,
		AccessKeyID:      creds.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            wsOrdersList,
		AccountID:        accountID,
		Symbol:           fPair.String(),
		States:           "submitted,partial-filled",
	}

	hmac, err := h.wsGenerateSignature(creds, timestamp, wsOrdersListEndpoint)
	if err != nil {
		return nil, err
	}
	req.Signature = crypto.Base64Encode(hmac)
	req.ClientID = h.Websocket.AuthConn.GenerateMessageID(true)

	resp, err := h.Websocket.AuthConn.SendMessageReturnResponse(context.TODO(), request.Unset, req.ClientID, req)
	if err != nil {
		return nil, err
	}

	var response WsAuthenticatedOrdersResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}

	code, _ := response.ErrorCode.(int)
	if code != 0 {
		return nil, errors.New(response.ErrorMessage)
	}
	return &response, nil
}

func (h *HUOBI) wsGetOrderDetails(ctx context.Context, orderID string) (*WsAuthenticatedOrderDetailResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated cannot get order details", h.Name)
	}
	creds, err := h.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	req := WsAuthenticatedOrderDetailsRequest{
		Op:               requestOp,
		AccessKeyID:      creds.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: signatureVersion,
		Timestamp:        timestamp,
		Topic:            wsOrdersDetail,
		OrderID:          orderID,
	}
	hmac, err := h.wsGenerateSignature(creds, timestamp, wsOrdersDetailEndpoint)
	if err != nil {
		return nil, err
	}
	req.Signature = crypto.Base64Encode(hmac)
	req.ClientID = h.Websocket.AuthConn.GenerateMessageID(true)
	resp, err := h.Websocket.AuthConn.SendMessageReturnResponse(context.TODO(), request.Unset, req.ClientID, req)
	if err != nil {
		return nil, err
	}
	var response WsAuthenticatedOrderDetailResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}

	code, _ := response.ErrorCode.(int)
	if code != 0 {
		return nil, errors.New(response.ErrorMessage)
	}
	return &response, nil
}

// channelName converts global channel Names used in config of channel input into exchange channel names
// returns the name unchanged if no match is found
func channelName(s *subscription.Subscription, p currency.Pair) string {
	if n, ok := subscriptionNames[s.Channel]; ok {
		return fmt.Sprintf(n, p)
	}
	if s.Authenticated {
		panic(fmt.Errorf("%w: Private endpoints not currently supported", common.ErrNotYetImplemented))
	}
	panic(subscription.ErrUseConstChannelName)
}

const subTplText = `
{{- if $.S.Asset }}
	{{ range $asset, $pairs := $.AssetPairs }}
		{{- range $p := $pairs }}
			{{- channelName $.S $p -}}
			{{- if eq $.S.Channel "candles" -}} . {{- interval $.S.Interval }}{{ end }}
			{{- if eq $.S.Channel "orderbook" -}} .step {{- $.S.Levels }}{{ end }}
			{{ $.PairSeparator }}
		{{- end }}
		{{ $.AssetSeparator }}
	{{- end }}
{{- else -}}
	{{ channelName $.S nil }}
{{- end }}
`
