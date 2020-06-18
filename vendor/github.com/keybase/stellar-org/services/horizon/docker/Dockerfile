FROM golang:1.14-stretch AS builder

WORKDIR /go/src/github.com/stellar/go
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go install github.com/stellar/go/services/horizon

FROM ubuntu:16.04

# ca-certificates are required to make tls connections
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY --from=builder /go/bin/horizon ./

ENTRYPOINT ["./horizon"]
