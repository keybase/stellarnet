FROM golang:1.12
WORKDIR /go/src/github.com/stellar/go

COPY . .
ENV GO111MODULE=on
RUN go install github.com/stellar/go/tools/...
RUN go install github.com/stellar/go/services/...
