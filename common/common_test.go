package common

import (
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
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

func TestStringDataCompareUpper(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"hello", "WoRld", "USDT", "Contains", "string"}
	originalNeedle := "WoRld"
	anotherNeedle := "WoRldD"
	expectedOutput := true
	expectedOutputTwo := false
	actualResult := StringDataCompareInsensitive(originalHaystack, originalNeedle)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}

	actualResult = StringDataCompareInsensitive(originalHaystack, anotherNeedle)
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
	actualResult := StringDataContainsInsensitive(originalHaystack, originalNeedle)
	if actualResult != expectedOutput {
		t.Errorf("Test failed. Expected '%v'. Actual '%v'",
			expectedOutput, actualResult)
	}
	actualResult = StringDataContainsInsensitive(originalHaystack, anotherNeedle)
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

// TestReplaceString replaces a string with another
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

func TestSendHTTPRequest(t *testing.T) {
	methodPost := "pOst"
	methodGet := "GeT"
	methodDelete := "dEleTe"
	methodGarbage := "ding"

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	_, err := SendHTTPRequest(
		methodGarbage, "https://www.google.com", headers,
		strings.NewReader(""),
	)
	if err == nil {
		t.Error("Test failed. ")
	}
	_, err = SendHTTPRequest(
		methodPost, "https://www.google.com", headers,
		strings.NewReader(""),
	)
	if err != nil {
		t.Errorf("Test failed. %s ", err)
	}
	_, err = SendHTTPRequest(
		methodGet, "https://www.google.com", headers,
		strings.NewReader(""),
	)
	if err != nil {
		t.Errorf("Test failed. %s ", err)
	}
	_, err = SendHTTPRequest(
		methodDelete, "https://www.google.com", headers,
		strings.NewReader(""),
	)
	if err != nil {
		t.Errorf("Test failed. %s ", err)
	}
	_, err = SendHTTPRequest(
		methodGet, ":missingprotocolscheme", headers,
		strings.NewReader(""),
	)
	if err == nil {
		t.Error("Test failed. Common HTTPRequest accepted missing protocol")
	}
	_, err = SendHTTPRequest(
		methodGet, "test://unsupportedprotocolscheme", headers,
		strings.NewReader(""),
	)
	if err == nil {
		t.Error("Test failed. Common HTTPRequest accepted invalid protocol")
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
	ethURL := `https://api.ethplorer.io/getAddressInfo/0xff71cb760666ab06aa73f34995b42dd4b85ea07b?apiKey=freekey`
	result := test{}

	var badresult int

	err := SendHTTPGetRequest(ethURL, true, true, &result)
	if err != nil {
		t.Errorf("Test failed - common SendHTTPGetRequest error: %s", err)
	}
	err = SendHTTPGetRequest("DINGDONG", true, false, &result)
	if err == nil {
		t.Error("Test failed - common SendHTTPGetRequest error")
	}
	err = SendHTTPGetRequest(ethURL, false, false, &result)
	if err != nil {
		t.Errorf("Test failed - common SendHTTPGetRequest error: %s", err)
	}
	err = SendHTTPGetRequest("https://httpstat.us/202", false, false, &result)
	if err == nil {
		t.Error("Test failed = common SendHTTPGetRequest error: Ignored unexpected status code")
	}
	err = SendHTTPGetRequest(ethURL, true, false, &badresult)
	if err == nil {
		t.Error("Test failed - common SendHTTPGetRequest error: Unmarshalled into bad type")
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

func TestJSONDecode(t *testing.T) {
	t.Parallel()
	var data []byte
	result := "Not a memory address"
	err := JSONDecode(data, result)
	if err == nil {
		t.Error("Test failed. Common JSONDecode, unmarshalled when address not supplied")
	}

	type test struct {
		Status int `json:"status"`
		Data   []struct {
			Address string  `json:"address"`
			Balance float64 `json:"balance"`
		} `json:"data"`
	}

	var v test
	data = []byte(`{"status":1,"data":null}`)
	err = JSONDecode(data, &v)
	if err != nil || v.Status != 1 {
		t.Errorf("Test failed. Common JSONDecode. Data: %v \nError: %s",
			v, err)
	}
}

func TestEncodeURLValues(t *testing.T) {
	urlstring := "https://www.test.com"
	expectedOutput := `https://www.test.com?env=TEST%2FDATABASE&format=json`
	values := url.Values{}
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
	var data [][]string
	rowOne := []string{"Appended", "to", "two", "dimensional", "array"}
	rowTwo := []string{"Appended", "to", "two", "dimensional", "array", "two"}
	data = append(data, rowOne, rowTwo)

	err := OutputCSV(path, data)
	if err != nil {
		t.Errorf("Test failed - common OutputCSV error: %s", err)
	}
	err = OutputCSV("/:::notapath:::", data)
	if err == nil {
		t.Error("Test failed - common OutputCSV, tried writing to invalid path")
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
		"https://api.pro.coinbase.com/accounts":         "/accounts",
		"https://api.pro.coinbase.com/accounts?a=1&b=2": "/accounts?a=1&b=2",
		"http://www.google.com/accounts?!@#$%;^^":       "",
	}
	for testInput, expectedOutput := range testTable {
		actualOutput := GetURIPath(testInput)
		if actualOutput != expectedOutput {
			t.Errorf("Test failed. Expected '%s'. Actual '%s'.",
				expectedOutput, actualOutput)
		}
	}
}

func TestGetExecutablePath(t *testing.T) {
	t.Parallel()
	_, err := GetExecutablePath()
	if err != nil {
		t.Errorf("Test failed. Common GetExecutablePath. Error: %s", err)
	}
}

func TestGetOSPathSlash(t *testing.T) {
	output := GetOSPathSlash()
	if output != "/" && output != "\\" {
		t.Errorf("Test failed. Common GetOSPathSlash. Returned '%s'", output)
	}

}

func TestUnixMillis(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2014, time.October, 28, 0, 32, 0, 0, time.UTC)
	expectedOutput := int64(1414456320000)

	actualOutput := UnixMillis(testTime)
	if actualOutput != expectedOutput {
		t.Errorf("Test failed. Common UnixMillis. Expected '%d'. Actual '%d'.",
			expectedOutput, actualOutput)
	}
}

func TestRecvWindow(t *testing.T) {
	t.Parallel()
	testTime := time.Duration(24760000)
	expectedOutput := int64(24)

	actualOutput := RecvWindow(testTime)
	if actualOutput != expectedOutput {
		t.Errorf("Test failed. Common RecvWindow. Expected '%d'. Actual '%d'",
			expectedOutput, actualOutput)
	}
}

func TestFloatFromString(t *testing.T) {
	t.Parallel()
	testString := "1.41421356237"
	expectedOutput := float64(1.41421356237)

	actualOutput, err := FloatFromString(testString)
	if actualOutput != expectedOutput || err != nil {
		t.Errorf("Test failed. Common FloatFromString. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	var testByte []byte
	_, err = FloatFromString(testByte)
	if err == nil {
		t.Error("Test failed. Common FloatFromString. Converted non-string.")
	}

	testString = "   something unconvertible  "
	_, err = FloatFromString(testString)
	if err == nil {
		t.Error("Test failed. Common FloatFromString. Converted invalid syntax.")
	}
}

func TestIntFromString(t *testing.T) {
	t.Parallel()
	testString := "1337"
	expectedOutput := 1337

	actualOutput, err := IntFromString(testString)
	if actualOutput != expectedOutput || err != nil {
		t.Errorf("Test failed. Common IntFromString. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	var testByte []byte
	_, err = IntFromString(testByte)
	if err == nil {
		t.Error("Test failed. Common IntFromString. Converted non-string.")
	}

	testString = "1.41421356237"
	_, err = IntFromString(testString)
	if err == nil {
		t.Error("Test failed. Common IntFromString. Converted invalid syntax.")
	}
}

func TestInt64FromString(t *testing.T) {
	t.Parallel()
	testString := "4398046511104"
	expectedOutput := int64(1 << 42)

	actualOutput, err := Int64FromString(testString)
	if actualOutput != expectedOutput || err != nil {
		t.Errorf("Test failed. Common Int64FromString. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	var testByte []byte
	_, err = Int64FromString(testByte)
	if err == nil {
		t.Error("Test failed. Common Int64FromString. Converted non-string.")
	}

	testString = "1.41421356237"
	_, err = Int64FromString(testString)
	if err == nil {
		t.Error("Test failed. Common Int64FromString. Converted invalid syntax.")
	}
}

func TestTimeFromUnixTimestampFloat(t *testing.T) {
	t.Parallel()
	testTimestamp := float64(1414456320000)
	expectedOutput := time.Date(2014, time.October, 28, 0, 32, 0, 0, time.UTC)

	actualOutput, err := TimeFromUnixTimestampFloat(testTimestamp)
	if actualOutput.UTC().String() != expectedOutput.UTC().String() || err != nil {
		t.Errorf("Test failed. Common TimeFromUnixTimestampFloat. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	testString := "Time"
	_, err = TimeFromUnixTimestampFloat(testString)
	if err == nil {
		t.Error("Test failed. Common TimeFromUnixTimestampFloat. Converted invalid syntax.")
	}
}

func TestGetDefaultDataDir(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		dir, ok := os.LookupEnv("APPDATA")
		if !ok {
			t.Fatal("APPDATA is not set")
		}
		dir = filepath.Join(dir, "GoCryptoTrader")
		actualOutput := GetDefaultDataDir(runtime.GOOS)
		if actualOutput != dir {
			t.Fatalf("Unexpected result. Got: %v Expected: %v", actualOutput, dir)
		}
	default:
		var dir string
		usr, err := user.Current()
		if err == nil {
			dir = usr.HomeDir
		} else {
			var err error
			dir, err = os.UserHomeDir()
			if err != nil {
				dir = "."
			}
		}
		dir = filepath.Join(dir, ".gocryptotrader")
		actualOutput := GetDefaultDataDir(runtime.GOOS)
		if actualOutput != dir {
			t.Fatalf("Unexpected result. Got: %v Expected: %v", actualOutput, dir)
		}
	}
}

func TestCreateDir(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		// test for looking up an invalid directory
		err := CreateDir("")
		if err == nil {
			t.Fatal("expected err due to invalid path, but got nil")
		}

		// test for a directory that exists
		dir, ok := os.LookupEnv("TEMP")
		if !ok {
			t.Fatal("LookupEnv failed. TEMP is not set")
		}
		err = CreateDir(dir)
		if err != nil {
			t.Fatalf("CreateDir failed. Err: %v", err)
		}

		// test for creating a directory
		dir, ok = os.LookupEnv("APPDATA")
		if !ok {
			t.Fatal("LookupEnv failed. APPDATA is not set")
		}
		dir = dir + GetOSPathSlash() + "GoCryptoTrader\\TestFileASDFG"
		err = CreateDir(dir)
		if err != nil {
			t.Fatalf("CreateDir failed. Err: %v", err)
		}
		err = os.Remove(dir)
		if err != nil {
			t.Fatalf("Failed to remove file. Err: %v", err)
		}
	default:
		err := CreateDir("")
		if err == nil {
			t.Fatal("expected err due to invalid path, but got nil")
		}

		dir := "/home"
		err = CreateDir(dir)
		if err != nil {
			t.Fatalf("CreateDir failed. Err: %v", err)
		}
		var ok bool
		dir, ok = os.LookupEnv("HOME")
		if !ok {
			t.Fatal("LookupEnv of HOME failed")
		}
		dir = filepath.Join(dir, ".gocryptotrader", "TestFileASFG")
		err = CreateDir(dir)
		if err != nil {
			t.Errorf("CreateDir failed. Err: %s", err)
		}
		err = os.Remove(dir)
		if err != nil {
			t.Fatalf("Failed to remove file. Err: %v", err)
		}
	}
}

func TestChangePerm(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		err := ChangePerm("*")
		if err == nil {
			t.Fatal("expected an error on non-existent path")
		}
		err = os.Mkdir(GetDefaultDataDir(runtime.GOOS)+GetOSPathSlash()+"TestFileASDFGHJ", 0777)
		if err != nil {
			t.Fatalf("Mkdir failed. Err: %v", err)
		}
		err = ChangePerm(GetDefaultDataDir(runtime.GOOS))
		if err != nil {
			t.Fatalf("ChangePerm was unsuccessful. Err: %v", err)
		}
		_, err = os.Stat(GetDefaultDataDir(runtime.GOOS) + GetOSPathSlash() + "TestFileASDFGHJ")
		if err != nil {
			t.Fatalf("os.Stat failed. Err: %v", err)
		}
		err = RemoveFile(GetDefaultDataDir(runtime.GOOS) + GetOSPathSlash() + "TestFileASDFGHJ")
		if err != nil {
			t.Fatalf("RemoveFile failed. Err: %v", err)
		}
	default:
		err := ChangePerm("")
		if err == nil {
			t.Fatal("expected an error on non-existent path")
		}
		err = os.Mkdir(GetDefaultDataDir(runtime.GOOS)+GetOSPathSlash()+"TestFileASDFGHJ", 0777)
		if err != nil {
			t.Fatalf("Mkdir failed. Err: %v", err)
		}
		err = ChangePerm(GetDefaultDataDir(runtime.GOOS))
		if err != nil {
			t.Fatalf("ChangePerm was unsuccessful. Err: %v", err)
		}
		var a os.FileInfo
		a, err = os.Stat(GetDefaultDataDir(runtime.GOOS) + GetOSPathSlash() + "TestFileASDFGHJ")
		if err != nil {
			t.Fatalf("os.Stat failed. Err: %v", err)
		}
		if a.Mode().Perm() != 0770 {
			t.Fatalf("expected file permissions differ. expecting 0770 got %#o", a.Mode().Perm())
		}
		err = RemoveFile(GetDefaultDataDir(runtime.GOOS) + GetOSPathSlash() + "TestFileASDFGHJ")
		if err != nil {
			t.Fatalf("RemoveFile failed. Err: %v", err)
		}
	}
}
