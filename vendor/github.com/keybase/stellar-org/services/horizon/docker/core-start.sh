#!/usr/bin/env bash

set -e

source /etc/profile

echo "using config:"
cat stellar-core.cfg

# initialize new db
stellar-core new-db

if [ "$1" = "standalone" ]; then
  # start a network from scratch
  stellar-core force-scp

  # initialze history archive for stand alone network
  stellar-core new-hist vs

  # serve history archives to horizon on port 1570
  pushd /history/vs/
  python3 -m http.server 1570 &
  popd
fi

exec /init -- stellar-core run