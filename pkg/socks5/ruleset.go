package socks5

import (
	"errors"
	"github.com/armon/go-socks5"
	"github.com/gertd/wild"
	E "github.com/hadi77ir/wsproxy/pkg/errors"
	"github.com/russtone/iprange"
	"golang.org/x/net/context"
	"strconv"
	"strings"
)

type PortMatcherFunc func(port int) bool

type RuleAction int

const (
	ActionContinue = RuleAction(0)
	ActionAllow    = RuleAction(1)
	ActionBlock    = RuleAction(2)
)

type Rule interface {
	Match(addrSpec *socks5.AddrSpec) (RuleAction, bool)
}

type IPRangeRule struct {
	AddrRanges  iprange.Ranges
	PortMatcher PortMatcherFunc
	Action      RuleAction
}

func (r *IPRangeRule) Match(addrSpec *socks5.AddrSpec) (RuleAction, bool) {
	// check for any of ranges
	if r.AddrRanges.Contains(addrSpec.IP) && r.PortMatcher(addrSpec.Port) {
		return r.Action, true
	}
	return ActionContinue, false
}

var _ Rule = &IPRangeRule{}

type FQDNRule struct {
	WildcardPattern string
	PortMatcher     PortMatcherFunc
	Action          RuleAction
}

func (r *FQDNRule) Match(addrSpec *socks5.AddrSpec) (RuleAction, bool) {
	if wild.Match(r.WildcardPattern, addrSpec.FQDN, true) {
		if r.PortMatcher(addrSpec.Port) {
			return r.Action, true
		}
	}
	return ActionContinue, false
}

var _ Rule = &FQDNRule{}

type Ruleset struct {
	Rules         []Rule
	DefaultAction RuleAction
}

func (r *Ruleset) Allow(ctx context.Context, req *socks5.Request) (context.Context, bool) {
	for _, rule := range r.Rules {
		if action, matched := rule.Match(req.DestAddr); matched {
			return ctx, action == ActionAllow
		}
	}

	return ctx, r.DefaultAction == ActionAllow
}

var _ socks5.RuleSet = &Ruleset{}

func CreatePortMatcher(portsDef string) (PortMatcherFunc, error) {
	parts := strings.Split(portsDef, " ")
	funcs := make([]PortMatcherFunc, len(parts))
	var err error
	for i, part := range parts {
		funcs[i], err = createPortMatcherSingle(part)
		if err != nil {
			return nil, err
		}
	}
	return func(port int) bool {
		for _, matcherFunc := range funcs {
			if matcherFunc(port) {
				return true
			}
		}
		return false
	}, nil
}
func createPortMatcherSingle(portsDef string) (PortMatcherFunc, error) {
	matchedTrue := true
	if strings.HasPrefix(portsDef, "^") {
		// negate
		matchedTrue = false
	}
	portsDef = portsDef[1:]

	if portsDef == "*" {
		return func(_ int) bool {
			return matchedTrue
		}, nil
	}

	min, max, dashFound := strings.Cut(portsDef, "-")
	if dashFound {
		minInt, err := ParsePort(min)
		if err != nil {
			return nil, err
		}
		maxInt, err := ParsePort(max)
		if err != nil {
			return nil, err
		}

		return func(port int) bool {
			if port >= minInt && port < maxInt {
				return matchedTrue
			}
			return !matchedTrue
		}, nil
	}

	exactPort, err := ParsePort(max)
	if err != nil {
		return nil, err
	}
	return func(port int) bool {
		if exactPort == port {
			return matchedTrue
		}
		return !matchedTrue
	}, nil
}

var ErrInvalidPort = errors.New("port has to be 0-65535")

func ParsePort(port string) (int, error) {
	parsed, err := strconv.ParseInt(port, 10, 32)
	if err != nil {
		return 0, err
	}
	if parsed < 0 || parsed > 65536 {
		return 0, ErrInvalidPort
	}
	return int(parsed), nil
}

func ParseAction(input string) (RuleAction, error) {
	input = strings.ToLower(input)
	if input == "allow" || input == "accept" || input == "approve" {
		return ActionAllow, nil
	}
	if input == "reject" || input == "deny" || input == "block" {
		return ActionBlock, nil
	}
	return ActionContinue, E.ErrInvalidSyntax
}
func ParseRuleLine(line string) (Rule, error) {
	if strings.HasPrefix(line, "#") {
		return nil, nil
	}
	if len(line) == 0 {
		return nil, nil
	}
	parts := strings.SplitN(line, ",", 3)
	if len(parts) != 3 {
		return nil, E.ErrInvalidSyntax
	}
	action, err := ParseAction(parts[0])
	if err != nil {
		return nil, err
	}
	portMatcher, err := CreatePortMatcher(parts[2])
	if err != nil {
		return nil, err
	}
	// if it is going to be an FQDN, there has to be "F:" prefix
	if strings.HasPrefix(parts[1], "F:") {
		return &FQDNRule{Action: action, WildcardPattern: parts[1][2:], PortMatcher: portMatcher}, nil
	}
	// then it has to be ip range list.
	rangesStrParts := strings.Split(parts[1], " ")
	rangeList := make([]iprange.Range, len(rangesStrParts))
	for i, rangesStrPart := range rangesStrParts {
		parsed := iprange.Parse(rangesStrPart)
		if parsed == nil {
			return nil, E.ErrInvalidSyntax
		}
		rangeList[i] = parsed
	}
	return &IPRangeRule{AddrRanges: rangeList, PortMatcher: portMatcher, Action: action}, nil
}
