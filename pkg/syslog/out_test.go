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
		record := map[string]string{
			"log": "some-log-message",
		}
		err := out.Write(record, time.Unix(0, 0).UTC(), "")
		Expect(err).ToNot(HaveOccurred())

		spyDrain.expectReceived(
			`59 <14>1 1970-01-01T00:00:00+00:00 - - - - - some-log-message` + "\n",
		)
	})

	It("returns an error when unable to write the message", func() {
		spyDrain := newSpyDrain()
		out := syslog.NewOut(spyDrain.url())
		spyDrain.stop()

		err := out.Write(map[string]string{}, time.Unix(0, 0).UTC(), "")

		Expect(err).To(HaveOccurred())
	})

	It("eventually connects to a failing syslog drain", func() {
		spyDrain := newSpyDrain()
		spyDrain.stop()
		out := syslog.NewOut(spyDrain.url())

		spyDrain = newSpyDrain(spyDrain.url())

		record := map[string]string{
			"log": "some-log-message",
		}

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
		record := map[string]string{
			"log": "some-log-message",
		}

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
		record1 := map[string]string{
			"log": "some-log-message-1",
		}
		err := out.Write(record1, time.Unix(0, 0).UTC(), "")
		Expect(err).ToNot(HaveOccurred())
		spyDrain.expectReceived(
			`61 <14>1 1970-01-01T00:00:00+00:00 - - - - - some-log-message-1` + "\n",
		)

		spyDrain.stop()
		spyDrain = newSpyDrain(spyDrain.url())

		record2 := map[string]string{
			"log": "some-log-message-2",
		}

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
