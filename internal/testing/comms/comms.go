// Package comms stands in for the messaging services that the communications tests reach
package comms

import (
	"fmt"
	"net/http"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/standin"
)

// Replies Slack and Telegram send a client without a token, captured live
const (
	slackUnauthenticatedReply    = `{"ok":false,"error":"not_authed"}`
	telegramUnauthenticatedReply = `{"ok":false,"error_code":404,"description":"Not Found"}`
)

// UseTestHTTPClient installs the HTTP client the communications tests share. In mock runs it answers every Slack and
// Telegram request the way those services answer a client without a token; under mock_test_off it calls them live.
func UseTestHTTPClient() error {
	if !livetest.Mock {
		return common.SetHTTPClientWithTimeout(15 * time.Second)
	}
	return common.SetHTTPClient(standin.Client(unauthenticatedReply))
}

// unauthenticatedReply returns the reply Slack or Telegram sends a request without a token
func unauthenticatedReply(r *http.Request) (standin.Reply, error) {
	switch r.URL.Host {
	case "slack.com":
		return standin.Reply{Status: http.StatusOK, Body: slackUnauthenticatedReply}, nil
	case "api.telegram.org":
		return standin.Reply{Status: http.StatusNotFound, Body: telegramUnauthenticatedReply}, nil
	}
	return standin.Reply{}, fmt.Errorf("%w for host %q", standin.ErrNoReply, r.URL.Host)
}
