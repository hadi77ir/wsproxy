package net_test

import (
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	N "github.com/hadi77ir/wsproxy/pkg/net"
)

const (
	envWSSURL  = "WSPROXY_TEST_WSS_URL"
	envWSSHost = "WSPROXY_TEST_WSS_HOST"
	envWSSSNI  = "WSPROXY_TEST_WSS_SNI"
)

func TestWSSHTTP2SSHBanner(t *testing.T) {
	addr := os.Getenv(envWSSURL)
	if addr == "" {
		t.Skipf("%s is not set", envWSSURL)
	}
	host := os.Getenv(envWSSHost)
	sni := os.Getenv(envWSSSNI)
	if sni == "" {
		sni = host
	}

	params := url.Values{}
	params.Set("tls.alpn", "h2")
	if host != "" {
		params.Set("ws.host", host)
	}
	if sni != "" {
		params.Set("tls.sni", sni)
	}

	conn, err := N.DialURL(addr, params)
	if err != nil {
		t.Fatalf("DialURL over HTTP/2 failed: %v", err)
	}
	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline failed: %v", err)
	}
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("reading SSH banner failed: %v", err)
	}
	banner := string(buf[:n])
	if !strings.HasPrefix(banner, "SSH-") {
		t.Fatalf("unexpected SSH banner prefix: %q", banner)
	}
}
