package okx

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSupportCoins(t *testing.T) {
	t.Parallel()
	result, err := e.GetSupportCoins(contextGenerate())
	require.NoError(t, err, "GetSupportCoins must not error")
	require.NotNil(t, result, "GetSupportCoins must return the currencies")
	if mockTests {
		exp := &SupportCoinResponse{
			ContractCurrencies: []currency.Code{currency.BTC, currency.ETH, currency.OKB},
			OptionCurrencies:   []currency.Code{currency.BTC, currency.ETH, currency.SOL},
			OptionV2Currencies: []currency.Code{},
			SpotCurrencies:     []currency.Code{currency.BTC, currency.ETH, currency.OKB},
		}
		assert.Equal(t, exp, result, "GetSupportCoins should decode every field")
	} else {
		assert.NotEmpty(t, result.SpotCurrencies, "GetSupportCoins should return spot currencies")
	}
}

func TestGetContractOpenInterestHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *ContractStatisticsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument ID", arg: &ContractStatisticsRequest{Period: kline.FiveMin, Limit: 10}, err: errMissingInstrumentID},
		{name: "begin after end", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Begin: time.UnixMilli(1795363200000), End: time.UnixMilli(1795276800000)}, err: common.ErrStartAfterEnd},
		{name: "unsupported period", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.TenMin}, err: kline.ErrUnsupportedInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetContractOpenInterestHistory(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetContractOpenInterestHistory should reject an invalid request")
		})
	}

	result, err := e.GetContractOpenInterestHistory(contextGenerate(), &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.FiveMin, Limit: 10})
	require.NoError(t, err, "GetContractOpenInterestHistory must not error")
	require.NotEmpty(t, result, "GetContractOpenInterestHistory must return open interest")
	if mockTests {
		exp := []ContractOpenInterestHistoryResponse{
			{
				Timestamp:             types.Time(time.UnixMilli(1795406700000)),
				OpenInterestContracts: 3129207.79000001285,
				OpenInterestCurrency:  31292.0779000001285,
				OpenInterestUSD:       1917096635.7123478725011,
			},
			{
				Timestamp:             types.Time(time.UnixMilli(1795406400000)),
				OpenInterestContracts: 3129207.79000001285,
				OpenInterestCurrency:  31292.0779000001285,
				OpenInterestUSD:       1917096635.7123478725011,
			},
		}
		assert.Equal(t, exp, result[:2], "GetContractOpenInterestHistory should decode every field")
	}
}

func TestGetTakerVolume(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *TakerVolumeRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument type", arg: &TakerVolumeRequest{Currency: currency.BTC, Period: kline.OneDay}, err: errInvalidInstrumentType},
		{name: "missing currency", arg: &TakerVolumeRequest{InstrumentType: instTypeSpot, Period: kline.OneDay}, err: currency.ErrCurrencyCodeEmpty},
		{name: "begin after end", arg: &TakerVolumeRequest{Currency: currency.BTC, InstrumentType: instTypeSpot, Begin: time.UnixMilli(1795363200000), End: time.UnixMilli(1795276800000)}, err: common.ErrStartAfterEnd},
		{name: "unsupported period", arg: &TakerVolumeRequest{Currency: currency.BTC, InstrumentType: instTypeSpot, Period: kline.TenMin}, err: kline.ErrUnsupportedInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetTakerVolume(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetTakerVolume should reject an invalid request")
		})
	}

	result, err := e.GetTakerVolume(contextGenerate(), &TakerVolumeRequest{Currency: currency.BTC, InstrumentType: instTypeSpot, Period: kline.OneDay})
	require.NoError(t, err, "GetTakerVolume must not error")
	if mockTests {
		exp := []TakerVolumeResponse{
			{
				Timestamp:  types.Time(time.UnixMilli(1795363200000)),
				SellVolume: 3881.4043,
				BuyVolume:  4035.2973,
			},
			{
				Timestamp:  types.Time(time.UnixMilli(1795276800000)),
				SellVolume: 17633.5989,
				BuyVolume:  16308.4215,
			},
			{
				Timestamp:  types.Time(time.UnixMilli(1795190400000)),
				SellVolume: 8406.0432,
				BuyVolume:  7121.9827,
			},
		}
		assert.Equal(t, exp, result, "GetTakerVolume should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetTakerVolume should return volumes")
	}
}

func TestGetContractTakerVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractTakerVolume(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetContractTakerVolume must reject a nil request")
	_, err = e.GetContractTakerVolume(t.Context(), &ContractTakerVolumeRequest{Period: kline.FiveMin, Unit: "1", Limit: 10})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetContractTakerVolume must require an instrument ID")

	result, err := e.GetContractTakerVolume(contextGenerate(), &ContractTakerVolumeRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.FiveMin, Unit: "1", Limit: 10})
	require.NoError(t, err, "GetContractTakerVolume must not error")
	require.NotEmpty(t, result, "GetContractTakerVolume must return volumes")
	if mockTests {
		exp := []TakerVolumeResponse{
			{
				Timestamp:  types.Time(time.UnixMilli(1795406700000)),
				SellVolume: 366.189999999999998,
				BuyVolume:  1585.71000000000004,
			},
			{
				Timestamp:  types.Time(time.UnixMilli(1795406400000)),
				SellVolume: 35846.3199999999997,
				BuyVolume:  17285.8400000000001,
			},
		}
		assert.Equal(t, exp, result[:2], "GetContractTakerVolume should decode every field")
	}
}

func TestGetMarginLendingRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginLendingRatio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginLendingRatio must reject a nil request")
	_, err = e.GetMarginLendingRatio(t.Context(), &CurrencyStatisticsRequest{Period: kline.FiveMin})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMarginLendingRatio must require a currency")

	result, err := e.GetMarginLendingRatio(contextGenerate(), &CurrencyStatisticsRequest{Currency: currency.BTC, Period: kline.FiveMin})
	require.NoError(t, err, "GetMarginLendingRatio must not error")
	if mockTests {
		exp := []MarginLendingRatioResponse{
			{
				Timestamp: types.Time(time.UnixMilli(1795406700000)),
				Ratio:     43.0442,
			},
			{
				Timestamp: types.Time(time.UnixMilli(1795406400000)),
				Ratio:     43.0287,
			},
		}
		assert.Equal(t, exp, result, "GetMarginLendingRatio should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetMarginLendingRatio should return ratios")
	}
}

func TestGetTopTradersContractLongShortAccountRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetTopTradersContractLongShortAccountRatio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTopTradersContractLongShortAccountRatio must reject a nil request")
	_, err = e.GetTopTradersContractLongShortAccountRatio(t.Context(), &ContractStatisticsRequest{Period: kline.FiveMin, Limit: 10})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetTopTradersContractLongShortAccountRatio must require an instrument ID")

	result, err := e.GetTopTradersContractLongShortAccountRatio(contextGenerate(), &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.FiveMin, Limit: 10})
	require.NoError(t, err, "GetTopTradersContractLongShortAccountRatio must not error")
	require.NotEmpty(t, result, "GetTopTradersContractLongShortAccountRatio must return ratios")
	if mockTests {
		exp := []LongShortRatioResponse{
			{
				Timestamp: types.Time(time.UnixMilli(1795406700000)),
				Ratio:     1.6619987269255251,
			},
			{
				Timestamp: types.Time(time.UnixMilli(1795406400000)),
				Ratio:     1.6579783852511125,
			},
		}
		assert.Equal(t, exp, result[:2], "GetTopTradersContractLongShortAccountRatio should decode every field")
	}
}

func TestGetTopTradersContractLongShortPositionRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetTopTradersContractLongShortPositionRatio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTopTradersContractLongShortPositionRatio must reject a nil request")
	_, err = e.GetTopTradersContractLongShortPositionRatio(t.Context(), &ContractStatisticsRequest{Period: kline.FiveMin, Limit: 10})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetTopTradersContractLongShortPositionRatio must require an instrument ID")

	result, err := e.GetTopTradersContractLongShortPositionRatio(contextGenerate(), &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.FiveMin, Limit: 10})
	require.NoError(t, err, "GetTopTradersContractLongShortPositionRatio must not error")
	require.NotEmpty(t, result, "GetTopTradersContractLongShortPositionRatio must return ratios")
	if mockTests {
		exp := []LongShortRatioResponse{
			{
				Timestamp: types.Time(time.UnixMilli(1795406700000)),
				Ratio:     0.9329357359209828,
			},
			{
				Timestamp: types.Time(time.UnixMilli(1795406400000)),
				Ratio:     0.9322339710613383,
			},
		}
		assert.Equal(t, exp, result[:2], "GetTopTradersContractLongShortPositionRatio should decode every field")
	}
}

func TestGetContractLongShortAccountRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractLongShortAccountRatio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetContractLongShortAccountRatio must reject a nil request")
	_, err = e.GetContractLongShortAccountRatio(t.Context(), &ContractStatisticsRequest{Period: kline.FiveMin, Limit: 10})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetContractLongShortAccountRatio must require an instrument ID")

	result, err := e.GetContractLongShortAccountRatio(contextGenerate(), &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.FiveMin, Limit: 10})
	require.NoError(t, err, "GetContractLongShortAccountRatio must not error")
	require.NotEmpty(t, result, "GetContractLongShortAccountRatio must return ratios")
	if mockTests {
		exp := []LongShortRatioResponse{
			{
				Timestamp: types.Time(time.UnixMilli(1795406700000)),
				Ratio:     2.0336471842245514,
			},
			{
				Timestamp: types.Time(time.UnixMilli(1795406400000)),
				Ratio:     2.0345244444444444,
			},
		}
		assert.Equal(t, exp, result[:2], "GetContractLongShortAccountRatio should decode every field")
	}
}

func TestGetContractLongShortRatio(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *ContractStatisticsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument ID", arg: &ContractStatisticsRequest{Period: kline.FiveMin}, err: errMissingInstrumentID},
		{name: "unsupported period", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.TenMin}, err: kline.ErrUnsupportedInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.getContractLongShortRatio(t.Context(), "rubik/stat/contracts/long-short-account-ratio-contract", getContractLongShortAccountRatioEPL, tc.arg)
			assert.ErrorIs(t, err, tc.err, "getContractLongShortRatio should reject the request before sending it")
		})
	}

	t.Run("sends the query to the caller's path", func(t *testing.T) {
		t.Parallel()
		ex := newQueryPinExchange(t, "/rubik/stat/contracts/long-short-position-ratio-contract-top-trader",
			url.Values{"instId": {"BTC-USDT-SWAP"}, "period": {"1Dutc"}, "begin": {"1795320000000"}, "end": {"1795406400000"}, "limit": {"2"}},
			[][]string{{"1795406400000", "2.0345244444444444"}, {"1795320000000", "1.98"}})
		result, err := ex.getContractLongShortRatio(t.Context(), "rubik/stat/contracts/long-short-position-ratio-contract-top-trader", getTopTradersContractLongShortPositionRatioEPL, &ContractStatisticsRequest{
			InstrumentID: perpetualSwapPair.String(), Period: kline.OneDay, Begin: time.UnixMilli(1795320000000), End: time.UnixMilli(1795406400000), Limit: 2,
		})
		require.NoError(t, err, "getContractLongShortRatio must not error")
		exp := []LongShortRatioResponse{
			{Timestamp: types.Time(time.UnixMilli(1795406400000)), Ratio: 2.0345244444444444},
			{Timestamp: types.Time(time.UnixMilli(1795320000000)), Ratio: 1.98},
		}
		assert.Equal(t, exp, result, "getContractLongShortRatio should decode every ratio")
	})

	t.Run("contract long short account ratio", func(t *testing.T) {
		t.Parallel()
		result, err := e.getContractLongShortRatio(contextGenerate(), "rubik/stat/contracts/long-short-account-ratio-contract", getContractLongShortAccountRatioEPL, &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.FiveMin, Limit: 10})
		require.NoError(t, err, "getContractLongShortRatio must not error")
		require.NotEmpty(t, result, "getContractLongShortRatio must return ratios")
		if mockTests {
			assert.Equal(t, LongShortRatioResponse{Timestamp: types.Time(time.UnixMilli(1795406700000)), Ratio: 2.0336471842245514}, result[0], "getContractLongShortRatio should decode the latest ratio")
		}
	})
}

func TestGetLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetLongShortRatio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLongShortRatio must reject a nil request")
	_, err = e.GetLongShortRatio(t.Context(), &CurrencyStatisticsRequest{Period: kline.OneDay})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLongShortRatio must require a currency")

	result, err := e.GetLongShortRatio(contextGenerate(), &CurrencyStatisticsRequest{Currency: currency.BTC, Period: kline.OneDay})
	require.NoError(t, err, "GetLongShortRatio must not error")
	if mockTests {
		exp := []LongShortRatioResponse{
			{
				Timestamp: types.Time(time.UnixMilli(1795363200000)),
				Ratio:     2.21,
			},
			{
				Timestamp: types.Time(time.UnixMilli(1795276800000)),
				Ratio:     2.22,
			},
		}
		assert.Equal(t, exp, result, "GetLongShortRatio should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetLongShortRatio should return ratios")
	}
}

func TestGetContractsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractsOpenInterestAndVolume(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetContractsOpenInterestAndVolume must reject a nil request")
	_, err = e.GetContractsOpenInterestAndVolume(t.Context(), &CurrencyStatisticsRequest{Period: kline.OneDay})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetContractsOpenInterestAndVolume must require a currency")

	result, err := e.GetContractsOpenInterestAndVolume(contextGenerate(), &CurrencyStatisticsRequest{Currency: currency.BTC, Period: kline.OneDay})
	require.NoError(t, err, "GetContractsOpenInterestAndVolume must not error")
	if mockTests {
		exp := []OpenInterestVolumeResponse{
			{
				Timestamp:    types.Time(time.UnixMilli(1795363200000)),
				OpenInterest: 2840069640.7749,
				Volume:       6391685269.1852,
			},
			{
				Timestamp:    types.Time(time.UnixMilli(1795276800000)),
				OpenInterest: 2824717923.0351,
				Volume:       18253423455.3751,
			},
			{
				Timestamp:    types.Time(time.UnixMilli(1795190400000)),
				OpenInterest: 2923530284.6432,
				Volume:       13054125106.6203,
			},
		}
		assert.Equal(t, exp, result, "GetContractsOpenInterestAndVolume should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetContractsOpenInterestAndVolume should return open interest and volume")
	}
}

func TestGetOptionsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsOpenInterestAndVolume(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOptionsOpenInterestAndVolume must reject a nil request")
	_, err = e.GetOptionsOpenInterestAndVolume(t.Context(), &OptionStatisticsRequest{Period: kline.OneDay})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetOptionsOpenInterestAndVolume must require a currency")

	result, err := e.GetOptionsOpenInterestAndVolume(contextGenerate(), &OptionStatisticsRequest{Currency: currency.BTC, Period: kline.OneDay})
	require.NoError(t, err, "GetOptionsOpenInterestAndVolume must not error")
	if mockTests {
		exp := []OpenInterestVolumeResponse{
			{
				Timestamp:    types.Time(time.UnixMilli(1795363200000)),
				OpenInterest: 56892.49,
				Volume:       8833.58,
			},
			{
				Timestamp:    types.Time(time.UnixMilli(1795276800000)),
				OpenInterest: 56305.66,
				Volume:       27765.22,
			},
			{
				Timestamp:    types.Time(time.UnixMilli(1795190400000)),
				OpenInterest: 49347.16,
				Volume:       16875.5,
			},
		}
		assert.Equal(t, exp, result, "GetOptionsOpenInterestAndVolume should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetOptionsOpenInterestAndVolume should return open interest and volume")
	}
}

func TestGetPutCallRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetPutCallRatio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPutCallRatio must reject a nil request")
	_, err = e.GetPutCallRatio(t.Context(), &OptionStatisticsRequest{Period: kline.OneDay})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPutCallRatio must require a currency")

	result, err := e.GetPutCallRatio(contextGenerate(), &OptionStatisticsRequest{Currency: currency.BTC, Period: kline.OneDay})
	require.NoError(t, err, "GetPutCallRatio must not error")
	if mockTests {
		exp := []PutCallRatioResponse{
			{
				Timestamp:         types.Time(time.UnixMilli(1795363200000)),
				OpenInterestRatio: 1.0414,
				VolumeRatio:       1.1478,
			},
			{
				Timestamp:         types.Time(time.UnixMilli(1795276800000)),
				OpenInterestRatio: 1.0452,
				VolumeRatio:       0.8758,
			},
			{
				Timestamp:         types.Time(time.UnixMilli(1795190400000)),
				OpenInterestRatio: 1.0405,
				VolumeRatio:       0.9507,
			},
		}
		assert.Equal(t, exp, result, "GetPutCallRatio should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetPutCallRatio should return ratios")
	}
}

