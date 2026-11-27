package standin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportRoundTrip(t *testing.T) {
	t.Parallel()
	transport := Transport(func(r *http.Request) (Reply, error) {
		switch r.URL.Path {
		case "/known":
			return Reply{Status: http.StatusForbidden, Body: `{"error":true}`}, nil
		case "/echo":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return Reply{}, err
			}
			return Reply{Status: http.StatusOK, Body: string(body)}, nil
		}
		return Reply{}, ErrNoReply
	})
	client := Client(transport)

	t.Run("known request", func(t *testing.T) {
		t.Parallel()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/known", http.NoBody)
		require.NoError(t, err, "NewRequestWithContext must not error")
		resp, err := client.Do(req)
		require.NoError(t, err, "Do must not error")
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "ReadAll must not error")
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, "Do should return the reply's status")
		assert.Equal(t, "403 Forbidden", resp.Status, "Do should return the reply's status line")
		assert.Equal(t, `{"error":true}`, string(body), "Do should return the reply's body")
	})

	t.Run("unknown request", func(t *testing.T) {
		t.Parallel()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/unknown", http.NoBody)
		require.NoError(t, err, "NewRequestWithContext must not error")
		resp, err := client.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		assert.ErrorIs(t, err, ErrNoReply, "Do should fail a request no reply covers")
	})

	t.Run("reply reads the request body", func(t *testing.T) {
		t.Parallel()
		body := fileBody(t, `{"amount":1}`)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/echo", body)
		require.NoError(t, err, "NewRequestWithContext must not error")
		resp, err := client.Do(req)
		require.NoError(t, err, "Do must not error")
		defer resp.Body.Close()
		got, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "ReadAll must not error")
		assert.Equal(t, `{"amount":1}`, string(got), "the reply should read the request body")
		assert.ErrorIs(t, body.Close(), os.ErrClosed, "Do should close the request body")
	})

	t.Run("reply replaces the request body", func(t *testing.T) {
		t.Parallel()
		body := fileBody(t, `{}`)
		replacer := Client(func(r *http.Request) (Reply, error) {
			r.Body = http.NoBody
			return Reply{Status: http.StatusOK}, nil
		})
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/known", body)
		require.NoError(t, err, "NewRequestWithContext must not error")
		resp, err := replacer.Do(req)
		require.NoError(t, err, "Do must not error")
		defer resp.Body.Close()
		assert.ErrorIs(t, body.Close(), os.ErrClosed, "Do should close the body the request was sent with")
	})

	t.Run("unknown request closes the request body", func(t *testing.T) {
		t.Parallel()
		body := fileBody(t, `{}`)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/unknown", body)
		require.NoError(t, err, "NewRequestWithContext must not error")
		resp, err := client.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		assert.ErrorIs(t, err, ErrNoReply, "Do should fail a request no reply covers")
		assert.ErrorIs(t, body.Close(), os.ErrClosed, "Do should close the request body")
	})

	t.Run("cancelled request", func(t *testing.T) {
		t.Parallel()
		var asked atomic.Bool
		replier := Client(func(*http.Request) (Reply, error) {
			asked.Store(true)
			return Reply{Status: http.StatusOK}, nil
		})
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		body := fileBody(t, `{}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://example.com/known", body)
		require.NoError(t, err, "NewRequestWithContext must not error")
		resp, err := replier.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		assert.ErrorIs(t, err, context.Canceled, "Do should return the cancelled context's error")
		assert.False(t, asked.Load(), "Do should not ask for a reply once the request is cancelled")
		assert.ErrorIs(t, body.Close(), os.ErrClosed, "Do should close the request body")
	})

	t.Run("request body close error", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, path string
			cancelled  bool
			want       error
		}{
			{name: "after a reply", path: "/known", want: errCloseFailed},
			{name: "after a request no reply covers", path: "/unknown", want: ErrNoReply},
			{name: "after a cancelled request", path: "/known", cancelled: true, want: context.Canceled},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if tc.cancelled {
					cancel()
				}
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://example.com"+tc.path, closeFailingBody{strings.NewReader(`{}`)})
				require.NoError(t, err, "NewRequestWithContext must not error")
				resp, err := transport.RoundTrip(req)
				if resp != nil {
					resp.Body.Close()
				}
				assert.Nil(t, resp, "RoundTrip should not return a response with an error")
				assert.ErrorIs(t, err, tc.want, "RoundTrip should return the first error the request meets")
			})
		}
	})
}

var errCloseFailed = errors.New("close failed")

type closeFailingBody struct{ io.Reader }

func (closeFailingBody) Close() error { return errCloseFailed }

// fileBody returns an open file holding content: a request body that, once closed, can't be read
func fileBody(t *testing.T, content string) *os.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "body")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600), "WriteFile must not error")
	f, err := os.Open(path)
	require.NoError(t, err, "Open must not error")
	return f
}
