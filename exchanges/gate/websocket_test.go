package gate

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	v14 "github.com/thrasher-corp/gocryptotrader/config/versions/v14"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetWebsocketPingHandler(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		channel string
		err     error
	}{
		{optionsPingChannel, nil},
		{futuresPingChannel, nil},
		{spotPingChannel, nil},
		{"dong", errInvalidPingChannel},
	} {
		t.Run(tc.channel, func(t *testing.T) {
			t.Parallel()
			got, err := getWSPingHandler(tc.channel)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, time.Second*10, got.Delay)
			require.Equal(t, gws.TextMessage, got.MessageType)
			require.Contains(t, string(got.Message), tc.channel)
		})
	}
}

func TestParseWebsocketAPIResponse(t *testing.T) {
	t.Parallel()

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()
		got, err := parseWebsocketAPIResponse([]byte(`{"request_id":"1","header":{"response_time":"1677816784084","status":"200","channel":"spot.order_status","event":"api","client_id":"::1-0x1","conn_id":"c1","trace_id":"t1"},"data":{"result":{"id":"1"}}}`))
		require.NoError(t, err, "parseWebsocketAPIResponse must not error")
		exp := &WebsocketAPIResponse{
			Header: Header{ResponseTime: types.Time(time.UnixMilli(1677816784084)), Status: http.StatusOK, Channel: "spot.order_status", Event: "api", ClientID: "::1-0x1", ConnectionID: "c1", TraceID: "t1"},
			Data:   json.RawMessage(`{"result":{"id":"1"}}`),
		}
		assert.Equal(t, exp, got, "parseWebsocketAPIResponse should decode the reply")
	})

	t.Run("malformed reply", func(t *testing.T) {
		t.Parallel()
		got, err := parseWebsocketAPIResponse([]byte(`{`))
		assert.Error(t, err, "parseWebsocketAPIResponse should reject malformed JSON")
		assert.Nil(t, got, "parseWebsocketAPIResponse should return no reply for malformed JSON")
	})

	t.Run("null reply", func(t *testing.T) {
		t.Parallel()
		got, err := parseWebsocketAPIResponse([]byte(`null`))
		assert.ErrorIs(t, err, common.ErrNoResponse, "parseWebsocketAPIResponse should reject a null reply")
		assert.Nil(t, got, "parseWebsocketAPIResponse should return no reply for a null reply")
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()
		got, err := parseWebsocketAPIResponse([]byte(`{"request_id":"1","header":{"response_time":"1677816784084","status":"401","channel":"spot.login","event":"api"},"data":{"errs":{"label":"INVALID_KEY","message":"Invalid key provided"}}}`))
		assert.EqualError(t, err, "INVALID_KEY: Invalid key provided", "parseWebsocketAPIResponse should return Gate's error label and message")
		assert.Nil(t, got, "parseWebsocketAPIResponse should return no reply for a failed request")
	})

	t.Run("failed request with malformed errors", func(t *testing.T) {
		t.Parallel()
		got, err := parseWebsocketAPIResponse([]byte(`{"request_id":"1","header":{"response_time":"1677816784084","status":"400","channel":"spot.login","event":"api"},"data":{"errs":"invalid"}}`))
		assert.Error(t, err, "parseWebsocketAPIResponse should reject errors that do not decode")
		assert.Nil(t, got, "parseWebsocketAPIResponse should return no reply for a failed request")
	})

	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "failed request with null data", data: `null`},
		{name: "failed request without errors", data: `{}`},
		{name: "failed request with empty errors", data: `{"errs":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseWebsocketAPIResponse([]byte(`{"request_id":"1","header":{"response_time":"1677816784084","status":"400","channel":"spot.login","event":"api"},"data":` + tc.data + `}`))
			assert.ErrorIs(t, err, common.ErrNoResponse, "parseWebsocketAPIResponse should reject a failed request without Gate's error")
			assert.ErrorContains(t, err, "status 400", "parseWebsocketAPIResponse should report the failed request's status")
			assert.Nil(t, got, "parseWebsocketAPIResponse should return no reply for a failed request")
		})
	}
}

func TestGenerateWebsocketSignature(t *testing.T) {
	t.Parallel()
	// Expected values follow Gate's documented scheme: HMAC-SHA512 over channel=<channel>&event=<event>&time=<time>
	for _, tc := range []struct {
		name    string
		event   string
		channel string
		time    uint64
		exp     string
	}{
		{name: "subscribe", event: subscribeEvent, channel: futuresBalancesChannel, time: 1700000000, exp: "5ebba086069eae378fc48ba2a1fb8b5a680fd042b2541467402ee8f10d36a33cce6a4f026d87fcdf9810062889ccfb75f05a227b7815a33f2973581056763985"},
		{name: "unsubscribe", event: unsubscribeEvent, channel: spotBalancesChannel, time: 1700000001, exp: "32115f186d1f218c8811623b0eb3d3edd0226d2b2e2dee34d5dbc90209091e85eb2993f5f1ee92a2297d84c2385d80c9c9279f7d99b30c180495b3f9d43b8201"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sig, err := new(Exchange).generateWebsocketSignature("mock-api-secret", tc.event, tc.channel, tc.time)
			require.NoError(t, err, "generateWebsocketSignature must not error")
			assert.Equal(t, tc.exp, sig, "generateWebsocketSignature should sign the channel, event and time as Gate documents")
		})
	}
}

type websocketBalancesTest struct {
	input       []byte
	err         error
	deployCreds bool
	expected    accounts.SubAccounts
}

func TestProcessSpotBalances(t *testing.T) { //nolint:tparallel // Sequential tests, do not use t.Parallel(); Some timestamps are deliberately identical from trading activity
	t.Parallel()
	e := new(Exchange)
	e.SetDefaults()
	e.Name = "ProcessSpotBalancesTest"
	e.Accounts = accounts.MustNewAccounts(e)

	// Gate's websocket user value identifies the primary account, which REST saves with an empty ID, so the balances
	// carry that ID and their arrival time
	for i, tc := range []websocketBalancesTest{
		{
			input: []byte(`[{"timestamp":"1755718222","currency":"USDT"}]`),
			err:   exchange.ErrCredentialsAreEmpty,
		},
		{
			deployCreds: true,
			input:       []byte(`[{"timestamp":"1755718222"}]`),
			err:         currency.ErrCurrencyCodeEmpty,
		},
		{
			deployCreds: true,
			input:       []byte(`[{"timestamp":"1755718222","timestamp_ms":"1755718222394","user":"12870774","currency":"USDT","change":"0","total":"3087.01142272991036062136","available":"3081.68642272991036062136","freeze":"5.325","freeze_change":"5.32500000000000000000","change_type":"order-create"}]`),
			expected: accounts.SubAccounts{
				{
					ID:        "",
					AssetType: asset.Spot,
					Balances: accounts.CurrencyBalances{
						currency.USDT: accounts.Balance{
							Currency:               currency.USDT,
							Total:                  3087.01142272991036062136,
							Free:                   3081.68642272991036062136,
							Hold:                   5.325,
							AvailableWithoutBorrow: 3081.68642272991036062136,
						},
					},
				},
			},
		},
		{
			deployCreds: true,
			input:       []byte(`[{"timestamp":"1755718222","timestamp_ms":"1755718222394","user":"12870774","currency":"USDT","change":"-3.99375000000000000000","total":"3083.01767272991036062136","available":"3081.68642272991036062136","freeze":"1.33125","freeze_change":"-3.99375000000000000000","change_type":"order-match"}]`),
			expected: accounts.SubAccounts{
				{
					ID:        "",
					AssetType: asset.Spot,
					Balances: accounts.CurrencyBalances{
						currency.USDT: accounts.Balance{
							Currency:               currency.USDT,
							Total:                  3083.01767272991036062136,
							Free:                   3081.68642272991036062136,
							Hold:                   1.33125,
							AvailableWithoutBorrow: 3081.68642272991036062136,
						},
					},
				},
			},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			// Sequential tests, do not use t.Parallel(); Some timestamps are deliberately identical from trading activity
			ctx := t.Context()
			if tc.deployCreds {
				ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "test", Secret: "test"})
			}
			err := e.processSpotBalances(ctx, tc.input)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err, "processSpotBalances must not error")
				checkAccountChange(ctx, t, e, &tc)
			}
		})
	}
}

func TestProcessBalancePushData(t *testing.T) {
	t.Parallel()

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		malformed := []byte(`{`)
		assert.Equal(t, json.Unmarshal(malformed, new([]*WebsocketBalance)), ex.processBalancePushData(t.Context(), malformed, asset.USDTMarginedFutures), "processBalancePushData should return the decode error of malformed JSON")
	})

	e := new(Exchange)
	e.SetDefaults()
	e.Name = "ProcessFuturesBalancesTest"
	e.Accounts = accounts.MustNewAccounts(e)

	// Gate's websocket user value identifies the primary account, not a separate subaccount.
	// The ID must remain empty to match REST snapshots and prevent portfolio double counting.
	usdtLower := currency.USDT.Lower()

	for i, tc := range []websocketBalancesTest{
		{
			input: []byte(`[{"timestamp":"1755718222"}]`),
			err:   exchange.ErrCredentialsAreEmpty,
		},
		{
			deployCreds: true,
			input:       []byte(`[{"balance":2214.191673190433,"change":-0.0025776,"currency":"usdt","text":"TCOM_USDT:263179103241933596","time":1755738515,"time_ms":1755738515671,"type":"fee","user":"12870774"}]`),
			expected: accounts.SubAccounts{
				{
					ID:        "",
					AssetType: asset.USDTMarginedFutures,
					Balances: accounts.CurrencyBalances{
						usdtLower: accounts.Balance{
							Currency:               usdtLower,
							Total:                  2214.191673190433,
							Free:                   2214.191673190433,
							AvailableWithoutBorrow: 2214.191673190433,
						},
					},
				},
			},
		},
		{
			deployCreds: true,
			input:       []byte(`[{"balance":2214.189114310433,"change":-0.00255888,"currency":"usdt","text":"TCOM_USDT:263179103241933644","time":1755738516,"time_ms":1755738516430,"type":"fee","user":"12870774"}]`),
			expected: accounts.SubAccounts{
				{
					ID:        "",
					AssetType: asset.USDTMarginedFutures,
					Balances: accounts.CurrencyBalances{
						usdtLower: accounts.Balance{
							Currency:               usdtLower,
							Total:                  2214.189114310433,
							Free:                   2214.189114310433,
							AvailableWithoutBorrow: 2214.189114310433,
						},
					},
				},
			},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			// Sequential tests, do not use t.Parallel(); Some timestamps are deliberately identical from trading activity
			ctx := t.Context()
			if tc.deployCreds {
				ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "test", Secret: "test"})
			}
			err := e.processBalancePushData(ctx, tc.input, asset.USDTMarginedFutures)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err, "processBalancePushData must not error")
				checkAccountChange(ctx, t, e, &tc)
			}
		})
	}
}

func TestProcessFuturesBalanceCapturedPayloads(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	ex.Name = "ProcessFuturesBalanceCapturedPayloads"
	ex.Accounts = accounts.MustNewAccounts(ex)
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
	// REST uses the canonical empty subaccount ID, while websocket payloads include Gate's primary-account user ID.
	// Both transports must update one holding or portfolio aggregation will sum the same futures balance twice.
	restSnapshot := accounts.NewSubAccount(asset.USDTMarginedFutures, "")
	restSnapshot.Balances.Set(currency.USDT, accounts.Balance{
		Total: 6106.7961637458,
		Hold:  1500,
		Free:  4606.7961637458,
	})
	require.NoError(t, ex.Accounts.Save(ctx, accounts.SubAccounts{restSnapshot}, true),
		"Accounts.Save must seed the REST balance snapshot")
	// Captured from live Gate websocket data on 2026-09-03.
	// Treat these payloads as semi-trusted until independently confirmed.
	payloads := [][]byte{
		[]byte(`[{"balance":6625.2967002542,"change":0.0008823675,"text":"SCRT_USDT","time":1788148805,"time_ms":1788148805954,"type":"fund","user":"12870774","currency":"usdt"}]`),
		[]byte(`[{"balance":6625.2967002542,"change":0.0008823675,"text":"SCRT_USDT","time":1788148805,"time_ms":1788148805954,"type":"fund","user":"12870774","currency":"usdt"}]`),
		[]byte(`[{"balance":6521.586841973412,"change":-0.01556544,"text":"RAVE_USDT:294985777215626819","time":1788387249,"time_ms":1788387249995,"type":"fee","user":"12870774","currency":"usdt"}]`),
		[]byte(`[{"balance":6522.937261628622,"change":1.35041965521,"text":"RAVE_USDT:294985777215626819","time":1788387249,"time_ms":1788387249995,"type":"pnl","user":"12870774","currency":"usdt"}]`),
	}
	wantBalances := []float64{6625.2967002542, 6625.2967002542, 6521.586841973412, 6522.937261628622}
	names := []string{"fund", "repeated fund", "fee", "pnl"}

	// Cases update ex's stored balance and read its relay in order
	for i := range payloads {
		t.Run(names[i], func(t *testing.T) {
			startedAt := time.Now()
			require.NoError(t, ex.processBalancePushData(ctx, payloads[i], asset.USDTMarginedFutures),
				"processBalancePushData must process the captured balance update")
			require.Len(t, ex.Websocket.DataHandler.C, 1, "processBalancePushData must relay the captured balance update")
			message := <-ex.Websocket.DataHandler.C
			changes, ok := message.Data.(accounts.SubAccounts)
			require.True(t, ok, "captured balance payload must emit subaccount changes")
			require.Len(t, changes, 1, "captured balance payload must emit one subaccount change")
			balance, ok := changes[0].Balances[currency.USDT.Lower()]
			require.True(t, ok, "captured balance payload must contain USDT")
			assert.Equal(t, wantBalances[i], balance.Total, "total balance should be preserved")
			assert.Equal(t, 1500.0, balance.Hold, "held margin should survive a websocket balance update")
			assert.Equal(t, wantBalances[i]-1500, balance.Free, "free balance should exclude held margin")
			assert.False(t, balance.UpdatedAt.Before(startedAt), "websocket balance timestamp should use local arrival order")
		})
	}

	credentials, err := ex.GetCredentials(ctx)
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.GetBalance("", credentials, asset.USDTMarginedFutures, currency.USDT.Lower())
	require.NoError(t, err, "GetBalance must return the latest captured balance")
	latestBalance := wantBalances[len(wantBalances)-1]
	assert.Equal(t, latestBalance, stored.Total, "stored balance should contain the latest update")
	collated, err := ex.Accounts.CurrencyBalances(credentials, asset.USDTMarginedFutures)
	require.NoError(t, err, "CurrencyBalances must return the collated futures balance")
	assert.Equal(t, latestBalance, collated[currency.USDT].Total,
		"websocket balance should replace the REST snapshot without being double counted")
	refreshedRESTSnapshot := accounts.NewSubAccount(asset.USDTMarginedFutures, "")
	refreshedRESTSnapshot.Balances.Set(currency.USDT, accounts.Balance{
		Total: latestBalance,
		Hold:  100,
		Free:  latestBalance - 100,
	})
	require.NoError(t, ex.Accounts.Save(ctx, accounts.SubAccounts{refreshedRESTSnapshot}, true),
		"REST arrival must not be rejected after a websocket event stamped by Gate's clock")

	staleHoldSnapshot := accounts.NewSubAccount(asset.USDTMarginedFutures, "")
	staleHoldSnapshot.Balances.Set(currency.USDT, accounts.Balance{Total: 6106.7961637458, Hold: 5000, Free: 1106.7961637458})
	require.NoError(t, ex.Accounts.Save(ctx, accounts.SubAccounts{staleHoldSnapshot}, true),
		"Accounts.Save must seed a stale held margin")
	require.NoError(t, ex.processBalancePushData(ctx,
		[]byte(`[{"balance":1000.5,"time":1788148806,"time_ms":1788148806000,"user":"12870774","currency":"usdt"}]`),
		asset.USDTMarginedFutures), "processBalancePushData must floor free funds during a drawdown")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "processBalancePushData must relay the drawdown")
	message := <-ex.Websocket.DataHandler.C
	changes, ok := message.Data.(accounts.SubAccounts)
	require.True(t, ok, "captured balance payload must emit subaccount changes")
	drawnDown := changes[0].Balances[currency.USDT.Lower()]
	assert.Equal(t, 5000.0, drawnDown.Hold, "held margin should survive a temporary balance drawdown")
	assert.Zero(t, drawnDown.Free, "free balance should not become negative")
	require.NoError(t, ex.processBalancePushData(ctx,
		[]byte(`[{"balance":6106.7961637458,"time":1788148807,"time_ms":1788148807000,"user":"12870774","currency":"usdt"}]`),
		asset.USDTMarginedFutures), "processBalancePushData must process balance recovery")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "processBalancePushData must relay the recovery")
	message = <-ex.Websocket.DataHandler.C
	changes, ok = message.Data.(accounts.SubAccounts)
	require.True(t, ok, "captured balance payload must emit subaccount changes")
	recovered := changes[0].Balances[currency.USDT.Lower()]
	assert.Equal(t, 5000.0, recovered.Hold, "held margin should remain intact after balance recovery")
	assert.InDelta(t, 1106.7961637458, recovered.Free, 1e-12,
		"free balance should recover without being overstated")
}

func checkAccountChange(ctx context.Context, t *testing.T, exch *Exchange, tc *websocketBalancesTest) {
	t.Helper()

	require.Len(t, exch.Websocket.DataHandler.C, 1)
	payload := <-exch.Websocket.DataHandler.C
	received, ok := payload.Data.(accounts.SubAccounts)
	require.Truef(t, ok, "Expected account changes, got %T", payload)

	require.Lenf(t, received, len(tc.expected), "Expected %d changes, got %d", len(tc.expected), len(received))
	for i := range tc.expected {
		for c, expected := range tc.expected[i].Balances {
			if expected.UpdatedAt.IsZero() {
				receivedBalance := received[i].Balances[c]
				assert.False(t, receivedBalance.UpdatedAt.IsZero(), "balance arrival timestamp should be populated")
				expected.UpdatedAt = receivedBalance.UpdatedAt
				tc.expected[i].Balances[c] = expected
			}
		}
	}
	require.Equal(t, tc.expected, received)

	creds, err := exch.GetCredentials(ctx)
	require.NoError(t, err, "GetCredentials must not error")

	for _, change := range received {
		bal := slices.Collect(maps.Values(change.Balances))[0]
		stored, err := exch.Accounts.GetBalance(change.ID, creds, change.AssetType, bal.Currency)
		require.NoError(t, err, "GetBalance must not error")
		assert.Equal(t, bal.Free, stored.Free, "free balance should equal with accounts stored value")
	}
}

func TestProcessOrderbookUpdateWithSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("recovery snapshot ends resubscription", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			bids      string
			fillRelay bool
		}{
			{name: "invalid snapshot", bids: `[["100","1"],["100","1"]]`},
			{name: "relay full", bids: `[["100","1"],["99","1"]]`, fillRelay: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				e := new(Exchange)
				require.NoError(t, testexch.Setup(e), "test instance setup must not error")
				e.Name = t.Name()
				e.ValidateOrderbook = true
				pair := currency.NewBTCUSDT()
				initial := []byte(`{"t":1757377580046,"full":true,"s":"ob.BTC_USDT.50","u":100,"b":[["100","1"],["99","1"]],"a":[["101","1"]]}`)
				require.NoError(t, e.processOrderbookUpdateWithSnapshot(t.Context(), nil, initial, time.Now(), asset.Spot), "initial snapshot must load")
				e.wsOBResubMgr.lookup[key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: asset.Spot}] = true
				if tc.fillRelay {
					for len(e.Websocket.DataHandler.C) < cap(e.Websocket.DataHandler.C) {
						require.NoError(t, e.Websocket.DataHandler.Send(t.Context(), "backlog"), "relay must accept backlog")
					}
				}
				recovery := []byte(`{"t":1757377580046,"full":true,"s":"ob.BTC_USDT.50","u":200,"b":` + tc.bids + `,"a":[["101","1"]]}`)
				require.Error(t, e.processOrderbookUpdateWithSnapshot(t.Context(), nil, recovery, time.Now(), asset.Spot), "failed recovery snapshot must return an error")
				assert.False(t, e.wsOBResubMgr.IsResubscribing(pair, asset.Spot), "a recovery snapshot should end the resubscription so later updates are not dropped")
			})
		}
	})

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e))
	e.Name = t.Name()
	e.ValidateOrderbook = true
	e.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: spotOrderbookV2, Asset: asset.Spot, Levels: 50},
	}
	subs, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err)

	baseConn, err := e.Websocket.CreateTestConnection(asset.Spot)
	require.NoError(t, err, "test connection creation must succeed")
	conn := &FixtureConnection{Connection: baseConn}
	require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, conn), "fixture connection registration must succeed")
	err = e.Websocket.AddSubscriptions(conn, subs...)
	require.NoError(t, err)

	e.wsOBResubMgr.lookup[key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}] = true

	ctx := t.Context()
	// Cases build on e's orderbook and resubscription state, so run in order
	for _, tc := range []struct {
		name         string
		payload      []byte
		err          error
		assertUpdate bool
	}{
		{name: "invalid timestamp", payload: []byte(`{"t":"bingbong"}`), err: types.ErrInvalidTimestampFormat},
		{name: "channel without a pair", payload: []byte(`{"s":"ob.50"}`), err: common.ErrMalformedData},
		{name: "empty pair", payload: []byte(`{"s":"ob..50"}`), err: currency.ErrCreatingPair},
		{
			name:    "update dropped while resubscribing",
			payload: []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
		},
		{name: "snapshot without an update ID", payload: []byte(`{"s":"ob.BTC_USDT.50","full":true}`), err: orderbook.ErrLastUpdatedNotSet},
		{
			name:    "snapshot ends resubscription",
			payload: []byte(`{"t":1757377580046,"full":true,"s":"ob.BTC_USDT.50","u":27053258981,"b":[["111666","0.131287"],["111665.3","0.048403"],["111665.2","0.268681"],["111665.1","0.153269"],["111664.9","0.004"],["111663.8","0.010919"],["111663.7","0.214867"],["111661.8","0.268681"],["111659.4","0.01144"],["111659.3","0.184127"],["111658.4","0.268681"],["111658.3","0.11897"],["111656.9","0.00653"],["111656.7","0.184127"],["111656.1","0.040381"],["111655","0.044859"],["111654.9","0.268681"],["111654.8","0.033575"],["111653.9","0.184127"],["111653.6","0.601785"],["111653.5","0.017118"],["111651.7","0.160346"],["111651.6","0.184127"],["111651.5","0.268681"],["111650.1","0.09042"],["111647.9","0.191292"],["111647.5","0.268681"],["111646","0.098528"],["111645.9","0.1443"],["111645.6","0.184127"],["111643.8","1.015409"],["111643","0.099889"],["111641.5","0.004925"],["111641.2","0.179895"],["111641.1","0.184127"],["111640.7","0.268681"],["111638.6","0.184912"],["111638.4","0.010182"],["111637.6","0.026862"],["111637.5","0.09042"],["111636.6","0.184127"],["111634.8","0.129187"],["111634.7","0.014213"],["111633.9","0.268681"],["111632.1","0.184127"],["111631.8","0.1443"],["111631.6","0.027"],["111631.3","0.089539"],["111630.3","0.00001"],["111629.6","0.000029"]],"a":[["111666.1","0.818887"],["111668.3","0.008062"],["111668.5","0.005399"],["111670.3","0.043892"],["111670.4","0.019653"],["111673.7","0.046898"],["111674.1","0.004227"],["111674.4","0.026258"],["111674.8","0.09042"],["111674.9","0.268681"],["111675","0.004227"],["111676","0.004227"],["111676.8","0.005"],["111677","0.004227"],["111678.1","0.077789"],["111678.2","0.210991"],["111678.3","0.268681"],["111678.4","0.025039"],["111678.5","0.051456"],["111679.2","0.007163"],["111679.5","0.013019"],["111681.5","0.036343"],["111681.7","0.268681"],["111682.9","0.184127"],["111685.2","0.184127"],["111685.8","0.040538"],["111686.4","0.201931"],["111687.3","0.03"],["111687.4","0.09042"],["111687.5","0.452808"],["111687.6","1.815093"],["111691.9","0.139287"],["111692.2","0.184127"],["111693.7","0.268681"],["111694.3","1.05115"],["111694.5","0.184127"],["111697","0.184127"],["111697.1","0.268681"],["111697.4","0.0967"],["111698.7","0.1443"],["111699.5","0.014213"],["111700.2","0.601783"],["111700.7","0.09042"],["111700.9","0.367517"],["111701.5","0.184127"],["111705.2","0.017703"],["111706","0.184127"],["111707.6","0.268681"],["111709.9","0.1443"],["111710.2","0.004"]]}`),
		},
		{
			name:         "incremental update applies",
			payload:      []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
			assertUpdate: true,
		},
		{
			name:    "out of order update resubscribes",
			payload: []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := e.processOrderbookUpdateWithSnapshot(ctx, conn, tc.payload, time.Now(), asset.Spot)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			if tc.assertUpdate {
				book, err := e.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.Spot)
				require.NoError(t, err, "GetOrderbook must not error")
				assert.True(t, book.ValidateOrderbook, "V2 snapshot should retain configured validation")
				assert.Equal(t, int64(27053258987), book.LastUpdateID, "incremental update should advance the update ID")
				require.NotEmpty(t, book.Bids, "updated book must contain bids")
				assert.Equal(t, 0.146841, book.Bids[0].Amount, "incremental update should replace the matching bid amount")
			}
		})
	}

	require.Eventually(t, func() bool {
		sub := e.Websocket.GetSubscription(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "ob.BTC_USDT.50", Asset: asset.Spot}})
		return sub != nil && sub.State() == subscription.SubscribedState
	}, 10*time.Second, 10*time.Millisecond, "out-of-order update must successfully resubscribe in the background")
}

func TestShippedConfigsMatchDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "repository root must be found")
	expected, err := json.Marshal(defaultSubscriptions)
	require.NoError(t, err, "default subscriptions must marshal")
	for _, name := range []string{"config_example.json", filepath.Join("testdata", "configtest.json")} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join(root, name))
			require.NoError(t, err, "shipped config must be readable")
			var cfg config.Config
			require.NoError(t, json.Unmarshal(data, &cfg), "shipped config must unmarshal")
			exchangeConfig, err := cfg.GetExchangeConfig("Gate")
			require.NoError(t, err, "shipped config must contain Gate")
			require.NotNil(t, exchangeConfig.Features, "Gate features must be configured")
			got, err := json.Marshal(exchangeConfig.Features.Subscriptions)
			require.NoError(t, err, "shipped subscriptions must marshal")
			assert.JSONEq(t, string(expected), string(got), "shipped subscriptions should match defaultSubscriptions")
		})
	}
}

func TestDefaultSpotOrderbookSubscription(t *testing.T) {
	t.Parallel()
	var legacy, v2 *subscription.Subscription
	for _, sub := range defaultSubscriptions {
		if sub.Asset != asset.Spot {
			continue
		}
		switch sub.Channel {
		case subscription.OrderbookChannel:
			legacy = sub
		case spotOrderbookV2:
			v2 = sub
		}
	}
	require.NotNil(t, legacy, "legacy spot orderbook subscription must be defined")
	assert.False(t, legacy.Enabled, "legacy spot orderbook subscription should be disabled")
	require.NotNil(t, v2, "V2 spot orderbook subscription must be defined")
	assert.True(t, v2.Enabled, "V2 spot orderbook subscription should be enabled")
	assert.Equal(t, 50, v2.Levels, "V2 spot orderbook subscription should request 50 levels")
}

func TestV14MigrationGeneratesValidSpotSubscriptions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		upgrade bool
		input   string
		channel string
	}{
		{
			name:    "uppercase upgrade",
			upgrade: true,
			input:   `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"SPOT","interval":"100ms"},{"enabled":false,"channel":"spot.obu","asset":"Spot","levels":50}]}}`,
			channel: spotOrderbookV2,
		},
		{
			name:    "mixed-case downgrade",
			input:   `{"features":{"subscriptions":[{"enabled":false,"channel":"orderbook","asset":"Spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"SPOT","levels":50}]}}`,
			channel: spotOrderbookUpdateChannel,
		},
		{
			name:    "wildcard V2 downgrade",
			input:   `{"features":{"subscriptions":[{"enabled":false,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50},{"enabled":true,"channel":"spot.obu","asset":"all","levels":50,"pairs":"BTC_USDT"}]}}`,
			channel: spotOrderbookV2,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			version := new(v14.Version)
			migrate := version.DowngradeExchange
			if test.upgrade {
				migrate = version.UpgradeExchange
			}
			migrated, err := migrate(t.Context(), []byte(test.input))
			require.NoError(t, err, "migration must not error")
			var migratedConfig config.Exchange
			require.NoError(t, json.Unmarshal(migrated, &migratedConfig), "migrated configuration must unmarshal")
			require.NotNil(t, migratedConfig.Features, "migrated configuration features must not be nil")

			e := new(Exchange)
			require.NoError(t, testexch.Setup(e), "test instance setup must not error")
			e.Config.Features.Subscriptions = migratedConfig.Features.Subscriptions
			e.SetSubscriptionsFromConfig()
			subs, err := e.generateSubscriptionsSpot(t.Context())
			require.NoError(t, err, "migrated subscriptions must expand and validate")
			assert.NotEmpty(t, subs, "migrated configuration should generate subscriptions")
			for _, s := range subs {
				assert.Equal(t, test.channel, channelName(s), "migrated subscriptions should use the expected orderbook feed")
			}
		})
	}
}