func TestGetOpenInterestAndVolumeExpiry(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterestAndVolumeExpiry(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenInterestAndVolumeExpiry must reject a nil request")
	_, err = e.GetOpenInterestAndVolumeExpiry(t.Context(), &OptionStatisticsRequest{Period: kline.OneDay})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetOpenInterestAndVolumeExpiry must require a currency")

	result, err := e.GetOpenInterestAndVolumeExpiry(contextGenerate(), &OptionStatisticsRequest{Currency: currency.BTC, Period: kline.OneDay})
	require.NoError(t, err, "GetOpenInterestAndVolumeExpiry must not error")
	if mockTests {
		exp := []OpenInterestVolumeExpiryResponse{
			{
				Timestamp:        types.Time(time.UnixMilli(1795363200000)),
				ExpiryTime:       time.Date(2026, time.November, 23, 0, 0, 0, 0, time.UTC),
				CallOpenInterest: 2080.13,
				PutOpenInterest:  1522.28,
				CallVolume:       1436.53,
				PutVolume:        1122.86,
			},
			{
				Timestamp:        types.Time(time.UnixMilli(1795363200000)),
				ExpiryTime:       time.Date(2026, time.November, 24, 0, 0, 0, 0, time.UTC),
				CallOpenInterest: 900.02,
				PutOpenInterest:  1030.99,
				CallVolume:       617.59,
				PutVolume:        460.02,
			},
			{
				Timestamp:        types.Time(time.UnixMilli(1795363200000)),
				ExpiryTime:       time.Date(2026, time.November, 25, 0, 0, 0, 0, time.UTC),
				CallOpenInterest: 410.98,
				PutOpenInterest:  244.57,
				CallVolume:       511.89,
				PutVolume:        188.26,
			},
			{
				Timestamp:        types.Time(time.UnixMilli(1795363200000)),
				ExpiryTime:       time.Date(2026, time.November, 26, 0, 0, 0, 0, time.UTC),
				CallOpenInterest: 244.45,
				PutOpenInterest:  29.41,
				CallVolume:       244.85,
				PutVolume:        29.41,
			},
			{
				Timestamp:        types.Time(time.UnixMilli(1795363200000)),
				ExpiryTime:       time.Date(2026, time.November, 27, 0, 0, 0, 0, time.UTC),
				CallOpenInterest: 6507.16,
				PutOpenInterest:  8304.47,
				CallVolume:       579.7,
				PutVolume:        1052.29,
			},
			{
				Timestamp:        types.Time(time.UnixMilli(1795363200000)),
				ExpiryTime:       time.Date(2026, time.December, 4, 0, 0, 0, 0, time.UTC),
				CallOpenInterest: 2898.16,
				PutOpenInterest:  1817.72,
				CallVolume:       274.61,
				PutVolume:        312.8,
			},
		}
		assert.Equal(t, exp, result, "GetOpenInterestAndVolumeExpiry should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetOpenInterestAndVolumeExpiry should return expiries")
	}
}

func TestOpenInterestVolumeExpiryResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var got OpenInterestVolumeExpiryResponse
	require.NoError(t, json.Unmarshal([]byte(`["1791216000000","20261006","1275.95","1866.29","863.66","1685.48"]`), &got), "Unmarshal must not error")
	exp := OpenInterestVolumeExpiryResponse{
		Timestamp:        types.Time(time.UnixMilli(1791216000000)),
		ExpiryTime:       time.Date(2026, time.October, 6, 0, 0, 0, 0, time.UTC),
		CallOpenInterest: 1275.95,
		PutOpenInterest:  1866.29,
		CallVolume:       863.66,
		PutVolume:        1685.48,
	}
	assert.Equal(t, exp, got, "Unmarshal should decode every field")

	var noExpiry OpenInterestVolumeExpiryResponse
	require.NoError(t, json.Unmarshal([]byte(`["1791216000000","","0","0","0","0"]`), &noExpiry), "Unmarshal must not error without an expiry date")
	assert.Zero(t, noExpiry.ExpiryTime, "ExpiryTime should stay zero without an expiry date")

	kept := OpenInterestVolumeExpiryResponse{ExpiryTime: exp.ExpiryTime}
	var parseErr *time.ParseError
	err := json.Unmarshal([]byte(`["1791216000000","2026ab06","0","0","0","0"]`), &kept)
	assert.ErrorAs(t, err, &parseErr, "Unmarshal should reject a malformed expiry date")
	assert.Equal(t, exp.ExpiryTime, kept.ExpiryTime, "Unmarshal should keep ExpiryTime when the expiry date is malformed")

	err = json.Unmarshal([]byte(`["1791216000000",20261006,"0","0","0","0"]`), new(OpenInterestVolumeExpiryResponse))
	assert.Error(t, err, "Unmarshal should reject an expiry date that is not a string")
}

