package deribit

// SecurityKeyChallengeError is returned in place of a private method's result when Deribit asks for the request to be
// confirmed with a security key, as it does on an account with two-factor authentication for withdrawals, API key
// management and other sensitive requests; sending the request again within a minute, with a context from
// WithSecurityKeyAuthorisation, confirms it
type SecurityKeyChallengeError struct {
	AuthorisationRequired bool `json:"security_key_authorization_required"`
	// Challenge identifies the request to confirm, for one minute
	Challenge string `json:"challenge"`
	// RelyingPartyID identifies Deribit to a WebAuthn hardware key
	RelyingPartyID string        `json:"rp_id"`
	SecurityKeys   []SecurityKey `json:"security_keys"`
}

// SecurityKey is a security key that can confirm a request
type SecurityKey struct {
	// Type is tfa for the time-based one-time password of a two-factor authentication app
	Type string `json:"type"`
	Name string `json:"name"`
}

// securityKeyAuthorisation confirms a security key challenge
type securityKeyAuthorisation struct {
	AuthorisationData string `json:"authorization_data"`
	Challenge         string `json:"challenge"`
}
