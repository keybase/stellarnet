# Get state from stellar-core DB, colums match CSV printer
# FETCH_COUNT is there for circleci to use cursor-based method of getting rows (less RAM usage):
# https://dba.stackexchange.com/a/101510

echo "Fetching accounts from stellar-core DB..."
psql -d core -t -A -F"," --variable="FETCH_COUNT=10000" -c "select accountid, balance, seqnum, numsubentries, inflationdest, homedomain, thresholds, flags, COALESCE(buyingliabilities, 0), COALESCE(sellingliabilities, 0), signers from accounts" > accounts_core.csv
rm accounts_core_sorted.csv || true # Remove if exist in case original files are rebuilt

echo "Fetching accountdata from stellar-core DB..."
psql -d core -t -A -F"," --variable="FETCH_COUNT=10000" -c "select accountid, dataname, datavalue from accountdata" > accountdata_core.csv
rm accountdata_core_sorted.csv || true # Remove if exist in case original files are rebuilt

echo "Fetching offers from stellar-core DB..."
psql -d core -t -A -F"," --variable="FETCH_COUNT=10000" -c "select sellerid, offerid, sellingasset, buyingasset, amount, pricen, priced, flags from offers" > offers_core.csv
rm offers_core_sorted.csv || true # Remove if exist in case original files are rebuilt

echo "Fetching trustlines from stellar-core DB..."
psql -d core -t -A -F"," --variable="FETCH_COUNT=10000" -c "select accountid, assettype, issuer, assetcode, tlimit, balance, flags, COALESCE(buyingliabilities, 0), COALESCE(sellingliabilities, 0) from trustlines" > trustlines_core.csv
rm trustlines_core_sorted.csv || true # Remove if exist in case original files are rebuilt
