package tests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	t := m.Run()

	os.Exit(t)
}
