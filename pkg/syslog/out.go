package syslog

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/rfc5424"
)

// TODO: Address issues where messages are malformed but we are not notifying
// the user.

const (
	eventPrefix = "k8s.event"
	logPrefix   = "pod.log"
)

type SinkError struct {
	Msg       string    `json:"msg"`
	Timestamp time.Time `json:"timestamp"`
}

type SinkState struct {
	Name               string     `json:"name"`
	Namespace          string     `json:"namespace"`
	LastSuccessfulSend time.Time  `json:"last_successful_send"`
	Error              *SinkError `json:"error"`
}

type Sink struct {
	Addr      string
	Name      string
	Namespace string
	TLS       *TLS

	messages chan io.WriterTo

	messagesDropped      int64
	lastSendSuccessNanos int64
	lastSendAttemptNanos int64
	writeErr             atomic.Value

	conn               net.Conn
	writeTimeout       time.Duration
	maintainConnection func() error
}

type TLS struct {
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	RootCA             string `json:"root_ca"`
}

// Out writes fluentbit messages via syslog TCP (RFC 5424 and RFC 6587).
type Out struct {
	sinks        map[string][]*Sink
	clusterSinks []*Sink
	dialTimeout  time.Duration
	bufferSize   int
	writeTimeout time.Duration
}

// OutOption is the optional setting of write output.
type OutOption func(*Out)

// WithDialTimeout configures dial up timeout.
func WithDialTimeout(d time.Duration) OutOption {
	return func(o *Out) {
		o.dialTimeout = d
	}
}

// WithBufferSize configures write buffer size.
func WithBufferSize(s int) OutOption {
	return func(o *Out) {
		o.bufferSize = s
	}
}

// WithWriteTimeout configures write timeout.
func WithWriteTimeout(t time.Duration) OutOption {
	return func(o *Out) {
		o.writeTimeout = t
	}
}

// NewOut returns a new Out which handles both tcp and tls connections.
func NewOut(sinks, clusterSinks []*Sink, opts ...OutOption) *Out {
	out := &Out{
		dialTimeout:  5 * time.Second,
		bufferSize:   10000,
		writeTimeout: time.Second,
	}

	for _, o := range opts {
		o(out)
	}

	m := make(map[string][]*Sink)
	for _, s := range sinks {
		if s.TLS != nil {
			s.maintainConnection = tlsMaintainConn(s, out)
		} else {
			s.maintainConnection = tcpMaintainConn(s, out)
		}

		m[s.Namespace] = append(m[s.Namespace], s)
		s.writeTimeout = out.writeTimeout
		s.start(out.bufferSize)
	}
	for _, s := range clusterSinks {
		if s.TLS != nil {
			s.maintainConnection = tlsMaintainConn(s, out)
		} else {
			s.maintainConnection = tcpMaintainConn(s, out)
		}
		s.writeTimeout = out.writeTimeout
		s.start(out.bufferSize)
	}
	out.sinks = m
	out.clusterSinks = clusterSinks
	return out
}

// Write takes a record, timestamp, and tag, converts it into a syslog message
// and routes it to the connections with the matching namespace.
// Each sink has it's own backing network connection and queue. The queue's
// size is fixed to 10000 messages. It will report dropped messages via a log
// for every 1000 messages dropped.
// If no connection is established one will be established per sink upon a
// Write operation. Write will also write all messages to all cluster sinks
// provided.
func (o *Out) Write(
	record map[interface{}]interface{},
	ts time.Time,
	tag string,
) {
	msg, namespace := convert(record, ts, tag)

	for _, cs := range o.clusterSinks {
		cs.queueMessage(msg)
	}

	namespaceSinks, ok := o.sinks[namespace]
	if !ok {
		// TODO: track ignored messages
		return
	}

	for _, s := range namespaceSinks {
		s.queueMessage(msg)
	}
}

func (o *Out) SinkState() []SinkState {
	var stats []SinkState
	for _, sinks := range o.sinks {
		for _, s := range sinks {
			stats = append(stats, SinkState{
				Name:               s.Name,
				Namespace:          s.Namespace,
				LastSuccessfulSend: time.Unix(0, atomic.LoadInt64(&s.lastSendSuccessNanos)),
				Error:              s.LoadSinkError(),
			})
		}
	}

	for _, s := range o.clusterSinks {
		stats = append(stats, SinkState{
			Name:               s.Name,
			LastSuccessfulSend: time.Unix(0, atomic.LoadInt64(&s.lastSendSuccessNanos)),
			Error:              s.LoadSinkError(),
		})
	}

	return stats
}

func (s *Sink) LoadSinkError() *SinkError {
	if sinkError, ok := s.writeErr.Load().(SinkError); ok && sinkError.Msg != "" {
		return &sinkError
	}
	return nil
}

func (s *Sink) start(bufferSize int) {
	s.messages = make(chan io.WriterTo, bufferSize)
	go func() {
		for m := range s.messages {
			s.write(m)
		}
	}()
}

func (s *Sink) queueMessage(msg io.WriterTo) {
	select {
	case s.messages <- msg:
	default:
		md := atomic.AddInt64(&s.messagesDropped, 1)
		if md%1000 == 0 && md != 0 {
			log.Printf("Sink to address %s, at namespace [%s] dropped %d messages\n", s.Addr, s.Namespace, md)
		}
	}
}

