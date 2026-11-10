package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestCheckBasicAuth(t *testing.T) {
	t.Parallel()
	encode := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	valid := "Basic " + encode("test-user:test-password")
	for _, tc := range []struct {
		name        string
		values      []string
		credentials *BasicAuth
		wantErr     error
	}{
		{name: "missing metadata", wantErr: ErrMissingAuthHeader},
		{name: "empty value slice", values: []string{}, wantErr: ErrMissingAuthHeader},
		{name: "empty first value", values: []string{""}, wantErr: ErrNotBasicAuth},
		{name: "scheme without space", values: []string{"Basic"}, wantErr: ErrNotBasicAuth},
		{name: "wrong scheme", values: []string{"Bearer token"}, wantErr: ErrNotBasicAuth},
		{name: "scheme substring", values: []string{"NotBasic " + encode("test-user:test-password")}, wantErr: ErrNotBasicAuth},
		{name: "tab separator", values: []string{"Basic\t" + encode("test-user:test-password")}, wantErr: ErrNotBasicAuth},
		{name: "two spaces", values: []string{"Basic  " + encode("test-user:test-password")}, wantErr: ErrMalformedBasicAuth},
		{name: "empty token", values: []string{"Basic "}, wantErr: ErrMalformedBasicAuth},
		{name: "invalid base64", values: []string{"Basic !!!"}, wantErr: ErrMalformedBasicAuth},
		{name: "missing colon", values: []string{"Basic " + encode("test-user")}, wantErr: ErrMalformedBasicAuth},
		{name: "valid credentials", values: []string{valid}},
		{name: "lower case scheme", values: []string{"basic " + encode("test-user:test-password")}},
		{name: "wrong username", values: []string{"Basic " + encode("other-user:test-password")}, wantErr: ErrBasicAuthMismatch},
		{name: "wrong password", values: []string{"Basic " + encode("test-user:other-password")}, wantErr: ErrBasicAuthMismatch},
		{name: "unexpected empty password", values: []string{"Basic " + encode("test-user:")}, wantErr: ErrBasicAuthMismatch},
		{name: "configured empty password", values: []string{"Basic " + encode("test-user:")}, credentials: &BasicAuth{Username: "test-user"}},
		{name: "password containing colons", values: []string{"Basic " + encode("test-user:part:two:")}, credentials: &BasicAuth{Username: "test-user", Password: "part:two:"}},
		{name: "password suffix is compared", values: []string{"Basic " + encode("test-user:test-password:extra")}, wantErr: ErrBasicAuthMismatch},
		{name: "first value decides", values: []string{valid, "Basic"}},
		{name: "invalid first value is not skipped", values: []string{"Basic", valid}, wantErr: ErrNotBasicAuth},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			username, password := "test-user", "test-password"
			if tc.credentials != nil {
				username, password = tc.credentials.Username, tc.credentials.Password
			}
			var md metadata.MD
			if tc.values != nil {
				md = metadata.MD{"authorization": tc.values}
			}
			err := CheckBasicAuth(md, username, password)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "CheckBasicAuth should return the expected error")
			} else {
				assert.NoError(t, err, "CheckBasicAuth should accept matching credentials")
			}
			r := &http.Request{Header: http.Header{"Authorization": tc.values}}
			user, pass, ok := r.BasicAuth()
			assert.Equal(t, ok && user == username && pass == password, err == nil, "CheckBasicAuth should agree with net/http basic auth parsing")
		})
	}

	err := CheckBasicAuth(metadata.Pairs("authorization", "Basic !!!"), "test-user", "test-password")
	assert.ErrorIs(t, err, base64.CorruptInputError(0), "CheckBasicAuth should wrap the base64 decoding error")
}

func TestCredentialsMatch(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, user, pass, wantUser, wantPass string
		match                                bool
	}{
		{"matching", "user", "pass", "user", "pass", true},
		{"wrong username", "User", "pass", "user", "pass", false},
		{"wrong password", "user", "Pass", "user", "pass", false},
		{"longer username", "users", "pass", "user", "pass", false},
		{"longer password", "user", "passes", "user", "pass", false},
		{"empty credentials", "", "", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.match, CredentialsMatch(tc.user, tc.pass, tc.wantUser, tc.wantPass), "CredentialsMatch should compare both complete values")
		})
	}
}

func TestGetRequestMetadata(t *testing.T) {
	t.Parallel()
	creds := BasicAuth{Username: "test-user", Password: "part:two:"}
	md, err := creds.GetRequestMetadata(t.Context())
	require.NoError(t, err, "GetRequestMetadata must not error")
	assert.NoError(t, CheckBasicAuth(metadata.New(md), creds.Username, creds.Password), "CheckBasicAuth should accept the metadata GetRequestMetadata produces")
}

func TestBasicAuthFunc(t *testing.T) {
	t.Parallel()
	encode := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	authenticate := BasicAuthFunc("test-user", "test-password")
	for _, tc := range []struct {
		name    string
		md      metadata.MD
		wantErr error
	}{
		{name: "no incoming metadata", wantErr: ErrMissingAuthHeader},
		{name: "malformed header", md: metadata.Pairs("authorization", "Basic"), wantErr: ErrNotBasicAuth},
		{name: "wrong password", md: metadata.Pairs("authorization", "Basic "+encode("test-user:wrong")), wantErr: ErrBasicAuthMismatch},
		{name: "valid credentials", md: metadata.Pairs("authorization", "Basic "+encode("test-user:test-password"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			if tc.md != nil {
				ctx = metadata.NewIncomingContext(ctx, tc.md)
			}
			gotCtx, err := authenticate(ctx)
			assert.Same(t, ctx, gotCtx, "BasicAuthFunc should return the incoming context unchanged")
			if tc.wantErr == nil {
				assert.NoError(t, err, "BasicAuthFunc should accept matching credentials")
				return
			}
			assert.Equal(t, codes.Unauthenticated, status.Code(err), "BasicAuthFunc should reject with codes.Unauthenticated")
			assert.Equal(t, tc.wantErr.Error(), status.Convert(err).Message(), "BasicAuthFunc should report why the credentials were rejected")
		})
	}
}

func TestRequireBasicAuth(t *testing.T) {
	t.Parallel()
	handler := RequireBasicAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("MEOW"))
		assert.NoError(t, err, "Write should not error")
	}), "bobmarley", "Sup3rdup3rS3cr3t")

	for _, tc := range []struct {
		name               string
		username, password string
		noCredentials      bool
		wantAllowed        bool
	}{
		{name: "valid credentials", username: "bobmarley", password: "Sup3rdup3rS3cr3t", wantAllowed: true},
		{name: "valid username but invalid password", username: "bobmarley", password: "wrongpass"},
		{name: "invalid username but valid password", username: "bonk", password: "Sup3rdup3rS3cr3t"},
		{name: "invalid username and password", username: "bonk", password: "wif"},
		{name: "no credentials", noCredentials: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
			if !tc.noCredentials {
				req.SetBasicAuth(tc.username, tc.password)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if tc.wantAllowed {
				assert.Equal(t, http.StatusOK, rr.Code, "HTTP status code should be 200")
				assert.Equal(t, "MEOW", rr.Body.String(), "response body should come from the wrapped handler")
				return
			}
			assert.Equal(t, http.StatusUnauthorized, rr.Code, "HTTP status code should be 401")
			assert.Equal(t, `Basic realm="restricted"`, rr.Header().Get("WWW-Authenticate"), "a rejection should ask for Basic credentials")
			assert.Equal(t, "Access denied\n", rr.Body.String(), "a rejection should not reach the wrapped handler")
		})
	}
}
