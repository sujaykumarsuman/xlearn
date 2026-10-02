package main

// Section B (network), Go: arg is "<kind>:<target>". Every attempt must fail. The exec filter
// kills socket() (signal SIGSYS), and past it the jail's network namespace is empty with loopback
// down. "blocked: ..." means the attempt failed in-process; "REACHED ..." means a connection or
// an answer got through, which fails the suite.

import (
	"net"
	"os"
	"strings"
	"time"
)

// dnsQuery is a minimal DNS query (id 0x1234, RD, one question: "." IN A).
var dnsQuery = []byte{0x12, 0x34, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1}

func probe(arg string) string {
	kind, target, _ := strings.Cut(arg, ":")
	os.Stderr.WriteString("probing " + arg + "\n")
	switch kind {
	case "tcp", "udp":
		c, err := net.DialTimeout(kind, target, time.Second)
		if err != nil {
			return "blocked: " + err.Error()
		}
		defer c.Close()
		if kind == "udp" {
			c.SetDeadline(time.Now().Add(time.Second))
			c.Write(dnsQuery)
			b := make([]byte, 512)
			if _, err := c.Read(b); err != nil {
				return "blocked: no answer: " + err.Error()
			}
		}
		return "REACHED " + arg
	case "dns":
		addrs, err := net.LookupHost(target)
		if err != nil {
			return "blocked: " + err.Error()
		}
		return "REACHED " + strings.Join(addrs, ",")
	case "bind":
		l, err := net.Listen("tcp", target)
		if err != nil {
			return "blocked: " + err.Error()
		}
		l.Close()
		return "REACHED bound " + target
	}
	return "bad probe " + arg
}
