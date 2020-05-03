package gct

import (
	"testing"

	objects "github.com/d5/tengo/v2"
)

var (
	testStuff  = &objects.String{Value: "Hello"}
	testStuff2 = &objects.String{Value: ","}
	testStuff3 = &objects.String{Value: "World"}
)

func TestLoggerInfo(t *testing.T) {
	t.Parallel()
	_, err := Info()
	if err != nil {
		t.Fatal(err)
	}

	_, err = Info(testStuff, testStuff2, testStuff3)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoggerDebug(t *testing.T) {
	t.Parallel()
	_, err := Debug()
	if err != nil {
		t.Fatal(err)
	}

	_, err = Debug(testStuff, testStuff2, testStuff3)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoggerWarn(t *testing.T) {
	t.Parallel()
	_, err := Warn()
	if err != nil {
		t.Fatal(err)
	}

	_, err = Warn(testStuff, testStuff2, testStuff3)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoggerError(t *testing.T) {
	t.Parallel()
	_, err := Error()
	if err != nil {
		t.Fatal(err)
	}

	_, err = Error(testStuff, testStuff2, testStuff3)
	if err != nil {
		t.Fatal(err)
	}
}