func TestGetOpenInterestAndVolumeStrike(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *OpenInterestVolumeStrikeRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing expiry", arg: &OpenInterestVolumeStrikeRequest{Currency: currency.BTC, Period: kline.OneDay}, err: errMissingExpiryTimeParameter},
		{name: "missing currency", arg: &OpenInterestVolumeStrikeRequest{ExpiryTime: time.Date(2026, time.November, 27, 8, 0, 0, 0, time.UTC), Period: kline.OneDay}, err: currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetOpenInterestAndVolumeStrike(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetOpenInterestAndVolumeStrike should reject an invalid request")
		})
	}

	t.Run("sends the expiry's UTC date", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rubik/stat/option/open-interest-volume-strike", r.URL.Path, "request path should be the strike endpoint")
			assert.Equal(t, url.Values{"ccy": {"BTC"}, "expTime": {"20261006"}, "period": {"1D"}}, r.URL.Query(), "query should send the expiry's UTC date")
			writeOKXData(t, w, [][]string{{"1791216000000", "85000", "1.5", "2.5", "3.5", "4.5"}})
		}))
		expiry := time.Date(2026, time.October, 6, 8, 0, 0, 0, time.UTC).In(time.FixedZone("UTC-12", -12*60*60))
		got, err := ex.GetOpenInterestAndVolumeStrike(t.Context(), &OpenInterestVolumeStrikeRequest{Currency: currency.BTC, ExpiryTime: expiry, Period: kline.OneDay})
		require.NoError(t, err, "GetOpenInterestAndVolumeStrike must not error")
		exp := []OpenInterestVolumeStrikeResponse{{
			Timestamp:        types.Time(time.UnixMilli(1791216000000)),
			Strike:           85000,
			CallOpenInterest: 1.5,
			PutOpenInterest:  2.5,
			CallVolume:       3.5,
			PutVolume:        4.5,
		}}
		assert.Equal(t, exp, got, "GetOpenInterestAndVolumeStrike should decode every row")
	})

	t.Run("recorded expiry", func(t *testing.T) {
		t.Parallel()
		expiry := time.Date(2026, time.November, 27, 8, 0, 0, 0, time.UTC)
		if !mockTests {
			instruments, err := e.GetInstruments(contextGenerate(), &InstrumentsRequest{InstrumentType: instTypeOption, Underlying: optionsPair.String()})
			require.NoErrorf(t, err, "GetInstruments for options (underlying: %s) must not error", optionsPair)
			require.NotEmptyf(t, instruments, "GetInstruments for options (underlying: %s) must return at least one instrument", optionsPair)
			expiry = nextListedExpiry(instruments, time.Now())
			require.NotZero(t, expiry, "GetInstruments must return a later expiry whose instruments are all live and listed")
		}
		result, err := e.GetOpenInterestAndVolumeStrike(contextGenerate(), &OpenInterestVolumeStrikeRequest{Currency: currency.BTC, ExpiryTime: expiry, Period: kline.OneDay})
		require.NoErrorf(t, err, "GetOpenInterestAndVolumeStrike with expiry %s for currency %s must not error", expiry, currency.BTC)
		if mockTests {
			exp := []OpenInterestVolumeStrikeResponse{
				{
					Timestamp:        types.Time(time.UnixMilli(1795363200000)),
					Strike:           46000,
					CallOpenInterest: 0.87,
					PutOpenInterest:  207.22,
					PutVolume:        115.81,
				},
				{
					Timestamp:        types.Time(time.UnixMilli(1795363200000)),
					Strike:           48000,
					CallOpenInterest: 8.97,
					PutOpenInterest:  200.73,
					CallVolume:       3.2,
					PutVolume:        39.48,
				},
				{
					Timestamp:        types.Time(time.UnixMilli(1795363200000)),
					Strike:           50000,
					CallOpenInterest: 3.2,
					PutOpenInterest:  463.61,
					PutVolume:        194.18,
				},
				{
					Timestamp:        types.Time(time.UnixMilli(1795363200000)),
					Strike:           52000,
					CallOpenInterest: 36.41,
					PutOpenInterest:  211.77,
					PutVolume:        48.81,
				},
				{
					Timestamp:        types.Time(time.UnixMilli(1795363200000)),
					Strike:           53000,
					CallOpenInterest: 0.25,
					PutOpenInterest:  150.42,
					CallVolume:       0.12,
					PutVolume:        82.21,
				},
				{
					Timestamp:        types.Time(time.UnixMilli(1795363200000)),
					Strike:           54000,
					CallOpenInterest: 6.43,
					PutOpenInterest:  272.84,
					CallVolume:       0.39,
					PutVolume:        56.36,
				},
			}
			assert.Equal(t, exp, result, "GetOpenInterestAndVolumeStrike should decode every field")
		} else {
			assert.NotNilf(t, result, "GetOpenInterestAndVolumeStrike with expiry %s for currency %s should return a non-nil result", expiry, currency.BTC)
		}
	})
}

