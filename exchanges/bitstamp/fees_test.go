package bitstamp

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

func TestGetTradingFees(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	fees, err := e.GetTradingFees(t.Context())
	require.NoError(t, err, "GetTradingFees must not error")
	require.NotEmpty(t, fees, "GetTradingFees must return fees")
	if mockTests {
		exp := []TradingFeeResponse{
			{Market: "btcusd", Fees: MakerTakerFees{Maker: 0.3, Taker: 0.4}},
			{Market: "ltcbtc", Fees: MakerTakerFees{Maker: 0.3, Taker: 0.4}},
		}
		assert.Equal(t, exp, fees, "GetTradingFees should decode the fees")
	}
	for _, f := range fees {
		assert.NotEmpty(t, f.Market, "Market should be set")
		assert.GreaterOrEqual(t, f.Fees.Taker.Float64(), f.Fees.Maker.Float64(), "Taker fee should not be below the maker fee")
	}

	_, err = e.GetTradingFee(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTradingFee should error on an empty pair")
	fee, err := e.GetTradingFee(t.Context(), spotPair)
	require.NoError(t, err, "GetTradingFee must not error")
	if mockTests {
		assert.Equal(t, &TradingFeeResponse{Market: "btcusd", Fees: MakerTakerFees{Maker: 0.3, Taker: 0.4}}, fee, "GetTradingFee should decode the fee")
		return
	}
	assert.Equal(t, "btcusd", fee.Market, "Market should be correct")
}

func TestGetTradingFee(t *testing.T) {
	t.Parallel()
	t.Run("from a fee builder", func(t *testing.T) {
		t.Parallel()
		_, err := e.getTradingFee(t.Context(), &exchange.FeeBuilder{})
		assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "getTradingFee should error without a pair")

		skipIfLiveWithoutCredentials(t)
		ctx := t.Context()
		for _, tc := range []struct {
			name    string
			isMaker bool
			exp     float64
		}{
			{name: "taker", isMaker: false, exp: 0.4 / 100 * 1800 * 5},
			{name: "maker", isMaker: true, exp: 0.3 / 100 * 1800 * 5},
		} {
			// Serial and in order, as the fees are fetched through the same exchange
			t.Run(tc.name, func(t *testing.T) {
				isMaker, exp := tc.isMaker, tc.exp
				feeBuilder := setFeeBuilder()
				feeBuilder.IsMaker = isMaker
				fee, err := e.getTradingFee(ctx, feeBuilder)
				require.NoErrorf(t, err, "getTradingFee must not error for maker %t", isMaker)
				if mockTests {
					assert.InDeltaf(t, exp, fee, 1e-9, "getTradingFee should apply the fee percentage for maker %t", isMaker)
				}
			})
		}
	})

	for _, tc := range []struct {
		name string
		pair currency.Pair
		path string
		body string
		exp  *TradingFeeResponse
		err  error
	}{
		{name: "empty pair", pair: currency.EMPTYPAIR, err: currency.ErrCurrencyPairEmpty},
		{
			name: "spot pair",
			pair: spotPair,
			path: "/api/v2/fees/trading/btcusd/",
			body: `{"currency_pair":"btcusd","fees":{"maker":"0.30000","taker":"0.40000"},"market":"btcusd"}`,
			exp:  &TradingFeeResponse{Market: "btcusd", Fees: MakerTakerFees{Maker: 0.3, Taker: 0.4}},
		},
		{
			name: "perpetual pair",
			pair: perpetualPair,
			path: "/api/v2/fees/trading/btcusd-perp/",
			body: `{"currency_pair":"btcusd-perp","fees":{"maker":"0.02000","taker":"0.05000"},"market":"btcusd-perp"}`,
			exp:  &TradingFeeResponse{Market: "btcusd-perp", Fees: MakerTakerFees{Maker: 0.02, Taker: 0.05}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.exp == nil {
					assert.Failf(t, "unexpected request", "GetTradingFee should not request %s", r.URL.Path)
					http.Error(w, "unexpected request", http.StatusNotFound)
					return
				}
				assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
				assert.Equal(t, tc.path, r.URL.Path, "path should be the market's trading fee")
				_, err := w.Write([]byte(tc.body))
				assert.NoError(t, err, "Write should not error")
			})
			fee, err := ex.GetTradingFee(t.Context(), tc.pair)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "GetTradingFee should return the correct error")
				assert.Nil(t, fee, "GetTradingFee should not return a fee with an error")
				return
			}
			require.NoError(t, err, "GetTradingFee must not error")
			assert.Equal(t, tc.exp, fee, "GetTradingFee should decode the market's fees")
		})
	}
}

