#!/bin/bash

set -e

NETWORK=${1:-testnet}

case $NETWORK in
  standalone)
    DOCKER_FLAGS="-f docker-compose.yml -f docker-compose.standalone.yml"
    echo "running on standalone network"
    ;;

  pubnet)
    DOCKER_FLAGS="-f docker-compose.yml -f docker-compose.pubnet.yml"
    echo "running on public network"
    ;;

  testnet)
    echo "running on test network"
    ;;

  *)
    echo  "$1 is not a supported option "
    exit 1
    ;;
esac


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

docker-compose $DOCKER_FLAGS up --build -d