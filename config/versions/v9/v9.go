package v9

import (
	"context"

	"github.com/buger/jsonparser"
)

// Version is an ExchangeVersion to change the name of CoinbasePro to Coinbase
type Version struct{}

// Exchanges returns just CoinbasePro and Coinbase
func (*Version) Exchanges() []string { return []string{"CoinbasePro", "Coinbase"} }

// UpgradeExchange will change the exchange name from CoinbasePro to Coinbase
func (*Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == "CoinbasePro" {
		return jsonparser.Set(e, []byte(`"Coinbase"`), "name")
	}
	return e, nil
}

// DowngradeExchange will change the exchange name from Coinbase to CoinbasePro
func (*Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == "Coinbase" {
		return jsonparser.Set(e, []byte(`"CoinbasePro"`), "name")
	}
	return e, nil
}
