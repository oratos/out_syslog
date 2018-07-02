Test the functionality in docker-compose

# How to test

Install docker-compose locally

```
cd out_syslog/tests
docker-compose up -d

docker-compose ps
```

In the container of syslog-server, we're supposed to see some log

```
docker logs syslog-server
```
