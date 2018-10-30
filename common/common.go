package common

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	log "github.com/thrasher-/gocryptotrader/logger"
)

// Vars for common.go operations
var (
	HTTPClient    *http.Client
	HTTPUserAgent string

	// ErrNotYetImplemented defines a common error across the code base that
	// alerts of a function that has not been completed or tied into main code
	ErrNotYetImplemented = errors.New("not yet implemented")

	// ErrFunctionNotSupported defines a standardised error for an unsupported
	// wrapper function by an API
	ErrFunctionNotSupported = errors.New("unsupported wrapper function")
)

// Const declarations for common.go operations
const (
	SatoshisPerBTC = 100000000
	SatoshisPerLTC = 100000000
	WeiPerEther    = 1000000000000000000
)

func initialiseHTTPClient() {
	// If the HTTPClient isn't set, start a new client with a default timeout of 15 seconds
	if HTTPClient == nil {
		HTTPClient = NewHTTPClientWithTimeout(time.Second * 15)
	}
}

// NewHTTPClientWithTimeout initialises a new HTTP client with the specified
// timeout duration
func NewHTTPClientWithTimeout(t time.Duration) *http.Client {
	h := &http.Client{Timeout: t}
	return h
}

// StringSliceDifference concatenates slices together based on its index and
// returns an individual string array
func StringSliceDifference(slice1, slice2 []string) []string {
	var diff []string
	for i := 0; i < 2; i++ {
		for _, s1 := range slice1 {
			found := false
			for _, s2 := range slice2 {
				if s1 == s2 {
					found = true
					break
				}
			}
			if !found {
				diff = append(diff, s1)
			}
		}
		if i == 0 {
			slice1, slice2 = slice2, slice1
		}
	}
	return diff
}

// StringContains checks a substring if it contains your input then returns a
// bool
func StringContains(input, substring string) bool {
	return strings.Contains(input, substring)
}

// StringDataContains checks the substring array with an input and returns a bool
func StringDataContains(haystack []string, needle string) bool {
	data := strings.Join(haystack, ",")
	return strings.Contains(data, needle)
}

// StringDataCompare data checks the substring array with an input and returns a bool
func StringDataCompare(haystack []string, needle string) bool {
	for x := range haystack {
		if haystack[x] == needle {
			return true
		}
	}
	return false
}

// StringDataCompareInsensitive data checks the substring array with an input and returns
// a bool irrespective of lower or upper case strings
func StringDataCompareInsensitive(haystack []string, needle string) bool {
	for x := range haystack {
		if strings.EqualFold(haystack[x], needle) {
			return true
		}
	}
	return false
}

// StringDataContainsInsensitive checks the substring array with an input and returns
// a bool irrespective of lower or upper case strings
func StringDataContainsInsensitive(haystack []string, needle string) bool {
	for _, data := range haystack {
		if strings.Contains(StringToUpper(data), StringToUpper(needle)) {
			return true
		}
	}
	return false
}

// JoinStrings joins an array together with the required separator and returns
// it as a string
func JoinStrings(input []string, separator string) string {
	return strings.Join(input, separator)
}

// SplitStrings splits blocks of strings from string into a string array using
// a separator ie "," or "_"
func SplitStrings(input, separator string) []string {
	return strings.Split(input, separator)
}

// TrimString trims unwanted prefixes or postfixes
func TrimString(input, cutset string) string {
	return strings.Trim(input, cutset)
}

// ReplaceString replaces a string with another
func ReplaceString(input, old, newStr string, n int) string {
	return strings.Replace(input, old, newStr, n)
}

// StringToUpper changes strings to uppercase
func StringToUpper(input string) string {
	return strings.ToUpper(input)
}

// StringToLower changes strings to lowercase
func StringToLower(input string) string {
	return strings.ToLower(input)
}

// IsEnabled takes in a boolean param  and returns a string if it is enabled
// or disabled
func IsEnabled(isEnabled bool) string {
	if isEnabled {
		return "Enabled"
	}
	return "Disabled"
}

// IsValidCryptoAddress validates your cryptocurrency address string using the
// regexp package // Validation issues occurring because "3" is contained in
// litecoin and Bitcoin addresses - non-fatal
func IsValidCryptoAddress(address, crypto string) (bool, error) {
	switch StringToLower(crypto) {
	case "btc":
		return regexp.MatchString("^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$", address)
	case "ltc":
		return regexp.MatchString("^[L3M][a-km-zA-HJ-NP-Z1-9]{25,34}$", address)
	case "eth":
		return regexp.MatchString("^0x[a-km-z0-9]{40}$", address)
	default:
		return false, errors.New("invalid crypto currency")
	}
}

// YesOrNo returns a boolean variable to check if input is "y" or "yes"
func YesOrNo(input string) bool {
	if StringToLower(input) == "y" || StringToLower(input) == "yes" {
		return true
	}
	return false
}

// SendHTTPRequest sends a request using the http package and returns a response
// as a string and an error
func SendHTTPRequest(method, urlPath string, headers map[string]string, body io.Reader) (string, error) {
	result := strings.ToUpper(method)

	if result != http.MethodPost && result != http.MethodGet && result != http.MethodDelete {
		return "", errors.New("invalid HTTP method specified")
	}

	initialiseHTTPClient()

	req, err := http.NewRequest(method, urlPath, body)
	if err != nil {
		return "", err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	if HTTPUserAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Add("User-Agent", HTTPUserAgent)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", err
	}

	contents, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return "", err
	}

	return string(contents), nil
}

