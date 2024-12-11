package versions

import (
	"context"

	"github.com/buger/jsonparser"
)

// Version2 is an ExchangeVersion to change the name of GDAX to CoinbasePro
type Version2 struct{}

func init() {
	Manager.registerVersion(2, &Version2{})
}

const (
	from = "GDAX"
	to   = "CoinbasePro"
)

// Exchanges returns just GDAX and CoinbasePro
func (v *Version2) Exchanges() []string { return []string{from, to} }

// UpgradeExchange will change the exchange name from GDAX to CoinbasePro
func (v *Version2) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == from {
		return jsonparser.Set(e, []byte(`"`+to+`"`), "name")
	}
	return e, nil
}

// DowngradeExchange will change the exchange name from CoinbasePro to GDAX
func (v *Version2) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == to {
		return jsonparser.Set(e, []byte(`"`+from+`"`), "name")
	}
	return e, nil
}
