package ntpmanager

import (
	"reflect"
	"testing"
	"time"
)

func TestNTPClient(t *testing.T) {
	pool := []string{"pool.ntp.org:123", "0.pool.ntp.org:123"}
	v := checkTimeInPools(pool)

	if reflect.TypeOf(v) != reflect.TypeOf(time.Time{}) {
		t.Errorf("CheckTimeInPools should return time.Time{}")
	}

	if v.IsZero() {
		t.Error("CheckTimeInPools should return valid time received zero value")
	}

	const timeFormat = "2006-01-02T15:04"

	if v.UTC().Format(timeFormat) != time.Now().UTC().Format(timeFormat) {
		t.Errorf("CheckTimeInPools returned incorrect time received: %v", v.UTC().Format(timeFormat))
	}
}
