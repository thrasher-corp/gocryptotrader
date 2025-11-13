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
	errAssetRecords             = errors.New("subscription template did not generate the expected number of asset records")
	errPairRecords              = errors.New("subscription template did not generate the expected number of pair records")
	errTooManyBatchSizePerAsset = errors.New("more than one BatchSize directive inside an AssetSeparator")
	errNoTemplateContent        = errors.New("subscription template did not generate content")
	errInvalidTemplate          = errors.New("GetSubscriptionTemplate did not return a template")
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
// The exchange can optionally implement ListValidator to have custom validation on subscriptions
func (l List) ExpandTemplates(e IExchange) (List, error) {
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
			continue
		}

		subs = append(subs, expanded...)
	}

	// Validate the subscriptions after expansion to capture fields that will be used in the template
	if v, ok := e.(ListValidator); ok {
		// Need to check against the already stored subscriptions, as we add additional subscriptions
		storedSubs, err := e.GetSubscriptions()
		if err != nil {
			return nil, err
		}
		if err := v.ValidateSubscriptions(slices.Concat(subs, storedSubs)); err != nil {
			return nil, fmt.Errorf("error validating subscriptions: %w", err)
		}
	}

	return subs, err
}

func expandTemplate(e IExchange, s *Subscription, ap assetPairs, assets asset.Items) (List, error) {
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

	subs := List{}

	if len(s.Pairs) != 0 {
		// We deliberately do not check Availability of sub Pairs because users have edge cases to subscribe to non-existent pairs
		for a := range ap {
			ap[a] = s.Pairs
		}
	}

	switch s.Asset {
	case asset.All:
		if len(ap) == 0 {
			return List{}, nil // No assets enabled; only asset.Empty subs may continue
		}
		subCtx.AssetPairs = ap
	default:
		if s.Asset != asset.Empty && len(ap[s.Asset]) == 0 {
			return List{}, nil // No pairs enabled for this sub asset
		}
		// This deliberately includes asset.Empty to harmonise handling
		subCtx.AssetPairs = assetPairs{
			s.Asset: ap[s.Asset],
		}
		assets = asset.Items{s.Asset}
	}

	buf := &bytes.Buffer{}
	if err := t.Execute(buf, subCtx); err != nil {
		return nil, err
	}

	out := strings.TrimSpace(buf.String())

	// Remove a single trailing AssetSeparator; don't use a cutset to avoid removing 2 or more
	out = strings.TrimSpace(strings.TrimSuffix(out, subCtx.AssetSeparator))

	assetRecords := strings.Split(out, subCtx.AssetSeparator)
	if len(assetRecords) != len(assets) {
		return nil, fmt.Errorf("%w: Got %d; Expected %d", errAssetRecords, len(assetRecords), len(assets))
	}

	for i, assetChannels := range assetRecords {
		a := assets[i]
		pairs := subCtx.AssetPairs[a]

		xpandPairs := strings.Contains(assetChannels, subCtx.PairSeparator)

		/* Batching:
		- We start by assuming we'll get 1 batch sized to contain all pairs. Maybe a comma-separated list, or just the asset name
		- If a BatchSize directive is found, we expect it to come right at the end, and be followed by the batch size as a number
		- We'll then split into N batches of that size
		- If no batchSize was declared, but we saw a PairSeparator, then we expect to see one line per pair, so batchSize is 1
		*/
		batchSize := len(pairs)
		if b := strings.Split(assetChannels, subCtx.BatchSize); len(b) > 2 {
			return nil, fmt.Errorf("%w for %s", errTooManyBatchSizePerAsset, a)
		} else if len(b) == 2 {
			assetChannels = b[0]
			if batchSize, err = strconv.Atoi(strings.TrimSpace(b[1])); err != nil {
				return nil, fmt.Errorf("%s: %w", s, common.GetTypeAssertError("int", b[1], "batchSize"))
			}
		} else if xpandPairs {
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
			// The number of lines we get generated must match the number of pair batches we expect
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
