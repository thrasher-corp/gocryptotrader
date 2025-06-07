package asset

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestString(t *testing.T) {
	t.Parallel()
	for a := range All {
		if a == 0 {
			assert.Empty(t, a.String(), "Empty.String should return empty")
		} else {
			assert.NotEmptyf(t, a.String(), "%s.String should return empty", a)
		}
	}
}

func TestStrings(t *testing.T) {
	t.Parallel()
	assert.ElementsMatch(t, Items{Spot, Futures}.Strings(), []string{"spot", "futures"})
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
	for a := range All {
		if a.String() == "" {
			require.Falsef(t, a.IsValid(), "IsValid must return false with non-asset value %d", a)
		} else {
			require.Truef(t, a.IsValid(), "IsValid must return true for %s", a)
		}
	}
	require.False(t, All.IsValid(), "IsValid must return false for All")
}

func TestIsFutures(t *testing.T) {
	t.Parallel()
	valid := []Item{PerpetualContract, PerpetualSwap, Futures, DeliveryFutures, UpsideProfitContract, DownsideProfitContract, CoinMarginedFutures, USDTMarginedFutures, USDCMarginedFutures, FutureCombo, LinearContract, Spread}
	for a := range All {
		if slices.Contains(valid, a) {
			require.Truef(t, a.IsFutures(), "IsFutures must return true for %s", a)
		} else {
			require.Falsef(t, a.IsFutures(), "IsFutures must return false for non-asset value %d (%s)", a, a)
		}
	}
}

func TestIsOptions(t *testing.T) {
	t.Parallel()
	valid := []Item{Options, OptionCombo}
	for a := range All {
		if slices.Contains(valid, a) {
			require.Truef(t, a.IsOptions(), "IsOptions must return true for %s", a)
		} else {
			require.Falsef(t, a.IsOptions(), "IsOptions must return false for non-asset value %d (%s)", a, a)
		}
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
		{Input: "Options", Expected: Options},
		{Input: "Option", Expected: Options},
		{Input: "Future", Error: ErrNotSupported},
		{Input: "option_combo", Expected: OptionCombo},
		{Input: "future_combo", Expected: FutureCombo},
		{Input: "spread", Expected: Spread},
		{Input: "linearContract", Expected: LinearContract},
	}

	for _, tt := range cases {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			returned, err := New(tt.Input)
			require.ErrorIs(t, err, tt.Error)

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
	for i := range supportedList {
		if s[i] != supportedList[i] {
			t.Fatal("TestSupported returned an unexpected result")
		}
	}
}

func TestUnmarshalMarshal(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(Item(0))
	require.NoError(t, err)

	if string(data) != `""` {
		t.Fatal("unexpected value")
	}

	data, err = json.Marshal(Spot)
	require.NoError(t, err)

	if string(data) != `"spot"` {
		t.Fatal("unexpected value")
	}

	var spot Item

	err = json.Unmarshal(data, &spot)
	require.NoError(t, err)

	if spot != Spot {
		t.Fatal("unexpected value")
	}

	err = json.Unmarshal([]byte(`"confused"`), &spot)
	require.ErrorIs(t, err, ErrNotSupported)

	err = json.Unmarshal([]byte(`""`), &spot)
	require.NoError(t, err)

	err = json.Unmarshal([]byte(`123`), &spot)
	assert.Error(t, err, "Unmarshal should error correctly")
}

func TestUseDefault(t *testing.T) {
	t.Parallel()
	if UseDefault() != Spot {
		t.Fatalf("received: '%v' but expected: '%v'", UseDefault(), Spot)
	}
}
