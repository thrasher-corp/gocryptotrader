package deribit

// HelloRequest introduces the client software to Deribit
type HelloRequest struct {
	ClientName    string `json:"client_name"`
	ClientVersion string `json:"client_version"`
}

// HelloResponse contains the API version Deribit introduces itself with
type HelloResponse struct {
	Version string `json:"version"`
}

// StatusResponse contains the platform's lock status
type StatusResponse struct {
	// Locked is true when the platform is locked in every currency, partial when it is locked in some and false when in
	// none; Deribit sends it as a string
	Locked        string   `json:"locked"`
	LockedIndices []string `json:"locked_indices"`
}

// TestConnectionRequest contains the expected result of a public/test request
type TestConnectionRequest struct {
	ExpectedResult string `json:"expected_result"`
}

// TestConnectionResponse contains the API version
type TestConnectionResponse struct {
	Version string `json:"version"`
}
