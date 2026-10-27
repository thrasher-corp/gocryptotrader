package gateio

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestSpotOrderMetadata(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, tif, text string
		want            order.TimeInForce
		wantErr         bool
	}{
		{name: "gtc", tif: "gtc", text: "t-client", want: order.GoodTillCancel},
		{name: "ioc", tif: "ioc", text: "t-client", want: order.ImmediateOrCancel},
		{name: "poc", tif: "poc", text: "t-client", want: order.PostOnly},
		{name: "fok", tif: "fok", text: "t-client", want: order.FillOrKill},
		{name: "omitted", want: order.UnknownTIF},
		{name: "invalid", tif: "invalid", text: "t-client", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			require.NoError(t, ex.Requester.DisableRateLimiter(), "mock requests must not wait for rate limits")
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT()}, true), "test pair must be enabled")
			metadata := ""
			if tc.tif != "" {
				metadata = fmt.Sprintf(`,"time_in_force":%q,"text":%q`, tc.tif, tc.text)
			}
			payload := `{"id":"1","currency_pair":"BTC_USDT","status":"open","type":"limit","account":"spot","side":"buy","amount":"3","left":"1","price":"60000"` + metadata + `}`
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				response := payload
				if strings.HasSuffix(r.URL.Path, "/spot/open_orders") {
					response = `[{"currency_pair":"BTC_USDT","total":1,"orders":[` + payload + `]}]`
				}
				_, err := w.Write([]byte(response))
				assert.NoError(t, err, "mock order metadata should be written")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "test client must be installed")
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			detail, err := ex.GetOrderInfo(t.Context(), "1", currency.NewBTCUSDT(), asset.Spot)
			if tc.wantErr {
				assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "unsupported time in force should be rejected by detail lookup")
			} else {
				require.NoError(t, err, "detail lookup must accept reported metadata")
				assert.Equal(t, tc.want, detail.TimeInForce, "detail lookup should retain the reported time in force")
				assert.Equal(t, tc.text, detail.ClientOrderID, "detail lookup should retain the reported text without guessing")
			}
			active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{Pairs: currency.Pairs{currency.NewBTCUSDT()}, AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
			if tc.wantErr {
				assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "unsupported time in force should be rejected by active lookup")
			} else {
				require.NoError(t, err, "active lookup must accept reported metadata")
				require.Len(t, active, 1, "active lookup must return the fixture order")
				assert.Equal(t, tc.want, active[0].TimeInForce, "active lookup should retain the reported time in force")
				assert.Equal(t, tc.text, active[0].ClientOrderID, "active lookup should retain the reported text without guessing")
			}
			err = ex.processSpotOrders(t.Context(), []byte(`{"result":[`+payload+`]}`))
			if tc.wantErr {
				assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "unsupported websocket time in force should be rejected")
			} else {
				require.NoError(t, err, "spot order push must accept reported metadata")
				require.Len(t, ex.Websocket.DataHandler.C, 1, "spot order push must emit one update")
				got, ok := (<-ex.Websocket.DataHandler.C).Data.([]order.Detail)
				require.True(t, ok, "spot order push must carry order details")
				require.Len(t, got, 1, "spot order push must carry one order")
				assert.Equal(t, tc.want, got[0].TimeInForce, "spot order push should preserve time in force")
				assert.Equal(t, tc.text, got[0].ClientOrderID, "spot order push should preserve custom text")
			}
		})
	}
}

