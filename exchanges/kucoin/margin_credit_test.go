package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetLendingCurrencyInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	eth := LendingCurrencyInfo{
		Currency:           currency.ETH,
		PurchaseEnable:     true,
		Increment:          0.0000001,
		MinPurchaseSize:    0.01,
		MinInterestRate:    0.005,
		MaxInterestRate:    0.32,
		InterestIncrement:  0.0001,
		MaxPurchaseSize:    500,
		MarketInterestRate: 0.0073,
		AutoPurchaseEnable: true,
	}
	for _, tc := range []struct {
		name string
		ccy  currency.Code
		exp  []LendingCurrencyInfo
	}{
		{
			name: "all currencies",
			exp: []LendingCurrencyInfo{
				{
					Currency:           currency.BTC,
					PurchaseEnable:     true,
					RedeemEnable:       true,
					Increment:          0.00000001,
					MinPurchaseSize:    0.001,
					MinInterestRate:    0.005,
					MaxInterestRate:    0.32,
					InterestIncrement:  0.0001,
					MaxPurchaseSize:    40,
					MarketInterestRate: 0.005,
				},
				eth,
			},
		},
		{name: "one currency", ccy: currency.ETH, exp: []LendingCurrencyInfo{eth}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetLendingCurrencyInformation(t.Context(), tc.ccy)
			require.NoError(t, err, "GetLendingCurrencyInformation must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetLendingCurrencyInformation should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetLendingCurrencyInformation should return lending currencies")
		})
	}
}

func TestGetLoanMarketInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoanMarketInterestRate(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLoanMarketInterestRate must reject an empty currency")

	result, err := e.GetLoanMarketInterestRate(t.Context(), currency.ETH)
	require.NoError(t, err, "GetLoanMarketInterestRate must not error")
	if mockTests {
		exp := []LoanMarketInterestRate{
			{Time: LoanMarketInterestRateTime(time.Date(2026, 10, 13, 11, 0, 0, 0, time.UTC)), MarketInterestRate: 0.0001},
			{Time: LoanMarketInterestRateTime(time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)), MarketInterestRate: 0.0001},
			{Time: LoanMarketInterestRateTime(time.Date(2026, 10, 13, 13, 0, 0, 0, time.UTC)), MarketInterestRate: 0.0001},
		}
		assert.Equal(t, exp, result, "GetLoanMarketInterestRate should decode every field")
		return
	}
	require.NotEmpty(t, result, "GetLoanMarketInterestRate must return interest rates")
	// The latest rate is for the current hour, so a misread time zone shows up as an hours-long offset
	assert.WithinDuration(t, time.Now(), result[len(result)-1].Time.Time(), 2*time.Hour, "GetLoanMarketInterestRate should return the current hour's rate last")
}

func TestLoanMarketInterestRateTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var got LoanMarketInterestRateTime
	require.NoError(t, json.Unmarshal([]byte(`"202610132100"`), &got), "Unmarshal must not error")
	assert.Equal(t, time.Date(2026, 10, 13, 13, 0, 0, 0, time.UTC), got.Time(), "Unmarshal should decode the UTC+8 hour as UTC")

	var empty LoanMarketInterestRateTime
	require.NoError(t, json.Unmarshal([]byte(`""`), &empty), "Unmarshal must not error on an empty string")
	assert.True(t, empty.Time().IsZero(), "Unmarshal should decode an empty string as the zero time")

	assert.ErrorIs(t, json.Unmarshal([]byte(`"2026-10-13 21:00"`), &got), errInvalidLoanMarketInterestRateTime, "Unmarshal should reject another time format")
	assert.ErrorIs(t, json.Unmarshal([]byte(`202610132100`), &got), errInvalidLoanMarketInterestRateTime, "Unmarshal should reject a number")
}

