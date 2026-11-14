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

func TestGetCreditLineAccountInformation(t *testing.T) {
	t.Parallel()

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v2/cl/account", url.Values{}, `{}`)
		_, err := ex.GetCreditLineAccountInformation(t.Context())
		require.NoError(t, err, "GetCreditLineAccountInformation must not error")
	})

	t.Run("returns the account information", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCreditLineAccountInformation(t.Context())
		require.NoError(t, err, "GetCreditLineAccountInformation must not error")
		require.NotNil(t, result, "GetCreditLineAccountInformation must return a result")
		if mockTests {
			exp := &CreditLineAccountInformationResponse{
				CreditLineAccountID:          351010000,
				MasterAccountID:              351010001,
				NotificationMailAddress:      "email@binance.us",
				CurrentLoanToValue:           0.415,
				InitialLoanToValue:           0.62,
				MarginCallLoanToValue:        0.8,
				LiquidationLoanToValue:       0.54,
				InterestRate:                 0.01,
				LiquidationFeeRate:           0.02,
				ContractStartTime:            types.Time(time.UnixMilli(1676332800000)),
				ContractEndTime:              types.Time(time.UnixMilli(1680220802000)),
				IsActive:                     true,
				CanTrade:                     true,
				CanTransferIn:                true,
				CanTransferOut:               true,
				RequiredDepositAmount:        100,
				AvailableAmountToTransferOut: 500,
				LoanAssets:                   []CreditLineLoanAsset{{LoanAsset: currency.BNB, LoanQuantity: 900000, CurrentLoanQuantity: 5885.70162259, ReclaimSequence: 1}},
				Balances: []CreditLineBalance{
					{Asset: currency.BNB, Free: 14227.31994203, Locked: 0.5},
					{Asset: currency.USD, Free: 948.46},
				},
			}
			assert.Equal(t, exp, result, "GetCreditLineAccountInformation should decode the account information")
		}
	})
}

func TestGetCreditLineAlertHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCreditLineAlertHistory(t.Context(), &CreditLineAlertHistoryRequest{StartTime: time.UnixMilli(1672314508000), EndTime: time.UnixMilli(1671158880000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCreditLineAlertHistory must refuse a reversed time range")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CreditLineAlertHistoryRequest
			exp  url.Values
		}{
			{name: "nil request", exp: url.Values{}},
			{
				name: "every filter",
				req:  &CreditLineAlertHistoryRequest{StartTime: time.UnixMilli(1671158880000), EndTime: time.UnixMilli(1672314508000), Limit: 50, AlertType: CreditLineAlertTypeMarginCall},
				exp:  url.Values{"startTime": {"1671158880000"}, "endTime": {"1672314508000"}, "limit": {"50"}, "alertType": {"MARGIN_CALL"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v2/cl/alertHistory", tc.exp, `[]`)
				_, err := ex.GetCreditLineAlertHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetCreditLineAlertHistory must not error")
			})
		}
	})

	t.Run("returns the alerts", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCreditLineAlertHistory(t.Context(), &CreditLineAlertHistoryRequest{StartTime: time.UnixMilli(1671158880000), EndTime: time.UnixMilli(1672314508000)})
		require.NoError(t, err, "GetCreditLineAlertHistory must not error")
		if mockTests {
			loans := []CreditLineLoanAsset{{LoanAsset: currency.BNB, LoanQuantity: 900000, CurrentLoanQuantity: 5885.70162259, ReclaimSequence: 1}}
			exp := []CreditLineAlertRecordResponse{
				{
					AlertTime:              types.Time(time.UnixMilli(1672314507694)),
					AlertType:              CreditLineAlertTypeLiquidationCall,
					CurrentLoanToValue:     9,
					InitialLoanToValue:     0.62,
					MarginCallLoanToValue:  0.8,
					LiquidationLoanToValue: 0.54,
					TotalBalance:           3710355.24038655,
					LoanAssets:             loans,
					Balances: []CreditLineBalance{
						{Asset: currency.BNB, Free: 14257.81994203, Freeze: 1.5},
						{Asset: currency.USD, Free: 48.46},
					},
				},
				{
					AlertTime:              types.Time(time.UnixMilli(1671158880863)),
					AlertType:              CreditLineAlertTypeMarginCall,
					CurrentLoanToValue:     0.8124,
					InitialLoanToValue:     0.62,
					MarginCallLoanToValue:  0.8,
					LiquidationLoanToValue: 0.54,
					TotalBalance:           5839625.1599878,
					LoanAssets:             loans,
					Balances: []CreditLineBalance{
						{Asset: currency.BNB, Free: 14257.81994203},
						{Asset: currency.USD, Free: 48.46},
					},
				},
			}
			assert.Equal(t, exp, result, "GetCreditLineAlertHistory should decode the alerts")
		}
	})
}

func TestGetCreditLineTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCreditLineTransferHistory(t.Context(), &CreditLineTransferHistoryRequest{StartTime: time.UnixMilli(1676958403384), EndTime: time.UnixMilli(1676676502389)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCreditLineTransferHistory must refuse a reversed time range")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CreditLineTransferHistoryRequest
			exp  url.Values
		}{
			{name: "nil request", exp: url.Values{}},
			{
				name: "every filter",
				req: &CreditLineTransferHistoryRequest{
					StartTime:    time.UnixMilli(1676676502000),
					EndTime:      time.UnixMilli(1676958404000),
					Limit:        100,
					TransferType: CreditLineTransferTypeTransferOut,
					Asset:        currency.NewCode("usd"),
				},
				exp: url.Values{"startTime": {"1676676502000"}, "endTime": {"1676958404000"}, "limit": {"100"}, "transferType": {"TRANSFER_OUT"}, "asset": {"USD"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/cl/transferHistory", tc.exp, `[]`)
				_, err := ex.GetCreditLineTransferHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetCreditLineTransferHistory must not error")
			})
		}
	})

	t.Run("returns the transfers", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCreditLineTransferHistory(t.Context(), &CreditLineTransferHistoryRequest{TransferType: CreditLineTransferTypeTransferIn, Asset: currency.USD})
		require.NoError(t, err, "GetCreditLineTransferHistory must not error")
		if mockTests {
			exp := []CreditLineTransferRecordResponse{
				{TransferID: "t11119", TransferType: CreditLineTransferTypeTransferIn, Asset: currency.USD, Amount: 10, Status: "SUCCESS", TransferTime: types.Time(time.UnixMilli(1676958403384))},
				{TransferID: "t11117", TransferType: CreditLineTransferTypeTransferIn, Asset: currency.USD, Amount: 10, Status: "SUCCESS", TransferTime: types.Time(time.UnixMilli(1676676502389))},
			}
			assert.Equal(t, exp, result, "GetCreditLineTransferHistory should decode the transfers")
		}
	})
}

func TestExecuteCreditLineTransfer(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CreditLineTransferRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no transfer type", req: &CreditLineTransferRequest{TransferAssetType: currency.USD, Quantity: 10}, err: errTransferTypeRequired},
		{name: "no asset", req: &CreditLineTransferRequest{TransferType: CreditLineTransferTypeTransferIn, Quantity: 10}, err: currency.ErrCurrencyCodeEmpty},
		{name: "no quantity", req: &CreditLineTransferRequest{TransferType: CreditLineTransferTypeTransferIn, TransferAssetType: currency.USD}, err: order.ErrAmountIsInvalid},
		{name: "a negative quantity", req: &CreditLineTransferRequest{TransferType: CreditLineTransferTypeTransferIn, TransferAssetType: currency.USD, Quantity: -1}, err: order.ErrAmountIsInvalid},
	} {
		_, err := e.ExecuteCreditLineTransfer(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "ExecuteCreditLineTransfer must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/cl/transfer", url.Values{"transferType": {"TRANSFER_OUT"}, "transferAssetType": {"BTC"}, "quantity": {"0.00012345"}}, `{}`)
		_, err := ex.ExecuteCreditLineTransfer(t.Context(), &CreditLineTransferRequest{TransferType: CreditLineTransferTypeTransferOut, TransferAssetType: currency.NewCode("btc"), Quantity: 0.00012345})
		require.NoError(t, err, "ExecuteCreditLineTransfer must not error")
	})

	t.Run("transfers the asset", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.ExecuteCreditLineTransfer(t.Context(), &CreditLineTransferRequest{TransferType: CreditLineTransferTypeTransferIn, TransferAssetType: currency.USD, Quantity: 10})
		require.NoError(t, err, "ExecuteCreditLineTransfer must not error")
		require.NotNil(t, result, "ExecuteCreditLineTransfer must return a result")
		if mockTests {
			assert.Equal(t, &CreditLineTransferResponse{TransferID: 1331391, Status: "SUCCESS"}, result, "ExecuteCreditLineTransfer should decode the transfer")
		}
	})
}
