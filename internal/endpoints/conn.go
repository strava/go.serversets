package endpoints

import (
	"io"
	"time"
)

// A Conn represents a connection to an endpoint.
type Conn struct {
	ep       *endpoint
	Endpoint string

	// Conn is the item created by a Set.OpenSocket(host) call.
	Conn io.Closer

	// Data can be uses to store other info on the connection.
	// Used by the thriftset to store protocol and transport factories.
	Data interface{}
}

// Close on this connection calls close on the underlying connection/closer.
func (c *Conn) Close() error {
	return c.ep.RemoveAndClose(c)
}

// Release puts the connection back in the pool and allows others to use it.
func (c *Conn) Release() error {

	// TODO: this will return an error if the connection failed to be closed. Is that what we want?
	return c.ep.ReturnConn(c)
}

type idleConn struct {
	conn      *Conn
	lastUsage time.Time
}

func (ic idleConn) Close() error {
	return ic.conn.ep.RemoveAndClose(ic.conn)
}
