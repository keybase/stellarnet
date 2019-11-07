FROM golang:1.12-alpine3.10 AS builder

RUN apk --no-cache add git mercurial

WORKDIR /go/src/github.com/stellar/go
ENV GO111MODULE=on
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go install github.com/stellar/go/services/horizon

FROM alpine:3.10
# ca-certificates are required to make tls connections
RUN apk add --no-cache ca-certificates

COPY --from=builder /go/bin/horizon ./

ENTRYPOINT ["./horizon"]