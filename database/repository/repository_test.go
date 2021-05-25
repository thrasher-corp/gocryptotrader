package repository

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/database"
)

func TestGetSQLDialect(t *testing.T) {
	testCases := []struct {
		driver         string
		expectedReturn string
	}{
		{
			"postgresql",
			database.DBPostgreSQL,
		},
		{
			"sqlite",
			database.DBSQLite3,
		},
		{
			"sqlite3",
			database.DBSQLite3,
		},
		{
			"invalid",
			database.DBInvalidDriver,
		},
	}
	for x := range testCases {
		test := testCases[x]

		t.Run(test.driver, func(t *testing.T) {
			cfg := &database.Config{
				Driver: test.driver,
			}
			err := database.DB.SetConfig(cfg)
			if err != nil {
				t.Error(err)
			}
			ret := GetSQLDialect()
			if ret != test.expectedReturn {
				t.Fatalf("unexpected return: %v", ret)
			}
		})
	}
}
