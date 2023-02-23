package errors

import "errors"

var (
	ErrNoPortDefined     = errors.New("port not defined")
	ErrBacklogOutOfRange = errors.New("backlog out of range")
	ErrUnsupportedScheme = errors.New("unsupported scheme")
	ErrOpNotSupported    = errors.New("unsupported operation")
	ErrInvalidSyntax     = errors.New("invalid syntax")
)

type ErrMissingPart string

func (e ErrMissingPart) Error() string {
	return "missing part: " + string(e)
}

var _ error = ErrMissingPart("")
