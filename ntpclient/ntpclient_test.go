package ntpclient

import (
	"testing"
)

func TestNTPClient(t *testing.T) {
	pool := []string{"pool.ntp.org:123", "0.pool.ntp.org:123"}
	_, err := NTPClient(pool)
	if err != nil {
		t.Fatalf("failed to get time %v", err)
	}

	invalidpool := []string{"pool.thisisinvalid.org"}
	_, err = NTPClient(invalidpool)
	if err == nil {
		t.Errorf("Expected error")
	}

	firstInvalid := []string{"pool.thisisinvalid.org", "pool.ntp.org:123", "0.pool.ntp.org:123"}
	_, err = NTPClient(firstInvalid)
	if err != nil {
		t.Errorf("failed to get time %v", err)
	}
}
