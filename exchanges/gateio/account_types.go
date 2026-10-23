package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// gtFeeDeductionReply holds whether GT fee deduction is enabled
type gtFeeDeductionReply struct {
	Enabled bool `json:"enabled"`
}

// AccountDetailsResponse represents account detail information
type AccountDetailsResponse struct {
	IPWhitelist     []string          `json:"ip_whitelist"`
	CurrencyPairs   []currency.Pair   `json:"currency_pairs"`
	UserID          uint64            `json:"user_id"`
	VIPTier         uint64            `json:"tier"`
	Key             AccountKeyDetails `json:"key"`
	CopyTradingRole uint64            `json:"copy_trading_role"` // 0 - Normal user, 1 - Copy trading leader, 2 - Copy trading follower, 3 - Both leader and follower
}

// AccountKeyDetails holds the mode of the API key in use
type AccountKeyDetails struct {
	Mode uint64 `json:"mode"` // 1 - Classic mode, 2 - Legacy unified mode
}

// AccountMainKeyResponse holds a main account API key
type AccountMainKeyResponse struct {
	State         uint64               `json:"state"` // 1 - Normal, 2 - Locked, 3 - Frozen
	Mode          uint64               `json:"mode"`  // 1 - Classic mode, 2 - Legacy unified mode
	Name          string               `json:"name"`
	CurrencyPairs []currency.Pair      `json:"currency_pairs"`
	UserID        uint64               `json:"user_id"`
	IPWhitelist   []string             `json:"ip_whitelist"`
	Permissions   []*SubAccountKeyPerm `json:"perms"`
	MaskedKey     string               `json:"key"`
	CreatedAt     types.Time           `json:"created_at"`
	// Gate documents updated_at but its example responses send update_at, so both are decoded
	UpdatedAt  types.Time `json:"updated_at"`
	UpdateAt   types.Time `json:"update_at"`
	LastAccess types.Time `json:"last_access"`
}

// UserTransactionRateLimitInfoResponse represents user transaction rate limit information
type UserTransactionRateLimitInfoResponse struct {
	Type      string       `json:"type"`
	Tier      types.Number `json:"tier"`
	Ratio     types.Number `json:"ratio"`
	MainRatio types.Number `json:"main_ratio"`
	UpdatedAt types.Time   `json:"updated_at"`
}

// STPUserGroupResponse holds self-trade prevention user group
type STPUserGroupResponse struct {
	ID         uint64     `json:"id"`
	Name       string     `json:"name"`
	CreatorID  uint64     `json:"creator_id"`
	CreateTime types.Time `json:"create_time"`
}

// STPUserGroupMemberResponse holds STP user group member detail
type STPUserGroupMemberResponse struct {
	UserID     uint64     `json:"user_id"`
	STPID      uint64     `json:"stp_id"`
	CreateTime types.Time `json:"create_time"`
}
