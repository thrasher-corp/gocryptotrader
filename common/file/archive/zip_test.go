package archive

import (
	"archive/zip"
	"errors"
	"path/filepath"
	"testing"
)

func TestUnZip(t *testing.T) {
	tempDir := t.TempDir()
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
	tempDir := t.TempDir()
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
	var found bool
	for i := range o {
		if filepath.Base(o[i]) == "binance.json" {
			found = true
		}
	}
	if !found {
		t.Fatal("could not find file in zip")
	}

	if expected := 6; len(o) < expected {
		t.Fatalf("expected at least %v files to be extracted, received: %v ", expected, len(o))
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
