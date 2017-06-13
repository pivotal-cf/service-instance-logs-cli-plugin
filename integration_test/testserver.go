package main

import (
	"flag"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"io"

	"fmt"

	"github.com/cloudfoundry/dropsonde/envelope_sender"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
)

var (
	numberOfLogEntriesReturned int64
	writeOldestMessagesFirst   bool
)

type HTTPEventEmitter struct {
	mpw *multipart.Writer
}

func newHTTPEmitter(rw http.ResponseWriter) *HTTPEventEmitter {
	mp := multipart.NewWriter(rw)
	rw.Header().Set("Content-Type", "multipart/x-protobuf; boundary="+mp.Boundary())
	return &HTTPEventEmitter{mpw: mp}
}

func (e *HTTPEventEmitter) Origin() string {
	return "origin"
}

func (e *HTTPEventEmitter) Emit(event events.Event) error {
	partWriter, err := e.mpw.CreatePart(nil)
	binaryData, _ := event.(*events.LogMessage).Marshal()
	_, err = partWriter.Write(binaryData)
	return err
}

func (e *HTTPEventEmitter) EmitEnvelope(envelope *events.Envelope) error {
	partWriter, err := e.mpw.CreatePart(nil)
	if err != nil {
		return err
	}

	binaryData, err := envelope.Marshal()
	if err != nil {
		return err
	}

	_, err = partWriter.Write(binaryData)
	return err
}

func (e *HTTPEventEmitter) Close() error {
	return e.mpw.Close()
}

//noinspection GoUnusedParameter
func dumpServiceLogs(rw http.ResponseWriter, r *http.Request) {
	emitter := newHTTPEmitter(rw)
	sender := envelope_sender.NewEnvelopeSender(emitter)

	var secondsInPastGenerator func() int64
	if writeOldestMessagesFirst {
		secondsInPastGenerator = makeDescendingGenerator(numberOfLogEntriesReturned)
	} else {
		secondsInPastGenerator = makeAscendingGenerator()
	}

	var i int64
	for i = 0; i < numberOfLogEntriesReturned; i++ {
		env := &events.Envelope{
			EventType: events.Envelope_LogMessage.Enum(),
			Origin:    proto.String("origin"),
			LogMessage: makeLogMessage("appID",
				fmt.Sprintf("This is log message %d", i),
				"sourceType",
				"sourceInstance",
				events.LogMessage_OUT,
				secondsInPastGenerator()),
		}
		sender.SendEnvelope(env)
	}
	err := emitter.Close()
	if err != nil {
		panic(fmt.Sprintf("Error closing response writer: %s", err.Error()))
	}
}

func makeAscendingGenerator() func() int64 {
	i := int64(0)
	return func() (result int64) {
		result = i
		i += 1
		return
	}
}

func makeDescendingGenerator(start int64) func() int64 {
	i := start
	return func() (result int64) {
		result = i
		i -= 1
		return
	}
}

//noinspection GoUnusedParameter
func apiInfo(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	rw.Header().Set("X-Content-Type-Options", "nosniff")
	rw.Header().Set("X-Vcap-Request-Id", "dd8699dd-e5c9-4951-4b83-4fc2c70de676")
	rw.Header().Add("X-Vcap-Request-Id", "dd8699dd-e5c9-4951-4b83-4fc2c70de676::101c0ae8-7aff-40d3-8af8-eddb65205492")
	io.WriteString(rw, `{ "api_version": "2.75.0", "app_ssh_endpoint": "localhost:2222", "app_ssh_host_key_fingerprint": "9f:ae:12:42:19:33:6e:cc:5b:5b:44:af:13:a1:04:22", "app_ssh_oauth_client": "ssh-proxy", "authorization_endpoint": "http://localhost:8888", "build": "", "description": "fake api for integration testing purposes", "doppler_logging_endpoint": "wss://localhost:443", "logging_endpoint": "wss://localhost:443", "min_cli_version": "6.22.0", "min_recommended_cli_version": "6.23.0", "name": "integration", "routing_endpoint": "https://localhost:8888/routing", "support": "https://support.pivotal.io", "token_endpoint": "http://localhost:8888", "version": 0}`)
}

