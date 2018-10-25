package database

import (
	"os"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/database/models"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var db *ORM

func TestSetup(t *testing.T) {
	err := Setup("./")
	if err != nil {
		t.Fatal("test failed - Setup error", err)
	}
}

func TestStartDB(t *testing.T) {
	_, err := common.ReadFile("./testdatabase.db")
	if err == nil {
		err = os.Remove("./testdatabase.db")
		if err != nil {
			t.Error("test failed - TestStartDB file failed to delete")
		}
	}

	cfg := config.GetConfig()
	err = cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Error(err)
	}

	db, err = Connect("./testdatabase.db", true, cfg)
	if err != nil {
		t.Error("test failed - TestStartDB failed to connect", err)
	}
}

func TestLoadConfigurations(t *testing.T) {
	err := db.LoadConfigurations()
	if err != nil {
		t.Error("test failed - LoadConfiguration error", err)
	}

	// forces update logic
	err = db.LoadConfigurations()
	if err != nil {
		t.Error("test failed - LoadConfiguration error", err)
	}
}

func TestInsertDeleteTradeHistoryData(t *testing.T) {
	insertedTime := time.Now()
	err := db.InsertExchangeTradeHistoryData(1337,
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

	if id != 1337 {
		t.Errorf("test failed - expected 1337 recieved %d time failure error",
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

	if fullHistory[0].TID != 1337 {
		t.Errorf("test failed - expected 1337 recieved %d error",
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

func TestPurgeDB(t *testing.T) {
	_, err := models.ExchangeConfigs(qm.Where("config_id = ?",
		db.ConfigID)).DeleteAll(ctx, db.DB)
	if err != nil {
		t.Error("test failed - purging test exchange config data", err)
	}

	_, err = models.Configs(qm.Where("id = ?",
		db.ConfigID)).DeleteAll(ctx, db.DB)
	if err != nil {
		t.Error("test failed - purging test config data", err)
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

	if err := os.Remove("./schema.sql"); err != nil {
		t.Error("test failed - Disconnect() file failed to delete", err)
	}

	if err := os.Remove("./testdatabase.db"); err != nil {
		t.Error("test failed - Disconnect() file failed to delete", err)
	}
}
