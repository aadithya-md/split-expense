package util

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](items ...T) *Set[T] {
	s := make(Set[T])
	s.Add(items...)
	return &s
}

func (s *Set[T]) Add(items ...T) {
	for _, item := range items {
		(*s)[item] = struct{}{}
	}
}

func (s *Set[T]) IsMember(item T) bool {
	_, ok := (*s)[item]
	return ok
}

func (s *Set[T]) ToList() []T {
	list := make([]T, 0, len(*s))
	for item := range *s {
		list = append(list, item)
	}
	return list
}

func (s *Set[T]) Remove(item T) {
	delete(*s, item)
}
