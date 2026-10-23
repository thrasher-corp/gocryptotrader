package bitfinex

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// UnmarshalJSON decodes v1 orders whose symbols need not contain a delimiter.
func (o *Order) UnmarshalJSON(data []byte) error {
	type orderAlias Order
	var decoded orderAlias
	response := struct {
		*orderAlias
		Symbol string `json:"symbol"`
	}{orderAlias: &decoded}
	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}
	if response.Symbol != "" {
		pair, err := currency.NewPairFromString(response.Symbol)
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
