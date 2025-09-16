package gateio

import (
	"context"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
	expected    []account.Change
}

func TestProcessSpotBalances(t *testing.T) {
	t.Parallel()
	e := new(Exchange) //nolint:govet // Intentional shadow
	e.SetDefaults()
	e.Name = "ProcessSpotBalancesTest"

	// Sequential tests, do not use t.Run(); Some timestamps are deliberately identical from trading activity
	for _, tc := range []websocketBalancesTest{
		{
			input: []byte(`[{"timestamp":"1755718222"}]`),
			err:   exchange.ErrCredentialsAreEmpty,
		},
		{
			deployCreds: true,
			input:       []byte(`[{"timestamp":"1755718222","timestamp_ms":"1755718222394","user":"12870774","currency":"USDT","change":"0","total":"3087.01142272991036062136","available":"3081.68642272991036062136","freeze":"5.325","freeze_change":"5.32500000000000000000","change_type":"order-create"}]`),
			expected: []account.Change{
				{
					Account:   "12870774",
					AssetType: asset.Spot,
					Balance: &account.Balance{
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
		{
			deployCreds: true,
			input:       []byte(`[{"timestamp":"1755718222","timestamp_ms":"1755718222394","user":"12870774","currency":"USDT","change":"-3.99375000000000000000","total":"3083.01767272991036062136","available":"3081.68642272991036062136","freeze":"1.33125","freeze_change":"-3.99375000000000000000","change_type":"order-match"}]`),
			expected: []account.Change{
				{
					Account:   "12870774",
					AssetType: asset.Spot,
					Balance: &account.Balance{
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
	} {
		ctx := t.Context()
		if tc.deployCreds {
			ctx = account.DeployCredentialsToContext(ctx, &account.Credentials{Key: "test", Secret: "test"})
		}
		err := e.processSpotBalances(ctx, tc.input)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err, "processSpotBalances must not error")
		checkAccountChange(ctx, t, e, &tc)
	}
}

func TestProcessBalancePushData(t *testing.T) {
	t.Parallel()
	e := new(Exchange) //nolint:govet // Intentional shadow
	e.SetDefaults()
	e.Name = "ProcessFuturesBalancesTest"

	// Sequential tests, do not use t.Run(); Some timestamps are deliberately identical from trading activity
	for _, tc := range []websocketBalancesTest{
		{
			input: []byte(`[{"timestamp":"1755718222"}]`),
			err:   exchange.ErrCredentialsAreEmpty,
		},
		{
			deployCreds: true,
			input:       []byte(`[{"balance":2214.191673190433,"change":-0.0025776,"currency":"usdt","text":"TCOM_USDT:263179103241933596","time":1755738515,"time_ms":1755738515671,"type":"fee","user":"12870774"}]`),
			expected: []account.Change{
				{
					Account:   "12870774",
					AssetType: asset.USDTMarginedFutures,
					Balance: &account.Balance{
						Currency:               currency.USDT,
						Total:                  2214.191673190433,
						Free:                   2214.191673190433,
						AvailableWithoutBorrow: 2214.191673190433,
						UpdatedAt:              time.UnixMilli(1755738515671),
					},
				},
			},
		},
		{
			deployCreds: true,
			input:       []byte(`[{"balance":2214.189114310433,"change":-0.00255888,"currency":"usdt","text":"TCOM_USDT:263179103241933644","time":1755738516,"time_ms":1755738516430,"type":"fee","user":"12870774"}]`),
			expected: []account.Change{
				{
					Account:   "12870774",
					AssetType: asset.USDTMarginedFutures,
					Balance: &account.Balance{
						Currency:               currency.USDT,
						Total:                  2214.189114310433,
						Free:                   2214.189114310433,
						AvailableWithoutBorrow: 2214.189114310433,
						UpdatedAt:              time.UnixMilli(1755738516430),
					},
				},
			},
		},
	} {
		ctx := t.Context()
		if tc.deployCreds {
			ctx = account.DeployCredentialsToContext(ctx, &account.Credentials{Key: "test", Secret: "test"})
		}
		err := e.processBalancePushData(ctx, tc.input, asset.USDTMarginedFutures)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err, "processBalancePushData must not error")
		require.Len(t, e.Websocket.DataHandler, 1)
		checkAccountChange(ctx, t, e, &tc)
	}
}

func checkAccountChange(ctx context.Context, t *testing.T, exch *Exchange, tc *websocketBalancesTest) {
	t.Helper()

	require.Len(t, exch.Websocket.DataHandler, 1)
	payload := <-exch.Websocket.DataHandler
	received, ok := payload.([]account.Change)
	require.Truef(t, ok, "Expected account changes, got %T", payload)

	require.Lenf(t, received, len(tc.expected), "Expected %d changes, got %d", len(tc.expected), len(received))
	for i, change := range received {
		assert.Equal(t, tc.expected[i].Account, change.Account, "account should equal")
		assert.Equal(t, tc.expected[i].AssetType, change.AssetType, "asset type should equal")
		assert.True(t, tc.expected[i].Balance.Currency.Equal(change.Balance.Currency), "currency should equal")
		assert.Equal(t, tc.expected[i].Balance.Total, change.Balance.Total, "total should equal")
		assert.Equal(t, tc.expected[i].Balance.Hold, change.Balance.Hold, "hold should equal")
		assert.Equal(t, tc.expected[i].Balance.Free, change.Balance.Free, "free should equal")
		assert.Equal(t, tc.expected[i].Balance.AvailableWithoutBorrow, change.Balance.AvailableWithoutBorrow, "available without borrow should equal")
		assert.Equal(t, tc.expected[i].Balance.Borrowed, change.Balance.Borrowed, "borrowed should equal")
		assert.Equal(t, tc.expected[i].Balance.UpdatedAt, change.Balance.UpdatedAt, "updated at should equal")

		creds, err := exch.GetCredentials(ctx)
		require.NoError(t, err, "GetCredentials must not error")
		stored, err := account.GetBalance(exch.Name, tc.expected[i].Account, creds, tc.expected[i].AssetType, tc.expected[i].Balance.Currency)
		require.NoError(t, err, "GetBalance must not error")
		assert.Equal(t, tc.expected[i].Balance.Free, stored.GetFree(), "free balance should equal with accounts stored value")
	}
}
