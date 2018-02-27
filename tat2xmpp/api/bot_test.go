package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTypeChat(t *testing.T) {
	testString := "foo@conference.bar"
	assert.Exactly(t, typeGroupChat, getTypeChat(testString))
}
