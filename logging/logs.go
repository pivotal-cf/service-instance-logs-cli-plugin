package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
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
	serviceName := model.ServiceOffering.Name

	// get auth token
	accessToken, err := cfutil.GetToken(cliConnection)
	if err != nil {
		return err
	}

	serviceInstanceLogsEndpoint, err := obtainServiceInstanceLogsEndpoint(cliConnection, serviceName)
	if err != nil {
		return err
	}

	logClient := logClientBuilder.Endpoint(serviceInstanceLogsEndpoint).Build()

	// Print a blank line.
	fmt.Fprintln(w)

	if recent {
		return dumpRecentLogs(logClient, serviceInstanceGUID, accessToken, w)
	}

	return tailLogs(logClient, serviceInstanceGUID, accessToken, w)
}

type ServicesStructure struct {
	TotalResults int `json:"total_results"`
	Resources    []ResourceStructure
}

type ResourceStructure struct {
	Entity EntityStructure
}

type EntityStructure struct {
	Extra string
}

type ExtraStructure struct {
	ServiceInstanceLogsEndpoint string
}

func obtainServiceInstanceLogsEndpoint(cliConnection plugin.CliConnection, serviceName string) (string, error) {
	output, err := cliConnection.CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v2/services?q=label:%s", serviceName))
	if err != nil {
		return "", fmt.Errorf("/v2/services failed: %s", err)
	}

	var services ServicesStructure
	err = json.Unmarshal([]byte(strings.Join(output, "\n")), &services)
	if err != nil {
		return "", fmt.Errorf("/v2/services returned invalid JSON: %s", err)
	}

	if services.TotalResults == 0 {
		return "", fmt.Errorf("/v2/services did not return the service instance")
	}

	var extra ExtraStructure
	err = json.Unmarshal([]byte(services.Resources[0].Entity.Extra), &extra)
	if err != nil {
		return "", fmt.Errorf("/v2/services 'extra' field contained invalid JSON: %s", err)
	}

	if extra.ServiceInstanceLogsEndpoint == "" {
		return "", fmt.Errorf("/v2/services did not contain a service instance logs endpoint: maybe the broker version is too old")
	}

	return extra.ServiceInstanceLogsEndpoint, nil
}
