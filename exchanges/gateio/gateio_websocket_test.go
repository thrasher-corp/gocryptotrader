package gateio

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetWSPingHandler(t *testing.T) {
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
		got, err := getWSPingHandler(tc.channel)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, time.Second*10, got.Delay)
		require.Equal(t, gws.TextMessage, got.MessageType)
		require.Contains(t, string(got.Message), tc.channel)
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

	for i, tc := range []websocketBalancesTest{
		{
			input: []byte(`[{"timestamp":"1755718222"}]`),
			err:   exchange.ErrCredentialsAreEmpty,
		},
		{
			deployCreds: true,
			input:       []byte(`[{"timestamp":"1755718222","timestamp_ms":"1755718222394","user":"12870774","currency":"USDT","change":"0","total":"3087.01142272991036062136","available":"3081.68642272991036062136","freeze":"5.325","freeze_change":"5.32500000000000000000","change_type":"order-create"}]`),
			expected: accounts.SubAccounts{
				{
					ID:        "12870774",
					AssetType: asset.Spot,
					Balances: accounts.CurrencyBalances{
						currency.USDT: accounts.Balance{
							Currency:               currency.USDT,
							Total:                  3087.01142272991036062136,
							Free:                   3081.68642272991036062136,
							Hold:                   5.325,
							AvailableWithoutBorrow: 3081.68642272991036062136,
							UpdatedAt:              time.UnixMilli(1755718222394),
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
					ID:        "12870774",
					AssetType: asset.Spot,
					Balances: accounts.CurrencyBalances{
						currency.USDT: accounts.Balance{
							Currency:               currency.USDT,
							Total:                  3083.01767272991036062136,
							Free:                   3081.68642272991036062136,
							Hold:                   1.33125,
							AvailableWithoutBorrow: 3081.68642272991036062136,
							UpdatedAt:              time.UnixMilli(1755718222394),
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

func TestProcessBalancePushData(t *testing.T) { //nolint:tparallel // Sequential tests, do not use t.Parallel(); Some timestamps are deliberately identical from trading activity
	t.Parallel()
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
	// Captured from live GateIO websocket data on 2026-09-03.
	// Treat these payloads as semi-trusted until independently confirmed.
	payloads := [][]byte{
		[]byte(`[{"balance":6625.2967002542,"change":0.0008823675,"text":"SCRT_USDT","time":1788148805,"time_ms":1788148805954,"type":"fund","user":"12870774","currency":"usdt"}]`),
		[]byte(`[{"balance":6625.2967002542,"change":0.0008823675,"text":"SCRT_USDT","time":1788148805,"time_ms":1788148805954,"type":"fund","user":"12870774","currency":"usdt"}]`),
		[]byte(`[{"balance":6521.586841973412,"change":-0.01556544,"text":"RAVE_USDT:294985777215626819","time":1788387249,"time_ms":1788387249995,"type":"fee","user":"12870774","currency":"usdt"}]`),
		[]byte(`[{"balance":6522.937261628622,"change":1.35041965521,"text":"RAVE_USDT:294985777215626819","time":1788387249,"time_ms":1788387249995,"type":"pnl","user":"12870774","currency":"usdt"}]`),
	}
	wantBalances := []float64{6625.2967002542, 6625.2967002542, 6521.586841973412, 6522.937261628622}

	for i := range payloads {
		startedAt := time.Now()
		require.NoError(t, ex.processBalancePushData(ctx, payloads[i], asset.USDTMarginedFutures),
			"processBalancePushData must process the captured balance update")
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
	message := <-ex.Websocket.DataHandler.C
	changes, ok := message.Data.(accounts.SubAccounts)
	require.True(t, ok, "captured balance payload must emit subaccount changes")
	drawnDown := changes[0].Balances[currency.USDT.Lower()]
	assert.Equal(t, 5000.0, drawnDown.Hold, "held margin should survive a temporary balance drawdown")
	assert.Zero(t, drawnDown.Free, "free balance should not become negative")
	require.NoError(t, ex.processBalancePushData(ctx,
		[]byte(`[{"balance":6106.7961637458,"time":1788148807,"time_ms":1788148807000,"user":"12870774","currency":"usdt"}]`),
		asset.USDTMarginedFutures), "processBalancePushData must process balance recovery")
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

	for _, tc := range []struct {
		payload      []byte
		err          error
		assertUpdate bool
	}{
		{payload: []byte(`{"t":"bingbong"}`), err: types.ErrInvalidTimestampFormat},
		{payload: []byte(`{"s":"ob.50"}`), err: common.ErrMalformedData},
		{payload: []byte(`{"s":"ob..50"}`), err: currency.ErrCreatingPair},
		{
			// Simulate orderbook update already resubscribing
			payload: []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
		},
		{payload: []byte(`{"s":"ob.BTC_USDT.50","full":true}`), err: orderbook.ErrLastUpdatedNotSet},
		{
			// Full snapshot will reset resubscribing state
			payload: []byte(`{"t":1757377580046,"full":true,"s":"ob.BTC_USDT.50","u":27053258981,"b":[["111666","0.131287"],["111665.3","0.048403"],["111665.2","0.268681"],["111665.1","0.153269"],["111664.9","0.004"],["111663.8","0.010919"],["111663.7","0.214867"],["111661.8","0.268681"],["111659.4","0.01144"],["111659.3","0.184127"],["111658.4","0.268681"],["111658.3","0.11897"],["111656.9","0.00653"],["111656.7","0.184127"],["111656.1","0.040381"],["111655","0.044859"],["111654.9","0.268681"],["111654.8","0.033575"],["111653.9","0.184127"],["111653.6","0.601785"],["111653.5","0.017118"],["111651.7","0.160346"],["111651.6","0.184127"],["111651.5","0.268681"],["111650.1","0.09042"],["111647.9","0.191292"],["111647.5","0.268681"],["111646","0.098528"],["111645.9","0.1443"],["111645.6","0.184127"],["111643.8","1.015409"],["111643","0.099889"],["111641.5","0.004925"],["111641.2","0.179895"],["111641.1","0.184127"],["111640.7","0.268681"],["111638.6","0.184912"],["111638.4","0.010182"],["111637.6","0.026862"],["111637.5","0.09042"],["111636.6","0.184127"],["111634.8","0.129187"],["111634.7","0.014213"],["111633.9","0.268681"],["111632.1","0.184127"],["111631.8","0.1443"],["111631.6","0.027"],["111631.3","0.089539"],["111630.3","0.00001"],["111629.6","0.000029"]],"a":[["111666.1","0.818887"],["111668.3","0.008062"],["111668.5","0.005399"],["111670.3","0.043892"],["111670.4","0.019653"],["111673.7","0.046898"],["111674.1","0.004227"],["111674.4","0.026258"],["111674.8","0.09042"],["111674.9","0.268681"],["111675","0.004227"],["111676","0.004227"],["111676.8","0.005"],["111677","0.004227"],["111678.1","0.077789"],["111678.2","0.210991"],["111678.3","0.268681"],["111678.4","0.025039"],["111678.5","0.051456"],["111679.2","0.007163"],["111679.5","0.013019"],["111681.5","0.036343"],["111681.7","0.268681"],["111682.9","0.184127"],["111685.2","0.184127"],["111685.8","0.040538"],["111686.4","0.201931"],["111687.3","0.03"],["111687.4","0.09042"],["111687.5","0.452808"],["111687.6","1.815093"],["111691.9","0.139287"],["111692.2","0.184127"],["111693.7","0.268681"],["111694.3","1.05115"],["111694.5","0.184127"],["111697","0.184127"],["111697.1","0.268681"],["111697.4","0.0967"],["111698.7","0.1443"],["111699.5","0.014213"],["111700.2","0.601783"],["111700.7","0.09042"],["111700.9","0.367517"],["111701.5","0.184127"],["111705.2","0.017703"],["111706","0.184127"],["111707.6","0.268681"],["111709.9","0.1443"],["111710.2","0.004"]]}`),
		},
		{
			// Incremental update will apply correctly
			payload:      []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
			assertUpdate: true,
		},
		{
			// Incremental update out of order will force resubscription
			payload: []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
		},
	} {
		// Sequential tests, do not use t.Parallel(); Some timestamps are deliberately identical from trading activity
		err := e.processOrderbookUpdateWithSnapshot(t.Context(), conn, tc.payload, time.Now(), asset.Spot)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
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
	}

	require.Eventually(t, func() bool {
		sub := e.Websocket.GetSubscription(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "ob.BTC_USDT.50", Asset: asset.Spot}})
		return sub != nil && sub.State() == subscription.SubscribedState
	}, time.Second, 10*time.Millisecond, "out-of-order update must successfully resubscribe in the background")
}

func TestProcessOrderbookUpdateWithSnapshotEndsRecovery(t *testing.T) {
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
			exchangeConfig, err := cfg.GetExchangeConfig("GateIO")
			require.NoError(t, err, "shipped config must contain GateIO")
			require.NotNil(t, exchangeConfig.Features, "GateIO features must be configured")
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
			subs, err := e.generateSubscriptionsSpot()
			require.NoError(t, err, "migrated subscriptions must expand and validate")
			assert.NotEmpty(t, subs, "migrated configuration should generate subscriptions")
			for _, s := range subs {
				assert.Equal(t, test.channel, channelName(s), "migrated subscriptions should use the expected orderbook feed")
			}
		})
	}
}

func TestEnabledStandardMarginAssetsForPair(t *testing.T) {
	t.Parallel()

	ex := setupExchangeWithSpotOnlyEnabledBTCUSDT(t)
	pair := currency.NewPairWithDelimiter("BTC", "USDT", "_")

	assets := slices.Collect(ex.enabledStandardMarginAssetsForPair(pair))
	require.Equal(t, []asset.Item{asset.Spot}, assets, "enabledStandardMarginAssetsForPair must only return spot when margin assets are disabled")

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Margin, true), "SetAssetEnabled must not error")
	assets = slices.Collect(ex.enabledStandardMarginAssetsForPair(pair))
	require.Equal(t, []asset.Item{asset.Spot, asset.Margin}, assets, "enabledStandardMarginAssetsForPair must include margin after enabling margin")
}

func TestProcessTickerEnabledAssetsOnly(t *testing.T) {
	t.Parallel()

	ex := setupExchangeWithSpotOnlyEnabledBTCUSDT(t)
	err := ex.processTicker(t.Context(), []byte(`{
		"currency_pair":"BTC_USDT",
		"last":"19106.55",
		"lowest_ask":"19108.71",
		"highest_bid":"19106.55",
		"base_volume":"2811.3042155865",
		"quote_volume":"53441606.52411221454674732293",
		"high_24h":"19417.74",
		"low_24h":"18434.21"
	}`), time.Unix(1606291803, 0))
	require.NoError(t, err, "processTicker must not error")

	require.Len(t, ex.Websocket.DataHandler.C, 1, "websocket data handler must receive one ticker batch")
	payload := <-ex.Websocket.DataHandler.C
	tickerBatch, ok := payload.Data.([]ticker.Price)
	require.True(t, ok, "expected []ticker.Price payload")
	require.Len(t, tickerBatch, 1, "processTicker must only emit enabled spot asset data")
	assert.Equal(t, asset.Spot, tickerBatch[0].AssetType, "processTicker should only emit spot when margin assets are disabled")
}

func TestProcessCandlestickEnabledAssetsOnly(t *testing.T) {
	t.Parallel()

	ex := setupExchangeWithSpotOnlyEnabledBTCUSDT(t)
	err := ex.processCandlestick(t.Context(), []byte(`{
		"t":"1606292580",
		"v":"2362.32035",
		"c":"19128.1",
		"h":"19128.1",
		"l":"19128.1",
		"o":"19128.1",
		"n":"1m_BTC_USDT"
	}`))
	require.NoError(t, err, "processCandlestick must not error")

	require.Len(t, ex.Websocket.DataHandler.C, 1, "websocket data handler must receive one candle batch")
	payload := <-ex.Websocket.DataHandler.C
	candleBatch, ok := payload.Data.([]kline.Item)
	require.True(t, ok, "expected []kline.Item payload")
	require.Len(t, candleBatch, 1, "processCandlestick must only emit enabled spot asset data")
	assert.Equal(t, asset.Spot, candleBatch[0].Asset, "processCandlestick should only emit spot when margin assets are disabled")
}

func TestProcessOrderbookSnapshotEnabledAssetsOnly(t *testing.T) {
	t.Parallel()

	ex := setupExchangeWithSpotOnlyEnabledBTCUSDT(t)
	pair := currency.NewPairWithDelimiter("BTC", "USDT", "_")
	err := ex.processOrderbookSnapshot([]byte(`{
		"t":1606295412123,
		"lastUpdateId":48791820,
		"s":"BTC_USDT",
		"bids":[["19079.55","0.0195"]],
		"asks":[["19080.24","0.1638"]]
	}`), time.Unix(1606295412, 0))
	require.NoError(t, err, "processOrderbookSnapshot must not error")

	_, err = ex.Websocket.Orderbook.LastUpdateID(pair, asset.Spot)
	require.NoError(t, err, "spot orderbook must be updated")
	_, err = ex.Websocket.Orderbook.LastUpdateID(pair, asset.Margin)
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "margin orderbook should not be updated when margin is disabled")
	_, err = ex.Websocket.Orderbook.LastUpdateID(pair, asset.CrossMargin)
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "cross margin orderbook should not be updated when cross margin is disabled")
}

// setupExchangeWithSpotOnlyEnabledBTCUSDT returns a GateIO exchange where BTC_USDT is available for spot, margin and cross margin, but only spot is enabled.
func setupExchangeWithSpotOnlyEnabledBTCUSDT(t *testing.T) *Exchange {
	t.Helper()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	pair := currency.NewPairWithDelimiter("BTC", "USDT", "_")
	for _, a := range standardMarginAssetTypes {
		require.NoError(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{pair}, false), "StorePairs available must not error")
		require.NoError(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{pair}, true), "StorePairs enabled must not error")
	}
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Spot, true), "SetAssetEnabled must not error")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Margin, false), "SetAssetEnabled must not error")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.CrossMargin, false), "SetAssetEnabled must not error")
	return ex
}
