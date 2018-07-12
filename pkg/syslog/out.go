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
	var logmsg []byte
	var hostname, appname string
	var podname, containername string
	var k8sMap map[string]string

	// TODO: avoid unstable assumption for the data type
	for k, v := range record {
		key, ok := k.(string)
		if !ok { continue }

		switch key {
		case "log":
			v2, ok2 := v.([]byte)
			if !ok2 { continue }
			logmsg = v2
		case "host":
			v2, ok2 := v.(string)
			if !ok2 { continue }
			hostname = v2
		case "container_name":
			v2, ok2 := v.(string)
			if !ok2 { continue }
			containername = v2
		case "pod_name":
			v2, ok2 := v.(string)
			if !ok2 { continue }
			podname = v2
		case "kubernetes":
			v2, ok2 := v.(map[string]string)
			if !ok2 { continue }
			k8sMap = v2
		}
	}
	if len(k8sMap) != 0 {
		// sample: kube-system/pod/kube-dns-86f4d74b45-lfgj7/dnsmasq
		appname = fmt.Sprintf("%s/%s/%s/%s", k8sMap["namespace_name"], "pod", podname, containername)
	}

	if !bytes.HasSuffix(logmsg, []byte("\n")) {
		logmsg = append(logmsg, byte('\n'))
	}

	return &rfc5424.Message{
		Priority:  rfc5424.Info + rfc5424.User,
		Timestamp: ts,
		Hostname:  hostname,
		AppName:   appname,
		Message:   logmsg,
	}
}
