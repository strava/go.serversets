package mcset

import (
	"errors"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/golang/groupcache/consistenthash"
	"github.com/reusee/mmh3"
)

var (
	// ErrNoServers is returned when no servers are configured or available.
	ErrNoServers = errors.New("mcset: no servers configured or available")

	// DefaultLogger is used by default to print change event messages.
	DefaultLogger Logger = defaultLogger{}
)

// Logger is an interface that can be implemented to provide custom log output.
type Logger interface {
	Printf(string, ...interface{})
}

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

	Logger Logger

	// This channel will get an event when zookeeper updates things
	// calling SetEndpoints will not trigger this type of event.
	event chan struct{}

	consistent *consistenthash.Map

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
		Logger:  DefaultLogger,
		event:   make(chan struct{}, 1),
	}

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

	sort.StringSlice(endpoints).Sort()

	s.Logger.Printf("new endpoints for mcset: %v", endpoints)

	s.consistent = consistenthash.New(150, mmh3.Sum32)
	s.consistent.Add(endpoints...)
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
	if s.consistent == nil {
		return nil, ErrNoServers
	}

	server := s.consistent.Get(key)
	if server == "" {
		return nil, ErrNoServers
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

type defaultLogger struct{}

func (defaultLogger) Printf(format string, a ...interface{}) {
	log.Printf(format, a...)
}
