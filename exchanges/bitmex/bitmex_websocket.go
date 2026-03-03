package bitmex

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

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
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
	bitmexWSURL = "wss://ws.bitmex.com/realtime"

	// Public Subscription Channels
	bitmexWSAnnouncement        = "announcement"
	bitmexWSChat                = "chat"
	bitmexWSConnected           = "connected"
	bitmexWSFunding             = "funding"
	bitmexWSInstrument          = "instrument"
	bitmexWSInsurance           = "insurance"
	bitmexWSLiquidation         = "liquidation"
	bitmexWSOrderbookL2         = "orderBookL2"
	bitmexWSOrderbookL225       = "orderBookL2_25"
	bitmexWSOrderbookL10        = "orderBook10"
	bitmexWSPublicNotifications = "publicNotifications"
	bitmexWSQuote               = "quote"
	bitmexWSQuote1m             = "quoteBin1m"
	bitmexWSQuote5m             = "quoteBin5m"
	bitmexWSQuote1h             = "quoteBin1h"
	bitmexWSQuote1d             = "quoteBin1d"
	bitmexWSSettlement          = "settlement"
	bitmexWSTrade               = "trade"
	bitmexWSTrade1m             = "tradeBin1m"
	bitmexWSTrade5m             = "tradeBin5m"
	bitmexWSTrade1h             = "tradeBin1h"
	bitmexWSTrade1d             = "tradeBin1d"

	// Authenticated Subscription Channels
	bitmexWSAffiliate            = "affiliate"
	bitmexWSExecution            = "execution"
	bitmexWSOrder                = "order"
	bitmexWSMargin               = "margin"
	bitmexWSPosition             = "position"
	bitmexWSPrivateNotifications = "privateNotifications"
	bitmexWSTransact             = "transact"
	bitmexWSWallet               = "wallet"

	bitmexActionInitialData = "partial"
	bitmexActionInsertData  = "insert"
	bitmexActionDeleteData  = "delete"
	bitmexActionUpdateData  = "update"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Channel: bitmexWSOrderbookL2, Asset: asset.All},
	{Enabled: true, Channel: bitmexWSTrade, Asset: asset.All},
	{Enabled: true, Channel: bitmexWSAffiliate, Authenticated: true},
	{Enabled: true, Channel: bitmexWSOrder, Authenticated: true},
	{Enabled: true, Channel: bitmexWSMargin, Authenticated: true},
	{Enabled: true, Channel: bitmexWSTransact, Authenticated: true},
	{Enabled: true, Channel: bitmexWSWallet, Authenticated: true},
	{Enabled: true, Channel: bitmexWSExecution, Authenticated: true, Asset: asset.PerpetualContract},
	{Enabled: true, Channel: bitmexWSPosition, Authenticated: true, Asset: asset.PerpetualContract},
}

// WsConnect initiates a new websocket connection
func (e *Exchange) WsConnect() error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}

	ctx := context.TODO()
	var dialer gws.Dialer
	if err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{}); err != nil {
		return err
	}

	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)

	if e.Websocket.CanUseAuthenticatedEndpoints() {
		if err := e.websocketSendAuth(ctx); err != nil {
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", e.Name, err)
		}
	}

	return nil
}

