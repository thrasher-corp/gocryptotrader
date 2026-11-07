package json

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errBooleanValue    = errors.New("invalid boolean value")
	errTimestampFormat = errors.New("invalid timestamp format")
	errStopWalk        = errors.New("stop walking")
)

// boolean mirrors types.Boolean, which cannot be imported here without a cycle, so that 1 decodes by its method where a
// bool field would refuse it
type boolean bool

func (b *boolean) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "1", `"1"`, "true", `"true"`:
		*b = true
	case "0", `"0"`, "false", `"false"`:
		*b = false
	default:
		return errBooleanValue
	}
	return nil
}

// millisecondTime mirrors types.Time for the millisecond timestamps exchanges send, as types cannot be imported here
// without a cycle; 0 leaves it as it is
type millisecondTime time.Time

func (m *millisecondTime) UnmarshalJSON(data []byte) error {
	digits := bytes.Trim(data, `"`)
	if len(digits) == 0 || string(digits) == "0" {
		return nil
	}
	ms, err := strconv.ParseInt(string(digits), 10, 64)
	if err != nil || len(digits) != 13 {
		return fmt.Errorf("%w: %s", errTimestampFormat, data)
	}
	*m = millisecondTime(time.UnixMilli(ms))
	return nil
}

// upperCode is a code its method upper-cases, where a string field would keep it as sent
type upperCode string

func (c *upperCode) UnmarshalJSON(data []byte) error {
	var s string
	if err := Unmarshal(data, &s); err != nil {
		return err
	}
	*c = upperCode(strings.ToUpper(s))
	return nil
}

// tupleProbe has a field of every kind the tuple decoder handles, with position 15 left untagged
type tupleProbe struct {
	Float    float64           `tuple:"0"`
	Int      int64             `tuple:"1"`
	Uint     uint64            `tuple:"2"`
	Text     string            `tuple:"3"`
	Flag     bool              `tuple:"4"`
	Time     millisecondTime   `tuple:"5"`
	Boolean  boolean           `tuple:"6"`
	Raw      RawMessage        `tuple:"7"`
	Level    tupleProbeLevel   `tuple:"8"`
	Levels   []tupleProbeLevel `tuple:"9"`
	Number   number            `tuple:"10"`
	Code     upperCode         `tuple:"11"`
	Optional *float64          `tuple:"12"`
	Strings  []string          `tuple:"13"`
	Small    int               `tuple:"14"`
	Last     string            `tuple:"16"`
}

func (p *tupleProbe) UnmarshalJSON(data []byte) error { return UnmarshalTuple(data, p) }

type tupleProbeLevel struct {
	Price  float64 `tuple:"0"`
	Amount float64 `tuple:"2"`
}

func (l *tupleProbeLevel) UnmarshalJSON(data []byte) error { return UnmarshalTuple(data, l) }

// tupleProbeKeyed is a level prefixed with its symbol
type tupleProbeKeyed struct {
	Symbol string          `tuple:"0"`
	Level  tupleProbeLevel `tuple:"1,inline"`
}

func (k *tupleProbeKeyed) UnmarshalJSON(data []byte) error { return UnmarshalTuple(data, k) }

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

func (n *tupleNamedSlice) UnmarshalJSON(data []byte) error { return UnmarshalTuple(data, n) }

// tupleNested is a record holding records of its own type
type tupleNested struct {
	Value    float64       `tuple:"0"`
	Children []tupleNested `tuple:"1"`
}

func (n *tupleNested) UnmarshalJSON(data []byte) error { return UnmarshalTuple(data, n) }

// tupleConcurrent is decoded only by TestUnmarshalTuple's concurrent first use, so its plan is built there
type tupleConcurrent struct {
	ID     uint64            `tuple:"0"`
	Levels []tupleProbeLevel `tuple:"1"`
}

func (c *tupleConcurrent) UnmarshalJSON(data []byte) error { return UnmarshalTuple(data, c) }

// tupleOutOfOrder declares its fields out of position order
type tupleOutOfOrder struct {
	Last  string  `tuple:"2"`
	First float64 `tuple:"0"`
}

func (o *tupleOutOfOrder) UnmarshalJSON(data []byte) error { return UnmarshalTuple(data, o) }

