package net

import (
	"github.com/hadi77ir/go-registry"
	"github.com/hadi77ir/wsproxy/pkg/crypt"
	"github.com/hadi77ir/wsproxy/pkg/errors"
	"github.com/hadi77ir/wsproxy/pkg/utils"
	"github.com/hadi77ir/wsproxy/pkg/wsconn"
	utls "github.com/refraction-networking/utls"
	"net"
	"net/url"
	"strings"
)

type ListenFunc func(addr string, transportParams url.Values) (net.Listener, error)
type TransportListenFunc func(host string, transportParams url.Values) (net.Listener, error)

var Listeners = &registry.Registry[ListenFunc]{}

var listenerBacklog = 1024

func SetBacklogSize(backlog int) error {
	if backlog < 0 || backlog > 32767 {
		return errors.ErrBacklogOutOfRange
	}
	listenerBacklog = backlog
	return nil
}

func registerListeners() {
	Listeners.Register("tcp", listenTCP)
	Listeners.Register("tls", listenTLS)
	Listeners.Register("ws", newWSListener(listenTCP2))
	Listeners.Register("wss", newWSListener(listenTLS2))
}

func listenTCP(addr string, transportParams url.Values) (net.Listener, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(u.Scheme, "tcp") {
		return nil, errors.ErrUnsupportedScheme
	}
	if !strings.ContainsAny(u.Host, ":") {
		return nil, errors.ErrNoPortDefined
	}
	return listenTCP2(u.Host, transportParams)
}

func listenTCP2(host string, params url.Values) (net.Listener, error) {
	return net.Listen("tcp", host)
}

func listenTLS(addr string, transportParams url.Values) (net.Listener, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	// scheme check
	if !strings.EqualFold(u.Scheme, "tls") {
		return nil, errors.ErrUnsupportedScheme
	}

	return listenTLS2(u.Host, transportParams)
}

func listenTLS2(host string, transportParams url.Values) (net.Listener, error) {
	config, _, err := crypt.ParseUTLS(transportParams, false)
	if err != nil {
		return nil, err
	}

	return utls.Listen("tcp", host, config)
}

func newWSListener(transportListen TransportListenFunc) ListenFunc {
	return func(addr string, transportParams url.Values) (net.Listener, error) {
		u, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}

		listener, err := transportListen(u.Host, transportParams)
		if err != nil {
			return nil, err
		}

		wsListener, err := wsconn.WSServe(u.String(),
			listenerBacklog,
			listener,
			utils.IntegerFromParameters(transportParams, "ws.read_buffer", 0),
			utils.IntegerFromParameters(transportParams, "ws.write_buffer", 0))

		if err != nil {
			_ = listener.Close()
			return nil, err
		}
		return wsListener, nil
	}
}
