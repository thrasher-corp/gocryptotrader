package gateio

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	otcTestBankID          = "72"
	otcTestOrderID         = "203"
	otcTestQuoteToken      = "quote-token-1"
	otcTestPreUploadedFile = "b3RjX3RlbXAvMTIzNDU2Nzg5MC9iYW5rLzIwMjYwODMxMDc1MTI1MjgyMjMzOTQ1OS5wbmc="
)

func TestOTCActionResponseAsError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&OTCActionResponse{Message: "success"}).AsError(), "a zero code should not error")
	err := (&OTCActionResponse{Code: 10010400, Message: "parameter error"}).AsError()
	require.Error(t, err, "a non-zero code must error")
	assert.ErrorContains(t, err, "parameter error", "error should carry the message")
}

// TestOTCActionResponseErrors ensures a business error carried in a 200 response is returned rather than dropped
func TestOTCActionResponseErrors(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the rejected requests are only recorded as fixtures")
	}
	const rejectedBankID = "73"
	for name, call := range map[string]func() error{
		"DeleteBankCard":     func() error { return e.DeleteBankCard(t.Context(), rejectedBankID) },
		"SetDefaultBankCard": func() error { return e.SetDefaultBankCard(t.Context(), rejectedBankID) },
		"SubmitBankCardSupplementMaterials": func() error {
			return e.SubmitBankCardSupplementMaterials(t.Context(), &OTCBankPersonalSupplementMultipartRequest{BankID: rejectedBankID, AddressProof: "YWRkcmVzcw=="})
		},
		"SubmitEnterpriseBankCardSupplementMaterials": func() error {
			return e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), &OTCBankEnterpriseSupplementMultipartRequest{BankID: rejectedBankID, Certificate: "Y2VydGlmaWNhdGU="})
		},
	} {
		assert.ErrorContainsf(t, call(), "parameter error", "%s should return the response's business error", name)
	}
}

func TestGetFiatStablecoinQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatStablecoinQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFiatStablecoinQuote must reject a nil request")
	_, err = e.GetFiatStablecoinQuote(t.Context(), &OTCQuoteRequest{})
	require.ErrorIs(t, err, errOTCSideRequired, "GetFiatStablecoinQuote must require a side")
	_, err = e.GetFiatStablecoinQuote(t.Context(), &OTCQuoteRequest{Side: "PAY"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFiatStablecoinQuote must require a pay coin")
	_, err = e.GetFiatStablecoinQuote(t.Context(), &OTCQuoteRequest{Side: "PAY", PayCoin: currency.USD})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFiatStablecoinQuote must require a get coin")
	_, err = e.GetFiatStablecoinQuote(t.Context(), &OTCQuoteRequest{Side: "PAY", PayCoin: currency.USD, GetCoin: currency.USDT, GetAmount: 1})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "GetFiatStablecoinQuote must require a pay amount for a PAY quote")
	_, err = e.GetFiatStablecoinQuote(t.Context(), &OTCQuoteRequest{Side: "GET", PayCoin: currency.USD, GetCoin: currency.USDT, PayAmount: 1})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "GetFiatStablecoinQuote must require a get amount for a GET quote")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatStablecoinQuote(t.Context(), &OTCQuoteRequest{Side: "PAY", PayCoin: currency.USD, GetCoin: currency.USDT, PayAmount: 30000, CreateQuoteToken: "1"})
	require.NoError(t, err, "GetFiatStablecoinQuote must not error")
	if mockTests {
		exp := &OTCQuoteResponse{
			Type:                   "BUY",
			PayCoin:                currency.USD,
			GetCoin:                currency.USDT,
			PayAmount:              30000,
			GetAmount:              29891,
			Rate:                   1.0036,
			ReciprocalExchangeRate: 0.9964,
			Side:                   "PAY",
			OrderType:              "FIAT",
			QuoteToken:             otcTestQuoteToken,
			ValidityPeriodSeconds:  300,
			RefreshLimit:           20,
			RefreshLimitMessage:    "refresh limit reached",
		}
		assert.Equal(t, exp, result, "GetFiatStablecoinQuote should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFiatStablecoinQuote should return a quote")
}

func TestCreateFiatOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFiatOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFiatOrder must reject a nil request")

	arg := &OTCFiatOrderRequest{}
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, errOTCOrderTypeRequired, "CreateFiatOrder must require an order type")
	arg.Type = "BUY"
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateFiatOrder must require a side")
	arg.Side = "PAY"
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, errOTCQuoteTokenRequired, "CreateFiatOrder must require a quote token")
	arg.QuoteToken = otcTestQuoteToken
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, errOTCBankIDRequired, "CreateFiatOrder must require a bank ID")
	arg.BankID = otcTestBankID
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateFiatOrder must require a crypto currency")
	arg.CryptoCurrency = currency.USDT
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateFiatOrder must require a fiat currency")
	arg.FiatCurrency = currency.USD
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "CreateFiatOrder must require a crypto amount")
	arg.CryptoAmount = 29891
	_, err = e.CreateFiatOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "CreateFiatOrder must require a fiat amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.FiatAmount = 30000
	arg.ReceiveType = "GATE"
	result, err := e.CreateFiatOrder(t.Context(), arg)
	require.NoError(t, err, "CreateFiatOrder must not error")
	if mockTests {
		exp := &OTCActionResponse{Message: "success", Timestamp: types.Time(time.Unix(1752051076, 0))}
		assert.Equal(t, exp, result, "CreateFiatOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateFiatOrder should return a result")
}

func TestCreateStablecoinOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateStablecoinOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateStablecoinOrder must reject a nil request")

	arg := &OTCStablecoinOrderRequest{}
	_, err = e.CreateStablecoinOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateStablecoinOrder must require a pay coin")
	arg.PayCoin = currency.USDT
	_, err = e.CreateStablecoinOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateStablecoinOrder must require a get coin")
	arg.GetCoin = currency.USDC
	_, err = e.CreateStablecoinOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "CreateStablecoinOrder must require a pay amount")
	arg.PayAmount = 30000
	_, err = e.CreateStablecoinOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "CreateStablecoinOrder must require a get amount")
	arg.GetAmount = 29990
	_, err = e.CreateStablecoinOrder(t.Context(), arg)
	require.ErrorIs(t, err, errOTCSideRequired, "CreateStablecoinOrder must require a side")
	arg.Side = "PAY"
	_, err = e.CreateStablecoinOrder(t.Context(), arg)
	require.ErrorIs(t, err, errOTCQuoteTokenRequired, "CreateStablecoinOrder must require a quote token")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.QuoteToken = otcTestQuoteToken
	result, err := e.CreateStablecoinOrder(t.Context(), arg)
	require.NoError(t, err, "CreateStablecoinOrder must not error")
	if mockTests {
		exp := &OTCActionResponse{Message: "success", Timestamp: types.Time(time.Unix(1752051076, 0))}
		assert.Equal(t, exp, result, "CreateStablecoinOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateStablecoinOrder should return a result")
}

func TestGetUserBankCardList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserBankCardList(t.Context())
	require.NoError(t, err, "GetUserBankCardList must not error")
	if mockTests {
		exp := []*OTCBankCard{{
			ID:                   otcTestBankID,
			BankAccountName:      "John Doe",
			BankName:             "Bank of Test",
			BankCountry:          "US",
			BankAddress:          "123 Test Street",
			IBAN:                 "GB33BUKB20201555555555",
			Swift:                "BUKBGB22",
			RemittanceLineNumber: "021000021",
			AgentBankName:        "Agent Bank",
			AgentBankSwift:       "AGNTUS33",
			SubmitTime:           "2026-06-01 08:00:00",
			UpdateTime:           "2026-06-02 09:30:00",
			Status:               "approved",
			IsDefault:            true,
		}}
		assert.Equal(t, exp, result, "GetUserBankCardList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserBankCardList should return bank cards")
}

func TestGetUserBankCardListWithoutData(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v4/otc/bank/list", r.URL.Path, "request path should be the bank card list")
		_, err := w.Write([]byte(`{"code":0,"message":"success","data":null,"timestamp":1752051076}`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	_, err := ex.GetUserBankCardList(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResults, "GetUserBankCardList should report a response without data")
}

func TestCreateBankCard(t *testing.T) {
	t.Parallel()
	_, err := e.CreateBankCard(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateBankCard must reject a nil request")

	arg := &OTCBankCreateMultipartRequest{}
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errBankAccountNameRequired, "CreateBankCard must require a bank account name")
	arg.BankAccountName = "John Doe"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errBankNameRequired, "CreateBankCard must require a bank name")
	arg.BankName = "Bank of Test"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errBankCountryRequired, "CreateBankCard must require a bank country")
	arg.BankCountry = "US"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errBankAddressRequired, "CreateBankCard must require a bank address")
	arg.BankAddress = "123 Test Street"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errIBANRequired, "CreateBankCard must require an IBAN")
	arg.IBAN = "GB33BUKB20201555555555"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errSwiftCodeRequired, "CreateBankCard must require a SWIFT code")
	arg.Swift = "BUKBGB22"
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errDocumentationFileRequired, "CreateBankCard must require account-opening proof")
	arg.DocumentationFile = "ZmlsZQ=="
	arg.DocumentationFileKey = otcTestPreUploadedFile
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errDocumentationFileConflict, "CreateBankCard must reject both a file and a file key")
	arg.DocumentationFile = ""
	_, err = e.CreateBankCard(t.Context(), arg)
	require.ErrorIs(t, err, errFileTypeRequired, "CreateBankCard must require a file type with a file key")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.FileType = "image/png"
	result, err := e.CreateBankCard(t.Context(), arg)
	require.NoError(t, err, "CreateBankCard must not error")
	if mockTests {
		assert.Equal(t, &OTCBankCardRequestResponse{BankID: 72, Status: 1}, result, "CreateBankCard should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateBankCard should return the bank card")
}

