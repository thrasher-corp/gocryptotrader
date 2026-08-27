package okx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchangeoptions "github.com/thrasher-corp/gocryptotrader/exchange/options"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func connectOKXWithMockedWebsocket(t *testing.T, wsHandler mockws.WsMockFunc) *Exchange {
	t.Helper()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))
	instrumentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		instrumentID := r.URL.Query().Get("instId")
		if instrumentID == "" {
			instrumentID = mainPair.String()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"` + instrumentID + `","instIdCode":"42"}]}`))
	}))
	t.Cleanup(instrumentServer.Close)
	require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", instrumentServer.URL+"/"))

	server := httptest.NewServer(mockws.CurryWsMockUpgrader(t, wsHandler))
	t.Cleanup(server.Close)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ex.Websocket = websocket.NewManager()
	exchCfg := ex.Config
	require.NotNil(t, exchCfg)
	exchCfg.Features.Subscriptions = subscription.List{}
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
			return conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil)
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

	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs(wsURL))
	require.NoError(t, ex.Websocket.Connect(t.Context()))
	require.Eventually(t, func() bool {
		_, err := ex.Websocket.GetConnection(privateConnection)
		return err == nil
	}, time.Second, 10*time.Millisecond, "private websocket connection was not ready")
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
	case "sprd-order":
		response = `{"id":"` + req.ID + `","op":"sprd-order","code":"0","msg":"","data":[{"ordId":"spread-order","sCode":"0","sMsg":""}]}`
	case "sprd-amend-order":
		response = `{"id":"` + req.ID + `","op":"sprd-amend-order","code":"0","msg":"","data":[{"ordId":"spread-amended","sCode":"0","sMsg":""}]}`
	case "sprd-cancel-order":
		response = `{"id":"` + req.ID + `","op":"sprd-cancel-order","code":"0","msg":"","data":[{"ordId":"spread-cancelled","sCode":"0","sMsg":""}]}`
	default:
		response = `{"id":"` + req.ID + `","op":"` + req.Op + `","code":"1","msg":"operation failed","data":[{"sCode":"51000","sMsg":"failed"}]}`
	}
	return c.WriteMessage(gws.TextMessage, []byte(response))
}

func okxOrderWsErrorMock(_ testing.TB, p []byte, c *gws.Conn) error {
	var req struct {
		ID string `json:"id"`
		Op string `json:"op"`
	}
	if err := json.Unmarshal(p, &req); err != nil {
		return err
	}
	return c.WriteMessage(gws.TextMessage, []byte(`{"id":"`+req.ID+`","op":"`+req.Op+`","code":"1","msg":"operation failed","data":[]}`))
}

func TestWebsocketSubmitOrder(t *testing.T) {
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

	resp, err = ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  ex.Name,
		Pair:      spreadPair,
		AssetType: asset.Spread,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	})
	require.NoError(t, err)
	require.Equal(t, "spread-order", resp.OrderID)

	_, err = ex.WebsocketSubmitOrder(t.Context(), &order.Submit{})
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	wsError := connectOKXWithMockedWebsocket(t, okxOrderWsErrorMock)
	_, err = wsError.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  wsError.Name,
		Pair:      mainPair,
		AssetType: asset.Options,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	})
	require.Error(t, err)
	_, err = wsError.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  wsError.Name,
		Pair:      spreadPair,
		AssetType: asset.Spread,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	})
	require.Error(t, err)

	badFormat := connectOKXWithMockedWebsocket(t, okxOrderWsMock)
	badFormat.CurrencyPairs.UseGlobalFormat = true
	badFormat.CurrencyPairs.RequestFormat = nil
	_, err = badFormat.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  badFormat.Name,
		Pair:      spreadPair,
		AssetType: asset.Spread,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil)

	_, err = ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  ex.Name,
		Pair:      mainPair,
		AssetType: asset.Binary,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

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

	resolveError := connectOKXWithMockedWebsocket(t, okxOrderWsMock)
	instrumentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "failed", http.StatusInternalServerError)
	}))
	t.Cleanup(instrumentServer.Close)
	require.NoError(t, resolveError.API.Endpoints.SetRunningURL("RestSpotURL", instrumentServer.URL+"/"))
	_, err = resolveError.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:  resolveError.Name,
		Pair:      mainPair,
		AssetType: asset.Options,
		Side:      order.Long,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	})
	assert.Error(t, err, "instrument resolution failure should stop websocket order submission")
}

func TestWebsocketModifyOrder(t *testing.T) {
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

	resp, err = ex.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "spread-1",
		AssetType: asset.Spread,
		Pair:      spreadPair,
		Amount:    1,
		Price:     1,
	})
	require.NoError(t, err)
	require.Equal(t, "spread-1", resp.OrderID)

	_, err = ex.WebsocketModifyOrder(t.Context(), &order.Modify{})
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	invalid := *modify
	invalid.Amount = 1.5
	_, err = ex.WebsocketModifyOrder(t.Context(), &invalid)
	require.ErrorIs(t, err, errContractAmountCanNotBeDecimal)

	wsError := connectOKXWithMockedWebsocket(t, okxOrderWsErrorMock)
	_, err = wsError.WebsocketModifyOrder(t.Context(), modify)
	require.Error(t, err)
	_, err = wsError.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "spread-error",
		AssetType: asset.Spread,
		Pair:      spreadPair,
		Amount:    1,
		Price:     1,
	})
	require.Error(t, err)

	_, err = ex.WebsocketModifyOrder(t.Context(), &order.Modify{
		OrderID:   "unsupported",
		AssetType: asset.Binary,
		Pair:      mainPair,
		Amount:    1,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	_, err = ex.WebsocketModifyOrder(t.Context(), modify)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()

	ex := connectOKXWithMockedWebsocket(t, okxOrderWsMock)

	cancel := &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      mainPair,
	}
	err := ex.WebsocketCancelOrder(t.Context(), cancel)
	require.NoError(t, err)

	err = ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "spread-1",
		AssetType: asset.Spread,
	})
	require.NoError(t, err)

	err = ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.Options,
		Pair:      mainPair,
	})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	wsError := connectOKXWithMockedWebsocket(t, okxOrderWsErrorMock)
	err = wsError.WebsocketCancelOrder(t.Context(), cancel)
	require.Error(t, err)
	err = wsError.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "spread-error",
		AssetType: asset.Spread,
	})
	require.Error(t, err)

	err = ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
		OrderID:   "unsupported",
		AssetType: asset.Binary,
		Pair:      mainPair,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	err = ex.WebsocketCancelOrder(t.Context(), cancel)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestDeriveSubmitOrderArguments(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	t.Run("unsupported asset", func(t *testing.T) {
		t.Parallel()
		_, err := ex.deriveSubmitOrderArguments(&order.Submit{AssetType: asset.Binary, Amount: 1})
		require.ErrorIs(t, err, asset.ErrNotSupported)
	})

	t.Run("amount below minimum", func(t *testing.T) {
		t.Parallel()
		_, err := ex.deriveSubmitOrderArguments(&order.Submit{AssetType: asset.Spot})
		require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	})

	t.Run("spread uses dedicated endpoint", func(t *testing.T) {
		t.Parallel()
		_, err := ex.deriveSubmitOrderArguments(&order.Submit{AssetType: asset.Spread, Amount: 1})
		require.ErrorIs(t, err, asset.ErrNotSupported)
	})

	t.Run("empty pair", func(t *testing.T) {
		t.Parallel()
		_, err := ex.deriveSubmitOrderArguments(&order.Submit{
			AssetType: asset.Spot,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
		})
		require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	})

	t.Run("unsupported order type", func(t *testing.T) {
		t.Parallel()
		_, err := ex.deriveSubmitOrderArguments(&order.Submit{
			Pair:      mainPair,
			AssetType: asset.Spot,
			Side:      order.Buy,
			Type:      order.Trigger,
			Amount:    1,
		})
		require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	})

	t.Run("invalid order type", func(t *testing.T) {
		t.Parallel()
		_, err := ex.deriveSubmitOrderArguments(&order.Submit{
			Pair:      mainPair,
			AssetType: asset.Spot,
			Side:      order.Buy,
			Type:      order.UnknownType,
			Amount:    1,
		})
		require.ErrorIs(t, err, order.ErrUnsupportedOrderType)
	})

	t.Run("pair format unavailable", func(t *testing.T) {
		t.Parallel()
		badFormat := new(Exchange)
		require.NoError(t, testexch.Setup(badFormat), "Setup must not error")
		badFormat.CurrencyPairs.UseGlobalFormat = true
		badFormat.CurrencyPairs.RequestFormat = nil
		_, err := badFormat.deriveSubmitOrderArguments(&order.Submit{
			Pair:      mainPair,
			AssetType: asset.Spot,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
		})
		require.ErrorIs(t, err, currency.ErrPairFormatIsNil)
	})

	t.Run("spot market quote amount", func(t *testing.T) {
		t.Parallel()
		arg, err := ex.deriveSubmitOrderArguments(&order.Submit{
			Exchange:    ex.Name,
			Pair:        mainPair,
			AssetType:   asset.Spot,
			Side:        order.Buy,
			Type:        order.Market,
			QuoteAmount: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, order.Buy.Lower(), arg.Side)
		assert.Equal(t, "quote_ccy", arg.TargetCurrency)
		assert.Equal(t, 10.0, arg.Amount)
	})

	t.Run("futures leverage guard", func(t *testing.T) {
		t.Parallel()
		_, err := ex.deriveSubmitOrderArguments(&order.Submit{
			Exchange:  ex.Name,
			Pair:      mainPair,
			AssetType: asset.Futures,
			Side:      order.Long,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			Leverage:  2,
		})
		require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)
	})

	t.Run("futures reduce only position side", func(t *testing.T) {
		t.Parallel()
		arg, err := ex.deriveSubmitOrderArguments(&order.Submit{
			Exchange:   ex.Name,
			Pair:       mainPair,
			AssetType:  asset.Futures,
			Side:       order.Buy,
			Type:       order.Limit,
			Amount:     1,
			Price:      1,
			ReduceOnly: true,
		})
		require.NoError(t, err)
		assert.Equal(t, order.Buy.Lower(), arg.Side)
		assert.Equal(t, positionSideShort, arg.PositionSide)
	})

	t.Run("options side is set", func(t *testing.T) {
		t.Parallel()
		arg, err := ex.deriveSubmitOrderArguments(&order.Submit{
			Exchange:  ex.Name,
			Pair:      mainPair,
			AssetType: asset.Options,
			Side:      order.Sell,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
		})
		require.NoError(t, err)
		assert.Equal(t, order.Sell.Lower(), arg.Side)
		assert.Empty(t, arg.PositionSide)
	})

	t.Run("invalid side rejected", func(t *testing.T) {
		t.Parallel()
		_, err := ex.deriveSubmitOrderArguments(&order.Submit{
			Exchange:  ex.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			Side:      order.AnySide,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
		})
		require.ErrorIs(t, err, order.ErrSideIsInvalid)
	})
}

func TestDeriveOrderSide(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		side    order.Side
		want    string
		wantErr error
	}{
		{
			name: "buy",
			side: order.Buy,
			want: order.Buy.Lower(),
		},
		{
			name: "sell",
			side: order.Sell,
			want: order.Sell.Lower(),
		},
		{
			name:    "invalid",
			side:    order.AnySide,
			wantErr: order.ErrSideIsInvalid,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := deriveOrderSide(tc.side)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDerivePositionSide(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		sub  *order.Submit
		want string
	}{
		{
			name: "spot empty",
			sub: &order.Submit{
				AssetType: asset.Spot,
				Side:      order.Buy,
			},
			want: "",
		},
		{
			name: "futures long",
			sub: &order.Submit{
				AssetType: asset.Futures,
				Side:      order.Long,
			},
			want: positionSideLong,
		},
		{
			name: "futures short",
			sub: &order.Submit{
				AssetType: asset.Futures,
				Side:      order.Short,
			},
			want: positionSideShort,
		},
		{
			name: "futures reduce only buy",
			sub: &order.Submit{
				AssetType:  asset.Futures,
				Side:       order.Buy,
				ReduceOnly: true,
			},
			want: positionSideShort,
		},
		{
			name: "futures reduce only sell",
			sub: &order.Submit{
				AssetType:  asset.Futures,
				Side:       order.Sell,
				ReduceOnly: true,
			},
			want: positionSideLong,
		},
		{
			name: "futures buy",
			sub: &order.Submit{
				AssetType: asset.Futures,
				Side:      order.Buy,
			},
			want: positionSideLong,
		},
		{
			name: "futures sell",
			sub: &order.Submit{
				AssetType: asset.Futures,
				Side:      order.Sell,
			},
			want: positionSideShort,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, derivePositionSide(tc.sub))
		})
	}
}

func TestIsSpotMarketOrder(t *testing.T) {
	t.Parallel()

	require.True(t, isSpotMarketOrder(&order.Submit{AssetType: asset.Spot, Type: order.Market}))
	require.False(t, isSpotMarketOrder(&order.Submit{AssetType: asset.Spot, Type: order.Limit}))
	require.False(t, isSpotMarketOrder(&order.Submit{AssetType: asset.Futures, Type: order.Market}))
}

func TestIsSpotMarketBuyWithQuoteAmount(t *testing.T) {
	t.Parallel()

	require.True(t, isSpotMarketBuyWithQuoteAmount(&order.Submit{
		AssetType:   asset.Spot,
		Type:        order.Market,
		Side:        order.Buy,
		QuoteAmount: 1,
	}))
	require.False(t, isSpotMarketBuyWithQuoteAmount(&order.Submit{
		AssetType:   asset.Spot,
		Type:        order.Market,
		Side:        order.Sell,
		QuoteAmount: 1,
	}))
	require.False(t, isSpotMarketBuyWithQuoteAmount(&order.Submit{
		AssetType:   asset.Spot,
		Type:        order.Limit,
		Side:        order.Buy,
		QuoteAmount: 1,
	}))
	require.False(t, isSpotMarketBuyWithQuoteAmount(&order.Submit{
		AssetType:   asset.Spot,
		Type:        order.Market,
		Side:        order.Buy,
		QuoteAmount: 0,
	}))
}

func TestDeriveAmendOrderArguments(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	badFormat := new(Exchange)
	require.NoError(t, testexch.Setup(badFormat), "Setup must not error")
	badFormat.CurrencyPairs.UseGlobalFormat = true
	badFormat.CurrencyPairs.RequestFormat = nil
	_, err := badFormat.deriveAmendOrderArguments(&order.Modify{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      mainPair,
		Amount:    1,
	})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil)

	_, err = ex.deriveAmendOrderArguments(&order.Modify{})
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	_, err = ex.deriveAmendOrderArguments(nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil)

	_, err = ex.deriveAmendOrderArguments(&order.Modify{
		OrderID:   "1",
		AssetType: asset.Binary,
		Pair:      mainPair,
		Amount:    1,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = ex.deriveAmendOrderArguments(&order.Modify{
		OrderID:   "1",
		AssetType: asset.Spread,
		Pair:      spreadPair,
		Amount:    1,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = ex.deriveAmendOrderArguments(&order.Modify{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      mainPair,
		Amount:    1.5,
	})
	require.ErrorIs(t, err, errContractAmountCanNotBeDecimal)

	arg, err := ex.deriveAmendOrderArguments(&order.Modify{
		OrderID:       "1",
		ClientOrderID: "abc",
		AssetType:     asset.Options,
		Pair:          mainPair,
		Amount:        2,
		Price:         3,
	})
	require.NoError(t, err)
	assert.Equal(t, "BTC-USDT", arg.InstrumentID)
	assert.Equal(t, 2.0, arg.NewQuantity)
	assert.Equal(t, 3.0, arg.NewPrice)
	assert.Equal(t, "1", arg.OrderID)
	assert.Equal(t, "abc", arg.ClientOrderID)
}

func TestDeriveCancelOrderArguments(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	badFormat := new(Exchange)
	require.NoError(t, testexch.Setup(badFormat), "Setup must not error")
	badFormat.CurrencyPairs.UseGlobalFormat = true
	badFormat.CurrencyPairs.RequestFormat = nil
	_, err := badFormat.deriveCancelOrderArguments(&order.Cancel{
		AssetType: asset.Options,
		Pair:      mainPair,
		OrderID:   "1",
	})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil)

	_, err = ex.deriveCancelOrderArguments(&order.Cancel{
		AssetType: asset.Options,
		Pair:      mainPair,
		OrderID:   "1",
	})
	require.NoError(t, err)

	_, err = ex.deriveCancelOrderArguments(nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	_, err = ex.deriveCancelOrderArguments(&order.Cancel{
		AssetType: asset.Binary,
		Pair:      mainPair,
		OrderID:   "1",
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = ex.deriveCancelOrderArguments(&order.Cancel{
		AssetType: asset.Spread,
		Pair:      spreadPair,
		OrderID:   "1",
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = ex.deriveCancelOrderArguments(&order.Cancel{
		AssetType: asset.Options,
		OrderID:   "1",
	})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg, err := ex.deriveCancelOrderArguments(&order.Cancel{
		AssetType:     asset.Options,
		Pair:          mainPair,
		OrderID:       "1",
		ClientOrderID: "abc",
	})
	require.NoError(t, err)
	assert.Equal(t, "BTC-USDT", arg.InstrumentID)
	assert.Equal(t, "1", arg.OrderID)
	assert.Equal(t, "abc", arg.ClientOrderID)
}

func TestOptionInstrumentSelector(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		instrumentID string
		expected     string
	}{
		{name: "dash separator", instrumentID: "BTC-USDT-260101-100000-C", expected: "BTC-USDT"},
		{name: "underscore separator", instrumentID: "BTC_USDT_260101_100000_C", expected: "BTC_USDT"},
		{name: "single component", instrumentID: "BTC", expected: "BTC"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, optionInstrumentSelector(tc.instrumentID))
		})
	}
}

func TestIsInstFamilyChannel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		sub  *subscription.Subscription
		want bool
	}{
		{
			name: "options trades",
			sub:  &subscription.Subscription{Asset: asset.Options, Channel: subscription.AllTradesChannel},
			want: true,
		},
		{
			name: "options summary",
			sub:  &subscription.Subscription{Asset: asset.Options, Channel: channelOptSummary},
			want: true,
		},
		{
			name: "spot ticker",
			sub:  &subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, isInstFamilyChannel(tc.sub), "isInstFamilyChannel should classify the subscription")
		})
	}
}

func TestSubscriptionForAsset(t *testing.T) {
	t.Parallel()

	original := &subscription.Subscription{Asset: asset.All, Channel: subscription.TickerChannel}
	got := subscriptionForAsset(original, asset.Options)
	assert.NotSame(t, original, got, "subscriptionForAsset should clone the subscription")
	assert.Equal(t, asset.All, original.Asset, "subscriptionForAsset should not mutate the input")
	assert.Equal(t, asset.Options, got.Asset, "subscriptionForAsset should set the expanded asset")
}

func TestLookupInstrumentIDCode(t *testing.T) {
	t.Parallel()

	instrumentID := "BTC-USDT-260101-100000-C"
	testCases := []struct {
		name        string
		instruments []Instrument
		want        int64
	}{
		{
			name: "matching positive code",
			instruments: []Instrument{{
				InstrumentID:     currency.NewPairWithDelimiter("BTC", "USDT-260101-100000-C", currency.DashDelimiter),
				InstrumentIDCode: types.Number(42),
			}},
			want: 42,
		},
		{
			name: "matching zero code",
			instruments: []Instrument{{
				InstrumentID: currency.NewPairWithDelimiter("BTC", "USDT-260101-100000-C", currency.DashDelimiter),
			}},
		},
		{
			name: "single mismatched instrument is not trusted",
			instruments: []Instrument{{
				InstrumentID:     currency.NewPair(currency.ETH, currency.USDT),
				InstrumentIDCode: types.Number(99),
			}},
		},
		{name: "empty response"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, lookupInstrumentIDCode(tc.instruments, instrumentID), "lookupInstrumentIDCode should return the matching positive code only")
		})
	}
}

func TestResolveInstrumentIDCode(t *testing.T) {
	t.Parallel()

	const instrumentID = "BTC-USDT-260101-100000-C"
	var requests atomic.Int64
	var invalidOptionsQuery atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Query().Get("instFamily") == "BTC-USDT":
			if r.URL.Query().Get("uly") != "" || r.URL.Query().Get("instId") != "" {
				invalidOptionsQuery.Store(true)
			}
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"BTC-USDT-260101-100000-C","instIdCode":"42"}]}`))
		case r.URL.Query().Get("instId") == "FAIL-USDT":
			http.Error(w, "failed", http.StatusInternalServerError)
		default:
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
		}
	}))
	t.Cleanup(server.Close)

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/"), "SetRunningURL must not error")

	code, err := ex.resolveInstrumentIDCode(t.Context(), asset.Options, instrumentID)
	require.NoError(t, err)
	assert.Equal(t, int64(42), code, "resolveInstrumentIDCode should return the exchange code")
	assert.False(t, invalidOptionsQuery.Load(), "options lookup should use only the instrument family selector")
	assert.Equal(t, int64(1), requests.Load(), "first lookup should make one instrument request")

	code, err = ex.resolveInstrumentIDCode(t.Context(), asset.Options, instrumentID)
	require.NoError(t, err)
	assert.Equal(t, int64(42), code, "cached lookup should return the exchange code")
	assert.Equal(t, int64(1), requests.Load(), "cached lookup should not make another instrument request")

	_, err = ex.resolveInstrumentIDCode(t.Context(), asset.Options, "")
	assert.ErrorIs(t, err, errMissingInstrumentID, "resolveInstrumentIDCode should reject an empty instrument")

	_, err = ex.resolveInstrumentIDCode(t.Context(), asset.Binary, instrumentID)
	assert.ErrorIs(t, err, errInvalidInstrumentType, "resolveInstrumentIDCode should reject unsupported assets")

	_, err = ex.resolveInstrumentIDCode(t.Context(), asset.Spot, "UNKNOWN-USDT")
	assert.ErrorIs(t, err, errInstrumentIDCodeNotFound, "resolveInstrumentIDCode should report a missing code")

	_, err = ex.resolveInstrumentIDCode(t.Context(), asset.Spot, "FAIL-USDT")
	assert.Error(t, err, "resolveInstrumentIDCode should return instrument request errors")
}

