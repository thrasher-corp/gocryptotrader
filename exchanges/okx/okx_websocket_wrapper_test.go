package okx

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
	exchangeoptions "github.com/thrasher-corp/gocryptotrader/exchange/options"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func connectOKXWithMockedWebsocket(t *testing.T, wsHandler mockws.WsMockFunc) *Exchange {
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
	require.ErrorIs(t, err, errContractAmountCanNotBeDecimal)
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

func TestWebsocketSpreadRouting(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	t.Run("submit spread does not fail as unsupported asset", func(t *testing.T) {
		t.Parallel()
		_, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  ex.Name,
			Pair:      spreadPair,
			AssetType: asset.Spread,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
		})
		require.ErrorIs(t, err, common.ErrFunctionNotSupported)
		require.NotErrorIs(t, err, asset.ErrNotSupported)
	})

	t.Run("modify spread does not fail as unsupported asset", func(t *testing.T) {
		t.Parallel()
		_, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{
			OrderID:   "1",
			AssetType: asset.Spread,
			Pair:      spreadPair,
			Amount:    1,
			Price:     1,
		})
		require.ErrorIs(t, err, common.ErrFunctionNotSupported)
		require.NotErrorIs(t, err, asset.ErrNotSupported)
	})

	t.Run("cancel spread does not fail as unsupported asset", func(t *testing.T) {
		t.Parallel()
		err := ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
			OrderID:   "1",
			AssetType: asset.Spread,
		})
		require.ErrorIs(t, err, common.ErrFunctionNotSupported)
		require.NotErrorIs(t, err, asset.ErrNotSupported)
	})
}

func TestDeriveSubmitOrderArguments(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

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

	_, err := ex.deriveAmendOrderArguments(&order.Modify{})
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	_, err = ex.deriveAmendOrderArguments(nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil)

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
	require.Equal(t, "BTC-USDT", arg.InstrumentID)
	require.Equal(t, 2.0, arg.NewQuantity)
	require.Equal(t, 3.0, arg.NewPrice)
	require.Equal(t, "1", arg.OrderID)
	require.Equal(t, "abc", arg.ClientOrderID)
}

func TestDeriveCancelOrderArguments(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	_, err := ex.deriveCancelOrderArguments(&order.Cancel{
		AssetType: asset.Options,
		Pair:      mainPair,
		OrderID:   "1",
	})
	require.NoError(t, err)

	_, err = ex.deriveCancelOrderArguments(nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

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
	require.Equal(t, "BTC-USDT", arg.InstrumentID)
	require.Equal(t, "1", arg.OrderID)
	require.Equal(t, "abc", arg.ClientOrderID)
}

func TestWSProcessOptionSummary(t *testing.T) {
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
