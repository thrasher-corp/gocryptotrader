package common

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Vars for common.go operations
var (
	HTTPClient    *http.Client
	HTTPUserAgent string
)

// Const declarations for common.go operations
const (
	HashSHA1 = iota
	HashSHA256
	HashSHA512
	HashSHA512_384
	HashMD5
	SatoshisPerBTC = 100000000
	SatoshisPerLTC = 100000000
	WeiPerEther    = 1000000000000000000
)

func initialiseHTTPClient() {
	// If the HTTPClient isn't set, start a new client with a default timeout of 5 seconds
	if HTTPClient == nil {
		HTTPClient = NewHTTPClientWithTimeout(time.Duration(time.Second * 5))
	}
}

// NewHTTPClientWithTimeout initialises a new HTTP client with the specified
// timeout duration
func NewHTTPClientWithTimeout(t time.Duration) *http.Client {
	h := &http.Client{Timeout: t}
	return h
}

// GetRandomSalt returns a random salt
func GetRandomSalt(input []byte, saltLen int) ([]byte, error) {
	if saltLen <= 0 {
		return nil, errors.New("salt length is too small")
	}
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	var result []byte
	if input != nil {
		result = input
	}
	result = append(result, salt...)
	return result, nil
}

// GetMD5 returns a MD5 hash of a byte array
func GetMD5(input []byte) []byte {
	hash := md5.New()
	hash.Write(input)
	return hash.Sum(nil)
}

// GetSHA512 returns a SHA512 hash of a byte array
func GetSHA512(input []byte) []byte {
	sha := sha512.New()
	sha.Write(input)
	return sha.Sum(nil)
}

// GetSHA256 returns a SHA256 hash of a byte array
func GetSHA256(input []byte) []byte {
	sha := sha256.New()
	sha.Write(input)
	return sha.Sum(nil)
}

// GetHMAC returns a keyed-hash message authentication code using the desired
// hashtype
func GetHMAC(hashType int, input, key []byte) []byte {
	var hash func() hash.Hash

	switch hashType {
	case HashSHA1:
		{
			hash = sha1.New
		}
	case HashSHA256:
		{
			hash = sha256.New
		}
	case HashSHA512:
		{
			hash = sha512.New
		}
	case HashSHA512_384:
		{
			hash = sha512.New384
		}
	case HashMD5:
		{
			hash = md5.New
		}
	}

	hmac := hmac.New(hash, []byte(key))
	hmac.Write(input)
	return hmac.Sum(nil)
}

