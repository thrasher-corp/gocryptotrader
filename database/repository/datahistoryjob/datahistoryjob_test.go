package datahistoryjob

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	verbose       = false
	testExchanges = []exchange.Details{
		{
			Name: "one",
		},
		{
			Name: "two",
		},
	}
)

func TestMain(m *testing.M) {
	if verbose {
		testhelpers.EnableVerboseTestOutput()
	}
	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = ioutil.TempDir("", "gct-temp")
	if err != nil {
		log.Fatal(err)
	}
	t := m.Run()
	err = os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	os.Exit(t)
}

func seedDB() error {
	err := exchange.InsertMany(testExchanges)
	if err != nil {
		return err
	}

	for i := range testExchanges {
		lol, err := exchange.One(testExchanges[i].Name)
		if err != nil {
			return err
		}
		testExchanges[i].UUID = lol.UUID
	}

	return nil
}

func TestDataHistoryJob(t *testing.T) {
	testCases := []struct {
		name   string
		config *database.Config
		seedDB func() error
		runner func(t *testing.T)
		closer func(dbConn *database.Instance) error
	}{
		{
			name:   "postgresql",
			config: testhelpers.PostgresTestDatabase,
			seedDB: seedDB,
		},
		{
			name: "SQLite",
			config: &database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},
			seedDB: seedDB,
		},
	}

	for x := range testCases {
		test := testCases[x]
		t.Run(test.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(test.config)
			if err != nil {
				t.Fatal(err)
			}

			if test.seedDB != nil {
				err = test.seedDB()
				if err != nil {
					t.Error(err)
				}
			}

			db, err := Setup(dbConn)
			if err != nil {
				log.Fatal(err)
			}

			var jerberinos, jerberoos []*DataHistoryJob
			for i := 0; i < 20; i++ {
				uu, _ := uuid.NewV4()
				jerberinos = append(jerberinos, &DataHistoryJob{
					ID:           uu.String(),
					Nickname:     fmt.Sprintf("TestDataHistoryJob%v", i),
					ExchangeID:   testExchanges[0].UUID.String(),
					ExchangeName: testExchanges[0].Name,
					Asset:        asset.Spot.String(),
					Base:         currency.BTC.String(),
					Quote:        currency.USD.String(),
					StartDate:    time.Now().Add(time.Duration(i+1) * time.Second).UTC(),
					EndDate:      time.Now().Add(time.Minute * time.Duration(i+1)).UTC(),
					Interval:     int64(i),
				})
			}
			err = db.Upsert(jerberinos...)
			if err != nil {
				t.Fatal(err)
			}
			// insert the same jerbs to test conflict resolution
			for i := 0; i < 20; i++ {
				uu, _ := uuid.NewV4()
				j := &DataHistoryJob{
					ID:           uu.String(),
					Nickname:     fmt.Sprintf("TestDataHistoryJob%v", i),
					ExchangeID:   testExchanges[0].UUID.String(),
					ExchangeName: testExchanges[0].Name,
					Asset:        asset.Spot.String(),
					Base:         currency.BTC.String(),
					Quote:        currency.USD.String(),
					StartDate:    time.Now().Add(time.Duration(i+1) * time.Second).UTC(),
					EndDate:      time.Now().Add(time.Minute * time.Duration(i+1)).UTC(),
					Interval:     int64(i),
				}
				if i == 19 {
					j.Status = 1
				}
				jerberoos = append(jerberoos, j)
			}
			err = db.Upsert(jerberoos...)
			if err != nil {
				t.Fatal(err)
			}

			_, err = db.GetJobsBetween(time.Now(), time.Now().Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}

			resp, err := db.GetByNickName("TestDataHistoryJob19")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.EqualFold(resp.Nickname, "TestDataHistoryJob19") {
				t.Fatal("the database no longer functions")
			}

			results, err := db.GetAllIncompleteJobsAndResults()
			if err != nil {
				t.Error(err)
			}
			if len(results) != 19 {
				t.Errorf("expected 19, received %v", len(results))
			}

			jerb, err := db.GetJobAndAllResults(jerberoos[0].Nickname)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.EqualFold(jerb.Nickname, jerberoos[0].Nickname) {
				t.Errorf("expected %v, received %v", jerb.Nickname, jerberoos[0].Nickname)
			}

			results, err = db.GetJobsBetween(time.Now().Add(-time.Hour), time.Now())
			if err != nil {
				t.Error(err)
			}
			if len(results) != 20 {
				t.Errorf("expected 20, received %v", len(results))
			}

			jerb, err = db.GetJobAndAllResults(jerberoos[0].Nickname)
			if err != nil {
				t.Error(err)
			}
			if !strings.EqualFold(jerb.Nickname, jerberoos[0].Nickname) {
				t.Errorf("expected %v, received %v", jerb.Nickname, jerberoos[0].Nickname)
			}

			err = testhelpers.CloseDatabase(dbConn)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
