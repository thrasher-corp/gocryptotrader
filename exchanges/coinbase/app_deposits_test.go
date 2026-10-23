package coinbase

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// testAppFiatTransferID is the deposit and the withdrawal the docs' Show examples return
	testAppFiatTransferID = "67e0eaec-07d7-54c4-a72c-2e92826897df"
	// testAppUncommittedTransferID is the deposit and the withdrawal the docs' commit examples commit
	testAppUncommittedTransferID = "a333743d-184a-5b5b-abe8-11612fc44ab5"
	// testAppPaymentMethodID is the payment method the docs' examples deposit from and withdraw to
	testAppPaymentMethodID = "83562370-3e5c-51db-87da-752af5ab9559"
)

// testAppFiatTransferKinds are the paths of an account's deposits and withdrawals
var testAppFiatTransferKinds = []string{"deposits", "withdrawals"}

// testAppFiatTransfer returns the docs' list and show resource of a deposit or withdrawal
func testAppFiatTransfer(t *testing.T, resource string) AppFiatTransferResponse {
	t.Helper()
	return AppFiatTransferResponse{
		ID:     testAppFiatTransferID,
		Status: AppFiatTransferStatusCompleted,
		PaymentMethod: AppResourceReference{
			ID:           testAppPaymentMethodID,
			Resource:     "payment_method",
			ResourcePath: "/v2/payment-methods/83562370-3e5c-51db-87da-752af5ab9559",
		},
		Transaction: &AppResourceReference{
			ID:           "441b9494-b3f0-5b98-b9b0-4d82c21c252a",
			Resource:     "transaction",
			ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/transactions/441b9494-b3f0-5b98-b9b0-4d82c21c252a",
		},
		Amount: AppMoney{
			Amount:   10,
			Currency: currency.USD,
		},
		Subtotal: AppMoney{
			Amount:   10,
			Currency: currency.USD,
		},
		Fee: AppMoney{
			Currency: currency.USD,
		},
		CreatedAt:    time.Date(2015, 1, 31, 20, 49, 2, 0, time.UTC),
		UpdatedAt:    appTestTime(t, "2015-02-11T16:54:02-08:00"),
		Resource:     resource,
		ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/" + resource + "s/67e0eaec-07d7-54c4-a72c-2e92826897df",
		Committed:    true,
		PayoutAt:     appTestTime(t, "2015-02-18T16:54:00-08:00"),
	}
}

// testAppDepositTransfer returns the docs' deposit transfer, whose commit holds a user reference and its own ID
func testAppDepositTransfer(id, userReference string, payoutAt time.Time) *AppTransfer {
	return &AppTransfer{
		UserEnteredAmount: Amount{Value: 20, Currency: currency.USD},
		Amount:            Amount{Value: 20, Currency: currency.USD},
		Total:             Amount{Value: 20, Currency: currency.USD},
		Subtotal:          Amount{Value: 20, Currency: currency.USD},
		IdempotencyKey:    id,
		ID:                id,
		Instant:           true,
		Source: AppTransferParty{
			Type:    "EXTERNAL_PAYMENT_METHOD",
			Network: "ach",
			ExternalPaymentMethod: &AppExternalPaymentMethod{
				PaymentMethodID: "5a48fe239b15170130598e9c",
			},
		},
		Target: AppTransferParty{
			Type:    "LEDGER_ACCOUNT",
			Network: "internal_retail",
			LedgerAccount: &AppLedgerAccount{
				AccountID: "6c770048-a3aa-580b-b153-2a6791649ee4",
				Currency:  currency.USD,
				Owner: AppLedgerAccountOwner{
					ID:   "5a48fda3bbf66c03a6509af2",
					Type: "RETAIL",
				},
			},
		},
		PayoutAt:      payoutAt,
		UserReference: userReference,
		Type:          AppTransferTypeDeposit,
		UserWarnings:  []json.RawMessage{},
		Fees:          []AppTransferFee{},
		TotalFee: AppTransferFee{
			Title:       "Fee Total",
			Description: "Total fee associated with this transaction",
			Amount:      Amount{Currency: currency.USD},
			Type:        "COINBASE",
		},
		CancellationReason: json.RawMessage("null"),
		NextStep:           json.RawMessage("null"),
	}
}

