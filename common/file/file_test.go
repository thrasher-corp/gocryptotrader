package file

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	tester := func(in string) error {
		err := Write(in, []byte("GoCryptoTrader"))
		if err != nil {
			return err
		}
		return os.Remove(in)
	}

	type testTable struct {
		InFile      string
		ErrExpected bool
		Cleanup     bool
	}

	var tests []testTable
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "gcttest.txt")
	switch runtime.GOOS {
	case "windows":
		tests = []testTable{
			{InFile: "*", ErrExpected: true},
			{InFile: testFile, ErrExpected: false},
		}
	default:
		tests = []testTable{
			{InFile: "", ErrExpected: true},
			{InFile: testFile, ErrExpected: false},
		}
	}

	for x := range tests {
		err := tester(tests[x].InFile)
		if err != nil && !tests[x].ErrExpected {
			t.Errorf("Test %d failed, unexpected err %s\n", x, err)
		}
	}
}

func TestMove(t *testing.T) {
	tester := func(in, out string, write bool) error {
		if write {
			if err := os.WriteFile(in, []byte("GoCryptoTrader"), DefaultPermissionOctal); err != nil {
				return err
			}
		}

		if err := Move(in, out); err != nil {
			return err
		}

		contents, err := os.ReadFile(out)
		if err != nil {
			return err
		}

		if !strings.Contains(string(contents), "GoCryptoTrader") {
			return errors.New("unable to find previously written data")
		}

		return os.Remove(out)
	}

	type testTable struct {
		InFile      string
		OutFile     string
		Write       bool
		ErrExpected bool
	}

	var tests []testTable
	switch runtime.GOOS {
	case "windows":
		tests = []testTable{
			{InFile: "*", OutFile: "gct.txt", Write: true, ErrExpected: true},
			{InFile: "*", OutFile: "gct.txt", Write: false, ErrExpected: true},
			{InFile: "in.txt", OutFile: "*", Write: true, ErrExpected: true},
		}
	default:
		tests = []testTable{
			{InFile: "", OutFile: "gct.txt", Write: true, ErrExpected: true},
			{InFile: "", OutFile: "gct.txt", Write: false, ErrExpected: true},
			{InFile: "in.txt", OutFile: "", Write: true, ErrExpected: true},
		}
	}
	tests = append(tests, []testTable{
		{InFile: "in.txt", OutFile: "gct.txt", Write: true, ErrExpected: false},
		{InFile: "in.txt", OutFile: "non-existing/gct.txt", Write: true, ErrExpected: false},
		{InFile: "in.txt", OutFile: "in.txt", Write: true, ErrExpected: false},
	}...)

	if Exists("non-existing") {
		t.Error("target 'non-existing' should not exist")
	}
	defer os.RemoveAll("non-existing")
	defer os.Remove("in.txt")

	for x := range tests {
		err := tester(tests[x].InFile, tests[x].OutFile, tests[x].Write)
		if err != nil && !tests[x].ErrExpected {
			t.Errorf("Test %d failed, unexpected err %s\n", x, err)
		}
	}
}

func TestExists(t *testing.T) {
	if e := Exists("non-existent"); e {
		t.Error("non-existent file should not exist")
	}
	tmpFile := filepath.Join(os.TempDir(), "gct-test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello world"), os.ModeAppend); err != nil {
		t.Fatal(err)
	}
	if e := Exists(tmpFile); !e {
		t.Error("file should exist")
	}
	if err := os.Remove(tmpFile); err != nil {
		t.Errorf("unable to remove %s, manual deletion is required", tmpFile)
	}
}

func TestWriteAsCSV(t *testing.T) {
	tester := func(in string, data [][]string) error {
		err := WriteAsCSV(in, data)
		if err != nil {
			return err
		}
		return os.Remove(in)
	}

	type testTable struct {
		InFile      string
		Payload     [][]string
		ErrExpected bool
	}

	records := [][]string{
		{"title", "first_name", "last_name"},
		{"King", "Robert", "Baratheon"},
		{"Lord Regent of the Seven Kingdoms", "Eddard", "Stark"},
		{"Lord of Baelish Castle", "Petyr", "Baelish"},
	}

	missAligned := [][]string{
		{"first_name", "last_name", "username"},
		{"Sup", "bra"},
	}

	testFile, err := os.CreateTemp(os.TempDir(), "gct-csv-test.*.csv")
	if err != nil {
		t.Fatal(err)
	}
	testFile.Close()
	defer os.Remove(testFile.Name())

	tests := []testTable{
		{InFile: testFile.Name(), Payload: nil, ErrExpected: true},
		{InFile: testFile.Name(), Payload: records, ErrExpected: false},
		{InFile: testFile.Name(), Payload: missAligned, ErrExpected: true},
	}
	switch runtime.GOOS {
	case "windows":
		tests = append(tests,
			testTable{InFile: "*", Payload: [][]string{}, ErrExpected: true},
			testTable{InFile: "*", Payload: nil, ErrExpected: true},
		)
	default:
		tests = append(tests,
			testTable{InFile: "", Payload: [][]string{}, ErrExpected: true},
			testTable{InFile: "", Payload: nil, ErrExpected: true},
		)
	}

	for x := range tests {
		err := tester(tests[x].InFile, tests[x].Payload)
		if err != nil && !tests[x].ErrExpected {
			t.Errorf("Test %d failed, unexpected err %s\n", x, err)
		}
	}
}

func TestWriter(t *testing.T) {
	type args struct {
		file string
	}
	tmp := t.TempDir()

	testData := `data`

	tests := []struct {
		name    string
		args    args
		want    *os.File
		wantErr bool
	}{
		{
			name:    "invalid",
			args:    args{"//invalid-nofile\\"},
			wantErr: true,
		},
		{
			name:    "empty",
			args:    args{""},
			wantErr: true,
		},
		{
			name: "relative newfile",
			args: args{"newfile"},
		},
		{
			name: "deep file",
			args: args{filepath.Join(tmp, "new", "file", "multiple", "sub", "paths")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Writer(tt.args.file)
			if err != nil {
				if (err != nil) != tt.wantErr {
					t.Errorf("Writer() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			defer os.Remove(got.Name())
			fileInfo, err := os.Stat(got.Name())
			if err != nil {
				t.Fatal(err)
			}
			if !fileInfo.Mode().IsRegular() {
				t.Fatalf("Writer() error = expected to get a file %s", got.Name())
			}
			_, err = got.WriteString(testData)
			if err != nil {
				t.Fatal(err)
			}
			err = got.Close()
			if err != nil {
				t.Fatal(err)
			}
			if data, err := os.ReadFile(got.Name()); err != nil || string(data) != testData {
				t.Errorf("Could not write the file, or contents were wrong: expected = %s, got =%s", testData, string(data))
			}
		})
	}
}

func TestWriterNoPermissionFails(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getenv("GCT_DOCKER_CI") == "true" {
		t.Skip("Skipping file permission test on Windows or in Docker CI")
	}

	tempDir := t.TempDir()
	err := os.Chmod(tempDir, 0o555)
	require.NoError(t, err, "Chmod must not fail to set read-only permissions")
	_, err = Writer(filepath.Join(tempDir, "path", "to", "somefile"))
	assert.Error(t, err, "Writer should fail with no write permissions")
}
