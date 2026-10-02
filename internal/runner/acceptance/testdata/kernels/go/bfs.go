package main

// Calibration kernel "bfs" (t3 §7.3): BFS from vertex 0 over n random undirected edges on n/4
// vertices; the sum of (distance + 1) over the reached vertices.

func kernel(n int) int64 {
	v := n / 4
	adj := make([][]int32, v)
	x := uint32(2024)
	for i := 0; i < n; i++ {
		x = x*1664525 + 1013904223
		a := int(x % uint32(v))
		x = x*1664525 + 1013904223
		b := int(x % uint32(v))
		adj[a] = append(adj[a], int32(b))
		adj[b] = append(adj[b], int32(a))
	}
	dist := make([]int32, v)
	for i := range dist {
		dist[i] = -1
	}
	dist[0] = 0
	queue := []int32{0}
	var s int64
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		s += int64(dist[u]) + 1
		for _, w := range adj[u] {
			if dist[w] < 0 {
				dist[w] = dist[u] + 1
				queue = append(queue, w)
			}
		}
	}
	return s % 1000000007
}
