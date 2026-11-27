package order

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/internal/bsonwire"
)

// Errors returned when building, storing or reading fees
var (
	ErrDuplicateFeeCurrency = errors.New("duplicate fee currency")
	ErrInvalidFeeAmount     = errors.New("invalid fee amount")
	ErrInvalidFees          = errors.New("invalid fees")
)

// FeeAmount is a fee in one currency. A positive Amount was charged and a negative Amount is a rebate.
type FeeAmount struct {
	Currency currency.Code
	Amount   float64
}

// Fees holds the fees an exchange reported, one amount per currency. Amounts in different currencies are never summed
// or converted.
//
// The zero value holds fees that were not reported, so merging it into a stored order keeps the stored fees. NewFees
// returns an authoritative snapshot, which replaces them; a snapshot without amounts reports that there are no fees.
//
// Fees is immutable, so a copy can be stored or returned without aliasing the original.
type Fees struct {
	amounts  []FeeAmount
	reported bool
}

// NewFees returns an authoritative snapshot of the fees in amounts; no amounts reports that there are none. Currency
// codes are normalised to upper case and must be set and unique, amounts must be finite and keep their sign, and the
// snapshot orders its amounts by currency code.
func NewFees(amounts ...FeeAmount) (Fees, error) {
	f := Fees{reported: true}
	if len(amounts) == 0 {
		return f, nil
	}
	f.amounts = make([]FeeAmount, len(amounts))
	for i, a := range amounts {
		if a.Currency.IsEmpty() {
			return Fees{}, fmt.Errorf("fee amount %v: %w", a.Amount, currency.ErrCurrencyCodeEmpty)
		}
		if math.IsNaN(a.Amount) || math.IsInf(a.Amount, 0) {
			return Fees{}, fmt.Errorf("%w: %v %s", ErrInvalidFeeAmount, a.Amount, a.Currency)
		}
		f.amounts[i] = FeeAmount{Currency: a.Currency.Upper(), Amount: a.Amount}
	}
	slices.SortFunc(f.amounts, func(a, b FeeAmount) int { return cmp.Compare(a.Currency.String(), b.Currency.String()) })
	for i := 1; i < len(f.amounts); i++ {
		if f.amounts[i].Currency.String() == f.amounts[i-1].Currency.String() {
			return Fees{}, fmt.Errorf("%w: %s", ErrDuplicateFeeCurrency, f.amounts[i].Currency)
		}
	}
	return f, nil
}

// IsReported returns whether the exchange reported the fees, as opposed to leaving them out
func (f Fees) IsReported() bool {
	return f.reported
}

// Amounts returns a copy of the fee amounts ordered by currency code, or nil when there are none
func (f Fees) Amounts() []FeeAmount {
	return slices.Clone(f.amounts)
}

// MarshalJSON encodes fees that were not reported as null, and reported fees as an array of their amounts, which is
// empty when there are none
func (f Fees) MarshalJSON() ([]byte, error) {
	if !f.reported {
		return []byte("null"), nil
	}
	if len(f.amounts) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(f.amounts)
}

// UnmarshalJSON decodes fees as MarshalJSON encodes them: null as fees that were not reported, and an array as a
// snapshot that must meet the rules of NewFees
func (f *Fees) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*f = Fees{}
		return nil
	}
	var amounts []FeeAmount
	if err := json.Unmarshal(data, &amounts); err != nil {
		return err
	}
	fees, err := NewFees(amounts...)
	if err != nil {
		return err
	}
	*f = fees
	return nil
}

// MarshalBSONValue implements bson.ValueMarshaler, storing fees that were not reported as null, and reported fees as a
// document of each currency's amount as a string, which is empty when there are none
func (f Fees) MarshalBSONValue() (typ byte, data []byte, err error) {
	if !f.reported {
		return bsonwire.TypeNull, nil, nil
	}
	fields := make([]bsonwire.Field, len(f.amounts))
	for i, a := range f.amounts {
		code := a.Currency.String()
		if strings.IndexByte(code, 0) != -1 {
			return 0, nil, fmt.Errorf("%w: currency %q holds a NUL byte, which ends a BSON document key", ErrInvalidFees, code)
		}
		fields[i] = bsonwire.Field{Key: code, Value: strconv.FormatFloat(a.Amount, 'f', -1, 64)}
	}
	return bsonwire.TypeEmbeddedDocument, bsonwire.MarshalDocument(fields...), nil
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler, reading fees as MarshalBSONValue stores them: null as fees that
// were not reported, and a document as a snapshot that must meet the rules of NewFees. The decoded fees replace any
// held, and an error leaves them unchanged
func (f *Fees) UnmarshalBSONValue(typ byte, data []byte) error {
	if typ == bsonwire.TypeNull {
		if len(data) != 0 {
			return fmt.Errorf("%w: %w: BSON null followed by %d bytes", ErrInvalidFees, bsonwire.ErrLength, len(data))
		}
		*f = Fees{}
		return nil
	}
	fields, err := bsonwire.UnmarshalDocument(typ, data)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidFees, err)
	}
	amounts := make([]FeeAmount, len(fields))
	for i, field := range fields {
		amount, err := strconv.ParseFloat(field.Value, 64)
		if err != nil {
			return fmt.Errorf("%w: %q for currency %q: %w", ErrInvalidFeeAmount, field.Value, field.Key, err)
		}
		amounts[i] = FeeAmount{Currency: currency.NewCode(field.Key), Amount: amount}
	}
	fees, err := NewFees(amounts...)
	if err != nil {
		return err
	}
	*f = fees
	return nil
}

