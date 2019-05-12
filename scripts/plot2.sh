#!/bin/bash

set -euxo pipefail

NS=${1:-5}    # Number of servers
NR=${2:-8192} # Number of records to test
NP=${3:-128}  # Client window
RK=${4:-5}    # Key collsion is 5%
D0=${5:-150}  # Inter-group delay
JT=${6:-10}   # Std.Var. is 10% of mean

NK=$(($NR * 100 / $RK))

scripts/setup.sh "$NS" "$NR" "$NP" "$RK" "$D0" "$JT"

(sleep 50 && \
	docker kill epaxos-server-2 && \
	sleep 30 &&
	docker start epaxos-server-2 \
	) >/dev/null &

for I in '4'; do
    ./bin/client \
        -n "$NS" -t 30.0 --verbose \
        batch-put \
        -N "$NR" \
        --pipeline "$NP" \
        --throughput 1000 --random-key \
        "$I" "$NK" \
        2>"data/client-$I.log" \
        | pv -l \
        | sed "s/^/$NS,$NR,$NP,$RK,$D0,$JT,$I,/"
done

scripts/teardown.sh "$@"
