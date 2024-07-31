package gateio

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// WebsocketRequest defines a websocket request
type WebsocketRequest struct {
	Time    int64            `json:"time,omitempty"`
	ID      int64            `json:"id,omitempty"`
	Channel string           `json:"channel"`
	Event   string           `json:"event"`
	Payload WebsocketPayload `json:"payload"`
}

// WebsocketPayload defines an individualised websocket payload
type WebsocketPayload struct {
	RequestID string `json:"req_id,omitempty"`
	// APIKey and signature are only required in the initial login request
	// which is done when the connection is established.
	APIKey       string          `json:"api_key,omitempty"`
	Timestamp    string          `json:"timestamp,omitempty"`
	Signature    string          `json:"signature,omitempty"`
	RequestParam json.RawMessage `json:"req_param,omitempty"`
}

// // WebsocketResponse defines a websocket response
// type WebsocketResponse struct {
// 	RequestID     string `json:"req_id"`
// 	APIKey        string `json:"api_key"`
// 	Timestamp     string `json:"timestamp"`
// 	Signature     string `json:"signature"`
// 	TraceID       string `json:"trace_id"`
// 	RequestHeader struct {
// 		TraceID string `json:"trace_id"`
// 	} `json:"req_header"`
// 	Acknowleged bool `json:"ack"`
// 	// Header      WebsocketHeader `json:"header"`
// 	RequestParam json.RawMessage `json:"req_param"`
// 	Data         json.RawMessage `json:"data"`
// }

// // WebsocketHeader defines a websocket header
// type WebsocketHeader struct {
// 	ResponseTime int64  `json:"response_time"`
// 	Status       string `json:"status"`
// 	Channel      string `json:"channel"`
// 	Event        string `json:"event"`
// 	ClientID     string `json:"client_id"`
// }

type WebsocketErrors struct {
	Errors struct {
		Label   string `json:"label"`
		Message string `json:"message"`
	} `json:"errs"`
}

// GetRoute returns the route for a websocket request, this is a POC
// for the websocket wrapper.
func (g *Gateio) GetRoute(a asset.Item) (string, error) {
	switch a {
	case asset.Spot:
		return gateioWebsocketEndpoint, nil
	default:
		return "", common.ErrNotYetImplemented
	}
}

type Header struct {
	ResponseTime Time   `json:"response_time"`
	Status       string `json:"status"`
	Channel      string `json:"channel"`
	Event        string `json:"event"`
	ClientID     string `json:"client_id"`
	ConnectionID string `json:"conn_id"`
	TraceID      string `json:"trace_id"`
}

// LoginResult defines a login result
type LoginResult struct {
	APIKey string `json:"api_key"`
	UID    string `json:"uid"`
}

type LoginResponse struct {
	Header Header          `json:"header"`
	Data   json.RawMessage `json:"data"`
}

// WebsocketLogin logs in to the websocket
func (g *Gateio) WebsocketLogin(ctx context.Context, conn stream.Connection, channel string) (*LoginResult, error) {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}

	tn := time.Now()
	msg := "api\n" + channel + "\n" + string([]byte(nil)) + "\n" + strconv.FormatInt(tn.Unix(), 10)
	mac := hmac.New(sha512.New, []byte(creds.Secret))
	if _, err := mac.Write([]byte(msg)); err != nil {
		return nil, err
	}
	signature := hex.EncodeToString(mac.Sum(nil))

	outbound := WebsocketRequest{
		Time:    tn.Unix(),
		Channel: channel,
		Event:   "api",
		Payload: WebsocketPayload{
			RequestID: strconv.FormatInt(tn.UnixNano(), 10),
			APIKey:    creds.Key,
			Signature: signature,
			Timestamp: strconv.FormatInt(tn.Unix(), 10),
		},
	}

	resp, err := conn.SendMessageReturnResponse(outbound.Payload.RequestID, outbound)
	if err != nil {
		return nil, err
	}

	var inbound LoginResponse
	err = json.Unmarshal(resp, &inbound)
	if err != nil {
		return nil, err
	}

	if fail, dataType, _, _ := jsonparser.Get(inbound.Data, "errs"); dataType != jsonparser.NotExist {
		var wsErr WebsocketErrors
		err := json.Unmarshal(fail, &wsErr.Errors)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("gateio websocket error: %s %s", wsErr.Errors.Label, wsErr.Errors.Message)
	}

	var result struct {
		Result LoginResult `json:"result"`
	}
	err = json.Unmarshal(inbound.Data, &result)
	return &result.Result, err
}

// OrderPlace
// OrderCancel
// OrderCancelAllByIDList
// OrderCancelAllByPair
// OrderAmend
// OrderStatus

