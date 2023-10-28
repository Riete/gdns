package handler

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/riete/mkv/v2"

	"github.com/miekg/dns"
	"golang.org/x/net/proxy"
)

var cacheDefaultTTL = 30 * time.Second

type UpstreamForwarder struct {
	dnsServer []string
	client    *dns.Client
	dial      func(network, addr string) (net.Conn, error)
	store     *mkv.KVStorage[*dns.Msg]
}

func (u *UpstreamForwarder) readCache(domain string, qType uint16) (*dns.Msg, bool) {
	msg, err := u.store.Get(fmt.Sprintf("%s-%d", domain, qType))
	return msg, err == nil
}

func (u *UpstreamForwarder) writeCache(domain string, qType uint16, msg *dns.Msg) {
	u.store.Set(fmt.Sprintf("%s-%d", domain, qType), msg)
}

func (u *UpstreamForwarder) exchange(r *dns.Msg) (*dns.Msg, error) {
	domain := r.Question[0].Name
	qType := r.Question[0].Qtype
	if msg, ok := u.readCache(domain, qType); ok {
		return msg, nil
	}
	msg := new(dns.Msg)
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
			u.writeCache(domain, qType, resp)
			return resp, nil
		}
	}
	return resp, errors.New("query from all upstream dns server failed")
}

func (u *UpstreamForwarder) SetDial(dial func(network, addr string) (net.Conn, error)) {
	u.dial = dial
}

func (u *UpstreamForwarder) SetCacheTTL(ttl time.Duration) {
	u.store = mkv.NewKVStorage(ttl, &dns.Msg{})
}

func (u *UpstreamForwarder) SetUpstreamServer(dnsServer ...string) {
	u.dnsServer = dnsServer
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
// dnsServer is ip:port
func NewUpstreamForwarder(net string, dnsServer ...string) *UpstreamForwarder {
	return &UpstreamForwarder{
		dnsServer: dnsServer,
		client:    &dns.Client{Net: net},
		store:     mkv.NewKVStorage(cacheDefaultTTL, &dns.Msg{}),
	}
}

// NewProxyUpstreamForwarder
// net is tcp or udp
// proxyAddr is socks5 proxy server address
// dnsServer is ip:port
func NewProxyUpstreamForwarder(net, proxyAddr string, dnsServer ...string) (*UpstreamForwarder, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}
	return &UpstreamForwarder{
		dnsServer: dnsServer,
		client:    &dns.Client{Net: net},
		dial:      dialer.Dial,
		store:     mkv.NewKVStorage(cacheDefaultTTL, &dns.Msg{}),
	}, nil
}
