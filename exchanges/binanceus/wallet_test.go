package binanceus

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetAssetFeesAndWalletStatus(t *testing.T) {
	t.Parallel()
	skipIfCredentialsUnset(t)
	result, err := e.GetAssetFeesAndWalletStatus(t.Context())
	require.NoError(t, err, "GetAssetFeesAndWalletStatus must not error")
	if !mockTests {
		assert.NotEmpty(t, result, "GetAssetFeesAndWalletStatus should return the assets")
		return
	}
	memoRegex := `^[0-9A-Za-z\-_]{1,120}$`
	exp := []CoinInfoResponse{
		{
			Coin:              currency.BCH,
			DepositAllEnable:  true,
			WithdrawAllEnable: true,
			Name:              "BCH",
			Trading:           true,
			NetworkList: []CoinNetwork{
				{Network: "BCH", Coin: currency.BCH, WithdrawIntegerMultiple: 0.00000001, IsDefault: true, DepositEnable: true, WithdrawEnable: true, Name: "Bitcoin Cash", WithdrawMaximum: 9999999},
				{Network: "BNB", Coin: currency.BCH, WithdrawIntegerMultiple: 0.00000001, DepositEnable: true, WithdrawEnable: true, Name: "BEP222ee", AddressRegex: "^(bnb1)[0-9a-z]{38}$", MemoRegex: memoRegex, WithdrawMaximum: 9999999, MinimumConfirm: 15, UnlockConfirm: 3},
				{Network: "BSC", Coin: currency.BCH, WithdrawIntegerMultiple: 0.00000001, DepositEnable: true, WithdrawEnable: true, Name: "BSC (BEP20)", AddressRegex: "^(0x)[0-9A-Fa-f]{40}$", WithdrawMaximum: 9999999},
			},
		},
		{
			Coin:              currency.PAX,
			DepositAllEnable:  true,
			WithdrawAllEnable: true,
			Name:              "PAX",
			Free:              12.5,
			Locked:            1.5,
			Freeze:            0.5,
			Withdrawing:       2,
			IPOing:            0.25,
			IPOable:           0.75,
			Storage:           3,
			IsLegalMoney:      true,
			NetworkList: []CoinNetwork{{
				Network:             "BNB",
				Coin:                currency.PAX,
				DepositEnable:       true,
				WithdrawEnable:      true,
				DepositDescription:  "Deposits need a memo",
				WithdrawDescription: "Withdrawals need a memo",
				SpecialTips:         "Enter both the memo and the address",
				Name:                "BEP222ee",
				ResetAddressStatus:  true,
				AddressRegex:        "^(bnb1)[0-9a-z]{38}$",
				MemoRegex:           memoRegex,
				WithdrawFee:         0.5,
				WithdrawMinimum:     10,
				WithdrawMaximum:     9999999,
				MinimumConfirm:      15,
				UnlockConfirm:       3,
			}},
		},
	}
	assert.Equal(t, exp, result, "GetAssetFeesAndWalletStatus should decode every field")
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiat(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawFiat must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *WithdrawFiatRequest
		err  error
	}{
		{name: "no payment method", req: &WithdrawFiatRequest{PaymentAccount: "4a992541-c12d-4cca-bbd6-df637f801526", Amount: 100}, err: errPaymentMethodRequired},
		{name: "no payment account", req: &WithdrawFiatRequest{PaymentMethod: PaymentMethodBitGo, Amount: 100}, err: errPaymentAccountRequired},
		{name: "no amount", req: &WithdrawFiatRequest{PaymentMethod: PaymentMethodBitGo, PaymentAccount: "4a992541-c12d-4cca-bbd6-df637f801526"}, err: order.ErrAmountIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.WithdrawFiat(t.Context(), tc.req)
			require.ErrorIs(t, err, tc.err, "WithdrawFiat must reject an incomplete request")
		})
	}

	t.Run("posts the documented parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *WithdrawFiatRequest
			exp  url.Values
		}{
			{
				name: "every parameter",
				req:  &WithdrawFiatRequest{PaymentMethod: PaymentMethodWire, PaymentAccount: "9448807b1b024302befcea94b3f271f6", FiatCurrency: currency.NewCode("usd"), Amount: 100.25},
				exp:  url.Values{"paymentMethod": {"WIRE"}, "paymentAccount": {"9448807b1b024302befcea94b3f271f6"}, "fiatCurrency": {"USD"}, "amount": {"100.25"}},
			},
			{
				name: "required parameters only",
				req:  &WithdrawFiatRequest{PaymentMethod: PaymentMethodIntrabankTransfer, PaymentAccount: "external-account-id", Amount: 0.5},
				exp:  url.Values{"paymentMethod": {"INTRABANK_TRANSFER"}, "paymentAccount": {"external-account-id"}, "amount": {"0.5"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/fiatpayment/withdraw/apply", tc.exp, `{}`)
				_, err := ex.WithdrawFiat(t.Context(), tc.req)
				require.NoError(t, err, "WithdrawFiat must not error")
			})
		}
	})

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live withdrawal")
	}
	skipIfCredentialsUnset(t)
	result, err := e.WithdrawFiat(t.Context(), &WithdrawFiatRequest{PaymentMethod: PaymentMethodBitGo, PaymentAccount: "4a992541-c12d-4cca-bbd6-df637f801526", FiatCurrency: currency.USD, Amount: 100})
	require.NoError(t, err, "WithdrawFiat must not error")
	require.NotNil(t, result, "WithdrawFiat must return the withdrawal")
	if mockTests {
		exp := &WithdrawFiatResponse{OrderID: "6c2ff984890145fdac2b7160299062f0", ChannelCode: "BITGO", CurrencyCode: currency.USD, Amount: 100, OrderStatus: "INIT"}
		assert.Equal(t, exp, result, "WithdrawFiat should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "WithdrawFiat should return an order ID")
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCrypto(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawCrypto must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *WithdrawRequest
		err  error
	}{
		{name: "no coin", req: &WithdrawRequest{Network: "BTC", Address: core.BitcoinDonationAddress, Amount: 0.001}, err: currency.ErrCurrencyCodeEmpty},
		{name: "no network", req: &WithdrawRequest{Coin: currency.BTC, Address: core.BitcoinDonationAddress, Amount: 0.001}, err: errNetworkRequired},
		{name: "no address", req: &WithdrawRequest{Coin: currency.BTC, Network: "BTC", Amount: 0.001}, err: errAddressRequired},
		{name: "no amount", req: &WithdrawRequest{Coin: currency.BTC, Network: "BTC", Address: core.BitcoinDonationAddress}, err: order.ErrAmountIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.WithdrawCrypto(t.Context(), tc.req)
			require.ErrorIs(t, err, tc.err, "WithdrawCrypto must reject an incomplete request")
		})
	}

	t.Run("posts every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"coin": {"XRP"}, "network": {"XRP"}, "withdrawOrderId": {"gct-withdrawal-1"}, "address": {"test-address"}, "addressTag": {"100345"}, "amount": {"25.5"}}
		ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/capital/withdraw/apply", exp, `{}`)
		_, err := ex.WithdrawCrypto(t.Context(), &WithdrawRequest{Coin: currency.NewCode("xrp"), Network: "XRP", WithdrawOrderID: "gct-withdrawal-1", Address: "test-address", AddressTag: "100345", Amount: 25.5})
		require.NoError(t, err, "WithdrawCrypto must not error")
	})

	t.Run("sends the amount in full", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			amount float64
			exp    string
		}{
			{name: "fraction", amount: 1.9, exp: "1.9"},
			{name: "small fraction", amount: 0.00001234, exp: "0.00001234"},
			{name: "whole amount", amount: 2, exp: "2"},
			{name: "seventeen significant digits", amount: 0.12345678912345678, exp: "0.12345678912345678"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				exp := url.Values{"coin": {"BTC"}, "network": {"BTC"}, "address": {"test-address"}, "amount": {tc.exp}}
				ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/capital/withdraw/apply", exp, `{"id":"withdrawal-id"}`)
				result, err := ex.WithdrawCrypto(t.Context(), &WithdrawRequest{Coin: currency.BTC, Network: "BTC", Address: "test-address", Amount: tc.amount})
				require.NoError(t, err, "WithdrawCrypto must not error")
				assert.Equal(t, &WithdrawResponse{ID: "withdrawal-id"}, result, "WithdrawCrypto should decode the withdrawal ID")
			})
		}
	})

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live withdrawal")
	}
	skipIfCredentialsUnset(t)
	result, err := e.WithdrawCrypto(t.Context(), &WithdrawRequest{Coin: currency.BTC, Network: "BTC", Address: core.BitcoinDonationAddress, Amount: 0.001})
	require.NoError(t, err, "WithdrawCrypto must not error")
	require.NotNil(t, result, "WithdrawCrypto must return the withdrawal")
	if mockTests {
		assert.Equal(t, &WithdrawResponse{ID: "4e7f4f31560041ee880b5386f06d4053"}, result, "WithdrawCrypto should decode the withdrawal ID")
		return
	}
	assert.NotEmpty(t, result.ID, "WithdrawCrypto should return a withdrawal ID")
}

