package syslog

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
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

// SinkError represents sink error message and timestamp.
type SinkError struct {
	Msg       string    `json:"msg"`
	Timestamp time.Time `json:"timestamp"`
}

// SinkState represents sink successful/error state.
type SinkState struct {
	Name               string     `json:"name"`
	Namespace          string     `json:"namespace"`
	LastSuccessfulSend time.Time  `json:"last_successful_send"`
	Error              *SinkError `json:"error"`
}

// Sink represents sink information.
type Sink struct {
	Addr      string `json:"addr"`
	Namespace string `json:"namespace"`
	TLS       *TLS   `json:"tls"`
	Name      string `json:"name"`

	messages chan io.WriterTo

	messagesDropped      int64
	lastSendSuccessNanos int64
	lastSendAttemptNanos int64
	writeErr             atomic.Value

	conn               net.Conn
	writeTimeout       time.Duration
	maintainConnection func() error
}

// TLS represents sink TLS configuration.
type TLS struct {
	InsecureSkipVerify bool `json:"insecure_skip_verify"`
}

// Out writes fluentbit messages via syslog TCP (RFC 5424 and RFC 6587).
type Out struct {
	sinks        map[string][]*Sink
	clusterSinks []*Sink
	dialTimeout  time.Duration
	bufferSize   int
	writeTimeout time.Duration
}

type OutOption func(*Out)

func WithDialTimeout(d time.Duration) OutOption {
	return func(o *Out) {
		o.dialTimeout = d
	}
}

func WithBufferSize(s int) OutOption {
	return func(o *Out) {
		o.bufferSize = s
	}
}

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

// SinkState reports all sinks successful/error state.
func (o *Out) SinkState() []SinkState {
	var stats []SinkState
	for _, sinks := range o.sinks {
		for _, s := range sinks {
			stats = append(stats, SinkState{
				Name:               s.Name,
				Namespace:          s.Namespace,
				LastSuccessfulSend: time.Unix(0, atomic.LoadInt64(&s.lastSendSuccessNanos)),
				Error:              s.loadSinkError(),
			})
		}
	}

	for _, s := range o.clusterSinks {
		stats = append(stats, SinkState{
			Name:               s.Name,
			LastSuccessfulSend: time.Unix(0, atomic.LoadInt64(&s.lastSendSuccessNanos)),
			Error:              s.loadSinkError(),
		})
	}

	return stats
}

func (s *Sink) loadSinkError() *SinkError {
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

// MessagesDropped reports number of messages be dropped.
func (s *Sink) MessagesDropped() int64 {
	return atomic.LoadInt64(&s.messagesDropped)
}

func tlsMaintainConn(s *Sink, out *Out) func() error {
	return func() error {
		if s.conn == nil {
			dialer := net.Dialer{
				Timeout: out.dialTimeout,
			}
			var conn net.Conn // conn needs to be of type net.Conn, not *tls.Conn
			conn, err := tls.DialWithDialer(
				&dialer,
				"tcp",
				s.Addr,
				&tls.Config{
					InsecureSkipVerify: s.TLS.InsecureSkipVerify,
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
		logmsg                     []byte
		host, appName, pid, nsName string
		facility, severity         int
		priority                   rfc5424.Priority
		k8sMap                     map[interface{}]interface{}
	)

	for k, v := range record {
		key, ok := k.(string)
		if !ok {
			continue
		}

		switch key {
		case "MESSAGE", "log":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			logmsg = v2
		case "_HOSTNAME", "cluster_name":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			host = string(v2)
		case "_COMM":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			appName = string(v2)
		case "SYSLOG_IDENTIFIER":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			// Get appname from COMM first.
			// Ref: https://github.com/aiven/journalpump/blob/b299ddac373b25510106a1d771b9b7c8c233f1e1/journalpump/journalpump.py#L350-L356
			if appName == "" {
				appName = string(v2)
			}
		case "_PID":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			pid = string(v2)
		case "SYSLOG_FACILITY":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			facility, _ = strconv.Atoi(string(v2))
		case "PRIORITY":
			v2, ok2 := v.([]byte)
			if !ok2 {
				continue
			}
			severity, _ = strconv.Atoi(string(v2))
		case "kubernetes":
			v2, ok2 := v.(map[interface{}]interface{})
			if !ok2 {
				continue
			}
			k8sMap = v2
		default:
			// unsupported key
		}
	}

	// Priority = Facility x 8 + Severity
	priority = rfc5424.Priority(facility<<3 + severity)
	if priority == 0 {
		priority = rfc5424.User + rfc5424.Info
	}

	rfc5424Msg := rfc5424.Message{
		Priority:  priority,
		Timestamp: ts,
		Hostname:  host,
		AppName:   appName,
		ProcessID: pid,
		MessageID: "-",
		Message:   logmsg,
	}

	if len(k8sMap) > 0 {
		var (
			podName, containerName, vmID string
			labelParams                  []rfc5424.SDParam
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
				nsName = string(v2)
			case "labels":
				v2, ok2 := v.(map[interface{}]interface{})
				if !ok2 {
					continue
				}
				labelParams = processLabels(v2)
			default:
				// unsupported key
			}
		}

		prefix := logPrefix
		if strings.HasPrefix(tag, eventPrefix) {
			prefix = eventPrefix
		}
		appName = fmt.Sprintf(
			"%s/%s/%s/%s",
			prefix,
			nsName,
			podName,
			containerName,
		)
		// APP-NAME is limited to 48 chars in RFC 5424
		// https://tools.ietf.org/html/rfc5424#section-6
		if len(appName) > 48 {
			appName = appName[:48]
		}

		// Updates appname and hostname.
		rfc5424Msg.AppName = appName

		if rfc5424Msg.Hostname == "" {
			rfc5424Msg.Hostname = vmID
		}

		// Add to structured data.
		for _, label := range labelParams {
			rfc5424Msg.AddDatum("kubernetes", label.Name, label.Value)
		}
		rfc5424Msg.AddDatum("kubernetes", "namespace_name", nsName)
		rfc5424Msg.AddDatum("kubernetes", "pod_name", podName)
		rfc5424Msg.AddDatum("kubernetes", "container_name", containerName)
		if vmID != "" {
			rfc5424Msg.AddDatum("kubernetes", "vm_id", vmID)
		}
	}

	return &rfc5424Msg, nsName
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
