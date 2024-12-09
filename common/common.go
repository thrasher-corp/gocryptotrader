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
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// SimpleTimeFormatWithTimezone a common, but non-implemented time format in golang
	SimpleTimeFormatWithTimezone = time.DateTime + " MST"
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
	zeroValueUnix  = time.Unix(0, 0)
)

// Public common Errors
var (
	ErrNotYetImplemented      = errors.New("not yet implemented")
	ErrFunctionNotSupported   = errors.New("unsupported wrapper function")
	errInvalidCryptoCurrency  = errors.New("invalid crypto currency")
	ErrDateUnset              = errors.New("date unset")
	ErrStartAfterEnd          = errors.New("start date after end date")
	ErrStartEqualsEnd         = errors.New("start date equals end date")
	ErrStartAfterTimeNow      = errors.New("start date is after current time")
	ErrNilPointer             = errors.New("nil pointer")
	ErrEmptyParams            = errors.New("empty parameters")
	ErrCannotCalculateOffline = errors.New("cannot calculate offline, unsupported")
	ErrNoResponse             = errors.New("no response")
	ErrTypeAssertFailure      = errors.New("type assert failure")
	ErrUnknownError           = errors.New("unknown error")
	ErrGettingField           = errors.New("error getting field")
	ErrSettingField           = errors.New("error setting field")
)

