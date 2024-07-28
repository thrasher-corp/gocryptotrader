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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// WebsocketRequest defines a websocket request
type WebsocketRequest struct {
	App     string           `json:"app,omitempty"`
	Time    int64            `json:"time,omitempty"`
	ID      int64            `json:"id,omitempty"`
	Channel string           `json:"channel"`
	Event   string           `json:"event"`
	Payload WebsocketPayload `json:"payload"`
}

// WebsocketPayload defines an individualised websocket payload
type WebsocketPayload struct {
	APIKey       string `json:"api_key,omitempty"`
	Signature    string `json:"signature,omitempty"`
	Timestamp    string `json:"timestamp,omitempty"`
	RequestID    string `json:"req_id,omitempty"`
	RequestParam []byte `json:"req_param,omitempty"`
}

// WebsocketResponse defines a websocket response
type WebsocketResponse struct {
	RequestID   string          `json:"req_id"`
	Acknowleged bool            `json:"ack"`
	Header      WebsocketHeader `json:"req_header"`
	Data        json.RawMessage `json:"data"`
}

// WebsocketHeader defines a websocket header
type WebsocketHeader struct {
	ResponseTime int64  `json:"response_time"`
	Status       string `json:"status"`
	Channel      string `json:"channel"`
	Event        string `json:"event"`
	ClientID     string `json:"client_id"`
}

type WebsocketErrors struct {
	Label   string `json:"label"`
	Message string `json:"message"`
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

// LoginResult defines a login result
type LoginResult struct {
	APIKey string `json:"api_key"`
	UID    string `json:"uid"`
}

// WebsocketLogin logs in to the websocket
func (g *Gateio) WebsocketLogin(ctx context.Context, a asset.Item) (*LoginResult, error) {

	route, err := g.GetRoute(a)
	if err != nil {
		return nil, err
	}
	var resp *LoginResult
	err = g.SendWebsocketRequest(ctx, "spot.login", route, nil, &resp)
	return resp, err
}

// OrderPlace
// OrderCancel
// OrderCancelAllByIDList
// OrderCancelAllByPair
// OrderAmend
// OrderStatus

func (g *Gateio) OrderPlace(ctx context.Context, batch []order.Submit) string {
	return ""
}

// SendWebsocketRequest sends a websocket request to the exchange
func (g *Gateio) SendWebsocketRequest(ctx context.Context, channel, route string, params []byte, result any) error {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}

	tn := time.Now()
	msg := "api\n" + channel + "\n" + string(params) + "\n" + strconv.FormatInt(tn.Unix(), 10)
	mac := hmac.New(sha512.New, []byte(creds.Secret))
	if _, err := mac.Write([]byte(msg)); err != nil {
		return err
	}
	signature := hex.EncodeToString(mac.Sum(nil))

	outbound := WebsocketRequest{
		Time:    tn.Unix(),
		Channel: channel,
		Event:   "api",
		Payload: WebsocketPayload{
			RequestID:    strconv.FormatInt(tn.UnixNano(), 10),
			APIKey:       creds.Key,
			RequestParam: params,
			Signature:    signature,
			Timestamp:    strconv.FormatInt(tn.Unix(), 10),
		},
	}

	var inbound WebsocketResponse
	err = g.Websocket.SendRequest(ctx, route, outbound.Payload.RequestID, &outbound, &inbound)
	if err != nil {
		return err
	}
	if fail, dataType, _, _ := jsonparser.Get(inbound.Data, "errs"); dataType != jsonparser.NotExist {
		var wsErr WebsocketErrors
		err := json.Unmarshal(fail, &wsErr)
		if err != nil {
			return err
		}
		return fmt.Errorf("gateio websocket error: %s %s", wsErr.Label, wsErr.Message)
	}

	nested, _, _, err := jsonparser.Get(inbound.Data, "result")
	if err != nil {
		return err
	}

	return json.Unmarshal(nested, result)
}
