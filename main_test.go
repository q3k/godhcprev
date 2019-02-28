package main

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/miekg/dns"
)

const (
	testForward   = "example.com."
	testReverseV6 = "2.4.2.4.2.4.2.4.0.0.b.e.d.0.a.2.ip6.arpa."
)

func harness() chan interface{} {
	s := newServer(config{
		listen:       "[::]:8053",
		dnsForward:   testForward,
		dnsReverseV6: testReverseV6,
	})

	wg := sync.WaitGroup{}
	wg.Add(1)
	stopC := make(chan interface{})
	go func() {
		errC := make(chan error)
		go func() {
			errC <- s.listenAndServe()
		}()

		wg.Done()
		select {
		case <-stopC:
			s.shutdown()
			return
		case err := <-errC:
			glog.Error(err)
		}
	}()

	wg.Wait()
	// No idea why, but the DNS server takes a while to come up.
	time.Sleep(100 * time.Millisecond)

	return stopC
}

func TestServeV6(t *testing.T) {
	stopC := harness()

	c := dns.Client{}
	c.Net = "udp"
	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:     dns.Id(),
			Opcode: dns.OpcodeQuery,
		},
		Question: []dns.Question{
			dns.Question{
				Name:   "2.1.3.7.2.1.3.7.2.1.3.7.2.1.3.7." + testReverseV6,
				Qtype:  dns.TypePTR,
				Qclass: dns.ClassINET,
			},
		},
	}
	r, _, err := c.Exchange(m, "127.0.0.1:8053")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(r.Answer), 1; got != want {
		t.Fatalf("Expected %d answer, got %d", want, got)
	}
	fields := strings.Fields(r.Answer[0].String())
	if got, want := fields[len(fields)-1], "7312731273127312."+testForward; got != want {
		t.Fatalf("Expected answer %q, got %q", want, got)
	}

	close(stopC)
}

func TestServeForward(t *testing.T) {
	stopC := harness()

	c := dns.Client{}
	c.Net = "udp"
	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:     dns.Id(),
			Opcode: dns.OpcodeQuery,
		},
		Question: []dns.Question{
			dns.Question{
				Name:   "deadbeef21372137." + testForward,
				Qtype:  dns.TypeAAAA,
				Qclass: dns.ClassINET,
			},
		},
	}
	r, _, err := c.Exchange(m, "127.0.0.1:8053")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(r.Answer), 1; got != want {
		t.Fatalf("Expected %d answer, got %d", want, got)
	}
	fields := strings.Fields(r.Answer[0].String())
	if got, want := fields[len(fields)-1], "2a0d:eb00:4242:4242:dead:beef:2137:2137"; got != want {
		t.Fatalf("Expected answer %q, got %q", want, got)
	}

	close(stopC)
}