var (
	errCannotSetInvalidTimeout = errors.New("cannot set new HTTP client with timeout that is equal or less than 0")
	errUserAgentInvalid        = errors.New("cannot set invalid user agent")
	errHTTPClientInvalid       = errors.New("custom http client cannot be nil")
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

// SliceDifference returns the elements that are in slice1 or slice2 but not in both
func SliceDifference[T comparable](slice1, slice2 []T) []T {
	diff := make([]T, 0, len(slice1)+len(slice2))
	for x := range slice1 {
		if !slices.Contains(slice2, slice1[x]) {
			diff = append(diff, slice1[x])
			continue
		}
	}
	for x := range slice2 {
		if !slices.Contains(slice1, slice2[x]) {
			diff = append(diff, slice2[x])
		}
	}
	return slices.Clip(diff)
}

// StringSliceContains returns whether case sensitive needle is contained within haystack
func StringSliceContains(haystack []string, needle string) bool {
	return slices.ContainsFunc(haystack, func(s string) bool {
		return strings.Contains(s, needle)
	})
}

// StringSliceCompareInsensitive returns whether case insensitive needle exists within haystack
func StringSliceCompareInsensitive(haystack []string, needle string) bool {
	return slices.ContainsFunc(haystack, func(s string) bool {
		return strings.EqualFold(s, needle)
	})
}

// StringSliceContainsInsensitive returns whether case insensitive needle is contained within haystack
func StringSliceContainsInsensitive(haystack []string, needle string) bool {
	needleUpper := strings.ToUpper(needle)
	return slices.ContainsFunc(haystack, func(s string) bool {
		return strings.Contains(strings.ToUpper(s), needleUpper)
	})
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

// AddPaddingOnUpperCase adds padding to a string when detecting an upper case letter. If
// there are multiple upper case items like `ThisIsHTTPExample`, it will only
// pad between like this `This Is HTTP Example`.
func AddPaddingOnUpperCase(s string) string {
	if s == "" {
		return ""
	}
	var result []string
	left := 0
	for x := range s {
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

// fmtError holds a formatted msg and the errors which formatted it
type fmtError struct {
	errs []error
	msg  string
}

// multiError holds errors as a slice
type multiError struct {
	errs []error
}

type unwrappable interface {
	Unwrap() []error
	Error() string
}

// AppendError appends an error to a list of exesting errors
// Either argument may be:
// * A vanilla error
// * An error implementing Unwrap() []error e.g. fmt.Errorf("%w: %w")
// * nil
// The result will be an error which may be a multiError if multipleErrors were found
func AppendError(original, incoming error) error {
	if incoming == nil {
		return original
	}
	if original == nil {
		return incoming
	}
	if u, ok := incoming.(unwrappable); ok {
		incoming = &fmtError{
			errs: u.Unwrap(),
			msg:  u.Error(),
		}
	}
	switch v := original.(type) {
	case *multiError:
		v.errs = append(v.errs, incoming)
		return v
	case unwrappable:
		original = &fmtError{
			errs: v.Unwrap(),
			msg:  v.Error(),
		}
	}
	return &multiError{
		errs: append([]error{original}, incoming),
	}
}

func (e *fmtError) Error() string {
	return e.msg
}

func (e *fmtError) Unwrap() []error {
	return e.errs
}

// Error displays all errors comma separated
func (e *multiError) Error() string {
	allErrors := make([]string, len(e.errs))
	for x := range e.errs {
		allErrors[x] = e.errs[x].Error()
	}
	return strings.Join(allErrors, ", ")
}

// Unwrap returns all of the errors in the multiError
func (e *multiError) Unwrap() []error {
	errs := make([]error, 0, len(e.errs))
	for _, e := range e.errs {
		switch v := e.(type) {
		case unwrappable:
			errs = append(errs, unwrapDeep(v)...)
		default:
			errs = append(errs, v)
		}
	}
	return errs
}

// unwrapDeep walks down a stack of nested fmt.Errorf("%w: %w") errors
// This is necessary since fmt.wrapErrors doesn't flatten the error slices
func unwrapDeep(err unwrappable) []error {
	var n []error
	for _, e := range err.Unwrap() {
		if u, ok := e.(unwrappable); ok {
			n = append(n, u.Unwrap()...)
		} else {
			n = append(n, e)
		}
	}
	return n
}

// ExcludeError returns a new error excluding any errors matching excl
// For a standard error it will either return the error unchanged or nil
// For an error which implements Unwrap() []error it will remove any errors matching excl and return the remaining errors or nil
// Any non-error messages and formatting from fmt.Errorf will be lost; This function is written for conditions
func ExcludeError(err, excl error) error {
	if u, ok := err.(unwrappable); ok {
		var n error
		for _, e := range unwrapDeep(u) {
			if !errors.Is(e, excl) {
				n = AppendError(n, e)
			}
		}
		return n
	}
	if errors.Is(err, excl) {
		return nil
	}
	return err
}

// ErrorCollector allows collecting a stream of errors from concurrent go routines
// Users should call e.Wg.Done and send errors to e.C
type ErrorCollector struct {
	C  chan error
	Wg sync.WaitGroup
}

// CollectErrors returns an ErrorCollector with WaitGroup and Channel buffer set to n
func CollectErrors(n int) *ErrorCollector {
	e := &ErrorCollector{
		C: make(chan error, n),
	}
	e.Wg.Add(n)
	return e
}

// Collect runs waits for e.Wg to be Done, closes the error channel, and return a error collection
func (e *ErrorCollector) Collect() (errs error) {
	e.Wg.Wait()
	close(e.C)
	for err := range e.C {
		if err != nil {
			errs = AppendError(errs, err)
		}
	}
	return
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

// GetTypeAssertError returns additional information for when an assertion failure
// occurs.
// fieldDescription is an optional way to return what the affected field was for
func GetTypeAssertError(required string, received interface{}, fieldDescription ...string) error {
	var description string
	if len(fieldDescription) > 0 {
		description = " for: " + strings.Join(fieldDescription, ", ")
	}
	return fmt.Errorf("%w from %T to %s%s", ErrTypeAssertFailure, received, required, description)
}

// Batch takes a slice type and converts it into a slice of containing slices of length batchSize, and any remainder in the final batch
// batchSize <= 0 will return the entire input slice in one batch
func Batch[S ~[]E, E any](blobs S, batchSize int) []S {
	if len(blobs) == 0 {
		return []S{}
	}
	blobs = slices.Clone(blobs)
	if batchSize <= 0 {
		return []S{blobs}
	}
	i := 0
	batches := make([]S, (len(blobs)+batchSize-1)/batchSize)
	for batchSize < len(blobs) {
		blobs, batches[i] = blobs[batchSize:], blobs[:batchSize:batchSize]
		i++
	}
	if len(blobs) > 0 {
		batches[i] = blobs
	}
	return batches
}

// SortStrings takes a slice of fmt.Stringer implementers and returns a new ascending sorted slice
func SortStrings[S ~[]E, E fmt.Stringer](x S) S {
	n := slices.Clone(x)
	slices.SortFunc(n, func(a, b E) int {
		return strings.Compare(a.String(), b.String())
	})
	return n
}

// Counter is a thread-safe counter.
type Counter struct {
	n int64 // privatised so you can't use counter as a value type
}

// IncrementAndGet returns the next count after incrementing.
func (c *Counter) IncrementAndGet() int64 {
	newID := atomic.AddInt64(&c.n, 1)
	// Handle overflow by resetting the counter to 1 if it becomes negative
	if newID < 0 {
		atomic.StoreInt64(&c.n, 1)
		return 1
	}
	return newID
}
