package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractWord(t *testing.T) {
	word := "<foo>"
	assert.Exactly(t, "foo", extractWord(word))
}
