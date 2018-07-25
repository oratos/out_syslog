Table of Contents
=================

   * [Fluent Bit Syslog Output Plugin](#fluent-bit-syslog-output-plugin)
      * [How To Run In Local laptop](#how-to-run-in-local-laptop)
      * [How To Run In minikube](#how-to-run-in-minikube)
      * [How To Run Linter](#how-to-run-linter)

# Fluent Bit Syslog Output Plugin

**How to Test:**

```
cd $GOPATH

# get the code
mkdir -p src/github.com/pivotal-cf
cd src/github.com/pivotal-cf
git clone git@github.com:pivotal-cf/fluent-bit-out-syslog.git

# get dependencies
cd $GOPATH/src
go get -d -t github.com/pivotal-cf/fluent-bit-out-syslog/cmd...

# run code build
cd $GOPATH/src/github.com/pivotal-cf/fluent-bit-out-syslog/cmd
go build -buildmode c-shared -o out_syslog.so .

# run test
cd $GOPATH/src/github.com/pivotal-cf/fluent-bit-out-syslog/cmd
go test -v ./...
```

**How to Test in Docker-compose:**
```
cd $GOPATH/src/github.com/pivotal-cf/fluent-bit-out-syslog/
./tests/test.sh
```

## How To Run In Local laptop

```
fluent-bit \
    --input dummy \
    --plugin ./out_syslog.so \
    --output syslog \
    --prop Addr=localhost:12345
```

## How To Run In minikube

[tests/fluent-bit-kubernetes/README.md](tests/fluent-bit-kubernetes/README.md)

## How To Run Linter
```
./tests/run-linter.sh
```

## How to Configure In Fluent Bit Conf
Add the following output section to your Fluent Bit configuration file. Note
that the `EnableTLS` and `InsecureSkipVerify` configurations are optional and
only needed if you are connecting to an endpoint that supports TLS. By
default, both of thos optional fields are false.

```
[OUTPUT]
	Name syslog
	Match *
	Addr logs.papertrailapp.com:18271
	Enable_TLS  true
	Insecure_Skip_Verify  true
```
