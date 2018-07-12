package syslog

import (
	"bytes"
	"net"
	"time"
	"fmt"

	"code.cloudfoundry.org/rfc5424"
)

// Out writes fluentbit messages via syslog TCP (RFC 5424 and RFC 6587).
type Out struct {
	addr string
	conn net.Conn
}

// NewOut creates a new
func NewOut(addr string) *Out {
	return &Out{
		addr: addr,
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
	err := o.maintainConnection()
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

func (o *Out) maintainConnection() error {
	if o.conn == nil {
		conn, err := net.Dial("tcp", o.addr)
		o.conn = conn
		return err
	}
	return nil
}

func convert(
	record map[interface{}]interface{},
	ts time.Time,
	tag string,
) *rfc5424.Message {
	var log_msg []byte
	var hostname, appname string
	var podname string
	var k8sMap map[string]string
	for k, v := range record {
		key, ok := k.(string)
		if !ok { continue }

		switch key {
		case "log":
			log_msg = v.([]byte)
		case "host":
			hostname = v.(string)
		case "container_name":
			appname = v.(string)
		case "pod_name":
			podname = v.(string)
		case "kubernetes":
			k8sMap = v.(map[string]string)
		}
	}
	if len(k8sMap) != 0 {
		log_msg = []byte(fmt.Sprintf("Namespace: %s | Pod Name: %s | %s", 
			k8sMap["namespace_name"], podname, string(log_msg)))
	}

	if !bytes.HasSuffix(log_msg, []byte("\n")) {
		log_msg = append(log_msg, byte('\n'))
	}

	return &rfc5424.Message{
		Priority:  rfc5424.Info + rfc5424.User,
		Timestamp: ts,
		Hostname:  hostname,
		AppName:   appname,
		Message:   log_msg,
	}
}
