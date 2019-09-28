package main

import (
	"C"
	"encoding/json"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
)

//export FLBPluginRegister
func FLBPluginRegister(def unsafe.Pointer) int {
	return output.FLBPluginRegister(
		def,
		"syslog",
		"syslog output plugin that follows RFC 5424",
	)
}

//export FLBPluginInit
func FLBPluginInit(plugin unsafe.Pointer) int {
	addr := output.FLBPluginConfigKey(plugin, "addr")
	name := output.FLBPluginConfigKey(plugin, "instancename")
	namespace := output.FLBPluginConfigKey(plugin, "namespace")
	cluster := output.FLBPluginConfigKey(plugin, "cluster")
	tls := output.FLBPluginConfigKey(plugin, "tlsconfig")
	sanitizeHost := output.FLBPluginConfigKey(plugin, "sanitizehost")

	if addr == "" {
		log.Println("[out_syslog] ERROR: Addr is required")
		return output.FLB_ERROR
	}
	if name == "" {
		log.Println("[out_syslog] ERROR: InstanceName is required")
		return output.FLB_ERROR
	}

	var (
		sinks        []*syslog.Sink
		clusterSinks []*syslog.Sink
	)

	sink := &syslog.Sink{
		Addr:      addr,
		Name:      name,
		Namespace: namespace,
	}
	if tls != "" {
		var tlsConfig syslog.TLS
		err := json.Unmarshal([]byte(tls), &tlsConfig)
		if err != nil {
			log.Printf("[out_syslog] ERROR: Unable to unmarshal TLS config: %s", err)
			return output.FLB_ERROR
		}
		sink.TLS = &tlsConfig
	}
	if strings.ToLower(cluster) == "true" {
		clusterSinks = append(clusterSinks, sink)
	} else {
		sinks = append(sinks, sink)
	}

	// Defaults to true so that plugin conforms better with rfc5424#section-6.2.4
	sanitize := true
	if len(sanitizeHost) != 0 {
		var err error
		sanitize, err = strconv.ParseBool(sanitizeHost)
		if err != nil {
			log.Printf("[out_syslog] ERROR: Unable to parse SanitizeHost: %s", err)
			return output.FLB_ERROR
		}
	}
	out := syslog.NewOut(
		sinks,
		clusterSinks,
		syslog.WithSanitizeHost(sanitize),
	)

	// We are using runtime.KeepAlive to tell the Go Runtime to keep the
	// reference to this pointer because once it leaves this context and
	// enters cgo it will no longer be in scope of Go. If a GC event occurs
	// the memory is reclaimed.
	// NOTE 1: Yes we are passing the `out` pointer even though it points to a
	// struct that contains other Go pointers and this violates the rules as
	// defined here: https://golang.org/cmd/cgo/#hdr-Passing_pointers
	// > Go code may pass a Go pointer to C provided the Go memory to which it
	//   points does not contain any Go pointers.
	// But this seems to be the most stable solution even when comparing the
	// instance slice/index solution.
	// NOTE 2: Since we are asking the Go Runtime to not clean this memory
	// up, it can be a cause for a "memory leak" however we are not planning
	// on millions of sinks to be initialized.
	output.FLBPluginSetContext(plugin, unsafe.Pointer(out))
	runtime.KeepAlive(out)
	if strings.ToLower(cluster) == "true" {
		log.Printf("[out_syslog] Initializing plugin %s for cluster to destination %s", name, addr)
	} else {
		log.Printf("[out_syslog] Initializing plugin %s for namespace %s to destination %s", name, namespace, addr)
	}
	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	var (
		ret    int
		ts     interface{}
		record map[interface{}]interface{}
	)

	out := (*syslog.Out)(ctx)

	dec := output.NewDecoder(data, int(length))
	for {
		ret, ts, record = output.GetRecord(dec)
		if ret != 0 {
			break
		}

		var timestamp time.Time
		switch tts := ts.(type) {
		case output.FLBTime:
			timestamp = tts.Time
		case uint64:
			// From our observation, when ts is of type uint64 it appears to
			// be the amount of seconds since unix epoch.
			timestamp = time.Unix(int64(tts), 0)
		default:
			timestamp = time.Now()
		}

		out.Write(record, timestamp, C.GoString(tag))
	}

	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	// TODO: We should probably call conn.Close() for each sink connection
	return output.FLB_OK
}

func main() {
}
