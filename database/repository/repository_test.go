package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/database"
)

func TestGetSQLDialect(t *testing.T) {
	for _, tc := range []struct {
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
	} {
		t.Run(tc.driver, func(t *testing.T) {
			cfg := &database.Config{
				Driver: tc.driver,
			}
			require.NoError(t, database.DB.SetConfig(cfg))
			assert.Equal(t, tc.expectedReturn, GetSQLDialect())
		})
	}
}
