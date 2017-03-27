package common

import (
	//"bytes"
	"crypto/hmac"
	"crypto/md5"
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
	"strconv"
	"strings"
	"time"
)

const (
	HASH_SHA1 = iota
	HASH_SHA256
	HASH_SHA512
	HASH_SHA512_384
	SATOSHIS_PER_BTC = 100000000
	SATOSHIS_PER_LTC = 100000000
	WEI_PER_ETHER    = 1000000000000000000
)

func GetMD5(input []byte) []byte {
	hash := md5.New()
	hash.Write(input)
	return hash.Sum(nil)
}

func GetSHA512(input []byte) []byte {
	sha := sha512.New()
	sha.Write(input)
	return sha.Sum(nil)
}

func GetSHA256(input []byte) []byte {
	sha := sha256.New()
	sha.Write(input)
	return sha.Sum(nil)
}

func GetHMAC(hashType int, input, key []byte) []byte {
	var hash func() hash.Hash

	switch hashType {
	case HASH_SHA1:
		{
			hash = sha1.New
		}
	case HASH_SHA256:
		{
			hash = sha256.New
		}
	case HASH_SHA512:
		{
			hash = sha512.New
		}
	case HASH_SHA512_384:
		{
			hash = sha512.New384
		}
	}

	hmac := hmac.New(hash, []byte(key))
	hmac.Write(input)
	return hmac.Sum(nil)
}

func HexEncodeToString(input []byte) string {
	return hex.EncodeToString(input)
}

func Base64Decode(input string) ([]byte, error) {
	result, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func Base64Encode(input []byte) string {
	return base64.StdEncoding.EncodeToString(input)
}

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

func StringContains(input, substring string) bool {
	return strings.Contains(input, substring)
}

func DataContains(haystack []string, needle string) bool {
	data := strings.Join(haystack, ",")
	return strings.Contains(data, needle)
}

func JoinStrings(input []string, seperator string) string {
	return strings.Join(input, seperator)
}

func SplitStrings(input, seperator string) []string {
	return strings.Split(input, seperator)
}

func TrimString(input, cutset string) string {
	return strings.Trim(input, cutset)
}

func StringToUpper(input string) string {
	return strings.ToUpper(input)
}

func StringToLower(input string) string {
	return strings.ToLower(input)
}

func RoundFloat(x float64, prec int) float64 {
	var rounder float64
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)
	intermed += .5
	x = .5
	if frac < 0.0 {
		x = -.5
		intermed -= 1
	}
	if frac >= x {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}

	return rounder / pow
}

func IsEnabled(isEnabled bool) string {
	if isEnabled {
		return "Enabled"
	} else {
		return "Disabled"
	}
}

func YesOrNo(input string) bool {
	if StringToLower(input) == "y" || StringToLower(input) == "yes" {
		return true
	}
	return false
}

func CalculateAmountWithFee(amount, fee float64) float64 {
	return amount + CalculateFee(amount, fee)
}

func CalculateFee(amount, fee float64) float64 {
	return amount * (fee / 100)
}

func CalculatePercentageGainOrLoss(priceNow, priceThen float64) float64 {
	return (priceNow - priceThen) / priceThen * 100
}

func CalculatePercentageDifference(amount, secondAmount float64) float64 {
	return (amount - secondAmount) / ((amount + secondAmount) / 2) * 100
}

func CalculateNetProfit(amount, priceThen, priceNow, costs float64) float64 {
	return (priceNow * amount) - (priceThen * amount) - costs
}

func SendHTTPRequest(method, path string, headers map[string]string, body io.Reader) (string, error) {
	result := strings.ToUpper(method)

	if result != "POST" && result != "GET" && result != "DELETE" {
		return "", errors.New("Invalid HTTP method specified.")
	}

	req, err := http.NewRequest(method, path, body)

	if err != nil {
		return "", err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)

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

func SendHTTPGetRequest(url string, jsonDecode bool, result interface{}) (err error) {
	res, err := http.Get(url)

	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		log.Printf("HTTP status code: %d\n", res.StatusCode)
		return errors.New("Status code was not 200.")
	}

	contents, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	if jsonDecode {
		err := JSONDecode(contents, &result)
		if err != nil {
			log.Println(string(contents[:]))
			return err
		}
	} else {
		result = &contents
	}

	return nil
}

func JSONEncode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func JSONDecode(data []byte, to interface{}) error {
	return json.Unmarshal(data, to)
}

func EncodeURLValues(url string, values url.Values) string {
	path := url
	if len(values) > 0 {
		path += "?" + values.Encode()
	}
	return path
}

func ExtractHost(address string) string {
	host := SplitStrings(address, ":")[0]
	if host == "" {
		return "localhost"
	}
	return host
}

func ExtractPort(host string) int {
	portStr := SplitStrings(host, ":")[1]
	port, _ := strconv.Atoi(portStr)
	return port
}

func OutputCSV(path string, data [][]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	writer := csv.NewWriter(file)

	err = writer.WriteAll(data)
	if err != nil {
		return err
	}

	defer writer.Flush()
	return nil
}

func UnixTimestampToTime(timeint64 int64) time.Time {
	return time.Unix(timeint64, 0)
}

func UnixTimestampStrToTime(timeStr string) (time.Time, error) {
	i, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(i, 0), nil
}

func ReadFile(path string) ([]byte, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func WriteFile(file string, data []byte) error {
	err := ioutil.WriteFile(file, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// GetURIPath returns the path of a URL given a URL
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
