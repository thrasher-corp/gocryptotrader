package common

import (
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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// SimpleTimeFormat a common, but non-implemented time format in golang
	SimpleTimeFormat = "2006-01-02 15:04:05"
	// SimpleTimeFormatWithTimezone a common, but non-implemented time format in golang
	SimpleTimeFormatWithTimezone = "2006-01-02 15:04:05 MST"
	// GctExt is the extension for GCT Tengo script files
	GctExt = ".gct"
)

// Vars for common.go operations
var (
	HTTPClient    *http.Client
	HTTPUserAgent string
	m             sync.Mutex
	// ErrNotYetImplemented defines a common error across the code base that
	// alerts of a function that has not been completed or tied into main code
	ErrNotYetImplemented = errors.New("not yet implemented")
	// ErrFunctionNotSupported defines a standardised error for an unsupported
	// wrapper function by an API
	ErrFunctionNotSupported  = errors.New("unsupported wrapper function")
	errInvalidCryptoCurrency = errors.New("invalid crypto currency")
	// ErrDateUnset is an error for start end check calculations
	ErrDateUnset = errors.New("date unset")
	// ErrStartAfterEnd is an error for start end check calculations
	ErrStartAfterEnd = errors.New("start date after end date")
	// ErrStartEqualsEnd is an error for start end check calculations
	ErrStartEqualsEnd = errors.New("start date equals end date")
	// ErrStartAfterTimeNow is an error for start end check calculations
	ErrStartAfterTimeNow = errors.New("start date is after current time")
)

func initialiseHTTPClient() {
	m.Lock()
	// If the HTTPClient isn't set, start a new client with a default timeout of 15 seconds
	if HTTPClient == nil {
		HTTPClient = NewHTTPClientWithTimeout(time.Second * 15)
	}
	m.Unlock()
}

// SetHTTPClientWithTimeout protects the setting of the
// global HTTPClient
func SetHTTPClientWithTimeout(t time.Duration) {
	m.Lock()
	defer m.Unlock()
	HTTPClient = NewHTTPClientWithTimeout(t)
}

// NewHTTPClientWithTimeout initialises a new HTTP client and its underlying
// transport IdleConnTimeout with the specified timeout duration
func NewHTTPClientWithTimeout(t time.Duration) *http.Client {
	tr := &http.Transport{
		// Added IdleConnTimeout to reduce the time of idle connections which
		// could potentially slow macOS reconnection when there is a sudden
		// network disconnection/issue
		IdleConnTimeout: t,
		Proxy:           http.ProxyFromEnvironment,
	}
	h := &http.Client{
		Transport: tr,
		Timeout:   t}
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
		if strings.Contains(strings.ToUpper(data), strings.ToUpper(needle)) {
			return true
		}
	}
	return false
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
	switch strings.ToLower(crypto) {
	case "btc":
		return regexp.MatchString("^(bc1|[13])[a-zA-HJ-NP-Z0-9]{25,90}$", address)
	case "ltc":
		return regexp.MatchString("^[L3M][a-km-zA-HJ-NP-Z1-9]{25,34}$", address)
	case "eth":
		return regexp.MatchString("^0x[a-km-z0-9]{40}$", address)
	default:
		return false, fmt.Errorf("%w %s", errInvalidCryptoCurrency, crypto)
	}
}

// YesOrNo returns a boolean variable to check if input is "y" or "yes"
func YesOrNo(input string) bool {
	if strings.EqualFold(input, "y") || strings.EqualFold(input, "yes") {
		return true
	}
	return false
}

// SendHTTPRequest sends a request using the http package and returns a response
// as a string and an error
func SendHTTPRequest(method, urlPath string, headers map[string]string, body io.Reader) (string, error) {
	result := strings.ToUpper(method)

	if result != http.MethodOptions && result != http.MethodGet &&
		result != http.MethodHead && result != http.MethodPost &&
		result != http.MethodPut && result != http.MethodDelete &&
		result != http.MethodTrace && result != http.MethodConnect {
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

	m.Lock()
	resp, err := HTTPClient.Do(req)
	if err != nil {
		m.Unlock()
		return "", err
	}
	m.Unlock()

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
		log.Debugf(log.Global, "Raw URL: %s\n", urlPath)
	}

	initialiseHTTPClient()

	m.Lock()
	res, err := HTTPClient.Get(urlPath)
	if err != nil {
		m.Unlock()
		return err
	}
	m.Unlock()

	if res.StatusCode != 200 {
		return fmt.Errorf("common.SendHTTPGetRequest() error: HTTP status code %d", res.StatusCode)
	}

	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if isVerbose {
		log.Debugf(log.Global, "Raw Resp: %s\n", string(contents))
	}

	defer res.Body.Close()

	if jsonDecode {
		err := json.Unmarshal(contents, result)
		if err != nil {
			return err
		}
	}

	return nil
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
	host := strings.Split(address, ":")[0]
	if host == "" {
		return "localhost"
	}
	return host
}

// ExtractPort returns the port name out of a string
func ExtractPort(host string) int {
	portStrs := strings.Split(host, ":")
	if len(portStrs) == 1 {
		return 80
	}
	port, _ := strconv.Atoi(portStrs[1])
	return port
}

// GetURIPath returns the path of a URL given a URI
func GetURIPath(uri string) string {
	urip, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	if urip.RawQuery != "" {
		return urip.Path + "?" + urip.RawQuery
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
		log.Warnln(log.Global, "Environment variable unset, defaulting to current directory")
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

	log.Warnf(log.Global, "Directory %s does not exist.. creating.\n", dir)
	return os.MkdirAll(dir, 0770)
}

// ChangePermission lists all the directories and files in an array
func ChangePermission(directory string) error {
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

// SplitStringSliceByLimit splits a slice of strings into slices by input limit and returns a slice of slice of strings
func SplitStringSliceByLimit(in []string, limit uint) [][]string {
	var stringSlice []string
	sliceSlice := make([][]string, 0, len(in)/int(limit)+1)
	for len(in) >= int(limit) {
		stringSlice, in = in[:limit], in[limit:]
		sliceSlice = append(sliceSlice, stringSlice)
	}
	if len(in) > 0 {
		sliceSlice = append(sliceSlice, in)
	}
	return sliceSlice
}

// InArray checks if _val_ belongs to _array_
func InArray(val, array interface{}) (exists bool, index int) {
	exists = false
	index = -1
	if array == nil {
		return
	}
	switch reflect.TypeOf(array).Kind() {
	case reflect.Array, reflect.Slice:
		s := reflect.ValueOf(array)
		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(val, s.Index(i).Interface()) {
				index = i
				exists = true
				return
			}
		}
	}
	return
}

// Errors defines multiple errors
type Errors []error

// Error implements error interface
func (e Errors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var r string
	for i := range e {
		r += e[i].Error() + ", "
	}
	return r[:len(r)-2]
}

// StartEndTimeCheck provides some basic checks which occur
// frequently in the codebase
func StartEndTimeCheck(start, end time.Time) error {
	if start.IsZero() {
		return fmt.Errorf("start %w", ErrDateUnset)
	}
	if end.IsZero() {
		return fmt.Errorf("end %w", ErrDateUnset)
	}
	if start.After(time.Now()) {
		return ErrStartAfterTimeNow
	}
	if start.After(end) {
		return ErrStartAfterEnd
	}
	if start.Equal(end) {
		return ErrStartEqualsEnd
	}

	return nil
}
