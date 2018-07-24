package syslog

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"code.cloudfoundry.org/rfc5424"
)

// TODO: Address issues where messages are malformed but we are not notifying
// the user.

// Out writes fluentbit messages via syslog TCP (RFC 5424 and RFC 6587).
type Out struct {
	addr             string
	conn             net.Conn
	maintainConnFunc func(*Out) error
	tlsConfig        *tls.Config
}

func NewTLSOut(addr string, skipInsecure bool, timeout time.Duration) *Out {
	config := &tls.Config{
		InsecureSkipVerify: skipInsecure,
	}

	return &Out{
		addr:      addr,
		tlsConfig: config,
		maintainConnFunc: func(self *Out) error {
			if self.conn == nil {
				dialer := net.Dialer{
					Timeout: timeout,
				}

				conn, err := tls.DialWithDialer(
					&dialer,
					"tcp",
					self.addr,
					self.tlsConfig,
				)

				self.conn = conn
				return err
			}
			return nil
		},
	}
}

// NewOut creates a new
func NewOut(addr string) *Out {
	return &Out{
		addr: addr,
		maintainConnFunc: func(self *Out) error {
			if self.conn == nil {
				conn, err := net.Dial("tcp", self.addr)
				self.conn = conn
				return err
			}
			return nil
		},
	}
}

// Write takes a record, timestamp, and tag and converts it into a syslog
// message and writes it out to the connection. If no connection is
// established one will be established.
func (o *Out) Write(
	record map[interface{}]interface{},
	ts time.Time,
	tag string,
) error {
	err := o.maintainConnFunc(o)
	if err != nil {
		return err
	}

	msg := convert(record, ts, tag)
	_, err = msg.WriteTo(o.conn)
	if err != nil {
		o.conn = nil
		return err
	}
	return nil
}

func convert(
	record map[interface{}]interface{},
	ts time.Time,
	tag string,
) *rfc5424.Message {
	var (
		logmsg []byte
		k8sMap map[interface{}]interface{}
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
		}
	}

	var (
		host          string
		appName       string
		podName       string
		namespaceName string
		containerName string
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
			host = string(v2)
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
		}
	}

	if len(k8sMap) != 0 {
		// sample: kube-system/pod/kube-dns-86f4d74b45-lfgj7/dnsmasq
		appName = fmt.Sprintf(
			"%s/%s/%s/%s",
			namespaceName,
			"pod",
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

	return &rfc5424.Message{
		Priority:  rfc5424.Info + rfc5424.User,
		Timestamp: ts,
		Hostname:  host,
		AppName:   appName,
		Message:   logmsg,
	}
}
