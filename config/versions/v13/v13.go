// Package v13 migrates configuration values introduced by the HTX rename and
// deprecated CoinMarketCap account plan names.
package v13

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
)

const (
	accountPlanPath       = "currencyConfig.cryptocurrencyProvider.accountPlan"
	basic                 = "basic"
	builder               = "builder"
	growth                = "growth"
	hobbyist              = "hobbyist"
	legacyPlanPlaceholder = "accountplan"
	standard              = "standard"
)

// Version implements ConfigVersion and ExchangeVersion for the v13 migrations.
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

// UpgradeConfig replaces deprecated account plan names with their current
// CoinMarketCap equivalents.
func (*Version) UpgradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return replaceAccountPlan(config, map[string]string{
		"":                    basic,
		hobbyist:              builder,
		legacyPlanPlaceholder: basic,
		standard:              growth,
	})
}

// DowngradeConfig restores the deprecated account plan names understood by
// configurations predating v13.
func (*Version) DowngradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return replaceAccountPlan(config, map[string]string{
		builder: hobbyist,
		growth:  standard,
	})
}

func replaceAccountPlan(config []byte, replacements map[string]string) ([]byte, error) {
	_, valueType, _, err := jsonparser.Get(config, "currencyConfig", "cryptocurrencyProvider", "accountPlan")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
		return config, nil
	case err != nil:
		return config, fmt.Errorf("error getting %s: %w", accountPlanPath, err)
	case valueType != jsonparser.String:
		return config, nil
	}

	accountPlan, err := jsonparser.GetString(config, "currencyConfig", "cryptocurrencyProvider", "accountPlan")
	if err != nil {
		return config, fmt.Errorf("error getting %s: %w", accountPlanPath, err)
	}

	replacement, found := replacements[strings.ToLower(strings.TrimSpace(accountPlan))]
	if !found {
		return config, nil
	}

	return jsonparser.Set(config, []byte(strconv.Quote(replacement)), "currencyConfig", "cryptocurrencyProvider", "accountPlan")
}
