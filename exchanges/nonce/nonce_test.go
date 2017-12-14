package nonce

import (
	"strconv"
	"testing"
	"time"
)

func TestInc(t *testing.T) {
	var nonce Nonce
	nonce.Set(1)
	nonce.Inc()
	expected := int64(2)
	result := nonce.Get()
	if result != expected {
		t.Errorf("Test failed. Expected %d got %d", expected, result)
	}
}

func TestGet(t *testing.T) {
	var nonce Nonce
	nonce.Set(112321313)
	expected := int64(112321313)
	result := nonce.Get()
	if expected != result {
		t.Errorf("Test failed. Expected %d got %d", expected, result)
	}
}

func TestGetInc(t *testing.T) {
	var nonce Nonce
	nonce.Set(1)
	expected := int64(2)
	result := nonce.GetInc()
	if expected != result {
		t.Errorf("Test failed. Expected %d got %d", expected, result)
	}
}

func TestSet(t *testing.T) {
	var nonce Nonce
	nonce.Set(1)
	expected := int64(1)
	result := nonce.Get()
	if expected != result {
		t.Errorf("Test failed. Expected %d got %d", expected, result)
	}
}

func TestString(t *testing.T) {
	var nonce Nonce
	nonce.Set(12312313131)
	expected := "12312313131"
	result := nonce.String()
	if expected != result {
		t.Errorf("Test failed. Expected %s got %s", expected, result)
	}
}

func TestGetValue(t *testing.T) {
	var nonce Nonce
	timeNowNano := strconv.FormatInt(time.Now().UnixNano(), 10)
	time.Sleep(time.Millisecond * 100)
	nValue := nonce.GetValue("dingdong", true).String()

	if timeNowNano == nValue {
		t.Error("Test failed - GetValue() error, incorrect values")
	}

	if len(nValue) != 19 {
		t.Error("Test failed - GetValue() error, incorrect values")
	}

	timeNowUnix := nonce.GetValue("dongding", false)
	if len(timeNowUnix.String()) != 10 {
		t.Error("Test failed - GetValue() error, incorrect values")
	}

	n := nonce.GetValue("dongding", false)
	if n != timeNowUnix+1 {
		t.Error("Test failed - GetValue() error, incorrect values")
	}
}

func TestNonceConcurrency(t *testing.T) {
	var nonce Nonce
	nonce.Set(12312)

	for i := 0; i < 1000; i++ {
		go nonce.Inc()
	}

	// Allow sufficient time for all routines to finish
	time.Sleep(time.Second)

	result := nonce.Get()
	expected := int64(12312 + 1000)
	if expected != result {
		t.Errorf("Test failed. Expected %d got %d", expected, result)
	}
}
