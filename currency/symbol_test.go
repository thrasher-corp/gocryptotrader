package currency

import "testing"

func TestGetSymbolByCurrencyName(t *testing.T) {
	actual, err := GetSymbolByCurrencyName(KPW)
	if err != nil {
		t.Errorf("TestGetSymbolByCurrencyName error: %s", err)
	}

	if expected := "₩"; actual != expected {
		t.Errorf("TestGetSymbolByCurrencyName differing values")
	}

	_, err = GetSymbolByCurrencyName(EMPTYCODE)
	if err == nil {
		t.Errorf("TestGetSymbolByCurrencyNam returned nil on non-existent currency")
	}
}
