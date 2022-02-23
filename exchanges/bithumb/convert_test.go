package bithumb

import (
	"encoding/json"
	"testing"
)

func TestBithumbTime(t *testing.T) {
	var newTime bithumbTime
	err := json.Unmarshal([]byte("bad news"), &newTime)
	if err == nil {
		t.Fatal(err)
	}

	strData := []byte(`"1628739590000000"`) // Thursday, August 12, 2021 3:39:50 AM UTC
	err = json.Unmarshal(strData, &newTime)
	if err != nil {
		t.Fatal(err)
	}

	tt := newTime.Time()
	if tt.UTC().String() != "2021-08-12 03:39:50 +0000 UTC" {
		t.Fatalf("expected: %s but received: %s",
			"2021-08-12 03:39:50 +0000 UTC",
			tt.UTC().String())
	}
}
