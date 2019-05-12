#!/bin/bash

set -euxo pipefail

NS=${1:-5}   # Number of servers
NR=${2:-64}  # Number of records to test
NP=${3:-1}   # Client window
RK=${4:-5}   # Key collsion is 5%
D0=${5:-150} # Inter-group delay
JT=${6:-10}  # Std.Var. is 10% of mean

NK=$(($NR * 100 / $RK))

scripts/setup.sh "$NS" "$NR" "$NP" "$RK" "$D0" "$JT"

for I in $(seq 0 $(($NS-1))); do
    ./bin/client \
        -n "$NS" -t 30.0 --verbose \
        batch-put \
        -N "$NR" \
        --pipeline "$NP" \
        --latency --random-key \
        "$I" "$NK" \
        2>"data/client-$I.log" \
        | pv -s "$NR" -l \
        | sed "s/^/$NS,$NR,$NP,$RK,$D0,$JT,$I,/"
done

scripts/teardown.sh "$@"
