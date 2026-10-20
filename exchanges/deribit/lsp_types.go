package deribit

import "github.com/thrasher-corp/gocryptotrader/types"

// LSPParticipantConfigResponse contains the calling subaccount's Liquidity Support Program participation
type LSPParticipantConfigResponse struct {
	Config LSPParticipantConfig `json:"config"`
}

// LSPParticipantConfig contains a subaccount's Liquidity Support Program participation, which locks it to reduce only
// trading when the platform's participant lock is on
type LSPParticipantConfig struct {
	// Enabled is whether the subaccount is eligible to receive assigned positions
	Enabled bool `json:"enabled"`
	// GroupLimits overrides the assignment limit of a cooldown group, "1" to "6" for the crypto tiers or "rwa" for real
	// world asset instruments, as a percentage of the participant's equity; a group left out takes the platform default
	// and zero opts out of the group
	GroupLimits map[string]float64 `json:"group_limits"`
}

// LSPParticipantsRequest contains whether a private/get_lsp_participants request returns each participant's
// configuration
type LSPParticipantsRequest struct {
	Detailed bool `json:"detailed,omitempty"`
}

// LSPParticipantsResponse contains the Liquidity Support Program participants under the main account
type LSPParticipantsResponse struct {
	Participants []LSPParticipant `json:"participants"`
}

// LSPParticipant contains a Liquidity Support Program participant; Enabled is sent alone, and Config in its place for a
// detailed request
type LSPParticipant struct {
	UserID  uint64                `json:"user_id"`
	Enabled bool                  `json:"enabled"`
	Config  *LSPParticipantConfig `json:"config"`
}

// LSPParticipantsUsageRequest contains the participant a private/get_lsp_participants_usage request selects, every
// participant when zero
type LSPParticipantsUsageRequest struct {
	UserID uint64 `json:"user_id,omitempty"`
}

// LSPParticipantsUsageResponse contains the assignment usage of each Liquidity Support Program participant under the
// main account
type LSPParticipantsUsageResponse struct {
	// CooldownPeriod is the length of the fixed cooldown window in milliseconds
	CooldownPeriod uint64                `json:"cooldown_period_ms"`
	Participants   []LSPParticipantUsage `json:"participants"`
}

// LSPParticipantUsage contains a Liquidity Support Program participant's assignment usage by cooldown group, "1" to
// "6" for the crypto tiers or "rwa" for real world asset instruments; a group without a limit or usage may be left out
type LSPParticipantUsage struct {
	UserID uint64                   `json:"user_id"`
	Groups map[string]LSPGroupUsage `json:"groups"`
}

// LSPUsageResponse contains the calling participant's assignment usage by cooldown group, "1" to "6" for the crypto
// tiers or "rwa" for real world asset instruments; a group without a limit or usage may be left out
type LSPUsageResponse struct {
	// CooldownPeriod is the length of the fixed cooldown window in milliseconds
	CooldownPeriod uint64                   `json:"cooldown_period_ms"`
	Groups         map[string]LSPGroupUsage `json:"groups"`
}

// LSPGroupUsage contains a cooldown group's assignment limit and its usage within the current fixed window; the
// remaining capacity is left to the caller to derive from the participant's equity
type LSPGroupUsage struct {
	// LimitPercentage is the group's assignment limit as a percentage of the participant's equity
	LimitPercentage float64 `json:"pct"`
	// Used is the USD notional assigned within the window, which resets to zero at once when the window closes
	Used float64 `json:"used"`
	// CooldownStart is the first assignment of the window and CooldownEnd its close; both are zero without usage
	CooldownStart types.Time `json:"cooldown_start"`
	CooldownEnd   types.Time `json:"cooldown_end"`
}
