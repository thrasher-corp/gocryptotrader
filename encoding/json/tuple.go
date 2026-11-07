package json

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/buger/jsonparser"
)

// Tuple errors
var (
	// ErrTupleNotArray is returned for a tuple, or an array FirstArrayElement or EachArrayElement reads, that is not a
	// JSON array
	ErrTupleNotArray = errors.New("tuple must be a JSON array")
	// ErrMalformedTuple is returned for an array that is not well formed
	ErrMalformedTuple = errors.New("malformed tuple")
	// ErrInvalidTupleType is returned for a type whose tuple tags cannot be decoded
	ErrInvalidTupleType = errors.New("invalid tuple type")
	// ErrTupleElementType is returned for an element of a type its field cannot hold
	ErrTupleElementType = errors.New("unexpected tuple element type")
)

// elementDecoder decodes an array element into a field, at base plus the field's offset. value is the element with any
// string's quotes removed, raw the element as sent
type elementDecoder func(base unsafe.Pointer, value, raw []byte, dataType jsonparser.ValueType) error

// tupleField is a tagged field and its array position
type tupleField struct {
	name     string
	position int
	index    []int
	typ      reflect.Type
	offset   uintptr
	decode   elementDecoder
	// nilable is whether null sets the field to nil: a pointer, slice, map or interface
	nilable bool
}

// tuplePlan holds a struct type's tagged fields in position order
type tuplePlan struct {
	name   string
	fields []tupleField
}

var (
	// tuplePlans holds every completed plan by its struct type
	tuplePlans sync.Map
	// tuplePlansMu serialises plan building, so a plan is only published once complete, with the plans it nests
	tuplePlansMu sync.Mutex

	rawMessageType  = reflect.TypeFor[RawMessage]()
	unmarshalerType = reflect.TypeFor[interface{ UnmarshalJSON([]byte) error }]()
)

// UnmarshalTuple decodes a JSON array into the struct v points to, at the positions its tuple tags give. Some exchanges
// send records as arrays whose positions carry the fields, and append fields to them without notice. A struct
// describes such a record with a tag giving each field's position, `tuple:"16"`, and decodes it with one line:
//
//	func (o *Order) UnmarshalJSON(data []byte) error { return json.UnmarshalTuple(data, o) }
//
// `tuple:"1,inline"` on a named struct field decodes that struct's fields from position 1 on, for records that start
// with a symbol or key. A struct field whose type has tuple tags is decoded through them, and the type needs an
// UnmarshalJSON method all the same, so it decodes alone too; a slice of such structs is decoded through their tags
// as well, unless the slice's own type has an UnmarshalJSON method. Any other field whose type has an UnmarshalJSON
// method is decoded by it, and a field of a kind the decoder does not parse itself, such as int, whose size varies, by
// Unmarshal. A float64, int64 or uint64 field also takes a number quoted in a string, as some exchanges send them, and
// an empty string leaves it as it is, as null does.
//
// Positions no field takes are skipped, decoding stops once the last field is set, and a null element leaves its field
// as it is, but for a pointer, slice, map or interface, which it sets to nil, RawMessage included, so a struct decoded
// into again keeps none of what those held. A null tuple leaves the struct as it is.
//
// Each type's tags are compiled once into a decoder per field, which writes the element straight into the field at its
// offset, so a tuple decodes in a single pass. The fields the decoder parses itself, those of nested tuples included,
// decode the same way whichever JSON implementation the build uses, while those left to Unmarshal follow the build's
// implementation.
func UnmarshalTuple[T any](data []byte, v *T) error {
	p, err := tuplePlanOf(reflect.TypeFor[T]())
	if err != nil {
		return err
	}
	return p.decode(data, unsafe.Pointer(v))
}

