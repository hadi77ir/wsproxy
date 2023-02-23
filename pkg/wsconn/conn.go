package wsconn

import (
	"github.com/gorilla/websocket"
	"github.com/hadi77ir/wsproxy/pkg/utils"
	"io"
	"net"
	"time"
)

const maxConsecutiveEmptyReads = 100

type Conn struct {
	base   *websocket.Conn
	reader io.Reader
	closed chan struct{}
}

func (c *Conn) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}

func (c *Conn) nextReader() (io.Reader, error) {
	for i := 0; i < maxConsecutiveEmptyReads; i++ {
		msgType, reader, err := c.base.NextReader()
		if err != nil {
			_ = c.Close()
			return nil, err
		}
		if msgType == websocket.BinaryMessage {
			return reader, nil
		}
	}
	_ = c.Close()
	return nil, io.EOF
}

func (c *Conn) Write(b []byte) (n int, err error) {
	err = c.base.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		_ = c.Close()
		return
	}
	n = len(b)
	return
}

func (c *Conn) Close() error {
	c.tryClose()
	return c.base.Close()
}

func (c *Conn) LocalAddr() net.Addr {
	return c.base.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.base.RemoteAddr()
}

func (c *Conn) SetDeadline(t time.Time) error {
	if err := c.base.SetReadDeadline(t); err != nil {
		return err
	}
	if err := c.base.SetWriteDeadline(t); err != nil {
		return err
	}
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.base.SetReadDeadline(t)
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.base.SetWriteDeadline(t)
}

func (c *Conn) CloseChan() <-chan struct{} {
	return c.closed
}
func (c *Conn) isOpen() bool {
	select {
	case <-c.closed:
		return false
	default:
	}
	return true
}

func (c *Conn) tryClose() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}

var _ net.Conn = &Conn{}

func WrapConn(conn *websocket.Conn) *Conn {
	c := &Conn{base: conn, closed: make(chan struct{}, 1)}
	c.reader = utils.NewMultiReader(c.isOpen, c.nextReader)
	return c
}
