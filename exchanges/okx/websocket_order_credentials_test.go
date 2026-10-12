package okx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestCheckWebsocketOrderCredentials(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		creds   accounts.Credentials
		missing bool
	}{
		{name: "same account", creds: accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase"}},
		{name: "different key", creds: accounts.Credentials{Key: "other", Secret: "secret", ClientID: "passphrase"}},
		{name: "different secret", creds: accounts.Credentials{Key: "key", Secret: "other", ClientID: "passphrase"}},
		{name: "different passphrase", creds: accounts.Credentials{Key: "key", Secret: "secret", ClientID: "other"}},
		{name: "different subaccount", creds: accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase", SubAccount: "other"}},
		{name: "one time password", creds: accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase", OneTimePassword: "123456"}},
		{name: "unconfirmed login", creds: accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase"}, missing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := connectOKXWithMockedWebsocket(t, okxOrderWsMock)
			conn, err := ex.Websocket.GetConnection(privateConnection)
			require.NoError(t, err, "private connection must be available")
			loggedIn := accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase"}
			ex.SetCredentials(&loggedIn)
			ex.setWebsocketOrderCredentials(conn, &loggedIn)
			if tc.missing {
				ex.setWebsocketOrderCredentials(conn, nil)
			}
			store := &accounts.ContextCredentialsStore{}
			store.Load(&tc.creds)
			ctx := context.WithValue(t.Context(), accounts.ContextCredentialsFlag, store)
			err = ex.checkWebsocketOrderCredentials(ctx, conn)
			if tc.missing || tc.creds.Key != loggedIn.Key || tc.creds.Secret != loggedIn.Secret || tc.creds.ClientID != loggedIn.ClientID {
				assert.ErrorIs(t, err, errWebsocketOrderCredentialsMismatch, "unconfirmed or different credentials should be rejected")
			} else {
				assert.NoError(t, err, "confirmed matching credentials should be accepted")
			}
			ex.SkipAuthCheck, ex.API.AuthenticatedSupport, ex.API.AuthenticatedWebsocketSupport = false, false, false
			assert.Error(t, ex.checkWebsocketOrderCredentials(t.Context(), conn), "unavailable credentials should return the credential error")
		})
	}
}

func TestOrderAccountRouting(t *testing.T) {
	t.Parallel()
	loggedIn := accounts.Credentials{Key: "account-a", Secret: "secret-a", ClientID: "passphrase-a"}
	other := accounts.Credentials{Key: "account-b", Secret: "secret-b", ClientID: "passphrase-b"}
	optionPair := currency.NewPairWithDelimiter("BTC", "USD-261225-100000-C", "-")
	futuresContract := currency.NewPairWithDelimiter("BTC", "USD-261225", "-")
	type routingCase struct {
		name    string
		pair    currency.Pair
		pending int
		ops     []string
		route   func(context.Context, *Exchange) error
	}
	submit := func(a asset.Item, pair currency.Pair) func(context.Context, *Exchange) error {
		return func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SubmitOrder(ctx, &order.Submit{Exchange: ex.Name, AssetType: a, Pair: pair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.Multi, ReduceOnly: a == asset.Margin})
			return err
		}
	}
	modify := func(a asset.Item, pair currency.Pair) func(context.Context, *Exchange) error {
		return func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ModifyOrder(ctx, &order.Modify{OrderID: "1", AssetType: a, Pair: pair, Amount: 2, Price: 2})
			return err
		}
	}
	cancel := func(a asset.Item, pair currency.Pair) func(context.Context, *Exchange) error {
		return func(ctx context.Context, ex *Exchange) error {
			return ex.CancelOrder(ctx, &order.Cancel{OrderID: "1", AssetType: a, Pair: pair})
		}
	}
	cancelAll := func(ctx context.Context, ex *Exchange) error {
		_, err := ex.CancelAllOrders(ctx, &order.Cancel{AssetType: asset.Spot, Pair: mainPair})
		return err
	}
	for _, tc := range []routingCase{
		{"submit spot", mainPair, 0, []string{"order"}, submit(asset.Spot, mainPair)},
		{"submit margin", mainPair, 0, []string{"order"}, submit(asset.Margin, mainPair)},
		{"submit options", optionPair, 0, []string{"order"}, submit(asset.Options, optionPair)},
		{"submit futures", futuresContract, 0, []string{"order"}, submit(asset.Futures, futuresContract)},
		{"submit perpetual swap", perpetualSwapPair, 0, []string{"order"}, submit(asset.PerpetualSwap, perpetualSwapPair)},
		{"submit spread", spreadPair, 0, []string{"sprd-order"}, submit(asset.Spread, spreadPair)},
		{"modify spot", mainPair, 0, []string{"amend-order"}, modify(asset.Spot, mainPair)},
		{"modify spread", spreadPair, 0, []string{"sprd-amend-order"}, modify(asset.Spread, spreadPair)},
		{"cancel spot", mainPair, 0, []string{"cancel-order"}, cancel(asset.Spot, mainPair)},
		{"cancel spread", spreadPair, 0, []string{"sprd-cancel-order"}, cancel(asset.Spread, spreadPair)},
		{"cancel batch spot", mainPair, 0, []string{"batch-cancel-orders"}, func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelBatchOrders(ctx, []order.Cancel{{OrderID: "1", AssetType: asset.Spot, Pair: mainPair}})
			return err
		}},
		{"cancel all spot", mainPair, 1, []string{"batch-cancel-orders"}, cancelAll},
		{"cancel all spot over two batches", mainPair, 21, []string{"batch-cancel-orders", "batch-cancel-orders"}, cancelAll},
		{"cancel all spot connection drops", mainPair, 21, []string{"batch-cancel-orders"}, cancelAll},
	} {
		for _, creds := range []accounts.Credentials{loggedIn, other} {
			t.Run(tc.name+"/"+creds.Key, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var websocketOps, restKeys []string
				var ex *Exchange
				ex = connectOKXWithMockedWebsocket(t, func(tb testing.TB, payload []byte, conn *gws.Conn) error {
					tb.Helper()
					var req struct {
						ID        string `json:"id"`
						Operation string `json:"op"`
					}
					if err := json.Unmarshal(payload, &req); err != nil {
						return err
					}
					if tc.name == "submit margin" {
						assert.Contains(t, string(payload), `"reduceOnly":"true"`, "margin orders should retain the reduce-only flag")
					}
					mu.Lock()
					websocketOps = append(websocketOps, req.Operation)
					mu.Unlock()
					if req.Operation == "batch-cancel-orders" {
						if strings.Contains(tc.name, "connection drops") {
							ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
						}
						return conn.WriteMessage(gws.TextMessage, []byte(`{"id":"`+req.ID+`","op":"batch-cancel-orders","code":"0","data":[{"ordId":"1","sCode":"0"}]}`))
					}
					return okxOrderWsMock(tb, payload, conn)
				})
				ex.SetCredentials(&loggedIn)
				for _, filter := range []string{privateConnection, businessConnection} {
					conn, err := ex.Websocket.GetConnection(filter)
					require.NoError(t, err, "mock connection must be available")
					ex.setWebsocketOrderCredentials(conn, &loggedIn)
				}
				server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response := `{"code":"0","data":[{"ordId":"1","sCode":"0"}]}`
					switch {
					case strings.HasSuffix(r.URL.Path, "account/config"):
						response = `{"code":"0","data":[{"posMode":"net_mode"}]}`
					case strings.HasSuffix(r.URL.Path, "public/instruments"):
						response = `{"code":"0","data":[{"instId":"` + tc.pair.String() + `","instIdCode":42}]}`
					case strings.HasSuffix(r.URL.Path, "trade/orders-pending"):
						pending := make([]string, tc.pending)
						for i := range pending {
							pending[i] = `{"instId":"` + tc.pair.String() + `","ordId":"` + strconv.Itoa(i+1) + `","side":"buy","ordType":"limit","state":"live"}`
						}
						response = `{"code":"0","data":[` + strings.Join(pending, ",") + `]}`
					default:
						if tc.name == "submit margin" {
							payload, err := io.ReadAll(r.Body)
							assert.NoError(t, err, "margin request should be readable")
							assert.Contains(t, string(payload), `"reduceOnly":"true"`, "REST margin orders should retain the reduce-only flag")
						}
						mu.Lock()
						restKeys = append(restKeys, r.Header.Get("OK-ACCESS-KEY"))
						mu.Unlock()
					}
					_, err := w.Write([]byte(response))
					assert.NoError(t, err, "mock response should write")
				}))
				require.NoError(t, ex.SetHTTPClient(server.Client()), "mock HTTP client must be configured")
				require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "mock endpoint must be configured")
				store := &accounts.ContextCredentialsStore{}
				store.Load(&creds)
				require.NoError(t, tc.route(context.WithValue(t.Context(), accounts.ContextCredentialsFlag, store), ex), "request must reach the requesting account")
				mu.Lock()
				defer mu.Unlock()
				if creds == loggedIn {
					assert.Equal(t, tc.ops, websocketOps, "the logged-in account's request should use its websocket")
					if strings.Contains(tc.name, "connection drops") {
						assert.Equal(t, []string{loggedIn.Key}, restKeys, "later batch should fall back to REST after authentication becomes unavailable")
					} else {
						assert.Empty(t, restKeys, "the logged-in account's request should not use REST")
					}
				} else {
					assert.Empty(t, websocketOps, "another account's request should not use the logged-in websocket")
					count := len(tc.ops)
					if strings.Contains(tc.name, "connection drops") {
						count = 2
					}
					want := make([]string, count)
					for i := range want {
						want[i] = other.Key
					}
					assert.Equal(t, want, restKeys, "another account's request should use REST with its own key")
				}
			})
		}
	}
}

