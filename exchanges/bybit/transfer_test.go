package bybit

import (
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestCreateInternalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.CreateInternalTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateInternalTransfer must reject a nil request")
	arg := &InternalTransferRequest{}
	_, err = e.CreateInternalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errMissingTransferID, "CreateInternalTransfer must reject an empty transfer ID")
	arg.TransferID = uuid.MustParse("42c0cfb0-6bca-c242-bc76-4e6df6cbcb16")
	_, err = e.CreateInternalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateInternalTransfer must reject an empty coin")
	arg.Coin = currency.BTC
	_, err = e.CreateInternalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateInternalTransfer must reject a zero amount")
	arg.Amount = 123.456
	_, err = e.CreateInternalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errMissingAccountType, "CreateInternalTransfer must reject an empty from account type")
	arg.FromAccountType = "UNIFIED"
	_, err = e.CreateInternalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errMissingAccountType, "CreateInternalTransfer must reject an empty to account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		arg.TransferID = uuid.NewV7()
	}
	arg.ToAccountType = "FUND"
	result, err := e.CreateInternalTransfer(t.Context(), arg)
	require.NoError(t, err, "CreateInternalTransfer must not error")
	if mockTests {
		exp := &TransferResponse{
			TransferID: "42c0cfb0-6bca-c242-bc76-4e6df6cbcb16",
			Status:     "SUCCESS",
		}
		assert.Equal(t, exp, result, "CreateInternalTransfer should decode every field")
		return
	}
	assert.Equal(t, arg.TransferID.String(), result.TransferID, "CreateInternalTransfer should return the transfer ID")
}

func TestGetInternalTransferRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetInternalTransferRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInternalTransferRecords must reject a nil request")
	_, err = e.GetInternalTransferRecords(t.Context(), &InternalTransferRecordsRequest{StartTime: time.UnixMilli(1667283263000), EndTime: time.UnixMilli(1667200000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetInternalTransferRecords must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInternalTransferRecords(t.Context(), &InternalTransferRecordsRequest{TransferID: "018bd458-dba0-728b-b5b6-ecd5bd296528", Coin: currency.BTC})
	require.NoError(t, err, "GetInternalTransferRecords must not error")
	t.Run("every filter", func(t *testing.T) {
		t.Parallel()
		// A window spans at most 7 days
		next, err := e.GetInternalTransferRecords(t.Context(), &InternalTransferRecordsRequest{
			TransferID: "019a7e3c-5d2b-7c1a-9e4f-3b2a1c0d9e8f",
			Coin:       currency.NewCode("usdt"),
			Status:     "SUCCESS",
			StartTime:  time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			EndTime:    time.Date(2026, time.September, 7, 0, 0, 0, 0, time.UTC),
			Limit:      20,
			Cursor:     result.NextPageCursor,
		})
		require.NoError(t, err, "GetInternalTransferRecords must not error")
		if mockTests {
			exp := &InternalTransferRecordsResponse{
				List: []InternalTransferRecord{
					{
						TransferID:      "019a7e3c-5d2b-7c1a-9e4f-3b2a1c0d9e8f",
						Coin:            currency.USDT,
						Amount:          250,
						FromAccountType: "FUND",
						ToAccountType:   "UNIFIED",
						Timestamp:       types.Time(time.UnixMilli(1788310000000)),
						Status:          "SUCCESS",
					},
				},
			}
			assert.Equal(t, exp, next, "GetInternalTransferRecords should send every filter and decode the matching page")
			return
		}
		assert.NotNil(t, next, "GetInternalTransferRecords should return a page")
	})
	if mockTests {
		exp := &InternalTransferRecordsResponse{
			List: []InternalTransferRecord{
				{
					TransferID:      "018bd458-dba0-728b-b5b6-ecd5bd296528",
					Coin:            currency.BTC,
					Amount:          0.05,
					FromAccountType: "UNIFIED",
					ToAccountType:   "FUND",
					Timestamp:       types.Time(time.Unix(1667283263, 0)),
					Status:          "SUCCESS",
				},
			},
			NextPageCursor: "eyJtaW5JRCI6MTM1ODQ2OCwibWF4SUQiOjEzNTg0Njh9",
		}
		assert.Equal(t, exp, result, "GetInternalTransferRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInternalTransferRecords should return a page")
}

func TestCreateUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.CreateUniversalTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateUniversalTransfer must reject a nil request")
	arg := &UniversalTransferRequest{}
	_, err = e.CreateUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errMissingTransferID, "CreateUniversalTransfer must reject an empty transfer ID")
	arg.TransferID = uuid.MustParse("be7a2462-1138-4e27-80b1-62653f24925e")
	_, err = e.CreateUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateUniversalTransfer must reject an empty coin")
	arg.Coin = currency.BTC
	_, err = e.CreateUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateUniversalTransfer must reject a zero amount")
	arg.Amount = 123.456
	_, err = e.CreateUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errMemberIDRequired, "CreateUniversalTransfer must reject an empty from member ID")
	arg.FromMemberID = 123
	_, err = e.CreateUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errMemberIDRequired, "CreateUniversalTransfer must reject an empty to member ID")
	arg.ToMemberID = 456
	_, err = e.CreateUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errMissingAccountType, "CreateUniversalTransfer must reject an empty from account type")
	arg.FromAccountType = "FUND"
	_, err = e.CreateUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errMissingAccountType, "CreateUniversalTransfer must reject an empty to account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		arg.TransferID = uuid.NewV7()
	}
	arg.ToAccountType = "UNIFIED"
	result, err := e.CreateUniversalTransfer(t.Context(), arg)
	require.NoError(t, err, "CreateUniversalTransfer must not error")
	if mockTests {
		exp := &TransferResponse{
			TransferID: "be7a2462-1138-4e27-80b1-62653f24925e",
			Status:     "PENDING",
		}
		assert.Equal(t, exp, result, "CreateUniversalTransfer should decode every field")
		return
	}
	assert.Equal(t, arg.TransferID.String(), result.TransferID, "CreateUniversalTransfer should return the transfer ID")
}

func TestGetUniversalTransferRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetUniversalTransferRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUniversalTransferRecords must reject a nil request")
	_, err = e.GetUniversalTransferRecords(t.Context(), &UniversalTransferRecordsRequest{StartTime: time.UnixMilli(1672134373000), EndTime: time.UnixMilli(1672100000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUniversalTransferRecords must reject a reversed window")

	arg := &UniversalTransferRecordsRequest{TransferID: "018bd461-cb9c-75ce-94d4-0d3f4d84c339", Coin: currency.BTC}
	if mockTests {
		// The member filters name placeholder UIDs, which only the fixture knows
		arg.FromMemberID, arg.ToMemberID = "100000001", "100000002"
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUniversalTransferRecords(t.Context(), arg)
	require.NoError(t, err, "GetUniversalTransferRecords must not error")
	t.Run("every filter", func(t *testing.T) {
		t.Parallel()
		// A window spans at most 7 days
		filtered := &UniversalTransferRecordsRequest{
			TransferID: "019a7e40-1c2d-7e3f-8a9b-0c1d2e3f4a5b",
			Coin:       currency.NewCode("usdt"),
			Status:     "SUCCESS",
			StartTime:  time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			EndTime:    time.Date(2026, time.September, 7, 0, 0, 0, 0, time.UTC),
			Limit:      20,
			Cursor:     result.NextPageCursor,
		}
		if mockTests {
			// As above, the member filters name placeholder UIDs
			filtered.FromMemberID, filtered.ToMemberID = "100000001", "100000002"
		}
		next, err := e.GetUniversalTransferRecords(t.Context(), filtered)
		require.NoError(t, err, "GetUniversalTransferRecords must not error")
		if mockTests {
			exp := &UniversalTransferRecordsResponse{
				List: []UniversalTransferRecord{
					{
						TransferID:        "019a7e40-1c2d-7e3f-8a9b-0c1d2e3f4a5b",
						Coin:              currency.USDT,
						Amount:            1000,
						FromMemberID:      "100000001",
						ToMemberID:        "100000002",
						WithdrawRequestID: "019a7e40-1c2d-7e3f-8a9b-0c1d2e3f4a5b",
						DepositRequestID:  "5e6f7a8b-9c0d-4e1f-a2b3-c4d5e6f7a8b9",
						FromAccountType:   "UNIFIED",
						ToAccountType:     "FUND",
						Timestamp:         types.Time(time.UnixMilli(1788400000000)),
						Status:            "SUCCESS",
					},
				},
			}
			assert.Equal(t, exp, next, "GetUniversalTransferRecords should send every filter and decode the matching page")
			return
		}
		assert.NotNil(t, next, "GetUniversalTransferRecords should return a page")
	})
	if mockTests {
		exp := &UniversalTransferRecordsResponse{
			List: []UniversalTransferRecord{
				{
					TransferID:        "018bd461-cb9c-75ce-94d4-0d3f4d84c339",
					Coin:              currency.BTC,
					Amount:            0.5,
					FromMemberID:      "100000001",
					ToMemberID:        "100000002",
					WithdrawRequestID: "018bd461-cb9c-75ce-94d4-0d3f4d84c339",
					DepositRequestID:  "c1fb8ebe-e309-472c-8485-c1b78f2048bb",
					FromAccountType:   "FUND",
					ToAccountType:     "UNIFIED",
					Timestamp:         types.Time(time.Unix(1672134373, 0)),
					Status:            "SUCCESS",
				},
			},
			NextPageCursor: "eyJtaW5JRCI6MTc4OTYwNSwibWF4SUQiOjE3ODk2MDV9",
		}
		assert.Equal(t, exp, result, "GetUniversalTransferRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUniversalTransferRecords should return a page")
}

func TestGetTransferableCoin(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransferableCoin(t.Context(), "", "UNIFIED")
	require.ErrorIs(t, err, errMissingAccountType, "GetTransferableCoin must reject an empty from account type")
	_, err = e.GetTransferableCoin(t.Context(), "FUND", "")
	require.ErrorIs(t, err, errMissingAccountType, "GetTransferableCoin must reject an empty to account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTransferableCoin(t.Context(), "FUND", "UNIFIED")
	require.NoError(t, err, "GetTransferableCoin must not error")
	if mockTests {
		exp := &TransferableCoinResponse{
			List: []currency.Code{
				currency.BTC,
				currency.ETH,
				currency.USDT,
			},
		}
		assert.Equal(t, exp, result, "GetTransferableCoin should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetTransferableCoin should return coins")
}
