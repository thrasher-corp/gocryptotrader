package subscription

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	recordSeparator = "\x1E"
)

var (
	errRecordSeparator         = errors.New("subscription template must not contain the RecordSeparator character")
	errInvalidAssetExpandPairs = errors.New("subscription template containing $pair with must contain either specific Asset or $asset with asset.All")
	errTemplateLines           = errors.New("subscription template did not generate the expected number of lines")
	errAssetTemplateWithoutAll = errors.New("sub.Asset must be set to All if $asset is used in Channel template")
)

// List is a container of subscription pointers
type List []*Subscription

type assetPairs map[asset.Item]currency.Pairs

type iExchange interface {
	GetAssetTypes(enabled bool) asset.Items
	GetEnabledPairs(asset.Item) (currency.Pairs, error)
	GetPairFormat(asset.Item, bool) (currency.PairFormat, error)
	GetSubscriptionTemplateFuncs() template.FuncMap
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

type tplCtx struct {
	Sub        *Subscription
	AssetPairs assetPairs
	Assets     asset.Items
}

/*
ExpandTemplates returns a list of Subscriptions with Template expanded
Format of the Template should be text/template compatible.
Template Variables:
  - $s is the subscription; e.g. {{$s.Interval}} {{$s.Params.freq}}
  - $asset will fan out the enabled assets
    sub.Asset must be All; Otherwise just hardcode the asset the subscription is for
  - $pair will fan out the pairs for the assets, formatted for request
    Must be used in cojunction with $asset when Asset is All, otherwise we don't know what pairs to use
    May not be used when Asset is Empty

Calls e.GetSubscriptionTemplateFuncs for a template.FuncMap for flexibility in pipelines, e.g. {{ assetName "$asset" }}
Filters out Authenticated subscriptions if CanUseAuthenticatedEndpoints is false
*/
func (l List) ExpandTemplates(e iExchange) (List, error) {
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

	assets := make([]asset.Item, 0, len(ap))
	for k := range ap {
		assets = append(assets, k)
	}

	baseTpl := template.New("channel")
	if funcs := e.GetSubscriptionTemplateFuncs(); funcs != nil {
		baseTpl = baseTpl.Funcs(funcs)
	}

	subs := List{}

	for _, s := range l {
		if strings.Contains(s.Template, recordSeparator) {
			return nil, errRecordSeparator
		}

		subCtx := &tplCtx{
			Sub:        s,
			AssetPairs: ap,
		}

		tpl := s.Template + recordSeparator

		xpandPairs := strings.Contains(s.Template, "$pair")
		if xpandPairs {
			tpl = "{{range $pair := index $ctx.AssetPairs $asset}}" + tpl + "{{end}}"
		}

		if xpandAssets := strings.Contains(s.Template, "$asset"); xpandAssets {
			if s.Asset != asset.All {
				return nil, errAssetTemplateWithoutAll
			}
			subCtx.Assets = assets
			tpl = "{{range $asset := $ctx.Assets}}" + tpl + "{{end}}"
		} else {
			if xpandPairs && (s.Asset == asset.All || s.Asset == asset.Empty) {
				// We don't currently support expanding Pairs without expanding Assets for All or Empty assets, but we could; waiting for a use-case
				return nil, errInvalidAssetExpandPairs
			}
			subCtx.Assets = asset.Items{s.Asset}
			if s.Asset != asset.Empty {
				// Don't use asset.Empty as a with chain, because it will evaluate to false
				tpl = "{{with $asset := $ctx.Sub.Asset}}" + tpl + "{{end}}"
			}
		}

		tpl = "{{with $ctx := .}}{{with $s := $ctx.Sub}}" + tpl + "{{end}}{{end}}"

		t, err := baseTpl.Parse(tpl)
		if err != nil {
			return nil, fmt.Errorf("%w parsing %s", err, tpl)
		}

		buf := &bytes.Buffer{}
		if err := t.Execute(buf, subCtx); err != nil {
			return nil, err
		}

		channels := strings.Split(strings.TrimSuffix(buf.String(), recordSeparator), recordSeparator)

		i := 0
		line := func(a asset.Item, p currency.Pairs) {
			if i < len(channels) {
				c := s.Clone()
				c.Asset = a
				c.Pairs = p
				c.QualifiedChannel = channels[i]
				subs = append(subs, c)
			}
			i++ // Trigger errTemplateLines if we go over len(channels)
		}

		for _, a := range subCtx.Assets {
			if xpandPairs {
				for _, p := range ap[a] {
					line(a, currency.Pairs{p})
				}
			} else {
				line(a, ap[a])
			}
		}
		if i != len(channels) {
			return nil, fmt.Errorf("%w: Got %d Expected %d", errTemplateLines, len(channels), i)
		}
	}

	return subs, nil
}
