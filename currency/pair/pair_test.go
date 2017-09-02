package pair

import "testing"

func TestLower(t *testing.T) {
	t.Parallel()
	pair := CurrencyItem("BTCUSD")
	actual := pair.Lower()
	expected := CurrencyItem("btcusd")
	if actual != expected {
		t.Errorf("Test failed. Lower(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestUpper(t *testing.T) {
	t.Parallel()
	pair := CurrencyItem("btcusd")
	actual := pair.Upper()
	expected := CurrencyItem("BTCUSD")
	if actual != expected {
		t.Errorf("Test failed. Upper(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := "BTCUSD"
	expected := pair.Pair().String()
	if actual != expected {
		t.Errorf("Test failed. String(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestGetFirstCurrency(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.GetFirstCurrency()
	expected := CurrencyItem("BTC")
	if actual != expected {
		t.Errorf(
			"Test failed. GetFirstCurrency(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestGetSecondCurrency(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.GetSecondCurrency()
	expected := CurrencyItem("USD")
	if actual != expected {
		t.Errorf(
			"Test failed. GetSecondCurrency(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestPair(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.Pair()
	expected := CurrencyItem("BTCUSD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestDisplay(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPairDelimiter("BTC-USD", "-")
	actual := pair.Pair()
	expected := CurrencyItem("BTC-USD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Display("", false)
	expected = CurrencyItem("btcusd")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Display("~", true)
	expected = CurrencyItem("BTC~USD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewCurrencyPair(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.Pair()
	expected := CurrencyItem("BTCUSD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewCurrencyPairDelimiter(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPairDelimiter("BTC-USD", "-")
	actual := pair.Pair()
	expected := CurrencyItem("BTC-USD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = CurrencyItem(pair.Delimiter)
	expected = "-"
	if actual != expected {
		t.Errorf(
			"Test failed. Delmiter: %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

// NewCurrencyPairFromIndex returns a CurrencyPair via a currency string and
// specific index
func TestNewCurrencyPairFromIndex(t *testing.T) {
	t.Parallel()
	currency := "BTCUSD"
	index := "BTC"

	pair := NewCurrencyPairFromIndex(currency, index)
	pair.Delimiter = "-"
	actual := pair.Pair()

	expected := CurrencyItem("BTC-USD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	currency = "DOGEBTC"

	pair = NewCurrencyPairFromIndex(currency, index)
	pair.Delimiter = "-"
	actual = pair.Pair()

	expected = CurrencyItem("DOGE-BTC")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewCurrencyPairFromString(t *testing.T) {
	t.Parallel()
	pairStr := "BTC-USD"
	pair := NewCurrencyPairFromString(pairStr)
	actual := pair.Pair()
	expected := CurrencyItem("BTC-USD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	pairStr = "BTCUSD"
	pair = NewCurrencyPairFromString(pairStr)
	actual = pair.Pair()
	expected = CurrencyItem("BTCUSD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}
