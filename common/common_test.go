package common

import (
	"bytes"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestIsEnabled(t *testing.T) {
	t.Parallel()
	expected := "Enabled"
	actual := IsEnabled(true)
	if actual != expected {
		t.Errorf("Test failed. Expected %s. Actual %s", expected, actual)
	}

	expected = "Disabled"
	actual = IsEnabled(false)
	if actual != expected {
		t.Errorf("Test failed. Expected %s. Actual %s", expected, actual)
	}
}

func TestIsValidCryptoAddress(t *testing.T) {
	t.Parallel()

	b, err := IsValidCryptoAddress("1Mz7153HMuxXTuR2R1t78mGSdzaAtNbBWX", "bTC")
	if err != nil && !b {
		t.Errorf("Test Failed - Common IsValidCryptoAddress error: %s", err)
	}
	b, err = IsValidCryptoAddress("0Mz7153HMuxXTuR2R1t78mGSdzaAtNbBWX", "btc")
	if err == nil && b {
		t.Error("Test Failed - Common IsValidCryptoAddress error")
	}
	b, err = IsValidCryptoAddress("1Mz7153HMuxXTuR2R1t78mGSdzaAtNbBWX", "lTc")
	if err == nil && b {
		t.Error("Test Failed - Common IsValidCryptoAddress error")
	}
	b, err = IsValidCryptoAddress("3CDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj", "ltc")
	if err != nil && !b {
		t.Errorf("Test Failed - Common IsValidCryptoAddress error: %s", err)
	}
	b, err = IsValidCryptoAddress("NCDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj", "lTc")
	if err == nil && b {
		t.Error("Test Failed - Common IsValidCryptoAddress error")
	}
	b, err = IsValidCryptoAddress(
		"0xb794f5ea0ba39494ce839613fffba74279579268",
		"eth",
	)
	if err != nil && b {
		t.Errorf("Test Failed - Common IsValidCryptoAddress error: %s", err)
	}
	b, err = IsValidCryptoAddress(
		"xxb794f5ea0ba39494ce839613fffba74279579268",
		"eTh",
	)
	if err == nil && b {
		t.Error("Test Failed - Common IsValidCryptoAddress error")
	}
	b, err = IsValidCryptoAddress(
		"xxb794f5ea0ba39494ce839613fffba74279579268",
		"ding",
	)
	if err == nil && b {
		t.Error("Test Failed - Common IsValidCryptoAddress error")
	}
}

func TestGetMD5(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the MD5 function in common!")
	var expectedOutput = []byte("18fddf4a41ba90a7352765e62e7a8744")
	actualOutput := GetMD5(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, []byte(actualStr))
	}

}

func TestGetSHA512(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the GetSHA512 function in common!")
	var expectedOutput = []byte(
		`a2273f492ea73fddc4f25c267b34b3b74998bd8a6301149e1e1c835678e3c0b90859fce22e4e7af33bde1711cbb924809aedf5d759d648d61774b7185c5dc02b`,
	)
	actualOutput := GetSHA512(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Errorf("Test failed. Expected '%x'. Actual '%x'",
			expectedOutput, []byte(actualStr))
	}
}

func TestGetSHA256(t *testing.T) {
	t.Parallel()
	var originalString = []byte("I am testing the GetSHA256 function in common!")
	var expectedOutput = []byte(
		"0962813d7a9f739cdcb7f0c0be0c2a13bd630167e6e54468266e4af6b1ad9303",
	)
	actualOutput := GetSHA256(originalString)
	actualStr := HexEncodeToString(actualOutput)
	if !bytes.Equal(expectedOutput, []byte(actualStr)) {
		t.Errorf("Test failed. Expected '%x'. Actual '%x'", expectedOutput,
			[]byte(actualStr))
	}
}