const (
	wsSubscribeOp   = "subscribe"
	wsUnsubscribeOp = "unsubscribe"
)

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
	// We don't need to know about errors, since we're looking optimistically into the json
	op, _ := jsonparser.GetString(respRaw, "request", "op")
	errMsg, _ := jsonparser.GetString(respRaw, "error")
	success, _ := jsonparser.GetBoolean(respRaw, "success")
	version, _ := jsonparser.GetString(respRaw, "version")
	switch {
	case version != "":
		var welcomeResp WebsocketWelcome
		if err := json.Unmarshal(respRaw, &welcomeResp); err != nil {
			return err
		}

		if e.Verbose {
			log.Debugf(log.ExchangeSys, "%s successfully connected to websocket API at time: %s Limit: %d", e.Name, welcomeResp.Timestamp, welcomeResp.Limit.Remaining)
		}
		return nil
	case errMsg != "", success:
		var req any
		if op == "authKeyExpires" {
			req = op
		} else {
			reqBytes, _, _, err := jsonparser.Get(respRaw, "request")
			if err != nil {
				return err
			}
			req = string(reqBytes)
		}
		if err := e.Websocket.Match.RequireMatchWithData(req, respRaw); err != nil {
			return fmt.Errorf("%w: %s", err, op)
		}
		return nil
	}

	tableName, err := jsonparser.GetString(respRaw, "table")
	if err != nil {
		// Anything that's not a table isn't expected
		return fmt.Errorf("unknown message format: %s", respRaw)
	}

	switch tableName {
	case bitmexWSOrderbookL2, bitmexWSOrderbookL225, bitmexWSOrderbookL10:
		var orderbooks OrderBookData
		if err := json.Unmarshal(respRaw, &orderbooks); err != nil {
			return err
		}
		if len(orderbooks.Data) == 0 {
			return fmt.Errorf("empty orderbook data received: %s", respRaw)
		}

		pair, a, err := e.GetPairAndAssetTypeRequestFormatted(orderbooks.Data[0].Symbol)
		if err != nil {
			return err
		}

		err = e.processOrderbook(orderbooks.Data, orderbooks.Action, pair, a)
		if err != nil {
			return err
		}
	case bitmexWSTrade:
		return e.handleWsTrades(respRaw)
	case bitmexWSAnnouncement:
		var announcement AnnouncementData
		if err := json.Unmarshal(respRaw, &announcement); err != nil {
			return err
		}

		if announcement.Action == bitmexActionInitialData {
			return nil
		}

		return e.Websocket.DataHandler.Send(ctx, announcement.Data)
	case bitmexWSAffiliate:
		var response WsAffiliateResponse
		if err := json.Unmarshal(respRaw, &response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	case bitmexWSInstrument:
		// ticker
	case bitmexWSExecution:
		// trades of an order
		var response WsExecutionResponse
		if err := json.Unmarshal(respRaw, &response); err != nil {
			return err
		}

		for i := range response.Data {
			p, a, err := e.GetPairAndAssetTypeRequestFormatted(response.Data[i].Symbol)
			if err != nil {
				return err
			}
			oStatus, err := order.StringToOrderStatus(response.Data[i].OrdStatus)
			if err != nil {
				return err
			}
			oSide, err := order.StringToOrderSide(response.Data[i].Side)
			if err != nil {
				return err
			}
			if err := e.Websocket.DataHandler.Send(ctx, &order.Detail{
				Exchange:  e.Name,
				OrderID:   response.Data[i].OrderID,
				AccountID: strconv.FormatInt(response.Data[i].Account, 10),
				AssetType: a,
				Pair:      p,
				Status:    oStatus,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[i].Price,
						Amount:    response.Data[i].OrderQuantity,
						Exchange:  e.Name,
						TID:       response.Data[i].ExecID,
						Side:      oSide,
						Timestamp: response.Data[i].Timestamp,
						IsMaker:   false,
					},
				},
			}); err != nil {
				return err
			}
		}
	case bitmexWSOrder:
		var response WsOrderResponse
		if err := json.Unmarshal(respRaw, &response); err != nil {
			return err
		}
		switch response.Action {
		case "update", "insert":
			for x := range response.Data {
				p, a, err := e.GetRequestFormattedPairAndAssetType(response.Data[x].Symbol)
				if err != nil {
					return err
				}
				oSide, err := order.StringToOrderSide(response.Data[x].Side)
				if err != nil {
					return err
				}
				oType, err := order.StringToOrderType(response.Data[x].OrderType)
				if err != nil {
					return err
				}
				oStatus, err := order.StringToOrderStatus(response.Data[x].OrderStatus)
				if err != nil {
					return err
				}
				if err := e.Websocket.DataHandler.Send(ctx, &order.Detail{
					Price:     response.Data[x].Price,
					Amount:    response.Data[x].OrderQuantity,
					Exchange:  e.Name,
					OrderID:   response.Data[x].OrderID,
					AccountID: strconv.FormatInt(response.Data[x].Account, 10),
					Type:      oType,
					Side:      oSide,
					Status:    oStatus,
					AssetType: a,
					Date:      response.Data[x].TransactTime,
					Pair:      p,
				}); err != nil {
					return err
				}
			}
		case "delete":
			for x := range response.Data {
				p, a, err := e.GetRequestFormattedPairAndAssetType(response.Data[x].Symbol)
				if err != nil {
					return err
				}
				var oSide order.Side
				oSide, err = order.StringToOrderSide(response.Data[x].Side)
				if err != nil {
					return err
				}
				var oType order.Type
				oType, err = order.StringToOrderType(response.Data[x].OrderType)
				if err != nil {
					return err
				}
				var oStatus order.Status
				oStatus, err = order.StringToOrderStatus(response.Data[x].OrderStatus)
				if err != nil {
					return err
				}
				if err := e.Websocket.DataHandler.Send(ctx, &order.Detail{
					Price:     response.Data[x].Price,
					Amount:    response.Data[x].OrderQuantity,
					Exchange:  e.Name,
					OrderID:   response.Data[x].OrderID,
					AccountID: strconv.FormatInt(response.Data[x].Account, 10),
					Type:      oType,
					Side:      oSide,
					Status:    oStatus,
					AssetType: a,
					Date:      response.Data[x].TransactTime,
					Pair:      p,
				}); err != nil {
					return err
				}
			}
		default:
			return e.Websocket.DataHandler.Send(ctx, fmt.Errorf("%s - Unsupported order update %+v", e.Name, response))
		}
	case bitmexWSMargin:
		var response WsMarginResponse
		if err := json.Unmarshal(respRaw, &response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	case bitmexWSPosition:
		var response WsPositionResponse
		if err := json.Unmarshal(respRaw, &response); err != nil {
			return err
		}
	case bitmexWSPrivateNotifications:
		var response WsPrivateNotificationsResponse
		if err := json.Unmarshal(respRaw, &response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	case bitmexWSTransact:
		var response WsTransactResponse
		if err := json.Unmarshal(respRaw, &response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	case bitmexWSWallet:
		var response WsWalletResponse
		if err := json.Unmarshal(respRaw, &response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
	}

	return nil
}

// processOrderbook processes orderbook updates
func (e *Exchange) processOrderbook(data []OrderBookL2, action string, p currency.Pair, a asset.Item) error {
	if len(data) < 1 {
		return errors.New("no orderbook data")
	}

	switch action {
	case bitmexActionInitialData:
		book := orderbook.Book{
			Asks: make(orderbook.Levels, 0, len(data)),
			Bids: make(orderbook.Levels, 0, len(data)),
		}

		for i := range data {
			item := orderbook.Level{
				Price:  data[i].Price,
				Amount: float64(data[i].Size),
				ID:     data[i].ID,
			}
			switch {
			case strings.EqualFold(data[i].Side, order.Sell.String()):
				book.Asks = append(book.Asks, item)
			case strings.EqualFold(data[i].Side, order.Buy.String()):
				book.Bids = append(book.Bids, item)
			default:
				return fmt.Errorf("could not process websocket orderbook update, order side could not be matched for %s",
					data[i].Side)
			}
		}
		book.Asks.Reverse() // Reverse asks for correct alignment
		book.Asset = a
		book.Pair = p
		book.Exchange = e.Name
		book.ValidateOrderbook = e.ValidateOrderbook
		book.LastUpdated = data[0].Timestamp

		err := e.Websocket.Orderbook.LoadSnapshot(&book)
		if err != nil {
			return fmt.Errorf("process orderbook error -  %s",
				err)
		}
	default:
		updateAction, err := e.GetActionFromString(action)
		if err != nil {
			return err
		}

		asks := make([]orderbook.Level, 0, len(data))
		bids := make([]orderbook.Level, 0, len(data))
		for i := range data {
			nItem := orderbook.Level{
				Price:  data[i].Price,
				Amount: float64(data[i].Size),
				ID:     data[i].ID,
			}
			if strings.EqualFold(data[i].Side, "Sell") {
				asks = append(asks, nItem)
				continue
			}
			bids = append(bids, nItem)
		}

		err = e.Websocket.Orderbook.Update(&orderbook.Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       p,
			Asset:      a,
			Action:     updateAction,
			UpdateTime: data[0].Timestamp,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) handleWsTrades(msg []byte) error {
	if !e.IsSaveTradeDataEnabled() {
		return nil
	}
	var tradeHolder TradeData
	if err := json.Unmarshal(msg, &tradeHolder); err != nil {
		return err
	}
	trades := make([]trade.Data, 0, len(tradeHolder.Data))
	for _, t := range tradeHolder.Data {
		if t.Size == 0 {
			// Indices (symbols starting with .) post trades at intervals to the trade feed
			// These have a size of 0 and are used only to indicate a changing price
			continue
		}
		p, a, err := e.GetPairAndAssetTypeRequestFormatted(t.Symbol)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(t.Side)
		if err != nil {
			return err
		}

		trades = append(trades, trade.Data{
			TID:          t.TrdMatchID,
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    a,
			Side:         oSide,
			Price:        t.Price,
			Amount:       float64(t.Size),
			Timestamp:    t.Timestamp,
		})
	}
	return e.AddTradesToBuffer(trades...)
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName": channelName,
	}).Parse(subTplText)
}

// Subscribe subscribes to a websocket channel
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	return common.AppendError(
		e.ParallelChanOp(ctx, subs.Public(), func(ctx context.Context, l subscription.List) error { return e.manageSubs(ctx, wsSubscribeOp, l) }, len(subs)),
		e.ParallelChanOp(ctx, subs.Private(), func(ctx context.Context, l subscription.List) error { return e.manageSubs(ctx, wsSubscribeOp, l) }, len(subs)),
	)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	return common.AppendError(
		e.ParallelChanOp(ctx, subs.Public(), func(ctx context.Context, l subscription.List) error { return e.manageSubs(ctx, wsUnsubscribeOp, l) }, len(subs)),
		e.ParallelChanOp(ctx, subs.Private(), func(ctx context.Context, l subscription.List) error { return e.manageSubs(ctx, wsUnsubscribeOp, l) }, len(subs)),
	)
}

func (e *Exchange) manageSubs(ctx context.Context, op string, subs subscription.List) error {
	req := WebsocketRequest{
		Command: op,
	}
	exp := map[string]*subscription.Subscription{}
	for _, s := range subs {
		req.Arguments = append(req.Arguments, s.QualifiedChannel)
		exp[s.QualifiedChannel] = s
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resps, errs := e.Websocket.Conn.SendMessageReturnResponses(ctx, request.Unset, string(reqJSON), req, len(subs))
	for _, resp := range resps {
		if errMsg, _ := jsonparser.GetString(resp, "error"); errMsg != "" {
			errs = common.AppendError(errs, errors.New(errMsg))
		} else {
			chanName, err := jsonparser.GetString(resp, op)
			if err != nil {
				errs = common.AppendError(errs, err)
			}
			s, ok := exp[chanName]
			if !ok {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", subscription.ErrNotFound, chanName))
			} else {
				if op == wsSubscribeOp {
					errs = common.AppendError(errs, e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s))
				} else {
					errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s))
				}
			}
		}
	}
	return errs
}

