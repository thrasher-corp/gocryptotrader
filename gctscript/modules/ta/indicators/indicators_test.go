package indicators

import (
	"errors"
	"os"
	"testing"

	objects "github.com/d5/tengo/v2"
)

var (
	testOpen = &objects.Array{}
	testHigh = &objects.Array{}
	testLow = &objects.Array{}
	testClose = &objects.Array{}
	testVol = &objects.Array{}

	)

func TestMain(m *testing.M) {
	testOpen.Value = append(testOpen.Value, &objects.Array{})
	os.Exit(m.Run())
}

func TestMfi(t *testing.T) {
	_, err := mfi()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	ret, err := mfi(testHigh, testLow, testClose, testVol, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}
	t.Log(ret)
}