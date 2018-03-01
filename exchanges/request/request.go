package request

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	maxJobQueue = 100
)

var request service

type service struct {
	exchangeHandlers []*Handler
}

// checkHandles checks to see if there is a handle monitored by the service
func (s *service) checkHandles(exchName string) *Handler {
	for _, handle := range s.exchangeHandlers {
		if exchName == handle.exchName {
			return handle
		}
	}
	return nil
}

// removeHandle releases handle from service
func (s *service) removeHandle(exchName string) bool {
	for i, handle := range s.exchangeHandlers {
		if exchName == handle.exchName {
			handle.shutdown = true
			handle.wg.Wait()
			new := append(s.exchangeHandlers[:i-1], s.exchangeHandlers[i+1:]...)
			s.exchangeHandlers = new
			return true
		}
	}
	return false
}

// Handler is a generic exchange specific request handler.
type Handler struct {
	exchName     string
	Client       *http.Client
	shutdown     bool
	LimitAuth    time.Duration
	LimitUnauth  time.Duration
	requests     chan *exchRequest
	responses    chan *exchResponse
	timeLockAuth chan int
	timeLock     chan int
	wg           sync.WaitGroup
}

// GetRequestHandler returns a pointer to a requestHandler service.
func GetRequestHandler(exchName string, authRate, unauthRate int, client *http.Client) *Handler {
	if handle := request.checkHandles(exchName); handle != nil {
		return handle
	}

	h := Handler{
		exchName:     exchName,
		Client:       client,
		shutdown:     false,
		LimitAuth:    time.Duration(authRate) * time.Millisecond,
		LimitUnauth:  time.Duration(unauthRate) * time.Millisecond,
		requests:     make(chan *exchRequest, maxJobQueue),
		responses:    make(chan *exchResponse, 1),
		timeLockAuth: make(chan int, 1),
		timeLock:     make(chan int, 1),
	}

	request.exchangeHandlers = append(request.exchangeHandlers, &h)
	h.startWorkers()

	return &h
}

// SetRateLimit sets limit rates for exchange requests
func (h *Handler) SetRateLimit(authRate, unauthRate int) {
	h.LimitAuth = time.Duration(authRate) * time.Millisecond
	h.LimitUnauth = time.Duration(unauthRate) * time.Millisecond
}

// Send packages a request, sends it to a channel, then a worker executes it
func (h *Handler) Send(method, path string, headers map[string]string, body io.Reader, result interface{}, authRequest, verbose bool) error {
	method = strings.ToUpper(method)

	if method != "POST" && method != "GET" && method != "DELETE" {
		return errors.New("incorrect method - either POST, GET or DELETE")
	}

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	err = h.attachJob(req, path, authRequest)
	if err != nil {
		return err
	}

	contents, err := h.getResponse()
	if err != nil {
		return err
	}

	if verbose {
		log.Println("RAW RESP: ", string(contents[:]))
	}

	return common.JSONDecode(contents, result)
}

func (h *Handler) startWorkers() {
	h.wg.Add(3)
	go h.requestWorker()

	// routine to monitor Autheticated limit rates
	go func() {
		h.timeLockAuth <- 1
		for !h.shutdown {
			<-h.timeLockAuth
			time.Sleep(h.LimitAuth)
			h.timeLockAuth <- 1
		}
		h.wg.Done()
	}()
	// routine to monitor Unauthenticated limit rates
	go func() {
		h.timeLock <- 1
		for !h.shutdown {
			<-h.timeLock
			time.Sleep(h.LimitUnauth)
			h.timeLock <- 1
		}
		h.wg.Done()
	}()
}

// requestWorker handles the request queue
func (h *Handler) requestWorker() {
	for job := range h.requests {
		if h.shutdown {
			break
		}

		var httpResponse *http.Response
		var err error

		if job.Auth {
			<-h.timeLockAuth
			httpResponse, err = h.Client.Do(job.Request)
			h.timeLockAuth <- 1
		} else {
			<-h.timeLock
			httpResponse, err = h.Client.Get(job.Path)
			h.timeLock <- 1
		}

		for b := false; !b; {
			select {
			case h.responses <- &exchResponse{Response: httpResponse, ResError: err}:
				b = true
			default:
				continue
			}
		}
	}
	h.wg.Done()
}

// exchRequest is the request type
type exchRequest struct {
	Request *http.Request
	Path    string
	Auth    bool
}

// attachJob sends a request using the http package to the request channel
func (h *Handler) attachJob(req *http.Request, path string, isAuth bool) error {
	select {
	case h.requests <- &exchRequest{Request: req, Path: path, Auth: isAuth}:
		return nil
	default:
		return errors.New("job queue exceeded")
	}
}

// exchResponse is the main response type for requests
type exchResponse struct {
	Response *http.Response
	ResError error
}

// getResponse monitors the current resp channel and returns the contents
func (h *Handler) getResponse() ([]byte, error) {
	resp := <-h.responses
	if resp.ResError != nil {
		return []byte(""), resp.ResError
	}

	defer resp.Response.Body.Close()
	contents, err := ioutil.ReadAll(resp.Response.Body)
	if err != nil {
		return []byte(""), err
	}
	return contents, nil
}
