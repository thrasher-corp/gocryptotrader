package assets

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

func TestString(t *testing.T) {
	a := AssetTypeSpot
	if a.String() != "spot" {
		t.Fatal("Test failed - TestString returned an unexpected result")
	}
}

func TestToStringArray(t *testing.T) {
	a := AssetTypes{AssetTypeSpot, AssetTypeFutures}
	result := a.Strings()
	for x := range a {
		if !common.StringDataCompare(result, a[x].String()) {
			t.Fatal("Test failed - TestToStringArray returned an unexpected result")
		}
	}
}

func TestContains(t *testing.T) {
	a := AssetTypes{AssetTypeSpot, AssetTypeFutures}
	if a.Contains("meow") {
		t.Fatal("Test failed - TestContains returned an unexpected result")
	}

	if !a.Contains(AssetTypeSpot) {
		t.Fatal("Test failed - TestContains returned an unexpected result")
	}

	if a.Contains(AssetTypeBinary) {
		t.Fatal("Test failed - TestContains returned an unexpected result")
	}
}

func TestJoinToString(t *testing.T) {
	a := AssetTypes{AssetTypeSpot, AssetTypeFutures}
	if a.JoinToString(",") != "spot,futures" {
		t.Fatal("Test failed - TestJoinToString returned an unexpected result")
	}
}

func TestIsValid(t *testing.T) {
	if IsValid("rawr") {
		t.Fatal("Test failed - TestIsValid returned an unexpected result")
	}

	if !IsValid(AssetTypeSpot) {
		t.Fatal("Test failed - TestIsValid returned an unexpected result")
	}
}

func TestNew(t *testing.T) {
	a := New("Spota")
	if a != nil {
		t.Fatal("Test failed - TestNew returned an unexpected result")
	}

	a = New("SpOt")
	if a == nil {
		t.Fatal("Test failed - TestNew returned an unexpected result")
	}

	a = New("spot,futures")
	if a.JoinToString(",") != "spot,futures" {
		t.Fatal("Test failed - TestNew returned an unexpected result")
	}

	if a := New("Spot_rawr"); a != nil {
		t.Fatal("Test failed - TestNew returned an unexpected result")
	}

	if a := New("Spot,Rawr"); a != nil {
		t.Fatal("Test failed - TestNew returned an unexpected result")
	}
}
