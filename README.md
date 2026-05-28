wsproxy
=======

[![Go Report Card](https://goreportcard.com/badge/github.com/hadi77ir/wsproxy)](https://goreportcard.com/report/github.com/hadi77ir/wsproxy)

Hassle-free and secure Websockify implementation in the [Go](https://golang.org) programming language.

This program uses uTLS library and gorilla's WebSocket implementation.

Installation
------------
With a correctly configured Go toolchain:

```sh
go get -u github.com/hadi77ir/wsproxy
```

Example
------------
To expose MySQL port over WebSockets, run the following on your server:

```sh
wsproxy "ws://127.0.0.1:8090/mysql-ws" "tcp://127.0.0.1:3306"
```

Then point your `nginx` installation to reverse proxy requests coming on `/mysql-ws` to `wsproxy` running on port 90.

```nginx
location /mysql-ws {
    proxy_pass http://127.0.0.1:8090/;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "Upgrade";
    proxy_set_header Host $host;
}
```

On your client:

```sh
wsproxy "tcp://127.0.0.1:3306" "wss://mywebsite.com/mysql-ws"
```

Configuration
-------------
It takes two positional arguments: `LOCAL` and `REMOTE`.

```sh
wsproxy [flags] LOCAL REMOTE
```

Each endpoint is normally a URL. Supported endpoint schemes are:

- `tcp://127.0.0.1:9050`
- `tls://localhost:443`
- `ws://mysite.com/wspoint`
- `wss://mysite.com/wspoint`
- `socks5://`
- `stdio://`, `-`, or `--` for stdin/stdout tunneling

`LOCAL` is listened on or accepted from. `REMOTE` is dialed or handled for each incoming connection. `socks5://` is a
remote handler endpoint that runs the built-in SOCKS5 server on the accepted connection.

CLI flags:

- `--lo`, `-l`: transport parameters for the local endpoint, one `key=value` per flag.
- `--ro`, `-r`: transport parameters for the remote endpoint, one `key=value` per flag.
- `--help`, `-h`: print help to stderr.
- `--version`, `-v`: print version information to stderr.

Transport parameters can also be placed in endpoint query strings. Query parameters whose names start with `tcp.`, `tls.`,
or `ws.` are treated as transport parameters. Other query parameters remain part of the endpoint URL and are passed to the
endpoint handler, such as `socks5://?socks5.action=deny`.

Environment variables:

- `LOG_LEVEL`: log level passed to the logging backend, such as `debug`, `info`, `warn`, or `error`.

Transport Parameters
-----------------------
Multiple declarations of the same option are not supported. Some options accept comma-separated values or colon-separated
paths as noted below.

### TCP client

- `tcp.dial_timeout`: TCP dial timeout. Go duration string. Default: `5s`.
- `tcp.keepalive`: TCP keepalive period. Go duration string. Default: disabled. Example: `30s`.

### WebSocket client and server

- `ws.host`: Override the HTTP `Host` header for HTTP/1.1 WebSockets and the `:authority` pseudo-header for HTTP/2
  WebSockets. Useful when dialing an IP address while the virtual host is a domain.
- `ws.read_buffer`: WebSocket read buffer size in bytes. Default: `0`, which lets the HTTP/WebSocket stack choose.
- `ws.write_buffer`: WebSocket write buffer size in bytes. Default: `0`, which lets the HTTP/WebSocket stack choose.

`wss://` clients support WebSockets over HTTP/2 using RFC 8441 extended CONNECT when TLS negotiates `h2` and the peer
advertises `SETTINGS_ENABLE_CONNECT_PROTOCOL`. If the peer negotiates `h2` without RFC 8441 support, clients
automatically retry with HTTP/1.1 unless ALPN was explicitly configured.

### TLS client and server

- `tls.sni`: Server Name Indication value. Used by TLS clients. Also available to server config as the configured server
  name.
- `tls.alpn`: ALPN protocol identifiers separated by comma. Examples: `h2,http/1.1`, `http/1.1`. Set to `none`,
  `off`, `disable`, `disabled`, `noalpn`, or `no-alpn` to disable ALPN.
- `tls.alpn.force_http11`: Boolean. Force ALPN to only `http/1.1`.
- `tls.alpn.disable`: Boolean. Disable ALPN. For uTLS browser profiles this removes the profile's ALPN and related
  application settings extensions.
- `tls.cert`: Certificate path. Required for TLS servers. Optional for clients when client certificate authentication is
  required by the server. Multiple certificate paths are separated with `:`.
- `tls.key`: Private key path matching `tls.cert`. Required whenever `tls.cert` is set. Multiple private key paths are
  separated with `:`.

Certificate and key path values may also be inline data using `base64:` or `base32:` prefixes.

### TLS client only

- `tls.profile`: uTLS ClientHello profile. Default: Go's standard TLS profile.
- `tls.fragment`: Fragment outbound TLS writes before the handshake. Set to `true` for default `0,1,10,20,0,0`, or
  provide `packetsFrom,packetsTo,lengthMin,lengthMax,delayMin,delayMax`. Example:
  `0,1,10,20,10ms,15ms`.
- `tls.pin`: Certificate public key pinning. Format: `digest:hex`, with multiple pins separated by comma. Supported
  digest names are `sha1`, `sha224`, `sha256`, `sha384`, `sha512`, and `sha3`.
- `tls.insecure`: Boolean. Disable certificate verification.
- `tls.ca`: CA certificate path for verifying the server. Multiple CA paths are separated with `:`.

Supported `tls.profile` aliases include:

- `go`, `golang`
- `custom`
- `random`, `randomized`, `randomized-alpn`, `randomized-no-alpn`, `random-no-alpn`, `randomized-without-alpn`
- `chrome`, `chrome-auto`, `chrome-58`, `chrome-62`, `chrome-70`, `chrome-72`, `chrome-83`, `chrome-87`,
  `chrome-96`, `chrome-100`, `chrome-102`, `chrome-106`, `chrome-106-shuffle`
- `firefox`, `firefox-auto`, `firefox-55`, `firefox-56`, `firefox-63`, `firefox-65`, `firefox-99`,
  `firefox-102`, `firefox-105`
- `ios`, `ios-auto`, `ios-11.1`, `ios-12.1`, `ios-13`, `ios-14`
- `android`, `android-11`, `android-11-okhttp`
- `edge`, `edge-auto`, `edge-85`, `edge-106`
- `safari`, `safari-auto`, `safari-16.0`
- `360`, `360-auto`, `360-7.5`, `360-11.0`
- `qq`, `qq-auto`, `qq-11.1`

Profiles may also be written as `client,version`, `client:version`, `client/version`, `client_version`, or
`client-version`, such as `chrome,106` or `firefox-105`.

### TLS server only

- `tls.clientca`: Client CA certificate path. If set, clients must present a certificate signed by one of these CAs.
  Multiple Client CA paths are separated with `:`.

### SOCKS5 handler

These options apply to `socks5://` remote endpoints and may be supplied through `--ro` or in the `socks5://` URL query.

- `socks5.username`: Username for simple username/password authentication. Must be used with `socks5.password`.
- `socks5.password`: Password for simple username/password authentication. Must be used with `socks5.username`.
- `socks5.credentials`: Path or inline data for an additional credentials file. Each non-comment line is
  `username:password`.
- `socks5.action`: Default action when no ruleset rule matches. Values: `allow`, `accept`, `approve`, `deny`, `reject`,
  or `block`. Default: `allow`.
- `socks5.ruleset`: Path or inline data for a ruleset file. Each line is `ACTION,ADDRESS,PORT`.
- `socks5.rewrites`: Path or inline data for a rewrite file. Each line is `ADDRESS,PORT,TARGET_ADDRESS,TARGET_PORT`.

Ruleset `ACTION` values are `allow`/`accept`/`approve` or `deny`/`reject`/`block`.

Rule and rewrite addresses can be:

- `F:google.com`
- `F:www.*exam*le.co*`
- `F:*.google.com`
- `192.168.0.0/24`
- `192.168.10.0-192.168.20.40`
- `192.168.10.10`

Ports can be:

- `*`: all ports
- `90`: exactly port 90
- `90-800`: ports from 90 up to, but not including, 800
- `90 92`: port 90 or port 92
- `^443`: all ports except 443

STDIO mode
----------
Use `stdio://`, `-`, or `--` as the local endpoint to tunnel a single connection over stdin/stdout. This is intended for
tools such as SSH `ProxyCommand`, similar to using `nc`.

```sh
ssh -o 'ProxyCommand=wsproxy - wss://mywebsite.com/ssh-ws --ro tls.alpn=http/1.1' user@target
```

Bonus! SOCKS Proxy Deployment
---------------------
You may use it as `gsocks` client and server too! If you run your own simple SOCKS5 server on the server or in an even more
complicated scenario, a Tor client instance, you may use this program to TLSify it.

On your server:
```sh
wsproxy tls://0.0.0.0:8443/ tcp://127.0.0.1:9050/ --lo tls.cert=cert.pem --lo tls.key=key.pem
```

or use built-in SOCKS5 server!
```sh
wsproxy tls://0.0.0.0:8443/ socks5:// --lo tls.cert=cert.pem --lo tls.key=key.pem
```

On your client:
```sh
wsproxy tcp://127.0.0.1:1080/ tls://myserver.com:8443/
```

The built-in SOCKS5 server supports authentication, rulesets, and rewrites. See the SOCKS5 handler options in
Transport Parameters above.

Contributions
-------------
Please don't hesitate to fork the project and send a pull request or submit issues, but keep in mind that this project
with its low-quality, non-documented code is going to be soon archived after my work-in-progress project reaches its
stable state and replaces this project with better functionality.

License
-------
The Apache License, Version 2.0 - see LICENSE for more details.

Credits
---------
- Burak Sezer ([buraksezer](https://github.com/buraksezer)) - for implementation of `gsocks`
- [Refraction Networking](https://github.com/refraction-networking) for [uTLS](https://github.com/refraction-networking/utls)
- [Gorilla Toolkit](https://github.com/gorilla) for WebSocket implementation

Todo
--------
- gRPC connection (server and client), as implemented by [gun](https://github.com/Qv2ray/gun) and [v2ray-core](https://github.com/v2fly/v2ray-core/tree/e9943b5a7295ca76341c996a4937f7e03a5015f9/transport/internet/grpc)
- toml/yaml configuration
- Tests
