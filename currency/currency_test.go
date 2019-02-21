package currency

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

func TestCodeString(t *testing.T) {
	expected := "TEST"
	cc := NewCurrencyCode("TEST")
	if cc.String() != expected {
		t.Errorf("Test Failed - Currency Code String() error expected %s but recieved %s",
			expected, cc)
	}
}

func TestNewConversionFromString(t *testing.T) {
	expected := "AUDUSD"
	conv := NewConversionFromString(expected)
	if conv.String() != expected {
		t.Errorf("Test Failed - NewConversion() error expected %s but received %s",
			expected,
			conv)
	}

	newexpected := common.StringToLower(expected)
	conv = NewConversionFromString(newexpected)
	if conv.String() != newexpected {
		t.Errorf("Test Failed - NewConversion() error expected %s but received %s",
			newexpected,
			conv)
	}
}

func TestNewConversion(t *testing.T) {
	from := "AUD"
	to := "USD"
	expected := "AUDUSD"

	conv := NewConversion(from, to)
	if conv.String() != expected {
		t.Errorf("Test Failed - NewConversion() error expected %s but received %s",
			expected,
			conv)
	}
}

func TestNewConversionFromCode(t *testing.T) {
	from := NewCurrencyCode("AUD")
	to := NewCurrencyCode("USD")
	expected := "AUDUSD"

	conv, err := NewConversionFromCode(from, to)
	if err != nil {
		t.Error("Test Failed - NewConversionFromCode() error", err)
	}

	if conv.String() != expected {
		t.Errorf("Test Failed - NewConversion() error expected %s but received %s",
			expected,
			conv)
	}
}

func TestGetDefaultExchangeRates(t *testing.T) {
	// rates, err := GetDefaultExchangeRates()
	// if err != nil {
	// 	t.Error("Test failed - GetDefaultExchangeRates() err", err)
	// }

	// for key := range rates {
	// 	if !key.IsFiatPair() {
	// 		t.Errorf("Test failed - GetDefaultExchangeRates() %s is not fiat pair",
	// 			key)
	// 	}
	// }
}

func TestGetSystemExchangeRates(t *testing.T) {
	// rates, err := GetSystemExchangeRates()
	// if err != nil {
	// 	t.Error("Test failed - GetDefaultExchangeRates() err", err)
	// }

	// for key := range rates {
	// 	if !key.IsFiatPair() {
	// 		t.Errorf("Test failed - GetDefaultExchangeRates() %s is not fiat pair",
	// 			key)
	// 	}
	// }
}

func TestUpdateBaseCurrency(t *testing.T) {
	err := UpdateBaseCurrency(AUD)
	if err != nil {
		t.Error("Test failed - UpdateBaseCurrency() err", err)
	}

	err = UpdateBaseCurrency(LTC)
	if err == nil {
		t.Error("Test failed - UpdateBaseCurrency() cannot be nil")
	}

	if GetBaseCurrency() != AUD {
		t.Errorf("Test failed - GetBaseCurrency() expected %s but recieved %s",
			AUD, GetBaseCurrency())
	}
}

func TestGetDefaultBaseCurrency(t *testing.T) {
	if GetDefaultBaseCurrency() != USD {
		t.Errorf("Test failed - GetDefaultBaseCurrency() expected %s but recieved %s",
			USD, GetDefaultBaseCurrency())
	}
}

func TestGetDefaulCryptoCurrencies(t *testing.T) {
	expected := Currencies{BTC, LTC, ETH, DOGE, DASH, XRP, XMR}
	if !GetDefaultCryptocurrencies().Match(expected) {
		t.Errorf("Test failed - GetDefaultCryptocurrencies() expected %s but recieved %s",
			expected, GetDefaultCryptocurrencies())
	}
}

func TestGetDefaultFiatCurrencies(t *testing.T) {
	expected := Currencies{USD, AUD, EUR, CNY}
	if !GetDefaultFiatCurrencies().Match(expected) {
		t.Errorf("Test failed - GetDefaultFiatCurrencies() expected %s but recieved %s",
			expected, GetDefaultFiatCurrencies())
	}
}

func TestUpdateCurrencies(t *testing.T) {
	fiat := Currencies{HKN, JPY}
	UpdateCurrencies(fiat, false)
	rFiat := GetSystemFiatCurrencies()
	if !rFiat.Contains(HKN) || !rFiat.Contains(JPY) {
		t.Error("Test failed - UpdateCurrencies() currencies did not update")
	}

	crypto := Currencies{ZAR, ZCAD, B2}
	UpdateCurrencies(crypto, true)
	rCrypto := GetSystemCryptoCurrencies()
	if !rCrypto.Contains(ZAR) || !rCrypto.Contains(ZAR) || !rCrypto.Contains(B2) {
		t.Error("Test failed - UpdateCurrencies() currencies did not update")
	}
}

func TestConvertCurrency(t *testing.T) {
	_, err := ConvertCurrency(100, AUD, USD)
	if err != nil {
		t.Fatal(err)
	}

	r, err := ConvertCurrency(100, AUD, AUD)
	if err != nil {
		t.Fatal(err)
	}

	if r != 100 {
		t.Errorf("Test Failed - ConvertCurrency error, incorrect rate return %2.f but received %2.f",
			100.00, r)
	}

	_, err = ConvertCurrency(100, USD, AUD)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ConvertCurrency(100, CNY, AUD)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ConvertCurrency(100, LTC, USD)
	if err == nil {
		t.Fatal("Expected err on non-existent currency")
	}

	_, err = ConvertCurrency(100, USD, LTC)
	if err == nil {
		t.Fatal("Expected err on non-existent currency")
	}
}
