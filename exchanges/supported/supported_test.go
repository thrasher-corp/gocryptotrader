package supported

import (
	"testing"
)

func TestCheckExchange(t *testing.T) {
	_, err := CheckExchange("meow")
	if err == nil {
		t.Fatal(err)
	}

	s, err := CheckExchange("btcmarkets")
	if err != nil {
		t.Fatal(err)
	}

	if s != Btcmarkets {
		t.Fatal("wrong string")
	}

	s, err = CheckExchange("btc markets")
	if err != nil {
		t.Fatal(err)
	}

	if s != Btcmarkets {
		t.Fatal("wrong string")
	}
}