// testAppFiatTransferRequestErrors checks a deposit or withdrawal refuses every invalid request before sending it
func testAppFiatTransferRequestErrors(t *testing.T, name string, create func(context.Context, *AppFiatTransferRequest) (*AppTransfer, error)) {
	t.Helper()
	for _, tc := range []struct {
		req *AppFiatTransferRequest
		err error
	}{
		{nil, common.ErrNilPointer},
		{&AppFiatTransferRequest{Amount: 10, Currency: currency.USD, PaymentMethodID: testAppPaymentMethodID}, errAccountIDRequired},
		{&AppFiatTransferRequest{AccountID: testAppAccountID, Currency: currency.USD, PaymentMethodID: testAppPaymentMethodID}, errInvalidAmount},
		{&AppFiatTransferRequest{AccountID: testAppAccountID, Amount: -1, Currency: currency.USD, PaymentMethodID: testAppPaymentMethodID}, errInvalidAmount},
		{&AppFiatTransferRequest{AccountID: testAppAccountID, Amount: 10, PaymentMethodID: testAppPaymentMethodID}, currency.ErrCurrencyCodeEmpty},
		{&AppFiatTransferRequest{AccountID: testAppAccountID, Amount: 10, Currency: currency.USD}, errAppPaymentMethodRequired},
	} {
		_, err := create(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "%s must refuse %+v", name, tc.req)
	}
}

// testAppFiatTransferIDRequestErrors checks a method naming a deposit or withdrawal refuses every invalid request
// before sending it
func testAppFiatTransferIDRequestErrors[T any](t *testing.T, name string, call func(context.Context, *AppFiatTransferIDRequest) (T, error)) {
	t.Helper()
	for _, tc := range []struct {
		req *AppFiatTransferIDRequest
		err error
	}{
		{nil, common.ErrNilPointer},
		{&AppFiatTransferIDRequest{TransferID: testAppFiatTransferID}, errAccountIDRequired},
		{&AppFiatTransferIDRequest{AccountID: testAppAccountID}, errAppTransferIDRequired},
	} {
		_, err := call(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "%s must refuse %+v", name, tc.req)
	}
}

// testAppListFiatTransfersRequestErrors checks a deposit or withdrawal list refuses every invalid request before
// sending it
func testAppListFiatTransfersRequestErrors(t *testing.T, name string, list func(context.Context, *AppListFiatTransfersRequest) (*AppListResponse[AppFiatTransferResponse], error)) {
	t.Helper()
	for _, tc := range []struct {
		req *AppListFiatTransfersRequest
		err error
	}{
		{nil, common.ErrNilPointer},
		{&AppListFiatTransfersRequest{}, errAccountIDRequired},
		{&AppListFiatTransfersRequest{AccountID: testAppAccountID, Page: &AppPageRequest{Limit: appMaximumPageLimit + 1}}, errInvalidLimit},
	} {
		_, err := list(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "%s must refuse %+v", name, tc.req)
	}
}

func TestAppDepositFunds(t *testing.T) {
	t.Parallel()
	testAppFiatTransferRequestErrors(t, "DepositFunds", e.App().DepositFunds)

	if !mockTests {
		t.Skip("a deposit needs a payment method's ID, which Advanced Trade's List Payment Methods returns, so DepositFunds is tested against its documented response only")
	}
	resp, err := e.App().DepositFunds(t.Context(), &AppFiatTransferRequest{AccountID: testAppAccountID, Amount: 10, Currency: currency.USD, PaymentMethodID: testAppPaymentMethodID})
	require.NoError(t, err, "DepositFunds must not error")
	exp := testAppDepositTransfer("7ada05f0-4ab9-4e42-8cb9-4501e795315d", "", time.Date(2025, 4, 9, 20, 13, 48, 917581730, time.UTC))
	assert.Equal(t, exp, resp, "DepositFunds should decode every field")

	req := &AppFiatTransferRequest{AccountID: testAppAccountID, Amount: 10, Currency: currency.USD, PaymentMethodID: testAppPaymentMethodID}
	_, err = newResponseServerExchange(t, `null`).App().DepositFunds(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "DepositFunds should error for a null response")
	_, err = newUnavailableServerExchange(t).App().DepositFunds(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "DepositFunds should return the request failure")
}

