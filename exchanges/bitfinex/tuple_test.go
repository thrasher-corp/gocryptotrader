package bitfinex

import (
	"math"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// tupleProbe has a field of every kind the tuple decoder handles, with position 15 left untagged
type tupleProbe struct {
	Float    float64           `tuple:"0"`
	Int      int64             `tuple:"1"`
	Uint     uint64            `tuple:"2"`
	Text     string            `tuple:"3"`
	Flag     bool              `tuple:"4"`
	Time     types.Time        `tuple:"5"`
	Boolean  types.Boolean     `tuple:"6"`
	Raw      json.RawMessage   `tuple:"7"`
	Level    tupleProbeLevel   `tuple:"8"`
	Levels   []tupleProbeLevel `tuple:"9"`
	Number   types.Number      `tuple:"10"`
	Code     currency.Code     `tuple:"11"`
	Optional *float64          `tuple:"12"`
	Strings  []string          `tuple:"13"`
	Small    int               `tuple:"14"`
	Last     string            `tuple:"16"`
}

func (p *tupleProbe) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, p) }

type tupleProbeLevel struct {
	Price  float64 `tuple:"0"`
	Amount float64 `tuple:"2"`
}

func (l *tupleProbeLevel) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, l) }

// tupleProbeKeyed is a level Bitfinex prefixes with its symbol
type tupleProbeKeyed struct {
	Symbol string          `tuple:"0"`
	Level  tupleProbeLevel `tuple:"1,inline"`
}

func (k *tupleProbeKeyed) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, k) }

func TestUnmarshalTuple(t *testing.T) {
	t.Parallel()
	var got tupleProbe
	require.NoError(t, json.Unmarshal([]byte(` [ 1.5, -2, 18446744073709551615, "a \"quoted\" é", true, 1700000000123, 1,
		{"a":[1,"]"]}, [10.5, 3, 0.25], [[1,null,2], null, [3]], "4.5", "UST", 6.5, ["x","y"], 7, {"skipped":true},
		"last", "appended", [1,2] ] `), &got), "Unmarshal must not error")
	optional := 6.5
	assert.Equal(t, tupleProbe{
		Float:    1.5,
		Int:      -2,
		Uint:     math.MaxUint64,
		Text:     `a "quoted" é`,
		Flag:     true,
		Time:     types.Time(time.UnixMilli(1700000000123)),
		Boolean:  true,
		Raw:      json.RawMessage(`{"a":[1,"]"]}`),
		Level:    tupleProbeLevel{Price: 10.5, Amount: 0.25},
		Levels:   []tupleProbeLevel{{Price: 1, Amount: 2}, {}, {Price: 3}},
		Number:   4.5,
		Code:     currency.UST,
		Optional: &optional,
		Strings:  []string{"x", "y"},
		Small:    7,
		Last:     "last",
	}, got, "Unmarshal should decode every kind, skipping untagged and appended elements")

	got = tupleProbe{Float: 9, Text: "kept"}
	require.NoError(t, json.Unmarshal([]byte(`[null,null,null,null,null,null,null,null,null,null,null,null,null]`), &got), "Unmarshal must not error for null elements")
	assert.Equal(t, tupleProbe{Float: 9, Text: "kept"}, got, "Null elements should leave their fields unset")

	for _, data := range []string{`[1.5]`, `[1.5 ]`} {
		var short tupleProbe
		require.NoErrorf(t, json.Unmarshal([]byte(data), &short), "Unmarshal must not error for short array %s", data)
		assert.Equalf(t, tupleProbe{Float: 1.5}, short, "A short array %s should set only its elements' fields", data)
	}

	var empty tupleProbe
	require.NoError(t, unmarshalTuple([]byte(`[ ]`), &empty), "unmarshalTuple must not error for an empty array")
	assert.Zero(t, empty, "An empty array should set no field")
	unchanged := tupleProbe{Text: "kept"}
	require.NoError(t, unmarshalTuple([]byte(` null `), &unchanged), "unmarshalTuple must not error for null")
	assert.Equal(t, tupleProbe{Text: "kept"}, unchanged, "A null tuple should leave the struct unset")

	var keyed []tupleProbeKeyed
	require.NoError(t, json.Unmarshal([]byte(`[["tBTCUSD",1,2,3],["fUSD",4]]`), &keyed), "Unmarshal must not error for inline levels")
	assert.Equal(t, []tupleProbeKeyed{{Symbol: "tBTCUSD", Level: tupleProbeLevel{Price: 1, Amount: 3}}, {Symbol: "fUSD", Level: tupleProbeLevel{Price: 4}}}, keyed, "Inline levels should decode from the position after their symbol")

	var quoted tupleProbe
	require.NoError(t, unmarshalTuple([]byte(`["1.5","-2","3"]`), &quoted), "unmarshalTuple must not error for quoted numbers")
	assert.Equal(t, tupleProbe{Float: 1.5, Int: -2, Uint: 3}, quoted, "Quoted numbers should decode as numbers")
	blank := tupleProbe{Float: 9, Int: 9, Uint: 9}
	require.NoError(t, unmarshalTuple([]byte(`["","",""]`), &blank), "unmarshalTuple must not error for empty strings")
	assert.Equal(t, tupleProbe{Float: 9, Int: 9, Uint: 9}, blank, "Empty strings should leave number fields unset")

	reused := tupleProbe{Levels: make([]tupleProbeLevel, 2, 4)}
	reused.Levels[0] = tupleProbeLevel{Price: 8, Amount: 8}
	require.NoError(t, unmarshalTuple([]byte(`[0,0,0,"",false,0,0,null,null,[[1,null]]]`), &reused), "unmarshalTuple must not error into a reused slice")
	assert.Equal(t, []tupleProbeLevel{{Price: 1}}, reused.Levels, "A reused slice's elements should not keep earlier values")
}

func TestUnmarshalTupleErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		data, field string
		err         error
	}{
		{data: `{"a":1}`, err: errTupleNotArray},
		{data: `"text"`, err: errTupleNotArray},
		{data: ``, err: errTupleNotArray},
		{data: `[1.5,`, err: errMalformedTuple},
		{data: `[1.5`, err: errMalformedTuple},
		{data: `[1.5 2]`, err: errMalformedTuple},
		{data: `[1.5,]`, err: errMalformedTuple},
		{data: `[1.5,}`, err: errMalformedTuple},
		{data: `["x"]`, field: "Float at position 0", err: errTupleElementType},
		{data: `["NaN"]`, field: "Float at position 0", err: errTupleElementType},
		{data: `["1e400"]`, field: "Float at position 0", err: errTupleElementType},
		{data: `["1\u0035"]`, field: "Float at position 0", err: errTupleElementType},
		{data: `[true]`, field: "Float at position 0", err: errTupleElementType},
		{data: `[1,1.5]`, field: "Int at position 1", err: errTupleElementType},
		{data: `[1,"2.5"]`, field: "Int at position 1", err: errTupleElementType},
		{data: `[1,2,-3]`, field: "Uint at position 2", err: errTupleElementType},
		{data: `[1,2,"-3"]`, field: "Uint at position 2", err: errTupleElementType},
		{data: `[1,2,18446744073709551616]`, field: "Uint at position 2", err: errTupleElementType},
		{data: `[1,2,3,4]`, field: "Text at position 3", err: errTupleElementType},
		{data: `[1,2,3,"a",1]`, field: "Flag at position 4", err: errTupleElementType},
		{data: `[1,2,3,"a",true,123]`, field: "Time at position 5", err: types.ErrInvalidTimestampFormat},
		{data: `[1,2,3,"a",true,0,1,null,{}]`, field: "Level at position 8", err: errTupleNotArray},
		{data: `[1,2,3,"a",true,0,1,null,null,{}]`, field: "Levels at position 9", err: errTupleElementType},
		{data: `[1,2,3,"a",true,0,1,null,null,[[1],["x"]]]`, field: "Levels at position 9: element 1: bitfinex.tupleProbeLevel.Price at position 0", err: errTupleElementType},
	} {
		var got tupleProbe
		err := unmarshalTuple([]byte(tc.data), &got)
		assert.ErrorIsf(t, err, tc.err, "unmarshalTuple should return the expected error for %s", tc.data)
		if tc.field != "" {
			assert.ErrorContainsf(t, err, "bitfinex.tupleProbe."+tc.field, "unmarshalTuple should name the field for %s", tc.data)
		}
	}
}

// tupleProbeLevels is a named slice type decoding one level as a list of it, by its own method
type tupleProbeLevels []tupleProbeLevel

