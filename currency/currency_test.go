package currency

import (
	"reflect"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

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
	t.Parallel()

	if IsFiatCurrency("") {
		t.Error("Test failed. TestIsFiatCurrency returned true on an empty string")
	}

	BaseCurrencies = "USD,AUD"
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
	t.Parallel()

	if IsCryptocurrency("") {
		t.Error("Test failed. TestIsCryptocurrency returned true on an empty string")
	}

	CryptoCurrencies = "BTC,LTC,DASH"
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

func TestContainsSeparator(t *testing.T) {
	t.Parallel()

	var str1, str2, str3, str4 string = "ding-dong", "ding_dong", "dong_ding-dang", "ding"

	doesIt, whatIsIt := ContainsSeparator(str1)
	if doesIt != true || whatIsIt != "-" {
		t.Errorf(
			"Test Failed. ContainsSeparator: \nCannot find separator, %s.", str1,
		)
	}
	doesIt2, whatIsIt2 := ContainsSeparator(str2)
	if doesIt2 != true || whatIsIt2 != "_" {
		t.Errorf(
			"Test Failed. ContainsSeparator: \nCannot find separator, %s.", str2,
		)
	}
	doesIt3, whatIsIt3 := ContainsSeparator(str3)
	if doesIt3 != true || len(whatIsIt3) != 3 {
		t.Errorf(
			"Test Failed. ContainsSeparator: \nCannot find or incorrect separator, %s.",
			str3,
		)
	}
	doesIt4, whatIsIt4 := ContainsSeparator(str4)
	if doesIt4 != false || whatIsIt4 != "" {
		t.Errorf(
			"Test Failed. ContainsSeparator: \nReturn Issues with string, %s.", str3,
		)
	}
}

func TestContainsBaseCurrencyIndex(t *testing.T) {
	t.Parallel()

	baseCurrencies := []string{"USD", "AUD", "EUR", "CNY"}
	currency1, currency2 := "USD", "DINGDONG"

	isIt, whatIsIt := ContainsBaseCurrencyIndex(baseCurrencies, currency1)
	if !isIt && whatIsIt != "USD" {
		t.Errorf(
			"Test Failed. ContainsBaseCurrencyIndex: \nReturned: %t & %s, with Currency as %s.",
			isIt, whatIsIt, currency1,
		)
	}
	isIt2, whatIsIt2 := ContainsBaseCurrencyIndex(baseCurrencies, currency2)
	if isIt2 && whatIsIt2 != "DINGDONG" {
		t.Errorf(
			"Test Failed. ContainsBaseCurrencyIndex: \nReturned: %t & %s, with Currency as %s.",
			isIt2, whatIsIt2, currency2,
		)
	}
}

func TestContainsBaseCurrency(t *testing.T) {
	t.Parallel()

	baseCurrencies := []string{"USD", "AUD", "EUR", "CNY"}
	currency1, currency2 := "USD", "DINGDONG"

	isIt := ContainsBaseCurrency(baseCurrencies, currency1)
	if !isIt {
		t.Errorf("Test Failed. ContainsBaseCurrency: \nReturned: %t, with Currency as %s.",
			isIt, currency1,
		)
	}
	isIt2 := ContainsBaseCurrency(baseCurrencies, currency2)
	if isIt2 {
		t.Errorf("Test Failed. ContainsBaseCurrency: \nReturned: %t, with Currency as %s.",
			isIt2, currency2,
		)
	}
}

func TestCheckAndAddCurrency(t *testing.T) {
	t.Parallel()

	inputFiat := []string{"USD", "AUD", "EUR"}
	inputCrypto := []string{"BTC", "LTC", "ETH", "DOGE", "DASH", "XRP"}
	testError := []string{"Testy"}
	fiat := "USD"
	fiatIncrease := "CNY"
	crypto := "LTC"
	cryptoIncrease := "XMR"
	obtuse := "CATSANDDOGS"

	appendedString := CheckAndAddCurrency(inputFiat, fiat)
	if len(appendedString) > len(inputFiat) {
		t.Errorf(
			"Test Failed. CheckAndAddCurrency: Error with inputFiat, currency as %s.",
			fiat,
		)
	}
	appendedString = CheckAndAddCurrency(inputFiat, fiatIncrease)
	if len(appendedString) <= len(inputFiat) {
		t.Errorf(
			"Test Failed. CheckAndAddCurrency: Error with inputFiat, currency as %s.",
			fiatIncrease,
		)
	}
	appendedString = CheckAndAddCurrency(inputFiat, crypto)
	if len(appendedString) > len(inputFiat) {
		t.Log(appendedString)
		t.Errorf(
			"Test Failed. CheckAndAddCurrency: Error with inputFiat, currency as %s.",
			crypto,
		)
	}
	appendedString = CheckAndAddCurrency(inputFiat, obtuse)
	if len(appendedString) > len(inputFiat) {
		t.Errorf(
			"Test Failed. CheckAndAddCurrency: Error with inputFiat, currency as %s.",
			obtuse,
		)
	}

	appendedString = CheckAndAddCurrency(inputCrypto, crypto)
	if len(appendedString) > len(inputCrypto) {
		t.Errorf(
			"Test Failed. CheckAndAddCurrency: Error with inputCrytpo, currency as %s.",
			crypto,
		)
	}
	appendedString = CheckAndAddCurrency(inputCrypto, cryptoIncrease)
	if len(appendedString) <= len(inputCrypto) {
		t.Errorf(
			"Test Failed. CheckAndAddCurrency: Error with inputCrytpo, currency as %s.",
			cryptoIncrease,
		)
	}
	appendedString = CheckAndAddCurrency(inputCrypto, fiat)
	if len(appendedString) > len(inputCrypto) {
		t.Errorf(
			"Test Failed. CheckAndAddCurrency: Error with inputCrytpo, currency as %s.",
			fiat,
		)
	}
	appendedString = CheckAndAddCurrency(inputCrypto, obtuse)
	if len(appendedString) > len(inputCrypto) {
		t.Errorf(
			"Test Failed. CheckAndAddCurrency: Error with inputCrytpo, currency as %s.",
			obtuse,
		)
	}

	appendedString = CheckAndAddCurrency(testError, "USD")
	if appendedString[0] != testError[0] {
		t.Errorf(
			"Test Failed. CheckAndAddCurrency: Error with inputCrytpo, basecurrency as %s.",
			testError,
		)
	}
}

func TestSeedCurrencyData(t *testing.T) {
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

	YahooEnabled = false
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

	YahooEnabled = false
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

	// Test non-existant currencies

	_, err = ConvertCurrency(1000, "ASDF", "USD")
	if err == nil {
		t.Errorf("Test failed. ConvertCurrency non-existant currency -> USD. Error %s", err)
	}

	_, err = ConvertCurrency(1000, "USD", "ASDF")
	if err == nil {
		t.Errorf("Test failed. ConvertCurrency USD -> non-existant currency. Error %s", err)
	}

	_, err = ConvertCurrency(1000, "CNY", "UAHF")
	if err == nil {
		t.Errorf("Test failed. ConvertCurrency non-USD currency CNY -> non-existant currency. Error %s", err)
	}

	_, err = ConvertCurrency(1000, "UASF", "UAHF")
	if err == nil {
		t.Errorf("Test failed. ConvertCurrency non-existant currency -> non-existant currency. Error %s", err)
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
		return
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
		return
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
