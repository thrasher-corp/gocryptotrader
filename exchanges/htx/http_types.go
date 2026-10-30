package htx

// httpResponse checks exchange errors before decoding the endpoint's concrete response.
// Received distinguishes HTTP 204, for which the requester skips JSON decoding.
type httpResponse struct {
	Result        any
	Authenticated bool
	Version2      bool
	Futures       bool
	Version5      bool // V3 and V5 share numeric status metadata.
	Received      bool
}
