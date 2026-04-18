package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/file"
)

func TestSendHTTPRequest(t *testing.T) {
	// t.Parallel() not used to maintain code coverage for assigning the default
	// HTTPClient.
	methodPost := "pOst"
	methodGet := "GeT"
	methodDelete := "dEleTe"
	methodGarbage := "ding"

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	_, err := SendHTTPRequest(t.Context(),
		methodGarbage, "https://www.google.com", headers,
		strings.NewReader(""), true,
	)
	if err == nil {
		t.Error("Expected error 'invalid HTTP method specified'")
	}
	_, err = SendHTTPRequest(t.Context(),
		methodPost, "https://www.google.com", headers,
		strings.NewReader(""), true,
	)
	if err != nil {
		t.Error(err)
	}
	_, err = SendHTTPRequest(t.Context(),
		methodGet, "https://www.google.com", headers,
		strings.NewReader(""), true,
	)
	if err != nil {
		t.Error(err)
	}

	err = SetHTTPUserAgent("GCTbot/1337.69 (+http://www.lol.com/)")
	require.NoError(t, err)

	_, err = SendHTTPRequest(t.Context(),
		methodDelete, "https://www.google.com", headers,
		strings.NewReader(""), true,
	)
	if err != nil {
		t.Error(err)
	}
	_, err = SendHTTPRequest(t.Context(),
		methodGet, ":missingprotocolscheme", headers,
		strings.NewReader(""), true,
	)
	if err == nil {
		t.Error("Common HTTPRequest accepted missing protocol")
	}
	_, err = SendHTTPRequest(t.Context(),
		methodGet, "test://unsupportedprotocolscheme", headers,
		strings.NewReader(""), true,
	)
	if err == nil {
		t.Error("Common HTTPRequest accepted invalid protocol")
	}
}

func TestSetHTTPClientWithTimeout(t *testing.T) {
	t.Parallel()
	err := SetHTTPClientWithTimeout(-0)
	require.ErrorIs(t, err, errCannotSetInvalidTimeout)

	err = SetHTTPClientWithTimeout(time.Second * 15)
	require.NoError(t, err)
}

func TestSetHTTPUserAgent(t *testing.T) {
	t.Parallel()
	err := SetHTTPUserAgent("")
	require.ErrorIs(t, err, errUserAgentInvalid)

	err = SetHTTPUserAgent("testy test")
	require.NoError(t, err)
}

func TestSetHTTPClient(t *testing.T) {
	t.Parallel()
	err := SetHTTPClient(nil)
	require.ErrorIs(t, err, errHTTPClientInvalid)

	err = SetHTTPClient(new(http.Client))
	require.NoError(t, err)
}

func TestIsEnabled(t *testing.T) {
	t.Parallel()
	expected := "Enabled"
	actual := IsEnabled(true)
	if actual != expected {
		t.Errorf("Expected %s. Actual %s", expected, actual)
	}

	expected = "Disabled"
	actual = IsEnabled(false)
	if actual != expected {
		t.Errorf("Expected %s. Actual %s", expected, actual)
	}
}

func TestIsValidCryptoAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, addr, code string
		err              error
	}{
		{"Valid BTC legacy", "1Mz7153HMuxXTuR2R1t78mGSdzaAtNbBWX", "bTC", nil},
		{"Valid BTC bech32", "bc1qw508d6qejxtdg4y5r3zarvaly0c5xw7kv8f3t4", "bTC", nil},
		{"Invalid BTC (too long)", "an84characterslonghumanreadablepartthatcontainsthenumber1andtheexcludedcharactersbio1569pvx", "bTC", ErrAddressIsEmptyOrInvalid},
		{"Valid BTC bech32 (longer)", "bc1qc7slrfxkknqcq2jevvvkdgvrt8080852dfjewde450xdlk4ugp7szw5tk9", "bTC", nil},
		{"Invalid BTC (starts with 0)", "0Mz7153HMuxXTuR2R1t78mGSdzaAtNbBWX", "bTC", ErrAddressIsEmptyOrInvalid},
		{"Invalid LTC (BTC address)", "1Mz7153HMuxXTuR2R1t78mGSdzaAtNbBWX", "lTc", ErrAddressIsEmptyOrInvalid},
		{"Valid LTC", "3CDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj", "lTc", nil},
		{"Invalid LTC (starts with N)", "NCDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj", "lTc", ErrAddressIsEmptyOrInvalid},
		{"Valid ETH", "0xb794f5ea0ba39494ce839613fffba74279579268", "eth", nil},
		{"Invalid ETH (starts with xx)", "xxb794f5ea0ba39494ce839613fffba74279579268", "eth", ErrAddressIsEmptyOrInvalid},
		{"Unsupported crypto", "xxb794f5ea0ba39494ce839613fffba74279579268", "wif", ErrUnsupportedCryptocurrency},
		{"Empty address", "", "btc", ErrAddressIsEmptyOrInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, IsValidCryptoAddress(tc.addr, tc.code), tc.err)
		})
	}
}

func TestSliceDifference(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, []string{"world", "go"}, SliceDifference([]string{"hello", "world"}, []string{"hello", "go"}))
	assert.ElementsMatch(t, []int64{1, 2, 5, 6}, SliceDifference([]int64{1, 2, 3, 4}, []int64{3, 4, 5, 6}))
	assert.ElementsMatch(t, []float64{1.1, 4.4}, SliceDifference([]float64{1.1, 2.2, 3.3}, []float64{2.2, 3.3, 4.4}))
	type mixedType struct {
		A string
		B int
	}
	assert.ElementsMatch(t, []mixedType{{"A", 1}, {"D", 4}}, SliceDifference([]mixedType{{"A", 1}, {"B", 2}, {"C", 3}}, []mixedType{{"B", 2}, {"C", 3}, {"D", 4}}))
	assert.ElementsMatch(t, []int{1, 2, 3}, SliceDifference([]int{}, []int{1, 2, 3}))
	assert.ElementsMatch(t, []int{1, 2, 3}, SliceDifference([]int{1, 2, 3}, []int{}))
	assert.Empty(t, SliceDifference([]int{}, []int{}))
}

func TestStringSliceContains(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"hello", "world", "USDT", "Contains", "string"}
	assert.True(t, StringSliceContains(originalHaystack, "USD"), "Should contain 'USD'")
	assert.False(t, StringSliceContains(originalHaystack, "thing"), "Should not contain 'thing'")
}

func TestStringSliceCompareInsensitive(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"hello", "WoRld", "USDT", "Contains", "string"}
	assert.False(t, StringSliceCompareInsensitive(originalHaystack, "USD"), "Should not contain 'USD'")
	assert.True(t, StringSliceCompareInsensitive(originalHaystack, "WORLD"), "Should find 'WoRld'")
}

func TestStringSliceContainsInsensitive(t *testing.T) {
	t.Parallel()
	originalHaystack := []string{"bLa", "BrO", "sUp"}
	assert.True(t, StringSliceContainsInsensitive(originalHaystack, "Bla"), "Should contain 'Bla'")
	assert.False(t, StringSliceContainsInsensitive(originalHaystack, "ning"), "Should not contain 'ning'")
}

func TestYesOrNo(t *testing.T) {
	t.Parallel()
	if !YesOrNo("y") {
		t.Error("Common YesOrNo Error.")
	}
	if !YesOrNo("yes") {
		t.Error("Common YesOrNo Error.")
	}
	if YesOrNo("ding") {
		t.Error("Common YesOrNo Error.")
	}
}

func TestEncodeURLValues(t *testing.T) {
	t.Parallel()
	urlstring := "https://www.test.com"
	expectedOutput := `https://www.test.com?env=TEST%2FDATABASE&format=json`
	values := url.Values{}
	values.Set("format", "json")
	values.Set("env", "TEST/DATABASE")

	output := EncodeURLValues(urlstring, values)
	if output != expectedOutput {
		t.Error("common EncodeURLValues error")
	}
}

