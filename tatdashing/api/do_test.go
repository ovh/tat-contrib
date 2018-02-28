package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortedKeysLen(t *testing.T) {
	testKeys := sortedKeys{"a", "b", "c", "d"}
	assert.Exactly(t, 4, testKeys.Len())
}
