package logging

import (
	"fmt"
	"io"

	"sync"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/cfutil"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logclient"
)

func dumpRecentLogs(logClient logclient.LogClient, serviceGUID string, accessToken string, w io.Writer) error {
	messages, err := logClient.RecentLogs(serviceGUID, accessToken)
	if err != nil {
		return err
	}

	for _, msg := range messages {
		fmt.Fprintln(w, msg)
	}

	return nil
}

func tailLogs(logClient logclient.LogClient, serviceGUID string, accessToken string, w io.Writer) error {
	msgChan, errorChan := logClient.TailingLogs(serviceGUID, accessToken)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			msg, ok := <-msgChan
			if !ok {
				break
			}
			fmt.Fprintf(w, "%s\n", msg)
		}
	}()

	outcome := make(chan error)

	go func() {
		defer wg.Done()
		for {
			logErr, ok := <-errorChan
			if !ok {
				break
			}
			outcome <- logErr
		}
	}()

	outcomeErr := <-outcome
	if outcomeErr != nil {
		return outcomeErr
	}

	wg.Wait()

	return nil

}

func Logs(cliConnection plugin.CliConnection, w io.Writer, serviceInstanceName string, recent bool, logClientBuilder logclient.LogClientBuilder) error {
	// get metadata from cf curl /v2/services // is there a way to get just the service we need?
	// deserialise metadata
	// pluck out the logs endpoint URL
	// FIXME: temporary code to use doppler endpoint rather than that provided for the service
	url, err := cliConnection.DopplerEndpoint()
	if err != nil {
		return err
	}

	logClient := logClientBuilder.Endpoint(url).Build()

	// get service GUID from service instance name
	// FIXME: temporary hack - pass an app GUID as the SI name
	guid := serviceInstanceName

	// get auth token
	accessToken, err := cfutil.GetToken(cliConnection)
	if err != nil {
		return err
	}

	// Print a blank line.
	fmt.Fprintln(w)

	if recent {
		return dumpRecentLogs(logClient, guid, accessToken, w)
	}

	return tailLogs(logClient, guid, accessToken, w)
}
