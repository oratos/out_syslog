# How to run function test via docker-compose

Install docker-compose locally

```
cd out_syslog/tests

# build code by spining up a temporary docker container
build_code.sh

# start process
docker-compose up -d

docker-compose ps
```

In the container of syslog-server, we're supposed to see some log

```
docker logs syslog-server
```

# How to run lint check

```
# get the code
cd out_syslog/tests

# run test script
./test.sh
```