func TestProcessSpotOrders(t *testing.T) {
	t.Parallel()

	t.Run("order lifecycle", func(t *testing.T) {
		t.Parallel()
		pair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
		created, updated := time.UnixMilli(1694655225315), time.UnixMilli(1694655226315)
		const times = `"create_time_ms":"1694655225315","update_time_ms":"1694655226315"`
		for _, tc := range []struct {
			name  string
			order string
			exp   order.Detail
		}{
			{
				name:  "open limit sell",
				order: `{"id":"11","text":"t-open","event":"put","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"0.5","price":"70000","left":"0.5","filled_total":"0","filled_amount":"0","avg_deal_price":"0","fee":"0","fee_currency":"USDT",` + times + `}`,
				exp:   order.Detail{OrderID: "11", ClientOrderID: "t-open", Side: order.Sell, Type: order.Limit, AssetType: asset.Spot, Price: 70000, Amount: 0.5},
			},
			{
				name:  "partly filled limit buy",
				order: `{"id":"12","text":"apiv4","event":"update","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"margin","side":"buy","amount":"0.5","price":"60000","left":"0.125","filled_total":"22500","filled_amount":"0.375","avg_deal_price":"60000","fee":"0.000375","fee_currency":"BTC",` + times + `}`,
				exp:   order.Detail{OrderID: "12", Side: order.Buy, Type: order.Limit, AssetType: asset.Margin, Price: 60000, Amount: 0.5, ExecutedAmount: 0.375},
			},
			{
				name:  "filled market buy",
				order: `{"id":"13","text":"t-mb","event":"finish","finish_as":"filled","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"buy","amount":"100","price":"0","left":"0.25","filled_total":"99.75","filled_amount":"0.0015","avg_deal_price":"66500","fee":"0.0000015","fee_currency":"BTC",` + times + `}`,
				exp:   order.Detail{OrderID: "13", ClientOrderID: "t-mb", Status: order.Filled, Side: order.Buy, Type: order.Market, AssetType: asset.Spot, Amount: 100, ExecutedAmount: 99.75, AverageExecutedPrice: 66500, Cost: 99.75, CostAsset: currency.USDT, Fee: 0.0000015, FeeAsset: currency.BTC},
			},
			{
				name:  "filled market sell sized in the base currency",
				order: `{"id":"14","event":"finish","finish_as":"filled","currency_pair":"BTC_USDT","type":"market","account":"cross_margin","side":"sell","amount":"0.25","price":"0","left":"0","filled_total":"16662.5","filled_amount":"0.25","avg_deal_price":"66650","fee":"16.66","fee_currency":"USDT",` + times + `}`,
				exp:   order.Detail{OrderID: "14", Status: order.Filled, Side: order.Sell, Type: order.Market, AssetType: asset.CrossMargin, Amount: 0.25, ExecutedAmount: 0.25, AverageExecutedPrice: 66650, Cost: 16662.5, CostAsset: currency.USDT, Fee: 16.66, FeeAsset: currency.USDT},
			},
			{
				name:  "cancelled limit sell",
				order: `{"id":"15","event":"finish","finish_as":"cancelled","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"1",` + times + `}`,
				exp:   order.Detail{OrderID: "15", Status: order.Cancelled, Side: order.Sell, Type: order.Limit, AssetType: asset.Spot, Price: 80000, Amount: 1, CostAsset: currency.USDT},
			},
			{
				name:  "unified limit borrow on a unified order book",
				order: `{"id":"16","event":"put","finish_as":"open","currency_pair":"BTC_USDT","money":"USDC","type":"limit_borrow","account":"unified","side":"buy","amount":"0.5","price":"60000","left":"0.5",` + times + `}`,
				exp:   order.Detail{OrderID: "16", Side: order.Buy, Type: order.Limit, AssetType: asset.Spot, Price: 60000, Amount: 0.5},
			},
			{
				name:  "unified limit borrow filled on a unified order book",
				order: `{"id":"18","event":"finish","finish_as":"filled","currency_pair":"BTC_USDT","money":"USDC","type":"limit_borrow","account":"unified","side":"buy","amount":"0.5","price":"60000","left":"0","filled_total":"30000","filled_amount":"0.5","avg_deal_price":"60000",` + times + `}`,
				exp:   order.Detail{OrderID: "18", Status: order.Filled, Side: order.Buy, Type: order.Limit, AssetType: asset.Spot, Price: 60000, Amount: 0.5, ExecutedAmount: 0.5, AverageExecutedPrice: 60000, Cost: 30000, CostAsset: currency.USDC},
			},
			{
				name:  "unified market borrow and repay sell finished by its time in force",
				order: `{"id":"17","event":"finish","finish_as":"ioc","currency_pair":"BTC_USDT","type":"market_borrow_repay","account":"unified","side":"sell","amount":"0.5","left":"0.25","filled_amount":"0.25",` + times + `}`,
				exp:   order.Detail{OrderID: "17", Status: order.Cancelled, Side: order.Sell, Type: order.Market, AssetType: asset.Spot, Amount: 0.5, ExecutedAmount: 0.25, CostAsset: currency.USDT},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				push := `{"time":1694655226,"channel":"spot.orders","event":"update","result":[` + tc.order + `]}`
				require.NoError(t, ex.processSpotOrders(t.Context(), []byte(push)), "processSpotOrders must not error")
				require.Len(t, ex.Websocket.DataHandler.C, 1, "processSpotOrders must relay the push")
				tc.exp.Exchange, tc.exp.Pair, tc.exp.Date, tc.exp.LastUpdated = ex.Name, pair, created, updated
				assert.Equal(t, []order.Detail{tc.exp}, (<-ex.Websocket.DataHandler.C).Data, "processSpotOrders should relay every field of the order")
			})
		}
	})

	t.Run("mixed events in one push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		push := `{"time":1694655226,"channel":"spot.orders","event":"update","result":[` +
			`{"id":"31","event":"put","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amount":"1","price":"60000","left":"1"},` +
			`{"id":"32","event":"finish","finish_as":"filled","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amount":"1","price":"60000","left":"0","filled_total":"60000","avg_deal_price":"60000","fee":"0.001","fee_currency":"BTC"},` +
			`{"id":"33","event":"update","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"60000","left":"0.5"}]}`
		require.NoError(t, ex.processSpotOrders(t.Context(), []byte(push)), "processSpotOrders must not error")
		require.Len(t, ex.Websocket.DataHandler.C, 1, "processSpotOrders must relay the push")
		details, ok := (<-ex.Websocket.DataHandler.C).Data.([]order.Detail)
		require.True(t, ok, "processSpotOrders must relay order details")
		require.Len(t, details, 3, "processSpotOrders must relay every order of the push")
		assert.Equal(t, "31", details[0].OrderID, "processSpotOrders should relay the put order")
		assert.Equal(t, order.UnknownStatus, details[0].Status, "a put order should carry no status")
		assert.Equal(t, "32", details[1].OrderID, "processSpotOrders should relay the finished order after the put")
		assert.Equal(t, order.Filled, details[1].Status, "the finished order should carry its status")
		assert.Equal(t, 0.001, details[1].Fee, "the finished order should carry its fee")
		assert.Equal(t, "33", details[2].OrderID, "processSpotOrders should relay the update after the finished order")
		assert.Equal(t, 0.5, details[2].ExecutedAmount, "the updated order should carry what it executed")
		assert.Zero(t, details[2].Fee, "the updated order should carry no fee")
	})

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	push := func(account string) []byte {
		return []byte(`{"time":1605175506,"channel":"spot.orders","event":"update","result":[{"id":"30784435","user":123456,"text":"t-abc","create_time_ms":"1605175506123","update_time_ms":"1605175507123","event":"put","currency_pair":"BTC_USDT","type":"limit","account":"` + account + `","side":"sell","amount":"1","price":"10001","left":"0.25","fee":"0.5"}]}`)
	}
	ctx := t.Context()
	// Cases share ex's relay and the refusals below call processSpotOrders after them, so run in turn
	for account, a := range map[string]asset.Item{
		"":             asset.Spot,
		"spot":         asset.Spot,
		"unified":      asset.Spot,
		"margin":       asset.Margin,
		"cross_margin": asset.CrossMargin,
	} {
		t.Run(cmp.Or(account, "no account"), func(t *testing.T) {
			require.NoErrorf(t, ex.processSpotOrders(ctx, push(account)), "processSpotOrders must not error for account %q", account)
			require.Lenf(t, ex.Websocket.DataHandler.C, 1, "processSpotOrders must relay the order for account %q", account)
			exp := []order.Detail{{
				Amount:         1,
				Exchange:       ex.Name,
				OrderID:        "30784435",
				ClientOrderID:  "t-abc",
				Side:           order.Sell,
				Type:           order.Limit,
				Pair:           currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
				AssetType:      a,
				Price:          10001,
				ExecutedAmount: 0.75,
				Date:           time.UnixMilli(1605175506123),
				LastUpdated:    time.UnixMilli(1605175507123),
			}}
			assert.Equalf(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processSpotOrders should relay the order as %s for account %q", a, account)
		})
	}
	assert.ErrorIs(t, ex.processSpotOrders(t.Context(), push("bogus")), asset.ErrNotSupported, "processSpotOrders should reject an unknown account")
	assert.Error(t, ex.processSpotOrders(t.Context(), []byte(`{`)), "processSpotOrders should reject malformed JSON")
	assert.ErrorIs(t, ex.processSpotOrders(t.Context(), []byte(`{"time":1605175506,"channel":"spot.orders","event":"update","result":[{"id":"1","side":"bogus","type":"limit","account":"spot"}]}`)), order.ErrSideIsInvalid, "processSpotOrders should reject an unknown side")
	assert.ErrorIs(t, ex.processSpotOrders(t.Context(), []byte(`{"time":1605175506,"channel":"spot.orders","event":"update","result":[{"id":"1","side":"sell","type":"iceberg","account":"spot"}]}`)), order.ErrUnrecognisedOrderType, "processSpotOrders should reject an unknown order type")
	assert.ErrorContains(t, ex.processSpotOrders(t.Context(), []byte(`{"time":1605175506,"channel":"spot.orders","event":"finish","result":[{"id":"1","event":"finish","side":"sell","type":"limit","account":"spot","finish_as":"bogus"}]}`)), "unrecognised order status", "processSpotOrders should reject an unknown finish reason")
	require.Empty(t, ex.Websocket.DataHandler.C, "processSpotOrders must not relay a rejected push")
}

