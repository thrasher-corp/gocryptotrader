package gateio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func connectGateioWithMockedWebsocket(t *testing.T, wsHandler mockws.WsMockFunc) *Exchange {
	t.Helper()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, wsHandler))
	t.Cleanup(server.Close)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ex.Websocket = websocket.NewManager()
	exchCfg := ex.Config
	require.NotNil(t, exchCfg)
	exchCfg.Features.Subscriptions = subscription.List{}
	exchCfg.WebsocketTrafficTimeout = time.Hour
	exchCfg.ConnectionMonitorDelay = time.Hour
	require.NoError(t, ex.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exchCfg,
		Features:                     &ex.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement: true,
	}))

	setupConn := func(filter any) {
		require.NoError(t, ex.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
			URL:                  wsURL,
			ResponseCheckTimeout: exchCfg.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exchCfg.WebsocketResponseMaxLimit,
			Connector: func(ctx context.Context, conn websocket.Connection) error {
				return conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil)
			},
			Subscriber: func(context.Context, websocket.Connection, subscription.List) error { return nil },
			Unsubscriber: func(context.Context, websocket.Connection, subscription.List) error {
				return nil
			},
			GenerateSubscriptions: func() (subscription.List, error) { return subscription.List{}, nil },
			Handler: func(_ context.Context, conn websocket.Connection, incoming []byte) error {
				var m struct {
					RequestID string `json:"request_id"`
					ID        int64  `json:"id"`
				}
				if err := json.Unmarshal(incoming, &m); err != nil {
					return err
				}
				if m.RequestID != "" {
					return conn.RequireMatchWithData(m.RequestID, incoming)
				}
				if m.ID != 0 {
					return conn.RequireMatchWithData(m.ID, incoming)
				}
				return nil
			},
			MessageFilter: filter,
		}))
	}

	setupConn(asset.Spot)
	setupConn(asset.USDTMarginedFutures)
	setupConn(asset.CoinMarginedFutures)
	setupConn(asset.DeliveryFutures)

	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.Connect(t.Context()))
	t.Cleanup(func() {
		_ = ex.Websocket.Shutdown()
	})
	return ex
}

