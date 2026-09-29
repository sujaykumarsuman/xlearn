// netprobe tries TCP to 1.1.1.1:443, UDP to kube-dns 10.43.0.10:53 and TCP to loopback: in an
// empty network namespace with loopback down every one must fail. It prints each result and
// exits with the number that got through.
package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	reached := 0
	for _, t := range []struct{ network, addr string }{
		{"tcp", "1.1.1.1:443"},
		{"tcp", "127.0.0.1:80"},
		{"udp", "10.43.0.10:53"},
	} {
		c, err := net.DialTimeout(t.network, t.addr, time.Second)
		if err != nil {
			fmt.Println(t.network, t.addr, "blocked:", err)
			continue
		}
		if t.network == "udp" {
			// UDP "connects" without a handshake; only an answer counts.
			c.SetDeadline(time.Now().Add(time.Second))
			c.Write([]byte{0x12, 0x34, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1})
			b := make([]byte, 512)
			if _, err := c.Read(b); err != nil {
				fmt.Println(t.network, t.addr, "no answer:", err)
				c.Close()
				continue
			}
		}
		c.Close()
		fmt.Println(t.network, t.addr, "REACHED")
		reached++
	}
	os.Exit(reached)
}
