package socks5

import (
	"github.com/armon/go-socks5"
	"github.com/hadi77ir/wsproxy/pkg/errors"
	"net"
	"strings"
)

func ParseAddrSpec(host string, port string) (*socks5.AddrSpec, error) {
	portParsed, err := ParsePort(port)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(host, "F:") {
		return &socks5.AddrSpec{FQDN: host[2:], Port: portParsed}, nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, errors.ErrInvalidSyntax
	}
	return &socks5.AddrSpec{IP: ip, Port: portParsed}, nil
}
