package subscription

import (
	"slices"
	"text/template"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// List is a container of subscription pointers
type List []*Subscription

type assetPairs map[asset.Item]currency.Pairs

type iExchange interface {
	GetAssetTypes(enabled bool) asset.Items
	GetEnabledPairs(asset.Item) (currency.Pairs, error)
	GetPairFormat(asset.Item, bool) (currency.PairFormat, error)
	GetSubscriptionTemplate(*Subscription) (*template.Template, error)
	CanUseAuthenticatedWebsocketEndpoints() bool
}

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

// Clone returns a deep clone of the List
func (l List) Clone() List {
	n := make(List, len(l))
	for i, s := range l {
		n[i] = s.Clone()
	}
	return n
}

// QualifiedChannels returns a sorted list of all the qualified Channels in the list
func (l List) QualifiedChannels() []string {
	c := make([]string, len(l))
	for i := range l {
		c[i] = l[i].QualifiedChannel
	}
	slices.Sort(c)
	return c
}

// SetStates sets the state for all the subs in a list
// Errors are collected for any subscriptions already in the state
// On error all changes are reverted
func (l List) SetStates(state State) error {
	var err error
	for _, sub := range l {
		err = common.AppendError(err, sub.SetState(state))
	}
	return err
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
	ap[a] = common.SortStrings(p.Format(f))
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
