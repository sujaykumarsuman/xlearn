package main

import "sort"

// groupWords keys each word by its sorted bytes; groups keep first-appearance order.
func groupWords(words []string) [][]string {
	index := map[string]int{}
	out := [][]string{}
	for _, w := range words {
		b := []byte(w)
		sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })
		k := string(b)
		i, ok := index[k]
		if !ok {
			i = len(out)
			index[k] = i
			out = append(out, nil)
		}
		out[i] = append(out[i], w)
	}
	return out
}
