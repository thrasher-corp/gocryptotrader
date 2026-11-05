package bitstamp

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UnmarshalJSON implements json.Unmarshaler
func (s *OrderSide) UnmarshalJSON(data []byte) error {
	// Tickers and order placement responses quote the side, whereas cancellations and websocket events send a bare number
	switch string(bytes.Trim(data, `"`)) {
	case "0":
		*s = OrderSide(order.Buy)
	case "1":
		*s = OrderSide(order.Sell)
	default:
		return fmt.Errorf("%w: %s", order.ErrSideIsInvalid, data)
	}

	return nil
}

// Side returns the order side
func (s OrderSide) Side() order.Side {
	return order.Side(s)
}

// UnmarshalJSON implements json.Unmarshaler
func (s *EnabledStatus) UnmarshalJSON(data []byte) error {
	var status *string
	if err := json.Unmarshal(data, &status); err != nil {
		return fmt.Errorf("%w: %s", errInvalidEnabledStatus, data)
	}
	*s = status != nil && *status == statusEnabled
	return nil
}

// String returns the status as the API sends it
func (s EnabledStatus) String() string {
	if s {
		return statusEnabled
	}
	return statusDisabled
}

// dateTimeLayouts are the forms in which the API sends date-time values
var dateTimeLayouts = []string{time.RFC3339, "2006-01-02T15:04:05", time.DateTime, "2006-01-02 15:04:05Z07:00"}

// UnmarshalJSON implements json.Unmarshaler
func (d *DateTime) UnmarshalJSON(data []byte) error {
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("%w: %s", errInvalidDateTime, data)
	}
	if s == nil || *s == "" {
		*d = DateTime{}
		return nil
	}
	for _, layout := range dateTimeLayouts {
		if t, err := time.Parse(layout, *s); err == nil {
			*d = DateTime(t)
			return nil
		}
	}
	return fmt.Errorf("%w: %q", errInvalidDateTime, *s)
}

// Time returns the date and time as a time.Time
func (d DateTime) Time() time.Time {
	return time.Time(d)
}

// MarshalJSON implements json.Marshaler
func (r RetailInfo) MarshalJSON() ([]byte, error) { //nolint:gocritic // hugeParam: a value receiver lets RetailInfo values marshal with the documented fields on every JSON backend
	info := retailInfoJSON{
		FirstName:            r.FirstName,
		LastName:             r.LastName,
		PlaceOfBirth:         r.PlaceOfBirth,
		IDType:               r.IDType,
		IDNumber:             r.IDNumber,
		StreetAndHouseNumber: r.StreetAndHouseNumber,
		City:                 r.City,
		PostalCode:           r.PostalCode,
		Country:              r.Country,
	}
	if !r.DateOfBirth.IsZero() {
		info.DateOfBirth = r.DateOfBirth.Format(time.DateOnly)
	}
	return json.Marshal(info)
}

// UnmarshalJSON implements json.Unmarshaler
func (r *RetailInfo) UnmarshalJSON(data []byte) error {
	var info retailInfoJSON
	if err := json.Unmarshal(data, &info); err != nil {
		return err
	}
	*r = RetailInfo{
		FirstName:            info.FirstName,
		LastName:             info.LastName,
		PlaceOfBirth:         info.PlaceOfBirth,
		IDType:               info.IDType,
		IDNumber:             info.IDNumber,
		StreetAndHouseNumber: info.StreetAndHouseNumber,
		City:                 info.City,
		PostalCode:           info.PostalCode,
		Country:              info.Country,
	}
	if info.DateOfBirth != "" {
		dateOfBirth, err := time.Parse(time.DateOnly, info.DateOfBirth)
		if err != nil {
			return fmt.Errorf("%w: date of birth %q", errInvalidDateTime, info.DateOfBirth)
		}
		r.DateOfBirth = dateOfBirth
	}
	return nil
}

// UnmarshalJSON decodes an orderbook level sent as [price, amount] or [price, amount, order_id]
func (l *OrderbookLevel) UnmarshalJSON(data []byte) error {
	var level []string
	if err := json.Unmarshal(data, &level); err != nil {
		return err
	}
	if len(level) != 2 && len(level) != 3 {
		return fmt.Errorf("%w: %s", errMalformedOrderbookLevel, data)
	}
	*l = OrderbookLevel{}
	var err error
	if l.Price, err = strconv.ParseFloat(level[0], 64); err != nil {
		return fmt.Errorf("%w price %q: %w", errMalformedOrderbookLevel, level[0], err)
	}
	if l.Amount, err = strconv.ParseFloat(level[1], 64); err != nil {
		return fmt.Errorf("%w amount %q: %w", errMalformedOrderbookLevel, level[1], err)
	}
	if len(level) == 3 {
		if l.OrderID, err = strconv.ParseUint(level[2], 10, 64); err != nil {
			return fmt.Errorf("%w order id %q: %w", errMalformedOrderbookLevel, level[2], err)
		}
	}
	return nil
}

