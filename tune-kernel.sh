#!/bin/sh

sysctl -w net.core.rmem_max=33554432
sysctl -w net.core.wmem_max=1048576
sysctl -w net.core.rmem_default=2097152
sysctl -w net.core.wmem_default=65536
sysctl -w net.ipv4.tcp_rmem='4096 2097152 33554432'
sysctl -w net.ipv4.tcp_wmem='4096 65536 33554432'
sysctl -w net.ipv4.tcp_mem='8388608 33554432 33554432'
sysctl -w net.ipv4.route.flush=1
