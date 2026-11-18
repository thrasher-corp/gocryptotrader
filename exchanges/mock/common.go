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
	errJSONMapPayloadMustBeArray  = errors.New("json map payload must be an array")
)

// structuredValuePrefix marks an object, array or null derived from a JSON body, so it never matches a string holding
// the same JSON text, and escapedStringPrefix marks a string starting with the byte the markers start with, so no string
// matches a marked value
const (
	structuredValuePrefix = "\x00json:"
	escapedStringPrefix   = "\x00string:"
)

// stringValue returns a JSON string value as it is compared, escaped when it could be taken for a marked value
func stringValue(s string) string {
	if strings.HasPrefix(s, "\x00") {
		return escapedStringPrefix + s
	}
	return s
}

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

// DeriveURLValsFromJSONArray converts a JSON array into a slice of url.Values by processing each array element as a JSON object
func DeriveURLValsFromJSONArray(payload []byte) ([]url.Values, error) {
	if len(payload) == 0 {
		return []url.Values{}, nil
	}
	var intermediary []json.RawMessage
	err := json.Unmarshal(payload, &intermediary)
	if err != nil {
		return nil, err
	}

	vals := make([]url.Values, len(intermediary))
	for i := range intermediary {
		var result url.Values
		if getJSONBodyShape(strings.TrimSpace(string(intermediary[i]))) == jsonBodyArray {
			result, err = DeriveURLValsFromJSONArrayAsMap(intermediary[i])
		} else {
			result, err = DeriveURLValsFromJSONMap(intermediary[i])
		}
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
		return nil, errJSONMapPayloadMustBeObject
	}
	intermediary := make(map[string]any)
	if err := json.Unmarshal(payload, &intermediary); err != nil {
		return nil, err
	}

	for k, v := range intermediary {
		switch val := v.(type) {
		case string:
			vals.Add(k, stringValue(val))
		case bool:
			vals.Add(k, strconv.FormatBool(val))
		case float64:
			vals.Add(k, strconv.FormatFloat(val, 'f', -1, 64))
		case map[string]any, []any, nil:
			b, err := json.Marshal(val)
			if err != nil {
				return nil, err
			}
			vals.Add(k, structuredValuePrefix+string(b))
		default:
			return nil, fmt.Errorf("unhandled conversion type: %T, please add as needed", val)
		}
	}

	return vals, nil
}

// DeriveURLValsFromJSONArrayAsMap converts a JSON array into url.Values,
// using indices as keys and stringified values
func DeriveURLValsFromJSONArrayAsMap(payload []byte) (url.Values, error) {
	vals := url.Values{}
	if len(payload) == 0 {
		return vals, nil
	}
	if getJSONBodyShape(strings.TrimSpace(string(payload))) != jsonBodyArray {
		return nil, errJSONMapPayloadMustBeArray
	}
	var intermediary []any
	if err := json.Unmarshal(payload, &intermediary); err != nil {
		return nil, err
	}

	for k, v := range intermediary {
		switch val := v.(type) {
		case string:
			vals.Add(strconv.Itoa(k), stringValue(val))
		case bool:
			vals.Add(strconv.Itoa(k), strconv.FormatBool(val))
		case float64:
			vals.Add(strconv.Itoa(k), strconv.FormatFloat(val, 'f', -1, 64))
		case map[string]any, []any, nil:
			b, err := json.Marshal(val)
			if err != nil {
				return vals, err
			}
			vals.Add(strconv.Itoa(k), structuredValuePrefix+string(b))
		default:
			return nil, fmt.Errorf("unhandled conversion type: %T, please add as needed", val)
		}
	}

	return vals, nil
}
