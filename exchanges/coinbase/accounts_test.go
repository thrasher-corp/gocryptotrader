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

func TestGetAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccount(t.Context(), "")
	require.ErrorIs(t, err, errAccountIDRequired, "GetAccount must require an account UUID")

	skipLiveWithoutCredentials(t)
	// The sandbox's recorded account
	accountUUID := "66f975a6-bb2e-44be-82e9-cd8669e404b0"
	if !mockTests {
		accounts, err := e.ListAccounts(t.Context(), &ListAccountsRequest{Limit: 1})
		require.NoError(t, err, "ListAccounts must not error")
		require.NotEmpty(t, accounts.Accounts, "ListAccounts must return an account")
		accountUUID = accounts.Accounts[0].UUID
	}
	resp, err := e.GetAccount(t.Context(), accountUUID)
	require.NoError(t, err, "GetAccount must not error")
	if mockTests {
		exp := &Account{
			UUID:              "66f975a6-bb2e-44be-82e9-cd8669e404b0",
			Name:              "USDC Wallet",
			Currency:          currency.USDC,
			AvailableBalance:  Amount{Value: 100, Currency: currency.USDC},
			Default:           true,
			Active:            true,
			CreatedAt:         time.Date(2023, 12, 14, 21, 40, 32, 181000000, time.UTC),
			UpdatedAt:         time.Date(2023, 12, 19, 18, 21, 42, 850000000, time.UTC),
			Type:              AccountTypeCrypto,
			Ready:             true,
			Hold:              Amount{Currency: currency.USDC},
			RetailPortfolioID: "1fb3c56a-489f-418c-8b5c-ce04268be5a6",
			Platform:          AccountPlatformConsumer,
		}
		assert.Equal(t, exp, resp, "GetAccount should decode every field")
	} else {
		assert.Equal(t, accountUUID, resp.UUID, "GetAccount should return the account")
	}

	_, err = newJWTCheckingServerExchange(t, `null`).GetAccount(t.Context(), accountUUID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccount should error for a null response")
	_, err = newJWTCheckingServerExchange(t, `{"account":null}`).GetAccount(t.Context(), accountUUID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccount should error for a null account")
	_, err = newUnavailableServerExchange(t).GetAccount(t.Context(), accountUUID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccount should return the request failure")
}

func TestListAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.ListAccounts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListAccounts must reject a nil request")

	skipLiveWithoutCredentials(t)
	resp, err := e.ListAccounts(t.Context(), &ListAccountsRequest{Limit: 5})
	require.NoError(t, err, "ListAccounts must not error")
	if mockTests {
		exp := &AccountsResponse{
			Accounts: []Account{
				{
					UUID:              "66f975a6-bb2e-44be-82e9-cd8669e404b0",
					Name:              "USDC Wallet",
					Currency:          currency.USDC,
					AvailableBalance:  Amount{Value: 100, Currency: currency.USDC},
					Default:           true,
					Active:            true,
					CreatedAt:         time.Date(2023, 12, 14, 21, 40, 32, 181000000, time.UTC),
					UpdatedAt:         time.Date(2023, 12, 19, 18, 21, 42, 850000000, time.UTC),
					Type:              AccountTypeCrypto,
					Ready:             true,
					Hold:              Amount{Currency: currency.USDC},
					RetailPortfolioID: "1fb3c56a-489f-418c-8b5c-ce04268be5a6",
					Platform:          AccountPlatformConsumer,
				},
				{
					UUID:              "646ec0d2-d874-4f2a-9482-32176f297211",
					Name:              "ETH Wallet",
					Currency:          currency.ETH,
					AvailableBalance:  Amount{Value: 1, Currency: currency.ETH},
					Default:           true,
					Active:            true,
					CreatedAt:         time.Date(2023, 9, 26, 22, 21, 19, 524000000, time.UTC),
					UpdatedAt:         time.Date(2023, 12, 19, 18, 21, 42, 806000000, time.UTC),
					Type:              AccountTypeCrypto,
					Ready:             true,
					Hold:              Amount{Currency: currency.ETH},
					RetailPortfolioID: "1fb3c56a-489f-418c-8b5c-ce04268be5a6",
					Platform:          AccountPlatformCFMConsumer,
				},
				{
					UUID:              "81b5dca0-5775-4a21-b872-fc850046c99b",
					Name:              "Cash (USD)",
					Currency:          currency.USD,
					AvailableBalance:  Amount{Value: 100, Currency: currency.USD},
					Active:            true,
					CreatedAt:         time.Date(2023, 8, 11, 3, 31, 8, 743000000, time.UTC),
					UpdatedAt:         time.Date(2023, 8, 11, 3, 31, 8, 990000000, time.UTC),
					Type:              AccountTypeFiat,
					Ready:             true,
					Hold:              Amount{Currency: currency.USD},
					RetailPortfolioID: "1fb3c56a-489f-418c-8b5c-ce04268be5a6",
					Platform:          AccountPlatformConsumer,
				},
				{
					UUID:              "31e698c7-edd1-43b7-a0db-5c87fc15ea52",
					Name:              "BTC Wallet",
					Currency:          currency.BTC,
					AvailableBalance:  Amount{Value: 0.5, Currency: currency.BTC},
					Default:           true,
					Active:            true,
					CreatedAt:         time.Date(2023, 8, 9, 14, 17, 42, 913000000, time.UTC),
					UpdatedAt:         time.Date(2023, 8, 9, 14, 17, 42, 913000000, time.UTC),
					Type:              AccountTypeCrypto,
					Ready:             true,
					Hold:              Amount{Currency: currency.BTC},
					RetailPortfolioID: "1fb3c56a-489f-418c-8b5c-ce04268be5a6",
					Platform:          AccountPlatformConsumer,
				},
			},
			Size: 4,
		}
		assert.Equal(t, exp, resp, "ListAccounts should decode every field")
	} else {
		assert.NotEmpty(t, resp.Accounts, "ListAccounts should return accounts")
		assert.LessOrEqual(t, len(resp.Accounts), 5, "ListAccounts should return at most the limited accounts")
	}

	_, err = newJWTCheckingServerExchange(t, `null`).ListAccounts(t.Context(), &ListAccountsRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "ListAccounts should error for a null response")
	_, err = newUnavailableServerExchange(t).ListAccounts(t.Context(), &ListAccountsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListAccounts should return the request failure")
}

func TestListAccountsRequestParams(t *testing.T) {
	t.Parallel()
	req := &ListAccountsRequest{Limit: 250, Cursor: "789100", RetailPortfolioID: "b87a2d3f-8a1e-49b3-a4ea-402d8c389aca"}
	assert.Equal(t, "cursor=789100&limit=250&retail_portfolio_id=b87a2d3f-8a1e-49b3-a4ea-402d8c389aca", req.params().Encode(), "params should send every field")
	assert.Empty(t, (&ListAccountsRequest{}).params(), "params should send nothing for an empty request")
}
