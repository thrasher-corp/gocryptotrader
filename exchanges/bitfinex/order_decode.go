package bitfinex

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var errOrderArrayTooShort = errors.New("order array must contain at least 18 fields")

// UnmarshalJSON accepts v1 objects and v2 history arrays without changing the
// public Order return type. V1 symbols are not necessarily delimited pairs.
func (o *Order) UnmarshalJSON(data []byte) error {
	type orderAlias Order
	var decoded orderAlias
	var symbol string
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
		var fields []json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return err
		}
		if len(fields) < 18 {
			return errOrderArrayTooShort
		}
		// https://docs.bitfinex.com/reference/rest-auth-orders-history-by-symbol
		for _, field := range []struct {
			index int
			value any
		}{
			{0, &decoded.ID},
			{3, &symbol},
			{4, &decoded.Timestamp},
			{5, &decoded.LastUpdated},
			{6, &decoded.RemainingAmount},
			{7, &decoded.OriginalAmount},
			{8, &decoded.Type},
			{13, &decoded.Status},
			{16, &decoded.Price},
			{17, &decoded.AverageExecutionPrice},
		} {
			if err := json.Unmarshal(fields[field.index], field.value); err != nil {
				return fmt.Errorf("order field %d: %w", field.index, err)
			}
		}
		symbol = strings.TrimPrefix(symbol, "t")
		decoded.Side = "buy"
		if decoded.OriginalAmount < 0 {
			decoded.Side = "sell"
		}
		decoded.OriginalAmount = math.Abs(decoded.OriginalAmount)
		decoded.RemainingAmount = math.Abs(decoded.RemainingAmount)
		decoded.ExecutedAmount = decoded.OriginalAmount - decoded.RemainingAmount
	} else {
		response := struct {
			*orderAlias
			Symbol string `json:"symbol"`
		}{orderAlias: &decoded}
		if err := json.Unmarshal(data, &response); err != nil {
			return err
		}
		symbol = response.Symbol
	}
	if symbol != "" {
		pair, err := currency.NewPairFromString(symbol)
		if err != nil {
			return err
		}
		decoded.Symbol = pair
	}
	if decoded.ID == 0 {
		decoded.ID = decoded.OrderID
	}
	*o = Order(decoded)
	return nil
}
