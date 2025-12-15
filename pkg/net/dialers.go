package net

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/hadi77ir/go-registry"
	"github.com/hadi77ir/wsproxy/pkg/crypt"
	"github.com/hadi77ir/wsproxy/pkg/errors"
	"github.com/hadi77ir/wsproxy/pkg/utils"
	"github.com/hadi77ir/wsproxy/pkg/wsconn"
	utls "github.com/refraction-networking/utls"
)

const defaultDialTimeout = time.Duration(5) * time.Second

type DialFunc func(addr string, transportParams url.Values) (net.Conn, error)
type TransportDialFunc func(host string, transportParams url.Values) (net.Conn, error)

var Dialers = &registry.Registry[DialFunc]{}

func registerDialers() {
	Dialers.Register("tcp", dialTCP)
	Dialers.Register("tls", dialTLS)
	Dialers.Register("ws", newWSDialer(dialTCPTransport, "ws"))
	Dialers.Register("wss", newWSDialer(dialTLSTransport, "wss"))
}

func dialTCP(addr string, transportParams url.Values) (net.Conn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	// scheme check
	if !strings.EqualFold(u.Scheme, "tcp") {
		return nil, errors.ErrUnsupportedScheme
	}
	return dialTCPTransport(u.Host, transportParams)
}

func dialTCPTransport(host string, transportParams url.Values) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", host, utils.DurationFromParameters(transportParams, "tcp.dial_timeout", defaultDialTimeout))
	if err != nil {
		return nil, err
	}
	// apply keep alive
	keepalive := utils.DurationFromParameters(transportParams, "tcp.keepalive", 0)
	if keepalive > 0 {
		err = conn.(*net.TCPConn).SetKeepAlive(true)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		err = conn.(*net.TCPConn).SetKeepAlivePeriod(keepalive)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return conn, nil
}
func dialTLS(addr string, transportParams url.Values) (net.Conn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	// scheme check
	if !strings.EqualFold(u.Scheme, "tls") {
		return nil, errors.ErrUnsupportedScheme
	}
	return dialTLSTransport(u.Host, transportParams)
}
func dialTLSTransport(host string, transportParams url.Values) (net.Conn, error) {
	config, helloId, err := crypt.ParseUTLS(transportParams, true)
	if err != nil {
		return nil, err
	}

	conn, err := dialTCPTransport(host, transportParams)
	if err != nil {
		return nil, err
	}

	tlsConn := utls.UClient(conn, config, helloId)
	err = tlsConn.Handshake()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return tlsConn, nil
}

func newWSDialer(transportDialer TransportDialFunc, scheme string) DialFunc {
	return func(addr string, transportParams url.Values) (net.Conn, error) {
		u, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}

		// scheme check
		if !strings.EqualFold(u.Scheme, scheme) {
			return nil, errors.ErrUnsupportedScheme
		}

		baseConn, err := transportDialer(addDefaultPort(u.Host, scheme), transportParams)
		if err != nil {
			return nil, err
		}

		conn, err := wsconn.WSClient(addr,
			baseConn,
			utils.IntegerFromParameters(transportParams, "ws.read_buffer", 0),
			utils.IntegerFromParameters(transportParams, "ws.write_buffer", 0))

		if err != nil {
			_ = baseConn.Close()
			return nil, err
		}
		return conn, nil
	}
}

func addDefaultPort(host string, scheme string) string {
	if !strings.ContainsAny(host, ":") {
		switch scheme {
		case "ws":
			return host + ":80"
		case "wss":
			return host + ":443"
		}
	}
	return host
}
