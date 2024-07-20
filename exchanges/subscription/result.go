package subscription

import "sync"

// Result stores the result of a subscription request
type Result struct {
	store map[*Subscription]error
	m     sync.Mutex
}

// Add adds a subscription to the result store
func (r *Result) Add(sub *Subscription, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.store == nil {
		r.store = make(map[*Subscription]error)
	}
	r.store[sub] = err
}

// GetSuccessful returns a list of successful subscriptions
func (r *Result) GetSuccessful() List {
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
	r.m.Lock()
	defer r.m.Unlock()
	out := make(map[*Subscription]error)
	for sub, err := range r.store {
		if err == nil {
			continue
		}
		out[sub] = err
	}
	return r.store
}
