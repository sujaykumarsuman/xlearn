// Command ops prints one random fx-003 call sequence: a small shelf, Puts and Gets over few
// keys, always storing at least one key.
package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"strings"
)

func main() {
	seed := flag.Uint64("seed", 0, "the case seed (packlint derives it)")
	maxOps := flag.Int("max-ops", 30, "the most calls after the constructor")
	maxKey := flag.Int("max-key", 6, "the largest key")
	flag.Parse()
	r := rand.New(rand.NewPCG(*seed, 3))
	ops := []string{`"FixedShelf"`}
	args := []string{fmt.Sprintf("[%d]", 1+r.IntN(4))}
	n := 1 + r.IntN(*maxOps)
	for i := 0; i < n; i++ {
		key := r.IntN(*maxKey + 1)
		if i == 0 || r.IntN(2) == 0 {
			ops = append(ops, `"Put"`)
			args = append(args, fmt.Sprintf("[%d,%d]", key, r.IntN(100)))
		} else {
			ops = append(ops, `"Get"`)
			args = append(args, fmt.Sprintf("[%d]", key))
		}
	}
	fmt.Printf("{\"ops\":[%s],\"args\":[%s]}\n", strings.Join(ops, ","), strings.Join(args, ","))
}
