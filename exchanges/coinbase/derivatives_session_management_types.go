package coinbase

// derivativesCancelOnDisconnectParams are the cancel on disconnect methods' params; an empty scope is left out, so the
// gateway applies its default, the connection scope
type derivativesCancelOnDisconnectParams struct {
	Scope string `json:"scope,omitempty"`
}

// DerivativesCancelOnDisconnectResponse is whether cancel on disconnect is enabled for a connection or the account
type DerivativesCancelOnDisconnectResponse struct {
	Enabled bool `json:"enabled"`
	// Scope is connection or account, as read
	Scope string `json:"scope"`
}
