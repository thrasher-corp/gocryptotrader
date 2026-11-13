package bitstamp

// RevokeAPIKeysResponse holds the API keys revoked by a revoke all API keys request
type RevokeAPIKeysResponse struct {
	RevokedAPIKeys []string `json:"revoked_api_keys"`
}
