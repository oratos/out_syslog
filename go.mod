module github.com/pivotal-cf/fluent-bit-out-syslog

require (
	code.cloudfoundry.org/rfc5424 v0.0.0-20180905210152-236a6d29298a
	github.com/fluent/fluent-bit-go v0.0.0-20171103221316-c4a158a6e3a7
	github.com/fsnotify/fsnotify v1.4.7 // indirect
	github.com/golang/protobuf v1.2.0 // indirect
	github.com/hpcloud/tail v1.0.0 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/onsi/ginkgo v1.6.0
	github.com/onsi/gomega v1.4.1
	golang.org/x/net v0.0.0-20180906233101-161cd47e91fd // indirect
	golang.org/x/sync v0.0.0-20180314180146-1d60e4601c6f // indirect
	golang.org/x/sys v0.0.0-20180909124046-d0be0721c37e // indirect
	golang.org/x/text v0.3.0 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.2.1 // indirect
)

replace github.com/fluent/fluent-bit-go => github.com/wfernandes/fluent-bit-go v0.0.0-20190416184736-06ac16c1ccf5

go 1.13
