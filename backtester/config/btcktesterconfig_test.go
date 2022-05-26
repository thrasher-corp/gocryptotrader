package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/file"
)

func TestLoadBacktesterConfig(t *testing.T) {
	t.Parallel()
	cfg, err := GenerateDefaultConfig()
	if err != nil {
		t.Error(err)
	}
	testConfig, err := json.Marshal(cfg)
	if err != nil {
		t.Error(err)
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "test.config")
	err = file.Write(f, testConfig)
	if err != nil {
		t.Error(err)
	}
	_, err = ReadBacktesterConfigFromPath(f)
	if err != nil {
		t.Error(err)
	}
}

func TestGenerateDefaultConfig(t *testing.T) {
	t.Parallel()
	cfg, err := GenerateDefaultConfig()
	if err != nil {
		t.Error(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	if cfg.SingleRunStrategyConfig != filepath.Join(wd, "config", "examples", "ftx-cash-carry.strat") {
		t.Error("Wrong default SingleRunStrategyConfig")
	}
}
