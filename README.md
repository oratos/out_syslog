
## Fluent Bit Syslog Output Plugin

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

**How to Run In Local laptop:**

```
fluent-bit \
    --input dummy \
    --plugin ./out_syslog.so \
    --output syslog \
    --prop Addr=localhost:12345
```

**Run Linter:**
```
./tests/run-linter.sh
```