// WebsocketOrder defines a websocket order
type WebsocketOrder struct {
	Text         string `json:"text"`
	CurrencyPair string `json:"currency_pair,omitempty"`
	Type         string `json:"type,omitempty"`
	Account      string `json:"account,omitempty"`
	Side         string `json:"side,omitempty"`
	Amount       string `json:"amount,omitempty"`
	Price        string `json:"price,omitempty"`
	TimeInForce  string `json:"time_in_force,omitempty"`
	Iceberg      string `json:"iceberg,omitempty"`
	AutoBorrow   bool   `json:"auto_borrow,omitempty"`
	AutoRepay    bool   `json:"auto_repay,omitempty"`
	StpAct       string `json:"stp_act,omitempty"`
}

type WebscocketOrderResponse struct {
	ReqID        string `json:"req_id"`
	RequestParam any    `json:"req_param"`
	APIKey       string `json:"api_key"`
	Timestamp    string `json:"timestamp"`
	Signature    string `json:"signature"`
}

type WebsocketOrderParamResponse struct {
	Text         string `json:"text"`
	CurrencyPair string `json:"currency_pair"`
	Type         string `json:"type"`
	Account      string `json:"account"`
	Side         string `json:"side"`
	Amount       string `json:"amount"`
	Price        string `json:"price"`
}

var errBatchSliceEmpty = fmt.Errorf("batch cannot be empty")

// WebsocketOrderPlace places an order via the websocket connection. You can
// send multiple orders in a single request. But only for one asset route.
// So this can only batch spot orders or futures orders, not both.
func (g *Gateio) WebsocketOrderPlace(ctx context.Context, batch []WebsocketOrder, a asset.Item) ([]WebsocketOrderParamResponse, error) {
	if len(batch) == 0 {
		return nil, errBatchSliceEmpty
	}

	for i := range batch {
		if batch[i].Text == "" {
			// For some reason the API requires a text field, or it will be
			// rejected in the second response. This is a workaround.
			batch[i].Text = "t-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		}
		if batch[i].CurrencyPair == "" {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if batch[i].Side == "" {
			return nil, order.ErrSideIsInvalid
		}
		if batch[i].Amount == "" {
			return nil, errInvalidAmount
		}
		if batch[i].Type == "limit" && batch[i].Price == "" {
			return nil, errInvalidPrice
		}
	}

	route, err := g.GetRoute(a)
	if err != nil {
		return nil, err
	}

	var resp WebscocketOrderResponse
	if len(batch) == 1 {
		var incoming WebsocketOrderParamResponse
		resp.RequestParam = &incoming

		batchBytes, err := json.Marshal(batch[0])
		if err != nil {
			return nil, err
		}

		err = g.SendWebsocketRequest(ctx, "spot.order_place", route, batchBytes, &resp)
		return []WebsocketOrderParamResponse{incoming}, err
	}

	var incoming []WebsocketOrderParamResponse
	resp.RequestParam = &incoming
	err = g.SendWebsocketRequest(ctx, "spot.order_place", route, []byte{}, &resp)
	return incoming, err
}

// SendWebsocketRequest sends a websocket request to the exchange
func (g *Gateio) SendWebsocketRequest(ctx context.Context, channel, route string, params json.RawMessage, result any) error {
	// creds, err := g.GetCredentials(ctx)
	// if err != nil {
	// 	return err
	// }

	tn := time.Now()
	// msg := "api\n" + channel + "\n" + string(params) + "\n" + strconv.FormatInt(tn.Unix(), 10)
	// mac := hmac.New(sha512.New, []byte(creds.Secret))
	// if _, err := mac.Write([]byte(msg)); err != nil {
	// 	return err
	// }
	// signature := hex.EncodeToString(mac.Sum(nil))

	mainPayload := WebsocketPayload{
		// This request ID associated with the payload is the match to the
		// response.
		RequestID: strconv.FormatInt(tn.UnixNano(), 10),
		// APIKey:       creds.Key,
		RequestParam: params,
		// Signature:    signature,
		Timestamp: strconv.FormatInt(tn.Unix(), 10),
	}

	outbound := WebsocketRequest{
		Time:    tn.Unix(),
		Channel: channel,
		Event:   "api",
		Payload: mainPayload,
	}

	var inbound GeneralWebsocketResponse
	err := g.Websocket.SendRequest(ctx, route, outbound.Payload.RequestID, &outbound, &inbound)
	if err != nil {
		return err
	}

	if inbound.Header.Status != "200" {
		var wsErr WebsocketErrors
		err = json.Unmarshal(inbound.Data, &wsErr)
		if err != nil {
			return err
		}
		return fmt.Errorf("gateio websocket error: %s %s", wsErr.Errors.Label, wsErr.Errors.Message)
	}

	return json.Unmarshal(inbound.Data, result)
}

type GeneralWebsocketResponse struct {
	Header Header          `json:"header"`
	Data   json.RawMessage `json:"data"`
}
