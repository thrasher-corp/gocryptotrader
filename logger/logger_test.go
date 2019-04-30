package logger

import (
	"os"
	"path/filepath"
	"testing"
)

var (
	trueptr  = func(b bool) *bool { return &b }(true)
	falseptr = func(b bool) *bool { return &b }(false)
)

func TestCloseLogFile(t *testing.T) {
	Logger = &Logging{
		Enabled:      trueptr,
		Level:        "DEBUG",
		ColourOutput: false,
		File:         "",
		Rotate:       false,
	}
	SetupLogger()
	err := CloseLogFile()
	if err != nil {
		t.Fatalf("CloseLogFile failed with %v", err)
	}
	os.Remove(filepath.Join(LogPath, Logger.File))
}

func TestSetupOutputsValidPath(t *testing.T) {
	Logger.Enabled = trueptr
	Logger.File = "debug.txt"
	LogPath = "../testdata/"
	err := setupOutputs()
	if err != nil {
		t.Fatalf("SetupOutputs failed expected nil got %v", err)
	}

	err = CloseLogFile()
	if err != nil {
		t.Fatalf("CloseLogFile failed with %v", err)
	}

	err = os.Remove(filepath.Join(LogPath, Logger.File))
	if err != nil {
		t.Fatal("Test Failed - SetupOutputsValidPath() error could not remove test file", err)
	}
}

func TestSetupOutputsInValidPath(t *testing.T) {
	Logger.Enabled = trueptr
	Logger.File = "debug.txt"
	LogPath = "../testdataa/"
	err := setupOutputs()
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("SetupOutputs failed expected %v got %v", os.ErrNotExist, err)
		}
	}
	err = os.Remove(filepath.Join(LogPath, Logger.File))
	if err == nil {
		t.Fatal("Test Failed - SetupOutputsInValidPath() error cannot be nil")
	}
}

func BenchmarkDebugf(b *testing.B) {
	Logger = &Logging{
		Enabled:      trueptr,
		Level:        "DEBUG",
		ColourOutput: false,
		File:         "",
		Rotate:       false,
	}
	SetupLogger()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Debugf("This is a debug benchmark %d", n)
	}
}

func BenchmarkDebugfLoggerDisabled(b *testing.B) {
	clearAllLoggers()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Debugf("this is a debug benchmark")
	}
}
