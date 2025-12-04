#!/bin/sh
set -e

if [ "$#" -gt 0 ]; then
	exec /usr/local/bin/mrcoordinator "$@"
fi

set -- /data/pg-*.txt
if [ "$1" = "/data/pg-*.txt" ]; then
	echo "no /data/pg-*.txt files found" >&2
	exit 1
fi

exec /usr/local/bin/mrcoordinator "$@"

