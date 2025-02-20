package mock

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var errUnsupportedType = errors.New("unsupported type")

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
	var marshaledResult interface{}
	var intermediary []map[string]interface{}
	err := json.Unmarshal(payload, &marshaledResult)
	if err != nil {
		return vals, err
	}
	switch value := marshaledResult.(type) {
	case []interface{}:
		intermediary = make([]map[string]interface{}, len(value))
		var okay bool
		for i := range value {
			intermediary[i], okay = value[i].(map[string]interface{})
			if !okay {
				return nil, errUnsupportedType
			}
		}
	case map[string]interface{}:
		intermediary = []map[string]interface{}{value}
	default:
		return nil, errUnsupportedType
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
				return vals, errors.New("unhandled conversion type, please add as needed")
			}
		}
		vals = append(vals, valsItem)
	}

	return vals, nil
}
