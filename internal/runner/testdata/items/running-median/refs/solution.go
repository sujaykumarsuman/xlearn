package main

import "container/heap"

type maxHeap []int

func (h maxHeap) Len() int           { return len(h) }
func (h maxHeap) Less(i, j int) bool { return h[i] > h[j] }
func (h maxHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *maxHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *maxHeap) Pop() any          { o := *h; x := o[len(o)-1]; *h = o[:len(o)-1]; return x }

type minHeap []int

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *minHeap) Pop() any          { o := *h; x := o[len(o)-1]; *h = o[:len(o)-1]; return x }

// runningMedian keeps the lower half in a max-heap and the upper half in a min-heap.
func runningMedian(nums []int) []float64 {
	lo, hi := &maxHeap{}, &minHeap{}
	out := make([]float64, 0, len(nums))
	for _, x := range nums {
		if lo.Len() == 0 || x <= (*lo)[0] {
			heap.Push(lo, x)
		} else {
			heap.Push(hi, x)
		}
		if lo.Len() > hi.Len()+1 {
			heap.Push(hi, heap.Pop(lo))
		} else if hi.Len() > lo.Len() {
			heap.Push(lo, heap.Pop(hi))
		}
		if lo.Len() > hi.Len() {
			out = append(out, float64((*lo)[0]))
		} else {
			out = append(out, float64((*lo)[0]+(*hi)[0])/2)
		}
	}
	return out
}
