// Copyright 2017 Burak Sezer
// Copyright 2023 Mohammad Hadi Hosseinpour
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"io"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/hadi77ir/go-logging"
	N "github.com/hadi77ir/wsproxy/pkg/net"
)

type Endpoint struct {
	Addr            string
	TransportParams url.Values
}

type Proxy struct {
	localEndpoint  Endpoint
	remoteEndpoint Endpoint
	remoteDialer   N.PrimedDialerFunc
	wg             sync.WaitGroup
	errChan        chan error
	signal         chan os.Signal
	done           chan struct{}
	logger         logging.Logger
	connHandler    ConnHandlerFunc
}

func NewProxy(localEndpoint, remoteEndpoint Endpoint, logger logging.Logger, sigChan chan os.Signal) *Proxy {
	return &Proxy{
		localEndpoint:  localEndpoint,
		remoteEndpoint: remoteEndpoint,
		logger:         logger,
		errChan:        make(chan error, 1),
		signal:         sigChan,
		done:           make(chan struct{}),
	}
}

func closeConn(logger logging.Logger, closer io.Closer) {
	err := closer.Close()
	if err != nil {
		if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
			logger.Log(logging.DebugLevel, "Error while closing socket", err)
		}
	}
}

func (c *Proxy) proxyConn(conn net.Conn) {
	defer c.wg.Done()
	defer closeConn(c.logger, conn)
	c.connHandler(conn, c.logger, &c.wg, c.done)
}

func (c *Proxy) serve(l net.Listener) {
	defer c.wg.Done()
	for {
		conn, err := l.Accept()
		if err != nil {
			c.logger.Log(logging.DebugLevel, "Listener error:", err)
			// Shutdown the client immediately.
			c.Shutdown()
			if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
				c.errChan <- err
				return
			}
			c.errChan <- nil
			return
		}

		c.wg.Add(1)
		go c.proxyConn(conn)
	}
}

func (c *Proxy) Shutdown() {
	select {
	case <-c.done:
		return
	default:
	}
	close(c.done)
}

func (c *Proxy) Run() error {
	var err error
	c.connHandler, err = CreateHandler(c.remoteEndpoint.Addr, c.remoteEndpoint.TransportParams)
	if err != nil {
		return err
	}

	ln, err := N.ListenURL(c.localEndpoint.Addr, c.localEndpoint.TransportParams)
	if err != nil {
		return err
	}

	c.logger.Log(logging.InfoLevel, "Proxy listener runs on", c.localEndpoint.Addr)
	c.wg.Add(1)
	go c.serve(ln)

	select {
	// Wait for SIGINT or SIGTERM
	case <-c.signal:
	// Wait for a listener error
	case <-c.done:
	}

	// Signal all running goroutines to stop.
	c.Shutdown()

	c.logger.Log(logging.InfoLevel, "Stopping proxy listener", c.localEndpoint.Addr)
	if err = ln.Close(); err != nil {
		c.logger.Log(logging.ErrorLevel, "Failed to close listener", err)
	}

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		c.wg.Wait()
	}()

	select {
	case <-ch:
	case <-time.After(time.Duration(10) * time.Second):
		c.logger.Log(logging.WarnLevel, "Some goroutines will be stopped immediately")
	}
	return <-c.errChan
}
