package mock

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// HTTPResponse defines expected response from the end point including request
// data for pathing on the VCR server
type HTTPResponse struct {
	Data        json.RawMessage     `json:"data"`
	QueryString string              `json:"queryString"`
	BodyParams  string              `json:"bodyParams"`
	Headers     map[string][]string `json:"headers"`
}

// HTTPRecord will record the request and response to a default JSON file for
// mocking purposes
func HTTPRecord(res *http.Response, service string, respContents []byte) error {
	if res == nil {
		return errors.New("http.Response cannot be nil")
	}

	if res.Request == nil {
		return errors.New("http.Request cannot be nil")
	}

	if res.Request.Method == "" {
		return errors.New("request method not supplied")
	}

	if service == "" {
		return errors.New("service not supplied cannot access correct mock file")
	}
	service = strings.ToLower(service)

	outputFilePath := filepath.Join(DefaultDirectory, service, service+".json")
	_, err := os.Stat(outputFilePath)
	if err != nil {
		if os.IsExist(err) {
			return err
		}
		// check alternative path to add compatibility with /internal/testing/exchange/exchange.go MockHTTPInstance
		outputFilePath = filepath.Join("..", service, "testdata", "http.json")
		_, err = os.Stat(outputFilePath)
		if err != nil {
			return err
		}
	}

	contents, err := os.ReadFile(outputFilePath)
	if err != nil {
		return err
	}

	var m VCRMock
	err = json.Unmarshal(contents, &m)
	if err != nil {
		return err
	}

	if m.Routes == nil {
		m.Routes = make(map[string]map[string][]HTTPResponse)
	}

	var httpResponse HTTPResponse
	cleanedContents, err := CheckResponsePayload(respContents)
	if err != nil {
		return err
	}

	err = json.Unmarshal(cleanedContents, &httpResponse.Data)
	if err != nil {
		return err
	}

	var body string
	if res.Request.GetBody != nil {
		bodycopy, bodyErr := res.Request.GetBody()
		if bodyErr != nil {
			return bodyErr
		}
		payload, bodyErr := io.ReadAll(bodycopy)
		if bodyErr != nil {
			return bodyErr
		}
		body = string(payload)
	}

	switch res.Request.Header.Get(contentType) {
	case applicationURLEncoded:
		vals, urlErr := url.ParseQuery(body)
		if urlErr != nil {
			return urlErr
		}

		httpResponse.BodyParams, urlErr = GetFilteredURLVals(vals)
		if urlErr != nil {
			return urlErr
		}

	case textPlain:
		payload := res.Request.Header.Get("X-Gemini-Payload")
		j, dErr := base64.StdEncoding.DecodeString(payload)
		if dErr != nil {
			return dErr
		}

		httpResponse.BodyParams = string(j)

	default:
		httpResponse.BodyParams = body
	}

	httpResponse.Headers, err = GetFilteredHeader(res)
	if err != nil {
		return err
	}

	httpResponse.QueryString, err = GetFilteredURLVals(res.Request.URL.Query())
	if err != nil {
		return err
	}

	_, ok := m.Routes[res.Request.URL.Path]
	if !ok {
		m.Routes[res.Request.URL.Path] = make(map[string][]HTTPResponse)
		m.Routes[res.Request.URL.Path][res.Request.Method] = []HTTPResponse{httpResponse}
	} else {
		mockResponses, ok := m.Routes[res.Request.URL.Path][res.Request.Method]
		if !ok {
			m.Routes[res.Request.URL.Path][res.Request.Method] = []HTTPResponse{httpResponse}
		} else {
			switch res.Request.Method { // Based off method - check add or replace
			case http.MethodGet:
				for i := range mockResponses {
					mockQuery, urlErr := url.ParseQuery(mockResponses[i].QueryString)
					if urlErr != nil {
						return urlErr
					}

					if MatchURLVals(mockQuery, res.Request.URL.Query()) {
						mockResponses = slices.Delete(mockResponses, i, i+1)
						break
					}
				}

			case http.MethodPost:
				for i := range mockResponses {
					cType, ok := mockResponses[i].Headers[contentType]

					jCType := strings.Join(cType, "")
					var found bool
					switch jCType {
					case applicationURLEncoded:
						respQueryVals, urlErr := url.ParseQuery(body)
						if urlErr != nil {
							return urlErr
						}

						mockRespVals, urlErr := url.ParseQuery(mockResponses[i].BodyParams)
						if urlErr != nil {
							return urlErr
						}

						if MatchURLVals(respQueryVals, mockRespVals) {
							// if found will delete instance and overwrite with new
							// data
							mockResponses = slices.Delete(mockResponses, i, i+1)
							found = true
						}

					case applicationJSON, textPlain:
						reqVals, jErr := DeriveURLValsFromJSONMap([]byte(body))
						if jErr != nil {
							return jErr
						}

						mockVals, jErr := DeriveURLValsFromJSONMap([]byte(mockResponses[i].BodyParams))
						if jErr != nil {
							return jErr
						}

						if MatchURLVals(reqVals, mockVals) {
							// if found will delete instance and overwrite with new
							// data
							mockResponses = slices.Delete(mockResponses, i, i+1)
							found = true
						}
					case "":
						if !ok {
							// Assume query params are used
							mockQuery, urlErr := url.ParseQuery(mockResponses[i].QueryString)
							if urlErr != nil {
								return urlErr
							}

							if MatchURLVals(mockQuery, res.Request.URL.Query()) {
								// if found will delete instance and overwrite with new data
								mockResponses = slices.Delete(mockResponses, i, i+1)
								found = true
							}

							break
						}

						fallthrough
					default:
						return fmt.Errorf("unhandled content type %s", jCType)
					}
					if found {
						break
					}
				}

			default:
				return fmt.Errorf("unhandled request method %s", res.Request.Method)
			}

			m.Routes[res.Request.URL.Path][res.Request.Method] = append(mockResponses, httpResponse)
		}
	}

	payload, err := json.MarshalIndent(m, "", " ")
	if err != nil {
		return err
	}

	return file.Write(outputFilePath, payload)
}

