package mock

import (
	"encoding/json"
	"errors"
	"net/url"
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
	intermediary := make(map[string]string)
	err := json.Unmarshal(payload, &intermediary)
	if err != nil {
		return vals,
			errors.New("unexpected JSON format needs to be of type map[string]string")
	}

	for k, v := range intermediary {
		vals.Add(k, v)
	}

	return vals, nil
}
