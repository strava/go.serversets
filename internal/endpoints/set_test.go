package endpoints

import (
	"reflect"
	"testing"
)

func TestSetClose(t *testing.T) {
	tp := &testPooler{}
	set := NewSet(tp)
	set.SetEndpoints([]string{"host1", "host2"})

	c1, _ := set.GetConn()
	c2, _ := set.GetConn()

	// should be able to call close multiple times.
	set.Close()
	set.Close()

	for _, e := range set.list {
		if !e.IsClosed() {
			t.Errorf("endpoints should be closed")
		}
	}

	// should be able to call close multiple times.
	c1.Close()
	c1.Close()

	c2.Close()
	c2.Close()
}

func TestSetGetConn(t *testing.T) {
	tp := &testPooler{}
	set := NewSet(tp)
	set.SetEndpoints([]string{"host1", "host2"})

	c1, _ := set.GetConn()
	c2, _ := set.GetConn()

	if v := set.list[0].ActiveConnections(); v != 1 {
		t.Errorf("none should be active, set closed, got %v", v)
	}

	if v := set.list[1].ActiveConnections(); v != 1 {
		t.Errorf("none should be active, set closed, got %v", v)
	}

	c1.Close()
	c2.Close()

	if v := set.list[0].ActiveConnections(); v != 0 {
		t.Errorf("none should be active, set closed, got %v", v)
	}

	if v := set.list[1].ActiveConnections(); v != 0 {
		t.Errorf("none should be active, set closed, got %v", v)
	}
}

func TestSetGetConnClosedEndpoint(t *testing.T) {
	tp := &testPooler{}
	set := NewSet(tp)
	set.SetEndpoints([]string{"host"})

	_, err := set.GetConn()
	if err != nil {
		t.Errorf("should have server, got %v", err)
	}

	// close the only endpoint
	set.list[0].Close()
	_, err = set.GetConn()
	if err != ErrNoEndpoints {
		t.Errorf("should return no endpoints, got %v", err)
	}

	// close the whole thriftset
	set.Close()
	_, err = set.GetConn()
	if err != ErrNoEndpoints {
		t.Errorf("should return no endpoints, got %v", err)
	}
}

func TestRelease(t *testing.T) {
	tp := &testPooler{}
	set := NewSet(tp)
	set.SetEndpoints([]string{"host1"})

	c1, _ := set.GetConn()
	c1.Release()

	c2, _ := set.GetConn()
	if c1 != c2 {
		t.Errorf("should reuse connections")
	}
}

func TestSetSetEndpoints(t *testing.T) {
	tp := &testPooler{}
	set := NewSet(tp)
	set.SetEndpoints([]string{"endpoint"})

	if l := len(set.list); l != 1 {
		t.Errorf("length of endpoints wrong, got %v", l)
	}

	// add one host
	added, removed := set.SetEndpoints([]string{"endpoint", "endpoint2"})
	if added != 1 {
		t.Errorf("wrong number added, got %v", added)
	}

	if removed != 0 {
		t.Errorf("wrong number removed, got %v", removed)
	}

	if l := len(set.list); l != 2 {
		t.Errorf("length of endpoints wrong, got %v", l)
	}

	// add another host
	added, removed = set.SetEndpoints([]string{"endpoint", "endpoint2", "endpoint3"})
	if added != 1 {
		t.Errorf("wrong number added, got %v", added)
	}

	if removed != 0 {
		t.Errorf("wrong number removed, got %v", removed)
	}

	if l := len(set.list); l != 3 {
		t.Errorf("length of endpoints wrong, got %v", l)
	}

	// remove two hosts
	added, removed = set.SetEndpoints([]string{"endpoint"})
	if added != 0 {
		t.Errorf("wrong number added, got %v", added)
	}

	if removed != 2 {
		t.Errorf("wrong number removed, got %v", removed)
	}

	if l := len(set.list); l != 1 {
		t.Errorf("length of endpoints wrong, got %v", l)
	}

	if h := set.list[0].Host(); h != "endpoint" {
		t.Errorf("incorrect host, got %v", h)
	}

	// add one host
	added, removed = set.SetEndpoints([]string{"endpoint", "endpoint2"})
	if added != 1 {
		t.Errorf("wrong number added, got %v", added)
	}

	if removed != 0 {
		t.Errorf("wrong number removed, got %v", removed)
	}

	// set as one different host
	added, removed = set.SetEndpoints([]string{"endpoint3"})
	if added != 1 {
		t.Errorf("wrong number added, got %v", added)
	}

	if removed != 2 {
		t.Errorf("wrong number removed, got %v", removed)
	}
}

func TestShuffleHosts(t *testing.T) {
	original := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	hosts := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	shuffleHosts(hosts)

	if reflect.DeepEqual(hosts, original) {
		t.Errorf("did not shuffle, got %v", hosts)
	}
}