func TestSetWebsocketOrderCredentials(t *testing.T) {
	t.Parallel()
	ex := connectOKXWithMockedWebsocket(t, okxOrderWsMock)
	conn, err := ex.Websocket.GetConnection(privateConnection)
	require.NoError(t, err, "private connection must be available")
	ex.websocketOrderCredentials = nil
	creds := accounts.Credentials{Key: "first"}
	ex.setWebsocketOrderCredentials(conn, &creds)
	creds.Key = "mutated"
	assert.Equal(t, "first", ex.websocketOrderCredentials[conn].Key, "login identity should retain an independent credential snapshot")
	ex.setWebsocketOrderCredentials(conn, &creds)
	assert.Equal(t, "mutated", ex.websocketOrderCredentials[conn].Key, "a subsequent confirmed login should replace the snapshot")
	ex.setWebsocketOrderCredentials(conn, nil)
	assert.Empty(t, ex.websocketOrderCredentials, "invalidated login should remove the identity")
	t.Run("live logins survive reconnect and overflow churn", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		login := func() *liveTestConnection {
			conn := new(liveTestConnection)
			conn.live.Store(true)
			ex.setWebsocketOrderCredentials(conn, &creds)
			return conn
		}
		check := func(connections ...*liveTestConnection) {
			t.Helper()
			require.Len(t, ex.websocketOrderCredentials, len(connections), "only live logins must remain")
			for _, conn := range connections {
				assert.Contains(t, ex.websocketOrderCredentials, conn, "every live login should remain")
			}
		}
		private, business := login(), login()
		for range 100 {
			overflow := login()
			check(private, business, overflow)
			overflow.live.Store(false)
		}
		for range 100 {
			private.live.Store(false)
			business.live.Store(false)
			private, business = login(), login()
			check(private, business)
		}
	})
}

