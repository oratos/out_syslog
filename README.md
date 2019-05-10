# Fluent Bit Syslog Output Plugin

   * [How to Configure Fluent Bit Conf](#how-to-configure-fluent-bit-conf)
   * [Sample Config File](#sample-config-file)
   * [Development](#development)

The Fluent Bit Syslog Output plugin translates kubernetes cluster logs
into [RFC5424][rfc5424] syslog messages. It uses [Cloud Foundry's RFC5424
library][cfrfc5424].

## How to Configure Fluent Bit Conf

Below is a sample fluent bit configuration file.
`Sinks` are also referred to as namespace sinks and forward logs from
a particular namespace onto a syslog destination. Whereas, `ClusterSinks`
forward all logs from all namespaces to the specified syslog destination.

The `tls` configuration is optional and is required only if connecting to
an endpoint that supports TLS.


## Sample Config File

 **Syslog output plugin with kubernetes namespace filter**

```ini
[INPUT]
    Name              tail
    Tag               kube.*
    Path              /var/log/containers/*.log
    Parser            docker
    DB                /var/log/flb_kube.db
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Refresh_Interval  10

[FILTER]
    Name                kubernetes
    Match               kube.*
    Kube_URL            https://kubernetes.default.svc.cluster.local:443
    Merge_Log           On
    K8S-Logging.Parser  On

[OUTPUT]
    Name          syslog
    InstanceName  insecure-namespace-sink
    Match         *
    Addr          logs.papertrailapp.com:18271
    Namespace     myns
    TLSConfig     {"insecure_skip_verify":"true"}

[OUTPUT]
    Name          syslog
    InstanceName  plaintext-cluster-sink
    Match         *
    Addr          logs.papertrailapp.com:18271
    Cluster       true
    TLSConfig     {"root_ca":"/path/to/root/ca"}
```


## Development
### How to Test and Build the plugin:

```
# get the code
cd workspace
git clone git@github.com:pivotal-cf/fluent-bit-out-syslog.git
cd fluent-bit-out-syslog

# run the linter
./tests/run-linter.sh

# run test
go test -v ./...

# build the plugin
go build -mod vendor -buildmode c-shared -o out_syslog.so cmd/main.go
```

### How To Run In Local laptop

```
fluent-bit \
    --input dummy \
    --plugin ./out_syslog.so \
    --output syslog \
    --prop InstanceName='testing' \
    --prop Addr='localhost:12345' \
    --prop Cluster='true'
```

[rfc5424]:   https://tools.ietf.org/html/rfc5424
[cfrfc5424]: https://github.com/cloudfoundry-incubator/rfc5424
