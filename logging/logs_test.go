package logging_test

import (
	"errors"
	"sync"
	"time"

	"code.cloudfoundry.org/cli/plugin/pluginfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logclient/logclientfakes"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logging"
	"code.cloudfoundry.org/cli/plugin/models"
)

var _ = Describe("Logs", func() {
	const (
		errMessage          = "no dice"
		serviceInstanceName = "siname"
		testToken           = "some-token"
		serviceGUID = "870cdf18-7e15-435a-8459-6c38a8452d79"
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
		fakeCliConnection.GetServiceReturns(plugin_models.GetService_Model{Guid:serviceGUID}, nil)
		fakeCliConnection.AccessTokenReturns("bearer "+testToken, nil)
		fakeLogClientBuilder = &logclientfakes.FakeLogClientBuilder{}
		fakeLogClient = &logclientfakes.FakeLogClient{}
		fakeLogClientBuilder.EndpointReturns(fakeLogClientBuilder)
		fakeLogClientBuilder.BuildReturns(fakeLogClient)
		recent = true
		testError = errors.New(errMessage)
		output = gbytes.NewBuffer()
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
					time.Sleep(time.Second)
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

					time.Sleep(time.Second)

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
