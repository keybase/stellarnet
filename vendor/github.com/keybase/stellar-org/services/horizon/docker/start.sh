#!/bin/bash

set -e

case $OSTYPE in
linux-*)
    if ! grep -E "^127.0.0.1 .*host\.docker\.internal.*$" /etc/hosts > /dev/null 2>&1; then
        echo "Missing: \`127.0.0.1 host.docker.internal\` in /etc/hosts"
        exit 1
    fi

    echo "NETWORK_MODE=host" > .env
    ;;
*)
    echo "NETWORK_MODE=bridge" > .env
    ;;
esac

docker-compose up --build -d