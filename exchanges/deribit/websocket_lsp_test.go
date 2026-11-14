package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebsocketAPIGetLiquiditySupportProgramParticipantConfig(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetLiquiditySupportProgramParticipantConfig(t.Context())
		if requireAnsweredOnMerits(t, err, "WebsocketAPI.GetLiquiditySupportProgramParticipantConfig") {
			assert.NotNil(t, result, "WebsocketAPI.GetLiquiditySupportProgramParticipantConfig should return the participant's configuration")
		}
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetLiquiditySupportProgramParticipantConfig(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetLiquiditySupportProgramParticipantConfig must not error")
		assert.Equal(t, testLiquiditySupportProgramParticipantConfigResponse(), result, "WebsocketAPI.GetLiquiditySupportProgramParticipantConfig should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetLiquiditySupportProgramParticipantConfig(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetLiquiditySupportProgramParticipantConfig should error without an authenticated connection")
}

func TestWebsocketAPIGetLiquiditySupportProgramParticipants(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetLiquiditySupportProgramParticipants(t.Context(), false)
	require.NoError(t, err, "WebsocketAPI.GetLiquiditySupportProgramParticipants must not error")
	if mockTests {
		assert.Equal(t, testLiquiditySupportProgramParticipantsResponse(), result, "WebsocketAPI.GetLiquiditySupportProgramParticipants should decode every field")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLiquiditySupportProgramParticipants(t.Context(), false)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetLiquiditySupportProgramParticipants should error without an authenticated connection")
}

func TestWebsocketAPIGetLiquiditySupportProgramParticipantsUsage(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	var userID uint64 = mockLiquiditySupportProgramParticipantUserID
	if !mockTests {
		userID = 0
	}
	result, err := ex.WebsocketAPI().GetLiquiditySupportProgramParticipantsUsage(t.Context(), userID)
	require.NoError(t, err, "WebsocketAPI.GetLiquiditySupportProgramParticipantsUsage must not error")
	if mockTests {
		assert.Equal(t, testLiquiditySupportProgramParticipantsUsageResponse(), result, "WebsocketAPI.GetLiquiditySupportProgramParticipantsUsage should decode every field")
	} else {
		assert.NotZero(t, result.CooldownPeriod, "WebsocketAPI.GetLiquiditySupportProgramParticipantsUsage should return the cooldown period")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLiquiditySupportProgramParticipantsUsage(t.Context(), userID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetLiquiditySupportProgramParticipantsUsage should error without an authenticated connection")
}

func TestWebsocketAPIGetLiquiditySupportProgramUsage(t *testing.T) {
	t.Parallel()
	t.Run("live API", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mocked case below checks the recorded replies")
		}
		skipLiveWithoutCredentials(t)
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetLiquiditySupportProgramUsage(t.Context())
		if requireAnsweredOnMerits(t, err, "WebsocketAPI.GetLiquiditySupportProgramUsage") {
			assert.NotNil(t, result, "WebsocketAPI.GetLiquiditySupportProgramUsage should return the participant's usage")
		}
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded replies are checked in mock mode")
		}
		ex, _ := newWebsocketTestExchange(t)
		result, err := ex.WebsocketAPI().GetLiquiditySupportProgramUsage(t.Context())
		require.NoError(t, err, "WebsocketAPI.GetLiquiditySupportProgramUsage must not error")
		assert.Equal(t, &LiquiditySupportProgramUsageResponse{CooldownPeriod: 300000, Groups: testLiquiditySupportProgramGroupUsage()}, result, "WebsocketAPI.GetLiquiditySupportProgramUsage should decode every field")
	})

	_, err := newTestExchange(t).WebsocketAPI().GetLiquiditySupportProgramUsage(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetLiquiditySupportProgramUsage should error without an authenticated connection")
}
