// spin burns CPU: forever (TLE) with an empty input, or for the number of milliseconds its
// input names (the L14 tests-cap run spins each case just under its TL).
package main

import (
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	in, _ := io.ReadAll(os.NewFile(3, "input"))
	ms, err := strconv.Atoi(strings.TrimSpace(string(in)))
	deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)
	x := 0
	for err != nil || time.Now().Before(deadline) {
		for i := 0; i < 100000; i++ {
			x += i ^ x
		}
	}
	os.NewFile(4, "result").Write([]byte(strconv.Itoa(x & 1)))
}
