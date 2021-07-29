package connchecker

import (
	"testing"
	"time"
)

func TestConnection(t *testing.T) {
	faultyDomain := []string{"faultyIP"}
	faultyHost := []string{"faultyHost"}
	_, err := New(faultyDomain, nil, 1*time.Second)
	if err == nil {
		t.Fatal("New error cannot be nil")
	}

	_, err = New(DefaultDNSList, nil, 1*time.Second)
	if err != nil {
		t.Fatal("New error", err)
	}

	_, err = New(nil, faultyHost, 1*time.Second)
	if err != nil {
		t.Fatal("New error cannot be nil", err)
	}

	c, err := New(nil, nil, 0)
	if err != nil {
		t.Fatal("New error", err)
	}

	if !c.IsConnected() {
		t.Log("Test - No internet connection found")
	} else {
		t.Log("Test - Internet connection found")
	}

	c.Shutdown()
}
