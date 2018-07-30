package syslog_test

import (
	"bufio"
	"crypto/tls"
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type spyDrain struct {
	lis net.Listener
}

func newTLSSpyDrain(addr ...string) *spyDrain {
	a := ":0"
	if len(addr) != 0 {
		a = addr[0]
	}

	cert, err := tls.X509KeyPair(tlsCert, tlsKey)
	Expect(err).ToNot(HaveOccurred())

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	lis, err := tls.Listen("tcp", a, config)
	Expect(err).ToNot(HaveOccurred())

	return &spyDrain{
		lis: lis,
	}
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

func (s *spyDrain) expectReceivedOnly(msgs ...string) {
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

	read := make(chan struct{})
	go func() {
		defer close(read)
		_, _ = buf.ReadString('\n')
	}()
	select {
	case <-read:
		Fail("unexpected read occurred")
	case <-time.After(300 * time.Millisecond):
	}
}
