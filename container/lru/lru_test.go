package lru

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvictionListener(t *testing.T) {

	count := 0
	var k string
	var v interface{}
	el := func(key string, value interface{}) {
		count++
		k = key
		v = value
	}
	cache := New(5, el)

	cache.Set("a", 0)
	cache.Set("b", 1)
	cache.Set("c", 2)
	cache.Set("d", 3)
	cache.Set("e", 4)
	cache.Set("f", 5)

	require.Equal(t, 1, count, "Expected the evition listener to be called once")
	require.Equal(t, 0, v, "Expected value was not correct")
	require.Equal(t, "a", k, "Expected key was not correct")
}

func TestPeek(t *testing.T) {

	count := 0
	var k string
	var v interface{}
	el := func(key string, value interface{}) {
		count++
		k = key
		v = value
	}
	cache := New(5, el)

	cache.Set("a", 0)
	cache.Set("b", 1)
	cache.Set("c", 2)
	cache.Set("d", 3)
	cache.Set("e", 4)

	elm := cache.l.Back()

	n := elm.Value.(*node)
	require.Equal(t, 0, n.value, "Expected value was not correct")
	require.Equal(t, "a", n.key, "Expected key was not correct")

	cache.Peek("e")
	elm = cache.l.Back()

	n = elm.Value.(*node)
	require.Equal(t, 0, n.value, "Expected value was not correct")
	require.Equal(t, "a", n.key, "Expected key was not correct")

}

func TestReplace(t *testing.T) {

	count := 0
	var k string
	var v interface{}
	el := func(key string, value interface{}) {
		count++
		k = key
		v = value
	}
	cache := New(5, el)

	cache.Set("a", 0)
	cache.Set("b", 1)
	cache.Set("c", 2)
	cache.Set("d", 3)
	cache.Set("e", 4)

	elm := cache.l.Front()
	n := elm.Value.(*node)
	require.Equal(t, 4, n.value, "Expected value was not correct")
	require.Equal(t, "e", n.key, "Expected key was not correct")

	cache.Set("b", 10)

	elm = cache.l.Front()
	n = elm.Value.(*node)
	require.Equal(t, 10, n.value, "Expected value was not correct")
	require.Equal(t, "b", n.key, "Expected key was not correct")

}

func TestDelete(t *testing.T) {

	count := 0
	var k string
	var v interface{}
	el := func(key string, value interface{}) {
		count++
		k = key
		v = value
	}
	cache := New(5, el)

	cache.Set("a", 0)
	cache.Set("b", 1)
	cache.Set("c", 2)
	cache.Set("d", 3)
	cache.Set("e", 4)

	elm := cache.l.Front()
	n := elm.Value.(*node)
	require.Equal(t, 4, n.value, "Expected value was not correct")
	require.Equal(t, "e", n.key, "Expected key was not correct")

	cache.Delete("e")

	elm = cache.l.Front()
	n = elm.Value.(*node)
	require.Equal(t, 3, n.value, "Expected value was not correct")
	require.Equal(t, "d", n.key, "Expected key was not correct")

	require.Equal(t, 4, cache.Size())

}