// MarshalTuple encodes the struct v points to as a JSON array, with each field at its tuple position and null between
func MarshalTuple[T any](v *T) ([]byte, error) {
	p, err := tuplePlanOf(reflect.TypeFor[T]())
	if err != nil {
		return nil, err
	}
	elements := make([]any, p.fields[len(p.fields)-1].position+1)
	source := reflect.ValueOf(v).Elem()
	for i := range p.fields {
		elements[p.fields[i].position] = source.FieldByIndex(p.fields[i].index).Interface()
	}
	return Marshal(elements)
}

// FirstArrayElement returns a JSON array's first element and its type, with a string's quotes removed. It reads the
// element directly, as a jsonparser index lookup walks every element of the array
func FirstArrayElement(data []byte) ([]byte, jsonparser.ValueType, error) {
	i := skipSpace(data, 0)
	if i == len(data) || data[i] != '[' {
		return nil, jsonparser.NotExist, ErrTupleNotArray
	}
	value, dataType, _, err := jsonparser.Get(data[i+1:])
	if err != nil {
		return nil, dataType, fmt.Errorf("%w: %w", ErrMalformedTuple, err)
	}
	return value, dataType, nil
}

// EachArrayElement calls fn with each element of a JSON array as sent, stopping at the first error fn returns or the
// first malformed element
func EachArrayElement(data []byte, fn func(n int, raw []byte, dataType jsonparser.ValueType) error) error {
	i := skipSpace(data, 0)
	if i == len(data) || data[i] != '[' {
		return ErrTupleNotArray
	}
	if i = skipSpace(data, i+1); i < len(data) && data[i] == ']' {
		return nil
	}
	for n := 0; ; n++ {
		_, dataType, end, err := jsonparser.Get(data[i:])
		if err != nil {
			return fmt.Errorf("%w: %w", ErrMalformedTuple, err)
		}
		if err := fn(n, data[i:i+end], dataType); err != nil {
			return err
		}
		if i, err = nextElement(data, i+end); err != nil || i == len(data) {
			return err
		}
	}
}

// tuplePlanOf returns the plan of a struct type, building it, and any it nests, on first use
func tuplePlanOf(t reflect.Type) (*tuplePlan, error) {
	if p, ok := loadTuplePlan(t); ok {
		return p, nil
	}
	tuplePlansMu.Lock()
	defer tuplePlansMu.Unlock()
	// buildTuplePlan returns a plan published while this waited for the lock as it is
	building := make(map[reflect.Type]*tuplePlan)
	p, err := buildTuplePlan(t, building)
	if err != nil {
		return nil, err
	}
	for typ, built := range building {
		tuplePlans.Store(typ, built)
	}
	return p, nil
}

// loadTuplePlan returns the published plan of a struct type
func loadTuplePlan(t reflect.Type) (*tuplePlan, bool) {
	stored, ok := tuplePlans.Load(t)
	if !ok {
		return nil, false
	}
	p, ok := stored.(*tuplePlan)
	return p, ok
}

// buildTuplePlan builds the plan of t, recording it and the plans it nests in building until all are complete
func buildTuplePlan(t reflect.Type, building map[reflect.Type]*tuplePlan) (*tuplePlan, error) {
	if p, ok := loadTuplePlan(t); ok {
		return p, nil
	}
	if p, ok := building[t]; ok {
		// A type nesting itself decodes through this plan once it is complete
		return p, nil
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: %s is not a struct", ErrInvalidTupleType, t)
	}
	p := &tuplePlan{name: t.String()}
	building[t] = p
	if err := p.addFields(t, nil, 0, 0); err != nil {
		return nil, err
	}
	if len(p.fields) == 0 {
		return nil, fmt.Errorf("%w: %s has no tuple tags", ErrInvalidTupleType, t)
	}
	slices.SortFunc(p.fields, func(a, b tupleField) int { return a.position - b.position })
	for i := range p.fields {
		if i > 0 && p.fields[i].position == p.fields[i-1].position {
			return nil, fmt.Errorf("%w: %s and %s share position %d", ErrInvalidTupleType, p.fields[i-1].name, p.fields[i].name, p.fields[i].position)
		}
		decode, err := newElementDecoder(p.fields[i].typ, p.fields[i].offset, building)
		if err != nil {
			return nil, err
		}
		p.fields[i].decode = decode
		switch p.fields[i].typ.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
			p.fields[i].nilable = true
		}
	}
	return p, nil
}