func TestGetHMAC(t *testing.T) {
	expectedSha1 := []byte{
		74, 253, 245, 154, 87, 168, 110, 182, 172, 101, 177, 49, 142, 2, 253, 165,
		100, 66, 86, 246,
	}
	expectedsha256 := []byte{
		54, 68, 6, 12, 32, 158, 80, 22, 142, 8, 131, 111, 248, 145, 17, 202, 224,
		59, 135, 206, 11, 170, 154, 197, 183, 28, 150, 79, 168, 105, 62, 102,
	}
	expectedsha512 := []byte{
		249, 212, 31, 38, 23, 3, 93, 220, 81, 209, 214, 112, 92, 75, 126, 40, 109,
		95, 247, 182, 210, 54, 217, 224, 199, 252, 129, 226, 97, 201, 245, 220, 37,
		201, 240, 15, 137, 236, 75, 6, 97, 12, 190, 31, 53, 153, 223, 17, 214, 11,
		153, 203, 49, 29, 158, 217, 204, 93, 179, 109, 140, 216, 202, 71,
	}
	expectedsha512384 := []byte{
		121, 203, 109, 105, 178, 68, 179, 57, 21, 217, 76, 82, 94, 100, 210, 1, 55,
		201, 8, 232, 194, 168, 165, 58, 192, 26, 193, 167, 254, 183, 172, 4, 189,
		158, 158, 150, 173, 33, 119, 125, 94, 13, 125, 89, 241, 184, 166, 128,
	}

	sha1 := GetHMAC(HashSHA1, []byte("Hello,World"), []byte("1234"))
	if string(sha1) != string(expectedSha1) {
		t.Errorf("Test failed.Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedSha1, sha1,
		)
	}
	sha256 := GetHMAC(HashSHA256, []byte("Hello,World"), []byte("1234"))
	if string(sha256) != string(expectedsha256) {
		t.Errorf("Test failed.Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedSha1, sha1,
		)
	}
	sha512 := GetHMAC(HashSHA512, []byte("Hello,World"), []byte("1234"))
	if string(sha512) != string(expectedsha512) {
		t.Errorf("Test failed.Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedSha1, sha1,
		)
	}
	sha512384 := GetHMAC(HashSHA512_384, []byte("Hello,World"), []byte("1234"))
	if string(sha512384) != string(expectedsha512384) {
		t.Errorf("Test failed.Common GetHMAC error: Expected '%x'. Actual '%x'",
			expectedSha1, sha1,
		)
	}
}

func TestStringToLower(t *testing.T) {
	t.Parallel()
	upperCaseString := "HEY MAN"
	expectedResult := "hey man"
	actualResult := StringToLower(upperCaseString)
	if actualResult != expectedResult {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedResult, actualResult)
	}
}

func TestStringToUpper(t *testing.T) {
	t.Parallel()
	upperCaseString := "hey man"
	expectedResult := "HEY MAN"
	actualResult := StringToUpper(upperCaseString)
	if actualResult != expectedResult {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedResult, actualResult)
	}
}

func TestHexEncodeToString(t *testing.T) {
	t.Parallel()
	originalInput := []byte("string")
	expectedOutput := "737472696e67"
	actualResult := HexEncodeToString(originalInput)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestBase64Decode(t *testing.T) {
	t.Parallel()
	originalInput := "aGVsbG8="
	expectedOutput := []byte("hello")
	actualResult, err := Base64Decode(originalInput)
	if !bytes.Equal(actualResult, expectedOutput) {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'. Error: %s",
			expectedOutput, actualResult, err)
	}

	_, err = Base64Decode("-")
	if err == nil {
		t.Error("Test failed. Bad base64 string failed returned nil error")
	}
}

