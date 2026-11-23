package comms

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/standin"
)

func TestUnauthenticatedReply(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		url  string
		want standin.Reply
	}{
		{url: "https://slack.com/api/rtm.start", want: standin.Reply{Status: http.StatusOK, Body: slackUnauthenticatedReply}},
		{url: "https://api.telegram.org/bot/getMe", want: standin.Reply{Status: http.StatusNotFound, Body: telegramUnauthenticatedReply}},
	} {
		t.Run(tc.url, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, tc.url, http.NoBody)
			require.NoError(t, err, "NewRequestWithContext must not error")
			got, err := unauthenticatedReply(req)
			require.NoError(t, err, "unauthenticatedReply must not error")
			assert.Equal(t, tc.want, got, "unauthenticatedReply should return the service's reply")
		})
	}

	t.Run("other hosts", func(t *testing.T) {
		t.Parallel()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
		require.NoError(t, err, "NewRequestWithContext must not error")
		_, err = unauthenticatedReply(req)
		assert.ErrorIs(t, err, standin.ErrNoReply, "unauthenticatedReply should refuse a host it has no reply for")
	})
}
