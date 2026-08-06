// Package v13 migrates deprecated CoinMarketCap account plan names.
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

// Version implements ConfigVersion to migrate CoinMarketCap account plan names.
type Version struct{}

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
