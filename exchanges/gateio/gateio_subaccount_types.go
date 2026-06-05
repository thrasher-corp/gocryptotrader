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
	Remark    string     `json:"remark"`
	LoginName string     `json:"login_name"`
	Password  string     `json:"password"`
	Email     string     `json:"email"`
	State     int64      `json:"state"`
	Type      int64      `json:"type"`
	UserID    uint64     `json:"user_id"`
	CreatedAt types.Time `json:"create_time"`
}

// SubAccountKeyRequest holds parameters for creating or updating a sub-account API key.
type SubAccountKeyRequest struct {
	Mode        int64                `json:"mode,omitempty"`
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
	Mode        int64                `json:"mode"`
	Name        string               `json:"name"`
	Permissions []*SubAccountKeyPerm `json:"perms"`
	IPWhitelist []string             `json:"ip_whitelist"`
	Key         string               `json:"key"`
	Secret      string               `json:"secret,omitempty"`
	State       int64                `json:"state"`
	CreatedAt   types.Time           `json:"created_at"`
	UpdatedAt   types.Time           `json:"updated_at"`
}

// SubAccountMode holds unified account mode information for a sub-account.
type SubAccountMode struct {
	UserID    uint64 `json:"user_id"`
	IsUnified bool   `json:"is_unified"`
	Mode      string `json:"mode"`
}
