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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
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
	}, 10*time.Second, 10*time.Millisecond, "out-of-order update must successfully resubscribe in the background")
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

func TestProcessSpotOrders(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	push := func(account string) []byte {
		return []byte(`{"time":1605175506,"channel":"spot.orders","event":"update","result":[{"id":"30784435","user":123456,"text":"t-abc","create_time_ms":"1605175506123","update_time_ms":"1605175507123","event":"put","currency_pair":"BTC_USDT","type":"limit","account":"` + account + `","side":"sell","amount":"1","price":"10001","left":"0.25","fee":"0.5"}]}`)
	}
	for account, a := range map[string]asset.Item{
		"":             asset.Spot,
		"spot":         asset.Spot,
		"unified":      asset.Spot,
		"margin":       asset.Margin,
		"cross_margin": asset.CrossMargin,
	} {
		require.NoErrorf(t, ex.processSpotOrders(t.Context(), push(account)), "processSpotOrders must not error for account %q", account)
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
	}
	assert.ErrorIs(t, ex.processSpotOrders(t.Context(), push("bogus")), asset.ErrNotSupported, "processSpotOrders should reject an unknown account")
	assert.Error(t, ex.processSpotOrders(t.Context(), []byte(`{`)), "processSpotOrders should reject malformed JSON")
	assert.ErrorIs(t, ex.processSpotOrders(t.Context(), []byte(`{"time":1605175506,"channel":"spot.orders","event":"update","result":[{"id":"1","side":"bogus","type":"limit","account":"spot"}]}`)), order.ErrSideIsInvalid, "processSpotOrders should reject an unknown side")
	assert.ErrorIs(t, ex.processSpotOrders(t.Context(), []byte(`{"time":1605175506,"channel":"spot.orders","event":"update","result":[{"id":"1","side":"sell","type":"iceberg","account":"spot"}]}`)), order.ErrUnrecognisedOrderType, "processSpotOrders should reject an unknown order type")
	assert.ErrorContains(t, ex.processSpotOrders(t.Context(), []byte(`{"time":1605175506,"channel":"spot.orders","event":"finish","result":[{"id":"1","event":"finish","side":"sell","type":"limit","account":"spot","finish_as":"bogus"}]}`)), "unrecognised order status", "processSpotOrders should reject an unknown finish reason")
	require.Empty(t, ex.Websocket.DataHandler.C, "processSpotOrders must not relay a rejected push")
}

func TestProcessSpotOrdersLifecycle(t *testing.T) {
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
}

func TestProcessSpotOrdersMixedEvents(t *testing.T) {
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
}

// TestProcessSpotOrdersMerge relays successive pushes into a stored order, merging each as the order manager does
func TestProcessSpotOrdersMerge(t *testing.T) {
	t.Parallel()
	const times = `"create_time_ms":"1694655225315","update_time_ms":"1694655226315"`
	relay := func(t *testing.T, ex *Exchange, o string) *order.Detail {
		t.Helper()
		require.NoError(t, ex.processSpotOrders(t.Context(), []byte(`{"time":1694655226,"channel":"spot.orders","event":"update","result":[`+o+`]}`)), "processSpotOrders must not error")
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
		stored := relay(t, ex, `{"id":"21","event":"finish","finish_as":"cancelled","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"0.5","filled_amount":"0.5","fee":"0.001","fee_currency":"USDT",`+times+`}`)
		late := relay(t, ex, `{"id":"21","event":"update","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"0.5","filled_amount":"0.5","fee":"0.001","fee_currency":"USDT",`+times+`}`)
		require.NoError(t, stored.UpdateOrderFromDetail(late), "UpdateOrderFromDetail must not error")
		assert.Equal(t, order.Cancelled, stored.Status, "a delayed open push should not reopen a cancelled order")
	})

	t.Run("a filled order keeps no remainder from its partial fills", func(t *testing.T) {
		t.Parallel()
		ex := newExchange(t)
		stored := relay(t, ex, `{"id":"22","event":"put","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"1",`+times+`}`)
		for _, o := range []string{
			`{"id":"22","event":"update","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"0.5","filled_amount":"0.5",` + times + `}`,
			`{"id":"22","event":"finish","finish_as":"filled","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"80000","left":"0","filled_amount":"1",` + times + `}`,
		} {
			require.NoError(t, stored.UpdateOrderFromDetail(relay(t, ex, o)), "UpdateOrderFromDetail must not error")
		}
		assert.Equal(t, order.Filled, stored.Status, "the finish push should leave the order filled")
		assert.Equal(t, 1.0, stored.ExecutedAmount, "the order should be fully executed")
		assert.Zero(t, stored.RemainingAmount, "a filled order should keep no remainder from a partial fill")
	})

	t.Run("a finish without a fee does not relabel a stored fee", func(t *testing.T) {
		t.Parallel()
		ex := newExchange(t)
		stored := &order.Detail{Exchange: ex.Name, OrderID: "24", Fee: 0.001, FeeAsset: currency.BTC}
		finished := relay(t, ex, `{"id":"24","event":"finish","finish_as":"filled","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amount":"1","price":"60000","left":"0","filled_total":"60000","filled_amount":"1","avg_deal_price":"60000","fee":"0","fee_currency":"GT",`+times+`}`)
		require.NoError(t, stored.UpdateOrderFromDetail(finished), "UpdateOrderFromDetail must not error")
		assert.Equal(t, 0.001, stored.Fee, "a finish without a fee should keep the stored fee")
		assert.Equal(t, currency.BTC, stored.FeeAsset, "a finish without a fee should keep the stored fee's currency")
	})

	t.Run("a delayed earlier push does not put back an older fee, cost or average price", func(t *testing.T) {
		t.Parallel()
		ex := newExchange(t)
		stored := relay(t, ex, `{"id":"23","event":"finish","finish_as":"filled","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amount":"1","price":"65000","left":"0","filled_total":"65000","filled_amount":"1","avg_deal_price":"65000","fee":"0.001","fee_currency":"BTC",`+times+`}`)
		late := relay(t, ex, `{"id":"23","event":"update","finish_as":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amount":"1","price":"65000","left":"0.5","filled_total":"30000","filled_amount":"0.5","avg_deal_price":"60000","fee":"0.0005","fee_currency":"BTC",`+times+`}`)
		require.NoError(t, stored.UpdateOrderFromDetail(late), "UpdateOrderFromDetail must not error")
		assert.Equal(t, 0.001, stored.Fee, "a delayed push should not put back an older fee")
		assert.Equal(t, 65000.0, stored.Cost, "a delayed push should not put back an older cost")
		assert.Equal(t, 65000.0, stored.AverageExecutedPrice, "a delayed push should not put back an older average price")
	})
}

func TestWebsocketHandleSpotDataNullResult(t *testing.T) {
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
