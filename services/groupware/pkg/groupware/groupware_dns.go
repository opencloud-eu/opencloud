package groupware

import (
	"errors"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

var (
	errDnsNoServerToAnswer = errors.New("no name server to resolve") // TODO better error message
)

type DnsSessionUrlResolver struct {
	defaultSessionUrl *url.URL
	defaultDomain     string
	domainGreenList   []string
	domainRedList     []string
	config            *dns.ClientConfig
	client            *dns.Client
}

func NewDnsSessionUrlResolver(defaultSessionUrl *url.URL, defaultDomain string,
	config *dns.ClientConfig, domainGreenList []string, domainRedList []string,
	dialTimeout time.Duration, readTimeout time.Duration,
) (DnsSessionUrlResolver, error) {
	// TODO the whole udp or tcp dialier configuration, see https://github.com/miekg/exdns/blob/master/q/q.go

	c := &dns.Client{
		DialTimeout: dialTimeout,
		ReadTimeout: readTimeout,
	}

	return DnsSessionUrlResolver{
		defaultSessionUrl: defaultSessionUrl,
		defaultDomain:     defaultDomain,
		config:            config,
		client:            c,
	}, nil
}

func (d DnsSessionUrlResolver) isGreenListed(domain string) bool {
	if d.domainGreenList == nil {
		return true
	}
	// normalize the domain name by stripping a potential "." at the end
	if strings.HasSuffix(domain, ".") {
		domain = domain[0 : len(domain)-2]
	}
	return slices.Contains(d.domainGreenList, domain)
}

func (d DnsSessionUrlResolver) isRedListed(domain string) bool {
	if d.domainRedList == nil {
		return true
	}
	// normalize the domain name by stripping a potential "." at the end
	if strings.HasSuffix(domain, ".") {
		domain = domain[0 : len(domain)-2]
	}
	return !slices.Contains(d.domainRedList, domain)
}

func (d DnsSessionUrlResolver) Resolve(username string) (*url.URL, *GroupwareError) {
	// heuristic to detect whether the username is an email address
	parts := strings.Split(username, "@")
	domain := d.defaultDomain
	if len(parts) <= 1 {
		// it's not, but do we have a defaultDomain configured that we should use
		// nevertheless then?
		if d.defaultDomain == "" {
			// we don't, then let's fall back to the static session URL instead
			return d.defaultSessionUrl, nil
		}
	} else {
		domain = parts[len(parts)-1]
		if !d.isGreenListed(domain) {
			return nil, &ErrorUsernameEmailDomainIsNotGreenlisted
		}
		if d.isRedListed(domain) {
			return nil, &ErrorUsernameEmailDomainIsRedlisted
		}
	}

	// https://jmap.io/spec-core.html#service-autodiscovery
	//
	// A JMAP-supporting host for the domain example.com SHOULD publish a
	//   SRV record _jmap._tcp.example.com
	// that gives a hostname and port (usually port 443).
	//
	// The JMAP Session resource is then https://${hostname}[:${port}]/.well-known/jmap
	// (following any redirects).

	// we need a fully qualified domain name: must end with a dot
	name := dns.Fqdn("_jmap._tcp." + domain)

	msg := &dns.Msg{
		MsgHdr:   dns.MsgHdr{RecursionDesired: true},
		Question: make([]dns.Question, 1),
	}
	msg.SetQuestion(name, dns.TypeSRV)

	r, err := d.dnsQuery(d.client, msg)
	if err != nil {
		// TODO error
	}
	if r == nil || r.Rcode == dns.RcodeNameError {
		// TODO domain not found
	}

	for _, ans := range r.Answer {
		switch t := ans.(type) {
		case *dns.SRV:
			scheme := "https"
			host := t.Target // TODO need to check whether the hostname is indeed in t.Target?
			port := t.Port
			if (scheme == "https" && port != 443) || (scheme == "http" && port != 80) {
				host = net.JoinHostPort(host, strconv.Itoa(int(port)))
			}

			u := &url.URL{
				Scheme: scheme,
				Host:   host,
				Path:   "/.well-known/jmap",
			}

			return u, nil
		}
	}

	return d.defaultSessionUrl, nil
}

func (d DnsSessionUrlResolver) dnsQuery(c *dns.Client, msg *dns.Msg) (*dns.Msg, error) {
	for _, server := range d.config.Servers {
		address := ""
		// if the server is IPv6, it is already expected to be wrapped in [brackets] when
		// the configuration comes from /etc/resolv.conf and has been parsed using
		// dns.ClientConfigFromFile, but let's check to make sure
		if strings.HasPrefix(server, "[") && strings.HasSuffix(server, "]") {
			address = server + ":" + d.config.Port
		} else {
			// this function will take care of properly wrapping in [brackets] if it's
			// an IPv6 address string:
			address = net.JoinHostPort(server, d.config.Port)
		}

		r, _, err := c.Exchange(msg, address)
		if err != nil {
			return nil, err
		}
		if r == nil || r.Rcode == dns.RcodeNameError || r.Rcode == dns.RcodeSuccess {
			return r, err
		}
	}
	return nil, errDnsNoServerToAnswer
}
