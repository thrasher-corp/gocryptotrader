package ntpclient

import (
	"reflect"
	"testing"
	"time"
)

func TestNTPClient(t *testing.T) {
	pool := []string{"pool.ntp.org:123", "0.pool.ntp.org:123"}
	v := NTPClient(pool)

	if reflect.TypeOf(v) != reflect.TypeOf(time.Time{}) {
		t.Errorf("NTPClient should return time.Time{}")
	}
}
