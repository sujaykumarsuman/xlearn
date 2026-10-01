package main

// cloneGraph copies every node reachable from node breadth first, mapping originals to copies.
func cloneGraph(node *Node) *Node {
	if node == nil {
		return nil
	}
	copies := map[*Node]*Node{node: {Val: node.Val}}
	queue := []*Node{node}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, m := range n.Neighbors {
			c, ok := copies[m]
			if !ok {
				c = &Node{Val: m.Val}
				copies[m] = c
				queue = append(queue, m)
			}
			copies[n].Neighbors = append(copies[n].Neighbors, c)
		}
	}
	return copies[node]
}
