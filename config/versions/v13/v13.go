package v13

import (
	"context"

	"github.com/buger/jsonparser"
)

// Version implements ExchangeVersion to migrate Huobi configurations to HTX.
type Version struct{}

// Exchanges returns the legacy and current exchange names handled by this migration.
func (*Version) Exchanges() []string { return []string{"Huobi", "HTX"} }

// UpgradeExchange changes the legacy Huobi exchange name to HTX.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	if name, err := jsonparser.GetString(exchange, "name"); err == nil && name == "Huobi" {
		return jsonparser.Set(exchange, []byte(`"HTX"`), "name")
	}
	return exchange, nil
}

// DowngradeExchange changes the HTX exchange name back to Huobi.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	if name, err := jsonparser.GetString(exchange, "name"); err == nil && name == "HTX" {
		return jsonparser.Set(exchange, []byte(`"Huobi"`), "name")
	}
	return exchange, nil
}
