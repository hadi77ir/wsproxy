package utils

import (
	"bytes"
	"io"
	"net"
)

type BufferedConn struct {
	net.Conn
	r      io.Reader
	buffer *bytes.Buffer
}

// Buffer bytes for next call to Read.
// Used to implement rewinding for connections.
func (c *BufferedConn) Buffer(b []byte) (n int, err error) {
	return c.buffer.Write(b)
}

func (c *BufferedConn) Read(b []byte) (n int, err error) {
	return c.r.Read(b)
}

var _ net.Conn = &BufferedConn{}

func NewBufferedConn(conn net.Conn, buf *bytes.Buffer) *BufferedConn {
	return &BufferedConn{
		buffer: buf,
		r:      io.MultiReader(buf, conn),
		Conn:   conn,
	}
}
