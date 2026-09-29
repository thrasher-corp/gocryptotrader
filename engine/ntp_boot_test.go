package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestNTPBootModes(t *testing.T) {
	// Engine setup owns process-wide services, so these boot tests are serial.
	for _, level := range []int{-1, 0, 1} {
		t.Run(map[int]string{-1: "disabled", 0: "legacy startup value", 1: "periodic"}[level], func(t *testing.T) {
			address, received := ntpTestServer(t, nil)
			bot, err := NewFromSettings(&Settings{ConfigFile: config.TestFile, EnableDryRun: true, DataDir: t.TempDir(), EnableNTPClient: true}, nil)
			require.NoError(t, err, "engine must load its real configuration")
			for i := range bot.Config.Exchanges {
				bot.Config.Exchanges[i].Enabled = bot.Config.Exchanges[i].Name == testExchange
			}
			bot.Config.NTPClient = config.NTPClientConfig{Level: level, Pool: []string{address}}
			bot.Config.CheckNTPConfig()
			require.NoError(t, bot.Start(), "headless engine must start without waiting for an NTP reply or prompt")
			stopped := false
			t.Cleanup(func() {
				if !stopped {
					bot.Stop()
				}
			})
			if level == -1 {
				assert.Nil(t, bot.ntpManager, "disabled mode should not register a worker")
				return
			}
			require.NotNil(t, bot.ntpManager, "enabled mode must register the NTP manager")
			select {
			case <-received:
			case <-time.After(time.Second):
				t.Fatal("enabled manager must send its initial query during boot")
			}
			assert.True(t, bot.ntpManager.IsRunning(), "unanswered initial query should belong to a running worker")
			bot.Stop()
			stopped = true
			assert.False(t, bot.ntpManager.IsRunning(), "engine shutdown should join the NTP worker")
		})
	}
}

func TestNTPBootFailureJoinsWorker(t *testing.T) {
	bot, err := NewFromSettings(&Settings{ConfigFile: config.TestFile, EnableDryRun: true, DataDir: t.TempDir(), EnableNTPClient: true}, nil)
	require.NoError(t, err, "engine must load its configuration")
	address, _ := ntpTestServer(t, nil)
	bot.Config.NTPClient = config.NTPClientConfig{Level: 1, Pool: []string{address}}
	bot.Config.CheckNTPConfig()
	// Force startup to fail after NTP worker is registered and launched.
	// Verify failed startup joins worker without normal engine Stop call.
	bot.Settings.Exchanges = "not-a-real-exchange"
	require.Error(t, bot.Start(), "invalid exchange must fail engine startup")
	require.NotNil(t, bot.ntpManager, "NTP must have been registered before the failure")
	assert.False(t, bot.ntpManager.IsRunning(), "failed boot should return only after the NTP worker exits")
}
