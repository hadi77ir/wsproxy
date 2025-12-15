package net

import (
	"net"
	"net/url"
	"strings"

	"github.com/hadi77ir/wsproxy/pkg/errors"
)

type PrimedDialerFunc func() (net.Conn, error)

func init() {
	registerDialers()
	registerListeners()
}

func transformParams(uQ, tP url.Values) (filteredParams url.Values, transportParams url.Values) {
	filteredParams = make(url.Values)
	transportParams = make(url.Values)
	for k, v := range tP {
		transportParams[k] = v
	}
	for k, v := range uQ {
		if strings.HasPrefix(k, "tcp.") || strings.HasPrefix(k, "tls.") || strings.HasPrefix(k, "ws.") {
			transportParams[k] = v
		} else {
			filteredParams[k] = v
		}
	}
	return
}

func ListenURL(addr string, transportParams url.Values) (net.Listener, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	filteredParams, newTransportParams := transformParams(u.Query(), transportParams)
	u.RawQuery = filteredParams.Encode()
	addr = u.String()

	if listenFunc, found := Listeners.Get(u.Scheme); found {
		return listenFunc(addr, newTransportParams)
	}
	return nil, errors.ErrUnsupportedScheme
}

func DialURL(addr string, transportParams url.Values) (net.Conn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	filteredParams, newTransportParams := transformParams(u.Query(), transportParams)
	u.RawQuery = filteredParams.Encode()
	addr = u.String()

	if dialFunc, found := Dialers.Get(u.Scheme); found {
		return dialFunc(addr, newTransportParams)
	}
	return nil, errors.ErrUnsupportedScheme
}

func CreateDialer(addr string, transportParams url.Values) (PrimedDialerFunc, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	filteredParams, newTransportParams := transformParams(u.Query(), transportParams)
	u.RawQuery = filteredParams.Encode()
	addr = u.String()

	if dialFunc, found := Dialers.Get(u.Scheme); found {
		return func() (net.Conn, error) {
			return dialFunc(addr, newTransportParams)
		}, nil
	}
	return nil, errors.ErrUnsupportedScheme
}
