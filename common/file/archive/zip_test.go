package archive

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var (
	tempDir string
)

func TestMain(m *testing.M) {
	var err error
	tempDir, err = ioutil.TempDir("", "gct-temp")
	if err != nil {
		fmt.Printf("failed to create tempDir: %v", err)
		os.Exit(1)
	}
	t := m.Run()
	err = os.RemoveAll(tempDir)
	if err != nil {
		fmt.Printf("Failed to remove tempDir %v", err)
	}
	os.Exit(t)
}

func TestUnZip(t *testing.T) {
	zipFile := filepath.Join("..", "..", "..", "testdata", "testdata.zip")
	files, err := UnZip(zipFile, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files to be extracted received: %v ", len(files))
	}

	zipFile = filepath.Join("..", "..", "..", "testdata", "zip-slip.zip")
	_, err = UnZip(zipFile, tempDir)
	if err == nil {
		t.Fatal("Zip() expected to error due to ZipSlip detection but extracted successfully")
	}

	zipFile = filepath.Join("..", "..", "..", "testdata", "configtest.json")
	_, err = UnZip(zipFile, tempDir)
	if err == nil {
		t.Fatal("Zip() expected to error due to invalid zipfile")
	}
}

func TestZip(t *testing.T) {
	singleFile := filepath.Join("..", "..", "..", "testdata", "configtest.json")
	outFile := filepath.Join(tempDir, "out.zip")
	err := Zip(singleFile, outFile)
	if err != nil {
		t.Fatal(err)
	}
	o, err := UnZip(outFile, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(o) != 1 {
		t.Fatalf("expected 1 files to be extracted received: %v ", len(o))
	}

	folder := filepath.Join("..", "..", "..", "testdata", "http_mock")
	outFolderZip := filepath.Join(tempDir, "out_folder.zip")
	err = Zip(folder, outFolderZip)
	if err != nil {
		t.Fatal(err)
	}
	o, err = UnZip(outFolderZip, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(o[0]) != "binance.json" || filepath.Base(o[4]) != "localbitcoins.json" {
		t.Fatal("unexpected archive result received")
	}
	expected := 7
	if len(o) != expected {
		t.Fatalf("expected %v files to be extracted received: %v ", expected, len(o))
	}

	folder = filepath.Join("..", "..", "..", "testdata", "invalid_file.json")
	outFolderZip = filepath.Join(tempDir, "invalid.zip")
	err = Zip(folder, outFolderZip)
	if err == nil {
		t.Fatal("expected IsNotExistError on invalid file")
	}

	addFilesToZip = addFilesToZipTestWrapper
	folder = filepath.Join("..", "..", "..", "testdata", "http_mock")
	outFolderZip = filepath.Join(tempDir, "error_zip.zip")
	err = Zip(folder, outFolderZip)
	if err == nil {
		t.Fatal("expected Zip() to fail due to invalid addFilesToZipTestWrapper()")
	}
}

func addFilesToZipTestWrapper(_ *zip.Writer, _ string, _ bool) error {
	return errors.New("error")
}
