package utils

import (
	"encoding/base32"
	"encoding/base64"
	"io"
	"os"
	"strings"
)

type Decoder interface {
	Decode(dst, src []byte) (n int, err error)
	DecodedLen(int) int
}

// ReadFile supports reading files at given directions, along base64 and base32 values.
func ReadFile(path string) ([]byte, error) {
	var decoder Decoder
	var toDecode []byte
	switch {
	case strings.HasPrefix(path, "base64:"):
		decoder = base64.StdEncoding
		toDecode = []byte(path[len("base64:"):])
		break
	case strings.HasPrefix(path, "base32:"):
		decoder = base32.StdEncoding
		toDecode = []byte(path[len("base32:"):])
		break
	}
	if decoder != nil {
		decoded := make([]byte, decoder.DecodedLen(len(path)))
		_, err := decoder.Decode(decoded, toDecode)
		if err != nil {
			return nil, err
		}
		return decoded, nil
	}
	return os.ReadFile(path)
}

// Below code is a heavily modified version of io/multi.go
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) {
	return 0, io.EOF
}

var EOFReader io.Reader = &eofReader{}

type DynamicMultiReader struct {
	lastReader io.Reader
	nextReader func() (io.Reader, error)
	isOpen     func() bool
}

func (mr *DynamicMultiReader) Read(p []byte) (n int, err error) {
	for mr.isOpen() || mr.lastReader != EOFReader {
		n, err = mr.lastReader.Read(p)
		if err == io.EOF {
			// move to the next reader
			next, err := mr.nextReader()
			if err != nil {
				if err == io.EOF {
					// End it. There will be no more readers.
					mr.isOpen = retFalse
					mr.nextReader = nextEOF
				}
				mr.lastReader = EOFReader
				return 0, err
			}
			mr.lastReader = next
		}
		if n > 0 || err != io.EOF {
			if err == io.EOF && mr.isOpen() {
				// Don't return EOF yet. More readers remain.
				err = nil
			}
			return
		}
	}
	return 0, io.EOF
}

func nextEOF() (io.Reader, error) {
	return nil, io.EOF
}

func retFalse() bool {
	return false
}

func (mr *DynamicMultiReader) WriteTo(w io.Writer) (sum int64, err error) {
	return mr.writeToWithBuffer(w, make([]byte, 1024*32))
}

func (mr *DynamicMultiReader) writeToWithBuffer(w io.Writer, buf []byte) (sum int64, err error) {
	for mr.isOpen() || mr.lastReader != EOFReader {
		var n int64
		n, err = io.CopyBuffer(w, mr.lastReader, buf)
		sum += n
		if err != nil && err != io.EOF {
			// If there was an error but wasn't an EOF, return.
			// If it is an EOF, we can move on to the next reader.
			return sum, err
		}

		mr.lastReader, err = mr.nextReader()
		if err != nil {
			mr.lastReader = EOFReader
			if err == io.EOF {
				// End it. There will be no more readers.
				mr.isOpen = retFalse
				mr.nextReader = nextEOF
			}
			return sum, err
		}
	}
	mr.lastReader = EOFReader
	return sum, nil
}

var _ io.WriterTo = (*DynamicMultiReader)(nil)

// NewMultiReader returns a Reader that's the logical concatenation of
// the provided input readers. They're read sequentially. Once all
// inputs have returned EOF, Read will return EOF.  If any of the readers
// return a non-nil, non-EOF error, Read will return that error.
func NewMultiReader(isOpen func() bool, nextReader func() (io.Reader, error)) io.Reader {
	return &DynamicMultiReader{
		lastReader: EOFReader,
		isOpen:     isOpen,
		nextReader: nextReader,
	}
}
