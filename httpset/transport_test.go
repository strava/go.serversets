package httpset

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/strava/go.serversets/fixedset"
)

type StubRoundTripper struct {
	PrevRequest *http.Request
}

func (rt *StubRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.PrevRequest = req
	return nil, nil
}

func TestNewTransport(t *testing.T) {
	fs := fixedset.New([]string{"localhost:2181"})
	transport := NewTransport(fs)
	if l := len(transport.Endpoints()); l != 1 {
		t.Errorf("should have one endpoint but got %v", l)
	}

	fs.SetEndpoints([]string{"localhost:2181", "localhost:2182"})

	<-transport.Event()
	if len(transport.Endpoints()) != 2 {
		t.Errorf("should have two endpoint but got %v", transport.Endpoints())
	}
}

func TestTransportRoundTrip(t *testing.T) {
	transport := NewTransport(nil)
	stub := &StubRoundTripper{}
	transport.BaseTransport = stub

	r, _ := http.NewRequest("GET", "http://localhost", nil)
	_, err := transport.RoundTrip(r)
	if err != ErrNoServers {
		t.Errorf("should get error if no servers")
	}

	transport.SetEndpoints([]string{"a:80", "b:80"})
	<-transport.Event()

	r, _ = http.NewRequest("GET", "http://localhost", nil)
	transport.RoundTrip(r)
	if v := stub.PrevRequest.URL.String(); v != "http://b:80" {
		t.Errorf("incorrect url, got %v", v)
	}

	transport.RoundTrip(r)
	if v := stub.PrevRequest.URL.String(); v != "http://a:80" {
		t.Errorf("incorrect url, got %v", v)
	}

	transport.RoundTrip(r)
	if v := stub.PrevRequest.URL.String(); v != "http://b:80" {
		t.Errorf("incorrect url, got %v", v)
	}
}

func TestTransportCloseWatch(t *testing.T) {
	count := 0
	event := make(chan struct{}, 1)
	watcherClosed = func() {
		count++
		event <- struct{}{}
	}

	fs := fixedset.New(nil)
	NewTransport(fs)

	fs.Close()

	// little event channel to allow the other goroutine to run and quit as it should
	<-event
	if count != 1 {
		t.Error("should close watching goroutine on watcher channel close")
	}
}

func TestNewTransportWithoutWatcher(t *testing.T) {
	NewTransport(nil)
	// should not panic or anything
}

func TestTransportRotateServer(t *testing.T) {
	transport := NewTransport(nil)
	transport.SetEndpoints([]string{"localhost:2181", "localhost:2182"})

	if ep, _ := transport.RotateEndpoint(); ep != "localhost:2182" {
		t.Errorf("incorrect server, got %v", ep)
	}

	if ep, _ := transport.RotateEndpoint(); ep != "localhost:2181" {
		t.Errorf("incorrect server, got %v", ep)
	}

	if ep, _ := transport.RotateEndpoint(); ep != "localhost:2182" {
		t.Errorf("incorrect server, got %v", ep)
	}
}

func TestTransportTriggerEvent(t *testing.T) {
	transport := NewTransport(nil)

	transport.triggerEvent()
	transport.triggerEvent()
	transport.triggerEvent()

	if transport.EventCount != 3 {
		t.Errorf("event count not right, got %v", transport.EventCount)
	}
}

func TestTransportReplaceHost(t *testing.T) {
	transport := NewTransport(nil)

	r, _ := http.NewRequest("GET", "http://hostname/path/path2", nil)
	err := transport.replaceHost(r)
	if err != ErrNoServers {
		t.Errorf("should get error, but got %v", err)
	}

	transport.SetEndpoints([]string{"host:123"})
	tests := [][2]string{
		[2]string{"http://hostname/path/path2", "http://host:123/path/path2"},
		[2]string{"http://hostname:321/path/path2", "http://host:123/path/path2"},
		[2]string{"https://hostname:321/path/path2", "https://host:123/path/path2"},
		[2]string{"https://hostname:321/path/path2?key=value", "https://host:123/path/path2?key=value"},
		[2]string{"/path/path2?key=value", "http://host:123/path/path2?key=value"},
	}

	for _, test := range tests {
		r, _ := http.NewRequest("GET", test[0], nil)
		transport.replaceHost(r)
		if v := r.URL.String(); v != test[1] {
			t.Errorf("host not replaced, expected %s, got %s", test[1], v)
		}
	}

	// UseHTTPS
	transport.UseHTTPS = true
	r, _ = http.NewRequest("GET", "/path/path2?key=value", nil)
	transport.replaceHost(r)

	if v := r.URL.String(); v != "https://host:123/path/path2?key=value" {
		t.Errorf("host not replaced, got %s", v)
	}
}

func TestTransport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	count1 := 0
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count1++
	}))
	defer server1.Close()

	count2 := 0
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count2++
	}))
	defer server2.Close()

	transport := NewTransport(nil)

	u1, _ := url.Parse(server1.URL)
	u2, _ := url.Parse(server2.URL)

	transport.SetEndpoints([]string{u1.Host, u2.Host})
	<-transport.Event()

	r, _ := http.NewRequest("GET", "http://somehost/", nil)
	_, err := transport.RoundTrip(r)
	if err != nil {
		t.Errorf("request error: %v", err)
	}

	if count2 != 1 {
		t.Errorf("should hit the second server, got %v", count2)
	}

	_, err = transport.RoundTrip(r)
	if err != nil {
		t.Errorf("request error: %v", err)
	}

	if count1 != 1 {
		t.Errorf("should hit the first server, got %v", count1)
	}
}
