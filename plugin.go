/*
 * Copyright 2017-Present the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/cli"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/format"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/httpclient"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logging"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/pluginutil"
)

// Plugin version. Substitute "<major>.<minor>.<build>" at build time, e.g. using -ldflags='-X main.pluginVersion=1.2.3'
var pluginVersion = "invalid version - plugin was not built correctly"

// Plugin is a struct implementing the Plugin interface, defined by the core CLI, which can
// be found in "code.cloudfoundry.org/cli/plugin/plugin.go".
type Plugin struct{}

func (c *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	recent, skipSslValidation, positionalArgs, err := cli.ParseFlags(args)
	if err != nil {
		format.Diagnose(string(err.Error()), os.Stderr, func() {
			os.Exit(1)
		})
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSslValidation},
	}
	client := &http.Client{Transport: tr}
	authClient := httpclient.NewAuthenticatedClient(client)

	switch args[0] {

	case "service-instance-logs":
		serviceInstanceName := getServiceInstanceName(positionalArgs, args[0])
		var behaviour string
		if recent {
			behaviour = "Dumping recent"
		} else {
			behaviour = "Tailing"
		}
		runAction(cliConnection, fmt.Sprintf("%s logs for service instance %s", behaviour, format.Bold(format.Cyan(serviceInstanceName))), func() (string, error) {
			return logging.Logs(cliConnection, serviceInstanceName, recent, authClient)
		})

	default:
		os.Exit(0) // Ignore CLI-MESSAGE-UNINSTALL etc.

	}
}

func getServiceInstanceName(args []string, operation string) string {
	if len(args) < 2 || args[1] == "" {
		diagnoseWithHelp("Service instance name not specified.", operation)
	}
	return args[1]

}

func runAction(cliConnection plugin.CliConnection, message string, action func() (string, error)) {
	format.RunAction(cliConnection, message, action, os.Stdout, func() {
		os.Exit(1)
	})
}

func runActionQuietly(cliConnection plugin.CliConnection, action func() (string, error)) {
	format.RunActionQuietly(cliConnection, action, os.Stdout, func() {
		os.Exit(1)
	})
}

func diagnoseWithHelp(message string, operation string) {
	fmt.Printf("%s See 'cf help %s.'\n", message, operation)
	os.Exit(1)
}

func failInstallation(format string, inserts ...interface{}) {
	// There is currently no way to emit the message to the command line during plugin installation. Standard output and error are swallowed.
	fmt.Printf(format, inserts...)
	fmt.Println("")

	// Fail the installation
	os.Exit(64)
}

func (c *Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:    "SILogs",
		Version: pluginutil.ParsePluginVersion(pluginVersion, failInstallation),
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 25,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "service-instance-logs",
				HelpText: "Tail or show recent logs for a service instance",
				Alias:    "sil",
				UsageDetails: plugin.Usage{
					Usage: "   cf service-instance-logs SERVICE_INSTANCE_NAME",
					Options: map[string]string{"--skip-ssl-validation": cli.SkipSslValidationUsage,
						"--recent": cli.RecentUsage},
				},
			},
		},
	}
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("This program is a plugin which expects to be installed into the cf CLI. It is not intended to be run stand-alone.")
		pv := pluginutil.ParsePluginVersion(pluginVersion, failInstallation)
		fmt.Printf("Plugin version: %d.%d.%d\n", pv.Major, pv.Minor, pv.Build)
		os.Exit(0)
	}
	plugin.Start(new(Plugin))
}