// TupleExportedLevel is exported, so a struct embedding it has an exported field, refused for being embedded rather
// than for being unexported
type TupleExportedLevel struct {
	Price float64 `tuple:"0"`
}

func (l *TupleExportedLevel) UnmarshalJSON(data []byte) error { return UnmarshalTuple(data, l) }

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
	tupleEmbeddedExportedInline struct {
		TupleExportedLevel `tuple:"1,inline"`
	}
	tupleScalarInline struct {
		A int64 `tuple:"1,inline"`
	}
	// tupleInlineBadPosition has a valid field of its own, so it is refused for its inline struct's tag rather than for
	// having no tagged fields
	tupleInlineBadPosition struct {
		A     int64            `tuple:"0"`
		Inner tupleBadPosition `tuple:"1,inline"`
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
	tupleSharedPositionMethod struct {
		A int64 `tuple:"1"`
		B int64 `tuple:"1"`
	}
	tupleNestsSharedPosition struct {
		Nested tupleSharedPositionMethod `tuple:"0"`
	}
	tupleNestsSharedPositions struct {
		Nested []tupleSharedPosition `tuple:"0"`
	}
)

func (s *tupleSharedPositionMethod) UnmarshalJSON(data []byte) error { return UnmarshalTuple(data, s) }

func TestUnmarshalTuple(t *testing.T) {
	t.Parallel()

	t.Run("every kind, skipping untagged and appended elements", func(t *testing.T) {
		t.Parallel()
		var got tupleProbe
		require.NoError(t, Unmarshal([]byte(` [ 1.5, -2, 18446744073709551615, "a \"quoted\" é", true, 1700000000123, 1,
		{"a":[1,"]"]}, [10.5, 3, 0.25], [[1,null,2], null, [3]], "4.5", "ust", 6.5, ["x","y"], 7, {"skipped":true},
		"last", "appended", [1,2] ] `), &got), "Unmarshal must not error")
		optional := 6.5
		assert.Equal(t, tupleProbe{
			Float:    1.5,
			Int:      -2,
			Uint:     math.MaxUint64,
			Text:     `a "quoted" é`,
			Flag:     true,
			Time:     millisecondTime(time.UnixMilli(1700000000123)),
			Boolean:  true,
			Raw:      RawMessage(`{"a":[1,"]"]}`),
			Level:    tupleProbeLevel{Price: 10.5, Amount: 0.25},
			Levels:   []tupleProbeLevel{{Price: 1, Amount: 2}, {}, {Price: 3}},
			Number:   4.5,
			Code:     "UST",
			Optional: &optional,
			Strings:  []string{"x", "y"},
			Small:    7,
			Last:     "last",
		}, got, "Unmarshal should decode every kind, skipping untagged and appended elements")
	})

	t.Run("null elements leave fields unset", func(t *testing.T) {
		t.Parallel()
		got := tupleProbe{Float: 9, Text: "kept"}
		require.NoError(t, Unmarshal([]byte(`[null,null,null,null,null,null,null,null,null,null,null,null,null]`), &got), "Unmarshal must not error for null elements")
		assert.Equal(t, tupleProbe{Float: 9, Text: "kept"}, got, "Null elements should leave their fields unset")
	})

	t.Run("null elements decoded into again set pointers and slices to nil", func(t *testing.T) {
		t.Parallel()
		// As Unmarshal does, null sets a pointer or slice decoded into again to nil, and leaves anything else as it is
		optional := 6.5
		reused := tupleProbe{Float: 9, Raw: RawMessage(`{}`), Level: tupleProbeLevel{Price: 2}, Levels: []tupleProbeLevel{{Price: 1}}, Optional: &optional, Strings: []string{"x"}}
		require.NoError(t, Unmarshal([]byte(`[null,null,null,null,null,null,null,null,null,null,null,null,null,null]`), &reused), "Unmarshal must not error for null elements decoded into again")
		assert.Equal(t, tupleProbe{Float: 9, Level: tupleProbeLevel{Price: 2}}, reused, "Null elements should set pointers and slices to nil, leaving other fields as they were")
	})

	for _, data := range []string{`[1.5]`, `[1.5 ]`} {
		t.Run("short array "+data, func(t *testing.T) {
			t.Parallel()
			var short tupleProbe
			require.NoErrorf(t, Unmarshal([]byte(data), &short), "Unmarshal must not error for short array %s", data)
			assert.Equalf(t, tupleProbe{Float: 1.5}, short, "A short array %s should set only its elements' fields", data)
		})
	}

	t.Run("empty array", func(t *testing.T) {
		t.Parallel()
		var empty tupleProbe
		require.NoError(t, UnmarshalTuple([]byte(`[ ]`), &empty), "UnmarshalTuple must not error for an empty array")
		assert.Zero(t, empty, "An empty array should set no field")
	})

	t.Run("null tuple", func(t *testing.T) {
		t.Parallel()
		unchanged := tupleProbe{Text: "kept"}
		require.NoError(t, UnmarshalTuple([]byte(` null `), &unchanged), "UnmarshalTuple must not error for null")
		assert.Equal(t, tupleProbe{Text: "kept"}, unchanged, "A null tuple should leave the struct unset")
	})

	t.Run("inline levels", func(t *testing.T) {
		t.Parallel()
		var keyed []tupleProbeKeyed
		require.NoError(t, Unmarshal([]byte(`[["tBTCUSD",1,2,3],["fUSD",4]]`), &keyed), "Unmarshal must not error for inline levels")
		assert.Equal(t, []tupleProbeKeyed{{Symbol: "tBTCUSD", Level: tupleProbeLevel{Price: 1, Amount: 3}}, {Symbol: "fUSD", Level: tupleProbeLevel{Price: 4}}}, keyed, "Inline levels should decode from the position after their symbol")
	})

	t.Run("quoted numbers", func(t *testing.T) {
		t.Parallel()
		var quoted tupleProbe
		require.NoError(t, UnmarshalTuple([]byte(`["1.5","-2","3"]`), &quoted), "UnmarshalTuple must not error for quoted numbers")
		assert.Equal(t, tupleProbe{Float: 1.5, Int: -2, Uint: 3}, quoted, "Quoted numbers should decode as numbers")
	})

	t.Run("empty strings leave numbers unset", func(t *testing.T) {
		t.Parallel()
		blank := tupleProbe{Float: 9, Int: 9, Uint: 9}
		require.NoError(t, UnmarshalTuple([]byte(`["","",""]`), &blank), "UnmarshalTuple must not error for empty strings")
		assert.Equal(t, tupleProbe{Float: 9, Int: 9, Uint: 9}, blank, "Empty strings should leave number fields unset")
	})

	t.Run("reused slice", func(t *testing.T) {
		t.Parallel()
		reusedSlice := tupleProbe{Levels: make([]tupleProbeLevel, 2, 4)}
		reusedSlice.Levels[0] = tupleProbeLevel{Price: 8, Amount: 8}
		require.NoError(t, UnmarshalTuple([]byte(`[0,0,0,"",false,0,0,null,null,[[1,null]]]`), &reusedSlice), "UnmarshalTuple must not error into a reused slice")
		assert.Equal(t, []tupleProbeLevel{{Price: 1}}, reusedSlice.Levels, "A reused slice's elements should not keep earlier values")
	})

	t.Run("empty array empties a reused slice", func(t *testing.T) {
		t.Parallel()
		reused := tupleProbe{Levels: []tupleProbeLevel{{Price: 8}, {Price: 9}}}
		require.NoError(t, UnmarshalTuple([]byte(`[0,0,0,"",false,0,0,null,null,[]]`), &reused), "UnmarshalTuple must not error for an empty array into a reused slice")
		assert.Empty(t, reused.Levels, "An empty array should leave a reused slice empty")
	})

	for data, want := range map[string]boolean{`1`: true, `0`: false, `"1"`: true, `true`: true, `false`: false} {
		t.Run("boolean "+data+" decodes by its method", func(t *testing.T) {
			t.Parallel()
			got := tupleProbe{Boolean: !want}
			require.NoErrorf(t, UnmarshalTuple([]byte(`[0,0,0,"",false,0,`+data+`]`), &got), "UnmarshalTuple must decode boolean %s", data)
			assert.Equalf(t, want, got.Boolean, "UnmarshalTuple should decode boolean %s as its method does", data)
		})
	}

	for _, data := range []string{`1700000000123`, `"1700000000123"`} {
		t.Run("timestamp "+data+" decodes by its method", func(t *testing.T) {
			t.Parallel()
			var want millisecondTime
			require.NoErrorf(t, want.UnmarshalJSON([]byte(data)), "millisecondTime must decode %s", data)
			var got tupleProbe
			require.NoErrorf(t, UnmarshalTuple([]byte(`[0,0,0,"",false,`+data+`]`), &got), "UnmarshalTuple must decode timestamp %s", data)
			assert.Truef(t, time.Time(want).Equal(time.Time(got.Time)), "UnmarshalTuple should decode timestamp %s as its method does", data)
		})
	}

	t.Run("false clears a flag decoded into again", func(t *testing.T) {
		t.Parallel()
		reused := tupleProbe{Flag: true}
		require.NoError(t, UnmarshalTuple([]byte(`[0,0,0,"",false]`), &reused), "UnmarshalTuple must not error for false")
		assert.False(t, reused.Flag, "false should clear a flag decoded into again")
	})

	t.Run("fields declared out of position order", func(t *testing.T) {
		t.Parallel()
		var got tupleOutOfOrder
		require.NoError(t, UnmarshalTuple([]byte(`[1.5,null,"last"]`), &got), "UnmarshalTuple must not error for fields declared out of position order")
		assert.Equal(t, tupleOutOfOrder{Last: "last", First: 1.5}, got, "Fields declared out of position order should decode at their positions")
	})

	t.Run("named slice type decodes by its own method", func(t *testing.T) {
		t.Parallel()
		var got tupleNamedSlice
		require.NoError(t, Unmarshal([]byte(`[[1,0,2]]`), &got), "Unmarshal must not error for a named slice type")
		assert.Equal(t, tupleNamedSlice{Levels: tupleProbeLevels{{Price: 1, Amount: 2}}}, got, "A named slice type should decode by its own method rather than as a slice of tuples")
	})

	t.Run("concurrent first use", func(t *testing.T) {
		t.Parallel()
		var wg sync.WaitGroup
		results := make([]tupleConcurrent, 32)
		errs := make([]error, len(results))
		for i := range results {
			wg.Go(func() {
				errs[i] = Unmarshal([]byte(`[`+strconv.Itoa(i)+`,[[1,0,2]]]`), &results[i])
			})
		}
		wg.Wait()
		for i := range results {
			require.NoErrorf(t, errs[i], "Unmarshal must not error in goroutine %d", i)
			assert.Equalf(t, tupleConcurrent{ID: uint64(i), Levels: []tupleProbeLevel{{Price: 1, Amount: 2}}}, results[i], "Goroutine %d should decode its tuple", i)
		}
	})

	for _, tc := range []struct {
		name, data, field string
		err               error
	}{
		{name: "object", data: `{"a":1}`, err: ErrTupleNotArray},
		{name: "string", data: `"text"`, err: ErrTupleNotArray},
		{name: "empty input", data: ``, err: ErrTupleNotArray},
		{name: "unterminated after a comma", data: `[1.5,`, err: ErrMalformedTuple},
		{name: "unterminated", data: `[1.5`, err: ErrMalformedTuple},
		{name: "missing comma", data: `[1.5 2]`, err: ErrMalformedTuple},
		{name: "trailing comma after the last field", data: `[1.5,-2,3,"a",true,0,1,null,null,null,"4.5","ust",null,null,7,null,"last",]`, err: ErrMalformedTuple},
		{name: "trailing comma", data: `[1.5,]`, err: ErrMalformedTuple},
		{name: "closed by a brace", data: `[1.5,}`, err: ErrMalformedTuple},
		{name: "float from text", data: `["x"]`, field: "Float at position 0", err: ErrTupleElementType},
		{name: "float from NaN", data: `["NaN"]`, field: "Float at position 0", err: ErrTupleElementType},
		{name: "float out of range", data: `["1e400"]`, field: "Float at position 0", err: ErrTupleElementType},
		{name: "float from an escaped string", data: `["1\u0035"]`, field: "Float at position 0", err: ErrTupleElementType},
		{name: "float from a boolean", data: `[true]`, field: "Float at position 0", err: ErrTupleElementType},
		{name: "integer from a fraction", data: `[1,1.5]`, field: "Int at position 1", err: ErrTupleElementType},
		{name: "integer from a quoted fraction", data: `[1,"2.5"]`, field: "Int at position 1", err: ErrTupleElementType},
		{name: "integer from a boolean", data: `[1,true]`, field: "Int at position 1", err: ErrTupleElementType},
		{name: "unsigned integer from a negative", data: `[1,2,-3]`, field: "Uint at position 2", err: ErrTupleElementType},
		{name: "unsigned integer from a quoted negative", data: `[1,2,"-3"]`, field: "Uint at position 2", err: ErrTupleElementType},
		{name: "unsigned integer overflow", data: `[1,2,18446744073709551616]`, field: "Uint at position 2", err: ErrTupleElementType},
		{name: "unsigned integer from a boolean", data: `[1,2,true]`, field: "Uint at position 2", err: ErrTupleElementType},
		{name: "string with a bad escape", data: `[1,2,3,"\x"]`, field: "Text at position 3", err: ErrTupleElementType},
		{name: "malformed slice of tuples", data: `[1,2,3,"a",true,0,1,null,null,[x]]`, field: "Levels at position 9", err: ErrMalformedTuple},
		{name: "string from a number", data: `[1,2,3,4]`, field: "Text at position 3", err: ErrTupleElementType},
		{name: "bool from a number", data: `[1,2,3,"a",1]`, field: "Flag at position 4", err: ErrTupleElementType},
		{name: "field decoded by its method", data: `[1,2,3,"a",true,123]`, field: "Time at position 5", err: errTimestampFormat},
		{name: "nested tuple that is not an array", data: `[1,2,3,"a",true,0,1,null,{}]`, field: "Level at position 8", err: ErrTupleNotArray},
		{name: "slice of tuples that is not an array", data: `[1,2,3,"a",true,0,1,null,null,{}]`, field: "Levels at position 9", err: ErrTupleElementType},
		{name: "slice of tuples with a malformed element", data: `[1,2,3,"a",true,0,1,null,null,[[1],["x"]]]`, field: "Levels at position 9: element 1: json.tupleProbeLevel.Price at position 0", err: ErrTupleElementType},
	} {
		t.Run("error for "+tc.name, func(t *testing.T) {
			t.Parallel()
			var got tupleProbe
			err := UnmarshalTuple([]byte(tc.data), &got)
			assert.ErrorIsf(t, err, tc.err, "UnmarshalTuple should return the expected error for %s", tc.data)
			if tc.field != "" {
				assert.ErrorContainsf(t, err, "json.tupleProbe."+tc.field, "UnmarshalTuple should name the field for %s", tc.data)
			}
		})
	}

	data := []byte(`[1,2]`)
	for _, tc := range []struct {
		name   string
		decode func() error
	}{
		{name: "shared position", decode: func() error { return UnmarshalTuple(data, &tupleSharedPosition{}) }},
		{name: "bad position", decode: func() error { return UnmarshalTuple(data, &tupleBadPosition{}) }},
		{name: "negative position", decode: func() error { return UnmarshalTuple(data, &tupleNegativePosition{}) }},
		{name: "unexported field", decode: func() error { return UnmarshalTuple(data, &tupleUnexported{}) }},
		{name: "unknown option", decode: func() error { return UnmarshalTuple(data, &tupleUnknownOption{}) }},
		{name: "embedded inline", decode: func() error { return UnmarshalTuple(data, &tupleEmbeddedInline{}) }},
		{name: "embedded exported inline", decode: func() error { return UnmarshalTuple(data, &tupleEmbeddedExportedInline{}) }},
		{name: "scalar inline", decode: func() error { return UnmarshalTuple(data, &tupleScalarInline{}) }},
		{name: "inline struct with a bad position", decode: func() error { return UnmarshalTuple(data, &tupleInlineBadPosition{}) }},
		{name: "no tags", decode: func() error { return UnmarshalTuple(data, &tupleUntagged{}) }},
		{name: "not a struct", decode: func() error { return UnmarshalTuple(data, new(int64)) }},
		{name: "nested tuple without UnmarshalJSON", decode: func() error { return UnmarshalTuple(data, &tupleNestsNoMethod{}) }},
		{name: "nested tuple with a shared position", decode: func() error { return UnmarshalTuple(data, &tupleNestsSharedPosition{}) }},
		{name: "slice of tuples with a shared position", decode: func() error { return UnmarshalTuple(data, &tupleNestsSharedPositions{}) }},
	} {
		t.Run("invalid type with "+tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.decode(), ErrInvalidTupleType, "A tuple type with %s should be refused", tc.name)
		})
	}
	_ = tupleUnexported{}.a
}

