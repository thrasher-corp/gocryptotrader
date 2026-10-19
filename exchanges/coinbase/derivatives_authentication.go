package coinbase

// The authentication section's HTTP method, public/auth, is GlobalDerivativesAPI.Authenticate in derivatives.go, as
// every private HTTP request depends on it; private/logout is websocket only, in websocket_derivatives_authentication.go

// derivativesLogoutParams are private/logout's params; the gateway invalidates the session's tokens when
// invalidate_token is left out, so it is always sent
type derivativesLogoutParams struct {
	InvalidateToken bool `json:"invalidate_token"`
}
