package net

import (
	"io"
	stdnet "net"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/hadi77ir/wsproxy/pkg/errors"
)

const StdioScheme = "stdio"

func IsStdioAddr(addr string) bool {
	return addr == "-" || addr == "--" || addr == StdioScheme+"://"
}

type stdioAddr string

func (a stdioAddr) Network() string {
	return StdioScheme
}

func (a stdioAddr) String() string {
	return string(a)
}

type stdioConn struct {
	reader io.Reader
	writer io.Writer
	closed chan struct{}
	once   sync.Once
}

func NewStdioConn() stdnet.Conn {
	return &stdioConn{
		reader: os.Stdin,
		writer: os.Stdout,
		closed: make(chan struct{}),
	}
}

func (c *stdioConn) Read(b []byte) (int, error) {
	select {
	case <-c.closed:
		return 0, stdnet.ErrClosed
	default:
	}
	return c.reader.Read(b)
}

func (c *stdioConn) Write(b []byte) (int, error) {
	select {
	case <-c.closed:
		return 0, stdnet.ErrClosed
	default:
	}
	return c.writer.Write(b)
}

func (c *stdioConn) Close() error {
	c.once.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *stdioConn) LocalAddr() stdnet.Addr {
	return stdioAddr("stdin")
}

func (c *stdioConn) RemoteAddr() stdnet.Addr {
	return stdioAddr("stdout")
}

func (c *stdioConn) SetDeadline(_ time.Time) error {
	return nil
}

func (c *stdioConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (c *stdioConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

type stdioListener struct {
	conn     stdnet.Conn
	closed   chan struct{}
	accepted bool
	mu       sync.Mutex
}

func listenStdio(addr string, _ url.Values) (stdnet.Listener, error) {
	if IsStdioAddr(addr) {
		return &stdioListener{conn: NewStdioConn(), closed: make(chan struct{})}, nil
	}
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if u.Scheme != StdioScheme {
		return nil, errors.ErrUnsupportedScheme
	}
	return &stdioListener{conn: NewStdioConn(), closed: make(chan struct{})}, nil
}

func dialStdio(addr string, _ url.Values) (stdnet.Conn, error) {
	if IsStdioAddr(addr) {
		return NewStdioConn(), nil
	}
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if u.Scheme != StdioScheme {
		return nil, errors.ErrUnsupportedScheme
	}
	return NewStdioConn(), nil
}

func (l *stdioListener) Accept() (stdnet.Conn, error) {
	l.mu.Lock()
	if !l.accepted {
		l.accepted = true
		conn := l.conn
		l.mu.Unlock()
		return conn, nil
	}
	l.mu.Unlock()

	<-l.closed
	return nil, stdnet.ErrClosed
}

func (l *stdioListener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
	return l.conn.Close()
}

func (l *stdioListener) Addr() stdnet.Addr {
	return stdioAddr("stdio")
}