func TestWsAuthenticateConnection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		response string
		success  bool
	}{
		{name: "confirmed", response: `{"event":"login","code":"0"}`, success: true},
		{name: "rejected", response: `{"event":"login","code":"60009","msg":"rejected"}`},
		{name: "malformed code", response: `{"event":"login","code":"invalid"}`},
		{name: "missing code", response: `{"event":"login"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var orders atomic.Int64
			ex := connectOKXWithMockedWebsocket(t, func(tb testing.TB, payload []byte, conn *gws.Conn) error {
				tb.Helper()
				var req WebsocketAuthLogin
				if err := json.Unmarshal(payload, &req); err != nil {
					return err
				}
				if req.Operation == operationLogin {
					assert.Equal(tb, "account-a", req.Arguments[0].APIKey, "login should use the requested account")
					assert.Equal(tb, "passphrase", req.Arguments[0].Passphrase, "login should use the requested passphrase")
					return conn.WriteMessage(gws.TextMessage, []byte(tc.response))
				}
				orders.Add(1)
				return okxOrderWsMock(tb, payload, conn)
			})
			creds := accounts.Credentials{Key: "account-a", Secret: "secret", ClientID: "passphrase"}
			ex.SetCredentials(&creds)
			conn, err := ex.Websocket.GetConnection(privateConnection)
			require.NoError(t, err, "private connection must be available")
			ex.setWebsocketOrderCredentials(conn, &creds)
			err = ex.wsAuthenticateConnection(t.Context(), conn)
			if !tc.success {
				require.ErrorIs(t, err, request.ErrAuthRequestFailed, "failed login must retain the authentication sentinel")
				assert.ErrorIs(t, ex.checkWebsocketOrderCredentials(t.Context(), conn), errWebsocketOrderCredentialsMismatch, "failed login should remove the prior account identity")
				return
			}
			require.NoError(t, err, "real mocked login acknowledgement must authenticate")
			assert.NoError(t, ex.checkWebsocketOrderCredentials(t.Context(), conn), "confirmed login should register its account")
			other := creds
			other.Key = "account-b"
			store := &accounts.ContextCredentialsStore{}
			store.Load(&other)
			ctx := context.WithValue(t.Context(), accounts.ContextCredentialsFlag, store)
			for _, operation := range []string{"order", "batch-orders", "amend-order", "batch-amend-orders", "cancel-order", "batch-cancel-orders", "mass-cancel", "sprd-order", "sprd-amend-order", "sprd-cancel-order", "sprd-mass-cancel"} {
				assert.ErrorIs(t, ex.SendAuthenticatedWebsocketRequest(ctx, placeOrderEPL, ex.MessageID(), operation, []PlaceOrderRequestParam{{}}, nil), errWebsocketOrderCredentialsMismatch, "another account's order should be rejected before sending")
			}
			assert.Zero(t, orders.Load(), "mismatched account requests should never reach the socket")
			var response any
			require.NoError(t, ex.SendAuthenticatedWebsocketRequest(t.Context(), placeOrderEPL, ex.MessageID(), "order", []PlaceOrderRequestParam{{}}, &response), "matching account order must use the authenticated socket")
			assert.EqualValues(t, 1, orders.Load(), "matching account order should reach the socket")
			newConn, err := ex.Websocket.CreateTestConnection(privateConnection)
			require.NoError(t, err, "replacement connection must be created")
			assert.ErrorIs(t, ex.checkWebsocketOrderCredentials(t.Context(), newConn), errWebsocketOrderCredentialsMismatch, "replacement connection should require its own successful login")
		})
	}
}

// liveTestConnection exposes the same lifecycle status as manager-owned connections.
type liveTestConnection struct {
	websocket.Connection
	live atomic.Bool
}

func (c *liveTestConnection) IsConnected() bool { return c.live.Load() }

func TestCanUseWebsocketForAccount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, filter                 string
		disabled, missing, different bool
		want                         bool
	}{
		{name: "private matching", filter: privateConnection, want: true},
		{name: "business matching", filter: businessConnection, want: true},
		{name: "missing connection", filter: "unknown"},
		{name: "disabled", filter: privateConnection, disabled: true},
		{name: "unconfirmed", filter: privateConnection, missing: true},
		{name: "different account", filter: businessConnection, different: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := connectOKXWithMockedWebsocket(t, okxOrderWsMock)
			if tc.disabled {
				ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
			}
			if tc.missing {
				conn, err := ex.Websocket.GetConnection(tc.filter)
				require.NoError(t, err, "connection must exist")
				ex.setWebsocketOrderCredentials(conn, nil)
			}
			ctx := t.Context()
			if tc.different {
				store := &accounts.ContextCredentialsStore{}
				store.Load(&accounts.Credentials{Key: "other", Secret: "secret", ClientID: "passphrase"})
				ctx = context.WithValue(ctx, accounts.ContextCredentialsFlag, store)
			}
			assert.Equal(t, tc.want, ex.canUseWebsocketForAccount(ctx, tc.filter), "transport selection should require a confirmed matching account")
		})
	}
}
