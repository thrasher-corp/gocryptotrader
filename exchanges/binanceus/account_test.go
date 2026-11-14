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
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestGetAccount(t *testing.T) {
	t.Parallel()
	skipIfCredentialsUnset(t)
	result, err := e.GetAccount(t.Context())
	require.NoError(t, err, "GetAccount must not error")
	require.NotNil(t, result, "GetAccount must return a result")
	if !mockTests {
		assert.NotEmpty(t, result.AccountType, "GetAccount should return the account type")
		return
	}
	exp := &AccountResponse{
		MakerCommissionInBasisPoints:  15,
		TakerCommissionInBasisPoints:  15,
		BuyerCommissionInBasisPoints:  1,
		SellerCommissionInBasisPoints: -2,
		CommissionRates:               CommissionRates{Maker: 0.0015, Taker: 0.0015, Buyer: 0.0001, Seller: -0.0002},
		CanTrade:                      true,
		CanWithdraw:                   true,
		CanDeposit:                    true,
		Brokered:                      true,
		RequireSelfTradePrevention:    true,
		UpdateTime:                    types.Time(time.UnixMilli(1794464635319)),
		AccountType:                   "SPOT",
		Balances: []Balance{
			{Asset: currency.BTC, Free: decimal.MustFromString("0.00123456"), Locked: decimal.MustFromString("0.0001")},
			{Asset: currency.USDT, Free: decimal.MustFromString("125.5"), Locked: decimal.Zero},
		},
		Permissions: []Permission{PermissionSpot},
		UID:         123456789,
	}
	require.Len(t, result.Balances, len(exp.Balances), "GetAccount must return every balance")
	for i := range exp.Balances {
		assert.Truef(t, exp.Balances[i].Free.Equal(result.Balances[i].Free), "GetAccount should decode balance %d's free amount exactly", i)
		assert.Truef(t, exp.Balances[i].Locked.Equal(result.Balances[i].Locked), "GetAccount should decode balance %d's locked amount exactly", i)
		exp.Balances[i].Free, exp.Balances[i].Locked = result.Balances[i].Free, result.Balances[i].Locked
	}
	assert.Equal(t, exp, result, "GetAccount should decode every field")
}

func TestGetUserAccountStatus(t *testing.T) {
	t.Parallel()
	skipIfCredentialsUnset(t)
	result, err := e.GetUserAccountStatus(t.Context())
	require.NoError(t, err, "GetUserAccountStatus must not error")
	require.NotNil(t, result, "GetUserAccountStatus must return a result")
	if mockTests {
		exp := &AccountStatusResponse{Message: "Order failed:Low Order fill rate! Will be reactivated after 5 minutes.", Success: true, Objects: []string{"5"}}
		assert.Equal(t, exp, result, "GetUserAccountStatus should decode every field")
	}
}

func TestGetUserAPITradingStatus(t *testing.T) {
	t.Parallel()
	skipIfCredentialsUnset(t)
	result, err := e.GetUserAPITradingStatus(t.Context())
	require.NoError(t, err, "GetUserAPITradingStatus must not error")
	require.NotNil(t, result, "GetUserAPITradingStatus must return a result")
	if mockTests {
		exp := &APITradingStatusResponse{
			Success: true,
			Status: APITradingStatus{
				IsLocked:           true,
				PlannedRecoverTime: types.Time(time.UnixMilli(1794465235319)),
				TriggerCondition:   APITradingTriggerCondition{GTCCancellationRatio: 150, IOCOrFOKExpirationRatio: 150, UnfilledRatio: 300},
				Indicators: map[string][]TradingIndicator{"BTCUSDT": {
					{Indicator: "UFR", Count: 20, CurrentValue: 0.05, TriggerValue: 0.995},
					{Indicator: "IFER", Count: 20, CurrentValue: 0.99, TriggerValue: 0.99},
					{Indicator: "GCR", Count: 20, CurrentValue: 0.99, TriggerValue: 0.99},
				}},
				UpdateTime: types.Time(time.UnixMilli(1794464635319)),
			},
		}
		assert.Equal(t, exp, result, "GetUserAPITradingStatus should decode every field")
	}
}

