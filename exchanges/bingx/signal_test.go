package bingx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestCreateSignal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CreateSignalRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing symbol", arg: &CreateSignalRequest{Side: SignalSideLong, EntryPrice: 65000}, err: currency.ErrCurrencyPairEmpty},
		{name: "missing side", arg: &CreateSignalRequest{Symbol: currency.NewBTCUSDT(), EntryPrice: 65000}, err: errSignalSideRequired},
		{name: "missing entry price", arg: &CreateSignalRequest{Symbol: currency.NewBTCUSDT(), Side: SignalSideLong}, err: errSignalEntryPriceRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CreateSignal(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "CreateSignal should reject the request")
		})
	}
	t.Run("JSON body", func(t *testing.T) {
		t.Parallel()
		var received map[string]json.RawMessage
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
			assert.Equal(t, "/openApi/social/v1/swap/signal", r.URL.Path, "path should be the create signal path")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "content type should be JSON")
			assert.Empty(t, r.URL.RawQuery, "query should be empty")
			raw, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the body should not error")
			assert.NoError(t, json.Unmarshal(raw, &received), "the body should be a JSON object")
			_, _ = io.WriteString(w, `{"code":0,"timestamp":1786690121277,"data":{"signalId":"a","shortUrl":"b"}}`)
		})
		arg := &CreateSignalRequest{
			Symbol:          currency.NewBTCUSDT(),
			Side:            SignalSideLong,
			EntryPrice:      65000,
			TakeProfitPrice: 68000,
			StopLossPrice:   62000.5,
			OrderType:       OrderTypeLimit,
			Leverage:        20,
			SizePercentage:  10,
		}
		result, err := ex.CreateSignal(t.Context(), arg)
		require.NoError(t, err, "CreateSignal must not error")
		assert.Equal(t, &CreateSignalResponse{SignalID: "a", ShortURL: "b"}, result, "CreateSignal should decode the reply")
		assert.Equal(t, currency.NewBTCUSDT(), arg.Symbol, "CreateSignal should leave the caller's symbol unformatted")
		require.NotNil(t, received, "a body must be received")
		for k, v := range map[string]string{
			"symbol": `"BTC-USDT"`, "side": `"long"`, "entryPrice": `"65000"`, "tpPrice": `"68000"`, "slPrice": `"62000.5"`,
			"orderType": `"LIMIT"`, "leverage": `20`, "sizePct": `10`, "recvWindow": `5000`,
		} {
			assert.JSONEqf(t, v, string(received[k]), "%s should be sent as documented", k)
		}
		require.Contains(t, received, "timestamp", "timestamp must be sent")
		var signature string
		require.NoError(t, json.Unmarshal(received["signature"], &signature), "signature must be a string")
		expected := "entryPrice=65000&leverage=20&orderType=LIMIT&recvWindow=5000&side=long&sizePct=10&slPrice=62000.5&symbol=BTC-USDT&timestamp=" + string(received["timestamp"]) + "&tpPrice=68000"
		mac := hmac.New(sha256.New, []byte(testAPISecret))
		mac.Write([]byte(expected))
		assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), signature, "signature should cover the sorted members with strings unquoted")
	})
	t.Run("optional fields left out", func(t *testing.T) {
		t.Parallel()
		var received map[string]json.RawMessage
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			raw, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the body should not error")
			assert.NoError(t, json.Unmarshal(raw, &received), "the body should be a JSON object")
			_, _ = io.WriteString(w, `{"code":0,"data":{"signalId":"a","shortUrl":"b"}}`)
		})
		_, err := ex.CreateSignal(t.Context(), &CreateSignalRequest{Symbol: currency.NewBTCUSDT(), Side: SignalSideShort, EntryPrice: 0.00000005})
		require.NoError(t, err, "CreateSignal must not error")
		assert.Equal(t, []string{"entryPrice", "recvWindow", "side", "signature", "symbol", "timestamp"}, slices.Sorted(maps.Keys(received)), "only the required fields and the signing fields should be sent")
		assert.JSONEq(t, `"0.00000005"`, string(received["entryPrice"]), "a small price should be sent in decimal notation")
	})
	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).CreateSignal(t.Context(), &CreateSignalRequest{Symbol: currency.NewBTCUSDT(), Side: SignalSideLong, EntryPrice: 65000})
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "CreateSignal should return the symbol formatting error")
	})
	t.Run("request error", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":100500,"msg":"busy"}`)
		})
		result, err := ex.CreateSignal(t.Context(), &CreateSignalRequest{Symbol: currency.NewBTCUSDT(), Side: SignalSideLong, EntryPrice: 65000})
		assert.ErrorIs(t, err, errAPIResponse, "CreateSignal should return the request's error")
		assert.Nil(t, result, "CreateSignal should return no signal for a failed request")
	})
	t.Run("empty signal ID", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":0,"data":{"signalId":"","shortUrl":""}}`)
		})
		_, err := ex.CreateSignal(t.Context(), &CreateSignalRequest{Symbol: currency.NewBTCUSDT(), Side: SignalSideLong, EntryPrice: 65000})
		assert.ErrorIs(t, err, errSignalIDEmpty, "CreateSignal should reject a reply without a signal ID")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.CreateSignal(t.Context(), &CreateSignalRequest{
			Symbol:          currency.NewBTCUSDT(),
			Side:            SignalSideLong,
			EntryPrice:      65000,
			TakeProfitPrice: 68000,
			StopLossPrice:   62000,
			OrderType:       OrderTypeLimit,
			Leverage:        20,
			SizePercentage:  10,
		})
		require.NoError(t, err, "CreateSignal must not error")
		if !mockTests {
			return
		}
		exp := &CreateSignalResponse{
			SignalID: "decabcb948154187be32d412e9ce0963",
			ShortURL: "https://bingx.com/signalclone/0H4Gzz",
		}
		assert.Equal(t, exp, result, "CreateSignal should decode the documented example")
	})
}
