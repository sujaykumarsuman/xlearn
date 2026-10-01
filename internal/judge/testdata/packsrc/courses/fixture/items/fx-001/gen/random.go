// Command random prints one random fx-001 input: an array of 2..max-n small integers.
package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"strings"
)

func main() {
	seed := flag.Uint64("seed", 0, "the case seed (packlint derives it)")
	maxN := flag.Int("max-n", 40, "the longest array")
	maxV := flag.Int("max-v", 5, "the largest magnitude")
	flag.Parse()
	r := rand.New(rand.NewPCG(*seed, 1))
	n := 2 + r.IntN(*maxN-1)
	vals := make([]string, n)
	for i := range vals {
		vals[i] = fmt.Sprint(r.IntN(2**maxV+1) - *maxV)
	}
	fmt.Printf("{\"args\":[[%s]]}\n", strings.Join(vals, ","))
}
