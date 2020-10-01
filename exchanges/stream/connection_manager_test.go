package stream

import "testing"

func TestLoadConnection(t *testing.T) {
	man, err := NewConnectionManager(nil)
	if err != nil {
		t.Fatal(err)
	}
	a := &WebsocketConnection{}
	err = man.LoadNewConnection(a)
	if err != nil {
		t.Fatal(err)
	}
	err = man.LoadNewConnection(a)
	if err == nil {
		t.Fatal(err)
	}
}
