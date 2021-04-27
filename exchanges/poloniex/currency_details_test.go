package poloniex

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestWsCurrencyMap(t *testing.T) {
	var m CurrencyDetails

	if !m.isInitial() {
		t.Fatal("unexpected value")
	}

	err := m.loadPairs(nil)
	if !errors.Is(err, errCannotLoadNoData) {
		t.Fatalf("expected: %v but received: %v", errCannotLoadNoData, err)
	}

	err = m.loadCodes(nil)
	if !errors.Is(err, errCannotLoadNoData) {
		t.Fatalf("expected: %v but received: %v", errCannotLoadNoData, err)
	}

	_, err = m.GetPair(1337)
	if !errors.Is(err, errPairMapIsNil) {
		t.Fatalf("expected: %v but received: %v", errPairMapIsNil, err)
	}

	_, err = m.GetCode(1337)
	if !errors.Is(err, errCodeMapIsNil) {
		t.Fatalf("expected: %v but received: %v", errCodeMapIsNil, err)
	}

	_, err = m.GetWithdrawalTXFee(currency.Code{})
	if !errors.Is(err, errCodeMapIsNil) {
		t.Fatalf("expected: %v but received: %v", errCodeMapIsNil, err)
	}

	_, err = m.GetDepositAddress(currency.Code{})
	if !errors.Is(err, errCodeMapIsNil) {
		t.Fatalf("expected: %v but received: %v", errCodeMapIsNil, err)
	}

	_, err = m.IsWithdrawAndDepositsEnabled(currency.Code{})
	if !errors.Is(err, errCodeMapIsNil) {
		t.Fatalf("expected: %v but received: %v", errCodeMapIsNil, err)
	}

	_, err = m.IsTradingEnabledForCurrency(currency.Code{})
	if !errors.Is(err, errCodeMapIsNil) {
		t.Fatalf("expected: %v but received: %v", errCodeMapIsNil, err)
	}

	_, err = m.IsTradingEnabledForPair(currency.Pair{})
	if !errors.Is(err, errCodeMapIsNil) {
		t.Fatalf("expected: %v but received: %v", errCodeMapIsNil, err)
	}

	_, err = m.IsPostOnlyForPair(currency.Pair{})
	if !errors.Is(err, errCodeMapIsNil) {
		t.Fatalf("expected: %v but received: %v", errCodeMapIsNil, err)
	}

	c, err := p.GetCurrencies()
	if err != nil {
		t.Fatal(err)
	}

	err = m.loadCodes(c)
	if err != nil {
		t.Fatal(err)
	}

	tick, err := p.GetTicker()
	if err != nil {
		t.Fatal(err)
	}

	err = m.loadPairs(tick)
	if err != nil {
		t.Fatal(err)
	}

	pTest, err := m.GetPair(1337)
	if !errors.Is(err, errIDNotFoundInPairMap) {
		t.Fatalf("expected: %v but received: %v", errIDNotFoundInPairMap, err)
	}

	if pTest.String() != "1337" {
		t.Fatal("unexpected value")
	}

	_, err = m.GetCode(1337)
	if !errors.Is(err, errIDNotFoundInCodeMap) {
		t.Fatalf("expected: %v but received: %v", errIDNotFoundInCodeMap, err)
	}

	btcusdt, err := m.GetPair(121)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if btcusdt.String() != "USDT_BTC" {
		t.Fatal("expecting USDT_BTC pair")
	}

	maid, err := m.GetCode(127)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if maid.String() != "MAID" {
		t.Fatal("unexpected value")
	}

	txFee, err := m.GetWithdrawalTXFee(maid)
	if err != nil {
		t.Fatal(err)
	}

	if txFee != 80 {
		t.Fatal("unexpected value")
	}

	_, err = m.GetDepositAddress(maid)
	if !errors.Is(err, errNoDepositAddress) {
		t.Fatalf("expected: %v but received: %v", errNoDepositAddress, err)
	}

	dAddr, err := m.GetDepositAddress(currency.NewCode("BCN"))
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if dAddr != "25cZNQYVAi3issDCoa6fWA2Aogd4FgPhYdpX3p8KLfhKC6sN8s6Q9WpcW4778TPwcUS5jEM25JrQvjD3XjsvXuNHSWhYUsu" {
		t.Fatal("unexpected deposit address")
	}

	wdEnabled, err := m.IsWithdrawAndDepositsEnabled(maid)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !wdEnabled {
		t.Fatal("unexpected results")
	}

	tEnabled, err := m.IsTradingEnabledForCurrency(maid)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !tEnabled {
		t.Fatal("unexpected results")
	}

	cp := currency.NewPair(currency.USDT, currency.BTC)

	tEnabled, err = m.IsTradingEnabledForPair(cp)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !tEnabled {
		t.Fatal("unexpected results")
	}

	postOnly, err := m.IsPostOnlyForPair(cp)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if postOnly {
		t.Fatal("unexpected results")
	}

	_, err = m.GetWithdrawalTXFee(currency.Code{})
	if !errors.Is(err, errCurrencyNotFoundInMap) {
		t.Fatalf("expected: %v but received: %v", errCurrencyNotFoundInMap, err)
	}

	_, err = m.GetDepositAddress(currency.Code{})
	if !errors.Is(err, errCurrencyNotFoundInMap) {
		t.Fatalf("expected: %v but received: %v", errCurrencyNotFoundInMap, err)
	}

	_, err = m.IsWithdrawAndDepositsEnabled(currency.Code{})
	if !errors.Is(err, errCurrencyNotFoundInMap) {
		t.Fatalf("expected: %v but received: %v", errCurrencyNotFoundInMap, err)
	}

	_, err = m.IsTradingEnabledForCurrency(currency.Code{})
	if !errors.Is(err, errCurrencyNotFoundInMap) {
		t.Fatalf("expected: %v but received: %v", errCurrencyNotFoundInMap, err)
	}

	_, err = m.IsTradingEnabledForPair(currency.Pair{})
	if !errors.Is(err, errCurrencyNotFoundInMap) {
		t.Fatalf("expected: %v but received: %v", errCurrencyNotFoundInMap, err)
	}

	_, err = m.IsPostOnlyForPair(currency.Pair{})
	if !errors.Is(err, errCurrencyNotFoundInMap) {
		t.Fatalf("expected: %v but received: %v", errCurrencyNotFoundInMap, err)
	}
}
