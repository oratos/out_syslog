
## Fluent Bit Syslog Output Plugin

**How to Test:**

```
cd $GOPATH

# get the code
mkdir -p src/github.com/pivotal-cf
cd src/github.com/pivotal-cf
git clone git@github.com:pivotal-cf/fluent-bit-out-syslog.git

# get dependencies
cd fluent-bit-out-syslog/cmd
go get -d -t ./ ...

# run code build
go build -buildmode c-shared -o out_syslog.so github.com/pivotal-cf/fluent-bit-out-syslog/cmd

# run test
go test -v
```

**How to Run:**

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
