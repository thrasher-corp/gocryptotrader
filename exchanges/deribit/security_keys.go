package deribit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// ErrSecurityKeyAuthorisationRequired is the error every SecurityKeyChallengeError wraps
var ErrSecurityKeyAuthorisationRequired = errors.New("security key authorisation required")

var errSecurityKeyParams = errors.New("params must encode as a JSON object to carry a security key authorisation")

// securityKeyChallengeField is the result field marking a security key challenge
var securityKeyChallengeField = []byte(`"security_key_authorization_required"`)

// securityKeyAuthorisationKey is the context key of a security key authorisation
type securityKeyAuthorisationKey struct{}

// Error reports the types of security key that can confirm the request
func (s *SecurityKeyChallengeError) Error() string {
	if len(s.SecurityKeys) == 0 {
		return ErrSecurityKeyAuthorisationRequired.Error()
	}
	keyTypes := make([]string, len(s.SecurityKeys))
	for i := range s.SecurityKeys {
		keyTypes[i] = s.SecurityKeys[i].Type
	}
	return ErrSecurityKeyAuthorisationRequired.Error() + ": " + strings.Join(keyTypes, ", ")
}

// Unwrap returns ErrSecurityKeyAuthorisationRequired
func (s *SecurityKeyChallengeError) Unwrap() error {
	return ErrSecurityKeyAuthorisationRequired
}

// WithSecurityKeyAuthorisation returns a context whose private requests confirm a security key challenge with the
// authorisation data of one of its security keys, such as the current code of a two-factor authentication app; Deribit
// accepts each code once, so the context should only send the challenged request again
func WithSecurityKeyAuthorisation(ctx context.Context, challenge, authorisationData string) context.Context {
	return context.WithValue(ctx, securityKeyAuthorisationKey{}, &securityKeyAuthorisation{AuthorisationData: authorisationData, Challenge: challenge})
}

// authorisedParams returns a private method's params with the context's security key authorisation added, or the
// params unchanged when the context holds none
func authorisedParams(ctx context.Context, params any) (any, error) {
	authorisation, ok := ctx.Value(securityKeyAuthorisationKey{}).(*securityKeyAuthorisation)
	if !ok {
		return params, nil
	}
	merged := make(map[string]json.RawMessage)
	for _, fields := range []any{params, authorisation} {
		encoded, err := json.Marshal(fields)
		if err != nil {
			return nil, err
		}
		// Decoding null would discard the fields merged so far
		if bytes.Equal(encoded, []byte("null")) {
			continue
		}
		if err := json.Unmarshal(encoded, &merged); err != nil {
			return nil, fmt.Errorf("%w: %w", errSecurityKeyParams, err)
		}
	}
	return merged, nil
}

// privateResult decodes a private method's result, or the security key challenge Deribit sends in its place as a
// successful result
type privateResult struct {
	result    any
	challenge *SecurityKeyChallengeError
}

// UnmarshalJSON decodes a security key challenge, or else the result
func (p *privateResult) UnmarshalJSON(data []byte) error {
	// Checking for the field first spares other results a second decode
	if bytes.Contains(data, securityKeyChallengeField) {
		var challenge SecurityKeyChallengeError
		if err := json.Unmarshal(data, &challenge); err == nil && challenge.AuthorisationRequired {
			p.challenge = &challenge
			return nil
		}
	}
	if p.result == nil {
		return nil
	}
	return json.Unmarshal(data, p.result)
}
