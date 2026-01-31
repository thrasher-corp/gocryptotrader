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
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
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
					ID:        "12870774",
					AssetType: asset.USDTMarginedFutures,
					Balances: accounts.CurrencyBalances{
						usdtLower: accounts.Balance{
							Currency:               usdtLower,
							Total:                  2214.191673190433,
							Free:                   2214.191673190433,
							AvailableWithoutBorrow: 2214.191673190433,
							UpdatedAt:              time.UnixMilli(1755738515671),
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
					ID:        "12870774",
					AssetType: asset.USDTMarginedFutures,
					Balances: accounts.CurrencyBalances{
						usdtLower: accounts.Balance{
							Currency:               usdtLower,
							Total:                  2214.189114310433,
							Free:                   2214.189114310433,
							AvailableWithoutBorrow: 2214.189114310433,
							UpdatedAt:              time.UnixMilli(1755738516430),
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

func checkAccountChange(ctx context.Context, t *testing.T, exch *Exchange, tc *websocketBalancesTest) {
	t.Helper()

	require.Len(t, exch.Websocket.DataHandler.C, 1)
	payload := <-exch.Websocket.DataHandler.C
	received, ok := payload.Data.(accounts.SubAccounts)
	require.Truef(t, ok, "Expected account changes, got %T", payload)

	require.Lenf(t, received, len(tc.expected), "Expected %d changes, got %d", len(tc.expected), len(received))
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

func TestExtractOrderbookLimit(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	_, err := e.extractOrderbookLimit(1337)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.extractOrderbookLimit(asset.Spot)
	require.ErrorIs(t, err, subscription.ErrNotFound)

	err = e.Websocket.AddSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.Interval(time.Millisecond * 420)})
	require.NoError(t, err)

	_, err = e.extractOrderbookLimit(asset.Spot)
	require.ErrorIs(t, err, errInvalidOrderbookUpdateInterval)

	err = e.Websocket.RemoveSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.Interval(time.Millisecond * 420)})
	require.NoError(t, err)

	// Add dummy subscription so that it can be matched and a limit/level can be extracted for initial orderbook sync spot.
	err = e.Websocket.AddSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds})
	require.NoError(t, err)

	for _, tc := range []struct {
		asset asset.Item
		exp   uint64
	}{
		{asset: asset.Spot, exp: 100},
		{asset: asset.USDTMarginedFutures, exp: futuresOrderbookUpdateLimit},
		{asset: asset.CoinMarginedFutures, exp: futuresOrderbookUpdateLimit},
		{asset: asset.DeliveryFutures, exp: deliveryFuturesUpdateLimit},
		{asset: asset.Options, exp: optionOrderbookUpdateLimit},
	} {
		limit, err := e.extractOrderbookLimit(tc.asset)
		require.NoError(t, err)
		require.Equal(t, tc.exp, limit)
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
		{payload: []byte(`{"t":"bingbong"}`), err: strconv.ErrSyntax},
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