func (l *tupleProbeLevels) UnmarshalJSON(data []byte) error {
	var level tupleProbeLevel
	if err := level.UnmarshalJSON(data); err != nil {
		return err
	}
	*l = tupleProbeLevels{level}
	return nil
}

type tupleNamedSlice struct {
	Levels tupleProbeLevels `tuple:"0"`
}

func (n *tupleNamedSlice) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, n) }

func TestUnmarshalTupleNamedSlice(t *testing.T) {
	t.Parallel()
	var got tupleNamedSlice
	require.NoError(t, json.Unmarshal([]byte(`[[1,0,2]]`), &got), "Unmarshal must not error for a named slice type")
	assert.Equal(t, tupleNamedSlice{Levels: tupleProbeLevels{{Price: 1, Amount: 2}}}, got, "A named slice type should decode by its own method rather than as a slice of tuples")
}

type (
	tupleSharedPosition struct {
		A int64 `tuple:"1"`
		B int64 `tuple:"1"`
	}
	tupleBadPosition struct {
		A int64 `tuple:"first"`
	}
	tupleNegativePosition struct {
		A int64 `tuple:"-1"`
	}
	tupleUnexported struct {
		a int64 `tuple:"0"`
	}
	tupleUnknownOption struct {
		A int64 `tuple:"0,omitempty"`
	}
	tupleEmbeddedInline struct {
		tupleProbeLevel `tuple:"1,inline"`
	}
	tupleScalarInline struct {
		A int64 `tuple:"1,inline"`
	}
	tupleUntagged struct {
		A int64
	}
	tupleNoMethod struct {
		A int64 `tuple:"0"`
	}
	tupleNestsNoMethod struct {
		Nested tupleNoMethod `tuple:"0"`
	}
)

func TestUnmarshalTupleInvalidTypes(t *testing.T) {
	t.Parallel()
	data := []byte(`[1,2]`)
	for name, err := range map[string]error{
		"shared position":    unmarshalTuple(data, &tupleSharedPosition{}),
		"bad position":       unmarshalTuple(data, &tupleBadPosition{}),
		"negative position":  unmarshalTuple(data, &tupleNegativePosition{}),
		"unexported field":   unmarshalTuple(data, &tupleUnexported{}),
		"unknown option":     unmarshalTuple(data, &tupleUnknownOption{}),
		"embedded inline":    unmarshalTuple(data, &tupleEmbeddedInline{}),
		"scalar inline":      unmarshalTuple(data, &tupleScalarInline{}),
		"no tags":            unmarshalTuple(data, &tupleUntagged{}),
		"not a struct":       unmarshalTuple(data, new(int64)),
		"nested no method":   unmarshalTuple(data, &tupleNestsNoMethod{}),
		"marshal no tags":    func() error { _, err := marshalTuple(&tupleUntagged{}); return err }(),
		"marshal not struct": func() error { _, err := marshalTuple(new(string)); return err }(),
	} {
		assert.ErrorIsf(t, err, errInvalidTupleType, "A tuple type with %s should be refused", name)
	}
	_ = tupleUnexported{}.a
}

// tupleConcurrent is decoded only by TestUnmarshalTupleConcurrentFirstUse, so its plan is built there
type tupleConcurrent struct {
	ID     uint64            `tuple:"0"`
	Levels []tupleProbeLevel `tuple:"1"`
}

func (c *tupleConcurrent) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, c) }

func TestUnmarshalTupleConcurrentFirstUse(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	results := make([]tupleConcurrent, 32)
	errs := make([]error, len(results))
	for i := range results {
		wg.Go(func() {
			errs[i] = json.Unmarshal([]byte(`[`+strconv.Itoa(i)+`,[[1,0,2]]]`), &results[i])
		})
	}
	wg.Wait()
	for i := range results {
		require.NoErrorf(t, errs[i], "Unmarshal must not error in goroutine %d", i)
		assert.Equalf(t, tupleConcurrent{ID: uint64(i), Levels: []tupleProbeLevel{{Price: 1, Amount: 2}}}, results[i], "Goroutine %d should decode its tuple", i)
	}
}