func TestGetTakerFlow(t *testing.T) {
	t.Parallel()
	_, err := e.GetTakerFlow(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTakerFlow must reject a nil request")
	_, err = e.GetTakerFlow(t.Context(), &OptionStatisticsRequest{Period: kline.OneDay})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTakerFlow must require a currency")

	result, err := e.GetTakerFlow(contextGenerate(), &OptionStatisticsRequest{Currency: currency.BTC, Period: kline.OneDay})
	require.NoError(t, err, "GetTakerFlow must not error")
	require.NotNil(t, result, "GetTakerFlow must return the flow")
	if mockTests {
		exp := &TakerFlowResponse{
			Timestamp:       types.Time(time.UnixMilli(1795363200000)),
			CallBuyVolume:   955.24,
			CallSellVolume:  1405.1,
			PutBuyVolume:    942.33,
			PutSellVolume:   1114.12,
			CallBlockVolume: 4599.65,
			PutBlockVolume:  4022.64,
		}
		assert.Equal(t, exp, result, "GetTakerFlow should decode every field")
	} else {
		assert.False(t, result.Timestamp.Time().IsZero(), "Timestamp should not be zero")
	}
}

func TestContractStatisticsParams(t *testing.T) {
	t.Parallel()
	begin, end := time.UnixMilli(1795320000000), time.UnixMilli(1795406400000)
	instrument := url.Values{"instId": {"BTC-USDT-SWAP"}}
	with := func(key, value string) url.Values {
		params := url.Values{"instId": {"BTC-USDT-SWAP"}}
		params.Set(key, value)
		return params
	}
	for _, tc := range []struct {
		name string
		arg  *ContractStatisticsRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument ID", arg: &ContractStatisticsRequest{Period: kline.FiveMin, Limit: 10}, err: errMissingInstrumentID},
		{name: "begin after end", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Begin: end, End: begin}, err: common.ErrStartAfterEnd},
		{name: "unsupported period", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.TenMin}, err: kline.ErrUnsupportedInterval},
		{name: "instrument only", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String()}, exp: instrument},
		{name: "five minute period", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.FiveMin}, exp: with("period", "5m")},
		{name: "daily period opens at UTC", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.OneDay}, exp: with("period", "1Dutc")},
		{name: "begin", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Begin: begin}, exp: with("begin", "1795320000000")},
		{name: "end", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), End: end}, exp: with("end", "1795406400000")},
		{
			name: "equal begin and end",
			arg:  &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Begin: begin, End: begin},
			exp:  url.Values{"instId": {"BTC-USDT-SWAP"}, "begin": {"1795320000000"}, "end": {"1795320000000"}},
		},
		{name: "limit", arg: &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Limit: 10}, exp: with("limit", "10")},
		{
			name: "every parameter",
			arg:  &ContractStatisticsRequest{InstrumentID: perpetualSwapPair.String(), Period: kline.OneHour, Begin: begin, End: end, Limit: 10},
			exp:  url.Values{"instId": {"BTC-USDT-SWAP"}, "period": {"1H"}, "begin": {"1795320000000"}, "end": {"1795406400000"}, "limit": {"10"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := contractStatisticsParams(tc.arg)
			require.ErrorIs(t, err, tc.err, "contractStatisticsParams must return the expected error")
			assert.Equal(t, tc.exp, params, "contractStatisticsParams should encode every set parameter")
		})
	}
}

