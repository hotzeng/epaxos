# EPaxos

## Server side

### Run locally

```sh
source env.sh
make
./bin/server
```

### Run small-scale with docker-compose

```sh
source env.sh
make debug
./compose 3 up -d
```

### Run large-scale with kubernetes

Build the docker image:
```sh
source env.sh
make dist
```

Apply k8s config:
```sh
kubectl apply -f scripts/statefulset.yml
kubectl apply -f scripts/service.yml
```

## Client side

TODO
