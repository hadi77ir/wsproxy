package wsconn

import (
	"context"
	"net"

	"github.com/gorilla/websocket"
)

func DialWS(addr string) (net.Conn, error) {
	return DialCustomWS(addr, websocket.DefaultDialer)
}

func DialCustomWS(addr string, dialer *websocket.Dialer) (net.Conn, error) {
	ws, _, err := dialer.DialContext(context.Background(), addr, nil)
	if err != nil {
		return nil, err
	}
	// wrap
	return WrapConn(ws), nil
}

func WSClient(addr string, conn net.Conn, readBufSize, writeBufSize int) (net.Conn, error) {
	dialer := websocket.Dialer{
		NetDialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return conn, nil
		},
		NetDialTLSContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return conn, nil
		},
		ReadBufferSize:  readBufSize,
		WriteBufferSize: writeBufSize,
	}

	ws, _, err := dialer.Dial(addr, nil)
	if err != nil {
		return nil, err
	}

	// wrap
	return WrapConn(ws), nil
}
