package auth

import (
	"context"
	"encoding/base64"
)

// BasicAuth stores a basic auth username/password
type BasicAuth struct {
	Username string
	Password string
}

// GetRequestMetadata is a implementation of the GetRequestMetadata function
func (b BasicAuth) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	auth := b.Username + ":" + b.Password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))
	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

// RequireTransportSecurity is required for basic auth
func (BasicAuth) RequireTransportSecurity() bool {
	return true
}
