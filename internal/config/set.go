package config

import "maps"

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](items ...T) Set[T] {
	s := make(Set[T], len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

func (s Set[T]) Has(item T) bool {
	_, ok := s[item]
	return ok
}

func (s Set[T]) With(items ...T) Set[T] {
	out := maps.Clone(s)
	if out == nil {
		out = make(Set[T], len(items))
	}
	for _, item := range items {
		out[item] = struct{}{}
	}
	return out
}

func (s Set[T]) Without(items ...T) Set[T] {
	out := maps.Clone(s)
	for _, item := range items {
		delete(out, item)
	}
	return out
}
