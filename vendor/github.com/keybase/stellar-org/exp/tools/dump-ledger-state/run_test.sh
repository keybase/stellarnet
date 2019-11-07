#! /bin/bash
set -e

if [ -z ${LATEST_LEDGER+x} ]; then
    # Get latest ledger
    echo "Getting latest checkpoint ledger..."
    export LATEST_LEDGER=`curl -s http://history.stellar.org/prd/core-live/core_live_001/.well-known/stellar-history.json | jq -r '.currentLedger'`
    echo "Latest ledger: $LATEST_LEDGER"
fi

# Dump state using Golang
echo "Dumping state using ingest..."
go run ./main.go
echo "State dumped..."

# Catchup core
stellar-core --conf ./stellar-core.cfg catchup $LATEST_LEDGER/1

echo "Dumping state from stellar-core..."
./dump_core_db.sh
echo "State dumped..."

echo "Comparing state dumps..."
./diff_test.sh
