package lru

import (
	"container/list"
	"sync"
)

// node type holds the actual value and it's key - this allows removal of the LRU element
// from the linked list and then to delete it from the map with the key.
type node struct {
	key   string
	value interface{}
}

// EvictionListener functions will be notified when an element from the LRU is removed via evition (not deletion or replacement).
type EvictionListener func(key string, value interface{})

// LRU is the basic implementation of an LRU cache
type LRU struct {
	rw       *sync.RWMutex
	m        map[string]*list.Element
	l        *list.List
	size     int
	listener EvictionListener
}

// Set will add/replace the given key with the specified value.
func (l *LRU) Set(key string, value interface{}) {
	l.rw.Lock()
	defer l.rw.Unlock()

	n := &node{
		key:   key,
		value: value,
	}

	elm := l.l.PushFront(n)
	l.m[key] = elm

	if len(l.m) > l.size {
		elm := l.l.Back()
		n := elm.Value.(*node)
		delete(l.m, n.key)
		l.l.Remove(elm)

		if l.listener != nil {
			l.listener(n.key, n.value)
		}
	}
}

// Get will fetch the value for the given key or return nil if it does not exist
func (l *LRU) Get(key string) interface{} {
	l.rw.Lock()
	defer l.rw.Unlock()

	if e, ok := l.m[key]; ok {
		n := e.Value.(*node)
		l.l.MoveToFront(e)
		return n.value
	}

	return nil
}

// Peek will fetch the value for the given key or return nil if it does not exist. This function will not update the LRU index.
func (l *LRU) Peek(key string) interface{} {
	l.rw.RLock()
	defer l.rw.RUnlock()

	elm, ok := l.m[key]
	if ok {
		return elm.Value.(*node).value
	}

	return nil
}

// Exists will determine if there is an entry for the given key
func (l *LRU) Exists(key string) bool {
	l.rw.RLock()
	defer l.rw.RUnlock()

	_, ok := l.m[key]

	return ok
}

// Delete will remove an entry for the given key if it exists
func (l *LRU) Delete(key string) {
	l.rw.Lock()
	defer l.rw.Unlock()

	elm, ok := l.m[key]

	if !ok {
		return
	}

	delete(l.m, key)
	l.l.Remove(elm)
}

// Size reports the number of active elements in the LRU
func (l *LRU) Size() int {
	l.rw.RLock()
	defer l.rw.RUnlock()
	return len(l.m)
}

// New will create a LRU instance with the given size of elements
func New(size int, listener EvictionListener) *LRU {
	lru := &LRU{
		rw:       &sync.RWMutex{},
		m:        map[string]*list.Element{},
		l:        list.New(),
		size:     size,
		listener: listener,
	}

	return lru
}
