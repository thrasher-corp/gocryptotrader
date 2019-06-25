package currency

import (
	"testing"

	"github.com/idoall/gocryptotrader/common"
)

func TestNewConversionFromString(t *testing.T) {
	expected := "AUDUSD"
	conv, err := NewConversionFromString(expected)
	if err != nil {
		t.Error("Test Failed - NewConversionFromString() error", err)
	}
	if conv.String() != expected {
		t.Errorf("Test Failed - NewConversion() error expected %s but received %s",
			expected,
			conv)
	}

	newexpected := common.StringToLower(expected)
	conv, err = NewConversionFromString(newexpected)
	if err != nil {
		t.Error("Test Failed - NewConversionFromString() error", err)
	}
	if conv.String() != newexpected {
		t.Errorf("Test Failed - NewConversion() error expected %s but received %s",
			newexpected,
			conv)
	}
}

func TestNewConversionFromStrings(t *testing.T) {
	from := "AUD"
	to := "USD"
	expected := "AUDUSD"

	conv, err := NewConversionFromStrings(from, to)
	if err != nil {
		t.Error("Test Failed - NewConversionFromString() error", err)
	}

	if conv.String() != expected {
		t.Errorf("Test Failed - NewConversion() error expected %s but received %s",
			expected,
			conv)
	}
}

func TestNewConversion(t *testing.T) {
	from := NewCode("AUD")
	to := NewCode("USD")
	expected := "AUDUSD"

	conv, err := NewConversion(from, to)
	if err != nil {
		t.Error("Test Failed - NewConversionFromCode() error", err)
	}

	if conv.String() != expected {
		t.Errorf("Test Failed - NewConversion() error expected %s but received %s",
			expected,
			conv)
	}
}

func TestConversionIsInvalid(t *testing.T) {
	from := AUD
	to := USD

	conv, err := NewConversion(from, to)
	if err != nil {
		t.Fatal("Test Failed - NewConversion() error", err)
	}

	if conv.IsInvalid() {
		t.Errorf("Test Failed - IsInvalid() error expected false but received %v",
			conv.IsInvalid())
	}

	to = AUD
	conv, err = NewConversion(from, to)
	if err == nil {
		t.Fatal("Test Failed - NewConversion() error", err)
	}
}

func TestConversionIsFiatPair(t *testing.T) {
	from := AUD
	to := USD

	conv, err := NewConversion(from, to)
	if err != nil {
		t.Fatal("Test Failed - NewConversion() error", err)
	}

	if !conv.IsFiat() {
		t.Errorf("Test Failed - IsFiatPair() error expected true but received %v",
			conv.IsFiat())
	}

	to = LTC
	conv, err = NewConversion(from, to)
	if err == nil {
		t.Fatal("Test Failed - NewConversion() error", err)
	}
}

func TestConversionsRatesSystem(t *testing.T) {
	var SuperDuperConversionSystem ConversionRates

	if SuperDuperConversionSystem.HasData() {
		t.Fatalf("Test Failed - HasData() error expected false but recieved %v",
			SuperDuperConversionSystem.HasData())
	}

	testmap := map[string]float64{
		"USDAUD": 1.3969317581,
		"USDBRL": 3.7047257979,
		"USDCAD": 1.3186386881,
		"USDCHF": 1,
		"USDCNY": 6.7222712044,
		"USDCZK": 22.6406277552,
		"USDDKK": 6.5785575736,
		"USDEUR": 0.8816787163,
		"USDGBP": 0.7665755599,
		"USDHKD": 7.8492329395,
		"USDILS": 3.6152354082,
		"USDINR": 71.154558279,
		"USDJPY": 110.7476635514,
		"USDKRW": 1122.7913948157,
		"USDMXN": 19.1589666725,
		"USDNOK": 8.5818197849,
		"USDNZD": 1.4559160642,
		"USDPLN": 3.8304531829,
		"USDRUB": 65.7533062952,
		"USDSEK": 9.3196085346,
		"USDSGD": 1.3512608006,
		"USDTHB": 31.0950449656,
		"USDZAR": 14.138070887,
	}

	err := SuperDuperConversionSystem.Update(testmap)
	if err != nil {
		t.Fatal("Test Failed - Update() error", err)
	}

	err = SuperDuperConversionSystem.Update(nil)
	if err == nil {
		t.Fatal("Test Failed - Update() error cannnot be nil")
	}

	if !SuperDuperConversionSystem.HasData() {
		t.Fatalf("Test Failed - HasData() error expected true but recieved %v",
			SuperDuperConversionSystem.HasData())
	}

	// * to a rate
	p := SuperDuperConversionSystem.m[USD.Item][AUD.Item]
	// inverse * to a rate
	pi := SuperDuperConversionSystem.m[AUD.Item][USD.Item]
	r := *p * 1000
	expectedRate := 1396.9317581
	if r != expectedRate {
		t.Errorf("Test Failed - Convert() error expected %.13f but recieved %.13f",
			expectedRate,
			r)
	}

	inverseR := *pi * expectedRate
	expectedInverseRate := float64(1000)
	if inverseR != expectedInverseRate {
		t.Errorf("Test Failed - Convert() error expected %.13f but recieved %.13f",
			expectedInverseRate,
			inverseR)
	}
}