func TestAppCommitDeposit(t *testing.T) {
	t.Parallel()
	testAppFiatTransferIDRequestErrors(t, "CommitDeposit", e.App().CommitDeposit)

	if !mockTests {
		t.Skip("a committed deposit cannot be undone, so CommitDeposit is tested against its documented response only")
	}
	resp, err := e.App().CommitDeposit(t.Context(), &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppUncommittedTransferID})
	require.NoError(t, err, "CommitDeposit must not error")
	exp := testAppDepositTransfer("bd4d2728-9d0c-478e-829e-8f4b4888b108", "CODE", time.Date(2025, 4, 10, 18, 7, 25, 938533583, time.UTC))
	assert.Equal(t, exp, resp, "CommitDeposit should decode every field")

	req := &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppUncommittedTransferID}
	_, err = newResponseServerExchange(t, `null`).App().CommitDeposit(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CommitDeposit should error for a null response")
	_, err = newUnavailableServerExchange(t).App().CommitDeposit(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CommitDeposit should return the request failure")
}

func TestAppListDeposits(t *testing.T) {
	t.Parallel()
	testAppListFiatTransfersRequestErrors(t, "ListDeposits", e.App().ListDeposits)

	skipLiveWithoutCredentials(t)
	resp, err := e.App().ListDeposits(t.Context(), &AppListFiatTransfersRequest{AccountID: testAppAccount(t)})
	require.NoError(t, err, "ListDeposits must not error")
	if mockTests {
		exp := &AppListResponse[AppFiatTransferResponse]{
			Data:       []AppFiatTransferResponse{testAppFiatTransfer(t, "deposit")},
			Pagination: AppPagination{Limit: 25, Order: AppOrderDescending},
		}
		assert.Equal(t, exp, resp, "ListDeposits should decode every field")
	} else {
		for i := range resp.Data {
			assert.Equalf(t, "deposit", resp.Data[i].Resource, "ListDeposits should return deposit %d as a deposit", i)
		}
	}

	_, err = newUnavailableServerExchange(t).App().ListDeposits(t.Context(), &AppListFiatTransfersRequest{AccountID: testAppAccountID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListDeposits should return the request failure")
}

func TestAppGetDeposit(t *testing.T) {
	t.Parallel()
	testAppFiatTransferIDRequestErrors(t, "GetDeposit", e.App().GetDeposit)

	skipLiveWithoutCredentials(t)
	req := &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppFiatTransferID}
	if !mockTests {
		req.AccountID = testAppAccount(t)
		deposits, err := e.App().ListDeposits(t.Context(), &AppListFiatTransfersRequest{AccountID: req.AccountID, Page: &AppPageRequest{Limit: 1}})
		require.NoError(t, err, "ListDeposits must not error")
		if len(deposits.Data) == 0 {
			t.Skip("the account has no deposits")
		}
		req.TransferID = deposits.Data[0].ID
	}
	resp, err := e.App().GetDeposit(t.Context(), req)
	require.NoError(t, err, "GetDeposit must not error")
	if mockTests {
		exp := testAppFiatTransfer(t, "deposit")
		assert.Equal(t, &exp, resp, "GetDeposit should decode every field")
	} else {
		assert.Equal(t, req.TransferID, resp.ID, "GetDeposit should return the deposit")
	}

	_, err = newResponseServerExchange(t, `null`).App().GetDeposit(t.Context(), &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppFiatTransferID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetDeposit should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetDeposit(t.Context(), &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppFiatTransferID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDeposit should return the request failure")
}

func TestAppCreateFiatTransfer(t *testing.T) {
	t.Parallel()
	testAppFiatTransferRequestErrors(t, "createFiatTransfer", func(ctx context.Context, req *AppFiatTransferRequest) (*AppTransfer, error) {
		return e.App().createFiatTransfer(ctx, "deposits", req)
	})
	for _, kind := range testAppFiatTransferKinds {
		for _, commit := range []bool{false, true} {
			ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equalf(t, http.MethodPost, r.Method, "createFiatTransfer should post the %s", kind)
				assert.Equalf(t, "/v2/accounts/"+testAppAccountID+"/"+kind, r.URL.Path, "createFiatTransfer should post to the account's %s", kind)
				body, err := io.ReadAll(r.Body)
				if assert.NoError(t, err, "Reading the request body should not error") {
					assert.JSONEqf(t, `{"amount":"10.5","currency":"USD","payment_method":"`+testAppPaymentMethodID+`","commit":`+strconv.FormatBool(commit)+`}`, string(body), "createFiatTransfer should send the commit flag as %t", commit)
				}
				_, err = w.Write([]byte(`{"transfer":{"id":"7ada05f0-4ab9-4e42-8cb9-4501e795315d","committed":` + strconv.FormatBool(commit) + `}}`))
				assert.NoError(t, err, "Writing the response should not error")
			}))
			resp, err := ex.App().createFiatTransfer(t.Context(), kind, &AppFiatTransferRequest{AccountID: testAppAccountID, Amount: 10.5, Currency: currency.USD.Lower(), PaymentMethodID: testAppPaymentMethodID, Commit: commit})
			require.NoErrorf(t, err, "createFiatTransfer must not error for %s", kind)
			assert.Equalf(t, &AppTransfer{ID: "7ada05f0-4ab9-4e42-8cb9-4501e795315d", Committed: commit}, resp, "createFiatTransfer should return the %s", kind)
		}
	}
}

func TestAppCommitFiatTransfer(t *testing.T) {
	t.Parallel()
	testAppFiatTransferIDRequestErrors(t, "commitFiatTransfer", func(ctx context.Context, req *AppFiatTransferIDRequest) (*AppTransfer, error) {
		return e.App().commitFiatTransfer(ctx, "deposits", req)
	})
	for _, kind := range testAppFiatTransferKinds {
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equalf(t, http.MethodPost, r.Method, "commitFiatTransfer should post the %s commit", kind)
			assert.Equalf(t, "/v2/accounts/"+testAppAccountID+"/"+kind+"/"+testAppUncommittedTransferID+"/commit", r.URL.Path, "commitFiatTransfer should commit the account's %s", kind)
			assert.Empty(t, r.Header.Get("Content-Type"), "commitFiatTransfer should send no body")
			_, err := w.Write([]byte(`{"transfer":{"id":"` + testAppUncommittedTransferID + `"}}`))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		resp, err := ex.App().commitFiatTransfer(t.Context(), kind, &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppUncommittedTransferID})
		require.NoErrorf(t, err, "commitFiatTransfer must not error for %s", kind)
		assert.Equalf(t, &AppTransfer{ID: testAppUncommittedTransferID}, resp, "commitFiatTransfer should return the committed %s", kind)
	}
}