func TestMarginLendingSubscription(t *testing.T) {
	t.Parallel()
	_, err := e.MarginLendingSubscription(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "MarginLendingSubscription must reject a nil request")
	_, err = e.MarginLendingSubscription(t.Context(), &MarginLendingSubscriptionRequest{Size: 1, InterestRate: 0.22})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "MarginLendingSubscription must reject an empty currency")
	_, err = e.MarginLendingSubscription(t.Context(), &MarginLendingSubscriptionRequest{Currency: currency.ETH, InterestRate: 0.22})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "MarginLendingSubscription must reject a zero size")
	_, err = e.MarginLendingSubscription(t.Context(), &MarginLendingSubscriptionRequest{Currency: currency.ETH, Size: 1})
	require.ErrorIs(t, err, errMissingInterestRate, "MarginLendingSubscription must reject a zero interest rate")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.MarginLendingSubscription(t.Context(), &MarginLendingSubscriptionRequest{Currency: currency.ETH, Size: 1, InterestRate: 0.22})
		return err
	})
	assert.JSONEq(t, `{"currency":"ETH","size":"1","interestRate":"0.22"}`, sent.body, "MarginLendingSubscription should send the size and rate as the strings KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.MarginLendingSubscription(t.Context(), &MarginLendingSubscriptionRequest{Currency: currency.ETH, Size: 1, InterestRate: 0.22})
	require.NoError(t, err, "MarginLendingSubscription must not error")
	if mockTests {
		exp := &OrderNumberResponse{OrderNumber: "671bafa804c26d000773c533"}
		assert.Equal(t, exp, result, "MarginLendingSubscription should decode every field")
		return
	}
	require.NotNil(t, result, "MarginLendingSubscription must return a result")
	assert.NotEmpty(t, result.OrderNumber, "MarginLendingSubscription should return an order number")
}

