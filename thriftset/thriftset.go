package thriftset

import (
	"errors"
	"io"
	"time"

	"github.com/strava/go.serversets/internal/endpoints"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/strava/go.statsd"
)

const (
	sdConnRequested = "thriftset.conn.requested"
	sdConnCreated   = "thriftset.conn.created"
	sdConnCreateErr = "thriftset.conn.create_error"
	sdConnReleased  = "thriftset.conn.released"
	sdConnClosed    = "thriftset.conn.closed"
	sdZKEvent       = "thriftset.zk_event"
)

// Default values for ThriftSet parameters
const (
	DefaultMaxIdlePerHost   = 10
	DefaultMaxActivePerHost = 10
	DefaultTimeout          = time.Second
	DefaultIdleTimeout      = 5 * time.Minute
)

var (
	// ErrNoEndpoints is returned when no endpoints are configured or available.
	ErrNoEndpoints = errors.New("thriftset: no endpoints configured or available")

	// ErrGetOnClosedSet is returned when requesting a connection from a closed thrift set.
	ErrGetOnClosedSet = errors.New("thriftset: get on closed set")

	// ErrGetOnClosedEndpoint is returned by endpoint.GetConn if the endpoint
	// has been closed because it was removed from the watch.
	// This error is retryable.
	ErrGetOnClosedEndpoint = errors.New("endpoint closed")

	// errMap maps the internal endpoints errors to these exported ones.
	errMap = map[error]error{
		endpoints.ErrNoEndpoints:         ErrNoEndpoints,
		endpoints.ErrGetOnClosedEndpoint: ErrGetOnClosedEndpoint,
	}
)

// A Watcher represents how a serverset.Watch is used so we can use the zookeeper kind
// or a fixed set or stub it out for testing.
type Watcher interface {
	Endpoints() []string
	Event() <-chan struct{}
	IsClosed() bool
}

// ThriftSet defines a set of thift connections. It loadbalances over
// the set of hosts using the "least active connections" strategy.
type ThriftSet struct {
	watch Watcher

	LastEvent  time.Time
	EventCount int
	StatsD     statsd.Stater

	maxIdlePerHost   int // max idle must be >= max active
	maxActivePerHost int
	timeout          time.Duration
	idleTimeout      time.Duration

	endpoints *endpoints.Set

	// This channel will get an event when zookeeper updates things
	// calling SetEndpoints will not trigger this type of event.
	event         chan struct{}
	watcherClosed chan struct{}
	done          chan struct{}
}

// Conn is the connection returned by the pool.
type Conn struct {
	thriftset *ThriftSet // just used for statsd at this point, really needed?
	parent    *endpoints.Conn

	Socket *thrift.TSocket // used to create new clients
	Client interface{}     // a place to cache thrift clients.
}

// New creates a new ThriftSet with default parameters.
func New(watch Watcher) *ThriftSet {
	ts := &ThriftSet{
		watch:         watch,
		event:         make(chan struct{}, 1),
		watcherClosed: make(chan struct{}, 1),

		StatsD: statsd.NoopClient{},

		maxIdlePerHost:   DefaultMaxIdlePerHost,
		maxActivePerHost: DefaultMaxActivePerHost,
		timeout:          DefaultTimeout,
		idleTimeout:      DefaultIdleTimeout,

		done: make(chan struct{}),
	}

	ts.endpoints = endpoints.NewSet(ts)
	ts.resetEndpoints()

	go func() {
		defer func() {
			close(ts.watcherClosed) // the Close method waits for this goroutine to quit.
			watcherClosed()
		}()

		for {
			select {
			case <-ts.done:
				return
			case <-watch.Event():
				ts.StatsD.Count(sdZKEvent, 1.0)

				ts.resetEndpoints()
				ts.triggerEvent()
			}

			if watch.IsClosed() {
				break
			}
		}
	}()

	return ts
}

// for use during testing. Saw this in the net/http standard lib.
var watcherClosed = func() {}

// OpenConn creats a new thrift socket for the host. This is used
// by the endpoints.Set to create connections for a given endpoint.
// This is part of the endpoints.Pooler interface.
func (ts *ThriftSet) OpenConn(hostPort string) (io.Closer, error) {
	ts.StatsD.Count(sdConnCreated)

	socket, err := socketBuilder(hostPort, ts.timeout)
	if err != nil {
		ts.StatsD.Count(sdConnCreateErr, 1.0)
		return nil, err
	}

	return socket, err
}

