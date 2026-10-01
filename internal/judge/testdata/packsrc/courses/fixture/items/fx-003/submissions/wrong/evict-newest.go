package main

// FixedShelf (wrong: WA) evicts the key stored most recently instead of the earliest.
type FixedShelf struct {
	capacity int
	values   map[int]int
	order    []int
}

func Constructor(capacity int) FixedShelf {
	return FixedShelf{capacity: capacity, values: make(map[int]int)}
}

func (s *FixedShelf) Put(key int, value int) {
	if _, ok := s.values[key]; ok {
		s.values[key] = value
		return
	}
	if len(s.order) == s.capacity {
		last := len(s.order) - 1
		delete(s.values, s.order[last])
		s.order = s.order[:last]
	}
	s.order = append(s.order, key)
	s.values[key] = value
}

func (s *FixedShelf) Get(key int) int {
	if v, ok := s.values[key]; ok {
		return v
	}
	return -1
}