// TestProcessSpotOrdersMerge relays successive pushes into a stored order, merging each as the order manager does
func TestProcessSpotOrdersMerge(t *testing.T) {
	t.Parallel()
	const times = `"create_time_ms":"1694655225315","update_time_ms":"1694655226315"`
	relay := func(ctx context.Context, t *testing.T, ex *Exchange, o string) *order.Detail {
		t.Helper()
		require.NoError(t, ex.processSpotOrders(ctx, []byte(`{"time":1694655226,"channel":"spot.orders","event":"update","result":[`+o+`]}`)), "processSpotOrders must not error")
		require.Len(t, ex.Websocket.DataHandler.C, 1, "processSpotOrders must relay the push")
		details, ok := (<-ex.Websocket.DataHandler.C).Data.([]order.Detail)
		require.True(t, ok, "processSpotOrders must relay order details")
		require.Len(t, details, 1, "processSpotOrders must relay one order")
		return &details[0]
	}
	newExchange := func(t *testing.T) *Exchange {
		t.Helper()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		return ex
	}

	t.Run("a delayed earlier push does not reopen a finished order", func(t *testing.T) {
		t.Parallel()
		ex := newExchange(t)
		stored := relay(t.Context(), t, ex, `{"id":"21","event":"finish","finish_as":"cancelled","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"0.5","filled_amount":"0.5","fee":"0.001","fee_currency":"USDT",`+times+`}`)
		late := relay(t.Context(), t, ex, `{"id":"21","event":"update","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"0.5","filled_amount":"0.5","fee":"0.001","fee_currency":"USDT",`+times+`}`)
		require.NoError(t, stored.UpdateOrderFromDetail(late), "UpdateOrderFromDetail must not error")
		assert.Equal(t, order.Cancelled, stored.Status, "a delayed open push should not reopen a cancelled order")
	})

	t.Run("a filled order keeps no remainder from its partial fills", func(t *testing.T) {
		t.Parallel()
		ex := newExchange(t)
		ctx := t.Context()
		stored := relay(ctx, t, ex, `{"id":"22","event":"put","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"1",`+times+`}`)
		// Cases merge into stored in turn
		for _, tc := range []struct{ name, order string }{
			{name: "partial fill", order: `{"id":"22","event":"update","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"0.5","filled_amount":"0.5",` + times + `}`},
			{name: "fill", order: `{"id":"22","event":"finish","finish_as":"filled","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"0","filled_amount":"1",` + times + `}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				require.NoError(t, stored.UpdateOrderFromDetail(relay(ctx, t, ex, tc.order)), "UpdateOrderFromDetail must not error")
			})
		}
		assert.Equal(t, order.Filled, stored.Status, "the finish push should leave the order filled")
		assert.Equal(t, 1.0, stored.ExecutedAmount, "the order should be fully executed")
		assert.Zero(t, stored.RemainingAmount, "a filled order should keep no remainder from a partial fill")
	})

	t.Run("a finish without a fee does not relabel a stored fee", func(t *testing.T) {
		t.Parallel()
		ex := newExchange(t)
		stored := &order.Detail{Exchange: ex.Name, OrderID: "24", Fee: 0.001, FeeAsset: currency.BTC}
		finished := relay(t.Context(), t, ex, `{"id":"24","event":"finish","finish_as":"filled","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amount":"1","price":"60000","left":"0","filled_total":"60000","filled_amount":"1","avg_deal_price":"60000","fee":"0","fee_currency":"GT",`+times+`}`)
		require.NoError(t, stored.UpdateOrderFromDetail(finished), "UpdateOrderFromDetail must not error")
		assert.Equal(t, 0.001, stored.Fee, "a finish without a fee should keep the stored fee")
		assert.Equal(t, currency.BTC, stored.FeeAsset, "a finish without a fee should keep the stored fee's currency")
	})

	t.Run("a delayed earlier push does not put back an older fee, cost or average price", func(t *testing.T) {
		t.Parallel()
		ex := newExchange(t)
		stored := relay(t.Context(), t, ex, `{"id":"23","event":"finish","finish_as":"filled","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amount":"1","price":"65000","left":"0","filled_total":"65000","filled_amount":"1","avg_deal_price":"65000","fee":"0.001","fee_currency":"BTC",`+times+`}`)
		late := relay(t.Context(), t, ex, `{"id":"23","event":"update","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amount":"1","price":"65000","left":"0.5","filled_total":"30000","filled_amount":"0.5","avg_deal_price":"60000","fee":"0.0005","fee_currency":"BTC",`+times+`}`)
		require.NoError(t, stored.UpdateOrderFromDetail(late), "UpdateOrderFromDetail must not error")
		assert.Equal(t, 0.001, stored.Fee, "a delayed push should not put back an older fee")
		assert.Equal(t, 65000.0, stored.Cost, "a delayed push should not put back an older cost")
		assert.Equal(t, 65000.0, stored.AverageExecutedPrice, "a delayed push should not put back an older average price")
	})
}

func TestProcessSpotBalancesUpdatesRESTSnapshot(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	ex.Name = t.Name()
	ex.Accounts = accounts.MustNewAccounts(ex)
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
	restSnapshot := accounts.NewSubAccount(asset.Spot, "")
	restSnapshot.Balances.Set(currency.USDT, accounts.Balance{Total: 100, Free: 100})
	restSnapshot.Balances.Set(currency.BTC, accounts.Balance{Total: 2, Free: 1, Hold: 1})
	require.NoError(t, ex.Accounts.Save(ctx, accounts.SubAccounts{restSnapshot}, true), "Accounts.Save must seed the REST balance snapshot")
	assert.Error(t, ex.processSpotBalances(ctx, []byte(`{`)), "processSpotBalances should reject malformed JSON")

	require.NoError(t, ex.processSpotBalances(ctx, []byte(`[{"timestamp":"1755718222","timestamp_ms":"1755718222394","user":"12870774","currency":"USDT","change":"-10","total":"100","available":"90","freeze":"10"}]`)), "processSpotBalances must not error")
	creds, err := ex.GetCredentials(ctx)
	require.NoError(t, err, "GetCredentials must not error")
	balances, err := ex.Accounts.CurrencyBalances(creds, asset.Spot)
	require.NoError(t, err, "CurrencyBalances must not error")
	assert.Equal(t, 100.0, balances[currency.USDT].Total, "processSpotBalances should replace the REST balance rather than add to it")
	assert.Equal(t, 90.0, balances[currency.USDT].Free, "processSpotBalances should update the free balance")
	assert.Equal(t, 10.0, balances[currency.USDT].Hold, "processSpotBalances should update the held balance")
	assert.Equal(t, 2.0, balances[currency.BTC].Total, "processSpotBalances should keep balances the push does not carry")
}

// newBalancesTestExchange returns an exchange that checks credentials and keeps its own accounts, named after the test
func newBalancesTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	ex.SetDefaults()
	ex.Name = t.Name()
	ex.Accounts = accounts.MustNewAccounts(ex)
	return ex
}

func TestProcessMarginBalances(t *testing.T) {
	t.Parallel()
	const push = `{"time":1605248616,"channel":"spot.margin_balances","event":"update","result":[{"timestamp":"1605248616","timestamp_ms":"1605248616123","user":"1000001","currency_pair":"BTC_USDT","currency":"BTC","change":"-0.5","available":"2.5","freeze":"0.5","borrowed":"0","interest":"0"}]}`

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := newBalancesTestExchange(t)
		ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
		assert.Error(t, ex.processMarginBalances(ctx, []byte(`{`)), "processMarginBalances should reject malformed JSON")
	})

	t.Run("without credentials", func(t *testing.T) {
		t.Parallel()
		ex := newBalancesTestExchange(t)
		assert.ErrorIs(t, ex.processMarginBalances(t.Context(), []byte(push)), exchange.ErrCredentialsAreEmpty, "processMarginBalances should return a failure to save the balances")
		assert.Empty(t, ex.Websocket.DataHandler.C, "processMarginBalances should not relay balances it could not save")
	})

	t.Run("saves and relays the balances", func(t *testing.T) {
		t.Parallel()
		ex := newBalancesTestExchange(t)
		ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
		require.NoError(t, ex.processMarginBalances(ctx, []byte(push)), "processMarginBalances must not error")
		require.Len(t, ex.Websocket.DataHandler.C, 1, "processMarginBalances must relay the balances")
		exp := accounts.SubAccounts{{
			ID:        "1000001",
			AssetType: asset.Margin,
			Balances: accounts.CurrencyBalances{
				currency.BTC: {Currency: currency.BTC, Total: 3, Free: 2.5, Hold: 0.5, UpdatedAt: time.UnixMilli(1605248616123)},
			},
		}}
		assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processMarginBalances should relay the balances")
		creds, err := ex.GetCredentials(ctx)
		require.NoError(t, err, "GetCredentials must not error")
		stored, err := ex.Accounts.GetBalance("1000001", creds, asset.Margin, currency.BTC)
		require.NoError(t, err, "GetBalance must find the saved balance")
		assert.Equal(t, exp[0].Balances[currency.BTC], stored, "processMarginBalances should save the balance")
	})
}

func TestProcessCrossMarginBalance(t *testing.T) {
	t.Parallel()
	const push = `{"time":1605248616,"channel":"spot.cross_balances","event":"update","result":[{"timestamp":"1605248616","timestamp_ms":"1605248616123","user":"1000001","currency":"USDT","change":"100","total":"1032951.325075926","available":"1022943.325075926"}]}`

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := newBalancesTestExchange(t)
		ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
		assert.Error(t, ex.processCrossMarginBalance(ctx, []byte(`{`)), "processCrossMarginBalance should reject malformed JSON")
	})

	t.Run("without credentials", func(t *testing.T) {
		t.Parallel()
		ex := newBalancesTestExchange(t)
		assert.ErrorIs(t, ex.processCrossMarginBalance(t.Context(), []byte(push)), exchange.ErrCredentialsAreEmpty, "processCrossMarginBalance should return a failure to save the balances")
		assert.Empty(t, ex.Websocket.DataHandler.C, "processCrossMarginBalance should not relay balances it could not save")
	})

	t.Run("saves and relays the balances", func(t *testing.T) {
		t.Parallel()
		ex := newBalancesTestExchange(t)
		ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
		require.NoError(t, ex.processCrossMarginBalance(ctx, []byte(push)), "processCrossMarginBalance must not error")
		require.Len(t, ex.Websocket.DataHandler.C, 1, "processCrossMarginBalance must relay the balances")
		exp := accounts.SubAccounts{{
			ID:        "1000001",
			AssetType: asset.CrossMargin,
			Balances: accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 1032951.325075926, Free: 1022943.325075926, UpdatedAt: time.UnixMilli(1605248616123)},
			},
		}}
		assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processCrossMarginBalance should relay the balances")
		creds, err := ex.GetCredentials(ctx)
		require.NoError(t, err, "GetCredentials must not error")
		stored, err := ex.Accounts.GetBalance("1000001", creds, asset.CrossMargin, currency.USDT)
		require.NoError(t, err, "GetBalance must find the saved balance")
		assert.Equal(t, exp[0].Balances[currency.USDT], stored, "processCrossMarginBalance should save the balance")
	})
}

func TestProcessCrossMarginLoans(t *testing.T) {
	t.Parallel()

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		assert.Error(t, ex.processCrossMarginLoans(t.Context(), []byte(`{`)), "processCrossMarginLoans should reject malformed JSON")
		assert.Empty(t, ex.Websocket.DataHandler.C, "processCrossMarginLoans should not relay a malformed push")
	})

	t.Run("relays the loan", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.NoError(t, ex.processCrossMarginLoans(t.Context(), []byte(`{"time":1658289372,"channel":"spot.cross_loan","event":"update","result":{"timestamp":1658289372338,"user":"1000001","currency":"BTC","change":"0.01","total":"4.992341029566","available":"0.078054772536","borrowed":"0.01","interest":"0.00001375"}}`)), "processCrossMarginLoans must not error")
		require.Len(t, ex.Websocket.DataHandler.C, 1, "processCrossMarginLoans must relay the loan")
		exp := WebsocketPush[WebsocketCrossMarginLoan]{
			Time:    types.Time(time.Unix(1658289372, 0)),
			Channel: crossMarginLoanChannel,
			Event:   "update",
			Result: WebsocketCrossMarginLoan{
				Timestamp: types.Time(time.UnixMilli(1658289372338)),
				User:      "1000001",
				Currency:  currency.BTC,
				Change:    0.01,
				Total:     4.992341029566,
				Available: 0.078054772536,
				Borrowed:  0.01,
				Interest:  0.00001375,
			},
		}
		assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processCrossMarginLoans should relay the decoded push")
	})
}

func TestProcessFundingBalances(t *testing.T) {
	t.Parallel()

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		malformed := []byte(`{`)
		assert.Equal(t, json.Unmarshal(malformed, new(WebsocketPush[[]*WebsocketFundingBalance])), ex.processFundingBalances(t.Context(), malformed), "processFundingBalances should return the decode error of malformed JSON")
	})
}

func TestProcessUserPersonalTrades(t *testing.T) {
	t.Parallel()

	t.Run("ignored and rejected pushes", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		malformed := []byte(`{`)
		ex.SetFillsFeedStatus(false)
		require.NoError(t, ex.processUserPersonalTrades(malformed), "processUserPersonalTrades must ignore pushes while the fills feed is off")
		ex.SetFillsFeedStatus(true)
		assert.Equal(t, json.Unmarshal(malformed, new(WebsocketPush[[]*WebsocketUserPersonalTrade])), ex.processUserPersonalTrades(malformed), "processUserPersonalTrades should return the decode error of malformed JSON")
		assert.ErrorIs(t, ex.processUserPersonalTrades([]byte(`{"channel":"spot.usertrades","event":"update","result":[{"id":1,"currency_pair":"BTC_USDT","side":"bogus"}]}`)), order.ErrSideIsInvalid, "processUserPersonalTrades should reject a fill with an unknown side")
	})
}

func TestProcessTicker(t *testing.T) {
	t.Parallel()

	t.Run("untracked pair relays nothing", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		require.NoError(t, ex.processTicker(t.Context(), []byte(`{"currency_pair":"UNTRACKED_USDT","last":"1"}`), time.Now()), "processTicker must not error for an untracked pair")
		assert.Empty(t, ex.Websocket.DataHandler.C, "processTicker should not relay an empty batch")
	})
}

func TestProcessTrades(t *testing.T) {
	t.Parallel()
	const push = `{"id":309143071,"create_time":1606292218,"create_time_ms":"1606292218213","side":"sell","currency_pair":"BTC_USDT","amount":"16.47","price":"0.4705"}`

	t.Run("ignored while the trade feed and saving are off", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.SetTradeFeedStatus(false)
		ex.SetSaveTradeDataStatus(false)
		require.NoError(t, ex.processTrades([]byte(`{`)), "processTrades must ignore pushes while the trade feed and saving are off")
	})

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.SetTradeFeedStatus(true)
		assert.Error(t, ex.processTrades([]byte(`{`)), "processTrades should reject malformed JSON")
	})

	t.Run("unknown side", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.SetTradeFeedStatus(true)
		ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
		assert.ErrorIs(t, ex.processTrades([]byte(strings.Replace(push, `"side":"sell"`, `"side":"bogus"`, 1))), order.ErrSideIsInvalid, "processTrades should reject an unknown side")
		assert.Empty(t, ex.Websocket.DataHandler.C, "processTrades should not relay a rejected trade")
	})

	t.Run("relays the trade for each asset the pair is enabled on", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.SetTradeFeedStatus(true)
		ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
		require.NoError(t, ex.processTrades([]byte(push)), "processTrades must not error")
		pair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
		var exp []trade.Data
		for _, a := range standardMarginAssetTypes {
			if enabled, err := ex.CurrencyPairs.IsPairEnabled(pair, a); err == nil && enabled {
				exp = append(exp, trade.Data{TID: "309143071", Exchange: ex.Name, CurrencyPair: pair, AssetType: a, Side: order.Sell, Price: 0.4705, Amount: 16.47, Timestamp: time.UnixMilli(1606292218213)})
			}
		}
		require.NotEmpty(t, exp, "BTC_USDT must be enabled on a spot market asset")
		require.Len(t, ex.Websocket.DataHandler.C, len(exp), "processTrades must relay the trade once for each asset")
		relayed := make([]trade.Data, 0, len(exp))
		for range exp {
			trades, ok := (<-ex.Websocket.DataHandler.C).Data.([]trade.Data)
			require.True(t, ok, "processTrades must relay trades")
			relayed = append(relayed, trades...)
		}
		assert.Equal(t, exp, relayed, "processTrades should relay the trade for each asset BTC_USDT is enabled on")
	})
}

func TestProcessCandlestick(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	assert.Error(t, ex.processCandlestick(t.Context(), []byte(`{`)), "processCandlestick should reject malformed JSON")
	assert.ErrorIs(t, ex.processCandlestick(t.Context(), []byte(`{"t":"1606292580","n":"1m"}`)), common.ErrMalformedData, "processCandlestick should reject a name without a pair")
	assert.ErrorIs(t, ex.processCandlestick(t.Context(), []byte(`{"t":"1606292580","n":"9x_BTC_USDT"}`)), kline.ErrInvalidInterval, "processCandlestick should reject an unknown interval")
	assert.ErrorIs(t, ex.processCandlestick(t.Context(), []byte(`{"t":"1606292580","n":"1m__"}`)), currency.ErrCreatingPair, "processCandlestick should reject an invalid pair")
	require.Empty(t, ex.Websocket.DataHandler.C, "processCandlestick must not relay rejected candles")

	require.NoError(t, ex.processCandlestick(t.Context(), []byte(`{"t":"1606292580","v":"2362.32035","c":"19128.1","h":"19129.2","l":"19127.3","o":"19128.4","n":"1m_BTC_USDT"}`)), "processCandlestick must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "processCandlestick must relay the candle")
	items, ok := (<-ex.Websocket.DataHandler.C).Data.([]kline.Item)
	require.True(t, ok, "processCandlestick must relay kline items")
	var enabled []asset.Item
	for _, a := range standardMarginAssetTypes {
		if ok, err := ex.CurrencyPairs.IsPairEnabled(currency.NewBTCUSDT(), a); err == nil && ok {
			enabled = append(enabled, a)
		}
	}
	require.NotEmpty(t, enabled, "BTC_USDT must be enabled on a spot market asset")
	relayed := make([]asset.Item, len(items))
	for i := range items {
		relayed[i] = items[i].Asset
	}
	assert.ElementsMatch(t, enabled, relayed, "processCandlestick should relay the candle for each asset BTC_USDT is enabled on")
	for _, item := range items {
		assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), item.Pair, "processCandlestick should relay the pair")
		assert.Equal(t, kline.OneMin, item.Interval, "processCandlestick should relay the interval")
		assert.Equal(t, []kline.Candle{{Time: time.Unix(1606292580, 0), Open: 19128.4, Close: 19128.1, High: 19129.2, Low: 19127.3, Volume: 2362.32035}}, item.Candles, "processCandlestick should relay the candle")
	}
}

func TestProcessOrderbookTicker(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	assert.Error(t, ex.processOrderbookTicker(t.Context(), []byte(`{`), time.Now()), "processOrderbookTicker should reject malformed JSON")
	require.NoError(t, ex.processOrderbookTicker(t.Context(), []byte(`{"t":1606293275123,"u":48733182,"s":"BTC_USDT","b":"19177.79","B":"0.0003341504","a":"19179.38","A":"0.09"}`), time.Now()), "processOrderbookTicker must not error")
	ob, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "GetOrderbook must find the loaded book")
	assert.Equal(t, orderbook.Levels{{Price: 19177.79, Amount: 0.0003341504}}, ob.Bids, "processOrderbookTicker should load the best bid")
	assert.Equal(t, orderbook.Levels{{Price: 19179.38, Amount: 0.09}}, ob.Asks, "processOrderbookTicker should load the best ask")
}

func TestProcessOrderbookUpdate(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	assert.Error(t, ex.processOrderbookUpdate(t.Context(), []byte(`{`), time.Now()), "processOrderbookUpdate should reject malformed JSON")
	assert.ErrorIs(t, ex.processOrderbookUpdate(t.Context(), []byte(`{"t":1606294781123,"s":"","U":48776301,"u":48776306}`), time.Now()), currency.ErrCurrencyPairEmpty, "processOrderbookUpdate should reject an update without a pair")
}

func TestProcessOrderbookSnapshot(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	snapshot := []byte(`{"t":1606295412123,"lastUpdateId":48791820,"s":"BTC_USDT","bids":[["19079.55","0.0195"]],"asks":[["19080.24","0.1638"]]}`)
	assert.Error(t, ex.processOrderbookSnapshot(t.Context(), []byte(`{`), time.Now()), "processOrderbookSnapshot should reject malformed JSON")
	require.NoError(t, ex.processOrderbookSnapshot(t.Context(), snapshot, time.Now()), "processOrderbookSnapshot must not error")
	ob, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "GetOrderbook must find the loaded book")
	assert.Equal(t, orderbook.Levels{{Price: 19079.55, Amount: 0.0195}}, ob.Bids, "processOrderbookSnapshot should load the bids")
	assert.Equal(t, orderbook.Levels{{Price: 19080.24, Amount: 0.1638}}, ob.Asks, "processOrderbookSnapshot should load the asks")

	unnamed := new(Exchange)
	require.NoError(t, testexch.Setup(unnamed), "Setup must not error")
	unnamed.Name = ""
	assert.ErrorIs(t, unnamed.processOrderbookSnapshot(t.Context(), snapshot, time.Now()), common.ErrExchangeNameNotSet, "processOrderbookSnapshot should return a failure to load the book")
}

var mapSpotPublicPushData = map[string]string{
	"spot.tickers":           `{"time": 1606291803,	"channel": "spot.tickers",	"event": "update",	"result": {	  "currency_pair": "BTC_USDT",	  "last": "19106.55",	  "lowest_ask": "19108.71",	  "highest_bid": "19106.55",	  "change_percentage": "3.66",	  "base_volume": "2811.3042155865",	  "quote_volume": "53441606.52411221454674732293",	  "high_24h": "19417.74",	  "low_24h": "18434.21"	}}`,
	"spot.trades":            `{"time": 1606292218,	"channel": "spot.trades",	"event": "update",	"result": {	  "id": 309143071,	  "create_time": 1606292218,	  "create_time_ms": "1606292218213.4578",	  "side": "sell",	  "currency_pair": "BTC_USDT",	  "amount": "16.4700000000",	  "price": "0.4705000000"}}`,
	"spot.candlesticks":      `{"time": 1606292600,	"channel": "spot.candlesticks",	"event": "update",	"result": {	  "t": "1606292580",	  "v": "2362.32035",	  "c": "19128.1",	  "h": "19128.1",	  "l": "19128.1",	  "o": "19128.1","n": "1m_BTC_USDT"}}`,
	"spot.book_ticker":       `{"time": 1606293275,	"channel": "spot.book_ticker",	"event": "update",	"result": {	  "t": 1606293275123,	  "u": 48733182,	  "s": "BTC_USDT",	  "b": "19177.79",	  "B": "0.0003341504",	  "a": "19179.38",	  "A": "0.09"	}}`,
	"spot.order_book_update": `{"time": 1606294781,	"channel": "spot.order_book_update",	"event": "update",	"result": {	  "t": 1606294781123,	  "e": "depthUpdate",	  "E": 1606294781,"s": "BTC_USDT","U": 48776301,"u": 48776306,"b": [["19137.74","0.0001"],["19088.37","0"]],"a": [["19137.75","0.6135"]]	}}`,
	"spot.order_book":        `{"time":1606295412,"channel": "spot.order_book",	"event": "update",	"result": {	  "t": 1606295412123,	  "lastUpdateId": 48791820,	  "s": "BTC_USDT",	  "bids": [		[		  "19079.55",		  "0.0195"		],		[		  "19079.07",		  "0.7341"],["19076.23",		  "0.00011808"		],		[		  "19073.9",		  "0.105"		],		[		  "19068.83",		  "0.1009"		]	  ],	  "asks": [		[		  "19080.24",		  "0.1638"		],		[		  "19080.91","0.1366"],["19080.92","0.01"],["19081.29","0.01"],["19083.8","0.097"]]}}`,
	"spot.orders":            `{"time": 1605175506,	"channel": "spot.orders",	"event": "update",	"result": [	  {		"id": "30784435",		"user": 123456,		"text": "t-abc",		"create_time": "1605175506",		"create_time_ms": "1605175506123",		"update_time": "1605175506",		"update_time_ms": "1605175506123",		"event": "put",		"currency_pair": "BTC_USDT",		"type": "limit",		"account": "spot",		"side": "sell",		"amount": "1",		"price": "10001",		"time_in_force": "gtc",		"left": "1",		"filled_total": "0",		"fee": "0",		"fee_currency": "USDT",		"point_fee": "0",		"gt_fee": "0",		"gt_discount": true,		"rebated_fee": "0",		"rebated_fee_currency": "USDT"}	]}`,
	"spot.usertrades":        `{"time": 1605176741,	"channel": "spot.usertrades",	"event": "update",	"result": [	  {		"id": 5736713,		"user_id": 1000001,		"order_id": "30784428",		"currency_pair": "BTC_USDT",		"create_time": 1605176741,		"create_time_ms": "1605176741123.456",		"side": "sell",		"amount": "1.00000000",		"role": "taker",		"price": "10000.00000000",		"fee": "0.00200000000000",		"point_fee": "0",		"gt_fee": "0",		"text": "apiv4"	  }	]}`,
	"spot.cross_loan":        `{"time":1658289372,	"channel":"spot.cross_loan",	"event":"update",	"result":{	  "timestamp":1658289372338,	  "user":"1000001",	  "currency":"BTC",	  "change":"0.01",	  "total":"4.992341029566",	  "available":"0.078054772536",	  "borrowed":"0.01",	  "interest":"0.00001375"	}}`,
}

func TestWebsocketHandleSpotData(t *testing.T) {
	t.Parallel()

	t.Run("null result", func(t *testing.T) {
		t.Parallel()
		// Gate sends a null result alongside an error; these channels decode one object, which must not dereference nil
		for _, tc := range []struct {
			channel     string
			err         error
			errContains string
		}{
			{channel: spotCandlesticksChannel, err: common.ErrMalformedData},
			{channel: spotOrderbookTickerChannel, errContains: "currency pair not set"},
			{channel: spotOrderbookUpdateChannel, err: currency.ErrCurrencyPairEmpty},
			{channel: spotOrderbookChannel},
		} {
			t.Run(tc.channel, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				var err error
				require.NotPanics(t, func() {
					err = ex.WebsocketHandleSpotData(t.Context(), nil, []byte(`{"time":1,"channel":"`+tc.channel+`","event":"update","result":null}`))
				}, "WebsocketHandleSpotData must not panic on a null result")
				if tc.errContains != "" {
					assert.ErrorContains(t, err, tc.errContains, "WebsocketHandleSpotData should reject a null result")
					return
				}
				assert.ErrorIs(t, err, tc.err, "WebsocketHandleSpotData should return the expected error for a null result")
			})
		}
	})

	// Private pushes share e with the public ones below, so run first
	t.Run("private pushes", func(t *testing.T) {
		ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
		// Cases share e, so run in turn
		for x := range mapSpotPrivatePushData {
			t.Run(x, func(t *testing.T) {
				err := e.WebsocketHandleSpotData(ctx, nil, []byte(mapSpotPrivatePushData[x]))
				require.NoError(t, err)
			})
		}
	})

	for k, v := range mapSpotPublicPushData {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			err := e.WebsocketHandleSpotData(t.Context(), nil, []byte(v))
			require.NoError(t, err)
		})
	}
}

var mapSpotPrivatePushData = map[string]string{
	"spot.balances":         `{"time": 1605248616,	"channel": "spot.balances",	"event": "update",	"result": [	  {		"timestamp": "1605248616",		"timestamp_ms": "1605248616123",		"user": "1000001",		"currency": "USDT",		"change": "100",		"total": "1032951.325075926","available": "1022943.325075926"}	]}`,
	"spot.funding_balances": `{"time": 1605248616,	"channel": "spot.funding_balances",	"event": "update",	"result": [	  {"timestamp": "1605248616","timestamp_ms": "1605248616123","user": "1000001","currency": "USDT","change": "100","freeze": "100","lent": "0"}	]}`,
	"spot.cross_balances":   `{"time": 1605248616,"channel": "spot.cross_balances","event": "update",	"result": [{"timestamp": "1605248616","timestamp_ms": "1605248616123","user": "1000001","currency": "USDT",	"change": "100","total": "1032951.325075926","available": "1022943.325075926"}]}`,
}

func TestGenerateSubscriptionsSpot(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := e.generateSubscriptionsSpot(t.Context())
	require.NoError(t, err, "generateSubscriptions must not error")
	var spotV2Count int
	for _, sub := range subs {
		if sub.Asset == asset.Spot && sub.Channel == subscription.OrderbookChannel {
			assert.Fail(t, "legacy spot orderbook subscription should be disabled")
		}
		if sub.Asset == asset.Spot && sub.Channel == spotOrderbookV2 {
			spotV2Count++
			assert.Equal(t, 50, sub.Levels, "V2 spot orderbook subscription should request 50 levels")
		}
	}
	spotPairs, err := e.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	assert.Equal(t, len(spotPairs), spotV2Count, "each enabled spot pair should generate one V2 orderbook subscription")
	exp := subscription.List{}
	assets := slices.DeleteFunc(e.GetAssetTypes(true), func(a asset.Item) bool { return !e.IsAssetWebsocketSupported(a) })
	for _, s := range e.Features.Subscriptions {
		for _, a := range assets {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			pairs = common.SortStrings(pairs).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"})
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.Asset = a
			if singleSymbolChannel(channelName(s)) {
				for i := range pairs {
					s := s.Clone() //nolint:govet // Intentional lexical scope shadow
					switch s.Channel {
					case subscription.CandlesChannel:
						s.QualifiedChannel = "5m," + pairs[i].String()
					case subscription.OrderbookChannel:
						s.QualifiedChannel = pairs[i].String() + ",100ms"
					case spotOrderbookChannel:
						s.QualifiedChannel = pairs[i].String() + ",5,1000ms"
					case spotOrderbookV2:
						s.QualifiedChannel = fmt.Sprintf("ob.%s.%d", pairs[i].String(), s.Levels)
					}
					s.Pairs = pairs[i : i+1]
					exp = append(exp, s)
				}
			} else {
				s.Pairs = pairs
				s.QualifiedChannel = pairs.Join()
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	subs, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	e.Features.Subscriptions = subscription.List{}
	err = e.Subscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "Subscribe must not error")
}

func TestHandleSubscription(t *testing.T) {
	t.Parallel()

	t.Run("payload generation failure", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		err := ex.handleSubscription(t.Context(), &FixtureConnection{}, subscribeEvent, subscription.List{{Channel: subscription.OrderbookChannel}}, func(context.Context, string, subscription.List) ([]*WebsocketChannelRequest, error) {
			return nil, errNoChannelsSupplied
		})
		assert.ErrorIs(t, err, errNoChannelsSupplied, "handleSubscription should return a failure to generate the payloads")
	})

	t.Run("malformed reply", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		err := ex.handleSubscription(t.Context(), &replyConnection{reply: []byte(`{`)}, subscribeEvent, subscription.List{{Channel: subscription.OrderbookChannel}}, func(context.Context, string, subscription.List) ([]*WebsocketChannelRequest, error) {
			return []*WebsocketChannelRequest{{}}, nil
		})
		assert.Error(t, err, "handleSubscription should reject a malformed reply")
	})

	t.Run("refused subscription", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		reply := []byte(`{"time":1726121320,"id":1,"channel":"spot.order_book_update","event":"subscribe","error":{"code":2,"message":"invalid argument"},"result":{"status":"fail"}}`)
		err := ex.handleSubscription(t.Context(), &replyConnection{reply: reply}, subscribeEvent, subscription.List{{Channel: subscription.OrderbookChannel}}, func(context.Context, string, subscription.List) ([]*WebsocketChannelRequest, error) {
			return []*WebsocketChannelRequest{{Event: subscribeEvent, Channel: spotOrderbookUpdateChannel}}, nil
		})
		assert.ErrorContains(t, err, "error while subscribe to channel spot.order_book_update error code: 2 message: invalid argument", "handleSubscription should return Gate's refusal")
	})

	subs := subscription.List{{Channel: subscription.OrderbookChannel}}
	err := e.handleSubscription(t.Context(), &FixtureConnection{}, subscribeEvent, subs, func(context.Context, string, subscription.List) ([]*WebsocketChannelRequest, error) {
		return []*WebsocketChannelRequest{{}}, nil
	})
	require.NoError(t, err)

	err = e.handleSubscription(t.Context(), &FixtureConnection{}, unsubscribeEvent, subs, func(context.Context, string, subscription.List) ([]*WebsocketChannelRequest, error) {
		return []*WebsocketChannelRequest{{}}, nil
	})
	require.NoError(t, err)

	err = e.handleSubscription(t.Context(), &replyConnection{reply: []byte(`null`)}, subscribeEvent, subs, func(context.Context, string, subscription.List) ([]*WebsocketChannelRequest, error) {
		return []*WebsocketChannelRequest{{}}, nil
	})
	assert.ErrorIs(t, err, common.ErrNoResponse, "handleSubscription should reject a null reply")
}

func TestManageSubs(t *testing.T) {
	t.Parallel()

	t.Run("null reply", func(t *testing.T) {
		t.Parallel()
		subs := subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{getPair(t, asset.Spot)}}}
		err := e.manageSubs(t.Context(), subscribeEvent, &replyConnection{reply: []byte(`null`)}, subs)
		assert.ErrorIs(t, err, common.ErrNoResponse, "manageSubs should reject a null reply")
	})
}

func TestManageSubReq(t *testing.T) {
	t.Parallel()

	t.Run("public channel", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		before := unixSeconds(time.Now())
		req, err := ex.manageSubReq(t.Context(), subscribeEvent, &subscription.Subscription{Channel: subscription.CandlesChannel, QualifiedChannel: "5m,BTC_USDT"})
		require.NoError(t, err, "manageSubReq must not error")
		require.NotNil(t, req, "manageSubReq must return a request")
		assert.GreaterOrEqual(t, req.Time, before, "manageSubReq should stamp the request with the current time")
		assert.LessOrEqual(t, req.Time, unixSeconds(time.Now()), "manageSubReq should stamp the request with the current time")
		exp := &WebsocketChannelRequest{ID: 1, Event: subscribeEvent, Channel: spotCandlesticksChannel, Payload: []string{"5m", "BTC_USDT"}, Time: req.Time}
		assert.Equal(t, exp, req, "manageSubReq should send Gate's channel name with the qualified channel's parts and no signature")
	})

	t.Run("authenticated channel", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
		req, err := ex.manageSubReq(t.Context(), unsubscribeEvent, &subscription.Subscription{Channel: spotOrdersChannel, QualifiedChannel: "BTC_USDT,ETH_USDT", Authenticated: true})
		require.NoError(t, err, "manageSubReq must not error")
		require.NotNil(t, req, "manageSubReq must return a request")
		sig, err := ex.generateWebsocketSignature("mock-api-secret", unsubscribeEvent, spotOrdersChannel, req.Time)
		require.NoError(t, err, "generateWebsocketSignature must not error")
		exp := &WebsocketChannelRequest{
			ID:      1,
			Event:   unsubscribeEvent,
			Channel: spotOrdersChannel,
			Payload: []string{"BTC_USDT", "ETH_USDT"},
			Auth:    &WebsocketAuthInput{Method: "api_key", Key: "mock-api-key", Sign: sig},
			Time:    req.Time,
		}
		assert.Equal(t, exp, req, "manageSubReq should sign the request with the API key")
	})

	t.Run("authenticated channel without credentials", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		req, err := ex.manageSubReq(t.Context(), subscribeEvent, &subscription.Subscription{Channel: spotOrdersChannel, QualifiedChannel: "BTC_USDT", Authenticated: true})
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "manageSubReq should need credentials to sign an authenticated channel")
		assert.Nil(t, req, "manageSubReq should return no request without credentials")
	})
}

func TestSendWebsocketRequest(t *testing.T) {
	t.Parallel()

	t.Run("parameters that do not marshal", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		params := make(chan int)
		_, marshalErr := json.Marshal(params)
		require.Error(t, marshalErr, "a channel must not marshal")
		err := ex.SendWebsocketRequest(t.Context(), spotGetOrdersEPL, "spot.order_status", asset.Spot, params, new(*WebsocketOrderResponse), 1)
		assert.EqualError(t, err, marshalErr.Error(), "SendWebsocketRequest should return the error marshalling its parameters")
	})

	t.Run("not connected", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		err := ex.SendWebsocketRequest(t.Context(), spotGetOrdersEPL, "spot.order_status", asset.Spot, &WebsocketOrderRequest{OrderID: "1700664330", Pair: "BTC_USDT"}, new(*WebsocketOrderResponse), 1)
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "SendWebsocketRequest should need a connection")
	})

	t.Run("result from the WebSocket API", func(t *testing.T) {
		t.Parallel()
		ex := newWsAPITestExchange(t, asset.Spot)
		startTime, endTime := getTime()
		params := &websocketSpotListOrdersParams{Pair: "BTC_USDT", Status: statusFinished, Page: 1, Limit: 3, Account: "spot", From: unixSeconds(startTime), To: unixSeconds(endTime), Side: order.Sell.Lower()}
		var got []*WebsocketOrderResponse
		require.NoError(t, ex.SendWebsocketRequest(t.Context(), spotGetOrdersEPL, "spot.order_list", asset.Spot, params, &got, 1), "SendWebsocketRequest must not error")
		if mockTests {
			exp := []*WebsocketOrderResponse{wsAPISpotOrder(func(o *WebsocketOrderResponse) {
				o.ID, o.Text, o.Status, o.FinishAs, o.Side, o.Left, o.FilledTotal = "20874890569", "web", "closed", "filled", "sell", 0, 1
			})}
			assert.Equal(t, exp, got, "SendWebsocketRequest should decode the result")
			return
		}
		assert.NotNil(t, got, "SendWebsocketRequest should decode the result")
	})

	t.Run("result after an acknowledgement", func(t *testing.T) {
		t.Parallel()
		ex := newWsAPITestExchange(t, asset.Spot, canManipulateRealOrders)
		var got *WebsocketOrderResponse
		params := &CreateOrderRequest{Text: "t-my-custom-id", CurrencyPair: BTCUSDT, Type: "limit", Account: asset.Spot, Side: "buy", Amount: 1, Price: 1}
		require.NoError(t, ex.SendWebsocketRequest(t.Context(), spotPlaceOrderEPL, "spot.order_place", asset.Spot, params, &got, 2), "SendWebsocketRequest must not error")
		if mockTests {
			assert.Equal(t, wsAPISpotOrder(nil), got, "SendWebsocketRequest should decode the result that follows the acknowledgement")
			return
		}
		assert.NotEmpty(t, got, "SendWebsocketRequest should decode the result that follows the acknowledgement")
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the refused order is only recorded for mock tests")
		}
		ex := newWsAPITestExchange(t, asset.Spot)
		params := &CreateOrderRequest{Text: "t-rejected", CurrencyPair: BTCUSDT, Type: "limit", Account: asset.Spot, Side: "buy", Amount: 1, Price: 1}
		err := ex.SendWebsocketRequest(t.Context(), spotPlaceOrderEPL, "spot.order_place", asset.Spot, params, new(*WebsocketOrderResponse), 2)
		assert.ErrorContains(t, err, "BALANCE_NOT_ENOUGH", "SendWebsocketRequest should return Gate's error for a failed request")
	})
}

func TestParseWebsocketHeader(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input string
	}{
		{name: "time and time_ms", input: `{"time":1726121320,"time_ms":1726121320745,"id":1,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`},
		{name: "time_ms only", input: `{"time_ms":1726121320746,"id":2,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`},
		{name: "time only", input: `{"time":1726121321,"id":3,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h, err := parseWSHeader([]byte(tc.input))
			require.NoError(t, err)
			require.NotEmpty(t, h.ID)
			assert.Equal(t, "a4", h.RequestID)
			assert.Equal(t, "spot.tickers", h.Channel)
			assert.Equal(t, "subscribe", h.Event)
			assert.NotEmpty(t, h.Result)
			switch h.ID {
			case 1:
				assert.Equal(t, int64(1726121320745), h.Time.UnixMilli())
			case 2:
				assert.Equal(t, int64(1726121320746), h.Time.UnixMilli())
			case 3:
				assert.Equal(t, int64(1726121321), h.Time.Unix())
			}
		})
	}
}

func TestIsSingleOrderbookChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel  string
		expected bool
	}{
		{channel: spotOrderbookUpdateChannel, expected: true},
		{channel: spotOrderbookChannel, expected: true},
		{channel: spotOrderbookTickerChannel, expected: true},
		{channel: futuresOrderbookChannel, expected: true},
		{channel: futuresOrderbookTickerChannel, expected: true},
		{channel: futuresOrderbookUpdateChannel, expected: true},
		{channel: optionsOrderbookChannel, expected: true},
		{channel: optionsOrderbookTickerChannel, expected: true},
		{channel: optionsOrderbookUpdateChannel, expected: true},
		{channel: spotTickerChannel, expected: false},
		{channel: "sad", expected: false},
	} {
		t.Run(tc.channel, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isSingleOrderbookChannel(tc.channel))
		})
	}
}

func TestValidateSubscriptions(t *testing.T) {
	t.Parallel()
	require.NoError(t, e.ValidateSubscriptions(nil))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSDT()}}}))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
	}))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD(), currency.NewBTCUSDT()}},
	}))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSDT()}},
	}))
	require.ErrorIs(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
	}), subscription.ErrExclusiveSubscription)
}

