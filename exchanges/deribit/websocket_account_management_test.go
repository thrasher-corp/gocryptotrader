package deribit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPIChangeAPIKeyName(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testChangeAPIKeyNameRequest(t)
		result, err := ex.WebsocketAPI().ChangeAPIKeyName(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.ChangeAPIKeyName must not error")
		assert.Equal(t, arg.Name, result.Name, "WebsocketAPI.ChangeAPIKeyName should return the new name")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		arg := testChangeAPIKeyNameRequest(t)
		result, err := ex.WebsocketAPI().ChangeAPIKeyName(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.ChangeAPIKeyName must not error")
		assert.Equal(t, testAPIKey(), result, "WebsocketAPI.ChangeAPIKeyName should decode every field")
	})

	arg := &ChangeAPIKeyNameRequest{ID: mockAPIKeyID, Name: mockAPIKeyName}
	_, err := newTestExchange(t).WebsocketAPI().ChangeAPIKeyName(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ChangeAPIKeyName should error without an authenticated connection")
}

func TestWebsocketAPIChangeMarginModel(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().ChangeMarginModel(t.Context(), testChangeMarginModelRequest())
		require.NoError(t, err, "WebsocketAPI.ChangeMarginModel must not error")
		assert.NotEmpty(t, result, "WebsocketAPI.ChangeMarginModel should return the change's effect on each currency")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().ChangeMarginModel(t.Context(), testChangeMarginModelRequest())
		require.NoError(t, err, "WebsocketAPI.ChangeMarginModel must not error")
		assert.Equal(t, testMarginModelChanges(), result, "WebsocketAPI.ChangeMarginModel should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().ChangeMarginModel(t.Context(), testChangeMarginModelRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ChangeMarginModel should error without an authenticated connection")
}

func TestWebsocketAPIChangeScopeInAPIKey(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testChangeScopeInAPIKeyRequest(t)
		result, err := ex.WebsocketAPI().ChangeScopeInAPIKey(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.ChangeScopeInAPIKey must not error")
		assert.ElementsMatch(t, arg.MaxScope, result.MaxScope, "WebsocketAPI.ChangeScopeInAPIKey should return the new scope")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		arg := testChangeScopeInAPIKeyRequest(t)
		result, err := ex.WebsocketAPI().ChangeScopeInAPIKey(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.ChangeScopeInAPIKey must not error")
		assert.Equal(t, testAPIKey(), result, "WebsocketAPI.ChangeScopeInAPIKey should decode every field")
	})

	arg := &ChangeScopeInAPIKeyRequest{ID: mockAPIKeyID, MaxScope: testAPIKeyScopes()}
	_, err := newTestExchange(t).WebsocketAPI().ChangeScopeInAPIKey(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ChangeScopeInAPIKey should error without an authenticated connection")
}

func TestWebsocketAPIChangeSubaccountName(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testChangeSubaccountNameRequest(t)
		require.NoError(t, ex.WebsocketAPI().ChangeSubaccountName(t.Context(), arg), "WebsocketAPI.ChangeSubaccountName must not error")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		arg := testChangeSubaccountNameRequest(t)
		require.NoError(t, ex.WebsocketAPI().ChangeSubaccountName(t.Context(), arg), "WebsocketAPI.ChangeSubaccountName must not error")
	})

	arg := &ChangeSubaccountNameRequest{SubaccountID: mockAccountSubaccountID, Name: mockSubaccountName}
	err := newTestExchange(t).WebsocketAPI().ChangeSubaccountName(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ChangeSubaccountName should error without an authenticated connection")
}

func TestWebsocketAPICreateAPIKey(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().CreateAPIKey(t.Context(), liveTestAPIKeyRequest())
		require.NoError(t, err, "WebsocketAPI.CreateAPIKey must not error")
		assert.NotEmpty(t, result.ClientSecret, "WebsocketAPI.CreateAPIKey should return the key's secret")
		require.NoError(t, ex.WebsocketAPI().RemoveAPIKey(t.Context(), result.ID), "WebsocketAPI.RemoveAPIKey must remove the created API key")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().CreateAPIKey(t.Context(), testCreateAPIKeyRequest())
		require.NoError(t, err, "WebsocketAPI.CreateAPIKey must not error")
		assert.Equal(t, testAPIKey(), result, "WebsocketAPI.CreateAPIKey should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().CreateAPIKey(t.Context(), testCreateAPIKeyRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CreateAPIKey should error without an authenticated connection")

	t.Run("security key challenge", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("confirming a security key challenge needs an account with two-factor authentication and its secret")
		}
		ex, venue := newWebsocketTestExchange(t)
		testSecurityKeyAuthorisation(t, ex.WebsocketAPI().CreateAPIKey, "WebsocketAPI.CreateAPIKey")
		requests := venue.received("private/create_api_key")
		require.Len(t, requests, 2, "the mock websocket must receive the challenged and the confirmed request")
		exp := `{"max_scope":"account:read","name":"` + mockChallengedAPIKeyName + `","authorization_data":"` + testSecurityKeyCode + `","challenge":"` + testSecurityKeyChallenge + `"}`
		assert.JSONEq(t, exp, string(requests[1].Params), "WebsocketAPI.CreateAPIKey should add the authorisation to the params")
	})
}

