package wsconn

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const (
	h2SettingEnableConnectProtocol http2.SettingID = 0x8
	h2DefaultMaxFrameSize                          = 16 * 1024
	h2InitialWindowSize                            = 65535
)

var (
	errH2ConnectNotSupported = errors.New("http2 extended connect not supported by peer")
	errH2StreamClosed        = errors.New("http2 stream closed")
)

type h2StreamConn struct {
	base     net.Conn
	framer   *http2.Framer
	streamID uint32
	readBuf  bytes.Buffer
	readMu   sync.Mutex
	writeMu  sync.Mutex
	closed   chan struct{}
	once     sync.Once
}

func H2Client(addr string, conn net.Conn, host string) (net.Conn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if u.Path == "" {
		u.Path = "/"
	}
	if host == "" {
		host = u.Host
	}

	h2conn := &h2StreamConn{
		base:     conn,
		framer:   http2.NewFramer(conn, conn),
		streamID: 1,
		closed:   make(chan struct{}),
	}
	if err := h2conn.clientPreface(); err != nil {
		return nil, err
	}
	if err := h2conn.writeConnectHeaders(u, host); err != nil {
		return nil, err
	}
	if err := h2conn.readConnectResponse(); err != nil {
		return nil, err
	}
	return WrapRawConn(h2conn, true), nil
}

func (c *h2StreamConn) clientPreface() error {
	c.writeMu.Lock()
	if _, err := c.base.Write([]byte(http2.ClientPreface)); err != nil {
		c.writeMu.Unlock()
		return err
	}
	if err := c.framer.WriteSettings(
		http2.Setting{ID: h2SettingEnableConnectProtocol, Val: 1},
		http2.Setting{ID: http2.SettingInitialWindowSize, Val: h2InitialWindowSize},
	); err != nil {
		c.writeMu.Unlock()
		return err
	}
	c.writeMu.Unlock()

	for {
		frame, err := c.framer.ReadFrame()
		if err != nil {
			return err
		}
		switch f := frame.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				enableConnect := false
				f.ForeachSetting(func(setting http2.Setting) error {
					if setting.ID == h2SettingEnableConnectProtocol && setting.Val == 1 {
						enableConnect = true
					}
					return nil
				})
				c.writeMu.Lock()
				err = c.framer.WriteSettingsAck()
				c.writeMu.Unlock()
				if err != nil {
					return err
				}
				if !enableConnect {
					return errH2ConnectNotSupported
				}
				return nil
			}
		case *http2.WindowUpdateFrame:
		case *http2.PingFrame:
			if !f.IsAck() {
				c.writeMu.Lock()
				err = c.framer.WritePing(true, f.Data)
				c.writeMu.Unlock()
				if err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("http2: unexpected frame before settings: %T", frame)
		}
	}
}

func (c *h2StreamConn) writeConnectHeaders(u *url.URL, host string) error {
	var buf bytes.Buffer
	encoder := hpack.NewEncoder(&buf)
	scheme := "https"
	if strings.EqualFold(u.Scheme, "ws") {
		scheme = "http"
	}
	headers := []hpack.HeaderField{
		{Name: ":method", Value: "CONNECT"},
		{Name: ":scheme", Value: scheme},
		{Name: ":authority", Value: host},
		{Name: ":path", Value: u.RequestURI()},
		{Name: ":protocol", Value: "websocket"},
		{Name: "user-agent", Value: "wsproxy"},
	}
	for _, header := range headers {
		if err := encoder.WriteField(header); err != nil {
			return err
		}
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      c.streamID,
		BlockFragment: buf.Bytes(),
		EndHeaders:    true,
		EndStream:     false,
	})
}

