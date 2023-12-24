package binance

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

const (
	binanceWebsocketAPIURL = "wss://ws-api.binance.com:443/ws-api/v3"
)

// websocket request status codes
var websocketStatusCodes = map[int64]string{
	400: "request failed",
	403: "request blocked",
	409: "request partially failed but also partially succeeded",
	418: "auto-banned for repeated violation of rate limits",
	419: "exceeded API request rate limit",
}

// GetWsOrderbook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (b *Binance) GetWsOrderbook(obd OrderBookDataRequestParams) (*OrderBook, error) {
	if err := b.CheckLimit(obd.Limit); err != nil {
		return nil, err
	}
	var resp OrderBookData
	if err := b.SendWsRequest("depth", obd, &resp); err != nil {
		return nil, err
	}
	orderbook := OrderBook{
		Bids:         make([]OrderbookItem, len(resp.Bids)),
		Asks:         make([]OrderbookItem, len(resp.Asks)),
		LastUpdateID: resp.LastUpdateID,
	}
	for x := range resp.Bids {
		orderbook.Bids[x] = OrderbookItem{Price: resp.Bids[x][0].Float64(), Quantity: resp.Bids[x][1].Float64()}
	}
	for x := range resp.Asks {
		orderbook.Asks[x] = OrderbookItem{Price: resp.Asks[x][0].Float64(), Quantity: resp.Asks[x][1].Float64()}
	}
	return &orderbook, nil
}

// SendWsRequest sends websocket endpoint request through the websocket connection
func (b *Binance) SendWsRequest(method string, param, result interface{}) error {
	if !b.Websocket.IsConnected() {
		return stream.ErrNotConnected
	}
	input := &struct {
		ID     string      `json:"id"`
		Method string      `json:"method"`
		Params interface{} `json:"params"`
	}{
		ID:     strconv.FormatInt(b.Websocket.AuthConn.GenerateMessageID(false), 10),
		Method: method,
		Params: param,
	}
	respRaw, err := b.Websocket.AuthConn.SendMessageReturnResponse(input.ID, input)
	if err != nil {
		return err
	}
	resp := &struct {
		ID     string      `json:"id"`
		Status int64       `json:"status"`
		Result interface{} `json:"result"`
	}{
		Result: result,
	}
	err = json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	switch resp.Status {
	case 200:
		return nil
	case 400, 403, 409, 418, 419:
		return errors.New(websocketStatusCodes[resp.Status])
	default:
		if resp.Status >= 500 {
			return errors.New("internal server error")
		}
		return errors.New("request failed")
	}
}
