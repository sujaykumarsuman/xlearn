// stdoutflood writes to stdout forever: past the 1 MiB fd cap the front asks for a kill (OLE).
package main

import "os"

func main() {
	buf := make([]byte, 64<<10)
	for i := range buf {
		buf[i] = 'x'
	}
	for {
		if _, err := os.Stdout.Write(buf); err != nil {
			os.Exit(3)
		}
	}
}
