package logclient_test

import (
	"net/url"

	"os"

	"github.com/cloudfoundry/noaa/consumer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logclient"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logclient/logclientfakes"
)

var _ = Describe("LogclientBuilder", func() {
	var (
		fakeConsumer *logclientfakes.FakeConsumer
		builder      logclient.LogClientBuilder
	)

	JustBeforeEach(func() {
		fakeConsumer = &logclientfakes.FakeConsumer{}
		builder = logclient.NewLogClientBuilder()
		if b, ok := builder.(logclient.BuildWithConsumer); ok {
			b.BuildFromConsumer(fakeConsumer)
		} else {
			Fail("LogClientBuilder did not implement BuildWithConsumer")
		}
	})

	Context("DEBUG environment variable tests", func() {
		var (
			debugWasSet   bool
			oldDebugValue string
		)

		BeforeEach(func() {
			oldDebugValue, debugWasSet = os.LookupEnv("DEBUG")
		})

		AfterEach(func() {
			if debugWasSet {
				os.Setenv("DEBUG", oldDebugValue)
			}
		})

		Context("when $DEBUG is not set", func() {
			BeforeEach(func() {
				os.Unsetenv("DEBUG")
			})

			It("should not set debug printing", func() {
				Expect(fakeConsumer.SetDebugPrinterCallCount()).To(Equal(0))
			})
		})

		Context("when $DEBUG is  set", func() {
			BeforeEach(func() {
				os.Setenv("DEBUG", "true")
			})

			AfterEach(func() {
				os.Unsetenv("DEBUG")
			})

			It("should set debug printing", func() {
				Expect(fakeConsumer.SetDebugPrinterCallCount()).To(Equal(1))
			})
		})
	})

	Describe("recent path builder", func() {
		var recentPathBuilder consumer.RecentPathBuilder

		JustBeforeEach(func() {
			recentPathBuilder = fakeConsumer.SetRecentPathBuilderArgsForCall(0)
		})

		It("should be set", func() {
			Expect(fakeConsumer.SetRecentPathBuilderCallCount()).To(Equal(1))
		})

		It("should correctly compute the recent path for a ws traffic controller scheme", func() {
			url, err := url.Parse("ws://some.host/a/path")
			Expect(err).NotTo(HaveOccurred())
			Expect(recentPathBuilder(url, "appguid", "endpoint")).To(Equal("http://some.host/logs/appguid/endpoint"))
		})

		It("should correctly compute the recent path for a wss traffic controller scheme", func() {
			url, err := url.Parse("wss://some.host/a/path")
			Expect(err).NotTo(HaveOccurred())
			Expect(recentPathBuilder(url, "appguid", "endpoint")).To(Equal("https://some.host/logs/appguid/endpoint"))
		})
	})

	Describe("stream path builder", func() {
		var streamPathBuilder consumer.StreamPathBuilder

		JustBeforeEach(func() {
			streamPathBuilder = fakeConsumer.SetStreamPathBuilderArgsForCall(0)
		})

		It("should be set", func() {
			Expect(fakeConsumer.SetStreamPathBuilderCallCount()).To(Equal(1))
		})

		It("should correctly compute the stream path", func() {
			Expect(streamPathBuilder("appguid")).To(Equal("/logs/appguid/stream"))
		})
	})
})
