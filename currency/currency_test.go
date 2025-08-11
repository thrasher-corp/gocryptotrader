package currency

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDefaultExchangeRates(t *testing.T) {
	rates, err := GetDefaultExchangeRates()
	if err != nil {
		t.Error("GetDefaultExchangeRates() err", err)
	}

	for _, val := range rates {
		if !val.IsFiat() {
			t.Errorf("GetDefaultExchangeRates() %s is not fiat pair",
				val)
		}
	}
}

func TestGetExchangeRates(t *testing.T) {
	rates, err := GetExchangeRates()
	if err != nil {
		t.Error("GetExchangeRates() err", err)
	}

	for _, val := range rates {
		if !val.IsFiat() {
			t.Errorf("GetExchangeRates() %s is not fiat pair",
				val)
		}
	}
}

func TestUpdateBaseCurrency(t *testing.T) {
	err := UpdateBaseCurrency(AUD)
	if err != nil {
		t.Error("UpdateBaseCurrency() err", err)
	}

	err = UpdateBaseCurrency(LTC)
	if err == nil {
		t.Error("UpdateBaseCurrency() error cannot be nil")
	}

	if !GetBaseCurrency().Equal(AUD) {
		t.Errorf("GetBaseCurrency() expected %s but received %s",
			AUD, GetBaseCurrency())
	}
}

func TestGetDefaultBaseCurrency(t *testing.T) {
	if !GetDefaultBaseCurrency().Equal(USD) {
		t.Errorf("GetDefaultBaseCurrency() expected %s but received %s",
			USD, GetDefaultBaseCurrency())
	}
}

func TestGetDefaulCryptoCurrencies(t *testing.T) {
	expected := Currencies{BTC, LTC, ETH, DOGE, DASH, XRP, XMR, USDT, UST}
	if !GetDefaultCryptocurrencies().Match(expected) {
		t.Errorf("GetDefaultCryptocurrencies() expected %s but received %s",
			expected, GetDefaultCryptocurrencies())
	}
}

func TestGetDefaultFiatCurrencies(t *testing.T) {
	expected := Currencies{BZD, KYD, LRD, SAR, MKD, SRD, BMD, KHR, COP, CRC, GIP, NIO, CHF, VEF, ILS, BSD, CUP, HKD, IDR, SYP, AWG, TTD, DOP, JPY, PAB, SHP, BGN, JEP, AZN, JMD, MXN, CAD, GGP, RUR, GBP, GTQ, LBP, THB, MZN, RSD, ARS, BYN, HRK, GHS, MUR, ANG, QAR, ZWD, CLP, INR, IRR, NOK, PHP, LKR, TRY, BAM, EGP, TVD, SVC, FJD, PEN, RUB, SOS, XCD, KZT, BWP, ISK, KPW, KRW, PKR, UYU, BND, MNT, SEK, UAH, BBD, GYD, NZD, SCR, ZAR, FKP, HUF, RON, AFN, PLN, OMR, USD, CZK, YER, AUD, EUR, TWD, BRL, DKK, KGS, PYG, SBD, UZS, IMP, MYR, NAD, NPR, LAK, VND, ALL, BOB, HNL, SGD, CNY, NGN}
	if !GetDefaultFiatCurrencies().Match(expected) {
		t.Errorf("GetDefaultFiatCurrencies() expected %s but received %s",
			expected, GetDefaultFiatCurrencies())
	}
}

func TestUpdateCurrencies(t *testing.T) {
	fiat := Currencies{HKN, JPY}
	UpdateCurrencies(fiat, false)
	rFiat := GetFiatCurrencies()
	if !rFiat.Contains(HKN) || !rFiat.Contains(JPY) {
		t.Error("UpdateCurrencies() currencies did not update")
	}

	crypto := Currencies{ZAR, ZCAD, B2}
	UpdateCurrencies(crypto, true)
	rCrypto := GetCryptocurrencies()
	if !rCrypto.Contains(ZAR) || !rCrypto.Contains(ZCAD) || !rCrypto.Contains(B2) {
		t.Error("UpdateCurrencies() currencies did not update")
	}
}

func TestConvertFiat(t *testing.T) {
	_, err := ConvertFiat(0, LTC, USD)
	require.ErrorIs(t, err, errInvalidAmount)

	_, err = ConvertFiat(100, LTC, USD)
	require.ErrorIs(t, err, errNotFiatCurrency)

	_, err = ConvertFiat(100, USD, LTC)
	require.ErrorIs(t, err, errNotFiatCurrency)

	_, err = ConvertFiat(100, AUD, USD)
	if err != nil {
		t.Fatal(err)
	}

	r, err := ConvertFiat(100, AUD, AUD)
	if err != nil {
		t.Fatal(err)
	}

	if r != 100 {
		t.Errorf("ConvertCurrency error, incorrect rate return %2.f but received %2.f",
			100.00, r)
	}

	_, err = ConvertFiat(100, USD, AUD)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ConvertFiat(100, CNY, AUD)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetForeignExchangeRate(t *testing.T) {
	_, err := GetForeignExchangeRate(NewPair(EMPTYCODE, EMPTYCODE))
	require.ErrorIs(t, err, errNotFiatCurrency)

	_, err = GetForeignExchangeRate(NewPair(USD, EMPTYCODE))
	require.ErrorIs(t, err, errNotFiatCurrency)

	one, err := GetForeignExchangeRate(NewPair(USD, USD))
	require.NoError(t, err)

	if one != 1 {
		t.Fatal("unexpected value")
	}

	rate, err := GetForeignExchangeRate(NewPair(AUD, USD))
	require.NoError(t, err)

	if rate <= 0 {
		t.Fatal("unexpected value")
	}
}

func TestAllFXSettingsIsEnabled(t *testing.T) {
	var settings AllFXSettings
	if received := settings.IsEnabled("wow"); received {
		t.Fatalf("received: '%v' but expected: '%v'", received, false)
	}

	settings = []FXSettings{
		{
			Name: "wOow",
		},
		{
			Name:    "amICool?",
			Enabled: true,
		},
	}

	if received := settings.IsEnabled("AMICOOL?"); !received {
		t.Fatalf("received: '%v' but expected: '%v'", received, true)
	}
}
