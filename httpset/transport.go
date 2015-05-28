package httpset

import (
	"net/http"
	"sync/atomic"
	"time"
)

// Transport implements the http.RoundTripper interface loadbalancing
// over a set of hosts.
type Transport struct {
	Watcher

	UseHTTPS bool // if scheme not specified, will use https

	// BaseTransport is what's used after the url is rewritten to the correct host.
	// If not set, http.DefaultTransport will be used.
	BaseTransport http.RoundTripper

	LastEvent  time.Time
	EventCount int

	event     chan struct{}
	count     int64
	endpoints []string
}

// NewTransport creates a new Transport given the server set.
// Pass in nil and use SetEndpoints to balance over a fixed set of endpoints.
func NewTransport(watch Watcher) *Transport {
	t := &Transport{
		Watcher: watch,
		event:   make(chan struct{}, 1),
	}

	if watch != nil {
		// don't trigger an event the first time
		t.setEndpoints(watch.Endpoints())

		go func() {
			for {
				select {
				case <-watch.Event():
					t.SetEndpoints(watch.Endpoints())
				}

				if watch.IsClosed() {
					break
				}
			}

			watcherClosed()
		}()
	}

	return t
}

// for use during testing. Saw this in the net/http standard lib.
var watcherClosed = func() {}

// RoundTrip is here to implement the http.RoundTripper interface so this
// can be used as Transport for an http.Client. It simply rewrites the host
// and passit it to http.DefaultTransport or t.BaseTransport if defined.
// The default transport does it's own connection pooling based on hostname.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	err := t.replaceHost(req)
	if err != nil {
		return nil, err
	}

	if t.BaseTransport == nil {
		return http.DefaultTransport.RoundTrip(req)
	}

	return t.BaseTransport.RoundTrip(req)
}

func (t *Transport) replaceHost(req *http.Request) error {
	host, err := t.RotateEndpoint()
	if err != nil {
		return err
	}

	req.URL.Host = host
	if req.URL.Scheme == "" {
		if t.UseHTTPS {
			req.URL.Scheme = "https"
		} else {
			req.URL.Scheme = "http"
		}
	}

	return nil
}

// Endpoints returns the current endpoints for this service.
// This can be those set via the serverset.Watch or manually via SetEndpoints()
func (t *Transport) Endpoints() []string {
	return t.endpoints
}

// Event returns the event channel. This channel will get an object
// whenever something changes with the list of endpoints.
// Mostly just a passthrough of the underlying watch event and used for testing.
func (t *Transport) Event() <-chan struct{} {
	return t.event
}

// SetEndpoints sets current list of endpoints. This will override the list
// returned by the serverset. An event by the serverset will override these values.
// This should be used to take advantage of the round robin features of this library without a serverset.Watch.
func (t *Transport) SetEndpoints(endpoints []string) {
	t.setEndpoints(endpoints)
	t.triggerEvent()
}

func (t *Transport) setEndpoints(endpoints []string) {
	// copy the contents,
	// just to be triple sure an external client won't mess with stuff.
	eps := make([]string, len(endpoints), len(endpoints))
	copy(eps, endpoints)

	t.endpoints = eps
}

// RotateEndpoint returns host:port for the endpoints in a round-robin fashion.
func (t *Transport) RotateEndpoint() (string, error) {
	if len(t.endpoints) == 0 {
		return "", ErrNoServers
	}

	c := atomic.AddInt64(&t.count, 1)
	eps := t.endpoints
	return eps[c%int64(len(eps))], nil
}

// triggerEvent, will queue up something in the Event channel if there isn't already something there.
func (t *Transport) triggerEvent() {
	t.EventCount++
	t.LastEvent = time.Now()

	select {
	case t.event <- struct{}{}:
	default:
	}
}
