package engine

import (
	"testing"
)

func TestNewSyncManager(t *testing.T) {
	SetupTestHelpers(t)

	_, err := NewSyncManager(SyncConfig{})
	if err == nil {
		t.Error("error cannot be nil")
	}

	sm, err := NewSyncManager(SyncConfig{AccountBalance: true})
	if err != nil {
		t.Error(err)
	}

	if err = sm.Stop(); err == nil {
		t.Fatal("error cannot be nil")
	}

	if err = sm.Start(); err != nil {
		t.Fatal(err)
	}

	if err = sm.Start(); err == nil {
		t.Fatal("error cannot be nil")
	}

	if err = sm.Stop(); err != nil {
		t.Fatal(err)
	}

	Bot.ServicesWG.Wait()

	if err = sm.Start(); err != nil {
		t.Fatal(err)
	}
}
