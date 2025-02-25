package handler

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/riete/go-cache"

	"github.com/miekg/dns"
	"golang.org/x/net/proxy"
)

type UpstreamForwarder struct {
	upstreams   []string
	client      *dns.Client
	dial        func(network, addr string) (net.Conn, error)
	cache       *cache.Cache[*dns.Msg]
	cacheEnable bool
}

func (u *UpstreamForwarder) readCache(domain string, qType uint16) (*dns.Msg, bool) {
	msg, err := u.cache.Get(fmt.Sprintf("%s-%d", domain, qType))
	return msg, err
}

func (u *UpstreamForwarder) writeCache(domain string, qType uint16, msg *dns.Msg) {
	u.cache.Set(fmt.Sprintf("%s-%d", domain, qType), msg)
}

func (u *UpstreamForwarder) exchange(r *dns.Msg) (*dns.Msg, error) {
	domain := r.Question[0].Name
	qType := r.Question[0].Qtype
	log.Println("query domain:", domain, "query type:", dns.Type(qType).String())
	if u.cacheEnable {
		if msg, ok := u.readCache(domain, qType); ok {
			log.Println("response from cache:")
			log.Println(msg)
			return msg, nil
		}
	}
	msg := new(dns.Msg)
	msg.SetQuestion(domain, qType)

	var resp *dns.Msg
	var err error
	conn := new(dns.Conn)
	for _, server := range u.upstreams {
		if u.dial == nil {
			conn, err = u.client.Dial(server)
		} else {
			conn.Conn, err = u.dial(u.client.Net, server)
		}

		if err == nil {
			resp, _, err = u.client.ExchangeWithConn(msg, conn)
			_ = conn.Close()
		}

		if err == nil {
			log.Printf("response from upstream: %s", server)
			log.Println(resp)
			if u.cacheEnable {
				u.writeCache(domain, qType, resp)
			}
			return resp, nil
		}
		log.Println(err)
	}
	return resp, errors.New("query from all upstream dns server failed")
}

func (u *UpstreamForwarder) EnableCache(ttl time.Duration) {
	u.cacheEnable = true
	c := cache.NewDefaultConfig()
	c.TTL = ttl
	u.cache = cache.New[*dns.Msg](c)
}

func (u *UpstreamForwarder) SetProxy(proxyAddr string) error {
	dial, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err == nil {
		u.client.Net = "tcp"
		u.dial = dial.Dial
	}
	return err
}

func (u *UpstreamForwarder) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg, err := u.exchange(r)
	if err != nil {
		log.Println(err)
		msg = new(dns.Msg)
		msg.Answer = []dns.RR{}
	}
	msg.SetReply(r)
	if err = w.WriteMsg(msg); err != nil {
		log.Println("send dns query reply failed:", err)
	}
}

// NewUpstreamForwarder
// net is tcp or udp
// upstreams is ip:port
func NewUpstreamForwarder(net string, upstreams ...string) *UpstreamForwarder {
	return &UpstreamForwarder{upstreams: upstreams, client: &dns.Client{Net: net}}
}

func NewTcpUpstreamForwarder(upstreams ...string) *UpstreamForwarder {
	return NewUpstreamForwarder("tcp", upstreams...)
}

func NewUdpUpstreamForwarder(upstreams ...string) *UpstreamForwarder {
	return NewUpstreamForwarder("udp", upstreams...)
}

// NewProxyUpstreamForwarder
// proxyAddr is socks5 proxy server address
// upstreams is ip:port
func NewProxyUpstreamForwarder(proxyAddr string, upstreams ...string) (*UpstreamForwarder, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}
	return &UpstreamForwarder{upstreams: upstreams, client: &dns.Client{Net: "tcp"}, dial: dialer.Dial}, nil
}
