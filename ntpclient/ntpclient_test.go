package ntpclient

import (
	"testing"
)

func TestNTPClient(t *testing.T) {
	pool := []string{"pool.ntp.org:123", "0.pool.ntp.org:123"}
	NTPTime, err := NTPClient(pool)
	if err != nil {
		t.Errorf("failed to get time %v", err)
	}
	if NTPTime.IsZero() {
		t.Error("expected time but 0,0 returned")
	}
	invalidpool := []string{"pool.thisisinvalid.org"}
	_, err = NTPClient(invalidpool)
	if err == nil {
		t.Errorf("failed to get time %v", err)
	}

	firstInvalid := []string{"pool.thisisinvalid.org", "pool.ntp.org:123", "0.pool.ntp.org:123"}
	_, err = NTPClient(firstInvalid)
	if err != nil {
		t.Errorf("failed to get time %v", err)
	}
}
