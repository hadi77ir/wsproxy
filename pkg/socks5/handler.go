package socks5

import (
	"github.com/armon/go-socks5"
	"github.com/hadi77ir/go-logging"
	"github.com/hadi77ir/wsproxy/pkg/proxy"
	"github.com/hadi77ir/wsproxy/pkg/utils"
	"net"
	"net/url"
	"sync"
)

func init() {
	proxy.HandlerCreators.Register("socks5", CreateSocks5Handler)
}

func CreateSocks5Handler(addr string, transportParams url.Values) (proxy.ConnHandlerFunc, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	conf, err := ParseConfig(utils.MergeParams(u.Query(), transportParams))
	if err != nil {
		return nil, err
	}

	server, err := socks5.New(conf)
	if err != nil {
		return nil, err
	}

	return func(incoming net.Conn, logger logging.Logger, wg *sync.WaitGroup, done chan struct{}) {
		if err := server.ServeConn(incoming); err != nil {
			logger.Log(logging.ErrorLevel, "Error serving connection:", err)
		}
	}, nil
}
