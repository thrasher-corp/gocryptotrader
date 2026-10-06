package okx

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestMessageID(t *testing.T) {
	t.Parallel()
	id := new(Exchange).MessageID()
	require.Len(t, id, 32, "Must return the correct length of message id")
	u, err := uuid.Parse(id)
	require.NoError(t, err, "MessageID must return a valid UUID")
	require.Equal(t, byte(7), u[6]>>4, "MessageID must return a V7 uuid") // RFC 9562 version nibble
	require.Len(t, u.String(), 36, "UUID v7 string representation must be 36 characters long")
}

// 7696807	       153.1 ns/op	      48 B/op	       2 allocs/op
func BenchmarkMessageID(b *testing.B) {
	e := new(Exchange)
	for b.Loop() {
		_ = e.MessageID()
	}
}

// TestGetOpenInterestSingleOptionsKey guards the options branch of the
// single-key open interest query: OKX rejects a family that doesn't match the
// instrument with 51002, so only the instrument's own family is queried, with
// the instrument pinned by ID.
func TestGetOpenInterestSingleOptionsKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		instID    string
		tickBands string
		family    string
		err       error
	}{
		{
			name:      "plain family option, _UM family listed first",
			instID:    "BTC-USD-260928-79000-C",
			tickBands: `[{"instType":"OPTION","instFamily":"BTC-USD_UM","tickBand":[]},{"instType":"OPTION","instFamily":"BTC-USD","tickBand":[]}]`,
			family:    "BTC-USD",
		},
		{
			name:      "_UM option, plain family listed first",
			instID:    "BTC-USD_UM-260928-79000-C",
			tickBands: `[{"instType":"OPTION","instFamily":"BTC-USD","tickBand":[]},{"instType":"OPTION","instFamily":"BTC-USD_UM","tickBand":[]}]`,
			family:    "BTC-USD_UM",
		},
		{
			name:      "family missing from the tick bands",
			instID:    "ETH-USD-260928-2000-C",
			tickBands: `[{"instType":"OPTION","instFamily":"BTC-USD","tickBand":[]},{"instType":"OPTION","instFamily":"BTC-USD_UM","tickBand":[]}]`,
			err:       errInstrumentFamilyRequired,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := new(Exchange)
			require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

			var mu sync.Mutex
			var paths []string
			srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/public/instrument-tick-bands":
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":` + tc.tickBands + `}`))
				case "/public/open-interest":
					if r.URL.Query().Get("instFamily") != tc.family {
						_, _ = w.Write([]byte(`{"code":"51002","msg":"Instrument ID doesn't match underlying index.","data":[]}`))
						return
					}
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"` + tc.instID + `","instType":"OPTION","oi":"100","oiCcy":"1","oiUsd":"10","ts":"1597026383085"}]}`))
				default:
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
				}
			}))
			b := e.GetBase()
			b.SkipAuthCheck = true
			require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
			for k := range b.API.Endpoints.GetURLMap() {
				require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
			}

			optionInstID, err := currency.NewPairFromString(tc.instID)
			require.NoError(t, err, "option instrument ID must parse")
			sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, e, asset.Options, optionInstID)

			resp, err := e.GetOpenInterest(t.Context(), key.PairAsset{
				Base:  optionInstID.Base.Item,
				Quote: optionInstID.Quote.Item,
				Asset: asset.Options,
			})
			mu.Lock()
			defer mu.Unlock()
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "GetOpenInterest must reject an option whose family isn't listed")
				assert.Equal(t, []string{"/public/instrument-tick-bands?instType=OPTION"}, paths, "no open interest query should be made without the family")
				return
			}
			require.NoError(t, err, "GetOpenInterest with an Options key must not error")
			require.Len(t, resp, 1, "GetOpenInterest with an Options key must return the requested instrument entry")
			assert.Equal(t, 100.0, resp[0].OpenInterest, "the entry should carry the mocked open interest")
			assert.Truef(t, resp[0].Key.Pair().Equal(optionInstID), "the entry should resolve to the requested pair, got %v", resp[0].Key.Pair())
			assert.Equal(t, []string{
				"/public/instrument-tick-bands?instType=OPTION",
				"/public/open-interest?instFamily=" + tc.family + "&instId=" + tc.instID + "&instType=OPTION",
			}, paths, "only the instrument's own family should be queried, with the instrument pinned by ID")
		})
	}
}
