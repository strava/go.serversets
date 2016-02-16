package thriftset

import (
	"testing"
	"time"

	"github.com/strava/go.serversets/fixedset"
	"github.com/strava/go.serversets/internal/endpoints"

	"github.com/apache/thrift/lib/go/thrift"
)

func TestThriftSet(t *testing.T) {
	// this will not compile if ThriftSet does not
	// implement the pooler interface.
	var _ endpoints.Pooler = New(fixedset.New([]string{}))
}

func TestThriftSetClose(t *testing.T) {
	count := 0
	event := make(chan struct{}, 1)

	watcherClosed = func() {
		count++
		event <- struct{}{}
	}

	ts := New(fixedset.New([]string{"endpoint"}))
	socketBuilder = func(string, time.Duration) (*thrift.TSocket, error) {
		return &thrift.TSocket{}, nil
	}

	c, _ := ts.GetConn()
	c.Release()

	ts.Close()

	// little event channel to allow the other goroutine to run and quit as it should
	<-event
	if count != 1 {
		t.Error("should close watching goroutine on watcher channel close")
	}

	// can all closed multiple times.
	ts.Close()
	ts.Close()
	ts.Close()

	if v := ts.IsClosed(); !v {
		t.Errorf("should not be closed, because it is")
	}

	// This get request must releases the lock
	_, err := ts.GetConn()
	if err != ErrGetOnClosedSet {
		t.Errorf("wrong error, got %v", err)
	}

	_, err = ts.GetConn()
	if err != ErrGetOnClosedSet {
		t.Errorf("wrong error, got %v", err)
	}

	// reset
	watcherClosed = func() {}
}

func TestThriftSetGetConn(t *testing.T) {
	socketBuilder = func(string, time.Duration) (*thrift.TSocket, error) {
		return &thrift.TSocket{}, nil
	}

	fs := fixedset.New([]string{})
	ts := New(fs)
	defer ts.Close()

	_, err := ts.GetConn()
	if err != ErrNoEndpoints {
		t.Errorf("incorrect error, got %v", err)
	}

	fs.SetEndpoints([]string{"endpoint"})
	<-ts.Event()

	_, err = ts.GetConn()
	if err != nil {
		t.Errorf("should have server, got %v", err)
	}
}

func TestConn(t *testing.T) {
	type testStruct struct {
		Something int
	}

	fs := fixedset.New([]string{"host"})
	ts := New(fs)
	defer ts.Close()

	object := &testStruct{123}

	c, _ := ts.GetConn()
	c.Client = object
	c.Release()

	c, _ = ts.GetConn()
	if c.Client != object {
		t.Errorf("should have same object in new conn")
	}
	c.Release()
}

func TestConnRelease(t *testing.T) {
	ts := New(fixedset.New([]string{"endpoint"}))
	defer ts.Close()

	s1 := &thrift.TSocket{}
	socketBuilder = func(string, time.Duration) (*thrift.TSocket, error) {
		return s1, nil
	}

	c, _ := ts.GetConn()
	if c.Socket != s1 {
		t.Errorf("should get correct socket")
	}
	c.Release()

	c, _ = ts.GetConn()
	if c.Socket != s1 {
		t.Errorf("should still get the same socket")
	}

	s2 := &thrift.TSocket{}
	socketBuilder = func(string, time.Duration) (*thrift.TSocket, error) {
		return s2, nil
	}

	c, _ = ts.GetConn()
	if c.Socket != s2 {
		t.Errorf("should get new second socket because first not released")
	}
}

func TestConnClose(t *testing.T) {
	ts := New(fixedset.New([]string{"endpoint"}))
	defer ts.Close()

	s1 := &thrift.TSocket{}
	socketBuilder = func(string, time.Duration) (*thrift.TSocket, error) {
		return s1, nil
	}

	c, _ := ts.GetConn()
	if c.Socket != s1 {
		t.Errorf("should get correct socket")
	}
	c.Close()

	s2 := &thrift.TSocket{}
	socketBuilder = func(string, time.Duration) (*thrift.TSocket, error) {
		return s2, nil
	}

	c, _ = ts.GetConn()
	if c.Socket != s2 {
		t.Errorf("should get another socket because first closed and not returned")
	}
}
