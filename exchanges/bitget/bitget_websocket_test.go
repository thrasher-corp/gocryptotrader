package bitget

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// func TestWsReadData(t *testing.T) {
// 	mock := func(tb testing.TB, msg []byte, w *gws.Conn) error {
// 		tb.Helper()
// 		msg, err := json.Marshal("pong")
// 		require.NoError(t, err)
// 		return w.WriteMessage(gws.TextMessage, msg)
// 	}
// 	wsTest := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, mock))
// 	wsTest.Websocket.Enable()
// 	err := exchangeBaseHelper(wsTest)
// 	require.NoError(t, err)
// 	var dialer gws.Dialer
// 	err = wsTest.Websocket.Conn.Dial(context.TODO(), &dialer, http.Header{})
// 	require.NoError(t, err)
// 	err = wsTest.Websocket.AuthConn.Dial(context.TODO(), &dialer, http.Header{})
// 	require.NoError(t, err)
// 	// e.Websocket.Wg.Add(1)
// 	// go e.wsReadData(e.Websocket.Conn)
// 	err = wsTest.Subscribe(defaultSubscriptions)
// 	require.NoError(t, err)
// 	// Implement internal/testing/websocket mockws stuff after merging
// 	// See: https://github.com/thrasher-corp/gocryptotrader/blob/master/exchanges/kraken/kraken_test.go#L1169
// }

func TestWsHandleData(t *testing.T) {
	// Not sure what issues this is preventing. If you figure that out, add a comment about it
	// ch := make(chan struct{})
	// t.Cleanup(func() {
	// 	close(ch)
	// })
	// go func() {
	// 	for {
	// 		select {
	// 		case <-e.Websocket.DataHandler:
	// 			continue
	// 		case <-ch:
	// 			return
	// 		}
	// 	}
	// }()
	verboseTemp := e.Verbose
	e.Verbose = true
	t.Cleanup(func() {
		e.Verbose = verboseTemp
	})
	mockJSON := []byte(`pong`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`notjson`)
	err = e.wsHandleData(t.Context(), mockJSON)
	errInvalidChar := "invalid char"
	assert.ErrorContains(t, err, errInvalidChar)
	mockJSON = []byte(`{"event":"subscribe"}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"error"}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	expectedErr := fmt.Sprintf(errWebsocketGeneric, "Bitget", 0, "")
	assert.EqualError(t, err, expectedErr)
	mockJSON = []byte(`{"event":"login","code":0}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"login","code":1}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	expectedErr = fmt.Sprintf(errWebsocketLoginFailed, "Bitget", "")
	assert.EqualError(t, err, expectedErr)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fakeChannelNotReal"}}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"fakeChannelNotReal"}}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"fakeEventNotReal"}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestTickerDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"SPOT"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"SPOT"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"SPOT"},"data":[{"InstId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"USDT-FUTURES"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"USDT-FUTURES"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"USDT-FUTURES"},"data":[{"InstId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"moo"},"data":[{"InstId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestCandleDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"candle1D"},"data":[["1","2","3","4","5","6","",""]]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["a","2","3","4","5","6","",""]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","a","3","4","5","6","",""]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","a","4","5","6","",""]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","a","5","6","",""]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","4","a","6","",""]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","4","5","a","",""]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","4","5","6","",""]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"candle1D"},"data":[[[{}]]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
}

func TestTradeDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"trade","instId":"BTCUSD"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"trade","instId":"BTCUSD"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"trade"},"data":[]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
}

func TestOrderbookDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"books"},"data":[]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, common.ErrNoResults)
	mockJSON = []byte(`{"action":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[{"bids":[["a","1"]]}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"action":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[{"asks":[["1","a"]]}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"action":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, orderbook.ErrAssetTypeNotSet)
	mockJSON = []byte(`{"action":"update","arg":{"channel":"books","instId":"BTCUSD"},"data":[{"asks":[["1","2"]]}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound)
	mockJSON = []byte(`{"action":"update","arg":{"channel":"books","instId":"BTCUSD"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestAccountSnapshotDataHandler(t *testing.T) {
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"account"},"data":[]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"spot"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"spot"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"futures"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"futures"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestFillDataHandler(t *testing.T) {
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"fill"},"data":[]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"spot"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"spot"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"spot"},"data":[{"symbol":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"futures"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"futures"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"futures"},"data":[{"symbol":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestGenOrderDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"orders"},"data":[]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[{"instId":"BTCUSD","side":"buy","orderType":"limit","feeDetail":[{}]}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[{"instId":"BTCUSD","side":"sell"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"futures"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"futures"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"futures"},"data":[{"instId":"BTCUSD","side":"buy","orderType":"limit","feeDetail":[{}]}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"futures"},"data":[{"instId":"BTCUSD","side":"sell"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestTriggerOrderDatHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"orders-algo"},"data":[]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"spot"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"spot"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"spot"},"data":[{"instId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"futures"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"futures"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"futures"},"data":[{"instId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestPositionsDataHandler(t *testing.T) {
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"positions"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions"},"data":[{"instId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestPositionsHistoryDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"positions-history"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions-history"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions-history"},"data":[{"instId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestIndexPriceDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"index-price"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"index-price","instType":"spot"},"data":[{"symbol":"BTCUSDT"},{"symbol":"USDT/USDT"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestCrossAccountDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"account-crossed"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account-crossed"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestMarginOrderDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"orders-crossed"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-crossed"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-isolated","instId":"BTCUSD"},"data":[{"feeDetail":[{}]}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-crossed","instId":"BTCUSD"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestIsolatedAccountDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"account-isolated"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account-isolated"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}

func TestAccountUpdateDataHandler(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	mockJSON := []byte(`{"event":"update","arg":{"channel":"account"},"data":[]}`)
	err := e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"spot"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"spot"},"data":[{"uTime":"1750142570"}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"futures"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"futures"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), mockJSON)
	assert.NoError(t, err)
}
