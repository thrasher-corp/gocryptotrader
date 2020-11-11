package protocol

import (
	"fmt"
	"testing"
)

func TestFunctionality(t *testing.T) {
	var s *State
	_, err := s.Functionality()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	s = &State{}
	f, err := s.Functionality()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(f)
}
