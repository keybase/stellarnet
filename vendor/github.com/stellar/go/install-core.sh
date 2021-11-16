#!/usr/bin/env bash

# CORE_VERSION="15.3.0-498.7a7f18c.xenial~SetTrustlineFlagsPR~buildtests"

CORE_PACKAGE=stellar-core
if [[ "$CORE_VERSION" != "" ]]; then
  CORE_PACKAGE="$CORE_PACKAGE=$CORE_VERSION"
fi
sudo wget -qO - https://apt.stellar.org/SDF.asc | APT_KEY_DONT_WARN_ON_DANGEROUS_USAGE=true sudo apt-key add -
sudo bash -c 'echo "deb https://apt.stellar.org xenial unstable" > /etc/apt/sources.list.d/SDF-unstable.list'
sudo apt-get update && sudo apt-get install -y "$CORE_PACKAGE"
echo "using stellar core version $(stellar-core version)"
echo "export CAPTIVE_CORE_BIN=/usr/bin/stellar-core" >> ~/.bashrc

