package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// MatchURLVals matches url.Value query strings
func MatchURLVals(v1, v2 url.Values) bool {
	if len(v1) != len(v2) {
		return false
	}

	if len(v1) == 0 && len(v2) == 0 {
		return true
	}

	for key, val := range v1 {
		if key == "nonce" || key == "signature" || key == "timestamp" || key == "tonce" || key == "key" { // delta values
			if _, ok := v2[key]; !ok {
				return false
			}
			continue
		}

		if val2, ok := v2[key]; ok {
			if strings.Join(val2, "") == strings.Join(val, "") {
				continue
			}
		}
		return false
	}
	return true
}

// DeriveURLValsFromJSONMap gets url vals from a map[string]string encoded JSON body
func DeriveURLValsFromJSONMap(payload []byte) ([]url.Values, error) {
	var vals = []url.Values{}
	if len(payload) == 0 {
		return vals, nil
	}
	intermediary := make([]map[string]interface{}, 1, 5)
	intermediary[0] = make(map[string]interface{})
	err := json.Unmarshal(payload, &intermediary[0])
	if err != nil {
		if strings.EqualFold(err.Error(), "json: cannot unmarshal array into Go value of type map[string]interface {}") {
			err = json.Unmarshal(payload, &intermediary)
			if err != nil {
				return nil, err
			}
		} else {
			return vals, err
		}
	}
	for x := range intermediary {
		valsItem := url.Values{}
		for k, v := range intermediary[x] {
			switch val := v.(type) {
			case string:
				valsItem.Add(k, val)
			case bool:
				valsItem.Add(k, strconv.FormatBool(val))
			case float64:
				valsItem.Add(k, strconv.FormatFloat(val, 'f', -1, 64))
			case map[string]interface{}, []interface{}, nil:
				valsItem.Add(k, fmt.Sprintf("%v", val))
			default:
				log.Println(reflect.TypeOf(val))
				return vals, errors.New("unhandled conversion type, please add as needed")
			}
		}
		vals = append(vals, valsItem)
	}

	return vals, nil
}
