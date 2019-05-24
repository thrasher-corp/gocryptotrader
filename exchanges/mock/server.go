package mock

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"testing"
)

// DefaultDirectory defines the main mock directory
const DefaultDirectory = "../../testdata/http_mock/"

const (
	defaultHost           = ":3000"
	contentType           = "Content-Type"
	applicationURLEncoded = "application/x-www-form-urlencoded"
	applicationJSON       = "application/json"
)

// VCRMock defines the main mock JSON file and attributes
type VCRMock struct {
	Host   string                               `json:"host"`
	Routes map[string]map[string][]HTTPResponse `json:"routes"`
}

// NewVCRServer starts a new VCR server for replaying HTTP requests for testing
// purposes
func NewVCRServer(path string, t *testing.T) error {
	if t == nil {
		return errors.New("this service needs to be utilised in a testing environment")
	}

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	// Get mocking data for the specific service
	var mockFile VCRMock
	err = json.Unmarshal(contents, &mockFile)
	if err != nil {
		return err
	}

	// range over routes and assign responses to explicit paths and http methods
	for pattern, mockResponses := range mockFile.Routes {
		RegisterHandler(pattern, mockResponses)
	}

	go func() {
		log.Fatal(http.ListenAndServe(mockFile.Host, nil))
	}()

	return nil
}

// RegisterHandler registers a generalised mock response logic for specific
// routes
func RegisterHandler(pattern string, mock map[string][]HTTPResponse) {
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		httpResponses, ok := mock[r.Method]
		if !ok {
			log.Fatalf("Mock Test Failure - Method %s not present in mock file",
				r.Method)
		}

		switch r.Method {
		case http.MethodGet:
			vals, err := url.ParseRequestURI(r.RequestURI)
			if err != nil {
				log.Fatal("Mock Test Failure - Parse request URI error", err)
			}

			payload, err := MatchAndGetResponse(httpResponses, vals.Query(), true)
			if err != nil {
				log.Fatalf("Mock Test Failure - MatchAndGetResponse error %s for %s",
					err, r.RequestURI)
			}

			MessageWriteJSON(w, http.StatusOK, payload)
			return

		case http.MethodPost:
			switch r.Header.Get(contentType) {
			case applicationURLEncoded:
				readBody, err := ioutil.ReadAll(r.Body)
				if err != nil {
					log.Fatal("Mock Test Failure - ReadAll error", err)
				}

				vals, err := url.ParseQuery(string(readBody))
				if err != nil {
					log.Fatal("Mock Test Failure - parse query error", err)
				}

				payload, err := MatchAndGetResponse(httpResponses, vals, false)
				if err != nil {
					log.Fatal("Mock Test Failure - MatchAndGetResponse error ", err)
				}

				MessageWriteJSON(w, http.StatusOK, payload)
				return

			case "":
				payload, err := MatchAndGetResponse(httpResponses, r.URL.Query(), true)
				if err != nil {
					log.Fatal("Mock Test Failure - MatchAndGetResponse error ", err)
				}

				MessageWriteJSON(w, http.StatusOK, payload)
				return

			case applicationJSON:
				readBody, err := ioutil.ReadAll(r.Body)
				if err != nil {
					log.Fatalf("Mock Test Failure - %v", err)
				}

				reqVals, err := DeriveURLValsFromJSONMap(readBody)
				if err != nil {
					log.Fatalf("Mock Test Failure - %v", err)
				}

				payload, err := MatchAndGetResponse(httpResponses, reqVals, false)
				if err != nil {
					log.Fatal("Mock Test Failure - MatchAndGetResponse error ", err)
				}

				MessageWriteJSON(w, http.StatusOK, payload)
				return

			default:
				log.Fatalf("Mock Test Failure - Unhandled content type %v",
					r.Header.Get(contentType))
			}

		case http.MethodDelete:
			payload, err := MatchAndGetResponse(httpResponses, r.URL.Query(), true)
			if err != nil {
				log.Println(r.URL.Query())
				log.Fatal("Mock Test Failure - MatchAndGetResponse error ", err)
			}

			MessageWriteJSON(w, http.StatusOK, payload)
			return

		default:
			log.Fatal("Mock Test Failure - Unhandled HTTP method:",
				r.Header.Get(contentType))
		}

		MessageWriteJSON(w, http.StatusNotFound, "Unhandle Request")
	})
}

// MessageWriteJSON writes JSON to a connection
func MessageWriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set(contentType, applicationJSON)
	w.WriteHeader(status)
	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			log.Fatal("Mock Test Failure - JSON encode error", err)
		}
	}
}

// MatchAndGetResponse matches incoming request values with mockdata response
// values and returns the payload
func MatchAndGetResponse(mockData []HTTPResponse, requestVals url.Values, isQueryData bool) (json.RawMessage, error) {
	for i := range mockData {
		var data string
		if isQueryData {
			data = mockData[i].QueryString
		} else {
			data = mockData[i].BodyParams
		}

		mockVals, err := url.ParseQuery(data)
		if err != nil {
			return nil, err
		}

		if MatchURLVals(mockVals, requestVals) {
			return mockData[i].Data, nil
		}
	}
	return nil, errors.New("no data could be matched")
}
