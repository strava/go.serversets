package httpset

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

var (
	// ErrNoServers is returned when no servers are configured or available.
	ErrNoServers = errors.New("httpset: no servers configured or available")
)

// A Watcher represents how a serverset.Watch is used so it can be stubbed out for tests.
type Watcher interface {
	Endpoints() []string
	Event() <-chan struct{}
	IsClosed() bool
}

// A HTTPSet is a wrapper around the serverset.Watch to handle making requests to a set of servers.
// The same net/http Do, Get, Head, Post and PostForm methods are provided with just the
// the host replaces with one from the list. Servers are picked in a simple round-robin fashion.
type HTTPSet struct {
	Watcher

	// The underlying client to use. Will default to the http.DefaultClient
	HTTPClient *http.Client
	UseHTTPS   bool // if scheme not specified, will use https

	LastEvent  time.Time
	EventCount int

	// This channel will get an event when zookeeper updates things
	// calling SetEndpoints will not trigger this type of event.
	event     chan struct{}
	count     int64
	endpoints []string
}

// New creates a new set of http clients backed by the serverset.
// Can be used to round robin over a set of servers by having watch = nil
// and then calling SetEndpoints with the known set of hosts.
func New(watch Watcher) *HTTPSet {
	httpset := &HTTPSet{
		Watcher:    watch,
		HTTPClient: http.DefaultClient,
		event:      make(chan struct{}, 1),
	}

	if watch != nil {
		// don't trigger and event the first time
		httpset.setEndpoints(watch.Endpoints())

		go func() {
			for {
				select {
				case <-watch.Event():
					httpset.SetEndpoints(watch.Endpoints())
				}

				if watch.IsClosed() {
					break
				}
			}

			watcherClosed()
		}()
	}

	return httpset
}

// for use during testing. Saw this in the net/http standard lib.
var watcherClosed = func() {}

// Do sends an HTTP request to one of the hosts at one of the hosts in the serverset.
// This function behaves the same as the same function in the net/http library,
// but updates the URL.Host to a endpoint (host:port) from the list.
// The hosts are picked in a round-robin fashion.
func (s *HTTPSet) Do(req *http.Request) (*http.Response, error) {
	host, err := s.RotateEndpoint()
	if err != nil {
		return nil, err
	}

	req.URL.Host = host
	return s.HTTPClient.Do(req)
}

// Get issues a GET to the specified URL at one of the hosts in the serverset.
// This function behaves the same as the same function in the net/http library,
// but updates the URL.Host to a endpoint (host:port) from the list.
// The hosts are picked in a round-robin fashion.
func (s *HTTPSet) Get(url string) (*http.Response, error) {
	url, err := s.replaceHost(url)
	if err != nil {
		return nil, err
	}

	return s.HTTPClient.Get(url)
}

// Head issues a HEAD to the specified URL at one of the hosts in the serverset.
// This function behaves the same as the same function in the net/http library,
// but updates the URL.Host to a endpoint (host:port) from the list.
// The hosts are picked in a round-robin fashion.
func (s *HTTPSet) Head(url string) (*http.Response, error) {
	url, err := s.replaceHost(url)
	if err != nil {
		return nil, err
	}

	return s.HTTPClient.Head(url)
}

// Post issues a POST to the specified URL at one of the hosts in the serverset.
// This function behaves the same as the same function in the net/http library,
// but updates the URL.Host to a endpoint (host:port) from the list.
// The hosts are picked in a round-robin fashion.
func (s *HTTPSet) Post(url string, bodyType string, body io.Reader) (*http.Response, error) {
	url, err := s.replaceHost(url)
	if err != nil {
		return nil, err
	}

	return s.HTTPClient.Post(url, bodyType, body)
}

// PostForm issues a POST to the specified URL at one of the hosts in the serverset.
// This function behaves the same as the same function in the net/http library,
// but updates the URL.Host to a endpoint (host:port) from the list.
// The hosts are picked in a round-robin fashion.
func (s *HTTPSet) PostForm(url string, data url.Values) (*http.Response, error) {
	url, err := s.replaceHost(url)
	if err != nil {
		return nil, err
	}

	return s.HTTPClient.PostForm(url, data)
}

// replaceHost replaces the host in the raw url with one from the current list of endpoints.
func (s *HTTPSet) replaceHost(rawurl string) (string, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}

	host, err := s.RotateEndpoint()
	if err != nil {
		return "", err
	}
	u.Host = host
	if u.Scheme == "" {
		if s.UseHTTPS {
			u.Scheme = "https"
		} else {
			u.Scheme = "http"
		}
	}

	return u.String(), nil
}

// Endpoints returns the current endpoints for this service.
// This can be those set via the serverset.Watch or manually via SetEndpoints()
func (s *HTTPSet) Endpoints() []string {
	return s.endpoints
}

// Event returns the event channel. This channel will get an object
// whenever something changes with the list of endpoints.
// Mostly just a passthrough of the underlying watch event.
func (s *HTTPSet) Event() <-chan struct{} {
	return s.event
}

// SetEndpoints sets current list of endpoints. This will override the list
// returned by the serverset. An event by the serverset will override these values.
// This should be used to take advantage of the round robin features of this library without a serverset.Watch.
func (s *HTTPSet) SetEndpoints(endpoints []string) {
	s.setEndpoints(endpoints)
	s.triggerEvent()
}

func (s *HTTPSet) setEndpoints(endpoints []string) {
	// copy the contents,
	// just to be triple sure an external client won't mess with stuff.
	eps := make([]string, len(endpoints), len(endpoints))
	copy(eps, endpoints)

	s.endpoints = eps
}

// RotateEndpoint returns host:port for the endpoints in a round-robin fashion.
func (s *HTTPSet) RotateEndpoint() (string, error) {
	if len(s.endpoints) == 0 {
		return "", ErrNoServers
	}

	c := atomic.AddInt64(&s.count, 1)
	eps := s.endpoints
	return eps[c%int64(len(eps))], nil
}

// triggerEvent, will queue up something in the Event channel if there isn't already something there.
func (s *HTTPSet) triggerEvent() {
	s.EventCount++
	s.LastEvent = time.Now()

	select {
	case s.event <- struct{}{}:
	default:
	}
}
