package nonce

import (
	"sync"
	"testing"
)

func TestGet(t *testing.T) {
	var nonce Nonce
	nonce.Set(112321313)
	if expected, result := Value(112321313), nonce.Get(); expected != result {
		t.Errorf("Expected %d got %d", expected, result)
	}
}

func TestGetInc(t *testing.T) {
	var nonce Nonce
	nonce.Set(1)
	if expected, result := Value(2), nonce.GetInc(); expected != result {
		t.Errorf("Expected %d got %d", expected, result)
	}
}

func TestSet(t *testing.T) {
	var nonce Nonce
	nonce.Set(1)
	if result, expected := nonce.Get(), Value(1); expected != result {
		t.Errorf("Expected %d got %d", expected, result)
	}
}

func TestString(t *testing.T) {
	var nonce Nonce
	nonce.Set(12312313131)
	expected := "12312313131"
	result := nonce.String()
	if expected != result {
		t.Errorf("Expected %s got %s", expected, result)
	}

	v := nonce.Get()
	if expected != v.String() {
		t.Errorf("Expected %s got %s", expected, result)
	}
}

func TestNonceConcurrency(t *testing.T) {
	var nonce Nonce
	nonce.Set(12312)

	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() { nonce.GetInc(); wg.Done() }()
	}

	wg.Wait()

	if expected, result := Value(12312+1000), nonce.Get(); expected != result {
		t.Errorf("Expected %d got %d", expected, result)
	}
}
