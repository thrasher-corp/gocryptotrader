package subscription

import (
	"slices"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// List is a container of subscription pointers
type List []*Subscription

// Strings returns a sorted slice of subscriptions
func (l List) Strings() []string {
	s := make([]string, len(l))
	for i := range l {
		s[i] = l[i].String()
	}
	slices.Sort(s)
	return s
}

// GroupPairs groups subscriptions which are identical apart from the Pairs
// The returned List contains cloned Subscriptions, and the original Subscriptions are left alone
func (l List) GroupPairs() (n List) {
	src := slices.Clone(l)
	for len(src) > 0 {
		s := src[0].Clone()
		key := &IgnoringPairsKey{s}
		n = append(n, s)
		// Remove the first element, and any which match it
		src = slices.DeleteFunc(src[1:], func(eachSub *Subscription) bool {
			if key.Match(&IgnoringPairsKey{eachSub}) {
				s.AddPairs(eachSub.Pairs...)
				return true
			}
			return false
		})
	}
	return
}

type assetPairs map[asset.Item]currency.Pairs
type iExchange interface {
	GetAssetTypes(enabled bool) asset.Items
	GetEnabledPairs(asset.Item) (currency.Pairs, error)
	GetPairFormat(asset.Item, bool) (currency.PairFormat, error)
}

func fillAssetPairs(ap assetPairs, a asset.Item, e iExchange) error {
	p, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	f, err := e.GetPairFormat(a, true)
	if err != nil {
		return err
	}
	ap[a] = p.Format(f)
	return nil
}

// AssetPairs returns a map of enabled pairs for the subscriptions in the list, formatted for the asset
func (l List) AssetPairs(e iExchange) (assetPairs, error) { //nolint:revive // unexported-return is acceptable since it's a undecorated primitive
	at := e.GetAssetTypes(true)
	ap := assetPairs{}
	for _, s := range l {
		switch s.Asset {
		case asset.Empty:
			// Nothing to do
		case asset.All:
			for _, a := range at {
				if err := fillAssetPairs(ap, a, e); err != nil {
					return nil, err
				}
			}
		default:
			if slices.Contains(at, s.Asset) {
				if err := fillAssetPairs(ap, s.Asset, e); err != nil {
					return nil, err
				}
			}
		}
	}
	return ap, nil
}
