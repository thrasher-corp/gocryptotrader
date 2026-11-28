package binanceus

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestConvertDust(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		fromAssets []currency.Code
		toAsset    currency.Code
		param      string
	}{
		{name: "no from asset", toAsset: currency.BNB, param: "fromAsset"},
		{name: "empty from asset", fromAssets: []currency.Code{currency.BTC, currency.EMPTYCODE}, toAsset: currency.BNB, param: "fromAsset 1"},
		{name: "no to asset", fromAssets: []currency.Code{currency.BTC}, param: "toAsset"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ConvertDust(t.Context(), tc.fromAssets, tc.toAsset)
			require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ConvertDust must reject a missing asset")
			assert.ErrorContains(t, err, tc.param, "ConvertDust should name the missing asset")
		})
	}

	t.Run("sends each asset to convert as a fromAsset", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/asset/dust", url.Values{"fromAsset": {"BTC", "ETH"}, "toAsset": {"BNB"}}, `{}`)
		_, err := ex.ConvertDust(t.Context(), []currency.Code{currency.NewCode("btc"), currency.ETH}, currency.NewCode("bnb"))
		require.NoError(t, err, "ConvertDust must not error")
	})

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live dust conversion")
	}
	skipIfCredentialsUnset(t)
	result, err := e.ConvertDust(t.Context(), []currency.Code{currency.LTC}, currency.BTC)
	require.NoError(t, err, "ConvertDust must not error")
	require.NotNil(t, result, "ConvertDust must return the conversion")
	if mockTests {
		exp := &ConvertDustResponse{
			TotalTransferred:   0.00289855,
			TotalServiceCharge: 0.00005797,
			TransferResult: []DustAssetConversion{{
				TransactionID:       28822799,
				FromAsset:           currency.LTC,
				ToAsset:             currency.BTC,
				Amount:              0.2,
				TransferredAmount:   0.00289855,
				ServiceChargeAmount: 0.00005797,
				OperateTime:         types.Time(time.UnixMilli(1659583487273)),
				Result:              "S",
			}},
		}
		assert.Equal(t, exp, result, "ConvertDust should decode every field")
	}
}

func TestGetConvertDustHistory(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1659583000000)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		err        error
		param      string
	}{
		{name: "no start time", end: start, err: common.ErrDateUnset, param: "startTime"},
		{name: "no end time", start: start, err: common.ErrDateUnset, param: "endTime"},
		{name: "start after end", start: start, end: start.Add(-time.Second), err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetConvertDustHistory(t.Context(), tc.start, tc.end)
			require.ErrorIs(t, err, tc.err, "GetConvertDustHistory must reject an invalid time range")
			if tc.param != "" {
				assert.ErrorContains(t, err, tc.param, "GetConvertDustHistory should name the missing time")
			}
		})
	}

	t.Run("sends the time range in milliseconds", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/asset/query/dust-logs", url.Values{"startTime": {"1659583000000"}, "endTime": {"1659584000000"}}, `{}`)
		_, err := ex.GetConvertDustHistory(t.Context(), start, start.Add(1000*time.Second))
		require.NoError(t, err, "GetConvertDustHistory must not error")
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetConvertDustHistory(t.Context(), start, start.Add(1000*time.Second))
	require.NoError(t, err, "GetConvertDustHistory must not error")
	require.NotNil(t, result, "GetConvertDustHistory must return the conversions")
	if mockTests {
		exp := &ConvertDustHistoryResponse{
			Total: 1,
			UserDustConvertHistory: []DustConversion{{
				OperateTime:              types.Time(time.UnixMilli(1659583487000)),
				TotalServiceChargeAmount: 0.00005797,
				TotalTransferredAmount:   0.00284058,
				UserAssetDribbletDetails: []DustAssetConversion{{
					TransactionID:       28822799,
					FromAsset:           currency.LTC,
					ToAsset:             currency.BTC,
					Amount:              0.2,
					TransferredAmount:   0.00284058,
					ServiceChargeAmount: 0.00005797,
					OperateTime:         types.Time(time.UnixMilli(1659583487000)),
				}},
			}},
		}
		assert.Equal(t, exp, result, "GetConvertDustHistory should decode every field")
	}
}

func TestGetAssetsThatCanBeConverted(t *testing.T) {
	t.Parallel()
	_, err := e.GetAssetsThatCanBeConverted(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAssetsThatCanBeConverted must reject an empty asset")

	t.Run("sends the asset to convert to", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/asset/query/dust-assets", url.Values{"toAsset": {"BNB"}}, `{}`)
		_, err := ex.GetAssetsThatCanBeConverted(t.Context(), currency.NewCode("bnb"))
		require.NoError(t, err, "GetAssetsThatCanBeConverted must not error")
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetAssetsThatCanBeConverted(t.Context(), currency.BNB)
	require.NoError(t, err, "GetAssetsThatCanBeConverted must not error")
	require.NotNil(t, result, "GetAssetsThatCanBeConverted must return the convertible assets")
	if mockTests {
		exp := &ConvertibleAssetsResponse{
			ConvertibleAssets: []ConvertibleAsset{
				{FromAsset: currency.BTC, ToAsset: currency.BNB, AvailableBalance: 0.0001, ConvertedAsset: 0.0028987, USDValueConvertedAsset: 2.0001, ConversionFee: 0.00005797, ReceivedAsset: 0.00284073},
				{FromAsset: currency.USDT, ToAsset: currency.BNB, AvailableBalance: 14.286, ConvertedAsset: 0.02029026, USDValueConvertedAsset: 14.00028, ConversionFee: 0.00040581, ReceivedAsset: 0.01988445},
			},
			TotalConvertedAsset:         0.02318896,
			USDValueTotalConvertedAsset: 16.00038,
			TotalConversionFee:          0.00046378,
			TotalReceivedAsset:          0.02272518,
			FeeRate:                     0.02,
			WithinRestrictedTime:        true,
		}
		assert.Equal(t, exp, result, "GetAssetsThatCanBeConverted should decode every field")
	}
}
