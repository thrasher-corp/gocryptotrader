package subscription

import "sync"

// Result stores the result of a subscription request, this is helpful when you need concurrent subscription requests
// and need to know which ones failed.
type Result struct {
	store map[*Subscription]error
	wg    sync.WaitGroup
	m     sync.Mutex
}

// Add adds a subscription to the result store
func (r *Result) add(sub *Subscription, err error) {
	if r == nil || sub == nil {
		return
	}
	r.m.Lock()
	defer r.m.Unlock()
	if r.store == nil {
		r.store = make(map[*Subscription]error)
	}
	r.store[sub] = err
}

// GetSuccessful returns a list of successful subscriptions
func (r *Result) GetSuccessful() List {
	if r == nil {
		return List{}
	}
	r.m.Lock()
	defer r.m.Unlock()
	list := make(List, 0, len(r.store))
	for sub, err := range r.store {
		if err != nil {
			continue
		}
		list = append(list, sub)
	}
	return list
}

// GetUnsuccessful returns a map of failed subscriptions
func (r *Result) GetUnsuccessful() map[*Subscription]error {
	if r == nil {
		return make(map[*Subscription]error)
	}
	r.m.Lock()
	defer r.m.Unlock()
	out := make(map[*Subscription]error)
	for sub, err := range r.store {
		if err == nil {
			continue
		}
		out[sub] = err
	}
	return out
}

// Action is a function that returns an error for specific subscription actions
type Action func() error

// RunRoutine runs the subscription routine
func (r *Result) RunRoutine(s *Subscription, subscribeOrUnsubscribe Action) {
	r.wg.Add(1)
	go func() {
		r.add(s, subscribeOrUnsubscribe())
		r.wg.Done()
	}()
}

// ReturnWhenFinished waits for all routines to finish and returns the result
func (r *Result) ReturnWhenFinished() *Result {
	r.wg.Wait()
	return r
}
