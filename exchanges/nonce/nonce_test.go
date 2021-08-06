package nonce

import (
	"sync"
	"testing"
)

func TestGet(t *testing.T) {
	var nonce Nonce
	nonce.Set(112321313)
	expected := Value(112321313)
	result := nonce.Get()
	if expected != result {
		t.Errorf("Expected %d got %d", expected, result)
	}
}

func TestGetInc(t *testing.T) {
	var nonce Nonce
	nonce.Set(1)
	expected := Value(2)
	result := nonce.GetInc()
	if expected != result {
		t.Errorf("Expected %d got %d", expected, result)
	}
}

func TestSet(t *testing.T) {
	var nonce Nonce
	nonce.Set(1)
	expected := Value(1)
	result := nonce.Get()
	if expected != result {
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

	result := nonce.Get()
	expected := Value(12312 + 1000)
	if expected != result {
		t.Errorf("Expected %d got %d", expected, result)
	}
}