func TestCandlesChannelIntervals(t *testing.T) {
	t.Parallel()
	s := &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: 0}
	_, err := candlesChannelInterval(s)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "candlestickChannelInterval must error correctly with a 0 interval")
	s.Interval = kline.ThousandMilliseconds
	i, err := candlesChannelInterval(s)
	require.NoError(t, err)
	assert.Equal(t, "1000ms", i)
}

func TestOrderbookChannelIntervals(t *testing.T) {
	t.Parallel()

	s := &subscription.Subscription{Channel: futuresOrderbookUpdateChannel, Interval: kline.TwentyMilliseconds, Levels: 100}
	_, err := orderbookChannelInterval(s, asset.Futures)
	require.ErrorIs(t, err, subscription.ErrInvalidInterval)
	require.ErrorContains(t, err, "20ms only valid with Levels 20")
	s.Levels = 20
	i, err := orderbookChannelInterval(s, asset.Futures)
	require.NoError(t, err)
	assert.Equal(t, "20ms", i)

	for s, exp := range map[*subscription.Subscription]error{
		{Asset: asset.Binary, Channel: "unknown_channel", Interval: kline.OneYear}:                    nil,
		{Asset: asset.Spot, Channel: spotOrderbookTickerChannel, Interval: kline.OneDay}:              subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookTickerChannel, Interval: 0}:                         nil,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.OneDay}:                    subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.HundredMilliseconds}:       nil,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.ThousandMilliseconds}:      nil,
		{Asset: asset.Spot, Channel: spotOrderbookUpdateChannel, Interval: kline.OneDay}:              subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}: nil,

		{Asset: asset.Futures, Channel: futuresOrderbookTickerChannel, Interval: kline.TenMilliseconds}:     subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookTickerChannel, Interval: 0}:                         nil,
		{Asset: asset.Futures, Channel: futuresOrderbookChannel, Interval: kline.TenMilliseconds}:           subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookChannel, Interval: 0}:                               nil,
		{Asset: asset.Futures, Channel: futuresOrderbookUpdateChannel, Interval: kline.OneDay}:              subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}: nil,

		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookTickerChannel, Interval: kline.TenMilliseconds}:      subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookTickerChannel, Interval: 0}:                          nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookChannel, Interval: kline.TenMilliseconds}:            subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookChannel, Interval: 0}:                                nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.OneDay}:               subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}:  nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.ThousandMilliseconds}: nil,

		{Asset: asset.Options, Channel: optionsOrderbookTickerChannel, Interval: kline.TenMilliseconds}:          subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookTickerChannel, Interval: 0}:                              nil,
		{Asset: asset.Options, Channel: optionsOrderbookChannel, Interval: kline.TwoHundredAndFiftyMilliseconds}: subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookChannel, Interval: 0}:                                    nil,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.OneDay}:                   subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}:      nil,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.ThousandMilliseconds}:     nil,
	} {
		t.Run(s.Asset.String()+"/"+s.Channel+"/"+s.Interval.Short(), func(t *testing.T) {
			t.Parallel()
			i, err := orderbookChannelInterval(s, s.Asset)
			if exp != nil {
				require.ErrorIs(t, err, exp)
			} else {
				switch {
				case s.Channel == "unknown_channel":
					assert.Empty(t, i, "orderbookChannelInterval should return empty for unknown channels")
				case strings.HasSuffix(s.Channel, "_ticker"):
					assert.Empty(t, i)
				case s.Interval == 0:
					assert.Equal(t, "0", i)
				default:
					exp, err2 := getIntervalString(s.Interval)
					require.NoError(t, err2, "getIntervalString must not error for validating expected value")
					require.Equal(t, exp, i)
				}
			}
		})
	}
}

