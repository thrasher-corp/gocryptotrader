package currency

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConversionFromString(t *testing.T) {
	conv, err := NewConversionFromString("AUDUSD")
	assert.NoError(t, err, "NewConversionFromString should not error")
	assert.Equal(t, "AUDUSD", conv.String(), "Should provide correct conversion currency")
	r, err := conv.GetRate()
	assert.NoError(t, err, "GetRate should not error")
	assert.Positive(t, r, "Should provide correct conversion rate")

	conv, err = NewConversionFromString("audusd")
	assert.NoError(t, err, "NewConversionFromString should not error")
	assert.Equal(t, "audusd", conv.String(), "Should provide correct conversion for lowercase")
	r, err = conv.GetRate()
	assert.NoError(t, err, "GetRate should not error")
	assert.Positive(t, r, "Should provide correct conversion rate")
}

func TestNewConversionFromStrings(t *testing.T) {
	from := "AUD"
	to := "USD"

	conv, err := NewConversionFromStrings(from, to)
	if err != nil {
		t.Error(err)
	}

	if expected := "AUDUSD"; conv.String() != expected {
		t.Errorf("NewConversion() error expected %s but received %s",
			expected,
			conv)
	}
}

func TestNewConversion(t *testing.T) {
	from := NewCode("AUD")
	to := NewCode("USD")

	conv, err := NewConversion(from, to)
	if err != nil {
		t.Error(err)
	}

	if expected := "AUDUSD"; conv.String() != expected {
		t.Errorf("NewConversion() error expected %s but received %s",
			expected,
			conv)
	}
}

func TestConversionIsInvalid(t *testing.T) {
	from := AUD
	to := USD

	conv, err := NewConversion(from, to)
	if err != nil {
		t.Fatal(err)
	}

	if conv.IsInvalid() {
		t.Errorf("IsInvalid() error expected false but received %v",
			conv.IsInvalid())
	}

	to = AUD
	if _, err = NewConversion(from, to); err == nil {
		t.Error("Expected error")
	}
}

func TestConversionIsFiatPair(t *testing.T) {
	from := AUD
	to := USD

	conv, err := NewConversion(from, to)
	if err != nil {
		t.Fatal(err)
	}

	if !conv.IsFiat() {
		t.Errorf("IsFiatPair() error expected true but received %v",
			conv.IsFiat())
	}

	to = LTC
	if _, err = NewConversion(from, to); err == nil {
		t.Error("Expected error")
	}
}

func TestConversionsRatesSystem(t *testing.T) {
	var SuperDuperConversionSystem ConversionRates

	if SuperDuperConversionSystem.HasData() {
		t.Fatalf("HasData() error expected false but received %v",
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
		t.Fatal(err)
	}

	err = SuperDuperConversionSystem.Update(nil)
	if err == nil {
		t.Fatal("Update() error cannot be nil")
	}

	if !SuperDuperConversionSystem.HasData() {
		t.Fatalf("HasData() error expected true but received %v",
			SuperDuperConversionSystem.HasData())
	}

	// * to a rate
	p := SuperDuperConversionSystem.m[USD.Item][AUD.Item]
	// inverse * to a rate
	pi := SuperDuperConversionSystem.m[AUD.Item][USD.Item]

	expectedRate := 1396.9317581
	if r := *p * 1000; r != expectedRate {
		t.Errorf("Convert() error expected %.13f but received %.13f",
			expectedRate,
			r)
	}

	expectedInverseRate := float64(1000)
	if inverseR := *pi * expectedRate; inverseR != expectedInverseRate {
		t.Errorf("Convert() error expected %.13f but received %.13f",
			expectedInverseRate,
			inverseR)
	}
}

func TestGetRate(t *testing.T) {
	from := NewCode("AUD")
	to := NewCode("USD")

	c, err := NewConversion(from, to)
	if err != nil {
		t.Fatal(err)
	}
	rate, err := c.GetRate()
	if err != nil {
		t.Error(err)
	}
	if rate == 0 {
		t.Error("Rate not set")
	}
	inv, err := c.GetInversionRate()
	if err != nil {
		t.Error(err)
	}
	if inv == 0 {
		t.Error("Inverted rate not set")
	}
	conv, err := c.Convert(1)
	if err != nil {
		t.Error(err)
	}
	if rate != conv {
		t.Errorf("Incorrect rate %v %v", rate, conv)
	}
	invConv, err := c.ConvertInverse(1)
	if err != nil {
		t.Error(err)
	}
	if inv != invConv {
		t.Errorf("Incorrect rate %v %v", conv, invConv)
	}

	var convs ConversionRates
	var convRate float64
	_, err = convs.GetRate(BTC, USDT)
	if err == nil {
		t.Errorf("Expected %s", fmt.Errorf("rate not found for from %s to %s conversion",
			BTC,
			USD))
	}
	convRate, err = convs.GetRate(USDT, USD)
	if err != nil {
		t.Error(err)
	}
	if convRate != 1 {
		t.Errorf("Expected rate to be 1")
	}

	convRate, err = convs.GetRate(RUR, RUB)
	if err != nil {
		t.Error(err)
	}
	if convRate != 1 {
		t.Errorf("Expected rate to be 1")
	}

	convRate, err = convs.GetRate(RUB, RUR)
	if err != nil {
		t.Error(err)
	}
	if convRate != 1 {
		t.Errorf("Expected rate to be 1")
	}
}