func TestCurrencyStatisticsParams(t *testing.T) {
	t.Parallel()
	begin, end := time.UnixMilli(1795320000000), time.UnixMilli(1795406400000)
	for _, tc := range []struct {
		name       string
		ccy        currency.Code
		begin, end time.Time
		period     kline.Interval
		exp        url.Values
		err        error
	}{
		{name: "empty currency", period: kline.OneDay, err: currency.ErrCurrencyCodeEmpty},
		{name: "begin after end", ccy: currency.BTC, begin: end, end: begin, err: common.ErrStartAfterEnd},
		{name: "unsupported period", ccy: currency.BTC, period: kline.TenMin, err: kline.ErrUnsupportedInterval},
		{name: "currency only", ccy: currency.BTC, exp: url.Values{"ccy": {"BTC"}}},
		{name: "begin", ccy: currency.BTC, begin: begin, exp: url.Values{"ccy": {"BTC"}, "begin": {"1795320000000"}}},
		{name: "end", ccy: currency.BTC, end: end, exp: url.Values{"ccy": {"BTC"}, "end": {"1795406400000"}}},
		{name: "equal begin and end", ccy: currency.BTC, begin: begin, end: begin, exp: url.Values{"ccy": {"BTC"}, "begin": {"1795320000000"}, "end": {"1795320000000"}}},
		{name: "daily period without the UTC variant", ccy: currency.BTC, period: kline.OneDay, exp: url.Values{"ccy": {"BTC"}, "period": {"1D"}}},
		{
			name: "every parameter", ccy: currency.BTC, begin: begin, end: end, period: kline.FiveMin,
			exp: url.Values{"ccy": {"BTC"}, "begin": {"1795320000000"}, "end": {"1795406400000"}, "period": {"5m"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := currencyStatisticsParams(tc.ccy, tc.begin, tc.end, tc.period)
			require.ErrorIs(t, err, tc.err, "currencyStatisticsParams must return the expected error")
			assert.Equal(t, tc.exp, params, "currencyStatisticsParams should encode every set parameter")
		})
	}
}

func TestOptionStatisticsParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *OptionStatisticsRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty currency", arg: &OptionStatisticsRequest{Period: kline.OneDay}, err: currency.ErrCurrencyCodeEmpty},
		{name: "unsupported period", arg: &OptionStatisticsRequest{Currency: currency.BTC, Period: kline.TenMin}, err: kline.ErrUnsupportedInterval},
		{name: "currency only", arg: &OptionStatisticsRequest{Currency: currency.BTC}, exp: url.Values{"ccy": {"BTC"}}},
		{name: "eight hour period", arg: &OptionStatisticsRequest{Currency: currency.BTC, Period: kline.EightHour}, exp: url.Values{"ccy": {"BTC"}, "period": {"8H"}}},
		{name: "daily period without the UTC variant", arg: &OptionStatisticsRequest{Currency: currency.ETH, Period: kline.OneDay}, exp: url.Values{"ccy": {"ETH"}, "period": {"1D"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := optionStatisticsParams(tc.arg)
			require.ErrorIs(t, err, tc.err, "optionStatisticsParams must return the expected error")
			assert.Equal(t, tc.exp, params, "optionStatisticsParams should encode every set parameter")
		})
	}
}

func TestSetStatisticsPeriod(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		period    kline.Interval
		appendUTC bool
		exp       url.Values
		err       error
	}{
		{name: "no period", appendUTC: true, exp: url.Values{"ccy": {"BTC"}}},
		{name: "five minutes", period: kline.FiveMin, exp: url.Values{"ccy": {"BTC"}, "period": {"5m"}}},
		{name: "five minutes has no UTC variant", period: kline.FiveMin, appendUTC: true, exp: url.Values{"ccy": {"BTC"}, "period": {"5m"}}},
		{name: "one day in Hong Kong time", period: kline.OneDay, exp: url.Values{"ccy": {"BTC"}, "period": {"1D"}}},
		{name: "one day in UTC", period: kline.OneDay, appendUTC: true, exp: url.Values{"ccy": {"BTC"}, "period": {"1Dutc"}}},
		{name: "unsupported period", period: kline.TenMin, appendUTC: true, exp: url.Values{"ccy": {"BTC"}}, err: kline.ErrUnsupportedInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{"ccy": {"BTC"}}
			err := setStatisticsPeriod(params, tc.period, tc.appendUTC)
			require.ErrorIs(t, err, tc.err, "setStatisticsPeriod must return the expected error")
			assert.Equal(t, tc.exp, params, "setStatisticsPeriod should set a supported period and keep the other parameters")
		})
	}
}

func TestSetBeginEndParams(t *testing.T) {
	t.Parallel()
	begin, end := time.UnixMilli(1795320000000), time.UnixMilli(1795406400000)
	for _, tc := range []struct {
		name       string
		begin, end time.Time
		exp        url.Values
	}{
		{name: "no bounds", exp: url.Values{"ccy": {"BTC"}}},
		{name: "begin", begin: begin, exp: url.Values{"ccy": {"BTC"}, "begin": {"1795320000000"}}},
		{name: "end", end: end, exp: url.Values{"ccy": {"BTC"}, "end": {"1795406400000"}}},
		{name: "begin and end", begin: begin, end: end, exp: url.Values{"ccy": {"BTC"}, "begin": {"1795320000000"}, "end": {"1795406400000"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{"ccy": {"BTC"}}
			setBeginEndParams(params, tc.begin, tc.end)
			assert.Equal(t, tc.exp, params, "setBeginEndParams should add each set bound in milliseconds and keep the other parameters")
		})
	}
}