// IsZero implements bson.Zeroer, reporting fees that were not reported as zero. Without it, a driver omitting empty
// fields with OmitZeroStruct judges Fees by its exported fields, of which it has none, and drops reported fees too
func (f Fees) IsZero() bool {
	return !f.reported
}

// equal returns whether f and o hold the same presence and amounts
func (f Fees) equal(o Fees) bool {
	return f.reported == o.reported && slices.Equal(f.amounts, o.amounts)
}

// scalar returns the amount and currency a single fee field holds for f: its only amount, or zero and an empty
// currency when it holds none or several, because one amount cannot stand for fees in several currencies
func (f Fees) scalar() (float64, currency.Code) {
	if len(f.amounts) != 1 {
		return 0, currency.EMPTYCODE
	}
	return f.amounts[0].Amount, f.amounts[0].Currency
}

// SetFees sets the order's cumulative fees. Reported fees also set Fee and FeeAsset, which hold their only amount, or
// zero and an empty currency when they hold none or several.
func (d *Detail) SetFees(f Fees) {
	d.Fees = f
	if f.reported {
		d.Fee, d.FeeAsset = f.scalar()
	}
}

// SetFees sets the order's cumulative fees at submission. Reported fees also set Fee and FeeAsset, which hold their
// only amount, or zero and an empty currency when they hold none or several.
func (s *SubmitResponse) SetFees(f Fees) {
	s.Fees = f
	if f.reported {
		s.Fee, s.FeeAsset = f.scalar()
	}
}

// SetFees sets the fees for this fill alone. Reported fees also set Fee and FeeAsset, which hold their only amount, or
// zero and an empty currency when they hold none or several.
func (t *TradeHistory) SetFees(f Fees) {
	t.Fees = f
	if f.reported {
		var feeAsset currency.Code
		t.Fee, feeAsset = f.scalar()
		t.FeeAsset = feeAsset.String()
	}
}

// mergeFees merges the fees in m into d and returns whether d changed. Reported fees replace d's fees and set Fee and
// FeeAsset from them, while unreported fees keep d's. Once d's fees have been reported, Fee and FeeAsset follow them
// alone, so a single fee in m cannot stand for part of fees in several currencies; until then, a positive Fee and a set
// FeeAsset in m replace d's.
func (d *Detail) mergeFees(m *Detail) bool {
	if m.Fees.reported {
		fee, feeAsset := m.Fees.scalar()
		if d.Fees.equal(m.Fees) && d.Fee == fee && d.FeeAsset == feeAsset {
			return false
		}
		d.Fees, d.Fee, d.FeeAsset = m.Fees, fee, feeAsset
		return true
	}
	if d.Fees.reported {
		return false
	}
	var changed bool
	if m.Fee > 0 && m.Fee != d.Fee {
		d.Fee = m.Fee
		changed = true
	}
	if !m.FeeAsset.IsEmpty() && !m.FeeAsset.Equal(d.FeeAsset) {
		d.FeeAsset = m.FeeAsset
		changed = true
	}
	return changed
}

// mergeFees merges the fees of fill m into fill t and returns whether t changed. It follows Detail.mergeFees, except
// that until t's fees have been reported, m's Fee replaces t's whatever its value, as it always has.
func (t *TradeHistory) mergeFees(m *TradeHistory) bool {
	if m.Fees.reported {
		fee, feeAsset := m.Fees.scalar()
		if t.Fees.equal(m.Fees) && t.Fee == fee && t.FeeAsset == feeAsset.String() {
			return false
		}
		t.Fees, t.Fee, t.FeeAsset = m.Fees, fee, feeAsset.String()
		return true
	}
	if t.Fees.reported || t.Fee == m.Fee {
		return false
	}
	t.Fee = m.Fee
	return true
}
