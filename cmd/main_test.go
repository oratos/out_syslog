package main_test

import (
	"bufio"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Syslog Output Plugin", func() {
	DescribeTable("writes out logs to syslog", func(msgs []string) {
		logPath, cleanup := writeLog(msgs)
		defer cleanup()
		spyDrain := newSpyDrain()
		defer spyDrain.stop()

		cmd := exec.Command(
			"docker",
			"run",
			"--network", "host",
			"--volume", path.Dir(pluginPath)+":/plugin",
			"--volume", path.Dir(logPath)+":/input",
			"fluent/fluent-bit:0.13.4",
			"/fluent-bit/bin/fluent-bit",
			"--flush", "1",
			"--plugin", path.Join("/plugin", path.Base(pluginPath)),
			"--input", "tail",
			"--prop", "Path="+path.Join("/input", path.Base(logPath)),
			"--output", "syslog",
			"--prop", "Addr="+spyDrain.url(),
		)
		sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		defer sess.Wait()
		defer sess.Kill()

		spyDrain.expectReceivedMsgs(msgs...)
	},
		Entry("text message", []string{
			"some-test-message",
		}),
		Entry("binary message", []string{
			"\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98",
		}),
		Entry("multiple messages", []string{
			"some-test-message-1",
			"\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98",
			"some-test-message-2",
			"some-test-\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98-message-3",
		}),
	)
})

func writeLog(msgs []string) (string, func()) {
	tmpDir, err := ioutil.TempDir("/tmp", "")
	Expect(err).ToNot(HaveOccurred())

	f, err := ioutil.TempFile(tmpDir, "")
	Expect(err).ToNot(HaveOccurred())
	defer f.Close()

	for _, msg := range msgs {
		n, err := f.Write([]byte(msg + "\n"))
		Expect(err).ToNot(HaveOccurred())
		if n != len(msg)+1 {
			Fail("unable to write log to temp file")
		}
	}

	return f.Name(), func() {
		err := os.RemoveAll(tmpDir)
		Expect(err).ToNot(HaveOccurred())
	}
}

type spyDrain struct {
	lis net.Listener
}

func newSpyDrain(addr ...string) *spyDrain {
	a := "localhost:0"
	if len(addr) != 0 {
		a = addr[0]
	}
	lis, err := net.Listen("tcp", a)
	Expect(err).ToNot(HaveOccurred())

	return &spyDrain{
		lis: lis,
	}
}

func (s *spyDrain) url() string {
	if runtime.GOOS == "darwin" {
		_, port, err := net.SplitHostPort(s.lis.Addr().String())
		Expect(err).ToNot(HaveOccurred())
		return "host.docker.internal:" + port
	}
	return s.lis.Addr().String()
}

func (s *spyDrain) stop() {
	s.lis.Close()
}

func (s *spyDrain) accept() net.Conn {
	conn, err := s.lis.Accept()
	Expect(err).ToNot(HaveOccurred())
	return conn
}

func (s *spyDrain) expectReceivedMsgs(msgs ...string) {
	conn := s.accept()
	defer conn.Close()
	buf := bufio.NewReader(conn)

	for _, expected := range msgs {
		actual, err := buf.ReadString('\n')
		l := strings.Split(actual, " ")
		actual = strings.Join(l[3:], " ")
		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(`- - - - - ` + expected + "\n"))
	}
}
