package httpset

import (
	"errors"
	"net/http"
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
// It encapsulates a http.Client using a httpset.Transport that does all the balancing.
// This object is DEPRECATED, one should use Transport and build their own http.Clients.
type HTTPSet struct {
	*Transport
	*http.Client
}

// New creates a new set of http clients backed by the serverset.
// This object is DEPRECATED, one should use Transport and build their own http.Clients.
func New(watch Watcher) *HTTPSet {
	transport := NewTransport(watch)
	return &HTTPSet{
		Transport: transport,
		Client: &http.Client{
			Transport: transport,
		},
	}
}