func TestChannelLevels(t *testing.T) {
	t.Parallel()

	for s, exp := range map[*subscription.Subscription]error{
		{Channel: "unknown_channel", Asset: asset.Binary}:                                   nil,
		{Channel: spotOrderbookTickerChannel, Asset: asset.Spot}:                            nil,
		{Channel: spotOrderbookTickerChannel, Asset: asset.Spot, Levels: 1}:                 subscription.ErrInvalidLevel,
		{Channel: spotOrderbookUpdateChannel, Asset: asset.Spot}:                            nil,
		{Channel: spotOrderbookUpdateChannel, Asset: asset.Spot, Levels: 100}:               subscription.ErrInvalidLevel,
		{Channel: spotOrderbookChannel, Asset: asset.Spot}:                                  subscription.ErrInvalidLevel,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 5}:                       nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 10}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 20}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 50}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 100}:                     nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures}:                            subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 1}:                 nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 5}:                 nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 10}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 20}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 50}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 100}:               nil,
		{Channel: futuresOrderbookTickerChannel, Asset: asset.Futures}:                      nil,
		{Channel: futuresOrderbookTickerChannel, Asset: asset.Futures, Levels: 1}:           subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures}:                      subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures, Levels: 20}:          nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures, Levels: 50}:          nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures}:              subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 5}:   nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 10}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 20}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 50}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 100}: nil,
		{Channel: optionsOrderbookTickerChannel, Asset: asset.Options}:                      nil,
		{Channel: optionsOrderbookTickerChannel, Asset: asset.Options, Levels: 1}:           subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options}:                      subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 5}:           nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 10}:          nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 20}:          nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 50}:          nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options}:                            subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 5}:                 nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 10}:                nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 20}:                nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 50}:                nil,
	} {
		t.Run(s.Asset.String()+"/"+s.Channel+"/"+strconv.Itoa(s.Levels), func(t *testing.T) {
			t.Parallel()
			l, err := channelLevels(s, s.Asset)
			switch {
			case exp != nil:
				require.ErrorIs(t, err, exp)
			case s.Levels == 0:
				assert.Empty(t, l)
			default:
				require.NoError(t, err)
				require.NotEmpty(t, l)
			}
		})
	}
}

