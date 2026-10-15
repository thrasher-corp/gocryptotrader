package htx

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// wsHandleV2ping handles v2 style pings, currently only used with private connections
func (e *Exchange) wsHandleV2ping(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if err := common.NilGuard(conn); err != nil {
		return err
	}
	ts, err := jsonparser.GetInt(respRaw, "data", "ts")
	if err != nil {
		return fmt.Errorf("error getting ts from auth ping: %w", err)
	}
	if err := conn.SendJSONMessage(ctx, request.Unset, json.RawMessage(`{"action":"pong","data":{"ts":`+strconv.FormatInt(ts, 10)+`}}`)); err != nil {
		return fmt.Errorf("error sending auth pong response: %w", err)
	}
	return nil
}

func (e *Exchange) wsGenerateSignature(conn websocket.Connection, creds *accounts.Credentials, timestamp string) ([]byte, error) {
	if err := common.NilGuard(conn, creds); err != nil {
		return nil, err
	}
	signatureHost, err := getSignatureHost(conn.GetURL())
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Set("accessKey", creds.Key)
	values.Set("signatureMethod", signatureMethod)
	values.Set("signatureVersion", signatureVersion)
	values.Set("timestamp", timestamp)
	payload := http.MethodGet + "\n" + signatureHost + "\n" + wsPrivatePath + "\n" + values.Encode()
	return crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(creds.Secret))
}

func (e *Exchange) wsLogin(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}

	ts := time.Now().UTC().Format(wsDateTimeFormatting)
	hmac, err := e.wsGenerateSignature(conn, creds, ts)
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
	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, wsAuthChannel, req)
	if err != nil {
		return err
	}
	if err := getErrResp(resp); err != nil {
		return err
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return nil
}

func (e *Exchange) wsHandleMyAccountMsg(ctx context.Context, respRaw []byte) error {
	u := &wsAccountUpdateMsg{}
	if err := json.Unmarshal(respRaw, u); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, u.Data)
}

func (e *Exchange) wsHandleMyOrdersMsg(ctx context.Context, s *subscription.Subscription, respRaw []byte) error {
	var msg wsOrderUpdateMsg
	if err := json.Unmarshal(respRaw, &msg); err != nil {
		return err
	}
	o := msg.Data
	p, err := e.CurrencyPairs.Match(o.Symbol, s.Asset)
	if err != nil {
		return err
	}
	d := &order.Detail{
		ClientOrderID:   o.ClientOrderID,
		Price:           o.Price.Float64(),
		Amount:          o.Size.Float64(),
		ExecutedAmount:  o.ExecutedAmount.Float64(),
		RemainingAmount: o.RemainingAmount.Float64(),
		Exchange:        e.Name,
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
			Exchange: e.Name,
			OrderID:  d.OrderID,
			Err:      err,
		}
	}
	if o.Side == order.UnknownSide {
		d.Side, err = stringToOrderSide(o.OrderType)
		if err != nil {
			return &order.ClassificationError{
				Exchange: e.Name,
				OrderID:  d.OrderID,
				Err:      err,
			}
		}
	}
	if o.OrderType != "" {
		d.Type, err = stringToOrderType(o.OrderType)
		if err != nil {
			return &order.ClassificationError{
				Exchange: e.Name,
				OrderID:  d.OrderID,
				Err:      err,
			}
		}
	}
	if err := e.Websocket.DataHandler.Send(ctx, d); err != nil {
		return err
	}
	if o.ErrCode != 0 {
		return fmt.Errorf("%w: error with order %q: %s (%v)", errAPIResponse, o.ClientOrderID, o.ErrMessage, o.ErrCode)
	}
	return nil
}

func (e *Exchange) wsHandleMyTradesMsg(ctx context.Context, s *subscription.Subscription, respRaw []byte) error {
	var msg wsTradeUpdateMsg
	if err := json.Unmarshal(respRaw, &msg); err != nil {
		return err
	}
	t := msg.Data
	p, err := e.CurrencyPairs.Match(t.Symbol, s.Asset)
	if err != nil {
		return err
	}
	d := &order.Detail{
		ClientOrderID: t.ClientOrderID,
		Price:         t.OrderPrice.Float64(),
		Amount:        t.OrderSize.Float64(),
		Exchange:      e.Name,
		Side:          t.Side,
		AssetType:     s.Asset,
		Pair:          p,
		Date:          t.OrderCreateTime.Time(),
		LastUpdated:   t.TradeTime.Time(),
		OrderID:       strconv.FormatInt(t.OrderID, 10),
	}
	if d.Status, err = order.StringToOrderStatus(t.OrderStatus); err != nil {
		return &order.ClassificationError{
			Exchange: e.Name,
			OrderID:  d.OrderID,
			Err:      err,
		}
	}
	if t.Side == order.UnknownSide {
		d.Side, err = stringToOrderSide(t.OrderType)
		if err != nil {
			return &order.ClassificationError{
				Exchange: e.Name,
				OrderID:  d.OrderID,
				Err:      err,
			}
		}
	}
	if t.OrderType != "" {
		d.Type, err = stringToOrderType(t.OrderType)
		if err != nil {
			return &order.ClassificationError{
				Exchange: e.Name,
				OrderID:  d.OrderID,
				Err:      err,
			}
		}
	}
	d.Trades = []order.TradeHistory{
		{
			Price:     t.TradePrice.Float64(),
			Amount:    t.TradeVolume.Float64(),
			Fee:       t.TransactFee.Float64(),
			FeeAsset:  t.FeeCurrency,
			Exchange:  e.Name,
			TID:       strconv.FormatInt(t.TradeID, 10),
			Type:      d.Type,
			Side:      d.Side,
			IsMaker:   !t.IsTaker,
			Timestamp: t.TradeTime.Time(),
		},
	}
	return e.Websocket.DataHandler.Send(ctx, d)
}

const (
	wsPrivatePath      = "/ws/v2"
	wsMyOrdersChannel  = "orders#*"
	wsMyTradesChannel  = "trade.clearing#*#1" // 0=Only trade events, 1=Trade and Cancellation events
	wsMyAccountChannel = "accounts.update#2"  // 0=Only balance, 1=Balance or Available, 2=Balance and Available when either change
	signatureVersion   = "2.1"
	wsRequestOp        = "req"
)