func TestExtractHostOrDefault(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "localhost", ExtractHostOrDefault("localhost:1337"))
	assert.Equal(t, "localhost", ExtractHostOrDefault(":1337"))
	assert.Equal(t, "192.168.1.100", ExtractHostOrDefault("192.168.1.100:1337"))
}

func TestExtractPortOrDefault(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1337, ExtractPortOrDefault("localhost:1337"))
	assert.Equal(t, 80, ExtractPortOrDefault("localhost"))
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
		assert.Equal(t, expectedOutput, GetURIPath(testInput))
	}
}

func TestGetExecutablePath(t *testing.T) {
	t.Parallel()
	if _, err := GetExecutablePath(); err != nil {
		t.Errorf("Common GetExecutablePath. Error: %s", err)
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
		dir = filepath.Join(dir, "GoCryptoTrader", "TestFileASDFG")
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

func TestChangePermission(t *testing.T) {
	t.Parallel()
	testDir := filepath.Join(os.TempDir(), "TestFileASDFGHJ")
	switch runtime.GOOS {
	case "windows":
		err := ChangePermission("*")
		if err == nil {
			t.Fatal("expected an error on non-existent path")
		}
		err = os.Mkdir(testDir, 0o777)
		if err != nil {
			t.Fatalf("Mkdir failed. Err: %v", err)
		}
		err = ChangePermission(testDir)
		if err != nil {
			t.Fatalf("ChangePerm was unsuccessful. Err: %v", err)
		}
		_, err = os.Stat(testDir)
		if err != nil {
			t.Fatalf("os.Stat failed. Err: %v", err)
		}
		err = os.Remove(testDir)
		if err != nil {
			t.Fatalf("os.Remove failed. Err: %v", err)
		}
	default:
		err := ChangePermission("")
		if err == nil {
			t.Fatal("expected an error on non-existent path")
		}
		err = os.Mkdir(testDir, 0o777)
		if err != nil {
			t.Fatalf("Mkdir failed. Err: %v", err)
		}
		err = ChangePermission(testDir)
		if err != nil {
			t.Fatalf("ChangePerm was unsuccessful. Err: %v", err)
		}
		var a os.FileInfo
		a, err = os.Stat(testDir)
		if err != nil {
			t.Fatalf("os.Stat failed. Err: %v", err)
		}
		if a.Mode().Perm() != file.DefaultPermissionOctal {
			t.Fatalf("expected file permissions differ. expecting file.DefaultPermissionOctal got %#o", a.Mode().Perm())
		}
		err = os.Remove(testDir)
		if err != nil {
			t.Fatalf("os.Remove failed. Err: %v", err)
		}
	}
}

func TestAddPaddingOnUpperCase(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Supplied string
		Expected string
	}{
		{
			// empty
		},
		{
			Supplied: "ExpectedHTTPRainbow",
			Expected: "Expected HTTP Rainbow",
		},
		{
			Supplied: "SmellyCatSmellsBad",
			Expected: "Smelly Cat Smells Bad",
		},
		{
			Supplied: "Gronk",
			Expected: "Gronk",
		},
	}

	for x := range testCases {
		if received := AddPaddingOnUpperCase(testCases[x].Supplied); received != testCases[x].Expected {
			t.Fatalf("received '%v' but expected '%v'", received, testCases[x].Expected)
		}
	}
}

func TestErrors(t *testing.T) {
	t.Parallel()

	e1 := errors.New("inconsistent gravity")
	e2 := errors.New("barely marginal interest in your story")
	e3 := errors.New("error making dinner")
	e4 := errors.New("inconsistent gravy")
	e5 := errors.New("add vodka")

	// Nil tests
	assert.NoError(t, AppendError(nil, nil), "Append nil to nil should nil")
	assert.Same(t, AppendError(e1, nil), e1, "Append nil to e1 should e1")
	assert.Same(t, AppendError(nil, e2), e2, "Append e2 to nil should e2")

	// Vanila error tests
	err := AppendError(AppendError(AppendError(nil, e1), e2), e1)
	assert.ErrorContains(t, err, "inconsistent gravity, barely marginal interest in your story, inconsistent gravity", "Should format consistently")
	assert.ErrorIs(t, err, e1, "Should have inconsistent gravity")
	assert.ErrorIs(t, err, e2, "Should be bored by your witty tales")

	err = ExcludeError(err, e2)
	assert.ErrorIs(t, err, e1, "Should still be bored")
	assert.NotErrorIs(t, err, e2, "Should not be an e2")
	me, ok := err.(*multiError)
	if assert.True(t, ok, "Should be a multiError") {
		assert.Len(t, me.errs, 2, "Should only have 2 errors")
	}
	err = ExcludeError(err, e1)
	assert.NoError(t, err, "Error should be empty")
	err = ExcludeError(err, e1)
	assert.NoError(t, err, "Excluding a nil error should be okay")

	// Wrapped error tests
	err = fmt.Errorf("%w: %w", e3, fmt.Errorf("%w: %w", e4, e5))
	assert.ErrorIs(t, ExcludeError(err, e4), e3, "Excluding e4 should retain e3")
	assert.ErrorIs(t, ExcludeError(err, e4), e5, "Excluding e4 should retain the vanilla co-wrapped e5")
	assert.NotErrorIs(t, ExcludeError(err, e4), e4, "e4 should be excluded")
	assert.ErrorIs(t, ExcludeError(err, e5), e3, "Excluding e5 should retain e3")
	assert.ErrorIs(t, ExcludeError(err, e5), e4, "Excluding e5 should retain the vanilla co-wrapped e4")
	assert.NotErrorIs(t, ExcludeError(err, e5), e5, "e5 should be excluded")

	// Hybrid tests
	err = AppendError(fmt.Errorf("%w: %w", e4, e5), e3)
	assert.ErrorIs(t, ExcludeError(err, e4), e3, "Excluding e4 should retain e3")
	assert.ErrorIs(t, ExcludeError(err, e4), e5, "Excluding e4 should retain the vanilla co-wrapped e5")
	assert.NotErrorIs(t, ExcludeError(err, e4), e4, "e4 should be excluded")
	assert.ErrorIs(t, ExcludeError(err, e5), e3, "Excluding e5 should retain e3")
	assert.ErrorIs(t, ExcludeError(err, e5), e4, "Excluding e5 should retain the vanilla co-wrapped e4")
	assert.NotErrorIs(t, ExcludeError(err, e5), e5, "e4 should be excluded")

	// Formatting retention
	err = AppendError(e1, fmt.Errorf("%w: Run out of %q: %w", e3, "sausages", e5))
	assert.ErrorIs(t, err, e1, "Should be an e1")
	assert.ErrorIs(t, err, e3, "Should be an e3")
	assert.ErrorIs(t, err, e5, "Should be an e5")
	assert.ErrorContains(t, err, "sausages", "Should know about secret sausages")
}

func TestParseStartEndDate(t *testing.T) {
	t.Parallel()
	pt := time.Date(1999, 1, 1, 0, 0, 0, 0, time.Local)
	ft := time.Date(2222, 1, 1, 0, 0, 0, 0, time.Local)
	et := time.Date(2020, 1, 1, 1, 0, 0, 0, time.Local)
	nt := time.Time{}

	err := StartEndTimeCheck(nt, nt)
	assert.ErrorIs(t, err, ErrDateUnset)

	err = StartEndTimeCheck(et, nt)
	assert.ErrorIs(t, err, ErrDateUnset)

	err = StartEndTimeCheck(et, zeroValueUnix)
	assert.ErrorIs(t, err, ErrDateUnset)

	err = StartEndTimeCheck(zeroValueUnix, et)
	assert.ErrorIs(t, err, ErrDateUnset)

	err = StartEndTimeCheck(et, et)
	assert.ErrorIs(t, err, ErrStartEqualsEnd)

	err = StartEndTimeCheck(et, pt)
	assert.ErrorIs(t, err, ErrStartAfterEnd)

	err = StartEndTimeCheck(ft, ft.Add(time.Hour))
	assert.ErrorIs(t, err, ErrStartAfterTimeNow)

	err = StartEndTimeCheck(pt, et)
	assert.NoError(t, err)
}

func TestGetAssertError(t *testing.T) {
	err := GetTypeAssertError("*[]string", float64(0))
	if err.Error() != "type assert failure from float64 to *[]string" {
		t.Fatal(err)
	}

	err = GetTypeAssertError("<nil>", nil)
	if err.Error() != "type assert failure from <nil> to <nil>" {
		t.Fatal(err)
	}

	err = GetTypeAssertError("bruh", struct{}{})
	require.ErrorIs(t, err, ErrTypeAssertFailure)

	err = GetTypeAssertError("string", struct{}{})
	if err.Error() != "type assert failure from struct {} to string" {
		t.Errorf("unexpected error message: %v", err)
	}

	err = GetTypeAssertError("string", struct{}{}, "bidSize")
	if err.Error() != "type assert failure from struct {} to string for: bidSize" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMatchesEmailPattern(t *testing.T) {
	success := MatchesEmailPattern("someone semail")
	if success {
		t.Error("MatchesEmailPattern() unexpected test validation result")
	}
	success = MatchesEmailPattern("someone esemail@gmail")
	if success {
		t.Error("MatchesEmailPattern() unexpected test validation result")
	}
	success = MatchesEmailPattern("123@gmail")
	if !success {
		t.Error("MatchesEmailPattern() unexpected test validation result")
	}
	success = MatchesEmailPattern("someonesemail@email.com")
	if !success {
		t.Error("MatchesEmailPattern() unexpected test validation result")
	}
}

func TestGenerateRandomString(t *testing.T) {
	t.Parallel()
	sample, err := GenerateRandomString(5, NumberCharacters)
	if err != nil {
		t.Errorf("GenerateRandomString()  %v", err)
	}
	value, err := strconv.Atoi(sample)
	if len(sample) != 5 || err != nil || value < 0 {
		t.Error("GenerateRandomString() unexpected test validation result")
	}
	sample, err = GenerateRandomString(5)
	if err != nil {
		t.Errorf("GenerateRandomString()  %v", err)
	}
	values, err := strconv.ParseInt(sample, 10, 64)
	if len(sample) != 5 || err != nil || values < 0 {
		t.Error("GenerateRandomString() unexpected test validation result")
	}
	_, err = GenerateRandomString(1, "")
	if err == nil {
		t.Errorf("GenerateRandomString() expecting %s, but found %v", "invalid characters, character must not be empty", err)
	}
	sample, err = GenerateRandomString(0, "")
	if err != nil && !strings.Contains(err.Error(), "invalid length") {
		t.Errorf("GenerateRandomString()  %v", err)
	}
	if sample != "" {
		t.Error("GenerateRandomString() unexpected test validation result")
	}
}

func TestErrorCollector(t *testing.T) {
	var e ErrorCollector
	require.Panics(t, func() { e.Go(nil) }, "Go with nil function must panic")
	for i := range 4 {
		e.Go(func() error {
			if i%2 == 0 {
				return errors.New("collected error")
			}
			return nil
		})
	}
	v := e.Collect()
	errs, ok := v.(*multiError)
	require.True(t, ok, "Must return a multiError")
	assert.Len(t, errs.Unwrap(), 2, "Should have 2 errors")
	assert.NoError(t, e.Collect(), "should return nil when a previous collection emptied the errors")
}

// TestBatch ensures the Batch function does not regress into common behavioural faults if implementation changes
func TestBatch(t *testing.T) {
	s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	b := Batch(s, 3)
	require.Len(t, b, 4)
	assert.Len(t, b[0], 3)
	assert.Len(t, b[3], 1)

	b[0][0] = 42
	assert.Equal(t, 1, s[0], "Changing the batches should not change the source")

	require.NotPanics(t, func() { Batch(s, -1) }, "Must not panic on negative batch size")
	done := make(chan any, 1)
	go func() { done <- Batch(s, 0) }()
	require.Eventually(t, func() bool { return len(done) > 0 }, time.Second, time.Millisecond, "Batch 0 must not hang")

	for _, i := range []int{-1, 0, 50} {
		b = Batch(s, i)
		require.Lenf(t, b, 1, "A batch size of %v must produce a single batch", i)
		assert.Lenf(t, b[0], len(s), "A batch size of %v should produce a single batch", i)
	}
}

type A int

func (a A) String() string {
	return strconv.Itoa(int(a))
}

func TestSortStrings(t *testing.T) {
	assert.Equal(t, []A{1, 2, 5, 6}, SortStrings([]A{6, 2, 5, 1}))
}

func TestCounter(t *testing.T) {
	t.Parallel()
	c := Counter{}
	c.n.Store(-5)
	require.Equal(t, int64(1), c.IncrementAndGet(), "Adding to a negative Counter must reset to zero and then increment")
	require.Equal(t, int64(2), c.IncrementAndGet())
}

// 683185328	         1.787 ns/op	       0 B/op	       0 allocs/op
func BenchmarkCounter(b *testing.B) {
	c := Counter{}
	for b.Loop() {
		c.IncrementAndGet()
	}
}

func TestNilGuard(t *testing.T) {
	t.Parallel()
	err := NilGuard((*int)(nil))
	assert.ErrorIs(t, err, ErrNilPointer)
	assert.ErrorContains(t, err, "*int")

	s := "normal input"
	err = NilGuard(&s, 2, &[]int{4, 5, 6}, []int{1, 2, 3}, new(A))
	assert.NoError(t, err)

	err = NilGuard(&s, nil, (*int)(nil))
	assert.ErrorIs(t, err, ErrNilPointer)
	assert.ErrorContains(t, err, "*int")
	var mErr *multiError
	require.ErrorAs(t, err, &mErr, "err must be a multiError")
	assert.Len(t, mErr.Unwrap(), 2, "Should get 2 errors back")

	assert.ErrorIs(t, NilGuard(nil), ErrNilPointer, "Unusual input of an untyped nil should still error correctly")

	err = NilGuard()
	require.NoError(t, err, "NilGuard with no arguments must not error")
}

func TestSetIfZero(t *testing.T) {
	t.Parallel()
	s := "hello"
	changed := SetIfZero(&s, "world")
	assert.False(t, changed, "SetIfZero should not change a non-zero value")
	assert.Equal(t, "hello", s, "SetIfZero should not change a non-zero value")
	s = ""
	changed = SetIfZero(&s, "world")
	assert.True(t, changed, "SetIfZero should change a zero value")
	assert.Equal(t, "world", s, "SetIfZero should change a zero value")
}

func TestContextFunctions(t *testing.T) {
	t.Parallel()

	type key string
	const k1 key = "key1"
	const k2 key = "key2"
	const k3 key = "key3"

	RegisterContextKey(k1)
	RegisterContextKey(k2)

	ctx := context.WithValue(context.Background(), k1, "value1")
	ctx = context.WithValue(ctx, k2, "value2")
	ctx = context.WithValue(ctx, k3, "value3") // Not registered

	frozen := FreezeContext(ctx)

	assert.Equal(t, "value1", frozen[k1], "should have captured k1")
	assert.Equal(t, "value2", frozen[k2], "should have captured k2")
	assert.Zero(t, frozen[k3], "k3 should not be captured")

	thawed := ThawContext(frozen)
	assert.Equal(t, "value1", thawed.Value(k1), "should have k1 after thaw")
	assert.Equal(t, "value2", thawed.Value(k2), "should have k2 after thaw")
	assert.Nil(t, thawed.Value(k3), "Thawed context should not have k3")

	ctx2 := context.WithValue(context.Background(), k3, "value3_new")
	merged := MergeContext(ctx2, frozen)
	assert.Equal(t, "value1", merged.Value(k1), "should have k1 from frozen")
	assert.Equal(t, "value2", merged.Value(k2), "should have k2 from frozen")
	assert.Equal(t, "value3_new", merged.Value(k3), "should have k3 from parent")
}
