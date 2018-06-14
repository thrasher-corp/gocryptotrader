package database

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var (
	o         *ORM
	connected bool
	cfg       *config.Config
)

func TestStartDB(t *testing.T) {
	cfg = config.GetConfig()
	err := cfg.LoadConfig("../testdata/configtest.json")
	if err != nil {
		t.Fatal(err)
	}

	o, err = Connect("gocryptotrader", "localhost", "gocryptotrader", "gocryptotrader", false, cfg)
	if err != nil {
		t.Fatal("test failed - Database Connect() error", err)
	}
	connected = true
}

func TestCheckLoadedConfiguration(t *testing.T) {
	if connected {
		b := o.checkLoadedConfiguration(cfg.Name)
		if !b {
			t.Error("test failed - Database checkLoadedConfiguration() error")
		}
	}
}

func TestGetLoadedConfigurationID(t *testing.T) {
	if connected {
		i, err := o.getLoadedConfigurationID(cfg.Name)
		if err != nil {
			t.Error("test failed - Database getLoadedConfigurationID() error", err)
		}
		if i != 0 {
			t.Error("test failed - Database getLoadedConfigurationID() error")
		}
	}
}

func TestDatabaseFlush(t *testing.T) {
	if connected {
		err := o.DatabaseFlush()
		if err != nil {
			t.Error("test failed - Database DatabaseFlush() error", err)
		}
	}
}
