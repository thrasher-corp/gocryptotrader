package bingx

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSwapAccountData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapAccountData(t.Context())
	require.NoError(t, err, "GetSwapAccountData must not error")
	if mockTests {
		exp := []SwapAccountDataResponse{
			{
				UserID:           "116***295",
				Asset:            currency.USDT,
				Balance:          194.8212,
				Equity:           196.7431,
				UnrealisedProfit: 1.9219,
				RealisedProfit:   -109.2504,
				AvailableMargin:  193.7609,
				UsedMargin:       1.0602,
			},
		}
		assert.Equal(t, exp, result, "GetSwapAccountData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetSwapAccountData should return the account's assets")
}

func TestGetSwapPositionData(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapPositionData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapPositionData must reject a nil request")
	_, err = new(Exchange).GetSwapPositionData(t.Context(), &SwapPositionDataRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapPositionData must return the symbol formatting error")

	t.Run("request parameters", func(t *testing.T) {
		t.Parallel()
		var method, path, rawQuery string
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			method, path, rawQuery = r.Method, r.URL.Path, r.URL.RawQuery
			_, err := io.WriteString(w, `{"code":0,"msg":"","data":[]}`)
			assert.NoError(t, err, "WriteString should not error")
		})
		result, err := ex.GetSwapPositionData(t.Context(), &SwapPositionDataRequest{Symbol: currency.NewPair(currency.BTC, currency.USDC), Currency: currency.NewCode("usdc")})
		require.NoError(t, err, "GetSwapPositionData must not error")
		assert.Empty(t, result, "GetSwapPositionData should return no positions for an empty list")
		assert.Equal(t, http.MethodGet, method, "GetSwapPositionData should send a GET request")
		assert.Equal(t, "/openApi/swap/v2/user/positions", path, "GetSwapPositionData should request its path")
		keys, values := orderedQuery(t, rawQuery)
		require.Equal(t, []string{"currency", "recvWindow", "symbol", "timestamp", "signature"}, keys, "GetSwapPositionData must send the symbol and settlement currency, signed")
		assert.Equal(t, "USDC", values[0], "currency should be sent in upper case")
		assert.Equal(t, "BTC-USDC", values[2], "symbol should be sent as a perpetual symbol")
		verifyQuerySignature(t, rawQuery)
	})

	t.Run("positions", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetSwapPositionData(t.Context(), &SwapPositionDataRequest{Symbol: currency.NewPair(currency.BTC, currency.USDC), Currency: currency.USDC})
		require.NoError(t, err, "GetSwapPositionData must not error")
		if mockTests {
			exp := []SwapPositionDataResponse{
				{
					Symbol:           "BTC-USDC",
					Currency:         currency.USDC,
					PositionID:       "1735*****52",
					PositionSide:     PositionSideLong,
					Isolated:         true,
					PositionAmount:   0.0314,
					AvailableAmount:  0.0314,
					UnrealisedProfit: -0.0003,
					RealisedProfit:   -4.0346,
					InitialMargin:    118.6701,
					LiquidationPrice: 72162.04,
					AveragePrice:     75586.05,
					Leverage:         20,
				},
			}
			assert.Equal(t, exp, result, "GetSwapPositionData should decode every field")
			return
		}
		for _, p := range result {
			assert.Equal(t, currency.USDC, p.Currency, "Currency should be the requested settlement currency")
		}
	})
}

func TestGetSwapFundFlow(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapFundFlow(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapFundFlow must reject a nil request")
	_, err = new(Exchange).GetSwapFundFlow(t.Context(), &SwapFundFlowRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapFundFlow must return the symbol formatting error")
	startTime, endTime := time.UnixMilli(1702713615001), time.UnixMilli(1702731787011)
	_, err = e.GetSwapFundFlow(t.Context(), &SwapFundFlowRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapFundFlow must reject a start time after the end time")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	arg := &SwapFundFlowRequest{
		Symbol:     currency.NewPair(currency.LDO, currency.USDT),
		Currency:   currency.USDT,
		IncomeType: IncomeTypeFundingFee,
		StartTime:  startTime,
		EndTime:    endTime,
		Limit:      1000,
	}
	if !mockTests {
		// BingX keeps only the last three months, so a live request takes its default window of the last seven days
		arg = &SwapFundFlowRequest{Limit: 10}
	}
	result, err := e.GetSwapFundFlow(t.Context(), arg)
	require.NoError(t, err, "GetSwapFundFlow must not error")
	if mockTests {
		exp := []SwapFundFlowResponse{
			{
				Symbol:     "LDO-USDT",
				IncomeType: IncomeTypeFundingFee,
				Income:     -0.0292,
				Asset:      currency.USDT,
				Info:       "Funding Fee",
				Time:       types.Time(time.UnixMilli(1702713615000)),
				TransferID: "170***6*2_3*9_20***97",
				TradeID:    "170***6*2_3*9_20***97",
			},
		}
		assert.Equal(t, exp, result, "GetSwapFundFlow should decode every field")
		return
	}
	assert.LessOrEqual(t, len(result), 10, "GetSwapFundFlow should return at most the limit")
}

func TestGetSwapTradingCommissionRate(t *testing.T) {
	t.Parallel()
	t.Run("missing commission", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, err := io.WriteString(w, `{"code":0,"msg":"","data":{}}`)
			assert.NoError(t, err, "WriteString should not error")
		})
		_, err := ex.GetSwapTradingCommissionRate(t.Context())
		assert.ErrorIs(t, err, common.ErrNoResponse, "GetSwapTradingCommissionRate should reject a reply without commission rates")
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, err := io.WriteString(w, `{"code":100001,"msg":"Signature verification failed","data":{}}`)
			assert.NoError(t, err, "WriteString should not error")
		})
		_, err := ex.GetSwapTradingCommissionRate(t.Context())
		assert.ErrorIs(t, err, errAPIResponse, "GetSwapTradingCommissionRate should return the API error")
	})

	t.Run("commission rates", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetSwapTradingCommissionRate(t.Context())
		require.NoError(t, err, "GetSwapTradingCommissionRate must not error")
		require.NotNil(t, result, "GetSwapTradingCommissionRate must return a result")
		if mockTests {
			exp := &SwapTradingCommissionRateResponse{Commission: &SwapCommissionRate{TakerCommissionRate: 0.0005, MakerCommissionRate: 0.0002}}
			assert.Equal(t, exp, result, "GetSwapTradingCommissionRate should decode every field")
			return
		}
		assert.NotNil(t, result.Commission, "Commission should be returned")
	})
}
