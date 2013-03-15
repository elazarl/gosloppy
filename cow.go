package main

// This is a functional-like data structure of copy-on-write array
// When an element of the array is set, a new array is allocated and there this element is changed.
// For example:
//     a := cow([]int { 1, 2, 3 })
//     b := a.Set(1, 5)
//     // a = [1 2 3] ([1, 5] -> b)
//     // b = [1 5 3]

type cow struct {
	ar       []ScopeVisitor
	children map[int]map[ScopeVisitor]*cow
}

func newCow(v ...ScopeVisitor) *cow {
	return &cow{v, make(map[int]map[ScopeVisitor]*cow)}
}

func (a *cow) Set(i int, v ScopeVisitor) *cow {
	if a.ar[i] == v {
		return a
	}
	m, ok := a.children[i]
	if !ok {
		m = make(map[ScopeVisitor]*cow)
		a.children[i] = m
	}
	c, ok := m[v]
	if ok {
		return c
	}
	copyar := make([]ScopeVisitor, len(a.ar))
	copy(copyar, a.ar)
	copyar[i] = v
	child := newCow(copyar...)
	m[v] = child
	return child
}
