package gateio

import "github.com/thrasher-corp/gocryptotrader/types"

// CreateSubAccountRequest holds parameters for creating a new sub-account.
type CreateSubAccountRequest struct {
	Remark    string `json:"remark,omitempty"`
	LoginName string `json:"login_name"`
	Password  string `json:"password,omitempty"`
	Email     string `json:"email,omitempty"`
}

// SubAccount holds sub-account detail information.
type SubAccount struct {
	Remark     string     `json:"remark"`
	LoginName  string     `json:"login_name"`
	Password   string     `json:"password"`
	Email      string     `json:"email"`
	State      uint64     `json:"state"` // 1 - Normal, 2 - Locked
	Type       uint64     `json:"type"`  // 1 - Regular sub-account, 3 - Cross margin sub-account
	UserID     uint64     `json:"user_id"`
	CreateTime types.Time `json:"create_time"`
}

// SubAccountKeyRequest holds parameters for creating or updating a sub-account API key.
type SubAccountKeyRequest struct {
	Mode        uint64               `json:"mode,omitempty"` // 1 - Classic, 2 - Portfolio account
	Name        string               `json:"name,omitempty"`
	Permissions []*SubAccountKeyPerm `json:"perms,omitempty"`
	IPWhitelist []string             `json:"ip_whitelist,omitempty"`
}

// SubAccountKeyPerm holds permission name and read-only status for a sub-account API key.
type SubAccountKeyPerm struct {
	Name     string `json:"name,omitempty"`
	ReadOnly bool   `json:"read_only,omitempty"`
}

// SubAccountAPIKey holds detailed information about a sub-account API key.
type SubAccountAPIKey struct {
	UserID      uint64               `json:"user_id"`
	Mode        uint64               `json:"mode"` // 1 - Classic, 2 - Portfolio account
	Name        string               `json:"name"`
	Permissions []*SubAccountKeyPerm `json:"perms"`
	IPWhitelist []string             `json:"ip_whitelist"`
	Key         string               `json:"key"`
	Secret      string               `json:"secret,omitempty"`
	State       uint64               `json:"state"` // 1 - Normal, 2 - Frozen, 3 - Locked
	CreatedAt   types.Time           `json:"created_at"`
	// Gate documents updated_at but its list and get examples send update_at, so both are decoded
	UpdatedAt  types.Time `json:"updated_at"`
	UpdateAt   types.Time `json:"update_at"`
	LastAccess types.Time `json:"last_access"`
}

// SubAccountMode holds unified account mode information for a sub-account.
type SubAccountMode struct {
	UserID    uint64 `json:"user_id"`
	IsUnified bool   `json:"is_unified"`
	Mode      string `json:"mode"`
}
