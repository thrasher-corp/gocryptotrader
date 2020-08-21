package backtest

import "testing"

func TestPercentageFee_Calculate(t *testing.T) {
	v := PercentageFee{
		ExchangeFee{
			Fee: 2.0,
		},
	}

	ret, err := v.Calculate(1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if ret != 20 {
		t.Fatalf("expected fee to return 20 received: %v", ret)
	}

	ret, err = v.Calculate(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ret != 0 {
		t.Fatalf("expected fee to return 20 received: %v", ret)
	}
}

func TestFixedExchangeFee_Calculate(t *testing.T) {
	v := FixedExchangeFee{
		ExchangeFee{
			Fee: 2.0,
		},
	}

	ret, err := v.Calculate(1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if ret != 2 {
		t.Fatalf("expected fee to return 2 received: %v", ret)
	}
}