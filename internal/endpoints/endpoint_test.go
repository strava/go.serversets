package endpoints

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
)

type testPooler struct {
	idleTimeout      time.Duration
	maxActivePerHost int
	maxIdlePerHost   int
}

func (tp *testPooler) OpenConn(host string) (io.Closer, error) {
	return &thrift.TSocket{}, nil
}

func (tp *testPooler) IdleTimeout() time.Duration {
	return tp.idleTimeout
}

func (tp *testPooler) MaxActivePerHost() int {
	return tp.maxActivePerHost
}

func (tp *testPooler) MaxIdlePerHost() int {
	return tp.maxIdlePerHost
}

func TestEndpointActive(t *testing.T) {
	tp := &testPooler{}
	set := NewSet(tp)

	_, err := set.GetConn()
	if err != ErrNoEndpoints {
		t.Errorf("incorrect number of endpoints, got %v", err)
	}
	set.SetEndpoints([]string{"hosts"})

	c, _ := set.GetConn()
	if c.Endpoint != "hosts" {
		t.Errorf("should set endpoint value, got %v", c.Endpoint)
	}

	if v := set.list[0].ActiveConnections(); v != 1 {
		t.Errorf("should have 1 active but got %v", v)
	}
	c.Release()

	if v := set.list[0].ActiveConnections(); v != 0 {
		t.Errorf("should have 0 active but got %v", v)
	}

	c, _ = set.GetConn()
	if c.Endpoint != "hosts" {
		t.Errorf("should set endpoint value, got %v", c.Endpoint)
	}
	c.Close()

	if v := set.list[0].ActiveConnections(); v != 0 {
		t.Errorf("should have 0 active but got %v", v)
	}
}

func TestEndpointClose(t *testing.T) {
	tp := &testPooler{
		maxActivePerHost: 1,
	}
	ep := newEndpoint(tp, "host")

	c, _ := ep.GetConn()
	c.Release()
	ep.Close()

	// can all closed multiple times.
	ep.Close()
	ep.Close()
	ep.Close()

	if v := ep.IsClosed(); !v {
		t.Errorf("should be closed, because it is")
	}

	// This get request must release the lock
	_, err := ep.GetConn()
	if err != ErrGetOnClosedEndpoint {
		t.Errorf("wrong error, got %v", err)
	}
}

func TestEndpointGetConn(t *testing.T) {
	tp := &testPooler{}
	ep := newEndpoint(tp, "host")

	c1, _ := ep.GetConn()
	if c1.Endpoint != "host" {
		t.Errorf("should set endpoint value, got %v", c1.Endpoint)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		c1.Release()
		log.Printf("released")
	}()

	// This should not deadlock and wait for the first to be released
	c2, _ := ep.GetConn()
	c2.Release()
}
