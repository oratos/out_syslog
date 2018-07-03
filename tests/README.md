# Run test with one command
Make sure docker-compose has been installed in your env.

```
cd fluent-bit-out-syslog/tests
./test.sh
```

# Run test in a hard way

Since fluent-bit haven't been fully tested in mac, we start a container to build the go plugin.

Then mount the output file(out_syslog.so) to our docker-compose env for testing
```
cd out_syslog/tests

# build code by spining up a temporary docker container, which will generate out_syslog.so
./build-code.sh

# Docker-compose will mount out_syslog.so to fluent-bit container
# start process
docker-compose up -d

docker-compose ps
```

In the container of syslog-server, we're supposed to see some log

```
docker logs syslog-server
```
