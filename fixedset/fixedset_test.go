package fixedset

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	endpoints := []string{"a", "b"}
	fs := New(endpoints)

	if !reflect.DeepEqual(fs.Endpoints(), endpoints) {
		t.Errorf("endpoints not set properly")
	}
}

func TestFixedSetSetEndpoints(t *testing.T) {
	fs := New([]string{})

	endpoints := []string{"b", "a"}
	fs.SetEndpoints(endpoints)
	if c := fs.EventCount; c != 1 {
		t.Errorf("should trigger event on set endpoints, got %v", c)
	}

	if !reflect.DeepEqual(fs.Endpoints(), []string{"a", "b"}) {
		t.Errorf("endpoints not set properly, or sorted, got %v", fs.Endpoints())
	}

	endpoints[0] = "c"
	if !reflect.DeepEqual(fs.Endpoints(), []string{"a", "b"}) {
		t.Errorf("endpoints should be copied completely, got %v", fs.Endpoints())
	}
}

func TestFixedSetClose(t *testing.T) {
	fs := New([]string{})

	// should multi-close
	fs.Close()
	fs.Close()
	fs.Close()
	fs.Close()
}

func TestFixedSetTriggerEvent(t *testing.T) {
	fs := New(nil)

	fs.triggerEvent()
	fs.triggerEvent()
	fs.triggerEvent()

	if c := fs.EventCount; c != 3 {
		t.Errorf("event count not right, got %v", c)
	}
}
