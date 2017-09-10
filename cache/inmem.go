package cache

import (
	"github.com/devork/grava/container/lru"
	log "github.com/sirupsen/logrus"
)

type memcache struct {
	cache *lru.LRU
}

func (m *memcache) Set(key string, tile []byte) error {
	m.cache.Set(key, tile)

	log.Debugf("Added tile: key = %s, size = %d", key, len(tile))
	return nil
}

func (m *memcache) Get(key string) ([]byte, error) {
	value := m.cache.Get(key)

	if value == nil {
		return nil, nil
	}

	return value.([]byte), nil
}

func (m *memcache) Exists(key string) bool {
	return m.cache.Exists(key)
}

func (m *memcache) Delete(key string) {
	m.cache.Delete(key)
}

// NewMemoryCacher will create an in-memory cache of tiles.
func NewMemoryCacher(size int) Cacher {
	listener := func(key string, value interface{}) {
		log.WithFields(log.Fields{
			"tile": key,
		})
		log.Debugf("Evicted tile: key = %s, size = %d", key, len(value.([]byte)))
	}
	return &memcache{
		cache: lru.New(size, listener),
	}
}
