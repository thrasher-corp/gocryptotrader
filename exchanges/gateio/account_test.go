package gateio

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetAccountDetails(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountDetails(t.Context())
	require.NoError(t, err, "GetAccountDetails must not error")
	if mockTests {
		exp := &AccountDetailsResponse{
			IPWhitelist:     []string{"127.0.0.1"},
			CurrencyPairs:   []currency.Pair{currency.NewPairWithDelimiter("USDT", "BTC", currency.UnderscoreDelimiter)},
			UserID:          1667201533,
			VIPTier:         2,
			Key:             AccountKeyDetails{Mode: 1},
			CopyTradingRole: 1,
		}
		assert.Equal(t, exp, result, "GetAccountDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountDetails should return details")
}

func TestGetAccountMainKeys(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountMainKeys(t.Context())
	require.NoError(t, err, "GetAccountMainKeys must not error")
	if mockTests {
		exp := []*AccountMainKeyResponse{{
			State:         1,
			Mode:          1,
			Name:          "test1",
			CurrencyPairs: []currency.Pair{currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter), currency.NewPairWithDelimiter("ETH", "USD", currency.UnderscoreDelimiter)},
			UserID:        1,
			IPWhitelist:   []string{},
			Permissions:   []*SubAccountKeyPerm{{Name: "account"}, {Name: "spot", ReadOnly: true}},
			MaskedKey:     "c5dcfbf1f3a7*****",
			CreatedAt:     types.Time(time.Unix(1753030929, 0)),
			UpdateAt:      types.Time(time.Unix(1756300567, 0)),
			LastAccess:    types.Time(time.Unix(1753030930, 0)),
		}}
		assert.Equal(t, exp, result, "GetAccountMainKeys should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountMainKeys should return keys")
}

func TestGetUserTransactionRateLimitInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserTransactionRateLimitInfo(t.Context())
	require.NoError(t, err, "GetUserTransactionRateLimitInfo must not error")
	if mockTests {
		exp := []*UserTransactionRateLimitInfoResponse{
			{Type: "spot", Tier: 1, Ratio: 0.5, MainRatio: 0.25, UpdatedAt: types.Time(time.Unix(1728230400, 0))},
			{Type: "futures", Tier: 2, Ratio: 0, MainRatio: 0, UpdatedAt: types.Time(time.Unix(1728230400, 0))},
		}
		assert.Equal(t, exp, result, "GetUserTransactionRateLimitInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserTransactionRateLimitInfo should return rate limits")
}

func TestCreateSelfTradePreventionUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSelfTradePreventionUserGroup(t.Context(), "")
	require.ErrorIs(t, err, errSTPGroupNameRequired, "CreateSelfTradePreventionUserGroup must reject an empty name")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSelfTradePreventionUserGroup(t.Context(), "stp_groups")
	require.NoError(t, err, "CreateSelfTradePreventionUserGroup must not error")
	if mockTests {
		exp := &STPUserGroupResponse{ID: 123435, Name: "stp_groups", CreatorID: 10000, CreateTime: types.Time(time.Unix(1548000000, 0))}
		assert.Equal(t, exp, result, "CreateSelfTradePreventionUserGroup should decode every field")
	}
}