func TestAppListFiatTransfers(t *testing.T) {
	t.Parallel()
	testAppListFiatTransfersRequestErrors(t, "listFiatTransfers", func(ctx context.Context, req *AppListFiatTransfersRequest) (*AppListResponse[AppFiatTransferResponse], error) {
		return e.App().listFiatTransfers(ctx, "deposits", req)
	})
	for _, kind := range testAppFiatTransferKinds {
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equalf(t, http.MethodGet, r.Method, "listFiatTransfers should get the %s", kind)
			assert.Equalf(t, "/v2/accounts/"+testAppAccountID+"/"+kind, r.URL.Path, "listFiatTransfers should list the account's %s", kind)
			assert.Equal(t, "limit=1&order=asc&starting_after="+testAppFiatTransferID, r.URL.RawQuery, "listFiatTransfers should send the page")
			_, err := w.Write([]byte(`{"pagination":{"limit":1,"order":"asc","next_uri":"/v2/accounts/x"},"data":[{"id":"` + testAppFiatTransferID + `"}]}`))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		resp, err := ex.App().listFiatTransfers(t.Context(), kind, &AppListFiatTransfersRequest{AccountID: testAppAccountID, Page: &AppPageRequest{Limit: 1, Order: AppOrderAscending, StartingAfter: testAppFiatTransferID}})
		require.NoErrorf(t, err, "listFiatTransfers must not error for %s", kind)
		exp := &AppListResponse[AppFiatTransferResponse]{
			Data:       []AppFiatTransferResponse{{ID: testAppFiatTransferID}},
			Pagination: AppPagination{Limit: 1, Order: AppOrderAscending, NextURI: "/v2/accounts/x"},
		}
		assert.Equalf(t, exp, resp, "listFiatTransfers should return the page of %s", kind)
	}
}

