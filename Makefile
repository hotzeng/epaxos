FILES=$(shell find src/ -type f -name '*')
LDFLAGS=-X main.VERSION=$$(cat VERSION)

all: bin/server bin/client

bin/server: $(FILES) VERSION
	go install -ldflags "$(LDFLAGS)" epaxos/server

bin/client: $(FILES) VERSION
	go install -ldflags "$(LDFLAGS)" epaxos/client

dist: VERSION
	docker build -t b1f6c1c4/epaxos:latest .
	docker push b1f6c1c4/epaxos:latest

VERSION: FORCE
	-./version.sh

FORCE:

docker:
	CGO_ENABLED=0 GOOS=linux go install \
        -installsuffix cgo \
        -ldflags "$(LDFLAGS)" \
        epaxos/server epaxos/client