func TestGetUserSelfTradePreventionGroups(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserSelfTradePreventionGroups(t.Context(), "thrasher")
	require.NoError(t, err, "GetUserSelfTradePreventionGroups must not error")
	if mockTests {
		exp := []*STPUserGroupResponse{{ID: 123435, Name: "thrasher", CreatorID: 10000, CreateTime: types.Time(time.Unix(1548000000, 0))}}
		assert.Equal(t, exp, result, "GetUserSelfTradePreventionGroups should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserSelfTradePreventionGroups should return groups")
}

func TestGetUsersInSTPUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersInSTPUserGroup(t.Context(), 0)
	require.ErrorIs(t, err, errSTPGroupIDRequired, "GetUsersInSTPUserGroup must reject an empty group ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUsersInSTPUserGroup(t.Context(), 1)
	require.NoError(t, err, "GetUsersInSTPUserGroup must not error")
	if mockTests {
		exp := []*STPUserGroupMemberResponse{{UserID: 10000, STPID: 1, CreateTime: types.Time(time.Unix(1548000000, 0))}}
		assert.Equal(t, exp, result, "GetUsersInSTPUserGroup should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersInSTPUserGroup should return users")
}

func TestAddUsersToSTPUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.AddUsersToSTPUserGroup(t.Context(), 0, []uint64{12312312})
	require.ErrorIs(t, err, errSTPGroupIDRequired, "AddUsersToSTPUserGroup must reject an empty group ID")

	_, err = e.AddUsersToSTPUserGroup(t.Context(), 2123, nil)
	require.ErrorIs(t, err, errUserIDRequired, "AddUsersToSTPUserGroup must reject an empty user list")

	_, err = e.AddUsersToSTPUserGroup(t.Context(), 2123, []uint64{12312312, 0})
	require.ErrorIs(t, err, errUserIDRequired, "AddUsersToSTPUserGroup must reject an empty user ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		result, err := e.AddUsersToSTPUserGroup(t.Context(), 1, []uint64{12312312, 2132234232})
		require.NoError(t, err, "AddUsersToSTPUserGroup must not error")
		assert.NotEmpty(t, result, "AddUsersToSTPUserGroup should return the group's users")
		return
	}

	// The recorded mock server cannot match a body that is an array of numbers, so a test server checks the request
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "request method should be POST")
		assert.Equal(t, "/api/v4/account/stp_groups/1/users", r.URL.Path, "request path should name the STP group")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the request body should not error")
		assert.JSONEq(t, `[12312312,2132234232]`, string(body), "request body should list the user IDs")
		_, err = w.Write([]byte(`[{"user_id":12312312,"stp_id":1,"create_time":1548000000},{"user_id":2132234232,"stp_id":1,"create_time":1548000001}]`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})

	result, err := ex.AddUsersToSTPUserGroup(t.Context(), 1, []uint64{12312312, 2132234232})
	require.NoError(t, err, "AddUsersToSTPUserGroup must not error")
	exp := []*STPUserGroupMemberResponse{
		{UserID: 12312312, STPID: 1, CreateTime: types.Time(time.Unix(1548000000, 0))},
		{UserID: 2132234232, STPID: 1, CreateTime: types.Time(time.Unix(1548000001, 0))},
	}
	assert.Equal(t, exp, result, "AddUsersToSTPUserGroup should decode every field")
}

func TestDeleteUserFromSTPUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteUserFromSTPUserGroup(t.Context(), 0, []uint64{12312312})
	require.ErrorIs(t, err, errSTPGroupIDRequired, "DeleteUserFromSTPUserGroup must reject an empty group ID")

	_, err = e.DeleteUserFromSTPUserGroup(t.Context(), 2123, nil)
	require.ErrorIs(t, err, errUserIDRequired, "DeleteUserFromSTPUserGroup must reject an empty user list")

	_, err = e.DeleteUserFromSTPUserGroup(t.Context(), 2123, []uint64{0})
	require.ErrorIs(t, err, errUserIDRequired, "DeleteUserFromSTPUserGroup must reject an empty user ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DeleteUserFromSTPUserGroup(t.Context(), 1, []uint64{12312312, 2132234232})
	require.NoError(t, err, "DeleteUserFromSTPUserGroup must not error")
	if mockTests {
		exp := []*STPUserGroupMemberResponse{{UserID: 10000, STPID: 1, CreateTime: types.Time(time.Unix(1548000000, 0))}}
		assert.Equal(t, exp, result, "DeleteUserFromSTPUserGroup should decode every field")
	}
}

func TestConfigureGTFeeDeduction(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err := e.ConfigureGTFeeDeduction(t.Context(), false)
	assert.NoError(t, err, "ConfigureGTFeeDeduction should not error")
}

func TestGetGTFeeDeductionConfiguration(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	enabled, err := e.GetGTFeeDeductionConfiguration(t.Context())
	require.NoError(t, err, "GetGTFeeDeductionConfiguration must not error")
	if mockTests {
		assert.True(t, enabled, "GetGTFeeDeductionConfiguration should decode the setting")
	}
}
