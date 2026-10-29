package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebsocketAPISecurityKeyAuthorisation(t *testing.T) {
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
}
