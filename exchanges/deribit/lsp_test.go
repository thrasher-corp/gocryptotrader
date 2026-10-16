package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// mockLSPParticipantUserID is the participant of Deribit's documented private/get_lsp_participants_usage example
	mockLSPParticipantUserID = 172345
	// liveLSPParticipantSkip explains why live tests skip the methods only a participant subaccount can call
	liveLSPParticipantSkip = "it needs the API key of a Liquidity Support Program participant subaccount"
)

// testLSPGroupUsage is the documented usage of each cooldown group the mock usage requests return
func testLSPGroupUsage() map[string]LSPGroupUsage {
	return map[string]LSPGroupUsage{
		"1":   {LimitPercentage: 15, Used: 250000, CooldownStart: tradingTime(1750000100000), CooldownEnd: tradingTime(1750000400000)},
		"2":   {LimitPercentage: 10},
		"3":   {LimitPercentage: 10},
		"4":   {LimitPercentage: 10},
		"5":   {LimitPercentage: 10},
		"6":   {},
		"rwa": {LimitPercentage: 10},
	}
}

// testLSPParticipantConfigResponse is the documented participant configuration the mock requests return
func testLSPParticipantConfigResponse() *LSPParticipantConfigResponse {
	return &LSPParticipantConfigResponse{Config: LSPParticipantConfig{Enabled: true, GroupLimits: map[string]float64{"1": 15, "6": 0}}}
}

// testLSPParticipantsResponse is the documented participants response the mock requests return
func testLSPParticipantsResponse() *LSPParticipantsResponse {
	return &LSPParticipantsResponse{Participants: []LSPParticipant{{UserID: mockLSPParticipantUserID, Enabled: true}, {UserID: 172346}}}
}

// testLSPParticipantsUsageResponse is the documented participants usage response the mock requests return
func testLSPParticipantsUsageResponse() *LSPParticipantsUsageResponse {
	return &LSPParticipantsUsageResponse{
		CooldownPeriod: 300000,
		Participants:   []LSPParticipantUsage{{UserID: mockLSPParticipantUserID, Groups: testLSPGroupUsage()}},
	}
}

func TestGetLSPParticipantConfig(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("GetLSPParticipantConfig is not tested live, as " + liveLSPParticipantSkip)
	}
	result, err := e.GetLSPParticipantConfig(t.Context())
	require.NoError(t, err, "GetLSPParticipantConfig must not error")
	assert.Equal(t, testLSPParticipantConfigResponse(), result, "GetLSPParticipantConfig should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLSPParticipantConfig(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLSPParticipantConfig should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLSPParticipantConfig(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLSPParticipantConfig should return the request failure")
}

func TestGetLSPParticipants(t *testing.T) {
	t.Parallel()
	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"participants":[]}}`)
	_, err := ex.GetLSPParticipants(t.Context(), true)
	require.NoError(t, err, "GetLSPParticipants must not error for a detailed request")
	assert.JSONEq(t, `{"detailed":true}`, lastParams(), "GetLSPParticipants should ask for each participant's configuration")

	skipLiveWithoutCredentials(t)
	result, err := e.GetLSPParticipants(t.Context(), false)
	require.NoError(t, err, "GetLSPParticipants must not error")
	if mockTests {
		assert.Equal(t, testLSPParticipantsResponse(), result, "GetLSPParticipants should decode every field")
	}

	result, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"participants":[{"user_id":172345,"config":{"enabled":true,"group_limits":{"rwa":5.5}}}]}}`).GetLSPParticipants(t.Context(), true)
	require.NoError(t, err, "GetLSPParticipants must not error for a detailed response")
	exp := &LSPParticipantsResponse{Participants: []LSPParticipant{{UserID: mockLSPParticipantUserID, Config: &LSPParticipantConfig{Enabled: true, GroupLimits: map[string]float64{"rwa": 5.5}}}}}
	assert.Equal(t, exp, result, "GetLSPParticipants should decode each participant's configuration")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLSPParticipants(t.Context(), false)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLSPParticipants should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLSPParticipants(t.Context(), false)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLSPParticipants should return the request failure")
}

func TestGetLSPParticipantsUsage(t *testing.T) {
	t.Parallel()
	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"participants":[]}}`)
	_, err := ex.GetLSPParticipantsUsage(t.Context(), 0)
	require.NoError(t, err, "GetLSPParticipantsUsage must not error for every participant")
	assert.JSONEq(t, `{}`, lastParams(), "GetLSPParticipantsUsage should leave out a zero user ID, so Deribit returns every participant")

	skipLiveWithoutCredentials(t)
	var userID uint64 = mockLSPParticipantUserID
	if !mockTests {
		userID = 0
	}
	result, err := e.GetLSPParticipantsUsage(t.Context(), userID)
	require.NoError(t, err, "GetLSPParticipantsUsage must not error")
	if mockTests {
		assert.Equal(t, testLSPParticipantsUsageResponse(), result, "GetLSPParticipantsUsage should decode every field")
	} else {
		assert.NotZero(t, result.CooldownPeriod, "GetLSPParticipantsUsage should return the cooldown period")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLSPParticipantsUsage(t.Context(), userID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLSPParticipantsUsage should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLSPParticipantsUsage(t.Context(), userID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLSPParticipantsUsage should return the request failure")
}

func TestGetLSPUsage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("GetLSPUsage is not tested live, as " + liveLSPParticipantSkip)
	}
	result, err := e.GetLSPUsage(t.Context())
	require.NoError(t, err, "GetLSPUsage must not error")
	assert.Equal(t, &LSPUsageResponse{CooldownPeriod: 300000, Groups: testLSPGroupUsage()}, result, "GetLSPUsage should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLSPUsage(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLSPUsage should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLSPUsage(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLSPUsage should return the request failure")
}
