package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetStakingCoins(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStakingCoins(t.Context(), "")
	require.NoError(t, err, "GetStakingCoins must not error")
	if mockTests {
		exp := []*StakingCoinResponse{{
			ProductID:           1,
			Currencies:          currency.Currencies{currency.GT},
			EstimatedAPR:        36,
			MinStakeAmount:      1,
			MaxStakeAmount:      700,
			ProtocolName:        "Gatechain",
			ExchangeRate:        1,
			ExchangeRateReserve: 1,
			ExtraInterest: []*StakingExtraInterest{{
				StartTime:  types.Time(time.Unix(1749427201, 0)),
				EndTime:    types.Time(time.Unix(1765497600, 0)),
				RewardCoin: currency.GT,
				SegmentInterest: []*StakingSegmentInterest{
					{MoneyMax: 1000, MoneyRate: 10},
					{MoneyMin: 1000, MoneyMax: 2000, MoneyRate: 15},
				},
			}},
			CurrencyRewards: []*StakingCurrencyReward{{APR: 6, RewardCoin: currency.NewCode("GT2"), RewardDelayDays: -1, InterestDelayDays: 1}},
		}}
		assert.Equal(t, exp, result, "GetStakingCoins should decode every field")
		return
	}
	assert.NotNil(t, result, "GetStakingCoins should return a result")
}

func TestSwapStakingCoins(t *testing.T) {
	t.Parallel()
	_, err := e.SwapStakingCoins(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SwapStakingCoins must reject a nil request")

	_, err = e.SwapStakingCoins(t.Context(), &StakingSwapRequest{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SwapStakingCoins must require a coin")

	_, err = e.SwapStakingCoins(t.Context(), &StakingSwapRequest{Coin: currency.ETH, Side: 2, Amount: 1})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "SwapStakingCoins must reject a side other than stake or redeem")

	_, err = e.SwapStakingCoins(t.Context(), &StakingSwapRequest{Coin: currency.ETH})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "SwapStakingCoins must require an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SwapStakingCoins(t.Context(), &StakingSwapRequest{Coin: currency.ETH, Amount: 0.01})
	require.NoError(t, err, "SwapStakingCoins must not error")
	if mockTests {
		exp := &StakingSwapResponse{
			ID:             21000,
			ProductID:      2,
			UserID:         12345,
			Coin:           currency.ETH,
			Subtype:        "0",
			Amount:         0.01,
			ExchangeRate:   1,
			ExchangeAmount: 0.01,
			UpdateTime:     types.Time(time.Unix(1752200661, 0)),
			CreateTime:     types.Time(time.Unix(1752200661, 0)),
			Status:         1,
			Source:         "api",
		}
		assert.Equal(t, exp, result, "SwapStakingCoins should decode every field")
		return
	}
	assert.NotNil(t, result, "SwapStakingCoins should return the order")
}

func TestGetStakingOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	stake := &StakingOrderItem{
		ProductID:      2,
		Coin:           currency.SOL,
		Amount:         1,
		Status:         1,
		CreateTime:     types.Time(time.Unix(1756105456, 0)),
		ExchangeAmount: 1,
	}
	result, err := e.GetStakingOrders(t.Context(), currency.EMPTYCODE, 0, "", 0)
	require.NoError(t, err, "GetStakingOrders must not error")
	if mockTests {
		exp := &StakingOrdersResponse{
			Page:       1,
			PageSize:   20,
			PageCount:  1,
			TotalCount: 2,
			List: []*StakingOrderItem{stake, {
				ProductID:      2,
				Coin:           currency.SOL,
				Amount:         1,
				Type:           1,
				Status:         3,
				RedeemTime:     types.Time(time.Unix(1756191856, 0)),
				CreateTime:     types.Time(time.Unix(1755588122, 0)),
				ExchangeAmount: 0.8,
				Fee:            0.001,
			}},
		}
		assert.Equal(t, exp, result, "GetStakingOrders should decode every field")
	} else {
		assert.NotNil(t, result, "GetStakingOrders should return a result")
	}

	// Type 0 selects staking orders, so it must reach the API rather than be dropped as empty
	result, err = e.GetStakingOrders(t.Context(), currency.SOL, 2, "0", 1)
	require.NoError(t, err, "GetStakingOrders must not error when filtering staking orders")
	if mockTests {
		exp := &StakingOrdersResponse{Page: 1, PageSize: 20, PageCount: 1, TotalCount: 1, List: []*StakingOrderItem{stake}}
		assert.Equal(t, exp, result, "GetStakingOrders should decode every field when filtering staking orders")
		return
	}
	assert.NotNil(t, result, "GetStakingOrders should return a result when filtering staking orders")
}

func TestGetStakingDividendRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStakingDividendRecords(t.Context(), currency.EMPTYCODE, 0, 10)
	require.NoError(t, err, "GetStakingDividendRecords must not error")
	if mockTests {
		exp := &StakingDividendRecordsResponse{
			Page:       10,
			PageSize:   20,
			PageCount:  10,
			TotalCount: 181,
			List: []*StakingDividendRecord{{
				ProductID:          64,
				MortgageCoin:       currency.USDT,
				Amount:             0.00000191,
				RewardCoin:         currency.COMP,
				Interest:           0.00000191,
				Fee:                0.00000007,
				Status:             4,
				BonusDate:          types.DateTime(time.Date(2025, 8, 8, 0, 0, 0, 0, time.UTC)),
				ScheduledBonusTime: types.Time(time.Unix(1754611200, 0)),
			}},
		}
		assert.Equal(t, exp, result, "GetStakingDividendRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetStakingDividendRecords should return a result")
}

func TestGetStakingAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStakingAssets(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "GetStakingAssets must not error")
	if mockTests {
		exp := []*StakingAssetItemResponse{{
			ProductID:            64,
			MortgageCurrencies:   currency.Currencies{currency.USDT},
			MortgageAmount:       1,
			CreateTime:           types.Time(time.Unix(1750764156, 0)),
			Type:                 1,
			Status:               1,
			YesterdayIncomeMulti: []json.RawMessage{},
			RewardCoins: []*StakingRewardCoin{
				{RewardCoin: currency.USDT, InterestDelayDays: 1, RewardDelayDays: -1},
				{RewardCoin: currency.COMP, InterestDelayDays: 1, RewardDelayDays: 15},
			},
			DeFiIncome: &StakingDeFiIncome{Total: []*StakingCoinAmount{{Coin: currency.COMP, Amount: 0.00000762}}},
		}}
		assert.Equal(t, exp, result, "GetStakingAssets should decode every field")
		return
	}
	assert.NotNil(t, result, "GetStakingAssets should return a result")
}
