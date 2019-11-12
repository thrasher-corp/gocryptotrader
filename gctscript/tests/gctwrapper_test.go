package tests

import (
	"reflect"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/gctscript/gctwrapper"
)

func TestSetup(t *testing.T) {
	x := gctwrapper.Setup()
	xType := reflect.TypeOf(x).String()
	if xType != "*gctwrapper.Wrapper" {
		t.Fatalf("vm.New should return pointer to VM instead received: %v", x)
	}
}
