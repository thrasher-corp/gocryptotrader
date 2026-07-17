package v12

import (
	"context"

	"github.com/buger/jsonparser"
)

// Version is an ExchangeVersion to change the name of Huobi to HTX.
type Version struct{}

// Exchanges returns Huobi and HTX.
func (*Version) Exchanges() []string { return []string{"Huobi", "HTX"} }

// UpgradeExchange will change the exchange name from Huobi to HTX.
func (*Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == "Huobi" {
		return jsonparser.Set(e, []byte(`"HTX"`), "name")
	}
	return e, nil
}

// DowngradeExchange will change the exchange name from HTX to Huobi.
func (*Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if n, err := jsonparser.GetString(e, "name"); err == nil && n == "HTX" {
		return jsonparser.Set(e, []byte(`"Huobi"`), "name")
	}
	return e, nil
}
