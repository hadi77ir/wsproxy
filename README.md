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
It takes two main arguments as positional arguments: Local Endpoint and Remote Endpoint, speaking of which, both are in URL format.

Examples for endpoints are:
- `tcp://127.0.0.1:9050`
- `tls://localhost:443`
- `ws://mysite.com/wspoint`
- `wss://mysite.com/wspoint`

Transport parameter configuration and tuning is done through `--ro` and `--lo` options.

Transport Parameters
-----------------------
- WS Server and Client
  - `ws.read_buf` and `ws.write_buf`: Read and write buffer sizes. Both are numbers, in bytes. If set to zero, buffers from HTTP stack will be used. 
- TLS Client:
  - `tls.profile`: The client fingerprint to imitate during initial handshake.
  - `tls.pin`: Certificate pinning, enables safe and secure deployments using self-signed certificates. Format: `sha256:abcdef...`<br>
    To supply multiple private keys, separate them using commas (`,`).
  - `tls.insecure`: Disables certificate verification.
- TLS Client and Server:
  - `tls.sni`: Server Name Indicator
  - `tls.alpn`: Application Level Protocol Negotiation identifiers, separated by comma (`,`) 
  - `tls.cert`: TLS Certificate, required for servers and optional for clients. Clients must provide it if the server requires
    Client Authentication. To supply multiple certificates, separate their paths using colons (`:`).
  - `tls.key`: TLS Private Key, required for servers and optional for clients. Clients must provide it if the server requires
    Client Authentication. To supply multiple private keys, separate their paths using colons (`:`).
- TLS Server: 
  - `tls.clientca`: TLS Client Certificate Authorities. Optional. If set, users will be required to authenticate using a certificate that has to be signed with these certificates.
    To supply multiple Client CAs, separate their paths using colons (`:`).
- TCP Client:
  - `tcp.keepalive`: TCP Keepalive. Default is disabled.
  - `tcp.dial_timeout`: Dial timeout. Default is 5s.

Note that multiple declaration of each option is not supported but some options support separators for multiple values.

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

This SOCKS5 server supports authentication, ruleset and rewrites and these are configurable through both transport parameters and
URL query parameters.
- `socks5.username` and `socks5.password`: For a simple single-user authentication method, you may use this. 
- `socks5.credentials`: For a multi-user authentication method, you may supply a file containing credentials. Usernames and passwords are separated by colons (`:`) in each line.
- `socks5.ruleset`: Path to a file containing ruleset in the following format: `ACTION,ADDRESS,PORT` in each line, where action can be any of "allow" and "deny" and address can be either IPv4 address, CIDR range, FQDN with wildcard support.
- `socks5.rewrites`: Path to a file containing `ADDRESS,PORT,TARGETADDR,TARGETPORT` in lines.

Addresses can be in the following format:
- `F:google.com`
- `F:www.*exam*le.co*`
- `F:*.google.com`
- `192.168.0.0/24`
- `192.168.10.0-192.168.20.40`
- `192.168.10.10`

Also port can be in the following format:
- `*`: all
- `90`: just 90
- `90-800`: 90 to 800
- `90 92`: 90 and 92
- `^443`: all but 443

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