func TestGetWithdrawalFees(t *testing.T) { //nolint:tparallel // The networks' cases go through the package exchange the calls above use, so they run in order
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	fees, err := e.GetWithdrawalFees(t.Context())
	require.NoError(t, err, "GetWithdrawalFees must not error")
	require.NotEmpty(t, fees, "GetWithdrawalFees must return fees")
	if mockTests {
		// The API sends lower case currency codes
		exp := []WithdrawalFeeResponse{
			{Currency: currency.NewCode("btc"), Fee: 0.00015, Network: "bitcoin"},
			{Currency: currency.NewCode("eth"), Fee: 0.001, Network: "ethereum"},
		}
		assert.Equal(t, exp, fees, "GetWithdrawalFees should decode the fees")
	}

	_, err = e.GetWithdrawalFee(t.Context(), currency.EMPTYCODE, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetWithdrawalFee should error on an empty currency")
	ctx := t.Context()
	for _, tc := range []struct {
		name    string
		network string
	}{
		{name: "default network", network: ""},
		{name: "lightning network", network: "lightning"},
	} {
		// Serial and in order, as the fees are fetched through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			network := tc.network
			fee, err := e.GetWithdrawalFee(ctx, currency.BTC, network)
			require.NoErrorf(t, err, "GetWithdrawalFee must not error for network %q", network)
			assert.True(t, fee.Currency.Equal(currency.BTC), "Currency should be correct")
			if mockTests {
				exp := map[string]WithdrawalFeeResponse{
					"":          {Currency: currency.NewCode("btc"), Fee: 0.00015, Network: "bitcoin"},
					"lightning": {Currency: currency.NewCode("btc"), Fee: 0.00001, Network: "lightning"},
				}[network]
				assert.Equalf(t, exp, *fee, "GetWithdrawalFee should return the fee for network %q", network)
			}
		})
	}
}

func TestGetWithdrawalFee(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		c       currency.Code
		network string
		form    url.Values
		body    string
		exp     *WithdrawalFeeResponse
		err     error
	}{
		{name: "empty currency", c: currency.EMPTYCODE, err: currency.ErrCurrencyCodeEmpty},
		{
			name: "default network",
			c:    currency.BTC,
			form: url.Values{},
			body: `{"currency":"btc","fee":"0.00015000","network":"bitcoin"}`,
			exp:  &WithdrawalFeeResponse{Currency: currency.NewCode("btc"), Fee: 0.00015, Network: "bitcoin"},
		},
		{
			name:    "lightning network",
			c:       currency.BTC,
			network: "lightning",
			form:    url.Values{"network": {"lightning"}},
			body:    `{"currency":"btc","fee":"0.00001000","network":"lightning"}`,
			exp:     &WithdrawalFeeResponse{Currency: currency.NewCode("btc"), Fee: 0.00001, Network: "lightning"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.exp == nil {
					assert.Failf(t, "unexpected request", "GetWithdrawalFee should not request %s", r.URL.Path)
					http.Error(w, "unexpected request", http.StatusNotFound)
					return
				}
				assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
				assert.Equal(t, "/api/v2/fees/withdrawal/btc/", r.URL.Path, "path should be the lower case currency's withdrawal fee")
				assert.NoError(t, r.ParseForm(), "ParseForm should not error")
				assert.Equal(t, tc.form, r.PostForm, "GetWithdrawalFee should send the network only when one is set")
				_, err := w.Write([]byte(tc.body))
				assert.NoError(t, err, "Write should not error")
			})
			fee, err := ex.GetWithdrawalFee(t.Context(), tc.c, tc.network)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "GetWithdrawalFee should return the correct error")
				assert.Nil(t, fee, "GetWithdrawalFee should not return a fee with an error")
				return
			}
			require.NoError(t, err, "GetWithdrawalFee must not error")
			assert.Equal(t, tc.exp, fee, "GetWithdrawalFee should decode the network's fee")
		})
	}
}
