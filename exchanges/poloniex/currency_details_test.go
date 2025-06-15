package poloniex

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestWsCurrencyMap(t *testing.T) {
	var m CurrencyDetails

	if !m.isInitial() {
		t.Fatal("unexpected value")
	}

	err := m.loadPairs(nil)
	require.ErrorIs(t, err, errCannotLoadNoData)

	err = m.loadCodes(nil)
	require.ErrorIs(t, err, errCannotLoadNoData)

	_, err = m.GetPair(1337)
	require.ErrorIs(t, err, errPairMapIsNil)

	_, err = m.GetCode(1337)
	require.ErrorIs(t, err, errCodeMapIsNil)

	_, err = m.GetWithdrawalTXFee(currency.EMPTYCODE)
	require.ErrorIs(t, err, errCodeMapIsNil)

	_, err = m.GetDepositAddress(currency.EMPTYCODE)
	require.ErrorIs(t, err, errCodeMapIsNil)

	_, err = m.IsWithdrawAndDepositsEnabled(currency.EMPTYCODE)
	require.ErrorIs(t, err, errCodeMapIsNil)

	_, err = m.IsTradingEnabledForCurrency(currency.EMPTYCODE)
	require.ErrorIs(t, err, errCodeMapIsNil)

	_, err = m.IsTradingEnabledForPair(currency.EMPTYPAIR)
	require.ErrorIs(t, err, errCodeMapIsNil)

	_, err = m.IsPostOnlyForPair(currency.EMPTYPAIR)
	require.ErrorIs(t, err, errCodeMapIsNil)

	c, err := p.GetCurrencies(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	err = m.loadCodes(c)
	if err != nil {
		t.Fatal(err)
	}

	tick, err := p.GetTicker(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	err = m.loadPairs(tick)
	if err != nil {
		t.Fatal(err)
	}

	pTest, err := m.GetPair(1337)
	require.ErrorIs(t, err, errIDNotFoundInPairMap)

	if pTest.String() != "1337" {
		t.Fatal("unexpected value")
	}

	_, err = m.GetCode(1337)
	require.ErrorIs(t, err, errIDNotFoundInCodeMap)

	btcusdt, err := m.GetPair(121)
	require.NoError(t, err)

	if btcusdt.String() != "USDT_BTC" {
		t.Fatal("expecting USDT_BTC pair")
	}

	eth, err := m.GetCode(267)
	require.NoError(t, err)

	if eth.String() != "ETH" {
		t.Fatal("unexpected value")
	}

	txFee, err := m.GetWithdrawalTXFee(eth)
	if err != nil {
		t.Fatal(err)
	}

	if txFee == 0 {
		t.Fatal("unexpected value")
	}

	_, err = m.GetDepositAddress(eth)
	require.ErrorIs(t, err, errNoDepositAddress)

	dAddr, err := m.GetDepositAddress(currency.NewCode("BCN"))
	require.NoError(t, err)

	if dAddr != "25cZNQYVAi3issDCoa6fWA2Aogd4FgPhYdpX3p8KLfhKC6sN8s6Q9WpcW4778TPwcUS5jEM25JrQvjD3XjsvXuNHSWhYUsu" {
		t.Fatal("unexpected deposit address")
	}

	wdEnabled, err := m.IsWithdrawAndDepositsEnabled(eth)
	require.NoError(t, err)

	if !wdEnabled {
		t.Fatal("unexpected results")
	}

	tEnabled, err := m.IsTradingEnabledForCurrency(eth)
	require.NoError(t, err)

	if !tEnabled {
		t.Fatal("unexpected results")
	}

	cp := currency.NewPair(currency.USDT, currency.BTC)

	tEnabled, err = m.IsTradingEnabledForPair(cp)
	require.NoError(t, err)

	if !tEnabled {
		t.Fatal("unexpected results")
	}

	postOnly, err := m.IsPostOnlyForPair(cp)
	require.NoError(t, err)

	if postOnly {
		t.Fatal("unexpected results")
	}

	_, err = m.GetWithdrawalTXFee(currency.EMPTYCODE)
	require.ErrorIs(t, err, errCurrencyNotFoundInMap)

	_, err = m.GetDepositAddress(currency.EMPTYCODE)
	require.ErrorIs(t, err, errCurrencyNotFoundInMap)

	_, err = m.IsWithdrawAndDepositsEnabled(currency.EMPTYCODE)
	require.ErrorIs(t, err, errCurrencyNotFoundInMap)

	_, err = m.IsTradingEnabledForCurrency(currency.EMPTYCODE)
	require.ErrorIs(t, err, errCurrencyNotFoundInMap)

	_, err = m.IsTradingEnabledForPair(currency.EMPTYPAIR)
	require.ErrorIs(t, err, errCurrencyNotFoundInMap)

	_, err = m.IsPostOnlyForPair(currency.EMPTYPAIR)
	require.ErrorIs(t, err, errCurrencyNotFoundInMap)
}
