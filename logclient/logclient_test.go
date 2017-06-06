package logclient_test

import (
	"errors"
	"fmt"

	"time"

	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logclient"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logclient/logclientfakes"
)

var _ = Describe("Logclient", func() {
	const (
		errMessage  = "that's torn it"
		endpointUrl = "endpointUrl"
		serviceGuid = "serviceGuid"
		authToken   = "authToken"
	)

	var (
		logClient        logclient.LogClient
		fakeConsumer     *logclientfakes.FakeConsumer
		testError        error
		currentTimestamp int64
	)

	BeforeEach(func() {
		testError = errors.New(errMessage)
		fakeConsumer = &logclientfakes.FakeConsumer{}

		builder := logclient.NewLogClientBuilder()
		logClient = builder.InsecureSkipVerify(true).Endpoint(endpointUrl).Build()

		if logClient, ok := logClient.(logclient.ConsumerSetter); ok {
			logClient.SetConsumer(fakeConsumer)
		} else {
			Fail("logClient did not implement ConsumerSetter")
		}
	})

	Describe("RecentLogs", func() {
		var (
			result []string
			err    error
		)

		JustBeforeEach(func() {
			result, err = logClient.RecentLogs(serviceGuid, authToken)
		})

		Context("when request for recent logs from consumer returns an error", func() {
			BeforeEach(func() {
				fakeConsumer.RecentLogsReturns([]*events.LogMessage{}, testError)
			})

			It("should call the consumer RecentLogs function", func() {
				Expect(fakeConsumer.RecentLogsCallCount()).To(Equal(1))
			})

			It("should use the supplied serviceGUID and authToken for the consumer call", func() {
				svcGuid, token := fakeConsumer.RecentLogsArgsForCall(0)
				Expect(svcGuid).To(Equal(serviceGuid))
				Expect(token).To(Equal("bearer " + authToken))
			})

			It("should propagate the error", func() {
				Expect(err).To(MatchError(errMessage))
			})
		})

		Context("when request for recent logs from consumer returns normally", func() {
			BeforeEach(func() {
				currentTimestamp = time.Now().UnixNano()

				lm1 := createLogMessage(1, events.LogMessage_OUT, currentTimestamp)
				lm2 := createLogMessage(2, events.LogMessage_OUT, currentTimestamp)
				lm3 := createLogMessage(3, events.LogMessage_ERR, currentTimestamp)

				fakeConsumer.RecentLogsReturns([]*events.LogMessage{&lm1, &lm2, &lm3}, nil)
			})

			It("should call the consumer RecentLogs function", func() {
				Expect(fakeConsumer.RecentLogsCallCount()).To(Equal(1))
			})

			It("should use the supplied serviceGUID and authToken for the consumer call", func() {
				svcGuid, token := fakeConsumer.RecentLogsArgsForCall(0)
				Expect(svcGuid).To(Equal(serviceGuid))
				Expect(token).To(Equal("bearer " + authToken))
			})

			It("should return normally", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return correctly formatted messages based on response from consumer", func() {
				Expect(len(result)).To(Equal(3))

				Expect(result[0]).Should(Equal(fmt.Sprintf("%s [ST1/SI1] OUT MESSAGE1",
					formatUnixTimestamp(currentTimestamp))))
				Expect(result[1]).Should(Equal(fmt.Sprintf("%s [ST2/SI2] OUT MESSAGE2",
					formatUnixTimestamp(currentTimestamp))))
				Expect(result[2]).Should(Equal(fmt.Sprintf("%s [ST3/SI3] ERR MESSAGE3",
					formatUnixTimestamp(currentTimestamp))))
			})
		})
	})

	Describe("TailingLogs", func() {
		var (
			logMsgsChan    chan *events.LogMessage
			logErrChan     chan error
			logStringsChan <-chan string
			errChan        <-chan error
		)

		BeforeEach(func() {
			logMsgsChan = make(chan *events.LogMessage, 3)
			logErrChan = make(chan error, 3)
			fakeConsumer.TailingLogsReturns(logMsgsChan, logErrChan)
		})

		AfterEach(func() {
			close(logMsgsChan)
			close(logErrChan)
		})

		JustBeforeEach(func() {
			logStringsChan, errChan = logClient.TailingLogs(serviceGuid, authToken)
		})

		Context("in the normal case", func() {
			BeforeEach(func() {
				currentTimestamp = time.Now().UnixNano()
				lm1 := createLogMessage(1, events.LogMessage_OUT, currentTimestamp)
				logMsgsChan <- &lm1

				lm2 := createLogMessage(2, events.LogMessage_ERR, currentTimestamp)
				logMsgsChan <- &lm2

				lm3 := createLogMessage(3, events.LogMessage_OUT, currentTimestamp)
				logMsgsChan <- &lm3
			})

			It("should call the consumer TailingLogs function", func() {
				Expect(fakeConsumer.TailingLogsCallCount()).To(Equal(1))
			})

			It("should use the supplied serviceGUID and authToken for the consumer call", func() {
				svcGuid, token := fakeConsumer.TailingLogsArgsForCall(0)
				Expect(svcGuid).To(Equal(serviceGuid))
				Expect(token).To(Equal("bearer " + authToken))
			})

			It("should send expected string messages in correct sequence to returned message channel", func() {
				var receivedMsg string
				Eventually(logStringsChan).Should(Receive(&receivedMsg))
				Expect(receivedMsg).Should(Equal(fmt.Sprintf("%s [ST1/SI1] OUT MESSAGE1",
					formatUnixTimestamp(currentTimestamp))))

				Eventually(logStringsChan).Should(Receive(&receivedMsg))
				Expect(receivedMsg).Should(Equal(fmt.Sprintf("%s [ST2/SI2] ERR MESSAGE2",
					formatUnixTimestamp(currentTimestamp))))

				Eventually(logStringsChan).Should(Receive(&receivedMsg))
				Expect(receivedMsg).Should(Equal(fmt.Sprintf("%s [ST3/SI3] OUT MESSAGE3",
					formatUnixTimestamp(currentTimestamp))))
			})

			It("should have no messages in the returned error channel", func() {
				Expect(errChan).To(BeEmpty())
			})
		})

		Context("when there are error messages in the error channel", func() {
			BeforeEach(func() {
				err1 := errors.New("Error 1")
				err2 := errors.New("Error 2")
				err3 := errors.New("Error 3")
				logErrChan <- err1
				logErrChan <- err2
				logErrChan <- err3
			})

			It("should call the consumer TailingLogs function", func() {
				Expect(fakeConsumer.TailingLogsCallCount()).To(Equal(1))
			})

			It("should use the supplied serviceGUID and authToken for the consumer call", func() {
				svcGuid, token := fakeConsumer.TailingLogsArgsForCall(0)
				Expect(svcGuid).To(Equal(serviceGuid))
				Expect(token).To(Equal("bearer " + authToken))
			})

			It("should have no messages in the returned messages channel", func() {
				Expect(logStringsChan).To(BeEmpty())
			})

			It("should send expected error messages in correct sequence to returned error channel", func() {
				var receivedErrorMessage error
				Eventually(errChan).Should(Receive(&receivedErrorMessage))
				Expect(receivedErrorMessage.Error()).Should(Equal("Error 1"))

				Eventually(errChan).Should(Receive(&receivedErrorMessage))
				Expect(receivedErrorMessage.Error()).Should(Equal("Error 2"))

				Eventually(errChan).Should(Receive(&receivedErrorMessage))
				Expect(receivedErrorMessage.Error()).Should(Equal("Error 3"))
			})
		})
	})
})

func createLogMessage(fieldSuffix int, messageType events.LogMessage_MessageType, unixTimestamp int64) events.LogMessage {
	sourceTypeValue := new(string)
	*sourceTypeValue = fmt.Sprintf("ST%d", fieldSuffix)
	sourceInstanceValue := new(string)
	*sourceInstanceValue = fmt.Sprintf("SI%d", fieldSuffix)
	messageTypeValue := new(events.LogMessage_MessageType)
	*messageTypeValue = messageType
	return events.LogMessage{Message: []byte(fmt.Sprintf("MESSAGE%d", fieldSuffix)),
		SourceType:     sourceTypeValue,
		SourceInstance: sourceInstanceValue,
		MessageType:    messageTypeValue,
		Timestamp:      &unixTimestamp}
}

func formatUnixTimestamp(nanosSinceEpoch int64) string {
	secs := nanosSinceEpoch / 1000000000
	nanosecs := nanosSinceEpoch - (secs * 1000000000)
	return time.Unix(secs, nanosecs).Format("2006-01-02T15:04:05.00-0700")
}
