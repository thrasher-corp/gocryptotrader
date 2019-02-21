package currency

import (
	"errors"
	"fmt"
	"sync"

	log "github.com/thrasher-/gocryptotrader/logger"
)

// ConversionRates defines protected conversion rate map for concurrent updating
// and retrieval of foreign exchange rates
type ConversionRates struct {
	c   map[*code]map[*code]*float64
	mtx sync.Mutex
}

// HasData returns if conversion rates are present
func (c *ConversionRates) HasData() bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.c == nil {
		return false
	}
	return len(c.c) != 0
}

// GetRate returns a rate from the conversion rate list
func (c *ConversionRates) GetRate(from, to Code) (float64, error) {
	if from.C == USDT.C {
		from = USD
	}

	if to.C == USDT.C {
		to = USD
	}

	if from.C == RUR.C {
		from = RUB
	}

	if to.C == RUR.C {
		to = RUB
	}

	if from.C == to.C {
		return 1, nil
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	p, ok := c.c[from.C][to.C]
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
	p, ok := c.c[from.C][to.C]
	if !ok {
		log.Errorf("currency conversion rate not found from %s to %s", from, to)
		return Conversion{}, errors.New("no rate found, totally sucks")
	}

	i, ok := c.c[to.C][from.C]
	if !ok {
		log.Errorf("currency conversion inversion rate not found from %s to %s", to, from)
		return Conversion{}, errors.New("no rate found, totally sucks")
	}

	return Conversion{From: from, To: to, rate: p, mtx: &c.mtx, inverseRate: i},
		nil
}

// Update updates the full conversion rate values including inversion and
// cross rates
func (c *ConversionRates) Update(m map[string]float64) error {
	if len(m) == 0 {
		return errors.New("No data given")
	}

	solidvalues := make(map[Code]map[Code]float64)

	var list []Code // Verification list, cross check all currencies coming in

	var mainBaseCurrency Code

	for key, val := range m {
		code1 := NewCurrencyCode(key[:3])
		if mainBaseCurrency == (Code{}) {
			mainBaseCurrency = code1
		}
		code2 := NewCurrencyCode(key[3:])

		if code1 == code2 { // Get rid of same conversions
			continue
		}

		var codeOneFound, codeTwoFound bool
		// Check and add to our funky list
		for i := range list {
			if list[i] == code1 {
				codeOneFound = true
				if codeTwoFound {
					break
				}
			}

			if list[i] == code2 {
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

		if solidvalues[code1] == nil {
			solidvalues[code1] = make(map[Code]float64)
		}

		solidvalues[code1][code2] = val

		// Input inverse values 1/val to swap from -> to and vice versa

		if solidvalues[code2] == nil {
			solidvalues[code2] = make(map[Code]float64)
		}

		solidvalues[code2][code1] = 1 / val
	}

	for _, base := range list {
		for _, term := range list {
			if base == term {
				continue
			}
			_, ok := solidvalues[base][term]
			if !ok {
				var crossRate float64
				// Check inversion to speed things up
				v, ok := solidvalues[term][base]
				if !ok {
					v1, ok := solidvalues[mainBaseCurrency][base]
					if !ok {
						return fmt.Errorf("Value not found base %s term %s",
							mainBaseCurrency,
							base)
					}
					v2, ok := solidvalues[mainBaseCurrency][term]
					if !ok {
						return fmt.Errorf("Value not found base %s term %s",
							mainBaseCurrency,
							term)
					}
					crossRate = v2 / v1
				} else {
					crossRate = 1 / v
				}
				if system.IsVerbose() {
					log.Debugf("Conversion from %s to %s deriving cross rate value %f",
						base,
						term,
						crossRate)
				}
				solidvalues[base][term] = crossRate
			}
		}
	}

	c.mtx.Lock()
	for key, val := range solidvalues {
		for key2, val2 := range val {
			if c.c == nil {
				c.c = make(map[*code]map[*code]*float64)
			}

			if c.c[key.C] == nil {
				c.c[key.C] = make(map[*code]*float64)
			}

			p := c.c[key.C][key2.C]
			if p == nil {
				c.c[key.C][key2.C] = &val2
			} else {
				*p = val2
			}
		}
	}
	c.mtx.Unlock()
	return nil
}

// GetFullRates returns the full conversion list
func (c *ConversionRates) GetFullRates() Conversions {
	var conversions Conversions
	c.mtx.Lock()
	for key, val := range c.c {
		for key2, val2 := range val {
			conversions = append(conversions, Conversion{
				From: Code{C: key},
				To:   Code{C: key2},
				rate: val2,
				mtx:  &c.mtx,
			})
		}
	}
	c.mtx.Unlock()
	return conversions
}

// ExtractBase extracts base from loaded currency conversion rates
func (c *ConversionRates) ExtractBase() Code {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	for key := range c.c {
		return Code{C: key}
	}
	return Code{}
}

// Conversions define a list of conversion data
type Conversions []Conversion

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
	if c.From.C == nil || c.To.C == nil {
		return true
	}
	return c.From.C.name == c.To.C.name
}

// IsFiat checks to see if the from and to currency is a fiat e.g. EURUSD
func (c Conversion) IsFiat() bool {
	return system.IsFiatCurrency(c.From) && system.IsFiatCurrency(c.To)
}

// String returns the stringed fields
func (c Conversion) String() string {
	return c.From.String() + c.To.String()
}

// GetRate returns system rate if availabled
func (c Conversion) GetRate() (float64, error) {
	if c.mtx == nil {
		return 0, errors.New("mutex copy failure")
	}

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
		return 0, errors.New("Not fiat pair, sad days fellaz")
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
		return 0, errors.New("Not fiat pair, sad days fellaz")
	}

	r, err := c.GetInversionRate()
	if err != nil {
		return 0, err
	}

	return r * fromAmount, nil
}
