package cryptodotcom

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// stakingHistoryStart and stakingHistoryEnd bound the staking history fixtures' window
var (
	stakingHistoryStart = time.UnixMilli(1691455454495)
	stakingHistoryEnd   = time.UnixMilli(1691545277000)
)

// requireStakingHistoryValidation checks a staking history method refuses each request its validation rejects
func requireStakingHistoryValidation(t *testing.T, method string, call func(*StakingHistoryRequest) error) {
	t.Helper()
	require.ErrorIsf(t, call(nil), common.ErrNilPointer, "%s must error for a nil request", method)
	require.ErrorIsf(t, call(&StakingHistoryRequest{StartTime: stakingHistoryEnd, EndTime: stakingHistoryStart}), common.ErrStartAfterEnd, "%s must error for a start after the end", method)
	require.ErrorIsf(t, call(&StakingHistoryRequest{Limit: stakingHistoryMaximumLimit + 1}), errInvalidCount, "%s must error for a limit above the maximum", method)
	require.ErrorIsf(t, call(&StakingHistoryRequest{StartTime: time.UnixMilli(-1)}), errTimeBeforeUnixEpoch, "%s must error for a time before 1970", method)
	require.ErrorIsf(t, call(&StakingHistoryRequest{EndTime: time.UnixMilli(-1)}), errTimeBeforeUnixEpoch, "%s must error for an end before 1970", method)
}

// requireStakingConvertHistoryValidation checks a convert history method refuses each request its validation rejects
func requireStakingConvertHistoryValidation(t *testing.T, method string, call func(*StakingConvertHistoryRequest) error) {
	t.Helper()
	require.ErrorIsf(t, call(nil), common.ErrNilPointer, "%s must error for a nil request", method)
	require.ErrorIsf(t, call(&StakingConvertHistoryRequest{StartTime: stakingHistoryEnd, EndTime: stakingHistoryStart}), common.ErrStartAfterEnd, "%s must error for a start after the end", method)
	require.ErrorIsf(t, call(&StakingConvertHistoryRequest{Limit: stakingHistoryMaximumLimit + 1}), errInvalidCount, "%s must error for a limit above the maximum", method)
	require.ErrorIsf(t, call(&StakingConvertHistoryRequest{StartTime: time.UnixMilli(-1)}), errTimeBeforeUnixEpoch, "%s must error for a time before 1970", method)
}

func TestGetStakingConversionRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetStakingConversionRate(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetStakingConversionRate must error without an instrument name")

	result, err := e.GetStakingConversionRate(t.Context(), "CDCETH")
	require.NoError(t, err, "GetStakingConversionRate must not error")
	if mockTests {
		exp := &StakingConversionRateResponse{InstrumentName: "CDCETH", ConversionRate: 1.0768}
		assert.Equal(t, exp, result, "GetStakingConversionRate should decode every field")
	} else {
		assert.Equal(t, "CDCETH", result.InstrumentName, "GetStakingConversionRate should return the requested instrument's rate")
		assert.Positive(t, result.ConversionRate.Float64(), "GetStakingConversionRate should return a positive rate")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "Reading the request body should not error") {
			return
		}
		var body requestBody
		if !assert.NoError(t, json.Unmarshal(raw, &body), "The request body should decode") {
			return
		}
		assert.Equal(t, http.MethodPost, r.Method, "GetStakingConversionRate should send a POST")
		assert.Equal(t, "/exchange/v1/public/staking/get-conversion-rate", r.URL.Path, "GetStakingConversionRate should be sent to its method's path")
		assert.Equal(t, "public/staking/get-conversion-rate", body.Method, "GetStakingConversionRate should name its method")
		assert.Empty(t, body.APIKey, "GetStakingConversionRate should not send the API key")
		assert.Empty(t, body.Signature, "GetStakingConversionRate should not sign its request")
		assert.JSONEq(t, `{"instrument_name":"CDCETH"}`, string(body.Params), "GetStakingConversionRate should send the instrument name")
		_, err = w.Write([]byte(`{"id":1,"method":"public/staking/get-conversion-rate","code":0,"result":{"instrument_name":"CDCETH","conversion_rate":"1.0768"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = ex.GetStakingConversionRate(t.Context(), "CDCETH")
	require.NoError(t, err, "GetStakingConversionRate must not error against the test server, though it holds credentials")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"public/staking/get-conversion-rate","code":0,"result":null}`).GetStakingConversionRate(t.Context(), "CDCETH")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetStakingConversionRate should error for a null result")
	_, err = newUnavailableServerExchange(t).GetStakingConversionRate(t.Context(), "CDCETH")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetStakingConversionRate should return a request failure")
}

func TestStake(t *testing.T) {
	t.Parallel()
	_, err := e.Stake(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Stake must error for a nil request")
	_, err = e.Stake(t.Context(), &StakeRequest{Quantity: 1})
	require.ErrorIs(t, err, errInstrumentNameRequired, "Stake must error without an instrument name")
	_, err = e.Stake(t.Context(), &StakeRequest{InstrumentName: "SOL.staked"})
	require.ErrorIs(t, err, errInvalidQuantity, "Stake must error without a quantity")
	_, err = e.Stake(t.Context(), &StakeRequest{InstrumentName: "SOL.staked", Quantity: -1})
	require.ErrorIs(t, err, errInvalidQuantity, "Stake must error for a negative quantity")
	skipLiveUnlessCanManipulate(t)

	req := &StakeRequest{InstrumentName: "SOL.staked", Quantity: 1}
	result, err := e.Stake(t.Context(), req)
	require.NoError(t, err, "Stake must not error")
	if mockTests {
		exp := &StakeResponse{
			StakingID:                     "4611686018445261031",
			InstrumentName:                "SOL.staked",
			Status:                        StakingStatusNew,
			Quantity:                      preciseNumber("1"),
			UnderlyingCurrency:            currency.SOL,
			PreStakeChargeRateBasisPoints: 50,
			PreStakeCharge:                preciseNumber("0.005"),
			Reason:                        "NO_ERROR",
		}
		assert.Equal(t, exp, result, "Stake should decode every field")
	} else {
		assert.NotEmpty(t, result.StakingID, "Stake should return the stake request's ID")
	}

	ex := newSignedParamsServer(t, "private/staking/stake", `{"instrument_name":"SOL.staked","quantity":"1"}`, `{"id":1,"method":"private/staking/stake","code":0,"result":{"staking_id":"1"}}`)
	_, err = ex.Stake(t.Context(), req)
	require.NoError(t, err, "Stake must not error against the test server")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/staking/stake","code":0,"result":null}`).Stake(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "Stake should error for a null result")
	_, err = newUnavailableServerExchange(t).Stake(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "Stake should return a request failure")
}

func TestUnstake(t *testing.T) {
	t.Parallel()
	_, err := e.Unstake(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Unstake must error for a nil request")
	_, err = e.Unstake(t.Context(), &UnstakeRequest{Quantity: 0.5})
	require.ErrorIs(t, err, errInstrumentNameRequired, "Unstake must error without an instrument name")
	_, err = e.Unstake(t.Context(), &UnstakeRequest{InstrumentName: "SOL.staked"})
	require.ErrorIs(t, err, errInvalidQuantity, "Unstake must error without a quantity")
	skipLiveUnlessCanManipulate(t)

	req := &UnstakeRequest{InstrumentName: "SOL.staked", Quantity: 0.5}
	result, err := e.Unstake(t.Context(), req)
	require.NoError(t, err, "Unstake must not error")
	if mockTests {
		exp := &UnstakeResponse{
			StakingID:          "4611686018445261047",
			InstrumentName:     "SOL.staked",
			Status:             StakingStatusNew,
			Quantity:           preciseNumber("0.5"),
			UnderlyingCurrency: currency.SOL,
			Reason:             "NO_ERROR",
		}
		assert.Equal(t, exp, result, "Unstake should decode every field")
	} else {
		assert.NotEmpty(t, result.StakingID, "Unstake should return the unstake request's ID")
	}

	ex := newSignedParamsServer(t, "private/staking/unstake", `{"instrument_name":"SOL.staked","quantity":"0.5"}`, `{"id":1,"method":"private/staking/unstake","code":0,"result":{"staking_id":"1"}}`)
	_, err = ex.Unstake(t.Context(), req)
	require.NoError(t, err, "Unstake must not error against the test server")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/staking/unstake","code":0,"result":null}`).Unstake(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "Unstake should error for a null result")
	_, err = newUnavailableServerExchange(t).Unstake(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "Unstake should return a request failure")
}

func TestGetStakingPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetStakingPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetStakingPosition must error for a nil request")
	skipLiveWithoutCredentials(t)

	result, err := e.GetStakingPosition(t.Context(), &StakingPositionRequest{InstrumentName: "SOL.staked"})
	require.NoError(t, err, "GetStakingPosition must not error")
	if mockTests {
		exp := []StakingPositionResponse{
			{
				InstrumentName:          "SOL.staked",
				UnderlyingCurrency:      currency.SOL,
				StakedQuantity:          preciseNumber("30000.00"),
				PendingStakedQuantity:   preciseNumber("20000.00"),
				PendingUnstakedQuantity: preciseNumber("10000.00"),
				RewardEligibleQuantity:  preciseNumber("10000.00"),
			},
		}
		assert.Equal(t, exp, result, "GetStakingPosition should decode every field")
	} else {
		for i := range result {
			assert.Equalf(t, "SOL.staked", result[i].InstrumentName, "GetStakingPosition should return position %d of the instrument requested", i)
		}
	}

	for _, tc := range []struct {
		req *StakingPositionRequest
		exp string
	}{
		{req: &StakingPositionRequest{InstrumentName: "SOL.staked"}, exp: `{"instrument_name":"SOL.staked"}`},
		{req: &StakingPositionRequest{}, exp: `{}`},
	} {
		ex := newSignedParamsServer(t, "private/staking/get-staking-position", tc.exp, `{"id":1,"method":"private/staking/get-staking-position","code":0,"result":{"data":[]}}`)
		_, err = ex.GetStakingPosition(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetStakingPosition must not error against the test server for %+v", tc.req)
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/staking/get-staking-position","code":0,"result":null}`).GetStakingPosition(t.Context(), &StakingPositionRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetStakingPosition should error for a null result")
	_, err = newUnavailableServerExchange(t).GetStakingPosition(t.Context(), &StakingPositionRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetStakingPosition should return a request failure")
}

func TestGetStakingInstruments(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)

	result, err := e.GetStakingInstruments(t.Context())
	require.NoError(t, err, "GetStakingInstruments must not error")
	if mockTests {
		exp := []StakingInstrumentResponse{
			{
				InstrumentName:       "SOL.staked",
				UnderlyingCurrency:   currency.SOL,
				RewardInstrumentName: "SOL.staked",
				EstimatedRewards:     0.0661,
				RewardRateType:       StakingRewardRateTypeAPR,
				MinimumStakeAmount:   0.00000001,
				RewardFrequencyDays:  2.5,
				LockUpPeriodDays:     5,
				IsCompoundReward:     true,
				AdditionalRewards:    []StakingAdditionalReward{},
			},
			{
				InstrumentName:       "DYDX.staked",
				UnderlyingCurrency:   currency.DYDX,
				RewardInstrumentName: "DYDX",
				EstimatedRewards:     0.05,
				RewardRateType:       StakingRewardRateTypeAPR,
				MinimumStakeAmount:   0.00000001,
				RewardFrequencyDays:  1,
				LockUpPeriodDays:     31,
				AdditionalRewards:    []StakingAdditionalReward{{RewardInstrumentName: "USD_Stable_Coin"}},
			},
		}
		assert.Equal(t, exp, result, "GetStakingInstruments should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetStakingInstruments should return instruments")
	}

	ex := newSignedParamsServer(t, "private/staking/get-staking-instruments", `{}`, `{"id":1,"method":"private/staking/get-staking-instruments","code":0,"result":{"data":[]}}`)
	_, err = ex.GetStakingInstruments(t.Context())
	require.NoError(t, err, "GetStakingInstruments must not error against the test server")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/staking/get-staking-instruments","code":0,"result":null}`).GetStakingInstruments(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetStakingInstruments should error for a null result")
	_, err = newUnavailableServerExchange(t).GetStakingInstruments(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetStakingInstruments should return a request failure")
}

// stakingHistoryParamCases are a staking history query with every param set and with none, and the params each sends
var stakingHistoryParamCases = []struct {
	req *StakingHistoryRequest
	exp string
}{
	{
		req: &StakingHistoryRequest{InstrumentName: "SOL.staked", StartTime: stakingHistoryStart, EndTime: stakingHistoryEnd, Limit: 10},
		exp: `{"instrument_name":"SOL.staked","start_time":"1691455454495000000","end_time":"1691545277000000000","limit":"10"}`,
	},
	{req: &StakingHistoryRequest{}, exp: `{}`},
}

func TestGetOpenStakes(t *testing.T) {
	t.Parallel()
	requireStakingHistoryValidation(t, "GetOpenStakes", func(req *StakingHistoryRequest) error {
		_, err := e.GetOpenStakes(t.Context(), req)
		return err
	})
	skipLiveWithoutCredentials(t)

	req := &StakingHistoryRequest{InstrumentName: "SOL.staked", StartTime: stakingHistoryStart, EndTime: stakingHistoryEnd, Limit: 10}
	result, err := e.GetOpenStakes(t.Context(), req)
	require.NoError(t, err, "GetOpenStakes must not error")
	if mockTests {
		exp := []StakingRequestResponse{
			{
				InstrumentName:     "SOL.staked",
				UnderlyingCurrency: currency.SOL,
				CycleID:            "5821",
				StakingID:          "4611686018445098512",
				Status:             StakingStatusPending,
				AccountID:          "8b1fd0a5-3d2e-4f7a-9c61-2e5b7a4d9f30",
				Quantity:           preciseNumber("1"),
				Side:               StakingSideStake,
				CreateTime:         types.Time(time.UnixMilli(1691461234567)),
			},
			{
				InstrumentName:     "SOL.staked",
				UnderlyingCurrency: currency.SOL,
				CycleID:            "5822",
				StakingID:          "4611686018445101227",
				Status:             StakingStatusPendingUnstaking,
				AccountID:          "8b1fd0a5-3d2e-4f7a-9c61-2e5b7a4d9f30",
				Quantity:           preciseNumber("0.5"),
				Side:               StakingSideUnstake,
				CreateTime:         types.Time(time.UnixMilli(1691502345678)),
			},
		}
		assert.Equal(t, exp, result, "GetOpenStakes should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].StakingID, "GetOpenStakes should return request %d's ID", i)
		}
	}

	for _, tc := range stakingHistoryParamCases {
		ex := newSignedParamsServer(t, "private/staking/get-open-stake", tc.exp, `{"id":1,"method":"private/staking/get-open-stake","code":0,"result":{"data":[]}}`)
		_, err = ex.GetOpenStakes(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetOpenStakes must not error against the test server for %+v", tc.req)
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/staking/get-open-stake","code":0,"result":null}`).GetOpenStakes(t.Context(), &StakingHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOpenStakes should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOpenStakes(t.Context(), &StakingHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenStakes should return a request failure")
}

func TestGetStakeHistory(t *testing.T) {
	t.Parallel()
	requireStakingHistoryValidation(t, "GetStakeHistory", func(req *StakingHistoryRequest) error {
		_, err := e.GetStakeHistory(t.Context(), req)
		return err
	})
	skipLiveWithoutCredentials(t)

	req := &StakingHistoryRequest{InstrumentName: "SOL.staked", StartTime: stakingHistoryStart, EndTime: stakingHistoryEnd, Limit: 10}
	result, err := e.GetStakeHistory(t.Context(), req)
	require.NoError(t, err, "GetStakeHistory must not error")
	if mockTests {
		exp := []StakingRequestResponse{
			{
				InstrumentName:     "SOL.staked",
				UnderlyingCurrency: currency.SOL,
				CycleID:            "5819",
				StakingID:          "4611686018444920315",
				Status:             StakingStatusCompleted,
				AccountID:          "8b1fd0a5-3d2e-4f7a-9c61-2e5b7a4d9f30",
				Quantity:           preciseNumber("1"),
				Side:               StakingSideStake,
				CreateTime:         types.Time(time.UnixMilli(1691458765432)),
			},
			{
				InstrumentName:     "SOL.staked",
				UnderlyingCurrency: currency.SOL,
				CycleID:            "5820",
				StakingID:          "4611686018444937786",
				Status:             StakingStatusRejected,
				AccountID:          "8b1fd0a5-3d2e-4f7a-9c61-2e5b7a4d9f30",
				Quantity:           preciseNumber("0.5"),
				Side:               StakingSideUnstake,
				CreateTime:         types.Time(time.UnixMilli(1691512345678)),
			},
		}
		assert.Equal(t, exp, result, "GetStakeHistory should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].StakingID, "GetStakeHistory should return request %d's ID", i)
		}
	}

	for _, tc := range stakingHistoryParamCases {
		ex := newSignedParamsServer(t, "private/staking/get-stake-history", tc.exp, `{"id":1,"method":"private/staking/get-stake-history","code":0,"result":{"data":[]}}`)
		_, err = ex.GetStakeHistory(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetStakeHistory must not error against the test server for %+v", tc.req)
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/staking/get-stake-history","code":0,"result":null}`).GetStakeHistory(t.Context(), &StakingHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetStakeHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetStakeHistory(t.Context(), &StakingHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetStakeHistory should return a request failure")
}

func TestGetStakingRewardHistory(t *testing.T) {
	t.Parallel()
	requireStakingHistoryValidation(t, "GetStakingRewardHistory", func(req *StakingHistoryRequest) error {
		_, err := e.GetStakingRewardHistory(t.Context(), req)
		return err
	})
	skipLiveWithoutCredentials(t)

	req := &StakingHistoryRequest{InstrumentName: "SOL.staked", StartTime: stakingHistoryStart, EndTime: stakingHistoryEnd, Limit: 10}
	result, err := e.GetStakingRewardHistory(t.Context(), req)
	require.NoError(t, err, "GetStakingRewardHistory must not error")
	if mockTests {
		exp := []StakingRewardResponse{
			{
				StakingInstrumentName: "SOL.staked",
				UnderlyingCurrency:    currency.SOL,
				RewardInstrumentName:  "SOL.staked",
				RewardQuantity:        preciseNumber("123.4567"),
				StakedBalance:         preciseNumber("1234567"),
				EventTime:             types.Time(time.UnixMilli(1691539200000)),
			},
		}
		assert.Equal(t, exp, result, "GetStakingRewardHistory should decode every field")
	} else {
		for i := range result {
			assert.Positivef(t, result[i].RewardQuantity.Float64(), "GetStakingRewardHistory should return reward %d's quantity", i)
		}
	}

	for _, tc := range stakingHistoryParamCases {
		ex := newSignedParamsServer(t, "private/staking/get-reward-history", tc.exp, `{"id":1,"method":"private/staking/get-reward-history","code":0,"result":{"data":[]}}`)
		_, err = ex.GetStakingRewardHistory(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetStakingRewardHistory must not error against the test server for %+v", tc.req)
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/staking/get-reward-history","code":0,"result":null}`).GetStakingRewardHistory(t.Context(), &StakingHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetStakingRewardHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetStakingRewardHistory(t.Context(), &StakingHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetStakingRewardHistory should return a request failure")
}

func TestConvertStakedToken(t *testing.T) {
	t.Parallel()
	_, err := e.ConvertStakedToken(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ConvertStakedToken must error for a nil request")
	req := &ConvertStakedTokenRequest{FromInstrumentName: "ETH.staked", ToInstrumentName: "CDCETH", ExpectedRate: 1.0203, FromQuantity: 3.14159265, SlippageToleranceBasisPoints: 3}
	for _, tc := range []struct {
		name   string
		modify func(*ConvertStakedTokenRequest)
		err    error
	}{
		{name: "without a from instrument", modify: func(r *ConvertStakedTokenRequest) { r.FromInstrumentName = "" }, err: errInstrumentNameRequired},
		{name: "without a to instrument", modify: func(r *ConvertStakedTokenRequest) { r.ToInstrumentName = "" }, err: errInstrumentNameRequired},
		{name: "without an expected rate", modify: func(r *ConvertStakedTokenRequest) { r.ExpectedRate = 0 }, err: errInvalidStakingConversionRate},
		{name: "without a quantity", modify: func(r *ConvertStakedTokenRequest) { r.FromQuantity = 0 }, err: errInvalidQuantity},
		{name: "for a negative slippage tolerance", modify: func(r *ConvertStakedTokenRequest) { r.SlippageToleranceBasisPoints = -1 }, err: errInvalidStakingSlippageTolerance},
	} {
		invalid := *req
		tc.modify(&invalid)
		_, err = e.ConvertStakedToken(t.Context(), &invalid)
		require.ErrorIsf(t, err, tc.err, "ConvertStakedToken must error %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	result, err := e.ConvertStakedToken(t.Context(), req)
	require.NoError(t, err, "ConvertStakedToken must not error")
	if mockTests {
		exp := &ConvertStakedTokenResponse{
			FromInstrumentName:           "ETH.staked",
			ToInstrumentName:             "CDCETH",
			ExpectedRate:                 1.0203,
			FromQuantity:                 preciseNumber("3.14159265"),
			SlippageToleranceBasisPoints: 3,
			ConvertID:                    "4611686018445302411",
			Reason:                       "NO_ERROR",
		}
		assert.Equal(t, exp, result, "ConvertStakedToken should decode every field")
	} else {
		assert.NotEmpty(t, result.ConvertID, "ConvertStakedToken should return the convert request's ID")
	}

	zeroSlippage := *req
	zeroSlippage.SlippageToleranceBasisPoints = 0
	for _, tc := range []struct {
		req *ConvertStakedTokenRequest
		exp string
	}{
		{req: req, exp: `{"from_instrument_name":"ETH.staked","to_instrument_name":"CDCETH","expected_rate":"1.0203","from_quantity":"3.14159265","slippage_tolerance_bps":"3"}`},
		{req: &zeroSlippage, exp: `{"from_instrument_name":"ETH.staked","to_instrument_name":"CDCETH","expected_rate":"1.0203","from_quantity":"3.14159265","slippage_tolerance_bps":"0"}`},
	} {
		ex := newSignedParamsServer(t, "private/staking/convert", tc.exp, `{"id":1,"method":"private/staking/convert","code":0,"result":{"convert_id":"1"}}`)
		_, err = ex.ConvertStakedToken(t.Context(), tc.req)
		require.NoErrorf(t, err, "ConvertStakedToken must not error against the test server for %+v", tc.req)
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/staking/convert","code":0,"result":null}`).ConvertStakedToken(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "ConvertStakedToken should error for a null result")
	_, err = newUnavailableServerExchange(t).ConvertStakedToken(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ConvertStakedToken should return a request failure")
}

// stakingConvertHistoryParamCases are a convert history query with every param set and with none, and the params each
// sends
var stakingConvertHistoryParamCases = []struct {
	req *StakingConvertHistoryRequest
	exp string
}{
	{
		req: &StakingConvertHistoryRequest{StartTime: stakingHistoryStart, EndTime: stakingHistoryEnd, Limit: 10},
		exp: `{"start_time":"1691455454495000000","end_time":"1691545277000000000","limit":"10"}`,
	},
	{req: &StakingConvertHistoryRequest{}, exp: `{}`},
}

func TestGetOpenStakingConverts(t *testing.T) {
	t.Parallel()
	requireStakingConvertHistoryValidation(t, "GetOpenStakingConverts", func(req *StakingConvertHistoryRequest) error {
		_, err := e.GetOpenStakingConverts(t.Context(), req)
		return err
	})
	skipLiveWithoutCredentials(t)

	req := &StakingConvertHistoryRequest{StartTime: stakingHistoryStart, EndTime: stakingHistoryEnd, Limit: 10}
	result, err := e.GetOpenStakingConverts(t.Context(), req)
	require.NoError(t, err, "GetOpenStakingConverts must not error")
	if mockTests {
		exp := []StakingConvertRequestResponse{
			{
				FromInstrumentName:           "ETH.staked",
				ToInstrumentName:             "CDCETH",
				ExpectedRate:                 1.0203,
				FromQuantity:                 preciseNumber("3.14159265"),
				ToQuantity:                   preciseNumber("0"),
				SlippageToleranceBasisPoints: 3,
				ConvertID:                    "4611686018445319652",
				Status:                       StakingConvertStatusNew,
				CreateTime:                   types.Time(time.UnixMilli(1691530000123)),
			},
		}
		assert.Equal(t, exp, result, "GetOpenStakingConverts should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].ConvertID, "GetOpenStakingConverts should return request %d's ID", i)
		}
	}

	for _, tc := range stakingConvertHistoryParamCases {
		ex := newSignedParamsServer(t, "private/staking/get-open-convert", tc.exp, `{"id":1,"method":"private/staking/get-open-convert","code":0,"result":{"data":[]}}`)
		_, err = ex.GetOpenStakingConverts(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetOpenStakingConverts must not error against the test server for %+v", tc.req)
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/staking/get-open-convert","code":0,"result":null}`).GetOpenStakingConverts(t.Context(), &StakingConvertHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOpenStakingConverts should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOpenStakingConverts(t.Context(), &StakingConvertHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenStakingConverts should return a request failure")
}

func TestGetStakingConvertHistory(t *testing.T) {
	t.Parallel()
	requireStakingConvertHistoryValidation(t, "GetStakingConvertHistory", func(req *StakingConvertHistoryRequest) error {
		_, err := e.GetStakingConvertHistory(t.Context(), req)
		return err
	})
	skipLiveWithoutCredentials(t)

	req := &StakingConvertHistoryRequest{StartTime: stakingHistoryStart, EndTime: stakingHistoryEnd, Limit: 10}
	result, err := e.GetStakingConvertHistory(t.Context(), req)
	require.NoError(t, err, "GetStakingConvertHistory must not error")
	if mockTests {
		exp := []StakingConvertRequestResponse{
			{
				FromInstrumentName:           "ETH.staked",
				ToInstrumentName:             "CDCETH",
				ExpectedRate:                 1.0203,
				FromQuantity:                 preciseNumber("3.14159265"),
				SlippageToleranceBasisPoints: 3,
				ActualRate:                   1.0203,
				ToQuantity:                   preciseNumber("3.07908718"),
				ConvertID:                    "4611686018445288107",
				Status:                       StakingConvertStatusCompleted,
				CreateTime:                   types.Time(time.UnixMilli(1691470000456)),
			},
		}
		assert.Equal(t, exp, result, "GetStakingConvertHistory should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].ConvertID, "GetStakingConvertHistory should return request %d's ID", i)
		}
	}

	for _, tc := range stakingConvertHistoryParamCases {
		ex := newSignedParamsServer(t, "private/staking/get-convert-history", tc.exp, `{"id":1,"method":"private/staking/get-convert-history","code":0,"result":{"data":[]}}`)
		_, err = ex.GetStakingConvertHistory(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetStakingConvertHistory must not error against the test server for %+v", tc.req)
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/staking/get-convert-history","code":0,"result":null}`).GetStakingConvertHistory(t.Context(), &StakingConvertHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetStakingConvertHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetStakingConvertHistory(t.Context(), &StakingConvertHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetStakingConvertHistory should return a request failure")
}
