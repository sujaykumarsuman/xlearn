package main

// MinStack keeps, beside each value, the minimum of the stack up to it.
type MinStack struct {
	vals, mins []int
}

func Constructor() MinStack { return MinStack{} }

func (s *MinStack) push(val int) {
	m := val
	if n := len(s.mins); n > 0 && s.mins[n-1] < m {
		m = s.mins[n-1]
	}
	s.vals = append(s.vals, val)
	s.mins = append(s.mins, m)
}

func (s *MinStack) pop() {
	s.vals = s.vals[:len(s.vals)-1]
	s.mins = s.mins[:len(s.mins)-1]
}

func (s *MinStack) top() int { return s.vals[len(s.vals)-1] }

func (s *MinStack) getMin() int { return s.mins[len(s.mins)-1] }