func gateioOrderWsMock(_ testing.TB, p []byte, c *gws.Conn) error {
	var req struct {
		Channel string `json:"channel"`
		Payload struct {
			RequestID string `json:"req_id"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(p, &req); err != nil {
		return err
	}

	if req.Channel == "spot.order_place" || req.Channel == "futures.order_place" {
		if err := c.WriteMessage(gws.TextMessage, []byte(`{"request_id":"`+req.Payload.RequestID+`","ack":true}`)); err != nil {
			return err
		}
	}

	var response string
	switch req.Channel {
	case "spot.order_place":
		response = `{"request_id":"` + req.Payload.RequestID + `","header":{"status":"200"},"data":{"result":{"id":"spot-order","side":"buy","type":"limit","time_in_force":"gtc","currency_pair":"BTC_USDT","account":"spot","amount":"1","left":"1","price":"100","create_time_ms":"1700000000000","update_time_ms":"1700000000000","text":"t-spot"}}}`
	case "futures.order_place":
		response = `{"request_id":"` + req.Payload.RequestID + `","header":{"status":"200"},"data":{"result":{"id":12345,"contract":"BTC_USDT","size":1,"left":1,"price":"100","tif":"gtc","create_time":"1700000000","update_time":"1700000000","status":"open"}}}`
	case "spot.order_amend":
		response = `{"request_id":"` + req.Payload.RequestID + `","header":{"status":"200"},"data":{"result":{"id":"spot-amended","status":"open"}}}`
	case "futures.order_amend":
		response = `{"request_id":"` + req.Payload.RequestID + `","header":{"status":"200"},"data":{"result":{"id":999,"status":"open"}}}`
	case "spot.order_cancel", "futures.order_cancel":
		response = `{"request_id":"` + req.Payload.RequestID + `","header":{"status":"200"},"data":{"result":{"status":"cancelled"}}}`
	default:
		response = `{"request_id":"` + req.Payload.RequestID + `","header":{"status":"500"},"data":{"errs":{"label":"bad_channel","message":"unsupported channel"}}}`
	}
	return c.WriteMessage(gws.TextMessage, []byte(response))
}

func gateioAmendStatusWsMock(status string) mockws.WsMockFunc {
	return func(tb testing.TB, p []byte, c *gws.Conn) error {
		tb.Helper()
		var req struct {
			Channel string `json:"channel"`
			Payload struct {
				RequestID string `json:"req_id"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(p, &req); err != nil {
			return err
		}
		switch req.Channel {
		case "spot.order_amend":
			return c.WriteMessage(gws.TextMessage, []byte(`{"request_id":"`+req.Payload.RequestID+`","header":{"status":"200"},"data":{"result":{"id":"spot-amended","status":"`+status+`"}}}`))
		case "futures.order_amend":
			return c.WriteMessage(gws.TextMessage, []byte(`{"request_id":"`+req.Payload.RequestID+`","header":{"status":"200"},"data":{"result":{"id":999,"status":"`+status+`"}}}`))
		default:
			return gateioOrderWsMock(tb, p, c)
		}
	}
}

func gateioOrderWsErrorMock(_ testing.TB, p []byte, c *gws.Conn) error {
	var req struct {
		Channel string `json:"channel"`
		Payload struct {
			RequestID string `json:"req_id"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(p, &req); err != nil {
		return err
	}
	return c.WriteMessage(gws.TextMessage, []byte(`{"request_id":"`+req.Payload.RequestID+`","header":{"status":"500"},"data":{"errs":{"label":"mock_error","message":"request failed"}}}`))
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()

	ex := connectGateioWithMockedWebsocket(t, gateioOrderWsMock)

	spotResp, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  ex.Name,
		AssetType: asset.Spot,
		Pair:      getPair(t, asset.Spot),
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     100,
	})
	require.NoError(t, err)
	require.Equal(t, "spot-order", spotResp.OrderID)

	futuresResp, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  ex.Name,
		AssetType: asset.USDTMarginedFutures,
		Pair:      getPair(t, asset.USDTMarginedFutures),
		Side:      order.Long,
		Type:      order.Limit,
		Amount:    1,
		Price:     100,
	})
	require.NoError(t, err)
	require.Equal(t, "12345", futuresResp.OrderID)

	deliveryResp, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  ex.Name,
		AssetType: asset.DeliveryFutures,
		Pair:      getPair(t, asset.DeliveryFutures),
		Side:      order.Long,
		Type:      order.Limit,
		Amount:    1,
		Price:     100,
	})
	require.NoError(t, err)
	require.Equal(t, "12345", deliveryResp.OrderID)

	_, err = ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  ex.Name,
		AssetType: asset.Options,
		Pair:      getPair(t, asset.Options),
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()

	ex := connectGateioWithMockedWebsocket(t, gateioOrderWsMock)

	spotResp, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "spot-1",
		AssetType: asset.Spot,
		Pair:      getPair(t, asset.Spot),
		Amount:    1,
		Price:     101,
	})
	require.NoError(t, err)
	require.Equal(t, "spot-amended", spotResp.OrderID)

	futuresResp, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "futures-1",
		AssetType: asset.USDTMarginedFutures,
		Pair:      getPair(t, asset.USDTMarginedFutures),
		Side:      order.Buy,
		Amount:    0.5,
		Price:     101,
	})
	require.NoError(t, err)
	require.Equal(t, "999", futuresResp.OrderID)

	shortSizes := make(chan types.Number, 1)
	short := connectGateioWithMockedWebsocket(t, func(tb testing.TB, payload []byte, conn *gws.Conn) error {
		tb.Helper()
		var req WebsocketRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			return err
		}
		if req.Channel == "futures.order_amend" {
			assert.Contains(tb, string(req.Payload.RequestParam), `"size":"-0.5"`, "futures amendment should encode decimal size as a string")
			var amend WebsocketFuturesAmendOrder
			if err := json.Unmarshal(req.Payload.RequestParam, &amend); err != nil {
				return err
			}
			shortSizes <- amend.Size
		}
		return gateioOrderWsMock(tb, payload, conn)
	})
	_, err = short.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "futures-short",
		AssetType: asset.USDTMarginedFutures,
		Pair:      getPair(t, asset.USDTMarginedFutures),
		Side:      order.Sell,
		Amount:    0.5,
		Price:     101,
	})
	require.NoError(t, err)
	assert.Equal(t, types.Number(-0.5), <-shortSizes, "short futures amendment should send a negative decimal size")

	closed := connectGateioWithMockedWebsocket(t, gateioAmendStatusWsMock("cancelled"))
	closedResp, err := closed.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "spot-closed",
		AssetType: asset.Spot,
		Pair:      getPair(t, asset.Spot),
		Amount:    1,
	})
	require.NoError(t, err)
	require.Equal(t, order.Cancelled, closedResp.Status)
	closedResp, err = closed.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "futures-closed",
		AssetType: asset.USDTMarginedFutures,
		Pair:      getPair(t, asset.USDTMarginedFutures),
		Side:      order.Buy,
		Amount:    1,
	})
	require.NoError(t, err)
	require.Equal(t, order.Cancelled, closedResp.Status)

	invalidStatus := connectGateioWithMockedWebsocket(t, gateioAmendStatusWsMock("not-a-status"))
	_, err = invalidStatus.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "spot-invalid-status",
		AssetType: asset.Spot,
		Pair:      getPair(t, asset.Spot),
		Amount:    1,
	})
	require.Error(t, err)
	_, err = invalidStatus.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "futures-invalid-status",
		AssetType: asset.USDTMarginedFutures,
		Pair:      getPair(t, asset.USDTMarginedFutures),
		Side:      order.Buy,
		Amount:    1,
	})
	require.Error(t, err)

	wsError := connectGateioWithMockedWebsocket(t, gateioOrderWsErrorMock)
	_, err = wsError.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "spot-error",
		AssetType: asset.Spot,
		Pair:      getPair(t, asset.Spot),
		Amount:    1,
	})
	require.Error(t, err)
	_, err = wsError.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "futures-error",
		AssetType: asset.USDTMarginedFutures,
		Pair:      getPair(t, asset.USDTMarginedFutures),
		Side:      order.Buy,
		Amount:    1,
	})
	require.Error(t, err)

	badFormat := connectGateioWithMockedWebsocket(t, gateioOrderWsMock)
	badFormat.CurrencyPairs.UseGlobalFormat = true
	badFormat.CurrencyPairs.RequestFormat = nil
	_, err = badFormat.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "bad-format",
		AssetType: asset.Spot,
		Pair:      getPair(t, asset.Spot),
		Amount:    1,
	})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil)

	_, err = ex.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "1",
		AssetType: asset.Binary,
		Pair:      currency.NewBTCUSD(),
		Amount:    1,
	})
	require.ErrorIs(t, err, common.ErrNotYetImplemented)

	_, err = ex.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      getPair(t, asset.Options),
		Amount:    1,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	_, err = ex.WebsocketModifyOrder(t.Context(), &order.Modify{})
	require.ErrorIs(t, err, order.ErrPairIsEmpty)
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()

	ex := connectGateioWithMockedWebsocket(t, gateioOrderWsMock)

	err := ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "spot-1",
		AssetType: asset.Spot,
		Pair:      getPair(t, asset.Spot),
	})
	require.NoError(t, err)

	err = ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "futures-1",
		AssetType: asset.USDTMarginedFutures,
		Pair:      getPair(t, asset.USDTMarginedFutures),
	})
	require.NoError(t, err)

	wsError := connectGateioWithMockedWebsocket(t, gateioOrderWsErrorMock)
	err = wsError.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "spot-error",
		AssetType: asset.Spot,
		Pair:      getPair(t, asset.Spot),
	})
	require.Error(t, err)
	err = wsError.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "futures-error",
		AssetType: asset.USDTMarginedFutures,
		Pair:      getPair(t, asset.USDTMarginedFutures),
	})
	require.Error(t, err)

	badFormat := connectGateioWithMockedWebsocket(t, gateioOrderWsMock)
	spotPair := getPair(t, asset.Spot)
	futuresPair := getPair(t, asset.USDTMarginedFutures)
	badFormat.CurrencyPairs.UseGlobalFormat = true
	badFormat.CurrencyPairs.RequestFormat = nil
	err = badFormat.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "spot-bad-format",
		AssetType: asset.Spot,
		Pair:      spotPair,
	})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil)
	err = badFormat.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "futures-bad-format",
		AssetType: asset.USDTMarginedFutures,
		Pair:      futuresPair,
	})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil)

	err = ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      getPair(t, asset.Options),
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	err = ex.WebsocketCancelOrder(t.Context(), &order.Cancel{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	err = ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Binary,
		Pair:      currency.NewBTCUSD(),
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)
}
