package socks5

import (
	"context"
	"github.com/armon/go-socks5"
	"github.com/gertd/wild"
	"github.com/hadi77ir/wsproxy/pkg/errors"
	"github.com/russtone/iprange"
	"strings"
)

type RewriteRule interface {
	Rewrite(addrSpec *socks5.AddrSpec) (*socks5.AddrSpec, bool)
}

type Rewriter struct {
	Rules []RewriteRule
}

func (r *Rewriter) Rewrite(ctx context.Context, request *socks5.Request) (context.Context, *socks5.AddrSpec) {
	for _, rule := range r.Rules {
		if addr, matched := rule.Rewrite(request.DestAddr); matched {
			return ctx, addr
		}
	}
	return ctx, request.DestAddr
}

var _ socks5.AddressRewriter = &Rewriter{}

type IPRangeRewriter struct {
	AddrRanges  iprange.Ranges
	PortMatcher PortMatcherFunc
	Target      *socks5.AddrSpec
}

func (r *IPRangeRewriter) Rewrite(addrSpec *socks5.AddrSpec) (*socks5.AddrSpec, bool) {
	// check for any of ranges
	if r.AddrRanges.Contains(addrSpec.IP) && r.PortMatcher(addrSpec.Port) {
		return r.Target, true
	}
	return addrSpec, false
}

var _ RewriteRule = &IPRangeRewriter{}

type FQDNRewriter struct {
	WildcardPattern string
	PortMatcher     PortMatcherFunc
	Target          *socks5.AddrSpec
}

func (r *FQDNRewriter) Rewrite(addrSpec *socks5.AddrSpec) (*socks5.AddrSpec, bool) {
	if wild.Match(r.WildcardPattern, addrSpec.FQDN, true) && r.PortMatcher(addrSpec.Port) {
		return r.Target, true
	}
	return addrSpec, false
}

var _ RewriteRule = &FQDNRewriter{}

func ParseRewriteLine(line string) (RewriteRule, error) {
	if strings.HasPrefix(line, "#") {
		return nil, nil
	}
	if len(line) == 0 {
		return nil, nil
	}
	parts := strings.SplitN(line, ",", 4)
	if len(parts) != 4 {
		return nil, errors.ErrInvalidSyntax
	}
	target, err := ParseAddrSpec(parts[2], parts[3])
	if err != nil {
		return nil, err
	}
	portMatcher, err := CreatePortMatcher(parts[1])
	if err != nil {
		return nil, err
	}
	// if it is going to be an FQDN, there has to be "F:" prefix
	if strings.HasPrefix(parts[0], "F:") {
		return &FQDNRewriter{WildcardPattern: parts[0][2:], PortMatcher: portMatcher, Target: target}, nil
	}
	// then it has to be ip range list.
	rangesStrParts := strings.Split(parts[0], " ")
	rangeList := make([]iprange.Range, len(rangesStrParts))
	for i, rangesStrPart := range rangesStrParts {
		parsed := iprange.Parse(rangesStrPart)
		if parsed == nil {
			return nil, errors.ErrInvalidSyntax
		}
		rangeList[i] = parsed
	}
	return &IPRangeRewriter{AddrRanges: rangeList, PortMatcher: portMatcher, Target: target}, nil
}
