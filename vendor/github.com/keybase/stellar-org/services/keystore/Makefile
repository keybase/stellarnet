# Check if we need to prepend docker commands with sudo
SUDO := $(shell docker version >/dev/null 2>&1 || echo "sudo")

# If TAG is not provided set default value
TAG ?= stellar/keystore:$(shell git rev-parse --short HEAD)$(and $(shell git status -s),-dirty-$(shell id -u -n))

docker-build:
	cd ../../ && \
	$(SUDO) docker build -f services/keystore/docker/Dockerfile -t $(TAG) .

docker-push:
	cd ../../ && \
	$(SUDO) docker push $(TAG)
