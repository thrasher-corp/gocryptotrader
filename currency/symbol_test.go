package currency

import "testing"

func TestGetSymbolByCurrencyName(t *testing.T) {
	expected := "₩"
	actual, err := GetSymbolByCurrencyName(KPW)
	if err != nil {
		t.Errorf("TestGetSymbolByCurrencyName error: %s", err)
	}

	if actual != expected {
		t.Errorf("TestGetSymbolByCurrencyName differing values")
	}

	_, err = GetSymbolByCurrencyName(Code{})
	if err == nil {
		t.Errorf("TestGetSymbolByCurrencyNam returned nil on non-existent currency")
	}
}
