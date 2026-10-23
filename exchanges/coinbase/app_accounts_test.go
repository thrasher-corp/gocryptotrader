package coinbase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// testAppAccountID is the account the docs' examples act on, which the mock fixtures answer for
const testAppAccountID = "2bbf394c-193b-5b2a-9155-3b4732659ede"

// appTestTime parses an RFC 3339 time as decoding does, so an expected time with an offset has the location decoding
// gives it on any host
func appTestTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	require.NoErrorf(t, err, "time.Parse must not error for %s", s)
	return parsed
}

// testAppAccount returns the account a test of an account's resources acts on: the docs' account the mock fixtures
// answer for, or live the primary BTC account
func testAppAccount(t *testing.T) string {
	t.Helper()
	if mockTests {
		return testAppAccountID
	}
	account, err := e.App().GetAccount(t.Context(), &AppGetAccountRequest{Currency: currency.BTC})
	require.NoError(t, err, "GetAccount must not error")
	return account.ID
}

func TestAppGetCurrentUser(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.App().GetCurrentUser(t.Context())
	require.NoError(t, err, "GetCurrentUser must not error")
	if mockTests {
		exp := &AppUserResponse{
			ID:             "9bd290f2-beed-52e5-84b7-2c36d961a161",
			Name:           "First Last",
			AvatarURL:      "https://static-assets.coinbase.com/profile/avatar.jpg",
			Resource:       "user",
			ResourcePath:   "/v2/user",
			TimeZone:       "Pacific Time (US & Canada)",
			NativeCurrency: currency.USD,
			BitcoinUnit:    "BTC",
			Country: AppCountry{
				Code: "US",
				Name: "United States of America",
			},
			CreatedAt: time.Date(2024, 8, 22, 2, 55, 49, 0, time.UTC),
			Email:     "first.last@example.com",
		}
		assert.Equal(t, exp, resp, "GetCurrentUser should decode every field")
	} else {
		assert.NotEmpty(t, resp.ID, "GetCurrentUser should return the user's ID")
	}

	_, err = newResponseServerExchange(t, `null`).App().GetCurrentUser(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetCurrentUser should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetCurrentUser(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCurrentUser should return the request failure")
}

func TestAppGetPersonalDetails(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.App().GetPersonalDetails(t.Context())
	require.NoError(t, err, "GetPersonalDetails must not error")
	if mockTests {
		exp := &AppPersonalDetailsResponse{
			FirstName: "First",
			LastName:  "Last",
			PrimaryAddress: &AppPersonalAddress{
				Line1:      "123 Main Street",
				City:       "San Francisco",
				State:      "CA",
				PostalCode: "94105",
				Country: AppCountry{
					Code: "US",
					Name: "United States of America",
				},
			},
		}
		assert.Equal(t, exp, resp, "GetPersonalDetails should decode every field")
	} else {
		assert.NotEmpty(t, resp.LastName, "GetPersonalDetails should return the user's last name")
	}

	_, err = newResponseServerExchange(t, `null`).App().GetPersonalDetails(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPersonalDetails should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetPersonalDetails(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPersonalDetails should return the request failure")
}

func TestAppListAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.App().ListAccounts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListAccounts must reject a nil request")
	_, err = e.App().ListAccounts(t.Context(), &AppListAccountsRequest{Page: &AppPageRequest{Limit: appMaximumPageLimit + 1}})
	require.ErrorIs(t, err, errInvalidLimit, "ListAccounts must refuse an invalid page")

	skipLiveWithoutCredentials(t)
	resp, err := e.App().ListAccounts(t.Context(), &AppListAccountsRequest{IncludeAvailableToSend: true})
	require.NoError(t, err, "ListAccounts must not error")
	if mockTests {
		bitcoin := AppCurrencyResponse{
			AssetID:      "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
			Code:         currency.BTC,
			Name:         "Bitcoin",
			Colour:       "#F7931A",
			SortIndex:    100,
			Exponent:     8,
			Type:         "crypto",
			AddressRegex: "^([13][a-km-zA-HJ-NP-Z1-9]{25,34})|^(bc1[qzry9x8gf2tvdw0s3jn54khce6mua7l]([qpzry9x8gf2tvdw0s3jn54khce6mua7l]{38}|[qpzry9x8gf2tvdw0s3jn54khce6mua7l]{58}))$",
			Slug:         "bitcoin",
		}
		exp := &AppListResponse[AppAccountResponse]{
			Data: []AppAccountResponse{
				{
					ID:       "58542935-67b5-56e1-a3f9-42686e07fa40",
					Name:     "My Vault",
					Type:     AppAccountTypeVault,
					Currency: bitcoin,
					Balance: AppMoney{
						Amount:   4,
						Currency: currency.BTC,
					},
					CreatedAt:    time.Date(2024, 1, 31, 20, 49, 2, 0, time.UTC),
					UpdatedAt:    time.Date(2024, 1, 31, 20, 49, 2, 0, time.UTC),
					Resource:     "account",
					ResourcePath: "/v2/accounts/58542935-67b5-56e1-a3f9-42686e07fa40",
					Ready:        true,
				},
				{
					ID:       "2bbf394c-193b-5b2a-9155-3b4732659ede",
					Name:     "My Wallet",
					Primary:  true,
					Type:     AppAccountTypeWallet,
					Currency: bitcoin,
					Balance: AppMoney{
						Amount:   39.59,
						Currency: currency.BTC,
					},
					CreatedAt:    time.Date(2024, 1, 31, 20, 49, 2, 0, time.UTC),
					UpdatedAt:    time.Date(2024, 1, 31, 20, 49, 2, 0, time.UTC),
					Resource:     "account",
					ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede",
				},
				{
					ID:      "8d5f2c4e-7a1b-5c3d-9e6f-0a1b2c3d4e5f",
					Name:    "USDC Wallet",
					Primary: true,
					Type:    AppAccountTypeWallet,
					Currency: AppCurrencyResponse{
						AssetID:  "2b92315d-eab7-5bef-84fa-089a131333f5",
						Code:     currency.USDC,
						Name:     "USDC",
						Exponent: 6,
						Type:     "crypto",
					},
					Balance: AppMoney{
						Amount:   250,
						Currency: currency.USDC,
					},
					AvailableToSend: &AppMoney{
						Amount:   200,
						Currency: currency.USDC,
					},
					CreatedAt:    time.Date(2024, 1, 31, 20, 49, 2, 0, time.UTC),
					UpdatedAt:    time.Date(2024, 1, 31, 20, 49, 2, 0, time.UTC),
					Resource:     "account",
					ResourcePath: "/v2/accounts/8d5f2c4e-7a1b-5c3d-9e6f-0a1b2c3d4e5f",
				},
			},
			Pagination: AppPagination{
				Limit: 25,
				Order: AppOrderDescending,
			},
		}
		assert.Equal(t, exp, resp, "ListAccounts should decode every field, leaving AvailableToSend nil where it is not sent")
	} else {
		assert.NotEmpty(t, resp.Data, "ListAccounts should return accounts")
	}

	_, err = newUnavailableServerExchange(t).App().ListAccounts(t.Context(), &AppListAccountsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListAccounts should return the request failure")
}

func TestAppGetAccount(t *testing.T) {
	t.Parallel()
	_, err := e.App().GetAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccount must reject a nil request")
	_, err = e.App().GetAccount(t.Context(), &AppGetAccountRequest{})
	require.ErrorIs(t, err, errAppAccountOrCurrencyRequired, "GetAccount must require an account ID or a currency")
	_, err = e.App().GetAccount(t.Context(), &AppGetAccountRequest{AccountID: testAppAccountID, Currency: currency.BTC})
	require.ErrorIs(t, err, errAppAccountAndCurrencySet, "GetAccount must refuse both an account ID and a currency")

	skipLiveWithoutCredentials(t)
	primaryCurrency := currency.USDC
	if !mockTests {
		primaryCurrency = currency.BTC
	}
	// The currency is sent upper case, as Coinbase names currencies
	primary, err := e.App().GetAccount(t.Context(), &AppGetAccountRequest{Currency: primaryCurrency.Lower(), IncludeAvailableToSend: true})
	require.NoError(t, err, "GetAccount must not error for a currency's primary account")
	accountID := testAppAccountID
	if !mockTests {
		accountID = primary.ID
	}
	resp, err := e.App().GetAccount(t.Context(), &AppGetAccountRequest{AccountID: accountID})
	require.NoError(t, err, "GetAccount must not error for an account ID")
	if mockTests {
		exp := &AppAccountResponse{
			ID:      "8d5f2c4e-7a1b-5c3d-9e6f-0a1b2c3d4e5f",
			Name:    "USDC Wallet",
			Primary: true,
			Type:    AppAccountTypeWallet,
			Currency: AppCurrencyResponse{
				AssetID:  "2b92315d-eab7-5bef-84fa-089a131333f5",
				Code:     currency.USDC,
				Name:     "USDC",
				Exponent: 6,
				Type:     "crypto",
			},
			Balance: AppMoney{
				Amount:   250,
				Currency: currency.USDC,
			},
			AvailableToSend: &AppMoney{
				Amount:   200,
				Currency: currency.USDC,
			},
			CreatedAt:    time.Date(2024, 1, 31, 20, 49, 2, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 1, 31, 20, 49, 2, 0, time.UTC),
			Resource:     "account",
			ResourcePath: "/v2/accounts/8d5f2c4e-7a1b-5c3d-9e6f-0a1b2c3d4e5f",
		}
		assert.Equal(t, exp, primary, "GetAccount should decode a currency's primary account")
		exp = &AppAccountResponse{
			ID:      "2bbf394c-193b-5b2a-9155-3b4732659ede",
			Name:    "My Wallet",
			Primary: true,
			Type:    AppAccountTypeWallet,
			Currency: AppCurrencyResponse{
				AssetID:      "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
				Code:         currency.BTC,
				Name:         "Bitcoin",
				Colour:       "#F7931A",
				SortIndex:    100,
				Exponent:     8,
				Type:         "crypto",
				AddressRegex: "^([13][a-km-zA-HJ-NP-Z1-9]{25,34})|^(bc1[qzry9x8gf2tvdw0s3jn54khce6mua7l]([qpzry9x8gf2tvdw0s3jn54khce6mua7l]{38}|[qpzry9x8gf2tvdw0s3jn54khce6mua7l]{58}))$",
				Slug:         "bitcoin",
			},
			Balance: AppMoney{
				Amount:   39.59,
				Currency: currency.BTC,
			},
			CreatedAt:    time.Date(2024, 1, 31, 20, 49, 2, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 1, 31, 20, 49, 2, 0, time.UTC),
			Resource:     "account",
			ResourcePath: "/v2/accounts/2bbf394c-193b-5b2a-9155-3b4732659ede",
		}
		assert.Equal(t, exp, resp, "GetAccount should decode every field")
	} else {
		assert.True(t, primary.Primary, "GetAccount should return the currency's primary account")
		assert.Equal(t, primary.ID, resp.ID, "GetAccount should return the account the ID names")
	}

	_, err = newResponseServerExchange(t, `null`).App().GetAccount(t.Context(), &AppGetAccountRequest{AccountID: testAppAccountID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccount should error for a null response")
	_, err = newUnavailableServerExchange(t).App().GetAccount(t.Context(), &AppGetAccountRequest{AccountID: testAppAccountID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccount should return the request failure")
}
