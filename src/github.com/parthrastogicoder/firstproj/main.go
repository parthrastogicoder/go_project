// # make a go program that implement concurrent set 
package main

import (
	"fmt"
	"sync"
)

type Set struct {
	m map[int]bool
	sync.RWMutex
}

func New() *Set {
	return &Set{
		m: map[int]bool{},
	}
}

func (s *Set) Add(item int) {
	s.Lock()
	defer s.Unlock()
	s.m[item] = true
}

func (s *Set) Remove(item int) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, item)
}

func (s *Set) Has(item int) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.m[item]
	return ok
}

func (s *Set) Len() int {
	return len(s.List())
}

func (s *Set) Clear() {
	s.Lock()
	defer s.Unlock()
	s.m = map[int]bool{}
}

func (s *Set) IsEmpty() bool {
	if s.Len() == 0 {
		return true
	}
	return false
}

func (s *Set) List() []int {
	s.RLock()
	defer s.RUnlock()
	list := []int{}
	for item := range s.m {
		list = append(list, item)
	}
	return list
}

func main() {
	s := New()
	s.Add(1)
	s.Add(2)
	s.Add(3)
	fmt.Println(s.Has(1)) // true
	fmt.Println(s.Has(4)) // false
	fmt.Println(s.List()) // [1, 2, 3]
	s.Remove(2)
	fmt.Println(s.List()) // [1, 3]
	s.Clear()
	fmt.Println(s.IsEmpty()) // true
}
// Output
// true
// false
// [1 2 3]
// [1 3]
// true
