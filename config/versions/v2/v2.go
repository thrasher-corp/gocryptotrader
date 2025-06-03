package v2

import (
	"context"

	"github.com/buger/jsonparser"
)

// Version is an ExchangeVersion to change the name of GDAX to CoinbasePro
type Version struct{}

// Exchanges returns just GDAX and CoinbasePro
func (*Version) Exchanges() []string { return []string{"GDAX", "CoinbasePro"} }

// UpgradeExchange will change the exchange name from GDAX to CoinbasePro
func (*Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == "GDAX" {
		return jsonparser.Set(e, []byte(`"CoinbasePro"`), "name")
	}
	return e, nil
}

// DowngradeExchange will change the exchange name from CoinbasePro to GDAX
func (*Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == "CoinbasePro" {
		return jsonparser.Set(e, []byte(`"GDAX"`), "name")
	}
	return e, nil
}
