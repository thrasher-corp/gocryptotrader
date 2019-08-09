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
func DeriveURLValsFromJSONMap(payload []byte) (url.Values, error) {
	var vals = url.Values{}
	if string(payload) == "" {
		return vals, nil
	}
	intermediary := make(map[string]interface{})
	err := json.Unmarshal(payload, &intermediary)
	if err != nil {
		return vals, err
	}

	for k, v := range intermediary {
		switch v.(type) {
		case string:
			vals.Add(k, v.(string))
		case bool:
			vals.Add(k, strconv.FormatBool(v.(bool)))
		case map[string]interface{}, []interface{}, nil:
			vals.Add(k, fmt.Sprintf("%v", v))
		default:
			log.Println(reflect.TypeOf(v))
			return vals, errors.New("unhandled conversion type, please add as needed")
		}

	}

	return vals, nil
}
