package logging

import (
	"code.cloudfoundry.org/cli/plugin"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/httpclient"
)

func Logs(cliConnection plugin.CliConnection, serviceInstanceName string, recent bool, authClient httpclient.AuthenticatedClient) (string, error) {
	return "", nil
}
