package poloniex

import (
	"errors"
	"strconv"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// CurrencyDetails stores a map of currencies associated with their ID
type CurrencyDetails struct {
	pairs map[float64]PairSummaryInfo
	codes map[float64]CodeSummaryInfo
	// Mutex used for future when we will periodically update this table every
	// week or so in production
	m sync.RWMutex
}

// PairSummaryInfo defines currency pair information
type PairSummaryInfo struct {
	Pair     currency.Pair
	IsFrozen bool
	PostOnly bool
}

// CodeSummaryInfo defines currency information
type CodeSummaryInfo struct {
	Currency                  currency.Code
	WithdrawalTXFee           float64
	MinimumConfirmations      int64
	DepositAddress            string
	WithdrawalDepositDisabled bool
	Frozen                    bool
}

var (
	errCannotLoadNoData      = errors.New("cannot load websocket currency data as data is nil")
	errNoDepositAddress      = errors.New("no public deposit address for currency")
	errPairMapIsNil          = errors.New("cannot get currency pair, map is nil")
	errCodeMapIsNil          = errors.New("cannot get currency code, map is nil")
	errCurrencyNotFoundInMap = errors.New("currency not found")
)

// loadPairs loads currency pair associations with unique identifiers from
// ticker data map
func (w *CurrencyDetails) loadPairs(data map[string]Ticker) error {
	if data == nil {
		return errCannotLoadNoData
	}
	w.m.Lock()
	defer w.m.Unlock()
	for k, v := range data {
		pair, err := currency.NewPairFromString(k)
		if err != nil {
			return err
		}

		if w.pairs == nil {
			w.pairs = make(map[float64]PairSummaryInfo)
		}
		w.pairs[v.ID] = PairSummaryInfo{
			Pair:     pair,
			IsFrozen: v.IsFrozen == 1,
			PostOnly: v.PostOnly == 1,
		}
	}
	return nil
}

// loadCodes loads currency codes from currency map
func (w *CurrencyDetails) loadCodes(data map[string]*Currencies) error {
	if data == nil {
		return errCannotLoadNoData
	}
	w.m.Lock()
	defer w.m.Unlock()
	for k, v := range data {
		if v.Delisted == 1 {
			continue
		}

		if w.codes == nil {
			w.codes = make(map[float64]CodeSummaryInfo)
		}

		w.codes[v.ID] = CodeSummaryInfo{
			Currency:                  currency.NewCode(k),
			WithdrawalTXFee:           v.TxFee,
			MinimumConfirmations:      v.MinConfirmations,
			DepositAddress:            v.DepositAddress,
			WithdrawalDepositDisabled: v.WithdrawalDepositDisabled == 1,
			Frozen:                    v.Frozen == 1,
		}
	}
	return nil
}

// GetPair returns a currency pair by its ID
func (w *CurrencyDetails) GetPair(id float64) (currency.Pair, error) {
	w.m.RLock()
	defer w.m.RUnlock()
	if w.pairs == nil {
		return currency.EMPTYPAIR, errPairMapIsNil
	}

	p, ok := w.pairs[id]
	if ok {
		return p.Pair, nil
	}

	// This is here so we can still log an order with the ID as the currency
	// pair which you can then cross reference later with the exchange ID list,
	// rather than error out.
	op, err := currency.NewPairFromString(strconv.FormatFloat(id, 'f', -1, 64))
	if err != nil {
		return op, err
	}
	return op, errIDNotFoundInPairMap
}

// GetCode returns a currency code by its ID
func (w *CurrencyDetails) GetCode(id float64) (currency.Code, error) {
	w.m.RLock()
	defer w.m.RUnlock()
	if w.codes == nil {
		return currency.EMPTYCODE, errCodeMapIsNil
	}
	c, ok := w.codes[id]
	if ok {
		return c.Currency, nil
	}
	return currency.EMPTYCODE, errIDNotFoundInCodeMap
}

// GetWithdrawalTXFee returns withdrawal transaction fee for the currency
func (w *CurrencyDetails) GetWithdrawalTXFee(c currency.Code) (float64, error) {
	w.m.RLock()
	defer w.m.RUnlock()
	if w.codes == nil {
		return 0, errCodeMapIsNil
	}
	for _, v := range w.codes {
		if v.Currency == c {
			return v.WithdrawalTXFee, nil
		}
	}
	return 0, errCurrencyNotFoundInMap
}

// GetDepositAddress returns the public deposit address details for the currency
func (w *CurrencyDetails) GetDepositAddress(c currency.Code) (string, error) {
	w.m.RLock()
	defer w.m.RUnlock()
	if w.codes == nil {
		return "", errCodeMapIsNil
	}
	for _, v := range w.codes {
		if v.Currency == c {
			if v.DepositAddress == "" {
				return "", errNoDepositAddress
			}
			return v.DepositAddress, nil
		}
	}
	return "", errCurrencyNotFoundInMap
}

// IsWithdrawAndDepositsEnabled returns if withdrawals or deposits are enabled
func (w *CurrencyDetails) IsWithdrawAndDepositsEnabled(c currency.Code) (bool, error) {
	w.m.RLock()
	defer w.m.RUnlock()
	if w.codes == nil {
		return false, errCodeMapIsNil
	}
	for _, v := range w.codes {
		if v.Currency == c {
			return !v.WithdrawalDepositDisabled, nil
		}
	}
	return false, errCurrencyNotFoundInMap
}

// IsTradingEnabledForCurrency returns if the currency is allowed to be traded
func (w *CurrencyDetails) IsTradingEnabledForCurrency(c currency.Code) (bool, error) {
	w.m.RLock()
	defer w.m.RUnlock()
	if w.codes == nil {
		return false, errCodeMapIsNil
	}
	for _, v := range w.codes {
		if v.Currency == c {
			return !v.Frozen, nil
		}
	}
	return false, errCurrencyNotFoundInMap
}

// IsTradingEnabledForPair returns if the currency pair is allowed to be traded
func (w *CurrencyDetails) IsTradingEnabledForPair(pair currency.Pair) (bool, error) {
	w.m.RLock()
	defer w.m.RUnlock()
	if w.codes == nil {
		return false, errCodeMapIsNil
	}
	for _, v := range w.pairs {
		if v.Pair.Equal(pair) {
			return !v.IsFrozen, nil
		}
	}
	return false, errCurrencyNotFoundInMap
}

// IsPostOnlyForPair returns if an order is allowed to take liquidity from the
// books or reduce positions
func (w *CurrencyDetails) IsPostOnlyForPair(pair currency.Pair) (bool, error) {
	w.m.RLock()
	defer w.m.RUnlock()
	if w.codes == nil {
		return false, errCodeMapIsNil
	}
	for _, v := range w.pairs {
		if v.Pair.Equal(pair) {
			return v.PostOnly, nil
		}
	}
	return false, errCurrencyNotFoundInMap
}

// isInitial checks state of maps to determine if they have been loaded or not
func (w *CurrencyDetails) isInitial() bool {
	w.m.RLock()
	defer w.m.RUnlock()
	return w.codes == nil || w.pairs == nil
}
