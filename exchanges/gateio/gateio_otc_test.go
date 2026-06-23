package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetFlatStablecoinQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatStablecoinQuote(t.Context(), &OTCQuoteRequest{})
	require.ErrorIs(t, err, errOTCSideRequired)

	_, err = e.GetFiatStablecoinQuote(t.Context(), &OTCQuoteRequest{Side: "PAY"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetFiatStablecoinQuote(t.Context(), &OTCQuoteRequest{Side: "PAY", PayCoin: currency.USD})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFiatStablecoinQuote(t.Context(), &OTCQuoteRequest{
		Side:      "PAY",
		PayCoin:   currency.USD,
		GetCoin:   currency.USDT,
		PayAmount: 100,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateFlatOrder(t *testing.T) {
	t.Parallel()
	arg := &OTCFiatOrderRequest{}
	_, err := e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, errOTCOrderTypeRequired)

	arg.Type = "BUY"
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, errOTCQuoteTokenRequired)

	arg.QuoteToken = "token"
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, errOTCBankIDRequired)

	arg.BankID = "2"
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.CryptoCurrency = currency.USDT
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.FiatCurrency = currency.USD
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	arg.CryptoAmount = 1
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CreateFiatOrder(t.Context(), &OTCFiatOrderRequest{
		Type:           "BUY",
		Side:           "FIAT",
		FiatCurrency:   currency.USD,
		CryptoCurrency: currency.USDT,
		CryptoAmount:   100,
		FiatAmount:     100,
		QuoteToken:     "some_token",
		BankID:         "72",
	})
	require.NoError(t, err)
}

func TestCreateStablecoinOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateStablecoinOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CreateStablecoinOrder(t.Context(), &OTCStablecoinOrderRequest{
		PayCoin:    currency.USD,
		GetCoin:    currency.USDT,
		PayAmount:  100,
		QuoteToken: "some_token",
	})
	require.NoError(t, err)
}

func TestGetUserDefaultBankAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserDefaultBankAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result, "result should not be nil")
}

func TestGetUserBankCardList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserBankCardList(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result, "result should not be nil")
}

func TestCreateBankCard(t *testing.T) {
	t.Parallel()
	_, err := e.CreateBankCard(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &OTCBankCreateMultipartRequest{}
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errBankAccountNameRequired)

	arg.BankAccountName = "John Doe"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errBankNameRequired)

	arg.BankName = "Bank of Test"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errBankCountryRequired)

	arg.BankCountry = "US"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errBankAddressRequired)

	arg.BankAddress = "123 Test Street"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errIBANAddresRequired)

	arg.IBAN = "GB33BUKB20201555555555"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errSwiftAddressRequired)

	arg.Swift = "BUKBGB22"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errDocumentationFileRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.DocumentationFile = "base64encodeddocument"
	result, err := e.CreateBankCard(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result, "result should not be nil")
}

func TestDeleteBankCard(t *testing.T) {
	t.Parallel()
	err := e.DeleteBankCard(t.Context(), "")
	require.ErrorIs(t, err, errOTCBankIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.DeleteBankCard(t.Context(), "123")
	require.NoError(t, err)
}

func TestSubmitBankCardSupplementMaterials(t *testing.T) {
	t.Parallel()
	err := e.SubmitBankCardSupplementMaterials(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &OTCBankPersonalSupplementMultipartRequest{}
	err = e.SubmitBankCardSupplementMaterials(t.Context(), arg)
	require.ErrorIs(t, err, errOTCBankIDRequired)

	arg.BankID = "123"
	err = e.SubmitBankCardSupplementMaterials(t.Context(), arg)
	require.ErrorIs(t, err, errDocumentationFileRequired)

	arg.IDDocumentFront = "base64frontdocument"
	err = e.SubmitBankCardSupplementMaterials(t.Context(), arg)
	require.ErrorIs(t, err, errDocumentationFileRequired)

	arg.IDDocumentBack = "base64backdocument"
	err = e.SubmitBankCardSupplementMaterials(t.Context(), arg)
	require.ErrorIs(t, err, errBankAddressRequired)

	arg.AddressProof = "base64addressproof"
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SubmitBankCardSupplementMaterials(t.Context(), arg)
	require.NoError(t, err)
}

func TestSubmitEnterpriseBankCardSupplementMaterials(t *testing.T) {
	t.Parallel()
	err := e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &OTCBankEnterpriseSupplementMultipartRequest{}
	err = e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), arg)
	require.ErrorIs(t, err, errOTCBankIDRequired)

	arg.BankID = "123"
	err = e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), arg)
	require.ErrorIs(t, err, errBusinessLicenseCertificateRequired)

	arg.Certificate = "base64certificate"
	err = e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), arg)
	require.ErrorIs(t, err, errShareholdersRequired)

	arg.ShareHolders = "base64shareholders"
	err = e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), arg)
	require.ErrorIs(t, err, errPassportRequired)

	arg.Passport = "base64passport"
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), arg)
	require.NoError(t, err)
}

func TestSetDefaultBankCard(t *testing.T) {
	t.Parallel()
	err := e.SetDefaultBankCard(t.Context(), "")
	require.ErrorIs(t, err, errOTCBankIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetDefaultBankCard(t.Context(), "123")
	require.NoError(t, err)
}

func TestGetChecklistOfMaterialsToSupplementForBankCard(t *testing.T) {
	t.Parallel()
	_, err := e.GetCheckListOfMaterialsToSupplementForBankCard(t.Context(), "")
	require.ErrorIs(t, err, errOTCBankIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCheckListOfMaterialsToSupplementForBankCard(t.Context(), "123")
	require.NoError(t, err)
}

func TestMarkFlatOrderAsPaid(t *testing.T) {
	t.Parallel()
	err := e.MarkFiatOrderAsPaid(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.MarkFiatOrderAsPaid(t.Context(), "203")
	require.NoError(t, err)
}

func TestCancelFlatOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelFiatOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CancelFiatOrder(t.Context(), "203")
	require.NoError(t, err)
}

func TestGetFlatOrderList(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetFiatOrderList(t.Context(), "", "", currency.EMPTYCODE, currency.EMPTYCODE, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFiatOrderList(t.Context(), "BUY", "", currency.EMPTYCODE, currency.USDT, startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStablecoinOrderList(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetStablecoinOrderList(t.Context(), currency.EMPTYCODE, "", endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStablecoinOrderList(t.Context(), currency.USDT, "", startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFlatOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatOrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFiatOrderDetail(t.Context(), "203")
	require.NoError(t, err)
	assert.NotNil(t, result)
}
