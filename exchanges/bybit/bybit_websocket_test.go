package bybit

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

func TestWSHandleTradeData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input []byte
		match string
		err   error
	}{
		{input: []byte(`{"reqId":"12345"}`), match: "12345", err: nil},
		{input: []byte(`{"op":"auth"}`), match: "auth", err: nil},
		{input: []byte(`{"op":"pong"}`), err: nil},
		{input: []byte(`{"op":"pewpewpew"}`), err: errUnhandledStreamData},
	} {
		conn := &FixtureConnection{match: websocket.NewMatch()}
		var ch <-chan []byte
		if tc.match != "" {
			var err error
			ch, err = conn.match.Set(tc.match, 1)
			require.NoError(t, err, "match.Set must not error")
		}
		err := e.wsHandleTradeData(conn, tc.input)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err)
		if tc.match != "" {
			require.Len(t, ch, 1, "must receive 1 message from channel")
			require.Equal(t, tc.input, <-ch, "must be correct")
		}
	}
}

func TestConvertPublicTrades(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		asset   asset.Item
		symbols []string
		side    string
		err     error
	}{
		{name: "known options", asset: asset.Options, symbols: []string{"known", "known"}, side: "Sell"},
		{name: "unknown option first", asset: asset.Options, symbols: []string{"unknown", "known", "known"}, side: "Sell"},
		{name: "unknown option middle", asset: asset.Options, symbols: []string{"known", "unknown", "known"}, side: "Sell"},
		{name: "unknown option last", asset: asset.Options, symbols: []string{"known", "known", "unknown"}, side: "Sell"},
		{name: "only unknown options", asset: asset.Options, symbols: []string{"unknown"}, side: "Sell"},
		{name: "empty options batch", asset: asset.Options, side: "Sell"},
		{name: "invalid option side", asset: asset.Options, symbols: []string{"known"}, side: "invalid", err: order.ErrSideIsInvalid},
		{name: "unknown spot remains an error", asset: asset.Spot, symbols: []string{"known", "unknown"}, side: "Sell", err: currency.ErrPairNotFound},
		{name: "unknown futures remains an error", asset: asset.USDTMarginedFutures, symbols: []string{"unknown", "known"}, side: "Sell", err: currency.ErrPairNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := testInstance()
			pairs, err := ex.GetEnabledPairs(tc.asset)
			require.NoError(t, err, "GetEnabledPairs must not error")
			require.NotEmpty(t, pairs, "enabled pairs must contain a known symbol")
			entries := make([]string, 0, len(tc.symbols))
			expected := make([]trade.Data, 0, len(tc.symbols))
			for i, symbol := range tc.symbols {
				id := fmt.Sprintf("trade-%d", i)
				if symbol == "known" {
					symbol = pairs[0].String()
					expected = append(expected, trade.Data{
						Timestamp: time.UnixMilli(1690720953111), CurrencyPair: pairs[0], AssetType: tc.asset,
						Exchange: ex.Name, Price: 3.6279, Amount: 1.3637, Side: order.Sell, TID: id,
					})
				} else {
					symbol = "BTC-10OCT26-99000-C-USDT"
				}
				entries = append(entries, fmt.Sprintf(`{"i":%q,"T":1690720953111,"p":"3.6279","v":"1.3637","S":%q,"s":%q,"BT":false}`, id, tc.side, symbol))
			}
			var result WebsocketPublicTrades
			require.NoError(t, json.Unmarshal([]byte("["+strings.Join(entries, ",")+"]"), &result), "trade fixture must decode")
			actual, err := ex.convertPublicTrades(tc.asset, result)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "conversion should retain strict errors outside unknown options")
				assert.Nil(t, actual, "invalid batches should not return converted trades")
				return
			}
			require.NoError(t, err, "conversion must tolerate newly listed option contracts")
			assert.Equal(t, expected, actual, "conversion should retain every field and the order of known trades")
		})
	}
}

func TestWsProcessPublicTrade(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "unknown option contracts", data: `[{"s":"BTC-10OCT26-99000-C-USDT","S":"Sell"}]`},
		{name: "empty batch", data: `[]`},
		{name: "malformed payload", data: `{`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := testInstance().wsProcessPublicTrade(asset.Options, &WebsocketResponse{Data: json.RawMessage(tc.data)})
			if tc.name == "malformed payload" {
				var syntaxError *json.SyntaxError
				assert.ErrorAs(t, err, &syntaxError, "malformed payload should return its decode error")
				return
			}
			assert.NoError(t, err, "option batches containing no known contracts should not error")
		})
	}
}
