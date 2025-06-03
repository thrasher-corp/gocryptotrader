package kraken

import (
	"fmt"
	"strings"
	"testing"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func mockWsServer(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	event, err := jsonparser.GetUnsafeString(msg, "event")
	if err != nil {
		return err
	}
	switch event {
	case krakenWsCancelOrder:
		return mockWsCancelOrders(tb, msg, w)
	case krakenWsAddOrder:
		return mockWsAddOrder(tb, msg, w)
	}
	return nil
}

func mockWsCancelOrders(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	var req WsCancelOrderRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	resp := WsCancelOrderResponse{
		Event:     krakenWsCancelOrderStatus,
		Status:    "ok",
		RequestID: req.RequestID,
		Count:     int64(len(req.TransactionIDs)),
	}
	if len(req.TransactionIDs) == 0 || strings.Contains(req.TransactionIDs[0], "FISH") { // Reject anything that smells suspicious
		resp.Status = "error"
		resp.ErrorMessage = "[EOrder:Unknown order]"
	}
	msg, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return w.WriteMessage(gws.TextMessage, msg)
}

func mockWsAddOrder(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	var req WsAddOrderRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}

	assert.Equal(tb, "buy", req.OrderSide, "OrderSide should be correct")
	assert.Equal(tb, "limit", req.OrderType, "OrderType should be correct")
	assert.Equal(tb, "XBT/USD", req.Pair, "Pair should be correct")
	assert.Equal(tb, 80000.0, req.Price, "Pair should be correct")

	resp := WsAddOrderResponse{
		Event:         krakenWsAddOrderStatus,
		Status:        "ok",
		RequestID:     req.RequestID,
		TransactionID: "ONPNXH-KMKMU-F4MR5V",
		Description:   fmt.Sprintf("%s %.f %s @ %s %.f", req.OrderSide, req.Volume, req.Pair, req.OrderSide, req.Price),
	}
	msg, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return w.WriteMessage(gws.TextMessage, msg)
}
