package kline

import (
	"errors"
	"testing"
	"time"
)

func TestGetBuilder(t *testing.T) {
	_, err := GetBuilder(0, 0)
	if errors.Is(err, ErrUnsetInterval) {
		t.Fatal(err)
	}
}

func TestWow(t *testing.T) {

	resp, err := Wow(FifteenSecond * 2)
	if err != nil {
		t.Fatal(err)
	}
	if resp != FifteenSecond {
		t.Fatalf("not what I wanted %s", resp)
	}

	resp, err = Wow(FifteenSecond*3 + Interval(time.Second)) // 46 seconds does not exactly match
	if err != nil {
		t.Fatal(err)
	}
	if resp != FifteenSecond {
		t.Fatalf("not what I wanted %s", resp)
	}
}
