package logger

import (
	"os"
	"path"
	"testing"
)

func TestCloseLogFile(t *testing.T) {
	LogPath = "../testdata/"
	Logger.Enabled = true
	Logger.File = "testdebug.txt"
	SetupLogger()
	err := CloseLogFile()
	if err != nil {
		t.Fatalf("CloseLogFile failed with %v", err)
	}
	os.Remove(path.Join(LogPath, Logger.File))
}

func TestSetupOutputsValidPath(t *testing.T) {
	Logger.Enabled = true
	Logger.File = "debug.txt"
	LogPath = "../testdata/"
	err := setupOutputs()
	if err != nil {
		t.Fatalf("SetupOutputs failed expected nil got %v", err)
	}
	os.Remove(path.Join(LogPath, Logger.File))
}

func TestSetupOutputsInValidPath(t *testing.T) {
	Logger.Enabled = true
	Logger.File = "debug.txt"
	LogPath = "../testdataa/"
	err := setupOutputs()
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("SetupOutputs failed expected %v got %v", os.ErrNotExist, err)
		}
	}
	os.Remove(path.Join(LogPath, Logger.File))
}

func BenchmarkDebugf(b *testing.B) {
	Logger.Enabled = true
	Logger.Level = "DEBUG"
	SetupLogger()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Debugf("This is a debug benchmark %d", n)
	}
}

func BenchmarkDebugfLoggerDisabled(b *testing.B) {
	//Logger.Enabled = false
	clearAllLoggers()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Debugf("this is a debug benchmark")
	}
}
