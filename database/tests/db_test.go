package tests

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/thrasher-/gocryptotrader/database"
	"github.com/thrasher-/gocryptotrader/database/drivers"
	db "github.com/thrasher-/gocryptotrader/database/drivers/sqlite"
)

var (
	tempDir string
	trueptr = func(b bool) *bool { return &b }(true)
)

func TestMain(m *testing.M) {
	var err error
	tempDir, err = ioutil.TempDir("", "gct-temp")

	if err != nil {
		fmt.Printf("failed to create temp file: %v", err)
		os.Exit(1)
	}

	t := m.Run()

	err = os.RemoveAll(tempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	os.Exit(t)
}

func TestDatabase(t *testing.T) {
	testCases := []struct {
		name    string
		config  database.Config
		output  interface{}
		cleanup bool
	}{
		{
			"Connect",
			database.Config{
				ConnectionDetails: drivers.ConnectionDetails{Database: path.Join(tempDir, "./testdb.db")},
			},
			nil,
			true,
		},
		{
			"NoDatabase",
			database.Config{
				Enabled: trueptr,
				Driver:  "sqlite",
			},
			errors.New("no database provided"),
			false,
		},
	}

	for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {
			database.Conn.Config = &test.config
			dbConn, err := db.Connect()

			switch v := test.output.(type) {

			case error:
				if v.Error() != test.output.(error).Error() {
					t.Fatal(err)
				}
			default:
				return
			}
			if test.cleanup {
				err = dbConn.SQL.Close()
				if err != nil {
					t.Error("Failed to close database")
				}
			}
		})
	}
}
