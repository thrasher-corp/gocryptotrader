package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestCheckExchangeName(t *testing.T) {
	tester := []struct {
		Name        string
		ErrExpected error
	}{
		{
			Name:        "test exch",
			ErrExpected: errInvalidExchangeName,
		},
		{
			ErrExpected: errInvalidExchangeName,
		},
		{
			Name:        " ",
			ErrExpected: errInvalidExchangeName,
		},
		{
			Name:        "m",
			ErrExpected: errInvalidExchangeName,
		},
		{
			Name:        "mu",
			ErrExpected: errInvalidExchangeName,
		},
		{
			Name: "testexch",
		},
	}

	for x := range tester {
		if r := checkExchangeName(tester[x].Name); r != tester[x].ErrExpected {
			t.Errorf("test: %d unexpected result", x)
		}
	}
}

func TestNewExchange(t *testing.T) {
	testExchangeName := "testexch"
	testExchangeDir := filepath.Join(targetPath, testExchangeName)

	_, err := makeExchange(
		testExchangeDir,
		config.GetConfig(),
		&exchange{
			Name: testExchangeName,
			REST: true,
			WS:   true,
		})
	if err != nil {
		t.Error(err)
	}

	if err := os.RemoveAll(testExchangeDir); err != nil {
		t.Errorf("unable to remove dir: %s, manual removal required", err)
	}
}
