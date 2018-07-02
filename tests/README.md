Test the functionality in docker-compose

# How to test

Install docker-compose locally

```
cd out_syslog/tests

# build docker image, if you have code changes
docker-compose build

# start process
docker-compose up -d

docker-compose ps
```

In the container of syslog-server, we're supposed to see some log

```
docker logs syslog-server
```
=======
# How To Test

```
# get the code
cd out_syslog/tests

# run test script
./test.sh
```