//noinspection GoUnusedParameter
func login(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	io.WriteString(rw, `{
  "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleS0xIiwidHlwIjoiSldUIn0.eyJqdGkiOiIyNWQ3NmZmY2IyYTA0ZjEwOWFlZGMxMzE5ZDZkMTMzNiIsInN1YiI6IjJkMDVmMmM3LTBlNjItNGNhYi04NTIyLWQ0MDM0Yjg5ODM1YSIsInNjb3BlIjpbIm9wZW5pZCIsInJvdXRpbmcucm91dGVyX2dyb3Vwcy53cml0ZSIsInNjaW0ucmVhZCIsImNsb3VkX2NvbnRyb2xsZXIuYWRtaW4iLCJ1YWEudXNlciIsInJvdXRpbmcucm91dGVyX2dyb3Vwcy5yZWFkIiwiY2xvdWRfY29udHJvbGxlci5yZWFkIiwicGFzc3dvcmQud3JpdGUiLCJjbG91ZF9jb250cm9sbGVyLndyaXRlIiwibmV0d29yay5hZG1pbiIsImRvcHBsZXIuZmlyZWhvc2UiLCJzY2ltLndyaXRlIl0sImNsaWVudF9pZCI6ImNmIiwiY2lkIjoiY2YiLCJhenAiOiJjZiIsImdyYW50X3R5cGUiOiJwYXNzd29yZCIsInVzZXJfaWQiOiIyZDA1ZjJjNy0wZTYyLTRjYWItODUyMi1kNDAzNGI4OTgzNWEiLCJvcmlnaW4iOiJ1YWEiLCJ1c2VyX25hbWUiOiJhZG1pbiIsImVtYWlsIjoiYWRtaW4iLCJyZXZfc2lnIjoiMTQ3ZDViN2UiLCJpYXQiOjE0OTcwMjk3MzQsImV4cCI6MTQ5NzAzNjkzNCwiaXNzIjoiaHR0cHM6Ly91YWEub2xpdmUuc3ByaW5nYXBwcy5pby9vYXV0aC90b2tlbiIsInppZCI6InVhYSIsImF1ZCI6WyJjbG91ZF9jb250cm9sbGVyIiwic2NpbSIsInBhc3N3b3JkIiwiY2YiLCJ1YWEiLCJvcGVuaWQiLCJkb3BwbGVyIiwicm91dGluZy5yb3V0ZXJfZ3JvdXBzIiwibmV0d29yayJdfQ.RtIkPPjJA76rZY84Sc9sb-_Kyk0lN8y-oqpsP-h29VC9mZMlSERy3KQoMXzQzWlcta6Ft80l5r57GA6LUtEn6Q1OpfpmG20l4sU16Xek-Gz0rJvtwUaE09TBIvCkFBdzR211USW88s7TbJNTXTcGv1QUikmEoIqJAyZT8P1HE48wJarVFyL2_iYvZoltQMyEnUuKPfwV0kcnfYWcWvZpinC5VtAU6RtZ0hlNLWZBn7oV72QKIVD7LXJud3g_NMepLvUMYUB7CgAmOwzjyxiq3a2EZzHQl1prIgX1iBa2TD9V6SyTgkr2DXF5cex1mcU7sABEk01jCwbw1Bx1ebd5aQ",
  "expires_in": 7199,
  "jti": "77cc685e8e244bd88ef3e7227ab15110",
  "refresh_token": "655b19516a6f4c4a949244e88569a514-r",
  "scope": "cloud_controller.adminrouting.router_groups.readcloud_controller.writenetwork.admindoppler.firehoseopenidrouting.router_groups.writescim.readuaa.usercloud_controller.readpassword.writescim.write",
  "token_type": "bearer"
}`)
}

