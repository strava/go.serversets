package httpset

import (
	"testing"
)

func TestNew(t *testing.T) {
	tw := NewTestWatcher()
	tw.endpoints = []string{"localhost:2181"}

	httpset := New(tw)
	if l := len(httpset.Endpoints()); l != 1 {
		t.Errorf("should have one endpoint but got %v", l)
	}

	tw.endpoints = []string{"localhost:2181", "localhost:2182"}
	tw.event <- struct{}{}

	<-httpset.Event()
	if len(httpset.Endpoints()) != 2 {
		t.Errorf("should have two endpoint but got %v", httpset.Endpoints())
	}
}

func TestCloseWatch(t *testing.T) {
	count := 0
	event := make(chan struct{}, 1)
	watcherClosed = func() {
		count++
		event <- struct{}{}
	}

	tw := NewTestWatcher()
	New(tw)

	tw.Close()

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

func TestHTTPSetReplaceHost(t *testing.T) {
	httpset := New(nil)
	_, err := httpset.replaceHost("http://hostname/path/path2")
	if err != ErrNoServers {
		t.Errorf("should get error, but got %v", err)
	}

	httpset.SetEndpoints([]string{"host:123"})

	tests := [][2]string{
		[2]string{"http://hostname/path/path2", "http://host:123/path/path2"},
		[2]string{"http://hostname:321/path/path2", "http://host:123/path/path2"},
		[2]string{"https://hostname:321/path/path2", "https://host:123/path/path2"},
		[2]string{"https://hostname:321/path/path2?key=value", "https://host:123/path/path2?key=value"},
		[2]string{"/path/path2?key=value", "http://host:123/path/path2?key=value"},
	}

	for _, test := range tests {
		answer, _ := httpset.replaceHost(test[0])
		if test[1] != answer {
			t.Errorf("host not replaced, expected %s, got %s", test[0], answer)
		}
	}

	// UseHTTPS
	httpset.UseHTTPS = true
	answer, _ := httpset.replaceHost("/path/path2?key=value")
	if answer != "https://host:123/path/path2?key=value" {
		t.Errorf("host not replaced, got %s", answer)
	}
}

func TestHTTPSetRotateServer(t *testing.T) {
	httpset := New(nil)
	httpset.SetEndpoints([]string{"localhost:2181", "localhost:2182"})

	if ep, _ := httpset.RotateEndpoint(); ep != "localhost:2182" {
		t.Errorf("incorrect server, got %v", ep)
	}

	if ep, _ := httpset.RotateEndpoint(); ep != "localhost:2181" {
		t.Errorf("incorrect server, got %v", ep)
	}

	if ep, _ := httpset.RotateEndpoint(); ep != "localhost:2182" {
		t.Errorf("incorrect server, got %v", ep)
	}
}

func TestHTTPSetTriggerEvent(t *testing.T) {
	httpset := New(nil)

	httpset.triggerEvent()
	httpset.triggerEvent()
	httpset.triggerEvent()

	if httpset.EventCount != 3 {
		t.Errorf("event count not right, got %v", httpset.EventCount)
	}
}

type TestWatcher struct {
	endpoints []string
	event     chan struct{}
	closed    bool
}

func NewTestWatcher() *TestWatcher {
	return &TestWatcher{
		event: make(chan struct{}, 1),
	}
}

func (tw *TestWatcher) Endpoints() []string {
	return tw.endpoints
}

func (tw *TestWatcher) Event() <-chan struct{} {
	return tw.event
}

func (tw *TestWatcher) Close() {
	tw.closed = true
	close(tw.event)
}

func (tw *TestWatcher) IsClosed() bool {
	return tw.closed
}