func (e *Exchange) websocketSendAuth(ctx context.Context) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}

	timestamp := time.Now().Add(time.Hour * 1).Unix()
	timestampStr := strconv.FormatInt(timestamp, 10)
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte("GET/realtime"+timestampStr), []byte(creds.Secret))
	if err != nil {
		return err
	}

	req := WebsocketRequest{
		Command:   "authKeyExpires",
		Arguments: []any{creds.Key, timestamp, hex.EncodeToString(hmac)},
	}

	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.Command, req)
	if err != nil {
		return err
	}
	if errMsg, _ := jsonparser.GetString(resp, "error"); errMsg != "" {
		return errors.New(errMsg)
	}
	if e.Verbose {
		log.Debugf(log.ExchangeSys, "%s websocket: Successfully authenticated websocket connection", e.Name)
	}
	return nil
}

// GetActionFromString matches a string action to an internal action.
func (e *Exchange) GetActionFromString(s string) (orderbook.ActionType, error) {
	switch s {
	case "update":
		return orderbook.UpdateAction, nil
	case "delete":
		return orderbook.DeleteAction, nil
	case "insert":
		return orderbook.InsertAction, nil
	case "update/insert":
		return orderbook.UpdateOrInsertAction, nil
	}
	return 0, fmt.Errorf("%s %w", s, orderbook.ErrInvalidAction)
}

// channelName returns the correct channel name for the asset
func channelName(s *subscription.Subscription, a asset.Item) string {
	switch s.Channel {
	case subscription.OrderbookChannel:
		if a == asset.Index {
			return "" // There are no L2 orderbook for index assets
		}
		return bitmexWSOrderbookL2
	case subscription.AllTradesChannel:
		return bitmexWSTrade
	}
	return s.Channel
}

const subTplText = `
{{- if $.S.Asset }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- with $name := channelName $.S $asset }}
			{{- range $i, $p := $pairs }}
				{{- $name -}} : {{- $p }}
				{{- $.PairSeparator }}
			{{- end }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- channelName $.S $.S.Asset }}
{{- end }}
`
