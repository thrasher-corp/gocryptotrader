package currency

import (
	"errors"
	"fmt"
	"unicode"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// EMPTYFORMAT defines an empty pair format
var EMPTYFORMAT = PairFormat{}

// ErrCurrencyNotAssociatedWithPair defines an error where a currency is not
// associated with a pair.
var ErrCurrencyNotAssociatedWithPair = errors.New("currency not associated with pair")

// String returns a currency pair string
func (p Pair) String() string {
	return p.Base.String() + p.Delimiter + p.Quote.String()
}

// Lower converts the pair object to lowercase
func (p Pair) Lower() Pair {
	p.Base = p.Base.Lower()
	p.Quote = p.Quote.Lower()
	return p
}

// Upper converts the pair object to uppercase
func (p Pair) Upper() Pair {
	p.Base = p.Base.Upper()
	p.Quote = p.Quote.Upper()
	return p
}

// UnmarshalJSON implements json.Unmarshaler
func (p *Pair) UnmarshalJSON(d []byte) error {
	var pair string
	err := json.Unmarshal(d, &pair)
	if err != nil {
		return err
	}

	if pair == "" {
		*p = EMPTYPAIR
		return nil
	}

	// Check if pair is in the format of BTC-USD
	for x := range pair {
		if unicode.IsPunct(rune(pair[x])) {
			p.Base = NewCode(pair[:x])
			p.Delimiter = string(pair[x])
			p.Quote = NewCode(pair[x+1:])
			return nil
		}
	}

	// NOTE: Pair string could be in format DUSKUSDT (Kucoin) which will be
	// incorrectly converted to DUS-KUSDT, ELKRW (Bithumb) which will convert
	// converted to ELK-RW and HTUSDT (Lbank) which will be incorrectly
	// converted to HTU-SDT.
	return fmt.Errorf("%w from %s cannot ensure pair is in correct format, please use exchange method MatchSymbolWithAvailablePairs", errCannotCreatePair, pair)
}

// MarshalJSON conforms type to the marshaler interface
func (p Pair) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// Format changes the currency based on user preferences overriding the default
// String() display
func (p Pair) Format(pf PairFormat) Pair {
	p.Delimiter = pf.Delimiter
	if pf.Uppercase {
		return p.Upper()
	}
	return p.Lower()
}

// Equal compares two currency pairs and returns whether or not they are equal
func (p Pair) Equal(cPair Pair) bool {
	return p.Base.Equal(cPair.Base) && p.Quote.Equal(cPair.Quote)
}

// EqualIncludeReciprocal compares two currency pairs and returns whether or not
// they are the same including reciprocal currencies.
func (p Pair) EqualIncludeReciprocal(cPair Pair) bool {
	return (p.Base.Equal(cPair.Base) && p.Quote.Equal(cPair.Quote)) ||
		(p.Base.Equal(cPair.Quote) && p.Quote.Equal(cPair.Base))
}

// IsCryptoPair checks to see if the pair is a crypto pair e.g. BTCLTC
func (p Pair) IsCryptoPair() bool {
	return p.Base.IsCryptocurrency() && p.Quote.IsCryptocurrency()
}

// IsCryptoFiatPair checks to see if the pair is a crypto fiat pair e.g. BTCUSD
func (p Pair) IsCryptoFiatPair() bool {
	return (p.Base.IsCryptocurrency() && p.Quote.IsFiatCurrency()) ||
		(p.Base.IsFiatCurrency() && p.Quote.IsCryptocurrency())
}

// IsFiatPair checks to see if the pair is a fiat pair e.g. EURUSD
func (p Pair) IsFiatPair() bool {
	return p.Base.IsFiatCurrency() && p.Quote.IsFiatCurrency()
}

// IsCryptoStablePair checks to see if the pair is a crypto stable pair e.g.
// LTC-USDT
func (p Pair) IsCryptoStablePair() bool {
	return (p.Base.IsCryptocurrency() && p.Quote.IsStableCurrency()) ||
		(p.Base.IsStableCurrency() && p.Quote.IsCryptocurrency())
}

// IsStablePair checks to see if the pair is a stable pair e.g. USDT-DAI
func (p Pair) IsStablePair() bool {
	return p.Base.IsStableCurrency() && p.Quote.IsStableCurrency()
}

// IsInvalid checks invalid pair if base and quote are the same
func (p Pair) IsInvalid() bool {
	return p.Base.Equal(p.Quote)
}

// Swap turns the currency pair into its reciprocal
func (p Pair) Swap() Pair {
	return Pair{Base: p.Quote, Quote: p.Base}
}

// IsEmpty returns whether or not the pair is empty or is missing a currency
// code
func (p Pair) IsEmpty() bool {
	return p.Base.IsEmpty() && p.Quote.IsEmpty()
}

// Contains checks to see if a pair contains a specific currency
func (p Pair) Contains(c Code) bool {
	return p.Base.Equal(c) || p.Quote.Equal(c)
}

// Len derives full length for match exclusion.
func (p Pair) Len() int {
	return len(p.Base.String()) + len(p.Quote.String())
}

// Other returns the other currency from pair, if not matched returns empty code.
func (p Pair) Other(c Code) (Code, error) {
	if p.Base.Equal(c) {
		return p.Quote, nil
	}
	if p.Quote.Equal(c) {
		return p.Base, nil
	}
	return EMPTYCODE, ErrCurrencyCodeEmpty
}

// IsPopulated returns true if the currency pair have both non-empty values for
// base and quote.
func (p Pair) IsPopulated() bool {
	return !p.Base.IsEmpty() && !p.Quote.IsEmpty()
}

// MarketSellOrderParameters returns order parameters for when you want to sell
// a currency which purchases another currency. This specifically returns what
// liquidity side you will be affecting, what order side you will be placing and
// what currency you will be purchasing.
func (p Pair) MarketSellOrderParameters(wantingToSell Code) (*OrderParameters, error) {
	return p.getOrderParameters(wantingToSell, true, true)
}

// MarketBuyOrderParameters returns order parameters for when you want to sell a
// currency which purchases another currency. This specifically returns what
// liquidity side you will be affecting, what order side you will be placing and
// what currency you will be purchasing.
func (p Pair) MarketBuyOrderParameters(wantingToBuy Code) (*OrderParameters, error) {
	return p.getOrderParameters(wantingToBuy, false, true)
}

// LimitSellOrderParameters returns order parameters for when you want to sell a
// currency which purchases another currency. This specifically returns what
// liquidity side you will be affecting, what order side you will be placing and
// what currency you will be purchasing.
func (p Pair) LimitSellOrderParameters(wantingToSell Code) (*OrderParameters, error) {
	return p.getOrderParameters(wantingToSell, true, false)
}

// LimitBuyOrderParameters returns order parameters for when you want to
// sell a currency which purchases another currency. This specifically returns
// what liquidity side you will be affecting, what order side you will be
// placing and what currency you will be purchasing.
func (p Pair) LimitBuyOrderParameters(wantingToBuy Code) (*OrderParameters, error) {
	return p.getOrderParameters(wantingToBuy, false, false)
}

// getOrderParameters returns order parameters for the currency pair using
// the provided currency code, whether or not you are selling and whether or not
// you are placing a market order.
func (p Pair) getOrderParameters(c Code, selling, market bool) (*OrderParameters, error) {
	if !p.IsPopulated() {
		return nil, ErrCurrencyPairEmpty
	}
	if c.IsEmpty() {
		return nil, ErrCurrencyCodeEmpty
	}
	params := OrderParameters{}
	switch {
	case p.Base.Equal(c):
		if selling {
			params.SellingCurrency = p.Base
			params.PurchasingCurrency = p.Quote
			params.IsAskLiquidity = !market
		} else {
			params.SellingCurrency = p.Quote
			params.PurchasingCurrency = p.Base
			params.IsBuySide = true
			params.IsAskLiquidity = market
		}
	case p.Quote.Equal(c):
		if selling {
			params.SellingCurrency = p.Quote
			params.PurchasingCurrency = p.Base
			params.IsBuySide = true
			params.IsAskLiquidity = market
		} else {
			params.SellingCurrency = p.Base
			params.PurchasingCurrency = p.Quote
			params.IsAskLiquidity = !market
		}
	default:
		return nil, fmt.Errorf("%w %v: %v", ErrCurrencyNotAssociatedWithPair, c, p)
	}
	params.Pair = p
	return &params, nil
}

// IsAssociated checks to see if the pair is associated with another pair
func (p Pair) IsAssociated(a Pair) bool {
	return p.Base.Equal(a.Base) || p.Quote.Equal(a.Base) || p.Base.Equal(a.Quote) || p.Quote.Equal(a.Quote)
}
