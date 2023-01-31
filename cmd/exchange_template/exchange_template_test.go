package main

import (
	"os"
	"path/filepath"
	"testing"

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
		if r := checkExchangeName(tester[x].Name); r != tester[x].ErrExpected {
			t.Errorf("test: %d unexpected result", x)
		}
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
	if err != nil {
		t.Fatal(err)
	}

	cfgData, err := os.ReadFile(exchangeConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = saveConfig(testExchangeDir, cfg, exchCfg); err != nil {
		t.Error(err)
	}
	if err = os.WriteFile(exchangeConfigPath, cfgData, file.DefaultPermissionOctal); err != nil {
		t.Error(err)
	}
}
