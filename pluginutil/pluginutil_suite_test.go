package pluginutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPluginutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pluginutil Suite")
}
