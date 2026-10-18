package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetBrokerWhitelistIP(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerWhitelistIP(t.Context())
	require.NoError(t, err, "GetBrokerWhitelistIP must not error")
	if mockTests {
		exp := []BrokerWhitelistIP{
			{
				Name:           "CoinXXX",
				EndpointDomain: "cabxxx0bnh.bybit-aws.com",
				IPList: []string{
					"192.0.2.10",
					"198.51.100.1",
				},
			},
			{
				Name:           "CoinXXX",
				EndpointDomain: "vgjzztjbx91ajfj.bybit.com",
				IPList: []string{
					"192.0.2.10",
					"198.51.100.1",
					"198.51.100.2",
				},
			},
		}
		assert.Equal(t, exp, result, "GetBrokerWhitelistIP should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerWhitelistIP should return a result")
}

func TestGetBrokerIPChangeLog(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerIPChangeLog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBrokerIPChangeLog must reject a nil request")
	_, err = e.GetBrokerIPChangeLog(t.Context(), &GetBrokerIPChangeLogRequest{StartTime: time.UnixMilli(1779362000000), EndTime: time.UnixMilli(1779000000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBrokerIPChangeLog must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerIPChangeLog(t.Context(), &GetBrokerIPChangeLogRequest{
		StartTime: time.UnixMilli(1779000000000),
		EndTime:   time.UnixMilli(1779362000000),
		Limit:     3,
		Cursor:    "eyJpZCI6NDE3LCJjcmVhdGVkQXQiOiIyMDI2LTA1LTIxVDA2OjAyOjM2In0=",
	})
	require.NoError(t, err, "GetBrokerIPChangeLog must not error")
	if mockTests {
		exp := &BrokerIPChangeLogResponse{
			List: []BrokerIPChange{
				{
					OperationType: "DISABLE",
					IPAddress:     "203.0.113.47",
					DomainType:    "WS",
					ChangeReason:  "do not meet the requirement",
					CreatedAt:     time.Date(2026, 5, 21, 6, 2, 36, 0, time.UTC),
				},
				{
					OperationType: "ADD",
					IPAddress:     "198.51.100.128/25",
					DomainType:    "GW",
					ChangeReason:  "client request",
					CreatedAt:     time.Date(2026, 5, 20, 9, 15, 0, 0, time.UTC),
				},
			},
			NextPageCursor: "eyJpZCI6NDE0LCJjcmVhdGVkQXQiOiIyMDI2LTA1LTIxVDA2OjAyOjM2In0=",
		}
		assert.Equal(t, exp, result, "GetBrokerIPChangeLog should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerIPChangeLog should return a result")
}

func TestGetExchangeBrokerEarning(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerEarning(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExchangeBrokerEarning must reject a nil request")
	begin, end := time.Date(2023, 11, 29, 0, 0, 0, 0, time.UTC), time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC)
	_, err = e.GetExchangeBrokerEarning(t.Context(), &GetExchangeBrokerEarningRequest{Begin: begin})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetExchangeBrokerEarning must reject a begin date without an end date")
	_, err = e.GetExchangeBrokerEarning(t.Context(), &GetExchangeBrokerEarningRequest{End: end})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetExchangeBrokerEarning must reject an end date without a begin date")
	_, err = e.GetExchangeBrokerEarning(t.Context(), &GetExchangeBrokerEarningRequest{Begin: end, End: begin})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetExchangeBrokerEarning must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExchangeBrokerEarning(t.Context(), &GetExchangeBrokerEarningRequest{
		BusinessType: "SPOT",
		Begin:        begin,
		End:          end,
		UID:          "100000002",
		Limit:        50,
		Cursor:       "eyJtaW5JRCI6MTYwMjM3MjMxfQ==",
	})
	require.NoError(t, err, "GetExchangeBrokerEarning must not error")
	if mockTests {
		exp := &ExchangeBrokerEarningResponse{
			TotalEarningCategory: ExchangeBrokerEarningCategories{
				Spot: []BrokerCoinEarning{
					{
						Coin:    currency.USDT,
						Earning: 0.0021,
					},
				},
				Derivatives: []BrokerCoinEarning{
					{
						Coin:    currency.USDT,
						Earning: 0.0154,
					},
				},
				Options: []BrokerCoinEarning{
					{
						Coin:    currency.USDC,
						Earning: 0.0007,
					},
				},
				Convert: []BrokerCoinEarning{
					{
						Coin:    currency.USDT,
						Earning: 0.0012,
					},
				},
				FiatConvert: []BrokerCoinEarning{
					{
						Coin:    currency.USDT,
						Earning: 0.00054991,
					},
				},
				Total: []BrokerCoinEarning{
					{
						Coin:    currency.USDT,
						Earning: 0.01924991,
					},
					{
						Coin:    currency.USDC,
						Earning: 0.0007,
					},
				},
			},
			Details: []ExchangeBrokerEarningDetail{
				{
					UserID:         "100000002",
					BusinessType:   "SPOT",
					Symbol:         "BTCUSDT",
					Coin:           currency.USDT,
					Earning:        0.0021,
					MarkupEarning:  0.0012,
					BaseFeeEarning: 0.0009,
					OrderID:        "f26255a9-3db4-43d6-8c4a-9b31e295a16b",
					ExecutionID:    "35c3ac53-2d71-5e0a-9098-69aa5c911a65",
					ExecutionTime:  types.Time(time.UnixMilli(1701216000123)),
				},
			},
			NextPageCursor: "eyJtaW5JRCI6MTYwMjM3MjMyfQ==",
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerEarning should decode every field")
		return
	}
	assert.NotNil(t, result, "GetExchangeBrokerEarning should return a result")
}

func TestGetExchangeBrokerAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExchangeBrokerAccountInfo(t.Context())
	require.NoError(t, err, "GetExchangeBrokerAccountInfo must not error")
	if mockTests {
		exp := &ExchangeBrokerAccountInfoResponse{
			SubAccountQuantity:    2,
			MaxSubAccountQuantity: 20,
			BaseFeeRebateRate: ExchangeBrokerBaseFeeRebateRate{
				Spot:        "10.0%",
				Derivatives: "10.0%",
			},
			MarkupFeeRebateRate: ExchangeBrokerMarkupFeeRebateRate{
				Spot:        "6.00%",
				Derivatives: "9.00%",
				Convert:     "3.00%",
			},
			Timestamp: types.Time(time.UnixMilli(1701395633402)),
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerAccountInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.MaxSubAccountQuantity, "GetExchangeBrokerAccountInfo should return the subaccount quota")
}

func TestGetBrokerSubAccountDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerSubAccountDepositRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBrokerSubAccountDepositRecords must reject a nil request")
	_, err = e.GetBrokerSubAccountDepositRecords(t.Context(), &GetBrokerSubAccountDepositRecordsRequest{StartTime: time.UnixMilli(1742800000000), EndTime: time.UnixMilli(1742500000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBrokerSubAccountDepositRecords must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerSubAccountDepositRecords(t.Context(), &GetBrokerSubAccountDepositRecordsRequest{ID: "160237231", TransactionID: "04bf3fbad2fc85b107a42cfdc5ff83110092b606ca754efa0f032f8b94b3262e"})
	require.NoError(t, err, "GetBrokerSubAccountDepositRecords must not error")
	if mockTests {
		require.Len(t, result.Rows, 1, "GetBrokerSubAccountDepositRecords must return the deposit")
		assert.Equal(t, "160237231", result.Rows[0].ID, "GetBrokerSubAccountDepositRecords should filter by ID")
	}
	result, err = e.GetBrokerSubAccountDepositRecords(t.Context(), &GetBrokerSubAccountDepositRecordsRequest{
		SubMemberID: "100000002",
		Coin:        currency.USDT,
		StartTime:   time.UnixMilli(1742500000000),
		EndTime:     time.UnixMilli(1742800000000),
		Limit:       2,
		Cursor:      "eyJtaW5JRCI6MTYwMjM3MjkwfQ==",
	})
	require.NoError(t, err, "GetBrokerSubAccountDepositRecords must not error")
	if mockTests {
		exp := &BrokerSubAccountDepositRecordsResponse{
			Rows: []BrokerSubAccountDeposit{
				{
					ID:                  "160237290",
					SubMemberID:         "100000002",
					Coin:                currency.USDT,
					Chain:               "ETH",
					Amount:              250,
					TransactionID:       "0x6a3c7f0d5b8e2a91c4d7f3b0e9a1c5d8f2b4e6a8c0d2f4b6e8a0c2d4f6b8e0a2",
					Status:              1,
					ToAddress:           "0x9f2c3e4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e",
					Tag:                 "memo-1",
					Confirmations:       3,
					BlockHash:           "0x1b2c3d4e5f60718293a4b5c6d7e8f9011223344556677889900aabbccddeeff0",
					BatchReleaseLimit:   1000000,
					DepositType:         "20",
					TaxDepositRecordsID: "8821",
					TaxStatus:           1,
					TravelRuleStatus:    1,
				},
				{
					ID:                  "160237231",
					SubMemberID:         "100000002",
					Coin:                currency.USDT,
					Chain:               "TRX",
					Amount:              999.0496,
					TransactionID:       "04bf3fbad2fc85b107a42cfdc5ff83110092b606ca754efa0f032f8b94b3262e",
					Status:              3,
					ToAddress:           "TDGYpm5zPacnEqKV34TJPuhJhHom9hcXAy",
					DepositFee:          1,
					SuccessAt:           types.Time(time.Unix(1742728163, 0)),
					Confirmations:       50,
					TransactionIndex:    2,
					BlockHash:           "000000000436ab4dabc8a4a87beb2262d2d87f6761a825494c4f1d5ae11b27e8",
					BatchReleaseLimit:   -1,
					DepositType:         "0",
					FromAddress:         "TJ7hhYhVhaxNx6BPyq7yFpqZrQULL3JSdb",
					TaxDepositRecordsID: "0",
					TaxStatus:           2,
				},
			},
			NextPageCursor: "eyJtaW5JRCI6MTYwMjM3MjMxfQ==",
		}
		assert.Equal(t, exp, result, "GetBrokerSubAccountDepositRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerSubAccountDepositRecords should return a result")
}

func TestSetBrokerRateLimit(t *testing.T) {
	t.Parallel()
	_, err := e.SetBrokerRateLimit(t.Context(), nil)
	require.ErrorIs(t, err, errBrokerRateLimitsRequired, "SetBrokerRateLimit must reject no rate limits")
	for _, tc := range []struct {
		name  string
		limit BrokerRateLimitRequest
		err   error
	}{
		{name: "UIDs", limit: BrokerRateLimitRequest{BusinessType: "SPOT", Rate: 600}, err: errBrokerUIDsRequired},
		{name: "business type", limit: BrokerRateLimitRequest{UIDs: []string{"100000002"}, Rate: 600}, err: errBrokerBusinessTypeRequired},
		{name: "rate", limit: BrokerRateLimitRequest{UIDs: []string{"100000002"}, BusinessType: "SPOT"}, err: errBrokerRateRequired},
	} {
		_, err = e.SetBrokerRateLimit(t.Context(), []BrokerRateLimitRequest{tc.limit})
		require.ErrorIsf(t, err, tc.err, "SetBrokerRateLimit must reject a rate limit without %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	spot := BrokerRateLimitRequest{UIDs: []string{"100000002"}, BusinessType: "SPOT", Rate: 600}
	result, err := e.SetBrokerRateLimit(t.Context(), []BrokerRateLimitRequest{spot})
	require.NoError(t, err, "SetBrokerRateLimit must not error")
	if !mockTests {
		assert.NotEmpty(t, result.Result, "SetBrokerRateLimit should report each rate limit")
		return
	}
	assert.Equal(t, &SetBrokerRateLimitResponse{Result: []BrokerRateLimitResult{{UIDs: "100000002", BusinessType: "SPOT", Rate: 600, Success: true, Message: "API limit updated successfully"}}}, result, "SetBrokerRateLimit should decode every field")
	result, err = e.SetBrokerRateLimit(t.Context(), []BrokerRateLimitRequest{spot, {UIDs: []string{"100000002", "100000003"}, BusinessType: "DERIVATIVES", Rate: 900}})
	assert.ErrorIs(t, err, errBrokerRateLimitNotSet, "SetBrokerRateLimit should report a rate limit it did not set")
	assert.ErrorContains(t, err, "Requested limit exceeds maximum allowed per user", "SetBrokerRateLimit should include the reason")
	exp := &SetBrokerRateLimitResponse{
		Result: []BrokerRateLimitResult{
			{
				UIDs:         "100000002",
				BusinessType: "SPOT",
				Rate:         600,
				Success:      true,
				Message:      "API limit updated successfully",
			},
			{
				UIDs:         "100000002,100000003",
				BusinessType: "DERIVATIVES",
				Rate:         900,
				Message:      "Requested limit exceeds maximum allowed per user",
			},
		},
	}
	assert.Equal(t, exp, result, "SetBrokerRateLimit should return every outcome with the error")
	result, err = e.SetBrokerRateLimit(t.Context(), []BrokerRateLimitRequest{{UIDs: []string{"100000004"}, BusinessType: "SPOT", Rate: 600}})
	assert.ErrorIs(t, err, errAPIResponse, "SetBrokerRateLimit should return an error reply")
	assert.Nil(t, result, "SetBrokerRateLimit should return no outcomes with an error reply")
}

func TestGetBrokerRateLimitCap(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerRateLimitCap(t.Context())
	require.NoError(t, err, "GetBrokerRateLimitCap must not error")
	if mockTests {
		exp := &BrokerRateLimitCapResponse{
			List: []BrokerRateLimitCap{
				{
					BusinessType:      "SPOT",
					TotalRate:         825,
					ExchangeBrokerCap: 900,
					UIDCap:            100,
				},
				{
					BusinessType:      "OPTIONS",
					TotalRate:         760,
					ExchangeBrokerCap: 900,
					UIDCap:            100,
				},
				{
					BusinessType:      "DERIVATIVES",
					TotalRate:         760,
					ExchangeBrokerCap: 900,
					UIDCap:            100,
				},
			},
		}
		assert.Equal(t, exp, result, "GetBrokerRateLimitCap should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerRateLimitCap should return a result")
}

func TestGetBrokerAllRateLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerAllRateLimits(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBrokerAllRateLimits must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerAllRateLimits(t.Context(), &GetBrokerAllRateLimitsRequest{Limit: 3, Cursor: "eyJpZCI6M30=", UIDs: []string{"100000002", "100000003"}})
	require.NoError(t, err, "GetBrokerAllRateLimits must not error")
	if mockTests {
		exp := &BrokerAllRateLimitsResponse{
			NextPageCursor: "eyJpZCI6Nn0=",
			List: []BrokerRateLimit{
				{
					UIDs:         "100000002,100000003",
					BusinessType: "SPOT",
					Rate:         223,
				},
				{
					UIDs:         "100000002",
					BusinessType: "OPTIONS",
					Rate:         223,
				},
				{
					UIDs:         "100000003",
					BusinessType: "DERIVATIVES",
					Rate:         298,
				},
			},
		}
		assert.Equal(t, exp, result, "GetBrokerAllRateLimits should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerAllRateLimits should return a result")
}

func TestGetBrokerVoucherSpec(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerVoucherSpec(t.Context(), "")
	require.ErrorIs(t, err, errBrokerVoucherIDRequired, "GetBrokerVoucherSpec must reject an empty voucher ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerVoucherSpec(t.Context(), "80209")
	require.NoError(t, err, "GetBrokerVoucherSpec must not error")
	if mockTests {
		exp := &BrokerVoucherSpecResponse{
			ID:             "80209",
			Coin:           currency.USDT,
			AmountUnit:     "AWARD_AMOUNT_UNIT_USD",
			ProductLine:    "PRODUCT_LINE_CONTRACT",
			SubProductLine: "SUB_PRODUCT_LINE_CONTRACT_DEFAULT",
			TotalAmount:    10000,
			UsedAmount:     100,
		}
		assert.Equal(t, exp, result, "GetBrokerVoucherSpec should decode every field")
		return
	}
	assert.NotEmpty(t, result.ID, "GetBrokerVoucherSpec should return the voucher")
}

func TestIssueBrokerVoucher(t *testing.T) {
	t.Parallel()
	err := e.IssueBrokerVoucher(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "IssueBrokerVoucher must reject a nil request")
	for _, tc := range []struct {
		name   string
		mutate func(*IssueBrokerVoucherRequest)
		err    error
	}{
		{name: "account ID", mutate: func(r *IssueBrokerVoucherRequest) { r.AccountID = "" }, err: errBrokerAccountIDRequired},
		{name: "voucher ID", mutate: func(r *IssueBrokerVoucherRequest) { r.AwardID = "" }, err: errBrokerVoucherIDRequired},
		{name: "spec code", mutate: func(r *IssueBrokerVoucherRequest) { r.SpecCode = "" }, err: errBrokerSpecCodeRequired},
		{name: "amount", mutate: func(r *IssueBrokerVoucherRequest) { r.Amount = 0 }, err: errBrokerVoucherAmountRequired},
		{name: "broker ID", mutate: func(r *IssueBrokerVoucherRequest) { r.BrokerID = "" }, err: errBrokerIDRequired},
	} {
		r := &IssueBrokerVoucherRequest{AccountID: "100000002", AwardID: "123456", SpecCode: "award001", Amount: 100, BrokerID: "v-28478"}
		tc.mutate(r)
		require.ErrorIsf(t, e.IssueBrokerVoucher(t.Context(), r), tc.err, "IssueBrokerVoucher must reject a request without %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.IssueBrokerVoucher(t.Context(), &IssueBrokerVoucherRequest{AccountID: "100000002", AwardID: "123456", SpecCode: "award001", Amount: 100, BrokerID: "v-28478"})
	assert.NoError(t, err, "IssueBrokerVoucher should not error")
}

func TestGetBrokerIssuedVoucher(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerIssuedVoucher(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBrokerIssuedVoucher must reject a nil request")
	for _, tc := range []struct {
		name   string
		mutate func(*GetBrokerIssuedVoucherRequest)
		err    error
	}{
		{name: "account ID", mutate: func(r *GetBrokerIssuedVoucherRequest) { r.AccountID = "" }, err: errBrokerAccountIDRequired},
		{name: "voucher ID", mutate: func(r *GetBrokerIssuedVoucherRequest) { r.AwardID = "" }, err: errBrokerVoucherIDRequired},
		{name: "spec code", mutate: func(r *GetBrokerIssuedVoucherRequest) { r.SpecCode = "" }, err: errBrokerSpecCodeRequired},
	} {
		r := &GetBrokerIssuedVoucherRequest{AccountID: "100000002", AwardID: "189528", SpecCode: "demo000"}
		tc.mutate(r)
		_, err = e.GetBrokerIssuedVoucher(t.Context(), r)
		require.ErrorIsf(t, err, tc.err, "GetBrokerIssuedVoucher must reject a request without %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerIssuedVoucher(t.Context(), &GetBrokerIssuedVoucherRequest{AccountID: "100000002", AwardID: "189528", SpecCode: "demo000", WithUsedAmount: true})
	require.NoError(t, err, "GetBrokerIssuedVoucher must not error")
	if mockTests {
		exp := &BrokerIssuedVoucherResponse{
			AccountID:     "100000002",
			AwardID:       "189528",
			SpecCode:      "demo000",
			Amount:        1,
			IsClaimed:     true,
			StartAt:       types.Time(time.Unix(1725926400, 0)),
			EndAt:         types.Time(time.Unix(1733788800, 0)),
			EffectiveAt:   types.Time(time.Unix(1726531200, 0)),
			IneffectiveAt: types.Time(time.Unix(1733817600, 0)),
			UsedAmount:    0.25,
		}
		assert.Equal(t, exp, result, "GetBrokerIssuedVoucher should decode every field")
		return
	}
	assert.NotEmpty(t, result.AwardID, "GetBrokerIssuedVoucher should return the voucher")
}
