package socks5

import (
	"github.com/armon/go-socks5"
	"github.com/hadi77ir/wsproxy/pkg/utils"
	"net/url"
	"strings"
)

func ParseConfig(params url.Values) (*socks5.Config, error) {
	config := &socks5.Config{
		Logger: nil,
	}
	var err error
	config.AuthMethods, config.Credentials, err = ParseCredentials(params)
	if err != nil {
		return nil, err
	}
	actionStr, actionFound := utils.GetParameter(params, "socks5.action")
	action := ActionAllow
	if actionFound {
		action, err = ParseAction(actionStr)
		if err != nil {
			return nil, err
		}
	}

	if ruleset, found := utils.GetParameter(params, "socks5.ruleset"); found {
		config.Rules, err = ParseRuleset(ruleset, action)
		if err != nil {
			return nil, err
		}
	} else {
		// absolute rule.
		if action == ActionAllow {
			config.Rules = socks5.PermitAll()
		} else {
			config.Rules = socks5.PermitNone()
		}
	}

	if rewrites, found := utils.GetParameter(params, "socks5.rewrites"); found {
		config.Rewriter, err = ParseRewrites(rewrites)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func ParseRewrites(filePath string) (socks5.AddressRewriter, error) {
	fileBytes, err := utils.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(fileBytes), "\n")
	rules := make([]RewriteRule, 0)
	for _, line := range lines {
		line = strings.Trim(line, "\r\n")
		line = strings.TrimLeft(line, "\t ")
		rule, err := ParseRewriteLine(line)
		if err != nil {
			return nil, err
		}
		if rule != nil {
			rules = append(rules, rule)
		}
	}
	return &Rewriter{
		Rules: rules,
	}, nil
}

func ParseRuleset(filePath string, defaultAction RuleAction) (socks5.RuleSet, error) {
	fileBytes, err := utils.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(fileBytes), "\n")
	rules := make([]Rule, 0)
	for _, line := range lines {
		line = strings.Trim(line, "\r\n")
		line = strings.TrimLeft(line, "\t ")
		rule, err := ParseRuleLine(line)
		if err != nil {
			return nil, err
		}
		if rule != nil {
			rules = append(rules, rule)
		}
	}
	return &Ruleset{
		DefaultAction: defaultAction,
		Rules:         rules,
	}, nil
}
