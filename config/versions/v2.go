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

// Exchanges returns just GDAX and CoinbasePro
func (v *Version2) Exchanges() []string { return []string{"GDAX", "CoinbasePro"} }

// UpgradeExchange will change the exchange name from GDAX to CoinbasePro
func (v *Version2) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == "GDAX" {
		return jsonparser.Set(e, []byte(`"CoinbasePro"`), "name")
	}
	return e, nil
}

// DowngradeExchange will change the exchange name from CoinbasePro to GDAX
func (v *Version2) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == "CoinbasePro" {
		return jsonparser.Set(e, []byte(`"GDAX"`), "name")
	}
	return e, nil
}
