package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Basic auth validation errors
var (
	ErrMissingAuthHeader  = errors.New("authorization header missing")
	ErrNotBasicAuth       = errors.New("basic scheme not found in authorization header")
	ErrMalformedBasicAuth = errors.New("malformed basic authorization credentials")
	ErrBasicAuthMismatch  = errors.New("username/password mismatch")
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

// CheckBasicAuth validates the first authorization metadata value against the expected username and password
func CheckBasicAuth(md metadata.MD, username, password string) error {
	values := md.Get("authorization")
	if len(values) == 0 {
		return ErrMissingAuthHeader
	}
	scheme, token, ok := strings.Cut(values[0], " ")
	if !ok || !strings.EqualFold(scheme, "Basic") {
		return ErrNotBasicAuth
	}
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrMalformedBasicAuth, err)
	}
	user, pass, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return fmt.Errorf("%w: missing ':' separator", ErrMalformedBasicAuth)
	}
	if !CredentialsMatch(user, pass, username, password) {
		return ErrBasicAuthMismatch
	}
	return nil
}

// CredentialsMatch compares both credential pairs in constant time
func CredentialsMatch(user, pass, wantUser, wantPass string) bool {
	userHash, wantUserHash := sha256.Sum256([]byte(user)), sha256.Sum256([]byte(wantUser))
	passHash, wantPassHash := sha256.Sum256([]byte(pass)), sha256.Sum256([]byte(wantPass))
	return subtle.ConstantTimeCompare(userHash[:], wantUserHash[:])&subtle.ConstantTimeCompare(passHash[:], wantPassHash[:]) == 1
}

// BasicAuthFunc returns a gRPC auth function which requires the incoming metadata to carry the given Basic auth
// credentials, rejecting anything else with codes.Unauthenticated
func BasicAuthFunc(username, password string) func(context.Context) (context.Context, error) {
	return func(ctx context.Context) (context.Context, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		if err := CheckBasicAuth(md, username, password); err != nil {
			return ctx, status.Error(codes.Unauthenticated, err.Error())
		}
		return ctx, nil
	}
}

// RequireBasicAuth wraps next so that only requests carrying the given Basic auth credentials reach it
func RequireBasicAuth(next http.Handler, username, password string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || !CredentialsMatch(user, pass, username, password) {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
			http.Error(w, "Access denied", http.StatusUnauthorized)
			log.Warnf(log.GRPCSys, "gRPC proxy unauthorised access attempt. IP: %s Path: %q", r.RemoteAddr, r.URL.Path)
			return
		}
		next.ServeHTTP(w, r)
	})
}
