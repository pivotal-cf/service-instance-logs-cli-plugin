package logging_test

import (
	"errors"
	"sync"
	"time"

	"code.cloudfoundry.org/cli/plugin/models"
	"code.cloudfoundry.org/cli/plugin/pluginfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logclient/logclientfakes"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logging"
)

var _ = Describe("Logs", func() {
	const (
		errMessage          = "no dice"
		serviceInstanceName = "siname"
		testToken           = "some-token"
		serviceGUID         = "870cdf18-7e15-435a-8459-6c38a8452d79"
	)

	var (
		fakeCliConnection    *pluginfakes.FakeCliConnection
		recent               bool
		fakeLogClientBuilder *logclientfakes.FakeLogClientBuilder
		fakeLogClient        *logclientfakes.FakeLogClient
		err                  error
		testError            error
		output               *gbytes.Buffer
	)

	BeforeEach(func() {
		fakeCliConnection = &pluginfakes.FakeCliConnection{}
		fakeCliConnection.GetServiceReturns(plugin_models.GetService_Model{Guid: serviceGUID}, nil)
		fakeCliConnection.AccessTokenReturns("bearer "+testToken, nil)
		fakeLogClientBuilder = &logclientfakes.FakeLogClientBuilder{}
		fakeLogClient = &logclientfakes.FakeLogClient{}
		fakeLogClientBuilder.EndpointReturns(fakeLogClientBuilder)
		fakeLogClientBuilder.BuildReturns(fakeLogClient)
		recent = true
		testError = errors.New(errMessage)
		output = gbytes.NewBuffer()

		servicesOutput := []string{
			`{`,
			`"total_results": 1,`,
			`"resources": [`,
			`{`,
			`"entity": {`,
			`"extra": "{\"documentationUrl\":\"http://docs.pivotal.io/spring-cloud-services/\",\"serviceInstanceLogsEndpoint\":\"https://service-instance-logs/logs/\"}"`,
			`}`,
			`}`,
			`]`,
			`}`}
		fakeCliConnection.CliCommandWithoutTerminalOutputReturns(servicesOutput, nil)
	})

	JustBeforeEach(func() {
		err = logging.Logs(fakeCliConnection, output, serviceInstanceName, recent, fakeLogClientBuilder)
	})

	Context("when obtaining the service instance GUID returns an error", func() {
		BeforeEach(func() {
			fakeCliConnection.GetServiceReturns(plugin_models.GetService_Model{}, testError)
		})

		It("should propagate the error", func() {
			Expect(err).To(Equal(testError))
		})
	})

	Context("when obtaining an access token returns an error", func() {
		BeforeEach(func() {
			fakeCliConnection.AccessTokenReturns("", testError)
		})

		It("should propagate the error", func() {
			Expect(err).To(MatchError("Access token not available: " + errMessage))
		})
	})

	Context("when obtaining logs endpoint returns an error", func() {
		BeforeEach(func() {
			fakeCliConnection.CliCommandWithoutTerminalOutputReturns([]string{}, testError)
		})

		It("should propagate the error", func() {
			Expect(err).To(MatchError("/v2/services failed: " + errMessage))
		})
	})

	Context("when logs endpoint is not found", func() {
		BeforeEach(func() {
			output := []string{`{`, `"total_results": 0,`, `"resources": []`, `}`}
			fakeCliConnection.CliCommandWithoutTerminalOutputReturns(output, nil)
		})

		It("should return a suitable error", func() {
			Expect(err).To(MatchError("/v2/services did not return the service instance"))
		})
	})

	Context("when services output is malformed JSON", func() {
		BeforeEach(func() {
			output := []string{`{`}
			fakeCliConnection.CliCommandWithoutTerminalOutputReturns(output, nil)
		})

		It("should return a suitable error", func() {
			Expect(err).To(MatchError("/v2/services returned invalid JSON: unexpected end of JSON input"))
		})
	})

	Context("when the extras field contains malformed JSON", func() {
		BeforeEach(func() {
			servicesOutput := []string{
				`{`,
				`"total_results": 1,`,
				`"resources": [`,
				`{`,
				`"entity": {`,
				`"extra": "{"`,
				`}`,
				`}`,
				`]`,
				`}`}
			fakeCliConnection.CliCommandWithoutTerminalOutputReturns(servicesOutput, nil)
		})

		It("should return a suitable error", func() {
			Expect(err).To(MatchError("/v2/services 'extra' field contained invalid JSON: unexpected end of JSON input"))
		})
	})

	Context("when the extras field does not contain the logs endpoint", func() {
		BeforeEach(func() {
			servicesOutput := []string{
				`{`,
				`"total_results": 1,`,
				`"resources": [`,
				`{`,
				`"entity": {`,
				`"extra": "{\"documentationUrl\":\"http://docs.pivotal.io/spring-cloud-services/\"}"`,
				`}`,
				`}`,
				`]`,
				`}`}
			fakeCliConnection.CliCommandWithoutTerminalOutputReturns(servicesOutput, nil)
		})

		It("should return a suitable error", func() {
			Expect(err).To(MatchError("/v2/services did not contain a service instance logs endpoint: maybe the broker version is too old"))
		})
	})

	Context("when logs endpoint is found", func() {
		It("should pass the endpoint to the log client builder", func() {
			Expect(fakeLogClientBuilder.EndpointArgsForCall(0)).To(Equal("https://service-instance-logs/logs/"))
		})
	})

	Context("when dumping recent logs", func() {
		It("should call log client recent logs with the correct parameters", func() {
			Expect(fakeLogClient.RecentLogsCallCount()).To(Equal(1))
			guid, tok := fakeLogClient.RecentLogsArgsForCall(0)
			Expect(guid).To(Equal(serviceGUID))
			Expect(tok).To(Equal(testToken))
		})

		Context("when log client recent logs return an error", func() {
			BeforeEach(func() {
				fakeLogClient.RecentLogsReturns([]string{}, testError)
			})

			It("should propagate the error", func() {
				Expect(err).To(Equal(testError))
			})
		})

		Context("when log client recent logs returns normally", func() {
			BeforeEach(func() {
				fakeLogClient.RecentLogsReturns([]string{"hello", "goodbye"}, nil)
			})

			It("should return normally", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should print the logs", func() {
				Expect(output).To(gbytes.Say("hello"))
				Expect(output).To(gbytes.Say("goodbye"))
			})
		})
	})

	Context("when tailing logs", func() {
		var (
			messageChan chan string
			errChan     chan error
		)

		BeforeEach(func() {
			recent = false
			messageChan = make(chan string)
			errChan = make(chan error, 1)
			fakeLogClient.TailingLogsReturns(messageChan, errChan)
		})

		Context("in the normal case", func() {
			BeforeEach(func() {
				errChan <- testError // any kind of termination will do for this context
			})

			AfterEach(func() {
				close(messageChan)
				close(errChan)
			})

			It("should correctly transform the endpoint passed to the LogClientBuilder", func() {
				Expect(fakeLogClientBuilder.EndpointCallCount()).To(Equal(1))
				Expect(fakeLogClientBuilder.EndpointArgsForCall(0)).To(Equal("wss://service-instance-logs"))
			})

			Context("when the logs endpoint is insecure", func() {
				BeforeEach(func() {
					servicesOutput := []string{
						`{`,
						`"total_results": 1,`,
						`"resources": [`,
						`{`,
						`"entity": {`,
						`"extra": "{\"documentationUrl\":\"http://docs.pivotal.io/spring-cloud-services/\",\"serviceInstanceLogsEndpoint\":\"http://service-instance-logs/logs/\"}"`,
						`}`,
						`}`,
						`]`,
						`}`}
					fakeCliConnection.CliCommandWithoutTerminalOutputReturns(servicesOutput, nil)
				})

				It("should correctly transform the endpoint passed to the LogClientBuilder", func() {
					Expect(fakeLogClientBuilder.EndpointCallCount()).To(Equal(1))
					Expect(fakeLogClientBuilder.EndpointArgsForCall(0)).To(Equal("ws://service-instance-logs"))
				})
			})

			Context("when the logs endpoint is malformed", func() {
				BeforeEach(func() {
					servicesOutput := []string{
						`{`,
						`"total_results": 1,`,
						`"resources": [`,
						`{`,
						`"entity": {`,
						`"extra": "{\"documentationUrl\":\"http://docs.pivotal.io/spring-cloud-services/\",\"serviceInstanceLogsEndpoint\":\"::\"}"`,
						`}`,
						`}`,
						`]`,
						`}`}
					fakeCliConnection.CliCommandWithoutTerminalOutputReturns(servicesOutput, nil)
				})

				It("should return a suitable error", func() {
					Expect(err.Error()).To(ContainSubstring("missing protocol scheme"))
				})
			})

			It("should call the log client tailing logs method", func() {
				Expect(fakeLogClient.TailingLogsCallCount()).To(Equal(1))
			})

			It("should pass the access token to the log client", func() {
				_, tok := fakeLogClient.TailingLogsArgsForCall(0)
				Expect(tok).To(Equal(testToken))
			})
		})

		Context("when an error is sent to the error channel", func() {
			BeforeEach(func() {
				errChan <- testError
			})

			AfterEach(func() {
				close(messageChan)
				close(errChan)
			})

			It("should return the error", func() {
				Expect(err).To(Equal(testError))
			})
		})

		Context("when the message and error channels are closed", func() {
			var wg sync.WaitGroup

			BeforeEach(func() {
				wg = sync.WaitGroup{}
				wg.Add(1)

				go func() {
					defer wg.Done()
					time.Sleep(50 * time.Millisecond)
					close(messageChan)
					close(errChan)
				}()
			})

			AfterEach(func() {
				wg.Wait()
			})

			It("should return normally", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when a message is sent before the message and error channels are closed", func() {
			var wg sync.WaitGroup

			BeforeEach(func() {
				wg = sync.WaitGroup{}
				wg.Add(1)

				// Note: in theory this test is race-prone, but it's better than nothing.
				go func() {
					defer wg.Done()

					messageChan <- "hello"

					time.Sleep(50 * time.Millisecond)

					close(messageChan)
					close(errChan)
				}()
			})

			AfterEach(func() {
				wg.Wait()
			})

			It("should print the message", func() {
				Expect(output).To(gbytes.Say("hello"))
			})

			It("should return normally", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
