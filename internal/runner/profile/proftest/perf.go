package proftest

import (
	"encoding/json"
	"math/rand/v2"
)

// PerfCases are the generated perf inputs of the seven synthetic items under
// internal/runner/testdata/items (deterministic seeds), as raw func-json@1 / class-ops@1 input
// JSON. The runner-it lane (refs_it_test.go, m3-04) and the acceptance suite's section G (m3-15)
// share them, so the wrong solutions classified TLE meet the same perf block in both.
func PerfCases(slug string) [][]byte {
	r := rand.New(rand.NewPCG(4, uint64(len(slug))))
	js := func(v any) []byte { b, _ := json.Marshal(v); return b }
	args := func(a ...any) []byte { return js(map[string]any{"args": a}) }
	switch slug {
	case "pair-sum":
		var out [][]byte
		for _, n := range []int{100000, 200000} {
			nums := make([]int, n)
			for i := range nums {
				nums[i] = 2*i + 1 // odd values: the only pair summing to target is the last two
			}
			out = append(out, args(nums, nums[n-2]+nums[n-1]))
		}
		return out
	case "group-words":
		words := make([]string, 50000)
		for i := range words {
			b := make([]byte, 8)
			for k := range b {
				b[k] = byte('a' + r.IntN(4))
			}
			words[i] = string(b)
		}
		return [][]byte{args(words)}
	case "reverse-list":
		vals := make([]int, 300000)
		for i := range vals {
			vals[i] = r.IntN(2000001) - 1000000
		}
		return [][]byte{args(vals)}
	case "level-order":
		vals := make([]int, 200000)
		for i := range vals {
			vals[i] = i + 1
		}
		return [][]byte{args(vals)}
	case "clone-graph":
		n := 50000
		adj := make([][]int, n)
		for i := range adj {
			adj[i] = []int{(i+n-1)%n + 1, (i+1)%n + 1, (i+n/2)%n + 1}
		}
		return [][]byte{args(adj)}
	case "running-median":
		nums := make([]int, 100000)
		for i := range nums {
			nums[i] = r.IntN(2000000001) - 1000000000
		}
		return [][]byte{args(nums)}
	case "min-stack":
		ops, as := []string{"MinStack"}, []any{[]int{}}
		depth := 0
		for i := 0; i < 200000; i++ {
			switch k := r.IntN(10); {
			case depth == 0 || k < 4:
				ops, as = append(ops, "push"), append(as, []int{r.IntN(2000001) - 1000000})
				depth++
			case k < 6:
				ops, as = append(ops, "pop"), append(as, []int{})
				depth--
			case k < 8:
				ops, as = append(ops, "top"), append(as, []int{})
			default:
				ops, as = append(ops, "getMin"), append(as, []int{})
			}
		}
		return [][]byte{js(map[string]any{"ops": ops, "args": as})}
	}
	return nil
}
