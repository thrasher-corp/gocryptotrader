package config

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestNTPConfigRoundTrip(t *testing.T) {
	t.Parallel()
	latestConfig, err := versions.Manager.Deploy(t.Context(), []byte(`{}`), versions.UseLatestVersion)
	require.NoError(t, err, "latest config version must resolve")
	var latest struct {
		Version int `json:"version"`
	}
	require.NoError(t, json.Unmarshal(latestConfig, &latest), "latest config version must decode")
	for _, level := range []int{-1, 0, 1} {
		for _, pool := range [][]string{
			{},
			{"pool.ntp.org:123"},
			{" POOL.NTP.ORG.:00123 "},
			{"private.invalid:123"},
			{"pool.ntp.org:123", "private.invalid:123"},
		} {
			tolerance := 75 * time.Millisecond
			saved := Config{Version: 15, EncryptConfig: -1, Name: "NTP configuration", NTPClient: NTPClientConfig{
				Level: level, Pool: pool, AllowedDifference: &tolerance, AllowedNegativeDifference: &tolerance,
			}}
			path := filepath.Join(t.TempDir(), "config.json")
			require.NoError(t, saved.SaveConfigToFile(path), "config must save")
			var loaded Config
			require.NoError(t, loaded.ReadConfigFromFile(path, true), "real loader must read the config")
			loaded.CheckNTPConfig()
			assert.Equal(t, latest.Version, loaded.Version, "loading should upgrade the configuration to the latest version")
			assert.Equal(t, saved.NTPClient, loaded.NTPClient, "loading should preserve explicit servers, empty defaults, mode and tolerances")
			require.NoError(t, loaded.SaveConfigToFile(path), "loaded config must save")
			var reloaded Config
			require.NoError(t, reloaded.ReadConfigFromFile(path, true), "saved config must reload")
			assert.Equal(t, saved.NTPClient, reloaded.NTPClient, "save and reload should preserve NTP settings without expanding defaults")
		}
	}
}

func TestExampleConfigLoads(t *testing.T) {
	t.Parallel()
	var c Config
	require.NoError(t, c.ReadConfigFromFile(filepath.Join("..", "config_example.json"), true), "example config must load through real loader")
	assert.Empty(t, c.NTPClient.Pool, "example config should follow maintained NTP defaults")
}