func (c *h2StreamConn) readConnectResponse() error {
	decoder := hpack.NewDecoder(4096, nil)
	for {
		frame, err := c.framer.ReadFrame()
		if err != nil {
			return err
		}
		switch f := frame.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				c.writeMu.Lock()
				err = c.framer.WriteSettingsAck()
				c.writeMu.Unlock()
				if err != nil {
					return err
				}
			}
		case *http2.HeadersFrame:
			if f.StreamID != c.streamID {
				continue
			}
			fields, err := decoder.DecodeFull(f.HeaderBlockFragment())
			if err != nil {
				return err
			}
			status := ""
			for _, field := range fields {
				if field.Name == ":status" {
					status = field.Value
					break
				}
			}
			if status != "200" {
				return fmt.Errorf("http2 websocket connect failed with status %s", status)
			}
			return nil
		case *http2.RSTStreamFrame:
			if f.StreamID == c.streamID {
				return fmt.Errorf("http2 websocket stream reset: %s", f.ErrCode)
			}
		case *http2.GoAwayFrame:
			return fmt.Errorf("http2 connection closed: %s", f.ErrCode)
		case *http2.PingFrame:
			if !f.IsAck() {
				c.writeMu.Lock()
				err = c.framer.WritePing(true, f.Data)
				c.writeMu.Unlock()
				if err != nil {
					return err
				}
			}
		}
	}
}

func (c *h2StreamConn) Read(b []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	for c.readBuf.Len() == 0 {
		frame, err := c.framer.ReadFrame()
		if err != nil {
			return 0, err
		}
		switch f := frame.(type) {
		case *http2.DataFrame:
			if f.StreamID != c.streamID {
				continue
			}
			data := f.Data()
			if len(data) > 0 {
				c.readBuf.Write(data)
				c.writeMu.Lock()
				err = c.framer.WriteWindowUpdate(0, uint32(len(data)))
				if err == nil {
					err = c.framer.WriteWindowUpdate(c.streamID, uint32(len(data)))
				}
				c.writeMu.Unlock()
				if err != nil {
					return 0, err
				}
			}
			if f.StreamEnded() && c.readBuf.Len() == 0 {
				return 0, io.EOF
			}
		case *http2.SettingsFrame:
			if !f.IsAck() {
				c.writeMu.Lock()
				err = c.framer.WriteSettingsAck()
				c.writeMu.Unlock()
				if err != nil {
					return 0, err
				}
			}
		case *http2.PingFrame:
			if !f.IsAck() {
				c.writeMu.Lock()
				err = c.framer.WritePing(true, f.Data)
				c.writeMu.Unlock()
				if err != nil {
					return 0, err
				}
			}
		case *http2.RSTStreamFrame:
			if f.StreamID == c.streamID {
				return 0, errH2StreamClosed
			}
		case *http2.GoAwayFrame:
			return 0, io.EOF
		}
	}
	return c.readBuf.Read(b)
}

func (c *h2StreamConn) Write(b []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	for written := 0; written < len(b); {
		end := written + h2DefaultMaxFrameSize
		if end > len(b) {
			end = len(b)
		}
		if err := c.framer.WriteData(c.streamID, false, b[written:end]); err != nil {
			return written, err
		}
		written = end
	}
	return len(b), nil
}

func (c *h2StreamConn) Close() error {
	c.once.Do(func() {
		close(c.closed)
		c.writeMu.Lock()
		_ = c.framer.WriteData(c.streamID, true, nil)
		c.writeMu.Unlock()
	})
	return c.base.Close()
}

func (c *h2StreamConn) LocalAddr() net.Addr {
	return c.base.LocalAddr()
}

func (c *h2StreamConn) RemoteAddr() net.Addr {
	return c.base.RemoteAddr()
}

func (c *h2StreamConn) SetDeadline(t time.Time) error {
	return c.base.SetDeadline(t)
}

func (c *h2StreamConn) SetReadDeadline(t time.Time) error {
	return c.base.SetReadDeadline(t)
}

func (c *h2StreamConn) SetWriteDeadline(t time.Time) error {
	return c.base.SetWriteDeadline(t)
}

var _ net.Conn = &h2StreamConn{}