// Sha1ToHex takes a string, sha1 hashes it and return a hex string of the
// result
func Sha1ToHex(data string) string {
	h := sha1.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// HexEncodeToString takes in a hexadecimal byte array and returns a string
func HexEncodeToString(input []byte) string {
	return hex.EncodeToString(input)
}

// Base64Decode takes in a Base64 string and returns a byte array and an error
func Base64Decode(input string) ([]byte, error) {
	result, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Base64Encode takes in a byte array then returns an encoded base64 string
func Base64Encode(input []byte) string {
	return base64.StdEncoding.EncodeToString(input)
}

// StringSliceDifference concatenates slices together based on its index and
// returns an individual string array
func StringSliceDifference(slice1 []string, slice2 []string) []string {
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

// StringDataCompareUpper data checks the substring array with an input and returns
// a bool irrespective of lower or upper case strings
func StringDataCompareUpper(haystack []string, needle string) bool {
	for x := range haystack {
		if StringToUpper(haystack[x]) == StringToUpper(needle) {
			return true
		}
	}
	return false
}

// StringDataContainsUpper checks the substring array with an input and returns
// a bool irrespective of lower or upper case strings
func StringDataContainsUpper(haystack []string, needle string) bool {
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
func ReplaceString(input, old, new string, n int) string {
	return strings.Replace(input, old, new, n)
}

// StringToUpper changes strings to uppercase
func StringToUpper(input string) string {
	return strings.ToUpper(input)
}

// StringToLower changes strings to lowercase
func StringToLower(input string) string {
	return strings.ToLower(input)
}

// RoundFloat rounds your floating point number to the desired decimal place
func RoundFloat(x float64, prec int) float64 {
	var rounder float64
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)
	intermed += .5
	x = .5
	if frac < 0.0 {
		x = -.5
		intermed--
	}
	if frac >= x {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}

	return rounder / pow
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
		return false, errors.New("Invalid crypto currency")
	}
}

// YesOrNo returns a boolean variable to check if input is "y" or "yes"
func YesOrNo(input string) bool {
	if StringToLower(input) == "y" || StringToLower(input) == "yes" {
		return true
	}
	return false
}

// CalculateAmountWithFee returns a calculated fee included amount on fee
func CalculateAmountWithFee(amount, fee float64) float64 {
	return amount + CalculateFee(amount, fee)
}

// CalculateFee returns a simple fee on amount
func CalculateFee(amount, fee float64) float64 {
	return amount * (fee / 100)
}

// CalculatePercentageGainOrLoss returns the percentage rise over a certain
// period
func CalculatePercentageGainOrLoss(priceNow, priceThen float64) float64 {
	return (priceNow - priceThen) / priceThen * 100
}

// CalculatePercentageDifference returns the percentage of difference between
// multiple time periods
func CalculatePercentageDifference(amount, secondAmount float64) float64 {
	return (amount - secondAmount) / ((amount + secondAmount) / 2) * 100
}

// CalculateNetProfit returns net profit
func CalculateNetProfit(amount, priceThen, priceNow, costs float64) float64 {
	return (priceNow * amount) - (priceThen * amount) - costs
}

// SendHTTPRequest sends a request using the http package and returns a response
// as a string and an error
func SendHTTPRequest(method, path string, headers map[string]string, body io.Reader) (string, error) {
	result := strings.ToUpper(method)

	if result != "POST" && result != "GET" && result != "DELETE" {
		return "", errors.New("invalid HTTP method specified")
	}

	initialiseHTTPClient()

	req, err := http.NewRequest(method, path, body)
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
func SendHTTPGetRequest(url string, jsonDecode, isVerbose bool, result interface{}) error {
	if isVerbose {
		log.Println("Raw URL: ", url)
	}

	initialiseHTTPClient()

	res, err := HTTPClient.Get(url)
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
		log.Println("Raw Resp: ", string(contents[:]))
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
func EncodeURLValues(url string, values url.Values) string {
	path := url
	if len(values) > 0 {
		path += "?" + values.Encode()
	}
	return path
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
func OutputCSV(path string, data [][]string) error {
	_, err := ReadFile(path)
	if err != nil {
		errTwo := WriteFile(path, nil)
		if errTwo != nil {
			return errTwo
		}
	}

	file, err := os.Create(path)
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
func ReadFile(path string) ([]byte, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// WriteFile writes selected data to a file and returns an error
func WriteFile(file string, data []byte) error {
	err := ioutil.WriteFile(file, data, 0644)
	if err != nil {
		return err
	}
	return nil
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
		return 0, fmt.Errorf("Could not convert value: %s Error: %s", str, err)
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
		return os.Getenv("APPDATA") + GetOSPathSlash() + "GoCryptoTrader"
	}
	return path.Join(os.ExpandEnv("$HOME"), ".gocryptotrader")
}

// CheckDir checks to see if a particular directory exists
// and attempts to create it if desired, if it doesn't exist
func CheckDir(dir string, create bool) error {
	_, err := os.Stat(dir)
	if !os.IsNotExist(err) {
		return nil
	}

	if !create {
		return fmt.Errorf("directory %s does not exist. Err: %s", dir, err)
	}

	log.Printf("Directory %s does not exist.. creating.", dir)
	err = os.Mkdir(dir, 0777)
	if err != nil {
		return fmt.Errorf("failed to create dir. Err: %s", err)
	}
	return nil
}
