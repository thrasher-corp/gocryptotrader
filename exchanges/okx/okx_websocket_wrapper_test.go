package okx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testpath "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func loadOKXExchangeConfig(t *testing.T, exchangeName string) *config.Exchange {
	t.Helper()

	root, err := testpath.RootPathFromCWD()
	require.NoError(t, err)

	cfg := &config.Config{}
	require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true))

	exchCfg, err := cfg.GetExchangeConfig(exchangeName)
	require.NoError(t, err)
	exchCfg.Features.Subscriptions = subscription.List{}
	return exchCfg
}

func connectOKXWithMockedWebsocket(t *testing.T, wsHandler mockws.WsMockFunc) *Exchange {
	t.Helper()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, wsHandler))
	t.Cleanup(server.Close)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ex.Websocket = websocket.NewManager()
	exchCfg := loadOKXExchangeConfig(t, ex.Name)
	require.NoError(t, ex.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exchCfg,
		Features:                     &ex.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement: true,
	}))

	require.NoError(t, ex.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  wsURL,
		ResponseCheckTimeout: exchCfg.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exchCfg.WebsocketResponseMaxLimit,
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
				ID string `json:"id"`
			}
			if err := json.Unmarshal(incoming, &m); err != nil {
				return err
			}
			if m.ID != "" {
				return conn.RequireMatchWithData(m.ID, incoming)
			}
			return nil
		},
		MessageFilter: privateConnection,
	}))

	require.NoError(t, ex.Websocket.Connect(t.Context()))
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	t.Cleanup(func() {
		_ = ex.Websocket.Shutdown()
	})
	return ex
}

func okxOrderWsMock(_ testing.TB, p []byte, c *gws.Conn) error {
	var req struct {
		ID string `json:"id"`
		Op string `json:"op"`
	}
	if err := json.Unmarshal(p, &req); err != nil {
		return err
	}
	if req.ID == "" {
		req.ID = "mock-id"
	}

	var response string
	switch req.Op {
	case "order":
		response = `{"id":"` + req.ID + `","op":"order","code":"0","msg":"","data":[{"ordId":"submit-order","sCode":"0","sMsg":""}]}`
	case "amend-order":
		response = `{"id":"` + req.ID + `","op":"amend-order","code":"0","msg":"","data":[{"ordId":"amended-order","sCode":"0","sMsg":""}]}`
	case "cancel-order":
		response = `{"id":"` + req.ID + `","op":"cancel-order","code":"0","msg":"","data":[{"ordId":"cancelled-order","sCode":"0","sMsg":""}]}`
	default:
		response = `{"id":"` + req.ID + `","op":"` + req.Op + `","code":"1","msg":"operation failed","data":[{"sCode":"51000","sMsg":"failed"}]}`
	}
	return c.WriteMessage(gws.TextMessage, []byte(response))
}

func TestWebsocketSubmitOrderMocked(t *testing.T) {
	t.Parallel()

	ex := connectOKXWithMockedWebsocket(t, okxOrderWsMock)

	resp, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  ex.Name,
		Pair:      mainPair,
		AssetType: asset.Options,
		Side:      order.Long,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	})
	require.NoError(t, err)
	require.Equal(t, "submit-order", resp.OrderID)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	_, err = ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  ex.Name,
		Pair:      mainPair,
		AssetType: asset.Options,
		Side:      order.Long,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWebsocketModifyOrderMocked(t *testing.T) {
	t.Parallel()

	ex := connectOKXWithMockedWebsocket(t, okxOrderWsMock)

	modify := &order.Modify{
		OrderID:   "order-1",
		AssetType: asset.Options,
		Pair:      mainPair,
		Amount:    1,
		Price:     1,
	}
	resp, err := ex.WebsocketModifyOrder(t.Context(), modify)
	require.NoError(t, err)
	require.Equal(t, "order-1", resp.OrderID)

	invalid := *modify
	invalid.Amount = 1.5
	_, err = ex.WebsocketModifyOrder(t.Context(), &invalid)
	require.ErrorContains(t, err, "contract amount can not be decimal")
}

func TestWebsocketCancelOrderMocked(t *testing.T) {
	t.Parallel()

	ex := connectOKXWithMockedWebsocket(t, okxOrderWsMock)

	cancel := &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      mainPair,
	}
	err := ex.WebsocketCancelOrder(t.Context(), cancel)
	require.NoError(t, err)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	err = ex.WebsocketCancelOrder(t.Context(), cancel)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}
