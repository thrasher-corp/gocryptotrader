package main

import (
	"net/http"
	"hash"
	"crypto/md5"
	"crypto/hmac"
	"crypto/sha512"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/hex"
	"io/ioutil"
	"errors"
	"math"
	"log"
)

func GetMD5(input []byte) ([]byte) {
	hash := md5.New()
	hash.Write(input)
	return hash.Sum(nil)
}

func GetSHA512(input []byte) ([]byte) {
	sha := sha512.New()
	sha.Write(input)
	return sha.Sum(nil)
}

func GetSHA256(input []byte) ([]byte) {
	sha := sha256.New()
	sha.Write(input)
	return sha.Sum(nil)
}

func GetHMAC(hash func() hash.Hash, input, key []byte) ([]byte) {
	hmac := hmac.New(hash, []byte(key))
	hmac.Write(input)
	return hmac.Sum(nil)
}

func HexEncodeToString(input []byte) (string) {
	return hex.EncodeToString(input)
}

func Base64Decode(input string) ([]byte, error) {
	result, err := base64.StdEncoding.DecodeString(input) 
	if err != nil {
		return nil, err
	}
	return result, nil
}

func Base64Encode(input []byte) (string) {
	return base64.StdEncoding.EncodeToString(input)
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

func CalculateAmountWithFee(amount, fee float64) (float64) {
	return amount + CalculateFee(amount, fee)
}

func CalculateFee(amount, fee float64) (float64) {
	return amount * (fee / 100)
}

func CalculatePercentageDifference(amount, secondAmount float64) (float64) {
	return (secondAmount - amount) / amount * 100
}

func CalculateNetProfit(amount, priceThen, priceNow, costs float64) (float64) {
	return (priceNow * amount) - (priceThen * amount) - costs
}

func SendHTTPRequest(url string, jsonDecode bool, result interface{}) (err error) {
	res, err := http.Get(url)

	if err != nil {
		log.Println(err)
		return err
	}

	if res.StatusCode != 200 {
		log.Printf("HTTP status code: %d\n", res.StatusCode)
		return errors.New("Status code was not 200.")
	}

	contents, _ := ioutil.ReadAll(res.Body)
	//log.Printf("Recieved raw: %s\n", string(contents))

	if jsonDecode {
		err := json.Unmarshal(contents, &result)

		if err != nil {
			return errors.New("Unable to JSON decode body.")
		}
	} else {
		result = contents
	}
	return
}