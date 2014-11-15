package main

import (
	"net/http"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"errors"
)

func SendHTTPRequest(url string, jsonDecode bool, result interface{}) (err error) {
	res, err := http.Get(url)

	if err != nil {
		fmt.Println(err)
		return err
	}

	if res.StatusCode != 200 {
		fmt.Printf("HTTP status code: %d", res.StatusCode)
		return errors.New("Status code was not 200.")
	}

	contents, _ := ioutil.ReadAll(res.Body)
	//fmt.Printf("Recieved raw: %s\n", string(contents))

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