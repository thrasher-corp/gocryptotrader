package kraken

import "time"

// FuturesCheckAPIKeyResponse holds the details of the API key that signed a request
type FuturesCheckAPIKeyResponse struct {
	APIKey     string `json:"apiKey"`
	AccountUID string `json:"accountUid"`
	// IIBAN is the account's Kraken IIBAN, such as AA08 N84G CPAR UN7A, and is empty for an account without one
	IIBAN       string                   `json:"iiban"`
	CreatedAt   time.Time                `json:"createdAt"`
	Permissions FuturesAPIKeyPermissions `json:"permissions"`
	// AllowedCIDRBlock and AllowedCIDRBlocks hold ranges of IP addresses that may use the key, such as 192.168.0.0/16
	AllowedCIDRBlock  string   `json:"allowedCidrBlock"`
	AllowedCIDRBlocks []string `json:"allowedCidrBlocks"`
}

// FuturesAPIKeyPermissions holds an API key's access levels, each NO_ACCESS, READ_ONLY or FULL_ACCESS
type FuturesAPIKeyPermissions struct {
	General  string `json:"general"`
	Transfer string `json:"transfer"`
}
