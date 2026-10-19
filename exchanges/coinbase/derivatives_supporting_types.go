package coinbase

// DerivativesTestExpectedResultException makes public/test fail with an internal_server_error, to exercise a client's
// error handling
const DerivativesTestExpectedResultException = "exception"

// Gateway lock statuses, which the gateway sends as strings
const (
	DerivativesLockedTrue    = "true"
	DerivativesLockedPartial = "partial"
	DerivativesLockedFalse   = "false"
)

// DerivativesHelloRequest introduces the client software to the gateway
type DerivativesHelloRequest struct {
	ClientName    string
	ClientVersion string
}

// derivativesHelloParams are public/hello's params
type derivativesHelloParams struct {
	ClientName    string `json:"client_name"`
	ClientVersion string `json:"client_version"`
}

// DerivativesHelloResponse is the API version the gateway introduces itself with
type DerivativesHelloResponse struct {
	Version string `json:"version"`
}

// DerivativesStatusResponse is the platform's lock status
type DerivativesStatusResponse struct {
	// Locked is true when the platform is locked in every currency, partial when it is locked in some and false when in
	// none
	Locked string `json:"locked"`
	// LockedIndices are the price indexes locked platform-wide, which the gateway leaves out while nothing is locked
	LockedIndices []string `json:"locked_indices"`
}

// derivativesTestParams are public/test's params
type derivativesTestParams struct {
	ExpectedResult string `json:"expected_result"`
}

// DerivativesTestConnectionResponse is the API version
type DerivativesTestConnectionResponse struct {
	Version string `json:"version"`
}
