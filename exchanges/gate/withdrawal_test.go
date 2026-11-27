package gate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawCurrency must reject a nil request")

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalRequest{})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "WithdrawCurrency must reject a zero amount")

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalRequest{Amount: 0.000000001})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "WithdrawCurrency must reject an empty currency")

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalRequest{Amount: 1, Currency: currency.BTC})
	require.ErrorIs(t, err, errInvalidCurrencyChain, "WithdrawCurrency must reject an empty chain")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WithdrawCurrency(t.Context(), &WithdrawalRequest{
		Currency: currency.BTC,
		Amount:   0.00000001,
		Chain:    "ERC20",
		Address:  core.BitcoinDonationAddress,
	})
	require.NoError(t, err, "WithdrawCurrency must not error")
	if mockTests {
		exp := &WithdrawalResponse{
			ID:                "210496",
			WithdrawalID:      "w1879219868",
			Timestamp:         types.Time(time.Unix(1542000000, 0)),
			Currency:          currency.BTC,
			WithdrawalAddress: core.BitcoinDonationAddress,
			Amount:            0.00000001,
			Status:            "REQUEST",
			Chain:             "ERC20",
			AssetClass:        "SPOT",
		}
		assert.Equal(t, exp, result, "WithdrawCurrency should decode every field")
	}
}

func TestCancelWithdrawalWithSpecifiedID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawalWithSpecifiedID(t.Context(), "")
	require.ErrorIs(t, err, errMissingWithdrawalID, "CancelWithdrawalWithSpecifiedID must reject an empty withdrawal ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelWithdrawalWithSpecifiedID(t.Context(), "1234567")
	require.NoError(t, err, "CancelWithdrawalWithSpecifiedID must not error")
	if mockTests {
		exp := &WithdrawalResponse{
			ID:                "1234567",
			Timestamp:         types.Time(time.Unix(1542000000, 0)),
			Currency:          currency.USDT,
			WithdrawalAddress: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
			TransactionID:     "128988928203223323290",
			BlockNumber:       18217349,
			Amount:            222.61,
			Status:            "CANCELPEND",
			Chain:             "TRX",
		}
		assert.Equal(t, exp, result, "CancelWithdrawalWithSpecifiedID should decode every field")
	}
}

func TestUIDTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.UIDTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UIDTransfer must reject a nil request")
	_, err = e.UIDTransfer(t.Context(), &UIDTransferRequest{})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "UIDTransfer must reject a zero amount")
	_, err = e.UIDTransfer(t.Context(), &UIDTransferRequest{Amount: 1.})
	require.ErrorIs(t, err, errUserIDRequired, "UIDTransfer must reject an empty recipient")

	_, err = e.UIDTransfer(t.Context(), &UIDTransferRequest{Amount: 1., RecipientUserID: 2231234})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UIDTransfer must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UIDTransfer(t.Context(), &UIDTransferRequest{
		Amount:          1.,
		RecipientUserID: 2231234,
		Currency:        currency.ETH,
	})
	require.NoError(t, err, "UIDTransfer must not error")
	if mockTests {
		assert.Equal(t, &UIDTransferResponse{ID: "111"}, result, "UIDTransfer should decode the order ID")
	}
}
