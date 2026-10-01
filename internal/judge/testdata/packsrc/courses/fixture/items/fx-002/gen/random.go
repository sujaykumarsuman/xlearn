// Command random prints one random fx-002 input: 3..max-n small integers and a small target.
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
	flag.Parse()
	r := rand.New(rand.NewPCG(*seed, 2))
	n := 3 + r.IntN(*maxN-2)
	vals := make([]string, n)
	for i := range vals {
		vals[i] = fmt.Sprint(r.IntN(13) - 6)
	}
	fmt.Printf("{\"args\":[[%s],%d]}\n", strings.Join(vals, ","), r.IntN(7)-3)
}
