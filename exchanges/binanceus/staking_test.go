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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetStakingAssetInformation(t *testing.T) {
	t.Parallel()

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name         string
			stakingAsset currency.Code
			exp          url.Values
		}{
			{name: "every asset", exp: url.Values{}},
			{name: "one asset", stakingAsset: currency.NewCode("bnb"), exp: url.Values{"stakingAsset": {"BNB"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/staking/asset", tc.exp, `[]`)
				_, err := ex.GetStakingAssetInformation(t.Context(), tc.stakingAsset)
				require.NoError(t, err, "GetStakingAssetInformation must not error")
			})
		}
	})

	t.Run("returns the staking terms", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetStakingAssetInformation(t.Context(), currency.BNB)
		require.NoError(t, err, "GetStakingAssetInformation must not error")
		if mockTests {
			exp := []StakingAssetInformationResponse{{
				StakingAsset:           currency.BNB,
				RewardAsset:            currency.BNB,
				AnnualPercentageRate:   0.0517,
				AnnualPercentageYield:  0.04,
				UnstakingPeriodInHours: 168,
				MinStakingLimit:        0.01,
				MaxStakingLimit:        10000,
				AutoRestake:            true,
			}}
			assert.Equal(t, exp, result, "GetStakingAssetInformation should decode the staking terms")
		}
	})
}

func TestStakeAsset(t *testing.T) {
	t.Parallel()
	_, err := e.StakeAsset(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "StakeAsset must refuse a nil request")
	_, err = e.StakeAsset(t.Context(), &StakeAssetRequest{Amount: 0.01})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "StakeAsset must require a staking asset")
	_, err = e.StakeAsset(t.Context(), &StakeAssetRequest{StakingAsset: currency.BNB})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "StakeAsset must require an amount")
	_, err = e.StakeAsset(t.Context(), &StakeAssetRequest{StakingAsset: currency.BNB, Amount: -1})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "StakeAsset must refuse a negative amount")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *StakeAssetRequest
			exp  url.Values
		}{
			{
				name: "default restaking",
				req:  &StakeAssetRequest{StakingAsset: currency.NewCode("bnb"), Amount: 0.00012345},
				exp:  url.Values{"stakingAsset": {"BNB"}, "amount": {"0.00012345"}},
			},
			{
				name: "without restaking",
				req:  &StakeAssetRequest{StakingAsset: currency.BNB, Amount: 12.5, AutoRestake: new(false)},
				exp:  url.Values{"stakingAsset": {"BNB"}, "amount": {"12.5"}, "autoRestake": {"false"}},
			},
			{
				name: "with restaking",
				req:  &StakeAssetRequest{StakingAsset: currency.BNB, Amount: 1, AutoRestake: new(true)},
				exp:  url.Values{"stakingAsset": {"BNB"}, "amount": {"1"}, "autoRestake": {"true"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/staking/stake", tc.exp, `{"code":"000000","message":"success","data":{"result":"SUCCESS"},"success":true}`)
				_, err := ex.StakeAsset(t.Context(), tc.req)
				require.NoError(t, err, "StakeAsset must not error")
			})
		}
	})

	t.Run("returns a failed stake as an API error", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/staking/stake", url.Values{"stakingAsset": {"BNB"}, "amount": {"0.01"}}, `{"code":"100001","message":"Staking is not available.","success":false}`)
		result, err := ex.StakeAsset(t.Context(), &StakeAssetRequest{StakingAsset: currency.BNB, Amount: 0.01})
		require.ErrorIs(t, err, errAPIResponse, "StakeAsset must return the API error")
		assert.Nil(t, result, "StakeAsset should not return a stake")
	})

	t.Run("stakes the asset", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.StakeAsset(t.Context(), &StakeAssetRequest{StakingAsset: currency.BNB, Amount: 0.01, AutoRestake: new(false)})
		require.NoError(t, err, "StakeAsset must not error")
		require.NotNil(t, result, "StakeAsset must return a result")
		if mockTests {
			assert.Equal(t, &StakeAssetResponse{Result: "SUCCESS", PurchaseRecordID: "111"}, result, "StakeAsset should decode the stake")
		}
	})
}

