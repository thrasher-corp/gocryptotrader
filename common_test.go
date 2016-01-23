package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestIsEnabled(t *testing.T) {
	t.Parallel()
	expected := "Enabled"
	actual := IsEnabled(true)
	if actual != expected {
		t.Error(fmt.Sprintf("Test failed. Expected %s. Actual %s", expected, actual))
	}

	expected = "Disabled"
	actual = IsEnabled(false)
	if actual != expected {
		t.Error(fmt.Sprintf("Test failed. Expected %s. Actual %s", expected, actual))
	}
}

func TestGetMD5(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the MD5 function in common!")
	var expectedOutput = []byte("18fddf4a41ba90a7352765e62e7a8744")
	actualOutput := GetMD5(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'", expectedOutput, []byte(actualStr)))
	}

}

func TestGetSHA512(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the GetSHA512 function in common!")
	var expectedOutput = []byte("a2273f492ea73fddc4f25c267b34b3b74998bd8a6301149e1e1c835678e3c0b90859fce22e4e7af33bde1711cbb924809aedf5d759d648d61774b7185c5dc02b")
	actualOutput := GetSHA512(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Error(fmt.Sprintf("Test failed. Expected '%x'. Actual '%x'", expectedOutput, []byte(actualStr)))
	}
}

func TestGetSHA256(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the GetSHA256 function in common!")
	var expectedOutput = []byte("0962813d7a9f739cdcb7f0c0be0c2a13bd630167e6e54468266e4af6b1ad9303")
	actualOutput := GetSHA256(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Error(fmt.Sprintf("Test failed. Expected '%x'. Actual '%x'", expectedOutput, []byte(actualStr)))
	}
}
