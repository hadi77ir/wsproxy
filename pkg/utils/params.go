package utils

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var BoolParseError = errors.New("bad value for bool field") // placeholder not passed to user

func QueryParametersWithoutPrefix(input url.Values, prefix string) url.Values {
	cleaned := make(map[string][]string)
	for k, v := range input {
		if !strings.HasPrefix(k, prefix) {
			cleaned[k] = v
		}
	}
	return cleaned
}

func GetParameter(parameters url.Values, key string) (string, bool) {
	if values, found := parameters[key]; found {
		if len(values) > 0 {
			return values[0], true
		}
	}
	return "", false
}

func DurationFromParameters(params url.Values, key string, defaultValue time.Duration) time.Duration {
	if value, found := GetParameter(params, key); found {
		parsed, err := time.ParseDuration(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func MultiStringFromParameters(params url.Values, key string, defaultValue []string) []string {
	if value, found := GetParameter(params, key); found {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func StringFromParameters(params url.Values, key string, defaultValue string) string {
	if value, found := GetParameter(params, key); found {
		return value
	}
	return defaultValue
}

func IntegerFromParameters(params url.Values, key string, defaultValue int) int {
	if value, found := GetParameter(params, key); found {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func BoolFromParameters(params url.Values, key string, defaultValue bool) bool {
	if value, found := GetParameter(params, key); found {
		parsed, err := ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func ParseBool(value string) (bool, error) {
	if StrIsTrue(value) {
		return true, nil
	}
	if StrIsFalse(value) {
		return false, nil
	}
	return false, BoolParseError
}
func StrIsTrue(value string) bool {
	value = strings.ToLower(value)
	if value == "1" || value == "y" || value == "yes" || value == "true" || value == "t" {
		return true
	}
	return false
}
func StrIsFalse(value string) bool {
	value = strings.ToLower(value)
	if value == "0" || value == "n" || value == "no" || value == "false" || value == "f" {
		return true
	}
	return false
}

func MergeParams(a, b map[string][]string) map[string][]string {
	c := make(map[string][]string)
	for k, v := range a {
		c[k] = append(c[k], v...)
	}
	for k, v := range b {
		c[k] = append(c[k], v...)
	}
	return c
}