//noinspection GoUnusedParameter
func servicesInfo(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	io.WriteString(rw, `{
  "total_results": 1,
  "total_pages": 1,
  "prev_url": null,
  "next_url": null,
  "resources": [{
    "metadata": {
      "guid": "service-instance-guid",
      "url": "/v2/services/service-instance-guid",
      "created_at": "2017-03-22T02:30:50Z",
      "updated_at": "2017-03-22T02:30:50Z"
    },
    "entity": {
      "label": "p-config-server",
      "provider": null,
      "url": null,
      "description": "Config Server for Spring Cloud Applications",
      "long_description": null,
      "version": null,
      "info_url": null,
      "active": true,
      "bindable": true,
      "unique_id": "unique-id",
      "extra": "{\"serviceInstanceLogsEndpoint\":\"ws://localhost:8888\"}",
      "tags": ["configuration", "spring-cloud"],
      "requires": [],
      "documentation_url": null,
      "service_broker_guid": "b1293bff-eda6-4583-9b26-bf181eef6627",
      "plan_updateable": false,
      "service_plans_url": "/v2/services/service-instance-guid/service_plans"
    }
  }]
}`)
}

//noinspection GoUnusedParameter
func orgsInfo(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	io.WriteString(rw, `{
  "total_results": 1,
  "total_pages": 1,
  "prev_url": null,
  "next_url": null,
  "resources": [{
    "metadata": {
      "guid": "test-org-guid",
      "url": "/v2/organizations/test-org-guid",
      "created_at": "2017-03-27T16:07:07Z",
      "updated_at": "2017-03-27T16:07:07Z"
    },
    "entity": {
      "name": "testorg",
      "billing_enabled": false,
      "quota_definition_guid": "7d4da11f-4e96-445d-ac2b-35f209739b88",
      "status": "active",
      "default_isolation_segment_guid": null,
      "quota_definition_url": "/v2/quota_definitions/7d4da11f-4e96-445d-ac2b-35f209739b88",
      "spaces_url": "/v2/organizations/test-org-guid/spaces",
      "domains_url": "/v2/organizations/test-org-guid/domains",
      "private_domains_url": "/v2/organizations/test-org-guid/private_domains",
      "users_url": "/v2/organizations/test-org-guid/users",
      "managers_url": "/v2/organizations/test-org-guid/managers",
      "billing_managers_url": "/v2/organizations/test-org-guid/billing_managers",
      "auditors_url": "/v2/organizations/test-org-guid/auditors",
      "app_events_url": "/v2/organizations/test-org-guid/app_events",
      "space_quota_definitions_url": "/v2/organizations/test-org-guid/space_quota_definitions"
    }
  }]
}`)
}

//noinspection GoUnusedParameter
func spacesInfo(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	io.WriteString(rw, `{
  "total_results": 1,
  "total_pages": 1,
  "prev_url": null,
  "next_url": null,
  "resources": [{
    "metadata": {
      "guid": "test-space-guid",
      "url": "/v2/spaces/test-space-guid",
      "created_at": "2017-03-22T09:56:50Z",
      "updated_at": "2017-03-22T09:56:50Z"
    },
    "entity": {
      "name": "testspace",
      "organization_guid": "test-org-guid",
      "space_quota_definition_guid": null,
      "isolation_segment_guid": null,
      "allow_ssh": true,
      "organization_url": "/v2/organizations/test-org-guid",
      "developers_url": "/v2/spaces/test-space-guid/developers",
      "managers_url": "/v2/spaces/test-space-guid/managers",
      "auditors_url": "/v2/spaces/test-space-guid/auditors",
      "apps_url": "/v2/spaces/test-space-guid/apps",
      "routes_url": "/v2/spaces/test-space-guid/routes",
      "domains_url": "/v2/spaces/test-space-guid/domains",
      "service_instances_url": "/v2/spaces/test-space-guid/service_instances",
      "app_events_url": "/v2/spaces/test-space-guid/app_events",
      "events_url": "/v2/spaces/test-space-guid/events",
      "security_groups_url": "/v2/spaces/test-space-guid/security_groups",
      "staging_security_groups_url": "/v2/spaces/test-space-guid/staging_security_groups"
    }
  }]
}`)
}

