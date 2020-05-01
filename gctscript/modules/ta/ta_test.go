package ta

import (
	"reflect"
	"testing"
)

func TestGetModuleMap(t *testing.T) {
	x := AllModuleNames()
	xType := reflect.TypeOf(x).Kind()
	if xType != reflect.Slice {
		t.Fatalf("AllModuleNames() should return slice instead received: %v", x)
	}
	if len(x) != 8 {
		t.Fatalf("unexpected results received expected 7 received: %v", len(x))
	}
}
