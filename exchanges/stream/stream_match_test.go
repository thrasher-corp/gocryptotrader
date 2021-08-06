package stream

import (
	"fmt"
	"testing"
)

func TestMatch(t *testing.T) {
	t.Parallel()
	bm := &Match{}
	if bm.Incoming("wow") {
		t.Fatal("Should not have matched")
	}

	nm := NewMatch()
	// try to match with unset signature
	if nm.Incoming("hello") {
		t.Fatal("should not be able to match")
	}

	m, err := nm.set("hello")
	if err != nil {
		t.Fatal(err)
	}

	_, err = nm.set("hello")
	if err == nil {
		t.Fatal("error cannot be nil as this collision cannot occur")
	}

	if m.sig != "hello" {
		t.Fatal(err)
	}

	// try and match with initial payload
	if !nm.Incoming("hello") {
		t.Fatal("should of matched")
	}

	// put in secondary payload with conflicting signature
	if nm.Incoming("hello") {
		fmt.Println("should not have been able to match")
	}

	data := <-m.C
	if data != nil {
		t.Fatal("wow")
	}

	m.Cleanup()
}
