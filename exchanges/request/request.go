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
	maxHandles  = 27
)

var request service

type service struct {
	exchangeHandlers []*Handler
}

// checkHandles checks to see if there is a handle monitored by the service
func (s *service) checkHandles(exchName string, h *Handler) bool {
	for _, handle := range s.exchangeHandlers {
		if exchName == handle.exchName || handle == h {
			return true
		}
	}
	return false
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

// limit contains the limit rate value which has a Mutex
type limit struct {
	Val time.Duration
	sync.Mutex
}

// getLimitRate returns limit rate with a protected call
func (l *limit) getLimitRate() time.Duration {
	l.Lock()
	defer l.Unlock()
	return l.Val
}

// setLimitRates sets initial limit rates with a protected call
func (l *limit) setLimitRate(rate int) {
	l.Lock()
	l.Val = time.Duration(rate) * time.Millisecond
	l.Unlock()
}

// Handler is a generic exchange specific request handler.
type Handler struct {
	exchName     string
	Client       *http.Client
	shutdown     bool
	LimitAuth    *limit
	LimitUnauth  *limit
	requests     chan *exchRequest
	responses    chan *exchResponse
	timeLockAuth chan int
	timeLock     chan int
	wg           sync.WaitGroup
}

// SetRequestHandler sets initial variables for the request handler and returns
// an error
func (h *Handler) SetRequestHandler(exchName string, authRate, unauthRate int, client *http.Client) error {
	if request.checkHandles(exchName, h) {
		return errors.New("handler already registered for an exchange")
	}

	h.exchName = exchName
	h.Client = client
	h.shutdown = false
	h.LimitAuth = new(limit)
	h.LimitAuth.setLimitRate(authRate)
	h.LimitUnauth = new(limit)
	h.LimitUnauth.setLimitRate(unauthRate)
	h.requests = make(chan *exchRequest, maxJobQueue)
	h.responses = make(chan *exchResponse, 1)
	h.timeLockAuth = make(chan int, 1)
	h.timeLock = make(chan int, 1)

	request.exchangeHandlers = append(request.exchangeHandlers, h)
	h.startWorkers()

	return nil
}

// SetRateLimit sets limit rates for exchange requests
func (h *Handler) SetRateLimit(authRate, unauthRate int) {
	h.LimitAuth.setLimitRate(authRate)
	h.LimitUnauth.setLimitRate(unauthRate)
}

// SendPayload packages a request, sends it to a channel, then a worker executes it
func (h *Handler) SendPayload(method, path string, headers map[string]string, body io.Reader, result interface{}, authRequest, verbose bool) error {
	if h.exchName == "" {
		return errors.New("request handler not initialised")
	}

	method = strings.ToUpper(method)

	if method != "POST" && method != "GET" && method != "DELETE" {
		return errors.New("incorrect method - either POST, GET or DELETE")
	}

	if verbose {
		log.Printf("%s exchange request path: %s", h.exchName, path)
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
		log.Printf("%s exchange raw response: %s", h.exchName, string(contents[:]))
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
			time.Sleep(h.LimitAuth.getLimitRate())
			h.timeLockAuth <- 1
		}
		h.wg.Done()
	}()
	// routine to monitor Unauthenticated limit rates
	go func() {
		h.timeLock <- 1
		for !h.shutdown {
			<-h.timeLock
			time.Sleep(h.LimitUnauth.getLimitRate())
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
			if job.Request.Method != "GET" {
				httpResponse, err = h.Client.Do(job.Request)
			} else {
				httpResponse, err = h.Client.Get(job.Path)
			}
			h.timeLockAuth <- 1
		} else {
			<-h.timeLock
			if job.Request.Method != "GET" {
				httpResponse, err = h.Client.Do(job.Request)
			} else {
				httpResponse, err = h.Client.Get(job.Path)
			}
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