// addFields adds the tagged fields of t, offset by an inline field's position and memory offset
func (p *tuplePlan) addFields(t reflect.Type, index []int, position int, offset uintptr) error {
	for i := range t.NumField() {
		f := t.Field(i)
		tag, ok := f.Tag.Lookup("tuple")
		if !ok {
			continue
		}
		positionText, option, _ := strings.Cut(tag, ",")
		fieldPosition, err := strconv.Atoi(positionText)
		if err != nil || fieldPosition < 0 {
			return fmt.Errorf("%w: %s.%s has tag %q, want a position", ErrInvalidTupleType, t, f.Name, tag)
		}
		if !f.IsExported() {
			return fmt.Errorf("%w: %s.%s is unexported", ErrInvalidTupleType, t, f.Name)
		}
		fieldIndex := append(slices.Clip(index), i)
		switch option {
		case "":
			p.fields = append(p.fields, tupleField{
				name:     t.String() + "." + f.Name,
				position: position + fieldPosition,
				index:    fieldIndex,
				typ:      f.Type,
				offset:   offset + f.Offset,
			})
		case "inline":
			// An embedded struct would promote its UnmarshalJSON to the outer type, which would then decode the whole
			// array as the inner struct, shifted, without error
			if f.Anonymous || f.Type.Kind() != reflect.Struct {
				return fmt.Errorf("%w: inline %s.%s must be a named struct field", ErrInvalidTupleType, t, f.Name)
			}
			if err := p.addFields(f.Type, fieldIndex, position+fieldPosition, offset+f.Offset); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: %s.%s has unknown option %q", ErrInvalidTupleType, t, f.Name, option)
		}
	}
	return nil
}

// hasTupleTags reports whether a struct type has a tuple tagged field
func hasTupleTags(t reflect.Type) bool {
	for f := range t.Fields() {
		if _, ok := f.Tag.Lookup("tuple"); ok {
			return true
		}
	}
	return false
}

