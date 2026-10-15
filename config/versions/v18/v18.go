// Package v18 removes the options REST endpoint, which earlier versions reject, when downgrading.
package v18

import (
	"context"

	"github.com/buger/jsonparser"
)

// Version implements ExchangeVersion for the options REST endpoint key added in this version.
type Version struct{}

// Exchanges applies this migration to every exchange.
func (*Version) Exchanges() []string { return []string{"*"} }

// UpgradeExchange is a no-op: an exchange with an options REST endpoint sets its default when the saved configuration
// has none.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return exchange, nil
}

// DowngradeExchange removes every saved options REST endpoint, whatever its value. Earlier versions do not recognise
// the key and fail to set the exchange up from a configuration that contains it.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	for {
		var found bool
		if exchange, found = jsonparser.DeleteFound(exchange, "api", "urlEndpoints", "RestOptionsURL"); !found {
			return exchange, nil
		}
	}
}