func TestWithdrawalHistory(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawalHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawalHistory must reject a nil request")
	_, err = e.WithdrawalHistory(t.Context(), &WithdrawalHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "WithdrawalHistory must reject a request without a coin")
	start := time.UnixMilli(1645155364000)
	_, err = e.WithdrawalHistory(t.Context(), &WithdrawalHistoryRequest{Coin: currency.BNB, StartTime: start, EndTime: start.Add(-time.Second)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "WithdrawalHistory must reject a start after the end")

	t.Run("sends the set filters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *WithdrawalHistoryRequest
			exp  url.Values
		}{
			{name: "coin only", req: &WithdrawalHistoryRequest{Coin: currency.NewCode("bnb")}, exp: url.Values{"coin": {"BNB"}}},
			{
				name: "every filter",
				req:  &WithdrawalHistoryRequest{Coin: currency.BNB, WithdrawOrderID: "gct-withdrawal-1", Status: new(uint64(0)), StartTime: start, EndTime: start.Add(time.Hour), Offset: 10, Limit: 1000},
				exp:  url.Values{"coin": {"BNB"}, "withdrawOrderId": {"gct-withdrawal-1"}, "status": {"0"}, "startTime": {"1645155364000"}, "endTime": {"1645158964000"}, "offset": {"10"}, "limit": {"1000"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/capital/withdraw/history", tc.exp, `[]`)
				_, err := ex.WithdrawalHistory(t.Context(), tc.req)
				require.NoError(t, err, "WithdrawalHistory must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.WithdrawalHistory(t.Context(), &WithdrawalHistoryRequest{Coin: currency.BNB})
	require.NoError(t, err, "WithdrawalHistory must not error")
	if mockTests {
		exp := []WithdrawalRecordResponse{{
			ID:             "4e7f4f31560041ee880b5386f06d4053",
			Amount:         0.9,
			TransactionFee: 0.1,
			Coin:           currency.BNB,
			Status:         2,
			Address:        "0xd709f9d0bbc6b0e746a13142dfe353086edf87c2",
			ApplyTime:      types.DateTime(time.Date(2022, time.February, 18, 3, 36, 4, 0, time.UTC)),
			Network:        "BSC",
			TransferType:   1,
		}}
		assert.Equal(t, exp, result, "WithdrawalHistory should decode every field")
	}
}

func TestFiatWithdrawalHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FiatWithdrawalHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FiatWithdrawalHistory must reject a nil request")

	t.Run("sends a past end time", func(t *testing.T) {
		t.Parallel()
		past := time.Now().Add(-time.Hour)
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/fiatpayment/query/withdraw/history", url.Values{"endTime": {strconv.FormatInt(past.UnixMilli(), 10)}}, `{}`)
		_, err := ex.FiatWithdrawalHistory(t.Context(), &FiatHistoryRequest{EndTime: past})
		require.NoError(t, err, "FiatWithdrawalHistory must not error")
	})

	skipIfCredentialsUnset(t)
	result, err := e.FiatWithdrawalHistory(t.Context(), &FiatHistoryRequest{})
	require.NoError(t, err, "FiatWithdrawalHistory must not error")
	require.NotNil(t, result, "FiatWithdrawalHistory must return the withdrawals")
	if mockTests {
		exp := &FiatHistoryResponse{AssetLogRecordList: []FiatAssetLogRecord{{
			OrderID:        "6c2ff984890145fdac2b7160299062f0",
			PaymentAccount: "4a992541-c12d-4cca-bbd6-df637f801526",
			PaymentChannel: "PRIMETRUST",
			PaymentMethod:  "WIRE_INTERNATIONAL",
			OrderStatus:    "Processing",
			Amount:         65,
			TransactionFee: 20,
			PlatformFee:    0.5,
		}}}
		assert.Equal(t, exp, result, "FiatWithdrawalHistory should decode every field")
	}
}

func TestFiatHistory(t *testing.T) {
	t.Parallel()
	const path = "/sapi/v1/fiatpayment/query/withdraw/history"
	_, err := e.fiatHistory(t.Context(), path, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "fiatHistory must reject a nil request")
	now := time.Now()
	start, past, future := now.Add(-48*time.Hour), now.Add(-time.Hour), now.Add(time.Hour)
	_, err = e.fiatHistory(t.Context(), path, &FiatHistoryRequest{StartTime: past, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "fiatHistory must reject a start after the end")

	startMilli, pastMilli, futureMilli := strconv.FormatInt(start.UnixMilli(), 10), strconv.FormatInt(past.UnixMilli(), 10), strconv.FormatInt(future.UnixMilli(), 10)
	for _, tc := range []struct {
		name string
		req  *FiatHistoryRequest
		exp  url.Values
	}{
		{name: "no filters", req: &FiatHistoryRequest{}, exp: url.Values{}},
		{name: "numeric order ID", req: &FiatHistoryRequest{FiatCurrency: currency.USD, OrderID: "1337"}, exp: url.Values{"fiatCurrency": {"USD"}, "orderId": {"1337"}}},
		{name: "hexadecimal order ID", req: &FiatHistoryRequest{FiatCurrency: currency.USD, OrderID: "6c2ff984890145fdac2b7160299062f0"}, exp: url.Values{"fiatCurrency": {"USD"}, "orderId": {"6c2ff984890145fdac2b7160299062f0"}}},
		{name: "no order ID", req: &FiatHistoryRequest{FiatCurrency: currency.USD}, exp: url.Values{"fiatCurrency": {"USD"}}},
		{name: "past end time", req: &FiatHistoryRequest{EndTime: past}, exp: url.Values{"endTime": {pastMilli}}},
		// Binance.US clamps a future time to the present, so it is sent as set
		{name: "future end time", req: &FiatHistoryRequest{EndTime: future}, exp: url.Values{"endTime": {futureMilli}}},
		{name: "past start time", req: &FiatHistoryRequest{StartTime: start}, exp: url.Values{"startTime": {startMilli}}},
		{name: "past start and end times", req: &FiatHistoryRequest{StartTime: start, EndTime: past}, exp: url.Values{"startTime": {startMilli}, "endTime": {pastMilli}}},
		{name: "past start time and future end time", req: &FiatHistoryRequest{StartTime: start, EndTime: future}, exp: url.Values{"startTime": {startMilli}, "endTime": {futureMilli}}},
		{
			name: "every filter",
			req:  &FiatHistoryRequest{FiatCurrency: currency.NewCode("usd"), OrderID: "6c2ff984890145fdac2b7160299062f0", Offset: 10, PaymentChannel: "WAB", PaymentMethod: PaymentMethodIntrabankTransfer, StartTime: start, EndTime: past},
			exp:  url.Values{"fiatCurrency": {"USD"}, "orderId": {"6c2ff984890145fdac2b7160299062f0"}, "offset": {"10"}, "paymentChannel": {"WAB"}, "paymentMethod": {"INTRABANK_TRANSFER"}, "startTime": {startMilli}, "endTime": {pastMilli}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := signedRequestExchange(t, http.MethodGet, path, tc.exp, `{"assetLogRecordList":[]}`)
			result, err := ex.fiatHistory(t.Context(), path, tc.req)
			require.NoError(t, err, "fiatHistory must not error")
			assert.Equal(t, &FiatHistoryResponse{AssetLogRecordList: []FiatAssetLogRecord{}}, result, "fiatHistory should decode an empty history")
		})
	}
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddressForCurrency(t.Context(), currency.EMPTYCODE, "BSC")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddressForCurrency must reject an empty coin")

	t.Run("sends the coin and the set network", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name    string
			network string
			exp     url.Values
		}{
			{name: "default network", exp: url.Values{"coin": {"BNB"}}},
			{name: "network", network: "BSC", exp: url.Values{"coin": {"BNB"}, "network": {"BSC"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/capital/deposit/address", tc.exp, `{}`)
				_, err := ex.GetDepositAddressForCurrency(t.Context(), currency.NewCode("bnb"), tc.network)
				require.NoError(t, err, "GetDepositAddressForCurrency must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetDepositAddressForCurrency(t.Context(), currency.BNB, "")
	require.NoError(t, err, "GetDepositAddressForCurrency must not error")
	require.NotNil(t, result, "GetDepositAddressForCurrency must return an address")
	if mockTests {
		exp := &DepositAddressResponse{Coin: currency.BNB, Address: "0xcf1988d27e402086941284313b632466fdf28a22", Tag: "100345", URL: "0xcf1988d27e402086941284313b632466fdf28a22"}
		assert.Equal(t, exp, result, "GetDepositAddressForCurrency should decode every field")
		return
	}
	assert.NotEmpty(t, result.Address, "GetDepositAddressForCurrency should return an address")
}

func TestDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.DepositHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "DepositHistory must reject a nil request")
	_, err = e.DepositHistory(t.Context(), &DepositHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "DepositHistory must reject a request without a coin")
	start := time.UnixMilli(1638342348000)
	_, err = e.DepositHistory(t.Context(), &DepositHistoryRequest{Coin: currency.BNB, StartTime: start, EndTime: start.Add(-time.Second)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "DepositHistory must reject a start after the end")

	t.Run("sends the set filters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *DepositHistoryRequest
			exp  url.Values
		}{
			{name: "coin only", req: &DepositHistoryRequest{Coin: currency.NewCode("bnb")}, exp: url.Values{"coin": {"BNB"}}},
			{
				name: "every filter",
				req:  &DepositHistoryRequest{Coin: currency.BNB, Status: new(uint64(0)), StartTime: start, EndTime: start.Add(time.Hour), Offset: 10, Limit: 1000},
				exp:  url.Values{"coin": {"BNB"}, "status": {"0"}, "startTime": {"1638342348000"}, "endTime": {"1638345948000"}, "offset": {"10"}, "limit": {"1000"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/capital/deposit/hisrec", tc.exp, `[]`)
				_, err := ex.DepositHistory(t.Context(), tc.req)
				require.NoError(t, err, "DepositHistory must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.DepositHistory(t.Context(), &DepositHistoryRequest{Coin: currency.BNB})
	require.NoError(t, err, "DepositHistory must not error")
	if mockTests {
		exp := []DepositRecordResponse{{
			Amount:       8.73234,
			Coin:         currency.BNB,
			Network:      "BSC",
			Status:       1,
			Address:      "0xd709f9d0bbc6b0e746a13142dfe353086edf87c2",
			AddressTag:   "100345",
			TxID:         "0xa9ebf3f4f60bc18bd6bdf4616ff8ffa14ef93a08fe79cad40519b31ea1044290",
			InsertTime:   types.Time(time.UnixMilli(1638342348000)),
			TransferType: 1,
			ConfirmTimes: "0/0",
		}}
		assert.Equal(t, exp, result, "DepositHistory should decode every field")
	}
}

func TestFiatDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FiatDepositHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FiatDepositHistory must reject a nil request")

	t.Run("sends a past end time", func(t *testing.T) {
		t.Parallel()
		past := time.Now().Add(-time.Hour)
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/fiatpayment/query/deposit/history", url.Values{"endTime": {strconv.FormatInt(past.UnixMilli(), 10)}}, `{}`)
		_, err := ex.FiatDepositHistory(t.Context(), &FiatHistoryRequest{EndTime: past})
		require.NoError(t, err, "FiatDepositHistory must not error")
	})

	skipIfCredentialsUnset(t)
	result, err := e.FiatDepositHistory(t.Context(), &FiatHistoryRequest{})
	require.NoError(t, err, "FiatDepositHistory must not error")
	require.NotNil(t, result, "FiatDepositHistory must return the deposits")
	if mockTests {
		exp := &FiatHistoryResponse{AssetLogRecordList: []FiatAssetLogRecord{{
			OrderID:        "c3415d574fa747df88a75c66c037f890",
			PaymentAccount: "ab13f629-b074-470a-bc80-b830e169691f",
			PaymentChannel: "MODERNTREASURY",
			PaymentMethod:  "ACH",
			OrderStatus:    "Successful",
			FiatCurrency:   currency.USD,
			Amount:         0.99,
			TransactionFee: 0.01,
		}}}
		assert.Equal(t, exp, result, "FiatDepositHistory should decode every field")
	}
}

func TestGetSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountDepositAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountDepositAddress must reject a nil request")
	_, err = e.GetSubAccountDepositAddress(t.Context(), &SubAccountDepositAddressRequest{Coin: currency.BNB})
	require.ErrorIs(t, err, errEmailRequired, "GetSubAccountDepositAddress must reject a request without an email")
	_, err = e.GetSubAccountDepositAddress(t.Context(), &SubAccountDepositAddressRequest{Email: "sub@test.com"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSubAccountDepositAddress must reject a request without a coin")

	t.Run("sends the set network", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name    string
			network string
			exp     url.Values
		}{
			{name: "default network", exp: url.Values{"email": {"sub@test.com"}, "coin": {"BNB"}}},
			{name: "network", network: "BSC", exp: url.Values{"email": {"sub@test.com"}, "coin": {"BNB"}, "network": {"BSC"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/capital/sub-account/deposit/address", tc.exp, `{}`)
				_, err := ex.GetSubAccountDepositAddress(t.Context(), &SubAccountDepositAddressRequest{Email: "sub@test.com", Coin: currency.NewCode("bnb"), Network: tc.network})
				require.NoError(t, err, "GetSubAccountDepositAddress must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	email := "sub@test.com"
	if !mockTests {
		subAccounts, err := e.GetSubAccountInformation(t.Context(), nil)
		require.NoError(t, err, "GetSubAccountInformation must not error")
		require.NotNil(t, subAccounts, "GetSubAccountInformation must return the sub-accounts")
		if len(subAccounts.SubAccounts) == 0 {
			t.Skip("the account has no sub-accounts")
		}
		email = subAccounts.SubAccounts[0].Email
	}
	result, err := e.GetSubAccountDepositAddress(t.Context(), &SubAccountDepositAddressRequest{Email: email, Coin: currency.BNB})
	require.NoError(t, err, "GetSubAccountDepositAddress must not error")
	require.NotNil(t, result, "GetSubAccountDepositAddress must return an address")
	if mockTests {
		exp := &DepositAddressResponse{Coin: currency.BNB, Address: "0xf79b6274f18425441b8f05c0a711d171c0fe119f", URL: "https://etherscan.io/address/0xf79b6274f18425441b8f05c0a711d171c0fe119f"}
		assert.Equal(t, exp, result, "GetSubAccountDepositAddress should decode every field")
		return
	}
	assert.NotEmpty(t, result.Address, "GetSubAccountDepositAddress should return an address")
}

func TestGetSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountDepositHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountDepositHistory must reject a nil request")
	_, err = e.GetSubAccountDepositHistory(t.Context(), &SubAccountDepositHistoryRequest{Coin: currency.BTC})
	require.ErrorIs(t, err, errEmailRequired, "GetSubAccountDepositHistory must reject a request without an email")
	start := time.UnixMilli(1652942429211)
	_, err = e.GetSubAccountDepositHistory(t.Context(), &SubAccountDepositHistoryRequest{Email: "sub@test.com", StartTime: start, EndTime: start.Add(-time.Second)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubAccountDepositHistory must reject a start after the end")

	t.Run("sends the set filters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *SubAccountDepositHistoryRequest
			exp  url.Values
		}{
			{name: "email only", req: &SubAccountDepositHistoryRequest{Email: "sub@test.com"}, exp: url.Values{"email": {"sub@test.com"}}},
			{
				name: "every filter",
				req:  &SubAccountDepositHistoryRequest{Email: "sub@test.com", Coin: currency.NewCode("btc"), Status: new(uint64(0)), StartTime: start, EndTime: start.Add(time.Hour), Limit: 500, Offset: 10},
				exp:  url.Values{"email": {"sub@test.com"}, "coin": {"BTC"}, "status": {"0"}, "startTime": {"1652942429211"}, "endTime": {"1652946029211"}, "limit": {"500"}, "offset": {"10"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/capital/sub-account/deposit/history", tc.exp, `[]`)
				_, err := ex.GetSubAccountDepositHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetSubAccountDepositHistory must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	email := "sub@test.com"
	if !mockTests {
		subAccounts, err := e.GetSubAccountInformation(t.Context(), nil)
		require.NoError(t, err, "GetSubAccountInformation must not error")
		require.NotNil(t, subAccounts, "GetSubAccountInformation must return the sub-accounts")
		if len(subAccounts.SubAccounts) == 0 {
			t.Skip("the account has no sub-accounts")
		}
		email = subAccounts.SubAccounts[0].Email
	}
	result, err := e.GetSubAccountDepositHistory(t.Context(), &SubAccountDepositHistoryRequest{Email: email})
	require.NoError(t, err, "GetSubAccountDepositHistory must not error")
	if mockTests {
		exp := []DepositRecordResponse{{
			Amount:       9.9749,
			Coin:         currency.BTC,
			Network:      "btc",
			Status:       4,
			Address:      "bc1qxurvdd7tzn09agdvg3j8xpm3f7e978y07wg83s",
			TxID:         "0x1b4b8c8090d15e3c1b0476b1c19118b1f00066e01de567cd7bc5b6e9c100193f",
			InsertTime:   types.Time(time.UnixMilli(1652942429211)),
			ConfirmTimes: "0/0",
		}}
		assert.Equal(t, exp, result, "GetSubAccountDepositHistory should decode every field")
	}
}
