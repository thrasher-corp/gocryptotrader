package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestNewExchange(t *testing.T) {
	testExchangeName := "testexch"
	testExchangeDir := filepath.Join(targetPath, testExchangeName)

	if err := makeExchange(&exchange{
		Name: testExchangeName,
		REST: true,
		WS:   true,
	}); err != nil {
		t.Error(err)
	}

	if err := os.RemoveAll(testExchangeDir); err != nil {
		t.Errorf("unable to remove dir: %s, manual removal required", err)
	}

	cfg := config.GetConfig()
	if err := cfg.LoadConfig(exchangeConfigPath, true); err != nil {
		t.Fatal(err)
	}
	if success := cfg.RemoveExchange(testExchangeName); !success {
		t.Fatalf("unable to remove exchange config for %s, manual removal required\n",
			testExchangeName)
	}
	if err := cfg.SaveConfig(exchangeConfigPath, false); err != nil {
		t.Fatal(err)
	}
}