// GetFilteredHeader filters excluded http headers for insertion into a mock
// test file
func GetFilteredHeader(res *http.Response) (http.Header, error) {
	items, err := GetExcludedItems()
	if err != nil {
		return res.Header, err
	}

	for i := range items.Headers {
		if res.Request.Header.Get(items.Headers[i]) != "" {
			res.Request.Header.Set(items.Headers[i], "")
		}
	}

	return res.Request.Header, nil
}

// GetFilteredURLVals filters excluded url value variables for insertion into a
// mock test file
func GetFilteredURLVals(vals url.Values) (string, error) {
	items, err := GetExcludedItems()
	if err != nil {
		return "", err
	}

	for key, val := range vals {
		for i := range items.Variables {
			if strings.EqualFold(items.Variables[i], val[0]) {
				vals.Set(key, "")
			}
		}
	}
	return vals.Encode(), nil
}

// CheckResponsePayload checks to see if there are any response body variables
// that should not be there.
func CheckResponsePayload(data []byte) ([]byte, error) {
	items, err := GetExcludedItems()
	if err != nil {
		return nil, err
	}

	var intermediary any
	err = json.Unmarshal(data, &intermediary)
	if err != nil {
		return nil, err
	}

	payload, err := CheckJSON(intermediary, &items)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(payload, "", " ")
}

// Reflection consts
const (
	Int64   = "int64"
	Float64 = "float64"
	Slice   = "slice"
	String  = "string"
	Bool    = "bool"
	Invalid = "invalid"
)

