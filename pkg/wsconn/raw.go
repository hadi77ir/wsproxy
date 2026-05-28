package wsconn

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/hadi77ir/wsproxy/pkg/utils"
)

const (
	wsFrameContinuation = 0
	wsFrameText         = 1
	wsFrameBinary       = 2
	wsFrameClose        = 8
	wsFramePing         = 9
	wsFramePong         = 10
)

type RawConn struct {
	base     net.Conn
	isClient bool
	reader   io.Reader
	closed   chan struct{}
	writeMu  chan struct{}
}

func WrapRawConn(conn net.Conn, isClient bool) *RawConn {
	c := &RawConn{
		base:     conn,
		isClient: isClient,
		closed:   make(chan struct{}, 1),
		writeMu:  make(chan struct{}, 1),
	}
	c.writeMu <- struct{}{}
	c.reader = utils.NewMultiReader(c.isOpen, c.nextReader)
	return c
}

func (c *RawConn) Read(b []byte) (int, error) {
	return c.reader.Read(b)
}

func (c *RawConn) Write(b []byte) (int, error) {
	<-c.writeMu
	defer func() {
		c.writeMu <- struct{}{}
	}()
	if err := c.writeFrame(wsFrameBinary, b); err != nil {
		_ = c.Close()
		return 0, err
	}
	return len(b), nil
}

func (c *RawConn) Close() error {
	c.tryClose()
	return c.base.Close()
}

func (c *RawConn) LocalAddr() net.Addr {
	return c.base.LocalAddr()
}

func (c *RawConn) RemoteAddr() net.Addr {
	return c.base.RemoteAddr()
}

func (c *RawConn) SetDeadline(t time.Time) error {
	return c.base.SetDeadline(t)
}

func (c *RawConn) SetReadDeadline(t time.Time) error {
	return c.base.SetReadDeadline(t)
}

func (c *RawConn) SetWriteDeadline(t time.Time) error {
	return c.base.SetWriteDeadline(t)
}

func (c *RawConn) isOpen() bool {
	select {
	case <-c.closed:
		return false
	default:
		return true
	}
}

func (c *RawConn) tryClose() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}

func (c *RawConn) nextReader() (io.Reader, error) {
	var message bytes.Buffer
	for {
		opcode, payload, err := c.readFrame()
		if err != nil {
			_ = c.Close()
			return nil, err
		}
		switch opcode {
		case wsFrameBinary, wsFrameText, wsFrameContinuation:
			message.Write(payload)
			return bytes.NewReader(message.Bytes()), nil
		case wsFrameClose:
			_ = c.Close()
			return nil, io.EOF
		case wsFramePing:
			if err := c.writeFrame(wsFramePong, payload); err != nil {
				_ = c.Close()
				return nil, err
			}
		case wsFramePong:
		}
	}
}

func (c *RawConn) readFrame() (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.base, header); err != nil {
		return 0, nil, err
	}
	opcode := header[0] & 0x0f
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7f)
	switch length {
	case 126:
		extended := make([]byte, 2)
		if _, err := io.ReadFull(c.base, extended); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := io.ReadFull(c.base, extended); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(extended)
	}

	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(c.base, mask[:]); err != nil {
			return 0, nil, err
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(c.base, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return opcode, payload, nil
}

func (c *RawConn) writeFrame(opcode byte, payload []byte) error {
	header := []byte{0x80 | opcode, 0}
	if c.isClient {
		header[1] = 0x80
	}
	length := len(payload)
	switch {
	case length < 126:
		header[1] |= byte(length)
	case length <= 0xffff:
		header[1] |= 126
		extended := make([]byte, 2)
		binary.BigEndian.PutUint16(extended, uint16(length))
		header = append(header, extended...)
	default:
		header[1] |= 127
		extended := make([]byte, 8)
		binary.BigEndian.PutUint64(extended, uint64(length))
		header = append(header, extended...)
	}

	if c.isClient {
		var mask [4]byte
		if _, err := rand.Read(mask[:]); err != nil {
			return err
		}
		header = append(header, mask[:]...)
		masked := make([]byte, len(payload))
		for i := range payload {
			masked[i] = payload[i] ^ mask[i%4]
		}
		payload = masked
	}
	if _, err := c.base.Write(header); err != nil {
		return err
	}
	_, err := c.base.Write(payload)
	return err
}

var _ net.Conn = &RawConn{}
