#!/usr/bin/env bash

set -e
set -x

source /etc/profile

echo "using config:"
cat stellar-core.cfg

# initialize new db
stellar-core new-db

if [ "$1" = "standalone" ]; then
  # initialize history archive for standalone network
  stellar-core new-hist vs

  # serve history archives to horizon on port 1570
  pushd /history/vs/
  python3 -m http.server 1570 &
  popd
fi

exec stellar-core run
