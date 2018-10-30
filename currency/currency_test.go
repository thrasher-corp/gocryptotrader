package currency

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

func TestSetDefaults(t *testing.T) {
	FXRates = nil
	BaseCurrency = "BLAH"
	FXProviders = nil

	SetDefaults()

	if FXRates == nil {
		t.Fatal("Expected FXRates to be non-nil")
	}

	if BaseCurrency != DefaultBaseCurrency {
		t.Fatal("Expected BaseCurrency to be 'USD'")
	}

	if FXProviders == nil {
		t.Fatal("Expected FXRates to be non-nil")
	}
}

func TestSeedCurrencyData(t *testing.T) {
	err := Seed("AUD")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetExchangeRates(t *testing.T) {
	result := GetExchangeRates()
	backup := FXRates

	FXRates = nil
	result = GetExchangeRates()
	if result != nil {
		t.Fatal("Expected nil map")
	}

	FXRates = backup
}

func TestIsDefaultCurrency(t *testing.T) {
	t.Parallel()

	var str1, str2, str3 string = "USD", "usd", "cats123"

	if !IsDefaultCurrency(str1) {
		t.Errorf(
			"Test Failed. TestIsDefaultCurrency: \nCannot match currency, %s.", str1,
		)
	}
	if !IsDefaultCurrency(str2) {
		t.Errorf(
			"Test Failed. TestIsDefaultCurrency: \nCannot match currency, %s.", str2,
		)
	}
	if IsDefaultCurrency(str3) {
		t.Errorf(
			"Test Failed. TestIsDefaultCurrency: \nFunction return is incorrect with, %s.",
			str3,
		)
	}
}

func TestIsDefaultCryptocurrency(t *testing.T) {
	t.Parallel()

	var str1, str2, str3 string = "BTC", "btc", "dogs123"

	if !IsDefaultCryptocurrency(str1) {
		t.Errorf(
			"Test Failed. TestIsDefaultCryptocurrency: \nCannot match currency, %s.",
			str1,
		)
	}
	if !IsDefaultCryptocurrency(str2) {
		t.Errorf(
			"Test Failed. TestIsDefaultCryptocurrency: \nCannot match currency, %s.",
			str2,
		)
	}
	if IsDefaultCryptocurrency(str3) {
		t.Errorf(
			"Test Failed. TestIsDefaultCryptocurrency: \nFunction return is incorrect with, %s.",
			str3,
		)
	}
}

func TestIsFiatCurrency(t *testing.T) {
	if IsFiatCurrency("") {
		t.Error("Test failed. TestIsFiatCurrency returned true on an empty string")
	}

	FiatCurrencies = []string{"USD", "AUD"}
	var str1, str2, str3 string = "BTC", "USD", "birds123"

	if IsFiatCurrency(str1) {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency: \nCannot match currency, %s.", str1,
		)
	}
	if !IsFiatCurrency(str2) {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency: \nCannot match currency, %s.", str2,
		)
	}
	if IsFiatCurrency(str3) {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency: \nCannot match currency, %s.", str3,
		)
	}
}

func TestIsCryptocurrency(t *testing.T) {
	if IsCryptocurrency("") {
		t.Error("Test failed. TestIsCryptocurrency returned true on an empty string")
	}

	CryptoCurrencies = []string{"BTC", "LTC", "DASH"}
	var str1, str2, str3 string = "USD", "BTC", "pterodactyl123"

	if IsCryptocurrency(str1) {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency: \nCannot match currency, %s.", str1,
		)
	}
	if !IsCryptocurrency(str2) {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency: \nCannot match currency, %s.", str2,
		)
	}
	if IsCryptocurrency(str3) {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency: \nCannot match currency, %s.", str3,
		)
	}
}

func TestIsCryptoPair(t *testing.T) {
	if IsCryptocurrency("") {
		t.Error("Test failed. TestIsCryptocurrency returned true on an empty string")
	}

	CryptoCurrencies = []string{"BTC", "LTC", "DASH"}
	FiatCurrencies = []string{"USD"}

	if !IsCryptoPair(pair.NewCurrencyPair("BTC", "LTC")) {
		t.Error("Test Failed. TestIsCryptoPair. Expected true result")
	}

	if IsCryptoPair(pair.NewCurrencyPair("BTC", "USD")) {
		t.Error("Test Failed. TestIsCryptoPair. Expected false result")
	}
}

func TestIsCryptoFiatPair(t *testing.T) {
	if IsCryptocurrency("") {
		t.Error("Test failed. TestIsCryptocurrency returned true on an empty string")
	}

	CryptoCurrencies = []string{"BTC", "LTC", "DASH"}
	FiatCurrencies = []string{"USD"}

	if !IsCryptoFiatPair(pair.NewCurrencyPair("BTC", "USD")) {
		t.Error("Test Failed. TestIsCryptoPair. Expected true result")
	}

	if IsCryptoFiatPair(pair.NewCurrencyPair("BTC", "LTC")) {
		t.Error("Test Failed. TestIsCryptoPair. Expected false result")
	}
}

func TestIsFiatPair(t *testing.T) {
	CryptoCurrencies = []string{"BTC", "LTC", "DASH"}
	FiatCurrencies = []string{"USD", "AUD", "EUR"}

	if !IsFiatPair(pair.NewCurrencyPair("AUD", "USD")) {
		t.Error("Test Failed. TestIsFiatPair. Expected true result")
	}

	if IsFiatPair(pair.NewCurrencyPair("BTC", "AUD")) {
		t.Error("Test Failed. TestIsFiatPair. Expected false result")
	}
}

func TestUpdate(t *testing.T) {
	CryptoCurrencies = []string{"BTC", "LTC", "DASH"}
	FiatCurrencies = []string{"USD", "AUD"}

	Update([]string{"ETH"}, true)
	Update([]string{"JPY"}, false)

	if !IsCryptocurrency("ETH") {
		t.Error(
			"Test Failed. TestUpdate: \nCannot match currency: ETH",
		)
	}

	if !IsFiatCurrency("JPY") {
		t.Errorf(
			"Test Failed. TestUpdate: \nCannot match currency: JPY",
		)
	}
}

func TestExtractBaseCurrency(t *testing.T) {
	backup := FXRates
	FXRates = nil
	FXRates = make(map[string]float64)

	if extractBaseCurrency() != "" {
		t.Fatalf("Test failed. Expected '' as base currency")
	}

	FXRates["USDAUD"] = 120

	if extractBaseCurrency() != "USD" {
		t.Fatalf("Test failed. Expected 'USD' as base currency")
	}
	FXRates = backup
}
func TestConvertCurrency(t *testing.T) {
	_, err := ConvertCurrency(100, "AUD", "USD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = ConvertCurrency(100, "USD", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = ConvertCurrency(100, "CNY", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = ConvertCurrency(100, "meow", "USD")
	if err == nil {
		t.Fatal("Expected err on non-existent currency")
	}

	_, err = ConvertCurrency(100, "USD", "meow")
	if err == nil {
		t.Fatal("Expected err on non-existent currency")
	}

}
