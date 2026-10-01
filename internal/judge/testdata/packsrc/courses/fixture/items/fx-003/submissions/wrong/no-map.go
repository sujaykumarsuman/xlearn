package main

// FixedShelf (wrong: RE) never makes its map, so the first Put writes to a nil map.
type FixedShelf struct {
	capacity int
	values   map[int]int
	order    []int
}

func Constructor(capacity int) FixedShelf {
	return FixedShelf{capacity: capacity}
}

func (s *FixedShelf) Put(key int, value int) {
	if _, ok := s.values[key]; ok {
		s.values[key] = value
		return
	}
	if len(s.order) == s.capacity {
		delete(s.values, s.order[0])
		s.order = s.order[1:]
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
