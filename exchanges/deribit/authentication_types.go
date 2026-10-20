package deribit

import "time"

// AuthenticateRequest requests an access token; the fields a grant type needs are listed with it
type AuthenticateRequest struct {
	// GrantType is client_credentials, which needs ClientID and ClientSecret, client_signature, which needs ClientID,
	// Timestamp and Signature, or refresh_token, which needs RefreshToken
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	// Timestamp, Signature, Nonce and Data are a client signature's fields; NewClientSignatureRequest fills them
	Timestamp time.Time `json:"-"`
	Signature string    `json:"signature,omitempty"`
	Nonce     string    `json:"nonce,omitempty"`
	Data      string    `json:"data,omitempty"`
	// State is returned in the response
	State string `json:"state,omitempty"`
	// Scope limits the token's access, such as connection, session:name, trade:read or expires:600, separated by spaces
	Scope string `json:"scope,omitempty"`
}

// AuthenticateParams contains an authentication request's params, with its timestamp in Unix milliseconds
type AuthenticateParams struct {
	*AuthenticateRequest
	Timestamp int64 `json:"timestamp,omitempty"`
}

// AccessTokenResponse contains an access token and the refresh token that renews it
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	// TokenType is bearer
	TokenType string `json:"token_type"`
	// ExpiresIn is the token's lifetime in seconds
	ExpiresIn    uint64 `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	State        string `json:"state"`
	SessionID    string `json:"sid"`
	// EnabledFeatures lists the API key's advanced features, restricted_block_trades or block_trade_approval
	EnabledFeatures []string `json:"enabled_features"`
	// MandatoryTwoFactorAuthenticationStatus is enabled when privileged methods need two-factor authentication
	MandatoryTwoFactorAuthenticationStatus string `json:"mandatory_tfa_status"`
	GoogleLogin                            bool   `json:"google_login"`
}

// ExchangeTokenRequest requests a token for another subject, such as a subaccount
type ExchangeTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
	SubjectID    uint64 `json:"subject_id"`
	// Scope overrides the new session's scope within the caller's permissions
	Scope string `json:"scope,omitempty"`
}

// ForkTokenRequest requests a token for a new named session
type ForkTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
	SessionName  string `json:"session_name"`
}

// LogoutRequest contains whether a logout invalidates the session's tokens, which Deribit does when it is left out, so
// it is always sent
type LogoutRequest struct {
	InvalidateToken bool `json:"invalidate_token"`
}