func TestDecodeTimeAndBoolean(t *testing.T) {
	t.Parallel()
	for _, data := range []string{`1700000000123`, `"1700000000123"`, `1575287438.515`, `1700000000`, `0`, `1700000000123456`} {
		var want types.Time
		require.NoErrorf(t, want.UnmarshalJSON([]byte(data)), "types.Time must decode %s", data)
		var got struct {
			Time types.Time `tuple:"0"`
		}
		require.NoErrorf(t, unmarshalTuple([]byte("["+data+"]"), &got), "unmarshalTuple must decode time %s", data)
		assert.Truef(t, want.Time().Equal(got.Time.Time()), "unmarshalTuple should decode time %s as types.Time does", data)
	}
	for data, want := range map[string]types.Boolean{`1`: true, `0`: false, `"1"`: true, `true`: true, `false`: false} {
		var got struct {
			Boolean types.Boolean `tuple:"0"`
		}
		require.NoErrorf(t, unmarshalTuple([]byte("["+data+"]"), &got), "unmarshalTuple must decode boolean %s", data)
		assert.Equalf(t, want, got.Boolean, "unmarshalTuple should decode boolean %s", data)
	}
}

func TestParseUint64(t *testing.T) {
	t.Parallel()
	for data, want := range map[string]uint64{"0": 0, "42": 42, "18446744073709551615": math.MaxUint64} {
		got, ok := parseUint64([]byte(data))
		assert.Truef(t, ok, "parseUint64 should parse %s", data)
		assert.Equalf(t, want, got, "parseUint64 should parse %s", data)
	}
	for _, data := range []string{"", "-1", "1a", "1.5", "18446744073709551616", "999999999999999999999"} {
		_, ok := parseUint64([]byte(data))
		assert.Falsef(t, ok, "parseUint64 should refuse %q", data)
	}
}

func TestMarshalTuple(t *testing.T) {
	t.Parallel()
	out, err := marshalTuple(&tupleProbeKeyed{Symbol: "tBTCUSD", Level: tupleProbeLevel{Price: 1.5, Amount: -2}})
	require.NoError(t, err, "marshalTuple must not error")
	assert.Equal(t, `["tBTCUSD",1.5,null,-2]`, string(out), "marshalTuple should place each field at its position with null between")
	var got tupleProbeKeyed
	require.NoError(t, unmarshalTuple(out, &got), "unmarshalTuple must decode what marshalTuple encodes")
	assert.Equal(t, tupleProbeKeyed{Symbol: "tBTCUSD", Level: tupleProbeLevel{Price: 1.5, Amount: -2}}, got, "A tuple should survive a round trip")
}

// The frames a busy order book sends most, and a full order
var (
	benchmarkBookFrame  = []byte(`[17082,[8744.9,2,0.45603413],10345]`)
	benchmarkOrderTuple = []byte(`[33950998275,null,1573476747887,"tETHUSD",1573476748000,1573476748000,-0.5,-0.5,"LIMIT",null,null,null,0,"ACTIVE",null,null,220,0,0,0,null,null,null,0,1,null,null,null,"BFX",null,null,{"$F7":1}]`)
)

type benchmarkBookUpdate struct {
	ChannelID uint64          `tuple:"0"`
	Level     tupleProbeLevel `tuple:"1"`
	Sequence  uint64          `tuple:"2"`
}

type benchmarkOrder struct {
	ID             uint64          `tuple:"0"`
	GroupID        uint64          `tuple:"1"`
	ClientOrderID  uint64          `tuple:"2"`
	Symbol         string          `tuple:"3"`
	CreatedAt      types.Time      `tuple:"4"`
	UpdatedAt      types.Time      `tuple:"5"`
	Amount         float64         `tuple:"6"`
	OriginalAmount float64         `tuple:"7"`
	Type           string          `tuple:"8"`
	Status         string          `tuple:"13"`
	Price          float64         `tuple:"16"`
	AveragePrice   float64         `tuple:"17"`
	Hidden         types.Boolean   `tuple:"24"`
	Routing        string          `tuple:"28"`
	Meta           json.RawMessage `tuple:"31"`
}

func BenchmarkUnmarshalTuple(b *testing.B) {
	b.Run("book update", func(b *testing.B) {
		for b.Loop() {
			var u benchmarkBookUpdate
			if err := unmarshalTuple(benchmarkBookFrame, &u); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("order", func(b *testing.B) {
		for b.Loop() {
			var o benchmarkOrder
			if err := unmarshalTuple(benchmarkOrderTuple, &o); err != nil {
				b.Fatal(err)
			}
		}
	})
}
