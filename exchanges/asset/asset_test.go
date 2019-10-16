package asset

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
)

func TestString(t *testing.T) {
	a := Spot
	if a.String() != "spot" {
		t.Fatal("TestString returned an unexpected result")
	}
}

func TestToStringArray(t *testing.T) {
	a := Items{Spot, Futures}
	result := a.Strings()
	for x := range a {
		if !common.StringDataCompare(result, a[x].String()) {
			t.Fatal("TestToStringArray returned an unexpected result")
		}
	}
}

func TestContains(t *testing.T) {
	a := Items{Spot, Futures}
	if a.Contains("meow") {
		t.Fatal("TestContains returned an unexpected result")
	}

	if !a.Contains(Spot) {
		t.Fatal("TestContains returned an unexpected result")
	}

	if a.Contains(Binary) {
		t.Fatal("TestContains returned an unexpected result")
	}

	if !a.Contains("SpOt") {
		t.Error("TestContains returned an unexpected result")
	}
}

func TestJoinToString(t *testing.T) {
	a := Items{Spot, Futures}
	if a.JoinToString(",") != "spot,futures" {
		t.Fatal("TestJoinToString returned an unexpected result")
	}
}

func TestIsValid(t *testing.T) {
	if IsValid("rawr") {
		t.Fatal("TestIsValid returned an unexpected result")
	}

	if !IsValid(Spot) {
		t.Fatal("TestIsValid returned an unexpected result")
	}
}

func TestNew(t *testing.T) {
	a := New("Spota")
	if a != nil {
		t.Fatal("TestNew returned an unexpected result")
	}

	a = New("SpOt")
	if a == nil {
		t.Fatal("TestNew returned an unexpected result")
	}

	a = New("spot,futures")
	if a.JoinToString(",") != "spot,futures" {
		t.Fatal("TestNew returned an unexpected result")
	}

	if a := New("Spot_rawr"); a != nil {
		t.Fatal("TestNew returned an unexpected result")
	}

	if a := New("Spot,Rawr"); a != nil {
		t.Fatal("TestNew returned an unexpected result")
	}
}