// newElementDecoder returns the decoder writing an element into a field of typ at offset. RawMessage takes the element
// as sent. A struct with tuple tags decodes through its plan, and is refused without an UnmarshalJSON method of its own;
// any other type with an UnmarshalJSON method decodes by it, a slice of tuple structs without one through the structs'
// plan, and float64, int64, uint64, string and bool kinds are parsed and written directly. Anything else is left to
// Unmarshal
func newElementDecoder(typ reflect.Type, offset uintptr, building map[reflect.Type]*tuplePlan) (elementDecoder, error) {
	switch {
	case typ == rawMessageType:
		return func(base unsafe.Pointer, _, raw []byte, _ jsonparser.ValueType) error {
			*(*RawMessage)(unsafe.Add(base, offset)) = bytes.Clone(raw)
			return nil
		}, nil
	case typ.Kind() == reflect.Struct && hasTupleTags(typ) && !reflect.PointerTo(typ).Implements(unmarshalerType):
		return nil, fmt.Errorf("%w: %s has tuple tags but no UnmarshalJSON method", ErrInvalidTupleType, typ)
	case typ.Kind() == reflect.Struct && hasTupleTags(typ):
		sub, err := buildTuplePlan(typ, building)
		if err != nil {
			return nil, err
		}
		return func(base unsafe.Pointer, _, raw []byte, _ jsonparser.ValueType) error {
			return sub.decode(raw, unsafe.Add(base, offset))
		}, nil
	case reflect.PointerTo(typ).Implements(unmarshalerType):
		// Checked before a slice of tuples, so a named slice type decodes by its own method
		return func(base unsafe.Pointer, _, raw []byte, _ jsonparser.ValueType) error {
			// The field's address is a pointer to typ, which implements UnmarshalJSON, so the assertion holds
			u, _ := reflect.TypeAssert[interface{ UnmarshalJSON([]byte) error }](reflect.NewAt(typ, unsafe.Add(base, offset)))
			return u.UnmarshalJSON(raw)
		}, nil
	case typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Struct && hasTupleTags(typ.Elem()):
		sub, err := buildTuplePlan(typ.Elem(), building)
		if err != nil {
			return nil, err
		}
		return decodeTupleSlice(typ, offset, sub), nil
	}
	switch typ.Kind() {
	case reflect.Float64:
		return func(base unsafe.Pointer, value, raw []byte, dataType jsonparser.ValueType) error {
			if !isNumeric(value, dataType) {
				return fmt.Errorf("%w: want a number, got %s", ErrTupleElementType, raw)
			}
			if len(value) == 0 {
				return nil
			}
			v, err := strconv.ParseFloat(unsafe.String(unsafe.SliceData(value), len(value)), 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("%w: want a number, got %s", ErrTupleElementType, raw)
			}
			*(*float64)(unsafe.Add(base, offset)) = v
			return nil
		}, nil
	case reflect.Int64:
		return func(base unsafe.Pointer, value, raw []byte, dataType jsonparser.ValueType) error {
			if !isNumeric(value, dataType) {
				return fmt.Errorf("%w: want an integer, got %s", ErrTupleElementType, raw)
			}
			if len(value) == 0 {
				return nil
			}
			v, err := jsonparser.ParseInt(value)
			if err != nil {
				return fmt.Errorf("%w: want an integer, got %s", ErrTupleElementType, raw)
			}
			*(*int64)(unsafe.Add(base, offset)) = v
			return nil
		}, nil
	case reflect.Uint64:
		return func(base unsafe.Pointer, value, raw []byte, dataType jsonparser.ValueType) error {
			if !isNumeric(value, dataType) {
				return fmt.Errorf("%w: want an unsigned integer, got %s", ErrTupleElementType, raw)
			}
			if len(value) == 0 {
				return nil
			}
			v, ok := parseUint64(value)
			if !ok {
				return fmt.Errorf("%w: want an unsigned integer, got %s", ErrTupleElementType, raw)
			}
			*(*uint64)(unsafe.Add(base, offset)) = v
			return nil
		}, nil
	case reflect.String:
		return func(base unsafe.Pointer, value, raw []byte, dataType jsonparser.ValueType) error {
			if dataType != jsonparser.String {
				return fmt.Errorf("%w: want a string, got %s", ErrTupleElementType, raw)
			}
			s := string(value)
			if bytes.IndexByte(value, '\\') >= 0 {
				var err error
				if s, err = jsonparser.ParseString(value); err != nil {
					return fmt.Errorf("%w: %s", ErrTupleElementType, raw)
				}
			}
			*(*string)(unsafe.Add(base, offset)) = s
			return nil
		}, nil
	case reflect.Bool:
		return func(base unsafe.Pointer, value, raw []byte, dataType jsonparser.ValueType) error {
			if dataType != jsonparser.Boolean {
				return fmt.Errorf("%w: want a boolean, got %s", ErrTupleElementType, raw)
			}
			*(*bool)(unsafe.Add(base, offset)) = value[0] == 't'
			return nil
		}, nil
	}
	// Other kinds, such as int, whose size varies, narrow numbers, pointers, maps and slices, are rare enough to leave
	// to Unmarshal
	return func(base unsafe.Pointer, _, raw []byte, _ jsonparser.ValueType) error {
		return Unmarshal(raw, reflect.NewAt(typ, unsafe.Add(base, offset)).Interface())
	}, nil
}

