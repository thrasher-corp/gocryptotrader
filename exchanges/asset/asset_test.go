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

	// Every asset should be created and matched with func New so this should
	// not be matched against list
	if a.Contains("SpOt") {
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
	if Item("rawr").IsValid() {
		t.Fatal("TestIsValid returned an unexpected result")
	}

	if !Spot.IsValid() {
		t.Fatal("TestIsValid returned an unexpected result")
	}
}

func TestNew(t *testing.T) {
	if _, err := New("Spota"); err == nil {
		t.Fatal("TestNew returned an unexpected result")
	}

	a, err := New("SpOt")
	if err != nil {
		t.Fatal("TestNew returned an unexpected result", err)
	}

	if a != Spot {
		t.Fatal("TestNew returned an unexpected result")
	}
}

func TestSupported(t *testing.T) {
	s := Supported()
	if len(supported) != len(s) {
		t.Fatal("TestSupported mismatched lengths")
	}
	for i := 0; i < len(supported); i++ {
		if s[i] != supported[i] {
			t.Fatal("TestSupported returned an unexpected result")
		}
	}
}

func TestIsFutures(t *testing.T) {
	t.Parallel()
	type scenario struct {
		item      Item
		isFutures bool
	}
	scenarios := []scenario{
		{
			item:      Spot,
			isFutures: false,
		},
		{
			item:      Margin,
			isFutures: false,
		},
		{
			item:      MarginFunding,
			isFutures: false,
		},
		{
			item:      Index,
			isFutures: false,
		},
		{
			item:      Binary,
			isFutures: false,
		},
		{
			item:      PerpetualContract,
			isFutures: true,
		},
		{
			item:      PerpetualSwap,
			isFutures: true,
		},
		{
			item:      Futures,
			isFutures: true,
		},
		{
			item:      UpsideProfitContract,
			isFutures: true,
		},
		{
			item:      DownsideProfitContract,
			isFutures: true,
		},
		{
			item:      CoinMarginedFutures,
			isFutures: true,
		},
		{
			item:      USDTMarginedFutures,
			isFutures: true,
		},
	}
	for _, s := range scenarios {
		testScenario := s
		t.Run(testScenario.item.String(), func(t *testing.T) {
			t.Parallel()
			if testScenario.item.IsFutures() != testScenario.isFutures {
				t.Errorf("expected %v isFutures to be %v", testScenario.item, testScenario.isFutures)
			}
		})
	}
}
