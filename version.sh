#!/bin/sh

VER="$(git describe --always)"
git status --porcelain 2> /dev/null | tail -n1 > /dev/null
if [ "$?" = 0 ]; then
	VER="$VER-dirty"
fi
echo "$VER" | cmp --silent VERSION -
if [ ! "$?" = 0 ]; then
	echo "$VER" > VERSION
fi
