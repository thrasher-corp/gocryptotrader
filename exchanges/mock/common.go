package mock

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var (
	errJSONMapPayloadMustBeObject = errors.New("json map payload must be an object")
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

// DeriveURLValsFromJSONSlice converts a JSON array into a slice of url.Values by processing each array element as a JSON object
func DeriveURLValsFromJSONSlice(payload []byte) ([]url.Values, error) {
	if len(payload) == 0 {
		return []url.Values{}, nil
	}
	var intermediary []json.RawMessage
	if err := json.Unmarshal(payload, &intermediary); err != nil {
		return nil, err
	}

	vals := make([]url.Values, len(intermediary))
	for i := range intermediary {
		result, err := DeriveURLValsFromJSONMap(intermediary[i])
		if err != nil {
			return nil, err
		}
		vals[i] = result
	}
	return vals, nil
}

// DeriveURLValsFromJSONMap gets url vals from a map[string]string encoded JSON body
func DeriveURLValsFromJSONMap(payload []byte) (url.Values, error) {
	vals := url.Values{}
	if len(payload) == 0 {
		return vals, nil
	}
	if getJSONBodyShape(strings.TrimSpace(string(payload))) == jsonBodyArray {
		return vals, errJSONMapPayloadMustBeObject
	}
	intermediary := make(map[string]any)
	if err := json.Unmarshal(payload, &intermediary); err != nil {
		return vals, err
	}

	for k, v := range intermediary {
		switch val := v.(type) {
		case string:
			vals.Add(k, val)
		case bool:
			vals.Add(k, strconv.FormatBool(val))
		case float64:
			vals.Add(k, strconv.FormatFloat(val, 'f', -1, 64))
		case map[string]any, []any, nil:
			b, err := json.Marshal(val)
			if err != nil {
				return vals, err
			}
			vals.Add(k, string(b))
		default:
			return vals, fmt.Errorf("unhandled conversion type: %T, please add as needed", val)
		}
	}

	return vals, nil
}
