FILES=$(shell find src/ -type f -name '*')
LDFLAGS=-X main.VERSION=$$(cat VERSION)

.PHONY: all debug dist FORCE docker clean dist-clean data-clean fmt

all: bin/server bin/client

bin/server: $(FILES) VERSION
	go install -ldflags "$(LDFLAGS)" epaxos/server

bin/client: $(FILES) VERSION
	go install -ldflags "$(LDFLAGS)" epaxos/client

debug: all
	cp Dockerfile.debug bin/Dockerfile
	docker build -t b1f6c1c4/epaxos:latest bin/

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

clean:
	rm -rf bin/

data-clean:
	rm -rf data/

fmt:
	go fmt epaxos/common epaxos/server epaxos/client
