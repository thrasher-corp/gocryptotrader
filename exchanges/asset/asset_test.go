package asset

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
)

func TestString(t *testing.T) {
	t.Parallel()
	a := Spot
	if a.String() != "spot" {
		t.Fatal("TestString returned an unexpected result")
	}

	a = 0
	if a.String() != "" {
		t.Fatal("TestString returned an unexpected result")
	}
}

func TestToStringArray(t *testing.T) {
	t.Parallel()
	a := Items{Spot, Futures}
	result := a.Strings()
	for x := range a {
		if !common.StringDataCompare(result, a[x].String()) {
			t.Fatal("TestToStringArray returned an unexpected result")
		}
	}
}

func TestContains(t *testing.T) {
	t.Parallel()
	a := Items{Spot, Futures}
	if a.Contains(666) {
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
	if a.Contains(0) {
		t.Error("TestContains returned an unexpected result")
	}
}

func TestJoinToString(t *testing.T) {
	t.Parallel()
	a := Items{Spot, Futures}
	if a.JoinToString(",") != "spot,futures" {
		t.Fatal("TestJoinToString returned an unexpected result")
	}
}

func TestIsValid(t *testing.T) {
	t.Parallel()
	if Item(0).IsValid() {
		t.Fatal("TestIsValid returned an unexpected result")
	}

	if !Spot.IsValid() {
		t.Fatal("TestIsValid returned an unexpected result")
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Input    string
		Expected Item
		Error    error
	}{
		{Input: "Spota", Error: ErrNotSupported},
		{Input: "MARGIN", Expected: Margin},
		{Input: "MARGINFUNDING", Expected: MarginFunding},
		{Input: "INDEX", Expected: Index},
		{Input: "BINARY", Expected: Binary},
		{Input: "PERPETUALCONTRACT", Expected: PerpetualContract},
		{Input: "PERPETUALSWAP", Expected: PerpetualSwap},
		{Input: "FUTURES", Expected: Futures},
		{Input: "UpsideProfitContract", Expected: UpsideProfitContract},
		{Input: "DownsideProfitContract", Expected: DownsideProfitContract},
		{Input: "CoinMarginedFutures", Expected: CoinMarginedFutures},
		{Input: "USDTMarginedFutures", Expected: USDTMarginedFutures},
		{Input: "USDCMarginedFutures", Expected: USDCMarginedFutures},
	}

	for x := range cases {
		tt := cases[x]
		t.Run("", func(t *testing.T) {
			t.Parallel()
			returned, err := New(tt.Input)
			if !errors.Is(err, tt.Error) {
				t.Fatalf("received: '%v' but expected: '%v'", err, tt.Error)
			}
			if returned != tt.Expected {
				t.Fatalf("received: '%v' but expected: '%v'", returned, tt.Expected)
			}
		})
	}
}

func TestSupported(t *testing.T) {
	t.Parallel()
	s := Supported()
	if len(supportedList) != len(s) {
		t.Fatal("TestSupported mismatched lengths")
	}
	for i := 0; i < len(supportedList); i++ {
		if s[i] != supportedList[i] {
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
		{
			item:      USDCMarginedFutures,
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

func TestUnmarshalMarshal(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(Item(0))
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if string(data) != `""` {
		t.Fatal("unexpected value")
	}

	data, err = json.Marshal(Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if string(data) != `"spot"` {
		t.Fatal("unexpected value")
	}

	var spot Item

	err = json.Unmarshal(data, &spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if spot != Spot {
		t.Fatal("unexpected value")
	}

	err = json.Unmarshal([]byte(`"confused"`), &spot)
	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNotSupported)
	}

	err = json.Unmarshal([]byte(`""`), &spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = json.Unmarshal([]byte(`123`), &spot)
	if errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", nil, "an error")
	}
}

func TestUseDefault(t *testing.T) {
	t.Parallel()
	if UseDefault() != Spot {
		t.Fatalf("received: '%v' but expected: '%v'", UseDefault(), Spot)
	}
}
