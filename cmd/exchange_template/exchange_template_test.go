package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestCheckExchangeName(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name        string
		ErrExpected error
	}{
		{
			Name:        "test exch",
			ErrExpected: errInvalidExchangeName,
		},
		{
			ErrExpected: errInvalidExchangeName,
		},
		{
			Name:        " ",
			ErrExpected: errInvalidExchangeName,
		},
		{
			Name:        "m",
			ErrExpected: errInvalidExchangeName,
		},
		{
			Name:        "mu",
			ErrExpected: errInvalidExchangeName,
		},
		{
			Name: "testexch",
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			err := checkExchangeName(tt.Name)
			if tt.ErrExpected == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tt.ErrExpected, err)
			}
		})
	}
}

func TestNewExchangeAndSaveConfig(t *testing.T) {
	const testExchangeName = "testexch"
	testExchangeDir := filepath.Join(targetPath, testExchangeName)
	cfg := config.GetConfig()

	t.Cleanup(func() {
		if err := os.RemoveAll(testExchangeDir); err != nil {
			t.Errorf("RemoveAll failed: %s, manual deletion of test directory required", err)
		}
	})

	_, err := makeExchange(
		testExchangeDir,
		cfg,
		&exchange{
			Name: testExchangeName,
			REST: true,
			WS:   true,
		},
	)
	assert.NoError(t, err)

	err = os.RemoveAll(testExchangeDir)
	require.NoErrorf(t, err, "RemoveAll failed: %s, manual deletion of test directory required", err)

	exchCfg, err := makeExchange(
		testExchangeDir,
		cfg,
		&exchange{
			Name: testExchangeName,
			REST: true,
			WS:   false,
		},
	)
	require.NoError(t, err)

	cfgData, err := os.ReadFile(exchangeConfigPath)
	require.NoError(t, err, "os.ReadFile must not error")

	err = saveConfig(testExchangeDir, cfg, exchCfg)
	require.NoError(t, err, "saveConfig must not error")

	err = os.WriteFile(exchangeConfigPath, cfgData, file.DefaultPermissionOctal)
	require.NoError(t, err, "os.WriteFile must not error")
}
