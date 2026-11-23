package standin

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportRoundTrip(t *testing.T) {
	t.Parallel()
	client := Client(func(r *http.Request) (Reply, error) {
		if r.URL.Path != "/known" {
			return Reply{}, ErrNoReply
		}
		return Reply{Status: http.StatusForbidden, Body: `{"error":true}`}, nil
	})

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
}
