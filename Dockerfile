FROM arm32v6/golang:1.10.1-alpine3.7 AS builder

ENV GOPATH /go
WORKDIR /go/src

RUN mkdir -p /go/src/github.com/txn2/txagent
COPY . /go/src/github.com/txn2/txagent

RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o /go/bin/agent /go/src/github.com/txn2/iotwifi/agent.go

FROM arm32v6/alpine:3.7

RUN apk --no-cache add ca-certificates
WORKDIR /

COPY --from=builder /go/bin/agent /agent
ENTRYPOINT ["/agent"]