func TestMarshalTuple(t *testing.T) {
	t.Parallel()

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()
		out, err := MarshalTuple(&tupleProbeKeyed{Symbol: "tBTCUSD", Level: tupleProbeLevel{Price: 1.5, Amount: -2}})
		require.NoError(t, err, "MarshalTuple must not error")
		assert.Equal(t, `["tBTCUSD",1.5,null,-2]`, string(out), "MarshalTuple should place each field at its position with null between")
		var got tupleProbeKeyed
		require.NoError(t, UnmarshalTuple(out, &got), "UnmarshalTuple must decode what MarshalTuple encodes")
		assert.Equal(t, tupleProbeKeyed{Symbol: "tBTCUSD", Level: tupleProbeLevel{Price: 1.5, Amount: -2}}, got, "A tuple should survive a round trip")
	})

	t.Run("invalid type with no tags", func(t *testing.T) {
		t.Parallel()
		_, err := MarshalTuple(&tupleUntagged{})
		assert.ErrorIs(t, err, ErrInvalidTupleType, "MarshalTuple should refuse a type with no tags")
	})

	t.Run("invalid type that is not a struct", func(t *testing.T) {
		t.Parallel()
		_, err := MarshalTuple(new(string))
		assert.ErrorIs(t, err, ErrInvalidTupleType, "MarshalTuple should refuse a type that is not a struct")
	})
}

