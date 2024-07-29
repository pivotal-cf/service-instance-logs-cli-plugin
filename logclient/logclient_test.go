package logclient_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo/v2"
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
		fakeSorter       *logclientfakes.FakeSorter
		testError        error
		currentTimestamp int64
	)

	BeforeEach(func() {
		testError = errors.New(errMessage)
		fakeConsumer = &logclientfakes.FakeConsumer{}
		fakeSorter = &logclientfakes.FakeSorter{}

		builder := logclient.NewLogClientBuilder()
		logClient = builder.InsecureSkipVerify(true).Endpoint(endpointUrl).Build()

		if logClient, ok := logClient.(logclient.FieldSetter); ok {
			logClient.SetConsumer(fakeConsumer)
		} else {
			Fail("logClient did not implement FieldSetter")
		}
	})

	Describe("RecentLogs", func() {
		var (
			result              []string
			err                 error
			mostRecentTimestamp int64
			olderTimestamp      int64
			oldestTimestamp     int64
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
			var consumerRecentLogsResponse []*events.LogMessage

			BeforeEach(func() {
				if logClient, ok := logClient.(logclient.FieldSetter); ok {
					logClient.SetSorter(fakeSorter)
				} else {
					Fail("logClient did not implement FieldSetter")
				}

				consumerRecentLogsResponse = []*events.LogMessage{}
				fakeConsumer.RecentLogsReturns(consumerRecentLogsResponse, nil)
			})

			It("should call the consumer RecentLogs function", func() {
				Expect(fakeConsumer.RecentLogsCallCount()).To(Equal(1))
			})

			It("should use the supplied serviceGUID and authToken for the consumer call", func() {
				svcGuid, token := fakeConsumer.RecentLogsArgsForCall(0)
				Expect(svcGuid).To(Equal(serviceGuid))
				Expect(token).To(Equal("bearer " + authToken))
			})

			It("should call the sorter", func() {
				Expect(fakeSorter.SortRecentCallCount()).To(Equal(1))
			})

			It("should pass the log messages received from the consumer to the sorter", func() {
				messages := fakeSorter.SortRecentArgsForCall(0)
				Expect(messages).To(Equal(consumerRecentLogsResponse))
			})

			It("should return normally", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when request for recent logs from consumer returns normally and received log messages are out of time sequence", func() {
			BeforeEach(func() {
				currentTimestamp = time.Now().UnixNano()
				mostRecentTimestamp = currentTimestamp
				olderTimestamp = currentTimestamp - 1e9  // 1 second ago
				oldestTimestamp = currentTimestamp - 2e9 // 2 seconds ago

				lm1 := createLogMessage("RECENT", events.LogMessage_OUT, mostRecentTimestamp)
				lm2 := createLogMessage("OLDER", events.LogMessage_OUT, olderTimestamp)
				lm3 := createLogMessage("OLDEST", events.LogMessage_ERR, oldestTimestamp)

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

			It("should return correctly formatted messages sorted by timestamp with oldest first", func() {
				Expect(len(result)).To(Equal(3))

				Expect(result[0]).Should(Equal(fmt.Sprintf("%s [ST-OLDEST/SI-OLDEST] ERR MESSAGE-OLDEST",
					formatUnixTimestamp(oldestTimestamp))))
				Expect(result[1]).Should(Equal(fmt.Sprintf("%s [ST-OLDER/SI-OLDER] OUT MESSAGE-OLDER",
					formatUnixTimestamp(olderTimestamp))))
				Expect(result[2]).Should(Equal(fmt.Sprintf("%s [ST-RECENT/SI-RECENT] OUT MESSAGE-RECENT",
					formatUnixTimestamp(mostRecentTimestamp))))
			})
		})

		Context("when request for recent logs returns normally and received log messages all have same timestamp", func() {
			BeforeEach(func() {
				currentTimestamp = time.Now().UnixNano()

				lm1 := createLogMessage("RECEIVED-FIRST", events.LogMessage_OUT, currentTimestamp)
				lm2 := createLogMessage("RECEIVED-SECOND", events.LogMessage_OUT, currentTimestamp)
				lm3 := createLogMessage("RECEIVED-THIRD", events.LogMessage_ERR, currentTimestamp)

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

			It("should return correctly formatted messages sorted by order received", func() {
				Expect(len(result)).To(Equal(3))

				Expect(result[0]).Should(Equal(fmt.Sprintf("%s [ST-RECEIVED-FIRST/SI-RECEIVED-FIRST] OUT MESSAGE-RECEIVED-FIRST",
					formatUnixTimestamp(currentTimestamp))))
				Expect(result[1]).Should(Equal(fmt.Sprintf("%s [ST-RECEIVED-SECOND/SI-RECEIVED-SECOND] OUT MESSAGE-RECEIVED-SECOND",
					formatUnixTimestamp(currentTimestamp))))
				Expect(result[2]).Should(Equal(fmt.Sprintf("%s [ST-RECEIVED-THIRD/SI-RECEIVED-THIRD] ERR MESSAGE-RECEIVED-THIRD",
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
				lm1 := createLogMessage("1", events.LogMessage_OUT, currentTimestamp)
				logMsgsChan <- &lm1

				lm2 := createLogMessage("2", events.LogMessage_ERR, currentTimestamp)
				logMsgsChan <- &lm2

				lm3 := createLogMessage("3", events.LogMessage_OUT, currentTimestamp)
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
				Expect(receivedMsg).Should(Equal(fmt.Sprintf("%s [ST-1/SI-1] OUT MESSAGE-1",
					formatUnixTimestamp(currentTimestamp))))

				Eventually(logStringsChan).Should(Receive(&receivedMsg))
				Expect(receivedMsg).Should(Equal(fmt.Sprintf("%s [ST-2/SI-2] ERR MESSAGE-2",
					formatUnixTimestamp(currentTimestamp))))

				Eventually(logStringsChan).Should(Receive(&receivedMsg))
				Expect(receivedMsg).Should(Equal(fmt.Sprintf("%s [ST-3/SI-3] OUT MESSAGE-3",
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

func createLogMessage(fieldSuffix string, messageType events.LogMessage_MessageType, unixTimestamp int64) events.LogMessage {
	sourceTypeValue := new(string)
	*sourceTypeValue = fmt.Sprintf("ST-%s", fieldSuffix)
	sourceInstanceValue := new(string)
	*sourceInstanceValue = fmt.Sprintf("SI-%s", fieldSuffix)
	messageTypeValue := new(events.LogMessage_MessageType)
	*messageTypeValue = messageType
	return events.LogMessage{
		Message:        []byte(fmt.Sprintf("MESSAGE-%s", fieldSuffix)),
		SourceType:     sourceTypeValue,
		SourceInstance: sourceInstanceValue,
		MessageType:    messageTypeValue,
		Timestamp:      &unixTimestamp,
	}
}

func formatUnixTimestamp(nanosSinceEpoch int64) string {
	secs := nanosSinceEpoch / 1000000000
	nanosecs := nanosSinceEpoch - (secs * 1000000000)
	return time.Unix(secs, nanosecs).Format("2006-01-02T15:04:05.00-0700")
}
