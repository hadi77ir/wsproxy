package utils

import (
	E "github.com/hadi77ir/wsproxy/pkg/errors"
	"net/url"
	"strings"
)

func ParseTransportParamsFromFlags(arr []string) (url.Values, error) {
	result := make(url.Values)
	for _, str := range arr {
		key, value, found := strings.Cut(str, "=")
		if !found {
			return nil, E.ErrInvalidSyntax
		}
		result.Set(key, value)
	}
	return result, nil
}