// to allow stubbing for tests
var socketBuilder = func(hostPort string, timeout time.Duration) (*thrift.TSocket, error) {
	return thrift.NewTSocketTimeout(hostPort, timeout)
}

// IdleTimeout returns the timeout for connections to live in the idle pool.
// This is part of the endpoints.Pooler interface.
func (ts *ThriftSet) IdleTimeout() time.Duration {
	return ts.idleTimeout
}

// SetIdleTimeout sets the amount of time a connection can live in the idle pool.
func (ts *ThriftSet) SetIdleTimeout(t time.Duration) {
	ts.idleTimeout = t
}

// MaxActivePerHost returna the max active connections for a given host.
// This is part of the endpoints.Pooler interface.
func (ts *ThriftSet) MaxActivePerHost() int {
	return ts.maxActivePerHost
}

// SetMaxActivePerHost sets the max number of active connections to a given endpoint.
func (ts *ThriftSet) SetMaxActivePerHost(max int) {
	ts.maxActivePerHost = max
}

// MaxIdlePerHost returns the max number of idle connections to keep in the pool.
// This is part of the endpoints.Pooler interface.
func (ts *ThriftSet) MaxIdlePerHost() int {
	return ts.maxIdlePerHost
}

// SetMaxIdlePerHost sets the max number of connections in the idle pool.
func (ts *ThriftSet) SetMaxIdlePerHost(max int) {
	ts.maxIdlePerHost = max
}

// Timeout is the max length for a given request to the thrift service.
func (ts *ThriftSet) Timeout() time.Duration {
	return ts.timeout
}

// SetTimeout sets the thrift request timeout for new connections in the pool.
// This should be set at startup, as existing/live connections will not be updated
// with these new values.
func (ts *ThriftSet) SetTimeout(t time.Duration) {
	ts.timeout = t
}

// Event returns the event channel. This channel will get an object
// whenever something changes with the list of endpoints.
// Mostly used for testing as this will trigger after all the watch events handling completes.
func (ts *ThriftSet) Event() <-chan struct{} {
	return ts.event
}

// Close releases the resources used by the set. ie. closes all connections.
// There may still be connections in flight when this function returns.
func (ts *ThriftSet) Close() error {
	if ts.IsClosed() {
		return nil
	}
	close(ts.done)

	err := ts.endpoints.Close()
	<-ts.watcherClosed // wait for the watcher goroutine to quit

	return err
}

// IsClosed returns true if the set has been closed.
// There still might be active connections in flight but they will
// be closed as they are released.
func (ts *ThriftSet) IsClosed() bool {
	select {
	case <-ts.done:
		return true
	default:
	}

	return false
}

// GetConn will create a connection or return one from the idle list.
// It will use a host from the Watcher with the least ammount of active connections.
func (ts *ThriftSet) GetConn() (*Conn, error) {

	if ts.IsClosed() {
		return nil, ErrGetOnClosedSet
	}

	ts.StatsD.Count(sdConnRequested)

	c, err := ts.endpoints.GetConn()
	if err != nil {
		if v, ok := errMap[err]; ok {
			return nil, v
		}

		return nil, err
	}

	return &Conn{
		thriftset: ts,
		parent:    c,
		Socket:    c.Conn.(*thrift.TSocket),
		Client:    c.Data,
	}, nil
}

// resetEndpoints closes idle connections on old endpoints.
func (ts *ThriftSet) resetEndpoints() {
	hosts := ts.watch.Endpoints()
	ts.endpoints.SetEndpoints(hosts)
}

// Release puts the connection back in the pool and allows others to use it.
func (c *Conn) Release() error {
	c.thriftset.StatsD.Count(sdConnReleased)

	// TODO: this will return an error if the connection
	// failed to be closed. Is that what we want?

	c.parent.Data = c.Client
	return c.parent.Release()
}

// Close does not put this connection back into the pool.
// This should be called if there is some sort of problem with the connection.
func (c *Conn) Close() error {
	c.thriftset.StatsD.Count(sdConnClosed, 1.0)
	return c.parent.Close()
}

// triggerEvent will queue up something in the Event channel if there
// isn't already something there.
func (ts *ThriftSet) triggerEvent() {
	ts.EventCount++
	ts.LastEvent = time.Now()

	select {
	case ts.event <- struct{}{}:
	default:
	}
}
