package currency

// GetTranslation returns similar strings for a particular currency if not found
// returns the code back
func GetTranslation(currency Code) Code {
	val, ok := translations[currency.Item]
	if !ok {
		return currency
	}
	return val
}

var translations = map[*Item]Code{
	BTC.Item:  XBT,
	ETH.Item:  XETH,
	DOGE.Item: XDG,
	USD.Item:  USDT,
	XBT.Item:  BTC,
	XETH.Item: ETH,
	XDG.Item:  DOGE,
	USDT.Item: USD,
}

// Translations is a map of translations for a specific exchange implementation
type Translations map[*Item]Code

// NewTranslations returns a new translation map, the key indicates the exchange
// representation and the value indicates the internal representation/common/standard
// representation. e.g. XBT as key and BTC as value, this is useful for exchanges
// that use different naming conventions.
// TODO: Expand for specific assets.
func NewTranslations(t map[Code]Code) Translations {
	lookup := make(map[*Item]Code)
	for k, v := range t {
		lookup[k.Item] = v
	}
	return lookup
}

// Translate returns the translated currency code, usually used to convert
// exchange specific currency codes to common currency codes. If no translation
// is found it will return the original currency code.
// TODO: Add TranslateToCommon and TranslateToExchange methods to allow for
// translation to and from exchange specific currency codes.
func (t Translations) Translate(incoming Code) Code {
	if len(t) == 0 {
		return incoming
	}
	val, ok := (t)[incoming.Item]
	if !ok {
		return incoming
	}
	return val
}

// Translator is an interface for translating currency codes
type Translator interface {
	// TODO: Add a asset.Item param so that we can translate for asset
	// permutations. Also return error.
	Translate(Code) Code
}

// PairsWithTranslation is a pair list with a translator for a specific exchange.
type PairsWithTranslation struct {
	Pairs      Pairs
	Translator Translator
}

// keyPair defines an immutable pair for lookup purposes
type keyPair struct {
	Base  *Item
	Quote *Item
}

// FindMatchingPairsBetween returns all pairs that match the incoming pairs.
// Translator is used to convert exchange specific currency codes to common
// currency codes used in lookup process. The pairs are not modified. So that
// the original pairs are returned for deployment to the specific exchange.
// NOTE: Translator is optional and can be nil. Translator can be obtained from
// the exchange implementation by calling Base() method and accessing Features
// and Translation fields.
func FindMatchingPairsBetween(this, that PairsWithTranslation) map[Pair]Pair {
	lookup := make(map[keyPair]*Pair)
	var k keyPair
	for i := range this.Pairs {
		if this.Translator != nil {
			k = keyPair{Base: this.Translator.Translate(this.Pairs[i].Base).Item, Quote: this.Translator.Translate(this.Pairs[i].Quote).Item}
			lookup[k] = &this.Pairs[i]
			continue
		}
		lookup[keyPair{Base: this.Pairs[i].Base.Item, Quote: this.Pairs[i].Quote.Item}] = &this.Pairs[i]
	}
	outgoing := make(map[Pair]Pair)
	for i := range that.Pairs {
		if that.Translator != nil {
			k = keyPair{Base: that.Translator.Translate(that.Pairs[i].Base).Item, Quote: that.Translator.Translate(that.Pairs[i].Quote).Item}
		} else {
			k = keyPair{Base: that.Pairs[i].Base.Item, Quote: that.Pairs[i].Quote.Item}
		}
		if p, ok := lookup[k]; ok {
			outgoing[*p] = that.Pairs[i]
		}
	}
	return outgoing
}
