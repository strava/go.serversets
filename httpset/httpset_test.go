package httpset

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/strava/go.serversets/fixedset"
)

func TestNew(t *testing.T) {
	fs := fixedset.New([]string{"localhost:2181"})

	httpset := New(fs)
	if l := len(httpset.Endpoints()); l != 1 {
		t.Errorf("should have one endpoint but got %v", l)
	}

	fs.SetEndpoints([]string{"localhost:2181", "localhost:2182"})

	<-httpset.Event()
	if len(httpset.Endpoints()) != 2 {
		t.Errorf("should have two endpoint but got %v", httpset.Endpoints())
	}
}

func TestHTTPSet(t *testing.T) {
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

	set := New(nil)

	u1, _ := url.Parse(server1.URL)
	u2, _ := url.Parse(server2.URL)

	set.SetEndpoints([]string{u1.Host, u2.Host})
	<-set.Event()

	set.Get("http://somehost/")
	if count2 != 1 {
		t.Errorf("should hit the second server, got %v", count2)
	}

	set.Head("http://somehost/")
	if count1 != 1 {
		t.Errorf("should hit the first server, got %v", count1)
	}
}
