package gateio

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
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
	e.Name = "ProcessOrderbookUpdateWithSnapshot"
	e.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: spotOrderbookV2, Asset: asset.Spot, Levels: 50},
	}
	subs, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err)

	conn := &FixtureConnection{}
	err = e.Websocket.AddSubscriptions(conn, subs...)
	require.NoError(t, err)

	e.wsOBResubMgr.lookup[key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}] = true

	for _, tc := range []struct {
		payload []byte
		err     error
	}{
		{payload: []byte(`{"t":"bingbong"}`), err: types.ErrInvalidTimestampFormat},
		{payload: []byte(`{"s":"ob.50"}`), err: common.ErrMalformedData},
		{payload: []byte(`{"s":"ob..50"}`), err: currency.ErrCreatingPair},
		{payload: []byte(`{"s":"ob.BTC_USDT.50","full":true}`), err: orderbook.ErrLastUpdatedNotSet},
		{
			// Simulate orderbook update already resubscribing
			payload: []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
		},
		{
			// Full snapshot will reset resubscribing state
			payload: []byte(`{"t":1757377580046,"full":true,"s":"ob.BTC_USDT.50","u":27053258981,"b":[["111666","0.131287"],["111665.3","0.048403"],["111665.2","0.268681"],["111665.1","0.153269"],["111664.9","0.004"],["111663.8","0.010919"],["111663.7","0.214867"],["111661.8","0.268681"],["111659.4","0.01144"],["111659.3","0.184127"],["111658.4","0.268681"],["111658.3","0.11897"],["111656.9","0.00653"],["111656.7","0.184127"],["111656.1","0.040381"],["111655","0.044859"],["111654.9","0.268681"],["111654.8","0.033575"],["111653.9","0.184127"],["111653.6","0.601785"],["111653.5","0.017118"],["111651.7","0.160346"],["111651.6","0.184127"],["111651.5","0.268681"],["111650.1","0.09042"],["111647.9","0.191292"],["111647.5","0.268681"],["111646","0.098528"],["111645.9","0.1443"],["111645.6","0.184127"],["111643.8","1.015409"],["111643","0.099889"],["111641.5","0.004925"],["111641.2","0.179895"],["111641.1","0.184127"],["111640.7","0.268681"],["111638.6","0.184912"],["111638.4","0.010182"],["111637.6","0.026862"],["111637.5","0.09042"],["111636.6","0.184127"],["111634.8","0.129187"],["111634.7","0.014213"],["111633.9","0.268681"],["111632.1","0.184127"],["111631.8","0.1443"],["111631.6","0.027"],["111631.3","0.089539"],["111630.3","0.00001"],["111629.6","0.000029"]],"a":[["111666.1","0.818887"],["111668.3","0.008062"],["111668.5","0.005399"],["111670.3","0.043892"],["111670.4","0.019653"],["111673.7","0.046898"],["111674.1","0.004227"],["111674.4","0.026258"],["111674.8","0.09042"],["111674.9","0.268681"],["111675","0.004227"],["111676","0.004227"],["111676.8","0.005"],["111677","0.004227"],["111678.1","0.077789"],["111678.2","0.210991"],["111678.3","0.268681"],["111678.4","0.025039"],["111678.5","0.051456"],["111679.2","0.007163"],["111679.5","0.013019"],["111681.5","0.036343"],["111681.7","0.268681"],["111682.9","0.184127"],["111685.2","0.184127"],["111685.8","0.040538"],["111686.4","0.201931"],["111687.3","0.03"],["111687.4","0.09042"],["111687.5","0.452808"],["111687.6","1.815093"],["111691.9","0.139287"],["111692.2","0.184127"],["111693.7","0.268681"],["111694.3","1.05115"],["111694.5","0.184127"],["111697","0.184127"],["111697.1","0.268681"],["111697.4","0.0967"],["111698.7","0.1443"],["111699.5","0.014213"],["111700.2","0.601783"],["111700.7","0.09042"],["111700.9","0.367517"],["111701.5","0.184127"],["111705.2","0.017703"],["111706","0.184127"],["111707.6","0.268681"],["111709.9","0.1443"],["111710.2","0.004"]]}`),
		},
		{
			// Incremental update will apply correctly
			payload: []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
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
	}
}
