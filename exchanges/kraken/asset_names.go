package kraken

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var errInvalidSymbol = errors.New("invalid symbol")

// assetNames translates between the names Kraken gives an asset. GoCryptoTrader names assets and pairs by their
// alternative names, such as XBT and XBT/USD, as Kraken's REST pair names and AssetPairs' wsname do. REST results key
// assets by internal names, such as XXBT, while websocket v2 and assetVersion=1 use display names, such as BTC and
// BTC/USD
type assetNames struct {
	m sync.RWMutex
	// alternative maps an internal or display name to its alternative name
	alternative map[string]string
	// display maps an alternative name to its display name, for the assets whose names differ
	display map[string]string
}

// seed loads the names from Get Asset Info, keyed by internal name in internal and by display name in display
func (a *assetNames) seed(internal, display map[string]AssetInfo) {
	alternative := make(map[string]string, len(internal)+len(display))
	displayNames := make(map[string]string)
	for name, info := range internal {
		alternative[name] = info.AlternativeName
	}
	for name, info := range display {
		alternative[name] = info.AlternativeName
		if name != info.AlternativeName {
			displayNames[info.AlternativeName] = name
		}
	}
	a.m.Lock()
	a.alternative, a.display = alternative, displayNames
	a.m.Unlock()
}

// seeded reports whether the names have been loaded
func (a *assetNames) seeded() bool {
	a.m.RLock()
	defer a.m.RUnlock()
	return len(a.alternative) != 0
}

// alternativeName returns the alternative name of an internal or display name, and whether the name is known
func (a *assetNames) alternativeName(name string) (string, bool) {
	a.m.RLock()
	defer a.m.RUnlock()
	alt, ok := a.alternative[name]
	return alt, ok
}

// displayName returns the display name of an alternative name, which is the alternative name itself for most assets
func (a *assetNames) displayName(altName string) string {
	a.m.RLock()
	defer a.m.RUnlock()
	if name, ok := a.display[altName]; ok {
		return name
	}
	return altName
}

// SeedAssets loads Kraken's asset names, which the websocket and the results keyed by internal or display names are
// translated with
func (e *Exchange) SeedAssets(ctx context.Context) error {
	internal, err := e.GetAssets(ctx, nil)
	if err != nil {
		return err
	}
	display, err := e.GetAssets(ctx, &AssetsRequest{DisplayNames: true})
	if err != nil {
		return err
	}
	e.assetNames.seed(internal, display)
	return nil
}

// displaySymbol returns a spot pair as websocket v2 and assetVersion=1 name it, such as BTC/USD for XBT/USD
func (e *Exchange) displaySymbol(p currency.Pair) string {
	return e.assetNames.displayName(p.Base.Upper().String()) + "/" + e.assetNames.displayName(p.Quote.Upper().String())
}

// pairFromDisplaySymbol returns the spot pair a websocket v2 or assetVersion=1 symbol names, such as XBT/USD for
// BTC/USD, in the configured pair format. A name the seeded names do not know is its own alternative name, as every
// recently listed asset's is
func (e *Exchange) pairFromDisplaySymbol(symbol string) (currency.Pair, error) {
	base, quote, ok := strings.Cut(symbol, "/")
	if !ok || base == "" || quote == "" {
		return currency.EMPTYPAIR, fmt.Errorf("%w: %q", errInvalidSymbol, symbol)
	}
	if alt, ok := e.assetNames.alternativeName(base); ok {
		base = alt
	}
	if alt, ok := e.assetNames.alternativeName(quote); ok {
		quote = alt
	}
	pFmt, err := e.GetPairFormat(asset.Spot, false)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	return currency.NewPair(currency.NewCode(base), currency.NewCode(quote)).Format(pFmt), nil
}
