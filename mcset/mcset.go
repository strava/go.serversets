package mcset

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/strava/go.serversets/mcset/consistent"
)

var (
	// ErrNoServers is returned when no servers are configured or available.
	ErrNoServers = errors.New("mcset: no servers configured or available")
)

// A Watcher represents how a serverset.Watch is used so it can be stubbed out for tests.
type Watcher interface {
	Endpoints() []string
	Event() <-chan struct{}
	IsClosed() bool
}

// A MCSet is a wrapper around the serverset.Watch to handle the memcache use case.
// Basically provides some helper functions to pick the servers consistently.
type MCSet struct {
	Watcher

	LastEvent  time.Time
	EventCount int

	// This channel will get an event when zookeeper updates things
	// calling SetEndpoints will not trigger this type of event.
	event chan struct{}

	consistent *consistent.Consistent

	lock      sync.Mutex
	endpoints []string
	addresses map[string]net.Addr
}

// New creates a new memcache server set.
// Can be used to just consistently hash keys to a known set of servers by
// having watch = nil and then calling SetEndpoints with the known set of memcache hosts.
func New(watch Watcher) *MCSet {
	mcset := &MCSet{
		Watcher: watch,

		event:      make(chan struct{}, 1),
		consistent: consistent.New(),
	}

	mcset.consistent.NumberOfReplicas = 100

	if watch != nil {
		// first time don't trigger an event
		mcset.setEndpoints(watch.Endpoints())

		go func() {
			for {
				select {
				case <-watch.Event():
					mcset.SetEndpoints(watch.Endpoints())
				}

				if watch.IsClosed() {
					break
				}
			}

			watcherClosed()
		}()
	}

	return mcset
}

// for use during testing. Saw this in the net/http standard lib.
var watcherClosed = func() {}

// SetEndpoints sets current list of endpoints. This will override the list
// returned by the serverset. An event by the serverset will override these values.
func (s *MCSet) SetEndpoints(endpoints []string) {
	s.setEndpoints(endpoints)
	s.triggerEvent()
}

func (s *MCSet) setEndpoints(endpoints []string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	addresses := make(map[string]net.Addr)
	for _, e := range endpoints {

		a, err := net.ResolveTCPAddr("tcp", e)
		if err != nil {
			// TODO: if the hostname doesn't resolve, what should we do?
			// panic(err)
		}

		addresses[e] = a
	}

	s.addresses = addresses
	s.endpoints = endpoints

	s.consistent.Set(endpoints)
}

// Endpoints returns the current endpoints for this service.
// This can be those set via the serverset.Watch or manually via SetEndpoints()
func (s *MCSet) Endpoints() []string {
	return s.endpoints
}

// Event returns the event channel. This channel will get an object
// whenever something changes with the list of endpoints.
// Mostly just a passthrough of the underlying watch event.
func (s *MCSet) Event() <-chan struct{} {
	return s.event
}

// PickServer consistently picks a server from the list.
// Kind of a weird signature but is necessary to satisfy the memcache.ServerSelector interface.
func (s *MCSet) PickServer(key string) (net.Addr, error) {
	if len(s.consistent.Members()) == 0 {
		return nil, ErrNoServers
	}

	server, err := s.consistent.Get(key)
	if err != nil {
		return nil, err
	}
	return s.addresses[server], nil
}

// Each runs the function over each server currently in the set.
// Kind of a weird signature but is necessary to satisfy the memcache.ServerSelector interface.
func (s *MCSet) Each(f func(net.Addr) error) error {
	addresses := s.addresses

	for _, a := range addresses {
		if err := f(a); nil != err {
			return err
		}
	}

	return nil
}

// triggerEvent, will queue up something in the Event channel if there isn't already something there.
func (s *MCSet) triggerEvent() {
	s.EventCount++
	s.LastEvent = time.Now()

	select {
	case s.event <- struct{}{}:
	default:
	}
}
