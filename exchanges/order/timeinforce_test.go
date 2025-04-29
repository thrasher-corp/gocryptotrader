package order

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestIsValid(t *testing.T) {
	t.Parallel()
	timeInForceValidityMap := map[TimeInForce]bool{
		TimeInForce(1):                     false,
		ImmediateOrCancel:                  true,
		GoodTillTime:                       true,
		GoodTillCancel:                     true,
		GoodTillDay:                        true,
		FillOrKill:                         true,
		PostOnly:                           true,
		FillOrKill | ImmediateOrCancel:     false,
		FillOrKill | GoodTillCancel:        false,
		FillOrKill | PostOnly:              false,
		ImmediateOrCancel | GoodTillCancel: false,
		ImmediateOrCancel | PostOnly:       false,
		UnknownTIF:                         true,
	}
	var tif TimeInForce
	for tif = range timeInForceValidityMap {
		assert.Equalf(t, timeInForceValidityMap[tif], tif.IsValid(), "got %v, expected %v for %v with id %d", tif.IsValid(), timeInForceValidityMap[tif], tif, tif)
	}
}

var timeInForceStringToValueMap = map[string]struct {
	TIF   TimeInForce
	Error error
}{
	"GoodTillCancel":               {TIF: GoodTillCancel},
	"GOOD_TILL_CANCELED":           {TIF: GoodTillCancel},
	"GTT":                          {TIF: GoodTillTime},
	"GOOD_TIL_TIME":                {TIF: GoodTillTime},
	"FILLORKILL":                   {TIF: FillOrKill},
	"POST_ONLY_GOOD_TIL_CANCELLED": {TIF: GoodTillCancel | PostOnly},
	"immedIate_Or_Cancel":          {TIF: ImmediateOrCancel},
	"IOC":                          {TIF: ImmediateOrCancel},
	"immediate_or_cancel":          {TIF: ImmediateOrCancel},
	"IMMEDIATE_OR_CANCEL":          {TIF: ImmediateOrCancel},
	"IMMEDIATEORCANCEL":            {TIF: ImmediateOrCancel},
	"GOOD_TILL_CANCELLED":          {TIF: GoodTillCancel},
	"good_till_day":                {TIF: GoodTillDay},
	"GOOD_TILL_DAY":                {TIF: GoodTillDay},
	"GTD":                          {TIF: GoodTillDay},
	"GOODtillday":                  {TIF: GoodTillDay},
	"abcdfeg":                      {TIF: UnknownTIF, Error: ErrInvalidTimeInForce},
	"PoC":                          {TIF: PostOnly},
	"PendingORCANCEL":              {TIF: PostOnly},
	"GTX":                          {TIF: GoodTillCrossing},
	"GOOD_TILL_CROSSING":           {TIF: GoodTillCrossing},
	"Good Til crossing":            {TIF: GoodTillCrossing},
}

func TestStringToTimeInForce(t *testing.T) {
	t.Parallel()
	for tk, exp := range timeInForceStringToValueMap {
		t.Run(tk, func(t *testing.T) {
			t.Parallel()
			result, err := StringToTimeInForce(tk)
			if exp.Error != nil {
				require.ErrorIs(t, err, exp.Error)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, exp.TIF, result)
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	valMap := map[TimeInForce]string{
		ImmediateOrCancel:              "IOC",
		GoodTillCancel:                 "GTC",
		GoodTillTime:                   "GTT",
		GoodTillDay:                    "GTD",
		FillOrKill:                     "FOK",
		UnknownTIF:                     "",
		PostOnly:                       "POSTONLY",
		GoodTillCancel | PostOnly:      "GTC,POSTONLY",
		GoodTillTime | PostOnly:        "GTT,POSTONLY",
		GoodTillDay | PostOnly:         "GTD,POSTONLY",
		FillOrKill | ImmediateOrCancel: "IOC,FOK",
	}
	for x := range valMap {
		result := x.String()
		assert.Equalf(t, valMap[x], result, "expected %v, got %v", x, result)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()
	targets := []TimeInForce{
		GoodTillCancel | PostOnly | ImmediateOrCancel, GoodTillCancel | PostOnly, GoodTillCancel, UnknownTIF, PostOnly | ImmediateOrCancel,
		GoodTillCancel, GoodTillCancel, PostOnly, PostOnly, ImmediateOrCancel, GoodTillDay, GoodTillDay, GoodTillTime, FillOrKill, FillOrKill,
	}
	data := `{"tifs": ["GTC,POSTONLY,IOC", "GTC,POSTONLY", "GTC", "", "POSTONLY,IOC", "GoodTilCancel", "GoodTILLCANCEL", "POST_ONLY", "POC","IOC", "GTD", "gtd","gtt", "fok", "fillOrKill"]}`
	target := &struct {
		TIFs []TimeInForce `json:"tifs"`
	}{}
	err := json.Unmarshal([]byte(data), &target)
	require.NoError(t, err)
	require.Equal(t, targets, target.TIFs)
}

func TestMarshalJSON(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(GoodTillCrossing)
	require.NoError(t, err)
	assert.Equal(t, []byte(`"GTX"`), data)

	data = []byte(`{"tif":"IOC"}`)
	target := &struct {
		TimeInForce TimeInForce `json:"tif"`
	}{}
	err = json.Unmarshal(data, &target)
	require.NoError(t, err)
	assert.Equal(t, "IOC", target.TimeInForce.String())
}

// BenchmarkStringToTimeInForce-8            416595              2834 ns/op            1368 B/op         81 allocs/op
func BenchmarkStringToTimeInForce(b *testing.B) {
	for b.Loop() {
		for k := range timeInForceStringToValueMap {
			_, _ = StringToTimeInForce(k)
		}
	}
}
