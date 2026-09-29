package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAssertionMessages(t *testing.T) {
	t.Parallel()
	analysistest.Run(t, analysistest.TestData(), newAnalyzer(), "./messages")
}

func TestTrackedSources(t *testing.T) {
	t.Parallel()
	data := analysistest.TestData()
	list := filepath.Join(t.TempDir(), "sources")
	err := os.WriteFile(list, []byte(filepath.Join(data, "filtered", "tracked.go")+"\x00"), 0o600)
	require.NoError(t, err, "source list must be written")
	analyzer := newAnalyzer()
	require.NoError(t, analyzer.Flags.Set("files", list), "source list flag must be accepted")
	analysistest.Run(t, data, analyzer, "./filtered")
}
