// v2 is an ExchangeVersion to change the name of GDAX to CoinbasePro
package v2

import (
	"context"

	"github.com/buger/jsonparser"
)

// Version implements ExchangeVersion
type Version struct{}

const (
	from = "GDAX"
	to   = "CoinbasePro"
)

// Exchanges returns just GDAX and CoinbasePro
func (v *Version) Exchanges() []string { return []string{from, to} }

// UpgradeExchange will change the exchange name from GDAX to CoinbasePro
func (v *Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == from {
		return jsonparser.Set(e, []byte(`"`+to+`"`), "name")
	}
	return e, nil
}

// DowngradeExchange will change the exchange name from CoinbasePro to GDAX
func (v *Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == to {
		return jsonparser.Set(e, []byte(`"`+from+`"`), "name")
	}
	return e, nil
}
