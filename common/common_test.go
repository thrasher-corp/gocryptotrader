package common

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
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

func TestStringToLower(t *testing.T) {
	t.Parallel()
	upperCaseString := "HEY MAN"
	expectedResult := "hey man"
	actualResult := StringToLower(upperCaseString)
	if actualResult != expectedResult {
		t.Error("...")
	}
}

func TestStringToUpper(t *testing.T) {
	t.Parallel()
	upperCaseString := "hey man"
	expectedResult := "HEY MAN"
	actualResult := StringToUpper(upperCaseString)
	if actualResult != expectedResult {
		t.Error("...")
	}
}

func TestHexEncodeToString(t *testing.T) {
	t.Parallel()
	originalInput := []byte("string")
	expectedOutput := "737472696e67"
	actualResult := HexEncodeToString(originalInput)
	if actualResult != expectedOutput {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'", expectedOutput, actualResult))
	}
}

func TestBase64Decode(t *testing.T) {
	t.Parallel()
	originalInput := "aGVsbG8="
	expectedOutput := []byte("hello")
	actualResult, err := Base64Decode(originalInput)
	if !bytes.Equal(actualResult, expectedOutput) {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'. Error: %s", expectedOutput, actualResult, err))
	}
}

func TestBase64Encode(t *testing.T) {
	t.Parallel()
	originalInput := []byte("hello")
	expectedOutput := "aGVsbG8="
	actualResult := Base64Encode(originalInput)
	if actualResult != expectedOutput {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'", expectedOutput, actualResult))
	}
}

func TestStringSliceDifference(t *testing.T) {
	t.Parallel()
	originalInputOne := []string{"hello"}
	originalInputTwo := []string{"moto"}
	expectedOutput := []string{"hello moto"}
	actualResult := StringSliceDifference(originalInputOne, originalInputTwo)
	if reflect.DeepEqual(expectedOutput, actualResult) {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'", expectedOutput, actualResult))
	}
}

func TestStringContains(t *testing.T) {
	t.Parallel()
	originalInput := "hello"
	originalInputSubstring := "he"
	expectedOutput := true
	actualResult := StringContains(originalInput, originalInputSubstring)
	if actualResult != expectedOutput {
		t.Error(fmt.Sprintf("Test failed. Expected '%t'. Actual '%t'", expectedOutput, actualResult))
	}
}

func TestJoinStrings(t *testing.T) {
	t.Parallel()
	originalInputOne := []string{"hello", "moto"}
	seperator := ","
	expectedOutput := "hello,moto"
	actualResult := JoinStrings(originalInputOne, seperator)
	if expectedOutput != actualResult {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'", expectedOutput, actualResult))
	}
}

func TestSplitStrings(t *testing.T) {
	t.Parallel()
	originalInputOne := "hello,moto"
	seperator := ","
	expectedOutput := []string{"hello", "moto"}
	actualResult := SplitStrings(originalInputOne, seperator)
	if !reflect.DeepEqual(expectedOutput, actualResult) {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'", expectedOutput, actualResult))
	}
}

func TestRoundFloat(t *testing.T) {
	t.Parallel()
	originalInput := float64(1.4545445445)
	precisionInput := 2
	expectedOutput := float64(1.45)
	actualResult := RoundFloat(originalInput, precisionInput)
	if expectedOutput != actualResult {
		t.Error(fmt.Sprintf("Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult))
	}
}

func TestCalculateFee(t *testing.T) {
	t.Parallel()
	originalInput := float64(1)
	fee := float64(1)
	expectedOutput := float64(0.01)
	actualResult := CalculateFee(originalInput, fee)
	if expectedOutput != actualResult {
		t.Error(fmt.Sprintf("Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult))
	}
}

func TestCalculateAmountWithFee(t *testing.T) {
	t.Parallel()
	originalInput := float64(1)
	fee := float64(1)
	expectedOutput := float64(1.01)
	actualResult := CalculateAmountWithFee(originalInput, fee)
	if expectedOutput != actualResult {
		t.Error(fmt.Sprintf("Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult))
	}
}

func TestCalculatePercentageGainOrLoss(t *testing.T) {
	t.Parallel()
	originalInput := float64(9300)
	secondInput := float64(9000)
	expectedOutput := 3.3333333333333335
	actualResult := CalculatePercentageGainOrLoss(originalInput, secondInput)
	if expectedOutput != actualResult {
		t.Error(fmt.Sprintf("Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult))
	}
}

func TestCalculatePercentageDifference(t *testing.T) {
	t.Parallel()
	originalInput := float64(10)
	secondAmount := float64(5)
	expectedOutput := 66.66666666666666
	actualResult := CalculatePercentageDifference(originalInput, secondAmount)
	if expectedOutput != actualResult {
		t.Error(fmt.Sprintf("Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult))
	}
}

func TestCalculateNetProfit(t *testing.T) {
	t.Parallel()
	amount := float64(5)
	priceThen := float64(1)
	priceNow := float64(10)
	costs := float64(1)
	expectedOutput := float64(44)
	actualResult := CalculateNetProfit(amount, priceThen, priceNow, costs)
	if expectedOutput != actualResult {
		t.Error(fmt.Sprintf("Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult))
	}
}

func TestExtractHost(t *testing.T) {
	t.Parallel()
	address := "localhost:1337"
	expectedOutput := "localhost"
	actualResult := ExtractHost(address)
	if expectedOutput != actualResult {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult))
	}

	address = "192.168.1.100:1337"
	expectedOutput = "192.168.1.100"
	actualResult = ExtractHost(address)
	if expectedOutput != actualResult {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult))
	}
}

func TestExtractPort(t *testing.T) {
	t.Parallel()
	address := "localhost:1337"
	expectedOutput := 1337
	actualResult := ExtractPort(address)
	if expectedOutput != actualResult {
		t.Error(fmt.Sprintf("Test failed. Expected '%d'. Actual '%d'.", expectedOutput, actualResult))
	}
}

func TestUnixTimestampToTime(t *testing.T) {
	t.Parallel()
	testTime := int64(1489439831)
	tm := time.Unix(testTime, 0)
	expectedOutput := "2017-03-13 21:17:11 +0000 UTC"
	actualResult := UnixTimestampToTime(testTime)
	if tm.String() != actualResult.String() {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult))
	}
}

func TestUnixTimestampStrToTime(t *testing.T) {
	t.Parallel()
	testTime := "1489439831"
	expectedOutput := "2017-03-13 21:17:11 +0000 UTC"
	actualResult, err := UnixTimestampStrToTime(testTime)
	if err != nil {
		t.Error(err)
	}

	if actualResult.UTC().String() != expectedOutput {
		t.Error(fmt.Sprintf("Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult))
	}
}
