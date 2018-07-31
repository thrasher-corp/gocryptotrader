package main

import "testing"

func TestSetupBotExchanges(t *testing.T) {
	// setupBotExchanges()
}

func TestMain(t *testing.T) {
	// Nothing
}

func TestAdjustGoMaxProcs(t *testing.T) {
	AdjustGoMaxProcs()
}

func TestHandleInterrupt(t *testing.T) {
	HandleInterrupt()
}

func TestShutdown(t *testing.T) {
	// Nothing
}

func TestSeedExchangeAccountInfo(t *testing.T) {
	SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)
}
