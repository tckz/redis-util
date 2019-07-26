package redisutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanup0(t *testing.T) {
	cleanups := &Cleanups{}

	called := []string{}
	cleanups.Do()

	assert.Equal(t, []string{}, called)
}

func TestCleanup1(t *testing.T) {
	cleanups := &Cleanups{}

	called := []string{}
	cleanups.Add(func() {
		called = append(called, "call1")
	})

	cleanups.Do()

	assert.Equal(t, []string{"call1"}, called)
}

func TestCleanup3(t *testing.T) {
	cleanups := &Cleanups{}

	called := []string{}

	cleanups.Add(func() {
		called = append(called, "call1")
	})

	cleanups.Add(func() {
		called = append(called, "call2")
	})

	cleanups.Add(func() {
		called = append(called, "call3")
	})

	cleanups.Do()

	assert.Equal(t, []string{
		"call3",
		"call2",
		"call1",
	}, called)
}
