set -e

docker-machine create swarm-manager \
    --engine-install-url experimental.docker.com \
    --driver google \
    --google-machine-type n1-standard-1 \
    --google-zone us-east4-c \
    --google-disk-size "500" \
    --google-tags swarm-cluster \
    --google-project 'decisive-scion-234720'

use-manager() {
    eval $(docker-machine env swarm-manager)
}

docker swarm init

docker-machine create swarm-worker-1 \
    --engine-install-url experimental.docker.com \
    --driver google \
    --google-machine-type n1-standard-1 \
    --google-zone us-east4-c \
    --google-disk-size "500" \
    --google-tags swarm-cluster \
    --google-project 'decisive-scion-234720'
