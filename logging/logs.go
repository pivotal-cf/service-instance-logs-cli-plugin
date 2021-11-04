package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"net/url"

	"errors"

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

	exitChan := make(chan error, 1)
	errorChanNotOK := errors.New("errorChan not OK")
	go func() {
		for {
			err, ok := <-errorChan
			if !ok {
				exitChan <- errorChanNotOK
				break
			}

			if !strings.Contains(err.Error(), "1006") {
				exitChan <- err
				break
			}
		}
	}()

	exitErr := <-exitChan
	if exitErr != errorChanNotOK {
		return exitErr
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

	// obtain the service plan for the specific instance. This is needed when multiple service brokers provide the same service with the same label
	output, err := cliConnection.CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v2/service_plans/%s", model.ServicePlan.Guid))

	if err != nil {
		return fmt.Errorf("/v2/service_plans failed: %w", err)
	}

	var servicePlan ServicePlanStructure
	err = json.Unmarshal([]byte(strings.Join(output, "\n")), &servicePlan)
	if err != nil {
		return fmt.Errorf("/v2/service_plan returned invalid JSON: %s", err)
	}

	serviceGuid := servicePlan.Entity.ServiceGuid

	// get auth token
	accessToken, err := cfutil.GetToken(cliConnection)
	if err != nil {
		return err
	}

	serviceInstanceLogsEndpoint, err := obtainServiceInstanceLogsEndpoint(cliConnection, serviceGuid)
	if err != nil {
		return err
	}

	if !recent {
		serviceInstanceLogsEndpoint, err = convertServiceInstanceLogsEndpoint(serviceInstanceLogsEndpoint)
		if err != nil {
			return err
		}
	}

	logClient := logClientBuilder.Endpoint(serviceInstanceLogsEndpoint).Build()

	// Print a blank line.
	fmt.Fprintln(w)

	if recent {
		return dumpRecentLogs(logClient, serviceInstanceGUID, accessToken, w)
	}

	return tailLogs(logClient, serviceInstanceGUID, accessToken, w)
}

type ServiceStructure struct {
	Entity EntityStructure
}

type ServicePlanStructure struct {
	Entity ServicePlanEntity `json:"entity"`
}

type ServicePlanEntity struct {
	ServiceGuid string `json:"service_guid"`
}

type ResourceStructure struct {
	Entity EntityStructure `json:"entity"`
}

type EntityStructure struct {
	Extra string
}

type ExtraStructure struct {
	ServiceInstanceLogsEndpoint string
}

func obtainServiceInstanceLogsEndpoint(cliConnection plugin.CliConnection, serviceGuid string) (string, error) {
	output, err := cliConnection.CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v2/services/%s", serviceGuid))

	if err != nil {
		return "", fmt.Errorf("/v2/services failed: %s", err)
	}

	var service ServiceStructure
	err = json.Unmarshal([]byte(strings.Join(output, "\n")), &service)
	if err != nil {
		return "", fmt.Errorf("/v2/services returned invalid JSON: %s", err)
	}

	var extra ExtraStructure
	err = json.Unmarshal([]byte(service.Entity.Extra), &extra)
	if err != nil {
		return "", fmt.Errorf("/v2/services 'extra' field contained invalid JSON: %s", err)
	}

	if extra.ServiceInstanceLogsEndpoint == "" {
		return "", fmt.Errorf("/v2/services did not contain a service instance logs endpoint: maybe the broker version is too old")
	}

	return extra.ServiceInstanceLogsEndpoint, nil
}

func convertServiceInstanceLogsEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = ""
	return u.String(), nil
}
