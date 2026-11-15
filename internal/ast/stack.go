package ast

// Delimiter indicates whether a nesting layer is an object or an array.
type Delimiter byte

const (
	DelimObject Delimiter = '{'
	DelimArray  Delimiter = '['
)

// Stack maintains nested delimiters for structural JSON balancing.
type Stack struct {
	items []Delimiter
}

// NewStack creates an empty delimiter stack with preallocated capacity.
func NewStack(capacity int) *Stack {
	if capacity <= 0 {
		capacity = 32
	}
	return &Stack{
		items: make([]Delimiter, 0, capacity),
	}
}

// Push adds an opening delimiter onto the stack.
func (s *Stack) Push(d Delimiter) {
	s.items = append(s.items, d)
}

// Pop removes and returns the top delimiter, or 0 if empty.
func (s *Stack) Pop() (Delimiter, bool) {
	n := len(s.items)
	if n == 0 {
		return 0, false
	}
	top := s.items[n-1]
	s.items = s.items[:n-1]
	return top, true
}

// Peek returns the top delimiter without removing it.
func (s *Stack) Peek() (Delimiter, bool) {
	n := len(s.items)
	if n == 0 {
		return 0, false
	}
	return s.items[n-1], true
}

// Len returns the current depth of the stack.
func (s *Stack) Len() int {
	return len(s.items)
}

// Clear resets the stack for reuse.
func (s *Stack) Clear() {
	s.items = s.items[:0]
}

// Slice returns the internal slice of delimiters.
func (s *Stack) Slice() []Delimiter {
	return s.items
}
