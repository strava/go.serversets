package fixedset

import (
	"sort"
	"sync"
	"time"
)

// FixedSet can be used as a fixed set of endpoints for testing or
// to use the load balancing functions without zookeeper.
type FixedSet struct {
	LastEvent  time.Time
	EventCount int
	event      chan struct{}

	endpoints []string
	lock      sync.RWMutex
	done      chan struct{}
}

// New creates a new FixedSet with the given endpoints.
func New(endpoints []string) *FixedSet {
	fs := &FixedSet{
		event: make(chan struct{}, 1),
		done:  make(chan struct{}),
	}
	fs.setEndpoints(endpoints)

	return fs
}

// Endpoints returns a slice of the current list of servers/endpoints associated with this watch.
func (fs *FixedSet) Endpoints() []string {
	fs.lock.RLock()
	defer fs.lock.RUnlock()

	return fs.endpoints
}

// SetEndpoints sets current list of endpoints.
func (fs *FixedSet) SetEndpoints(endpoints []string) {
	fs.setEndpoints(endpoints)
	fs.triggerEvent()
}

func (fs *FixedSet) setEndpoints(endpoints []string) {
	fs.lock.Lock()
	defer fs.lock.Unlock()

	fs.endpoints = make([]string, len(endpoints))
	sort.Strings(endpoints)
	copy(fs.endpoints, endpoints)
}

// Event returns the event channel. This channel will get an object
// whenever something changes with the list of endpoints.
func (fs *FixedSet) Event() <-chan struct{} {
	return fs.event
}

// Close for this service just sets a boolean since there isn't a lot of async stuff going on.
func (fs *FixedSet) Close() {
	select {
	case <-fs.done:
		return
	default:
	}

	close(fs.done)
	close(fs.event)
}

// IsClosed returns if this fixed set has been closed.
func (fs *FixedSet) IsClosed() bool {
	select {
	case <-fs.done:
		return true
	default:
	}

	return false
}

// triggerEvent will queue up something in the Event channel if there isn't already something there.
func (fs *FixedSet) triggerEvent() {
	fs.EventCount++
	fs.LastEvent = time.Now()

	select {
	case fs.event <- struct{}{}:
	default:
	}
}
