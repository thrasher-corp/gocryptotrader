package datahistoryjob

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		err := testhelpers.EnableVerboseTestOutput()
		if err != nil {
			fmt.Printf("failed to enable verbose test output: %v", err)
			os.Exit(1)
		}
	}
	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = os.MkdirTemp("", "gct-temp")
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
	t.Parallel()
	for _, tc := range []struct {
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)

			if tc.seedDB != nil {
				require.NoError(t, tc.seedDB())
			}

			db, err := Setup(dbConn)
			require.NoError(t, err)

			var jerberinos, jerberoos []*DataHistoryJob
			for i := range 20 {
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
			require.NoError(t, err)

			// insert the same jerbs to test conflict resolution
			for i := range 20 {
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
			require.NoError(t, err)

			_, err = db.GetJobsBetween(time.Now(), time.Now().Add(time.Hour))
			require.NoError(t, err)

			resp, err := db.GetByNickName("TestDataHistoryJob19")
			require.NoError(t, err)
			assert.True(t, strings.EqualFold("TestDataHistoryJob19", resp.Nickname))

			results, err := db.GetAllIncompleteJobsAndResults()
			require.NoError(t, err)
			assert.Len(t, results, 19)

			jerb, err := db.GetJobAndAllResults(jerberoos[0].Nickname)
			require.NoError(t, err)
			assert.True(t, strings.EqualFold(jerberoos[0].Nickname, jerb.Nickname))

			results, err = db.GetJobsBetween(time.Now().Add(-time.Hour), time.Now())
			require.NoError(t, err)
			require.Len(t, results, 20)

			jerb, err = db.GetJobAndAllResults(results[0].Nickname)
			require.NoError(t, err)

			assert.Equal(t, jerb.Nickname, results[0].Nickname)

			err = db.SetRelationshipByID(results[0].ID, results[1].ID, 1337)
			require.NoError(t, err)

			jerb, err = db.GetByID(results[1].ID)
			require.NoError(t, err)
			assert.Equal(t, int64(1337), jerb.Status)

			rel, err := db.GetRelatedUpcomingJobs(results[0].Nickname)
			require.NoError(t, err)
			require.Len(t, rel, 1)
			assert.Equal(t, rel[0].ID, results[1].ID)

			err = db.SetRelationshipByID(results[0].ID, results[2].ID, 1337)
			assert.NoError(t, err)

			rel, err = db.GetRelatedUpcomingJobs(results[0].Nickname)
			require.NoError(t, err)
			require.Len(t, rel, 2)
			expectedIDs := []string{results[1].ID, results[2].ID}
			actualIDs := []string{rel[0].ID, rel[1].ID}
			assert.ElementsMatch(t, expectedIDs, actualIDs)

			jerb, err = db.GetPrerequisiteJob(results[1].Nickname)
			require.NoError(t, err)

			assert.Equal(t, jerb.ID, results[0].ID)

			jerb, err = db.GetPrerequisiteJob(results[2].Nickname)
			require.NoError(t, err)

			assert.Equal(t, jerb.ID, results[0].ID)

			err = db.SetRelationshipByNickname(results[4].Nickname, results[2].Nickname, 0)
			require.NoError(t, err)

			err = db.SetRelationshipByNickname(results[2].Nickname, results[2].Nickname, 0)
			assert.ErrorIs(t, err, errCannotSetSamePrerequisite)

			err = db.SetRelationshipByNickname(results[3].Nickname, results[2].Nickname, 0)
			assert.NoError(t, err)

			// ensure only one prerequisite can be associated at once
			// after setting the prerequisite twice
			rel, err = db.GetRelatedUpcomingJobs(results[4].Nickname)
			require.NoError(t, err)
			assert.Empty(t, rel)

			rel, err = db.GetRelatedUpcomingJobs(results[3].Nickname)
			require.NoError(t, err)
			assert.Len(t, rel, 1)

			err = testhelpers.CloseDatabase(dbConn)
			assert.NoError(t, err)
		})
	}
}