// CheckJSON recursively parses json data to retract keywords, quite intensive.
func CheckJSON(data any, excluded *Exclusion) (any, error) {
	if d, ok := data.([]any); ok {
		var sData []any
		for i := range d {
			v := d[i]
			switch v.(type) {
			case map[string]any, []any:
				checkedData, err := CheckJSON(v, excluded)
				if err != nil {
					return nil, err
				}

				sData = append(sData, checkedData)
			default:
				// Primitive value doesn't need exclusions applied, e.g. float64 or string
				sData = append(sData, v)
			}
		}
		return sData, nil
	}

	conv, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var context map[string]any
	err = json.Unmarshal(conv, &context)
	if err != nil {
		return nil, err
	}

	if len(context) == 0 {
		// Nil for some reason, should error out before in json.Unmarshal
		return nil, nil
	}

	for key, val := range context {
		switch reflect.ValueOf(val).Kind().String() {
		case String:
			if IsExcluded(key, excluded.Variables) {
				context[key] = "" // Zero val string
			}
		case Int64:
			if IsExcluded(key, excluded.Variables) {
				context[key] = 0 // Zero val int
			}
		case Float64:
			if IsExcluded(key, excluded.Variables) {
				context[key] = 0.0 // Zero val float
			}
		case Slice:
			slice, ok := val.([]any)
			if !ok {
				return nil, common.GetTypeAssertError("[]any", val)
			}
			if len(slice) < 1 {
				// Empty slice found
				context[key] = slice
			} else {
				if _, ok := slice[0].(map[string]any); ok {
					var cleanSlice []any
					for i := range slice {
						cleanMap, sErr := CheckJSON(slice[i], excluded)
						if sErr != nil {
							return nil, sErr
						}
						cleanSlice = append(cleanSlice, cleanMap)
					}
					context[key] = cleanSlice
				} else if IsExcluded(key, excluded.Variables) {
					context[key] = nil // Zero val slice
				}
			}

		case Bool, Invalid: // Skip these bad boys for now
		default:
			// Recursively check map data
			contextValue, err := CheckJSON(val, excluded)
			if err != nil {
				return nil, err
			}
			context[key] = contextValue
		}
	}

	return context, nil
}

// IsExcluded cross references the key with the excluded variables
func IsExcluded(key string, excludedVars []string) bool {
	for i := range excludedVars {
		if strings.EqualFold(key, excludedVars[i]) {
			return true
		}
	}
	return false
}

var (
	excludedList  Exclusion
	m             sync.Mutex
	set           bool
	exclusionFile = DefaultDirectory + "exclusion.json"
)

var defaultExcludedHeaders = []string{
	"Key",
	"X-Mbx-Apikey",
	"Rest-Key",
	"Apiauth-Key",
	"X-Bapi-Api-Key",
}

var defaultExcludedVariables = []string{
	"bsb",
	"user",
	"name",
	"real_name",
	"receiver_name",
	"account_number",
	"username",
	"apiKey",
}

// Exclusion defines a list of items to be excluded from the main mock output
// this attempts a catch all approach and needs to be updated per exchange basis
type Exclusion struct {
	Headers   []string `json:"headers"`
	Variables []string `json:"variables"`
}

// GetExcludedItems checks to see if the variable is in the exclusion list as to
// not display secure items in mock file generator output
func GetExcludedItems() (Exclusion, error) {
	m.Lock()
	defer m.Unlock()
	if !set {
		file, err := os.ReadFile(exclusionFile)
		if err != nil {
			if !strings.Contains(err.Error(), "no such file or directory") {
				return excludedList, err
			}

			excludedList.Headers = defaultExcludedHeaders
			excludedList.Variables = defaultExcludedVariables

			data, mErr := json.MarshalIndent(excludedList, "", " ")
			if mErr != nil {
				return excludedList, mErr
			}

			mErr = os.WriteFile(exclusionFile, data, os.ModePerm)
			if mErr != nil {
				return excludedList, mErr
			}
		} else {
			err = json.Unmarshal(file, &excludedList)
			if err != nil {
				return excludedList, err
			}

			if len(excludedList.Headers) == 0 || len(excludedList.Variables) == 0 {
				return excludedList, errors.New("exclusion list does not have names")
			}
		}

		set = true
	}

	return excludedList, nil
}
