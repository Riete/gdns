package gdns

import (
	"errors"
	"log"
	"net"

	"github.com/miekg/dns"
	"golang.org/x/net/proxy"
)

type UpstreamForwarder struct {
	dnsServer []string
	client    *dns.Client
	dial      func(network, addr string) (net.Conn, error)
}

func (u UpstreamForwarder) exchange(r *dns.Msg) (*dns.Msg, error) {
	domain := r.Question[0].Name
	qType := r.Question[0].Qtype
	msg := new(dns.Msg)
	msg.RecursionDesired = true
	msg.SetQuestion(domain, qType)

	var resp *dns.Msg
	var err error
	var conn net.Conn
	for _, server := range u.dnsServer {
		if u.dial == nil {
			resp, _, err = u.client.Exchange(msg, server)
		} else {
			conn, err = u.dial("tcp", server)
			if err == nil {
				resp, _, err = u.client.ExchangeWithConn(msg, &dns.Conn{Conn: conn})
				_ = conn.Close()
			}
		}
		if err == nil {
			return resp, nil
		}
	}
	return resp, errors.New("query from all upstream dns server failed")
}

func (u UpstreamForwarder) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg, err := u.exchange(r)
	if err != nil {
		log.Println(err)
		msg = new(dns.Msg)
		msg.Answer = []dns.RR{}
	}
	msg.SetReply(r)
	msg.Authoritative = true
	if err = w.WriteMsg(msg); err != nil {
		log.Println("send dns query reply failed:", err)
	}
}

// NewUpstreamForwarder net is tcp or udp
func NewUpstreamForwarder(net string, dnsServer ...string) dns.Handler {
	return &UpstreamForwarder{dnsServer: dnsServer, client: &dns.Client{Net: net}}
}

// NewProxyUpstreamForwarder net is tcp or udp, proxyAddr is socks5 proxy server address
func NewProxyUpstreamForwarder(net, proxyAddr string, dnsServer ...string) (dns.Handler, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}
	return &UpstreamForwarder{dnsServer: dnsServer, client: &dns.Client{Net: net}, dial: dialer.Dial}, nil
}