func TestWsProcessOptionSummary(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsProcessOptionSummary(t.Context(), []byte("{"))
	require.ErrorIs(t, err, errOptionSummaryUnmarshal)

	err = ex.wsProcessOptionSummary(t.Context(), []byte(`{"data":[{"instId":"BTC-USD-230224-18000-C","delta":"0.1","gamma":"0.2","theta":"-0.3","vega":"0.4","bidVol":"0.5","askVol":"0.6","markVol":"0.55","ts":"1700000000000"}]}`))
	require.NoError(t, err)

	select {
	case got := <-ex.Websocket.DataHandler.C:
		opt, ok := got.Data.(*exchangeoptions.Greeks)
		require.True(t, ok)
		require.Equal(t, ex.Name, opt.ExchangeName)
		require.Equal(t, asset.Options, opt.AssetType)
		require.Equal(t, "BTC-USD-230224-18000-C", opt.Pair.String())
		require.Equal(t, 0.1, opt.Delta)
		require.Equal(t, 0.2, opt.Gamma)
		require.Equal(t, -0.3, opt.Theta)
		require.Equal(t, 0.4, opt.Vega)
		require.Equal(t, 0.5, opt.BidImpliedVolatility)
		require.Equal(t, 0.6, opt.AskImpliedVolatility)
		require.Equal(t, 0.55, opt.MarkImpliedVolatility)
	default:
		t.Fatal("expected option payload on data handler")
	}

	err = ex.wsProcessOptionSummary(t.Context(), []byte(`{"data":[{"instId":"AB"}]}`))
	require.ErrorIs(t, err, errOptionSummaryPairParse)
	require.ErrorIs(t, err, currency.ErrCreatingPair)

	ex.Websocket.DataHandler = stream.NewRelay(1)
	require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "saturate"))
	err = ex.wsProcessOptionSummary(t.Context(), []byte(`{"data":[{"instId":"BTC-USD-230224-18000-C","delta":"0.1","gamma":"0.2","theta":"-0.3","vega":"0.4","bidVol":"0.5","askVol":"0.6","markVol":"0.55","ts":"1700000000000"}]}`))
	require.ErrorIs(t, err, errOptionSummaryDispatch)
}