//noinspection GoUnusedParameter
func serviceInstances(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	io.WriteString(rw, `{
  "total_results": 1,
  "total_pages": 1,
  "prev_url": null,
  "next_url": null,
  "resources": [{
    "metadata": {
      "guid": "test-service-instance-guid",
      "url": "/v2/service_instances/test-service-instance-guid",
      "created_at": "2017-06-12T08:09:58Z",
      "updated_at": "2017-06-12T08:09:58Z"
    },
    "entity": {
      "name": "test-service",
      "credentials": {},
      "service_plan_guid": "test-service-plan-guid",
      "space_guid": "test-space-guid",
      "gateway_data": null,
      "dashboard_url": "https://spring-cloud-broke.olive.springapps.io/dashboard/p-config-server/test-service-instance-guid",
      "type": "managed_service_instance",
      "last_operation": {
        "type": "create",
        "state": "succeeded",
        "description": "",
        "updated_at": "2017-06-12T08:11:02Z",
        "created_at": "2017-06-12T08:09:58Z"
      },
      "tags": [],
      "service_guid": "test-service-guid",
      "space_url": "/v2/spaces/test-space-guid",
      "space": {
        "metadata": {
          "guid": "test-space-guid",
          "url": "/v2/spaces/test-space-guid",
          "created_at": "2017-03-22T09:56:50Z",
          "updated_at": "2017-03-22T09:56:50Z"
        },
        "entity": {
          "name": "testspace",
          "organization_guid": "test-org-guid",
          "space_quota_definition_guid": null,
          "isolation_segment_guid": null,
          "allow_ssh": true,
          "organization_url": "/v2/organizations/test-org-guid",
          "developers_url": "/v2/spaces/test-space-guid/developers",
          "managers_url": "/v2/spaces/test-space-guid/managers",
          "auditors_url": "/v2/spaces/test-space-guid/auditors",
          "apps_url": "/v2/spaces/test-space-guid/apps",
          "routes_url": "/v2/spaces/test-space-guid/routes",
          "domains_url": "/v2/spaces/test-space-guid/domains",
          "service_instances_url": "/v2/spaces/test-space-guid/service_instances",
          "app_events_url": "/v2/spaces/test-space-guid/app_events",
          "events_url": "/v2/spaces/test-space-guid/events",
          "security_groups_url": "/v2/spaces/test-space-guid/security_groups",
          "staging_security_groups_url": "/v2/spaces/test-space-guid/staging_security_groups"
        }
      },
      "service_plan_url": "/v2/service_plans/test-service-plan-guid",
      "service_plan": {
        "metadata": {
          "guid": "test-service-plan-guid",
          "url": "/v2/service_plans/test-service-plan-guid",
          "created_at": "2017-03-22T16:26:22Z",
          "updated_at": "2017-03-22T16:26:23Z"
        },
        "entity": {
          "name": "standard",
          "free": true,
          "description": "StandardPlan",
          "service_guid": "test-service-guid",
          "extra": "{\"bullets\":[\"Single-tenant\",\"Backedbyuser-providedGitrepository\"]}",
          "unique_id": "unique-id",
          "public": true,
          "bindable": true,
          "active": true,
          "service_url": "/v2/services/test-service-guid",
          "service_instances_url": "/v2/service_plans/test-service-plan-guid/service_instances"
        }
      },
      "service_bindings_url": "/v2/service_instances/test-service-instance-guid/service_bindings",
      "service_bindings": [],
      "service_keys_url": "/v2/service_instances/test-service-instance-guid/service_keys",
      "service_keys": [],
      "routes_url": "/v2/service_instances/test-service-instance-guid/routes",
      "routes": [],
      "service_url": "/v2/services/test-service-guid"
    }
  }]
}`)
}