func TestFirstArrayElement(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, data, exp string
		dataType        jsonparser.ValueType
	}{
		{name: "number", data: `[1,"hb",2]`, exp: `1`, dataType: jsonparser.Number},
		{name: "string after spaces", data: ` [ "error", 10020, "symbol: invalid"]`, exp: `error`, dataType: jsonparser.String},
		{name: "array", data: `[[1,2],3]`, exp: `[1,2]`, dataType: jsonparser.Array},
		{name: "null", data: `[null]`, exp: `null`, dataType: jsonparser.Null},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, dataType, err := FirstArrayElement([]byte(tc.data))
			require.NoErrorf(t, err, "FirstArrayElement must not error for %s", tc.data)
			assert.Equalf(t, tc.exp, string(got), "FirstArrayElement should return %s's first element", tc.data)
			assert.Equalf(t, tc.dataType, dataType, "FirstArrayElement should return the type of %s's first element", tc.data)
		})
	}
	for _, data := range []string{`{"a":1}`, `1`, ``, ` `} {
		t.Run("not an array "+strconv.Quote(data), func(t *testing.T) {
			t.Parallel()
			_, _, err := FirstArrayElement([]byte(data))
			assert.ErrorIsf(t, err, ErrTupleNotArray, "FirstArrayElement should refuse %q, which is not an array", data)
		})
	}
	for _, data := range []string{`[]`, `[ ]`, `[`, `[,1]`} {
		t.Run("no first element "+strconv.Quote(data), func(t *testing.T) {
			t.Parallel()
			_, _, err := FirstArrayElement([]byte(data))
			assert.ErrorIsf(t, err, ErrMalformedTuple, "FirstArrayElement should refuse %q, which has no first element", data)
		})
	}
}

