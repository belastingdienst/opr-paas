package v1alpha2

import (
	"testing"

	"github.com/belastingdienst/opr-paas/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestConvertTo(t *testing.T) {
	// TODO
	dst := &v1alpha1.Paas{}
	src := &Paas{}

	err := src.ConvertTo(dst)

	assert.NoError(t, err)
}

func TestConvertFrom(t *testing.T) {
	// TODO
	src := &v1alpha1.Paas{}
	dst := &Paas{}

	err := dst.ConvertFrom(src)

	assert.NoError(t, err)
}
