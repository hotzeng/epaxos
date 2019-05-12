#!/bin/bash

set -euxo pipefail

pv --version >&2

NS=${1:-5}   # Number of servers
NR=${2:-64}  # Number of records to test
NP=${3:-1}   # Client window
RK=${4:-5}   # Key collsion is 5%
D0=${5:-150} # Inter-group delay
JT=${6:-10}  # Std.Var. is 10% of mean

NK=$(($NR * 100 / $RK))

make -j2 debug >&2
make pumba-down >&2
if [ -f "./docker-compose.yml" ]; then
    docker-compose down
fi

sudo find ./data -type f -delete

./compose.sh --prod "$NS" up -d

G=$(($NS / 2))
IPS=()
for I in $(seq 0 $(($NS-1))); do
    IP="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "epaxos-server-$I")"
    IPS+=("$IP")
done
for I in $(seq 0 $(($NS-1))); do
    D="$(($D0 / 2))"
    IPX=()
    for J in $(seq 0 $(($NS-1))); do
        if [ "$I" -eq "$J" ]; then
            continue
        elif ([ "$I" -lt "$G" ] && [ "$J" -lt "$G" ]) || \
            ([ "$I" -ge "$G" ] && [ "$J" -ge "$G" ]); then
            continue
        else
            IPX+=("${IPS[$J]}")
        fi
        echo $I "->" $J "+=" $D >&2
    done
    IPXS="$(printf ' --target=%s' "${IPX[@]}")"
    docker run -d --name "epaxos-pumba-delay-$I-out" \
        --volume /var/run/docker.sock:/var/run/docker.sock \
        gaiaadm/pumba --log-level info \
        netem --duration 1000000h \
        $IPXS \
        delay --time "$D" \
        --jitter "$(($D * $JT / 100))" \
        "epaxos-server-$I" >&2
done

sleep 10