func TestUnstakeAsset(t *testing.T) {
	t.Parallel()
	_, err := e.UnstakeAsset(t.Context(), currency.EMPTYCODE, 0.01)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UnstakeAsset must require a staking asset")
	_, err = e.UnstakeAsset(t.Context(), currency.BNB, 0)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "UnstakeAsset must require an amount")
	_, err = e.UnstakeAsset(t.Context(), currency.BNB, -1)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "UnstakeAsset must refuse a negative amount")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/staking/unstake", url.Values{"stakingAsset": {"BNB"}, "amount": {"0.00012345"}}, `{"code":"000000","message":"success","data":{"result":"SUCCESS"},"success":true}`)
		_, err := ex.UnstakeAsset(t.Context(), currency.NewCode("bnb"), 0.00012345)
		require.NoError(t, err, "UnstakeAsset must not error")
	})

	t.Run("unstakes the asset", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.UnstakeAsset(t.Context(), currency.BNB, 0.01)
		require.NoError(t, err, "UnstakeAsset must not error")
		require.NotNil(t, result, "UnstakeAsset must return a result")
		if mockTests {
			assert.Equal(t, &UnstakeAssetResponse{Result: "SUCCESS"}, result, "UnstakeAsset should decode the unstake")
		}
	})
}

func TestGetStakingBalance(t *testing.T) {
	t.Parallel()

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			asset currency.Code
			exp   url.Values
		}{
			{name: "every asset", exp: url.Values{}},
			{name: "one asset", asset: currency.NewCode("bnb"), exp: url.Values{"asset": {"BNB"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/staking/stakingBalance", tc.exp, `{"code":"000000","message":"success","data":[],"status":["holding"],"success":true}`)
				_, err := ex.GetStakingBalance(t.Context(), tc.asset)
				require.NoError(t, err, "GetStakingBalance must not error")
			})
		}
	})

	t.Run("returns the staking balances", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetStakingBalance(t.Context(), currency.BNB)
		require.NoError(t, err, "GetStakingBalance must not error")
		require.NotNil(t, result, "GetStakingBalance must return a result")
		if mockTests {
			exp := &StakingBalanceResponse{
				Code:    "000000",
				Message: "success",
				Data: []StakingBalance{{
					Asset:                 currency.BNB,
					StakingAmount:         3882.50133916,
					RewardAsset:           currency.BNB,
					AnnualPercentageRate:  0.0517,
					AnnualPercentageYield: 0.0869,
					AutoRestake:           true,
				}},
				Success: true,
				Status:  []string{"holding"},
			}
			assert.Equal(t, exp, result, "GetStakingBalance should decode the staking balances")
		}
	})
}

func TestGetStakingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetStakingHistory(t.Context(), &StakingHistoryRequest{StartTime: time.UnixMilli(1665462090000), EndTime: time.UnixMilli(1663096490000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetStakingHistory must refuse a reversed time range")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *StakingHistoryRequest
			exp  url.Values
		}{
			{name: "nil request", exp: url.Values{}},
			{
				name: "every filter",
				req:  &StakingHistoryRequest{Asset: currency.NewCode("eth"), StartTime: time.UnixMilli(1663096490000), EndTime: time.UnixMilli(1665462090000), Page: 2, Limit: 100},
				exp:  url.Values{"asset": {"ETH"}, "startTime": {"1663096490000"}, "endTime": {"1665462090000"}, "page": {"2"}, "limit": {"100"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/staking/history", tc.exp, `[]`)
				_, err := ex.GetStakingHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetStakingHistory must not error")
			})
		}
	})

	t.Run("returns the staking transactions", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetStakingHistory(t.Context(), &StakingHistoryRequest{Asset: currency.ETH, StartTime: time.UnixMilli(1663096490000), EndTime: time.UnixMilli(1665462090000), Page: 1, Limit: 500})
		require.NoError(t, err, "GetStakingHistory must not error")
		if mockTests {
			exp := []StakingRecordResponse{
				{Asset: currency.ETH, Amount: 2500, Type: "staked", InitiatedTime: types.Time(time.UnixMilli(1663096490748)), Status: "SUCCESS"},
				{Asset: currency.ETH, Amount: 1, Type: "staked", InitiatedTime: types.Time(time.UnixMilli(1665462088011)), Status: "SUCCESS"},
			}
			assert.Equal(t, exp, result, "GetStakingHistory should decode the staking transactions")
		}
	})
}

func TestGetStakingRewardsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetStakingRewardsHistory(t.Context(), &StakingHistoryRequest{StartTime: time.UnixMilli(1658475133593), EndTime: time.UnixMilli(1658475133000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetStakingRewardsHistory must refuse a reversed time range")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *StakingHistoryRequest
			exp  url.Values
		}{
			{name: "nil request", exp: url.Values{}},
			{
				name: "every filter",
				req:  &StakingHistoryRequest{Asset: currency.NewCode("bnb"), StartTime: time.UnixMilli(1658475133000), EndTime: time.UnixMilli(1658475133593), Page: 3, Limit: 50},
				exp:  url.Values{"asset": {"BNB"}, "startTime": {"1658475133000"}, "endTime": {"1658475133593"}, "page": {"3"}, "limit": {"50"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/staking/stakingRewardsHistory", tc.exp, `{"code":"000000","message":"success","data":[],"total":0,"success":true}`)
				_, err := ex.GetStakingRewardsHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetStakingRewardsHistory must not error")
			})
		}
	})

	t.Run("returns the staking rewards", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetStakingRewardsHistory(t.Context(), &StakingHistoryRequest{Asset: currency.BNB, StartTime: time.UnixMilli(1658475133000), EndTime: time.UnixMilli(1658475133593)})
		require.NoError(t, err, "GetStakingRewardsHistory must not error")
		require.NotNil(t, result, "GetStakingRewardsHistory must return a result")
		if mockTests {
			exp := &StakingRewardsHistoryResponse{
				Code:    "000000",
				Message: "success",
				Data: []StakingRewardRecord{{
					Asset:         currency.BNB,
					Amount:        0.03371769,
					USDValue:      20.23,
					Time:          types.Time(time.UnixMilli(1658475133009)),
					TransactionID: 3123201,
					AutoRestaked:  true,
				}},
				Success: true,
				Total:   1,
			}
			assert.Equal(t, exp, result, "GetStakingRewardsHistory should decode the staking rewards")
		}
	})
}

func TestStakingHistoryParams(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1663096490000), time.UnixMilli(1665462090000)
	for _, tc := range []struct {
		name string
		req  *StakingHistoryRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", exp: url.Values{}},
		{name: "no filters", req: &StakingHistoryRequest{}, exp: url.Values{}},
		{name: "asset only", req: &StakingHistoryRequest{Asset: currency.NewCode("eth")}, exp: url.Values{"asset": {"ETH"}}},
		{name: "start time only", req: &StakingHistoryRequest{StartTime: start}, exp: url.Values{"startTime": {"1663096490000"}}},
		{name: "end time only", req: &StakingHistoryRequest{EndTime: end}, exp: url.Values{"endTime": {"1665462090000"}}},
		{
			name: "every filter",
			req:  &StakingHistoryRequest{Asset: currency.ETH, StartTime: start, EndTime: end, Page: 2, Limit: 500},
			exp:  url.Values{"asset": {"ETH"}, "startTime": {"1663096490000"}, "endTime": {"1665462090000"}, "page": {"2"}, "limit": {"500"}},
		},
		{name: "reversed time range", req: &StakingHistoryRequest{StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := stakingHistoryParams(tc.req)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "stakingHistoryParams must refuse the request")
				return
			}
			require.NoError(t, err, "stakingHistoryParams must not error")
			assert.Equal(t, tc.exp, params, "stakingHistoryParams should set the request's filters")
		})
	}
}
