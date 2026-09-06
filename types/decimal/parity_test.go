package decimal

import "database/sql/driver"

type decimalFacade interface {
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
	MarshalJSON() ([]byte, error)
	MarshalText() ([]byte, error)
	MarshalBinary() ([]byte, error)
	Value() (driver.Value, error)
}

type decimalPtrFacade interface {
	UnmarshalJSON([]byte) error
	UnmarshalText([]byte) error
	UnmarshalBinary([]byte) error
	Scan(any) error
}

var (
	_ decimalFacade    = Decimal{}
	_ decimalPtrFacade = (*Decimal)(nil)
)
