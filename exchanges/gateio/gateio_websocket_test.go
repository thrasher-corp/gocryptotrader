package gateio

import (
	"strconv"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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

func TestProcessOrderbookUpdateWithSnapshot(t *testing.T) {
	t.Parallel()

	e := new(Exchange) //nolint:govet // Intentional shadow
	require.NoError(t, testexch.Setup(e))
	e.Name = "ProcessOrderbookUpdateWithSnapshot"
	e.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: spotOrderbookUpdateWithSnapshotChannel, Asset: asset.Spot, Levels: 50},
	}
	expanded, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err)

	conn := &FixtureConnection{}
	err = e.Websocket.AddSubscriptions(conn, expanded...)
	require.NoError(t, err)

	e.wsOBResubMgr.lookup[key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}] = true

	for _, tc := range []struct {
		payload []byte
		err     error
	}{
		{payload: []byte(`{"t":"bingbong"}`), err: strconv.ErrSyntax},
		{payload: []byte(`{"s":"ob.50"}`), err: errMalformedData},
		{payload: []byte(`{"s":"ob..50"}`), err: currency.ErrCannotCreatePair},
		{payload: []byte(`{"s":"ob.BTC_USDT.50","full":true}`), err: orderbook.ErrLastUpdatedNotSet},
		{payload: []byte(`{"s":"ob.DING_USDT.50","full":true}`), err: nil}, // asset not enabled
		{
			// Simulate orderbook update already resubscribing
			payload: []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
			err:     nil,
		},
		{
			// Full snapshot will reset resubscribing state
			payload: []byte(`{"t":1757377580046,"full":true,"s":"ob.BTC_USDT.50","u":27053258981,"b":[["111666","0.131287"],["111665.3","0.048403"],["111665.2","0.268681"],["111665.1","0.153269"],["111664.9","0.004"],["111663.8","0.010919"],["111663.7","0.214867"],["111661.8","0.268681"],["111659.4","0.01144"],["111659.3","0.184127"],["111658.4","0.268681"],["111658.3","0.11897"],["111656.9","0.00653"],["111656.7","0.184127"],["111656.1","0.040381"],["111655","0.044859"],["111654.9","0.268681"],["111654.8","0.033575"],["111653.9","0.184127"],["111653.6","0.601785"],["111653.5","0.017118"],["111651.7","0.160346"],["111651.6","0.184127"],["111651.5","0.268681"],["111650.1","0.09042"],["111647.9","0.191292"],["111647.5","0.268681"],["111646","0.098528"],["111645.9","0.1443"],["111645.6","0.184127"],["111643.8","1.015409"],["111643","0.099889"],["111641.5","0.004925"],["111641.2","0.179895"],["111641.1","0.184127"],["111640.7","0.268681"],["111638.6","0.184912"],["111638.4","0.010182"],["111637.6","0.026862"],["111637.5","0.09042"],["111636.6","0.184127"],["111634.8","0.129187"],["111634.7","0.014213"],["111633.9","0.268681"],["111632.1","0.184127"],["111631.8","0.1443"],["111631.6","0.027"],["111631.3","0.089539"],["111630.3","0.00001"],["111629.6","0.000029"]],"a":[["111666.1","0.818887"],["111668.3","0.008062"],["111668.5","0.005399"],["111670.3","0.043892"],["111670.4","0.019653"],["111673.7","0.046898"],["111674.1","0.004227"],["111674.4","0.026258"],["111674.8","0.09042"],["111674.9","0.268681"],["111675","0.004227"],["111676","0.004227"],["111676.8","0.005"],["111677","0.004227"],["111678.1","0.077789"],["111678.2","0.210991"],["111678.3","0.268681"],["111678.4","0.025039"],["111678.5","0.051456"],["111679.2","0.007163"],["111679.5","0.013019"],["111681.5","0.036343"],["111681.7","0.268681"],["111682.9","0.184127"],["111685.2","0.184127"],["111685.8","0.040538"],["111686.4","0.201931"],["111687.3","0.03"],["111687.4","0.09042"],["111687.5","0.452808"],["111687.6","1.815093"],["111691.9","0.139287"],["111692.2","0.184127"],["111693.7","0.268681"],["111694.3","1.05115"],["111694.5","0.184127"],["111697","0.184127"],["111697.1","0.268681"],["111697.4","0.0967"],["111698.7","0.1443"],["111699.5","0.014213"],["111700.2","0.601783"],["111700.7","0.09042"],["111700.9","0.367517"],["111701.5","0.184127"],["111705.2","0.017703"],["111706","0.184127"],["111707.6","0.268681"],["111709.9","0.1443"],["111710.2","0.004"]]}`),
			err:     nil,
		},
		{
			// Incremental update will apply correctly
			payload: []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
			err:     nil,
		},
		{
			// Incremental update out of order will force resubscription
			payload: []byte(`{"t":1757377580073,"s":"ob.BTC_USDT.50","u":27053258987,"U":27053258982,"b":[["111666","0.146841"]],"a":[["111666.1","0.791633"],["111676.8","0.014"]]}`),
			err:     nil,
		},
	} {
		err := e.processOrderbookUpdateWithSnapshot(conn, tc.payload, time.Now())
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err)
	}
}
