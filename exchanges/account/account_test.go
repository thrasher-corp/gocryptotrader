package account

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/dispatch"
)

func TestMain(m *testing.M) {
	err := dispatch.Start(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestDeployHoldings(t *testing.T) {
	_, err := DeployHoldings("", false)
	if !errors.Is(err, errExchangeNameUnset) {
		t.Fatalf("expected: %v but received: %v", errExchangeNameUnset, err)
	}

	h, err := DeployHoldings("test", false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	if h == nil {
		t.Fatal("holdings variable should not be nil")
	}
	if h.id.String() == "" {
		t.Fatal("mux id should have been populated")
	}

	service.Lock()
	h2, ok := service.accounts["test"]
	service.Unlock()
	if !ok {
		t.Fatal("holdings should be populated in account services exchanges map")
	}

	if h != h2 {
		t.Fatal("these two instances should be the same")
	}

	_, err = DeployHoldings("test", false)
	if !errors.Is(err, errExchangeAlreadyDeployed) {
		t.Fatalf("expected: %v but received: %v", errExchangeAlreadyDeployed, err)
	}
}

func TestSubscribeToExchangeAccount(t *testing.T) {
	_, err := SubscribeToExchangeAccount("")
	if !errors.Is(err, errExchangeNameUnset) {
		t.Fatalf("expected: %v but received: %v", errExchangeNameUnset, err)
	}

	_, err = SubscribeToExchangeAccount("bro")
	if !errors.Is(err, errExchangeHoldingsNotFound) {
		t.Fatalf("expected: %v but received: %v", errExchangeHoldingsNotFound, err)
	}

	DeployHoldings("test", false)

	_, err = SubscribeToExchangeAccount("test")
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
}
