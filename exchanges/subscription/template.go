package subscription

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	deviceControl   = "\x11"
	groupSeparator  = "\x1D"
	recordSeparator = "\x1E"
)

var (
	errInvalidAssetExpandPairs = errors.New("subscription template containing PairSeparator with must contain either specific Asset or AssetSeparator")
	errAssetRecords            = errors.New("subscription template did not generate the expected number of asset records")
	errPairRecords             = errors.New("subscription template did not generate the expected number of pair records")
	errTooManyBatchSize        = errors.New("too many BatchSize directives")
	errAssetTemplateWithoutAll = errors.New("sub.Asset must be set to All if AssetSeparator is used in Channel template")
	errNoTemplateContent       = errors.New("subscription template did not generate content")
	errInvalidTemplate         = errors.New("GetSubscriptionTemplate did not return a template")
)

type tplCtx struct {
	S              *Subscription
	AssetPairs     assetPairs
	PairSeparator  string
	AssetSeparator string
	BatchSize      string
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
		expanded, err2 := expandTemplate(e, s, maps.Clone(ap), assets)
		if err2 != nil {
			err = common.AppendError(err, fmt.Errorf("%s: %w", s, err2))
		} else {
			subs = append(subs, expanded...)
		}
	}

	return subs, err
}

func expandTemplate(e iExchange, s *Subscription, ap assetPairs, assets asset.Items) (List, error) {
	if s.QualifiedChannel != "" {
		return List{s}, nil
	}

	t, err := e.GetSubscriptionTemplate(s)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errInvalidTemplate
	}

	subCtx := &tplCtx{
		S:              s,
		PairSeparator:  recordSeparator,
		AssetSeparator: groupSeparator,
		BatchSize:      deviceControl + "BS",
	}

	switch s.Asset {
	case asset.All:
		subCtx.AssetPairs = maps.Clone(ap)
	case asset.Empty:
		subCtx.AssetPairs = assetPairs{}
	default:
		subCtx.AssetPairs = assetPairs{
			s.Asset: ap[s.Asset],
		}
	}

	buf := &bytes.Buffer{}
	if err := t.Execute(buf, subCtx); err != nil { //nolint:govet // Shadow, or gocritic will complain sloppyReassign
		return nil, err
	}

	out := strings.TrimSpace(buf.String())

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
		assets = []asset.Item{s.Asset}
	}

	// Remove a single trailing AssetSeparator; don't use a cutset to avoid removing 2 or more
	out = strings.TrimSpace(strings.TrimSuffix(out, subCtx.AssetSeparator))

	assetRecords := strings.Split(out, subCtx.AssetSeparator)
	if len(assetRecords) != len(assets) {
		return nil, fmt.Errorf("%w: Got %d; Expected %d", errAssetRecords, len(assetRecords), len(assets))
	}

	subs := List{}
	for i, assetChannels := range assetRecords {
		a := assets[i]
		pairs := subCtx.AssetPairs[a]

		batchSize := len(pairs) // Default to all pairs in one batch
		if b := strings.Split(assetChannels, subCtx.BatchSize); len(b) > 2 {
			return nil, fmt.Errorf("%w for %s", errTooManyBatchSize, a)
		} else if len(b) == 2 { // If there's a batch size indicator we batch by that
			assetChannels = b[0]
			if batchSize, err = strconv.Atoi(strings.TrimSpace(b[1])); err != nil {
				return nil, fmt.Errorf("%s: %w", s, common.GetTypeAssertError("int", b[1], "batchSize"))
			}
		} else if xpandPairs { // expanding pairs but not batching so batch size is 1
			batchSize = 1
		}

		// Trim space, then only one pair separator, then any more space.
		assetChannels = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(assetChannels), subCtx.PairSeparator))

		if assetChannels == "" {
			continue
		}

		batches := common.Batch(pairs, batchSize)

		pairLines := strings.Split(assetChannels, subCtx.PairSeparator)

		if s.Asset != asset.Empty && len(pairLines) != len(batches) {
			return nil, fmt.Errorf("%w for %s: Got %d; Expected %d", errPairRecords, a, len(pairLines), len(batches))
		}

		for j, channel := range pairLines {
			c := s.Clone()
			c.Asset = a
			channel = strings.TrimSpace(channel)
			if channel == "" {
				return nil, fmt.Errorf("%w for %s: %s", errNoTemplateContent, a, s)
			}
			c.QualifiedChannel = channel
			if s.Asset != asset.Empty {
				c.Pairs = batches[j]
			}
			subs = append(subs, c)
		}
	}

	return subs, nil
}