func TestEachArrayElement(t *testing.T) {
	t.Parallel()

	type element struct {
		raw      string
		dataType jsonparser.ValueType
	}
	walk := func(data string) ([]element, error) {
		var got []element
		err := EachArrayElement([]byte(data), func(n int, raw []byte, dataType jsonparser.ValueType) error {
			if n != len(got) {
				return fmt.Errorf("element %d numbered %d", len(got), n)
			}
			got = append(got, element{raw: string(raw), dataType: dataType})
			return nil
		})
		return got, err
	}

	t.Run("each element as sent", func(t *testing.T) {
		t.Parallel()
		got, err := walk(` [ 1 , "a\"b" ,[2, 3], {"c":[4]}, null,true ] `)
		require.NoError(t, err, "EachArrayElement must not error")
		assert.Equal(t, []element{
			{raw: `1`, dataType: jsonparser.Number},
			{raw: `"a\"b"`, dataType: jsonparser.String},
			{raw: `[2, 3]`, dataType: jsonparser.Array},
			{raw: `{"c":[4]}`, dataType: jsonparser.Object},
			{raw: `null`, dataType: jsonparser.Null},
			{raw: `true`, dataType: jsonparser.Boolean},
		}, got, "EachArrayElement should call fn with each element as sent, in order")
	})

	for _, data := range []string{`[]`, ` [ ] `} {
		t.Run("empty array "+strconv.Quote(data), func(t *testing.T) {
			t.Parallel()
			got, err := walk(data)
			require.NoErrorf(t, err, "EachArrayElement must not error for %q", data)
			assert.Emptyf(t, got, "EachArrayElement should call fn for no element of %q", data)
		})
	}

	t.Run("stops at fn's error", func(t *testing.T) {
		t.Parallel()
		var calls int
		err := EachArrayElement([]byte(`[1,2,3]`), func(n int, _ []byte, _ jsonparser.ValueType) error {
			calls++
			if n == 1 {
				return errStopWalk
			}
			return nil
		})
		assert.ErrorIs(t, err, errStopWalk, "EachArrayElement should return fn's error")
		assert.Equal(t, 2, calls, "EachArrayElement should stop calling fn once it errors")
	})

	for _, data := range []string{`{"a":1}`, `1`, ``, ` `} {
		t.Run("not an array "+strconv.Quote(data), func(t *testing.T) {
			t.Parallel()
			_, err := walk(data)
			assert.ErrorIsf(t, err, ErrTupleNotArray, "EachArrayElement should refuse %q, which is not an array", data)
		})
	}

	for _, data := range []string{`[`, `[1,`, `[1`, `[1 2]`, `[1,]`, `[1,}`, `[,1]`} {
		t.Run("malformed "+strconv.Quote(data), func(t *testing.T) {
			t.Parallel()
			_, err := walk(data)
			assert.ErrorIsf(t, err, ErrMalformedTuple, "EachArrayElement should refuse %q, which is malformed", data)
		})
	}
}

