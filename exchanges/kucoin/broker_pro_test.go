package kucoin

import (
	"net/url"
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

func TestApplyForFastWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.ApplyForFastWithdrawal(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ApplyForFastWithdrawal must reject a nil request")
	_, err = e.ApplyForFastWithdrawal(t.Context(), &ApplyForFastWithdrawalRequest{Amount: 1, Address: "0x5439fa013b0b7c2c8fde3281754c464b30dfc111"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ApplyForFastWithdrawal must reject an empty currency")
	_, err = e.ApplyForFastWithdrawal(t.Context(), &ApplyForFastWithdrawalRequest{Currency: currency.USDT, Address: "0x5439fa013b0b7c2c8fde3281754c464b30dfc111"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "ApplyForFastWithdrawal must reject a zero amount")
	_, err = e.ApplyForFastWithdrawal(t.Context(), &ApplyForFastWithdrawalRequest{Currency: currency.USDT, Amount: 1})
	require.ErrorIs(t, err, errAddressRequired, "ApplyForFastWithdrawal must reject an empty address")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.ApplyForFastWithdrawal(t.Context(), &ApplyForFastWithdrawalRequest{
			Currency: currency.USDT, Amount: 25.5, Address: "100000003", Memo: "7025734", Remark: "Invoice 1001", Chain: "kcc",
			WithdrawType: "UID", FeeDeductType: "INTERNAL", IsInner: true, Wallet: "kucoin", TIN: "123456789", Country: "AU",
			FirstName: "Kaylah", LastName: "Padberg", IdentityNumber: "55", IdentityType: "passport",
			CompanyName: "Example Trading Pty Ltd", ReceiverType: "COMPANY", Questionnaire: "Q1:A1",
			ReasonOfTransfer: "Payment for services",
			Validation: &FastWithdrawalValidation{
				Factors:       []FastWithdrawalFactor{{FactorType: "EMV", FactorValue: "142262"}, {FactorType: "GAV", FactorValue: "616566"}},
				TransactionID: "4d6e05a173da36b277e0f3e0",
			},
		})
		return err
	})
	assert.JSONEq(t, `{
		"currency":"USDT","amount":25.5,"address":"100000003","memo":"7025734","remark":"Invoice 1001","chain":"kcc",
		"withdrawType":"UID","feeDeductType":"INTERNAL","isInner":true,"wallet":"kucoin","tin":"123456789","country":"AU",
		"firstName":"Kaylah","lastName":"Padberg","identityNo":"55","identityType":"passport",
		"companyName":"Example Trading Pty Ltd","receiverType":"COMPANY","questionnaire":"Q1:A1",
		"reasonOfTransfer":"Payment for services",
		"validation":{"factors":[{"factorType":"EMV","factorValue":"142262"},{"factorType":"GAV","factorValue":"616566"}],"transactionId":"4d6e05a173da36b277e0f3e0"}
	}`, sent.body, "ApplyForFastWithdrawal should send the amount as the number KuCoin documents")

	if !mockTests {
		t.Skip("a fast withdrawal sends funds to the docs' example address or UID, which a test must not do")
	}
	initial := ApplyForFastWithdrawalRequest{
		Currency:      currency.USDT,
		Amount:        1,
		Address:       "0x5439fa013b0b7c2c8fde3281754c464b30dfc111",
		Chain:         "kcc",
		WithdrawType:  "ADDRESS",
		FeeDeductType: "EXTERNAL",
		Wallet:        "kucoin",
	}
	validated := initial
	validated.Validation = &FastWithdrawalValidation{
		Factors: []FastWithdrawalFactor{
			{FactorType: "EMV", FactorValue: "142262"},
			{FactorType: "GAV", FactorValue: "616566"},
		},
		TransactionID: "4d6e05a173da36b277e0f3e0",
	}
	for _, tc := range []struct {
		name string
		req  *ApplyForFastWithdrawalRequest
		exp  *ApplyForFastWithdrawalResponse
	}{
		{
			name: "initial",
			req:  &initial,
			exp: &ApplyForFastWithdrawalResponse{
				AllFactors:    [][]string{{"EMV", "GAV"}},
				SentFactors:   [][]string{{"EMV"}},
				TransactionID: "4d6e05a173da36b277e0f3e0",
			},
		},
		{
			name: "validated",
			req:  &validated,
			exp:  &ApplyForFastWithdrawalResponse{WithdrawalID: "6a15a46889f75d0007de09f4"},
		},
		{
			name: "inner",
			req: &ApplyForFastWithdrawalRequest{
				Currency:         currency.USDT,
				Amount:           25.5,
				Address:          "100000003",
				Remark:           "Invoice 1001",
				WithdrawType:     "UID",
				FeeDeductType:    "INTERNAL",
				IsInner:          true,
				TIN:              "123456789",
				Country:          "AU",
				FirstName:        "Kaylah",
				LastName:         "Padberg",
				IdentityNumber:   "55",
				IdentityType:     "passport",
				CompanyName:      "Example Trading Pty Ltd",
				ReceiverType:     "COMPANY",
				Questionnaire:    "Q1:A1",
				ReasonOfTransfer: "Payment for services",
			},
			exp: &ApplyForFastWithdrawalResponse{
				AllFactors:    [][]string{{"SMS", "GAV"}},
				SentFactors:   [][]string{{"SMS"}},
				TransactionID: "5e7f16b284eb47c388f1f4f1",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ApplyForFastWithdrawal(t.Context(), tc.req)
			require.NoError(t, err, "ApplyForFastWithdrawal must not error")
			assert.Equal(t, tc.exp, result, "ApplyForFastWithdrawal should decode every field")
		})
	}
}

func TestGetBrokerProRebate(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerProRebate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBrokerProRebate must reject a nil request")
	_, err = e.GetBrokerProRebate(t.Context(), &GetBrokerProRebateRequest{TradeType: "SPOT"})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetBrokerProRebate must reject an unset period")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerProRebate(t.Context(), &GetBrokerProRebateRequest{
		Begin:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		End:       time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		TradeType: "SPOT",
	})
	require.NoError(t, err, "GetBrokerProRebate must not error")
	require.NotNil(t, result, "GetBrokerProRebate must return a result")
	if mockTests {
		exp := &BrokerRebateDownloadResponse{
			URL: "https://kc-v2-promotion.s3.ap-northeast-1.amazonaws.com/broker/671aec522593f600019766d0_file.csv?X-Amz-Security-Token=IQo*********2cd90f14efb",
		}
		assert.Equal(t, exp, result, "GetBrokerProRebate should decode every field")
		return
	}
	assert.NotEmpty(t, result.URL, "GetBrokerProRebate should return a download link")
}

