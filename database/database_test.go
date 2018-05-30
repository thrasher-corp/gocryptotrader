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
	var err error
	o, err = NewORMConnection("gocryptotrader", "localhost", "gocryptotrader", "gocryptotrader", false)
	if err != nil {
		t.Fatal("test failed - Database NewORMConnection() error", err)
	}
	connected = true
	cfg = config.GetConfig()
	cfg.LoadConfig("../testdata/configtest.json")
}

func TestLoadConfiguration(t *testing.T) {
	if connected {
		if err := o.LoadConfiguration("default"); err == nil {
			t.Error("test failed - Database LoadConfiguration() error", err)
		}
	}
}

func TestInsertNewConfiguration(t *testing.T) {
	if connected {
		err := o.InsertNewConfiguration(cfg, "newPassword")
		if err != nil {
			t.Error("test failed - Database InsertNewConfiguration() error", err)
		}
	}
}

func TestUpdateConfiguration(t *testing.T) {
	if connected {
		err := o.UpdateConfiguration(cfg)
		if err != nil {
			t.Error("test failed - Database UpdateConfiguration() error", err)
		}
	}
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
