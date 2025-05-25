package archive

import (
	"archive/zip"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	outFile := filepath.Join(tempDir, "out.zip")
	err := Zip(filepath.Join("..", "..", "..", "testdata", "configtest.json"), outFile)
	require.NoError(t, err, "Zip must not error")
	o, err := UnZip(outFile, tempDir)
	require.NoError(t, err, "UnZip must not error")
	assert.Len(t, o, 1, "Should extract 1 file")

	folder := filepath.Join("..", "..", "..", "testdata", "gctscript")
	outFolderZip := filepath.Join(tempDir, "out_folder.zip")
	err = Zip(folder, outFolderZip)
	require.NoError(t, err, "Zip must not error")
	o, err = UnZip(outFolderZip, tempDir)
	require.NoError(t, err, "UnZip must not error")
	var found bool
	for i := range o {
		if filepath.Base(o[i]) == "timer.gct" {
			found = true
		}
	}
	assert.True(t, found, "Should find a gctscript in the zip")
	assert.GreaterOrEqual(t, len(o), 6, "Should extract at least 6 files")

	folder = filepath.Join("..", "..", "..", "testdata", "invalid_file.json")
	err = Zip(folder, filepath.Join(tempDir, "invalid.zip"))
	assert.ErrorIs(t, err, fs.ErrNotExist, "Zip should error correctly")

	addFilesToZip = addFilesToZipTestWrapper
	folder = filepath.Join("..", "..", "..", "testdata", "http_mock")
	outFolderZip = filepath.Join(tempDir, "error_zip.zip")
	err = Zip(folder, outFolderZip)
	assert.ErrorContains(t, err, "specific error", "Zip should error correctly")
}

func addFilesToZipTestWrapper(_ *zip.Writer, _ string, _ bool) error {
	return errors.New("specific error")
}
