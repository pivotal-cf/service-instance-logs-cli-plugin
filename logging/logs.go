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
	wg.Add(1)

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

	err := <-errorChan
	if err != nil {
		return err
	}

	wg.Wait()

	return nil

}

func Logs(cliConnection plugin.CliConnection, w io.Writer, serviceInstanceName string, recent bool, logClientBuilder logclient.LogClientBuilder) error {
	// get service GUID from service instance name
	model, err := cliConnection.GetService(serviceInstanceName)
	if err != nil {
		return err
	}
	serviceInstanceGUID := model.Guid

	// get auth token
	accessToken, err := cfutil.GetToken(cliConnection)
	if err != nil {
		return err
	}

	// get metadata from cf curl /v2/services // is there a way to get just the service we need?
	// deserialise metadata
	// pluck out the logs endpoint URL
	// FIXME: temporary code to use doppler endpoint rather than that provided for the service
	url, err := cliConnection.DopplerEndpoint()
	if err != nil {
		return err
	}

	logClient := logClientBuilder.Endpoint(url).Build()

	// Print a blank line.
	fmt.Fprintln(w)

	if recent {
		return dumpRecentLogs(logClient, serviceInstanceGUID, accessToken, w)
	}

	return tailLogs(logClient, serviceInstanceGUID, accessToken, w)
}
