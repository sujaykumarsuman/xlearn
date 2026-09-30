// echo copies its case input (fd 3) to the harness channel (fd 4) and says "ok" on stdout and
// "note" on stderr.
package main

import (
	"io"
	"os"
)

func main() {
	in, err := io.ReadAll(os.NewFile(3, "input"))
	if err != nil {
		os.Exit(3)
	}
	if _, err := os.NewFile(4, "result").Write(in); err != nil {
		os.Exit(4)
	}
	os.Stdout.Write([]byte("ok\n"))
	os.Stderr.Write([]byte("note\n"))
}