func TestOrderSourceMarkerText(t *testing.T) {
	t.Parallel()
	for _, text := range []string{"", "apiv4", "apiv4-ws", "2", "3", "101", "102", "103", "104", "t-client", "t-42", "t-Ab_C.9", "t-apiv4", "t-client-ws", "t-abcdefghijklmnopqrstuvwxyz01"} {
		t.Run(text, func(t *testing.T) {
			t.Parallel()
			want := ""
			if strings.HasPrefix(text, "t-") {
				want = text
			}
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			require.NoError(t, ex.Requester.DisableRateLimiter(), "mock requests must not wait for rate limits")
			spot := currency.NewBTCUSDT()
			optionPair := currency.NewPairWithDelimiter("BTC", "USDT-20301220-50000-C", "_")
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{spot}, true), "spot pair must be enabled")
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{optionPair}, true), "option pair must be enabled")
			spotPayload := fmt.Sprintf(`{"id":"1","currency_pair":"BTC_USDT","status":"open","type":"limit","account":"spot","side":"buy","amount":"3","left":"1","price":"60000","time_in_force":"gtc","text":%q}`, text)
			optionsPayload := fmt.Sprintf(`{"id":2,"contract":%q,"status":"open","size":"3","left":"1","price":"2","tif":"gtc","text":%q}`, optionPair.String(), text)
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				payload := spotPayload
				switch {
				case strings.HasSuffix(r.URL.Path, "/spot/open_orders"):
					payload = `[{"currency_pair":"BTC_USDT","total":1,"orders":[` + spotPayload + `]}]`
				case strings.HasSuffix(r.URL.Path, "/options/orders"):
					payload = optionsPayload
					if r.Method == http.MethodGet {
						payload = `[` + optionsPayload + `]`
					}
				}
				_, err := w.Write([]byte(payload))
				assert.NoError(t, err, "fixture order should be written")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "fixture client must be installed")
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			detail, err := ex.GetOrderInfo(t.Context(), "1", spot, asset.Spot)
			require.NoError(t, err, "spot detail lookup must succeed")
			assert.Equal(t, want, detail.ClientOrderID, "spot lookup should report only custom client IDs")
			for _, a := range []asset.Item{asset.Spot, asset.Options} {
				pair, price := spot, 60000.0
				if a == asset.Options {
					pair, price = optionPair, 2
				}
				submitted, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: pair, AssetType: a, Type: order.Limit, Side: order.Buy, Amount: 3, Price: price, TimeInForce: order.GoodTillCancel})
				require.NoError(t, err, "fixture submission must succeed")
				assert.Equal(t, want, submitted.ClientOrderID, "submission should report only custom client IDs")
				active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{Pairs: currency.Pairs{pair}, AssetType: a, Side: order.AnySide, Type: order.AnyType})
				require.NoError(t, err, "active lookup must succeed")
				require.Len(t, active, 1, "active lookup must return one order")
				assert.Equal(t, want, active[0].ClientOrderID, "active lookup should report only custom client IDs")
			}
			var response *WebsocketOrderResponse
			require.NoError(t, json.Unmarshal([]byte(spotPayload), &response), "websocket response fixture must decode")
			submitted, err := ex.deriveSpotWebsocketOrderResponse(response)
			require.NoError(t, err, "single websocket submission must succeed")
			assert.Equal(t, want, submitted.ClientOrderID, "single websocket submission should filter source markers")
			failed := &WebsocketOrderResponse{Text: text, Label: "REJECTED", Message: "fixture rejection"}
			batch, err := ex.deriveSpotWebsocketOrderResponses([]*WebsocketOrderResponse{response, failed})
			require.NoError(t, err, "batch websocket response must decode")
			require.Len(t, batch, 2, "batch response must retain success and rejection")
			for _, item := range batch {
				assert.Equal(t, want, item.ClientOrderID, "each batch response should filter source markers")
			}
			require.NoError(t, ex.processSpotOrders(t.Context(), []byte(`{"result":[`+spotPayload+`]}`)), "spot push must decode")
			require.Len(t, ex.Websocket.DataHandler.C, 1, "spot push must emit one update")
			pushed, ok := (<-ex.Websocket.DataHandler.C).Data.([]order.Detail)
			require.True(t, ok, "spot push must contain order details")
			require.Len(t, pushed, 1, "spot push must contain one order")
			assert.Equal(t, want, pushed[0].ClientOrderID, "spot push should filter source markers")
		})
	}
}
