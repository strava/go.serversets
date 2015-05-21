package mcset

import (
	"net"

	"github.com/strava/go.serversets/fixedset"

	"testing"
)

func TestNew(t *testing.T) {
	fs := fixedset.New([]string{"localhost:2181"})

	mcset := New(fs)
	if l := len(mcset.Endpoints()); l != 1 {
		t.Errorf("should have one endpoint but got %v", l)
	}

	fs.SetEndpoints([]string{"localhost:2181", "localhost:2182"})

	<-mcset.Event()
	if len(mcset.Endpoints()) != 2 {
		t.Errorf("should have two endpoint but got %v", mcset.Endpoints())
	}
}

func TestCloseWatch(t *testing.T) {
	count := 0
	event := make(chan struct{}, 1)
	watcherClosed = func() {
		count++
		event <- struct{}{}
	}

	fs := fixedset.New(nil)
	New(fs)

	fs.Close()

	// little event channel to allow the other goroutine to run and quit as it should
	<-event
	if count != 1 {
		t.Error("should close watching goroutine on watcher channel close")
	}
}

func TestNewWithoutWatcher(t *testing.T) {
	New(nil)
	// should not panic or anything
}

func TestMCSetSetEndpoints(t *testing.T) {
	mcset := New(nil)
	mcset.SetEndpoints([]string{"localhost:2181"})

	if l := len(mcset.consistent.Members()); l != 1 {
		t.Errorf("should set members of consistent hash")
	}
}

func TestMCSetPickServer(t *testing.T) {
	mcset := New(nil)

	mcset.SetEndpoints([]string{"localhost:2181", "localhost:2182", "localhost:2183", "localhost:2184"})
	server1, _ := mcset.PickServer("fooo")

	mcset.SetEndpoints([]string{"localhost:2181", "localhost:2182", "localhost:2183"})
	server2, _ := mcset.PickServer("fooo")

	if server1.String() != server2.String() {
		t.Errorf("should be consistent %v != %v", server1, server2)
	}
}

func TestMCSetEach(t *testing.T) {
	count := 0
	f := func(net.Addr) error {
		count++
		return nil
	}

	mcset := New(nil)
	mcset.SetEndpoints([]string{"localhost:2181", "localhost:2182", "localhost:2183", "localhost:2184"})

	mcset.Each(f)
	if count != len(mcset.Endpoints()) {
		t.Errorf("should run for all endpoints, but run %d time(s)", count)
	}

	count = 0
	f = func(net.Addr) error {
		if count == 2 {
			return ErrNoServers
		}
		count++
		return nil
	}

	mcset.Each(f)
	if count != 2 {
		t.Errorf("should quite once there is an error, but run %d time(s)", count)
	}
}

func TestMCSetTriggerEvent(t *testing.T) {
	mcset := New(nil)

	mcset.triggerEvent()
	mcset.triggerEvent()
	mcset.triggerEvent()

	if mcset.EventCount != 3 {
		t.Errorf("event count not right, got %v", mcset.EventCount)
	}
}
