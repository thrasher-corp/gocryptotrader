package coinbase

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// testAppTransactionID is the transaction the docs' Show Transaction example returns
const testAppTransactionID = "57ffb4ae-0c59-5430-bcd3-3f98f797a66c"

func TestAppSendMoney(t *testing.T) {
	t.Parallel()
	_, err := e.App().SendMoney(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SendMoney must reject a nil request")
	_, err = e.App().SendMoney(t.Context(), &AppSendMoneyRequest{})
	require.ErrorIs(t, err, errAccountIDRequired, "SendMoney must require an account ID")
	_, err = e.App().SendMoney(t.Context(), &AppSendMoneyRequest{AccountID: testAppAccountID})
	require.ErrorIs(t, err, errAppSendRecipientRequired, "SendMoney must require a recipient")
	_, err = e.App().SendMoney(t.Context(), &AppSendMoneyRequest{AccountID: testAppAccountID, To: "1AUJ8z5RuHRTqD1eikyfUUetzGmdWLGkpT"})
	require.ErrorIs(t, err, errInvalidAmount, "SendMoney must require an amount")
	_, err = e.App().SendMoney(t.Context(), &AppSendMoneyRequest{AccountID: testAppAccountID, To: "1AUJ8z5RuHRTqD1eikyfUUetzGmdWLGkpT", Amount: 0.1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SendMoney must require a currency")
	_, err = e.App().SendMoney(t.Context(), &AppSendMoneyRequest{AccountID: testAppAccountID, To: "1AUJ8z5RuHRTqD1eikyfUUetzGmdWLGkpT", Amount: 0.1, Currency: currency.BTC, TravelRuleData: &AppTravelRuleData{IsSelf: "yes"}})
	require.ErrorIs(t, err, errAppInvalidTravelRuleIsSelf, "SendMoney must refuse invalid travel rule data")

	if !mockTests {
		t.Skip("a send cannot be undone, so SendMoney is tested against its documented response only")
	}
	resp, err := e.App().SendMoney(t.Context(), &AppSendMoneyRequest{
		AccountID:      testAppAccountID,
		To:             "1AUJ8z5RuHRTqD1eikyfUUetzGmdWLGkpT",
		Amount:         0.1,
		Currency:       currency.BTC,
		IdempotencyKey: "df087dce-92a8-45cf-b112-60aad22c0976",
		Network:        "bitcoin",
		TravelRuleData: &AppTravelRuleData{
			BeneficiaryWalletType: AppTravelRuleWalletTypeExchange,
			IsSelf:                AppTravelRuleIsSelfTrue,
			BeneficiaryName:       "San Francisco City Hall",
			BeneficiaryAddress: &AppTravelRuleAddress{
				Address1:   "1 Dr Carlton B Goodlett Pl",
				Address2:   "Unit 201",
				City:       "San Francisco",
				State:      "CA",
				PostalCode: "94102",
				Country:    "US",
			},
			BeneficiaryFinancialInstitution: "009BBC11-03FB-4224-8CC7-CCE7CCDE4706",
			TransferPurpose:                 "Test",
		},
	})
	require.NoError(t, err, "SendMoney must not error")
	exp := &AppTransaction{
		ID:     "3c04e35e-8e5a-5ff1-9155-00675db4ac02",
		Type:   AppTransactionTypeSend,
		Status: AppTransactionStatusPending,
		Amount: AppMoney{
			Amount:   -0.1,
			Currency: currency.BTC,
		},
		NativeAmount: AppMoney{
			Amount:   -1,
			Currency: currency.USD,
		},
		CreatedAt:    time.Date(2015, 1, 31, 20, 49, 2, 0, time.UTC),
		UpdatedAt:    appTestTime(t, "2015-03-31T17:25:29-07:00"),
		Resource:     "transaction",
		ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/transactions/3c04e35e-8e5a-5ff1-9155-00675db4ac02",
		Network: &AppTransactionNetwork{
			Status: AppTransactionNetworkStatusPending,
		},
		To: &AppTransactionParty{
			Resource: "bitcoin_address",
			Address:  "1AUJ8z5RuHRTqD1eikyfUUetzGmdWLGkpT",
		},
	}
	assert.Equal(t, exp, resp, "SendMoney should decode every field")

	// The fixture's body holds only the docs' fields, so a body with every field is checked as it is sent
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "SendMoney should post the send")
		assert.Equal(t, "/v2/accounts/"+testAppAccountID+"/transactions", r.URL.Path, "SendMoney should post to the account's transactions")
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			assert.JSONEq(t, `{"type":"send","to":"rb@coinbase.com","amount":"0.00000001","currency":"XRP","description":"tip","skip_notifications":true,"idem":"df087dce-92a8-45cf-b112-60aad22c0976","destination_tag":"123","network":"ripple","travel_rule_data":{"beneficiary_wallet_type":"WALLET_TYPE_SELF_HOSTED","is_self":"IS_SELF_FALSE","beneficiary_address":{"address3":"Level 3","country":"GB"}}}`, string(body), "SendMoney should send every field")
		}
		_, err = w.Write([]byte(`{"data":{"id":"3c04e35e-8e5a-5ff1-9155-00675db4ac02"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	resp, err = ex.App().SendMoney(t.Context(), &AppSendMoneyRequest{
		AccountID:         testAppAccountID,
		To:                "rb@coinbase.com",
		Amount:            1e-8,
		Currency:          currency.XRP.Lower(),
		Description:       "tip",
		SkipNotifications: true,
		IdempotencyKey:    "df087dce-92a8-45cf-b112-60aad22c0976",
		DestinationTag:    "123",
		Network:           "ripple",
		TravelRuleData: &AppTravelRuleData{
			BeneficiaryWalletType: AppTravelRuleWalletTypeSelfHosted,
			IsSelf:                AppTravelRuleIsSelfFalse,
			BeneficiaryAddress:    &AppTravelRuleAddress{Address3: "Level 3", Country: "GB"},
		},
	})
	require.NoError(t, err, "SendMoney must not error with every field set")
	assert.Equal(t, &AppTransaction{ID: "3c04e35e-8e5a-5ff1-9155-00675db4ac02"}, resp, "SendMoney should return the send")

	req := &AppSendMoneyRequest{AccountID: testAppAccountID, To: "1AUJ8z5RuHRTqD1eikyfUUetzGmdWLGkpT", Amount: 0.1, Currency: currency.BTC}
	_, err = newResponseServerExchange(t, `null`).App().SendMoney(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "SendMoney should error for a null response")
	_, err = newUnavailableServerExchange(t).App().SendMoney(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendMoney should return the request failure")
}

func TestAppTravelRuleDataValidate(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (*AppTravelRuleData)(nil).validate(), "validate should accept no travel rule data")
	assert.NoError(t, (&AppTravelRuleData{}).validate(), "validate should accept empty travel rule data")
	assert.NoError(t, (&AppTravelRuleData{BeneficiaryWalletType: AppTravelRuleWalletTypeExchange, IsSelf: AppTravelRuleIsSelfTrue}).validate(), "validate should accept documented values")
	// The docs' examples name a WALLET_TYPE_SELF_EXCHANGE their argument table does not list
	assert.ErrorIs(t, (&AppTravelRuleData{BeneficiaryWalletType: "WALLET_TYPE_SELF_EXCHANGE"}).validate(), errAppInvalidTravelRuleWalletType, "validate should refuse an undocumented wallet type")
	assert.ErrorIs(t, (&AppTravelRuleData{IsSelf: "true"}).validate(), errAppInvalidTravelRuleIsSelf, "validate should refuse an undocumented is self value")
}

func TestAppListTransactions(t *testing.T) {
	t.Parallel()
	_, err := e.App().ListTransactions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListTransactions must reject a nil request")
	_, err = e.App().ListTransactions(t.Context(), &AppListTransactionsRequest{})
	require.ErrorIs(t, err, errAccountIDRequired, "ListTransactions must require an account ID")
	_, err = e.App().ListTransactions(t.Context(), &AppListTransactionsRequest{AccountID: testAppAccountID, Page: &AppPageRequest{Order: "up"}})
	require.ErrorIs(t, err, errInvalidAppOrder, "ListTransactions must refuse an invalid page")

	skipLiveWithoutCredentials(t)
	resp, err := e.App().ListTransactions(t.Context(), &AppListTransactionsRequest{AccountID: testAppAccount(t), Page: &AppPageRequest{Limit: 25, Order: AppOrderDescending}})
	require.NoError(t, err, "ListTransactions must not error")
	if mockTests {
		exp := &AppListResponse[AppTransaction]{
			Data: []AppTransaction{
				{
					ID:     "4117f7d6-5694-5b36-bc8f-847509850ea4",
					Type:   AppTransactionTypeBuy,
					Status: AppTransactionStatusPending,
					Amount: AppMoney{
						Amount:   486.34313725,
						Currency: currency.BTC,
					},
					NativeAmount: AppMoney{
						Amount:   4863.43,
						Currency: currency.USD,
					},
					CreatedAt:    appTestTime(t, "2015-03-26T23:44:08-07:00"),
					UpdatedAt:    appTestTime(t, "2015-03-26T23:44:08-07:00"),
					Resource:     "transaction",
					ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/transactions/4117f7d6-5694-5b36-bc8f-847509850ea4",
					Details: &AppTransactionDetails{
						Title:    "Bought bitcoin",
						Subtitle: "using Capital One Bank",
					},
				},
				{
					ID:     "005e55d1-f23a-5d1e-80a4-72943682c055",
					Type:   AppTransactionTypeRequest,
					Status: AppTransactionStatusPending,
					Amount: AppMoney{
						Amount:   0.1,
						Currency: currency.BTC,
					},
					NativeAmount: AppMoney{
						Amount:   1,
						Currency: currency.USD,
					},
					CreatedAt:    appTestTime(t, "2015-03-24T18:32:35-07:00"),
					UpdatedAt:    time.Date(2015, 1, 31, 20, 49, 2, 0, time.UTC),
					Resource:     "transaction",
					ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/transactions/005e55d1-f23a-5d1e-80a4-72943682c055",
					Details: &AppTransactionDetails{
						Title:    "Requested bitcoin",
						Subtitle: "from rb@coinbase.com",
					},
					To: &AppTransactionParty{
						Resource: "email",
						Email:    "rb@coinbase.com",
					},
				},
				{
					ID:     "ff01bbc6-c4ad-59e1-9601-e87b5b709458",
					Type:   AppTransactionTypeTransfer,
					Status: AppTransactionStatusCompleted,
					Amount: AppMoney{
						Amount:   -5,
						Currency: currency.BTC,
					},
					NativeAmount: AppMoney{
						Amount:   -50,
						Currency: currency.USD,
					},
					CreatedAt:    appTestTime(t, "2015-03-12T15:51:38-07:00"),
					UpdatedAt:    time.Date(2015, 1, 31, 20, 49, 2, 0, time.UTC),
					Resource:     "transaction",
					ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/transactions/ff01bbc6-c4ad-59e1-9601-e87b5b709458",
					Details: &AppTransactionDetails{
						Title:    "Transferred bitcoin",
						Subtitle: "to Secondary Account",
					},
					To: &AppTransactionParty{
						ID:           "58542935-67b5-56e1-a3f9-42686e07fa40",
						Resource:     "account",
						ResourcePath: "/v2/accounts/58542935-67b5-56e1-a3f9-42686e07fa40",
					},
				},
				{
					ID:     "57ffb4ae-0c59-5430-bcd3-3f98f797a66c",
					Type:   AppTransactionTypeSend,
					Status: AppTransactionStatusCompleted,
					Amount: AppMoney{
						Amount:   -0.001,
						Currency: currency.BTC,
					},
					NativeAmount: AppMoney{
						Amount:   -0.01,
						Currency: currency.USD,
					},
					CreatedAt:    appTestTime(t, "2015-03-11T13:13:35-07:00"),
					UpdatedAt:    appTestTime(t, "2015-03-26T15:55:43-07:00"),
					Resource:     "transaction",
					ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/transactions/57ffb4ae-0c59-5430-bcd3-3f98f797a66c",
					Details: &AppTransactionDetails{
						Title:    "Send bitcoin",
						Subtitle: "to User 2",
					},
					Network: &AppTransactionNetwork{
						Status: AppTransactionNetworkStatusOffBlockchain,
						Name:   "bitcoin",
					},
					To: &AppTransactionParty{
						ID:           "a6b4c2df-a62c-5d68-822a-dd4e2102e703",
						Resource:     "user",
						ResourcePath: "/v2/users/a6b4c2df-a62c-5d68-822a-dd4e2102e703",
					},
				},
			},
			Pagination: AppPagination{
				Limit: 25,
				Order: AppOrderDescending,
			},
		}
		assert.Equal(t, exp, resp, "ListTransactions should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Data), 25, "ListTransactions should return at most the page's limit")
	}

	_, err = newUnavailableServerExchange(t).App().ListTransactions(t.Context(), &AppListTransactionsRequest{AccountID: testAppAccountID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListTransactions should return the request failure")
}

func TestAppGetTransaction(t *testing.T) {
	t.Parallel()
	_, err := e.App().GetTransaction(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTransaction must reject a nil request")
	_, err = e.App().GetTransaction(t.Context(), &AppGetTransactionRequest{TransactionID: testAppTransactionID})
	require.ErrorIs(t, err, errAccountIDRequired, "GetTransaction must require an account ID")
	_, err = e.App().GetTransaction(t.Context(), &AppGetTransactionRequest{AccountID: testAppAccountID})
	require.ErrorIs(t, err, errAppTransactionIDRequired, "GetTransaction must require a transaction ID")

	skipLiveWithoutCredentials(t)
	req := &AppGetTransactionRequest{AccountID: testAppAccountID, TransactionID: testAppTransactionID}
	if !mockTests {
		req.AccountID = testAppAccount(t)
		transactions, err := e.App().ListTransactions(t.Context(), &AppListTransactionsRequest{AccountID: req.AccountID, Page: &AppPageRequest{Limit: 1}})
		require.NoError(t, err, "ListTransactions must not error")
		if len(transactions.Data) == 0 {
			t.Skip("the account has no transactions")
		}
		req.TransactionID = transactions.Data[0].ID
	}
	resp, err := e.App().GetTransaction(t.Context(), req)
	require.NoError(t, err, "GetTransaction must not error")
	if mockTests {
		exp := &AppTransaction{
			ID:     "57ffb4ae-0c59-5430-bcd3-3f98f797a66c",
			Type:   AppTransactionTypeSend,
			Status: AppTransactionStatusCompleted,
			Amount: AppMoney{
				Amount:   -0.001,
				Currency: currency.BTC,
			},
			NativeAmount: AppMoney{
				Amount:   -0.01,
				Currency: currency.USD,
			},
			CreatedAt:    appTestTime(t, "2015-03-11T13:13:35-07:00"),
			UpdatedAt:    appTestTime(t, "2015-03-26T15:55:43-07:00"),
			Resource:     "transaction",
			ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/transactions/57ffb4ae-0c59-5430-bcd3-3f98f797a66c",
			Details: &AppTransactionDetails{
				Title:    "Send bitcoin",
				Subtitle: "to User 2",
			},
			Network: &AppTransactionNetwork{
				Status: AppTransactionNetworkStatusOffBlockchain,
				Name:   "bitcoin",
			},
			To: &AppTransactionParty{
				ID:           "a6b4c2df-a62c-5d68-822a-dd4e2102e703",
				Resource:     "user",
				ResourcePath: "/v2/users/a6b4c2df-a62c-5d68-822a-dd4e2102e703",
			},
		}
		assert.Equal(t, exp, resp, "GetTransaction should decode every field")
	} else {
		assert.Equal(t, req.TransactionID, resp.ID, "GetTransaction should return the transaction")
	}

	_, err = newResponseServerExchange(t, `null`).App().GetTransaction(t.Context(), &AppGetTransactionRequest{AccountID: testAppAccountID, TransactionID: testAppTransactionID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTransaction should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetTransaction(t.Context(), &AppGetTransactionRequest{AccountID: testAppAccountID, TransactionID: testAppTransactionID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTransaction should return the request failure")
}
