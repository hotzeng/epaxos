all: bin/server bin/client

bin/server: FORCE
	go install epaxos/server

bin/client: FORCE
	go install epaxos/client

FORCE:
