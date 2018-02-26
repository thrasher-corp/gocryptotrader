package currency

import (
	"reflect"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
)

func TestSetProvider(t *testing.T) {
	defaultVal := YahooEnabled
	expected := "yahoo"
	SetProvider(true)
	actual := GetProvider()
	if expected != actual {
		t.Errorf("Test failed. TestGetProvider expected %s got %s", expected, actual)
	}

	SetProvider(false)
	expected = "fixer"
	actual = GetProvider()
	if expected != actual {
		t.Errorf("Test failed. TestGetProvider expected %s got %s", expected, actual)
	}

	SetProvider(defaultVal)
}

func TestSwapProvider(t *testing.T) {
	defaultVal := YahooEnabled
	expected := "fixer"
	SetProvider(true)
	SwapProvider()
	actual := GetProvider()
	if expected != actual {
		t.Errorf("Test failed. TestGetProvider expected %s got %s", expected, actual)
	}

	SetProvider(false)
	SwapProvider()
	expected = "yahoo"
	actual = GetProvider()
	if expected != actual {
		t.Errorf("Test failed. TestGetProvider expected %s got %s", expected, actual)
	}

	SetProvider(defaultVal)
}

func TestGetProvider(t *testing.T) {
	defaultVal := YahooEnabled
	SetProvider(true)
	expected := "yahoo"
	actual := GetProvider()
	if expected != actual {
		t.Errorf("Test failed. TestGetProvider expected %s got %s", expected, actual)
	}

	SetProvider(false)
	expected = "fixer"
	actual = GetProvider()
	if expected != actual {
		t.Errorf("Test failed. TestGetProvider expected %s got %s", expected, actual)
	}

	SetProvider(defaultVal)
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

	BaseCurrencies = []string{"USD", "AUD"}
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
	BaseCurrencies = []string{"USD"}

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
	BaseCurrencies = []string{"USD"}

	if !IsCryptoFiatPair(pair.NewCurrencyPair("BTC", "USD")) {
		t.Error("Test Failed. TestIsCryptoPair. Expected true result")
	}

	if IsCryptoFiatPair(pair.NewCurrencyPair("BTC", "LTC")) {
		t.Error("Test Failed. TestIsCryptoPair. Expected false result")
	}
}

func TestIsFiatPair(t *testing.T) {
	CryptoCurrencies = []string{"BTC", "LTC", "DASH"}
	BaseCurrencies = []string{"USD", "AUD", "EUR"}

	if !IsFiatPair(pair.NewCurrencyPair("AUD", "USD")) {
		t.Error("Test Failed. TestIsFiatPair. Expected true result")
	}

	if IsFiatPair(pair.NewCurrencyPair("BTC", "AUD")) {
		t.Error("Test Failed. TestIsFiatPair. Expected false result")
	}
}

func TestUpdate(t *testing.T) {
	CryptoCurrencies = []string{"BTC", "LTC", "DASH"}
	BaseCurrencies = []string{"USD", "AUD"}

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

func TestSeedCurrencyData(t *testing.T) {
	//	SetProvider(true)
	if YahooEnabled {
		currencyRequestDefault := ""
		currencyRequestUSDAUD := "USD,AUD"
		currencyRequestObtuse := "WigWham"

		err := SeedCurrencyData(currencyRequestDefault)
		if err != nil {
			t.Errorf(
				"Test Failed. SeedCurrencyData: Error %s with currency as %s.",
				err, currencyRequestDefault,
			)
		}
		err2 := SeedCurrencyData(currencyRequestUSDAUD)
		if err2 != nil {
			t.Errorf(
				"Test Failed. SeedCurrencyData: Error %s with currency as %s.",
				err2, currencyRequestUSDAUD,
			)
		}
		err3 := SeedCurrencyData(currencyRequestObtuse)
		if err3 == nil {
			t.Errorf(
				"Test Failed. SeedCurrencyData: Error %s with currency as %s.",
				err3, currencyRequestObtuse,
			)
		}
	}

	//SetProvider(false)
	err := SeedCurrencyData("")
	if err != nil {
		t.Errorf("Test failed. SeedCurrencyData via Fixer. Error: %s", err)
	}
}

func TestMakecurrencyPairs(t *testing.T) {
	t.Parallel()

	lengthDefault := len(common.SplitStrings(DefaultCurrencies, ","))
	fiatPairsLength := len(
		common.SplitStrings(MakecurrencyPairs(DefaultCurrencies), ","),
	)

	if lengthDefault*(lengthDefault-1) > fiatPairsLength {
		t.Error("Test Failed. MakecurrencyPairs: Error, mismatched length")
	}
}

func TestConvertCurrency(t *testing.T) {
	//	SetProvider(true)
	if YahooEnabled {
		fiatCurrencies := DefaultCurrencies
		for _, currencyFrom := range common.SplitStrings(fiatCurrencies, ",") {
			for _, currencyTo := range common.SplitStrings(fiatCurrencies, ",") {
				floatyMcfloat, err := ConvertCurrency(1000, currencyFrom, currencyTo)
				if err != nil {
					t.Errorf(
						"Test Failed. ConvertCurrency: Error %s with return: %.2f Currency 1: %s Currency 2: %s",
						err, floatyMcfloat, currencyFrom, currencyTo,
					)
				}
				if reflect.TypeOf(floatyMcfloat).String() != "float64" {
					t.Error("Test Failed. ConvertCurrency: Error, incorrect return type")
				}
				if floatyMcfloat <= 0 {
					t.Error(
						"Test Failed. ConvertCurrency: Error, negative return or a serious issue with current fiat",
					)
				}
			}
		}
	}

	//	SetProvider(false)
	_, err := ConvertCurrency(1000, "USD", "AUD")
	if err != nil {
		t.Errorf("Test failed. ConvertCurrency USD -> AUD. Error %s", err)
	}

	_, err = ConvertCurrency(1000, "AUD", "USD")
	if err != nil {
		t.Errorf("Test failed. ConvertCurrency AUD -> AUD. Error %s", err)
	}

	_, err = ConvertCurrency(1000, "CNY", "AUD")
	if err != nil {
		t.Errorf("Test failed. ConvertCurrency USD -> AUD. Error %s", err)
	}

	// Test non-existent currencies

	_, err = ConvertCurrency(1000, "ASDF", "USD")
	if err == nil {
		t.Errorf("Test failed. ConvertCurrency non-existent currency -> USD. Error %s", err)
	}

	_, err = ConvertCurrency(1000, "USD", "ASDF")
	if err == nil {
		t.Errorf("Test failed. ConvertCurrency USD -> non-existent currency. Error %s", err)
	}

	_, err = ConvertCurrency(1000, "CNY", "UAHF")
	if err == nil {
		t.Errorf("Test failed. ConvertCurrency non-USD currency CNY -> non-existent currency. Error %s", err)
	}

	_, err = ConvertCurrency(1000, "UASF", "UAHF")
	if err == nil {
		t.Errorf("Test failed. ConvertCurrency non-existent currency -> non-existent currency. Error %s", err)
	}
}

func TestFetchFixerCurrencyData(t *testing.T) {
	err := FetchFixerCurrencyData()
	if err != nil {
		t.Errorf("Test failed. FetchFixerCurrencyData returned %s", err)
	}
}

func TestFetchYahooCurrencyData(t *testing.T) {
	if !YahooEnabled {
		t.Skip()
	}

	t.Parallel()
	var fetchData []string
	fiatCurrencies := DefaultCurrencies

	for _, currencyOne := range common.SplitStrings(fiatCurrencies, ",") {
		for _, currencyTwo := range common.SplitStrings(fiatCurrencies, ",") {
			if currencyOne == currencyTwo {
				continue
			} else {
				fetchData = append(fetchData, currencyOne+currencyTwo)
			}
		}
	}
	err := FetchYahooCurrencyData(fetchData)
	if err != nil {
		t.Errorf("Test Failed. FetchYahooCurrencyData: Error %s", err)
	}
}

func TestQueryYahooCurrencyValues(t *testing.T) {
	if !YahooEnabled {
		t.Skip()
	}

	err := QueryYahooCurrencyValues(DefaultCurrencies)
	if err != nil {
		t.Errorf("Test Failed. QueryYahooCurrencyValues: Error, %s", err)
	}

	err = QueryYahooCurrencyValues(DefaultCryptoCurrencies)
	if err == nil {
		t.Errorf("Test Failed. QueryYahooCurrencyValues: Error, %s", err)
	}
}
