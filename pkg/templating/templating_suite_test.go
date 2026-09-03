package templating

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTemplating(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Templating Suite")
}
