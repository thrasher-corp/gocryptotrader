package seed

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
)

var (
	verbose = false
)

func TestMain(m *testing.M) {
	if verbose {
		testhelpers.EnableVerboseTestOutput()
	}

	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = ioutil.TempDir("", "gct-temp")
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

func TestRun(t *testing.T) {

}

func TestExchange(t *testing.T) {

}
