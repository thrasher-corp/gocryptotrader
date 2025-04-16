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

// IExchange provides method requirements for exchanges to use subscription templating
type IExchange interface {
	GetAssetTypes(enabled bool) asset.Items
	GetEnabledPairs(asset.Item) (currency.Pairs, error)
	GetPairFormat(asset.Item, bool) (currency.PairFormat, error)
	GetSubscriptionTemplate(*Subscription) (*template.Template, error)
	CanUseAuthenticatedWebsocketEndpoints() bool
	IsAssetWebsocketSupported(a asset.Item) bool
	GetSubscriptions() (List, error)
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

// GroupByPairs groups subscriptions which have the same Pairs
func (l List) GroupByPairs() []List {
	n := []List{}
outer:
	for _, a := range l {
		for i, b := range n {
			if a.Pairs.Equal(b[0].Pairs) { // Note: b is guaranteed to have 1 element by the append(n) below
				n[i] = append(n[i], a)
				continue outer
			}
		}
		n = append(n, List{a})
	}
	return n
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

// assetPairs returns a map of enabled pairs for the subscriptions in the list, formatted for the asset
func (l List) assetPairs(e IExchange) (assetPairs, error) {
	at := []asset.Item{}
	for _, a := range e.GetAssetTypes(true) {
		if e.IsAssetWebsocketSupported(a) {
			at = append(at, a)
		}
	}
	ap := assetPairs{}
	for _, s := range l {
		switch s.Asset {
		case asset.Empty:
			// Nothing to do
		case asset.All:
			for _, a := range at {
				if err := ap.populate(e, a); err != nil {
					return nil, err
				}
			}
		default:
			if slices.Contains(at, s.Asset) {
				if err := ap.populate(e, s.Asset); err != nil {
					return nil, err
				}
			}
		}
	}
	return ap, nil
}

// Enabled returns a new list of only enabled subscriptions
func (l List) Enabled() List {
	n := make(List, 0, len(l))
	for _, s := range l {
		if s.Enabled {
			n = append(n, s)
		}
	}
	return slices.Clip(n)
}

// Private returns only subscriptions which require authentication
func (l List) Private() List {
	n := List{}
	for _, s := range l {
		if s.Authenticated {
			n = append(n, s)
		}
	}
	return n
}

// Public returns only subscriptions which do not require authentication
func (l List) Public() List {
	n := List{}
	for _, s := range l {
		if !s.Authenticated {
			n = append(n, s)
		}
	}
	return n
}

// populate adds all enabled pairs for an asset to the assetPair map
func (ap assetPairs) populate(e IExchange, a asset.Item) error {
	p, err := e.GetEnabledPairs(a)
	if err != nil || len(p) == 0 {
		return err
	}
	f, err := e.GetPairFormat(a, true)
	if err != nil {
		return err
	}
	ap[a] = common.SortStrings(p.Format(f))
	return nil
}
