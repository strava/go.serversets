package endpoints

import (
	"container/list"
	"sync"
	"time"
)

// An endpoint contains the active and idle connection pools to the host.
type endpoint struct {
	Pooler Pooler
	host   string
	active int

	lock sync.RWMutex
	cond *sync.Cond

	idle *list.List
	done chan struct{}
}

// New creates a new endpoint for the given set and hostname.
func newEndpoint(pooler Pooler, host string) *endpoint {
	return &endpoint{
		Pooler: pooler,
		host:   host,
		idle:   list.New(),
		done:   make(chan struct{}),
	}
}

// Host returns the host (ip or domain) this endpoint represents.
func (ep *endpoint) Host() string {
	return ep.host
}

// GetConn will create a connection or return one from the idle list.
func (ep *endpoint) GetConn() (*Conn, error) {
	if ep.IsClosed() {
		return nil, ErrGetOnClosedEndpoint
	}

	// close stale connections
	if timeout := ep.Pooler.IdleTimeout(); timeout > 0 {
		ep.closeIdleConn(time.Now().Add(-timeout))
	}

	ep.lock.Lock()
	for {
		// try to get a connection from the idle list.
		for i, n := 0, ep.idle.Len(); i < n; i++ {
			elm := ep.idle.Front()
			if elm == nil {
				break
			}

			idled := elm.Value.(idleConn)
			ep.idle.Remove(elm)

			ep.active++
			ep.lock.Unlock()
			return idled.conn, nil
		}

		// Check for pool closed before dialing a new connection.
		if ep.IsClosed() {
			ep.lock.Unlock()
			return nil, ErrGetOnClosedEndpoint
		}

		// create a new connection if under limit.
		if maph := ep.Pooler.MaxActivePerHost(); maph == 0 || ep.active < maph {
			ep.active++
			ep.lock.Unlock()
			return ep.newConn()
		}

		if ep.cond == nil {
			ep.cond = sync.NewCond(&ep.lock)
		}

		// wait for a signal from a close event or a release event.
		// This will unlock the lock and relock it once it returns. See sync.Cond docs.
		ep.cond.Wait()
	}
}

// ActiveConnections returns the number of connections current in use.
func (ep *endpoint) ActiveConnections() int {
	ep.lock.RLock()
	defer ep.lock.RUnlock()

	return ep.active
}

// Close will close all the connections associated with this host.
func (ep *endpoint) Close() error {
	if ep.IsClosed() {
		return nil
	}

	ep.lock.Lock()
	idle := ep.idle
	ep.idle.Init()
	close(ep.done)

	ep.active -= ep.idle.Len()

	if ep.cond != nil {
		// everyone waiting for a connection should now wake
		// and error out since the pool is now closed.
		ep.cond.Broadcast()
	}

	// release the lock before we close all the idle connections
	ep.lock.Unlock()

	for elm := idle.Front(); elm != nil; elm = elm.Next() {
		elm.Value.(idleConn).conn.Conn.Close()
	}

	return nil
}

// IsClosed returns true if the endpoint has been closed.
// There still might be active connections in flight but they will
// be closed as they are released.
func (ep *endpoint) IsClosed() bool {
	select {
	case <-ep.done:
		return true
	default:
	}

	return false
}

// ReturnConn puts the connection back in the pool.
func (ep *endpoint) ReturnConn(conn *Conn) error {
	ep.lock.Lock()

	ep.active--
	if ep.IsClosed() {
		ep.lock.Unlock()
		return conn.Close()
	}

	ep.idle.PushFront(idleConn{
		conn:      conn,
		lastUsage: time.Now(),
	})

	var idled *Conn
	if m := ep.Pooler.MaxIdlePerHost(); m != 0 && ep.idle.Len() > m {
		idled = ep.idle.Remove(ep.idle.Back()).(idleConn).conn
	}

	if idled == nil {
		if ep.cond != nil {
			ep.cond.Signal()
		}

		ep.lock.Unlock()
		return nil
	}

	ep.lock.Unlock() // release the lock before we close the
	return idled.Close()
}

// RemoveAndClose will close this connection, remove it from the "active" pool
// and close it. It will not be put in the idle connection pool.
func (ep *endpoint) RemoveAndClose(conn *Conn) error {
	ep.lock.Lock()
	ep.active--
	ep.lock.Unlock()

	return conn.Conn.Close()
}

func (ep *endpoint) closeIdleConn(limit time.Time) int {
	ep.lock.Lock()
	defer ep.lock.Unlock()

	closed := 0

	// since idle connections are oldest at the back.
	// close until we find one not idle long enough.
	for i, n := 0, ep.idle.Len(); i < n; i++ {
		elm := ep.idle.Back()
		if elm == nil {
			break
		}

		c := elm.Value.(idleConn)
		if c.lastUsage.After(limit) {
			break
		}

		ep.idle.Remove(elm)

		ep.lock.Unlock()
		c.conn.Close() // release the lock while we close
		closed++

		ep.lock.Lock()
		// There can be a quick lock/unlock at the end here
		// but that will only happen if all idle connections are closed.
		// Alternately there could be a quick lock/unlock to get n := ep.idle.Len()
		// but that would happen on every call to this function.
	}

	return closed
}

// newConn creates a new connection to this endpoint
func (ep *endpoint) newConn() (*Conn, error) {
	socket, err := ep.Pooler.OpenConn(ep.host)
	if err != nil {
		return nil, err
	}

	return &Conn{
		Endpoint: ep.host,
		ep:       ep,
		Conn:     socket,
	}, nil
}