func TestGetAssetDistributionHistory(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1794464000000)
	_, err := e.GetAssetDistributionHistory(t.Context(), &AssetDistributionHistoryRequest{StartTime: start, EndTime: start.Add(-time.Second)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAssetDistributionHistory must reject a start after the end")

	t.Run("sends the set filters", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"asset": {"BNB"}, "category": {"API Partner Rebate"}, "startTime": {"1794464000000"}, "endTime": {"1794467600000"}, "limit": {"500"}}
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/asset/assetDistributionHistory", exp, `{}`)
		_, err := ex.GetAssetDistributionHistory(t.Context(), &AssetDistributionHistoryRequest{Asset: currency.NewCode("bnb"), Category: "API Partner Rebate", StartTime: start, EndTime: start.Add(time.Hour), Limit: 500})
		require.NoError(t, err, "GetAssetDistributionHistory must not error")
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetAssetDistributionHistory(t.Context(), nil)
	require.NoError(t, err, "GetAssetDistributionHistory must not error")
	require.NotNil(t, result, "GetAssetDistributionHistory must return a result")
	if mockTests {
		exp := &AssetDistributionHistoryResponse{
			Rows:  []AssetDistribution{{Amount: 10, Asset: currency.NewCode("BHFT"), DistributionTime: types.Time(time.UnixMilli(1563189166000)), Category: "BHFT distribution", TransactionID: 2968885920}},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetAssetDistributionHistory should decode every field")
	}
}

func TestGetTradeFee(t *testing.T) {
	t.Parallel()
	t.Run("sends no symbol for every symbol", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/asset/query/trading-fee", url.Values{}, `[]`)
		_, err := ex.GetTradeFee(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetTradeFee must not error")
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetTradeFee(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetTradeFee must not error")
	if mockTests {
		assert.Equal(t, []TradeFeeResponse{{Symbol: "BTCUSDT", MakerCommission: 0.004, TakerCommission: 0.006}}, result, "GetTradeFee should decode every field")
	}
}

func TestGetCommissionRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetCommissionRates(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCommissionRates must reject an empty pair")

	skipIfCredentialsUnset(t)
	result, err := e.GetCommissionRates(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetCommissionRates must not error")
	require.NotNil(t, result, "GetCommissionRates must return a result")
	if mockTests {
		exp := &AccountCommissionResponse{
			Symbol:             "BTCUSDT",
			StandardCommission: CommissionRates{Maker: 0.0000001, Taker: 0.0000002, Buyer: 0.0000003, Seller: 0.0000004},
			TaxCommission:      CommissionRates{Maker: 0.00000112, Taker: 0.00000114, Buyer: 0.00000118, Seller: 0.00000116},
			Discount:           CommissionDiscount{EnabledForAccount: true, EnabledForSymbol: true, DiscountAsset: currency.BNB, Discount: 0.25},
		}
		assert.Equal(t, exp, result, "GetCommissionRates should decode every field")
	}
}

func TestGetPast30DaysTradeVolume(t *testing.T) {
	t.Parallel()
	skipIfCredentialsUnset(t)
	result, err := e.GetPast30DaysTradeVolume(t.Context())
	require.NoError(t, err, "GetPast30DaysTradeVolume must not error")
	require.NotNil(t, result, "GetPast30DaysTradeVolume must return a result")
	if mockTests {
		assert.Equal(t, &TradeVolumeResponse{Past30DaysTradingVolume: 100.5}, result, "GetPast30DaysTradeVolume should decode the volume")
	}
}

func TestGetMyFilters(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyFilters(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMyFilters must reject an empty pair")

	skipIfCredentialsUnset(t)
	result, err := e.GetMyFilters(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetMyFilters must not error")
	require.NotNil(t, result, "GetMyFilters must return a result")
	if mockTests {
		exp := &MyFiltersResponse{
			ExchangeFilters: []ExchangeFilter{
				{FilterType: "EXCHANGE_MAX_NUM_ORDERS", MaxNumberOrders: 1000},
				{FilterType: "EXCHANGE_MAX_NUM_ALGO_ORDERS", MaxNumberAlgoOrders: 200},
				{FilterType: "EXCHANGE_MAX_NUM_ICEBERG_ORDERS", MaxNumberIcebergOrders: 10000},
			},
			SymbolFilters: []SymbolFilter{{FilterType: "MAX_NUM_ORDER_LISTS", MaxNumberOrderLists: 20}},
			AssetFilters:  []AssetFilter{{FilterType: "MAX_ASSET", Asset: currency.USD, Limit: 1000000}},
		}
		assert.Equal(t, exp, result, "GetMyFilters should decode every field")
	}
}
