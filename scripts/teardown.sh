#!/bin/bash

set -euxo pipefail

NS=${1:-5}   # Number of servers

make pumba-down >&2
./compose.sh --prod "$NS" down
