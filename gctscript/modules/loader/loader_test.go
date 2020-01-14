package loader

import (
	"reflect"
	"testing"
)

func TestGetModuleMap(t *testing.T) {
	x := GetModuleMap()
	xType := reflect.TypeOf(x).String()
	if xType != "*tengo.ModuleMap" {
		t.Fatalf("GetModuleMap() should return pointer to ModuleMap instead received: %v", x)
	}

	if x.Len() == 0 {
		t.Fatal("expected GetModuleMap() to contain module results instead received 0 value")
	}
}
