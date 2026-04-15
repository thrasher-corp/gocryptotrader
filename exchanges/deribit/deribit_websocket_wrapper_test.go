package deribit

import (
	"net/http/httptest"
	"strings"
	"testing"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func defaultDeribitOrderWSResponse(method string) string {
	switch method {
	case submitBuy:
		return `{"jsonrpc":"2.0","id":"{{id}}","result":{"order":{"order_id":"buy-order"}}}`
	case submitSell:
		return `{"jsonrpc":"2.0","id":"{{id}}","result":{"order":{"order_id":"sell-order"}}}`
	case submitEdit:
		return `{"jsonrpc":"2.0","id":"{{id}}","result":{"order":{"order_id":"edited-order"}}}`
	case submitCancel:
		return `{"jsonrpc":"2.0","id":"{{id}}","result":{"order_id":"1"}}`
	default:
		return `{"jsonrpc":"2.0","id":"{{id}}","result":{}}`
	}
}

func deribitOrderWSMock(overrides map[string]string) mockws.WsMockFunc {
	return func(_ testing.TB, p []byte, c *gws.Conn) error {
		var req struct {
			ID     string `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(p, &req); err != nil {
			return err
		}

		response, ok := overrides[req.Method]
		if !ok {
			response = defaultDeribitOrderWSResponse(req.Method)
		}
		response = strings.ReplaceAll(response, "{{id}}", req.ID)
		return c.WriteMessage(gws.TextMessage, []byte(response))
	}
}

func connectDeribitWithMockedWebsocket(t *testing.T, wsHandler mockws.WsMockFunc) *Exchange {
	t.Helper()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, wsHandler))
	t.Cleanup(server.Close)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	require.NoError(t, ex.Websocket.SetAllConnectionURLs(wsURL))
	ex.Features.Subscriptions = subscription.List{}
	ex.Websocket.SetSubscriptionsNotRequired()
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	require.NoError(t, ex.Websocket.Connect(t.Context()))
	t.Cleanup(func() {
		_ = ex.Websocket.Shutdown()
	})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return ex
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	sub := &order.Submit{
		Exchange:    ex.Name,
		Pair:        optionsTradablePair,
		AssetType:   asset.Options,
		Side:        order.Buy,
		Type:        order.Limit,
		Amount:      1,
		Price:       1,
		QuoteAmount: 1,
	}
	_, err := ex.WebsocketSubmitOrder(t.Context(), sub)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)

	// Wrapper websocket usage requires both authenticated endpoints and an active websocket connection.
	// Setting auth capability alone is insufficient.
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	_, err = ex.WebsocketSubmitOrder(t.Context(), sub)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	modify := &order.Modify{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      optionsTradablePair,
		Amount:    1,
	}
	_, err := ex.WebsocketModifyOrder(t.Context(), modify)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	_, err = ex.WebsocketModifyOrder(t.Context(), modify)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	cancel := &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      optionsTradablePair,
	}
	err := ex.WebsocketCancelOrder(t.Context(), cancel)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	err = ex.WebsocketCancelOrder(t.Context(), cancel)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
}

func TestSymbolChannelSeparator(t *testing.T) {
	t.Parallel()

	assert.Empty(t, symbolChannelSeparator(&subscription.Subscription{Channel: subscription.MyAccountChannel}))
	assert.Equal(t, ".", symbolChannelSeparator(&subscription.Subscription{Channel: subscription.MyOrdersChannel}))
}

func TestFormatChannelPair(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		pair currency.Pair
		want string
	}{
		{
			name: "standard option pair unchanged",
			pair: currency.NewPairWithDelimiter("BTC", "USD-230224-18000-C", "-"),
			want: "BTC-USD-230224-18000-C",
		},
		{
			name: "perpetual quote with dash uses underscore delimiter",
			pair: currency.NewPair(currency.BTC, currency.NewCode("USDT-PERPETUAL")),
			want: "BTC_USDT-PERPETUAL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, formatChannelPair(tc.pair))
		})
	}
}

func TestWebsocketSubmitOrderMocked(t *testing.T) {
	t.Parallel()

	ex := connectDeribitWithMockedWebsocket(t, deribitOrderWSMock(nil))

	sub := &order.Submit{
		Exchange:  ex.Name,
		Pair:      optionsTradablePair,
		AssetType: asset.Options,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	}

	_, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{})
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	unsupported := *sub
	unsupported.AssetType = asset.Binary
	_, err = ex.WebsocketSubmitOrder(t.Context(), &unsupported)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	badSide := *sub
	badSide.Side = order.AnySide
	_, err = ex.WebsocketSubmitOrder(t.Context(), &badSide)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	exError := connectDeribitWithMockedWebsocket(t, deribitOrderWSMock(map[string]string{
		submitBuy: `{"jsonrpc":"2.0","id":"{{id}}","error":{"code":13009,"message":"ws buy failed"}}`,
	}))
	_, err = exError.WebsocketSubmitOrder(t.Context(), sub)
	require.ErrorIs(t, err, request.ErrAuthRequestFailed)

	exNoResp := connectDeribitWithMockedWebsocket(t, deribitOrderWSMock(map[string]string{
		submitBuy: `{"jsonrpc":"2.0","id":"{{id}}","result":null}`,
	}))
	_, err = exNoResp.WebsocketSubmitOrder(t.Context(), sub)
	require.ErrorIs(t, err, common.ErrNoResponse)

	resp, err := ex.WebsocketSubmitOrder(t.Context(), sub)
	require.NoError(t, err)
	require.Equal(t, "buy-order", resp.OrderID)
	require.Equal(t, order.New, resp.Status)

	ioc := *sub
	ioc.TimeInForce = order.ImmediateOrCancel
	resp, err = ex.WebsocketSubmitOrder(t.Context(), &ioc)
	require.NoError(t, err)
	require.Equal(t, "buy-order", resp.OrderID)

	sell := *sub
	sell.Side = order.Sell
	resp, err = ex.WebsocketSubmitOrder(t.Context(), &sell)
	require.NoError(t, err)
	require.Equal(t, "sell-order", resp.OrderID)
}

func TestWebsocketModifyOrderMocked(t *testing.T) {
	t.Parallel()

	ex := connectDeribitWithMockedWebsocket(t, deribitOrderWSMock(nil))

	mod := &order.Modify{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      optionsTradablePair,
		Amount:    1,
	}

	_, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{})
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	unsupported := *mod
	unsupported.AssetType = asset.Binary
	_, err = ex.WebsocketModifyOrder(t.Context(), &unsupported)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	exError := connectDeribitWithMockedWebsocket(t, deribitOrderWSMock(map[string]string{
		submitEdit: `{"jsonrpc":"2.0","id":"{{id}}","error":{"code":13010,"message":"ws edit failed"}}`,
	}))
	_, err = exError.WebsocketModifyOrder(t.Context(), mod)
	require.ErrorIs(t, err, request.ErrAuthRequestFailed)

	resp, err := ex.WebsocketModifyOrder(t.Context(), mod)
	require.NoError(t, err)
	require.Equal(t, "edited-order", resp.OrderID)
}

func TestWebsocketCancelOrderMocked(t *testing.T) {
	t.Parallel()

	ex := connectDeribitWithMockedWebsocket(t, deribitOrderWSMock(nil))

	cancel := &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      optionsTradablePair,
	}

	unsupported := *cancel
	unsupported.AssetType = asset.Binary
	err := ex.WebsocketCancelOrder(t.Context(), &unsupported)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	invalid := *cancel
	invalid.OrderID = ""
	err = ex.WebsocketCancelOrder(t.Context(), &invalid)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	exError := connectDeribitWithMockedWebsocket(t, deribitOrderWSMock(map[string]string{
		submitCancel: `{"jsonrpc":"2.0","id":"{{id}}","error":{"code":13011,"message":"ws cancel failed"}}`,
	}))
	err = exError.WebsocketCancelOrder(t.Context(), cancel)
	require.ErrorIs(t, err, request.ErrAuthRequestFailed)

	err = ex.WebsocketCancelOrder(t.Context(), cancel)
	require.NoError(t, err)
}
