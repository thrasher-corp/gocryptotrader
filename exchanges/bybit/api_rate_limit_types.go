package bybit

import "github.com/thrasher-corp/gocryptotrader/types"

// SetAPIRateLimitRequest holds the per second API rate limit to set for UIDs and a business type
type SetAPIRateLimitRequest struct {
	UIDs []string
	// BusinessType is SPOT, DERIVATIVES or OPTIONS
	BusinessType string
	Rate         uint64
}

// SetAPIRateLimitResponse holds the outcome of each API rate limit set. Bybit's parameter table names the list "list"
// but its example response, and the matching broker endpoint's docs, name it "result"
type SetAPIRateLimitResponse struct {
	Result []SetAPIRateLimitResult `json:"result"`
}

// SetAPIRateLimitResult holds the outcome of setting an API rate limit
type SetAPIRateLimitResult struct {
	// UIDs is a comma separated list of UIDs
	UIDs         string `json:"uids"`
	BusinessType string `json:"bizType"`
	Rate         uint64 `json:"rate"`
	Success      bool   `json:"success"`
	// Message explains the outcome, such as "Requested limit exceeds maximum allowed per user"
	Message string `json:"msg"`
}

// APIRateLimitResponse holds the API rate limits of the requested UIDs
type APIRateLimitResponse struct {
	List []APIRateLimit `json:"list"`
}

// APIRateLimit holds the per second API rate limit of UIDs for a business type
type APIRateLimit struct {
	// UIDs is a comma separated list of UIDs
	UIDs         string `json:"uids"`
	BusinessType string `json:"bizType"`
	Rate         uint64 `json:"rate"`
}

// APIRateLimitCapResponse holds the institution's API rate limit caps per business type
type APIRateLimitCapResponse struct {
	List []APIRateLimitCap `json:"list"`
}

// APIRateLimitCap holds the API rate limit caps and usage of a business type, per second. Bybit documents the limits
// as integers but sends them as strings in its example response, so either is accepted
type APIRateLimitCap struct {
	BusinessType string `json:"bizType"`
	// TotalRate is the rate allocated across the master account and all subaccounts
	TotalRate types.Number `json:"totalRate"`
	// InstitutionalCap depends on the institution's pro level
	InstitutionalCap types.Number `json:"insCap"`
	UIDCap           types.Number `json:"uidCap"`
}

// AllAPIRateLimitsRequest holds the parameters of GetAllAPIRateLimits; every field is optional
type AllAPIRateLimitsRequest struct {
	// Limit is the page size, from 1 to 1000
	Limit  uint64
	Cursor string
	// UIDs limits the result to these UIDs, which may belong to different master accounts
	UIDs []string
}

// AllAPIRateLimitsResponse holds a page of API rate limits
type AllAPIRateLimitsResponse struct {
	NextPageCursor string         `json:"nextPageCursor"`
	List           []APIRateLimit `json:"list"`
}
