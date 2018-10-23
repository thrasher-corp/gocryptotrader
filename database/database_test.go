package database

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/database/models"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var (
	testWorkingDir, _ = os.Getwd()
)

var (
	testConn  *ORM
	connected bool
	cfg       *config.Config
)

func TestStartDB(t *testing.T) {
	cfg = config.GetConfig()
	err := cfg.LoadConfig("../testdata/configtest.json")
	if err != nil {
		t.Fatal(err)
	}

	var pathToDB string
	if common.StringContains(testWorkingDir, "database") {
		pathToDB = testWorkingDir + "/database.db"
	} else {
		pathToDB = testWorkingDir + "/database/database.db"
	}

	testConn, err = Connect(pathToDB, true, cfg)
	if err != nil {
		log.Println("WARNING - NO DATABASE CONNECTION!", err)
	} else {
		connected = true
	}
}

func TestLoadConfigurations(t *testing.T) {
	if !connected {
		t.Skip()
	}

	err := testConn.LoadConfigurations()
	if err != nil {
		t.Fatal("test failed - LoadConfiguration error", err)
	}

	// forces update logic
	err = testConn.LoadConfigurations()
	if err != nil {
		t.Fatal("test failed - LoadConfiguration error", err)
	}
}

func TestPurgeDB(t *testing.T) {
	if !connected {
		t.Skip()
	}

	_, err := models.ExchangeConfigs(qm.Where("config_id = ?",
		testConn.ConfigID)).DeleteAll(ctx, testConn.DB)
	if err != nil {
		t.Error("test failed - purging test exchange config data", err)
	}

	_, err = models.Configs(qm.Where("id = ?",
		testConn.ConfigID)).DeleteAll(ctx, testConn.DB)
	if err != nil {
		t.Error("test failed - purging test config data", err)
	}
}
