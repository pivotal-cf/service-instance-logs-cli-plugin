package logclient_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLogclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logclient Suite")
}