// isNumeric reports whether an element can hold a number: a number, or a string without escapes, as some exchanges
// quote numbers
func isNumeric(value []byte, dataType jsonparser.ValueType) bool {
	return dataType == jsonparser.Number || dataType == jsonparser.String && bytes.IndexByte(value, '\\') < 0
}

// decodeTupleSlice decodes an array of tuples into a slice field, element by element through the element's plan
func decodeTupleSlice(typ reflect.Type, offset uintptr, sub *tuplePlan) elementDecoder {
	return func(base unsafe.Pointer, _, raw []byte, dataType jsonparser.ValueType) error {
		if dataType != jsonparser.Array {
			return fmt.Errorf("%w: want an array, got %s", ErrTupleElementType, raw)
		}
		slice := reflect.NewAt(typ, unsafe.Add(base, offset)).Elem()
		slice.SetLen(0)
		return EachArrayElement(raw, func(n int, element []byte, elementType jsonparser.ValueType) error {
			slice.Grow(1)
			slice.SetLen(n + 1)
			// A reused backing array holds the elements of an earlier decode, whose fields a null must not keep
			slice.Index(n).SetZero()
			if elementType == jsonparser.Null {
				return nil
			}
			if err := sub.decode(element, slice.Index(n).Addr().UnsafePointer()); err != nil {
				return fmt.Errorf("element %d: %w", n, err)
			}
			return nil
		})
	}
}

// parseUint64 parses the digits of an unsigned integer, reporting false for anything else or an overflow
func parseUint64(b []byte) (uint64, bool) {
	if len(b) == 0 || len(b) > 20 {
		return 0, false
	}
	var v uint64
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}
		digit := uint64(c - '0')
		if v > (math.MaxUint64-digit)/10 {
			return 0, false
		}
		v = v*10 + digit
	}
	return v, true
}

// skipSpace returns the index of the first byte at or after i that is not JSON whitespace
func skipSpace(data []byte, i int) int {
	for i < len(data) {
		switch data[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

// nextElement returns the index of the element after the one ending at i, or len(data) at the array's end
func nextElement(data []byte, i int) (int, error) {
	i = skipSpace(data, i)
	if i == len(data) {
		return 0, ErrMalformedTuple
	}
	switch data[i] {
	case ',':
		if i = skipSpace(data, i+1); i == len(data) || data[i] == ']' {
			return 0, ErrMalformedTuple
		}
		return i, nil
	case ']':
		return len(data), nil
	default:
		return 0, ErrMalformedTuple
	}
}

// decode decodes a tuple into the struct at base, stopping once its last field is set; null leaves the struct unset
func (p *tuplePlan) decode(data []byte, base unsafe.Pointer) error {
	i := skipSpace(data, 0)
	if i == len(data) || data[i] != '[' {
		if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
			return nil
		}
		return fmt.Errorf("%w: %s from %.64s", ErrTupleNotArray, p.name, data)
	}
	if i = skipSpace(data, i+1); i < len(data) && data[i] == ']' {
		return nil
	}
	for position, next := 0, 0; next < len(p.fields); position++ {
		value, dataType, end, err := jsonparser.Get(data[i:])
		if err != nil {
			return fmt.Errorf("%w: %s at position %d: %w", ErrMalformedTuple, p.name, position, err)
		}
		if f := &p.fields[next]; f.position == position {
			next++
			switch {
			case dataType != jsonparser.Null:
				if err := f.decode(base, value, data[i:i+end], dataType); err != nil {
					return fmt.Errorf("%s at position %d: %w", f.name, f.position, err)
				}
			case f.nilable:
				// A struct decoded into again keeps nothing from before where a pointer, slice or map is null
				reflect.NewAt(f.typ, unsafe.Add(base, f.offset)).Elem().SetZero()
			}
		}
		if i, err = nextElement(data, i+end); err != nil {
			return fmt.Errorf("%s: %w", p.name, err)
		}
		if i == len(data) {
			return nil
		}
	}
	return nil
}