func TestBuildTuplePlan(t *testing.T) {
	t.Parallel()

	t.Run("a type nesting itself", func(t *testing.T) {
		t.Parallel()
		nested, err := buildTuplePlan(reflect.TypeFor[tupleNested](), make(map[reflect.Type]*tuplePlan))
		require.NoError(t, err, "buildTuplePlan must not error for a type nesting itself")
		require.Len(t, nested.fields, 2, "buildTuplePlan must plan both of a type nesting itself's fields")
		var tree tupleNested
		require.NoError(t, Unmarshal([]byte(`[1,[[2,[]],[3,null]]]`), &tree), "Unmarshal must not error for a type nesting itself")
		assert.Equal(t, tupleNested{Value: 1, Children: []tupleNested{{Value: 2}, {Value: 3}}}, tree, "A type nesting itself should decode through its own plan")
	})

	t.Run("fields in position order", func(t *testing.T) {
		t.Parallel()
		p, err := buildTuplePlan(reflect.TypeFor[tupleProbe](), make(map[reflect.Type]*tuplePlan))
		require.NoError(t, err, "buildTuplePlan must not error")
		positions := make([]int, len(p.fields))
		nilable := make(map[string]bool, len(p.fields))
		for i := range p.fields {
			positions[i] = p.fields[i].position
			nilable[strings.TrimPrefix(p.fields[i].name, "json.tupleProbe.")] = p.fields[i].nilable
		}
		assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 16}, positions, "buildTuplePlan should order the fields by position")
		assert.Equal(t, map[string]bool{
			"Float": false, "Int": false, "Uint": false, "Text": false, "Flag": false, "Time": false, "Boolean": false, "Raw": true, "Level": false,
			"Levels": true, "Number": false, "Code": false, "Optional": true, "Strings": true, "Small": false, "Last": false,
		}, nilable, "buildTuplePlan should mark the pointer and slice fields, which null sets to nil")
	})

	for name, typ := range map[string]reflect.Type{
		"a shared position":                    reflect.TypeFor[tupleSharedPosition](),
		"no tags":                              reflect.TypeFor[tupleUntagged](),
		"no struct":                            reflect.TypeFor[int64](),
		"a nested tuple without UnmarshalJSON": reflect.TypeFor[tupleNestsNoMethod](),
	} {
		t.Run("refuses a type with "+name, func(t *testing.T) {
			t.Parallel()
			_, err := buildTuplePlan(typ, make(map[reflect.Type]*tuplePlan))
			assert.ErrorIsf(t, err, ErrInvalidTupleType, "buildTuplePlan should refuse a type with %s", name)
		})
	}
}