func TestAppGetFiatTransfer(t *testing.T) {
	t.Parallel()
	testAppFiatTransferIDRequestErrors(t, "getFiatTransfer", func(ctx context.Context, req *AppFiatTransferIDRequest) (*AppFiatTransferResponse, error) {
		return e.App().getFiatTransfer(ctx, "deposits", req)
	})
	for _, kind := range testAppFiatTransferKinds {
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equalf(t, http.MethodGet, r.Method, "getFiatTransfer should get the %s", kind)
			assert.Equalf(t, "/v2/accounts/"+testAppAccountID+"/"+kind+"/"+testAppFiatTransferID, r.URL.Path, "getFiatTransfer should get one of the account's %s", kind)
			_, err := w.Write([]byte(`{"data":{"id":"` + testAppFiatTransferID + `"}}`))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		resp, err := ex.App().getFiatTransfer(t.Context(), kind, &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppFiatTransferID})
		require.NoErrorf(t, err, "getFiatTransfer must not error for %s", kind)
		assert.Equalf(t, &AppFiatTransferResponse{ID: testAppFiatTransferID}, resp, "getFiatTransfer should return the %s", kind)
	}
	_, err := newResponseServerExchange(t, `{"data":null}`).App().getFiatTransfer(t.Context(), "deposits", &AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppFiatTransferID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "getFiatTransfer should error for a null transfer")
}

func TestAppSendTransferRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "sendTransferRequest should post the request")
		assert.Equal(t, "/v2/accounts/"+testAppAccountID+"/deposits", r.URL.Path, "sendTransferRequest should post to the path under /v2/")
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			assert.JSONEq(t, `{"commit":true}`, string(body), "sendTransferRequest should send the body")
		}
		_, err = w.Write([]byte(`{"transfer":{"id":"7ada05f0-4ab9-4e42-8cb9-4501e795315d"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	resp, err := ex.App().sendTransferRequest(t.Context(), "accounts/"+testAppAccountID+"/deposits", map[string]bool{"commit": true})
	require.NoError(t, err, "sendTransferRequest must not error")
	assert.Equal(t, &AppTransfer{ID: "7ada05f0-4ab9-4e42-8cb9-4501e795315d"}, resp, "sendTransferRequest should return the transfer outside the data envelope")

	for _, body := range []string{`null`, `{}`, `{"data":{"id":"7ada05f0-4ab9-4e42-8cb9-4501e795315d"}}`} {
		_, err = newResponseServerExchange(t, body).App().sendTransferRequest(t.Context(), "accounts/"+testAppAccountID+"/deposits", nil)
		assert.ErrorIsf(t, err, common.ErrNoResponse, "sendTransferRequest should error for a response without a transfer: %s", body)
	}
	_, err = newUnavailableServerExchange(t).App().sendTransferRequest(t.Context(), "accounts/"+testAppAccountID+"/deposits", nil)
	assert.ErrorIs(t, err, request.ErrBadStatus, "sendTransferRequest should return the request failure")
}

func TestAppFiatTransferIDRequestValidate(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, (*AppFiatTransferIDRequest)(nil).validate(), common.ErrNilPointer, "validate should reject a nil request")
	assert.ErrorIs(t, (&AppFiatTransferIDRequest{TransferID: testAppFiatTransferID}).validate(), errAccountIDRequired, "validate should require an account ID")
	assert.ErrorIs(t, (&AppFiatTransferIDRequest{AccountID: testAppAccountID}).validate(), errAppTransferIDRequired, "validate should require a transfer ID")
	assert.NoError(t, (&AppFiatTransferIDRequest{AccountID: testAppAccountID, TransferID: testAppFiatTransferID}).validate(), "validate should accept a named transfer")
}
