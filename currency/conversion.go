package currency

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/log"
)

// ConversionRates defines protected conversion rate map for concurrent updating
// and retrieval of foreign exchange rates for mainly fiat currencies
type ConversionRates struct {
	m   map[*Item]map[*Item]*float64
	mtx sync.Mutex
}

// HasData returns if conversion rates are present
func (c *ConversionRates) HasData() bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return len(c.m) != 0
}

// GetRate returns a rate from the conversion rate list
func (c *ConversionRates) GetRate(from, to Code) (float64, error) {
	if from.Item == USDT.Item {
		from = USD
	}

	if to.Item == USDT.Item {
		to = USD
	}

	if from.Item == RUR.Item {
		from = RUB
	}

	if to.Item == RUR.Item {
		to = RUB
	}

	if from.Item == to.Item {
		return 1, nil
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	p, ok := c.m[from.Item][to.Item]
	if !ok {
		return 0, fmt.Errorf("rate not found for from %s to %s conversion",
			from,
			to)
	}

	return *p, nil
}

// Register registers a new conversion rate if not found adds it and allows for
// quick updates
func (c *ConversionRates) Register(from, to Code) (Conversion, error) {
	if from.IsCryptocurrency() {
		return Conversion{}, errors.New("from currency is a cryptocurrency value")
	}

	if to.IsCryptocurrency() {
		return Conversion{}, errors.New("to currency is a cryptocurrency value")
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	p, ok := c.m[from.Item][to.Item]
	if !ok {
		log.Errorf(log.Global,
			"currency conversion rate not found from %s to %s\n", from, to)
		return Conversion{}, errors.New("no rate found")
	}

	i, ok := c.m[to.Item][from.Item]
	if !ok {
		log.Errorf(log.Global,
			"currency conversion inversion rate not found from %s to %s\n",
			to,
			from)
		return Conversion{}, errors.New("no rate found")
	}

	return Conversion{From: from, To: to, rate: p, mtx: &c.mtx, inverseRate: i},
		nil
}

// Update updates the full conversion rate values including inversion and
// cross rates
func (c *ConversionRates) Update(m map[string]float64) error {
	if len(m) == 0 {
		return errors.New("no data given")
	}

	if storage.IsVerbose() {
		log.Debugln(log.Global, "Conversion rates are being updated.")
	}

	solidvalues := make(map[*Item]map[*Item]float64)

	var list []Code // Verification list, cross check all currencies coming in

	var mainBaseCurrency Code
	for key, val := range m {
		code1 := storage.ValidateFiatCode(key[:3])

		if mainBaseCurrency.Equal(EMPTYCODE) {
			mainBaseCurrency = code1
		}

		code2 := storage.ValidateFiatCode(key[3:])
		if code1.Equal(code2) { // Get rid of same conversions
			continue
		}

		var codeOneFound, codeTwoFound bool
		// Check and add to our funky list
		for i := range list {
			if list[i].Equal(code1) {
				codeOneFound = true
				if codeTwoFound {
					break
				}
			}

			if list[i].Equal(code2) {
				codeTwoFound = true
				if codeOneFound {
					break
				}
			}
		}

		if !codeOneFound {
			list = append(list, code1)
		}

		if !codeTwoFound {
			list = append(list, code2)
		}

		if solidvalues[code1.Item] == nil {
			solidvalues[code1.Item] = make(map[*Item]float64)
		}

		solidvalues[code1.Item][code2.Item] = val

		// Input inverse values 1/val to swap from -> to and vice versa

		if solidvalues[code2.Item] == nil {
			solidvalues[code2.Item] = make(map[*Item]float64)
		}

		solidvalues[code2.Item][code1.Item] = 1 / val
	}

	for _, base := range list {
		for _, term := range list {
			if base.Equal(term) {
				continue
			}
			_, ok := solidvalues[base.Item][term.Item]
			if !ok {
				var crossRate float64
				// Check inversion to speed things up
				v, ok := solidvalues[term.Item][base.Item]
				if !ok {
					v1, ok := solidvalues[mainBaseCurrency.Item][base.Item]
					if !ok {
						return fmt.Errorf("value not found base %s term %s",
							mainBaseCurrency,
							base)
					}
					v2, ok := solidvalues[mainBaseCurrency.Item][term.Item]
					if !ok {
						return fmt.Errorf("value not found base %s term %s",
							mainBaseCurrency,
							term)
					}
					crossRate = v2 / v1
				} else {
					crossRate = 1 / v
				}
				if storage.IsVerbose() {
					log.Debugf(log.Global,
						"Conversion from %s to %s deriving cross rate value %f\n",
						base,
						term,
						crossRate)
				}
				solidvalues[base.Item][term.Item] = crossRate
			}
		}
	}

	c.m = nil
	for key, val := range solidvalues {
		for key2, val2 := range val {
			if c.m == nil {
				c.m = make(map[*Item]map[*Item]*float64)
			}

			if c.m[key] == nil {
				c.m[key] = make(map[*Item]*float64)
			}

			p := c.m[key][key2]
			if p == nil {
				newPalsAndFriends := val2
				c.m[key][key2] = &newPalsAndFriends
			} else {
				*p = val2
			}
		}
	}
	return nil
}

// GetFullRates returns the full conversion list
func (c *ConversionRates) GetFullRates() Conversions {
	var conversions Conversions
	c.mtx.Lock()
	for key, val := range c.m {
		for key2, val2 := range val {
			conversions = append(conversions, Conversion{
				From: Code{Item: key},
				To:   Code{Item: key2},
				rate: val2,
				mtx:  &c.mtx,
			})
		}
	}
	c.mtx.Unlock()
	return conversions
}

// Conversions define a list of conversion data
type Conversions []Conversion

// NewConversionFromString splits a string from a foreign exchange provider
func NewConversionFromString(p string) (Conversion, error) {
	return NewConversionFromStrings(p[:3], p[3:])
}

// NewConversion returns a conversion rate object that allows for
// obtaining efficient rate values when needed
func NewConversion(from, to Code) (Conversion, error) {
	return storage.NewConversion(from, to)
}

// NewConversionFromStrings assigns or finds a new conversion unit
func NewConversionFromStrings(from, to string) (Conversion, error) {
	return NewConversion(NewCode(from), NewCode(to))
}

// Conversion defines a specific currency conversion for a rate
type Conversion struct {
	From        Code
	To          Code
	rate        *float64
	inverseRate *float64
	mtx         *sync.Mutex
}

// IsInvalid returns true if both from and to currencies are the same
func (c Conversion) IsInvalid() bool {
	if c.From.Item == nil || c.To.Item == nil {
		return true
	}
	return c.From.Item == c.To.Item
}

// IsFiat checks to see if the from and to currency is a fiat e.g. EURUSD
func (c Conversion) IsFiat() bool {
	return c.From.IsFiatCurrency() && c.To.IsFiatCurrency()
}

// String returns the stringed fields
func (c Conversion) String() string {
	return c.From.String() + c.To.String()
}

// GetRate returns system rate if available
func (c Conversion) GetRate() (float64, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.rate == nil {
		return 0, errors.New("rate undefined")
	}
	return *c.rate, nil
}

// GetInversionRate returns the rate of the inversion of the conversion pair
func (c Conversion) GetInversionRate() (float64, error) {
	if c.mtx == nil {
		return 0, errors.New("mutex copy failure")
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.rate == nil {
		return 0, errors.New("rate undefined")
	}
	return *c.inverseRate, nil
}

// Convert for example converts $1 USD to the equivalent Japanese Yen or vice
// versa.
func (c Conversion) Convert(fromAmount float64) (float64, error) {
	if c.IsInvalid() {
		return fromAmount, nil
	}

	if !c.IsFiat() {
		return 0, errors.New("not fiat pair")
	}

	r, err := c.GetRate()
	if err != nil {
		return 0, err
	}

	return r * fromAmount, nil
}

// ConvertInverse converts backwards if needed
func (c Conversion) ConvertInverse(fromAmount float64) (float64, error) {
	if c.IsInvalid() {
		return fromAmount, nil
	}

	if !c.IsFiat() {
		return 0, errors.New("not fiat pair")
	}

	r, err := c.GetInversionRate()
	if err != nil {
		return 0, err
	}

	return r * fromAmount, nil
}
