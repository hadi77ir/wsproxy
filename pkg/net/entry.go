package net

import (
	"github.com/hadi77ir/wsproxy/pkg/errors"
	"net"
	"net/url"
)

type PrimedDialerFunc func() (net.Conn, error)

func init() {
	registerDialers()
	registerListeners()
}

func ListenURL(addr string, transportParams url.Values) (net.Listener, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if listenFunc, found := Listeners.Get(u.Scheme); found {
		return listenFunc(addr, transportParams)
	}
	return nil, errors.ErrUnsupportedScheme
}

func DialURL(addr string, transportParams url.Values) (net.Conn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if dialFunc, found := Dialers.Get(u.Scheme); found {
		return dialFunc(addr, transportParams)
	}
	return nil, errors.ErrUnsupportedScheme
}

func CreateDialer(addr string, transportParams url.Values) (PrimedDialerFunc, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if dialFunc, found := Dialers.Get(u.Scheme); found {
		return func() (net.Conn, error) {
			return dialFunc(addr, transportParams)
		}, nil
	}
	return nil, errors.ErrUnsupportedScheme
}
