package main_test

import (
	"os/exec"
	"time"

	"strconv"

	"bufio"

	"syscall"

	"strings"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/logclient"
)

const (
	testServerPort              = "8888"
	testServerAddress           = "localhost:" + testServerPort
	endpointUrl                 = "ws://" + testServerAddress
	requestedNumberOfLogEntries = 10
	logTimestampFormat          = "2006-01-02T15:04:05.00-0700"
	oauthToken                  = "oauthtoken"
	serviceGuid                 = "test-service-instance-guid"
)

var (
	testServer *exec.Cmd
	logClient  logclient.LogClient
	logs       []string
	err        error
)

var _ = BeforeSuite(func() {
	testServer = exec.Command("go")
	testServer.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // required for successful termination of server later
	testServer.Args = append(testServer.Args, "run", "testserver.go")
	testServer.Args = append(testServer.Args, "-num", strconv.Itoa(requestedNumberOfLogEntries))
	testServer.Args = append(testServer.Args, "-oldlast")
	testServer.Args = append(testServer.Args, "-addr", testServerAddress)

	stdout, err := testServer.StdoutPipe()
	Expect(err).NotTo(HaveOccurred())

	err = testServer.Start()
	Expect(err).NotTo(HaveOccurred())

	// The test server does not start immediately but, rather than have a fixed sleep, block for up to a maximum
	// of 10 seconds until the test server writes its startup status message out.
	c1 := make(chan string, 1)

	go func() {
		r := bufio.NewReader(stdout)
		line, _, err := r.ReadLine()
		Expect(err).NotTo(HaveOccurred())
		c1 <- string(line)
	}()

	select {
	case line := <-c1:
		fmt.Printf("%s\n", line)
	case <-time.After(time.Second * 10):
		Fail("Timed out waiting for test server to start")
	}
})

var _ = AfterSuite(func() {
	// testServer.Process.Kill() does not kill the test server as expected but the following code
	// will kill the test server process. See https://groups.google.com/forum/#!topic/Golang-Nuts/XoQ3RhFBJl8
	processGroupId, err := syscall.Getpgid(testServer.Process.Pid)
	Expect(err).NotTo(HaveOccurred())

	err = syscall.Kill(-processGroupId, 15) // note the minus sign
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Logclient integration test", func() {

	BeforeEach(func() {
		builder := logclient.NewLogClientBuilder()
		logClient = builder.InsecureSkipVerify(true).Endpoint(endpointUrl).Build()
	})

	Describe("Verify Logclient interaction with service instance logs endpoint", func() {

		JustBeforeEach(func() {
			logs, err = logClient.RecentLogs(serviceGuid, oauthToken)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when recent logs are requested", func() {
			It("should have sorted log entries by timestamp in ascending order (most recent last)", func() {
				for i := 0; i < len(logs)-1; i++ {
					currentTimestamp := getUnixTimestampFromLogEntry(logs[i])
					nextTimestamp := getUnixTimestampFromLogEntry(logs[i+1])
					Expect(currentTimestamp).To(BeNumerically("<", nextTimestamp))
				}
			})
		})
	})
})

func getUnixTimestampFromLogEntry(logMessage string) int64 {
	timestampString := strings.Split(logMessage, " ")[0]
	t, err := time.Parse(logTimestampFormat, timestampString)
	if err != nil {
		panic(err.Error())
	}

	return t.Unix()
}
