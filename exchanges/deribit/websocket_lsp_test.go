package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebsocketAPIGetLSPParticipantConfig(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.GetLSPParticipantConfig is not tested live, as " + liveLSPParticipantSkip)
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetLSPParticipantConfig(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetLSPParticipantConfig must not error")
	assert.Equal(t, testLSPParticipantConfigResponse(), result, "WebsocketAPI.GetLSPParticipantConfig should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().GetLSPParticipantConfig(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetLSPParticipantConfig should error without an authenticated connection")
}

func TestWebsocketAPIGetLSPParticipants(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetLSPParticipants(t.Context(), false)
	require.NoError(t, err, "WebsocketAPI.GetLSPParticipants must not error")
	if mockTests {
		assert.Equal(t, testLSPParticipantsResponse(), result, "WebsocketAPI.GetLSPParticipants should decode every field")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLSPParticipants(t.Context(), false)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetLSPParticipants should error without an authenticated connection")
}

func TestWebsocketAPIGetLSPParticipantsUsage(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	var userID uint64 = mockLSPParticipantUserID
	if !mockTests {
		userID = 0
	}
	result, err := ex.WebsocketAPI().GetLSPParticipantsUsage(t.Context(), userID)
	require.NoError(t, err, "WebsocketAPI.GetLSPParticipantsUsage must not error")
	if mockTests {
		assert.Equal(t, testLSPParticipantsUsageResponse(), result, "WebsocketAPI.GetLSPParticipantsUsage should decode every field")
	} else {
		assert.NotZero(t, result.CooldownPeriod, "WebsocketAPI.GetLSPParticipantsUsage should return the cooldown period")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLSPParticipantsUsage(t.Context(), userID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetLSPParticipantsUsage should error without an authenticated connection")
}

func TestWebsocketAPIGetLSPUsage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.GetLSPUsage is not tested live, as " + liveLSPParticipantSkip)
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetLSPUsage(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetLSPUsage must not error")
	assert.Equal(t, &LSPUsageResponse{CooldownPeriod: 300000, Groups: testLSPGroupUsage()}, result, "WebsocketAPI.GetLSPUsage should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().GetLSPUsage(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetLSPUsage should error without an authenticated connection")
}
