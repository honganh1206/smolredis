package store

import "sync"

type InMemoryStore struct {
	Data sync.Map
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{}
}
