#!/bin/sh

echo "This is EPaxos docker on $(hostname), version $(cat VERSION)"

uname -a
cat /etc/*-release

env | grep -v '^EPAXOS_'

if [ -z "$EPAXOS_REPLICA_ID" ]; then
	export EPAXOS_REPLICA_ID="${HOSTNAME##*-}"
fi

env | grep '^EPAXOS_'

if [ "$#" -eq 0 ]; then
	echo "Error: no cmd found"
	exit 127
fi

echo "Launching" "$@" "..."

exec "$@"
