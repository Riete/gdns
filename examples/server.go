package main

import (
	"log"
	"time"

	"github.com/riete/gdns/handler"

	"github.com/riete/gdns"
)

func main() {
	h := handler.NewUdpUpstreamForwarder("223.5.5.5:53")
	// h, _ := handler.NewProxyUpstreamForwarder("127.0.0.1:7890", "223.5.5.5:53")
	h.EnableCache(30 * time.Second)
	s := gdns.NewUdpDnsServer("127.0.0.1", "10053", h)
	log.Println(s.ListenAndServe())
}
