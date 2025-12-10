package util

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSet(t *testing.T) {
	s := NewSet[string]("a", "b", "c")
	assert.True(t, s.IsMember("a"))
	assert.True(t, s.IsMember("b"))
	assert.True(t, s.IsMember("c"))
	assert.False(t, s.IsMember("d"))
	assert.Equal(t, 3, len(*s))

	sInt := NewSet[int](1, 2, 3)
	assert.True(t, sInt.IsMember(1))
	assert.False(t, sInt.IsMember(4))
	assert.Equal(t, 3, len(*sInt))
}

func TestAdd(t *testing.T) {
	s := NewSet[string]()
	assert.False(t, s.IsMember("apple"))

	s.Add("apple")
	assert.True(t, s.IsMember("apple"))
	assert.Equal(t, 1, len(*s))

	s.Add("banana", "cherry")
	assert.True(t, s.IsMember("banana"))
	assert.True(t, s.IsMember("cherry"))
	assert.Equal(t, 3, len(*s))

	// Adding existing item should not change size
	s.Add("apple")
	assert.Equal(t, 3, len(*s))
}

func TestIsMember(t *testing.T) {
	s := NewSet[string]("apple", "banana")

	assert.True(t, s.IsMember("apple"))
	assert.True(t, s.IsMember("banana"))
	assert.False(t, s.IsMember("cherry"))
	assert.False(t, s.IsMember(""))
}

func TestToList(t *testing.T) {
	s := NewSet[string]("c", "a", "b")
	list := s.ToList()

	// Sort for consistent comparison
	sort.Strings(list)
	expected := []string{"a", "b", "c"}

	assert.True(t, reflect.DeepEqual(expected, list))

	sEmpty := NewSet[string]()
	listEmpty := sEmpty.ToList()
	assert.Empty(t, listEmpty)
	assert.Equal(t, 0, len(listEmpty))
}

func TestRemove(t *testing.T) {
	s := NewSet[string]("apple", "banana", "cherry")
	assert.True(t, s.IsMember("banana"))
	assert.Equal(t, 3, len(*s))

	s.Remove("banana")
	assert.False(t, s.IsMember("banana"))
	assert.Equal(t, 2, len(*s))

	// Removing non-existent item should not cause error or change size
	s.Remove("grape")
	assert.Equal(t, 2, len(*s))
}