func TestDeleteBankCard(t *testing.T) {
	t.Parallel()
	err := e.DeleteBankCard(t.Context(), "")
	require.ErrorIs(t, err, errOTCBankIDRequired, "DeleteBankCard must require a bank ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.DeleteBankCard(t.Context(), otcTestBankID), "DeleteBankCard should not error")
}

func TestSetDefaultBankCard(t *testing.T) {
	t.Parallel()
	err := e.SetDefaultBankCard(t.Context(), "")
	require.ErrorIs(t, err, errOTCBankIDRequired, "SetDefaultBankCard must require a bank ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.SetDefaultBankCard(t.Context(), otcTestBankID), "SetDefaultBankCard should not error")
}

func TestGetCheckListOfMaterialsToSupplementForBankCard(t *testing.T) {
	t.Parallel()
	_, err := e.GetCheckListOfMaterialsToSupplementForBankCard(t.Context(), "")
	require.ErrorIs(t, err, errOTCBankIDRequired, "GetCheckListOfMaterialsToSupplementForBankCard must require a bank ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCheckListOfMaterialsToSupplementForBankCard(t.Context(), otcTestBankID)
	require.NoError(t, err, "GetCheckListOfMaterialsToSupplementForBankCard must not error")
	if mockTests {
		exp := &OTCBankSupplementChecklistResponse{
			UserType: "personal",
			Items: []OTCBankSupplementChecklistItem{
				{Description: "Front of a government issued ID document"},
				{Description: "Proof of address issued within the last three months"},
			},
		}
		assert.Equal(t, exp, result, "GetCheckListOfMaterialsToSupplementForBankCard should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCheckListOfMaterialsToSupplementForBankCard should return a checklist")
}

func TestSubmitBankCardSupplementMaterials(t *testing.T) {
	t.Parallel()
	err := e.SubmitBankCardSupplementMaterials(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubmitBankCardSupplementMaterials must reject a nil request")
	err = e.SubmitBankCardSupplementMaterials(t.Context(), &OTCBankPersonalSupplementMultipartRequest{})
	require.ErrorIs(t, err, errOTCBankIDRequired, "SubmitBankCardSupplementMaterials must require a bank ID")
	err = e.SubmitBankCardSupplementMaterials(t.Context(), &OTCBankPersonalSupplementMultipartRequest{BankID: otcTestBankID})
	require.ErrorIs(t, err, errSupplementMaterialRequired, "SubmitBankCardSupplementMaterials must require a material")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.SubmitBankCardSupplementMaterials(t.Context(), &OTCBankPersonalSupplementMultipartRequest{
		BankID:          otcTestBankID,
		IDDocumentFront: "ZnJvbnQ=",
		IDDocumentBack:  "YmFjaw==",
		AddressProof:    "YWRkcmVzcw==",
	})
	assert.NoError(t, err, "SubmitBankCardSupplementMaterials should not error")
}

func TestSubmitEnterpriseBankCardSupplementMaterials(t *testing.T) {
	t.Parallel()
	err := e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubmitEnterpriseBankCardSupplementMaterials must reject a nil request")
	err = e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), &OTCBankEnterpriseSupplementMultipartRequest{})
	require.ErrorIs(t, err, errOTCBankIDRequired, "SubmitEnterpriseBankCardSupplementMaterials must require a bank ID")
	err = e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), &OTCBankEnterpriseSupplementMultipartRequest{BankID: otcTestBankID})
	require.ErrorIs(t, err, errSupplementMaterialRequired, "SubmitEnterpriseBankCardSupplementMaterials must require a material")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.SubmitEnterpriseBankCardSupplementMaterials(t.Context(), &OTCBankEnterpriseSupplementMultipartRequest{
		BankID:            otcTestBankID,
		RelationshipProof: `{"certificate":[{"key":"otc_temp/1234567890/bank/certificate.pdf","file_type":"application/pdf"}]}`,
	})
	assert.NoError(t, err, "SubmitEnterpriseBankCardSupplementMaterials should not error")
}

func TestPreUploadOTCFile(t *testing.T) {
	t.Parallel()
	_, err := e.PreUploadOTCFile(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PreUploadOTCFile must reject a nil request")
	_, err = e.PreUploadOTCFile(t.Context(), &OTCPreUploadRequest{})
	require.ErrorIs(t, err, errContentTypeRequired, "PreUploadOTCFile must require a content type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.PreUploadOTCFile(t.Context(), &OTCPreUploadRequest{ContentType: "image/png", Scene: "bank"})
	require.NoError(t, err, "PreUploadOTCFile must not error")
	if mockTests {
		exp := &OTCPreUploadResponse{
			FileKey: otcTestPreUploadedFile,
			URL:     "https://example-bucket.s3.us-east-2.amazonaws.com",
			Fields: OTCPreUploadPolicyFields{
				Key:         "otc_temp/1234567890/bank/202608310751252822339459.png",
				ContentType: "image/png",
				Credential:  "AKIAEXAMPLE/20260831/us-east-2/s3/aws4_request",
				Algorithm:   "AWS4-HMAC-SHA256",
				Date:        "20260831T075125Z",
				Policy:      "eyJleHBpcmF0aW9uIjoiMjAyNi0wOC0zMVQwOToyMToyNVoiLCJjb25kaXRpb25zIjpbWyJjb250ZW50LWxlbmd0aC1yYW5nZSIsMSwxMDQ4NTc2MF1dfQ==",
				Signature:   "EXAMPLE_SIGNATURE",
			},
			ExpiresInSeconds: 5400,
		}
		assert.Equal(t, exp, result, "PreUploadOTCFile should decode every field")
		return
	}
	assert.NotEmpty(t, result.FileKey, "PreUploadOTCFile should return a file key")
}

func TestMarkFiatOrderAsPaid(t *testing.T) {
	t.Parallel()
	err := e.MarkFiatOrderAsPaid(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "MarkFiatOrderAsPaid must reject a nil request")
	err = e.MarkFiatOrderAsPaid(t.Context(), &OTCMarkOrderPaidRequest{ClientOrderID: "client-order-id"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "MarkFiatOrderAsPaid must require an order ID")
	err = e.MarkFiatOrderAsPaid(t.Context(), &OTCMarkOrderPaidRequest{OrderID: otcTestOrderID})
	require.ErrorIs(t, err, errPaymentReceiptFileKeyRequired, "MarkFiatOrderAsPaid must require a payment receipt file key")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.MarkFiatOrderAsPaid(t.Context(), &OTCMarkOrderPaidRequest{OrderID: otcTestOrderID, PaymentReceiptFileKey: otcTestPreUploadedFile})
	assert.NoError(t, err, "MarkFiatOrderAsPaid should not error")
}

func TestCancelFiatOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelFiatOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelFiatOrder must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.CancelFiatOrder(t.Context(), otcTestOrderID), "CancelFiatOrder should not error")
}

func TestGetFiatOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFiatOrderList must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetFiatOrderList(t.Context(), &OTCFiatOrderListRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFiatOrderList must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatOrderList(t.Context(), &OTCFiatOrderListRequest{
		Type:           "SELL",
		FiatCurrency:   currency.USD,
		CryptoCurrency: currency.USDT,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         "DONE",
		PageNumber:     1,
		PageSize:       10,
	})
	require.NoError(t, err, "GetFiatOrderList must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetFiatOrderList should return an order list")
		return
	}
	exp := &OTCOrderListResponse{
		PageNumber:      1,
		PageSize:        10,
		TotalPageNumber: 1,
		Count:           1,
		List: []*OTCOrderListItem{{
			Time:               "2025-02-11 07:45:06",
			Timestamp:          types.Time(time.Unix(1739259906, 0)),
			OrderID:            "41",
			TradeNumber:        "20250207043457590939",
			Type:               "SELL",
			Status:             "DONE",
			FiatCurrency:       currency.USD,
			FiatCurrencyInfo:   &NameAndIconURL{Name: "USD", Icon: "https://icon.example/usd.png"},
			FiatAmount:         199600,
			CryptoCurrency:     currency.USDT,
			CryptoCurrencyInfo: &NameAndIconURL{Name: "USDT", Icon: "https://icon.example/usdt.png"},
			CryptoAmount:       200000,
			Rate:               0.998,
			PromotionCode:      "PROMO1",
		}},
	}
	assert.Equal(t, exp, result, "GetFiatOrderList should decode every field")

	// The recorded live response carries its orders under "lists" rather than the documented "list"
	result, err = e.GetFiatOrderList(t.Context(), &OTCFiatOrderListRequest{
		Type:           "BUY",
		CryptoCurrency: currency.USDT,
		StartTime:      startTime,
		EndTime:        endTime,
		PageNumber:     1,
		PageSize:       10,
	})
	require.NoError(t, err, "GetFiatOrderList must not error")
	assert.Equal(t, &OTCOrderListResponse{PageNumber: 1, PageSize: 10, List: []*OTCOrderListItem{}}, result, "GetFiatOrderList should decode the recorded live response")
}

func TestOTCOrderListDataUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"list", "lists"} {
		var got OTCOrderListResponse
		require.NoErrorf(t, got.UnmarshalJSON([]byte(`{"pn":2,"`+key+`":[{"order_id":"41"}]}`)), "UnmarshalJSON must not error for %s", key)
		assert.Equalf(t, OTCOrderListResponse{PageNumber: 2, List: []*OTCOrderListItem{{OrderID: "41"}}}, got, "orders should decode from %s", key)
	}
	var got OTCOrderListResponse
	assert.Error(t, got.UnmarshalJSON([]byte(`{"pn":"x"}`)), "UnmarshalJSON should error on an invalid page number")
}

func TestGetStablecoinOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetStablecoinOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetStablecoinOrderList must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetStablecoinOrderList(t.Context(), &OTCStablecoinOrderListRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetStablecoinOrderList must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStablecoinOrderList(t.Context(), &OTCStablecoinOrderListRequest{
		Currency:   currency.USDT,
		Status:     "DONE",
		StartTime:  startTime,
		EndTime:    endTime,
		PageNumber: 1,
		PageSize:   10,
	})
	require.NoError(t, err, "GetStablecoinOrderList must not error")
	if mockTests {
		exp := &OTCStablecoinOrderListResponse{
			Total:      1,
			PageSize:   10,
			PageNumber: 1,
			TotalPage:  1,
			List: []*OTCStableCoinOrder{{
				ID:                     1,
				TradeNumber:            "89875324974",
				PayCoin:                currency.USDT,
				PayIcon:                "https://icon.example/usdt.png",
				PayAmount:              30000,
				GetCoin:                currency.USDC,
				GetIcon:                "https://icon.example/usdc.png",
				GetAmount:              29990,
				Rate:                   0.9997,
				ReciprocalExchangeRate: 1.0003,
				Status:                 "DONE",
				CreateTimestamp:        types.Time(time.Unix(1757412000, 0)),
				CreateTime:             "2025-09-09 10:00:00",
			}},
		}
		assert.Equal(t, exp, result, "GetStablecoinOrderList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetStablecoinOrderList should return an order list")
}

func TestGetFiatOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatOrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetFiatOrderDetail must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatOrderDetail(t.Context(), "265")
	require.NoError(t, err, "GetFiatOrderDetail must not error")
	if mockTests {
		exp := &OTCOrderDetailResponse{
			OrderID:                       "265",
			UID:                           "2124269088",
			Type:                          "BUY",
			FiatCurrency:                  currency.USD,
			FiatAmount:                    300000,
			CryptoCurrency:                currency.USDT,
			CryptoAmount:                  299700,
			Rate:                          1.001001,
			BankAccountName:               "John Doe",
			BankName:                      "Bank of Test",
			BankCountry:                   "US",
			BankAddress:                   "123 Test Street",
			BankAccountNumberIBAN:         "GB33BUKB20201555555555",
			SwiftCode:                     "BUKBGB22",
			IntermediaryBankName:          "Agent Bank",
			IntermediaryBankSwiftCode:     "AGNTUS33",
			GateBankAccountName:           "Gate Technology",
			GateBankName:                  "Gate Bank",
			GateBankCountry:               "SG",
			GateBankAddress:               "1 Gate Road",
			GateBankAccountNumberIBAN:     "SG00GATE0000000001",
			GateSwiftCode:                 "GATESGSG",
			GateIntermediaryBankName:      "Gate Intermediary",
			GateIntermediaryBankSwiftCode: "GATEUS33",
			GateReferenceCode:             "OTIK7M39QF42",
			Status:                        "PROCESSING",
			CreateTime:                    types.DateTime(time.Date(2025, 3, 7, 7, 51, 52, 0, time.UTC)),
		}
		assert.Equal(t, exp, result, "GetFiatOrderDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFiatOrderDetail should return an order")
}