func TestBase64Encode(t *testing.T) {
	t.Parallel()
	originalInput := []byte("hello")
	expectedOutput := "aGVsbG8="
	actualResult := Base64Encode(originalInput)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestStringSliceDifference(t *testing.T) {
	t.Parallel()
	originalInputOne := []string{"hello"}
	originalInputTwo := []string{"hello", "moto"}
	expectedOutput := []string{"hello moto"}
	actualResult := StringSliceDifference(originalInputOne, originalInputTwo)
	if reflect.DeepEqual(expectedOutput, actualResult) {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestStringContains(t *testing.T) {
	t.Parallel()
	originalInput := "hello"
	originalInputSubstring := "he"
	expectedOutput := true
	actualResult := StringContains(originalInput, originalInputSubstring)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
}

func TestStringDataContains(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"hello", "world", "USDT", "Contains", "string"}
	originalNeedle := "USD"
	anotherNeedle := "thing"
	expectedOutput := true
	expectedOutputTwo := false
	actualResult := StringDataContains(originalHaystack, originalNeedle)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
	actualResult = StringDataContains(originalHaystack, anotherNeedle)
	if actualResult != expectedOutputTwo {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
}

func TestStringDataCompare(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"hello", "WoRld", "USDT", "Contains", "string"}
	originalNeedle := "WoRld"
	anotherNeedle := "USD"
	expectedOutput := true
	expectedOutputTwo := false
	actualResult := StringDataCompare(originalHaystack, originalNeedle)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
	actualResult = StringDataCompare(originalHaystack, anotherNeedle)
	if actualResult != expectedOutputTwo {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
}

func TestStringDataContainsUpper(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"bLa", "BrO", "sUp"}
	originalNeedle := "Bla"
	anotherNeedle := "ning"
	expectedOutput := true
	expectedOutputTwo := false
	actualResult := StringDataContainsUpper(originalHaystack, originalNeedle)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
	actualResult = StringDataContainsUpper(originalHaystack, anotherNeedle)
	if actualResult != expectedOutputTwo {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
}

func TestJoinStrings(t *testing.T) {
	t.Parallel()
	originalInputOne := []string{"hello", "moto"}
	separator := ","
	expectedOutput := "hello,moto"
	actualResult := JoinStrings(originalInputOne, separator)
	if expectedOutput != actualResult {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestSplitStrings(t *testing.T) {
	t.Parallel()
	originalInputOne := "hello,moto"
	separator := ","
	expectedOutput := []string{"hello", "moto"}
	actualResult := SplitStrings(originalInputOne, separator)
	if !reflect.DeepEqual(expectedOutput, actualResult) {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

func TestTrimString(t *testing.T) {
	t.Parallel()
	originalInput := "abcd"
	cutset := "ad"
	expectedOutput := "bc"
	actualResult := TrimString(originalInput, cutset)
	if expectedOutput != actualResult {
		t.Errorf("Test failed. Expected '%s'. Actual '%s'",
			expectedOutput, actualResult)
	}
}

// ReplaceString replaces a string with another
func TestReplaceString(t *testing.T) {
	t.Parallel()
	currency := "BTC-USD"
	expectedOutput := "BTCUSD"

	actualResult := ReplaceString(currency, "-", "", -1)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'", expectedOutput, actualResult,
		)
	}

	currency = "BTC-USD--"
	actualResult = ReplaceString(currency, "-", "", 3)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'", expectedOutput, actualResult,
		)
	}
}

func TestRoundFloat(t *testing.T) {
	t.Parallel()
	// mapping of input vs expected result
	testTable := map[float64]float64{
		2.3232323:  2.32,
		-2.3232323: -2.32,
	}
	for testInput, expectedOutput := range testTable {
		actualOutput := RoundFloat(testInput, 2)
		if actualOutput != expectedOutput {
			t.Errorf("Test failed. RoundFloat Expected '%f'. Actual '%f'.",
				expectedOutput, actualOutput)
		}
	}
}

func TestYesOrNo(t *testing.T) {
	t.Parallel()
	if !YesOrNo("y") {
		t.Error("Test failed - Common YesOrNo Error.")
	}
	if !YesOrNo("yes") {
		t.Error("Test failed - Common YesOrNo Error.")
	}
	if YesOrNo("ding") {
		t.Error("Test failed - Common YesOrNo Error.")
	}
}

func TestCalculateFee(t *testing.T) {
	t.Parallel()
	originalInput := float64(1)
	fee := float64(1)
	expectedOutput := float64(0.01)
	actualResult := CalculateFee(originalInput, fee)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculateAmountWithFee(t *testing.T) {
	t.Parallel()
	originalInput := float64(1)
	fee := float64(1)
	expectedOutput := float64(1.01)
	actualResult := CalculateAmountWithFee(originalInput, fee)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculatePercentageGainOrLoss(t *testing.T) {
	t.Parallel()
	originalInput := float64(9300)
	secondInput := float64(9000)
	expectedOutput := 3.3333333333333335
	actualResult := CalculatePercentageGainOrLoss(originalInput, secondInput)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestCalculatePercentageDifference(t *testing.T) {
	t.Parallel()
	originalInput := float64(10)
	secondAmount := float64(5)
	expectedOutput := 66.66666666666666
	actualResult := CalculatePercentageDifference(originalInput, secondAmount)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
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
		t.Errorf(
			"Test failed. Expected '%f'. Actual '%f'.", expectedOutput, actualResult)
	}
}

func TestSendHTTPRequest(t *testing.T) {
	methodPost := "pOst"
	methodGet := "GeT"
	methodDelete := "dEleTe"
	methodGarbage := "ding"

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	_, err := SendHTTPRequest(
		methodGarbage, "https://query.yahooapis.com/v1/public/yql", headers,
		strings.NewReader(""),
	)
	if err == nil {
		t.Error("Test failed. ")
	}
	_, err = SendHTTPRequest(
		methodPost, "https://query.yahooapis.com/v1/public/yql", headers,
		strings.NewReader(""),
	)
	if err != nil {
		t.Errorf("Test failed. %s ", err)
	}
	_, err = SendHTTPRequest(
		methodGet, "https://query.yahooapis.com/v1/public/yql", headers,
		strings.NewReader(""),
	)
	if err != nil {
		t.Errorf("Test failed. %s ", err)
	}
	_, err = SendHTTPRequest(
		methodDelete, "https://query.yahooapis.com/v1/public/yql", headers,
		strings.NewReader(""),
	)
	if err != nil {
		t.Errorf("Test failed. %s ", err)
	}
}

func TestSendHTTPGetRequest(t *testing.T) {
	type test struct {
		Address string `json:"address"`
		ETH     struct {
			Balance  int `json:"balance"`
			TotalIn  int `json:"totalIn"`
			TotalOut int `json:"totalOut"`
		} `json:"ETH"`
	}
	url := `https://api.ethplorer.io/getAddressInfo/0xff71cb760666ab06aa73f34995b42dd4b85ea07b?apiKey=freekey`
	result := test{}

	err := SendHTTPGetRequest(url, true, false, &result)
	if err != nil {
		t.Errorf("Test failed - common SendHTTPGetRequest error: %s", err)
	}
	err = SendHTTPGetRequest("DINGDONG", true, false, &result)
	if err == nil {
		t.Error("Test failed - common SendHTTPGetRequest error")
	}
	err = SendHTTPGetRequest(url, false, false, &result)
	if err != nil {
		t.Error("Test failed - common SendHTTPGetRequest error")
	}
}

func TestJSONEncode(t *testing.T) {
	type test struct {
		Status int `json:"status"`
		Data   []struct {
			Address   string      `json:"address"`
			Balance   float64     `json:"balance"`
			Nonce     interface{} `json:"nonce"`
			Code      string      `json:"code"`
			Name      interface{} `json:"name"`
			Storage   interface{} `json:"storage"`
			FirstSeen interface{} `json:"firstSeen"`
		} `json:"data"`
	}
	expectOutputString := `{"status":0,"data":null}`
	v := test{}

	bitey, err := JSONEncode(v)
	if err != nil {
		t.Errorf("Test failed - common JSONEncode error: %s", err)
	}
	if string(bitey) != expectOutputString {
		t.Error("Test failed - common JSONEncode error")
	}
	_, err = JSONEncode("WigWham")
	if err != nil {
		t.Errorf("Test failed - common JSONEncode error: %s", err)
	}
}

func TestEncodeURLValues(t *testing.T) {
	urlstring := "https://www.test.com"
	expectedOutput := `https://www.test.com?env=TEST%2FDATABASE&format=json&q=SELECT+%2A+from+yahoo.finance.xchange+WHERE+pair+in+%28%22BTC%2CUSD%22%29`
	values := url.Values{}
	values.Set("q", fmt.Sprintf(
		"SELECT * from yahoo.finance.xchange WHERE pair in (\"%s\")", "BTC,USD"),
	)
	values.Set("format", "json")
	values.Set("env", "TEST/DATABASE")

	output := EncodeURLValues(urlstring, values)
	if output != expectedOutput {
		t.Error("Test Failed - common EncodeURLValues error")
	}
}

func TestExtractHost(t *testing.T) {
	t.Parallel()
	address := "localhost:1337"
	addresstwo := ":1337"
	expectedOutput := "localhost"
	actualResult := ExtractHost(address)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}
	actualResultTwo := ExtractHost(addresstwo)
	if expectedOutput != actualResultTwo {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}

	address = "192.168.1.100:1337"
	expectedOutput = "192.168.1.100"
	actualResult = ExtractHost(address)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}
}

func TestExtractPort(t *testing.T) {
	t.Parallel()
	address := "localhost:1337"
	expectedOutput := 1337
	actualResult := ExtractPort(address)
	if expectedOutput != actualResult {
		t.Errorf(
			"Test failed. Expected '%d'. Actual '%d'.", expectedOutput, actualResult)
	}
}

func TestOutputCSV(t *testing.T) {
	path := "../testdata/dump"
	data := [][]string{}
	rowOne := []string{"Appended", "to", "two", "dimensional", "array"}
	rowTwo := []string{"Appended", "to", "two", "dimensional", "array", "two"}
	data = append(data, rowOne)
	data = append(data, rowTwo)

	err := OutputCSV(path, data)
	if err != nil {
		t.Errorf("Test failed - common OutputCSV error: %s", err)
	}
}

func TestUnixTimestampToTime(t *testing.T) {
	t.Parallel()
	testTime := int64(1489439831)
	tm := time.Unix(testTime, 0)
	expectedOutput := "2017-03-13 21:17:11 +0000 UTC"
	actualResult := UnixTimestampToTime(testTime)
	if tm.String() != actualResult.String() {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}
}

func TestUnixTimestampStrToTime(t *testing.T) {
	t.Parallel()
	testTime := "1489439831"
	incorrectTime := "DINGDONG"
	expectedOutput := "2017-03-13 21:17:11 +0000 UTC"
	actualResult, err := UnixTimestampStrToTime(testTime)
	if err != nil {
		t.Error(err)
	}
	if actualResult.UTC().String() != expectedOutput {
		t.Errorf(
			"Test failed. Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}
	actualResult, err = UnixTimestampStrToTime(incorrectTime)
	if err == nil {
		t.Error("Test failed. Common UnixTimestampStrToTime error")
	}
}

func TestReadFile(t *testing.T) {
	pathCorrect := "../testdata/dump"
	pathIncorrect := "testdata/dump"

	_, err := ReadFile(pathCorrect)
	if err != nil {
		t.Errorf("Test failed - Common ReadFile error: %s", err)
	}
	_, err = ReadFile(pathIncorrect)
	if err == nil {
		t.Errorf("Test failed - Common ReadFile error")
	}
}

func TestWriteFile(t *testing.T) {
	path := "../testdata/writefiletest"
	err := WriteFile(path, nil)
	if err != nil {
		t.Errorf("Test failed. Common WriteFile error: %s", err)
	}
	_, err = ReadFile(path)
	if err != nil {
		t.Errorf("Test failed. Common WriteFile error: %s", err)
	}

	err = WriteFile("", nil)
	if err == nil {
		t.Error("Test failed. Common WriteFile allowed bad path")
	}
}

func TestRemoveFile(t *testing.T) {
	TestWriteFile(t)
	path := "../testdata/writefiletest"
	err := RemoveFile(path)
	if err != nil {
		t.Errorf("Test failed. Common RemoveFile error: %s", err)
	}

	TestOutputCSV(t)
	path = "../testdata/dump"
	err = RemoveFile(path)
	if err != nil {
		t.Errorf("Test failed. Common RemoveFile error: %s", err)
	}
}

func TestGetURIPath(t *testing.T) {
	t.Parallel()
	// mapping of input vs expected result
	testTable := map[string]string{
		"https://api.gdax.com/accounts":           "/accounts",
		"https://api.gdax.com/accounts?a=1&b=2":   "/accounts?a=1&b=2",
		"http://www.google.com/accounts?!@#$%;^^": "",
	}
	for testInput, expectedOutput := range testTable {
		actualOutput := GetURIPath(testInput)
		if actualOutput != expectedOutput {
			t.Errorf("Test failed. Expected '%s'. Actual '%s'.",
				expectedOutput, actualOutput)
		}
	}
}
