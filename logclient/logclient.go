package logclient

import (
	"crypto/tls"

	"time"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
)

const LogTimestampFormat = "2006-01-02T15:04:05.00-0700"

var CurrentTimezoneLocation = time.Now().Location()

type logClientBuilder struct {
	endpoint           string
	insecureSkipVerify bool
}

func NewLogClientBuilder() *logClientBuilder {
	return &logClientBuilder{}
}

func (builder *logClientBuilder) Endpoint(url string) LogClientBuilder {
	builder.endpoint = url
	return builder
}

func (builder *logClientBuilder) InsecureSkipVerify(skipVerify bool) LogClientBuilder {
	builder.insecureSkipVerify = skipVerify
	return builder
}

func (builder *logClientBuilder) Build() LogClient {
	return &logClient{
		endpoint: builder.endpoint,
		consumer: consumer.New(builder.endpoint, &tls.Config{InsecureSkipVerify: builder.insecureSkipVerify}, nil),
	}
}

type LogClientBuilder interface {
	Endpoint(url string) LogClientBuilder
	InsecureSkipVerify(skipVerify bool) LogClientBuilder
	Build() LogClient
}

type LogClient interface {
	// TODO: do we need to sort the recent logs? Compare the cf CLI
	RecentLogs(serviceGUID string, authToken string) ([]string, error)
	TailingLogs(serviceGUID string, authToken string) (<-chan string, <-chan error)
}

type logClient struct {
	endpoint           string
	consumer           *consumer.Consumer
}

func (lc *logClient) RecentLogs(serviceGUID string, authToken string) ([]string, error) {
	messages, err := lc.consumer.RecentLogs(serviceGUID, "bearer "+authToken)
	if err != nil {
		return nil, err
	}

	result := []string{}
	for _, msg := range messages {
		result = append(result, convertLogMessageToString(msg))
	}

	return result, nil
}

func (lc *logClient) TailingLogs(serviceGUID string, authToken string) (<-chan string, <-chan error) {
	msgChan, errorChan := lc.consumer.TailingLogs(serviceGUID, "bearer "+authToken)
	strMsgChan := make(chan string)

	go func() {
		for msg := range msgChan {
			strMsgChan <- convertLogMessageToString(msg)
		}
	}()

	return strMsgChan, errorChan
}

func convertLogMessageToString(msg *events.LogMessage) string {
	formattedMsg := convertTimestampEpochNanosToString(msg) +
		" [" + msg.GetSourceType() + "/" +
		msg.GetSourceInstance() + "] " +
		msg.GetMessageType().String() + " " +
		string(msg.GetMessage())
	return formattedMsg
}

func convertTimestampEpochNanosToString(message *events.LogMessage) string {
	// The message timestamp appears to be epoch nanoseconds
	timestamp := message.GetTimestamp()
	secs := timestamp / 1000000000
	nanos := timestamp - (1000000000 * secs)
	return time.Unix(secs, nanos).In(CurrentTimezoneLocation).Format(LogTimestampFormat)
}
