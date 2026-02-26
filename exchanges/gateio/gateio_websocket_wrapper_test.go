package gateio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testpath "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func loadGateioExchangeConfig(t *testing.T, exchangeName string) *config.Exchange {
	t.Helper()

	root, err := testpath.RootPathFromCWD()
	require.NoError(t, err)

	cfg := &config.Config{}
	require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true))

	exchCfg, err := cfg.GetExchangeConfig(exchangeName)
	require.NoError(t, err)
	exchCfg.Features.Subscriptions = subscription.List{}
	exchCfg.WebsocketTrafficTimeout = time.Hour
	exchCfg.ConnectionMonitorDelay = time.Hour
	return exchCfg
}

func connectGateioWithMockedWebsocket(t *testing.T, wsHandler mockws.WsMockFunc) *Exchange {
	t.Helper()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, wsHandler))
	t.Cleanup(server.Close)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ex.Websocket = websocket.NewManager()
	exchCfg := loadGateioExchangeConfig(t, ex.Name)
	require.NoError(t, ex.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exchCfg,
		Features:                     &ex.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement: true,
	}))

	setupConn := func(filter any) {
		require.NoError(t, ex.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
			URL:                      wsURL,
			ResponseCheckTimeout:     exchCfg.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:         exchCfg.WebsocketResponseMaxLimit,
			SubscriptionsNotRequired: true,
			Connector: func(ctx context.Context, conn websocket.Connection) error {
				return conn.Dial(ctx, &gws.Dialer{}, http.Header{})
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

func TestWebsocketSubmitOrderMocked(t *testing.T) {
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

func TestWebsocketModifyOrderMocked(t *testing.T) {
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
		Amount:    1,
		Price:     101,
	})
	require.NoError(t, err)
	require.Equal(t, "999", futuresResp.OrderID)

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
}

func TestWebsocketCancelOrderMocked(t *testing.T) {
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

	err = ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      getPair(t, asset.Options),
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}