func TestWebsocketAPICreateSubaccount(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		skipLiveIfSecurityKeysEnabled(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().CreateSubaccount(t.Context())
		if requireAnsweredOnMerits(t, err, "WebsocketAPI.CreateSubaccount") {
			require.NotZero(t, result.ID, "WebsocketAPI.CreateSubaccount must return the new subaccount's ID")
			t.Cleanup(func() {
				assert.NoError(t, e.RemoveSubaccount(context.WithoutCancel(t.Context()), result.ID), "RemoveSubaccount should remove the test's subaccount")
			})
		}
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().CreateSubaccount(t.Context())
		require.NoError(t, err, "WebsocketAPI.CreateSubaccount must not error")
		assert.Equal(t, testCreatedSubaccount(), result, "WebsocketAPI.CreateSubaccount should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().CreateSubaccount(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CreateSubaccount should error without an authenticated connection")
}

func TestWebsocketAPIDeleteMember(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().DeleteMember(t.Context(), liveForeignID)
		requireRefusedOnMerits(t, err, "WebsocketAPI.DeleteMember")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().DeleteMember(t.Context(), mockMemberID)
		require.NoError(t, err, "WebsocketAPI.DeleteMember must not error")
		assert.Equal(t, &MembersResponse{Members: []Member{}, IsDirectAccessAllowed: true}, result, "WebsocketAPI.DeleteMember should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().DeleteMember(t.Context(), mockMemberID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.DeleteMember should error without an authenticated connection")
}

func TestWebsocketAPIDisableAPIKey(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		id := testAPIKeyID(t)
		result, err := ex.WebsocketAPI().DisableAPIKey(t.Context(), id)
		require.NoError(t, err, "WebsocketAPI.DisableAPIKey must not error")
		assert.False(t, result.Enabled, "WebsocketAPI.DisableAPIKey should return the key disabled")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		id := testAPIKeyID(t)
		result, err := ex.WebsocketAPI().DisableAPIKey(t.Context(), id)
		require.NoError(t, err, "WebsocketAPI.DisableAPIKey must not error")
		assert.Equal(t, testAPIKey(), result, "WebsocketAPI.DisableAPIKey should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().DisableAPIKey(t.Context(), mockAPIKeyID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.DisableAPIKey should error without an authenticated connection")
}

func TestWebsocketAPIEditAPIKey(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testEditAPIKeyRequest(t)
		result, err := ex.WebsocketAPI().EditAPIKey(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.EditAPIKey must not error")
		assert.Equal(t, arg.Name, result.Name, "WebsocketAPI.EditAPIKey should return the new name")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		arg := testEditAPIKeyRequest(t)
		result, err := ex.WebsocketAPI().EditAPIKey(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.EditAPIKey must not error")
		assert.Equal(t, testAPIKey(), result, "WebsocketAPI.EditAPIKey should decode every field")
	})

	arg := &EditAPIKeyRequest{ID: mockAPIKeyID, MaxScope: testAPIKeyScopes(), Name: mockEditedAPIKeyName}
	_, err := newTestExchange(t).WebsocketAPI().EditAPIKey(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.EditAPIKey should error without an authenticated connection")
}

func TestWebsocketAPIEnableAffiliateProgram(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		info, err := ex.WebsocketAPI().GetAffiliateProgramInfo(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetAffiliateProgramInfo must not error")
		if !info.IsEnabled {
			t.Skip("enabling the affiliate program cannot be undone, and it is not enabled on the account")
		}
		requireAnsweredOnMerits(t, ex.WebsocketAPI().EnableAffiliateProgram(t.Context()), "WebsocketAPI.EnableAffiliateProgram")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketAPI().EnableAffiliateProgram(t.Context()), "WebsocketAPI.EnableAffiliateProgram must not error")
	})

	err := newTestExchange(t).WebsocketAPI().EnableAffiliateProgram(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.EnableAffiliateProgram should error without an authenticated connection")
}

func TestWebsocketAPIEnableAPIKey(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		id := testAPIKeyID(t)
		result, err := ex.WebsocketAPI().EnableAPIKey(t.Context(), id)
		require.NoError(t, err, "WebsocketAPI.EnableAPIKey must not error")
		assert.True(t, result.Enabled, "WebsocketAPI.EnableAPIKey should return the key enabled")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		id := testAPIKeyID(t)
		result, err := ex.WebsocketAPI().EnableAPIKey(t.Context(), id)
		require.NoError(t, err, "WebsocketAPI.EnableAPIKey must not error")
		assert.Equal(t, testAPIKey(), result, "WebsocketAPI.EnableAPIKey should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().EnableAPIKey(t.Context(), mockAPIKeyID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.EnableAPIKey should error without an authenticated connection")
}

func TestWebsocketAPIGetAccessLog(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAccessLog(t.Context(), &GetAccessLogRequest{Count: mockAccessLogCount})
		require.NoError(t, err, "WebsocketAPI.GetAccessLog must not error")
		assert.LessOrEqual(t, len(result.Entries), mockAccessLogCount, "WebsocketAPI.GetAccessLog should return at most the entries asked for")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAccessLog(t.Context(), &GetAccessLogRequest{Count: mockAccessLogCount})
		require.NoError(t, err, "WebsocketAPI.GetAccessLog must not error")
		assert.Equal(t, testAccessLog(), result, "WebsocketAPI.GetAccessLog should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetAccessLog(t.Context(), &GetAccessLogRequest{})
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetAccessLog should error without an authenticated connection")
}

func TestWebsocketAPIGetAccountSummaries(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAccountSummaries(t.Context(), &GetAccountSummariesRequest{Extended: true})
		require.NoError(t, err, "WebsocketAPI.GetAccountSummaries must not error")
		assert.NotEmpty(t, result.Summaries, "WebsocketAPI.GetAccountSummaries should return a summary for each currency the account holds")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAccountSummaries(t.Context(), &GetAccountSummariesRequest{Extended: true})
		require.NoError(t, err, "WebsocketAPI.GetAccountSummaries must not error")
		assert.Equal(t, testAccountSummaries(), result, "WebsocketAPI.GetAccountSummaries should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetAccountSummaries(t.Context(), &GetAccountSummariesRequest{})
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetAccountSummaries should error without an authenticated connection")
}

func TestWebsocketAPIGetAccountSummary(t *testing.T) {
	t.Parallel()
	arg := &GetAccountSummaryRequest{Currency: currency.BTC, Extended: true}

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAccountSummary(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetAccountSummary must not error")
		assert.Equal(t, currency.BTC, result.Currency, "WebsocketAPI.GetAccountSummary should return the summary in the currency asked for")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAccountSummary(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetAccountSummary must not error")
		exp := &AccountSummaryResponse{AccountSummary: testBTCAccountSummary(), AccountDetails: testAccountDetails()}
		assert.Equal(t, exp, result, "WebsocketAPI.GetAccountSummary should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetAccountSummary(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetAccountSummary should error without an authenticated connection")
}

func TestWebsocketAPIGetAffiliateProgramInfo(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAffiliateProgramInfo(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetAffiliateProgramInfo must not error")
		assert.NotNil(t, result, "WebsocketAPI.GetAffiliateProgramInfo should return the affiliate program information")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAffiliateProgramInfo(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetAffiliateProgramInfo must not error")
		assert.Equal(t, testAffiliateProgramInfo(), result, "WebsocketAPI.GetAffiliateProgramInfo should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetAffiliateProgramInfo(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetAffiliateProgramInfo should error without an authenticated connection")
}

func TestWebsocketAPIGetAccountCurrencies(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAccountCurrencies(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetAccountCurrencies must not error")
		assert.NotEmpty(t, result, "WebsocketAPI.GetAccountCurrencies should return the account's currencies")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetAccountCurrencies(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetAccountCurrencies must not error")
		exp := []AccountCurrencyResponse{{Currency: currency.NewCode("btc")}, {Currency: currency.NewCode("eth")}, {Currency: currency.NewCode("usdc")}}
		assert.Equal(t, exp, result, "WebsocketAPI.GetAccountCurrencies should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetAccountCurrencies(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetAccountCurrencies should error without an authenticated connection")
}

func TestWebsocketAPIGetEmailLanguage(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetEmailLanguage(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetEmailLanguage must not error")
		assert.NotEmpty(t, result, "WebsocketAPI.GetEmailLanguage should return the language")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetEmailLanguage(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetEmailLanguage must not error")
		assert.Equal(t, mockEmailLanguage, result, "WebsocketAPI.GetEmailLanguage should decode the language")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetEmailLanguage(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetEmailLanguage should error without an authenticated connection")
}

func TestWebsocketAPIGetMembers(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetMembers(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetMembers must not error")
		assert.NotNil(t, result, "WebsocketAPI.GetMembers should return the account's members")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetMembers(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetMembers must not error")
		assert.Equal(t, testMembers(), result, "WebsocketAPI.GetMembers should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetMembers(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetMembers should error without an authenticated connection")
}

func TestWebsocketAPIGetNewAnnouncements(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().GetNewAnnouncements(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetNewAnnouncements must not error")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetNewAnnouncements(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetNewAnnouncements must not error")
		assert.Equal(t, testNewAnnouncements(), result, "WebsocketAPI.GetNewAnnouncements should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetNewAnnouncements(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetNewAnnouncements should error without an authenticated connection")
}

func TestWebsocketAPIGetPosition(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetPosition(t.Context(), mockPositionInstrument)
		require.NoError(t, err, "WebsocketAPI.GetPosition must not error")
		assert.Equal(t, mockPositionInstrument, result.InstrumentName, "WebsocketAPI.GetPosition should return the position in the instrument asked for")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetPosition(t.Context(), mockPositionInstrument)
		require.NoError(t, err, "WebsocketAPI.GetPosition must not error")
		assert.Equal(t, testPosition(), result, "WebsocketAPI.GetPosition should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetPosition(t.Context(), mockPositionInstrument)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetPosition should error without an authenticated connection")
}

func TestWebsocketAPIGetPositions(t *testing.T) {
	t.Parallel()
	arg := &GetPositionsRequest{Currency: currency.BTC, Kind: KindFuture}

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetPositions(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetPositions must not error")
		for i := range result {
			assert.Equal(t, KindFuture, result[i].Kind, "WebsocketAPI.GetPositions should return positions of the kind asked for")
		}
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetPositions(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetPositions must not error")
		assert.Equal(t, testPositions(), result, "WebsocketAPI.GetPositions should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetPositions(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetPositions should error without an authenticated connection")
}

func TestWebsocketAPIGetSubaccounts(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetSubaccounts(t.Context(), true)
		require.NoError(t, err, "WebsocketAPI.GetSubaccounts must not error")
		assert.NotEmpty(t, result, "WebsocketAPI.GetSubaccounts should return the main account")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetSubaccounts(t.Context(), true)
		require.NoError(t, err, "WebsocketAPI.GetSubaccounts must not error")
		assert.Equal(t, testSubaccounts(), result, "WebsocketAPI.GetSubaccounts should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetSubaccounts(t.Context(), true)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetSubaccounts should error without an authenticated connection")
}

func TestWebsocketAPIGetSubaccountsDetails(t *testing.T) {
	t.Parallel()
	arg := &GetSubaccountsDetailsRequest{Currency: currency.BTC, WithOpenOrders: true}

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetSubaccountsDetails(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetSubaccountsDetails must not error")
		for i := range result {
			assert.NotZero(t, result[i].UserID, "WebsocketAPI.GetSubaccountsDetails should identify each account")
		}
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetSubaccountsDetails(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.GetSubaccountsDetails must not error")
		assert.Equal(t, testSubaccountsDetails(), result, "WebsocketAPI.GetSubaccountsDetails should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetSubaccountsDetails(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetSubaccountsDetails should error without an authenticated connection")
}

func TestWebsocketAPIGetTransactionLog(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetTransactionLog(t.Context(), testTransactionLogRequest())
		require.NoError(t, err, "WebsocketAPI.GetTransactionLog must not error")
		assert.LessOrEqual(t, len(result.Logs), mockTransactionLogCount, "WebsocketAPI.GetTransactionLog should return at most the entries asked for")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetTransactionLog(t.Context(), testTransactionLogRequest())
		require.NoError(t, err, "WebsocketAPI.GetTransactionLog must not error")
		assert.Equal(t, testTransactionLog(), result, "WebsocketAPI.GetTransactionLog should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetTransactionLog(t.Context(), testTransactionLogRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetTransactionLog should error without an authenticated connection")
}

func TestWebsocketAPIGetUserLocks(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetUserLocks(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetUserLocks must not error")
		assert.NotEmpty(t, result, "WebsocketAPI.GetUserLocks should return the account's state in each currency")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetUserLocks(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetUserLocks must not error")
		assert.Equal(t, testUserLocks(), result, "WebsocketAPI.GetUserLocks should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetUserLocks(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetUserLocks should error without an authenticated connection")
}

func TestWebsocketAPIListAPIKeys(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().ListAPIKeys(t.Context())
		require.NoError(t, err, "WebsocketAPI.ListAPIKeys must not error")
		assert.NotEmpty(t, result, "WebsocketAPI.ListAPIKeys should return the API key in use")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().ListAPIKeys(t.Context())
		require.NoError(t, err, "WebsocketAPI.ListAPIKeys must not error")
		assert.Equal(t, testAPIKeys(), result, "WebsocketAPI.ListAPIKeys should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().ListAPIKeys(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ListAPIKeys should error without an authenticated connection")
}

func TestWebsocketAPIListCustodyAccounts(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().ListCustodyAccounts(t.Context(), currency.BTC)
		require.NoError(t, err, "WebsocketAPI.ListCustodyAccounts must not error")
		for i := range result {
			assert.Equal(t, currency.BTC, result[i].Currency, "WebsocketAPI.ListCustodyAccounts should return custody accounts in the currency asked for")
		}
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().ListCustodyAccounts(t.Context(), currency.BTC)
		require.NoError(t, err, "WebsocketAPI.ListCustodyAccounts must not error")
		assert.Equal(t, testCustodyAccounts(), result, "WebsocketAPI.ListCustodyAccounts should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().ListCustodyAccounts(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ListCustodyAccounts should error without an authenticated connection")
}

func TestWebsocketAPIPMESimulate(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().PMESimulate(t.Context(), &SimulatePortfolioRequest{Currency: currency.BTC})
		require.NoError(t, err, "WebsocketAPI.PMESimulate must not error")
		assert.NotEmpty(t, result.IndexPrices, "WebsocketAPI.PMESimulate should return the index prices simulated")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().PMESimulate(t.Context(), &SimulatePortfolioRequest{Currency: currency.BTC})
		require.NoError(t, err, "WebsocketAPI.PMESimulate must not error")
		assert.Equal(t, testPMESimulation(), result, "WebsocketAPI.PMESimulate should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().PMESimulate(t.Context(), &SimulatePortfolioRequest{Currency: currency.BTC})
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.PMESimulate should error without an authenticated connection")
}

func TestWebsocketAPIRemoveAPIKey(t *testing.T) {
	t.Parallel()
	id := uint64(mockAPIKeyID)

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		key, err := ex.WebsocketAPI().CreateAPIKey(t.Context(), liveTestAPIKeyRequest())
		require.NoError(t, err, "WebsocketAPI.CreateAPIKey must create an API key to remove")
		require.NoError(t, ex.WebsocketAPI().RemoveAPIKey(t.Context(), key.ID), "WebsocketAPI.RemoveAPIKey must not error")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketAPI().RemoveAPIKey(t.Context(), id), "WebsocketAPI.RemoveAPIKey must not error")
	})

	err := newTestExchange(t).WebsocketAPI().RemoveAPIKey(t.Context(), id)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.RemoveAPIKey should error without an authenticated connection")
}

func TestWebsocketAPIRemoveSubaccount(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		requireRefusedOnMerits(t, ex.WebsocketAPI().RemoveSubaccount(t.Context(), liveForeignID), "WebsocketAPI.RemoveSubaccount")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketAPI().RemoveSubaccount(t.Context(), mockAccountSubaccountID), "WebsocketAPI.RemoveSubaccount must not error")
	})

	err := newTestExchange(t).WebsocketAPI().RemoveSubaccount(t.Context(), mockAccountSubaccountID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.RemoveSubaccount should error without an authenticated connection")
}

func TestWebsocketAPIResetAPIKey(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		id := testAPIKeyID(t)
		result, err := ex.WebsocketAPI().ResetAPIKey(t.Context(), id)
		require.NoError(t, err, "WebsocketAPI.ResetAPIKey must not error")
		assert.NotEmpty(t, result.ClientSecret, "WebsocketAPI.ResetAPIKey should return the key's new secret")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		id := testAPIKeyID(t)
		result, err := ex.WebsocketAPI().ResetAPIKey(t.Context(), id)
		require.NoError(t, err, "WebsocketAPI.ResetAPIKey must not error")
		assert.Equal(t, testAPIKey(), result, "WebsocketAPI.ResetAPIKey should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().ResetAPIKey(t.Context(), mockAPIKeyID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ResetAPIKey should error without an authenticated connection")
}

func TestWebsocketAPISetAnnouncementAsRead(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		requireAnsweredOnMerits(t, ex.WebsocketAPI().SetAnnouncementAsRead(t.Context(), liveForeignID), "WebsocketAPI.SetAnnouncementAsRead")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketAPI().SetAnnouncementAsRead(t.Context(), mockAnnouncementID), "WebsocketAPI.SetAnnouncementAsRead must not error")
	})

	err := newTestExchange(t).WebsocketAPI().SetAnnouncementAsRead(t.Context(), mockAnnouncementID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetAnnouncementAsRead should error without an authenticated connection")
}

func TestWebsocketAPISetDisabledTradingProducts(t *testing.T) {
	t.Parallel()
	arg := &SetDisabledTradingProductsRequest{UserID: mockAccountSubaccountID, TradingProducts: []string{TradingProductFutures, TradingProductOptions}}

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		err := ex.WebsocketAPI().SetDisabledTradingProducts(t.Context(), &SetDisabledTradingProductsRequest{UserID: liveForeignID, TradingProducts: arg.TradingProducts})
		requireRefusedOnMerits(t, err, "WebsocketAPI.SetDisabledTradingProducts")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketAPI().SetDisabledTradingProducts(t.Context(), arg), "WebsocketAPI.SetDisabledTradingProducts must not error")
	})

	err := newTestExchange(t).WebsocketAPI().SetDisabledTradingProducts(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetDisabledTradingProducts should error without an authenticated connection")
}

func TestWebsocketAPISetEmailForSubaccount(t *testing.T) {
	t.Parallel()
	arg := &SetEmailForSubaccountRequest{SubaccountID: mockAccountSubaccountID, Email: mockSubaccountEmail}

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		err := ex.WebsocketAPI().SetEmailForSubaccount(t.Context(), &SetEmailForSubaccountRequest{SubaccountID: liveForeignID, Email: "gct-live-test@example.com"})
		requireRefusedOnMerits(t, err, "WebsocketAPI.SetEmailForSubaccount")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketAPI().SetEmailForSubaccount(t.Context(), arg), "WebsocketAPI.SetEmailForSubaccount must not error")
	})

	err := newTestExchange(t).WebsocketAPI().SetEmailForSubaccount(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetEmailForSubaccount should error without an authenticated connection")
}

func TestWebsocketAPISetEmailLanguage(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		language := testEmailLanguage(t)
		require.NoError(t, ex.WebsocketAPI().SetEmailLanguage(t.Context(), language), "WebsocketAPI.SetEmailLanguage must not error")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		language := testEmailLanguage(t)
		require.NoError(t, ex.WebsocketAPI().SetEmailLanguage(t.Context(), language), "WebsocketAPI.SetEmailLanguage must not error")
	})

	err := newTestExchange(t).WebsocketAPI().SetEmailLanguage(t.Context(), mockEmailLanguage)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetEmailLanguage should error without an authenticated connection")
}

func TestWebsocketAPISetMember(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testSetMemberRequestFor(t)
		result, err := ex.WebsocketAPI().SetMember(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.SetMember must not error")
		assert.NotEmpty(t, result.Members, "WebsocketAPI.SetMember should return the account's members")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		arg := testSetMemberRequestFor(t)
		result, err := ex.WebsocketAPI().SetMember(t.Context(), arg)
		require.NoError(t, err, "WebsocketAPI.SetMember must not error")
		assert.Equal(t, testMembers(), result, "WebsocketAPI.SetMember should decode every field")
	})

	arg := testSetMemberRequest()
	_, err := newTestExchange(t).WebsocketAPI().SetMember(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetMember should error without an authenticated connection")
}

func TestWebsocketAPISetSelfTradingConfig(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testSetSelfTradingConfigRequestFor(t)
		require.NoError(t, ex.WebsocketAPI().SetSelfTradingConfig(t.Context(), arg), "WebsocketAPI.SetSelfTradingConfig must not error")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		arg := testSetSelfTradingConfigRequestFor(t)
		require.NoError(t, ex.WebsocketAPI().SetSelfTradingConfig(t.Context(), arg), "WebsocketAPI.SetSelfTradingConfig must not error")
	})

	arg := testSetSelfTradingConfigRequest()
	err := newTestExchange(t).WebsocketAPI().SetSelfTradingConfig(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SetSelfTradingConfig should error without an authenticated connection")
}

func TestWebsocketAPISimulatePortfolio(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().SimulatePortfolio(t.Context(), testSimulatePortfolioRequest())
		require.NoError(t, err, "WebsocketAPI.SimulatePortfolio must not error")
		assert.Equal(t, currency.BTC, result.Currency, "WebsocketAPI.SimulatePortfolio should simulate the currency asked for")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().SimulatePortfolio(t.Context(), testSimulatePortfolioRequest())
		require.NoError(t, err, "WebsocketAPI.SimulatePortfolio must not error")
		assert.Equal(t, testSimulatedPortfolio(), result, "WebsocketAPI.SimulatePortfolio should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().SimulatePortfolio(t.Context(), testSimulatePortfolioRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SimulatePortfolio should error without an authenticated connection")
}

func TestWebsocketAPIToggleNotificationsFromSubaccount(t *testing.T) {
	t.Parallel()
	arg := &ToggleNotificationsFromSubaccountRequest{SubaccountID: mockAccountSubaccountID, State: true}

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		err := ex.WebsocketAPI().ToggleNotificationsFromSubaccount(t.Context(), &ToggleNotificationsFromSubaccountRequest{SubaccountID: liveForeignID, State: true})
		requireRefusedOnMerits(t, err, "WebsocketAPI.ToggleNotificationsFromSubaccount")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		require.NoError(t, ex.WebsocketAPI().ToggleNotificationsFromSubaccount(t.Context(), arg), "WebsocketAPI.ToggleNotificationsFromSubaccount must not error")
	})

	err := newTestExchange(t).WebsocketAPI().ToggleNotificationsFromSubaccount(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ToggleNotificationsFromSubaccount should error without an authenticated connection")
}

func TestWebsocketAPIToggleSubaccountLogin(t *testing.T) {
	t.Parallel()

	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newWebsocketTestExchange(t)
		arg := testToggleSubaccountLoginRequest(t)
		require.NoError(t, ex.WebsocketAPI().ToggleSubaccountLogin(t.Context(), arg), "WebsocketAPI.ToggleSubaccountLogin must not error")
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		arg := testToggleSubaccountLoginRequest(t)
		require.NoError(t, ex.WebsocketAPI().ToggleSubaccountLogin(t.Context(), arg), "WebsocketAPI.ToggleSubaccountLogin must not error")
	})

	arg := &ToggleSubaccountLoginRequest{SubaccountID: mockAccountSubaccountID, State: SubaccountLoginEnable}
	err := newTestExchange(t).WebsocketAPI().ToggleSubaccountLogin(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ToggleSubaccountLogin should error without an authenticated connection")
}

func TestWebsocketAPIGetAnnouncements(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetAnnouncements(t.Context(), testAnnouncementsRequest())
	require.NoError(t, err, "WebsocketAPI.GetAnnouncements must not error")
	if mockTests {
		assert.Equal(t, testAnnouncements(), result, "WebsocketAPI.GetAnnouncements should decode every field")
	} else {
		require.Len(t, result, mockAnnouncementsCount, "WebsocketAPI.GetAnnouncements must return the announcements asked for")
		assert.False(t, result[0].PublicationTimestamp.Time().After(time.UnixMilli(mockAnnouncementsStartTime)), "WebsocketAPI.GetAnnouncements should return announcements published up to the start time")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetAnnouncements(t.Context(), testAnnouncementsRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetAnnouncements should error without a websocket connection")
}
