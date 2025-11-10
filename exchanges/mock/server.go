package mock

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// DefaultDirectory defines the main mock directory
const DefaultDirectory = "../../testdata/http_mock/"

const (
	contentType           = "Content-Type"
	applicationURLEncoded = "application/x-www-form-urlencoded"
	applicationJSON       = "application/json"
	textPlain             = "text/plain"
)

// error declarations
var (
	errJSONMockFilePathRequired = errors.New("no path to json mock file found")
)

// VCRMock defines the main mock JSON file and attributes
type VCRMock struct {
	Routes map[string]map[string][]HTTPResponse `json:"routes"`
}

// NewVCRServer starts a new VCR server for replaying HTTP requests for testing
// purposes and returns the server connection details
func NewVCRServer(path string) (string, *http.Client, error) {
	if path == "" {
		return "", nil, errJSONMockFilePathRequired
	}

	var mockFile VCRMock

	contents, err := os.ReadFile(path)
	if err != nil {
		pathing := strings.Split(path, "/")
		dirPathing := pathing[:len(pathing)-1]
		dir := strings.Join(dirPathing, "/")
		err = common.CreateDir(dir)
		if err != nil {
			return "", nil, err
		}

		data, jErr := json.MarshalIndent(mockFile, "", " ")
		if jErr != nil {
			return "", nil, jErr
		}

		err = file.Write(path, data)
		if err != nil {
			return "", nil, err
		}
		contents = data
	}

	if !json.Valid(contents) {
		return "",
			nil,
			fmt.Errorf("contents of file %s are not valid JSON", path)
	}

	// Get mocking data for the specific service
	err = json.Unmarshal(contents, &mockFile)
	if err != nil {
		return "", nil, err
	}

	newMux := http.NewServeMux()
	// Range over routes and assign responses to explicit paths and http
	// methods
	if len(mockFile.Routes) != 0 {
		for pattern, mockResponses := range mockFile.Routes {
			RegisterHandler(pattern, mockResponses, newMux)
		}
	} else {
		newMux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			err := json.NewEncoder(w).Encode("There is no mock data available in file please record a new HTTP response. Please follow README.md in the mock package.")
			if err != nil {
				panic(err)
			}
		})
	}
	tlsServer := httptest.NewTLSServer(newMux)

	return tlsServer.URL, tlsServer.Client(), nil
}

// RegisterHandler registers a generalised mock response logic for specific
// routes
func RegisterHandler(pattern string, mock map[string][]HTTPResponse, mux *http.ServeMux) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
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

		case http.MethodPost, http.MethodPut:
			switch r.Header.Get(contentType) {
			case applicationURLEncoded:
				readBody, err := io.ReadAll(r.Body)
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
				readBody, err := io.ReadAll(r.Body)
				if err != nil {
					log.Fatalf("Mock Test Failure - %v", err)
				}

				reqVals, err := DeriveURLValsFromJSONMap(readBody)
				if err != nil {
					log.Fatalf("DeriveURLValsFromJSONMap Mock Test Failure - %v", err)
				}

				payload, err := MatchAndGetResponse(httpResponses, reqVals, false)
				if err != nil {
					log.Fatal("Mock Test Failure - MatchAndGetResponse error ", err)
				}

				MessageWriteJSON(w, http.StatusOK, payload)
				return

			case textPlain:
				headerData, ok := r.Header["X-Gemini-Payload"]
				if !ok {
					log.Fatal("Mock Test Failure - Cannot find header in request")
				}

				jsonThings, err := base64.StdEncoding.DecodeString(strings.Join(headerData, ""))
				if err != nil {
					log.Fatal("Mock Test Failure - ", err)
				}

				reqVals, err := DeriveURLValsFromJSONMap(jsonThings)
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
	})
}

// MessageWriteJSON writes JSON to a connection
func MessageWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set(contentType, applicationJSON)
	w.WriteHeader(status)
	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, wErr := w.Write([]byte(err.Error()))
			if wErr != nil {
				log.Println("Mock Test Failure - Writing to HTTP connection", wErr)
			}
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

		mockVals := url.Values{}
		var err error
		if json.Valid([]byte(data)) {
			something := make(map[string]any)
			err = json.Unmarshal([]byte(data), &something)
			if err != nil {
				return nil, err
			}

			for k, v := range something {
				switch val := v.(type) {
				case string:
					mockVals.Add(k, val)
				case bool:
					mockVals.Add(k, strconv.FormatBool(val))
				case float64:
					mockVals.Add(k, strconv.FormatFloat(val, 'f', -1, 64))
				case map[string]any, []any, nil:
					mockVals.Add(k, fmt.Sprintf("%v", val))
				default:
					log.Println(reflect.TypeOf(val))
					log.Fatal("unhandled type please add as needed")
				}
			}
		} else {
			mockVals, err = url.ParseQuery(data)
			if err != nil {
				return nil, err
			}
		}

		if MatchURLVals(mockVals, requestVals) {
			return mockData[i].Data, nil
		}
	}
	return nil, errors.New("no data could be matched")
}
