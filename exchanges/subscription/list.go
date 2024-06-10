package subscription

import (
	"slices"
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
	s := NewStore()
	for _, sub := range l {
		if found := s.match(&IgnoringPairsKey{sub}); found == nil {
			s.unsafeAdd(sub.Clone())
		} else {
			found.AddPairs(sub.Pairs...)
		}
	}
	return s.List()
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

// assetPairs returns a map of enabled pairs for the subscriptions in the list, formatted for the asset
func (l List) assetPairs(e iExchange) (assetPairs, error) {
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
