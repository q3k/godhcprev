package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/miekg/dns"
)

func init() {
	flag.Set("logtostderr", "true")
}

type config struct {
	listen       string
	dnsForward   string
	dnsReverseV6 string
	ns           string
}

type server struct {
	c   config
	mux *dns.ServeMux
	srv *dns.Server
}

func (s *server) handleForward(w dns.ResponseWriter, r *dns.Msg) {
	rrs := s.forwardV6For(r)

	m := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative: true,
		},
		Answer: rrs,
	}
	m.SetReply(r)
	w.WriteMsg(&m)
}

func (s *server) handleReverseV6(w dns.ResponseWriter, r *dns.Msg) {
	rrs := s.reverseV6For(r)

	m := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative: true,
		},
		Answer: rrs,
	}
	m.SetReply(r)
	w.WriteMsg(&m)
}

func (c *config) check() error {
	if !strings.HasSuffix(c.dnsReverseV6, ".ip6.arpa.") {
		return fmt.Errorf("DNS Reverse Suffix must end in '.ip6.arpa.'")
	}
	parts := strings.Split(strings.TrimSuffix(c.dnsReverseV6, ".ip6.arpa."), ".")
	if len(parts) != 16 {
		return fmt.Errorf("DNS Reverse Suffix must be for a /64 block")
	}
	if len(c.dnsForward) == 0 {
	}
	if strings.HasPrefix(c.dnsForward, ".") || !strings.HasSuffix(c.dnsForward, ".") {
		return fmt.Errorf("DNS Forward Suffix must be given as 'example.com.' (no leading dot, with trailing dot)")
	}
	return nil
}

func newServer(c config) *server {
	s := &server{
		c:   c,
		mux: dns.NewServeMux(),
		srv: nil,
	}
	s.mux.HandleFunc(c.dnsForward, s.handleForward)
	s.mux.HandleFunc(c.dnsReverseV6, s.handleReverseV6)
	return s
}

func (s *server) shutdown() error {
	return s.srv.Shutdown()
}

func (s *server) listenAndServe() error {
	srv := &dns.Server{
		Addr:      s.c.listen,
		Net:       "udp",
		ReusePort: true,
		Handler:   s.mux,
	}
	s.srv = srv
	return srv.ListenAndServe()
}

func main() {
	c := config{}
	flag.StringVar(&c.listen, "listen", "[::]:53", "DNS address to listen on")
	flag.StringVar(&c.dnsForward, "dns_forward", "dyn.hackerspace.pl.", "DNS forward zone for dynamic leases")
	flag.StringVar(&c.dnsReverseV6, "dns_reverse_v6", "2.4.2.4.2.4.2.4.0.0.b.e.d.0.a.2.ip6.arpa.", "DNS forward zone for dynamic leases")
	flag.StringVar(&c.ns, "ns", "ns1.example.com", "Identity of this NS")
	flag.Parse()
	if err := c.check(); err != nil {
		glog.Fatalf("Configuration error: %v", err)
	}
	glog.Infof("Starting up...")

	s := newServer(c)
	glog.Exit(s.listenAndServe())
}