// write writes a rfc5424 syslog message to the connection of the specified
// sink. It recreates the connection if one isn't established yet.
func (s *Sink) write(w io.WriterTo) {
	defer atomic.StoreInt64(&s.lastSendAttemptNanos, time.Now().UnixNano())

	err := s.maintainConnection()
	if err != nil {
		atomic.AddInt64(&s.messagesDropped, 1)
		s.writeErr.Store(SinkError{
			Msg:       err.Error(),
			Timestamp: time.Now(),
		})
		return
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
	_, err = w.WriteTo(s.conn)
	if err != nil {
		s.conn.Close()
		s.conn = nil
		atomic.AddInt64(&s.messagesDropped, 1)
		s.writeErr.Store(SinkError{
			Msg:       err.Error(),
			Timestamp: time.Now(),
		})
		return
	}
	s.writeErr.Store(SinkError{})
	atomic.StoreInt64(&s.lastSendSuccessNanos, time.Now().UnixNano())
}

func (s *Sink) MessagesDropped() int64 {
	return atomic.LoadInt64(&s.messagesDropped)
}

func tlsMaintainConn(s *Sink, out *Out) func() error {
	return func() error {
		if s.conn == nil {
			var (
				conn  net.Conn // conn needs to be of type net.Conn, not *tls.Conn
				roots *x509.CertPool
				pem   []byte
				err   error
			)

			if !s.TLS.InsecureSkipVerify && s.TLS.RootCA != "" {
				roots = x509.NewCertPool()

				pem, err = ioutil.ReadFile(s.TLS.RootCA)
				if err != nil {
					return err
				}

				if ok := roots.AppendCertsFromPEM(pem); !ok {
					return fmt.Errorf("append certificate failed")
				}
			}

			conn, err = tls.DialWithDialer(
				&net.Dialer{
					Timeout: out.dialTimeout,
				},
				"tcp",
				s.Addr,
				&tls.Config{
					InsecureSkipVerify: s.TLS.InsecureSkipVerify,
					RootCAs:            roots,
				},
			)

			if err == nil {
				s.conn = conn
			}
			return err
		}
		return nil
	}
}

func tcpMaintainConn(s *Sink, out *Out) func() error {
	return func() error {
		if s.conn == nil {
			dialer := net.Dialer{
				Timeout: out.dialTimeout,
			}
			conn, err := dialer.Dial("tcp", s.Addr)
			if err == nil {
				s.conn = conn
			}
			return err
		}
		return nil
	}
}

func convert(
	record map[interface{}]interface{},
	ts time.Time,
	tag string,
) (*rfc5424.Message, string) {
	var (
		logmsg []byte
		k8sMap map[interface{}]interface{}
		host   string
	)

	for k, v := range record {
		key, ok := k.(string)
		if !ok {
			continue
		}

		switch key {
		case "log":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			logmsg = v2
		case "kubernetes":
			v2, ok2 := v.(map[interface{}]interface{})
			if !ok2 {
				continue
			}
			k8sMap = v2
		case "cluster_name":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			host = string(v2)
		}
	}

	var (
		vmID          string
		appName       string
		podName       string
		namespaceName string
		containerName string
		labelParams   []rfc5424.SDParam
	)
	for k, v := range k8sMap {
		key, ok := k.(string)
		if !ok {
			continue
		}

		switch key {
		case "host":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			vmID = string(v2)
		case "container_name":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			containerName = string(v2)
		case "pod_name":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			podName = string(v2)
		case "namespace_name":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			namespaceName = string(v2)
		case "labels":
			v2, ok2 := v.(map[interface{}]interface{})
			if !ok2 {
				continue
			}
			labelParams = processLabels(v2)
		}
	}

	k8sStructuredData := buildStructuredData(
		labelParams,
		namespaceName,
		podName,
		containerName,
		vmID,
	)

	if len(k8sMap) != 0 {
		prefix := logPrefix
		if strings.HasPrefix(tag, eventPrefix) {
			prefix = eventPrefix
		}
		appName = fmt.Sprintf(
			"%s/%s/%s/%s",
			prefix,
			namespaceName,
			podName,
			containerName,
		)
		// APP-NAME is limited to 48 chars in RFC 5424
		// https://tools.ietf.org/html/rfc5424#section-6
		if len(appName) > 48 {
			appName = appName[:48]
		}
	}

	if !bytes.HasSuffix(logmsg, []byte("\n")) {
		logmsg = append(logmsg, byte('\n'))
	}

	if host == "" {
		host = vmID
	}

	return &rfc5424.Message{
		Priority:  rfc5424.Info + rfc5424.User,
		Timestamp: ts,
		Hostname:  host,
		AppName:   appName,
		Message:   logmsg,
		StructuredData: []rfc5424.StructuredData{
			k8sStructuredData,
		},
	}, namespaceName
}

func processLabels(labels map[interface{}]interface{}) []rfc5424.SDParam {
	params := make([]rfc5424.SDParam, 0, len(labels))
	for k, v := range labels {
		ks, ok := k.(string)
		if !ok {
			continue
		}
		vb, ok := v.([]byte)
		if !ok {
			continue
		}

		params = append(params, rfc5424.SDParam{
			Name:  ks,
			Value: string(vb),
		})
	}
	return params
}

func buildStructuredData(labels []rfc5424.SDParam, ns, pn, cn, vmID string) rfc5424.StructuredData {
	labels = append(
		labels,
		rfc5424.SDParam{
			Name:  "namespace_name",
			Value: ns,
		},
		rfc5424.SDParam{
			Name:  "object_name",
			Value: pn,
		},
		rfc5424.SDParam{
			Name:  "container_name",
			Value: cn,
		},
	)
	if vmID != "" {
		labels = append(
			labels,
			rfc5424.SDParam{
				Name:  "vm_id",
				Value: vmID,
			},
		)
	}

	return rfc5424.StructuredData{
		ID:         "kubernetes@47450",
		Parameters: labels,
	}
}
