package database

import (
	"os"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	TESTDIR      = "./"
	TESTDBFILE   = "./testdatabase.db"
	TESTCONFNAME = "TEST"
)

var db *ORM

func TestSetup(t *testing.T) {
	err := Setup(TESTDIR, true)
	if err != nil {
		t.Fatal("test failed - Setup error", err)
	}
}

func TestStartDB(t *testing.T) {
	_, err := common.ReadFile(TESTDBFILE)
	if err == nil {
		err = os.Remove(TESTDBFILE)
		if err != nil {
			t.Fatal("test failed - TestStartDB file failed to delete")
		}
	}

	cfg := config.GetConfig()
	err = cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatal(err)
	}

	cfg.Name = TESTCONFNAME

	db, err = Connect(TESTDBFILE, true)
	if err != nil {
		t.Fatal("test failed - TestStartDB failed to connect", err)
	}
}

func TestInsertUser(t *testing.T) {
	err := db.insertUser("testuser", []byte("testpassword"))
	if err != nil {
		t.Fatal("test failed - InsertUser() error", err)
	}
}

func TestGetUser(t *testing.T) {
	_, err := db.getUser("testuser")
	if err != nil {
		t.Fatal("test failed - GetUser() error", err)
	}
}

func TestSetUserID(t *testing.T) {
	err := db.SetSessionData("testuser", []byte("testpw"))
	if err != nil {
		t.Fatal("test failed - setUserID() error", err)
	}
}

func TestGetConfig(t *testing.T) {
	cfg, err := db.GetConfig(TESTCONFNAME, config.ConfigTestFile, true, true)
	if err != nil {
		t.Fatal("test failed - GetConfig() error", err)
	}

	if cfg.Name != TESTCONFNAME {
		t.Error("test failed - GetConfig() error - name mismatch")
	}

	cfg, err = db.GetConfig(TESTCONFNAME, "", true, true)
	if err == nil {
		t.Fatal("test failed - GetConfig() error, configuration path nil")
	}

	cfg, err = db.GetConfig("", config.ConfigTestFile, true, true)
	if err != nil {
		t.Fatal("test failed - GetConfig() error, configuration name nil", err)
	}

	cfg, err = db.GetConfig(TESTCONFNAME, "", false, true)
	if err == nil {
		t.Fatal("test failed - GetConfig() error, configuration path nil, override off")
	}

	cfg, err = db.GetConfig("", config.ConfigTestFile, false, true)
	if err == nil {
		t.Fatal("test failed - GetConfig() error, configuration name nil, overide off")
	}

	cfg, err = db.GetConfig(TESTCONFNAME, "", false, false)
	if err != nil {
		t.Fatal("test failed - GetConfig() error, configuration path nil, override off, saveconfig off")
	}

	cfg, err = db.GetConfig("", config.ConfigTestFile, false, false)
	if err == nil {
		t.Fatal("test failed - GetConfig() error, configuration name nil, override off, saveconfig off")
	}

	cfg, err = db.GetConfig(TESTCONFNAME, "", true, false)
	if err == nil {
		t.Fatal("test failed - GetConfig() error, configuration path nil, override on, saveconfig off")
	}

	cfg, err = db.GetConfig("", config.ConfigTestFile, true, false)
	if err != nil {
		t.Fatal("test failed - GetConfig() error, configuration name nil, override on, saveconfig off")
	}
}

func TestGetSavedConfiguration(t *testing.T) {
	_, err := db.getSavedConfiguration(TESTCONFNAME)
	if err != nil {
		t.Fatal("test failed - GetSavedConfiguration error", err)
	}
}

func TestSaveConfiguration(t *testing.T) {
	var cfg = config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatal("test failed - SaveConfiguration() error", err)
	}
	cfg.Name = "SavedConfigOne"
	err = db.saveConfiguration(cfg)
	if err != nil {
		t.Fatal("test failed - SaveConfiguration error", err)
	}

	retrievedCfg, err := db.getSavedConfiguration("SavedConfigOne")
	if err != nil {
		t.Fatal("test failed - SaveConfiguration error", err)
	}

	if retrievedCfg.Name != "SavedConfigOne" {
		t.Fatal("test failed -SaveConfiguration error")
	}

	if len(retrievedCfg.GetEnabledExchanges()) != len(cfg.GetEnabledExchanges()) {
		t.Fatal("test failed - SaveConfiguration error - data mismatch")
	}
}

func TestInsertDeleteTradeHistoryData(t *testing.T) {
	insertedTime := time.Now()
	err := db.InsertExchangeTradeHistoryData("1337",
		"Bitstamp",
		"BTCUSD",
		"SPOT",
		"BUY",
		1000.00,
		100,
		insertedTime)

	if err != nil {
		t.Error("test failed - InsertExchangeTradeHistoryData() error", err)
	}

	getTime, id, err := db.GetExchangeTradeHistoryLast("Bitstamp", "BTCUSD", "SPOT")
	if err != nil {
		t.Error("test failed - GetExchangeTradeHistoryLast() error", err)
	}

	if !insertedTime.Equal(getTime) {
		t.Errorf("test failed - expected %s recieved %s time failure error",
			insertedTime.String(),
			getTime.String())
	}

	if id != "1337" {
		t.Errorf("test failed - expected 1337 recieved %s time failure error",
			id)
	}

	fullHistory, err := db.GetExchangeTradeHistory("Bitstamp", "BTCUSD", "SPOT")
	if err != nil {
		t.Error("test failed - GetExchangeTradeHistory() error", err)
	}

	if len(fullHistory) != 1 {
		t.Error("test failed - GetExchangeTradeHistory() too many entries returned")
	}

	if fullHistory[0].Amount != 1000 {
		t.Errorf("test failed - expected 100 recieved %f error",
			fullHistory[0].Amount)
	}

	if fullHistory[0].Exchange != "Bitstamp" {
		t.Errorf("test failed - expected testExchange recieved %s error",
			fullHistory[0].Exchange)
	}

	if fullHistory[0].Price != 100 {
		t.Errorf("test failed - expected 100 recieved %f error",
			fullHistory[0].Price)
	}

	if fullHistory[0].TID != "1337" {
		t.Errorf("test failed - expected 1337 recieved %s error",
			fullHistory[0].TID)
	}

	if !fullHistory[0].Timestamp.Equal(insertedTime) {
		t.Errorf("test failed - expected %s recieved %s error",
			insertedTime.String(),
			fullHistory[0].Timestamp.String())
	}

	if fullHistory[0].Type != "BUY" {
		t.Errorf("test failed - expected BUY recieved %s error",
			fullHistory[0].Type)
	}
}

func TestDisconnect(t *testing.T) {
	if err := db.Disconnect(); err != nil {
		t.Error("test failed - Disconnect() file failed to close connection",
			err)
	}

	if err := os.Remove("./sqlboiler.toml"); err != nil {
		t.Error("test failed - Disconnect() file failed to delete", err)
	}

	if err := os.Remove("./db.schema"); err != nil {
		t.Error("test failed - Disconnect() file failed to delete", err)
	}

	if err := os.Remove("./testdatabase.db"); err != nil {
		t.Error("test failed - Disconnect() file failed to delete", err)
	}
}
