package cfutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCfutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cfutil Suite")
}