// UnmarshalJSON decodes an order transaction, collecting the dynamically named currency amounts
func (t *OrderTransaction) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*t = OrderTransaction{Amounts: make(map[currency.Code]float64)}
	for key, value := range fields {
		var err error
		switch key {
		case "tid":
			err = json.Unmarshal(value, &t.TradeID)
		case "price":
			t.Price, err = parseNumber(value)
		case "fee":
			t.Fee, err = parseOptionalNumber(value)
		case "datetime":
			t.DateTime, err = parseOptionalDateTime(value)
		case "type":
			err = unmarshalTransactionType(value, &t.Type)
		default:
			if !isAlphanumeric(key) {
				continue // Undocumented fields cannot be currency amounts
			}
			amount, parseErr := parseNumber(value)
			if parseErr != nil {
				continue // Undocumented non-numeric fields cannot be currency amounts
			}
			t.Amounts[currency.NewCode(key).Upper()] = amount
		}
		if err != nil {
			return fmt.Errorf("%w %q: %w", errMalformedTransaction, key, err)
		}
	}
	return nil
}

// UnmarshalJSON decodes a user transaction, collecting the dynamically named currency amounts and exchange rates
func (t *UserTransactionResponse) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*t = UserTransactionResponse{
		Amounts:       make(map[currency.Code]float64),
		ExchangeRates: make(map[currency.Pair]float64),
	}
	rates := make(map[string]float64)
	for key, value := range fields {
		var err error
		switch key {
		case "id":
			err = json.Unmarshal(value, &t.ID)
		case "datetime":
			t.DateTime, err = parseOptionalDateTime(value)
		case "type":
			err = unmarshalTransactionType(value, &t.Type)
		case "fee":
			t.Fee, err = parseOptionalNumber(value)
		case "order_id":
			err = json.Unmarshal(value, &t.OrderID)
		case "self_trade":
			err = json.Unmarshal(value, &t.SelfTrade)
		case "self_trade_order_id":
			err = json.Unmarshal(value, &t.SelfTradeOrderID)
		default:
			base, quote, isPair := strings.Cut(key, "_")
			if !isAlphanumeric(base) || (isPair && !isAlphanumeric(quote)) {
				continue // Undocumented fields cannot be currency amounts or exchange rates
			}
			amount, parseErr := parseNumber(value)
			if parseErr != nil {
				continue // Undocumented non-numeric fields cannot be currency amounts or exchange rates
			}
			if isPair {
				rates[key] = amount
				continue
			}
			t.Amounts[currency.NewCode(key).Upper()] = amount
		}
		if err != nil {
			return fmt.Errorf("%w %q: %w", errMalformedTransaction, key, err)
		}
	}
	// An exchange rate is keyed by its market and accompanies the amounts of both of the market's currencies, which
	// distinguishes it from undocumented numeric fields whose names also contain an underscore
	for key, rate := range rates {
		base, quote, _ := strings.Cut(key, "_")
		baseCode, quoteCode := currency.NewCode(base).Upper(), currency.NewCode(quote).Upper()
		if _, ok := t.Amounts[baseCode]; !ok {
			continue
		}
		if _, ok := t.Amounts[quoteCode]; !ok {
			continue
		}
		t.ExchangeRates[currency.NewPair(baseCode, quoteCode)] = rate
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler
func (u *unconfirmedDeposits) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '{' {
		var deposit UnconfirmedDepositResponse
		if err := json.Unmarshal(data, &deposit); err != nil {
			return err
		}
		*u = unconfirmedDeposits{deposit}
		return nil
	}
	var deposits []UnconfirmedDepositResponse
	if err := json.Unmarshal(data, &deposits); err != nil {
		return err
	}
	*u = deposits
	return nil
}

// parseOptionalNumber decodes a number which may be sent as a JSON number, a numeric string or null, which decodes as 0
func parseOptionalNumber(data json.RawMessage) (float64, error) {
	if bytes.Equal(data, []byte("null")) {
		return 0, nil
	}
	return parseNumber(data)
}

// parseOptionalDateTime decodes a date and time sent in Bitstamp's "2006-01-02 15:04:05" form, or null which decodes
// as the zero time
func parseOptionalDateTime(data json.RawMessage) (time.Time, error) {
	if bytes.Equal(data, []byte("null")) {
		return time.Time{}, nil
	}
	var dt types.DateTime
	if err := json.Unmarshal(data, &dt); err != nil {
		return time.Time{}, err
	}
	return dt.Time(), nil
}

// parseNumber decodes a number which may be sent as either a JSON number or a numeric string
func parseNumber(data json.RawMessage) (float64, error) {
	if bytes.Equal(data, []byte("null")) {
		return 0, errNullNumber
	}
	var n types.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return 0, err
	}
	return n.Float64(), nil
}

// isAlphanumeric reports whether s is a non-empty string of ASCII letters and digits, as used by currency codes
func isAlphanumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// unmarshalTransactionType decodes a transaction type which may be sent as either a JSON number or a numeric string
func unmarshalTransactionType(data json.RawMessage, t *TransactionType) error {
	v, err := strconv.ParseUint(string(bytes.Trim(data, `"`)), 10, 8)
	if err != nil {
		return err
	}
	*t = TransactionType(v)
	return nil
}

// retailInfoJSON is the JSON representation of RetailInfo
type retailInfoJSON struct {
	FirstName            string `json:"first_name"`
	LastName             string `json:"last_name"`
	DateOfBirth          string `json:"date_of_birth,omitempty"`
	PlaceOfBirth         string `json:"place_of_birth,omitempty"`
	IDType               string `json:"id_type,omitempty"`
	IDNumber             string `json:"id_number,omitempty"`
	StreetAndHouseNumber string `json:"street_and_house_number,omitempty"`
	City                 string `json:"city,omitempty"`
	PostalCode           string `json:"zip,omitempty"`
	Country              string `json:"country,omitempty"`
}