// newLiveWebsocketConnection connects a live instance's websocket with only asset a enabled, signing in when the test
// credentials are set, and returns it with that asset's connection
func newLiveWebsocketConnection(t *testing.T, a asset.Item) (*Exchange, websocket.Connection) {
	t.Helper()
	ex := newTestExchange(t)
	testexch.UpdatePairsOnce(t, ex)
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		ex.API.AuthenticatedSupport = true
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(apiCredentials)
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	for _, enabled := range ex.GetAssetTypes(true) {
		if enabled != a {
			require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(enabled, false), "SetAssetEnabled must not error for %s", enabled)
		}
	}
	// Registered before connecting, as the connection monitor would otherwise keep redialling after a failed test
	t.Cleanup(func() {
		if ex.Websocket.IsEnabled() {
			assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		}
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only error when already disconnected")
		}
	})
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	conn, err := ex.Websocket.GetConnection(a)
	require.NoErrorf(t, err, "GetConnection must not error for %s", a)
	return ex, conn
}

// liveSubscriptionCase connects a live instance for asset a, unsubscribes one of its subscriptions with unsubscribe
// and, when subscribe is set, subscribes it again with subscribe
func liveSubscriptionCase(t *testing.T, a asset.Item, unsubscribe, subscribe func(*Exchange, context.Context, websocket.Connection, subscription.List) error) {
	t.Helper()
	if mockTests {
		t.Skip("the mock build serves no Gate websocket")
	}
	ex, conn := newLiveWebsocketConnection(t, a)
	subs := ex.Websocket.GetSubscriptions()
	require.NotEmptyf(t, subs, "Connect must subscribe the %s connection", a)
	sub := subs[0]
	require.NoError(t, unsubscribe(ex, t.Context(), conn, subscription.List{sub}), "unsubscribing must not error")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "the unsubscribed channel should leave the store")
	if subscribe == nil {
		return
	}
	require.NoError(t, subscribe(ex, t.Context(), conn, subscription.List{sub}), "subscribing must not error")
	assert.NotNil(t, ex.Websocket.GetSubscription(sub), "the subscribed channel should be stored")
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	t.Run("live spot channel", func(t *testing.T) {
		t.Parallel()
		liveSubscriptionCase(t, asset.Spot, (*Exchange).Unsubscribe, nil)
	})
}
