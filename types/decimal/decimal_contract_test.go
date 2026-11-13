package decimal

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json" //nolint:depguard // Compile-time interface checks only; the custom JSON package does not export these interfaces.
)

// decimalAPI checks the shared method signatures under either backend build.
type decimalAPI interface {
	Add(Decimal) Decimal
	Sub(Decimal) Decimal
	Mul(Decimal) Decimal
	Div(Decimal) Decimal
	Mod(Decimal) Decimal
	Pow(Decimal) Decimal
	Abs() Decimal
	Neg() Decimal
	Cmp(Decimal) int
	Compare(Decimal) int
	Equal(Decimal) bool
	GreaterThan(Decimal) bool
	GreaterThanOrEqual(Decimal) bool
	LessThan(Decimal) bool
	LessThanOrEqual(Decimal) bool
	IsZero() bool
	IsPositive() bool
	IsNegative() bool
	IsInteger() bool
	Round(int32) Decimal
	Truncate(int32) Decimal
	Floor() Decimal
	Ceil() Decimal
	String() string
	StringFixed(int32) string
	Float64() (float64, bool)
	InexactFloat64() float64
	IntPart() int64
}

var (
	_ decimalAPI                 = Decimal{}
	_ driver.Valuer              = Decimal{}
	_ encoding.BinaryMarshaler   = Decimal{}
	_ encoding.BinaryUnmarshaler = (*Decimal)(nil)
	_ encoding.TextMarshaler     = Decimal{}
	_ encoding.TextUnmarshaler   = (*Decimal)(nil)
	_ json.Marshaler             = Decimal{}
	_ json.Unmarshaler           = (*Decimal)(nil)
	_ sql.Scanner                = (*Decimal)(nil)
)
