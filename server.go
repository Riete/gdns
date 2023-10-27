package gdns

import (
	"log"

	"github.com/miekg/dns"
)

type Server struct {
	s *dns.Server
}

func (s Server) ListenAndServe() error {
	log.Printf("dns server listen at: %s://%s\n", s.s.Net, s.s.Addr)
	return s.s.ListenAndServe()
}

func (s *Server) Shutdown() error {
	return s.s.Shutdown()
}

func NewTcpDnsServer(ip, port string, handler dns.Handler) *Server {
	return &Server{s: &dns.Server{Addr: ip + ":" + port, Net: "tcp", Handler: handler}}
}

func NewUdpDnsServer(ip, port string, handler dns.Handler) *Server {
	return &Server{s: &dns.Server{Addr: ip + ":" + port, Net: "udp", Handler: handler}}

}
