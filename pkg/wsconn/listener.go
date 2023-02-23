package wsconn

import (
	"github.com/gorilla/websocket"
	E "github.com/hadi77ir/wsproxy/pkg/errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type Listener struct {
	backlog  chan *Conn
	addr     net.Addr
	upgrader *websocket.Upgrader
	server   *http.Server
	path     string
	logger   log.Logger
	err      error
}

func (l *Listener) Close() error {
	l.server.SetKeepAlivesEnabled(false)
	return l.server.Close()
}

func (l *Listener) serve(listener net.Listener) {
	l.server.Handler = HttpHandler(l.handle)
	var err error
	if listener == nil {
		err = l.server.ListenAndServe()
	} else {
		err = l.server.Serve(listener)
		_ = listener.Close()
	}
	if err != nil {
		l.err = err
	}
	_ = l.Close()
	close(l.backlog)
}
func (l *Listener) handle(response http.ResponseWriter, request *http.Request) {
	if l.path != request.URL.Path {
		http.NotFound(response, request)
		return
	}
	conn, err := l.upgrader.Upgrade(response, request, nil)
	if err != nil {
		return
	}
	wrapped := WrapConn(conn)
	l.backlog <- wrapped
	<-wrapped.CloseChan()
}

func (l *Listener) Accept() (net.Conn, error) {
	conn := <-l.backlog
	if conn != nil {
		return conn, nil
	}
	if l.err == nil {
		l.err = net.ErrClosed
	}
	return nil, &net.OpError{Op: "accept", Net: l.addr.Network(), Source: nil, Addr: l.addr, Err: l.err}
}

func (l *Listener) Addr() net.Addr {
	return l.addr
}

var _ net.Listener = &Listener{}

type HttpHandler func(response http.ResponseWriter, request *http.Request)

func (h HttpHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	h(response, request)
}

// when "innerListener" is set to null, will start listening on address defined in "addr"
func WSServe(addr string, backlog int, innerListener net.Listener, readBufSize, writeBufSize int) (net.Listener, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(u.Scheme, "ws") && !strings.EqualFold(u.Scheme, "wss") {
		return nil, E.ErrUnsupportedScheme
	}

	if u.Path == "" {
		u.Path = "/"
	}

	listener := &Listener{
		server:  &http.Server{Addr: addr},
		addr:    wsAddr(addr),
		path:    u.Path,
		backlog: make(chan *Conn, backlog),
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// todo: check origin
				return true
			},
			ReadBufferSize:  readBufSize,
			WriteBufferSize: writeBufSize,
		},
	}

	// start accepting and putting websockets into backlog
	go listener.serve(innerListener)

	return listener, nil
}

type wsAddr string

func (w wsAddr) Network() string {
	u := (string)(w)
	if strings.HasPrefix(u, "ws") {
		return "ws"
	} else if strings.HasPrefix(u, "wss") {
		return "wss"
	}
	return "unknown"
}

func (w wsAddr) String() string {
	return (string)(w)
}

var _ net.Addr = wsAddr("ws://")
