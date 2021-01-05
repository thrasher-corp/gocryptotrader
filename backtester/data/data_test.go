package data

import "testing"

func TestSomething(t *testing.T) {
	var d Data
	err := d.Load()
	if err != nil {
		t.Error(err)
	}
	d.Latest()
	d.Next()
	o := d.Offset()
	if o != 0 {
		t.Error("something went wrong")
	}
	d.AppendStream(nil)
	d.AppendStream(nil)

	d.Next()
	o = d.Offset()
	if o != 1 {
		t.Error("expected 1")
	}
	d.List()
	d.History()
	d.SetStream(nil)
	st := d.GetStream()
	if st != nil {
		t.Error("expected nil")
	}
	d.Reset()
	d.GetStream()
	d.StreamOpen()
	d.StreamHigh()
	d.StreamClose()
	d.StreamLow()
	d.StreamVol()
	d.SortStream()

}