func TestTuplePlanDecode(t *testing.T) {
	t.Parallel()

	t.Run("level", func(t *testing.T) {
		t.Parallel()
		level, err := tuplePlanOf(reflect.TypeFor[tupleProbeLevel]())
		require.NoError(t, err, "tuplePlanOf must not error")
		got := tupleProbeLevel{Price: 9, Amount: 9}
		require.NoError(t, level.decode([]byte(`[1,null,null]`), unsafe.Pointer(&got)), "decode must not error")
		assert.Equal(t, tupleProbeLevel{Price: 1, Amount: 9}, got, "decode should set the fields of the elements given, and leave a number null leaves as it was")
		require.NoError(t, level.decode([]byte(` null `), unsafe.Pointer(&got)), "decode must not error for null")
		assert.Equal(t, tupleProbeLevel{Price: 1, Amount: 9}, got, "decode should leave the struct as it was for null")
		assert.ErrorIs(t, level.decode([]byte(`{}`), unsafe.Pointer(&got)), ErrTupleNotArray, "decode should refuse an object")
		assert.ErrorIs(t, level.decode([]byte(`[1,`), unsafe.Pointer(&got)), ErrMalformedTuple, "decode should refuse a malformed array")
		assert.ErrorIs(t, level.decode([]byte(`["x"]`), unsafe.Pointer(&got)), ErrTupleElementType, "decode should refuse an element of the wrong type")
	})

	t.Run("null elements decoded into again", func(t *testing.T) {
		t.Parallel()
		probe, err := tuplePlanOf(reflect.TypeFor[tupleProbe]())
		require.NoError(t, err, "tuplePlanOf must not error")
		optional := 1.0
		reused := tupleProbe{Raw: RawMessage(`{}`), Levels: []tupleProbeLevel{{Price: 1}}, Optional: &optional, Strings: []string{"x"}}
		require.NoError(t, probe.decode([]byte(`[null,null,null,null,null,null,null,null,null,null,null,null,null,null]`), unsafe.Pointer(&reused)), "decode must not error for null elements")
		assert.Zero(t, reused, "decode should set the pointers and slices of null elements to nil")
	})
}

func TestParseUint64(t *testing.T) {
	t.Parallel()
	for data, want := range map[string]uint64{"0": 0, "42": 42, "18446744073709551615": math.MaxUint64} {
		t.Run(data, func(t *testing.T) {
			t.Parallel()
			got, ok := parseUint64([]byte(data))
			assert.Truef(t, ok, "parseUint64 should parse %s", data)
			assert.Equalf(t, want, got, "parseUint64 should parse %s", data)
		})
	}
	for _, data := range []string{"", "-1", "1a", "1.5", "18446744073709551616", "999999999999999999999"} {
		t.Run("refuses "+strconv.Quote(data), func(t *testing.T) {
			t.Parallel()
			_, ok := parseUint64([]byte(data))
			assert.Falsef(t, ok, "parseUint64 should refuse %q", data)
		})
	}
}
