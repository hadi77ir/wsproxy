package socks5

import (
	"github.com/armon/go-socks5"
	E "github.com/hadi77ir/wsproxy/pkg/errors"
	"github.com/hadi77ir/wsproxy/pkg/utils"
	"net/url"
	"strings"
)

func ParseCredentials(params url.Values) ([]socks5.Authenticator, socks5.CredentialStore, error) {
	username, usernameFound := utils.GetParameter(params, "socks5.username")
	password, passwordFound := utils.GetParameter(params, "socks5.password")
	authenticationEnabled := false
	credentials := socks5.StaticCredentials{}
	if usernameFound != passwordFound {
		return nil, nil, E.ErrMissingPart("socks5 credentials must be a pair")
	}
	if usernameFound && passwordFound {
		authenticationEnabled = true
		credentials[username] = password
	}

	if authList, found := utils.GetParameter(params, "socks5.credentials"); found {
		listBytes, err := utils.ReadFile(authList)
		if err != nil {
			return nil, nil, err
		}
		lines := strings.Split(string(listBytes), "\n")
		for _, line := range lines {
			line = strings.Trim(line, "\r\n")
			line = strings.TrimLeft(line, "\t ")
			if strings.HasPrefix(line, "#") {
				continue
			}
			if len(line) == 0 {
				continue
			}
			username, password, found := strings.Cut(line, ":")
			if !found || len(username) == 0 || len(password) == 0 {
				return nil, nil, E.ErrInvalidSyntax
			}
			credentials[username] = password
		}
	}

	if authenticationEnabled {
		return []socks5.Authenticator{&socks5.UserPassAuthenticator{credentials}}, credentials, nil
	}
	return nil, nil, nil
}
