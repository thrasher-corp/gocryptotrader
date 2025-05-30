package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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

	_, err = ReadBacktesterConfigFromPath("test")
	assert.ErrorIs(t, err, common.ErrFileNotFound)
}

func TestGenerateDefaultConfig(t *testing.T) {
	t.Parallel()
	cfg, err := GenerateDefaultConfig()
	if err != nil {
		t.Error(err)
	}
	if !cfg.PrintLogo {
		t.Errorf("received '%v' expected '%v'", cfg.PrintLogo, true)
	}
}
