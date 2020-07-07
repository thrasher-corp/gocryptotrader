package funding

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// GetFundingRates gets funding rates
func GetFundingRates(path string) error {
	resp, err := http.Get(path)
	if err != nil {
		return err
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))
	// var more interface{}
	// err = json.Unmarshal(bytes, &resp)
	// fmt.Println(more)
	return err
}
