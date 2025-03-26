package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringIsRegex_valid(t *testing.T) {
	is, err := StringIsRegex("^foo$")

	assert.True(t, is)
	assert.Nil(t, err)
}

func TestStringIsRegex_invalid(t *testing.T) {
	is, err := StringIsRegex("((invalid regex")

	assert.False(t, is)
	assert.Error(t, err)
}
