package deribit

// setHeartbeatParams contains the heartbeat interval, in seconds, of a public/set_heartbeat request
type setHeartbeatParams struct {
	Interval float64 `json:"interval"`
}

// cancelOnDisconnectParams contains the scope of a cancel on disconnect request; an empty scope is left out, so Deribit
// applies its default, the connection scope
type cancelOnDisconnectParams struct {
	Scope string `json:"scope,omitempty"`
}

// CancelOnDisconnectResponse contains whether cancel on disconnect is enabled for a connection or the account
type CancelOnDisconnectResponse struct {
	// Scope is connection or account
	Scope   string `json:"scope"`
	Enabled bool   `json:"enabled"`
}
