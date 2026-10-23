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

// testAppAddressID is the address the docs' Show Address example returns
const testAppAddressID = "dd3183eb-af1d-5f5d-a90d-cbff946435ff"

// testAppAddress returns the address a test of an address's resources acts on: the docs' address the mock fixtures
// answer for, or live the first address of the primary BTC account, skipping the test when it has none
func testAppAddress(t *testing.T) (accountID, addressID string) {
	t.Helper()
	if mockTests {
		return testAppAccountID, testAppAddressID
	}
	accountID = testAppAccount(t)
	addresses, err := e.App().ListAddresses(t.Context(), &AppListAddressesRequest{AccountID: accountID, Page: &AppPageRequest{Limit: 1}})
	require.NoError(t, err, "ListAddresses must not error")
	if len(addresses.Data) == 0 {
		t.Skip("the account has no addresses")
	}
	return accountID, addresses.Data[0].ID
}

func TestAppCreateAddress(t *testing.T) {
	t.Parallel()
	_, err := e.App().CreateAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateAddress must reject a nil request")
	_, err = e.App().CreateAddress(t.Context(), &AppCreateAddressRequest{})
	require.ErrorIs(t, err, errAccountIDRequired, "CreateAddress must require an account ID")

	skipLiveUnlessCanManipulate(t)
	req := &AppCreateAddressRequest{AccountID: testAppAccountID, Name: "New receive address", Network: "ethereum"}
	if !mockTests {
		req = &AppCreateAddressRequest{AccountID: testAppAccount(t), Name: "GCT test"}
	}
	resp, err := e.App().CreateAddress(t.Context(), req)
	require.NoError(t, err, "CreateAddress must not error")
	if mockTests {
		exp := &AppAddressResponse{
			ID:           "dd3183eb-af1d-5f5d-a90d-cbff946435ff",
			Address:      "0xc0ffee254729296a45a3885639AC7E10F9d54979",
			Name:         "New receive address",
			Network:      "ethereum",
			CreatedAt:    time.Date(2015, 1, 31, 20, 49, 2, 0, time.UTC),
			UpdatedAt:    appTestTime(t, "2015-03-31T17:25:29-07:00"),
			Resource:     "address",
			ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/addresses/dd3183eb-af1d-5f5d-a90d-cbff946435ff",
		}
		assert.Equal(t, exp, resp, "CreateAddress should decode every field")
	} else {
		assert.NotEmpty(t, resp.Address, "CreateAddress should return the address")
	}

	// Every argument is optional, so an address on the default network is created with an empty object
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "CreateAddress should post the request")
		assert.Equal(t, "/v2/accounts/"+testAppAccountID+"/addresses", r.URL.Path, "CreateAddress should post to the account's addresses")
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			assert.JSONEq(t, `{}`, string(body), "CreateAddress should leave unset arguments out")
		}
		_, err = w.Write([]byte(`{"data":{"id":"dd3183eb-af1d-5f5d-a90d-cbff946435ff"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	resp, err = ex.App().CreateAddress(t.Context(), &AppCreateAddressRequest{AccountID: testAppAccountID})
	require.NoError(t, err, "CreateAddress must not error without a name or network")
	assert.Equal(t, &AppAddressResponse{ID: testAppAddressID}, resp, "CreateAddress should return the address")

	_, err = newResponseServerExchange(t, `null`).App().CreateAddress(t.Context(), &AppCreateAddressRequest{AccountID: testAppAccountID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateAddress should error for a null response")
	_, err = newUnavailableServerExchange(t).App().CreateAddress(t.Context(), &AppCreateAddressRequest{AccountID: testAppAccountID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateAddress should return the request failure")
}

func TestAppListAddresses(t *testing.T) {
	t.Parallel()
	_, err := e.App().ListAddresses(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListAddresses must reject a nil request")
	_, err = e.App().ListAddresses(t.Context(), &AppListAddressesRequest{})
	require.ErrorIs(t, err, errAccountIDRequired, "ListAddresses must require an account ID")
	_, err = e.App().ListAddresses(t.Context(), &AppListAddressesRequest{AccountID: testAppAccountID, Page: &AppPageRequest{StartingAfter: "a", EndingBefore: "b"}})
	require.ErrorIs(t, err, errAppPageCursors, "ListAddresses must refuse an invalid page")

	skipLiveWithoutCredentials(t)
	resp, err := e.App().ListAddresses(t.Context(), &AppListAddressesRequest{AccountID: testAppAccount(t)})
	require.NoError(t, err, "ListAddresses must not error")
	if mockTests {
		exp := &AppListResponse[AppAddressResponse]{
			Data: []AppAddressResponse{
				{
					ID:           "dd3183eb-af1d-5f5d-a90d-cbff946435ff",
					Address:      "mswUGcPHp1YnkLCgF1TtoryqSc5E9Q8xFa",
					Network:      "bitcoin",
					CreatedAt:    time.Date(2015, 1, 31, 20, 49, 2, 0, time.UTC),
					UpdatedAt:    appTestTime(t, "2015-03-31T17:25:29-07:00"),
					Resource:     "address",
					ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/addresses/dd3183eb-af1d-5f5d-a90d-cbff946435ff",
				},
				{
					ID:           "ac5c5f15-0b1d-54f5-8912-fecbf66c2a64",
					Address:      "mgSvu1z1amUFAPkB4cUg8ujaDxKAfZBt5Q",
					Network:      "bitcoin",
					CreatedAt:    appTestTime(t, "2015-03-31T17:23:52-07:00"),
					UpdatedAt:    time.Date(2015, 1, 31, 20, 49, 2, 0, time.UTC),
					Resource:     "address",
					ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/addresses/ac5c5f15-0b1d-54f5-8912-fecbf66c2a64",
				},
			},
			Pagination: AppPagination{
				Limit: 25,
				Order: AppOrderDescending,
			},
		}
		assert.Equal(t, exp, resp, "ListAddresses should decode every field")
	} else {
		for i := range resp.Data {
			assert.NotEmptyf(t, resp.Data[i].Address, "ListAddresses should return address %d's address", i)
		}
	}

	_, err = newUnavailableServerExchange(t).App().ListAddresses(t.Context(), &AppListAddressesRequest{AccountID: testAppAccountID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListAddresses should return the request failure")
}

func TestAppGetAddress(t *testing.T) {
	t.Parallel()
	_, err := e.App().GetAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAddress must reject a nil request")
	_, err = e.App().GetAddress(t.Context(), &AppGetAddressRequest{AddressID: testAppAddressID})
	require.ErrorIs(t, err, errAccountIDRequired, "GetAddress must require an account ID")
	_, err = e.App().GetAddress(t.Context(), &AppGetAddressRequest{AccountID: testAppAccountID})
	require.ErrorIs(t, err, errAppAddressIDRequired, "GetAddress must require an address ID")

	skipLiveWithoutCredentials(t)
	accountID, addressID := testAppAddress(t)
	resp, err := e.App().GetAddress(t.Context(), &AppGetAddressRequest{AccountID: accountID, AddressID: addressID})
	require.NoError(t, err, "GetAddress must not error")
	if mockTests {
		exp := &AppAddressResponse{
			ID:           "dd3183eb-af1d-5f5d-a90d-cbff946435ff",
			Address:      "mswUGcPHp1YnkLCgF1TtoryqSc5E9Q8xFa",
			Network:      "bitcoin",
			CreatedAt:    time.Date(2015, 1, 31, 20, 49, 2, 0, time.UTC),
			UpdatedAt:    appTestTime(t, "2015-03-31T17:25:29-07:00"),
			Resource:     "address",
			ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/addresses/dd3183eb-af1d-5f5d-a90d-cbff946435ff/",
		}
		assert.Equal(t, exp, resp, "GetAddress should decode every field")
	} else {
		assert.Equal(t, addressID, resp.ID, "GetAddress should return the address")
	}

	_, err = newResponseServerExchange(t, `null`).App().GetAddress(t.Context(), &AppGetAddressRequest{AccountID: testAppAccountID, AddressID: testAppAddressID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAddress should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetAddress(t.Context(), &AppGetAddressRequest{AccountID: testAppAccountID, AddressID: testAppAddressID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAddress should return the request failure")
}

func TestAppListAddressTransactions(t *testing.T) {
	t.Parallel()
	_, err := e.App().ListAddressTransactions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListAddressTransactions must reject a nil request")
	_, err = e.App().ListAddressTransactions(t.Context(), &AppListAddressTransactionsRequest{AddressID: testAppAddressID})
	require.ErrorIs(t, err, errAccountIDRequired, "ListAddressTransactions must require an account ID")
	_, err = e.App().ListAddressTransactions(t.Context(), &AppListAddressTransactionsRequest{AccountID: testAppAccountID})
	require.ErrorIs(t, err, errAppAddressIDRequired, "ListAddressTransactions must require an address ID")
	_, err = e.App().ListAddressTransactions(t.Context(), &AppListAddressTransactionsRequest{AccountID: testAppAccountID, AddressID: testAppAddressID, Page: &AppPageRequest{Limit: appMaximumPageLimit + 1}})
	require.ErrorIs(t, err, errInvalidLimit, "ListAddressTransactions must refuse an invalid page")

	skipLiveWithoutCredentials(t)
	accountID, addressID := testAppAddress(t)
	resp, err := e.App().ListAddressTransactions(t.Context(), &AppListAddressTransactionsRequest{AccountID: accountID, AddressID: addressID})
	require.NoError(t, err, "ListAddressTransactions must not error")
	if mockTests {
		exp := &AppListResponse[AppTransactionResponse]{
			Data: []AppTransactionResponse{
				{
					ID:     "57ffb4ae-0c59-5430-bcd3-3f98f797a66c",
					Type:   AppTransactionTypeSend,
					Status: AppTransactionStatusCompleted,
					Amount: AppMoney{
						Amount:   0.001,
						Currency: currency.BTC,
					},
					NativeAmount: AppMoney{
						Amount:   0.01,
						Currency: currency.USD,
					},
					CreatedAt:    appTestTime(t, "2015-03-11T13:13:35-07:00"),
					UpdatedAt:    appTestTime(t, "2015-03-26T15:55:43-07:00"),
					Resource:     "transaction",
					ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede/transactions/57ffb4ae-0c59-5430-bcd3-3f98f797a66c",
					Network: &AppTransactionNetwork{
						Status: AppTransactionNetworkStatusOffBlockchain,
						Name:   "bitcoin",
					},
					From: &AppTransactionParty{
						ID:       "a6b4c2df-a62c-5d68-822a-dd4e2102e703",
						Resource: "user",
					},
				},
			},
			Pagination: AppPagination{
				Limit: 25,
				Order: AppOrderDescending,
			},
		}
		assert.Equal(t, exp, resp, "ListAddressTransactions should decode every field")
	} else {
		assert.NotNil(t, resp.Data, "ListAddressTransactions should return the address's transactions")
	}

	_, err = newUnavailableServerExchange(t).App().ListAddressTransactions(t.Context(), &AppListAddressTransactionsRequest{AccountID: testAppAccountID, AddressID: testAppAddressID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListAddressTransactions should return the request failure")
}
