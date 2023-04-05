package common

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
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
	"unicode"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// SimpleTimeFormat a common, but non-implemented time format in golang
	SimpleTimeFormat = "2006-01-02 15:04:05"
	// SimpleTimeFormatWithTimezone a common, but non-implemented time format in golang
	SimpleTimeFormatWithTimezone = "2006-01-02 15:04:05 MST"
	// GctExt is the extension for GCT Tengo script files
	GctExt         = ".gct"
	defaultTimeout = time.Second * 15
)

// Strings representing the full lower, upper case English character alphabet and base-10 numbers for generating a random string.
const (
	SmallLetters     = "abcdefghijklmnopqrstuvwxyz"
	CapitalLetters   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	NumberCharacters = "0123456789"
)

var (
	// emailRX represents email address matching pattern
	emailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// Vars for common.go operations
var (
	_HTTPClient    *http.Client
	_HTTPUserAgent string
	m              sync.RWMutex
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
	// ErrNilPointer defines an error for a nil pointer
	ErrNilPointer = errors.New("nil pointer")
	// ErrCannotCalculateOffline is returned when a request wishes to calculate
	// something offline, but has an online requirement
	ErrCannotCalculateOffline = errors.New("cannot calculate offline")
	// ErrNoResponse is returned when a response has no entries/is empty
	// when one is expected
	ErrNoResponse = errors.New("no response")

	errCannotSetInvalidTimeout = errors.New("cannot set new HTTP client with timeout that is equal or less than 0")
	errUserAgentInvalid        = errors.New("cannot set invalid user agent")
	errHTTPClientInvalid       = errors.New("custom http client cannot be nil")

	zeroValueUnix = time.Unix(0, 0)
	// ErrTypeAssertFailure defines an error when type assertion fails
	ErrTypeAssertFailure = errors.New("type assert failure")
)

// MatchesEmailPattern ensures that the string is an email address by regexp check
func MatchesEmailPattern(value string) bool {
	if len(value) < 3 || len(value) > 254 {
		return false
	}
	return emailRX.MatchString(value)
}

// SetHTTPClientWithTimeout sets a new *http.Client with different timeout
// settings
func SetHTTPClientWithTimeout(t time.Duration) error {
	if t <= 0 {
		return errCannotSetInvalidTimeout
	}
	m.Lock()
	_HTTPClient = NewHTTPClientWithTimeout(t)
	m.Unlock()
	return nil
}

// SetHTTPUserAgent sets the user agent which will be used for all common HTTP
// requests.
func SetHTTPUserAgent(agent string) error {
	if agent == "" {
		return errUserAgentInvalid
	}
	m.Lock()
	_HTTPUserAgent = agent
	m.Unlock()
	return nil
}

// SetHTTPClient sets a custom HTTP client.
func SetHTTPClient(client *http.Client) error {
	if client == nil {
		return errHTTPClientInvalid
	}
	m.Lock()
	_HTTPClient = client
	m.Unlock()
	return nil
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

// SendHTTPRequest sends a request using the http package and returns the body
// contents
func SendHTTPRequest(ctx context.Context, method, urlPath string, headers map[string]string, body io.Reader, verbose bool) ([]byte, error) {
	method = strings.ToUpper(method)

	if method != http.MethodOptions && method != http.MethodGet &&
		method != http.MethodHead && method != http.MethodPost &&
		method != http.MethodPut && method != http.MethodDelete &&
		method != http.MethodTrace && method != http.MethodConnect {
		return nil, errors.New("invalid HTTP method specified")
	}

	req, err := http.NewRequestWithContext(ctx, method, urlPath, body)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	if verbose {
		log.Debugf(log.Global, "Request path: %s", urlPath)
		for k, d := range req.Header {
			log.Debugf(log.Global, "Request header [%s]: %s", k, d)
		}
		log.Debugf(log.Global, "Request type: %s", method)
		if body != nil {
			log.Debugf(log.Global, "Request body: %v", body)
		}
	}

	m.RLock()
	if _HTTPUserAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Add("User-Agent", _HTTPUserAgent)
	}

	if _HTTPClient == nil {
		m.RUnlock()
		m.Lock()
		// Set *http.Client with default timeout if not populated.
		_HTTPClient = NewHTTPClientWithTimeout(defaultTimeout)
		m.Unlock()
		m.RLock()
	}

	resp, err := _HTTPClient.Do(req)
	m.RUnlock()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	contents, err := io.ReadAll(resp.Body)

	if verbose {
		log.Debugf(log.Global, "HTTP status: %s, Code: %v",
			resp.Status,
			resp.StatusCode)
		log.Debugf(log.Global, "Raw response: %s", string(contents))
	}

	return contents, err
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
	host, _, _ := net.SplitHostPort(address)
	if host == "" {
		return "localhost"
	}
	return host
}

// ExtractPort returns the port name out of a string
func ExtractPort(host string) int {
	_, port, _ := net.SplitHostPort(host)
	if port == "" {
		return 80
	}
	portInt, _ := strconv.Atoi(port)
	return portInt
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
	return os.MkdirAll(dir, file.DefaultPermissionOctal)
}

// ChangePermission lists all the directories and files in an array
func ChangePermission(directory string) error {
	return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().Perm() != file.DefaultPermissionOctal {
			return os.Chmod(path, file.DefaultPermissionOctal)
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

// AddPaddingOnUpperCase adds padding to a string when detecting an upper case letter. If
// there are multiple upper case items like `ThisIsHTTPExample`, it will only
// pad between like this `This Is HTTP Example`.
func AddPaddingOnUpperCase(s string) string {
	if s == "" {
		return ""
	}
	var result []string
	left := 0
	for x := 0; x < len(s); x++ {
		if x == 0 {
			continue
		}

		if unicode.IsUpper(rune(s[x])) {
			if !unicode.IsUpper(rune(s[x-1])) {
				result = append(result, s[left:x])
				left = x
			}
		} else if x > 1 && unicode.IsUpper(rune(s[x-1])) {
			if s[left:x-1] == "" {
				continue
			}
			result = append(result, s[left:x-1])
			left = x - 1
		}
	}
	result = append(result, s[left:])
	return strings.Join(result, " ")
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

// multiError holds all the errors as a slice, this is unexported, so it forces
// inbuilt error handling.
type multiError struct {
	loadedErrors []error
	offset       *int
}

// AppendError appends error in a more idiomatic way. This can start out as a
// standard error e.g. err := errors.New("random error")
// err = AppendError(err, errors.New("another random error"))
func AppendError(original, incoming error) error {
	errSliceP, ok := original.(*multiError)
	if ok {
		errSliceP.offset = nil
	}
	if incoming == nil {
		return original // Skip append - continue as normal.
	}
	if !ok {
		// This assumes that a standard error is passed in and we can want to
		// track it and add additional errors.
		errSliceP = &multiError{}
		if original != nil {
			errSliceP.loadedErrors = append(errSliceP.loadedErrors, original)
		}
	}
	if incomingSlice, ok := incoming.(*multiError); ok {
		// Join slices if needed.
		errSliceP.loadedErrors = append(errSliceP.loadedErrors, incomingSlice.loadedErrors...)
	} else {
		errSliceP.loadedErrors = append(errSliceP.loadedErrors, incoming)
	}
	return errSliceP
}

// Error displays all errors comma separated, if unwrapped has been called and
// has not been reset will display the individual error
func (e *multiError) Error() string {
	if e.offset != nil {
		return e.loadedErrors[*e.offset].Error()
	}
	allErrors := make([]string, len(e.loadedErrors))
	for x := range e.loadedErrors {
		allErrors[x] = e.loadedErrors[x].Error()
	}
	return strings.Join(allErrors, ", ")
}

// Unwrap increments the offset so errors.Is() can be called to its individual
// error for correct matching.
func (e *multiError) Unwrap() error {
	if e.offset == nil {
		e.offset = new(int)
	} else {
		*e.offset++
	}
	if *e.offset == len(e.loadedErrors) {
		e.offset = nil
		return nil // Force errors.Is package to return false.
	}
	return e
}

// Is checks to see if the errors match. It calls package errors.Is() so that
// we can keep fmt.Errorf() trimmings. This is called in errors package at
// interface assertion err.(interface{ Is(error) bool }).
func (e *multiError) Is(incoming error) bool {
	if e.offset != nil && errors.Is(e.loadedErrors[*e.offset], incoming) {
		e.offset = nil
		return true
	}
	return false
}

// StartEndTimeCheck provides some basic checks which occur
// frequently in the codebase
func StartEndTimeCheck(start, end time.Time) error {
	if start.IsZero() || start.Equal(zeroValueUnix) {
		return fmt.Errorf("start %w", ErrDateUnset)
	}
	if end.IsZero() || end.Equal(zeroValueUnix) {
		return fmt.Errorf("end %w", ErrDateUnset)
	}
	if start.After(end) {
		return ErrStartAfterEnd
	}
	if start.Equal(end) {
		return ErrStartEqualsEnd
	}
	if start.After(time.Now()) {
		return ErrStartAfterTimeNow
	}

	return nil
}

// GenerateRandomString generates a random string provided a length and list of Character types { SmallLetters, CapitalLetters, NumberCharacters}.
// if no characters are provided, the function uses a NumberCharacters(string of numeric characters).
func GenerateRandomString(length uint, characters ...string) (string, error) {
	if length == 0 {
		return "", errors.New("invalid length, length must be non-zero positive integer")
	}
	b := make([]byte, length)
	chars := strings.Replace(strings.Join(characters, ""), " ", "", -1)
	if chars == "" && len(characters) != 0 {
		return "", errors.New("invalid characters, character must not be empty")
	} else if chars == "" {
		chars = NumberCharacters
	}
	for i := range b {
		nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		n := nBig.Int64()
		b[i] = chars[n]
	}
	return string(b), nil
}

// GetAssertError returns additional information for when an assertion failure
// occurs.
func GetAssertError(required string, received interface{}) error {
	return fmt.Errorf("%w from %T to %s", ErrTypeAssertFailure, received, required)
}
