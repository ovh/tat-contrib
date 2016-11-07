package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArrayContains(t *testing.T) {
	array1 := []string{"a", "b"}
	assert.True(t, ArrayContains(array1, "a"), "should be true, a should be find in array{a, b}")
}

func TestArrayNotContains(t *testing.T) {
	array1 := []string{"a", "b", "c"}
	assert.False(t, ArrayContains(array1, "d"), "should be false")
}
