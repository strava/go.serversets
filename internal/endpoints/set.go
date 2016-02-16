package endpoints

import (
	"errors"
	"io"
	"math/rand"
	"sync"
	"time"
)

var (
	// ErrNoEndpoints is returned when no endpoints are configured or available.
	ErrNoEndpoints = errors.New("endpoints: no endpoints configured or available")

	// ErrGetOnClosedEndpoint is returned by endpoint.GetConn if the endpoint
	// has been closed because it was removed from the watch.
	// This error is retryable.
	ErrGetOnClosedEndpoint = errors.New("endpoint closed")
)

// Pooler can be throught of as a parent wrapper to a bunch of endpoints.
// It defines how connections are opened. And the general behavior of the pool.
type Pooler interface {
	// OpenConn creates the network connection to the host.
	// This could be tcp, udp, higher level thrift, etc.
	OpenConn(host string) (io.Closer, error)

	IdleTimeout() time.Duration
	MaxActivePerHost() int
	MaxIdlePerHost() int
}

// Set contains a list of endpoints and defines
// some operation on top of it.
type Set struct {
	Pooler Pooler
	lock   sync.RWMutex
	list   []*endpoint
}

// NewSet creates a new endpoint set.
func NewSet(pooler Pooler) *Set {
	return &Set{
		Pooler: pooler,
	}
}

// Close will close all the hosts and thus all the connections
// for all the endpoints.
func (s *Set) Close() error {
	s.lock.Lock()
	list := s.list
	s.lock.Unlock()

	for _, ep := range list {
		ep.Close()
	}

	return nil
}

// GetConn returns a connection from the endpoint with the current
// least amount of active connections.
func (s *Set) GetConn() (*Conn, error) {
	s.lock.RLock()
	if len(s.list) == 0 {
		s.lock.RUnlock()
		return nil, ErrNoEndpoints
	}

	// math.MaxInt32, just greater than the practical maximum for active connections
	min := 1<<31 - 1
	var minEP *endpoint

	// TODO: would be interesting to implement a min-heap here.
	for _, ep := range s.list {
		if ep.IsClosed() {
			continue
		}

		if a := ep.ActiveConnections(); a < min {
			min = a
			minEP = ep
		}
	}
	s.lock.RUnlock()

	if minEP == nil {
		return nil, ErrNoEndpoints
	}

	c, err := minEP.GetConn()
	if err == ErrGetOnClosedEndpoint {
		// This is a super tight race condition with the above 5 lines.
		// TODO: figure out if we should retry? or handle in application code?
	}

	return c, err
}

// SetEndpoints will do a smart update of the endpoint lists. New hosts
// will be added, old ones will be removed.
func (s *Set) SetEndpoints(hosts []string) (added, removed int) {
	shuffleHosts(hosts)
	s.lock.Lock()

	// Remove hosts that are currently in the endpoints list, but shouldn't be.
	// Is there a cleaner implementation of this?
	var toRemove []*endpoint
	for i := len(s.list) - 1; i >= 0; i-- {
		ep := s.list[i]
		found := false
		for _, host := range hosts {
			if ep.Host() == host {
				found = true
				break
			}
		}

		if !found {
			toRemove = append(toRemove, s.list[i])

			s.list[i] = s.list[len(s.list)-1]
			s.list = s.list[:len(s.list)-1]
		}
	}

	// Add in hosts that are not currently in the endpoints list.
	for _, host := range hosts {
		found := false
		for _, ep := range s.list {
			if ep.Host() == host {
				found = true
				break
			}
		}

		if !found {
			s.list = append(s.list, newEndpoint(s.Pooler, host))
			added++
		}
	}

	s.lock.Unlock()

	for _, e := range toRemove {
		e.Close()
	}

	return added, len(toRemove) // return value is used for testing
}

func shuffleHosts(hosts []string) {
	// TODO: maybe not recreate the random source everytime.
	r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	for i := len(hosts) - 1; i > 0; i-- {
		j := r.Intn(i)
		hosts[i], hosts[j] = hosts[j], hosts[i]
	}
}
