package main

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/miekg/dns"
)

func (s *server) serveNS(r *dns.Msg) []dns.RR {
	rr, err := dns.NewRR(fmt.Sprintf("%s 60 NS %s", r.Question[0].Name, s.c.ns))
	if err != nil {
		glog.Errorf("NewRR: %v", err)
		return []dns.RR{}
	}

	return []dns.RR{rr}
}

func (s *server) forwardV6For(r *dns.Msg) []dns.RR {
	rrs := []dns.RR{}

	question := ""
	for _, q := range r.Question {
		if q.Qtype == dns.TypeNS {
			return s.serveNS(r)
		}
		if q.Qtype != dns.TypeAAAA && q.Qtype != dns.TypeA {
			break
		}
		if strings.HasSuffix(q.Name, s.c.dnsForward) {
			question = q.Name
			break
		}
	}
	if question == "" {
		return rrs
	}

	question = strings.ToLower(question)

	prefix := strings.TrimSuffix(question, s.c.dnsForward)
	prefix = strings.TrimSuffix(prefix, ".")
	prefix = strings.TrimSpace(prefix)

	if len(prefix) != 16 {
		return rrs
	}

	record := ""
	suffixParts := strings.Split(strings.TrimSuffix(s.c.dnsReverseV6, ".ip6.arpa."), ".")
	for i, _ := range suffixParts {
		p := string(suffixParts[15-i])
		record += p
		if i%4 == 3 {
			record += ":"
		}
	}
	for i, p := range prefix {
		if !strings.Contains("0123456789abcdef", string(p)) {
			return rrs
		}
		record += string(p)
		if i%4 == 3 {
			record += ":"
		}
	}
	record = strings.TrimSuffix(record, ":")

	rr, err := dns.NewRR(fmt.Sprintf("%s 60 AAAA %s", r.Question[0].Name, record))
	if err != nil {
		glog.Errorf("NewRR: %v", err)
		return rrs
	}

	return []dns.RR{rr}
}

func (s *server) reverseV6For(r *dns.Msg) []dns.RR {
	rrs := []dns.RR{}

	question := ""
	for _, q := range r.Question {
		if strings.HasSuffix(q.Name, s.c.dnsReverseV6) {
			question = q.Name
			break
		}
	}
	if question == "" {
		return rrs
	}

	question = strings.ToLower(question)

	prefix := strings.TrimSuffix(question, s.c.dnsReverseV6)
	prefix = strings.TrimSuffix(prefix, ".")
	prefix = strings.TrimSpace(prefix)

	parts := strings.Split(prefix, ".")
	if len(parts) != 16 {
		return rrs
	}

	record := ""
	for i, _ := range parts {
		p := parts[(len(parts)-1)-i]
		if len(p) != 1 {
			return rrs
		}
		if !strings.Contains("0123456789abcdef", p) {
			return rrs
		}
		record += p
	}

	record += "."
	record += s.c.dnsForward

	rr, err := dns.NewRR(fmt.Sprintf("%s 60 IN PTR %s", r.Question[0].Name, record))
	if err != nil {
		glog.Errorf("NewRR: %v", err)
		return rrs
	}

	return []dns.RR{rr}
}
