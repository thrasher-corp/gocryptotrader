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
	tester := []struct {
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
	}

	for x := range tester {
		r := checkExchangeName(tester[x].Name)
		assert.Equal(t, tester[x].ErrExpected, r)
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

	exchCfg, err := makeExchange(
		testExchangeDir,
		cfg,
		&exchange{
			Name: testExchangeName,
			REST: true,
			WS:   true,
		},
	)
	require.NoError(t, err)

	cfgData, err := os.ReadFile(exchangeConfigPath)
	require.NoError(t, err)

	err = saveConfig(testExchangeDir, cfg, exchCfg)
	assert.NoError(t, err)

	err = os.WriteFile(exchangeConfigPath, cfgData, file.DefaultPermissionOctal)
	assert.NoError(t, err)
}
