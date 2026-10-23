package exchange

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/goose"
)

var (
	verbose = false

	testExchanges = []Details{
		{
			Name: "one",
		},
		{
			Name: "two",
		},
		{
			Name: "three",
		},
	}
)

//nolint:forbidigo // TestMain reports setup and teardown failures before or after a *testing.T exists
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
		fmt.Printf("failed to create temp file: %v", err)
		os.Exit(1)
	}

	t := m.Run()

	err = os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	os.Exit(t)
}

func TestInsertMany(t *testing.T) {
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
			seedDB: seed,
		},
		{
			name: "SQLite",
			config: &database.Config{
				Driver:   database.DBSQLite3,
				Database: "./testdb",
			},
			seedDB: seed,
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

			require.NoError(t, InsertMany(testExchanges))

			err = testhelpers.CloseDatabase(dbConn)
			assert.NoError(t, err)
		})
	}
}

func TestOneAndOneByUUID(t *testing.T) {
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
			seedDB: seed,
		},
		{
			name: "SQLite",
			config: &database.Config{
				Driver:   database.DBSQLite3,
				Database: "./testdb",
			},
			seedDB: seed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)

			if tc.seedDB != nil {
				require.NoError(t, tc.seedDB())
			}

			ret, err := One("one")
			require.NoError(t, err)

			ret2, err := OneByUUID(ret.UUID)
			require.NoError(t, err)

			assert.Equal(t, ret.Name, ret2.Name)
			assert.NoError(t, testhelpers.CloseDatabase(dbConn))
		})
	}
}

func seed() error {
	return InsertMany(testExchanges)
}

func TestLoadCSV(t *testing.T) {
	testData := filepath.Join("..", "..", "..", "testdata", "exchangelist.csv")
	if _, err := LoadCSV(testData); err != nil {
		t.Fatal(err)
	}
}

