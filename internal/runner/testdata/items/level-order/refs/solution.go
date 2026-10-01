package main

// levelOrder walks the tree breadth first, one slice per level.
func levelOrder(root *TreeNode) [][]int {
	out := [][]int{}
	if root == nil {
		return out
	}
	level := []*TreeNode{root}
	for len(level) > 0 {
		var next []*TreeNode
		vals := make([]int, 0, len(level))
		for _, n := range level {
			vals = append(vals, n.Val)
			if n.Left != nil {
				next = append(next, n.Left)
			}
			if n.Right != nil {
				next = append(next, n.Right)
			}
		}
		out = append(out, vals)
		level = next
	}
	return out
}