// SendHTTPGetRequest sends a simple get request using a url string & JSON
// decodes the response into a struct pointer you have supplied. Returns an error
// on failure.
func SendHTTPGetRequest(urlPath string, jsonDecode, isVerbose bool, result interface{}) error {
	if isVerbose {
		log.Debugf("Raw URL: %s", urlPath)
	}

	initialiseHTTPClient()

	res, err := HTTPClient.Get(urlPath)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("common.SendHTTPGetRequest() error: HTTP status code %d", res.StatusCode)
	}

	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if isVerbose {
		log.Debugf("Raw Resp: %s", string(contents))
	}

	defer res.Body.Close()

	if jsonDecode {
		err := JSONDecode(contents, result)
		if err != nil {
			return err
		}
	}

	return nil
}

// JSONEncode encodes structure data into JSON
func JSONEncode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// JSONDecode decodes JSON data into a structure
func JSONDecode(data []byte, to interface{}) error {
	if !StringContains(reflect.ValueOf(to).Type().String(), "*") {
		return errors.New("json decode error - memory address not supplied")
	}
	return json.Unmarshal(data, to)
}

// EncodeURLValues concatenates url values onto a url string and returns a
// string
func EncodeURLValues(urlPath string, values url.Values) string {
	u := urlPath
	if len(values) > 0 {
		u += "?" + values.Encode()
	}
	return u
}

// ExtractHost returns the hostname out of a string
func ExtractHost(address string) string {
	host := SplitStrings(address, ":")[0]
	if host == "" {
		return "localhost"
	}
	return host
}

// ExtractPort returns the port name out of a string
func ExtractPort(host string) int {
	portStr := SplitStrings(host, ":")[1]
	port, _ := strconv.Atoi(portStr)
	return port
}

// OutputCSV dumps data into a file as comma-separated values
func OutputCSV(filePath string, data [][]string) error {
	_, err := ReadFile(filePath)
	if err != nil {
		errTwo := WriteFile(filePath, nil)
		if errTwo != nil {
			return errTwo
		}
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	writer := csv.NewWriter(file)

	err = writer.WriteAll(data)
	if err != nil {
		return err
	}

	writer.Flush()
	file.Close()
	return nil
}

// UnixTimestampToTime returns time.time
func UnixTimestampToTime(timeint64 int64) time.Time {
	return time.Unix(timeint64, 0)
}

// UnixTimestampStrToTime returns a time.time and an error
func UnixTimestampStrToTime(timeStr string) (time.Time, error) {
	i, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(i, 0), nil
}

// ReadFile reads a file and returns read data as byte array.
func ReadFile(file string) ([]byte, error) {
	return ioutil.ReadFile(file)
}

// WriteFile writes selected data to a file and returns an error
func WriteFile(file string, data []byte) error {
	return ioutil.WriteFile(file, data, 0644)
}

// RemoveFile removes a file
func RemoveFile(file string) error {
	return os.Remove(file)
}

// GetURIPath returns the path of a URL given a URI
func GetURIPath(uri string) string {
	urip, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	if urip.RawQuery != "" {
		return fmt.Sprintf("%s?%s", urip.Path, urip.RawQuery)
	}
	return urip.Path
}

// GetExecutablePath returns the executables launch path
func GetExecutablePath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(ex), nil
}

// GetOSPathSlash returns the slash used by the operating systems
// file system
func GetOSPathSlash() string {
	if runtime.GOOS == "windows" {
		return "\\"
	}
	return "/"
}

// UnixMillis converts a UnixNano timestamp to milliseconds
func UnixMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// RecvWindow converts a supplied time.Duration to milliseconds
func RecvWindow(d time.Duration) int64 {
	return int64(d) / int64(time.Millisecond)
}

// FloatFromString format
func FloatFromString(raw interface{}) (float64, error) {
	str, ok := raw.(string)
	if !ok {
		return 0, fmt.Errorf("unable to parse, value not string: %T", raw)
	}
	flt, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("could not convert value: %s Error: %s", str, err)
	}
	return flt, nil
}

// IntFromString format
func IntFromString(raw interface{}) (int, error) {
	str, ok := raw.(string)
	if !ok {
		return 0, fmt.Errorf("unable to parse, value not string: %T", raw)
	}
	n, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("unable to parse as int: %T", raw)
	}
	return n, nil
}

// Int64FromString format
func Int64FromString(raw interface{}) (int64, error) {
	str, ok := raw.(string)
	if !ok {
		return 0, fmt.Errorf("unable to parse, value not string: %T", raw)
	}
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse as int64: %T", raw)
	}
	return n, nil
}

// TimeFromUnixTimestampFloat format
func TimeFromUnixTimestampFloat(raw interface{}) (time.Time, error) {
	ts, ok := raw.(float64)
	if !ok {
		return time.Time{}, fmt.Errorf("unable to parse, value not float64: %T", raw)
	}
	return time.Unix(0, int64(ts)*int64(time.Millisecond)), nil
}

// GetDefaultDataDir returns the default data directory
// Windows - C:\Users\%USER%\AppData\Roaming\GoCryptoTrader
// Linux/Unix or OSX - $HOME/.gocryptotrader
func GetDefaultDataDir(env string) string {
	if env == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "GoCryptoTrader")
	}

	usr, err := user.Current()
	if err == nil {
		return filepath.Join(usr.HomeDir, ".gocryptotrader")
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		log.Warn("Environment variable unset, defaulting to current directory")
		dir = "."
	}
	return filepath.Join(dir, ".gocryptotrader")
}

// CreateDir creates a directory based on the supplied parameter
func CreateDir(dir string) error {
	_, err := os.Stat(dir)
	if !os.IsNotExist(err) {
		return nil
	}

	log.Warnf("Directory %s does not exist.. creating.", dir)
	return os.MkdirAll(dir, 0770)
}

// ChangePerm lists all the directories and files in an array
func ChangePerm(directory string) error {
	return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().Perm() != 0770 {
			return os.Chmod(path, 0770)
		}
		return nil
	})
}