// TestRenameGateIOMigration runs the rename migration on SQLite, from the schema the migration before it leaves, and
// checks which exchange rows remain, by name and ID, after migrating up and then down
func TestRenameGateIOMigration(t *testing.T) {
	// Each row an exchange's history keeps in another table, as a statement taking the exchange's ID
	history := map[string]string{
		"candle":           `INSERT INTO candle (id, exchange_name_id, Base, Quote, Interval, Timestamp, Open, High, Low, Close, Volume, Asset) VALUES ('candle-%[1]s', '%[1]s', 'BTC', 'USDT', '60', '2026-10-23 00:00:00', 1, 1, 1, 1, 1, 'spot')`,
		"trade":            `INSERT INTO trade (id, exchange_name_id, tid, base, quote, asset, price, amount, timestamp) VALUES ('trade-%[1]s', '%[1]s', '%[1]s', 'BTC', 'USDT', 'spot', 1, 1, '2026-10-23 00:00:00')`,
		"withdrawal":       `INSERT INTO withdrawal_history (id, exchange_name_id, exchange_id, status, currency, amount, withdraw_type) VALUES ('withdrawal-%[1]s', '%[1]s', 'w1', 'done', 'BTC', 1, 0)`,
		"data history job": `INSERT INTO datahistoryjob (id, nickname, exchange_name_id, asset, base, quote, start_time, end_time, interval, data_type, request_size, max_retries, batch_count, status) VALUES ('job-%[1]s', 'job-%[1]s', '%[1]s', 'spot', 'BTC', 'USDT', '2026-10-01', '2026-10-02', 60, 0, 1, 1, 1, 0)`,
		"conversion job":   `INSERT INTO datahistoryjob (id, nickname, exchange_name_id, secondary_exchange_id, asset, base, quote, start_time, end_time, interval, data_type, request_size, max_retries, batch_count, status) VALUES ('conversion-%[1]s', 'conversion-%[1]s', 'kraken-id', '%[1]s', 'spot', 'BTC', 'USDT', '2026-10-01', '2026-10-02', 60, 0, 1, 1, 1, 0)`,
	}
	const (
		gateIOAndKraken  = `INSERT INTO exchange (id, name) VALUES ('gateio-id', 'gateio'), ('kraken-id', 'kraken')`
		gateIOGateKraken = `INSERT INTO exchange (id, name) VALUES ('gateio-id', 'gateio'), ('gate-id', 'gate'), ('kraken-id', 'kraken')`
	)
	type scenario struct {
		name       string
		seed       []string // Run before migrating up
		beforeDown []string // Run between migrating up and down
		up, down   map[string]string
	}
	run := func(t *testing.T, tc scenario) {
		t.Helper()
		db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "rename.db"))
		require.NoError(t, err, "Open must not error")
		t.Cleanup(func() { assert.NoError(t, db.Close(), "Close should not error") })
		exec := func(statements []string) {
			t.Helper()
			for _, statement := range statements {
				_, err := db.ExecContext(t.Context(), statement)
				require.NoErrorf(t, err, "Exec must not error for %s", statement)
			}
		}
		exchanges := func() map[string]string {
			t.Helper()
			rows, err := db.QueryContext(t.Context(), `SELECT name, id FROM exchange`)
			require.NoError(t, err, "Query must not error")
			defer rows.Close()
			out := make(map[string]string)
			for rows.Next() {
				var name, id string
				require.NoError(t, rows.Scan(&name, &id), "Scan must not error")
				out[name] = id
			}
			require.NoError(t, rows.Err(), "rows must not error")
			return out
		}
		dump := func(query string) []string {
			t.Helper()
			rows, err := db.QueryContext(t.Context(), query)
			require.NoErrorf(t, err, "Query must not error for %s", query)
			defer rows.Close()
			columns, err := rows.Columns()
			require.NoError(t, err, "Columns must not error")
			var out []string
			for rows.Next() {
				values := make([]any, len(columns))
				pointers := make([]any, len(columns))
				for i := range values {
					pointers[i] = &values[i]
				}
				require.NoError(t, rows.Scan(pointers...), "Scan must not error")
				out = append(out, fmt.Sprint(values...))
			}
			require.NoError(t, rows.Err(), "rows must not error")
			return out
		}
		// The rename must move no history between exchanges and change none of it
		children := func() [][]string {
			t.Helper()
			return [][]string{
				dump(`SELECT * FROM candle ORDER BY id`),
				dump(`SELECT * FROM trade ORDER BY id`),
				dump(`SELECT * FROM withdrawal_history ORDER BY id`),
				dump(`SELECT * FROM datahistoryjob ORDER BY id`),
			}
		}

		require.NoError(t, goose.Run("up-to", db, database.DBSQLite3, testhelpers.MigrationDir, "20210709133512"), "up-to must not error")
		exec(tc.seed)
		before := children()
		require.NoError(t, goose.Run("up", db, database.DBSQLite3, testhelpers.MigrationDir), "up must not error")
		assert.Equal(t, tc.up, exchanges(), "up should leave the expected exchange rows")
		assert.Equal(t, before, children(), "up should leave every exchange's history unchanged")
		exec(tc.beforeDown)
		require.NoError(t, goose.Run("down", db, database.DBSQLite3, testhelpers.MigrationDir), "down must not error")
		assert.Equal(t, tc.down, exchanges(), "down should leave the expected exchange rows")
		assert.Equal(t, before, children(), "down should leave every exchange's history unchanged")
	}
	for _, tc := range []scenario{
		{
			name: "renames gateio with its history",
			seed: []string{
				gateIOAndKraken, fmt.Sprintf(history["candle"], "gateio-id"), fmt.Sprintf(history["trade"], "gateio-id"),
				fmt.Sprintf(history["withdrawal"], "gateio-id"), fmt.Sprintf(history["data history job"], "gateio-id"),
			},
			up:   map[string]string{"gate": "gateio-id", "kraken": "kraken-id"},
			down: map[string]string{"gateio": "gateio-id", "kraken": "kraken-id"},
		},
		{
			// Seeding the new exchange list before migrating leaves an empty gate row
			name: "drops an empty gate row for the rename",
			seed: []string{gateIOGateKraken, fmt.Sprintf(history["trade"], "gateio-id")},
			up:   map[string]string{"gate": "gateio-id", "kraken": "kraken-id"},
			down: map[string]string{"gateio": "gateio-id", "kraken": "kraken-id"},
		},
		{
			name:       "drops an empty gateio row when downgrading",
			seed:       []string{gateIOAndKraken, fmt.Sprintf(history["trade"], "gateio-id")},
			beforeDown: []string{`INSERT INTO exchange (id, name) VALUES ('stale-id', 'gateio')`},
			up:         map[string]string{"gate": "gateio-id", "kraken": "kraken-id"},
			down:       map[string]string{"gateio": "gateio-id", "kraken": "kraken-id"},
		},
		{
			// A database seeded only after the rename has nothing to rename and must keep its gate row
			name: "keeps a gate row when there is no gateio row",
			seed: []string{`INSERT INTO exchange (id, name) VALUES ('gate-id', 'gate'), ('kraken-id', 'kraken')`},
			up:   map[string]string{"gate": "gate-id", "kraken": "kraken-id"},
			down: map[string]string{"gateio": "gate-id", "kraken": "kraken-id"},
		},
		{
			name:       "keeps a gateio row when downgrading without a gate row",
			seed:       []string{`INSERT INTO exchange (id, name) VALUES ('kraken-id', 'kraken')`},
			beforeDown: []string{`INSERT INTO exchange (id, name) VALUES ('gateio-id', 'gateio')`},
			up:         map[string]string{"kraken": "kraken-id"},
			down:       map[string]string{"gateio": "gateio-id", "kraken": "kraken-id"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) { run(t, tc) })
	}
	for kind, insert := range history {
		// Merging two exchanges' history could duplicate it, so a row holding any is never dropped
		t.Run("keeps both rows when each holds a "+kind, func(t *testing.T) {
			run(t, scenario{
				seed: []string{gateIOGateKraken, fmt.Sprintf(insert, "gateio-id"), fmt.Sprintf(insert, "gate-id")},
				up:   map[string]string{"gateio": "gateio-id", "gate": "gate-id", "kraken": "kraken-id"},
				down: map[string]string{"gateio": "gateio-id", "gate": "gate-id", "kraken": "kraken-id"},
			})
		})
	}
}
