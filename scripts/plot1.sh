#!/bin/bash

set -euxo pipefail

pv --version >&2

NS=${1:-5}   # Number of servers
NR=${2:-64}  # Number of records to test
NP=${3:-1}   # Client window
RK=${4:-5}   # Key collsion is 5%
D0=${5:-75}  # Inter-group delay
DD=${6:-10}  # Intra-group delay
JT=${7:-25}  # Std.Var. is 10% of mean

NK=$(($NR * 100 / $RK))

make -j2 debug >&2
make pumba-down >&2
if [ -f "./docker-compose.yml" ]; then
    docker-compose down
fi

sudo find ./data -type f -delete

./compose.sh "$NS" --prod up -d

G=$(($NS / 2))
IPS=()
for I in $(seq 0 $(($NS-1))); do
    IP="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "epaxos-server-$I")"
    IPS+=("$IP")
done
for I in $(seq 0 $(($NS-1))); do
    for J in $(seq 0 $(($NS-1))); do
        if [ "$I" -eq "$J" ]; then
            continue
        elif ([ "$I" -lt "$G" ] && [ "$J" -lt "$G" ]) || \
            ([ "$I" -ge "$G" ] && [ "$J" -ge "$G" ]); then
            D="$DD"
        else
            D="$D0"
        fi
        echo $I "->" $J "=" $D >&2
        docker run -d --name "epaxos-pumba-delay-$I-$J" \
            --volume /var/run/docker.sock:/var/run/docker.sock \
            gaiaadm/pumba --log-level info \
            netem --duration 1000000h \
            --target "${IPS[$J]}" \
            delay --time "$D" \
            --jitter "$(($D * $JT / 100))" \
            "epaxos-server-$I" >&2
    done
done

sleep 20

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
        | sed "s/^/$NS,$NR,$NP,$RK,$D0,$DD,$JT,$I,/"
done

make pumba-down >&2
./compose.sh "$NS" --prod down
