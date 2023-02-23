package proxy

import (
	"github.com/hadi77ir/go-logging"
	"github.com/hadi77ir/go-registry"
	N "github.com/hadi77ir/wsproxy/pkg/net"
	"net"
	"net/url"
	"sync"
)

type ConnHandlerFunc func(incoming net.Conn, logger logging.Logger, wg *sync.WaitGroup, done chan struct{})
type ConnHandlerCreatorFunc func(addr string, transportParams url.Values) (ConnHandlerFunc, error)

var HandlerCreators = &registry.Registry[ConnHandlerCreatorFunc]{}

func CreateDirectDialHandler(addr string, transportParams url.Values) (ConnHandlerFunc, error) {
	dialer, err := N.CreateDialer(addr, transportParams)
	if err != nil {
		return nil, err
	}
	return PrimedDialerToHandler(addr, dialer)
}

func PrimedDialerToHandler(addr string, dialer N.PrimedDialerFunc) (ConnHandlerFunc, error) {
	return func(incoming net.Conn, logger logging.Logger, wg *sync.WaitGroup, done chan struct{}) {
		rConn, err := dialer()
		if err != nil {
			logger.Log(logging.ErrorLevel, "Failed to dial", addr, err)
			return
		}
		defer closeConn(logger, rConn)

		// copy
		ch := make(chan struct{})
		wg.Add(1)
		go DuplexCopy(incoming, rConn, logger, wg, ch)

		select {
		case <-done:
		case <-ch:
		}
	}, nil
}

func CreateHandler(addr string, transportParams url.Values) (ConnHandlerFunc, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if handlerCreator, found := HandlerCreators.Get(u.Scheme); found {
		return handlerCreator(addr, transportParams)
	}
	return CreateDirectDialHandler(addr, transportParams)
}
