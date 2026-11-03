package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// mockLiquiditySupportProgramParticipantUserID is the participant of Deribit's documented
	// private/get_lsp_participants_usage example
	mockLiquiditySupportProgramParticipantUserID = 172345
	// liveLiquiditySupportProgramParticipantSkip explains why live tests skip the methods only a participant subaccount
	// can call
	liveLiquiditySupportProgramParticipantSkip = "it needs the API key of a Liquidity Support Program participant subaccount"
)

// testLiquiditySupportProgramGroupUsage is the documented usage of each cooldown group the mock usage requests return
func testLiquiditySupportProgramGroupUsage() map[string]LiquiditySupportProgramGroupUsage {
	return map[string]LiquiditySupportProgramGroupUsage{
		"1":   {LimitPercentage: 15, Used: 250000, CooldownStart: tradingTime(1750000100000), CooldownEnd: tradingTime(1750000400000)},
		"2":   {LimitPercentage: 10},
		"3":   {LimitPercentage: 10},
		"4":   {LimitPercentage: 10},
		"5":   {LimitPercentage: 10},
		"6":   {},
		"rwa": {LimitPercentage: 10},
	}
}

// testLiquiditySupportProgramParticipantConfigResponse is the documented participant configuration the mock requests
// return
func testLiquiditySupportProgramParticipantConfigResponse() *LiquiditySupportProgramParticipantConfigResponse {
	return &LiquiditySupportProgramParticipantConfigResponse{Config: LiquiditySupportProgramParticipantConfig{Enabled: true, GroupLimits: map[string]float64{"1": 15, "6": 0}}}
}

// testLiquiditySupportProgramParticipantsResponse is the documented participants response the mock requests return
func testLiquiditySupportProgramParticipantsResponse() *LiquiditySupportProgramParticipantsResponse {
	return &LiquiditySupportProgramParticipantsResponse{Participants: []LiquiditySupportProgramParticipant{{UserID: mockLiquiditySupportProgramParticipantUserID, Enabled: true}, {UserID: 172346}}}
}

// testLiquiditySupportProgramParticipantsUsageResponse is the documented participants usage response the mock requests
// return
func testLiquiditySupportProgramParticipantsUsageResponse() *LiquiditySupportProgramParticipantsUsageResponse {
	return &LiquiditySupportProgramParticipantsUsageResponse{
		CooldownPeriod: 300000,
		Participants:   []LiquiditySupportProgramParticipantUsage{{UserID: mockLiquiditySupportProgramParticipantUserID, Groups: testLiquiditySupportProgramGroupUsage()}},
	}
}

func TestGetLiquiditySupportProgramParticipantConfig(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("GetLiquiditySupportProgramParticipantConfig is not tested live, as " + liveLiquiditySupportProgramParticipantSkip)
	}
	result, err := e.GetLiquiditySupportProgramParticipantConfig(t.Context())
	require.NoError(t, err, "GetLiquiditySupportProgramParticipantConfig must not error")
	assert.Equal(t, testLiquiditySupportProgramParticipantConfigResponse(), result, "GetLiquiditySupportProgramParticipantConfig should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLiquiditySupportProgramParticipantConfig(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLiquiditySupportProgramParticipantConfig should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLiquiditySupportProgramParticipantConfig(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLiquiditySupportProgramParticipantConfig should return the request failure")
}

func TestGetLiquiditySupportProgramParticipants(t *testing.T) {
	t.Parallel()
	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"participants":[]}}`)
	_, err := ex.GetLiquiditySupportProgramParticipants(t.Context(), true)
	require.NoError(t, err, "GetLiquiditySupportProgramParticipants must not error for a detailed request")
	assert.JSONEq(t, `{"detailed":true}`, lastParams(), "GetLiquiditySupportProgramParticipants should ask for each participant's configuration")

	skipLiveWithoutCredentials(t)
	result, err := e.GetLiquiditySupportProgramParticipants(t.Context(), false)
	require.NoError(t, err, "GetLiquiditySupportProgramParticipants must not error")
	if mockTests {
		assert.Equal(t, testLiquiditySupportProgramParticipantsResponse(), result, "GetLiquiditySupportProgramParticipants should decode every field")
	}

	result, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"participants":[{"user_id":172345,"config":{"enabled":true,"group_limits":{"rwa":5.5}}}]}}`).GetLiquiditySupportProgramParticipants(t.Context(), true)
	require.NoError(t, err, "GetLiquiditySupportProgramParticipants must not error for a detailed response")
	exp := &LiquiditySupportProgramParticipantsResponse{Participants: []LiquiditySupportProgramParticipant{{UserID: mockLiquiditySupportProgramParticipantUserID, Config: &LiquiditySupportProgramParticipantConfig{Enabled: true, GroupLimits: map[string]float64{"rwa": 5.5}}}}}
	assert.Equal(t, exp, result, "GetLiquiditySupportProgramParticipants should decode each participant's configuration")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLiquiditySupportProgramParticipants(t.Context(), false)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLiquiditySupportProgramParticipants should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLiquiditySupportProgramParticipants(t.Context(), false)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLiquiditySupportProgramParticipants should return the request failure")
}

func TestGetLiquiditySupportProgramParticipantsUsage(t *testing.T) {
	t.Parallel()
	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"participants":[]}}`)
	_, err := ex.GetLiquiditySupportProgramParticipantsUsage(t.Context(), 0)
	require.NoError(t, err, "GetLiquiditySupportProgramParticipantsUsage must not error for every participant")
	assert.JSONEq(t, `{}`, lastParams(), "GetLiquiditySupportProgramParticipantsUsage should leave out a zero user ID, so Deribit returns every participant")

	skipLiveWithoutCredentials(t)
	var userID uint64 = mockLiquiditySupportProgramParticipantUserID
	if !mockTests {
		userID = 0
	}
	result, err := e.GetLiquiditySupportProgramParticipantsUsage(t.Context(), userID)
	require.NoError(t, err, "GetLiquiditySupportProgramParticipantsUsage must not error")
	if mockTests {
		assert.Equal(t, testLiquiditySupportProgramParticipantsUsageResponse(), result, "GetLiquiditySupportProgramParticipantsUsage should decode every field")
	} else {
		assert.NotZero(t, result.CooldownPeriod, "GetLiquiditySupportProgramParticipantsUsage should return the cooldown period")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLiquiditySupportProgramParticipantsUsage(t.Context(), userID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLiquiditySupportProgramParticipantsUsage should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLiquiditySupportProgramParticipantsUsage(t.Context(), userID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLiquiditySupportProgramParticipantsUsage should return the request failure")
}

func TestGetLiquiditySupportProgramUsage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("GetLiquiditySupportProgramUsage is not tested live, as " + liveLiquiditySupportProgramParticipantSkip)
	}
	result, err := e.GetLiquiditySupportProgramUsage(t.Context())
	require.NoError(t, err, "GetLiquiditySupportProgramUsage must not error")
	assert.Equal(t, &LiquiditySupportProgramUsageResponse{CooldownPeriod: 300000, Groups: testLiquiditySupportProgramGroupUsage()}, result, "GetLiquiditySupportProgramUsage should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLiquiditySupportProgramUsage(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLiquiditySupportProgramUsage should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLiquiditySupportProgramUsage(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLiquiditySupportProgramUsage should return the request failure")
}
