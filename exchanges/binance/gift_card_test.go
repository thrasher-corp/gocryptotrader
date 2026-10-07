package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestCreateDualTokenGiftCard(t *testing.T) {
	t.Parallel()
	_, err := e.CreateDualTokenGiftCard(t.Context(), currency.EMPTYCODE, currency.BNB, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateDualTokenGiftCard must reject an empty base token")
	_, err = e.CreateDualTokenGiftCard(t.Context(), currency.USDT, currency.EMPTYCODE, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateDualTokenGiftCard must reject an empty face token")
	_, err = e.CreateDualTokenGiftCard(t.Context(), currency.USDT, currency.BNB, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateDualTokenGiftCard must reject an empty amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateDualTokenGiftCard(t.Context(), currency.USDT, currency.BNB, 10)
	require.NoError(t, err, "CreateDualTokenGiftCard must not error")
	if mockTests {
		exp := &GiftCardResponse{
			Code:    "000000",
			Message: "success",
			Data:    GiftCard{ReferenceNumber: "0033002144060553", Code: "6H9EKF5ECCWFBHGE", ExpiredTime: types.Time(time.UnixMilli(1727417154000))},
			Success: true,
		}
		assert.Equal(t, exp, result, "CreateDualTokenGiftCard should decode every field")
		return
	}
	assert.True(t, result.Success, "CreateDualTokenGiftCard should succeed")
}

func TestCreateSingleTokenGiftCard(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSingleTokenGiftCard(t.Context(), currency.EMPTYCODE, 0.1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateSingleTokenGiftCard must reject an empty token")
	_, err = e.CreateSingleTokenGiftCard(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateSingleTokenGiftCard must reject an empty amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSingleTokenGiftCard(t.Context(), currency.USDT, 0.1234)
	require.NoError(t, err, "CreateSingleTokenGiftCard must not error")
	if mockTests {
		exp := &GiftCardResponse{
			Code:    "000000",
			Message: "success",
			Data:    GiftCard{ReferenceNumber: "0033002144060554", Code: "6H9EKF5ECCWFBHGF", ExpiredTime: types.Time(time.UnixMilli(1727417155000))},
			Success: true,
		}
		assert.Equal(t, exp, result, "CreateSingleTokenGiftCard should decode every field")
		return
	}
	assert.True(t, result.Success, "CreateSingleTokenGiftCard should succeed")
}

func TestFetchRSAPublicKey(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FetchRSAPublicKey(t.Context())
	require.NoError(t, err, "FetchRSAPublicKey must not error")
	if mockTests {
		exp := &RSAPublicKeyResponse{
			Code:    "000000",
			Message: "success",
			Data:    "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCXBBVKLAc1GQ5FsIFFqOHrPTox5noBONIKr+IAedTR9FkVxq6e65updEbfdhRNkMOeYIO2i0UylrjGC0X8YSoIszmrVHeV0l06Zh1oJuZos1+7N+WLuz9JvlPaawof3GUakTxYWWCa9+8KIbLKsoKMdfS96VT+8iOXO3quMGKUQQIDAQAB",
			Success: true,
		}
		assert.Equal(t, exp, result, "FetchRSAPublicKey should decode every field")
		return
	}
	assert.NotEmpty(t, result.Data, "FetchRSAPublicKey should return a key")
}

func TestFetchTokenLimit(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTokenLimit(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FetchTokenLimit must reject an empty base token")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FetchTokenLimit(t.Context(), currency.USDT)
	require.NoError(t, err, "FetchTokenLimit must not error")
	if mockTests {
		exp := &TokenLimitResponse{
			Code:    "000000",
			Message: "success",
			Data:    []GiftCardTokenLimit{{Coin: currency.BNB, FromMin: 0.01, FromMax: 1}},
			Success: true,
		}
		assert.Equal(t, exp, result, "FetchTokenLimit should decode every field")
		return
	}
	assert.True(t, result.Success, "FetchTokenLimit should succeed")
}

func TestRedeemBinanceGiftCard(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemBinanceGiftCard(t.Context(), "", "12345")
	require.ErrorIs(t, err, errCodeRequired, "RedeemBinanceGiftCard must reject an empty code")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RedeemBinanceGiftCard(t.Context(), "0033002328060227", "12345")
	require.NoError(t, err, "RedeemBinanceGiftCard must not error")
	if mockTests {
		exp := &RedeemGiftCardResponse{
			Code:    "000000",
			Message: "success",
			Data:    RedeemedGiftCard{ReferenceNumber: "0033002328060227", IdentityNumber: "10317392647411060736", Token: currency.BNB, Amount: 0.00000001},
			Success: true,
		}
		assert.Equal(t, exp, result, "RedeemBinanceGiftCard should decode every field")
		return
	}
	assert.True(t, result.Success, "RedeemBinanceGiftCard should succeed")
}

func TestVerifyBinanceGiftCardNumber(t *testing.T) {
	t.Parallel()
	_, err := e.VerifyBinanceGiftCardNumber(t.Context(), "")
	require.ErrorIs(t, err, errReferenceNumberRequired, "VerifyBinanceGiftCardNumber must reject an empty card number")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.VerifyBinanceGiftCardNumber(t.Context(), "123456")
	require.NoError(t, err, "VerifyBinanceGiftCardNumber must not error")
	if mockTests {
		exp := &GiftCardVerificationResponse{
			Code:    "000000",
			Message: "success",
			Data:    GiftCardVerification{Valid: true, Token: currency.BNB, Amount: 0.00000001},
			Success: true,
		}
		assert.Equal(t, exp, result, "VerifyBinanceGiftCardNumber should decode every field")
		return
	}
	assert.NotNil(t, result, "VerifyBinanceGiftCardNumber should return a result")
}
