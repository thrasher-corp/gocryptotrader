package telegram

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	testErrNotFound = "Not Found"
)

func TestInitialConnectReportsEachUnauthorisedUsernameOnce(t *testing.T) {
	// Not parallel: the log hook, log config and common's HTTP client are process-wide; cleanup disables logging, removes the hook and installs a default client.
	var mu sync.Mutex
	var warnings []string
	t.Cleanup(func() {
		log.SetCustomLogHook(nil)
		assert.NoError(t, log.SetGlobalLogConfig(&log.Config{}), "SetGlobalLogConfig should not error")
		assert.NoError(t, common.SetHTTPClientWithTimeout(15*time.Second), "SetHTTPClientWithTimeout should not error")
	})
	require.NoError(t, log.SetGlobalLogConfig(log.GenDefaultSettings()), "SetGlobalLogConfig must not error")
	log.SetCustomLogHook(func(_, _ string, a ...any) bool {
		mu.Lock()
		warnings = append(warnings, fmt.Sprint(a...))
		mu.Unlock()
		return true
	})
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botdummy/getUpdates" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		// carol shares bob's ID and alice returns under another, so only a username key reports each name once
		_, err := fmt.Fprint(w, `{"ok":true,"result":[`+
			`{"update_id":1,"message":{"from":{"id":1,"username":"alice"}}},`+
			`{"update_id":2,"message":{"from":{"id":1,"username":"alice"}}},`+
			`{"update_id":3,"message":{"from":{"id":2,"username":"bob"}}},`+
			`{"update_id":4,"message":{"from":{"id":2,"username":"carol"}}},`+
			`{"update_id":5,"message":{"from":{"id":3,"username":"alice"}}}]}`)
		assert.NoError(t, err, "Fprint should not error")
	}))
	require.NoError(t, common.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")

	bot := Telegram{Token: "dummy", AuthorisedClients: map[string]int64{}}
	require.NoError(t, bot.InitialConnect(), "InitialConnect must not error")
	mu.Lock()
	defer mu.Unlock()
	assert.ElementsMatch(t, []string{
		"Telegram: Received message from unauthorised user: alice\n",
		"Telegram: Received message from unauthorised user: bob\n",
		"Telegram: Received message from unauthorised user: carol\n",
	}, warnings, "InitialConnect should report each unauthorised username once")
}

func TestSetup(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Communications: base.CommunicationsConfig{
		TelegramConfig: base.TelegramConfig{
			Name:              "Telegram",
			Enabled:           false,
			Verbose:           false,
			VerificationToken: "testest",
			AuthorisedClients: map[string]int64{"sender": 0},
		},
	}}
	commsCfg := cfg.GetCommunicationsConfig()
	var T Telegram
	T.Setup(&commsCfg)
	if T.Name != "Telegram" || T.Enabled || T.Token != "testest" || T.Verbose || len(T.AuthorisedClients) != 1 {
		t.Error("telegram Setup() error, unexpected setup values",
			T.Name,
			T.Enabled,
			T.Token,
			T.Verbose)
	}
}

func TestConnect(t *testing.T) {
	t.Parallel()
	var T Telegram
	if err := T.Connect(); err == nil {
		t.Error("expected error")
	}
}

func TestPushEvent(t *testing.T) {
	t.Parallel()
	var T Telegram
	err := T.PushEvent(base.Event{})
	assert.ErrorIs(t, err, ErrNotConnected)

	T.Connected = true
	T.AuthorisedClients = map[string]int64{"sender": 0}
	err = T.PushEvent(base.Event{})
	assert.NoError(t, err, "PushEvent should not error")

	T.AuthorisedClients = map[string]int64{"sender": 1337}
	err = T.PushEvent(base.Event{})
	assert.ErrorContains(t, err, testErrNotFound)
}

func TestHandleMessages(t *testing.T) {
	t.Parallel()
	var T Telegram
	for _, c := range []string{cmdHelp, cmdStart, cmdStatus, "Not a command"} {
		assert.ErrorContainsf(t, T.HandleMessages(c, 1337), testErrNotFound,
			"HandleMessages with command %q should error correctly", c)
	}
}

func TestGetUpdates(t *testing.T) {
	t.Parallel()
	var T Telegram
	if _, err := T.GetUpdates(); err != nil {
		t.Error("telegram GetUpdates() error", err)
	}
}

func TestTestConnection(t *testing.T) {
	t.Parallel()
	var T Telegram
	if err := T.TestConnection(); err.Error() != testErrNotFound {
		t.Errorf("received %s, expected: %s", err, testErrNotFound)
	}
}

func TestSendMessage(t *testing.T) {
	t.Parallel()
	var T Telegram
	err := T.SendMessage("Test message", 1337)
	assert.ErrorContains(t, err, testErrNotFound, "SendMessage should error correctly")
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	var T Telegram
	err := T.SendHTTPRequest("0.0.0.0", nil, nil)
	if err == nil {
		t.Error("telegram SendHTTPRequest() error")
	}
}
