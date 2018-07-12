package syslog_test

import (
	"bufio"
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
)

var _ = Describe("Out", func() {
	It("writes messages via syslog", func() {
		spyDrain := newSpyDrain()
		defer spyDrain.stop()

		out := syslog.NewOut(spyDrain.url())

		record := make(map[interface{}]interface{})
		record["log"] = "some-log-message"
		err := out.Write(record, time.Unix(0, 0).UTC(), "")
		Expect(err).ToNot(HaveOccurred())

		spyDrain.expectReceived(
			`59 <14>1 1970-01-01T00:00:00+00:00 - - - - - some-log-message` + "\n",
		)
	})

	It("writes kubernetes metadata as syslog structured data to message", func(){
		spyDrain := newSpyDrain()
		defer spyDrain.stop()

		out := syslog.NewOut(spyDrain.url())
		record := make(map[interface{}]interface{})
		record["log"] = "2018-07-09 05:17:23.054078 I | etcdmain: Git SHA: 918698add"
		record["stream"] = "stderr"
		record["time"] = "2018-07-09T05:17:23.054249066Z"
		record["kubernetes"] = map[string]string{"pod_name":"etcd-minikube", "namespace_name":"kube-system"}
		record["host"] = "minikube"
		record["container_name"] = "etcd"
		record["docker_id"] = "3d6e6ca31dda9714588d6ae856b1c90b28f9c461c1f3c2b15c631ca4a89f561c"

		err := out.Write(record, time.Unix(0, 0).UTC(), "")
		Expect(err).ToNot(HaveOccurred())

		spyDrain.expectReceived(
			`150 <14>1 1970-01-01T00:00:00+00:00 minikube etcd - - - Namespace: kube-system | Pod Name:  | 2018-07-09 05:17:23.054078 I | etcdmain: Git SHA: 918698add` + "\n",
		)
	})

	It("returns an error when unable to write the message", func() {
		spyDrain := newSpyDrain()
		out := syslog.NewOut(spyDrain.url())
		spyDrain.stop()

		record1 := make(map[interface{}]interface{})
		record1[""] = ""
		err := out.Write(record1, time.Unix(0, 0).UTC(), "")
		Expect(err).To(HaveOccurred())

		record2 := make(map[interface{}]interface{})
		record2["log"] = "2018-07-09 05:17:23.054078 I | etcdmain: Git SHA: 918698add"
		record2["stream"] = "stderr"
		record2["time"] = "2018-07-09T05:17:23.054249066Z"
		record2["kubernetes"] = map[string]int{"namespace_name":1234}
		record2["host"] = "minikube"
		record2["container_name"] = "etcd"
		record2["docker_id"] = "3d6e6ca31dda9714588d6ae856b1c90b28f9c461c1f3c2b15c631ca4a89f561c"

		err = out.Write(record2, time.Unix(0, 0).UTC(), "")
		Expect(err).To(HaveOccurred())
	})

	It("eventually connects to a failing syslog drain", func() {
		spyDrain := newSpyDrain()
		spyDrain.stop()
		out := syslog.NewOut(spyDrain.url())

		spyDrain = newSpyDrain(spyDrain.url())

		record := make(map[interface{}]interface{})
		record["log"] = "some-log-message"

		err := out.Write(record, time.Unix(0, 0).UTC(), "")
		Expect(err).ToNot(HaveOccurred())

		spyDrain.expectReceived(
			`59 <14>1 1970-01-01T00:00:00+00:00 - - - - - some-log-message` + "\n",
		)
	})

	It("doesn't reconnect if connection already established", func() {
		spyDrain := newSpyDrain()
		defer spyDrain.stop()
		out := syslog.NewOut(spyDrain.url())

		record := make(map[interface{}]interface{})
		record["log"] = "some-log-message"

		err := out.Write(record, time.Unix(0, 0).UTC(), "")
		Expect(err).ToNot(HaveOccurred())

		spyDrain.expectReceived(
			`59 <14>1 1970-01-01T00:00:00+00:00 - - - - - some-log-message` + "\n",
		)

		err = out.Write(record, time.Unix(0, 0).UTC(), "")
		Expect(err).ToNot(HaveOccurred())

		done := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(done)
			_, _ = spyDrain.lis.Accept()
		}()
		Consistently(done).ShouldNot(BeClosed())
	})

	It("reconnects if previous connection went away", func() {
		spyDrain := newSpyDrain()
		out := syslog.NewOut(spyDrain.url())
		record1 := make(map[interface{}]interface{})
		record1["log"] = "some-log-message-1"

		err := out.Write(record1, time.Unix(0, 0).UTC(), "")
		Expect(err).ToNot(HaveOccurred())
		spyDrain.expectReceived(
			`61 <14>1 1970-01-01T00:00:00+00:00 - - - - - some-log-message-1` + "\n",
		)

		spyDrain.stop()
		spyDrain = newSpyDrain(spyDrain.url())

		record2 := make(map[interface{}]interface{})
		record2["log"] = "some-log-message-2"

		f := func() error {
			return out.Write(record2, time.Unix(0, 0).UTC(), "")
		}
		Eventually(f).Should(HaveOccurred())

		err = out.Write(record2, time.Unix(0, 0).UTC(), "")
		Expect(err).ToNot(HaveOccurred())

		spyDrain.expectReceived(
			`61 <14>1 1970-01-01T00:00:00+00:00 - - - - - some-log-message-2` + "\n",
		)
	})
})

type spyDrain struct {
	lis net.Listener
}

func newSpyDrain(addr ...string) *spyDrain {
	a := ":0"
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
	return s.lis.Addr().String()
}

func (s *spyDrain) stop() {
	_ = s.lis.Close()
}

func (s *spyDrain) accept() net.Conn {
	conn, err := s.lis.Accept()
	Expect(err).ToNot(HaveOccurred())
	return conn
}

func (s *spyDrain) expectReceived(msgs ...string) {
	conn := s.accept()
	defer func() {
		_ = conn.Close()
	}()
	buf := bufio.NewReader(conn)

	for _, expected := range msgs {
		actual, err := buf.ReadString('\n')
		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(expected))
	}
}
