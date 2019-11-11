package currency

import "testing"

func TestGetSymbolByCurrencyName(t *testing.T) {
	expected := "₩"
	actual, err := GetSymbolByCurrencyName(KPW)
	if err != nil {
		t.Errorf("Test failed. TestGetSymbolByCurrencyName error: %s", err)
	}

	if actual != expected {
		t.Errorf("Test failed. TestGetSymbolByCurrencyName differing values")
	}

	_, err = GetSymbolByCurrencyName(Code{})
	if err == nil {
		t.Errorf("Test failed. TestGetSymbolByCurrencyNam returned nil on non-existent currency")
	}
}
