package extract

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var (
	tempDir  string
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

func TestUnzip(t *testing.T) {
	zipFile := filepath.Join("..","..","testdata","gctscript","archived.zip")
	files, err := Unzip(zipFile, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(files)
}