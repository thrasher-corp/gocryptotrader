package connchecker

import (
	"testing"
)

func TestConnection(t *testing.T) {
	faultyDomain := []string{"faultyIP"}
	faultyHost := []string{"faultyHost"}
	_, err := New(faultyDomain, nil, 100000)
	if err == nil {
		t.Fatal("Test Failed - New error cannot be nil")
	}

	_, err = New(DefaultDNSList, nil, 100000)
	if err != nil {
		t.Fatal("Test Failed - New error", err)
	}

	_, err = New(nil, faultyHost, 100000)
	if err != nil {
		t.Fatal("Test Failed - New error cannot be nil", err)
	}

	c, err := New(nil, nil, 0)
	if err != nil {
		t.Fatal("Test Failed - New error", err)
	}

	if !c.IsConnected() {
		t.Log("Test - No internet connection found")
	} else {
		t.Log("Test - Internet connection found")
	}

	c.Shutdown()
}
