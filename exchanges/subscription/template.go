package subscription

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	groupSeparator  = "\x1D"
	recordSeparator = "\x1E"
)

var (
	errInvalidAssetExpandPairs = errors.New("subscription template containing PairSeparator with must contain either specific Asset or AssetSeparator")
	errAssetRecords            = errors.New("subscription template did not generate the expected number of asset records")
	errPairRecords             = errors.New("subscription template did not generate the expected number of pair records")
	errAssetTemplateWithoutAll = errors.New("sub.Asset must be set to All if AssetSeparator is used in Channel template")
	errNoTemplateContent       = errors.New("subscription template did not generate content")
	errInvalidTemplate         = errors.New("GetSubscriptionTemplate did not return a template")
)

type tplCtx struct {
	S              *Subscription
	AssetPairs     assetPairs
	PairSeparator  string
	AssetSeparator string
}

// ExpandTemplates returns a list of Subscriptions with Template expanded
// May be called on already expanded subscriptions: Passes $s through unprocessed if QualifiedChannel is already populated
// Calls e.GetSubscriptionTemplate to find a template for each subscription
// Filters out Authenticated subscriptions if !e.CanUseAuthenticatedEndpoints
// See README.md for more details
func (l List) ExpandTemplates(e iExchange) (List, error) {
	if !slices.ContainsFunc(l, func(s *Subscription) bool { return s.QualifiedChannel == "" }) {
		// Empty list, or already processed
		return slices.Clone(l), nil
	}

	if !e.CanUseAuthenticatedWebsocketEndpoints() {
		n := List{}
		for _, s := range l {
			if !s.Authenticated {
				n = append(n, s)
			}
		}
		l = n
	}

	ap, err := l.assetPairs(e)
	if err != nil {
		return nil, err
	}

	assets := make(asset.Items, 0, len(ap))
	for k := range ap {
		assets = append(assets, k)
	}
	slices.Sort(assets) // text/template ranges maps in sorted order
	subs := List{}

	for _, s := range l {
		if s.QualifiedChannel != "" {
			subs = append(subs, s)
			continue
		}

		subCtx := &tplCtx{
			S:              s,
			AssetPairs:     ap,
			PairSeparator:  recordSeparator,
			AssetSeparator: groupSeparator,
		}

		t, err := e.GetSubscriptionTemplate(s)
		if err != nil {
			return nil, err
		}
		if t == nil {
			return nil, errInvalidTemplate
		}

		buf := &bytes.Buffer{}
		if err := t.Execute(buf, subCtx); err != nil {
			return nil, err
		}

		out := buf.String()

		subAssets := assets
		xpandPairs := strings.Contains(out, subCtx.PairSeparator)
		if xpandAssets := strings.Contains(out, subCtx.AssetSeparator); xpandAssets {
			if s.Asset != asset.All {
				return nil, errAssetTemplateWithoutAll
			}
		} else {
			if xpandPairs && (s.Asset == asset.All || s.Asset == asset.Empty) {
				// We don't currently support expanding Pairs without expanding Assets for All or Empty assets, but we could; waiting for a use-case
				return nil, errInvalidAssetExpandPairs
			}
			// No expansion so update expected Assets for consistent behaviour below
			subAssets = []asset.Item{s.Asset}
		}

		out = strings.TrimRight(out, " \n\r\t"+subCtx.PairSeparator+subCtx.AssetSeparator)

		assetRecords := strings.Split(out, subCtx.AssetSeparator)
		if len(assetRecords) != len(subAssets) {
			return nil, fmt.Errorf("%w: Got %d; Expected %d", errAssetRecords, len(assetRecords), len(subAssets))
		}

		for i, assetChannels := range assetRecords {
			a := subAssets[i]
			assetChannels = strings.TrimRight(assetChannels, " \n\r\t"+recordSeparator)
			pairLines := strings.Split(assetChannels, subCtx.PairSeparator)
			pairs, ok := ap[a]
			if xpandPairs {
				if !ok {
					return nil, fmt.Errorf("%w: %s", asset.ErrInvalidAsset, a)
				}
				if len(pairLines) != len(pairs) {
					return nil, fmt.Errorf("%w: Got %d; Expected %d", errPairRecords, len(pairLines), len(pairs))
				}
			}
			for j, channel := range pairLines {
				c := s.Clone()
				c.Asset = a
				channel = strings.TrimSpace(channel)
				if channel == "" {
					return nil, fmt.Errorf("%w: %s", errNoTemplateContent, s)
				}
				c.QualifiedChannel = strings.TrimSpace(channel)
				if xpandPairs {
					c.Pairs = currency.Pairs{pairs[j]}
				} else {
					c.Pairs = pairs
				}
				subs = append(subs, c)
			}
		}
	}

	return subs, nil
}
