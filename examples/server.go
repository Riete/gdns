package main

import (
	"log"

	"github.com/riete/gdns/handler"

	"github.com/riete/gdns"
)

func main() {
	// proxyHandler, _ := handler.NewProxyUpstreamForwarder("udp", "127.0.0.1:2223", "127.0.0.1:11", "100.100.2.136:53")
	// s := gdns.NewTcpDnsServer(
	// 	"127.0.0.1",
	// 	"10053",
	// 	proxyHandler,
	// )

	handler := handler.NewUpstreamForwarder("udp", "127.0.0.1:11", "223.5.5.5:53")
	s := gdns.NewTcpDnsServer(
		"127.0.0.1",
		"10053",
		handler,
	)
	log.Println(s.ListenAndServe())
}
