package cache

// Cacher manages a backing cache of tiles
type Cacher interface {
	// Set will push the given tile data into the cache optionally returning
	// an error if the cache is unable to store the value
	Set(key string, tile []byte) error

	// Get will return the tile for the given key - it will return a nil value for the data
	// if no such key exists. An error is raised if some underlying problem accessing the cache occurs.
	// This function will not return an error for a non-existent key
	Get(key string) ([]byte, error)

	// Exists checks if there is anything mapped to the given key in the cache
	Exists(key string) bool

	// Delete removes (if present) the value associated with the given key
	Delete(key string)
}

type noop struct{}

func (m *noop) Set(key string, tile []byte) error {
	return nil
}

func (m *noop) Get(key string) ([]byte, error) {
	return nil, nil
}

func (m *noop) Exists(key string) bool {
	return false
}

func (m *noop) Delete(key string) {
}

// NewNOOP returns a /dev/null cache which does nothing
func NewNOOP() Cacher {
	return &noop{}
}
