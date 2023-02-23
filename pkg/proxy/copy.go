package proxy

import (
	"github.com/hadi77ir/go-logging"
	"io"
	"net"
	"sync"
)

func ConnCopy(dst, src net.Conn, logger logging.Logger, wg *sync.WaitGroup, copyDone chan struct{}) {
	defer wg.Done()
	defer func() {
		select {
		case <-copyDone:
			return
		default:
			close(copyDone)
		}
	}()
	_, err := io.Copy(dst, src)
	if err != nil {
		opErr, ok := err.(*net.OpError)
		switch {
		case ok && opErr.Op == "readfrom":
			return
		case ok && opErr.Op == "read":
			return
		default:
		}
		logger.Log(logging.ErrorLevel, "Failed to copy connection from",
			src.RemoteAddr(), "to", dst.RemoteAddr(), ":", err)
	}
}

func DuplexCopy(conn, rConn net.Conn, logger logging.Logger, wg *sync.WaitGroup, ch chan struct{}) {
	defer wg.Done()

	wg.Add(2)
	go ConnCopy(rConn, conn, logger, wg, ch)
	go ConnCopy(conn, rConn, logger, wg, ch)
	// rConn and conn will be closed by defer calls in handlers and proxyConn. There is nothing to do here.
	<-ch
}