//noinspection GoUnusedParameter
func testServiceInstanceInfo(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	io.WriteString(rw, `{
  "metadata": {
    "guid": "test-service-instance-guid",
    "url": "/v2/service_instances/test-service-instance-guid",
    "created_at": "2017-06-12T08:09:58Z",
    "updated_at": "2017-06-12T08:09:58Z"
  },
  "entity": {
    "label": "test-service",
    "provider": null,
    "url": null,
    "description": "Oldest log entries logged first",
    "long_description": null,
    "version": null,
    "info_url": null,
    "active": true,
    "bindable": true,
    "unique_id": "test-service-unique-id",
    "extra": "{ \"longDescription\": \"Whatevs\", \"documentationUrl\": \"http://docs.pivotal.io/spring-cloud-services/\", \"providerDisplayName\": \"Pivotal\", \"displayName\": \"ConfigServer\", \"supportUrl\": \"http://support.pivotal.io/\",\"serviceInstanceLogsEndpoint\":\"wss://localhost:8888\"}",
    "tags": ["configuration", "spring-cloud"],
    "requires": [],
    "documentation_url": null,
    "service_broker_guid": "service-broker-guid",
    "plan_updateable": false,
    "service_plans_url": "/v2/services/test-service-instance-guid/service_plans"
  }
}`)
}

//noinspection GoUnusedParameter
func servicesSummary(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	io.WriteString(rw, `{
  "guid": "test-space-guid",
  "name": "testspace",
  "apps": [

  ],
  "services": [
    {
      "guid": "test-service-instance-guid",
      "name": "testservice",
      "bound_app_count": 0,
      "last_operation": {
        "type": "create",
        "state": "succeeded",
        "description": "",
        "updated_at": "2017-06-12T08:11:02Z",
        "created_at": "2017-06-12T08:09:58Z"
      },
      "dashboard_url": "https://fakeserver/test-service-instance-guid",
      "service_plan": {
        "guid": "test-service-plan-guid",
        "name": "standard",
        "service": {
          "guid": "cab8959f-10d2-4467-a963-00d43cf511f1",
          "label": "p-config-server",
          "provider": null,
          "version": null
        }
      }
    }
  ]
}`)
}

func makeLogMessage(appID, message, sourceType, sourceInstance string, messageType events.LogMessage_MessageType, secondsInPast int64) *events.LogMessage {
	ts := time.Now().UnixNano() - (secondsInPast * 1e9)

	return &events.LogMessage{
		Message:        []byte(message),
		AppId:          proto.String(appID),
		MessageType:    &messageType,
		SourceType:     &sourceType,
		SourceInstance: &sourceInstance,
		Timestamp:      proto.Int64(ts),
	}
}

func main() {
	flag.Usage = func() {
		fmt.Printf(`Usage: testserver [options]

Starts a simple test log server that will return a number of recent log
entries for a fake SCS service.

`)
		flag.PrintDefaults()
		fmt.Println()
	}

	addrPtr := flag.String("addr", "localhost:8888", "Log server address")
	numberOfLogEntriesReturnedPtr := flag.Int64("num", 200, "Number of log entries to return")
	oldestFirstPtr := flag.Bool("oldfirst", false, "Write older timestamped messages to log first")
	flag.Parse()

	numberOfLogEntriesReturned = *numberOfLogEntriesReturnedPtr
	writeOldestMessagesFirst = *oldestFirstPtr

	log.SetFlags(0)
	log.Printf("Server starting on %s", *addrPtr)

	http.HandleFunc("/v2/info", apiInfo)
	http.HandleFunc("/login", login)
	http.HandleFunc("/oauth/token", login)
	http.HandleFunc("/v2/organizations", orgsInfo)
	http.HandleFunc("/v2/spaces", spacesInfo)
	http.HandleFunc("/v2/spaces/test-space-guid/summary", servicesSummary)
	http.HandleFunc("/v2/services", servicesInfo)
	http.HandleFunc("/v2/spaces/test-space-guid/service_instances", serviceInstances)
	http.HandleFunc("/v2/services/test-service-guid", testServiceInstanceInfo)
	http.HandleFunc("/apps/test-service-instance-guid/recentlogs", dumpServiceLogs)

	if err := http.ListenAndServe(*addrPtr, nil); err != nil {
		log.Fatal(err)
	}
}
