FILES=$(shell find src/ -type f -name '*')

all: bin/server bin/client

bin/server: $(FILES)
	go install epaxos/server

bin/client: $(FILES)
	go install epaxos/client