func TestGetBrokerProCommission(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerProCommission(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBrokerProCommission must reject a nil request")
	_, err = e.GetBrokerProCommission(t.Context(), &GetBrokerProCommissionRequest{StartAt: time.UnixMilli(1758729599999), EndAt: time.UnixMilli(1758643200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBrokerProCommission must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerProCommission(t.Context(), &GetBrokerProCommissionRequest{
		SiteType:   "europe",
		TradeType:  "SPOT",
		RebateType: "1",
		StartAt:    time.UnixMilli(1758643200000),
		EndAt:      time.UnixMilli(1758729599999),
		Page:       1,
		PageSize:   1,
	})
	require.NoError(t, err, "GetBrokerProCommission must not error")
	if mockTests {
		exp := &BrokerProCommissionResponse{
			CurrentPage: 1,
			PageSize:    1,
			TotalNumber: 17,
			TotalPage:   17,
			Items: []BrokerProCommission{
				{
					SiteType:         "europe",
					RebateType:       1,
					PayoutTime:       types.Time(time.UnixMilli(1758761640000)),
					PeriodStartTime:  types.Time(time.UnixMilli(1758643200000)),
					PeriodEndTime:    types.Time(time.UnixMilli(1758729599999)),
					Status:           2,
					TotalTradeUser:   1,
					TagUser:          1,
					TagTradeVolume:   2.953,
					TagCommission:    0.0007,
					InvitedUser:      1,
					NoTagTradeVolume: 3,
					NoTagCommission:  0.0008,
					TotalCommission:  0.0015,
					Currency:         currency.USDT,
				},
			},
		}
		assert.Equal(t, exp, result, "GetBrokerProCommission should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerProCommission should return a result")
}

func TestGetBrokerProUserList(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerProUserList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBrokerProUserList must reject a nil request")
	_, err = e.GetBrokerProUserList(t.Context(), &GetBrokerProUserListRequest{StartAt: time.UnixMilli(1716249600000), EndAt: time.UnixMilli(1716163200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBrokerProUserList must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerProUserList(t.Context(), &GetBrokerProUserListRequest{
		TradeType:    "all",
		UID:          "100000001",
		ReferralCode: "TFDAFA",
		Tag:          "TESTKOBE",
		StartAt:      time.UnixMilli(1716163200000),
		EndAt:        time.UnixMilli(1716249600000),
		Page:         1,
		PageSize:     1,
	})
	require.NoError(t, err, "GetBrokerProUserList must not error")
	if mockTests {
		exp := &BrokerProUserListResponse{
			CurrentPage: 1,
			PageSize:    1,
			TotalNumber: 18,
			TotalPage:   18,
			Items: []BrokerProUser{
				{
					UID:                            "100000001",
					NickName:                       "kg901",
					RegistrationTime:               types.Time(time.UnixMilli(1716191866000)),
					RateUp:                         true,
					TotalCommission:                0.015,
					TotalTradingVolume:             20,
					TotalFee:                       1.5,
					InvitedByMe:                    true,
					ReferralCode:                   "TFDAFA",
					Tags:                           "TESTKOBE",
					SpotTradingVolumeWithTag:       10,
					FutureTradingVolumeWithTag:     5,
					TradingFeeWithTag:              1,
					CommissionWithTag:              0.01,
					SpotTradingVolumeWithoutTag:    2,
					FutureTradingVolumeWithoutTag:  3,
					TradingFeeWithoutTag:           0.5,
					CommissionWithoutTag:           0.005,
					Currency:                       currency.USDT,
					FuturesTradingVolumeWithTag:    5,
					FuturesTradingVolumeWithoutTag: 3,
					RefferralCode:                  "TFDAFA",
					CompletedKYC:                   new(types.Boolean(true)),
					CompletedFirstTrade:            new(types.Boolean(true)),
					FirstTradeTime:                 "2025-08-01",
					FirstTradeAmount:               10.12,
					CompletedFirstDeposit:          new(types.Boolean(true)),
					FirstDepositTime:               "2025-08-01",
					FirstDepositAmount:             10.12,
					Country:                        "HK",
				},
			},
		}
		assert.Equal(t, exp, result, "GetBrokerProUserList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerProUserList should return a result")
}

func TestBrokerProUserUnmarshal(t *testing.T) {
	t.Parallel()
	var got BrokerProUser
	err := json.Unmarshal([]byte(`{"uid":"100000002","rcode":null,"completedKYC":null,"completedFirstTrade":false,"firstTradeTime":null,"completedFirstDeposit":"true"}`), &got)
	require.NoError(t, err, "Unmarshal must decode a user whose flags KuCoin leaves null")
	exp := BrokerProUser{UID: "100000002", CompletedFirstTrade: new(types.Boolean(false)), CompletedFirstDeposit: new(types.Boolean(true))}
	assert.Equal(t, exp, got, "Unmarshal should leave a null flag unset and decode boolean and quoted flags")
}

func TestGetBrokerProUserTransactions(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerProUserTransactions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBrokerProUserTransactions must reject a nil request")
	_, err = e.GetBrokerProUserTransactions(t.Context(), &GetBrokerProUserTransactionsRequest{StartAt: time.UnixMilli(1758758400000), EndAt: time.UnixMilli(1758672000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBrokerProUserTransactions must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerProUserTransactions(t.Context(), &GetBrokerProUserTransactionsRequest{
		TradeType: "all",
		UID:       "100000002",
		StartAt:   time.UnixMilli(1758672000000),
		EndAt:     time.UnixMilli(1758758400000),
		LastID:    "68d4138bafa05500017f7700",
		Direction: "NEXT",
		PageSize:  1,
	})
	require.NoError(t, err, "GetBrokerProUserTransactions must not error")
	if mockTests {
		exp := []BrokerProUserTransactionResponse{
			{
				TradeTime:                      types.Time(time.UnixMilli(1758729060000)),
				UID:                            "100000002",
				NickName:                       "tes***@example.com",
				InvitedByMe:                    true,
				TotalCommission:                0.0012,
				TotalVolume:                    5.2045,
				TotalFee:                       0.0036,
				ReferralCode:                   "TESTRCODE",
				SpotTradingVolumeWithTag:       2.9545,
				FutureTradingVolumeWithTag:     1.5,
				TradingFeeWithTag:              0.0029,
				CommissionWithTag:              0.001,
				SpotTradingVolumeWithNoTag:     0.5,
				FutureTradingVolumeWithNoTag:   0.25,
				TradingFeeWithNoTag:            0.0007,
				CommissionWithNoTag:            0.0002,
				Currency:                       currency.USDC,
				LastID:                         "68d4138bafa05500017f7801",
				SpotTradingVolumeWithoutTag:    0.5,
				FuturesTradingVolumeWithoutTag: 0.25,
				FuturesTradingVolumeWithTag:    1.5,
				TradingFeeWithoutTag:           0.0007,
				CommissionWithoutTag:           0.0002,
			},
		}
		assert.Equal(t, exp, result, "GetBrokerProUserTransactions should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerProUserTransactions should return a result")
}

func TestBrokerPeriodParams(t *testing.T) {
	t.Parallel()
	begin, end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	_, err := brokerPeriodParams(time.Time{}, end, "SPOT")
	require.ErrorIs(t, err, common.ErrDateUnset, "brokerPeriodParams must reject an unset begin")
	_, err = brokerPeriodParams(begin, time.Time{}, "SPOT")
	require.ErrorIs(t, err, common.ErrDateUnset, "brokerPeriodParams must reject an unset end")
	_, err = brokerPeriodParams(end, begin, "SPOT")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "brokerPeriodParams must reject a begin after the end")
	_, err = brokerPeriodParams(begin, end, "")
	require.ErrorIs(t, err, errTradeTypeMissing, "brokerPeriodParams must reject an empty trade type")

	params, err := brokerPeriodParams(begin, end, "SPOT")
	require.NoError(t, err, "brokerPeriodParams must not error")
	exp := url.Values{"begin": {"20250101"}, "end": {"20250201"}, "tradeType": {"SPOT"}}
	assert.Equal(t, exp, params, "brokerPeriodParams should format both days")

	params, err = brokerPeriodParams(begin.Add(time.Hour*12), begin, "1")
	require.NoError(t, err, "brokerPeriodParams must accept a period of one day")
	exp = url.Values{"begin": {"20250101"}, "end": {"20250101"}, "tradeType": {"1"}}
	assert.Equal(t, exp, params, "brokerPeriodParams should send a one-day period's day as both begin and end")
}
