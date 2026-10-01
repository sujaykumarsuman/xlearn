// want: RE
package main

// reverseList dereferences a nil pointer: nil_dereference.
func reverseList(head *ListNode) *ListNode {
	var p *ListNode
	return p.Next
}