func TestModifySubscriptionOrder(t *testing.T) {
	t.Parallel()
	err := e.ModifySubscriptionOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifySubscriptionOrder must reject a nil request")
	err = e.ModifySubscriptionOrder(t.Context(), &ModifySubscriptionOrderRequest{InterestRate: 0.09, PurchaseOrderNumber: "671bafa804c26d000773c533"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ModifySubscriptionOrder must reject an empty currency")
	err = e.ModifySubscriptionOrder(t.Context(), &ModifySubscriptionOrderRequest{Currency: currency.ETH, PurchaseOrderNumber: "671bafa804c26d000773c533"})
	require.ErrorIs(t, err, errMissingInterestRate, "ModifySubscriptionOrder must reject a zero interest rate")
	err = e.ModifySubscriptionOrder(t.Context(), &ModifySubscriptionOrderRequest{Currency: currency.ETH, InterestRate: 0.09})
	require.ErrorIs(t, err, errMissingPurchaseOrderNumber, "ModifySubscriptionOrder must reject an empty purchase order number")

	sent := sentRequest(t, func(ex *Exchange) error {
		return ex.ModifySubscriptionOrder(t.Context(), &ModifySubscriptionOrderRequest{Currency: currency.ETH, InterestRate: 0.09, PurchaseOrderNumber: "671bafa804c26d000773c533"})
	})
	assert.JSONEq(t, `{"currency":"ETH","interestRate":"0.09","purchaseOrderNo":"671bafa804c26d000773c533"}`, sent.body, "ModifySubscriptionOrder should send the rate as the string KuCoin documents")

	if !mockTests {
		t.Skip("the purchase order number is the docs' example, and changing a real purchase order is not safe in a test")
	}
	err = e.ModifySubscriptionOrder(t.Context(), &ModifySubscriptionOrderRequest{Currency: currency.ETH, InterestRate: 0.09, PurchaseOrderNumber: "671bafa804c26d000773c533"})
	assert.NoError(t, err, "ModifySubscriptionOrder should not error")
}

func TestGetSubscriptionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubscriptionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubscriptionOrders must reject a nil request")
	_, err = e.GetSubscriptionOrders(t.Context(), &SubscriptionOrdersRequest{Currency: currency.ETH})
	require.ErrorIs(t, err, errStatusMissing, "GetSubscriptionOrders must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *SubscriptionOrdersRequest
		exp  *SubscriptionOrdersResponse
	}{
		{
			name: "pending",
			req:  &SubscriptionOrdersRequest{Status: "PENDING"},
			exp: &SubscriptionOrdersResponse{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []SubscriptionOrder{
					{
						Currency:            currency.BTC,
						PurchaseOrderNumber: "671bb15a3b3f930007880bae",
						PurchaseSize:        0.001,
						InterestRate:        0.1,
						ApplyTime:           types.Time(time.UnixMilli(1729868122172)),
						Status:              "PENDING",
					},
				},
			},
		},
		{
			name: "done",
			req: &SubscriptionOrdersRequest{
				Status:              "DONE",
				Currency:            currency.ETH,
				PurchaseOrderNumber: "671bafa804c26d000773c533",
				CurrentPage:         1,
				PageSize:            20,
			},
			exp: &SubscriptionOrdersResponse{
				CurrentPage: 1,
				PageSize:    20,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []SubscriptionOrder{
					{
						Currency:            currency.ETH,
						PurchaseOrderNumber: "671bafa804c26d000773c533",
						PurchaseSize:        1,
						MatchSize:           1,
						InterestRate:        0.22,
						IncomeSize:          1,
						ApplyTime:           types.Time(time.UnixMilli(1729867688172)),
						Status:              "DONE",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSubscriptionOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetSubscriptionOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSubscriptionOrders should decode every field")
				return
			}
			assert.NotNil(t, result, "GetSubscriptionOrders should return a result")
		})
	}
}

func TestRedemption(t *testing.T) {
	t.Parallel()
	_, err := e.Redemption(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Redemption must reject a nil request")
	_, err = e.Redemption(t.Context(), &RedemptionRequest{Size: 1, PurchaseOrderNumber: "671bafa804c26d000773c533"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "Redemption must reject an empty currency")
	_, err = e.Redemption(t.Context(), &RedemptionRequest{Currency: currency.ETH, PurchaseOrderNumber: "671bafa804c26d000773c533"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "Redemption must reject a zero size")
	_, err = e.Redemption(t.Context(), &RedemptionRequest{Currency: currency.ETH, Size: 1})
	require.ErrorIs(t, err, errMissingPurchaseOrderNumber, "Redemption must reject an empty purchase order number")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.Redemption(t.Context(), &RedemptionRequest{Currency: currency.ETH, Size: 1, PurchaseOrderNumber: "671bafa804c26d000773c533"})
		return err
	})
	assert.JSONEq(t, `{"currency":"ETH","size":"1","purchaseOrderNo":"671bafa804c26d000773c533"}`, sent.body, "Redemption should send the size as the string KuCoin documents")

	if !mockTests {
		t.Skip("the purchase order number is the docs' example, and redeeming a real purchase order is not safe in a test")
	}
	result, err := e.Redemption(t.Context(), &RedemptionRequest{Currency: currency.ETH, Size: 1, PurchaseOrderNumber: "671bafa804c26d000773c533"})
	require.NoError(t, err, "Redemption must not error")
	exp := &OrderNumberResponse{OrderNumber: "671bb01004c26d000773c55c"}
	assert.Equal(t, exp, result, "Redemption should decode every field")
}

func TestGetRedemptionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetRedemptionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRedemptionOrders must reject a nil request")
	_, err = e.GetRedemptionOrders(t.Context(), &RedemptionOrdersRequest{Currency: currency.ETH})
	require.ErrorIs(t, err, errStatusMissing, "GetRedemptionOrders must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *RedemptionOrdersRequest
		exp  *RedemptionOrdersResponse
	}{
		{
			name: "done",
			req:  &RedemptionOrdersRequest{Status: "DONE"},
			exp: &RedemptionOrdersResponse{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []RedemptionOrder{
					{
						Currency:            currency.BTC,
						PurchaseOrderNumber: "671bafa804c26d000773c533",
						RedeemOrderNumber:   "671bb01004c26d000773c55c",
						RedeemSize:          0.001,
						ReceiptSize:         0.001,
						Status:              "DONE",
					},
				},
			},
		},
		{
			name: "pending",
			req: &RedemptionOrdersRequest{
				Status:            "PENDING",
				Currency:          currency.ETH,
				RedeemOrderNumber: "671bb01004c26d000773c55d",
				CurrentPage:       1,
				PageSize:          20,
			},
			exp: &RedemptionOrdersResponse{
				CurrentPage: 1,
				PageSize:    20,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []RedemptionOrder{
					{
						Currency:            currency.ETH,
						PurchaseOrderNumber: "671bafa804c26d000773c533",
						RedeemOrderNumber:   "671bb01004c26d000773c55d",
						RedeemSize:          1,
						ReceiptSize:         0.5,
						ApplyTime:           types.Time(time.UnixMilli(1729868210000)),
						Status:              "PENDING",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRedemptionOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetRedemptionOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetRedemptionOrders should decode every field")
				return
			}
			assert.NotNil(t, result, "GetRedemptionOrders should return a result")
		})
	}
}
