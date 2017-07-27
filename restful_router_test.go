package main

import (
	"testing"
)

func TestNewRouter(t *testing.T) {
	if value := NewRouter(bot.exchanges); value.KeepContext {
		t.Error("Test Failed - Restful_Router_Test.go - NewRouter Error")
	}
}
